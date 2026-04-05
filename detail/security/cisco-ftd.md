# The Engineering of Cisco FTD — Packet Pipeline, Snort Internals, and NAT Processing

> *FTD merges two fundamentally different engines — a stateful firewall (ASA/Lina) and a deep packet inspection engine (Snort) — into a single pipeline where every packet traverses both, creating both power and complexity.*

---

## 1. FTD Packet Processing Pipeline (DAQ to Snort to Verdict)

### Pipeline Architecture

```
                        +-----------+
   Ingress NIC -------> | Prefilter | ---> Fast-path (bypass Snort)
                        +-----------+
                              |
                              v
                        +-----------+
                        |    DAQ    | (Data Acquisition Layer)
                        +-----------+
                              |
                        Shared Memory
                              |
                              v
                    +------------------+
                    |   Snort Engine   |
                    |  +------------+  |
                    |  | SSL Decrypt|  |
                    |  +------------+  |
                    |  | App ID     |  |
                    |  +------------+  |
                    |  | ACP Match  |  |
                    |  +------------+  |
                    |  | IPS Rules  |  |
                    |  +------------+  |
                    |  | File/AMP   |  |
                    |  +------------+  |
                    +------------------+
                              |
                         Verdict
                     (allow/drop/block)
                              |
                              v
                    +------------------+
                    |   Lina (ASA)     |
                    |  +------------+  |
                    |  | Conn State |  |
                    |  +------------+  |
                    |  | NAT        |  |
                    |  +------------+  |
                    |  | Routing    |  |
                    |  +------------+  |
                    |  | QoS        |  |
                    |  +------------+  |
                    +------------------+
                              |
                              v
                        Egress NIC
```

### DAQ (Data Acquisition Layer)

The DAQ is the bridge between Lina and Snort:

1. **Lina receives packet** on ingress interface
2. **Prefilter evaluation:** If fast-path, skip Snort entirely
3. **DAQ copies packet** to Snort via shared memory ring buffer
4. **Snort processes** and returns a verdict via DAQ
5. **Lina acts on verdict:** forward, drop, or inject modified packet

### Shared Memory Performance

The DAQ uses a ring buffer in shared memory. Performance characteristics:

$$Throughput_{max} = \frac{Ring\_Size \times Packet\_Size}{T_{snort\_processing}}$$

Snort processing time varies by enabled features:

| Feature | Additional Latency per Packet |
|---------|------------------------------|
| App ID (first packets only) | 50-200 us |
| IPS (rule evaluation) | 10-100 us |
| SSL decryption | 200-1000 us |
| File inspection | 100-500 us |
| AMP cloud lookup | 1-50 ms (async) |

### Verdict Types

| Verdict | DAQ Code | Meaning |
|---------|----------|---------|
| PASS | DAQ_VERDICT_PASS | Allow packet |
| BLOCK | DAQ_VERDICT_BLOCK | Drop packet silently |
| REPLACE | DAQ_VERDICT_REPLACE | Allow with modified content |
| WHITELIST | DAQ_VERDICT_WHITELIST | Allow and fast-path remaining packets in flow |
| BLACKLIST | DAQ_VERDICT_BLACKLIST | Drop this and all future packets in flow |
| IGNORE | DAQ_VERDICT_IGNORE | Remove from Snort processing (fast-path) |

### First-Packet vs Subsequent Packets

For a new connection:
1. First few packets (typically 3-7) go through full Snort evaluation for app identification
2. Once app is identified and ACP rule matched, Snort caches the verdict
3. Subsequent packets may get WHITELIST verdict (Trust action) or continue full inspection (Allow action with IPS)

---

## 2. Snort IPS Engine Internals

### Snort Architecture (Snort 3 on Modern FTD)

```
Packet Input
     |
     v
+----------+     +-----------+     +------------+
| Decoder  | --> | Preprocessor | --> | Detection  |
+----------+     +-----------+     |  Engine    |
                      |            +------------+
              Stream reassembly         |
              HTTP normalization   Pattern matching
              DNS inspection       (Aho-Corasick +
              Frag reassembly      PCRE + Hyperscan)
                                        |
                                   +----------+
                                   |  Output  |
                                   | (events, |
                                   |  alerts) |
                                   +----------+
```

### Detection Engine

The Snort detection engine uses a multi-pattern matching algorithm:

1. **Aho-Corasick automaton:** Matches all content patterns simultaneously in a single pass through the packet
2. **PCRE (Perl Compatible Regular Expressions):** For complex pattern matching after Aho-Corasick narrows candidates
3. **Hyperscan (Intel):** Hardware-accelerated regex matching on supported platforms

### Rule Processing

