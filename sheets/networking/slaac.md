# SLAAC (Stateless Address Autoconfiguration for IPv6)

Stateless Address Autoconfiguration allows IPv6 hosts to generate their own globally routable addresses without a DHCP server. Defined in RFC 4862, SLAAC combines a network prefix learned from Router Advertisements with a locally generated interface identifier to produce a complete 128-bit address. The host performs Duplicate Address Detection before using the address. SLAAC is the default address configuration mechanism for IPv6 and works exclusively with /64 prefixes.

---

## How SLAAC Works

### The SLAAC Process (Step by Step)

```bash
# 1. Interface comes up — link-local address generated immediately
#    fe80:: + interface identifier (EUI-64, random, or stable-privacy)
#    DAD performed on the link-local address first

# 2. Host sends Router Solicitation (RS) to ff02::2 (all-routers)
#    Triggers immediate RA instead of waiting for periodic RA

# 3. Router sends Router Advertisement (RA) containing:
#    - One or more prefix info options (prefix, length, L/A flags, lifetimes)
#    - M flag (Managed Address Config) — if set, use DHCPv6 for addresses
#    - O flag (Other Config) — if set, use DHCPv6 for DNS/options
#    - Default router lifetime, reachable time, retrans timer

# 4. For each prefix with A (Autonomous) flag set:
#    Host generates: prefix + interface identifier = global address
#    Host performs DAD on the new address

# 5. Address is usable after DAD succeeds (typically 1 second)

# Watch this process happen in real time
ip monitor address &
sudo tcpdump -i eth0 -n icmp6 &
sudo ip link set eth0 down && sudo ip link set eth0 up
```

### Verify SLAAC Is Working

```bash
# Check that SLAAC (autoconf) is enabled
sysctl net.ipv6.conf.eth0.autoconf
# 1 = enabled (default)

# Check that RAs are being accepted
sysctl net.ipv6.conf.eth0.accept_ra
# 0 = never, 1 = accept if not forwarding (default), 2 = always accept

# Show addresses with flags — look for "dynamic" (SLAAC-assigned)
ip -6 addr show dev eth0
# "scope global dynamic" = SLAAC address
# "scope global dynamic mngtmpaddr" = SLAAC with privacy extensions
# "scope global dynamic noprefixroute" = DHCPv6 address

# Request RA manually to trigger SLAAC
rdisc6 eth0
```

## Interface Identifier Generation

### EUI-64 (Traditional Method)

```bash
# Derives 64-bit interface ID from 48-bit MAC address
# MAC: 00:1a:2b:3c:4d:5e
# Step 1: Split into OUI + device ID: 00:1a:2b | 3c:4d:5e
# Step 2: Insert ff:fe in middle:      00:1a:2b:ff:fe:3c:4d:5e
# Step 3: Flip bit 7 (U/L bit):        02:1a:2b:ff:fe:3c:4d:5e
# Result interface ID:                 021a:2bff:fe3c:4d5e

# With prefix 2001:db8:1:1::/64, full address is:
# 2001:db8:1:1:021a:2bff:fe3c:4d5e

# Detect EUI-64 addresses (contain ff:fe in the interface ID)
ip -6 addr show dev eth0 | grep "ff:fe"

# Problem: EUI-64 exposes MAC address, enabling device tracking across networks
# Solution: Use stable-privacy or temporary addresses instead
```

### Stable Privacy Addresses (RFC 7217)

```bash
# Generates consistent address per network, but not trackable across networks
# Uses: secret_key + prefix + interface_name + DAD_counter + network_id
# Result: deterministic per-network, but different on each network

# Enable stable-privacy mode
# Generate a secret key first (stored in kernel)
# Method varies by distro — NetworkManager handles this automatically

# Check current address generation mode
sysctl net.ipv6.conf.eth0.addr_gen_mode
# 0 = EUI-64 (default on many systems)
# 2 = stable-privacy (RFC 7217)
# 3 = random (fully random for each address)

# Set to stable-privacy
sudo sysctl -w net.ipv6.conf.eth0.addr_gen_mode=2

# Verify the secret key exists (required for stable-privacy)
sysctl net.ipv6.conf.eth0.stable_secret
# If "error fetching" → no secret set, generate one:
sudo sysctl -w net.ipv6.conf.eth0.stable_secret=auto

# Persistent configuration in /etc/sysctl.d/10-ipv6-privacy.conf
# net.ipv6.conf.default.addr_gen_mode = 2
# net.ipv6.conf.default.stable_secret = auto
```

