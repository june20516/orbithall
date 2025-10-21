package sanitizer

import "testing"

func TestSanitizeComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain text should remain unchanged",
			input:    "This is a plain text comment",
			expected: "This is a plain text comment",
		},
		{
			name:     "Remove script tags",
			input:    "Hello <script>alert('XSS')</script> World",
			expected: "Hello  World",
		},
		{
			name:     "Remove all HTML tags",
			input:    "<b>Bold</b> and <i>italic</i> text",
			expected: "Bold and italic text",
		},
		{
			name:     "Remove dangerous onclick attribute",
			input:    "<div onclick=\"alert('XSS')\">Click me</div>",
			expected: "Click me",
		},
		{
			name:     "Remove iframe tags",
			input:    "Check this <iframe src=\"http://evil.com\"></iframe>",
			expected: "Check this ",
		},
		{
			name:     "Preserve Korean text",
			input:    "안녕하세요 <strong>반갑습니다</strong>",
			expected: "안녕하세요 반갑습니다",
		},
		{
			name:     "Handle empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Remove nested tags",
			input:    "<div><span><b>Nested</b></span></div>",
			expected: "Nested",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeComment(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeComment(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