A Snort rule example:
```
alert tcp $EXTERNAL_NET any -> $HOME_NET $HTTP_PORTS (
    msg:"SQL Injection attempt";
    flow:to_server,established;
    content:"SELECT"; nocase;
    content:"FROM"; nocase; distance:0;
    pcre:"/SELECT\s+.*FROM\s+/i";
    classtype:web-application-attack;
    sid:1000001; rev:1;
)
```

Processing order for this rule:
1. **Protocol match:** TCP
2. **IP/Port match:** External to internal HTTP ports
3. **Flow match:** Established, client-to-server
4. **Content match (Aho-Corasick):** "SELECT" and "FROM" found
5. **PCRE match:** Regex validation
6. **Alert generated** if all conditions pass

### Rule Evaluation Complexity

For $N$ rules with $P$ total unique content patterns:

$$T_{detection} = T_{aho\_corasick}(P) + T_{pcre}(M_{candidates})$$

Where:
- $T_{aho\_corasick}(P)$: Linear in packet size, independent of $P$ (compiled automaton)
- $T_{pcre}(M_{candidates})$: Applied only to rules whose content patterns matched (much smaller than $N$)
- Aho-Corasick makes the detection engine $O(packet\_length)$, not $O(N \times packet\_length)$

### Snort Instance Model

FTD runs multiple Snort instances (one per CPU core):

$$Instances = CPU\_Cores - Reserved\_Cores$$

