# iptables (Linux Packet Filter)

The classic Linux netfilter userspace tool — rules organized in tables and chains, traversed in a strict order, matched on protocol/address/state, then dispatched to a target. See `nftables` for the modern successor; iptables is being phased out but remains ubiquitous on production Linux as of 2026.

## Setup

### The three iptables binaries
On modern distros there are effectively three implementations exposed via the `iptables` command name. They all speak the same CLI but differ in what kernel subsystem actually programs the rules.

```bash
iptables --version                     # see which backend you are talking to
# iptables v1.8.10 (nf_tables)         # iptables-nft  — translates to nftables backend
# iptables v1.8.10 (legacy)            # iptables-legacy — original xtables/x_tables backend
```

Distros ship `iptables` as a symlink, usually managed by `update-alternatives`:

```bash
ls -l /usr/sbin/iptables               # symlink to xtables-nft-multi or xtables-legacy-multi
update-alternatives --display iptables # see all candidates and current selection
sudo update-alternatives --config iptables       # interactively switch backend
sudo update-alternatives --set iptables /usr/sbin/iptables-legacy
sudo update-alternatives --set iptables /usr/sbin/iptables-nft
```

Equivalent for IPv6, ARP, and the eb (Ethernet bridge) tool:

```bash
update-alternatives --config ip6tables
update-alternatives --config arptables
update-alternatives --config ebtables
```

### What each backend means
- `iptables-legacy` — the original implementation. Rules live in the kernel's `x_tables` subsystem. Stable, works with any kernel since 2.4, but slow with large rulesets and not actively developed.
- `iptables-nft` — a compatibility shim. Userspace parses iptables syntax, then programs the kernel's `nf_tables` backend (the engine behind `nftables`). Default on Debian 11+, Ubuntu 20.04+, RHEL 8+, Fedora, Arch.
- `nftables` (`nft`) — the native modern tool. Its own syntax, its own ruleset namespace, but uses the same kernel engine as iptables-nft.

The canonical advice: "iptables is being phased out for nftables — but the iptables CLI is still ubiquitous, and iptables-nft means new tools can keep using it without slowing the kernel." Mixing iptables-legacy and iptables-nft on the same host is dangerous: rules from one backend are invisible to the other.

### Detect which backend is active
```bash
iptables --version                              # parenthesized "(nf_tables)" or "(legacy)"
ls -l /usr/sbin/iptables                        # symlink target reveals the multi-call name
nft list ruleset                                 # if iptables-nft is in use, rules show up here too
sudo lsmod | grep -E '^nf_tables|^ip_tables'    # which kernel modules are loaded
```

If both `nf_tables` and `ip_tables` modules are loaded, both backends are programming the kernel — verify which the `iptables` binary points at.

### Install
Debian/Ubuntu:
```bash
sudo apt install iptables iptables-persistent netfilter-persistent
```

RHEL/CentOS/Fedora/Rocky/Alma:
```bash
sudo dnf install iptables iptables-services iptables-utils
sudo systemctl enable --now iptables ip6tables
```

Arch:
```bash
sudo pacman -S iptables-nft           # default; replaces iptables
sudo pacman -S iptables               # legacy variant (conflicts with iptables-nft)
```

Alpine:
```bash
sudo apk add iptables ip6tables
sudo rc-update add iptables default
```

### Required kernel modules
Most ship built-in or autoload, but for embedded kernels you may need to ensure:

```bash
sudo modprobe ip_tables nf_conntrack nf_nat iptable_filter iptable_nat iptable_mangle iptable_raw \
              xt_conntrack xt_state xt_multiport xt_recent xt_limit xt_mark xt_owner xt_tcpudp xt_LOG
```

For nftables-backend equivalents:
```bash
sudo modprobe nf_tables nft_compat
```

## Architecture

The Linux packet filter is `netfilter` — a set of hook points inside the kernel's network stack. iptables (and nftables) is the userspace tool that programs rules at those hooks.

```
            +-------------------+
            |   netfilter hooks |  (in-kernel)
            |                   |
NIC --> [PREROUTING] -> route? -+-> [INPUT]    -> local socket
                                |
                                +-> [FORWARD]  -> [POSTROUTING] --> NIC
                                                      ^
local socket -> [OUTPUT] ----------------------------+
```

### Layers
- **netfilter** — the kernel hook framework (hooks: PRE_ROUTING, LOCAL_IN, FORWARD, LOCAL_OUT, POST_ROUTING).
- **xtables / nf_tables** — the rule engine sitting on top of those hooks (legacy or modern backend).
- **tables** — logical groupings of chains by purpose: `filter`, `nat`, `mangle`, `raw`, `security`.
- **chains** — ordered lists of rules; each chain attaches to a specific hook.
- **rules** — match expressions plus a target (verdict).
- **targets** — verdicts: `ACCEPT`, `DROP`, `REJECT`, `RETURN`, `LOG`, `MARK`, `SNAT`, `DNAT`, `MASQUERADE`, custom user chain, etc.
- **matches** — extension modules loaded with `-m NAME` that add new match criteria.

A packet enters a hook, walks the chains attached to that hook in table-priority order, and at each rule the kernel checks every match in the rule. If all match, the target fires; if the target is a terminal verdict the walk stops, otherwise it continues to the next rule.

## Tables

Each table groups chains by purpose. You select a table with `-t TABLE`; the default is `filter`.

### filter — accept/drop/reject (the default)
The table most people mean when they say "iptables." Chains: INPUT, OUTPUT, FORWARD.

```bash
iptables -L -n -v                      # equivalent to: iptables -t filter -L -n -v
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
```

### nat — address/port translation
Triggered only on the first packet of a connection (NEW state); conntrack reapplies the same NAT to subsequent packets. Chains: PREROUTING (DNAT), POSTROUTING (SNAT/MASQUERADE), OUTPUT (locally generated DNAT), INPUT (rare; for stateless source-side translation in some setups).

```bash
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 80 -j DNAT --to-destination 10.0.0.5:80
```

### mangle — modify packet headers
Used to change TOS, TTL, mark packets for QoS / policy routing, etc. Chains: PREROUTING, INPUT, FORWARD, OUTPUT, POSTROUTING.

```bash
iptables -t mangle -A PREROUTING -p tcp --dport 22 -j MARK --set-mark 0x10
iptables -t mangle -A POSTROUTING -p tcp --sport 80 -j TOS --set-tos Maximize-Throughput
```

### raw — bypass connection tracking
Used early (before conntrack) to mark traffic as untracked. Chains: PREROUTING, OUTPUT.

```bash
iptables -t raw -A PREROUTING -p udp --dport 53 -j NOTRACK     # skip conntrack for DNS
iptables -t raw -A OUTPUT -p udp --dport 53 -j NOTRACK
```

### security — SELinux / Smack contexts
Used by Mandatory Access Control to apply per-packet security contexts. Chains: INPUT, OUTPUT, FORWARD. Rare outside hardened RHEL/SELinux deployments.

```bash
iptables -t security -A OUTPUT -p tcp --dport 443 -j SECMARK --selctx system_u:object_r:https_packet_t:s0
```

The canonical: "filter table is what most people mean; nat is for routers; mangle for QoS; raw for performance escapes."

## Chains — Built-In

Every built-in chain is bound to one of the five netfilter hooks.

| Chain | Hook | Triggered when |
| --- | --- | --- |
| PREROUTING | NF_INET_PRE_ROUTING | Packet just arrived on an interface, before the routing decision |
| INPUT | NF_INET_LOCAL_IN | Packet routed to a socket on this host |
| FORWARD | NF_INET_FORWARD | Packet routed through this host (not local) |
| OUTPUT | NF_INET_LOCAL_OUT | Packet generated locally, before routing decision |
| POSTROUTING | NF_INET_POST_ROUTING | Packet about to leave an interface, after routing |

### Chain × Table matrix
Not every table has every chain. Check via `iptables -t TABLE -S`.

| Chain        | filter | nat | mangle | raw | security |
| ------------ | :----: | :-: | :----: | :-: | :------: |
| PREROUTING   |        |  X  |   X    |  X  |          |
| INPUT        |   X    |  X  |   X    |     |    X     |
| FORWARD      |   X    |     |   X    |     |    X     |
| OUTPUT       |   X    |  X  |   X    |  X  |    X     |
| POSTROUTING  |        |  X  |   X    |     |          |

(`nat-INPUT` exists only on Linux 3.7+ for stateless source-side rewrites — rare.)

### Default policies
Each built-in chain has a default policy that fires when no rule matches. Default is `ACCEPT`. Set with `-P CHAIN POLICY`:

```bash
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT
```

User-defined chains have no policy — they implicitly RETURN.

## Chain Traversal Order

The packet flow is the most important diagram in iptables. Memorize it.

```
                                 +-----------------+
                                 | network stack   |
                                 +-----------------+
                                          |
                       +------------------+
                       v
                  +--------+
in --> [raw/PREROUTING] --> [conntrack] --> [mangle/PREROUTING] --> [nat/PREROUTING] -> ROUTING DECISION
                                                                                 |
                                            +------------------------------------+----------------------+
                                            |  destination = local?                                     |
                                            v                                                           v
                                       LOCAL DELIVERY                                              FORWARDING
                                            |                                                           |
                       [mangle/INPUT] -> [filter/INPUT] -> [nat/INPUT] -> local socket          [mangle/FORWARD] -> [filter/FORWARD]
                                                                                                                            |
                                                                                                                            v
LOCAL OUT:                                                                                                            ROUTING DECISION (out)
local socket -> [raw/OUTPUT] -> [conntrack] -> [mangle/OUTPUT] -> [nat/OUTPUT] -> [filter/OUTPUT]                            |
                                                                                          |                                  |
                                                                                          v                                  v
                                                                       [mangle/POSTROUTING] -> [nat/POSTROUTING] --> out interface
```

### Order of tables on each hook
- **PREROUTING:** raw → mangle → nat
- **INPUT:** mangle → filter → security → nat (Linux 3.7+, rare)
- **FORWARD:** mangle → filter → security
- **OUTPUT:** raw → mangle → nat → filter → security
- **POSTROUTING:** mangle → nat

### Routing decision
The kernel's routing table runs once after PREROUTING and again before POSTROUTING. NAT in PREROUTING happens before the routing decision; SNAT in POSTROUTING happens after. That's why `--to-destination` in PREROUTING actually changes where the packet goes, and `--to-source` in POSTROUTING preserves the original routing.

