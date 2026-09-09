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

// EmbedChunks processes chunks concurrently with memory pooling, rate limiting, and safe error propagation.
func (b *BatchEmbedder) EmbedChunks(ctx context.Context, chunks []domain.CodeChunk) ([]domain.CodeChunk, error) {
	if len(chunks) == 0 {
		return chunks, nil
	}

	totalChunks := len(chunks)
	embeddedChunks := make([]domain.CodeChunk, totalChunks)
	copy(embeddedChunks, chunks)

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, b.maxWorkers)
	ticker := time.NewTicker(b.requestRate)
	defer ticker.Stop()

	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	setErr := func(err error) {
		once.Do(func() {
			firstErr = err
			cancel()
		})
	}

ProcessingLoop:
	for i := 0; i < totalChunks; i += b.batchSize {
		end := i + b.batchSize
		if end > totalChunks {
			end = totalChunks
		}

		select {
		case <-subCtx.Done():
			setErr(subCtx.Err())
			break ProcessingLoop
		case <-ticker.C:
		}

		if subCtx.Err() != nil {
			break ProcessingLoop
		}

		wg.Add(1)
		select {
		case sem <- struct{}{}:
		case <-subCtx.Done():
			wg.Done()
			setErr(subCtx.Err())
			break ProcessingLoop
		}

		if subCtx.Err() != nil {
			break ProcessingLoop
		}

		go func(startIndex, endIndex int) {
			defer wg.Done()
			defer func() { <-sem }()

			payloadsPtr := b.payloadPool.Get().(*[]string)
			payloads := (*payloadsPtr)[:0]

			batch := embeddedChunks[startIndex:endIndex]

			for _, chunk := range batch {
				payloads = append(payloads, b.formatter.Format(chunk))
			}

			*payloadsPtr = payloads
			defer b.payloadPool.Put(payloadsPtr)

			vectors, err := b.provider.GenerateEmbeddings(subCtx, payloads)
			if err != nil {
				setErr(fmt.Errorf("batch [%d:%d] failed: %w", startIndex, endIndex, err))
				return
			}

			if len(vectors) != len(batch) {
				setErr(fmt.Errorf("vector length mismatch for batch [%d:%d]", startIndex, endIndex))
				return
			}

			for j := range batch {
				embeddedChunks[startIndex+j].Vector = vectors[j]
			}
		}(i, end)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return embeddedChunks, nil
}
