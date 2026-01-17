package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScannerBasic(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	createTestFile(t, tmpDir, "README.md", "# README")
	createTestDir(t, tmpDir, "docs")
	createTestFile(t, tmpDir, "docs/guide.md", "# Guide")
	createTestFile(t, tmpDir, "docs/api.md", "# API")

	config := Config{
		RootPath: tmpDir,
	}

	s := New(config)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Check tree structure
	if tree.Root == nil {
		t.Fatal("tree root is nil")
	}

	// Should have docs folder and README.md
	if len(tree.Root.Children) != 2 {
		t.Errorf("expected 2 children at root, got %d", len(tree.Root.Children))
	}
}

func TestScannerIgnorePatterns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	createTestFile(t, tmpDir, "README.md", "# README")
	createTestFile(t, tmpDir, "IGNORE.md", "# Ignore me")
	createTestDir(t, tmpDir, "vendor")
	createTestFile(t, tmpDir, "vendor/lib.md", "# Vendor lib")

	config := Config{
		RootPath:       tmpDir,
		IgnorePatterns: []string{"IGNORE.md", "vendor"},
	}

	s := New(config)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should only have README.md
	if len(tree.Root.Children) != 1 {
		t.Errorf("expected 1 child (README.md), got %d", len(tree.Root.Children))
	}
	if tree.Root.Children[0].Name != "README.md" {
		t.Errorf("expected README.md, got %s", tree.Root.Children[0].Name)
	}
}

func TestScannerMaxDepth(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	createTestFile(t, tmpDir, "root.md", "# Root")
	createTestDir(t, tmpDir, "level1")
	createTestFile(t, tmpDir, "level1/file1.md", "# Level 1")
	createTestDir(t, tmpDir, "level1/level2")
	createTestFile(t, tmpDir, "level1/level2/file2.md", "# Level 2")

	config := Config{
		RootPath: tmpDir,
		MaxDepth: 1,
	}

	s := New(config)
	files, err := s.GetMarkdownFiles()
	if err != nil {
		t.Fatalf("GetMarkdownFiles failed: %v", err)
	}

	// Should only have root.md (depth 1)
	if len(files) != 1 {
		t.Errorf("expected 1 file at depth 1, got %d", len(files))
	}
}

func TestScannerHiddenFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	createTestFile(t, tmpDir, "visible.md", "# Visible")
	createTestFile(t, tmpDir, ".hidden.md", "# Hidden")
	createTestDir(t, tmpDir, ".hidden-dir")
	createTestFile(t, tmpDir, ".hidden-dir/file.md", "# In hidden dir")

	config := Config{
		RootPath: tmpDir,
	}

	s := New(config)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should only have visible.md
	if len(tree.Root.Children) != 1 {
		t.Errorf("expected 1 child (visible.md), got %d", len(tree.Root.Children))
	}
}

func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"file.md", true},
		{"file.MD", true},
		{"file.markdown", true},
		{"file.txt", false},
		{"file.go", false},
		{"README", false},
	}

	for _, tt := range tests {
		result := isMarkdownFile(tt.path)
		if result != tt.expected {
			t.Errorf("isMarkdownFile(%q): expected %v, got %v", tt.path, tt.expected, result)
		}
	}
}

func TestMatchDoublestar(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		// Basic patterns
		{"**/*.md", "docs/file.md", true},
		{"**/*.md", "file.md", true},
		{"**/*.md", "a/b/c/file.md", true},
		{"**/*.go", "file.md", false},

		// Prefix patterns
		{"vendor/**", "vendor/lib/file.go", true},
		{"vendor/**", "vendor/file.go", true},
		{"vendor/**", "other/file.go", false},
		{"docs/**", "docs", false}, // ** requires at least one segment after

		// Combined prefix and suffix
		{"docs/**/*.md", "docs/api/ref.md", true},
		{"docs/**/*.md", "docs/api/v2/ref.md", true},
		{"docs/**/*.md", "docs/ref.md", true},
		{"docs/**/*.md", "docs/api/ref.go", false},
		{"docs/**/*.md", "other/api/ref.md", false},

		// Multiple directory levels
		{"src/**/*.go", "src/pkg/internal/file.go", true},
		{"**/test/**", "foo/test/bar", true},
		{"**/test/**", "test/bar", true},

		// Edge cases
		{"**", "anything/at/all", true},
		{"**", "file.txt", true},
		{"*.md", "file.md", true}, // No **, falls back to regular glob
		{"*.md", "dir/file.md", false},

		// Prefix boundary check
		{"docs/**", "documentation/file.md", false}, // "docs" is not a prefix of "documentation"
	}

	for _, tt := range tests {
		result := matchDoublestar(tt.pattern, tt.path)
		if result != tt.expected {
			t.Errorf("matchDoublestar(%q, %q): expected %v, got %v", tt.pattern, tt.path, tt.expected, result)
		}
	}
}

