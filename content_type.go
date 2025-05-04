package rules

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// RuleValidContentType checks if the content type detected by http.DetectContentType
// matches one of the allowed MIME types.
type RuleValidContentType struct {
	RuleBase               // Embed for execution path handling
	fieldName    string    // Name of the field being validated (e.g., "UploadedFile")
	reader       io.Reader // Reader providing the file content
	allowedMIMEs []string  // List of allowed MIME types (e.g., ["image/jpeg", "application/pdf"])
}

// Ensure RuleValidContentType implements the Rule interface.
var _ Rule = (*RuleValidContentType)(nil)

// NewRuleContentType creates a new instance of the content type validation rule.
// allowedMIMEs should be standard MIME type strings.
func NewRuleContentType(fieldName string, reader io.Reader, allowedMIMEs []string) Rule {
	// Normalize allowed MIME types to lowercase for case-insensitive comparison
	normalizedMIMEs := make([]string, len(allowedMIMEs))
	for i, mime := range allowedMIMEs {
		normalizedMIMEs[i] = strings.ToLower(strings.TrimSpace(mime))
	}

	return &RuleValidContentType{
		fieldName:    fieldName,
		reader:       reader,
		allowedMIMEs: normalizedMIMEs,
	}
}

// Name returns the name of the rule.
func (r *RuleValidContentType) Name() string {
	return fmt.Sprintf("RuleValidContentType[%s, mimes=%v]", r.fieldName, r.allowedMIMEs)
}

// Prepare checks if the reader is nil.
func (r *RuleValidContentType) Prepare(ctx context.Context) error {
	if r.reader == nil {
		return fmt.Errorf("reader for rule '%s' is nil", r.Name())
	}
	return nil
}

// Validate performs the content type detection and check.
func (r *RuleValidContentType) Validate(ctx context.Context) error {
	// http.DetectContentType requires sniffing the first 512 bytes.
	const sniffLen = 512
	buffer := make([]byte, sniffLen)

	// Read up to sniffLen bytes.
	// We use io.ReadAtLeast to ensure we read *something* if the file isn't empty,
	// but it's okay if the file is smaller than sniffLen.
	n, err := io.ReadAtLeast(r.reader, buffer, 1) // Read at least 1 byte

	// Handle errors during reading
	if err != nil {
		// If the error is EOF and we read 0 bytes, it's an empty file.
		if err == io.EOF && n == 0 {
			// If no MIME types are specified as allowed, an empty file might be okay.
			if len(r.allowedMIMEs) == 0 {
				return nil
			}
			// Otherwise, an empty file cannot match any specific MIME type.
			return Error{
				Field: r.fieldName,
				Err:   "File is empty",
				Code:  "CONTENT_TYPE_EMPTY_FILE",
			}
		}
		// If it's EOF or UnexpectedEOF but we *did* read some bytes (n > 0),
		// that's fine, we just use the bytes we got.
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			// Any other error is a problem.
			return Error{
				Field: r.fieldName,
				Err:   fmt.Sprintf("Failed to read file content for content type detection: %v", err),
				Code:  "CONTENT_TYPE_READ_ERROR",
			}
		}
		// If we fall through here, it means err was EOF/UnexpectedEOF but n > 0.
	}

	// Use the actual bytes read (up to sniffLen)
	dataToSniff := buffer[:n]

	// Detect the content type using the standard library function.
	detectedContentType := http.DetectContentType(dataToSniff)
	// DetectContentType returns format "type/subtype; param=value", we often only care about "type/subtype"
	// Split on ";" and take the first part, converting to lowercase for comparison.
	mimeOnly := strings.ToLower(strings.SplitN(detectedContentType, ";", 2)[0])

	// If no specific MIME types are required, any detected type is acceptable.
	if len(r.allowedMIMEs) == 0 {
		return nil
	}

	// Check if the detected MIME type is in the allowed list.
	if slices.Contains(r.allowedMIMEs, mimeOnly) {
		// Found a match! Validation passes.
		return nil
	}

	// If no match was found in the allowed list.
	return Error{
		Field: r.fieldName,
		Err:   fmt.Sprintf("Detected content type '%s' is not in the allowed list: %v", detectedContentType, r.allowedMIMEs),
		Code:  "CONTENT_TYPE_MISMATCH",
	}
}
