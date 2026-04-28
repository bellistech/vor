// Package stackoverflow is the optional bonus client for Stack Exchange's
// public Search API. It is only invoked when a user has explicitly configured
// a STACK_OVERFLOW_API_KEY (env or ~/.config/cs/secrets.env). The default
// vör experience — the offline encyclopedia — does not touch this package.
//
// Design:
//   - stdlib net/http only (single-binary discipline)
//   - 10s timeout, never a zero-timeout client
//   - URL built via url.Values{} — never string-concat, so the key never
//     leaks into log lines accidentally
//   - JSON unmarshalled into typed structs; we treat the response as
//     untrusted input and accept only the fields we recognize
package stackoverflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bellistech/vor/internal/secrets"
)

// Endpoint is the Stack Exchange v2.3 advanced search URL. Exposed (lowercase
// `endpoint`) via tests through SetTransport so we can swap it for httptest.
var endpoint = "https://api.stackexchange.com/2.3/search/advanced"

// httpClient is the package-level HTTP client. Tests can swap its Transport
// (or replace the whole client via SetClient) for hermetic httptest fakes.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// SetClient overrides the HTTP client. Test-only.
func SetClient(c *http.Client) { httpClient = c }

// SetEndpoint overrides the search URL. Test-only — points at httptest server.
func SetEndpoint(u string) { endpoint = u }

// Question is one search hit, with the body stripped of HTML at render time.
type Question struct {
	Title        string   `json:"title"`
	Link         string   `json:"link"`
	Body         string   `json:"body"`
	Score        int      `json:"score"`
	AnswerCount  int      `json:"answer_count"`
	IsAnswered   bool     `json:"is_answered"`
	Tags         []string `json:"tags"`
	CreationDate int64    `json:"creation_date"`
}

// apiResponse mirrors only the fields we care about from the Stack Exchange
// v2.3 wrapper object. Other fields are silently ignored.
type apiResponse struct {
	Items          []Question `json:"items"`
	HasMore        bool       `json:"has_more"`
	QuotaMax       int        `json:"quota_max"`
	QuotaRemaining int        `json:"quota_remaining"`
	// Backoff is non-zero when the API is asking us to wait that many
	// seconds before our next request. We surface it via Result.Backoff
	// so callers can avoid hammering after a soft throttle.
	Backoff      int    `json:"backoff,omitempty"`
	ErrorID      int    `json:"error_id,omitempty"`
	ErrorName    string `json:"error_name,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Result is the typed search result — what callers render or cache.
type Result struct {
	Questions      []Question `json:"questions"`
	QuotaMax       int        `json:"quota_max"`
	QuotaRemaining int        `json:"quota_remaining"`
	Backoff        int        `json:"backoff,omitempty"`
}

// UserAgent identifies vör in outbound requests. Set externally (cmd/vor)
// to inject the build version; otherwise a generic default is used.
var UserAgent = "vor-cli (https://github.com/bellistech/cs)"

// Common error sentinels (callers can errors.Is them).
var (
	ErrEmptyQuery  = errors.New("empty query")
	ErrAuth        = errors.New("invalid api key")
	ErrRateLimited = errors.New("rate limited or quota exhausted")
	ErrServerError = errors.New("server error")
	ErrMalformed   = errors.New("malformed response")
)

// redactErr returns a new error whose message has every literal `key` replaced
// with "***", so the API key cannot leak via error chains. Preserves the type
// only as a plain error — callers should errors.Is the sentinels above.
func redactErr(err error, key string) error {
	if err == nil {
		return nil
	}
	return errors.New(secrets.Redact(err.Error(), key))
}

// Search runs the live Stack Overflow search. The key is required (no key →
// caller should not invoke Search; that's a friendly-error path higher up).
// Returns a redacted error if the key was reflected in the wrapped message.
func Search(ctx context.Context, query, key string) (*Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}

	q := url.Values{}
	q.Set("order", "desc")
	q.Set("sort", "relevance")
	q.Set("site", "stackoverflow") // api_site_parameter short form
	q.Set("filter", "withbody")    // built-in named filter — adds the body field
	q.Set("pagesize", "10")        // terminal-friendly cap (API default is 30)
	q.Set("q", query)
	q.Set("key", key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, redactErr(err, key)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	// IMPORTANT: do NOT set Accept-Encoding here. The Stack Exchange API
	// always returns gzipped responses, and Go's net/http auto-decodes
	// gzip ONLY when the user did not explicitly request it. Setting
	// Accept-Encoding manually disables auto-decompression, which would
	// leave us holding a gzipped byte slice that won't parse as JSON.

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, redactErr(err, key)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20)) // 4 MB cap
	if err != nil {
		return nil, redactErr(err, key)
	}

	var ar apiResponse
	if jerr := json.Unmarshal(body, &ar); jerr != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformed, jerr)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized,
		resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w: %s", ErrAuth, ar.ErrorMessage)
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %s", ErrRateLimited, ar.ErrorMessage)
	case resp.StatusCode >= 500:
		return nil, fmt.Errorf("%w: status %d", ErrServerError, resp.StatusCode)
	case resp.StatusCode >= 400:
		// generic 4xx — surface the api error_message but redact the key
		msg := ar.ErrorMessage
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return nil, redactErr(errors.New(msg), key)
	}

	if ar.QuotaRemaining == 0 && len(ar.Items) == 0 {
		return nil, ErrRateLimited
	}

	return &Result{
		Questions:      ar.Items,
		QuotaMax:       ar.QuotaMax,
		QuotaRemaining: ar.QuotaRemaining,
		Backoff:        ar.Backoff,
	}, nil
}
