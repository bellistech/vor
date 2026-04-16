package cscore

type categorySummary struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type categoryTopicsResponse struct {
	Category string         `json:"category"`
	Topics   []topicSummary `json:"topics"`
}

// CategoriesJSON returns all categories with counts as JSON.
func CategoriesJSON() string {
	r := mustReg()
	cats := r.Categories()
	result := make([]categorySummary, 0, len(cats))
	for _, cat := range cats {
		result = append(result, categorySummary{
			Name:  cat,
			Count: len(r.ByCategory(cat)),
		})
	}
	return jsonMarshal(result)
}

// CategoryTopicsJSON returns topics in a category as JSON.
func CategoryTopicsJSON(category string) string {
	if err := validateCategory(category); err != nil {
		return errorJSON(err)
	}
	r := mustReg()
	sheets := r.ByCategory(category)
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
	return jsonMarshal(categoryTopicsResponse{
		Category: category,
		Topics:   topics,
	})
}
