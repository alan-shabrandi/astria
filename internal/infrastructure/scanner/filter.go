package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Filter rules for skipping non-source files and system/binary directories.
type Filter struct {
	ignoredDirs    map[string]struct{}
	ignoredExts    map[string]struct{}
	gitIgnoreRules []string
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

func (f *Filter) loadGitIgnore(gitIgnorePath string) {
	file, err := os.Open(gitIgnorePath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f.gitIgnoreRules = append(f.gitIgnoreRules, line)
	}
	if err := scanner.Err(); err != nil {
		return
	}
}

// ShouldIgnore checks whether a file or directory path matches ignore rules.
func (f *Filter) ShouldIgnore(path string, isDir bool) bool {
	base := filepath.Base(path)

	if isDir {
		if _, exists := f.ignoredDirs[base]; exists {
			return true
		}
	} else {
		ext := strings.ToLower(filepath.Ext(path))
		if _, exists := f.ignoredExts[ext]; exists {
			return true
		}
	}

	for _, rule := range f.gitIgnoreRules {
		cleanRule := strings.TrimPrefix(strings.TrimSuffix(rule, "/"), "/")
		if base == cleanRule || strings.Contains(path, filepath.FromSlash(cleanRule)) {
			return true
		}
		if matched, _ := filepath.Match(rule, base); matched {
			return true
		}
	}

	return false
}
