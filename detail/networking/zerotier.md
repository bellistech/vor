# The Mathematics of ZeroTier — Virtual Network Topology and Cryptographic Identity

> *ZeroTier creates software-defined Ethernet networks spanning the internet. The mathematics cover cryptographic addressing from Curve25519 key hashes, planet/moon hierarchical topology as a graph problem, flow rule evaluation cost, and multicast scaling.*

---

## 1. Cryptographic Identity (Hash-Based Addressing)

### The Problem

Every ZeroTier node has a unique 40-bit (10-hex-digit) address derived from its public key. This address must be collision-resistant and unforgeable.

### The Formula

The ZeroTier address is derived from an identity keypair (Curve25519 + Ed25519):

$$\text{Address} = \text{SHA-512}(\text{PublicKey})_{[0:40\text{ bits}]}$$

With a proof-of-work requirement: the first byte of the hash must satisfy:

$$\text{SHA-512}(\text{PublicKey})[0] < T$$

where $T$ is the difficulty threshold, requiring on average:

$$E[\text{attempts}] = \frac{256}{T}$$

### Collision Probability (Birthday Problem)

For $n$ nodes and 40-bit address space ($N = 2^{40} \approx 10^{12}$):

$$P(\text{collision}) \approx 1 - e^{-\frac{n^2}{2N}}$$

$$P(\text{collision}) \approx \frac{n^2}{2 \times 2^{40}}$$

| Active Nodes | Collision Probability |
|:---:|:---:|
| 1,000 | $4.5 \times 10^{-7}$ |
| 1,000,000 | $4.5 \times 10^{-1}$ |
| 10,000,000 | $\approx 1.0$ |

The 40-bit space is sufficient for practical deployments but not for global-scale unique addressing. ZeroTier handles collisions through identity validation.

---

## 2. Planet/Moon Topology (Hierarchical Graph)

### The Problem

ZeroTier uses a two-tier root server hierarchy: planets (global roots) and moons (user-defined roots). This creates a hierarchical overlay network for peer discovery.

### The Formula

The network forms a tree with root servers at the top:

$$G = (V, E), \quad V = V_{\text{planet}} \cup V_{\text{moon}} \cup V_{\text{leaf}}$$

Path length between two leaf nodes (worst case, via root):

$$d_{\text{max}} = 2 \times (\text{depth}_{\text{moon}} + 1) = 2 \times h$$

With moons, average path discovery hops:

$$E[d] = \begin{cases} 1 & \text{if same moon} \\ 2 & \text{if different moons, same planet} \\ 4 & \text{if different planets} \end{cases}$$

### Peer Discovery Time

Root-mediated peer discovery latency:

$$L_{\text{discovery}} = 2 \times L_{\text{root RTT}} + L_{\text{NAT traversal}}$$

With a local moon (RTT = 5 ms) vs planet (RTT = 100 ms):

$$L_{\text{moon}} = 2 \times 5 + 50 = 60 \text{ ms}$$
$$L_{\text{planet}} = 2 \times 100 + 50 = 250 \text{ ms}$$

---

## 3. Network Membership (Set Theory and Access Control)

### The Problem

Each ZeroTier network is identified by a 64-bit network ID (16 hex chars). The network controller decides membership and IP assignment.

### The Formula

Network ID construction:

$$\text{NetworkID} = \text{ControllerAddress}_{40\text{-bit}} \| \text{NetworkNumber}_{24\text{-bit}}$$

This means each controller can manage:

$$N_{\text{networks}} = 2^{24} = 16,777,216 \text{ networks}$$

Member authorization is a set membership test:

$$\text{Authorized}(n, \text{net}) = n \in S_{\text{members}}(\text{net})$$

IP assignment from a pool $[a, b]$:

$$\text{Available IPs} = b - a + 1 - |S_{\text{assigned}}|$$

For a /24 network:

$$\text{Max members} = 2^{8} - 2 = 254$$

---

## 4. Flow Rule Evaluation (Packet Classification)

### The Problem

ZeroTier flow rules are evaluated per-packet as a sequence of match-action pairs. The evaluation cost depends on rule chain length and match complexity.

### The Formula

Linear rule evaluation:

$$C_{\text{eval}} = \sum_{i=1}^{k} C_{\text{match}_i} \times P(\text{reach rule } i)$$

