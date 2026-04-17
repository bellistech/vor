# WebAssembly (WASM, WASI, Component Model)

Stack-based binary instruction format for a portable virtual machine, originally designed for the web but now a universal compile target for sandboxed, near-native execution across browsers, servers, edge, and embedded — standardized by W3C, with the WebAssembly System Interface (WASI) 0.2 and the Component Model enabling polyglot, capability-based server-side applications.

## Core Concepts

### Module Structure
```
A Wasm module consists of:
  types      — function signatures
  imports    — host-provided functions, memories, globals, tables
  functions  — code defined in the module
  tables     — arrays of references (often function pointers)
  memories   — linear byte-addressable memory (64 KiB pages)
  globals    — typed mutable/immutable values
  exports    — functions, memories, globals exposed to host
  elem       — table initialization
  data       — linear memory initialization
  start      — function run on instantiation
```

### Text Format (WAT)
```wasm
(module
  (import "env" "print" (func $print (param i32)))
  (memory (export "memory") 1)
  (func $add (export "add") (param $a i32) (param $b i32) (result i32)
    local.get $a
    local.get $b
    i32.add)
  (func $main (export "main")
    i32.const 40
    i32.const 2
    call $add
    call $print))
```

## Toolchain

### wabt (WebAssembly Binary Toolkit)
```bash
# Install
brew install wabt

# Compile text → binary
wat2wasm module.wat -o module.wasm

# Disassemble
wasm2wat module.wasm -o module.wat

# Validate
wasm-validate module.wasm

# Human-readable objdump
wasm-objdump -x module.wasm
wasm-objdump -d module.wasm  # disassemble

# Strip, optimize
wasm-strip module.wasm
```

### Compiling From Source Languages
```bash
# Rust → Wasm (browser target)
rustup target add wasm32-unknown-unknown
cargo build --target wasm32-unknown-unknown --release

# Rust → WASI
rustup target add wasm32-wasip2
cargo build --target wasm32-wasip2 --release

# Go → Wasm (browser)
GOOS=js GOARCH=wasm go build -o main.wasm

# Go → WASI
GOOS=wasip1 GOARCH=wasm go build -o main.wasm

# C/C++ → Wasm via Emscripten
emcc hello.c -o hello.html
emcc hello.c -O3 -s WASM=1 -s EXPORTED_FUNCTIONS='["_add"]' -o hello.wasm

# C → WASI via wasi-sdk
clang --target=wasm32-wasi --sysroot=$WASI_SDK/share/wasi-sysroot \
  hello.c -o hello.wasm

# AssemblyScript (TS-like → Wasm)
npx asc module.ts --outFile module.wasm --optimize

# Zig → Wasm
zig build-exe main.zig -target wasm32-wasi -O ReleaseSmall
```

### wasm-tools (Component Model + binary tools)
```bash
# Install
cargo install wasm-tools

# Print a Wasm binary as WAT
wasm-tools print module.wasm

# Parse WAT → binary
wasm-tools parse module.wat -o module.wasm

# Component Model
wasm-tools component new core.wasm --adapt wasi_snapshot_preview1=adapter.wasm -o component.wasm
wasm-tools component wit component.wasm   # show WIT interface
wasm-tools component embed world.wit core.wasm -o embedded.wasm
wasm-tools validate component.wasm

# Optimize
wasm-tools strip module.wasm -o stripped.wasm
```

### Binaryen (wasm-opt)
```bash
# Install
brew install binaryen

# Optimize (often 30-50% size reduction)
wasm-opt -O3 module.wasm -o module-opt.wasm

# Specific optimizations
wasm-opt --dce --vacuum --merge-blocks module.wasm -o out.wasm

# Generate bindings
wasm-opt --emit-source-map module.wasm -o out.wasm
```

## Runtimes

### Wasmtime (Bytecode Alliance, reference implementation)
```bash
# Install
curl https://wasmtime.dev/install.sh -sSf | bash

# Run a WASI module
wasmtime run module.wasm arg1 arg2

# Grant filesystem access (capability-based)
wasmtime run --dir=./data module.wasm

# Environment variables
wasmtime run --env FOO=bar module.wasm

# Preview 2 components
wasmtime run --wasi preview2 component.wasm

# Network (socket access)
wasmtime run --wasi inherit-network --wasi allow-ip-name-lookup module.wasm

# Compile AOT for faster startup
wasmtime compile module.wasm -o module.cwasm
wasmtime run --allow-precompiled module.cwasm

# Configuration file
cat <<'EOF' > wasmtime.toml
[wasi]
inherit-env = true
preopened-dirs = [{ host = ".", guest = "/" }]
EOF
```

