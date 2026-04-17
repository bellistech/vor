# The Mathematics of WebAssembly — Type Systems, Sandbox Guarantees, and Compilation Theory

> *WebAssembly is not merely a binary format; it is a formally specified virtual instruction set with a denotational semantics, a sound type system, and provable isolation properties. The security of WASI capabilities, the correctness of stack-machine execution, the efficiency of tier-up JIT compilation, and the composability of the Component Model all reduce to mathematical structures: lattices, type lattices, abstract interpretation, and category-theoretic composition.*

---

## 1. The Stack Machine — Operational Semantics

### The Problem

WebAssembly executes on a typed stack machine. Every instruction has a precise operational effect described by transitions over an abstract machine state. Reasoning about correctness requires formalizing these transitions.

### The Formula

The execution state is a tuple:

$$\sigma = \langle s, f, c \rangle$$

Where $s$ is the value stack, $f$ is the call frame (locals + module instance), $c$ is the code continuation.

Each instruction $i$ induces a transition $\sigma \xrightarrow{i} \sigma'$. For example, `i32.add`:

$$\frac{s = s' \cdot v_1 \cdot v_2 \quad v_1, v_2 : \texttt{i32}}{\langle s, f, \texttt{i32.add} \cdot c \rangle \to \langle s' \cdot (v_1 + v_2 \bmod 2^{32}), f, c \rangle}$$

For `br $l` (branch to label $l$), unwind until the $l$-th enclosing label, restoring the stack to the label's declared arity.

A program is a composition of such rules; the WebAssembly specification provides a small-step operational semantics covering every instruction, which has been mechanized in Isabelle/HOL (Watt, 2018) and Coq (WasmCert).

### Worked Example

Execute `(i32.const 40) (i32.const 2) (i32.add)`:

$$\langle \varnothing, f, [40, 2, \text{add}] \rangle \to \langle [40], f, [2, \text{add}] \rangle$$
$$\to \langle [40, 2], f, [\text{add}] \rangle \to \langle [42], f, [\, ] \rangle$$

Terminates with result 42 on the stack.

### Why It Matters

Because the semantics are formal, properties like "execution never exceeds declared stack depth" are theorems, not hopes. This is the foundation for sandbox guarantees.

---

## 2. Type Soundness — Progress and Preservation

### The Problem

Wasm modules are validated before execution. The type system must guarantee that validated modules do not "go wrong" — i.e., stack underflow, type mismatch, or branch to nonexistent labels cannot occur at runtime.

### The Formula

For a validation judgment $C \vdash i : \tau_1^* \to \tau_2^*$ (instruction $i$ consumes stack type $\tau_1^*$ and produces $\tau_2^*$ under context $C$):

**Progress**: if $\vdash \sigma \text{ OK}$ and $\sigma$ is not a terminal state, then there exists $\sigma'$ with $\sigma \to \sigma'$.

**Preservation**: if $\vdash \sigma \text{ OK}$ and $\sigma \to \sigma'$, then $\vdash \sigma' \text{ OK}$.

Together: a validated module never reaches a stuck state. Indirect calls are checked against runtime table types; out-of-bounds memory accesses trap deterministically.

**Theorem (Type Soundness, Rossberg et al.)**: if $\vdash M \text{ OK}$ then every execution of $M$ either diverges or reduces to a value or a trap — never to a malformed state.

### Worked Example

Consider attempting to add an `f32` and `i32`:

```wat
(func (result f32)
  f32.const 1.5
  i32.const 2
  f32.add)   ;; validation error
```

Validator computes stack type after each instruction:

- After `f32.const 1.5`: stack type $[\text{f32}]$
- After `i32.const 2`: stack type $[\text{f32}, \text{i32}]$
- `f32.add` expects $[\text{f32}, \text{f32}]$ → TYPE ERROR

Validation rejects the module before execution. No runtime check needed.

### Why It Matters

Dynamic languages embedded in Wasm (JS, Python via CPython-Wasm) lose type checking internally, but the Wasm host *itself* cannot be subverted by a malformed module — type soundness is enforced at the ISA level.

---

## 3. Linear Memory — Bounds and Capability Algebra

