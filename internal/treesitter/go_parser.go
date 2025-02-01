package treesitter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// -----------------------------------------------------------------------------
// Type Definitions (for Go parsing)
// -----------------------------------------------------------------------------

// Language represents a programming language.
type Language string

// ProjectMetadata holds metadata about the project, including detected languages,
// relative source file paths, and the ignore patterns derived from .gitignore.
type ProjectMetadata struct {
	Languages    []Language `json:"languages"`
	SourceFiles  []string   `json:"source_files"`
	IgnoredFiles []string   `json:"ignored_files"`
}

// FunctionDependencies represents additional dependency information for a function.
type FunctionDependencies struct {
	Handlers    []string `json:"handlers"`    // e.g., handler functions used
	Utilities   []string `json:"utilities"`   // e.g., utility functions used
	Invocations []string `json:"invocations"` // e.g., file paths where the function is invoked
}

// FunctionInfo represents an extracted function definition.
type FunctionInfo struct {
	Name         string               `json:"name"`
	File         string               `json:"file"` // Relative path (e.g. "/prbuddy-go/cmd/root.go")
	StartLine    int                  `json:"start_line"`
	EndLine      int                  `json:"end_line"`
	Imports      []string             `json:"imports"`      // List of file paths that import this function
	Returns      []string             `json:"returns"`      // List of return types
	Dependencies FunctionDependencies `json:"dependencies"` // Nested dependency info
}

// ProjectMap holds the function-level dependency map.
type ProjectMap struct {
	Functions []FunctionInfo `json:"functions"`
}

// Parser is the interface for tree-sitter–based parsing.
type Parser interface {
	DetectLanguages(rootDir string) ([]Language, error)
	BuildProjectMetadata(rootDir string) (*ProjectMetadata, error)
	BuildProjectMap(rootDir string) (*ProjectMap, error)
}

// -----------------------------------------------------------------------------
// GoParser Implementation Using Tree-Sitter
// -----------------------------------------------------------------------------

// GoParser is a concrete implementation of Parser for Go projects using Tree-Sitter.
type GoParser struct {
	ignoredPatterns []*regexp.Regexp // Regex patterns compiled from .gitignore
}

// NewGoParser creates a new GoParser instance.
func NewGoParser() Parser {
	return &GoParser{}
}

// DetectLanguages scans the project for .go files that are not ignored,
// and returns "go" if any are found.
func (p *GoParser) DetectLanguages(rootDir string) ([]Language, error) {
	var detected []Language

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			if !utils.IsIgnored(path, p.ignoredPatterns) {
				detected = append(detected, "go")
				return filepath.SkipDir // Stop after detecting Go.
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return detected, nil
}

// BuildProjectMetadata scans for .go files (converting absolute paths
// to relative paths based on the repository's base name) and loads .gitignore patterns.
func (p *GoParser) BuildProjectMetadata(rootDir string) (*ProjectMetadata, error) {
	// Read .gitignore patterns
	patterns, err := utils.ReadGitignore(rootDir)
	if err != nil {
		// If .gitignore doesn't exist or fails to open, proceed with no patterns
		patterns = []*regexp.Regexp{}
	}
	p.ignoredPatterns = patterns

	var sourceFiles []string
	repoName := filepath.Base(rootDir)

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			if !utils.IsIgnored(path, p.ignoredPatterns) {
				relPath, relErr := filepath.Rel(rootDir, path)
				if relErr == nil {
					// e.g., "/prbuddy-go/cmd/root.go"
					sourceFiles = append(sourceFiles, fmt.Sprintf("/%s/%s", repoName, relPath))
				} else {
					sourceFiles = append(sourceFiles, path)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	metadata := &ProjectMetadata{
		Languages:    []Language{"go"},
		SourceFiles:  sourceFiles,
		IgnoredFiles: patternStrings(patterns),
	}
	return metadata, nil
}

// BuildProjectMap parses each Go source file using Tree-Sitter to extract function declarations,
// return types, and populates placeholders for imports and dependencies.
func (p *GoParser) BuildProjectMap(rootDir string) (*ProjectMap, error) {
	metadata, err := p.BuildProjectMetadata(rootDir)
	if err != nil {
		return nil, err
	}

	var functions []FunctionInfo

	tsParser := sitter.NewParser()
	tsParser.SetLanguage(golang.GetLanguage())

	// Updated Tree-Sitter query to capture function declarations and their return types
	queryStr := `
		(function_declaration
			name: (identifier) @func.name
			parameters: (parameter_list) @func.params
			result: (parameter_list) @func.result
		)
	`

	query, err := sitter.NewQuery([]byte(queryStr), golang.GetLanguage())
	if err != nil {
		return nil, fmt.Errorf("failed to compile tree-sitter query: %w", err)
	}
	defer query.Close()

	for _, file := range metadata.SourceFiles {
		parts := strings.SplitN(file, "/", 3)
		var absPath string
		if len(parts) == 3 {
			absPath = filepath.Join(rootDir, parts[2])
		} else {
			absPath = file
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		ctx := context.Background()
		tree, err := tsParser.ParseCtx(ctx, nil, content)
		if err != nil || tree == nil {
			continue
		}

		qCursor := sitter.NewQueryCursor()
		qCursor.Exec(query, tree.RootNode())

		for {
			match, ok := qCursor.NextMatch()
			if !ok {
				break
			}

			var funcName string
			var returns []string
			var startLine, endLine int

			for _, capture := range match.Captures {
				node := capture.Node
				name := query.CaptureNameForId(capture.Index)

				switch name {
				case "func.name":
					funcName = node.Content(content)
					startLine = int(node.StartPoint().Row) + 1
					endLine = int(node.EndPoint().Row) + 1
				case "func.result":
					returnType := node.Content(content)
					returns = append(returns, returnType)
				}
			}

			if funcName != "" {
				funcInfo := FunctionInfo{
					Name:      funcName,
					File:      file,
					StartLine: startLine,
					EndLine:   endLine,
					Imports:   []string{},
					Returns:   returns,
					Dependencies: FunctionDependencies{
						Handlers:    []string{},
						Utilities:   []string{},
						Invocations: []string{},
					},
				}
				functions = append(functions, funcInfo)
			}
		}
		qCursor.Close()
	}

	projectMap := &ProjectMap{Functions: functions}
	return projectMap, nil
}

// patternStrings converts a slice of compiled regexes to their string representations.
func patternStrings(patterns []*regexp.Regexp) []string {
	var out []string
	for _, pat := range patterns {
		out = append(out, pat.String())
	}
	return out
}
