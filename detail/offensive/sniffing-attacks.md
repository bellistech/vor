# Sniffing Attacks — Deep Dive into Network Traffic Interception

> This document provides detailed technical background for network sniffing concepts covered in the cheat sheet. For quick command reference, see `sheets/offensive/sniffing-attacks.md`.

## Prerequisites

- Understanding of OSI Layer 2 (Data Link) and Layer 3 (Network) operations
- Familiarity with Ethernet frame structure and IP packet headers
- Basic knowledge of ARP, DHCP, and DNS protocols
- Linux networking fundamentals (interfaces, routing, iptables)
- Lab environment with VMs or containers for safe practice (never on production networks)

## 1. ARP Protocol Vulnerabilities

ARP (Address Resolution Protocol) resolves IPv4 addresses to MAC addresses on a local network segment. Its design predates modern security concerns, making it inherently vulnerable.

### Stateless and Unauthenticated

ARP has two critical design flaws that enable spoofing:

**Stateless operation.** Hosts maintain an ARP cache but do not track whether they sent a request. Any host can send a gratuitous ARP reply at any time, and recipients will update their cache without verification. There is no request-reply correlation — a reply is accepted even if no request was made.

**No authentication.** ARP messages carry no credentials, signatures, or validation mechanisms. Any device on the broadcast domain can claim any IP-to-MAC mapping. The protocol provides no way to verify the identity of the sender.

### Gratuitous ARP

A gratuitous ARP is a broadcast where the sender IP and target IP are the same. Legitimate uses include:

- Announcing a new IP address after boot or interface change
- Detecting IP conflicts (if a reply comes back, the IP is already in use)
- Updating caches after a failover (e.g., VRRP/HSRP virtual IP migration)

Attackers exploit this by sending gratuitous ARPs with a forged sender MAC, causing all hosts on the segment to associate the attacker's MAC with the target IP.

### ARP Cache Timing

Most operating systems set ARP cache timeouts between 60 seconds and 20 minutes. An attacker must send poisoned ARP replies at intervals shorter than the cache timeout to maintain the spoofed mapping. Typical intervals:

- Linux default ARP timeout: ~60 seconds (`/proc/sys/net/ipv4/neigh/default/gc_stale_time`)
- Windows default: ~15-45 seconds (varies by version)
- macOS default: 20 minutes (`net.link.ether.inet.max_age`)

Tools like `arpspoof` and `ettercap` automatically re-send poisoned entries at appropriate intervals.

### ARP Spoofing Detection

Detection methods include:

- **Static ARP entries** for critical hosts (gateway, DNS server) — prevents cache poisoning but does not scale.
- **arpwatch/arpalert** — monitors ARP traffic and alerts on MAC/IP pairing changes.
- **Dynamic ARP Inspection (DAI)** on managed switches — validates ARP packets against DHCP snooping binding table.
- **IDS signatures** — detect anomalous ARP traffic patterns (multiple replies, gratuitous ARP floods).

## 2. CAM Table Sizing and Overflow Calculations

### CAM Table Architecture

A switch's Content Addressable Memory (CAM) table maps MAC addresses to physical ports. When a frame arrives, the switch looks up the destination MAC in the CAM table to determine the egress port. If the MAC is not found (unknown unicast), the switch floods the frame to all ports except the ingress — behaving like a hub.

### Table Sizing by Switch Class

| Switch Class       | Typical CAM Size   | Overflow Time (macof) |
|--------------------|--------------------|----------------------|
| Unmanaged/SOHO     | 1,024 - 4,096      | < 10 seconds         |
| Managed access     | 8,192 - 16,384     | 10-30 seconds        |
| Enterprise access  | 32,768 - 128,000   | 30-120 seconds       |
| Data center        | 128,000 - 512,000+ | Minutes; often mitigated |

### Overflow Mechanics

`macof` generates approximately 155,000 random MAC-address packets per second on a gigabit link. For a switch with a 16,384-entry CAM table:

```
Overflow time = table_size / packet_rate
             = 16,384 / 155,000
             ≈ 0.1 seconds
```

In practice, some entries expire and are replaced, so sustained flooding is needed. The switch may also implement protections:

- **Port security** limits the number of MACs learned per port.
- **Storm control** rate-limits broadcast/unknown-unicast traffic.
- **MAC move detection** alerts when a MAC appears on multiple ports.

### Aging and Expiry

CAM table entries have an aging timer (typically 300 seconds on Cisco). Entries not refreshed within this window are purged. During a flood attack, legitimate entries are evicted by random entries, and the switch cannot re-learn them because the table remains full.

## 3. SSL/TLS Interception

### SSL Stripping in Detail

SSL stripping works by intercepting the initial HTTP connection (before the TLS upgrade) and maintaining two separate connections:

```
Victim <--HTTP--> Attacker <--HTTPS--> Server
```

The attacker downgrades the victim's connection to HTTP while maintaining a legitimate HTTPS connection to the server. This requires a MITM position (typically via ARP spoofing).

### mitmproxy for TLS Interception

`mitmproxy` is a more capable alternative to `sslstrip` for TLS interception:

1. **Transparent proxy mode** — intercepts connections without client configuration.
2. **CA certificate injection** — generates certificates on-the-fly signed by the mitmproxy CA.
3. **Scripting** — Python scripts can modify requests/responses in transit.

For full TLS interception (not just stripping), the attacker's CA certificate must be trusted by the victim. Methods include:

- Social engineering the user to install the CA cert
- Compromising the system and adding to the trust store
- Exploiting enterprise MDM to push the CA cert
- Using a compromised or misissued CA certificate (rare, high-impact)

### Certificate Pinning Bypass

