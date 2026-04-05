# JunOS Security HA — Cluster Architecture, RG Election, Session Sync Protocol, and Convergence Analysis

> *SRX chassis clustering is a stateful high-availability mechanism where two nodes operate as a single logical device. The cluster relies on dedicated fabric links for heartbeat exchange and session synchronization, redundancy group election for active/standby determination, and real-time object replication for seamless failover. Understanding the internal architecture — from split-brain prevention to session sync timing — separates a stable production deployment from one that fails unpredictably under stress.*

---

## 1. Chassis Cluster Internal Architecture

### Component Architecture

The SRX chassis cluster is managed by the JSRP daemon (Junos Services Redundancy Protocol), which runs on the Routing Engine of each node:

```
Node 0 (Primary)                         Node 1 (Secondary)
┌─────────────────────────────────┐     ┌─────────────────────────────────┐
│  Routing Engine (RE)            │     │  Routing Engine (RE)            │
│  ┌───────────────────────────┐  │     │  ┌───────────────────────────┐  │
│  │  JSRP daemon              │◄─┼─────┼─►│  JSRP daemon              │  │
│  │  ├─ RG election           │  │ fxp1│  │  ├─ RG election           │  │
│  │  ├─ Heartbeat tx/rx       │  │     │  │  ├─ Heartbeat tx/rx       │  │
│  │  ├─ Config sync           │  │     │  │  ├─ Config sync           │  │
│  │  └─ Failover orchestration│  │     │  │  └─ Failover orchestration│  │
│  └───────────────────────────┘  │     │  └───────────────────────────┘  │
│                                 │     │                                 │
│  Packet Forwarding Engine (PFE) │     │  Packet Forwarding Engine (PFE) │
│  ┌───────────────────────────┐  │     │  ┌───────────────────────────┐  │
│  │  Flow module              │  │     │  │  Flow module              │  │
│  │  ├─ Session table         │◄─┼─────┼─►│  Session table (replica)  │  │
│  │  ├─ NAT state             │  │ fab │  │  ├─ NAT state (replica)   │  │
│  │  ├─ IDP state             │  │     │  │  ├─ IDP state (replica)   │  │
│  │  └─ Screen counters       │  │     │  │  └─ Screen counters       │  │
│  └───────────────────────────┘  │     │  └───────────────────────────┘  │
└─────────────────────────────────┘     └─────────────────────────────────┘
```

### JSRP Daemon Responsibilities

The JSRP daemon handles:

1. **Heartbeat exchange**: Periodic messages over fxp1 to verify peer liveness
2. **RG election**: Priority comparison and failover decision for each RG
3. **Configuration synchronization**: Primary RE pushes committed config to secondary
4. **Interface monitoring**: Tracks link state of monitored interfaces per RG
5. **IP monitoring**: Active probing of configured IP addresses
6. **Failover orchestration**: Coordinates RG state transitions, reth MAC updates, GARP

### Data Path During Normal Operation

When a packet arrives on a physical interface that is a reth member:

```
Case 1: Packet arrives on ACTIVE reth member (on primary node for that RG)
  └→ Normal flow processing (session lookup → policy → NAT → forward)

Case 2: Packet arrives on STANDBY reth member (on secondary node)
  └→ Packet forwarded over fabric (fab) link to primary node
     └→ Primary node processes the packet
     └→ Primary node forwards out the appropriate egress interface
     └→ If egress interface is on secondary node, packet crosses fabric again

  Note: Double fabric traversal adds latency and consumes fabric bandwidth
  This is why active/active requires careful RG-to-interface planning
```

---

## 2. RG Election Algorithm

### Priority-Based Election

Each RG independently elects a primary node. The election follows a deterministic algorithm:

```
RG Election Algorithm:
  Input: priority_node0, priority_node1, preempt_enabled, current_state

  Phase 1: Initial election (cluster boot)
    if priority_node0 > priority_node1:
      node0 = PRIMARY
    elif priority_node1 > priority_node0:
      node1 = PRIMARY
    else:  # equal priority
      node0 = PRIMARY  # lower node-id wins tie

  Phase 2: Failover trigger
    for each monitored_item in RG:
      if item.state == DOWN:
        effective_priority[item.node] -= item.weight

    if effective_priority[current_primary] < effective_priority[current_secondary]:
      FAILOVER to secondary
    elif effective_priority[current_primary] <= 0:
      FAILOVER to secondary (even if secondary priority is also low)

  Phase 3: Preemption (if enabled)
    if preempt_enabled AND original_primary != current_primary:
      if effective_priority[original_primary] > effective_priority[current_primary]:
        wait(preempt_delay)
        FAILOVER back to original_primary
```

### Effective Priority Calculation

```
effective_priority = configured_priority - SUM(weight of failed monitors)

Example:
  Node 0 configured priority: 200
  Interface ge-0/0/1 (weight 128): DOWN
  Interface ge-0/0/2 (weight 64):  UP
  IP monitor 10.0.0.1 (weight 50): DOWN

  effective_priority_node0 = 200 - 128 - 50 = 22

  Node 1 configured priority: 100
  All monitors UP

  effective_priority_node1 = 100

  22 < 100 → RG fails over to Node 1
```

### Failover Decision Matrix

| Node 0 Effective | Node 1 Effective | Current Primary | Action |
|:---:|:---:|:---:|:---|
| 200 | 100 | Node 0 | No action (stable) |
| 72 | 100 | Node 0 | Failover to Node 1 |
| 72 | 100 | Node 1 | Stay on Node 1 (even with preempt, Node 1 is higher) |
| 200 | 100 | Node 1 | Preempt to Node 0 (if preempt enabled + delay passed) |
| 0 | 100 | Node 0 | Failover to Node 1 |
| 0 | 0 | Node 0 | Stay on Node 0 (both failed, no valid target) |

### RG0 vs RG1+ Election Differences

| Aspect | RG0 | RG1+ |
|:---|:---|:---|
| What it controls | Routing Engine (control plane) | Data plane (reth interfaces) |
| Preemption | Not supported | Supported |
| Manual failover | Supported | Supported |
| Interface monitoring | Not applicable | Supported |
| IP monitoring | Not applicable | Supported |
| Failover impact | CLI session lost, routing restarts | Traffic reroutes, sessions preserved |
| Independent failover | Yes | Yes (per RG) |

---

## 3. Session Synchronization Protocol

### RTO Architecture

Real-Time Objects (RTOs) are the synchronization primitives used to replicate stateful data between nodes:

```
RTO Types:
  ┌─────────────────────────────────────────────────────┐
  │  Type           │  Content                          │
  │─────────────────┼───────────────────────────────────│
  │  Flow session   │  5-tuple, state, timers, bytes    │
  │  NAT binding    │  Original → translated mapping    │
  │  Persistent NAT │  IP:port binding table entry      │
  │  IPsec SA       │  SPI, keys, sequence numbers      │
  │  IDP session    │  Protocol state machine state     │
  │  ALG pinhole    │  Predicted session parameters     │
  │  Screen counter │  Connection rate, SYN counts      │
  └─────────────────────────────────────────────────────┘
```

### Synchronization Flow

```
Active Node                             Standby Node
    │                                       │
    │  1. New session created               │
    │  ┌──────────────────┐                 │
    │  │ Flow module      │                 │
    │  │ creates session  │                 │
    │  └────────┬─────────┘                 │
    │           │                           │
    │  2. RTO generated                     │
    │  ┌────────▼─────────┐                 │
    │  │ RTO: session     │                 │
    │  │ 5-tuple + state  │                 │
    │  └────────┬─────────┘                 │
    │           │                           │
    │  3. RTO sent over fabric              │
    │           │        ┌──────────────┐   │
    │           └───────►│ RTO receiver │   │
    │                    │ applies to   │   │
    │                    │ session table│   │
    │                    └──────────────┘   │
    │                                       │
    │  4. Session modification              │
    │     (counter update, state change)    │
    │     → Incremental RTO sync            │
    │                                       │
    │  5. Session close                     │
    │     → Delete RTO sync                 │
```

