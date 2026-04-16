package cscore

import "sort"

type learnPathEntry struct {
	Order         int      `json:"order"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	HasDetail     bool     `json:"has_detail"`
	Prerequisites []string `json:"prerequisites,omitempty"`
	PrereqCount   int      `json:"prereq_count"`
}

type learnPathResponse struct {
	Category string           `json:"category"`
	Path     []learnPathEntry `json:"path"`
}

// LearnPathJSON returns topics in a category ordered by prerequisite count.
func LearnPathJSON(category string) string {
	if err := validateCategory(category); err != nil {
		return errorJSON(err)
	}
	r := mustReg()

	if !r.IsCategory(category) {
		return errorJSON(&validationError{"category", "unknown category"})
	}

	sheets := r.ByCategory(category)
	if len(sheets) == 0 {
		return errorJSON(&validationError{"category", "no sheets in category"})
	}

	type entry struct {
		name        string
		description string
		hasDetail   bool
		prereqs     []string
		prereqCount int
	}
	entries := make([]entry, 0, len(sheets))
	for _, s := range sheets {
		d := r.GetDetail(s.Name)
		var prereqs []string
		if d != nil {
			prereqs = d.Prerequisites
		}
		entries = append(entries, entry{
			name:        s.Name,
			description: s.Description,
			hasDetail:   r.HasDetail(s.Name),
			prereqs:     prereqs,
			prereqCount: len(prereqs),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].prereqCount != entries[j].prereqCount {
			return entries[i].prereqCount < entries[j].prereqCount
		}
		return entries[i].name < entries[j].name
	})

	path := make([]learnPathEntry, 0, len(entries))
	for i, e := range entries {
		path = append(path, learnPathEntry{
			Order:         i + 1,
			Name:          e.name,
			Description:   e.description,
			HasDetail:     e.hasDetail,
			Prerequisites: e.prereqs,
			PrereqCount:   e.prereqCount,
		})
	}

	return jsonMarshal(learnPathResponse{
		Category: category,
		Path:     path,
	})
}
