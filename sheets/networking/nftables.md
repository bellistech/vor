# nftables (Modern Packet Filtering)

Successor to iptables/ip6tables/arptables/ebtables — unified framework with better syntax, atomic updates, and sets/maps.

## Tables

### Manage tables
```bash
nft list tables                          # list all tables
nft list table inet filter               # show full table with rules
nft add table inet filter                # create table (inet = IPv4+IPv6)
nft delete table inet filter             # delete table and all contents
nft flush table inet filter              # remove all rules, keep structure
```

### Table families
```bash
# ip     — IPv4 only
# ip6    — IPv6 only
# inet   — IPv4 + IPv6 (recommended)
# arp    — ARP
# bridge — bridge filtering
# netdev — ingress filtering
```

## Chains

### Create chains
```bash
# Base chain (attached to netfilter hook)
nft add chain inet filter input '{ type filter hook input priority 0; policy drop; }'
nft add chain inet filter forward '{ type filter hook forward priority 0; policy drop; }'
nft add chain inet filter output '{ type filter hook output priority 0; policy accept; }'

# NAT chains
nft add chain inet nat prerouting '{ type nat hook prerouting priority -100; }'
nft add chain inet nat postrouting '{ type nat hook postrouting priority 100; }'

# Regular chain (for jumping)
nft add chain inet filter logging
```

### Set chain policy
```bash
nft chain inet filter input '{ policy drop; }'
nft chain inet filter input '{ policy accept; }'
```

## Rules

### Add rules
```bash
nft add rule inet filter input iif lo accept
nft add rule inet filter input ct state established,related accept
nft add rule inet filter input ct state invalid drop
nft add rule inet filter input tcp dport 22 accept
nft add rule inet filter input tcp dport { 80, 443 } accept   # anonymous set
nft add rule inet filter input ip saddr 10.0.0.0/8 accept
nft add rule inet filter input icmp type echo-request accept
nft add rule inet filter input counter drop                    # count dropped
```

### Insert rules (at beginning)
```bash
nft insert rule inet filter input tcp dport 22 accept
```

### Delete rules
```bash
nft -a list chain inet filter input     # show with handles (-a)
nft delete rule inet filter input handle 7
```

### Rule with logging
```bash
nft add rule inet filter input log prefix \"dropped: \" level warn counter drop
nft add rule inet filter input tcp dport 22 log prefix \"ssh: \" accept
```

### Rate limiting
```bash
nft add rule inet filter input tcp dport 22 ct state new limit rate 3/minute accept
nft add rule inet filter input icmp type echo-request limit rate 5/second accept
```

## Sets

### Named sets
```bash
# Create a set
nft add set inet filter allowed_ips '{ type ipv4_addr; }'
nft add element inet filter allowed_ips '{ 10.0.0.1, 10.0.0.2, 10.0.0.3 }'
nft delete element inet filter allowed_ips '{ 10.0.0.3 }'

# Use in a rule
nft add rule inet filter input ip saddr @allowed_ips accept

# Set with timeout (auto-expire elements)
nft add set inet filter blocklist '{ type ipv4_addr; timeout 1h; }'
nft add element inet filter blocklist '{ 203.0.113.99 timeout 30m }'
```

### Interval sets (CIDR ranges)
```bash
nft add set inet filter internal '{ type ipv4_addr; flags interval; }'
nft add element inet filter internal '{ 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 }'
```

## Maps

### Verdict maps
```bash
nft add map inet filter port_policy '{ type inet_service : verdict; }'
nft add element inet filter port_policy '{ 22 : accept, 80 : accept, 443 : accept }'
nft add rule inet filter input tcp dport vmap @port_policy
```

## NAT

### Masquerade (dynamic SNAT)
```bash
nft add rule inet nat postrouting oif eth0 masquerade
```

### SNAT (static)
```bash
nft add rule inet nat postrouting oif eth0 snat to 203.0.113.5
```

### DNAT (port forwarding)
```bash
nft add rule inet nat prerouting tcp dport 8080 dnat to 10.0.0.2:80
nft add rule inet filter forward ip daddr 10.0.0.2 tcp dport 80 accept
```

### Redirect (local port)
```bash
nft add rule inet nat prerouting tcp dport 80 redirect to :8080
```

## Firewall Example

### Complete minimal firewall
```bash
nft flush ruleset

nft add table inet filter
nft add chain inet filter input '{ type filter hook input priority 0; policy drop; }'
nft add chain inet filter forward '{ type filter hook forward priority 0; policy drop; }'
nft add chain inet filter output '{ type filter hook output priority 0; policy accept; }'

nft add rule inet filter input iif lo accept
nft add rule inet filter input ct state established,related accept
nft add rule inet filter input ct state invalid drop
nft add rule inet filter input tcp dport { 22, 80, 443 } accept
nft add rule inet filter input icmp type echo-request accept
nft add rule inet filter input icmpv6 type { nd-neighbor-solicit, nd-router-advert, nd-neighbor-advert } accept
nft add rule inet filter input counter drop
```

## Save and Restore

### Persist rules
```bash
nft list ruleset > /etc/nftables.conf       # export full ruleset
nft -f /etc/nftables.conf                   # load from file
systemctl enable nftables                   # load on boot
```

## Migration from iptables

### Translate iptables rules
```bash
iptables-translate -A INPUT -p tcp --dport 22 -j ACCEPT
# Output: nft add rule ip filter INPUT tcp dport 22 counter accept

iptables-save | iptables-restore-translate    # translate full ruleset
```

## Tips

- Use `inet` family for dual-stack (IPv4+IPv6) in a single table
- `nft -a list ruleset` shows handles needed for deleting specific rules
- `nft flush ruleset` is atomic — no window where no rules are loaded
- Loading from a file (`nft -f`) is atomic — all rules apply at once or none do
- Sets are far more efficient than repeated rules for large IP/port lists
- Timeout sets are great for dynamic blocklists without external tools
- `nft monitor` watches for rule changes in real time
- `nft describe tcp dport` shows valid types and ranges for any selector
- iptables compatibility layer (`iptables-nft`) lets old scripts work with nftables kernel backend

## References

- [nftables Wiki](https://wiki.nftables.org/)
- [nftables Wiki — Quick Reference](https://wiki.nftables.org/wiki-nftables/index.php/Quick_reference-nftables_in_10_minutes)
- [nftables Wiki — Moving from iptables to nftables](https://wiki.nftables.org/wiki-nftables/index.php/Moving_from_iptables_to_nftables)
- [man nft](https://man7.org/linux/man-pages/man8/nft.8.html)
- [Netfilter Project — nftables](https://www.netfilter.org/projects/nftables/)
- [Linux Kernel — Netfilter Documentation](https://www.kernel.org/doc/html/latest/networking/netfilter.html)
- [Red Hat — Getting Started with nftables](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/configuring_firewalls_and_packet_filters/getting-started-with-nftables_firewall-packet-filters)
- [Debian Wiki — nftables](https://wiki.debian.org/nftables)
- [Arch Wiki — nftables](https://wiki.archlinux.org/title/Nftables)
