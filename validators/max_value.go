package validators

import (
	"fmt"

	"github.com/mishudark/rules"
	"golang.org/x/exp/constraints"
)

// MaxValue creates a validation Rule that checks if a given numeric value
// is less than or equal to a specified maximum value.
// It uses generics to work with any ordered type (int, float64, etc.).
func MaxValue[T constraints.Ordered](fieldName string, value T, maxValue T) rules.Rule {
	ruleName := fmt.Sprintf("RuleMaxValue[%s]", fieldName)

	return rules.NewRulePure(ruleName, func() error {
		// Check if the value exceeds the maximum allowed value
		if value > maxValue {
			return rules.Error{
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
