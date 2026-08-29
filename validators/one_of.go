package validators

import (
	"fmt"
	"slices"

	"github.com/mishudark/rules"
)

// OneOf creates a validation Rule that checks if a given value is one of the
// allowed values. Comparison uses Go's == on the underlying type.
// As with the format validators, an empty string is considered valid
// (use a separate 'Required' rule if emptiness is not allowed).
func OneOf[T comparable](fieldName string, value T, allowed []T) rules.Rule {
	ruleName := fmt.Sprintf("RuleOneOf[%s]", fieldName)

	return rules.NewRulePure(ruleName, func() error {
		if s, ok := any(value).(string); ok && s == "" {
			return nil
		}

		if !slices.Contains(allowed, value) {
			return rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("value %v is not one of the allowed values", value),
				Code:  "VALUE_NOT_ALLOWED",
			}
		}

		return nil
	})
}
