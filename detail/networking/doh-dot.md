# Encrypted DNS Deep Dive -- DoT Handshakes, DoH Wire Format, and Privacy Analysis

> *Encrypted DNS transports protect the last major plaintext metadata leak in the HTTPS ecosystem. Understanding their wire formats, handshake costs, padding strategies, and privacy boundaries is essential for both deployment and circumvention analysis.*

---

## 1. DoT TLS Handshake Flow

### Connection Establishment

A DoT connection requires a TCP handshake followed by a TLS handshake before any DNS query can be sent:

```
Client                         Server (port 853)
  |                                |
  |  ---- SYN ------------------> |   TCP handshake
  |  <--- SYN-ACK --------------- |   (1 RTT)
  |  ---- ACK ------------------> |
  |                                |
  |  ---- ClientHello ----------> |   TLS 1.3 handshake
  |       (SNI: dns.resolver)     |   (1 RTT for first connection)
  |  <--- ServerHello ----------- |
  |       EncryptedExtensions     |
  |       Certificate             |
  |       CertificateVerify       |
  |       Finished                |
  |  ---- Finished -------------> |
  |                                |
  |  ---- DNS Query ------------> |   Encrypted DNS wire format
  |  <--- DNS Response ---------- |   (2-byte length prefix + message)
  |                                |
```

### TLS 1.3 vs TLS 1.2 Handshake Cost

With TLS 1.3 (required by most modern DoT resolvers):

$$T_{first} = T_{TCP} + T_{TLS} + T_{query} = 1\text{RTT} + 1\text{RTT} + 1\text{RTT} = 3\text{RTT}$$

With TLS 1.2:

$$T_{first} = T_{TCP} + T_{TLS} + T_{query} = 1\text{RTT} + 2\text{RTT} + 1\text{RTT} = 4\text{RTT}$$

### TLS Session Resumption

Subsequent connections benefit from TLS 1.3 0-RTT resumption (using pre-shared keys):

$$T_{resumed} = T_{TCP} + T_{0RTT} = 1\text{RTT} + 0\text{RTT} = 1\text{RTT}$$

The DNS query can be sent alongside the TLS ClientHello with early data (0-RTT), but this is vulnerable to replay attacks. Since DNS queries are inherently idempotent, replay risk is generally acceptable for DoT.

### Connection Reuse

RFC 7858 Section 3.4 recommends reusing TCP connections for multiple queries:

$$T_{nth} = T_{query} = 1\text{RTT} \quad \text{(for } n > 1 \text{ on same connection)}$$

Pipelining multiple queries on a single connection amortizes the handshake cost:

$$T_{avg} = \frac{T_{handshake} + n \cdot T_{query}}{n} = \frac{2\text{RTT} + n \cdot 1\text{RTT}}{n}$$

As $n \to \infty$, $T_{avg} \to 1\text{RTT}$, converging to plaintext DNS latency.

### Authentication Modes

| Mode | Certificate Validation | Protection Level |
|:---|:---|:---|
| Opportunistic | None (accept any cert) | Passive eavesdropping only |
| Strict (PKIX) | CA chain validation + hostname check | Passive + active attacks |
| Strict (SPKI pin) | Public key pin match | Passive + active + CA compromise |

In strict mode, the client verifies the TLS certificate against the authentication domain name (ADN). For example, connecting to `1.1.1.1:853` with ADN `cloudflare-dns.com` means the certificate must be valid for `cloudflare-dns.com` and chain to a trusted CA.

---

## 2. DoH Wire Format

### HTTP Message Framing

DoH wraps DNS messages in HTTP/2 (or HTTP/3) frames. Two methods are defined:

**POST method:**

```
POST /dns-query HTTP/2
Host: cloudflare-dns.com
Content-Type: application/dns-message
Content-Length: 33
Accept: application/dns-message

[33 bytes: raw DNS wire format query]
```

**GET method:**

