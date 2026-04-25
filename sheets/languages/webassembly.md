# WebAssembly (Programming Language / Bytecode)

Stack-based binary instruction format with a strict type system, deterministic semantics, and a sandbox-by-default execution model — designed as a portable compilation target for languages like C, C++, Rust, Go, and AssemblyScript, originally for the web but now used universally on servers, edge runtimes, embedded devices, plug-in systems, and as the foundation for the Component Model and the WebAssembly System Interface (WASI).

## Setup

```bash
# wabt — WebAssembly Binary Toolkit (text↔binary, validate, objdump)
brew install wabt
# or
sudo apt install wabt
# Provides:
#   wat2wasm        — text format (.wat) → binary (.wasm)
#   wasm2wat        — binary (.wasm)     → text format (.wat)
#   wasm-objdump    — disassemble + dump section info
#   wasm-validate   — verify a .wasm file against the spec
#   wasm-strip      — remove custom (debug) sections
#   wasm-decompile  — produce pseudo-C from .wasm

# binaryen — wasm-opt and friends (optimizer + compiler library)
brew install binaryen
# Provides:
#   wasm-opt        — optimize a .wasm (similar role to LLVM opt)
#   wasm-as         — assemble Binaryen text → .wasm
#   wasm-dis        — disassemble .wasm → Binaryen text
#   wasm-ctor-eval  — pre-evaluate constructors at build time
#   wasm-merge      — merge multiple modules
#   wasm2js         — emit equivalent JS

# emscripten — C/C++ → wasm + JS glue (Asyncify, MAIN_MODULE, etc.)
brew install emscripten
# Provides:
#   emcc            — clang frontend producing .wasm + .js
#   em++            — C++ frontend
#   emcmake / emmake — wrappers for cmake/make
#   emrun           — run html in a browser
#   emar            — archive

# wasi-sdk — clang + sysroot targeting wasm32-wasi (no JS glue)
# Download from https://github.com/WebAssembly/wasi-sdk/releases
export WASI_SDK=/opt/wasi-sdk
$WASI_SDK/bin/clang --target=wasm32-wasi hello.c -o hello.wasm

# wasm-tools — Bytecode Alliance binary toolchain (validate, dump, components)
cargo install wasm-tools
# Provides:
#   wasm-tools validate
#   wasm-tools print / parse
#   wasm-tools strip
#   wasm-tools component new / wit / embed / compose
#   wasm-tools smith   — fuzz-test generator
#   wasm-tools mutate  — mutational fuzzing
#   wasm-tools dump

# Runtimes — pick one (or several)
curl https://wasmtime.dev/install.sh -sSf | bash         # wasmtime (reference)
curl https://get.wasmer.io -sSfL | sh                   # wasmer
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash  # WasmEdge
cargo install wasm3                                      # wasm3 (interpreter, embedded)

# Sanity check
wat2wasm --version
wasm-opt --version
wasmtime --version
wasm-tools --version
```

## .wasm vs .wat

```bash
# .wasm   — the binary format (what runtimes load)
# .wat    — the text format (what humans read/write)
# They are isomorphic — a perfect round-trip via wat2wasm/wasm2wat.

# Round-trip example
cat > add.wat <<'EOF'
(module
  (func (export "add") (param $a i32) (param $b i32) (result i32)
    local.get $a
    local.get $b
    i32.add))
EOF

wat2wasm add.wat -o add.wasm
wasm2wat add.wasm -o add.roundtrip.wat
diff add.wat add.roundtrip.wat   # may differ in formatting only

# A binary file always begins with the magic bytes
#   0x00 0x61 0x73 0x6D    (the bytes "\0asm")
# followed by the version
#   0x01 0x00 0x00 0x00    (version 1)
xxd add.wasm | head -1
# 00000000: 0061 736d 0100 0000 ...

# Print binary as text without writing a file
wasm-tools print add.wasm
```

## Module Structure

```bash
# A wasm module is a sequence of sections in a *fixed order*.
# A validator rejects out-of-order sections.

# Section IDs and order:
#   0  custom        any number, anywhere — ignored by semantics (debug names live here)
#   1  type          function signatures
#   2  import        host-supplied funcs/tables/memories/globals
#   3  function      function indices → type indices
#   4  table         table declarations
#   5  memory        linear memory declarations
#   6  global        global declarations
#   7  export        named exports (func/table/memory/global)
#   8  start         the optional start function
#   9  element       table initializers
#  10  code          actual function bodies (paired with section 3)
#  11  data          memory initializers
#  12  datacount     (since bulk-memory) — count of data segments

# Inspect the section table:
wasm-objdump -h module.wasm
# Output:
#   Sections:
#     Type start=0x0000000a end=0x00000016 (size=0x0000000c) count: 1
#     Function start=0x00000018 end=0x0000001a (size=0x00000002) count: 1
#     Export   start=0x0000001c end=0x00000026 (size=0x0000000a) count: 1
#     Code     start=0x00000028 end=0x00000031 (size=0x00000009) count: 1
```

## Value Types

```bash
# Numeric (MVP):
#   i32   — 32-bit integer (signedness is per-instruction, not per-type)
#   i64   — 64-bit integer
#   f32   — IEEE 754 single
#   f64   — IEEE 754 double

# Vector (post-MVP, "simd128" proposal):
#   v128  — 128-bit packed value (interpreted by lane-shaped instructions)

# Reference (post-MVP, "reference-types" proposal):
#   funcref    — opaque function reference (formerly called "anyfunc")
#   externref  — opaque host reference (e.g., a JS object)

# Sizes & alignment:
#   i32, f32, funcref32 lanes  → 4 bytes, 4-byte preferred alignment
#   i64, f64                    → 8 bytes, 8-byte preferred alignment
#   v128                        → 16 bytes, 16-byte preferred alignment

# Signedness ambiguity:
#   The TYPE i32 is just 32 raw bits. The INSTRUCTION encodes the sign:
#     i32.div_s   — signed
#     i32.div_u   — unsigned
#     i32.lt_s    — signed compare
#     i32.lt_u    — unsigned compare
#   Bitwise ops (and/or/xor/shl) don't care about sign.
#   Right shift does:  shr_s (arithmetic) vs shr_u (logical).
```

## Functions — Type Signatures and Locals

```bash
# Function in text form:
#   (func $name (param $a i32) (param $b i32) (result i32)
#     (local $tmp i32)
#     ... body ...)

# Anonymous parameters (positional):
#   (func (param i32 i32) (result i32) ...)

# Multiple results (since multi-value):
#   (func (param i32) (result i32 i32) ...)

# Locals are zero-initialized.
# Index space within a function: params first, then locals.
#   (func (param i32 i32) (local i64 i32))
#   index 0 = param 0 (i32)
#   index 1 = param 1 (i32)
#   index 2 = local 0 (i64)
#   index 3 = local 1 (i32)

# Read/write locals:
#   local.get $name      — push value of local on stack
#   local.set $name      — pop value into local
#   local.tee $name      — pop, store, then push back (chained)

# Reading params is identical to reading locals.
# Function call:
#   call $func              — direct call by index
#   call_indirect (type $T) — table-based call, type-checked at runtime

# Example with all features:
(func $sum_two (param $a i32) (param $b i32) (result i32)
  (local $tmp i32)
  local.get $a
  local.get $b
  i32.add
  local.tee $tmp
  i32.const 1
  i32.add)
```

## Numeric Instructions — i32

