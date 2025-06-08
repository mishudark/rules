package validators

import "testing"

func TestDecimalValidator(t *testing.T) {
	testCases := []struct {
		name          string
		value         string
		maxDigits     int
		decimalPlaces int
		wantErr       bool
	}{
		{
			name:          "valid_decimal",
			value:         "123.45",
			maxDigits:     5,
			decimalPlaces: 2,
			wantErr:       false,
		},
		{
			name:          "valid_integer",
			value:         "123",
			maxDigits:     5,
			decimalPlaces: 2,
			wantErr:       false,
		},
		{
			name:          "exceeds_max_digits",
			value:         "12345.67",
			maxDigits:     6,
			decimalPlaces: 2,
			wantErr:       true,
		},
		{
			name:          "exceeds_decimal_places",
			value:         "123.456",
			maxDigits:     6,
			decimalPlaces: 2,
			wantErr:       true,
		},
		{
			name:          "exceeds_integer_digits",
			value:         "12345.6",
			maxDigits:     6,
			decimalPlaces: 2,
			wantErr:       true,
		},
		{
			name:          "negative_valid_decimal",
			value:         "-123.45",
			maxDigits:     5,
			decimalPlaces: 2,
			wantErr:       false,
		},
		{
			name:          "invalid_format",
			value:         "1.2.3",
			maxDigits:     5,
			decimalPlaces: 2,
			wantErr:       true,
		},
		{
			name:          "empty_value",
			value:         "",
			maxDigits:     5,
			decimalPlaces: 2,
			wantErr:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := DecimalValidator(tc.value, tc.maxDigits, tc.decimalPlaces)
			if (err != nil) != tc.wantErr {
				t.Errorf("DecimalValidator() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
