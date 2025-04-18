package rules

import (
	"testing"
)

func TestValidate(t *testing.T) {
	user := struct {
		age int
	}{
		age: 33,
	}

	tree := Or(
		Node(ageGt30(user.age), Rules(rule1(t))),
		Or(
			Node(ageGt1(user.age),
				Rules(rule2(t))),
			Node(ageLte30(user.age),
				Rules(rule3(t))),
		),
	)

	_, rules := tree.Evaluate()
	length := len(rules)

	for _, rule := range rules {
		rule()
	}

	if length != 2 {
		t.Errorf("expected 2 rule, got: %d", length)
	}
}

func rule1(t *testing.T) func() *Error {
	return func() *Error {

		t.Log("rule1")
		return nil
	}
}

func rule2(t *testing.T) func() *Error {
	return func() *Error {

		t.Log("rule2")
		return nil
	}
}

func rule3(t *testing.T) func() *Error {
	return func() *Error {

		t.Log("rule1")
		return nil
	}
}

func ageGt30(age int) Predicate {
	return func() bool {
		return age > 30
	}
}

func ageGt1(age int) Predicate {
	return func() bool {
		return age > 1
	}
}

func ageLte30(age int) Predicate {
	return func() bool {
		return age <= 30
	}
}
