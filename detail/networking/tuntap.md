# The Mathematics of TUN/TAP -- Virtual Interface Throughput and Encapsulation Overhead

> *Every tunnel adds bytes, and every byte costs time. The art of virtual networking is knowing exactly how many bytes you can afford to lose.*

---

## 1. Encapsulation Overhead (MTU Arithmetic)

### The Problem

When packets traverse a TUN-based VPN tunnel, each packet gains encapsulation
headers. Given the physical link MTU and encapsulation protocol, what is the
effective payload MTU inside the tunnel, and what happens when packets exceed it?

### The Formula

The effective tunnel MTU is:

$$MTU_{tunnel} = MTU_{link} - H_{outer\_IP} - H_{transport} - H_{vpn}$$

For common VPN protocols:

$$MTU_{wireguard} = MTU_{link} - 20_{IP} - 8_{UDP} - 32_{WG} = MTU_{link} - 60$$

$$MTU_{openvpn} = MTU_{link} - 20_{IP} - 8_{UDP} - 48_{OVPN} = MTU_{link} - 76$$

$$MTU_{ipsec} = MTU_{link} - 20_{IP} - 8_{UDP/ESP} - 22_{ESP} - P_{pad} = MTU_{link} - 50 - P_{pad}$$

The overhead ratio:

$$\eta = \frac{MTU_{tunnel}}{MTU_{link}} = 1 - \frac{H_{total}}{MTU_{link}}$$

### Worked Examples

**Example 1:** Standard Ethernet link (MTU 1500) with WireGuard tunnel:

$$MTU_{tunnel} = 1500 - 60 = 1440 \text{ bytes}$$

$$\eta = \frac{1440}{1500} = 96.0\%$$

For small packets (100 bytes payload), effective overhead is much larger:

$$\eta_{small} = \frac{100}{100 + 60} = 62.5\%$$

**Example 2:** Nested tunnels (VPN inside VPN for double-hop):

$$MTU_{inner} = (1500 - 60) - 60 = 1380 \text{ bytes}$$

$$\eta_{nested} = \frac{1380}{1500} = 92.0\%$$

## 2. Fragmentation Penalty (Packet Size Distribution)

### The Problem

When tunnel MTU is not set correctly, packets exceeding the tunnel MTU must be
fragmented. What is the throughput penalty of fragmentation as a function of
packet size distribution?

### The Formula

A packet of size $S > MTU_{tunnel}$ is split into $\lceil S / MTU_{tunnel} \rceil$
fragments, each with its own IP header (20 bytes):

$$N_{frags} = \left\lceil \frac{S}{MTU_{tunnel}} \right\rceil$$

$$S_{total} = S + (N_{frags} - 1) \times H_{IP}$$

The fragmentation amplification factor:

$$A_f = \frac{S_{total}}{S} = 1 + \frac{(N_{frags} - 1) \times H_{IP}}{S}$$

For a distribution of packet sizes $P(S)$, the expected amplification:

$$E[A_f] = \int_0^{\infty} A_f(S) \cdot P(S) \, dS$$

### Worked Examples

**Example 1:** A 1500-byte packet enters a tunnel with MTU 1420:

$$N_{frags} = \left\lceil \frac{1500}{1420} \right\rceil = 2$$

$$S_{total} = 1500 + (2 - 1) \times 20 = 1520 \text{ bytes}$$

$$A_f = \frac{1520}{1500} = 1.013 \quad (1.3\% \text{ overhead})$$

**Example 2:** A 4500-byte jumbo frame entering the same tunnel:

$$N_{frags} = \left\lceil \frac{4500}{1420} \right\rceil = 4$$

$$S_{total} = 4500 + (4 - 1) \times 20 = 4560 \text{ bytes}$$

$$A_f = \frac{4560}{4500} = 1.013 \quad (1.3\% \text{ overhead})$$

But reassembly cost is the real penalty: $O(N_{frags})$ memory and CPU.

## 3. Copy Overhead (Kernel-Userspace Transitions)

### The Problem

TUN/TAP devices require packets to cross the kernel-userspace boundary via
`read()` and `write()` system calls. Each crossing involves a memory copy.
What is the throughput ceiling imposed by copy overhead?

### The Formula

Time per packet through TUN device:

$$T_{packet} = T_{syscall} + T_{copy} + T_{process} + T_{syscall} + T_{copy}$$

Where the copy time depends on packet size and memory bandwidth:

$$T_{copy} = \frac{S}{BW_{mem}}$$

Maximum throughput:

$$R_{max} = \frac{1}{2T_{syscall} + 2T_{copy} + T_{process}}$$

With `vhost-net` or `io_uring`, one copy is eliminated:

$$T_{vhost} = T_{syscall} + T_{copy} + T_{process}$$

### Worked Examples

**Example 1:** Standard TUN read/write cycle. $T_{syscall} = 200$ ns,
$S = 1500$ bytes, $BW_{mem} = 30$ GB/s, $T_{process} = 100$ ns:

$$T_{copy} = \frac{1500}{30 \times 10^9} = 50 \text{ ns}$$

$$T_{packet} = 2(200) + 2(50) + 100 = 600 \text{ ns}$$

$$R_{max} = \frac{1}{600 \times 10^{-9}} = 1.67 \text{ Mpps}$$

**Example 2:** With `io_uring` batching 32 packets per syscall:

$$T_{batch} = T_{syscall} + 32 \times (T_{copy} + T_{process}) + T_{syscall}$$

$$T_{per\_pkt} = \frac{400 + 32 \times 150}{32} = \frac{5200}{32} = 162.5 \text{ ns}$$