```bash
# Constants and basic arithmetic
i32.const N            # push 32-bit literal
i32.add                # pop b, pop a, push a+b   (wrap)
i32.sub                # pop b, pop a, push a-b   (wrap)
i32.mul                # pop b, pop a, push a*b   (wrap)
i32.div_s              # signed division   (TRAP on /0 or INT_MIN/-1)
i32.div_u              # unsigned division (TRAP on /0)
i32.rem_s              # signed remainder  (TRAP on /0; rem_s(INT_MIN, -1) = 0)
i32.rem_u              # unsigned remainder(TRAP on /0)

# Bitwise
i32.and
i32.or
i32.xor
i32.shl                # left shift    (shift count mod 32)
i32.shr_s              # arithmetic right shift
i32.shr_u              # logical right shift
i32.rotl
i32.rotr

# Comparisons (all push 0 or 1, an i32 boolean)
i32.eqz                # ==0   (unary)
i32.eq                 # ==
i32.ne                 # !=
i32.lt_s   i32.lt_u    # <
i32.le_s   i32.le_u    # <=
i32.gt_s   i32.gt_u    # >
i32.ge_s   i32.ge_u    # >=

# Bit-population unary
i32.clz                # leading zeros
i32.ctz                # trailing zeros
i32.popcnt             # set-bit count
```

## Numeric Instructions — i64

```bash
# Same family as i32 — every i32.* op has an i64.* sibling.
i64.const N
i64.add  i64.sub  i64.mul
i64.div_s  i64.div_u
i64.rem_s  i64.rem_u
i64.and  i64.or  i64.xor
i64.shl  i64.shr_s  i64.shr_u
i64.rotl i64.rotr
i64.eqz  i64.eq  i64.ne
i64.lt_s i64.lt_u i64.le_s i64.le_u
i64.gt_s i64.gt_u i64.ge_s i64.ge_u
i64.clz  i64.ctz  i64.popcnt

# Crossing widths:
i64.extend_i32_s       # sign-extend an i32 → i64
i64.extend_i32_u       # zero-extend an i32 → i64
i32.wrap_i64           # truncate i64 → i32 (drops upper 32 bits)
```

## Numeric Instructions — f32 / f64

```bash
# Constants
f32.const 3.14
f64.const 3.14159265358979

# Arithmetic — IEEE 754 semantics, never trap
f32.add  f32.sub  f32.mul  f32.div
f32.sqrt
f32.min  f32.max          # IEEE-style: NaN propagates
f32.ceil f32.floor f32.trunc f32.nearest   # rounding modes
f32.abs  f32.neg
f32.copysign

# Same for f64:
f64.add  f64.sub  f64.mul  f64.div
f64.sqrt  f64.min  f64.max
f64.ceil  f64.floor  f64.trunc  f64.nearest
f64.abs   f64.neg   f64.copysign

# Comparisons (push i32 0/1)
f32.eq  f32.ne  f32.lt  f32.le  f32.gt  f32.ge
f64.eq  f64.ne  f64.lt  f64.le  f64.gt  f64.ge

# Conversions to/from integers
i32.trunc_f32_s        # TRAP if NaN, +/-Inf, or out of i32 range
i32.trunc_f32_u
i32.trunc_f64_s   i32.trunc_f64_u
i64.trunc_f32_s   i64.trunc_f32_u
i64.trunc_f64_s   i64.trunc_f64_u
f32.convert_i32_s f32.convert_i32_u
f32.convert_i64_s f32.convert_i64_u
f64.convert_i32_s f64.convert_i32_u
f64.convert_i64_s f64.convert_i64_u
f32.demote_f64
f64.promote_f32

# Saturating versions (since the "non-trapping float-to-int" proposal)
# Replace TRAP with clamping behavior
i32.trunc_sat_f32_s  i32.trunc_sat_f32_u
i32.trunc_sat_f64_s  i32.trunc_sat_f64_u
i64.trunc_sat_f32_s  i64.trunc_sat_f32_u
i64.trunc_sat_f64_s  i64.trunc_sat_f64_u
```

## Conversions

```bash
# Integer width
i32.wrap_i64                  # i64 → i32 (low 32 bits)
i64.extend_i32_s              # signed promote i32 → i64
i64.extend_i32_u              # zero-extend i32 → i64

# Sign-extension within a width (since the "sign-extension-ops" proposal)
i32.extend8_s                 # treat low byte as i8, sign-extend in i32
i32.extend16_s
i64.extend8_s
i64.extend16_s
i64.extend32_s

# Float ↔ Int (trapping versions)
i32.trunc_f32_s   i32.trunc_f32_u
i32.trunc_f64_s   i32.trunc_f64_u
i64.trunc_f32_s   i64.trunc_f32_u
i64.trunc_f64_s   i64.trunc_f64_u
f32.convert_i32_s f32.convert_i32_u
f32.convert_i64_s f32.convert_i64_u
f64.convert_i32_s f64.convert_i32_u
f64.convert_i64_s f64.convert_i64_u

# Float width
f32.demote_f64                # f64 → f32 (rounding)
f64.promote_f32               # f32 → f64 (exact)

# Reinterpret — bit-level cast (no value transformation)
i32.reinterpret_f32           # raw bits of f32 as i32
i64.reinterpret_f64
f32.reinterpret_i32
f64.reinterpret_i64

# Rule of thumb:
#   convert_*    → numeric (rounds, may saturate or trap)
#   reinterpret  → bit-cast only (always loss-free)
```

## Control Flow

```bash
# Structured control: no arbitrary jumps. Branches target labels.
# A label is created by  block | loop | if  and ends at  end.

# block — branch target = AFTER the block (forward jump)
#   block        $L           ;; label $L = end of block
#     ...
#     br $L                   ;; jumps to end of block
#   end

# loop — branch target = START of the loop (back-edge)
#   loop         $L
#     ...
#     br_if $L                ;; jump back to top if condition non-zero
#   end

# if/else
#   i32.const 1
#   if (result i32)
#     i32.const 10
#   else
#     i32.const 20
#   end                       ;; pushes 10 or 20

# Branches
br $label                     # unconditional
br_if $label                  # pop i32, branch if != 0
br_table $L0 $L1 $L2 ... $Ldefault   # pop index → branch to L[i] (or default)

# Function-level
return                        # pop result, return from function
call $f                       # direct call to function $f
call_indirect (type $T)       # pops table-index, type-checked at runtime

# Unreachable
unreachable                   # always traps; type-polymorphic (any result type)

# Drop and select
drop                          # discard top of stack
select                        # pop cond, pop b, pop a → push (cond?a:b)
```

## Block Types and Multi-Value Results

```bash
# A block can declare a result (or even params + multiple results)
# since the "multi-value" proposal (now in the spec).

# Empty block
block
  nop
end

# Block with one result
block (result i32)
  i32.const 42
end                           ;; stack: [42]

# Block with multiple results
block (result i32 i32)
  i32.const 1
  i32.const 2
end                           ;; stack: [1, 2]

# Block with parameters AND results — referenced via a type
(type $swap_t (func (param i32 i32) (result i32 i32)))
i32.const 1
i32.const 2
block (type $swap_t)
  ;; stack on entry: [1 2]
  ;; ... do something ...
end

# A br to a labelled block jumps to its END;
# any values needed by the block's RESULT must be on the stack at the br site.
```

## Memory

