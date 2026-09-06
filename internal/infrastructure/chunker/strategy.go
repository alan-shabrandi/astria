package chunker

import (
	"fmt"
	"github.com/alanshabrandi/astria/internal/domain"
	"github.com/alanshabrandi/astria/internal/infrastructure/parser"
	sitter "github.com/smacker/go-tree-sitter"
)

// ChunkingStrategy defines the contract for language-specific chunking logic.
type ChunkingStrategy interface {
	Process(file domain.SourceFile, tree *sitter.Tree, extractor *parser.Extractor) ([]domain.CodeChunk, error)
}

// DefaultChunkingStrategy handles the standard mapping from Tree-sitter nodes to Domain chunks.
type DefaultChunkingStrategy struct {
	languageName string
}

func NewDefaultChunkingStrategy(lang string) *DefaultChunkingStrategy {
	return &DefaultChunkingStrategy{languageName: lang}
}

func (s *DefaultChunkingStrategy) Process(file domain.SourceFile, tree *sitter.Tree, extractor *parser.Extractor) ([]domain.CodeChunk, error) {
	var chunks []domain.CodeChunk

	structs, err := extractor.ExtractStructures(tree, file.Content, file.Extension)
	if err != nil {
		return nil, fmt.Errorf("failed to extract structures: %w", err)
	}

	for _, st := range structs {
		chunks = append(chunks, domain.CodeChunk{
			FilePath:   file.Path,
			Language:   s.languageName,
			Type:       domain.ChunkType(st.Type),
			Name:       st.Name,
			DocComment: st.DocComment,
			StartLine:  st.StartPoint.Row + 1, // Tree-sitter starts lines from 0
			EndLine:    st.EndPoint.Row + 1,
			Content:    st.Content,
		})
	}

	funcs, err := extractor.ExtractFunctions(tree, file.Content, file.Extension)
	if err != nil {
		return nil, fmt.Errorf("failed to extract functions: %w", err)
	}

	for _, fn := range funcs {
		chunkType := domain.ChunkTypeFunction
		if fn.IsMethod {
			chunkType = domain.ChunkTypeMethod
		}

		signature := fn.Parameters
		if fn.Returns != "" {
			signature += " -> " + fn.Returns
		}
		if fn.Receiver != "" {
			signature = fn.Receiver + " " + signature
		}

		chunks = append(chunks, domain.CodeChunk{
			FilePath:   file.Path,
			Language:   s.languageName,
			Type:       chunkType,
			Name:       fn.Name,
			DocComment: fn.DocComment,
			Signature:  signature,
			StartLine:  fn.StartPoint.Row + 1,
			EndLine:    fn.EndPoint.Row + 1,
			Content:    fn.Content,
		})
	}

	return chunks, nil
}
