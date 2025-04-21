package rules

import (
	"fmt"
)

// Error contains a structured definition for validation errors, including
// the field related to the error, a descriptive error message, and an
// optional error code for easier identification or localization.
type Error struct {
	Field string // Field indicates the specific input field or area where the error occurred.
	Err   string // Err provides a human-readable description of the error.
	Code  string // Code is an optional identifier for the type of error.
}

// Error implements the standard Go error interface, providing a formatted
// string representation of the validation error details.
func (e Error) Error() string {
	return fmt.Sprintf("code: %s, field: %s, error: %s", e.Code, e.Field, e.Err)
}

// Condition represents a function that evaluates to true or false, typically
// used within conditional nodes (like ConditionNode) to determine whether
// associated rules or child nodes should be processed.
type Condition interface {
	// GetName is a method to retrieve the name of the condition for debugging or logging.
	GetName() string
	// Evaluate returns true if the condition is met, otherwise false.
	IsValid() bool
}

type SimpleCondition struct {
	Name      string
	Condition func() bool
}

func (c *SimpleCondition) GetName() string {
	return c.Name
}

func (c *SimpleCondition) IsValid() bool {
	if c.Condition == nil {
		// Avoid nil pointer dereference if Condition func wasn't provided.
		return false
	}

	return c.Condition()
}

var _ Condition = (*SimpleCondition)(nil) // Ensure SimpleCondition implements the Condition interface.

// Rule represents a single unit of validation logic. It includes a Prepare
// step (potentially for setup or pre-checks) and a Validate step that performs
// the actual validation check. Both methods return an *Error if validation fails
// at that stage, or nil otherwise.
type Rule interface {
	// Prepare allows for initialization or pre-checks before the main validation.
	// Returns an *Error if preparation fails, otherwise nil.
	Prepare() *Error
	// Validate performs the core validation logic.
	// Returns an *Error if validation fails, otherwise nil.
	Validate() *Error
	// SetExecutionPath allows setting a path for execution context.
	SetExecutionPath(path string)
	// GetExecutionPath retrieves the execution path for the rule.
	GetExecutionPath() string
}

// Evaluable represents any component (like a node or a set of rules) within the
// validation structure that can be evaluated. The evaluation determines if the
// component's conditions are met (returning true) and provides the list of Rules
// that should be executed as a result.
type Evaluable interface {
	// Evaluate checks the conditions of the component and returns whether it
	// passes (bool) and the list of Rules associated with it if it passes.
	// If the conditions are not met, it returns false and a nil slice of Rules.
	Evaluate(executionPath string) (bool, []Rule)
}

// LeafNode represents a terminal node in the validation evaluation tree.
// It directly contains a slice of Rules that should be executed if this
// node is reached and evaluated successfully.
type LeafNode struct {
	Rules []Rule
}

// Evaluate implements the Evaluable interface for LeafNode. It always
// returns true, indicating success, along with the slice of Rules contained
// within the node.
func (n *LeafNode) Evaluate(executionPath string) (bool, []Rule) {

	for _, rule := range n.Rules {
		// Set the execution path for each rule.
		rule.SetExecutionPath(fmt.Sprintf("%s.%s", executionPath, "leafNode"))
	}

	return true, n.Rules
}

var _ Evaluable = (*LeafNode)(nil) // Ensure LeafNode implements the Evaluable interface.

// ConditionNode represents a node in the validation evaluation tree that has an
// associated Condition. If the Condition evaluates to true, the ConditionNode
// then evaluates its child Evaluables, accumulating the Rules from those children
// that also evaluate successfully.
type ConditionNode struct {
	Condition  Condition   // The condition that must be true for children to be evaluated.
	Evaluables []Evaluable // The child nodes or rule sets to evaluate if Condition is true.
}

