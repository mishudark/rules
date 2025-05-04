package rules

import (
	"context"
	"strings"
	"testing"
)

func TestRuleValidDomainNameAdvanced(t *testing.T) {
	testCases := []struct {
		testName   string
		domain     string
		acceptIdna bool
		expectFail bool
		errCode    string
	}{
		{"Valid ASCII", "example.com", false, false, ""},
		{"Valid ASCII Subdomain", "sub.example.co.uk", false, false, ""},
		{"Valid IDNA Punycode", "xn--prfg0a.xn--kpry57d", true, false, ""}, // Prüfening.de
		{"Valid Unicode", "prüfening.de", true, false, ""},
		{"Invalid Chars ASCII", "ex@mple.com", false, true, "INVALID_DOMAIN_FORMAT_REGEX"},
		{"Invalid Chars IDNA", "ex@mple.com", true, true, "INVALID_DOMAIN_FORMAT_REGEX"},
		{"Invalid TLD ASCII", "example.c", false, true, "INVALID_DOMAIN_TLD_LENGTH"}, // Regex checks TLD length >= 2
		{"Invalid TLD IDNA", "example.c", true, true, "INVALID_DOMAIN_TLD_LENGTH"},
		{"Leading Hyphen Label", "-label.com", true, true, "INVALID_DOMAIN_LABEL_HYPHEN"},
		{"Trailing Hyphen Label", "label-.com", true, true, "INVALID_DOMAIN_LABEL_HYPHEN"},
		{"Leading Hyphen TLD", "example.com-", true, true, "INVALID_DOMAIN_LABEL_HYPHEN"},  // TLD is also a label
		{"Trailing Hyphen TLD", "example.-com", true, true, "INVALID_DOMAIN_LABEL_HYPHEN"}, // TLD is also a label
		{"Empty Label Start", ".example.com", true, true, "INVALID_DOMAIN_EMPTY_LABEL"},    // Added test
		{"Empty Label Middle", "example..com", true, true, "INVALID_DOMAIN_EMPTY_LABEL"},
		{"Empty Label End", "example.com.", true, true, "INVALID_DOMAIN_TRAILING_DOT"}, // Added test (assuming trailing dot invalid)
		{"Long Label", strings.Repeat("a", 64) + ".com", true, true, "INVALID_DOMAIN_LABEL_LENGTH"},
		{"Long Domain", strings.Repeat("a", 64) + "." + strings.Repeat("b", 64) + "." + strings.Repeat("c", 64) + "." + strings.Repeat("d", 64) + ".com", true, true, "INVALID_DOMAIN_LENGTH"}, // > 255 bytes
		{"Unicode Not Allowed", "prüfening.de", false, true, "NON_ASCII_DOMAIN_NOT_ALLOWED"},
		{"Only TLD", "com", true, true, "INVALID_DOMAIN_STRUCTURE"},          // Added test
		{"Just Hyphens", "--.--", true, true, "INVALID_DOMAIN_LABEL_HYPHEN"}, // Added test
		{"Valid Punycode TLD", "example.xn--fiqs8s", true, false, ""},        // Example: example.中国
	}

	// Create a background context (often needed for rule execution)
	ctx := context.Background()

	// Iterate through test cases using t.Run for better output separation
	for _, tc := range testCases {
		// Use t.Run to create a subtest for each case
		t.Run(tc.testName, func(t *testing.T) {
			// Create the rule instance for this test case
			rule := RuleValidDomainNameAdvanced(tc.testName, tc.domain, tc.acceptIdna)

			// Execute the validation logic
			// Prepare is no-op for RulePure, so we directly call Validate
			err := rule.Validate(ctx)

			// Assert the results
			if tc.expectFail {
				// We expect an error
				if err == nil {
					t.Errorf("Expected validation to fail for domain '%s' (idna=%t), but it succeeded.", tc.domain, tc.acceptIdna)
				} else {
					// Optional: Check if the error is the specific rules.Error type and code
					if rulesErr, ok := err.(Error); ok {
						if rulesErr.Code != tc.errCode {
							t.Errorf("Expected error code '%s' but got '%s' for domain '%s' (idna=%t). Error: %v",
								tc.errCode, rulesErr.Code, tc.domain, tc.acceptIdna, rulesErr)
						}
						// If codes match, test passes for this failure case - no further output needed unless verbose
					} else {
						// If it's a different error type, we might still consider it a pass
						// depending on how strict the test needs to be about the error type/code.
						// For now, just log that an error occurred as expected.
						t.Logf("Validation failed as expected for domain '%s' (idna=%t), error: %v", tc.domain, tc.acceptIdna, err)
					}
				}
			} else {
				// We expect success (no error)
				if err != nil {
					t.Errorf("Expected validation to succeed for domain '%s' (idna=%t), but it failed: %v", tc.domain, tc.acceptIdna, err)
				}
			}
		})
	}
}
