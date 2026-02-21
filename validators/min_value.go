package validators

import (
	"cmp"
	"fmt"

	"github.com/mishudark/rules"
)

// MinValue creates a validation Rule that checks if a given numeric value
// is more than or equal to a specified minimum value.
// It uses generics to work with any ordered type (int, float64, etc.).
func MinValue[T cmp.Ordered](fieldName string, value T, minValue T) rules.Rule {
	ruleName := fmt.Sprintf("RuleMinValue[%s]", fieldName)

	return rules.NewRulePure(ruleName, func() error {
		// Check if the value is lower than the minimum allowed value
		if value < minValue {
			return rules.Error{
				Field: fieldName,
				// Use %v for generic printing of the values
				Err:  fmt.Sprintf("Value (%v) below the minimum allowed value (%v)", value, minValue),
				Code: "VALUE_LOWER_MIN",
			}
		}

		// Value is within the allowed range (>= minValue)
		return nil
	})
}
