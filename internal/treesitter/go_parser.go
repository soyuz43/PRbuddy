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

// ProjectMetadata holds project metadata including source files and ignored patterns.
type ProjectMetadata struct {
	Languages    []Language `json:"languages"`
	SourceFiles  []string   `json:"source_files"`
	IgnoredFiles []string   `json:"ignored_files"`
}

// FunctionDependencies tracks function relationships and invocations.
type FunctionDependencies struct {
	Handlers    []string `json:"handlers"`
	Utilities   []string `json:"utilities"`
	Invocations []string `json:"invocations"`
}

// FunctionInfo contains metadata about a Go function.
type FunctionInfo struct {
	Name         string               `json:"name"`
	File         string               `json:"file"`
	StartLine    int                  `json:"start_line"`
	EndLine      int                  `json:"end_line"`
	Imports      []string             `json:"imports"`
	Returns      []string             `json:"returns"`
	Dependencies FunctionDependencies `json:"dependencies"`
}

// ProjectMap represents the complete project function mapping.
type ProjectMap struct {
	Functions []FunctionInfo `json:"functions"`
}

// Parser interface for project analysis operations.
type Parser interface {
	DetectLanguages(rootDir string) ([]Language, error)
	BuildProjectMetadata(rootDir string) (*ProjectMetadata, error)
	BuildProjectMap(rootDir string) (*ProjectMap, error)
}

// -----------------------------------------------------------------------------
// GoParser Implementation
// -----------------------------------------------------------------------------

// GoParser implements Parser for Go projects using Tree-Sitter.
type GoParser struct {
	ignoredPatterns []*regexp.Regexp
}

// NewGoParser creates a new GoParser instance.
func NewGoParser() Parser {
	return &GoParser{}
}

// goParserState manages Tree-Sitter parsing state.
type goParserState struct {
	parser         *sitter.Parser
	functionQuery  *sitter.Query
	importQuery    *sitter.Query
	functionCursor *sitter.QueryCursor
	importCursor   *sitter.QueryCursor
}

// BuildProjectMap constructs a project map with function dependencies.
func (p *GoParser) BuildProjectMap(rootDir string) (*ProjectMap, error) {
	metadata, err := p.BuildProjectMetadata(rootDir)
	if err != nil {
		return nil, err
	}

	state, err := p.setupParserState()
	if err != nil {
		return nil, err
	}
	defer state.parser.Close()
	defer state.functionCursor.Close()
	defer state.importCursor.Close()

	var functions []FunctionInfo
	for _, file := range metadata.SourceFiles {
		fileFuncs, err := p.processGoFile(state, rootDir, file)
		if err != nil {
			continue // Skip problematic files but continue processing
		}
		functions = append(functions, fileFuncs...)
	}

	return &ProjectMap{Functions: functions}, nil
}

// setupParserState initializes Tree-Sitter components.
func (p *GoParser) setupParserState() (*goParserState, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	// Function declaration query
	funcQuery, err := sitter.NewQuery([]byte(`
		(function_declaration
			name: (identifier) @name
			result: (_) @result
		) @func`), golang.GetLanguage())
	if err != nil {
		return nil, fmt.Errorf("failed to create function query: %w", err)
	}

	// Import statement query
	importQuery, err := sitter.NewQuery([]byte(`
    (import_spec
        path: (interpreted_string_literal) @path
    )`), golang.GetLanguage())
	if err != nil {
		return nil, fmt.Errorf("failed to create import query: %w", err)
	}

	return &goParserState{
		parser:         parser,
		functionQuery:  funcQuery,
		importQuery:    importQuery,
		functionCursor: sitter.NewQueryCursor(),
		importCursor:   sitter.NewQueryCursor(),
	}, nil
}

// processGoFile handles processing of individual Go files.
func (p *GoParser) processGoFile(state *goParserState, rootDir string, file string) ([]FunctionInfo, error) {
	absPath, err := p.resolveAbsPath(rootDir, file)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", absPath, err)
	}

	tree, err := state.parser.ParseCtx(context.Background(), nil, content)
	if err != nil || tree == nil {
		return nil, fmt.Errorf("failed to parse %s: %w", absPath, err)
	}

	// Extract imports first
	imports := p.parseImports(state, tree, content)

	// Extract functions with imports
	functions := p.parseFunctions(state, tree, content, file, imports)

	return functions, nil
}

// parseImports extracts import paths from a parsed tree.
func (p *GoParser) parseImports(state *goParserState, tree *sitter.Tree, content []byte) []string {
	var imports []string
	state.importCursor.Exec(state.importQuery, tree.RootNode())

	for {
		match, ok := state.importCursor.NextMatch()
		if !ok {
			break
		}

		for _, capture := range match.Captures {
			if state.importQuery.CaptureNameForId(capture.Index) == "path" {
				path := strings.Trim(capture.Node.Content(content), `"`)
				imports = append(imports, path)
			}
		}
	}

	return imports
}

// parseFunctions extracts function declarations from a parsed tree.
func (p *GoParser) parseFunctions(state *goParserState, tree *sitter.Tree, content []byte, file string, imports []string) []FunctionInfo {
	var functions []FunctionInfo
	state.functionCursor.Exec(state.functionQuery, tree.RootNode())

	for {
		match, ok := state.functionCursor.NextMatch()
		if !ok {
			break
		}

		var funcInfo FunctionInfo
		var returns []string

		for _, capture := range match.Captures {
			node := capture.Node
			switch state.functionQuery.CaptureNameForId(capture.Index) {
			case "func":
				funcInfo.StartLine = int(node.StartPoint().Row) + 1
				funcInfo.EndLine = int(node.EndPoint().Row) + 1
			case "name":
				funcInfo.Name = node.Content(content)
			case "result":
				returns = append(returns, node.Content(content))
			}
		}

		if funcInfo.Name != "" {
			funcInfo.File = file
			funcInfo.Returns = returns
			funcInfo.Imports = imports
			funcInfo.Dependencies = FunctionDependencies{
				Handlers:    []string{},
				Utilities:   []string{},
				Invocations: []string{},
			}
			functions = append(functions, funcInfo)
		}
	}

	return functions
}

// resolveAbsPath converts relative path to absolute path.
func (p *GoParser) resolveAbsPath(rootDir, file string) (string, error) {
	parts := strings.SplitN(file, "/", 3)
	if len(parts) < 3 {
		return filepath.Abs(file)
	}
	return filepath.Join(rootDir, parts[2]), nil
}

// -----------------------------------------------------------------------------
// Remaining Methods (DetectLanguages, BuildProjectMetadata, patternStrings)
// -----------------------------------------------------------------------------

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

// patternStrings converts a slice of compiled regexes to their string representations.
func patternStrings(patterns []*regexp.Regexp) []string {
	var out []string
	for _, pat := range patterns {
		out = append(out, pat.String())
	}
	return out
}
