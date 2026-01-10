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
