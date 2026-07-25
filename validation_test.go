package rules

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestValidate_HookError(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("hook failed")
	hooks := ProcessingHooks{
		AfterEvaluateConditions: func(ctx context.Context) error {
			return hookErr
		},
	}

	rule := NewRulePure("noop", func() error { return nil })
	tree := Root(Rules(rule))

	err := Validate(context.Background(), tree, hooks, "test")
	if err == nil {
		t.Fatal("expected error from hook")
	}
	if !errors.Is(err, hookErr) {
		t.Fatalf("expected hook error, got: %v", err)
	}
}

func TestValidate_HookOrder(t *testing.T) {
	t.Parallel()

	var order []string

	hooks := ProcessingHooks{
		AfterPrepareConditions: func(ctx context.Context) error {
			order = append(order, "afterPrepareConditions")
			return nil
		},
		AfterEvaluateConditions: func(ctx context.Context) error {
			order = append(order, "afterEvaluateConditions")
			return nil
		},
		AfterPrepareRules: func(ctx context.Context) error {
			order = append(order, "afterPrepareRules")
			return nil
		},
		AfterValidateRules: func(ctx context.Context) error {
			order = append(order, "afterValidateRules")
			return nil
		},
	}

	rule := NewRulePure("noop", func() error { return nil })
	tree := Root(Rules(rule))

	err := Validate(context.Background(), tree, hooks, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"afterPrepareConditions",
		"afterEvaluateConditions",
		"afterPrepareRules",
		"afterValidateRules",
	}

	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, order)
		}
	}
}

func TestValidate_HookWithCondition(t *testing.T) {
	t.Parallel()

	hookCalled := false

	// Condition that always passes
	condition := NewCondition("alwaysTrue", func(ctx context.Context) bool {
		return true
	})

	rule := NewRulePure("noop", func() error { return nil })

	tree := Root(Node(condition, Rules(rule)))
	hooks := ProcessingHooks{
		AfterEvaluateConditions: func(ctx context.Context) error {
			hookCalled = true
			return nil
		},
	}

	err := Validate(context.Background(), tree, hooks, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hookCalled {
		t.Fatal("expected hook to be called")
	}
}

func TestValidate_HookWithFailingCondition(t *testing.T) {
	t.Parallel()

	hookCalled := false

	// Condition that always fails - rules should not be reached
	condition := NewCondition("alwaysFalse", func(ctx context.Context) bool {
		return false
	})

	rule := NewRule("shouldNotRun", func(ctx context.Context, data any) error {
		return fmt.Errorf("rule should not have been evaluated")
	})

	tree := Root(Node(condition, Rules(rule)))
	hooks := ProcessingHooks{
		AfterEvaluateConditions: func(ctx context.Context) error {
			hookCalled = true
			return nil
		},
	}

	err := ValidateWithData(context.Background(), tree, hooks, "test", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Hook should still be called because it's run after condition evaluation,
	// not after rule selection. This is intentional - hooks may need to run
	// based on the evaluated conditions.
	if !hookCalled {
		t.Fatal("expected hook to be called even when condition fails")
	}
}

func TestValidate_NilHooks(t *testing.T) {
	t.Parallel()

	rule := NewRulePure("noop", func() error { return nil })
	tree := Root(Rules(rule))
	hooks := ProcessingHooks{}

	err := Validate(context.Background(), tree, hooks, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMulti_HookOrder(t *testing.T) {
	t.Parallel()

	var order []string

	hooks := ProcessingHooks{
		AfterPrepareConditions: func(ctx context.Context) error {
			order = append(order, "afterPrepareConditions")
			return nil
		},
		AfterEvaluateConditions: func(ctx context.Context) error {
			order = append(order, "afterEvaluateConditions")
			return nil
		},
		AfterPrepareRules: func(ctx context.Context) error {
			order = append(order, "afterPrepareRules")
			return nil
		},
		AfterValidateRules: func(ctx context.Context) error {
			order = append(order, "afterValidateRules")
			return nil
		},
	}

	rule := NewRulePure("noop", func() error { return nil })
	tree := Root(Rules(rule))

	targets := []Target{
		{tree: tree, ctx: context.Background()},
	}

	err := ValidateMulti(context.Background(), targets, hooks, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"afterPrepareConditions",
		"afterEvaluateConditions",
		"afterPrepareRules",
		"afterValidateRules",
	}

	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, order)
		}
	}
}

func TestValidateMulti_ErrorAccumulation(t *testing.T) {
	t.Parallel()

	errPrepare := errors.New("prepare error")
	errValidate := errors.New("validate error")

	t.Run("accumulates prepare and validate errors across targets", func(t *testing.T) {
		t.Parallel()

		prepareFails := &FailingRule{name: "prepareFail", err: errPrepare}
		validateFails := &FailingRule{name: "validateFail", err: errValidate}
		passes := NewRulePure("pass", func() error { return nil })

		targets := []Target{
			{tree: Rules(prepareFails), ctx: context.Background()},
			{tree: Rules(validateFails), ctx: context.Background()},
			{tree: Rules(passes), ctx: context.Background()},
		}

		err := ValidateMulti(context.Background(), targets, ProcessingHooks{}, "test")
		if err == nil {
			t.Fatal("expected errors from multi validation")
		}

		// Should contain both prepare and validate errors
		if !strings.Contains(err.Error(), "prepare error") {
			t.Error("expected prepare error in accumulated errors")
		}
		if !strings.Contains(err.Error(), "validate error") {
			t.Error("expected validate error in accumulated errors")
		}
	})

	t.Run("all targets pass returns nil", func(t *testing.T) {
		t.Parallel()

		passes := NewRulePure("pass", func() error { return nil })
		targets := []Target{
			{tree: Rules(passes), ctx: context.Background()},
			{tree: Rules(passes), ctx: context.Background()},
		}

		err := ValidateMulti(context.Background(), targets, ProcessingHooks{}, "test")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("cancelled context passes through to rules", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		rule := NewRule("checkCancelled", func(ctx context.Context, data any) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context cancelled: %w", err)
			}
			return nil
		})

		tree := Root(Rules(rule))
		err := ValidateWithData(ctx, tree, ProcessingHooks{}, "test", "data")
		if err == nil {
			t.Error("expected error from cancelled context")
		}
	})

	t.Run("active context passes through normally", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		rule := NewRulePure("noop", func() error { return nil })
		tree := Root(Rules(rule))

		err := Validate(ctx, tree, ProcessingHooks{}, "test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestValidateMulti_HookError(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("hook failed")
	hooks := ProcessingHooks{
		AfterPrepareRules: func(ctx context.Context) error {
			return hookErr
		},
	}

	rule := NewRulePure("noop", func() error { return nil })
	tree := Root(Rules(rule))

	targets := []Target{
		{tree: tree, ctx: context.Background()},
	}

	err := ValidateMulti(context.Background(), targets, hooks, "test")
	if err == nil {
		t.Fatal("expected error from hook")
	}
	if !errors.Is(err, hookErr) {
		t.Fatalf("expected hook error, got: %v", err)
	}
}
