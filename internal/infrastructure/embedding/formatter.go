package embedding

import (
	"fmt"
	"strings"

	"github.com/alanshabrandi/astria/internal/domain"
)

// ChunkFormatter defines the contract for enriching code chunks before embedding.
type ChunkFormatter interface {
	Format(chunk domain.CodeChunk) string
}

// DefaultFormatter implements ChunkFormatter with optimized metadata ordering
// specifically designed for Transformer-based embedding models.
type DefaultFormatter struct{}

func NewDefaultFormatter() *DefaultFormatter {
	return &DefaultFormatter{}
}

// Format constructs a rich textual representation of a chunk.
// It places critical metadata at the top so that the embedding model's
// attention mechanism captures the context before the actual code syntax.
func (f *DefaultFormatter) Format(chunk domain.CodeChunk) string {
	var sb strings.Builder

	sb.Grow(256 + len(chunk.Content))

	sb.WriteString(fmt.Sprintf("File: %s\n", chunk.FilePath))

	sb.WriteString(fmt.Sprintf("Type: %s\n", chunk.Type))

	sb.WriteString(fmt.Sprintf("Name: %s\n", chunk.Name))

	if chunk.Signature != "" {
		sb.WriteString(fmt.Sprintf("Signature: %s\n", chunk.Signature))
	}

	if chunk.DocComment != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", chunk.DocComment))
	}

	sb.WriteString("\nCode:\n")
	sb.WriteString(chunk.Content)

	return sb.String()
}
