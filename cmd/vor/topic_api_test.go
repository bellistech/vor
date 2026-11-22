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

// dirReg is a repository source: a file's category is its directory, so
// README is one name in four categories, three of them more than one
// segment deep or at the top (".").
func dirReg(t *testing.T) *registry.Registry {
	t.Helper()
	reg, err := registry.New(fstest.MapFS{
		"README.md":          {Data: []byte("# Top\n\nThe top of the tree, the longest README of all four of them.\n")},
		"docs/README.md":     {Data: []byte("# Docs\n")},
		"docs/adr/README.md": {Data: []byte("# ADRs\n")},
		"wiki/README.md":     {Data: []byte("# Wiki\n")},
		"a/b/c/d/e/f.md":     {Data: []byte("# Deep\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

// A category may be a directory: segments under the one-segment rule,
// joined by "/". Before 2026-11-22 vor refused any "/" in ?category= and
// the three READMEs below the top were reachable by nobody.
func TestTopicAPIDirectoryCategory(t *testing.T) {
	mux := newMux(dirReg(t))
	for _, c := range []struct {
		target, category, title string
	}{
		{"/api/topics/README?category=docs/adr", "docs/adr", "ADRs"},
		{"/api/topics/README?category=docs%2Fadr", "docs/adr", "ADRs"},
		{"/api/topics/README?category=docs", "docs", "Docs"},
		{"/api/topics/README?category=wiki", "wiki", "Wiki"},
		{"/api/topics/f?category=a/b/c/d/e", "a/b/c/d/e", "Deep"},
		{"/api/topics/README/related?category=docs/adr", "", ""},
	} {
		code, m, body := getTopic(t, mux, c.target)
		if code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200 (%s)", c.target, code, short(body))
			continue
		}
		if c.category != "" && (m["category"] != c.category || m["title"] != c.title) {
			t.Errorf("GET %s = %v %q, want %s %q", c.target, m["category"], m["title"], c.category, c.title)
		}
	}
	// A well-formed directory that holds no such sheet is 404, not
	// another README.
	for _, target := range []string{
		"/api/topics/README?category=docs/wiki",
		"/api/topics/README?category=adr",
		"/api/topics/README?category=docs/adr/x",
		"/api/topics/README?category=Docs/ADR",
		"/api/topics/f?category=a/b/c/d",
	} {
		if code, _, body := getTopic(t, mux, target); code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404 (%s)", target, code, short(body))
		}
	}
}

// Hostile categories are refused with 400 before the lookup: an empty or
// "." or ".." segment, a leading or trailing slash, a backslash, a NUL, and
// more than maxCategoryLength bytes in all. The top of a source (category
// ".") is not asked for by category; a caller reads it by name alone.
func TestTopicAPIRefusesHostileDirectoryCategory(t *testing.T) {
	mux := newMux(dirReg(t))
	for _, category := range []string{
		"/",
		"//",
		"/docs",
		"docs/",
		"/docs/adr/",
		"docs//adr",
		"docs/./adr",
		"docs/../adr",
		"docs/../../etc",
		"../docs",
		"./docs",
		"docs/..",
		"docs/.",
		"docs/a..b",
		"docs/...",
		"docs%2F..%2Fadr",
		"%2F%2F",
		"docs%5Cadr",
		"docs/adr%5C",
		"docs/adr%00",
		"docs%00/adr",
		".",
		"..",
		strings.Repeat("a/", 128) + "a",
	} {
		code, m, body := getTopic(t, mux, "/api/topics/README?category="+category)
		if code != http.StatusBadRequest || m["error"] != "invalid category" {
			shown := category
			if len(shown) > 24 {
				shown = shown[:24] + "..."
			}
			t.Errorf("category=%s: %d, want 400 invalid category (%s)", shown, code, short(body))
		}
	}
	// 256 bytes in all, segments and slashes, reaches the lookup.
	long := strings.Repeat("a/", 127) + "aa"
	if len(long) != maxCategoryLength {
		t.Fatalf("test category is %d bytes, want %d", len(long), maxCategoryLength)
	}
	if code, _, body := getTopic(t, mux, "/api/topics/README?category="+long); code != http.StatusNotFound {
		t.Errorf("256-byte directory category: %d, want 404 (%s)", code, short(body))
	}
}

func TestValidCategory(t *testing.T) {
	for s, want := range map[string]bool{
		"auth": true, "docs/adr": true, "a/b/c": true, "ramp-up": true, "x.y/z": true,
		"": false, ".": false, "..": false, "/": false, "a/": false, "/a": false, "a//b": false,
		"a/./b": false, "a/../b": false, "a\\b": false, "a\x00b": false, "a/b\x00": false,
		strings.Repeat("a", maxCategoryLength): true, strings.Repeat("a", maxCategoryLength+1): false,
	} {
		if got := validCategory(s); got != want {
			t.Errorf("validCategory(%q) = %v, want %v", s, got, want)
		}
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
