package validators

import (
	"strings"

	"github.com/mishudark/rules"
)

// Luhn creates a validation Rule that checks if a given string is a valid
// payment card number according to the Luhn checksum. Spaces and dashes are
// ignored; the remaining characters must be digits (12 to 19, the common
// card number lengths).
// It considers an empty string as valid (use a separate 'Required' rule if
// emptiness is not allowed). This checks the checksum only, not whether the
// number was actually issued.
func Luhn(fieldName, value string) rules.Rule {
	return rules.NewRulePure("RuleValidLuhn", func() error {
		digits := strings.Map(func(r rune) rune {
			switch r {
			case ' ', '-':
				return -1
			default:
				return r
			}
		}, value)

		if digits == "" {
			return nil
		}

		if len(digits) < 12 || len(digits) > 19 {
			return rules.Error{
				Field: fieldName,
				Err:   "card number must contain between 12 and 19 digits",
				Code:  "INVALID_CARD_NUMBER",
			}
		}

		sum := 0
		for i := 0; i < len(digits); i++ {
			d := digits[len(digits)-1-i] - '0'
			if d > 9 {
				return rules.Error{
					Field: fieldName,
					Err:   "card number must contain only digits",
					Code:  "INVALID_CARD_NUMBER",
				}
			}

			if i%2 == 1 {
				d *= 2
				if d > 9 {
					d -= 9
				}
			}

			sum += int(d)
		}

		if sum%10 != 0 {
			return rules.Error{
				Field: fieldName,
				Err:   "card number failed the Luhn checksum",
				Code:  "INVALID_CARD_NUMBER",
			}
		}

		return nil
	})
}
