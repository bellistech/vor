# The Internals of WebAssembly — Type System, Validation, Execution, and the Component Model

> *WebAssembly (Wasm) is a low-level, statically-typed, stack-based virtual machine with a binary instruction format, a structured control flow discipline, a deterministic execution model, and a rigorous specification that has been mechanized in Coq, Isabelle/HOL, and K. Beneath every "compile Rust to wasm" tutorial sits a precise abstract machine: a value stack, a frame stack, linear memories, tables, globals, and a validation algorithm that proves type-safety before a single instruction executes. This deep-dive traces what the practical sheet skips: the binary layout byte-by-byte, the validation type-checker, the execution semantics, the Component Model and WIT IDL, the WASI Preview1-vs-Preview2 transition, the producer toolchains for Rust/C/Go, and the host integration with browsers and standalone runtimes.*

---

## 1. The Wasm Abstract Machine

### Stack-Based With Structured Control Flow

WebAssembly is a stack machine, but it is **not** a free-form stack machine like Forth or the JVM's bytecode lineage. Wasm's control flow is **structured**: there is no arbitrary `goto`, no computed jump target, and no fall-through between blocks. Instead, control flow is expressed using nested `block`, `loop`, and `if` constructs, with branches (`br`, `br_if`, `br_table`) targeting **labels** introduced by those constructs.

This restriction is deliberate. Structured control flow makes:

- **Validation** linear-time and decidable.
- **Compilation to native code** straightforward (every Wasm function maps to a reducible CFG).
- **Static analysis** (e.g., for security tools) tractable.
- **Decompilation** into human-readable forms (e.g., back to C-like pseudocode) reasonable.

The price: irreducible control flow that exists naturally in some languages (Go's `goto`, certain optimizer outputs, JavaScript's `try/finally`) requires re-encoding via the **Relooper** algorithm or the newer **Stackifier**. Both are well-understood transformations that recover structured CFGs from arbitrary CFGs.

### The Value Stack

Each function executes against a **value stack**: a LIFO of typed values. Instructions consume operands from the stack and produce results back on the stack.

```wat
(func (result i32)
  i32.const 40    ;; stack: [40]
  i32.const 2     ;; stack: [40, 2]
  i32.add)        ;; stack: [42]
```

The validator tracks the *types* on this stack, not the values. After validation, the runtime knows that types match — so the JIT can emit unchecked native code.

### Deterministic Execution

Wasm is **deterministic** modulo a small set of clearly-specified non-determinism:

- Floating-point NaN bit patterns may differ between hosts (the spec allows multiple "canonical" encodings).
- The result of `memory.grow` may fail or succeed depending on host memory pressure.
- Resource exhaustion (stack overflow, OOM) is host-dependent.

Otherwise, `(i32.const 1) (i32.const 2) (i32.add)` produces 3 on every conforming engine, on every architecture, in every browser. This matters for blockchain (where every node must compute the same result) and for reproducible builds.

### Sandboxing Guarantees

The abstract machine guarantees:

- **Memory safety**: every load/store is bounds-checked against linear memory size.
- **Control-flow integrity**: indirect calls (`call_indirect`) are checked against a runtime type signature.
- **Stack safety**: the value stack is not addressable; there is no "stack-smashing" within Wasm.
- **No ambient authority**: a Wasm module can only call host functions explicitly imported.

These are enforced **by construction**, not by runtime checks atop a permissive ISA. That is why a Wasm sandbox is fundamentally different from running native code in a process: an unsafe-Rust panic inside Wasm cannot escape the sandbox in any way that breaks the host.

---

## 2. The Type System

### Numeric Types

Wasm has four numeric types:

| Type  | Bytes | Description                          |
|-------|-------|--------------------------------------|
| `i32` | 4     | 32-bit integer (signed/unsigned per op) |
| `i64` | 8     | 64-bit integer (signed/unsigned per op) |
| `f32` | 4     | 32-bit IEEE 754 float                |
| `f64` | 8     | 64-bit IEEE 754 float                |

Note that Wasm has no separate `u32` / `u64` — signedness is encoded **per-instruction**. `i32.div_s` is signed divide, `i32.div_u` is unsigned divide, both operating on the same `i32` storage type.

### Vector Type (SIMD)

The `simd128` proposal (now stable as part of Wasm 2.0) introduces:

| Type   | Bytes | Description                                   |
|--------|-------|-----------------------------------------------|
| `v128` | 16    | 128-bit packed vector (lane-typed by op)      |

A single `v128` is interpreted as 16×i8, 8×i16, 4×i32, 2×i64, 4×f32, or 2×f64 depending on the instruction (`i32x4.add`, `f32x4.mul`, etc.).

### Reference Types

The reference-types proposal (stable) adds:

| Type        | Description                                          |
|-------------|------------------------------------------------------|
| `funcref`   | Opaque reference to a Wasm function                  |
| `externref` | Opaque reference to a host-managed value             |

Reference types are **opaque**: their bit-level representation is not exposed; you cannot dereference, compare bit-wise, or convert to numeric. They are passed by value in the abstract machine but stored as host pointers internally.

### Function Types

A function type is a signature mapping parameter types to result types:

```
FT ::= (param t*) -> (result t*)
```

Multi-value returns are stable: `(func (result i32 i64))` is legal and returns two values.

```wat
(func $divmod (param i32 i32) (result i32 i32)
  local.get 0
  local.get 1
  i32.div_s
  local.get 0
  local.get 1
  i32.rem_s)
```

Caller pops both results: `(call $divmod) ;; stack now has q, r`.

### Block Types (Structured Control Flow Types)

Each `block`/`loop`/`if` declares a **block type** describing the stack arity entering and leaving the block:

```wat
(block (result i32)             ;; produces 1 i32
  i32.const 42)

(block (param i32) (result i32) ;; consumes 1 i32, produces 1 i32
  i32.const 1
  i32.add)
```

Block types reuse the function-type encoding, so they may consume and produce arbitrary stacks (with multi-value).

### Reference: WebAssembly Specification §3 Validation

The spec's §3 defines, for every instruction, the **input stack type** required and the **output stack type** produced. The validator walks instructions, simulating the type stack, rejecting any program that fails these constraints. We expand this into Section 5.

---

## 3. Module Structure

### The Binary Layout

A `.wasm` binary is a sequence of bytes:

```
\0asm           4 bytes  magic
0x01000000      4 bytes  version (currently 1)
section*        variable repeated sections
```

After the magic+version header, the file is a sequence of sections. Each section has the form:

```
section_id  byte
size        ULEB128 (size of payload in bytes)
payload     `size` bytes
```

### Section IDs (Stable)

| ID  | Name        | Contents                                                  |
|-----|-------------|-----------------------------------------------------------|
| 0   | Custom      | Names, DWARF, producers, etc. — not validated semantically |
| 1   | Type        | Function types (signatures) used by the module            |
| 2   | Import      | Module + name + descriptor for each imported function/table/memory/global |
| 3   | Function    | Type indices for each defined function                    |
| 4   | Table       | Table declarations (count, element type, limits)          |
| 5   | Memory      | Memory declarations (limits in pages)                     |
| 6   | Global      | Global variable declarations (mutability, type, init)     |
| 7   | Export      | Exported names → descriptors                              |
| 8   | Start       | Optional `start` function index                           |
| 9   | Element     | Table initialization data                                 |
| 10  | Code        | Function bodies (locals + bytecode)                       |
| 11  | Data        | Linear memory initialization                              |
| 12  | DataCount   | Number of data segments (bulk-memory proposal)            |

### Section Order

Non-custom sections **must** appear in the order above (1, 2, 3, ..., 11). Custom sections (ID 0) may appear anywhere, multiple times. The DataCount section, if present, must precede the Code section so streaming validators can verify `memory.init` instructions on the fly.

### The .wat Text Format

Every binary has an equivalent **WebAssembly Text Format** (.wat) — an S-expression syntax used for hand-written modules and disassembly:

```wat
(module
  (type $t0 (func (param i32 i32) (result i32)))
  (func $add (type $t0)
    local.get 0
    local.get 1
    i32.add)
  (export "add" (func $add)))
```

`wat2wasm` (from wabt) compiles text to binary; `wasm2wat` decompiles binary to text. Round-trip is lossless modulo formatting and synthetic labels.

### Imports and Exports

Imports specify a *module name* and *field name*, plus a descriptor:

```wat
(import "env" "log" (func $log (param i32)))
(import "env" "memory" (memory 1 10))
```

Imports are resolved at instantiation time by the host's `importObject` (in browsers) or by the runtime's linker (e.g., wasmtime CLI).

Exports expose internal items by name:

```wat
(export "add"  (func   $add))
(export "memory" (memory $mem))
(export "table"  (table  $tbl))
(export "global" (global $g))
```

### Custom Sections

The spec deliberately permits arbitrary user-defined custom sections. Conventional ones:

| Section name       | Purpose                                    |
|--------------------|--------------------------------------------|
| `name`             | Function/local/symbol names for debugging  |
| `producers`        | Toolchain attribution                      |
| `target_features`  | CPU features required (e.g., simd128)      |
| `.debug_*`         | DWARF debug info                           |
| `linking`          | Relocations for static linking with wasm-ld |

