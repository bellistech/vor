# The Mathematics of Metasploit — Exploit Framework Architecture and Payload Engineering

> *Metasploit is a modular exploit framework where each attack is a composition of exploit + payload + encoder + post module. The mathematics involve shellcode encoding to evade detection, payload size constraints, and the probability calculus of exploit reliability.*

---

## 1. Framework Architecture — Modular Composition

### Module Types

$$\text{Attack} = \text{Exploit} \circ \text{Payload} \circ \text{Encoder} \circ \text{NOP sled (optional)}$$

| Module Type | Count (MSF 6) | Purpose |
|:---|:---:|:---|
| Exploits | ~2,200 | Deliver payload via vulnerability |
| Payloads (singles) | ~600 | Self-contained shellcode |
| Payloads (stagers) | ~200 | Small loader → download stage |
| Payloads (stages) | ~100 | Full payload delivered by stager |
| Encoders | ~45 | Evade signature detection |
| Post modules | ~350 | Post-exploitation actions |
| Auxiliary | ~1,200 | Scanning, fuzzing, DoS |

### Payload Size Constraints

| Vulnerability Type | Typical Buffer Size | Usable Space |
|:---|:---:|:---:|
| Stack overflow (small) | 256-512 bytes | 100-400 bytes |
| Stack overflow (large) | 1024-4096 bytes | 500-3500 bytes |
| Heap overflow | Variable | Limited by chunk size |
| Format string | Unlimited (via writes) | N/A (write-what-where) |
| Web exploit (command inj) | Unlimited | Line length (~8 KB) |

### Staged vs Singles

| Approach | Size | Reliability | Stealth |
|:---|:---:|:---:|:---:|
| Single payload | 300-5000 bytes | Higher (one shot) | Lower (large) |
| Stager + stage | 50-200 bytes (stager) | Lower (two-stage) | Higher (small stager) |

Stager size is critical when buffer space is limited:

$$\text{Stager fits} \iff |\text{stager}| \leq |\text{buffer}| - |\text{NOP}| - |\text{return address}|$$

---

## 2. Shellcode Encoding — Evasion Mathematics

### Why Encode?

Raw shellcode often contains bad characters (null bytes, newlines) and matches AV signatures.

### XOR Encoder (shikata_ga_nai)

The `shikata_ga_nai` encoder uses polymorphic XOR:

$$C_i = P_i \oplus K_{(i \bmod |K|)}$$

Where $P_i$ is plaintext byte $i$, $K$ is the key, and $C_i$ is encoded byte.

Each encoding produces different output (key is random):

$$|\text{Unique encodings}| = 256^{|K|}$$

For a 4-byte key: $256^4 = 4.3 \times 10^9$ unique variants.

### Multi-Encoding

Encoding $n$ times with different encoders:

$$\text{Encoded} = E_n(E_{n-1}(\cdots E_1(\text{payload})))$$

### Encoder Size Overhead

| Encoder | Overhead | Bad Chars Avoided | AV Bypass Rate |
|:---|:---:|:---|:---:|
| xor (simple) | 10-20 bytes | Null | Low |
| shikata_ga_nai (1 pass) | 30-50 bytes | Configurable | Medium |
| shikata_ga_nai (5 passes) | 150-250 bytes | Configurable | Higher |
| alpha_mixed | 2x payload size | Non-alphanumeric | Medium |
| unicode_mixed | 3x payload size | Non-unicode | Low |

### Total Payload Size

$$|\text{final}| = |\text{NOP sled}| + |\text{decoder stub}| + |\text{encoded payload}| + |\text{alignment}|$$

---

## 3. Meterpreter — Post-Exploitation Platform

### Meterpreter Architecture

| Component | Size | Function |
|:---|:---:|:---|
| Stager (reverse_tcp) | ~200 bytes | Connect back, load stage |
| Stage (meterpreter) | ~750 KB | Full post-exploitation |
| Extensions (loaded) | 50-200 KB each | stdapi, priv, kiwi, etc. |

### Communication Protocol

Meterpreter uses TLV (Type-Length-Value) packets over encrypted channel:

$$\text{Packet} = \text{XOR key}(4) \| \text{session GUID}(16) \| \text{encrypt IV}(16) \| \text{AES-CBC}(\text{TLV data})$$

### Detection Evasion Metrics

| Technique | Detection Rate (2024) | Notes |
|:---|:---:|:---|
| Default meterpreter | 60-80% | Well-known signatures |
| Encoded (shikata x3) | 30-50% | Pattern matching still works |
| Custom stager (C) | 5-15% | No known signatures |
| Reflective DLL | 10-20% | Memory-only, no disk |
| Custom C2 protocol | <5% | No signature match |

---

## 4. Exploit Reliability Mathematics

### Success Probability Model