### Wasmer
```bash
# Install
curl https://get.wasmer.io -sSfL | sh

# Run
wasmer run module.wasm

# Package and publish
wasmer publish
wasmer run wasmer/python
```

### WasmEdge (CNCF sandbox, optimized for cloud-native)
```bash
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash
wasmedge module.wasm
```

### Embedded in Host (Rust with wasmtime crate)
```rust
use wasmtime::{Engine, Module, Store, Instance, Linker};
use wasmtime_wasi::WasiCtxBuilder;

fn main() -> wasmtime::Result<()> {
    let engine = Engine::default();
    let mut linker = Linker::new(&engine);
    wasmtime_wasi::add_to_linker_sync(&mut linker, |s| s)?;

    let wasi = WasiCtxBuilder::new().inherit_stdio().build_p1();
    let mut store = Store::new(&engine, wasi);

    let module = Module::from_file(&engine, "module.wasm")?;
    let instance = linker.instantiate(&mut store, &module)?;

    let add = instance.get_typed_func::<(i32, i32), i32>(&mut store, "add")?;
    let result = add.call(&mut store, (40, 2))?;
    println!("{}", result);
    Ok(())
}
```

## WASI (WebAssembly System Interface)

### WASI Versions
```
wasi_snapshot_preview1  (WASI 0.1 / preview1)
  - POSIX-like filesystem, clocks, random, args, env
  - Widely supported, still the default for most toolchains

wasip2 (WASI 0.2, stable Jan 2024)
  - Built on Component Model
  - Worlds and interfaces defined in WIT
  - Replaces monolithic snapshot with modular components

WASI interfaces (0.2):
  wasi:cli          command-line apps
  wasi:io           streams, polling
  wasi:filesystem   file/directory access
  wasi:sockets      TCP/UDP/IP
  wasi:http         HTTP client/server
  wasi:random       randomness
  wasi:clocks       monotonic/wall-clock
  wasi:logging      structured logs
```

### Capability-Based Security
```bash
# WASI is capability-based: no ambient authority
# Modules receive ONLY what the host explicitly grants

# Example: module needs file access
wasmtime run \
  --dir=/data::/app/data \    # host:/data mapped to guest:/app/data
  --env API_KEY=secret \
  --wasi inherit-stdio=false \
  --wasi allow-network=false \
  module.wasm

# Contrast with Unix: no --dir → module cannot open any file
# No --env → module sees empty env
# No network flag → sockets fail with permission error
```

## Component Model

### WIT (Wasm Interface Type) Language
```wit
// greet.wit — defines an interface
package local:greet;

interface api {
  greet: func(name: string) -> string;
  stats: func() -> record {
    count: u64,
    avg-length: float32,
  };
}

world greet-world {
  export api;
}
```

### Building a Component
```bash
# Rust with cargo-component
cargo install cargo-component
cargo component new --lib my-component
cargo component build --release

# Inspect the resulting component
wasm-tools component wit target/wasm32-wasip2/release/my_component.wasm

# Compose components (link one to another)
wasm-tools compose producer.wasm -d consumer.wasm -o composed.wasm
```

### Polyglot Composition
```
A Rust producer component can export an interface
A JavaScript consumer component can import the same interface
The Component Model handles type marshaling (strings, records, lists)
  across language boundaries with zero glue code
```

## Debugging and Observability

### Source Maps and DWARF
```bash
# Generate DWARF debug info (Rust)
cargo build --target wasm32-wasip1 --profile=dev

# Run with debugger
lldb -O 'target create --arch wasm32-wasi ./target/wasm32-wasip1/debug/app.wasm'

# Profile with wasmtime
wasmtime run --profile=jitdump --profile-output=profile.dump module.wasm
perf inject -j -i profile.dump -o profile.out.dump
```

### Tracing
```bash
# wasmtime with OpenTelemetry integration
RUST_LOG=trace wasmtime run module.wasm

# Component Model boundary tracing (wit-bindgen)
# Automatic span around every imported/exported function call
```

## Browser Integration

