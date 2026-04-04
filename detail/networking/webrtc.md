# The Mathematics of WebRTC — ICE Prioritization, Bandwidth Estimation, and Jitter Analysis

> *WebRTC hides extraordinary mathematical complexity behind simple browser APIs. From ICE candidate pairing combinatorics to Kalman filter-based bandwidth estimation, the protocol stack solves real-time optimization problems on every call.*

---

## 1. ICE Candidate Priority Calculation (Combinatorics)

### Priority Formula

Each ICE candidate receives a 32-bit priority computed as:

$$\text{priority} = 2^{24} \times \text{type\_pref} + 2^8 \times \text{local\_pref} + (256 - \text{component\_id})$$

Where:
- $\text{type\_pref} \in [0, 126]$ — candidate type preference
- $\text{local\_pref} \in [0, 65535]$ — local interface preference
- $\text{component\_id} \in \{1, 2\}$ — RTP=1, RTCP=2

| Candidate Type | type_pref | Example Priority (component 1) |
|:---|:---:|:---:|
| Host | 126 | 2,130,706,431 |
| Peer-reflexive | 110 | 1,862,270,975 |
| Server-reflexive | 100 | 1,694,498,815 |
| Relay | 0 | 16,777,215 |

### Candidate Pair Priority

The pair priority determines check ordering:

$$P_{pair} = 2^{32} \times \min(G, D) + 2 \times \max(G, D) + \begin{cases} 1 & \text{if } G > D \\ 0 & \text{otherwise} \end{cases}$$

Where $G$ is the controlling agent's candidate priority and $D$ is the controlled agent's.

### Combinatorial Explosion

With $n$ local candidates and $m$ remote candidates, the number of pairs:

$$\text{pairs} = n \times m$$

| Local | Remote | Pairs | STUN Checks |
|:---:|:---:|:---:|:---:|
| 3 | 3 | 9 | 9 |
| 6 | 6 | 36 | 36 |
| 10 | 10 | 100 | 100 |
| 15 | 15 | 225 | 225 |

RFC 8445 limits checks to 100 pairs per foundation to prevent combinatorial explosion on multihomed hosts.

---

## 2. Google Congestion Control (GCC) Algorithm

### Delay-Based Estimation

GCC uses one-way delay gradient to detect congestion. For packet group $i$:

$$d_i = (T_i^{recv} - T_{i-1}^{recv}) - (T_i^{send} - T_{i-1}^{send})$$

Where $d_i$ is the inter-arrival delay variation. Positive $d_i$ suggests growing queues (congestion).

### Kalman Filter for Noise Estimation

GCC applies a Kalman filter to separate network jitter from congestion signal:

$$\hat{m}_i = \hat{m}_{i-1} + K_i(d_i - \hat{m}_{i-1})$$

$$K_i = \frac{P_{i-1} + Q}{P_{i-1} + Q + R}$$

$$P_i = (1 - K_i)(P_{i-1} + Q)$$

Where:
- $\hat{m}_i$ = estimated delay trend
- $K_i$ = Kalman gain
- $P_i$ = estimation error covariance
- $Q$ = process noise variance
- $R$ = measurement noise variance

### Rate Adaptation

The estimated bitrate adjusts based on the delay signal:

$$\hat{A}_i = \begin{cases} \eta \times \hat{A}_{i-1} & \text{if overuse (increase queue)} \\ \alpha \times \hat{A}_{i-1} & \text{if underuse (decrease detected)} \\ \hat{A}_{i-1} & \text{if normal} \end{cases}$$

Where:
- $\eta = 1.05$ (multiplicative increase, 5% per interval)
- $\alpha = 0.85$ (multiplicative decrease, 15% reduction)

This AIMD-like (Additive Increase, Multiplicative Decrease) behavior keeps bitrate near the available capacity without overshooting.

---

## 3. Jitter Buffer Analysis (Statistics)

### Jitter Calculation (RFC 3550)

RTP jitter is the smoothed inter-arrival delay variance:

$$J_i = J_{i-1} + \frac{|D_i| - J_{i-1}}{16}$$

Where $D_i$ is the difference in transit time between consecutive packets:

$$D_i = (R_i - R_{i-1}) - (S_i - S_{i-1})$$

The $\frac{1}{16}$ smoothing factor gives an exponentially weighted moving average with time constant:

$$\tau = -\frac{T_{packet}}{\ln(15/16)} \approx 15.5 \times T_{packet}$$

For 20 ms audio packets: $\tau \approx 310$ ms.

### Adaptive Jitter Buffer Sizing

The buffer target delay balances latency against loss:

$$T_{buffer} = \bar{J} + k \times \sigma_J$$

