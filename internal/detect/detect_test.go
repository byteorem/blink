package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAddon_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	// Create a .toc file so addon name comes from it
	os.WriteFile(filepath.Join(dir, "MyAddon.toc"), []byte("## Title: My Addon"), 0o644)

	srcDir, name, err := FindAddon(dir)
	if err != nil {
		t.Fatalf("FindAddon() error = %v", err)
	}
	if srcDir != dir {
		t.Errorf("srcDir = %q, want %q", srcDir, dir)
	}
	if name != "MyAddon" {
		t.Errorf("name = %q, want %q", name, "MyAddon")
	}
}

func TestFindAddon_ExplicitPathNoToc(t *testing.T) {
	dir := t.TempDir()
	// No .toc, should use directory basename
	srcDir, name, err := FindAddon(dir)
	if err != nil {
		t.Fatalf("FindAddon() error = %v", err)
	}
	if srcDir != dir {
		t.Errorf("srcDir = %q, want %q", srcDir, dir)
	}
	if name != filepath.Base(dir) {
		t.Errorf("name = %q, want %q", name, filepath.Base(dir))
	}
}

func TestFindAddon_AutoDetect(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CoolAddon.toc"), []byte(""), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	srcDir, name, err := FindAddon("auto")
	if err != nil {
		t.Fatalf("FindAddon() error = %v", err)
	}
	if srcDir != dir {
		t.Errorf("srcDir = %q, want %q", srcDir, dir)
	}
	if name != "CoolAddon" {
		t.Errorf("name = %q, want %q", name, "CoolAddon")
	}
}

func TestFindAddon_AutoDetectSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "SubAddon")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "SubAddon.toc"), []byte(""), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, name, err := FindAddon("auto")
	if err != nil {
		t.Fatalf("FindAddon() error = %v", err)
	}
	if name != "SubAddon" {
		t.Errorf("name = %q, want %q", name, "SubAddon")
	}
}

func TestFindAddon_NoTocError(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, _, err := FindAddon("auto")
	if err == nil {
		t.Fatal("FindAddon() expected error when no .toc found")
	}
}

func TestFindWowPath_Explicit(t *testing.T) {
	dir := t.TempDir()
	path, err := FindWowPath(dir)
	if err != nil {
		t.Fatalf("FindWowPath() error = %v", err)
	}
	if path != dir {
		t.Errorf("path = %q, want %q", path, dir)
	}
}

func TestFindWowPath_InvalidPath(t *testing.T) {
	_, err := FindWowPath("/nonexistent/path/that/doesnt/exist")
	if err == nil {
		t.Fatal("FindWowPath() expected error for nonexistent path")
	}
}

func TestBuildTargetPath(t *testing.T) {
	tests := []struct {
		version string
		dir     string
	}{
		{"retail", "_retail_"},
		{"classic", "_classic_"},
		{"classic_era", "_classic_era_"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := BuildTargetPath("/wow", tt.version, "TestAddon")
			want := filepath.Join("/wow", tt.dir, "Interface", "AddOns", "TestAddon")
			if got != want {
				t.Errorf("BuildTargetPath() = %q, want %q", got, want)
			}
		})
	}
}
