package registry

import (
	"strings"
)

// Sheet represents a parsed cheatsheet.
type Sheet struct {
	Name        string
	Category    string
	Title       string
	Description string
	Content     string
	Sections    []Section
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
	return s
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
