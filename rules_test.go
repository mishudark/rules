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
				Node(ageGt30(user.age), Rules(rule1())),
				Or(
					Node(ageGt1(user.age),
						Rules(rule2())),
					Node(ageLte30(user.age),
						Rules(rule3())),
				),
			),
			expect: 2,
			executionPaths: []string{
				"tree -> root -> ageGt30 -> leafNode -> rule1",
				"tree -> root -> orNode -> ageGt1 -> leafNode -> rule2",
			},
		},

		{
			name: "testWithAnd",
			tree: Root(
				And(
					Node(ageLte30(user.age), Rules(rule1())),
					Node(nameEqBob(user.name), Rules(rule2())),
				),
			),
			expect:         0,
			executionPaths: []string{},
		},

		{
			name: "test with Or node",
			tree: Root(
				Or(
					Node(ageLte30(user.age), Rules(rule1())),
					Node(nameEqBob(user.name), Rules(rule2())),
				),
			),
			expect:         1,
			executionPaths: []string{"tree -> root -> orNode -> nameEqBob -> leafNode -> rule2"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, rules := tc.tree.Evaluate("tree")
			length := len(rules)

			if length != tc.expect {
				t.Errorf("expected: %d  rule, got: %d", tc.expect, length)
			}

			if len(rules) != len(tc.executionPaths) {
				t.Errorf("expected: %d \ngot: %d", len(tc.executionPaths), len(rules))

				return
			}

			for i, rule := range rules {
				if rule.GetExecutionPath() != tc.executionPaths[i] {
					t.Errorf("index: %d\nexpected: %s\ngot: %s", i, tc.executionPaths[i], rule.GetExecutionPath())
				}
			}
		})
	}
}

func rule1() Rule {
	return NewSimpleRule("rule1",
		func() *Error {
			return nil
		},
	)
}

func rule2() Rule {
	return NewSimpleRule("rule2",
		func() *Error {
			return nil
		},
	)
}

func rule3() Rule {
	return NewSimpleRule("rule3",
		func() *Error {
			return nil
		},
	)
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
