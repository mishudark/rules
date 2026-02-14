package rules

import (
	"context"
	"fmt"
	"testing"
)

type testUser struct {
	Name  string
	Email string
	Age   int
}

type testProduct struct {
	ID    int
	Name  string
	Price float64
}

func TestDataRegistry_GetAndGetAs(t *testing.T) {
	t.Parallel()

	user := testUser{Name: "Alice", Email: "alice@example.com", Age: 25}

	// Test Get
	ctx := WithRegistry(context.Background(), NewDataRegistry(user))
	data, ok := Get(ctx)
	if !ok {
		t.Fatal("expected data to be found")
	}
	if data != user {
		t.Errorf("expected data to be user, got %v", data)
	}

	// Test GetAs
	typedUser, ok := GetAs[testUser](ctx)
	if !ok {
		t.Fatal("expected typed user to be found")
	}
	if typedUser.Name != user.Name {
		t.Errorf("expected name %q, got %q", user.Name, typedUser.Name)
	}
}

func TestDataRegistry_GetAs_WrongType(t *testing.T) {
	t.Parallel()

	user := testUser{Name: "Alice"}
	ctx := WithRegistry(context.Background(), NewDataRegistry(user))

	// Try to get as Product - should fail
	_, ok := GetAs[testProduct](ctx)
	if ok {
		t.Error("expected GetAs to return false for wrong type")
	}
}

func TestDataRegistry_NoData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, ok := Get(ctx)
	if ok {
		t.Error("expected Get to return false when no data in context")
	}

	_, ok = GetAs[testUser](ctx)
	if ok {
		t.Error("expected GetAs to return false when no data in context")
	}
}

func TestValidateWithData(t *testing.T) {
	t.Parallel()

	// Build reusable tree that works with any data
	tree := Node(
		IsA[testUser]("isUser"),
		Rules(
			NewTypedRule[testUser]("checkAge", func(ctx context.Context, user testUser) error {
				if user.Age < 18 {
					return fmt.Errorf("must be 18 or older")
				}
				return nil
			}),
		),
	)

	// Test with valid user
	user := testUser{Name: "Alice", Age: 25}
	err := ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", user)
	if err != nil {
		t.Errorf("expected no error for valid user, got: %v", err)
	}

	// Test with underage user
	underageUser := testUser{Name: "Bob", Age: 16}
	err = ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", underageUser)
	if err == nil {
		t.Error("expected error for underage user")
	}

	// Test with product (wrong type) - condition should fail, no rules executed
	product := testProduct{ID: 1, Name: "Widget", Price: 10.99}
	err = ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", product)
	if err != nil {
		t.Errorf("expected no error when type doesn't match (condition fails), got: %v", err)
	}
}

func TestIsA(t *testing.T) {
	t.Parallel()

	user := testUser{Name: "Alice"}
	product := testProduct{ID: 1}

	// Test with user data
	ctx := WithRegistry(context.Background(), NewDataRegistry(user))

	if !IsA[testUser]("isUser").IsValid(ctx) {
		t.Error("expected IsA[testUser] to return true for user data")
	}

	if IsA[testProduct]("isProduct").IsValid(ctx) {
		t.Error("expected IsA[testProduct] to return false for user data")
	}

	// Test with product data
	ctx = WithRegistry(context.Background(), NewDataRegistry(product))

	if IsA[testUser]("isUser").IsValid(ctx) {
		t.Error("expected IsA[testUser] to return false for product data")
	}

	if !IsA[testProduct]("isProduct").IsValid(ctx) {
		t.Error("expected IsA[testProduct] to return true for product data")
	}
}

func TestIsNotNil(t *testing.T) {
	t.Parallel()

	// Test with data
	ctx := WithRegistry(context.Background(), NewDataRegistry(testUser{Name: "Alice"}))
	if !IsNotNil("hasData").IsValid(ctx) {
		t.Error("expected IsNotNil to return true when data exists")
	}

	// Test with no registry
	ctx = context.Background()
	if IsNotNil("hasData").IsValid(ctx) {
		t.Error("expected IsNotNil to return false when no registry")
	}

	// Test with nil data
	ctx = WithRegistry(context.Background(), NewDataRegistry(nil))
	if IsNotNil("hasData").IsValid(ctx) {
		t.Error("expected IsNotNil to return false when data is nil")
	}
}

func TestNewCondition(t *testing.T) {
	t.Parallel()

	user := testUser{Age: 25}
	ctx := WithRegistry(context.Background(), NewDataRegistry(user))

	condition := NewCondition("isAdult", func(ctx context.Context) bool {
		u, ok := GetAs[testUser](ctx)
		if !ok {
			return false
		}
		return u.Age >= 18
	})

	if !condition.IsValid(ctx) {
		t.Error("expected isAdult to return true for 25-year-old")
	}

	// Test with underage user
	user2 := testUser{Age: 16}
	ctx = WithRegistry(context.Background(), NewDataRegistry(user2))

	if condition.IsValid(ctx) {
		t.Error("expected isAdult to return false for 16-year-old")
	}
}

