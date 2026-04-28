# Spanning Tree Protocol — Deep Dive

> *Spanning Tree is graph theory enforced by 35-byte BPDUs. The math behind STP is the math of minimum spanning trees over a multigraph of bridges and LANs, intertwined with a finite-state machine, a set of timers calibrated for 1990-era propagation delay, and a set of guard mechanisms layered on top to defend against the protocol's own pathologies. RSTP shrinks the timers from tens of seconds to milliseconds; MSTP federates multiple instances over a common physical fabric; PVST+ trades scaling efficiency for per-VLAN steering. Every loop, every blackhole, every "the network was fine until somebody plugged in a switch" outage in a campus or data-center fabric reduces back to BPDU exchange and the topology that emerges.*

---

## 0. Notation and Preliminaries

Throughout this deep dive we use the following notation:

- $G = (V, E)$ — the bridged LAN modelled as a graph. $V$ is the set of bridges (switches) plus pseudonodes for each multi-access LAN segment; $E$ is the set of bridge-port-to-LAN attachments.
- $|V| = n$, $|E| = m$. For typical campus topologies $m \approx 2n$; for ring topologies $m = n$.
- $\text{BID}(v)$ — the 8-byte Bridge ID of bridge $v$: 2 bytes priority concatenated with the 6-byte MAC base address. We treat BID as a 64-bit integer for ordering purposes.
- $c : E \to \mathbb{N}_{\geq 1}$ — the IEEE-defined path cost on each bridge port, with $c \in [1, 2^{16}-1]$ for short-form 802.1D and $c \in [1, 2^{32}-1]$ for long-form 802.1t.
- $r$ — the root bridge: $r = \arg\min_{v \in V} \text{BID}(v)$.
- $d(v) = \min_{P \in \mathcal{P}(v, r)} \sum_{e \in P} c(e)$ — root path cost from bridge $v$ to root $r$.
- $\text{HT}, \text{MaxAge}, \text{FD}$ — HelloTime (default 2s), MaxAge (default 20s), and ForwardDelay (default 15s).

The ten BPDU fields by byte index in the 802.1D Configuration BPDU:

| Offset | Length | Field                                  |
|-------:|-------:|:---------------------------------------|
|   0    |   2    | Protocol Identifier (always 0x0000)    |
|   2    |   1    | Protocol Version Identifier            |
|   3    |   1    | BPDU Type                              |
|   4    |   1    | Flags (TC, TCAck, Proposal, etc.)      |
|   5    |   8    | Root Identifier (Bridge ID)            |
|  13    |   4    | Root Path Cost                         |
|  17    |   8    | Bridge Identifier                      |
|  25    |   2    | Port Identifier                        |
|  27    |   2    | Message Age                            |
|  29    |   2    | Max Age                                |
|  31    |   2    | Hello Time                             |
|  33    |   2    | Forward Delay                          |

That's 35 bytes total; RSTP adds 2 bytes (Version 1 Length = 0x00) for a 37-byte BPDU; MSTP adds 5 bytes plus a variable-length MSTI configuration list.

---

## 1. STP / RSTP / MSTP / PVST+ — The Variant Landscape

### 1.1 Comparison Table

| Variant       | Standard         | Year | Convergence (typical)    | Tree Count          | Inventor / Holder        | Notes                                                                                              |
|:--------------|:-----------------|-----:|:-------------------------|:---------------------|:--------------------------|:---------------------------------------------------------------------------------------------------|
| STP           | IEEE 802.1D-1990 | 1990 | 30–50 s                  | 1 (CST)              | Radia Perlman / DEC       | Original; uses 802.3 SAP frames; single tree for all VLANs. Now obsolete but defines the wire format.|
| RSTP          | IEEE 802.1w-2001 | 2001 | < 1 s (often 100–200 ms) | 1 (CST)              | IEEE                      | Rolled into 802.1D-2004; introduces proposal/agreement; deprecates listening state.                  |
| MSTP          | IEEE 802.1s-2002 | 2002 | < 1 s per instance       | up to 64 MSTIs + CIST| IEEE                      | Rolled into 802.1Q-2005; multiple instances over one physical mesh; region concept.                  |
| PVST+         | Cisco proprietary| 1995 | 30–50 s                  | one per VLAN         | Cisco                     | Per-VLAN; SSTP-formatted BPDUs sent to PVST+ multicast 0x0100.0CCC.CCCD; trunk-tagged on non-1.       |
| Rapid PVST+   | Cisco proprietary| 2003 | < 1 s                    | one per VLAN         | Cisco                     | RSTP per VLAN; convergence behavior of RSTP with per-VLAN load balance.                              |
| MST (Cisco impl)| based on 802.1s| 2003 | < 1 s per instance       | up to 65 MSTIs       | Cisco                     | Cisco's MSTP variant: 64 MSTIs + IST/MST0 = 65 instances; revision-numbered region boundary.         |

The two practical lineages are (a) IEEE-standard tree-per-fabric vs (b) Cisco's per-VLAN tree. MSTP is the IEEE compromise that gives back the load-balance benefit of per-VLAN STP without the BPDU explosion.

### 1.2 Wire Compatibility Matrix

```
            | STP send | STP recv | RSTP send | RSTP recv | MSTP send | MSTP recv |
------------+----------+----------+-----------+-----------+-----------+-----------+
STP bridge  |   yes    |   yes    |     no    |    yes    |     no    |    yes*   |
RSTP bridge |   yes    |   yes    |    yes    |    yes    |     no    |    yes*   |
MSTP bridge |   yes    |   yes    |    yes    |    yes    |    yes    |    yes    |
```
\* MSTP BPDUs received by an STP/RSTP bridge are interpreted as RSTP (the BPDU type field 0x02 is shared); only the MSTP-specific tail is ignored.

A single legacy STP bridge in an RSTP region drops the entire region back to STP-style convergence on any segment that touches it (RSTP detects the BPDU version and falls back). This is one of the most common "why is convergence still 30 seconds" causes.

---

## 2. BPDU Format — Field-by-Field Math

### 2.1 802.1D Configuration BPDU (35 bytes)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       Protocol Identifier       |Vers |   BPDU Type   |Flags|
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+               Root Bridge ID  (8 bytes)                       +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Root Path Cost  (4 bytes)               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+               Bridge ID  (8 bytes)                            +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Port ID         |       Message Age (1/256 s)         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|        Max Age          |          Hello Time                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       Forward Delay     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

| Bytes | Field                | Notes                                                                              |
|------:|:---------------------|:-----------------------------------------------------------------------------------|
|   2   | Protocol Identifier  | 0x0000 — IEEE 802.1D                                                                |
|   1   | Version              | 0x00 STP, 0x02 RSTP, 0x03 MSTP                                                      |
|   1   | BPDU Type            | 0x00 Configuration, 0x80 TCN, 0x02 RST/MST                                          |
|   1   | Flags                | bit0 TC, bit7 TCAck, RSTP bits1-2 = port role, bit3 Learning, bit4 Forwarding, bit5 Agreement, bit6 Proposal |
|   8   | Root ID              | 2-byte priority + 6-byte MAC                                                        |
|   4   | Root Path Cost       | unsigned, summed along root path                                                    |
|   8   | Sender Bridge ID     | priority + MAC of the transmitting bridge                                           |
|   2   | Port ID              | 4-bit priority + 12-bit port number (or 8/8 in long-PortID extension)               |
|   2   | Message Age          | hops × $\frac{1}{256}$ s offset; incremented by each bridge                         |
|   2   | Max Age              | timer in $\frac{1}{256}$ s; default 0x1400 = 20 s                                   |
|   2   | Hello Time           | timer in $\frac{1}{256}$ s; default 0x0200 = 2 s                                    |
|   2   | Forward Delay        | timer in $\frac{1}{256}$ s; default 0x0F00 = 15 s                                   |

Total: $2+1+1+1+8+4+8+2+2+2+2+2 = 35$ bytes payload. Add 14-byte 802.3 header (DA `01:80:C2:00:00:00`, SA = port MAC, EtherType = length 0x0026), 3-byte LLC (DSAP `0x42`, SSAP `0x42`, Ctrl `0x03`), and 4-byte FCS for a 56-byte total frame. With the 8-byte preamble + 12-byte IFG, on the wire it's 76 byte-times.

### 2.2 802.1w (RSTP) Configuration BPDU (37 bytes)

RSTP appends a single byte called *Version 1 Length* set to 0x00 and a reserved 1-byte field. The flags byte is fully populated:

| Bit | Meaning                |
|----:|:-----------------------|
|  0  | Topology Change        |
|  1  | Proposal               |
| 2-3 | Port Role (00 unknown, 01 alternate/backup, 10 root, 11 designated) |
|  4  | Learning               |
|  5  | Forwarding             |
|  6  | Agreement              |
|  7  | Topology Change Ack    |

Two roles share encoding 01: the receiver disambiguates *alternate* (a backup root path on a different LAN) from *backup* (a backup designated port on the same shared LAN) using port and BID matching.

### 2.3 802.1s (MSTP) BPDU (variable length)

