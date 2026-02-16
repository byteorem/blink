// Package copier handles file synchronization and ignore-pattern matching.
package copier

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	cp "github.com/otiai10/copy"
	ignore "github.com/sabhiram/go-gitignore"
)

// Ignorer determines which files should be excluded from syncing.
type Ignorer struct {
	gi *ignore.GitIgnore
}

// NewIgnorer creates an Ignorer from .gitignore, .pkgmeta (if enabled), and extra patterns.
func NewIgnorer(srcDir string, extraPatterns []string, useGitignore bool, usePkgMeta bool) *Ignorer {
	patterns := []string{"blink.toml", ".git"}

	if useGitignore {
		gitignorePath := filepath.Join(srcDir, ".gitignore")
		if f, err := os.Open(gitignorePath); err == nil {
			defer func() { _ = f.Close() }()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				patterns = append(patterns, line)
			}
		}
	}

	if usePkgMeta {
		patterns = append(patterns, parsePkgMetaIgnore(srcDir)...)
	}

	patterns = append(patterns, extraPatterns...)

	return &Ignorer{gi: ignore.CompileIgnoreLines(patterns...)}
}

// parsePkgMetaIgnore reads .pkgmeta and extracts patterns from the ignore: block.
func parsePkgMetaIgnore(srcDir string) []string {
	f, err := os.Open(filepath.Join(srcDir, ".pkgmeta"))
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var patterns []string
	inIgnore := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "ignore:" {
			inIgnore = true
			continue
		}
		if inIgnore {
			if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "    - ") || strings.HasPrefix(line, "\t- ") {
				pattern := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				if pattern != "" {
					patterns = append(patterns, pattern)
				}
			} else if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inIgnore = false
			}
		}
	}
	return patterns
}

// ShouldIgnore reports whether the given relative path should be excluded.
func (ig *Ignorer) ShouldIgnore(relPath string) bool {
	if ig.gi.MatchesPath(relPath) {
		return true
	}
	// Also check with trailing slash so directory-only patterns (e.g. "node_modules/")
	// match the directory path itself, not just its children.
	if !strings.HasSuffix(relPath, "/") {
		return ig.gi.MatchesPath(relPath + "/")
	}
	return false
}

// CountFiles returns the number of non-ignored files under src.
func CountFiles(src string, ig *Ignorer) (int, error) {
	count := 0
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		if ig.ShouldIgnore(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

// InitialSyncWithProgress copies files from src to dst, calling onFile after each file.
func InitialSyncWithProgress(src, dst string, ig *Ignorer, onFile func(copied int)) (int, error) {
	count := 0
	err := cp.Copy(src, dst, cp.Options{
		Skip: func(info os.FileInfo, srcPath, _ string) (bool, error) {
			rel, err := filepath.Rel(src, srcPath)
			if err != nil || rel == "." {
				return false, nil
			}
			if ig.ShouldIgnore(rel) {
				return true, nil
			}
			if !info.IsDir() {
				count++
				if onFile != nil {
					onFile(count)
				}
			}
			return false, nil
		},
		OnDirExists: func(_, _ string) cp.DirExistsAction {
			return cp.Merge
		},
	})
	return count, err
}

// InitialSync copies all non-ignored files from src to dst.
func InitialSync(src, dst string, ig *Ignorer) (int, error) {
	count := 0
	err := cp.Copy(src, dst, cp.Options{
		Skip: func(info os.FileInfo, srcPath, _ string) (bool, error) {
			rel, err := filepath.Rel(src, srcPath)
			if err != nil || rel == "." {
				return false, nil
			}
			if ig.ShouldIgnore(rel) {
				return true, nil
			}
			if !info.IsDir() {
				count++
			}
			return false, nil
		},
		OnDirExists: func(_, _ string) cp.DirExistsAction {
			return cp.Merge
		},
	})
	return count, err
}

// CopyFile copies a single file from src to dst, creating directories as needed.
func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0o644)
}

// CleanDestination removes files from dst that don't exist in src or match
// ignore rules. Returns the count of removed files.
func CleanDestination(src, dst string, ig *Ignorer) (int, error) {
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return 0, nil
	}

	removed := 0

	// First pass: remove stale files
	err := filepath.WalkDir(dst, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(dst, path)
		if err != nil {
			return err
		}
		if relPath == "." || d.IsDir() {
			return nil
		}
		shouldRemove := false
		if ig != nil && ig.ShouldIgnore(relPath) {
			shouldRemove = true
		} else {
			srcPath := filepath.Join(src, relPath)
			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				shouldRemove = true
			}
		}
		if shouldRemove {
			if err := os.Remove(path); err != nil {
				return err
			}
			removed++
		}
		return nil
	})
	if err != nil {
		return removed, err
	}

	// Second pass: remove empty directories (bottom-up)
	var dirs []string
	_ = filepath.WalkDir(dst, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == dst {
			return nil
		}
		dirs = append(dirs, path)
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		entries, err := os.ReadDir(dirs[i])
		if err == nil && len(entries) == 0 {
			_ = os.Remove(dirs[i])
		}
	}

	return removed, nil
}

// DeleteFile removes the file at dst, returning nil if it does not exist.
func DeleteFile(dst string) error {
	err := os.Remove(dst)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