func TestNewRule(t *testing.T) {
	t.Parallel()

	tree := Rules(
		NewRule("validateEmail", func(ctx context.Context, data any) error {
			user, ok := data.(testUser)
			if !ok {
				return fmt.Errorf("expected testUser, got %T", data)
			}
			if user.Email == "" {
				return fmt.Errorf("email is required")
			}
			return nil
		}),
	)

	// Test with valid email
	user := testUser{Name: "Alice", Email: "alice@example.com"}
	err := ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", user)
	if err != nil {
		t.Errorf("expected no error for user with email, got: %v", err)
	}

	// Test without email
	user2 := testUser{Name: "Bob", Email: ""}
	err = ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", user2)
	if err == nil {
		t.Error("expected error for user without email")
	}
}

func TestNewTypedRule(t *testing.T) {
	t.Parallel()

	tree := Rules(
		NewTypedRule[testUser]("checkName", func(ctx context.Context, user testUser) error {
			if len(user.Name) < 2 {
				return fmt.Errorf("name must be at least 2 characters")
			}
			return nil
		}),
	)

	// Test with valid name
	user := testUser{Name: "Alice"}
	err := ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", user)
	if err != nil {
		t.Errorf("expected no error for valid name, got: %v", err)
	}

	// Test with short name
	user2 := testUser{Name: "A"}
	err = ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", user2)
	if err == nil {
		t.Error("expected error for short name")
	}

	// Test with wrong type - should get type mismatch error
	product := testProduct{Name: "Widget"}
	err = ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", product)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestNewTypedCondition(t *testing.T) {
	t.Parallel()

	condition := NewTypedCondition[testUser]("isAdult", func(ctx context.Context, user testUser) bool {
		return user.Age >= 18
	})

	// Test with adult user
	user := testUser{Age: 25}
	ctx := WithRegistry(context.Background(), NewDataRegistry(user))
	if !condition.IsValid(ctx) {
		t.Error("expected isAdult to return true for 25-year-old")
	}

	// Test with underage user
	user2 := testUser{Age: 16}
	ctx = WithRegistry(context.Background(), NewDataRegistry(user2))
	if condition.IsValid(ctx) {
		t.Error("expected isAdult to return false for 16-year-old")
	}

	// Test with wrong type
	product := testProduct{ID: 1}
	ctx = WithRegistry(context.Background(), NewDataRegistry(product))
	if condition.IsValid(ctx) {
		t.Error("expected isAdult to return false for non-user type")
	}
}

func TestCrossPackageTreeComposition(t *testing.T) {
	t.Parallel()

	// Simulate trees built in different packages
	userTree := Node(
		IsA[testUser]("isUser"),
		Rules(
			NewTypedRule[testUser]("userRule", func(ctx context.Context, u testUser) error {
				if u.Email == "" {
					return fmt.Errorf("user email required")
				}
				return nil
			}),
		),
	)

	productTree := Node(
		IsA[testProduct]("isProduct"),
		Rules(
			NewTypedRule[testProduct]("productRule", func(ctx context.Context, p testProduct) error {
				if p.Price <= 0 {
					return fmt.Errorf("product price must be positive")
				}
				return nil
			}),
		),
	)

	// Merge trees using Root (AnyOf)
	mergedTree := Root(userTree, productTree)

	// Test with valid user
	user := testUser{Name: "Alice", Email: "alice@example.com", Age: 25}
	err := ValidateWithData(context.Background(), mergedTree, ProcessingHooks{}, "test", user)
	if err != nil {
		t.Errorf("expected no error for valid user, got: %v", err)
	}

	// Test with invalid user (should fail user's email check)
	user2 := testUser{Name: "Bob", Email: ""}
	err = ValidateWithData(context.Background(), mergedTree, ProcessingHooks{}, "test", user2)
	if err == nil {
		t.Error("expected error for user without email")
	}

	// Test with valid product
	product := testProduct{ID: 1, Name: "Widget", Price: 10.99}
	err = ValidateWithData(context.Background(), mergedTree, ProcessingHooks{}, "test", product)
	if err != nil {
		t.Errorf("expected no error for valid product, got: %v", err)
	}

	// Test with invalid product
	product2 := testProduct{ID: 2, Name: "Free", Price: 0}
	err = ValidateWithData(context.Background(), mergedTree, ProcessingHooks{}, "test", product2)
	if err == nil {
		t.Error("expected error for product with zero price")
	}
}

