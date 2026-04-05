# The Engineering of Web Security Proxies — Architecture, Interception, and Content Inspection

> *A web security proxy sits at the boundary between users and the internet, inspecting every HTTP/HTTPS transaction for threats, enforcing acceptable use, and providing visibility into application usage. The proxy's power comes from its position: it sees everything the user sees, and can block what should not pass.*

---

## 1. Proxy Architecture — Explicit vs Transparent

### Explicit Forward Proxy

In explicit proxy mode, the client is configured to send all web requests to the proxy. The client knows it is being proxied.

```
Client                    Proxy                    Web Server
  |                        |                           |
  |--- CONNECT host:443 -->|                           |
  |<-- 200 Connection -----|                           |
  |    Established         |                           |
  |=== TLS to proxy =======|                           |  (if decrypting)
  |    or                  |=== TLS to server =========|  (if pass-through)
  |=== TLS to server ======|===========================|  (tunnel mode)
  |                        |                           |
```

For HTTP (non-TLS), the client sends the full URL in the request line:

```
GET http://www.example.com/page.html HTTP/1.1
Host: www.example.com
```

For HTTPS, the client first uses the CONNECT method to establish a tunnel:

```
CONNECT www.example.com:443 HTTP/1.1
Host: www.example.com:443
```

The proxy then either:
1. **Tunnels** the TLS connection (pass-through, no inspection)
2. **Terminates** the TLS connection (decryption, full inspection)

### Transparent Proxy

In transparent mode, the client is unaware of the proxy. Traffic is redirected at the network layer:

```
Client                 Router/Switch             Proxy              Web Server
  |                        |                      |                     |
  |--- TCP SYN (port 80) ->|                      |                     |
  |                        |--- WCCP/PBR redirect->|                    |
  |                        |                      |                     |
  |                        |                      |--- TCP SYN -------->|
  |                        |                      |<-- TCP SYN-ACK -----|
  |                        |                      |                     |
  |<-- TCP SYN-ACK --------|<---------------------|                     |
  |--- HTTP GET /page ---->|--- redirect -------->|                     |
  |                        |                      |--- HTTP GET ------->|
  |                        |                      |<-- Response --------|
  |<-- Response -----------|<---------------------|                     |
```

### Architectural Comparison

| Aspect | Explicit Proxy | Transparent Proxy |
|--------|---------------|-------------------|
| Client awareness | Client configured to use proxy | Client unaware |
| HTTPS handling | CONNECT method (client knows) | TCP intercept (network redirect) |
| Authentication | HTTP 407 Proxy-Authentication | HTTP 302 redirect to captive portal |
| Proxy failure mode | Client gets connection error | Traffic either passes or drops |
| Configuration management | PAC file, GPO, MDM | Network infrastructure (WCCP, PBR) |
| Non-HTTP traffic | Client can use CONNECT for any port | Only redirected ports inspected |
| Application compatibility | Better (client expects proxy) | Can break some applications |
| Bypass difficulty | Easy (change proxy settings) | Harder (requires network bypass) |

### Proxy Decision: When to Use Which

$$Suitability_{explicit} = W_{mgmt} \times M_{managed\_devices} + W_{auth} \times A_{SSO} + W_{compat} \times C_{apps}$$

$$Suitability_{transparent} = W_{byod} \times B_{unmanaged} + W_{stealth} \times S_{no\_config} + W_{coverage} \times C_{all\_traffic}$$

Most enterprise deployments use both: explicit proxy for managed endpoints (via PAC file) and transparent proxy (via WCCP) for unmanaged/guest devices.

---

## 2. WCCP Protocol Operations

### WCCP v2 Protocol (RFC 3040 / Cisco Proprietary Extensions)

WCCP enables routers to redirect web traffic to proxy appliances with health monitoring, load balancing, and failover.

```
Router                              WSA (Service Group Member)
  |                                       |
  |<-- WCCP "Here I Am" (UDP 2048) ------|  (WSA announces itself)
  |--- WCCP "I See You" (UDP 2048) ----->|  (Router acknowledges)
  |                                       |
  |  [Periodic keepalives every 10s]      |
  |                                       |
  |--- Redirect traffic (GRE/L2) ------->|  (HTTP/HTTPS intercepted)
  |<-- Return traffic (GRE/L2) ----------|  (Inspected response)
  |                                       |
  |  [If keepalive missed 3x → remove]   |
```

### WCCP Redirect Methods

