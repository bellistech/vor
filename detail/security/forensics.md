# The Mathematics of Digital Forensics — Evidence Integrity and Analysis

> *Digital forensics is applied mathematics: disk imaging relies on sector arithmetic, evidence integrity depends on hash collision probability, file carving uses signature pattern matching, and timeline analysis is event correlation across ordered sequences.*

---

## 1. Disk Imaging Mathematics

### Image Size Calculation

$$\text{Image size} = \text{sector size} \times \text{sector count}$$

Standard sector sizes:

| Sector Size | Typical Use | 1 TB Disk Sectors |
|:---:|:---|:---:|
| 512 bytes | Legacy HDD, older drives | $2{,}048{,}000{,}000$ |
| 4096 bytes (4K) | Modern HDD/SSD | $256{,}000{,}000$ |

### Imaging Time Estimation

$$T_{image} = \frac{\text{disk capacity}}{\text{read speed}} + T_{hash}$$

| Drive Type | Read Speed | 1 TB Image Time | Hash Overhead |
|:---|:---:|:---:|:---:|
| HDD (5400 RPM) | 80 MB/s | 3.5 hours | +15 min |
| HDD (7200 RPM) | 150 MB/s | 1.9 hours | +15 min |
| SATA SSD | 550 MB/s | 31 min | +10 min |
| NVMe SSD | 3500 MB/s | 4.9 min | +10 min |

### Write Blocker Verification

A write blocker must guarantee zero writes. Verification:

$$H(\text{source before}) = H(\text{source after}) \implies \text{no writes occurred}$$

Where $H$ is SHA-256. This is the **forensic soundness** proof.

### Image Formats and Overhead

| Format | Compression | Metadata | Overhead |
|:---|:---:|:---|:---:|
| dd (raw) | None | None | 0% |
| E01 (EnCase) | zlib | Case info, hash | 30-60% smaller |
| AFF4 | zlib/lz4 | Arbitrary metadata | 30-60% smaller |
| QCOW2 | zlib | Snapshots | Variable |

Compressed image size: $S_{compressed} \approx S_{raw} \times (1 - r)$ where $r$ is compression ratio (typically 0.3-0.6 for used space).

---

## 2. Hash Integrity — Collision Probability

### Hash Collision (Birthday Problem)

For a hash function with $n$-bit output, the probability of collision among $k$ hashes:

$$P(\text{collision}) \approx 1 - e^{-k^2 / (2 \times 2^n)}$$

For 50% collision probability:

$$k_{50\%} \approx 1.177 \times 2^{n/2}$$

### Collision Resistance by Algorithm

| Algorithm | Output Bits | 50% Collision at | Status |
|:---|:---:|:---:|:---|
| MD5 | 128 | $2^{64} \approx 1.8 \times 10^{19}$ | Broken (practical collisions) |
| SHA-1 | 160 | $2^{80} \approx 1.2 \times 10^{24}$ | Broken (SHAttered, 2017) |
| SHA-256 | 256 | $2^{128} \approx 3.4 \times 10^{38}$ | Secure |
| SHA-3-256 | 256 | $2^{128}$ | Secure |

### Forensic Hash Verification

Chain of custody requires **dual hashing** — compute both MD5 and SHA-256:

$$P(\text{both collide}) = P(\text{MD5 collision}) \times P(\text{SHA-256 collision}) \approx 0$$

Even though MD5 is broken for intentional collisions, the probability of an accidental collision matching BOTH algorithms is negligible.

### Worked Example: Evidence Integrity

Imaging a 500 GB drive:

1. Compute: $\text{MD5}(\text{source}) = \text{d41d8cd9...}$
2. Compute: $\text{SHA256}(\text{source}) = \text{e3b0c442...}$
3. Create image
4. Verify: $\text{MD5}(\text{image}) \stackrel{?}{=} \text{MD5}(\text{source})$ AND $\text{SHA256}(\text{image}) \stackrel{?}{=} \text{SHA256}(\text{source})$

If both match: evidence integrity proven to $2^{-384}$ probability of error.

---

## 3. File Carving — Header/Footer Signature Matching

### The Algorithm

File carving recovers files from raw disk images by searching for known byte patterns:

$$\text{Carve}(I, H, F) = \{(s, e) : I[s:s+|H|] = H \land I[e-|F|:e] = F \land s < e\}$$

Where $I$ is the image, $H$ is the header signature, $F$ is the footer signature.

### Common File Signatures

| File Type | Header (hex) | Footer (hex) | Max Size |
|:---|:---|:---|:---:|
| JPEG | `FF D8 FF E0/E1` | `FF D9` | 20 MB |
| PNG | `89 50 4E 47` | `49 45 4E 44 AE 42 60 82` | 50 MB |
| PDF | `25 50 44 46` | `25 25 45 4F 46` | 100 MB |
| ZIP | `50 4B 03 04` | `50 4B 05 06` | variable |
| ELF | `7F 45 4C 46` | none (use size field) | variable |
| SQLite | `53 51 4C 69 74 65` | none (use header size) | variable |

### False Positive Rate in Carving

The probability of a random byte sequence matching a $k$-byte header:

$$P(\text{false header}) = \frac{1}{256^k}$$

| Header Length | $P(\text{false match})$ | Expected False Matches (1 TB) |
|:---:|:---:|:---:|
| 2 bytes | $1.5 \times 10^{-5}$ | 16 million |
| 3 bytes | $6.0 \times 10^{-8}$ | 64,000 |
| 4 bytes | $2.3 \times 10^{-10}$ | 250 |
| 8 bytes | $1.5 \times 10^{-19}$ | ~0 |

