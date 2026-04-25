# nftables (Modern Linux Packet Filter)

The successor to iptables/ip6tables/arptables/ebtables — a unified packet-filtering framework that replaces all four legacy tools with a single binary (`nft`), a single kernel subsystem (nf_tables), unified syntax, native sets/maps, faster rule loading, and atomic rule replacement.

## Setup

### Kernel and userspace

```bash
# nftables in mainline kernel since 3.13 (Jan 2014)
# default backend on Debian 10+, Ubuntu 20.04+, RHEL 8+, Fedora 32+, Arch
uname -r                                # kernel version
nft --version                           # nftables-1.0.9 (Old Doc Yak #3)
# v0.8 (2017): vmaps, sets with timeout
# v0.9 (2019): meters, dynamic sets
# v1.0 (2022): stable userspace API
```

### Install

```bash
# Debian/Ubuntu
sudo apt install nftables
sudo systemctl enable --now nftables

# RHEL/Fedora/Rocky/AlmaLinux
sudo dnf install nftables
sudo systemctl enable --now nftables

# Arch
sudo pacman -S nftables
sudo systemctl enable --now nftables

# Alpine
sudo apk add nftables
sudo rc-update add nftables
sudo rc-service nftables start
```

### Files and units

```bash
/etc/nftables.conf                      # canonical ruleset, loaded at boot
/etc/sysconfig/nftables.conf            # RHEL location (alternative)
/usr/share/nftables/                    # example rulesets
/usr/lib/systemd/system/nftables.service
/var/log/syslog or journalctl -t kernel # logged packets land here
```

### nftables.service

```bash
sudo systemctl status nftables          # check status
sudo systemctl reload nftables          # atomic reload from /etc/nftables.conf
sudo systemctl restart nftables         # full restart (flush + reload)
sudo systemctl enable nftables          # load at boot
sudo systemctl disable nftables         # don't load at boot
```

### Verify backend

```bash
sudo nft list ruleset                   # if empty, no rules loaded
sudo nft list tables                    # list every table by family
sudo iptables -V                        # iptables v1.8.9 (nf_tables) — using nftables backend
sudo iptables -V | grep -o nf_tables    # confirms nft backend underneath
ls -l /sbin/iptables                    # → /etc/alternatives/iptables → iptables-nft
```

### Conflicting tools — disable iptables-legacy

```bash
# bad: leaving iptables-legacy active alongside nftables-loaded rules
sudo iptables-legacy -L                 # if this returns rules, they coexist with nft
# fixed:
sudo update-alternatives --set iptables /usr/sbin/iptables-nft
sudo update-alternatives --set ip6tables /usr/sbin/ip6tables-nft
sudo systemctl disable --now iptables   # if package present
sudo apt purge iptables-persistent      # Debian/Ubuntu legacy persistence
```

## Why nftables vs iptables

### One binary, one syntax

```bash
# iptables (legacy): four tools, four syntaxes
iptables    -A INPUT  -p tcp --dport 22 -j ACCEPT     # IPv4
ip6tables   -A INPUT  -p tcp --dport 22 -j ACCEPT     # IPv6
arptables   -A INPUT ...                              # ARP
ebtables    -A INPUT ...                              # bridge

# nftables: one tool, one syntax, dual-stack via inet
nft add rule inet filter input tcp dport 22 accept    # IPv4 + IPv6 both
```

### Headline advantages

```bash
# 1. atomic rule loading — no half-applied state
nft -f new-rules.nft                    # entire ruleset replaced atomically

# 2. native sets — no ipset dependency
nft add set inet filter blocklist '{ type ipv4_addr; flags timeout; }'

# 3. native maps — key/value lookup baked into rules
nft add map inet filter ports '{ type inet_service : verdict; }'

# 4. faster rule loading and lookup (sets are O(log N), vmaps O(1))

# 5. cleaner syntax — fewer flags, more readable
# iptables: -A INPUT -i eth0 -p tcp ! --syn --dport 22 -j ACCEPT
# nftables: iif "eth0" tcp dport 22 tcp flags != syn accept

# 6. variables, expressions, and named objects
define WAN = "eth0"
nft add rule inet nat postrouting oifname $WAN masquerade
```

### iptables-translate compatibility

```bash
# convert single rule
iptables-translate -A INPUT -p tcp --dport 22 -j ACCEPT
# Output: nft add rule ip filter INPUT tcp dport 22 counter accept

# convert entire ruleset
iptables-save                       > rules.v4.legacy
iptables-restore-translate -f rules.v4.legacy > rules.nft
ip6tables-save                      > rules.v6.legacy
ip6tables-restore-translate -f rules.v6.legacy > rules6.nft
```

## Architecture

### Hierarchy

```bash
# table = namespace (per address-family container)
#   chain = hook into the netfilter pipeline
#     rule = match + action (verdict)
#       expression = match condition
#       statement  = side effect (accept/drop/log/counter/...)
```

### Mental model

```bash
# Tables hold chains. Chains hold rules. Rules contain expressions and a verdict.
# Each table belongs to ONE address family (ip, ip6, inet, arp, bridge, netdev).
# Each base chain attaches to ONE hook in the netfilter pipeline.
# Multiple base chains can attach to the same hook (ordered by priority).
```

### Visual

```bash
table inet filter {                       # family=inet, name=filter
  chain input {                           # base chain
    type filter hook input priority 0;    # netfilter hook: input
    policy drop;                          # default verdict if no rule matches
    ct state established,related accept   # rule (expression + verdict)
    tcp dport 22 accept                   # rule
  }
}
```

## Address Families

### Six families

```bash
# ip       — IPv4 only        (matches packets at PF_INET hooks)
# ip6      — IPv6 only        (matches packets at PF_INET6 hooks)
# inet     — IPv4 + IPv6      (dual-stack, recommended for modern firewalls)
# arp      — ARP packets      (filter ARP requests/replies on Ethernet)
# bridge   — bridge frames    (filter at the Layer 2 bridge level)
# netdev   — ingress/egress   (per-interface, very early in stack — for DDoS, XDP-style filtering)
```

### Choosing a family

```bash
# Most home/server firewalls:
nft add table inet filter               # one table covers v4 and v6

# Container / router with many bridges:
nft add table bridge filter             # plus inet filter

# Network device drop / DDoS at line rate:
nft add table netdev shield
```

### Family + hook compatibility

```bash
# ip, ip6, inet:  prerouting, input, forward, output, postrouting
# arp:            input, output
# bridge:         prerouting, input, forward, output, postrouting
# netdev:         ingress (4.2+), egress (5.16+)
```

## Chains — Filter, NAT, Route Types

### Three base-chain types

```bash
# type filter — generic packet filter (most common)
# type nat    — network address translation (only at prerouting/input/output/postrouting)
# type route  — packet rerouting/mangling (output hook only); replaces iptables mangle
```

### Filter chain example

```bash
nft add chain inet filter input \
  '{ type filter hook input priority 0; policy drop; }'
```

### NAT chain example

```bash
# DNAT (destination NAT) goes at prerouting with priority dstnat (-100)
nft add chain ip nat prerouting \
  '{ type nat hook prerouting priority dstnat; }'

# SNAT/MASQUERADE goes at postrouting with priority srcnat (100)
nft add chain ip nat postrouting \
  '{ type nat hook postrouting priority srcnat; }'
```

### Route chain example (mangle replacement)

```bash
nft add chain inet mangle output \
  '{ type route hook output priority -150; }'
nft add rule  inet mangle output ip daddr 10.0.0.0/8 meta mark set 0x1
```

### Regular chains (jump targets)

```bash
# regular chain — no type/hook/priority; only enterable via jump/goto
nft add chain inet filter ssh-rules
nft add rule  inet filter ssh-rules ct state new limit rate 4/minute accept
nft add rule  inet filter ssh-rules drop
nft add rule  inet filter input tcp dport 22 jump ssh-rules
```

## Hooks

### Where chains attach in the netfilter pipeline

