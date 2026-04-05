# Distributed Systems (Consensus, Consistency, and Coordination)

A practitioner's reference for the fundamental building blocks of reliable distributed systems — from consistency models to consensus protocols.

## CAP and PACELC

### CAP Theorem

```
Pick two (in practice, pick CP or AP since P is non-negotiable):

     Consistency
        /\
       /  \
      /    \
     /  CP  \
    /________\
   /\   /\   \
  /  \ /  \   \
 / AP \/ CA \   \
/______\______\
Availability  Partition Tolerance
```

| System | Type | Behavior During Partition |
|---|---|---|
| etcd, ZooKeeper, Consul | CP | Rejects writes to minority |
| Cassandra, DynamoDB | AP | Accepts writes, resolves later |
| PostgreSQL (single node) | CA | No partition tolerance |

### PACELC

```
If Partition:
    Choose Availability or Consistency
Else (normal operation):
    Choose Latency or Consistency

Examples:
    Cassandra:  PA/EL  (Available during partition, Low latency normally)
    MongoDB:    PC/EC  (Consistent always)
    DynamoDB:   PA/EL  (with eventual consistency reads)
    CockroachDB: PC/EC (Consistent always, higher latency)
```

## Consistency Models

### Spectrum from Strongest to Weakest

```
Strongest ─────────────────────────────────── Weakest

Linearizable → Sequential → Causal → Eventual
     │              │           │          │
  Real-time      Per-process  Respects   Eventually
  ordering       ordering    causality   converges
```

### Linearizability (Strong Consistency)

```
Every operation appears to take effect atomically at some point
between its invocation and response.

Timeline:
Client A:  |--write(x=1)--|
Client B:       |--read(x)--| must return 1
Client C:              |--read(x)--| must return 1

If read of x returns 1, all subsequent reads must return >= 1
```

### Sequential Consistency

```
All operations appear in some sequential order consistent with
per-process ordering. No real-time guarantee.

Thread 1: write(x=1)  write(y=2)
Thread 2: read(y)=2   read(x)=0  ← ALLOWED (different global order)

Legal ordering: T2.read(y)=0, T1.write(x=1), T1.write(y=2), T2.read(y)=2, T2.read(x)=0
Illegal: T1.write(y=2), T2.read(y)=2, T1.write(x=1), T2.read(x)=0
  (T2 read of x must see 1 if this ordering used, since write(x=1) precedes)
```

### Causal Consistency

```
If operation A causally precedes B (A → B), then every process
sees A before B. Concurrent operations may be seen in any order.

Process 1: write(x=1)
Process 2: read(x)=1, write(y=2)    ← y=2 causally depends on x=1
Process 3: must see x=1 before y=2  ← causal order preserved
Process 3: may see y=2 before z=3 if z=3 is concurrent with y=2
```

### Eventual Consistency

```
If no new updates, all replicas eventually converge to same value.
No ordering guarantees during convergence.

t=0:   Node A: x=1    Node B: x=0    Node C: x=0
t=1:   Node A: x=1    Node B: x=1    Node C: x=0
t=2:   Node A: x=1    Node B: x=1    Node C: x=1  ← converged
```

## Consensus Protocols

### Raft — Leader Election

```
States: Follower → Candidate → Leader

Election Rules:
1. All nodes start as Followers with random election timeout (150-300ms)
2. If no heartbeat received before timeout, become Candidate
3. Candidate increments term, votes for self, requests votes from all
4. Node grants vote if:
   - Haven't voted in this term yet
   - Candidate's log is at least as up-to-date
5. Candidate becomes Leader if receives majority (N/2 + 1) votes
6. Leader sends periodic heartbeats to prevent new elections
```

