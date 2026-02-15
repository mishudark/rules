package rules

import (
	"context"
	"testing"
)

// MockImpureCondition is a condition that tracks if it was prepared.
type MockImpureCondition struct {
	name     string
	prepared bool
	valid    bool
}

func (m *MockImpureCondition) Name() string {
	return m.name
}

func (m *MockImpureCondition) Prepare(ctx context.Context) error {
	m.prepared = true
	return nil
}

func (m *MockImpureCondition) IsValid(ctx context.Context) bool {
	return m.valid
}

func (m *MockImpureCondition) IsPure() bool {
	return false
}

// MockEvaluable tracks if PrepareConditions was called.
type MockEvaluable struct {
	prepared bool
}

func (m *MockEvaluable) PrepareConditions(ctx context.Context) error {
	m.prepared = true
	return nil
}

func (m *MockEvaluable) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	return true, nil
}

func TestPureConditionPruning(t *testing.T) {
	ctx := context.Background()

	// Case 1: Pure condition returns false. Children should NOT be prepared.
	t.Run("PureFalse_PrunesChildren", func(t *testing.T) {
		child := &MockEvaluable{}
		condition := NewCondition("pureFalse", func(ctx context.Context) bool { return false })

		node := Node(condition, child)
		err := node.PrepareConditions(ctx)
		if err != nil {
			t.Fatalf("PrepareConditions failed: %v", err)
		}

		if child.prepared {
			t.Error("Expected child NOT to be prepared when pure condition is false, but it was.")
		}
	})

	// Case 2: Pure condition returns true. Children SHOULD be prepared.
	t.Run("PureTrue_PreparesChildren", func(t *testing.T) {
		child := &MockEvaluable{}
		condition := NewCondition("pureTrue", func(ctx context.Context) bool { return true })

		node := Node(condition, child)
		err := node.PrepareConditions(ctx)
		if err != nil {
			t.Fatalf("PrepareConditions failed: %v", err)
		}

		if !child.prepared {
			t.Error("Expected child to be prepared when pure condition is true, but it wasn't.")
		}
	})

	// Case 3: Impure condition returns false. Children SHOULD be prepared (because we don't check validity during prepare for impure).
	// Note: The current logic for impure conditions is: Prepare condition -> Prepare children.
	// Validity check happens at Evaluate time.
	t.Run("ImpureFalse_PreparesChildren", func(t *testing.T) {
		child := &MockEvaluable{}
		condition := &MockImpureCondition{name: "impureFalse", valid: false}

		node := Node(condition, child)
		err := node.PrepareConditions(ctx)
		if err != nil {
			t.Fatalf("PrepareConditions failed: %v", err)
		}

		if !condition.prepared {
			t.Error("Expected impure condition to be prepared.")
		}
		if !child.prepared {
			t.Error("Expected child to be prepared when condition is impure (even if invalid), but it wasn't.")
		}
	})
}

func TestNotConditionIsPure(t *testing.T) {
	pure := NewCondition("pure", func(ctx context.Context) bool { return true })
	impure := &MockImpureCondition{name: "impure"}

	if !Not(pure).IsPure() {
		t.Error("Expected Not(pure) to be pure")
	}

	if Not(impure).IsPure() {
		t.Error("Expected Not(impure) to be impure")
	}
}

// Test to ensure we didn't break existing functionality
func TestExistingFunctionality(t *testing.T) {
	rule := NewRulePure("testRule", func() error {
		return nil
	})

	condition := NewCondition("true", func(ctx context.Context) bool { return true })
	node := Node(condition, Rules(rule))

	ctx := context.Background()
	err := node.PrepareConditions(ctx)
	if err != nil {
		t.Fatalf("PrepareConditions failed: %v", err)
	}

	ok, _ := node.Evaluate(ctx, "root")
	if !ok {
		t.Error("Expected evaluation to succeed")
	}
}
