# Evasion Techniques — IDS, Firewall, AV & Honeypot Bypass Theory

> Deep-dive reference for CEH v13 Module 12. Covers the theory behind evasion
> techniques: how detection systems work, where their weaknesses lie, and why
> specific bypass methods succeed. Pair with the corresponding cheat sheet for
> tool syntax and quick commands.

---

## Prerequisites

- TCP/IP fundamentals (IP fragmentation, TCP state machine, protocol headers)
- Understanding of network security devices (IDS/IPS, firewalls, WAFs)
- Familiarity with malware analysis concepts (static vs dynamic analysis)
- Working knowledge of Linux networking tools (iptables, tcpdump, Wireshark)
- Basic scripting (Python or Bash) for custom payload construction

---

## 1. IP Fragment Reassembly Vulnerabilities

IP fragmentation splits datagrams that exceed the MTU into smaller fragments. Each fragment carries an offset and length. The receiving host must reassemble them. Security devices that inspect traffic must also reassemble — and differences in reassembly policy create evasion opportunities.

**Reassembly policies by OS:**

| Policy | Behavior | Operating Systems |
|--------|----------|-------------------|
| First | Keeps first fragment data at overlapping offset | Linux, macOS, FreeBSD |
| Last | Overwrites with later fragment at overlapping offset | Windows (most versions) |
| Most recent | Uses most recently received data | Cisco IOS (older) |

**Teardrop attack:** Sends fragments with overlapping offsets that produce negative or impossible reassembly math. Older TCP/IP stacks crashed or panicked when attempting reassembly. Modern systems handle this gracefully, but the principle demonstrates that reassembly is a parsing problem with edge cases.

**Overlapping fragment evasion:** The attacker sends two fragments that cover the same byte range with different content. If the IDS uses a "first" policy but the target OS uses "last," the IDS sees benign content while the target receives the malicious payload. The attacker must know (or guess) the target's reassembly policy and the IDS's reassembly policy — if they differ, evasion is possible.

**Tiny fragment attack (RFC 5765 / Ptacek-Newsham):** By setting an extremely small fragment size (e.g., 8 bytes), the TCP header is split across multiple fragments. The first fragment contains only the source and destination ports. Flags, sequence numbers, and payload arrive in subsequent fragments. Many older IDS implementations only inspected the first fragment, missing the actual content.

