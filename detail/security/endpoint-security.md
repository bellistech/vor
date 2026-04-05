# Endpoint Security — Deep Dive

Theoretical foundations of endpoint protection, detection, and response technologies including malware analysis techniques, retrospective detection, MITRE ATT&CK mapping, and zero-day detection approaches.

## EDR vs EPP vs XDR

### Endpoint Protection Platform (EPP)

EPP focuses on prevention — stopping known threats before they execute. Traditional EPP relies on signature-based detection, heuristic analysis, and behavioral rules applied at the point of execution.

- **Signature matching:** Compare file hash or byte pattern against known malware database. Fast but only detects known threats. Signature databases can contain millions of entries; lookup is typically O(1) via hash table.
- **Heuristic analysis:** Static rules that flag suspicious characteristics (e.g., packed executable, high entropy sections, suspicious imports). Produces more false positives but catches variants.
- **Behavioral blocking:** Monitor process actions in real time; block if behavior matches malware patterns (e.g., rapid file encryption = ransomware).

EPP weakness: cannot detect threats that bypass prevention. Once a threat executes and evades initial checks, EPP has no further visibility.

### Endpoint Detection and Response (EDR)

EDR assumes prevention will fail and focuses on visibility, detection, and response after execution.

- **Continuous telemetry:** Record all process executions, file operations, network connections, registry changes, DNS queries, and inter-process communication.
- **Behavioral analytics:** Correlate sequences of events against attack patterns rather than individual file signatures. Example: Word spawning PowerShell that downloads a binary and creates a scheduled task.
- **Forensic investigation:** Full timeline reconstruction of attacker activity. Device trajectory shows exactly what happened, when, and in what order.
- **Response actions:** Remote isolation, file quarantine, process termination, forensic snapshot collection.

EDR weakness: high data volume requires significant storage and processing. Alert fatigue is common without proper tuning.

### Extended Detection and Response (XDR)

XDR extends EDR by correlating telemetry across multiple security domains:

```
XDR Telemetry Sources:
+------------+     +----------+     +-----------+
| Endpoint   |     | Network  |     | Email     |
| (EDR data) |     | (firewall|     | (mail     |
|            |     |  IDS/IPS)|     |  gateway) |
+-----+------+     +----+-----+     +-----+-----+
      |                  |                 |
      +--------+---------+---------+-------+
               |                   |
         +-----v-----+      +-----v-----+
         | XDR       |      | Cloud     |
         | Analytics |      | Workload  |
         | Engine    |      | Telemetry |
         +-----------+      +-----------+
```

XDR provides cross-domain correlation: an email with a malicious attachment (email telemetry) leads to a file execution (endpoint telemetry) that connects to a C2 server (network telemetry). XDR links these events into a single incident automatically.

## Malware Analysis Techniques

### Static Analysis

Static analysis examines malware without executing it. Increasing levels of depth:

**Level 1 — File metadata:**
- File type identification (magic bytes, not extension)
- File size, timestamps, digital signature verification
- Hash computation (MD5, SHA-1, SHA-256) for lookup in threat intelligence databases
- Fuzzy hashing (ssdeep) for similarity matching against known malware families

**Level 2 — String extraction and structure:**
- Extract ASCII/Unicode strings: URLs, IP addresses, registry paths, API names, error messages
- PE header analysis (Windows): imports, exports, sections, compile timestamp, entry point
- ELF analysis (Linux): section headers, symbol tables, dynamic linking
- Entropy analysis per section: high entropy (>7.0 on 0-8 scale) suggests packing or encryption

**Level 3 — Disassembly and decompilation:**
- Disassemble to assembly language (IDA Pro, Ghidra, radare2)
- Identify function boundaries, control flow graphs, cross-references
- Recognize crypto constants (S-boxes, initialization vectors) to identify encryption algorithms
- Detect anti-analysis techniques: debugger detection, VM detection, timing checks

### Dynamic Analysis

Dynamic analysis executes malware in a controlled environment (sandbox) and observes behavior.

**Sandbox architecture:**
```
+------------------+
| Analysis VM      |
| +------+  +----+ |
| |Malware|  |Mon.| |     +-------------+
| |Sample |  |Agent| +---->| Analysis    |
| +------+  +----+ |     | Engine      |
| +------+  +----+ |     | - API calls |
| |Fake   |  |Net | |     | - File ops  |
| |Services|  |Mon.| |     | - Network   |
| +------+  +----+ |     | - Registry  |
+------------------+     +-------------+
```