```bash
# prerouting   — before routing decision (DNAT here)
# input        — packets destined for the local host
# forward      — packets transiting through the host
# output       — packets generated by the local host
# postrouting  — after routing decision (SNAT/MASQUERADE here)
# ingress      — netdev family only — earliest possible (per-interface RX)
# egress       — netdev family only — per-interface TX (5.16+)
```

### Hook diagram (text)

```bash
# Incoming packet:
#   NIC -> ingress(netdev) -> prerouting -> [routing] -> input -> local process
#                                                      \-> forward -> postrouting -> NIC
# Outgoing packet:
#   local process -> output -> [routing] -> postrouting -> egress(netdev) -> NIC
```

### Hook with no chain → no filtering

```bash
# A hook without a base chain attached does nothing. Multiple chains
# at the same hook run in priority order (lowest first).
```

## Priority

### Numeric priorities

```bash
# lower priority = runs first
# Canonical netfilter priorities (matching iptables behaviour):
#   -400  conntrack defrag (kernel internal)
#   -300  raw         (NOTRACK, conntrack helpers)
#   -225  selinux pre
#   -200  mangle      (header mangling pre-routing)
#   -150  dstnat      (DNAT — prerouting/output)
#      0  filter      (generic packet filter)
#     50  security    (selinux)
#    100  srcnat      (SNAT/MASQUERADE — postrouting/input)
#    225  selinux post
#    300  conntrack helpers
```

### Named priorities (v0.9+)

```bash
nft add chain inet filter input \
  '{ type filter hook input priority filter; policy drop; }'

nft add chain ip nat prerouting \
  '{ type nat hook prerouting priority dstnat; }'

nft add chain ip nat postrouting \
  '{ type nat hook postrouting priority srcnat; }'

nft add chain inet mangle output \
  '{ type route hook output priority mangle; }'

nft add chain inet raw prerouting \
  '{ type filter hook prerouting priority raw; }'
```

### Offsets from named priority

```bash
nft add chain inet filter input \
  '{ type filter hook input priority filter - 5; policy drop; }'  # runs before priority filter
```

## Basic Ruleset Anatomy

### Minimal default-drop firewall

```bash
table inet filter {
  chain input {
    type filter hook input priority 0; policy drop;
    ct state established,related accept
    iif "lo" accept
    tcp dport 22 accept
    log prefix "DROP: " drop
  }
  chain forward { type filter hook forward priority 0; policy drop; }
  chain output  { type filter hook output  priority 0; policy accept; }
}
```

### Anatomy

```bash
# table inet filter { ... }                       — declares table
#   chain input { ... }                           — declares base chain
#     type filter hook input priority 0;          — attaches to input hook
#     policy drop;                                — default verdict
#     ct state established,related accept         — first rule: stateful allow
#     iif "lo" accept                             — second rule: loopback
#     tcp dport 22 accept                         — third rule: SSH
#     log prefix "DROP: " drop                    — fourth rule: log + drop everything else
```

### Loading the ruleset from a file

```bash
sudo nft -f /etc/nftables.conf          # load atomically
sudo nft -c -f /etc/nftables.conf       # syntax-check only (no apply)
```

## nft Command — Common Operations

### Listing

```bash
sudo nft list ruleset                   # entire ruleset, all families
sudo nft list tables                    # just table names + families
sudo nft list table inet filter         # one table, all chains
sudo nft list chain inet filter input   # one chain
sudo nft list set   inet filter blocklist
sudo nft list map   inet filter ports
sudo nft -a list ruleset                # include rule handles (for delete)
sudo nft -nn list ruleset               # numeric (don't resolve service names)
sudo nft -j list ruleset                # JSON output
sudo nft -t list ruleset                # terse — skip set/map element listings
```

### Adding rules

```bash
sudo nft add rule inet filter input tcp dport 80 accept       # append
sudo nft insert rule inet filter input position 1 tcp dport 22 accept   # insert at top
sudo nft insert rule inet filter input tcp dport 22 accept    # without position = at top
sudo nft add rule inet filter input position 5 tcp dport 443 accept     # insert after handle 5
```

### Deleting rules (need handles)

```bash
sudo nft -a list chain inet filter input
# table inet filter {
#   chain input { # handle 1
#     tcp dport 22 accept # handle 4
#     tcp dport 80 accept # handle 5
#   }
# }
sudo nft delete rule inet filter input handle 5
```

### Replacing rules

```bash
sudo nft replace rule inet filter input handle 4 tcp dport 22 limit rate 4/minute accept
```

### Adding/deleting tables, chains, sets

```bash
sudo nft add table  inet filter
sudo nft add chain  inet filter input '{ type filter hook input priority 0; policy drop; }'
sudo nft add set    inet filter blocklist '{ type ipv4_addr; flags timeout; }'
sudo nft add map    inet filter ports     '{ type inet_service : verdict; }'

sudo nft delete chain inet filter input
sudo nft delete table inet filter         # nukes everything in the table
sudo nft flush  table inet filter         # remove rules but keep structure
sudo nft flush  ruleset                   # nuke everything (all tables, all families)
```

### Loading from file

```bash
sudo nft -f rules.nft                   # atomic apply
sudo nft -c -f rules.nft                # syntax check only
sudo nft -e -f rules.nft                # echo rules as they apply
sudo nft -i                             # interactive REPL
```

### Live monitor

```bash
sudo nft monitor                        # follow all events (rule add/del, set updates)
sudo nft monitor new rules              # only new-rule events
sudo nft monitor trace                  # packet trace (needs `meta nftrace set 1` rule)
```

## Saving and Restoring

### Dump ruleset

```bash
sudo nft list ruleset > /etc/nftables.conf
sudo nft -s list ruleset > rules.nft    # with stateless output (no counter values)
```

### Restore atomically

```bash
sudo nft -f /etc/nftables.conf
```

### Canonical /etc/nftables.conf

```bash
#!/usr/sbin/nft -f
# /etc/nftables.conf — loaded by nftables.service at boot
flush ruleset

table inet filter {
  chain input {
    type filter hook input priority 0; policy drop;
    ct state established,related accept
    ct state invalid drop
    iif "lo" accept
    icmp   type echo-request limit rate 10/second accept
    icmpv6 type echo-request limit rate 10/second accept
    tcp dport 22 ct state new limit rate 4/minute accept
    tcp dport { 80, 443 } accept
    log prefix "DROP-INPUT: " drop
  }
  chain forward { type filter hook forward priority 0; policy drop; }
  chain output  { type filter hook output  priority 0; policy accept; }
}
```

### Reload via systemctl

```bash
sudo nft -c -f /etc/nftables.conf       # always sanity-check first
sudo systemctl reload nftables          # atomic reload
```

### Persistence on Debian/Ubuntu (no extra package needed)

```bash
sudo nft list ruleset | sudo tee /etc/nftables.conf
sudo systemctl enable --now nftables
```

## Verdict Statements

### Terminal verdicts

```bash
accept                                  # let the packet through this chain
drop                                    # silently drop
reject                                  # send rejection (default: ICMP port-unreachable)
reject with icmp   type host-unreachable        # ip / inet
reject with icmp   type admin-prohibited
reject with icmpx  type port-unreachable        # works in inet (auto-picks v4 or v6)
reject with icmpv6 type admin-prohibited        # ip6 / inet
reject with tcp reset                            # TCP RST (closes the connection)
queue num 0                              # send to userspace (NFQUEUE)
queue num 0-3 fanout                     # spread across queues 0-3
```

### Non-terminating verdicts

```bash
continue                                 # implicit fall-through (rarely needed)
return                                   # return to caller chain (after jump)
jump CHAIN                               # call CHAIN; return when done
goto CHAIN                               # tail-call CHAIN; never return
```

### jump vs goto

