package toc

import (
	"fmt"
	"strings"
)

const (
	treeVertical   = "│"
	treeBranch     = "├──"
	treeLastBranch = "└──"
	treeSpace      = "    "
	treePipe       = "│   "
)

// GeneratorConfig holds options for ToC generation.
type GeneratorConfig struct {
	Title          string // Title for the ToC
	IncludeSummary bool   // Whether to include file summaries
	Summaries      map[string]string // Map of file path to summary
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
	var sb strings.Builder

	// Write title
	sb.WriteString("# ")
	sb.WriteString(g.config.Title)
	sb.WriteString("\n\n")

	// Track the prefix for each depth level
	// This determines whether to show │ or space for each level
	prefixes := make([]string, 0)

	tree.Walk(func(node *Node, depth int, isLast bool) {
		// Adjust prefixes slice to current depth
		for len(prefixes) > depth {
			prefixes = prefixes[:len(prefixes)-1]
		}

		// Build the line prefix
		var linePrefix string
		for _, p := range prefixes {
			linePrefix += p
		}

		// Add branch indicator
		if isLast {
			linePrefix += treeLastBranch + " "
		} else {
			linePrefix += treeBranch + " "
		}

		// Write the line
		sb.WriteString(linePrefix)

		if node.IsDir {
			// Directory: just show name with trailing /
			sb.WriteString(node.Name)
			sb.WriteString("/\n")
		} else {
			// File: show as markdown link
			sb.WriteString("[")
			sb.WriteString(node.Name)
			sb.WriteString("](")
			sb.WriteString(node.Path)
			sb.WriteString(")\n")

			// Add summary if enabled and available
			if g.config.IncludeSummary {
				summary := node.Summary
				if summary == "" {
					summary = g.config.Summaries[node.Path]
				}

				if summary != "" {
					// Build summary prefix (same as line but with continuation)
					var summaryPrefix string
					for _, p := range prefixes {
						summaryPrefix += p
					}
					if isLast {
						summaryPrefix += treeSpace
					} else {
						summaryPrefix += treePipe
					}

					sb.WriteString(summaryPrefix)
					sb.WriteString("> ")
					sb.WriteString(summary)
					sb.WriteString("\n")
				}
			}
		}

		// Update prefixes for children
		if node.IsDir {
			if isLast {
				prefixes = append(prefixes, treeSpace)
			} else {
				prefixes = append(prefixes, treePipe)
			}
		}
	})

	return sb.String()
}

// GenerateSimple creates a simple bullet list ToC (alternative format).
func (g *Generator) GenerateSimple(tree *Tree) string {
	var sb strings.Builder

	// Write title
	sb.WriteString("# ")
	sb.WriteString(g.config.Title)
	sb.WriteString("\n\n")

	tree.Walk(func(node *Node, depth int, isLast bool) {
		// Indent based on depth
		indent := strings.Repeat("  ", depth)

		sb.WriteString(indent)
		sb.WriteString("- ")

		if node.IsDir {
			sb.WriteString("**")
			sb.WriteString(node.Name)
			sb.WriteString("/**\n")
		} else {
			sb.WriteString("[")
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
					sb.WriteString("  > ")
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
