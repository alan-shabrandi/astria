//go:build integration

package embedding_test

import (
	"context"
	"os"
	"testing"

	"github.com/alanshabrandi/astria/internal/domain"
	"github.com/alanshabrandi/astria/internal/infrastructure/embedding"
)

func TestBatchEmbedder_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: OPENAI_API_KEY environment variable is not set")
	}

	provider := embedding.NewOpenAIProvider(apiKey)
	formatter := embedding.NewDefaultFormatter()
	embedder := embedding.NewBatchEmbedder(provider, formatter, 10)

	sampleChunks := []domain.CodeChunk{
		{
			FilePath:   "internal/domain/user.go",
			Language:   "go",
			Type:       domain.ChunkTypeStruct,
			Name:       "User",
			DocComment: "User represents an authenticated system user in the Astria platform.",
			Signature:  "type User struct",
			StartLine:  10,
			EndLine:    15,
			Content:    "type User struct {\n\tID int64\n\tEmail string\n}",
		},
		{
			FilePath:   "internal/domain/user.go",
			Language:   "go",
			Type:       domain.ChunkTypeFunction,
			Name:       "NewUser",
			DocComment: "NewUser constructs and validates a new User instance.",
			Signature:  "func NewUser(email string) (*User, error)",
			StartLine:  17,
			EndLine:    25,
			Content:    "func NewUser(email string) (*User, error) {\n\treturn &User{Email: email}, nil\n}",
		},
	}

	ctx := context.Background()
	embeddedChunks, err := embedder.EmbedChunks(ctx, sampleChunks)

	if err != nil {
		t.Fatalf("Batch embedding failed with error: %v", err)
	}

	if len(embeddedChunks) != len(sampleChunks) {
		t.Fatalf("Expected %d embedded chunks, got %d", len(sampleChunks), len(embeddedChunks))
	}

	expectedDimensions := provider.Dimensions()

	for i, chunk := range embeddedChunks {
		if len(chunk.Vector) != expectedDimensions {
			t.Errorf("Chunk [%d] (%s): expected vector dimension %d, got %d",
				i, chunk.Name, expectedDimensions, len(chunk.Vector))
		}

		hasNonZero := false
		for _, v := range chunk.Vector {
			if v != 0 {
				hasNonZero = true
				break
			}
		}
		if !hasNonZero {
			t.Errorf("Chunk [%d] (%s): generated vector is all zeros", i, chunk.Name)
		}

		t.Logf("✅ Successfully embedded Chunk [%d] '%s' (%s)\n"+
			"   Dimensions: %d\n"+
			"   Sample Float32 Values (First 5): %v\n",
			i, chunk.Name, chunk.Type, len(chunk.Vector), chunk.Vector[:5])
	}
}
