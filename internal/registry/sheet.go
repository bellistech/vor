package registry

import (
	"strings"
)

// SourceKind labels how a sheet entered the registry. Consumers — agent
// runtimes, tool-call gates — use this to compute trust on retrieval
// results. Embedded canonical sheets are the most trusted; user-symlinked
// external sources are the least.
type SourceKind int

const (
	// SourceEmbedded — compiled into the cs binary via go:embed. Default
	// for back-compat with callers that don't supply provenance.
	SourceEmbedded SourceKind = iota

	// SourceUserCustom — sheet authored by the user under
	// ~/.config/cs/sheets/<category>/<topic>.md.
	SourceUserCustom

	// SourceUserSource — sheet from a directory the user symlinked into
	// ~/.config/cs/sources/<name>/. The most-flexible, least-trusted
	// retrieval path. See Phase A `internal/sources` and
	// unheaded:eval/coding-gate/probe-2026-05-02/A1-source-poison.md
	// for the threat model.
	SourceUserSource
)

// String renders SourceKind in the public JSON form used by /api/* responses.
func (k SourceKind) String() string {
	switch k {
	case SourceEmbedded:
		return "embedded"
	case SourceUserCustom:
		return "user-custom"
	case SourceUserSource:
		return "user-source"
	default:
		return "unknown"
	}
}

// Trust returns the consumer-facing trust label for the kind. Used by
// agent runtimes that need to decide whether retrieval content is
// authoritative or attacker-controllable.
func (k SourceKind) Trust() string {
	switch k {
	case SourceEmbedded:
		return "canonical"
	case SourceUserCustom:
		return "local"
	case SourceUserSource:
		return "external"
	default:
		return "unknown"
	}
}

// Sheet represents a parsed cheatsheet.
type Sheet struct {
	Name          string
	Category      string
	Title         string
	Description   string
	Content       string
	Sections      []Section
	SeeAlso       []string // parsed from ## See Also section
	Prerequisites []string // parsed from ## Prerequisites section (detail pages)
	Complexity    string   // parsed from ## Complexity section (detail pages)

	// Source provenance — populated by the registry constructor based on
	// the SourceSpec the sheet was loaded from. Embedded sheets default
	// to SourceEmbedded with empty Path/Label.
	SourceKind  SourceKind // how this sheet entered the registry
	SourcePath  string     // resolved filesystem path for non-embedded sheets
	SourceLabel string     // user-facing label (symlink name for SourceUserSource)

	lower        string // lowercased "category name content" — search haystack
	lowerNameCat string // lowercased "category name" — used for ranking signal
}

// Section represents an H2 or H3 section within a sheet.
type Section struct {
	Title   string
	Level   int
	Content string
}

// Match represents a search result.
type Match struct {
	Sheet   *Sheet
	Section string
	Line    string
}

// ParseSheet parses raw markdown into a Sheet.
func ParseSheet(name, category, raw string) *Sheet {
	s := &Sheet{
		Name:     name,
		Category: category,
		Content:  raw,
	}

	lines := strings.Split(raw, "\n")
	var descLines []string
	foundTitle := false
	foundFirstSection := false

	for _, line := range lines {
		if !foundTitle && strings.HasPrefix(line, "# ") {
			s.Title = strings.TrimPrefix(line, "# ")
			foundTitle = true
			continue
		}
		if foundTitle && !foundFirstSection {
			if strings.HasPrefix(line, "## ") {
				foundFirstSection = true
			} else if strings.TrimSpace(line) != "" {
				descLines = append(descLines, strings.TrimSpace(line))
			}
		}
		if foundFirstSection {
			break
		}
	}
	s.Description = strings.Join(descLines, " ")

	s.Sections = parseSections(raw)
	s.SeeAlso = parseSeeAlso(s.Sections)
	s.Prerequisites = parseListSection(s.Sections, "Prerequisites")
	s.Complexity = parseRawSection(s.Sections, "Complexity")

	s.lowerNameCat = strings.ToLower(category + " " + name)
	s.lower = s.lowerNameCat + " " + strings.ToLower(raw)
	return s
}

// parseSeeAlso extracts topic names from a "See Also" section.
func parseSeeAlso(sections []Section) []string {
	for _, sec := range sections {
		if strings.EqualFold(sec.Title, "See Also") {
			return parseTopicList(sec.Content)
		}
	}
	return nil
}

// parseListSection extracts a bulleted list from a named section.
func parseListSection(sections []Section, title string) []string {
	for _, sec := range sections {
		if strings.EqualFold(sec.Title, title) {
			return parseTopicList(sec.Content)
		}
	}
	return nil
}

// parseRawSection returns the raw content of a named section.
func parseRawSection(sections []Section, title string) string {
	for _, sec := range sections {
		if strings.EqualFold(sec.Title, title) {
			return strings.TrimSpace(sec.Content)
		}
	}
	return ""
}

// parseTopicList parses lines like "- topic1, topic2" or "- topic1\n- topic2"
func parseTopicList(content string) []string {
	var topics []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		if line == "" {
			continue
		}
		// Handle comma-separated lists
		for _, item := range strings.Split(line, ",") {
			item = strings.TrimSpace(item)
			// Strip markdown links: [text](url) → text
			if idx := strings.Index(item, "]("); idx != -1 {
				item = strings.TrimPrefix(item[:idx], "[")
			}
			item = strings.ToLower(item)
			if item != "" {
				topics = append(topics, item)
			}
		}
	}
	return topics
}

func parseSections(raw string) []Section {
	lines := strings.Split(raw, "\n")
	var sections []Section
	var current *Section
	var contentLines []string

	flush := func() {
		if current != nil {
			current.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
			sections = append(sections, *current)
			contentLines = nil
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "### ") {
			flush()
			current = &Section{
				Title: strings.TrimPrefix(line, "### "),
				Level: 3,
			}
			contentLines = nil
		} else if strings.HasPrefix(line, "## ") {
			flush()
			current = &Section{
				Title: strings.TrimPrefix(line, "## "),
				Level: 2,
			}
			contentLines = nil
		} else if current != nil {
			contentLines = append(contentLines, line)
		}
	}
	flush()
	return sections
}
