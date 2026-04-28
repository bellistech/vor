# Queue Management — Deep Dive

> *Math-heavy companion to `ramp-up/queue-management-eli5`. Every formula, every controller,
> every constant from RFC 7567 / 8033 / 8289 / 8290 / 8311 / 9330 / 9332, plus the seminal
> Floyd & Jacobson 1993 RED paper and Nichols & Jacobson 2012 CoDel paper. Optimised for the
> in-terminal reader who needs the number, the recurrence, and the worked example without
> leaving the shell.*

---

## 0. Stack View and Notation

```
+----------------------------------------------------+
|  Application                                       |  bytes per second offered
+----------------------------------------------------+
|  TCP / QUIC / UDP                                  |  congestion control reacts to drops + ECN
+----------------------------------------------------+
|  IP                                                |
+----------------------------------------------------+
|  Network Device Queue (qdisc)                      |  AQM lives HERE — pfifo_fast, fq_codel, cake
+----------------------------------------------------+
|  Driver TX ring (DMA buffer)                       |  bufferbloat hides here too (BQL helps)
+----------------------------------------------------+
|  PHY / wire                                        |  fixed link rate
+----------------------------------------------------+
```

Notation used throughout:

| Symbol  | Meaning                                                                     |
|:--------|:----------------------------------------------------------------------------|
| `λ`     | mean arrival rate (packets/s or bytes/s)                                    |
| `μ`     | mean service rate (packets/s or bytes/s)                                    |
| `ρ`     | utilization, `ρ = λ/μ`, dimensionless, valid only for `ρ < 1`               |
| `L`     | mean number in system (queue + in-service)                                  |
| `L_q`   | mean number in queue (excluding the one in service)                         |
| `W`     | mean sojourn time (queue wait + service time)                               |
| `W_q`   | mean queue waiting time (excludes service)                                  |
| `σ`     | standard deviation                                                          |
| `B`     | buffer capacity (bytes)                                                     |
| `r`     | shaper rate (bytes/s)                                                       |
| `MTU`   | max transmission unit, default 1500 B (Ethernet, no jumbo)                  |
| `t_*`   | timestamps; sojourn time `s = t_dequeue − t_enqueue`                        |
| `avg`   | exponentially-weighted moving average of queue length (RED)                 |
| `min_th`, `max_th` | RED thresholds on `avg`                                          |
| `max_p` | maximum drop probability at `avg = max_th` (RED)                            |
| `w_q`   | RED EWMA weight, typically `1/512` to `1/2048`                              |
| `target`| CoDel target sojourn time, default 5 ms                                     |
| `interval` | CoDel sliding-window interval, default 100 ms                            |
| `count` | CoDel drop counter, used in `t_next = t_prev + interval/sqrt(count)`        |
| `α, β`  | PIE PI controller gains (`α` = integral, `β` = proportional)                |
| `ECT(0)`| ECN-Capable Transport, original (RFC 3168)                                  |
| `ECT(1)`| ECN-Capable Transport, scalable / L4S identifier (RFC 9330)                 |
| `CE`    | Congestion Experienced codepoint                                            |

---

## 1. Queue Theory Primer

Networking AQM is applied queueing theory under adversarial workloads. You don't need a PhD,
but you need to know **why ρ→1 is a cliff**, not a slope.

### 1.1 The M/M/1 Model

Kendall notation `M/M/1`: Poisson arrivals (memoryless), exponential service times,
1 server, infinite buffer, FCFS.

For arrival rate `λ` (packets/s) and service rate `μ` (packets/s) with `ρ = λ/μ`:

```
  Mean number in system:        L     = ρ / (1 − ρ)
  Mean queue length:            L_q   = ρ² / (1 − ρ)
  Mean sojourn time:            W     = 1 / (μ − λ)
  Mean waiting time:            W_q   = ρ / (μ − λ)
  Probability of n in system:   P_n   = (1 − ρ) · ρ^n
  Probability the queue is empty: P_0 = 1 − ρ
```

**Key intuition:** delay is `1/(μ−λ)`. Subtract the rates *before* you invert. If `μ` and
`λ` are close, the denominator is tiny, and `W` blows up. The whole AQM field exists because
real network traffic spends a non-trivial fraction of its time near `ρ ≈ 0.95`.

### 1.2 Little's Law (Universal)

```
  L = λ · W                  (system-wide)
  L_q = λ · W_q              (queue only)
```

Holds for **any** stable queueing system regardless of arrival/service distribution. If you
know two of the three, you know the third. It is the single most useful equation in networking
operations:

| Given                                  | Solve for                  |
|:---------------------------------------|:---------------------------|
| 100 Mbps link, 1500 B packets, queue 200 KB | latency at saturation: `W = L/λ = 200_000 / 12_500_000 = 16 ms` |
| 10 packets average in queue, 8333 pps  | `W = 10/8333 = 1.2 ms`     |
| Want 5 ms latency, link is 100 Mbps    | `L_q = λ · W_q = 12.5e6 · 5e-3 = 62.5 KB` max queue depth |

### 1.3 Utilization Cliff

Plot `W vs ρ`:

```
  W
  |                                                       *
  |                                                     *
  |                                                  *
  |                                              *
  |                                          *
  |                                     *
  |                              *
  |                       *
  |              *  *
  |   *  *  *
  +---|----|----|----|----|----|----|----|----|----|---->  ρ
   0.0 0.1  0.2  0.3  0.4  0.5  0.6  0.7  0.8  0.9  1.0
```

For M/M/1, doubling the offered load from `ρ=0.5` to `ρ=0.95` moves you from `W = 2/μ` to
`W = 20/μ` — a **10×** latency increase for a `1.9×` load increase.

### 1.4 Power-Law Variance

Standard deviation of sojourn time for M/M/1:

```
  σ_W = 1 / (μ − λ)
  σ_W = W                            (variance equals mean for memoryless service)

  Var(L) = ρ / (1 − ρ)²              ≈ scales with 1/(1−ρ)²
  σ(delay) ∝ ρ²/(1−ρ)²               (variance of delay grows with ρ²)
```

In English: as the link saturates, jitter grows even faster than mean delay. This is why a
"barely overloaded" link feels broken — your mean RTT moved from 5 ms to 50 ms, but the
**99th percentile** moved from 10 ms to 500 ms.

### 1.5 M/G/1 (General Service Time) — Pollaczek–Khinchine

