package registry

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"storage/lvm.md": &fstest.MapFile{
			Data: []byte(`# LVM (Logical Volume Manager)

Storage virtualization for logical volumes.

## Physical Volumes

` + "```bash" + `
pvs
pvdisplay
pvcreate /dev/sdb1
` + "```" + `

## Volume Groups

` + "```bash" + `
vgs
vgcreate myvg /dev/sdb1
` + "```" + `

## Logical Volumes

### Create

` + "```bash" + `
lvcreate -L 10G -n myvolume myvg
` + "```" + `

### Extend

` + "```bash" + `
lvextend -L +5G /dev/myvg/myvolume
lvextend -r -L +5G /dev/myvg/myvolume
` + "```" + `
`),
		},
		"networking/ss.md": &fstest.MapFile{
			Data: []byte(`# ss (Socket Statistics)

Display socket information, replacement for netstat.

## List Listening Ports

` + "```bash" + `
ss -tulpn
` + "```" + `

## TCP Connections

` + "```bash" + `
ss -t
ss -ta
` + "```" + `
`),
		},
		"shell/bash.md": &fstest.MapFile{
			Data: []byte(`# Bash

Bourne Again Shell — the default shell on most Linux systems.

## Variables

` + "```bash" + `
NAME="world"
echo "Hello $NAME"
` + "```" + `
`),
		},
	}
}

func TestNew(t *testing.T) {
	reg, err := New(testFS())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if got := len(reg.List()); got != 3 {
		t.Errorf("List() = %d sheets, want 3", got)
	}
}

func TestGet(t *testing.T) {
	reg, _ := New(testFS())

	s := reg.Get("lvm")
	if s == nil {
		t.Fatal("Get(lvm) = nil")
	}
	if s.Title != "LVM (Logical Volume Manager)" {
		t.Errorf("Title = %q", s.Title)
	}
	if s.Category != "storage" {
		t.Errorf("Category = %q", s.Category)
	}
	if !strings.Contains(s.Description, "Storage virtualization") {
		t.Errorf("Description = %q", s.Description)
	}
}

func TestGetUnknown(t *testing.T) {
	reg, _ := New(testFS())
	if s := reg.Get("nonexistent"); s != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", s)
	}
}

func TestCategories(t *testing.T) {
	reg, _ := New(testFS())
	cats := reg.Categories()
	if len(cats) != 3 {
		t.Fatalf("Categories() = %v, want 3 categories", cats)
	}
	// Should be sorted
	if cats[0] != "networking" || cats[1] != "shell" || cats[2] != "storage" {
		t.Errorf("Categories() = %v", cats)
	}
}

func TestIsCategory(t *testing.T) {
	reg, _ := New(testFS())
	if !reg.IsCategory("storage") {
		t.Error("IsCategory(storage) = false")
	}
	if reg.IsCategory("nope") {
		t.Error("IsCategory(nope) = true")
	}
}

func TestByCategory(t *testing.T) {
	reg, _ := New(testFS())
	sheets := reg.ByCategory("storage")
	if len(sheets) != 1 {
		t.Fatalf("ByCategory(storage) = %d, want 1", len(sheets))
	}
	if sheets[0].Name != "lvm" {
		t.Errorf("Name = %q", sheets[0].Name)
	}
}