### User-defined chains as subroutines
A rule can `-j MYCHAIN` to dive into a user chain. The walk continues there; if the user chain `RETURN`s (or runs off the end), control returns to the caller and continues at the next rule. Think of user chains as functions.

## iptables Command Anatomy

The general form:

```bash
iptables -t TABLE OPERATION CHAIN [match...] -j TARGET [target options]
```

### Operations (mutually exclusive)
- `-A CHAIN` — append rule to end of chain
- `-I CHAIN [POS]` — insert rule (default position 1; top of chain)
- `-D CHAIN N` — delete rule N (1-indexed)
- `-D CHAIN spec...` — delete rule matching exact spec
- `-R CHAIN N spec...` — replace rule N with new spec
- `-L [CHAIN]` — list rules (in human form)
- `-S [CHAIN]` — dump rules in iptables-save format
- `-F [CHAIN]` — flush all rules in chain (or all chains in table)
- `-X [CHAIN]` — delete user chain (must be empty and unreferenced)
- `-Z [CHAIN]` — zero packet/byte counters
- `-N CHAIN` — create new user-defined chain
- `-E OLD NEW` — rename user chain
- `-P CHAIN POLICY` — set default policy on a built-in chain (ACCEPT or DROP)
- `-C CHAIN spec...` — check whether a rule exists (exit 0 = yes)

### Common matches
- `-p {tcp|udp|icmp|icmpv6|sctp|all|N}` — protocol
- `-s ADDR[/MASK]` — source address (CIDR allowed)
- `-d ADDR[/MASK]` — destination address
- `-i IFACE` — incoming interface (only valid in INPUT, FORWARD, PREROUTING)
- `-o IFACE` — outgoing interface (only valid in OUTPUT, FORWARD, POSTROUTING)
- `--sport PORT[:PORT]` — source port (with `-p tcp` or `-p udp`)
- `--dport PORT[:PORT]` — destination port
- `-m MODULE` — load match module (state, conntrack, multiport, recent, limit, ...)

### Flag ordering
The order of flags doesn't matter except for the operation flag (`-A`, `-I`, `-D`, etc.) which must come first and name the chain. Match modules loaded with `-m FOO` must precede their `--foo-...` options.

```bash
# These three are equivalent:
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -j ACCEPT -p tcp --dport 22
iptables -A INPUT --dport 22 -p tcp -j ACCEPT
```

### -A vs -I vs -D vs -R
```bash
iptables -A INPUT -p tcp --dport 80 -j ACCEPT      # append at the bottom
iptables -I INPUT -p tcp --dport 80 -j ACCEPT      # insert at top (position 1)
iptables -I INPUT 5 -p tcp --dport 80 -j ACCEPT    # insert at position 5
iptables -D INPUT -p tcp --dport 80 -j ACCEPT      # delete by exact spec
iptables -D INPUT 5                                # delete rule #5
iptables -R INPUT 5 -p tcp --dport 80 -j DROP      # replace rule #5
```

### Negation
Many flags accept `!` to negate:

```bash
iptables -A INPUT -p tcp ! --dport 22 -j DROP            # everything except SSH
iptables -A INPUT ! -i lo -d 127.0.0.0/8 -j DROP         # spoofed loopback from non-lo
iptables -A INPUT ! -s 192.168.1.0/24 -j DROP
```

## Rule Inspection

### List rules
```bash
iptables -L                            # filter table, all chains, names resolved (slow)
iptables -L -n                         # numeric — skip DNS / port name resolution
iptables -L -n -v                      # verbose — packet & byte counters
iptables -L -n -v --line-numbers       # add 1-indexed rule numbers
iptables -L INPUT -n -v --line-numbers # one chain only
iptables -t nat -L -n -v               # NAT table
iptables -t mangle -L -n -v
iptables -L -x                         # exact counter values (no K/M/G suffixes)
```

The canonical: "always use `-n` to skip DNS resolution slowdown." A `iptables -L` on a busy router can hang for minutes doing reverse DNS on counters.

### Save-format dump (rule export)
`-S` prints rules in the same syntax you'd type to recreate them:

```bash
iptables -S                            # all chains, filter table
iptables -S INPUT                      # one chain
iptables -t nat -S
iptables -S | grep -i ssh
```

### iptables-save — full state dump
Dumps every table in a single, restorable format:

```bash
iptables-save                          # to stdout
iptables-save > /tmp/rules.v4
iptables-save -c                       # include packet/byte counters
iptables-save -t filter                # one table
ip6tables-save                         # IPv6 sibling
```

### Counters
```bash
iptables -L -n -v                      # see pkts/bytes per rule
iptables -Z                            # zero all counters
iptables -Z INPUT 3                    # zero rule 3 in INPUT
```

## Saving and Restoring

iptables rules live only in the running kernel. Reboot or `iptables -F` and they're gone. Persistence is a separate concern.

### Manual save/restore
```bash
sudo iptables-save  > /etc/iptables/rules.v4
sudo ip6tables-save > /etc/iptables/rules.v6

sudo iptables-restore  < /etc/iptables/rules.v4
sudo ip6tables-restore < /etc/iptables/rules.v6
```

`iptables-restore` flushes and replaces the entire ruleset atomically. There is no "merge" mode — to apply incrementally use `-n` (no-flush):

```bash
iptables-restore -n < incremental.v4   # do not flush; add to existing
iptables-restore -t < rules.v4         # test parse only, do not apply
```

### Debian / Ubuntu — iptables-persistent
```bash
sudo apt install iptables-persistent netfilter-persistent
sudo netfilter-persistent save         # writes /etc/iptables/rules.v4 and rules.v6
sudo netfilter-persistent reload       # reload from /etc/iptables/
sudo systemctl status netfilter-persistent
```

### RHEL / CentOS / Fedora — iptables-services
```bash
sudo dnf install iptables-services
sudo systemctl enable --now iptables ip6tables
sudo service iptables save             # writes /etc/sysconfig/iptables
```

`/etc/sysconfig/iptables` is the conventional location on RHEL-family.

### Arch — manual systemd unit
```bash
sudo iptables-save > /etc/iptables/iptables.rules
sudo systemctl enable iptables
```

### rules.v4 file format
The on-disk format is exactly what `iptables-save` emits:

```
*filter
:INPUT DROP [0:0]
:FORWARD DROP [0:0]
:OUTPUT ACCEPT [0:0]
-A INPUT -i lo -j ACCEPT
-A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
-A INPUT -p tcp --dport 22 -j ACCEPT
COMMIT
```

`*table` opens, `:CHAIN POLICY [pkts:bytes]` declares chain, `-A ...` appends, `COMMIT` flushes. Multiple tables can appear in one file.

## Common Matches — Protocol

`-p` selects layer-4 protocol. Aliases come from `/etc/protocols`.

```bash
iptables -A INPUT -p tcp     -j ACCEPT     # TCP
iptables -A INPUT -p udp     -j ACCEPT     # UDP
iptables -A INPUT -p icmp    -j ACCEPT     # ICMPv4
iptables -A INPUT -p icmpv6  -j ACCEPT     # ICMPv6 (ip6tables)
iptables -A INPUT -p sctp    -j ACCEPT     # SCTP (telecom)
iptables -A INPUT -p esp     -j ACCEPT     # IPsec ESP
iptables -A INPUT -p ah      -j ACCEPT     # IPsec AH
iptables -A INPUT -p gre     -j ACCEPT     # GRE tunnels
iptables -A INPUT -p 47      -j ACCEPT     # protocol number works too
iptables -A INPUT -p all     -j ACCEPT     # any (the default)
```

### TCP/UDP ports
With `-p tcp` or `-p udp` you can use `--sport` and `--dport`:

```bash
iptables -A INPUT -p tcp --dport 22                -j ACCEPT
iptables -A INPUT -p tcp --dport 1024:65535        -j ACCEPT     # range
iptables -A INPUT -p udp --sport 53 --dport 1024:65535 -j ACCEPT # DNS reply
iptables -A INPUT -p tcp --dport ssh               -j ACCEPT     # name from /etc/services
iptables -A INPUT -p tcp ! --dport 22              -j DROP        # everything except SSH
```

### TCP flags
```bash
iptables -A INPUT -p tcp --syn                   -j LOG_AND_DROP   # any new TCP connection
iptables -A INPUT -p tcp --tcp-flags ALL NONE    -j DROP           # NULL scan
iptables -A INPUT -p tcp --tcp-flags ALL ALL     -j DROP           # XMAS scan
iptables -A INPUT -p tcp --tcp-flags SYN,FIN SYN,FIN -j DROP       # invalid combo
iptables -A INPUT -p tcp --tcp-flags SYN,RST SYN,RST -j DROP
```

### ICMP type
```bash
iptables -A INPUT -p icmp --icmp-type echo-request -j ACCEPT       # ping in
iptables -A INPUT -p icmp --icmp-type 8            -j ACCEPT       # numeric form
iptables -A INPUT -p icmp --icmp-type destination-unreachable -j ACCEPT
iptables -A INPUT -p icmp --icmp-type time-exceeded -j ACCEPT      # for traceroute
```

For IPv6 (`ip6tables`):
```bash
ip6tables -A INPUT -p icmpv6 --icmpv6-type neighbour-solicitation  -j ACCEPT
ip6tables -A INPUT -p icmpv6 --icmpv6-type neighbour-advertisement -j ACCEPT
ip6tables -A INPUT -p icmpv6 --icmpv6-type echo-request            -j ACCEPT
ip6tables -A INPUT -p icmpv6 --icmpv6-type packet-too-big          -j ACCEPT
```

### multiport — many ports in one rule
```bash
iptables -A INPUT -p tcp -m multiport --dports 22,80,443,8443 -j ACCEPT
iptables -A INPUT -p tcp -m multiport --dports 1000:2000,3000 -j ACCEPT  # mix range/list
iptables -A INPUT -p tcp -m multiport --sports 80,443 -j ACCEPT
iptables -A INPUT -p tcp -m multiport --ports 22,80   -j ACCEPT          # src OR dst match
```

Maximum 15 ports per rule (kernel limit).

## Common Matches — Address

```bash
iptables -A INPUT -s 192.168.1.10                   -j ACCEPT     # single host
iptables -A INPUT -s 192.168.1.0/24                 -j ACCEPT     # subnet
iptables -A INPUT -s 192.168.1.10-192.168.1.20      -j ACCEPT     # range (-m iprange)
iptables -A INPUT -m iprange --src-range 10.0.0.10-10.0.0.20 -j ACCEPT
iptables -A INPUT -d 10.0.0.1                       -j ACCEPT     # destination
iptables -A INPUT ! -s 192.168.1.0/24               -j DROP       # negation
```

