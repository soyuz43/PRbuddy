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

// GoParser implements Parser for Go projects using Tree-sitter.
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
	functionCursor *sitter.QueryCursor
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

	funcQuery, err := sitter.NewQuery([]byte(`
(function_declaration
  name: (identifier) @name
  parameters: (parameter_list
    (parameter_declaration
      name: (identifier)? @param_name
      type: (_) @param_type
    )*
  ) @parameters
  result: (parameter_list
    (parameter_declaration
      name: (identifier)? @return_name
      type: (_) @return_type
    )*
  )? @results
  body: (block) @body
) @func
  `), golang.GetLanguage())
	if err != nil {
		return nil, fmt.Errorf("failed to create function query: %w", err)
	}

	return &goParserState{
		parser:         parser,
		functionQuery:  funcQuery,
		functionCursor: sitter.NewQueryCursor(),
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

	// Parse the file content into a syntax tree (CST) in memory.
	tree, err := state.parser.ParseCtx(context.Background(), nil, content)
	if err != nil || tree == nil {
		return nil, fmt.Errorf("failed to parse %s: %w", absPath, err)
	}

	// === Dump the syntax tree for inspection ===
	if err := saveSyntaxTree(file, tree, content); err != nil {
		fmt.Printf("Warning: Failed to save syntax tree for %s: %s\n", absPath, err)
	}

	// Extract function metadata and dependencies
	functions := p.parseFunctions(state, tree, content, file)

	return functions, nil
}

// parseFunctions extracts function declarations and dependencies from a parsed tree.
func (p *GoParser) parseFunctions(state *goParserState, tree *sitter.Tree, content []byte, file string) []FunctionInfo {
	var functions []FunctionInfo
	state.functionCursor.Exec(state.functionQuery, tree.RootNode())

	for {
		match, ok := state.functionCursor.NextMatch()
		if !ok {
			break
		}

		var funcInfo FunctionInfo
		var returns []string
		var bodyNode *sitter.Node

		// Process captures from the function query.
		for _, capture := range match.Captures {
			node := capture.Node
			switch state.functionQuery.CaptureNameForId(capture.Index) {
			case "name":
				funcInfo.Name = string(node.Content(content))
			case "return_type":
				returns = append(returns, string(node.Content(content)))
			case "body":
				bodyNode = node
				funcInfo.StartLine = int(node.Parent().StartPoint().Row) + 1
				funcInfo.EndLine = int(node.Parent().EndPoint().Row) + 1
			}
		}
		funcInfo.Returns = returns
		funcInfo.File = file

		// Initialize dependencies.
		funcInfo.Dependencies = FunctionDependencies{}

		// Extract function dependencies.
		if bodyNode != nil {
			depQuery, err := sitter.NewQuery([]byte(`
				(call_expression
					function: (identifier) @invocation
				)
			`), golang.GetLanguage())

			if err == nil {
				depCursor := sitter.NewQueryCursor()
				depCursor.Exec(depQuery, bodyNode)

				for {
					depMatch, ok := depCursor.NextMatch()
					if !ok {
						break
					}
					for _, depCapture := range depMatch.Captures {
						if depQuery.CaptureNameForId(depCapture.Index) == "invocation" {
							invocationName := string(depCapture.Node.Content(content))

							// Check if the function is locally defined in the same file (utility function).
							isUtility := false
							for _, f := range functions {
								if f.Name == invocationName {
									isUtility = true
									break
								}
							}

							// Categorize dependencies
							if strings.HasPrefix(invocationName, "Handle") {
								funcInfo.Dependencies.Handlers = append(funcInfo.Dependencies.Handlers, invocationName)
							} else if isUtility {
								funcInfo.Dependencies.Utilities = append(funcInfo.Dependencies.Utilities, invocationName)
							} else {
								funcInfo.Dependencies.Invocations = append(funcInfo.Dependencies.Invocations, invocationName)
							}
						}
					}
				}
			}
		}

		if funcInfo.Name != "" {
			functions = append(functions, funcInfo)
		}
	}

	return functions
}

// resolveAbsPath converts a relative path to an absolute path.
func (p *GoParser) resolveAbsPath(rootDir, file string) (string, error) {
	parts := strings.SplitN(file, "/", 3)
	if len(parts) < 3 {
		return filepath.Abs(file)
	}
	return filepath.Join(rootDir, parts[2]), nil
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
	// Read .gitignore patterns.
	patterns, err := utils.ReadGitignore(rootDir)
	if err != nil {
		// If .gitignore doesn't exist or fails to open, proceed with no patterns.
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

// -----------------------------------------------------------------------------
// Dump Tree Utilities
// -----------------------------------------------------------------------------

// dumpTree recursively builds an indented string representation of the syntax tree.
// It includes each node's type and its start/end positions.
func dumpTree(node *sitter.Node, source []byte, indent string) string {
	result := fmt.Sprintf("%s%s [%d:%d - %d:%d]\n",
		indent,
		node.Type(),
		node.StartPoint().Row, node.StartPoint().Column,
		node.EndPoint().Row, node.EndPoint().Column)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		result += dumpTree(child, source, indent+"  ")
	}
	return result
}

// saveSyntaxTree saves the full syntax tree to a file for inspection.
// The file is written to .git/prbuddy_db/scaffold/<filename>_tree.txt.
func saveSyntaxTree(file string, tree *sitter.Tree, source []byte) error {
	scaffoldDir := filepath.Join(".git", "prbuddy_db", "scaffold")
	if err := os.MkdirAll(scaffoldDir, 0755); err != nil {
		return fmt.Errorf("failed to create scaffold directory: %w", err)
	}
	outputFile := filepath.Join(scaffoldDir, filepath.Base(file)+"_tree.txt")
	treeDump := dumpTree(tree.RootNode(), source, "")
	if err := os.WriteFile(outputFile, []byte(treeDump), 0644); err != nil {
		return fmt.Errorf("failed to write syntax tree to file: %w", err)
	}
	return nil
}
