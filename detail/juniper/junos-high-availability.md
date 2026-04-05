# JunOS High Availability — Architecture, State Replication, and Convergence Analysis

> *JunOS HA operates at multiple layers: kernel-level state replication (GRES), protocol daemon mirroring (NSR), hardware-assisted failure detection (BFD), and multi-chassis redundancy (MC-LAG, Virtual Chassis). Understanding the replication mechanics and convergence characteristics of each mechanism is essential for building networks that maintain forwarding continuity during planned and unplanned outages.*

---

## 1. JunOS HA Architecture

### RE/PFE Separation as HA Foundation

The fundamental architectural advantage of JunOS for HA is the strict separation between the Routing Engine (RE) and the Packet Forwarding Engine (PFE):

```
┌─────────────────────────────────────────────────┐
│                 Routing Engine (RE)              │
│  ┌─────────┐  ┌─────────┐  ┌──────────────┐    │
│  │   rpd   │  │  chassisd│  │  kernel (RE)  │   │
│  │(routing)│  │(chassis) │  │  (FIB, IFL)   │   │
│  └────┬────┘  └────┬────┘  └──────┬───────┘    │
│       │            │              │             │
│       └────────────┴──────────────┘             │
│                     │ internal link              │
│  ┌──────────────────┴───────────────────────┐   │
│  │         Packet Forwarding Engine (PFE)    │   │
│  │  ┌──────────┐  ┌──────────┐              │   │
│  │  │ Memory   │  │ Memory   │  (ASICs)     │   │
│  │  │ (FIB)    │  │ (filters)│              │   │
│  │  └──────────┘  └──────────┘              │   │
│  └──────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

Because the PFE operates independently with its own copy of the forwarding table, an RE restart or switchover does not immediately impact packet forwarding. The PFE continues forwarding with the last known good FIB until the RE recovers and pushes updated state.

### HA Mechanisms Hierarchy

```
Level 0: No HA
  RE failure → forwarding stops, protocols reset
  Recovery: full reboot, full convergence

Level 1: GRES (Graceful Routing Engine Switchover)
  RE failure → backup RE takes over with kernel state intact
  PFE forwarding continues (nonstop forwarding)
  Protocols restart, peers must support graceful restart
  Recovery: protocol reconvergence (seconds to minutes)

Level 2: NSR (Nonstop Active Routing)
  RE failure → backup RE takes over with full protocol state
  PFE forwarding continues
  Protocol sessions maintained — peers see no disruption
  Recovery: transparent (sub-second)

Level 3: NSR + ISSU
  Planned upgrade → software upgraded without traffic loss
  Both REs upgraded sequentially with state preservation
  Recovery: not needed (no outage)
```

---

## 2. GRES Kernel State Replication

### What Gets Replicated

GRES replicates the following state from master RE to backup RE:

```
Replicated (kernel state):              NOT replicated (daemon state):
├── Interface configuration (IFLs)      ├── rpd routing protocol state
├── Forwarding table (FIB)              ├── BGP sessions, FSM state
├── Firewall filter state               ├── OSPF adjacencies, LSA database
├── CoS configuration                   ├── IS-IS adjacencies, LSDB
├── Policer state                       ├── LDP sessions, bindings
├── MPLS label bindings (kernel)        ├── RSVP sessions, path state
├── Chassis state                       ├── BFD sessions
├── Interface link state                ├── VRRP state
└── ARP/NDP cache                       └── PIM state
```

### GRES Switchover Timeline

```
Time 0:     Master RE fails (hardware fault, kernel panic, manual switch)
            ↓
Time +0ms:  PFE detects RE loss via internal keepalive
            PFE continues forwarding with existing FIB (nonstop forwarding enabled)
            ↓
Time +100ms: Backup RE detects mastership change
             Kernel state already replicated — interfaces, FIB intact
             ↓
Time +1-5s: Backup RE assumes mastership
            chassisd stabilizes
            rpd starts from scratch
            ↓
Time +5-30s: rpd re-reads configuration
             Routing protocols re-initialize
             BGP: sends graceful restart notification to peers
             OSPF: enters GR helper mode on peers
             IS-IS: enters restart mode
             ↓
Time +30-120s: Protocols reconverge
               BGP peers hold routes during restart-time window
               OSPF neighbors maintain adjacency during grace-period
               FIB updated with fresh routes
               ↓
Time +120-300s: Full convergence complete
                All routing protocols stable
