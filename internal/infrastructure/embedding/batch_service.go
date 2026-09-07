package embedding

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alanshabrandi/astria/internal/domain"
)

const (
	defaultBatchSize   = 100
	defaultMaxWorkers  = 5
	defaultRequestRate = 200 * time.Millisecond
)

type BatchEmbedder struct {
	provider    domain.EmbeddingProvider
	formatter   ChunkFormatter
	batchSize   int
	maxWorkers  int
	requestRate time.Duration
	payloadPool sync.Pool
}

func NewBatchEmbedder(provider domain.EmbeddingProvider, formatter ChunkFormatter, batchSize int) *BatchEmbedder {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	if formatter == nil {
		formatter = NewDefaultFormatter()
	}

	return &BatchEmbedder{
		provider:    provider,
		formatter:   formatter,
		batchSize:   batchSize,
		maxWorkers:  defaultMaxWorkers,
		requestRate: defaultRequestRate,
		payloadPool: sync.Pool{
			New: func() any {
				s := make([]string, 0, batchSize)
				return &s
			},
		},
	}
}

// EmbedChunks processes chunks concurrently with memory pooling and rate limiting.
func (b *BatchEmbedder) EmbedChunks(ctx context.Context, chunks []domain.CodeChunk) ([]domain.CodeChunk, error) {
	if len(chunks) == 0 {
		return chunks, nil
	}

	totalChunks := len(chunks)
	embeddedChunks := make([]domain.CodeChunk, totalChunks)
	copy(embeddedChunks, chunks)

	sem := make(chan struct{}, b.maxWorkers)

	ticker := time.NewTicker(b.requestRate)
	defer ticker.Stop()

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	for i := 0; i < totalChunks; i += b.batchSize {
		end := i + b.batchSize
		if end > totalChunks {
			end = totalChunks
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled before processing batch: %w", ctx.Err())
		case <-ticker.C:
		case err := <-errCh:
			return nil, err
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(startIndex, endIndex int) {
			defer wg.Done()
			defer func() { <-sem }()

			payloadsPtr := b.payloadPool.Get().(*[]string)
			payloads := (*payloadsPtr)[:0]

			defer b.payloadPool.Put(payloadsPtr)

			batch := embeddedChunks[startIndex:endIndex]

			for _, chunk := range batch {
				payloads = append(payloads, b.formatter.Format(chunk))
			}

			vectors, err := b.provider.GenerateEmbeddings(ctx, payloads)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("batch [%d:%d] failed: %w", startIndex, endIndex, err):
				default:
				}
				return
			}

			if len(vectors) != len(batch) {
				select {
				case errCh <- fmt.Errorf("vector length mismatch for batch [%d:%d]", startIndex, endIndex):
				default:
				}
				return
			}

			for j := range batch {
				embeddedChunks[startIndex+j].Vector = vectors[j]
			}
		}(i, end)
	}

	wg.Wait()
	close(errCh)

	if err := <-errCh; err != nil {
		return nil, err
	}

	return embeddedChunks, nil
}