$$R_{max} = \frac{1}{162.5 \times 10^{-9}} = 6.15 \text{ Mpps}$$

## 4. Queue Theory (TUN Device Buffering)

### The Problem

The TUN device has a transmit queue (`txqueuelen`) that buffers packets between
the kernel and the userspace reader. What is the relationship between queue
length, latency, and packet loss?

### The Formula

Modeling the TUN queue as an M/D/1 queue (Poisson arrivals, deterministic
service, single server):

$$\rho = \frac{\lambda}{\mu}$$

Average queue length:

$$L_q = \frac{\rho^2}{2(1 - \rho)}$$

Average waiting time:

$$W_q = \frac{\rho}{2\mu(1 - \rho)}$$

Packet loss probability when queue is full (size $K$):

$$P_{loss} = \frac{(1 - \rho)\rho^K}{1 - \rho^{K+1}}$$

### Worked Examples

**Example 1:** Arrival rate $\lambda = 100{,}000$ pps, service rate $\mu = 150{,}000$
pps (userspace processing capacity):

$$\rho = \frac{100{,}000}{150{,}000} = 0.667$$

$$L_q = \frac{0.667^2}{2(1 - 0.667)} = \frac{0.444}{0.666} = 0.667 \text{ packets}$$

$$W_q = \frac{0.667}{2 \times 150{,}000 \times 0.333} = 6.67 \text{ } \mu s$$

**Example 2:** Same rates but txqueuelen $K = 500$. Loss probability:

$$P_{loss} = \frac{(1 - 0.667) \times 0.667^{500}}{1 - 0.667^{501}} \approx 0$$

With $\rho < 1$ and $K = 500$, loss is negligible. But at $\rho = 0.99$, $K = 100$:

$$P_{loss} = \frac{0.01 \times 0.99^{100}}{1 - 0.99^{101}} = \frac{0.01 \times 0.366}{1 - 0.363} = \frac{0.00366}{0.637} = 0.57\%$$

## 5. Multi-Queue Scaling (Parallelism Bounds)

### The Problem

Multi-queue TUN/TAP devices distribute packets across file descriptors.
What is the theoretical speedup from $N$ queues, and what limits scaling?

### The Formula

Amdahl's Law for packet processing with parallelizable fraction $p$:

$$S(N) = \frac{1}{(1 - p) + \frac{p}{N}}$$

For TUN/TAP, the serial fraction includes:
- Queue selection (hash computation): $\sim$5%
- Lock contention on shared state: $\sim$5-15%

$$S(N) = \frac{1}{0.1 + \frac{0.9}{N}}$$

### Worked Examples

**Example 1:** 4 queues with 10% serial overhead:

$$S(4) = \frac{1}{0.1 + \frac{0.9}{4}} = \frac{1}{0.1 + 0.225} = \frac{1}{0.325} = 3.08\times$$

**Example 2:** 16 queues:

$$S(16) = \frac{1}{0.1 + \frac{0.9}{16}} = \frac{1}{0.1 + 0.056} = \frac{1}{0.156} = 6.41\times$$

Diminishing returns: 4x more queues (16 vs 4) yields only 2.08x more speedup.

## 6. VPN Throughput Estimation (End-to-End Model)

### The Problem

Given encryption overhead, tunnel encapsulation, and TUN device processing,
what is the maximum achievable VPN throughput on given hardware?

### The Formula

End-to-end throughput:

$$T_{vpn} = \min\left(\frac{BW_{link}}{1 + \frac{H_{encap}}{S_{payload}}}, \; R_{crypto} \times S_{payload}, \; R_{tun} \times S_{payload}\right)$$

Where $R_{crypto}$ is the encryption rate in packets/sec and $R_{tun}$ is the
TUN device throughput in packets/sec.

For AES-256-GCM hardware acceleration:

$$R_{crypto} = \frac{BW_{aesni}}{S_{payload}} \approx \frac{10 \text{ GB/s}}{1400} = 7.14 \text{ Mpps}$$

### Worked Examples

**Example 1:** 10 GbE link, WireGuard, 1420-byte payloads, AES-NI available:

$$T_{link} = \frac{10 \text{ Gbps}}{1 + \frac{60}{1420}} = \frac{10}{1.042} = 9.60 \text{ Gbps}$$

$$T_{crypto} = 7.14 \times 10^6 \times 1420 \times 8 = 81.1 \text{ Gbps}$$

$$T_{tun} = 1.67 \times 10^6 \times 1420 \times 8 = 18.9 \text{ Gbps}$$

$$T_{vpn} = \min(9.60, 81.1, 18.9) = 9.60 \text{ Gbps}$$

The bottleneck is the link bandwidth (encapsulation overhead is small).

**Example 2:** Same setup but 64-byte packets (small-packet VoIP):

$$T_{link} = \frac{10 \text{ Gbps}}{1 + \frac{60}{64}} = \frac{10}{1.938} = 5.16 \text{ Gbps}$$

$$T_{tun} = 1.67 \times 10^6 \times 64 \times 8 = 0.855 \text{ Gbps}$$

$$T_{vpn} = \min(5.16, ..., 0.855) = 0.855 \text{ Gbps}$$

Now the TUN device (syscall overhead) is the bottleneck.

## Prerequisites

- IP networking fundamentals (MTU, fragmentation, IP headers)
- System call mechanics (kernel-userspace transitions, `read`/`write`)
- Queueing theory (M/D/1 queues, Little's Law)
- Amdahl's Law and parallel scaling limits
- Cryptographic performance characteristics (AES-NI throughput)
- Linux kernel networking (netdevice, sk_buff, routing)
- Memory bandwidth concepts (DMA, cache-line effects)
