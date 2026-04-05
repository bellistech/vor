# Network Scanning — Deep Dive into TCP/IP Probing, Evasion, and Detection

> This document supplements `sheets/offensive/network-scanning.md` with protocol-level internals, detection mathematics, and advanced techniques tested on the CEH v13 exam (Module 03).

## 1. TCP/IP Stack Fingerprinting Internals

Operating systems implement TCP/IP differently. Active OS fingerprinting tools like nmap send crafted probes and compare responses against a signature database. The key discriminators:

### Initial TTL Values

| OS Family | TTL | Common Window Size |
|-----------|-----|-------------------|
| Linux 2.4–6.x | 64 | 5840, 14600, 29200 |
| Windows 10/11/Server | 128 | 8192, 65535 |
| macOS / FreeBSD | 64 | 65535 |
| Cisco IOS | 255 | 4128 |
| Solaris | 255 | 8760 |

### DF (Don't Fragment) Bit

- **Linux:** Almost always set (DF=1). Uses Path MTU Discovery.
- **Windows:** Set by default (DF=1). Older versions (pre-XP SP2) sometimes cleared it.
- **FreeBSD/macOS:** Set (DF=1).
- **Solaris:** Not always set. Varies by version.

### TCP Options Order and Values

Nmap's fingerprinting sends six TCP probes (T1–T7) and one UDP probe (U1). The response analysis checks:

1. **Options order** — The sequence in which TCP options appear in the SYN/ACK. Linux typically sends: MSS, SACK permitted, Timestamps, NOP, Window Scale. Windows historically sends: MSS, NOP, Window Scale, NOP, NOP, SACK permitted.

2. **Window scale factor** — Linux commonly uses 7 (multiplier 128). Windows uses 8 (multiplier 256). This determines the maximum receive window.

3. **MSS (Maximum Segment Size)** — Usually 1460 on Ethernet. Some OS set unusual values.

4. **Timestamp behavior** — Whether TCP timestamps are present, their initial value, and increment rate. Linux timestamps increment at ~1000 Hz; Windows at ~100 Hz (when enabled, which is not the default).

5. **SACK (Selective Acknowledgment)** — Almost universally supported now, but the position in the options list differs.

### Nmap Probe Specifics

Nmap sends the following probe categories for OS detection:

- **SEQ probes (S1–S6):** Six TCP SYN packets to an open port with varying options and window sizes. Analyzes ISN (Initial Sequence Number) increments, IP ID generation, and timestamp patterns.
- **ECN probe (ECE):** SYN with ECE and CWR flags to an open port. Tests explicit congestion notification support.
- **T2–T7 probes:** Packets with unusual flag combinations (e.g., no flags, SYN+FIN+URG+PSH) to open and closed ports. Measures how the stack handles protocol violations.
- **U1 probe:** UDP packet to a closed port. Analyzes the ICMP port unreachable response (IP TTL, IP ID, ICMP error quoting behavior).

### ISN (Initial Sequence Number) Analysis

ISN generation reveals OS identity and security posture:

- **Truly random:** Modern Linux, Windows 10+. Passes statistical randomness tests.
- **Time-dependent:** Older systems. ISN increments correlate with time, making prediction feasible.
- **Constant or predictable:** Embedded devices, old firmware. Vulnerable to TCP sequence prediction attacks.

Nmap computes the GCD (Greatest Common Divisor) of ISN differences across the six SEQ probes and classifies the generation algorithm.

## 2. Idle Scan (Zombie) Technique In Depth

The idle scan (`nmap -sI`) is a side-channel attack that determines port state without sending any packet from the attacker's real IP.

### Prerequisites

The zombie host must:
- Be idle (no other traffic that would increment its IP ID).
- Use a **globally incrementing IP ID** counter (not random, not zero). Many modern OS use random IP IDs, making them unsuitable.
- Be reachable from both the attacker and target.

### Step-by-Step Mechanism

**Phase 1 — Probe the zombie's current IP ID:**

```
Attacker → SYN/ACK → Zombie
Zombie → RST (IP ID = X) → Attacker
```

The zombie was not expecting a SYN/ACK so it replies RST. The attacker records IP ID = X.

**Phase 2 — Send spoofed SYN to target:**

```
Attacker → SYN (src = zombie's IP) → Target
```

The attacker spoofs the source address as the zombie.