```bash
# Linear memory: a single mutable, zero-indexed array of bytes.
# Pages are 64 KiB (65536 bytes). Modules typically declare:
#   (memory $m 1 4)    ;; initial 1 page (64 KiB), max 4 pages (256 KiB)
#
# A module can have at most ONE memory in MVP; the "multi-memory" proposal lifts that.
# The host can also import a memory:
#   (import "env" "memory" (memory 1))

# Loads (push value)
i32.load     offset=0 align=2     ;; aligned 4-byte load → i32
i32.load8_s  offset=0 align=0     ;; load 1 byte, sign-extend to i32
i32.load8_u  offset=0 align=0     ;; load 1 byte, zero-extend
i32.load16_s offset=0 align=1     ;; load 2 bytes signed
i32.load16_u
i64.load     offset=0 align=3
i64.load8_s  / i64.load8_u
i64.load16_s / i64.load16_u
i64.load32_s / i64.load32_u
f32.load
f64.load

# Stores (pop address + value)
i32.store    offset=0 align=2
i32.store8                          ;; low byte
i32.store16
i64.store
i64.store8 / i64.store16 / i64.store32
f32.store
f64.store

# Effective address = base (i32) + immediate offset.
# alignment is a hint (declared in log2 form); mismatched alignment is legal but slower.
# Out-of-bounds load/store TRAPs.

# Bulk memory (since the "bulk-memory" proposal)
memory.copy        # pop dst, src, len → copy bytes within memory
memory.fill        # pop dst, value, len → memset
memory.init $seg   # initialize memory from a passive data segment
data.drop $seg     # mark a passive data segment unusable
```

## Memory.grow / Memory.size

```bash
# Memory can be GROWN at runtime, but never SHRUNK.

# Push current size in PAGES (64 KiB each)
memory.size                  ;; → i32 (current page count)

# Grow by N pages — push N first
memory.grow                  ;; pop delta, return previous size
                             ;; OR -1 (0xFFFFFFFF) on failure

# Typical guard pattern:
#   i32.const 4
#   memory.grow
#   i32.const -1
#   i32.eq
#   if
#     unreachable            ;; or trap explicitly
#   end

# JS host equivalent:
#   memory.grow(n) extends the underlying ArrayBuffer
#   ⚠ The old ArrayBuffer is detached after grow! Re-create your views:
#       const view = new Uint8Array(instance.exports.memory.buffer);
```

## Globals

```bash
# Globals declare a typed value visible to all functions in the module.
# Mutability is part of the type:
#   (global $g_const i32 (i32.const 7))
#   (global $g_mut (mut i32) (i32.const 0))

# Imported globals
(import "env" "version" (global $v i32))

# Exported globals
(global $count (export "count") (mut i32) (i32.const 0))

# Read/write
global.get $name             # push value
global.set $name             # pop value into global (must be (mut ...))

# Initializer expressions are limited to constants and global.get on imports.
```

## Tables

```bash
# A table is a typed array of references — usually funcref.
# In MVP, exactly one table per module is allowed and only of type funcref.
# The "reference-types" proposal lifts both restrictions.

# Declare
(table $t 8 16 funcref)              ;; initial 8, max 16

# Imported table
(import "env" "table" (table 8 16 funcref))

# Element segment — initialize a slice of a table at instantiation
(elem (i32.const 0) $f0 $f1 $f2)

# call_indirect — invoke a function via table index
(type $sig (func (param i32) (result i32)))
local.get $idx              ;; index into table
call_indirect (type $sig)   ;; runtime type-check vs $sig

# Active vs passive segments (since reference-types):
#   active  — runs at instantiation (the form above)
#   passive — declared with (elem funcref ...) and explicitly applied via table.init

# Modifying tables at runtime (since reference-types)
table.get   $t              ;; pop index → push reference
table.set   $t              ;; pop index, ref → store
table.size  $t              ;; current size
table.grow  $t              ;; pop init-ref, delta → previous size or -1
table.fill  $t
table.copy  $dst $src
table.init  $t $seg
elem.drop   $seg
```

## Imports & Exports

```bash
# IMPORT a function, table, memory, or global from the host.
(import "module-name" "field-name" (func $h_log (param i32)))
(import "env" "memory" (memory 1))
(import "env" "table" (table 8 funcref))

# EXPORT — make available to the host.
(func $add (export "add") (param i32 i32) (result i32) ...)
(memory (export "memory") 1 4)
(global (export "version") i32 (i32.const 1))

# JavaScript host glue — the "importObject" mirrors the import structure:
#   const importObject = {
#     env: {
#       log: (x) => console.log(x),
#       memory: new WebAssembly.Memory({ initial: 1 }),
#     },
#   };
#   const { instance } = await WebAssembly.instantiate(bytes, importObject);

# Wasmtime/WASI:
#   The host must provide WASI imports under "wasi_snapshot_preview1"
#   (or, with the component model, satisfy the world's required imports).
```

## Start Function

```bash
# A module can declare ONE start function — runs once at instantiation,
# AFTER imports are satisfied and BEFORE any export is callable.
#
# Type signature is fixed: (func (param) (result))   ;; no params, no result.

(module
  (func $init
    ;; e.g., zero out a buffer
    i32.const 0
    i32.const 0
    i32.const 1024
    memory.fill)
  (start $init))

# Use cases:
#   - Pre-populate globals or memory
#   - Bootstrap a runtime (Rust uses this for ctor code in some setups)
#
# Pitfall: a panic in the start function poisons the module — instantiation
# itself fails with whatever trap occurred.
```

## Validation

```bash
# Every wasm module must pass formal validation BEFORE execution.
# Validation is single-pass and decidable. Failures: CompileError / ValidationError.

# What the verifier checks (high level):
#   - Section ordering and uniqueness
#   - All referenced indices are in bounds (functions, types, locals, globals,
#     tables, memories, data segments, element segments, labels)
#   - Each function body type-checks under a stack discipline:
#       * Operand types match instruction signatures
#       * Branch targets receive correct stack types
#       * No "underflow" — popping more than the available stack
#       * No leftover values on stack at function return (must match result types)
#   - Control structures are well-nested (no escaping a block from the wrong depth)
#   - Memory and table accesses use valid alignment hints (≤ natural alignment)
#   - Constant initializer expressions only reference imported globals / consts
#   - call_indirect type indices are valid
#   - At most one memory and one table in MVP (relaxed by proposals)

# CLI validation:
wasm-validate module.wasm
wasm-tools validate module.wasm

# Example failures:
#   "type mismatch in i32.add"
#   "function body must end with end opcode"
#   "section size mismatch"
#   "function index out of bounds"
#   "global is immutable"
```

## Traps

```bash
# A trap is an unrecoverable runtime error — the host reports it (e.g., as a
# JS WebAssembly.RuntimeError) and the module instance is no longer usable.

# Sources of traps:
unreachable            # explicit "this is impossible" instruction
i32.div_s   / div_u    # divide by 0
i32.rem_s   / rem_u    # divide by 0
i64.div_*   / rem_*
i32.trunc_f32_s        # NaN, +/-Inf, or out of int range
i32.trunc_f64_s        # (use trunc_sat_* to clamp instead of trap)
i32.load   / store     # OOB access — addr+offset+size > memory.size * 64KiB
i64.load   / store     # OOB
f32.load   / store     # OOB
call_indirect          # null entry, OOB, or signature mismatch
table.get / table.set  # OOB
memory.fill / .copy    # OOB ranges
memory.init / table.init  # OOB into segment

# Stack overflow is also a trap (host-defined limit).

# In a JS host you'll see:
#   RuntimeError: unreachable executed
#   RuntimeError: divide by zero
#   RuntimeError: out of bounds memory access
#   RuntimeError: integer overflow
#   RuntimeError: indirect call signature mismatch
```

## Tooling — wat2wasm / wasm2wat