Certificate pinning binds a host to a specific certificate or public key, defeating CA-based interception. Bypass techniques (primarily for mobile app testing):

- **Frida/Objection** — runtime hooking to disable pinning checks in Android/iOS apps.
- **Magisk/Xposed modules** — system-level SSL pinning bypass on rooted Android.
- **Patching the APK/IPA** — decompile, remove pinning code, recompile.

These require control over the client device and are relevant to mobile security testing, not network-level attacks.

### HSTS Limitations

HTTP Strict Transport Security (HSTS) prevents SSL stripping for known domains by instructing browsers to always use HTTPS. Limitations for attackers:

- **Preload lists** are compiled into browsers — cannot be bypassed via network attacks.
- **First-visit vulnerability** — HSTS is set via a response header, so the first connection can be stripped (unless preloaded).
- **Subdomain bypass** — if `includeSubDomains` is not set, subdomains may be strippable.
- **NTP attacks** — manipulating time to expire HSTS entries (theoretical, requires additional MITM).

## 4. Network Tap vs SPAN Port for Legitimate Capture

### SPAN (Switched Port Analyzer) / Mirror Port

SPAN ports are configured on managed switches to copy traffic from one or more source ports to a designated monitoring port.

**Advantages:**
- No additional hardware required.
- Can be configured remotely (RSPAN for remote monitoring).
- Flexible — can mirror specific ports, VLANs, or traffic directions.

**Limitations:**
- Shared switch backplane — under heavy load, mirrored traffic may be dropped.
- Cannot capture Layer 1 errors (runt frames, CRC errors, etc.).
- Typically limited to one or two SPAN sessions per switch.
- Monitoring port must handle combined ingress+egress bandwidth.
- Adds CPU load to the switch.

### Network Tap

A network tap is a passive hardware device inserted inline on a network cable. It copies all traffic to monitoring ports without affecting the production link.

**Advantages:**
- Sees all traffic including errors and malformed frames.
- Zero impact on network performance — no CPU load, no dropped packets.
- Passive (non-powered taps work even during power failure for copper).
- Tamper-evident — physical access is visible.

**Limitations:**
- Requires physical access and a brief link interruption to install.
- Cost per tap point.
- Aggregation taps may need to handle full-duplex traffic (2x bandwidth).

**Types:**
- **Passive copper tap** — no power, uses signal splitting (slight attenuation).
- **Active/regeneration tap** — powered, regenerates signal, can aggregate.
- **Fiber tap** — splits optical signal, very low loss.
- **Aggregation tap** — combines full-duplex into single monitoring stream.

### Choosing Between Them

| Factor              | SPAN               | Network Tap         |
|---------------------|---------------------|---------------------|
| Packet fidelity     | May drop under load | All packets, all errors |
| Cost                | Free (switch feature) | $200-$5,000+ per tap |
| Setup               | Remote CLI config   | Physical installation |
| Production impact   | Possible CPU load   | None                |
| Best for            | Ad-hoc, temporary   | Continuous, forensic |

## 5. BPF Filter Compilation and Performance

### Berkeley Packet Filter Architecture

BPF (Berkeley Packet Filter) provides in-kernel packet filtering. When a capture filter is applied, the BPF program runs in kernel space and only matching packets are copied to userspace. This dramatically reduces the cost of packet capture.

### Filter Compilation

BPF filters written in the tcpdump/libpcap syntax are compiled into BPF bytecode — a simple virtual machine instruction set. The compilation process:

1. **Parse** the human-readable expression (e.g., `tcp port 80`).
2. **Compile** to BPF bytecode instructions (load, jump, return).
3. **Optimize** the bytecode (dead code elimination, branch optimization).
4. **Inject** the program into the kernel via `setsockopt(SO_ATTACH_FILTER)`.

To inspect compiled bytecode:

```
tcpdump -d 'tcp port 80'
```

This outputs the BPF instructions, showing how the filter evaluates each packet field.

### Performance Considerations

**Kernel-level filtering.** BPF runs before packets reach userspace, so filtered-out packets never incur the cost of context switches or memory copies. On a 10 Gbps link, this can be the difference between capturing at line rate and dropping packets.

**Filter complexity.** More complex filters compile to more BPF instructions. Each packet is evaluated against the program, so excessively complex filters (many OR clauses, nested conditions) can reduce throughput. In practice, even complex filters handle multi-gigabit rates on modern hardware.

**cBPF vs eBPF.** Classic BPF (cBPF) is used by tcpdump/libpcap. Extended BPF (eBPF) is a modern Linux kernel framework that generalizes BPF for tracing, networking, and security. eBPF programs can be attached to various kernel hooks, offering far more capability than packet filtering alone. Tools like `bpftrace` and XDP (eXpress Data Path) leverage eBPF for high-performance network processing.

### Common BPF Filter Patterns

```
# Efficient: specific protocol and port
tcp port 443

# Less efficient but still fast: host-based
host 10.0.0.5 and tcp port 80

# Complex: multiple conditions
(tcp[13] & 0x02 != 0) and (dst net 10.0.0.0/8)    # SYN to 10.x

# Capture only TCP flags
tcp[tcpflags] & (tcp-syn|tcp-fin) != 0             # SYN or FIN

# VLAN-tagged traffic
vlan and host 192.168.1.1

# Specific payload bytes (use sparingly — expensive)
tcp[20:4] = 0x47455420                              # "GET " in TCP payload
```

### Filter Testing

Always test filters before deploying on production captures:

```
# Verify filter syntax
tcpdump -d 'your filter here'          # dump BPF instructions

# Test against existing pcap
tcpdump -r existing.pcap -c 10 'your filter here'

# Count matches without full output
tcpdump -r existing.pcap 'your filter here' | wc -l
```
