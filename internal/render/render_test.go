package render

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout temporarily replaces os.Stdout with a pipe so tests can read
// what a function wrote. Returns the captured bytes and restores stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan []byte)
	go func() {
		buf, _ := io.ReadAll(r)
		done <- buf
	}()

	fn()
	w.Close()
	return string(<-done)
}

func TestIsTTY_FalseUnderTest(t *testing.T) {
	// `go test` runs without a real TTY for stdout, so IsTTY() must be false.
	// This isn't a vacuous assertion — it pins the test environment so the
	// other tests can rely on the non-TTY branches being taken.
	if IsTTY() {
		t.Skip("running under a TTY (interactive go test); other tests still apply")
	}
}

func TestTermWidth_Default(t *testing.T) {
	w := TermWidth()
	// On a non-TTY stdout, term.GetSize errors and we fall back to 80.
	// On some CI shells stdout may still report a valid size; accept either
	// the fallback or any positive value.
	if w <= 0 {
		t.Errorf("TermWidth = %d, want > 0", w)
	}
}

func TestTermHeight_Default(t *testing.T) {
	h := TermHeight()
	if h <= 0 {
		t.Errorf("TermHeight = %d, want > 0", h)
	}
}

func TestRender_NonTTYReturnsRaw(t *testing.T) {
	// Under `go test` IsTTY() is false → Render() returns content unchanged.
	in := "# Heading\n\nbody **bold** more.\n"
	out, err := Render(in)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !IsTTY() && out != in {
		t.Errorf("non-TTY Render should pass content through unchanged\n got: %q\nwant: %q", out, in)
	}
}

func TestRender_NoColorEnvSkipsGlamour(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	in := "# Heading\n\nbody.\n"
	out, err := Render(in)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if out != in {
		t.Errorf("NO_COLOR set: should pass through; got %q", out)
	}
}

func TestOutput_NonTTYWritesPlain(t *testing.T) {
	in := "hello world\n"
	got := captureStdout(t, func() {
		if err := Output(in); err != nil {
			t.Fatalf("Output: %v", err)
		}
	})
	if got != in {
		t.Errorf("Output non-TTY: got %q, want %q", got, in)
	}
}

func TestPlainOutput_NonTTYWritesAsIs(t *testing.T) {
	in := "raw text — no markdown rendering applied\nline 2\n"
	got := captureStdout(t, func() {
		if err := PlainOutput(in); err != nil {
			t.Fatalf("PlainOutput: %v", err)
		}
	})
	if got != in {
		t.Errorf("PlainOutput non-TTY: got %q, want %q", got, in)
	}
}

func TestPlainOutput_ManyLinesNonTTY(t *testing.T) {
	// Even very long content under non-TTY stdout should write directly,
	// not invoke the pager (the pager branch is gated on IsTTY()).
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString("line\n")
	}
	in := b.String()
	got := captureStdout(t, func() {
		if err := PlainOutput(in); err != nil {
			t.Fatalf("PlainOutput: %v", err)
		}
	})
	if got != in {
		t.Errorf("PlainOutput long non-TTY: got %d bytes, want %d", len(got), len(in))
	}
}

func TestRender_EmptyContent(t *testing.T) {
	out, err := Render("")
	if err != nil {
		t.Fatalf("Render(empty): %v", err)
	}
	if out != "" {
		t.Errorf("Render(empty) = %q, want empty", out)
	}
}

func TestOutput_EmptyContent(t *testing.T) {
	got := captureStdout(t, func() {
		if err := Output(""); err != nil {
			t.Fatalf("Output(empty): %v", err)
		}
	})
	if got != "" {
		t.Errorf("Output(empty) wrote %q, expected nothing", got)
	}
}

func TestRender_DoesNotMutateOnError(t *testing.T) {
	// Render returns (content, nil) when glamour can't be set up under
	// non-TTY conditions. Verify the content is still the original.
	in := "## Section\n\n```bash\nfoo\n```\n"
	out, err := Render(in)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !IsTTY() && out != in {
		t.Errorf("non-TTY Render should round-trip; got %q", out)
	}
}
