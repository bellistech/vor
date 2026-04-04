# DHCPv6 (Dynamic Host Configuration Protocol for IPv6)

DHCPv6 provides stateful address assignment, prefix delegation, and configuration distribution for IPv6 networks. Defined in RFC 8415, it operates on UDP ports 546 (client) and 547 (server/relay) using a four-message exchange (Solicit-Advertise-Request-Reply) or two-message rapid commit. Unlike DHCPv4, DHCPv6 does not provide a default gateway — that always comes from Router Advertisements. DHCPv6 identifies clients by DUID (DHCP Unique Identifier) and IAID (Identity Association Identifier) rather than MAC address.

---

## Message Types

### Four-Message Exchange (Normal)

```
Client                         Server
  |                              |
  |  -- Solicit (1) ---------->  |  Client multicasts to ff02::1:2
  |     src: link-local:546      |  dst: ff02::1:2:547
  |     contains: client DUID,   |
  |     IA_NA/IA_PD options      |
  |                              |
  |  <- Advertise (2) ---------  |  Server offers addresses/config
  |     contains: server DUID,   |
  |     offered addresses,       |
  |     preference value         |
  |                              |
  |  -- Request (3) ---------->  |  Client requests from chosen server
  |     contains: server DUID,   |
  |     requested addresses      |
  |                              |
  |  <- Reply (4) -------------  |  Server confirms assignment
  |     contains: assigned       |
  |     addresses, DNS, lifetimes|
  |                              |
```

### Two-Message Exchange (Rapid Commit)

```bash
# Rapid Commit reduces the exchange to Solicit + Reply (skips Advertise + Request)
# Both client and server must support it
# Client includes Rapid Commit option in Solicit
# Server responds directly with Reply containing Rapid Commit option

# Useful when only one server exists on the link
# Reduces latency from 4 messages to 2
```

### All Message Types

```bash
# Core messages (four-message exchange)
# 1  Solicit       — Client seeks servers
# 2  Advertise     — Server responds to Solicit
# 3  Request       — Client requests addresses from specific server
# 4  Confirm       — Client verifies addresses after link change
# 5  Renew         — Client renews addresses from original server (unicast)
# 6  Rebind        — Client renews addresses from any server (multicast)
# 7  Reply         — Server response to Request/Renew/Rebind/Release/Decline
# 8  Release       — Client releases addresses
# 9  Decline       — Client reports address already in use (DAD failed)
# 10 Reconfigure   — Server triggers client to re-request config
# 11 Information-Request — Client requests config only (no addresses)
# 12 Relay-Forward — Relay forwards client message to server
# 13 Relay-Reply   — Server sends reply through relay to client
```

## Stateful vs Stateless Mode

### Stateful DHCPv6 (Full Address Assignment)

```bash
# Server assigns addresses AND configuration options
# Triggered by M (Managed) flag in Router Advertisement
# Uses IA_NA (Identity Association for Non-temporary Addresses)

# Client perspective — request address via dhclient
sudo dhclient -6 eth0

# Client perspective — request address via dhcpcd
sudo dhcpcd -6 eth0

# Verify stateful DHCPv6 address
ip -6 addr show dev eth0
# Look for "scope global dynamic" WITHOUT "mngtmpaddr" flag
# DHCPv6 addresses show "noprefixroute" flag

# Check that M flag is set in RA
rdisc6 -1 eth0
# "Managed address configuration" should show "Yes"
```

### Stateless DHCPv6 (Information-Only)

```bash
# Server provides ONLY configuration (DNS, NTP, etc.) — no addresses
# Triggered by O (Other) flag in Router Advertisement
# Addresses come from SLAAC, config comes from DHCPv6
# Uses Information-Request message (type 11)

# Client perspective — request info only
sudo dhclient -6 -S eth0

# Verify: address from SLAAC, DNS from DHCPv6
ip -6 addr show dev eth0        # address should show "mngtmpaddr" (SLAAC)
cat /etc/resolv.conf             # DNS servers from DHCPv6

# Check that O flag is set (M flag may be off)
rdisc6 -1 eth0
# "Other configuration" should show "Yes"
```

## Client Identification

### DUID (DHCP Unique Identifier)

