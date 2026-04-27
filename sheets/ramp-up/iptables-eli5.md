# iptables / netfilter — ELI5

> iptables is the bouncer at every door of your computer. Every packet that arrives or leaves has to walk past the bouncer, and the bouncer has a clipboard of rules that say "let this one in" or "throw this one out."

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` and `cs ramp-up tcp-eli5` will make this sheet way easier)

You do not need to be a Linux expert to read this sheet. You do not need to know what a "packet" is. You do not need to know what an IP address is. By the end of this sheet you will know all of those things in plain English, and you will have typed real `iptables` and `nft` commands and watched real packets get blocked, accepted, and translated.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath are what your computer prints back at you. We call that "output." If you see a `#` at the start of a line, that means "you need to be the **root** user to do this — use `sudo`."

## What Even Is iptables / netfilter

### Imagine your computer is a building with lots of doors

Picture a really big building. People keep walking up to it. Some are trying to come in. Some are trying to leave. Some are just passing through (your building is on the way to somewhere else).

If anybody could just walk in and out of any door whenever they wanted, the building would be chaos. Bad guys would walk in. Important visitors would walk into the wrong room. People would steal things. People would leave with things they shouldn't have.

So the building has **bouncers.** A bouncer stands at every door. Each bouncer has a clipboard. The clipboard has a long list of rules. The bouncer reads the rules from top to bottom. As soon as a rule matches the person trying to come in, the bouncer does what that rule says: let them in, throw them out, send them somewhere else, write down their name, or pass them to a different bouncer for a second opinion.

**Those bouncers are iptables.** The doors are the places in your computer where network packets go in and out. The clipboard is a **chain.** The rules are exactly that — rules. The list of "what to do" is called a **target.**

### What is a packet, anyway?

