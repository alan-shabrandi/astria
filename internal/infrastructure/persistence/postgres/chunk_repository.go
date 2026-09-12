package postgres

import (
	"context"
	"fmt"

	"github.com/alanshabrandi/astria/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxBatchSize = 500

type ChunkRepository struct {
	pool *pgxpool.Pool
}

func NewChunkRepository(pool *pgxpool.Pool) *ChunkRepository {
	return &ChunkRepository{
		pool: pool,
	}
}

// BatchInsert inserts a list of code chunks into PostgreSQL concurrently using pgx.Batch.
// It automatically splits large slices into smaller batches to prevent memory spikes and timeouts.
func (r *ChunkRepository) BatchInsert(ctx context.Context, repoName string, chunks []domain.CodeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	query := `
		INSERT INTO code_chunks (
			repo_name, file_path, language, chunk_type, name,
			signature, doc_comment, content, start_line, end_line,
			content_hash, embedding, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14
		)
		ON CONFLICT (repo_name, content_hash) DO NOTHING;
	`

	for i := 0; i < len(chunks); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		currentBatchChunks := chunks[i:end]
		batch := &pgx.Batch{}

		for _, chunk := range currentBatchChunks {
			m := FromDomain(repoName, chunk)
			batch.Queue(
				query,
				m.RepoName, m.FilePath, m.Language, m.ChunkType, m.Name,
				m.Signature, m.DocComment, m.Content, m.StartLine, m.EndLine,
				m.ContentHash, m.Embedding, m.CreatedAt, m.UpdatedAt,
			)
		}

		if err := r.executeBatch(ctx, batch, len(currentBatchChunks)); err != nil {
			return fmt.Errorf("failed to process chunk batch starting at index %d: %w", i, err)
		}
	}

	return nil
}

// executeBatch is a helper method to safely execute a batch and ensure br.Close()
// is called immediately after execution, preventing resource leaks in loops.
func (r *ChunkRepository) executeBatch(ctx context.Context, batch *pgx.Batch, count int) error {
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to execute chunk insert at batch index %d: %w", i, err)
		}
	}

	return nil
}
