# The Mathematics of eBPF — Verifier Algorithms, Map Complexity & Ring Buffer Sizing

> *eBPF is a virtual machine inside the kernel with a static verifier that proves program safety before execution. The verifier's algorithm, the map data structures' complexity, and the ring buffer sizing are all governed by precise mathematical constraints.*

---

## 1. The eBPF Verifier — DAG Exploration

### Verification as Graph Problem

The verifier treats the eBPF program as a **directed acyclic graph** of instructions:

$$G = (V, E) \text{ where } V = \text{instructions}, E = \text{control flow edges}$$

### Verification Algorithm

The verifier performs a **depth-first exploration** of all reachable paths, tracking register state at each instruction:

$$state(insn) = (R_0, R_1, ..., R_{10}, stack\_state, map\_state)$$

At each instruction, the verifier:
1. Checks preconditions (register types, bounds)
2. Computes post-state (register transformations)
3. At branches: explores both paths

### Complexity Bounds

$$max\_instructions = 1,000,000 \text{ (BPF\_COMPLEXITY\_LIMIT, kernel 5.3+)}$$

Old limit: 4,096 instructions. The verifier also limits:

$$max\_states = O(N \times B) \text{ where } B = \text{branching factor}$$

### Verification Time

$$T_{verify} = O(V \times S \times C)$$

Where:
- $V$ = number of instructions (up to 1M)
- $S$ = number of tracked states per instruction
- $C$ = cost of state comparison

Typical: 1-100 ms for moderate programs. Worst case: seconds for complex programs with many branches.

### Bounded Loops

Since kernel 5.3, the verifier allows bounded loops:

$$verified \iff \exists n : loop\_bound \leq n \text{ (provable upper bound)}$$

The verifier unrolls loops mentally and tracks that iteration count is bounded:

$$iterations \leq max\_iterations \text{ (inferred from register bounds)}$$

---

## 2. eBPF Maps — Data Structure Complexity

### Map Types and Operations

| Map Type | Lookup | Insert | Delete | Space |
|:---|:---:|:---:|:---:|:---:|
| BPF_MAP_TYPE_HASH | $O(1)$ avg | $O(1)$ avg | $O(1)$ avg | $O(n)$ |
| BPF_MAP_TYPE_ARRAY | $O(1)$ | $O(1)$ | N/A | $O(max\_entries)$ |
| BPF_MAP_TYPE_LRU_HASH | $O(1)$ avg | $O(1)$ avg | $O(1)$ amort | $O(n)$ |
| BPF_MAP_TYPE_PERCPU_HASH | $O(1)$ avg | $O(1)$ avg | $O(1)$ avg | $O(n \times n_{cpu})$ |
| BPF_MAP_TYPE_LPM_TRIE | $O(k)$ | $O(k)$ | $O(k)$ | $O(n \times k)$ |
| BPF_MAP_TYPE_RINGBUF | $O(1)$ | $O(1)$ | N/A | $O(size)$ |

Where $k$ = key length in bits for LPM trie, $n$ = number of entries.

### Hash Map Internals

eBPF hash maps use a **hash table with chaining**:

$$buckets = next\_power\_of\_2(max\_entries)$$

$$load\_factor = \frac{entries}{buckets}$$

$$avg\_chain\_length = load\_factor$$

### Memory Cost per Map

$$memory = \begin{cases} max\_entries \times (key\_size + value\_size + overhead) & \text{hash/array} \\ max\_entries \times (key\_size + value\_size + overhead) \times n_{cpu} & \text{percpu} \end{cases}$$

Per-entry overhead: ~64 bytes (hash map) or 0 bytes (array).

**Example:** Hash map with 10,000 entries, 4-byte key, 8-byte value:

$$memory = 10000 \times (4 + 8 + 64) = 760 KB$$

Per-CPU variant on 16-core system:

$$memory = 10000 \times (4 + 8 + 64) \times 16 = 12.2 MB$$

---

## 3. Ring Buffer Sizing

### BPF_MAP_TYPE_RINGBUF

A single-producer, multi-consumer ring buffer shared between kernel and userspace.

### Sizing Formula

$$ringbuf\_size = 2^n \text{ (must be power of 2)}$$

$$required\_size \geq event\_rate \times event\_size \times T_{drain}$$

Where $T_{drain}$ = time for userspace to consume events.

### Worked Example

Network event tracing: 100,000 events/s, 64 bytes each, userspace polls every 10 ms:

$$required = 100000 \times 64 \times 0.01 = 64,000 \text{ bytes}$$

$$ringbuf\_size = 2^{17} = 131072 \text{ bytes (next power of 2)}$$

### Overflow Detection

$$overflow \iff write\_offset - read\_offset \geq ringbuf\_size$$

Events dropped on overflow:

$$drop\_rate = \max(0, event\_rate - drain\_rate)$$

$$drain\_rate = \frac{ringbuf\_size}{T_{poll} \times event\_size}$$

### Per-CPU Ring Buffer vs Shared

| Type | Memory | Ordering | Contention |
|:---|:---:|:---|:---|
| BPF_MAP_TYPE_PERF_EVENT_ARRAY | $size \times n_{cpu}$ | Per-CPU only | None |
| BPF_MAP_TYPE_RINGBUF | $size$ (shared) | Global | Lock-free |

