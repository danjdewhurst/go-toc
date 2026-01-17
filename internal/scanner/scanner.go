package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/danjdewhurst/go-toc/internal/toc"
)

// Config holds the scanner configuration options.
type Config struct {
	RootPath       string   // Root directory to scan
	IgnorePatterns []string // Glob patterns to ignore
	UseGitignore   bool     // Whether to use .gitignore patterns
	MaxDepth       int      // Maximum recursion depth (0 = unlimited)
}

// Scanner handles recursive directory scanning for markdown files.
type Scanner struct {
	config        Config
	gitignoreMgr  *GitignoreManager
	ignoredByGlob map[string]bool // Cache for glob pattern matches
}

// New creates a new Scanner with the given configuration.
func New(config Config) *Scanner {
	s := &Scanner{
		config:        config,
		ignoredByGlob: make(map[string]bool),
	}

	if config.UseGitignore {
		s.gitignoreMgr = NewGitignoreManager(config.RootPath)
	}

	return s
}

// ScanResult contains both the tree and the list of markdown files.
type ScanResult struct {
	Tree  *toc.Tree
	Files []string // Absolute paths to markdown files
}

// Scan performs the directory scan and returns a tree of markdown files.
func (s *Scanner) Scan() (*toc.Tree, error) {
	result, err := s.ScanWithFiles()
	if err != nil {
		return nil, err
	}
	return result.Tree, nil
}

// GetMarkdownFiles returns a slice of all markdown file paths found.
func (s *Scanner) GetMarkdownFiles() ([]string, error) {
	result, err := s.ScanWithFiles()
	if err != nil {
		return nil, err
	}
	return result.Files, nil
}

// ScanWithFiles performs a single directory walk and returns both the tree
// and list of markdown files. This is more efficient than calling Scan()
// and GetMarkdownFiles() separately.
func (s *Scanner) ScanWithFiles() (*ScanResult, error) {
	tree := toc.NewTree(filepath.Base(s.config.RootPath))
	var files []string

	err := filepath.WalkDir(s.config.RootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(s.config.RootPath, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Calculate depth
		depth := strings.Count(relPath, string(os.PathSeparator)) + 1

		// Check max depth
		if s.config.MaxDepth > 0 && depth > s.config.MaxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Load nested .gitignore files as we traverse
		if d.IsDir() && s.gitignoreMgr != nil {
			s.gitignoreMgr.LoadGitignoreForDir(path)
		}

		// Check if path should be ignored
		if s.shouldIgnore(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Process entry
		if d.IsDir() {
			// Only add directory if it contains markdown files
			if s.containsMarkdown(path) {
				tree.AddDirectory(relPath)
			}
		} else if isMarkdownFile(path) {
			tree.AddFile(relPath)
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	tree.Sort()
	return &ScanResult{Tree: tree, Files: files}, nil
}

// shouldIgnore checks if a path should be ignored based on patterns.
func (s *Scanner) shouldIgnore(relPath string, isDir bool) bool {
	// Always ignore hidden files and directories (starting with .)
	name := filepath.Base(relPath)
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}

	// Check glob patterns
	for _, pattern := range s.config.IgnorePatterns {
		// Try matching the full path
		matched, err := filepath.Match(pattern, relPath)
		if err == nil && matched {
			return true
		}

		// Try matching just the name
		matched, err = filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}

		// For patterns with /, try matching from root
		if strings.Contains(pattern, "/") {
			// Normalize both to forward slashes for comparison
			normalizedPath := filepath.ToSlash(relPath)
			normalizedPattern := filepath.ToSlash(pattern)

			// Handle ** patterns
			if strings.Contains(normalizedPattern, "**") {
				if matchDoublestar(normalizedPattern, normalizedPath) {
					return true
				}
			}
		}
	}

	// Check gitignore patterns
	if s.gitignoreMgr != nil && s.gitignoreMgr.IsIgnored(relPath, isDir) {
		return true
	}

	return false
}

// containsMarkdown checks if a directory contains any markdown files.
func (s *Scanner) containsMarkdown(dirPath string) bool {
	hasMarkdown := false

	_ = filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(s.config.RootPath, path)
		if err != nil {
			return err
		}

		// Skip hidden and ignored
		if s.shouldIgnore(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() && isMarkdownFile(path) {
			hasMarkdown = true
			return filepath.SkipAll
		}

		return nil
	})

	return hasMarkdown
}

// isMarkdownFile checks if a file has a markdown extension.
func isMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

// matchDoublestar handles ** glob patterns.
func matchDoublestar(pattern, path string) bool {
	// Simple implementation for common cases
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return false
	}

	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	// Check prefix
	if prefix != "" && !strings.HasPrefix(path, prefix) {
		return false
	}

	// Check suffix
	if suffix != "" {
		remaining := path
		if prefix != "" {
			remaining = strings.TrimPrefix(path, prefix)
			remaining = strings.TrimPrefix(remaining, "/")
		}

		// Match suffix against the remaining path or any subpath
		if strings.HasSuffix(remaining, suffix) {
			return true
		}

		// Check if suffix matches filename
		matched, _ := filepath.Match(suffix, filepath.Base(path))
		return matched
	}

	return true
}
