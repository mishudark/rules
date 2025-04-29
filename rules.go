package rules

import (
	"context"
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
	// Prepare is executed before the main validation logic. It can be used to retrieve information.
	Prepare(ctx context.Context) error
	// GetName is a method to retrieve the name of the condition for debugging or logging.
	GetName() string
	// Evaluate returns true if the condition is met, otherwise false.
	IsValid(ctx context.Context) bool
}

// Rule represents a single unit of validation logic. It includes a Prepare
// step (potentially for setup or pre-checks) and a Validate step that performs
// the actual validation check. Both methods return an error if validation fails
// at that stage, or nil otherwise.
type Rule interface {
	// Name returns the name of the rule for identification.
	Name() string
	// Prepare allows for initialization or pre-checks before the main validation.
	// Returns an error if preparation fails, otherwise nil.
	Prepare(ctx context.Context) error
	// Validate performs the core validation logic.
	// Returns an error if validation fails, otherwise nil.
	Validate(ctx context.Context) error
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
	// PrepareConditions is executed before the main validation logic. It can be used to retrieve information.
	// specifically its target are the children condition nodes.
	PrepareConditions(ctx context.Context) error
	// Evaluate checks the conditions of the component and returns whether it
	// passes (bool) and the list of Rules associated with it if it passes.
	// If the conditions are not met, it returns false and a nil slice of Rules.
	Evaluate(ctx context.Context, executionPath string) (bool, []Rule)
}

// LeafNode represents a terminal node in the validation evaluation tree.
// It directly contains a slice of Rules that should be executed if this
// node is reached and evaluated successfully.
type LeafNode struct {
	Rules []Rule
}

// PrepareConditions is a no-op for LeafNode. It always returns nil.
func (r *LeafNode) PrepareConditions(ctx context.Context) error {
	// LeafNode does not have conditions to prepare.
	return nil
}