### Interface
```bash
iptables -A INPUT  -i lo                 -j ACCEPT      # incoming on loopback
iptables -A INPUT  -i eth0               -j ACCEPT      # incoming on eth0
iptables -A OUTPUT -o eth1               -j ACCEPT      # outgoing on eth1
iptables -A FORWARD -i eth0 -o eth1      -j ACCEPT      # both directions for forwarding
iptables -A INPUT  -i 'wlan+'            -j DROP        # wildcard: wlan0, wlan1, ...
```

`-i` is valid only in chains where the packet has already chosen an inbound interface (PREROUTING, INPUT, FORWARD). `-o` only where the outbound interface is known (OUTPUT, FORWARD, POSTROUTING). Using the wrong direction produces:

```
iptables v1.8.10: Can't use -i with OUTPUT
```

## Common Matches — State / Conntrack

`-m state` is the legacy module; `-m conntrack` is the modern, more featureful replacement. Use `-m conntrack` in new rules.

### state values
- `NEW` — first packet of a new connection (TCP SYN, first UDP datagram, first ICMP)
- `ESTABLISHED` — packet of an existing tracked connection
- `RELATED` — packet related to an existing connection (FTP data, ICMP error for tracked conn)
- `INVALID` — packet conntrack cannot identify (wrong sequence, garbage)
- `UNTRACKED` — explicitly untracked via `raw -j NOTRACK`
- `DNAT` / `SNAT` — packet has been NATed in this direction (conntrack-only)

### Canonical pattern
Always allow ESTABLISHED,RELATED at the very top of INPUT to short-circuit ongoing flows. This single rule is the difference between a firewall that works and one that drops half its return traffic.

```bash
# legacy module
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -m state --state INVALID             -j DROP

# modern module (preferred)
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -m conntrack --ctstate INVALID             -j DROP
iptables -A INPUT -m conntrack --ctstate NEW -p tcp --dport 22 -j ACCEPT
```

### TCP SYN shortcut
`--syn` is shorthand for `--tcp-flags SYN,RST,ACK,FIN SYN`:

```bash
iptables -A INPUT -p tcp --syn -j ACCEPT       # only NEW TCP connections
iptables -A INPUT -p tcp ! --syn -m conntrack --ctstate NEW -j DROP   # invalid NEW (non-SYN)
```

### Extra conntrack options
```bash
iptables -A INPUT -m conntrack --ctorigsrc 10.0.0.0/24 -j ACCEPT      # original src
iptables -A INPUT -m conntrack --ctstatus ASSURED      -j ACCEPT      # only confirmed conns
iptables -A INPUT -m conntrack --ctexpire 1:60         -j LOG         # expiry seconds
iptables -A INPUT -m conntrack --ctproto tcp           -j ACCEPT
```

## Common Matches — multiport

`-m multiport` matches several ports in a single rule. The canonical "ssh + http + https in one line" pattern:

```bash
iptables -A INPUT -p tcp -m multiport --dports 22,80,443 -j ACCEPT
```

- `--dports a,b,c[:d,...]` — match destination port list/range (max 15 entries total)
- `--sports a,b,c` — match source port list
- `--ports a,b,c` — match either source OR destination

Cannot be combined with `--sport`/`--dport` directly; multiport replaces them. Cannot be used with port ranges in TCP options that exceed 15 expanded entries.

## Common Matches — recent

`-m recent` keeps a kernel-side list of source IPs and timestamps. Excellent for brute-force protection without external daemons.

```bash
# brute-force SSH guard: drop sources that hit dport 22 more than 4 times in 60s
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW \
   -m recent --set --name SSH --rsource

iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW \
   -m recent --update --seconds 60 --hitcount 4 --name SSH --rsource -j DROP

iptables -A INPUT -p tcp --dport 22 -j ACCEPT
```

### Options
- `--name LIST` — pick which recent table; default DEFAULT
- `--set` — add the source to the list
- `--update` — like `--set` but also matches if already present and updates timestamp
- `--rcheck` — like `--update` but does not refresh the timestamp
- `--remove` — delete entry on match
- `--seconds N` — only consider entries within last N seconds
- `--hitcount N` — match if at least N entries
- `--rsource` / `--rdest` — track by source or destination IP
- `--rttl` — also require matching TTL (catches simple IP spoofing)
- `--reap` — purge expired entries; required with `--seconds` for cleanup

The recent list lives at `/proc/net/xt_recent/SSH` (one file per name). You can read it, append IPs by writing to it (`echo "+1.2.3.4" > /proc/net/xt_recent/SSH`), or clear it (`echo / > .../SSH`).

The canonical: "fail2ban-style brute-force protection in pure iptables — no daemon required."

## Common Matches — limit

`-m limit` is a token bucket — match if rate is below the limit. Different from `recent`: limit is global per-rule (not per-source), and is intended for things like log throttling and ICMP rate-shaping.

```bash
iptables -A INPUT -p icmp --icmp-type echo-request \
   -m limit --limit 1/s --limit-burst 5 -j ACCEPT
iptables -A INPUT -p icmp --icmp-type echo-request -j DROP
```

```bash
# log dropped packets, but no more than 5/min, with a burst of 10
iptables -A INPUT -m limit --limit 5/min --limit-burst 10 -j LOG --log-prefix "iptables-drop: "
```

### Options
- `--limit N/{second|minute|hour|day}` — average rate
- `--limit-burst N` — initial bucket size

`hashlimit` is the per-source variant — see `man iptables-extensions`:

```bash
iptables -A INPUT -p tcp --syn -m hashlimit \
    --hashlimit-name syn-flood --hashlimit-mode srcip \
    --hashlimit-above 50/sec --hashlimit-burst 100 -j DROP
```

## Common Matches — Other

### owner — packets from local UID/GID
Only valid in OUTPUT (and POSTROUTING for some kernels). The packet must be locally generated.

```bash
iptables -A OUTPUT -m owner --uid-owner 1000 -j ACCEPT          # user with UID 1000
iptables -A OUTPUT -m owner --uid-owner alice -j ACCEPT
iptables -A OUTPUT -m owner --gid-owner docker -j ACCEPT
iptables -A OUTPUT -m owner ! --uid-owner root -p tcp --dport 80 -j DROP
```

### mac — match source MAC address
Only valid where the packet still has L2 info: PREROUTING, INPUT, FORWARD on Ethernet.

```bash
iptables -A INPUT -i eth0 -m mac --mac-source AA:BB:CC:DD:EE:FF -j ACCEPT
iptables -A INPUT -i eth0 ! -m mac --mac-source AA:BB:CC:DD:EE:FF -j DROP
```

### string — payload pattern match
Searches packet payload for a byte string. `--algo bm` (Boyer-Moore) is the usual choice.

```bash
iptables -A FORWARD -p tcp --dport 80 -m string --algo bm --string "User-Agent: badbot" -j DROP
iptables -A INPUT -m string --algo kmp --string "GET /admin" -j LOG --log-prefix "ADMIN-PROBE "
```

Be careful: per-packet payload scan is expensive at high rates and easy to evade with TLS or HTTP/2.

### comment — annotate every rule
Comments are visible in `iptables -L` and `iptables-save`. Always comment.

```bash
iptables -A INPUT -p tcp --dport 22 -m comment --comment "ssh in from anywhere" -j ACCEPT
iptables -A INPUT -p tcp --dport 9090 -m comment --comment "TICKET-123 prom from monitoring" -j ACCEPT
```

### mark — match packet skb mark
Used together with `MARK`/`CONNMARK` targets to do cross-stage tagging.

```bash
iptables -A FORWARD -m mark --mark 0x1 -j ACCEPT
iptables -A FORWARD -m mark --mark 0x1/0xff -j ACCEPT          # masked match
iptables -t mangle -A PREROUTING -p tcp --dport 22 -j MARK --set-mark 0x1
```

### addrtype — match LOCAL / BROADCAST / MULTICAST
```bash
iptables -A INPUT -m addrtype --dst-type LOCAL          -j ACCEPT
iptables -A INPUT -m addrtype --dst-type BROADCAST       -j DROP
iptables -A INPUT -m addrtype --dst-type MULTICAST       -j DROP
iptables -A INPUT -m addrtype --src-type LOCAL --limit-iface-in -j ACCEPT
```

### connlimit — limit concurrent connections per source
```bash
iptables -A INPUT -p tcp --syn --dport 80 \
    -m connlimit --connlimit-above 50 --connlimit-mask 32 -j REJECT
```

### time — schedule rules by time of day
```bash
iptables -A FORWARD -m time --timestart 22:00 --timestop 06:00 \
    --weekdays Mon,Tue,Wed,Thu,Fri -j DROP
```

### length — match packet length
```bash
iptables -A INPUT -p tcp -m length --length 0:64 -j DROP   # tiny packet flood
iptables -A INPUT -p icmp -m length --length 1000: -j DROP
```

### tos — match Type of Service
```bash
iptables -A FORWARD -m tos --tos Minimize-Delay -j ACCEPT
```

### ttl — match IP TTL
```bash
iptables -A INPUT -m ttl --ttl-eq 64 -j ACCEPT
iptables -A INPUT -m ttl --ttl-lt 5  -j DROP            # likely traceroute
iptables -A INPUT -m ttl --ttl-gt 64 -j LOG --log-prefix "weird-ttl: "
```

### pkttype — match broadcast/multicast/unicast
```bash
iptables -A INPUT -m pkttype --pkt-type broadcast -j DROP
iptables -A INPUT -m pkttype --pkt-type multicast -j DROP
iptables -A INPUT -m pkttype --pkt-type unicast   -j ACCEPT
```

### policy — match IPsec policy
```bash
iptables -A INPUT -m policy --dir in --pol ipsec --proto esp -j ACCEPT
iptables -A INPUT -m policy --dir in --pol none -p tcp --dport 22 -j DROP   # require IPsec
```

### dscp — match DSCP value
```bash
iptables -A FORWARD -m dscp --dscp-class EF -j ACCEPT          # voice
iptables -A FORWARD -m dscp --dscp 0x2e     -j ACCEPT          # numeric
```

### iprange — non-CIDR IP ranges
```bash
iptables -A INPUT -m iprange --src-range 10.0.0.10-10.0.0.50 -j ACCEPT
iptables -A INPUT -m iprange --dst-range 192.168.1.100-192.168.1.150 -j ACCEPT
```

### physdev — bridged interface match (for bridges with brnf enabled)
```bash
iptables -A FORWARD -m physdev --physdev-in eth1 --physdev-out eth2 -j ACCEPT
```