```
GET /dns-query?dns=AAABAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE HTTP/2
Host: cloudflare-dns.com
Accept: application/dns-message
```

The `dns` parameter contains the DNS wire format query encoded in base64url (RFC 4648, Section 5) without padding characters.

### Content Type: application/dns-message

The `application/dns-message` media type (registered by RFC 8484) contains the exact same binary format as a standard DNS UDP message:

```
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                      ID                       |   2 bytes (usually 0 for DoH)
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|QR|   Opcode  |AA|TC|RD|RA| Z|AD|CD|   RCODE   |   2 bytes flags
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    QDCOUNT                     |   2 bytes
|                    ANCOUNT                     |   2 bytes
|                    NSCOUNT                     |   2 bytes
|                    ARCOUNT                     |   2 bytes
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                   Question Section             |   variable
|                   Answer Section               |   variable
|                   Authority Section            |   variable
|                   Additional Section           |   variable
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

Key differences from traditional DNS:

- The DNS message ID field SHOULD be set to 0 (the HTTP request/response pairing provides message correlation, making the ID redundant).
- The TC (truncation) bit is not meaningful since HTTP handles arbitrary message sizes.
- HTTP/2 multiplexing replaces DNS pipelining.

### HTTP/2 Framing

Each DoH query/response is carried in an HTTP/2 stream:

```
HTTP/2 Frame: HEADERS (stream 1)
  :method = POST
  :path = /dns-query
  content-type = application/dns-message

HTTP/2 Frame: DATA (stream 1)
  [DNS wire format bytes]

HTTP/2 Frame: HEADERS (stream 1, response)
  :status = 200
  content-type = application/dns-message
  cache-control = max-age=300   (mirrors DNS TTL)

HTTP/2 Frame: DATA (stream 1, response)
  [DNS wire format response bytes]
