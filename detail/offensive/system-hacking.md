> For authorized security testing, red team exercises, and educational study only.

# System Hacking — Deep Dive (CEH Module 06)

> This document covers the theory, algorithm internals, and attack chain mechanics behind system hacking techniques. It complements the cheat sheet with the "why" and "how it works" rather than just the commands.

## Prerequisites

- Solid understanding of operating system fundamentals (processes, memory, file systems)
- Familiarity with cryptographic hash functions and symmetric/asymmetric encryption
- Working knowledge of Active Directory and Kerberos authentication
- Linux command-line proficiency and basic Windows administration
- Networking fundamentals (TCP/IP, SMB, LDAP, DNS)

---

## 1. Password Hash Algorithm Internals

### 1.1 NTLM (NT LAN Manager)

NTLM is the legacy Windows authentication hash. It is fast, unsalted, and considered weak by modern standards.

**Algorithm:**

```
NTLM_hash = MD4(UTF-16LE(password))
```

- The plaintext password is encoded as UTF-16 Little Endian (two bytes per character).
- The MD4 digest is computed over the encoded bytes, producing a 128-bit (16-byte) hash.
- No salt is used. Identical passwords always produce identical hashes across all systems.
- No iteration/key stretching. A single GPU can compute billions of NTLM hashes per second.

**Why it matters:** The lack of salt means NTLM hashes are vulnerable to rainbow tables. The speed of MD4 makes brute-force and dictionary attacks trivially fast. Despite being deprecated, NTLM remains present in modern Windows environments for backward compatibility.

**NTLMv2 challenge-response** adds a server challenge and timestamp to the authentication protocol, but the underlying NT hash stored on disk is still the same MD4 output.

### 1.2 bcrypt

bcrypt is a password hashing function based on the Blowfish cipher, designed to be computationally expensive and resistant to hardware-accelerated cracking.

**Algorithm overview:**

```
bcrypt(cost, salt, password):
  1. Derive 128-bit salt from input salt
  2. key = password (truncated to 72 bytes)
  3. state = EksBlowfishSetup(cost, salt, key)
     - Standard Blowfish key schedule
     - Then 2^cost iterations of:
       - ExpandKey(state, salt)
       - ExpandKey(state, key)
  4. ctext = "OrpheanBeholderScryDoubt" (192-bit constant)
  5. Repeat 64 times: ctext = Encrypt_ECB(state, ctext)
  6. Return concatenation of cost + salt + ctext
```

**Key properties:**

- **Cost factor (work factor):** The `cost` parameter (typically 10-14) controls the number of iterations as 2^cost. Each increment doubles the computation time.
- **72-byte password limit:** Passwords longer than 72 bytes are silently truncated in most implementations.
- **128-bit salt:** Generated randomly per hash, defeating rainbow tables.
- **Memory-hard: No.** bcrypt is CPU-hard but not memory-hard, making it somewhat vulnerable to GPU and ASIC attacks, though significantly slower than MD5/SHA-family.

**Hash format:** `$2b$12$salt22chars.hash31chars.`

### 1.3 scrypt

scrypt was designed to be both CPU-hard and memory-hard, making it expensive to attack with custom hardware (GPUs, ASICs, FPGAs).

**Algorithm overview:**

```
scrypt(password, salt, N, r, p, dkLen):
  1. B = PBKDF2-HMAC-SHA256(password, salt, 1, p * 128 * r)
     - Generates p blocks of 128*r bytes each
  2. For i = 0 to p-1:
       B[i] = ROMix(r, B[i], N)
  3. Return PBKDF2-HMAC-SHA256(password, B, 1, dkLen)
```

**ROMix** is the memory-hard core:

```
ROMix(r, B, N):
  1. Allocate array V of N entries, each 128*r bytes
  2. V[0] = B
  3. For i = 1 to N-1: V[i] = BlockMix(V[i-1])
  4. X = V[N-1]
  5. For i = 0 to N-1:
       j = Integerify(X) mod N     # pseudo-random index into V
       X = BlockMix(X XOR V[j])
  6. Return X
```

**Key properties:**

- **N** (CPU/memory cost): Must be a power of 2. Memory required is approximately 128 * r * N bytes. With N=2^20 and r=8, that is about 1 GB.
- **r** (block size): Controls the sequential memory-read size.
- **p** (parallelism): Number of independent mixing operations.
- The sequential random lookups into array V force the attacker to keep the full array in memory; computing on the fly is prohibitively expensive.

### 1.4 Argon2

Argon2 is the winner of the 2015 Password Hashing Competition. It comes in three variants:

| Variant | Designed For | Memory Access Pattern |
|---------|-------------|----------------------|
| Argon2d | Cryptocurrency, backend | Data-dependent (faster, vulnerable to side-channel) |
| Argon2i | Password hashing (interactive) | Data-independent (resistant to side-channel) |
| Argon2id | Recommended default | Hybrid: first pass is Argon2i, subsequent passes are Argon2d |

**Algorithm overview (Argon2id):**

```
Argon2id(password, salt, timeCost, memoryCost, parallelism, hashLength):
  1. Compute H0 = Blake2b(parameters || password || salt || secret || associated_data)
  2. Allocate memory: matrix of (parallelism) lanes x (memoryCost/parallelism) columns
     - Each cell is a 1024-byte block
  3. Fill phase:
     - First pass: Argon2i indexing (data-independent)
     - Subsequent passes (timeCost - 1): Argon2d indexing (data-dependent)
     - Each block is computed as:
       B[i][j] = G(B[i][j-1], B[l][z])
       where G is a compression function based on Blake2b
       and (l, z) is a reference block determined by the indexing mode
  4. Finalize: XOR the last column across all lanes, then Blake2b to desired length
```

**Key parameters:**

- **timeCost (t):** Number of passes over memory. More passes = more CPU work.
- **memoryCost (m):** Total memory in KiB. Directly controls attack cost.
- **parallelism (p):** Number of threads. Does not reduce security; allows defenders to use multi-core CPUs.

**Why Argon2id is recommended:** It combines side-channel resistance (first pass) with brute-force resistance (subsequent passes). OWASP recommends Argon2id with m=19456 (19 MiB), t=2, p=1 as a minimum.

---

## 2. Rainbow Table Time-Memory Trade-Off Analysis

### 2.1 The Core Problem

For a hash function H and a password space P of size N, a brute-force attack requires O(N) time and O(1) space. Pre-computing all hashes requires O(1) lookup time but O(N) space. Rainbow tables offer a middle ground.

### 2.2 Hellman's Original Trade-Off (1980)

Martin Hellman proposed chains of alternating hash and reduction operations:

```
p0 -H-> h0 -R-> p1 -H-> h1 -R-> p2 -H-> h2 ... -R-> p_t
```

- **H** is the hash function.
- **R** is a reduction function mapping hash outputs back to the password space.
- Only the start point (p0) and end point (p_t) of each chain are stored.
- Chain length t and number of chains m are chosen so that m * t >= N.

**Lookup:** Given a target hash h, apply R then H repeatedly to generate a chain. If any intermediate value matches a stored endpoint, retrieve the corresponding start point and recompute the chain to find the password.

**Problem:** Chain collisions (merges) cause false alarms and reduce coverage.

### 2.3 Rainbow Tables (Oechslin, 2003)

Philippe Oechslin's improvement uses a different reduction function at each position in the chain:

```
p0 -H-> h0 -R1-> p1 -H-> h1 -R2-> p2 -H-> h2 -R3-> p3 ...
```

- R_i is a distinct reduction function for position i in the chain.
- This prevents chain merges: two chains can only collide if they share a value at the same position, dramatically reducing collision probability.

### 2.4 Complexity Analysis

| Parameter | Symbol | Description |
|-----------|--------|-------------|
| Password space size | N | Total possible passwords |
| Chain length | t | Number of links per chain |
| Number of chains | m | Rows stored in the table |
| Coverage per table | | ~N when m * t ~= N |

**Storage:** O(m) -- only start and end points stored, each pair typically 16 bytes.

**Lookup time:** O(t^2) worst case per table (must check up to t positions, each requiring up to t hash/reduce operations). In practice, average case is much better.

**Trade-off equation:**

```
Time * Memory^2 = N^2   (approximately, for Hellman tables)
```

For rainbow tables specifically:

```
Storage = m * (size of start + end point)
Precomputation = m * t hash operations
Online lookup = O(t^2) hash operations per target hash
Success rate ≈ 1 - (1 - 1/N)^(m*t) ≈ 1 - e^(-mt/N)
```

### 2.5 Defenses

- **Salting:** A unique random salt per password multiplies the effective password space by the number of possible salts. A 128-bit salt makes rainbow tables infeasible (would need 2^128 separate tables).
- **Key stretching (iterations):** Makes each hash computation expensive, increasing both precomputation and lookup time proportionally.
- **Memory-hard functions (scrypt, Argon2):** Make each hash computation require significant RAM, preventing massively parallel precomputation on GPUs/ASICs.