### u32 — generic byte-offset match
Match arbitrary bits in the packet header. Power user only — fragile across kernel changes.

```bash
# Match TCP MSS option == 1460 in SYN packets (offset math)
iptables -A INPUT -p tcp --syn -m u32 --u32 "0>>22&0x3C@4=0x02040000" -j ACCEPT
```

### bpf — attach a cBPF program (legacy backend)
Compile a tcpdump-style filter to cBPF and reference it. Modern equivalents use eBPF via tc/xdp.

```bash
iptables -A INPUT -m bpf --bytecode "4,48 0 0 9,21 0 1 6,6 0 0 65535,6 0 0 0" -j ACCEPT
```

### cgroup — match by cgroup membership
```bash
iptables -A OUTPUT -m cgroup --path "system.slice/sshd.service" -j ACCEPT
iptables -A OUTPUT -m cgroup --cgroup 0x10 -j ACCEPT          # numeric net_cls classid
```

### devgroup — match by interface group
```bash
ip link set dev eth0 group 1
iptables -A INPUT -m devgroup --src-group 1 -j ACCEPT
```

### realm — match routing realm
```bash
iptables -A INPUT -m realm --realm 5 -j ACCEPT
```

### rt — match IPv6 routing header (ip6tables)
```bash
ip6tables -A INPUT -m rt --rt-type 0 -j DROP        # type-0 routing header (deprecated, attack vector)
```

### hashlimit — per-source rate limiting
```bash
iptables -A INPUT -p tcp --syn --dport 80 -m hashlimit \
    --hashlimit-name http-syn \
    --hashlimit-mode srcip \
    --hashlimit-above 100/sec \
    --hashlimit-burst 200 \
    --hashlimit-htable-expire 300000 \
    -j DROP
```

## Targets — Filter

A target is the verdict. Some targets are *terminating* (the chain walk stops) and some are *non-terminating* (e.g. LOG, MARK).

### ACCEPT
Pass the packet to the next stage of the netfilter pipeline.

```bash
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
```

### DROP
Drop the packet silently. Sender sees timeout; great for stealth, terrible for diagnosing legitimate failures.

```bash
iptables -A INPUT -s 1.2.3.4 -j DROP
```

### REJECT
Drop with an explicit error. Sender sees an immediate failure; preferred for INPUT on internal services.

```bash
iptables -A INPUT -p tcp --dport 25 -j REJECT --reject-with tcp-reset
iptables -A INPUT -p udp --dport 53 -j REJECT --reject-with icmp-port-unreachable
iptables -A INPUT -p icmp -j REJECT --reject-with icmp-host-unreachable
```

`--reject-with` types: `icmp-net-unreachable`, `icmp-host-unreachable`, `icmp-port-unreachable`, `icmp-proto-unreachable`, `icmp-net-prohibited`, `icmp-host-prohibited`, `icmp-admin-prohibited`, and (TCP only) `tcp-reset`.

### RETURN
Stop walking this chain; resume in the calling chain at the next rule. In a built-in chain, RETURN means "fall through to the chain default policy."

```bash
iptables -A MYCHAIN -p tcp --dport 80 -j ACCEPT
iptables -A MYCHAIN -j RETURN
```

### LOG (non-terminating)
Emit a kernel log message and continue. Pair with REJECT/DROP on the next rule.

```bash
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW \
    -j LOG --log-prefix "ssh-attempt: " --log-level info
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW -j DROP
```

Options: `--log-prefix STR`, `--log-level {emerg,alert,crit,err,warn,notice,info,debug}` (or numeric 0-7), `--log-tcp-options`, `--log-ip-options`, `--log-tcp-sequence`, `--log-uid`.

### NFLOG (non-terminating)
Like LOG but emits via netlink; consumed by `ulogd2`, `nftnl`, etc. Required for high-volume logging.

```bash
iptables -A INPUT -j NFLOG --nflog-group 5 --nflog-prefix "drop:"
```

### AUDIT (non-terminating)
Send a record to the kernel audit subsystem.

```bash
iptables -A INPUT -p tcp --dport 22 -j AUDIT --type accept
```

### TRACE (non-terminating, raw table only)
Mark packets so that every rule they touch is logged via the trace mechanism. Invaluable for "why did this packet drop?"

```bash
iptables -t raw -A PREROUTING -p tcp --dport 80 -s 192.0.2.5 -j TRACE
# then watch the journal:
journalctl -kf | grep TRACE
```

Output looks like:

```
TRACE: raw:PREROUTING:policy:2 IN=eth0 OUT= SRC=192.0.2.5 DST=10.0.0.1 LEN=60 ...
TRACE: filter:INPUT:rule:3 IN=eth0 OUT= SRC=192.0.2.5 DST=10.0.0.1 ...
TRACE: filter:INPUT:policy:0 IN=eth0 OUT= SRC=192.0.2.5 DST=10.0.0.1 ... (DROP)
```

Each line names `table:chain:rule:N` — the exact rule the packet hit. Remove the TRACE rule when finished, since trace events are expensive.

### CHECKSUM — recompute IPv4 checksum
Useful when something modified the packet without updating the checksum (rare, niche).

```bash
iptables -t mangle -A POSTROUTING -p udp --dport 68 -j CHECKSUM --checksum-fill
```

### CLASSIFY — set tc classifier
For shaping with `tc`:

```bash
iptables -t mangle -A POSTROUTING -p tcp --sport 22 -j CLASSIFY --set-class 1:10
```

### ECN — clear ECN bits
```bash
iptables -t mangle -A POSTROUTING -p tcp -j ECN --ecn-tcp-remove
```

### IDLETIMER — generate uevent on idle
```bash
iptables -A INPUT -j IDLETIMER --timeout 600 --label idle10min
```

### NETMAP — 1:1 stateless network mapping
```bash
iptables -t nat -A PREROUTING  -d 198.51.100.0/24 -j NETMAP --to 10.0.0.0/24
iptables -t nat -A POSTROUTING -s 10.0.0.0/24     -j NETMAP --to 198.51.100.0/24
```

### NFQUEUE — hand off to userspace
For deep packet inspection in userspace (Suricata in IPS mode, Snort inline, custom Go/Rust netfilterqueue tools).

```bash
iptables -A FORWARD -j NFQUEUE --queue-num 0 --queue-bypass
```

`--queue-bypass` accepts the packet if no userspace listener is attached (otherwise traffic stalls).

## Targets — NAT

NAT happens only on the first (NEW) packet of a connection; conntrack remembers the translation and reapplies it. Therefore NAT rules go in the `nat` table only.

### SNAT — Source NAT
Rewrite the source IP to a fixed address as the packet leaves. POSTROUTING only.

```bash
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j SNAT --to-source 203.0.113.5
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j SNAT --to-source 203.0.113.5-203.0.113.10
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j SNAT --to-source 203.0.113.5:1024-65535
```

### MASQUERADE
Like SNAT but uses whatever IP the outbound interface currently has. Use when the egress IP is dynamic (DHCP, PPPoE). Slightly slower than SNAT because it queries the interface address per connection.

```bash
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o ppp0 -j MASQUERADE
```

The canonical "internet-sharing" pattern (one interface to LAN, one to WAN, MASQUERADE to WAN, FORWARD allowed both ways).

### DNAT — Destination NAT
Rewrite the destination IP/port. PREROUTING (incoming) or OUTPUT (locally generated).

```bash
# Forward external 80 -> internal 10.0.0.10:80
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 80 -j DNAT --to-destination 10.0.0.10
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 80 -j DNAT --to-destination 10.0.0.10:8080
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 8080 -j DNAT --to-destination 10.0.0.10-10.0.0.20:80

# Don't forget the FORWARD rule!
iptables -A FORWARD -i eth0 -d 10.0.0.10 -p tcp --dport 80 -m conntrack --ctstate NEW -j ACCEPT
iptables -A FORWARD -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
```

### REDIRECT
Special-case DNAT to the local host. Use to bend a port to a service running on the firewall itself.

```bash
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080      # transparent proxy
iptables -t nat -A OUTPUT     -p tcp --dport 80 -o lo -j REDIRECT --to-port 3128  # local proxy on lo
```

### Persistent NAT options
- `--persistent` — same source always picks same translated IP (for ranges)
- `--random` — source port randomized
- `--random-fully` — even better randomization (Linux 3.13+)

```bash
iptables -t nat -A POSTROUTING -o eth0 -j SNAT --to-source 203.0.113.5-203.0.113.10 --persistent
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE --random
```

## Targets — Mangle

### MARK — set skb fwmark
The mark is a 32-bit value attached to the packet's in-kernel skb. Used for QoS classification, policy routing (`ip rule fwmark`), and matching downstream.

```bash
iptables -t mangle -A PREROUTING -p tcp --dport 22 -j MARK --set-mark 0x10
iptables -t mangle -A PREROUTING -p tcp --dport 80 -j MARK --set-xmark 0x20/0xff   # mask
ip rule add fwmark 0x10 table 100
ip route add default via 10.0.0.1 table 100
```

### CONNMARK — set/save/restore conntrack mark
The connmark is a 32-bit value attached to the conntrack entry. Useful for "tag once, remember for the life of the connection."

```bash
iptables -t mangle -A PREROUTING -p tcp --dport 22 -j CONNMARK --set-mark 0x10
iptables -t mangle -A PREROUTING -j CONNMARK --restore-mark        # connmark -> skbmark
iptables -t mangle -A POSTROUTING -j CONNMARK --save-mark           # skbmark -> connmark
```

### TPROXY — transparent proxy redirection
Direct a packet to a local process without changing the destination. Common in intercepting proxies (squid, envoy in transparent mode).

```bash
iptables -t mangle -A PREROUTING -p tcp --dport 80 \
    -j TPROXY --tproxy-mark 0x1/0x1 --on-port 3129
ip rule add fwmark 0x1/0x1 lookup 100
ip route add local 0.0.0.0/0 dev lo table 100
```

### TOS — set Type of Service / DSCP
```bash
iptables -t mangle -A POSTROUTING -p tcp --sport 22 -j TOS --set-tos Minimize-Delay
iptables -t mangle -A POSTROUTING -p tcp --sport 80 -j TOS --set-tos 0x10
iptables -t mangle -A POSTROUTING -p tcp --sport 5060 -j DSCP --set-dscp-class EF
```

### TTL — set time-to-live
```bash
iptables -t mangle -A POSTROUTING -o eth0 -j TTL --ttl-set 64
iptables -t mangle -A POSTROUTING -o eth0 -j TTL --ttl-dec 1
iptables -t mangle -A POSTROUTING -o eth0 -j TTL --ttl-inc 5
```