A packet is a tiny envelope. The internet is built out of these envelopes. Every time you load a web page, your computer sends out hundreds of envelopes. Every reply comes back as more envelopes. Each envelope has an address on the outside (where it's going), a return address (where it came from), and a little message inside.

If you have not read `cs ramp-up tcp-eli5` and `cs ramp-up ip-eli5` yet, that's okay — for this sheet, just remember:

- A **packet** is one envelope.
- The outside of the envelope has the **source IP** (where it came from) and **destination IP** (where it's going).
- It also says what kind of envelope it is: **TCP** (the kind that asks for receipts), **UDP** (fire-and-forget), **ICMP** (the kind ping uses), and a few others.
- It has a **port number**, which is like an apartment number inside the address. Port 22 is SSH. Port 80 is HTTP. Port 443 is HTTPS. Port 53 is DNS.

Every single packet that comes into your computer, or leaves your computer, or passes through your computer on the way somewhere else, has to walk past the bouncers. iptables is the system that gives those bouncers their clipboards.

### What is netfilter?

Here is the slightly confusing part. Inside your kernel there is a piece of software called **netfilter.** Netfilter is the actual machinery that intercepts packets. It is the building structure: the doors, the hallways, the signs that say "use this entrance."

`iptables` is the **command-line tool** you use to write rules and put them on netfilter's clipboards. The bouncers are netfilter. The clipboards are inside netfilter. iptables is just how you tell netfilter what to write on the clipboards.

You will hear people say "iptables" and "netfilter" almost interchangeably. That's fine. In this sheet we mostly say "iptables" because that's the tool name.

### And what is nftables?

`nftables` (or `nft`) is the new, modernized version of iptables. Same idea. Same bouncers. Same clipboards. Cleaner words. Fewer weird abbreviations. Faster engine inside.

If iptables is a worn paperback you've owned for twenty years, nftables is the same book with a new cover, better paper, and the typos fixed. Both books tell the same story.

Most modern Linux systems (RHEL 8+, Debian 11+, Ubuntu 21.04+) run nftables under the hood, but they keep an `iptables` command around that translates to the new format so old habits still work. That's the **iptables-nft** compatibility layer. We'll cover that more later.

### One last picture: the building map

```
                           +------------------+
                           |  Your computer   |
                           |   (the building) |
                           |                  |
   ARRIVE -> [PRE]---+---->| (apps inside)    |
                     |     |                  |
                     +-->[FORWARD]----[POST]--> LEAVE for somewhere else
                           |                  |
                           |     [LOCAL_OUT]--> [POST]--> LEAVE (we sent it)
                           +------------------+
```

- `[PRE]` is the bouncer at the front door (PREROUTING).
- `[FORWARD]` is the hallway bouncer who handles people just passing through.
- `[LOCAL_OUT]` is the bouncer at the inside of the back door (OUTPUT chain — packets you yourself made).
- `[POST]` is the bouncer at the actual back door (POSTROUTING — last chance before the packet hits the wire).

Hold this picture in your head. We will fill it in piece by piece.

## Tables × Chains

iptables is organized like a spreadsheet. The columns are **tables.** The rows are **chains.** Each cell in the spreadsheet is a clipboard the bouncer reads through.

There are five tables. Each table is a different bouncer crew with a different job.

### The filter table

The most common, the most basic, the one almost every rule lives in.

The filter table's job is one question: "should this packet live or die?"

It has three chains:

| Chain | What it watches |
|------|-----------------|
| `INPUT` | packets coming IN to your computer |
| `OUTPUT` | packets your computer is sending OUT |
| `FORWARD` | packets passing THROUGH your computer (on their way somewhere else — only matters if you turned on routing) |

If you only ever learn one table, learn this one. 90% of firewall rules live in `filter`.

### The nat table

The nat table's job is **rewriting addresses on envelopes.** That's what NAT means: Network Address Translation. Translation. Rewriting.

It has three chains:

| Chain | What it does |
|------|--------------|
| `PREROUTING` | rewrite the destination address right after the packet arrives |
| `POSTROUTING` | rewrite the source address right before the packet leaves |
| `OUTPUT` | rewrite destinations on packets your computer made (rare) |

If you've ever heard "port forwarding," that's `PREROUTING` rewriting the destination. If you've ever heard "my home router shares one IP with all my devices," that's `POSTROUTING` rewriting the source.

### The mangle table

The mangle table is for weird, advanced surgery on packets. It can change packet fields like the **TTL** (how many hops the packet can take before dying) or the **TOS** (priority hints for routers). It has chains for all five spots: `PREROUTING`, `INPUT`, `OUTPUT`, `FORWARD`, `POSTROUTING`.

You will rarely touch the mangle table unless you are doing fancy QoS or marking packets for policy routing.

### The raw table

The raw table is for telling netfilter "don't bother tracking this packet." It runs **before** connection tracking (we'll cover conntrack soon). Its chains are `PREROUTING` and `OUTPUT`.

If you have a super-high-traffic server and conntrack is slowing you down, you can `NOTRACK` certain packets in `raw` so they skip the bookkeeping.

### The security table

The security table is for SELinux/Mandatory Access Control labels on packets. Almost nobody touches it manually. It has `INPUT`, `OUTPUT`, `FORWARD` chains.

### The full picture

```
              filter   nat     mangle   raw      security
PREROUTING    -        yes     yes      yes      -
INPUT         yes      -       yes      -        yes
FORWARD       yes      -       yes      -        yes
OUTPUT        yes      yes     yes      yes      yes
POSTROUTING   -        yes     yes      -        -
```

A `-` means "this combo doesn't exist." A `yes` means "this is a real chain you can put rules on."

## Packet Flow Through netfilter

This is the famous flowchart. Every netfilter wiki page has a version of it. We are going to draw a simpler version that fits in your head.

Every packet that touches your computer takes one of three paths:

```
INCOMING from the wire
       |
       v
   [raw PREROUTING]
       |
       v
   [conntrack: figure out which connection]
       |
       v
   [mangle PREROUTING]
       |
       v
   [nat PREROUTING]      <- DNAT happens here (rewrite dest)
       |
       v
   ROUTING DECISION: is this packet for me or for someone else?
       |
       +----- FOR ME -----+              +----- FOR SOMEONE ELSE -----+
       |                  |              |                            |
       v                  v              v                            v
[mangle INPUT]      [filter INPUT]  [mangle FORWARD]            [filter FORWARD]
       |                  |              |                            |
       v                  v              v                            v
   delivered to local app             [mangle POSTROUTING]    [nat POSTROUTING]    <- SNAT/MASQUERADE here
                                            |
                                            v
                                       OUT to the wire

OUTGOING from local app
       |
       v
   ROUTING DECISION
       |
       v
   [raw OUTPUT]
       |
       v
   [conntrack: figure out which connection]
       |
       v
   [mangle OUTPUT]
       |
       v
   [nat OUTPUT]            <- rare DNAT for local apps
       |
       v
   [filter OUTPUT]
       |
       v
   [mangle POSTROUTING]
       |
       v
   [nat POSTROUTING]       <- SNAT/MASQUERADE
       |
       v
   OUT to the wire
```

The three important things to remember:

1. **DNAT is early.** It happens at PREROUTING, before the routing decision. So when you DNAT a packet to a different destination, the routing happens against the *new* destination.
2. **SNAT is late.** It happens at POSTROUTING, right before the packet leaves. The packet has already been routed and the source rewrite is the very last thing.
3. **Filter INPUT only sees packets going to local apps.** If a packet is forwarded through your machine, it never touches the INPUT chain. People get confused by this all the time.

## Match Modules

A rule is "if the packet looks like THIS, do THAT." The "looks like this" part is called a **match.** Iptables ships with a ton of match modules you can plug in with `-m NAME`.

Here are the most common ones, in plain English.

### -m state (the legacy one) and -m conntrack (the modern one)

These two are basically the same thing. `state` is older and simpler. `conntrack` is newer and has more options. They both ask, "what state is this connection in?"

States:

- `NEW` — first packet of a brand-new connection.
- `ESTABLISHED` — packet of a connection we've already seen replies on. Both sides have spoken.
- `RELATED` — a new connection but it's spawned by an existing one. Classic example: an FTP data connection that's spawned by an FTP control connection.
- `INVALID` — netfilter has no idea what this packet is. Often broken or spoofed.
- `UNTRACKED` — packet was marked NOTRACK in the raw table.

You'll see this rule on almost every Linux firewall, and it is basically magic:

```bash
# iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
```

That one rule is what lets your replies come back. Without it, you'd block your own browser's incoming HTTP responses.

### -m mark and -m connmark

Marks are sticky notes. You can put a number on a packet (`MARK`) or on a whole connection (`CONNMARK`) and check for it later. Useful for policy routing and complex setups.

```bash
# iptables -t mangle -A PREROUTING -p tcp --dport 22 -j MARK --set-mark 1
# iptables -A INPUT -m mark --mark 1 -j ACCEPT
```

### -m hashlimit

Rate limiting. "Don't let the same source send more than N packets per second." Way more flexible than the old `-m limit`.

```bash
# iptables -A INPUT -p icmp --icmp-type echo-request \
    -m hashlimit --hashlimit 1/sec --hashlimit-burst 5 \
    --hashlimit-name ping -j ACCEPT
```

That rule says, "let any one source send at most 1 ping per second, with a burst of 5."

### -m recent

Remembers IP addresses for a window of time. Used for port-knocking, brute-force protection, "ban this IP for an hour after they hit me too hard."

```bash
# iptables -A INPUT -p tcp --dport 22 -m recent --name SSH --update \
    --seconds 60 --hitcount 4 -j DROP
```

That rule says, "if the same IP made 4 SSH attempts in the last 60 seconds, drop them."

### -m string

Match a literal string anywhere in the packet payload. Crude content filtering. Don't use it for security (it's easy to bypass with encryption) but useful for marking traffic.

```bash
# iptables -A FORWARD -m string --string "BitTorrent" --algo bm -j DROP
```

### -m owner

Match based on which local user/group/process is sending the packet. Only works in OUTPUT/POSTROUTING because incoming packets don't have a local owner.

```bash
# iptables -A OUTPUT -m owner --uid-owner 1000 -p tcp --dport 25 -j REJECT
```

That blocks user 1000 from sending mail directly.

### -m geoip

Match based on the country a source/destination IP belongs to. Needs an external IP-to-country database. Useful for "block all traffic from country X." Not always shipped by default.

```bash
# iptables -A INPUT -m geoip --src-cc CN,RU -j DROP
```

### -m set (works with ipset)

Match against a set of IPs/ports stored in `ipset`. We'll cover ipset later — it's how you efficiently match against a list of 100,000 IPs without writing 100,000 rules.

```bash
# iptables -A INPUT -m set --match-set blocklist src -j DROP
```

### Many more

There's `-m multiport` (match multiple ports), `-m iprange` (match a range of IPs), `-m time` (match a time of day), `-m mac` (match a MAC address), `-m physdev` (match a bridge port), and many others. Run `man iptables-extensions` for the full list — it is long.

## Targets

A **target** is what happens when a rule matches. We've already seen `ACCEPT` and `DROP`. Here are the rest.

### The terminating targets (the rule decides what to do, packet stops here)

- **ACCEPT** — let the packet through. The bouncer stamps it and waves it on.
- **DROP** — silently throw it in the trash. The sender has no idea what happened. Their connection just hangs until they time out.
- **REJECT** — send the sender a polite "no" message. By default it's an ICMP "port unreachable." You can change it with `--reject-with`. The sender knows immediately that they were refused.
- **MASQUERADE** — rewrite the packet's source to be whatever IP the outgoing interface has right now. Used in `POSTROUTING` of the `nat` table when you have a dynamic IP (DHCP, DSL).
- **SNAT** — like MASQUERADE but with a fixed source IP. Use this if your outside IP doesn't change.
- **DNAT** — rewrite the destination. The classic "port forward to my internal server" target. Lives in `PREROUTING` of the `nat` table.
- **REDIRECT** — like DNAT but always to "this same machine, different port." Used to send traffic to a local proxy.

### The non-terminating targets (do something, then keep walking through more rules)

- **LOG** — write a kernel log message about this packet. Doesn't accept or drop. The packet keeps walking.
- **NFLOG** — like LOG but writes to a netlink socket so userspace tools (`ulogd`) can pick it up.
- **ULOG** — older version of NFLOG. Deprecated.
- **MARK** — slap a sticky note on the packet (an integer mark).
- **CONNMARK** — slap a sticky note on the whole connection.
- **TPROXY** — transparent proxy. Send the packet to a local socket without rewriting it. Used by Squid and similar proxies.
- **NOTRACK** — tell conntrack "don't track this packet." Used in the raw table for performance.

### The flow-control targets

- **RETURN** — pop back out to the chain that called this one. If you're in a user chain, return to the parent. If you're in a built-in chain, RETURN means "default policy applies."
- **(jump to a user chain)** — `-j MYCHAIN` jumps execution to your custom chain. When that chain RETURNs, you continue here.

### Quick reference

```
ACCEPT  - "you may pass"
DROP    - "you don't exist" (silent)
REJECT  - "you may not pass" (audible)
LOG     - "I am writing your name down" (passes through)
DNAT    - "your destination is now over there"
SNAT    - "your source is now over there"
MASQ    - "your source is now whatever my outside IP is"
RETURN  - "out of this chain, back to the parent"
```

## Connection Tracking (conntrack)

This is one of the most important and most confusing pieces. Stay with me.

### What conntrack does

Imagine the bouncer at the door wants to be smart. The bouncer wants to know, "is this packet the *first* packet of a new conversation, or is it part of a conversation that's already happening?"

Without conntrack, the bouncer can only look at the current packet. With conntrack, the bouncer has a **memory.** The kernel keeps a giant table of every active connection: source IP, destination IP, source port, destination port, protocol, state. When a new packet shows up, conntrack looks it up in the table and says, "ah, this belongs to connection #4392, which is in state ESTABLISHED."

This is what makes the magic rule work:

```bash
# iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
```

You don't have to write rules for "let in HTTP responses on a random port between 32768 and 60999." Conntrack already remembers that you sent out an HTTP request, so when the response arrives, conntrack matches it to that outgoing request and the packet's state is ESTABLISHED. You ACCEPT it. Done.

### The conntrack table

The kernel keeps the table at `/proc/net/nf_conntrack` (newer kernels) or `/proc/net/ip_conntrack` (older). You can also use the `conntrack` command:

```bash
# conntrack -L
tcp      6 431999 ESTABLISHED src=192.168.1.10 dst=8.8.8.8 sport=54321 dport=443 src=8.8.8.8 dst=203.0.113.5 sport=443 dport=54321 [ASSURED] mark=0 use=1
```

The numbers after `src=` and `dst=` show both directions of the connection: what you sent (outgoing tuple) and what should come back (return tuple). When NAT is involved, those will differ.

### State transitions

```
   (no entry yet)
        |
        |  first SYN packet seen
        v
       NEW
        |
        |  reply seen (SYN/ACK from the other side)
        v
   ESTABLISHED
        |
        |  related child connection started (e.g. FTP data)
        |  --> spawns a new entry in state RELATED
        |
        |  connection closes (FIN/RST or timeout)
        v
     (entry expires)
```

There's also INVALID (we don't recognize this packet at all) and UNTRACKED (we deliberately stopped tracking).

### Conntrack capacity and tuning

Conntrack has a maximum number of connections. By default it might be 65,536 or 262,144 depending on your distro. On a busy server you might need to raise it.

```bash
# sysctl net.netfilter.nf_conntrack_max
net.netfilter.nf_conntrack_max = 262144

# sysctl net.netfilter.nf_conntrack_max=1048576
# sysctl net.netfilter.nf_conntrack_count
net.netfilter.nf_conntrack_count = 4231
```

If conntrack fills up, **new connections get dropped.** You will see this in `dmesg`:

```
nf_conntrack: table full, dropping packet
```

Bad day. Raise `nf_conntrack_max` and probably `nf_conntrack_buckets` (the hash table size).

### Conntrack helpers

Some protocols are too clever to track without help. FTP is the classic one: an FTP client opens a control connection on port 21 and tells the server "send the file data on port 32145." Without help, conntrack would never know that port 32145 is "related" to the FTP control session.

So there are **conntrack helpers:** `nf_conntrack_ftp`, `nf_conntrack_sip`, `nf_conntrack_h323`, `nf_conntrack_pptp`, etc. These read the inside of the packets and create the right RELATED expectations.

```bash
# modprobe nf_conntrack_ftp
```

Modern kernels require you to opt in explicitly with `CT --helper ftp` rules in `raw` for security reasons.

## NAT Modes

NAT means rewriting addresses. There are several flavors.

### SNAT — Source NAT

You have a fixed outside IP, and you want all traffic from your inside network to look like it came from that fixed IP.

```bash
# iptables -t nat -A POSTROUTING -o eth0 -s 10.0.0.0/8 -j SNAT --to-source 203.0.113.5
```

"For every packet leaving via eth0 that came from 10.0.0.0/8, rewrite the source to 203.0.113.5."

### MASQUERADE — Source NAT for dynamic IPs

You have a DHCP/PPPoE outside IP that changes whenever your ISP feels like it. You can't hardcode the IP. So MASQUERADE looks up "what IP is on this interface right now" each time.

```bash
# iptables -t nat -A POSTROUTING -o eth0 -s 10.0.0.0/8 -j MASQUERADE
```

It's slower than SNAT (does a lookup per packet) but easier when your IP changes.

### DNAT — Destination NAT (port forwarding)

Public IP gets a packet on port 80, but the actual web server is at 10.0.0.5:8080 on your inside network.

```bash
# iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to-destination 10.0.0.5:8080
```

Don't forget you also need to ALLOW the forwarded traffic in the FORWARD chain:

```bash
# iptables -A FORWARD -p tcp -d 10.0.0.5 --dport 8080 -j ACCEPT
```

### Hairpin NAT (NAT loopback)

This is the weird one. You're inside your own network. You want to reach your own public IP (the one your DNS name resolves to). The packet has to bounce off your router and come back inside. By default this doesn't work, because the reply path is messed up.

The fix: SNAT the *source* on the way back in, so the internal server sees the traffic as coming from the router and replies to the router instead of directly to the inside client.

```bash
# iptables -t nat -A POSTROUTING -s 10.0.0.0/8 -d 10.0.0.5 -p tcp --dport 8080 -j MASQUERADE
```

If your home router supports it, it's usually called "NAT loopback" or "hairpinning."

## Saving and Loading Rules

Rules added with `iptables -A` live in memory. When you reboot, **they vanish.** This catches everybody at least once. The cure: save the rules to a file, and load them on boot.

### iptables-save and iptables-restore

```bash
# iptables-save > /etc/iptables/rules.v4
# ip6tables-save > /etc/iptables/rules.v6
```

`iptables-save` writes the current ruleset in a portable format. To reload:

```bash
# iptables-restore < /etc/iptables/rules.v4
```

By default `iptables-restore` flushes the existing rules first. Pass `-n` to merge instead of flush.

### iptables-persistent (Debian/Ubuntu)

There's a package that handles this for you:

```bash
# apt install iptables-persistent
# netfilter-persistent save
# netfilter-persistent reload
```

It loads `/etc/iptables/rules.v4` and `/etc/iptables/rules.v6` on boot.

### RHEL/CentOS

The classic config file is `/etc/sysconfig/iptables`. Modern Red Hat (8+) wants you to use `firewalld` or `nftables` instead, but the legacy file still works.

### nftables

`nft list ruleset > /etc/nftables.conf` saves. The `nftables.service` systemd unit loads `/etc/nftables.conf` on boot.

## Common Patterns

### Pattern 1: default DROP, explicit ALLOW

The "fortress" pattern. Block everything by default, allow only the things you need.

```bash
# Set defaults
# iptables -P INPUT DROP
# iptables -P FORWARD DROP
# iptables -P OUTPUT ACCEPT

# Always allow loopback (your computer talking to itself)
# iptables -A INPUT -i lo -j ACCEPT

# Always allow already-established connections
# iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Drop obviously bad packets
# iptables -A INPUT -m state --state INVALID -j DROP

# Allow SSH from a specific IP
# iptables -A INPUT -p tcp -s 203.0.113.7 --dport 22 -j ACCEPT

# Allow web
# iptables -A INPUT -p tcp --dport 80 -j ACCEPT
# iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Allow ICMP echo (ping)
# iptables -A INPUT -p icmp --icmp-type echo-request -j ACCEPT
```

**WARNING:** if you do this over SSH and forget the SSH allow rule, you will lock yourself out. Always test with a `at` command or a cron that resets the firewall in 5 minutes:

```bash
# echo 'iptables -F; iptables -P INPUT ACCEPT' | at now + 5 minutes
```

If you get locked out, the at job kicks in and resets you.

### Pattern 2: rate-limit pings

```bash
# iptables -A INPUT -p icmp --icmp-type echo-request \
    -m hashlimit --hashlimit 1/sec --hashlimit-burst 5 \
    --hashlimit-name ping -j ACCEPT
# iptables -A INPUT -p icmp --icmp-type echo-request -j DROP
```

### Pattern 3: SSH brute-force protection

```bash
# iptables -A INPUT -p tcp --dport 22 -m state --state NEW -m recent --name SSH --set
# iptables -A INPUT -p tcp --dport 22 -m state --state NEW -m recent --name SSH --update \
    --seconds 60 --hitcount 4 -j DROP
# iptables -A INPUT -p tcp --dport 22 -j ACCEPT
```

Three rules, in order:
1. Note every new SSH attempt.
2. If the same IP made 4+ attempts in the last 60 seconds, drop.
3. Otherwise allow.

### Pattern 4: port-knocking

Three "secret" port hits in order, then SSH opens.

```bash
# Start with two stages
# iptables -N STAGE1
# iptables -N STAGE2
# iptables -N DOOR

# Stage 1: hit port 7000
# iptables -A INPUT -p tcp --dport 7000 -m recent --set --name AUTH1 -j DROP

# Stage 2: hit port 8000 within 30s of port 7000
# iptables -A INPUT -p tcp --dport 8000 -m recent --rcheck --seconds 30 --name AUTH1 \
    -m recent --set --name AUTH2 -j DROP

# Door: hit SSH within 30s of port 8000 — allowed
# iptables -A INPUT -p tcp --dport 22 -m recent --rcheck --seconds 30 --name AUTH2 -j ACCEPT
```

You probably don't actually want port-knocking on a real server (it's security through obscurity). But it's a fun demo of `-m recent`.