Tools may strip or rewrite custom sections without affecting semantics.

---

## 4. Instructions — Categorized

### Numeric Integer Operations

Per integer width (i32, i64), the following operations are stable:

| Family       | Instructions                                                      |
|--------------|-------------------------------------------------------------------|
| Constants    | `iN.const c`                                                      |
| Arithmetic   | `iN.add`, `iN.sub`, `iN.mul`, `iN.div_s`, `iN.div_u`, `iN.rem_s`, `iN.rem_u` |
| Bitwise      | `iN.and`, `iN.or`, `iN.xor`, `iN.shl`, `iN.shr_s`, `iN.shr_u`, `iN.rotl`, `iN.rotr` |
| Comparison   | `iN.eqz`, `iN.eq`, `iN.ne`, `iN.lt_s`, `iN.lt_u`, `iN.le_s`, `iN.le_u`, `iN.gt_s`, `iN.gt_u`, `iN.ge_s`, `iN.ge_u` |
| Bit counting | `iN.clz`, `iN.ctz`, `iN.popcnt`                                   |

`div_s` traps on division-by-zero **and** on integer-overflow-on-trunc (the `INT_MIN / -1` case for signed divide). `div_u` only traps on division-by-zero.

`clz` / `ctz` count leading / trailing zeros; `popcnt` counts set bits — all population-count-style operations are single instructions on the abstract machine.

### Numeric Float Operations

```
fN.const c
fN.add  fN.sub  fN.mul  fN.div
fN.eq   fN.ne   fN.lt   fN.le   fN.gt   fN.ge
fN.sqrt fN.min  fN.max  fN.abs  fN.neg
fN.ceil fN.floor fN.trunc fN.nearest
fN.copysign
```

