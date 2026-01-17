package toc

import (
	"fmt"
	"strings"
)

const (
	// ASCII tree characters
	treeBranch     = "├──"
	treeLastBranch = "└──"
	treeSpace      = "    "
	treePipe       = "│   "

	// Fancy emoji characters
	emojiFolder = "📁"
	emojiFile   = "📄"
)

// GeneratorConfig holds options for ToC generation.
type GeneratorConfig struct {
	Title           string            // Title for the ToC
	IncludeSummary  bool              // Whether to include file summaries
	Summaries       map[string]string // Map of file path to summary
	Fancy           bool              // Use emoji icons instead of ASCII tree
	GenerateAnchors bool              // Add anchor IDs to entries for linking
}

// Generator creates markdown table of contents output.
type Generator struct {
	config GeneratorConfig
}

// NewGenerator creates a new ToC generator.
func NewGenerator(config GeneratorConfig) *Generator {
	if config.Title == "" {
		config.Title = "Table of Contents"
	}
	if config.Summaries == nil {
		config.Summaries = make(map[string]string)
	}

	return &Generator{
		config: config,
	}
}

// Generate creates the markdown ToC from a tree.
func (g *Generator) Generate(tree *Tree) string {
	if g.config.Fancy {
		return g.generateFancy(tree)
	}
	return g.generateASCII(tree)
}

// generateASCII creates ASCII tree style output.
func (g *Generator) generateASCII(tree *Tree) string {
	var sb strings.Builder

	// Write title
	sb.WriteString("# ")
	sb.WriteString(g.config.Title)
	sb.WriteString("\n\n")

	// Track which levels have more siblings coming (for drawing │ vs space)
	// isLastAtLevel[i] = true means level i's parent was the last child
	isLastAtLevel := make([]bool, 0)

	tree.Walk(func(node *Node, depth int, isLast bool) {
		// Trim to current depth
		if len(isLastAtLevel) > depth {
			isLastAtLevel = isLastAtLevel[:depth]
		}

		// Build prefix from ancestor information
		linePrefix := buildPrefix(isLastAtLevel, isLast)

		// Write the entry
		sb.WriteString(linePrefix)

		if node.IsDir {
			if g.config.GenerateAnchors {
				fmt.Fprintf(&sb, "<a id=\"%s\"></a>", generateSlug(node.Path))
			}
			sb.WriteString(node.Name)
			sb.WriteString("/\n")
		} else {
			if g.config.GenerateAnchors {
				fmt.Fprintf(&sb, "<a id=\"%s\"></a>", generateSlug(node.Path))
			}
			fmt.Fprintf(&sb, "[%s](%s)\n", node.Name, node.Path)

			// Add summary if enabled
			if g.config.IncludeSummary {
				summary := node.Summary
				if summary == "" {
					summary = g.config.Summaries[node.Path]
				}
				if summary != "" {
					summaryPrefix := buildContinuationPrefix(isLastAtLevel, isLast)
					sb.WriteString(summaryPrefix)
					sb.WriteString("> ")
					sb.WriteString(summary)
					sb.WriteString("\n")
				}
			}
		}

		// Track this level for children
		if node.IsDir {
			isLastAtLevel = append(isLastAtLevel, isLast)
		}
	})

	return sb.String()
}

// buildPrefix constructs the tree prefix for a node.
func buildPrefix(isLastAtLevel []bool, isLast bool) string {
	var sb strings.Builder
	for _, wasLast := range isLastAtLevel {
		if wasLast {
			sb.WriteString(treeSpace)
		} else {
			sb.WriteString(treePipe)
		}
	}
	if isLast {
		sb.WriteString(treeLastBranch)
	} else {
		sb.WriteString(treeBranch)
	}
	sb.WriteString(" ")
	return sb.String()
}

// buildContinuationPrefix constructs the prefix for continuation lines (like summaries).
func buildContinuationPrefix(isLastAtLevel []bool, isLast bool) string {
	var sb strings.Builder
	for _, wasLast := range isLastAtLevel {
		if wasLast {
			sb.WriteString(treeSpace)
		} else {
			sb.WriteString(treePipe)
		}
	}
	if isLast {
		sb.WriteString(treeSpace)
	} else {
		sb.WriteString(treePipe)
	}
	return sb.String()
}

