package validators

import (
	"fmt"
	"strings"

	"github.com/mishudark/rules"
)

// DecimalValidator validates a decimal number string against max_digits and decimal_places.
func DecimalValidator(value string, maxDigits, decimalPlaces int) error {
	if value == "" {
		return fmt.Errorf("value cannot be empty")
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
		return fmt.Errorf("invalid decimal format: too many periods")
	}

	if strings.HasPrefix(integerPart, "-") {
		integerPart = integerPart[1:]
	}

	if len(integerPart)+len(fractionalPart) > maxDigits {
		return fmt.Errorf("the number of digits (%d) exceeds the allowed maximum of %d", len(integerPart)+len(fractionalPart), maxDigits)
	}

	if len(fractionalPart) > decimalPlaces {
		return fmt.Errorf("the number of decimal places (%d) exceeds the allowed maximum of %d", len(fractionalPart), decimalPlaces)
	}

	if len(integerPart) > maxDigits-decimalPlaces {
		return fmt.Errorf("the number of integer digits (%d) exceeds the allowed maximum of %d", len(integerPart), maxDigits-decimalPlaces)
	}

	return nil
}

// NewDecimalValidator returns a new Rule that validates if a string is a valid decimal number
// with the given constraints.
func NewDecimalValidator(value string, maxDigits, decimalPlaces int) rules.Rule {
	return rules.NewRulePure("decimal_validator", func() error {
		return DecimalValidator(value, maxDigits, decimalPlaces)
	})
}
