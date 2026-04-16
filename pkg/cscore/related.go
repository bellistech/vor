package cscore

type relatedResponse struct {
	Topic   string         `json:"topic"`
	Related []topicSummary `json:"related"`
}

// RelatedJSON returns the SeeAlso topics for a given topic as JSON.
func RelatedJSON(name string) string {
	if err := validateTopic(name); err != nil {
		return errorJSON(err)
	}
	r := mustReg()
	s := r.Get(name)
	if s == nil {
		return errorJSON(&validationError{"topic", "not found"})
	}
	related := r.Related(name)
	topics := make([]topicSummary, 0, len(related))
	for _, rel := range related {
		topics = append(topics, topicSummary{
			Name:        rel.Name,
			Category:    rel.Category,
			Title:       rel.Title,
			Description: rel.Description,
			HasDetail:   r.HasDetail(rel.Name),
			SeeAlso:     rel.SeeAlso,
		})
	}
	return jsonMarshal(relatedResponse{
		Topic:   s.Name,
		Related: topics,
	})
}
