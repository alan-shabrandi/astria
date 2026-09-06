package chunker

import (
	"context"
	"strings"
	"testing"

	"github.com/alanshabrandi/astria/internal/domain"
	"github.com/alanshabrandi/astria/internal/infrastructure/parser"
)

func TestASTChunker_EdgeCases(t *testing.T) {
	registry := parser.NewLanguageRegistry()
	engine := parser.NewEngine(registry)
	extractor := parser.NewExtractor(registry)
	chunker := NewASTChunker(engine, extractor)

	complexGoCode := `
package complex

// ComplexCache is a generic struct representing an edge case.
// It supports comparable keys and any value types.
type ComplexCache[K comparable, V any] struct {
	items map[K]V
}

// Set adds an item to the cache.
// If the internal map is nil, it initializes it.
func (c *ComplexCache[K, V]) Set(key K, value V) error {
	if c.items == nil {
		c.items = make(map[K]V)
	}
	c.items[key] = value

	// Nested anonymous function (closure) - should not break the outer function extraction
	cleanup := func() {
		_ = "cleaning up..."
	}
	cleanup()

	return nil
}

/*
Processor interface handles generic processing.
It uses multiple lines of block comments.
*/
type Processor interface {
	Process(data []byte) (int, error)
}
`

	file := domain.SourceFile{
		Path:      "complex_edge_test.go",
		Name:      "complex_edge_test.go",
		Extension: ".go",
		Content:   []byte(complexGoCode),
	}

	ctx := context.Background()
	chunks, err := chunker.ChunkFile(ctx, file)

	if err != nil {
		t.Fatalf("ChunkFile failed on edge cases: %v", err)
	}

	expectedChunks := 3
	if len(chunks) != expectedChunks {
		t.Errorf("Expected %d chunks, got %d", expectedChunks, len(chunks))
		for _, c := range chunks {
			t.Logf("Found unexpected chunk: [%s] %s", c.Type, c.Name)
		}
	}

	foundMethod := false
	for _, chunk := range chunks {
		if chunk.Type == domain.ChunkTypeMethod && chunk.Name == "Set" {
			foundMethod = true

			if !strings.Contains(chunk.Signature, "(c *ComplexCache[K, V])") {
				t.Errorf("Method signature missing generic receiver: %s", chunk.Signature)
			}
			if !strings.Contains(chunk.Signature, "(key K, value V)") {
				t.Errorf("Method signature missing parameters: %s", chunk.Signature)
			}

			if !strings.Contains(chunk.DocComment, "Set adds an item") {
				t.Errorf("Method doc comment was not extracted properly: %q", chunk.DocComment)
			}

			if !strings.Contains(chunk.Content, "cleanup := func()") {
				t.Errorf("Method body is missing the nested closure function")
			}
		}
	}

	if !foundMethod {
		t.Errorf("Generic method 'Set' was completely missed by the extractor")
	}
}