// Helper functions

func createTestFile(t *testing.T, base, path, content string) {
	t.Helper()
	fullPath := filepath.Join(base, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file %s: %v", fullPath, err)
	}
}

func createTestDir(t *testing.T, base, path string) {
	t.Helper()
	fullPath := filepath.Join(base, path)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", fullPath, err)
	}
}

func TestScannerWithGitignore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .gitignore
	createTestFile(t, tmpDir, ".gitignore", "ignored/\n*.tmp.md\n")

	// Create files
	createTestFile(t, tmpDir, "README.md", "# README")
	createTestFile(t, tmpDir, "temp.tmp.md", "# Temp")
	createTestDir(t, tmpDir, "ignored")
	createTestFile(t, tmpDir, "ignored/secret.md", "# Secret")
	createTestDir(t, tmpDir, "docs")
	createTestFile(t, tmpDir, "docs/guide.md", "# Guide")

	config := Config{
		RootPath:     tmpDir,
		UseGitignore: true,
	}

	s := New(config)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should have docs and README.md, but not ignored/ or temp.tmp.md
	// Simple check - count root children
	if len(tree.Root.Children) != 2 {
		t.Errorf("expected 2 children (docs, README.md), got %d", len(tree.Root.Children))
	}

	for _, child := range tree.Root.Children {
		if child.Name == "ignored" || child.Name == "temp.tmp.md" {
			t.Errorf("should not include ignored file/dir: %s", child.Name)
		}
	}
}

func TestScannerEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		RootPath: tmpDir,
	}

	s := New(config)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(tree.Root.Children) != 0 {
		t.Errorf("expected 0 children for empty dir, got %d", len(tree.Root.Children))
	}
}

func TestScannerNoMarkdownFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create only non-markdown files
	createTestFile(t, tmpDir, "main.go", "package main")
	createTestFile(t, tmpDir, "config.json", "{}")
	createTestDir(t, tmpDir, "src")
	createTestFile(t, tmpDir, "src/lib.go", "package lib")

	config := Config{
		RootPath: tmpDir,
	}

	s := New(config)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(tree.Root.Children) != 0 {
		t.Errorf("expected 0 children (no markdown), got %d", len(tree.Root.Children))
	}
}

func TestScannerDeepNesting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create deeply nested structure
	createTestFile(t, tmpDir, "a/b/c/d/e/deep.md", "# Deep")

	config := Config{
		RootPath: tmpDir,
	}

	s := New(config)
	files, err := s.GetMarkdownFiles()
	if err != nil {
		t.Fatalf("GetMarkdownFiles failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestScannerMaxDepthVariations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create structure at different depths
	createTestFile(t, tmpDir, "depth1.md", "# Depth 1")
	createTestFile(t, tmpDir, "a/depth2.md", "# Depth 2")
	createTestFile(t, tmpDir, "a/b/depth3.md", "# Depth 3")
	createTestFile(t, tmpDir, "a/b/c/depth4.md", "# Depth 4")

	tests := []struct {
		maxDepth int
		expected int
	}{
		{0, 4}, // unlimited
		{1, 1}, // only depth1.md
		{2, 2}, // depth1.md, depth2.md
		{3, 3}, // depth1.md, depth2.md, depth3.md
		{4, 4}, // all
		{10, 4}, // more than exists
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			config := Config{
				RootPath: tmpDir,
				MaxDepth: tt.maxDepth,
			}

			s := New(config)
			files, err := s.GetMarkdownFiles()
			if err != nil {
				t.Fatalf("GetMarkdownFiles failed: %v", err)
			}

			if len(files) != tt.expected {
				t.Errorf("maxDepth=%d: expected %d files, got %d", tt.maxDepth, tt.expected, len(files))
			}
		})
	}
}

