// Package sources discovers additional cheatsheet/markdown sources by
// walking ~/.config/cs/sources/. Anything in that directory that resolves
// (directly or via symlink) to a directory becomes one os.DirFS the
// registry can ingest.
//
// Convention: users symlink any project they want indexed, e.g.
//
//     ln -s /home/govan/tmp/unheaded ~/.config/cs/sources/unheaded
//
// The existing registry walker filters for `.md` files, so non-markdown
// content under symlinked targets is implicitly ignored.
//
// Stdlib-only — no JSON, no glob library. The discovery directory IS the
// schema.
package sources

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// dirOverride lets tests redirect the discovery directory.
var dirOverride string

// SetDir overrides the discovery directory. Test-only.
func SetDir(path string) { dirOverride = path }

// Dir returns the resolved discovery directory path. Empty string if HOME
// is unavailable and no override is set.
func Dir() string {
	if dirOverride != "" {
		return dirOverride
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "cs", "sources")
}

// Source records a single discovered user-symlinked source. The Label
// is the symlink-name as it appeared under ~/.config/cs/sources/ (e.g.
// "unheaded" for ~/.config/cs/sources/unheaded → /home/.../unheaded).
// The Path is the resolved target directory. Trusted is true when the
// label appears in ~/.config/cs/sources/.trusted (one name per line)
// — caller will tag those as SourceUserCustom (local trust) instead
// of SourceUserSource (external).
type Source struct {
	FS      fs.FS
	Path    string
	Label   string
	Trusted bool
}

// loadTrustList reads ~/.config/cs/sources/.trusted and returns the set
// of symlink names the user has opted-in to "local" trust for. Lines
// starting with `#` and blank lines are ignored. Missing file → empty
// set (default behavior: every symlinked source is external).
//
// Why a flat file: same minimalism as Phase A — no JSON, no glob lib,
// no schema. The discovery directory IS the schema; .trusted is the
// only escape hatch.
func loadTrustList(dir string) map[string]struct{} {
	trustFile := filepath.Join(dir, ".trusted")
	data, err := os.ReadFile(trustFile)
	if err != nil {
		return nil
	}
	out := make(map[string]struct{})
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[line] = struct{}{}
	}
	return out
}

// Load enumerates ~/.config/cs/sources/ and returns one Source per
// entry that resolves to a directory. Entries that are files, dangling
// symlinks, or otherwise inaccessible are silently skipped —
// additional sources are opt-in and a malformed entry must not break
// startup.
//
// Returns an empty slice (no error) when the discovery directory does
// not exist; that's the normal case for users who haven't opted in.
func Load() ([]Source, error) {
	dir := Dir()
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	trusted := loadTrustList(dir)

	var out []Source
	for _, e := range entries {
		// Skip dotfiles — that's where the .trusted config lives, and
		// keeps the convention symmetrical with shell hidden-files.
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		// EvalSymlinks resolves symlinks AND verifies the target exists;
		// a broken symlink yields an error we use to skip it.
		resolved, err := filepath.EvalSymlinks(full)
		if err != nil {
			continue
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			continue
		}
		_, isTrusted := trusted[e.Name()]
		out = append(out, Source{
			FS:      os.DirFS(resolved),
			Path:    resolved,
			Label:   e.Name(),
			Trusted: isTrusted,
		})
	}
	return out, nil
}
