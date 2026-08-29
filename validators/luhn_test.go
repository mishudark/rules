package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestLuhn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{name: "Valid_Visa", value: "4111111111111111", wantErr: false},
		{name: "Valid_Mastercard", value: "5555555555554444", wantErr: false},
		{name: "Valid_With_Separators", value: "4111 1111-1111-1111", wantErr: false},
		{name: "Empty_Is_Valid", value: "", wantErr: false},
		{name: "Failed_Checksum", value: "4111111111111112", wantErr: true, errCode: "INVALID_CARD_NUMBER"},
		{name: "Too_Short", value: "41111111111", wantErr: true, errCode: "INVALID_CARD_NUMBER"},
		{name: "Too_Long", value: "41111111111111111111", wantErr: true, errCode: "INVALID_CARD_NUMBER"},
		{name: "Non_Digit_Characters", value: "4111abcd11111111", wantErr: true, errCode: "INVALID_CARD_NUMBER"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := Luhn("card", tc.value)
			err := rule.Validate(context.Background())

			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				if e, ok := err.(rules.Error); ok && e.Code != tc.errCode {
					t.Errorf("errorCode = %s, want %s", e.Code, tc.errCode)
				}
			}
		})
	}
}
