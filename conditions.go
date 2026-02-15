package rules

import (
	"context"
	"fmt"
	"reflect"
)

// ConditionFunc is a condition implementation that uses a predicate function
// with access to context data. This enables runtime type checking and data-driven conditions.
type ConditionFunc struct {
	name      string
	predicate func(ctx context.Context) bool
	pure      bool
}

// Prepare implements Condition interface. It's a no-op for pure conditions.
func (c *ConditionFunc) Prepare(ctx context.Context) error {
	return nil
}

// Name returns the condition name.
func (c *ConditionFunc) Name() string {
	return c.name
}

// IsValid evaluates the condition using the predicate.
func (c *ConditionFunc) IsValid(ctx context.Context) bool {
	return c.predicate(ctx)
}

// IsPure returns whether this condition has side effects.
func (c *ConditionFunc) IsPure() bool {
	return c.pure
}

var _ Condition = (*ConditionFunc)(nil)

// NewCondition creates a condition with a predicate that has access to context data.
// This is the primary way to create data-driven conditions for reusable rule trees.
//
// Example:
//
//	isAdult := rules.NewCondition("isAdult", func(ctx context.Context) bool {
//	    user, ok := rules.GetAs[User](ctx)
//	    if !ok {
//	        return false
//	    }
//	    return user.Age >= 18
//	})
func NewCondition(name string, predicate func(ctx context.Context) bool) Condition {
	return &ConditionFunc{
		name:      name,
		predicate: predicate,
		pure:      true,
	}
}

// typeChecker is an optimized condition that caches type information.
// It uses reflection only once during construction for better performance.
type typeChecker struct {
	name       string
	targetType reflect.Type
}

func (c *typeChecker) Prepare(ctx context.Context) error { return nil }
func (c *typeChecker) Name() string                      { return c.name }
func (c *typeChecker) IsValid(ctx context.Context) bool {
	data, ok := Get(ctx)
	if !ok {
		return false
	}
	return reflect.TypeOf(data) == c.targetType
}
func (c *typeChecker) IsPure() bool { return true }

var _ Condition = (*typeChecker)(nil)

// IsA creates a condition that checks if the data in context is exactly type T.
// This is useful for runtime type dispatch in merged rule trees.
//
// Performance: Uses reflection for type checking. For high-throughput scenarios
// (1000s of evaluations), consider using NewCondition with type assertions directly.
//
// Example:
//
//	tree := rules.Root(
//	    rules.Node(
//	        rules.IsA[User]("isUser"),
//	        rules.Rules(userValidationRules...),
//	    ),
//	    rules.Node(
//	        rules.IsA[Product]("isProduct"),
//	        rules.Rules(productValidationRules...),
//	    ),
//	)
func IsA[T any](name string) Condition {
	var zero T
	return &typeChecker{
		name:       name,
		targetType: reflect.TypeOf(zero),
	}
}

// assignableChecker checks if data can be assigned to a target type.
type assignableChecker struct {
	name       string
	targetType reflect.Type
}

func (c *assignableChecker) Prepare(ctx context.Context) error { return nil }
func (c *assignableChecker) Name() string                      { return c.name }
func (c *assignableChecker) IsValid(ctx context.Context) bool {
	data, ok := Get(ctx)
	if !ok {
		return false
	}
	return reflect.TypeOf(data).AssignableTo(c.targetType)
}
func (c *assignableChecker) IsPure() bool { return true }

var _ Condition = (*assignableChecker)(nil)

// IsAssignableTo creates a condition that checks if the data in context can be assigned to type T.
// This is more flexible than IsA, allowing interface implementations and embedding.
//
// Performance: Uses reflection for type checking. For high-throughput scenarios,
// consider using NewCondition with type assertions directly.
//
// Example:
//
//	type Validatable interface {
//	    Validate() error
//	}
//
//	tree := rules.Node(
//	    rules.IsAssignableTo[Validatable]("isValidatable"),
//	    rules.Rules(rules.NewRule("validate", func(ctx context.Context, data any) error {
//	        v := data.(Validatable)
//	        return v.Validate()
//	    })),
//	)
func IsAssignableTo[T any](name string) Condition {
	var zero T
	return &assignableChecker{
		name:       name,
		targetType: reflect.TypeOf(zero),
	}
}

