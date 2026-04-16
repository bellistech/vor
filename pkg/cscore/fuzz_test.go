package cscore

import (
	"encoding/json"
	"strings"
	"testing"
)

func FuzzCalcEval(f *testing.F) {
	initTestRegistry()
	f.Add("2+2")
	f.Add("10GB/2")
	f.Add("")
	f.Add("))))(((")
	f.Add("0xDEADBEEF ** 2")
	f.Add(strings.Repeat("9", 500))
	f.Add("sqrt(-1)")
	f.Add("1/0")
	f.Fuzz(func(t *testing.T, expr string) {
		result := CalcEval(expr)
		if !json.Valid([]byte(result)) {
			t.Errorf("invalid JSON for expr %q: %s", expr, result)
		}
	})
}

func FuzzSubnetCalc(f *testing.F) {
	initTestRegistry()
	f.Add("192.168.1.0/24")
	f.Add("2001:db8::/32")
	f.Add("")
	f.Add("not-an-ip")
	f.Add("999.999.999.999/99")
	f.Add("0.0.0.0/0")
	f.Fuzz(func(t *testing.T, input string) {
		result := SubnetCalc(input)
		if !json.Valid([]byte(result)) {
			t.Errorf("invalid JSON for input %q: %s", input, result)
		}
	})
}

func FuzzSearchJSON(f *testing.F) {
	initTestRegistry()
	f.Add("bash")
	f.Add("")
	f.Add(string([]byte{0, 0, 0}))
	f.Add("../../etc/passwd")
	f.Add(strings.Repeat("a", 1000))
	f.Add("<script>alert(1)</script>")
	f.Fuzz(func(t *testing.T, query string) {
		result := SearchJSON(query)
		if !json.Valid([]byte(result)) {
			t.Errorf("invalid JSON for query %q: %s", query, result)
		}
	})
}

func FuzzRenderMarkdownToHTML(f *testing.F) {
	initTestRegistry()
	f.Add("# Hello")
	f.Add("```go\nfmt.Println()\n```")
	f.Add("")
	f.Add("<script>alert(1)</script>")
	f.Add("| a | b |\n|---|---|\n| 1 | 2 |")
	f.Fuzz(func(t *testing.T, md string) {
		result := RenderMarkdownToHTML(md)
		if !json.Valid([]byte(result)) {
			t.Errorf("invalid JSON for markdown: %s", result)
		}
	})
}

func FuzzGetSheetJSON(f *testing.F) {
	initTestRegistry()
	f.Add("bash")
	f.Add("")
	f.Add("../../../etc/passwd")
	f.Add(strings.Repeat("x", 5000))
	f.Add(string([]byte{0xff, 0xfe}))
	f.Fuzz(func(t *testing.T, name string) {
		result := GetSheetJSON(name)
		if !json.Valid([]byte(result)) {
			t.Errorf("invalid JSON for name %q: %s", name, result)
		}
	})
}
