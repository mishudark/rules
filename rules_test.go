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

	tree := And(P(ageGt30(user.age),
		func() *Error {
			return nil
		}))

	_, rules := tree.Evaluate()
	length := len(rules)

	if length != 1 {
		t.Errorf("expected 1 rule, got: %d", length)
	}
}

func ageGt30(age int) Predicate {
	return func() bool {
		return age > 30
	}
}

func ageLte30(age int) Predicate {
	return func() bool {
		return age <= 30
	}
}
