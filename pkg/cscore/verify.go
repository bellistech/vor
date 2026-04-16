package cscore

import (
	"github.com/bellistech/cs/internal/verify"
)

type verifyResponse struct {
	Topic   string         `json:"topic"`
	Results []verifyResult `json:"results"`
	Pass    int            `json:"pass"`
	Fail    int            `json:"fail"`
	Total   int            `json:"total"`
}

type verifyResult struct {
	Expression string  `json:"expression"`
	Expected   float64 `json:"expected"`
	Got        float64 `json:"got"`
	Pass       bool    `json:"pass"`
	Line       string  `json:"line"`
}

// VerifyJSON checks math expressions in a topic's detail page and returns JSON.
func VerifyJSON(topic string) string {
	if err := validateTopic(topic); err != nil {
		return errorJSON(err)
	}

	r := mustReg()
	d := r.GetDetail(topic)
	if d == nil {
		return errorJSON(&validationError{"topic", "no detail page"})
	}

	report := verify.Verify(topic, d.Content)

	results := make([]verifyResult, 0, len(report.Results))
	for _, res := range report.Results {
		results = append(results, verifyResult{
			Expression: res.Expression,
			Expected:   res.Expected,
			Got:        res.Got,
			Pass:       res.Pass,
			Line:       res.Line,
		})
	}

	return jsonMarshal(verifyResponse{
		Topic:   topic,
		Results: results,
		Pass:    report.Pass,
		Fail:    report.Fail,
		Total:   len(report.Results),
	})
}