```bash
# wat2wasm — text → binary
wat2wasm hello.wat -o hello.wasm

# Useful flags:
#   -o, --output=FILE          output filename
#   -v, --verbose              print actions
#   --debug-names              embed local/function names into the binary
#   --no-canonicalize-leb128s  preserve original LEB128 sizes (for diffing)
#   --enable-FEATURE / --disable-FEATURE  toggle proposals
#       (e.g., --enable-bulk-memory, --enable-simd, --enable-threads,
#              --enable-reference-types, --enable-multi-value, --enable-tail-call)
#   --no-check                 skip validation (debug only — almost never use)
#   --inline-imports           inline import section in error messages

# wasm2wat — binary → text
wasm2wat hello.wasm -o hello.wat
#   --generate-names           invent symbolic names where missing
#   --no-debug-names           drop name-section names
#   --inline-exports           place exports beside their definitions
#   --inline-imports
#   --fold-exprs               render in folded (s-expression) style

# Common error: "missing end" — the .wat lacks a closing block/end opcode.
# Common error: "duplicate identifier $foo" — two locals/functions share a name.
```

## Tooling — wasm-objdump

```bash
# Like binutils objdump but for wasm. Powerful for reverse engineering.

wasm-objdump -h module.wasm                 # section headers only
wasm-objdump -x module.wasm                 # all section details
wasm-objdump -d module.wasm                 # disassemble code section
wasm-objdump --disassemble -d module.wasm   # explicit form
wasm-objdump --section CODE module.wasm     # specific section
wasm-objdump --details -x module.wasm       # full details for every section

# Useful flags:
#   -h, --headers              show section headers
#   -x, --details              dump section details
#   -d, --disassemble          disassemble code
#   -s, --full-contents        hex dump of every section
#   --section=NAME             limit to one section (CODE, EXPORT, IMPORT, ...)
#   -j, --section-name=NAME    same as --section
#   -r, --reloc                show relocations (object files only)

# Example output:
#   add.wasm:	file format wasm 0x1
#   Code Disassembly:
#   00001a func[0] <add>:
#    00001b: 20 00                      | local.get 0
#    00001d: 20 01                      | local.get 1
#    00001f: 6a                         | i32.add
#    000020: 0b                         | end
```

## Tooling — wasm-opt

```bash
# wasm-opt is binaryen's optimizer — runs SSA-style passes on a wasm binary.
# Massive size and speed wins on un-optimized modules (Rust debug, raw emcc, etc.)

# Optimization levels (analogous to -O0..-O3 plus size-tuned variants)
wasm-opt -O0 in.wasm -o out.wasm     # no optimization
wasm-opt -O1 in.wasm -o out.wasm
wasm-opt -O2 in.wasm -o out.wasm
wasm-opt -O3 in.wasm -o out.wasm     # speed-focused
wasm-opt -O4 in.wasm -o out.wasm     # max speed
wasm-opt -Os in.wasm -o out.wasm     # size-leaning
wasm-opt -Oz in.wasm -o out.wasm     # max size reduction (great for ship)

# Specific passes
wasm-opt --strip-debug          in.wasm -o out.wasm
wasm-opt --strip-producers      in.wasm -o out.wasm
wasm-opt --remove-unused-module-elements in.wasm -o out.wasm
wasm-opt --dce                  in.wasm -o out.wasm
wasm-opt --vacuum               in.wasm -o out.wasm
wasm-opt --merge-blocks         in.wasm -o out.wasm
wasm-opt --converge -O3         in.wasm -o out.wasm    # iterate to fixed point

# Toggle proposals
wasm-opt --enable-simd --enable-bulk-memory --enable-reference-types ...
wasm-opt --disable-mutable-globals ...

# Emscripten integration
wasm-opt --post-emscripten -O3 in.wasm -o out.wasm
wasm-opt -g -O0 ...             # preserve names (DWARF survives -O0..-O1; stripped at higher)

# Common error:
#   "validation failed (in wasm-opt)" — your input wasm is malformed; run wasm-validate first.
#   "[wasm-validator error in module] ... must match operand type"
#       → mismatched feature flags between your producer and wasm-opt; pass --enable-X.
```

## Tooling — wasm-tools

```bash
# Bytecode Alliance binary toolkit — Rust-based, the canonical CLI for the
# component model and WASI 0.2.

# Validate
wasm-tools validate module.wasm

# Pretty-print binary as text
wasm-tools print module.wasm

# Parse text → binary
wasm-tools parse module.wat -o module.wasm

# Strip debug + producers
wasm-tools strip module.wasm -o stripped.wasm

# Dump every section in a structured form
wasm-tools dump module.wasm

# Generate "smith" — fuzz-ready random valid modules
wasm-tools smith --num-imports 0 --max-functions 16 < /dev/urandom > rand.wasm

# Mutate an existing module
wasm-tools mutate input.wasm -o mutated.wasm --seed 42

# Component model
wasm-tools component new core.wasm \
   --adapt wasi_snapshot_preview1=adapter.wasm \
   -o component.wasm
wasm-tools component wit component.wasm        # show WIT
wasm-tools component embed world.wit core.wasm -o embedded.wasm
wasm-tools component link a.wasm b.wasm -o linked.wasm

# Diff two binaries
wasm-tools metadata show module.wasm

# Common error:
#   "expected component, got core module" — you passed a raw .wasm where a
#   component was required; wrap with `wasm-tools component new` first.
```

## Producer: Rust

```bash
# Three target triples for wasm:
rustup target add wasm32-unknown-unknown   # browser, no system imports
rustup target add wasm32-wasi              # WASI preview1 (legacy alias of wasm32-wasip1)
rustup target add wasm32-wasip1            # WASI preview1 (canonical name from 1.78+)
rustup target add wasm32-wasip2            # WASI preview2 (component model)

# Plain build
cargo build --target wasm32-unknown-unknown --release

# Minimum-viable lib for browser exports
cat > src/lib.rs <<'EOF'
# #[no_mangle]
# pub extern "C" fn add(a: i32, b: i32) -> i32 { a + b }
EOF
# Cargo.toml:
# [lib]
# crate-type = ["cdylib"]

# Strip + size flags in Cargo.toml
# [profile.release]
# opt-level = "z"   # 's' or 'z' for size; '3' for speed
# lto = true
# codegen-units = 1
# strip = true      # removes symbols
# panic = "abort"   # smaller, no unwinding tables

# rustflags for additional shrinkage
RUSTFLAGS="-C link-arg=--export-table -C link-arg=--import-memory" \
  cargo build --target wasm32-unknown-unknown --release

# wasm-pack — toolchain wrapper for browser/Node packaging
cargo install wasm-pack
wasm-pack build --target web        # outputs pkg/ with .wasm + JS glue + .d.ts
wasm-pack build --target nodejs
wasm-pack build --target bundler
wasm-pack build --target no-modules

# wasm-bindgen — high-level JS interop (strings, closures, structs)
# Cargo.toml: wasm-bindgen = "0.2"
# src/lib.rs:
#   use wasm_bindgen::prelude::*;
#   #[wasm_bindgen]
#   pub fn greet(name: &str) -> String { format!("Hello, {name}") }

# WASI binary (preview1)
cargo build --target wasm32-wasip1 --release
wasmtime target/wasm32-wasip1/release/myapp.wasm

# Component (preview2) — use cargo-component
cargo install cargo-component
cargo component new --lib mylib
cargo component build --release
wasm-tools component wit target/wasm32-wasip2/release/mylib.wasm

# Common error:
#   "the trait `wasm_bindgen::convert::FromWasmAbi` is not implemented"
#       → exposed an unsupported type from wasm_bindgen; wrap in JsValue or Box.
```

## Producer: C / C++

