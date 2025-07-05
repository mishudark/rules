package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestRuleValidSlug(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		testName   string
		slug       string
		expectFail bool
		errCode    string
	}{
		{
			testName:   "Valid ASCII",
			slug:       "valid-slug_123",
			expectFail: false,
		},
		{
			testName:   "Invalid ASCII (space)",
			slug:       "invalid slug!",
			expectFail: true,
			errCode:    "INVALID_SLUG",
		},
		{
			testName:   "Invalid ASCII (unicode present)",
			slug:       "valid-slug_üñ1",
			expectFail: true,
			errCode:    "INVALID_SLUG",
		},
		{
			testName:   "Empty Slug",
			slug:       "",
			expectFail: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			rule := RuleValidSlug(tc.testName, tc.slug)
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

func TestRuleValidUnicodeSlug(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		testName   string
		slug       string
		expectFail bool
		errCode    string
	}{
		{
			testName:   "Valid ASCII",
			slug:       "valid-slug_123",
			expectFail: false,
		},
		{
			testName:   "Valid Unicode",
			slug:       "valid-slug_üñ1",
			expectFail: false,
		},
		{
			testName:   "Valid Unicode Korean",
			slug:       "유효한-슬러그_123",
			expectFail: false,
		},
		{
			testName:   "Invalid Unicode (special char)",
			slug:       "invalid-슬러그!",
			expectFail: true,
			errCode:    "INVALID_UNICODE_SLUG",
		},
		{
			testName:   "Empty Slug",
			slug:       "",
			expectFail: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			rule := RuleValidUnicodeSlug(tc.testName, tc.slug)
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
