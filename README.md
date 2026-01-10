<p align="center">
  <h1 align="center">go-toc</h1>
  <p align="center">
    <strong>Blazing fast markdown table of contents generator</strong>
  </p>
  <p align="center">
    Zero dependencies • Single binary • Powered by Go
  </p>
</p>

<p align="center">
  <a href="https://github.com/danjdewhurst/go-toc/releases"><img src="https://img.shields.io/github/v/release/danjdewhurst/go-toc?style=flat-square" alt="Release"></a>
  <a href="https://github.com/danjdewhurst/go-toc/blob/main/LICENSE"><img src="https://img.shields.io/github/license/danjdewhurst/go-toc?style=flat-square" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/danjdewhurst/go-toc"><img src="https://goreportcard.com/badge/github.com/danjdewhurst/go-toc?style=flat-square" alt="Go Report Card"></a>
</p>

---

**go-toc** scans your project directories and generates beautiful, navigable table of contents in markdown format. Perfect for documentation sites, wikis, and project indexes.

```
├── api/
│   ├── [handlers.md](api/handlers.md)
│   │   > HTTP request handlers for the REST API endpoints...
│   └── [routes.md](api/routes.md)
│       > Route definitions and middleware configuration...
└── [README.md](README.md)
    > Main project documentation and overview...
```

## Features

- **Lightning fast** — Concurrent file processing with goroutines
- **Smart filtering** — Respects `.gitignore` patterns out of the box
- **Summary extraction** — Automatically pulls first paragraph from each file
- **AI agent friendly** — Perfect context file for LLM coding assistants
- **Flexible output** — ASCII tree or fancy emoji mode
- **Zero config** — Sensible defaults, works instantly
- **Single binary** — No runtime dependencies, just download and run

## Installation

### Download Binary

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/danjdewhurst/go-toc/releases/latest/download/go-toc_darwin_arm64.tar.gz | tar xz
sudo mv go-toc /usr/local/bin/

# macOS (Intel)
curl -sL https://github.com/danjdewhurst/go-toc/releases/latest/download/go-toc_darwin_amd64.tar.gz | tar xz
sudo mv go-toc /usr/local/bin/

# Linux (x86_64)
curl -sL https://github.com/danjdewhurst/go-toc/releases/latest/download/go-toc_linux_amd64.tar.gz | tar xz
sudo mv go-toc /usr/local/bin/

# Linux (ARM64)
curl -sL https://github.com/danjdewhurst/go-toc/releases/latest/download/go-toc_linux_arm64.tar.gz | tar xz
sudo mv go-toc /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/danjdewhurst/go-toc/releases/latest/download/go-toc_windows_amd64.zip -OutFile go-toc.zip
Expand-Archive go-toc.zip -DestinationPath .
```

Or grab from the [releases page](https://github.com/danjdewhurst/go-toc/releases).

### Go Install

```bash
go install github.com/danjdewhurst/go-toc@latest
```

### Build from Source

```bash
git clone https://github.com/danjdewhurst/go-toc.git
cd go-toc
go build -o go-toc .
```

## Quick Start

```bash
# Generate TOC for current directory
go-toc .

# Include file summaries
go-toc ./docs --summary

# Fancy mode with emojis
go-toc . --fancy --summary

# Output to file
go-toc . --summary --output toc.md
```

## Usage

```bash
go-toc [directory] [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--summary` | `-s` | `false` | Include first paragraph summary for each file |
| `--summary-chars` | `-c` | `100` | Maximum characters for summary |
| `--fancy` | `-f` | `false` | Use emoji icons instead of ASCII tree |
| `--gitignore` | `-g` | `false` | Respect `.gitignore` patterns |
| `--ignore` | `-i` | `[]` | Additional glob patterns to ignore |
| `--max-depth` | `-d` | `0` | Maximum recursion depth (0 = unlimited) |
| `--output` | `-o` | stdout | Output file path |
| `--title` | `-t` | `"Table of Contents"` | Custom title |
| `--single-threaded` | | `false` | Disable concurrent processing |

### Examples

```bash
# Limit depth and ignore vendor
go-toc . --max-depth 2 --ignore "vendor/*"

# Full featured run
go-toc ./docs -s -c 150 -d 3 -g -o docs-toc.md

# Custom title
go-toc . --title "Documentation Index"
```

## Output Formats

### ASCII Tree (default)

```markdown
# Table of Contents

├── api/
│   ├── [handlers.md](api/handlers.md)
│   │   > HTTP request handlers for the REST API endpoints...
│   └── [routes.md](api/routes.md)
│       > Route definitions and middleware configuration...
└── [README.md](README.md)
    > Main project documentation and overview...
```

### Fancy Mode (`--fancy`)

```markdown
# Table of Contents 📚

- 📁 **api/**
  - 📄 [handlers.md](api/handlers.md)
    > 💬 HTTP request handlers for the REST API endpoints...
  - 📄 [routes.md](api/routes.md)
    > 💬 Route definitions and middleware configuration...
- 📄 [README.md](README.md)
  > 💬 Main project documentation and overview...
```

## AI Agent Context

The generated TOC is ideal for providing context to AI coding agents. Instead of searching through directories and reading unnecessary files, an agent can read a single TOC file to understand what documentation exists and where to find relevant information — saving context window space and reducing hallucination.

```bash
# Generate a docs map for your AI agent
go-toc ./docs --summary --output docs-toc.md
```

Include the output file in your agent's context or system prompt, and it can navigate directly to the files it needs.

## How It Works

1. **Scan** — Recursively walks directory tree, identifying markdown files
2. **Filter** — Applies ignore patterns and `.gitignore` rules
3. **Parse** — Extracts summaries (skipping frontmatter and headings)
4. **Generate** — Builds tree structure and outputs markdown

Summary extraction intelligently:
- Skips YAML frontmatter between `---` delimiters
- Ignores headings, list items, and code blocks
- Strips markdown formatting (bold, italic, links)
- Truncates to configured character limit

## Development

```bash
# Run tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Lint
golangci-lint run
```

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<p align="center">
  Made by <a href="https://github.com/danjdewhurst">Daniel Dewhurst</a>
</p>
