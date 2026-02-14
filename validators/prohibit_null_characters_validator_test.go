package validators

import "testing"

func TestProhibitNullCharactersValidator(t *testing.T) {
	testCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "string_with_null_character",
			value:   "hello\x00world",
			wantErr: true,
		},
		{
			name:    "string_without_null_character",
			value:   "hello world",
			wantErr: false,
		},
		{
			name:    "empty_string",
			value:   "",
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := prohibitNullCharactersValidator(tc.value)
			if (err != nil) != tc.wantErr {
				t.Errorf("ProhibitNullCharactersValidator() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