### Pattern 5: log dropped packets (for debugging)

```bash
# iptables -A INPUT -j LOG --log-prefix "DROPPED: " --log-level 4
# iptables -A INPUT -j DROP
```

The LOG line lets the packet pass through. The DROP catches it. You'll see entries in `dmesg` or `/var/log/syslog`:

```
DROPPED: IN=eth0 OUT= MAC=00:11:22:... SRC=1.2.3.4 DST=203.0.113.5 LEN=60 TOS=0x00 ...
```

Be careful: logging every packet on a busy server fills disks fast. Add a hashlimit:

```bash
# iptables -A INPUT -m hashlimit --hashlimit 5/min --hashlimit-name droplog -j LOG --log-prefix "DROPPED: "
```

## ipset

`ipset` is the answer to "I have 50,000 bad IPs to block." Without ipset, you'd have 50,000 rules, and netfilter would walk all 50,000 for every packet. With ipset, you have **one** rule that points at a hash set, and the lookup is constant-time.

### Creating sets

```bash
# ipset create blocklist hash:ip
# ipset add blocklist 1.2.3.4
# ipset add blocklist 5.6.7.8
# ipset add blocklist 10.20.30.0/24
# ipset list blocklist
Name: blocklist
Type: hash:ip
Revision: 4
Header: family inet hashsize 1024 maxelem 65536
Size in memory: 248
References: 0
Members:
1.2.3.4
5.6.7.8
```

