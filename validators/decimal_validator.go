package validators

import (
	"fmt"
	"strings"

	"github.com/mishudark/rules"
)

// decimalValidator validates a decimal number string against max_digits and decimal_places.
func decimalValidator(value string, maxDigits, decimalPlaces int) error {
	if value == "" {
		return rules.Error{
			Code: "DECIMAL_VALUE_CANNOT_BE_EMPTY",
			Err:  "value cannot be empty",
		}
	}

	parts := strings.Split(value, ".")
	var integerPart, fractionalPart string

	if len(parts) == 1 {
		integerPart = parts[0]
		fractionalPart = ""
	} else if len(parts) == 2 {
		integerPart = parts[0]
		fractionalPart = parts[1]
	} else {
		return rules.Error{
			Code: "INVALID_DECIMAL_FORMAT",
			Err:  "invalid decimal format: too many periods",
		}
	}

	if strings.HasPrefix(integerPart, "-") {
		integerPart = integerPart[1:]
	}

	if len(integerPart)+len(fractionalPart) > maxDigits {
		return rules.Error{
			Code: "DECIMAL_TOO_MANY_DIGITS",
			Err:  fmt.Sprintf("the number of digits (%d) exceeds the allowed maximum of %d", len(integerPart)+len(fractionalPart), maxDigits),
		}
	}

	if len(fractionalPart) > decimalPlaces {
		return rules.Error{
			Code: "DECIMAL_TOO_MANY_DECIMAL_PLACES",
			Err:  fmt.Sprintf("the number of decimal places (%d) exceeds the allowed maximum of %d", len(fractionalPart), decimalPlaces),
		}
	}

	if len(integerPart) > maxDigits-decimalPlaces {
		return rules.Error{
			Code: "DECIMAL_TOO_MANY_INTEGER_DIGITS",
			Err:  fmt.Sprintf("the number of integer digits (%d) exceeds the allowed maximum of %d", len(integerPart), maxDigits-decimalPlaces),
		}
	}

	return nil
}

// DecimalValidator returns a new Rule that validates if a string is a valid decimal number
// with the given constraints.
func DecimalValidator(value string, maxDigits, decimalPlaces int) rules.Rule {
	return rules.NewRulePure("decimal_validator", func() error {
		return decimalValidator(value, maxDigits, decimalPlaces)
	})
}