Memory savings with shared ringbuf on 64-core system: $64\times$.

---

## 4. Helper Function Cost

### Common Helper Latencies

| Helper | Typical Cost | Notes |
|:---|:---:|:---|
| bpf_map_lookup_elem | 20-100 ns | Hash map lookup |
| bpf_map_update_elem | 50-200 ns | Hash map insert/update |
| bpf_get_current_pid_tgid | 10-20 ns | Read task_struct |
| bpf_ktime_get_ns | 5-15 ns | Read clock |
| bpf_probe_read_kernel | 20-50 ns | Safe kernel memory read |
| bpf_ringbuf_submit | 10-30 ns | Write to ring buffer |
| bpf_perf_event_output | 50-150 ns | Write to perf buffer |

### Total Program Cost

$$T_{program} = \sum_{insn} T_{insn} + \sum_{helper} T_{helper}$$

For a typical tracing program (50 instructions + 3 helpers):

$$T_{program} \approx 50 \times 1ns + 3 \times 50ns = 200ns$$

### Overhead Percentage

$$overhead = \frac{T_{program}}{T_{event\_interval}} \times 100\%$$

At 100,000 events/s: $T_{interval} = 10\mu s$

$$overhead = \frac{200ns}{10\mu s} = 2\%$$

---

## 5. Tail Calls and Program Chaining

### Tail Call Chain

eBPF programs can chain via `bpf_tail_call()`:

$$max\_depth = 33 \text{ (BPF\_MAX\_TAIL\_CALL\_CNT)}$$

### Effective Program Size

$$effective\_instructions = \sum_{i=0}^{depth} instructions(program_i)$$

$$effective \leq 33 \times 1,000,000 = 33M \text{ instructions}$$

### Tail Call Cost

$$T_{tail\_call} \approx 10-30ns \text{ (jump to new program)}$$

Much cheaper than returning to kernel and re-entering eBPF.

---

## 6. JIT Compilation — Native Performance

### JIT Speedup

$$speedup = \frac{T_{interpreted}}{T_{JIT}}$$

| Metric | Interpreted | JIT Compiled | Speedup |
|:---|:---:|:---:|:---:|
| Instruction cost | 5-10 ns | 0.3-1 ns | 5-30x |
| Helper call | +2-5 ns overhead | Native call | 1-2x |
| Map lookup | 30-120 ns | 20-100 ns | 1.2-1.5x |

### JIT Memory Cost

$$memory_{JIT} = instructions \times expansion\_ratio \times avg\_native\_size$$

Expansion ratio (eBPF → x86-64): typically 1.0-2.0x in instruction count.

Average native instruction: 4 bytes.

$$memory_{JIT} \approx N_{insn} \times 1.5 \times 4 = 6 \times N_{insn} \text{ bytes}$$

1000-instruction program: ~6 KB of JIT code.

### JIT Hardening

With `net.core.bpf_jit_harden=1`:

$$T_{insn} += T_{constant\_blinding} \approx 0.1-0.5 ns$$

Constant blinding XORs immediate values to prevent gadget construction.

---

## 7. CO-RE (Compile Once, Run Everywhere)

### BTF Type Matching

CO-RE uses BTF (BPF Type Format) to relocate struct field accesses:

$$offset(field) = BTF\_lookup(struct, field\_name)$$

### Relocation Cost

$$T_{relocation} = N_{relocations} \times T_{BTF\_lookup}$$

Where $T_{BTF\_lookup} \approx 1-10\mu s$ (string comparison in type info).

This happens once at load time, not at runtime.

### BTF Size

$$BTF\_size \approx N_{types} \times avg\_type\_size$$

Kernel BTF: typically 2-5 MB for ~50,000 types.

---

## 8. Summary of eBPF Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Verifier complexity | $O(V \times S \times C)$ | State exploration |
| Instruction limit | 1,000,000 | Hard bound |
| Hash map lookup | $O(1)$ average | Hash table |
| Map memory | $entries \times (key + value + overhead)$ | Linear |
| Ring buffer size | $event\_rate \times event\_size \times T_{drain}$ | Throughput |
| Program cost | $\Sigma T_{insn} + \Sigma T_{helper}$ | Latency sum |
| JIT speedup | $T_{interp} / T_{JIT} \approx 5-30\times$ | Performance ratio |
| Tail call depth | Max 33 programs | Chaining limit |

## Prerequisites

- virtual machines, static analysis, DAG verification, hash tables, ring buffers, JIT compilation, kernel internals

## Complexity

| Operation | Time Complexity | Notes |
|:---|:---|:---|
| Verifier pass | $O(V \times S \times C)$ | V=vertices, S=states, C=conditions |
| Hash map lookup | $O(1)$ average | Per-CPU or shared |
| Array map access | $O(1)$ | Direct index |
| Ring buffer write | $O(1)$ | Lock-free single producer |
| JIT compilation | $O(N_{insn})$ | One-time per program load |
| Tail call dispatch | $O(1)$ | Array index jump |

---

*eBPF is a verified virtual machine: every program must pass a static safety proof before the kernel will execute it. The verifier's DAG exploration guarantees termination, the JIT eliminates interpretation overhead, and the map types provide kernel-safe data structures with known complexity bounds.*
