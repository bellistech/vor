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

// Search does case-insensitive substring search across all sheets.
func (r *Registry) Search(query string) []Match {
	q := strings.ToLower(query)
	var matches []Match

	for _, s := range r.all {
		lines := strings.Split(s.Content, "\n")
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), q) {
				section := findSectionForLine(s, line)
				matches = append(matches, Match{
					Sheet:   s,
					Section: section,
					Line:    strings.TrimSpace(line),
				})
			}
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
