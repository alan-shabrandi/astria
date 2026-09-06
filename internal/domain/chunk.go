package domain

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
}
