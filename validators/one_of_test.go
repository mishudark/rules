package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestOneOf(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name    string
			value   string
			wantErr bool
			errCode string
		}{
			{name: "Allowed_Value", value: "admin", wantErr: false},
			{name: "Disallowed_Value", value: "superuser", wantErr: true, errCode: "VALUE_NOT_ALLOWED"},
			{name: "Case_Sensitive", value: "Admin", wantErr: true, errCode: "VALUE_NOT_ALLOWED"},
			{name: "Empty_String_Is_Valid", value: "", wantErr: false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				rule := OneOf("role", tc.value, []string{"admin", "user", "guest"})
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
	})

	t.Run("Int", func(t *testing.T) {
		t.Parallel()

		rule := OneOf("status", 2, []int{1, 2, 3})
		if err := rule.Validate(context.Background()); err != nil {
			t.Fatalf("error = %v, wantErr false", err)
		}

		err := OneOf("status", 9, []int{1, 2, 3}).Validate(context.Background())
		if err == nil {
			t.Fatal("error = nil, wantErr true")
		}
		if e, ok := err.(rules.Error); ok && e.Code != "VALUE_NOT_ALLOWED" {
			t.Errorf("errorCode = %s, want VALUE_NOT_ALLOWED", e.Code)
		}
	})
}
