package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type logLine struct {
	TS        string `json:"ts"`
	Method    string `json:"method"`
	URI       string `json:"uri"`
	Truncated bool   `json:"uri_truncated"`
	Status    int    `json:"status"`
	Bytes     int    `json:"bytes"`
	DurUS     int64  `json:"dur_us"`
}

// syncBuffer is a bytes.Buffer safe for the concurrent test.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) lines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Split(strings.TrimRight(s.b.String(), "\n"), "\n")
}

func serveOne(t *testing.T, h http.HandlerFunc, r *http.Request) logLine {
	t.Helper()
	var buf syncBuffer
	accessLog(h, &buf, time.Now).ServeHTTP(httptest.NewRecorder(), r)
	lines := buf.lines()
	if len(lines) != 1 {
		t.Fatalf("want one line, got %d: %q", len(lines), lines)
	}
	var l logLine
	if err := json.Unmarshal([]byte(lines[0]), &l); err != nil {
		t.Fatalf("not JSON: %q: %v", lines[0], err)
	}
	return l
}

func TestAccessLogLine(t *testing.T) {
	l := serveOne(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) },
		httptest.NewRequest(http.MethodGet, "/api/search?q=bgp", nil))
	if l.Method != "GET" || l.URI != "/api/search?q=bgp" || l.Status != 200 || l.Bytes != 11 || l.DurUS < 0 {
		t.Fatalf("line: %+v", l)
	}
	if _, err := time.Parse(time.RFC3339Nano, l.TS); err != nil {
		t.Fatalf("ts %q: %v", l.TS, err)
	}
}

func TestAccessLogStatus(t *testing.T) {
	l := serveOne(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		httptest.NewRequest(http.MethodGet, "/api/topics/nope", nil))
	if l.Status != 404 {
		t.Fatalf("status %d, want 404", l.Status)
	}
}

func TestAccessLogTruncatesURI(t *testing.T) {
	l := serveOne(t, func(http.ResponseWriter, *http.Request) {},
		httptest.NewRequest(http.MethodGet, "/api/search?q="+strings.Repeat("a", 5000), nil))
	if len(l.URI) != 2048 || !l.Truncated {
		t.Fatalf("uri %d bytes, truncated %v; want 2048 and true", len(l.URI), l.Truncated)
	}
}

func TestAccessLogMethodFolded(t *testing.T) {
	l := serveOne(t, func(http.ResponseWriter, *http.Request) {}, httptest.NewRequest("BREW", "/", nil))
	if l.Method != "OTHER" {
		t.Fatalf("method %q, want OTHER", l.Method)
	}
}

func TestAccessLogNoClientAddress(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	r.RemoteAddr = "203.0.113.7:51234"
	r.Header.Set("X-Forwarded-For", "198.51.100.9")
	var buf syncBuffer
	accessLog(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), &buf, time.Now).ServeHTTP(httptest.NewRecorder(), r)
	line := strings.Join(buf.lines(), "")
	if strings.Contains(line, "203.0.113.7") || strings.Contains(line, "198.51.100.9") {
		t.Fatalf("a client address reached the log: %s", line)
	}
}

func TestAccessLogConcurrent(t *testing.T) {
	var buf syncBuffer
	h := accessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("x")) }), &buf, time.Now)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/stats", nil))
		}()
	}
	wg.Wait()
	lines := buf.lines()
	if len(lines) != 100 {
		t.Fatalf("%d lines, want 100", len(lines))
	}
	for _, l := range lines {
		if !json.Valid([]byte(l)) {
			t.Fatalf("not JSON: %q", l)
		}
	}
}