```go
// Simplified Raft election state
type RaftNode struct {
    mu          sync.Mutex
    state       NodeState
    currentTerm uint64
    votedFor    string
    log         []LogEntry
    commitIndex uint64

    // Leader state
    nextIndex  map[string]uint64
    matchIndex map[string]uint64
}

func (n *RaftNode) RequestVote(req VoteRequest) VoteResponse {
    n.mu.Lock()
    defer n.mu.Unlock()

    if req.Term < n.currentTerm {
        return VoteResponse{Term: n.currentTerm, Granted: false}
    }

    if req.Term > n.currentTerm {
        n.currentTerm = req.Term
        n.votedFor = ""
        n.state = Follower
    }

    logOK := req.LastLogTerm > n.lastLogTerm() ||
        (req.LastLogTerm == n.lastLogTerm() && req.LastLogIndex >= n.lastLogIndex())

    if (n.votedFor == "" || n.votedFor == req.CandidateID) && logOK {
        n.votedFor = req.CandidateID
        return VoteResponse{Term: n.currentTerm, Granted: true}
    }
    return VoteResponse{Term: n.currentTerm, Granted: false}
}
```

### Raft — Log Replication

```
Leader appends entry, replicates to followers:

Leader:    [1:set x=1] [2:set y=2] [3:set x=3]
Follower A: [1:set x=1] [2:set y=2] [3:set x=3]  ← up to date
Follower B: [1:set x=1] [2:set y=2]                ← behind
Follower C: [1:set x=1]                             ← far behind

Commit rule: Entry committed when replicated to majority.
With 3/3 nodes having entry 2: entry 2 is committed.
Entry 3 on 2/3 nodes (Leader + A): committed (majority).
```

### Paxos Overview

```
Roles: Proposers, Acceptors, Learners

Phase 1 (Prepare):
  Proposer → Acceptors: "Prepare(n)" where n = proposal number
  Acceptor: If n > highest seen, promise not to accept < n
            Return any previously accepted (n_a, v_a)

Phase 2 (Accept):
  Proposer: If majority promised, send "Accept(n, v)"
            v = highest-numbered accepted value, or proposer's choice
  Acceptor: If no promise to higher n, accept (n, v)

Consensus reached when majority of acceptors accept same (n, v)
```

## Vector Clocks and Version Vectors

```go
type VectorClock map[string]uint64

func (vc VectorClock) Increment(nodeID string) {
    vc[nodeID]++
}

func (vc VectorClock) Merge(other VectorClock) {
    for k, v := range other {
        if v > vc[k] {
            vc[k] = v
        }
    }
}

// Compare: returns -1 (before), 0 (concurrent), 1 (after)
func (vc VectorClock) Compare(other VectorClock) int {
    less, greater := false, false
    allKeys := make(map[string]bool)
    for k := range vc    { allKeys[k] = true }
    for k := range other { allKeys[k] = true }

    for k := range allKeys {
        if vc[k] < other[k] { less = true }
        if vc[k] > other[k] { greater = true }
    }

    if less && !greater  { return -1 } // vc happened before other
    if greater && !less  { return 1 }  // vc happened after other
    if !less && !greater { return -1 } // equal
    return 0                            // concurrent
}
```

## CRDTs (Conflict-Free Replicated Data Types)

### G-Counter (Grow-Only Counter)

```go
// Each node maintains its own counter; total = sum of all
type GCounter struct {
    counts map[string]uint64 // nodeID -> count
}

func (g *GCounter) Increment(nodeID string) {
    g.counts[nodeID]++
}

func (g *GCounter) Value() uint64 {
    var total uint64
    for _, v := range g.counts {
        total += v
    }
    return total
}

func (g *GCounter) Merge(other *GCounter) {
    for k, v := range other.counts {
        if v > g.counts[k] {
            g.counts[k] = v
        }
    }
}
```

### PN-Counter (Positive-Negative Counter)

```go
type PNCounter struct {
    positive *GCounter
    negative *GCounter
}

func (pn *PNCounter) Increment(nodeID string) { pn.positive.Increment(nodeID) }
func (pn *PNCounter) Decrement(nodeID string) { pn.negative.Increment(nodeID) }
func (pn *PNCounter) Value() int64 {
    return int64(pn.positive.Value()) - int64(pn.negative.Value())
}
func (pn *PNCounter) Merge(other *PNCounter) {
    pn.positive.Merge(other.positive)
    pn.negative.Merge(other.negative)
}
```

