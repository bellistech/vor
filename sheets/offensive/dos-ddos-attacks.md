> For authorized security testing, red team exercises, and educational study only.

# DoS/DDoS Attacks (CEH Module 10)

Denial of Service and Distributed Denial of Service attack vectors, tools, detection, and countermeasures.

## DoS vs DDoS

| Aspect         | DoS                        | DDoS                                  |
|----------------|----------------------------|---------------------------------------|
| Source         | Single host                | Multiple hosts (botnet)               |
| Scale          | Limited by attacker BW     | Aggregate bandwidth of all bots       |
| Traceability   | Easier to trace/block      | Hard — distributed, often spoofed     |
| Infrastructure | One machine                | Botnet (thousands to millions of bots)|
| Mitigation     | IP block, rate limit       | Scrubbing centers, anycast, CDN       |

## Attack Categories

```
Layer 7 — Application    HTTP flood, Slowloris, RUDY         (requests/sec)
Layer 4 — Protocol        SYN flood, ACK flood, Smurf        (packets/sec)
Layer 3 — Volumetric      UDP flood, amplification attacks   (bits/sec)
```

- **Volumetric**: saturate target bandwidth
- **Protocol**: exhaust state tables (firewalls, load balancers)
- **Application (L7)**: exhaust server resources (CPU, memory, connections)

## Volumetric Attacks

```
Attack                  Vector                    Amplification Factor
──────────────────────────────────────────────────────────────────────
UDP Flood               Random UDP → target       1x (raw bandwidth)
ICMP Flood              Echo requests → target    1x
DNS Amplification       ANY query, spoofed src    28–54x
NTP Amplification       monlist, spoofed src      556x
Memcached Amplification UDP port 11211            10,000–51,000x
SSDP Amplification      M-SEARCH, spoofed src     30x
```

**Amplification pattern**: small request with spoofed source IP to reflector, large response sent to victim.

```
Attacker → spoofed request → Reflector (DNS/NTP/memcached)
                                   ↓ amplified response
                                Victim
```

## Protocol Attacks

### SYN Flood
```
Attacker sends high volume of SYN packets (spoofed source IPs)
Target allocates TCB for each half-open connection
Backlog queue fills → legitimate connections refused

# SYN flood with hping3
hping3 -S --flood -V -p 80 <target>
hping3 -S --flood --rand-source -p 80 <target>    # random spoofed IPs
```

### ACK Flood
```
# Bypass stateless firewalls that allow established traffic
hping3 -A --flood -p 80 <target>
```

### Other Protocol Attacks
```
Ping of Death       Oversized ICMP (>65,535 bytes via fragmentation)
Smurf Attack        ICMP echo to broadcast address, spoofed source = victim
                    All hosts on subnet reply → victim flooded
Fragmentation       Overlapping/malformed fragments exhaust reassembly buffers
TCP State Exhaust   Open legitimate connections, hold them open
```

## Application Layer Attacks

### HTTP Flood
```
# GET flood — request heavy pages/endpoints
GET /search?q=randomstring HTTP/1.1
# POST flood — submit large form data repeatedly
POST /login HTTP/1.1
Content-Length: 10000
```

### Slowloris
```
# Hold connections open by sending partial HTTP headers slowly
# Never completes the request → server keeps connection open

# Using slowloris.py
slowloris <target> -p 80 -s 500    # 500 sockets

Sends:  GET / HTTP/1.1\r\n
        X-header: value\r\n        ← sent every ~15s, never sends \r\n\r\n
```

### RUDY (R-U-Dead-Yet)
```
# Sends POST with large Content-Length, then sends body 1 byte at a time
POST /form HTTP/1.1
Content-Length: 100000
<sends 1 byte every 10 seconds>
```

### Slow Read
```
# Complete the request normally but read the response very slowly
# Advertise tiny TCP window size → server holds buffer
TCP Window Size: 1 byte
```

### SSL/TLS Exhaustion
```
# SSL handshake is CPU-intensive on server side
# Repeatedly initiate and drop SSL handshakes
# THC-SSL-DOS: renegotiation attack
thc-ssl-dos <target> 443
```

## Botnet Architecture

```
C2 Channel Types:
──────────────────
IRC-based        Bot joins IRC channel, receives commands
HTTP-based       Bot polls C2 web server for instructions
P2P              No central server, bots relay commands peer-to-peer
Domain Flux      DGA (Domain Generation Algorithm) — bots generate
                 pseudorandom domains daily, attacker registers one
Fast Flux        Rapidly changing DNS A records (single-flux)
                 or NS + A records (double-flux), short TTL

Bot Lifecycle:
1. Initial infection (phishing, exploit kit, drive-by)
2. C2 callback (beacon/registration)
3. Secondary payload download
4. Idle/standby (await commands)
5. Attack execution
6. Update/migration (new C2, new payloads)
```

## DDoS-as-a-Service