### Synchronization Timing and Guarantees

```
RTO delivery characteristics:
  - Protocol: proprietary over fabric link (Layer 2)
  - Ordering: in-order delivery per RTO type
  - Reliability: reliable delivery with acknowledgment
  - Latency: typically < 10ms for RTO to be applied on standby
  - Batching: RTOs are batched for efficiency (up to 50ms batch window)

Synchronization window (potential loss on failover):
  - Best case: 0 sessions lost (failover during idle period)
  - Typical case: < 100ms worth of new sessions lost
  - Worst case: batch window (50ms) + fabric latency + processing time
                 ≈ 100-200ms of sessions potentially lost

  Sessions created in this window:
    - Not present on new active node
    - Client/server must re-establish (TCP retransmit triggers new session)
    - For UDP: application-level retry needed
```

### What Survives Failover

| State | Preserved? | Notes |
|:---|:---|:---|
| TCP sessions (established) | Yes | Full state synced, continues seamlessly |
| TCP sessions (SYN_SENT) | Partial | May need retransmit if in sync window |
| UDP sessions | Yes | If session existed before failover |
| IPsec tunnels | Yes | SA state synced, no renegotiation |
| NAT translations | Yes | Active mappings survive |
| Persistent NAT bindings | Yes | External hosts can still reach internal |
| IDP sessions | Yes | Protocol state preserved |
| ALG pinholes | Yes | Data channel pinholes survive |
| Security policy counters | No | Reset on new active node |
| Traceoptions | No | Debug state not synced |
| Management sessions (SSH) | No | SSH to fxp0/reth management drops |

---

## 4. Split-Brain Prevention

### Detection Mechanisms

Split-brain occurs when both nodes believe they are the primary for the same RG. This is catastrophic — both nodes actively forward traffic, causing duplicate MAC addresses, ARP conflicts, and session inconsistency.

```
Prevention mechanisms:

1. Heartbeat over fxp1 (primary detection method)
   - 1-second heartbeat interval
   - heartbeat-threshold consecutive misses → declare peer dead
   - Default threshold: 3 (peer declared dead after 3 seconds)
   - Conservative threshold: 8-10 (prevents false positive in CPU-heavy scenarios)

2. Fabric link heartbeat (secondary detection)
   - Heartbeat also sent over fab links
   - If fxp1 fails but fab is up: peer liveness still detected
   - Only when BOTH fxp1 AND fab fail: risk of split-brain

3. Dual-fabric redundancy
   - fab0 + fab1 provide redundant data path
   - Both must fail for complete fabric loss
   - Recommended: use different physical paths for fab0 and fab1
```

### Split-Brain Resolution

```
When both fxp1 and all fab links fail simultaneously:

  Node 0 perspective:         Node 1 perspective:
  "Peer is dead"              "Peer is dead"
  "I am PRIMARY for RG1"      "I am PRIMARY for RG1"
  → Both forward traffic       → Both forward traffic
  → Duplicate MACs on reth     → ARP conflicts
  → Session table divergence   → Traffic black-holes

Resolution strategy:
  1. JSRP uses secondary priority (node-id) as tiebreaker
     - Node with LOWER node-id remains primary
     - Node with HIGHER node-id should relinquish (disable)

  2. In practice: dual-link failure is often a complete network partition
     - One node cannot reach the network anyway
     - The reachable node continues serving traffic

  3. Manual intervention required to recover:
     - Restore at least one fabric link
     - Verify which node has the current session state
     - Force secondary to resync from primary
```

### Fabric Link Design for Split-Brain Avoidance

