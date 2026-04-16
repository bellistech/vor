package cscore

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestListTopicsJSON(t *testing.T) {
	initTestRegistry()
	result := ListTopicsJSON()

	var topics []topicSummary
	if err := json.Unmarshal([]byte(result), &topics); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result)
	}
	if len(topics) != 3 {
		t.Errorf("expected 3 topics, got %d", len(topics))
	}
	// Should be sorted alphabetically
	for i := 1; i < len(topics); i++ {
		if topics[i].Name < topics[i-1].Name {
			t.Errorf("topics not sorted: %s before %s", topics[i-1].Name, topics[i].Name)
		}
	}
	// bash should have detail
	for _, tp := range topics {
		if tp.Name == "bash" && !tp.HasDetail {
			t.Error("bash should have HasDetail=true")
		}
		if tp.Name == "curl" && tp.HasDetail {
			t.Error("curl should have HasDetail=false")
		}
	}
}

func TestGetSheetJSON(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		input     string
		wantError bool
		checkName string
	}{
		{"valid", "bash", false, "bash"},
		{"case insensitive", "BASH", false, "bash"},
		{"empty", "", true, ""},
		{"nonexistent", "nonexistent-topic-xyz", true, ""},
		{"path traversal", "../etc/passwd", true, ""},
		{"too long", strings.Repeat("a", 200), true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSheetJSON(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantError {
				if _, ok := data["error"]; !ok {
					t.Error("expected error field in response")
				}
			} else {
				if data["name"] != tt.checkName {
					t.Errorf("name = %v, want %q", data["name"], tt.checkName)
				}
				if data["content"] == nil || data["content"] == "" {
					t.Error("expected non-empty content")
				}
			}
		})
	}
}

func TestGetSheetJSON_Sections(t *testing.T) {
	initTestRegistry()
	result := GetSheetJSON("bash")
	var resp sheetResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp.Sections) == 0 {
		t.Error("expected sections")
	}
	if len(resp.SeeAlso) == 0 {
		t.Error("expected see_also entries")
	}
}

func TestGetDetailJSON(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid", "bash", false},
		{"no detail", "curl", true},
		{"empty", "", true},
		{"nonexistent", "nonexistent", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDetailJSON(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantError {
				if _, ok := data["error"]; !ok {
					t.Error("expected error field")
				}
			} else {
				if data["content"] == nil || data["content"] == "" {
					t.Error("expected content")
				}
			}
		})
	}
}

func TestGetDetailJSON_Fields(t *testing.T) {
	initTestRegistry()
	result := GetDetailJSON("bash")
	var resp detailResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp.Prerequisites) == 0 {
		t.Error("expected prerequisites")
	}
	if resp.Complexity == "" {
		t.Error("expected complexity")
	}
}

func TestRandomTopicJSON(t *testing.T) {
	initTestRegistry()
	result := RandomTopicJSON()
	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result)
	}
	if data["name"] == nil || data["name"] == "" {
		t.Error("expected a topic name")
	}
}
