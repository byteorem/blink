package copier

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldIgnore_AlwaysIgnored(t *testing.T) {
	ig := &Ignorer{}

	alwaysIgnored := []string{"blink.toml", ".git", ".git/config", ".git/HEAD"}
	for _, p := range alwaysIgnored {
		if !ig.ShouldIgnore(p) {
			t.Errorf("ShouldIgnore(%q) = false, want true", p)
		}
	}
}

func TestShouldIgnore_GlobPatterns(t *testing.T) {
	ig := &Ignorer{patterns: []string{"*.bak", "*.log"}}

	if !ig.ShouldIgnore("test.bak") {
		t.Error("ShouldIgnore(test.bak) = false, want true")
	}
	if !ig.ShouldIgnore("sub/dir/file.log") {
		t.Error("ShouldIgnore(sub/dir/file.log) = false, want true")
	}
	if ig.ShouldIgnore("main.lua") {
		t.Error("ShouldIgnore(main.lua) = true, want false")
	}
}

func TestShouldIgnore_DirPatterns(t *testing.T) {
	ig := &Ignorer{patterns: []string{"node_modules/"}}

	if !ig.ShouldIgnore("node_modules") {
		t.Error("ShouldIgnore(node_modules) = false, want true")
	}
	if !ig.ShouldIgnore("node_modules/foo") {
		t.Error("ShouldIgnore(node_modules/foo) = false, want true")
	}
	if ig.ShouldIgnore("src/main.go") {
		t.Error("ShouldIgnore(src/main.go) = true, want false")
	}
}

func TestNewIgnorer_GitignorePatterns(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.tmp\n# comment\n\nbuild/\n"), 0o644)

	ig := NewIgnorer(dir, nil, true)

	if !ig.ShouldIgnore("foo.tmp") {
		t.Error("should ignore *.tmp from .gitignore")
	}
	if !ig.ShouldIgnore("build/output") {
		t.Error("should ignore build/ from .gitignore")
	}
}

func TestNewIgnorer_NoGitignore(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.tmp\n"), 0o644)

	ig := NewIgnorer(dir, []string{"*.bak"}, false)

	if ig.ShouldIgnore("foo.tmp") {
		t.Error("should not ignore *.tmp when useGitignore=false")
	}
	if !ig.ShouldIgnore("foo.bak") {
		t.Error("should ignore *.bak from extra patterns")
	}
}

func TestInitialSync(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create source files
	os.WriteFile(filepath.Join(src, "main.lua"), []byte("print('hi')"), 0o644)
	os.MkdirAll(filepath.Join(src, "libs"), 0o755)
	os.WriteFile(filepath.Join(src, "libs", "helper.lua"), []byte("-- help"), 0o644)
	os.WriteFile(filepath.Join(src, "blink.toml"), []byte("ignored"), 0o644)
	os.MkdirAll(filepath.Join(src, ".git"), 0o755)
	os.WriteFile(filepath.Join(src, ".git", "HEAD"), []byte("ref"), 0o644)

	ig := NewIgnorer(src, nil, false)
	count, err := InitialSync(src, dst, ig)
	if err != nil {
		t.Fatalf("InitialSync() error = %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// Verify files exist in dst
	if _, err := os.Stat(filepath.Join(dst, "main.lua")); err != nil {
		t.Error("main.lua not copied")
	}
	if _, err := os.Stat(filepath.Join(dst, "libs", "helper.lua")); err != nil {
		t.Error("libs/helper.lua not copied")
	}
	// Verify ignored files not copied
	if _, err := os.Stat(filepath.Join(dst, "blink.toml")); !os.IsNotExist(err) {
		t.Error("blink.toml should not be copied")
	}
}

func TestCopyFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "test.txt")
	os.WriteFile(srcFile, []byte("hello"), 0o644)

	dstFile := filepath.Join(dst, "sub", "dir", "test.txt")
	if err := CopyFile(srcFile, dstFile); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}
}

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "delete-me.txt")
	os.WriteFile(f, []byte("bye"), 0o644)

	if err := DeleteFile(f); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestDeleteFile_NonExistent(t *testing.T) {
	if err := DeleteFile("/nonexistent/file/path"); err != nil {
		t.Errorf("DeleteFile() non-existent should return nil, got %v", err)
	}
}
