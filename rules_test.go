package rules

import (
	"context"
	"errors"
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

type FailingRule struct {
	RuleBase
	name string
	err  error
}

func (f *FailingRule) Name() string                       { return f.name }
func (f *FailingRule) Prepare(ctx context.Context) error  { return f.err }
func (f *FailingRule) Validate(ctx context.Context) error { return f.err }

func TestChainRules(t *testing.T) {
	t.Parallel()

	errFail := errors.New("fail")

	t.Run("all succeed", func(t *testing.T) {
		t.Parallel()

		cr := NewChainRules(
			NewRulePure("r1", func() error { return nil }),
			NewRulePure("r2", func() error { return nil }),
		)

		ctx := context.Background()
		if err := cr.Prepare(ctx); err != nil {
			t.Errorf("unexpected prepare error: %v", err)
		}
		if err := cr.Validate(ctx); err != nil {
			t.Errorf("unexpected validate error: %v", err)
		}
	})

	t.Run("prepare stops at first error", func(t *testing.T) {
		t.Parallel()

		cr := NewChainRules(
			&FailingRule{name: "fail1", err: errFail},
			NewRulePure("r2", func() error { return nil }),
		)

		ctx := context.Background()
		if err := cr.Prepare(ctx); err == nil {
			t.Error("expected prepare error")
		}
	})

	t.Run("validate stops at first error", func(t *testing.T) {
		t.Parallel()

		cr := NewChainRules(
			&FailingRule{name: "fail1", err: errFail},
			NewRulePure("r2", func() error { return nil }),
		)

		ctx := context.Background()
		if err := cr.Validate(ctx); err == nil {
			t.Error("expected validate error")
		}
	})

	t.Run("empty rules", func(t *testing.T) {
		t.Parallel()

		cr := NewChainRules()
		ctx := context.Background()
		if err := cr.Prepare(ctx); err != nil {
			t.Errorf("unexpected prepare error: %v", err)
		}
		if err := cr.Validate(ctx); err != nil {
			t.Errorf("unexpected validate error: %v", err)
		}
	})
}

func TestConditionEither(t *testing.T) {
	t.Parallel()

	trueCond := NewConditionPure("true", func() bool { return true })
	falseCond := NewConditionPure("false", func() bool { return false })

	leftRule := NewRulePure("left", func() error { return nil })
	rightRule := NewRulePure("right", func() error { return nil })
	errRule := NewRulePure("err", func() error { return errors.New("should not run") })

	t.Run("condition true evaluates left", func(t *testing.T) {
		t.Parallel()

		either := Either(trueCond, []Evaluable{Rules(leftRule)}, []Evaluable{Rules(errRule)})
		ctx := context.Background()
		ok, rules := either.Evaluate(ctx, "test")
		if !ok {
			t.Error("expected evaluation to succeed")
		}
		if len(rules) != 1 || rules[0].Name() != "left" {
			t.Errorf("expected left rule, got %v", rules)
		}
	})

	t.Run("condition false evaluates right", func(t *testing.T) {
		t.Parallel()

		either := Either(falseCond, []Evaluable{Rules(errRule)}, []Evaluable{Rules(rightRule)})
		ctx := context.Background()
		ok, rules := either.Evaluate(ctx, "test")
		if !ok {
			t.Error("expected evaluation to succeed")
		}
		if len(rules) != 1 || rules[0].Name() != "right" {
			t.Errorf("expected right rule, got %v", rules)
		}
	})

	t.Run("nil condition evaluates right", func(t *testing.T) {
		t.Parallel()

		either := Either(nil, []Evaluable{Rules(errRule)}, []Evaluable{Rules(rightRule)})
		ctx := context.Background()
		ok, rules := either.Evaluate(ctx, "test")
		if !ok {
			t.Error("expected evaluation to succeed")
		}
		if len(rules) != 1 || rules[0].Name() != "right" {
			t.Errorf("expected right rule, got %v", rules)
		}
	})
}

func TestIsNil(t *testing.T) {
	t.Parallel()

	t.Run("nil data returns true", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(nil))
		if !IsNil("isNil").IsValid(ctx) {
			t.Error("expected IsNil to return true for nil data")
		}
	})

	t.Run("non-nil data returns false", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry("hello"))
		if IsNil("isNil").IsValid(ctx) {
			t.Error("expected IsNil to return false for non-nil data")
		}
	})

	t.Run("no registry returns true", func(t *testing.T) {
		ctx := context.Background()
		if !IsNil("isNil").IsValid(ctx) {
			t.Error("expected IsNil to return true when no registry")
		}
	})
}