**Phase 3 — Target responds to the zombie:**

If the target port is **open**:
```
Target → SYN/ACK → Zombie
Zombie → RST (IP ID = X+1) → Target    [zombie increments its IP ID]
```

If the target port is **closed**:
```
Target → RST → Zombie
Zombie does nothing                      [no IP ID increment]
```

If the target port is **filtered**:
```
Target → (nothing) → Zombie
Zombie does nothing                      [no IP ID increment]
```

**Phase 4 — Probe the zombie's IP ID again:**

```
Attacker → SYN/ACK → Zombie
Zombie → RST (IP ID = ?) → Attacker
```

- **IP ID = X+2:** Port is **open** (zombie sent one extra RST to the target, incrementing once, plus this probe increments once more).
- **IP ID = X+1:** Port is **closed or filtered** (zombie only incremented for this probe).

### Finding Suitable Zombies

```bash
# Check if a host has incremental IP ID
sudo nmap -O -v 10.0.0.50 | grep "IP ID Sequence"

# Look for "Incremental" — suitable zombie
# "Randomized" or "All zeros" — not suitable

# hping3 — watch IP ID increments
sudo hping3 -S -p 80 -r 10.0.0.50
# -r shows IP ID increments between replies; look for +1 pattern
```

### Detection

Idle scans are detectable at the zombie, not at the attacker. The zombie sees unsolicited SYN/ACKs from the target (for open ports). IDS on the zombie's network may flag this anomaly.

## 3. Scan Detection Algorithms

### Threshold-Based Detection

The simplest approach: flag a source IP that connects to more than N ports (or hosts) within T seconds.

**Snort threshold example:**
```
threshold gen_id 1, sig_id 1000001, type both, track by_src, count 25, seconds 5
```

This fires when a source hits 25 ports in 5 seconds.

**Limitations:**
- Slow scans (`-T1`, `--max-rate 1`) stay below the threshold.
- Distributed scans from multiple sources each stay below per-source thresholds.
- Legitimate services (load balancers, monitoring) can trigger false positives.

### Connection Tracking (Stateful Analysis)

More sophisticated IDS engines track TCP state machines:

1. **SYN without completion:** A SYN followed by RST (half-open scan) or no further packets. A high ratio of incomplete handshakes indicates scanning.

2. **Invalid flag combinations:** FIN without prior SYN, XMAS tree flags, NULL packets. These never appear in legitimate traffic.

3. **Sequential port access:** Legitimate clients connect to one or two ports. A source touching ports in sequence (1, 2, 3, ...) or hitting many distinct ports is almost certainly scanning.

4. **One-to-many pattern:** A single source contacting port 80 on 200 hosts in a /24 within seconds is a horizontal scan (sweep).

### TRW (Threshold Random Walk) Algorithm

A probabilistic model that classifies remote hosts as scanners or benign based on connection outcomes:

- Each successful connection (SYN/ACK received) shifts the hypothesis toward "benign."
- Each failed connection (RST or timeout) shifts toward "scanner."
- A likelihood ratio is maintained. When it crosses an upper threshold, the host is classified as a scanner. When it crosses a lower threshold, it is classified as benign.

**Parameters:**
- Detection probability P_d (e.g., 0.99)
- False positive probability P_f (e.g., 0.01)
- Upper threshold eta_1 = (1 - P_f) / P_d
- Lower threshold eta_0 = P_f / (1 - P_d)

This approach catches scans much faster than fixed thresholds because scanners inherently have a high failure rate (most ports are closed).

## 4. Mathematical Analysis of Scan Timing and Detection

### Scan Duration vs Detection Probability

Let:
- R = scan rate (packets per second)
- T_w = IDS detection window (seconds)
- N_t = IDS threshold (connection attempts triggering an alert)
- P = total ports to scan
- D = scan duration

Basic relationship:

```
D = P / R
```

Detection occurs when the number of probes in any window T_w exceeds N_t:

```
Detection condition: R * T_w >= N_t
Evasion condition:   R < N_t / T_w
```

**Example:** IDS window = 60 seconds, threshold = 20 connections. To evade:

```
R < 20 / 60 = 0.33 packets/sec
```

Scanning all 65535 TCP ports at 0.33 pps takes:

```
D = 65535 / 0.33 ≈ 198,590 seconds ≈ 55 hours
```

### Nmap Timing Templates Mapped to Rates