func TestSearch(t *testing.T) {
	reg, _ := New(testFS())

	matches := reg.Search("lvextend")
	if len(matches) == 0 {
		t.Fatal("Search(lvextend) = no results")
	}
	if matches[0].Sheet.Name != "lvm" {
		t.Errorf("match sheet = %q, want lvm", matches[0].Sheet.Name)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	reg, _ := New(testFS())

	matches := reg.Search("LVEXTEND")
	if len(matches) == 0 {
		t.Fatal("Search(LVEXTEND) = no results")
	}
}

func TestSearchMultipleTermsAND(t *testing.T) {
	reg, _ := New(testFS())

	// Both terms appear in the lvm sheet → match, scoped to that sheet only.
	matches := reg.Search("lvm", "extend")
	if len(matches) == 0 {
		t.Fatal("Search(lvm, extend) = no results, want lvm matches")
	}
	for _, m := range matches {
		if m.Sheet.Name != "lvm" {
			t.Errorf("multi-term match returned %q, want only lvm", m.Sheet.Name)
		}
	}

	// Term that doesn't co-occur in any fixture sheet → no results.
	if got := reg.Search("lvextend", "kubernetes"); len(got) != 0 {
		t.Errorf("Search(lvextend, kubernetes) = %d, want 0", len(got))
	}
}

func TestSearchPrefersStrictLines(t *testing.T) {
	reg, _ := New(testFS())

	// "lvextend" and "myvg" co-occur on the same lvextend command lines in
	// the fixture, so the strict pass should return only those lines and
	// skip looser any-term matches like the "## Logical Volumes" header.
	matches := reg.Search("lvextend", "myvg")
	if len(matches) == 0 {
		t.Fatal("Search(lvextend, myvg) = no results")
	}
	for _, m := range matches {
		line := strings.ToLower(m.Line)
		if !strings.Contains(line, "lvextend") || !strings.Contains(line, "myvg") {
			t.Errorf("strict pass should only return lines with both terms, got %q", m.Line)
		}
	}
}

func TestSearchSplitsWhitespace(t *testing.T) {
	reg, _ := New(testFS())
	a := reg.Search("lvm extend")
	b := reg.Search("lvm", "extend")
	if len(a) == 0 || len(a) != len(b) {
		t.Errorf("space-separated and variadic forms differ: %d vs %d", len(a), len(b))
	}
}

func TestSearchEmpty(t *testing.T) {
	reg, _ := New(testFS())
	if got := reg.Search(""); got != nil {
		t.Errorf("Search(\"\") = %d matches, want nil", len(got))
	}
}

func TestSearchRanksNameMatchFirst(t *testing.T) {
	// Two sheets both contain the terms "python" and "list", but only one
	// has "python" in its name. The named sheet must rank first.
	fs := fstest.MapFS{
		"languages/python.md": &fstest.MapFile{
			Data: []byte("# Python\n\nA language.\n\n## Lists\n\nlist examples\n"),
		},
		"ai-ml/notes.md": &fstest.MapFile{
			Data: []byte("# Notes\n\nUsing python for list operations.\n"),
		},
	}
	reg, err := New(fs)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	matches := reg.Search("python", "list")
	if len(matches) == 0 {
		t.Fatal("Search(python, list) = no results")
	}
	if matches[0].Sheet.Name != "python" {
		t.Errorf("first match sheet = %q, want python (name match should rank first)", matches[0].Sheet.Name)
	}
}

func TestSearchRanksShorterNameOnTokenTie(t *testing.T) {
	// Both sheets score one whole-token name match (python's name has
	// "python", merge-k-sorted-lists' name has "lists") AND have multiple
	// strict-AND lines. The single-token "python" must win because more of
	// the sheet's name is captured by the query.
	fs := fstest.MapFS{
		"languages/python.md": &fstest.MapFile{
			Data: []byte("# Python\n\n## Lists\n\nPython lists are mutable.\nMore python lists examples.\n"),
		},
		"coding-problems/merge-k-sorted-lists.md": &fstest.MapFile{
			Data: []byte("# Merge K Sorted Lists\n\nPython lists merge.\nIn Python, lists work like so.\nPython lists everywhere.\n"),
		},
	}
	reg, err := New(fs)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	matches := reg.Search("python", "lists")
	if len(matches) == 0 {
		t.Fatal("no results")
	}
	if matches[0].Sheet.Name != "python" {
		t.Errorf("first match = %q, want python (single-token name should beat multi-token name on whole-match tie)", matches[0].Sheet.Name)
	}
}

func TestSearchSplitsHyphenatedTerm(t *testing.T) {
	// A user typing `cs -s shell-scripting` wants the shell-scripting sheet,
	// not the bash sheet that just mentions it. Hyphens in the search term
	// must split into the same sub-tokens used when tokenizing sheet names.
	fs := fstest.MapFS{
		"shell/shell-scripting.md": &fstest.MapFile{
			Data: []byte(`# Shell Scripting

## Setup

Portable Bourne shell scripts run on dash, busybox, and ksh.
`),
		},
		"shell/bash.md": &fstest.MapFile{
			Data: []byte(`# Bash

## See Also

- shell-scripting
`),
		},
	}
	reg, err := New(fs)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	matches := reg.Search("shell-scripting")
	if len(matches) == 0 {
		t.Fatal("Search(shell-scripting) = no results")
	}
	if matches[0].Sheet.Name != "shell-scripting" {
		t.Errorf("first match = %q, want shell-scripting (hyphen should split into tokens matching the name)", matches[0].Sheet.Name)
	}
}

func TestSearchPrefersTitleHitSections(t *testing.T) {
	// Two sections both contain strict-AND lines (rust + tuple). The section
	// whose title contains a search term ("Tuples") must rank above the one
	// whose title doesn't ("Pointers").
	fs := fstest.MapFS{
		"languages/rust.md": &fstest.MapFile{
			Data: []byte(`# Rust

## Pointers

In Rust, tuples are not pointers.
let p = &x;

## Tuples

A tuple in Rust is heterogeneous.
let t = (1, 2, 3);
`),
		},
	}
	reg, err := New(fs)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	matches := reg.Search("rust", "tuple")
	if len(matches) == 0 {
		t.Fatal("Search(rust, tuple) = no results")
	}
	if matches[0].Section != "Tuples" {
		t.Errorf("first match section = %q, want \"Tuples\"", matches[0].Section)
	}
}

func TestSearchPrefersDeeperContentOnNameCollision(t *testing.T) {
	// Two sheets share the same name across different categories. With
	// identical name+token scores, the longer/more-comprehensive sheet
	// must win. This guards against the patterns/distributed-systems vs
	// cs-theory/distributed-systems collision.
	short := strings.Repeat("Distributed systems patterns overview.\n", 5)
	long := strings.Repeat("Distributed systems theory deep dive with proofs.\n", 100)
	fs := fstest.MapFS{
		"patterns/distributed-systems.md": &fstest.MapFile{
			Data: []byte("# Distributed Systems Patterns\n\n## Overview\n\n" + short),
		},
		"cs-theory/distributed-systems.md": &fstest.MapFile{
			Data: []byte("# Distributed Systems Theory\n\n## Foundations\n\n" + long),
		},
	}
	reg, err := New(fs)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	matches := reg.Search("distributed-systems")
	if len(matches) == 0 {
		t.Fatal("Search(distributed-systems) = no results")
	}
	if matches[0].Sheet.Category != "cs-theory" {
		t.Errorf("first match category = %q, want %q (deeper content should win on token-tie)",
			matches[0].Sheet.Category, "cs-theory")
	}
}

func TestSearchTermsCap(t *testing.T) {
	reg, _ := New(testFS())

	// 16 terms that all appear in the lvm fixture content.
	base := []string{
		"lvm", "lvextend", "myvg", "myvolume", "lvcreate",
		"extend", "create", "physical", "logical", "volume",
		"manager", "storage", "virtualization", "groups", "vgs", "vgcreate",
	}
	if len(base) != 16 {
		t.Fatalf("test setup: base has %d terms, want 16", len(base))
	}

	capped := reg.Search(base...)
	if len(capped) == 0 {
		t.Fatal("16-term lvm search returned 0 matches; fixture changed?")
	}

	// Add a 17th term that does NOT appear in any fixture sheet. Without
	// the cap the AND would zero out; with the cap the 17th is dropped and
	// the result equals the 16-term search.
	overflow := reg.Search(append(append([]string{}, base...), "kubernetesxyz")...)
	if len(overflow) != len(capped) {
		t.Errorf("term cap should drop 17th term: capped=%d overflow=%d", len(capped), len(overflow))
	}
}

func TestFindSection(t *testing.T) {
	reg, _ := New(testFS())

	content, err := reg.FindSection("lvm", "extend")
	if err != nil {
		t.Fatalf("FindSection: %v", err)
	}
	if !strings.Contains(content, "lvextend") {
		t.Errorf("section content missing lvextend:\n%s", content)
	}
	if !strings.Contains(content, "LVM") {
		t.Error("section content missing title")
	}
}

func TestFindSectionNotFound(t *testing.T) {
	reg, _ := New(testFS())

	_, err := reg.FindSection("lvm", "nonexistent")
	if err == nil {
		t.Error("FindSection(nonexistent) = nil error")
	}
}

func TestFindSectionUnknownTopic(t *testing.T) {
	reg, _ := New(testFS())

	_, err := reg.FindSection("unknown", "extend")
	if err == nil {
		t.Error("FindSection(unknown) = nil error")
	}
}

func TestSections(t *testing.T) {
	reg, _ := New(testFS())
	s := reg.Get("lvm")
	if s == nil {
		t.Fatal("Get(lvm) = nil")
	}

	// Should have: Physical Volumes, Volume Groups, Logical Volumes, Create, Extend
	if len(s.Sections) < 4 {
		t.Errorf("sections = %d, want >= 4", len(s.Sections))
		for _, sec := range s.Sections {
			t.Logf("  %s (level %d)", sec.Title, sec.Level)
		}
	}

	// Check that Extend is level 3
	found := false
	for _, sec := range s.Sections {
		if sec.Title == "Extend" {
			found = true
			if sec.Level != 3 {
				t.Errorf("Extend level = %d, want 3", sec.Level)
			}
		}
	}
	if !found {
		t.Error("no Extend section found")
	}
}

func TestOverlay(t *testing.T) {
	base := fstest.MapFS{
		"shell/bash.md": &fstest.MapFile{
			Data: []byte("# Bash\n\nOriginal.\n"),
		},
	}
	override := fstest.MapFS{
		"shell/bash.md": &fstest.MapFile{
			Data: []byte("# Bash\n\nCustom version.\n"),
		},
	}

	reg, err := New(base, override)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	s := reg.Get("bash")
	if s == nil {
		t.Fatal("Get(bash) = nil")
	}
	if !strings.Contains(s.Description, "Custom version") {
		t.Errorf("expected custom override, got: %q", s.Description)
	}
}

func TestDetailCount_FileSystemAccurate(t *testing.T) {
	// Two detail files share the same name across categories. The lookup
	// map deduplicates by name (last-write-wins), but DetailCount() should
	// reflect the file-system count (both files contributed) so that
	// /api/health and `cs --count` match `find detail -name '*.md' | wc -l`.
	sheetFS := fstest.MapFS{
		"storage/lvm.md": &fstest.MapFile{Data: []byte("# LVM\n")},
	}

	// The detail walker trims a "detail/" prefix from paths. Provide the
	// trimmed-form so the walker maps storage/topic.md → category=storage.
	detailFS := fstest.MapFS{
		"storage/topic.md":    &fstest.MapFile{Data: []byte("# topic (storage)\n")},
		"networking/topic.md": &fstest.MapFile{Data: []byte("# topic (networking)\n")},
		"databases/redis.md":  &fstest.MapFile{Data: []byte("# Redis\n")},
	}

	reg, err := NewWithDetails([]fs.FS{sheetFS}, []fs.FS{detailFS})
	if err != nil {
		t.Fatalf("NewWithDetails: %v", err)
	}

	// File-system count: 3 detail files exist on the synthetic FS.
	if got := reg.DetailCount(); got != 3 {
		t.Errorf("DetailCount() = %d, want 3 (file-system count, not deduped)", got)
	}
	// Unique-name count: only 2 distinct topic names because storage/topic
	// and networking/topic collide.
	if got := reg.DetailUniqueCount(); got != 2 {
		t.Errorf("DetailUniqueCount() = %d, want 2 (deduped by name)", got)
	}
	// HasDetail still works by-name (one of the two same-named files won
	// the map slot — both lookups should hit it).
	if !reg.HasDetail("topic") {
		t.Error("HasDetail(topic) should be true")
	}
	if !reg.HasDetail("redis") {
		t.Error("HasDetail(redis) should be true")
	}
}

func TestDetailCount_NoCollisions(t *testing.T) {
	// Sanity: when there are no name collisions, DetailCount and
	// DetailUniqueCount return the same value.
	sheetFS := fstest.MapFS{
		"storage/lvm.md": &fstest.MapFile{Data: []byte("# LVM\n")},
	}
	detailFS := fstest.MapFS{
		"storage/lvm.md":         &fstest.MapFile{Data: []byte("# LVM\n")},
		"networking/bgp.md":      &fstest.MapFile{Data: []byte("# BGP\n")},
		"databases/postgresql.md": &fstest.MapFile{Data: []byte("# Postgres\n")},
	}
	reg, err := NewWithDetails([]fs.FS{sheetFS}, []fs.FS{detailFS})
	if err != nil {
		t.Fatalf("NewWithDetails: %v", err)
	}
	if reg.DetailCount() != 3 {
		t.Errorf("DetailCount() = %d, want 3", reg.DetailCount())
	}
	if reg.DetailUniqueCount() != 3 {
		t.Errorf("DetailUniqueCount() = %d, want 3", reg.DetailUniqueCount())
	}
	if reg.DetailCount() != reg.DetailUniqueCount() {
		t.Error("with no collisions, DetailCount and DetailUniqueCount should match")
	}
}