```bash
# jump = "function call"
# After jump CHAIN finishes (without an accept/drop), control returns
# to the calling chain at the next rule.

# goto = "tail call"
# After goto CHAIN finishes, control does NOT return; if no terminal verdict
# was hit, the policy of the original chain applies.

# Canonical subchain pattern with jump:
chain web-rules {
  tcp dport 80  accept
  tcp dport 443 accept
}
chain input {
  type filter hook input priority 0; policy drop;
  ct state established,related accept
  iif "lo" accept
  jump web-rules                         # returns here if no match in web-rules
  log prefix "DROP: " drop
}
```

## Match Expressions — Layer 2

### Ethernet matching (bridge family or with ether keyword)

```bash
ether saddr 02:11:22:33:44:55 accept
ether daddr 02:11:22:33:44:55 accept
ether type ip                            # 0x0800 — IPv4
ether type ip6                           # 0x86dd — IPv6
ether type arp                           # 0x0806
ether type vlan                          # 0x8100
```

### VLAN

```bash
vlan id 100 accept
vlan tag 100 accept
vlan pcp 5 accept
```

### Bridge family example

```bash
table bridge filter {
  chain forward {
    type filter hook forward priority 0;
    ether saddr 02:de:ad:be:ef:00 drop   # block by source MAC
    vlan id != 100 drop                   # only allow VLAN 100
  }
}
```

## Match Expressions — Layer 3

### IPv4

```bash
ip saddr  192.168.1.0/24                 # source range
ip daddr  10.0.0.5
ip daddr  != 10.0.0.5                    # negation
ip saddr { 1.2.3.4, 5.6.7.8 } accept     # anonymous set
ip protocol tcp                          # tcp/udp/icmp/sctp/...
ip dscp ef                               # 0x2e
ip ecn ce                                # not-ect, ect0, ect1, ce
ip ttl < 5                               # low TTL = traceroute / loop hint
ip frag-off & 0x1fff != 0                # is-fragment
ip length > 1500
ip version 4
ip ihl 5                                 # internet header length (32-bit words)
ip checksum 0x1234                       # rarely needed — kernel validates
```

### IPv6

```bash
ip6 saddr 2001:db8::/32
ip6 daddr ::1
ip6 nexthdr tcp                          # next-header value
ip6 nexthdr ipv6-icmp                    # ICMPv6
ip6 hoplimit < 5                         # equivalent to ip ttl
ip6 flowlabel 0x12345
```

### Generic L3 in inet family

```bash
meta nfproto ipv4 ip saddr 10.0.0.0/8 accept
meta nfproto ipv6 ip6 saddr 2001:db8::/32 accept
meta l4proto tcp tcp dport 22 accept     # works for both v4 and v6
```

## Match Expressions — Layer 4

### TCP

```bash
tcp sport 1024-65535
tcp dport 22
tcp dport != 22
tcp dport { 80, 443, 8080 }              # anonymous set
tcp dport 80-89                           # range
tcp flags syn                             # only SYN set
tcp flags syn,ack                         # both SYN and ACK
tcp flags & (syn|ack) == syn             # mask + comparison
tcp flags != syn                          # NOT SYN-only
tcp window > 0
tcp option mss 1460
tcp option sack-permitted exists
```

### UDP

```bash
udp sport 53
udp dport { 53, 67, 68, 123 }
udp length 0-512
```

### SCTP

```bash
sctp dport 36412
sctp chunk init exists
```

### ICMP

```bash
icmp type echo-request                    # IPv4 only
icmp type { echo-request, echo-reply, destination-unreachable }
icmp code 3                               # numeric code
icmp id 1234
icmp sequence 1
```

### ICMPv6

```bash
icmpv6 type echo-request
icmpv6 type { echo-request, echo-reply, nd-router-solicit, nd-router-advert, \
              nd-neighbor-solicit, nd-neighbor-advert, mld-listener-query }
```

### ICMPx (ip-version-agnostic in inet family)

```bash
meta l4proto { icmp, ipv6-icmp } icmpx type echo-request accept
```

## Match Expressions — Connection Tracking

### ct state

```bash
ct state new                              # new flow (no prior packet)
ct state established                      # part of an existing flow
ct state related                          # ICMP error or expected child (FTP data)
ct state invalid                          # malformed / out-of-window
ct state untracked                        # marked NOTRACK in raw
ct state new,established,related accept   # combined
```

### Canonical first rule (huge perf win)

```bash
# Match the bulk of packets early so they skip the rest of the chain
ct state established,related accept
ct state invalid drop
```

### ct status

```bash
ct status assured                         # conntrack saw bidirectional traffic
ct status seen-reply                      # at least one reply packet
ct status confirmed                       # entry committed to conntrack table
ct status nat                             # NAT was applied
ct status snat
ct status dnat
```

### ct mark / ct label

```bash
ct mark 0x1                               # match conntrack mark
ct mark set 0x1                           # set conntrack mark
ct label set web                          # symbolic label (defined elsewhere)
ct label web                              # match label
```

### ct expiration / count

```bash
ct expiration < 60s                       # close to expiring
ct count 100                              # match if conntrack has >= 100 entries for tuple
```

### ct helper (FTP, SIP, ...)

```bash
ct helper "ftp"                           # matches connections being tracked by ftp helper
ct helper set "ftp"                       # assign ftp helper to this flow
```

### Conntrack zones

```bash
ct zone 5 accept                          # match flow in conntrack zone 5
ct zone set 5                             # tag flow into a conntrack zone (multi-tenant NAT)
```

## Match Expressions — Other

### meta — interface, mark, protocol, etc.

```bash
meta iif "eth0"                           # input interface (numeric index, fast)
meta oif "eth0"                           # output interface
meta iifname "eth0"                       # by name (slower; works after rename)
meta oifname "wg*"                        # wildcard (interval set under the hood)
meta iiftype ether                        # interface type
meta protocol ip                          # ip / ip6 / arp / vlan
meta nfproto ipv4                         # netfilter protocol (good with inet)
meta l4proto tcp                          # transport protocol (best in inet)
meta mark 0x1                             # nfmark
meta mark set 0x1                         # set nfmark
meta priority 0x1:0x2                     # tc class
meta cgroup 0x10001                       # match cgroup id (cgroupv1)
meta cpu 0                                # current CPU
meta skuid 1000                           # match owning user UID (output chain)
meta skgid 1000                           # match owning group GID
meta secmark 1                            # SELinux secmark
meta time "2024-12-31T00:00:00"           # absolute time match
meta day "Saturday"                       # weekday match
meta hour "08:00"-"17:00"                 # business hours
```

### rt — routing info

```bash
rt classid "ssh"                          # match if route uses this realm
rt nexthop 10.0.0.1                       # route nexthop
rt mtu 1500
rt ipsec exists                           # packet went through xfrm/IPsec
```

### socket — match by socket existence

```bash
socket transparent 1                      # tproxy transparent socket
socket cgroupv2 level 2 "system.slice/sshd.service"
```

### numgen — stateless load-balance

```bash
numgen inc mod 5 vmap {                   # round-robin across 5 backends
  0 : jump backend0,
  1 : jump backend1,
  2 : jump backend2,
  3 : jump backend3,
  4 : jump backend4
}
numgen random mod 100 < 50 accept         # 50% sampling
```

### hash — consistent hashing

```bash
ip saddr . ip daddr . tcp sport . tcp dport . meta l4proto \
  hash mod 4 vmap { 0 : jump be0, 1 : jump be1, 2 : jump be2, 3 : jump be3 }
```

### fib — forwarding info base

```bash
fib saddr type local accept                # source matches local IP
fib daddr type local accept                # packet for us
fib saddr . iif oif missing drop           # uRPF strict-mode anti-spoof
```

## Sets — Anonymous

### Inline anonymous sets

```bash
tcp dport { 22, 80, 443 } accept                            # OR-match
ip saddr { 10.0.0.1, 10.0.0.2, 192.168.0.0/16 } accept      # mix host + CIDR
tcp dport != { 22, 23 } accept                              # negation
tcp dport { 80-89, 443, 8080-8089 } accept                  # ranges (need flags interval impl)
```

