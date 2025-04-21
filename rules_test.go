package rules

import (
	"testing"
)

func TestValidateTree(t *testing.T) {
	t.Parallel()

	user := struct {
		age  int
		name string
	}{
		age:  33,
		name: "Bob",
	}

	tt := [...]struct {
		name           string
		tree           Evaluable
		expect         int
		executionPaths []string
	}{
		{
			name: "test with Or and And nodes",
			tree: Root(
				Node(ageGt30(user.age), Rules(rule1(t))),
				Or(
					Node(ageGt1(user.age),
						Rules(rule2(t))),
					Node(ageLte30(user.age),
						Rules(rule3(t))),
				),
			),
			expect: 2,
			executionPaths: []string{
				"root -> orNode -> ageGt30 -> leafNode",
				"root -> orNode -> orNode -> ageGt1 -> leafNode",
			},
		},

		{
			name: "testWithAnd",
			tree: Root(
				And(
					Node(ageLte30(user.age), Rules(rule1(t))),
					Node(nameEqBob(user.name), Rules(rule2(t))),
				),
			),
			expect:         0,
			executionPaths: []string{},
		},

		{
			name: "test with Or node",
			tree: Root(
				Or(
					Node(ageLte30(user.age), Rules(rule1(t))),
					Node(nameEqBob(user.name), Rules(rule2(t))),
				),
			),
			expect:         1,
			executionPaths: []string{"root -> orNode -> orNode -> nameEqBob -> leafNode"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, rules := tc.tree.Evaluate("root")
			length := len(rules)

			if length != tc.expect {
				t.Errorf("expected %d  rule, got: %d", tc.expect, length)
			}

			if len(rules) != len(tc.executionPaths) {
				t.Errorf("expected %d execution paths, got: %d", len(tc.executionPaths), len(rules))

				return
			}

			for i, rule := range rules {
				if rule.GetExecutionPath() != tc.executionPaths[i] {
					t.Errorf("index: %d, expected %s, got: %s", i, tc.executionPaths[i], rule.GetExecutionPath())
				}
			}
		})
	}
}

func rule1(t *testing.T) Rule {
	return &SimpleRule{
		Rule: func() *Error {
			t.Log("rule1")
			return nil
		},
	}
}

func rule2(t *testing.T) Rule {
	return &SimpleRule{
		Rule: func() *Error {

			t.Log("rule2")
			return nil
		},
	}
}

func rule3(t *testing.T) Rule {
	return &SimpleRule{
		Rule: func() *Error {
			t.Log("rule1")
			return nil
		},
	}
}

func ageGt30(age int) Condition {
	return &SimpleCondition{
		Name: "ageGt30",
		Condition: func() bool {
			return age > 30
		},
	}
}

func ageGt1(age int) Condition {
	return &SimpleCondition{
		Name: "ageGt1",
		Condition: func() bool {
			return age > 1
		},
	}
}

func ageLte30(age int) Condition {
	return &SimpleCondition{
		Name: "ageLte30",
		Condition: func() bool {
			return age <= 30
		},
	}
}

func nameEqBob(name string) Condition {
	return &SimpleCondition{
		Name: "nameEqBob",
		Condition: func() bool {
			return name == "Bob"
		},
	}
}
