package stackoverflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// cacheDir is overridable for tests via SetCacheDir.
var cacheDir string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cacheDir = filepath.Join(home, ".cache", "cs", "stackoverflow")
}

// SetCacheDir overrides the cache directory. Test-only.
func SetCacheDir(dir string) { cacheDir = dir }

// CacheDir returns the current cache directory.
func CacheDir() string { return cacheDir }

// cacheEntry wraps Result with a timestamp so we can age out stale hits.
type cacheEntry struct {
	StoredAt int64   `json:"stored_at"`
	Result   *Result `json:"result"`
}

// hashQuery hashes a query string into a hex sha256 (cache filename).
func hashQuery(q string) string {
	sum := sha256.Sum256([]byte(q))
	return hex.EncodeToString(sum[:])
}

// cachePath returns the absolute path of the cache file for a given query.
func cachePath(query string) string {
	if cacheDir == "" {
		return ""
	}
	return filepath.Join(cacheDir, hashQuery(query)+".json")
}

// Read returns a cached Result if one exists and is younger than ttl.
// Returns (nil, false) on miss / expired / corrupt — never an error.
// Cache contents are treated as untrusted; only typed JSON unmarshal.
func Read(query string, ttl time.Duration) (*Result, bool) {
	p := cachePath(query)
	if p == "" {
		return nil, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	var e cacheEntry
	if jerr := json.Unmarshal(data, &e); jerr != nil {
		return nil, false
	}
	if e.Result == nil {
		return nil, false
	}
	if time.Since(time.Unix(e.StoredAt, 0)) > ttl {
		return nil, false
	}
	return e.Result, true
}

// Write atomically stores a Result for query. Best-effort: on any disk error
// the function returns the error but callers may safely ignore it (the
// network call still produced a result we'll show the user this run).
func Write(query string, r *Result) error {
	p := cachePath(query)
	if p == "" {
		return errors.New("no cache dir configured")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cacheEntry{
		StoredAt: time.Now().Unix(),
		Result:   r,
	})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".so-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, werr := tmp.Write(data); werr != nil {
		tmp.Close()
		os.Remove(tmpName)
		return werr
	}
	if cerr := tmp.Close(); cerr != nil {
		os.Remove(tmpName)
		return cerr
	}
	return os.Rename(tmpName, p)
}

// Clear removes the cache directory entirely (used by `vor -so help` clear
// instructions; not invoked automatically).
func Clear() error {
	if cacheDir == "" {
		return nil
	}
	return os.RemoveAll(cacheDir)
}