```
Booter / Stresser Services:
- Web-based panels, subscription model ($10–$500/month)
- Leverage amplification reflectors and rented botnets
- Advertise as "stress testing" — used for attacks
- Often accept cryptocurrency
- Law enforcement takedowns: Operation Power Off (Europol)
```

## Tools

```
Tool          Type              Notes
─────────────────────────────────────────────────────────────────
hping3        Packet crafter    SYN/ACK/UDP floods, spoofing
LOIC          L4/L7 flood       GUI, TCP/UDP/HTTP, "voluntary botnet" mode
HOIC          L7 flood          HTTP flood with booster scripts
Slowloris     L7 slow           Holds connections with partial headers
GoldenEye     L7 HTTP           HTTP flood (GET/POST), keep-alive abuse
Torshammer    L7 slow POST      Slow POST through Tor for anonymity
THC-SSL-DOS   SSL exhaustion    SSL renegotiation abuse
```

### hping3 Examples
```bash
# SYN flood on port 80
hping3 -S --flood -p 80 <target>

# UDP flood
hping3 --udp --flood -p 53 <target>

# ICMP flood
hping3 --icmp --flood <target>

# Spoofed source
hping3 -S --flood --rand-source -p 80 <target>

# Specific packet size
hping3 -S --flood -p 80 -d 120 <target>
```

## Detection

```
Traffic Baselines       Establish normal patterns, alert on deviation
NetFlow / sFlow         Analyze flow data for volume anomalies
                        Top talkers, protocol distribution, flow duration
Anomaly Detection       Sudden spike in SYN packets (SYN:SYN-ACK ratio)
                        Unusual protocol distribution
                        Traffic from unexpected geolocations
                        High packet rate with uniform size
SNMP Monitoring         Interface utilization, packet counters
IDS/IPS Signatures      Known attack tool signatures
                        Rate-based rules
```

```bash
# Quick NetFlow analysis indicators
# SYN flood: high SYN count, low SYN-ACK ratio
# Amplification: high inbound UDP from port 53/123/11211
# Slowloris: many connections in ESTABLISHED from few IPs, low throughput
```

## Countermeasures

```
Technique               Layer    Effect
───────────────────────────────────────────────────────────────
SYN Cookies             L4       No state until handshake completes
Rate Limiting           L4/L7    Cap requests per source IP/subnet
Connection Limits       L4       Max concurrent connections per IP
Anycast Routing         L3       Distribute traffic across PoPs
CDN / Scrubbing         L3-L7    Filter attack traffic at edge
RTBH (Blackholing)      L3       Null-route victim IP at upstream
FlowSpec BGP            L3-L4    Push filter rules via BGP to edge routers
BCP38 (RFC 2827)        L3       Ingress filtering — block spoofed source IPs
BCP84 (RFC 3704)        L3       Extends BCP38, multi-homed networks
Web App Firewall        L7       Block malicious request patterns
CAPTCHA / JS Challenge  L7       Filter bot traffic from legitimate users
Geo-blocking            L3       Block traffic from non-relevant regions
```

### SYN Cookies
```
Server encodes SYN state into the initial sequence number (ISN):
ISN = hash(src_ip, src_port, dst_ip, dst_port, secret) + timestamp
No TCB allocated until valid ACK received with correct ISN+1

# Enable on Linux
sysctl -w net.ipv4.tcp_syncookies=1
```

### Mitigation Services
```
Cloudflare      Anycast CDN, L3-L7 DDoS protection, free tier available
Akamai Prolexic On-demand/always-on scrubbing, BGP re-route
AWS Shield      Standard (free, L3/L4) and Advanced ($3k/mo, L7, DRT)
```

## Tips

- Amplification attacks require IP spoofing — BCP38/84 at ISP level prevents them
- Slowloris is effective against Apache (thread-per-connection) but not nginx (event-driven)
- SYN cookies trade some TCP features (window scaling, SACK) for stateless defense
- Volumetric attacks are measured in Gbps; protocol in pps; application in rps
- Multi-vector attacks combine all three categories simultaneously
- LOIC traffic is trivially detectable and traceable — no spoofing capability
- The largest recorded DDoS attacks exceeded 3 Tbps (memcached amplification)
- Know the difference between reflection (using third-party) and amplification (response > request)

## See Also

- `sheets/offensive/botnets.md`
- `sheets/offensive/network-attacks.md`
- `sheets/defensive/ids-ips.md`
- `sheets/defensive/firewall-config.md`
- `detail/offensive/dos-ddos-attacks.md`

## References

- CEH v13 Module 10: Denial-of-Service
- NIST SP 800-61 Rev 2: Computer Security Incident Handling Guide
- US-CERT: Understanding Denial-of-Service Attacks
- RFC 4987: TCP SYN Flooding Attacks and Common Mitigations
- RFC 2827 (BCP38): Network Ingress Filtering
- RFC 3704 (BCP84): Ingress Filtering for Multihomed Networks
- Cloudflare Learning Center: DDoS Attack Types
- OWASP: Denial of Service Cheat Sheet
