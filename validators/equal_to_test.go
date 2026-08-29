package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestEqualTo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{name: "Matches", value: "secret123", wantErr: false},
		{name: "Mismatch", value: "other", wantErr: true, errCode: "VALUE_MISMATCH"},
		{name: "Case_Sensitive_Mismatch", value: "Secret123", wantErr: true, errCode: "VALUE_MISMATCH"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := EqualTo("passwordConfirm", tc.value, "secret123")
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

func TestEqualTo_Empty_Expected(t *testing.T) {
	t.Parallel()

	if err := EqualTo("field", "", "").Validate(context.Background()); err != nil {
		t.Fatalf("error = %v, wantErr false", err)
	}
}

func TestBetween(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   float64
		min     float64
		max     float64
		wantErr bool
		errCode string
	}{
		{name: "In_Range", value: 5, min: 1, max: 10, wantErr: false},
		{name: "At_Min_Boundary", value: 1, min: 1, max: 10, wantErr: false},
		{name: "At_Max_Boundary", value: 10, min: 1, max: 10, wantErr: false},
		{name: "Below_Min", value: 0, min: 1, max: 10, wantErr: true, errCode: "VALUE_OUT_OF_RANGE"},
		{name: "Above_Max", value: 11, min: 1, max: 10, wantErr: true, errCode: "VALUE_OUT_OF_RANGE"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := Between("score", tc.value, tc.min, tc.max)
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