When service times are not exponential (real packets aren't), use Pollaczek–Khinchine:

```
  W_q = (λ · E[S²]) / (2 · (1 − ρ))

  with C_S = σ_S / E[S] (coefficient of variation of service time):

  W_q = (ρ / (1 − ρ)) · ((1 + C_S²) / 2) · E[S]
```

Bursty service (large `C_S²`) is *quadratically* worse than smooth service. This is why
shapers (token buckets) reduce latency variance even when they don't change mean throughput:
they reduce `C_S²`.

### 1.6 The Throughput–Latency Tradeoff

```
  Throughput := λ · (1 − P_drop)              (goodput at the egress)
  Latency    := W = L / λ
```

A larger buffer *raises* `λ` you can push through (fewer drops) but *raises* `L`, so latency
grows. The whole point of AQM is to **break this monotone tradeoff** by dropping smarter, not
deeper.

---

## 2. Drop Algorithms — Foundations

### 2.1 Tail-Drop (FIFO with Hard Limit)

The default forever. Enqueue is:

```c
if (qlen + pkt.len > qlim) {
    drop(pkt);             // tail-drop: reject the new arrival
    return;
}
enqueue(pkt);
```

**Pros:** simplest possible, O(1), no state per flow.
**Cons:**
- **Lock-out:** a single greedy flow can fill the buffer and starve everyone.
- **Global synchronization:** when buffer fills, *every* TCP flow sees a drop in the same
  RTT, all back off in lockstep, link goes idle, all ramp back up together — sawtooth on the
  utilization plot.
- **Bufferbloat:** if `qlim` is large, latency at saturation = `qlim / r`. A 256 KB buffer
  on a 1 Mbps DSL link injects **2 seconds** of delay before any drop happens.

### 2.2 Head-Drop

When full, drop the packet at the **head** of the queue rather than the new arrival:

```c
if (qlen + pkt.len > qlim) {
    drop(dequeue());       // head-drop: throw out the oldest
}
enqueue(pkt);
```

**Theory:** delivers fresher data, breaks some lock-step (loss is signaled to the sender of
the *oldest* packet, which is staler), and a TCP head-of-line drop triggers fast retransmit
sooner than a tail drop.
**Practice:** rarely used standalone; mostly seen inside CoDel where head-drop is the
mechanism by which the sojourn-time controller acts.

### 2.3 Random-Drop (Uniform)

When the buffer is full, drop a *random* packet from the queue (uniform over occupants):

```c
if (qlen + pkt.len > qlim) {
    int victim = rand() % qlen;
    drop(packet_at(victim));
}
enqueue(pkt);
```

Breaks lockstep more thoroughly than tail-drop because flows are signaled in proportion to
their queue occupancy. This is the seed idea behind RED: instead of waiting for the buffer
to overflow, drop *probabilistically as it fills*.

### 2.4 Drop-from-Front-of-Flow (DFOF)

A refinement used in flow-aware AQMs (CAKE, FQ-CoDel): when a per-flow sub-queue must shed,
drop the *front* of that flow's queue. Rationale: TCP's fast retransmit triggers off the
*next* ACK after the gap, so dropping the head of a flow signals the loss one RTT sooner
than dropping the tail.

---

## 3. RED — Random Early Detection (Floyd & Jacobson, 1993)

The 1993 paper "Random Early Detection Gateways for Congestion Avoidance" is the founding
document of AQM. The whole idea: **drop probabilistically before the queue is full, and
weight the probability by recent average occupancy** so that bursts don't trigger drops but
sustained pressure does.

### 3.1 The EWMA on Queue Length

RED computes a low-pass-filtered queue length, not the instantaneous one. Let `q(n)` be the
queue length sample on enqueue `n`. The EWMA:

```
  avg(n) = (1 − w_q) · avg(n−1) + w_q · q(n)
```

Equivalently, in the "idle period" when the queue is empty for `m` packet times:

```
  avg ← avg · (1 − w_q)^m
```

Typical `w_q` choices:

| `w_q`    | Time constant (in samples) | Behavior                                  |
|:---------|:---------------------------|:------------------------------------------|
| `1/128`  | ~128                       | Aggressive, tracks instantaneous quickly  |
| `1/512`  | ~512                       | Balanced default                          |
| `1/1024` | ~1024                      | Smooths heavy bursts, slower to react     |
| `1/2048` | ~2048                      | Almost never reacts to single bursts      |

The sample interval is "per enqueue" not "per second", so `w_q` must be tuned for the link
rate.

### 3.2 The Drop Probability Ramp

Three parameters define the ramp on `avg`:

```
  if      avg < min_th:        p = 0
  else if avg < max_th:        p_b = max_p · (avg − min_th) / (max_th − min_th)
  else                          p   = 1                  (force-drop region)
```

The "marking" probability `p_b` is then **adjusted for inter-drop spacing** so that drops
arrive in a Bernoulli (uniform) process rather than bursts:

```
  p = p_b / (1 − count · p_b)            count = packets since last drop
```

`count` increments on every enqueue and resets to 0 on every drop. This adjustment makes
the inter-drop gap roughly geometric rather than Poisson-clumpy, which empirically reduces
TCP global synchronization further.

### 3.3 Visualizing the RED Ramp

```
  drop p
  1.0 |                                            *********
      |                                            *
  max_p                                  *  *  *  *
      |                              *
      |                          *
      |                      *
      |                  *
      |              *
  0.0 |  ************
      +--+-----+--------+--------+--------+--------+----->  avg
         0  min_th             max_th     2·max_th
```

### 3.4 Parameter Tuning (Floyd, 1997)

The "right" RED params are notoriously fiddly. Floyd's later guidance:

```
  min_th  = 5 · MTU / link_rate · BW         # ~5 packets at the link rate
  max_th  = 3 · min_th                       # 3× headroom (gentle slope)
  max_p   = 0.1                              # 10 % at max_th
  w_q     = 1 − exp(−1 / (link_rate / MTU))  # ~50 ms equivalent time constant
```

**Worked example, 100 Mbps, 1500 B MTU:**

```
  link_rate    = 100_000_000 b/s = 12_500_000 B/s = 8333 packets/s @ 1500 B
  pkt_time     = 1 / 8333 = 120 µs/pkt
  buffer_BW    = 100 ms · 12.5 MB/s = 1.25 MB ≈ 833 packets

  min_th       = 5 · 1500 / 100e6 · 100e6 = 5 packets        # raw paper formula
                ≈ 0.6 % of buffer

  Floyd-2-3 rule of thumb:
  min_th       = 0.5 · BDP        ≈ 416 packets    # for a 100 ms RTT path
  max_th       = 1.5 · min_th     ≈ 624 packets
  max_p        = 0.10
  w_q          = 0.002            (≈ 1/512)
```

### 3.5 GENTLE-RED Variant

Standard RED jumps from `p = max_p` straight to `p = 1` at `avg = max_th`. Real traffic hates
this discontinuity — a microburst over `max_th` triggers a flood of drops. GENTLE-RED smooths
the ramp:

```
  if      avg < min_th:                p = 0
  else if avg < max_th:                p = max_p · (avg − min_th) / (max_th − min_th)
  else if avg < 2·max_th:              p = max_p + (1 − max_p) · (avg − max_th) / max_th
  else                                  p = 1
```

The drop curve becomes a piecewise-linear bowtie up to `2·max_th`, with continuity at the
joining point. Almost every modern RED implementation defaults to GENTLE.

### 3.6 RED Limitations

- **Param-sensitive:** `min_th, max_th, max_p, w_q` interact non-trivially with link rate
  and RTT. A correctly-tuned RED on a 1 Mbps DSL is wildly mistuned at 100 Mbps.
- **Class-blind:** all flows get the same treatment, even VoIP that should have priority.
- **Can't handle burst floors:** a steady microburst pattern can keep `avg` permanently above
  `min_th` even though the queue is mostly empty between bursts.

These motivate WRED (per-class), then ARED (auto-tune), then CoDel (no-tune).

---

## 4. WRED — Weighted RED

WRED runs **multiple RED state machines in parallel**, one per traffic class, and picks the
appropriate parameter set based on a packet classifier (typically DSCP or IP precedence).

### 4.1 Per-Class Parameter Tables

```
  Class      DSCP        min_th   max_th   max_p   w_q
  EF (voice) 46 (101110)  20       40      0.02   1/512
  AF41       34 (100010)  60      120      0.05   1/512
  AF31       26 (011010)  80      160      0.10   1/512
  AF21       18 (010010) 100      200      0.10   1/512
  CS0/BE     0  (000000) 120      240      0.10   1/512
  Scavenger  8  (001000)  40       80      0.20   1/512   # drop early
```

Lower-priority classes get **lower thresholds** and **higher max_p** so they shed first.
EF (Expedited Forwarding) gets the highest thresholds and the lowest max_p so it shed last.

### 4.2 Cisco WRED Profile (Reference)

```
                        +--------+
  packet ──── classify ─►│  DSCP  ├─► look up profile  ─► RED state per class
                        +--------+
                                  │
                                  └─► common avg, but per-class min_th/max_th/max_p
```

In Cisco IOS:

```cisco
random-detect dscp-based
random-detect dscp 46 40 60 100        # EF: min 40, max 60, mark-prob denom 100 (=0.01)
random-detect dscp 34 60 120 50        # AF41
random-detect dscp 0  120 240 10       # default
```

The third number `denom` is `1/max_p`, so `denom = 100` ⇒ `max_p = 0.01`.

### 4.3 ARED — Adaptive RED

WRED still requires per-class tuning. ARED (Floyd, Gummadi, Shenker, 2001) closes the loop:

```
  every 0.5 s:
      if avg > target_q:  max_p ← max_p · α    (α > 1, e.g. 1.10)
      if avg < target_q:  max_p ← max_p · β    (β < 1, e.g. 0.90)
      clamp max_p ∈ [0.01, 0.50]
```

`target_q = (min_th + max_th) / 2`. ARED is the bridge from "static profile" to modern
controller-based AQM — but CoDel and PIE went further by removing the queue-length notion
entirely.

---

## 5. CoDel — Controlled Delay (Nichols & Jacobson, 2012)

CoDel ("co-del") is a controller that targets **sojourn time, not queue length**. Buffers
are sized by RAM, not by network engineering — so any algorithm that targets queue length is
implicitly tuning to the wrong metric. CoDel keys on the time a packet spent in the queue.

### 5.1 The Two Constants

```
  target   = 5 ms        # acceptable standing queue
  interval = 100 ms      # the worst-case good RTT for the Internet
```

`target` is "the latency budget you'll accept under load."
`interval` is "the slack we give a flow's congestion controller to react before we drop again."

**These are universally good defaults** — Nichols & Jacobson explicitly designed CoDel to
**not require tuning per link**, and the LEDBAT and L4S working groups have validated this
across rates from 64 kbps to 100 Gbps.

### 5.2 Sojourn Time

Each packet is timestamped on enqueue. On dequeue:

```
  s = now − pkt.enqueue_ts        # sojourn time, in seconds
```

A second timer, `first_above_time`, is the timestamp at which `s` *first* exceeded `target`
in the current "above" episode:

```
  on dequeue:
      s = now − pkt.enqueue_ts
      if s < target:
          first_above_time = 0    # reset
      else if first_above_time == 0:
          first_above_time = now + interval
      else if now ≥ first_above_time:
          ok_to_drop = true       # been above target for ≥ interval
```

The 100 ms grace period before the first drop is what makes CoDel **transparent to short
RTT bursts** — a 50 ms RTT TCP slow-start spike never triggers a drop.

### 5.3 The Drop Schedule

Once dropping is enabled, drops are scheduled in **decreasing time intervals**:

```
  t_next = t_prev + interval / sqrt(count)

  count = number of drops in this dropping episode
```

Concretely:

| `count` | `interval / sqrt(count)` (with interval=100 ms) |
|:--------|:------------------------------------------------|
| 1       | 100.0 ms                                        |
| 2       |  70.7 ms                                        |
| 3       |  57.7 ms                                        |
| 4       |  50.0 ms                                        |
| 9       |  33.3 ms                                        |
| 16      |  25.0 ms                                        |
| 25      |  20.0 ms                                        |
| 100     |  10.0 ms                                        |

Why `1/sqrt(count)`? Because TCP's congestion window response to a drop is roughly
`Δcwnd ≈ −cwnd/2`, and the steady-state throughput obeys
`bandwidth ∝ 1 / sqrt(p)` (the Mathis equation). So dropping at rate
`∝ sqrt(count)` produces a *linear* reduction in offered load — the controller is in fact
a closed-loop linear controller on TCP throughput.

### 5.4 The State Machine

```
  ┌──────────────────────────────────────────────────────────┐
  │   not_dropping                                          │
  │                                                         │
  │   on dequeue, if s ≥ target for ≥ interval:             │
  │      drop(pkt); count++; if count was 0: set t_next     │
  │      transition → dropping                              │
  └─────────────────────────┬────────────────────────────────┘
                            │
                            ▼
  ┌──────────────────────────────────────────────────────────┐
  │   dropping                                              │
  │                                                         │
  │   on dequeue:                                           │
  │      if s < target:                                     │
  │         transition → not_dropping (reset count, t_next) │
  │      else if now ≥ t_next:                              │
  │         drop(pkt)                                       │
  │         count++                                         │
  │         t_next = t_prev + interval / sqrt(count)        │
  └──────────────────────────────────────────────────────────┘
```

### 5.5 ECN-Aware CoDel

If a packet's IP header has `ECT(0)` or `ECT(1)`, CoDel **marks** (sets `CE`) instead of
dropping. Same controller; cheaper signal.

### 5.6 CoDel Pseudocode

```c
struct codel_state {
    u64  first_above_time;   // 0 = below target
    u64  drop_next;          // schedule for next drop
    u32  count;              // drops in this episode
    bool dropping;
};

bool codel_should_drop(pkt, state, now) {
    u64 s = now - pkt->enqueue_ts;
    if (s < TARGET) {
        state->first_above_time = 0;
        return false;
    }
    if (state->first_above_time == 0) {
        state->first_above_time = now + INTERVAL;
        return false;
    }
    return now >= state->first_above_time;
}

pkt_t *codel_dequeue(state, now) {
    pkt_t *p = queue_dequeue();
    if (!p) {
        state->dropping = false;
        return NULL;
    }

    bool drop = codel_should_drop(p, state, now);

    if (state->dropping) {
        if (!drop) {
            state->dropping = false;
        } else if (now >= state->drop_next) {
            drop_pkt(p);
            state->count++;
            state->drop_next = state->drop_next +
                               INTERVAL / int_sqrt(state->count);
            p = queue_dequeue();
        }
    } else if (drop) {
        drop_pkt(p);
        state->dropping = true;
        // first drop in episode: re-derive count from time gap
        if (now - state->drop_next < 16 * INTERVAL && state->count > 2)
            state->count -= 2;       // hysteresis
        else
            state->count = 1;
        state->drop_next = now + INTERVAL / int_sqrt(state->count);
        p = queue_dequeue();
    }
    return p;
}
```

The `int_sqrt` is a fast Newton-iteration integer sqrt — Linux's `codel_inv_sqrt_cache[]`
table memoizes it for the common range.

---

## 6. FQ-CoDel — Fair Queueing + CoDel (RFC 8290)

CoDel solves the "what to drop" problem; FQ-CoDel solves the "whom to drop from" problem.
Standardized as RFC 8290 in 2018; the default `qdisc` in Linux 5.13+ when systemd-networkd
manages a link.

### 6.1 The 1024 Sub-Queue Hash

Each enqueued packet is hashed by 5-tuple `(src_ip, dst_ip, src_port, dst_port, proto)` into
one of 1024 (default) sub-queues:

```
  hash = jhash(src_ip ⊕ dst_ip, src_port ⊕ dst_port, proto, perturb)
  bucket = hash mod 1024
```

`perturb` is rotated every 600 s to defeat hash-flooding attacks. A flow lives in exactly one
bucket; many flows can share a bucket (collision).

### 6.2 Birthday-Paradox Collision Math

For `N` flows uniformly hashed into `K` buckets, the expected number of collisions is:

```
  E[colliding flows] = N − K · (1 − (1 − 1/K)^N)
                    ≈ N − K · (1 − e^(−N/K))      (large K)
```

For `K = 1024`:

| `N` flows | Expected collisions | P(no collision) |
|:----------|:--------------------|:----------------|
| 32        | 0.49                | 0.61            |
| 64        | 1.96                | 0.13            |
| 128       | 7.69                | 0.0030          |
| 256       | 29.4                | ~10⁻¹⁵          |
| 1024      | 376                 | 0               |

The collision rate is benign: when two flows share a bucket, they share that bucket's
**DRR quantum**, so each gets half the deficit allocation. They aren't *combined* — CoDel
still discriminates by sojourn time per packet.

### 6.3 Two-List Scheduling: New vs Old

FQ-CoDel maintains two ordered lists of *active* sub-queues:

- `new_list` — sub-queues that have just become active (no packets dequeued from them yet
  in this round).
- `old_list` — sub-queues that have already been visited at least once.

**On dequeue:** prefer `new_list` first. Within a list, the scheduler is **DRR (Deficit Round
Robin)**.

This bias is the magic ingredient: short, sparse flows (DNS, SYN, ACK, VoIP) hit `new_list`
on every burst, jump the queue, get instant service. Bulk flows (TCP downloads) sit on
`old_list` and share the link round-robin among themselves.

### 6.4 DRR (Deficit Round Robin) — The Math

Each sub-queue carries an integer `deficit` counter. The scheduler:

```
  on round-robin pass over active sub-queues:
      bucket.deficit += quantum         # quantum = 1514 (Ethernet MTU + L2 header)
      while bucket has packets and bucket.deficit ≥ pkt.size:
          dequeue_and_send(pkt)
          bucket.deficit -= pkt.size
```

`quantum = 1514` (the actual frame size with 14 B Ethernet header) means: **on each round, a
sub-queue gets exactly one MTU's worth of credit**. So `N` active flows each see throughput
`r/N` (where `r` is the link rate) plus or minus one MTU — provably fair.

DRR's key property: it is **work-conserving** (idle flows don't waste credit) and **O(1) per
packet** (no priority queue or list-search needed).

