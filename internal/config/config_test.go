package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Source != "auto" {
		t.Errorf("Source = %q, want %q", cfg.Source, "auto")
	}
	if cfg.WowPath != "auto" {
		t.Errorf("WowPath = %q, want %q", cfg.WowPath, "auto")
	}
	if cfg.UseGitignore != true {
		t.Error("UseGitignore = false, want true")
	}
	if len(cfg.Ignore) != 0 {
		t.Errorf("Ignore = %v, want empty", cfg.Ignore)
	}
}

func TestLoad_NoFile(t *testing.T) {
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	_ = os.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Source != "auto" {
		t.Errorf("Source = %q, want %q", cfg.Source, "auto")
	}
}

func TestLoad_ValidTOML(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	_ = os.Chdir(dir)

	toml := `source = "/my/addon"
wowPath = "/mnt/c/WoW/_retail_"
ignore = ["*.bak"]
useGitignore = false
`
	_ = os.WriteFile(filepath.Join(dir, "blink.toml"), []byte(toml), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Source != "/my/addon" {
		t.Errorf("Source = %q, want %q", cfg.Source, "/my/addon")
	}
	if cfg.WowPath != "/mnt/c/WoW/_retail_" {
		t.Errorf("WowPath = %q, want %q", cfg.WowPath, "/mnt/c/WoW/_retail_")
	}
	if cfg.UseGitignore != false {
		t.Error("UseGitignore = true, want false")
	}
	if len(cfg.Ignore) != 1 || cfg.Ignore[0] != "*.bak" {
		t.Errorf("Ignore = %v, want [*.bak]", cfg.Ignore)
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	_ = os.Chdir(dir)

	_ = os.WriteFile(filepath.Join(dir, "blink.toml"), []byte("not valid {{toml"), 0o644)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid TOML")
	}
}

func TestMergeFlags(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wowPath string
	}{
		{"no flags", "", ""},
		{"override source", "/foo", ""},
		{"override wowPath", "", "/mnt/c/WoW/_retail_"},
		{"override both", "/foo", "/mnt/c/WoW/_retail_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			MergeFlags(&cfg, tt.source, tt.wowPath)
			if tt.source != "" && cfg.Source != tt.source {
				t.Errorf("Source = %q, want %q", cfg.Source, tt.source)
			}
			if tt.wowPath != "" && cfg.WowPath != tt.wowPath {
				t.Errorf("WowPath = %q, want %q", cfg.WowPath, tt.wowPath)
			}
		})
	}
}
