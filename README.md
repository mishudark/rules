# Go Rules Engine: Flexible Validation Trees


This library helps you build flexible and powerful validation logic in Go. Think of it like creating decision trees where you can define conditions ("IF this is true...") and corresponding validation rules ("THEN check these things..."). It's designed to be clear, extensible, and easy to follow.

## ✨ Quick Start: Usage Example

Let's dive right in with an example. Imagine you want to validate user data based on certain criteria:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/mishudark/rules"
)

// Mock User Data
type User struct {
	Name    string
	Age     int
	Zipcode string
	Country string
}

// Define Your Conditions (The "IFs")

// Condition: Check if the name is not empty
func nameNotEmpty(name string) rules.Condition {
	return rules.NewConditionPure("nameNotEmpty", func() bool {
		return name != ""
	})
}

// Condition: Check if country is USA
func countryIsUSA(country string) rules.Condition {
	return rules.NewConditionPure("countryIsUSA", func() bool {
		return strings.ToUpper(country) == "USA"
	})
}

// Define Your Rules (The "THEN Checks")

// Rule: Check if age is greater than 20
func ageGreaterThan20(age int) rules.Rule {
	return rules.NewRulePure("ageGreaterThan20", func() error {
		if age <= 20 {
			return rules.Error{Field: "Age", Err: "Age must be greater than 20", Code: "AGE_TOO_LOW"}
		}
		return nil // Success!
	})
}

// Rule: Check if zipcode is valid (simple example: 5 digits)
func ruleValidZipCode(zip string) rules.Rule {
	return rules.NewRulePure("ruleValidZipCode", func() error {
		if len(zip) != 5 {
			return rules.Error{Field: "Zipcode", Err: "Zipcode must be 5 digits", Code: "INVALID_ZIP"}
		}
		// Add more sophisticated zip checks if needed
		return nil // Success!
	})
}

// Rule: Action to take if the user is generally invalid (e.g., name empty)
func ruleInvalidUser() rules.Rule {
	return rules.NewRulePure("ruleInvalidUser", func() error {
		return errors.New("invalid user configuration detected")
		// Or return a structured error:
		// return rules.Error{Field: "User", Err: "General invalid user data", Code: "INVALID_USER"}
	})
}


