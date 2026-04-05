# Digital Forensics — Theory, Legal Framework, and Advanced Analysis

> *Digital forensics bridges technology and law. Every tool choice, every acquisition method, every analytical step must withstand legal scrutiny. The theoretical foundations of evidence integrity, file system analysis, and memory forensics determine whether digital evidence proves a case or gets excluded from court.*

---

## 1. Legal Admissibility Requirements

### Standards for Digital Evidence Admissibility

Digital evidence must satisfy general evidence rules adapted for the digital domain:

**Authentication (Federal Rules of Evidence 901):**
The proponent must produce evidence sufficient to support a finding that the evidence is what they claim it is. For digital evidence:
- Hash verification proves the analyzed copy matches the original
- Chain of custody proves continuous control
- Forensic methodology proves the analysis was conducted properly

**Best Evidence Rule (FRE 1002):**
The original is required to prove content. For digital evidence, an accurate copy is generally acceptable if:
- The copy process is verified (hash match)
- The original is unavailable or impractical to produce
- No genuine question of authenticity exists

**Hearsay (FRE 802/803):**
Computer-generated records may be:
- **Business records exception (803(6)):** Regularly maintained records (logs, database entries)
- **Not hearsay at all:** Computer output that is not a "statement" (automated logs vs human-authored emails)

### Daubert vs Frye Standards

| Standard | Jurisdiction | Test |
|:---|:---|:---|
| Daubert | Federal courts, majority of states | Judge as gatekeeper: testable, peer-reviewed, known error rate, standards, accepted |
| Frye | Some states (CA, NY, FL, others) | "General acceptance" in the relevant scientific community |
| Federal Rules 702 | Federal | Expert may testify if: sufficient facts, reliable principles, reliably applied |

### International Considerations

| Jurisdiction | Framework | Key Requirement |
|:---|:---|:---|
| EU/GDPR | Data protection law | Evidence collection must comply with data protection principles |
| UK | Police and Criminal Evidence Act (PACE) | S.69 (repealed) but integrity standards remain via case law |
| US | 4th Amendment | Warrant required for government searches (exceptions: consent, exigent circumstances) |
| Australia | Electronic Transactions Act | Electronic records admissible if integrity maintained |

### Search and Seizure (US Law)

| Scenario | Warrant Needed? | Authority |
|:---|:---|:---|
| Government criminal investigation | Yes (with exceptions) | 4th Amendment |
| Corporate internal investigation | No (employer-owned systems) | Employment agreement |
| Civil litigation (e-discovery) | Subpoena/court order | Federal Rules of Civil Procedure |
| Consent search | No (if valid consent) | Voluntary consent |
| Exigent circumstances | No (imminent destruction of evidence) | Exception to warrant requirement |
| Border search | No (reduced expectation of privacy) | Border search exception |

---

## 2. Evidence Integrity — Cryptographic Hashing

### Hash Functions for Forensics

| Algorithm | Output Size | Status | Usage |
|:---|:---|:---|:---|
| MD5 | 128 bits | Broken (collisions) | Legacy verification, not sufficient alone |
| SHA-1 | 160 bits | Broken (collisions) | Legacy, being phased out |
| SHA-256 | 256 bits | Secure | Current standard for forensics |
| SHA-3-256 | 256 bits | Secure | Alternative to SHA-2 |
| BLAKE3 | 256 bits | Secure, very fast | Emerging for large datasets |

### Integrity Verification Protocol

**At acquisition:**
1. Hash the source media before imaging: $H_{source} = \text{SHA-256}(source)$
2. Create forensic image
3. Hash the image: $H_{image} = \text{SHA-256}(image)$
4. Verify: $H_{source} = H_{image}$
5. Document both hashes with timestamp and examiner signature

**At each transfer:**
1. Hash before transfer: $H_{before}$
2. Transfer evidence
3. Hash after transfer: $H_{after}$
4. Verify: $H_{before} = H_{after}$

**At analysis:**
1. Hash working copy before analysis
2. Perform analysis on working copy (never the original image)
3. Hash working copy after analysis (should match — read-only analysis)
4. If modification needed, document and re-hash

### Collision Resistance and Legal Implications

