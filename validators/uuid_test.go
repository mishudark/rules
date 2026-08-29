package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestUUID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{name: "Valid_V4", value: "123e4567-e89b-12d3-a456-426614174000", wantErr: false},
		{name: "Valid_Uppercase", value: "123E4567-E89B-12D3-A456-426614174000", wantErr: false},
		{name: "Empty_Is_Valid", value: "", wantErr: false},
		{name: "Missing_Dashes", value: "123e4567e89b12d3a456426614174000", wantErr: true, errCode: "INVALID_UUID"},
		{name: "Too_Short", value: "123e4567-e89b-12d3-a456-42661417400", wantErr: true, errCode: "INVALID_UUID"},
		{name: "Non_Hex", value: "123e4567-e89b-12d3-a456-42661417400g", wantErr: true, errCode: "INVALID_UUID"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := UUID("id", tc.value)
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
