package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestMinLengthString(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		testName   string
		value      string
		min        int
		expectFail bool
		errCode    string
	}{
		{
			testName:   "Valid length",
			value:      "hello",
			min:        5,
			expectFail: false,
		},
		{
			testName:   "Valid length (unicode)",
			value:      "héllo",
			min:        5,
			expectFail: false,
		},
		{
			testName:   "Invalid length",
			value:      "hi",
			min:        5,
			expectFail: true,
			errCode:    "MIN_LENGTH_STRING",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			rule := MinLengthString(tc.testName, tc.value, tc.min)
			err := rule.Validate(ctx)

			if tc.expectFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it succeeded.")
				} else if rulesErr, ok := err.(rules.Error); ok {
					if rulesErr.Code != tc.errCode {
						t.Errorf("Expected error code '%s' but got '%s'", tc.errCode, rulesErr.Code)
					}
				}
			} else if err != nil {
				t.Errorf("Expected validation to succeed, but it failed: %v", err)
			}
		})
	}
}

func TestMaxLengthString(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		testName   string
		value      string
		max        int
		expectFail bool
		errCode    string
	}{
		{
			testName:   "Valid length",
			value:      "hello",
			max:        5,
			expectFail: false,
		},
		{
			testName:   "Valid length (unicode)",
			value:      "héllo",
			max:        5,
			expectFail: false,
		},
		{
			testName:   "Invalid length",
			value:      "hello world",
			max:        5,
			expectFail: true,
			errCode:    "MAX_LENGTH_STRING",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			rule := MaxLengthString(tc.testName, tc.value, tc.max)
			err := rule.Validate(ctx)

			if tc.expectFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it succeeded.")
				} else if rulesErr, ok := err.(rules.Error); ok {
					if rulesErr.Code != tc.errCode {
						t.Errorf("Expected error code '%s' but got '%s'", tc.errCode, rulesErr.Code)
					}
				} else {
					t.Errorf("Expected rules.Error value, got %T: %v", err, err)
				}
			} else if err != nil {
				t.Errorf("Expected validation to succeed, but it failed: %v", err)
			}
		})
	}
}

func TestMinLengthSlice(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		testName   string
		value      []any
		min        int
		expectFail bool
		errCode    string
	}{
		{
			testName:   "Valid length",
			value:      []any{1, 2, 3},
			min:        3,
			expectFail: false,
		},
		{
			testName:   "Invalid length",
			value:      []any{1},
			min:        3,
			expectFail: true,
			errCode:    "MIN_LENGTH_SLICE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			rule := MinLengthSlice(tc.testName, tc.value, tc.min)
			err := rule.Validate(ctx)

			if tc.expectFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it succeeded.")
				} else if rulesErr, ok := err.(rules.Error); ok {
					if rulesErr.Code != tc.errCode {
						t.Errorf("Expected error code '%s' but got '%s'", tc.errCode, rulesErr.Code)
					}
				}
			} else if err != nil {
				t.Errorf("Expected validation to succeed, but it failed: %v", err)
			}
		})
	}
}

func TestMaxLengthSlice(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		testName   string
		value      []any
		max        int
		expectFail bool
		errCode    string
	}{
		{
			testName:   "Valid length",
			value:      []any{1, 2, 3},
			max:        3,
			expectFail: false,
		},
		{
			testName:   "Invalid length",
			value:      []any{1, 2, 3, 4},
			max:        3,
			expectFail: true,
			errCode:    "MAX_LENGTH_SLICE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			rule := MaxLengthSlice(tc.testName, tc.value, tc.max)
			err := rule.Validate(ctx)

			if tc.expectFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it succeeded.")
				} else if rulesErr, ok := err.(rules.Error); ok {
					if rulesErr.Code != tc.errCode {
						t.Errorf("Expected error code '%s' but got '%s'", tc.errCode, rulesErr.Code)
					}
				} else {
					t.Errorf("Expected rules.Error value, got %T: %v", err, err)
				}
			} else if err != nil {
				t.Errorf("Expected validation to succeed, but it failed: %v", err)
			}
		})
	}
}

// Regression test: error message must report the rune count (grapheme clusters
// approximated by runes), not the byte length, for Unicode correctness.
func TestMinLengthString_UnicodeErrorReportsRuneCount(t *testing.T) {
	ctx := context.Background()

	// "héllo" is 5 runes but 6 bytes (é is 2 bytes in UTF-8).
	// Require min 10 to force a failure and inspect the reported count.
	const value = "héllo"
	rule := MinLengthString("field", value, 10)
	err := rule.Validate(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rErr, ok := err.(rules.Error)
	if !ok {
		t.Fatalf("expected rules.Error value, got %T: %v", err, err)
	}
	if rErr.Code != "MIN_LENGTH_STRING" {
		t.Fatalf("unexpected error code: %s", rErr.Code)
	}

	want := "expected minimum 10, got 5"
	if rErr.Err != want {
		t.Errorf("error message = %q, want %q", rErr.Err, want)
	}
}

// Regression test: see TestMinLengthString_UnicodeErrorReportsRuneCount.
func TestMaxLengthString_UnicodeErrorReportsRuneCount(t *testing.T) {
	ctx := context.Background()

	// "héllo!!" is 7 runes but 9 bytes (é is 2 bytes).
	// Allow max 3 to force a failure.
	const value = "héllo!!"
	rule := MaxLengthString("field", value, 3)
	err := rule.Validate(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rErr, ok := err.(rules.Error)
	if !ok {
		t.Fatalf("expected rules.Error value, got %T: %v", err, err)
	}
	if rErr.Code != "MAX_LENGTH_STRING" {
		t.Fatalf("unexpected error code: %s", rErr.Code)
	}

	want := "expected maximum 3, got 7"
	if rErr.Err != want {
		t.Errorf("error message = %q, want %q", rErr.Err, want)
	}
}

// Regression test: validators must return a rules.Error *value*, not a pointer,
// so that callers can assert `err.(rules.Error)` and inspect the Code field.
func TestLengthValidators_ReturnErrorValueNotPointer(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		rule rules.Rule
		code string
	}{
		{"MinLengthString", MinLengthString("f", "x", 5), "MIN_LENGTH_STRING"},
		{"MaxLengthString", MaxLengthString("f", "xxxxxx", 5), "MAX_LENGTH_STRING"},
		{"MinLengthSlice", MinLengthSlice("f", []any{1}, 3), "MIN_LENGTH_SLICE"},
		{"MaxLengthSlice", MaxLengthSlice("f", []any{1, 2, 3, 4}, 3), "MAX_LENGTH_SLICE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.rule.Validate(ctx)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			rErr, ok := err.(rules.Error)
			if !ok {
				t.Fatalf("%s returned %T, expected rules.Error value. %v", tc.name, err, err)
			}
			if rErr.Code != tc.code {
				t.Fatalf("%s returned code %q, expected %q", tc.name, rErr.Code, tc.code)
			}
		})
	}
}