### Set element types are inferred

```bash
# tcp dport {...}     -> inet_service
# ip  saddr {...}     -> ipv4_addr
# ip6 saddr {...}     -> ipv6_addr
# ether saddr {...}   -> ether_addr
# meta iifname {...}  -> ifname
```

### Mixing types via concat

```bash
ip saddr . tcp dport { 10.0.0.1 . 22, 10.0.0.2 . 80 } accept
```

## Sets — Named

### Static set

```bash
nft add set inet filter trusted_v4 '{ type ipv4_addr; }'
nft add element inet filter trusted_v4 '{ 1.2.3.4, 5.6.7.0/24 }'    # needs `flags interval` for /24
```

### CIDR sets need interval flag

```bash
nft add set inet filter cidr_set '{ type ipv4_addr; flags interval; }'
nft add element inet filter cidr_set '{ 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 }'
nft add rule inet filter input ip saddr @cidr_set accept
```

### Dynamic timeout set (replaces ipset)

```bash
nft add set inet filter fail2ban '{
  type ipv4_addr;
  flags timeout;
  timeout 1h;
  size 65536;
}'
# add element with explicit timeout:
nft add element inet filter fail2ban '{ 1.2.3.4 timeout 30m }'
# reference in rules:
nft add rule inet filter input ip saddr @fail2ban drop
# self-populating from a rule (very common):
nft add rule inet filter input tcp dport 22 ct state new \
  add @fail2ban { ip saddr timeout 1h limit rate 5/minute } accept
```

### Concat sets (multi-key)

```bash
nft add set inet filter allow_pair '{
  type ipv4_addr . inet_service;
  flags interval;
}'
nft add element inet filter allow_pair '{ 10.0.0.1 . 22, 10.0.0.0/24 . 80 }'
nft add rule inet filter input ip saddr . tcp dport @allow_pair accept
```

### Set listing and elements

```bash
nft list set inet filter fail2ban
nft list sets
nft delete element inet filter fail2ban '{ 1.2.3.4 }'
nft flush set inet filter fail2ban
```

### Set flags reference

```bash
# constant   — set elements never change after creation (compiled at load)
# dynamic    — elements can be added by rules at runtime
# interval   — supports ranges and CIDR (uses interval tree)
# timeout    — elements expire
```

### Set sizes and performance

```bash
# size N is mandatory for dynamic / timeout sets
nft add set inet filter big '{
  type ipv4_addr;
  flags timeout;
  timeout 5m;
  size 1000000;                           # cap — overflow returns ENOSPC
}'
```

## Maps

### Static map

```bash
nft add map inet filter dst_to_port '{
  type ipv4_addr : inet_service;
}'
nft add element inet filter dst_to_port '{
  10.0.0.1 : 80,
  10.0.0.2 : 8080,
  10.0.0.3 : 443
}'
# use in DNAT (look up port by source IP):
nft add rule inet nat prerouting iif "eth0" \
  dnat ip to ip saddr map @dst_to_port
```

### IP-to-IP map (DNAT redirection)

```bash
nft add map inet nat external_to_internal '{
  type ipv4_addr : ipv4_addr;
}'
nft add element inet nat external_to_internal '{
  203.0.113.10 : 10.0.0.10,
  203.0.113.11 : 10.0.0.11
}'
nft add rule inet nat prerouting \
  dnat to ip daddr map @external_to_internal
```

### Concat map keys

```bash
nft add map inet nat fanout '{
  type ipv4_addr . inet_service : ipv4_addr . inet_service;
}'
nft add element inet nat fanout '{
  203.0.113.10 . 443 : 10.0.0.10 . 8443,
  203.0.113.10 . 80  : 10.0.0.11 . 8080
}'
nft add rule inet nat prerouting \
  dnat ip to ip daddr . tcp dport map @fanout
```

## Verdict Maps

### vmap basics (since v0.7)

```bash
# A verdict map (vmap) maps keys to verdicts. Use it to dispatch
# without long if-else chains.
tcp dport vmap { 22 : accept, 80 : jump web, 443 : accept, 25 : drop }

# Equivalent to:
#   tcp dport 22  accept
#   tcp dport 80  jump web
#   tcp dport 443 accept
#   tcp dport 25  drop
# but O(1) instead of O(N).
```

### Named vmap

```bash
nft add map inet filter port_actions '{
  type inet_service : verdict;
}'
nft add element inet filter port_actions '{
  22  : accept,
  80  : jump web_rules,
  443 : jump web_rules,
  25  : drop
}'
nft add rule inet filter input tcp dport vmap @port_actions
```

### Dispatch by ct state

```bash
ct state vmap {
  established : accept,
  related     : accept,
  invalid     : drop,
  new         : jump new_traffic
}
```

### Dispatch by interface

```bash
iifname vmap {
  "lo"   : accept,
  "wg0"  : jump vpn,
  "eth0" : jump wan,
  "eth1" : jump lan
}
```

## NAT — DNAT

### Single-port forward

```bash
table ip nat {
  chain prerouting {
    type nat hook prerouting priority dstnat;
    iif "eth0" tcp dport 80 dnat to 192.168.1.10:80
  }
  chain postrouting {
    type nat hook postrouting priority srcnat;
    oif "eth0" masquerade
  }
}
# also need:
nft add rule ip filter forward iif "eth0" oif "eth1" \
  ip daddr 192.168.1.10 tcp dport 80 ct state new accept
```

### Multi-port + IP via map

```bash
nft add map ip nat web_backends '{
  type inet_service : ipv4_addr . inet_service;
  elements = {
    80   : 10.0.0.10 . 8080,
    443  : 10.0.0.10 . 8443
  };
}'
nft add rule ip nat prerouting iif "eth0" \
  dnat ip to tcp dport map @web_backends
```

### Round-robin DNAT (load balancing)

```bash
nft add rule ip nat prerouting tcp dport 80 \
  dnat to numgen inc mod 3 map { 0 : 10.0.0.10, 1 : 10.0.0.11, 2 : 10.0.0.12 }
```

### Persistent (sticky) load-balance

```bash
nft add rule ip nat prerouting tcp dport 80 \
  dnat to jhash ip saddr mod 3 map { 0 : 10.0.0.10, 1 : 10.0.0.11, 2 : 10.0.0.12 }
```

### Required forward rule

```bash
# DNAT rewrites destination but does NOT bypass forward chain.
# Add explicit accept for the post-DNAT destination:
nft add rule ip filter forward ip daddr 192.168.1.10 tcp dport 80 \
  ct state new,established,related accept
```

## NAT — SNAT/MASQUERADE

### Masquerade (use interface IP, even if it changes)

```bash
table ip nat {
  chain postrouting {
    type nat hook postrouting priority srcnat;
    oifname "eth0" masquerade
  }
}
```

### SNAT to fixed public IP

```bash
nft add rule ip nat postrouting oifname "eth0" snat to 203.0.113.5
```

### SNAT to a pool

```bash
nft add rule ip nat postrouting oifname "eth0" \
  snat to 203.0.113.5-203.0.113.10
```

### Per-source SNAT via map

```bash
nft add map ip nat src_pool '{
  type ipv4_addr : ipv4_addr;
  flags interval;
  elements = {
    10.0.1.0/24 : 203.0.113.5,
    10.0.2.0/24 : 203.0.113.6
  };
}'
nft add rule ip nat postrouting oifname "eth0" \
  snat to ip saddr map @src_pool
```

### Required: enable IP forwarding

```bash
echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward                  # IPv4
echo 1 | sudo tee /proc/sys/net/ipv6/conf/all/forwarding          # IPv6
# persist:
sudo tee /etc/sysctl.d/99-router.conf <<EOF
net.ipv4.ip_forward=1
net.ipv6.conf.all.forwarding=1
EOF
sudo sysctl --system
```

## Counter and Quota

### Inline counter

```bash
nft add rule inet filter input tcp dport 22 counter accept
nft list ruleset                          # shows packets/bytes counters
```

