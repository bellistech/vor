package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/bellistech/vor/internal/registry"
)

// topicReg has one name in two categories. Get("saml") returns the
// longer sheet (auth); the security sheet is reachable only by category.
func topicReg(t *testing.T) *registry.Registry {
	t.Helper()
	reg, err := registry.New(fstest.MapFS{
		"auth/saml.md":     {Data: []byte("# SAML (auth)\n\nThe single sign-on sheet, the longer of the two.\n\n## See Also\n\n- bash\n")},
		"security/saml.md": {Data: []byte("# SAML (security)\n")},
		"shell/bash.md":    {Data: []byte("# Bash\n")},
		"shell/samba.md":   {Data: []byte("# Samba\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func getTopic(t *testing.T, mux *http.ServeMux, target string) (int, map[string]any, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	body := rec.Body.String()
	var m map[string]any
	_ = json.Unmarshal([]byte(body), &m)
	return rec.Code, m, body
}

// short keeps a failure message to one line.
func short(body string) string {
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 80 {
		body = body[:80] + "..."
	}
	return body
}

func TestTopicAPIExactNames(t *testing.T) {
	mux := newMux(topicReg(t))

	// Unknown names are 404. Before 2026-11-12 the API fell back to a
	// fuzzy match (prefix, substring, edit distance) and answered 200
	// with another topic.
	for _, name := range []string{"nosuchtopicxyz", "sa", "sam", "samls", "bas", "amba"} {
		code, _, body := getTopic(t, mux, "/api/topics/"+name)
		if code != http.StatusNotFound {
			t.Errorf("GET /api/topics/%s = %d, want 404 (%s)", name, code, short(body))
		}
	}

	// The name matches in any case and comes back as stored.
	for _, name := range []string{"SAML", "Saml", "saml"} {
		code, m, body := getTopic(t, mux, "/api/topics/"+name)
		if code != http.StatusOK || m["name"] != "saml" || m["category"] != "auth" {
			t.Errorf("GET /api/topics/%s = %d %v/%v, want 200 auth/saml (%s)", name, code, m["category"], m["name"], short(body))
		}
	}
}

func TestTopicAPICategory(t *testing.T) {
	mux := newMux(topicReg(t))
	cases := []struct {
		target   string
		code     int
		category string
	}{
		{"/api/topics/saml?category=auth", 200, "auth"},
		{"/api/topics/saml?category=security", 200, "security"},
		{"/api/topics/SAML?category=security", 200, "security"},
		{"/api/topics/bash?category=shell", 200, "shell"},
		{"/api/topics/saml?category=nosuch", 404, ""},
		{"/api/topics/saml?category=shell", 404, ""},
		{"/api/topics/saml?category=AUTH", 404, ""},
		{"/api/topics/sam?category=auth", 404, ""},
		// An empty category is the same as none.
		{"/api/topics/saml?category=", 200, "auth"},
	}
	for _, c := range cases {
		code, m, body := getTopic(t, mux, c.target)
		if code != c.code {
			t.Errorf("GET %s = %d, want %d (%s)", c.target, code, c.code, short(body))
			continue
		}
		if c.code == 200 && (m["category"] != c.category || m["name"] != strings.ToLower(strings.TrimSuffix(strings.Split(strings.TrimPrefix(c.target, "/api/topics/"), "?")[0], "/"))) {
			t.Errorf("GET %s = %v/%v, want category %q", c.target, m["category"], m["name"], c.category)
		}
	}
}

func TestTopicAPISubRoutesNeedAnExactName(t *testing.T) {
	mux := newMux(topicReg(t))
	for _, target := range []string{
		"/api/topics/nosuch/detail",
		"/api/topics/nosuch/related",
		"/api/topics/sam/detail",
		"/api/topics/sam/related",
		"/api/topics/saml/related?category=nosuch",
	} {
		code, m, body := getTopic(t, mux, target)
		if code != http.StatusNotFound || m["error"] != "not found" {
			t.Errorf("GET %s = %d %q, want 404 not found", target, code, short(body))
		}
	}
	code, m, body := getTopic(t, mux, "/api/topics/saml/related?category=security")
	if code != http.StatusOK || m["topic"] != "saml" {
		t.Errorf("GET related by category = %d (%s)", code, short(body))
	}
}

func TestTopicAPIRefusesBadCategory(t *testing.T) {
	mux := newMux(topicReg(t))
	for _, category := range []string{
		"..",
		"a..b",
		"auth%2Fsaml",
		"auth%5Csaml",
		"auth%00",
		".",
		strings.Repeat("a", 257),
	} {
		code, _, body := getTopic(t, mux, "/api/topics/saml?category="+category)
		if code != http.StatusBadRequest {
			shown := category
			if len(shown) > 20 {
				shown = shown[:20] + "..."
			}
			t.Errorf("category=%s: %d, want 400 (%s)", shown, code, short(body))
		}
	}
	// 256 bytes is allowed through to the lookup, which finds nothing.
	if code, _, _ := getTopic(t, mux, "/api/topics/saml?category="+strings.Repeat("a", 256)); code != http.StatusNotFound {
		t.Errorf("256-byte category: %d, want 404", code)
	}
}

func TestTopicAPITraversalStillRefused(t *testing.T) {
	mux := newMux(topicReg(t))
	for _, name := range []string{"..%2Fetc", "a..b", "a%5Cb", "a%00b"} {
		if code, _, body := getTopic(t, mux, "/api/topics/"+name); code != http.StatusBadRequest {
			t.Errorf("GET /api/topics/%s = %d, want 400 (%s)", name, code, short(body))
		}
	}
}
