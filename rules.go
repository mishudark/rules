package rules

import "fmt"

// Error contains a definition with Code, Field and Error to get a better reference
type Error struct {
	Field string
	Err   string
	Code  string
}

func (e Error) Error() string {
	return fmt.Sprintf("code: %s, field: %s, error: %s", e.Code, e.Field, e.Err)
}

// Condition is used with When statement to determine if the next rule should be executed
type Condition func() bool

// Rule represents a validation that can returns either an error or a nil value
type Rule interface {
	Prepare() *Error
	Validate() *Error
}

// Evaluate contains the predicate and rules to execute if predicate is true
type Evaluable interface {
	Evaluate() (bool, []Rule)
}

// LeafNode is the leaf node that contains the set of rules to be executed
type LeafNode struct {
	Rules []Rule
}

func (n *LeafNode) Evaluate() (bool, []Rule) {
	return true, n.Rules
}

// ConditionNode is a node with a condition that must be valid
type ConditionNode struct {
	Condition  Condition
	Evaluables []Evaluable
}

func (n *ConditionNode) Evaluate() (bool, []Rule) {
	if n.Condition == nil || !n.Condition() {
		return false, nil
	}

	matchRules := []Rule{}

	for _, evaluable := range n.Evaluables {
		ok, rules := evaluable.Evaluate()
		if ok {
			matchRules = append(matchRules, rules...)
		}
	}

	return true, matchRules
}

type AndNode struct {
	Children []Evaluable
}

func (n *AndNode) Evaluate() (bool, []Rule) {

	acc := []Rule{}

	if len(n.Children) == 0 {
		return true, acc
	}

	for i := 0; i < len(n.Children); i++ {
		child := n.Children[i]
		ok, rules := child.Evaluate()
		if ok {
			acc = append(acc, rules...)
		} else {
			return false, nil
		}
	}

	return true, acc
}

type OrNode struct {
	Children []Evaluable
}

func (n *OrNode) Evaluate() (bool, []Rule) {
	acc := []Rule{}

	if len(n.Children) == 0 {
		return true, acc
	}

	var anyOk bool

	for i := 0; i < len(n.Children); i++ {
		child := n.Children[i]
		ok, rules := child.Evaluate()
		if ok {
			anyOk = true
			acc = append(acc, rules...)
		}
	}

	if !anyOk {
		return false, nil
	}

	return true, acc
}

func And(children ...Evaluable) Evaluable {
	return &AndNode{Children: children}
}

func Rules(rules ...Rule) Evaluable {
	return &LeafNode{Rules: rules}
}

func Node(condition Condition, children ...Evaluable) Evaluable {
	return &ConditionNode{
		Condition:  condition,
		Evaluables: children,
	}
}

// Or creates an OrNode.
func Or(Children ...Evaluable) Evaluable {
	return &OrNode{Children: Children}
}

// Root creates an OrNode.
func Root(Children ...Evaluable) Evaluable {
	return &OrNode{Children: Children}
}

func Not(condition Condition) bool {
	return !condition()
}

// NopRule is useful for test or when operation doesn't need to performa rule
func NopRule() *Error {
	return nil
}

type ChinRules struct {
	Rules []Rule
}

func (c *ChinRules) Prepare() *Error {
	for _, rule := range c.Rules {
		if err := rule.Prepare(); err != nil {
			return err
		}
	}

	return nil
}

func (c *ChinRules) Validate() *Error {
	for _, rule := range c.Rules {
		err := rule.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate executes the provided rules in order and returns a set of errors
func Validate(rules ...Rule) (errors []error) {
	for _, rule := range rules {
		err := rule.Validate()
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

type SimpleRule struct {
	Rule func() *Error
}

func (r *SimpleRule) Prepare() *Error {
	return nil
}

func (r *SimpleRule) Validate() *Error {
	return r.Rule()
}
