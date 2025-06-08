package validators

import "testing"

func TestURLValidator(t *testing.T) {
	testCases := []struct {
		name    string
		value   string
		schemes []string
		wantErr bool
	}{
		{
			name:    "valid_http_url",
			value:   "http://example.com",
			schemes: []string{"http", "https"},
			wantErr: false,
		},
		{
			name:    "valid_https_url",
			value:   "https://example.com",
			schemes: []string{"http", "https"},
			wantErr: false,
		},
		{
			name:    "unsupported_scheme",
			value:   "ftp://example.com",
			schemes: []string{"http", "https"},
			wantErr: true,
		},
		{
			name:    "no_scheme_provided",
			value:   "example.com",
			schemes: []string{"http", "https"},
			wantErr: true,
		},
		{
			name:    "invalid_url",
			value:   "not a url",
			schemes: []string{"http", "https"},
			wantErr: true,
		},
		{
			name:    "empty_url",
			value:   "",
			schemes: []string{"http", "https"},
			wantErr: true,
		},
		{
			name:    "no_scheme_restriction",
			value:   "ftp://example.com",
			schemes: []string{},
			wantErr: false,
		},
		{
			name:    "case_insensitive_scheme",
			value:   "HTTP://example.com",
			schemes: []string{"http", "https"},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := URLValidator(tc.value, tc.schemes)
			if (err != nil) != tc.wantErr {
				t.Errorf("URLValidator() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