// Evaluate implements the Evaluable interface for ConditionNode. It first checks
// the Condition. If the Condition is nil or evaluates to false, Evaluate returns
// false and nil rules. If the Condition is true, it evaluates each child Evaluable,
// collecting and returning all Rules from children that evaluate successfully (return true).
func (n *ConditionNode) Evaluate(executionPath string) (bool, []Rule) {
	if n.Condition == nil || !n.Condition.IsValid() {
		return false, nil
	}

	matchRules := []Rule{}

	for _, evaluable := range n.Evaluables {
		ok, rules := evaluable.Evaluate(fmt.Sprintf("%s.%s", executionPath, n.Condition.GetName()))
		if ok {
			matchRules = append(matchRules, rules...)
		}
	}

	// The ConditionNode itself succeeded because its condition was met.
	// It returns the aggregated rules from its successful children.
	return true, matchRules
}

var _ Evaluable = (*ConditionNode)(nil) // Ensure ConditionNode implements the Evaluable interface.

// AndNode represents a logical AND operation in the validation evaluation tree.
// All of its child Evaluables must evaluate successfully for the AndNode itself
// to be considered successful.
type AndNode struct {
	Children []Evaluable // The children that must all evaluate successfully.
}

// Evaluate implements the Evaluable interface for AndNode. It iterates through
// all its Children. If any child evaluates to false, the AndNode immediately
// returns false and nil rules. If all children evaluate to true, it returns true
// and the combined list of Rules gathered from all children. An empty AndNode
// is considered successful.
func (n *AndNode) Evaluate(executionPath string) (bool, []Rule) {
	acc := []Rule{}

	if len(n.Children) == 0 {
		return true, acc // An empty AND condition is trivially true.
	}

	for i := 0; i < len(n.Children); i++ {
		child := n.Children[i]
		ok, rules := child.Evaluate(fmt.Sprintf("%s.%s", executionPath, "andNode"))
		if ok {
			acc = append(acc, rules...)
		} else {
			// If any child fails, the AND condition fails.
			return false, nil
		}
	}

	// All children succeeded.
	return true, acc
}

var _ Evaluable = (*AndNode)(nil) // Ensure AndNode implements the Evaluable interface.

// OrNode represents a logical OR operation in the validation evaluation tree.
// At least one of its child Evaluables must evaluate successfully for the OrNode
// itself to be considered successful.
type OrNode struct {
	Children []Evaluable // The children, where at least one must evaluate successfully.
}

// Evaluate implements the Evaluable interface for OrNode. It iterates through
// all its Children. If at least one child evaluates to true, the OrNode returns
// true along with the combined list of Rules gathered from *all* successful
// children. If no children evaluate to true, it returns false and nil rules.
// An empty OrNode is considered successful (or perhaps should be false, depending on desired logic - current impl returns true).
func (n *OrNode) Evaluate(executionPath string) (bool, []Rule) {
	acc := []Rule{}

	if len(n.Children) == 0 {
		// Current implementation returns true, similar to AndNode.
		return true, acc
	}

	var anyOk bool

	for i := 0; i < len(n.Children); i++ {
		child := n.Children[i]
		ok, rules := child.Evaluate(fmt.Sprintf("%s.%s", executionPath, "orNode"))
		if ok {
			anyOk = true
			acc = append(acc, rules...) // Collect rules from all successful children.
		}
	}

	if !anyOk {
		// No child succeeded.
		return false, nil
	}

	// At least one child succeeded.
	return true, acc
}

var _ Evaluable = (*OrNode)(nil) // Ensure OrNode implements the Evaluable interface.

// And is a constructor function that creates and returns a new AndNode
// containing the provided child Evaluables.
func And(children ...Evaluable) Evaluable {
	return &AndNode{Children: children}
}

// Rules is a constructor function that creates and returns a new LeafNode
// containing the provided Rules. This is typically used to define the set
// of validations to run at the end of a branch in the evaluation tree.
func Rules(rules ...Rule) Evaluable {
	return &LeafNode{Rules: rules}
}

// Node is a constructor function that creates and returns a new ConditionNode.
// It associates a Condition with a set of child Evaluables.
func Node(condition Condition, children ...Evaluable) Evaluable {
	return &ConditionNode{
		Condition:  condition,
		Evaluables: children,
	}
}

// Or is a constructor function that creates and returns a new OrNode
// containing the provided child Evaluables.
func Or(Children ...Evaluable) Evaluable {
	return &OrNode{Children: Children}
}

