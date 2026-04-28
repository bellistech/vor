package stackoverflow

import (
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
