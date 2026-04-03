# The Mathematics of SFTP — SSH File Transfer Protocol Internals

> *SFTP is a file transfer protocol running over SSH. The math covers request pipelining, packet sizing, directory listing performance, and the read-ahead/write-behind optimization.*

---

## 1. SFTP Protocol — Request/Response Model

### The Model

SFTP uses a request-response protocol over an SSH channel. Each operation (read, write, stat) is a separate request.

### Basic Latency

$$T_{operation} = T_{request} + T_{RTT} + T_{response}$$

For a single read:

$$T_{read} = T_{request\_encode} + \text{RTT} + T_{data\_transfer}$$

### Without Pipelining

$$T_{file} = \frac{\text{File Size}}{\text{Block Size}} \times (T_{request} + \text{RTT})$$

### Worked Example

*"10 MiB file, 32 KiB blocks, 50 ms RTT."*

$$\text{Blocks} = \frac{10 \times 1024}{32} = 320$$

$$T_{sequential} = 320 \times 50\text{ms} = 16\text{s}$$

$$\text{Throughput} = \frac{10 \text{ MiB}}{16\text{s}} = 640 \text{ KiB/s} \quad (\text{terrible!})$$

---

## 2. Pipelining — Read-Ahead Optimization

### The Model

SFTP clients pipeline multiple requests without waiting for responses. OpenSSH sends up to 64 outstanding requests.

### Pipelined Throughput

$$T_{pipelined} = \text{RTT} + \frac{\text{File Size}}{\text{Network BW}} + \frac{\text{Blocks} - \text{Pipeline Depth}}{\text{Pipeline Depth}} \times \text{RTT}$$

For large files with sufficient pipeline depth:

$$T_{pipelined} \approx \text{RTT} + \frac{\text{File Size}}{\text{BW}}$$

### Pipeline Depth Impact

$$\text{Optimal Pipeline} = \lceil \frac{\text{BDP}}{\text{Block Size}} \rceil$$

Where BDP = Bandwidth × RTT.

| RTT | BW | BDP | Block Size | Optimal Pipeline |
|:---:|:---:|:---:|:---:|:---:|
| 1 ms | 1 Gbps | 125 KiB | 32 KiB | 4 |
| 50 ms | 100 Mbps | 625 KiB | 32 KiB | 20 |
| 50 ms | 1 Gbps | 6.25 MiB | 32 KiB | 200 |
| 200 ms | 100 Mbps | 2.5 MiB | 32 KiB | 80 |

### Throughput Comparison

*"10 MiB file, 50 ms RTT, 100 Mbps link."*

| Pipeline Depth | Time | Throughput |
|:---:|:---:|:---:|
| 1 (sequential) | 16s | 640 KiB/s |
| 8 | 2.1s | 4.9 MiB/s |
| 32 | 0.85s | 12 MiB/s |
| 64 (OpenSSH default) | 0.82s | 12.2 MiB/s |
| Unlimited | 0.80s | 12.5 MiB/s |

---

## 3. Directory Listing — READDIR Performance

### The Model

SFTP lists directories using READDIR requests, each returning a batch of entries.

### Listing Cost

$$T_{listing} = \lceil \frac{n}{\text{entries per response}} \rceil \times \text{RTT}$$

OpenSSH returns ~100 entries per READDIR response.

| Directory Entries | READDIR Calls | Time (50ms RTT) |
|:---:|:---:|:---:|
| 50 | 1 | 50 ms |
| 500 | 5 | 250 ms |
| 5,000 | 50 | 2.5 sec |
| 50,000 | 500 | 25 sec |
| 500,000 | 5,000 | 250 sec |

### Recursive Listing

$$T_{recursive} = \sum_{\text{dirs}} T_{listing_i} + \sum_{\text{entries}} T_{stat_i}$$

With pipelining for stat calls:

$$T_{recursive} \approx \text{Depth} \times T_{listing} + \frac{\text{Total Entries}}{\text{Pipeline}} \times \text{RTT}$$

---

## 4. Write Performance — Buffered Writes

### The Model

SFTP writes use a write-behind buffer. The client sends write requests without waiting for acknowledgments.

### Write Throughput

$$\text{Throughput}_{write} = \frac{\text{Pipeline Depth} \times \text{Block Size}}{\text{RTT}}$$

$$\text{Max Throughput} = \min(\text{Pipeline BW}, \text{Network BW}, \text{Disk BW})$$

### Block Size Impact

| Block Size | Requests per MiB | Overhead | Throughput (50ms RTT, pipeline=64) |
|:---:|:---:|:---:|:---:|
| 4 KiB | 256 | High | 5 MiB/s |
| 32 KiB | 32 | Medium | 40 MiB/s |
| 64 KiB | 16 | Low | 80 MiB/s |
| 256 KiB | 4 | Minimal | 320 MiB/s |

### SFTP Subsystem Buffer

$$\text{Server-side buffer} = \text{sftpd internal buffer (typically 256 KiB)}$$

---

## 5. Resume and Partial Transfer

### The Model

SFTP supports seeking and partial reads/writes, enabling resume of interrupted transfers.

### Resume Savings

$$\text{Resume Transfer} = \text{File Size} - \text{Already Transferred}$$

$$\text{Resume Overhead} = T_{SSH\_setup} + T_{stat} + T_{seek}$$

### Integrity Verification on Resume

$$T_{verify} = \frac{\text{Transferred Portion}}{\text{Hash Speed}} + \text{RTT}$$

Most SFTP clients verify by file size only (not checksum), risking corruption:

| Verification | Integrity | Time |
|:---|:---:|:---:|
| Size only | Low | O(1) |
| Size + mtime | Medium | O(1) |
| Full checksum | High | O(file size) |

---

## 6. SFTP vs SCP vs rsync

### Feature Comparison

| Feature | SFTP | SCP | rsync/SSH |
|:---|:---:|:---:|:---:|
| Resume | Yes | No | Yes |
| Delta transfer | No | No | Yes |
| Directory listing | Yes | No | Yes |
| Rename/delete remote | Yes | No | Yes (--delete) |
| Interactive browsing | Yes | No | No |
| Pipeline requests | Yes (64) | No (stream) | Partial |
| Protocol overhead | Medium | Low | Medium |

### Throughput Comparison (100 Mbps, 50ms RTT)

| Tool | Single 1 GiB File | 10,000 × 1 KiB Files |
|:---|:---:|:---:|
| SCP | 11.5 MiB/s | 0.1 MiB/s |
| SFTP (pipelined) | 11.5 MiB/s | 2 MiB/s |
| rsync (full) | 11 MiB/s | 1 MiB/s |
| rsync (delta, 5% change) | 0.6 MiB transferred | N/A |
| tar\|ssh | 11.5 MiB/s | 11 MiB/s |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{\text{Size}}{\text{Block}} \times \text{RTT}$ | Linear (sequential) | Non-pipelined time |
| $\text{RTT} + \frac{\text{Size}}{\text{BW}}$ | Addition | Pipelined time |
| $\lceil \frac{\text{BDP}}{\text{Block}} \rceil$ | Ceiling | Optimal pipeline depth |
| $\lceil \frac{n}{100} \rceil \times \text{RTT}$ | Linear | Directory listing time |
| $\frac{\text{Pipeline} \times \text{Block}}{\text{RTT}}$ | Product / division | Write throughput |

---

*Every `sftp`, `sshfs mount`, and `put/get` command uses this request-response protocol — where pipeline depth and block size are the critical tuning parameters that make the difference between 640 KiB/s and 12 MiB/s on the same link.*