### 6.5 Per-Flow CoDel State

Each of the 1024 sub-queues has its own `codel_state` (target = 5 ms, interval = 100 ms).
A bulk flow that builds a queue in its own bucket gets dropped from *its own* sub-queue
without affecting the other 1023.

### 6.6 FQ-CoDel Defaults

```
  flows         = 1024
  quantum       = 1514 (≈ MTU)
  target        = 5 ms
  interval      = 100 ms
  memory_limit  = 32 MB
  ecn           = enabled
  ce_threshold  = disabled (FQ-CoDel; enabled for L4S variant)
```

### 6.7 Linux Reference

```bash
# Show
tc qdisc show dev eth0

# Replace pfifo_fast with fq_codel
sudo tc qdisc replace dev eth0 root fq_codel

# Tune
sudo tc qdisc replace dev eth0 root fq_codel \
    target 5ms interval 100ms quantum 1514 \
    flows 1024 memory_limit 32Mb ecn

# Stats
tc -s qdisc show dev eth0
# look for "drop_overlimit", "new_flow_count", "ecn_mark", "drop_overmemory"
```

### 6.8 Why It Works So Well

A single-flow benchmark (iperf3 −P 1) shows almost no behavior change vs CoDel — but real
traffic isn't a single flow. The instant a second flow appears, FQ-CoDel halves its share
of the link without measurable startup delay. Bufferbloat in mixed-flow real-world conditions
is essentially eliminated.

Tested as default in Linux 5.13 (June 2021), OpenWrt 19.07, FreeBSD 13. Commonly recommended
as the "no-knobs win" for any router that doesn't need shaping.

---

## 7. CAKE — Common Applications Kept Enhanced

CAKE (Common Applications Kept Enhanced) is FQ-CoDel + token-bucket shaper + diffserv
classifier + flow-isolation modes, all in one qdisc. Standardized informally; widely
deployed in OpenWrt's "SQM" (Smart Queue Management) since 2017.

### 7.1 The 8-Tier Diffserv Mapping (`diffserv4` default)

| Tier  | Name        | DSCP examples              | Bandwidth share |
|:------|:------------|:---------------------------|:----------------|
| 0     | Bulk        | CS1, LE                    |  6.25 %         |
| 1     | Best-Effort | CS0, default               | 31.25 %         |
| 2     | Video       | AF4x, EF (in `diffserv8`)  | 50.00 %         |
| 3     | Voice       | EF, VA, CS5–CS7            | 12.50 %         |

In `diffserv8` mode there are 8 tiers with finer separation. Each tier is itself a full
FQ-CoDel instance with **shared deficit** but separate target/interval.

### 7.2 Flow-Isolation Modes

CAKE exposes four flow-isolation modes via `flowblind`/`srchost`/`dsthost`/`hosts`/`flows`/
`dual-srchost`/`dual-dsthost`/`triple-isolate`:

| Mode             | Hash key                                        | Use case                          |
|:-----------------|:------------------------------------------------|:----------------------------------|
| `flowblind`      | none — single FIFO                              | testing                           |
| `srchost`        | source IP                                       | server uplink (per client)        |
| `dsthost`        | destination IP                                  | server downlink                   |
| `hosts`          | both endpoints                                  | symmetric                         |
| `flows`          | 5-tuple                                         | classic FQ-CoDel                  |
| `dual-srchost`   | source IP, then 5-tuple inside                  | LAN gateway upstream              |
| `dual-dsthost`   | destination IP, then 5-tuple inside             | LAN gateway downstream            |
| `triple-isolate` | source, destination, 5-tuple                    | symmetric LAN gateway             |