For SHA-256, the probability of accidental collision:

$$P(\text{collision in } n \text{ hashes}) \approx \frac{n^2}{2^{257}}$$

For $n = 10^{18}$ (one quintillion images), $P \approx 10^{-41}$. This is astronomically unlikely, making SHA-256 sufficient for legal proceedings.

**MD5 weakness:** Practical collision attacks exist (2004, Wang et al.). Two different files can have the same MD5 hash. However, finding a meaningful file that collides with a specific evidence hash (second preimage) remains computationally infeasible. Courts generally still accept MD5 as one of multiple integrity checks, but SHA-256 is preferred.

---

## 3. Forensic Imaging Theory

### Imaging Methods

| Method | Description | Forensic Soundness | Speed |
|:---|:---|:---|:---|
| Physical (raw/dd) | Bit-for-bit copy of entire media | Highest | Medium |
| Logical | File system level copy | Lower (misses deleted, slack) | Fast |
| Sparse | Only copies allocated blocks | Medium (smaller image) | Fast |
| Live | Image of running system | Necessary but less sound | Variable |

### Disk Geometry and Addressing

**Logical Block Addressing (LBA):** Modern drives address sectors linearly (0 to $N-1$).

$$\text{Drive capacity} = \text{sector count} \times \text{sector size (512 or 4096 bytes)}$$

**Host Protected Area (HPA) and Device Configuration Overlay (DCO):**
Hidden areas of disk not visible to the OS but may contain evidence.

```bash
# Detect HPA
hdparm -N /dev/sda
# If reported size < actual size, HPA exists

# Remove HPA temporarily (for acquisition)
hdparm -N p<real_max_sectors> /dev/sda

# Detect DCO
hdparm --dco-identify /dev/sda
```

**Forensic implication:** A complete forensic image must include HPA/DCO areas, as attackers may use them to hide data.

### Write Blocking Theory

A forensic write blocker intercepts I/O commands between the host and evidence drive:

**Allowed commands:** Read, identify device, read capacity
**Blocked commands:** Write, format, trim, secure erase, write cache flush

**NIST SP 800-72** defines testing requirements for write blockers:
- No command that modifies the drive shall be transmitted
- All read commands shall be passed without modification
- Any data returned by the drive shall be passed without modification
- If a write command is transmitted, it shall be returned with an error

---

## 4. File System Forensics

### NTFS Artifacts

**Master File Table (MFT):**
Every file and directory has at least one MFT entry (1024 bytes). Key attributes:

| Attribute | Content | Forensic Value |
|:---|:---|:---|
| $STANDARD_INFORMATION | Created, modified, accessed, entry modified times | Timestomping detection (only this is easily modified) |
| $FILE_NAME | Created, modified, accessed times + filename | Harder to modify — compare with $SI for timestomping |
| $DATA | File content (resident if <700 bytes, non-resident otherwise) | The actual file data |
| $INDEX_ROOT/$INDEX_ALLOCATION | Directory entries ($I30 index) | Contains deleted file entries |
| $BITMAP | Allocation status of clusters | Identifies free space for carving |

**Timestomping detection:**

$STANDARD_INFORMATION timestamps can be modified with tools like Timestomp. $FILE_NAME timestamps are updated only by the kernel and are much harder to fake.

$$\text{If } T_{SI} < T_{FN} \implies \text{possible timestomping}$$
$$\text{If } T_{SI\_created} > T_{SI\_modified} \implies \text{definite timestomping}$$

**$UsnJrnl (USN Journal):**
Change journal recording all file system modifications. Even if a file is deleted and the MFT entry overwritten, the USN Journal may retain a record of its existence and operations performed on it.

**Volume Shadow Copies (VSS):**
Point-in-time snapshots of volumes. May contain previous versions of modified/deleted files:

```bash
# List shadow copies
vssadmin list shadows

# Mount shadow copy for analysis
mklink /D C:\shadow \\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy1\
```

### ext4 Forensics

| Structure | Forensic Value |
|:---|:---|
| Superblock | File system metadata, mount count, last mount time |
| Inode table | File metadata, timestamps, block pointers |
| Journal (jbd2) | Transaction log — may contain deleted file data |
| Extent tree | File block mapping (replaces indirect blocks) |
| Directory entries | Linked list of filenames → inodes |

