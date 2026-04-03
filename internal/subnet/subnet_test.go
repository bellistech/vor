package subnet

import (
	"testing"
)

func TestCalculateIPv4(t *testing.T) {
	tests := []struct {
		input     string
		network   string
		broadcast string
		prefix    int
		usable    string
	}{
		{"192.168.1.0/24", "192.168.1.0", "192.168.1.255", 24, "254"},
		{"10.0.0.0/8", "10.0.0.0", "10.255.255.255", 8, "16777214"},
		{"172.16.0.0/12", "172.16.0.0", "172.31.255.255", 12, "1048574"},
		{"192.168.1.0/30", "192.168.1.0", "192.168.1.3", 30, "2"},
		{"192.168.1.0/32", "192.168.1.0", "192.168.1.0", 32, "1"},
		{"192.168.1.0/31", "192.168.1.0", "192.168.1.1", 31, "2"},
	}

	for _, tt := range tests {
		info, err := Calculate(tt.input)
		if err != nil {
			t.Errorf("Calculate(%q) error: %v", tt.input, err)
			continue
		}
		if info.Network.String() != tt.network {
			t.Errorf("Calculate(%q) network = %s, want %s", tt.input, info.Network, tt.network)
		}
		if info.Broadcast.String() != tt.broadcast {
			t.Errorf("Calculate(%q) broadcast = %s, want %s", tt.input, info.Broadcast, tt.broadcast)
		}
		if info.Prefix != tt.prefix {
			t.Errorf("Calculate(%q) prefix = %d, want %d", tt.input, info.Prefix, tt.prefix)
		}
		if info.UsableHosts.String() != tt.usable {
			t.Errorf("Calculate(%q) usable = %s, want %s", tt.input, info.UsableHosts, tt.usable)
		}
	}
}

func TestCalculateIPv6(t *testing.T) {
	info, err := Calculate("2001:db8::/32")
	if err != nil {
		t.Fatalf("Calculate IPv6 error: %v", err)
	}
	if !info.IsIPv6 {
		t.Error("expected IsIPv6 = true")
	}
	if info.Prefix != 32 {
		t.Errorf("prefix = %d, want 32", info.Prefix)
	}
}

func TestCalculateWithMask(t *testing.T) {
	info, err := Calculate("192.168.1.0 255.255.255.0")
	if err != nil {
		t.Fatalf("Calculate with mask error: %v", err)
	}
	if info.Prefix != 24 {
		t.Errorf("prefix = %d, want 24", info.Prefix)
	}
}

func TestCalculateErrors(t *testing.T) {
	errors := []string{
		"not-an-ip",
		"192.168.1.0",
		"192.168.1.0 999.999.999.999",
	}
	for _, input := range errors {
		_, err := Calculate(input)
		if err == nil {
			t.Errorf("Calculate(%q) expected error", input)
		}
	}
}

func TestFormat(t *testing.T) {
	info, err := Calculate("192.168.1.0/24")
	if err != nil {
		t.Fatal(err)
	}
	out := Format(info)
	if out == "" {
		t.Fatal("Format returned empty string")
	}
}
