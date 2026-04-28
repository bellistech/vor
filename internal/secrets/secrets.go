// Package secrets loads opt-in API credentials from either an environment
// variable (preferred) or a KEY=VALUE-format dotfile at ~/.config/cs/secrets.env
// (mode 0600 expected). It exists to support optional bonus features such as
// the Stack Overflow live-lookup flag — the offline-encyclopedia core of vör
// continues to work without any of this package being touched.
//
// The package is deliberately tiny and zero-dependency: stdlib only, ~80 LOC,
// designed to mirror internal/bookmarks for consistency.
package secrets

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrNotSet is returned by Load when neither the environment variable nor the
// secrets file contains a value for the requested key. Callers should treat
// this as a friendly "not configured" signal and prompt the user via help.
var ErrNotSet = errors.New("secret not set: configure env var or secrets.env")

var (
	secretsFile string
	modeWarn    sync.Once
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	secretsFile = filepath.Join(home, ".config", "cs", "secrets.env")
}

// SetFile overrides the secrets file path. Intended for tests.
func SetFile(path string) {
	secretsFile = path
	modeWarn = sync.Once{}
}

// File returns the path to the secrets dotfile.
func File() string {
	return secretsFile
}

// Load returns the value of the named secret, with the source it came from
// ("env" or "file"). The environment variable always wins; the file is the
// persistent default. Both missing → ErrNotSet.
//
// If the secrets file exists with permission bits beyond the owner (any of
// group or world bits set), a one-shot warning is written to stderr — but
// the value is still returned. We warn, we don't fail.
func Load(name string) (string, string, error) {
	if v := os.Getenv(name); v != "" {
		return v, "env", nil
	}
	if secretsFile == "" {
		return "", "", ErrNotSet
	}
	f, err := os.Open(secretsFile)
	if err != nil {
		return "", "", ErrNotSet
	}
	defer f.Close()

	if info, statErr := f.Stat(); statErr == nil {
		if info.Mode().Perm()&0o077 != 0 {
			modeWarn.Do(func() {
				fmt.Fprintf(os.Stderr,
					"warning: %s is group/world-readable; chmod 600 %s\n",
					secretsFile, secretsFile)
			})
		}
	}

	val, found := scan(f, name)
	if !found {
		return "", "", ErrNotSet
	}
	return val, "file", nil
}

// scan parses KEY=VALUE lines, ignoring blank lines and `#` comments.
// Returns the value (without surrounding quotes) and a found flag.
func scan(r io.Reader, name string) (string, bool) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// allow `export KEY=VALUE` for shell-source-compatibility
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		if k != name {
			continue
		}
		v := strings.TrimSpace(line[eq+1:])
		// strip optional surrounding "..." or '...' (single set only)
		if len(v) >= 2 {
			first, last := v[0], v[len(v)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				v = v[1 : len(v)-1]
			}
		}
		return v, true
	}
	return "", false
}

// Redact replaces every literal occurrence of secret in s with "***". Used
// when wrapping errors that may have been built from a URL containing the key.
// Empty secret is a no-op.
func Redact(s, secret string) string {
	if secret == "" {
		return s
	}
	return strings.ReplaceAll(s, secret, "***")
}