Triple-isolate is the OpenWrt SQM default for residential gateways: prevents a single host
from monopolizing bandwidth even if it opens 1000 sockets, because the per-host outer hash
caps that host's deficit.

### 7.3 Rate Shaper Math

CAKE's bandwidth shaper is a single token-bucket integrated into the dequeue path:

```
  on dequeue at time t:
      tokens += rate · (t − last_t)
      tokens = min(tokens, burst)
      if pkt.len ≤ tokens:
          send(pkt)
          tokens -= pkt.len
      else:
          schedule_dequeue(t + (pkt.len − tokens)/rate)
```

`burst` defaults to 1 ms × `rate`. On a 100 Mbps shaper that's 12.5 KB — small enough that
the shaper imposes a smooth rate, large enough that a single MTU never causes scheduling
jitter.

### 7.4 ATM / DOCSIS Overhead Compensation

CAKE knows the underlying L2 is sometimes not Ethernet:

```
  per-packet overhead = MTU + ATM_cell_padding   # 53 B cells on ADSL
                      = MTU + 18                 # DOCSIS upstream
                      = MTU + 14                 # vanilla Ethernet
                      = MTU + 8                  # PPPoE
```

Setting `overhead 18 atm` (or `noatm` / `ptm` for VDSL) makes the shaper account for L1/L2
framing accurately. Without it, you can be 10 % off on a DSL link — and 10 % overshoot at
the bottleneck *is* bufferbloat.

### 7.5 Linux Reference

```bash
# Residential gateway, 100 Mbps down, 20 Mbps up
sudo tc qdisc replace dev eth0 root cake \
    bandwidth 100Mbit \
    diffserv4 \
    triple-isolate \
    nat \
    overhead 18 \
    docsis ack-filter

sudo tc qdisc replace dev wlan0 root cake \
    bandwidth 20Mbit \
    diffserv4 \
    triple-isolate \
    overhead 18 \
    docsis
```

`nat` enables conntrack lookup so that NATed flows hash on the *internal* IP, not the public
one. `ack-filter` collapses redundant ACKs in the upstream direction (a real win on
asymmetric DSL).

---

## 8. PIE — Proportional Integral controller Enhanced (RFC 8033)

PIE (RFC 8033) is the AQM that DOCSIS 3.1 cable modems and 3GPP cellular base stations were
required to ship with. It targets sojourn time like CoDel, but with a smoother PI controller
instead of CoDel's `1/sqrt(count)` step function.

### 8.1 Two Sojourn Estimates

PIE doesn't timestamp every packet (CPU expensive at line rate). It estimates sojourn time
from queue length and departure rate:

```
  τ_now = qlen / departure_rate

  departure_rate = total_bytes_dequeued / measurement_interval
```

Updated every `T_update = 15 ms` typically.

### 8.2 The PI Controller

The drop probability is updated each `T_update`:

```
  p_drop ← p_drop  +  α · (τ_now − τ_target)  +  β · (τ_now − τ_prev)

  α = 0.125 / s   (integral gain)
  β = 1.25  / s   (proportional gain)

  clamp  p_drop ∈ [0, 1]
```

- `α (τ_now − τ_target)` — integral term. As long as we're above target, `p` ramps up.
- `β (τ_now − τ_prev)` — proportional term. Reacts to the *trend* — if delay is rising fast,
  ramp `p` faster.

The relative magnitudes (`β = 10·α`) reflect that the proportional response is a "shock
absorber" — useful in transients but with no static bias.

### 8.3 Auto-Tuning by Drop Probability

The gains scale with the current `p_drop`:

```
  if p_drop < 0.000001:    α ← α/2048;  β ← β/2048
  if p_drop < 0.00001:     α ← α/512;   β ← β/512
  if p_drop < 0.0001:      α ← α/128;   β ← β/128
  if p_drop < 0.001:       α ← α/32;    β ← β/32
  if p_drop < 0.01:        α ← α/8;     β ← β/8
  if p_drop < 0.1:         α ← α/2;     β ← β/2
```

This is a logarithmic gain schedule: at very low drop probability, the controller moves
slowly; at high drop probability, it moves fast. Prevents oscillation when load is light.

### 8.4 Burst Allowance

PIE has a built-in "do not drop for the first 150 ms after queue filled" allowance:

```
  burst_allowance = 150 ms   on first instance of qlen > min_pkt_threshold
  decremented by T_update each interval
  while burst_allowance > 0:  no drops, ECN marks only
```

This is what makes PIE friendly to short-RTT bursty traffic — a 100 ms RTT TCP flow's slow
start never triggers a drop.

### 8.5 Drop Decision

```c
if (random() < p_drop) {
    if (pkt.is_ect()) mark_ce(pkt);
    else              drop(pkt);
}
```

### 8.6 PIE vs CoDel

| Property                 | CoDel                                     | PIE                                       |
|:-------------------------|:------------------------------------------|:------------------------------------------|
| Target metric            | sojourn time                              | sojourn time                              |
| Sojourn measurement      | per-packet timestamp                      | qlen / departure_rate estimate            |
| Drop schedule            | `interval/sqrt(count)` step function      | smooth PI controller                      |
| Tuning required          | none                                      | `α, β, T_update, τ_target`                |
| Per-packet overhead      | one timestamp                             | none (uses queue counters)                |
| Smoothness               | step                                      | smooth                                    |
| Required for             | -                                         | DOCSIS 3.1 (PIE), DOCSIS 4.0 (PIE+L4S)    |
| Default `target`         | 5 ms                                      | 15–20 ms                                  |

PIE is widely deployed in cable modems precisely because it avoids per-packet timestamps
(matters at 10 Gbps line rate on cheap ASICs).

---

## 9. ECN Integration and L4S (RFC 3168 / 8311 / 9330 / 9332)

Explicit Congestion Notification lets a router signal congestion **without dropping** the
packet. The endpoints react to CE-marks the same way they'd react to drops, but the packet
still arrives.

### 9.1 Classic ECN Codepoints (RFC 3168)

The two-bit ECN field lives in the IP header (TOS byte, lower 2 bits):

| Bits | Codepoint  | Meaning                                              |
|:-----|:-----------|:-----------------------------------------------------|
| 00   | Not-ECT    | Not ECN-Capable Transport (drops only)               |
| 01   | ECT(1)     | ECN-Capable Transport, "experimental"                |
| 10   | ECT(0)     | ECN-Capable Transport, original                      |
| 11   | CE         | Congestion Experienced (set by AQM)                  |

