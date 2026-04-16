package cscore

import (
	"sort"
	"strings"

	"github.com/bellistech/cs/internal/bookmarks"
)

type statsResponse struct {
	TotalSheets    int               `json:"total_sheets"`
	DetailPages    int               `json:"detail_pages"`
	Categories     int               `json:"categories"`
	SeeAlsoCoverage int             `json:"see_also_coverage"`
	Bookmarks      int               `json:"bookmarks"`
	TotalLines     int               `json:"total_lines"`
	PerCategory    []categorySummary `json:"per_category"`
}

// StatsJSON returns overall statistics as JSON.
func StatsJSON() string {
	r := mustReg()
	sheets := r.List()
	cats := r.Categories()

	totalLines := 0
	for _, s := range sheets {
		totalLines += strings.Count(s.Content, "\n")
	}

	counts := make([]categorySummary, 0, len(cats))
	for _, cat := range cats {
		counts = append(counts, categorySummary{
			Name:  cat,
			Count: len(r.ByCategory(cat)),
		})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	return jsonMarshal(statsResponse{
		TotalSheets:    len(sheets),
		DetailPages:    r.DetailCount(),
		Categories:     len(cats),
		SeeAlsoCoverage: r.SeeAlsoCoverage(),
		Bookmarks:      len(bookmarks.List()),
		TotalLines:     totalLines,
		PerCategory:    counts,
	})
}