// FastIsA returns a condition that uses type assertion for the check.
// This is faster than IsA because it avoids reflection by using generics.
// Use this for high-throughput scenarios.
//
// Example:
//
//	condition := rules.FastIsA[User]("isUser")
func FastIsA[T any](name string) Condition {
	return &genericChecker[T]{name: name}
}

// genericChecker is a type-safe condition checker using type assertion.
type genericChecker[T any] struct {
	name string
}

func (c *genericChecker[T]) Prepare(ctx context.Context) error { return nil }
func (c *genericChecker[T]) Name() string                      { return c.name }
func (c *genericChecker[T]) IsValid(ctx context.Context) bool {
	data, ok := Get(ctx)
	if !ok {
		return false
	}
	_, ok = data.(T)
	return ok
}
func (c *genericChecker[T]) IsPure() bool { return true }

var _ Condition = (*genericChecker[any])(nil)

// FastTypeSwitch creates a condition using a type switch for maximum performance.
// Use this when you need to check against multiple types in a high-throughput scenario.
//
// Example:
//
//	condition := rules.FastTypeSwitch("typeCheck", func(data any) bool {
//	    switch data.(type) {
//	    case User, *User:
//	        return true
//	    default:
//	        return false
//	    }
//	})
func FastTypeSwitch(name string, check func(data any) bool) Condition {
	return &ConditionFunc{
		name: name,
		predicate: func(ctx context.Context) bool {
			data, ok := Get(ctx)
			if !ok {
				return false
			}
			return check(data)
		},
		pure: true,
	}
}

// IsNil creates a condition that checks if the data in context is nil.
func IsNil(name string) Condition {
	return &ConditionFunc{
		name: name,
		predicate: func(ctx context.Context) bool {
			data, ok := Get(ctx)
			if !ok {
				return true // No data is considered nil
			}
			if data == nil {
				return true
			}
			// Check if it's a nil interface with underlying nil value
			v := reflect.ValueOf(data)
			return v.Kind() == reflect.Pointer && v.IsNil()
		},
		pure: true,
	}
}

// IsNotNil creates a condition that checks if the data in context is not nil.
func IsNotNil(name string) Condition {
	return &ConditionFunc{
		name: name,
		predicate: func(ctx context.Context) bool {
			data, ok := Get(ctx)
			if !ok {
				return false
			}
			if data == nil {
				return false
			}
			v := reflect.ValueOf(data)
			if v.Kind() == reflect.Pointer && v.IsNil() {
				return false
			}
			return true
		},
		pure: true,
	}
}

// HasField creates a condition that checks if the data has a specific field (for structs) or key (for maps).
// This is useful for duck-typing validation.
func HasField(name string, fieldName string) Condition {
	return &ConditionFunc{
		name: name,
		predicate: func(ctx context.Context) bool {
			data, ok := Get(ctx)
			if !ok {
				return false
			}

			v := reflect.ValueOf(data)

			// Dereference pointer if needed
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					return false
				}
				v = v.Elem()
			}

			switch v.Kind() {
			case reflect.Struct:
				field := v.FieldByName(fieldName)
				return field.IsValid()
			case reflect.Map:
				key := reflect.ValueOf(fieldName)
				return v.MapIndex(key).IsValid()
			default:
				return false
			}
		},
		pure: true,
	}
}