| Method | Mechanism | Requirements | Performance |
|--------|-----------|-------------|-------------|
| GRE (Generic Routing Encapsulation) | Encapsulate in GRE tunnel (IP proto 47) | Router + WSA on any L3 segment | Good (adds 24-byte header) |
| L2 (Layer 2) | MAC address rewrite | Router + WSA on same VLAN | Better (no encapsulation overhead) |

GRE redirect:

$$Overhead_{GRE} = 24 \text{ bytes per packet (GRE header)}$$

$$Throughput_{effective} = \frac{MTU - Overhead_{GRE}}{MTU} \times Throughput_{link}$$

For a 1500-byte MTU:
$$Throughput_{effective} = \frac{1476}{1500} \times 1\text{Gbps} = 984\text{Mbps}$$

This 1.6% overhead is negligible, but fragmentation issues can arise if the original packet is already at MTU. Some deployments reduce the interface MTU to 1476 to avoid fragmentation.

### WCCP Load Balancing

WCCP distributes traffic across multiple WSA appliances using either hash-based or mask-based assignment:

**Hash-based assignment:**

$$WSA_{target} = Hash(SrcIP, DstIP, SrcPort, DstPort) \mod N_{WSA}$$

The hash space is divided into 256 buckets, distributed across WSA appliances:

$$Buckets_{per\_WSA} = \frac{256}{N_{WSA}}$$

**Mask-based assignment (preferred):**

$$WSA_{target} = (Field \mathbin{\&} Mask) \gg Shift \mod N_{WSA}$$

Mask-based assignment is more flexible: the administrator defines which bits of the source/destination IP and port to use for assignment. This allows for better distribution when source IPs are clustered.

### WCCP Failover

When a WSA fails (misses 3 consecutive keepalives, ~30 seconds):

1. Router removes failed WSA from the service group
2. Router redistributes hash/mask buckets to surviving WSAs
3. Traffic flows to surviving WSAs without interruption
4. When failed WSA recovers, it re-announces and buckets are rebalanced

$$Failover_{time} = 3 \times T_{keepalive} = 3 \times 10s = 30s$$

During failover, traffic can optionally bypass the proxy entirely (open mode) or be dropped (closed mode):

| Mode | Behavior During Failover | Risk |
|------|------------------------|------|
| Open | Traffic bypasses proxy, goes direct | Uninspected traffic reaches internet |
| Closed | Traffic is dropped | Users lose internet access |

Most security-conscious deployments use closed mode with redundant WSAs to avoid uninspected traffic.

---

## 3. SSL/TLS Interception Trust Model

### MitM Architecture

HTTPS decryption requires the proxy to act as a man-in-the-middle:

```
Client              WSA (MitM)              Web Server
  |                    |                        |
  |--- ClientHello --->|                        |
  |                    |--- ClientHello -------->|
  |                    |<-- ServerHello + cert --|
  |                    |                        |
  |                    | [Validate server cert]  |
  |                    | [Generate new cert      |
  |                    |  for server domain,     |
  |                    |  signed by WSA CA]      |
  |                    |                        |
  |<-- ServerHello ----|                        |
  |    + WSA-signed    |                        |
  |    certificate     |                        |
  |                    |                        |
  |=== TLS session 1 ==|=== TLS session 2 ======|
  |  (WSA CA trust)    |  (Server CA trust)     |
  |                    |                        |
  | [Client encrypts   | [WSA decrypts,         |
  |  with WSA cert]    |  inspects, re-encrypts |
  |                    |  with server cert]      |
```

### Trust Chain Requirements

For decryption to work without browser warnings:

$$Trust_{client} = CA_{WSA} \in TrustStore_{client}$$

The WSA's root CA certificate must be in every client's trusted certificate store. Deployment methods:

| Method | Scope | Automation |
|--------|-------|-----------|
| GPO (Group Policy) | Windows domain-joined | Automatic |
| MDM (Intune, JAMF) | Managed mobile devices | Automatic |
| SCEP/NDES | Certificate auto-enrollment | Automatic |
| Manual install | Individual devices | Per-user action |
| Browser-specific | Firefox (uses own store) | Separate deployment |

### Security Implications of TLS Interception

TLS interception introduces significant security considerations:

1. **Reduced cipher security:** The proxy negotiates two separate TLS sessions; the client-to-proxy session may use a weaker cipher than the proxy-to-server session
2. **Certificate validation gaps:** The proxy must correctly validate upstream certificates; bugs in proxy validation can make clients vulnerable to real MitM attacks
3. **Key storage:** The proxy's CA private key, if compromised, allows impersonation of any website for all clients that trust the CA
4. **Compliance:** Some regulations (GDPR, HIPAA) restrict interception of certain categories of traffic
5. **Certificate pinning breakage:** Applications with HPKP or built-in pins will reject the proxy-signed certificate

