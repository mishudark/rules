# AGENT Guidelines for github.com/mishudark/rules

This document outlines essential information for agents working within this Go codebase.

## 1. Project Overview

This repository implements a flexible **rule engine** in Go for creating and evaluating complex validation logic through a tree-like structure. It supports:

- **Feature flags** — enable features based on user attributes
- **A/B testing** — route users to different experiences  
- **Form validation** — validate complex forms with conditions
- **Business rules** — implement decision trees
- **Reusable validation** — build rule trees once, validate against different data

**Module**: `github.com/mishudark/rules`  
**Go Version**: 1.24.1  
**Dependency**: `golang.org/x/exp`

## 2. Essential Commands

```bash
# Run all tests (parallelized)
go test ./...

# Run tests with verbose output (as used in CI)
go test -v ./...

# Build the project
go build ./...

# Format code
go fmt ./...

# Lint/vet code
go vet ./...

# Run benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkFullTree

# CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Memory profiling
go test -memprofile=mem.prof -bench=.
```

**CI/CD**: GitHub Actions runs on push/PR to `master` branch using Go 1.26 on `ubuntu-slim`.

## 3. Code Organization and Structure

### Core Interfaces (in root package)

| Interface | Purpose | Key Methods |
|-----------|---------|-------------|
| `Rule` | Single validation unit | `Prepare()`, `Validate()`, `Name()`, `SetExecutionPath()`, `GetExecutionPath()` |
| `Condition` | Boolean check for conditional logic | `Prepare()`, `IsValid()`, `Name()`, `IsPure()` |
| `Evaluable` | Tree component that can be evaluated | `PrepareConditions()`, `Evaluate()` |

### Node Types (Tree Building Blocks)

| Type | Description |
|------|-------------|
| `LeafNode` | Terminal node containing `[]Rule` |
| `ConditionNode` | Node with a `Condition`; evaluates children only if condition passes |
| `AllOfNode` | Logical AND - all children must succeed |
| `AnyOfNode` | Logical OR - at least one child must succeed |
| `ConditionEither` | If-else: evaluates left branch if condition true, else right branch |

### Constructor Functions

```go
Root(children ...Evaluable) Evaluable          // Top-level OR container
Node(condition Condition, children ...Evaluable) Evaluable  // Conditional branch
Rules(rules ...Rule) Evaluable                  // Leaf node with rules
AllOf(children ...Evaluable) Evaluable          // AND logic
AnyOf(children ...Evaluable) Evaluable          // OR logic
Either(condition Condition, left, right []Evaluable) Evaluable  // If-else
Not(condition Condition) Condition              // Logical negation
Or(rule Rule, rules ...Rule) Rule               // At least one rule passes
```

### Two Usage Patterns

**1. Closure-Based (Legacy - Single Use)**
- Data bound at construction time
- Use `NewRulePure()`, `NewConditionPure()`
- Good for one-off validations

**2. Data Registry Pattern (Recommended - Reusable)**
- Data bound at validation time via context
- Build tree once, reuse with different data
- Use `NewRule()`, `NewTypedRule()`, `NewCondition()`, `ValidateWithData()`

### Data Registry Functions

```go
// Create registry and validate with data
ValidateWithData(ctx, tree, hooks, "name", data)

// Manual registry management
reg := NewDataRegistry(data)
ctx = WithRegistry(ctx, reg)
err := Validate(ctx, tree, hooks, "name")

// Access data in rules/conditions
data, ok := Get(ctx)
user, ok := GetAs[User](ctx)  // Type-safe access
```

### Directory Structure

```
.
├── *.go                    # Core rule engine (rules, conditions, validation, data_registry)
├── *_test.go               # Core tests and benchmarks
├── validators/             # Common validators (email, url, length, etc.)
│   ├── *.go               # Validator implementations
│   └── *_test.go          # Validator tests
├── go.mod                 # Module definition
├── go.sum                 # Dependencies
├── README.md              # User documentation with examples
├── PERFORMANCE.md         # Performance optimization guide
└── AGENTS.md              # This file
```

## 4. Naming Conventions and Style Patterns

- **Go Standard**: Follows standard Go naming conventions
  - `CamelCase` for exported types and functions
  - `camelCase` for unexported variables
  - `PascalCase` for exported struct fields
- **Interfaces**: Named with capability description (e.g., `Rule`, `Condition`, `Evaluable`)
- **Error Codes**: UPPER_SNAKE_CASE (e.g., `INVALID_EMAIL_FORMAT`, `MIN_LENGTH_STRING`)
- **File Names**: snake_case (e.g., `email_test.go`, `data_registry.go`)
- **Package Name**: `rules` (root), `validators` (subpackage)

## 5. Testing Approach and Patterns

### Test Structure
- **Location**: Co-located with source files (`foo.go` → `foo_test.go`)
- **Parallel**: All tests use `t.Parallel()` for parallel execution
- **Style**: Table-driven tests with `t.Run()` subtests

