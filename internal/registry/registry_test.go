package registry

import (
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
