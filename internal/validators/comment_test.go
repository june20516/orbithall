package validators

import (
	"strings"
	"testing"
)

func TestValidateCommentCreate(t *testing.T) {
	tests := []struct {
		name          string
		input         CommentCreateInput
		expectError   bool
		expectedField string
	}{
		{
			name: "Valid input",
			input: CommentCreateInput{
				AuthorName: "John Doe",
				Password:   "password123",
				Content:    "This is a test comment",
				ParentID:   nil,
			},
			expectError: false,
		},
		{
			name: "Empty author name",
			input: CommentCreateInput{
				AuthorName: "",
				Password:   "password123",
				Content:    "This is a test comment",
			},
			expectError:   true,
			expectedField: "author_name",
		},
		{
			name: "Author name too long (>100 chars)",
			input: CommentCreateInput{
				AuthorName: strings.Repeat("a", 101),
				Password:   "password123",
				Content:    "This is a test comment",
			},
			expectError:   true,
			expectedField: "author_name",
		},
		{
			name: "Author name with only spaces",
			input: CommentCreateInput{
				AuthorName: "   ",
				Password:   "password123",
				Content:    "This is a test comment",
			},
			expectError:   true,
			expectedField: "author_name",
		},
		{
			name: "Password too short (<4 chars)",
			input: CommentCreateInput{
				AuthorName: "John Doe",
				Password:   "abc",
				Content:    "This is a test comment",
			},
			expectError:   true,
			expectedField: "password",
		},
		{
			name: "Password too long (>50 chars)",
			input: CommentCreateInput{
				AuthorName: "John Doe",
				Password:   strings.Repeat("a", 51),
				Content:    "This is a test comment",
			},
			expectError:   true,
			expectedField: "password",
		},
		{
			name: "Empty content",
			input: CommentCreateInput{
				AuthorName: "John Doe",
				Password:   "password123",
				Content:    "",
			},
			expectError:   true,
			expectedField: "content",
		},
		{
			name: "Content too long (>10000 chars)",
			input: CommentCreateInput{
				AuthorName: "John Doe",
				Password:   "password123",
				Content:    strings.Repeat("a", 10001),
			},
			expectError:   true,
			expectedField: "content",
		},
		{
			name: "Valid with parent_id",
			input: CommentCreateInput{
				AuthorName: "John Doe",
				Password:   "password123",
				Content:    "This is a reply",
				ParentID:   intPtr(1),
			},
			expectError: false,
		},
		{
			name: "Invalid parent_id (zero)",
			input: CommentCreateInput{
				AuthorName: "John Doe",
				Password:   "password123",
				Content:    "This is a reply",
				ParentID:   intPtr(0),
			},
			expectError:   true,
			expectedField: "parent_id",
		},
		{
			name: "Invalid parent_id (negative)",
			input: CommentCreateInput{
				AuthorName: "John Doe",
				Password:   "password123",
				Content:    "This is a reply",
				ParentID:   intPtr(-1),
			},
			expectError:   true,
			expectedField: "parent_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				valErr, ok := err.(ValidationErrors)
				if !ok {
					t.Errorf("Expected ValidationErrors but got %T", err)
					return
				}
				if _, exists := valErr[tt.expectedField]; !exists {
					t.Errorf("Expected error for field %q but got errors: %v", tt.expectedField, valErr)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateCommentUpdate(t *testing.T) {
	tests := []struct {
		name          string
		input         CommentUpdateInput
		expectError   bool
		expectedField string
	}{
		{
			name: "Valid input",
			input: CommentUpdateInput{
				Password: "password123",
				Content:  "Updated content",
			},
			expectError: false,
		},
		{
			name: "Empty password",
			input: CommentUpdateInput{
				Password: "",
				Content:  "Updated content",
			},
			expectError:   true,
			expectedField: "password",
		},
		{
			name: "Empty content",
			input: CommentUpdateInput{
				Password: "password123",
				Content:  "",
			},
			expectError:   true,
			expectedField: "content",
		},
		{
			name: "Content too long",
			input: CommentUpdateInput{
				Password: "password123",
				Content:  strings.Repeat("a", 10001),
			},
			expectError:   true,
			expectedField: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				valErr, ok := err.(ValidationErrors)
				if !ok {
					t.Errorf("Expected ValidationErrors but got %T", err)
					return
				}
				if _, exists := valErr[tt.expectedField]; !exists {
					t.Errorf("Expected error for field %q but got errors: %v", tt.expectedField, valErr)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateCommentDelete(t *testing.T) {
	tests := []struct {
		name          string
		input         CommentDeleteInput
		expectError   bool
		expectedField string
	}{
		{
			name: "Valid input",
			input: CommentDeleteInput{
				Password: "password123",
			},
			expectError: false,
		},
		{
			name: "Empty password",
			input: CommentDeleteInput{
				Password: "",
			},
			expectError:   true,
			expectedField: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				valErr, ok := err.(ValidationErrors)
				if !ok {
					t.Errorf("Expected ValidationErrors but got %T", err)
					return
				}
				if _, exists := valErr[tt.expectedField]; !exists {
					t.Errorf("Expected error for field %q but got errors: %v", tt.expectedField, valErr)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
