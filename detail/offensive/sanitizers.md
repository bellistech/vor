# The Mathematics of Sanitizers -- Shadow Memory and Error Detection Theory

> *A sanitizer transforms every memory access into a two-step operation: first consult the shadow state to determine legality, then perform the access -- trading constant-factor runtime overhead for deterministic detection of an entire class of undefined behavior.*

---

## 1. Shadow Memory Architecture (Memory Mapping Theory)

### The Problem

AddressSanitizer must track the accessibility state of every byte in the program's address space. Naive per-byte tracking would double memory consumption. ASAN uses a compressed shadow memory scheme that maps 8 application bytes to 1 shadow byte, encoding whether each byte in the group is accessible.

### The Formula

The shadow memory mapping for ASAN on 64-bit systems:

$$\text{shadow\_addr} = \frac{\text{app\_addr}}{8} + \text{shadow\_offset}$$

where $\text{shadow\_offset} = 0x7FFF8000$ on Linux x86-64 (varies by platform).

Each shadow byte encodes the state of 8 application bytes:

$$\text{shadow}(a) = \begin{cases} 0 & \text{all 8 bytes accessible} \\ k \in [1,7] & \text{first } k \text{ bytes accessible, rest poisoned} \\ \text{negative} & \text{all 8 bytes poisoned (various error codes)} \end{cases}$$

The negative shadow byte values encode the poison reason:

| Shadow Value | Meaning |
|:---:|:---|
| 0 | Fully accessible |
| 1-7 | Partially accessible (first k bytes) |
| 0xfa | Stack left redzone |
| 0xfb | Stack mid redzone |
| 0xfc | Stack right redzone |
| 0xfd | Stack after return |
| 0xfe | Stack use after scope |
| 0xf1 | Heap left redzone |
| 0xf2 | Heap right redzone |
| 0xf3 | Freed heap region |
| 0xf5 | Global redzone |

Total memory overhead:

$$\text{overhead} = \frac{1}{8} \cdot |\text{address space used}| + |\text{redzones}| + |\text{quarantine}|$$

For a program using 100 MB of heap: shadow = 12.5 MB, redzones approximately 10-30% extra, quarantine 256 MB default. Total: approximately 370 MB for 100 MB of actual data (3.7x).

### Worked Example

Application allocates `char buf[13]` at address `0x10000`:

Shadow addresses: `0x10000/8 + offset = 0x2000 + offset`

The 13 bytes occupy shadow bytes at offsets 0 and 1:
- Shadow byte 0: value `0` (bytes 0-7 all accessible)
- Shadow byte 1: value `5` (bytes 8-12 accessible, 13-15 are redzone)

Access check for `buf[14]` (address `0x1000e`):
- Shadow address: `0x1000e / 8 = shadow byte 1`
- Shadow value: `5`
- Byte position within group: `0x1000e % 8 = 6`
- Check: `6 >= 5` means inaccessible. ASAN reports heap-buffer-overflow.

## 2. Redzone Theory and Overflow Detection (Spatial Safety)

### The Problem

Buffer overflows read or write beyond allocated boundaries. ASAN surrounds every allocation with "redzones" -- poisoned memory regions that trigger an error when accessed. The redzone size determines the maximum detectable overflow distance.

### The Formula

For a heap allocation of $n$ bytes, ASAN allocates:

$$\text{total} = \text{left\_redzone} + \lceil n / 8 \rceil \cdot 8 + \text{right\_redzone}$$

Default left redzone: 16 bytes. Right redzone: $\max(16, \text{next alignment boundary})$.

The detection guarantee:

$$\text{detectable overflow} \leq \text{redzone\_size}$$

With default 16-byte redzones, overflows of 1-16 bytes past the buffer are always detected. Overflows of 17+ bytes may land in another allocation's valid region and go undetected:

$$P(\text{miss}) = \begin{cases} 0 & \text{if overflow} \leq \text{redzone} \\ \frac{\text{overflow} - \text{redzone}}{\text{gap to next alloc}} & \text{otherwise} \end{cases}$$

For stack variables, redzones are placed between each variable:

$$[\text{rz}_0 | \text{var}_1 | \text{rz}_1 | \text{var}_2 | \text{rz}_2 | \ldots | \text{rz}_n]$$

The probability that a random overflow of $d$ bytes from a buffer of size $n$ is detected:

$$P(\text{detect} \mid d) = \begin{cases} 1 & d \leq r \\ 1 - \frac{d - r}{g} & r < d \leq r + g \\ 0 & d > r + g \end{cases}$$

where $r$ is the redzone size and $g$ is the gap to the next valid allocation.

## 3. Quarantine and Use-After-Free Detection (Temporal Safety)

### The Problem

Use-after-free occurs when memory is accessed after being freed. ASAN detects this by delaying the reuse of freed memory via a quarantine queue. Freed memory is poisoned and placed in the quarantine; only when the quarantine reaches its size limit are old entries actually returned to the allocator.

### The Formula

The quarantine operates as a FIFO queue with size limit $Q$ (default 256 MB):

$$\text{quarantine} = [f_1, f_2, \ldots, f_k] \quad \text{where } \sum_{i=1}^{k} |f_i| \leq Q$$

When a new free of size $s$ occurs:
1. Poison shadow memory for $s$ bytes
2. Enqueue $f_{\text{new}}$ into quarantine
3. If $\sum |f_i| + s > Q$, dequeue oldest entries until under limit

The detection window for a use-after-free on allocation $a$:

$$t_{\text{detect}} = \frac{Q}{\text{free\_rate}} \text{ (time before } a \text{ leaves quarantine)}$$

For a program freeing 10 MB/s with 256 MB quarantine:

$$t_{\text{detect}} = \frac{256}{10} = 25.6 \text{ seconds}$$

Any access to $a$ within 25.6 seconds of freeing is detected. After that, the memory may be reallocated and the UAF becomes invisible.

The probability of detecting a UAF that occurs $\Delta t$ after free:

$$P(\text{detect}) = \begin{cases} 1 & \Delta t < \frac{Q}{\text{free\_rate}} \\ 0 & \Delta t \geq \frac{Q}{\text{free\_rate}} \end{cases}$$

### Worked Example

Program frees 1000 allocations per second, average size 1 KB. Quarantine size 256 MB.

Free rate: $1000 \times 1024 = 1$ MB/s.

Quarantine holds: $256 \times 1024 / 1 = 262{,}144$ freed allocations.

Detection window: 256 seconds.

A UAF on allocation freed at $t=0$ is guaranteed detected if the stale access occurs before $t=256$. In practice, most UAFs occur within milliseconds of the free, so the detection rate is very high.

## 4. Happens-Before and Data Race Detection (Partial Order Theory)

### The Problem

ThreadSanitizer detects data races: concurrent unsynchronized accesses to the same memory location where at least one is a write. Detection requires tracking the happens-before partial order established by synchronization primitives (locks, signals, atomic operations).

### The Formula

The happens-before relation $\to$ is the smallest partial order satisfying:

1. **Program order:** Within a thread, $a \to b$ if $a$ precedes $b$
2. **Synchronization:** For release $r$ and matching acquire $a$: $r \to a$
3. **Transitivity:** If $a \to b$ and $b \to c$, then $a \to c$

Two events $a$ and $b$ are concurrent (and potentially racy) iff:

$$a \| b \iff \neg(a \to b) \land \neg(b \to a)$$

A data race exists iff:

$$\exists a, b: \text{loc}(a) = \text{loc}(b) \land (a \| b) \land (\text{write}(a) \lor \text{write}(b))$$

TSAN tracks this using vector clocks. Each thread $T_i$ maintains a vector clock $VC_i[1..n]$ for $n$ threads:

$$VC_i[j] = \text{last known epoch of thread } j \text{ at thread } i$$

Synchronization update (lock release by $T_i$, acquire by $T_j$):

$$VC_j := \max(VC_j, VC_i) \quad \text{(component-wise maximum)}$$

A race is detected when accessing location $\ell$ if:

$$VC_{\text{access}}[\text{last\_writer}] < \text{last\_write\_epoch}$$

The space complexity for $n$ threads and $m$ monitored locations:

$$\text{space} = O(n \cdot m)$$

This explains TSAN's 5-10x memory overhead.

## 5. ASAN Instrumentation Check Complexity (Compiler Theory)

### The Problem

ASAN inserts a shadow memory check before every memory access. The check overhead per access is constant, but the total overhead depends on the density of memory accesses in the program. Understanding the check structure reveals why ASAN achieves only 2x slowdown despite checking every access.

### The Formula

For each memory access of size $k$ at address $a$, ASAN inserts:

```
shadow_value = *(int8_t*)(a >> 3 + SHADOW_OFFSET)
if (shadow_value != 0) {
    if ((a & 7) + k > shadow_value) {
        report_error(a, k);
    }
}
```

The fast path (shadow byte is 0, meaning all 8 bytes accessible) requires:

$$T_{\text{fast}} = T_{\text{shift}} + T_{\text{add}} + T_{\text{load}} + T_{\text{branch}} \approx 4 \text{ cycles}$$

The slow path (partial accessibility check):

$$T_{\text{slow}} = T_{\text{fast}} + T_{\text{mask}} + T_{\text{add}} + T_{\text{compare}} \approx 8 \text{ cycles}$$

For a program with $N$ memory accesses per second, the overhead:

$$\text{overhead} = \frac{N \cdot (P_{\text{fast}} \cdot T_{\text{fast}} + P_{\text{slow}} \cdot T_{\text{slow}})}{N \cdot T_{\text{original}}}$$

In practice, $P_{\text{fast}} \approx 0.95$ (most accesses hit fully-accessible 8-byte groups), giving:

$$\text{overhead} \approx \frac{0.95 \times 4 + 0.05 \times 8}{1} = 4.2 \text{ cycles per access}$$

With average memory access cost of 4 cycles (L1 hit): approximately 2x total slowdown.

## 6. Undefined Behavior Detection Coverage (Type Theory)

### The Problem

UBSAN checks for violations of the C/C++ language specification that produce undefined behavior. Each check targets a specific class of UB, and the coverage of all checks together determines what fraction of possible UB is caught at runtime.

### The Formula

The C11 standard defines approximately 200 categories of undefined behavior. UBSAN covers $k$ categories:

$$\text{coverage} = \frac{k}{200} \approx \frac{25}{200} = 12.5\%$$

However, the checked categories account for a disproportionate fraction of real-world bugs. By frequency in CVE databases:

| UB Category | UBSAN Check | CVE Frequency |
|:---|:---:|:---:|
| Signed integer overflow | Yes | 18% |
| Null pointer deref | Yes | 15% |
| Buffer overflow | Via ASAN | 25% |
| Use after free | Via ASAN | 12% |
| Shift overflow | Yes | 5% |
| Division by zero | Yes | 3% |
| Type confusion | Partial | 8% |
| Alignment violation | Yes | 2% |

The combined detection rate of ASAN + UBSAN for common vulnerability classes:

$$P(\text{detect} \mid \text{CVE}) \approx 0.88$$

This 88% detection rate for common vulnerability classes explains why the ASAN+UBSAN combination is the standard for security testing.

The per-check overhead of UBSAN varies:

$$T_{\text{check}} = \begin{cases} 1 \text{ cycle} & \text{overflow check (add + jo)} \\ 2 \text{ cycles} & \text{null check (test + jz)} \\ 3 \text{ cycles} & \text{alignment check (and + test + jnz)} \\ 10+ \text{ cycles} & \text{vptr check (vtable lookup + compare)} \end{cases}$$

Total UBSAN overhead: 10-20% in computation-heavy code, negligible in I/O-bound code.

---

*Sanitizers represent a fundamental engineering tradeoff: by accepting a constant-factor slowdown during testing, they provide deterministic detection guarantees for entire classes of memory safety and behavioral correctness violations that would otherwise manifest as silent corruption, exploitable vulnerabilities, or heisenbugs that resist reproduction.*

## Prerequisites

- Computer architecture (memory addressing, cache hierarchy, virtual memory)
- Compiler theory (instrumentation passes, code generation, shadow variables)
- Partial order theory (happens-before, vector clocks, concurrent event ordering)
- Type theory (undefined behavior categories, language specification compliance)
- Queueing theory (quarantine modeling, FIFO cache eviction)
- Probability theory (detection rates, coverage estimation)

## Complexity

- **Beginner:** Compiling with `-fsanitize=address`, interpreting crash reports, running Go race detector, understanding redzone concept
- **Intermediate:** Multi-sanitizer CI matrices, suppression file management, MSAN with full dependency rebuilds, TSAN for complex concurrent code
- **Advanced:** Custom sanitizer pass development, shadow memory scheme design, combining sanitizers with coverage-guided fuzzing, Miri for unsafe Rust verification, kernel-mode sanitizer deployment (KASAN)