### LWW-Register (Last-Writer-Wins)

```go
type LWWRegister struct {
    value     any
    timestamp time.Time
}

func (r *LWWRegister) Set(value any, ts time.Time) {
    if ts.After(r.timestamp) {
        r.value = value
        r.timestamp = ts
    }
}

func (r *LWWRegister) Merge(other *LWWRegister) {
    if other.timestamp.After(r.timestamp) {
        r.value = other.value
        r.timestamp = other.timestamp
    }
}
```

### OR-Set (Observed-Remove Set)

```go
type ORSet struct {
    elements map[string]map[string]bool // value -> set of unique tags
}

func (s *ORSet) Add(value, nodeID string) {
    tag := fmt.Sprintf("%s-%d", nodeID, time.Now().UnixNano())
    if s.elements[value] == nil {
        s.elements[value] = make(map[string]bool)
    }
    s.elements[value][tag] = true
}

func (s *ORSet) Remove(value string) {
    delete(s.elements, value) // removes all tags for value
}

func (s *ORSet) Contains(value string) bool {
    return len(s.elements[value]) > 0
}

func (s *ORSet) Merge(other *ORSet) {
    for val, tags := range other.elements {
        if s.elements[val] == nil {
            s.elements[val] = make(map[string]bool)
        }
        for tag := range tags {
            s.elements[val][tag] = true
        }
    }
}
```

## Consistent Hashing

```go
type ConsistentHash struct {
    ring     map[uint32]string // hash -> node
    sorted   []uint32          // sorted hash values
    vnodes   int               // virtual nodes per physical node
    hashFunc func([]byte) uint32
}

func (ch *ConsistentHash) AddNode(node string) {
    for i := 0; i < ch.vnodes; i++ {
        key := fmt.Sprintf("%s-vnode-%d", node, i)
        hash := ch.hashFunc([]byte(key))
        ch.ring[hash] = node
        ch.sorted = append(ch.sorted, hash)
    }
    sort.Slice(ch.sorted, func(i, j int) bool {
        return ch.sorted[i] < ch.sorted[j]
    })
}

func (ch *ConsistentHash) GetNode(key string) string {
    hash := ch.hashFunc([]byte(key))
    idx := sort.Search(len(ch.sorted), func(i int) bool {
        return ch.sorted[i] >= hash
    })
    if idx == len(ch.sorted) {
        idx = 0 // wrap around ring
    }
    return ch.ring[ch.sorted[idx]]
}
```

## Quorum Systems

```
N = total replicas
R = read quorum (nodes that must respond to read)
W = write quorum (nodes that must acknowledge write)

Strong consistency: R + W > N
  Ensures read and write quorums overlap

Common configurations:
  N=3, R=2, W=2: Standard quorum (overlap=1)
  N=3, R=1, W=3: Fast reads, slow writes
  N=3, R=3, W=1: Fast writes, slow reads
  N=5, R=3, W=3: Higher fault tolerance (survives 2 failures)
```

## Gossip Protocols

```go
type GossipNode struct {
    mu      sync.Mutex
    members map[string]*MemberState
}

type MemberState struct {
    Address     string
    Heartbeat   uint64
    LastUpdated time.Time
    Status      string // alive, suspect, dead
}

func (g *GossipNode) Gossip() {
    g.mu.Lock()
    // Select random peer
    peers := g.alivePeers()
    if len(peers) == 0 {
        g.mu.Unlock()
        return
    }
    target := peers[rand.Intn(len(peers))]
    digest := g.buildDigest()
    g.mu.Unlock()

    // Send digest, receive updates
    updates, err := g.sendGossip(target, digest)
    if err != nil {
        g.markSuspect(target)
        return
    }

    g.mu.Lock()
    defer g.mu.Unlock()
    for _, update := range updates {
        existing := g.members[update.Address]
        if existing == nil || update.Heartbeat > existing.Heartbeat {
            g.members[update.Address] = update
        }
    }
}

// Dissemination: information reaches all N nodes in O(log N) rounds
// with high probability when each node contacts O(log N) peers per round
```