### Set types

| Type | What it stores |
|------|----------------|
| `hash:ip` | individual IPs |
| `hash:net` | networks (CIDR ranges) |
| `hash:ip,port` | IP + port pairs |
| `hash:net,port` | network + port pairs |
| `hash:ip,port,ip` | IP + port + dest IP |
| `bitmap:port` | a range of ports (very fast) |
| `list:set` | a set of sets (for grouping) |

### Using a set in iptables

```bash
# iptables -A INPUT -m set --match-set blocklist src -j DROP
```

That **one rule** matches against the entire set in O(1) hash lookup time.

### Saving sets across reboots

```bash
# ipset save > /etc/ipset.conf
# ipset restore < /etc/ipset.conf
```

You usually want to load the sets *before* iptables, so the rules don't reference missing sets.

## Switching to nftables

nftables is the successor. Same job, modern syntax. On most current distros, the `iptables` command actually writes nftables rules under the hood — that's the **iptables-nft** compatibility layer.

### Check which backend you're on

```bash
# iptables -V
iptables v1.8.7 (nf_tables)
```

If you see `(nf_tables)`, you're on the modern backend. If you see `(legacy)`, you're on the old one.

You can also check the binaries directly:

```bash
# update-alternatives --display iptables
iptables - auto mode
  link best version is /usr/sbin/iptables-nft
```

### Looking at nftables directly

```bash
# nft list ruleset
table inet filter {
    chain input {
        type filter hook input priority 0; policy drop;
        iif "lo" accept
        ct state established,related accept
        tcp dport 22 accept
    }
}
```

Already cleaner than iptables.

### Translating old iptables rules

`iptables-translate` reads an iptables command and prints the equivalent nft command without applying anything:

```bash
# iptables-translate -A INPUT -p tcp --dport 80 -j ACCEPT
nft 'add rule ip filter INPUT tcp dport 80 counter accept'
```

There's also `iptables-restore-translate` which reads an entire iptables-save dump and writes nftables rules.

### Don't mix backends

This is the classic gotcha. If you have rules in both `iptables-legacy` and `iptables-nft` at the same time, they all apply. The kernel runs both pipelines. You will see dropped packets that don't match either ruleset alone, because *the other backend* dropped them.

Pick one. Use `update-alternatives` or distro tools to lock it in.

## nftables Anatomy

### Tables, chains, rules

In nftables, the hierarchy is the same words but slightly different meaning:

```
family
  table
    chain
      rule (with match and verdict)
      ...
    chain
      ...
  table
    ...
```

A **table** is a named container for chains and other objects. You name tables yourself.

A **chain** is a list of rules. There are two kinds:
- **base chains** are hooked into netfilter at a specific point.
- **regular chains** are just functions you jump to.

### Hooks

Chains hook into netfilter at one of these hook points:

| Hook | Where |
|------|-------|
| `prerouting` | right after the packet arrives |
| `input` | packets bound for local apps |
| `forward` | packets passing through |
| `output` | packets from local apps |
| `postrouting` | right before the packet leaves |
| `ingress` | even earlier than prerouting (netdev family only) |

### Priorities

Multiple chains can hook the same point. Their priorities decide the order. iptables uses fixed priorities:

| iptables table | nft priority |
|----------------|--------------|
| raw | -300 |
| mangle | -150 |
| nat (PREROUTING/OUTPUT, dstnat) | -100 |
| filter | 0 |
| security | 50 |
| nat (POSTROUTING, srcnat) | 100 |

Lower numbers run first.

### Families

iptables had separate tools for IPv4 (`iptables`), IPv6 (`ip6tables`), bridge (`ebtables`), ARP (`arptables`). nftables collapses them all under one tool. You pick a family per table:

| Family | Covers |
|--------|--------|
| `ip` | IPv4 only |
| `ip6` | IPv6 only |
| `inet` | IPv4 and IPv6 together |
| `arp` | ARP packets |
| `bridge` | bridged Ethernet frames |
| `netdev` | very early hook, per-interface |

The `inet` family is the most common. Write your rules once and they apply to both v4 and v6.

### Example: build a basic firewall in nftables

```bash
# nft add table inet filter
# nft 'add chain inet filter input { type filter hook input priority 0; policy drop; }'
# nft 'add chain inet filter forward { type filter hook forward priority 0; policy drop; }'
# nft 'add chain inet filter output { type filter hook output priority 0; policy accept; }'

# nft add rule inet filter input iif lo accept
# nft add rule inet filter input ct state established,related accept
# nft add rule inet filter input ct state invalid drop
# nft add rule inet filter input tcp dport 22 accept
# nft add rule inet filter input tcp dport { 80, 443 } accept
# nft add rule inet filter input ip protocol icmp icmp type echo-request accept
```