**Behavioral indicators captured:**
- Process creation tree (parent-child relationships)
- File system operations (create, modify, delete, encrypt)
- Registry modifications (persistence, configuration)
- Network connections (DNS queries, HTTP requests, C2 communication)
- API calls (CreateRemoteThread, VirtualAllocEx, WriteProcessMemory = code injection)
- Mutex creation (coordination between malware instances)
- Sleep calls and timing (evasion of short sandbox analysis windows)

**Sandbox evasion techniques attackers use:**
- VM detection: check for VMware tools, VirtualBox Guest Additions, CPUID hypervisor bit
- Human interaction detection: wait for mouse movement, keystrokes, or scrolling
- Time-based evasion: sleep for extended periods; check system uptime > 20 minutes
- Environment checks: require minimum RAM, CPU cores, or specific installed software
- Geofencing: only execute in target country/region based on keyboard layout or IP geolocation

### Machine Learning-Based Detection

**Feature extraction for ML classifiers:**
- PE structural features: section count, import table size, resource section entropy
- Opcode frequency analysis: n-gram distribution of instruction opcodes
- API call sequences: ordered list of Windows API calls during execution
- Network behavior features: DNS query patterns, connection frequency, data volume

**Common ML approaches:**
- **Random Forest / Gradient Boosting:** Effective on structured features (PE metadata, import tables). Interpretable, fast inference. Used in many commercial EPP products.
- **Deep Learning (CNN/RNN):** Applied to raw byte sequences or API call sequences. Higher accuracy but requires more training data and compute. Less interpretable.
- **Clustering (unsupervised):** Group unknown files by behavioral similarity to identify new malware families without labeled training data.

**ML limitations:**
- Adversarial examples: small perturbations to binaries that flip classification without changing functionality
- Concept drift: malware evolves, model accuracy degrades over time, requires retraining
- False positives on novel legitimate software that shares features with malware

## Retrospective Detection Theory

Traditional AV makes a point-in-time decision: clean or malicious at time of execution. Retrospective detection decouples the decision point from the enforcement point.

**Architecture:**

```
Time T0: File executes
  |
  v
Cloud lookup: SHA-256 = Unknown disposition
  |
  v
Allow execution (log SHA-256 + endpoint + timestamp)
  |
  v
Time T0+N: New intelligence received
  |
  v
SHA-256 reclassified: Unknown -> Malicious
  |
  v
Query: Which endpoints executed this SHA-256?
  |
  v
Retrospective alert to all affected endpoints
  |
  v
Automated response: quarantine file, alert SOC
```

**Why retrospective detection matters:**
- Zero-day malware has zero signatures at time of release; it will always pass initial checks
- Average dwell time (time between compromise and detection) is measured in days to months
- Retrospective detection can reduce dwell time to minutes once new intelligence arrives
- The entire organization is protected simultaneously, not one endpoint at a time

**Data requirements:**
- Every file execution must be logged with SHA-256 hash, endpoint ID, timestamp, and parent process
- Storage scales linearly with endpoints and activity volume
- Cloud backend must support real-time reverse lookups: given a hash, return all endpoints

## File Reputation and Sandbox Analysis

### File Reputation System

File reputation assigns a trust score based on multiple factors:

- **Prevalence:** How many endpoints globally have seen this file. High prevalence + no detections = likely clean.
- **Age:** How long the file has been known. Old files with no malicious reports gain trust over time.
- **Source:** Files from trusted publishers (valid code signing) start with higher reputation.
- **Behavioral history:** Any past sandbox analysis results or behavioral detections.
- **Community intelligence:** Detections reported by other organizations using the same platform.

Reputation score is continuous, not binary. Thresholds determine disposition:
- Score > 85: Clean disposition (allow)
- Score 50-85: Unknown (monitor, potentially sandbox)
- Score < 50: Malicious disposition (block)

### Sandbox Analysis Pipeline

```
File submission
  |
  v
Pre-filter: known hash? --> Yes --> Return cached result
  |
  No
  v
Static pre-analysis: file type, size, structure
  |
  v
VM selection: match OS/architecture to target environment
  |
  v
Execution: run for 5-10 minutes with behavioral monitoring
  |
  v
Post-analysis: correlate indicators, generate report
  |
  v
Scoring: combine static + dynamic indicators
  |
  v
Update global reputation database
```

## MITRE ATT&CK Mapping

MITRE ATT&CK provides a knowledge base of adversary tactics and techniques based on real-world observations. Endpoint security products map detections to ATT&CK for standardized classification.

