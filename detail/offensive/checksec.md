# The Mathematics of Checksec -- Exploit Mitigation Entropy and Attack Complexity

> *Each binary hardening mechanism multiplies the attacker's work factor by a quantifiable amount; the art of offensive security is finding the combination of bypasses whose total cost remains tractable.*

---

## 1. ASLR Entropy and Brute-Force Complexity (Probability Theory)

### The Problem

Address Space Layout Randomization shifts memory regions by a random offset at load time. The security of ASLR depends on the number of random bits (entropy) in each region's base address. An attacker who must guess the correct address faces an exponentially growing search space with each additional bit of entropy.

### The Formula

For a memory region with $b$ bits of entropy, the number of possible base addresses:

$$N = 2^b$$

The probability of a single correct guess:

$$P(\text{correct}) = \frac{1}{2^b}$$

Expected attempts for brute-force:

$$E[\text{attempts}] = 2^{b-1} \quad \text{(on average, half the space)}$$

For a forking server that does not re-randomize (child inherits parent layout), the cost is amortized:

$$E[\text{attempts}_{\text{fork}}] = 2^{b-1} \quad \text{(one-time cost, then reuse)}$$

With a crash rate limit of $r$ attempts per second:

$$T_{\text{brute}} = \frac{2^{b-1}}{r}$$

| Region | Bits (64-bit) | Bits (32-bit) | Time at 1000/s (64-bit) | Time at 1000/s (32-bit) |
|:---|:---:|:---:|:---:|:---:|
| Stack | 30 | 19 | 6.2 days | 4.4 min |
| mmap/libc | 28 | 8 | 1.6 days | 0.13 s |
| Heap | 13 | 5 | 4.1 s | 0.016 s |
| PIE base | 28 | 8 | 1.6 days | 0.13 s |

On 32-bit systems, ASLR is trivially brute-forceable. On 64-bit, information leaks are required.

### Worked Example

A 64-bit server with 28-bit mmap ASLR and no info leak. Attacker can crash and reconnect 10 times per second.

Expected time: $\frac{2^{27}}{10} = 13{,}421{,}773$ seconds = 155 days.

With an info leak (1 libc pointer): ASLR entropy drops to 0 bits. Attack time: 1 attempt = instant.

This demonstrates why preventing info leaks is more important than increasing ASLR entropy.

## 2. Stack Canary Entropy and Attack Models (Information Theory)

### The Problem

Stack canaries place a random value between the buffer and the saved return address. An overflow that modifies the return address must also corrupt the canary, which is checked before function return. The security depends on the canary's entropy and the attacker's ability to learn or bypass it.

### The Formula

A canary of $k$ random bytes (with 1 null byte) has entropy:

$$H = 8 \times (k - 1) = 8 \times 7 = 56 \text{ bits (on 64-bit systems)}$$

Brute-force probability per attempt:

$$P(\text{correct guess}) = \frac{1}{2^{56}} = 1.4 \times 10^{-17}$$

Byte-by-byte brute-force on a forking server (canary preserved across fork):

$$E[\text{attempts}] = 7 \times \frac{256}{2} = 896$$

This works because the attacker can overwrite one byte at a time:
- Guess byte 1: 128 average attempts (256/2), crash = wrong, no crash = correct
- Guess byte 2: 128 more attempts
- Total: $7 \times 128 = 896$ attempts

Compare to full brute-force without byte-by-byte:

$$E[\text{full brute}] = \frac{2^{56}}{2} = 3.6 \times 10^{16}$$

The byte-by-byte attack reduces complexity from $O(2^{56})$ to $O(7 \times 256) = O(1792)$.

### Worked Example

Forking server, 8-byte canary (7 random bytes + null terminator). Attacker can send 50 attempts per second.

Byte-by-byte: $896 / 50 = 17.9$ seconds average.

Full brute-force: $3.6 \times 10^{16} / 50 = 7.2 \times 10^{14}$ seconds = 22.8 million years.

The fork server model reduces the effective entropy from 56 bits to approximately 10 bits.

## 3. NX and Code Reuse Attack Surface (Computational Complexity)

### The Problem

NX/DEP prevents execution of injected shellcode by marking data pages as non-executable. Attackers must instead use code already present in the process (ROP, JOP, COP). The feasibility of code reuse depends on the available gadget density and the computational expressiveness of the gadget set.