Notice that you can match a *set* of ports inline: `tcp dport { 80, 443 }`. iptables can't do that without `-m multiport`.

### Loading from a file

You can write a config file and load it all at once:

```bash
# cat /etc/nftables.conf
#!/usr/sbin/nft -f
flush ruleset
table inet filter {
    chain input {
        type filter hook input priority 0; policy drop;
        iif lo accept
        ct state established,related accept
        tcp dport 22 accept
    }
}

# nft -f /etc/nftables.conf
```

The `flush ruleset` line at the top is a tradition: clear everything, then load fresh.

## ufw and firewalld

iptables and nftables are both powerful and both finicky. Most people don't want to write the rules by hand. That's where **frontends** come in. They give you a friendly command-line that generates the underlying iptables/nftables rules for you.

### ufw — Uncomplicated Firewall

Default on Ubuntu. Simple syntax. Generates iptables rules.

```bash
# ufw enable
# ufw default deny incoming
# ufw default allow outgoing
# ufw allow 22/tcp
# ufw allow from 203.0.113.7 to any port 22
# ufw allow http
# ufw allow https
# ufw status verbose
```

### firewalld

Default on RHEL/Fedora. Zone-based. Generates nftables rules (or iptables on older versions).

The idea is **zones.** You assign each network interface to a zone. Each zone has its own ruleset.

```bash
# firewall-cmd --get-default-zone
public

# firewall-cmd --list-all
public (active)
  target: default
  icmp-block-inversion: no
  interfaces: eth0
  sources:
  services: dhcpv6-client ssh

# firewall-cmd --permanent --add-service=http
# firewall-cmd --permanent --add-port=8080/tcp
# firewall-cmd --reload
```

`--permanent` writes to disk. Without it, the change is only in memory. `--reload` re-applies the on-disk config.

### NetworkManager firewall zones

NetworkManager can also assign interfaces to firewalld zones automatically:

```bash
# nmcli connection modify "Wired connection 1" connection.zone trusted
```

That tells NetworkManager "when this connection comes up, put the interface in the `trusted` zone."

## Common Errors

Verbatim error messages and what they mean.

### `iptables: No chain/target/match by that name.`

You used a target/match name that doesn't exist or isn't loaded. Common causes:
- Typo in `-j ACCPT` instead of `-j ACCEPT`.
- The kernel module isn't loaded (try `modprobe ipt_REDIRECT` for example).
- You're using a target that needs a specific table — `MASQUERADE` only works in `nat`.

### `iptables v1.8.7: can't initialize iptables table 'filter': Permission denied (you must be root)`

You forgot `sudo`. Only root can read or write iptables rules.

### `iptables: Bad rule (does a matching rule exist in that chain?).`

You tried to delete (`-D`) a rule by full specification, but the spec doesn't exactly match any existing rule. Even one missing space or a different argument order can cause this. Use `-L --line-numbers` and delete by line number instead:

```bash
# iptables -L INPUT --line-numbers
# iptables -D INPUT 3
```

### `iptables: Couldn't load match 'state': No such file or directory.`

Match module not available. Probably means the kernel module isn't loaded:

```bash
# modprobe xt_state
```

On modern kernels, `state` is provided by `xt_conntrack` automatically. If you're on a very stripped kernel (like an embedded device), the module might not be present.

### `iptables: Resource temporarily unavailable.`

Conntrack table full, or you're hitting a netlink socket limit. Check:

```bash
# sysctl net.netfilter.nf_conntrack_count
# sysctl net.netfilter.nf_conntrack_max
```

### `iptables-restore: line 17 failed`

Something on line 17 of your saved rules file is malformed or refers to a chain/table that doesn't exist. Run `iptables-restore --test < rules.v4` to validate without applying.

### `nft: Could not process rule: No such file or directory`

The table or chain doesn't exist yet. nftables won't auto-create them. You have to `nft add table` and `nft add chain` first.

### `nft: syntax error, unexpected end of file`

You're missing a closing brace `}` or a quoted command got chopped. nftables syntax is precise. Double-check braces and semicolons.

### `ip6tables-restore v1.8.7: getsockopt failed strangely: Operation not supported`

Trying to use ip6tables on a kernel without IPv6 netfilter compiled in. Rare but happens on tiny embedded systems.

### `iptables-legacy vs iptables-nft confusion: please use either iptables-legacy or iptables-nft, but not both`

You have rules in both backends. Pick one and clear the other:

```bash
# iptables-legacy -F
# iptables-legacy -X
# iptables-legacy -t nat -F
```

(Or do the same to `iptables-nft -F` if you want to keep legacy.)

## Hands-On

Roll up your sleeves. These commands are paste-and-run. **All of them need root** — prepend `sudo` if you're not already root, or use `sudo -i` to get a root shell.

### List the current ruleset

```bash
# iptables -L
Chain INPUT (policy ACCEPT)
target     prot opt source               destination

Chain FORWARD (policy ACCEPT)
target     prot opt source               destination

Chain OUTPUT (policy ACCEPT)
target     prot opt source               destination
```

Empty by default on most fresh systems. `(policy ACCEPT)` means the default is "let it through."

### List with verbose info, no DNS, line numbers

```bash
# iptables -L -v -n --line-numbers
Chain INPUT (policy ACCEPT 0 packets, 0 bytes)
num   pkts bytes target     prot opt in     out     source               destination
1        0     0 ACCEPT     all  --  lo     *       0.0.0.0/0            0.0.0.0/0
2        0     0 ACCEPT     all  --  *      *       0.0.0.0/0            0.0.0.0/0            ctstate RELATED,ESTABLISHED
```

`-v` adds packet/byte counters. `-n` skips DNS reverse-lookups (otherwise `-L` is **slow** because it tries to resolve every IP). `--line-numbers` is essential for deleting rules by number.

### List a different table

```bash
# iptables -t nat -L
# iptables -t mangle -L
# iptables -t raw -L
```

If you don't pass `-t`, you get the filter table (the default).

### Flush all rules in a chain or table

```bash
# iptables -F            # flush all chains in the filter table
# iptables -F INPUT      # flush only the INPUT chain
# iptables -t nat -F     # flush all chains in the nat table
```

WARNING: if your default policy is DROP and you flush, you cut off all access. Set policies to ACCEPT before flushing if you're remote.

### Delete user-defined chains

```bash
# iptables -X            # delete all empty user chains
# iptables -X MYCHAIN    # delete one specific user chain
```

You can't delete a chain that has rules in it or that's referenced by another rule. Flush it first.

### Set the default policy

```bash
# iptables -P INPUT DROP
# iptables -P FORWARD DROP
# iptables -P OUTPUT ACCEPT
```

Default policy applies when no rule in the chain matches. Choose carefully.

### Append (add) a rule to allow SSH

```bash
# iptables -A INPUT -p tcp --dport 22 -j ACCEPT
```

`-A INPUT` means append to the INPUT chain. `-p tcp` means TCP protocol. `--dport 22` means destination port 22. `-j ACCEPT` means jump to the ACCEPT target.

### The most important rule on the planet

```bash
# iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
```

This is what lets your replies come back in. Without it, you cannot browse the web, ssh out, or do basically anything network-related, even if your default OUTPUT policy is ACCEPT.

### Rate-limited ping

```bash
# iptables -A INPUT -p icmp --icmp-type echo-request \
    -m hashlimit --hashlimit 1/sec --hashlimit-burst 5 \
    --hashlimit-name ping -j ACCEPT
```

### Source-NAT for an internal subnet

