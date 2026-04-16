package cscore

import (
	"strings"
	"testing"
)

func TestValidateTopic(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "bash", false},
		{"valid hyphen", "linux-basics", false},
		{"valid underscore", "my_topic", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 129), true},
		{"path traversal dotdot", "../etc/passwd", true},
		{"path traversal slash", "foo/bar", true},
		{"backslash", "foo\\bar", true},
		{"null byte", "bash\x00evil", true},
		{"invalid utf8", string([]byte{0xff, 0xfe}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTopic(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTopic(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "kubernetes", false},
		{"empty allowed", "", false},
		{"too long", strings.Repeat("q", 513), true},
		{"null byte", "test\x00evil", true},
		{"invalid utf8", string([]byte{0xff, 0xfe}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQuery(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateQuery(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateExpr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "2+2", false},
		{"empty", "", true},
		{"too long", strings.Repeat("1+", 600), true},
		{"null byte", "1+1\x002+2", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExpr(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateExpr(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "192.168.1.0/24", false},
		{"empty", "", true},
		{"too long", strings.Repeat("1", 129), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCIDR(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCIDR(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCategory(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "networking", false},
		{"empty", "", true},
		{"too long", strings.Repeat("c", 129), true},
		{"path traversal", "../hack", true},
		{"slash", "net/work", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCategory(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCategory(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "# Hello", false},
		{"empty", "", true},
		{"too large", strings.Repeat("x", 1<<20+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMarkdown(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMarkdown err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