| Template | Parallelism | Probe Timeout | Inter-probe Delay | Typical Rate |
|----------|-------------|---------------|-------------------|-------------|
| T0 (Paranoid) | Serial | 5 min | 5 min | ~0.003 pps |
| T1 (Sneaky) | Serial | 15 sec | 15 sec | ~0.067 pps |
| T2 (Polite) | Serial | 10 sec | 0.4 sec | ~2.5 pps |
| T3 (Normal) | Parallel | 10 sec | 0 | ~100–1000 pps |
| T4 (Aggressive) | Parallel | 1.25 sec | 0 | ~1000–5000 pps |
| T5 (Insane) | Parallel | 0.3 sec | 0 | ~5000+ pps |

### Decoy Effectiveness

With D decoy addresses plus the real address, an analyst examining logs sees D+1 source IPs. If all appear equally likely, the probability of correctly identifying the real scanner is:

```
P(correct identification) = 1 / (D + 1)
```

With `nmap -D RND:10`, P = 1/11 ≈ 9.1%. However, TTL analysis and reverse path verification can eliminate decoys that lack valid routes.

### Fragmentation Analysis

When nmap uses `-f`, each probe is split into 8-byte IP fragments. A 20-byte TCP header requires 3 fragments. The first fragment contains source/destination ports; reassembly is needed to inspect flags. Some older firewalls and IDS inspect only the first fragment or fail to reassemble, allowing the scan to pass.

Modern systems perform full reassembly before inspection, making simple fragmentation less effective. Overlapping fragments (where later fragments overwrite earlier ones) can still confuse some implementations.

## 5. IPv6 Scanning Challenges

IPv6 fundamentally changes the scanning landscape:

### Address Space Problem

A standard /64 subnet contains 2^64 (18.4 quintillion) addresses. Exhaustive scanning is impossible:

```
At 1 million packets/sec:
2^64 / 10^6 = 1.84 * 10^13 seconds ≈ 584,942 years
```

### Discovery Techniques for IPv6

Since brute-force is infeasible, alternative approaches are required:

1. **DNS enumeration:** AAAA record lookups, zone transfers, reverse DNS sweeping of known allocation patterns.

2. **Multicast discovery:** IPv6 mandates multicast. Sending to `ff02::1` (all-nodes) on the local link reveals all IPv6 hosts.
   ```bash
   ping6 -c 2 ff02::1%eth0
   nmap -6 --script=targets-ipv6-multicast-echo
   ```

3. **Address pattern analysis:** Many organizations assign IPv6 addresses with predictable patterns (e.g., embedding the IPv4 address, using low-numbered host parts like ::1, ::2). Tools like `ipv6-attack-toolkit` generate candidate lists.

4. **NDP (Neighbor Discovery Protocol):** The IPv6 equivalent of ARP. Local-link NDP solicitation reveals neighbors:
   ```bash
   nmap -6 --script=ipv6-multicast-mld-list
   ip -6 neigh show
   ```

5. **Traffic sniffing:** Passively capture IPv6 traffic to build a host inventory.

### Nmap IPv6 Support

```bash
# Basic IPv6 scan
nmap -6 -sS fe80::1%eth0

# IPv6 ping sweep (multicast)
nmap -6 --script=targets-ipv6-multicast-echo --script-args newtargets -sn

# OS detection works but has a smaller signature database
nmap -6 -O 2001:db8::1
```

### IPv6-Specific Evasion

- **Extension headers:** IPv6 allows chaining extension headers (Hop-by-Hop, Routing, Fragment, etc.). Crafted chains can confuse firewalls that do not fully parse them.
- **Flow labels:** The 20-bit flow label field can be used to evade per-flow rate limiting.
- **Fragmentation:** IPv6 fragmentation is handled by endpoints (not routers), using a Fragment extension header. The minimum MTU is 1280 bytes, and fragments below this can cause issues for some IDS.

## Prerequisites

- Understanding of the TCP three-way handshake and TCP state machine (SYN, SYN/ACK, ACK, FIN, RST).
- IP header fields: TTL, IP ID, DF flag, protocol number.
- TCP header fields: flags, window size, options, sequence numbers.
- Basic nmap usage and command-line syntax.
- Root/sudo access for raw socket operations (SYN scan, OS detection, idle scan).
- Familiarity with Wireshark or tcpdump for packet analysis.
