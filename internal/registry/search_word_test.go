package registry

import (
	"strings"
	"testing"
	"testing/fstest"
)

func wordFS() fstest.MapFS {
	return fstest.MapFS{
		// "ccna" appears only inside KRB5CCNAME: a substring, not a word.
		"auth/kerberos.md": {Data: []byte("# Kerberos\n\nTicket caches.\n\n## Environment\n\nexport KRB5CCNAME=/tmp/krb5cc_0\n")},
		"certs/ccna.md":    {Data: []byte("# CCNA\n\nCisco exam notes.\n\n## Topics\n\nThe ccna covers switching.\n")},
		"wiki/memory.md":   {Data: []byte("# Memory\n\n## Owner\n\nThis is wotan's ring.\nSee Wotan-Memory for the layout.\npath: services/wotan/main.go\nmetric unheaded_wotan_up\nhost.wotan.lan\n")},
		// "wotan" appears only inside another word here.
		"wiki/other.md":    {Data: []byte("# Other\n\n## Notes\n\nThe antiwotan and xwotan forms.\n")},
		"shell/scripts.md": {Data: []byte("# Shell scripting\n\n## Loops\n\nshell loops and scripting idioms\n")},
		// Has both halves only as tails of other words.
		"shell/tails.md": {Data: []byte("# Tails\n\n## Notes\n\nnutshell descripting\n")},
	}
}

func sheetNames(ms []Match) map[string]bool {
	out := make(map[string]bool)
	for _, m := range ms {
		out[m.Sheet.Category+"/"+m.Sheet.Name] = true
	}
	return out
}

func TestSearchWordsNeedsAWordStart(t *testing.T) {
	reg, err := New(wordFS())
	if err != nil {
		t.Fatal(err)
	}

	// Default search is unchanged: a substring is enough.
	got := sheetNames(reg.Search("ccna"))
	if !got["auth/kerberos"] || !got["certs/ccna"] {
		t.Errorf("Search(ccna) = %v, want kerberos and ccna", got)
	}

	got = sheetNames(reg.SearchWords("ccna"))
	if got["auth/kerberos"] {
		t.Errorf("SearchWords(ccna) matched KRB5CCNAME: %v", got)
	}
	if !got["certs/ccna"] || len(got) != 1 {
		t.Errorf("SearchWords(ccna) = %v, want only certs/ccna", got)
	}
}

func TestSearchWordsBoundaries(t *testing.T) {
	reg, err := New(wordFS())
	if err != nil {
		t.Fatal(err)
	}
	ms := reg.SearchWords("wotan")
	got := sheetNames(ms)
	if !got["wiki/memory"] {
		t.Fatalf("SearchWords(wotan) = %v, want wiki/memory", got)
	}
	if got["wiki/other"] {
		t.Errorf("SearchWords(wotan) matched antiwotan/xwotan: %v", got)
	}
	// Apostrophe, hyphen, slash, underscore, dot, space and line start
	// all count as the start of a word.
	want := []string{"wotan's ring", "Wotan-Memory", "services/wotan/main.go", "unheaded_wotan_up", "host.wotan.lan"}
	for _, w := range want {
		found := false
		for _, m := range ms {
			if strings.Contains(m.Line, w) {
				found = true
			}
		}
		if !found {
			t.Errorf("SearchWords(wotan): no line with %q", w)
		}
	}
	// Every returned line carries the term at a word start.
	for _, m := range ms {
		if !startsWord(strings.ToLower(m.Line), "wotan") {
			t.Errorf("SearchWords(wotan) returned a line without the word: %q", m.Line)
		}
	}
}

func TestSearchWordsKeepsTheHyphenSplit(t *testing.T) {
	reg, err := New(wordFS())
	if err != nil {
		t.Fatal(err)
	}
	got := sheetNames(reg.SearchWords("shell-scripting"))
	if !got["shell/scripts"] {
		t.Errorf("SearchWords(shell-scripting) = %v, want shell/scripts", got)
	}
	if got["shell/tails"] {
		t.Errorf("SearchWords(shell-scripting) matched nutshell/descripting: %v", got)
	}
	// Substring mode still finds both.
	if got := sheetNames(reg.Search("shell-scripting")); !got["shell/tails"] || !got["shell/scripts"] {
		t.Errorf("Search(shell-scripting) = %v, want scripts and tails", got)
	}
}

func TestSearchWordsFallsBackToLooseLines(t *testing.T) {
	reg, err := New(wordFS())
	if err != nil {
		t.Fatal(err)
	}
	// Both words are in the sheet, never on one line: the lines that
	// carry either word come back, as in substring mode.
	ms := reg.SearchWords("ring layout")
	if len(ms) == 0 {
		t.Fatal("SearchWords(ring layout) = nothing")
	}
	for _, m := range ms {
		l := strings.ToLower(m.Line)
		if !startsWord(l, "ring") && !startsWord(l, "layout") {
			t.Errorf("loose line without either word: %q", m.Line)
		}
	}
}

func TestSearchWordsTermCap(t *testing.T) {
	reg, err := New(wordFS())
	if err != nil {
		t.Fatal(err)
	}
	// 16 terms that match, then one that does not. The 17th is dropped,
	// as in Search, so the sheet still matches.
	q := strings.Repeat("ccna ", 1) + "cisco exam notes topics the covers switching c ci cis ex exa no not top zzznomatch"
	if n := len(strings.Fields(q)); n != 17 {
		t.Fatalf("test query has %d terms, want 17", n)
	}
	if got := sheetNames(reg.SearchWords(q)); !got["certs/ccna"] {
		t.Errorf("SearchWords with 17 terms = %v, want certs/ccna (17th dropped)", got)
	}
	if len(reg.SearchWords()) != 0 || len(reg.SearchWords("   ")) != 0 {
		t.Error("SearchWords with no terms returned matches")
	}
}

func TestStartsWord(t *testing.T) {
	cases := []struct {
		hay, term string
		want      bool
	}{
		{"ccna", "ccna", true},
		{"the ccna exam", "ccna", true},
		{"krb5ccname", "ccna", false},
		{"xccna ccna", "ccna", true}, // a later hit counts
		{"ccnax", "ccna", true},      // start of word; the tail is free
		{"a_ccna", "ccna", true},
		{"a-ccna", "ccna", true},
		{"a/ccna", "ccna", true},
		{"a.ccna", "ccna", true},
		{"a'ccna", "ccna", true},
		{"(ccna)", "ccna", true},
		{"9ccna", "ccna", false},
		{"é ccna", "ccna", true},
		{"", "ccna", false},
		{"ccna", "", false},
	}
	for _, c := range cases {
		if got := startsWord(c.hay, c.term); got != c.want {
			t.Errorf("startsWord(%q, %q) = %v, want %v", c.hay, c.term, got, c.want)
		}
	}
}
