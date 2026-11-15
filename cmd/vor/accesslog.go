package main

// accesslog.go: one JSON line per request for `vor serve` (Unheaded ADR-147
// decision 23): time, method, URI, status, bytes and microseconds, so a day
// of requests can be measured and replayed. No client address is written:
// a recorded day then needs no scrubbing.

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// maxLoggedURI bounds the uri field; a longer one is cut and flagged.
const maxLoggedURI = 2048

type accessEntry struct {
	TS        string `json:"ts"`
	Method    string `json:"method"`
	URI       string `json:"uri"`
	Truncated bool   `json:"uri_truncated,omitempty"`
	Status    int    `json:"status"`
	Bytes     int64  `json:"bytes"`
	DurUS     int64  `json:"dur_us"`
}

// countingWriter keeps the status (200 when the handler never sets one) and
// counts the body bytes written.
type countingWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (c *countingWriter) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	n, err := c.ResponseWriter.Write(p)
	c.bytes += int64(n)
	return n, err
}

// foldMethod keeps the label set bounded: anything outside the standard
// methods is OTHER.
func foldMethod(m string) string {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodOptions, http.MethodPatch:
		return m
	}
	return "OTHER"
}

// accessLog wraps next and writes one JSON line per request to w.
func accessLog(next http.Handler, w io.Writer, now func() time.Time) http.Handler {
	var mu sync.Mutex
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := now()
		cw := &countingWriter{ResponseWriter: rw}
		next.ServeHTTP(cw, r)
		if cw.status == 0 {
			cw.status = http.StatusOK
		}
		e := accessEntry{
			TS:     start.UTC().Format(time.RFC3339Nano),
			Method: foldMethod(r.Method),
			URI:    r.URL.RequestURI(),
			Status: cw.status,
			Bytes:  cw.bytes,
			DurUS:  now().Sub(start).Microseconds(),
		}
		if len(e.URI) > maxLoggedURI {
			e.URI, e.Truncated = e.URI[:maxLoggedURI], true
		}
		line, err := json.Marshal(e)
		if err != nil {
			return
		}
		line = append(line, '\n')
		mu.Lock()
		_, _ = w.Write(line)
		mu.Unlock()
	})
}
