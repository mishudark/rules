# Go Rules Engine

A flexible validation library that lets you build complex rules as a tree structure. Think of it as "if this, then that" for validation logic.

## When to use this?

- **Feature flags** — enable features based on user attributes
- **A/B testing** — route users to different experiences
- **Form validation** — validate complex forms with conditions
- **Business rules** — implement decision trees that non-developers can visualize
- **Reusable validation** — build rule trees once, validate against different data

---

## Quick Examples

### Simple validation (closure-based - data bound at construction)

Validate that a user is over 21 and from the USA:

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

### Reusable validation (data registry pattern - data bound at validation time)

Build the tree once, reuse it with different data:

```go
// Build tree once (can be done at init or in a separate package)
tree := rules.Node(
    rules.IsA[User]("isUser"),  // Runtime type check
    rules.Rules(
        rules.NewTypedRule[User]("checkAge", func(ctx context.Context, user User) error {
            if user.Age < 21 {
                return fmt.Errorf("must be 21 or older")
            }
            return nil
        }),
    ),
)

// Reuse tree with different users
for _, user := range users {
    err := rules.ValidateWithData(ctx, tree, hooks, "ageCheck", user)
}
```

### Multiple rules (all must pass)

Pass multiple rules to `Rules()` — all of them must pass:

```go
tree := rules.Rules(
    validators.MinValue("age", 25, 18),
    validators.MaxValue("age", 25, 65),
    validators.Email("email", "user@example.com", nil),
)
```

### At least one rule must pass

Use `Or()` when at least one rule should pass:

```go
tree := rules.Rules(
    rules.Or(
        validators.Email("contact", "user@example.com", nil),
        validators.ValidDomainNameAdvanced("contact", "example.com", false),
    ),
)
```

---

## Reusable Trees (Data Registry Pattern)

This is the **recommended pattern** for most use cases. Build validation trees once, reuse them across many data instances.