// Evaluate implements the Evaluable interface for LeafNode. It always
// returns true, indicating success, along with the slice of Rules contained
// within the node.
func (n *LeafNode) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {

	for _, rule := range n.Rules {
		// Set the execution path for each rule.
		rule.SetExecutionPath(fmt.Sprintf("%s -> %s -> %s", executionPath, "leafNode", rule.Name()))
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

// PrepareConditions prepares the ConditionNode by preparing its Condition.
func (n *ConditionNode) PrepareConditions(ctx context.Context) error {
	if n.Condition == nil {
		// Avoid nil pointer dereference if Condition func wasn't provided.
		return nil
	}

	return n.Condition.Prepare(ctx)
}

// Evaluate implements the Evaluable interface for ConditionNode. It first checks
// the Condition. If the Condition is nil or evaluates to false, Evaluate returns
// false and nil rules. If the Condition is true, it evaluates each child Evaluable,
// collecting and returning all Rules from children that evaluate successfully (return true).
func (n *ConditionNode) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	if n.Condition == nil || !n.Condition.IsValid(ctx) {
		return false, nil
	}

	matchRules := []Rule{}

	for _, evaluable := range n.Evaluables {
		ok, rules := evaluable.Evaluate(ctx, fmt.Sprintf("%s -> %s", executionPath, n.Condition.GetName()))
		if ok {
			matchRules = append(matchRules, rules...)
		}
	}

	// The ConditionNode itself succeeded because its condition was met.
	// It returns the aggregated rules from its successful children.
	return true, matchRules
}

var _ Evaluable = (*ConditionNode)(nil) // Ensure ConditionNode implements the Evaluable interface.

// AllOfNode represents a logical AND operation in the validation evaluation tree.
// All of its child Evaluables must evaluate successfully for the AllOfNode itself
// to be considered successful.
type AllOfNode struct {
	Children []Evaluable // The children that must all evaluate successfully.
}

// PrepareConditions is a no-op for AllOfNode.
func (n *AllOfNode) PrepareConditions(ctx context.Context) error {
	for _, child := range n.Children {
		err := child.PrepareConditions(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Evaluate implements the Evaluable interface for AllOfNode. It iterates through
// all its Children. If any child evaluates to false, the AllOfNode immediately
// returns false and nil rules. If all children evaluate to true, it returns true
// and the combined list of Rules gathered from all children. An empty AllOfNode
// is considered successful.
func (n *AllOfNode) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	acc := []Rule{}

	if len(n.Children) == 0 {
		return true, acc // An empty AND condition is trivially true.
	}

	for i := range n.Children {
		child := n.Children[i]
		ok, rules := child.Evaluate(ctx, fmt.Sprintf("%s -> %s", executionPath, "allOfNode"))
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

var _ Evaluable = (*AllOfNode)(nil) // Ensure AllOfNode implements the Evaluable interface.

// AnyOfNode represents a logical OR operation in the validation evaluation tree.
// At least one of its child Evaluables must evaluate successfully for the AnyOfNode
// itself to be considered successful.
type AnyOfNode struct {
	name     string      // Name of the AnyOfNode (optional) for identification or debugging.
	Children []Evaluable // The children, where at least one must evaluate successfully.
}

// PrepareConditions is a no-op for AnyOfNode.
func (n *AnyOfNode) PrepareConditions(ctx context.Context) error {
	for _, child := range n.Children {
		err := child.PrepareConditions(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Evaluate implements the Evaluable interface for AnyOfNode. It iterates through
// all its Children. If at least one child evaluates to true, the AnyOfNode returns
// true along with the combined list of Rules gathered from *all* successful
// children. If no children evaluate to true, it returns false and nil rules.
// An empty AnyOfNode is considered successful (or perhaps should be false, depending on desired logic - current impl returns true).
func (n *AnyOfNode) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	acc := []Rule{}

	if len(n.Children) == 0 {
		// Current implementation returns true, similar to AllOfNode.
		return true, acc
	}

	var anyOk bool

	nodeName := n.name
	if nodeName == "" {
		nodeName = "anyOfNode"
	}

	for i := range n.Children {
		child := n.Children[i]
		ok, rules := child.Evaluate(ctx, fmt.Sprintf("%s -> %s", executionPath, nodeName))
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

var _ Evaluable = (*AnyOfNode)(nil) // Ensure AnyOfNode implements the Evaluable interface.

// And is a constructor function that creates and returns a new AllOfNode
// containing the provided child Evaluables.
func AllOf(children ...Evaluable) Evaluable {
	return &AllOfNode{Children: children}
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

// Or is a constructor function that creates and returns a new AnyOfNode
// containing the provided child Evaluables.
func AnyOf(Children ...Evaluable) Evaluable {
	return &AnyOfNode{Children: Children}
}

// Root is a constructor function often used to define the top-level node of
// the validation evaluation tree. Currently, it creates an AnyOfNode, implying the
// root requires at least one of its top-level children to evaluate successfully.
func Root(Children ...Evaluable) Evaluable {
	// Note: Currently identical to AnyOf().
	return &AnyOfNode{Children: Children, name: "root"}
}

type NotCondition struct {
	condition Condition
}

func (n *NotCondition) GetName() string {
	return fmt.Sprintf("Not -> %s", n.condition.GetName())
}

func (n *NotCondition) Prepare(ctx context.Context) error {
	return n.condition.Prepare(ctx)
}

func (n *NotCondition) IsValid(ctx context.Context) bool {
	if n.condition == nil {
		// Avoid nil pointer dereference if Condition func wasn't provided.
		return false
	}

	return !n.condition.IsValid(ctx)
}

var _ Condition = (*NotCondition)(nil) // Ensure NotCondition implements the Condition interface.

// Not is a helper function that takes a Condition and returns a Conditiona with
// the logical negation of the Condition's result.
func Not(condition Condition) Condition {
	return &NotCondition{
		condition: condition,
	}
}

// NopRule is intended as a placeholder or no-operation function within validation logic.
// It returns nil, signifying success without performing any action. It can be useful
// in conditional logic where one branch requires no validation or during testing.
// Note: This function itself does not implement the Rule interface. It returns the
// error expected from Rule's Prepare or Validate methods.
func NopRule() error {
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
func (c *ChainRules) Prepare(ctx context.Context) error {
	for _, rule := range c.Rules {
		err := rule.Prepare(ctx)
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
func (c *ChainRules) Validate(ctx context.Context) error {
	for _, rule := range c.Rules {
		err := rule.Validate(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// RuleBase provides a basic implementation of the Rule execution path.
type RuleBase struct {
	executionPath string
}

// SetExecutionPath sets the execution path for the RuleBase.
func (r *RuleBase) SetExecutionPath(path string) {
	r.executionPath = path
}

// GetExecutionPath retrieves the execution path for the RuleBase.
func (r *RuleBase) GetExecutionPath() string {
	return r.executionPath
}

// RulePure provides a basic implementation of the Rule interface by wrapping
// a single function. This function represents the core validation logic.
// The Prepare method for a RulePure is a no-op.
type RulePure struct {
	RuleBase
	executionPath string
	name          string
	rule          func() error
}

var _ Rule = (*RulePure)(nil) // Ensure RulePure implements the Rule interface.

// Prepare implements the Rule interface for RulePure. It performs no action
// and always returns nil.
func (r *RulePure) Prepare(ctx context.Context) error {
	return nil // Simple rules typically don't require preparation.
}

// Name returns the name of the RulePure. This is useful for debugging.
func (r *RulePure) Name() string {
	return r.name
}

// Validate implements the Rule interface for RulePure. It executes the
// wrapped Rule function and returns its result (error or nil).
func (r *RulePure) Validate(ctx context.Context) error {
	if r.rule == nil {
		// Avoid nil pointer dereference if Rule func wasn't provided.
		// Consider returning an error here or handling it based on requirements.
		return nil // Or return fmt.Errorf("RulePure's Rule function is nil")?
	}

	return r.rule()
}

// NewRulePure is a constructor function that creates and returns a new
func NewRulePure(name string, rule func() error) Rule {
	return &RulePure{
		name: name,
		rule: rule,
	}
}

// ConditionPure does not need to be prepared and is used as a placeholder
type ConditionPure struct {
	name      string
	condition func() bool
}

var _ Condition = (*ConditionPure)(nil) // Ensure ConditionPure implements the Condition interface.

// Prepare is a no-op for ConditionPure, it always returns nil.
func (c *ConditionPure) Prepare(context.Context) error {
	return nil
}

func (c *ConditionPure) GetName() string {
	return c.name
}

func (c *ConditionPure) IsValid(ctx context.Context) bool {
	return c.condition()
}

// NewConditionPure  function that creates and returns a new ConditionPure.
func NewConditionPure(name string, condition func() bool) Condition {
	return &ConditionPure{
		name:      name,
		condition: condition,
	}
}
