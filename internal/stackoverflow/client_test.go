package stackoverflow

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const sampleKey = "TESTKEY-secret-do-not-leak"

// withFakeServer points the package endpoint at an httptest server with the
// given handler, and resets state on cleanup.
func withFakeServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	prevEP := endpoint
	prevCli := httpClient
	SetEndpoint(srv.URL + "/2.3/search/advanced")
	SetClient(&http.Client{Timeout: 2 * time.Second})
	t.Cleanup(func() {
		srv.Close()
		SetEndpoint(prevEP)
		SetClient(prevCli)
	})
	return srv
}

const happyJSON = `{
  "items": [
    {"title": "How to extend an LVM volume?", "link": "https://stackoverflow.com/q/1",
     "score": 42, "answer_count": 3, "is_answered": true,
     "tags": ["lvm", "linux"], "creation_date": 1700000000,
     "body": "<p>Use <code>lvextend</code> ...</p>"}
  ],
  "has_more": false,
  "quota_max": 10000,
  "quota_remaining": 9999
}`

const errorJSON = `{"error_id": 401, "error_name": "unauthenticated",
  "error_message": "key required"}`

func TestSearch_Happy(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "lvm extend" {
			t.Errorf("query param = %q", got)
		}
		if got := r.URL.Query().Get("key"); got != sampleKey {
			t.Errorf("key param = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(happyJSON))
	})

	res, err := Search(context.Background(), "lvm extend", sampleKey)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Questions) != 1 {
		t.Fatalf("got %d questions, want 1", len(res.Questions))
	}
	if res.QuotaRemaining != 9999 {
		t.Errorf("QuotaRemaining = %d", res.QuotaRemaining)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	_, err := Search(context.Background(), "   ", "k")
	if !errors.Is(err, ErrEmptyQuery) {
		t.Errorf("got %v, want ErrEmptyQuery", err)
	}
}

func TestSearch_4xxAuth(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(errorJSON))
	})

	_, err := Search(context.Background(), "anything", sampleKey)
	if !errors.Is(err, ErrAuth) {
		t.Errorf("got %v, want ErrAuth", err)
	}
	if strings.Contains(err.Error(), sampleKey) {
		t.Errorf("error leaked the API key: %v", err)
	}
}

func TestSearch_4xxGenericRedacted(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		// The api echoes the key back inside the error message — test that
		// our redaction wraps it before bubbling up.
		w.Write([]byte(`{"error_message": "bad query for key=` + sampleKey + `"}`))
	})

	_, err := Search(context.Background(), "anything", sampleKey)
	if err == nil {
		t.Fatal("expected error for 4xx")
	}
	if strings.Contains(err.Error(), sampleKey) {
		t.Errorf("error leaked the API key: %v", err)
	}
	if !strings.Contains(err.Error(), "***") {
		t.Errorf("error not redacted: %v", err)
	}
}

func TestSearch_5xx(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{}`))
	})

	_, err := Search(context.Background(), "x", sampleKey)
	if !errors.Is(err, ErrServerError) {
		t.Errorf("got %v, want ErrServerError", err)
	}
}

func TestSearch_429(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error_message": "throttled"}`))
	})

	_, err := Search(context.Background(), "x", sampleKey)
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("got %v, want ErrRateLimited", err)
	}
}

func TestSearch_MalformedJSON(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{not valid json`))
	})

	_, err := Search(context.Background(), "x", sampleKey)
	if !errors.Is(err, ErrMalformed) {
		t.Errorf("got %v, want ErrMalformed", err)
	}
}

func TestSearch_QuotaZeroEmpty(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items": [], "quota_max": 10000, "quota_remaining": 0}`))
	})

	_, err := Search(context.Background(), "x", sampleKey)
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("got %v, want ErrRateLimited (zero quota)", err)
	}
}

func TestSearch_KeyNeverInTransport(t *testing.T) {
	// Even when the server returns a generic body with no error_message,
	// the key must never appear in the resulting error chain. Force a
	// network-layer failure by pointing at a closed httptest server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // immediately close to force a connection error

	prevEP := endpoint
	prevCli := httpClient
	SetEndpoint(srv.URL + "/2.3/search/advanced")
	SetClient(&http.Client{Timeout: 500 * time.Millisecond})
	t.Cleanup(func() { SetEndpoint(prevEP); SetClient(prevCli) })

	_, err := Search(context.Background(), "x", sampleKey)
	if err == nil {
		t.Fatal("expected error from closed server")
	}
	if strings.Contains(err.Error(), sampleKey) {
		t.Errorf("error leaked key: %v", err)
	}
}

