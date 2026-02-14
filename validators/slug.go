package validators

import (
	"regexp"

	"github.com/mishudark/rules"
)

var (
	// slugRegex matches ASCII slugs: letters, numbers, underscores, hyphens.
	slugRegex = regexp.MustCompile(`^[-a-zA-Z0-9_]+$`)

	// unicodeSlugRegex matches Unicode slugs: Unicode letters, Unicode numbers, underscores, hyphens.
	unicodeSlugRegex = regexp.MustCompile(`^[-_\p{L}\p{N}]+$`)
)

// Slug creates a validation Rule that checks if a given string is a valid ASCII slug.
// Slugs can consist of letters, numbers, underscores, or hyphens.
// It considers an empty string as valid (use a separate 'Required' rule if needed).
func Slug(fieldName string, slug string) rules.Rule {
	return rules.NewRulePure("RuleValidSlug", func() error {
		// Allow empty slugs by default, similar to how other rules handle empty strings.
		// Add a separate "Required" rule if slugs cannot be empty.
		if slug == "" {
			return nil
		}

		if !slugRegex.MatchString(slug) {
			return rules.Error{
				Field: fieldName,
				Err:   "Slug must consist only of ASCII letters, numbers, underscores, or hyphens.",
				Code:  "INVALID_SLUG",
			}
		}

		return nil
	})
}

// UnicodeSlug creates a validation Rule that checks if a given string is a valid Unicode slug.
// Slugs can consist of Unicode letters, Unicode numbers, underscores, or hyphens.
// It considers an empty string as valid (use a separate 'Required' rule if needed).
func UnicodeSlug(fieldName string, slug string) rules.Rule {
	return rules.NewRulePure("RuleValidUnicodeSlug", func() error {
		// Allow empty slugs by default, similar to how other rules handle empty strings.
		// Add a separate "Required" rule if slugs cannot be empty.
		if slug == "" {
			return nil
		}

		if !unicodeSlugRegex.MatchString(slug) {
			return rules.Error{
				Field: fieldName,
				Err:   "Slug must consist only of Unicode letters, numbers, underscores, or hyphens.",
				Code:  "INVALID_UNICODE_SLUG",
			}
		}

		return nil
	})
}
