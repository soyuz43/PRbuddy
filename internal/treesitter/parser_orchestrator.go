// internal/treesitter/parser_orchestrator.go

package treesitter

import (
	"fmt"
)

// NewParserForLanguage returns the appropriate parser based on the provided language.
// For now, if lang is "go", it returns a GoParser. You can extend this function
// to support additional languages by adding new cases.
func NewParserForLanguage(rootDir string, lang Language) (Parser, error) {
	switch lang {
	case "go":
		return NewGoParser(), nil
		// Future extensions:
		// case "python":
		//     return NewPythonParser(), nil
		// case "javascript":
		//     return NewJavaScriptParser(), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}
