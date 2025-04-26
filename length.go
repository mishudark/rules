package rules

import "fmt"

func LengthString(fieldName, value string, length int) Rule {
	return NewSimpleRule("lengthString",
		func() error {
			if len([]rune(value)) == length {
				return nil
			}

			return &Error{
				Field: fieldName,
				Err:   fmt.Sprintf("expected %d, got %d", length, len(value)),
			}
		},
	)
}

func LengthSlice(fieldName string, value []any, length int) Rule {
	return NewSimpleRule("lengthSlice",
		func() error {
			if len(value) == length {
				return nil
			}

			return &Error{
				Field: fieldName,
				Err:   fmt.Sprintf("expected %d, got %d", length, len(value)),
			}
		},
	)
}
