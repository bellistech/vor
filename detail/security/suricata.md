# The Mathematics of Suricata -- Multi-Pattern Matching and Flow Analysis

> *In the stream of packets, the engine must find needles at wire speed -- pattern matching algorithms determine whether the sensor keeps up or drowns.*

---

## 1. Multi-Pattern Matching (Detection Engine)

### The Problem

Suricata must match thousands of rules against every packet simultaneously. A naive approach checking each rule sequentially would scale as O(n * m) where n is the number of rules and m is the payload length, making wire-speed detection impossible on modern networks.

### The Formula

The Aho-Corasick algorithm constructs a finite automaton from all pattern strings. Given a set of patterns P = {p_1, p_2, ..., p_k} with total length L, the automaton is built in O(L) time and searches a text of length m in O(m + z) time, where z is the number of matches:

$$T_{search} = O(m + z), \quad T_{build} = O(L \cdot |\Sigma|)$$

where $|\Sigma|$ is the alphabet size (256 for byte-level matching). The state transition function is:

$$\delta(s, a) = \begin{cases} s' & \text{if } \exists \text{ goto}(s, a) = s' \\ \delta(f(s), a) & \text{otherwise, where } f(s) \text{ is the failure function} \end{cases}$$

The failure function f(s) points to the longest proper suffix of the string represented by state s that is also a prefix of some pattern.

### Worked Examples

**Example 1: Three-pattern automaton**

Patterns: {"he", "she", "his"}. Total pattern length L = 8.

Build time: O(8 * 256) = O(2048) state transitions (worst case).

Searching text "ushers" (m=6): The automaton processes each byte once, finding "she" at position 1 and "he" at position 2. Total operations: 6 + 2 = 8.

**Example 2: Ruleset scaling**

Given 30,000 rules with average content match length of 12 bytes:
- Total pattern length: L = 360,000
- Automaton build: O(360,000) -- done once at startup
- Per-packet search of 1500-byte payload: O(1500 + z)
- At 10 Gbps (~820,000 pps): 820,000 * 1,500 = 1.23 billion bytes/second throughput required

## 2. Flow Tracking and Stream Reassembly (State Machine)

### The Problem

TCP connections must be tracked bidirectionally to apply stateful detection. Suricata maintains a flow table hashing on the 5-tuple, and must handle millions of concurrent flows without hash collisions degrading performance.

### The Formula

Flow hash table sizing follows the birthday problem. For n concurrent flows in a table of size m buckets, the expected number of collisions is:

$$E[\text{collisions}] = n - m + m \cdot \left(1 - \frac{1}{m}\right)^n \approx n - m + m \cdot e^{-n/m}$$

The load factor is defined as:

$$\alpha = \frac{n}{m}$$

For chained hashing with load factor alpha, average lookup cost is:

$$T_{lookup} = 1 + \frac{\alpha}{2}$$

The flow hash function for a 5-tuple (src_ip, dst_ip, src_port, dst_port, proto) uses:

$$h = \text{hashword}(src\_ip \oplus dst\_ip, src\_port | (dst\_port \ll 16), proto) \mod m$$

### Worked Examples

**Example 1: Flow table sizing**

Network with 500,000 concurrent flows, hash table size m = 65,536:
- Load factor: alpha = 500,000 / 65,536 = 7.63
- Average lookup: 1 + 7.63/2 = 4.82 chain traversals
- This is too high; increasing to m = 1,048,576: alpha = 0.48, lookup = 1.24

**Example 2: Memory estimation**

Each flow entry consumes approximately 600 bytes. For 1 million concurrent flows:
- Flow table memory: 1,000,000 * 600 = 600 MB
- Stream reassembly buffers (2 * depth per flow): 2 * 1 MB * active_streams
- If 10% of flows are actively reassembled: 100,000 * 2 MB = 200 GB (too high)
- With depth = 32 KB: 100,000 * 64 KB = 6.4 GB (manageable)

## 3. Packet Loss and Capture Performance (Queueing Theory)

### The Problem

When the processing rate falls below the arrival rate, packets are dropped. Modeling Suricata as a queueing system helps predict when the sensor will begin losing visibility.

### The Formula

Using an M/D/1 queue model (Poisson arrivals, deterministic service time), the probability of the ring buffer being full given arrival rate lambda, service rate mu, and buffer size K:

$$\rho = \frac{\lambda}{\mu}, \quad P_{loss} = \frac{(1-\rho)\rho^K}{1-\rho^{K+1}}$$

The average time a packet spends in the system:

$$W = \frac{1}{\mu} + \frac{\rho}{2\mu(1-\rho)}$$

For multi-threaded processing with c worker threads (M/D/c queue), the utilization per thread becomes:

$$\rho_c = \frac{\lambda}{c \cdot \mu}$$

### Worked Examples

**Example 1: Single-thread capacity**

Service rate mu = 200,000 pps per worker. Arrival rate lambda = 180,000 pps.
- Utilization: rho = 180,000 / 200,000 = 0.9
- Ring buffer size K = 200,000 packets
- P_loss = (0.1 * 0.9^200000) / (1 - 0.9^200001) approximately 0 (buffer large enough)
- Average system time: W = 5 us + (0.9 / (2 * 200,000 * 0.1)) = 5 us + 22.5 us = 27.5 us