### TCPMSS — clamp MSS to PMTU (PPPoE / GRE / VPN fix)
```bash
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN \
    -j TCPMSS --clamp-mss-to-pmtu
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 1360
```

The canonical traffic-classification pipeline: mark in PREROUTING → match mark elsewhere → restore from CONNMARK in incoming traffic.

## Targets — Logging

### LOG basics
```bash
iptables -A INPUT -j LOG \
    --log-prefix "[iptables-dropped] " \
    --log-level info \
    --log-tcp-options \
    --log-ip-options \
    --log-tcp-sequence \
    --log-uid
```

LOG writes to the kernel ring buffer (`dmesg`), which is normally relayed to syslog/journald. View:

```bash
sudo dmesg -wT | grep iptables-dropped
journalctl -kf | grep iptables-dropped
tail -F /var/log/kern.log         # Debian/Ubuntu
tail -F /var/log/messages          # RHEL
```

### Rate-limited LOG
Without rate limiting, LOG can DOS your disk during an attack.

```bash
iptables -A INPUT -m limit --limit 5/min --limit-burst 10 \
    -j LOG --log-prefix "iptables-dropped: " --log-level info
iptables -A INPUT -j DROP
```

### LOG before terminal verdicts (canonical "log dropped packets" pattern)
```bash
iptables -N LOG_DROP
iptables -A LOG_DROP -m limit --limit 5/min -j LOG --log-prefix "DROP: " --log-level info
iptables -A LOG_DROP -j DROP

iptables -A INPUT -p tcp --dport 23 -j LOG_DROP
iptables -A INPUT -j LOG_DROP                  # default
```

Putting LOG and DROP in their own chain means you can swap "log + drop" for "log + accept" in one place during debugging.

### NFLOG for high-volume / structured logging
```bash
iptables -A INPUT -j NFLOG --nflog-group 5 --nflog-prefix "drop:" --nflog-range 64
```

`ulogd2` consumes `--nflog-group 5` and writes JSON, MySQL, pcap, etc. Far better than syslog at sustained rates.

## User-Defined Chains

User chains are subroutines: a `-j MYCHAIN` from anywhere dives in; on RETURN (or fall-through) control returns to the caller.

### Lifecycle
```bash
iptables -N MY-INPUT-RULES                     # create
iptables -A MY-INPUT-RULES -p tcp --dport 22 -j ACCEPT
iptables -A MY-INPUT-RULES -p tcp --dport 80 -j ACCEPT
iptables -A MY-INPUT-RULES -j RETURN           # explicit return (also implicit at end)
iptables -A INPUT -j MY-INPUT-RULES            # call it from INPUT

iptables -F MY-INPUT-RULES                     # flush rules in chain
iptables -X MY-INPUT-RULES                     # delete chain (must be empty + unreferenced)
iptables -E MY-INPUT-RULES NEW-NAME            # rename
```

### Canonical pattern: organize by purpose
```bash
iptables -N TCP-IN
iptables -N UDP-IN
iptables -N ICMP-IN
iptables -N LOG_DROP

iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -m conntrack --ctstate INVALID -j LOG_DROP

iptables -A INPUT -p tcp  -j TCP-IN
iptables -A INPUT -p udp  -j UDP-IN
iptables -A INPUT -p icmp -j ICMP-IN

iptables -A TCP-IN  -p tcp  --dport 22  -j ACCEPT
iptables -A TCP-IN  -p tcp  --dport 80  -j ACCEPT
iptables -A TCP-IN  -p tcp  --dport 443 -j ACCEPT
iptables -A UDP-IN  -p udp  --dport 53  -j ACCEPT
iptables -A ICMP-IN -p icmp --icmp-type echo-request -j ACCEPT

iptables -A INPUT -j LOG_DROP
```

Reads top-down like a router config; each chain has one job.

## Common Workflows

### Allow established + new SSH (canonical default-allow ordering)
```bash
iptables -F INPUT
iptables -A INPUT -i lo                                                -j ACCEPT
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED            -j ACCEPT
iptables -A INPUT -m conntrack --ctstate INVALID                        -j DROP
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW          -j ACCEPT
iptables -A INPUT                                                       -j DROP
```

### Block specific IP at the top of INPUT
```bash
iptables -I INPUT 1 -s 1.2.3.4 -j DROP
iptables -I INPUT 1 -s 1.2.3.0/24 -j DROP
```

### NAT outbound for an internal network (router pattern)
```bash
sysctl -w net.ipv4.ip_forward=1
iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o eth0 -j MASQUERADE
iptables -A FORWARD -i eth1 -o eth0 -s 192.168.1.0/24 -j ACCEPT
iptables -A FORWARD -i eth0 -o eth1 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
```

### Port forward 80 -> internal 10.0.0.10:8080
```bash
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 80 -j DNAT --to-destination 10.0.0.10:8080
iptables -A FORWARD -i eth0 -o eth1 -d 10.0.0.10 -p tcp --dport 8080 \
   -m conntrack --ctstate NEW,ESTABLISHED,RELATED -j ACCEPT
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE       # for replies
```

### Rate-limit SSH (recent module variant)
```bash
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW \
    -m recent --set --name SSH --rsource

iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW \
    -m recent --update --seconds 60 --hitcount 4 --rttl --name SSH --rsource -j DROP

iptables -A INPUT -p tcp --dport 22 -j ACCEPT
```

### Allow only a country's IP space (with ipset and a country list)
```bash
ipset create de hash:net
for cidr in $(curl -s https://www.ipdeny.com/ipblocks/data/countries/de.zone); do
    ipset add de "$cidr"
done
iptables -A INPUT -p tcp --dport 22 -m set --match-set de src -j ACCEPT
iptables -A INPUT -p tcp --dport 22 -j DROP
```

### IPsec / WireGuard pass-through
```bash
iptables -A INPUT -p udp --dport 500   -j ACCEPT     # IKE
iptables -A INPUT -p udp --dport 4500  -j ACCEPT     # NAT-T
iptables -A INPUT -p esp               -j ACCEPT
iptables -A INPUT -p udp --dport 51820 -j ACCEPT     # WireGuard
```

### Container egress (Docker classic)
Docker installs its own DOCKER, DOCKER-USER, DOCKER-ISOLATION-* chains. Rules in DOCKER-USER are applied to forwarded container traffic before Docker's own rules — that's where to add custom blocks.

```bash
iptables -I DOCKER-USER -i eth0 ! -s 10.10.0.0/16 -j DROP    # only allow VPN to reach containers
```

### Drop all but a list of trusted bastions
```bash
ipset create bastions hash:ip
ipset add bastions 198.51.100.5
ipset add bastions 198.51.100.6
ipset add bastions 198.51.100.7
iptables -A INPUT -p tcp --dport 22 -m set --match-set bastions src -m conntrack --ctstate NEW -j ACCEPT
iptables -A INPUT -p tcp --dport 22 -j DROP
```

### Allow private subnet to talk to itself only
```bash
iptables -P FORWARD DROP
iptables -A FORWARD -i eth1 -o eth1 -s 192.168.1.0/24 -d 192.168.1.0/24 -j ACCEPT
iptables -A FORWARD -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
```

### Allow inbound only from CDN egress IPs
```bash
ipset create cf hash:net
for cidr in $(curl -s https://www.cloudflare.com/ips-v4); do
    ipset add cf "$cidr"
done
iptables -A INPUT -p tcp --dport 443 -m set --match-set cf src -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j DROP
```

### Anti port-scan via psd
```bash
iptables -A INPUT -m psd --psd-weight-threshold 21 --psd-delay-threshold 300 \
    --psd-lo-ports-weight 3 --psd-hi-ports-weight 1 -j DROP
```

### Block IPv4 fragments
Some attacks rely on fragmentation. Most modern protocols don't need fragmented IPv4.

```bash
iptables -A INPUT -f -j DROP
```

### Allow Plex / DLNA on local subnet only
```bash
iptables -A INPUT -s 192.168.1.0/24 -p tcp --dport 32400 -j ACCEPT
iptables -A INPUT -s 192.168.1.0/24 -p udp --dport 1900  -j ACCEPT
iptables -A INPUT -s 192.168.1.0/24 -p udp --dport 5353  -j ACCEPT
```

## Connection Tracking

Conntrack is the kernel's flow table. NAT, stateful filtering, and many matches depend on it.

### Inspect the conntrack table
```bash
cat /proc/net/nf_conntrack | head
sudo conntrack -L                          # human-readable
sudo conntrack -L -p tcp --dport 443       # filter
sudo conntrack -L -s 10.0.0.5
sudo conntrack -E                          # live event stream (NEW/UPDATE/DESTROY)
sudo conntrack -C                          # count entries
sudo conntrack -F                          # flush table (drops in-flight conns!)
sudo conntrack -D -p tcp --dport 443       # delete matching entries
```

### Tune timeouts
```bash
ls /proc/sys/net/netfilter/nf_conntrack_*timeout*
sysctl net.netfilter.nf_conntrack_tcp_timeout_established      # default 432000s (5 days)
sysctl -w net.netfilter.nf_conntrack_tcp_timeout_established=86400
sysctl -w net.netfilter.nf_conntrack_udp_timeout=30
sysctl -w net.netfilter.nf_conntrack_udp_timeout_stream=120
```

### Resize the conntrack table
On busy gateways the default is too small. Symptoms: `nf_conntrack: table full, dropping packet` in dmesg.

```bash
sudo sysctl -w net.netfilter.nf_conntrack_max=1048576
sudo sysctl -w net.netfilter.nf_conntrack_buckets=262144
echo 1048576 | sudo tee /sys/module/nf_conntrack/parameters/hashsize
```

Persist via `/etc/sysctl.d/99-conntrack.conf`.

### NOTRACK — opt out of conntrack
For high-traffic stateless paths (DNS authoritative servers, NTP, public TFTP), conntrack is pure overhead. Use the raw table to bypass:

```bash
iptables -t raw -A PREROUTING -p udp --dport 53 -j NOTRACK
iptables -t raw -A OUTPUT     -p udp --sport 53 -j NOTRACK
```

Untracked packets still pass filter rules, but never enter the conntrack table.

### Conntrack helpers (ALG)
Some protocols carry IPs in their payload (FTP, SIP, IRC DCC). Conntrack helpers parse the L7 to open the secondary flow as RELATED.

```bash
sudo modprobe nf_conntrack_ftp
sudo modprobe nf_conntrack_sip
ls /proc/net/nf_conntrack_helpers 2>/dev/null
```