```
+------------------------------------------+
| Standard 35-byte CIST configuration      |  (acts as common-and-internal spanning tree)
+------------------------------------------+
| Version 1 Length = 0x00         (1 byte) |
+------------------------------------------+
| Version 3 Length                (2 bytes)|
+------------------------------------------+
| MST Configuration Identifier (51 bytes)  |  (Format ID 1 byte, Name 32, Revision 2, Digest 16)
+------------------------------------------+
| CIST Internal Root Path Cost    (4 bytes)|
+------------------------------------------+
| CIST Bridge Identifier          (8 bytes)|
+------------------------------------------+
| CIST Remaining Hops             (1 byte) |
+------------------------------------------+
| MSTI Configuration Messages   (16 × N)   |  (one per active MSTI)
+------------------------------------------+
```

A typical MSTP BPDU with 4 MSTIs is $35 + 1 + 2 + 51 + 4 + 8 + 1 + 16 \times 4 = 166$ bytes payload.

The 16-byte digest is HMAC-MD5 over the VLAN-to-instance map using the constant key `0x13AC06A62E47FD51F95D2BA243CD0346`. Two bridges only join the same MST region if all three of (region name, revision number, digest) match exactly. A typo in the region name puts the bridge at the boundary, where it appears as a single virtual STP bridge to other regions.

---

## 3. Bridge Priority and Tiebreaks

### 3.1 Bridge ID Layout (legacy 802.1D, no system extension)

```
| Priority (16 bits) | MAC Address (48 bits) |
|--------------------|------------------------|
| 0x8000 = 32768     | 00:1B:21:3C:8F:21      |
```

Default priority is `0x8000` = 32768. Range is 0–65535, but commonly restricted by hardware to multiples of 4096 (i.e. 0, 4096, 8192, …, 61440) — this gives 16 settable values: `0x0000`, `0x1000`, `0x2000`, `0x3000`, `0x4000`, `0x5000`, `0x6000`, `0x7000`, `0x8000`, `0x9000`, `0xA000`, `0xB000`, `0xC000`, `0xD000`, `0xE000`, `0xF000`.

Bridge ID is treated as a 64-bit unsigned integer; smaller numerical value wins.

### 3.2 Extended System ID (PVST+ / MSTP)

Modern switches encode the per-VLAN priority in the *high* nibble plus an 12-bit System ID Extension:

```
| 4 bits | 12 bits | 48 bits |
| Pri.    | VLAN ID | MAC     |
```

Effective priority for VLAN $V$ on a bridge configured with priority value $P$ (where $P$ is a multiple of 4096):

$$
\text{Pri}_{\text{eff}}(P, V) = \left\lfloor \frac{P}{4096} \right\rfloor \cdot 4096 + V
$$

So a bridge configured `spanning-tree vlan 100 priority 24576` yields effective priority $24576 + 100 = 24676$. A bridge configured `spanning-tree vlan 200 priority 24576` yields $24576 + 200 = 24776$. Both come from the same `priority 24576` knob, but the VLAN-ID lift gives PVST+ its per-VLAN root selection.

This means *the configurable priority value is constrained to multiples of 4096*. Setting `priority 24500` is rejected because the 12 low bits are reserved for the system ID extension.

### 3.3 Election Algorithm

```
ROOT_ELECT(G):
    for each v in V:
        v.root      <- v
        v.root_cost <- 0
        broadcast Configuration BPDU(root=v, root_cost=0, sender=v, ...)
    repeat:
        for each received BPDU(R, c, sender, port):
            if BID(R) < BID(v.root) or
               (R == v.root and c + cost(port) < v.root_cost) or
               (R == v.root and c + cost(port) == v.root_cost and BID(sender) < BID(v.designated)):
                v.root          <- R
                v.root_cost     <- c + cost(port)
                v.designated    <- sender
                v.root_port     <- port
                rebroadcast updated BPDU on every other port
    until no change for MaxAge
```

Tiebreak ordering (used both for root election and for root-port selection at intermediate bridges):

1. Lowest **Root Bridge ID**.
2. Lowest **Root Path Cost**.
3. Lowest **Sender (Designated) Bridge ID**.
4. Lowest **Sender (Designated) Port ID**.
5. Lowest **Receiving Port ID**.

The fifth criterion handles the corner case where the same bridge connects to a single peer over two parallel links — both BPDUs carry identical sender BID and port priority, so the local port number breaks the tie.

### 3.4 Worked Tiebreak

Two bridges A and B both configured with `priority 32768`:

```
A: priority=32768, MAC=00:1A:2B:3C:4D:50  -> BID = 0x80001A2B3C4D50
B: priority=32768, MAC=00:1A:2B:3C:4D:60  -> BID = 0x80001A2B3C4D60
```
$\text{BID}(A) < \text{BID}(B)$, so **A wins**. Tiebreak rule: lowest MAC wins, by 16 (0x60 - 0x50) numeric difference in the last byte.

Practical lesson: in a fresh fabric the root tends to be the *oldest* switch (oldest MAC OUI assignment). Always pin priority deliberately on the intended root and secondary-root.

---

## 4. Path Cost — IEEE Defined Tables

### 4.1 Short-form (802.1D-1998)

The original 16-bit cost field gave room only for coarse buckets:

| Link Speed     | Cost  |
|---------------:|------:|
| 4 Mb/s         | 250   |
| 10 Mb/s        | 100   |
| 16 Mb/s        | 62    |
| 100 Mb/s       | 19    |
| 1 Gb/s         | 4     |
| 10 Gb/s        | 2     |
| 100 Gb/s       | 1     |
| ≥ 1 Tb/s       | 1 (saturates) |

Note the unreasonable compression: 10 Gb/s and 1 Tb/s both round to ≤ 2. This is why the long-form table was added.

### 4.2 Long-form (802.1t-2001 / 802.1D-2004)

The cost field expanded to 32 bits and a new schedule was defined:

| Link Speed     | Cost           |
|---------------:|---------------:|
| 100 kb/s       | 200 000 000    |
| 1 Mb/s         | 20 000 000     |
| 10 Mb/s        | 2 000 000      |
| 100 Mb/s       | 200 000        |
| 1 Gb/s         | 20 000         |
| 10 Gb/s        | 2 000          |
| 40 Gb/s        | 500            |
| 100 Gb/s       | 200            |
| 400 Gb/s       | 50             |
| 1 Tb/s         | 20             |
| 10 Tb/s        | 2              |

Closed form: $\text{cost} = \lceil 20\,000\,000\,000 / R_{\text{bps}}\rceil$ rounded to a discrete table value. The table is hand-tuned to keep small integers across mixed-speed networks.

### 4.3 Aggregate-Link Cost

For an EtherChannel / LAG / LACP bundle, the effective link rate is the sum of member-link rates and the cost is recomputed:

$$\text{cost}_{\text{LAG}} = \begin{cases} \text{cost}(R_{\text{total}}) & \text{(IEEE: recompute)} \\ \frac{\text{cost}(R_{\text{member}})}{N} & \text{(approximation)} \end{cases}$$

For a 2-member 1 Gb/s LAG (effective 2 Gb/s) the IEEE long-form cost is interpolated to ~10 000. For a 4-member 10 Gb/s LAG (effective 40 Gb/s) the cost is exactly 500, matching the native 40 Gb/s row.

### 4.4 Cost Mode Mismatch

```bash
# Cisco IOS — switch between modes
spanning-tree pathcost method short    # 802.1D legacy
spanning-tree pathcost method long     # 802.1t — recommended for any 10 G+ network

# Verify
show spanning-tree pathcost method
```

A topology with mixed short/long mode at different bridges produces *non-monotone* root-path-cost comparisons across hops (a bridge sees `cost = 19 + 2 = 21` in short-form land but the RPC field in the BPDU is interpreted as 32-bit long-form `21`, which is an absurd number). The flag in modern IOS-XE/NX-OS to ensure all bridges use long-form is `spanning-tree pathcost method long` set globally on every bridge.

### 4.5 Manual Cost Override

```bash
# Cisco IOS — set port cost
interface GigabitEthernet0/1
 spanning-tree cost 500           # short-form value
 spanning-tree vlan 10 cost 500   # PVST+ per-VLAN

# Junos
[edit protocols rstp]
interface ge-0/0/1 cost 500;

# Arista EOS
interface Ethernet1
 spanning-tree cost 500
```

