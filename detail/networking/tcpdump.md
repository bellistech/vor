# tcpdump Deep Dive — Theory & Internals

> *tcpdump captures packets using the kernel's BPF (Berkeley Packet Filter) subsystem. Understanding its internals means understanding BPF bytecode compilation, capture ring buffer sizing, timestamp precision, and the filter expression algebra that determines which packets survive the kernel-to-userspace path.*

---

## 1. BPF Filter — Compilation and Complexity

### How Filters Work

tcpdump compiles filter expressions into BPF bytecode that runs in the kernel:

$$\text{Expression} \xrightarrow{compile} \text{BPF bytecode} \xrightarrow{kernel} \text{per-packet evaluation}$$

### Filter Evaluation Cost

BPF is a simple register machine. Each filter instruction: ~1-5 ns.

$$T_{filter} = N_{instructions} \times T_{instruction}$$

| Filter Expression | BPF Instructions | Cost per Packet |
|:---|:---:|:---:|
| (none — capture all) | 1 | ~1 ns |
| `host 10.0.0.1` | ~8 | ~10 ns |
| `tcp port 80` | ~12 | ~15 ns |
| `tcp port 80 and host 10.0.0.1` | ~20 | ~25 ns |
| Complex filter (10 terms) | ~60 | ~75 ns |

At 10 Mpps (10 Gbps small packets), a 75 ns filter consumes: $10 \times 10^6 \times 75 \times 10^{-9} = 0.75$ seconds of CPU per second = 75% of one core.

### Filter Algebra

BPF filters form a Boolean algebra:

$$\text{host A and port 80} = (\text{src}=A \lor \text{dst}=A) \land (\text{sport}=80 \lor \text{dport}=80)$$

Expanded: 4 conditions checked per packet.

### Kernel vs Userspace Filtering

| Location | Packets Processed | CPU Cost |
|:---|:---|:---|
| Kernel BPF | All packets on interface | $O(1)$ per packet |
| Userspace (no filter) | All packets copied to userspace | Massive memory/CPU |
| Userspace (with filter) | Only matching packets copied | Minimal |

**Rule:** Always filter in the kernel. Capturing without a filter on a busy interface can drop packets.

---

## 2. Capture Buffer Sizing

### Ring Buffer Model

tcpdump uses a memory-mapped ring buffer:

$$B_{total} = N_{slots} \times S_{slot}$$

Default buffer: typically 2 MB. With `-B` flag, can be increased.

### Drop Probability

When the buffer is full and userspace can't keep up:

$$P_{drop} = P(R_{arrival} > R_{drain})$$

$$R_{drain} = \frac{B_{total}}{T_{process}}$$

### Sizing for Zero-Drop Capture

$$B_{required} = R_{peak} \times T_{burst} \times S_{avg}$$

| Peak Rate | Burst Duration | Avg Packet | Buffer Needed |
|:---:|:---:|:---:|:---:|
| 10,000 pps | 1 sec | 500 B | 5 MB |
| 100,000 pps | 1 sec | 500 B | 50 MB |
| 1,000,000 pps | 0.5 sec | 200 B | 100 MB |

### Snap Length (`-s`)

$$S_{captured} = \min(S_{packet}, \text{snaplen})$$

Default snaplen: 262,144 bytes (captures full packet). Setting `-s 96` captures only headers:

$$\text{Buffer efficiency} = \frac{S_{snaplen}}{S_{full}} = \frac{96}{1500} = 6.4\%$$

With 96-byte snap: a 2 MB buffer holds ~21,000 packets instead of ~1,400.

---

## 3. Timestamp Precision

### Timestamp Sources

| Source | Resolution | Accuracy | Flag |
|:---|:---:|:---:|:---:|
| Kernel (default) | ~1 us | ~10 us | (default) |
| Adapter hardware | ~10 ns | ~100 ns | `--time-stamp-precision=nano` |
| NTP-synchronized | ~1 us | ~1 ms | (system clock) |
| PTP-synchronized | ~10 ns | ~100 ns | (system clock + PTP) |

### Inter-Packet Timing Analysis

$$\Delta t_{i} = ts_{i+1} - ts_i$$

$$\text{Jitter} = \text{StDev}(\Delta t_i)$$

$$\text{Wire rate} = \frac{S_{packet} \times 8}{\Delta t}$$

