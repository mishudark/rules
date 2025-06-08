package validators

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mishudark/rules"
)

func TestRuleValidContentType(t *testing.T) {
	ctx := context.Background()

	// Simulate file content for various types
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52} // Real PNG
	jpgData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}                         // Real JPG
	txtData := []byte("This is plain text, which should be detected as text/plain.")                                  // Plain text
	htmlData := []byte("<!DOCTYPE html><html><head><title>Test</title></head><body>Hello</body></html>")              // HTML
	emptyData := []byte{}                                                                                             // Empty

	// Define test cases
	testCases := []struct {
		testName     string   // Name for the subtest
		data         []byte   // Input data for the reader
		allowedMIMEs []string // Allowed MIME types for the rule
		expectFail   bool     // Whether validation is expected to fail
		errCode      string   // Expected error code on failure (if applicable)
	}{
		{
			testName:     "PNG allowed PNG/JPG",
			data:         pngData,
			allowedMIMEs: []string{"image/png", "image/jpeg"},
			expectFail:   false,
		},
		{
			testName:     "JPG allowed PNG/JPG",
			data:         jpgData,
			allowedMIMEs: []string{"image/png", "image/jpeg"},
			expectFail:   false,
		},
		{
			testName:     "TXT not allowed PNG/JPG",
			data:         txtData,
			allowedMIMEs: []string{"image/png", "image/jpeg"},
			expectFail:   true,
			errCode:      "CONTENT_TYPE_MISMATCH",
		},
		{
			testName:     "PNG not allowed JPG",
			data:         pngData,
			allowedMIMEs: []string{"image/jpeg"},
			expectFail:   true,
			errCode:      "CONTENT_TYPE_MISMATCH",
		},
		{
			testName:     "HTML allowed text/html",
			data:         htmlData,
			allowedMIMEs: []string{"text/html"},
			expectFail:   false,
		},
		{
			testName:     "Empty file requires PNG",
			data:         emptyData,
			allowedMIMEs: []string{"image/png"},
			expectFail:   true,
			errCode:      "CONTENT_TYPE_EMPTY_FILE",
		},
		{
			testName:     "Empty file allowed (no specific MIME)",
			data:         emptyData,
			allowedMIMEs: []string{}, // Empty list means allow anything detected
			expectFail:   false,
		},
		{
			testName:     "Case insensitivity",
			data:         jpgData,
			allowedMIMEs: []string{"IMAGE/JPEG"}, // Uppercase allowed MIME
			expectFail:   false,                  // Should still pass
		},
		{
			testName:     "MIME with parameters mismatch",
			data:         txtData,                                    // Detects text/plain; charset=utf-8
			allowedMIMEs: []string{"text/plain; charset=iso-8859-1"}, // Different param
			expectFail:   true,                                       // Should fail as only type/subtype is compared
			errCode:      "CONTENT_TYPE_MISMATCH",
		},
		{
			testName:     "MIME with parameters match type/subtype",
			data:         txtData,                // Detects text/plain; charset=utf-8
			allowedMIMEs: []string{"text/plain"}, // Allows base type
			expectFail:   false,                  // Should pass
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create a fresh reader for this specific test run
			freshReader := bytes.NewReader(tc.data)

			// Create a new rule instance for this test case
			// Field name can be derived from test name or be static
			fieldName := strings.ReplaceAll(tc.testName, " ", "_")
			rule := NewRuleContentType(fieldName, freshReader, tc.allowedMIMEs)

			// Execute the validation logic
			var err error
			prepareErr := rule.Prepare(ctx)
			if prepareErr != nil {
				// If Prepare fails (e.g., reader is nil, though unlikely here), capture that error
				err = prepareErr
			} else {
				// Otherwise, run Validate
				err = rule.Validate(ctx)
			}

			// Assert the results
			if tc.expectFail {
				// We expect an error
				if err == nil {
					t.Errorf("Expected validation to fail, but it succeeded.")
				} else {
					// Optional: Check if the error is the specific rules.Error type and code
					if rulesErr, ok := err.(rules.Error); ok {
						if tc.errCode != "" && rulesErr.Code != tc.errCode {
							t.Errorf("Expected error code '%s' but got '%s'. Error: %v",
								tc.errCode, rulesErr.Code, rulesErr)
						}
						// If codes match or no specific code was expected, log success implicitly
					} else {
						// If it's a different error type, log it but might still be considered a pass for 'expectFail'
						t.Logf("Validation failed as expected with non-rules.Error: %v", err)
					}
				}
			} else {
				// We expect success (no error)
				if err != nil {
					t.Errorf("Expected validation to succeed, but it failed: %v", err)
				}
				// If err is nil, test passes for this success case
			}
		})
	}
}
