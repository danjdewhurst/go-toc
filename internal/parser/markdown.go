package parser

import (
	"bufio"
	"os"
	"strings"
	"unicode"
)

// ExtractSummary extracts the first paragraph from a markdown file.
// It skips YAML frontmatter (content between --- delimiters) and headings.
// Returns an empty string if no suitable content is found.
func ExtractSummary(filePath string, maxChars int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	inFrontmatter := false
	frontmatterStart := false
	inCodeBlock := false
	foundContent := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Handle YAML frontmatter
		if trimmed == "---" {
			if !frontmatterStart {
				// Start of frontmatter (must be at the beginning)
				if len(lines) == 0 && !foundContent {
					frontmatterStart = true
					inFrontmatter = true
					continue
				}
			} else if inFrontmatter {
				// End of frontmatter
				inFrontmatter = false
				continue
			}
		}

		// Skip frontmatter content
		if inFrontmatter {
			continue
		}

		// Track fenced code blocks (``` or ~~~)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}

		// Skip code block content
		if inCodeBlock {
			continue
		}

		// Skip empty lines at the start
		if trimmed == "" && !foundContent {
			continue
		}

		// Skip headings (lines starting with #)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Skip HTML comments
		if strings.HasPrefix(trimmed, "<!--") {
			continue
		}

		// Skip horizontal rules
		if isHorizontalRule(trimmed) {
			continue
		}

		// Skip list items and blockquotes
		if strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "*") ||
			strings.HasPrefix(trimmed, ">") {
			continue
		}

		// Found actual content
		if trimmed != "" {
			foundContent = true
			lines = append(lines, trimmed)

			// Check if we have a complete paragraph (next line is empty or we have enough)
			// We'll collect until we hit an empty line
		} else if foundContent {
			// Empty line after content means end of paragraph
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if len(lines) == 0 {
		return "", nil
	}

	// Join lines and clean up
	paragraph := strings.Join(lines, " ")
	paragraph = cleanMarkdown(paragraph)

	// Truncate if needed
	return truncate(paragraph, maxChars), nil
}

// cleanMarkdown removes common markdown formatting from text.
func cleanMarkdown(text string) string {
	// Strip inline code backticks but keep the content
	text = stripDelimiters(text, "`", "`")

	// Remove bold/italic markers in single pass
	text = removeFormattingMarkers(text)

	// Remove images before links: ![alt](url) -> alt
	text = removeImagesSyntax(text)

	// Remove links but keep text: [text](url) -> text
	text = removeLinksSyntax(text)

	// Clean up extra whitespace
	text = strings.Join(strings.Fields(text), " ")

	return text
}

// removeFormattingMarkers removes **, __, *, _ in a single pass.
func removeFormattingMarkers(text string) string {
	var result strings.Builder
	result.Grow(len(text))

	i := 0
	for i < len(text) {
		// Check for ** or __
		if i+1 < len(text) {
			pair := text[i : i+2]
			if pair == "**" || pair == "__" {
				i += 2
				continue
			}
		}
		// Check for single * or _ (but not in the middle of words for _)
		if text[i] == '*' {
			i++
			continue
		}
		if text[i] == '_' {
			i++
			continue
		}
		result.WriteByte(text[i])
		i++
	}

	return result.String()
}

// stripDelimiters removes start/end delimiter markers but keeps the content between them.
// For example: stripDelimiters("`code`", "`", "`") returns "code".
func stripDelimiters(text, start, end string) string {
	if !strings.Contains(text, start) {
		return text
	}

	var result strings.Builder
	result.Grow(len(text))

	i := 0
	for i < len(text) {
		if i+len(start) <= len(text) && text[i:i+len(start)] == start {
			endIdx := strings.Index(text[i+len(start):], end)
			if endIdx != -1 {
				// Keep the content, skip the delimiters
				result.WriteString(text[i+len(start) : i+len(start)+endIdx])
				i = i + len(start) + endIdx + len(end)
				continue
			}
		}
		result.WriteByte(text[i])
		i++
	}

	return result.String()
}

