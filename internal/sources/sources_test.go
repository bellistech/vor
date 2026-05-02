package sources

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withTempDir points the package at a fresh discovery directory for the
// duration of the test. Returns the directory path. The override is
// reset automatically when the test ends.
func withTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir("") })
	return dir
}

// readMD reads a .md file out of one of the loaded fs.FS entries by
// path; helper for the assertion that loaded sources are usable.
func readMD(fsys fs.FS, path string) (string, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestDir_Default(t *testing.T) {
	// Reset any override from prior tests.
	SetDir("")
	t.Cleanup(func() { SetDir("") })

	got := Dir()
	if got == "" {
		// HOME unavailable — accept empty per documented behavior.
		return
	}
	want := ".config" + string(filepath.Separator) + "cs" + string(filepath.Separator) + "sources"
	if !strings.HasSuffix(got, want) {
		t.Errorf("Dir() = %q; want suffix %q", got, want)
	}
}

func TestDir_Override(t *testing.T) {
	SetDir("/tmp/test-sources")
	t.Cleanup(func() { SetDir("") })
	if got := Dir(); got != "/tmp/test-sources" {
		t.Errorf("Dir() override = %q; want %q", got, "/tmp/test-sources")
	}
}

func TestLoad_MissingDir(t *testing.T) {
	dir := t.TempDir()
	SetDir(filepath.Join(dir, "does-not-exist"))
	t.Cleanup(func() { SetDir("") })

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() = %d sources; want 0 (missing dir is opt-in OK)", len(got))
	}
}

func TestLoad_EmptyDir(t *testing.T) {
	withTempDir(t)
	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() = %d; want 0 from empty dir", len(got))
	}
}

func TestLoad_RegularDirectory(t *testing.T) {
	dir := withTempDir(t)
	// Make a regular subdirectory (not a symlink) with one .md file.
	sub := filepath.Join(dir, "project1")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "hello.md"), []byte("# hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Load() = %d; want 1", len(got))
	}
	body, err := readMD(got[0].FS, "hello.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if body != "# hello" {
		t.Errorf("read = %q; want %q", body, "# hello")
	}
}

func TestLoad_SymlinkToDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need elevated perms on Windows")
	}
	dir := withTempDir(t)

	// Real target outside the discovery dir.
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "doc.md"), []byte("from target"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Symlink discovery/proj -> target
	if err := os.Symlink(target, filepath.Join(dir, "proj")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Load() = %d; want 1 (symlinked dir)", len(got))
	}
	body, err := readMD(got[0].FS, "doc.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if body != "from target" {
		t.Errorf("read = %q; want %q", body, "from target")
	}
}

func TestLoad_DanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need elevated perms on Windows")
	}
	dir := withTempDir(t)
	// Symlink to a path that does not exist.
	if err := os.Symlink(filepath.Join(t.TempDir(), "nope"), filepath.Join(dir, "broken")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error on dangling symlink: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() = %d; want 0 (dangling symlink should be skipped)", len(got))
	}
}

func TestLoad_SymlinkToFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need elevated perms on Windows")
	}
	dir := withTempDir(t)
	// Create a real file outside, then symlink to it inside discovery.
	scratch := t.TempDir()
	target := filepath.Join(scratch, "single.md")
	if err := os.WriteFile(target, []byte("file"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "as-symlink")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() = %d; want 0 (file symlink should be skipped, only dirs accepted)", len(got))
	}
}

func TestLoad_MultipleSources(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need elevated perms on Windows")
	}
	dir := withTempDir(t)

	// One regular subdir, one symlinked dir.
	sub := filepath.Join(dir, "regular")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.md"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}

	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "b.md"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write b: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "linked")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Load() = %d; want 2", len(got))
	}
}

func TestLoad_TrustListPromotes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need elevated perms on Windows")
	}
	dir := withTempDir(t)

	// Two sources: one in the trust list, one not.
	for _, name := range []string{"trusted-src", "external-src"} {
		target := t.TempDir()
		if err := os.WriteFile(filepath.Join(target, "x.md"), []byte("# x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Fatalf("symlink: %v", err)
		}
	}

	// Trust list elevates trusted-src.
	trustFile := filepath.Join(dir, ".trusted")
	if err := os.WriteFile(trustFile, []byte("# user's own repos\ntrusted-src\n"), 0o644); err != nil {
		t.Fatalf("write trust file: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d sources; want 2", len(got))
	}

	// .trusted itself should NOT appear as a source.
	for _, s := range got {
		if s.Label == ".trusted" {
			t.Errorf(".trusted file should be skipped, not loaded as source")
		}
	}

	byLabel := map[string]Source{}
	for _, s := range got {
		byLabel[s.Label] = s
	}
	if !byLabel["trusted-src"].Trusted {
		t.Errorf("trusted-src should be Trusted=true; got Source=%+v", byLabel["trusted-src"])
	}
	if byLabel["external-src"].Trusted {
		t.Errorf("external-src should be Trusted=false; got Source=%+v", byLabel["external-src"])
	}
}

func TestLoad_TrustList_HandlesCommentsAndBlankLines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need elevated perms on Windows")
	}
	dir := withTempDir(t)
	target := t.TempDir()
	_ = os.WriteFile(filepath.Join(target, "x.md"), []byte("x"), 0o644)
	_ = os.Symlink(target, filepath.Join(dir, "myrepo"))

	// Trust file with comments, blanks, and trailing whitespace.
	trustContent := `# trust list
# blank line below should be ignored

  myrepo
# this line is just a comment
`
	if err := os.WriteFile(filepath.Join(dir, ".trusted"), []byte(trustContent), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 1 || got[0].Label != "myrepo" || !got[0].Trusted {
		t.Errorf("Load() = %+v; want [{Label:myrepo Trusted:true}]", got)
	}
}

func TestLoad_NoTrustFile_AllExternal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need elevated perms on Windows")
	}
	dir := withTempDir(t)
	target := t.TempDir()
	_ = os.WriteFile(filepath.Join(target, "x.md"), []byte("x"), 0o644)
	_ = os.Symlink(target, filepath.Join(dir, "src"))

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d; want 1", len(got))
	}
	if got[0].Trusted {
		t.Errorf("Trusted = true; want false (no .trusted file)")
	}
}