Modern kernels require explicit attachment via `CT --helper`:

```bash
iptables -t raw -A PREROUTING -p tcp --dport 21 -j CT --helper ftp
iptables -t raw -A OUTPUT     -p tcp --dport 21 -j CT --helper ftp
iptables -A INPUT -m conntrack --ctstate RELATED -m helper --helper ftp -j ACCEPT
```

Helpers are a known security risk (parsing untrusted payload in kernel). Disable globally if not needed:

```bash
sudo sysctl -w net.netfilter.nf_conntrack_helper=0
```

### conntrack zones
Multiple isolated conntrack namespaces — useful when you have overlapping address spaces (CGNAT, multi-tenant gateways).

```bash
iptables -t raw -A PREROUTING -i tenant1 -j CT --zone 1
iptables -t raw -A PREROUTING -i tenant2 -j CT --zone 2
```

### Track only what you need
If you only need conntrack for a specific port range, NOTRACK everything else:

```bash
iptables -t raw -A PREROUTING -p tcp --dport 1:65535 -j ACCEPT      # explicit pass-through
iptables -t raw -A PREROUTING                       -j NOTRACK
```

## ipset Integration

`ipset` stores large sets of addresses, networks, MACs, or port pairs in efficient hash structures. iptables matches against a set in O(1) — far better than 100k separate `-s` rules.

### Install
```bash
sudo apt install ipset
sudo dnf install ipset ipset-service
```

### Set types
- `hash:ip` — single IPs
- `hash:net` — CIDR networks
- `hash:ip,port` — IP + port pair
- `hash:mac` — MAC addresses
- `bitmap:ip` — dense IP ranges (small, fast)
- `list:set` — set of sets

### Create / populate / use
```bash
ipset create blacklist hash:ip family inet hashsize 65536 maxelem 1000000
ipset add  blacklist 1.2.3.4
ipset add  blacklist 5.6.7.0/24                # works on hash:net
ipset test blacklist 1.2.3.4
ipset list blacklist
ipset save blacklist > /etc/ipset/blacklist
ipset restore < /etc/ipset/blacklist
ipset destroy blacklist

iptables -I INPUT -m set --match-set blacklist src -j DROP
iptables -A INPUT -p tcp --dport 22 -m set ! --match-set whitelist src -j DROP
```

### Persist
Debian: `ipset save > /etc/iptables/ipsets`, restore via systemd unit before iptables-persistent.
RHEL: `systemctl enable ipset` and put files in `/etc/sysconfig/ipset.d/`.

The canonical: "block 100k IPs efficiently" — load the set once, single iptables rule references it.

## Common Patterns

### Default-DROP firewall (host)
The minimum production INPUT policy:

```bash
iptables -F
iptables -X
iptables -P INPUT   DROP
iptables -P FORWARD DROP
iptables -P OUTPUT  ACCEPT

iptables -A INPUT -i lo                                                -j ACCEPT
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED            -j ACCEPT
iptables -A INPUT -m conntrack --ctstate INVALID                        -j DROP
iptables -A INPUT -p icmp --icmp-type echo-request                      -j ACCEPT
iptables -A INPUT -p tcp -m multiport --dports 22,80,443                -j ACCEPT
```

### Hairpin NAT
Internal client wants to reach a service on the firewall's external IP, but routed back inside. Without hairpin, the SYN goes out, comes back via DNAT, and the reply is asymmetric and dropped.

```bash
iptables -t nat -A PREROUTING  -d 203.0.113.5 -p tcp --dport 80 \
    -j DNAT --to-destination 10.0.0.10:80
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -d 10.0.0.10 -p tcp --dport 80 \
    -j SNAT --to-source 10.0.0.1
```

### Transparent proxy (REDIRECT)
```bash
iptables -t nat -A PREROUTING -i br0 -p tcp --dport 80 -j REDIRECT --to-port 3128
```

### Anti-spoofing (martian filter)
```bash
iptables -A INPUT  -i eth0 -s 10.0.0.0/8       -j DROP    # external claiming RFC1918
iptables -A INPUT  -i eth0 -s 192.168.0.0/16   -j DROP
iptables -A INPUT  -i eth0 -s 172.16.0.0/12    -j DROP
iptables -A INPUT  -i eth0 -s 127.0.0.0/8      -j DROP
iptables -A INPUT ! -i lo  -d 127.0.0.0/8      -j DROP
```

### Anti-flood (SYN limit)
```bash
iptables -N SYN-FLOOD
iptables -A INPUT -p tcp --syn -j SYN-FLOOD
iptables -A SYN-FLOOD -m limit --limit 1/s --limit-burst 4 -j RETURN
iptables -A SYN-FLOOD -j DROP
```

### Block low-and-slow (slowloris-ish)
```bash
iptables -A INPUT -p tcp --syn --dport 80 \
    -m connlimit --connlimit-above 25 --connlimit-mask 32 -j REJECT
```

### Allow Docker host but block container egress to private nets
```bash
iptables -I DOCKER-USER -d 10.0.0.0/8     -j REJECT
iptables -I DOCKER-USER -d 192.168.0.0/16 -j REJECT
iptables -I DOCKER-USER -d 172.16.0.0/12 ! -i docker0 -j REJECT
```

### Kubernetes — never edit kube-managed chains
`kube-proxy` (iptables mode) owns chains beginning with `KUBE-*`. Treat them as ephemeral — add custom rules only via `INPUT-USER`-style sub-chains called from INPUT, never inside `KUBE-SERVICES` or `KUBE-FORWARD`.

## Migration to nftables

The canonical strategy is: `iptables-save` -> `iptables-restore-translate` -> review -> apply via `nft`. The translation is mechanical but imperfect — review every line.

### iptables-translate — single rule
```bash
iptables-translate -A INPUT -p tcp --dport 22 -j ACCEPT
# nft 'add rule ip filter INPUT tcp dport 22 counter accept'

ip6tables-translate -A INPUT -p tcp --dport 22 -j ACCEPT
```

### iptables-restore-translate — full ruleset
```bash
sudo iptables-save > rules.v4
iptables-restore-translate -f rules.v4 > rules.nft

# review carefully
less rules.nft

# apply
sudo nft -f rules.nft

# verify
sudo nft list ruleset
```

### iptables-nft compat layer (no translation needed)
On modern distros the `iptables` binary is already `iptables-nft`, so existing iptables rules already program nf_tables. Migrating to native `nft` syntax is a quality-of-life upgrade (sets, maps, intervals, jhash, vmaps), not a correctness one.

```bash
sudo update-alternatives --set iptables /usr/sbin/iptables-nft
sudo iptables-restore < rules.v4               # programs nf_tables backend
sudo nft list table ip filter                  # see them as nft would
```

### Things that don't translate cleanly
- Some custom match modules (`xt_recent` does, but with a separate map)
- Hand-crafted IPv4-only patterns that need to be re-expressed for `inet` family
- `ipset` -> nftables sets (mechanical but cosmetic differences)
- Counters: nft does not start counters by default; add `counter` clause if you want stats

The kernel runs both `nf_tables` and `x_tables` backends in parallel. Once you switch to nft natively, do `iptables -F` and disable iptables persistence to avoid mixed rulesets.

### Side-by-side syntax comparison
| Action | iptables | nft |
| --- | --- | --- |
| Allow ssh | `iptables -A INPUT -p tcp --dport 22 -j ACCEPT` | `nft 'add rule inet filter input tcp dport 22 accept'` |
| Drop CIDR | `iptables -A INPUT -s 1.2.3.0/24 -j DROP` | `nft 'add rule inet filter input ip saddr 1.2.3.0/24 drop'` |
| Stateful  | `iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT` | `nft 'add rule inet filter input ct state established,related accept'` |
| MASQ      | `iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE` | `nft 'add rule inet nat postrouting oifname "eth0" masquerade'` |
| DNAT      | `iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to-destination 10.0.0.10:80` | `nft 'add rule inet nat prerouting tcp dport 80 dnat ip to 10.0.0.10:80'` |
| ipset     | `iptables -A INPUT -m set --match-set bl src -j DROP` | `nft 'add rule inet filter input ip saddr @bl drop'` |

### Disable iptables-persistent after nftables migration
```bash
sudo systemctl disable --now netfilter-persistent
sudo systemctl enable --now nftables
sudo nft -f /etc/nftables.conf
```

### Migration order of operations
1. Audit current rules: `iptables-save > /tmp/before.v4`.
2. Translate: `iptables-restore-translate -f /tmp/before.v4 > /tmp/after.nft`.
3. Review `/tmp/after.nft` line by line. Pay attention to:
   - Comments (preserved but reformatted)
   - Sets and ipset references (must become nft sets)
   - Counters (often dropped — re-add `counter` clauses)
   - User-defined chains (translated as nft chains)
4. Test in a transactional file: `sudo nft -c -f /tmp/after.nft` (parse only).
5. Apply on a non-production host first.
6. Once stable, switch the alternative: `sudo update-alternatives --set iptables /usr/sbin/iptables-nft` if not already, or remove iptables entirely and rely on nft.
7. Disable iptables-persistent and enable nftables service.
8. Reboot and verify with `nft list ruleset`.

### What you gain by switching natively
- **Sets and maps as first class** — `set blacklist { type ipv4_addr; flags interval; elements = { 1.2.0.0/16, 5.6.0.0/16 } }`.
- **Verdict maps** — direct dispatch instead of cascaded matches: `tcp dport vmap { 22: jump ssh-rules, 80: jump http-rules }`.
- **inet family** — one ruleset for IPv4 + IPv6.
- **Atomic transactions** — `nft -f` applies all-or-nothing.
- **Modern syntax** — fewer footguns, no `-m` module loading dance.

## Common Errors and Fixes (EXACT text)

### `iptables: No chain/target/match by that name.`
Either the chain doesn't exist (typo, or you forgot to `-N` it) or the kernel module for the match/target isn't loaded.

```bash
# example
$ iptables -A INPUT -m receent --update -j DROP
iptables: No chain/target/match by that name.

# fix: typo  ->  recent
sudo modprobe xt_recent
iptables -A INPUT -m recent --update --name SSH -j DROP
```

### `iptables v1.8.10 (nf_tables): Couldn't load match \`X\`:No such file or directory`
Missing kernel module.

```bash
sudo modprobe xt_string xt_owner xt_recent xt_conntrack
```

### `iptables v1.8.x: setting -i in postrouting chain is illegal`
You used `-i` (incoming interface) in POSTROUTING, where the packet is already on its way out. Use `-o` instead. PREROUTING / INPUT / FORWARD use `-i`; OUTPUT / FORWARD / POSTROUTING use `-o`.

