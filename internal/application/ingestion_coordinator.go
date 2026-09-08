package application

import (
	"context"
	"fmt"
	"log"

	"github.com/alanshabrandi/astria/internal/domain"
)

// IngestionCoordinator orchestrates the end-to-end flow of reading files,
// extracting AST chunks, generating embeddings, and saving to the database.
type IngestionCoordinator struct {
	scanner   domain.FileScanner
	chunker   domain.ASTChunker
	embedder  domain.EmbeddingService
	repo      domain.ChunkRepository
	batchSize int
}

func NewIngestionCoordinator(
	scanner domain.FileScanner,
	chunker domain.ASTChunker,
	embedder domain.EmbeddingService,
	repo domain.ChunkRepository,
	batchSize int,
) *IngestionCoordinator {
	return &IngestionCoordinator{
		scanner:   scanner,
		chunker:   chunker,
		embedder:  embedder,
		repo:      repo,
		batchSize: batchSize,
	}
}

// ProcessRepository is the main entry point for the pipeline.
func (c *IngestionCoordinator) ProcessRepository(ctx context.Context, repoPath string, repoName string) error {
	log.Printf("Starting ingestion for repository: %s at %s", repoName, repoPath)

	files, errCh := c.scanner.Scan(ctx, repoPath)

	var chunkBuffer []domain.CodeChunk

	for file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		chunks, err := c.chunker.ChunkFile(ctx, file)
		if err != nil {
			log.Printf("Warning: failed to chunk file %v: %v", file, err)
			continue
		}

		chunkBuffer = append(chunkBuffer, chunks...)

		for len(chunkBuffer) >= c.batchSize {
			batch := chunkBuffer[:c.batchSize]
			chunkBuffer = chunkBuffer[c.batchSize:]

			if err := c.processBatch(ctx, repoName, batch); err != nil {
				return err
			}
		}
	}

	if len(chunkBuffer) > 0 {
		if err := c.processBatch(ctx, repoName, chunkBuffer); err != nil {
			return err
		}
	}

	if err := <-errCh; err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	log.Printf("Successfully ingested repository: %s", repoName)
	return nil
}

// processBatch handles embedding generation and database insertion for a chunk batch.
func (c *IngestionCoordinator) processBatch(ctx context.Context, repoName string, chunks []domain.CodeChunk) error {
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = fmt.Sprintf("%s\n%s", chunk.Signature, chunk.Content)
	}

	vectors, err := c.embedder.GenerateEmbeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	if len(vectors) != len(chunks) {
		return fmt.Errorf("embedding count mismatch: expected %d, got %d", len(chunks), len(vectors))
	}

	for i := range chunks {
		chunks[i].Vector = vectors[i]
	}

	if err := c.repo.BatchInsert(ctx, repoName, chunks); err != nil {
		return fmt.Errorf("failed to insert chunks into db: %w", err)
	}

	log.Printf("Successfully processed and saved batch of %d chunks", len(chunks))
	return nil
}
