package parser

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/python"
)

// LanguageRegistry acts as a central hub for all supported Tree-sitter grammars.
type LanguageRegistry struct {
	grammars map[string]*sitter.Language
}

// NewLanguageRegistry initializes the registry with our pilot languages (Go & Python).
func NewLanguageRegistry() *LanguageRegistry {
	return &LanguageRegistry{
		grammars: map[string]*sitter.Language{
			".go": golang.GetLanguage(),
			".py": python.GetLanguage(),
		},
	}
}

// GetGrammar returns the appropriate Tree-sitter language based on file extension.
func (r *LanguageRegistry) GetGrammar(extension string) (*sitter.Language, error) {
	lang, exists := r.grammars[extension]
	if !exists {
		return nil, fmt.Errorf("unsupported language extension: %s", extension)
	}
	return lang, nil
}
