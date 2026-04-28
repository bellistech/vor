package secrets

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// withTempFile points the package at a per-test secrets file under t.TempDir()
// and resets package-level state so each test runs in isolation.
func withTempFile(t *testing.T, contents string, mode os.FileMode) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.env")
	if contents != "" {
		if err := os.WriteFile(path, []byte(contents), mode); err != nil {
			t.Fatalf("write secrets file: %v", err)
		}
	}
	prev := secretsFile
	SetFile(path)
	t.Cleanup(func() { SetFile(prev) })
	return path
}

func TestLoad_EnvOnly(t *testing.T) {
	withTempFile(t, "", 0o600)
	t.Setenv("VOR_TEST_KEY_ENVONLY", "from-env")

	v, src, err := Load("VOR_TEST_KEY_ENVONLY")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v != "from-env" {
		t.Errorf("value = %q, want %q", v, "from-env")
	}
	if src != "env" {
		t.Errorf("source = %q, want %q", src, "env")
	}
}

func TestLoad_FileOnly(t *testing.T) {
	withTempFile(t, "VOR_TEST_KEY_FILEONLY=from-file\n", 0o600)
	os.Unsetenv("VOR_TEST_KEY_FILEONLY")

	v, src, err := Load("VOR_TEST_KEY_FILEONLY")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v != "from-file" {
		t.Errorf("value = %q, want %q", v, "from-file")
	}
	if src != "file" {
		t.Errorf("source = %q, want %q", src, "file")
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	withTempFile(t, "VOR_TEST_KEY_BOTH=from-file\n", 0o600)
	t.Setenv("VOR_TEST_KEY_BOTH", "from-env")

	v, src, _ := Load("VOR_TEST_KEY_BOTH")
	if v != "from-env" {
		t.Errorf("env should win: got %q", v)
	}
	if src != "env" {
		t.Errorf("source should be env: got %q", src)
	}
}

func TestLoad_BothMissing(t *testing.T) {
	withTempFile(t, "OTHER=value\n", 0o600)
	os.Unsetenv("VOR_TEST_KEY_MISSING")

	_, _, err := Load("VOR_TEST_KEY_MISSING")
	if !errors.Is(err, ErrNotSet) {
		t.Errorf("expected ErrNotSet, got %v", err)
	}
}

func TestLoad_FileMissing(t *testing.T) {
	dir := t.TempDir()
	prev := secretsFile
	SetFile(filepath.Join(dir, "does-not-exist.env"))
	t.Cleanup(func() { SetFile(prev) })
	os.Unsetenv("VOR_TEST_KEY_NOFILE")

	_, _, err := Load("VOR_TEST_KEY_NOFILE")
	if !errors.Is(err, ErrNotSet) {
		t.Errorf("expected ErrNotSet, got %v", err)
	}
}

func TestLoad_MalformedFile(t *testing.T) {
	// lines without `=` are skipped; subsequent lines still parse
	contents := "this line has no equals sign\n# comment\nVOR_TEST_KEY_MAL=ok\n"
	withTempFile(t, contents, 0o600)
	os.Unsetenv("VOR_TEST_KEY_MAL")

	v, _, err := Load("VOR_TEST_KEY_MAL")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v != "ok" {
		t.Errorf("value = %q, want %q", v, "ok")
	}
}

func TestLoad_QuotedValues(t *testing.T) {
	contents := `VOR_DQ="double quoted"
VOR_SQ='single quoted'
VOR_PLAIN=plain value
`
	withTempFile(t, contents, 0o600)

	for _, c := range []struct {
		name, want string
	}{
		{"VOR_DQ", "double quoted"},
		{"VOR_SQ", "single quoted"},
		{"VOR_PLAIN", "plain value"},
	} {
		os.Unsetenv(c.name)
		v, _, err := Load(c.name)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if v != c.want {
			t.Errorf("%s: got %q, want %q", c.name, v, c.want)
		}
	}
}

func TestLoad_ExportPrefix(t *testing.T) {
	withTempFile(t, "export VOR_EXP=shellcompat\n", 0o600)
	os.Unsetenv("VOR_EXP")

	v, _, err := Load("VOR_EXP")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if v != "shellcompat" {
		t.Errorf("value = %q, want %q", v, "shellcompat")
	}
}

func TestLoad_FileModeWarning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode bits don't match Unix semantics on Windows")
	}
	path := withTempFile(t, "VOR_WARN_KEY=val\n", 0o644) // group/world-readable
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	os.Unsetenv("VOR_WARN_KEY")

	r, w, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	// reset the once so this test always sees the warning
	modeWarn = sync.Once{}

	_, _, err := Load("VOR_WARN_KEY")
	w.Close()
	if err != nil {
		t.Fatalf("Load returned error despite warn-only path: %v", err)
	}

	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "group/world-readable") {
		t.Errorf("expected warning on stderr, got %q", out)
	}

	// second call should NOT emit a second warning (sync.Once gate)
	r2, w2, _ := os.Pipe()
	os.Stderr = w2
	_, _, _ = Load("VOR_WARN_KEY")
	w2.Close()
	out2, _ := io.ReadAll(r2)
	if strings.Contains(string(out2), "group/world-readable") {
		t.Errorf("warning should fire once; second call also warned: %q", out2)
	}
}

func TestRedact(t *testing.T) {
	if got := Redact("the quick brown fox", ""); got != "the quick brown fox" {
		t.Errorf("empty secret should be no-op: got %q", got)
	}
	if got := Redact("token=abc123 in URL", "abc123"); got != "token=*** in URL" {
		t.Errorf("redact failed: got %q", got)
	}
	if got := Redact("twice abc twice abc", "abc"); got != "twice *** twice ***" {
		t.Errorf("multi redact failed: got %q", got)
	}
}

func TestFile(t *testing.T) {
	withTempFile(t, "", 0o600)
	if File() == "" {
		t.Errorf("File() should return non-empty path after SetFile")
	}
}
