package scanner

import (
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// Filter rules for skipping non-source files and system/binary directories.
type Filter struct {
	ignoredDirs map[string]struct{}
	ignoredExts map[string]struct{}
	gitIgnore   *ignore.GitIgnore
}

// NewFilter initializes default filters and loads root .gitignore if present.
func NewFilter(rootDir string) *Filter {
	f := &Filter{
		ignoredDirs: map[string]struct{}{
			".git":         {},
			"node_modules": {},
			"vendor":       {},
			"bin":          {},
			"obj":          {},
			".idea":        {},
			".vscode":      {},
			"dist":         {},
			"build":        {},
			"coverage":     {},
		},
		ignoredExts: map[string]struct{}{
			".exe": {}, ".dll": {}, ".so": {}, ".dylib": {},
			".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".svg": {}, ".ico": {},
			".pdf": {}, ".zip": {}, ".tar": {}, ".gz": {}, ".7z": {},
			".db": {}, ".sqlite": {}, ".bin": {}, ".log": {},
		},
	}

	f.loadGitIgnore(filepath.Join(rootDir, ".gitignore"))
	return f
}

// loadGitIgnore compiles the rules from the given .gitignore file path.
func (f *Filter) loadGitIgnore(gitIgnorePath string) {
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		return // No .gitignore found, silently continue
	}

	// CompileIgnoreFile safely parses the standard gitignore syntax
	ign, err := ignore.CompileIgnoreFile(gitIgnorePath)
	if err == nil {
		f.gitIgnore = ign
	}
}

// ShouldIgnore checks whether a file or directory path matches ignore rules.
func (f *Filter) ShouldIgnore(path string, isDir bool) bool {
	base := filepath.Base(path)

	// 1. Fast path: check hardcoded standard ignored directories
	if isDir {
		if _, exists := f.ignoredDirs[base]; exists {
			return true
		}
	} else {
		// 2. Fast path: check hardcoded standard ignored extensions
		ext := strings.ToLower(filepath.Ext(path))
		if _, exists := f.ignoredExts[ext]; exists {
			return true
		}
	}

	// 3. Robust path: check against compiled .gitignore rules
	if f.gitIgnore != nil {
		// MatchesPath safely evaluates Git Ignore patterns (like **, !, trailing slashes, etc.)
		return f.gitIgnore.MatchesPath(path)
	}

	return false
}
