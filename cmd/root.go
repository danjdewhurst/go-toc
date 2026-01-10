package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/danjdewhurst/go-toc/internal/parser"
	"github.com/danjdewhurst/go-toc/internal/scanner"
	"github.com/danjdewhurst/go-toc/internal/toc"
	"github.com/danjdewhurst/go-toc/internal/worker"
)

// Version is set at build time.
var Version = "dev"

var (
	ignorePatterns []string
	useGitignore   bool
	maxDepth       int
	includeSummary bool
	summaryChars   int
	singleThreaded bool
	outputFile     string
	title          string
	fancy          bool
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "go-toc [directory]",
	Short: "Generate a table of contents from markdown files",
	Long: `go-toc scans a directory recursively for markdown files and generates
a table of contents in a tree structure format.

Example:
  go-toc .
  go-toc ./docs --summary --max-depth 3
  go-toc . --ignore "vendor/*" --gitignore`,
	Args: cobra.MaximumNArgs(1),
	RunE: runToc,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringArrayVarP(&ignorePatterns, "ignore", "i", []string{}, "glob patterns to ignore (can be specified multiple times)")
	rootCmd.Flags().BoolVarP(&useGitignore, "gitignore", "g", false, "include .gitignore patterns")
	rootCmd.Flags().IntVarP(&maxDepth, "max-depth", "d", 0, "maximum recursion depth (0 = unlimited)")
	rootCmd.Flags().BoolVarP(&includeSummary, "summary", "s", false, "include first paragraph summary for each file")
	rootCmd.Flags().IntVarP(&summaryChars, "summary-chars", "c", 100, "maximum characters for summary")
	rootCmd.Flags().BoolVar(&singleThreaded, "single-threaded", false, "disable concurrent processing")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")
	rootCmd.Flags().StringVarP(&title, "title", "t", "Table of Contents", "title for the table of contents")
	rootCmd.Flags().BoolVarP(&fancy, "fancy", "f", false, "use emoji icons instead of ASCII tree")

	rootCmd.Version = Version
}

func runToc(cmd *cobra.Command, args []string) error {
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verify directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", targetDir)
	}

	// Create scanner
	scannerConfig := scanner.Config{
		RootPath:       absPath,
		IgnorePatterns: ignorePatterns,
		UseGitignore:   useGitignore,
		MaxDepth:       maxDepth,
	}

	s := scanner.New(scannerConfig)

	// Scan directory
	tree, err := s.Scan()
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Extract summaries if requested
	summaries := make(map[string]string)
	if includeSummary {
		files, err := s.GetMarkdownFiles()
		if err != nil {
			return fmt.Errorf("failed to get markdown files: %w", err)
		}

		summaries = extractSummaries(files, absPath, summaryChars, singleThreaded)
	}

	// Generate ToC
	genConfig := toc.GeneratorConfig{
		Title:          title,
		IncludeSummary: includeSummary,
		Summaries:      summaries,
		Fancy:          fancy,
	}

	gen := toc.NewGenerator(genConfig)
	output := gen.Generate(tree)

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "ToC written to %s\n", outputFile)
	} else {
		fmt.Fprint(cmd.OutOrStdout(), output)
	}

	return nil
}

func extractSummaries(files []string, rootPath string, maxChars int, sequential bool) map[string]string {
	if len(files) == 0 {
		return make(map[string]string)
	}

	// Create jobs
	jobs := make([]worker.Job, len(files))
	for i, file := range files {
		jobs[i] = worker.Job{
			FilePath: file,
			Data:     maxChars,
		}
	}

	// Process function
	processFunc := func(job worker.Job) worker.Result {
		maxChars := job.Data.(int)
		summary, err := parser.ExtractSummary(job.FilePath, maxChars)
		return worker.Result{
			FilePath: job.FilePath,
			Summary:  summary,
			Error:    err,
		}
	}

	// Process jobs
	var results map[string]worker.Result
	if sequential {
		results = worker.ProcessSequential(jobs, processFunc)
	} else {
		numWorkers := runtime.NumCPU()
		if numWorkers > len(files) {
			numWorkers = len(files)
		}
		results = worker.ProcessAll(jobs, numWorkers, processFunc)
	}

	// Convert to relative paths
	summaries := make(map[string]string)
	for filePath, result := range results {
		if result.Error == nil && result.Summary != "" {
			relPath, err := filepath.Rel(rootPath, filePath)
			if err == nil {
				summaries[relPath] = result.Summary
			}
		}
	}

	return summaries
}
