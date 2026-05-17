package validators

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/mishudark/rules"
)

// urlValidator validates that a given value is a valid URL.
// It can also check against a list of allowed schemes (e.g., "http", "https").
func urlValidator(value string, schemes []string) error {
	if value == "" {
		return rules.Error{
			Code: "URL_CANNOT_BE_EMPTY",
			Err:  "url cannot be empty",
		}
	}

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return rules.Error{
			Code: "INVALID_URL_FORMAT",
			Err:  fmt.Sprintf("invalid url format: %s", err.Error()),
		}
	}

	if len(schemes) > 0 && !slices.ContainsFunc(schemes, func(s string) bool {
		return strings.EqualFold(parsedURL.Scheme, s)
	}) {
		return rules.Error{
			Code: "URL_SCHEME_NOT_ALLOWED",
			Err:  fmt.Sprintf("url scheme '%s' is not in the list of allowed schemes", parsedURL.Scheme),
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