```bash
# Emscripten — for browser-targeting (heavy JS glue + filesystem emulation)
emcc hello.c -o hello.html        # html + js + wasm
emcc hello.c -o hello.js          # js + wasm
emcc hello.c -O3 -o hello.js

# Important emcc flags (passed via -s KEY=VALUE):
emcc hello.c \
    -s WASM=1                     # emit .wasm (default) vs asm.js
    -s STANDALONE_WASM=1          # produce a runnable-without-JS wasm
    -s ENVIRONMENT=web            # web | webview | worker | node | shell
    -s MODULARIZE=1               # wrap output in a module factory
    -s EXPORT_ES6=1               # emit ES6 module
    -s EXPORTED_FUNCTIONS='["_main","_my_func"]'
    -s EXPORTED_RUNTIME_METHODS='["ccall","cwrap"]'
    -s ALLOW_MEMORY_GROWTH=1      # memory.grow allowed
    -s INITIAL_MEMORY=16MB
    -s MAXIMUM_MEMORY=512MB
    -s TOTAL_STACK=64KB
    -O3 -Oz                       # speed and/or size
    -g                            # debug info (DWARF)
    -gsource-map                  # source maps for browser DevTools
    -o app.js

# emcmake / emmake — wrap cmake/make
emcmake cmake -S . -B build
emmake make -C build

# wasi-sdk — for raw WASI binaries with no JS glue
$WASI_SDK/bin/clang \
    --target=wasm32-wasi \
    --sysroot=$WASI_SDK/share/wasi-sysroot \
    -O2 hello.c -o hello.wasm
wasmtime hello.wasm

# Common error:
#   "wasm-ld: error: undefined symbol: _start" — you compiled as a library;
#   either define main(), or pass `-mexec-model=reactor`, or `-Wl,--no-entry`.
```

## Producer: Go

```bash
# Browser target — needs syscall/js + wasm_exec.js
GOOS=js GOARCH=wasm go build -o main.wasm
cp $(go env GOROOT)/misc/wasm/wasm_exec.js .

# index.html (minimal):
#   <script src="wasm_exec.js"></script>
#   <script>
#     const go = new Go();
#     WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
#       .then(r => go.run(r.instance));
#   </script>

# WASI target (Go 1.21+)
GOOS=wasip1 GOARCH=wasm go build -o main.wasm
wasmtime main.wasm

# Tinygo — much smaller binaries (10-100x), strict subset
brew install tinygo
tinygo build -target=wasm -o main.wasm main.go            # browser
tinygo build -target=wasi -o main.wasm main.go            # WASI
tinygo build -target=wasm-unknown -o main.wasm main.go    # bare wasm

# Tinygo size advantage example: a hello-world is ~40 KB instead of ~2 MB.

# Common error:
#   "panic: syscall/js: call of Value.Call on null" — you tried to use a
#   browser API (DOM/window) under a non-browser host. Run with the matching
#   wasm_exec.js or switch to GOOS=wasip1 + a WASI runtime.
```

## Producer: AssemblyScript

```bash
# AssemblyScript — TypeScript-like input that compiles to clean wasm.
# No runtime DOM; provides a small standard library.

npm install --save-dev assemblyscript
npx asinit .                 # generate boilerplate (assembly/, asconfig.json)
npm run asbuild              # debug + release builds

# Direct compile
npx asc assembly/index.ts -o build/release.wasm --optimize
npx asc assembly/index.ts -o build/debug.wasm --debug --sourceMap

# Common asc flags:
#   --target release|debug        named target from asconfig.json
#   --optimize                    full optimization
#   --runtime stub|incremental    GC strategy
#   --noAssert                    drop assert calls in release
#   --bindings raw|esm            JS binding generation

# Example AssemblyScript:
#   export function add(a: i32, b: i32): i32 { return a + b; }

# Common error:
#   "TS2322: Type 'i64' is not assignable to type 'i32'" — AssemblyScript is
#   strict about widths; insert an explicit cast like  `<i32>x`.
```

## Browser Host

```bash
# Object surface:
#   WebAssembly.Module       — compiled but not instantiated
#   WebAssembly.Instance     — bound to imports, has .exports
#   WebAssembly.Memory       — view-able as ArrayBuffer
#   WebAssembly.Table        — array of references (funcref)
#   WebAssembly.Global       — typed mutable/immutable cell
#   WebAssembly.Tag          — for exception handling proposal
#   WebAssembly.Exception    — exception object

# Modern instantiation — overlaps fetch and compile, fast path
const { instance } = await WebAssembly.instantiateStreaming(
  fetch("module.wasm"),
  { env: { log: x => console.log(x) } }
);
instance.exports.add(40, 2);

# Fallback for non-application/wasm Content-Type
const bytes = await (await fetch("module.wasm")).arrayBuffer();
const { instance } = await WebAssembly.instantiate(bytes, importObject);

# Sharing a host-allocated memory
const memory = new WebAssembly.Memory({ initial: 1, maximum: 16 });
const importObject = { env: { memory } };

# Reading from the wasm memory after a call:
const view = new Uint8Array(instance.exports.memory.buffer);
const ptr  = instance.exports.alloc(64);
view.set(new TextEncoder().encode("hi"), ptr);

# Growing memory — note the buffer is REPLACED on grow
instance.exports.memory.grow(2);
const newView = new Uint8Array(instance.exports.memory.buffer);   // re-create

# Module from prebuilt bytes (workers)
postMessage({ type: "wasm", bytes });
// receiver:  WebAssembly.compile(e.data.bytes).then(mod => ...)

# Common error:
#   "TypeError: WebAssembly.instantiate(): Import #N module=\"env\" function=\"foo\" error: function import requires a callable"
#       → your importObject is missing a function the module expects.
```

## Node Host

```bash
# Node has full WebAssembly.* support since v8 and stable WASI since v20.

# Plain wasm
const fs = require("node:fs/promises");
const bytes = await fs.readFile("./module.wasm");
const { instance } = await WebAssembly.instantiate(bytes, importObject);
instance.exports.add(40, 2);

# WASI module via node:wasi (Node 20+)
const { WASI } = require("node:wasi");
const wasi = new WASI({
  version: "preview1",
  args: process.argv.slice(2),
  env: process.env,
  preopens: { "/": "/tmp" },
  returnOnExit: true,
});
const wasm = await fs.readFile("./app.wasm");
const mod = await WebAssembly.compile(wasm);
const inst = await WebAssembly.instantiate(mod, wasi.getImportObject());
wasi.start(inst);

# CLI flag (older Node):
#   node --experimental-wasi-unstable-preview1 ...

# Common error:
#   "Error [ERR_INVALID_ARG_VALUE]: WASI options.version" — you forgot
#   `version: "preview1"` (required since Node 20).
```

## Deno / Bun Hosts

```bash
# Both runtimes ship WebAssembly + WASI built-in.

# Deno — wasi via the standard library
import { Context } from "https://deno.land/std/wasi/snapshot_preview1.ts";
const wasi = new Context({ args: Deno.args, env: Deno.env.toObject() });
const bin  = await Deno.readFile("./app.wasm");
const mod  = await WebAssembly.compile(bin);
const inst = await WebAssembly.instantiate(mod, { wasi_snapshot_preview1: wasi.exports });
wasi.start(inst);

# Bun — Bun.file + WebAssembly are first-class
const wasm = await Bun.file("./app.wasm").arrayBuffer();
const { instance } = await WebAssembly.instantiate(wasm, importObject);

# Bun also bundles wasmtime-like WASI behavior under the hood:
bun run app.wasm                  # if you use Bun's CLI runner

# Common error:
#   Deno: "TypeError: Method WebAssembly.Instance.prototype.exports called on incompatible receiver"
#       → you passed a Module to WebAssembly.instantiate twice; pass bytes OR a Module, not Instance.
```

## WASI — preview1

