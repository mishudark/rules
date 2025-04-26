package rules

import "fmt"

func LengthString(fieldName, value string, length int) Rule {
	return NewSimpleRule("lengthString",
		func() *Error {
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
		func() *Error {
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
