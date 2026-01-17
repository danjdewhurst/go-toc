package toc

import (
	"strings"
	"testing"
)

func TestGenerator(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.AddFile("docs/guide.md")
	tree.Sort()

	gen := NewGenerator(GeneratorConfig{
		Title: "Test ToC",
	})

	output := gen.Generate(tree)

	// Check title
	if !strings.Contains(output, "# Test ToC") {
		t.Error("output should contain title")
	}

	// Check file links
	if !strings.Contains(output, "[README.md](README.md)") {
		t.Error("output should contain README link")
	}
	if !strings.Contains(output, "[guide.md](docs/guide.md)") {
		t.Error("output should contain guide link")
	}

	// Check tree characters
	if !strings.Contains(output, "├──") || !strings.Contains(output, "└──") {
		t.Error("output should contain tree characters")
	}
}

func TestGeneratorWithSummary(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.Sort()

	summaries := map[string]string{
		"README.md": "This is the project overview.",
	}

	gen := NewGenerator(GeneratorConfig{
		Title:          "Test ToC",
		IncludeSummary: true,
		Summaries:      summaries,
	})

	output := gen.Generate(tree)

	// Check summary is included
	if !strings.Contains(output, "> This is the project overview.") {
		t.Error("output should contain summary as blockquote")
	}
}

func TestGeneratorDirectories(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("docs/api/handlers.md")
	tree.AddFile("docs/api/routes.md")
	tree.AddFile("docs/guide.md")
	tree.Sort()

	gen := NewGenerator(GeneratorConfig{
		Title: "Test ToC",
	})

	output := gen.Generate(tree)

	// Check directory is shown with trailing slash
	if !strings.Contains(output, "docs/") {
		t.Error("output should show directory with trailing slash")
	}
	if !strings.Contains(output, "api/") {
		t.Error("output should show nested directory")
	}
}

func TestGeneratorFancy(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.AddFile("docs/guide.md")
	tree.Sort()

	gen := NewGenerator(GeneratorConfig{
		Title: "Test ToC",
		Fancy: true,
	})

	output := gen.Generate(tree)

	// Check title has emoji
	if !strings.Contains(output, "# Test ToC 📚") {
		t.Error("fancy output should contain title with emoji")
	}

	// Check folder emoji
	if !strings.Contains(output, "📁") {
		t.Error("fancy output should contain folder emoji")
	}

	// Check file emoji
	if !strings.Contains(output, "📄") {
		t.Error("fancy output should contain file emoji")
	}

	// Check file links still work
	if !strings.Contains(output, "[README.md](README.md)") {
		t.Error("fancy output should contain README link")
	}

	// Should NOT contain ASCII tree characters
	if strings.Contains(output, "├──") || strings.Contains(output, "└──") {
		t.Error("fancy output should not contain ASCII tree characters")
	}
}

func TestGeneratorFancyWithSummary(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.Sort()

	summaries := map[string]string{
		"README.md": "This is the project overview.",
	}

	gen := NewGenerator(GeneratorConfig{
		Title:          "Test ToC",
		Fancy:          true,
		IncludeSummary: true,
		Summaries:      summaries,
	})

	output := gen.Generate(tree)

	// Check summary has speech bubble emoji
	if !strings.Contains(output, "> 💬 This is the project overview.") {
		t.Error("fancy output should contain summary with emoji")
	}
}

func TestGetStats(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.AddFile("docs/guide.md")
	tree.AddFile("docs/api/handlers.md")
	tree.Sort()

	stats := GetStats(tree)

	if stats.TotalFiles != 3 {
		t.Errorf("expected 3 files, got %d", stats.TotalFiles)
	}
	if stats.TotalDirectories != 2 {
		t.Errorf("expected 2 directories, got %d", stats.TotalDirectories)
	}
	if stats.MaxDepth != 3 {
		t.Errorf("expected max depth 3, got %d", stats.MaxDepth)
	}
}

func TestFormatStats(t *testing.T) {
	stats := Stats{
		TotalFiles:       5,
		TotalDirectories: 2,
		MaxDepth:         3,
	}

	output := FormatStats(stats)

	if output != "5 files, 2 directories, max depth 3" {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestGeneratorWithAnchors(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.AddFile("docs/API Guide.md")
	tree.Sort()

	gen := NewGenerator(GeneratorConfig{
		Title:           "Test ToC",
		GenerateAnchors: true,
	})

	output := gen.Generate(tree)

	// Check anchors are generated
	if !strings.Contains(output, `<a id="readme"></a>`) {
		t.Error("output should contain anchor for README")
	}
	if !strings.Contains(output, `<a id="docs-api-guide"></a>`) {
		t.Error("output should contain anchor for docs/API Guide.md")
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"README.md", "readme"},
		{"docs/guide.md", "docs-guide"},
		{"docs/API Guide.md", "docs-api-guide"},
		{"my_file.md", "my-file"},
		{"path/to/file.md", "path-to-file"},
		{"UPPERCASE.md", "uppercase"},
		{"file-with-dashes.md", "file-with-dashes"},
		{"file...dots.md", "file-dots"},
	}

	for _, tt := range tests {
		result := generateSlug(tt.input)
		if result != tt.expected {
			t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSetSummary(t *testing.T) {
	gen := NewGenerator(GeneratorConfig{
		Title:          "Test",
		IncludeSummary: true,
	})

	// Set a summary
	gen.SetSummary("README.md", "This is the readme summary.")

	// Verify it was set
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.Sort()

	output := gen.Generate(tree)

	if !strings.Contains(output, "This is the readme summary.") {
		t.Error("output should contain the set summary")
	}
}

func TestSetSummaryOverwrite(t *testing.T) {
	gen := NewGenerator(GeneratorConfig{
		Title:          "Test",
		IncludeSummary: true,
	})

	// Set a summary then overwrite it
	gen.SetSummary("README.md", "Original summary")
	gen.SetSummary("README.md", "Updated summary")

	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.Sort()

	output := gen.Generate(tree)

	if strings.Contains(output, "Original summary") {
		t.Error("output should not contain original summary after overwrite")
	}
	if !strings.Contains(output, "Updated summary") {
		t.Error("output should contain updated summary")
	}
}

func TestFormatTree(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.AddFile("docs/guide.md")
	tree.Sort()

	gen := NewGenerator(GeneratorConfig{
		Title: "My Custom Title",
	})

	output := gen.FormatTree(tree)

	// Should NOT contain the title
	if strings.Contains(output, "# My Custom Title") {
		t.Error("FormatTree output should not contain title")
	}

	// Should contain the file links
	if !strings.Contains(output, "[README.md](README.md)") {
		t.Error("FormatTree output should contain README link")
	}
	if !strings.Contains(output, "[guide.md](docs/guide.md)") {
		t.Error("FormatTree output should contain guide link")
	}
}

func TestFormatTreeFancy(t *testing.T) {
	tree := NewTree("project")
	tree.AddFile("README.md")
	tree.Sort()

	gen := NewGenerator(GeneratorConfig{
		Title: "Fancy Title",
		Fancy: true,
	})

	output := gen.FormatTree(tree)

	// Should NOT contain the title with emoji
	if strings.Contains(output, "# Fancy Title 📚") {
		t.Error("FormatTree output should not contain title")
	}

	// Should contain the file emoji and link
	if !strings.Contains(output, "📄") {
		t.Error("FormatTree fancy output should contain file emoji")
	}
}
