package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestPhoneE164(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{name: "Valid_US", value: "+14155552671", wantErr: false},
		{name: "Valid_DE", value: "+493083050", wantErr: false},
		{name: "Empty_Is_Valid", value: "", wantErr: false},
		{name: "Missing_Plus", value: "14155552671", wantErr: true, errCode: "INVALID_PHONE_FORMAT"},
		{name: "Leading_Zero_Country_Code", value: "+04155552671", wantErr: true, errCode: "INVALID_PHONE_FORMAT"},
		{name: "Contains_Separator", value: "+1 415 555 2671", wantErr: true, errCode: "INVALID_PHONE_FORMAT"},
		{name: "Too_Many_Digits", value: "+141555526711234567", wantErr: true, errCode: "INVALID_PHONE_FORMAT"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := PhoneE164("phone", tc.value)
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
