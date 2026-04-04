# DHCP (Dynamic Host Configuration Protocol)

Client-server protocol that automatically assigns IP addresses, subnet masks, gateways, and DNS servers to hosts on a network using a four-step DORA exchange over UDP ports 67 (server) and 68 (client).

## DORA Process

```
Client                          Server
  |                               |
  |  -- DHCPDISCOVER (broadcast) ->  |   Client broadcasts to 255.255.255.255
  |     src: 0.0.0.0:68              |   dst: 255.255.255.255:67
  |                               |
  |  <- DHCPOFFER ---------------  |   Server offers an IP address + options
  |     src: server_ip:67           |   dst: 255.255.255.255:68 (or unicast)
  |                               |
  |  -- DHCPREQUEST (broadcast) -->  |   Client requests the offered address
  |     src: 0.0.0.0:68              |   (broadcast so other servers withdraw offers)
  |                               |
  |  <- DHCPACK -----------------  |   Server confirms the lease
  |     src: server_ip:67           |
  |                               |
```

### Other Message Types

```
DHCPNAK       — Server rejects DHCPREQUEST (address no longer available)
DHCPDECLINE   — Client detected address conflict (ARP probe failed)
DHCPRELEASE   — Client voluntarily releases its lease
DHCPINFORM    — Client already has IP, just wants other config (DNS, NTP, etc.)
```

## Lease Lifecycle

```
  0%          50%              87.5%           100%
  |-----------|-----------------|----------------|
  Grant       T1 (renew)       T2 (rebind)      Expiry
              unicast to       broadcast to     Address
              original server  any server       released

  T1 = lease_time / 2          (renew timer, default)
  T2 = lease_time * 0.875      (rebind timer, default)
```

## Relay Agents

```
# DHCP broadcasts don't cross routers — relay agents forward them

Client (VLAN 10)                 Relay (Router)                 Server
  |                                |                              |
  |  -- DHCPDISCOVER (bcast) --->  |                              |
  |                                |  -- DHCPDISCOVER (unicast) -> |
  |                                |     giaddr = relay_ip         |
  |                                |     (server uses giaddr to    |
  |                                |      pick correct pool)       |
  |                                |                              |
  |                                |  <- DHCPOFFER (unicast) ---  |
  |  <- DHCPOFFER (bcast/ucast) --|                              |

# Cisco IOS relay
interface GigabitEthernet0/1
  ip helper-address 10.0.0.50

# Linux relay
dhcrelay -i eth0 10.0.0.50
```

## DHCP Options (Common)

```
Option  Name                    Description
──────────────────────────────────────────────────────────────
1       Subnet Mask             e.g., 255.255.255.0
3       Router (Default GW)     e.g., 192.168.1.1
6       DNS Servers             e.g., 8.8.8.8, 8.8.4.4
12      Hostname                Client hostname
15      Domain Name             e.g., example.com
28      Broadcast Address       e.g., 192.168.1.255
42      NTP Servers             Time server addresses
43      Vendor-Specific Info    Used by PXE boot, APs, phones
44      WINS/NBNS Servers       NetBIOS name servers
51      Lease Time              In seconds (e.g., 86400 = 24h)
53      Message Type            1=Discover, 2=Offer, 3=Request, 5=ACK, 6=NAK
55      Parameter Request List  Options the client wants
60      Vendor Class ID         e.g., "PXEClient" for PXE boot
61      Client Identifier       Unique client ID (often MAC)
66      TFTP Server Name        PXE boot server
67      Bootfile Name           PXE boot filename
82      Relay Agent Info        Sub-options: circuit-id, remote-id
121     Classless Static Routes RFC 3442 — push static routes to clients
150     TFTP Server Address     Cisco IP phone provisioning
```

## ISC DHCP Server Config

```bash
# /etc/dhcp/dhcpd.conf

# Global options
option domain-name "example.com";
option domain-name-servers 8.8.8.8, 8.8.4.4;
default-lease-time 3600;          # 1 hour
max-lease-time 86400;             # 24 hours
authoritative;                    # this server is authoritative for its subnets

# Subnet declaration
subnet 192.168.1.0 netmask 255.255.255.0 {
    range 192.168.1.100 192.168.1.200;
    option routers 192.168.1.1;
    option broadcast-address 192.168.1.255;
    option subnet-mask 255.255.255.0;
}

# Static reservation by MAC
host printer {
    hardware ethernet 00:11:22:33:44:55;
    fixed-address 192.168.1.50;
    option host-name "printer";
}

# Class-based assignment (e.g., IP phones)
class "ip-phones" {
    match if substring(option vendor-class-identifier, 0, 9) = "Cisco-i";
}
subnet 10.10.10.0 netmask 255.255.255.0 {
    pool {
        allow members of "ip-phones";
        range 10.10.10.100 10.10.10.150;
        option routers 10.10.10.1;
    }
}
```

```bash
# Start / manage ISC DHCP
systemctl start isc-dhcp-server
systemctl enable isc-dhcp-server

# Check config syntax
dhcpd -t -cf /etc/dhcp/dhcpd.conf

# View active leases
cat /var/lib/dhcp/dhcpd.leases

# Listen on specific interface
# In /etc/default/isc-dhcp-server:
INTERFACESv4="eth0"
```

## dnsmasq DHCP

