package validators

import (
	"strings"

	"github.com/mishudark/rules"
)

// Required creates a validation Rule that checks if a given string is present
// (non-empty after trimming whitespace). It is the counterpart to the format
// validators, which consider empty strings valid by convention.
func Required(fieldName, value string) rules.Rule {
	return rules.NewRulePure("RuleRequired", func() error {
		if strings.TrimSpace(value) == "" {
			return rules.Error{
				Field: fieldName,
				Err:   "this field is required",
				Code:  "REQUIRED",
			}
		}

		return nil
	})
}

// RequiredSlice creates a validation Rule that checks if a given slice is
// present (non-nil and non-empty).
func RequiredSlice[T any](fieldName string, value []T) rules.Rule {
	return rules.NewRulePure("RuleRequiredSlice", func() error {
		if len(value) == 0 {
			return rules.Error{
				Field: fieldName,
				Err:   "this field is required",
				Code:  "REQUIRED",
			}
		}

		return nil
	})
}
