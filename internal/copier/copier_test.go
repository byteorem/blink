package copier

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldIgnore_AlwaysIgnored(t *testing.T) {
	ig := NewIgnorer(t.TempDir(), nil, false, false)

	alwaysIgnored := []string{"blink.toml", ".git", ".git/config", ".git/HEAD"}
	for _, p := range alwaysIgnored {
		if !ig.ShouldIgnore(p) {
			t.Errorf("ShouldIgnore(%q) = false, want true", p)
		}
	}
}

func TestShouldIgnore_GlobPatterns(t *testing.T) {
	ig := NewIgnorer(t.TempDir(), []string{"*.bak", "*.log"}, false, false)

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
	ig := NewIgnorer(t.TempDir(), []string{"node_modules/"}, false, false)

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
	_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.tmp\n# comment\n\nbuild/\n"), 0o644)

	ig := NewIgnorer(dir, nil, true, false)

	if !ig.ShouldIgnore("foo.tmp") {
		t.Error("should ignore *.tmp from .gitignore")
	}
	if !ig.ShouldIgnore("build/output") {
		t.Error("should ignore build/ from .gitignore")
	}
}

func TestNewIgnorer_NoGitignore(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.tmp\n"), 0o644)

	ig := NewIgnorer(dir, []string{"*.bak"}, false, false)

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
	_ = os.WriteFile(filepath.Join(src, "main.lua"), []byte("print('hi')"), 0o644)
	_ = os.MkdirAll(filepath.Join(src, "libs"), 0o755)
	_ = os.WriteFile(filepath.Join(src, "libs", "helper.lua"), []byte("-- help"), 0o644)
	_ = os.WriteFile(filepath.Join(src, "blink.toml"), []byte("ignored"), 0o644)
	_ = os.MkdirAll(filepath.Join(src, ".git"), 0o755)
	_ = os.WriteFile(filepath.Join(src, ".git", "HEAD"), []byte("ref"), 0o644)

	ig := NewIgnorer(src, nil, false, false)
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
	_ = os.WriteFile(srcFile, []byte("hello"), 0o644)

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
	_ = os.WriteFile(f, []byte("bye"), 0o644)

	if err := DeleteFile(f); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestNewIgnorer_PkgMetaIgnore(t *testing.T) {
	dir := t.TempDir()
	pkgmeta := `package-as: MyAddon

ignore:
  - README.md
  - tests/
  - .github

manual-changelog:
  filename: CHANGELOG.md
`
	_ = os.WriteFile(filepath.Join(dir, ".pkgmeta"), []byte(pkgmeta), 0o644)

	ig := NewIgnorer(dir, nil, false, true)

	if !ig.ShouldIgnore("README.md") {
		t.Error("should ignore README.md from .pkgmeta")
	}
	if !ig.ShouldIgnore("tests/foo.lua") {
		t.Error("should ignore tests/ from .pkgmeta")
	}
	if !ig.ShouldIgnore(".github") {
		t.Error("should ignore .github from .pkgmeta")
	}
	if ig.ShouldIgnore("main.lua") {
		t.Error("should not ignore main.lua")
	}
}

func TestNewIgnorer_PkgMetaDisabled(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".pkgmeta"), []byte("ignore:\n  - README.md\n"), 0o644)

	ig := NewIgnorer(dir, nil, false, false)

	if ig.ShouldIgnore("README.md") {
		t.Error("should not ignore README.md when usePkgMeta=false")
	}
}

func TestNewIgnorer_PkgMetaNoIgnoreBlock(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".pkgmeta"), []byte("package-as: MyAddon\n"), 0o644)

	ig := NewIgnorer(dir, nil, false, true)

	if ig.ShouldIgnore("main.lua") {
		t.Error("should not ignore main.lua with no pkgmeta ignore block")
	}
}

func TestCleanDestination_RemovesStaleFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Source has one file
	_ = os.WriteFile(filepath.Join(src, "main.lua"), []byte("keep"), 0o644)

	// Destination has that file plus a stale one
	_ = os.WriteFile(filepath.Join(dst, "main.lua"), []byte("keep"), 0o644)
	_ = os.WriteFile(filepath.Join(dst, "old.lua"), []byte("stale"), 0o644)

	removed, err := CleanDestination(src, dst, nil)
	if err != nil {
		t.Fatalf("CleanDestination() error = %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(filepath.Join(dst, "main.lua")); err != nil {
		t.Error("main.lua should still exist")
	}
	if _, err := os.Stat(filepath.Join(dst, "old.lua")); !os.IsNotExist(err) {
		t.Error("old.lua should be removed")
	}
}

func TestCleanDestination_RemovesEmptyDirs(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Destination has a subdir with only a stale file
	_ = os.MkdirAll(filepath.Join(dst, "libs"), 0o755)
	_ = os.WriteFile(filepath.Join(dst, "libs", "old.lua"), []byte("stale"), 0o644)

	removed, err := CleanDestination(src, dst, nil)
	if err != nil {
		t.Fatalf("CleanDestination() error = %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(filepath.Join(dst, "libs")); !os.IsNotExist(err) {
		t.Error("empty libs/ dir should be removed")
	}
}

func TestCleanDestination_NonExistentDst(t *testing.T) {
	src := t.TempDir()
	removed, err := CleanDestination(src, "/nonexistent/path", nil)
	if err != nil {
		t.Fatalf("CleanDestination() error = %v", err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
}

func TestCleanDestination_RemovesIgnoredFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Source and dst both have main.lua (should be kept)
	_ = os.WriteFile(filepath.Join(src, "main.lua"), []byte("keep"), 0o644)
	_ = os.WriteFile(filepath.Join(dst, "main.lua"), []byte("keep"), 0o644)

	// Dst has files that match ignore patterns (previously synced)
	_ = os.MkdirAll(filepath.Join(dst, ".github", "workflows"), 0o755)
	_ = os.WriteFile(filepath.Join(dst, ".github", "workflows", "ci.yml"), []byte("ci"), 0o644)
	_ = os.WriteFile(filepath.Join(dst, "README.md"), []byte("readme"), 0o644)

	// Source also has these files (they exist in source but should still be cleaned because they're ignored)
	_ = os.MkdirAll(filepath.Join(src, ".github", "workflows"), 0o755)
	_ = os.WriteFile(filepath.Join(src, ".github", "workflows", "ci.yml"), []byte("ci"), 0o644)
	_ = os.WriteFile(filepath.Join(src, "README.md"), []byte("readme"), 0o644)

	ig := NewIgnorer(src, []string{".github/", "README.md"}, false, false)

	removed, err := CleanDestination(src, dst, ig)
	if err != nil {
		t.Fatalf("CleanDestination() error = %v", err)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
	if _, err := os.Stat(filepath.Join(dst, "main.lua")); err != nil {
		t.Error("main.lua should still exist")
	}
	if _, err := os.Stat(filepath.Join(dst, "README.md")); !os.IsNotExist(err) {
		t.Error("README.md should be removed")
	}
	if _, err := os.Stat(filepath.Join(dst, ".github")); !os.IsNotExist(err) {
		t.Error(".github/ dir should be removed")
	}
}

func TestDeleteFile_NonExistent(t *testing.T) {
	if err := DeleteFile("/nonexistent/file/path"); err != nil {
		t.Errorf("DeleteFile() non-existent should return nil, got %v", err)
	}
}