```
Recommended topology:

  Node 0                            Node 1
  ge-0/0/3 ──── (direct cable) ──── ge-5/0/3    fab0 (path A)
  ge-0/0/4 ──── (direct cable) ──── ge-5/0/4    fab1 (path B)
  fxp1     ──── (direct cable) ──── fxp1         control

  Three independent physical paths:
    - If one cable fails: two paths remain
    - If switch fails (if using a switch): direct cables unaffected
    - Never route fxp1 and fab through the same switch/path

Avoid:
  - Single switch between nodes (SPOF)
  - fxp1 and fab on same physical cable (SPOF)
  - Long cable runs (> 100m for copper, use fiber)
```

---

## 5. HA Convergence Analysis

### Failover Timeline

```
Event: Interface ge-0/0/1 on Node 0 goes DOWN (weight 255, triggers RG1 failover)

T+0ms:      Link failure detected by PFE (hardware interrupt)
T+1-5ms:    PFE notifies JSRP daemon on Node 0
T+5-10ms:   JSRP recalculates effective priority
            effective_priority_node0 = 200 - 255 = -55
            effective_priority_node1 = 100
            Decision: failover RG1 to Node 1
T+10-15ms:  JSRP sends failover notification to Node 1 via fxp1
T+15-20ms:  Node 1 JSRP acknowledges, begins RG1 primary role
T+20-50ms:  Reth interfaces activate on Node 1
            - Standby reth members become active
            - GARP (Gratuitous ARP) sent for all reth IP addresses
            - GARP sent for all NAT pool addresses
            - GARP sent for all virtual IP addresses
T+50-100ms: Connected switches update MAC tables (GARP processing)
T+100-200ms: First packets forwarded by Node 1

Total convergence: 100-200ms typical
  - During this window: traffic to reth MAC is black-holed
  - TCP: retransmits recover within 200ms-1s
  - UDP: application-dependent recovery
  - BGP/OSPF: adjacencies maintained (sessions synced via RTO)
```

### Convergence Variables

| Factor | Impact on Convergence | Mitigation |
|:---|:---|:---|
| Number of reth interfaces | More reths = more GARPs = longer | Minimize reth count |
| Number of NAT pool addresses | More addresses = more GARPs | Consolidate pools |
| Switch MAC learning speed | Slow switches extend black-hole | Use fast-learning switches |
| Spanning tree | Can add 30+ seconds | Use RSTP or portfast |
| IP monitoring interval | Slower detection = longer | Reduce probe interval |
| Preemption delay | Adds configurable delay | Set appropriate for environment |
| Routing protocol convergence | OSPF/BGP re-convergence on new node | Use NSR + GRES |

### Measuring Convergence

```
# Method 1: Continuous ping during planned failover
ping 10.0.0.1 rapid count 1000 interval 0.01    # 10ms intervals from external host
# Count lost replies = approximate downtime

# Method 2: Session continuity test
# Establish TCP session (e.g., SSH) through SRX
# Trigger failover
# Observe if session survives
# Check session table on new active node:
show security flow session session-identifier <id>

# Method 3: Failover event timing
show chassis cluster switch-events
# Shows timestamp of each failover event
# Delta between events = convergence time
```

---

## 6. Active/Active Traffic Distribution

### RG-to-Interface Mapping

In active/active, traffic distribution is controlled by which RG owns which reth interfaces:

```
Design: two uplinks, two downlinks, distributed across nodes

  RG1 (Node 0 primary):          RG2 (Node 1 primary):
  ├─ reth0 (uplink-A)            ├─ reth2 (uplink-B)
  └─ reth1 (downlink-A)          └─ reth3 (downlink-B)

  Traffic flow A: Client-A → reth0 → policy → reth1 → Server-A
    (entirely on Node 0, no fabric traversal)

  Traffic flow B: Client-B → reth2 → policy → reth3 → Server-B
    (entirely on Node 1, no fabric traversal)
```

### Asymmetric Routing Problem

The challenge with active/active is return traffic:

```
Problem scenario:
  Client sends to reth0 (Node 0)
  SRX creates session on Node 0
  Server responds via reth2 (Node 1)    ← asymmetric!

  Node 1 receives the response but has no session (it is on Node 0)
  → Packet forwarded over fabric to Node 0
  → Node 0 processes as existing session
  → Egress via reth0 (which is on Node 0)
  → Works, but adds fabric latency and bandwidth consumption

Better design:
  Ensure each traffic flow enters and exits via the same node
  Use routing to steer return traffic to the correct node
  Or accept the fabric traversal penalty
```

