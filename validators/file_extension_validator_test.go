package validators

import "testing"

func TestFileExtensionValidator(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		value             string
		allowedExtensions []string
		wantErr           bool
	}{
		{
			name:              "valid_extension",
			value:             "document.pdf",
			allowedExtensions: []string{"pdf", "doc", "txt"},
			wantErr:           false,
		},
		{
			name:              "invalid_extension",
			value:             "document.exe",
			allowedExtensions: []string{"pdf", "doc", "txt"},
			wantErr:           true,
		},
		{
			name:              "case_insensitive",
			value:             "document.PDF",
			allowedExtensions: []string{"pdf", "doc"},
			wantErr:           false,
		},
		{
			name:              "no_extension",
			value:             "README",
			allowedExtensions: []string{"pdf", "txt"},
			wantErr:           true,
		},
		{
			name:              "empty_filename",
			value:             "",
			allowedExtensions: []string{"pdf"},
			wantErr:           true,
		},
		{
			name:              "file_without_dot",
			value:             "makefile",
			allowedExtensions: []string{"makefile"},
			wantErr:           true,
		},
		{
			name:              "multiple_dots",
			value:             "archive.tar.gz",
			allowedExtensions: []string{"gz"},
			wantErr:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := fileExtensionValidator(tc.value, tc.allowedExtensions)
			if (err != nil) != tc.wantErr {
				t.Errorf("fileExtensionValidator() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
