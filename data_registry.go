package rules

import (
	"context"
	"reflect"
)

type registryKey struct{}

// DataRegistry holds validation data as any (interface{}).
// It enables tree reuse by separating rule definitions from data binding.
//
// A registry holds a single payload. Rules that need more than one input
// value should load the additional data in their Prepare step (see
// NewTypedRuleWithPrepare and NewTypedConditionWithPrepare) or compose the
// payload into one struct.
type DataRegistry struct {
	data any
}

// NewDataRegistry creates a registry with the provided data.
// The data can be any type and is accessed at validation time via Get/GetAs.
func NewDataRegistry(data any) *DataRegistry {
	return &DataRegistry{data: data}
}

// Get retrieves the raw data from context.
// Returns the data and a boolean indicating if data was found.
func Get(ctx context.Context) (any, bool) {
	reg, ok := ctx.Value(registryKey{}).(*DataRegistry)
	if !ok {
		return nil, false
	}
	return reg.data, true
}

// GetAs retrieves typed data from context with runtime type assertion.
// Returns the typed data and a boolean indicating if the type matches.
//
// Example:
//
//	user, ok := rules.GetAs[User](ctx)
//	if ok {
//	    // Use user with type safety
//	}
func GetAs[T any](ctx context.Context) (T, bool) {
	var zero T
	data, ok := Get(ctx)
	if !ok {
		return zero, false
	}
	typed, ok := data.(T)
	return typed, ok
}

// WithRegistry returns a new context with the provided registry.
// The registry can then be accessed via Get/GetAs in rules and conditions.
//
// Example:
//
//	tree := buildRulesTree() // build once
//	for _, user := range users {
//	    ctx := rules.WithRegistry(context.Background(), rules.NewDataRegistry(user))
//	    err := rules.Validate(ctx, tree, hooks, "userValidation")
//	}
func WithRegistry(ctx context.Context, reg *DataRegistry) context.Context {
	return context.WithValue(ctx, registryKey{}, reg)
}

// ValidateWithData executes validation with the provided data.
// This is a convenience function that wraps data in a registry and executes validation.
//
// Example:
//
//	user := User{Name: "Alice", Age: 25}
//	err := rules.ValidateWithData(ctx, tree, hooks, "userValidation", user)
func ValidateWithData(ctx context.Context, tree Evaluable, hooks ProcessingHooks, name string, data any) error {
	reg := NewDataRegistry(data)
	ctx = WithRegistry(ctx, reg)
	return Validate(ctx, tree, hooks, name)
}

// EvaluateMetricsWithData evaluates the tree for validation errors and metric
// indicators with the provided data. This is a convenience function that
// wraps data in a registry and executes EvaluateMetrics.
//
// Example:
//
//	user := User{Name: "Alice", Age: 25}
//	report, err := rules.EvaluateMetricsWithData(ctx, tree, hooks, "healthCheck", user)
func EvaluateMetricsWithData(ctx context.Context, tree Evaluable, hooks ProcessingHooks, name string, data any) (Report, error) {
	reg := NewDataRegistry(data)
	ctx = WithRegistry(ctx, reg)
	return EvaluateMetrics(ctx, tree, hooks, name)
}

type TreeAndData struct {
	Tree Evaluable
	Data any
}

// ValidateMultiWithData executes multiple targets with their respective data.
// This allows validating multiple different data objects against the same or different trees.
func ValidateMultiWithData(ctx context.Context, targets []TreeAndData, hooks ProcessingHooks, name string,
) error {
	targetsWithCtx := make([]Target, len(targets))
	for i, t := range targets {
		reg := NewDataRegistry(t.Data)
		targetCtx := WithRegistry(ctx, reg)
		targetsWithCtx[i] = Target{
			tree: t.Tree,
			ctx:  targetCtx,
		}
	}
	return ValidateMulti(ctx, targetsWithCtx, hooks, name)
}

// EvaluateMetricsMultiWithData executes multiple targets with their respective
// data, collecting and aggregating metric indicators per target. This allows
// evaluating multiple different data objects against the same or different
// trees in a single batched pass.
func EvaluateMetricsMultiWithData(ctx context.Context, targets []TreeAndData, hooks ProcessingHooks, name string,
) ([]Report, error) {
	targetsWithCtx := make([]Target, len(targets))
	for i, t := range targets {
		reg := NewDataRegistry(t.Data)
		targetCtx := WithRegistry(ctx, reg)
		targetsWithCtx[i] = Target{
			tree: t.Tree,
			ctx:  targetCtx,
		}
	}
	return EvaluateMetricsMulti(ctx, targetsWithCtx, hooks, name)
}

