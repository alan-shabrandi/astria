package embedding

import (
	"strings"

	"github.com/alanshabrandi/astria/internal/domain"
)

type ChunkFormatter interface {
	Format(chunk domain.CodeChunk) string
}

type DefaultFormatter struct{}

func NewDefaultFormatter() *DefaultFormatter {
	return &DefaultFormatter{}
}

// Format constructs a rich textual representation with zero fmt.Sprintf allocations.
func (f *DefaultFormatter) Format(chunk domain.CodeChunk) string {
	var sb strings.Builder

	sb.Grow(256 + len(chunk.Content))

	sb.WriteString("File: ")
	sb.WriteString(chunk.FilePath)
	sb.WriteString("\nType: ")
	sb.WriteString(string(chunk.Type))
	sb.WriteString("\nName: ")
	sb.WriteString(chunk.Name)
	sb.WriteString("\n")

	if chunk.Signature != "" {
		sb.WriteString("Signature: ")
		sb.WriteString(chunk.Signature)
		sb.WriteString("\n")
	}

	if chunk.DocComment != "" {
		sb.WriteString("Description: ")
		sb.WriteString(chunk.DocComment)
		sb.WriteString("\n")
	}

	sb.WriteString("\nCode:\n")
	sb.WriteString(chunk.Content)

	return sb.String()
}
