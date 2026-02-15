package copier

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Ignorer struct {
	patterns []string
}

func NewIgnorer(srcDir string, extraPatterns []string, useGitignore bool) *Ignorer {
	var patterns []string

	if useGitignore {
		gitignorePath := filepath.Join(srcDir, ".gitignore")
		if f, err := os.Open(gitignorePath); err == nil {
			defer f.Close()
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

	patterns = append(patterns, extraPatterns...)

	return &Ignorer{patterns: patterns}
}

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

func DeleteFile(dst string) error {
	err := os.Remove(dst)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
