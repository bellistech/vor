# DoS/DDoS Attacks — Deep Dive

> This document supplements `sheets/offensive/dos-ddos-attacks.md` with in-depth technical analysis of SYN flood mechanics, amplification factor calculations, botnet C2 communication, DDoS traffic modeling, and legal considerations. For authorized security testing, red team exercises, and educational study only.

## Prerequisites

- TCP/IP fundamentals (three-way handshake, IP headers, TCP state machine)
- Understanding of UDP, ICMP, and DNS at the packet level
- Familiarity with BGP routing basics
- Network traffic capture and analysis (Wireshark, tcpdump)
- Content from `sheets/offensive/dos-ddos-attacks.md`

## 1. SYN Flood Mechanics and SYN Cookie Algorithm

### 1.1 The TCP Three-Way Handshake Under Attack

Normal TCP connection establishment:

```
Client                    Server
  |--- SYN (seq=x) -------->|    Server allocates TCB (Transmission Control Block)
  |<-- SYN-ACK (seq=y, -----|
  |    ack=x+1)              |    TCB sits in SYN_RECEIVED state
  |--- ACK (ack=y+1) ------>|    Connection moves to ESTABLISHED
```

During a SYN flood, the attacker sends thousands of SYN packets with spoofed source IPs. The server allocates a TCB (typically 280-300 bytes) for each and places it in the backlog queue. Since the spoofed hosts never respond with ACK, these half-open connections persist until the SYN timeout (typically 75 seconds, with retransmissions extending to several minutes).

```
Backlog queue capacity (Linux default): 128-1024 entries
TCB memory per entry: ~280 bytes
SYN timeout (with retries): ~63-75 seconds (exponential backoff: 1+2+4+8+16+32s)
Attack rate needed to fill queue: backlog_size / timeout ≈ 1024/75 ≈ 14 SYN/s

At even modest rates (10,000 SYN/s), the queue is perpetually full.
```

### 1.2 SYN Cookie Algorithm (Bernstein, 1996)

SYN cookies eliminate server-side state during the handshake by encoding connection parameters into the Initial Sequence Number (ISN) of the SYN-ACK.

**Encoding the ISN:**

```
ISN (32 bits) is constructed as:

Bits 31-27 (5 bits):  t mod 32
                      t = slowly incrementing timestamp counter (every 64 seconds)

Bits 26-24 (3 bits):  MSS index
                      Encodes the server's selected MSS from a table of 8 values
                      (e.g., 0=536, 1=1024, 2=1460, 3=1360, 4=4312, ...)

Bits 23-0  (24 bits): MAC = hash(server_secret, src_ip, src_port, dst_ip, dst_port, t)
                      Cryptographic hash truncated to 24 bits
```

**Validation on ACK receipt:**

```
1. Server receives ACK with ack_num = ISN + 1
2. Subtract 1 to recover ISN
3. Extract t from bits 31-27
4. Verify t is recent (within last 2 intervals = 128 seconds)
5. Recompute MAC using extracted t and connection 4-tuple
6. Compare computed MAC with bits 23-0 of ISN
7. If match → reconstruct TCB using encoded MSS, complete handshake
```

**Trade-offs:**

```
Advantages:
- Zero server state until handshake completes
- Legitimate connections still succeed during attack
- Transparent to well-behaved clients

Limitations:
- Only 8 MSS values (3-bit encoding) — may select suboptimal MSS
- No TCP options preserved: window scaling, SACK, timestamps lost
  (because there is no TCB to store them in)
- 24-bit MAC: 1 in 16.7 million false positive rate per attempt
  (acceptable given the need for correct 4-tuple + valid timestamp)
- Slight CPU overhead for hash computation per SYN
```

**Linux implementation:**

```
# Enable (usually enabled by default)
sysctl -w net.ipv4.tcp_syncookies=1

# Behavior: SYN cookies activate only when the SYN backlog overflows
# This preserves full TCP features for normal operation
# and falls back to cookies only under attack conditions

# Relevant kernel parameters
net.ipv4.tcp_max_syn_backlog = 4096    # backlog queue size
net.ipv4.tcp_synack_retries = 2        # reduce from default 5
net.core.somaxconn = 4096              # listen() backlog
```

## 2. Amplification Factor Calculations

Amplification attacks exploit connectionless protocols (UDP) where the response is significantly larger than the request. The attacker spoofs the victim's IP as the source address.

