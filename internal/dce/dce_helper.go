package dce

import (
	"regexp"
)

// Centralized regex patterns and helper functions for the DCE module.

// FuncPattern matches function definitions (e.g. "func FunctionName(").
var FuncPattern = regexp.MustCompile(`(?i)^\s*(func|def|function|public|private|static|void)\s+([A-Za-z0-9_]+)\s*\(`)

// ImportExportPattern matches import or export statements.
var ImportExportPattern = regexp.MustCompile(`(?i)^\s*(import|from|require\(|export)\s+(.+)`)

// DiffHeaderPattern matches diff header lines (e.g. "diff --git a/file.go b/file.go").
var DiffHeaderPattern = regexp.MustCompile(`^diff --git\s+`)

// ParseFunctionNames extracts function names from the provided content using FuncPattern.
func ParseFunctionNames(content string) []string {
	matches := FuncPattern.FindAllStringSubmatch(content, -1)
	var functions []string
	for _, m := range matches {
		if len(m) >= 3 {
			functions = append(functions, m[2])
		}
	}
	return functions
}

// ParseImportExportStatements extracts complete import/export statements from the content.
func ParseImportExportStatements(content string) []string {
	matches := ImportExportPattern.FindAllStringSubmatch(content, -1)
	var statements []string
	for _, m := range matches {
		if len(m) >= 3 {
			statements = append(statements, m[0])
		}
	}
	return statements
}

// ExtractFilePathFromDiff extracts the file path from a diff header line.
// For example, given "diff --git a/foo.go b/foo.go", it returns "foo.go".
func ExtractFilePathFromDiff(line string) string {
	parts := regexp.MustCompile(`\s+`).Split(line, -1)
	if len(parts) >= 3 {
		return trimPrefix(parts[2], "b/")
	}
	return "unknown_file"
}

// trimPrefix removes the specified prefix from a string.
func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
