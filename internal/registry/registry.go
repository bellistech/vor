package registry

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Registry holds all loaded cheatsheets.
type Registry struct {
	sheets     map[string]*Sheet
	details    map[string]*Sheet
	byCategory map[string][]*Sheet
	all        []*Sheet
	categories []string
}

// New creates a Registry from one or more fs.FS sources.
// Later sources override earlier ones (custom overrides embedded).
// NewWithDetails creates a Registry with both sheets and detail sources.
func NewWithDetails(sheetSources []fs.FS, detailSources []fs.FS) (*Registry, error) {
	r, err := New(sheetSources...)
	if err != nil {
		return nil, err
	}
	for _, fsys := range detailSources {
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			data, readErr := fs.ReadFile(fsys, path)
			if readErr != nil {
				return fmt.Errorf("read %s: %w", path, readErr)
			}
			cleaned := strings.TrimPrefix(path, "detail/")
			category := filepath.Dir(cleaned)
			name := strings.TrimSuffix(filepath.Base(cleaned), ".md")
			sheet := ParseSheet(name, category, string(data))
			r.details[name] = sheet
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

// GetDetail returns a detail sheet by name, or nil.
func (r *Registry) GetDetail(name string) *Sheet {
	return r.details[strings.ToLower(name)]
}

// HasDetail returns true if a detail exists for the named topic.
func (r *Registry) HasDetail(name string) bool {
	return r.details[strings.ToLower(name)] != nil
}

// DetailCount returns the number of loaded detail sheets.
func (r *Registry) DetailCount() int {
	return len(r.details)
}

func New(sources ...fs.FS) (*Registry, error) {
	r := &Registry{
		sheets:     make(map[string]*Sheet),
		details:    make(map[string]*Sheet),
		byCategory: make(map[string][]*Sheet),
	}

	for _, fsys := range sources {
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}

			data, err := fs.ReadFile(fsys, path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}

			// path is like "shell/bash.md" or "sheets/shell/bash.md"
			cleaned := strings.TrimPrefix(path, "sheets/")
			category := filepath.Dir(cleaned)
			name := strings.TrimSuffix(filepath.Base(cleaned), ".md")

			sheet := ParseSheet(name, category, string(data))
			r.sheets[name] = sheet
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Build category index and sorted lists
	catSet := make(map[string]bool)
	for _, s := range r.sheets {
		r.all = append(r.all, s)
		r.byCategory[s.Category] = append(r.byCategory[s.Category], s)
		catSet[s.Category] = true
	}

	sort.Slice(r.all, func(i, j int) bool {
		return r.all[i].Name < r.all[j].Name
	})

	for cat, sheets := range r.byCategory {
		sort.Slice(sheets, func(i, j int) bool {
			return sheets[i].Name < sheets[j].Name
		})
		r.byCategory[cat] = sheets
	}

	for cat := range catSet {
		r.categories = append(r.categories, cat)
	}
	sort.Strings(r.categories)

	return r, nil
}

// Get returns a sheet by name, or nil.
func (r *Registry) Get(name string) *Sheet {
	return r.sheets[strings.ToLower(name)]
}

// List returns all sheets sorted by name.
func (r *Registry) List() []*Sheet {
	return r.all
}

// Categories returns sorted category names.
func (r *Registry) Categories() []string {
	return r.categories
}

// IsCategory returns true if the name matches a category.
func (r *Registry) IsCategory(name string) bool {
	for _, c := range r.categories {
		if c == name {
			return true
		}
	}
	return false
}

// ByCategory returns sheets in a category.
func (r *Registry) ByCategory(cat string) []*Sheet {
	return r.byCategory[cat]
}

// maxSearchTerms caps distinct query terms to bound per-call work. Terms
// beyond the cap are silently dropped — defense against an unauthenticated
// REST caller crafting many short tokens to amplify CPU/memory cost.
const maxSearchTerms = 16

// tokenize splits a hyphenated identifier into its parts, dropping empties.
func tokenize(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "-")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Search does case-insensitive AND-of-terms search across all sheets.
// Each input is split on whitespace. A sheet matches when all terms appear
// somewhere in its category/name/content. Matching sheets are ranked by
// (1) terms that exactly match a name/category token, then (2) total terms
// found across all section titles in the sheet, then (3) shorter names
// (more of the name is captured by the query), then (4) lines containing
// all terms (strict matches), then (5) sheet name. Within each sheet,
// matches from sections whose title contains a term float to the top, and
// strict-AND lines are returned when any exist; otherwise lines containing
// any term are returned as a fallback.
func (r *Registry) Search(queries ...string) []Match {
	var terms []string
	seen := make(map[string]bool)
collect:
	for _, q := range queries {
		for _, w := range strings.Fields(strings.ToLower(q)) {
			if seen[w] {
				continue
			}
			seen[w] = true
			terms = append(terms, w)
			if len(terms) == maxSearchTerms {
				break collect
			}
		}
	}
	if len(terms) == 0 {
		return nil
	}

	containsAll := func(s string, ts []string) bool {
		for _, t := range ts {
			if !strings.Contains(s, t) {
				return false
			}
		}
		return true
	}
	containsAny := func(s string, ts []string) bool {
		for _, t := range ts {
			if strings.Contains(s, t) {
				return true
			}
		}
		return false
	}

	type sheetHit struct {
		sheet        *Sheet
		wholeMatches int // terms that exactly match a name/category token
		titleMatches int // sum of search-term hits across section titles
		nameTokens   int // number of tokens in the sheet name (smaller = more specific)
		strict       []Match
		loose        []Match
	}

	var hits []sheetHit
	for _, s := range r.all {
		if !containsAll(s.lower, terms) {
			continue
		}
		nameToks := tokenize(strings.ToLower(s.Name))
		catToks := tokenize(strings.ToLower(s.Category))
		tokenSet := make(map[string]bool, len(nameToks)+len(catToks))
		for _, t := range nameToks {
			tokenSet[t] = true
		}
		for _, t := range catToks {
			tokenSet[t] = true
		}
		whole := 0
		isAnchor := make(map[string]bool, len(terms))
		for _, t := range terms {
			if tokenSet[t] {
				whole++
				isAnchor[t] = true
			}
		}

		// Title-hit scoring uses non-anchor (filter) terms when any exist —
		// otherwise the sheet name leaks into every section that mentions
		// the topic (e.g. "rust tuple" should privilege titles with "tuple",
		// not titles with "Rust" since the whole sheet is about Rust).
		titleHitTerms := make([]string, 0, len(terms))
		for _, t := range terms {
			if !isAnchor[t] {
				titleHitTerms = append(titleHitTerms, t)
			}
		}
		if len(titleHitTerms) == 0 {
			titleHitTerms = terms
		}

		sectionTitleHits := make(map[string]int, len(s.Sections))
		titleHitTotal := 0
		for _, sec := range s.Sections {
			titleLower := strings.ToLower(sec.Title)
			h := 0
			for _, t := range titleHitTerms {
				if strings.Contains(titleLower, t) {
					h++
				}
			}
			if h > 0 {
				sectionTitleHits[sec.Title] = h
				titleHitTotal += h
			}
		}

		var strict, loose []Match
		for _, line := range strings.Split(s.Content, "\n") {
			lower := strings.ToLower(line)
			switch {
			case containsAll(lower, terms):
				strict = append(strict, Match{
					Sheet:   s,
					Section: findSectionForLine(s, line),
					Line:    strings.TrimSpace(line),
				})
			case containsAny(lower, terms):
				loose = append(loose, Match{
					Sheet:   s,
					Section: findSectionForLine(s, line),
					Line:    strings.TrimSpace(line),
				})
			}
		}

		// Within each tier, float matches from title-hit sections to the top.
		titleScore := func(m Match) int { return sectionTitleHits[m.Section] }
		sort.SliceStable(strict, func(i, j int) bool {
			return titleScore(strict[i]) > titleScore(strict[j])
		})
		sort.SliceStable(loose, func(i, j int) bool {
			return titleScore(loose[i]) > titleScore(loose[j])
		})

		hits = append(hits, sheetHit{
			sheet:        s,
			wholeMatches: whole,
			titleMatches: titleHitTotal,
			nameTokens:   len(nameToks),
			strict:       strict,
			loose:        loose,
		})
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].wholeMatches != hits[j].wholeMatches {
			return hits[i].wholeMatches > hits[j].wholeMatches
		}
		if hits[i].titleMatches != hits[j].titleMatches {
			return hits[i].titleMatches > hits[j].titleMatches
		}
		if hits[i].nameTokens != hits[j].nameTokens {
			return hits[i].nameTokens < hits[j].nameTokens
		}
		if len(hits[i].strict) != len(hits[j].strict) {
			return len(hits[i].strict) > len(hits[j].strict)
		}
		return hits[i].sheet.Name < hits[j].sheet.Name
	})

	var matches []Match
	for _, h := range hits {
		if len(h.strict) > 0 {
			matches = append(matches, h.strict...)
		} else {
			matches = append(matches, h.loose...)
		}
	}
	return matches
}