### Temporary / Privacy Addresses (RFC 4941 / RFC 8981)

```bash
# Generates randomized temporary addresses that rotate periodically
# Prevents long-term tracking even within the same network
# Outbound connections prefer temporary addresses; inbound uses stable

# Check if privacy extensions are enabled
sysctl net.ipv6.conf.eth0.use_tempaddr
# 0 = disabled
# 1 = generate temporary addresses but prefer public
# 2 = prefer temporary addresses (recommended for clients)

# Enable privacy extensions (prefer temporary)
sudo sysctl -w net.ipv6.conf.all.use_tempaddr=2
sudo sysctl -w net.ipv6.conf.default.use_tempaddr=2

# View temporary addresses (shown as "temporary" or "mngtmpaddr")
ip -6 addr show dev eth0 scope global temporary

# Temporary address lifetimes
sysctl net.ipv6.conf.eth0.temp_valid_lft      # max lifetime (default: 604800 = 7 days)
sysctl net.ipv6.conf.eth0.temp_prefrd_lft     # preferred lifetime (default: 86400 = 24h)

# Adjust rotation frequency (shorter = more privacy, more connections to re-establish)
sudo sysctl -w net.ipv6.conf.all.temp_prefrd_lft=3600   # new address every hour

# Maximum number of temporary addresses generated per prefix
sysctl net.ipv6.conf.eth0.max_addresses
# Default: 16

# Persistent in /etc/sysctl.d/10-ipv6-privacy.conf
echo "net.ipv6.conf.all.use_tempaddr = 2" | sudo tee /etc/sysctl.d/10-ipv6-privacy.conf
echo "net.ipv6.conf.default.use_tempaddr = 2" | sudo tee -a /etc/sysctl.d/10-ipv6-privacy.conf
```

## Address Lifetimes

### Valid and Preferred Lifetimes

```bash
# Each SLAAC address has two lifetimes (from the RA prefix option):
# Valid Lifetime — total time the address can be used (including deprecated state)
# Preferred Lifetime — time the address is preferred for new connections

# Timeline:
# |------- preferred -------|---- deprecated (valid but not preferred) ----|
# 0                  preferred_lft                                  valid_lft
#                           ^                                            ^
#                    no new connections                            address removed
#                    (existing continue)

# View address lifetimes
ip -6 addr show dev eth0
# Look for "valid_lft" and "preferred_lft" fields
# "valid_lft forever" = static address (no expiry)

# Typical RA-advertised lifetimes
# Preferred: 600-3600 seconds (10 min to 1 hour)
# Valid: 1800-86400 seconds (30 min to 24 hours)

# When preferred lifetime expires, address enters "deprecated" state
# Deprecated addresses:
# - Still valid for existing connections
# - NOT used for new outgoing connections
# - Eventually removed when valid lifetime expires
```

### Lifetime Refresh

```bash
# Periodic RAs refresh lifetimes — address stays active as long as router advertises
# If router stops advertising prefix, addresses eventually expire

# Monitor lifetime changes
watch -n 5 'ip -6 addr show dev eth0 | grep -A1 "scope global"'

# Simulate prefix withdrawal (on radvd router)
# Set AdvValidLifetime 0 and AdvPreferredLifetime 0 to withdraw prefix
# Hosts will deprecate and then remove addresses

# Flash renumbering: advertise new prefix with long lifetime,
# then advertise old prefix with short lifetime
# RFC 4192 — Procedures for Renumbering an IPv6 Network
```

## M/O Flag Interaction

### SLAAC vs DHCPv6 Decision Matrix

```bash
# Router Advertisement carries M and O flags that control host behavior:
#
# M=0, O=0 → SLAAC only (no DHCPv6 at all)
# M=0, O=1 → SLAAC for addresses + DHCPv6 for DNS/NTP/other options
# M=1, O=0 → DHCPv6 for addresses (SLAAC may still run if A flag set in prefix)
# M=1, O=1 → DHCPv6 for everything (most common stateful mode)
#
# Note: A flag is per-prefix in the prefix option, not a global RA flag
# A=1 in prefix → hosts use this prefix for SLAAC
# A=0 in prefix → hosts do NOT autoconfigure from this prefix

# Check what flags the router is advertising
rdisc6 -1 eth0
# Look for: "Managed address configuration" (M flag)
#           "Other configuration" (O flag)
#           "Autonomous address conf." in prefix info (A flag)

# View the flags in tcpdump
sudo tcpdump -i eth0 -n -vv 'icmp6 and ip6[40] == 134'
# M and O flags visible in RA header decode
```