func main() {
	// Example User
	user := User{
		Name:    "Alice",
		Age:     22,
		Zipcode: "12345",
		Country: "USA",
	}

    // This tree defines the logic:
    // - IF name is not empty:
    //   - THEN check the ZipCode validity.
    //   - AND IF the country is USA:
    //     - THEN check if age > 20.
    // - IF name IS empty (using Not):
    //   - THEN run the 'invalid user' rule.
	tree := rules.Root( // Root usually acts like an "AnyOf" - at least one main branch must succeed
		rules.Node( // A conditional node: IF Condition THEN check children
			nameNotEmpty(user.Name), // The Condition for this branch
			rules.Rules(ruleValidZipCode(user.Zipcode)), // A rule to run if the condition is true
			rules.Node( // A nested conditional node
				countryIsUSA(user.Country), // Nested condition: only checked if parent condition (nameNotEmpty) was true
				rules.Rules( // The rules to run if THIS nested condition is also true
					ageGreaterThan20(user.Age),
				),
			),
		),
		rules.Node( // Another top-level branch
			rules.Not(nameNotEmpty(user.Name)), // Condition: IF name IS empty
			rules.Rules(ruleInvalidUser()),    // Rule to run if name is empty
		),
	)

    // (Optional) Define Hooks
    // Hooks let you run code at different stages of the validation process
    hooks := rules.ProcessingHooks {
        AfterPrepareConditions: func(ctx context.Context) {
            log.Println("Hooks: Finished preparing all conditions.")
        },
        AfterEvaluateConditions: func(ctx context.Context) {
            log.Println("Hooks: Finished evaluating conditions.")
        },
        AfterPrepareRules: func(ctx context.Context) {
            log.Println("Hooks: Finished preparing the selected rules.")
        },
        AfterValidateRules: func(ctx context.Context) {
            log.Println("Hooks: Finished validating the selected rules.")
        },
    }

	// Validate the Tree
	ctx := context.Background()
	// Give your validation process a name for context (e.g., in logs or rule paths)
	validationName := "userValidationProcess"
	err := rules.Validate(ctx, tree, hooks, validationName)

	// Check for Validation Errors
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Validation completed successfully!")
	}
}
```

## 📚 Documentation

🤔 How It Works: The Friendly Guide
Okay, so how does this magic happen? Let's break it down.

Imagine you're building a flowchart for checking things. This library helps you build that flowchart in code.

- **The Big Picture: The Tree** (`Evaluable`)
    - Everything starts with a Tree. This tree is made up of different kinds of Nodes.
    - The rules.Validate function is the engine that walks through your tree.
    - Each part of the tree that can be checked or evaluated is called an Evaluable.
- **Decision Points: Conditions (`Condition`)**
    - How does the tree decide which path to take? With **Conditions**!
    - A `Condition` is like a question: "Is this true?". It has an `IsValid()` method that returns `true` or `false`.
    - You create conditions using helpers like `NewConditionPure` for simple checks, or you can implement the `Condition` interface yourself for more complex logic (like fetching data in the `Prepare` step before checking `IsValid`).
    - We also have `rules.Not(condition)` which flips the result of a condition (like "IF name is NOT empty").
- **Conditional Branches: `ConditionNode` (`rules.Node`)**
    - This is the most common building block. It pairs a `Condition` with one or more children (`Evaluables`).
    - **IF** the `Condition` is `true`, **THEN** the engine looks at the children of this node.
    - You create these using `rules.Node(myCondition, child1, child2, ...)`.
- **Actions: Rules (`Rule`)**
    - When the engine follows a path down the tree and reaches the end of a branch (or a specific point), it finds the **Rules** it needs to execute.
    - A `Rule` represents a single validation check (e.g., "is the age over 20?").
    - It has a `Validate()` method. If the check fails, `Validate()` returns an `error` (preferably a `rules.Error` with details). If it passes, it returns `nil`.
    - Rules can also have a `Prepare()` step, just like conditions, for any setup needed before validation.
    - You define rules using helpers like `NewRulePure` for simple functions, or by implementing the `Rule` interface.
- **Rule Containers: `LeafNode` (`rules.Rules`)**
    - Often, when a condition is met, you want to run one or more rules.
    - `rules.Rules(myRule1, myRule2)` creates a simple container (`LeafNode`) that holds these rules. It's placed as a child in the tree (often inside a `ConditionNode`). It always evaluates to `true` (meaning "yes, run these rules if you reach me") and provides its list of rules.
- **Logical Grouping: `AllOfNode` (`rules.AllOf`) and `AnyOfNode` (`rules.AnyOf`, `rules.Root`)**
    - Sometimes you need more complex logic than just one condition:
        - `rules.AllOf(child1, child2)`: All of these children must evaluate successfully for the AllOf node to succeed. Think **AND**.
        - `rules.AnyOf(child1, child2)`: At least one of these children must evaluate successfully for the AnyOf node to succeed. Think **OR**.
        - `rules.Root(...)`: This is often used for the very top of your tree and typically behaves like `AnyOf`.

## 🧩 Core Concepts & Components

Here's a quick reference to the main pieces:

* **`Evaluable`**: The interface for any part of the tree that can be evaluated (Nodes, Rule sets). Has `PrepareConditions` and `Evaluate` methods.
* **`Condition`**: Interface for checks that return `true` or `false`. Has `Prepare`, `Name`, and `IsValid` methods.
    * `ConditionPure`: A simple implementation using just a function.
    * `NotCondition`: Wraps another condition to negate its result.
* **`Rule`**: Interface for the actual validation logic. Has `Name`, `Prepare`, `Validate`, `SetExecutionPath`, and `GetExecutionPath` methods.
    * `RulePure`: A simple implementation using just a function for `Validate`.
    * `ChainRules`: Executes multiple rules in sequence.
    * `RuleBase`: Embeddable helper for handling execution paths.
* **Nodes**:
    * `ConditionNode`: Links a `Condition` to child `Evaluable`s. (`rules.Node`)
    * `LeafNode`: Holds a list of `Rule`s to be executed. (`rules.Rules`)
    * `AllOfNode`: Logical AND for child `Evaluable`s. (`rules.AllOf`)
    * `AnyOfNode`: Logical OR for child `Evaluable`s. (`rules.AnyOf`, `rules.Root`)
* **`Error`**: A struct for detailed validation errors.
* **`ProcessingHooks`**: Struct holding functions to hook into the validation lifecycle.
* **`Validate(...)`**: The main function to execute the validation against a tree.

## ✅ Available Validators

Here is a list of the currently available validators in this library:

- Comma Separated Integer List Validator
- Content Type
- Decimal Validator
- Domain
- Email
- File Extension Validator
- Ip Address Validator
- Length
- Max Value
- Min Value
- Prohibit Null Characters Validator
- Slug
- Step Value Validator
- Url Validator

## ✨ Creating Custom Conditions and Rules
The real power comes when you implement the Condition and Rule interfaces yourself!

**Custom Condition Example:**
```go
type UserExistsCondition struct {
    UserID    string
    UserAPI   YourUserAPIClient // Assuming you have an API client
    userFound bool              // Store result after Prepare
    name      string            // Store name for logging/debugging
}

