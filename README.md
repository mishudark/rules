# Go Rules Engine: Flexible Validation & Decision Trees

The Go Rules Engine is a powerful and extensible Go library designed to simplify the creation of complex validation and decision-making logic. It allows you to define flexible rule trees, enabling clear separation of concerns between conditions ("IF this is true...") and the actions or validations that follow ("THEN perform these checks...").

**Key Features:**

*   **Intuitive Tree Structure:** Build decision flows with `Condition`s and `Rule`s organized into a readable tree.
*   **Extensible Components:** Easily define custom conditions and rules to fit your specific business logic.
*   **Clear Separation of Concerns:** Distinguish between `Condition` evaluation and `Rule` execution.
*   **Context-Aware Processing:** Leverage `context.Context` for managing timeouts, cancellations, and request-scoped data.
*   **Detailed Error Reporting:** Capture and report specific validation errors with `rules.Error`.
*   **Lifecycle Hooks:** Integrate custom logic at various stages of the validation process.

This library is ideal for scenarios requiring dynamic validation, A/B testing logic, feature flagging, or any system where decisions are based on a set of defined criteria.

## 🚀 Getting Started

Here's a simple example to get you started. Let's say you want to validate a user's age and country.

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mishudark/rules"
	"github.com/mishudark/rules/validators"
)

// Mock User Data
type User struct {
	Age     int
	Country string
}

func main() {
	user := User{
		Age:     25,
		Country: "USA",
	}

	// Condition: Check if country is USA
	countryIsUSA := rules.NewConditionPure("countryIsUSA", func() bool {
		return strings.ToUpper(user.Country) == "USA"
	})

	// Rule: Check if age is greater than 20
	ageGreaterThan20 := validators.RuleMinValue("Age", user.Age, 21)

	// This tree defines the logic:
	// - IF the country is USA:
	//   - THEN check if age >= 21.
	tree := rules.Node(
		countryIsUSA,
		rules.Rules(ageGreaterThan20),
	)

	// Validate the Tree
	err := rules.Validate(context.Background(), tree, nil, "userValidation")

	// Check for Validation Errors
	if err != nil {
		fmt.Printf("Validation failed: %v
", err)
	} else {
		fmt.Println("Validation completed successfully!")
	}
}
```

## 📚 Core Concepts

The library is built around a few core concepts:

*   **`Evaluable`**: The interface for any part of the tree that can be evaluated (Nodes, Rule sets).
*   **`Condition`**: Represents a check that returns `true` or `false`.
*   **`Rule`**: Represents the actual validation logic.
*   **Nodes**:
    *   `ConditionNode`: Links a `Condition` to child `Evaluable`s. (`rules.Node`)
    *   `LeafNode`: Holds a list of `Rule`s to be executed. (`rules.Rules`)
    *   `AllOfNode`: Logical AND for child `Evaluable`s. (`rules.AllOf`)
    *   `AnyOfNode`: Logical OR for child `Evaluable`s. (`rules.AnyOf`, `rules.Root`)
*   **`Validate(...)`**: The main function to execute the validation against a tree.
## ✅ Available Validators

Here is a list of the currently available validators in this library, with examples for each.

### Comma Separated Integer List Validator

Validates that a string is a comma-separated list of integers.

```go
rule := validators.NewValidateCommaSeparatedIntegerList("1,2,3,4")
// err == nil

rule = validators.NewValidateCommaSeparatedIntegerList("1,2,a,4")
// err != nil
```

### Content Type

Checks if the content type of a file matches one of the allowed MIME types.

```go
file := strings.NewReader("this is a plain text file")
rule := validators.NewRuleContentType("file", file, []string{"text/plain"})
// err == nil

file = strings.NewReader("this is a plain text file")
rule = validators.NewRuleContentType("file", file, []string{"image/jpeg"})
// err != nil
```

### Decimal Validator

Validates a decimal number string against `max_digits` and `decimal_places`.

```go
rule := validators.NewDecimalValidator("123.45", 5, 2)
// err == nil

rule = validators.NewDecimalValidator("123.456", 5, 2)
// err != nil
```

### Domain

Validates that a string is a valid domain name.

```go
rule := validators.RuleValidDomainNameAdvanced("domain", "example.com", false)
// err == nil

rule = validators.RuleValidDomainNameAdvanced("domain", "example..com", false)
// err != nil
```

### Email

Validates that a string is a valid email address.

```go
rule := validators.RuleValidEmail("email", "test@example.com", nil)
// err == nil

rule = validators.RuleValidEmail("email", "not-an-email", nil)
// err != nil
```

### File Extension Validator

Validates that a filename has an allowed extension.

```go
rule := validators.NewFileExtensionValidator("document.pdf", []string{"pdf", "docx"})
// err == nil

rule = validators.NewFileExtensionValidator("image.jpg", []string{"pdf", "docx"})
// err != nil
```

### IP Address Validator

Validates that a string is a valid IPv4, IPv6 or any IP address.

```go
rule := validators.NewValidateIPv4Address("192.168.1.1")
// err == nil

rule = validators.NewValidateIPv6Address("2001:0db8:85a3:0000:0000:8a2e:0370:7334")
// err == nil

rule = validators.NewValidateIPv46Address("192.168.1.1")
// err == nil
```

### Length

Validates the length of a string or a slice.

```go
rule := validators.MinLengthString("name", "test", 3)
// err == nil

rule = validators.MaxLengthString("name", "testing", 5)
// err != nil

rule = validators.MinLengthSlice("items", []any{1, 2, 3}, 2)
// err == nil

rule = validators.MaxLengthSlice("items", []any{1, 2, 3, 4}, 3)
// err != nil
```

### Max Value

Validates that a numeric value is less than or equal to a specified maximum value.

```go
rule := validators.RuleMaxValue("age", 25, 30)
// err == nil

rule = validators.RuleMaxValue("age", 35, 30)
// err != nil
```

### Min Value

Validates that a numeric value is greater than or equal to a specified minimum value.

```go
rule := validators.RuleMinValue("age", 25, 20)
// err == nil

rule = validators.RuleMinValue("age", 15, 20)
// err != nil
```