$$Risk_{MitM\_proxy} = P_{CA\_key\_compromise} \times Impact_{all\_clients} + P_{validation\_bug} \times Impact_{MitM\_bypass}$$

### Certificate Pinning and Exemptions

Applications that use certificate pinning (HPKP, built-in pins, or certificate transparency requirements) will fail through a decrypting proxy:

```
Application with pinning:
  Expected pin: sha256/abc123... (original server cert)
  Received pin: sha256/xyz789... (WSA-generated cert)
  Result: Connection refused (pin mismatch)

Solution: Add domain to decryption bypass/pass-through list
```

Common applications requiring pass-through:
- Mobile banking apps (all major banks)
- OS update services (Windows Update, Apple Software Update)
- Cisco WebEx, Microsoft Teams (certificate pinned)
- Docker Hub, npm registry (CI/CD pipelines)
- Apple services (iCloud, App Store)

---

## 4. URL Categorization Techniques

### Categorization Methods

| Method | How | Latency | Accuracy |
|--------|-----|---------|----------|
| Local database | Pre-loaded URL/domain database on appliance | <1ms | High (for known URLs) |
| Cloud lookup | Real-time query to Talos cloud | 5-50ms | Very high (current) |
| Dynamic categorization | On-the-fly page content analysis | 50-200ms | Medium (new/unknown URLs) |
| Custom categories | Admin-defined URL lists | <1ms | Perfect (admin-controlled) |

### Categorization Pipeline

```
URL: https://www.example.com/new-page.html
        |
        v
1. Check local cache (recently categorized URLs)
   → Hit? Return cached category. Miss? Continue.
        |
        v
2. Check local database (pre-loaded from Talos)
   → Match on domain (example.com)? Return category. No match? Continue.
        |
        v
3. Cloud lookup (real-time Talos query)
   → Send URL/domain to cloud. Response: {category, reputation, confidence}
   → Match? Return category. No match? Continue.
        |
        v
4. Dynamic categorization (analyze page content)
   → Fetch page, analyze text/images/links
   → Classify based on content features
   → Return best-guess category (lower confidence)
        |
        v
5. Uncategorized (no category determined)
   → Apply "Uncategorized" policy (typically: allow + full scan)
```

### Category Database Scale

$$Categories \approx 80\text{-}90 \text{ URL categories}$$
$$URLs_{database} \approx 10\text{+} \text{ billion URLs categorized}$$
$$Updates_{frequency} \approx 3\text{-}5 \text{ minute intervals from Talos}$$

### Web Reputation Scoring

Web reputation combines multiple signals into a single score:

$$WRS(URL) = \sum_{i=1}^{N} W_i \times S_i$$

Signals include:

| Signal | Weight | Description |
|--------|--------|-------------|
| Domain age | Medium | Newer domains more suspicious |
| Registration info | Low | WHOIS privacy, free registrar |
| Hosting location | Low | Hosting in high-risk countries |
| Historical behavior | High | Past malware, phishing association |
| Network owner | Medium | Hosting provider reputation |
| Traffic patterns | Medium | Sudden spikes, DGA-like patterns |
| Content analysis | High | Exploit kit markers, phishing indicators |
| Third-party blocklists | High | Presence on Spamhaus, PhishTank, etc. |

---

## 5. AVC Deep Packet Inspection

### Application Identification Layers

```
Layer 1: Port/Protocol (least specific)
  → Port 80 = HTTP, Port 443 = HTTPS
  → Identifies protocol, not application

Layer 2: Host/SNI (moderate specificity)
  → Host: www.youtube.com
  → Identifies site, not specific activity

Layer 3: URL/URI Pattern (high specificity)
  → GET /api/v1/upload → identifies upload action
  → Distinguishes browsing from uploading

Layer 4: Payload Analysis (highest specificity)
  → Content-Type, POST body analysis
  → Identifies micro-applications within sites
```

### AVC Classification Process

$$App_{identified} = f(Port, SNI, Host, URI, Headers, Payload, Certificate)$$

For encrypted traffic without decryption, AVC relies on:

1. **TLS SNI (Server Name Indication):** Cleartext hostname in ClientHello
2. **Certificate CN/SAN:** Server certificate common name and subject alternative names
3. **JA3/JA3S fingerprinting:** TLS handshake parameter hashing for client/server identification
4. **Connection metadata:** Packet sizes, timing, connection patterns