```bash
# DUID uniquely identifies a DHCPv6 client or server
# Persists across reboots and interface changes (unlike MAC-based DHCPv4)

# DUID types:
# 1 — DUID-LLT (Link-Layer + Time) — most common
#     Contains: hardware type + time + link-layer address
# 2 — DUID-EN (Enterprise Number)
#     Contains: enterprise number + identifier
# 3 — DUID-LL (Link-Layer only)
#     Contains: hardware type + link-layer address
# 4 — DUID-UUID (RFC 6355)
#     Contains: UUID

# View client DUID (dhclient)
cat /var/lib/dhcp/dhclient6.leases | grep "dhcp6.client-id"

# View client DUID (dhcpcd)
cat /var/lib/dhcpcd/duid

# View client DUID (NetworkManager)
nmcli connection show eth0 | grep dhcp6.duid
```

### IAID (Identity Association Identifier)

```bash
# IAID identifies a specific collection of addresses on an interface
# Each interface gets a unique IAID (typically derived from interface index)

# IA types:
# IA_NA — Identity Association for Non-temporary Addresses
# IA_TA — Identity Association for Temporary Addresses (rarely used)
# IA_PD — Identity Association for Prefix Delegation

# A client may have multiple IAs:
# - One IA_NA per interface (for address assignment)
# - One IA_PD per interface (for prefix delegation)
```

## ISC DHCPv6 Server Configuration

### Basic Stateful Server

```bash
# /etc/dhcp/dhcpd6.conf

# Server DUID (auto-generated on first start, or set manually)
# server-duid "\000\001\000\001...";

# Global options
default-lease-time 3600;
preferred-lifetime 1800;
option dhcp6.name-servers 2001:db8:1::53, 2001:4860:4860::8888;
option dhcp6.domain-search "example.com", "internal.example.com";

# Subnet with address pool
subnet6 2001:db8:1:1::/64 {
    range6 2001:db8:1:1::100 2001:db8:1:1::1ff;
}

# Subnet with temporary address pool
subnet6 2001:db8:1:2::/64 {
    range6 2001:db8:1:2::/64 temporary;
}

# Fixed host reservation (by DUID)
host server1 {
    host-identifier option dhcp6.client-id 00:01:00:01:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx;
    fixed-address6 2001:db8:1:1::10;
}
```

### Prefix Delegation Server

```bash
# /etc/dhcp/dhcpd6.conf — prefix delegation section

# Delegate /48 prefixes from a /32 pool
subnet6 2001:db8::/32 {
    prefix6 2001:db8:100:: 2001:db8:1ff:: /48;
}

# Delegate /56 prefixes (common for residential CPE)
subnet6 2001:db8::/32 {
    prefix6 2001:db8:aa00:: 2001:db8:aaff:: /56;
}

# Fixed prefix delegation to specific CPE
host home-router {
    host-identifier option dhcp6.client-id 00:01:00:01:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx;
    fixed-prefix6 2001:db8:abcd:: /48;
}
```

### Server Management

```bash
# Start ISC DHCPv6 server (separate daemon from DHCPv4)
sudo systemctl start isc-dhcp-server6
sudo systemctl enable isc-dhcp-server6

# Check config syntax
dhcpd -6 -t -cf /etc/dhcp/dhcpd6.conf

# Listen on specific interface
# /etc/default/isc-dhcp-server:
INTERFACESv6="eth0"

# View active leases
cat /var/lib/dhcp/dhcpd6.leases

# Debug mode (foreground)
dhcpd -6 -d -cf /etc/dhcp/dhcpd6.conf
```

## dnsmasq DHCPv6 Configuration

### Basic dnsmasq DHCPv6

