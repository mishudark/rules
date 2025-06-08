package validators

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mishudark/rules"
)

// fileExtensionValidator validates that the given filename has an allowed extension.
func fileExtensionValidator(filename string, allowedExtensions []string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) > 0 {
		// Remove the leading dot.
		ext = ext[1:]
	}

	for _, allowedExt := range allowedExtensions {
		if strings.ToLower(allowedExt) == ext {
			return nil
		}
	}

	return fmt.Errorf("file extension '%s' is not allowed. Allowed extensions are: %v", ext, allowedExtensions)
}

// NewFileExtensionValidator returns a new rule that validates if a filename has an allowed extension.
// The check is case-insensitive.
func NewFileExtensionValidator(value string, allowedExtensions []string) rules.Rule {
	return rules.NewRulePure("fileExtensionValidator", func() error {
		return fileExtensionValidator(value, allowedExtensions)
	})
}
