package validators

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mishudark/rules"
)

// urlValidator validates that a given value is a valid URL.
// It can also check against a list of allowed schemes (e.g., "http", "https").
func urlValidator(value string, schemes []string) error {
	if value == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if len(schemes) > 0 {
		isSchemeAllowed := false
		for _, scheme := range schemes {
			if strings.EqualFold(parsedURL.Scheme, scheme) {
				isSchemeAllowed = true
				break
			}
		}
		if !isSchemeAllowed {
			return fmt.Errorf("URL scheme '%s' is not in the list of allowed schemes", parsedURL.Scheme)
		}
	}

	return nil
}

// URL returns a new Rule that validates if a string is a valid URL.
// If schemes are provided, it also validates that the URL's scheme is one of the allowed schemes.
func URL(value string, schemes []string) rules.Rule {
	return rules.NewRulePure("url_validator", func() error {
		return urlValidator(value, schemes)
	})
}