For a rule set with $k$ rules and average early termination at rule $j$:

$$E[C] = \sum_{i=1}^{k} C_i \times \prod_{m=1}^{i-1}(1 - P_{\text{match}_m})$$

### Worked Example

10 rules, each with cost 50 ns, 20% match probability per rule:

$$E[C] = 50 \sum_{i=1}^{10} 0.8^{i-1} = 50 \times \frac{1 - 0.8^{10}}{1 - 0.8} = 50 \times 4.46 = 223 \text{ ns}$$

### Tag-Based Rule Scaling

With $t$ tag dimensions and $v$ values per dimension:

$$\text{Rule combinations} = v^t$$

| Tags | Values/Tag | Rule Space |
|:---:|:---:|:---:|
| 2 | 4 | 16 |
| 3 | 5 | 125 |
| 4 | 10 | 10,000 |

---

## 5. Multicast Propagation (Gossip Protocol)

### The Problem

ZeroTier supports Ethernet multicast/broadcast. Multicast packets must reach all subscribers efficiently without flooding.

### The Formula

Multicast is limited by `multicastLimit` (default 32). For a group of $g$ members:

$$\text{Packets sent} = \min(g, \text{multicastLimit})$$

Bandwidth cost of a multicast frame of size $s$ bytes:

$$B_{\text{multicast}} = s \times \min(g, L) \times (1 + o_{\text{encap}})$$

For ARP (28 bytes payload) with 100 members and limit 32:

$$B_{\text{ARP}} = (28 + 66) \times 32 = 3,008 \text{ bytes}$$

### Multicast Storm Risk

Without limits, broadcast storm bandwidth:

$$B_{\text{storm}} = R_{\text{broadcast}} \times n \times s$$

For 100 nodes each broadcasting 10 packets/sec of 100 bytes:

$$B_{\text{storm}} = 100 \times 10 \times 100 \times 100 = 10 \text{ MB/s total network load}$$

---

## 6. Encryption Overhead (Salsa20/Poly1305)

### The Problem

ZeroTier encrypts all peer-to-peer traffic using Salsa20/12 with Poly1305 authentication.

### The Formula

Salsa20/12 uses 12 rounds (vs ChaCha20's 20 rounds), trading a slight security margin for speed:

$$\text{Security margin}_{\text{Salsa20/12}} = \frac{12}{2 \times r_{\text{best attack}}} = \frac{12}{2 \times 8} = 0.75$$

Throughput:

$$T_{\text{Salsa20/12}} = \frac{f_{\text{CPU}} \times 64}{12 \times C_{\text{round}}}$$

Per-packet overhead:

$$O_{\text{crypto}} = H_{\text{ZT}} + \text{IV}_{24} + \text{Tag}_{16} = 28 + 24 + 16 = 68 \text{ bytes}$$

### Effective Throughput

$$\eta = \frac{\text{MTU} - O_{\text{crypto}}}{\text{MTU}} = \frac{2800 - 68}{2800} = 97.6\%$$

---

## 7. Peer Connection Quality (Latency Modeling)

### The Problem

ZeroTier selects the best path between peers. When direct connections fail, traffic falls back through root servers.

### The Formula

Path selection minimizes weighted latency:

$$\text{path}^* = \arg\min_p \left( \alpha \cdot L_p + \beta \cdot J_p + \gamma \cdot \frac{1}{B_p} \right)$$

Where $L$ is latency, $J$ is jitter, and $B$ is bandwidth.

Direct vs relayed comparison:

$$\text{Direct: } L_d = \frac{\text{distance}}{c \times 0.66} \times 2$$

$$\text{Relayed: } L_r = L_{A \to \text{root}} + L_{\text{root} \to B}$$

### Worked Example

New York to London (5,570 km):

$$L_{\text{direct}} = \frac{5.57 \times 10^6}{2 \times 10^8} \times 2 \times 1000 = 55.7 \text{ ms}$$

Via Frankfurt root (added 1,200 km):

$$L_{\text{relayed}} \approx \frac{(5570 + 1200) \times 2}{2 \times 10^5} \times 1000 = 67.7 \text{ ms}$$

---

## Prerequisites

- elliptic-curve-cryptography, graph-theory, set-theory, networking-fundamentals