### 2.1 Bandwidth Amplification Factor (BAF)

```
BAF = (Response size in bytes) / (Request size in bytes)

Effective amplification = BAF × number_of_reflectors × request_rate
```

### 2.2 Protocol-Specific Analysis

**DNS Amplification:**

```
Request:  DNS ANY query for a domain with large zone
          ~60 bytes (DNS header + query)

Response: Full zone dump — A, AAAA, MX, NS, TXT, SOA, DNSKEY, RRSIG records
          ~3,400 bytes (varies by zone)

BAF = 3400 / 60 ≈ 56x (theoretical max)
Typical BAF: 28-54x

Example calculation:
  Attacker bandwidth: 1 Gbps
  BAF: 50x
  Attack volume: 1 × 50 = 50 Gbps directed at victim

Mitigation: Disable open recursion, rate-limit ANY queries,
            Response Rate Limiting (DNS RRL)
```

**NTP Amplification (monlist):**

```
Request:  NTP monlist command (MON_GETLIST_1)
          ~234 bytes (NTP control message)

Response: List of up to 600 recent clients
          Each entry: 72 bytes, up to 6 packets of 468 bytes each
          Total: ~100 × 482 bytes ≈ 48,200 bytes (varies by server activity)

BAF = 48200 / 234 ≈ 206x (with active server)
Commonly cited: 556x (peak observed)

Why so large: The monlist response contains the last 600 IP addresses
that contacted the NTP server — a single small request triggers a
multi-packet response.

Mitigation: Upgrade to NTP 4.2.7+ (monlist removed),
            use 'noquery' in ntp.conf, restrict default settings
```

**Memcached Amplification:**

```
Request:  UDP GET or STATS command
          ~15 bytes

Response: Cached data — can be up to 1 MB per key, multiple keys
          ~750,000 bytes (depends on stored data)

BAF = 750000 / 15 = 50,000x (theoretical)
Observed in the wild: 10,000-51,000x

This is the highest known amplification factor.
The 1.35 Tbps GitHub attack (2018) used memcached amplification.

Key factors:
- Memcached default: listens on UDP port 11211 with no authentication
- Attackers can SET large values, then trigger GET from spoofed source

Mitigation: Disable UDP (-U 0), bind to localhost,
            firewall port 11211, use SASL authentication
```

**SSDP Amplification:**

```
Request:  M-SEARCH * HTTP/1.1 (UPnP discovery)
          ~90 bytes

Response: Device description XML with all services
          ~2,700 bytes (varies by device complexity)

BAF = 2700 / 90 ≈ 30x

Commonly exploited devices: home routers, IoT, media servers
SSDP runs on UDP port 1900

Mitigation: Disable UPnP on edge devices,
            block port 1900 at network boundary
```

### 2.3 Comparative Summary

```
Protocol     Port    BAF (typical)  BAF (max)     Attack Record
────────────────────────────────────────────────────────────────
DNS          53      28-54x         ~70x          ~400 Gbps
NTP          123     20-200x        556x          ~400 Gbps
Memcached    11211   10,000x        51,000x       1.35 Tbps
SSDP         1900    30x            ~31x          ~100 Gbps
CharGen      19      358x           ~359x         (legacy)
SNMP         161     6x             ~6.3x         (legacy)
LDAP         389     46-55x         ~70x          ~500 Gbps
CLDAP        389     56-70x         ~70x          ~500 Gbps
```

## 3. Botnet C2 Communication Analysis

### 3.1 C2 Channel Architectures

**IRC-based (First Generation):**

```
Bot → DNS lookup → IRC server → Join #channel → PRIVMSG commands

Characteristics:
- Centralized: single IRC server or small network
- Commands sent as IRC messages in a control channel
- Easy to monitor once channel is identified
- Takedown: shut down IRC server or channel

Detection indicators:
- Outbound connections to TCP 6667/6697
- IRC protocol signatures (NICK, JOIN, PRIVMSG, PING/PONG)
- Multiple hosts connecting to same non-standard IRC server
- Binary talking IRC (no human-readable nicks or messages)

Example C2 traffic:
  :bot NICK drone_38a2f
  :bot JOIN #ctrl
  :master PRIVMSG #ctrl :!udp 192.0.2.1 80 120
  (attack 192.0.2.1 port 80 for 120 seconds)
```

**HTTP-based (Second Generation):**

