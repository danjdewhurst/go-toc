package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	// Create temp directory with test files
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		wantContain []string
	}{
		{
			name:        "basic scan",
			args:        []string{tmpDir},
			wantErr:     false,
			wantContain: []string{"Table of Contents", "README.md", "guide.md"},
		},
		{
			name:        "with summary",
			args:        []string{tmpDir, "--summary"},
			wantErr:     false,
			wantContain: []string{"This is the main readme", "Getting started guide"},
		},
		{
			name:        "custom title",
			args:        []string{tmpDir, "--title", "My Docs"},
			wantErr:     false,
			wantContain: []string{"# My Docs"},
		},
		{
			name:        "max depth 1",
			args:        []string{tmpDir, "--max-depth", "1"},
			wantErr:     false,
			wantContain: []string{"README.md"},
		},
		{
			name:        "ignore pattern",
			args:        []string{tmpDir, "--ignore", "docs/*"},
			wantErr:     false,
			wantContain: []string{"README.md"},
		},
		{
			name:    "invalid directory",
			args:    []string{"/nonexistent/path"},
			wantErr: true,
		},
		{
			name:    "file instead of directory",
			args:    []string{filepath.Join(tmpDir, "README.md")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags to defaults
			resetFlags()

			// Capture output
			var stdout bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stdout)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			output := stdout.String()
			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

func TestOutputToFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	outputFile := filepath.Join(tmpDir, "output.md")

	resetFlags()
	rootCmd.SetArgs([]string{tmpDir, "--output", outputFile})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "Table of Contents") {
		t.Error("output file should contain Table of Contents")
	}
}

func TestSummaryChars(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	resetFlags()

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{tmpDir, "--summary", "--summary-chars", "20"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	// With 20 char limit, summaries should be truncated
	if strings.Contains(output, "This is the main readme file") {
		t.Error("summary should be truncated")
	}
}

func TestSingleThreaded(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	resetFlags()

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{tmpDir, "--summary", "--single-threaded"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce same output as multi-threaded
	if !strings.Contains(stdout.String(), "Table of Contents") {
		t.Error("single-threaded mode should produce valid output")
	}
}

func TestGitignoreFlag(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	// Create .gitignore
	gitignore := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("ignored/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create ignored directory with markdown
	ignoredDir := filepath.Join(tmpDir, "ignored")
	if err := os.MkdirAll(ignoredDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredDir, "secret.md"), []byte("# Secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// Without gitignore flag - should include ignored dir
	resetFlags()
	var stdout1 bytes.Buffer
	rootCmd.SetOut(&stdout1)
	rootCmd.SetErr(&stdout1)
	rootCmd.SetArgs([]string{tmpDir})
	_ = rootCmd.Execute()

	// With gitignore flag - should exclude ignored dir
	resetFlags()
	var stdout2 bytes.Buffer
	rootCmd.SetOut(&stdout2)
	rootCmd.SetErr(&stdout2)
	rootCmd.SetArgs([]string{tmpDir, "--gitignore"})
	_ = rootCmd.Execute()

	if !strings.Contains(stdout1.String(), "secret.md") {
		t.Error("without --gitignore, ignored files should be included")
	}

	if strings.Contains(stdout2.String(), "secret.md") {
		t.Error("with --gitignore, ignored files should be excluded")
	}
}

func TestDefaultDirectory(t *testing.T) {
	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	// Change to test directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	resetFlags()
	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{}) // No directory argument

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout.String(), "README.md") {
		t.Error("should scan current directory by default")
	}
}

// Helper functions

func setupTestDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "go-toc-cmd-test")
	if err != nil {
		t.Fatal(err)
	}

	// Create test structure
	files := map[string]string{
		"README.md":          "# README\n\nThis is the main readme file for the project.",
		"docs/guide.md":      "# Guide\n\nGetting started guide for new users.",
		"docs/api/handlers.md": "# Handlers\n\nAPI handler documentation.",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

func resetFlags() {
	// Reset all flags to default values
	ignorePatterns = []string{}
	useGitignore = false
	maxDepth = 0
	includeSummary = false
	summaryChars = 100
	singleThreaded = false
	outputFile = ""
	title = "Table of Contents"
	fancy = false
}
