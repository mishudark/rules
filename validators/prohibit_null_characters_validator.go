package validators

import (
	"strings"

	"github.com/mishudark/rules"
)

// prohibitNullCharactersValidator checks if a string contains null characters ('\x00').
func prohibitNullCharactersValidator(value string) error {
	if strings.Contains(value, "\x00") {
		return rules.Error{
			Code: "NULL_CHARACTERS_FOUND",
			Err:  "null characters are not allowed",
		}
	}
	return nil
}

// ProhibitNullCharacters returns a new Rule that validates if a string
// contains any null characters.
func ProhibitNullCharacters(value string) rules.Rule {
	return rules.NewRulePure("prohibit_null_characters_validator", func() error {
		return prohibitNullCharactersValidator(value)
	})
}