### Common Deployment Patterns

```bash
# Pattern 1: Pure SLAAC (simplest, no server needed)
# radvd: AdvManagedFlag off; AdvOtherConfigFlag off;
# prefix: AdvAutonomous on;
# Host gets: address from SLAAC, DNS from RDNSS option in RA

# Pattern 2: SLAAC + stateless DHCPv6 (most common enterprise)
# radvd: AdvManagedFlag off; AdvOtherConfigFlag on;
# Host gets: address from SLAAC, DNS/NTP/options from DHCPv6

# Pattern 3: DHCPv6 stateful (full server control, like DHCPv4)
# radvd: AdvManagedFlag on; AdvOtherConfigFlag on;
# prefix: AdvAutonomous off;  (optional — can still allow SLAAC alongside)
# Host gets: address from DHCPv6, DNS/options from DHCPv6
# Note: default gateway STILL comes from RA (DHCPv6 does not provide it)

# Pattern 4: SLAAC + RDNSS (RFC 8106, no DHCPv6 at all)
# radvd: AdvManagedFlag off; AdvOtherConfigFlag off;
# radvd: RDNSS option with DNS server addresses
# Simplest deployment — single daemon, no DHCP infrastructure
```

## Duplicate Address Detection (DAD)

### DAD for SLAAC Addresses

```bash
# Every SLAAC-generated address goes through DAD before use
# Host sends NS with source :: to solicited-node multicast of tentative address
# Waits RetransTimer * DupAddrDetectTransmits (default: 1 second)
# If NA received → duplicate → address marked "dadfailed"
# If no response → address is unique → state changes to preferred

# Check current DAD setting
sysctl net.ipv6.conf.eth0.dad_transmits
# Default: 1 (one probe)

# Increase for unreliable links
sudo sysctl -w net.ipv6.conf.eth0.dad_transmits=3

# Check for failed addresses
ip -6 addr show dadfailed

# Clear a DAD failure (remove and re-add the address)
sudo ip -6 addr del 2001:db8::bad/64 dev eth0
sudo ip -6 addr add 2001:db8::bad/64 dev eth0

# Watch DAD happen in real time
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 135 and ip6 src ::' &
sudo ip link set eth0 down && sudo ip link set eth0 up
```

### Enhanced DAD (RFC 7527)

```bash
# Optimistic DAD allows using address immediately while DAD runs
# Reduces address configuration latency from ~1s to ~0s

# Enable optimistic DAD
sudo sysctl -w net.ipv6.conf.eth0.optimistic_dad=1

# With optimistic DAD, the address is immediately usable but:
# - Cannot be used as source for NS (neighbor resolution)
# - Cannot be used as source for RA
# - NA responses are delayed until DAD completes
# - If DAD fails, address is immediately removed

sysctl net.ipv6.conf.eth0.optimistic_dad
# 0 = standard DAD (default), 1 = optimistic DAD enabled
```

## Prefix Discovery from RA

### Prefix Information Option

```bash
# Each RA contains one or more Prefix Information options:
# - Prefix (e.g., 2001:db8:1:1::)
# - Prefix Length (must be /64 for SLAAC)
# - L flag (on-link) — prefix is on this link
# - A flag (autonomous) — use for SLAAC address generation
# - Valid Lifetime — how long the prefix is valid
# - Preferred Lifetime — how long addresses from this prefix are preferred

# View prefixes from RA
rdisc6 -1 eth0
# Shows each prefix with flags and lifetimes

# Multiple prefixes per interface
# A single RA can contain multiple prefix options
# Host generates a separate address for each A=1 prefix
# Example: host on a dual-uplink network may have addresses from both ISPs

# View all global addresses (one per SLAAC prefix + temporaries)
ip -6 addr show dev eth0 scope global
```

### RDNSS and DNSSL (RFC 8106)