### Named counter (reusable)

```bash
nft add counter inet filter ssh_in
nft add rule inet filter input tcp dport 22 counter name "ssh_in" accept
nft list counter inet filter ssh_in
nft reset counter inet filter ssh_in      # zero it
```

### Quota (volume cap)

```bash
nft add quota inet filter monthly_cap '{ 1 gbytes }'
nft add rule inet filter forward quota name "monthly_cap" accept
# when exceeded, the rule no longer matches. To drop instead:
nft add rule inet filter forward quota over 1 gbytes drop
nft add rule inet filter forward quota until 100 mbytes accept
```

### Reset counters

```bash
nft reset counters inet filter            # all counters in this table
nft reset rules inet filter input         # all rule counters in this chain
```

## Limit

### Rate limit (per match)

```bash
limit rate 5/minute             # absolute rate
limit rate 5/minute burst 10 packets   # allow 10-packet burst
limit rate 1/second
limit rate over 100/second drop  # drop ABOVE rate (i.e. policer)
limit rate 1 mbytes/second       # byte rate
```

### Canonical SSH brute-force protector

```bash
nft add rule inet filter input tcp dport 22 ct state new \
  limit rate 4/minute accept
nft add rule inet filter input tcp dport 22 ct state new \
  log prefix "SSH-FLOOD: " drop
```

### Per-source rate limit (meter — kernel 4.3+; deprecated 5.6+, use dynamic set with limit)

```bash
nft add rule inet filter input tcp dport 22 ct state new \
  meter ssh_meter { ip saddr limit rate 4/minute } accept
nft add rule inet filter input tcp dport 22 ct state new drop
```

### Per-source via dynamic set (modern)

```bash
nft add set inet filter ssh_throttle '{
  type ipv4_addr;
  flags dynamic, timeout;
  timeout 1m;
  size 65535;
}'
nft add rule inet filter input tcp dport 22 ct state new \
  add @ssh_throttle { ip saddr limit rate over 4/minute } drop
nft add rule inet filter input tcp dport 22 accept
```

## Logging

### Basic log

```bash
log                                       # log all matched packets
log prefix "DROP-INPUT: "
log prefix "DROP-INPUT: " level warn
log level info
log flags ip options                      # include IP options dump
log flags tcp sequence,options
log flags all                             # everything kernel knows
```

### Log levels

```bash
# emerg alert crit err warn notice info debug audit
log level emerg                           # most severe
log level debug                           # most verbose
```

### Combined log + drop in same rule

```bash
nft add rule inet filter input log prefix "INPUT-DROP: " level info counter drop
```

### Log to a specific group via NFLOG (efficient bulk logging)

```bash
nft add rule inet filter input log group 0
# then read with ulogd2 / tcpdump -i nflog:0
```

### Disable kernel rate-limit on logs

```bash
sudo sysctl -w net.netfilter.nf_log_all_netns=1
sudo sysctl -w net.core.message_burst=10
sudo sysctl -w net.core.message_cost=0
# OR per-rule with limit:
nft add rule inet filter input log prefix "X: " limit rate 10/second drop
```

### View logs

```bash
journalctl -k -f | grep "INPUT-DROP:"     # journald (most distros)
tail -f /var/log/kern.log | grep "INPUT-DROP:"   # rsyslog
dmesg -wT | grep "INPUT-DROP:"
```

## Common Workflows

### Default-drop firewall (template)

```bash
#!/usr/sbin/nft -f
flush ruleset

table inet filter {
  chain input {
    type filter hook input priority 0; policy drop;
    ct state vmap { established : accept, related : accept, invalid : drop }
    iif "lo" accept
    meta l4proto { icmp, ipv6-icmp } accept
    tcp dport 22 ct state new limit rate 4/minute accept
    tcp dport { 80, 443 } accept
    log prefix "INPUT-DROP: " drop
  }
  chain forward { type filter hook forward priority 0; policy drop; }
  chain output  { type filter hook output  priority 0; policy accept; }
}
```

### Internet gateway (router with NAT)

```bash
#!/usr/sbin/nft -f
flush ruleset

define WAN = "eth0"
define LAN = "eth1"

table inet filter {
  chain input {
    type filter hook input priority 0; policy drop;
    ct state established,related accept
    iif "lo" accept
    iif $LAN accept
    iif $WAN icmp type echo-request limit rate 5/second accept
    iif $WAN tcp dport 22 ct state new limit rate 4/minute accept
    log prefix "INPUT-DROP: " drop
  }
  chain forward {
    type filter hook forward priority 0; policy drop;
    ct state established,related accept
    iifname $LAN oifname $WAN accept
    iifname $WAN ct state new drop
    log prefix "FORWARD-DROP: " drop
  }
  chain output { type filter hook output priority 0; policy accept; }
}

table ip nat {
  chain prerouting  { type nat hook prerouting  priority dstnat; }
  chain postrouting {
    type nat hook postrouting priority srcnat;
    oifname $WAN masquerade
  }
}
```

### Port forward to internal host

```bash
table ip nat {
  chain prerouting {
    type nat hook prerouting priority dstnat;
    iifname "eth0" tcp dport 8080 dnat to 192.168.1.10:80
  }
  chain postrouting {
    type nat hook postrouting priority srcnat;
    oifname "eth0" masquerade
  }
}

table inet filter {
  chain forward {
    type filter hook forward priority 0; policy drop;
    ct state established,related accept
    iifname "eth0" oifname "eth1" ip daddr 192.168.1.10 tcp dport 80 accept
  }
}
```

### Dynamic SSH brute-force blacklist

```bash
table inet filter {
  set ssh_blocked {
    type ipv4_addr
    flags dynamic, timeout
    timeout 1h
    size 65536
  }
  chain input {
    type filter hook input priority 0; policy drop;
    ct state established,related accept
    iif "lo" accept
    ip saddr @ssh_blocked drop
    tcp dport 22 ct state new \
      add @ssh_blocked { ip saddr limit rate over 4/minute } \
      log prefix "SSH-BAN: " drop
    tcp dport 22 ct state new accept
    log prefix "INPUT-DROP: " drop
  }
}
```

### Mark + tc traffic shaping integration

```bash
nft add table inet mangle
nft add chain inet mangle output '{ type route hook output priority -150; }'
nft add rule  inet mangle output ip daddr 10.0.0.0/8 meta mark set 0x1
# then:
sudo tc qdisc add dev eth0 root handle 1: htb default 30
sudo tc filter add dev eth0 parent 1: handle 1 fw flowid 1:10
```

### Container host with bridge + host firewall

```bash
table inet filter {
  chain input {
    type filter hook input priority 0; policy drop;
    ct state established,related accept
    iif "lo" accept
    iifname "docker0" accept
    iifname "br-*"    accept
    tcp dport 22 accept
    drop
  }
  chain forward {
    type filter hook forward priority 0; policy accept;
    # Docker manages its own forward chain at higher priority
  }
}
```

## Migration from iptables

### Side-by-side translation

```bash
# iptables                                                nftables (inet family)
# iptables -A INPUT -i lo -j ACCEPT                       iif "lo" accept
# iptables -A INPUT -p tcp --dport 22 -j ACCEPT           tcp dport 22 accept
# iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
#                                                          ct state established,related accept
# iptables -A INPUT -p icmp --icmp-type echo-request -j ACCEPT
#                                                          icmp type echo-request accept
# iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE    oifname "eth0" masquerade
# iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to-destination 10.0.0.10:80
#                                                          tcp dport 80 dnat to 10.0.0.10:80
# iptables -A INPUT -p tcp -m multiport --dports 80,443 -j ACCEPT
#                                                          tcp dport { 80, 443 } accept
# iptables -A INPUT -m conntrack --ctstate INVALID -j DROP
#                                                          ct state invalid drop
# iptables -A INPUT -m limit --limit 5/min -j ACCEPT      limit rate 5/minute accept
# iptables -A INPUT -j LOG --log-prefix "X "              log prefix "X "
# iptables -A INPUT -j REJECT --reject-with icmp-port-unreachable
#                                                          reject with icmpx type port-unreachable
```

