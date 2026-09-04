package scanner

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/alanshabrandi/astria/internal/domain"
)

// LocalScanner implements the domain.FileScanner interface using a concurrent Worker Pool pattern.
type LocalScanner struct {
	workerCount int
}

// NewLocalScanner creates a new instance of LocalScanner with a specified pool size.
func NewLocalScanner(workerCount int) *LocalScanner {
	if workerCount <= 0 {
		workerCount = 4
	}
	return &LocalScanner{
		workerCount: workerCount,
	}
}

// Scan walks the given directory and uses a pool of workers to concurrently read file contents.
func (s *LocalScanner) Scan(ctx context.Context, rootDir string) (<-chan domain.SourceFile, <-chan error) {
	filesChan := make(chan domain.SourceFile, s.workerCount*2)
	errChan := make(chan error, 10)
	pathsChan := make(chan string, s.workerCount*2)

	var wg sync.WaitGroup

	// 1. Worker Pool to process file reading concurrently
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathsChan {
				select {
				case <-ctx.Done():
					return
				default:
					info, err := os.Stat(path)
					if err != nil {
						s.sendError(ctx, errChan, err)
						continue
					}

					content, err := os.ReadFile(path)
					if err != nil {
						s.sendError(ctx, errChan, err)
						continue
					}

					sf := domain.SourceFile{
						Path:      path,
						Name:      info.Name(),
						Extension: filepath.Ext(path),
						Size:      info.Size(),
						Content:   content,
					}

					select {
					case filesChan <- sf:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// 2. Traversal Routine to stream file paths into worker pool
	go func() {
		defer close(pathsChan)

		err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if !d.IsDir() {
				select {
				case pathsChan <- path:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})

		if err != nil && ctx.Err() == nil {
			s.sendError(ctx, errChan, err)
		}
	}()

	// 3. Cleanup Routine to wait for workers and safely close output channels
	go func() {
		wg.Wait()
		close(filesChan)
		close(errChan)
	}()

	return filesChan, errChan
}

func (s *LocalScanner) sendError(ctx context.Context, errChan chan<- error, err error) {
	select {
	case errChan <- err:
	case <-ctx.Done():
	}
}
