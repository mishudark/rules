package rules

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

// RuleMaxValue creates a validation Rule that checks if a given numeric value
// is less than or equal to a specified maximum value.
// It uses generics to work with any ordered type (int, float64, etc.).
func RuleMaxValue[T constraints.Ordered](fieldName string, value T, maxValue T) Rule {
	ruleName := fmt.Sprintf("RuleMaxValue[%s]", fieldName)

	return NewRulePure(ruleName, func() error {
		// Check if the value exceeds the maximum allowed value
		if value > maxValue {
			return Error{
				Field: fieldName,
				// Use %v for generic printing of the values
				Err:  fmt.Sprintf("Value (%v) exceeds the maximum allowed value (%v)", value, maxValue),
				Code: "VALUE_EXCEEDS_MAX",
			}
		}

		// Value is within the allowed range (<= maxValue)
		return nil
	})
}
