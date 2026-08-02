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
- **Key metric indicators (KMIs)** — counters, histograms, and scores computed in the same tree as validation


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

Validation executes in **4 strictly separated phases**:

```
PrepareConditions → Evaluate → Prepare → Validate
```

| Phase | What happens |
|-------|-------------|
| **PrepareConditions** | Conditions with side effects (impure) fetch data from DB/API and return it. Pure conditions skip this. |
| **Evaluate** | Walk the tree: check conditions, collect candidate rules |
| **Prepare** | Each candidate rule fetches data it needs and returns it |
| **Validate** | Run each rule's validation logic, reading its prepared data back typed via `GetPreparedAs[T]`, collect errors |

The phases never interleave: **every** condition is prepared before any rule is prepared, and **every** rule is prepared before any validation runs. `ValidateMulti` extends this across targets — all targets' conditions are prepared before any evaluation, and all targets' rules before any validation.

**Data injection:** `Prepare` returns the data it retrieved (`any`) and records it in a per-evaluation preparedStore (keyed by the rule/condition instance). `Validate` and `IsValid` read that data back typed via `GetPreparedAs[T]` (or untyped via `GetPrepared`). Rules and conditions keep no state — a single tree can be reused and shared across goroutines, and dataloaders can fan out every fetch in the Prepare sweep.

```go
data, ok := rules.GetPreparedAs[Permissions](ctx, r) // typed read
```

### Dataloader-friendly by design

