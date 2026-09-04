package domain

import "context"

// FileScanner defines the contract for traversing directories and extracting source files.
type FileScanner interface {
	Scan(ctx context.Context, rootDir string) (<-chan SourceFile, <-chan error)
}