### Why use reusable trees?

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
// Step 1: Build tree once (can be in a separate package)
var userValidationTree = rules.Node(
    rules.IsA[User]("isUser"),
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

// Step 2: Reuse for many users
func ProcessUsers(users []User) []error {
    hooks := rules.ProcessingHooks{}
    errs := make([]error, len(users))
    
    for i, user := range users {
        // Data is bound at validation time, not construction
        err := rules.ValidateWithData(ctx, userValidationTree, hooks, "validate", user)
        errs[i] = err
    }
    return errs
}
```

---

## Conditional Logic

### Either/Then (if-else)

Use `Either()` for if-else logic — if condition is true, evaluate left branch; otherwise, evaluate right branch:

```go
// IF user is premium, require age 18+ AND website; ELSE require just age 13+
tree := rules.Either(
    rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" }),
    // Left branch (condition is true): premium rules
    rules.Rules(
        validators.MinValue("age", user.Age, 18),
        validators.URL(user.Website, []string{"https"}),
    ),
    // Right branch (condition is false): free user rules
    rules.Rules(validators.MinValue("age", user.Age, 13)),
)
```

### Complex Conditional Trees

Combine multiple conditions into sophisticated validation logic:

```go
tree := rules.Root(
    // Always require valid email
    rules.Rules(validators.Email("email", user.Email, nil)),
    
    // Premium users: age 18+ AND valid website
    rules.Node(
        rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" }),
        rules.Rules(
            validators.MinValue("age", user.Age, 18),
            validators.URL(user.Website, []string{"https"}),
        ),
    ),
    
    // Free users: age 13+ AND valid country
    rules.Node(
        rules.NewConditionPure("isFree", func() bool { return user.Plan == "free" }),
        rules.Rules(
            validators.MinValue("age", user.Age, 13),
            validators.ValidDomainNameAdvanced("country", user.Country, false),
        ),
    ),
)
```

### Cross-package tree composition with type switching

Build rules in separate packages and merge them at runtime. Runtime type checks route validation to the appropriate rules:

```go
// package userrules/user_rules.go
type User struct { Name string; Age int }

func UserRules() rules.Evaluable {
    return rules.Node(
        rules.IsA[User]("isUser"),
        rules.Rules(
            rules.NewTypedRule[User]("checkAge", func(ctx context.Context, u User) error {
                if u.Age < 18 { return fmt.Errorf("too young") }
                return nil
            }),
        ),
    )
}

// package productrules/product_rules.go
type Product struct { Name string; Price float64 }

func ProductRules() rules.Evaluable {
    return rules.Node(
        rules.IsA[Product]("isProduct"),
        rules.Rules(
            rules.NewTypedRule[Product]("checkPrice", func(ctx context.Context, p Product) error {
                if p.Price <= 0 { return fmt.Errorf("invalid price") }
                return nil
            }),
        ),
    )
}

// main.go - merge and use
func main() {
    mergedTree := rules.Root(
        userrules.UserRules(),
        productrules.ProductRules(),
    )
    
    // Works with both User and Product!
    rules.ValidateWithData(ctx, mergedTree, hooks, "validate", user)
    rules.ValidateWithData(ctx, mergedTree, hooks, "validate", product)
}

---

## Runtime Type Conditions

Use these conditions for type switching in merged trees:

```go
rules.IsA[User]("isUser")                    // Exact type match
rules.IsAssignableTo[Named]("isNamed")       // Interface implementation
rules.IsNil("isNil")                         // Nil check
rules.IsNotNil("hasData")                    // Non-nil check
```

### Data-Driven Conditions

```go
condition := rules.NewCondition("isAdult", func(ctx context.Context) bool {
    user, ok := rules.GetAs[User](ctx)
    if !ok {
        return false
    }
    return user.Age >= 18
})
```

---

## Common Validators

Here's what you can validate out of the box:

| Validator | Use for |
|-----------|---------|
| `MinValue` / `MaxValue` | Numbers (age, price, quantity) |
| `Email` | Email addresses |
| `URL` | URLs with optional scheme restrictions |
| `ValidDomainNameAdvanced` | Domain names |
| `MinLengthString` / `MaxLengthString` | String sizes |
| `MinLengthSlice` / `MaxLengthSlice` | Array sizes |
| `FileExtensionValidator` | File types by extension |
| `ValidateIPv4Address` / `ValidateIPv6Address` | IP addresses |
| `DecimalValidator` | Decimal numbers with precision control |
| `CommaSeparatedIntegerList` | Comma-separated numbers |
| `ProhibitNullCharacters` | Strings without null chars |
| `Slug` | URL-friendly slugs |
| `StepValue` | Values in increments |
| `Schema` | Struct field validation (with data registry) |
| `RequiredField` | Required field validation |
| `MinValueField` / `MaxValueField` | Numeric field validation |
| `MinLengthField` / `MaxLengthField` | Length field validation |
| `EmailField` | Email field validation |

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
    Email    string
    Age      int
    Country  string
    Plan     string // "free" or "premium"
    Website  string
}