### Translate an existing legacy ruleset

```bash
# Single-rule translation (interactive)
iptables-translate -A INPUT -p tcp --dport 22 -j ACCEPT
# nft add rule ip filter INPUT tcp dport 22 counter accept

# Bulk translate v4 + v6
iptables-save  > /tmp/rules.v4
ip6tables-save > /tmp/rules.v6
iptables-restore-translate  -f /tmp/rules.v4 > /tmp/rules.v4.nft
ip6tables-restore-translate -f /tmp/rules.v6 > /tmp/rules.v6.nft
```

### Canonical migration steps

```bash
# 1. Translate
iptables-save  | iptables-restore-translate  -f /dev/stdin > rules.v4.nft
ip6tables-save | ip6tables-restore-translate -f /dev/stdin > rules.v6.nft

# 2. Combine + clean — usually merge to single inet table
$EDITOR rules.v4.nft rules.v6.nft

# 3. Sanity-check syntax without applying
sudo nft -c -f rules.v4.nft

# 4. Atomic apply
sudo nft flush ruleset
sudo nft -f rules.v4.nft

# 5. Save
sudo nft list ruleset | sudo tee /etc/nftables.conf
sudo systemctl enable --now nftables
```

### Fall back to iptables-legacy if needed

```bash
sudo update-alternatives --set iptables  /usr/sbin/iptables-legacy
sudo update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy
```

## Atomic Updates

### Why atomic matters

```bash
# With iptables -F + iptables-restore, there is a window where the
# old ruleset is gone but the new one isn't loaded. Connections die.
# With `nft -f`, the entire ruleset swap is one transaction.
```

### Safe edit-test-load loop

```bash
sudo nft list ruleset > /tmp/current.nft
cp /tmp/current.nft /tmp/new.nft
$EDITOR /tmp/new.nft
sudo nft -c -f /tmp/new.nft               # syntax check only
sudo nft -f  /tmp/new.nft                 # atomic apply
```

### Roll forward via systemd reload

```bash
sudoedit /etc/nftables.conf
sudo nft -c -f /etc/nftables.conf
sudo systemctl reload nftables             # atomic; no downtime
```

### Locking yourself out — use a "lockout test"

```bash
# Schedule a flush in 60s; only cancel if you can still reach the box
sudo bash -c '(sleep 60 && nft flush ruleset) & echo $! > /tmp/rb.pid'
sudo nft -f /tmp/new.nft
# if box still reachable:
sudo kill "$(cat /tmp/rb.pid)"            # cancel the flush
```

## Common Errors and Fixes

### Exact text → cause → fix

```bash
# Error: syntax error, unexpected newline, expecting string
#   → likely a missing keyword or stray semicolon. Run `nft -c -f` and read the line/column.

# Error: Could not process rule: No such file or directory
#   → table or chain referenced doesn't exist yet. Add it first:
#     nft add table inet filter; nft add chain inet filter input '{ type ... }'

# Error: Could not process rule: Operation not supported
#   → kernel module missing (e.g. nf_tables, nf_nat, nft_ct).
#     Run `dmesg | tail`; on minimal/embedded kernels the module isn't built.

# Error: Could not process rule: Numerical result out of range
#   → set/map full or too many rules. Increase `size`:
#     nft add set inet filter big '{ type ipv4_addr; flags timeout; size 1000000; }'

# Error: 'invalid' is invalid
#   → typo in keyword (probably "ct state invalid" written wrong). Check spelling.

# Error: conflicting protocols specified: ip vs ip6
#   → can't mix `ip saddr` with `ip6 saddr` in a single rule unless using inet family
#     and then matching `meta nfproto` first.

# Error: Could not process rule: Device or resource busy
#   → table/chain in use. Flush before deleting:
#     nft flush table inet filter; nft delete table inet filter

# Error: Could not process rule: Address already in use
#   → a base chain at the same priority already exists for that hook.

# Error: Could not process rule: Permission denied
#   → not running as root or missing CAP_NET_ADMIN.

# Error: Set has no elements
#   → trying to add element to a set with no `flags interval` when using a CIDR. Add `flags interval`.

# Error: invalid hook
#   → the family doesn't support that hook (e.g. arp doesn't have forward).

# Error: Could not process rule: Cannot allocate memory
#   → kernel ran out of memory loading the ruleset. Trim sets or raise vm limits.

# Warning: nft list ruleset > rules.nft creates "table ip filter" not "table inet filter"
#   → this is correct: nftables preserves whatever family you used originally.
```

### Diagnostics

```bash
sudo nft -c -f /etc/nftables.conf 2>&1 | less   # check syntax with full path
sudo nft monitor                                  # see live add/del events
sudo journalctl -k -f                             # kernel log for module errors
sudo dmesg | tail -50                             # kernel module load failures
sudo modprobe nf_tables nf_tables_inet nf_nat nft_chain_nat
```

## Common Gotchas

### Old iptables rules still loaded

```bash
# bad: nftables loaded but iptables-legacy rules also active
sudo iptables-legacy -L                           # if non-empty, double-firewall
# fixed:
sudo iptables-legacy -F
sudo update-alternatives --set iptables /usr/sbin/iptables-nft
sudo apt-mark hold iptables-persistent || sudo apt purge iptables-persistent
```

### Wrong NAT chain priority

```bash
# bad — NAT chain at priority filter (0) — runs AFTER routing decision
nft add chain ip nat prerouting '{ type nat hook prerouting priority 0; }'
# fixed — DNAT must be at priority dstnat (-100)
nft add chain ip nat prerouting '{ type nat hook prerouting priority dstnat; }'
nft add chain ip nat postrouting '{ type nat hook postrouting priority srcnat; }'
```

### Skipping syntax check

```bash
# bad — edit /etc/nftables.conf, reload, lock self out
sudoedit /etc/nftables.conf
sudo systemctl reload nftables       # error halfway, partial state? (atomic, but ruleset still wrong)
# fixed — always sanity-check first
sudo nft -c -f /etc/nftables.conf && sudo systemctl reload nftables
```

### Assuming inet supports ARP

```bash
# bad — ARP rule in inet family
nft add rule inet filter input arp operation request drop
# Error: ARP only available in arp family
# fixed — use the arp family
nft add table arp filter
nft add chain arp filter input '{ type filter hook input priority 0; }'
nft add rule  arp filter input arp operation request drop
```

### iptables -p vs nftables protocol matching

```bash
# bad — copy-pasting `-p tcp` style
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
# fixed in nft (no separate `-p`):
nft add rule inet filter input tcp dport 22 accept
# (or `meta l4proto tcp tcp dport 22` if you want explicit, but the bare `tcp dport` already implies tcp)
```

### Forgetting policy drop for tight firewalls

```bash
# bad — chain default policy is accept; forgetting to set drop leaves everything open
nft add chain inet filter input '{ type filter hook input priority 0; }'
# fixed — explicit policy drop
nft add chain inet filter input '{ type filter hook input priority 0; policy drop; }'
```

### Forgetting forward rule with DNAT

```bash
# bad — DNAT works, but packet still gets policy-dropped in forward chain
nft add rule ip nat prerouting tcp dport 80 dnat to 192.168.1.10:80
# (forward chain has policy drop)
# fixed — explicit forward accept
nft add rule inet filter forward iifname "eth0" oifname "eth1" \
  ip daddr 192.168.1.10 tcp dport 80 ct state new accept
```

### Wrong iif/iifname distinction

```bash
# bad — using iifname when interface is renamed; iif uses index
# fixed — iif is fastest (kernel index), iifname is name-based:
iif "eth0"            # by interface index — fastest
iifname "eth0"        # by name — survives renumbering, supports wildcards
iifname "wg*"         # wildcard — only works with iifname (not iif)
```

### Hardcoded interface names

