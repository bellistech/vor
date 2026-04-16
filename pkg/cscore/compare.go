package cscore

import (
	"sort"
	"strings"

	"github.com/bellistech/cs/internal/registry"
)

type compareSide struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Sections    int      `json:"sections"`
	Lines       int      `json:"lines"`
	HasDetail   bool     `json:"has_detail"`
	SeeAlso     []string `json:"see_also,omitempty"`
}

type compareResponse struct {
	A           compareSide `json:"a"`
	B           compareSide `json:"b"`
	AllSections []string    `json:"all_sections"`
	SectionsA   []string    `json:"sections_a"`
	SectionsB   []string    `json:"sections_b"`
}

// CompareJSON returns a structured comparison of two topics as JSON.
func CompareJSON(nameA, nameB string) string {
	if err := validateTopic(nameA); err != nil {
		return errorJSON(err)
	}
	if err := validateTopic(nameB); err != nil {
		return errorJSON(err)
	}
	r := mustReg()

	a := r.Get(nameA)
	if a == nil {
		a = fuzzyFind(nameA)
	}
	b := r.Get(nameB)
	if b == nil {
		b = fuzzyFind(nameB)
	}
	if a == nil {
		return errorJSON(&validationError{"topic", "topic A not found: " + nameA})
	}
	if b == nil {
		return errorJSON(&validationError{"topic", "topic B not found: " + nameB})
	}

	secA := sectionNames(a)
	secB := sectionNames(b)
	allSec := mergeKeys(secA, secB)

	return jsonMarshal(compareResponse{
		A:           toCompareSide(r, a),
		B:           toCompareSide(r, b),
		AllSections: allSec,
		SectionsA:   mapKeys(secA),
		SectionsB:   mapKeys(secB),
	})
}

func toCompareSide(r *registry.Registry, s *registry.Sheet) compareSide {
	return compareSide{
		Name:        s.Name,
		Category:    s.Category,
		Description: s.Description,
		Sections:    len(s.Sections),
		Lines:       strings.Count(s.Content, "\n"),
		HasDetail:   r.HasDetail(s.Name),
		SeeAlso:     s.SeeAlso,
	}
}

func sectionNames(s *registry.Sheet) map[string]bool {
	m := make(map[string]bool)
	for _, sec := range s.Sections {
		if sec.Level == 2 {
			m[sec.Title] = true
		}
	}
	return m
}

func mergeKeys(a, b map[string]bool) []string {
	all := make(map[string]bool)
	for k := range a {
		all[k] = true
	}
	for k := range b {
		all[k] = true
	}
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