```bash
# /etc/dnsmasq.conf or /etc/dnsmasq.d/dhcpv6.conf

# Enable DHCPv6 on interface
interface=eth0

# Stateful DHCPv6 address range
dhcp-range=::100,::1ff,constructor:eth0,64,12h

# Stateless DHCPv6 (config only, SLAAC for addresses)
# dhcp-range=::,constructor:eth0,ra-stateless

# Enable Router Advertisements (required for DHCPv6 to work)
enable-ra

# RA with M flag (managed — use DHCPv6 for addresses)
dhcp-range=::100,::1ff,constructor:eth0,ra-names,64,12h

# RA with O flag only (SLAAC addresses, DHCPv6 for config)
# dhcp-range=::,constructor:eth0,ra-stateless,ra-names

# Set DNS server via DHCPv6
dhcp-option=option6:dns-server,[2001:db8:1::53],[2001:4860:4860::8888]

# Set domain search list
dhcp-option=option6:domain-search,example.com,internal.example.com

# DHCPv6 host reservation (by DUID)
dhcp-host=id:00:01:00:01:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx,[2001:db8:1:1::10],server1

# Prefix delegation (delegate /56 from /48)
dhcp-range=2001:db8:1:100::,2001:db8:1:1ff::,56,12h
```

### dnsmasq Management

```bash
# Test configuration
dnsmasq --test

# Start dnsmasq
sudo systemctl start dnsmasq
sudo systemctl enable dnsmasq

# View DHCPv6 leases
cat /var/lib/misc/dnsmasq.leases

# Debug mode
dnsmasq --no-daemon --log-dhcp --log-queries
```

## Prefix Delegation (PD)

### How PD Works (RFC 3633)

```bash
# Prefix Delegation assigns entire prefixes (not just addresses) to routers
# Common use: ISP delegates /48 or /56 to customer CPE (home/office router)
# CPE then subnets the delegated prefix across its internal interfaces

# Typical flow:
# 1. CPE sends Solicit with IA_PD option on WAN interface
# 2. ISP DHCPv6 server replies with delegated prefix (e.g., 2001:db8:abcd::/48)
# 3. CPE carves /64 subnets from the delegated prefix
# 4. CPE runs radvd on LAN interfaces advertising the /64 subnets

# Request prefix delegation with dhclient
# /etc/dhcp/dhclient6.conf:
# interface "eth0" {
#     send dhcp6.client-id = concat(00:01:00:01,
#         hardware);
#     request;
#     also request dhcp6.ia-pd;
# }

sudo dhclient -6 -P eth0    # request prefix only
sudo dhclient -6 -N -P eth0 # request both address and prefix

# dhcpcd prefix delegation
sudo dhcpcd -6 --ia_pd 1 eth0

# View delegated prefix
ip -6 route show | grep "proto ra"
```

### CPE Prefix Splitting

```bash
# Example: received /48, split into /64 subnets for LAN interfaces
# 2001:db8:abcd::/48 splits into 65,536 possible /64 subnets:
# 2001:db8:abcd:0000::/64 → LAN1
# 2001:db8:abcd:0001::/64 → LAN2
# 2001:db8:abcd:0002::/64 → Guest WiFi
# 2001:db8:abcd:0003::/64 → IoT VLAN
# ...
# 2001:db8:abcd:ffff::/64 → last possible subnet

# radvd on LAN interface advertising delegated subnet
# /etc/radvd.conf:
# interface eth1 {
#     AdvSendAdvert on;
#     prefix 2001:db8:abcd:1::/64 {
#         AdvOnLink on;
#         AdvAutonomous on;
#     };
# };
```

## Relay Agents

### DHCPv6 Relay Configuration

```bash
# DHCPv6 clients multicast to ff02::1:2 (link-scoped, does not cross routers)
# Relay agents forward client messages to servers on other subnets

# ISC DHCPv6 relay
dhcrelay -6 -l eth0 -u 2001:db8:1::dhcp%eth1
# -l = listen interface (downstream, toward clients)
# -u = upstream server address and interface

# Multiple upstream servers
dhcrelay -6 -l eth0 -u 2001:db8:1::dhcp1%eth1 -u 2001:db8:1::dhcp2%eth1

# Relay encapsulation:
# Client sends Solicit to ff02::1:2
# Relay wraps in Relay-Forward (type 12) with:
#   - link-address: address on the client-facing interface
#   - peer-address: client's link-local address
# Server replies with Relay-Reply (type 13)
# Relay unwraps and forwards Reply to client

# Cisco IOS DHCPv6 relay
# interface GigabitEthernet0/1
#   ipv6 dhcp relay destination 2001:db8:1::dhcp
```

## Client Configuration

### dhclient (ISC)

