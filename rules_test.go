package rules

import (
	"context"
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
			name: "test with AnyOf and And nodes",
			tree: Root(
				Node(ageGt30(user.age), Rules(rule1())),
				AnyOf(
					Node(ageGt1(user.age),
						Rules(rule2())),
					Node(ageLte30(user.age),
						Rules(rule3())),
				),
			),
			expect: 2,
			executionPaths: []string{
				"tree -> root -> ageGt30 -> leafNode -> rule1",
				"tree -> root -> anyOfNode -> ageGt1 -> leafNode -> rule2",
			},
		},

		{
			name: "testWithAnd",
			tree: Root(
				AllOf(
					Node(ageLte30(user.age), Rules(rule1())),
					Node(nameEqBob(user.name), Rules(rule2())),
				),
			),
			expect:         0,
			executionPaths: []string{},
		},

		{
			name: "test with AnyOf node",
			tree: Root(
				AnyOf(
					Node(ageLte30(user.age), Rules(rule1())),
					Node(nameEqBob(user.name), Rules(rule2())),
				),
			),
			expect:         1,
			executionPaths: []string{"tree -> root -> anyOfNode -> nameEqBob -> leafNode -> rule2"},
		},
	}

	ctx := context.Background()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, rules := tc.tree.Evaluate(ctx, "tree")
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
	return NewRulePure("rule1",
		func() error {
			return nil
		},
	)
}

func rule2() Rule {
	return NewRulePure("rule2",
		func() error {
			return nil
		},
	)
}

func rule3() Rule {
	return NewRulePure("rule3",
		func() error {
			return nil
		},
	)
}

func ageGt30(age int) Condition {
	return NewConditionPure(
		"ageGt30",
		func() bool {
			return age > 30
		},
	)
}

func ageGt1(age int) Condition {
	return NewConditionPure(
		"ageGt1",
		func() bool {
			return age > 1
		},
	)
}

func ageLte30(age int) Condition {
	return NewConditionPure(
		"ageLte30",
		func() bool {
			return age <= 30
		},
	)
}

func nameEqBob(name string) Condition {
	return NewConditionPure(
		"nameEqBob",
		func() bool {
			return name == "Bob"
		},
	)
}