### The Problem

Each Wasm module has a linear memory $M: \{0, 1, \ldots, 2^{32} - 1\} \to \text{byte}$ (or 64-bit in `memory64` proposal). All memory access must be verified to not escape its bounds.

### The Formula

A memory access `i32.load offset=k align=a` at address $x$ succeeds iff:

$$x + k + \text{size}(\tau) \leq |M|$$

Otherwise it traps. Growing memory is monotone:

$$M_0 \subseteq M_1 \subseteq M_2 \subseteq \ldots$$

Memory size is always a multiple of the page size ($2^{16}$ bytes = 64 KiB).

Capability algebra for multiple modules:

$$\text{caps}(I) = \{\text{import}(I, x) : x \in \text{exports}\}$$

Host caps available to instance $I$ form a set; module has no ambient authority beyond $\text{caps}(I)$.

### Worked Example

Module has 1 page memory (65536 bytes). Instruction `i32.load offset=65534`:

- Size of i32 = 4 bytes
- Required: $x + 65534 + 4 \leq 65536$
- Only $x = 0$ succeeds → trap on any other $x$

No address sanitizer needed; no ASLR needed; no buffer overflow that escapes the module's own memory. The module cannot even *observe* memory outside its linear space because there is no instruction to address it.

### Why It Matters

This is why Wasm sandboxes achieve what Docker and Firecracker achieve at much higher cost. The security boundary is enforced by the ISA itself, not by a separate kernel.

---

## 4. Compilation — Tier-Up and Abstract Interpretation

### The Problem

Wasm bytecode must be compiled to native machine code. Two-tier JITs balance compile time against runtime throughput. Correctness requires that optimizations preserve the operational semantics.

### The Formula

Baseline compiler $\beta$: compiles each instruction locally with near-zero analysis. Cost model:

$$T_{\text{compile}}^\beta(M) = O(|M|), \quad T_{\text{execute}}^\beta = O(k \cdot T_{\text{ideal}})$$

Where $k \approx 2$–3 is the baseline slowdown factor.

Optimizing compiler $\Omega$ (e.g., Cranelift): performs SSA construction, register allocation, dead code elimination, instruction selection. Cost:

$$T_{\text{compile}}^\Omega(M) = O(|M|^{1.5}), \quad T_{\text{execute}}^\Omega \approx T_{\text{ideal}}$$

Correctness via **abstract interpretation**: each optimization is sound iff its abstract transfer function $\alpha$ satisfies:

$$\alpha(f(x)) \sqsubseteq f^\#(\alpha(x))$$

Where $\sqsubseteq$ is the abstract lattice order. Register allocation preserves the value-level semantics by construction when the colored graph has no spills; with spills, proof obligations are discharged via translation validation.

### Worked Example

Wasmtime Cranelift pipeline:

1. Parse Wasm → CLIF (Cranelift IR)
2. Legalize: lower unsupported operations to target primitives
3. Regalloc: graph coloring with live-range splitting
4. Machine code emission

For a 1 MB Wasm module:
- Baseline: 50 ms compile, ~60% of native speed
- Cranelift: 300 ms compile, ~95% of native speed

AOT compile (`wasmtime compile`) produces `.cwasm` with code cache ready; load time drops to ~50 µs because Cranelift's output is already native machine code.

### Why It Matters

Startup-latency-dominated workloads (edge functions, serverless, plugins) use AOT artifacts to achieve cold starts 100× faster than container cold starts. The math of compilation economics favors Wasm for such workloads.

---

## 5. Capability-Based Security — The POLA Lattice

### The Problem

WASI embodies the Principle of Least Authority (POLA). Capabilities form a lattice where each instance receives the minimum set required for its task.

### The Formula

Capabilities form a partial order $(C, \sqsubseteq)$ where $c_1 \sqsubseteq c_2$ means $c_1$ is weaker than or equal to $c_2$.

Example capability lattice for filesystem:

```
              {rw/}  (all dirs, read+write)
             /      \
         {rw/data}  {r/}
         /          /
     {r/data}   {r/data/public}
          \    /
           {}  (no fs access)
```