```bash
# Modern RAs carry DNS configuration directly (no DHCPv6 needed)
# RDNSS — Recursive DNS Server (equivalent to nameserver in resolv.conf)
# DNSSL — DNS Search List (equivalent to search/domain in resolv.conf)

# View RDNSS/DNSSL from RA
rdisc6 -1 eth0
# Look for "Recursive DNS server" and "DNS search list"

# radvd configuration for RDNSS
# RDNSS 2001:db8:1:1::53 2001:4860:4860::8888
# {
#     AdvRDNSSLifetime 1800;
# };
#
# DNSSL example.com internal.example.com
# {
#     AdvDNSSLLifetime 1800;
# };

# Check if your system processes RDNSS (NetworkManager does by default)
nmcli device show eth0 | grep DNS
```

## Source Address Selection

### Default Address Selection (RFC 6724)

```bash
# When multiple addresses exist (SLAAC, temporary, DHCPv6), which is used?
# RFC 6724 defines a sorted preference table

# Key rules (simplified):
# 1. Prefer same scope as destination (global for global destinations)
# 2. Prefer non-deprecated addresses
# 3. Prefer temporary addresses over public (if use_tempaddr=2)
# 4. Prefer address with longest matching prefix to destination
# 5. Prefer addresses from the same label as destination

# View address selection policy
ip -6 rule show
cat /etc/gai.conf

# Force source address for testing
ping6 -I 2001:db8::1 2001:db8::2
curl --interface 2001:db8::1 -6 https://example.com

# List all candidate source addresses
ip -6 addr show dev eth0 scope global
# "preferred" = preferred for new connections
# "deprecated" = only for existing connections
```

## Sysctl Reference

### All SLAAC-Related Sysctls

```bash
# Core SLAAC controls
sysctl net.ipv6.conf.eth0.autoconf           # 1=enable SLAAC (default: 1)
sysctl net.ipv6.conf.eth0.accept_ra           # accept RAs (0=no, 1=if not fwd, 2=always)

# Interface identifier generation
sysctl net.ipv6.conf.eth0.addr_gen_mode       # 0=EUI-64, 2=stable-privacy, 3=random
sysctl net.ipv6.conf.eth0.stable_secret       # secret key for stable-privacy

# Privacy extensions (temporary addresses)
sysctl net.ipv6.conf.eth0.use_tempaddr        # 0=off, 1=on, 2=prefer temporary
sysctl net.ipv6.conf.eth0.temp_valid_lft      # temporary addr valid lifetime (604800)
sysctl net.ipv6.conf.eth0.temp_prefrd_lft     # temporary addr preferred lifetime (86400)
sysctl net.ipv6.conf.eth0.max_addresses        # max addresses per interface (16)

# DAD
sysctl net.ipv6.conf.eth0.dad_transmits       # DAD probes to send (1)
sysctl net.ipv6.conf.eth0.optimistic_dad      # optimistic DAD (0)

# Forwarding interaction
sysctl net.ipv6.conf.eth0.forwarding          # 1=router mode (disables SLAAC!)
# WARNING: enabling forwarding sets accept_ra=0 and autoconf=0 by default
# To keep SLAAC on a forwarding interface: set accept_ra=2
```

## NetworkManager Integration

### Managing SLAAC with nmcli

```bash
# View current IPv6 method
nmcli connection show eth0 | grep ipv6.method
# "auto" = SLAAC (default)
# "dhcp" = DHCPv6 only
# "manual" = static
# "ignore" = no IPv6

# Set to SLAAC (default)
nmcli connection modify eth0 ipv6.method auto

# SLAAC + additional static address
nmcli connection modify eth0 ipv6.method auto
nmcli connection modify eth0 +ipv6.addresses "2001:db8::42/64"

# Control privacy extensions via NetworkManager
nmcli connection modify eth0 ipv6.ip6-privacy 2
# 0 = disabled, 1 = enabled (prefer public), 2 = enabled (prefer temporary)

# Disable SLAAC, use DHCPv6 only
nmcli connection modify eth0 ipv6.method dhcp

# Apply changes
nmcli connection up eth0

# View assigned addresses
nmcli device show eth0 | grep IP6
```

## Troubleshooting

### Common SLAAC Problems

