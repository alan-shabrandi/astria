package chunker

import (
	"context"
	"fmt"

	"github.com/alanshabrandi/astria/internal/domain"
	"github.com/alanshabrandi/astria/internal/infrastructure/parser"
)

// ASTChunker coordinates the parsing and chunking of source files.
type ASTChunker struct {
	engine     *parser.Engine
	extractor  *parser.Extractor
	strategies map[string]ChunkingStrategy
}

func NewASTChunker(engine *parser.Engine, extractor *parser.Extractor) *ASTChunker {
	return &ASTChunker{
		engine:    engine,
		extractor: extractor,
		strategies: map[string]ChunkingStrategy{
			".go": NewDefaultChunkingStrategy("Go"),
			".py": NewDefaultChunkingStrategy("Python"),
		},
	}
}

// ChunkFile takes a raw SourceFile, parses the AST, and applies the mapped strategy.
func (c *ASTChunker) ChunkFile(ctx context.Context, file domain.SourceFile) ([]domain.CodeChunk, error) {
	strategy, exists := c.strategies[file.Extension]
	if !exists {
		return nil, nil
	}

	tree, cleanup, err := c.engine.Parse(ctx, file.Content, file.Extension)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", file.Path, err)
	}

	defer cleanup()

	return strategy.Process(file, tree, c.extractor)
}