```bash
# wasi_snapshot_preview1 — the historical baseline, broadly supported.
# Module name in import statements: "wasi_snapshot_preview1".

# Function shape: numeric return code (errno-like) where 0 = success.
# Strings live in linear memory and are passed as (ptr, len) pairs.

# Frequently used imports (selected):
fd_read       (fd, iovs_ptr, iovs_len, nread_ptr) -> errno
fd_write      (fd, iovs_ptr, iovs_len, nwritten_ptr) -> errno
fd_close      (fd) -> errno
fd_seek       (fd, offset, whence, newoffset_ptr) -> errno
path_open     (dirfd, dirflags, path_ptr, path_len, oflags, fs_rights, fs_rights_inh, fdflags, fd_ptr) -> errno
path_unlink_file (dirfd, path_ptr, path_len) -> errno
args_get      (argv_ptr, argv_buf_ptr) -> errno
args_sizes_get (argc_ptr, argv_buf_size_ptr) -> errno
environ_get
environ_sizes_get
clock_time_get(clock_id, precision, time_ptr) -> errno
random_get    (buf_ptr, buf_len) -> errno
proc_exit     (rval) -> noreturn

# Run a WASI binary
wasmtime run --dir=. ./app.wasm                       # grant cwd as a preopen
wasmer run --mapdir=/data:/host/data ./app.wasm
wasmedge --dir=. ./app.wasm

# Note: WASI preview1 is "frozen" — no new APIs. Use preview2 for sockets, http, etc.
```

## WASI — preview2 / Components

```bash
# WASI 0.2 (preview2) is built on the COMPONENT MODEL.
# Instead of one giant import namespace, it defines small WIT-typed interfaces:
#
#   wasi:cli/run                    — entry point
#   wasi:cli/environment
#   wasi:cli/exit
#   wasi:cli/stdin / stdout / stderr
#   wasi:io/streams                 — input/output
#   wasi:io/poll                    — async-style poll
#   wasi:filesystem/types / preopens
#   wasi:sockets/tcp / udp / network
#   wasi:http/types / outgoing-handler / incoming-handler
#   wasi:random/random
#   wasi:clocks/wall-clock / monotonic-clock
#   wasi:logging/logging

# Compile Rust to a preview2 component
cargo build --target wasm32-wasip2 --release
# Output is ALREADY a component; no need for "component new".

# Compile preview1 binary AND adapt it to preview2
cargo build --target wasm32-wasip1 --release
wasm-tools component new \
    target/wasm32-wasip1/release/app.wasm \
    --adapt wasi_snapshot_preview1=wasi_snapshot_preview1.command.wasm \
    -o app.component.wasm

# Run with wasmtime
wasmtime run --wasi preview2 app.component.wasm
# (newer wasmtime auto-detects components — flag often unnecessary)

# Common error:
#   "expected core module, got component" — you fed a component into a path
#   expecting raw wasm; switch to a runtime that supports components or
#   extract the core module via `wasm-tools component extract`.
```

## Component Model

```bash
# WIT (WebAssembly Interface Types) — a small IDL for typed cross-language interfaces.

# Example .wit:
#   package local:greet@0.1.0;
#
#   interface api {
#     greet: func(name: string) -> string;
#     stats: func() -> record { count: u64, avg-length: float32 };
#     enum color { red, green, blue }
#     resource file {
#       constructor(path: string);
#       read: func(n: u32) -> list<u8>;
#     }
#   }
#
#   world greet-world {
#     export api;
#     import wasi:clocks/monotonic-clock@0.2.0;
#   }

# Bindings via wit-bindgen
cargo install wit-bindgen-cli

# Generate Rust guest bindings
wit-bindgen rust ./wit --out-dir src/bindings

# Generate C bindings
wit-bindgen c ./wit --out-dir bindings

# Generate Go bindings
wit-bindgen tiny-go ./wit --out-dir bindings   # via tinygo
# (Standard Go has its own bindgen, see github.com/bytecodealliance/wasm-tools-go)

# Generate JS/TS bindings
wit-bindgen js ./wit --out-dir bindings

# Generate Python bindings
wit-bindgen py ./wit --out-dir bindings        # used by componentize-py

# Compose components — link a producer to a consumer (statically)
wac plug producer.wasm consumer.wasm -o app.wasm
# or:
wasm-tools compose producer.wasm -d consumer.wasm -o app.wasm
```

## Memory Sharing JS↔Wasm

```bash
# Wasm exports use only numeric types (i32/i64/f32/f64).
# To pass strings/structs, both sides agree on a memory layout and the host
# reads/writes through the exported memory.

# Pattern: pass-by-pointer-and-length

# 1) Wasm side exports an allocator
(func $alloc (export "alloc") (param $n i32) (result i32) ...)
(func $free  (export "free")  (param $p i32) ...)

# 2) JS side encodes a string and copies into wasm memory
function passString(instance, s) {
  const enc = new TextEncoder();
  const bytes = enc.encode(s);
  const ptr = instance.exports.alloc(bytes.length);
  const view = new Uint8Array(instance.exports.memory.buffer);
  view.set(bytes, ptr);
  return [ptr, bytes.length];
}
const [ptr, len] = passString(instance, "hello");
instance.exports.greet(ptr, len);
instance.exports.free(ptr);

# 3) Reading a string FROM wasm
function readString(instance, ptr, len) {
  const view = new Uint8Array(instance.exports.memory.buffer, ptr, len);
  return new TextDecoder().decode(view);
}

# Watch out: memory.grow detaches the previous buffer.
# After ANY call into wasm that may grow memory, re-create your typed views.

# wasm-bindgen automates ALL of this for Rust↔JS at the cost of glue size.
# The Component Model + JS host bindings (jco, componentize-js) automates it
# for any source language that targets components.
```

## Performance Considerations

```bash
# 1) Boundary crossings JS↔Wasm are the slowest thing.
#    - Batch work; avoid call-per-pixel or call-per-byte loops.
#    - Pass arrays via shared memory rather than per-element calls.

# 2) SIMD via v128 (post-MVP "simd128" proposal)
#    - 4-8x speedup for vectorizable inner loops.
#    - Browser support is broad in 2024+.
#    - Feature-detect:
#        WebAssembly.validate(simdProbeBytes)
#    - Compile: rustc --target wasm32-unknown-unknown -C target-feature=+simd128

# 3) Bulk memory ops (memory.copy / memory.fill)
#    - Replace hand-rolled byte loops; the runtime uses memcpy internally.
#    - Enable: --enable-bulk-memory in wabt/binaryen.

# 4) Reference types (externref, funcref) avoid table swaps.

# 5) AOT compilation
#    wasmtime compile module.wasm -o module.cwasm
#    wasmtime run --allow-precompiled module.cwasm

# 6) Tier compilation
#    Most runtimes start with a fast baseline compiler (Cranelift basic) and
#    re-compile hot functions optimized; profile-guided is in progress.

# 7) Threads (post-MVP) — require shared memory + atomics
#    (memory $m 1 1 shared)
#    Browser requires COOP/COEP headers to enable SharedArrayBuffer.

# 8) Reduce module size — smaller modules compile and decode faster.
#    Use wasm-opt -Oz, --strip-debug, --strip-producers.
```

## Debugging

```bash
# Browser DevTools — recent Chromium and Firefox have a wasm debugger.
# Compile with debug info:
emcc -g -gsource-map app.c -o app.js                  # DWARF + source map
cargo build --target wasm32-wasi --profile=dev        # Rust dev = -g

# Chrome extension: "C/C++ DevTools Support (DWARF)" enables source-level stepping
#   on .wasm with embedded DWARF.

# console.log via host import (no native print in wasm)
(import "env" "log" (func $log (param i32)))

# wasm3 — small interpreter ideal for tracing
wasm3 --func add --arg 40 --arg 2 module.wasm

# wasmtime tracing
RUST_LOG=trace wasmtime run module.wasm 2> trace.log
wasmtime run --profile=jitdump --profile-output=jit.dump module.wasm
perf inject -j -i jit.dump -o jit.out.dump

# Stripping debug info BEFORE shipping
wasm-tools strip module.wasm -o stripped.wasm
wasm-opt --strip-debug --strip-producers in.wasm -o out.wasm

# Common error:
#   "no source mapping found" — you stripped debug info before opening DevTools.
#   Rebuild with -g (or set the source-map URL).
```

