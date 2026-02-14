package validators

import (
	"fmt"
	"strings"

	"github.com/mishudark/rules"
)

// ProhibitNullCharactersValidator checks if a string contains null characters ('\x00').
func prohibitNullCharactersValidator(value string) error {
	if strings.Contains(value, "\x00") {
		return fmt.Errorf("null characters are not allowed")
	}
	return nil
}

// NewProhibitNullCharactersValidator returns a new Rule that validates if a string
// contains any null characters.
func ProhibitNullCharacters(value string) rules.Rule {
	return rules.NewRulePure("prohibit_null_characters_validator", func() error {
		return prohibitNullCharactersValidator(value)
	})
}