// generateFancy creates emoji-based output.
func (g *Generator) generateFancy(tree *Tree) string {
	var sb strings.Builder

	// Write title with emoji
	sb.WriteString("# ")
	sb.WriteString(g.config.Title)
	sb.WriteString(" 📚\n\n")

	tree.Walk(func(node *Node, depth int, isLast bool) {
		// Indent based on depth
		indent := strings.Repeat("  ", depth)

		sb.WriteString(indent)

		if node.IsDir {
			// Directory with folder emoji
			sb.WriteString("- ")
			if g.config.GenerateAnchors {
				fmt.Fprintf(&sb, "<a id=\"%s\"></a>", generateSlug(node.Path))
			}
			sb.WriteString(emojiFolder)
			sb.WriteString(" **")
			sb.WriteString(node.Name)
			sb.WriteString("/**\n")
		} else {
			// File with document emoji
			sb.WriteString("- ")
			if g.config.GenerateAnchors {
				fmt.Fprintf(&sb, "<a id=\"%s\"></a>", generateSlug(node.Path))
			}
			sb.WriteString(emojiFile)
			sb.WriteString(" [")
			sb.WriteString(node.Name)
			sb.WriteString("](")
			sb.WriteString(node.Path)
			sb.WriteString(")\n")

			// Add summary if enabled
			if g.config.IncludeSummary {
				summary := node.Summary
				if summary == "" {
					summary = g.config.Summaries[node.Path]
				}

				if summary != "" {
					sb.WriteString(indent)
					sb.WriteString("  > 💬 ")
					sb.WriteString(summary)
					sb.WriteString("\n")
				}
			}
		}
	})

	return sb.String()
}

// SetSummary adds a summary for a specific file path.
func (g *Generator) SetSummary(path, summary string) {
	g.config.Summaries[path] = summary
}

// FormatTree returns just the tree portion without title (for embedding).
func (g *Generator) FormatTree(tree *Tree) string {
	output := g.Generate(tree)

	// Remove the title lines
	lines := strings.Split(output, "\n")
	if len(lines) >= 2 {
		return strings.Join(lines[2:], "\n")
	}

	return output
}

// Summary statistics about the generated ToC.
type Stats struct {
	TotalFiles       int
	TotalDirectories int
	MaxDepth         int
}

// GetStats returns statistics about the tree.
func GetStats(tree *Tree) Stats {
	stats := Stats{}

	tree.Walk(func(node *Node, depth int, isLast bool) {
		if node.IsDir {
			stats.TotalDirectories++
		} else {
			stats.TotalFiles++
		}

		actualDepth := depth + 1
		if actualDepth > stats.MaxDepth {
			stats.MaxDepth = actualDepth
		}
	})

	return stats
}

// FormatStats returns a human-readable summary of stats.
func FormatStats(stats Stats) string {
	return fmt.Sprintf("%d files, %d directories, max depth %d",
		stats.TotalFiles, stats.TotalDirectories, stats.MaxDepth)
}

// generateSlug creates a GitHub-style anchor slug from a path.
// For example: "docs/API Guide.md" -> "docs-api-guide"
func generateSlug(path string) string {
	// Remove file extension
	if idx := strings.LastIndex(path, "."); idx != -1 {
		path = path[:idx]
	}

	var result strings.Builder
	result.Grow(len(path))

	prevDash := false
	for _, r := range strings.ToLower(path) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			result.WriteRune(r)
			prevDash = false
		case r == '-':
			if !prevDash && result.Len() > 0 {
				result.WriteRune('-')
				prevDash = true
			}
		case r == ' ', r == '/', r == '_', r == '.':
			// Convert separators to dashes
			if !prevDash && result.Len() > 0 {
				result.WriteRune('-')
				prevDash = true
			}
		}
		// Skip other characters
	}

	// Trim trailing dash
	s := result.String()
	return strings.TrimSuffix(s, "-")
}