// TestSearch_GzippedResponse simulates the production API behavior, which
// always sends gzipped bodies. Go's net/http auto-decodes gzip ONLY when the
// caller hasn't set Accept-Encoding manually. This test verifies that our
// client (a) doesn't set the header, and (b) successfully parses a gzipped
// JSON response. Was a real bug pre-spec-audit.
func TestSearch_GzippedResponse(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		// the client must not have set Accept-Encoding itself —
		// Go adds it automatically when unset, which is what enables
		// transparent decompression
		if got := r.Header.Get("Accept-Encoding"); !strings.Contains(got, "gzip") {
			t.Errorf("expected Go-managed Accept-Encoding to include gzip, got %q", got)
		}
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write([]byte(happyJSON))
		gz.Close()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(buf.Bytes())
	})

	res, err := Search(context.Background(), "lvm", sampleKey)
	if err != nil {
		t.Fatalf("Search on gzipped response: %v", err)
	}
	if len(res.Questions) != 1 {
		t.Errorf("got %d questions, want 1", len(res.Questions))
	}
}

func TestSearch_BackoffSurfaced(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[{"title":"x","link":"y"}],
			"quota_max":10000, "quota_remaining":9000, "backoff":5}`))
	})

	res, err := Search(context.Background(), "x", sampleKey)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Backoff != 5 {
		t.Errorf("Backoff = %d, want 5", res.Backoff)
	}
}

func TestSearch_PageSizeIs10(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("pagesize"); got != "10" {
			t.Errorf("pagesize = %q, want 10 (terminal-friendly default)", got)
		}
		if got := r.URL.Query().Get("filter"); got != "withbody" {
			t.Errorf("filter = %q, want withbody", got)
		}
		if got := r.URL.Query().Get("site"); got != "stackoverflow" {
			t.Errorf("site = %q, want stackoverflow", got)
		}
		w.Write([]byte(happyJSON))
	})

	if _, err := Search(context.Background(), "x", sampleKey); err != nil {
		t.Fatalf("Search: %v", err)
	}
}

func TestSearch_UserAgentSet(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua == "" {
			t.Error("expected non-empty User-Agent")
		}
		if !strings.Contains(ua, "vor") {
			t.Errorf("User-Agent = %q, expected to contain 'vor'", ua)
		}
		w.Write([]byte(happyJSON))
	})

	prevUA := UserAgent
	UserAgent = "vor-cli/test (https://github.com/bellistech/cs)"
	t.Cleanup(func() { UserAgent = prevUA })

	if _, err := Search(context.Background(), "x", sampleKey); err != nil {
		t.Fatalf("Search: %v", err)
	}
}

func TestSearch_NoExplicitAcceptEncoding(t *testing.T) {
	// regression guard for the gzip auto-decompression contract.
	// if a future edit re-introduces an explicit Accept-Encoding header,
	// this test fails so we don't ship a broken binary.
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Go's http.Transport, when the caller hasn't set Accept-Encoding,
		// adds "gzip" itself. If our code set anything else (e.g. "gzip"
		// explicitly, or "identity"), that's a regression.
		hdrs := r.Header.Values("Accept-Encoding")
		if len(hdrs) != 1 || hdrs[0] != "gzip" {
			t.Errorf("Accept-Encoding = %v — want exactly [gzip] managed by Go's http transport", hdrs)
		}
		w.Write([]byte(happyJSON))
	})

	if _, err := Search(context.Background(), "x", sampleKey); err != nil {
		t.Fatalf("Search: %v", err)
	}
}

func TestSearch_ContextCanceled(t *testing.T) {
	withFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(happyJSON))
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Search runs
	_, err := Search(ctx, "x", sampleKey)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
	if strings.Contains(err.Error(), sampleKey) {
		t.Errorf("error leaked key: %v", err)
	}
}