**ext4 deletion behavior:**
- Inode zeroed (timestamps, size set to 0)
- Directory entry removed
- Blocks marked free in bitmap
- Journal may retain pre-deletion inode and data blocks

**Recovery approach:**
1. Parse journal for deleted inodes
2. Match orphaned data blocks using file carving
3. Use extundelete or ext4magic for structured recovery

### APFS Forensics (macOS)

| Feature | Forensic Implication |
|:---|:---|
| Copy-on-write (CoW) | Previous versions may exist in free space |
| Snapshots | Built-in point-in-time recovery |
| Encryption (FileVault) | Class keys protect metadata; recovery key required |
| Space sharing | Multiple volumes share space — cross-volume artifacts |
| Nanosecond timestamps | Higher precision timeline analysis |

---

## 5. Memory Analysis Methodology

### Memory Structure

| Region | Contents | Forensic Value |
|:---|:---|:---|
| Kernel space | OS kernel, drivers, system tables | Rootkit detection, syscall table verification |
| Process memory | Code, heap, stack, environment | Running programs, command history, credentials |
| File cache | Recently accessed files | Reconstructing file access |
| Network buffers | Packet data in transit | Network connections, data in flight |
| Mapped files | DLLs, shared libraries | Loaded code analysis |

### Analysis Workflow

```
1. Profile identification
   - Determine OS version and architecture
   - Select correct symbol tables / profile

2. Process analysis
   - List all processes (pslist, pstree)
   - Find hidden processes (psscan — pool tag scanning)
   - Compare: processes visible to API vs pool scan
   - Hidden process = rootkit indicator

3. Network analysis
   - Active connections (netscan)
   - Listening ports
   - Correlate with processes

4. Code injection detection
   - malfind: VAD regions with PAGE_EXECUTE_READWRITE
   - Compare in-memory code against on-disk binary
   - Detect hollowed processes (process image mismatch)

5. Credential extraction
   - LSASS process memory (Windows)
   - /etc/shadow cached in memory (Linux)
   - SSH keys in process memory
   - Browser stored passwords

6. Artifact recovery
   - Command history (consoles, bash)
   - Clipboard contents
   - Registry hives from memory
   - Encryption keys (BitLocker, FileVault, LUKS)
```

### Rootkit Detection via Memory Analysis

| Technique | Description | Tool |
|:---|:---|:---|
| DKOM | Direct Kernel Object Manipulation — unlinking process from list | Compare pslist vs psscan |
| SSDT hooking | Modifying System Service Descriptor Table | check_ssdt, compare vs clean |
| IDT hooking | Modifying Interrupt Descriptor Table | check_idt |
| IRP hooking | Intercepting I/O Request Packets | check_irp |
| Inline hooking | Patching function prologues | Compare code vs disk |
| Hidden modules | Unlinking from kernel module list | modscan vs lsmod |

---

## 6. Timeline Analysis

### Super Timeline Construction

A super timeline merges timestamps from all available sources into a single chronological view:

**Sources:**
- File system timestamps (MACB: Modified, Accessed, Changed, Born)
- Event logs (Windows, syslog)
- Browser history and cache
- Registry last-write timestamps
- Prefetch timestamps
- Email headers
- Network logs and flow data
- Application-specific logs

**Tools:**

```bash
# Plaso / log2timeline (comprehensive timeline tool)
log2timeline.py --storage-file timeline.plaso /evidence/case001.dd
psort.py -o l2tcsv -w timeline.csv timeline.plaso

# Filter by time range
psort.py -o l2tcsv -w filtered.csv timeline.plaso \
  "date > '2026-04-01 00:00:00' AND date < '2026-04-05 23:59:59'"

# The Sleuth Kit mactime
fls -r -m "/" -o 2048 case001.dd > bodyfile.txt
mactime -b bodyfile.txt -d -z UTC > timeline.csv
```

### MACB Timestamp Interpretation