```bash
# Obtain DHCPv6 address
sudo dhclient -6 eth0

# Release DHCPv6 lease
sudo dhclient -6 -r eth0

# Request information only (stateless)
sudo dhclient -6 -S eth0

# Request prefix delegation
sudo dhclient -6 -P eth0

# Verbose debug mode
sudo dhclient -6 -d -v eth0

# View current lease
cat /var/lib/dhcp/dhclient6.leases

# Configuration: /etc/dhcp/dhclient6.conf
# interface "eth0" {
#     send dhcp6.client-id = concat(00:01:00:01,
#         hardware);
#     request dhcp6.name-servers;
#     request dhcp6.domain-search;
# }
```

### dhcpcd

```bash
# Obtain DHCPv6 address
sudo dhcpcd -6 eth0

# Request both address and prefix delegation
sudo dhcpcd -6 --ia_na 1 --ia_pd 1 eth0

# Release lease
sudo dhcpcd -6 -k eth0

# Configuration: /etc/dhcpcd.conf
# interface eth0
#   ipv6rs
#   ia_na 1
#   ia_pd 1
```

### NetworkManager (nmcli)

```bash
# Set to DHCPv6 for addresses
nmcli connection modify eth0 ipv6.method dhcp

# Set to SLAAC + stateless DHCPv6 (default "auto")
nmcli connection modify eth0 ipv6.method auto

# Set DHCP DUID type
nmcli connection modify eth0 ipv6.dhcp-duid ll

# Set DHCP request options
nmcli connection modify eth0 ipv6.dhcp-send-hostname yes

# Apply changes
nmcli connection up eth0

# View DHCPv6 status
nmcli device show eth0 | grep "IP6\|DHCP6"
```

## Dibbler (Advanced DHCPv6)

### Dibbler Server

```bash
# /etc/dibbler/server.conf
# Full-featured DHCPv6 server with PD support

# iface eth0 {
#     class {
#         pool 2001:db8:1:1::100-2001:db8:1:1::1ff
#     }
#     pd-class {
#         pd-pool 2001:db8:2::/48
#         pd-length 56
#     }
#     option dns-server 2001:db8:1::53
#     option domain example.com
#     rapid-commit yes
# }

# Start dibbler server
sudo dibbler-server start

# Check status
sudo dibbler-server status

# View leases
cat /var/lib/dibbler/server-AddrMgr.xml
```

### Dibbler Client

```bash
# /etc/dibbler/client.conf
# iface eth0 {
#     ia
#     pd
#     option dns-server
#     option domain
# }

# Start dibbler client
sudo dibbler-client start

# View assigned addresses/prefixes
sudo dibbler-client status
```

## Monitoring and Troubleshooting

### Packet Capture

```bash
# Capture all DHCPv6 traffic (ports 546/547)
sudo tcpdump -i eth0 -n 'port 546 or port 547'

# Verbose decode of DHCPv6 messages
sudo tcpdump -i eth0 -n -vv 'port 546 or port 547'

# Filter by message type using tshark
tshark -i eth0 -f 'port 546 or port 547' -Y 'dhcpv6.msgtype == 1'  # Solicit
tshark -i eth0 -f 'port 546 or port 547' -Y 'dhcpv6.msgtype == 7'  # Reply

# Watch for DHCPv6 relay messages
sudo tcpdump -i eth0 -n -vv 'port 547' | grep -i "relay"
```

### Lease Verification

```bash
# Verify DHCPv6 address is assigned
ip -6 addr show dev eth0 | grep "scope global dynamic"

# Check if address came from DHCPv6 vs SLAAC
ip -6 addr show dev eth0
# DHCPv6: "noprefixroute" flag present
# SLAAC:  "mngtmpaddr" flag present (or no special flag)

# View active DHCPv6 lease details
cat /var/lib/dhcp/dhclient6.leases     # ISC dhclient
cat /var/lib/dhcpcd/duid               # dhcpcd
```

### Common Issues