func TestValidateMultiWithData(t *testing.T) {
	t.Parallel()

	tree := Rules(
		NewTypedRule[testUser]("checkAge", func(ctx context.Context, u testUser) error {
			if u.Age < 18 {
				return fmt.Errorf("must be 18+")
			}
			return nil
		}),
	)

	targets := []TreeAndData{
		{Tree: tree, Data: testUser{Name: "Alice", Age: 25}},
		{Tree: tree, Data: testUser{Name: "Bob", Age: 16}},
	}

	err := ValidateMultiWithData(context.Background(), targets, ProcessingHooks{}, "test")
	if err == nil {
		t.Error("expected error for second user (underage)")
	}
}

func TestIsAssignableTo(t *testing.T) {
	t.Parallel()

	// Create an interface
	type Named interface {
		GetName() string
	}

	type hasName struct {
		Name string
	}

	h := hasName{Name: "Test"}
	ctx := WithRegistry(context.Background(), NewDataRegistry(h))

	// IsAssignableTo[Named] should work for structs implementing the interface
	// (Note: hasName doesn't actually implement Named, so this would return false)
	// This test mainly verifies the function doesn't panic
	_ = IsAssignableTo[testUser]("assignable").IsValid(ctx)
}

func TestNewTypedRuleWithPrepare(t *testing.T) {
	t.Parallel()

	prepareCalled := false
	validateCalled := false

	rule := NewTypedRuleWithPrepare[testUser](
		"checkUser",
		func(ctx context.Context, user testUser) error {
			prepareCalled = true
			if user.Email == "" {
				return fmt.Errorf("prepare: email required")
			}
			return nil
		},
		func(ctx context.Context, user testUser) error {
			validateCalled = true
			if user.Age < 18 {
				return fmt.Errorf("validate: must be 18+")
			}
			return nil
		},
	)

	// Test successful case
	user := testUser{Name: "Alice", Email: "alice@example.com", Age: 25}
	ctx := WithRegistry(context.Background(), NewDataRegistry(user))

	err := rule.Prepare(ctx)
	if err != nil {
		t.Errorf("unexpected prepare error: %v", err)
	}
	if !prepareCalled {
		t.Error("prepare should have been called")
	}

	err = rule.Validate(ctx)
	if err != nil {
		t.Errorf("unexpected validate error: %v", err)
	}
	if !validateCalled {
		t.Error("validate should have been called")
	}

	// Test prepare failure
	prepareCalled = false
	validateCalled = false
	user2 := testUser{Name: "Bob", Email: "", Age: 25}
	ctx = WithRegistry(context.Background(), NewDataRegistry(user2))

	err = rule.Prepare(ctx)
	if err == nil {
		t.Error("expected prepare error for empty email")
	}
	if !prepareCalled {
		t.Error("prepare should have been called")
	}

	// Test validate failure
	prepareCalled = false
	validateCalled = false
	user3 := testUser{Name: "Charlie", Email: "charlie@example.com", Age: 16}
	ctx = WithRegistry(context.Background(), NewDataRegistry(user3))

	err = rule.Prepare(ctx)
	if err != nil {
		t.Errorf("unexpected prepare error: %v", err)
	}

	err = rule.Validate(ctx)
	if err == nil {
		t.Error("expected validate error for underage user")
	}
	if !validateCalled {
		t.Error("validate should have been called")
	}

	// Test with nil validate function (uses default no-op)
	ruleNoValidate := NewTypedRuleWithPrepare[testUser](
		"prepareOnly",
		func(ctx context.Context, user testUser) error {
			return nil
		},
		nil,
	)

	ctx = WithRegistry(context.Background(), NewDataRegistry(user))
	err = ruleNoValidate.Validate(ctx)
	if err != nil {
		t.Errorf("unexpected error with nil validate: %v", err)
	}
}

func TestNewTypedRuleWithPrepare_TypeMismatch(t *testing.T) {
	t.Parallel()

	rule := NewTypedRuleWithPrepare[testUser](
		"checkUser",
		func(ctx context.Context, user testUser) error {
			return nil
		},
		func(ctx context.Context, user testUser) error {
			return nil
		},
	)

	// Pass wrong type
	product := testProduct{ID: 1}
	ctx := WithRegistry(context.Background(), NewDataRegistry(product))

	err := rule.Prepare(ctx)
	if err == nil {
		t.Error("expected error for type mismatch in prepare")
	}

	err = rule.Validate(ctx)
	if err == nil {
		t.Error("expected error for type mismatch in validate")
	}
}

func TestNewTypedRuleWithPrepare_NoData(t *testing.T) {
	t.Parallel()

	rule := NewTypedRuleWithPrepare[testUser](
		"checkUser",
		func(ctx context.Context, user testUser) error {
			return nil
		},
		func(ctx context.Context, user testUser) error {
			return nil
		},
	)

	// No data in context
	ctx := context.Background()

	err := rule.Prepare(ctx)
	if err == nil {
		t.Error("expected error when no data in context for prepare")
	}

	err = rule.Validate(ctx)
	if err == nil {
		t.Error("expected error when no data in context for validate")
	}
}