| Action | Modified | Accessed | Changed ($MFT) | Born (Created) |
|:---|:---:|:---:|:---:|:---:|
| File created | M | A | C | B |
| File read | | A | | |
| File written | M | A | C | |
| File renamed | | | C | |
| Permissions changed | | | C | |
| File copied (new file) | M* | A | C | B |
| File moved (same vol) | | | C | |

\* Copied file gets source's Modified time but new Created time

---

## 7. Anti-Forensics Countermeasures

### Detection Strategies

| Anti-Forensics Technique | Detection Method |
|:---|:---|
| Secure deletion (shred, sdelete) | Statistical analysis of free space (non-random patterns), gap analysis in file sequences |
| Timestomping | $SI vs $FN timestamp comparison, USN Journal cross-reference |
| Log deletion | Log sequence gaps, Event ID 1102 (audit log cleared), remote log comparison |
| Disk encryption | Cannot bypass without key; focus on RAM (key material) or legal compulsion |
| Steganography | Chi-squared analysis, LSB analysis, known stego tool signatures |
| Fileless malware | Memory forensics, ETW traces, PowerShell ScriptBlock logging, WMI persistence |
| Data hiding (ADS, HPA, slack) | Alternate data stream enumeration, HPA detection, slack space analysis |
| Virtual machine usage | VM artifacts on host, VM image forensics |

### Steganography Detection (Steganalysis)

**Visual inspection:** Limited effectiveness; human eye cannot detect LSB changes.

**Statistical analysis:**

**Chi-squared attack** (for LSB embedding in images):
Compare the frequency distribution of pixel values. Clean images have natural variation; LSB-embedded images show characteristic flattening of adjacent value pairs.

$$\chi^2 = \sum_{i=0}^{k} \frac{(f_{2i} - f_{2i+1})^2}{f_{2i} + f_{2i+1}}$$

Where $f_{2i}$ and $f_{2i+1}$ are frequencies of adjacent pixel values. A low $\chi^2$ value in regions that should have natural variation suggests steganographic embedding.

**Tools:** StegDetect, StegExpose, zsteg (PNG), stegsolve (visual analysis).

---

## 8. Cloud Forensics Challenges

### Fundamental Challenges

| Challenge | Description | Mitigation |
|:---|:---|:---|
| Lack of physical access | Cannot seize hardware | API-based acquisition (snapshots, log export) |
| Multi-tenancy | Shared infrastructure | Provider cooperation, isolation guarantees |
| Data jurisdiction | Data may span multiple countries | Contractual data residency, legal coordination |
| Elasticity | Instances auto-scale, terminate | Pre-configured forensic readiness (logging, snapshots) |
| Encryption | Provider-managed or customer-managed keys | Key escrow, key management procedures |
| Log availability | Logs may be limited or time-bounded | Forward to external SIEM, extended retention |
| Shared responsibility | Provider manages infrastructure | Clear delineation of forensic responsibilities |

### Cloud Provider Forensic Capabilities

| Capability | AWS | Azure | GCP |
|:---|:---|:---|:---|
| Disk snapshot | EBS Snapshots | Managed Disk Snapshots | Persistent Disk Snapshots |
| Memory acquisition | Not directly possible | Not directly possible | Not directly possible |
| API audit log | CloudTrail | Activity Log | Cloud Audit Logs |
| Network flow | VPC Flow Logs | NSG Flow Logs | VPC Flow Logs |
| DNS query log | Route 53 Query Logs | DNS Analytics | Cloud DNS Logging |
| Container forensics | ECS/EKS task logs | AKS logs | GKE logs |
| Serverless forensics | Lambda CloudWatch Logs | Functions App Insights | Cloud Functions Logs |

### Cloud Forensic Readiness

Pre-incident preparation to ensure evidence availability:

1. **Enable comprehensive logging:** CloudTrail (all regions), VPC Flow Logs, DNS query logging
2. **Centralize logs:** Forward to forensic SIEM with extended retention (1+ year)
3. **Immutable storage:** Write-once storage for critical logs (S3 Object Lock, Azure Immutable Blob)
4. **Automated snapshots:** Scheduled snapshots of critical volumes
5. **Forensic account:** Separate account with cross-account access for evidence isolation
6. **Incident response playbooks:** Pre-built automation for evidence collection
7. **Legal preparation:** Provider SLA review, law enforcement contact procedures

