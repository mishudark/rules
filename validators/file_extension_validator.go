package validators

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mishudark/rules"
)

// fileExtensionValidator validates that the given filename has an allowed extension.
func fileExtensionValidator(filename string, allowedExtensions []string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) > 0 {
		ext = ext[1:]
	}

	if ext == "" || !slices.ContainsFunc(allowedExtensions, func(e string) bool {
		return strings.EqualFold(e, ext)
	}) {
		return rules.Error{
			Code: "FILE_EXTENSION_NOT_ALLOWED",
			Err:  fmt.Sprintf("file extension '%s' is not allowed. allowed extensions are: %v", ext, allowedExtensions),
		}
	}

	return nil
}

// FileExtensionValidator returns a new rule that validates if a filename has an allowed extension.
// The check is case-insensitive.
func FileExtensionValidator(value string, allowedExtensions []string) rules.Rule {
	return rules.NewRulePure("fileExtensionValidator", func() error {
		return fileExtensionValidator(value, allowedExtensions)
	})
}
