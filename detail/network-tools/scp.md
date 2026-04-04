# The Mathematics of SCP — Secure Copy Protocol Internals

> *SCP copies files over SSH. The math covers SSH channel overhead, transfer throughput, encryption costs, and comparison with modern alternatives.*

---

## 1. SCP Protocol — Sequential Transfer Model

### The Model

SCP operates over an SSH channel, transferring files sequentially with minimal protocol overhead.

### Transfer Formula

$$T_{scp} = T_{ssh\_setup} + \sum_{i=1}^{n} (T_{metadata_i} + T_{transfer_i})$$

$$T_{transfer_i} = \frac{\text{File Size}_i}{\text{Effective Bandwidth}}$$

### SSH Setup Cost

$$T_{ssh} = T_{DNS} + T_{TCP} + T_{key\_exchange} + T_{auth}$$

| Component | Typical Time | Notes |
|:---|:---:|:---|
| DNS | 1-50 ms | Cached vs uncached |
| TCP handshake | 1 × RTT | Three-way handshake |
| Key exchange | 10-100 ms | Diffie-Hellman |
| Authentication | 10-500 ms | Password vs key |
| **Total** | **22-700 ms** | One-time per connection |

### Per-File Overhead

$$\text{Per-File Header} = \text{C} + \text{mode} + \text{size} + \text{filename} + \text{newline} \approx 30 \text{ bytes}$$

$$\text{Per-File Confirmation} = 1 \text{ byte (0x00 = OK)}$$

---

## 2. Encryption Overhead

### The Model

All SCP data passes through SSH encryption. The cipher choice affects throughput.

### Cipher Throughput

$$\text{Effective BW} = \min(\text{Network BW}, \text{Cipher Throughput}, \text{Disk I/O})$$

| Cipher | Throughput (AES-NI) | Without AES-NI |
|:---|:---:|:---:|
| aes128-ctr | ~6 GiB/s | ~200 MiB/s |
| aes256-ctr | ~4 GiB/s | ~150 MiB/s |
| aes128-gcm | ~8 GiB/s | ~300 MiB/s |
| chacha20-poly1305 | ~1.5 GiB/s | ~500 MiB/s |

### When Encryption is the Bottleneck

$$\text{Bottleneck} = \text{Encryption} \iff \text{Network BW} > \text{Cipher Throughput}$$

For a 10 Gbps link without AES-NI:

$$\text{Max SCP throughput} \approx 200 \text{ MiB/s} \quad (\text{1.6 Gbps — only 16\% of link})$$

### SSH Overhead per Packet

$$\text{SSH Overhead} = \text{Packet Length (4)} + \text{Padding Length (1)} + \text{Padding (4-255)} + \text{MAC (16-32)}$$

$$\text{Min Overhead} = 4 + 1 + 4 + 16 = 25 \text{ bytes per packet}$$

$$\text{Overhead \%} = \frac{25}{\text{Payload Size} + 25} \times 100$$

| Payload (MTU) | Overhead | Overhead % |
|:---:|:---:|:---:|
| 1,400 bytes | 25 bytes | 1.8% |
| 32 KiB | 25 bytes | 0.08% |
| 64 KiB | 25 bytes | 0.04% |

---

## 3. Transfer Performance — SCP vs Alternatives

### Throughput Comparison

$$\text{SCP Throughput} = \min(\text{Network}, \text{Cipher}, \text{Disk})$$

$$\text{rsync Throughput} = \min(\text{Network}, \text{Cipher}, \text{Disk}) - \text{Checksum Overhead}$$

$$\text{rsync Delta} = \text{Changed Data Only (potentially much less)}$$

### When to Use What

| Scenario | SCP | rsync | sftp |
|:---|:---:|:---:|:---:|
| Single large file | Good | Good | Good |
| Many small files | Slow (sequential) | Better (pipeline) | Better |
| Incremental sync | Bad (full copy) | Best (delta) | Bad |
| Resume interrupted | No | Yes | Yes |
| Bandwidth limit | No built-in | `--bwlimit` | No |
| Directory recursion | `-r` | `-r` (better) | Recursive |

### Many Small Files Problem

$$T_{scp\_many} = n \times (T_{overhead} + \frac{\text{Avg Size}}{\text{BW}})$$

