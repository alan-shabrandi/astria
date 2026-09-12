package scanner

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/alanshabrandi/astria/internal/domain"
)

type LocalScanner struct {
	workerCount int
}

func NewLocalScanner(workerCount int) *LocalScanner {
	if workerCount <= 0 {
		workerCount = 4
	}
	return &LocalScanner{
		workerCount: workerCount,
	}
}

// Scan matching domain.FileScanner interface
func (s *LocalScanner) Scan(ctx context.Context, rootDir string) <-chan domain.FileResult {
	resultsChan := make(chan domain.FileResult, s.workerCount*2)
	pathsChan := make(chan string, s.workerCount*2)

	filter := NewFilter(rootDir)
	var wg sync.WaitGroup

	// Worker Pool for reading files
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathsChan {
				if ctx.Err() != nil {
					return
				}

				info, err := os.Stat(path)
				if err != nil {
					s.sendResult(ctx, resultsChan, domain.FileResult{Err: err})
					continue
				}

				content, err := os.ReadFile(path)
				if err != nil {
					s.sendResult(ctx, resultsChan, domain.FileResult{Err: err})
					continue
				}

				sf := domain.SourceFile{
					Path:      path,
					Name:      info.Name(),
					Extension: filepath.Ext(path),
					Size:      info.Size(),
					Content:   content,
				}

				s.sendResult(ctx, resultsChan, domain.FileResult{File: sf})
			}
		}()
	}

	// Traversal Routine
	go func() {
		defer close(pathsChan)
		_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.IsDir() {
				if path != rootDir && filter.ShouldIgnore(path, true) {
					return filepath.SkipDir
				}
				return nil
			}
			if filter.ShouldIgnore(path, false) {
				return nil
			}

			select {
			case pathsChan <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}()

	// Cleanup Routine
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	return resultsChan
}

func (s *LocalScanner) sendResult(ctx context.Context, ch chan<- domain.FileResult, res domain.FileResult) {
	select {
	case ch <- res:
	case <-ctx.Done():
	}
}