`min`/`max` follow IEEE 754 semantics including NaN propagation. `nearest` is round-to-even (banker's rounding).

### Conversions Between Numeric Types

```
i32.wrap_i64                  ;; truncate i64 to low 32 bits
i64.extend_i32_s              ;; sign-extend i32 to i64
i64.extend_i32_u              ;; zero-extend i32 to i64
f32.convert_i32_s             ;; signed int → f32
f32.convert_i32_u             ;; unsigned int → f32
f64.convert_i64_s
i32.trunc_f32_s               ;; truncate f32 → i32 (TRAPS on out-of-range or NaN)
i32.trunc_sat_f32_s           ;; saturating trunc (no trap; clamps)
i32.reinterpret_f32           ;; bit-level reinterpret
f32.reinterpret_i32
```

The `trunc_sat_*` family is from the non-trapping-fp-to-int proposal (stable); it removes a frequent trap source for languages compiling math-heavy code.

### Control Flow

```
nop                           ;; do nothing
unreachable                   ;; trap unconditionally
block bt instr* end           ;; structured block
loop  bt instr* end           ;; loop (branch targets back to `loop`)
if bt instr* else instr* end  ;; conditional
br      label                 ;; unconditional branch to label
br_if   label                 ;; conditional branch (pops i32; nonzero = take)
br_table l_1...l_n l_default  ;; computed branch (pops index; jumps to l_index or l_default)
return                        ;; pop and return from function
```

`unreachable` is the canonical "this should never happen" instruction — Rust's `panic!()` lowers to it (after the panic-handling preamble, in `panic = "abort"` builds).

### Function Calls

```
call $f                       ;; direct call (function index in module)
call_indirect (type $T)       ;; pop i32 index, look up in table, type-check
return_call $f                ;; tail call (proposal — stable in Wasm 3.0 draft)
return_call_indirect (type $T)
```

Tail-call instructions reuse the caller's stack frame, allowing deep recursion in functional languages without stack overflow. Status: **Phase 4 (stable)** as of late 2024.

### Variables (Locals and Globals)

```
local.get  $idx               ;; push local
local.set  $idx               ;; pop and store
local.tee  $idx               ;; pop, store, push (= set + get without re-pop)
global.get $idx
global.set $idx               ;; only if global is mutable
```

Locals are addressable only by index inside a function. Globals are module-level and may be exported.

### Memory Instructions

```
i32.load    offset=N align=A   ;; load 4 bytes at addr+N
i32.store   offset=N align=A
i32.load8_s offset=N align=A   ;; load 1 byte, sign-extend to i32
i32.load8_u offset=N align=A   ;; load 1 byte, zero-extend
i32.load16_s
i32.load16_u
i32.store8                     ;; truncating store
i32.store16
i64.load   offset=N align=A
i64.store  offset=N align=A
;; ... and i64.load8/16/32_s/u, store8/16/32 variants
f32.load
f32.store
f64.load
f64.store

memory.size                    ;; pages of current memory
memory.grow                    ;; grow by N pages (returns prior size, or -1 on failure)
;; bulk-memory proposal (stable):
memory.copy                    ;; src, dst, len
memory.fill                    ;; dst, val, len
memory.init $data              ;; init from data segment
data.drop  $data               ;; mark data segment exhausted
```

The `memarg` `(offset N, align A)` is part of the instruction encoding. `align` is a **hint** to the engine that the address is aligned to `2^A` bytes — engines may emit faster unaligned-or-aligned code accordingly. Wrong alignment is **not** a trap; it is merely a missed optimization.

All multi-byte loads/stores are **little-endian**.

### Reference-Type Instructions

```
ref.null  funcref              ;; push null funcref
ref.null  externref
ref.is_null                    ;; pop ref; push 1 if null else 0
ref.func  $f                   ;; push funcref to function $f
table.get $t
table.set $t
table.size $t
table.grow $t
table.fill $t
table.copy $dst $src
table.init $tbl $elem
elem.drop  $elem
```

---

## 5. Validation Algorithm

### The Type Stack

Validation walks the instruction sequence of each function body, simulating a **type stack** of `valtype` (i32/i64/f32/f64/v128/funcref/externref). For each instruction, it:

1. Looks up the instruction's expected input types (top-of-stack pattern).
2. Pops those types from the type stack (errors if too few or wrong types).
3. Pushes the instruction's output types.

### Per-Block Stack Trace

Each `block`/`loop`/`if` introduces a **control frame** with:

- The block's parameter types (initial stack contents).
- The block's result types (required stack contents at end).
- The label arity (for branches into this block).
- A flag indicating whether this point is reachable.

The validator maintains a stack of control frames. A `br L` is type-checked by looking up frame `L` and demanding that the current value-stack contains at least the label's arity, with matching types.

### The Polymorphic Stack at `unreachable`

After `unreachable`, the value stack becomes **polymorphic**: it can be assumed to contain any types needed by subsequent instructions, until the next `end` or `else`. This lets the validator accept unreachable code without false errors:

```wat
(func (result i32)
  unreachable
  i32.add)        ;; type-checks even though no operands were pushed:
                   ;; the polymorphic stack supplies them
```

This is a subtle but important rule that makes validation compositional (the type checker doesn't have to prove unreachable-code consistency).

### The Canonical Type-Mismatch Error

A simple example of validation failure:

```wat
(func (result f32)
  f32.const 1.5
  i32.const 2
  f32.add)         ;; ERROR: f32.add expects [f32, f32] but stack is [f32, i32]
```

After `f32.const 1.5`: type stack = [f32].
After `i32.const 2`: type stack = [f32, i32].
`f32.add` requires [f32, f32]. Topmost is i32. Validation rejects.

### `wasm-validate` and Engine Integration

The reference command `wasm-validate file.wasm` runs exactly this algorithm. Browser engines run it as the first phase of compilation; standalone runtimes (wasmtime, wasmer) run it before instantiation. Validation is **linear-time**: each instruction is examined exactly once.

### Why Validation Makes Wasm Safe to Run Untrusted

Once a module is validated:

- No load/store can be type-confused.
- No `call_indirect` can call a function with the wrong signature (this is **also** runtime-checked, but only because the table can be mutated; the static path through validation guarantees the type signature).
- No branch can target an invalid label.
- No stack underflow can occur within a function.

These guarantees are why browsers will execute Wasm from arbitrary origins. The same property is what underpins Wasm's use as an extension format (Envoy filters, Shopify Functions, Fastly Compute@Edge, Cloudflare Workers' Wasm option, FiberPlane).

---

## 6. Execution Model

### The Instance

A **module** is the static artifact (the `.wasm` file). An **instance** is a runtime entity created by instantiating a module against an `importObject`. An instance owns:

- One linear memory (or zero, or — with multi-memory proposal — many).
- Zero or more tables (function tables for `call_indirect`).
- Globals, mutable or immutable.
- A function-table for direct calls.
- A reference to the module's data and element segments.
- Hosts may also attach engine-specific structures (compiled code, JIT caches).

### Frames and the Call Stack

When a Wasm function is called, a **frame** is pushed:

```
frame { locals[], module_instance }
```

Locals are initialized from the call's arguments (first N locals) plus zero-initialized declared locals. The frame stays alive until the function returns.

The frame stack is **internal to the engine** and not addressable from Wasm — there are no instructions that read frames or modify the call stack except `call`, `call_indirect`, `return`, and tail-calls.

### The Value Stack

Within a frame, instructions modify the **value stack**, also internal to the engine. There is no instruction to introspect the value stack as a whole.

### Traps

A **trap** is a non-recoverable abnormal termination of execution. The engine raises a host-level exception (in JS, this is `WebAssembly.RuntimeError`; in wasmtime, a Rust `Trap` enum). Traps are deterministic: the same input always traps at the same instruction. Sources:

| Source                                | Instruction                       |
|---------------------------------------|-----------------------------------|
| Integer divide-by-zero                | `iN.div_s`, `iN.div_u`, `iN.rem_s`, `iN.rem_u` |
| Signed-overflow on truncation         | `iN.div_s` with INT_MIN / -1      |
| Float→int out-of-range or NaN         | `iN.trunc_fM_s`, `iN.trunc_fM_u` (use `_sat_` to avoid) |
| Out-of-bounds memory access           | `iN.load*`, `iN.store*`, `memory.copy/fill/init` |
| Out-of-bounds table access            | `call_indirect` with bad index, `table.get`, `table.set` |
| `call_indirect` type mismatch         | When the table entry's type ≠ declared `(type $T)` |
| `unreachable` instruction             | Always traps                      |
| Stack overflow (host-defined limit)   | Any call                          |
| `ref.as_non_null` of a null reference | (with reference-types extensions) |

### Trap vs Throw Distinction

Traps are distinct from the **exception-handling proposal** (`try`/`catch`/`throw`). Traps are unrecoverable inside Wasm; only the host can observe and recover. Throws (when the proposal is enabled) are **catchable** within Wasm itself. Most current production Wasm uses the trap model exclusively; exception-handling is gradually shipping (Phase 4 for the legacy form, the new "exception-handling-with-tags" proposal is Phase 3 → 4).

### The Start Function

A module may declare a `start` function — invoked automatically at instantiation, after imports are resolved but before any export is callable. Used for:

- Initializing globals from imports.
- Validating runtime invariants.
- Calling C++-style static constructors (in Emscripten output).

```wat
(func $init ...)
(start $init)
```

---

## 7. Memory Model — Linear Memory

### Single-Instance for MVP

A Wasm 1.0 module has **at most one** linear memory. Multi-memory (Phase 4, shipping) lifts this restriction; for now most modules have exactly one.

### Page = 64KiB

Memory is allocated in **pages** of 65536 bytes (2^16). The MVP allows up to 65536 pages (4 GiB total); the `memory64` proposal extends this to 2^48 pages.

### Memory Operations

```wat
memory.size                  ;; pushes current size in pages (i32)
memory.grow                  ;; pops delta, pushes prior size or -1 on failure
```

`memory.grow` semantics:

- Pops the requested delta in pages.
- If the new size exceeds the declared maximum (or host limit), pushes `-1` and does not modify memory.
- Otherwise, allocates new pages (zero-initialized) and pushes the **prior** page count.

Engines may grow by `realloc`-style copy or by extending in-place via virtual memory tricks (mmap on x86_64 with reserved-but-unmapped pages, growing via `mprotect`).

### Loads, Stores, and `memarg`

Every load/store instruction encodes a `memarg` of the form `(offset N, align A)`:

- `offset` is added to the dynamic address.
- `align` is a hint (`2^A`).

The effective address is `addr + offset`. The bounds check is `addr + offset + size_of_type ≤ memory_size_in_bytes`. If not satisfied, the engine traps.

Because the bounds check is on the sum, a clever optimizer can often **fold** the offset into the base address at compile time, eliding the dynamic check via guard pages on 64-bit hosts (a 4 GiB region of virtual address space is reserved, with subsequent pages set to PROT_NONE; the hardware MMU then performs the bounds check for free).

### Endianness

All multi-byte loads/stores are **little-endian**. This is true on every host, including big-endian platforms (where the engine inserts byte-swap instructions).

### The Bulk-Memory Proposal (Stable)

Three high-throughput primitives:

```
memory.copy   src dst len     ;; byte-wise memcpy
memory.fill   dst val len     ;; byte-wise memset
memory.init $data dst src len ;; init from passive data segment
data.drop    $data            ;; mark data segment as no longer needed
```

`memory.copy` permits overlapping source and destination (with semantics like `memmove`). The engine emits SIMD-accelerated native code for these, often outperforming a hand-rolled loop by 5–20×.

### The Multi-Memory Proposal (Phase 4, Shipping)

Once enabled, modules may declare multiple memories:

```wat
(memory $heap 1 100)
(memory $stack 1 1)
(func
  i32.const 0
  i32.load (memory $heap))   ;; load from $heap
```

Use cases: separating code/data heaps, sandboxing untrusted memory regions within one module, providing aligned-vs-unaligned memory sections for SIMD.

### The Memory64 Proposal (Phase 3)

Replaces 32-bit indices and sizes with 64-bit equivalents. Required for >4 GiB heaps; particularly relevant for in-browser scientific computing (image processing, 3D modeling, ML inference).

---

## 8. Tables and call_indirect

### Function Tables

A **table** is an array of references — typically `funcref` for indirect calls, `externref` for host-managed handles. Modules declare tables and initialize them from element segments.

```wat
(table $vtable 16 funcref)
(elem (i32.const 0) $f0 $f1 $f2)
```

This places `$f0` at index 0, `$f1` at 1, `$f2` at 2.

### `call_indirect`

```wat
i32.const 1
call_indirect (type $sig)    ;; calls the funcref at table index 1, signature must match $sig
```

The runtime:

1. Pops the function index.
2. Bounds-checks against the table size (trap if OOB).
3. Loads the funcref at that index.
4. Type-checks the funcref's signature against the static type immediate `(type $sig)` (trap on mismatch).
5. Performs the call.

This is how C function pointers, C++ virtual dispatch, Rust trait objects, and dynamic-language method dispatch all lower to Wasm.

### Element Segments — Active vs Passive vs Declarative

| Kind         | When applied                                | Purpose                                |
|--------------|---------------------------------------------|----------------------------------------|
| Active       | At instantiation, into a specific table      | Static vtables                         |
| Passive      | On demand via `table.init`                   | Dynamic vtable construction            |
| Declarative  | Declares funcs as references; not copied     | Mark functions as used by ref.func     |

```wat
(elem $passive funcref (ref.func $a) (ref.func $b))    ;; passive
(elem declare funcref (ref.func $c))                    ;; declarative
(elem (i32.const 0) $a $b)                              ;; active
```

`table.init $tbl $elem` copies a slice from the passive element segment into the table at runtime. `elem.drop` marks the segment as no longer usable (engines may reclaim memory).

### `table.grow`, `table.size`, `table.fill`

Symmetric to `memory.*`:

```
table.grow $t  ;; pops init-value, pops delta, pushes prior size or -1
table.size $t
table.fill $t  ;; pops dst, val, len
```

### Reference Types Expanded Tables

Before reference-types, tables held only `funcref`. Now `externref` is allowed, enabling:

- Storing JS objects inside Wasm tables (with the JS API surface to retrieve them later).
- Building registry-style structures across the host boundary.

This is foundational for the Component Model's resource handles.

---

## 9. The Component Model — Wit and the Canonical ABI

### A Module of Modules

A **Component** is a packaging format that wraps Wasm modules with **interface types** — high-level types (strings, lists, records, variants, resources) that ordinary Wasm modules can't express directly. Components compose: a Component can import other Components and re-export them, forming a directed acyclic graph of typed interfaces.

The Component Model is **not** a separate VM — it sits atop core Wasm, using the **Canonical ABI** to lower interface-typed values into core Wasm operands (i32 indices, pointer-length pairs, etc.).

Status: Component Model is **Phase 4 (stable)** for its core types; some advanced features (async, streams) remain in earlier phases.

### WIT — The Interface Definition Language

WIT (Wasm Interface Types) is the IDL:

```wit
package example:image@0.1.0;

interface pixel {
  record point { x: u32, y: u32 }
  record color { r: u8, g: u8, b: u8, a: u8 }
}

interface buffer {
  use pixel.{point, color};

  resource image-buffer {
    constructor(width: u32, height: u32);
    get-pixel: func(p: point) -> color;
    set-pixel: func(p: point, c: color);
    dimensions: func() -> tuple<u32, u32>;
  }

  render: func(buf: borrow<image-buffer>) -> list<u8>;
}

world image-world {
  import logger: func(msg: string);
  export buffer;
}
```

Key syntactic elements:

- **`package`** — namespacing (`namespace:name@version`).
- **`interface`** — a named bundle of types and functions.
- **`record`** — a named struct type.
- **`variant`** — a tagged union (algebraic data type).
- **`enum`** — a flat enumeration (variant with no payloads).
- **`flags`** — a bitset of named bits.
- **`tuple<...>`** — anonymous positional tuple.
- **`list<T>`** — variable-length sequence (lowered as pointer + length).
- **`option<T>`**, **`result<T, E>`** — sum types.
- **`resource`** — opaque, host-managed entity with lifecycle (`constructor`, `dtor`, methods).
- **`borrow<R>`** / **`own<R>`** — linear-types-style ownership annotations on resources.
- **`world`** — the import + export contract of a Component.

### The Canonical ABI

The Canonical ABI specifies, byte-for-byte, how each interface type **lowers** to core Wasm:

| WIT type        | Core Wasm representation                                                  |
|-----------------|---------------------------------------------------------------------------|
| `bool`, `u8`–`u64`, `s8`–`s64`, `f32`, `f64` | One core operand of equivalent core type      |
| `char` (Unicode scalar value)                | i32                                            |
| `string`                                     | (i32 pointer, i32 byte-length) — UTF-8         |
| `list<T>`                                    | (i32 pointer, i32 element-count) in linear memory |
| `record { a: T, b: U }`                     | Flattened: two operands (or layout in memory)  |
| `tuple<T, U>`                               | Same as record                                 |
| `variant { v1(T), v2(U) }`                  | (i32 discriminant, T or U operand)             |
| `option<T>`                                 | Variant with `none` + `some(T)`                |
| `result<T, E>`                              | Variant with `ok(T)` + `err(E)`                |
| `resource R`                                | i32 handle (index into a per-instance table)   |

The Canonical ABI defines **lift** (from core values to WIT values) and **lower** (the inverse) operations. Each lift/lower walks the type and emits/consumes core Wasm primitives.

### `wit-bindgen` — Generated Host Bindings

`wit-bindgen` is the official tool for generating bindings from WIT files. Per-language generators emit:

- **Rust**: `wit-bindgen rust path/to/world.wit` produces a `bindgen!()` macro call you embed; the macro generates Rust traits/types matching the WIT.
- **Go**: bindings via `cm` (component-model) types — handles, lift/lower glue.
- **Python**: typed stubs and a wrapper class library.
- **JavaScript**: ES module exports with TypeScript declarations.
- **C**: header file with structs, function pointers, and ABI plumbing.

```bash
# Rust binding generation:
cargo install wit-bindgen-cli
wit-bindgen rust --out-dir src/bindings world.wit

# Or via the proc-macro path:
# (in Cargo.toml)
# [dependencies]
# wit-bindgen = "0.30"
```

Generated Rust:

```rust
wit_bindgen::generate!({
    path: "wit/world.wit",
    world: "image-world",
});

struct MyImage;

impl Guest for MyImage {
    fn render(buf: &Buffer::ImageBuffer) -> Vec<u8> {
        // implementation
        vec![]
    }
}

export!(MyImage);
```

### Component vs Module

| Aspect          | Module (.wasm)                             | Component (.wasm with component header) |
|-----------------|---------------------------------------------|------------------------------------------|
| Type system     | Core types only (i32/i64/f32/f64/v128/refs) | Full interface types                     |
| Linking         | Imports by (module, name) string            | Imports/exports by typed interface       |
| Resources       | Indirectly via tables                       | First-class with linear ownership        |
| Polyglot        | Hard (manual ABI)                           | Easy (canonical ABI)                     |
| Tooling status  | Stable since 2017                           | Stable core; ergonomics still improving  |

### Composing Components

`wasm-tools compose` and `wac compose` link multiple `.wasm` Components into a single artifact. The composer matches imports of one Component to exports of another, type-checked through WIT.

```bash
wasm-tools compose -o app.wasm parser.wasm renderer.wasm logger.wasm
```

The result is one Component you can run, with internal Components linked by interface types.

---

## 10. WASI — Preview1 vs Preview2

### Preview1 — The Historical Baseline

WASI Preview1 (also called `wasi_snapshot_preview1`) is the original WASI: a **single namespace** of imports providing POSIX-like syscalls.

```wat
(import "wasi_snapshot_preview1" "fd_read"
  (func $fd_read (param i32 i32 i32 i32) (result i32)))
(import "wasi_snapshot_preview1" "fd_write"
  (func $fd_write (param i32 i32 i32 i32) (result i32)))
(import "wasi_snapshot_preview1" "path_open" ...)
(import "wasi_snapshot_preview1" "clock_time_get" ...)
(import "wasi_snapshot_preview1" "random_get" ...)
(import "wasi_snapshot_preview1" "args_get" ...)
(import "wasi_snapshot_preview1" "environ_get" ...)
(import "wasi_snapshot_preview1" "proc_exit" ...)
```

Function signatures are flat: pointers (i32), lengths (i32), result is i32 errno-like.

| Function                | Purpose                              |
|-------------------------|--------------------------------------|
| `args_get`, `args_sizes_get`         | argv-like access            |
| `environ_get`, `environ_sizes_get`   | envp-like access            |
| `clock_res_get`, `clock_time_get`    | clock + monotonic time      |
| `fd_read`, `fd_write`, `fd_close`    | basic file descriptor I/O   |
| `path_open`                          | open a path → fd            |
| `path_create_directory`, `path_unlink_file` | filesystem mutation  |
| `random_get`                         | secure random bytes         |
| `proc_exit`                          | exit with status            |
| `sock_*` (limited)                   | sockets (incomplete)        |

Preview1 is a **module-level** API: compiled into core Wasm modules. It is supported by every major standalone runtime (wasmtime, wasmer, wasmedge, wasm3, wamr).

### Preview2 — Component-Model-Native

WASI Preview2 reimagines WASI on top of the Component Model. Rather than a single namespace, it is a constellation of **typed interfaces**:

| Interface           | Purpose                                                  |
|---------------------|----------------------------------------------------------|
| `wasi:cli`          | Command-line entry, stdin/stdout/stderr, env, args       |
| `wasi:filesystem`   | Files and directories with capability handles            |
| `wasi:io`           | Streams, pollables, errors                               |
| `wasi:sockets`      | TCP/UDP, IPv4/IPv6                                       |
| `wasi:clocks`       | Monotonic + wall clock                                   |
| `wasi:random`       | Secure RNG                                               |
| `wasi:http`         | HTTP/1.1 + HTTP/2 client and server (proxy-server target) |

Each is a WIT package with proper resources and types. Example:

```wit
// wasi:filesystem/types
resource descriptor {
  read-via-stream: func(offset: u64) -> result<input-stream, error-code>;
  write-via-stream: func(offset: u64) -> result<output-stream, error-code>;
  stat: func() -> result<descriptor-stat, error-code>;
  ...
}
```

A Preview2 program declares which interfaces it needs in its `world`, and the runtime supplies them.

### Migration: Modules → Components

Most existing toolchains target Preview1. To run a Preview1 module against a Preview2 runtime:

```bash
wasm-tools component new app.preview1.wasm \
  --adapt wasi_snapshot_preview1=wasi_snapshot_preview1.command.wasm \
  -o app.component.wasm
```

The `wasi_snapshot_preview1.command.wasm` adapter (shipped with wasmtime / wasi-sdk) shims Preview1 imports onto Preview2 interfaces. The result is a Preview2 Component that can be loaded directly.

### Runtime Support Matrix

| Runtime    | Language | Preview1 | Preview2 | Component Model | Notes                       |
|------------|----------|----------|----------|-----------------|-----------------------------|
| wasmtime   | Rust     | Yes      | Yes      | Yes             | Reference impl              |
| wasmer     | Rust     | Yes      | Partial  | Partial         | Strong commercial focus     |
| wasmedge   | C++      | Yes      | Partial  | Partial         | CNCF; AI/ML focus           |
| wasm3      | C        | Yes      | No       | No              | Tiny interpreter            |
| wamr       | C        | Yes      | Partial  | Partial         | Bytecode Alliance "micro"   |
| Node.js    | (V8)     | Limited  | No       | No              | `--experimental-wasi`       |
| Deno       | (V8)     | Yes      | No       | No              | Built-in                    |

---

## 11. Producers — Rust to Wasm

### Targets

Rust has multiple Wasm targets:

| Target                       | Purpose                                              |
|------------------------------|------------------------------------------------------|
| `wasm32-unknown-unknown`     | "Bare" Wasm — no system imports; for browsers + raw  |
| `wasm32-wasi` (deprecated)   | Old name for Preview1 target                         |
| `wasm32-wasip1`              | WASI Preview1 (modern name; same ABI)                |
| `wasm32-wasip2`              | WASI Preview2 (Component Model)                      |
| `wasm32-unknown-emscripten`  | Emscripten-compatible — for porting C-ish code       |

```bash
rustup target add wasm32-unknown-unknown
rustup target add wasm32-wasip1
rustup target add wasm32-wasip2
```

### Exporting Functions

For raw Wasm, mark functions for export:

```rust
#[no_mangle]
pub extern "C" fn add(a: i32, b: i32) -> i32 {
    a + b
}

#[no_mangle]
pub extern "C" fn fib(n: u32) -> u64 {
    let (mut a, mut b) = (0u64, 1u64);
    for _ in 0..n {
        let t = a + b;
        a = b;
        b = t;
    }
    a
}
```

Build:

```bash
cargo build --target wasm32-unknown-unknown --release
ls target/wasm32-unknown-unknown/release/myproject.wasm
```

### `wasm-pack` — Browser Bundles

For browser apps, `wasm-pack build` wraps `cargo build` + `wasm-bindgen` + bundler shims:

```bash
cargo install wasm-pack
wasm-pack build --target web --release
```

Output: `pkg/myproject.js` (loader), `pkg/myproject_bg.wasm` (binary), `pkg/myproject.d.ts` (TypeScript types).

### `wasm-bindgen` — Typed JS Interop

`wasm-bindgen` is the proc-macro-driven bridge that maps Rust types to JS:

```rust
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct Counter {
    n: u32,
}

#[wasm_bindgen]
impl Counter {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self { Self { n: 0 } }
    pub fn inc(&mut self) { self.n += 1; }
    pub fn value(&self) -> u32 { self.n }
}

#[wasm_bindgen]
pub fn greet(name: &str) -> String {
    format!("Hello, {name}!")
}
```

The generated JS exposes `Counter` as a class; strings and `Vec<u8>` are marshaled via `TextEncoder`/`TextDecoder`; lifetimes are managed by JS finalizers.

### `web-sys` and `js-sys`

For typed access to DOM and JS built-ins:

```rust
use wasm_bindgen::prelude::*;
use web_sys::{window, Document, HtmlElement};

#[wasm_bindgen(start)]
pub fn main() -> Result<(), JsValue> {
    let win = window().expect("no window");
    let doc = win.document().expect("no document");
    let body = doc.body().expect("no body");

    let p = doc.create_element("p")?
        .dyn_into::<HtmlElement>()?;
    p.set_inner_text("Hello from Rust+Wasm");
    body.append_child(&p)?;
    Ok(())
}
```

Every Web IDL interface has a typed Rust wrapper.

### `cargo-component` — Component Model Builds

For Component Model targets:

```bash
cargo install cargo-component
cargo component new --reactor my-component
cd my-component
# Edit wit/world.wit
cargo component build --release
```

Output is a true Component (`*.wasm` with the Component header), runnable on any Preview2 runtime.

### Size Optimization in Cargo

```toml
# Cargo.toml
[profile.release]
opt-level = "z"        # Optimize for size (vs "3" for speed)
lto = true             # Link-time optimization
codegen-units = 1      # Single codegen unit for better LTO
strip = true           # Strip symbols
panic = "abort"        # No unwind tables
```

Combined with `wasm-opt -Oz` and `wasm-strip`, this can take a non-trivial Rust binary from 1+ MB down to 30–80 KB.

---

## 12. Producers — C/C++ via Emscripten

### Emscripten as a Wrapper

`emcc` is Emscripten's drop-in compiler driver. Internally it invokes `clang` (with the upstream LLVM Wasm32 backend), runs `wasm-ld`, and produces:

- A `.wasm` file (the Wasm binary).
- A `.js` file (the loader / glue, JavaScript code that fetches, instantiates, and exposes the Wasm).
- Optionally `.data` files (preloaded virtual filesystem).
- Optionally `.html` (a default page wrapper).

```bash
emcc src/main.c -o app.html \
    -O3 \
    -s WASM=1 \
    -s ALLOW_MEMORY_GROWTH=1 \
    -s INITIAL_MEMORY=16MB \
    -s ENVIRONMENT=web \
    -s MODULARIZE=1 \
    -s EXPORT_ES6=1 \
    -s EXPORTED_FUNCTIONS='["_main","_compute"]' \
    -s EXPORTED_RUNTIME_METHODS='["ccall","cwrap"]'
```

### Common Flags

| Flag                                  | Purpose                                          |
|---------------------------------------|--------------------------------------------------|
| `-s WASM=1`                           | Emit Wasm (default; vs JS-only fallback)         |
| `-s STANDALONE_WASM`                  | No JS glue; for non-browser runtimes             |
| `-s ALLOW_MEMORY_GROWTH`              | Allow `memory.grow` at runtime                   |
| `-s INITIAL_MEMORY=N`                 | Initial linear memory in bytes                   |
| `-s MAXIMUM_MEMORY=N`                 | Cap                                              |
| `-s EXPORTED_FUNCTIONS='["_foo"]'`    | Functions to expose to JS (note leading `_`)     |
| `-s EXPORTED_RUNTIME_METHODS=...`     | JS helpers to expose (`ccall`, `cwrap`, `HEAP*`) |
| `-s ENVIRONMENT=web,worker,node`      | Target environments                              |
| `-s MODULARIZE=1`                     | Wrap in a factory function (for ES modules)      |
| `-s EXPORT_ES6=1`                     | Emit `export default` rather than UMD            |
| `--preload-file ./assets`             | Preload assets into virtual FS                   |
| `-s USE_PTHREADS=1`                   | Enable pthreads (needs SharedArrayBuffer)        |
| `-s PTHREAD_POOL_SIZE=N`              | Pre-spawn worker pool                            |

### Calling C from JS

```c
// math.c
#include <emscripten.h>

EMSCRIPTEN_KEEPALIVE
int add(int a, int b) { return a + b; }

EMSCRIPTEN_KEEPALIVE
double square(double x) { return x * x; }
```

```bash
emcc math.c -o math.js -s EXPORTED_FUNCTIONS='["_add","_square"]' \
    -s EXPORTED_RUNTIME_METHODS='["ccall","cwrap"]'
```

```javascript
import Module from './math.js';
const m = await Module();
const add = m.cwrap('add', 'number', ['number','number']);
const square = m.cwrap('square', 'number', ['number']);
console.log(add(2,3), square(2.5));
```

### Calling JS from C

```c
#include <emscripten.h>

EM_JS(int, prompt_for_int, (), {
  return parseInt(prompt("Enter a number"), 10);
});

int main(void) {
    int n = prompt_for_int();
    EM_ASM({ console.log("got " + $0); }, n);
    return 0;
}
```

`EM_JS` defines a C-callable function whose body is JS. `EM_ASM` is inline JS.

### `emrun` — Local Testing Server

```bash
emrun --no_browser --port 8080 app.html
```

Spins up a local server with cross-origin isolation headers (required for SharedArrayBuffer / pthreads).

### `emconfigure` and `emmake`

For autotools/CMake projects:

```bash
emconfigure ./configure
emmake make
```

These wrap `configure` / `make` to use `emcc` instead of `cc`.

### CMake Integration

```cmake
if(EMSCRIPTEN)
  set(CMAKE_EXECUTABLE_SUFFIX ".html")
  set(LINK_FLAGS "-s WASM=1 -s ALLOW_MEMORY_GROWTH=1")
endif()
```

```bash
emcmake cmake -B build
cmake --build build
```

---

## 13. Producers — Go to Wasm

### `GOOS=js GOARCH=wasm`

The original Go-to-Wasm target, intended for browser deployment:

```bash
GOOS=js GOARCH=wasm go build -o main.wasm
```

Requires the `wasm_exec.js` glue from `$(go env GOROOT)/misc/wasm/wasm_exec.js`:

```html
<script src="wasm_exec.js"></script>
<script>
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
    .then(result => go.run(result.instance));
</script>
```

The `syscall/js` package provides typed DOM-level interop:

```go
package main

import "syscall/js"

func add(this js.Value, args []js.Value) any {
    return args[0].Int() + args[1].Int()
}

func main() {
    js.Global().Set("add", js.FuncOf(add))
    select {} // block forever
}
```

`js.Global()` returns the JS global; `js.FuncOf` wraps a Go function as a callable JS function.

### `GOOS=wasip1 GOARCH=wasm` (Go 1.21+)

For standalone WASI runtimes:

```bash
GOOS=wasip1 GOARCH=wasm go build -o app.wasm
wasmtime app.wasm
```

This uses the `wasi_snapshot_preview1` ABI directly — no JS glue required. Output runs anywhere WASI Preview1 runs.

### `tinygo` — Smaller Binaries

The Go compiler emits a substantial runtime overhead (~2 MB minimum). TinyGo, an alternative compiler using LLVM, emits much smaller binaries:

```bash
tinygo build -target wasm -o tiny.wasm main.go        # browser
tinygo build -target wasi -o tiny.wasm main.go        # WASI
tinygo build -target wasm-unknown -o tiny.wasm main.go # bare
```

A Hello-World in tinygo can be under 10 KB; the equivalent in standard Go is ~2 MB.

Trade-offs: TinyGo supports a subset of Go (no full reflection, limited goroutines, restricted standard library). Excellent for embedded and edge-Wasm use cases.

### The "Go Runtime Is Heavy" Reality

Go's runtime — goroutine scheduler, garbage collector, escape analysis runtime support — is included in every binary. For compute-heavy interactive code (browsers), the goroutine scheduler runs as a JS-level event loop on `wasm_exec.js`, which adds overhead.

For backends that already run Go, recompiling for Wasm is straightforward; for new in-browser apps, Rust+wasm-bindgen typically yields a smaller, faster bundle.

---

## 14. Browser Host

### `WebAssembly.instantiate` and `instantiateStreaming`

The two entry points:

```javascript
// Approach 1: bytes you already have
const buf = await fetch("app.wasm").then(r => r.arrayBuffer());
const { instance } = await WebAssembly.instantiate(buf, importObject);
console.log(instance.exports.add(2, 3));

// Approach 2: streaming (preferred — overlaps fetch + compile)
const { instance } = await WebAssembly.instantiateStreaming(
  fetch("app.wasm"),
  importObject
);
```

`instantiateStreaming` requires the response's `Content-Type` to be `application/wasm`; it compiles bytes as they arrive over the wire, often reducing time-to-first-execute by 30–50%.

### `importObject` Shape

A two-level JavaScript object mapping module names → field names → values:

```javascript
const importObject = {
  env: {
    log: (ptr, len) => {
      const bytes = new Uint8Array(memory.buffer, ptr, len);
      console.log(new TextDecoder().decode(bytes));
    },
    memory: new WebAssembly.Memory({ initial: 16, maximum: 256 }),
    table: new WebAssembly.Table({ initial: 8, element: 'anyfunc' }),
  },
  wasi_snapshot_preview1: { /* WASI imports if needed */ },
};
```

### Core JS API Objects

| Object                  | Role                                             |
|-------------------------|--------------------------------------------------|
| `WebAssembly.Module`    | Compiled but uninstantiated module               |
| `WebAssembly.Instance`  | Instantiated module with exports                 |
| `WebAssembly.Memory`    | Linear memory; `.buffer` is an `ArrayBuffer`     |
| `WebAssembly.Table`     | Function table; `.get(i)`, `.set(i, fn)`         |
| `WebAssembly.Global`    | Mutable or immutable global; `.value`            |
| `WebAssembly.compile`   | Async compile → `Module`                         |
| `WebAssembly.compileStreaming` | Streaming compile → `Module`              |
| `WebAssembly.validate`  | Synchronous bool: is this binary valid?          |

### Growing Memory

```javascript
const mem = new WebAssembly.Memory({ initial: 1, maximum: 100 });
mem.grow(10); // add 10 pages → 11 total
const view = new Uint8Array(mem.buffer); // re-create after grow!
```

**After every `mem.grow()` or any in-Wasm `memory.grow`, JS-side `ArrayBuffer` views become detached.** Re-acquire `mem.buffer` and re-create your typed array views. This is the #1 source of subtle bugs in Wasm-JS code.

### Threads — SharedArrayBuffer + Atomics

For pthreads / Rust Rayon parallelism:

```javascript
const mem = new WebAssembly.Memory({
  initial: 16,
  maximum: 256,
  shared: true   // SAB-backed
});
```

The host page must serve cross-origin-isolated headers:

```
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Embedder-Policy: require-corp
```

Without these, `SharedArrayBuffer` is unavailable (post-Spectre browser policy).

`wasm-bindgen-rayon` then exposes Rust's Rayon library inside the browser, with worker threads receiving the same shared memory.

### Fetch + Streaming Compilation

```javascript
const resp = await fetch("app.wasm");
const module = await WebAssembly.compileStreaming(resp);
// `module` can be cached, instantiated multiple times, transferred to workers
```

Streaming compile pipelines parsing + validation + JIT generation as bytes arrive, reducing perceived startup latency. For a 5 MB module on a typical desktop, time-to-first-execute drops from ~250 ms (full download then compile) to ~80 ms (overlapped).

---

## 15. JS↔Wasm Memory Marshaling

### Only `i32` Crosses the Boundary (Easily)

The Wasm-JS function call ABI passes only numeric types. Anything else (strings, structs, arrays) requires explicit marshaling: write into linear memory, pass a `(pointer, length)` pair, on the other side decode.

### String Marshaling

```javascript
function passString(instance, s) {
  const bytes = new TextEncoder().encode(s);
  const len = bytes.length;
  // ask wasm to allocate
  const ptr = instance.exports.alloc(len);
  // copy bytes into linear memory
  new Uint8Array(instance.exports.memory.buffer, ptr, len).set(bytes);
  return [ptr, len];
}

function readString(instance, ptr, len) {
  const bytes = new Uint8Array(instance.exports.memory.buffer, ptr, len);
  return new TextDecoder("utf-8").decode(bytes);
}
```

The Wasm side typically exports its own `alloc` / `dealloc` matching its allocator; calling `malloc` directly is not safe across language boundaries because the layout / metadata can differ.

### `wasm-bindgen` Abstracts All This

```rust
#[wasm_bindgen]
pub fn greet(name: &str) -> String {
    format!("Hello, {name}!")
}
```

Generated JS:

```javascript
export function greet(name) {
  let ptr0, len0;
  try {
    const ret = wasm.greet(passStringToWasm0(name, /* malloc */, /* realloc */));
    return getStringFromWasm0(ret[0], ret[1]);
  } finally {
    wasm.__wbindgen_free(ptr0, len0, 1);
  }
}
```

The macro+codegen handles allocation, encoding, decoding, and lifetime cleanup.

### Bulk-Memory for Fast Copies

The `memory.copy` instruction lets a Wasm allocator copy in linear memory via the engine's optimized routine, faster than a JS-side `Uint8Array.set` followed by a Wasm-side `for` loop.

For very large transfers (hundreds of KB to MB), use the bulk-memory primitives directly (or rely on wasm-bindgen's modern code generator that prefers them).

### Lifetime: Memory Is Owned by Wasm

A typed-array view created from `WebAssembly.Memory.buffer` is **only valid until** memory grows. After `memory.grow`:

- The original `ArrayBuffer` is **detached**.
- All views referring to it have `byteLength === 0`.
- Re-create views from the new `memory.buffer`.

Robust code therefore:

```javascript
function getU8() { return new Uint8Array(instance.exports.memory.buffer); }
// re-call getU8() after every potential grow
```

Alternatively, install a `Proxy` that re-acquires the view on each access.

---

## 16. Performance Considerations

### Boundary-Crossing Cost

Every JS→Wasm or Wasm→JS call has overhead — typically 50–500 ns in V8. For small operations (e.g., `add(2, 3)`), this can dominate the call. The remedy is **batching**: call into Wasm with a chunk of work, not per-element.

```javascript
// Bad: 1 million boundary crossings
for (let i = 0; i < 1_000_000; i++) total += instance.exports.process(i);

// Good: 1 boundary crossing
const total = instance.exports.processBatch(0, 1_000_000);
```

### SIMD via `v128`

Hand-tuned C/Rust code can use the Wasm SIMD intrinsics:

```rust
use std::arch::wasm32::*;

#[target_feature(enable = "simd128")]
pub unsafe fn dot4(a: &[f32; 4], b: &[f32; 4]) -> f32 {
    let av = v128_load(a.as_ptr() as *const v128);
    let bv = v128_load(b.as_ptr() as *const v128);
    let prod = f32x4_mul(av, bv);
    f32x4_extract_lane::<0>(prod) +
    f32x4_extract_lane::<1>(prod) +
    f32x4_extract_lane::<2>(prod) +
    f32x4_extract_lane::<3>(prod)
}
```

Compile with `RUSTFLAGS='-C target-feature=+simd128'`. SIMD speedups of 2–4× are typical for vectorizable inner loops.

### Browser JIT Pipeline (V8)

V8's WebAssembly compilation has two tiers:

| Tier      | Compiler   | Compile time | Code quality |
|-----------|------------|--------------|--------------|
| Tier 1    | Liftoff    | ~1 ms / fn   | ~60% of native |
| Tier 2    | TurboFan   | ~10 ms / fn  | ~95% of native |

A function is compiled by Liftoff first (fast startup) and re-compiled by TurboFan in the background; once TurboFan finishes, subsequent calls dispatch to the optimized version.

There is also a **tier-down** path: if a TurboFan-compiled function deoptimizes (e.g., due to memory bounds-check failures triggering side exits), it can revert to Liftoff temporarily.

### Warmup Latency

A cold module:

1. Fetch — network-bound.
2. Compile (Liftoff) — proportional to code size (~5 MB/s).
3. Instantiate — copy data segments.
4. Begin executing.

For interactive web apps, total time-to-first-execute is in the 50–500 ms range for typical bundles. Streaming compilation overlaps (1) and (2).

### Wasm vs JS Speed

Wasm beats JS for:

- Cryptography (constant-time math, bignum ops).
- Compression / decompression.
- Image and video codecs.
- Numerical kernels (linear algebra, ML inference).
- Game-loop physics and graphics math.

Wasm is **not** automatically faster than JS for:

- DOM manipulation (Wasm has no DOM access; every operation crosses to JS).
- String-heavy logic where Wasm has to marshal each string.
- Sparse, allocation-heavy code (Wasm has no GC for the producer language unless that runtime is included).

Realistic speedups for compute-bound workloads: 2–10×. Outliers (heavy SIMD, cryptography) can hit 20–50×. Code that is "JS-ish" (event handlers, glue logic) often performs similarly or worse in Wasm.

---

## 17. Debugging

### DWARF Debug Info

The standard debug-info format is **DWARF**, embedded in custom sections:

```bash
# Rust
cargo build --target wasm32-unknown-unknown
# Default release strips debug info; for debugging, add:
# [profile.release]
# debug = true

# C/C++ via emcc
emcc src/main.c -g4 -o app.html

# Plain clang
clang --target=wasm32 -g -c main.c -o main.o
```

`-g` includes file-and-line info; `-g4` (Emscripten) generates source maps and full DWARF.

### Browser DevTools

Chrome and Firefox DevTools support stepping through Wasm at the source level (when DWARF info is present), inspecting locals, and setting breakpoints. Chrome's "Wasm Debugging" extension is required for full source-level debug.

### `wasm2wat` for Reverse Engineering

```bash
wasm2wat app.wasm -o app.wat
```

Produces a human-readable textual form. Combined with `--enable-all` to allow modern proposals, this is the simplest way to inspect binary output.

### `wasm-objdump`

```bash
wasm-objdump --headers app.wasm     # section sizes
wasm-objdump --details app.wasm     # full section detail
wasm-objdump --disassemble app.wasm # function-level disassembly
```

Output:

```
Disassembly of section: code
0006a0 func[0] <add>:
 6a1: 20 00       | local.get 0
 6a3: 20 01       | local.get 1
 6a5: 6a          | i32.add
 6a6: 0b          | end
```

### `wabt` Tools Summary

| Tool              | Purpose                                        |
|-------------------|------------------------------------------------|
| `wat2wasm`        | Compile text to binary                         |
| `wasm2wat`        | Decompile binary to text                       |
| `wasm-validate`   | Run validation (return code 0 = valid)         |
| `wasm-objdump`    | Inspect binary structure                       |
| `wasm-strip`      | Remove custom (debug/name) sections            |
| `wasm-decompile`  | Best-effort decompile to C-like pseudocode     |
| `wasm-interp`     | Reference interpreter (for spec testing)       |

---

## 18. Size Optimization

### `wasm-opt -Oz`

Binaryen's `wasm-opt` runs a full optimizing pipeline on a `.wasm` binary, often shaving 20–50% off Cargo's `--release` output:

```bash
wasm-opt -Oz -o app.opt.wasm app.wasm
```

`-Oz` means "optimize aggressively for size"; `-O3` is "for speed". Common pipeline includes:

- Dead-code elimination across functions.
- Inlining + outlining decisions.
- Local variable coalescing.
- Constant folding.
- Vacuum (removing no-ops).
- DCE of unused element/data segments.

### Closure Compiler for JS Glue (Emscripten)

```bash
emcc src/main.c -Oz --closure 1 -o app.html
```

`--closure 1` runs Google Closure Compiler on Emscripten's JS glue, which is otherwise verbose (~20 KB minified by default).

### Cargo Profile

```toml
[profile.release]
opt-level = "z"
lto = "fat"
codegen-units = 1
strip = true
panic = "abort"
```

### `wasm-snip` — Targeted Dead-Code

`wasm-snip` lets you replace specific functions with `unreachable`, useful for stripping panic / formatting code:

```bash
wasm-snip --snip-rust-fmt-code --snip-rust-panicking-code -o snipped.wasm input.wasm
```

The leftover machinery (panic infrastructure that's almost-but-not-quite reachable) often vanishes once these stubs land, via subsequent `wasm-opt -Oz`.

### Realistic Sizes

| Source         | Output size (after `wasm-opt -Oz`) |
|----------------|-------------------------------------|
| C "hello world" via wasi-sdk  | ~3 KB                |
| Rust `wasm32-unknown-unknown` (no_std hello) | ~250 bytes |
| Rust with `std` + minimal logic | 100–200 KB         |
| Rust with `wasm-bindgen` minimal | 30–80 KB           |
| AssemblyScript (TS subset) | 1–10 KB                  |
| TinyGo "hello world"       | ~10 KB                   |
| Go (standard) "hello"      | ~2 MB                    |

### AssemblyScript

A TypeScript-flavored language compiling directly to Wasm with no runtime:

```typescript
export function fib(n: u32): u64 {
  let a: u64 = 0, b: u64 = 1;
  for (let i: u32 = 0; i < n; i++) {
    const t = a + b;
    a = b;
    b = t;
  }
  return a;
}
```

```bash
asc index.ts --outFile build/index.wasm --optimize
```

Output of this `fib` is under 200 bytes. AssemblyScript has its own GC for arrays/strings; for simple arithmetic kernels, it produces extremely small Wasm.

---

## 19. Sandboxing and Security

### Capability-Based — No Ambient DOM

Inside Wasm there is **no built-in DOM** access, no `fetch`, no `window`, no `localStorage`. Anything the host wants to expose must be explicitly imported. This is the **capability-based** security model: a Wasm module can only do what it has been given permission to do, where "permission" means "an import in its `importObject`".

Compare to native code in a process: full POSIX access to filesystem, network, mmap, fork. Wasm has none of that by default.

### No Syscalls

Native code makes syscalls (`int 0x80`, `syscall`, ARM `svc`). Wasm has **no syscall instruction**. All host interaction goes through imports, which are typed and validated at instantiation.

### Bounds-Checked Memory

Every load and store is checked. The model is conceptually:

```c
if (addr + size > memory_size_in_bytes) trap();
```

In practice, 64-bit hosts use **virtual memory tricks**: the engine reserves 8 GiB of virtual address space (4 GiB heap + 4 GiB redzone) and sets PROT_NONE on the redzone; bounds checks become "is the address < 4 GiB?", which the optimizer can frequently elide entirely. Out-of-bounds accesses then trigger a SIGSEGV which the engine catches and converts to a Wasm trap.

### No JIT-Spray

JIT-spray attacks (filling memory with sequences that, when reinterpreted as native code, become useful gadgets) work against high-level JITs that emit unconstrained native code. Wasm validation prevents this:

- Code is in a separate code section, not addressable from data.
- Function references are opaque; you cannot construct a fake funcref pointing into your data.
- Tables hold typed references, not raw addresses.

### Spectre Mitigations

Spectre v1 (bounds-check bypass) is partially mitigated in V8 by:

- Inserting LFENCE-equivalent instructions on speculative bounds-check paths.
- Limiting cross-origin SAB sharing (cross-origin isolation requirement).
- Process isolation (each origin in a separate renderer process).

These mitigations have a small but measurable performance cost (~3–5%).

### Cross-Origin Isolation for SharedArrayBuffer

Post-Spectre, `SharedArrayBuffer` is gated behind:

```
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Embedder-Policy: require-corp
```

Without these, SharedArrayBuffer is unavailable; with them, the host page becomes **cross-origin isolated**, and SAB / `Atomics.wait` / Wasm threads work.

### Audit Surface

A Wasm sandbox's audit surface is:

1. The validator's correctness (well-studied; mechanized proofs exist).
2. The engine's compiler correctness (Cranelift, V8 TurboFan, Wasmer's LLVM backend).
3. The host's import implementations (e.g., the WASI runtime's `fd_read`).

Of these, (3) is the most fertile ground for vulnerabilities — bugs in WASI runtimes have shipped — but the **scope** of a vuln is constrained to what the embedder has imported.

---

## 20. Future Proposals

### Stable / Phase 4

| Proposal                     | Status                                     |
|------------------------------|--------------------------------------------|
| Multi-value returns          | Stable                                     |
| Reference types              | Stable                                     |
| Bulk memory                  | Stable                                     |
| Non-trapping fp-to-int       | Stable                                     |
| Sign-extension operators     | Stable                                     |
| Mutable globals              | Stable                                     |
| SIMD (simd128)               | Stable (Wasm 2.0)                          |
| Tail calls                   | Phase 4 (shipping)                         |
| Extended constants           | Phase 4                                    |
| Component Model (core)       | Phase 4 (stable for `0.2.0` worlds)        |

### In-Progress

| Proposal                  | Status     | Description                                       |
|---------------------------|------------|---------------------------------------------------|
| Threads                   | Phase 4    | SharedArrayBuffer + Atomics                       |
| Memory64                  | Phase 3    | 64-bit memory addressing                          |
| Multi-memory              | Phase 4    | Multiple linear memories per module               |
| Exception handling (new)  | Phase 3    | `try`/`catch`/`throw` with tags                   |
| GC (Garbage Collection)   | Phase 4    | Host-managed types: structs, arrays, anyref       |
| Stringref                 | Phase 1    | First-class Wasm string type                      |
| Relaxed SIMD              | Phase 4    | Lane-typed SIMD with relaxed semantics            |
| Stack switching           | Phase 1    | Lightweight coroutines / fibers                   |
| Function references       | Phase 4    | Typed funcref (vs opaque)                         |

### Component Model + WASI 0.3

The next big push is **WASI 0.3 / Preview3** — fully on the Component Model, with async APIs (`async-func`, streams, futures) for non-blocking I/O. Status: drafting, not yet shipped.

### GC Proposal in Detail

The GC proposal adds **host-managed types** — structs and arrays whose memory is allocated by the host's GC, not by the Wasm linear memory:

```wat
(type $point (struct (field $x f64) (field $y f64)))
(func (result (ref $point))
  (struct.new $point (f64.const 1.0) (f64.const 2.0)))
```

This enables high-level GC'd languages (Java, C#, Kotlin, Dart, OCaml, Scheme) to compile to Wasm without bundling their own GC, which dramatically reduces binary size.

Status: Phase 4 in late 2024; V8 ships an implementation; SpiderMonkey and JavaScriptCore are following.

---

## 21. Tooling Internals

### `wabt` — The Reference Implementation

- Written in C (and C++).
- The tools (`wat2wasm`, `wasm2wat`, `wasm-validate`, `wasm-interp`, etc.) implement the spec literally.
- Useful as a test oracle: if your engine disagrees with `wasm-interp` on the result of a binary, you have a bug.
- Open source at `github.com/WebAssembly/wabt`.

### `binaryen` — The Optimizer

- Written in C++.
- Has its own internal IR (called "Binaryen IR") that is a tree form of Wasm — easier to optimize than the linearized binary.
- Tools include `wasm-opt`, `wasm-as`, `wasm-dis`, `wasm-merge`, `wasm-metadce`.
- Used by Emscripten, AssemblyScript, J2CL (Java→Wasm), and others.
- Open source at `github.com/WebAssembly/binaryen`.

### `wasm-tools` — Bytecode Alliance Suite

- Written in Rust.
- A unified CLI for parsing, printing, dumping, validating, manipulating Wasm and Component Model artifacts.
- Subcommands: `parse`, `print`, `dump`, `strip`, `validate`, `compose`, `component new`, `metadata add`, `mutate`, `smith` (random module generator for fuzzing).
- Open source at `github.com/bytecodealliance/wasm-tools`.

### `wasi-sdk`

- A pre-built Clang + libc + WASI sysroot for compiling C/C++ to WASI.
- Bundles `wasi-libc`, `wasi-sdk-preview1` adapter, and ready-to-use `clang` invocations.
- Released by the Bytecode Alliance.

### `wit-bindgen`

- The reference WIT-to-host-language binding generator.
- Targets Rust (with `wit-bindgen` crate), C, Go, Python, JS, MoonBit, TeaVM, and more.
- Drives the Component Model adoption story across languages.

### Interpreters / Runtimes

| Runtime    | Language | LOC (approx) | Niche                                |
|------------|----------|--------------|--------------------------------------|
| wasmtime   | Rust     | ~200K        | Reference standalone runtime         |
| wasmer     | Rust     | ~150K        | Commercial focus, multi-backend      |
| wasmedge   | C++      | ~100K        | CNCF; AI/ML; container Wasm          |
| wasm3      | C        | ~10K         | Tiny interpreter; embedded           |
| wamr       | C        | ~50K         | Bytecode Alliance "micro"            |
| wazero     | Go       | ~80K         | Pure-Go, zero-dep, embeddable        |
| wagon      | Go       | ~20K         | Go interpreter (less active)         |

---

## 22. Idioms at the Internals Depth

### Canonical Exported Function Pattern

```rust
#[no_mangle]
pub extern "C" fn process(input_ptr: *const u8, input_len: usize) -> u64 {
    let input = unsafe { std::slice::from_raw_parts(input_ptr, input_len) };
    let result = do_work(input);
    let result_ptr = result.as_ptr() as u32;
    let result_len = result.len() as u32;
    std::mem::forget(result);   // leak; caller must dealloc
    ((result_ptr as u64) << 32) | (result_len as u64)
}
```

Returning two values packed into i64 because the MVP didn't have multi-value returns. Modern code with multi-value-stable target uses tuple returns.

### Manual Allocate / Free for Variable-Sized Data

```rust
#[no_mangle]
pub extern "C" fn alloc(len: usize) -> *mut u8 {
    let mut buf = Vec::<u8>::with_capacity(len);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

#[no_mangle]
pub unsafe extern "C" fn dealloc(ptr: *mut u8, len: usize) {
    let _ = Vec::from_raw_parts(ptr, 0, len);
}
```

JS asks Wasm for a buffer (`alloc`), writes data into linear memory, calls a processing function, then frees (`dealloc`). The contract is hand-coded in this MVP-style.

### String ABI: Pointer-Length Pair

UTF-8 byte sequence + length, sometimes packed into a single i64, sometimes returned via two values, sometimes via an out-pointer. WIT's canonical ABI standardizes this.

### Opaque Handles for Resources

Components return resource handles as i32 indices into a per-instance resource table. A handle is a capability: holding it lets you call methods; the instance can revoke handles by clearing table entries.

### Error-via-Out-Pointer (Preview1 Style)

```c
__wasi_errno_t
fd_read(__wasi_fd_t fd,
        const __wasi_iovec_t *iovs,
        size_t iovs_len,
        __wasi_size_t *retptr0);
```

The function returns an `errno`; the actual result is written through an out-pointer. This avoids needing multi-value returns or exceptions, at the cost of some ergonomics.

### "WASI Shim" Pattern for Portable Libraries

Libraries write code targeting a small WASI-like API (filesystem, time, random) and ship adapters for:

- True WASI Preview1 (for standalone runtimes).
- Browser shims (filesystem-via-IndexedDB, time-via-Date.now).
- Custom hosts (e.g., "filesystem" mapped to remote object store).

Rust crates like `cap-std` and `tokio` (with custom runtimes) use this pattern.

### Reactor Components

A **reactor component** is a Component without a `main` function; it exports zero or more interfaces and waits to be called. In WIT:

```wit
world reactor {
  export buffer;   // exposes the buffer interface
}
```

Reactors are the standard model for serverless / plugin / library deployments. The host instantiates the Component, retrieves exports, and calls them on demand.

### Command Components

A **command component** has a `wasi:cli/run.run` export; the host calls `run()` which acts as `main()`. This is the standard model for batch jobs / CLI tools.

```wit
world command {
  include wasi:cli/imports;
  export wasi:cli/run;
}
```

`wasmtime run app.wasm arg1 arg2` invokes the command's `run` function.

---

## Prerequisites

- **C** — understanding pointers, memory layout, and the basics of how compilers emit machine code helps to grasp Wasm's stack machine and linear memory.
- **JavaScript / TypeScript** — for browser host integration and the JS↔Wasm marshaling discussions.
- **Rust** or **C++** — for hands-on producer-side work; Rust's `wasm-bindgen` and Emscripten's emcc cover the most common paths.
- **Compilation theory** (lightly) — understanding control-flow graphs, register allocation, and SSA helps with grasping Wasm's design choices (structured control flow, explicit local indexing).
- **POSIX systems programming** (lightly) — many WASI operations mirror POSIX (fd_read, path_open, clock_gettime).

## Complexity

- **Wasm validation**: O(n) where n is the number of instructions; linear-time and decidable.
- **Wasm execution**: instruction-by-instruction; O(1) dispatch overhead in optimizing JITs after warmup.
- **Tier-up compilation latency**: Liftoff in O(n); TurboFan in O(n × k) where k is per-function optimization complexity (typically n^1.2–n^1.5 over the function body in worst cases).
- **Bounds-check overhead**: O(1) per memory access in unoptimized code; effectively zero in optimized code with guard-page tricks on 64-bit hosts.
- **Component Model lifting/lowering**: O(size of value) — strings/lists are linear-time; records/tuples are constant-cost-per-field.

## See Also

- `webassembly` — the practical sheet (categorical opposite of this internals deep-dive).
- `polyglot` — multi-language projects often use Wasm for the polyglot ABI; the Component Model is the modern path.
- `c` — the original Wasm producer language; understanding C's memory model helps with linear memory.
- `rust` — the dominant modern Wasm producer; `wasm-bindgen` and `cargo-component` are the canonical paths.
- `javascript` — the ubiquitous Wasm host; understanding JS's typed arrays and `WebAssembly.*` API is essential for browser deployments.

## References

- **WebAssembly Specification** — `webassembly.org/spec` — the normative source for the abstract machine, type system, validation, and binary format.
- **MDN WebAssembly Documentation** — `developer.mozilla.org/en-US/docs/WebAssembly` — comprehensive practical reference for browser API and core concepts.
- **WASI Repository** — `github.com/WebAssembly/wasi` — Preview1 spec, Preview2 development, runtime conformance tests.
- **Component Model Repository** — `github.com/WebAssembly/component-model` — WIT IDL spec, Canonical ABI, Component binary format.
- **wit-bindgen** — `github.com/bytecodealliance/wit-bindgen` — official multi-language binding generator.
- **wasm-tools** — `github.com/bytecodealliance/wasm-tools` — Rust suite for parse/print/validate/compose.
- **wabt** — `github.com/WebAssembly/wabt` — reference C tools (`wat2wasm`, `wasm2wat`, `wasm-validate`, `wasm-interp`).
- **binaryen** — `github.com/WebAssembly/binaryen` — C++ optimizer (`wasm-opt`).
- **Emscripten** — `emscripten.org` — C/C++ → Wasm toolchain.
- **wasmtime** — `wasmtime.dev` — Bytecode Alliance reference standalone runtime.
- **wasmer** — `wasmer.io` — commercial standalone runtime, multi-backend (Cranelift, LLVM, Singlepass).
- **"Programming WebAssembly with Rust"** by Kevin Hoffman (Pragmatic Bookshelf, 2019) — accessible introduction with extended Rust examples.
- **WebAssembly Podcast** — `bytecodealliance.org/podcast` — interviews with spec editors and runtime authors.
- **Bytecode Alliance Blog** — `bytecodealliance.org/articles` — long-form pieces on Component Model, WASI, security architecture.
- **WasmCon Conference Talks** — annual conference; videos on YouTube cover compiler internals, runtime architecture, deployment patterns.
- **"Bringing WebAssembly outside the web"** — Kripken et al., 2019 — the original Standalone Wasm + WASI motivation paper.
- **"Mechanising and verifying the WebAssembly specification"** — Watt, 2018 — Isabelle/HOL formalization establishing soundness.
