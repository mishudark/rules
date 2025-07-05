package validators

import (
	"fmt"

	"github.com/mishudark/rules"
)

// MinLengthString checks if the string value has a minimum length
func MinLengthString(fieldName, value string, min int) rules.Rule {
	return rules.NewRulePure("minLengthString",
		func() error {
			if len([]rune(value)) >= min {
				return nil
			}

			return &rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("expected minimum %d, got %d", min, len(value)),
				Code:  "MIN_LENGTH_STRING",
			}
		},
	)
}

// MaxLengthString checks if the string value has a maximum length
func MaxLengthString(fieldName, value string, max int) rules.Rule {
	return rules.NewRulePure("maxLengthString",
		func() error {
			if len([]rune(value)) <= max {
				return nil
			}

			return &rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("expected maximum %d, got %d", max, len(value)),
				Code:  "MAX_LENGTH_STRING",
			}
		},
	)
}

// MinLengthSlice checks if the slice value has a minimum length
func MinLengthSlice(fieldName string, value []any, min int) rules.Rule {
	return rules.NewRulePure("minLengthSlice",
		func() error {
			if len(value) >= min {
				return nil
			}

			return &rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("expected minimum %d, got %d", min, len(value)),
				Code:  "MIN_LENGTH_SLICE",
			}
		},
	)
}

// MaxLengthSlice checks if the slice value has a maximum length
func MaxLengthSlice(fieldName string, value []any, max int) rules.Rule {
	return rules.NewRulePure("maxLengthSlice",
		func() error {
			if len(value) <= max {
				return nil
			}

			return &rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("expected maximum %d, got %d", max, len(value)),
				Code:  "MAX_LENGTH_SLICE",
			}
		},
	)
}