// FieldEquals creates a condition that checks if a struct field or map key equals a value.
// Uses reflect.DeepEqual for comparison.
func FieldEquals(name string, fieldName string, expected any) Condition {
	return &ConditionFunc{
		name: name,
		predicate: func(ctx context.Context) bool {
			data, ok := Get(ctx)
			if !ok {
				return false
			}

			v := reflect.ValueOf(data)

			// Dereference pointer if needed
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					return false
				}
				v = v.Elem()
			}

			var fieldValue reflect.Value
			switch v.Kind() {
			case reflect.Struct:
				fieldValue = v.FieldByName(fieldName)
			case reflect.Map:
				key := reflect.ValueOf(fieldName)
				fieldValue = v.MapIndex(key)
			default:
				return false
			}

			if !fieldValue.IsValid() {
				return false
			}

			return reflect.DeepEqual(fieldValue.Interface(), expected)
		},
		pure: true,
	}
}

// TypedConditionWithPrepare is a condition that loads data during Prepare
// and uses it during IsValid. This enables separating data loading from evaluation.
// In is the input data type from the DataRegistry, T is the loaded data type.
type TypedConditionWithPrepare[In any, T any] struct {
	name       string
	prepare    func(ctx context.Context, input In) (T, error)
	condition  func(ctx context.Context, input In, data T) bool
	loadedData T
	hasData    bool
}

var _ Condition = (*TypedConditionWithPrepare[any, any])(nil)

// Prepare retrieves typed input data from context, loads additional data,
// and stores it for IsValid to use.
func (c *TypedConditionWithPrepare[In, T]) Prepare(ctx context.Context) error {
	input, ok := GetAs[In](ctx)
	if !ok {
		var zero In
		return Error{
			Field: c.name,
			Err:   fmt.Sprintf("expected data of type %T, got different type", zero),
			Code:  "TYPE_MISMATCH",
		}
	}
	data, err := c.prepare(ctx, input)
	if err != nil {
		return err
	}
	c.loadedData = data
	c.hasData = true
	return nil
}

// Name returns the condition name.
func (c *TypedConditionWithPrepare[In, T]) Name() string {
	return c.name
}

// IsValid evaluates the condition using the data loaded during Prepare.
func (c *TypedConditionWithPrepare[In, T]) IsValid(ctx context.Context) bool {
	input, ok := GetAs[In](ctx)
	if !ok {
		return false
	}

	if !c.hasData {
		return false
	}

	return c.condition(ctx, input, c.loadedData)
}

// IsPure returns false as this condition has side effects during Prepare.
func (c *TypedConditionWithPrepare[In, T]) IsPure() bool {
	return false
}

// NewTypedConditionWithPrepare creates a type-safe condition with Prepare support.
// In is the input data type from the DataRegistry, T is the loaded data type.
// The prepare function receives typed input data and loads additional data,
// which is then passed to the condition function during IsValid.
//
// ⚠️ IMPORTANT: This condition stores state (loadedData) and is NOT safe for concurrent
// use. When validating multiple items concurrently, create one tree per target:
//
//	// CORRECT: One tree per target
//	for _, user := range users {
//	    tree := buildTree() // Create tree inside loop
//	    err := rules.ValidateWithData(ctx, tree, hooks, "validate", user)
//	}
//
//	// WRONG: Sharing tree across goroutines causes race conditions
//	tree := buildTree()
//	for _, user := range users {
//	    go func(u User) {
//	        err := rules.ValidateWithData(ctx, tree, hooks, "validate", u) // RACE!
//	    }(user)
//	}
//
// Example:
//
//	condition := rules.NewTypedConditionWithPrepare(
//	    "userHasPermission",
//	    func(ctx context.Context, user User) (Permissions, error) {
//	        return db.LoadPermissions(ctx, user.ID)
//	    },
//	    func(ctx context.Context, user User, perms Permissions) bool {
//	        return perms.CanEdit
//	    },
//	)
func NewTypedConditionWithPrepare[In any, T any](
	name string,
	prepare func(ctx context.Context, input In) (T, error),
	condition func(ctx context.Context, input In, data T) bool,
) Condition {
	return &TypedConditionWithPrepare[In, T]{
		name:      name,
		prepare:   prepare,
		condition: condition,
		hasData:   false,
	}
}
