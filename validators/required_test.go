package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestRequired(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{name: "Non_Empty_Is_Valid", value: "hello", wantErr: false},
		{name: "Empty_Is_Invalid", value: "", wantErr: true, errCode: "REQUIRED"},
		{name: "Whitespace_Only_Is_Invalid", value: "   ", wantErr: true, errCode: "REQUIRED"},
		{name: "Padded_Value_Is_Valid", value: " x ", wantErr: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := Required("name", tc.value)
			err := rule.Validate(context.Background())

			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				e, ok := err.(rules.Error)
				if !ok {
					t.Fatalf("error type = %T, want rules.Error", err)
				}
				if e.Code != tc.errCode {
					t.Errorf("errorCode = %s, want %s", e.Code, tc.errCode)
				}
				if e.Field != "name" {
					t.Errorf("field = %s, want name", e.Field)
				}
			}
		})
	}
}

func TestRequiredSlice(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   []int
		wantErr bool
		errCode string
	}{
		{name: "Non_Empty_Is_Valid", value: []int{1}, wantErr: false},
		{name: "Nil_Is_Invalid", value: nil, wantErr: true, errCode: "REQUIRED"},
		{name: "Empty_Is_Invalid", value: []int{}, wantErr: true, errCode: "REQUIRED"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := RequiredSlice("items", tc.value)
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