### Active/Active Capacity Planning

```
Each node in active/active must be sized to handle FULL load if the other fails:

  Node 0 normal load: 50% (RG1 traffic)
  Node 1 normal load: 50% (RG2 traffic)

  Node 0 failure:
    All RGs fail to Node 1
    Node 1 load: 100%

  If Node 1 is sized for only 60% capacity:
    Traffic exceeding 60% is dropped or delayed
    This defeats the purpose of HA

Rule: each node must support 100% of total traffic
  Active/active provides load distribution, NOT capacity scaling
  The benefit is reduced failover impact (sessions on surviving node are unaffected)
```

---

## 7. HA Scaling Considerations

### Session Table Synchronization Limits

```
Session sync bandwidth required:
  RTO size per session: ~200 bytes (creation), ~64 bytes (update/delete)

  New sessions per second: S
  Session modifications per second: M (state changes, counter updates)
  Session closures per second: C

  Fabric bandwidth for sync = (S * 200) + (M * 64) + (C * 64) bytes/sec

  Example: 50,000 new sessions/sec
    Sync bandwidth = (50000 * 200) + (50000 * 64) + (50000 * 64)
                   = 10 MB/s + 3.2 MB/s + 3.2 MB/s
                   = 16.4 MB/s (131 Mbps)

  This must fit within the fabric link capacity
  1 GbE fabric: supports ~400,000 new sessions/sec for sync alone
  10 GbE fabric: supports ~4,000,000 new sessions/sec
```

### Maximum Cluster Capacity

| SRX Platform | Max Sessions | Max Policies | Max Reth | Notes |
|:---|:---|:---|:---|:---|
| SRX300 | 64,000 | 1,024 | 8 | Branch |
| SRX345 | 256,000 | 2,048 | 16 | Branch |
| SRX1500 | 2,000,000 | 8,192 | 128 | Mid-range |
| SRX4100 | 4,000,000 | 16,384 | 128 | Data center |
| SRX4600 | 10,000,000 | 32,768 | 128 | Data center + multi-node |
| SRX5400 | 20,000,000 | 65,536 | 128 | Service provider |
| SRX5800 | 60,000,000 | 131,072 | 128 | Service provider |

### Fabric Link Sizing

```
Fabric link must carry:
  1. RTO synchronization traffic (see calculation above)
  2. Transit traffic for asymmetric flows (active/active or misrouted traffic)
  3. Management sync (config, ARP, etc.)

Sizing rule:
  Fabric link >= max(RTO_bandwidth * 2, asymmetric_traffic_bandwidth)

  Factor of 2 for RTO accounts for burst synchronization during traffic spikes

  For most deployments:
    1 GbE fabric: sufficient for < 1 Gbps throughput clusters
    10 GbE fabric: recommended for > 1 Gbps throughput or active/active
    LAG fabric: recommended for high-availability of the fabric itself
```

## Prerequisites

- Security zones and policies, IP routing, SRX platform architecture, NAT fundamentals, chassis cluster basic configuration

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| RG election (priority comparison) | O(1) | O(monitored_items) |
| Heartbeat processing | O(1) per heartbeat | O(1) |
| RTO session sync | O(1) per session | O(sessions) total fabric |
| Failover convergence | O(reth_count) for GARP | O(1) |
| Session table resync (full) | O(sessions) | O(sessions) |
| Interface monitor evaluation | O(monitors_per_RG) | O(monitors) |

---

*A chassis cluster is only as reliable as its weakest link — literally. The fabric links, heartbeat path, and interface monitoring configuration determine whether a failover is graceful or catastrophic. Design for the failure mode you cannot predict: dual-link failure, split-brain, asymmetric routing under load. Test every scenario before production, because production will test every scenario without asking.*
