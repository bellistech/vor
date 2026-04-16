package cscore

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSubnetCalc(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid ipv4", "192.168.1.0/24", false},
		{"valid ipv6", "2001:db8::/32", false},
		{"ip mask format", "192.168.1.0 255.255.255.0", false},
		{"empty", "", true},
		{"too long", strings.Repeat("1", 129), true},
		{"invalid", "not-an-ip", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SubnetCalc(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantError {
				if _, ok := data["error"]; !ok {
					t.Errorf("expected error field, got: %v", data)
				}
			} else {
				if data["cidr"] == nil {
					t.Error("expected cidr field")
				}
				if data["network"] == nil {
					t.Error("expected network field")
				}
			}
		})
	}
}

func TestSubnetCalc_IPv4Fields(t *testing.T) {
	initTestRegistry()
	result := SubnetCalc("10.0.0.0/8")
	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if data["netmask"] != "255.0.0.0" {
		t.Errorf("netmask = %v, want 255.0.0.0", data["netmask"])
	}
	if data["is_ipv6"] != false {
		t.Error("expected is_ipv6=false")
	}
}
