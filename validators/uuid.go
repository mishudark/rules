package validators

import (
	"regexp"

	"github.com/mishudark/rules"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// UUID creates a validation Rule that checks if a given string is a
// canonically formatted UUID (8-4-4-4-12 hexadecimal digits, any version).
// It considers an empty string as valid (use a separate 'Required' rule if
// emptiness is not allowed).
func UUID(fieldName, value string) rules.Rule {
	return rules.NewRulePure("RuleValidUUID", func() error {
		if value == "" {
			return nil
		}

		if !uuidRegex.MatchString(value) {
			return rules.Error{
				Field: fieldName,
				Err:   "value must be a valid UUID (8-4-4-4-12 hexadecimal format)",
				Code:  "INVALID_UUID",
			}
		}

		return nil
	})
}
