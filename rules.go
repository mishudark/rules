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

// NopRule is useful for test or when operation doesn't need to performa rule
func NopRule() *Error {
	return nil
}

// When executes the provided rule only when the predicate returns true
func When(predicate Predicate, rule Rule) Rule {
	if !predicate() {
		return NopRule
	}

	return rule
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