func TestScannerMultipleIgnorePatterns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	createTestFile(t, tmpDir, "keep.md", "# Keep")
	createTestFile(t, tmpDir, "ignore1.md", "# Ignore 1")
	createTestFile(t, tmpDir, "ignore2.md", "# Ignore 2")
	createTestDir(t, tmpDir, "vendor")
	createTestFile(t, tmpDir, "vendor/lib.md", "# Lib")
	createTestDir(t, tmpDir, "node_modules")
	createTestFile(t, tmpDir, "node_modules/pkg.md", "# Pkg")

	config := Config{
		RootPath:       tmpDir,
		IgnorePatterns: []string{"ignore1.md", "ignore2.md", "vendor", "node_modules"},
	}

	s := New(config)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(tree.Root.Children) != 1 {
		t.Errorf("expected 1 child (keep.md), got %d", len(tree.Root.Children))
	}
	if tree.Root.Children[0].Name != "keep.md" {
		t.Errorf("expected keep.md, got %s", tree.Root.Children[0].Name)
	}
}

func TestScannerMarkdownExtension(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Use unique filenames to avoid case-sensitivity issues on macOS
	createTestFile(t, tmpDir, "file1.md", "# MD")
	createTestFile(t, tmpDir, "file2.markdown", "# Markdown")

	config := Config{
		RootPath: tmpDir,
	}

	s := New(config)
	files, err := s.GetMarkdownFiles()
	if err != nil {
		t.Fatalf("GetMarkdownFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 markdown files, got %d", len(files))
	}
}

func TestContainsMarkdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Directory with markdown
	createTestDir(t, tmpDir, "with-md")
	createTestFile(t, tmpDir, "with-md/doc.md", "# Doc")

	// Directory without markdown
	createTestDir(t, tmpDir, "no-md")
	createTestFile(t, tmpDir, "no-md/main.go", "package main")

	// Empty directory
	createTestDir(t, tmpDir, "empty")

	config := Config{
		RootPath: tmpDir,
	}

	s := New(config)

	if !s.containsMarkdown(filepath.Join(tmpDir, "with-md")) {
		t.Error("with-md should contain markdown")
	}

	if s.containsMarkdown(filepath.Join(tmpDir, "no-md")) {
		t.Error("no-md should not contain markdown")
	}

	if s.containsMarkdown(filepath.Join(tmpDir, "empty")) {
		t.Error("empty should not contain markdown")
	}
}

func TestScannerNestedGitignore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create root .gitignore that ignores *.log files
	createTestFile(t, tmpDir, ".gitignore", "*.log")

	// Create a subdirectory with its own .gitignore
	createTestDir(t, tmpDir, "subdir")
	createTestFile(t, tmpDir, "subdir/.gitignore", "ignored.md")

	// Create test files
	createTestFile(t, tmpDir, "README.md", "# Root readme")
	createTestFile(t, tmpDir, "subdir/included.md", "# Included")
	createTestFile(t, tmpDir, "subdir/ignored.md", "# Should be ignored by nested gitignore")

	config := Config{
		RootPath:     tmpDir,
		UseGitignore: true,
	}

	s := New(config)
	result, err := s.ScanWithFiles()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Check that ignored.md was not included
	for _, f := range result.Files {
		if filepath.Base(f) == "ignored.md" {
			t.Error("ignored.md should have been filtered by nested .gitignore")
		}
	}

	// Check that included.md was included
	foundIncluded := false
	for _, f := range result.Files {
		if filepath.Base(f) == "included.md" {
			foundIncluded = true
			break
		}
	}
	if !foundIncluded {
		t.Error("included.md should have been found")
	}
}