**Modern defenses:** Current IDS/IPS systems perform full reassembly using configurable OS-specific policies (Snort's `frag3` preprocessor supports per-host policies). However, edge cases in reassembly timeout, maximum fragment count, and fragment overlap handling still create narrow evasion windows in specific configurations.

---

## 2. IDS Detection Algorithms and Their Weaknesses

Intrusion detection systems use several detection methodologies, each with distinct strengths and exploitable weaknesses.

### 2.1 Signature-Based Detection

The IDS maintains a database of known attack patterns (signatures/rules). Each packet or reassembled stream is compared against these patterns.

**Strengths:** High accuracy for known attacks, low false positive rate, fast matching with optimized algorithms (Aho-Corasick, Hyperscan).

**Weaknesses:**
- Cannot detect novel attacks (zero-days) with no existing signature
- Encoding transformations defeat string matching (URL encoding, Unicode normalization, case variation, comment insertion in SQL/HTML)
- Fragmentation and session splicing break pattern continuity if the IDS does not fully reassemble
- Polymorphic payloads change their byte pattern on each execution
- Signature updates lag behind new attack variants

### 2.2 Anomaly-Based Detection

Establishes a baseline of "normal" network behavior (traffic volume, protocol distribution, connection patterns) and flags deviations.

**Strengths:** Can detect previously unknown attacks, no signature updates required, effective against insider threats with unusual access patterns.

**Weaknesses:**
- High false positive rate (legitimate traffic changes trigger alerts)
- Training period is vulnerable — if attacker is active during baseline, malicious traffic becomes "normal"
- Slow/low attacks that stay within normal thresholds evade detection
- Mimicry attacks shape malicious traffic to match normal profiles
- Concept drift: baselines must be retrained as legitimate usage patterns evolve

### 2.3 Stateful Protocol Analysis

Maintains protocol state machines and verifies that traffic conforms to expected protocol behavior (RFC compliance).

**Strengths:** Detects protocol violations and abuse, catches attacks that manipulate protocol state (SYN floods, session hijacking), works across multiple packets.

**Weaknesses:**
- Resource intensive — must track state for every connection
- State table exhaustion: flooding connections can overwhelm the IDS
- Protocol ambiguities: RFCs leave implementation choices to vendors, creating gaps between what the IDS expects and what the target accepts
- Encrypted protocols are opaque to stateful analysis without decryption

### 2.4 Machine Learning / Behavioral Detection

Modern IDS/EDR systems use ML models trained on network flow features, system call sequences, or file characteristics.

**Weaknesses:**
- Adversarial examples: carefully crafted inputs that cause misclassification
- Model drift: requires continuous retraining
- Explainability gap: difficult to tune or understand why a detection fired
- Feature engineering determines effectiveness — attackers who understand the features can craft traffic that presents benign feature values while carrying malicious payload

---

## 3. Honeypot Taxonomy

Honeypots are decoy systems designed to attract, detect, and analyze attacker behavior. Their classification depends on the level of interaction they allow.

### 3.1 Low-Interaction Honeypots

Emulate a limited set of services at the protocol level. No real OS or applications behind them.

**Examples:** Honeyd, Dionaea (partial), KFSensor

**Characteristics:**
- Easy to deploy and maintain (often a single process)
- Limited attack surface — only simulated services
- Capture connection attempts, basic scanning, and automated exploit attempts
- Cannot capture post-exploitation behavior
- Easily fingerprinted: limited protocol depth, identical responses to varied inputs, TCP/IP stack inconsistencies vs claimed OS

**Detection methods:** Send unexpected protocol commands and observe responses. Low-interaction honeypots often return errors or empty responses for valid-but-unusual commands. Compare TCP/IP fingerprint (nmap OS detection) against claimed service versions — mismatches indicate emulation.

### 3.2 Medium-Interaction Honeypots

Provide more realistic service emulation, sometimes using real protocol implementations with sandboxed backends.

**Examples:** Cowrie (SSH/Telnet), Mailoney (SMTP), Glutton

**Characteristics:**
- Allow login attempts and basic command execution in a sandboxed environment
- Capture credentials, commands, and downloaded malware samples
- More convincing than low-interaction but still limited in depth
- Cowrie provides a fake filesystem and command output but lacks real process execution

**Detection methods:** Execute commands that probe system depth — process listing, kernel version checks, filesystem exploration beyond the fake root. Timing analysis: commands execute too quickly or too consistently (no I/O variance). Check for Cowrie's known SSH key fingerprints and specific key exchange behavior.

### 3.3 High-Interaction Honeypots

Real operating systems and applications, fully instrumented for monitoring.

**Examples:** Full VMs with monitoring (Sebek, Argos), MHN (Modern Honey Network) deployments

**Characteristics:**
- Maximum realism — real OS, real services, real vulnerabilities
- Capture complete attack lifecycle including post-exploitation, lateral movement, and exfiltration
- High maintenance cost and risk (attackers could pivot from the honeypot)
- Require strong containment (network isolation, outbound filtering)

**Detection methods:** Difficult to detect technically. Attackers look for contextual clues: no real user activity (empty browser history, no documents, no cron jobs), network isolation patterns, monitoring artifacts (unusual kernel modules, Sebek hooks), or the system appearing "too perfect" (fresh install, no updates, no customization).

### 3.4 Shodan Honeyscore

Shodan's honeyscore API uses machine learning to estimate the probability that a given IP is a honeypot. It analyzes banner consistency, port combinations, response patterns, and known honeypot signatures across Shodan's scan data. Scores range 0.0 (likely real) to 1.0 (likely honeypot). Useful as a quick check during reconnaissance, though not definitive.

---

## 4. AV Detection Methods and Bypass Theory

Antivirus and endpoint protection platforms use multiple detection layers. Effective evasion requires understanding each layer and bypassing them simultaneously.

### 4.1 Signature-Based Detection

The AV engine maintains a database of byte sequences (signatures) extracted from known malware samples. Files are scanned and matched against this database.

**Bypass theory:**
- **Packing:** Compress or encrypt the executable. The packed binary has different bytes than the original. However, the packer stub itself may be signatured — custom packers are more effective than public ones (UPX is immediately flagged by most engines).
- **Crypting:** Encrypt the payload with a custom key. A small stub decrypts and executes in memory. "FUD" (fully undetectable) crypters are valued because their stubs have no known signature — but they become detected once submitted to multi-AV scanners.
- **Code mutation:** Change variable names, instruction order, register allocation, and control flow while preserving functionality. Automated tools exist but manual modification is most effective against signature matching.

### 4.2 Heuristic Detection

Static analysis of file properties without executing: entropy analysis (high entropy suggests encryption/packing), import table analysis (suspicious API calls like VirtualAlloc + WriteProcessMemory + CreateRemoteThread), section characteristics, and structural anomalies.

**Bypass theory:**
- Reduce entropy by adding legitimate-looking data sections
- Use indirect API resolution (GetProcAddress at runtime instead of import table entries)
- Mimic legitimate PE structure (proper resource sections, valid digital signature if possible)
- Embed payload within a legitimate application template

### 4.3 Behavioral Detection

Execute or emulate the file and observe runtime behavior: process injection, registry modification, network connections to known C2 infrastructure, file encryption patterns (ransomware), credential access.

**Bypass theory:**
- **Sandbox detection:** Check for VM artifacts (VM tools processes, specific MAC address prefixes, low resource counts, registry keys). Delay execution past sandbox timeout (sandboxes typically analyze for 30-120 seconds). Check for user interaction (mouse movement, recent documents).
- **API unhooking:** EDR tools hook userland APIs (ntdll.dll). Map a fresh copy of ntdll from disk and redirect calls to bypass hooks.
- **Fileless execution:** Never touch disk — download and execute in memory via PowerShell, .NET reflection, or process hollowing. No file to scan with static methods.
- **LOLBins (Living-off-the-Land Binaries):** Use legitimate, signed OS binaries to execute payloads. certutil, mshta, rundll32, regsvr32, wmic, msiexec are all Microsoft-signed and can download/execute arbitrary code. EDR must distinguish legitimate use from abuse — a fundamentally harder problem.

### 4.4 Machine Learning Detection

ML models trained on large malware corpora learn features that distinguish malicious from benign files (byte n-grams, API call sequences, structural features, behavioral patterns).

**Bypass theory:**
- Adversarial ML: small perturbations to file features (appending benign strings, modifying header fields) can flip classification without changing functionality
- Feature-space attacks: understand what features the model weights heavily and manipulate those specifically
- Gradient-based evasion: if the model architecture is known, compute gradients to find minimal modifications for misclassification
- Practical limitation: ML models are retrained frequently, so evasion is temporary — custom, targeted payloads outperform generic adversarial techniques

### 4.5 AMSI (Antimalware Scan Interface)

Windows provides AMSI as a hook point for security products to scan scripts and memory buffers at runtime. PowerShell, VBScript, JScript, and .NET assemblies pass content through AMSI before execution.

**Bypass theory:** AMSI is implemented in amsi.dll loaded into the process. Patching the AmsiScanBuffer function in memory (setting it to return "clean" immediately) disables scanning for that process. Microsoft patches known bypass strings, so obfuscation of the bypass itself is necessary — an arms race where new obfuscation methods are continuously developed and detected.

---

## 5. Covert Channel Capacity Analysis

A covert channel is a communication path that was not designed for data transfer. Capacity analysis measures how much data can be exfiltrated through a given channel without detection.

### 5.1 Storage Channels

Data is encoded in protocol fields that are not normally inspected or that appear random.

| Channel | Capacity | Notes |
|---------|----------|-------|
| IP ID field | 16 bits/packet | Appears random in many implementations |
| TCP ISN | 32 bits/connection | Must model target's ISN generation to blend in |
| TCP urgent pointer | 16 bits/segment | Rarely used legitimately, may trigger alerts |
| DNS TXT records | ~200 bytes/query | High bandwidth but unusual query volume is detectable |
| DNS subdomain labels | ~60 bytes/query | Encoded in query name, harder to filter |
| ICMP payload | ~64 KB/packet | Large payloads anomalous, small payloads blend in |
| HTTP headers | Variable | Custom headers or cookie values carry encoded data |
| TCP timestamp | 32 bits/segment | Must maintain monotonicity to avoid detection |

### 5.2 Timing Channels

Data is encoded in the timing between events (packets, requests, responses). One bit per timing interval.

**Capacity:** Determined by channel bandwidth and noise floor. Binary timing channels (long delay = 1, short delay = 0) typically achieve 1-10 bits/second for reliability. Higher rates produce errors due to network jitter.

**Detection:** Statistical analysis of inter-packet arrival times. Legitimate traffic follows distributions shaped by application behavior and network conditions. Covert timing channels produce bimodal or unusually regular distributions.

### 5.3 Capacity Constraints

- **Bandwidth vs detectability tradeoff:** Higher data rates through covert channels produce more anomalous traffic patterns. Low-and-slow exfiltration (bytes per minute) is harder to detect but painfully slow for large datasets.
- **Channel reliability:** Network jitter, packet loss, and protocol normalization by middleboxes can corrupt covert data. Error correction coding reduces effective capacity.
- **Cover traffic:** Covert channels embedded in legitimate traffic (steganographic approach) are limited by the cover traffic volume and characteristics.
- **Protocol normalization:** Security devices that rewrite or normalize protocol fields (TCP ISN randomization, IP ID rewriting, DNS response rewriting) can destroy covert channels.

### 5.4 Practical Capacity Examples

- **DNS tunneling (iodine):** 50-100 KB/s under ideal conditions (NULL or TXT records, cooperative DNS resolver). Drops to 5-10 KB/s through restrictive resolvers.
- **ICMP tunneling (ptunnel):** 10-50 KB/s depending on ICMP rate limiting on path.
- **HTTP header channels:** Limited by request frequency. At 1 request/second with 100 bytes/header, theoretical max is 100 B/s — but must maintain realistic request patterns.
- **TCP ISN channel:** 32 bits per new connection. At 1 connection/second = 4 B/s. Extremely slow but very difficult to detect if ISN distribution matches OS behavior.