Traffic is distributed across instances by flow (5-tuple hash). Each instance:
- Has its own memory space and state
- Processes flows independently
- Can be restarted individually (with traffic impact for that instance's flows)

---

## 3. ACP Rule Evaluation Order

### Evaluation Sequence

```
1. Prefilter Policy (fast-path / analyze / tunnel)
         |
2. SSL Policy (decrypt / do-not-decrypt / block)
         |
3. Identity Policy (user-to-IP mapping)
         |
4. ACP Rules (top-down, first match)
    Rule 1: Zone/Network/Port/App/URL/User -> Action
    Rule 2: ...
    Rule N: ...
         |
5. Default Action (Block / Trust / Intrusion Prevention)
```

### ACP Rule Matching Details

ACP rules may require multiple packets to evaluate:

| Condition Type | Packets Needed | Notes |
|----------------|---------------|-------|
| Zone, Network, Port | 1 (SYN) | Available from first packet |
| VLAN tag | 1 | Available from first packet |
| Application | 3-7 | App ID needs initial handshake |
| URL | 2+ (HTTP) or TLS SNI | HTTP Host header or TLS ClientHello |
| User | 1 (if mapping exists) | Lookup in user-IP cache |

### Pending Rule Behavior

When a rule requires app identification but the app is not yet identified:
1. Packets are **allowed through** while identification is pending
2. Once app is identified, the matching rule's action is applied retroactively
3. If the action is Block, a TCP RST is sent (connection already partially established)
4. This means **the first few packets of blocked traffic always pass through**

### Rule Optimization

$$T_{evaluation} = \sum_{i=1}^{K} T_{condition\_check_i}$$

Where $K$ is the number of conditions checked before a match. Optimization strategies:
- Place most-hit rules at the top (reduces average $K$)
- Use zone/network conditions first (evaluated on first packet, cheapest)
- Group rules by zone pair (FMC does this internally)
- Minimize URL/App conditions on high-volume rules (require multi-packet evaluation)

---

## 4. SSL Decryption Architecture

### Decrypt-Resign Flow

```
Client          FTD                      Server
  |               |                        |
  |--ClientHello->|                        |
  |               |---ClientHello--------->|
  |               |<--ServerHello,Cert-----|
  |               |                        |
  |               | [FTD verifies server cert]
  |               | [Generates re-signed cert using internal CA]
  |               |                        |
  |<-ServerHello--|                        |
  | [Re-signed    |                        |
  |  cert with    |                        |
  |  internal CA] |                        |
  |               |                        |
  |--Key Exchange>|                        |
  |               |---Key Exchange-------->|
  |               |                        |
  |===TLS 1=======|========TLS 2==========|
  | (FTD CA cert) | (Original server cert) |
  |               |                        |
  | Decrypted     | Re-encrypted           |
  | for Snort     | for server             |
```

Two separate TLS sessions:
- **TLS 1:** Client <-> FTD (uses re-signed certificate)
- **TLS 2:** FTD <-> Server (uses original server certificate)

### Decrypt-Known Key Flow

For inbound traffic to servers where FTD has the private key:

```
Client          FTD                      Server
  |               |                        |
  |--ClientHello---------------->--------->|
  |<-ServerHello,Cert-----------<----------|
  |               |                        |
  | [FTD has server's private key]         |
  | [Passively decrypts the session]       |
  |               |                        |
  | FTD can inspect decrypted traffic      |
  | without modifying the TLS session      |
```

Note: Decrypt-Known Key only works with RSA key exchange (not ECDHE/DHE). Modern TLS 1.3 uses ephemeral keys exclusively, making this mode increasingly limited.

### Performance Impact

SSL decryption is the most CPU-intensive operation:

$$CPU_{SSL} \approx \frac{N_{sessions} \times (Handshake\_Cost + Bulk\_Cost)}{CPU\_Capacity}$$

| Operation | CPU Cost (relative) |
|-----------|-------------------|
| RSA 2048 handshake | 100x baseline |
| ECDHE handshake | 30x baseline |
| AES-256-GCM bulk | 3x baseline |
| No decryption | 1x baseline |

Hardware acceleration (crypto offload ASICs on 4100/9300) significantly reduces this overhead.

---

## 5. FMC-FTD Communication (sftunnel)

### sftunnel Architecture

FMC and FTD communicate over an encrypted tunnel called **sftunnel** (Sourcefire tunnel):

```
FMC                                    FTD
  |                                      |
  |<======= sftunnel (TCP 8305) ========>|
  |                                      |
  | Policy deployment (FMC -> FTD)       |
  | Event data (FTD -> FMC)             |
  | Health monitoring (bidirectional)    |
  | File transfer (policy, updates)     |
  |                                      |
```

### Communication Types

| Direction | Data | Frequency |
|-----------|------|-----------|
| FMC -> FTD | Policy deployment | On-demand (user-initiated) |
| FMC -> FTD | Rule updates (SRU) | Scheduled (daily/weekly) |
| FMC -> FTD | VDB updates | Scheduled |
| FTD -> FMC | Connection events | Real-time or batched |
| FTD -> FMC | Intrusion events | Real-time |
| FTD -> FMC | File/malware events | Real-time |
| FTD -> FMC | Health data | Periodic (5 min default) |

### Policy Deployment Process

1. Admin clicks Deploy on FMC
2. FMC compiles policy into Snort configuration files and Lina CLI
3. FMC packages and transfers via sftunnel
4. FTD receives and validates package
5. Lina applies CLI changes (may cause brief connection drops)
6. Snort reloads configuration (Snort restart if major changes)
7. FTD reports deployment status to FMC

### Deployment Impact

| Change Type | Traffic Impact |
|-------------|---------------|
| ACP rule add/modify | Snort reload (brief disruption, seconds) |
| NAT change | Lina reload (existing connections may drop) |
| Interface change | Interface flap possible |
| Snort version upgrade | Snort restart (seconds of inspection gap) |

---

## 6. NAT Rule Processing Order

### Three-Section Model

```
Section 1: Manual NAT (Twice NAT)
  Rule 1.1
  Rule 1.2
  ...
  (Evaluated top-down, first match wins)

Section 2: Auto NAT
  Auto-NAT rules sorted by:
    1. Static NAT before dynamic
    2. Longer prefix before shorter
    (Automatic ordering, not user-controlled)

Section 3: Manual NAT (after-auto)
  Rule 3.1
  Rule 3.2
  ...
  (Evaluated top-down, first match wins)
```

### Processing Details

For each packet, NAT is evaluated twice:

**Un-NAT (destination translation, inbound direction):**
1. Check Section 1 for matching destination translation
2. Check Section 2 for matching static NAT (destination side)
3. Check Section 3

**NAT (source translation, outbound direction):**
1. Check Section 1 for matching source translation
2. Check Section 2 for matching dynamic/static NAT (source side)
3. Check Section 3

### Auto NAT Sorting Algorithm

Within Section 2, auto NAT rules are sorted automatically:

1. **Static rules** before **dynamic rules**
2. Within static/dynamic: **longest prefix first** (most specific)
3. If prefix length is equal: **lowest network address first**

Example ordering:
```
1. Static NAT for 10.1.1.100/32 (most specific static)
2. Static NAT for 10.1.1.0/24
3. Static NAT for 10.1.0.0/16
4. Dynamic PAT for 10.1.1.0/24 (dynamic after all static)
5. Dynamic PAT for 10.0.0.0/8
```

### Twice NAT (Manual NAT) Capabilities

Twice NAT can translate **both** source and destination in a single rule:

$$Packet_{translated} = f(Src_{orig}, Dst_{orig}) \rightarrow (Src_{new}, Dst_{new})$$

Use cases:
- DNS doctoring (translate DNS response payload)
- Overlapping address resolution
- Policy NAT (translate based on destination, not just source)
- U-turn NAT (hairpin, internal to internal via public IP)

---

## 7. FTD vs ASA Comparison

### Architectural Differences

| Aspect | ASA | FTD |
|--------|-----|-----|
| Inspection engine | ASA MPF (Modular Policy Framework) | Snort |
| App awareness | Basic (NBAR-like) | Full app identification (4000+ apps) |
| IPS | External module (legacy) | Inline Snort IPS |
| File inspection | None (basic type checking) | AMP (cloud-based malware detection) |
| URL filtering | Basic (SmartFilter) | Category + reputation (Cisco TALOS) |
| Management | ASDM, CLI | FMC, FDM, CLI |
| Clustering | Active/active (up to 16 units) | Active/standby only (HA) |
| Multi-context | Yes (virtual firewalls) | Multi-instance (4100/9300 only) |
| Throughput | Higher (simpler pipeline) | Lower (Snort overhead) |
| Packet flow | Single engine | Dual engine (Lina + Snort) |

### When ASA Features Are Missing in FTD

Some ASA features require FlexConfig on FTD:
- EIGRP (partially supported natively now)
- Advanced WCCP
- Policy-based routing (added in later FTD versions)
- Some VPN features (older crypto maps)
- Multicast (IGMP, PIM — partial support)

### Migration Considerations

$$Risk_{migration} = f(Feature\_Gaps, Config\_Complexity, Testing\_Coverage)$$

Key gaps to evaluate:
1. Multi-context -> multi-instance (requires 4100/9300)
2. Active/active clustering -> active/standby HA
3. MPF policies -> ACP + prefilter + SSL policy
4. ASDM workflows -> FMC workflows

---

## 8. Multi-Instance Architecture

### Container-Based Isolation

On Firepower 4100/9300, FTD runs in container instances:

```
+-----------------------------------------------+
|           FXOS (Firepower eXtensible OS)       |
|  +----------+  +----------+  +----------+     |
|  | FTD      |  | FTD      |  | FTD      |     |
|  | Instance |  | Instance |  | Instance |     |
|  | 1        |  | 2        |  | 3        |     |
|  +----------+  +----------+  +----------+     |
|  | vNICs    |  | vNICs    |  | vNICs    |     |
|  +----------+  +----------+  +----------+     |
|                                                |
|  Physical NICs allocated to instances          |
+-----------------------------------------------+
```

### Resource Profiles

| Profile | CPU Cores | RAM |
|---------|-----------|-----|
| Small | 6 | 22 GB |
| Medium | 10 | 30 GB |
| Large | 20 | 54 GB |
| Custom | Configurable | Configurable |

Each instance gets dedicated CPU and memory. There is no oversubscription.

### Instance Throughput

$$Throughput_{instance} \approx \frac{Total\_Platform\_Throughput \times Instance\_Cores}{Total\_Cores}$$

For a Firepower 9300 with 72 cores and 3 large instances (20 cores each):
$$Throughput_{per\_instance} \approx \frac{72Gbps \times 20}{72} = 20 Gbps$$

(Approximate; actual throughput depends on traffic mix, enabled features, and packet size.)

---

## 9. Threat Intelligence (TID / TALOS)

### TALOS Feed Integration

Cisco TALOS provides threat intelligence feeds that integrate with FTD via FMC:

| Feed | Content | Update Frequency |
|------|---------|-----------------|
| Snort Rule Updates (SRU) | IPS signatures | Daily/weekly |
| VDB (Vulnerability Database) | App/OS fingerprints | Monthly |
| Geolocation Database | IP-to-country mapping | Monthly |
| URL Category Database | URL classifications | Continuous |
| AMP Cloud | File dispositions | Real-time |
| Security Intelligence (SI) | IP/URL/DNS blacklists | Minutes |

### Security Intelligence (SI) Processing

SI is evaluated **before** ACP rules:

```
Packet --> SI Check --> If blacklisted: Block (never reaches ACP)
                   --> If whitelisted: Continue to ACP
                   --> If not listed: Continue to ACP
```

SI sources:
- TALOS intelligence feeds (IP, URL, DNS)
- Custom blacklists/whitelists (manual or STIX/TAXII)
- Third-party threat feeds

### Threat Intelligence Director (TID)

TID aggregates multiple threat intelligence sources:
- STIX/TAXII feeds
- Flat file indicators (IP, URL, SHA256)
- Custom indicators
- Publishes unified indicators to FTD for enforcement

---

## See Also

- iptables, nftables, ipsec, radius, cisco-ise

## References

- [Cisco FTD Architecture White Paper](https://www.cisco.com/c/en/us/products/security/firepower-ngfw/white-paper-listing.html)
- [Snort 3 Architecture](https://www.snort.org/snort3)
- [Cisco FTD Configuration Guide](https://www.cisco.com/c/en/us/td/docs/security/firepower/70/configuration/guide/fpmc-config-guide-v70.html)
- [Cisco FXOS Configuration Guide](https://www.cisco.com/c/en/us/td/docs/security/firepower/fxos/fxos271/web-guide/b_GUI_FXOS_ConfigGuide_271.html)
- [Cisco TALOS Intelligence Group](https://talosintelligence.com/)
- [NIST SP 800-41 — Guidelines on Firewalls and Firewall Policy](https://csrc.nist.gov/publications/detail/sp/800-41/rev-1/final)
