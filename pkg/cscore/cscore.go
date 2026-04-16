// Package cscore provides the platform-independent core of cs.
// Designed for gomobile bind. All public functions return simple types
// or JSON-encoded strings. No interfaces, channels, func types in API.
// OFFLINE SACRED LAW: This package imports ZERO network packages.
package cscore

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/bellistech/cs/internal/bookmarks"
	"github.com/bellistech/cs/internal/registry"
)

var (
	mu       sync.RWMutex
	reg      *registry.Registry
	initDone bool
	dataDir  string
)

// Init loads sheets and details from the given embed.FS sources.
// Must be called before any other cscore function.
func Init(sheets, details fs.FS) error {
	mu.Lock()
	defer mu.Unlock()
	if initDone {
		return fmt.Errorf("cscore: already initialized")
	}
	r, err := registry.NewWithDetails([]fs.FS{sheets}, []fs.FS{details})
	if err != nil {
		return fmt.Errorf("cscore: init registry: %w", err)
	}
	reg = r
	initDone = true
	return nil
}

// SetDataDir sets the directory for bookmarks and custom sheets.
// For mobile: pass the app sandbox documents directory.
// Must be called before bookmark operations.
func SetDataDir(path string) {
	mu.Lock()
	defer mu.Unlock()
	dataDir = path
	if path != "" {
		bookmarks.SetBookmarkFile(filepath.Join(path, "bookmarks.json"))
	}
}

// GetDataDir returns the current data directory.
func GetDataDir() string {
	mu.RLock()
	defer mu.RUnlock()
	return dataDir
}

func mustReg() *registry.Registry {
	mu.RLock()
	defer mu.RUnlock()
	if reg == nil {
		panic("cscore: not initialized — call Init() first")
	}
	return reg
}

func jsonMarshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}

func errorJSON(err error) string {
	if ve, ok := err.(*validationError); ok {
		return jsonMarshal(map[string]string{
			"error": ve.Message,
			"field": ve.Field,
		})
	}
	return jsonMarshal(map[string]string{"error": err.Error()})
}

// resetForTesting clears all package state. Test-only.
func resetForTesting() {
	mu.Lock()
	defer mu.Unlock()
	reg = nil
	initDone = false
	dataDir = ""
}