JPEG's 3-byte effective signature (`FF D8 FF`) yields manageable false positives. Longer signatures like PNG's 8-byte magic are essentially false-positive-free.

---

## 4. Timeline Analysis — Event Correlation

### Timestamp Sources

| Source | Resolution | Clock Skew Risk | Trustworthiness |
|:---|:---:|:---:|:---|
| File system (NTFS) | 100 ns | Low (OS managed) | Easily tampered |
| Syslog | 1 second | Medium (NTP dependent) | Moderate |
| Windows Event Log | 100 ns | Low | High (protected) |
| Network capture (pcap) | 1 $\mu$s | Low (capture system) | High |
| Browser history (SQLite) | 1 $\mu$s | Low | Moderate |

### Clock Skew Correction

If two systems have clocks offset by $\Delta t$:

$$t_{corrected} = t_{observed} - \Delta t$$

Estimating $\Delta t$ from known correlated events (e.g., TCP handshakes captured on both sides):

$$\Delta t = \frac{1}{n} \sum_{i=1}^{n} (t_{A,i} - t_{B,i} - \text{RTT}/2)$$

### Event Correlation Window

Two events are correlated if they fall within a time window $w$:

$$\text{Correlated}(e_1, e_2) \iff |t_{e_1} - t_{e_2}| \leq w$$

Typical correlation windows:

| Analysis Type | Window ($w$) | Rationale |
|:---|:---:|:---|
| Login → file access | 5 seconds | User action latency |
| Malware download → execution | 60 seconds | Browser save + execute |
| Lateral movement | 300 seconds | SSH/RDP connection setup |
| Data staging → exfiltration | 3600 seconds | Compression + transfer |

---

## 5. Memory Forensics — Process Reconstruction

### Virtual Memory Layout (x86_64)

$$\text{Virtual address space} = 2^{48} = 256 \text{ TB (canonical addresses)}$$

| Region | Start | End | Size |
|:---|:---|:---|:---:|
| User space | 0x0000000000000000 | 0x00007FFFFFFFFFFF | 128 TB |
| Kernel space | 0xFFFF800000000000 | 0xFFFFFFFFFFFFFFFF | 128 TB |
| Non-canonical gap | 0x0000800000000000 | 0xFFFF7FFFFFFFFFFF | $2^{64} - 2^{49}$ |

### Page Table Walk

Virtual → Physical address translation (4-level paging):

$$\text{Physical} = \text{PML4}[\text{bits 47:39}] \rightarrow \text{PDPT}[\text{bits 38:30}] \rightarrow \text{PD}[\text{bits 29:21}] \rightarrow \text{PT}[\text{bits 20:12}] + \text{offset}[\text{bits 11:0}]$$

Page size: $2^{12} = 4096$ bytes. Pages in 8 GB RAM: $\frac{8 \times 2^{30}}{4096} = 2{,}097{,}152$ pages.

### Process Memory Analysis

String extraction from memory dump — expected strings per megabyte:

$$\text{ASCII strings} \approx \frac{\text{dump size}}{L_{avg}} \times P(\text{printable run} \geq L_{min})$$

Where $P(\text{printable}) = 95/256 = 0.371$ per byte.

Probability of a run of $k$ printable characters in random data:

$$P(\text{run} \geq k) = 0.371^k$$

| Min Length ($k$) | $P(\text{random run})$ | False Strings per MB |
|:---:|:---:|:---:|
| 4 | 0.019 | ~20,000 |
| 6 | 0.0026 | ~2,700 |
| 8 | 0.00036 | ~370 |
| 10 | 0.000049 | ~51 |

Tools like `strings -n 10` use a minimum length of 10 to reduce noise.

---

## 6. Entropy Analysis — Detecting Encryption and Compression

### Shannon Entropy

$$H(X) = -\sum_{i=0}^{255} p_i \log_2(p_i)$$

Where $p_i$ is the probability of byte value $i$.

| Data Type | Entropy (bits/byte) | Interpretation |
|:---|:---:|:---|
| English text | 3.5-4.5 | Structured, compressible |
| Source code | 4.0-5.0 | Moderate structure |
| Compressed data | 7.5-8.0 | Near-random |
| Encrypted data | 7.99-8.0 | Indistinguishable from random |
| Null/zero-filled | 0.0 | No information |
| Random data | 8.0 | Maximum entropy |

### Encryption Detection

If a disk region has entropy $> 7.9$ bits/byte, it is likely encrypted or compressed:

$$\text{Decision} = \begin{cases} \text{Encrypted/Compressed} & \text{if } H > 7.9 \\ \text{Plaintext} & \text{if } H < 6.0 \\ \text{Ambiguous} & \text{otherwise} \end{cases}$$

### Chi-Square Test for Randomness

$$\chi^2 = \sum_{i=0}^{255} \frac{(O_i - E_i)^2}{E_i}$$

Where $O_i$ = observed count of byte $i$, $E_i = N/256$ = expected count.

For truly random data: $\chi^2 \approx 255$ (degrees of freedom). Encrypted data typically yields $\chi^2 \in [200, 300]$.

---

## 7. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Sector arithmetic | Integer multiplication | Disk image sizing |
| Birthday problem | Combinatorial probability | Hash collision bounds |
| $1/256^k$ | Exponential probability | File carving false positives |
| $\Delta t$ estimation | Statistical average | Clock skew correction |
| Shannon entropy | Information theory | Encryption detection |
| $\chi^2$ test | Statistical hypothesis | Randomness testing |
| Page table walk | Bit-field extraction | Memory forensics |

---

*Digital forensics is evidence-grade mathematics — every hash verification, timeline correlation, and file carving operation must withstand cross-examination in court. The math is the proof.*