func NewUserExistsCondition(userID string, apiClient YourUserAPIClient) rules.Condition {
    return &UserExistsCondition{UserID: userID, UserAPI: apiClient, name: fmt.Sprintf("UserExists[%s]", userID)}
}

func (c *UserExistsCondition) Name() string {
    return c.name
}

// Prepare is great for fetching data needed for the condition
func (c *UserExistsCondition) Prepare(ctx context.Context) error {
    exists, err := c.UserAPI.CheckIfUserExists(ctx, c.UserID)
    if err != nil {
        // Maybe return an error if the API call fails, stopping validation early
        return fmt.Errorf("failed to check user existence for %s: %w", c.UserID, err)
    }
    c.userFound = exists
    log.Printf("Prepared %s: User found = %t\n", c.Name(), c.userFound)
    return nil // Prepare succeeded
}

// IsValid uses the data fetched during Prepare
func (c *UserExistsCondition) IsValid(ctx context.Context) bool {
    // The check logic is now very simple!
    return c.userFound
}

var _ rules.Condition = (*UserExistsCondition)(nil) // Compile-time check
```

**Custom Rule Example:**
```go
type DatabaseSyncRule struct {
    rules.RuleBase // Embed for Get/SetExecutionPath
    Data          YourDataType
    DB            YourDatabaseClient
    name          string
}

func NewDatabaseSyncRule(data YourDataType, db YourDatabaseClient) rules.Rule {
	return &DatabaseSyncRule{Data: data, DB: db, name: "DatabaseSyncRule"}
}

func (r *DatabaseSyncRule) Name() string {
	return r.name
}

// Prepare could potentially check DB connection, etc.
func (r *DatabaseSyncRule) Prepare(ctx context.Context) error {
    log.Printf("Preparing Rule: %s at path %s\n", r.Name(), r.GetExecutionPath())
	err := r.DB.Ping(ctx)
	if err != nil {
		return fmt.Errorf("database connection check failed for %s: %w", r.Name(), err)
	}
	return nil
}

// Validate performs the actual database operation
func (r *DatabaseSyncRule) Validate(ctx context.Context) error {
    log.Printf("Validating Rule: %s at path %s\n", r.Name(), r.GetExecutionPath())
	err := r.DB.SyncData(ctx, r.Data)
	if err != nil {
        // Return a structured error
		return rules.Error{
			Field: "DatabaseSync",
			Err:   fmt.Sprintf("Failed to sync data: %v", err),
			Code:  "DB_SYNC_FAILED",
		}
	}
	return nil // Success!
}

var _ rules.Rule = (*DatabaseSyncRule)(nil) // Compile-time check
```
