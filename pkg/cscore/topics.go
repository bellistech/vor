package cscore

import (
	"math/rand/v2"

	"github.com/bellistech/cs/internal/registry"
)

type topicSummary struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	HasDetail   bool     `json:"has_detail"`
	SeeAlso     []string `json:"see_also,omitempty"`
}

type sheetResponse struct {
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Content     string            `json:"content"`
	SeeAlso     []string          `json:"see_also,omitempty"`
	Sections    []sectionResponse `json:"sections"`
	HasDetail   bool              `json:"has_detail"`
}

type sectionResponse struct {
	Title   string `json:"title"`
	Level   int    `json:"level"`
	Content string `json:"content"`
}

type detailResponse struct {
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Prerequisites []string `json:"prerequisites,omitempty"`
	Complexity    string   `json:"complexity,omitempty"`
}

// ListTopicsJSON returns all topics as a JSON array sorted by name.
func ListTopicsJSON() string {
	r := mustReg()
	sheets := r.List()
	topics := make([]topicSummary, 0, len(sheets))
	for _, s := range sheets {
		topics = append(topics, topicSummary{
			Name:        s.Name,
			Category:    s.Category,
			Title:       s.Title,
			Description: s.Description,
			HasDetail:   r.HasDetail(s.Name),
			SeeAlso:     s.SeeAlso,
		})
	}
	return jsonMarshal(topics)
}

// GetSheetJSON returns a single sheet as JSON.
func GetSheetJSON(name string) string {
	if err := validateTopic(name); err != nil {
		return errorJSON(err)
	}
	r := mustReg()
	s := r.Get(name)
	if s == nil {
		s = fuzzyFind(name)
	}
	if s == nil {
		return errorJSON(&validationError{"topic", "not found"})
	}
	return jsonMarshal(sheetToResponse(r, s))
}

// GetDetailJSON returns a detail page as JSON.
func GetDetailJSON(name string) string {
	if err := validateTopic(name); err != nil {
		return errorJSON(err)
	}
	r := mustReg()
	d := r.GetDetail(name)
	if d == nil {
		return errorJSON(&validationError{"topic", "no detail page"})
	}
	return jsonMarshal(detailResponse{
		Name:          d.Name,
		Category:      d.Category,
		Title:         d.Title,
		Content:       d.Content,
		Prerequisites: d.Prerequisites,
		Complexity:    d.Complexity,
	})
}

// RandomTopicJSON returns a random sheet as JSON.
func RandomTopicJSON() string {
	r := mustReg()
	sheets := r.List()
	if len(sheets) == 0 {
		return errorJSON(&validationError{"topic", "no sheets loaded"})
	}
	s := sheets[rand.IntN(len(sheets))]
	return jsonMarshal(sheetToResponse(r, s))
}

func sheetToResponse(r *registry.Registry, s *registry.Sheet) sheetResponse {
	sections := make([]sectionResponse, 0, len(s.Sections))
	for _, sec := range s.Sections {
		sections = append(sections, sectionResponse{
			Title:   sec.Title,
			Level:   sec.Level,
			Content: sec.Content,
		})
	}
	return sheetResponse{
		Name:        s.Name,
		Category:    s.Category,
		Title:       s.Title,
		Description: s.Description,
		Content:     s.Content,
		SeeAlso:     s.SeeAlso,
		Sections:    sections,
		HasDetail:   r.HasDetail(s.Name),
	}
}
