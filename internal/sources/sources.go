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

// Load enumerates ~/.config/cs/sources/ and returns one fs.FS per entry
// that resolves to a directory. Entries that are files, dangling symlinks,
// or otherwise inaccessible are silently skipped — additional sources
// are opt-in and a malformed entry must not break startup.
//
// Returns an empty slice (no error) when the discovery directory does
// not exist; that's the normal case for users who haven't opted in.
func Load() ([]fs.FS, error) {
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

	var out []fs.FS
	for _, e := range entries {
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
		out = append(out, os.DirFS(resolved))
	}
	return out, nil
}
