package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestMatchesRegex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{name: "Matches", value: "hello", wantErr: false},
		{name: "Does_Not_Match", value: "Hello", wantErr: true, errCode: "PATTERN_MISMATCH"},
		{name: "Empty_Is_Valid", value: "", wantErr: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := MatchesRegex("code", tc.value, `^[a-z]+$`)
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