func TestHasField(t *testing.T) {
	t.Parallel()

	type myStruct struct {
		Name string
		Age  int
	}

	t.Run("struct has field", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(myStruct{Name: "Alice", Age: 25}))
		if !HasField("hasName", "Name").IsValid(ctx) {
			t.Error("expected HasField to return true for existing field")
		}
	})

	t.Run("struct missing field", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(myStruct{Name: "Alice", Age: 25}))
		if HasField("hasFoo", "Foo").IsValid(ctx) {
			t.Error("expected HasField to return false for missing field")
		}
	})

	t.Run("map has key", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(map[string]int{"count": 5}))
		if !HasField("hasCount", "count").IsValid(ctx) {
			t.Error("expected HasField to return true for existing map key")
		}
	})

	t.Run("map missing key", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(map[string]int{"count": 5}))
		if HasField("hasTotal", "total").IsValid(ctx) {
			t.Error("expected HasField to return false for missing map key")
		}
	})

	t.Run("no data returns false", func(t *testing.T) {
		ctx := context.Background()
		if HasField("hasField", "Name").IsValid(ctx) {
			t.Error("expected HasField to return false without data")
		}
	})
}

func TestFieldEquals(t *testing.T) {
	t.Parallel()

	type myStruct struct {
		Name string
		Age  int
	}

	t.Run("struct field equals value", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(myStruct{Name: "Alice", Age: 25}))
		if !FieldEquals("nameEqAlice", "Name", "Alice").IsValid(ctx) {
			t.Error("expected FieldEquals to return true")
		}
	})

	t.Run("struct field not equal", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(myStruct{Name: "Alice", Age: 25}))
		if FieldEquals("nameEqBob", "Name", "Bob").IsValid(ctx) {
			t.Error("expected FieldEquals to return false")
		}
	})

	t.Run("missing field returns false", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(myStruct{Name: "Alice", Age: 25}))
		if FieldEquals("missing", "Foo", "bar").IsValid(ctx) {
			t.Error("expected FieldEquals to return false for missing field")
		}
	})

	t.Run("map key equals value", func(t *testing.T) {
		ctx := WithRegistry(context.Background(), NewDataRegistry(map[string]int{"count": 5}))
		if !FieldEquals("countEq5", "count", 5).IsValid(ctx) {
			t.Error("expected FieldEquals to return true for matching map key")
		}
	})
}

func TestNotCondition(t *testing.T) {
	t.Parallel()

	t.Run("nil condition returns safe defaults", func(t *testing.T) {
		nc := Not(nil)
		ctx := context.Background()

		if nc.Name() != "Not -> nil" {
			t.Errorf("expected 'Not -> nil', got %q", nc.Name())
		}

		if err := nc.Prepare(ctx); err != nil {
			t.Errorf("unexpected prepare error: %v", err)
		}

		if nc.IsValid(ctx) {
			t.Error("expected IsValid to return false for nil condition")
		}

		if nc.IsPure() {
			t.Error("expected IsPure to return false for nil condition")
		}
	})

	t.Run("negates true condition", func(t *testing.T) {
		trueCond := NewConditionPure("true", func() bool { return true })
		nc := Not(trueCond)
		ctx := context.Background()

		if nc.IsValid(ctx) {
			t.Error("expected Not(true) to return false")
		}
	})

	t.Run("negates false condition", func(t *testing.T) {
		falseCond := NewConditionPure("false", func() bool { return false })
		nc := Not(falseCond)
		ctx := context.Background()

		if !nc.IsValid(ctx) {
			t.Error("expected Not(false) to return true")
		}
	})
}

func TestRulePureNilFunc(t *testing.T) {
	t.Parallel()

	rule := NewRulePure("nilRule", nil)
	ctx := context.Background()

	err := rule.Validate(ctx)
	if err == nil {
		t.Error("expected error for nil rule function")
	}

	var e Error
	if errors.As(err, &e) {
		if e.Code != "RULE_FUNC_NIL" {
			t.Errorf("expected RULE_FUNC_NIL code, got %q", e.Code)
		}
	}
}

func TestOrRules(t *testing.T) {
	t.Parallel()

	errFail := errors.New("fail")

	tt := []struct {
		name    string
		rules   []Rule
		wantErr bool
	}{
		{
			name: "all fail",
			rules: []Rule{
				&FailingRule{name: "fail1", err: errFail},
				&FailingRule{name: "fail2", err: errFail},
			},
			wantErr: true,
		},
		{
			name: "one succeeds",
			rules: []Rule{
				&FailingRule{name: "fail1", err: errFail},
				NewRulePure("success1", func() error { return nil }),
			},
			wantErr: false,
		},
		{
			name: "first succeeds",
			rules: []Rule{
				NewRulePure("success1", func() error { return nil }),
				&FailingRule{name: "fail1", err: errFail},
			},
			wantErr: false,
		},
		{
			name: "all succeed",
			rules: []Rule{
				NewRulePure("success1", func() error { return nil }),
				NewRulePure("success2", func() error { return nil }),
			},
			wantErr: false,
		},
	}

	ctx := context.Background()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if len(tc.rules) == 0 {
				t.Fatal("test case must have at least one rule")
			}

			or := Or(tc.rules[0], tc.rules[1:]...)
			err := or.Validate(ctx)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
