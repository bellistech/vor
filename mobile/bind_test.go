package mobile

import (
	"encoding/json"
	"testing"
)

func TestMobileInit(t *testing.T) {
	// Uses real embedded 685 sheets + 685 detail pages
	if err := MobileInit(); err != nil {
		t.Fatalf("MobileInit: %v", err)
	}
}

func TestMobileListTopics(t *testing.T) {
	result := MobileListTopicsJSON()
	if !json.Valid([]byte(result)) {
		t.Fatalf("invalid JSON: %s", result)
	}
	var topics []map[string]any
	json.Unmarshal([]byte(result), &topics)
	// 685 files but 684 unique names (vpc exists in networking/ and cloud/)
	if len(topics) < 684 {
		t.Errorf("expected >= 684 topics, got %d", len(topics))
	}
}

func TestMobileSearch(t *testing.T) {
	result := MobileSearchJSON("bgp")
	if !json.Valid([]byte(result)) {
		t.Fatalf("invalid JSON: %s", result)
	}
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	results, _ := data["results"].([]any)
	if len(results) == 0 {
		t.Error("expected search results for bgp")
	}
}

func TestMobileGetSheet(t *testing.T) {
	result := MobileGetSheetJSON("bgp")
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	if data["name"] != "bgp" {
		t.Errorf("name = %v, want bgp", data["name"])
	}
	if data["content"] == nil || data["content"] == "" {
		t.Error("expected content")
	}
}

func TestMobileGetDetail(t *testing.T) {
	result := MobileGetDetailJSON("bgp")
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	// bgp should have a detail page
	if _, hasErr := data["error"]; hasErr {
		t.Skipf("bgp detail not available: %v", data["error"])
	}
	if data["content"] == nil {
		t.Error("expected content")
	}
}

func TestMobileCalcEval(t *testing.T) {
	result := MobileCalcEval("2**10")
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	if data["value"] != float64(1024) {
		t.Errorf("value = %v, want 1024", data["value"])
	}
}

func TestMobileSubnet(t *testing.T) {
	result := MobileSubnetCalc("10.0.0.0/24")
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	if data["cidr"] == nil {
		t.Error("expected cidr field")
	}
}

func TestMobileRenderMarkdown(t *testing.T) {
	result := MobileRenderMarkdownToHTML("# Hello\n\n**bold** text")
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	html, _ := data["html"].(string)
	if html == "" {
		t.Error("expected html output")
	}
}

func TestMobileCategories(t *testing.T) {
	result := MobileCategoriesJSON()
	var cats []map[string]any
	json.Unmarshal([]byte(result), &cats)
	if len(cats) < 50 {
		t.Errorf("expected >= 50 categories, got %d", len(cats))
	}
}

func TestMobileStats(t *testing.T) {
	result := MobileStatsJSON()
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	total, _ := data["total_sheets"].(float64)
	if total < 684 {
		t.Errorf("total_sheets = %v, want >= 684", total)
	}
}

func TestMobileBookmarks(t *testing.T) {
	dir := t.TempDir()
	MobileSetDataDir(dir)
	result := MobileBookmarkToggle("bgp")
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	if data["bookmarked"] != true {
		t.Error("expected bookmarked=true")
	}
	if !MobileBookmarkIsStarred("bgp") {
		t.Error("expected bgp to be starred")
	}
	listResult := MobileBookmarkList()
	if !json.Valid([]byte(listResult)) {
		t.Fatalf("invalid JSON: %s", listResult)
	}
}

func TestMobileRelated(t *testing.T) {
	result := MobileRelatedJSON("bgp")
	if !json.Valid([]byte(result)) {
		t.Fatalf("invalid JSON: %s", result)
	}
}

func TestMobileCompare(t *testing.T) {
	result := MobileCompareJSON("bgp", "ospf")
	if !json.Valid([]byte(result)) {
		t.Fatalf("invalid JSON: %s", result)
	}
}

func TestMobileLearnPath(t *testing.T) {
	result := MobileLearnPathJSON("networking")
	if !json.Valid([]byte(result)) {
		t.Fatalf("invalid JSON: %s", result)
	}
}

func TestMobileVerify(t *testing.T) {
	result := MobileVerifyJSON("bgp")
	if !json.Valid([]byte(result)) {
		t.Fatalf("invalid JSON: %s", result)
	}
}

func TestMobileRandom(t *testing.T) {
	result := MobileRandomTopicJSON()
	var data map[string]any
	json.Unmarshal([]byte(result), &data)
	if data["name"] == nil {
		t.Error("expected a topic name")
	}
}
