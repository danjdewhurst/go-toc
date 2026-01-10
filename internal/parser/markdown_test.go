package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSummary(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		maxChars int
		expected string
	}{
		{
			name:     "simple paragraph",
			content:  "# Title\n\nThis is the first paragraph of the document.",
			maxChars: 100,
			expected: "This is the first paragraph of the document.",
		},
		{
			name:     "with frontmatter",
			content:  "---\ntitle: Test\ndate: 2024-01-01\n---\n\n# Heading\n\nActual content here.",
			maxChars: 100,
			expected: "Actual content here.",
		},
		{
			name:     "truncation",
			content:  "# Title\n\nThis is a very long paragraph that should be truncated because it exceeds the maximum character limit.",
			maxChars: 30,
			expected: "This is a very long paragraph...",
		},
		{
			name:     "with markdown formatting",
			content:  "# Title\n\nThis is **bold** and *italic* text with [links](http://example.com).",
			maxChars: 100,
			expected: "This is bold and italic text with links.",
		},
		{
			name:     "skip list items",
			content:  "# Title\n\n- list item\n- another item\n\nActual paragraph.",
			maxChars: 100,
			expected: "Actual paragraph.",
		},
		{
			name:     "empty file",
			content:  "",
			maxChars: 100,
			expected: "",
		},
		{
			name:     "only headings",
			content:  "# Title\n\n## Subtitle\n\n### Another heading",
			maxChars: 100,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test file
			filePath := filepath.Join(tmpDir, tt.name+".md")
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			result, err := ExtractSummary(filePath, tt.maxChars)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		text     string
		maxChars int
		expected string
	}{
		{"short text", 20, "short text"},
		{"this is longer text", 10, "this is..."},
		{"word boundary test here", 15, "word boundary..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.text, tt.maxChars)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d): expected %q, got %q", tt.text, tt.maxChars, tt.expected, result)
		}
	}
}

func TestCleanMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"**bold** text", "bold text"},
		{"*italic* text", "italic text"},
		{"`code` here", "here"},
		{"[link text](http://url)", "link text"},
		{"![alt text](image.png)", "!alt text"},
		{"  extra   spaces  ", "extra spaces"},
	}

	for _, tt := range tests {
		result := cleanMarkdown(tt.input)
		if result != tt.expected {
			t.Errorf("cleanMarkdown(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestIsHorizontalRule(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"---", true},
		{"***", true},
		{"___", true},
		{"- - -", true},
		{"--", false},
		{"text", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isHorizontalRule(tt.line)
		if result != tt.expected {
			t.Errorf("isHorizontalRule(%q): expected %v, got %v", tt.line, tt.expected, result)
		}
	}
}
