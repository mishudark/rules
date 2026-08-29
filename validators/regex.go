package validators

import (
	"fmt"
	"regexp"

	"github.com/mishudark/rules"
)

// MatchesRegex creates a validation Rule that checks if a given string
// matches the provided regular expression. The pattern is compiled once at
// construction time and panics on an invalid pattern (fail fast, like
// regexp.MustCompile).
// It considers an empty string as valid (use a separate 'Required' rule if
// emptiness is not allowed).
func MatchesRegex(fieldName, value, pattern string) rules.Rule {
	compiled := regexp.MustCompile(pattern)
	ruleName := fmt.Sprintf("RuleMatchesRegex[%s]", fieldName)

	return rules.NewRulePure(ruleName, func() error {
		if value == "" {
			return nil
		}

		if !compiled.MatchString(value) {
			return rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("value does not match the required pattern %q", pattern),
				Code:  "PATTERN_MISMATCH",
			}
		}

		return nil
	})
}