```bash
# bad
iptables -t nat -A POSTROUTING -i eth0 -j MASQUERADE

# fixed
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```

### `iptables v1.8.10 (nf_tables): Bad argument 'Foo'`
Syntax error — usually a malformed IP, port, or unknown TCP flag.

```bash
# bad
iptables -A INPUT -s 192.168.1.999 -j DROP

# fixed
iptables -A INPUT -s 192.168.1.99 -j DROP
```

### `iptables-restore v1.8.10 (nf_tables): line 12 failed`
The 12th non-comment line of the input file was rejected. Print the line and inspect:

```bash
awk '!/^#/ && !/^$/ {n++; if (n==12) print}' rules.v4
iptables-restore -t < rules.v4         # test parse without applying
```

### `nf_conntrack: table full, dropping packet`
Conntrack table exhausted. Bump it:

```bash
dmesg | grep nf_conntrack
sudo sysctl -w net.netfilter.nf_conntrack_max=1048576
echo 1048576 | sudo tee /sys/module/nf_conntrack/parameters/hashsize
```

### `iptables: Resource temporarily unavailable.`
Conntrack table full or an internal kernel resource is exhausted (rare). Same fix as above; check `dmesg`.

### `Another app is currently holding the xtables lock`
Two iptables operations colliding. Use `-w` to wait:

```bash
iptables -w -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -w 5 -A INPUT ...                    # wait up to 5 seconds
```

### `RTNETLINK answers: File exists` (when using ipset and iptables together)
Means a chain or set already exists. Flush before recreating:

```bash
ipset destroy blacklist 2>/dev/null
ipset create blacklist hash:ip
```

### `iptables -L` hangs forever
Reverse-DNS on packet counters. Always pass `-n`:

```bash
iptables -L -n -v
```

### `iptables -A INPUT -j ACCEPT` did nothing visible
You appended a rule below the default policy or below a `-j DROP` catch-all. Use `-I` to insert at top, or check rule order.

```bash
iptables -L INPUT --line-numbers -n -v
```

## Common Gotchas (broken + fixed)

### Locking yourself out via SSH
```bash
# bad — set policy DROP first, then SSH rule second; mid-update SSH session dies
iptables -P INPUT DROP
iptables -A INPUT -p tcp --dport 22 -j ACCEPT      # too late, you're already disconnected

# fixed — allow ESTABLISHED first, THEN tighten
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -P INPUT DROP
# safer still: schedule a panic-flush in case you really lock yourself out
echo 'iptables -F; iptables -P INPUT ACCEPT' | at now + 5 minutes
```

### Forgetting the loopback rule
```bash
# bad — local services break (DNS resolver, postgres on 127.0.0.1, etc.)
iptables -P INPUT DROP

# fixed
iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT ! -i lo -d 127.0.0.0/8 -j DROP   # plus anti-spoof
```

### Stateful firewall without ESTABLISHED rule
```bash
# bad — outgoing requests succeed but replies are dropped on INPUT
iptables -P INPUT DROP
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# fixed
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -P INPUT DROP
```

### NAT without ip_forward enabled
```bash
# bad — MASQUERADE rule installed, but kernel doesn't forward at all
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

# fixed
echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward
sudo sysctl -w net.ipv4.ip_forward=1
echo 'net.ipv4.ip_forward = 1' | sudo tee /etc/sysctl.d/99-forward.conf
sudo sysctl --system
```

### Using -A instead of -I when you needed top-of-chain
```bash
# bad — block IP appended after the default ACCEPT policy
iptables -A INPUT -s 1.2.3.4 -j DROP

# fixed — insert at the top
iptables -I INPUT 1 -s 1.2.3.4 -j DROP
```

### Changes lost on reboot
```bash
# bad — rules disappear after reboot, server is unprotected
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# fixed — Debian/Ubuntu
sudo netfilter-persistent save

# fixed — RHEL family
sudo service iptables save

# fixed — Arch / generic
sudo iptables-save | sudo tee /etc/iptables/iptables.rules
sudo systemctl enable iptables
```

### Mixing iptables-legacy and iptables-nft on the same host
```bash
# bad — apt installed iptables-nft, you used iptables-legacy in a script
sudo iptables-legacy -A INPUT -j DROP   # invisible to iptables-nft and nft commands!

# fixed — pick one, use it consistently
sudo update-alternatives --set iptables /usr/sbin/iptables-nft
which iptables   # always verify
```

### Wrong direction flag
```bash
# bad
iptables -t nat -A POSTROUTING -i eth0 -j MASQUERADE
# iptables: setting -i in postrouting chain is illegal

# fixed
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```

### Forgetting FORWARD rule alongside DNAT
```bash
# bad — port forwarded by DNAT but FORWARD chain drops the packet
iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 80 -j DNAT --to-destination 10.0.0.10:80

# fixed
iptables -A FORWARD -i eth0 -d 10.0.0.10 -p tcp --dport 80 -j ACCEPT
iptables -A FORWARD -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
```

### Logging without rate limit during attack
```bash
# bad — disk fills with kernel log entries during a SYN flood
iptables -A INPUT -p tcp --syn -j LOG --log-prefix "SYN: "

# fixed
iptables -A INPUT -p tcp --syn -m limit --limit 5/min -j LOG --log-prefix "SYN: "
```

### Putting LOG after a terminal verdict
```bash
# bad — DROP is terminal, the LOG never runs
iptables -A INPUT -j DROP
iptables -A INPUT -j LOG

# fixed — LOG first (non-terminal), then DROP
iptables -A INPUT -j LOG --log-prefix "INPUT-DROP: "
iptables -A INPUT -j DROP
```

### Comment forgotten — three weeks later you don't know why a rule exists
```bash
# bad
iptables -A INPUT -p tcp --dport 8443 -j ACCEPT

# fixed
iptables -A INPUT -p tcp --dport 8443 -m comment --comment "TICKET-4321 vendor monitoring agent" -j ACCEPT
```

### IPv6 forgotten
`iptables` is IPv4 only. IPv6 is handled by `ip6tables` — separate ruleset, separate persistence file.

```bash
# bad — host is wide open on IPv6
iptables -P INPUT DROP

# fixed
ip6tables -P INPUT DROP
ip6tables -A INPUT -i lo -j ACCEPT
ip6tables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
ip6tables -A INPUT -p ipv6-icmp -j ACCEPT
ip6tables -A INPUT -p tcp --dport 22 -j ACCEPT
```

### Trying to DROP loopback
```bash
# bad — many local services bind 127.0.0.1; this breaks DNS, postgres, nginx upstreams, ...
iptables -A INPUT -d 127.0.0.0/8 -j DROP

# fixed — never block lo; only block forged loopback from non-lo
iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT ! -i lo -d 127.0.0.0/8 -j DROP
```

### REJECT used in OUTPUT to a black-hole upstream
```bash
# bad — REJECT in OUTPUT generates ICMP back to itself; REJECT --reject-with tcp-reset only valid for tcp
iptables -A OUTPUT -d 1.2.3.4 -j REJECT --reject-with tcp-reset      # ok if tcp
iptables -A OUTPUT -d 1.2.3.4 -p udp -j REJECT --reject-with tcp-reset
# iptables: tcp-reset only valid for TCP

# fixed
iptables -A OUTPUT -d 1.2.3.4 -p tcp -j REJECT --reject-with tcp-reset
iptables -A OUTPUT -d 1.2.3.4 -p udp -j REJECT --reject-with icmp-port-unreachable
```

## Performance Tips

### Order rules by hit frequency
First-match-wins, every rule before the matching one is evaluated. Put high-volume allows at the top.

```bash
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT   # 99% of traffic
iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW -j ACCEPT
iptables -A INPUT -m conntrack --ctstate INVALID -j DROP
iptables -A INPUT -j DROP
```

### Use ipset for big lists
Twenty thousand `-s X.X.X.X -j DROP` rules cost O(N) per packet. One `ipset` lookup is O(1). Always use sets for long blocklists.

### Use raw NOTRACK for stateless paths
DNS authoritative, NTP, public TFTP, syslog UDP — bypass conntrack:

```bash
iptables -t raw -A PREROUTING -p udp --dport 53 -j NOTRACK
iptables -t raw -A OUTPUT     -p udp --sport 53 -j NOTRACK
```

### Avoid per-packet payload scans at line rate
`-m string` is expensive at high pps. Use it only on slow control-plane traffic, not data plane.

### Prefer iptables-nft over legacy on big rulesets
nf_tables backend uses sets and maps internally; iterates rules faster than xtables for large counts. On a typical k8s node that translates to multi-millisecond CPU savings per syscall.

### Counters are cheap; comments are free
Don't strip them in the name of "performance" — the kernel counts every rule regardless. The savings are imaginary.

### Atomic restores
`iptables-restore` is atomic — one syscall installs the entire ruleset. Don't apply rules one by one in a hot loop; emit `iptables-save` format and pipe to restore.

```bash
{
  echo "*filter"
  echo ":INPUT DROP [0:0]"
  echo ":FORWARD DROP [0:0]"
  echo ":OUTPUT ACCEPT [0:0]"
  echo "-A INPUT -i lo -j ACCEPT"
  echo "-A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT"
  echo "-A INPUT -p tcp --dport 22 -j ACCEPT"
  echo "COMMIT"
} | sudo iptables-restore
```

### Use the right hook
`raw` PREROUTING runs before conntrack — match here for the highest performance on stateless paths. `mangle` runs after raw but before nat. Avoid putting non-mangle work in mangle (slower because mangle hooks fire on every packet).

### Cache lookups via ipset hash:ip,port,net
For "match this IP from this network on this port", a single set lookup beats three rules.

```bash
ipset create svc hash:net,port,ip
ipset add svc 10.0.0.0/24,tcp:443,203.0.113.5
iptables -A FORWARD -m set --match-set svc src,dst,dst -j ACCEPT
```

### Avoid REJECT for high-volume drop paths
REJECT generates an outbound packet (ICMP or TCP RST). At hundreds of thousands of pps, that doubles CPU and outbound load. Use DROP for floods; reserve REJECT for low-volume internal services where fast failure is preferred.

### Avoid -L --line-numbers in hot paths
Pretty-printing builds strings; `-S` is faster for diff/automation:

```bash
iptables -S | sort > before
# ... change rules ...
iptables -S | sort > after
diff -u before after
```

## Idioms

