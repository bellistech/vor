# The Mathematics of ZIP — Random-Access Archive Internals

> *ZIP is an archive format with per-file compression and a central directory for random access. The math covers DEFLATE per-entry, central directory structure, ZIP64 extensions, and encryption overhead.*

---

## 1. ZIP Structure — Three-Part Layout

### The Model

A ZIP file has three sections:

$$\text{ZIP File} = \text{Local File Headers + Data} + \text{Central Directory} + \text{End of Central Directory}$$

### Local File Header (30 + variable bytes)

| Field | Size | Purpose |
|:---|:---:|:---|
| Signature | 4 | 0x04034b50 |
| Version needed | 2 | Minimum ZIP version |
| Flags | 2 | Encryption, data descriptor |
| Compression method | 2 | 0=store, 8=DEFLATE |
| Last mod time/date | 4 | DOS timestamp |
| CRC-32 | 4 | Data checksum |
| Compressed size | 4 | Compressed data size |
| Uncompressed size | 4 | Original data size |
| Filename length | 2 | Filename string length |
| Extra field length | 2 | Extension data length |

### Random Access via Central Directory

The central directory at the end of the file acts as an index:

$$T_{find\_file} = T_{read\_central\_dir} + T_{seek\_to\_offset}$$

$$T_{read\_central\_dir} = \frac{\text{CD Size}}{\text{Read Speed}}$$

### Central Directory Size

$$\text{CD Entry Size} = 46 + \text{Filename Length} + \text{Extra Field} + \text{Comment}$$

$$\text{CD Total} = \sum_{i=1}^{n} \text{CD Entry}_i$$

| Files | Avg Filename | CD Entry | CD Total | CD as % of 1 GiB |
|:---:|:---:|:---:|:---:|:---:|
| 100 | 30 bytes | 76 bytes | 7.4 KiB | 0.0007% |
| 10,000 | 40 bytes | 86 bytes | 840 KiB | 0.08% |
| 100,000 | 50 bytes | 96 bytes | 9.2 MiB | 0.9% |
| 1,000,000 | 60 bytes | 106 bytes | 101 MiB | 9.9% |

---

## 2. Per-File Compression — Method Selection

### The Model

ZIP compresses each file independently, allowing mixed compression methods.

### Compression Methods

| Method ID | Name | Typical Ratio | Speed |
|:---:|:---|:---:|:---:|
| 0 | Stored (no compression) | 1.0x | Maximum |
| 8 | DEFLATE | 2-5x | Good |
| 12 | BZIP2 | 3-6x | Slow |
| 14 | LZMA | 4-8x | Very slow |
| 93 | Zstandard | 3-6x | Fast |
| 95 | XZ | 4-8x | Very slow |

### Per-File vs Solid Compression

$$\text{ZIP (per-file):} \quad \text{Each file compressed independently}$$

$$\text{Solid (7z/tar.gz):} \quad \text{All files compressed as one stream}$$

| Archive Type | Cross-File Redundancy | Random Access | Ratio |
|:---|:---:|:---:|:---:|
| ZIP | Not exploited | Yes | 3x |
| tar.gz (solid) | Exploited | No | 4-5x |
| 7z (solid) | Exploited | Partial | 5-8x |

**ZIP sacrifices ~20-40% compression ratio for random access capability.**

---

## 3. ZIP64 Extensions — Breaking the 4 GiB Barrier

### Original ZIP Limits

$$\text{Max File Size} = 2^{32} - 1 = 4,294,967,295 \text{ bytes} \approx 4 \text{ GiB}$$

$$\text{Max Archive Size} = 2^{32} - 1 \approx 4 \text{ GiB}$$

$$\text{Max Files} = 2^{16} - 1 = 65,535$$

### ZIP64 Limits

$$\text{Max File Size} = 2^{64} - 1 \approx 16 \text{ EiB}$$

$$\text{Max Archive Size} = 2^{64} - 1 \approx 16 \text{ EiB}$$

$$\text{Max Files} = 2^{64} - 1$$

### ZIP64 Extra Field Overhead

Each ZIP64 entry adds 28 bytes to the extra field:

$$\text{ZIP64 Overhead per File} = 28 \text{ bytes (local)} + 28 \text{ bytes (CD)}$$

| Files | ZIP64 Overhead |
|:---:|:---:|
| 1,000 | 54.7 KiB |
| 100,000 | 5.3 MiB |
| 1,000,000 | 53.4 MiB |

---

## 4. CRC-32 Checksum

### The Model

ZIP uses CRC-32 for data integrity verification.

$$\text{CRC-32} = \text{Polynomial division of data by } x^{32} + x^{26} + \ldots + x + 1$$

### Properties

$$\text{CRC-32 Output} = 32 \text{ bits} = 4 \text{ bytes}$$

$$P(\text{undetected error}) = \frac{1}{2^{32}} = 2.33 \times 10^{-10}$$

$$\text{Detects:} \text{ all single-bit errors, all double-bit errors, all odd-bit errors, all burst errors} \leq 32 \text{ bits}$$

### CRC Computation Speed

$$\text{CRC-32 Throughput} \approx 5-20 \text{ GiB/s (hardware CRC32C)} \quad | \quad 500 \text{ MiB/s (software)}$$

CRC overhead is negligible compared to compression/decompression.

---

## 5. Encryption — ZipCrypto vs AES

### ZipCrypto (Legacy, Weak)

$$\text{Key Derivation:} \text{ 96-bit state from password, no salt}$$

$$\text{Known-Plaintext Attack:} \quad O(2^{38}) \text{ operations (broken)}$$

### AES-256 (Strong)

$$\text{Key: } 256 \text{ bits from PBKDF2(password, salt, iterations)}$$

$$\text{Salt} = 16 \text{ bytes per file}$$

$$\text{Overhead per File} = \text{Salt (16)} + \text{Password Verifier (2)} + \text{Auth Code (10)} = 28 \text{ bytes}$$

### Encryption Performance Impact

| Method | Encrypt Speed | Decrypt Speed | Overhead |
|:---|:---:|:---:|:---:|
| None | N/A | N/A | 0 bytes |
| ZipCrypto | ~1 GiB/s | ~1 GiB/s | 12 bytes/file |
| AES-128 | ~3 GiB/s (AES-NI) | ~3 GiB/s | 20 bytes/file |
| AES-256 | ~2.5 GiB/s (AES-NI) | ~2.5 GiB/s | 28 bytes/file |

---

## 6. Multi-Part (Split) Archives

### Spanning Formula

$$\text{Parts} = \lceil \frac{\text{Archive Size}}{\text{Part Size}} \rceil$$

$$\text{Part Overhead} = 4 \text{ bytes (signature per part)}$$

| Archive Size | Part Size | Parts |
|:---:|:---:|:---:|
| 4.7 GiB | 4.7 GiB (DVD) | 1 |
| 10 GiB | 4.7 GiB (DVD) | 3 |
| 50 GiB | 25 GiB (BD) | 2 |
| 100 GiB | 2 GiB (FAT32 limit) | 50 |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $46 + \text{name} + \text{extra}$ | Addition | CD entry size |
| $2^{32} - 1$ | Exponential constant | ZIP32 size limit |
| $\frac{1}{2^{32}}$ | Probability | CRC collision |
| $\lceil \frac{\text{Size}}{\text{Part}} \rceil$ | Ceiling | Split archive parts |
| Per-file DEFLATE | Independent | Compression per entry |
| $T_{CD} + T_{seek}$ | Additive | Random access time |

---

*Every `zip`, `unzip -l`, and `zipinfo` command reads this structure — a 1989 format (Phil Katz, PKZIP) designed for floppy disk distribution that became the universal container for random-access compressed archives.*

## Prerequisites

- DEFLATE compression (same as gzip)
- Central directory concept (archive metadata index)
- Cross-platform file attribute handling

## Complexity

- **Beginner:** Create/extract zip archives, recursive directory zipping
- **Intermediate:** Password protection, split archives, exclude patterns, update/freshen
- **Advanced:** Central directory vs local header structure, ZIP64 extensions, random-access seeking, ZipCrypto weaknesses