// FindSection returns the content of matching sections within a sheet.
func (r *Registry) FindSection(name, section string) (string, error) {
	s := r.Get(name)
	if s == nil {
		return "", fmt.Errorf("unknown topic: %s", name)
	}

	q := strings.ToLower(section)
	var parts []string
	for _, sec := range s.Sections {
		if strings.Contains(strings.ToLower(sec.Title), q) {
			header := strings.Repeat("#", sec.Level) + " " + sec.Title
			parts = append(parts, header+"\n\n"+sec.Content)
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("no section matching %q in %s", section, name)
	}

	// Include the sheet title at the top
	result := "# " + s.Title + "\n\n" + strings.Join(parts, "\n\n")
	return result, nil
}

// Related returns sheets referenced in a topic's SeeAlso field.
func (r *Registry) Related(name string) []*Sheet {
	s := r.Get(name)
	if s == nil {
		return nil
	}
	var related []*Sheet
	seen := make(map[string]bool)
	for _, ref := range s.SeeAlso {
		if seen[ref] {
			continue
		}
		seen[ref] = true
		if rs := r.Get(ref); rs != nil {
			related = append(related, rs)
		}
	}
	return related
}

// AllNames returns all sheet names sorted.
func (r *Registry) AllNames() []string {
	names := make([]string, 0, len(r.all))
	for _, s := range r.all {
		names = append(names, s.Name)
	}
	return names
}

// SeeAlsoCoverage returns count of sheets that have See Also sections.
func (r *Registry) SeeAlsoCoverage() int {
	count := 0
	for _, s := range r.all {
		if len(s.SeeAlso) > 0 {
			count++
		}
	}
	return count
}

func findSectionForLine(s *Sheet, line string) string {
	trimmed := strings.TrimSpace(line)
	for i := len(s.Sections) - 1; i >= 0; i-- {
		sec := s.Sections[i]
		if strings.Contains(sec.Content, trimmed) {
			return sec.Title
		}
	}
	return ""
}