### Tactics (the attacker's goal at each stage)

| Tactic | Description | Endpoint Visibility |
|--------|-------------|-------------------|
| Initial Access | How the attacker gets in | Email attachment, browser exploit |
| Execution | Running malicious code | Process creation, script execution |
| Persistence | Maintaining access across reboots | Registry run keys, scheduled tasks, services |
| Privilege Escalation | Gaining higher permissions | Token manipulation, exploit, UAC bypass |
| Defense Evasion | Avoiding detection | Process injection, obfuscation, timestomping |
| Credential Access | Stealing credentials | LSASS memory access, keylogging |
| Discovery | Learning the environment | Network scanning, account enumeration |
| Lateral Movement | Moving to other systems | PsExec, RDP, WMI, SMB |
| Collection | Gathering target data | Screen capture, clipboard, file staging |
| Command and Control | Communicating with attacker infrastructure | HTTP/HTTPS beaconing, DNS tunneling |
| Exfiltration | Stealing data out | Encrypted channel, cloud storage upload |
| Impact | Damage or disruption | Encryption (ransomware), wiper, defacement |

### Technique-to-Detection Mapping Example

```
T1055 — Process Injection
  Subtechniques:
    T1055.001 — Dynamic-link Library Injection
    T1055.002 — Portable Executable Injection
    T1055.003 — Thread Execution Hijacking
    T1055.012 — Process Hollowing

  Detection data sources:
    - Process: OS Credential Dumping (API monitoring)
    - Process: Process Access (handle opened to remote process)
    - Process: Process Modification (memory writes to remote process)
    - Module: Module Load (DLL loaded from unusual path)

  EDR detection logic:
    IF process A calls OpenProcess() on process B
    AND process A calls VirtualAllocEx() in process B
    AND process A calls WriteProcessMemory() to process B
    AND process A calls CreateRemoteThread() in process B
    THEN flag T1055 — Process Injection
```

## Threat Hunting on Endpoints

### Hypothesis-Driven Hunting

Threat hunting starts with a hypothesis derived from threat intelligence, industry reports, or known TTPs.

**Hunting loop:**
1. **Hypothesis:** "Attackers may be using PowerShell to download second-stage payloads via encoded commands."
2. **Data collection:** Query all endpoints for PowerShell executions with `-EncodedCommand` or `-e` flags.
3. **Analysis:** Decode Base64 commands, identify any that contain `Invoke-WebRequest`, `Net.WebClient`, or `DownloadString`.
4. **Findings:** Classify results as true positive (malicious), benign (legitimate automation), or indeterminate (requires further investigation).
5. **Response:** If malicious, initiate incident response. If benign, document to reduce future false positives.
6. **Feedback:** Update detection rules to automate this hunt going forward.

### Endpoint Telemetry for Hunting

Key data sources for endpoint threat hunting:

- **Process execution logs:** Command line arguments, parent process, user context, working directory
- **Network connections:** Destination IP/domain, port, protocol, bytes transferred, timing
- **File operations:** Files created in temp directories, renamed extensions, high-entropy files
- **Registry changes:** New run keys, services, scheduled tasks, COM objects
- **Authentication events:** Failed logins, privilege escalation, lateral movement attempts
- **DNS queries:** Algorithmically generated domains (DGA), unusual TLD usage, high query volume

### Hunting Queries (Orbital/osquery Style)

```sql
-- Find processes with suspicious parent-child relationships
SELECT p.name, p.cmdline, p.path, pp.name AS parent_name
FROM processes p
JOIN processes pp ON p.parent = pp.pid
WHERE pp.name = 'winword.exe'
  AND p.name IN ('powershell.exe', 'cmd.exe', 'wscript.exe', 'mshta.exe');

-- Find recently created files with high entropy in temp directories
SELECT path, size, mtime, btime
FROM file
WHERE directory LIKE '%\Temp\%'
  AND btime > (SELECT datetime('now', '-24 hours'))
  AND size > 10000;

-- Find processes making outbound connections to non-standard ports
SELECT p.name, p.cmdline, s.remote_address, s.remote_port
FROM process_open_sockets s
JOIN processes p ON s.pid = p.pid
WHERE s.remote_port NOT IN (80, 443, 53, 8080, 8443)
  AND s.remote_address NOT LIKE '10.%'
  AND s.remote_address NOT LIKE '172.16.%'
  AND s.remote_address NOT LIKE '192.168.%';
```