```
Bot → HTTP(S) GET/POST → C2 web server → JSON/encoded commands

Characteristics:
- Blends with normal web traffic
- Uses standard ports (80/443)
- Can hide behind legitimate-looking domains
- Often uses HTTPS to encrypt C2 traffic

Detection indicators:
- Periodic beaconing (regular intervals ± jitter)
- HTTP requests to newly registered domains
- POST requests with encoded/encrypted bodies
- Unusual User-Agent strings or missing standard headers
- Low entropy domain names or IP-only URLs

Beacon analysis:
  Interval: time between callbacks (e.g., 60s, 300s)
  Jitter: randomization factor (e.g., ±20%)
  Sleep pattern: (interval × (1 ± jitter_factor))
  Regularity in timing is a strong detection signal
```

**P2P-based (Third Generation):**

```
Bot ↔ Bot ↔ Bot ↔ Bot (mesh/overlay network)

Characteristics:
- No single point of failure
- Commands propagate through peer network
- Much harder to take down — no central server
- Examples: GameOver Zeus, Hajime, Mozi

Detection indicators:
- Unusual peer-to-peer protocol traffic
- High volume of connections to diverse IPs
- Custom protocols on non-standard ports
- Encrypted traffic with no legitimate service correlation
```

**Domain Generation Algorithms (DGA):**

```
Both bot and attacker run same algorithm:
  domain = hash(seed + date + counter) + TLD

Example (simplified):
  import hashlib
  def dga(date, seed="malware123", count=1000):
      domains = []
      for i in range(count):
          h = hashlib.md5(f"{seed}{date}{i}".encode()).hexdigest()
          domain = h[:12] + ".com"
          domains.append(domain)
      return domains

  # 2026-04-05 generates: a3f8b2c1d4e5.com, 7c9a2b1f3e8d.com, ...
  # Attacker registers 1 of 1000 domains each day
  # Bot tries all 1000 until it finds the live one

Detection:
- NXDomain rate: bots generate many failed lookups
- Lexical analysis: DGA domains have high entropy, no dictionary words
- Clustering: many hosts querying same NXDomain patterns
- ML classifiers: character frequency, bigram analysis, length distribution
```

**Fast Flux:**

```
Single Flux:
  Domain → rapidly changing A records (TTL: 30-300 seconds)
  Each A record points to a different compromised host (proxy)
  Proxy forwards traffic to actual C2 backend

Double Flux:
  Both NS records AND A records rotate rapidly
  Even the authoritative nameservers are compromised hosts

Detection:
- Very low TTL values
- Large number of unique A records over time
- A records pointing to diverse ASNs and geolocations
- NS records changing frequently (double flux)
```

### 3.2 C2 Traffic Analysis Methodology

```
Step 1: Capture — Full packet capture or NetFlow at egress points
Step 2: Protocol identification — Classify by port, protocol, DPI
Step 3: Beacon detection — Statistical analysis of timing intervals
        Autocorrelation, FFT for periodic signals
Step 4: Domain analysis — Reputation, age, registrar, WHOIS privacy
        Passive DNS for resolution history
Step 5: Payload inspection — Entropy analysis (encrypted vs cleartext)
        Known C2 framework signatures (Cobalt Strike, Metasploit)
Step 6: Correlation — Link multiple bots to same C2 infrastructure
        Shared domains, IPs, certificates, JA3/JA3S fingerprints
```

## 4. DDoS Attack Traffic Modeling

### 4.1 Volumetric Attack Capacity Estimation

```
Attack bandwidth = Σ (bot_bandwidth × utilization_factor)

Example botnet calculation:
  Botnet size: 50,000 bots
  Average bot bandwidth: 10 Mbps (residential connections)
  Utilization factor: 0.7 (not all bots active, shared connections)

  Raw capacity = 50,000 × 10 Mbps × 0.7 = 350 Gbps

With amplification:
  Amplification factor: 50x (DNS)
  Spoofed request rate per bot: 1 Mbps

  Attack volume = 50,000 × 1 Mbps × 50 = 2.5 Tbps
```

### 4.2 Protocol Attack Rate Modeling