| Packet Size | Delta | Implied Rate |
|:---:|:---:|:---:|
| 1,500 B | 12 us | 1 Gbps |
| 1,500 B | 1.2 us | 10 Gbps |
| 64 B | 0.672 us | 762 Mbps |

---

## 4. Capture File Sizing

### PCAP File Size

$$S_{file} = H_{global} + \sum_{i=1}^{N} (H_{packet} + S_{captured,i})$$

Where $H_{global} = 24$ bytes (pcap header), $H_{packet} = 16$ bytes per packet header.

### File Size Estimation

$$S_{file} \approx N \times (16 + \min(S_{avg}, \text{snaplen})) + 24$$

| Packets | Avg Size | Snap=Full | Snap=96 |
|:---:|:---:|:---:|:---:|
| 10,000 | 500 B | 5 MB | 1.1 MB |
| 100,000 | 500 B | 50 MB | 11 MB |
| 1,000,000 | 500 B | 500 MB | 110 MB |
| 10,000,000 | 500 B | 5 GB | 1.1 GB |

### Rotation (`-C` and `-W`)

With `-C 100 -W 10`: 10 files of 100 MB each = 1 GB maximum disk usage.

$$S_{max} = C_{size} \times W_{count}$$

---

## 5. Protocol Decoding Depth

### Dissection Layers

$$\text{Decode depth} = L2 + L3 + L4 + L7$$

| Flag | Verbosity | Typical Output per Packet |
|:---|:---|:---:|
| (none) | Summary | ~100 characters |
| `-v` | Verbose | ~200 characters |
| `-vv` | More verbose | ~500 characters |
| `-vvv` | Maximum | ~1,000 characters |
| `-X` | Hex + ASCII dump | ~500+ characters |
| `-XX` | Full hex dump with link header | ~600+ characters |

### Output Rate

$$R_{output} = R_{packets} \times S_{output\_per\_packet}$$

At 10,000 pps with `-vv`: $10,000 \times 500 = 5$ MB/s of text output. Writing to terminal at this rate causes drops — always write to file (`-w`).

---

## 6. Common Filter Patterns — Boolean Logic

### Expression Grammar

$$E = \text{primitive} \;|\; E \;\text{and}\; E \;|\; E \;\text{or}\; E \;|\; \text{not}\; E$$

### Efficiency of Compound Filters

Short-circuit evaluation:

$$\text{and}: \text{if first is false, skip second}$$

$$\text{or}: \text{if first is true, skip second}$$

**Optimization:** Put the most selective (most likely to reject) term first in AND chains:

$$\text{Efficient: } \text{port 443 and host 10.0.0.1}$$

If only 5% of packets are port 443, 95% of packets are eliminated at the first check.

### Byte-Offset Filters

For protocol fields not in the built-in grammar:

$$\text{tcp[13] \& 0x12 = 0x12} \quad \text{(SYN+ACK flags set)}$$

This indexes into the TCP header at byte offset 13 (flags) and applies a bitmask.

---

## 7. Performance Limits

### Maximum Capture Rate

| Method | Max PPS (single core) | Notes |
|:---|:---:|:---|
| tcpdump (libpcap) | ~200-500K pps | Kernel → userspace copy |
| tcpdump + ring buffer | ~500K-1M pps | mmap reduces copies |
| PF_RING | ~5-10M pps | Zero-copy kernel module |
| DPDK capture | ~15M+ pps | Kernel bypass |
| Hardware timestamping | Line rate | NIC-level capture |

### CPU per Packet

$$T_{per\_packet} = T_{BPF} + T_{copy} + T_{timestamp} + T_{write}$$

$$\approx 10 + 50 + 5 + 100 = 165 \text{ ns (to file)}$$

$$\approx 10 + 50 + 5 + 500 = 565 \text{ ns (to terminal)}$$

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $N_{instr} \times T_{instr}$ | Product | BPF filter cost |
| $R_{peak} \times T_{burst} \times S_{avg}$ | Product | Buffer sizing |
| $\min(S_{packet}, \text{snaplen})$ | Minimum | Captured bytes |
| $N \times (16 + S_{snap}) + 24$ | Summation | File size |
| $C_{size} \times W_{count}$ | Product | Rotation disk usage |
| $R_{packets} \times S_{output}$ | Product | Output data rate |

---

*tcpdump is the ground truth of network debugging — it shows you exactly what's on the wire, bit by bit. The BPF filter engine is so fundamental that it evolved into eBPF, which now powers everything from firewalls to observability to container networking.*
