package cscore

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderMarkdownToHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		wantErr  bool
	}{
		{"heading", "# Hello", "<h1>Hello</h1>", false},
		{"bold", "**bold**", "<strong>bold</strong>", false},
		{"code fence", "```go\nfmt.Println()\n```", "<code", false},
		{"link", "[text](https://example.com)", `href="https://example.com"`, false},
		{"table", "| a | b |\n|---|---|\n| 1 | 2 |", "<table>", false},
		{"empty", "", "", true},
		{"too large", strings.Repeat("x", 1<<20+1), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdownToHTML(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantErr {
				if _, ok := data["error"]; !ok {
					t.Error("expected error field")
				}
				return
			}
			html, ok := data["html"].(string)
			if !ok || html == "" {
				t.Fatal("expected html field")
			}
			if tt.contains != "" && !strings.Contains(html, tt.contains) {
				t.Errorf("html missing %q:\n%s", tt.contains, html)
			}
		})
	}
}

func TestRenderMarkdownToHTML_XSS(t *testing.T) {
	// BlackMage Gate S2: goldmark default escaping prevents XSS
	tests := []struct {
		name  string
		input string
		bad   string // must NOT appear in output
	}{
		{"script tag", "<script>alert(1)</script>", "<script>"},
		{"img onerror", "<img src=x onerror=alert(1)>", "onerror"},
		{"iframe", "<iframe src=evil.com>", "<iframe"},
		{"javascript href", "<a href='javascript:alert(1)'>x</a>", "javascript:"},
		{"form", "<form action='https://evil.com'><input>", "<form"},
		{"object tag", "<object data='evil.swf'>", "<object"},
		{"embed tag", "<embed src='evil.swf'>", "<embed"},
		{"link tag", "<link href='https://evil.com/steal.css'>", "<link"},
		{"meta refresh", "<meta http-equiv='refresh' content='0;url=evil'>", "<meta"},
		{"base tag", "<base href='https://evil.com'>", "<base"},
		{"svg onload", "<svg onload=alert(1)>", "onload"},
		{"event handler", "<div onclick=alert(1)>", "onclick"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdownToHTML(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			html, _ := data["html"].(string)
			if strings.Contains(html, tt.bad) {
				t.Errorf("XSS vector %q found in output:\n%s", tt.bad, html)
			}
		})
	}
}