```bash
# No global address assigned
# 1. Check RA reception
rdisc6 eth0                                    # manually request RA
# 2. Check accept_ra
sysctl net.ipv6.conf.eth0.accept_ra            # must be 1 (or 2 if forwarding)
# 3. Check autoconf
sysctl net.ipv6.conf.eth0.autoconf             # must be 1
# 4. Check forwarding (kills SLAAC by default!)
sysctl net.ipv6.conf.eth0.forwarding           # if 1, set accept_ra=2
# 5. Check firewall
sudo ip6tables -L -n | grep icmpv6             # NDP must not be blocked

# Address shows "dadfailed"
ip -6 addr show dadfailed
# Another device has the same address — check for duplicate MACs or static conflicts
# Investigate: ndisc6 <tentative-addr> eth0

# Multiple addresses appearing (expected with privacy extensions)
ip -6 addr show dev eth0 scope global
# "dynamic" = SLAAC stable address
# "temporary dynamic" = privacy extension address
# Both are normal when use_tempaddr >= 1

# Address disappears after enabling forwarding
# forwarding=1 sets accept_ra=0 and autoconf=0 by default
sudo sysctl -w net.ipv6.conf.eth0.accept_ra=2
sudo sysctl -w net.ipv6.conf.eth0.autoconf=1

# Prefix changed but old address lingers
# Old address stays until valid_lft expires — this is by design
# Force removal: sudo ip -6 addr flush dev eth0 scope global
```

---

## Tips

- SLAAC only works with /64 prefixes. This is a hard requirement in RFC 4862, not a convention. If your prefix is anything other than /64, SLAAC will not generate addresses.
- Enabling IPv6 forwarding (sysctl forwarding=1) disables SLAAC by default. If a Linux box needs to both forward and use SLAAC, set accept_ra=2 on that interface.
- Use stable-privacy addresses (addr_gen_mode=2) on servers for consistent addressing without exposing the MAC. Use temporary addresses (use_tempaddr=2) on client devices for privacy.
- DHCPv6 cannot provide a default gateway. Even in full stateful DHCPv6 mode, the default route comes from Router Advertisements. You always need RA for IPv6 routing.
- If a host has multiple global addresses (SLAAC + temporary + DHCPv6), RFC 6724 source address selection determines which is used. Temporary addresses are preferred when use_tempaddr=2.
- RDNSS (RFC 8106) in Router Advertisements can replace DHCPv6 for DNS. This is the simplest deployment: radvd with RDNSS provides addresses and DNS without any DHCP infrastructure.
- DAD adds approximately 1 second of latency to address configuration. For latency-sensitive environments (VMs, containers), enable optimistic_dad to use addresses immediately.
- Privacy extension addresses rotate based on temp_prefrd_lft (default 24 hours). For high-privacy environments, reduce this to 3600 (1 hour) at the cost of more frequent address churn.
- When troubleshooting "no IPv6 connectivity," follow this order: link-local present, RA received, global address assigned, default route exists, firewall rules allow NDP.
- The A flag (Autonomous) is per-prefix, not per-RA. A single RA can contain prefixes with A=1 (use SLAAC) and A=0 (do not autoconfigure) simultaneously.

---

## See Also

- ipv6, ndp, dhcpv6, dhcp

## References

- [RFC 4862 — IPv6 Stateless Address Autoconfiguration](https://www.rfc-editor.org/rfc/rfc4862)
- [RFC 7217 — A Method for Generating Semantically Opaque Interface Identifiers with IPv6 SLAAC](https://www.rfc-editor.org/rfc/rfc7217)
- [RFC 8981 — Temporary Address Extensions for Stateless Address Autoconfiguration in IPv6](https://www.rfc-editor.org/rfc/rfc8981)
- [RFC 4941 — Privacy Extensions for Stateless Address Autoconfiguration in IPv6](https://www.rfc-editor.org/rfc/rfc4941)
- [RFC 8106 — IPv6 Router Advertisement Options for DNS Configuration](https://www.rfc-editor.org/rfc/rfc8106)
- [RFC 6724 — Default Address Selection for Internet Protocol Version 6](https://www.rfc-editor.org/rfc/rfc6724)
- [RFC 4429 — Optimistic Duplicate Address Detection for IPv6](https://www.rfc-editor.org/rfc/rfc4429)
- [RFC 4192 — Procedures for Renumbering an IPv6 Network without a Flag Day](https://www.rfc-editor.org/rfc/rfc4192)
- [Linux Kernel — IPv6 Sysctl Documentation](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
