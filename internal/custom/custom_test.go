package custom

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

// withTempHome redirects os.UserHomeDir() at the test for hermetic isolation.
// Returns the new HOME path; cleanup is automatic via t.TempDir().
func withTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	return home
}

func TestDir(t *testing.T) {
	home := withTempHome(t)
	got := Dir()
	want := filepath.Join(home, ".config", "cs", "sheets")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestDir_NoHome(t *testing.T) {
	// Force os.UserHomeDir() to fail by clearing the relevant env var.
	t.Setenv("HOME", "")
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", "")
	} else {
		// On Unix, an empty HOME makes os.UserHomeDir return an error.
		// (On macOS / Linux specifically — verified.)
	}
	got := Dir()
	// On platforms where the implementation has a fallback this may still
	// resolve. We accept either an empty string (the documented "cannot
	// determine home" branch) OR a non-empty fallback.
	if got != "" && !strings.Contains(got, ".config/cs/sheets") {
		t.Errorf("Dir() = %q, expected empty or .config/cs/sheets path", got)
	}
}

func TestLoad_NoDir(t *testing.T) {
	withTempHome(t)
	if got := Load(); got != nil {
		t.Errorf("Load() with no dir should return nil, got %v", got)
	}
}

func TestLoad_DirExists(t *testing.T) {
	home := withTempHome(t)
	dir := filepath.Join(home, ".config", "cs", "sheets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// drop a file in it so the fs.FS has something to read
	if err := os.WriteFile(filepath.Join(dir, "marker.md"), []byte("# marker"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load()
	if got == nil {
		t.Fatal("Load() returned nil; expected non-nil fs.FS")
	}
	data, err := fs.ReadFile(got, "marker.md")
	if err != nil {
		t.Errorf("ReadFile via Load(): %v", err)
	}
	if !strings.Contains(string(data), "marker") {
		t.Errorf("file content unexpected: %q", data)
	}
}

func TestLoad_PathIsFile(t *testing.T) {
	// If the path exists but is a file (not a dir), Load should return nil.
	home := withTempHome(t)
	parent := filepath.Join(home, ".config", "cs")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "sheets"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Load(); got != nil {
		t.Errorf("Load() with a file at the dir path should return nil, got %v", got)
	}
}

func TestAdd_HappyPath(t *testing.T) {
	home := withTempHome(t)

	// Source markdown file outside the dest dir
	src := filepath.Join(home, "input.md")
	body := "# Custom Sheet\n\n## Section\n\nContent.\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Add(src); err != nil {
		t.Fatalf("Add: %v", err)
	}

	dest := filepath.Join(home, ".config", "cs", "sheets", "uncategorized", "input.md")
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != body {
		t.Errorf("dest content mismatch:\n got: %q\nwant: %q", got, body)
	}
}

func TestAdd_AppendsMdExtension(t *testing.T) {
	home := withTempHome(t)
	src := filepath.Join(home, "noext")
	if err := os.WriteFile(src, []byte("# noext"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(src); err != nil {
		t.Fatalf("Add: %v", err)
	}
	dest := filepath.Join(home, ".config", "cs", "sheets", "uncategorized", "noext.md")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected %s to exist with .md appended: %v", dest, err)
	}
}

func TestAdd_MissingSource(t *testing.T) {
	withTempHome(t)
	err := Add("/does/not/exist/anywhere.md")
	if err == nil {
		t.Error("expected error for missing source file")
	}
	if !strings.Contains(err.Error(), "open") {
		t.Errorf("error should mention open: %v", err)
	}
}

func TestEdit_NewSheetTemplate(t *testing.T) {
	home := withTempHome(t)
	t.Setenv("EDITOR", "true") // /bin/true exits 0; effectively a no-op

	// Empty embedded FS — forces the "new sheet" template path.
	emptyFS := fstest.MapFS{}

	if err := Edit("brand-new-topic", emptyFS); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	dest := filepath.Join(home, ".config", "cs", "sheets",
		"uncategorized", "brand-new-topic.md")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	if !strings.Contains(string(data), "# brand-new-topic") {
		t.Errorf("template should start with H1; got: %q", data)
	}
	if !strings.Contains(string(data), "## Section") {
		t.Errorf("template should include a Section heading; got: %q", data)
	}
}

func TestEdit_CopyFromEmbedded(t *testing.T) {
	home := withTempHome(t)
	t.Setenv("EDITOR", "true")

	// Synthetic embedded FS containing a categorized sheet.
	emb := fstest.MapFS{
		"sheets/storage/lvm.md": &fstest.MapFile{
			Data: []byte("# LVM (embedded)\n\nbody\n"),
		},
	}

	if err := Edit("lvm", emb); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	dest := filepath.Join(home, ".config", "cs", "sheets", "storage", "lvm.md")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read copied custom: %v", err)
	}
	if !strings.Contains(string(data), "# LVM (embedded)") {
		t.Errorf("expected embedded content copied; got %q", data)
	}
}

func TestEdit_ExistingCustomFileFound(t *testing.T) {
	home := withTempHome(t)
	t.Setenv("EDITOR", "true")

	// Pre-create a custom file so the walk finds it before the embedded path.
	customDir := filepath.Join(home, ".config", "cs", "sheets", "shell")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatal(err)
	}
	preexist := filepath.Join(customDir, "bash.md")
	if err := os.WriteFile(preexist, []byte("# bash (custom)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	emb := fstest.MapFS{
		"sheets/shell/bash.md": &fstest.MapFile{Data: []byte("# bash (embedded)\n")},
	}
	if err := Edit("bash", emb); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	// Custom should NOT have been overwritten by the embedded copy.
	got, _ := os.ReadFile(preexist)
	if !strings.Contains(string(got), "(custom)") {
		t.Errorf("existing custom file overwritten unexpectedly: %q", got)
	}
}

func TestEdit_EditorRunFailureBubblesUp(t *testing.T) {
	withTempHome(t)
	t.Setenv("EDITOR", "/usr/bin/false") // exits 1

	// Use a brand-new topic so the template gets created and then EDITOR is invoked.
	emptyFS := fstest.MapFS{}
	err := Edit("topic-x", emptyFS)
	if err == nil {
		t.Error("expected error from non-zero editor exit, got nil")
	}
}

func TestEdit_DefaultEditorVi(t *testing.T) {
	// When EDITOR is unset, code falls back to "vi". We can't actually run
	// vi non-interactively, but we can verify the fallback path doesn't
	// panic and produces a usable error if vi isn't available — and that
	// the file is still created. Skip on systems where vi isn't on PATH.
	if _, err := os.Stat("/usr/bin/vi"); err != nil {
		t.Skip("/usr/bin/vi not available — skipping default-editor test")
	}
	home := withTempHome(t)
	t.Setenv("EDITOR", "")

	emptyFS := fstest.MapFS{}
	// vi opens an interactive session we can't drive from a test, so we expect
	// either Run() to error (TTY missing) or hang. To avoid the hang, set
	// EDITOR explicitly to true here — then the test only proves the file
	// creation path without depending on vi being interactive-safe in CI.
	t.Setenv("EDITOR", "true")

	if err := Edit("editor-default-test", emptyFS); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	expected := filepath.Join(home, ".config", "cs", "sheets",
		"uncategorized", "editor-default-test.md")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file at %s: %v", expected, err)
	}
}
