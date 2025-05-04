package rules

import (
	"context" // Context is often needed, even if not used by RulePure's Validate
	"testing" // Import the standard testing package
)

// TestRuleValidSlug tests the slug validation rule.
func TestRuleValidSlug(t *testing.T) {
	// Background context for validation
	ctx := context.Background()

	// Define test cases based on the example usage
	testCases := []struct {
		testName     string // Name for the subtest
		slug         string // Input slug string
		allowUnicode bool   // Parameter for the rule
		expectFail   bool   // Whether validation is expected to fail
		errCode      string // Expected error code on failure
	}{
		{
			testName:     "Valid ASCII",
			slug:         "valid-slug_123",
			allowUnicode: false,
			expectFail:   false,
		},
		{
			testName:     "Invalid ASCII (space)",
			slug:         "invalid slug!",
			allowUnicode: false,
			expectFail:   true,
			errCode:      "INVALID_SLUG",
		},
		{
			testName:     "Invalid ASCII (unicode present)",
			slug:         "valid-slug_üñ1",
			allowUnicode: false,
			expectFail:   true,
			errCode:      "INVALID_SLUG",
		},
		{
			testName:     "Valid ASCII (unicode allowed)",
			slug:         "valid-slug_123",
			allowUnicode: true,
			expectFail:   false,
		},
		{
			testName:     "Valid Unicode (unicode allowed)",
			slug:         "valid-slug_üñ1",
			allowUnicode: true,
			expectFail:   false,
		},
		{
			testName:     "Valid Unicode Korean (unicode allowed)",
			slug:         "유효한-슬러그_123", // Korean example
			allowUnicode: true,
			expectFail:   false,
		},
		{
			testName:     "Invalid Unicode (special char)",
			slug:         "invalid-슬러그!", // Contains exclamation mark
			allowUnicode: true,
			expectFail:   true,
			errCode:      "INVALID_SLUG",
		},
		{
			testName:     "Empty Slug (unicode allowed)",
			slug:         "",
			allowUnicode: true,
			expectFail:   false, // Empty is allowed by the rule itself
		},
		{
			testName:     "Empty Slug (ASCII only)",
			slug:         "",
			allowUnicode: false,
			expectFail:   false, // Empty is allowed by the rule itself
		},
	}

	// Iterate through test cases using t.Run for better output separation
	for _, tc := range testCases {
		// Use t.Run to create a subtest for each case
		t.Run(tc.testName, func(t *testing.T) {
			// Create a new rule instance for this test case
			rule := RuleValidSlug(tc.testName, tc.slug, tc.allowUnicode)

			// Execute the validation logic
			// Prepare is no-op for RulePure, so we directly call Validate
			err := rule.Validate(ctx)

			// Assert the results
			if tc.expectFail {
				// We expect an error
				if err == nil {
					t.Errorf("Expected validation to fail for slug '%s' (unicode=%t), but it succeeded.", tc.slug, tc.allowUnicode)
				} else {
					// Optional: Check if the error is the specific rules.Error type and code
					if rulesErr, ok := err.(Error); ok {
						if tc.errCode != "" && rulesErr.Code != tc.errCode {
							t.Errorf("Expected error code '%s' but got '%s' for slug '%s' (unicode=%t). Error: %v",
								tc.errCode, rulesErr.Code, tc.slug, tc.allowUnicode, rulesErr)
						}
						// If codes match or no specific code was expected, log success implicitly
					} else {
						// If it's a different error type, log it but might still be considered a pass for 'expectFail'
						// t.Logf("Validation failed as expected for slug '%s' (unicode=%t), error: %v", tc.slug, tc.allowUnicode, err)
					}
				}
			} else {
				// We expect success (no error)
				if err != nil {
					t.Errorf("Expected validation to succeed for slug '%s' (unicode=%t), but it failed: %v", tc.slug, tc.allowUnicode, err)
				}
				// If err is nil, test passes for this success case
			}
		})
	}
}