$$Accuracy_{encrypted} < Accuracy_{decrypted}$$

Without decryption, AVC can identify the application (YouTube, Dropbox) but not the specific action (upload vs download, video vs search).

### Bandwidth Control

AVC can enforce bandwidth policies:

$$BW_{user,app} = \min(BW_{policy}, BW_{available})$$

Bandwidth control methods:

| Method | Granularity | Fairness |
|--------|------------|----------|
| Per-user throttle | Limit each user to X Mbps for an app | Fair per-user |
| Per-app global throttle | Limit total bandwidth for an app | Fair across users |
| Per-app per-user throttle | Limit each user's bandwidth per app | Most granular |

$$BW_{total\_app} = \sum_{u=1}^{U} BW_{u,app} \leq BW_{policy\_global}$$

---

## 6. SOCKS5 Protocol

### SOCKS5 Handshake (RFC 1928)

```
Client                    SOCKS Proxy                  Destination
  |                           |                            |
  |--- Version/Auth methods ->|                            |
  |    (05, 01, 00)           |  (v5, 1 method, no auth)  |
  |<-- Selected method -------|                            |
  |    (05, 00)               |  (v5, no auth selected)   |
  |                           |                            |
  |--- Connect request ------>|                            |
  |    (05, 01, 00, 03,       |  (v5, connect, reserved,  |
  |     len, hostname, port)  |   domain, host, port)     |
  |                           |--- TCP connect ----------->|
  |                           |<-- TCP established --------|
  |<-- Connect reply ---------|                            |
  |    (05, 00, ...)          |  (v5, success)            |
  |                           |                            |
  |=== Data ==================|=== Data ===================|
```

### SOCKS5 vs HTTP Proxy

| Feature | SOCKS5 | HTTP Proxy |
|---------|--------|-----------|
| Protocol support | Any TCP (and UDP) | HTTP/HTTPS only |
| Application awareness | None (raw socket relay) | Full HTTP inspection |
| Authentication | Username/password, GSS-API | HTTP Basic, NTLM, Kerberos, SAML |
| DNS resolution | Client or proxy (configurable) | Proxy resolves |
| Content inspection | Requires additional DPI | Native HTTP parsing |
| Use case | Non-HTTP applications, SSH, custom protocols | Web browsing, REST APIs |

### SOCKS5 Security Considerations

SOCKS5 proxies bypass HTTP-layer inspection unless the proxy also performs DPI on the tunneled content:

$$Visibility_{SOCKS} = \begin{cases} Full & \text{if DPI enabled on SOCKS traffic} \\ Metadata\_only & \text{if connection logging only} \\ None & \text{if simple relay} \end{cases}$$

SOCKS tunnels can be abused for proxy evasion: users can tunnel arbitrary traffic through SOCKS to bypass HTTP content filtering. Restrict SOCKS access to authorized applications and users only.

---

## 7. Proxy Scaling

### Capacity Planning

$$Users_{per\_WSA} = \frac{Throughput_{WSA}}{BW_{avg\_per\_user}}$$

For a WSA with 1 Gbps throughput and 2 Mbps average per active user:
$$Users_{per\_WSA} = \frac{1000}{2} = 500 \text{ concurrent active users}$$

However, throughput varies significantly with inspection depth:

| Inspection Level | Throughput (S690) | Throughput (S690 w/ HTTPS decrypt) |
|-----------------|-------------------|-----------------------------------|
| URL filtering only | 2+ Gbps | N/A |
| URL + anti-malware | 1.5 Gbps | 800 Mbps |
| URL + AV + AMP | 1.2 Gbps | 600 Mbps |
| Full (URL + AV + AMP + AVC + DLP) | 800 Mbps | 400 Mbps |

HTTPS decryption roughly halves throughput due to double TLS processing:

$$Throughput_{decrypt} \approx \frac{Throughput_{no\_decrypt}}{2}$$

### High Availability Architectures

```
Architecture 1: Active-Active with Load Balancer
Users → Load Balancer → WSA1
                      → WSA2
                      → WSA3
  - Even distribution, full redundancy
  - LB health checks proxy port (3128)

Architecture 2: Active-Active with WCCP
Users → Router (WCCP) → WSA1 (hash buckets 0-127)
                       → WSA2 (hash buckets 128-255)
  - WCCP handles load balancing and failover
  - No external load balancer needed

Architecture 3: Active-Standby with DNS
Users → PAC file → PROXY wsa1:3128; PROXY wsa2:3128
  - Client-side failover (PAC file fallback)
  - No true load balancing (first proxy preferred)
```