Use cases for manual override: forcing a non-shortest path to be preferred (e.g. a fiber-WDM trunk with low jitter even though it's slower), or making failover paths deterministic in symmetric topologies.

---

## 5. Convergence Time — Classic STP Timer Math

### 5.1 The Three Timers

| Timer            | Default | Range     | Function                                               |
|:-----------------|--------:|:----------|:-------------------------------------------------------|
| HelloTime (HT)   | 2 s     | 1–10 s    | BPDU emission interval from designated ports           |
| MaxAge           | 20 s    | 6–40 s    | BPDU age before it is considered stale                 |
| ForwardDelay (FD)| 15 s    | 4–30 s    | Time spent in Listening *and* in Learning              |

The constraint that anchors them all (802.1D §17.14, often called the *spec consistency inequality*):

$$
2 \cdot (\text{HT} + 1) \;\leq\; \text{MaxAge} \;\leq\; 2 \cdot (\text{FD} - 1)
$$

The left inequality ensures BPDUs can traverse a maximum-diameter network within MaxAge; the right ensures the bridge cannot transition through Listening + Learning faster than MaxAge expires upstream (otherwise downstream bridges could install forwarding state for a stale topology).

Plugging defaults: $2 \cdot (2 + 1) = 6 \leq 20 \leq 2 \cdot (15 - 1) = 28$. Satisfied.

### 5.2 Worst-Case Convergence

The four phases after a *root-port failure* on a downstream bridge:

```
event:  link down
        |
        | wait for MaxAge to expire on inferior alternate BPDUs    : up to 20 s
        v
        listening                                                  : 15 s (FD)
        |
        v
        learning                                                   : 15 s (FD)
        |
        v
        forwarding
```

Worst case: $20 + 15 + 15 = 50\ \text{s}$.

For an *indirect failure* (the local link is up but a peer two hops away has failed), the bridge keeps receiving cached BPDUs at HelloTime intervals; only when those expire after MaxAge does the bridge re-elect. Best case is then $20 + 0 + 15 + 15 = 50\ \text{s}$ as well.

For a *direct link failure* with `UplinkFast` (Cisco extension), the bridge has a pre-computed alternate root port and skips Listening + Learning, transitioning to Forwarding in 1–5 s. UplinkFast is essentially the precursor to RSTP's alternate-port concept.

### 5.3 Aggressive Timers — When Tuning Helps and When It Bites

```bash
# Cisco IOS — tune for a small fabric (only on the root!)
spanning-tree vlan 10 hello-time 1
spanning-tree vlan 10 forward-time 7
spanning-tree vlan 10 max-age 10
```

Substitute into the inequality: $2 \cdot 2 = 4 \leq 10 \leq 2 \cdot 6 = 12$. Satisfied.

Worst-case convergence shrinks to $10 + 7 + 7 = 24\ \text{s}$, but BPDU rate triples (HT halved). Since BPDUs are processed by the CPU on most low-end bridges, dense topologies with HT=1 risk control-plane saturation.

The root bridge owns the timers; downstream bridges adopt the timers carried in BPDUs. This is why you tune timers only on the root (or on the secondary root, in case of failover).

### 5.4 Bound on Diameter

The MaxAge timer also bounds the *legal* diameter of the topology. Each bridge increments the *Message Age* field by 1 (in $\frac{1}{256}$ s units) when forwarding a BPDU; when Message Age ≥ MaxAge the BPDU is dropped. With default MaxAge = 20 s and an increment of 1 s per hop (the IEEE-recommended ceiling), the maximum diameter is 7 bridges. Bridges beyond hop 7 cannot reliably maintain root-bridge state.

This is the origin of the "STP supports 7-deep topologies" rule of thumb. RSTP keeps the same default (6 hops by IOS default to leave headroom).

---

## 6. RSTP — Sub-Second Convergence

### 6.1 State Machine

802.1D had five port states: Disabled, Blocking, Listening, Learning, Forwarding. RSTP collapses to three: **Discarding, Learning, Forwarding**.

| RSTP state  | Includes 802.1D states          | MAC learning | Forwarding |
|:------------|:--------------------------------|:------------:|:----------:|
| Discarding  | Disabled, Blocking, Listening   | no           | no         |
| Learning    | Learning                        | yes          | no         |
| Forwarding  | Forwarding                      | yes          | yes        |

### 6.2 Port Roles

| Role         | Description                                                    |
|:-------------|:---------------------------------------------------------------|
| Root         | Best path to the root bridge from this bridge                  |
| Designated   | Best path *to* this LAN from the root                          |
| Alternate    | Backup root path, blocked, in Discarding                       |
| Backup       | Backup designated on the same shared LAN, blocked, in Discarding |
| Disabled     | Administratively down                                          |
| Edge         | Connected to a host (no BPDU expected) — PortFast equivalent   |

### 6.3 Proposal/Agreement Handshake

When two RSTP bridges form an adjacency over a *point-to-point* link:

```
A ────▶ B    A sends Proposal: "I propose to be designated for this segment"
A ◀──── B    B sets Sync flag: blocks all non-edge ports below
A ◀──── B    B sends Agreement: "OK, you're designated"
A ────▶ B    A and B transition to Forwarding
```

The whole exchange is a few BPDUs over a single link — typically 50–200 ms. No timer wait.

The critical constraint: this only works on **point-to-point** links. Half-duplex or shared-media segments (where multiple bridges could be designated candidates) revert to the slow Listening + Learning cycle.

### 6.4 P2P Inference

```bash
# By default, full-duplex Ethernet is treated as P2P:
show spanning-tree interface gi0/1 detail
  Link type is point-to-point by default

# Force or disable:
interface GigabitEthernet0/1
 spanning-tree link-type point-to-point   # force P2P
 # or
 spanning-tree link-type shared           # force shared (slow convergence)
```

Half-duplex auto-negotiation falls back to *shared* type and slow convergence. This is one of the silent regressions when a 1 Gb/s port renegotiates down to 100 Mb/s half-duplex on a flaky cable.

### 6.5 RSTP Topology Change Without TCN BPDUs

In 802.1D, only the root sent TC notifications. RSTP elevates every non-edge bridge to be able to flood TC: when a non-edge port goes to Forwarding (i.e. a real topology change, not just an end-host coming up), the bridge sets the TC flag in *every* BPDU on *every* designated port for `TC While = 2 × HT` seconds.

The MAC learning table on every receiving bridge is flushed for that VLAN, identical to legacy STP behavior, but the propagation is faster because every bridge originates its own TC.

---

## 7. MSTP — Federated Trees

### 7.1 The Region Concept

A bridge belongs to a region defined by three attributes:

1. **Region Name** — 32-byte ASCII string.
2. **Revision Number** — 16-bit integer.
3. **VLAN-to-Instance Map Digest** — 16-byte HMAC-MD5 of the VLAN-to-MSTI table using the constant key from 802.1Q.

If any one of these three differs across two adjacent bridges, the bridges are in *different regions*. At a region boundary, the entire neighbour region appears as a single virtual bridge running CST (Common Spanning Tree).

### 7.2 CIST and MSTIs

| Tree   | Scope                | Carries                                                       |
|:-------|:---------------------|:--------------------------------------------------------------|
| CIST   | All bridges, all regions | The "outer" tree across regions; equivalent to RSTP/CST   |
| IST    | Within one region    | The "inner" half of CIST inside a region; MST instance 0      |
| MSTI 1 | Within one region    | First user-defined tree; mapped VLANs                         |
| ...    |                      |                                                                |
| MSTI N | Within one region    | Up to 64 (IEEE) or 65 with IST (Cisco) per region             |

The full breakdown of the IEEE-allowed instance count is `MST0 + MSTI 1..63 = 64 instances` in the 802.1s text; Cisco's implementation uses `IST + MSTI 1..64 = 65`. Vendors differ here because of how they pack VLAN bitmaps into the BPDU.

### 7.3 VLAN-to-Instance Mapping

```bash
# Cisco IOS — assign VLANs to MSTI
spanning-tree mst configuration
 name CORE-REGION
 revision 1
 instance 1 vlan 10-19
 instance 2 vlan 20-29
 instance 3 vlan 30-39
 exit

# Verify the digest
show spanning-tree mst configuration
  Name      [CORE-REGION]
  Revision  1     Instances configured 4
  Instance  Vlans mapped
  --------  ---------------------------------------
  0         1-9,40-4094
  1         10-19
  2         20-29
  3         30-39
  Digest    0xDA47BC8B70D11D5BB29DCE3F234B6F47
```

If two bridges report different digests they cannot share an MSTP region. The most common cause is one bridge missing a VLAN range (e.g. bridge A has `instance 1 vlan 10-19` and bridge B has `instance 1 vlan 10-29`).

### 7.4 Inter-Region Behaviour

```
Region A          Region B           Region C
+--------+        +--------+         +--------+
|        |--------|        |---------|        |
|        |        |        |         |        |
+--------+        +--------+         +--------+

CST (CIST external):  one tree across the whole topology;
                      each region is one virtual bridge.

IST (CIST internal):  one tree per region; carries CIST + MSTI 0.
                      The IST root is the boundary bridge that won CST election.

MSTI N:               one tree per region per N; bounded by region edges.
```

The boundary port runs both the IST and CST states; from the outside it forwards with CIST role but internally it holds MSTI 0 (IST) state for the local region.

### 7.5 Calculating Boundary Cost

Inter-region path cost is the sum of *external* costs (CST) plus the IST cost inside each region traversed. This is why two regions of identical internal cost may produce asymmetric inter-region paths if the boundary links are mismatched.

---

## 8. Guards and Filters — Defending the Tree

### 8.1 BPDU Guard

```bash
# Cisco IOS — globally enable on all PortFast ports
spanning-tree portfast bpduguard default

# Per-port
interface GigabitEthernet0/1
 switchport mode access
 spanning-tree portfast
 spanning-tree bpduguard enable
```

Behaviour: when *any* BPDU arrives on a BPDU-guard-enabled port, the port is err-disabled (admin shutdown). The port stays down until either (a) recovery interval expires, or (b) admin issues `shutdown ; no shutdown`.

```bash
errdisable recovery cause bpduguard
errdisable recovery interval 300        # auto-recover after 5 min
```

This is the canonical defense against a user plugging a hub or unmanaged switch into a desk port.

### 8.2 BPDU Filter

```bash
spanning-tree portfast bpdufilter default
```

Behaviour: silently drops all BPDUs in *and* stops the port from sending any. The port becomes invisible to STP, which means an accidental hub on that port would silently form a loop because there's no detection.

> ⚠️ **DANGEROUS**. Use only when the operator absolutely understands that the downstream device is a host and that no Layer-2 loop can form. In data-center toplologies BPDU filter is rarely the right answer; BPDU guard is.

The semantics differ slightly between *globally enabled* (sends 10 BPDUs at startup, then stops) vs *per-port enabled* (never sends or accepts BPDUs).

### 8.3 Root Guard

```bash
interface GigabitEthernet0/1
 spanning-tree guard root
```

Behaviour: if a *superior* BPDU (lower BID than current root) is received on this port, the port enters the *root-inconsistent* state — equivalent to Discarding. Once superior BPDUs stop arriving for `Forward Delay` (15 s default), the port automatically reverts to Listening then Learning then Forwarding.

Use case: pin the root to the intended bridge; any rogue switch trying to claim root from a leaf-side port is silenced.

```
Root bridge S1 (priority 4096)
   |
   | core link with root guard
   |
S2 (priority 32768)  <-- if S2 becomes root, root guard discards superior BPDUs from rogue
   |
S3 (rogue, priority 0) ❌  blocked at S2's downstream port
```

### 8.4 Loop Guard

```bash
spanning-tree loopguard default        # global
# or per-interface
interface GigabitEthernet0/1
 spanning-tree guard loop
```

Behaviour: tracks BPDU reception on root and alternate ports. If a root or alternate port stops receiving BPDUs (typically due to a unidirectional fiber failure), instead of transitioning to Forwarding (the default behavior, since "no superior BPDU = I should become designated"), loop guard puts the port into *loop-inconsistent* state — Discarding.

This defends against the classic single-fiber-failure scenario: the receive direction breaks, the bridge stops receiving BPDUs, the port aged out, and then the bridge believes it's now the designated bridge for that segment, opens the port, and creates a loop because the transmit direction is still functional.

```
S1 -----TX/RX OK-----> S2     (normal)
S1 -----TX OK-------->        (RX broken on S1's link)
S1 <----RX broken---- S2

Without loop guard: S1 ages out S2's BPDUs, transitions blocked port to Forwarding -> LOOP
With loop guard:    S1 ages out, transitions to loop-inconsistent (Discarding) -> safe
```

When BPDUs resume, loop guard auto-recovers without intervention.

### 8.5 UDLD (UniDirectional Link Detection)

UDLD is a Cisco protocol (not part of 802.1D) that complements loop guard: it actively sends Layer-2 keepalives carrying the local bridge ID and port ID; if a peer reports it does not see the local ID, the link is considered unidirectional and is err-disabled.

```bash
udld aggressive                   # global, recommended
interface GigabitEthernet0/1
 udld port aggressive
```

| Mode      | Behaviour on detection |
|:----------|:-----------------------|
| Normal    | Logs the issue; keeps port up |
| Aggressive| Err-disables the port  |

UDLD acts at the physical layer; loop guard at the spanning-tree layer. Combined, they catch most unidirectional failures.

### 8.6 Guards Comparison

| Guard       | Detects                      | Action                       | Auto-recover    | Layer  |
|:------------|:-----------------------------|:-----------------------------|:----------------|:-------|
| BPDU guard  | BPDU on edge port            | err-disable                  | err-recover     | port   |
| BPDU filter | (drops BPDUs silently)       | none                         | n/a             | port   |
| Root guard  | Superior BPDU on internal port | root-inconsistent (block)  | yes, after FD   | port   |
| Loop guard  | BPDU absence on root/alt port| loop-inconsistent (block)    | yes, on BPDU    | port   |
| UDLD        | One-way fiber                | err-disable (aggressive)     | err-recover     | physical |

---

## 9. PortFast / Edge Ports

### 9.1 Definition

PortFast (Cisco) and Edge Port (IEEE) are functionally identical: a port configured as edge is assumed to be host-facing and skips Listening + Learning, transitioning directly from Disabled → Forwarding when the link comes up.

```bash
# Cisco IOS
interface GigabitEthernet0/24
 switchport mode access
 spanning-tree portfast            # interface mode
# global default for access ports
spanning-tree portfast default

# Junos
[edit protocols rstp]
interface ge-0/0/24 edge;

# Arista EOS
interface Ethernet24
 spanning-tree portfast            # access port equivalent
 spanning-tree portfast network    # network-port equivalent — RSTP non-edge
```

### 9.2 Edge Detection in RSTP

RSTP can auto-promote to edge: if no BPDU is received for `3 × HelloTime = 6 s`, the port is treated as edge. If a BPDU does arrive later, the port immediately demotes to non-edge.

This is why connecting a switch to a port previously used by a host briefly delays the new bridge's adjacency.

### 9.3 Why PortFast Without BPDU Guard is a Bug

```
                                         user plugs in a switch
                                                  |
PortFast goes edge -> Forwarding (bypass timers) v
                                          [BPDU received]
                                                  |
   PortFast retracts edge designation, runs full Listening+Learning
                                                  |
   But during the brief Forwarding window, frames may have looped
```

The standard recipe for access ports: **PortFast + BPDU guard, always**. The guard ensures that if BPDU arrives (i.e. a switch was plugged in), the port is shut down rather than just retracted.

### 9.4 PortFast Trunk

For server-to-virtual-switch trunks where the server is known to be a host:

```bash
interface TenGigabitEthernet1/1
 switchport mode trunk
 spanning-tree portfast trunk     # PortFast on a trunk port
 spanning-tree bpduguard enable
```

This lets PortFast apply on a trunk link (otherwise PortFast is ignored on trunks). Combined with BPDU guard it is safe.

---

## 10. TCN — Topology Change Notification

### 10.1 The TCN BPDU

A TCN BPDU is the simplest BPDU on the wire: just 4 bytes of payload.

```
+-----------------+-----------+----------------+
| Protocol ID 0x0000 | Version 0x00 | Type 0x80 |
+-----------------+-----------+----------------+
```

That's it. No timers, no path cost, no bridge ID. Just "topology changed somewhere."

### 10.2 Propagation

```
                  Root (top)
                /   |   \
              S1   S2   S3
              |    |    |
              ...   ...   ...
                                 1) Leaf bridge X detects local TC.
                                 2) X sends TCN BPDU upstream on its root port.
                                 3) Each upstream bridge ACKs (TC-Ack flag in next Conf BPDU)
                                    and forwards the TCN further up its own root port.
                                 4) When the root receives the TCN, it sets the TC flag in
                                    every Configuration BPDU it sends out.
                                 5) Every bridge receiving a Conf BPDU with TC=1 flushes
                                    its MAC table — but using ForwardDelay age, not the
                                    normal 5-min default age.
```

### 10.3 MAC Aging During TC

| State                | MAC table aging time |
|:---------------------|---------------------:|
| Stable               | 300 s default        |
| TC active            | 15 s (= ForwardDelay)|

Aging is reduced for `MaxAge + ForwardDelay = 35 s` after the TC flag was last set, then returns to 300 s.

### 10.4 Why MAC Flushing Matters

A topology change typically means traffic for some MAC is no longer reachable via the originally-learned port. Without aging, frames headed for that MAC are blackholed for up to 5 minutes (the default aging timer). The shortened aging guarantees the MAC will be re-learned via the new path within 35 s.

### 10.5 RSTP Variant: TCWhile

RSTP doesn't use TCN BPDUs. Instead, every non-edge bridge that detects a topology change starts a `tcWhile` timer of `2 × HelloTime` and floods TC=1 in all its outbound BPDUs. The flooding is parallel rather than upstream-then-downstream, halving propagation time.

```
S1 detects TC -> set TC flag in BPDUs -> next-hop bridges flush MAC -> they set TC flag too
   -> spreads in parallel via every designated port
```

Aging is still reduced to ForwardDelay during TC.

### 10.6 TC Generation Triggers

| Cause                                    | Generates TC | Notes                                   |
|:-----------------------------------------|:------------:|:----------------------------------------|
| Non-edge port goes to Forwarding         | yes          | Normal RSTP topology change             |
| Non-edge port goes to Discarding         | yes          | Normal RSTP topology change             |
| Edge port goes up                        | no           | Hosts coming up don't change topology   |
| Direct or indirect link failure (RSTP)   | yes          | Detected via BPDU absence or P2P loss   |
| Direct link failure (STP)                | yes          | TCN sent upstream to root               |
| Indirect failure (STP)                   | yes          | Detected via MaxAge expiry              |
| Bridge restart                           | yes          | Multiple TCs as ports come up           |

---

## 11. PVST+ — Per-VLAN Spanning Tree Math

### 11.1 Per-VLAN BPDU Multiplication

PVST+ runs an independent spanning-tree instance per VLAN. For each VLAN, the bridge generates a BPDU on every trunk port at HelloTime intervals.

For a switch with $V$ VLANs and $P$ trunk ports, the BPDU emission rate is:

$$\text{rate}_{\text{BPDU}} = \frac{V \cdot P}{\text{HelloTime}}$$

A core switch with 200 active VLANs and 24 trunk ports emits $\frac{200 \cdot 24}{2} = 2400$ BPDUs per second. With each BPDU at 76 byte-times on a 10 Gb/s link, that's 1.5 Mb/s of pure BPDU traffic — manageable but real.

For an entire fabric with $N$ bridges:

$$\text{aggregate BPDU rate} \;\;\sim\;\; \mathcal{O}(N \cdot V \cdot P)$$

This is why MSTP scales better: with 4 MSTIs covering 200 VLANs, the BPDU rate is $\frac{4 \cdot 24}{2} = 48$ BPDUs/s — a 50x reduction.

### 11.2 PVST+ Trunk Encoding

PVST+ uses two BPDU types:

1. On VLAN 1 (native) the BPDU is sent untagged with destination `01:80:C2:00:00:00` (standard 802.1D format). All bridges (PVST+, RSTP, MSTP, generic 802.1D) receive and process this.
2. On all other VLANs, the BPDU is sent with 802.1Q tagging and destination `01:00:0C:CC:CC:CD` — the Cisco SSTP multicast. Only Cisco bridges process these.

This dual-encoding is why interoperating PVST+ with non-Cisco MSTP works on the native VLAN but fails on tagged VLANs — a frequent source of root-bridge anomalies in mixed-vendor data centres.

### 11.3 Per-VLAN Load Balance

The classic PVST+ design pattern:

```
                        Distribution
                       /            \
      VLANs 10-19 root: D1     VLANs 20-29 root: D2
                       \            /
                        Access
```

Configure D1 with `spanning-tree vlan 10,11,...,19 priority 4096` and D2 with `spanning-tree vlan 20,21,...,29 priority 4096`. Traffic for the first VLAN range goes through D1; the second range through D2. Both uplinks carry traffic instead of one being blocked. This is the cheap-and-cheerful precursor to MLAG / EVPN multihoming.

### 11.4 Cost-of-Redundancy Math

For $N$ VLANs on $B$ bridges in a full-mesh redundant setup, the steady-state BPDU count flowing across the fabric per HelloTime is:

$$C_{\text{BPDU}} = N \cdot B \cdot (B-1)$$

For 100 VLANs on 16 bridges that's $100 \cdot 16 \cdot 15 = 24000$ BPDUs every 2 s. With MSTP and 4 instances it's $4 \cdot 16 \cdot 15 = 960$ — a factor of 25 reduction, the entire reason MSTP exists.

---

## 12. Failure Scenarios — Worked Examples

### 12.1 Single Link Failure with Redundant Uplinks

Topology:

```
        Root R (priority 4096)
        /  \
      cost  cost
        4    4
      /        \
   A          B
      \        /
       \      /
       (alternate, blocked)
```

Bridge A's uplink to R fails. With **STP**:

1. A detects link down (carrier loss, instantaneous).
2. A sends TCN upstream on its remaining port (toward B).
3. A's blocked port to B was running through Listening + Learning + Forwarding state for the alternate path; total time ~30 s on legacy STP, plus MaxAge wait if the failure is indirect (~50 s).

With **RSTP**:

1. A detects link down.
2. A's alternate port (toward B) is already a known root path.
3. RSTP transitions alternate → root in milliseconds via proposal/agreement.
4. Total convergence: 100–500 ms.

### 12.2 Root Failure with Multiple Priority-0 Candidates

Topology with two priority-0 bridges:

```
Root1 (priority 0, MAC ...:01)  ←  current root
Root2 (priority 0, MAC ...:02)  ←  hot standby
S3, S4, ... downstream
```

Root1 fails. Now root election runs again; Root2 wins (lower MAC tiebreak). Convergence:

- Each downstream bridge ages out Root1's BPDUs after MaxAge = 20 s.
- Root2 is sending its own BPDUs claiming root; downstream bridges accept these.
- Root ports re-elect; some bridges may need to flip a port from designated to root.
- RSTP: convergence in 1–6 s.
- STP: convergence in 30–50 s.

### 12.3 BPDU Storm (Race Condition)

Cause: a misconfigured bridge with priority 0 and timers `hello=1 max-age=6 fwd=4` introduced into a stable network where the existing root has `hello=2 max-age=20 fwd=15`.

```
T0   stable: root R (priority 4096, default timers)
T1   rogue X (priority 0, hello=1) plugged in
T2   X claims root, flooding 2x BPDU rate
T3   downstream bridges accept X's BPDUs; root re-elects
T4   meanwhile R is still emitting; both send simultaneously
T5   thrash: bridges flap between R and X as designated, especially on
     intermediate links where BPDUs arrive concurrently
T6   loop windows open during state transitions; broadcast storms
T7   network down until X is removed or root guard takes effect
```

Defense: root guard on every customer/leaf port; BPDU guard on every PortFast port; fixed bridge priorities; conservative aggressive-timer policy.

### 12.4 Asymmetric Fault (One Direction Up, Other Down)

```
S1 ────TX────▶ S2     OK
S1 ◀───RX─x─── S2     fiber broken in receive direction at S1
```

Without loop guard: S1's port stops receiving BPDUs from S2. After MaxAge, S1's blocked port (alternate) ages out, and since S1 doesn't see superior BPDUs, S1 transitions the port to Designated and starts forwarding. But S2 is still sending its BPDUs and forwarding. Both ends now believe they own the segment — perfect loop.

With loop guard: S1 detects BPDU absence on the alternate port and puts it into loop-inconsistent. Loop avoided.

With UDLD aggressive: S1's UDLD packets to S2 can't be acknowledged (since S2's response can't reach S1). UDLD err-disables the port at the physical layer. Loop avoided at L1.

### 12.5 Loop Guard Saving the Day

Topology:

```
          Root R
         /      \
        S1      S2
         \      /
          \    /
           link L (fiber pair, RX broken at S1's end)
           
   S1's port toward S2: alternate, blocked, listening for BPDUs
   S2's port toward S1: designated, forwarding, sending BPDUs
```

Without loop guard, after MaxAge=20 s, S1 ages out the BPDUs and transitions the alternate port to designated and forwarding. Since S2 is also forwarding, traffic that arrives via the root path now loops back through S1 → S2 → root → S1.

With loop guard on S1's port: when BPDUs stop arriving, S1's port enters loop-inconsistent (Discarding). Traffic continues to flow only through the working root path. When the broken fiber is replaced and BPDUs resume, S1's port auto-recovers.

---

## 13. Calculations Practice

### 13.1 Four-Bridge Ring

Topology and parameters:

```
        +--cost 4--+
        |          |
        S1        S2
        |          |
       cost 4    cost 4
        |          |
        S4--cost 4-S3
        
Bridge   Priority   MAC suffix
S1       4096       :01
S2       8192       :02
S3       8192       :03
S4       4096       :04
```

**Step 1: Root election.** Lowest BID wins. S1 and S4 tie on priority (4096); S1 has lower MAC (:01 vs :04). **Root = S1**.

**Step 2: Root paths from each bridge.**

- S2 → S1: direct (cost 4) or via S3,S4 (cost 12). Direct wins. RP = link to S1, RPC = 4.
- S3 → S1: via S2 (cost 8) or via S4 (cost 8). Tie on cost. Tiebreak: lower designated BID. S2 has BID `0x20000200...02` and S4 has BID `0x10000400...04`. S4 wins. RP = link to S4, RPC = 8.
- S4 → S1: direct (cost 4) or via S3,S2 (cost 12). Direct wins. RP = link to S1, RPC = 4.

**Step 3: Designated ports per LAN.** Each link connects two bridges; the bridge with lower RPC owns the designated end:

- Link S1-S2: S1 RPC=0 < S2 RPC=4. Designated = S1.
- Link S2-S3: S2 RPC=4 < S3 RPC=8. Designated = S2.
- Link S3-S4: S3 RPC=8 = S4 RPC+4=8. Tie. Lower BID = S4 wins.
- Link S4-S1: S1 RPC=0 < S4 RPC=4. Designated = S1.

**Step 4: Blocked ports.** At S3, the port toward S2 is *not* root and *not* designated (since S2 is designated for that link). Blocked. At S3, the port toward S4 is the root port. Forwarding.

**Result:** S3's S2-facing port is blocked. The ring is broken there.

### 13.2 Three-Tier Topology with MSTP

```
                  Core1   Core2     (priority 0 and 4096)
                    |       |
                   /         \
              Dist1           Dist2 (priority 8192 and 16384)
              /  \            /  \
           Acc1 Acc2       Acc3 Acc4 (priority default 32768)
```

VLAN-to-instance map:

```
MSTI 0 (IST):  VLANs 1-99, 4001-4094       (default + management)
MSTI 1:        VLANs 100-199               (data)
MSTI 2:        VLANs 200-299               (voice)
MSTI 3:        VLANs 300-399               (server)
```

Per-instance load balance:

- MSTI 0: Core1 root (priority 0). Dist1's left uplink to Core1 is RP; Dist1's right uplink to Core2 blocked.
- MSTI 1: Core1 root (priority 0). Same as MSTI 0 for VLANs 100-199.
- MSTI 2: Core2 root (configured priority 4096 for MSTI 2). Dist1's right uplink to Core2 is RP for VLAN 200-299; left uplink blocked.
- MSTI 3: Core1 root again, same as MSTI 1.

The result is per-instance load balancing without per-VLAN BPDU explosion. Total BPDUs per Dist switch: 4 instances × 2 trunk ports / 2 s = 4 BPDUs/s.

Per-VLAN equivalent (PVST+) would be: 400 VLANs × 2 trunk ports / 2 s = 400 BPDUs/s. **100x reduction.**

### 13.3 Worst-Case Convergence

Topology: a 7-bridge linear chain (maximum legal STP diameter).

```
R - B1 - B2 - B3 - B4 - B5 - B6
```

Indirect failure of the link R-B1. Recovery via an alternate path through a parallel backbone (not shown):

| Phase                                              | Time            |
|:---------------------------------------------------|:----------------|
| MaxAge expires on B1's stale BPDU from R           | 20 s            |
| Listening on alternate port                        | 15 s (FD)       |
| Learning on alternate port                         | 15 s (FD)       |
| **Total worst case**                               | **50 s**        |

For a network with maximum diameter 7 the same 50-s bound applies — the linear chain doesn't make it worse because each downstream bridge is using the same MaxAge value pre-cached in its locally-stored BPDUs. The MaxAge propagation through the topology is implicit.

In RSTP the same scenario:

| Phase                                              | Time            |
|:---------------------------------------------------|:----------------|
| Link-down detection (carrier or BPDU absence)      | 0–6 s           |
| Proposal/agreement on alternate (if P2P)           | 50–200 ms       |
| **Total worst case**                               | **0.5–6 s**     |

The 6 s upper bound is `3 × HelloTime`, the time required to declare a peer dead via missed Hellos when no carrier-loss signal was given.

---

## 14. Operational Show Commands — Multi-Vendor

### 14.1 Cisco IOS / IOS-XE

```bash
# Whole-tree summary
show spanning-tree
show spanning-tree summary
show spanning-tree summary totals       # aggregate counters

# Per-VLAN detail
show spanning-tree vlan 10
show spanning-tree vlan 10 detail

# Per-interface
show spanning-tree interface Gi0/1
show spanning-tree interface Gi0/1 detail

# MSTP region
show spanning-tree mst configuration
show spanning-tree mst configuration digest
show spanning-tree mst 0
show spanning-tree mst 1 detail

# Inconsistencies (root-inconsistent, loop-inconsistent, BPDU-inconsistent)
show spanning-tree inconsistentports

# BPDU statistics
show spanning-tree bridge
show spanning-tree root
show spanning-tree bpdu

# Guards and edge config
show spanning-tree interface Gi0/1 portfast
show spanning-tree interface Gi0/1 detail | include guard

# Errdisable recovery
show errdisable recovery
show errdisable detect
```

### 14.2 Juniper Junos

```bash
# RSTP (default protocol on Junos for new EX/QFX)
show spanning-tree bridge
show spanning-tree interface
show spanning-tree statistics

# MSTP
show spanning-tree mstp configuration
show spanning-tree mstp interface

# Specific instance
show spanning-tree bridge msti 1
show spanning-tree interface msti 1

# Topology
show spanning-tree topology

# BPDU statistics on a port
show spanning-tree statistics interface ge-0/0/1
```

### 14.3 Arista EOS

```bash
# Summary
show spanning-tree
show spanning-tree summary
show spanning-tree blockedports
show spanning-tree root
show spanning-tree bridge

# Per-instance
show spanning-tree mst 0
show spanning-tree mst configuration

# Per-interface
show spanning-tree interface Ethernet1
show spanning-tree interface Ethernet1 detail

# Counters
show spanning-tree counters
show spanning-tree counters detail
```

### 14.4 Diagnostic Recipes

```bash
# Find which bridge is the root for VLAN 10 (Cisco)
show spanning-tree vlan 10 root

# Trace path to root for a given VLAN
show spanning-tree vlan 10 detail | include "Root|Port|Cost"

# List all blocked/discarding ports
show spanning-tree | include BLK
show spanning-tree | include Discarding

# Detect PortFast misconfiguration (no BPDU guard)
show spanning-tree interface | include Edge
show running-config | section spanning-tree

# Look for unexpected TC events
show spanning-tree detail | include "topology change"
show spanning-tree vlan 10 detail | include "occurred|number"
```

A high "number of topology changes" counter on a leaf bridge often pinpoints a flapping access port — usually a duplex mismatch, a flaky cable, or a power-cycling end host.

---

## 15. Math Quick Reference

### 15.1 Convergence Bounds

| Variant | Best | Typical | Worst |
|:--------|-----:|--------:|------:|
| STP     | 30 s | 30 s    | 52 s (max-diameter)  |
| RSTP    | 100 ms | 200 ms| 6 s (3 × HT, indirect failure) |
| MSTP    | 100 ms | 200 ms| 6 s (per instance)             |

### 15.2 BPDU Frame Size

| Variant      | Payload | Total frame on wire | Frame on wire incl. preamble + IFG |
|:-------------|--------:|--------------------:|------------------------------------:|
| 802.1D Conf. | 35 B    | 56 B                | 76 B                                |
| 802.1D TCN   | 4 B     | 25 B                | 45 B                                |
| 802.1w RSTP  | 37 B    | 58 B                | 78 B                                |
| 802.1s MSTP  | 102+B   | 123+B               | 143+B                               |

### 15.3 Default Timers

| Timer        | Default | Range  |
|:-------------|--------:|:-------|
| HelloTime    | 2 s     | 1–10 s |
| MaxAge       | 20 s    | 6–40 s |
| ForwardDelay | 15 s    | 4–30 s |
| TC While     | 4 s     | 2 × HT |
| Hold Time    | 1 s     | fixed  |

### 15.4 Path Cost Long-Form Closed Form

$\text{cost}(R) \approx \dfrac{2 \cdot 10^{10}}{R_{\text{bps}}}$, snapped to the discrete IEEE table.

### 15.5 Maximum Diameter (Default Timers)

$D_{\max} = \left\lfloor \dfrac{\text{MaxAge}}{1 \text{ s/hop}} \right\rfloor = 20$ theoretical, $7$ practical (margins for processing delay).

### 15.6 Inequality Constraint

$2 \cdot (HT + 1) \leq MaxAge \leq 2 \cdot (FD - 1)$

### 15.7 Region-Boundary Digest

$\text{digest} = \text{HMAC-MD5}_{\text{key}}(\text{VLAN-to-instance map})$ where key is the 802.1Q-defined constant.

---

## 16. Common Misconceptions

### 16.1 "BPDU guard and BPDU filter do the same thing"

No. BPDU guard *err-disables on BPDU receipt*; BPDU filter *silently drops* BPDUs in both directions. The latter disables STP for that port — a hub plugged in there causes a silent loop.

### 16.2 "RSTP is always sub-second"

Only on **point-to-point full-duplex** links. Half-duplex or shared media collapses RSTP to STP-style timer-driven convergence.

### 16.3 "Setting low priorities on multiple bridges gives me redundant roots"

It gives you a deterministic primary + secondary, but only one is root at any time — the lower BID wins. The "secondary root" is just the next candidate after a failure.

### 16.4 "PortFast disables BPDU processing"

No. PortFast skips Listening + Learning at port-up. It does not disable BPDU receipt. A BPDU arriving on a PortFast port immediately retracts the edge designation. Only BPDU filter actually disables BPDU processing.

### 16.5 "PVST+ doesn't interoperate with MSTP at all"

It does, but only on the native VLAN (where PVST+ uses standard 802.1D format). Tagged VLANs use Cisco SSTP format and are invisible to MSTP. Mixed-vendor networks must ensure native VLAN matches and that the MSTP CST bridge is the intended root for the entire fabric.

### 16.6 "Loop guard auto-recovers"

Yes, automatically — when BPDUs resume on the affected port. Compare with BPDU guard, which requires admin intervention or err-disable timeout.

### 16.7 "MSTP can have unlimited instances"

64 MSTIs by IEEE 802.1s (plus the IST, which is sometimes called MSTI 0). Cisco implementations often state 65 instances total (IST + 64 MSTIs).

### 16.8 "Bridges with the same priority can both be root"

Only one is root at a time; the lower MAC tiebreak deterministically picks the winner. Ties are resolved at the BID-comparison level, never left ambiguous.

---

## 17. Vendor Implementation Quirks

### 17.1 Cisco PVST+ → Standard MSTP Migration

```
Old:  spanning-tree mode pvst       (or rapid-pvst)
New:  spanning-tree mode mst
      spanning-tree mst configuration
       name CORE
       revision 1
       instance 1 vlan 1-2094
       instance 2 vlan 2095-4094
```

Conversion gotchas:

- The bridge runs both PVST+ and MSTP on the boundary during cutover — a "PVST simulation" feature treats incoming PVST BPDUs as IST BPDUs.
- VLAN 1 (native) must be mapped to MSTI 0 (IST) in most designs; mapping VLAN 1 elsewhere causes BPDU encapsulation issues.
- Migration order: convert the root last to minimize re-elections.

### 17.2 Juniper Default RSTP

Junos defaults to RSTP, not MSTP. To enable MSTP:

```bash
[edit protocols]
delete rstp
set mstp configuration-name CORE
set mstp revision-level 1
set mstp msti 1 vlan 100-199
set mstp interface ge-0/0/1
```

### 17.3 Arista EOS — EVPN Replaces STP

In EVPN-VXLAN data centres, Arista (and most modern data-center vendors) recommend disabling STP on VXLAN-encapsulated traffic. STP only protects the underlay fabric; the overlay is loop-free by VXLAN flood-and-learn semantics with MAC mobility.

```bash
# Underlay: STP disabled or BPDU-blocking on VXLAN-only links
no spanning-tree
# (use MLAG with L3 underlay; STP only on the small server-attach VLANs)
```

### 17.4 Cumulus / SONiC

Open-source NOSes implement RSTP via `mstpd` (Linux STP daemon). MSTP is supported via the same daemon since version 0.0.4 (2010). Configuration is a Linux bridge attribute:

```bash
brctl addbr br0
brctl stp br0 on
ip link set dev br0 up
mstpctl setforcevers br0 rstp     # or mstp
mstpctl settreeprio br0 0 4096
```

---

## 18. Show-and-Tell — Putting It Together

A real campus deployment, two distribution and four access:

```
            +--Core1--+         (priority 4096 for IST; priority 0 for MSTI 1, 4096 for MSTI 2)
            |         |
        +Core2+        \         (priority 8192 for IST; priority 4096 for MSTI 1, 0 for MSTI 2)
        |     |         \
      Dist1  Dist2        \
       |  \   /  |          \
       |   \ /   |           \
      Acc1 Acc2 Acc3 Acc4   (default priority)
```

VLAN map:

| VLAN | Service       | MSTI |
|----:|:--------------|----:|
|   1 | Native/Mgmt   | 0    |
|  10 | User Data     | 1    |
|  20 | Voice         | 2    |
|  30 | Servers       | 1    |
|  40 | Guest WiFi    | 2    |

Configuration on Core1:

```bash
spanning-tree mode mst
spanning-tree mst configuration
 name CAMPUS
 revision 5
 instance 1 vlan 10,30
 instance 2 vlan 20,40
 exit
spanning-tree mst 0 priority 4096
spanning-tree mst 1 priority 0
spanning-tree mst 2 priority 4096

! Edge ports
interface range Gi1/0/1 - 48
 switchport mode access
 switchport access vlan 10
 spanning-tree portfast
 spanning-tree bpduguard enable

! Trunks
interface range Te1/1/1 - 4
 switchport mode trunk
 switchport trunk encapsulation dot1q
 spanning-tree guard root      ! prevent rogue switch from claiming root
```

Verify behavior:

```bash
show spanning-tree mst configuration
show spanning-tree mst 0 detail   # verify Core1 is IST root
show spanning-tree mst 1          # verify Core1 is root for MSTI 1
show spanning-tree mst 2          # verify Core2 is root for MSTI 2
show spanning-tree inconsistentports
```

Expected: Core1 root for MSTI 1, Core2 root for MSTI 2; Dist1's left link forwarding for MSTI 1, blocked for MSTI 2; Dist1's right link blocked for MSTI 1, forwarding for MSTI 2.

---

## 19. Operational Pitfalls

### 19.1 Forgetting Long-Form Path Cost

A bridge with `pathcost method short` mixed with bridges using `long` produces RPC values that aren't directly comparable. The bridge in short mode sees `cost = 4 + 4 = 8` for two 1-Gb hops; a long-mode peer sees `cost = 20 000 + 20 000 = 40 000`. The 8 vs 40 000 RPC compare gets the short-mode bridge picked as root path even when the long-mode path is faster. Fix: globally set `spanning-tree pathcost method long` on every bridge.

### 19.2 Tuning Timers Below Spec Bounds

```bash
spanning-tree vlan 10 hello-time 1
spanning-tree vlan 10 max-age 4         ! INVALID: 2*(1+1)=4 ≤ 4 ≤ 2*(15-1)=28 OK
spanning-tree vlan 10 max-age 3         ! INVALID: 2*(1+1)=4 > 3
```

The IOS validator catches the second case at config time. Some older platforms silently accept and produce subtle bugs (BPDUs aging out before they finish propagating).

### 19.3 PortFast on a Trunk Without `trunk` Keyword

```bash
interface Te1/1/1
 switchport mode trunk
 spanning-tree portfast              ! WARNING: ignored on trunk
 spanning-tree portfast trunk        ! correct keyword
```

The first form is silently ignored on most IOS releases.

### 19.4 BPDU Guard with Voice VLAN Plus PortFast

A voice-VLAN port carries two VLANs (data + voice) but is logically still an access port. PortFast + BPDU guard still applies. A daisy-chained IP phone is not a switch, but some phones do bridge frames internally — verify the phone vendor's docs before assuming the link is BPDU-clean.

### 19.5 Asymmetric MSTP Region

```
Bridge A: name=CORE, revision=1, vlan-map=instance 1 vlan 10-20
Bridge B: name=CORE, revision=1, vlan-map=instance 1 vlan 10-30
```

Different digests → A and B are in different regions even though name/revision match. The boundary appears at the link between them; convergence drops to inter-region (single virtual bridge) semantics. Symptom: an MSTI that should span the whole fabric ends at that link.

### 19.6 BPDU Storm From a Loop in the Underlay

If the Layer-2 underlay loops (e.g. a duplicate cable between two access switches), the BPDU rate spikes because TC propagates and floods on every adjacency. CPU saturation can cascade to dropped BPDUs, which causes more flapping. Symptoms: skyrocketing `show interfaces counters errors` (invalid CRC, runts), high CPU on supervisor, slow CLI response.

Mitigation: fast detect via storm control:

```bash
interface range Gi1/0/1 - 48
 storm-control broadcast level 1.0
 storm-control multicast level 1.0
 storm-control action shutdown
```

### 19.7 Trunk Mismatch Between PVST+ and Standard 802.1D

The trunk only carries SSTP-formatted BPDUs on tagged VLANs. If the peer is a non-Cisco bridge running plain RSTP, those BPDUs are dropped — the peer does not see the per-VLAN topology. Effect: the non-Cisco peer thinks it owns the link as designated, while the Cisco peer thinks it's blocked, and a half-loop forms.

Fix: use MSTP on both sides, or run only the native VLAN on the trunk, or migrate to all-Cisco PVST+ on both ends.

### 19.8 Topology-Change-Storm From Flapping Edge

A flapping access port (every 30 s) can generate a TC every 30 s, which flushes the entire MAC table for that VLAN every 30 s. Result: the network pauses for ~5 s after each TC while MACs re-learn. Symptom: 16% packet loss, 5 s every 30 s, on a campus network.

Diagnose with:

```bash
show spanning-tree vlan 10 detail | include "topology change"
debug spanning-tree events
debug spanning-tree topology-change
```

Find the offending port (the one whose link flap counter increments most), apply BPDU guard if it's edge, or replace the cable.

---

## 20. Algorithm Pseudocode Reference

### 20.1 RSTP Port-Role Selection

```
PORT_ROLE_SELECTION(port p, BPDU best_bpdu):
    if p is admin-disabled:
        p.role <- Disabled
        return

    if best_bpdu received on p:
        if best_bpdu.root < my_root or
           (best_bpdu.root == my_root and best_bpdu.cost < my_cost):
            // p's BPDU is superior
            update my_root, my_cost
            p.role <- Root
        elif best_bpdu.root == my_root and best_bpdu.cost == my_cost and
             best_bpdu.sender_bid < my_bid:
            // p sees a peer with same cost but lower BID
            p.role <- Alternate (blocked)
        else:
            // I have superior BPDU compared to what p sees
            p.role <- Designated

    else:  // no superior peer
        p.role <- Designated

    if p.role == Designated and p.peer == myself (loopback / shared LAN):
        p.role <- Backup
```

### 20.2 BPDU Reception Handler

```
RECEIVE_BPDU(port p, bpdu B):
    if B.protocol_id != 0x0000 or B.version not in {0, 2, 3}:
        log "unknown BPDU"; drop
        return

    if (B.root, B.cost, B.sender_bid, B.sender_port) <
       (p.last_seen.root, p.last_seen.cost, ...):
        // superior BPDU
        if root_guard(p):
            p.state <- root_inconsistent (Discarding)
            return
        update p.last_seen
        recompute roles
        if state changed:
            mark TC, flush MAC table for affected VLAN

    elif equal:
        refresh p.last_seen.expiry  (HelloTime * 3)

    else:
        // inferior BPDU
        if p.role == Designated:
            // peer thinks they should be designated; we override
            send_BPDU(p)

    if B.flags & TC:
        flush_MAC_table(VLAN, age=ForwardDelay)
        propagate_TC()
```

### 20.3 Hello Tx Loop

```
TX_HELLO_LOOP():
    while running:
        for each port p where p.role == Designated and not p.edge:
            send_BPDU(p, role=Designated, root=my_root, cost=my_cost)
        sleep(HelloTime)
```

### 20.4 Aging Loop

```
AGING_LOOP():
    while running:
        for each port p where p.role in {Root, Alternate}:
            if now() - p.last_seen.timestamp > MaxAge:
                p.last_seen <- expired
                if loop_guard(p) and p.role in {Root, Alternate}:
                    p.state <- loop_inconsistent (Discarding)
                else:
                    recompute roles
        sleep(1)
```

---

## 21. ASCII Topology Reference Cards

### 21.1 Core Distribution Access (PVST+ Per-VLAN)

```
                  Core
              priority 4096
                 /    \
               /        \
             /            \
         Dist1            Dist2
       priority 8192    priority 16384
        /    \           /     \
       /      \         /       \
      Acc1   Acc2      Acc3    Acc4
        (default, 32768)
        
   Per-VLAN balancing:
     Dist1 root for VLAN 10-19 (priority 4096 for those)
     Dist2 root for VLAN 20-29 (priority 4096 for those)
```

### 21.2 MSTP Region

```
   +---------+      +---------+
   | BridgeA | -----| BridgeB |
   |MSTI 1: P| BPDU |MSTI 1: P|
   |MSTI 2: B|<---->|MSTI 2: P|
   +---------+      +---------+
        |               |
        | (MSTI BPDUs   |
        |  carry P/B    |
        |  per instance)|
        v               v
   +---------+      +---------+
   | BridgeC |      | BridgeD |
   |MSTI 1: B|      |MSTI 1: P|
   |MSTI 2: P|      |MSTI 2: B|
   +---------+      +---------+
   
   (P = Primary/Forwarding, B = Backup/Discarding)
```

### 21.3 Loop Guard Saving Asymmetric Fault

```
  Root R
  /    \
 |      |
 S1     S2
 |      |
 +-LINK-+
 
 LINK fails in receive direction at S1:
 
 Without loop guard:
   T0:  S1 receives BPDU on LINK from S2 (port = Alternate)
   T0+MaxAge:  BPDU ages out, S1 transitions to Designated, Forwarding
   T0+MaxAge+:  Loop! Traffic from R via S1->LINK->S2->R loops
 
 With loop guard:
   T0+MaxAge:  S1 puts LINK port into loop-inconsistent (Discarding)
   T_recovery:  When BPDU resumes, S1 auto-recovers
```

### 21.4 BPDU Guard Catching a Hub

```
  S1  ---  S2  --- (hub) --- (hub) --- (loop!)
              |
              | PortFast + BPDU guard
              |
              | Hub forwards BPDUs from S1 -> hub -> back to S2
              |
              | S2's port receives a BPDU that S2 itself originated
              |
              | BPDU guard fires -> port err-disabled
              | -> Hub disconnected; loop avoided
```

---

## 22. Glossary

| Term                  | Definition                                                                                          |
|:----------------------|:----------------------------------------------------------------------------------------------------|
| BID                   | Bridge ID. 8 bytes: 2-byte priority + 6-byte MAC. Lowest wins root election.                        |
| BPDU                  | Bridge Protocol Data Unit. The control frame STP/RSTP/MSTP exchange.                                |
| Bridge                | Synonym for Layer-2 switch in IEEE terminology.                                                     |
| CIST                  | Common and Internal Spanning Tree. The MSTP "outer" tree spanning all regions.                       |
| Designated Bridge     | The bridge with the lowest RPC on a given LAN; owns forwarding for that LAN.                         |
| Designated Port       | The port on the designated bridge that connects to a specific LAN.                                  |
| Edge Port             | A port connected to a host (no BPDU expected). Synonym: PortFast.                                   |
| Forward Delay (FD)    | Time spent in Listening and in Learning. Default 15 s.                                              |
| Hello Time (HT)       | Interval between BPDU transmissions on designated ports. Default 2 s.                               |
| Inferior BPDU         | A BPDU that loses to the locally-stored best BPDU. Triggers Designated role for the receiving port. |
| IST                   | Internal Spanning Tree. The CIST inside one MSTP region; equivalent to MSTI 0.                       |
| MaxAge                | Maximum time a BPDU is considered valid. Default 20 s.                                              |
| MSTI                  | Multiple Spanning Tree Instance. Up to 64 (or 65 with IST) per region.                              |
| MSTP                  | Multiple Spanning Tree Protocol. IEEE 802.1s, now part of 802.1Q.                                   |
| PortFast              | Cisco term for IEEE Edge Port. Skip Listening and Learning at port-up.                              |
| Proposal/Agreement    | RSTP handshake to transition a port from Discarding to Forwarding sub-second on P2P links.          |
| PVST+                 | Per-VLAN Spanning Tree Plus. Cisco proprietary; one tree per VLAN.                                  |
| Rapid PVST+           | RSTP per VLAN. Same as PVST+ but with sub-second convergence.                                       |
| Region                | MSTP scope defined by name + revision + VLAN-to-instance digest.                                    |
| Root Bridge           | The bridge with the lowest BID. The center of the spanning tree.                                    |
| Root Path Cost (RPC)  | Sum of port costs along the path to the root.                                                       |
| Root Port (RP)        | The port on a non-root bridge with the lowest RPC; points toward the root.                          |
| Root Guard            | Feature that blocks superior BPDUs from being accepted on a port.                                   |
| RSTP                  | Rapid Spanning Tree Protocol. IEEE 802.1w, now in 802.1D-2004.                                       |
| Superior BPDU         | A BPDU that beats the locally-stored best BPDU. Causes role recomputation.                          |
| TCN                   | Topology Change Notification. A 4-byte BPDU sent upstream when a topology change is detected.        |
| TC While              | RSTP timer; how long to flood TC=1 in BPDUs after a topology change. Default `2 × HelloTime`.        |
| UDLD                  | UniDirectional Link Detection. Cisco protocol for detecting one-way fiber failures.                 |
| Version 1 BPDU        | The 35-byte 802.1D Configuration BPDU.                                                               |
| Version 2 BPDU        | The 37-byte 802.1w RSTP BPDU.                                                                        |
| Version 3 BPDU        | The variable-length 802.1s MSTP BPDU.                                                                |

---

## See Also

- `networking/stp` — operational cheatsheet (configuration, common commands, troubleshooting decision tree)
- `networking/vlan` — VLAN basics and 802.1Q tagging (essential context for PVST+ and MSTP)
- `networking/ethernet` — frame format, FCS, half/full duplex (foundation for understanding BPDU encoding)
- `ramp-up/spanning-tree-eli5` — ELI5-voiced narrative companion that walks through the same topics conversationally
- `ramp-up/spine-leaf-eli5` — modern data-center fabric design (where STP is replaced by routed underlay + EVPN overlay)

## References

- IEEE Std 802.1D-2004 — *Media Access Control (MAC) Bridges* (incorporates 802.1w)
- IEEE Std 802.1Q-2018 — *Bridges and Bridged Networks* (incorporates 802.1s MSTP)
- IEEE Std 802.1w-2001 — *Rapid Reconfiguration of Spanning Tree* (now superseded by 802.1D-2004)
- IEEE Std 802.1s-2002 — *Multiple Spanning Trees* (now superseded by 802.1Q)
- IEEE Std 802.1t-2001 — *MAC Bridges Amendment 1: Technical and Editorial Corrections* (defines long-form path cost)
- Radia Perlman, *Interconnections: Bridges, Routers, Switches and Internetworking Protocols* (2nd ed., Addison-Wesley, 2000) — original STP design rationale by the inventor
- Cisco *Spanning Tree Protocol Design Guide* — Document ID 24062 (`https://www.cisco.com/c/en/us/support/docs/lan-switching/spanning-tree-protocol/`)
- Cisco *Understanding Rapid Spanning Tree Protocol (802.1w)* — Document ID 24062
- Cisco *Understanding Multiple Spanning Tree Protocol (802.1s)* — Document ID 24248
- Cisco *Loop Guard and BPDU Skew Detection Feature* — Document ID 10596
- Cisco *Spanning Tree PortFast BPDU Guard Enhancement* — Document ID 10586
- Juniper *Junos Software Routing Protocols Configuration Guide — Spanning Tree Protocols*
- Arista *EOS Configuration Guide — Spanning Tree*
- BICSI ITSIMM (Information Technology Systems Installation Methods Manual), Ch. 8 — Network Cabling Reference for STP-Compliant Topologies
- BICSI TDMM (Telecommunications Distribution Methods Manual), 14th ed., Ch. 7 — LAN Architecture and STP/RSTP Considerations
- RFC 6325 — *Routing Bridges (RBridges): Base Protocol Specification* (TRILL — STP's L2 replacement attempt)
- RFC 7432 — *BGP MPLS-Based Ethernet VPN* (EVPN; the modern data-center alternative to STP-based fabric)
- IETF RFC 5556 — *Transparent Interconnection of Lots of Links (TRILL): Problem and Applicability Statement* (motivation for moving past STP)
