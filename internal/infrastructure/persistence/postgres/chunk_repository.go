package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/alanshabrandi/astria/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type ChunkRepository struct {
	pool *pgxpool.Pool
}

func NewChunkRepository(pool *pgxpool.Pool) *ChunkRepository {
	return &ChunkRepository{
		pool: pool,
	}
}

// BatchInsert inserts a list of code chunks into PostgreSQL concurrently using pgx.Batch.
// It uses ON CONFLICT DO NOTHING based on (repo_name, content_hash) for idempotency.
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

	batch := &pgx.Batch{}
	now := time.Now()

	for _, chunk := range chunks {
		contentHash := chunk.CalculateHash()

		vectorParam := pgvector.NewVector(chunk.Vector)

		batch.Queue(
			query,
			repoName,
			chunk.FilePath,
			chunk.Language,
			string(chunk.Type),
			chunk.Name,
			chunk.Signature,
			chunk.DocComment,
			chunk.Content,
			chunk.StartLine,
			chunk.EndLine,
			contentHash,
			vectorParam,
			now,
			now,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(chunks); i++ {
		ct, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute chunk insert at index %d: %w", i, err)
		}
		_ = ct
	}

	return nil
}
