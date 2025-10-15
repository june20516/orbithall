package sanitizer

import (
	"github.com/microcosm-cc/bluemonday"
)

var (
	// strictPolicy removes all HTML tags (for comments)
	strictPolicy = bluemonday.StrictPolicy()
)

// SanitizeComment sanitizes comment content by removing all HTML tags
// This prevents XSS attacks by only allowing plain text
func SanitizeComment(content string) string {
	return strictPolicy.Sanitize(content)
}
