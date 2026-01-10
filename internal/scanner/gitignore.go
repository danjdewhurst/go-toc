package scanner

import (
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// GitignoreManager handles parsing and matching of .gitignore patterns.
type GitignoreManager struct {
	rootPath string
	matchers map[string]*ignore.GitIgnore // Map of directory path to matcher
}

// NewGitignoreManager creates a new gitignore manager for the given root path.
func NewGitignoreManager(rootPath string) *GitignoreManager {
	mgr := &GitignoreManager{
		rootPath: rootPath,
		matchers: make(map[string]*ignore.GitIgnore),
	}

	// Load root .gitignore
	mgr.loadGitignore(rootPath)

	return mgr
}

// loadGitignore loads a .gitignore file from the specified directory.
func (m *GitignoreManager) loadGitignore(dirPath string) {
	gitignorePath := filepath.Join(dirPath, ".gitignore")

	if _, err := os.Stat(gitignorePath); err == nil {
		matcher, err := ignore.CompileIgnoreFile(gitignorePath)
		if err == nil {
			m.matchers[dirPath] = matcher
		}
	}
}

// IsIgnored checks if a path should be ignored based on gitignore patterns.
func (m *GitignoreManager) IsIgnored(relPath string, isDir bool) bool {
	if len(m.matchers) == 0 {
		return false
	}

	// Convert to forward slashes for consistent matching
	normalizedPath := filepath.ToSlash(relPath)

	// Add trailing slash for directories (gitignore convention)
	pathToCheck := normalizedPath
	if isDir {
		pathToCheck = normalizedPath + "/"
	}

	// Check against all loaded gitignore files
	// We need to check parent directories' gitignore files as well
	for matcherDir, matcher := range m.matchers {
		// Get relative path from matcher directory
		matcherRelPath, err := filepath.Rel(m.rootPath, matcherDir)
		if err != nil {
			continue
		}

		// Determine the path relative to this gitignore's location
		var relToMatcher string
		if matcherRelPath == "." {
			relToMatcher = pathToCheck
		} else {
			matcherRelNorm := filepath.ToSlash(matcherRelPath)
			if strings.HasPrefix(normalizedPath, matcherRelNorm+"/") {
				relToMatcher = strings.TrimPrefix(normalizedPath, matcherRelNorm+"/")
				if isDir {
					relToMatcher += "/"
				}
			} else {
				continue // Path is not under this gitignore's directory
			}
		}

		if matcher.MatchesPath(relToMatcher) {
			return true
		}
	}

	return false
}

// LoadGitignoreForDir loads the .gitignore file from a specific directory.
// Call this as you traverse directories to pick up nested .gitignore files.
func (m *GitignoreManager) LoadGitignoreForDir(dirPath string) {
	if _, exists := m.matchers[dirPath]; !exists {
		m.loadGitignore(dirPath)
	}
}
