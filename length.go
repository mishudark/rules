package rules

import "fmt"

func Lenght(fieldName string, item any, lenght int) Rule {
	switch typ := item.(type) {
	case string:
		return LengthString(fieldName, typ, lenght)
	default:
		return func() *Error {
			return &Error{
				Field: fieldName,
				Err:   "filed type doesn't have length rule",
				Code:  "1",
			}
		}
	}
}

func LengthString(fieldName, value string, length int) Rule {
	return func() *Error {
		if len(value) == length {
			return nil
		}

		return &Error{
			Field: fieldName,
			Err:   fmt.Sprintf("expected %d, got %d", length, len(value)),
		}
	}
}
