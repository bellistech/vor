// Package subnet implements an IPv4/IPv6 subnet calculator.
package subnet

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"net"
	"strconv"
	"strings"
)

// Info holds computed subnet information.
type Info struct {
	CIDR       string
	Network    net.IP
	Broadcast  net.IP // nil for IPv6
	Netmask    net.IP // nil for IPv6
	Wildcard   net.IP // nil for IPv6
	FirstHost  net.IP
	LastHost   net.IP
	TotalHosts *big.Int
	UsableHosts *big.Int
	Prefix     int
	IsIPv6     bool
}

// Calculate parses a CIDR string and returns subnet info.
func Calculate(input string) (*Info, error) {
	// Normalize: support "192.168.1.0 255.255.255.0" format
	input = strings.TrimSpace(input)
	if !strings.Contains(input, "/") {
		parts := strings.Fields(input)
		if len(parts) == 2 {
			prefix, err := maskToPrefix(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid subnet mask: %s", parts[1])
			}
			input = fmt.Sprintf("%s/%d", parts[0], prefix)
		} else {
			return nil, fmt.Errorf("expected CIDR notation (e.g., 192.168.1.0/24) or IP + mask")
		}
	}

	ip, network, err := net.ParseCIDR(input)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %v", err)
	}

	prefix, _ := network.Mask.Size()

	if ip.To4() != nil {
		return calcIPv4(network, prefix)
	}
	return calcIPv6(network, ip, prefix)
}

func calcIPv4(network *net.IPNet, prefix int) (*Info, error) {
	ip4 := network.IP.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("not a valid IPv4 address")
	}

	ipInt := binary.BigEndian.Uint32(ip4)
	maskInt := binary.BigEndian.Uint32(net.IP(network.Mask).To4())
	wildcardInt := ^maskInt
	broadcastInt := ipInt | wildcardInt

	networkIP := make(net.IP, 4)
	broadcast := make(net.IP, 4)
	mask := make(net.IP, 4)
	wildcard := make(net.IP, 4)

	binary.BigEndian.PutUint32(networkIP, ipInt)
	binary.BigEndian.PutUint32(broadcast, broadcastInt)
	binary.BigEndian.PutUint32(mask, maskInt)
	binary.BigEndian.PutUint32(wildcard, wildcardInt)

	totalHosts := uint64(1) << uint(32-prefix)
	usable := totalHosts
	if prefix < 31 {
		usable -= 2 // subtract network and broadcast
	}
	if prefix == 32 {
		usable = 1
	}

	firstHost := make(net.IP, 4)
	lastHost := make(net.IP, 4)

	if prefix >= 31 {
		copy(firstHost, networkIP)
		copy(lastHost, broadcast)
	} else {
		binary.BigEndian.PutUint32(firstHost, ipInt+1)
		binary.BigEndian.PutUint32(lastHost, broadcastInt-1)
	}

	return &Info{
		CIDR:        fmt.Sprintf("%s/%d", networkIP, prefix),
		Network:     networkIP,
		Broadcast:   broadcast,
		Netmask:     mask,
		Wildcard:    wildcard,
		FirstHost:   firstHost,
		LastHost:    lastHost,
		TotalHosts:  new(big.Int).SetUint64(totalHosts),
		UsableHosts: new(big.Int).SetUint64(usable),
		Prefix:      prefix,
		IsIPv6:      false,
	}, nil
}

func calcIPv6(network *net.IPNet, ip net.IP, prefix int) (*Info, error) {
	ip6 := ip.To16()
	if ip6 == nil {
		return nil, fmt.Errorf("not a valid IPv6 address")
	}

	networkIP := make(net.IP, 16)
	copy(networkIP, network.IP.To16())

	// Calculate total addresses: 2^(128-prefix)
	totalBits := 128 - prefix
	totalHosts := new(big.Int).Lsh(big.NewInt(1), uint(totalBits))
	usable := new(big.Int).Set(totalHosts) // IPv6 doesn't reserve network/broadcast

	// First host = network address
	firstHost := make(net.IP, 16)
	copy(firstHost, networkIP)

	// Last host = network | ~mask
	lastHost := make(net.IP, 16)
	copy(lastHost, networkIP)
	for i := 0; i < 16; i++ {
		lastHost[i] |= ^network.Mask[i]
	}

	return &Info{
		CIDR:        fmt.Sprintf("%s/%d", networkIP, prefix),
		Network:     networkIP,
		FirstHost:   firstHost,
		LastHost:    lastHost,
		TotalHosts:  totalHosts,
		UsableHosts: usable,
		Prefix:      prefix,
		IsIPv6:      true,
	}, nil
}

