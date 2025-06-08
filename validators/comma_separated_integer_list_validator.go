package validators

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mishudark/rules"
)

// validateCommaSeparatedIntegerList validates that the given value is a comma-separated list of integers.
func validateCommaSeparatedIntegerList(value string) error {
	parts := strings.SplitSeq(value, ",")
	for part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if _, err := strconv.Atoi(trimmedPart); err != nil {
			return fmt.Errorf("'%s' is not a valid integer in the list", trimmedPart)
		}
	}
	return nil
}

// NewValidateCommaSeparatedIntegerList returns a new rule that validates if a string is a comma-separated list of integers.
func NewValidateCommaSeparatedIntegerList(value string) rules.Rule {
	return rules.NewRulePure("validateCommaSeparatedIntegerList", func() error {
		return validateCommaSeparatedIntegerList(value)
	})
}
