// Package copier handles file synchronization and ignore-pattern matching.
package copier

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Ignorer determines which files should be excluded from syncing.
type Ignorer struct {
	patterns []string
}

// NewIgnorer creates an Ignorer from .gitignore, .pkgmeta (if enabled), and extra patterns.
func NewIgnorer(srcDir string, extraPatterns []string, useGitignore bool, usePkgMeta bool) *Ignorer {
	var patterns []string

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

	return &Ignorer{patterns: patterns}
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
	// Always ignore .git/ and blink.toml
	if relPath == "blink.toml" || relPath == ".git" || strings.HasPrefix(relPath, ".git/") || strings.HasPrefix(relPath, ".git\\") {
		return true
	}

	base := filepath.Base(relPath)

	for _, pattern := range ig.patterns {
		// Directory pattern (trailing slash)
		if strings.HasSuffix(pattern, "/") {
			dirName := strings.TrimSuffix(pattern, "/")
			// Check if the relPath starts with this dir or contains it as a segment
			if relPath == dirName || strings.HasPrefix(relPath, dirName+"/") || strings.HasPrefix(relPath, dirName+"\\") {
				return true
			}
			// Check path segments
			for _, seg := range strings.Split(filepath.ToSlash(relPath), "/") {
				if seg == dirName {
					return true
				}
			}
			continue
		}

		// Match against full relative path and base name
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
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
		if d.IsDir() {
			return nil
		}
		dstPath := filepath.Join(dst, relPath)
		if err := CopyFile(path, dstPath); err != nil {
			return err
		}
		count++
		if onFile != nil {
			onFile(count)
		}
		return nil
	})
	return count, err
}

// InitialSync copies all non-ignored files from src to dst.
func InitialSync(src, dst string, ig *Ignorer) (int, error) {
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

		if d.IsDir() {
			return nil
		}

		dstPath := filepath.Join(dst, relPath)
		if err := CopyFile(path, dstPath); err != nil {
			return err
		}
		count++
		return nil
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

// DeleteFile removes the file at dst, returning nil if it does not exist.
func DeleteFile(dst string) error {
	err := os.Remove(dst)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