func ValidateRegistration(ctx context.Context, req RegistrationRequest) error {
    tree := rules.Root(
        // Email is always required
        rules.Rules(validators.Email("email", req.Email, nil)),
        
        // Age check for everyone
        rules.Rules(validators.MinValue("age", req.Age, 13)),
        
        // Premium users need a valid website
        rules.Node(
            rules.NewConditionPure("isPremium", func() bool { return req.Plan == "premium" }),
            rules.Rules(validators.URL(req.Website, []string{"https"})),
        ),
        
        // US users need 18+, others need 21+
        rules.Node(
            rules.NewConditionPure("isUS", func() bool { return req.Country == "US" }),
            rules.Rules(validators.MinValue("age", req.Age, 18)),
        ),
        
        // Non-US users need 21+ (only runs if isUS is false)
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

### Core Types

- **`Rule`** — The actual validation logic (e.g., "age must be > 18")
- **`Condition`** — A boolean check that controls whether child rules run (e.g., "is user premium?")
- **`Evaluable`** — Anything that can be evaluated in the tree

### Tree Building Functions

| Function | What it does |
|----------|--------------|
| `rules.Root(children...)` | Top-level container, passes if any child passes |
| `rules.Node(condition, children...)` | Runs children only if condition is true |
| `rules.Either(condition, left, right)` | If-else: evaluates left if condition is true, otherwise right |
| `rules.Rules(rules...)` | Leaf node — all rules must pass |
| `rules.Or(rules...)` | Passes if at least one rule passes |

### Data Registry Functions

| Function | What it does |
|----------|--------------|
| `rules.NewDataRegistry(data)` | Creates a registry with validation data |
| `rules.WithRegistry(ctx, reg)` | Attaches registry to context |
| `rules.ValidateWithData(ctx, tree, hooks, name, data)` | Validates with data (convenience) |
| `rules.Get(ctx)` | Gets raw data from context |
| `rules.GetAs[T](ctx)` | Gets typed data from context |
| `rules.MustGet(ctx)` | Gets data, panics if not found |
| `rules.MustGetAs[T](ctx)` | Gets typed data, panics if not found |
| `rules.NewRule(name, fn)` | Creates a rule with `any` data parameter (pure) |
| `rules.NewTypedRule[T](name, fn)` | Creates a type-safe rule (pure) |
| `rules.NewTypedRuleWithPrepare[T](name, prepare, validate)` | Creates a type-safe rule with Prepare (impure) |
| `rules.NewCondition(name, fn)` | Creates a data-driven condition (pure) |
| `rules.NewTypedCondition[T](name, fn)` | Creates a type-safe condition (pure) |
| `rules.NewTypedConditionWithPrepare[In, T](name, prepare, condition)` | Creates a type-safe condition with Prepare (impure) |

### Runtime Type Conditions

| Function | What it does |
|----------|--------------|
| `rules.IsA[T]("name")` | True if data is exactly type T (uses reflection, ~6ns) |
| `rules.FastIsA("name", prototype)` | Same as IsA but with explicit prototype (same speed) |
| `rules.FastTypeSwitch("name", fn)` | Type check using type switch (flexible, fast) |
| `rules.IsAssignableTo[T]("name")` | True if data can be assigned to T |
| `rules.IsNil("name")` | True if data is nil |
| `rules.IsNotNil("name")` | True if data is not nil |

### Creating Custom Rules (Data Registry)

**Basic rule with any data:**

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
myRule := rules.NewTypedRule[User]("myRule", func(ctx context.Context, user User) error {
    if user.Disabled {
        return fmt.Errorf("user is disabled")
    }
    return nil
})
```

**Type-safe rule with Prepare (non-pure):**

Use this when you need side effects before validation (e.g., database checks, API calls):

```go
myRule := rules.NewTypedRuleWithPrepare[User](
    "checkEmailUnique",
    func(ctx context.Context, user User) error {
        // Prepare: Check database for existing email
        exists, err := db.EmailExists(ctx, user.Email)
        if err != nil {
            return err
        }
        if exists {
            return fmt.Errorf("email already exists")
        }
        return nil
    },
    func(ctx context.Context, user User) error {
        // Validate: Additional validation after prepare
        if !strings.Contains(user.Email, "@") {
            return fmt.Errorf("invalid email format")
        }
        return nil
    },
)
```

### Creating Custom Conditions (Data Registry)

**Data-driven condition:**

```go
myCondition := rules.NewCondition("isAdmin", func(ctx context.Context) bool {
    user, ok := rules.GetAs[User](ctx)
    if !ok {
        return false
    }
    return user.Role == "admin"
})
```

**Type-safe condition:**

```go
myCondition := rules.NewTypedCondition[User]("isAdult", func(ctx context.Context, user User) bool {
    return user.Age >= 18
})
```

**Type-safe condition with Prepare (non-pure):**

Use this when you need to load data before evaluating the condition (e.g., database lookups, API calls):

```go
myCondition := rules.NewTypedConditionWithPrepare[User, Permissions](
    "userHasPermission",
    func(ctx context.Context, user User) (Permissions, error) {
        // Prepare: Load permissions from database
        return db.LoadPermissions(ctx, user.ID)
    },
    func(ctx context.Context, perms Permissions) bool {
        // Evaluate: Check if user has edit permission
        return perms.CanEdit
    },
)
```

⚠️ **Concurrency Warning:** `NewTypedConditionWithPrepare` stores state (`loadedData`). When validating multiple items, create one tree per target, or use pure conditions for shared trees:

```go
// ✅ CORRECT: One tree per validation
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

### Creating Custom Rules (Closure-Based - Old Style)

```go
myRule := rules.NewRulePure("myRule", func() error {
    if user.Disabled {
        return fmt.Errorf("user is disabled")
    }
    return nil
})
```

### Creating Custom Conditions (Closure-Based - Old Style)

**Pure conditions** (no side effects):

```go
myCondition := rules.NewConditionPure("isAdmin", func() bool {
    return user.Role == "admin"
})
```

**Impure conditions** (with side effects like fetching data):

Use `NewConditionSideEffect()` to create conditions that fetch data before evaluating:

```go
var user User // closure variable to share state between Prepare and IsValid

tree := rules.Node(
    rules.NewConditionSideEffect(
        "userActive",
        func(ctx context.Context) error {
            // Side effect: fetch user from database
            var err error
            user, err = db.GetUser(ctx, userID)
            return err
        },
        func(ctx context.Context) bool {
            // Check if user is active
            return user.Active
        },
    ),
    rules.Rules(validators.RuleValidEmail("email", user.Email, nil)),
)
```

The `IsPure()` method is important:
- Return `true` if the condition has no side effects — the engine may skip calling `Prepare()` for optimization
- Return `false` if the condition has side effects — `Prepare()` will always be called before `IsValid()`

### Main Validation Functions

```go
// Standard validation (requires registry in context)
err := rules.Validate(ctx, tree, hooks, "validationName")

// Convenience: validate with data directly
err := rules.ValidateWithData(ctx, tree, hooks, "name", data)

// Validate multiple targets
err := rules.ValidateMultiWithData(ctx, []struct{Tree rules.Evaluable; Data any}{...}, hooks, "name")
```

### Error Handling

Validation errors include the field name, message, and code:

```go
type Error struct {
    Field string // Field name
    Err   string // Error message
    Code  string // Error code for internationalization
}
```

---

## Performance

The rules engine is designed for flexibility first, but performs well for high-throughput scenarios:

| Operation | Speed | Allocations |
|-----------|-------|-------------|
| `IsA[T]()` type check | ~6-7 ns/op | 0 |
| `GetAs[T]()` data access | ~6-7 ns/op | 0 |
| Full tree evaluation | ~500-1000 ns/op | ~10-17 |

For 1000s of evaluations per second:

1. **`IsA[T]()` is optimized** — It caches the target type, making reflection overhead negligible (~6ns)
2. **Use `ValidateMulti`** — Batch validations to amortize context creation cost
3. **Avoid deep nesting** — Each level adds overhead; flatten where possible
4. **Prefer direct field access** — Use `NewTypedRule` instead of `validators.Schema` for hot paths

See [PERFORMANCE.md](PERFORMANCE.md) for detailed benchmarks and optimization guides.

### Fast Type Switching

For maximum performance with multiple types, use `FastTypeSwitch`:

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

### Batching Validations

```go
// Process 1000s of items efficiently
targets := make([]rules.Target, len(items))
for i, item := range items {
    reg := rules.NewDataRegistry(item)
    targets[i] = rules.Target{
        // ... configure target
    }
}
err := rules.ValidateMulti(ctx, targets, hooks, "batch")
```

---

## Installation

```bash
go get github.com/mishudark/rules
```

---

## Best Practices

1. **Use Data Registry for reusable trees** — When you need to validate multiple instances against the same rules, use `NewRule`, `NewTypedRule`, and `ValidateWithData`.

2. **Use closures for one-off validations** — For simple, single-use validations, `NewRulePure` and `NewConditionPure` are fine.

3. **Use `IsA[T]` for type switching** — When merging trees from different packages, use `IsA[YourType]()` to route validation correctly.

4. **Prefer type-safe rules when possible** — `NewTypedRule[T]` gives you compile-time type safety within the rule function.

5. **Use schema validators for struct validation** — `validators.Schema` with field validators provides clean, declarative struct validation.