$$P(\text{shell}) = P(\text{vuln exists}) \times P(\text{exploit works}) \times P(\text{payload delivers}) \times P(\text{no AV/EDR})$$

### Worked Examples

**Example 1: MS17-010 (EternalBlue)**

| Factor | Probability | Reason |
|:---|:---:|:---|
| Vulnerable | 0.30 | Unpatched Windows 7/Server 2008 |
| Exploit works | 0.95 | Highly reliable |
| Payload delivers | 0.90 | Kernel-level, pre-auth |
| No AV block | 0.70 | Well-known, some AV catches |
| **Total** | **0.18** | ~1 in 5 targets |

**Example 2: Apache Struts RCE (CVE-2017-5638)**

| Factor | Probability | Reason |
|:---|:---:|:---|
| Vulnerable | 0.15 | Specific Struts versions |
| Exploit works | 0.99 | Deterministic (OGNL injection) |
| Payload delivers | 0.95 | Command execution |
| No AV block | 0.90 | Server-side, less AV coverage |
| **Total** | **0.13** | ~1 in 8 targets |

---

## 5. NOP Sled Mathematics

### Purpose

A NOP sled provides a landing zone for imprecise return address targeting:

$$P(\text{hit sled}) = \frac{|\text{NOP sled}|}{|\text{exploitable region}|}$$

### Worked Example

Buffer: 1024 bytes. NOP sled: 500 bytes. Shellcode: 200 bytes. Return address precision: $\pm$ 200 bytes.

$$P(\text{hit}) = \frac{500}{400} = 1.0 \quad \text{(sled covers the entire uncertainty range)}$$

Without NOP sled (must hit exact address):

$$P(\text{hit}) = \frac{1}{400} = 0.25\%$$

NOP sled improves reliability by **400x** in this example.

### NOP Alternatives

| NOP Type | Byte | Detectability |
|:---|:---:|:---|
| x86 NOP (0x90) | 1 byte | High (signature: long NOP runs) |
| Multi-byte NOP | 2-9 bytes | Medium |
| Equivalent instructions | varies | Low (e.g., `xchg eax, eax`) |
| Random safe instructions | varies | Very low |

---

## 6. Pivoting — Network Traversal

### Route Through Compromised Host

Metasploit routes traffic through Meterpreter sessions:

$$\text{Attacker} \xrightarrow{\text{session 1}} \text{Host A} \xrightarrow{\text{network}} \text{Host B (internal)}$$

### Pivot Chain Depth

| Depth | Hops | Latency Added | Reliability |
|:---:|:---:|:---:|:---:|
| 0 (direct) | 0 | 0 ms | 99% |
| 1 (single pivot) | 1 | 50-200 ms | 90% |
| 2 (double pivot) | 2 | 100-500 ms | 80% |
| 3 (triple pivot) | 3 | 200-1000 ms | 65% |

Reliability drops with each hop:

$$P(\text{chain intact}) = \prod_{i=1}^{n} P(\text{session } i \text{ alive}) = 0.9^n$$

### Port Forwarding Bandwidth

Through Meterpreter TCP relay:

$$\text{Effective bandwidth} = \min(\text{link speeds along path})$$

Typically 1-10 Mbps through Meterpreter (encrypted tunnel overhead).

---

## 7. Database Backend — Tracking Attack State

### Workspace Data Model

| Table | Records Per Engagement | Purpose |
|:---|:---:|:---|
| Hosts | 100-10,000 | Discovered hosts |
| Services | 500-50,000 | Open ports/services |
| Vulns | 50-5,000 | Identified vulnerabilities |
| Creds | 10-1,000 | Harvested credentials |
| Loots | 10-500 | Exfiltrated files |
| Notes | 100-5,000 | Analyst annotations |

### Attack Surface Quantification

$$\text{Attack surface} = \sum_{h \in \text{hosts}} \sum_{s \in \text{services}(h)} \text{vuln\_count}(s)$$

### Credential Reuse Graph

$$G_{creds} = (H, E) \text{ where } (h_i, h_j) \in E \iff \text{cred from } h_i \text{ works on } h_j$$

Average credential reuse in enterprise: 15-30% of systems share at least one credential.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| $P_i \oplus K_i$ (XOR) | Modular arithmetic | Shellcode encoding |
| $256^{|K|}$ variants | Exponential | Polymorphic encoding |
| $\prod P_i$ reliability | Probability product | Exploit success rate |
| NOP sled / range | Ratio (probability) | Exploit reliability |
| $0.9^n$ chain survival | Geometric decay | Pivot reliability |
| Host $\times$ service $\times$ vuln | Counting | Attack surface |

---

*Metasploit transforms exploitation from ad-hoc scripting into systematic engineering — each exploit is a probability calculation, each payload is a size optimization, and each pivot extends the attack graph deeper into the target network.*
