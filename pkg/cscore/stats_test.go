package cscore

import (
	"encoding/json"
	"testing"
)

func TestStatsJSON(t *testing.T) {
	initTestRegistry()
	result := StatsJSON()

	var stats statsResponse
	if err := json.Unmarshal([]byte(result), &stats); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result)
	}
	if stats.TotalSheets != 3 {
		t.Errorf("TotalSheets = %d, want 3", stats.TotalSheets)
	}
	if stats.DetailPages != 1 {
		t.Errorf("DetailPages = %d, want 1", stats.DetailPages)
	}
	if stats.Categories != 2 {
		t.Errorf("Categories = %d, want 2", stats.Categories)
	}
	if stats.TotalLines == 0 {
		t.Error("expected non-zero TotalLines")
	}
	if len(stats.PerCategory) != 2 {
		t.Errorf("PerCategory len = %d, want 2", len(stats.PerCategory))
	}
	// Should be sorted by count descending
	if len(stats.PerCategory) >= 2 {
		if stats.PerCategory[0].Count < stats.PerCategory[1].Count {
			t.Error("PerCategory not sorted by count descending")
		}
	}
}
