package domain

import "context"

// FileScanner defines the contract for traversing directories and extracting source files.
type FileScanner interface {
	// Scan uses a single channel with FileResult to prevent goroutine leaks
	// and simplify consumer loops.
	Scan(ctx context.Context, rootDir string) <-chan FileResult
}