```

### GRES Dependency on Graceful Restart

GRES alone relies on routing peers supporting RFC 4724 (BGP GR), RFC 3623 (OSPF GR), or RFC 5306 (IS-IS restart). If peers do not support GR helper mode, they will:

1. Detect the TCP/protocol session drop
2. Withdraw all routes learned from the restarting router
3. This causes traffic black-holing until protocols fully reconverge

**This is why NSR is strongly preferred over GRES alone in SP environments.**

---

## 3. NSR Protocol Daemon Replication

### rpd Mirroring Architecture

NSR runs a shadow copy of rpd on the backup RE that mirrors the master's protocol state in real-time:

```
Master RE                              Backup RE
┌─────────────────┐                   ┌─────────────────┐
│ rpd (active)    │    state sync     │ rpd (standby)   │
│ ┌─────────────┐ │ ──────────────→   │ ┌─────────────┐ │
│ │ BGP FSM     │ │                   │ │ BGP FSM     │ │
│ │ OSPF LSDB   │ │    incremental    │ │ OSPF LSDB   │ │
│ │ IS-IS LSDB  │ │    updates        │ │ IS-IS LSDB  │ │
│ │ LDP bindings│ │ ──────────────→   │ │ LDP bindings│ │
│ │ Routing tbl │ │                   │ │ Routing tbl │ │
│ └─────────────┘ │                   │ └─────────────┘ │
│                 │                   │                 │
│ TCP sessions    │   NOT replicated  │ TCP sessions    │
│ (BGP, LDP)      │   (re-created     │ (pre-staged)    │
│                 │    after switch)  │                 │
└─────────────────┘                   └─────────────────┘
```

### What NSR Replicates

```
Protocol state replicated via NSR:
├── BGP: neighbor state, received routes, best path, communities, AS paths
├── OSPF: adjacency state, LSDB, SPF results
├── IS-IS: adjacency state, LSDB, SPF results
├── LDP: session state, label bindings, FEC-label mappings
├── RSVP: session state, path/resv messages, ERO/RRO
├── BFD: session parameters (sessions re-established quickly)
└── Static routes: fully replicated via kernel

NOT replicated (re-created from replicated state):
├── TCP socket state (new TCP connection established)
├── Protocol keepalive timers (reset)
└── Transient protocol events (pending LSA flood, etc.)
```

### NSR Switchover Timeline

```
Time 0:      Master RE fails
             ↓
Time +0ms:   PFE continues forwarding (same as GRES)
             ↓
Time +50ms:  Backup RE assumes mastership
             Standby rpd promoted to active
             ↓
Time +100ms: rpd has full routing state — no protocol restart needed
             BGP: backup RE sends keepalive on existing sessions
                  (TCP re-establishment is transparent to BGP FSM)
             OSPF: backup RE continues as if no change
             IS-IS: backup RE continues sending/receiving hellos
             ↓
Time +1-3s:  Protocols fully stable
             No routing flaps visible to peers
             No route withdrawals
             No reconvergence
```

### NSR vs GRES Convergence Comparison

| Metric | GRES Only | GRES + NSR |
|:---|:---|:---|
| Forwarding continuity | Yes | Yes |
| Protocol session disruption | Yes (sessions reset) | No |
| Peer awareness of switchover | Yes (GR notification) | No |
| Route withdrawal risk | Yes (if peers lack GR support) | No |
| Convergence time | 30-300 seconds | 1-3 seconds |
| CPU/memory overhead on backup RE | Low | Moderate (standby rpd) |
| Dependency on peer GR support | Critical | None |

---

## 4. Unified ISSU Process

### ISSU State Machine

```
Phase 1: Validation
  ├── Check dual RE presence
  ├── Verify GRES/NSR operational
  ├── Validate software compatibility
  └── Confirm backup RE synchronized

Phase 2: Backup RE Upgrade
  ├── Copy new image to backup RE
  ├── Install and boot backup RE with new software
  ├── Backup RE runs new JunOS
  └── Re-establish state replication (master → backup)

Phase 3: Switchover
  ├── GRES/NSR switchover to backup RE (now master, new software)
  ├── PFE continues forwarding
  ├── Protocols maintained via NSR
  └── Traffic uninterrupted

Phase 4: Old Master Upgrade
  ├── Old master RE upgraded to new software
  ├── Old master RE rebooted
  ├── Comes up as backup RE
  └── State replication re-established (new master → backup)

Phase 5: Complete
  ├── Both REs running new software
  ├── Full redundancy restored
  └── ISSU complete
