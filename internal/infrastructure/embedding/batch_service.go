package embedding

import (
	"context"
	"fmt"

	"github.com/alanshabrandi/astria/internal/domain"
)

const defaultBatchSize = 100

// BatchEmbedder orchestrates batching and embedding generation for CodeChunks.
type BatchEmbedder struct {
	provider  domain.EmbeddingProvider
	batchSize int
}

func NewBatchEmbedder(provider domain.EmbeddingProvider, batchSize int) *BatchEmbedder {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	return &BatchEmbedder{
		provider:  provider,
		batchSize: batchSize,
	}
}

// EmbedChunks processes chunks in batches, updates their Vector fields, and returns them.
func (b *BatchEmbedder) EmbedChunks(ctx context.Context, chunks []domain.CodeChunk) ([]domain.CodeChunk, error) {
	if len(chunks) == 0 {
		return chunks, nil
	}

	totalChunks := len(chunks)
	embeddedChunks := make([]domain.CodeChunk, totalChunks)
	copy(embeddedChunks, chunks)

	for i := 0; i < totalChunks; i += b.batchSize {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled during batch embedding processing: %w", err)
		}

		end := i + b.batchSize
		if end > totalChunks {
			end = totalChunks
		}

		batch := embeddedChunks[i:end]
		payloads := make([]string, len(batch))

		for j, chunk := range batch {
			payloads[j] = chunk.PrepareForEmbedding()
		}

		vectors, err := b.provider.GenerateEmbeddings(ctx, payloads)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings for batch [%d:%d]: %w", i, end, err)
		}

		if len(vectors) != len(batch) {
			return nil, fmt.Errorf("vector length mismatch: expected %d, got %d", len(batch), len(vectors))
		}

		for j := range batch {
			embeddedChunks[i+j].Vector = vectors[j]
		}
	}

	return embeddedChunks, nil
}
