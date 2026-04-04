# NAT (Network Address Translation)

Translates private IP addresses to public addresses (and back) at a router or firewall, enabling multiple internal hosts to share limited public IPs using connection tracking and port mapping.

## NAT Types

```
Type          Direction    What Changes          Use Case
──────────────────────────────────────────────────────────────────────
SNAT          Outbound     Source IP (+ port)     Internal hosts → Internet
DNAT          Inbound      Destination IP (+ port) Internet → internal servers
Masquerade    Outbound     Source IP (dynamic)    SNAT with dynamic public IP
PAT/NAPT      Outbound     Source IP + port       Many-to-one (overload)
1:1 NAT       Both         IP only (no port)      Server with dedicated public IP
Hairpin NAT   Internal     Loopback translation   Internal access to public IP
```

## iptables NAT Rules

### SNAT (Outbound)

```bash
# Static SNAT — change source to fixed public IP
iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o eth0 \
    -j SNAT --to-source 203.0.113.10

# SNAT with port range (spread across ports)
iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o eth0 \
    -j SNAT --to-source 203.0.113.10:1024-65535

# Masquerade — auto-detect outbound IP (for dynamic IP / DHCP WAN)
iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o eth0 \
    -j MASQUERADE

# Masquerade with port range
iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o eth0 \
    -j MASQUERADE --to-ports 1024-65535
```

### DNAT (Inbound / Port Forwarding)

```bash
# Forward port 80 on public IP to internal server
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 80 \
    -j DNAT --to-destination 192.168.1.100:80

# Forward port range
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 8000:8010 \
    -j DNAT --to-destination 192.168.1.100

# Forward all traffic for a public IP (1:1 NAT)
iptables -t nat -A PREROUTING -i eth0 -d 203.0.113.20 \
    -j DNAT --to-destination 192.168.1.200

# IMPORTANT: also need FORWARD rule to allow traffic
iptables -A FORWARD -i eth0 -o eth1 -p tcp --dport 80 \
    -d 192.168.1.100 -j ACCEPT
```

### Hairpin NAT (NAT Reflection)

```bash
# Allow internal hosts to reach internal server via the public IP
# Problem: internal client → public IP → DNAT → internal server
# But the reply goes directly (no NAT), so client rejects it

# Solution: SNAT the internal traffic too
iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -d 192.168.1.100 \
    -p tcp --dport 80 -j MASQUERADE
```

### Enable IP Forwarding

```bash
# Required for any NAT/routing to work
sysctl -w net.ipv4.ip_forward=1

# Persist in /etc/sysctl.conf or /etc/sysctl.d/
echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.d/99-nat.conf
sysctl -p /etc/sysctl.d/99-nat.conf
```

## nftables NAT

```bash
# Create NAT table and chains
nft add table ip nat
nft add chain ip nat prerouting { type nat hook prerouting priority -100 \; }
nft add chain ip nat postrouting { type nat hook postrouting priority 100 \; }

# SNAT (outbound)
nft add rule ip nat postrouting oifname "eth0" ip saddr 192.168.1.0/24 \
    snat to 203.0.113.10

# Masquerade
nft add rule ip nat postrouting oifname "eth0" ip saddr 192.168.1.0/24 \
    masquerade

# DNAT (port forward)
nft add rule ip nat prerouting iifname "eth0" tcp dport 80 \
    dnat to 192.168.1.100:80

# DNAT with port redirect
nft add rule ip nat prerouting iifname "eth0" tcp dport 8080 \
    dnat to 192.168.1.100:80

# Full nftables NAT config file (/etc/nftables.conf)
table ip nat {
    chain prerouting {
        type nat hook prerouting priority -100; policy accept;
        iifname "eth0" tcp dport 80 dnat to 192.168.1.100:80
        iifname "eth0" tcp dport 443 dnat to 192.168.1.100:443
    }
    chain postrouting {
        type nat hook postrouting priority 100; policy accept;
        oifname "eth0" ip saddr 192.168.1.0/24 masquerade
    }
}
```

## Connection Tracking (conntrack)