### The Formula

For a binary with text segment of size $S$ bytes, the expected number of useful ROP gadgets:

$$|\mathcal{G}| \approx \alpha \cdot S$$

where $\alpha \approx 0.01-0.03$ is the gadget density (empirically measured). For x86-64, the variable-length instruction encoding means that unintended instruction boundaries yield "hidden" gadgets:

$$|\mathcal{G}_{\text{total}}| = |\mathcal{G}_{\text{intended}}| + |\mathcal{G}_{\text{unintended}}|$$

$$|\mathcal{G}_{\text{unintended}}| \approx 2.5 \times |\mathcal{G}_{\text{intended}}|$$

The minimum gadget set for Turing-complete computation:

$$|\mathcal{G}_{\text{min}}| = \{load, store, add, branch, syscall\} = 5$$

The probability that a binary of size $S$ contains all minimum gadgets:

$$P(\text{Turing complete}) = 1 - \prod_{g \in \mathcal{G}_{\text{min}}} (1 - p_g)^{|\mathcal{G}|}$$

For a 1 MB text segment: $|\mathcal{G}| \approx 20{,}000$ gadgets. The probability of finding all 5 minimum gadget types approaches 1.0 for any reasonably sized binary.

The work factor for constructing a ROP chain:

$$W_{\text{ROP}} = O(|\text{chain length}| \times |\mathcal{G}|)$$

With automated tools (ROPgadget, ropper, pwntools), chain construction is polynomial in gadget count.

## 4. RELRO and GOT Protection (Write Primitive Theory)

### The Problem

The Global Offset Table (GOT) contains function pointers resolved by the dynamic linker. Without RELRO, attackers with a write primitive can overwrite GOT entries to redirect function calls. Full RELRO marks the GOT read-only after startup, eliminating this attack surface but requiring all symbols to be resolved eagerly.

### The Formula

The value of the GOT as an attack target depends on the number of exploitable entries:

$$|\text{GOT entries}| = |\text{dynamic imports}|$$

Each GOT entry is a function pointer of size $w$ (8 bytes on 64-bit). The total GOT size:

$$|\text{GOT}| = w \times |\text{dynamic imports}|$$

A typical binary with 50 dynamic imports has a 400-byte GOT. The probability that a useful target function (system, execve, mprotect) is in the GOT:

$$P(\text{useful GOT entry}) = \frac{|\text{useful functions in GOT}|}{|\text{all GOT entries}|}$$

With Partial RELRO, the GOT is writable. Write-what-where cost:

$$W_{\text{GOT overwrite}} = W_{\text{write primitive}} + O(1) \quad \text{(single pointer write)}$$

With Full RELRO, the GOT is mmap'd read-only. The attacker must find alternative writable function pointers. Available alternatives:

| Target | Writable? | Glibc Version | Reliability |
|:---|:---:|:---:|:---:|
| `__malloc_hook` | Yes (removed 2.34) | < 2.34 | High |
| `__free_hook` | Yes (removed 2.34) | < 2.34 | High |
| `.fini_array` | Sometimes | All | Medium |
| `_IO_list_all` | Yes | All | Low (complex) |
| Stack return addr | Yes | All | High |
| vtable pointers | Yes | All (C++) | Medium |

Post glibc 2.34, Full RELRO significantly increases exploitation difficulty because the convenient hook targets were removed.

## 5. FORTIFY_SOURCE Bounds Checking (Static Analysis Theory)

### The Problem

FORTIFY_SOURCE replaces standard library functions with bounds-checked variants when the compiler can determine the destination buffer size at compile time. The effectiveness depends on how often the compiler can statically determine buffer sizes.

### The Formula

Let $F$ be the set of all calls to fortifiable functions, and $F_k \subseteq F$ be the calls where the compiler can determine the destination size. The coverage rate:

$$\text{coverage} = \frac{|F_k|}{|F|}$$

For each fortified call, the runtime check adds:

$$\text{check}: \text{if}(\text{copy\_size} > \text{dest\_size}) \text{ abort}()$$

The false negative rate (overflows not caught):

$$P(\text{miss}) = 1 - \text{coverage}$$

Empirical coverage measurements across common programs:

| Program | Total Calls | Fortified | Coverage |
|:---|:---:|:---:|:---:|
| OpenSSH | 342 | 198 | 57.9% |
| Apache httpd | 521 | 287 | 55.1% |
| nginx | 189 | 95 | 50.3% |
| coreutils | 2,847 | 1,876 | 65.9% |

FORTIFY_SOURCE level differences:

- **Level 1:** Only checks when the compiler knows the exact object size
- **Level 2:** Also checks sub-object boundaries (struct member sizes)
- **Level 3:** Uses `__builtin_dynamic_object_size` for runtime-computed sizes (GCC 12+)

$$\text{coverage}_1 < \text{coverage}_2 < \text{coverage}_3$$

Typical improvement: Level 2 catches 10-15% more cases than Level 1. Level 3 catches 20-30% more than Level 2.

## 6. Combined Mitigation Effectiveness (Attack Graph Theory)

### The Problem

Individual mitigations are bypassable, but their combination creates a layered defense where each bypass adds work. The total exploitation cost is the product of individual bypass costs, making the combination far more effective than any single mitigation.

### The Formula

For $n$ independent mitigations with bypass costs $C_1, C_2, \ldots, C_n$, the total exploitation work:

$$W_{\text{total}} = \prod_{i=1}^{n} C_i$$

With typical bypass costs:

| Mitigation | Bypass Cost | Requirement |
|:---|:---:|:---|
| No ASLR (32-bit) | $2^8 = 256$ | Brute force |
| ASLR (64-bit) | $2^{28}$ or info leak | Vulnerability |
| NX | $O(|\mathcal{G}|)$ | ROP gadgets |
| Stack Canary | $2^{56}$ or leak | Info leak |
| Full RELRO | Find alt target | Write primitive |
| PIE | $2^{28}$ or leak | Same as ASLR |
| FORTIFY | Find unfortified call | Code audit |

For a fully hardened 64-bit binary (PIE + Full RELRO + NX + Canary + FORTIFY + ASLR):

Without info leaks: $W \approx 2^{28} \times 2^{56} \times O(|\mathcal{G}|) = 2^{84+}$ (intractable)

With one info leak: $W \approx 1 \times 1 \times O(|\mathcal{G}|) \approx O(1000)$ (trivial after leak)

This demonstrates the critical role of info leaks: a single pointer leak collapses the security of ASLR, PIE, and canaries simultaneously, reducing the combined defense to just NX (requires ROP) and RELRO (requires alternative write target).

### Worked Example

CTF binary: 64-bit, NX enabled, Partial RELRO, No PIE, No Canary.

Attack cost:
- ASLR bypass via fixed PLT (no PIE): cost 1
- NX bypass via ROP: cost O(|gadgets|) = manageable
- Canary bypass: not needed (no canary)
- GOT overwrite: cost 1 (Partial RELRO)

Total: trivially exploitable with a stack buffer overflow.

Same binary with Full RELRO + PIE + Canary:
- Need info leak for PIE + canary: cost = vulnerability required
- Need alt write target for Full RELRO: cost = research
- Still need ROP for NX: cost = gadget search

Total: requires at least 2 vulnerabilities (overflow + info leak).

---

*Binary hardening is a numbers game: each mitigation multiplies the attacker's work factor, but the multiplication collapses to addition in log-space, and a single information leak subtracts the largest terms. The defender's goal is not to make exploitation impossible but to make it expensive enough that the attacker moves to softer targets.*

## Prerequisites

- Probability theory (entropy, brute-force complexity, birthday bounds)
- Information theory (bits of randomness, information leakage quantification)
- Computational complexity (polynomial vs. exponential bypass costs)
- Computer architecture (virtual memory, page permissions, TLB, MMU)
- Compiler theory (static analysis capabilities, object size tracking)
- Graph theory (attack graphs, exploit chain modeling)

## Complexity

- **Beginner:** Running checksec, understanding output fields, compiling with hardening flags, checking ASLR status
- **Intermediate:** Calculating ASLR entropy, identifying bypass techniques for each mitigation, building multi-stage exploits against partially hardened binaries
- **Advanced:** Quantifying combined mitigation effectiveness, developing novel bypass techniques for emerging mitigations (CFI, shadow stacks, CET), kernel-level KASLR analysis
