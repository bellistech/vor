package stackoverflow

import (
	"strings"
	"testing"
)

func TestToMarkdown_Empty(t *testing.T) {
	out := ToMarkdown(&Result{}, "no hits")
	if !strings.Contains(out, "No results") {
		t.Errorf("empty result missing fallback text: %q", out)
	}
	if !strings.Contains(out, "no hits") {
		t.Errorf("query not echoed: %q", out)
	}
}

func TestToMarkdown_Nil(t *testing.T) {
	out := ToMarkdown(nil, "anything")
	if !strings.Contains(out, "No results") {
		t.Errorf("nil result should produce No results message: %q", out)
	}
}

func TestToMarkdown_Happy(t *testing.T) {
	r := &Result{
		Questions: []Question{{
			Title:        "Why doesn't &lt;tag&gt; work?",
			Link:         "https://stackoverflow.com/q/42",
			Score:        7,
			AnswerCount:  2,
			IsAnswered:   true,
			Tags:         []string{"linux", "lvm"},
			CreationDate: 1700000000,
			Body:         "<p>Try <code>lvextend -L+1G</code> ...</p>",
		}},
		QuotaMax:       10000,
		QuotaRemaining: 9999,
	}
	out := ToMarkdown(r, "test")
	for _, want := range []string{
		"# Stack Overflow: test",
		"Why doesn't <tag> work?", // HTML entities decoded
		"7↑",
		"2 answer(s)",
		"`lvextend -L+1G`",
		"linux",
		"https://stackoverflow.com/q/42",
		"Powered by Stack Exchange",
		"CC BY-SA",
		"9999 / 10000",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing %q\nfull: %s", want, out)
		}
	}
}

func TestStripHTML_Basic(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"<p>hello</p>", "hello"},
		{"a<br>b", "a\nb"},
		{"foo <code>bar</code> baz", "foo `bar` baz"},
		{"<p>one</p><p>two</p>", "one\n\ntwo"},
		{"<pre>code\nblock</pre>", "```\ncode\nblock\n```"},
		{"<p>&amp; entity</p>", "& entity"},
		{"", ""},
	}
	for _, c := range cases {
		got := stripHTML(c.in)
		if got != c.want {
			t.Errorf("stripHTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToMarkdown_BodyTruncation(t *testing.T) {
	long := strings.Repeat("a", 2000)
	r := &Result{
		Questions: []Question{{Title: "x", Link: "y", Body: "<p>" + long + "</p>"}},
	}
	out := ToMarkdown(r, "q")
	if !strings.Contains(out, "…") {
		t.Errorf("expected truncation ellipsis in long body: %s", out[:200])
	}
}