// IsType checks if the data in context is exactly the target type using reflection.
// This is useful for runtime type dispatch in conditions.
func IsType(ctx context.Context, targetType reflect.Type) bool {
	data, ok := Get(ctx)
	if !ok {
		return false
	}
	return reflect.TypeOf(data) == targetType
}

// TypeOf returns the reflect.Type of data in context, or nil if no data.
func TypeOf(ctx context.Context) reflect.Type {
	data, ok := Get(ctx)
	if !ok {
		return nil
	}
	return reflect.TypeOf(data)
}

type preparedStoreKey struct{}

// preparedStore holds the data that rules and conditions retrieve during their
// Prepare step, keyed by the rule or condition instance. It is created by the
// engine once per evaluation (once per target in multi-target runs) and travels
// in the context.
//
// Per-evaluation scoping is what makes Prepare-based rules and conditions safe
// to share across goroutines: the data a rule reads back in Validate comes from
// its own evaluation's store, never from the rule/condition struct.
//
// The store is read with [GetPreparedAs] (typed) or [GetPrepared], and written
// with [PutPrepared]. Built-in typed rules and conditions self-record in their
// Prepare; custom implementations may use [PutPrepared] the same way.
type preparedStore struct {
	data map[any]any
}

// get returns the prepared data for the given rule or condition.
func (s *preparedStore) get(key any) (any, bool) {
	if s == nil {
		return nil, false
	}
	data, ok := s.data[key]
	return data, ok
}

// put stores the prepared data for the given rule or condition.
func (s *preparedStore) put(key any, value any) {
	if s == nil {
		return
	}
	s.data[key] = value
}

// withPreparedStore returns a context carrying a fresh preparedStore and the
// store itself.
func withPreparedStore(ctx context.Context) (context.Context, *preparedStore) {
	store := &preparedStore{data: make(map[any]any)}
	return context.WithValue(ctx, preparedStoreKey{}, store), store
}

// preparedStoreFromContext returns the preparedStore attached to ctx, or nil.
func preparedStoreFromContext(ctx context.Context) *preparedStore {
	store, _ := ctx.Value(preparedStoreKey{}).(*preparedStore)
	return store
}

// recordPrepared stores data keyed by key in the per-evaluation preparedStore.
// It is a no-op when no store is attached to ctx (e.g. when a rule/condition is
// exercised outside the engine). Built-in typed rules and conditions use this
// in their Prepare so the same data is available to their Validate/IsValid.
func recordPrepared(ctx context.Context, key, data any) {
	if store := preparedStoreFromContext(ctx); store != nil {
		store.put(key, data)
	}
}

// PutPrepared records data keyed by key in the per-evaluation preparedStore.
// Custom Rule or Condition implementations that fetch data in their Prepare
// call this so they can read it back in Validate / IsValid with [GetPreparedAs]
// (typed) or [GetPrepared] (untyped). It is a no-op when no store is attached
// to ctx (which only happens outside the engine entry points).
//
// Use the rule or condition instance itself as the key so the data is scoped to
// it and the same tree can be reused across goroutines without cross-talk.
func PutPrepared(ctx context.Context, key, data any) {
	recordPrepared(ctx, key, data)
}

// GetPrepared returns the data the given rule or condition recorded during its
// Prepare step, or nil when it has none. Prefer [GetPreparedAs] for type-safe
// access.
func GetPrepared(ctx context.Context, key any) any {
	store := preparedStoreFromContext(ctx)
	if store == nil {
		return nil
	}
	data, _ := store.get(key)
	return data
}

// GetPreparedAs returns the typed data the given rule or condition recorded
// during its Prepare step. The boolean is false when no store is attached, the
// key has no recorded data, or the recorded data is not of type T.
//
// Generic injection point: typed rules and conditions call this from their
// Validate / IsValid to pull back the data they returned from Prepare, without
// converting to and from any at the interface boundary.
//
// Example:
//
//	func (r *MyRule) Prepare(ctx context.Context) (any, error) {
//	    perms, err := loadPermissions(ctx)
//	    rules.PutPrepared(ctx, r, perms)
//	    return perms, err
//	}
//	func (r *MyRule) Validate(ctx context.Context) error {
//	    perms, ok := rules.GetPreparedAs[Permissions](ctx, r)
//	    if !ok {
//	        return errors.New("permissions not prepared")
//	    }
//	    return check(perms)
//	}
func GetPreparedAs[T any](ctx context.Context, key any) (T, bool) {
	var zero T
	store := preparedStoreFromContext(ctx)
	if store == nil {
		return zero, false
	}
	data, ok := store.get(key)
	if !ok {
		return zero, false
	}
	typed, ok := data.(T)
	return typed, ok
}