```bash
# No DHCPv6 response — check multicast
# DHCPv6 uses ff02::1:2 (all DHCP agents) and ff05::1:3 (all DHCP servers)
ping6 ff02::1:2%eth0                    # should get replies from servers/relays

# M/O flags not set in RA
rdisc6 -1 eth0
# If "Managed address configuration: No" → server is running but RA does not
# tell clients to use it. Fix: set M=1 on the router/radvd

# Server logs
journalctl -u isc-dhcp-server6 --since "1 hour ago"
journalctl -u dnsmasq --since "1 hour ago" | grep DHCPv6

# Client DUID mismatch — server rejects client
# Verify DUID matches reservation
cat /var/lib/dhcp/dhclient6.leases | grep "dhcp6.client-id"

# Relay not forwarding
# Check that relay is listening on correct interface
# Verify link-address in Relay-Forward matches a configured subnet on server

# Prefix delegation fails
# Ensure server has pd-pool configured
# Ensure client is requesting IA_PD (not just IA_NA)
sudo dhclient -6 -d -v -P eth0        # debug PD request
```

## DHCPv6 vs DHCPv4 Differences

```bash
# Key differences from DHCPv4:
# 1. No default gateway — always from RA (ICMPv6 type 134)
# 2. Client ID is DUID (persistent) not MAC (per-interface)
# 3. Uses multicast ff02::1:2 not broadcast 255.255.255.255
# 4. Ports: 546/547 (not 67/68)
# 5. No BOOTP compatibility
# 6. Prefix Delegation built-in (IA_PD)
# 7. Rapid Commit option for 2-message exchange
# 8. Reconfigure message (server-initiated)
# 9. Relay uses encapsulation (Relay-Forward/Reply) not giaddr field
# 10. Multiple addresses per IA (not just one per interface)
```

---

## Tips

- DHCPv6 never provides a default gateway. This is the most common DHCPv6 misconception. The default route always comes from Router Advertisements, even in full stateful mode.
- The M and O flags in Router Advertisements tell clients whether to use DHCPv6. If your DHCPv6 server is running but clients are not contacting it, check that M and/or O flags are set in the RA.
- DUID is persistent across interfaces and reboots, unlike DHCPv4's MAC-based identification. This means moving a NIC between machines does not transfer the lease.
- Prefix Delegation is how ISPs provide IPv6 to home routers. The CPE requests a /48 or /56 via IA_PD, then subnets it into /64s for internal networks running SLAAC.
- Rapid Commit reduces latency from 4 messages to 2 but should only be used when there is a single DHCPv6 server on the link. Multiple servers with Rapid Commit can cause address conflicts.
- DHCPv6 relay agents encapsulate the full client message in a Relay-Forward wrapper (unlike DHCPv4 which uses the giaddr field). The link-address in the relay message tells the server which subnet to allocate from.
- ISC dhcpd6 is a separate daemon from dhcpd (DHCPv4). They use separate config files, lease files, and systemd units. Running both requires managing two services.
- dnsmasq combines DHCPv4, DHCPv6, RA, and DNS in one process. For simple deployments, this is far easier than running separate radvd + dhcpd6 daemons.
- When debugging, capture on port 547 (server) to see both client-to-server and relay-to-server traffic. Port 546 only shows server-to-client replies.
- The Reconfigure message (type 10) lets the server force clients to re-request configuration. This enables pushing DNS changes without waiting for lease renewal.

---

## See Also

- dhcp, ipv6, ndp, slaac

## References

- [RFC 8415 — Dynamic Host Configuration Protocol for IPv6 (DHCPv6)](https://www.rfc-editor.org/rfc/rfc8415)
- [RFC 3633 — IPv6 Prefix Options for DHCPv6](https://www.rfc-editor.org/rfc/rfc3633)
- [RFC 6355 — Definition of the UUID-Based DHCPv6 Unique Identifier (DUID-UUID)](https://www.rfc-editor.org/rfc/rfc6355)
- [RFC 3315 — DHCPv6 (original, obsoleted by 8415)](https://www.rfc-editor.org/rfc/rfc3315)
- [RFC 8106 — IPv6 RA Options for DNS Configuration](https://www.rfc-editor.org/rfc/rfc8106)
- [ISC DHCP Documentation — DHCPv6](https://kb.isc.org/docs/isc-dhcp-44-manual-pages-dhcpd6conf)
- [dnsmasq DHCPv6 Documentation](https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html)
- [Dibbler — Portable DHCPv6](https://klub.com.pl/dhcpv6/)