$$T_{tar\_pipe} = T_{ssh} + \frac{\sum \text{Sizes}}{\text{BW}} \quad (\text{much faster for many files})$$

| Files | Avg Size | SCP Time (LAN) | tar\|ssh Time |
|:---:|:---:|:---:|:---:|
| 100 | 1 KiB | 2s | 0.1s |
| 10,000 | 1 KiB | 200s | 1s |
| 100,000 | 1 KiB | 2000s | 10s |

---

## 4. Window Size and Latency Impact

### The Model

SSH uses a channel window for flow control. Window size limits throughput over high-latency links.

### Bandwidth-Delay Product

$$\text{BDP} = \text{Bandwidth} \times \text{RTT}$$

$$\text{Max Throughput} = \frac{\text{Window Size}}{\text{RTT}}$$

### Worked Examples

| Link | BW | RTT | BDP | Default Window (2 MiB) | Achieved |
|:---|:---:|:---:|:---:|:---:|:---:|
| LAN | 1 Gbps | 0.5 ms | 62.5 KiB | 2 MiB > BDP | 1 Gbps |
| WAN | 1 Gbps | 50 ms | 6.25 MiB | 2 MiB < BDP | 320 Mbps |
| Intercontinental | 1 Gbps | 200 ms | 25 MiB | 2 MiB < BDP | 80 Mbps |

**SCP over high-latency links is often limited by SSH window size, not network bandwidth.**

### HPN-SSH (High Performance)

HPN patches increase the default window to match BDP:

$$\text{Optimal Window} = \text{BW} \times \text{RTT} \times 1.5$$

---

## 5. SCP Protocol Deprecation

### The Model

OpenSSH has deprecated the SCP protocol (since 8.0) in favor of SFTP internally.

### Protocol Differences

| Feature | SCP (legacy) | SCP (SFTP backend) | sftp |
|:---|:---:|:---:|:---:|
| Protocol | RCP over SSH | SFTP over SSH | SFTP |
| Resume | No | No | Yes |
| Glob on remote | Shell expansion | SFTP glob | SFTP glob |
| Filename escaping | Shell-dependent | Safe | Safe |
| Performance | Good | Slightly slower | Good |

---

## 6. Compression Impact

### SSH Compression (-C flag)

$$\text{Compressed Transfer} = \frac{\text{Data}}{\text{Compression Ratio}}$$

$$\text{Worth it if:} \quad \frac{T_{compress} + T_{transfer\_compressed}}{1} < T_{transfer\_raw}$$

| Data Type | Ratio | 100 Mbps Link | 1 Gbps Link |
|:---|:---:|:---:|:---:|
| Text/logs | 3-5x | Helpful | Not worth it |
| Binary | 1.5-2x | Marginal | Not worth it |
| Already compressed | 1.0x | Harmful (CPU waste) | Harmful |

$$\text{Break-even BW} \approx 50-100 \text{ Mbps (for compressible data)}$$

Above this bandwidth, compression adds latency without benefit.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{\text{Size}}{\min(\text{Net}, \text{Cipher}, \text{Disk})}$ | Rate equation | Transfer time |
| $\text{BW} \times \text{RTT}$ | Product | Bandwidth-delay product |
| $\frac{\text{Window}}{\text{RTT}}$ | Division | Max throughput |
| $n \times T_{overhead}$ | Linear | Small file penalty |
| $\frac{25}{\text{Payload} + 25}$ | Ratio | SSH packet overhead |
| $\frac{\text{Data}}{\text{Ratio}}$ | Division | Compression benefit |

---

*Every `scp file user@host:path` opens an SSH channel and streams file data through encrypted pipes — a protocol so simple it's being replaced, but so ubiquitous it remains the first tool most sysadmins reach for.*

## Prerequisites

- SSH channel and transport layer fundamentals
- Symmetric encryption overhead (AES-GCM, ChaCha20)
- TCP throughput and window scaling

## Complexity

- **Beginner:** Copy files to/from remote hosts, recursive copy (-r), port selection (-P)
- **Intermediate:** Compression (-C), cipher selection (-c), bandwidth limiting (-l), ProxyJump usage
- **Advanced:** SSH channel overhead calculations, encryption throughput bottlenecks, SCP vs SFTP protocol comparison, deprecation path to sftp-based transfer
