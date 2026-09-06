package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// Engine represents the Tree-sitter parsing engine capable of generating ASTs.
type Engine struct {
	registry *LanguageRegistry
}

// NewEngine creates a new instance of the parser engine with the given language registry.
func NewEngine(registry *LanguageRegistry) *Engine {
	return &Engine{
		registry: registry,
	}
}

// Parse generates an Abstract Syntax Tree (AST) for the given source code.
// IMPORTANT: The caller is responsible for calling Close() on the returned *sitter.Tree
// to prevent memory leaks in the underlying C library.
func (e *Engine) Parse(ctx context.Context, content []byte, extension string) (*sitter.Tree, error) {
	lang, err := e.registry.GetGrammar(extension)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve grammar for extension %s: %w", extension, err)
	}

	parser := sitter.NewParser()
	defer parser.Close()

	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse syntax tree: %w", err)
	}

	return tree, nil
}