```bash
# iptables -t nat -A POSTROUTING -o eth0 -s 10.0.0.0/8 -j MASQUERADE
```

This is what makes a "Linux router" work for a home network.

### Port forward to an internal server

```bash
# iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to-destination 10.0.0.5:8080
# iptables -A FORWARD -p tcp -d 10.0.0.5 --dport 8080 -j ACCEPT
```

Don't forget to enable IP forwarding:

```bash
# sysctl -w net.ipv4.ip_forward=1
```

And to make it persistent, add `net.ipv4.ip_forward = 1` to `/etc/sysctl.d/99-forwarding.conf`.

### Log dropped packets

```bash
# iptables -A INPUT -j LOG --log-prefix "DROPPED: "
```

You'll see entries in `dmesg`:

```
[ 1234.567890] DROPPED: IN=eth0 OUT= MAC=... SRC=1.2.3.4 DST=203.0.113.5 ...
```

### Brute-force protection on SSH

```bash
# iptables -A INPUT -p tcp --dport 22 -m state --state NEW \
    -m recent --name SSH --update --seconds 60 --hitcount 4 -j DROP
```

### Insert a rule at the top of a chain

```bash
# iptables -I INPUT 1 -p tcp --dport 22 -j ACCEPT
```

`-I INPUT 1` means "insert at position 1 in INPUT." Without the number it defaults to position 1.

### Delete a rule by exact spec

```bash
# iptables -D INPUT -p tcp --dport 22 -j ACCEPT
```

The spec must match exactly. Better:

### Delete a rule by line number

```bash
# iptables -L INPUT --line-numbers
Chain INPUT (policy DROP)
num  target  prot opt source       destination
1    ACCEPT  all  --  anywhere     anywhere    ctstate ESTABLISHED,RELATED
2    ACCEPT  all  --  anywhere     anywhere
3    ACCEPT  tcp  --  anywhere     anywhere    tcp dpt:ssh

# iptables -D INPUT 3
```

### Create a user-defined chain

```bash
# iptables -N MYCHAIN
# iptables -A INPUT -p tcp --dport 80 -j MYCHAIN
# iptables -A MYCHAIN -m string --string "evil" --algo bm -j DROP
# iptables -A MYCHAIN -j RETURN
```

### Rename a chain

```bash
# iptables -E OLDNAME NEWNAME
```

### Save current rules

```bash
# iptables-save > /etc/iptables/rules.v4
# ip6tables-save > /etc/iptables/rules.v6
```

### Restore rules

```bash
# iptables-restore < /etc/iptables/rules.v4
```

### IPv6 rules

```bash
# ip6tables -L
# ip6tables -A INPUT -p tcp --dport 22 -j ACCEPT
```

The `ip6tables` command is its own separate ruleset for IPv6. iptables is IPv4 only.

### ipset basics

```bash
# ipset create blocklist hash:ip
# ipset add blocklist 1.2.3.4
# ipset add blocklist 5.6.7.0/24    # error: hash:ip doesn't accept CIDR
# ipset list blocklist

# iptables -I INPUT 1 -m set --match-set blocklist src -j DROP
```

For CIDR blocks, use `hash:net`:

```bash
# ipset create badnets hash:net
# ipset add badnets 192.0.2.0/24
# iptables -I INPUT 1 -m set --match-set badnets src -j DROP
```

### Watch the conntrack table

```bash
# conntrack -L
tcp      6 431999 ESTABLISHED src=192.168.1.10 dst=8.8.8.8 sport=54321 dport=443 ...

# conntrack -L | wc -l
4231

# conntrack -F
# (flushes the entire table — disruptive! all existing connections lose state)

# conntrack -E
# (live event monitor — shows new/established/destroy events as they happen)
```

### nftables — list everything

```bash
# nft list ruleset
table inet filter {
    chain input {
        type filter hook input priority 0; policy accept;
    }
    chain forward {
        type filter hook forward priority 0; policy accept;
    }
    chain output {
        type filter hook output priority 0; policy accept;
    }
}
```

### nftables — list a specific table

```bash
# nft list table inet filter
```

### nftables — add a rule

```bash
# nft add rule inet filter input ip saddr 10.0.0.0/8 accept
```

### nftables — load from a file

```bash
# nft -f /etc/nftables.conf
```

### nftables — trace what's happening

```bash
# nft monitor trace
```

This streams a live trace of packets as they walk through the chains. Wonderful for debugging.

### Conntrack tunables

```bash
# sysctl net.netfilter.nf_conntrack_max
# sysctl net.netfilter.nf_conntrack_count
# sysctl net.nf_conntrack_max=1048576
```

### ufw essentials

```bash
# ufw status
Status: inactive

# ufw enable
Firewall is active and enabled on system startup

# ufw allow 22
Rule added
Rule added (v6)

# ufw status
Status: active
To                         Action      From
--                         ------      ----
22                         ALLOW       Anywhere
22 (v6)                    ALLOW       Anywhere (v6)
```

### firewalld essentials

```bash
# firewall-cmd --list-all
# firewall-cmd --permanent --add-service=http
# firewall-cmd --permanent --add-port=8080/tcp
# firewall-cmd --reload
```

### Translate iptables to nftables

```bash
# iptables-translate -A INPUT -p tcp --dport 80 -j ACCEPT
nft 'add rule ip filter INPUT tcp dport 80 counter accept'
```

## Common Confusions

### iptables-legacy vs iptables-nft

**Same command. Different backends.** On modern systems, the `iptables` binary is a wrapper that talks to either the old `xt_*` netfilter modules (legacy) or the new `nf_tables` engine (nft). They are *separate* rulesets in the kernel. If you have rules in both, both apply, and you'll lose your mind debugging.

Run `iptables -V` to see which one you're on. `nf_tables` means you're on the modern wrapper.

### Rule order matters — first match wins

Iptables walks the chain top to bottom. The **first matching rule decides.** If you put `-j ACCEPT` before `-j DROP`, the ACCEPT wins. This is the source of countless "but I added a drop rule!" bugs.

### Default DROP without saving rules locks you out on reboot

You set `-P INPUT DROP`, you flushed all the rules, you forgot to save. You reboot. Now nothing works.

The fix: save *before* setting DROP. Or use the `at`-in-5-minutes trick we mentioned. Or test on a console you can physically reach.

### conntrack INVALID vs NEW vs ESTABLISHED

- `NEW`: never seen this connection before. The first SYN.
- `ESTABLISHED`: we've seen replies in both directions.
- `INVALID`: doesn't fit any known state. Often spoofed, malformed, or out-of-order. Almost always safe to drop.

A common mistake is allowing INVALID along with ESTABLISHED,RELATED. Don't. Drop INVALID.

### NAT vs masquerade

NAT is the general concept (Network Address Translation). MASQUERADE is one specific way to do source NAT, designed for dynamic IPs. SNAT is the other way, for fixed IPs. They both rewrite the source. MASQUERADE just looks up "what's my IP right now?" each time.

### Rule numbering changes when you delete

If you delete rule 3, the rule that was number 4 becomes number 3. So deleting by number in a loop is dangerous: delete from highest number to lowest, or use `iptables-save` and edit the file.

### -A appends, -I inserts

`-A INPUT` adds to the *bottom* of the chain. `-I INPUT 1` inserts at the *top* (or at the position you specify). New rules at the bottom can be too late if earlier rules already match.

### Multiple jumps to a user chain count separately

If three different rules `-j MYCHAIN`, the chain runs three times for three different sets of packets. Each run is independent. If MYCHAIN does `-j RETURN`, it returns to whichever chain called it.

### filter INPUT vs FORWARD

`INPUT` is for packets going to *this machine.* `FORWARD` is for packets *passing through* this machine to somewhere else. These are completely different chains.

