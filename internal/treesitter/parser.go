package treesitter

// -----------------------------------------------------------------------------
// Types and Interface Definitions
// -----------------------------------------------------------------------------

// Language represents a programming language.
type Language string

// ProjectMetadata holds metadata about the project, such as detected languages,
// source file paths, and files to ignore (for example, as specified in .gitignore).
type ProjectMetadata struct {
	Languages    []Language `json:"languages"`
	SourceFiles  []string   `json:"source_files"`
	IgnoredFiles []string   `json:"ignored_files"`
}

// FunctionInfo represents an extracted function definition.
type FunctionInfo struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	// Additional fields such as parameters or return types can be added as needed.
}

// ProjectMap holds the function-level dependency map, including function definitions.
// Later you can expand this with information about import/export relationships and call hierarchies.
type ProjectMap struct {
	Functions []FunctionInfo `json:"functions"`
}

// Parser is the interface for tree-sitter–based parsing.
type Parser interface {
	// DetectLanguages scans the project and returns a list of detected languages.
	DetectLanguages(rootDir string) ([]Language, error)
	// BuildProjectMetadata builds and returns the project metadata.
	BuildProjectMetadata(rootDir string) (*ProjectMetadata, error)
	// BuildProjectMap builds and returns the project map (the function dependency map).
	BuildProjectMap(rootDir string) (*ProjectMap, error)
}

// -----------------------------------------------------------------------------
// Dummy Parser Implementation
// -----------------------------------------------------------------------------

// DummyParser is a stub implementation of the Parser interface.
type DummyParser struct{}

// NewDummyParser creates a new DummyParser instance.
func NewDummyParser() Parser {
	return &DummyParser{}
}

// DetectLanguages returns a dummy list of languages based on file extensions.
func (p *DummyParser) DetectLanguages(rootDir string) ([]Language, error) {
	// In a real implementation, you would scan the files in rootDir.
	return []Language{"go", "python", "javascript"}, nil
}

// BuildProjectMetadata builds dummy metadata.
// Later, this would scan rootDir, read .gitignore, etc.
func (p *DummyParser) BuildProjectMetadata(rootDir string) (*ProjectMetadata, error) {
	metadata := &ProjectMetadata{
		Languages:    []Language{"go"},
		SourceFiles:  []string{"internal/llm/server.go", "cmd/main.go"},
		IgnoredFiles: []string{"vendor/", "node_modules/"},
	}
	return metadata, nil
}

// BuildProjectMap builds a dummy project map.
// Later, you would replace this with AST parsing via tree-sitter for each source file.
func (p *DummyParser) BuildProjectMap(rootDir string) (*ProjectMap, error) {
	projectMap := &ProjectMap{
		Functions: []FunctionInfo{
			{
				Name:      "StartServer",
				File:      "internal/llm/server.go",
				StartLine: 10,
				EndLine:   50,
			},
			// Add additional dummy functions as needed.
		},
	}
	return projectMap, nil
}
