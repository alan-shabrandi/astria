package domain

// SourceFile represents a single source code file extracted from the target repository.
type SourceFile struct {
	Path      string // Absolute or relative path to the file
	Name      string // File name including extension
	Extension string // File extension (e.g., ".go", ".py")
	Size      int64  // File size in bytes
	Content   []byte // Raw content of the file
}
