package domain

import (
	"context"
)

type ChunkRepository interface {
	BatchInsert(ctx context.Context, repoName string, chunks []CodeChunk) error
}
