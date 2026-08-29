package validators

import (
	"regexp"

	"github.com/mishudark/rules"
)

var hexColorRegex = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

// HexColor creates a validation Rule that checks if a given string is a
// valid hexadecimal color: #RGB, #RGBA, #RRGGBB, or #RRGGBBAA.
// It considers an empty string as valid (use a separate 'Required' rule if
// emptiness is not allowed).
func HexColor(fieldName, value string) rules.Rule {
	return rules.NewRulePure("RuleValidHexColor", func() error {
		if value == "" {
			return nil
		}

		if !hexColorRegex.MatchString(value) {
			return rules.Error{
				Field: fieldName,
				Err:   "value must be a hexadecimal color (#RGB, #RGBA, #RRGGBB, or #RRGGBBAA)",
				Code:  "INVALID_HEX_COLOR",
			}
		}

		return nil
	})
}
