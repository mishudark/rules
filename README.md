# Go Rules Engine

A flexible validation library that lets you build complex rules as a tree structure. Think of it as "if this, then that" for validation logic.

## When to use this?

- **Feature flags** — enable features based on user attributes
- **A/B testing** — route users to different experiences
- **Form validation** — validate complex forms with conditions
- **Business rules** — implement decision trees that non-developers can visualize

---

## Quick Examples

### Simple validation

Validate that a user is over 21 and from the USA:

```go
user := User{Age: 25, Country: "USA"}

tree := rules.Node(
    rules.NewConditionPure("fromUSA", func() bool {
        return user.Country == "USA"
    }),
    rules.Rules(validators.RuleMinValue("age", user.Age, 21)),
)

err := rules.Validate(context.Background(), tree, rules.ProcessingHooks{}, "check")
// err == nil
```

### Multiple rules (all must pass)

Pass multiple rules to `Rules()` — all of them must pass:

```go
tree := rules.Rules(
    validators.RuleMinValue("age", 25, 18),
    validators.RuleMaxValue("age", 25, 65),
    validators.RuleValidEmail("email", "user@example.com", nil),
)
```

### At least one rule must pass

Use `Or()` when at least one rule should pass:

```go
tree := rules.Rules(
    rules.Or(
        validators.RuleValidEmail("contact", "user@example.com", nil),
        validators.RuleValidDomainNameAdvanced("contact", "example.com", false),
    ),
)
```

### Either/Then (if-else)

Use `Either()` for if-else logic — if condition is true, evaluate left branch; otherwise, evaluate right branch:

```go
// IF user is premium, require age 18+ AND website; ELSE require just age 13+
tree := rules.Either(
    rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" }),
    // Left branch (condition is true): premium rules
    rules.Rules(
        validators.RuleMinValue("age", user.Age, 18),
        validators.URLValidator(user.Website, []string{"https"}),
    ),
    // Right branch (condition is false): free user rules
    rules.Rules(validators.RuleMinValue("age", user.Age, 13)),
)
```

### Nested logic

Combine conditions into complex trees:

```go
tree := rules.Root(
    // IF user is premium
    rules.Node(
        rules.NewConditionPure("isPremium", func() bool { return user.IsPremium }),
        // THEN require email verification
        rules.Rules(validators.RuleValidEmail("email", user.Email, nil)),
    ),
    // IF user is NOT premium
    rules.Node(
        rules.NewConditionPure("isNotPremium", func() bool { return !user.IsPremium }),
        // THEN just check minimum age
        rules.Rules(validators.RuleMinValue("age", user.Age, 13)),
    ),
)
```

### More complex nesting

```go
tree := rules.Root(
    rules.Rules(validators.RuleValidEmail("email", user.Email, nil)),
    
    // Premium users: age 18+ AND valid website
    rules.Node(
        rules.NewConditionPure("isPremium", func() bool { return user.Plan == "premium" }),
        rules.Rules(
            validators.RuleMinValue("age", user.Age, 18),
            validators.URLValidator(user.Website, []string{"https"}),
        ),
    ),
    
    // Free users: age 13+ AND valid country
    rules.Node(
        rules.NewConditionPure("isFree", func() bool { return user.Plan == "free" }),
        rules.Rules(
            validators.RuleMinValue("age", user.Age, 13),
            validators.RuleValidDomainNameAdvanced("country", user.Country, false),
        ),
    ),
)
```

---

## Common Validators

Here's what you can validate out of the box:

| Validator | Use for |
|-----------|---------|
| `RuleMinValue` / `RuleMaxValue` | Numbers (age, price, quantity) |
| `RuleValidEmail` | Email addresses |
| `URLValidator` | URLs with optional scheme restrictions |
| `RuleValidDomainNameAdvanced` | Domain names |
| `MinLengthString` / `MaxLengthString` | String sizes |
| `MinLengthSlice` / `MaxLengthSlice` | Array sizes |
| `NewFileExtensionValidator` | File types by extension |
| `NewValidateIPv4Address` / `NewValidateIPv6Address` | IP addresses |
| `NewDecimalValidator` | Decimal numbers with precision control |
| `NewValidateCommaSeparatedIntegerList` | Comma-separated numbers |
| `NewProhibitNullCharactersValidator` | Strings without null chars |
| `NewSlugValidator` | URL-friendly slugs |
| `NewStepValueValidator` | Values in increments |

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
        rules.Rules(validators.RuleValidEmail("email", req.Email, nil)),
        
        // Age check for everyone
        rules.Rules(validators.RuleMinValue("age", req.Age, 13)),
        
        // Premium users need a valid website
        rules.Node(
            rules.NewConditionPure("isPremium", func() bool { return req.Plan == "premium" }),
            rules.Rules(validators.URLValidator(req.Website, []string{"https"})),
        ),
        
        // US users need 18+, others need 21+
        rules.Node(
            rules.NewConditionPure("isUS", func() bool { return req.Country == "US" }),
            rules.Rules(validators.RuleMinValue("age", req.Age, 18)),
        ),
        
        // Non-US users need 21+ (only runs if isUS is false)
        rules.Node(
            rules.NewConditionPure("isNotUS", func() bool { return req.Country != "US" }),
            rules.Rules(validators.RuleMinValue("age", req.Age, 21)),
        ),
    )
    
    return rules.Validate(ctx, tree, rules.ProcessingHooks{}, "registration")
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

### Main Function

```go
err := rules.Validate(ctx, tree, hooks, "validationName")
```

- `ctx` — Context for timeouts/cancellation
- `tree` — Your rule tree (any `Evaluable`)
- `hooks` — Optional hooks structure for processing
- `validationName` — Name for error reporting

### Creating Custom Rules

```go
myRule := rules.NewRulePure("myRule", func(ctx context.Context, data any) error {
    user := data.(User)
    if user.Disabled {
        return fmt.Errorf("user is disabled")
    }
    return nil
})
```

### Creating Custom Conditions

```go
myCondition := rules.NewConditionPure("isAdmin", func() bool {
    return user.Role == "admin"
})
```

### Error Handling

Validation errors include the rule name and message:

```go
type Error struct {
    Rule   string
    Reason string
}
```

---

## Installation

```bash
go get github.com/mishudark/rules
```
