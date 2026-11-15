package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/bellistech/vor/internal/registry"
)

func TestSearchLimits(t *testing.T) {
	cases := []struct {
		query           string
		limit, perTopic int
		ok              bool
	}{
		{"q=x", 50, 0, true},
		{"q=x&limit=1", 1, 0, true},
		{"q=x&limit=500&per_topic=50", 500, 50, true},
		{"q=x&per_topic=3", 50, 3, true},
		{"q=x&limit=0", 0, 0, false},
		{"q=x&limit=501", 0, 0, false},
		{"q=x&limit=-1", 0, 0, false},
		{"q=x&limit=", 0, 0, false},
		{"q=x&limit=ten", 0, 0, false},
		{"q=x&limit=1e2", 0, 0, false},
		{"q=x&per_topic=0", 0, 0, false},
		{"q=x&per_topic=51", 0, 0, false},
		{"q=x&limit=99999999999999999999", 0, 0, false},
	}
	for _, c := range cases {
		v, err := url.ParseQuery(c.query)
		if err != nil {
			t.Fatal(err)
		}
		limit, perTopic, err := searchLimits(v)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.query, err, c.ok)
			continue
		}
		if c.ok && (limit != c.limit || perTopic != c.perTopic) {
			t.Errorf("%s: got %d, %d; want %d, %d", c.query, limit, perTopic, c.limit, c.perTopic)
		}
	}
}

func TestLimitMatches(t *testing.T) {
	long := &registry.Sheet{Name: "long", Category: "a"}
	short := &registry.Sheet{Name: "short", Category: "a"}
	// Same name as long, other category: a different topic.
	twin := &registry.Sheet{Name: "long", Category: "b"}

	var ms []registry.Match
	for i := 0; i < 60; i++ {
		ms = append(ms, registry.Match{Sheet: long, Section: string(rune('A'+i%26)) + string(rune('a'+i/26)), Line: "x"})
	}
	ms = append(ms,
		registry.Match{Sheet: short, Section: "S", Line: "first"},
		registry.Match{Sheet: short, Section: "S", Line: "same section"},
		registry.Match{Sheet: short, Section: "T", Line: "second"},
		registry.Match{Sheet: twin, Section: "A" + "a", Line: "twin"},
	)

	// No per-topic cap: the long topic fills the answer, as before.
	got := limitMatches(ms, "", 50, 0)
	if len(got) != 50 {
		t.Fatalf("no cap: %d lines, want 50", len(got))
	}
	for _, m := range got {
		if m.Sheet != long {
			t.Fatalf("no cap: %s/%s in the answer, want only the long topic", m.Sheet.Category, m.Sheet.Name)
		}
	}

	// per_topic 2: every topic shows, in order, one line per section.
	got = limitMatches(ms, "", 50, 2)
	var lines []string
	for _, m := range got {
		lines = append(lines, m.Sheet.Category+"/"+m.Sheet.Name+":"+m.Line)
	}
	want := []string{"a/long:x", "a/long:x", "a/short:first", "a/short:second", "b/long:twin"}
	if len(lines) != len(want) {
		t.Fatalf("per_topic 2: got %v, want %v", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("per_topic 2: got %v, want %v", lines, want)
		}
	}

	// category keeps only that exact category.
	got = limitMatches(ms, "b", 50, 2)
	if len(got) != 1 || got[0].Sheet != twin {
		t.Fatalf("category b: got %d lines, want the twin only", len(got))
	}
	if got := limitMatches(ms, "nosuch", 50, 0); len(got) != 0 {
		t.Fatalf("unknown category: %d lines, want 0", len(got))
	}

	// limit still applies.
	if got := limitMatches(ms, "", 3, 2); len(got) != 3 {
		t.Fatalf("limit 3: %d lines", len(got))
	}
}