## Size Optimization

```bash
# Final-build size checklist (smaller is faster to load AND faster to compile).

# 1) wasm-opt at the end
wasm-opt -Oz \
   --strip-debug --strip-producers \
   --remove-unused-module-elements \
   in.wasm -o out.wasm

# 2) Source-language flags
# Rust: in Cargo.toml
# [profile.release]
# opt-level = "z"
# lto = true
# codegen-units = 1
# strip = true
# panic = "abort"
# Reduce std presence: features = []  (cut serde features, etc.)

# Emscripten / clang
emcc -Oz -flto \
   -s INITIAL_MEMORY=131072 \
   -s ALLOW_MEMORY_GROWTH=1 \
   -s EXPORTED_FUNCTIONS='["_main"]' \
   --no-entry app.c -o app.wasm

# 3) Strip extras
wasm-tools strip module.wasm -o module.stripped.wasm
wasm-strip module.wasm   # wabt's tool, removes custom sections only

# 4) wasm-snip — for Rust panic strings (advanced)
cargo install wasm-snip
wasm-snip --snip-rust-panicking-code in.wasm -o out.wasm

# 5) Compress for transport (the runtime decompresses)
brotli -Z module.wasm        # produces module.wasm.br for HTTP
zstd --ultra -22 module.wasm

# Common gain breakdown (typical):
#   Raw debug Rust          → 8 MB
#   Release + opt-level=z   → 800 KB
#   wasm-opt -Oz            → 600 KB
#   wasm-snip + strip       → 480 KB
#   brotli -Z over the wire → 180 KB
```

## Common Error Messages and Fixes

```bash
# 1) RuntimeError: unreachable executed
#    Cause: an `unreachable` instruction ran. In Rust, this is the lowering
#           of a panic!() — including `unwrap()` on None/Err, slice OOB, etc.
#    Fix:   Reduce unwraps; build with `panic = "abort"`; embed a panic hook
#           in browser builds:
#             console_error_panic_hook::set_once();

# 2) RuntimeError: out of bounds memory access
#    Cause: a load/store touched an address outside the allocated memory.
#    Fix:   Bounds-check pointer arithmetic; ensure ALLOW_MEMORY_GROWTH=1 is
#           set if you rely on growth; never reuse pointers across grow events
#           (see memory.grow detachment note).

# 3) RuntimeError: integer overflow / divide by zero
#    Cause: trapping conversion (e.g., i32.trunc_f32_s on a NaN/Inf/large value)
#           or signed div_s/rem_s on INT_MIN/-1.
#    Fix:   Use i32.trunc_sat_f32_s (saturating); pre-check the divisor; in Rust
#           use checked_div / wrapping_div.

# 4) TypeError: WebAssembly.instantiate(): Import #N module="env" function="foo" error
#    Cause: importObject does not contain the function the module declared.
#    Fix:   Check the module's import section (`wasm-objdump -j Import -x`)
#           and supply every entry under the right module/field path.

# 5) CompileError: WebAssembly.instantiate(): Wasm validation error
#    Cause: malformed binary (e.g., wrong section order) or a feature not
#           enabled by the host (SIMD, threads, GC, exception handling).
#    Fix:   Run `wasm-validate module.wasm` locally; when validation passes,
#           your runtime is older than the proposal — use a flag like
#             `wasmtime run --wasm-features=simd,threads ...`
#           or rebuild without the feature (`-C target-feature=-simd128`).

# 6) TypeError: WebAssembly.Memory(): could not allocate memory
#    Cause: requested initial size exceeds host limits.
#    Fix:   Lower INITIAL_MEMORY; on the browser, very large memories are
#           disallowed without crossOriginIsolated.

# 7) wasm-ld: error: undefined symbol: ___main_argc_argv
#    Cause: a wasi-sdk reactor build expects no main; or vice-versa.
#    Fix:   For libraries: pass `-mexec-model=reactor` and `-Wl,--no-entry`.
#           For executables: define `int main(int argc, char**argv)`.

# 8) TypeError: import object field 'wasi_snapshot_preview1' is not an Object
#    Cause: running a WASI module without a WASI shim.
#    Fix:   Use a WASI runtime (wasmtime/wasmer/wasmedge) or instantiate
#           through node:wasi / Deno's std/wasi / a polyfill.

# 9) RuntimeError: indirect call signature mismatch
#    Cause: call_indirect target function's signature differs from the type
#           supplied to call_indirect.
#    Fix:   Make sure every entry in the table that may be called with type T
#           actually has type T; check element segments and cast carefully.

# 10) "expected component, got core module"
#    Cause: passed a raw wasm where the host expected a component.
#    Fix:   wasm-tools component new core.wasm --adapt ... -o component.wasm
```

## Common Gotchas (Broken + Fixed)

```bash
# 1) i32-only addressing — pointers are u32 (32-bit linear memory in MVP).
#    BROKEN: in Rust on wasm32, `usize` is 32-bit; storing a *mut Foo as i64
#            and casting back loses the upper zero bytes ambiguously.
#    FIXED:  always use u32/i32 for pointer-sized ABI fields; treat usize as 32-bit.

# 2) No native strings — manual encode/decode.
#    BROKEN: const ptr = exports.greet("hi");   // passes a string-as-pointer? No.
#    FIXED:  encode UTF-8, alloc, copy, pass (ptr, len), free; OR use wasm-bindgen.

# 3) Memory cannot shrink.
#    BROKEN: assuming you can `memory.shrink()`. There is no such instruction.
#    FIXED:  design for monotonic growth; reuse buffers (memory.fill 0 to "clear");
#            for truly bursty workloads, instantiate a fresh module instance.

# 4) memory.grow detaches the underlying ArrayBuffer.
#    BROKEN: const view = new Uint8Array(memory.buffer);
#            instance.exports.workThatGrows();
#            view[0] = 1;   // throws TypeError: detached ArrayBuffer
#    FIXED:  re-create the view AFTER any call that may grow memory, OR use a
#            getter helper that always rebuilds the typed view.

# 5) Same module on Node WASI vs Deno WASI vs wasmtime — small behavioral diffs.
#    BROKEN: timestamps, file mode bits, error codes for fs ops can differ;
#            relying on errno values in user code breaks portability.
#    FIXED:  compare to constants from the WASI spec (ERRNO_NOENT etc.); test
#            against multiple runtimes in CI.

# 6) Sandbox by default — no DOM, no FS, no network unless host imports.
#    BROKEN: a Rust crate calls std::fs::File::open and you ran it in the
#            browser → unreachable trap.
#    FIXED:  in the browser, route I/O through host imports; on WASI, run with
#            `--dir=...` so the runtime exposes the right preopens.

# 7) Feature flags must agree across the toolchain.
#    BROKEN: cargo built with simd128, but wasm-opt run without --enable-simd
#            → wasm-opt fails validation.
#    FIXED:  pass matching `--enable-FEATURE` flags to wabt and binaryen tools.

# 8) Floating-point trapping vs saturating conversion.
#    BROKEN: using i32.trunc_f64_s on a value above 2^31 traps.
#    FIXED:  use i32.trunc_sat_f64_s; in Rust use `as i32` (saturating since 1.45)
#            via the `nontrapping-fptoint` feature.

# 9) Mismatched call_indirect type.
#    BROKEN: entry expects (i32) -> i32 but you call_indirect with type (i32 i32) -> i32.
#    FIXED:  unify type indices; double-check element segments after refactors.

# 10) JS importObject field is a value, not a function.
#    BROKEN: importObject = { env: { log: console.log } }   // unbound `this`
#    FIXED:  importObject = { env: { log: (x) => console.log(x) } }
```

