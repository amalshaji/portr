package utils

import "testing"

func TestTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "spaces on both sides",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "spaces on left",
			input:    "  hello world",
			expected: "hello world",
		},
		{
			name:     "spaces on right",
			input:    "hello world  ",
			expected: "hello world",
		},
		{
			name:     "tabs and newlines",
			input:    "\t\nhello world\t\n",
			expected: "hello world",
		},
		{
			name:     "mixed whitespace",
			input:    " \t \nhello world\n \t ",
			expected: "hello world",
		},
		{
			name:     "no spaces",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: "",
		},
		{
			name:     "only tabs and newlines",
			input:    "\t\n\r",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Trim(tt.input); got != tt.expected {
				t.Errorf("Trim() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic text",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "special characters",
			input:    "Hello! @World#$%",
			expected: "hello-world",
		},
		{
			name:     "multiple spaces",
			input:    "Hello   World",
			expected: "hello-world",
		},
		{
			name:     "leading and trailing hyphens",
			input:    "-hello world-",
			expected: "hello-world",
		},
		{
			name:     "multiple consecutive hyphens",
			input:    "hello---world",
			expected: "hello-world",
		},
		{
			name:     "mixed case with numbers",
			input:    "Hello World 123",
			expected: "hello-world-123",
		},
		{
			name:     "unicode characters",
			input:    "Héllö Wörld",
			expected: "hll-wrld",
		},
		{
			name:     "multiple special chars and spaces",
			input:    "  Hello!@#$%^&*()  World  ",
			expected: "hello-world",
		},
		{
			name:     "numbers and special chars",
			input:    "2023 - Hello & World!",
			expected: "2023-hello-world",
		},
		{
			name:     "dots and underscores",
			input:    "hello.world_example",
			expected: "helloworld-example",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "@#$%^&*",
			expected: "",
		},
		{
			name:     "only numbers",
			input:    "12345",
			expected: "12345",
		},
		{
			name:     "only hyphens",
			input:    "----",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.input); got != tt.expected {
				t.Errorf("Slugify() = %q, want %q", got, tt.expected)
			}
		})
	}
}
