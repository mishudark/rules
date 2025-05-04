package rules

import (
	"fmt"
	"net/mail" // Standard library package for email address parsing (RFC 5322)
	"strings"
)

// RuleValidEmail creates a validation Rule that checks if a given string
// is a valid email address according to RFC 5322 format.
// It uses Go's standard library `net/mail.ParseAddress`.
// Note: This rule checks format only. It does not check if the domain exists or if the mailbox is active.
// It considers an empty string as valid (use a separate 'Required' rule if the email must be present).
func RuleValidEmail(fieldName string, email string) Rule {
	ruleName := fmt.Sprintf("RuleValidEmail[%s]", fieldName)

	return NewRulePure(ruleName, func() error {
		// If the email string is empty, consider it valid for format purposes.
		// Use a separate 'Required' rule if emptiness is not allowed.
		if strings.TrimSpace(email) == "" {
			return nil // Empty string is not an invalid *format*
		}

		// Use the standard library's parser.
		// It parses addresses like "Bob <bob@example.com>" or just "bob@example.com".
		_, err := mail.ParseAddress(email)
		if err != nil {
			// Parsing failed, so the format is invalid.
			return Error{
				Field: fieldName,
				Err:   fmt.Sprintf("Invalid email address format: %v", err), // Include the parser's error for detail
				Code:  "INVALID_EMAIL_FORMAT",
			}
		}

		// If parsing succeeded without error, the format is valid.
		return nil
	})
}
