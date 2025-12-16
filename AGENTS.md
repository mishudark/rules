# AGENT Guidelines for github.com/mishudark/rules

This document outlines essential information for agents working within this Go codebase.

## 1. Project Overview

This repository implements a flexible rule engine in Go, allowing for the creation and evaluation of complex validation logic through a tree-like structure of rules and conditions.

## 2. Essential Commands

The following commands are standard for Go projects and should be used:

- **Run Tests**: `go test ./...`
  - Runs all tests in the current module. Tests are parallelized using `t.Parallel()`.
- **Build Project**: `go build ./...`
  - Compiles the entire module.
- **Format Code**: `go fmt ./...`
  - Formats Go source code according to Go's standard style.
- **Vet Code (Linting)**: `go vet ./...`
  - Reports suspicious constructs, such as `printf` calls whose arguments do not match their format strings.

## 3. Code Organization and Structure

The core logic revolves around three main interfaces and their implementations:

-   **`Rule`**: Represents a single unit of validation logic. It includes `Prepare` (for setup) and `Validate` (for actual checking).
-   **`Condition`**: Represents a function that evaluates to `true` or `false`. Used within `ConditionNode` to determine if child nodes/rules should be processed.
-   **`Evaluable`**: An interface for components that can be evaluated, determining if conditions are met and providing a list of `Rule`s to execute.

These interfaces are implemented by various node types to build a tree structure:

-   **`LeafNode`**: A terminal node containing a slice of `Rule`s.
-   **`ConditionNode`**: A node with an associated `Condition`. If the condition is true, it evaluates its child `Evaluable`s.
-   **`AllOfNode`**: Represents a logical AND. All child `Evaluable`s must succeed.
-   **`AnyOfNode`**: Represents a logical OR. At least one child `Evaluable` must succeed.

Helper functions are provided for constructing the rule tree:

-   `AllOf(children ...Evaluable) Evaluable`
-   `Rules(rules ...Rule) Evaluable` (creates a `LeafNode`)
-   `Node(condition Condition, children ...Evaluable) Evaluable` (creates a `ConditionNode`)
-   `AnyOf(children ...Evaluable) Evaluable`
-   `Root(children ...Evaluable) Evaluable` (currently creates an `AnyOfNode`)
-   `Not(condition Condition) Condition`

Basic implementations for `Rule` and `Condition` are provided:

-   **`RulePure`**: A simple rule wrapping a function.
-   **`ConditionPure`**: A simple condition wrapping a boolean function.
-   **`ChainRules`**: Allows grouping and sequential execution of multiple rules.

Error handling is done using the `rules.Error` struct, which provides structured error details.

## 4. Naming Conventions and Style Patterns

-   **Go Standard**: Follows standard Go naming conventions (`CamelCase` for types and functions, `camelCase` for variables).
-   **Interfaces**: Named either with an `-er` suffix or to describe their capability (e.g., `Condition`, `Rule`, `Evaluable`).
-   **Exports**: Struct fields are exported (`Uppercase`) if they need to be accessed outside the `rules` package.

## 5. Testing Approach and Patterns

-   **Location**: Tests are co-located with source files (e.g., `rules_test.go` for `rules.go`).
-   **Parallel Execution**: Tests use `t.Parallel()` to enable parallel test execution.
-   **Table-Driven Tests**: `t.Run()` is used within test functions to define and execute multiple test cases.
-   **Context**: `context.Background()` is commonly used for test contexts.
-   **Execution Path Verification**: Tests often verify the `GetExecutionPath()` of rules, which is crucial for tracing and debugging rule evaluation flow.

## 6. Important Gotchas and Non-Obvious Patterns

-   **`Root` Function Behavior**: The `Root` function currently creates an `AnyOfNode`. This means that when using `Root`, at least one of its immediate child `Evaluable`s must evaluate successfully for the root to pass.
-   **Pure Conditions/Rules**: `ConditionPure` and `RulePure` rely on function closures. Be mindful of variable scope when defining these to avoid unexpected behavior, especially with external variables.
-   **`Prepare` Methods**: The `Prepare` methods on `Rule` and `Condition` interfaces indicate that some validation logic might require asynchronous setup or pre-checks before the main `Validate` or `IsValid` execution. Agents should ensure these are properly handled when extending or debugging rules.
-   **Execution Path Tracing**: The `SetExecutionPath` and `GetExecutionPath` methods on `Rule`s are important for understanding the flow of execution through the rule tree. Utilize `GetExecutionPath()` for debugging complex rule interactions.