```bash
# bad
nft add rule ip nat postrouting oif "eth0" masquerade   # eth0 may not be your WAN
# fixed — use a defined variable
define WAN = "ens3"
nft add rule ip nat postrouting oifname $WAN masquerade
```

### Counter not incrementing

```bash
# bad — the rule never matches; counter stays at 0
# fixed — `nft -a list ruleset` to see handles, then look at neighboring rules
sudo nft -a list ruleset
# (move the rule earlier or check that conditions actually fire)
```

### Confusing accept/drop with iptables ACCEPT/DROP

```bash
# bad — RETURN equivalent
nft add rule inet filter input tcp dport 22 accept    # terminates this chain
# In a regular chain, `return` returns to caller.
# In a base chain, `accept` lets the packet through that base chain;
# but a packet still has to traverse other base chains at higher priority.
```

### Set without size = unbounded growth

```bash
# bad — dynamic set with no size cap
nft add set inet filter blocked '{ type ipv4_addr; flags dynamic, timeout; timeout 1h; }'
# Eventually exhausts memory.
# fixed — always cap with size
nft add set inet filter blocked '{ type ipv4_addr; flags dynamic, timeout; timeout 1h; size 65536; }'
```

### Mixing ip and ip6 rules in inet without nfproto guard

```bash
# bad — ip saddr in inet family on an IPv6 packet → no match (silent)
nft add rule inet filter input ip saddr 10.0.0.0/8 accept
# fixed — guard with nfproto
nft add rule inet filter input meta nfproto ipv4 ip  saddr 10.0.0.0/8 accept
nft add rule inet filter input meta nfproto ipv6 ip6 saddr 2001:db8::/32 accept
```

### Forgetting to enable IP forwarding for routing

```bash
# bad — wrote a beautiful NAT ruleset; nothing forwards
# fixed:
echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward
echo 1 | sudo tee /proc/sys/net/ipv6/conf/all/forwarding
```

### Reload doesn't reset counters

```bash
# bad — wondered why old packet counts persisted across `systemctl reload`
# fixed — reload preserves counters by design (atomic). Use:
sudo nft reset counters
```

## Performance Tips

### Sets are O(log N) for ranges, O(1) for hashes

```bash
# Use sets — not chains of rules — for large IP/port lists.
# 100 IPs across 100 rules = 100 rule evaluations.
# 100 IPs in one set = single set lookup.

# Worst (slow):
nft add rule inet filter input ip saddr 1.2.3.1 accept
nft add rule inet filter input ip saddr 1.2.3.2 accept
# ... 99 more ...

# Best (fast):
nft add set inet filter allowed '{ type ipv4_addr; }'
nft add element inet filter allowed '{ 1.2.3.1, 1.2.3.2, ... }'
nft add rule inet filter input ip saddr @allowed accept
```

### Verdict maps for dispatch

```bash
# Worst:
nft add rule inet filter input tcp dport 22 jump ssh
nft add rule inet filter input tcp dport 80 jump web
nft add rule inet filter input tcp dport 443 jump web
# ...

# Best:
nft add rule inet filter input tcp dport vmap { \
  22 : jump ssh, 80 : jump web, 443 : jump web }
```

### Fast-track common case to top of chain

```bash
# put `ct state established,related accept` FIRST so the bulk of packets
# hit it and skip the rest of the chain
chain input {
  type filter hook input priority 0; policy drop;
  ct state established,related accept   # 99% of traffic returns here
  ct state invalid drop
  ...
}
```

### NOTRACK for stateless flows (raw chain)

```bash
table ip raw {
  chain prerouting {
    type filter hook prerouting priority raw;
    udp dport 53 notrack
    ip protocol icmp notrack
  }
}
# Skips conntrack lookup; great for high-rate stateless DNS / ICMP.
```

### Flowtable (offload established flows — kernel 4.16+)

```bash
table inet filter {
  flowtable f {
    hook ingress priority 0;
    devices = { eth0, eth1 };
  }
  chain forward {
    type filter hook forward priority 0;
    ip protocol { tcp, udp } flow add @f
    ct state established,related accept
  }
}
# Hardware offload (5.13+):
#   flowtable f { hook ingress priority 0; devices = { eth0, eth1 }; flags offload; }
```

### Use `meta l4proto` over `ip protocol` for inet

```bash
# Worst — only works in ip family
ip protocol tcp tcp dport 80 accept

# Best — inet-friendly, works for v4 and v6 with one rule
meta l4proto tcp tcp dport 80 accept
```

### Concat indexes are fast

```bash
# 1M-element concat set lookup is microseconds — much better than a scan.
nft add set inet filter authz '{
  type ipv4_addr . inet_service;
  size 1000000;
}'
```

### Avoid logging in hot path

```bash
# Logging takes a syscall and serializes through klogd/journald.
# Either rate-limit or use NFLOG group + ulogd2.
log prefix "X: " limit rate 10/second drop
```

### "10x faster than iptables for large rulesets"

```bash
# nf_tables uses pseudo-code interpretation in the kernel, batched syscalls,
# native sets/maps, and atomic updates. For rulesets > 1000 rules,
# benchmarks routinely show 5-10x faster rule eval and 100x faster reload.
```

## Idioms

### inet table for everything

```bash
# Modern default: one inet table covers IPv4 and IPv6. Two-thirds the rules.
table inet filter { ... }
```

### Verdict-map dispatch on port/iface

```bash
tcp dport vmap { 22:jump ssh, 80:jump web, 443:jump web }
iifname vmap { "eth0":jump wan, "eth1":jump lan, "wg0":jump vpn }
```

### Timeout set as dynamic blacklist

```bash
set badguys { type ipv4_addr; flags dynamic, timeout; timeout 1h; size 65536 }
add @badguys { ip saddr limit rate over 4/minute }
ip saddr @badguys drop
```

### Policy drop, explicit allows

```bash
chain input { policy drop; ct state established,related accept; ... }
# Easier to reason about than policy accept + explicit drops.
```

### Comment your rules

```bash
nft add rule inet filter input tcp dport 22 accept comment \"ssh from anywhere\"
# (use single quotes if your shell collides with `"`)
```

### One file, atomically

```bash
# /etc/nftables.conf is the single source of truth. Edit, syntax-check, reload.
```

### Variables for tunables

```bash
define WAN     = "eth0"
define LAN     = "eth1"
define ADMINS  = { 203.0.113.10, 203.0.113.11 }
nft add rule inet filter input ip saddr $ADMINS tcp dport 22 accept
```

### Named counters for SLO tracking

```bash
nft add counter inet filter http_in
nft add rule inet filter input tcp dport 80 counter name "http_in" accept
nft list counter inet filter http_in
```

### Trace a specific flow

```bash
nft add rule inet filter prerouting ip saddr 192.0.2.1 meta nftrace set 1
sudo nft monitor trace
# emits per-rule trace lines for that source IP
```

## Tips

### Useful one-liners

```bash
# Fast firewall reload safety
alias nfreload='sudo nft -c -f /etc/nftables.conf && sudo systemctl reload nftables'

# Show ruleset with handles
sudo nft -a list ruleset

# JSON ruleset (for tooling)
sudo nft -j list ruleset | jq .

# Diff two rulesets
diff <(sudo nft list ruleset) <(cat /etc/nftables.conf)

# Show all sets and counts
sudo nft list sets
sudo nft list counters
sudo nft list maps

# Live-watch packet drops
sudo nft monitor | grep -i "DROP-INPUT"

# Trace SSH connections
sudo nft insert rule inet filter input tcp dport 22 meta nftrace set 1
sudo nft monitor trace
```

### Quick rules

```bash
# Block a country with a CIDR set
nft add set inet filter cn '{ type ipv4_addr; flags interval; }'
# (load CIDRs from a feed, e.g. ipdeny.com/ipblocks/data/countries/cn.zone)
nft add rule inet filter input ip saddr @cn drop

# Allow only one user's outbound port 25
nft add rule inet filter output tcp dport 25 meta skuid != 1000 reject

