package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestHexColor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{name: "Valid_Short", value: "#fff", wantErr: false},
		{name: "Valid_Short_Alpha", value: "#ffff", wantErr: false},
		{name: "Valid_Long", value: "#aabbcc", wantErr: false},
		{name: "Valid_Long_Alpha", value: "#AABBCCDD", wantErr: false},
		{name: "Empty_Is_Valid", value: "", wantErr: false},
		{name: "Missing_Hash", value: "aabbcc", wantErr: true, errCode: "INVALID_HEX_COLOR"},
		{name: "Too_Short", value: "#aabbc", wantErr: true, errCode: "INVALID_HEX_COLOR"},
		{name: "Too_Long", value: "#aabbccc", wantErr: true, errCode: "INVALID_HEX_COLOR"},
		{name: "Non_Hex", value: "#gggggg", wantErr: true, errCode: "INVALID_HEX_COLOR"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := HexColor("color", tc.value)
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
