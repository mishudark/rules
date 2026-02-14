package rules

import (
	"context"
	"fmt"
	"reflect"
)

type registryKey struct{}

// DataRegistry holds validation data as any (interface{}).
// It enables tree reuse by separating rule definitions from data binding.
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

// MustGet retrieves the data from context, panicking if not found.
// Use this when data is guaranteed to exist in the context.
func MustGet(ctx context.Context) any {
	v, ok := Get(ctx)
	if !ok {
		panic("validation data not found in context")
	}
	return v
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

// MustGetAs retrieves typed data from context, panicking if not found or type mismatch.
func MustGetAs[T any](ctx context.Context) T {
	v, ok := GetAs[T](ctx)
	if !ok {
		var zero T
		panic(fmt.Sprintf("validation data not found or not of type %T", zero))
	}
	return v
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