# DROP all traffic for a maintenance window (atomic on/off via flag set)
nft add set inet filter maintenance '{ type inet_service; }'
nft add rule inet filter input tcp dport @maintenance reject with tcp reset
nft add element inet filter maintenance '{ 80, 443 }'   # enable
nft delete element inet filter maintenance '{ 80, 443 }' # disable
```

### Visualization

```bash
# Pretty-print one chain
sudo nft -nn list chain inet filter input

# Tree of tables and chains
sudo nft list ruleset | grep -E '^(table|\s+chain)'

# Get just the structure (no rule bodies)
sudo nft -t list ruleset
```

### Cross-checks

```bash
# Can the kernel actually parse what you wrote?
sudo nft -c -f /etc/nftables.conf

# Does iptables-nft show your rules too?
sudo iptables -S
sudo ip6tables -S

# What's the active backend?
sudo iptables -V                          # look for "(nf_tables)"
```

### Daily-driver mental shortcuts

```bash
# I want to allow port X            -> tcp dport X accept
# I want to block IP Y              -> ip saddr Y drop
# I want to NAT outbound through Z  -> oifname "Z" masquerade in postrouting
# I want to forward port to host    -> dnat to A.B.C.D:P in prerouting + accept in forward
# I want a stateful firewall        -> ct state established,related accept first; policy drop
# I want to dispatch by port        -> tcp dport vmap { ... }
# I want a dynamic blacklist        -> set with flags dynamic,timeout; add via rule
# I want shaping                    -> meta mark set + tc filter handle N fw
```

### Useful kernel modules

```bash
# Load explicitly if needed (most distros auto-load):
sudo modprobe nf_tables
sudo modprobe nf_tables_inet
sudo modprobe nf_tables_ipv4
sudo modprobe nf_tables_ipv6
sudo modprobe nf_nat
sudo modprobe nf_conntrack
sudo modprobe nft_ct
sudo modprobe nft_log
sudo modprobe nft_limit
sudo modprobe nft_quota
sudo modprobe nft_reject
sudo modprobe nft_redir
sudo modprobe nft_masq
sudo modprobe nft_nat
sudo modprobe nft_chain_nat
sudo modprobe nft_chain_route
sudo modprobe nft_meta
sudo modprobe nft_set_hash
sudo modprobe nft_set_rbtree
```

### Useful sysctls

```bash
# IP forwarding (router)
net.ipv4.ip_forward = 1
net.ipv6.conf.all.forwarding = 1

# Conntrack capacity (busy boxes)
net.netfilter.nf_conntrack_max = 1000000
net.nf_conntrack_max          = 1000000

# Conntrack TCP timeouts
net.netfilter.nf_conntrack_tcp_timeout_established = 7200
net.netfilter.nf_conntrack_tcp_timeout_close_wait  = 60

# Strict reverse-path filtering (anti-spoof)
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1
```

### Service interaction notes

```bash
# Docker:        installs its own DOCKER, DOCKER-USER chains in iptables-nft. Don't `flush ruleset`
#                 unless you also restart Docker. Better: edit only your `inet filter` table.
# Podman:        rootless = no nft impact; rootful = uses CNI plugins which add nft rules.
# firewalld:     wraps nftables on RHEL/Fedora; either use firewalld OR raw nft, not both.
# ufw:           wraps iptables-nft on Ubuntu; works with nftables backend, but flushing nft
#                 ruleset removes ufw rules too. Re-enable ufw after.
# fail2ban:      writes iptables-nft rules by default; can be configured for `nftables-multiport`
#                 or `nftables-allports` actions to use raw nft sets directly.
# Kubernetes:    kube-proxy in `nftables` mode (1.29+) creates its own table `kube-proxy` —
#                 don't flush it.
```

### Hardware NIC offload requirements

```bash
# For flowtable HW offload:
#  - kernel 5.13+
#  - NIC + driver supports tc-flower offload
#  - check: ethtool -k eth0 | grep tc-offload
ethtool -K eth0 hw-tc-offload on
sudo dmesg | grep -i offload
```

### Debugging tools

```bash
sudo nft monitor                          # live event stream
sudo nft monitor trace                    # packet trace (after `meta nftrace set 1`)
sudo conntrack -L                         # current conntrack entries
sudo conntrack -E                         # conntrack event stream
sudo cat /proc/net/nf_conntrack | wc -l   # conntrack count
sudo nft list counters                    # all named counters
sudo cat /proc/net/netfilter/nf_log       # logger registration per family
sudo journalctl -k -g nftables            # kernel messages from nftables
```

### Capture-then-trace

```bash
# Trace only matching traffic (e.g. one source IP) without flooding logs
nft insert rule inet filter prerouting ip saddr 198.51.100.42 meta nftrace set 1
sudo nft monitor trace
# When done:
nft -a list chain inet filter prerouting
nft delete rule inet filter prerouting handle <N>
```

### Checking which rule is hitting

```bash
sudo nft -a list ruleset                  # show handles
sudo nft list ruleset | grep -E 'packets|bytes'   # see counters
# Or add a counter to the suspect rule, then check after traffic.
```

## See Also

- iptables, dns, dig, polyglot, bash

## References

- [nftables Wiki](https://wiki.nftables.org/)
- [nftables Wiki — Quick Reference](https://wiki.nftables.org/wiki-nftables/index.php/Quick_reference-nftables_in_10_minutes)
- [nftables Wiki — Moving from iptables to nftables](https://wiki.nftables.org/wiki-nftables/index.php/Moving_from_iptables_to_nftables)
- [nftables Wiki — Sets](https://wiki.nftables.org/wiki-nftables/index.php/Sets)
- [nftables Wiki — Maps](https://wiki.nftables.org/wiki-nftables/index.php/Maps)
- [nftables Wiki — Verdict Maps](https://wiki.nftables.org/wiki-nftables/index.php/Verdict_Maps_(vmaps))
- [nftables Wiki — Flowtables](https://wiki.nftables.org/wiki-nftables/index.php/Flowtables)
- [nftables Wiki — Performing Network Address Translation (NAT)](https://wiki.nftables.org/wiki-nftables/index.php/Performing_Network_Address_Translation_(NAT))
- [nftables Wiki — Counters](https://wiki.nftables.org/wiki-nftables/index.php/Counters)
- [nftables Wiki — Rate limiting matchings](https://wiki.nftables.org/wiki-nftables/index.php/Rate_limiting_matchings)
- [nftables Wiki — Logging traffic](https://wiki.nftables.org/wiki-nftables/index.php/Logging_traffic)
- [man nft(8)](https://man7.org/linux/man-pages/man8/nft.8.html)
- [Netfilter Project — nftables](https://www.netfilter.org/projects/nftables/)
- [Linux Kernel — Netfilter Documentation](https://www.kernel.org/doc/html/latest/networking/netfilter.html)
- [Red Hat — Getting Started with nftables](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/configuring_firewalls_and_packet_filters/getting-started-with-nftables_firewall-packet-filters)
- [Debian Wiki — nftables](https://wiki.debian.org/nftables)
- [Arch Wiki — nftables](https://wiki.archlinux.org/title/Nftables)
- [Ubuntu — nftables](https://ubuntu.com/server/docs/security-firewall)
- [Gentoo Wiki — nftables](https://wiki.gentoo.org/wiki/Nftables)
- [Eric Leblond — Why nftables](https://home.regit.org/2014/01/why-you-will-love-nftables/)
- [Pablo Neira Ayuso — nftables Talk (Netfilter Workshop)](https://workshop.netfilter.org/)
- [LWN — A new packet filter for Linux: nftables](https://lwn.net/Articles/564095/)
- [LWN — nftables: a new packet filtering engine](https://lwn.net/Articles/324989/)
- [Cloudflare — How we use nftables](https://blog.cloudflare.com/how-we-use-nftables/)
- [Wikipedia — nftables](https://en.wikipedia.org/wiki/Nftables)
