package main

import (
	"net/url"
	"testing"

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