func searchTopics(t *testing.T, mux *http.ServeMux, query string) (int, []string, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/search?"+query, nil))
	var rows []struct {
		Topic    string `json:"topic"`
		Category string `json:"category"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &rows)
	var topics []string
	for _, r := range rows {
		topics = append(topics, r.Category+"/"+r.Topic)
	}
	return rec.Code, topics, rec.Body.String()
}

func TestSearchAPIMatch(t *testing.T) {
	reg, err := registry.New(fstest.MapFS{
		"auth/kerberos.md":  {Data: []byte("# Kerberos\n\n## Environment\n\nexport KRB5CCNAME=/tmp/krb5cc_0\n")},
		"certs/ccna.md":     {Data: []byte("# CCNA\n\n## Topics\n\nThe ccna covers switching.\n\n## Labs\n\nccna lab one\n\n## Exam\n\nccna exam day\n")},
		"net/ccna-notes.md": {Data: []byte("# Notes\n\n## Scope\n\nccna scope\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := newMux(reg)

	has := func(topics []string, want string) bool {
		for _, x := range topics {
			if x == want {
				return true
			}
		}
		return false
	}

	// No match parameter and match=substring are the same search.
	for _, q := range []string{"q=ccna", "q=ccna&match=substring"} {
		code, topics, body := searchTopics(t, mux, q)
		if code != 200 || !has(topics, "auth/kerberos") || !has(topics, "certs/ccna") {
			t.Errorf("%s = %d %v, want kerberos and ccna (%s)", q, code, topics, short(body))
		}
	}

	// match=word drops the sheet that has the term only inside a word.
	code, topics, body := searchTopics(t, mux, "q=ccna&match=word")
	if code != 200 || has(topics, "auth/kerberos") || !has(topics, "certs/ccna") || !has(topics, "net/ccna-notes") {
		t.Errorf("match=word = %d %v, want ccna sheets without kerberos (%s)", code, topics, short(body))
	}

	// It combines with category, limit and per_topic.
	code, topics, body = searchTopics(t, mux, "q=ccna&match=word&category=certs")
	if code != 200 || len(topics) == 0 {
		t.Errorf("match=word&category=certs = %d %v (%s)", code, topics, short(body))
	}
	for _, x := range topics {
		if x != "certs/ccna" {
			t.Errorf("match=word&category=certs returned %s", x)
		}
	}
	code, topics, _ = searchTopics(t, mux, "q=ccna&match=word&category=auth")
	if code != 200 || len(topics) != 0 {
		t.Errorf("match=word&category=auth = %d %v, want no rows", code, topics)
	}
	code, topics, _ = searchTopics(t, mux, "q=ccna&match=word&limit=1")
	if code != 200 || len(topics) != 1 {
		t.Errorf("match=word&limit=1 = %d %v, want one row", code, topics)
	}
	code, topics, _ = searchTopics(t, mux, "q=ccna&match=word&per_topic=1")
	if code != 200 || len(topics) != 2 {
		t.Errorf("match=word&per_topic=1 = %d %v, want one row for each of two topics", code, topics)
	}

	// Anything else is refused with a JSON error.
	for _, q := range []string{"q=ccna&match=other", "q=ccna&match=WORD", "q=ccna&match=", "q=ccna&match=word&match=substring"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/search?"+q, nil))
		var e map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &e)
		if rec.Code != 400 || !strings.Contains(e["error"], "match") {
			t.Errorf("%s = %d %q, want 400 with an error naming match", q, rec.Code, short(rec.Body.String()))
		}
	}
}

func TestSearchMatchParam(t *testing.T) {
	cases := []struct {
		query string
		words bool
		ok    bool
	}{
		{"q=x", false, true},
		{"q=x&match=substring", false, true},
		{"q=x&match=word", true, true},
		{"q=x&match=", false, false},
		{"q=x&match=words", false, false},
		{"q=x&match=Word", false, false},
		{"q=x&match=word&match=word", false, false},
	}
	for _, c := range cases {
		v, err := url.ParseQuery(c.query)
		if err != nil {
			t.Fatal(err)
		}
		words, err := searchMatch(v)
		if (err == nil) != c.ok || words != c.words {
			t.Errorf("%s: got %v, %v; want words %v ok %v", c.query, words, err, c.words, c.ok)
		}
	}
}