**Example 2: Thread scaling for 10 Gbps**

At 10 Gbps with average 750-byte packets: lambda = 1,666,666 pps.
With mu = 250,000 pps per worker:
- Minimum threads: ceil(1,666,666 / 250,000) = 7 workers
- With 8 workers: rho_c = 1,666,666 / (8 * 250,000) = 0.833
- Safe margin at rho_c = 0.7 requires: ceil(1,666,666 / (0.7 * 250,000)) = 10 workers

## 4. Anomaly Scoring and Alert Correlation (Statistical Detection)

### The Problem

Beyond signature matching, Suricata supports threshold-based detection. Determining optimal thresholds requires modeling the baseline distribution of network events to minimize false positives while maintaining detection sensitivity.

### The Formula

For a rate-based detection rule with threshold count c in time window t, given a baseline event rate of lambda_0 events per second (Poisson distributed), the false positive rate is:

$$P_{FP} = P(X \geq c) = 1 - \sum_{k=0}^{c-1} \frac{(\lambda_0 \cdot t)^k \cdot e^{-\lambda_0 \cdot t}}{k!}$$

The detection rate against an attacker generating lambda_a additional events:

$$P_{TP} = 1 - \sum_{k=0}^{c-1} \frac{((\lambda_0 + \lambda_a) \cdot t)^k \cdot e^{-(\lambda_0 + \lambda_a) \cdot t}}{k!}$$

The Receiver Operating Characteristic (ROC) trade-off:

$$\text{AUC} = \int_0^1 P_{TP}(c(P_{FP})) \, dP_{FP}$$

### Worked Examples

**Example 1: SSH brute force threshold**

Baseline SSH attempts: lambda_0 = 0.5/min per source IP. Window t = 60s, threshold c = 5.
- Expected baseline count: lambda_0 * t = 0.5
- P_FP = 1 - sum(Poisson(0.5, k) for k=0..4) = 1 - 0.99985 = 0.00015
- One false positive per ~6,667 source IPs per minute

Attacker rate: lambda_a = 2/min (total 2.5/min, expected count = 2.5):
- P_TP = 1 - sum(Poisson(2.5, k) for k=0..4) = 1 - 0.8912 = 0.1088
- Only 10.9% detection rate; lower threshold to c = 3: P_TP = 0.4562, P_FP = 0.0014

## 5. JA3 Fingerprinting Entropy (Cryptographic Identification)

### The Problem

Encrypted traffic prevents payload inspection, but the TLS Client Hello message contains observable parameters (cipher suites, extensions, elliptic curves) that form a fingerprint. JA3 hashes these parameters into an MD5 digest. Understanding the entropy of JA3 fingerprints determines their discriminating power for identifying applications and malware.

### The Formula

A JA3 fingerprint is composed of 5 fields: TLS version, cipher suites, extensions, elliptic curves, and elliptic curve point formats. The fingerprint space cardinality:

$$|F| = |V| \times \binom{|C|}{k_c} \times \binom{|E|}{k_e} \times \binom{|G|}{k_g} \times \binom{|P|}{k_p}$$

where $|C|$ is the set of available cipher suites, $k_c$ is the number selected, and similarly for extensions (E), curves (G), and point formats (P). The entropy of the fingerprint distribution across observed traffic:

$$H_{JA3} = -\sum_{f \in F_{\text{obs}}} p_f \log_2 p_f$$

The collision probability between two random clients:

$$P_{\text{collision}} = \sum_{f} p_f^2$$

A low collision probability indicates high discriminating power.

### Worked Examples

**Example 1: Enterprise fingerprint diversity**

Observed 50 unique JA3 hashes across 1 million TLS sessions. Top 5 cover 92% of traffic:
- Chrome: 45%, Firefox: 25%, Edge: 12%, Python requests: 7%, curl: 3%
- H_JA3 = -(0.45*log2(0.45) + 0.25*log2(0.25) + ... ) = 2.8 bits
- P_collision = 0.45^2 + 0.25^2 + 0.12^2 + 0.07^2 + 0.03^2 + sum(remaining) = 0.283

A rare JA3 hash (p < 0.001) has surprisal > 10 bits and warrants investigation.

**Example 2: Malware C2 detection**

Known malware JA3: `a0e9f5d64349fb13191bc781f81f42e1`. Appears in 0.002% of traffic.
- Surprisal: -log2(0.00002) = 15.6 bits (highly anomalous)
- False positive rate if flagging this hash: depends on JA3 collision with legitimate software
- If 3 legitimate applications share this hash: P(malware | hash match) = P(malware) * 1 / (P(malware) + P(legit))
- With prior P(malware) = 0.001: posterior = 0.001 / (0.001 + 0.00006) = 0.943

## Prerequisites

- Probability theory (Poisson distribution, queueing models)
- Automata theory (finite state machines, string matching algorithms)
- Hash functions and hash table analysis
- TCP/IP protocol stack fundamentals
- Statistical hypothesis testing (Type I/II errors)
- Big-O computational complexity analysis
- Information theory (entropy, surprisal, collision probability)
- Cryptography fundamentals (TLS handshake, cipher suites)
- Linux networking stack (AF_PACKET, NFQUEUE, kernel bypass)
