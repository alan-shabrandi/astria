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
// It returns the parsed tree, a cleanup function to prevent CGO memory leaks, and an error.
// IMPORTANT: The caller MUST call the returned cleanup function (e.g., using defer).
func (e *Engine) Parse(ctx context.Context, content []byte, extension string) (*sitter.Tree, func(), error) {
	noop := func() {}

	lang, err := e.registry.GetGrammar(extension)
	if err != nil {
		return nil, noop, fmt.Errorf("failed to retrieve grammar for extension %s: %w", extension, err)
	}

	parser := sitter.NewParser()
	defer parser.Close()

	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, noop, fmt.Errorf("failed to parse syntax tree: %w", err)
	}

	cleanup := func() {
		if tree != nil {
			tree.Close()
		}
	}

	return tree, cleanup, nil
}
