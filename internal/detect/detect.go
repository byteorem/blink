package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindAddon(sourceFlag string) (srcDir string, addonName string, err error) {
	if sourceFlag != "" && sourceFlag != "auto" {
		srcDir, err = filepath.Abs(sourceFlag)
		if err != nil {
			return "", "", fmt.Errorf("invalid source path: %w", err)
		}
		addonName = filepath.Base(srcDir)

		// Try to derive addon name from .toc file in the source dir
		entries, _ := os.ReadDir(srcDir)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".toc") {
				addonName = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
				break
			}
		}
		return srcDir, addonName, nil
	}

	// Auto-detect: look for .toc files in current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check root for .toc files
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return "", "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".toc") {
			name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			return cwd, name, nil
		}
	}

	// Check subfolders for .toc files
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subEntries, err := os.ReadDir(filepath.Join(cwd, e.Name()))
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if !se.IsDir() && strings.HasSuffix(strings.ToLower(se.Name()), ".toc") {
				name := strings.TrimSuffix(se.Name(), filepath.Ext(se.Name()))
				return filepath.Join(cwd, e.Name()), name, nil
			}
		}
	}

	return "", "", fmt.Errorf("no .toc file found — set source in blink.toml or use --source")
}

func FindWowPath(wowPathFlag string) (string, error) {
	if wowPathFlag != "" && wowPathFlag != "auto" {
		info, err := os.Stat(wowPathFlag)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("wow-path %q does not exist or is not a directory", wowPathFlag)
		}
		return wowPathFlag, nil
	}

	// Auto-detect: look for WoW version folders
	bases := []string{
		"/mnt/c/Program Files (x86)/World of Warcraft",
		"/mnt/c/Program Files/World of Warcraft",
		"/mnt/d/World of Warcraft",
		"/mnt/d/Program Files/World of Warcraft",
		"/mnt/e/World of Warcraft",
		"/mnt/e/Program Files/World of Warcraft",
	}

	versionDirs := []string{"_retail_", "_classic_", "_classic_era_"}

	for _, base := range bases {
		for _, ver := range versionDirs {
			p := filepath.Join(base, ver)
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				return p, nil
			}
		}
	}

	return "", fmt.Errorf("WoW installation not found — set wowPath in blink.toml or use --wow-path")
}