---

## 9. Forensic Readiness

### Forensic Readiness Planning (ISO 27043)

Proactive preparation to maximize forensic capability and minimize investigation cost:

| Component | Purpose |
|:---|:---|
| Evidence sources inventory | Know what data is available and where |
| Retention policies | Ensure evidence is available when needed |
| Collection procedures | Documented, tested acquisition workflows |
| Tool validation | Verified forensic tools with known good results |
| Training program | Responders trained on evidence handling |
| Legal coordination | Pre-established relationships with legal counsel |
| Chain of custody forms | Ready-to-use documentation templates |
| Forensic workstations | Pre-configured analysis environments |

### Cost-Benefit of Forensic Readiness

$$\text{Cost of readiness} = C_{tools} + C_{training} + C_{storage} + C_{personnel}$$
$$\text{Cost without readiness} = C_{investigation\_delay} + C_{evidence\_loss} + C_{legal\_risk}$$

Organizations with forensic readiness programs typically reduce investigation time by 60-80% and significantly reduce the risk of evidence being ruled inadmissible.

---

## 10. Incident Response to Forensics Handoff

### Handoff Triggers

| Scenario | IR Handles | Forensics Takes Over |
|:---|:---|:---|
| Malware infection | Containment, eradication | Root cause analysis, attribution |
| Data breach | Stop the bleeding, notification | Scope determination, evidence for legal |
| Insider threat | Account suspension, access revocation | Evidence collection for HR/legal action |
| Fraud | Initial detection, preservation | Detailed analysis, expert testimony |
| Regulatory investigation | Initial response, legal counsel | Evidence production, compliance documentation |

### Handoff Protocol

```
1. IR team secures the scene
   - Isolate affected systems (network, not power off)
   - Document system state (screenshots, notes)
   - Begin chain of custody

2. Preserve volatile evidence
   - Memory dump (IR or forensics, depending on skill)
   - Running processes, network connections
   - System time verification

3. Formal handoff meeting
   - IR team briefs forensics on incident timeline
   - Transfer evidence custody (documented)
   - Share indicators of compromise (IOCs)
   - Define forensic investigation scope
   - Establish communication protocol

4. Forensics team takes over
   - Forensic imaging of affected systems
   - Deep analysis per methodology
   - Regular status updates to IR/legal

5. Findings feed back to IR
   - Root cause analysis informs eradication
   - Scope determination guides containment
   - IOCs from forensics improve detection
```

### Evidence Preservation During IR

**Critical rule:** Containment actions must not destroy evidence.

| IR Action | Evidence Risk | Mitigation |
|:---|:---|:---|
| Power off server | Lose volatile memory | Memory dump first |
| Reimage system | Lose disk evidence | Forensic image first |
| Change passwords | Alert attacker | Coordinate timing with forensics |
| Block IP at firewall | Attacker adapts | Capture traffic first |
| Deploy EDR agent | Modifies disk | Document installation, hash before/after |
| Patch vulnerability | Changes system state | Image first, then patch |

---

## References

- NIST SP 800-86: Guide to Integrating Forensic Techniques into Incident Response
- NIST SP 800-72: Guidelines on PDA Forensics (write blocker testing methodology)
- NIST SP 800-101r1: Guidelines on Mobile Device Forensics
- ISO/IEC 27037: Guidelines for Identification, Collection, Acquisition and Preservation of Digital Evidence
- ISO/IEC 27041: Guidance on Assuring Suitability of Investigative Methods
- ISO/IEC 27042: Guidelines for Analysis and Interpretation of Digital Evidence
- ISO/IEC 27043: Incident Investigation Principles and Processes
- RFC 3227: Guidelines for Evidence Collection and Archiving
- SWGDE Best Practices for Digital Evidence
- Federal Rules of Evidence (FRE) Rules 702, 803(6), 901, 1002
- Daubert v. Merrell Dow Pharmaceuticals, 509 U.S. 579 (1993)
- Carrier, B. "File System Forensic Analysis" (2005)
- Casey, E. "Digital Evidence and Computer Crime" (3rd ed., 2011)