## Endpoint Telemetry Collection

### Collection Architecture

```
Endpoint Agent
  |
  +-- Kernel hooks (syscall interception, file system filter drivers)
  |
  +-- User-mode hooks (API hooking, DLL injection monitoring)
  |
  +-- ETW consumers (Windows Event Tracing)
  |
  +-- Audit subsystem (Linux auditd, macOS Endpoint Security Framework)
  |
  v
Local buffer (ring buffer, typically 50-100 MB)
  |
  v
Compression + encryption
  |
  v
Transport (HTTPS to cloud, syslog to SIEM, or local storage)
```

### Data Volume Considerations

A typical endpoint generates 5-50 MB of telemetry per day depending on activity level and verbosity:

- Process events: 1-5 MB/day (creation, termination, command lines)
- Network events: 1-10 MB/day (connections, DNS queries)
- File events: 2-20 MB/day (creation, modification, deletion)
- Registry events: 0.5-5 MB/day (Windows only)
- Authentication events: 0.1-1 MB/day

For an organization with 10,000 endpoints, this translates to 50-500 GB/day of raw telemetry. Retention periods of 30-90 days require 1.5-45 TB of storage.

**Optimization strategies:**
- Filter at the agent: only send events matching detection rules or hunting queries
- Deduplicate: identical events from the same process within a time window
- Compress: typical 5-10x compression ratio on structured telemetry
- Tiered storage: hot (7 days, fast query), warm (30 days, slower), cold (90+ days, archive)

## Zero-Day Detection Approaches

Zero-day exploits target previously unknown vulnerabilities with no existing signatures. Detection requires behavioral and anomaly-based approaches.

### Exploit Prevention Techniques

- **Control Flow Integrity (CFI):** Verify that program execution follows the expected control flow graph. ROP chains and JOP gadgets violate CFI.
- **Address Space Layout Randomization (ASLR):** Randomize memory layout to make exploitation unreliable. ASLR bypass detection (information leaks) is itself a detection opportunity.
- **Data Execution Prevention (DEP/NX):** Mark data pages as non-executable. Attempts to execute code from data regions trigger hardware exceptions.
- **Stack canaries:** Detect stack buffer overflows by checking canary values before function return.
- **Heap integrity checks:** Detect heap corruption (use-after-free, double-free) at allocation boundaries.

### Behavioral Zero-Day Detection

Since zero-days have no signatures, detection focuses on exploitation behavior rather than exploit code:

```
Normal application behavior:
  Word.exe -> reads .docx -> renders content -> user interaction

Exploitation behavior:
  Word.exe -> reads .docx -> spawns cmd.exe -> downloads payload
                          -> allocates RWX memory
                          -> creates scheduled task
                          -> connects to unusual IP

Detection rule:
  IF known_office_app spawns shell_process
  OR known_office_app allocates executable_memory > threshold
  OR known_office_app creates network connection to uncategorized IP
  THEN flag as potential exploitation
```

### Machine Learning for Zero-Day Detection

- **Anomaly detection:** Build a baseline of normal endpoint behavior, flag deviations. Effective for detecting novel attack patterns but generates false positives during legitimate behavior changes (software updates, new workflows).
- **Behavioral sequence modeling:** Train RNN/LSTM on sequences of API calls from benign processes, flag sequences with low probability scores.
- **Graph-based detection:** Model process relationships as graphs; detect unusual graph structures (e.g., a browser spawning a command interpreter spawning a network tool).

## See Also

- cloud-security, security-operations

## References

- [MITRE ATT&CK Framework](https://attack.mitre.org/)
- [MITRE ATT&CK for Enterprise — Techniques](https://attack.mitre.org/techniques/enterprise/)
- [NIST SP 800-83 Rev 1 — Guide to Malware Incident Prevention and Handling](https://csrc.nist.gov/publications/detail/sp/800-83/rev-1/final)
- [Practical Malware Analysis — Sikorski & Honig](https://nostarch.com/malware)
- [The Art of Memory Forensics — Ligh, Case, Levy, Walters](https://www.wiley.com/en-us/The+Art+of+Memory+Forensics-p-9781118825099)
- [Cisco Secure Endpoint Architecture White Paper](https://www.cisco.com/c/en/us/products/security/amp-for-endpoints/white-paper.html)
- [SANS Endpoint Detection and Response Survey](https://www.sans.org/white-papers/)
- [Gartner EPP Magic Quadrant](https://www.gartner.com/reviews/market/endpoint-protection-platforms)