### Canonical INPUT order
1. `-A INPUT -i lo -j ACCEPT`
2. `-A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT`
3. `-A INPUT -m conntrack --ctstate INVALID -j DROP`
4. specific allows (multiport, per-source whitelists)
5. rate-limited LOG of drops
6. `-A INPUT -j DROP` (or `-P INPUT DROP`)

### Comment every rule
```bash
iptables -A INPUT -p tcp --dport 22 -m comment --comment "ssh: bastion only" -j ACCEPT
iptables -A INPUT -p tcp --dport 9100 -m comment --comment "node_exporter: prom" -j ACCEPT
```

### rules.v4 in version control
Keep `/etc/iptables/rules.v4` in git. Diff to detect drift, review with PRs.

```bash
sudo -u git cp /etc/iptables/rules.v4 /etc/iptables/rules.v4.tmp
git -C /etc/iptables add rules.v4
git -C /etc/iptables commit -m 'firewall: open 9090 for monitoring'
```

### Test before commit
```bash
sudo iptables-restore -t < rules.v4   # parse-test
sudo iptables-restore   < rules.v4
sudo iptables -L -n -v --line-numbers
```

### Always include a `pre-up` panic flush
On hosted VMs, schedule a recovery flush in case you lock yourself out:

```bash
echo 'iptables -F; iptables -P INPUT ACCEPT' | at now + 5 minutes
```

If you confirm everything still works, `atrm <jobid>` to cancel the flush. Otherwise it auto-rescues you in 5 minutes.

### Use sub-chains for organization
Group related rules in user-defined chains (TCP-IN, UDP-IN, LOG-DROP). Easier to flush/replace one chain than splice rules in the middle of INPUT.

### Pair every -A LOG with a downstream verdict
LOG by itself doesn't drop; always DROP / REJECT / ACCEPT after.

### Never mix iptables-legacy and iptables-nft
Pick one backend per host and stick with it. `update-alternatives --display iptables` to verify.

### Debugging dropped packets
```bash
# 1. Add a tagged LOG just before the catch-all DROP
iptables -I INPUT -j LOG --log-prefix "INPUT-FALLTHROUGH: " --log-level info

# 2. Reproduce
ssh user@host

# 3. Watch
journalctl -kf | grep INPUT-FALLTHROUGH
# INPUT-FALLTHROUGH: IN=eth0 OUT= MAC=... SRC=192.0.2.5 DST=10.0.0.1 LEN=60 \
#   TOS=0x00 PREC=0x00 TTL=63 ID=0 DF PROTO=TCP SPT=51823 DPT=2222 \
#   WINDOW=64240 RES=0x00 SYN URGP=0
```

### Counters as a debugging tool
A rule with zero packets/bytes is unmatched. A rule that should be matching but shows 0 is a syntax/order bug.

```bash
iptables -L INPUT -n -v --line-numbers
# num   pkts bytes target     prot opt in     out     source       destination
# 1   142103   12M ACCEPT     all  --  lo     *       0.0.0.0/0    0.0.0.0/0
# 2  9821201  1.2G ACCEPT     all  --  *      *       0.0.0.0/0    0.0.0.0/0    ctstate ESTABLISHED,RELATED
# 3        0     0 ACCEPT     tcp  --  *      *       0.0.0.0/0    0.0.0.0/0    tcp dpt:8080  ## <-- ZERO HITS
# 4    23914  1.4M DROP       all  --  *      *       0.0.0.0/0    0.0.0.0/0
```

Rule 3 has zero hits — either nobody's connecting to 8080 or it's masked by an earlier rule.

### iptables exit codes
- `0` — success
- `1` — generic error (unknown chain, syntax)
- `2` — invalid command-line option
- `3` — incompatible options
- `4` — already exists (e.g. `-N` of an existing chain)

### Useful tcpdump for verifying
```bash
sudo tcpdump -ni eth0 -c 20 'host 192.0.2.5 and tcp port 22'
sudo tcpdump -ni any 'icmp'
sudo tcpdump -ni eth0 -nnvv -A 'tcp port 80'
```

If iptables drops the packet before the socket layer, tcpdump on the wire still shows it (tcpdump taps before INPUT chain). If tcpdump shows but the app doesn't see — iptables is the suspect.

### conntrack -E for live flow visibility
```bash
sudo conntrack -E -p tcp                            # all TCP events
sudo conntrack -E -e NEW                            # only new flows
sudo conntrack -E -d 10.0.0.5
```

### Rule snapshot via systemd timer
```bash
# /etc/systemd/system/iptables-snapshot.service
[Service]
Type=oneshot
ExecStart=/bin/sh -c 'iptables-save > /var/lib/iptables/snapshots/rules-$(date +%%Y%%m%%d-%%H%%M).v4'

# /etc/systemd/system/iptables-snapshot.timer
[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
sudo systemctl enable --now iptables-snapshot.timer
```

## Tips

- `iptables -L -n -v --line-numbers` is the single most useful inspection command.
- `iptables-save` and `iptables-restore` are the only correct way to bulk-edit rules.
- Use `-m comment --comment "..."` on every rule. Future you will thank past you.
- Test rules before persisting: `iptables-restore -t < rules.v4`.
- Schedule a 5-minute panic flush via `at` before locking down SSH from a remote shell.
- IPv6 lives in `ip6tables` — don't forget to mirror v4 rules.
- `ip6tables` lacks NAT historically, but Linux 3.7+ supports `ip6tables -t nat`.
- `nft` (nftables) is the modern replacement; learn it eventually but iptables is fine for now.
- Mixing backends ((iptables-nft + iptables-legacy)) silently splits your ruleset — pick one.
- Conntrack is your friend; ESTABLISHED,RELATED at the top of INPUT covers 99% of traffic.
- ipset is mandatory once your blocklist exceeds a few hundred entries.
- DNAT requires a matching FORWARD allow; people forget this constantly.
- MASQUERADE is slower than SNAT but works on dynamic egress IPs (DHCP, PPPoE, hotspots).
- `--reject-with tcp-reset` is friendlier than DROP for internal services.
- TRACE in the raw table is the iptables-equivalent of "verbose mode" for diagnosing why a packet drops.
- `iptables -nvL --line-numbers` is also writeable as `iptables -L -n -v --line-numbers` — order doesn't matter.
- `xtables-monitor -e` (newer iptables) shows live netfilter trace events.
- `journalctl -kf | grep -i nf_conntrack` to spot table-full drops live.
- Containers with their own network namespace get their own iptables ruleset; `nsenter -t PID -n iptables -L` to inspect.
- `firewalld` and `ufw` are higher-level wrappers — they program iptables (or nftables) under the hood; don't run them alongside hand-written rules unless you understand the layering.

## See Also

- nftables, dns, dig, polyglot, bash

## References

- [man iptables(8)](https://man7.org/linux/man-pages/man8/iptables.8.html)
- [man iptables-extensions(8)](https://man7.org/linux/man-pages/man8/iptables-extensions.8.html)
- [man iptables-save(8)](https://man7.org/linux/man-pages/man8/iptables-save.8.html)
- [man iptables-restore(8)](https://man7.org/linux/man-pages/man8/iptables-restore.8.html)
- [man iptables-translate(8)](https://man7.org/linux/man-pages/man8/iptables-translate.8.html)
- [man ip6tables(8)](https://man7.org/linux/man-pages/man8/ip6tables.8.html)
- [man ip6tables-extensions(8)](https://man7.org/linux/man-pages/man8/ip6tables-extensions.8.html)
- [man conntrack(8)](https://man7.org/linux/man-pages/man8/conntrack.8.html)
- [man ipset(8)](https://man7.org/linux/man-pages/man8/ipset.8.html)
- [man nft(8)](https://man7.org/linux/man-pages/man8/nft.8.html)
- [Netfilter Project Documentation](https://www.netfilter.org/documentation/)
- [Netfilter Packet Flow Diagram](https://www.netfilter.org/documentation/HOWTO/packet-filtering-HOWTO.html)
- [Linux Kernel Netfilter Documentation](https://www.kernel.org/doc/html/latest/networking/netfilter.html)
- [iptables Tutorial by Oskar Andreasson](https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html)
- [Linux Firewalls — Steve Suehring (Pearson)](https://www.pearson.com/en-us/subject-catalog/p/linux-firewalls-enhancing-security-with-nftables-and-beyond/P200000009466)
- [Red Hat — Using Firewalls](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/8/html/securing_networks/using-and-configuring-firewalld_securing-networks)
- [Debian Wiki — iptables](https://wiki.debian.org/iptables)
- [Arch Wiki — iptables](https://wiki.archlinux.org/title/Iptables)
- [Cloudflare — Why we use the Linux kernel's TCP stack](https://blog.cloudflare.com/why-we-use-the-linux-kernels-tcp-stack/)
- [nftables wiki — Migrating from iptables](https://wiki.nftables.org/wiki-nftables/index.php/Moving_from_iptables_to_nftables)
- [nftables wiki — Main page](https://wiki.nftables.org/)
- [Wireguard NAT examples](https://www.wireguard.com/quickstart/)
- [Linux Foundation — netfilter project history](https://wiki.linuxfoundation.org/networking/netfilter)
- [Hacking Linux Exposed: Linux Security Secrets](https://www.oreilly.com/library/view/linux-firewalls-3rd/9780132366458/)
- [Phil Dibowitz — Configuring iptables With xtables-addons](https://www.phildev.net/iptables/)
- [SANS — Linux Firewalls Reading Room](https://www.sans.org/reading-room/whitepapers/linux/)
- [LWN — net: continuing migration from iptables to nftables](https://lwn.net/Articles/761310/)
- [Cilium — Why kube-proxy iptables mode struggles at scale](https://cilium.io/blog/2020/09/22/cilium-eBPF-based-host-routing/)
- [Docker docs — Packet filtering and firewalls](https://docs.docker.com/network/iptables/)
- [Kubernetes docs — IPVS-based In-Cluster Load Balancing](https://kubernetes.io/docs/reference/networking/virtual-ips/)
- [Ubuntu Server Guide — Firewall](https://ubuntu.com/server/docs/firewalls)
- [Linux Networking Documentation — netfilter sysctls](https://www.kernel.org/doc/Documentation/networking/nf_conntrack-sysctl.txt)
- [BPF Compiler Collection (bcc) tools — for kernel-level inspection](https://github.com/iovisor/bcc)
- [SUSE — Firewall and iptables HOWTO](https://documentation.suse.com/sles/15-SP4/html/SLES-all/cha-security-firewall.html)
- [Gentoo Wiki — iptables](https://wiki.gentoo.org/wiki/Iptables)