### Test Pattern Example
```go
func TestFeature(t *testing.T) {
    t.Parallel()

    testCases := []struct {
        name    string
        input   string
        wantErr bool
        errCode string
    }{
        {name: "Valid", input: "good", wantErr: false},
        {name: "Invalid", input: "bad", wantErr: true, errCode: "ERROR_CODE"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            rule := SomeValidator(tc.input)
            err := rule.Validate(context.Background())

            if (err != nil) != tc.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
            }

            if tc.wantErr {
                if e, ok := err.(rules.Error); ok && e.Code != tc.errCode {
                    t.Errorf("errorCode = %s, want %s", e.Code, tc.errCode)
                }
            }
        })
    }
}
```

### Benchmarking
- Located in `benchmark_test.go`
- Compares different approaches (reflection vs type assertions)
- Use `b.ResetTimer()` after setup

## 6. Important Gotchas and Non-Obvious Patterns

### 6.1 Root Creates AnyOfNode
The `Root()` function creates an `AnyOfNode`, meaning **at least one** immediate child must evaluate successfully for the root to pass. An empty `Root()` returns success.

### 6.2 Pure vs Impure Conditions
The `IsPure()` method is critical:
- **Pure** (`IsPure() == true`): No side effects; engine may optimize by skipping `Prepare()`
- **Impure** (`IsPure() == false`): Has side effects (DB calls, API calls); `Prepare()` is always called before `IsValid()`

Use `NewCondition()` for pure conditions, `NewConditionSideEffect()` or `NewTypedConditionWithPrepare()` for impure ones.

### 6.3 State Storage and Concurrency ⚠️
Rules/conditions with `Prepare()` may store state (e.g., `TypedRuleDataFunc`, `TypedConditionWithPrepare`). These are **NOT safe for concurrent use**:

```go
// ✅ CORRECT: Create tree per validation
for _, user := range users {
    tree := buildTree() // Create inside loop
    err := rules.ValidateWithData(ctx, tree, hooks, "validate", user)
}

// ❌ WRONG: Sharing across goroutines causes races
tree := buildTree()
for _, user := range users {
    go func(u User) {
        err := rules.ValidateWithData(ctx, tree, hooks, "validate", u) // RACE!
    }(user)
}
```

### 6.4 Execution Path Tracing
Rules track their execution path through the tree:
```go
rule.SetExecutionPath(path)  // Set during evaluation
rule.GetExecutionPath()      // Get for debugging/logging
```
Tests verify this path to ensure correct tree traversal.

### 6.5 Type Checking Options
Multiple ways to check types, with different performance characteristics:

```go
IsA[T]("name")                    // Reflection-based (~6ns), caches type
FastIsA[T]("name")                // Type assertion (~1-2ns), fastest
FastTypeSwitch("name", fn)        // Flexible type switch
IsAssignableTo[T]("name")         // Interface compatibility
```

### 6.6 Empty String Handling
Many validators (like `Email`) consider empty strings valid. Use a separate "required" check if needed:
```go
// Email returns nil (valid) for empty string
validators.Email("email", "", nil)  // passes

// Use additional validation if required field needed
```

### 6.7 Validation Flow (4 Steps)
1. `PrepareConditions()` - Prepare conditions (fetch data, etc.)
2. `Evaluate()` - Check conditions, collect candidate rules
3. `Prepare()` - Prepare each rule
4. `Validate()` - Run validation on prepared rules

Hooks can be injected at each phase via `ProcessingHooks`.

### 6.8 Error Type
The package defines a structured error type:
```go
type Error struct {
    Field string  // Field name
    Err   string  // Error message
    Code  string  // Error code for i18n
}
```

## 7. Adding New Validators

When creating validators in the `validators/` package:

1. Return `rules.Rule` from constructor functions
2. Use `rules.NewRulePure()` for simple validations
3. Return `rules.Error` with appropriate `Field`, `Err`, and `Code`
4. Consider empty values as valid (let caller add required check)
5. Add comprehensive table-driven tests with error code verification
6. Use rune counting for string length (not bytes) for Unicode correctness

Example:
```go
func MyValidator(fieldName string, value string) rules.Rule {
    return rules.NewRulePure("myValidator", func() error {
        if value == "" {
            return nil  // Empty is valid
        }
        if !isValid(value) {
            return rules.Error{
                Field: fieldName,
                Err:   "validation failed",
                Code:  "MY_VALIDATION_FAILED",
            }
        }
        return nil
    })
}
```

## 8. Performance Considerations

- Tree reuse is key: build once, validate many
- Use `ValidateMulti()` for batch validations
- `IsA[T]()` reflection is optimized (~6ns) - don't avoid for micro-optimizations
- Prefer flat trees over deep nesting for hot paths
- See `PERFORMANCE.md` for detailed benchmarks and optimization strategies
