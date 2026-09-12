package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/alanshabrandi/astria/internal/domain"
)

// IngestionCoordinator orchestrates the end-to-end flow of reading files,
// extracting AST chunks, generating embeddings, and saving to the database.
type IngestionCoordinator struct {
	scanner   domain.FileScanner
	chunker   domain.ASTChunker
	embedder  domain.EmbeddingProvider
	repo      domain.ChunkRepository
	batchSize int
	logger    *slog.Logger
}

func NewIngestionCoordinator(
	scanner domain.FileScanner,
	chunker domain.ASTChunker,
	embedder domain.EmbeddingProvider,
	repo domain.ChunkRepository,
	batchSize int,
	logger *slog.Logger,
) *IngestionCoordinator {
	if logger == nil {
		logger = slog.Default()
	}
	return &IngestionCoordinator{
		scanner:   scanner,
		chunker:   chunker,
		embedder:  embedder,
		repo:      repo,
		batchSize: batchSize,
		logger:    logger,
	}
}

// ProcessRepository is the main entry point for the pipeline.
func (c *IngestionCoordinator) ProcessRepository(ctx context.Context, repoPath string, repoName string) error {
	c.logger.Info("Starting ingestion for repository",
		slog.String("repo_name", repoName),
		slog.String("repo_path", repoPath),
	)

	// Stream file results using single-channel pattern to prevent leaks
	fileResults := c.scanner.Scan(ctx, repoPath)

	chunkBuffer := make([]domain.CodeChunk, 0, c.batchSize)

	for res := range fileResults {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("ingestion cancelled: %w", err)
		}

		if res.Err != nil {
			c.logger.Warn("Failed to scan file",
				slog.String("file", res.File.Path),
				slog.Any("error", res.Err),
			)
			continue
		}

		chunks, err := c.chunker.ChunkFile(ctx, res.File)
		if err != nil {
			c.logger.Warn("Failed to chunk file",
				slog.String("file", res.File.Path),
				slog.Any("error", err),
			)
			continue
		}

		for _, chunk := range chunks {
			chunkBuffer = append(chunkBuffer, chunk)

			if len(chunkBuffer) >= c.batchSize {
				if err := c.processBatch(ctx, repoName, chunkBuffer); err != nil {
					return fmt.Errorf("failed processing batch: %w", err)
				}
				// Reset slice length while maintaining allocated memory capacity
				chunkBuffer = chunkBuffer[:0]
			}
		}
	}

	// Flush remaining chunks in buffer
	if len(chunkBuffer) > 0 {
		if err := c.processBatch(ctx, repoName, chunkBuffer); err != nil {
			return fmt.Errorf("failed processing final batch: %w", err)
		}
	}

	c.logger.Info("Successfully ingested repository", slog.String("repo_name", repoName))
	return nil
}

// processBatch handles embedding generation and database insertion for a chunk batch.
func (c *IngestionCoordinator) processBatch(ctx context.Context, repoName string, chunks []domain.CodeChunk) error {
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		// Encapsulated method including Type, Name, Signature, DocComment, and Content
		texts[i] = chunk.PrepareForEmbedding()
	}

	// Generate embeddings with Exponential Backoff Retry mechanism
	vectors, err := c.generateEmbeddingsWithRetry(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings after retries: %w", err)
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

	c.logger.Info("Saved chunk batch", slog.Int("count", len(chunks)))
	return nil
}

// generateEmbeddingsWithRetry calls embedding provider using Exponential Backoff to handle Rate Limits (HTTP 429).
func (c *IngestionCoordinator) generateEmbeddingsWithRetry(ctx context.Context, texts []string) ([]domain.Vector, error) {
	const (
		maxAttempts = 5
		baseDelay   = 500 * time.Millisecond
		maxDelay    = 8 * time.Second
	)

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		vectors, err := c.embedder.GenerateEmbeddings(ctx, texts)
		if err == nil {
			return vectors, nil
		}

		lastErr = err

		if attempt == maxAttempts {
			break
		}

		// Calculate delay: baseDelay * 2^(attempt-1)
		delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
		if delay > maxDelay {
			delay = maxDelay
		}

		c.logger.Warn("Embedding generation failed, retrying with exponential backoff",
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", maxAttempts),
			slog.Duration("backoff_delay", delay),
			slog.Any("error", err),
		)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("exceeded max retry attempts (%d): %w", maxAttempts, lastErr)
}
