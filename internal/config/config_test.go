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
	if cfg.Version != "retail" {
		t.Errorf("Version = %q, want %q", cfg.Version, "retail")
	}
	if cfg.UseGitignore != true {
		t.Error("UseGitignore = false, want true")
	}
	if len(cfg.Ignore) != 0 {
		t.Errorf("Ignore = %v, want empty", cfg.Ignore)
	}
}

func TestLoad_NoFile(t *testing.T) {
	// Run in a temp dir with no blink.toml
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != "retail" {
		t.Errorf("Version = %q, want %q", cfg.Version, "retail")
	}
}

func TestLoad_ValidTOML(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	toml := `source = "/my/addon"
wowPath = "/mnt/c/WoW"
version = "classic"
ignore = ["*.bak"]
useGitignore = false
`
	os.WriteFile(filepath.Join(dir, "blink.toml"), []byte(toml), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Source != "/my/addon" {
		t.Errorf("Source = %q, want %q", cfg.Source, "/my/addon")
	}
	if cfg.Version != "classic" {
		t.Errorf("Version = %q, want %q", cfg.Version, "classic")
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
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.WriteFile(filepath.Join(dir, "blink.toml"), []byte("not valid {{toml"), 0o644)

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
		version string
		wantVer string
		wantErr bool
	}{
		{"no flags", "", "", "", "retail", false},
		{"override version", "", "", "classic", "classic", false},
		{"classic_era", "", "", "classic_era", "classic_era", false},
		{"invalid version", "", "", "beta", "", true},
		{"override source", "/foo", "", "", "retail", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			err := MergeFlags(&cfg, tt.source, tt.wowPath, tt.version)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MergeFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && cfg.Version != tt.wantVer {
				t.Errorf("Version = %q, want %q", cfg.Version, tt.wantVer)
			}
			if tt.source != "" && !tt.wantErr && cfg.Source != tt.source {
				t.Errorf("Source = %q, want %q", cfg.Source, tt.source)
			}
		})
	}
}