If you set up a router and only put rules in INPUT, your forwarded traffic isn't affected at all.

### --ctstate vs --state

`--state` is the old syntax (from `-m state`). `--ctstate` is the new syntax (from `-m conntrack`). They mean the same thing for the basic states (NEW, ESTABLISHED, RELATED, INVALID). Modern kernels deprecate `-m state` and want you to use `-m conntrack --ctstate ...`.

### Why ICMP is special

ICMP isn't TCP or UDP. It doesn't have ports. It has *types* (echo-request, echo-reply, destination-unreachable, time-exceeded, etc.). When you write iptables rules for ping, you use `--icmp-type echo-request`, not `--dport`.

You should usually allow at least these ICMP types:
- `echo-request` (incoming pings)
- `echo-reply` (replies to your outgoing pings)
- `destination-unreachable` (routers telling you "no")
- `time-exceeded` (how traceroute works)

Blocking all ICMP breaks a lot of stuff and is usually pointless.

### How to debug — use LOG

When a rule isn't matching the way you think, add a LOG before it:

```bash
# iptables -I INPUT 1 -p tcp --dport 22 -j LOG --log-prefix "SSH-DEBUG: "
```

Watch `dmesg -w` and try to connect. You'll see exactly what netfilter is seeing.

### Conntrack table size limit

If `nf_conntrack: table full` shows up in dmesg, **new connections are being dropped.** Raise `nf_conntrack_max`. On a busy server, 1 million is reasonable. Monitor `nf_conntrack_count` to see how close you are.

### ipset vs hash:ip vs hash:net

`ipset` is the tool. `hash:ip` is one set type that stores individual IPs. `hash:net` is another that stores CIDR ranges. They're not interchangeable — adding `1.2.3.0/24` to a `hash:ip` set fails because hash:ip wants single IPs only.

### nftables sets vs ipset

nftables has built-in sets, similar to ipset but native. You can write:

```bash
nft add rule inet filter input ip saddr { 1.2.3.4, 5.6.7.8, 9.10.11.0/24 } drop
```

You don't need a separate ipset for that anymore. Use ipset when you have giant lists you want to manage with a separate tool, or when you're stuck on iptables.

## Vocabulary

| Word | Plain English |
|------|--------------|
| netfilter | The kernel's packet-inspection machinery. The actual bouncers. |
| iptables | The legacy command-line tool that writes rules into netfilter. |
| ip6tables | iptables, but for IPv6 packets. |
| ebtables | Like iptables but for Ethernet frames in a bridge. |
| arptables | Like iptables but for ARP packets. |
| nftables | The modern replacement for all of the above, unified. |
| nft | The command-line tool for nftables. |
| libnftnl | The low-level library nft uses to talk to the kernel. |
| libmnl | The netlink helper library underneath libnftnl. |
| conntrack | The connection-tracking subsystem. Remembers active connections. |
| conntrackd | A daemon that replicates conntrack state to a peer (HA firewalls). |
| ipset | A subsystem for storing big sets of IPs/ports for efficient matching. |
| ufw | Uncomplicated Firewall — a friendly frontend that generates iptables rules. |
| firewalld | A daemon-based firewall manager with zones, common on RHEL. |
| firewall-cmd | The CLI for firewalld. |
| NetworkManager firewall zone | An interface attribute that auto-assigns it to a firewalld zone. |
| table | A top-level container in iptables/nftables (filter, nat, mangle, etc.). |
| chain | A list of rules attached to a hook point. |
| rule | One "if match then target" entry in a chain. |
| target | What happens when a rule matches: ACCEPT, DROP, etc. |
| jump | Going from one chain into another via `-j CHAINNAME`. |
| return | Coming back out of a user-defined chain. |
| accept | Let the packet through. |
| drop | Silently throw the packet away. |
| reject | Refuse the packet and tell the sender. |
| --reject-with | Specify which ICMP/TCP refusal message REJECT sends. |
| log | Write a kernel log message about the packet (non-terminating). |
| nflog | Like log but writes via netlink to userspace. |
| ulog | Older nflog, deprecated. |
| mark | Slap an integer on a packet for later matching. |
| connmark | Slap an integer on a whole connection. |
| tproxy | Transparent-proxy target — send to a local socket without rewriting. |
| notrack | Tell conntrack to skip this packet. |
| queue / NFQUEUE | Hand the packet to a userspace program for inspection. |
| filter table | Default table. Decides "should this packet live or die?" |
| INPUT | Filter chain for packets going to local apps. |
| OUTPUT | Filter chain for packets from local apps. |
| FORWARD | Filter chain for packets passing through. |
| PREROUTING | Chain that fires right after a packet arrives. |
| POSTROUTING | Chain that fires right before a packet leaves. |
| nat table | Table for rewriting addresses. |
| mangle table | Table for advanced packet edits (TTL, TOS, etc.). |
| raw table | Table for marking packets NOTRACK before conntrack. |
| security table | Table for SELinux/MAC labels on packets. |
| packet flow | The full path a packet takes through netfilter. |
| hook | A specific point in the kernel where chains can run (PRE_ROUTING, LOCAL_IN, FORWARD, LOCAL_OUT, POST_ROUTING). |
| priority | Determines the order of chains hooked to the same point. |
| family | nftables grouping: ip, ip6, inet, arp, bridge, netdev. |
| set | A named collection of values you can match against. |
| map | A named key-value store you can look up in a rule. |
| verdict | nftables word for "what to do" — equivalent to iptables target. |
| counter | Object that counts packets/bytes matching a rule. |
| quota | Counter that triggers when it hits a threshold. |
| limit | Match that drops once a rate is exceeded. |
| hashlimit | Like limit but per-hash-bucket — per-source rate limiting. |
| recent | Match module that remembers IPs across packets. |
| -m state | Legacy match for connection state. |
| -m conntrack | Modern match for connection state. |
| NEW | Conntrack state: first packet of a new connection. |
| ESTABLISHED | Conntrack state: replies seen in both directions. |
| RELATED | Conntrack state: a child connection of an existing one. |
| INVALID | Conntrack state: packet doesn't fit any known connection. |
| UNTRACKED | Conntrack state: marked NOTRACK. |
| ASSURED | Conntrack flag: connection is robustly established. |
| DNAT | Destination NAT — rewrite the destination address. |
| SNAT | Source NAT — rewrite the source address (fixed). |
| MASQUERADE | Source NAT for dynamic IPs — uses the interface's current IP. |
| REDIRECT | DNAT to "this machine, different port." |
| NETMAP | One-to-one rewrite of a whole subnet. |
| hairpin | NAT loopback — accessing your public IP from inside the LAN. |
| conntrack zone | A label that splits conntrack into independent regions. |
| conntrack helper | A module that helps track multi-flow protocols (FTP, SIP, etc.). |
| nf_conntrack_max | Sysctl for conntrack table capacity. |
| nf_conntrack_buckets | Sysctl for conntrack hash table size. |
| hashsize | Hash table size, often the same as nf_conntrack_buckets. |
| ipset hash:ip | Set type storing single IP addresses. |
| hash:net | Set type storing CIDR ranges. |
| hash:ip,port | Set type storing IP+port pairs. |
| hash:net,port | Set type storing network+port pairs. |
| list:set | Set type storing other sets (for grouping). |
| bitmap:port | Fast set type for a contiguous range of ports. |
| ipset family inet | IPv4 sets. |
| ipset family inet6 | IPv6 sets. |
| ipset save / restore | Tools to dump/load ipset state. |
| iptables-save format | The text format produced by iptables-save. |
| iptables-restore -n | Restore without flushing existing rules. |
| iptables-persistent | Debian/Ubuntu package that loads rules on boot. |
| netfilter-persistent | The systemd unit that iptables-persistent uses. |
| /etc/iptables/rules.v4 | Standard file path for saved IPv4 rules. |
| /etc/iptables/rules.v6 | Standard file path for saved IPv6 rules. |
| /etc/sysconfig/iptables | RHEL's standard saved-rules file. |
| /etc/nftables.conf | nftables config file loaded on boot. |
| /etc/sysctl.d/nf.conf | sysctl config file for netfilter tunables. |
| ipt_MASQUERADE | Kernel module that implements MASQUERADE target. |
| ipt_DNAT | Kernel module that implements DNAT target. |
| ipt_SNAT | Kernel module that implements SNAT target. |
| xt_state | Kernel module for `-m state`. |
| xt_conntrack | Kernel module for `-m conntrack`. |
| xt_recent | Kernel module for `-m recent`. |
| xt_hashlimit | Kernel module for `-m hashlimit`. |
| xt_string | Kernel module for `-m string`. |
| xt_owner | Kernel module for `-m owner`. |
| xt_set | Kernel module that links iptables to ipset. |
| xt_geoip | Kernel module for `-m geoip` (third-party). |
| xt_LOG | Kernel module for the LOG target. |
| xt_NFLOG | Kernel module for the NFLOG target. |
| xt_ULOG | Kernel module for the ULOG target. |
| modprobe iptable_nat | Load the nat-table support. |
| modprobe ip_conntrack | Load older-kernel conntrack. |
| modprobe nf_conntrack_ipv4 | Load IPv4 conntrack. |
| ip_forward | Sysctl flag that enables IPv4 packet forwarding. |
| sysctl net.ipv4.ip_forward | The full path of that flag. |
| sysctl net.ipv6.conf.all.forwarding | IPv6 forwarding flag. |
| rp_filter | Reverse-path filter sysctl — drops spoofed sources. |
| /proc/net/ip_conntrack | Old path to the conntrack table. |
| /proc/net/nf_conntrack | New path to the conntrack table. |
| /proc/sys/net/netfilter/ | Directory full of netfilter tunables. |
| network namespace | A separate network stack inside one kernel. |
| veth pair | Two virtual interfaces tied together — one in each namespace. |
| bridge | A virtual switch in software. |
| br_netfilter | Kernel module that runs iptables on bridged frames. |
| conntrack-tools | The package providing the `conntrack` command. |
| -j NFQUEUE | Target that hands the packet to a userspace queue. |
| scapy | Python library for crafting packets to test firewalls. |
| hping3 | Tool for crafting and sending custom packets. |
| tcpdump | Packet sniffer for verifying what's actually arriving. |
| nftables JSON output | `nft --json` produces machine-readable JSON. |