// Format returns a formatted string of subnet information.
func Format(info *Info) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\033[1;32mSubnet: %s\033[0m\n\n", info.CIDR))

	if !info.IsIPv6 {
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Network:", info.Network))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Broadcast:", info.Broadcast))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Netmask:", info.Netmask))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Wildcard:", info.Wildcard))
		sb.WriteString(fmt.Sprintf("  %-16s /%d\n", "Prefix:", info.Prefix))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "First Host:", info.FirstHost))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Last Host:", info.LastHost))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Total Addrs:", info.TotalHosts.String()))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Usable Hosts:", info.UsableHosts.String()))

		// Binary representation of mask
		mask4 := info.Netmask.To4()
		if mask4 != nil {
			maskBin := fmt.Sprintf("%08b.%08b.%08b.%08b", mask4[0], mask4[1], mask4[2], mask4[3])
			sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Mask (binary):", maskBin))
		}

		// Class
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Class:", ipv4Class(info.Network)))

		// Private/public
		if isPrivate(info.Network) {
			sb.WriteString(fmt.Sprintf("  %-16s Private (RFC 1918)\n", "Type:"))
		} else if info.Network.IsLoopback() {
			sb.WriteString(fmt.Sprintf("  %-16s Loopback\n", "Type:"))
		} else if info.Network.IsLinkLocalUnicast() {
			sb.WriteString(fmt.Sprintf("  %-16s Link-Local (APIPA)\n", "Type:"))
		} else if info.Network.IsMulticast() {
			sb.WriteString(fmt.Sprintf("  %-16s Multicast\n", "Type:"))
		} else {
			sb.WriteString(fmt.Sprintf("  %-16s Public\n", "Type:"))
		}
	} else {
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Network:", info.Network))
		sb.WriteString(fmt.Sprintf("  %-16s /%d\n", "Prefix:", info.Prefix))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "First Addr:", info.FirstHost))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Last Addr:", info.LastHost))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Total Addrs:", info.TotalHosts.String()))

		// IPv6 address type
		if info.Network.IsLoopback() {
			sb.WriteString(fmt.Sprintf("  %-16s Loopback\n", "Type:"))
		} else if info.Network.IsLinkLocalUnicast() {
			sb.WriteString(fmt.Sprintf("  %-16s Link-Local\n", "Type:"))
		} else if info.Network.IsMulticast() {
			sb.WriteString(fmt.Sprintf("  %-16s Multicast\n", "Type:"))
		} else if isULA(info.Network) {
			sb.WriteString(fmt.Sprintf("  %-16s Unique-Local (ULA)\n", "Type:"))
		} else if isGUA(info.Network) {
			sb.WriteString(fmt.Sprintf("  %-16s Global Unicast (GUA)\n", "Type:"))
		}

		// /64 subnets if prefix < 64
		if info.Prefix < 64 {
			slash64s := new(big.Int).Lsh(big.NewInt(1), uint(64-info.Prefix))
			sb.WriteString(fmt.Sprintf("  %-16s %s\n", "/64 Subnets:", slash64s.String()))
		}
	}

	return sb.String()
}

func maskToPrefix(mask string) (int, error) {
	ip := net.ParseIP(mask)
	if ip == nil {
		return 0, fmt.Errorf("invalid mask")
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return 0, fmt.Errorf("not IPv4 mask")
	}
	maskInt := binary.BigEndian.Uint32(ip4)

	// Count leading 1s
	prefix := 0
	for i := 31; i >= 0; i-- {
		if maskInt&(1<<uint(i)) != 0 {
			prefix++
		} else {
			break
		}
	}
	// Verify it's a valid mask (contiguous 1s)
	expected := uint32(math.MaxUint32) << uint(32-prefix)
	if maskInt != expected {
		return 0, fmt.Errorf("non-contiguous mask")
	}
	return prefix, nil
}

func ipv4Class(ip net.IP) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return "N/A"
	}
	first := ip4[0]
	switch {
	case first < 128:
		return "A (" + strconv.Itoa(int(first)) + ".0.0.0/8)"
	case first < 192:
		return "B (" + strconv.Itoa(int(first)) + ".0.0.0/16)"
	case first < 224:
		return "C (" + strconv.Itoa(int(first)) + ".0.0.0/24)"
	case first < 240:
		return "D (multicast)"
	default:
		return "E (reserved)"
	}
}

func isPrivate(ip net.IP) bool {
	private := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	for _, cidr := range private {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func isULA(ip net.IP) bool {
	ip16 := ip.To16()
	return ip16 != nil && (ip16[0]&0xfe) == 0xfc
}

func isGUA(ip net.IP) bool {
	ip16 := ip.To16()
	return ip16 != nil && (ip16[0]&0xe0) == 0x20
}