For RFC 3168 (classic) ECN: `ECT(0)` and `ECT(1)` are equivalent — both signal "I will
react to CE-marks." A router that wants to drop a packet may instead set `CE`. If the
packet was already `CE`, the router must drop (it can't double-mark).

### 9.2 The Reaction Rule

A TCP receiver that sees `CE` on an inbound packet sets the **ECE** flag on the next ACK.
The sender treats ECE the same as a fast-retransmit signal: cut `cwnd` in half (CUBIC) or
multiplicatively (BBR variants react differently). The "1 mark = 1 drop" mapping means
classic-ECN does **not** improve throughput vs drop; it only saves the retransmission cost
and the round-trip blackout that loss recovery imposes.

### 9.3 L4S — Low Latency, Low Loss, Scalable Throughput (RFC 9330)

L4S reuses the `ECT(1)` codepoint as a separate "scalable congestion control" signal. The
key contract: an L4S-aware AQM marks **proportionally to the queue overshoot**, not as a
hard threshold, and the L4S sender reacts in **fine increments**, not by halving `cwnd`.

```
  Classic ECN:     1 CE-mark per ~RTT, sender halves cwnd     (sawtooth)
  L4S:           many CE-marks per RTT, sender adjusts proportionally  (smooth)
```

The result is sub-millisecond average queueing delay on a fully loaded link, with throughput
within a few percent of theoretical max.

### 9.4 ECN Codepoint Identification under L4S

| Field           | Classic           | L4S                                      |
|:----------------|:------------------|:-----------------------------------------|
| `ECT(0)`        | classic ECN flow  | classic ECN flow                         |
| `ECT(1)`        | classic ECN flow  | **L4S flow** (RFC 9331, redefined)       |
| `CE`            | congestion        | congestion                               |
| `Not-ECT`       | drops only        | drops only                               |

A router cannot tell L4S apart from classic ECN at L3 just by looking at the codepoint —
unless it implements the **DualPI2** dual-queue scheme.

### 9.5 DualPI2 (RFC 9332)

DualPI2 is an AQM that runs **two queues in parallel**:

- **L-queue** ("low-latency queue") — for packets marked `ECT(1)` (L4S). Tiny target ≈ 1 ms.
  Marks at high frequency.
- **C-queue** ("classic queue") — for packets marked `ECT(0)` or `Not-ECT`. Larger target
  ≈ 15 ms. Drops or classic-ECN-marks at low frequency.

```
              +------------------+
   ECT(1) ───►|  L-queue         |---> wire
              |  PI(target=1ms)  |
              |  many marks/RTT  |
              +------------------+
                       |  coupling: p_C ≥ k·p_L²
                       v
              +------------------+
   ECT(0) ───►|  C-queue         |---> wire
   Not-ECT    |  PI(target=15ms) |
              |  one drop/RTT    |
              +------------------+
```

The two queues are scheduled with a strict-priority-with-coupling rule: L-queue is served
first, but its drop/mark probability `p_L` is constrained to keep classic flows from being
starved:

```
  p_C = k · p_L²       (coupling factor k ≈ 2)
```

This quadratic coupling guarantees that as L-traffic increases its mark rate, classic
traffic sees an increasing drop rate, and the two end up sharing the link in proportion to
their congestion-response sensitivity. **Result:** a 100 Mbps link can simultaneously carry
a 5 ms-target L4S video call and a 15 ms-target classic TCP download with no cross-traffic
penalty in either direction.

### 9.6 Linux Reference

```bash
# Enable ECN sysctl-wide (sender side)
sudo sysctl -w net.ipv4.tcp_ecn=1     # 0 off, 1 active, 2 passive only

# CoDel marks instead of drops
sudo tc qdisc replace dev eth0 root fq_codel ecn

# DualPI2 (Linux 5.16+, requires kernel module)
sudo modprobe sch_dualpi2
sudo tc qdisc replace dev eth0 root dualpi2 \
    target 15ms l_target 1ms \
    coupling_factor 2 limit 10000
```

---

## 10. Linux qdisc Reference

### 10.1 `tc` Commands

```bash
# Show
tc qdisc show
tc qdisc show dev eth0
tc -s qdisc show dev eth0          # with stats

# Add/replace/del root qdisc
sudo tc qdisc add  dev eth0 root  fq_codel
sudo tc qdisc replace dev eth0 root  fq_codel
sudo tc qdisc del  dev eth0 root

# Classes (HTB)
sudo tc qdisc add dev eth0 root handle 1: htb default 30
sudo tc class add dev eth0 parent 1: classid 1:1  htb rate 100Mbit ceil 100Mbit
sudo tc class add dev eth0 parent 1:1 classid 1:10 htb rate 30Mbit  ceil 100Mbit prio 1
sudo tc class add dev eth0 parent 1:1 classid 1:20 htb rate 50Mbit  ceil 100Mbit prio 2
sudo tc class add dev eth0 parent 1:1 classid 1:30 htb rate 20Mbit  ceil 100Mbit prio 3

# Filters (mark by DSCP)
sudo tc filter add dev eth0 parent 1: protocol ip prio 1 \
    u32 match ip dsfield 0xb8 0xfc flowid 1:10            # EF (DSCP 46)

# Show classes/filters
tc -s class show dev eth0
tc -s filter show dev eth0
```

### 10.2 Common Qdiscs at a Glance

| qdisc          | Type           | Use case                                 | Default? |
|:---------------|:---------------|:-----------------------------------------|:---------|
| `pfifo`        | classless FIFO | testing                                  |          |
| `bfifo`        | byte FIFO      | testing                                  |          |
| `pfifo_fast`   | 3-band FIFO    | classic Linux default (pre-5.13)         | <5.13    |
| `fq`           | per-flow shed  | bulk uplinks where pacing matters        |          |
| `fq_codel`     | per-flow CoDel | residential router, server uplink        | ≥5.13    |
| `cake`         | FQ + shaper    | residential gateway, asymmetric link     |          |
| `htb`          | hierarchical   | rate-limit per class                     |          |
| `hfsc`         | hierarchical   | latency + throughput dual-curve          |          |
| `tbf`          | token bucket   | hard rate-limit                          |          |
| `red`          | classless RED  | datacenter, big buffers                  |          |
| `sfq`          | stochastic FQ  | legacy fairness (FQ-CoDel obsoletes)     |          |
| `pie`          | classless PIE  | DOCSIS modems, fixed-rate links          |          |
| `dualpi2`      | classless L4S  | mixed L4S + classic (5.16+)              |          |
| `mq`           | multiqueue     | multiqueue NICs (1 child qdisc per HW Q) |          |
| `noqueue`      | none           | virtual ifaces (lo, dummy, …)            |          |

### 10.3 HTB (Hierarchical Token Bucket) Math

HTB models a tree of token buckets. Each class has:

```
  rate (assured)                     guaranteed bandwidth
  ceil (= rate by default)           absolute cap (borrowing limit)
  burst                              token bucket size  (rate-burst)
  cburst                             ceil-bucket size   (ceil-burst)
  prio                               0..7 (0 highest)
```

The bucket equation, evaluated at dequeue time `t`:

```
  B(t) = min(burst, B(t−Δt) + rate · Δt)
```

A class can dequeue a packet only if `B(t) ≥ pkt.len`. If `B(t)` is empty but the class has
**borrowing** rights (`ceil > rate`), the parent's tokens are consumed. The strict priority
levels `prio 0..7` only apply when there are tokens to borrow — assured-rate classes never
starve.

**Worked example: 100 Mbps link, three classes:**

```
  Total: 100 Mbps ceil = 100Mbit
  - Voice (1:10): rate 10Mbit ceil 100Mbit prio 1   (burst 1500 cburst 1500)
  - Video (1:20): rate 30Mbit ceil 100Mbit prio 2   (burst 15000 cburst 15000)
  - Bulk  (1:30): rate 60Mbit ceil 100Mbit prio 3   (burst 30000 cburst 30000)

  At full load, all three classes get exactly their assured rate (10/30/60 Mbps).
  At 50 % load, Voice and Video at full rate, Bulk gets remaining 30 Mbps.
  At idle Voice + Video, Bulk can borrow up to ceil = 100 Mbps.
```

`burst` should be ≥ `rate · 50 ms` for typical RTTs; otherwise the bucket empties between
HZ ticks and rates are systematically below target. Common rule:

```
  burst  ≈ rate · 100 ms
  cburst ≈ ceil · 100 ms
```

### 10.4 TBF (Token Bucket Filter)

The simplest shaper. One token bucket, no classes:

```
  rate    = target rate
  burst   = bucket size in bytes (default rate · HZ)
  latency = max queueing delay before drop (default ∞)
  limit   = max queue size in bytes
```

Equations:

```
  B(t) = min(burst, B(t−Δt) + rate · Δt)
  W_max = limit / rate                       (latency at saturation)
```

```bash
sudo tc qdisc replace dev eth0 root tbf \
    rate 50Mbit burst 100kb latency 100ms
```

### 10.5 BQL — Byte Queue Limits (Driver-Level Bufferbloat Fix)

Even with FQ-CoDel as your qdisc, the NIC driver's TX ring can hold milliseconds of bloat.
BQL dynamically sizes the ring:

```
  # /sys/class/net/eth0/queues/tx-0/byte_queue_limits/
  limit         current cap on outstanding bytes (auto-tuned)
  limit_max     ceiling (set this manually if needed)
  limit_min     floor
  hold_time     re-tuning interval, default 1000 ms
```

BQL automatically converges `limit` to the smallest value that keeps the link 100 % busy.
On a 1 Gbps link, that's typically 200–800 KB; on a 100 Mbps link, 20–80 KB. Without BQL,
many drivers default to a 4 MB ring — that's 320 ms of bufferbloat on a 100 Mbps link.

---

## 11. HFSC — Hierarchical Fair Service Curve

HFSC (Hierarchical Fair Service Curve) is an old but powerful Linux qdisc that achieves
something HTB cannot: **decoupled latency and throughput guarantees**. Used heavily in real-
time gaming routers and some telephony bridges.

### 11.1 The Service Curve Idea

A *service curve* `S(t)` defines the cumulative bytes a class should receive in any window
of length `t`. HFSC supports two simultaneous service curves per class:

```
  rt curve (real-time)     guaranteed minimum bandwidth and latency
  ls curve (link-share)    fair share of excess bandwidth
  ul curve (upper limit)   absolute cap (if set)
```

Each curve is parameterized as a **two-piece linear function**:

```
  S(t) = m1 · t              for 0 ≤ t < d
       = m1 · d + m2 · (t − d) for t ≥ d
```

with `m1` = initial slope (bytes/s), `d` = breakpoint (s), `m2` = sustained slope (bytes/s).

### 11.2 Convex vs Concave Curves

```
  Concave (m1 > m2):  high initial rate, drops to sustained.   Low latency.
       bytes
        |        ____________________
        |    ___/
        |   /
        |  /
        +---|---|---|--->  time
            d

  Convex (m1 < m2):  delayed start, high sustained rate.   High throughput.
       bytes
        |                   ____________
        |               ___/
        |          ____/
        |   ______/
        +---|---|---|--->  time
            d
```

**Use cases:**

| Goal                    | rt curve            | ls curve            |
|:------------------------|:--------------------|:--------------------|
| VoIP (low jitter)       | concave: m1=200kbit, d=10ms, m2=80kbit | m1=80kbit |
| Video conf              | concave: m1=2Mbit, d=100ms, m2=1Mbit | m1=1Mbit |
| Bulk download           | (none) or convex    | m1=10Mbit, d=∞      |
| Gaming                  | concave + tight ul  | small share         |

### 11.3 Eligibility and Virtual Time

HFSC schedules by **eligibility time** — the earliest time at which a packet can be sent
without violating the service curve. The dispatcher picks the eligible packet with the
*earliest deadline*. Internally it tracks two "virtual times":

```
  rt_vt   real-time virtual time (deadline-based)
  ls_vt   link-share virtual time (fair-queueing-based)
```

The scheduler is `EDF` (Earliest Deadline First) on `rt_vt` first, then `WFQ` (Weighted
Fair Queueing) on `ls_vt`.

**Bottom line:** HFSC is the only common Linux qdisc that lets you say "VoIP gets at most
10 ms of queueing, even if the link is otherwise full." It costs CPU and tuning effort, but
on real-time systems it's worth it.

### 11.4 Linux Reference

```bash
sudo tc qdisc add dev eth0 root handle 1: hfsc default 30
sudo tc class add dev eth0 parent 1: classid 1:1 hfsc \
    sc rate 100Mbit ul rate 100Mbit
sudo tc class add dev eth0 parent 1:1 classid 1:10 hfsc \
    rt m1 200kbit d 10ms m2 80kbit \
    ls m2 80kbit
sudo tc class add dev eth0 parent 1:1 classid 1:20 hfsc \
    rt m1 2Mbit d 50ms m2 1Mbit \
    ls m2 1Mbit
sudo tc class add dev eth0 parent 1:1 classid 1:30 hfsc \
    ls m2 50Mbit ul rate 100Mbit
```

---

## 12. Bufferbloat Math — Jim Gettys, 2010

Gettys's 2010 blog post and subsequent ACM Queue article showed that consumer routers, OSes,
and NICs were shipping with absurdly oversized FIFO buffers. The math is trivial:

```
  induced_latency  =  buffer_size / bandwidth
```

### 12.1 The Worked Numbers

| Device                     | Buffer   | Bandwidth | Induced latency |
|:---------------------------|:---------|:----------|:----------------|
| 2010 home cable modem      | 256 KB   | 1 Mbps    | **2.0 s**       |
| 2010 home DSL modem        | 128 KB   | 768 kbps  | 1.3 s           |
| 2010 cellular base station | 1 MB     | 5 Mbps    | 1.6 s           |
| 2010 enterprise switch     | 4 MB     | 1 Gbps    | 32 ms           |
| 2010 datacenter NIC ring   | 4 MB     | 10 Gbps   | 3.2 ms          |
| 2024 home Wi-Fi 6 router   | 8 MB     | 100 Mbps  | 640 ms (still!) |

The 1–2 second latencies on consumer broadband were ubiquitous and *invisible to throughput
benchmarks* — speedtest.net showed full bandwidth, but a concurrent ping went from 20 ms to
1500 ms.

### 12.2 Why TCP Loved Big Buffers

Reno-era TCP used `cwnd = buffer_size + BDP` as its target. Big buffers ⇒ big steady-state
`cwnd` ⇒ "good" throughput in single-flow lab tests. Vendors competed on benchmark numbers
and added more RAM. Real-world latency wasn't measured.

```
  Reno BDP-fill rule:        cwnd_max ≈ 2·BDP + buffer
  steady-state queueing:     extra_delay ≈ buffer / r
```

### 12.3 The Mitigation

The fix is **not** "smaller buffers" — that just trades latency for loss. The fix is
**AQM** (CoDel/PIE/CAKE/DualPI2): keep the buffer as deep as the RAM allows, but drop or
mark before the queue accumulates significant standing bytes. CoDel's 5 ms target on a
100 Mbps link translates to:

```
  target_bytes = target · r = 5 ms · 12.5 MB/s = 62.5 KB
```

The buffer can still be 4 MB physically — but CoDel will start dropping at 62.5 KB of
standing queue, so the *induced latency* never exceeds 5 ms.

### 12.4 Diagnosing Bufferbloat

```bash
# Continuous ping while running a throughput test on a separate window
ping -i 0.2 8.8.8.8 &
iperf3 -c speedtest.example -t 30

# DSLReports / Waveform Bufferbloat Test report A/B/C/F grades
# anything worse than B = AQM not enabled at the bottleneck
```

---

## 13. Worked Examples

### 13.1 Optimal RED Parameters for 100 Mbps

```
  link rate         r        = 100 Mbps   = 12.5 MB/s
  packet rate                = 100e6 / (1500·8)  ≈ 8333 pps
  RTT (assumed)     T        = 100 ms
  BDP                        = r·T         = 1.25 MB    ≈ 833 packets

  min_th (Floyd):  ≈ 0.5·BDP = 416 packets
  max_th         :  ≈ 1.5·min_th = 624 packets
  max_p          :  0.10
  w_q            : 1 − exp(−1/(r/MTU·T_avg))
                 = 1 − exp(−1/(8333·0.0001)) = 1 − e^(−1.2) = 0.70 ?  too aggressive
                 use empirical 1/512 ≈ 0.00195
  buffer (qlim)  :  ≥ 2·max_th = 1248 packets ≈ 1.87 MB

  Resulting peak induced latency:  W = max_th · MTU / r
                                    = 624 · 1500 · 8 / 100e6 = 75 ms
  At avg = max_th, drop probability = 0.10 = 1 in 10 packets
```

`tc` example:

```bash
sudo tc qdisc replace dev eth0 root \
    red limit 1248000 min 624000 max 936000 \
    avpkt 1500 burst 100 ecn probability 0.10
```

(Linux RED takes thresholds in **bytes**; 624000 = 416 pkts · 1500 B.)

### 13.2 CoDel State Machine Trace Through Bufferbloat Onset

Assume a 100 Mbps link, FQ-CoDel with default target=5 ms, interval=100 ms. A bulk TCP flow
ramps up:

```
  t = 0 ms     queue empty.  s = 0.  not_dropping.  first_above_time = 0.

  t = 50 ms    flow ramps; queue = 5 KB.  s = 0.4 ms (< 5 ms).  no action.
               first_above_time stays 0.

  t = 200 ms   queue grows to 70 KB.  s = 5.6 ms.  s ≥ target!
               first_above_time = 200 + 100 = 300 ms.

  t = 250 ms   queue 80 KB.  s = 6.4 ms.  still ≥ target.  now < first_above_time.  no drop.

  t = 305 ms   queue 100 KB.  s = 8 ms.  now ≥ first_above_time.
               DROP.  count = 1.  drop_next = 305 + 100/√1 = 405 ms.  enter dropping.

  t = 320 ms   queue 95 KB.  s = 7.6 ms.  still ≥ target.  now < drop_next.  no drop.
               TCP sender's cwnd halved (saw the drop), starts new ramp.

  t = 410 ms   queue 80 KB.  s = 6.4 ms.  still above.  now ≥ drop_next.
               DROP.  count = 2.  drop_next = 410 + 100/√2 = 410 + 70.7 = 480.7 ms.

  t = 481 ms   queue 60 KB.  s = 4.8 ms.  s < target!
               EXIT dropping.  count = 0.  first_above_time = 0.

  resulting induced latency profile: spiked to 8 ms briefly, settled at 4.8 ms, never reached 30+ ms.
```

Compare to tail-drop with a 4 MB buffer: queue would have grown to 4 MB = 320 ms of latency
before any drop. CoDel kept latency under 8 ms throughout.

### 13.3 FQ-CoDel Hash Collision Math (Birthday-Paradox)

For `K = 1024` buckets and `N` flows, probability that *some* flows collide:

```
  P(collision) = 1 − (K−1)/K · (K−2)/K · … · (K−N+1)/K
              ≈ 1 − exp(−N(N−1) / (2K))
```

Setting `P = 0.5`:

```
  N ≈ √(2K · ln 2) = √(2 · 1024 · 0.693) = √(1419) ≈ 37.7
```

So with **~38 flows**, you have a 50 % chance of at least one collision; with **~80 flows**,
the *expected number* of colliding flows passes 1.

In practice this is fine: a colliding pair of flows shares a sub-queue and therefore shares
its DRR quantum — they each get half the share they'd get in their own bucket. Across
a 1024-bucket table, even at 256 flows, you have ~30 colliding flows out of 256 — about 12 %
unfair, but bounded.

### 13.4 HTB Rate-Limiting at 50 Mbps with 100 ms Latency Target

Goal: shape a 50 Mbps virtual link with `W_max = 100 ms` and FQ-CoDel for AQM beneath.

```
  rate   = 50 Mbps = 6.25 MB/s
  burst  = rate · 100 ms = 625 KB             (HTB quantum is 200 KB minimum on 64-bit)
  buffer = rate · W_max = 6.25 MB/s · 0.1 s = 625 KB
```

```bash
sudo tc qdisc add  dev eth0 root  handle 1: htb default 10
sudo tc class add  dev eth0 parent 1: classid 1:1 \
    htb rate 50Mbit ceil 50Mbit burst 625kb cburst 625kb
sudo tc class add  dev eth0 parent 1:1 classid 1:10 \
    htb rate 50Mbit ceil 50Mbit burst 625kb cburst 625kb
sudo tc qdisc add  dev eth0 parent 1:10 handle 10: \
    fq_codel target 5ms interval 100ms ecn
```

The HTB caps at 50 Mbps; FQ-CoDel keeps the per-flow latency under 5 ms; the worst-case
queueing latency is bounded by the HTB burst of `100 ms`, but FQ-CoDel will start dropping
much sooner — typical observed latency under load is 2–8 ms.

### 13.5 L4S DualPI2 Marking Decision

A packet arrives at a DualPI2 qdisc. The decision tree:

```
  if pkt.ecn_codepoint == ECT(1):
      enqueue to L-queue
      compute p_L = max(0, ((L_queue_delay - 1ms) / 4ms) clipped to [0,1])
      if random() < p_L:
          set CE on pkt           # mark, do not drop
  elif pkt.ecn_codepoint in {ECT(0), Not-ECT}:
      enqueue to C-queue
      compute p_C = k · p_L²       # coupling
      if random() < p_C:
          if pkt.ecn_codepoint == ECT(0):
              set CE on pkt
          else:
              drop pkt
```

**Worked example:** L-queue delay = 3 ms, k=2.

```
  p_L = (3 − 1)/4 = 0.5
  p_C = 2 · 0.5² = 0.5
```

Half the L-flow packets get CE-marked; half the C-flow packets get dropped (or CE-marked
if ECT(0)). Both senders react in proportion. The L-flow steady state: target 1 ms, frequent
small adjustments. The C-flow: target 15 ms, halve cwnd on each drop.

---

## 14. When to Use Which AQM

| Environment / link             | Recommended qdisc              | Why                                                    |
|:-------------------------------|:-------------------------------|:-------------------------------------------------------|
| Loopback / virtual / dummy     | `noqueue`                      | no link layer to bloat                                 |
| Server NIC, no shaping needed  | `fq_codel`                     | Linux 5.13+ default; eliminates bloat with no tuning   |
| Server NIC, pacing matters     | `fq`                           | works with TCP-PRR, BBR (per-flow pacing in qdisc)     |
| Residential gateway WAN port   | `cake bandwidth X triple-isolate diffserv4` | shaper + diffserv + flow isolation in one |
| DSL / 4G / 5G uplink           | `cake` w/ ATM/PTM overhead     | accounts for L2 framing                                |
| DOCSIS 3.1 cable modem         | `pie` (built-in, can't change) | required by DOCSIS                                     |
| DOCSIS 4.0 / future 5G         | `dualpi2`                      | L4S support                                            |
| Real-time gaming / VoIP router | `hfsc` w/ rt curves            | latency-bounded class scheduling                       |
| Datacenter NIC, > 10 Gbps      | hardware queues + `mq fq_codel`| line-rate; tune RED in switch ASICs                    |
| Datacenter switch buffer       | `red ecn` w/ tight thresholds  | 50 µs target; ECN to avoid dropping bulk flows         |
| WiFi access point downlink     | `cake` over `mac80211 airtime` | per-station fairness on wireless                       |
| Embedded / IoT                 | `pfifo_fast` or `fq_codel`     | RAM-limited; pfifo_fast if very tight                  |

### 14.1 Decision Heuristics

1. **Are you the bottleneck?** Bufferbloat happens at the slowest link. If your machine is
   not on the bottleneck, AQM on your egress queue is mostly cosmetic. Find the bottleneck
   first (`mtr`, `bufferbloat.net` test).
2. **Do you need to shape?** If yes (asymmetric link, ISP cap, etc.), `cake` is almost
   always right. If no, `fq_codel` is the no-knobs choice.
3. **Do you need per-class latency guarantees?** Use `hfsc` with `rt` curves. Otherwise
   fairness alone (FQ-CoDel) is usually enough.
4. **Do you have L4S endpoints?** (Apple's APNs, some Nvidia GeForce NOW, experimental
   Chrome flags.) Then `dualpi2` is worth it.
5. **Are you stuck with a vendor stack?** PIE on DOCSIS, vendor RED on most enterprise
   switches. Tune the params, can't replace the algorithm.

### 14.2 The "Default for Everything" Recipe

For 90 % of Linux servers and routers in 2026:

```bash
# Server (no shaping)
sudo tc qdisc replace dev eth0 root fq_codel

# Residential router with 100/20 Mbps DOCSIS line
sudo tc qdisc replace dev eth0 root cake bandwidth 95Mbit \
    diffserv4 triple-isolate docsis nat ack-filter
sudo tc qdisc replace dev eth0.1 root cake bandwidth 18Mbit \
    diffserv4 triple-isolate docsis nat ack-filter
```

Cap the shaper 5–10 % below the ISP's advertised rate so that *your* AQM is the bottleneck,
not the modem's hidden tail-drop FIFO.

---

## 15. Operational Diagnostics and Recipes

### 15.1 Verifying AQM Is Active

```bash
tc -s qdisc show dev eth0
# look for:
#   "drop_overlimit"  drops because of memory_limit
#   "ecn_mark"        ECN marks issued
#   "drop_overmemory" memory pressure
#   "new_flow_count"  new sub-queues created (FQ-CoDel/CAKE)

# pfifo_fast → no AQM
# fq_codel    → look for non-zero ecn_mark or drop_overlimit
# cake        → look for tin stats (per-DSCP-tier counters)
```

### 15.2 Reading FQ-CoDel Stats

```
qdisc fq_codel 0: root refcnt 2 limit 10240p flows 1024 quantum 1514 target 5ms interval 100ms memory_limit 32Mb ecn drop_batch 64
 Sent 12345678 bytes 9876 pkt (dropped 12, overlimits 0 requeues 0)
 backlog 0b 0p requeues 0
  maxpacket 1514 drop_overlimit 0 new_flow_count 234 ecn_mark 78
  new_flows_len 0 old_flows_len 0
```

| Field             | Meaning                                                       |
|:------------------|:--------------------------------------------------------------|
| `dropped`         | total drops                                                   |
| `drop_overlimit`  | drops due to total memory_limit hit                           |
| `new_flow_count`  | how many distinct sub-queues activated (turnover)             |
| `ecn_mark`        | how many CE-marks issued (vs drops)                           |
| `maxpacket`       | largest packet seen (sanity check for MTU)                    |
| `new_flows_len`   | sub-queues currently in `new_list` (instantaneous)            |
| `old_flows_len`   | sub-queues currently in `old_list` (instantaneous)            |

### 15.3 Reading CAKE Stats

```
qdisc cake 8001: root refcnt 2 bandwidth 100Mbit diffserv4 triple-isolate nat nowash ack-filter split-gso rtt 100ms raw overhead 18

  Sent 1234567890 bytes 987654 pkt (dropped 5, overlimits 1234)
  backlog 0b 0p requeues 12

                  Bulk     Best Effort       Video           Voice
  thresh         6.25Mbit       31.25Mbit       50Mbit          12.5Mbit
  target            5ms             5ms          5ms              5ms
  interval        100ms           100ms        100ms            100ms
  pk_delay         3ms             4ms          2ms              1ms
  av_delay         1ms             2ms          1ms              0ms
  sp_delay        0us             0us          0us              0us
  pkts            1234           65432         12345              987
  bytes        1234567        87654321      12345678           987654
  way_inds         0              12             3                 1
  way_miss        12              45             8                 2
  way_cols         0               0             0                 0
```

`pk_delay` (peak) and `av_delay` (average) are sojourn times per tier. `way_inds`,
`way_miss`, `way_cols` are flow-table inserts/misses/collisions.

### 15.4 Bufferbloat A/B Test

```bash
# Baseline: ping while idle
ping -i 0.2 -c 50 1.1.1.1 | tail -5
# rtt min/avg/max/mdev = 8.2 / 9.1 / 10.4 / 0.6 ms

# Saturate uplink in another window
iperf3 -c speedtest.example -t 30 &

# Re-run ping in foreground
ping -i 0.2 -c 50 1.1.1.1 | tail -5
# WITHOUT AQM:    rtt = 280 / 1100 / 1750 / 320 ms
# WITH FQ-CoDel:  rtt = 8.5 / 12.1 / 18.3 / 2.1 ms
```

The whole point of AQM: idle-RTT and loaded-RTT differ by under 10×.

### 15.5 Sysctl Knobs Worth Knowing

```bash
# Default qdisc for all new interfaces
sysctl net.core.default_qdisc                # show
sudo sysctl -w net.core.default_qdisc=fq_codel
echo 'net.core.default_qdisc=fq_codel' | sudo tee -a /etc/sysctl.d/99-qdisc.conf

# Enable BQL globally (mostly automatic, this just verifies)
ls /sys/class/net/eth0/queues/tx-0/byte_queue_limits/

# TCP ECN: 1 = active+passive, 2 = passive only, 0 = off
sudo sysctl -w net.ipv4.tcp_ecn=1

# Pacing inside the qdisc (fq, fq_codel)
sudo sysctl -w net.ipv4.tcp_pacing_ss_ratio=200      # slow-start pacing
sudo sysctl -w net.ipv4.tcp_pacing_ca_ratio=120      # congestion-avoidance pacing
```

---

## 16. Common Errors and Fixes

| Symptom                                     | Likely cause                            | Fix                                        |
|:--------------------------------------------|:----------------------------------------|:-------------------------------------------|
| `tc: command not found`                     | `iproute2` not installed                | `apt install iproute2` / `dnf install iproute` |
| `RTNETLINK answers: No such file or directory` | qdisc module not loaded             | `modprobe sch_fq_codel` (or sch_cake, etc) |
| `tc qdisc add` says "File exists"           | qdisc already attached                  | use `tc qdisc replace` instead of `add`    |
| Latency unchanged after adding `fq_codel`   | bottleneck is elsewhere (modem, peer)   | shape on the egress *to* the bottleneck    |
| `tc -s qdisc` shows `dropped 0` always      | not actually congested in tested window | run a real saturation test (`iperf3`)      |
| CAKE shows `bandwidth limit reached` constantly | shaper rate set too high           | reduce `bandwidth X` 5–10 % below ISP cap  |
| ECN marks zero despite congestion           | `tcp_ecn=0` (off) on senders            | `sysctl net.ipv4.tcp_ecn=1`                |
| HTB borrowing not happening                 | `ceil` equal to `rate`, no headroom     | set `ceil > rate` to allow borrowing       |
| Pings spike under load to multi-second RTT  | classic bufferbloat (no AQM)            | install `fq_codel` or `cake` on bottleneck |
| `qdisc fq_codel dropped 1000s of pkts/s`    | `memory_limit` too low                  | `tc qdisc replace ... memory_limit 64Mb`   |
| Hash collision of two heavy flows           | unlucky 5-tuple hash                    | `cake triple-isolate` (host hash on top)   |

---

## 17. Vocabulary

| Term                | Definition                                                                        |
|:--------------------|:----------------------------------------------------------------------------------|
| AQM                 | Active Queue Management — drops/marks early, before buffer overflow               |
| BDP                 | Bandwidth-Delay Product, `r·RTT` — the in-flight data needed to fill a pipe        |
| BQL                 | Byte Queue Limits — driver-level dynamic ring sizing                              |
| Bufferbloat         | Excess queue depth causing high latency without throughput loss                   |
| CAKE                | Common Applications Kept Enhanced — FQ-CoDel + shaper + diffserv + flow-iso       |
| CE                  | Congestion Experienced — ECN codepoint set by AQM                                 |
| CoDel               | Controlled Delay — sojourn-time-targeting AQM                                     |
| DRR                 | Deficit Round Robin — fair scheduler with O(1) per-packet cost                    |
| DSCP                | Differentiated Services Code Point — 6-bit traffic class in IP header             |
| DualPI2             | Dual-queue PI2 — L4S + classic AQM in one (RFC 9332)                              |
| ECN                 | Explicit Congestion Notification — mark instead of drop (RFC 3168)                |
| ECT(0), ECT(1)      | ECN-Capable Transport codepoints; under L4S, ECT(1) = "scalable"                  |
| EWMA                | Exponentially-Weighted Moving Average                                             |
| FQ                  | Fair Queueing — per-flow scheduling for fairness                                  |
| FQ-CoDel            | Fair Queueing + CoDel (RFC 8290)                                                  |
| GENTLE-RED          | RED variant with smooth ramp through `max_th`                                     |
| Goodput             | Useful throughput excluding retransmits and protocol overhead                     |
| HFSC                | Hierarchical Fair Service Curve — dual-curve AQM                                  |
| HTB                 | Hierarchical Token Bucket — class-based shaper                                    |
| Jitter              | Variation in latency, σ(W)                                                        |
| L4S                 | Low Latency, Low Loss, Scalable Throughput (RFC 9330)                             |
| Little's Law        | `L = λ·W` — universal queueing identity                                           |
| MTU                 | Maximum Transmission Unit (1500 B Ethernet default)                               |
| PI controller       | Proportional-Integral controller — used by PIE                                    |
| PIE                 | Proportional Integral controller Enhanced (RFC 8033)                              |
| Pollaczek-Khinchine | M/G/1 mean-wait formula                                                           |
| qdisc               | Linux queueing discipline — the AQM/scheduler attached to an interface            |
| QFQ                 | Quick Fair Queueing — alt scheduler to DRR                                        |
| RED                 | Random Early Detection (Floyd & Jacobson 1993)                                    |
| Shaper              | Component that enforces an upper rate, e.g. token bucket                          |
| Sojourn time        | Time a packet spent in the queue (`now − enqueue_ts`)                             |
| Tail-drop           | FIFO with hard limit — drop new arrivals when full                                |
| TBF                 | Token Bucket Filter — Linux's simplest shaper qdisc                               |
| Utilization (ρ)     | `λ/μ` — fraction of time the server is busy                                       |
| WFQ                 | Weighted Fair Queueing — fair queueing with per-flow weights                      |
| WRED                | Weighted RED — per-class RED parameters, DSCP-aware                               |

---

## 18. Try This

1. **Measure your home bufferbloat.** Run `ping -i 0.2 1.1.1.1` while running an iperf3 or
   speedtest. Compute `(loaded RTT − idle RTT) / idle RTT`. If > 5×, you have bufferbloat.
2. **Switch your default qdisc.** `sysctl net.core.default_qdisc=fq_codel`, then bring an
   interface down/up. Re-run the test from #1. Improvement should be dramatic.
3. **Tune CAKE on a residential router.** Set bandwidth 5 % below your ISP rate, enable
   `triple-isolate` and `docsis`/`atm`/`ptm`. Re-run #1.
4. **Watch a CoDel state machine in action.** `tc -s qdisc show dev eth0` repeatedly during
   a saturation test; watch `dropped`, `ecn_mark`, and `new_flow_count` change.
5. **Hash-collide on purpose.** Open 200 connections to the same host; observe `way_cols`
   in CAKE stats grow. Then switch to `triple-isolate` and re-test — collisions stay but
   per-host fairness is restored.
6. **Run an HTB rate-limit experiment.** Cap eth0 at 50 Mbps with HTB + FQ-CoDel; verify
   with `iperf3 -c host -t 10` that throughput is exactly 50 Mbps, then ping concurrently to
   verify queueing latency under 10 ms.
7. **Try DualPI2.** On a 5.16+ kernel: `modprobe sch_dualpi2; tc qdisc replace dev eth0
   root dualpi2`. Test with an L4S-capable sender (TCP Prague, BBRv3 with ECT(1)).
8. **Build a PI controller in Python.** Take the PIE equations from §8.2 and simulate a
   single-link queue under varying offered load. Plot `p_drop` vs time. Tune `α, β` and
   observe oscillation thresholds.
9. **Compute optimal RED parameters for your real link.** Measure RTT, bandwidth, MTU; plug
   into §13.1's worked example; deploy with `tc qdisc replace dev X root red ...`.
10. **Profile a CPU-bound qdisc.** On a 10 Gbps NIC, `perf top -p $(pidof your-server)`
    while the link is saturated. Look for `__qdisc_run`, `fq_codel_dequeue`, etc. Now
    compare to `cake_dequeue` — CAKE has higher per-packet cost than FQ-CoDel.

---

## 19. See Also

- `networking/tcp` — congestion control reacts to AQM signals; see slow-start, fast-retransmit, CUBIC, BBR
- `networking/cos-qos` — DSCP marking and class-of-service that AQMs like WRED/CAKE consume
- `networking/qos-advanced` — policy-driven QoS, MQC config, hierarchical queueing
- `networking/tc` — the Linux user-space tool front-end to qdiscs
- `kernel-tuning/network-stack-tuning` — sysctls (BQL, TCP small queues, ECN, pacing)
- `ramp-up/queue-management-eli5` — narrative ELI5 companion to this page
- `ramp-up/tcp-eli5` — TCP behavior under loss and ECN

---

## 20. References

- **RFC 7567** — *IETF Recommendations Regarding Active Queue Management* (Baker, Fairhurst, 2015).
- **RFC 8033** — *Proportional Integral Controller Enhanced (PIE): A Lightweight Control Scheme to Address the Bufferbloat Problem* (Pan et al., 2017).
- **RFC 8289** — *Controlled Delay Active Queue Management* (Nichols, Jacobson, McGregor, Iyengar, 2018).
- **RFC 8290** — *The Flow Queue CoDel Packet Scheduler and Active Queue Management Algorithm* (Hoeiland-Joergensen, McKenney, Taht, Gettys, Dumazet, 2018).
- **RFC 8311** — *Relaxing Restrictions on Explicit Congestion Notification (ECN) Experimentation* (Black, 2018).
- **RFC 9330** — *Low Latency, Low Loss, and Scalable Throughput (L4S) Internet Service: Architecture* (Briscoe, De Schepper, Bagnulo, White, 2023).
- **RFC 9331** — *The Explicit Congestion Notification (ECN) Protocol for Low Latency, Low Loss, and Scalable Throughput (L4S)* (De Schepper, Briscoe, 2023).
- **RFC 9332** — *Dual-Queue Coupled Active Queue Management (AQM) for Low Latency, Low Loss, and Scalable Throughput (L4S)* (De Schepper, Briscoe, White, 2023).
- **RFC 3168** — *The Addition of Explicit Congestion Notification (ECN) to IP* (Ramakrishnan, Floyd, Black, 2001).
- **Floyd & Jacobson (1993)** — *Random Early Detection Gateways for Congestion Avoidance*, IEEE/ACM ToN.
- **Floyd, Gummadi, Shenker (2001)** — *Adaptive RED: An Algorithm for Increasing the Robustness of RED's Active Queue Management*, ICSI tech report.
- **Nichols & Jacobson (2012)** — *Controlling Queue Delay*, ACM Queue 10(5).
- **Gettys (2010)** — *Bufferbloat: Dark Buffers in the Internet*, ACM Queue 9(11).
- **Bufferbloat.net** — *https://www.bufferbloat.net/* — community resource, test tools, fq_codel/cake history.
- **Linux kernel** — `net/sched/sch_fq_codel.c`, `net/sched/sch_cake.c`, `net/sched/sch_codel.c`, `net/sched/sch_pie.c`, `net/sched/sch_red.c`, `net/sched/sch_htb.c`, `net/sched/sch_hfsc.c`, `net/sched/sch_dualpi2.c`.
- **`man tc`**, **`man tc-fq_codel`**, **`man tc-cake`**, **`man tc-codel`**, **`man tc-pie`**, **`man tc-red`**, **`man tc-htb`**, **`man tc-hfsc`**.