```

### ISSU Compatibility

Not all software versions support ISSU between them. The key rules:

1. Same major release family (e.g., 21.4R1 to 21.4R3 — compatible)
2. Adjacent major releases may be compatible (check release notes)
3. Skip-release ISSU generally not supported
4. Feature-level incompatibilities may exist even within compatible releases

### ISSU Failure Modes

```
Failure during Phase 2 (backup upgrade):
  → Master RE unaffected, abort ISSU, backup RE rebooted to old software
  → No traffic impact

Failure during Phase 3 (switchover):
  → Automatic rollback to old master RE
  → Brief forwarding disruption possible
  → Worst case: manual recovery with console

Failure during Phase 4 (old master upgrade):
  → New master operational on new software
  → Old master may need manual recovery
  → No traffic impact (single RE operation until fixed)
```

---

## 5. MC-LAG and ICCP Protocol

### ICCP Architecture

Inter-Chassis Control Protocol (RFC 7275) provides the control plane for MC-LAG:

```
Device A                                    Device B
┌────────────────────┐                     ┌────────────────────┐
│  ICCP daemon       │ ←─── TCP session ──→│  ICCP daemon       │
│  ├── State sync    │     (liveness +     │  ├── State sync    │
│  │   MAC table     │      state exchange)│  │   MAC table     │
│  │   ARP cache     │                     │  │   ARP cache     │
│  │   LACP state    │                     │  │   LACP state    │
│  │   STP state     │                     │  │   STP state     │
│  └── Arbitration   │                     │  └── Arbitration   │
│      (active/stby) │                     │      (active/stby) │
│                    │                     │                    │
│  ae0 ────────────── ICL ────────────────── ae0               │
│  ge-0/0/0 ─────┐                    ┌──── ge-0/0/0           │
│  ge-0/0/1 ─────┤   Downstream       ├──── ge-0/0/1           │
└────────────────┘│   Device           │└────────────────────┘
                  └────── ae0 ─────────┘
```

### ICCP Message Types

```
1. Connect/Disconnect: Session establishment/teardown
2. Application Data:
   ├── MAC sync: MAC addresses learned on one chassis replicated to other
   ├── ARP sync: ARP entries replicated
   ├── LACP sync: LACP system-id, port keys, state
   └── STP sync: STP root, port roles, topology changes
3. Liveness: BFD-style keepalives between chassis
4. Notification: State changes (link up/down, failover)
```

### MC-LAG Failure Scenarios

```
Scenario 1: Member link failure on Device A
  → LACP detects link down
  → Device A removes link from LAG
  → Traffic shifts to Device B member links
  → ICCP notifies Device B
  → No traffic loss (LACP handles redistribution)

Scenario 2: All member links fail on Device A
  → Device A ae0 goes down
  → ICCP notifies Device B
  → Device B becomes sole active for all traffic
  → ICL carries inter-chassis traffic if needed
  → Downstream device uses Device B links only

Scenario 3: ICCP session fails (split-brain)
  → Both devices think they are active
  → active-active mode: both continue forwarding (designed for this)
  → active-standby mode: standby device brings down its ae0 member links
  → Downstream device converges to single chassis

Scenario 4: Device A complete failure
  → ICCP session drops
  → Device B detects via BFD/liveness
  → Device B takes full ownership
  → MAC/ARP already replicated — minimal convergence
```

### Active-Active vs Active-Standby

```
Active-Active:
  ├── Both chassis forward traffic simultaneously
  ├── Load sharing across all member links
  ├── ICL carries inter-chassis traffic for unknown unicast / BUM
  ├── Better bandwidth utilization
  └── More complex failure handling

Active-Standby:
  ├── Only active chassis forwards traffic
  ├── Standby chassis member links in standby (LACP standby)
  ├── Simpler failure model
  ├── Wastes standby bandwidth
  └── ICL carries all traffic during failover
```

---

## 6. BFD Implementation

### BFD Session Establishment

```
1. Routing protocol configured with BFD
2. Protocol adjacency established (BGP, OSPF, IS-IS)
3. BFD session initiated (independent UDP session, port 3784)
4. Negotiation: each side advertises desired min TX/RX interval
5. Agreed interval = max(local desired, remote desired)
6. Session enters UP state
7. Periodic echo packets exchanged at agreed interval
```

### BFD Detection Time Calculation

$$T_{detect} = \text{agreed-interval} \times \text{multiplier}$$

Example: Both sides configure minimum-interval 300ms, multiplier 3:

```
Agreed interval: max(300, 300) = 300ms
Detection time: 300 * 3 = 900ms

