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

// Predicate is used with When statement to determine if the next rule should be executed
type Predicate func() bool

// Rule represents a validation that can returns either an error or a nil value
type Rule func() *Error

// Evaluate contains the predicate and rules to execute if predicate is true
type Evaluable interface {
	Evaluate() (bool, []Rule)
}

// PredicateNode is the leaf node that contains the set of rules to be executed if Predicate is met
type PredicateNode struct {
	Predicate Predicate
	Rules     []Rule
}

func (n *PredicateNode) Evaluate() (bool, []Rule) {
	if n.Predicate == nil || !n.Predicate() {
		return false, nil
	}

	return true, n.Rules
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

type NotNode struct {
	Child Evaluable
}

func (n *NotNode) Evaluate() (bool, []Rule) {
	if n.Child == nil {
		return true, []Rule{}
	}

	ok, rules := n.Child.Evaluate()
	return !ok, rules
}

func And(Children ...Evaluable) Evaluable {
	return &AndNode{Children: Children}
}

func P(cond Predicate, rules ...Rule) Evaluable {
	return &PredicateNode{Predicate: cond, Rules: rules}
}

// Or creates an OrNode.
func Or(Children ...Evaluable) Evaluable {
	return &OrNode{Children: Children}
}

// Not creates a NotNode.
func Not(child Evaluable) Evaluable {
	return &NotNode{Child: child}
}

// NopRule is useful for test or when operation doesn't need to performa rule
func NopRule() *Error {
	return nil
}

// Chain of rules that executes one rule after other until it finds an error
func Chain(rules ...Rule) Rule {
	return func() *Error {
		for _, rule := range rules {
			err := rule()
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// Validate executes the provided rules in order and returns a set of errors
func Validate(rules ...Rule) (errors []error) {
	for _, rule := range rules {
		err := rule()
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
