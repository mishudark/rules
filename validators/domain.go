package validators

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8" // Needed for RuneCountInString and ASCII check

	"github.com/mishudark/rules"
)

const (
	// Max length for domain names based on RFC 1034/1035, often cited around 253-255.
	maxDomainLengthAdvanced = 255
	// Max length for a single domain label.
	maxDomainLabelLength = 63
)

var (
	// Regex for domains allowing Unicode characters (IDNA).
	// More lenient: Checks general structure (non-empty labels separated by dots, ending in TLD).
	// Doesn't strictly enforce hyphen rules or label length here; those are checked manually.
	// Allows structure like: label.label.tld or label.tld
	// Handles basic Punycode prefix xn-- in TLD. Case-insensitive (?i).
	// Label part: `(?:[a-z\p{L}0-9](?:[a-z\p{L}0-9-]*[a-z\p{L}0-9])?)` - allows hyphens inside, starts/ends with alphanum/unicode letter
	// TLD part: `(?:[a-z\p{L}-]{2,}|xn--[a-z0-9]{1,})` - Allows letters/hyphens (min 2) or punycode
	// Combined: `^(label\.)+(tld)$` structure
	idnaDomainRegex = regexp.MustCompile(`(?i)^(?:[a-z\p{L}0-9](?:[a-z\p{L}0-9-]*[a-z\p{L}0-9])?\.)+(?:[a-z\p{L}-]{2,}|xn--[a-z0-9]{1,})$`)

	// Regex for ASCII-only domains. More lenient structure check.
	// Label part: `(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)`
	// TLD part: `[a-z]{2,}`
	asciiDomainRegex = regexp.MustCompile(`(?i)^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$`)

	// Regex to check if a string contains only ASCII characters.
	asciiOnlyRegex = regexp.MustCompile(`^[\x00-\x7F]*$`)
)

// ValidDomainNameAdvanced creates a validation Rule that checks if a given string
// is a valid domain name formaat.
// It supports enabling/disabling Internationalized Domain Names (IDNs).
//
// IMPORTANT: This validation checks format based on common rules and regex.
// It does NOT verify domain existence or check against official TLD lists.
// Due to Go's regex limitations (no lookarounds), it combines regex with manual checks.
//
// It considers an empty string as valid (use a separate 'Required' rule if needed).
func ValidDomainNameAdvanced(fieldName string, domain string, acceptIdna bool) rules.Rule {
	ruleName := fmt.Sprintf("RuleValidDomainNameAdvanced[%s, idna=%t]", fieldName, acceptIdna)

	return rules.NewRulePure(ruleName, func() error {
		trimmedDomain := strings.TrimSpace(domain)
		if trimmedDomain == "" {
			return nil // Empty string is not an invalid *format*
		}

		// Overall Length Check (Bytes)
		if len(trimmedDomain) > maxDomainLengthAdvanced {
			return rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("Domain name exceeds maximum length of %d bytes", maxDomainLengthAdvanced),
				Code:  "INVALID_DOMAIN_LENGTH",
			}
		}

		// ASCII Check (if IDNA is not accepted)
		if !acceptIdna {
			if !asciiOnlyRegex.MatchString(trimmedDomain) {
				return rules.Error{
					Field: fieldName,
					Err:   "Domain name contains non-ASCII characters, but IDNA is not accepted",
					Code:  "NON_ASCII_DOMAIN_NOT_ALLOWED",
				}
			}
		}

		// Reject trailing dot early, as Split behavior depends on it.
		if strings.HasSuffix(trimmedDomain, ".") {
			return rules.Error{
				Field: fieldName,
				Err:   "Domain name must not end with a dot",
				Code:  "INVALID_DOMAIN_TRAILING_DOT", // Specific code for this case
			}
		}

		labels := strings.Split(trimmedDomain, ".")

		if len(labels) < 2 { // Must have at least one label and a TLD
			// This also catches cases like "com" or just "hostname"
			return rules.Error{
				Field: fieldName,
				Err:   "Invalid domain name format (must contain at least one label and a TLD)",
				Code:  "INVALID_DOMAIN_STRUCTURE",
			}
		}

		for i, label := range labels {
			if len(label) == 0 {
				// Catch cases like "example..com"
				return rules.Error{
					Field: fieldName,
					Err:   fmt.Sprintf("Invalid domain name format (empty label found before '%s')", strings.Join(labels[i+1:], ".")),
					Code:  "INVALID_DOMAIN_EMPTY_LABEL",
				}
			}

			// Check label length (in characters, using RuneCountInString)
			// Note: RFC specifies octets (bytes) for length limits in DNS, but Unicode complicates this.
			// check character length. Let's stick to character count for labels.
			if utf8.RuneCountInString(label) > maxDomainLabelLength {
				return rules.Error{
					Field: fieldName,
					Err:   fmt.Sprintf("Domain label '%s' exceeds maximum length of %d characters", label, maxDomainLabelLength),
					Code:  "INVALID_DOMAIN_LABEL_LENGTH",
				}
			}

			// Check for leading or trailing hyphens in the label
			if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
				return rules.Error{
					Field: fieldName,
					Err:   fmt.Sprintf("Domain label '%s' must not start or end with a hyphen", label),
					Code:  "INVALID_DOMAIN_LABEL_HYPHEN",
				}
			}

			// TLD specific checks (last label)
			if i == len(labels)-1 {
				// Basic TLD length (characters) - should be at least 2
				if utf8.RuneCountInString(label) < 2 {
					return rules.Error{
						Field: fieldName,
						Err:   fmt.Sprintf("Top-level domain '%s' must be at least 2 characters long", label),
						Code:  "INVALID_DOMAIN_TLD_LENGTH",
					}
				}
				// Check for Punycode prefix if not accepting IDNA
				if !acceptIdna && strings.HasPrefix(label, "xn--") {
					return rules.Error{
						Field: fieldName,
						Err:   fmt.Sprintf("Top-level domain '%s' uses Punycode, but IDNA is not accepted", label),
						Code:  "PUNYCODE_TLD_NOT_ALLOWED",
					}
				}
			} else {
				// Non-TLD label checks (if any differ from TLD checks)
				// Ensure non-TLD labels don't look like Punycode TLDs if that's a requirement
				// (Usually not needed, but possible)
			}
		}

		// Now that specific errors are caught, use regex for general format validation.
		var chosenRegex *regexp.Regexp
		if acceptIdna {
			// Use a regex that checks structure but is lenient on content details already checked manually
			// This regex focuses on `label.label.tld` structure with allowed chars, minimum TLD length.
			// It's simplified because manual checks handle hyphens, label length etc.
			chosenRegex = regexp.MustCompile(`(?i)^(?:[a-z\p{L}0-9](?:[a-z\p{L}0-9-]{0,61}[a-z\p{L}0-9])?\.)+(?:[a-z\p{L}-]{2,}|xn--[a-z0-9]{1,59})$`)
		} else {
			// ASCII version - simplified structure check
			chosenRegex = regexp.MustCompile(`(?i)^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
		}

		if !chosenRegex.MatchString(trimmedDomain) {
			// This should ideally catch fewer cases now, maybe complex structural issues missed by manual checks.
			return rules.Error{
				Field: fieldName,
				Err:   "Invalid domain name format (failed final regex structure check)",
				Code:  "INVALID_DOMAIN_FORMAT_REGEX",
			}
		}

		// All checks passed
		return nil
	})
}
