# Go Rules Engine

This Go package provides a flexible and powerful system for defining and evaluating 
hierarchical rule trees. It allows you to build complex conditional logic using conditions
and rules, organized with logical operators like AND/OR.

---

## 🚀 Features

- Define custom conditions and rules.
- Build logical trees with AND/OR nodes.
- Validate and evaluate rules with hooks for customization.
- Supports hierarchical and nested conditions.

---

## 📖 How It Works

### Key Components

1. **`Condition`**  
   A boolean check that determines if a branch of the tree should be followed.  
   **Example:** Check if a user is older than 30.  

2. **`Rule`**  
   Represents a unit of logic or action to execute if its conditions are met.  
   **Example:** Print a message if the user is valid.

3. **`Evaluable`**  
   The building block of the rule tree, which can be evaluated.  
   Combines conditions and rules into logical structures.

4. **`Hooks`**
   Are customizable points in the validation process that allow custom logic to execute at key stages. Hooks include:

   - **`AfterPrepareConditions`**: Executes after preparing conditions for evaluation.
   - **`AfterEvaluateConditions`**: Executes after evaluating conditions and determining candidate rules.
   - **`AfterPrepareRules`**: Executes after preparing rules for evaluation.
   - **`AfterValidateRules`**: Executes after validating the prepared rules.
## 🔗 Logical Nodes

- **`Root`**: The starting point of your tree.
- **`Node`**: A single condition with child nodes or rules.
- **`And`**: All child nodes must evaluate to true.
- **`Or`**: At least one child node must evaluate to true.
- **`Rules`**: Contains the actions or validations to execute.


### The engine operates as a tree structure where:

1. **Leaf Nodes (Rules):**  
   These are the endpoints of the tree, containing the actual rules or actions to be executed.
   A rule can perform a specific validation or action when its conditions are satisfied.  
   **Example:** A rule might check if a user’s age is valid or log a message if a condition is met.

2. **Non-Leaf Nodes (Logic Operators and Conditions):**
   - **Conditions:** These are boolean checks that decide whether a particular branch of the tree should be followed.  
     **Example:** Check if a user is older than 30.
   - **Logic Operators (AND/OR):** These combine multiple conditions or nodes to form complex logical structures.
     - **AND Node:** All child nodes must evaluate to `true` for this node to succeed.
     - **OR Node:** At least one child node must evaluate to `true` for this node to succeed.


### Tree Structure Overview

The tree starts with a **Root Node**, which acts as the entry point for evaluation. Each branch of the tree can contain:
- **Nodes**: Represent conditions or logical operators (AND/OR).
- **Rules**: Contain the logic to be executed if the path through the tree is valid.

**Key Behavior:**  
If a condition in a node is not satisfied, the corresponding branch of the tree is skipped, and any rules under that branch
will not be evaluated. This ensures that only relevant rules are executed, based on the conditions provided.


---

## 🛠️ Building a Rule Tree

### Example: User Validation

```go
import (
	"context"
	"fmt"
	"github.com/mishudark/rules"
)

// Define User
type User struct {
	Age  int
	Name string
}

func main() {
	user := User{
		Age:  33,
		Name: "Bob",
	}

	// Build the rule tree
	tree := rules.Root(
		rules.Node(ageGt30(user.Age), rules.Rules(rule1())),
		rules.Node(nameEqBob(user.Name), rules.Rules(rule2())),
	)
}

// Define Conditions
func ageGt30(age int) rules.Condition {
	return rules.NewConditionPure("ageGt30", func() bool { return age > 30 })
}

func nameEqBob(name string) rules.Condition {
	return rules.NewConditionPure("nameEqBob", func() bool { return name == "Bob" })
}

// Define Rules
func rule1() rules.Rule {
	return rules.NewRulePure("rule1", func() error {
		fmt.Println("Rule 1: User is older than 30!")
		return nil
	})
}

func rule2() rules.Rule {
	return rules.NewRulePure("rule2", func() error {
		fmt.Println("Rule 2: User's name is Bob!")
		return nil
	})
}
```


## 🛠️ How to Run the validations

(Optional) You can add hooks by initializing a ProcessingHooks struct and assigning custom functions to the respective hooks.

### Example: Adding Logging Hooks
This example demonstrates how to use hooks to log specific stages of the validation process.

```go
package main

import (
	"context"
	"fmt"

	"github.com/mishudark/rules"
)

// Define User struct
type User struct {
	Name    string
	Age     int
	Address string
	Zipcode string
	Country string
}

func main() {
	// Example User
	user := User{
		Name:    "Alice",
		Age:     22,
		Zipcode: "12345",
		Country: "USA",
	}

	tree := rules.Root(
		rules.Node(
			nameNotEmpty(user.Name),
			rules.Rules(ruleValidZipCode(user.Zipcode)),
			rules.Node(
				countryIsUSA(user.Country),
				rules.Rules(
					ageGreaterThan20(user.Age),
				),
			),
		),
		rules.Node(
			rules.Not(
				nameNotEmpty(user.Name),
			),
			rules.Rules(ruleInvalidUser()),
		),
	)

	// Validate the tree
	ctx := context.Background()
	errors := rules.Validate(ctx, tree, rules.ProcessingHooks{}, "userValidationTree")

	// Check for validation errors
	if len(errors) > 0 {
		fmt.Printf("Validation errors: %v\n", errors)
	} else {
		fmt.Println("Validation completed successfully!")
	}
}

func nameNotEmpty(name string) rules.Condition {
	return rules.NewConditionPure("nameNotEmpty", func() bool {
		return name != ""
	})
}

func countryIsUSA(country string) rules.Condition {
	return rules.NewConditionPure("countryIsUSA", func() bool {
		return country == "USA"
	})
}

func ruleInvalidUser() rules.Rule {
	return rules.NewRulePure("ruleInvalidUser", func() error {
		return fmt.Errorf("User is invalid")
	})
}

func ruleValidZipCode(zipcode string) rules.Rule {
	return rules.NewRulePure("ruleInvalidZipCode", func() error {
		if len(zipcode) == 5 {
			return nil
		}

		return fmt.Errorf("Validation failed: ZIP code is invalid.")
	})
}

func ageGreaterThan20(age int) rules.Rule {
	return rules.NewRulePure("ageGreaterThan18", func() error {
		if age > 20 {
			return nil
		}

		return fmt.Errorf("age must be greater than 18")
	})
}


```

## Example Tree Explanation

This `tree` defines a hierarchical structure of rules for validating user data. The validation logic is implemented using nodes and rules. Below is an explanation of how this tree works:

## How the Tree Works

The tree processes the rules and conditions in a hierarchical manner, starting from the root node:

1. **First Node**:
   - If the user's name is not empty:
     - Validate the zip code.
     - Check if the user's country is the USA.
       - If the country is the USA, validate that the age is greater than 20.
   - If the user's name is empty, skip this node and move to the next.

2. **Second Node**:
   - If the user's name is empty:
     - Mark the user as invalid using `ruleInvalidUser()`.

---

## Example Flow

1. If the user's name is **not empty**:
   - Validate the zip code.
   - Check if the country is **USA**:
     - If true, validate that the age is greater than 20.

2. If the user's name **is empty**:
   - Mark the user as invalid.

---

## Key Features

- **Top-Down Evaluation**: The tree evaluates conditions and rules sequentially in a top-down manner.
- **Flexible Validation**: The hierarchical structure allows for modular validation logic.
- **Negation Support**: The `rules.Not` function enables handling cases where conditions should evaluate to `false`.

This structure provides a clear and modular way to define and validate complex user rules.
