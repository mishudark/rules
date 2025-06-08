package validators

import (
	"testing"
)

func TestValidateCommaSeparatedIntegerList(t *testing.T) {
	testCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid_list",
			value:   "1,2,3,4",
			wantErr: false,
		},
		{
			name:    "valid_list_with_spaces",
			value:   " 1, 2 , 3, 4 ",
			wantErr: false,
		},
		{
			name:    "single_integer",
			value:   "42",
			wantErr: false,
		},
		{
			name:    "list_with_non_integer",
			value:   "1,a,3",
			wantErr: true,
		},
		{
			name:    "list_with_trailing_comma",
			value:   "1,2,3,",
			wantErr: true,
		},
		{
			name:    "list_with_leading_comma",
			value:   ",1,2,3",
			wantErr: true,
		},
		{
			name:    "list_with_empty_part",
			value:   "1,,3",
			wantErr: true,
		},
		{
			name:    "empty_string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "string_with_only_comma",
			value:   ",",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCommaSeparatedIntegerList(tc.value)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateCommaSeparatedIntegerList() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
