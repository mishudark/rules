package validators

import (
	"fmt"

	"github.com/mishudark/rules"
)

func LengthString(fieldName, value string, length int) rules.Rule {
	return rules.NewRulePure("lengthString",
		func() error {
			if len([]rune(value)) == length {
				return nil
			}

			return &rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("expected %d, got %d", length, len(value)),
				Code:  "LENGTH_STRING",
			}
		},
	)
}

func LengthSlice(fieldName string, value []any, length int) rules.Rule {
	return rules.NewRulePure("lengthSlice",
		func() error {
			if len(value) == length {
				return nil
			}

			return &rules.Error{
				Field: fieldName,
				Err:   fmt.Sprintf("expected %d, got %d", length, len(value)),
				Code:  "LENGTH_SLICE",
			}
		},
	)
}