**NTLM is vulnerable** because it uses no salt and a single fast MD4 invocation. Rainbow tables for NTLM covering all alphanumeric passwords up to 8 characters fit on a few hundred GB.

---

## 3. Kerberos Attack Chain

### 3.1 Kerberos Protocol Overview

```
                   KDC (Key Distribution Center)
                   ┌─────────┬─────────┐
                   │   AS    │   TGS   │
                   │(AuthSvc)│(TktGrant)│
                   └────┬────┴────┬────┘
          AS-REQ (1)    │         │   TGS-REQ (3)
          ┌─────────────┘         └──────────────┐
          │  AS-REP (2)              TGS-REP (4) │
          ▼                                      ▼
      ┌───────┐         AP-REQ (5)          ┌────────┐
      │Client │ ──────────────────────────> │Service │
      │       │ <────────────────────────── │        │
      └───────┘         AP-REP (6)          └────────┘
```

1. **AS-REQ:** Client sends username + timestamp encrypted with user's password hash to Authentication Service.
2. **AS-REP:** KDC returns a TGT (Ticket Granting Ticket) encrypted with the krbtgt account's hash.
3. **TGS-REQ:** Client presents TGT to request a Service Ticket (TGS) for a specific service.
4. **TGS-REP:** KDC returns a Service Ticket encrypted with the target service account's hash.
5. **AP-REQ:** Client presents the Service Ticket to the target service.
6. **AP-REP:** Service validates the ticket and grants access.

### 3.2 AS-REP Roasting

**Prerequisite:** Target account has "Do not require Kerberos preauthentication" enabled (DONT_REQ_PREAUTH flag).

**Attack flow:**

1. Attacker sends an AS-REQ for the target account without the encrypted timestamp (no preauthentication).
2. The KDC responds with an AS-REP containing data encrypted with the user's password hash.
3. The encrypted portion of the AS-REP can be attacked offline.

**What is cracked:** The AS-REP contains an encrypted part using the user's key (derived from their password). Specifically, the encrypted timestamp in the response uses the user's long-term key (RC4 = NTLM hash, or AES256).

**Hashcat mode:** 18200 (Kerberos 5 AS-REP etype 23 / RC4)

**Why it works:** Without preauthentication, the KDC does not verify the requester knows the password before issuing the response. Any attacker who can reach the KDC can request these.

### 3.3 Kerberoasting

**Prerequisite:** Any authenticated domain user account (low privilege sufficient).

**Attack flow:**

1. Attacker authenticates normally and obtains a TGT.
2. Attacker requests TGS tickets for service accounts (accounts with SPNs -- Service Principal Names).
3. The TGS tickets are encrypted with the service account's password hash.
4. Attacker extracts the ticket and cracks it offline.

**What is cracked:** The TGS-REP contains data encrypted with the service account's long-term key. For RC4-encrypted tickets, this is effectively the NTLM hash of the service account's password.

**Hashcat mode:** 13100 (Kerberos 5 TGS-REP etype 23 / RC4)

**Why it is effective:** Service accounts often have weak passwords, long-standing credentials, and excessive privileges. The attack requires only standard authenticated user access.

### 3.4 Golden Ticket