```

The HTTP `cache-control: max-age` header SHOULD match the minimum TTL from the DNS response, enabling HTTP caches to cache DNS responses correctly.

### JSON Wire Format (Non-Standard)

Several resolvers offer a JSON API alongside the standard binary format:

```json
{
  "Status": 0,
  "TC": false,
  "RD": true,
  "RA": true,
  "AD": true,
  "CD": false,
  "Question": [
    { "name": "example.com", "type": 1 }
  ],
  "Answer": [
    { "name": "example.com", "type": 1, "TTL": 300,
      "data": "93.184.216.34" }
  ]
}
```

This uses `application/dns-json` (Cloudflare) or `application/x-javascript` (Google, legacy). The JSON format is easier to parse in web applications but is not part of RFC 8484.

---

## 3. DNS Padding (RFC 8467)

### The Problem: Query Size Analysis

Even with encryption, DNS message sizes vary by domain name length and record type. An observer can correlate encrypted message sizes with known query/response patterns to infer which domains are being resolved.

### Padding Strategy

RFC 8467 defines an EDNS(0) padding option (option code 12, from RFC 7830) and recommends block-based padding:

For queries (stub to recursive resolver):

$$L_{padded} = \lceil L_{original} / B \rceil \times B$$

where $B$ is the block size (recommended: 128 bytes).

For responses (recursive resolver to stub):

$$L_{padded} = \lceil L_{original} / B \rceil \times B$$

with recommended block size $B = 468$ bytes.

### Worked Example

A query for `mail.example.com A` has an original size of 45 bytes. With 128-byte block padding:

$$L_{padded} = \lceil 45 / 128 \rceil \times 128 = 1 \times 128 = 128 \text{ bytes}$$

A query for `very-long-subdomain.corporate.example.co.uk AAAA` at 72 bytes:

$$L_{padded} = \lceil 72 / 128 \rceil \times 128 = 1 \times 128 = 128 \text{ bytes}$$

Both queries pad to the same size, preventing size-based differentiation.

### Padding in DoH

DoH has two padding layers:

1. **DNS-level:** EDNS(0) padding option in the DNS message body.
2. **HTTP/2-level:** HTTP/2 PADDING frames or DATA frame padding can further obscure the DNS message size.

The combination provides stronger protection than either layer alone, at the cost of increased bandwidth usage.

---

## 4. EDNS Client Subnet (ECS) Privacy

### What ECS Does

EDNS Client Subnet (RFC 7871) allows a recursive resolver to forward a prefix of the client's IP address to the authoritative server. This enables CDN-optimized responses based on client location.

### The Privacy Conflict

ECS directly conflicts with the privacy goals of encrypted DNS:

| Without ECS | With ECS |
|:---|:---|
| Authoritative server sees resolver IP only | Authoritative server sees partial client IP |
| CDN routing based on resolver location | CDN routing based on client location |
| Better privacy | Better performance for geo-distributed content |

### ECS Scope Prefix Length

The source prefix length determines how much of the client address is revealed:

| Prefix Length (IPv4) | Addresses Covered | Granularity |
|:---:|:---:|:---|
| /32 | 1 | Exact address (worst privacy) |
| /24 | 256 | Single subnet |
| /20 | 4,096 | Small ISP block |
| /16 | 65,536 | Large ISP block |
| /0 | All | No location info (best privacy) |

### Resolver Policies

- **Cloudflare 1.1.1.1:** Does not send ECS to authoritative servers. Uses Anycast for geographic optimization instead.
- **Google 8.8.8.8:** Sends ECS with a /24 prefix by default for most queries. Can be disabled per-query with the `edns_client_subnet=0.0.0.0/0` parameter.
- **Quad9 9.9.9.9:** Does not send ECS.

For maximum privacy, choose a resolver that does not forward ECS data, or set `edns_client_subnet=0.0.0.0/0` in your resolver configuration.

---

## 5. Comparison Table: Do53 vs DoT vs DoH vs DoQ

| Property | Do53 | DoT | DoH | DoQ |
|:---|:---|:---|:---|:---|
| **RFC** | 1035 | 7858 | 8484 | 9250 |
| **Transport** | UDP/TCP | TCP + TLS | TCP + TLS (HTTP/2) | QUIC (UDP) |
| **Default Port** | 53 | 853 | 443 | 853 |
| **Encryption** | None | TLS 1.2/1.3 | TLS 1.2/1.3 | TLS 1.3 |
| **First Query Latency** | 0-1 RTT | 2-3 RTT | 2-3 RTT | 1 RTT |
| **Resumed Connection** | N/A | 1 RTT (0-RTT possible) | 1 RTT (0-RTT possible) | 0 RTT |
| **Head-of-Line Blocking** | None (UDP) / Yes (TCP) | Yes (TCP) | Yes (TCP, mitigated by HTTP/2 streams) | None (per-stream) |
| **Multiplexing** | No (UDP) | Pipelining | HTTP/2 streams | QUIC streams |
| **Blockability** | Easy (port 53) | Easy (port 853) | Hard (port 443, shared with HTTPS) | Moderate (port 853, but UDP) |
| **DNS Padding** | N/A | EDNS(0) option | EDNS(0) + HTTP/2 padding | QUIC padding |
| **Message Overhead** | 0 bytes | 2-byte length prefix | HTTP/2 framing (~30-60 bytes) | QUIC framing (~20-40 bytes) |
| **Server Auth** | None | TLS certificate | TLS certificate | TLS certificate |
| **Cacheability** | DNS caches | DNS caches | DNS + HTTP caches | DNS caches |
| **Browser Support** | Via system resolver | No (system-level only) | Yes (native) | Experimental |
| **Firewall Visibility** | Full query inspection | Encrypted, identifiable port | Encrypted, blends with HTTPS | Encrypted, identifiable port |

---

## 6. Latency Analysis

### Cold Start Comparison

Assuming 20 ms RTT to the resolver:

| Protocol | Handshake RTTs | Query RTT | Total | Time (at 20 ms RTT) |
|:---|:---:|:---:|:---:|:---:|
| Do53 (UDP) | 0 | 1 | 1 RTT | 20 ms |
| Do53 (TCP) | 1 | 1 | 2 RTT | 40 ms |
| DoT (TLS 1.3) | 2 | 1 | 3 RTT | 60 ms |
| DoT (TLS 1.2) | 3 | 1 | 4 RTT | 80 ms |
| DoH (TLS 1.3) | 2 | 1 | 3 RTT | 60 ms |
| DoQ (QUIC) | 0* | 1 | 1 RTT | 20 ms |

*QUIC combines cryptographic and transport handshakes in a single RTT, and with 0-RTT resumption the handshake cost drops to zero.

### Amortized Latency with Connection Reuse

For $n$ queries on a persistent connection:

$$T_{DoT} = \frac{2\text{RTT} + n \cdot 1\text{RTT}}{n} = 1\text{RTT} + \frac{2\text{RTT}}{n}$$

$$T_{DoH} = \frac{2\text{RTT} + n \cdot 1\text{RTT}}{n} = 1\text{RTT} + \frac{2\text{RTT}}{n}$$

$$T_{DoQ} = \frac{1\text{RTT} + n \cdot 1\text{RTT}}{n} = 1\text{RTT} + \frac{1\text{RTT}}{n}$$

After 10 queries: DoT/DoH average 1.2 RTT, DoQ averages 1.1 RTT. Both converge to 1 RTT, matching plaintext DNS throughput.

### HTTP/2 Multiplexing Advantage

DoH over HTTP/2 can send multiple queries concurrently without waiting for responses:

$$T_{batch}^{DoH} = T_{handshake} + 1\text{RTT} \quad \text{(for } k \text{ parallel queries)}$$

Compared to sequential DoT queries:

$$T_{batch}^{DoT} = T_{handshake} + k \cdot 1\text{RTT}$$

For a page load triggering 10 DNS lookups simultaneously, DoH with multiplexing completes all lookups in one RTT after the handshake, while sequential DoT requires 10 RTTs.

In practice, DoT also supports pipelining (sending queries without waiting for responses), but HTTP/2 stream multiplexing provides better flow control and prioritization.

---

## 7. DNSSEC Interaction

### DNSSEC and Encrypted DNS Are Complementary

DNSSEC and encrypted DNS solve different problems:

| Concern | DNSSEC | Encrypted DNS (DoT/DoH/DoQ) |
|:---|:---|:---|
| Response authenticity | Yes (cryptographic signatures) | No (trusts the resolver) |
| Response integrity | Yes (detects tampering) | Yes (TLS integrity) |
| Query confidentiality | No | Yes (TLS encryption) |
| Protection scope | End-to-end (authoritative to client) | Stub-to-resolver hop only |

### The Trust Gap

Encrypted DNS protects the link between the stub resolver and the recursive resolver. DNSSEC protects the link between the recursive resolver and the authoritative server (and transitively, the client). Neither alone provides complete security:

```
Client <--[DoT/DoH]--> Recursive Resolver <--[plaintext]--> Authoritative Server
         encrypted                          potentially interceptable
                                            (DNSSEC validates this hop)
