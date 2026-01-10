package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danjdewhurst/go-toc/internal/parser"
	"github.com/danjdewhurst/go-toc/internal/scanner"
	"github.com/danjdewhurst/go-toc/internal/toc"
	"github.com/danjdewhurst/go-toc/internal/worker"
)

// TestIntegration tests the full workflow from scanning to generation.
func TestIntegration(t *testing.T) {
	tmpDir := setupIntegrationTestDir(t)
	defer os.RemoveAll(tmpDir)

	// 1. Scan directory
	scanConfig := scanner.Config{
		RootPath:     tmpDir,
		UseGitignore: true,
		MaxDepth:     0,
	}

	s := scanner.New(scanConfig)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// 2. Get markdown files for summary extraction
	files, err := s.GetMarkdownFiles()
	if err != nil {
		t.Fatalf("GetMarkdownFiles failed: %v", err)
	}

	// Verify we found the expected files (not ignored ones)
	if len(files) < 3 {
		t.Errorf("expected at least 3 markdown files, got %d", len(files))
	}

	// Verify ignored files are not included
	for _, f := range files {
		if strings.Contains(f, "ignored") || strings.Contains(f, "node_modules") {
			t.Errorf("should not include ignored file: %s", f)
		}
	}

	// 3. Extract summaries using worker pool
	jobs := make([]worker.Job, len(files))
	for i, file := range files {
		jobs[i] = worker.Job{FilePath: file, Data: 100}
	}

	processFunc := func(job worker.Job) worker.Result {
		maxChars := job.Data.(int)
		summary, err := parser.ExtractSummary(job.FilePath, maxChars)
		return worker.Result{
			FilePath: job.FilePath,
			Summary:  summary,
			Error:    err,
		}
	}

	results := worker.ProcessAll(jobs, 4, processFunc)

	// Verify summaries were extracted
	summaryCount := 0
	for _, r := range results {
		if r.Error == nil && r.Summary != "" {
			summaryCount++
		}
	}

	if summaryCount == 0 {
		t.Error("expected at least one summary to be extracted")
	}

	// 4. Convert results to summaries map
	summaries := make(map[string]string)
	for filePath, result := range results {
		if result.Error == nil && result.Summary != "" {
			relPath, _ := filepath.Rel(tmpDir, filePath)
			summaries[relPath] = result.Summary
		}
	}

	// 5. Generate ToC
	genConfig := toc.GeneratorConfig{
		Title:          "Integration Test ToC",
		IncludeSummary: true,
		Summaries:      summaries,
	}

	gen := toc.NewGenerator(genConfig)
	output := gen.Generate(tree)

	// 6. Verify output
	expectedStrings := []string{
		"# Integration Test ToC",
		"README.md",
		"docs/",
		"guide.md",
		"├──",
		"└──",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q", expected)
		}
	}

	// Should contain at least one summary blockquote
	if !strings.Contains(output, ">") {
		t.Error("output should contain summary blockquotes")
	}

	// Should not contain ignored content
	if strings.Contains(output, "ignored") {
		t.Error("output should not contain ignored files")
	}
	if strings.Contains(output, "node_modules") {
		t.Error("output should not contain node_modules")
	}
}

// TestIntegrationSingleThreaded tests single-threaded processing.
func TestIntegrationSingleThreaded(t *testing.T) {
	tmpDir := setupIntegrationTestDir(t)
	defer os.RemoveAll(tmpDir)

	scanConfig := scanner.Config{
		RootPath: tmpDir,
	}

	s := scanner.New(scanConfig)
	files, err := s.GetMarkdownFiles()
	if err != nil {
		t.Fatalf("GetMarkdownFiles failed: %v", err)
	}

	jobs := make([]worker.Job, len(files))
	for i, file := range files {
		jobs[i] = worker.Job{FilePath: file, Data: 50}
	}

	processFunc := func(job worker.Job) worker.Result {
		maxChars := job.Data.(int)
		summary, err := parser.ExtractSummary(job.FilePath, maxChars)
		return worker.Result{
			FilePath: job.FilePath,
			Summary:  summary,
			Error:    err,
		}
	}

	// Single-threaded processing
	results := worker.ProcessSequential(jobs, processFunc)

	if len(results) != len(files) {
		t.Errorf("expected %d results, got %d", len(files), len(results))
	}
}