### Connection Pooling and Caching

The proxy maintains connection pools to frequently-accessed servers:

$$Connections_{pool} = \{(Server_1, N_1), (Server_2, N_2), \ldots\}$$

Benefits:
- Eliminates TCP handshake latency for subsequent requests ($\approx 1\text{ RTT saved}$)
- Eliminates TLS handshake latency ($\approx 1\text{-}2\text{ RTTs saved}$)
- Reduces server load (fewer connections)

$$Latency_{saved} = N_{requests} \times (T_{TCP\_handshake} + T_{TLS\_handshake})$$

Caching stores responses locally for future requests:

$$Cache_{hit\_ratio} = \frac{Requests_{served\_from\_cache}}{Requests_{total}}$$

Modern HTTPS-heavy traffic has low cache hit ratios (most responses are not cacheable), but caching still helps for software updates, static assets, and CRL/OCSP responses.

---

## 8. Cloud vs On-Prem Proxy

### Comparison Matrix

| Factor | On-Premises (WSA) | Cloud (Umbrella SIG, Zscaler) |
|--------|-------------------|-------------------------------|
| Deployment | Physical/virtual appliance in DC | Cloud service, no hardware |
| Latency (internal users) | Low (local appliance) | Higher (traffic to cloud PoP) |
| Latency (remote users) | High (VPN required) | Low (nearest cloud PoP) |
| Maintenance | Admin manages updates, patches | Vendor manages infrastructure |
| Scaling | Buy more appliances | Auto-scaling (vendor) |
| HTTPS inspection | Full control over CA | Vendor's CA or customer CA |
| Customization | Highly customizable | Limited to vendor's feature set |
| Cost model | CapEx (hardware) + OpEx (license) | OpEx only (subscription) |
| Authentication | AD/NTLM/Kerberos native | Requires agent or SAML federation |
| Data residency | Data stays on-prem | Data processed in vendor cloud |

### Hybrid Architecture (Recommended)

$$Coverage_{hybrid} = Coverage_{on\_prem} \cup Coverage_{cloud}$$

```
On-network users      → On-prem WSA (WCCP/PAC)    → Internet
  (office, campus)       Full inspection, AD auth

Remote users          → Cloud proxy (Umbrella SIG)  → Internet
  (WFH, travel)         Agent-based, SAML auth

Guest/BYOD            → On-prem WSA (transparent)   → Internet
  (unmanaged)            WCCP, IP-based policy

Branch offices        → Cloud proxy or IPsec tunnel → Internet
  (small sites)          to cloud proxy PoP
```

### Migration Considerations

Moving from on-prem to cloud proxy involves:

1. **Policy translation:** Convert WSA access policies to cloud format
2. **Authentication migration:** From NTLM/Kerberos to SAML/agent-based
3. **PAC file update:** Point to cloud proxy endpoints
4. **Certificate deployment:** Deploy cloud proxy CA to endpoints
5. **Traffic routing:** Redirect internet traffic to cloud PoP (IPsec, GRE, agent)
6. **Parallel running:** Run both for validation period
7. **Log integration:** Ensure SIEM receives logs from cloud proxy

---

## See Also

- tls, pki, cisco-ise, dns, waf

## References

- [Cisco Secure Web Appliance Architecture Guide](https://www.cisco.com/c/en/us/td/docs/security/wsa/wsa-15-0/user-guide/b_WSA_UserGuide_15_0.html)
- [RFC 3040 — Internet Web Replication and Caching Taxonomy](https://www.rfc-editor.org/rfc/rfc3040)
- [RFC 1928 — SOCKS Protocol Version 5](https://www.rfc-editor.org/rfc/rfc1928)
- [RFC 7235 — HTTP/1.1 Authentication](https://www.rfc-editor.org/rfc/rfc7235)
- [RFC 8446 — TLS 1.3](https://www.rfc-editor.org/rfc/rfc8446)
- [Cisco WCCP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp/configuration/xe-16/iap-xe-16-book/iap-wccp.html)
- [Cisco Umbrella SIG Documentation](https://docs.umbrella.com/umbrella-user-guide/docs/secure-internet-gateway)
- [NIST SP 800-52 — Guidelines for TLS Implementations](https://csrc.nist.gov/publications/detail/sp/800-52/rev-2/final)