**Prerequisite:** Compromise of the krbtgt account hash (the KDC's master key).

**Attack flow:**

1. Attacker obtains the krbtgt NTLM hash (e.g., via DCSync, NTDS.dit extraction).
2. Attacker forges a TGT with arbitrary content:
   - Any username (including non-existent users)
   - Any group memberships (e.g., Domain Admins, Enterprise Admins)
   - Any ticket lifetime (typically set to 10 years)
3. This forged TGT is accepted by any TGS in the domain because it is encrypted with the legitimate krbtgt key.

**Impact:** Complete domain compromise. The attacker can impersonate any user, access any service, and persist for as long as the krbtgt hash remains unchanged.

**Detection:**

- TGT lifetimes exceeding domain policy
- TGTs issued for non-existent users
- Ticket encryption downgrade (RC4 when AES is the domain default)
- Event ID 4769 with unusual requesting accounts

**Remediation:** Reset the krbtgt password twice (the KDC remembers the current and previous password). Wait for replication between resets. This invalidates all existing TGTs domain-wide.

### 3.5 Silver Ticket

**Prerequisite:** Compromise of a specific service account's NTLM hash.

**Attack flow:**

1. Attacker obtains the target service account's NTLM hash.
2. Attacker forges a TGS (Service Ticket) directly, bypassing the KDC entirely.
3. The forged ticket grants access to the specific service.

**Key differences from Golden Ticket:**

| Aspect | Golden Ticket | Silver Ticket |
|--------|--------------|---------------|
| Key required | krbtgt hash | Service account hash |
| Scope | Entire domain | Single service |
| KDC interaction | Forges TGT (used with KDC for TGS) | Forges TGS (no KDC contact) |
| Detection | KDC logs show anomalies | No KDC logs generated |
| Stealth | Moderate | High (no DC contact) |

### 3.6 Attack Chain Summary

```
Reconnaissance
  └─> Enumerate users with no preauth (AS-REP Roasting)
  └─> Enumerate service accounts with SPNs (Kerberoasting)
        └─> Crack service account passwords offline
              └─> Use service account to access sensitive systems
                    └─> Escalate to Domain Admin
                          └─> DCSync / NTDS.dit extraction
                                └─> Obtain krbtgt hash
                                      └─> Forge Golden Ticket (full domain persistence)
```

---

## 4. Rootkit Detection Algorithms

### 4.1 Cross-View Detection

The fundamental principle: compare what the OS reports (through potentially compromised APIs) against what a direct examination of raw data reveals.

**Process hiding detection:**

```
Method:
  1. Enumerate processes via standard API (e.g., /proc, ps, CreateToolhelp32Snapshot)
  2. Enumerate processes via alternative methods:
     - Walk /proc/<PID> directories directly
     - Read kernel memory structures (task_struct list)
     - Scan process scheduler queues
  3. Any process visible in (2) but not (1) is suspicious
```

**File hiding detection:**

```
Method:
  1. List files using standard system calls (readdir/FindFirstFile)
  2. List files by directly reading filesystem structures:
     - Parse raw inode tables (ext4) or MFT entries (NTFS)
     - Walk directory entries at the block device level
  3. Files present on raw disk but hidden from API are rootkit indicators
```

### 4.2 Integrity-Based Detection

Compare current system state against a known-good baseline.

**System call table verification:**

```
Method:
  1. Record addresses in the syscall table (sys_call_table) at a known-clean state
  2. Periodically compare current syscall table entries against the baseline
  3. Modified entries indicate syscall hooking

Linux implementation:
  - Read /boot/System.map for expected syscall addresses
  - Compare against current /proc/kallsyms entries
  - Kernel modules modifying syscall table entries are flagged
```

**Binary integrity:**

```
Method (tripwire model):
  1. At install/baseline time, hash all critical system binaries:
     /bin/*, /sbin/*, /usr/bin/*, /lib/*, kernel modules
  2. Store hashes in a signed, tamper-resistant database
  3. Periodically recompute hashes and compare
  4. Mismatches on binaries that should not have changed indicate compromise

Tools: AIDE, Tripwire, OSSEC (file integrity monitoring)
```

### 4.3 Memory Forensics Detection

Analyze a memory dump to identify rootkit artifacts.

**Volatility framework techniques:**

```
1. Process list comparison:
   - pslist (walks EPROCESS doubly-linked list)
   - psscan (scans memory for EPROCESS pool tags)
   - Processes found by psscan but not pslist = DKOM (Direct Kernel Object Manipulation)

2. Syscall hook detection:
   - linux_check_syscall: compares syscall table entries against known module ranges
   - Entries pointing outside kernel text or known modules = hooks

3. Inline hook detection:
   - Disassemble first bytes of critical kernel functions
   - JMP/CALL instructions at function entry pointing to unexpected addresses = inline hooks

4. Hidden kernel module detection:
   - Walk the kernel module list (lsmod equivalent)
   - Scan memory for module structures not in the list
   - Compare against /proc/modules output
```

### 4.4 Behavioral Detection

Monitor system behavior for rootkit-characteristic patterns.

```
Indicators:
  - Network traffic to/from processes not visible in process list
  - Disk I/O attributed to no visible process
  - CPU usage not accounted for by visible processes
  - Unexpected kernel module loads
  - Modified kernel text segment
  - Anomalous interrupt descriptor table (IDT) entries
```

---

## 5. Anti-Forensics Techniques and Countermeasures

### 5.1 Timestamp Manipulation (Timestomping)

**Technique:** Modify file MAC times (Modified, Accessed, Created) to blend malicious files with legitimate system files.

```
NTFS stores two sets of timestamps:
  1. $STANDARD_INFORMATION (SI) — displayed by dir, Explorer, most tools
  2. $FILE_NAME (FN) — maintained by the OS, harder to modify from user mode

Timestomping typically only modifies SI attributes.
Detection: Compare SI timestamps against FN timestamps.
If SI < FN (file appears older than its name entry), timestomping is likely.

Tools:
  - Timestomp (Metasploit): modifies SI timestamps
  - SetMACE: modifies both SI and FN (requires raw NTFS access)
  - MFTECmd (Eric Zimmerman): parses MFT to reveal both timestamp sets
```

**Countermeasures:**

- Compare $STANDARD_INFORMATION vs $FILE_NAME timestamps in MFT analysis
- Correlate file timestamps with USN Journal entries (records file system changes independently)
- Check $LogFile (NTFS journal) for timestamp change operations
- Windows Event Logs may record file creation even if timestamps are altered

### 5.2 Log Manipulation

**Techniques:**

```
1. Log deletion: Clear entire log files (crude, easily noticed)
2. Log editing: Remove specific entries (sed, custom parsers)
3. Log redirection: Modify syslog config to drop specific facilities
4. Log injection: Add false entries to create misleading narrative

Windows Event Log structure (EVTX):
  - Binary XML format, chunked with checksums
  - Simple truncation corrupts the file
  - Targeted record deletion requires recalculating chunk checksums
  - Tools: Danderspritz (leaked NSA tool), EvtxECmd, EventCleaner

Linux syslog:
  - Plain text, trivially editable
  - Remote syslog (rsyslog/syslog-ng to central server) defeats local log tampering
  - Append-only filesystem attributes (chattr +a) make deletion harder
```

**Countermeasures:**

- Central log aggregation (SIEM) with write-once storage
- Log integrity verification (signed logs, blockchain-based audit trails)
- Append-only filesystems and immutable backup copies
- Monitor for log gaps (time periods with no entries when activity is expected)
- Windows: monitor Event ID 1102 (audit log cleared) and 104 (system log cleared)

### 5.3 Data Hiding — Advanced Techniques

```
1. Slack space: Store data in the gap between file end and cluster boundary
   - File is 1000 bytes, cluster is 4096 bytes → 3096 bytes of slack
   - Tools: bmap (Linux), slacker (Metasploit)
   - Detection: Compare logical file size against cluster-aligned allocation

2. Bad block hiding: Mark good disk blocks as bad in the filesystem
   - Data stored in "bad" blocks is invisible to normal file operations
   - Detection: Verify bad blocks actually produce I/O errors

3. Steganographic filesystems: Deniable encryption (e.g., VeraCrypt hidden volumes)
   - Hidden volume inside a decoy volume; different passwords reveal different data
   - Detection: Statistical analysis of "free space" entropy

4. In-memory only (fileless): Malware exists only in RAM
   - Reflective DLL injection, PowerShell in-memory execution
   - Detection: Memory forensics (Volatility), EDR behavioral monitoring
```

### 5.4 Disk and Data Wiping

```
Secure deletion approaches:
  1. Single overwrite (DoD short): Sufficient for modern HDDs
  2. Multiple overwrite (Gutmann 35-pass): Unnecessary for modern drives
  3. Cryptographic erasure: Encrypt data with a random key, then destroy the key
  4. SSD TRIM + Secure Erase: SSD firmware command for flash media

Targeted wiping:
  - SDelete (Sysinternals): overwrites file content and free space
  - shred (Linux): overwrite specific files
  - cipher /w:C:\ (Windows): overwrite free space

Detection of wiped data:
  - MFT entries may persist even after file content is wiped
  - USN Journal records the deletion event
  - Volume Shadow Copies may retain pre-deletion versions
  - SSD wear leveling may preserve old data in spare blocks

Countermeasures:
  - Volume Shadow Copies and VSS-aware backup
  - USN Journal monitoring and preservation
  - Full disk imaging before analysis (never analyze live disk directly)
  - SSD firmware-level acquisition when feasible
```

### 5.5 Network Anti-Forensics

```
Techniques:
  - Encrypted C2 channels (HTTPS, DNS over HTTPS, domain fronting)
  - Protocol tunneling (DNS tunneling, ICMP tunneling)
  - Onion routing (Tor) for attribution resistance
  - Timestomped PCAP files (if attacker has access to capture infrastructure)
  - MAC address spoofing to defeat network access logs

Countermeasures:
  - TLS inspection at network boundary (controversial, privacy implications)
  - DNS query logging and anomaly detection (high-entropy subdomain queries = DNS tunnel)
  - NetFlow/IPFIX analysis for unusual traffic patterns
  - JA3/JA3S fingerprinting for TLS client/server identification
  - Full packet capture on critical segments (high storage cost)
```
