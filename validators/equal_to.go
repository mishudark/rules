package validators

import (
	"fmt"

	"github.com/mishudark/rules"
)

// EqualTo creates a validation Rule that checks if a given string exactly
// equals an expected value, typically a confirmation field (password
// confirmation, repeated email, ...).
func EqualTo(fieldName, value, expected string) rules.Rule {
	return rules.NewRulePure("RuleEqualTo", func() error {
		if value != expected {
			return rules.Error{
				Field: fieldName,
				Err:   "values do not match",
				Code:  "VALUE_MISMATCH",
			}
		}

		return nil
	})
}

// Between creates a validation Rule that checks if a given numeric value is
// within the inclusive range [min, max]. If min > max the rule never passes.
func Between[T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](fieldName string, value, min, max T) rules.Rule {
	ruleName := fmt.Sprintf("RuleBetween[%s]", fieldName)

	return rules.NewRulePure(ruleName, func() error {
		if value < min || value > max {
			return rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("value (%v) outside the allowed range [%v, %v]", value, min, max),
				Code:  "VALUE_OUT_OF_RANGE",
			}
		}

		return nil
	})
}