## Try This

Start small. Pretend you're learning to drive — empty parking lot first.

1. **Make sure you're not going to lock yourself out.** If you're SSH'd into a remote box, set up the at-in-5-minutes safety net:

   ```bash
   # echo 'iptables -F; iptables -P INPUT ACCEPT; iptables -P OUTPUT ACCEPT' | at now + 5 minutes
   ```

2. **Look at the empty firewall.**

   ```bash
   # iptables -L -v -n --line-numbers
   ```

3. **Add a single rule and watch the counter increase.**

   ```bash
   # iptables -A INPUT -p icmp --icmp-type echo-request -j ACCEPT
   # ping -c 5 your-host-ip   (from another machine)
   # iptables -L -v -n
   ```

   You should see the packet count rising on the rule.

4. **Drop pings instead.**

   ```bash
   # iptables -F INPUT
   # iptables -A INPUT -p icmp --icmp-type echo-request -j DROP
   # ping -c 5 your-host-ip   (from another machine)
   ```

   The pings should hang. Cancel them.

5. **Save your work.**

   ```bash
   # iptables-save > /tmp/my-rules.v4
   # cat /tmp/my-rules.v4
   ```

6. **Wipe and restore.**

   ```bash
   # iptables -F
   # iptables -L
   # iptables-restore < /tmp/my-rules.v4
   # iptables -L
   ```

7. **Look at conntrack live.**

   ```bash
   # conntrack -E
   ```

   In another terminal, `curl http://example.com`. Watch the events fly by.

8. **Build the same rule in nftables.**

   ```bash
   # nft add table inet myfilter
   # nft 'add chain inet myfilter input { type filter hook input priority 0; }'
   # nft add rule inet myfilter input ip protocol icmp drop
   # nft list ruleset
   ```

9. **Translate an existing iptables rule.**

   ```bash
   # iptables-translate -A INPUT -p tcp --dport 22 -m state --state NEW -j ACCEPT
   ```

   See how the modern syntax is shorter.

10. **Cancel the at job once you're sure things work.**

    ```bash
    # atq
    # atrm <jobnumber>
    ```

## Where to Go Next

When this sheet feels comfortable, move on:

- `cs networking iptables` — the dense reference version. Same content, less hand-holding.
- `cs networking icmp` — what those weird `--icmp-type` values actually mean.
- `cs networking ip` — the protocol the bouncer is filtering.
- `cs networking tcp` and `cs networking udp` — what's inside the envelopes.
- `cs security firewall-design` — how to plan a firewall, not just write rules.
- `cs security firewalld` — the daemon-based frontend.
- `cs security network-defense` — defense-in-depth around the firewall.
- `cs ramp-up icmp-eli5` — the ELI5 of ICMP if any of this confused you.
- `cs ramp-up tcp-eli5` — same for TCP.
- `cs ramp-up ip-eli5` — same for IP itself.
- `cs ramp-up linux-kernel-eli5` — where netfilter actually lives.

## See Also

- networking/iptables
- networking/icmp
- networking/ip
- networking/tcp
- networking/udp
- security/firewall-design
- security/firewalld
- security/network-defense
- ramp-up/icmp-eli5
- ramp-up/tcp-eli5
- ramp-up/ip-eli5
- ramp-up/linux-kernel-eli5

## References

- netfilter.org — the official project home with packet-flow diagrams, manuals, and the canonical wiki.
- "Linux Firewalls" by Steve Suehring — the long-standing book on iptables in production.
- "Linux iptables Pocket Reference" by Gregor N. Purdy — a tiny O'Reilly book that fits in a backpack and covers every flag.
- nftables.org and the nftables wiki — the canonical reference for the modern syntax.
- `man iptables` — built-in manual page for the legacy tool.
- `man iptables-extensions` — the long list of match modules and targets.
- `man ip6tables` — IPv6 variant.
- `man iptables-restore` and `man iptables-save` — the persistence tools.
- `man nft` — the nftables CLI reference.
- `man conntrack` — the conntrack tool reference.
- Linux kernel documentation under `Documentation/networking/nf_conntrack-sysctl.rst` for tunables.
- RFC 2663, "IP Network Address Translator (NAT) Terminology and Considerations" — the formal definitions.
- RFC 5382, "NAT Behavioral Requirements for TCP" — what a "good" NAT box should do.

### Version notes

- **iptables-nft is the default** on RHEL 8+, Debian 11+, Ubuntu 21.04+. The `iptables` command translates to nftables under the hood.
- **nftables** has been in mainline since Linux 3.13 (January 2014). It became production-ready around 2017 and is the recommended path for new firewalls.
- **ipset** has been in mainline since 2005 and is rock-solid.
- **conntrack** has been in mainline since 2.4 (2001). Conntrack helpers like `nf_conntrack_ftp` were tightened up in newer kernels — explicit `CT --helper ftp` rules in the raw table are now required for security.
- **ebtables/arptables** have been merged into nftables's `bridge` and `arp` families. The standalone tools are deprecated.
