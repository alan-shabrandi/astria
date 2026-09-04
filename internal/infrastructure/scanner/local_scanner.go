package scanner

import (
	"context"
	"log/slog"
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

	// Worker Pool for reading files
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for path := range pathsChan {
				select {
				case <-ctx.Done():
					return
				default:
					info, err := os.Stat(path)
					if err != nil {
						slog.Warn("Failed to stat file, skipping", "path", path, "error", err.Error(), "worker", workerID)
						s.sendError(ctx, errChan, err)
						continue
					}

					content, err := os.ReadFile(path)
					if err != nil {
						slog.Warn("Failed to read file due to lock or permission, skipping", "path", path, "error", err.Error(), "worker", workerID)
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
		}(i)
	}

	// Traversal Routine
	go func() {
		defer close(pathsChan)

		err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
			// Day 5: Error handling for unreadable paths (e.g. Permission Denied)
			if err != nil {
				slog.Warn("Access denied or error accessing path during walk", "path", path, "error", err.Error())
				if d != nil && d.IsDir() {
					return filepath.SkipDir // Skip bad directories instead of crashing
				}
				return nil // Skip bad files
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			if d.IsDir() {
				if path != rootDir && filter.ShouldIgnore(path, true) {
					slog.Debug("Skipping directory based on filter rules", "dir", path)
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

		if err != nil && ctx.Err() == nil {
			slog.Error("Critical error during directory walk", "error", err.Error())
			s.sendError(ctx, errChan, err)
		}
	}()

	// Cleanup Routine
	go func() {
		wg.Wait()
		close(filesChan)
		close(errChan)
		slog.Info("File scanning completed successfully")
	}()

	return filesChan, errChan
}

func (s *LocalScanner) sendError(ctx context.Context, errChan chan<- error, err error) {
	select {
	case errChan <- err:
	case <-ctx.Done():
	}
}
