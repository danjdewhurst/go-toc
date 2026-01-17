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
	Tree     *toc.Tree
	Files    []string // Relative paths to markdown files
	RootPath string   // Absolute path to root directory
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
// Returns absolute paths for backward compatibility.
func (s *Scanner) GetMarkdownFiles() ([]string, error) {
	result, err := s.ScanWithFiles()
	if err != nil {
		return nil, err
	}
	// Convert relative paths to absolute for backward compatibility
	absPaths := make([]string, len(result.Files))
	for i, relPath := range result.Files {
		absPaths[i] = filepath.Join(result.RootPath, relPath)
	}
	return absPaths, nil
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
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	tree.Sort()
	return &ScanResult{Tree: tree, Files: files, RootPath: s.config.RootPath}, nil
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
// Supports patterns like: **/*.md, docs/**, docs/**/*.md, **/test/**
func matchDoublestar(pattern, path string) bool {
	return matchDoublestarRecursive(pattern, path)
}

// matchDoublestarRecursive implements recursive doublestar matching.
func matchDoublestarRecursive(pattern, path string) bool {
	// Find the first ** in the pattern
	idx := strings.Index(pattern, "**")
	if idx == -1 {
		// No **, use regular glob matching
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	// Split pattern into before and after **
	before := pattern[:idx]
	after := pattern[idx+2:]

	// Remove leading/trailing slashes from after
	after = strings.TrimPrefix(after, "/")

	// Check that the path starts with the prefix (before **)
	if before != "" {
		before = strings.TrimSuffix(before, "/")
		if before != "" {
			if !strings.HasPrefix(path, before) {
				return false
			}
			// Check that prefix is followed by / or is exact match
			rest := path[len(before):]
			if rest == "" {
				// Path equals prefix exactly - only match if pattern is prefix/**
				// and there's no suffix, meaning we need something after the prefix
				// For "docs/**" matching "docs", we return false because docs is not inside docs/
				return false
			}
			if !strings.HasPrefix(rest, "/") {
				return false
			}
			path = strings.TrimPrefix(rest, "/")
		}
	}

	// If path is empty after prefix stripping, ** matches nothing
	if path == "" {
		return after == ""
	}

	// If no suffix pattern after **, match everything
	if after == "" {
		return true
	}

	// ** can match zero or more path segments
	// Try matching the suffix at each possible position
	pathParts := strings.Split(path, "/")
	for i := 0; i <= len(pathParts); i++ {
		remaining := strings.Join(pathParts[i:], "/")
		if matchDoublestarRecursive(after, remaining) {
			return true
		}
	}

	return false
}
