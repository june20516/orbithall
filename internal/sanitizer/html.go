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

// maxSanitizePasses limits how many times tag removal is repeated.
// Normal input settles within two passes.
const maxSanitizePasses = 5

// SanitizeComment removes all HTML tags and returns the remaining text as plain text.
// The result is stored as-is, so clients must render it as text (not HTML);
// XSS protection relies on that text rendering.
//
// Removing a tag can join the text around it into a new tag
// (e.g. "<<b>img ...>" becomes "<img ...>"), so removal is repeated until the result
// stops changing. If it keeps changing after maxSanitizePasses, an empty string is
// returned so that validation rejects the input.
func SanitizeComment(content string) string {
	for range maxSanitizePasses {
		sanitized := removeTags(content)
		if sanitized == content {
			return sanitized
		}
		content = sanitized
	}
	return ""
}

// removeTags removes HTML tags once and keeps every other character as typed.
//
// bluemonday parses the input as HTML and escapes the text it keeps.
// To keep every character the user typed (including literal "&amp;"),
// "&" is escaped before sanitizing so it is read as text, not as the start of an entity,
// and the escaped output is converted back to plain text.
func removeTags(content string) string {
	escapedAmpersands := strings.ReplaceAll(content, "&", "&amp;")
	return html.UnescapeString(strictPolicy.Sanitize(escapedAmpersands))
}
