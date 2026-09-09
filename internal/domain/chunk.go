package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
)

// ChunkType defines the structural category of the extracted code.
type ChunkType string

const (
	ChunkTypeFunction  ChunkType = "Function"
	ChunkTypeMethod    ChunkType = "Method"
	ChunkTypeStruct    ChunkType = "Struct"
	ChunkTypeInterface ChunkType = "Interface"
	ChunkTypeClass     ChunkType = "Class"
	// Added for Code Graph RAG support
	ChunkTypeImport   ChunkType = "Import"
	ChunkTypeVariable ChunkType = "Variable"
	ChunkTypeUnknown  ChunkType = "Unknown"
)

// CodeChunk is the unified domain model representing an extractable, embeddable piece of code.
type CodeChunk struct {
	FilePath   string
	Language   string
	Type       ChunkType
	Name       string
	DocComment string
	Signature  string
	StartLine  uint32
	EndLine    uint32
	Content    string
	Vector     Vector
}

// PrepareForEmbedding prepares a context-rich string for the embedding model.
// Refactored using strings.Builder for lower memory footprint on large files.
func (c *CodeChunk) PrepareForEmbedding() string {
	var builder strings.Builder
	// Approximate capacity to avoid slice reallocations
	builder.Grow(100 + len(c.Type) + len(c.Name) + len(c.Signature) + len(c.DocComment) + len(c.Content))

	builder.WriteString("Type: ")
	builder.WriteString(string(c.Type))
	builder.WriteString("\nName: ")
	builder.WriteString(c.Name)
	builder.WriteString("\nSignature: ")
	builder.WriteString(c.Signature)
	builder.WriteString("\nDoc: ")
	builder.WriteString(c.DocComment)
	builder.WriteString("\n\n")
	builder.WriteString(c.Content)

	return builder.String()
}

// CalculateHash generates a SHA-256 hash.
// Refactored to eliminate fmt.Sprintf allocations which reduces GC pressure
// when indexing large codebases with thousands of chunks.
func (c *CodeChunk) CalculateHash() string {
	hasher := sha256.New()

	io.WriteString(hasher, c.FilePath)
	io.WriteString(hasher, ":")
	io.WriteString(hasher, c.Signature)
	io.WriteString(hasher, ":")
	io.WriteString(hasher, c.Content)

	return hex.EncodeToString(hasher.Sum(nil))
}