## Security Model

```bash
# Wasm provides STRONG isolation guarantees; the host provides authority.

# 1) Memory safety — wasm code cannot access host memory; linear memory is
#    bounded and indexed by 32-bit (or 64-bit) integers; OOB → trap.

# 2) Control-flow integrity — only structured branches; no arbitrary jumps;
#    indirect calls are type-checked; the call stack is separate from data.

# 3) Capability-based imports — wasm has NO ambient authority. The host decides
#    what each instance can see:
#      - No --dir flag → no filesystem access
#      - No --env flag → no environment variables
#      - No socket import → no network
#      - No DOM import → no DOM (browser is implicitly allowing only what the
#        glue code wraps; pure wasm-from-wasm cannot reach window.*)

# 4) WASI 0.2 / Component Model raises the bar:
#    - Each interface (wasi:filesystem, wasi:sockets) is a separate import.
#    - Hosts can deny entire categories without partial enabling.
#    - Resources (handles) are unforgeable from wasm code.

# 5) Deterministic execution — given identical inputs, same wasm produces same
#    outputs (modulo nondeterminism explicitly granted, e.g., random_get).

# Recommendations:
#   - Use WASI 0.2 components for new server-side work.
#   - Avoid preview1 for greenfield projects; treat it as legacy.
#   - Distribute precompiled .cwasm only over signed channels; the binary
#     itself is portable, but trust the producer.
#   - Treat WASI capabilities like Unix file descriptors — least-privilege.
```

## Idioms

```bash
# 1) Canonical exported function pattern (Rust)
# #[no_mangle]
# pub extern "C" fn process(in_ptr: *const u8, in_len: u32, out_ptr: *mut u8, out_cap: u32) -> i32 {
#     let input = unsafe { core::slice::from_raw_parts(in_ptr, in_len as usize) };
#     let buf = unsafe { core::slice::from_raw_parts_mut(out_ptr, out_cap as usize) };
#     match do_work(input, buf) {
#         Ok(n) => n as i32,
#         Err(_) => -1,
#     }
# }

# 2) malloc / free for variable-sized data
# #[no_mangle]
# pub extern "C" fn alloc(n: u32) -> *mut u8 {
#     let mut v = Vec::<u8>::with_capacity(n as usize);
#     let p = v.as_mut_ptr();
#     core::mem::forget(v);
#     p
# }
# #[no_mangle]
# pub extern "C" fn free(p: *mut u8, n: u32) {
#     unsafe { Vec::from_raw_parts(p, 0, n as usize) };
# }

# 3) ABI for strings — always (ptr: i32, len: i32) UTF-8.

# 4) Opaque handles — never return a pointer; return an integer ID, keep the
#    real object in a side table inside wasm. Prevents the host from forging
#    pointers.

# 5) Error-via-out-pointer
#    fn fallible(out: *mut u8, out_len: *mut u32) -> i32  // 0=ok, nonzero=errno

# 6) Lifetime tied to instance — when the host drops the WebAssembly.Instance,
#    all wasm allocations vanish with it. Don't try to free across instances.

# 7) For Rust, wrap exports with a #[wasm_bindgen] for browser or WIT for
#    components. Hand-rolled raw exports are fine for tight, low-overhead APIs.
```

## Standard Library Equivalents

```bash
# Wasm has no built-in OS, so common stdlib facilities are provided by host imports.

# Print to stdout/stderr
#   Browser:   import a logger from JS:
#                (import "env" "log" (func $log (param i32 i32)))
#              and call with (ptr, len) into UTF-8 buffer.
#   WASI:      use fd_write on fd=1 (stdout) or fd=2 (stderr).
#   Component: wasi:cli/stdout writeln.

# Time
#   WASI preview1: clock_time_get(CLOCK_MONOTONIC|REALTIME, precision, &out_ns)
#   WASI preview2: wasi:clocks/monotonic-clock.now() / wasi:clocks/wall-clock.now()
#   Browser:       Date.now() bridged via host import.

# Random
#   WASI preview1: wasi_snapshot_preview1.random_get(buf_ptr, buf_len)
#   WASI preview2: wasi:random/random.get-random-bytes(len) -> list<u8>
#   Browser:       crypto.getRandomValues bridged via host import.

# Filesystem
#   WASI preview1: path_open / fd_read / fd_write
#   WASI preview2: wasi:filesystem/types resources (descriptor.read-via-stream(...))

# Sockets
#   WASI preview1: NOT AVAILABLE (use WasmEdge wasi-net or wasi-sockets proposal)
#   WASI preview2: wasi:sockets/tcp + wasi:sockets/network

# Spawn / threads
#   No CreateProcess. The "threads" proposal allows child workers via host glue;
#   shared memory + atomics enable in-process threading.
```

## Tips

```bash
# - Prefer wasm32-wasip2 (Component Model) for new server-side work; treat
#   wasm32-wasip1 as legacy though still widely supported.
# - Always finish with `wasm-opt -O3` (or -Oz for size); 30-50% reductions are routine.
# - Use the Component Model for polyglot systems — WIT eliminates hand-written FFI.
# - Precompile (`wasmtime compile`) on the build host and ship .cwasm to cut
#   cold-start from hundreds of ms to microseconds.
# - Treat WASI capabilities like Unix fds — grant least-privilege explicitly.
# - In the browser, prefer WebAssembly.instantiateStreaming() to overlap
#   fetch with compile; only fall back when the server's Content-Type is wrong.
# - Feature-detect SIMD and threads at load; ship two artifacts and pick
#   the best one at runtime.
# - wasm-bindgen generates TypeScript definitions automatically — your JS
#   callers get full type safety for free.
# - Keep modules small with #![no_std] in Rust or -Oz in C; sub-100KB modules
#   are routinely achievable for narrow APIs.
# - Benchmark wasmtime, wasmer, and WasmEdge with your actual workload before
#   committing to a runtime. Differences can be >2x.
# - Component resources (handles) cross language boundaries safely without
#   raw pointers — prefer them over passing memory addresses.
# - The Bytecode Alliance repos (`bytecodealliance/wasmtime`, `wasm-tools`,
#   `wit-bindgen`) are the canonical source of truth for wasm semantics.
# - For edge/serverless, wasm cold-starts are 10-100x faster than containers —
#   that is the defining deployment advantage.
# - When in doubt, run `wasm-validate` first; ~80% of "weird runtime errors"
#   are actually invalid binaries from a misconfigured pipeline.
```

## See Also
- c, rust, go, python, java, javascript, typescript, polyglot, make, bash, regex

## References
- [WebAssembly Core Specification](https://webassembly.github.io/spec/core/)
- [MDN — WebAssembly](https://developer.mozilla.org/en-US/docs/WebAssembly)
- [WebAssembly/WASI](https://github.com/WebAssembly/WASI)
- [bytecodealliance/wit-bindgen](https://github.com/bytecodealliance/wit-bindgen)
- [WebAssembly/binaryen](https://github.com/WebAssembly/binaryen)
- [WebAssembly/wabt](https://github.com/WebAssembly/wabt)
- [wasm-tools](https://github.com/bytecodealliance/wasm-tools)
- [The Rust and WebAssembly Book](https://rustwasm.github.io/docs/book/)
- [Emscripten Documentation](https://emscripten.org/docs/)
- [Component Model Documentation](https://component-model.bytecodealliance.org/)
- [Wasmtime Documentation](https://docs.wasmtime.dev/)
- [WasmEdge Documentation](https://wasmedge.org/docs/)
- [Bytecode Alliance](https://bytecodealliance.org/)
