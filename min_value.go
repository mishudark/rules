package rules

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

// RuleMinValue creates a validation Rule that checks if a given numeric value
// is more than or equal to a specified minimum value.
// It uses generics to work with any ordered type (int, float64, etc.).
func RuleMinValue[T constraints.Ordered](fieldName string, value T, minValue T) Rule {
	ruleName := fmt.Sprintf("RuleMinValue[%s]", fieldName)

	return NewRulePure(ruleName, func() error {
		// Check if the value is lower than the minimum allowed value
		if value < minValue {
			return Error{
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
