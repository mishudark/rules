package validators

import (
	"context"
	"testing"

	"github.com/mishudark/rules"
)

func TestRuleValidEmail(t *testing.T) {
	testCases := []struct {
		name      string
		email     string
		allowlist []string
		wantErr   bool
		errCode   string
	}{
		{
			name:      "Valid_Email_No_Allowlist",
			email:     "test@example.com",
			allowlist: nil,
			wantErr:   false,
		},
		{
			name:      "Invalid_Email_Format",
			email:     "not-an-email",
			allowlist: nil,
			wantErr:   true,
			errCode:   "INVALID_EMAIL_FORMAT",
		},
		{
			name:      "Empty_Email_Is_Valid",
			email:     "",
			allowlist: nil,
			wantErr:   false,
		},
		{
			name:      "Domain_In_Allowlist",
			email:     "user@allowed.com",
			allowlist: []string{"allowed.com", "example.com"},
			wantErr:   false,
		},
		{
			name:      "Domain_Not_In_Allowlist",
			email:     "user@notallowed.com",
			allowlist: []string{"allowed.com", "example.com"},
			wantErr:   true,
			errCode:   "DOMAIN_NOT_ALLOWED",
		},
		{
			name:      "Allowlist_Is_Empty_Allows_Any_Domain",
			email:     "user@anydomain.com",
			allowlist: []string{},
			wantErr:   false,
		},
		{
			name:      "Allowlist_Case_Insensitive",
			email:     "user@ALLOWED.COM",
			allowlist: []string{"allowed.com"},
			wantErr:   false,
		},
		{
			name:      "Email_With_Name",
			email:     "Some User <user@example.com>",
			allowlist: []string{"example.com"},
			wantErr:   false,
		},
		{
			name:      "Email_With_Name_Domain_Not_Allowed",
			email:     "Some User <user@notallowed.com>",
			allowlist: []string{"example.com"},
			wantErr:   true,
			errCode:   "DOMAIN_NOT_ALLOWED",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rule := RuleValidEmail("email", tc.email, tc.allowlist)
			err := rule.Validate(context.Background())

			if (err != nil) != tc.wantErr {
				t.Errorf("RuleValidEmail() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if tc.wantErr {
				if e, ok := err.(rules.Error); ok {
					if e.Code != tc.errCode {
						t.Errorf("RuleValidEmail() errorCode = %s, wantCode %s", e.Code, tc.errCode)
					}
				} else {
					t.Errorf("Expected a custom Error type, but got %T", err)
				}
			}
		})
	}
}