BFD packets: every 300ms ±25% jitter
Miss 3 consecutive: declare DOWN after 900ms
```

### BFD on Hardware vs Software

```
Software-based BFD (RE):
  ├── Minimum interval: typically 50-100ms
  ├── Processed by CPU
  ├── Subject to RE CPU load, scheduling jitter
  ├── Suitable for: 100ms+ detection times
  └── Risk: CPU spike → false positive BFD timeout

Hardware-assisted BFD (PFE/line card):
  ├── Minimum interval: down to 10ms on some platforms
  ├── Processed in ASIC/NPU
  ├── Independent of RE CPU load
  ├── Suitable for: sub-100ms detection times
  └── Platform dependent (MX with Memory, PTX)
```

### Micro-BFD for LAG

Standard LACP detects member link failures at the LACP PDU rate:

```
LACP slow: 30-second PDU interval → detection in 90 seconds (3 missed)
LACP fast: 1-second PDU interval → detection in 3 seconds (3 missed)

Micro-BFD: per-member-link BFD session
  300ms interval, multiplier 3 → detection in 900ms
  50ms interval, multiplier 3 → detection in 150ms
```

Micro-BFD runs independent BFD sessions on each member link of the LAG, detecting failures much faster than LACP can.

---

## 7. HA Convergence Analysis

### Failure Detection Times

| Mechanism | Detection Time | Notes |
|:---|:---|:---|
| Physical link failure | < 50ms | LOL/LOS at optical/electrical layer |
| LACP fast | 3 seconds | 3 missed 1-second PDUs |
| LACP slow | 90 seconds | 3 missed 30-second PDUs |
| BFD (aggressive) | 150ms | 50ms * 3 |
| BFD (standard) | 900ms-3s | 300ms-1s * 3 |
| BGP holdtime | 90 seconds (default) | 3 missed 30-second keepalives |
| OSPF dead interval | 40 seconds (default) | 4 * 10-second hello |
| IS-IS hold time | 30 seconds (default) | 3 * 10-second hello |
| GRES switchover | 1-5 seconds | Kernel state, no protocol |
| NSR switchover | 50-100ms | Full protocol state preserved |

### Total Convergence Time (End-to-End)

```
Total convergence = Detection + Notification + Computation + FIB update

Scenario: Link failure with BFD + OSPF
  Detection:     300ms (BFD)
  Notification:  < 100ms (BFD → OSPF, instant)
  Computation:   < 100ms (SPF on modern hardware)
  FIB update:    < 200ms (PFE programming)
  Total:         < 700ms

Scenario: RE failure with GRES (no NSR)
  Detection:     < 1s (internal keepalive)
  Switchover:    1-5s (backup RE assumes master)
  Protocol restart: 5-30s (rpd initialization)
  Reconvergence: 30-120s (GR completion)
  Total:         ~60-150 seconds

Scenario: RE failure with NSR
  Detection:     < 1s (internal keepalive)
  Switchover:    50-100ms (standby rpd promotion)
  Protocol:      0s (sessions maintained)
  FIB update:    0s (FIB unchanged)
  Total:         < 2 seconds
```

### Five-Nines Design

99.999% availability = 5.26 minutes downtime per year. To achieve this:

```
Required:
  ├── Dual RE with NSR (RE failure → <2s outage)
  ├── BFD on all routing adjacencies (<1s link failure detection)
  ├── Micro-BFD on all LAGs (<1s member link failure)
  ├── ECMP or standby paths (no single path dependency)
  ├── MC-LAG or dual-homing (no single device dependency)
  └── ISSU for planned maintenance (0s outage for upgrades)

Budget:
  Planned maintenance: 0 minutes (ISSU)
  Unplanned RE failure: 2s * estimated 2/year = 4 seconds
  Unplanned link failure: 1s * estimated 10/year = 10 seconds
  Total: ~14 seconds/year (well within 5.26 minutes)
```

## Prerequisites

- JunOS RE/PFE architecture, routing protocol fundamentals, MPLS basics, LAG/LACP concepts, TCP connection mechanics

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| GRES kernel state replication | O(state_size) | O(state_size) on backup RE |
| NSR rpd state sync (incremental) | O(delta) | O(routing_table) on backup RE |
| BFD session maintenance | O(1) per interval | O(sessions) |
| ICCP state synchronization | O(delta) | O(MAC_table + ARP_table) |
| ISSU full process | O(hours) wall-clock | O(image_size) per RE |

---

*High availability is not a feature you enable — it is an architecture you design. GRES provides the foundation, NSR eliminates protocol disruption, BFD accelerates failure detection, and MC-LAG removes single-device dependencies. Each layer addresses a different failure domain, and all must work together to achieve the sub-second convergence that SP networks demand.*