Where:
- $\bar{J}$ = mean jitter
- $\sigma_J$ = jitter standard deviation
- $k$ = confidence factor (typically 2-4)

| $k$ | Coverage | Buffer Size (10ms mean, 5ms stdev) |
|:---:|:---:|:---:|
| 1 | 68.3% | 15 ms |
| 2 | 95.4% | 20 ms |
| 3 | 99.7% | 25 ms |
| 4 | 99.99% | 30 ms |

---

## 4. SRTP Encryption Overhead

### Packet Size Analysis

SRTP adds authentication and optional encryption overhead to each RTP packet:

$$S_{SRTP} = S_{RTP} + L_{auth} + L_{MKI}$$

Where:
- $S_{RTP}$ = original RTP packet size
- $L_{auth}$ = authentication tag length (default 10 bytes, max 20)
- $L_{MKI}$ = Master Key Identifier (0 or 4 bytes)

### Bandwidth Overhead

For Opus audio at 48 kbps with 20 ms packetization:

$$\text{Payload} = \frac{48000 \times 0.020}{8} = 120 \text{ bytes}$$

| Component | Bytes |
|:---|:---:|
| IP header | 20 |
| UDP header | 8 |
| RTP header | 12 |
| Opus payload | 120 |
| SRTP auth tag | 10 |
| **Total** | **170** |

$$\text{Overhead ratio} = \frac{170 - 120}{120} = 41.7\%$$

$$\text{Wire bitrate} = \frac{170 \times 8}{0.020} = 68 \text{ kbps}$$

---

## 5. Simulcast Bandwidth Optimization

### Layer Selection Problem

An SFU with $N$ participants and 3 simulcast layers must choose which layer to forward to each receiver. Total downstream bandwidth:

$$BW_{down}^{(i)} = \sum_{j \neq i} BW_{layer(j \to i)}$$

Where $layer(j \to i)$ is the simulcast layer selected for sender $j$ to receiver $i$.

### Typical Simulcast Configuration

| Layer | Resolution | Framerate | Bitrate | Ratio to High |
|:---|:---:|:---:|:---:|:---:|
| High | 1280x720 | 30 fps | 2,500 kbps | 1.00 |
| Medium | 640x360 | 30 fps | 500 kbps | 0.20 |
| Low | 320x180 | 15 fps | 150 kbps | 0.06 |

### Bandwidth Savings in Group Calls

Without simulcast (single stream, all receive high):

$$BW_{server}^{no\_sim} = N \times (N-1) \times BW_{high}$$

With simulcast (active speaker gets high, others get low):

$$BW_{server}^{sim} = N \times [BW_{high} + (N-2) \times BW_{low}]$$

$$\text{Savings} = 1 - \frac{BW_{high} + (N-2) \times BW_{low}}{(N-1) \times BW_{high}}$$

| Participants | Without Simulcast | With Simulcast | Savings |
|:---:|:---:|:---:|:---:|
| 4 | 30.0 Mbps | 11.8 Mbps | 60.7% |
| 8 | 140.0 Mbps | 28.4 Mbps | 79.7% |
| 16 | 600.0 Mbps | 73.6 Mbps | 87.7% |
| 25 | 1,500.0 Mbps | 146.0 Mbps | 90.3% |

---

## 6. End-to-End Latency Budget

### Component Breakdown

Total glass-to-glass latency:

$$L_{total} = L_{capture} + L_{encode} + L_{jitter} + L_{network} + L_{decode} + L_{render}$$

| Component | Typical Value | Range |
|:---|:---:|:---:|
| Camera capture | 33 ms (30fps) | 16-66 ms |
| Encoding | 5-15 ms | 1-50 ms |
| Jitter buffer | 20-80 ms | 0-200 ms |
| Network RTT/2 | 25-75 ms | 5-150 ms |
| Decoding | 5-10 ms | 1-30 ms |
| Display render | 8-16 ms | 4-33 ms |
| **Total** | **96-229 ms** | **27-529 ms** |

The ITU-T G.114 recommendation for acceptable one-way delay:

$$L_{total} < 150 \text{ ms (good)}, \quad < 400 \text{ ms (acceptable)}$$

---

*Behind every "it just works" video call lies a cascade of mathematical optimizations — priority functions that search exponential candidate spaces, Kalman filters that separate signal from noise, and bandwidth estimators that balance quality against congestion in real time.*

## Prerequisites

- probability and statistics, linear algebra (Kalman filter), combinatorics, exponential smoothing

## Complexity

- **Beginner:** ICE priority calculation and packet overhead analysis
- **Intermediate:** Jitter buffer sizing and simulcast bandwidth optimization
- **Advanced:** GCC Kalman filter estimation and rate adaptation convergence analysis