// TestIntegrationFrontmatter tests that frontmatter is properly skipped.
func TestIntegrationFrontmatter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-frontmatter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file with frontmatter
	content := `---
title: Test Document
date: 2024-01-01
tags:
  - test
  - example
---

# Main Heading

This is the actual content that should be extracted as the summary.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "doc.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := parser.ExtractSummary(filepath.Join(tmpDir, "doc.md"), 100)
	if err != nil {
		t.Fatalf("ExtractSummary failed: %v", err)
	}

	if strings.Contains(summary, "title:") || strings.Contains(summary, "2024-01-01") {
		t.Error("summary should not contain frontmatter content")
	}

	if !strings.Contains(summary, "actual content") {
		t.Error("summary should contain the actual paragraph content")
	}
}

// TestIntegrationEmptyDir tests handling of empty directories.
func TestIntegrationEmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-empty-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	scanConfig := scanner.Config{
		RootPath: tmpDir,
	}

	s := scanner.New(scanConfig)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	gen := toc.NewGenerator(toc.GeneratorConfig{Title: "Empty"})
	output := gen.Generate(tree)

	// Should just have the title, no tree content
	if !strings.Contains(output, "# Empty") {
		t.Error("output should contain title")
	}

	// Should not have any tree characters
	if strings.Contains(output, "├──") || strings.Contains(output, "└──") {
		t.Error("output should not have tree structure for empty dir")
	}
}

// TestIntegrationDeepNesting tests deeply nested directories.
func TestIntegrationDeepNesting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-toc-deep-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create deeply nested structure
	deepPath := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deepPath, "deep.md"), []byte("# Deep\n\nDeep content."), 0644); err != nil {
		t.Fatal(err)
	}

	scanConfig := scanner.Config{
		RootPath: tmpDir,
	}

	s := scanner.New(scanConfig)
	tree, err := s.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	gen := toc.NewGenerator(toc.GeneratorConfig{Title: "Deep Test"})
	output := gen.Generate(tree)

	if !strings.Contains(output, "deep.md") {
		t.Errorf("output should contain deeply nested file, got:\n%s", output)
	}

	// Should have proper tree structure
	if !strings.Contains(output, "└──") {
		t.Errorf("output should have tree structure, got:\n%s", output)
	}
}

// Helper function to set up test directory
func setupIntegrationTestDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "go-toc-integration-test")
	if err != nil {
		t.Fatal(err)
	}

	// Create .gitignore
	gitignore := `
ignored/
node_modules/
*.log
`
	writeFile(t, tmpDir, ".gitignore", gitignore)

	// Create markdown files with various content
	writeFile(t, tmpDir, "README.md", `# Project README

This is the main project readme with important information.

## Features

- Feature 1
- Feature 2
`)

	writeFile(t, tmpDir, "docs/guide.md", `---
title: Getting Started
---

# Getting Started Guide

Welcome to the getting started guide for this project.

## Installation

Run the installer.
`)

	writeFile(t, tmpDir, "docs/api/reference.md", `# API Reference

Complete API reference documentation for developers.

## Endpoints

Various endpoints are documented here.
`)

	writeFile(t, tmpDir, "docs/api/examples.md", `# API Examples

Code examples showing how to use the API effectively.
`)

	// Create ignored content
	writeFile(t, tmpDir, "ignored/secret.md", "# Secret\n\nThis should be ignored.")
	writeFile(t, tmpDir, "node_modules/pkg/readme.md", "# Package\n\nThis should be ignored.")

	return tmpDir
}

func writeFile(t *testing.T, base, path, content string) {
	t.Helper()
	fullPath := filepath.Join(base, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}
