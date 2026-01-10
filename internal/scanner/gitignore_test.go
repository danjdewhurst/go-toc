package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitignoreManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-gitignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .gitignore
	gitignoreContent := `
# Comments should be ignored
*.log
node_modules/
dist/
!important.log
/root-only.txt
docs/*.tmp
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewGitignoreManager(tmpDir)

	tests := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"error.log", false, true},
		{"nested/error.log", false, true},
		{"node_modules", true, true},
		{"node_modules/package/index.js", false, true},
		{"dist", true, true},
		{"dist/bundle.js", false, true},
		{"src/main.go", false, false},
		{"README.md", false, false},
		{"docs/temp.tmp", false, true},
		{"docs/guide.md", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := mgr.IsIgnored(tt.path, tt.isDir)
			if result != tt.expected {
				t.Errorf("IsIgnored(%q, %v): expected %v, got %v", tt.path, tt.isDir, tt.expected, result)
			}
		})
	}
}

func TestGitignoreManagerNoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-gitignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// No .gitignore file
	mgr := NewGitignoreManager(tmpDir)

	// Nothing should be ignored
	if mgr.IsIgnored("anything.log", false) {
		t.Error("without .gitignore, nothing should be ignored")
	}
}

func TestGitignoreManagerNestedGitignore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-gitignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create root .gitignore
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create nested directory with its own .gitignore
	nestedDir := filepath.Join(tmpDir, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, ".gitignore"), []byte("*.tmp\n"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewGitignoreManager(tmpDir)

	// Load nested gitignore
	mgr.LoadGitignoreForDir(nestedDir)

	// Root patterns should work
	if !mgr.IsIgnored("error.log", false) {
		t.Error("root .gitignore pattern should match")
	}

	// Nested patterns should work for nested paths
	if !mgr.IsIgnored("nested/temp.tmp", false) {
		t.Error("nested .gitignore pattern should match")
	}
}

func TestGitignoreManagerDirectoryPatterns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-gitignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .gitignore with directory-specific patterns
	gitignoreContent := `
build/
**/temp/
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewGitignoreManager(tmpDir)

	tests := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"build", true, true},
		{"build/output.txt", false, true},
		{"src/build", true, true},
		{"temp", true, true},
		{"src/temp", true, true},
		{"src/deep/temp", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := mgr.IsIgnored(tt.path, tt.isDir)
			if result != tt.expected {
				t.Errorf("IsIgnored(%q, %v): expected %v, got %v", tt.path, tt.isDir, tt.expected, result)
			}
		})
	}
}

func TestGitignoreManagerLoadGitignoreForDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-gitignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewGitignoreManager(tmpDir)

	// Initially no matchers (no .gitignore exists)
	if len(mgr.matchers) != 0 {
		t.Error("should have no matchers initially")
	}

	// Create .gitignore
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Load it
	mgr.LoadGitignoreForDir(tmpDir)

	if len(mgr.matchers) != 1 {
		t.Error("should have one matcher after loading")
	}

	// Loading again shouldn't duplicate
	mgr.LoadGitignoreForDir(tmpDir)
	if len(mgr.matchers) != 1 {
		t.Error("loading same dir twice shouldn't add duplicate matcher")
	}
}