An instance $I$ with capability set $K_I$ can perform action $a$ iff $\text{required}(a) \sqsubseteq K_I$.

**Theorem (Capability confinement)**: if $I_1$ has capability set $K_1$ and calls into $I_2$ passing capabilities $K_{\text{pass}} \subseteq K_1$, then $I_2$'s authority is bounded by $K_2 \cup K_{\text{pass}}$. There is no ambient way for $I_2$ to acquire capabilities from $I_1$ that weren't explicitly passed.

### Worked Example

A Wasm web server component is instantiated with:

$$K = \{\text{wasi:sockets}/\text{tcp-bind: 8080}, \text{wasi:filesystem}/\text{r: /static}\}$$

It receives a request and calls a plugin component with:

$$K_{\text{pass}} = \{\text{wasi:filesystem}/\text{r: /static/plugins}\}$$

Plugin cannot:
- Write to filesystem (no write capability passed)
- Open sockets (no socket capability passed)
- Read /etc/passwd (not in its capability set)

This is enforced at link time, not at runtime via permission checks.

### Why It Matters

POLA lattices provide **provable** confinement, not best-effort. A malicious plugin cannot escape its capability bounds because the runtime simply lacks the import bindings to reach restricted resources.

---

## 6. The Component Model — Category-Theoretic Composition

### The Problem

The Component Model allows polyglot composition: a Rust component can link to a JavaScript component linked to a Python component. Type marshaling across languages must be sound and automatic.

### The Formula

WIT interfaces define morphisms between types in a category where objects are types and morphisms are conversions. For languages $L_1, L_2$, a conversion $\phi_{L_1 \to L_2}: T_{L_1} \to T_{L_2}$ must satisfy:

- **Identity**: $\phi_{L \to L}(x) = x$
- **Composition**: $\phi_{L_2 \to L_3} \circ \phi_{L_1 \to L_2} = \phi_{L_1 \to L_3}$

The canonical ABI (canonical lifting and lowering) defines these morphisms formally in the Component Model spec, reducing all high-level types to a core set (primitives, records, variants, lists, strings, resources).

Resource types use linear typing: a resource handle can be used exactly once unless explicitly duplicated, modeling opaque references safely.

### Worked Example

A WIT interface:

```wit
interface image {
  resource image-buffer {
    constructor(width: u32, height: u32);
    get-pixel: func(x: u32, y: u32) -> u32;
    set-pixel: func(x: u32, y: u32, color: u32);
  }
  render: func(buf: borrow<image-buffer>) -> list<u8>;
}
```

A Rust producer implements `image-buffer` as `Arc<Mutex<Vec<u32>>>`. A JavaScript consumer receives a handle (an integer index into the resource table) and calls `set-pixel(handle, x, y, color)`. The canonical ABI marshals arguments through linear memory and calls the Rust-compiled code.

No hand-written FFI code is required. The Component Model proves the composition is type-safe at both ends.

### Why It Matters

Polyglot systems historically required fragile manual bindings (FFI, gRPC + IDL compilers, JNI). The Component Model replaces this with a categorical composition discipline: interfaces compose, types marshal automatically, and the glue is formally specified.

---

## 7. Synthesis — Why the Math Actually Matters

WebAssembly is one of the most mathematically principled systems in wide industrial deployment:

| Property | Mathematical tool | Practical consequence |
|----------|-------------------|-----------------------|
| Execution correctness | Small-step operational semantics | Reasoning about runtime behavior |
| Sandbox integrity | Type soundness (progress + preservation) | No sandbox escapes via malformed code |
| Memory safety | Bounds algebra on linear memory | No buffer overflows escape module |
| Compilation correctness | Abstract interpretation | Optimizations preserve semantics |
| Security confinement | Capability lattice + POLA | Provable least-authority execution |
| Polyglot composition | Category-theoretic morphisms | Safe cross-language linking |

The reason Wasm replaces containers for edge functions is microsecond startup. The reason it replaces browser plugins is memory safety. The reason it will replace language-embedded extension systems (Lua, JS-in-apps) is the capability model. Each of these is a mathematical property translated into engineering delivery.

The substrate is theoretically sound. Adoption follows.

---
