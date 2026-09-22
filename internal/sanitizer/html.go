package sanitizer

import (
	"html"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var (
	// strictPolicy removes all HTML tags (for comments)
	strictPolicy = bluemonday.StrictPolicy()
)

// SanitizeComment removes all HTML tags and returns the remaining text as plain text.
// The result is stored as-is, so clients must render it as text (not HTML);
// XSS protection relies on that text rendering.
//
// bluemonday parses the input as HTML and escapes the text it keeps.
// To keep every character the user typed (including literal "&amp;"),
// "&" is escaped before sanitizing so it is read as text, not as the start of an entity,
// and the escaped output is converted back to plain text.
func SanitizeComment(content string) string {
	escapedAmpersands := strings.ReplaceAll(content, "&", "&amp;")
	return html.UnescapeString(strictPolicy.Sanitize(escapedAmpersands))
}
