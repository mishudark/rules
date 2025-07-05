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
				}
			} else if err != nil {
				t.Errorf("Expected validation to succeed, but it failed: %v", err)
			}
		})
	}
}