```

The recursive resolver is a trust bottleneck in both cases. With DNSSEC validation enabled at the stub resolver, the client can verify that responses were not modified by the recursive resolver itself.

### AD Flag and DO Flag

When using encrypted DNS with DNSSEC:

- Set the DO (DNSSEC OK) flag in queries to request DNSSEC records.
- Check the AD (Authenticated Data) flag in responses to confirm the resolver validated the DNSSEC chain.
- For full security, perform local DNSSEC validation rather than trusting the resolver's AD flag.

---

## 8. Censorship Circumvention Implications

### DNS-Based Censorship Mechanisms

Many censorship regimes rely on DNS manipulation:

1. **DNS injection:** Inject forged responses on the network path before the legitimate response arrives.
2. **DNS blocking:** ISP resolvers return NXDOMAIN or redirect to a block page.
3. **Transparent DNS proxy:** Intercept port 53 traffic and redirect it to a filtering resolver, regardless of the configured DNS server.

### How Encrypted DNS Defeats Each Mechanism

| Mechanism | Do53 | DoT | DoH |
|:---|:---|:---|:---|
| DNS injection | Vulnerable | Protected (TLS integrity) | Protected (TLS integrity) |
| ISP resolver blocking | Bypass by changing resolver | Bypass + encrypted | Bypass + encrypted + hidden |
| Transparent DNS proxy | Completely vulnerable | Fails (wrong TLS cert) | Hidden in HTTPS traffic |
| Port-based blocking | Easy (block port 53) | Easy (block port 853) | Hard (would block all HTTPS) |
| DPI-based blocking | Trivial (plaintext) | Possible (port 853 heuristic) | Difficult (standard HTTPS) |

### DoH as a Censorship Circumvention Tool

DoH is particularly resistant to blocking because:

1. It shares port 443 with all HTTPS traffic. Blocking port 443 would break the internet.
2. Resolvers can be hosted on CDN IP addresses shared with millions of other websites (domain fronting).
3. The HTTPS traffic pattern is indistinguishable from normal web browsing.
4. ECH (Encrypted Client Hello) hides the resolver's hostname in the TLS handshake.

### Counter-Censorship Limitations

Encrypted DNS is not a complete censorship circumvention solution:

- The IP addresses of well-known DoH resolvers (1.1.1.1, 8.8.8.8) can be blocked.
- After DNS resolution, the connection to the destination IP can still be blocked.
- SNI-based blocking (inspecting the TLS ClientHello) can block connections even if DNS resolution succeeds. ECH mitigates this but requires server-side support.
- Certificate transparency logs can reveal which domains have certificates, enabling IP-based blocking.

---

## 9. Deployment Considerations

### Resolver Selection Criteria

When choosing an encrypted DNS resolver, evaluate:

| Criterion | Why It Matters |
|:---|:---|
| Logging policy | Determines how much metadata the resolver retains |
| ECS forwarding | Whether client subnet info is sent to authoritative servers |
| DNSSEC validation | Whether the resolver validates DNSSEC signatures |
| Anycast coverage | Affects latency from your geographic location |
| Filtering | Whether the resolver blocks malware/phishing domains |
| Protocol support | DoT, DoH, DoQ availability |
| Audit frequency | Whether the resolver's privacy claims are independently verified |

### Bootstrap Problem

To connect to `https://cloudflare-dns.com/dns-query`, the client must first resolve `cloudflare-dns.com` -- which requires DNS. Solutions:

1. **Hardcode resolver IPs:** The client knows `1.1.1.1` and connects directly. The TLS certificate is validated against the hostname `cloudflare-dns.com`.
2. **System DNS bootstrap:** Use the system's plaintext DNS resolver for the initial resolution of the DoH hostname only. All subsequent queries use DoH.
3. **DDR (RFC 9462):** Query `_dns.resolver.arpa` via the network-provided resolver to discover its encrypted counterpart.

### Performance Optimization

To minimize the latency penalty of encrypted DNS:

- **Persistent connections:** Keep DoT/DoH connections open to amortize handshake cost.
- **Connection pooling:** Maintain connections to multiple resolvers for failover.
- **0-RTT resumption:** Enable TLS session tickets for faster reconnection.
- **Prefetching:** Resolve anticipated domains before they are needed (browser speculation).
- **Local caching:** Run a local caching resolver (Unbound, systemd-resolved) to avoid redundant encrypted queries.
- **DoQ adoption:** Where supported, prefer DoQ for the lowest cold-start latency.
