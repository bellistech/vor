package render

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// updateGoldens regenerates testdata/golden/*.txt from current output.
// Run via:  go test ./internal/render -run TestGolden -update-goldens
// Always inspect the diff before committing — these files lock the
// LaTeX preprocessor's behaviour on real corpus content.
var updateGoldens = flag.Bool("update-goldens", false, "regenerate golden render fixtures")

// goldenCases covers the math-heavy detail pages (the primary motivation
// for the preprocessor), a handful of ramp-up sheets (ELI5 narrative),
// and a few ordinary cheatsheets. Adjust the list as the corpus grows,
// but keep math-bearing pages first — that's where regressions hurt.
var goldenCases = []struct {
	name string // golden file basename
	path string // source path relative to project root
}{
	// Math-heavy details — main regression surface for the LaTeX preprocessor
	{"detail-number-theory-crypto", "detail/cs-theory/number-theory-crypto.md"},
	{"detail-jwt", "detail/security/jwt.md"},
	{"detail-lattice-crypto", "detail/cs-theory/lattice-crypto.md"},
	{"detail-information-theory", "detail/cs-theory/information-theory.md"},

	// Ramp-up sheets — heavy prose, light math
	{"sheet-rampup-linux-kernel", "sheets/ramp-up/linux-kernel-eli5.md"},
	{"sheet-rampup-tcp", "sheets/ramp-up/tcp-eli5.md"},
	{"sheet-rampup-dns", "sheets/ramp-up/dns-eli5.md"},

	// Ordinary cheatsheets — pure prose, no math
	{"sheet-bgp", "sheets/networking/bgp.md"},
	{"sheet-lvm", "sheets/storage/lvm.md"},
	{"sheet-bash", "sheets/shell/bash.md"},
}

func TestGoldenRender(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("find project root: %v", err)
	}

	for _, tc := range goldenCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			srcPath := filepath.Join(root, tc.path)
			content, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}

			rendered, err := Render(string(content))
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			rendered = stripANSI(rendered)

			goldenPath := filepath.Join(root, "internal/render/testdata/golden", tc.name+".txt")

			if *updateGoldens {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(goldenPath, []byte(rendered), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("updated %s (%d bytes)", goldenPath, len(rendered))
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v\n(hint: run `go test ./internal/render -run TestGolden -update-goldens` to seed)", goldenPath, err)
			}

			if string(want) != rendered {
				// Surface a snippet so failures are debuggable without
				// dumping the whole 2000-line render.
				diff := firstDiffLine(string(want), rendered)
				t.Errorf("golden mismatch in %s — first diff:\n%s\n(re-run with -update-goldens if intentional)", tc.name, diff)
			}
		})
	}
}

// TestGoldenNoLatexCanary is a cheap sanity gate: every captured golden
// must be free of raw-LaTeX canaries. If a preprocessor change accidentally
// reintroduces leakage, this test fires regardless of whether the golden
// content also changed in other ways.
func TestGoldenNoLatexCanary(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("find project root: %v", err)
	}

	canaries := []string{
		`\bmod`, `\cdot`, `\equiv`, `\sum`, `\prod`, `\sqrt`, `\frac`,
		`\mathbb`, `\mathcal`, `\mathit`, `\mathrm`, `\pmod`,
		`\lfloor`, `\rfloor`, `\lceil`, `\rceil`, `\gcd`,
		`\phi`, `\varphi`, `\oplus`, `\Rightarrow`, `\rightarrow`,
		`\leftarrow`, `\text{`,
	}

	for _, tc := range goldenCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			goldenPath := filepath.Join(root, "internal/render/testdata/golden", tc.name+".txt")
			b, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Skipf("golden not seeded: %v", err)
			}
			s := string(b)
			for _, c := range canaries {
				if strings.Contains(s, c) {
					t.Errorf("canary %q present in %s — LaTeX leaked through preprocessor", c, tc.name)
				}
			}
		})
	}
}

// projectRoot walks up from the test's working directory until it finds
// go.mod. This works regardless of which package the test runs in.
func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inEsc := false
	for i := 0; i < len(s); i++ {
		if inEsc {
			c := s[i]
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				inEsc = false
			}
			continue
		}
		if s[i] == 0x1b {
			inEsc = true
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func firstDiffLine(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	for i := 0; i < len(wantLines) && i < len(gotLines); i++ {
		if wantLines[i] != gotLines[i] {
			return "  line " + itoa(i+1) + ":\n    want: " + truncate(wantLines[i], 100) + "\n    got:  " + truncate(gotLines[i], 100)
		}
	}
	if len(wantLines) != len(gotLines) {
		return "  line counts differ: want=" + itoa(len(wantLines)) + " got=" + itoa(len(gotLines))
	}
	return "  (whitespace-only diff)"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
