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

func (s *LocalScanner) Scan(ctx context.Context, rootDir string) (<-chan domain.SourceFile, <-chan error) {
	filesChan := make(chan domain.SourceFile, s.workerCount*2)
	errChan := make(chan error, 10)
	pathsChan := make(chan string, s.workerCount*2)

	filter := NewFilter(rootDir)
	var wg sync.WaitGroup

	// Worker Pool
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

	// Traversal with early directory pruning and ignore rules
	go func() {
		defer close(pathsChan)

		err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Prune entire directory tree if ignored (e.g. .git, node_modules)
			if d.IsDir() {
				if path != rootDir && filter.ShouldIgnore(path, true) {
					return filepath.SkipDir
				}
				return nil
			}

			// Filter files based on extensions or .gitignore rules
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

		if err != nil && ctx.Err() == nil {
			s.sendError(ctx, errChan, err)
		}
	}()

	// Cleanup
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
