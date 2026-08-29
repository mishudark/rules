package validators

import (
	"regexp"

	"github.com/mishudark/rules"
)

var e164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// PhoneE164 creates a validation Rule that checks if a given string is a
// phone number in E.164 format (for example "+14155552671"): a leading plus,
// a nonzero country code, and up to 15 digits in total.
// It considers an empty string as valid (use a separate 'Required' rule if
// emptiness is not allowed). This checks format only; it does not verify the
// number was assigned or is reachable.
func PhoneE164(fieldName, value string) rules.Rule {
	return rules.NewRulePure("RuleValidPhoneE164", func() error {
		if value == "" {
			return nil
		}

		if !e164Regex.MatchString(value) {
			return rules.Error{
				Field: fieldName,
				Err:   "phone number must be in E.164 format (for example +14155552671)",
				Code:  "INVALID_PHONE_FORMAT",
			}
		}

		return nil
	})
}