### Loading Wasm in JavaScript
```javascript
// Fetch + streaming compile
const { instance } = await WebAssembly.instantiateStreaming(
  fetch('module.wasm'),
  {
    env: {
      print: (x) => console.log(x),
    }
  }
);

// Call exported function
console.log(instance.exports.add(40, 2)); // 42

// Access linear memory
const memory = new Uint8Array(instance.exports.memory.buffer);
```

### wasm-bindgen (Rust)
```rust
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}
```

```bash
cargo install wasm-pack
wasm-pack build --target web
# Outputs pkg/ with .wasm + JS glue + TypeScript defs
```

## Performance

### Tier Compilation
```
Most runtimes tier code:
  1. Baseline compiler — fast compile, slow execution (cranelift-free tier)
  2. Optimizing compiler — slow compile, fast execution (Cranelift)

Wasmtime AOT mode (.cwasm):
  Compile ahead of time on dev machine
  Load precompiled bytes on production (microsecond startup)
  Requires matching CPU architecture
```

### Feature Proposals (partial, post-MVP)
```
SIMD (128-bit)          — widely supported
threads                  — requires shared memory + atomics
reference-types          — function refs, externref
bulk-memory              — memory.copy, memory.fill
tail-call                — proper TCO
memory64                 — 64-bit linear memory
gc                       — garbage-collected references
exception-handling       — structured exceptions
relaxed-simd             — faster SIMD variants
component-model          — WIT-based composition
```

## Security Model

### Sandboxing Properties
```
- Memory is linear, bounded, indexed by 32-bit (or 64-bit) integers
- No raw pointers, no arbitrary host memory access
- Control flow is structured (no arbitrary jumps; only branch-to-labels)
- Function calls are typed (indirect calls checked at runtime)
- No ambient authority (WASI capability model)
- Host resources (files, sockets) only via explicit imports
- Stack is separate from linear memory (no stack smashing into data)
```

## Tips
- Prefer `wasm32-wasip2` (Component Model) for new server-side work; `wasip1` is stable but being superseded
- Always run `wasm-opt -O3` on release artifacts; 30–50% size reduction and measurable speedup are typical
- Use the Component Model for polyglot systems — WIT interfaces eliminate hand-written FFI glue between Rust, JS, Python (componentize-py), and Go
- In production, precompile (`wasmtime compile`) and distribute `.cwasm` artifacts to cut cold-start from hundreds of milliseconds to microseconds
- Treat WASI capabilities like Unix fds — grant least-privilege explicitly; a module denied `--dir` genuinely cannot read any file
- For browser use, streaming compile via `WebAssembly.instantiateStreaming()` overlaps fetch with compilation — always prefer this over fetching the full bytes first
- SIMD and threads are not universally available — feature-detect at load and ship both variants behind a picker
- wasm-bindgen generates TypeScript definitions from Rust public items — your JS callers get full type-safety for free
- Keep modules small by avoiding std-heavy dependencies; `#![no_std]` Rust or `-Oz` C produces sub-100KB modules routinely
- For server workloads, compare Wasmtime (reference correctness), WasmEdge (performance), and Wasmer (tooling) — benchmark with your actual workload before committing
- Component Model's resource types (handles) model opaque references safely across language boundaries — use them instead of raw pointer passing
- The Bytecode Alliance's `bytecodealliance/wasmtime` repo is the canonical source of truth for Wasm feature maturity and runtime semantics
- For edge/serverless, Wasm cold starts are 10–100× faster than container cold starts — this is the defining deployment advantage

## See Also
- go, rust, languages, containers, serverless, istio, envoy, grpc, supply-chain-security

## References
- [WebAssembly Core Specification](https://webassembly.github.io/spec/core/)
- [WASI 0.2 / Preview 2](https://github.com/WebAssembly/WASI)
- [Component Model](https://component-model.bytecodealliance.org/)
- [Wasmtime](https://docs.wasmtime.dev/)
- [WasmEdge](https://wasmedge.org/docs/)
- [Bytecode Alliance](https://bytecodealliance.org/)
- [wasm-tools](https://github.com/bytecodealliance/wasm-tools)
- [WIT Language Reference](https://component-model.bytecodealliance.org/design/wit.html)
- [wasm-bindgen Guide](https://rustwasm.github.io/docs/wasm-bindgen/)
- [Binaryen (wasm-opt)](https://github.com/WebAssembly/binaryen)
