package rules

import (
	"fmt"
	"regexp"
)

var (
	// slugRegex matches ASCII slugs: letters, numbers, underscores, hyphens.
	slugRegex = regexp.MustCompile(`^[-a-zA-Z0-9_]+$`)

	// unicodeSlugRegex matches Unicode slugs: Unicode letters, Unicode numbers, underscores, hyphens.
	// In Go, \w is ASCII-only ([0-9A-Za-z_]). For Unicode letters/numbers, use \p{L} and \p{N}.
	unicodeSlugRegex = regexp.MustCompile(`^[-_\p{L}\p{N}]+$`)
)

// RuleValidSlug creates a validation Rule that checks if a given string is a valid slug.
// Slugs can consist of letters, numbers, underscores, or hyphens.
// If allowUnicode is true, it permits Unicode letters and numbers; otherwise, only ASCII.
// It considers an empty string as valid (use a separate 'Required' rule if needed).
func RuleValidSlug(fieldName string, slug string, allowUnicode bool) Rule {
	ruleName := fmt.Sprintf("RuleValidSlug[%s, unicode=%t]", fieldName, allowUnicode)

	return NewRulePure(ruleName, func() error {
		// Allow empty slugs by default, similar to how other rules handle empty strings.
		// Add a separate "Required" rule if slugs cannot be empty.
		if slug == "" {
			return nil
		}

		var chosenRegex *regexp.Regexp
		var message string

		if allowUnicode {
			chosenRegex = unicodeSlugRegex
			message = "Slug must consist only of Unicode letters, numbers, underscores, or hyphens."
		} else {
			chosenRegex = slugRegex
			message = "Slug must consist only of ASCII letters, numbers, underscores, or hyphens."
		}

		// Check if the slug matches the chosen pattern.
		if !chosenRegex.MatchString(slug) {
			return Error{
				Field: fieldName,
				Err:   message,
				Code:  "INVALID_SLUG",
			}
		}

		return nil
	})
}
