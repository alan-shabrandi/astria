package domain

import "context"

// FileScanner defines the contract for traversing directories and extracting source files.
type FileScanner interface {
	Scan(ctx context.Context, rootDir string) (<-chan SourceFile, <-chan error)
}

type ASTChunker interface {
	ChunkFile(ctx context.Context, file SourceFile) ([]CodeChunk, error)
}
type EmbeddingService interface {
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}