## Leader Election

```go
// Bully algorithm (simplified)
func (n *Node) StartElection() {
    // Send election message to all higher-ID nodes
    higherNodes := n.nodesWithHigherID()

    if len(higherNodes) == 0 {
        // No higher nodes — I am the leader
        n.declareLeader()
        return
    }

    responses := make(chan bool, len(higherNodes))
    for _, node := range higherNodes {
        go func(target string) {
            ok := n.sendElection(target)
            responses <- ok
        }(node)
    }

    // Wait for responses with timeout
    timer := time.NewTimer(5 * time.Second)
    anyResponse := false
    for i := 0; i < len(higherNodes); i++ {
        select {
        case ok := <-responses:
            if ok {
                anyResponse = true
            }
        case <-timer.C:
            break
        }
    }

    if !anyResponse {
        n.declareLeader() // no higher node responded
    }
    // else: wait for higher node to declare itself leader
}
```

## Split-Brain Detection

```go
// Fencing tokens prevent split-brain writes
type FencingToken struct {
    Token  uint64
    Leader string
    Epoch  uint64
}

func (s *Storage) Write(key string, value []byte, token FencingToken) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if token.Epoch < s.currentEpoch {
        return fmt.Errorf("stale fencing token: epoch %d < %d",
            token.Epoch, s.currentEpoch)
    }

    s.currentEpoch = token.Epoch
    return s.store.Put(key, value)
}
```

## Replication Modes

| Mode | Durability | Latency | Consistency |
|---|---|---|---|
| Synchronous | Highest | Highest | Strong |
| Semi-synchronous | High | Medium | Strong (if quorum) |
| Asynchronous | Lower | Lowest | Eventual |

```go
// Semi-synchronous: wait for at least one replica ACK
func (l *Leader) Replicate(entry LogEntry) error {
    acks := make(chan error, len(l.followers))

    for _, follower := range l.followers {
        go func(f *Follower) {
            acks <- f.AppendEntry(entry)
        }(follower)
    }

    // Wait for quorum (majority)
    needed := len(l.followers)/2 + 1
    var succeeded int
    for i := 0; i < len(l.followers); i++ {
        if err := <-acks; err == nil {
            succeeded++
            if succeeded >= needed {
                return nil // quorum achieved
            }
        }
    }
    return fmt.Errorf("failed to achieve quorum: %d/%d", succeeded, needed)
}
```

## Tips

- Raft is easier to understand and implement than Paxos — prefer it for new systems
- Vector clocks grow with the number of nodes; for large clusters consider dotted version vectors
- CRDTs trade expressiveness for automatic conflict resolution — not all data structures have CRDT equivalents
- Consistent hashing with 100-200 virtual nodes per physical node provides good load distribution
- Gossip protocol convergence time is $O(\log N)$ rounds — efficient for large clusters
- Always use fencing tokens to prevent split-brain writes from stale leaders
- Quorum intersections (R+W>N) guarantee reading the latest write, but do not prevent all anomalies

## See Also

- `detail/patterns/distributed-systems.md` — FLP impossibility, CRDT lattice theory
- `sheets/patterns/microservices-patterns.md` — circuit breakers, sagas
- `sheets/patterns/event-driven-architecture.md` — event sourcing in distributed contexts

## References

- "Designing Data-Intensive Applications" by Martin Kleppmann (O'Reilly, 2017)
- Raft Paper: "In Search of an Understandable Consensus Algorithm" (Ongaro & Ousterhout, 2014)
- "A comprehensive study of CRDTs" by Shapiro et al. (2011)
- Dynamo Paper: "Dynamo: Amazon's Highly Available Key-value Store" (DeCandia et al., 2007)
