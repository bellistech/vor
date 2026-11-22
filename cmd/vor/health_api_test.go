package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The platform's health sweep GETs /health on every service it watches and
// counts only a 200. Before 2026-11-22 vor answered /health with 404 and
// had its liveness only under /api/health.
func TestHealthAtRoot(t *testing.T) {
	mux := newMux(topicReg(t))
	for _, target := range []string{"/health", "/api/health"} {
		code, m, body := getTopic(t, mux, target)
		if code != http.StatusOK || m["status"] != "ok" {
			t.Errorf("GET %s = %d %v, want 200 status ok (%s)", target, code, m["status"], short(body))
		}
		if m["sheets"] != float64(4) {
			t.Errorf("GET %s sheets = %v, want 4", target, m["sheets"])
		}
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("HEAD /health = %d, want 200", rec.Code)
	}

	// /health takes no input: a write method is refused, and nothing
	// below it is served.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/health", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /health = %d, want 405", rec.Code)
	}
	if code, _, _ := getTopic(t, mux, "/health/x"); code != http.StatusNotFound {
		t.Errorf("GET /health/x = %d, want 404", code)
	}
}