// Root is a constructor function often used to define the top-level node of
// the validation evaluation tree. Currently, it creates an OrNode, implying the
// root requires at least one of its top-level children to evaluate successfully.
// Consider if an AndNode or a different structure might be more appropriate
// depending on the desired overall validation logic.
func Root(Children ...Evaluable) Evaluable {
	// Note: Currently identical to Or().
	return &OrNode{Children: Children}
}

// Not is a helper function that takes a Condition and returns a Conditiona with
// the logical negation of the Condition's result.
func Not(condition Condition) Condition {
	return &SimpleCondition{
		Name: fmt.Sprintf("Not.%s", condition.GetName()),
		Condition: func() bool {
			return !condition.IsValid()
		},
	}
}

// NopRule is intended as a placeholder or no-operation function within validation logic.
// It returns nil, signifying success without performing any action. It can be useful
// in conditional logic where one branch requires no validation or during testing.
// Note: This function itself does not implement the Rule interface. It returns the
// *Error expected from Rule's Prepare or Validate methods.
func NopRule() *Error {
	return nil
}

// ChainRules (assuming typo correction from ChinRules) represents a Rule that encapsulates
// a sequence of other Rules. When Prepare or Validate is called on ChainRules,
// it executes the corresponding method on each child Rule in order, stopping
// and returning the first encountered error. If all child rules succeed, it returns nil.
type ChainRules struct { // Corrected typo from ChinRules
	Rules []Rule
}

// Prepare implements the Rule interface for ChainRules. It calls Prepare() on each
// Rule in the sequence. If any child Rule's Prepare() returns an error,
// this method stops and returns that error immediately. If all children's
// Prepare() methods succeed, it returns nil.
func (c *ChainRules) Prepare() *Error {
	for _, rule := range c.Rules {
		err := rule.Prepare()
		if err != nil {
			return err
		}
	}
	return nil
}

// Validate implements the Rule interface for ChainRules. It calls Validate() on each
// Rule in the sequence. If any child Rule's Validate() returns an error,
// this method stops and returns that error immediately. If all children's
// Validate() methods succeed, it returns nil.
func (c *ChainRules) Validate() *Error {
	for _, rule := range c.Rules {
		err := rule.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

// Validate is a convenience function that executes the Validate() method for each
// provided Rule. Unlike ChainRules, it runs *all* rules regardless of individual
// failures and collects all non-nil errors returned into a slice.
// Returns a slice of errors encountered, which will be empty if all rules passed.
func Validate(rules ...Rule) (errors []error) {
	for _, rule := range rules {
		err := rule.Validate()
		if err != nil {
			errors = append(errors, err) // Note: Appends *Error which implements error
		}
	}
	return errors
}

// SimpleRule provides a basic implementation of the Rule interface by wrapping
// a single function. This function represents the core validation logic.
// The Prepare method for a SimpleRule is a no-op.
type SimpleRule struct {
	executionPath string // executionPath is a string representing the path of execution context.
	// Rule is the function containing the validation logic.
	// It should return an *Error if validation fails, or nil if it passes.
	Rule func() *Error
}

var _ Rule = (*SimpleRule)(nil) // Ensure SimpleRule implements the Rule interface.

// Prepare implements the Rule interface for SimpleRule. It performs no action
// and always returns nil.
func (r *SimpleRule) Prepare() *Error {
	return nil // Simple rules typically don't require preparation.
}

// Validate implements the Rule interface for SimpleRule. It executes the
// wrapped Rule function and returns its result (*Error or nil).
func (r *SimpleRule) Validate() *Error {
	if r.Rule == nil {
		// Avoid nil pointer dereference if Rule func wasn't provided.
		// Consider returning an error here or handling it based on requirements.
		return nil // Or return fmt.Errorf("SimpleRule's Rule function is nil")?
	}
	return r.Rule()
}

// SetExecutionPath sets the execution path for the SimpleRule.
func (r *SimpleRule) SetExecutionPath(path string) {
	r.executionPath = path
}

// GetExecutionPath retrieves the execution path for the SimpleRule.
func (r *SimpleRule) GetExecutionPath() string {
	return r.executionPath
}