```bash
# /etc/dnsmasq.conf (or /etc/dnsmasq.d/dhcp.conf)

# Enable DHCP on interface
interface=eth0
dhcp-range=192.168.1.100,192.168.1.200,255.255.255.0,12h

# Static reservation
dhcp-host=00:11:22:33:44:55,192.168.1.50,printer

# Set gateway
dhcp-option=3,192.168.1.1

# Set DNS servers
dhcp-option=6,8.8.8.8,8.8.4.4

# Set domain
domain=example.com

# PXE boot
dhcp-boot=pxelinux.0,pxeserver,192.168.1.10

# DHCP authoritative mode
dhcp-authoritative

# Log all DHCP transactions
log-dhcp
```

```bash
# Start dnsmasq
systemctl start dnsmasq

# Test config
dnsmasq --test

# View leases
cat /var/lib/misc/dnsmasq.leases
```

## dhclient (Client)

```bash
# Obtain a lease on an interface
dhclient eth0

# Release current lease
dhclient -r eth0

# Obtain lease and run in foreground (debug)
dhclient -d -v eth0

# Request specific options
# /etc/dhcp/dhclient.conf
request subnet-mask, broadcast-address, routers,
        domain-name, domain-name-servers, ntp-servers;

# Send specific hostname
send host-name "my-hostname";

# View current lease
cat /var/lib/dhcp/dhclient.leases

# NetworkManager (modern distros)
nmcli device connect eth0
nmcli connection modify eth0 ipv4.method auto
```

## DHCPv6

```bash
# DHCPv6 uses different ports: 546 (client) and 547 (server)
# Two modes: Stateful (full address assignment) and Stateless (config only)

# Router Advertisement flags control DHCPv6 behavior:
# M flag (Managed) = 1 → use DHCPv6 for addresses
# O flag (Other)   = 1 → use DHCPv6 for other config (DNS, etc.)
# A flag (Auto)    = 1 → use SLAAC for addresses

# ISC DHCPv6 server
subnet6 2001:db8:1::/64 {
    range6 2001:db8:1::100 2001:db8:1::200;
    option dhcp6.name-servers 2001:4860:4860::8888;
}

# dnsmasq DHCPv6
dhcp-range=::100,::200,constructor:eth0,ra-names,slaac,64,12h
enable-ra                          # send Router Advertisements
```

## Monitoring & Troubleshooting

```bash
# Capture DHCP traffic
tcpdump -i eth0 -n port 67 or port 68

# Watch DHCP with verbose decode
tcpdump -i eth0 -vvv -n port 67 or port 68

# Test DHCP server with nmap
nmap --script broadcast-dhcp-discover -e eth0

# Check for rogue DHCP servers
dhcping -s 255.255.255.255 -c 192.168.1.100 -h 00:11:22:33:44:55

# View systemd-networkd DHCP status
networkctl status eth0

# journalctl for DHCP events
journalctl -u isc-dhcp-server --since "1 hour ago"
journalctl -u dnsmasq --since "1 hour ago"

# Verify client got correct config
ip addr show eth0
ip route show
cat /etc/resolv.conf
```

## Tips

- Always run `dhcpd -t` to syntax-check ISC DHCP config before restarting the service. A bad config silently kills the daemon on some distros.
- DHCP relay agents insert the `giaddr` field, which the server uses to select the right pool. If clients on a remote VLAN get no offers, check that the relay is setting `giaddr` to an address within a configured subnet.
- Option 82 (Relay Agent Information) carries circuit-id and remote-id, letting the server identify which switch port the client is on. Useful for per-port policies and rogue device detection.
- Set `authoritative` in ISC DHCP so the server will NAK clients requesting addresses from wrong subnets. Without it, the server silently ignores bad requests and clients hang.
- Duplicate IP conflicts usually mean either two DHCP servers are handing out overlapping ranges, or someone has a static IP inside the DHCP range. Use `arping` to detect conflicts before assigning.
- DHCPv6 does not send a default gateway. That comes from Router Advertisements (ICMPv6 type 134). If IPv6 hosts have addresses but no route, check that RA is enabled on the router.
- dnsmasq is simpler than ISC DHCP for small deployments and integrates DNS + DHCP in one process, automatically creating DNS entries for DHCP leases.
- Lease times are a tradeoff: short leases (5-15 min) recover addresses quickly on busy guest WiFi; long leases (8-24h) reduce broadcast traffic on stable LANs.
- PXE boot requires options 66 (TFTP server) and 67 (boot filename). For UEFI clients, use a different bootfile than BIOS clients and match on option 60 vendor class.
- Watch out for Windows DHCP client behavior: it caches the lease and tries to renew its old address even after moving to a new subnet, causing delays until the NAK arrives.

## See Also

- dns, arp, ipv4, ipv6, tcpdump, dnsmasq

## References

- [RFC 2131 — Dynamic Host Configuration Protocol](https://www.rfc-editor.org/rfc/rfc2131)
- [RFC 2132 — DHCP Options and BOOTP Vendor Extensions](https://www.rfc-editor.org/rfc/rfc2132)
- [RFC 3046 — DHCP Relay Agent Information Option (Option 82)](https://www.rfc-editor.org/rfc/rfc3046)
- [RFC 3442 — Classless Static Route Option (Option 121)](https://www.rfc-editor.org/rfc/rfc3442)
- [RFC 8415 — DHCPv6](https://www.rfc-editor.org/rfc/rfc8415)
- [ISC DHCP Documentation](https://kb.isc.org/docs/isc-dhcp-44-manual-pages-dhcpdconf)
- [dnsmasq man page](https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html)
- [man dhclient](https://linux.die.net/man/8/dhclient)