```
SYN flood resource exhaustion:

  Target backlog queue: Q (entries)
  SYN timeout with retries: T (seconds)
  Minimum attack rate to fill queue: R = Q / T

  With Q=4096, T=63s:
  R = 4096 / 63 ≈ 65 SYN/s (trivially achievable)

  SYN packet size: 40-60 bytes (no payload)
  Bandwidth needed: 65 × 60 = 3,900 bytes/s ≈ 31.2 Kbps

  SYN floods are effective at extremely low bandwidth —
  they exhaust state, not bandwidth.
```

### 4.3 Application Layer Attack Modeling

```
HTTP flood capacity estimation:

  Target server capacity: C requests/sec (e.g., 10,000 rps)
  Average request processing time: t seconds
  Concurrent connection limit: L

  Saturation rate = L / t

  Example:
    L = 10,000 connections (Apache MaxClients)
    t = 0.5s per request for /search endpoint
    Saturation rate = 10,000 / 0.5 = 20,000 rps

    But if attacker targets expensive endpoint (t = 5s):
    Saturation rate = 10,000 / 5 = 2,000 rps

  Slowloris efficiency:
    One attacker host with 1,000 sockets
    Each socket holds one connection indefinitely
    Against Apache with MaxClients=256:
    Only 256 sockets needed to fully deny service
    Single laptop can take down a misconfigured Apache server
```

### 4.4 Multi-Vector Attack Composition

```
Modern DDoS attacks combine vectors simultaneously:

Phase 1 (Volumetric):  NTP amplification, 200 Gbps
  Purpose: Saturate upstream links, force provider response

Phase 2 (Protocol):    SYN flood, 50 Mpps
  Purpose: Exhaust firewall/LB state tables

Phase 3 (Application): HTTP POST flood + Slowloris
  Purpose: Overwhelm any remaining application capacity

The defender must handle all three layers simultaneously.
Each layer requires different mitigation techniques.
Attackers switch vectors when one is mitigated.
```

## 5. Legal Considerations

### 5.1 United States: Computer Fraud and Abuse Act (CFAA)

```
18 U.S.C. § 1030

Key provisions for DoS/DDoS:
  § 1030(a)(5)(A): Knowingly causing damage to a protected computer
                    by transmitting a program, information, code, or command

  "Protected computer": Any computer used in or affecting interstate
                        or foreign commerce (effectively all internet-connected systems)

  "Damage": Impairment to integrity or availability of data, program,
            system, or information

Penalties:
  First offense: Up to 10 years imprisonment, fines
  Subsequent offense: Up to 20 years
  If serious bodily harm results: Up to life imprisonment
  Attempted attacks are also prosecutable

Key cases:
  - United States v. Hutchins (2017) — Mirai botnet
  - United States v. Bukoski (2019) — booter services
  - Operation Power Off (2018) — 15 DDoS-for-hire sites seized
```

### 5.2 United Kingdom: Computer Misuse Act (CMA) 1990

```
Key sections:
  § 1: Unauthorised access to computer material (up to 2 years)
  § 3: Unauthorised acts with intent to impair operation of computer
       (up to 10 years)
  § 3A: Making, supplying, or obtaining articles for use in offences
        under §1 or §3 (up to 2 years)

§ 3 is the primary provision for DoS/DDoS prosecution:
  "A person is guilty of an offence if he does any unauthorised act
   in relation to a computer with intent to impair the operation of
   any computer, prevent or hinder access to any program or data,
   or impair the operation of any program or the reliability of any data."

§ 3A covers tool distribution:
  Creating or distributing DDoS tools (LOIC, booter panels)
  is itself an offence even without conducting an attack

Amendments:
  Serious Crime Act 2015 extended § 3 penalties to 10 years
  (previously 5 years)
```

### 5.3 Authorized Testing Framework

```
Requirements for lawful DDoS testing:

1. Written authorization from system owner (scope, duration, methods)
2. Defined scope — target IPs, ports, attack types, intensity limits
3. Coordination with ISP/hosting provider (prevent upstream response)
4. Incident response plan in place
5. Testing window agreed upon
6. Real-time communication channel between tester and target team
7. Kill switch / immediate stop capability
8. No collateral impact on shared infrastructure or third parties
9. All traffic sourced from owned/authorized IPs (no spoofing)
10. Results documented and shared with authorizing party

Standards:
  - PTES (Penetration Testing Execution Standard) — DoS testing section
  - OWASP Testing Guide — denial of service test cases
  - NIST SP 800-115 — Technical Guide to Information Security Testing
  - PCI DSS requirement 11.3 — penetration testing methodology
```
