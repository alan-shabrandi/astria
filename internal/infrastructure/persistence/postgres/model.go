package postgres

import (
	"time"

	"github.com/alanshabrandi/astria/internal/domain"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type CodeChunkModel struct {
	ID          uuid.UUID       `db:"id"`
	RepoName    string          `db:"repo_name"`
	FilePath    string          `db:"file_path"`
	Language    string          `db:"language"`
	ChunkType   string          `db:"chunk_type"`
	Name        string          `db:"name"`
	Signature   string          `db:"signature"`
	DocComment  string          `db:"doc_comment"`
	Content     string          `db:"content"`
	StartLine   int32           `db:"start_line"`
	EndLine     int32           `db:"end_line"`
	ContentHash string          `db:"content_hash"`
	Embedding   pgvector.Vector `db:"embedding"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

func FromDomain(repoName string, chunk domain.CodeChunk) CodeChunkModel {
	return CodeChunkModel{
		ID:          uuid.New(),
		RepoName:    repoName,
		FilePath:    chunk.FilePath,
		Language:    chunk.Language,
		ChunkType:   string(chunk.Type),
		Name:        chunk.Name,
		Signature:   chunk.Signature,
		DocComment:  chunk.DocComment,
		Content:     chunk.Content,
		StartLine:   int32(chunk.StartLine),
		EndLine:     int32(chunk.EndLine),
		ContentHash: chunk.CalculateHash(),
		Embedding:   pgvector.NewVector(chunk.Vector),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
