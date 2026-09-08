package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ChunkType defines the structural category of the extracted code.
type ChunkType string

const (
	ChunkTypeFunction  ChunkType = "Function"
	ChunkTypeMethod    ChunkType = "Method"
	ChunkTypeStruct    ChunkType = "Struct"
	ChunkTypeInterface ChunkType = "Interface"
	ChunkTypeClass     ChunkType = "Class"
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
func (c *CodeChunk) PrepareForEmbedding() string {
	return fmt.Sprintf("Type: %s\nName: %s\nSignature: %s\nDoc: %s\n\n%s",
		c.Type, c.Name, c.Signature, c.DocComment, c.Content)
}

// CalculateHash generates a SHA-256 hash of the chunk's content and metadata
// to identify identical code segments across indexing runs.
func (c *CodeChunk) CalculateHash() string {
	hasher := sha256.New()
	data := fmt.Sprintf("%s:%s:%s", c.FilePath, c.Signature, c.Content)
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
