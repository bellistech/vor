package cscore

import (
	"strings"

	"github.com/bellistech/vor/internal/registry"
)

const maxSearchResults = 100

type searchResponse struct {
	Query   string         `json:"query"`
	Results []searchResult `json:"results"`
	Count   int            `json:"count"`
}

type searchResult struct {
	Topic    string `json:"topic"`
	Category string `json:"category"`
	Section  string `json:"section,omitempty"`
	Line     string `json:"line,omitempty"`
}

// SearchJSON searches all sheets and returns results as JSON.
// Empty query returns all topic names.
func SearchJSON(query string) string {
	if err := validateQuery(query); err != nil {
		return errorJSON(err)
	}
	r := mustReg()

	if query == "" {
		// Return all topics as results
		sheets := r.List()
		results := make([]searchResult, 0, len(sheets))
		for _, s := range sheets {
			results = append(results, searchResult{
				Topic:    s.Name,
				Category: s.Category,
			})
		}
		return jsonMarshal(searchResponse{
			Query:   "",
			Results: results,
			Count:   len(results),
		})
	}

	matches := r.Search(query)
	results := make([]searchResult, 0, len(matches))
	seen := make(map[string]bool)
	for _, m := range matches {
		key := m.Sheet.Name + "|" + m.Section
		if seen[key] {
			continue
		}
		seen[key] = true
		results = append(results, searchResult{
			Topic:    m.Sheet.Name,
			Category: m.Sheet.Category,
			Section:  m.Section,
			Line:     truncate(m.Line, 120),
		})
		if len(results) >= maxSearchResults {
			break
		}
	}

	return jsonMarshal(searchResponse{
		Query:   query,
		Results: results,
		Count:   len(results),
	})
}

// fuzzyFind attempts prefix, substring, then Levenshtein matching.
func fuzzyFind(name string) *registry.Sheet {
	r := mustReg()
	lower := strings.ToLower(name)

	for _, sheet := range r.List() {
		if strings.HasPrefix(sheet.Name, lower) {
			return sheet
		}
	}
	for _, sheet := range r.List() {
		if strings.Contains(sheet.Name, lower) {
			return sheet
		}
	}

	var best *registry.Sheet
	bestDist := len(name) + 1
	for _, sheet := range r.List() {
		d := levenshtein(lower, sheet.Name)
		if len(sheet.Name) > len(lower) {
			dp := levenshtein(lower, sheet.Name[:len(lower)])
			if dp < d {
				d = dp
			}
		}
		if d < bestDist && d <= len(name)/2+1 {
			bestDist = d
			best = sheet
		}
	}
	return best
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := curr[j-1] + 1
			ins := prev[j] + 1
			sub := prev[j-1] + cost
			curr[j] = del
			if ins < curr[j] {
				curr[j] = ins
			}
			if sub < curr[j] {
				curr[j] = sub
			}
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