```bash
# View active tracked connections
conntrack -L
conntrack -L -p tcp                    # TCP only
conntrack -L -p udp                    # UDP only
conntrack -L -s 192.168.1.100         # by source IP

# Count tracked connections
conntrack -C

# View conntrack table size limit
sysctl net.netfilter.nf_conntrack_max

# Set conntrack table size
sysctl -w net.netfilter.nf_conntrack_max=262144

# View current usage vs max
cat /proc/sys/net/netfilter/nf_conntrack_count
cat /proc/sys/net/netfilter/nf_conntrack_max

# Conntrack timeouts
sysctl net.netfilter.nf_conntrack_tcp_timeout_established  # default 432000 (5 days)
sysctl net.netfilter.nf_conntrack_tcp_timeout_time_wait    # default 120
sysctl net.netfilter.nf_conntrack_udp_timeout              # default 30
sysctl net.netfilter.nf_conntrack_udp_timeout_stream       # default 120

# Tune for busy NAT box
sysctl -w net.netfilter.nf_conntrack_max=524288
sysctl -w net.netfilter.nf_conntrack_tcp_timeout_established=86400
sysctl -w net.netfilter.nf_conntrack_buckets=131072

# Delete specific conntrack entries
conntrack -D -s 192.168.1.100         # delete by source
conntrack -D -p tcp --dport 80        # delete by port

# Watch conntrack events in real time
conntrack -E                           # all events
conntrack -E -p tcp                    # TCP events only
```

### Conntrack States

```
State         Description
──────────────────────────────────────────────────────────
NEW           First packet of a connection (SYN)
ESTABLISHED   Reply seen (bidirectional traffic confirmed)
RELATED       New connection related to existing one (FTP data, ICMP error)
INVALID       Packet doesn't belong to any known connection
UNTRACKED     Explicitly excluded from tracking (raw table NOTRACK)
DNAT          Destination was translated
SNAT          Source was translated
```

## Monitoring NAT

```bash
# Check NAT rules
iptables -t nat -L -n -v              # iptables
nft list table ip nat                  # nftables

# Watch NAT translations happening
conntrack -E -e NEW                   # new connections only

# Count connections per source IP (find heavy users)
conntrack -L 2>/dev/null | awk '{print $4}' | sort | uniq -c | sort -rn | head

# Check for conntrack table exhaustion
dmesg | grep "nf_conntrack: table full"

# Monitor port usage
conntrack -L -p tcp --src 192.168.1.100 2>/dev/null | wc -l
```

## Tips

- Masquerade is just SNAT that auto-detects the outgoing interface IP. Use SNAT with a fixed IP for better performance on static setups. Masquerade re-reads the interface IP for every new connection.
- When `nf_conntrack_max` is reached, new connections are silently dropped. Watch for `table full, dropping packet` in dmesg. This is the single most common cause of mysterious connection failures on busy NAT boxes.
- The conntrack hash table size (`nf_conntrack_buckets`) should be `nf_conntrack_max / 4` for optimal lookup performance. Set it early in boot via `/sys/module/nf_conntrack/parameters/hashsize`.
- DNAT only changes the destination in the PREROUTING chain. You still need a FORWARD rule to allow the traffic through, and the internal server must route replies back through the NAT box.
- Hairpin NAT (accessing your own public IP from inside the network) requires an additional SNAT/MASQUERADE rule on internal traffic so that the reply goes back through the NAT router instead of directly to the client.
- For high-performance NAT, consider nftables over iptables. nftables uses maps and sets for O(1) lookups instead of linear rule traversal, making a significant difference at scale.
- TCP connections consume conntrack entries for up to 5 days (`tcp_timeout_established=432000`). On a busy NAT box serving thousands of users, reduce this to 86400 (1 day) or less.
- Each conntrack entry uses approximately 320 bytes of kernel memory. At `nf_conntrack_max=262144`, that is about 80 MB. Plan accordingly on memory-constrained devices.
- UDP "connections" in conntrack are really just stateful expectations. The default 30-second timeout is often too short for VoIP and gaming; increase `udp_timeout_stream` to 180s for these workloads.
- Never use DNAT without rate limiting or access control. An open port forward is an invitation for port scanners. Always pair with explicit FORWARD rules limiting source networks or using connlimit.

## See Also

- iptables, nftables, ip, tcp, udp, conntrack

## References

- [RFC 3022 — Traditional IP Network Address Translator (Traditional NAT)](https://www.rfc-editor.org/rfc/rfc3022)
- [RFC 4787 — NAT Behavioral Requirements for UDP](https://www.rfc-editor.org/rfc/rfc4787)
- [RFC 5382 — NAT Behavioral Requirements for TCP](https://www.rfc-editor.org/rfc/rfc5382)
- [RFC 5389 — STUN (Session Traversal Utilities for NAT)](https://www.rfc-editor.org/rfc/rfc5389)
- [RFC 8656 — TURN (Traversal Using Relays around NAT)](https://www.rfc-editor.org/rfc/rfc8656)
- [Netfilter conntrack documentation](https://conntrack-tools.netfilter.org/manual.html)
- [nftables wiki — NAT](https://wiki.nftables.org/wiki-nftables/index.php/Performing_Network_Address_Translation_(NAT))
- [iptables-extensions man page](https://linux.die.net/man/8/iptables-extensions)