This ordering exists so `Prepare` implementations can fan out fetches through a [dataloader](https://github.com/graph-gophers/dataloader)-style batcher and pay a single round-trip per phase instead of one per node:

- **Impure `Node` conditions**: children are prepared even when the condition turns out false — short-circuiting would serialize fetches across branches (N+1).
- **Impure `Either` conditions**: **both** branches are prepared, for the same reason.
- **Pure conditions**: evaluated immediately during `PrepareConditions`; a pure-false condition prunes its whole subtree (safe, because pure means no fetches to lose).
- **Composite rules** (`Or`, `ChainRules`): `Prepare` runs on **all** children regardless of their short-circuit `Validate` semantics — preparation is setup work.

```go
rule := rules.NewTypedRuleWithPrepare(
    "checkEmailUnique",
    func(ctx context.Context, u User) (EmailData, error) {
        return loader.Load(ctx, u.Email) // batched with all other Prepare fetches
    },
    func(ctx context.Context, u User, data EmailData) error {
        if data.Exists { return fmt.Errorf("email already in use") }
        return nil
    },
)
```

Hooks can be injected at each phase boundary via `ProcessingHooks` — a natural place to flush a dataloader:

```go
hooks := rules.ProcessingHooks{
    AfterPrepareConditions:  func(ctx context.Context) error { loader.Flush(); return nil },
    AfterEvaluateConditions: func(ctx context.Context) error { log.Println("evaluated"); return nil },
    AfterPrepareRules:       func(ctx context.Context) error { loader.Flush(); return nil },
    AfterValidateRules:      func(ctx context.Context) error { log.Println("validated"); return nil },
}
```

Hook error semantics: errors from the first three hooks halt validation immediately; an `AfterValidateRules` error is joined with the collected validation errors via `errors.Join` and returned together.

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

> 💡 **Empty values are valid by convention.** All validators (e.g. `Email`, `URL`) treat an empty string as valid — they check *format*, not *presence*. If a field is required, add a separate presence check.

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

## Key Metric Indicators (KMIs)

The same tree engine also computes key metric indicators dynamically. Instead
of only returning pass/fail, a rule can **carry metric outcomes** — counters,
histograms, scores, or valid/invalid observations — alongside its validation
result. Use `EvaluateMetrics` (or `EvaluateMetricsMulti` for batches) instead
of `Validate`:

```go
tree := rules.Root(
    rules.Node(
        rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" }),
        rules.Rules(
            rules.NewMetricRulePure("mrr", rules.KindCounter, "mrr", func() (rules.Outcome, error) {
                return rules.CounterValue(1250.5), nil
            }),
        ),
    ),
    rules.Rules(
        rules.NewTypedMetricRule[User]("engagement", rules.KindScore, "engagement",
            func(ctx context.Context, u User) (rules.Outcome, error) {
                return rules.ScoreValue(u.Engagement, 1), nil
            }),
        rules.NewTypedMetricRule[User]("latency", rules.KindHistogram, "latency_ms",
            func(ctx context.Context, u User) (rules.Outcome, error) {
                hist := rules.NewHistogram([]float64{50, 100, 250, 1000, math.Inf(1)})
                for _, v := range u.Latencies { hist.Observe(v) }
                return rules.HistogramValue(hist), nil
            }),
        rules.NewTypedMetricRule[User]("compliant", rules.KindValid, "compliant",
            func(ctx context.Context, u User) (rules.Outcome, error) {
                return rules.ValidValue(u.Compliant, nil), nil
            }),
    ),
)

report, err := rules.EvaluateMetricsWithData(ctx, tree, hooks, "health", user)
// report.Valid, report.Errors, report.Metrics["mrr"].Count,
// report.Metrics["latency"].Histogram, report.Metrics["engagement"].Score
```

A metric-carrying rule returns both an `Outcome` and an `error`, so a single
rule both computes the KMI and decides pass/fail. A rule that fails still
reports its observation (e.g. counting attempts), and the error is surfaced
in `report.Errors`.

**Extending any rule:** any `Rule` can carry metrics by calling `rules.Emit`
from its `Validate` method — no dedicated constructor needed:

```go
rule := rules.NewTypedRule[User]("itemsInOrder", func(ctx context.Context, u User) error {
    rules.Emit(ctx, rules.CounterValue(float64(len(u.Order.Items))))
    return nil
})
```

**Aggregation:** same-name outcomes are combined when the report is built.
Defaults are kind-specific — counters sum, histograms merge bucket-wise,
scores weight-average — and can be overridden via the `Aggregation` field on
`Outcome`.

**Batching:** the `Prepare` step of `NewTypedMetricRuleWithPrepare` runs in
the same rule-prepare phase, so a dataloader batches metric fetches together
with rule and condition fetches in a single round-trip. Outcomes are only
collected by `EvaluateMetrics`; `Validate` ignores them, so validation-only
callers are unaffected.

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
| `rules.EvaluateMetrics(ctx, tree, hooks, name)` | Evaluates tree, returns `(Report, error)` with aggregated metrics |
| `rules.EvaluateMetricsWithData(ctx, tree, hooks, name, data)` | Evaluates with data (convenience) |
| `rules.EvaluateMetricsMulti(ctx, targets, hooks, name)` | Batch evaluation, one `Report` per target |
| `rules.EvaluateMetricsMultiWithData(ctx, targets, hooks, name, ...data)` | Batch evaluation with data |
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

### Metric-Carrying Rules

| Function | Description |
|----------|-------------|
| `rules.NewMetricRulePure(name, kind, field, fn)` | Closure-based rule carrying a metric (pure, legacy) |
| `rules.NewMetricRule(name, kind, field, fn)` | Rule carrying a metric with `any` data (pure) |
| `rules.NewTypedMetricRule[T](name, kind, field, fn)` | Type-safe rule carrying a metric (pure) |
| `rules.NewTypedMetricRuleWithPrepare[In, T](name, kind, field, prepare, fn)` | Type-safe rule carrying a metric with Prepare (impure) |
| `rules.Emit(ctx, outcome)` | Record a metric outcome from any rule's `Validate` |
| `rules.CounterValue(v)` | Build a counter `Outcome` |
| `rules.ScoreValue(score, weight)` | Build a score `Outcome` |
| `rules.HistogramValue(h)` | Build a histogram `Outcome` |
| `rules.ValidValue(valid, err)` | Build a valid/invalid `Outcome` |
| `rules.NewHistogram(buckets)` | Create an empty histogram with le boundaries |

`Kind` values: `KindValid`, `KindCounter`, `KindHistogram`, `KindScore`. Outcomes are aggregated by name in the returned `Report.Metrics`; see [Key Metric Indicators](#key-metric-indicators-kmis).

### Condition Constructors

| Function | Description |
|----------|-------------|
| `rules.NewCondition(name, fn)` | Data-driven condition (pure) |
| `rules.NewConditionSideEffect[T](name, prepare, condition)` | Condition with side effects (impure), typed loaded data |
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

**Basic rule (typed):**

```go
myRule := rules.NewTypedRule[User]("myRule", func(ctx context.Context, user User) error {
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

✅ **Concurrency:** rules and conditions never store prepared state — `Prepare` records its retrieved data in a per-evaluation preparedStore (keyed by the rule/condition instance), and `Validate`/`IsValid` read it back typed via `GetPreparedAs[T]`. Every tree, including ones built with `NewTypedRuleWithPrepare` and `NewTypedConditionWithPrepare`, is **safe to share across goroutines**. Build the tree once and validate many targets concurrently:

```go
// ✅ Correct: One tree, many targets concurrently
tree := buildTree()
for _, user := range users {
    go func(u User) {
        err := rules.ValidateWithData(ctx, tree, hooks, "validate", u) // safe
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
    rules.NewConditionSideEffect[User](
        "userActive",
        func(ctx context.Context) (User, error) {
            return db.GetUser(ctx, userID)
        },
        func(ctx context.Context, user User) bool {
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
| `INVALID_URL_FORMAT`, `URL_SCHEME_NOT_ALLOWED` | `URL` |
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

Across the batch, all targets' conditions are prepared before any rule preparation, so dataloader fetches for the entire batch can be coalesced.

---

## Execution Path Tracing

For debugging and logging, you can record the path each rule took through the tree. Tracing is opt-in and race-free — rules are never mutated during evaluation:

```go
ctx, trace := rules.WithExecutionTrace(ctx)
err := rules.ValidateWithData(ctx, tree, hooks, "validate", user)

fmt.Println(trace.Path(rule))
// "validate -> root -> isPremium -> leafNode -> checkAge"
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

7. **Share trees freely, even across goroutines** — rules and conditions hold no mutable state: `Prepare` records its retrieved data in the per-evaluation preparedStore (keyed by the rule/condition instance), and `Validate`/`IsValid` read it back typed via `GetPreparedAs[T]`. A tree built once (including `NewTypedRuleWithPrepare` and `NewTypedConditionWithPrepare`) can be validated concurrently against different targets. Cache trees globally.

8. **Inject, don't store** — `Prepare(ctx) (any, error)` retrieves the data and records it in the per-evaluation preparedStore (via `rules.PutPrepared`); `Validate(ctx)` / `IsValid(ctx)` read it back typed via `rules.GetPreparedAs[T](ctx, r)`. Never cache prepared data on the rule or condition: that is what made trees unsafe to share, and it defeats the dataloader fan-out (a dataloader batches every Prepare call across the whole tree and across targets in `ValidateMulti`).
