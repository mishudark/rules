package rules

import (
	"context"
	"strings"
	"testing"
)

// Regression test for fix #3: TypedRuleDataFunc.Validate's type-mismatch
// error must name the input type (In), not the loaded data type (T).
//
// Previously the error used `var zero T`, so the message labeled the
// expected type as the loaded-data type instead of the input type.
func TestTypedRuleDataFunc_ValidateTypeMismatch_ReportsInType(t *testing.T) {
	t.Parallel()

	type regInType struct{ Name string }
	type regTType struct{ Perms string }

	rule := NewTypedRuleWithPrepare[regInType, regTType](
		"checkPerms",
		func(ctx context.Context, in regInType) (regTType, error) {
			return regTType{Perms: "ok"}, nil
		},
		func(ctx context.Context, in regInType, data regTType) error {
			return nil
		},
	)

	// Prepare succeeds: input is regInType as expected.
	prepCtx := WithRegistry(context.Background(), NewDataRegistry(regInType{Name: "alice"}))
	if err := rule.Prepare(prepCtx); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}

	// Validate with a context holding a different type triggers the cast-fail
	// branch in Validate. The error must mention regInType, not regTType.
	validateCtx := WithRegistry(context.Background(), NewDataRegistry(regTType{Perms: "wrong"}))
	err := rule.Validate(validateCtx)
	if err == nil {
		t.Fatal("expected TYPE_MISMATCH error, got nil")
	}

	rErr, ok := err.(Error)
	if !ok {
		t.Fatalf("expected Error value, got %T: %v", err, err)
	}
	if rErr.Code != "TYPE_MISMATCH" {
		t.Fatalf("expected TYPE_MISMATCH, got %q (msg: %s)", rErr.Code, rErr.Err)
	}
	if !strings.Contains(rErr.Err, "expected input of type rules.regInType") {
		t.Errorf("error message %q should name the input type rules.regInType as the expected type", rErr.Err)
	}
	if strings.Contains(rErr.Err, "expected input of type rules.regTType") {
		t.Errorf("error message %q must not label the expected type as regTType (the loaded data type)", rErr.Err)
	}
}

// Regression test for fix #6: ConditionEither.Evaluate must return true
// whenever a branch has been selected, even if that branch contributed
// zero rules. Returning false in that case breaks composition inside an
// AllOfNode, which would discard its siblings' rules.
func TestConditionEither_TrueWhenBranchSelectedEvenWithNoRules(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	rule := NewRulePure("noop", func() error { return nil })

	// Left branch is selected (condition always true) but contributes no rules
	// (an empty Rules() leaf). Right branch has a rule but is not selected.
	either := Either(
		NewConditionPure("alwaysTrue", func() bool { return true }),
		[]Evaluable{Rules()}, // empty: contributes zero rules
		[]Evaluable{Rules(rule)},
	)

	if err := either.PrepareConditions(ctx); err != nil {
		t.Fatalf("PrepareConditions failed: %v", err)
	}

	ok, matched := either.Evaluate(ctx, "root")
	if !ok {
		t.Fatal("expected ConditionEither to return true when its selected branch has no rules, got false")
	}
	if len(matched) != 0 {
		t.Fatalf("expected zero matched rules, got %d", len(matched))
	}

	// Composition check: nested inside AllOf, the empty-but-selected Either
	// must NOT cause AllOf to short-circuit to false; the sibling's rules
	// must still be collected.
	other := Rules(rule)
	allOf := AllOf(either, other)
	ok, matched = allOf.Evaluate(ctx, "root")
	if !ok {
		t.Fatal("expected AllOf to pass when ConditionEither branch was selected (true) even though empty; got false")
	}
	if len(matched) != 1 {
		t.Fatalf("expected AllOf to collect 1 rule from the sibling, got %d", len(matched))
	}
}

// Regression test for the ConditionEither dataloader batching invariant:
//
// For impure conditions, PrepareConditions must prepare BOTH branches
// (regardless of validity) so the dataloader can batch and deduplicate
// data fetches across the whole tree in a single round-trip. Skipping
// the non-matching branch would serialize those fetches (N+1 problem).
//
// For pure conditions, only the matching branch is prepared (pure
// conditions have no side effects, so batching is moot).
func TestConditionEither_PrepareConditions_DataloaderBatching(t *testing.T) {
	t.Parallel()

	// Case A: impure-true condition -> both branches prepared.
	{
		leftChild := &MockEvaluable{}
		rightChild := &MockEvaluable{}
		impureTrue := &MockImpureCondition{name: "impureTrue", valid: true}

		either := Either(impureTrue, []Evaluable{leftChild}, []Evaluable{rightChild})
		if err := either.PrepareConditions(context.Background()); err != nil {
			t.Fatalf("PrepareConditions (true case) failed: %v", err)
		}
		if !impureTrue.prepared {
			t.Error("impure condition should have been prepared")
		}
		if !leftChild.prepared {
			t.Error("left branch (true case) should be prepared")
		}
		if !rightChild.prepared {
			t.Error("right branch (true case) should ALSO be prepared for dataloader batching")
		}
	}

	// Case B: impure-false condition -> both branches prepared.
	{
		leftChild := &MockEvaluable{}
		rightChild := &MockEvaluable{}
		impureFalse := &MockImpureCondition{name: "impureFalse", valid: false}

		either := Either(impureFalse, []Evaluable{leftChild}, []Evaluable{rightChild})
		if err := either.PrepareConditions(context.Background()); err != nil {
			t.Fatalf("PrepareConditions (false case) failed: %v", err)
		}
		if !impureFalse.prepared {
			t.Error("impure condition should have been prepared")
		}
		if !leftChild.prepared {
			t.Error("left branch (false case) should ALSO be prepared for dataloader batching")
		}
		if !rightChild.prepared {
			t.Error("right branch (false case) should be prepared")
		}
	}

	// Case C: pure-true condition -> only the matching (left) branch prepared.
	{
		leftChild := &MockEvaluable{}
		rightChild := &MockEvaluable{}
		pureTrue := NewCondition("pureTrue", func(ctx context.Context) bool { return true })

		either := Either(pureTrue, []Evaluable{leftChild}, []Evaluable{rightChild})
		if err := either.PrepareConditions(context.Background()); err != nil {
			t.Fatalf("PrepareConditions (pure-true) failed: %v", err)
		}
		if !leftChild.prepared {
			t.Error("left branch (pure true) should be prepared")
		}
		if rightChild.prepared {
			t.Error("right branch (pure true) should NOT be prepared: pure conditions short-circuit")
		}
	}

	// Case D: pure-false condition -> only the matching (right) branch prepared.
	{
		leftChild := &MockEvaluable{}
		rightChild := &MockEvaluable{}
		pureFalse := NewCondition("pureFalse", func(ctx context.Context) bool { return false })

		either := Either(pureFalse, []Evaluable{leftChild}, []Evaluable{rightChild})
		if err := either.PrepareConditions(context.Background()); err != nil {
			t.Fatalf("PrepareConditions (pure-false) failed: %v", err)
		}
		if leftChild.prepared {
			t.Error("left branch (pure false) should NOT be prepared: pure conditions short-circuit")
		}
		if !rightChild.prepared {
			t.Error("right branch (pure false) should be prepared")
		}
	}
}