// removePattern removes content between start and end delimiters.
// Uses single-pass algorithm for O(n) performance.
func removePattern(text, start, end string) string {
	if !strings.Contains(text, start) {
		return text
	}

	var result strings.Builder
	result.Grow(len(text))

	i := 0
	for i < len(text) {
		// Check if we're at a start delimiter
		if i+len(start) <= len(text) && text[i:i+len(start)] == start {
			// Find the end delimiter
			endIdx := strings.Index(text[i+len(start):], end)
			if endIdx != -1 {
				// Skip the delimited content
				i = i + len(start) + endIdx + len(end)
				continue
			}
		}
		result.WriteByte(text[i])
		i++
	}

	return result.String()
}

// removeLinksSyntax converts [text](url) to text.
func removeLinksSyntax(text string) string {
	var result strings.Builder
	i := 0

	for i < len(text) {
		// Look for [
		if text[i] == '[' {
			// Find matching ]
			bracketEnd := findMatchingBracket(text, i, '[', ']')
			if bracketEnd != -1 && bracketEnd+1 < len(text) && text[bracketEnd+1] == '(' {
				// Find matching )
				parenEnd := findMatchingBracket(text, bracketEnd+1, '(', ')')
				if parenEnd != -1 {
					// Extract link text
					linkText := text[i+1 : bracketEnd]
					result.WriteString(linkText)
					i = parenEnd + 1
					continue
				}
			}
		}
		result.WriteByte(text[i])
		i++
	}

	return result.String()
}

// removeImagesSyntax converts ![alt](url) to alt.
func removeImagesSyntax(text string) string {
	var result strings.Builder
	i := 0

	for i < len(text) {
		// Look for ![
		if i+1 < len(text) && text[i] == '!' && text[i+1] == '[' {
			// Find matching ]
			bracketEnd := findMatchingBracket(text, i+1, '[', ']')
			if bracketEnd != -1 && bracketEnd+1 < len(text) && text[bracketEnd+1] == '(' {
				// Find matching )
				parenEnd := findMatchingBracket(text, bracketEnd+1, '(', ')')
				if parenEnd != -1 {
					// Extract alt text
					altText := text[i+2 : bracketEnd]
					result.WriteString(altText)
					i = parenEnd + 1
					continue
				}
			}
		}
		result.WriteByte(text[i])
		i++
	}

	return result.String()
}

// findMatchingBracket finds the matching closing bracket.
// Handles escaped characters (e.g., \] is not treated as closing bracket).
func findMatchingBracket(text string, start int, open, close byte) int {
	if start >= len(text) || text[start] != open {
		return -1
	}

	count := 1
	for i := start + 1; i < len(text); i++ {
		// Skip escaped characters
		if text[i] == '\\' && i+1 < len(text) {
			i++ // Skip next character
			continue
		}

		if text[i] == open {
			count++
		} else if text[i] == close {
			count--
			if count == 0 {
				return i
			}
		}
	}

	return -1
}

// truncate shortens text to maxChars, adding "..." if truncated.
func truncate(text string, maxChars int) string {
	if maxChars <= 0 || len(text) <= maxChars {
		return text
	}

	// Find a good break point (word boundary)
	truncated := text[:maxChars]

	// Try to break at last space
	lastSpace := strings.LastIndexFunc(truncated, unicode.IsSpace)
	if lastSpace > maxChars/2 {
		truncated = truncated[:lastSpace]
	}

	return strings.TrimSpace(truncated) + "..."
}

// isHorizontalRule checks if a line is a markdown horizontal rule.
func isHorizontalRule(line string) bool {
	// Must be at least 3 characters
	if len(line) < 3 {
		return false
	}

	// Check for ---, ***, or ___
	clean := strings.ReplaceAll(line, " ", "")
	if len(clean) < 3 {
		return false
	}

	allSame := true
	char := clean[0]
	if char != '-' && char != '*' && char != '_' {
		return false
	}

	for i := 1; i < len(clean); i++ {
		if clean[i] != char {
			allSame = false
			break
		}
	}

	return allSame
}
