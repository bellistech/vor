package cscore

import (
	"fmt"
	"math"
	"strconv"

	"github.com/bellistech/cs/internal/calc"
)

type calcResponse struct {
	Expr      string  `json:"expr"`
	Value     float64 `json:"value"`
	Formatted string  `json:"formatted"`
	Hex       string  `json:"hex,omitempty"`
	Oct       string  `json:"oct,omitempty"`
	Bin       string  `json:"bin,omitempty"`
	Unit      string  `json:"unit,omitempty"`
}

// CalcEval evaluates an arithmetic expression and returns JSON.
func CalcEval(expr string) string {
	if err := validateExpr(expr); err != nil {
		return errorJSON(err)
	}

	// Try unit-aware eval first
	ur, err := calc.EvalWithUnits(expr)
	if err != nil {
		return errorJSON(err)
	}

	if math.IsInf(ur.Value, 0) || math.IsNaN(ur.Value) {
		return errorJSON(fmt.Errorf("result is %v", ur.Value))
	}

	resp := calcResponse{
		Expr:      expr,
		Value:     ur.Value,
		Formatted: calc.FormatWithUnit(ur),
		Unit:      ur.Unit,
	}

	// Add base conversions for integer values without units
	if ur.Unit == "" && ur.Value == math.Trunc(ur.Value) && math.Abs(ur.Value) < 1e18 {
		iv := int64(ur.Value)
		resp.Hex = fmt.Sprintf("0x%X", iv)
		resp.Oct = fmt.Sprintf("0o%o", iv)
		resp.Bin = fmt.Sprintf("0b%s", strconv.FormatInt(iv, 2))
	}

	return jsonMarshal(resp)
}
