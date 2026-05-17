# Go Rules Engine

[![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8?logo=go)](https://go.dev)
[![Go Reference](https://pkg.go.dev/badge/github.com/mishudark/rules.svg)](https://pkg.go.dev/github.com/mishudark/rules)
[![Build Status](https://github.com/mishudark/rules/actions/workflows/go.yml/badge.svg)](https://github.com/mishudark/rules/actions/workflows/go.yml)

A flexible **rule engine** for Go that lets you build and evaluate complex validation logic as a tree structure. Think of it as composable, conditional decision trees for validation.

- **Composable** — Build small pieces, compose into complex trees
- **Reusable** — Build trees once, validate many different data instances
- **Type-safe** — Generics for compile-time safety in rules and conditions
- **Performant** — ~500-1000ns per full tree evaluation

## When to use this?

- **Feature flags** — enable features based on user attributes
- **A/B testing** — route users to different experiences
- **Form validation** — validate complex forms with conditions
- **Business rules** — implement decision trees that non-developers can visualize
- **Reusable validation** — build rule trees once, validate against different data

---

## Quick Start

```bash
go get github.com/mishudark/rules
```

**Minimal example — validate a user's age:**

```go
import (
    "context"
    "github.com/mishudark/rules"
    "github.com/mishudark/rules/validators"
)

type User struct { Age int }

tree := rules.Rules(validators.MinValue("age", 25, 18))
err := rules.ValidateWithData(context.Background(), tree, rules.ProcessingHooks{}, "check", User{Age: 25})
// err == nil
```

---

## Quick Examples

### Simple validation (closure-based — data bound at construction)

```go
user := User{Age: 25, Country: "USA"}

tree := rules.Node(
    rules.NewConditionPure("fromUSA", func() bool {
        return user.Country == "USA"
    }),
    rules.Rules(validators.MinValue("age", user.Age, 21)),
)

err := rules.ValidateWithData(context.Background(), tree, rules.ProcessingHooks{}, "check", user)
// err == nil
```

### Reusable validation (data registry pattern — data bound at validation time)

Build the tree once, reuse with different data:

```go
tree := rules.Node(
    rules.FastIsA[User]("isUser"),
    rules.Rules(
        rules.NewTypedRule[User]("checkAge", func(ctx context.Context, user User) error {
            if user.Age < 21 {
                return fmt.Errorf("must be 21 or older")
            }
            return nil
        }),
    ),
)

for _, user := range users {
    err := rules.ValidateWithData(ctx, tree, hooks, "ageCheck", user)
}
```

### Multiple rules (all must pass)

```go
tree := rules.Rules(
    validators.MinValue("age", 25, 18),
    validators.MaxValue("age", 25, 65),
    validators.Email("email", "user@example.com", nil),
)
```

### At least one rule must pass

```go
tree := rules.Rules(
    rules.Or(
        validators.Email("contact", "user@example.com", nil),
        validators.ValidDomainNameAdvanced("contact", "example.com", false),
    ),
)
```

---

## How Validation Works

Validation executes in **4 phases**:

```
PrepareConditions → Evaluate → Prepare → Validate
```

| Phase | What happens |
|-------|-------------|
| **PrepareConditions** | Conditions with side effects (impure) fetch data from DB/API. Pure conditions skip this. |
| **Evaluate** | Walk the tree: check conditions, collect candidate rules |
| **Prepare** | Each candidate rule can fetch data it needs |
| **Validate** | Run each rule's validation logic, collect errors |

Hooks can be injected at each phase via `ProcessingHooks`:

```go
hooks := rules.ProcessingHooks{
    AfterPrepareConditions:  func(ctx context.Context) error { log.Println("conditions prepared"); return nil },
    AfterEvaluateConditions: func(ctx context.Context) error { log.Println("evaluated"); return nil },
    AfterPrepareRules:       func(ctx context.Context) error { log.Println("rules prepared"); return nil },
    AfterValidateRules:      func(ctx context.Context) error { log.Println("validated"); return nil },
}
```

---

## Reusable Trees (Data Registry Pattern)

This is the **recommended pattern** for most use cases. Build trees once, reuse across many data instances.

### Why reusable trees?

| Aspect | Closure-Based | Data Registry |
|--------|--------------|---------------|
| Tree reuse | Build per validation | Build once, reuse many |
| Performance | Slower (allocation per request) | Fast (shared tree) |
| Testability | Harder (data bound at construction) | Easier (inject data at validation) |
| Cross-package | Complex | Natural |

### How it works

1. **Build the tree** at startup or in a separate package using typed rules/conditions
2. **Store data** in the context using `DataRegistry` at validation time
3. **Access data** in rules/conditions via `Get` or `GetAs[T]`

```go
var userValidationTree = rules.Node(
    rules.FastIsA[User]("isUser"),
    rules.Rules(
        rules.NewTypedRule[User]("checkAge", func(ctx context.Context, u User) error {
            if u.Age < 18 {
                return fmt.Errorf("must be 18 or older")
            }
            return nil
        }),
        rules.NewTypedRule[User]("checkEmail", func(ctx context.Context, u User) error {
            if !strings.Contains(u.Email, "@") {
                return fmt.Errorf("invalid email")
            }
            return nil
        }),
    ),
)

for i, user := range users {
    err := rules.ValidateWithData(ctx, userValidationTree, hooks, "validate", user)
    errs[i] = err
}
```

---

## Conditional Logic

### Node (if condition, then validate)

```go
// Premium users must have age 18+
tree := rules.Node(
    rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" }),
    rules.Rules(validators.MinValue("age", user.Age, 18)),
)
```

### Either/Then (if-else)

```go
tree := rules.Either(
    rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" }),
    // Left branch (condition true): premium rules
    rules.Rules(
        validators.MinValue("age", user.Age, 18),
        validators.URL(user.Website, []string{"https"}),
    ),
    // Right branch (condition false): free user rules
    rules.Rules(validators.MinValue("age", user.Age, 13)),
)
```

### AllOf (AND) / AnyOf (OR) — logical composition

```go
// All must pass
tree := rules.AllOf(
    rules.Rules(validators.Email("email", req.Email, nil)),
    rules.Rules(validators.MinValue("age", req.Age, 13)),
)

// At least one must pass
tree := rules.AnyOf(
    rules.Rules(validators.Email("contact", val, nil)),
    rules.Rules(validators.URL(val, nil)),
)
```

### Not (negate a condition)

```go
tree := rules.Node(
    rules.Not(rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" })),
    rules.Rules(validators.MinValue("age", user.Age, 13)),
)
```

### Complex tree

```go
tree := rules.Root(
    rules.Rules(validators.Email("email", user.Email, nil)),

    rules.Node(
        rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" }),
        rules.Rules(
            validators.MinValue("age", user.Age, 18),
            validators.URL(user.Website, []string{"https"}),
        ),
    ),

    rules.Node(
        rules.NewConditionPure("isFree", func() bool { return user.Plan == "free" }),
        rules.Rules(
            validators.MinValue("age", user.Age, 13),
            validators.ValidDomainNameAdvanced("country", user.Country, false),
        ),
    ),
)
```

### Cross-package tree composition

Build rules in separate packages and merge at runtime:

```go
// package userrules
type User struct { Name string; Age int }

func UserRules() rules.Evaluable {
    return rules.Node(
        rules.FastIsA[User]("isUser"),
        rules.Rules(
            rules.NewTypedRule[User]("checkAge", func(ctx context.Context, u User) error {
                if u.Age < 18 { return fmt.Errorf("too young") }
                return nil
            }),
        ),
    )
}

// package productrules
type Product struct { Name string; Price float64 }

func ProductRules() rules.Evaluable {
    return rules.Node(
        rules.FastIsA[Product]("isProduct"),
        rules.Rules(
            rules.NewTypedRule[Product]("checkPrice", func(ctx context.Context, p Product) error {
                if p.Price <= 0 { return fmt.Errorf("invalid price") }
                return nil
            }),
        ),
    )
}

// main.go — merge and use
mergedTree := rules.Root(
    userrules.UserRules(),
    productrules.ProductRules(),
)

rules.ValidateWithData(ctx, mergedTree, hooks, "validate", user)
rules.ValidateWithData(ctx, mergedTree, hooks, "validate", product)
```

---

## Runtime Type Conditions

```go
rules.FastIsA[User]("isUser")               // Exact type match (generics, fastest)
rules.IsA[User]("isUser")                   // Exact type match (reflection, ~6ns)
rules.IsAssignableTo[Named]("isNamed")      // Interface implementation
rules.IsNil("isNil")                        // Nil check
rules.IsNotNil("hasData")                   // Non-nil check
rules.HasField("hasEmail", "Email")         // Struct field or map key exists
rules.FieldEquals("isAdmin", "Role", "admin") // Struct field or map key equals value
```

### Data-driven conditions

```go
condition := rules.NewCondition("isAdult", func(ctx context.Context) bool {
    user, ok := rules.GetAs[User](ctx)
    if !ok { return false }
    return user.Age >= 18
})
```

---

## Common Validators

| Function | What it validates |
|----------|-------------------|
| `MinValue(name, value, min)` / `MaxValue(name, value, max)` | Numeric bounds |
| `Email(name, value, allowlist)` | Email addresses (RFC 5322) |
| `URL(value, schemes)` | URLs with optional scheme allowlist |
| `ValidDomainNameAdvanced(name, domain, acceptIdna)` | Domain names |
| `MinLengthString(name, value, min)` / `MaxLengthString(name, value, max)` | String length (rune-aware) |
| `MinLengthSlice[T](name, value, min)` / `MaxLengthSlice[T](name, value, max)` | Slice length (generic, any slice type) |
| `Slug(name, value)` / `UnicodeSlug(name, value)` | ASCII and Unicode slugs |
| `IPv4Address(value)` / `IPv6Address(value)` / `IPv46Address(value)` | IP addresses |
| `FileExtensionValidator(value, allowed)` | File extensions (case-insensitive) |
| `DecimalValidator(value, maxDigits, decimalPlaces)` | Decimal numbers with precision |
| `CommaSeparatedIntegerList(value)` | Comma-separated list of integers |
| `ProhibitNullCharacters(value)` | Null character detection |
| `StepValue[T](value, step, offset)` | Values in fixed increments |
| `NewRuleContentType(name, reader, allowedMIMEs)` | MIME content type detection |
| `ValidateIPv4Address(value)` / `ValidateIPv6Address(value)` / `ValidateIPv46Address(value)` | Legacy IP validators (aliases above) |

> 💡 Validators without a `name` parameter return errors with an empty `Field`. Validators with `name` fill the `Field` field in `rules.Error` for structured error reporting.

---

## Full Example: User Registration

```go
package main

import (
    "context"
    "fmt"

    "github.com/mishudark/rules"
    "github.com/mishudark/rules/validators"
)

type RegistrationRequest struct {
    Email   string
    Age     int
    Country string
    Plan    string
    Website string
}

func ValidateRegistration(ctx context.Context, req RegistrationRequest) error {
    tree := rules.Root(
        rules.Rules(validators.Email("email", req.Email, nil)),
        rules.Rules(validators.MinValue("age", req.Age, 13)),

        rules.Node(
            rules.NewConditionPure("isPremium", func() bool { return req.Plan == "premium" }),
            rules.Rules(validators.URL(req.Website, []string{"https"})),
        ),

        rules.Node(
            rules.NewConditionPure("isUS", func() bool { return req.Country == "US" }),
            rules.Rules(validators.MinValue("age", req.Age, 18)),
        ),

        rules.Node(
            rules.NewConditionPure("isNotUS", func() bool { return req.Country != "US" }),
            rules.Rules(validators.MinValue("age", req.Age, 21)),
        ),
    )

    return rules.ValidateWithData(ctx, tree, rules.ProcessingHooks{}, "registration", req)
}

func main() {
    req := RegistrationRequest{
        Email:   "john@example.com",
        Age:     25,
        Country: "US",
        Plan:    "free",
        Website: "",
    }

    if err := ValidateRegistration(context.Background(), req); err != nil {
        fmt.Printf("Validation failed: %v\n", err)
    } else {
        fmt.Println("Registration valid!")
    }
}
```

---

## API Reference

### Core Interfaces

| Interface | Purpose |
|-----------|---------|
| `Rule` | Single validation unit (`Prepare`, `Validate`, `Name`) |
| `Condition` | Boolean check controlling if child rules run (`Prepare`, `IsValid`, `Name`, `IsPure`) |
| `Evaluable` | Tree component that can be evaluated (`PrepareConditions`, `Evaluate`) |

### Tree Building Functions

| Function | Returns | What it does |
|----------|---------|--------------|
| `rules.Root(children...)` | `Evaluable` | Top-level container (AnyOf) — passes if **any** child passes |
| `rules.Node(condition, children...)` | `Evaluable` | Runs children **only if** condition is true |
| `rules.Either(condition, left, right)` | `Evaluable` | If-else: left if true, right if false |
| `rules.Rules(rules...)` | `Evaluable` | Leaf node — **all** rules must pass |
| `rules.AllOf(children...)` | `Evaluable` | Logical AND — **all** children must succeed |
| `rules.AnyOf(children...)` | `Evaluable` | Logical OR — **at least one** child must succeed |
| `rules.Not(condition)` | `Condition` | Negate a condition |
| `rules.Or(rule, rules...)` | `Rule` | Rule-level OR (use inside `Rules()`) |
| `rules.NewChainRules(rules...)` | `Rule` | Sequential rules (stop on first error, use inside `Rules()`) |

### Data Registry Functions

| Function | What it does |
|----------|--------------|
| `rules.NewDataRegistry(data)` | Creates a registry with validation data |
| `rules.WithRegistry(ctx, reg)` | Attaches registry to context |
| `rules.ValidateWithData(ctx, tree, hooks, name, data)` | Validates with data (convenience) |
| `rules.Validate(ctx, tree, hooks, name)` | Validates using registry already in context |
| `rules.ValidateMulti(ctx, targets, hooks, name)` | Batch validation of multiple targets |
| `rules.ValidateMultiWithData(ctx, targets, hooks, name, ...data)` | Batch validation with data |
| `rules.Get(ctx)` | Gets raw data from context |
| `rules.GetAs[T](ctx)` | Gets typed data from context |
| `rules.TypeOf(ctx)` | Returns `reflect.Type` of data in context |
| `rules.IsType(ctx, type)` | Checks if data is exactly given type |

### Rule Constructors

| Function | Description |
|----------|-------------|
| `rules.NewRule(name, fn)` | Rule with `any` data parameter (pure) |
| `rules.NewTypedRule[T](name, fn)` | Type-safe rule (pure) |
| `rules.NewTypedRuleWithPrepare[In, T](name, prepare, validate)` | Type-safe rule with Prepare (impure) |
| `rules.NewRulePure(name, fn)` | Closure-based rule (pure, legacy) |

### Condition Constructors

| Function | Description |
|----------|-------------|
| `rules.NewCondition(name, fn)` | Data-driven condition (pure) |
| `rules.NewConditionSideEffect(name, prepare, condition)` | Condition with side effects (impure) |
| `rules.NewTypedCondition[T](name, fn)` | Type-safe condition (pure) |
| `rules.NewTypedConditionWithPrepare[In, T](name, prepare, condition)` | Type-safe condition with Prepare (impure) |
| `rules.NewConditionPure(name, fn)` | Closure-based condition (pure, legacy) |

### Runtime Type Conditions

| Function | What it does |
|----------|--------------|
| `rules.IsA[T]("name")` | True if data is exactly type T (reflection, ~6ns) |
| `rules.FastIsA[T]("name")` | Type assertion (~1-2ns), faster |
| `rules.FastTypeSwitch("name", fn)` | Type check using type switch (flexible, fast) |
| `rules.IsAssignableTo[T]("name")` | True if data can be assigned to T |
| `rules.IsNil("name")` | True if data is nil |
| `rules.IsNotNil("name")` | True if data is not nil |
| `rules.HasField("name", "fieldName")` | True if data has struct field or map key |
| `rules.FieldEquals("name", "fieldName", value)` | True if struct field/map key equals value |

### Creating Custom Rules (Data Registry)

**Basic rule:**

```go
myRule := rules.NewRule("myRule", func(ctx context.Context, data any) error {
    user := data.(User)
    if user.Disabled {
        return fmt.Errorf("user is disabled")
    }
    return nil
})
```

**Type-safe rule:**

```go
myRule := rules.NewTypedRule("myRule", func(ctx context.Context, user User) error {
    if user.Disabled {
        return fmt.Errorf("user is disabled")
    }
    return nil
})
```

**Type-safe rule with Prepare (impure):**

Use this for side effects before validation (database checks, API calls):

```go
myRule := rules.NewTypedRuleWithPrepare(
    "checkEmailUnique",
    func(ctx context.Context, user User) (StoredData, error) {
        return db.EmailData(ctx, user.Email)
    },
    func(ctx context.Context, user User, data StoredData) error {
        if !strings.Contains(user.Email, "@") {
            return fmt.Errorf("invalid email format")
        }
        if data.Exists {
            return fmt.Errorf("email already in use")
        }
        return nil
    },
)
```

⚠️ **Concurrency:** `NewTypedRuleWithPrepare` stores mutable state. Create one tree per validation target:

```go
// ✅ Correct: One tree per validation
for _, user := range users {
    tree := buildTree()
    err := rules.ValidateWithData(ctx, tree, hooks, "validate", user)
}

// ❌ Wrong: Sharing tree across goroutines causes race conditions
tree := buildTree()
for _, user := range users {
    go func(u User) {
        err := rules.ValidateWithData(ctx, tree, hooks, "validate", u) // RACE!
    }(user)
}
```

**Chained rules (sequential, stop on first error):**

```go
validationChain := rules.NewChainRules(
    validators.Email("email", req.Email, nil),
    validators.MinValue("age", req.Age, 13),
)
```

### Creating Custom Conditions (Data Registry)

**Data-driven condition:**

```go
myCondition := rules.NewCondition("isAdmin", func(ctx context.Context) bool {
    user, ok := rules.GetAs[User](ctx)
    if !ok { return false }
    return user.Role == "admin"
})
```

**Type-safe condition:**

```go
myCondition := rules.NewTypedCondition("isAdult", func(ctx context.Context, user User) bool {
    return user.Age >= 18
})
```

**Type-safe condition with Prepare (impure):**

```go
myCondition := rules.NewTypedConditionWithPrepare(
    "userHasPermission",
    func(ctx context.Context, user User) (Permissions, error) {
        return db.LoadPermissions(ctx, user.ID)
    },
    func(ctx context.Context, user User, perms Permissions) bool {
        return perms.CanEdit
    },
)
```

### Closure-Based (Legacy)

**Pure rule:**

```go
myRule := rules.NewRulePure("myRule", func() error {
    if user.Disabled {
        return fmt.Errorf("user is disabled")
    }
    return nil
})
```

**Pure condition:**

```go
myCondition := rules.NewConditionPure("isAdmin", func() bool {
    return user.Role == "admin"
})
```

**Impure condition (with Prepare):**

```go
var user User

tree := rules.Node(
    rules.NewConditionSideEffect(
        "userActive",
        func(ctx context.Context) error {
            var err error
            user, err = db.GetUser(ctx, userID)
            return err
        },
        func(ctx context.Context) bool {
            return user.Active
        },
    ),
    rules.Rules(validators.Email("email", user.Email, nil)),
)
```

The `IsPure()` method controls optimization:
- `true` — no side effects, engine may skip `Prepare()`
- `false` — has side effects, `Prepare()` always called before `IsValid()`

### Error Handling

All errors in this library are structured as `rules.Error`, which implements the standard `error` interface:

```go
type Error struct {
    Field string // Field name (empty if validator doesn't take a name)
    Err   string // Human-readable error message (lowercase, per Go convention)
    Code  string // Error code for programmatic handling and i18n
}
```

Check specific error codes:

```go
err := rules.ValidateWithData(ctx, tree, hooks, "validate", data)
if err != nil {
    var re rules.Error
    if errors.As(err, &re) {
        switch re.Code {
        case "INVALID_EMAIL_FORMAT":
            // Handle invalid email
        case "VALUE_LOWER_MIN":
            // Handle min value violation
        }
    }
}
```

**Common error codes:**

| Code | Validator |
|------|-----------|
| `INVALID_EMAIL_FORMAT`, `DOMAIN_NOT_ALLOWED` | `Email` |
| `VALUE_LOWER_MIN`, `VALUE_EXCEEDS_MAX` | `MinValue`, `MaxValue` |
| `MIN_LENGTH_STRING`, `MAX_LENGTH_STRING` | `MinLengthString`, `MaxLengthString` |
| `MIN_LENGTH_SLICE`, `MAX_LENGTH_SLICE` | `MinLengthSlice`, `MaxLengthSlice` |
| `INVALID_SLUG`, `INVALID_UNICODE_SLUG` | `Slug`, `UnicodeSlug` |
| `URL_CANNOT_BE_EMPTY`, `INVALID_URL_FORMAT`, `URL_SCHEME_NOT_ALLOWED` | `URL` |
| `INVALID_IPV4_ADDRESS`, `INVALID_IPV6_ADDRESS`, `INVALID_IP_ADDRESS` | `IPv4Address`, `IPv6Address`, `IPv46Address` |
| `FILE_EXTENSION_NOT_ALLOWED` | `FileExtensionValidator` |
| `CONTENT_TYPE_EMPTY_FILE`, `CONTENT_TYPE_MISMATCH` | `NewRuleContentType` |
| `NULL_CHARACTERS_FOUND` | `ProhibitNullCharacters` |
| `STEP_VALUE_ZERO`, `STEP_VALUE_INVALID` | `StepValue` |
| `TYPE_MISMATCH`, `DATA_NOT_FOUND`, `RULE_FUNC_NIL` | Core engine |

---

## Batch Validation

```go
targets := make([]rules.TreeAndData, len(items))
for i, item := range items {
    targets[i] = rules.TreeAndData{Tree: tree, Data: item}
}
err := rules.ValidateMultiWithData(ctx, targets, hooks, "batch")
```

---

## Performance

| Operation | Speed | Allocations |
|-----------|-------|-------------|
| `IsA[T]()` type check | ~6-7 ns/op | 0 |
| `GetAs[T]()` data access | ~6-7 ns/op | 0 |
| Full tree evaluation | ~500-1000 ns/op | ~10-17 |

For high-throughput scenarios (1000s of evaluations/sec):

1. **`IsA[T]()` is fast** — caches the target type; reflection overhead is negligible (~6ns)
2. **Use `ValidateMultiWithData`** — Batch validations to amortize context creation cost
3. **Avoid deep nesting** — Each level adds overhead; flatten where possible
4. **Cache pure trees** — Trees with only pure rules/conditions (`RuleDataFunc`, `ConditionFunc`, `RulePure`) can be safely cached globally. Trees with `TypedRuleDataFunc` or `TypedConditionWithPrepare` must be created per-goroutine.

See [PERFORMANCE.md](PERFORMANCE.md) for detailed benchmarks and optimization guides.

### Fast Type Switching

```go
condition := rules.FastTypeSwitch("isValid", func(data any) bool {
    switch data.(type) {
    case User, *User, Product, *Product:
        return true
    default:
        return false
    }
})
```

---

## Installation

```bash
go get github.com/mishudark/rules
```

Requires Go **1.26+**.

---

## Best Practices

1. **Use Data Registry for reusable trees** — Build trees once with `NewRule`, `NewTypedRule`, reuse with `ValidateWithData`.

2. **Use closures for one-off validations** — For simple, single-use validations, `NewRulePure` and `NewConditionPure` are fine.

3. **Use `FastIsA[T]` for type switching** — Merging trees from different packages? Use `FastIsA[YourType]("isYourType")`.

4. **Prefer type-safe rules** — `NewTypedRule[T]` gives compile-time type safety within the rule function.

5. **Use `ChainRules` for sequential checks** — When rules must run in order (stop on first error), use `ChainRules` instead of manual chaining.

6. **Know the difference: `Or` vs `AnyOf`** — `Or(rule, ...)` creates a `Rule` (use inside `Rules()`), while `AnyOf(children...)` creates an `Evaluable` (use as a tree node).

7. **Only cache pure trees globally** — Trees with `TypedRuleDataFunc` or `TypedConditionWithPrepare` store mutable state. Cache only trees with pure rules (`RuleDataFunc`, `ConditionFunc`, `RulePure`).

8. **Don't share stateful trees across goroutines** — `NewTypedRuleWithPrepare` and `NewTypedConditionWithPrepare` are **not safe for concurrent use**. Create one tree per goroutine:

    ```go
    // ✅ Safe: tree created per loop iteration
    for _, user := range users {
        tree := buildTree()
        err := rules.ValidateWithData(ctx, tree, hooks, "validate", user)
    }

    // ❌ Race condition: sharing stateful tree
    tree := buildTree()
    for _, user := range users {
        go func(u User) {
            err := rules.ValidateWithData(ctx, tree, hooks, "validate", u)
        }(user)
    }
    ```
