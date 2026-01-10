# go-toc

A CLI tool for generating markdown table of contents from directory structures.

## Features

- Recursively scans directories for markdown files (`.md`, `.markdown`)
- Generates ASCII tree structure output in markdown format
- Extracts first paragraph summaries from files
- Supports `.gitignore` patterns
- Configurable ignore patterns via glob syntax
- Concurrent file processing with goroutines
- Configurable recursion depth
- Output to file or stdout

## Installation

### From Releases

Download the latest binary from the [releases page](https://github.com/danjdewhurst/go-toc/releases).

### From Source

```bash
go install github.com/danjdewhurst/go-toc@latest
```

### Build Locally

```bash
git clone https://github.com/danjdewhurst/go-toc.git
cd go-toc
go build -o go-toc .
```

## Usage

```bash
go-toc [directory] [flags]
```

### Examples

```bash
# Scan current directory
go-toc .

# Scan specific directory with summaries
go-toc ./docs --summary

# Limit recursion depth
go-toc . --max-depth 2

# Ignore patterns
go-toc . --ignore "vendor/*" --ignore "*.tmp"

# Include .gitignore patterns
go-toc . --gitignore

# Output to file
go-toc . --summary --output toc.md

# Custom title
go-toc . --title "Documentation Index"

# Combine options
go-toc ./docs -s -c 150 -d 3 -g -o docs-toc.md
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--ignore` | `-i` | string[] | `[]` | Glob patterns to ignore (can be specified multiple times) |
| `--gitignore` | `-g` | bool | `false` | Include `.gitignore` patterns |
| `--max-depth` | `-d` | int | `0` | Maximum recursion depth (0 = unlimited) |
| `--summary` | `-s` | bool | `false` | Include first paragraph summary for each file |
| `--summary-chars` | `-c` | int | `100` | Maximum characters for summary |
| `--single-threaded` | | bool | `false` | Disable concurrent processing |
| `--output` | `-o` | string | `""` | Output file (default: stdout) |
| `--title` | `-t` | string | `"Table of Contents"` | Title for the table of contents |
| `--version` | `-v` | | | Show version |
| `--help` | `-h` | | | Show help |

## Output Format

The tool generates an ASCII tree structure:

```markdown
# Table of Contents

├── api/
│   ├── [handlers.md](api/handlers.md)
│   │   > HTTP request handlers for the REST API endpoints...
│   └── [routes.md](api/routes.md)
│       > Route definitions and middleware configuration...
├── docs/
│   └── [getting-started.md](docs/getting-started.md)
│       > Quick guide to get up and running with the project...
└── [README.md](README.md)
    > Main project documentation and overview of features...
```

## How It Works

1. **Scanning**: Recursively walks the directory tree, identifying markdown files
2. **Filtering**: Applies ignore patterns and `.gitignore` rules
3. **Parsing**: Extracts summaries from markdown files (skipping YAML frontmatter)
4. **Generation**: Builds the tree structure and outputs as markdown

### Summary Extraction

When `--summary` is enabled, the tool:

- Skips YAML frontmatter (content between `---` delimiters)
- Skips headings, list items, and code blocks
- Extracts the first paragraph of actual content
- Strips markdown formatting (bold, italic, links)
- Truncates to the specified character limit

## Development

### Prerequisites

- Go 1.21+

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o go-toc .
```

### Linting

```bash
golangci-lint run
```

## License

MIT License - see [LICENSE](LICENSE) for details.

Copyright (c) 2026 Daniel Dewhurst
