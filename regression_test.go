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

	rule := NewTypedRuleWithPrepare(
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
	if _, err := rule.Prepare(prepCtx); err != nil {
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

	ok, matched := either.Evaluate(ctx)
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
	ok, matched = allOf.Evaluate(ctx)
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

// Regression test: NewTypedRuleWithPrepare with a nil prepare function must
// still allow Validate to run (with the zero value of T). Previously Prepare
// returned nil without setting hasData, so Validate always failed with
// DATA_NOT_PREPARED.
func TestTypedRuleWithPrepare_NilPrepare_ValidateRuns(t *testing.T) {
	t.Parallel()

	type regIn struct{ Name string }
	type regT struct{ Perms string }

	called := false
	rule := NewTypedRuleWithPrepare(
		"nilPrepare",
		nil,
		func(ctx context.Context, in regIn, data regT) error {
			called = true
			return nil
		},
	)

	ctx := WithRegistry(context.Background(), NewDataRegistry(regIn{Name: "alice"}))
	ctx, _ = withPreparedStore(ctx)
	_, err := rule.Prepare(ctx)
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if err := rule.Validate(ctx); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !called {
		t.Fatal("expected validate function to be called")
	}
}

// Regression test: IsNil/IsNotNil must handle all nil-able kinds (maps,
// slices, etc.), not just pointers. Previously a nil map or slice in the
// registry was reported as "not nil".
func TestIsNil_AllNilableKinds(t *testing.T) {
	t.Parallel()

	var nilMap map[string]int
	var nilSlice []string
	var nilPtr *int
	notNilMap := map[string]int{}

	testCases := []struct {
		name    string
		data    any
		wantNil bool
	}{
		{name: "nil map", data: nilMap, wantNil: true},
		{name: "nil slice", data: nilSlice, wantNil: true},
		{name: "nil pointer", data: nilPtr, wantNil: true},
		{name: "untyped nil", data: nil, wantNil: true},
		{name: "non-nil map", data: notNilMap, wantNil: false},
		{name: "non-nil value", data: 42, wantNil: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := WithRegistry(context.Background(), NewDataRegistry(tc.data))
			if got := IsNil("isNil").IsValid(ctx); got != tc.wantNil {
				t.Errorf("IsNil = %v, want %v", got, tc.wantNil)
			}
			if got := IsNotNil("isNotNil").IsValid(ctx); got == tc.wantNil {
				t.Errorf("IsNotNil = %v, want %v", got, !tc.wantNil)
			}
		})
	}
}

// Regression test: HasField and FieldEquals must not panic when the data is
// a map with a non-string key type. Previously MapIndex was called with a
// string key on e.g. map[int]string, which panics.
func TestHasField_NonStringMapKey_NoPanic(t *testing.T) {
	t.Parallel()

	ctx := WithRegistry(context.Background(), NewDataRegistry(map[int]string{1: "one"}))

	if HasField("hasField", "name").IsValid(ctx) {
		t.Error("expected HasField to return false for a map with non-string keys")
	}
	if FieldEquals("fieldEquals", "name", "one").IsValid(ctx) {
		t.Error("expected FieldEquals to return false for a map with non-string keys")
	}
}

// Regression test: IsA and IsAssignableTo must work with interface types.
// Previously IsA[SomeInterface] never matched (nil target type) and
// IsAssignableTo[SomeInterface] panicked in AssignableTo.
func TestTypeCheckers_InterfaceTypes(t *testing.T) {
	t.Parallel()

	type stringer interface{ String() string }

	ctx := WithRegistry(context.Background(), NewDataRegistry(regStringer{}))

	if !IsA[stringer]("isStringer").IsValid(ctx) {
		t.Error("expected IsA[stringer] to match a value implementing the interface")
	}
	if IsA[int]("isInt").IsValid(ctx) {
		t.Error("expected IsA[int] to not match a struct value")
	}
	if !IsAssignableTo[stringer]("assignableToStringer").IsValid(ctx) {
		t.Error("expected IsAssignableTo[stringer] to match a value implementing the interface")
	}
}

// Regression test: a failing AfterValidateRules hook must not discard the
// accumulated validation errors. Previously the hook error replaced them.
func TestValidate_HookErrorPreservesValidationErrors(t *testing.T) {
	t.Parallel()

	validationErr := Error{Field: "f", Err: "boom", Code: "BOOM"}
	hookErr := Error{Field: "hook", Err: "hook failed", Code: "HOOK_FAIL"}

	tree := Root(Rules(NewRulePure("failing", func() error { return validationErr })))
	hooks := ProcessingHooks{
		AfterValidateRules: func(ctx context.Context) error { return hookErr },
	}

	err := Validate(context.Background(), tree, hooks, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "BOOM") {
		t.Errorf("expected joined error to contain validation error BOOM, got: %v", err)
	}
	if !strings.Contains(err.Error(), "HOOK_FAIL") {
		t.Errorf("expected joined error to contain hook error HOOK_FAIL, got: %v", err)
	}
}

// Regression test: OrRules.Prepare must prepare ALL child rules (the OR
// semantics apply only to Validate) and return the first child's error
// immediately, without preparing the remaining rules.
func TestOrRules_PrepareAllFailFast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prepareErr := Error{Field: "f", Err: "prepare failed", Code: "PREPARE_FAIL"}

	// Success path: every rule must be prepared.
	first := &mockPrepareRule{RuleBase: RuleBase{}, name: "first"}
	second := &mockPrepareRule{RuleBase: RuleBase{}, name: "second"}

	if _, err := Or(first, second).Prepare(ctx); err != nil {
		t.Fatalf("expected Prepare to succeed, got: %v", err)
	}
	if first.prepares != 1 || second.prepares != 1 {
		t.Errorf("expected both rules to be prepared once, got first=%d second=%d", first.prepares, second.prepares)
	}

	// Failure path: return the first error immediately, skip the rest.
	failing := &mockPrepareRule{RuleBase: RuleBase{}, name: "failing", err: prepareErr}
	skipped := &mockPrepareRule{RuleBase: RuleBase{}, name: "skipped"}

	_, err := Or(failing, skipped).Prepare(ctx)
	if err == nil {
		t.Fatal("expected Prepare to fail")
	}
	if !strings.Contains(err.Error(), "PREPARE_FAIL") {
		t.Errorf("expected the child's prepare error, got: %v", err)
	}
	if skipped.prepares != 0 {
		t.Errorf("expected rules after a failure to not be prepared, got %d calls", skipped.prepares)
	}
}

type regStringer struct{}

func (regStringer) String() string { return "regStringer" }

type mockPrepareRule struct {
	RuleBase
	name     string
	err      error
	prepares int
}

func (m *mockPrepareRule) Name() string { return m.name }
func (m *mockPrepareRule) Prepare(context.Context) (any, error) {
	m.prepares++
	return nil, m.err
}
func (m *mockPrepareRule) Validate(context.Context) error {
	return nil
}
