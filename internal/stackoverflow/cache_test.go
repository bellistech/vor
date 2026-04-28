package stackoverflow

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func withTempCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := cacheDir
	SetCacheDir(dir)
	t.Cleanup(func() { SetCacheDir(prev) })
	return dir
}

func sampleResult() *Result {
	return &Result{
		Questions: []Question{
			{Title: "Q?", Link: "https://stackoverflow.com/q/1", Score: 5},
		},
		QuotaMax:       10000,
		QuotaRemaining: 9998,
	}
}

func TestCache_RoundTrip(t *testing.T) {
	withTempCache(t)
	r := sampleResult()
	if err := Write("query 1", r); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, ok := Read("query 1", time.Hour)
	if !ok {
		t.Fatal("Read: expected hit")
	}
	if len(got.Questions) != 1 || got.Questions[0].Title != "Q?" {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestCache_Miss(t *testing.T) {
	withTempCache(t)
	if _, ok := Read("never written", time.Hour); ok {
		t.Error("expected cache miss")
	}
}

func TestCache_Expired(t *testing.T) {
	withTempCache(t)
	if err := Write("aged", sampleResult()); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// rewrite with past timestamp
	p := cachePath("aged")
	stale := []byte(`{"stored_at":1, "result":{"questions":[]}}`)
	if err := os.WriteFile(p, stale, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Read("aged", time.Hour); ok {
		t.Error("expected expired entry to miss")
	}
}

func TestCache_CorruptIgnored(t *testing.T) {
	dir := withTempCache(t)
	p := filepath.Join(dir, hashQuery("corrupt")+".json")
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Read("corrupt", time.Hour); ok {
		t.Error("expected corrupt entry to miss")
	}
}

func TestCache_FilenameIsHash(t *testing.T) {
	dir := withTempCache(t)
	if err := Write("the query", sampleResult()); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") && len(e.Name()) == 64+5 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sha256-hex.json filename, got entries: %v", entries)
	}
}

func TestCache_DoesNotContainKey(t *testing.T) {
	// The key should never be persisted — it's part of the URL, not the body
	// we store. Sanity-check that the cache file contains nothing key-shaped.
	withTempCache(t)
	r := sampleResult()
	if err := Write("test", r); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cachePath("test"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), sampleKey) {
		t.Errorf("cache file leaked sampleKey: %s", data)
	}
}

func TestCache_ConcurrentWrites(t *testing.T) {
	withTempCache(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = Write("concurrent", sampleResult())
		}()
	}
	wg.Wait()
	if _, ok := Read("concurrent", time.Hour); !ok {
		t.Error("expected at least one concurrent Write to land")
	}
}

func TestCache_Clear(t *testing.T) {
	dir := withTempCache(t)
	if err := Write("q", sampleResult()); err != nil {
		t.Fatal(err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected dir gone, got err=%v", err)
	}
}
