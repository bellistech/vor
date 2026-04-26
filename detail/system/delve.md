# Delve Internals

Deep dive into Delve's architecture, mechanisms, and the algorithms that make Go-aware debugging possible.

## Setup

Delve (`dlv`) exists because Go does not fit comfortably inside the C-centric debugging model that GDB and LLDB were designed to support. The Go runtime introduces concepts that have no direct analog in C, and trying to debug a Go program with GDB exposes the mismatch on every page: stack traces stop at runtime functions because GDB cannot unwind through Go's split-stack mechanism, goroutines are invisible because GDB expects an OS thread per execution context, interface variables print as opaque pointer pairs because GDB has no notion of Go's runtime type metadata, and channel state is unreadable because GDB does not know how to walk `runtime.hchan`.

Delve was started by Derek Parker in 2014 specifically to close that gap. The design premise is that the debugger must be Go-aware from the ground up — it must understand the runtime, the scheduler, the type system, and the calling convention as first-class concepts, not as ad-hoc patches on top of a C debugger. Concretely, this means:

- The debugger walks `runtime.allgs` to enumerate goroutines, treating goroutines as the primary unit of execution rather than threads.
- It understands the `g` struct, the `m` struct, and the `p` struct from the runtime, and uses them to associate a goroutine with the OS thread currently running it (or to recognize that a goroutine is parked, runnable, or dead).
- It walks the GC-tracked type metadata embedded by the compiler (`runtime._type`, `runtime.iface`, `runtime.itab`) to render interface values as their concrete types.
- It iterates maps by walking `runtime.hmap.buckets` directly, mirroring what `runtime.mapiterinit` does at runtime, instead of pretending the map is a C struct.
- It unwinds stacks across the boundary between the runtime's `morestack`/`lessstack` calls so that a goroutine that has been resized still produces a coherent backtrace.

The cost of this model is that Delve must track the Go runtime closely. When the Go team changes the layout of `runtime.hmap` between versions (which they have done several times), Delve must catch up; when the scheduler grows a new state, Delve's status renderer needs the new constant; when the compiler emits a new DWARF attribute, Delve's parser must handle it. The benefit is that in practice, on a debuggable Go binary, Delve produces correct, readable output for goroutines, channels, maps, slices, interfaces, and the rest of the language — work that simply cannot be done well by a C-model debugger.

The name "Delve" itself reflects the goal: not just attach a debugger but actually delve into the runtime structures that make Go what it is.

## Architecture

The `dlv` binary is structured as a layered system inside a single Go module. The layers are designed so that the same core can be driven from a CLI, an editor, a CI script, or a remote network client. Reading from the top down:

- **`cmd/dlv`** — the entry point, parses subcommands (`debug`, `test`, `exec`, `attach`, `connect`, `core`, `dap`, `trace`, `replay`), wires up flags, and decides which higher-level mode to run. This is intentionally thin; almost all logic lives below.
- **`pkg/terminal`** — the interactive REPL. It reads commands like `break main.go:42`, `continue`, `next`, `step`, `print x`, parses them into an AST, and dispatches to the debugger via the local API. It also handles tab completion, command aliases, history, and the column-aligned output formatting.
- **`service`** and **`service/rpc1`/`service/rpc2`/`service/dap`** — the network layer. The `service.Server` interface abstracts a transport-bearing API surface; concrete implementations are the legacy v1 JSON-RPC, the modern v2 JSON-RPC, and the Microsoft Debug Adapter Protocol (DAP) backend. Each maps RPC method calls (`RPCServer.CreateBreakpoint`, `RPCServer.Step`, `RPCServer.Eval`) onto debugger operations.
- **`pkg/debugger`** — the orchestration layer. It owns a `Target` (or several, for multi-process), a list of `Breakpoints`, the current goroutine and thread under inspection, and the `Continue`/`Step`/`Next`/`StepOut` state machine. When the RPC layer says "continue," the debugger decides whether that means resume all threads, resume just the current goroutine, run until the next line, or run until a function returns.
- **`pkg/proc`** — the process control layer, the most platform-sensitive part. It provides a `Process` interface that abstracts ptrace on Linux, Mach exceptions on macOS, and the Win32 debug API on Windows, plus the gdbserver-RSP backend for remote and `rr` integration. It exposes operations like `Halt`, `Continue`, `Step`, `WriteBreakpoint`, `EraseBreakpoint`, `ReadMemory`, `WriteMemory`, `ThreadList`, `Restart`. Above this layer, code is platform-agnostic.
- **`pkg/dwarf`** — DWARF parsing and evaluation. It reads the `.debug_info`, `.debug_line`, `.debug_loc`, `.debug_frame`, `.debug_pubnames`, `.debug_str` sections of the executable, builds an in-memory index of compilation units, functions, types, and variables, and provides DWARF-expression evaluation for variable location lookups (so that `DW_OP_fbreg -16` becomes "frame base minus 16 bytes").
- **`pkg/proc/eval`** (logically; in code split between `pkg/proc/eval.go` and helpers in `pkg/proc/variables.go`) — the Go expression evaluator. It uses `go/parser` to parse expressions like `s.field[3].Method()` into a Go AST, then walks the AST and resolves identifiers against the target's memory using DWARF metadata. This is what powers `print` and conditional breakpoints.

The result is that the `proc` layer knows how to read and write a process's memory and registers, the `dwarf` layer knows how to interpret those bytes as Go variables, the `debugger` layer knows when and why to do those things, the `service` layer exposes that to clients, and the `terminal`/IDEs above it provide the user interface. The clean separation is what allows the same logic to drive both `dlv debug` in a terminal and a VS Code remote debugging session.

## Process Control

Process control is where Delve interacts with the operating system, and the implementation is fundamentally different on each major platform. The `pkg/proc` package abstracts these into a common `Process` interface, but underneath, the mechanisms are platform-specific.

### Linux ptrace

On Linux, Delve uses the venerable `ptrace(2)` syscall, the same primitive that `gdb` and `strace` use. The relevant requests are:

- **`PTRACE_ATTACH`** — attach to a running process by PID. Requires either `CAP_SYS_PTRACE` or that the target process is a child of the tracer (or that yama allows it via `/proc/sys/kernel/yama/ptrace_scope`). After attaching, the kernel sends `SIGSTOP` to the tracee, and from that point on every signal delivered to the tracee causes the tracer to be notified via `waitpid`.
- **`PTRACE_TRACEME`** — used in the child of a `fork`/`exec` so that the child stops on the first instruction after `execve`. Delve uses this for `dlv debug` and `dlv exec` where it spawns the target itself.
- **`PTRACE_PEEKDATA`** — read a word of memory from the tracee. Delve uses this for memory reads, but on modern Linux, it actually prefers `process_vm_readv` for bulk transfers, falling back to `PTRACE_PEEKDATA` for single-word reads.
- **`PTRACE_POKEDATA`** — write a word of memory. Used to install breakpoints (overwriting an instruction with `INT3` on x86) and to write back the original byte when the breakpoint is removed.
- **`PTRACE_GETREGS` / `PTRACE_SETREGS`** — read and write the general-purpose registers. On x86_64 this returns a `user_regs_struct` containing RAX, RBX, ..., RIP, RFLAGS, etc. On ARM64, the equivalent uses `PTRACE_GETREGSET` with `NT_PRSTATUS`.
- **`PTRACE_SINGLESTEP`** — execute one instruction and stop. Delve uses this to step over a breakpoint: when execution stops on `INT3`, Delve restores the original byte, single-steps once, reinstalls the `INT3`, then continues. This is how breakpoints survive multiple hits.
- **`PTRACE_CONT`** — resume execution. The optional signal argument can be used to deliver a pending signal to the tracee, or to swallow it.
- **`PTRACE_SETOPTIONS`** — set tracing options, importantly `PTRACE_O_TRACECLONE` so that newly created threads (Go's M-threads) are automatically attached. Without this, when Go's scheduler creates a new thread to run a goroutine, the new thread would not be under Delve's control.

Delve maintains one `nativeThread` per OS thread of the target, and operates on threads individually because that is what ptrace exposes. Group operations (e.g., "stop all threads") are implemented as a loop over per-thread ptrace calls.

### macOS task_for_pid + Mach exceptions

macOS does not implement ptrace in the Linux sense — `ptrace` exists as a syscall, but is severely limited and not how serious debugging is done. Instead, debuggers use the Mach kernel API:

- **`task_for_pid`** — translate a UNIX PID into a Mach `task_t`, which is a port name for the address space. This is heavily restricted on modern macOS: the calling process must be either root, or signed with the `com.apple.security.cs.debugger` entitlement, or the target must opt in. SIP (System Integrity Protection) further restricts debugging system binaries.
- **Mach exception ports** — debug events are delivered as Mach messages on an exception port. The debugger sets itself as the target's exception handler with `task_set_exception_ports`, then receives messages when the target hits a breakpoint, segfaults, or otherwise traps.
- **`thread_get_state` / `thread_set_state`** — read and write a thread's registers via flavors like `x86_THREAD_STATE64` or `ARM_THREAD_STATE64`.
- **`mach_vm_read_overwrite` / `mach_vm_write`** — read and write the target's memory.
- **`thread_resume` / `thread_suspend`** — control execution per-thread.

Delve historically had two macOS backends: a native Mach implementation and a fallback that shelled out to `lldb-server` over the GDB Remote Serial Protocol. The native implementation was retired because Apple's restrictions on `task_for_pid` make it nearly unusable without signing and entitlement provisioning. Modern Delve on macOS uses LLDB's `debugserver` (shipped with Xcode) as the lower layer and speaks RSP to it. This is invisible to the user but means Delve on macOS depends on Xcode being installed.

### Windows DebugBreakProcess + WaitForDebugEvent

On Windows, debugging is built into the Win32 API:

- **`DebugActiveProcess`** — attach to a running process by PID, equivalent to `PTRACE_ATTACH`.
- **`CreateProcess` with `DEBUG_PROCESS` flag** — spawn a process under debugging.
- **`WaitForDebugEvent`** — block until the debugged process generates an event (exception, thread create, thread exit, DLL load, DLL unload, output debug string, breakpoint, etc.). This is the moral equivalent of `waitpid` on Linux.
- **`ContinueDebugEvent`** — resume after handling.
- **`ReadProcessMemory` / `WriteProcessMemory`** — memory access.
- **`GetThreadContext` / `SetThreadContext`** — register access via a `CONTEXT` structure.
- **`SuspendThread` / `ResumeThread`** — per-thread control.
- **`DebugBreakProcess`** — inject a breakpoint into a running process to halt it.

The Windows model is event-driven and centralized: there is one debug event loop, and the debugger handles events by class. Delve maps this into its `Process` interface so that the higher layers see the same operations regardless of platform.

### GO_DELVE_USE_NEW_PROC_API

Historically Delve had two layers of native backends — an "old" pre-Go-1.10 implementation that worked at the syscall level and a "new" implementation that uses Go's modern runtime hooks. The environment variable `GO_DELVE_USE_NEW_PROC_API` was used during the transition to opt into the new path. As of recent versions the new API is the default and the old path has largely been deleted, but you may still see references in older docs.

## The native vs gdbserver-rsp Backend

Delve supports two fundamentally different ways of talking to a target process: the **native** backend, which uses platform-specific OS APIs directly, and the **gdbserver-rsp** backend, which uses the GDB Remote Serial Protocol (RSP) to talk to a separate debug stub.

### Native

The native backend is the default on Linux. As described above, it issues ptrace syscalls directly, reads memory with `process_vm_readv`, and manages breakpoints with `PTRACE_POKEDATA`. The advantage is performance and tight integration: there is no extra process in the loop, memory reads can be batched, and the implementation can take advantage of Linux-specific features like `process_vm_readv` for fast bulk transfers.

The native backend is what runs when you do `dlv debug` on a typical Linux developer machine. It is also what runs inside containers and on most CI systems.

### gdbserver / lldb-server (RSP)

The Remote Serial Protocol (RSP) was originally designed by GDB to talk to embedded debug stubs over serial lines, but has become the lingua franca of remote debugging. RSP defines a packet format with command bytes like `g` (read all registers), `G` (write registers), `m addr,length` (read memory), `M addr,length:data` (write memory), `c` (continue), `s` (single step), `Z0,addr,kind` (set software breakpoint), `z0,addr,kind` (clear breakpoint), `vCont` (cooperative continue with per-thread actions), and many more.

Delve's `pkg/proc/gdbserial` backend speaks RSP. It can connect to:

- **`lldb-server` / `debugserver`** (Apple's RSP server, ships with Xcode) — used on macOS as described above.
- **`gdbserver`** — GNU's RSP server, used for some embedded or remote-machine scenarios.
- **`rr`** — Mozilla's record-and-replay debugger speaks RSP and is used for replay debugging via `dlv replay` (see the dlv replay section).

The advantage of the RSP backend is portability and the ability to debug across machine boundaries (RSP runs over TCP). The disadvantage is overhead: every memory read becomes a network packet, and DWARF-driven variable rendering can issue many small reads, so latency dominates on slow links.

Selection between backends is automatic in most cases — Delve picks native on Linux and Windows, RSP-via-debugserver on macOS — but can be forced with `--backend=native`, `--backend=lldb`, `--backend=rr`, or `--backend=default`.

## RPC API (JSON-RPC 2)

Delve exposes its functionality as a JSON-RPC service so that IDEs, editor plugins, and CI tools can drive it programmatically. There are two generations of this API:

### service/rpc1 (legacy v1)

The original RPC API. Methods are named `RPCServer.<MethodName>`, take a single argument struct, and return a single result struct, the standard Go net/rpc convention. The methods are documented but the API has known quirks: for instance, methods that act on a goroutine take goroutine ID as a parameter while methods that act on a thread take thread ID, and the relationship is not always obvious. Some return values are pointers when they should be values, leading to nil-vs-empty ambiguity.

v1 is preserved for backwards compatibility with old clients but new development targets v2.

### service/rpc2 (modern v2)

The current default. v2 fixes the API ergonomics and adds methods that were missing in v1:

- **`RPCServer.CreateBreakpoint`** — create a breakpoint at a file/line, function, or address. The argument struct includes the `Cond` field for conditional breakpoints, `HitCondition` for hit-count conditions, `Tracepoint` to mark this as a tracepoint, `Stacktrace` to capture a stack trace on hit, `LoadArgs` and `LoadLocals` to capture variable values on hit.
- **`RPCServer.Step`** — single-step one source line, descending into function calls.
- **`RPCServer.Next`** — single-step one source line, stepping over function calls.
- **`RPCServer.StepOut`** — run until the current function returns.
- **`RPCServer.Continue`** — resume execution; returns when a breakpoint is hit, the program exits, or a signal arrives.
- **`RPCServer.Eval`** — evaluate a Go expression in the context of a goroutine and stack frame; returns a Variable.
- **`RPCServer.ListGoroutines`** — enumerate goroutines, with filtering and pagination.
- **`RPCServer.Stacktrace`** — produce a stack trace for a goroutine, optionally with locals and arguments.
- **`RPCServer.SetVariable`** — assign a value to a variable in the target.
- **`RPCServer.ListBreakpoints`** — enumerate active breakpoints.
- **`RPCServer.AmendBreakpoint`** — modify an existing breakpoint (e.g., change its condition).

Each method has a request/response struct, and the implementation lives in `service/rpc2/server.go`. The struct shapes are stable across Delve releases and are what client libraries (the official Go `delve` library, the Python `dlv-py`, and the various IDE plugins) target.

The wire format is JSON-RPC 2.0 (despite the package name v2 referring to Delve's API generation, not JSON-RPC 2.0; the two coincided), so each request looks like:

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "RPCServer.Eval",
  "params": [{"Expr": "x.field", "Scope": {"GoroutineID": 1, "Frame": 0}}]
}
```

and the response is:

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "result": {"Variable": {"Name": "", "Value": "42", "Type": "int", ...}}
}
```

## JSON-RPC over Unix Socket

When `dlv` is invoked with `--headless`, it does not start the interactive REPL; instead it listens on a network endpoint and serves the JSON-RPC API. The endpoint can be either a TCP socket (`--listen=:2345`) or a Unix domain socket (`--listen=unix:///tmp/dlv.sock`). Unix sockets are preferred for local IDE integration because they are faster and bypass the TCP stack entirely, and because filesystem permissions provide natural access control.

JetBrains GoLand and the VS Code Go extension both speak this protocol. When you click "Debug" in GoLand, the IDE spawns `dlv` with `--headless`, waits for the listen socket to come up, then opens a JSON-RPC connection and starts driving it. The IDE issues `RPCServer.CreateBreakpoint` for each breakpoint set in the editor, `RPCServer.Continue` when the user clicks the green arrow, `RPCServer.Eval` to populate the variable inspector, and so on.

The headless mode supports multi-client operation via `--accept-multiclient`. Without this flag, when the first client disconnects, the dlv process exits. With it, dlv stays alive and accepts subsequent connections, which is useful when the IDE momentarily loses connection or when multiple tools want to attach.

## DAP Mode

The Debug Adapter Protocol (DAP) is Microsoft's debugger interface specification, designed originally for VS Code but now used by other editors. DAP is also JSON-based but has a different shape from Delve's native JSON-RPC. The key differences:

- DAP uses a **request/response/event** model with explicit `seq` numbers for sequencing, instead of JSON-RPC's request/response with id correlation.
- DAP has **standardized commands** like `setBreakpoints`, `continue`, `stackTrace`, `variables`, `evaluate` — defined by Microsoft and shared across all DAP-speaking debuggers (Python's `debugpy`, JavaScript's `js-debug`, .NET's `vsdbg`, and so on).
- DAP carries a **rich event stream**: `stopped`, `continued`, `thread`, `output`, `breakpoint`, `module`. The debugger pushes events to the client without being asked, which is essential for streaming output and notifying when execution stops.
- DAP encodes **scopes** explicitly: when the IDE asks for variables, it asks for variables in scope `Local` or `Arguments` or `Global`, not just "variables for goroutine 7 frame 0."

Delve runs DAP via `dlv dap`, which starts a server speaking DAP on a TCP socket, or via the more common pattern:

```
dlv --headless --listen=:38697 --api-version=2 --accept-multiclient
```

combined with VS Code's `launch.json` `mode: "remote"`. The VS Code Go extension internally uses `dlv-dap` which is a DAP-native build of Delve.

The DAP backend lives in `service/dap/server.go` and translates DAP commands into Delve's internal operations, often by calling into the same underlying functions as the JSON-RPC backend.

## Breakpoint Mechanism

A software breakpoint is, mechanically, an instruction overwrite. The CPU has a dedicated trap instruction that, when executed, raises an exception (`SIGTRAP` on Linux, `EXC_BREAKPOINT` on macOS, `EXCEPTION_BREAKPOINT` on Windows) which the kernel delivers to the debugger. By replacing one byte (or one instruction word) of the target's code with the trap instruction, the debugger arranges for control to transfer to itself when execution reaches that location.

### x86 / x86_64

On x86 and x86_64 the trap is `INT3`, encoded as the single byte `0xCC`. To set a breakpoint:

1. Read the current byte at the target address into `Breakpoint.OriginalData`.
2. Write `0xCC` to that address using `PTRACE_POKEDATA` (or `WriteProcessMemory` on Windows).
3. Resume the target.
4. When the target hits the breakpoint, the kernel signals the debugger with `SIGTRAP`.
5. The debugger's signal handler reads the instruction pointer (which now points one past the `INT3`, because INT3 has length 1 and the CPU advances RIP), decrements RIP by 1 to point at the original instruction location, and looks up the breakpoint.
6. To resume past the breakpoint, the debugger writes the original byte back, sets RIP to that address, single-steps, then re-installs the `INT3` and continues.

This last point is subtle: the breakpoint must be erased before stepping or else the single-step will trap again immediately. The state during the brief moment after stepping but before re-installing the `INT3` is when other threads could in principle race past the breakpoint. Delve handles this by stopping all threads (the "stop-the-world" mode) during the step, which it does using ptrace on Linux.

### ARM64

On ARM64 the trap is `BRK #imm`, encoded as a 4-byte instruction. The encoding for `BRK #0` is `D4200000` in little-endian order (the lowest byte at the lowest address). The procedure mirrors x86 but with a 4-byte read/write instead of a 1-byte read/write. ARM64 does not have x86's "RIP-after-the-trap" quirk in the same form, but the debugger still must restore the original instruction word, single-step, and re-install.

### Hardware Breakpoints

Delve supports software breakpoints by default. Hardware breakpoints (using DR0–DR3 on x86 or the ARM64 watchpoint registers) are used for **watchpoints** — breaking on data access rather than instruction execution. Watchpoints are limited in number (4 on x86, 4 on ARM64) and are managed via a separate code path in `pkg/proc`.

### Breakpoint Storage

Each breakpoint is represented as a `Breakpoint` struct in `pkg/proc/breakpoints.go`:

```go
type Breakpoint struct {
    FunctionName string
    File         string
    Line         int
    Addr         uint64
    OriginalData []byte
    Kind         BreakpointKind
    Cond         ast.Expr
    HitCondition string
    HitCount     map[int]uint64
    Tracepoint   bool
    Goroutine    bool
    Stacktrace   int
    LoadArgs     *LoadConfig
    LoadLocals   *LoadConfig
    Variables    []string
    LogMessage   string
}
```

`OriginalData` holds the bytes that were overwritten so they can be restored. `Cond` is an optional condition (see next section). `HitCount` is a map from goroutine ID to hits. `Tracepoint` is the auto-resume flag. `LoadArgs`/`LoadLocals` are configurations for what to capture when the breakpoint hits.

## Conditional Breakpoints

A conditional breakpoint stops only when an associated expression evaluates to `true`. The mechanism is straightforward:

1. The breakpoint hits as usual (the trap fires, the kernel signals the debugger).
2. Delve's signal handler discovers the breakpoint and sees that `Cond` is non-nil.
3. The condition is evaluated using Delve's `eval` package, in the context of the goroutine that hit the breakpoint and the topmost stack frame.
4. If the condition evaluates to `true`, Delve stops as normal and returns control to the user (or the IDE).
5. If it evaluates to `false`, Delve restores the original byte, single-steps, re-installs the `INT3`, and continues — all without notifying the user.

The condition is parsed once when the breakpoint is created (using `go/parser.ParseExpr`, since conditions are Go expressions, not separate condition syntax) and stored as an AST. Each evaluation walks the AST against the live target memory.

The performance cost of a conditional breakpoint is therefore proportional to:

- the cost of one round-trip through ptrace and one signal delivery (microseconds on Linux),
- plus the cost of evaluating the condition (which depends on what variables it touches; a simple `i == 100` is very fast, while `s.deep.path.lookup() == "x"` may walk many pointers and is slow),
- plus the cost of restoring the byte, single-stepping, and re-installing if the condition is false.

For high-fire-rate locations, conditional breakpoints can dramatically slow the target. The standard advice is to use them sparingly and prefer tracepoints (which auto-resume without user interaction) or `dlv trace` (which avoids breakpoints altogether for entry/exit hooks).

## Tracepoints

A tracepoint is a breakpoint that automatically continues execution after capturing data. Internally a tracepoint is just a `Breakpoint` with `Tracepoint: true`. When such a breakpoint hits:

1. Delve evaluates `LoadArgs` and `LoadLocals` to capture variable values.
2. Delve optionally captures a stack trace (controlled by the `Stacktrace` field, an integer count).
3. Delve emits a tracepoint event (in the JSON-RPC API this is part of the `Continue` response; in DAP it is an `output` event).
4. Delve immediately resumes the target.

The user/IDE sees a stream of "tracepoint hit" events with captured values, but execution is not paused. This is useful for "log when X happens" without manual stepping, and for time-sensitive code where pausing would change the system's behavior (network handling, real-time loops).

A tracepoint costs roughly one breakpoint hit (microseconds) plus one or more memory reads for the captured variables. It is much cheaper than a conditional breakpoint that fires often, because there is no per-hit user round-trip.

## The eval Package

Delve's expression evaluator is what powers `print x.field[3]`, the conditional-breakpoint condition, and the `set` command. It lives across `pkg/proc/eval.go` and `pkg/proc/variables.go` and is structured as:

1. **Parsing** — Delve uses `go/parser.ParseExpr` to parse the expression into a Go AST. This is the same parser the Go compiler uses, so any valid Go expression is accepted.
2. **Type checking (light)** — Delve does limited type checking, mostly to confirm that a method call is going to the right type. It does not run the full `go/types` check because it does not have the source — it has DWARF.
3. **Walking the AST** — the evaluator recursively walks the AST. Each node type has a handler:
    - `*ast.Ident` resolves a variable name, looking first in local variables (per the DWARF info for the current frame), then in function arguments, then in globals.
    - `*ast.SelectorExpr` (`.` operator) follows a struct field by computing the field offset from DWARF and reading the bytes there.
    - `*ast.IndexExpr` indexes into an array, slice, or map.
    - `*ast.UnaryExpr` with `*` dereferences a pointer.
    - `*ast.BinaryExpr` performs arithmetic or comparison on values that have been read from memory.
    - `*ast.CallExpr` is mostly disallowed (Delve cannot, in general, call functions in the target without risking corruption), with a small whitelist for safe operations like `len`, `cap`, `complex`, `real`, `imag`, `make` (no), and a few others.
4. **Reading memory** — when the evaluator needs the value of a variable, it consults DWARF for its location (typically `DW_OP_fbreg N`, meaning frame-pointer plus offset N) and issues a memory read for the appropriate number of bytes through the proc layer.
5. **Producing a Variable** — the result is a `Variable` struct containing the name, value (a Go-formatted string), type, address, and child variables (for structs/slices/maps).

The evaluator is read-only by default; `set x = 5` goes through a separate path that writes back to the target's memory after type-checking that the assignment is valid.

## Goroutines

Goroutines are Go's user-space threads, scheduled cooperatively onto OS threads by the runtime. To Delve, goroutines are first-class entities that must be enumerated, switched between, and inspected as easily as threads in a C debugger.

The runtime exposes a slice `runtime.allgs []*g` containing every goroutine ever created and not yet garbage-collected. Delve walks this slice to enumerate goroutines:

1. Locate `runtime.allgs` via DWARF — it has known symbol name and type.
2. Read the slice header to get the data pointer and length.
3. For each pointer, read the pointed-to `g` struct.
4. Filter out dead goroutines (those with status `_Gdead`, signaling the runtime has freed them but they are still in the slice for reuse).

Each `g` struct contains:

- **`goid`** — the unique goroutine ID assigned by the runtime.
- **`atomicstatus`** — the current state (`_Gidle`, `_Grunnable`, `_Grunning`, `_Gsyscall`, `_Gwaiting`, `_Gdead`, `_Gcopystack`, `_Gpreempted`).
- **`stack`** — the bottom and top of the goroutine's current stack.
- **`stackguard0`, `stackguard1`** — the stack guards for split-stack checking.
- **`m`** — pointer to the OS thread (`m` struct) currently executing this goroutine, or nil if the goroutine is not running.
- **`sched`** — a `gobuf` containing the saved registers (PC, SP, BP, etc.) for when the goroutine is parked. This is what the runtime restores when it switches back to the goroutine.
- **`waitreason`** — a string describing why the goroutine is parked (`chan receive`, `select`, `IO wait`, `GC`, etc.).
- **`gopc`** — the program counter of the `go` statement that created this goroutine. Delve uses this to render "goroutine 7 created at main.go:23" in stack traces.

### curg/m relationship

The `curg`/`m` relationship is critical for understanding what a goroutine "is" at a given instant:

- An `m` is an OS thread (the runtime's machine).
- A `g` is a goroutine.
- `m.curg` is the goroutine currently running on that thread.
- `g.m` is the thread currently running this goroutine, or nil.

When Delve stops the process, every OS thread is paused at some instruction. For each thread, the runtime maintains `m.curg`, which tells Delve which goroutine that thread was running. Delve pairs each thread with its current goroutine, and presents the user with a unified view: "thread 5 is running goroutine 12, which is at main.go:42 in function `process`."

For parked goroutines (those not currently running on any thread), Delve uses the saved `sched.gobuf` to reconstruct the program counter and stack pointer, allowing it to produce a stack trace as if the goroutine had been stopped at its parking point.

## Goroutine Scheduling Inspection

The Go scheduler's state is exposed via:

- **`runtime.sched`** — the global scheduler struct, containing the global run queue (`runq`), the count of idle/spinning Ms, the GC trigger state, and other globals.
- **`runtime.allp []*p`** — the per-processor (P) list. Each `p` represents one logical processor; their count is `GOMAXPROCS`.
- **`p.runq`** — the per-P circular run queue of runnable goroutines (size 256, fixed).
- **`p.runnext`** — the priority slot for "the goroutine that just woke us up" (a hint for cache locality).
- **`p.gFree`** — the per-P pool of dead goroutine structs available for reuse.

Delve's goroutine listing can show which `p` each running goroutine is on, and tools like `dlv` `goroutines` `-i` (idle filter) can prune dead goroutines from view. The `goroutines` command walks `runtime.allgs` and applies optional filters (running, runnable, sleeping, by user-supplied function, etc.).

Pruning dead goroutines is important: a long-running Go program may have millions of `g` structs in `allgs`, most of them in `_Gdead` state and reusable by the runtime. Delve filters these out by default.

## Stack Unwinding for Goroutines

Stack unwinding for a goroutine differs from stack unwinding in C:

1. **The starting registers** — for a running goroutine, Delve uses the OS thread's CPU registers as the starting point. For a parked goroutine, Delve uses the saved `sched.gobuf` registers.
2. **Frame pointer convention** — Go on most architectures uses a frame pointer chain (`FP` → previous `FP`), but only when compiled with frame pointers enabled. On Go 1.7+, frame pointers are the default on amd64 and arm64.
3. **DWARF call frame info (`.debug_frame`)** — Delve consults `.debug_frame` for each function to know how to unwind across it: which register holds the return address, how to compute the previous frame's stack pointer, etc. This is the same mechanism C debuggers use, but Go-compiled binaries have a particular dialect.
4. **Split stacks** — Go goroutines start with a small stack (originally 4 KB, currently 8 KB on most platforms) and grow as needed. The runtime's `morestack` function is called automatically when a function's prologue detects that the current stack is insufficient. `morestack` allocates a larger stack, copies the existing frames, and resumes execution. `lessstack` is the converse for shrinking.
   When a stack grows, all the stack pointers in the goroutine's runtime metadata are updated. But during the brief window of the move, frames straddle two segments. Delve handles this by detecting `runtime.morestack` and `runtime.lessstack` frames and skipping them in the user-visible trace. The runtime also explicitly marks "system" frames so Delve can decide whether to show them based on user preference.
5. **Cgo frames** — when Go calls into C via cgo, the C portion of the stack uses standard C frame conventions and is unwound via DWARF in the C library's debug info (if available). Delve can show C frames in the stack but usually cannot inspect their locals because the C code is rarely compiled with full debug info matching Delve's expectations.

## Stack Traces

Stack traces additionally use:

- **`runtime.stkbar`** — historically, "stack barriers" were a runtime mechanism for incremental stack scanning during garbage collection. They are no longer used in the latest Go versions (replaced by precise stack scanning), but Delve's older code paths handle them for compatibility with binaries built by older Go versions.
- **Symbol resolution** — to render an instruction pointer as `package.Function`, Delve looks up the symbol in the binary's symbol table and DWARF info. Go binaries include both the standard symbol table (`pclntab` — the program counter to line number table, which also maps PC ranges to function names) and the DWARF `.debug_info`.
- **`runtime.modulesinfo`** — when Go's `plugin` package is used, additional code is loaded after process start. The runtime maintains a list of loaded modules (`runtime.firstmoduledata` and the `next` chain), each with its own pclntab and symbol info. Delve walks this chain to resolve symbols across plugins.
- **`pclntab`** — the program counter to line number table is a Go-specific structure embedded in every Go binary. It maps from PC to (file, line) and to function name. Delve uses this for fast symbol lookup, falling back to DWARF for richer info (parameter names, locals).

## DWARF for Go

DWARF (Debugging With Attributed Record Format) is the standard debug-info format used by GCC, Clang, and the Go compiler. The Go compiler emits DWARF in several sections of the executable:

- **`.debug_info`** — the bulk of the debug info: a tree of "Debugging Information Entries" (DIEs) describing compilation units, functions (`DW_TAG_subprogram` with `DW_AT_low_pc`, `DW_AT_high_pc`, parameters as children), types (`DW_TAG_base_type`, `DW_TAG_structure_type`, `DW_TAG_array_type`, etc.), and variables (`DW_TAG_variable`, `DW_TAG_formal_parameter`, with `DW_AT_location`).
- **`.debug_abbrev`** — a compaction table for `.debug_info`. Each DIE references an abbreviation that lists which attributes it carries. This is a large space saving.
- **`.debug_line`** — the line number program. A bytecode that, when interpreted, produces a table mapping PC to (file, line, column, is_stmt). Delve uses this for "what line is the program at" and for setting breakpoints by line.
- **`.debug_str`** — a string pool that DIE attributes reference by offset.
- **`.debug_loc`** — location lists for variables whose location changes across PC ranges (e.g., a local that lives in a register early in the function and is spilled to the stack later). Delve evaluates these to find a variable at the current PC.
- **`.debug_ranges`** — non-contiguous PC ranges for entities that cover multiple regions.
- **`.debug_frame`** — call frame information for unwinding.
- **`.debug_pubnames` / `.debug_pubtypes`** — accelerated lookup tables (somewhat deprecated; Delve mostly builds its own indexes).

Go has historically deviated slightly from standard DWARF in a few places (the encoding of certain types, the use of non-standard attributes), and Delve's `pkg/dwarf` package contains Go-specific extensions to handle them.

### Split DWARF (`.debug_loc.dwo`)

Some toolchains support "split DWARF" where a binary contains a small skeleton and the bulk of the debug info lives in a separate `.dwo` file. Go does not commonly use this, but Delve's parser can handle it for compatibility with binaries built through cgo with a clang toolchain.

## Go-Specific Type Info

Beyond DWARF, the Go runtime carries its own type metadata at runtime — used by reflection, by interface dispatch, and by the garbage collector. This metadata is essential for Delve to correctly render certain values.

### `runtime._type`

Every type in a Go program has a corresponding `runtime._type` (or `internal/abi.Type` in modern Go) struct in the binary's data section. It contains:

- the type's size,
- the type's alignment,
- a kind tag (struct, slice, map, chan, interface, etc.),
- a pointer to the type's name string,
- pointers to additional metadata depending on kind (e.g., for a struct, a pointer to a `structtype` with the field list).

Delve can in principle read this metadata, but in practice it primarily uses DWARF, falling back to runtime metadata only when DWARF is missing or insufficient.

### `runtime.iface` and `runtime.itab`

A Go interface value is two words: `(itab, data)` for non-empty interfaces, or `(*_type, data)` for `interface{}`. The `itab` is a struct mapping the interface type to the concrete type:

```go
type itab struct {
    inter *interfacetype  // the interface type
    _type *_type          // the concrete type
    hash  uint32
    fun   [1]uintptr      // the method table (variable length)
}
```

When Delve is asked to print an interface value, it:

1. Reads the two words.
2. If the static type is `interface{}` (an empty interface), the first word is the `*_type`; reads it and consults the type to determine how to render the second word.
3. If the static type is a non-empty interface, the first word is the `*itab`; reads `itab._type` to find the concrete type, then renders the second word accordingly.
4. The "data" word is either the value itself (if it fits in a word) or a pointer to the value (if it does not). Delve uses the type's size to decide.

This is how `print x` where `x` is `interface{}` containing a `*MyStruct` produces `*main.MyStruct {Field: 42}` instead of `0x12345678`.

## eval.go's Variable Resolution

Local variable resolution is the hottest path in the evaluator and is worth tracing in detail.

When a user types `print x`:

1. The evaluator parses `x` as an `*ast.Ident`.
2. It needs to find the storage location of `x` in the current frame.
3. It consults the DWARF for the current function, looking for a `DW_TAG_variable` or `DW_TAG_formal_parameter` with `DW_AT_name = "x"`.
4. The DIE has a `DW_AT_location` attribute, which is a DWARF expression. Common forms:
    - `DW_OP_fbreg N` — frame-base-register plus offset N. The frame base is determined by another DWARF expression, typically `DW_OP_call_frame_cfa` (the call-frame address from `.debug_frame`).
    - `DW_OP_addr A` — absolute address A (used for globals).
    - `DW_OP_reg N` — value lives in register N.
    - `DW_OP_breg N M` — register N plus offset M (used for parameters passed in registers, then spilled).
    - A list of `DW_OP_*` opcodes forming a small stack-machine program.
5. Delve evaluates the location expression to compute the variable's address (or notes that it lives in a register).
6. It reads `type-size` bytes from that address (or from the register set).
7. It uses the variable's type DIE to interpret the bytes — converting raw bytes into a struct with named fields, a slice with header `(data, len, cap)`, a map header pointing to a `runtime.hmap`, and so on.

For pointer-following, the evaluator dereferences and recursively resolves the target. There are guards against pointer cycles (each address is tracked in a visited set during recursion) and against runaway recursion (configurable with `LoadConfig.MaxDepth`).

## Map Iteration

A Go map (`map[K]V`) is implemented as a `runtime.hmap`:

```go
type hmap struct {
    count     int
    flags     uint8
    B         uint8       // log2(num buckets)
    noverflow uint16
    hash0     uint32
    buckets    unsafe.Pointer  // 2^B buckets
    oldbuckets unsafe.Pointer  // for incremental resize
    nevacuate  uintptr
    extra      *mapextra
}
```

Each bucket holds 8 (`bucketCnt`) key-value pairs plus a small header with hashes ("tophash" bytes) and a pointer to an overflow bucket if the 8 slots are full.

When Delve iterates a map (for `print myMap` to render its contents), it:

1. Reads the `hmap` struct.
2. Computes `numBuckets = 1 << hmap.B`.
3. For each bucket from 0 to `numBuckets - 1`:
    a. Reads the bucket header (8 tophash bytes, then 8 keys, then 8 values).
    b. For each slot, if `tophash != emptyRest && tophash != emptyOne`, the slot is occupied; reads the key and value.
    c. If the bucket has an overflow pointer, follows it and reads the overflow bucket the same way.
4. Also iterates `hmap.oldbuckets` (if non-nil) for resizing maps mid-iteration.

This mirrors what `runtime.mapiterinit` and `runtime.mapiternext` do at runtime, but Delve does it without calling into the target. The reason for not calling into the target is that the target may be in any state, including mid-write to the map, and a runtime call could deadlock or crash. Walking the structure directly is read-only and safe.

The cost is that Delve must track Go's map layout precisely. The `hmap` layout has changed between Go versions (the order of fields, the addition of `extra`, the addition of swissmap in Go 1.24+ which uses a completely different table structure). Delve has version-specific code for each.

## Channel Inspection

A Go channel (`chan T`) is implemented as a `runtime.hchan`:

```go
type hchan struct {
    qcount   uint    // number of items in queue
    dataqsiz uint    // queue size (0 for unbuffered)
    buf      unsafe.Pointer // ring buffer
    elemsize uint16
    closed   uint32
    elemtype *_type
    sendx    uint    // send index
    recvx    uint    // recv index
    recvq    waitq   // list of waiting receivers
    sendq    waitq   // list of waiting senders
    lock     mutex
}
```

Each `waitq` is a linked list of `sudog` structs, each representing one parked goroutine waiting on the channel. The `sudog` carries a pointer to its `g`, so Delve can render the queue as "goroutines 7, 12, 18 waiting to send."

When Delve renders a channel, it:

1. Reads `hchan`.
2. Reports `qcount/dataqsiz` as "buffered: N/M."
3. If `qcount > 0`, walks the `buf` ring buffer from `recvx` to `sendx` (modulo `dataqsiz`) to render the queued elements.
4. Walks `sendq` and `recvq` to list waiting goroutines.

This gives a complete picture of channel state, useful for diagnosing deadlocks ("which goroutines are stuck on this channel?").

## Reflection

For an interface value where the concrete type is determined only at runtime, the evaluator reflects:

1. Read the interface's two words.
2. Determine whether the static type is `interface{}` (empty) or a method-bearing interface.
3. Read the type word's `itab` (for non-empty) or `*_type` (for empty).
4. From the `*_type`, get the concrete type's name and kind.
5. Render the data word using the concrete type's layout.

This is how `print someInterface` reveals "this is actually a `*main.UserSession{ID:42, Name:"alice"}`."

The same machinery handles `error` values: an `error` is just an interface, and most concrete error types are pointer-to-struct, so Delve unwraps to show the struct fields.

## Optimization Mode

Go's compiler is aggressive about optimization, and the optimized output often eliminates variables (constant-folded, kept only in registers, or computed fresh each time). The DWARF info reflects this: a variable with `DW_AT_location` of `<no entry>` or covering only part of the function's PC range means the variable does not have a unique address everywhere.

When Delve cannot find a location for a variable at the current PC, it reports `(value optimized away)`. To avoid this:

```bash
go build -gcflags="all=-N -l"
```

`-N` disables optimization (the compiler emits straight-line, unoptimized code). `-l` disables inlining (every function call is a real call, not an inline expansion). The `all=` prefix applies the flags to the whole build, including dependencies. Together, every variable retains a stable location across the function's lifetime, and DWARF describes that location accurately.

For `dlv debug` the gcflags are applied automatically. For pre-built binaries you must rebuild with these flags or accept the optimized-away message for many variables.

The downside is that unoptimized binaries are slower (no constant folding, no dead-code elimination, no register allocation) and larger. They are not a problem for development debugging but should not be deployed.

## PIE / ASLR

Position-Independent Executables (PIE) and Address Space Layout Randomization (ASLR) randomize the base address at which the binary's code is mapped. A breakpoint set at the binary-relative address `0x1000` must be translated to `base + 0x1000` at runtime, where `base` is the actual mapping address.

Delve handles this by:

1. Reading `/proc/PID/maps` (Linux) or the equivalent Mach API (macOS) or `EnumProcessModules`/`GetModuleInformation` (Windows) to find the binary's load address.
2. Computing the slide: `slide = load_address - link_address`.
3. Adjusting all symbol addresses, breakpoint addresses, and DWARF address ranges by the slide.

This is invisible to the user — `break main.go:42` works the same regardless of PIE — but is essential for breakpoints to land at the right place. Failure mode: a non-PIE-aware debugger with a PIE binary sets a breakpoint at the link-time address and never fires it because that address contains random heap data, not code.

Modern Go on most platforms produces PIE binaries by default, and Delve has been PIE-aware for years.

## Build IDs & Source Path Mapping

Each Go binary has an embedded build ID (a hash) used to verify that the source matches the binary. Delve uses this to detect mismatches: if you rebuild the binary while a debugger is attached, Delve warns that the source has changed.

For source path mapping, the binary's DWARF records absolute paths to source files. When debugging a binary built on machine A but run on machine B with sources at a different path, Delve's `--substitute-paths` flag (or the `SourcePath` setting in DAP) maps build-time paths to runtime paths:

```bash
dlv debug --substitute-path /build/src=/home/me/src ./mybinary
```

This is essential for remote debugging (build in CI, debug locally) and for containerized debugging (binary built inside a container, sources on host).

## Cgo Debugging

Cgo crosses the boundary between Go and C, and that boundary is hard to debug:

- The Go side uses Go's split stacks, calling convention, and runtime.
- The C side uses standard C stacks and conventions.
- Crossing requires synchronization with the Go runtime: the cgo runtime sets up a special M (a "G0" stack) for the C call so that the C code does not corrupt the goroutine stack.

Delve's support for cgo is limited:

- It can show C frames in stack traces if the C code was compiled with `-g` (debug info).
- It usually cannot inspect C local variables because the DWARF in the C code uses different conventions (no Go-style frame pointer, different register usage).
- Setting breakpoints in C code works if you can name the C symbol or file/line.
- Stepping from Go into C usually works (the runtime handles the transition), but stepping back from C into Go can be unreliable.

The runtime's cgo synchronization is handled by `runtime/cgo.cgocall`, which switches to the system stack, calls the C code, and switches back. Delve recognizes these frames and tries to skip them in the user-visible trace.

For serious mixed-language debugging, the typical workflow is:

- Use Delve for the Go code.
- Attach GDB or LLDB to the same process for the C code.
- Or use `dlv exec --backend=lldb` to use Apple's debugserver as the underlying debug stub, which has slightly better C support.

## Plugin Debugging

Go's `plugin` package allows loading shared libraries at runtime. Each loaded plugin adds new symbols, new types, and new code regions to the process. Delve handles plugins by:

1. When the process starts, Delve indexes the main binary's symbols and DWARF.
2. The runtime maintains a list of loaded modules (`runtime.modulesinfo`, a chain starting at `runtime.firstmoduledata`).
3. When a plugin is loaded, the runtime adds an entry to the module list with the plugin's pclntab, symbol table, and (if the plugin was built with `-gcflags="all=-N -l"`) DWARF.
4. Delve detects new modules either by re-reading the module list periodically or by setting a breakpoint on `runtime.pluginftabverify` (which the runtime calls after loading).
5. After detecting a new module, Delve re-indexes its symbols and adds its DWARF to the global type table.

The gotcha is that breakpoints set by symbol name in plugin code do not work until the plugin is loaded. The standard pattern is to set a breakpoint on the plugin's load function, run until it hits, and then set further breakpoints in the plugin code.

## Test Debugging

`dlv test` is a convenience wrapper:

1. It runs `go test -c -o /tmp/dlv_test_<pkg> ...` to compile a test binary without running it.
2. It then runs `dlv exec /tmp/dlv_test_<pkg>` with whatever flags the user provided.
3. The test binary is run with `-test.run=<pattern>` etc. to select tests.

`--build-flags` propagates flags to `go test -c`, so:

```bash
dlv test --build-flags="-tags=integration"
```

builds the test binary with the `integration` tag.

This is preferred to `dlv debug` for testing because the test framework's main is implicit and `dlv debug` would run main, not the tests.

## Headless Mode + Multi-Client

Headless mode is what makes IDE integration possible. Without it, `dlv` runs an interactive REPL. With `--headless`, it serves a network API and waits for clients.

The standard incantation:

```bash
dlv --headless --listen=:2345 --api-version=2 --accept-multiclient debug ./cmd/server
```

- `--headless` — no REPL.
- `--listen=:2345` — listen on TCP port 2345 (or `unix:///path/to/socket`).
- `--api-version=2` — use the v2 RPC API (default in modern Delve, but explicit is better).
- `--accept-multiclient` — survive client disconnects.

The IDE-attaches-now-detach-now flow is:

1. CI spawns dlv with `--headless` and waits for the listen port to be ready.
2. The developer's IDE attaches, sets breakpoints, debugs.
3. The developer detaches, fixes a bug, redeploys, and re-attaches.
4. The dlv process stays alive throughout because of `--accept-multiclient`.

Without `--accept-multiclient`, step 3 would kill dlv when the IDE detaches and step 4 would need a new dlv instance.

## dlv core

A core dump is a snapshot of a process at the time of a crash. Linux produces core dumps when a process dies from a fatal signal (SIGSEGV, SIGABRT, etc.) and `ulimit -c` is non-zero. Go programs can also be made to dump core on panic by setting:

```bash
GOTRACEBACK=crash
```

which causes the runtime to call `abort()` (raising SIGABRT) at the end of fatal-error handling, generating a core dump.

`dlv core /path/to/binary /path/to/core` opens the core dump for inspection:

```bash
dlv core ./mybinary /var/crash/core.12345
```

Inside, you can inspect goroutines, stacks, variables, and memory, just as if the process were live and stopped — except you cannot continue, step, or call functions. The core dump is a frozen snapshot.

The `gocore` project (github.com/golang/debug/cmd/gocore) is a related tool that does deep heap analysis of core dumps, including reachability from roots, type-aware traversal, and per-type byte counts. It uses some of Delve's parsing code under the hood.

## dlv trace

`dlv trace` is a printf-style trace of function entry and exit. It works by setting tracepoints on every function matching a regex:

```bash
dlv trace ./myprog 'main.process.*'
```

Each tracepoint logs `>>> entered main.processOrder(args...)` on entry and `<<< returning from main.processOrder` on exit, then auto-resumes. The output streams to stdout while the program runs.

This is useful for "what's happening?" exploration without a full debugger session, especially in combination with capture flags like `--stack=5` (capture 5 frames of stack trace at each tracepoint).

The implementation uses tracepoints (described above) which are essentially auto-resuming breakpoints with capture configuration.

## dlv attach

`dlv attach <PID>` attaches to a running process by PID. The mechanics are:

1. Issue `PTRACE_ATTACH` (Linux) / `task_for_pid` (macOS) / `DebugActiveProcess` (Windows).
2. Wait for the kernel to stop the process.
3. Read `/proc/PID/exe` (or equivalent) to find the binary path.
4. Open the binary and load DWARF.
5. Take inventory: list threads, walk `runtime.allgs`, etc.

Security restrictions:

- **Linux** — `/proc/sys/kernel/yama/ptrace_scope` controls who can attach. Value `0` allows any tracing, `1` (default on most distributions) restricts to descendants only, `2` requires `CAP_SYS_PTRACE`, `3` blocks all attachment. Delve gives a clear error on permission denied.
- **macOS** — requires either root (which is increasingly hard to get usefully on macOS), or signing with the `com.apple.security.cs.debugger` entitlement and `com.apple.security.cs.disable-library-validation`. The Delve binary distributed via Homebrew is signed with these entitlements; a self-built Delve will need to be signed manually.
- **SIP** (System Integrity Protection on macOS) — even with entitlements, you cannot debug system binaries (anything from `/System` or `/usr/bin` etc.). Disabling SIP requires booting into recovery mode.
- **Windows** — requires `SeDebugPrivilege`, which administrators have by default.

## dlv connect

`dlv connect <addr>` connects to a running headless dlv. This is the client side of the headless mode pattern:

```bash
# Terminal A:
dlv --headless --listen=:2345 --accept-multiclient debug ./myprog

# Terminal B:
dlv connect :2345
```

Terminal B gets the interactive REPL but driving the dlv in Terminal A. Multiple `dlv connect` clients can attach concurrently if `--accept-multiclient` is set.

This is also how to debug a program running in a remote container, VM, or another machine: run dlv headless inside the container, expose the port, and connect from outside.

## dlv replay

`dlv replay <rr-trace-dir>` runs Delve on top of Mozilla's `rr` (record-and-replay) debugger. The workflow:

1. Record the program's execution: `rr record ./myprog`.
2. The recording captures every syscall, every signal, and enough state to deterministically replay.
3. Run `dlv replay /path/to/recording-dir`.
4. Delve attaches to a replayed instance via the gdbserver-RSP backend; rr serves the protocol.

The advantage of replay debugging is determinism: you can step backwards (rr supports reverse execution), set a breakpoint earlier in time, run to a specific event by sequence number, and so on. Bug investigations that involve "by the time I notice the corruption, the cause has been overwritten" become tractable because you can reverse-step from the symptom to the cause.

The disadvantage is that rr currently only works on Linux x86_64 (with hardware performance counter support — most modern Intel CPUs work, AMD support is improving), and recording adds 1.5x to 5x runtime overhead.

## Performance Characteristics

Delve's startup time is dominated by DWARF parsing. A medium-sized Go binary (10 MB executable, lots of dependencies) has tens to hundreds of megabytes of DWARF, and Delve must parse and index it before the user can do anything. This typically takes 0.5 to 5 seconds.

Optimizations include:

- **Lazy loading** — only the index is built eagerly; full DIE trees for functions are loaded on demand when the user steps into them.
- **Symbol table caching** — `pclntab` is parsed and cached once.
- **Index-on-demand** — type information is indexed by name lazily.

Memory inspection (`print x` for a deep object) scales with the object size and the recursion depth (`MaxDepth` in `LoadConfig`, default 1 for top-level command, more for explicit `-depth`).

The expression evaluator's worst case is map iteration: a map with millions of entries takes tens of seconds to fully render. The `MaxArrayValues` and `MaxMapBuckets` limits cap this; default is around 64 visible entries.

## Memory Inspection

Delve can dump arbitrary memory via:

- **`examineMemory`** (terminal command) — dump bytes at an address.
- **`RPCServer.ListAllMemory`** (RPC, occasional) — used by some tools.
- **`RPCServer.ExamineMemory`** (RPC) — read N bytes from address A.

The API takes an address and a length, returns the bytes. The terminal renders them in a hex dump similar to `xxd`. This is useful for low-level debugging where you have a pointer and want to see what is there before adding type information.

## Internal Hardening

Delve is designed assuming the user is in control of both the debugger and the target. There is no security boundary between them — anyone who can run dlv on a process can read its memory and inject code. That said, Delve has a few hardening features:

- **`mode=runtime-restricted`** — a configuration flag that disables function calls in expressions (so `print obj.UnsafeMethod()` is rejected). Useful in CI where you do not want a debug session to mutate state.
- **The `GoDebug` environment variable** — can be set to control runtime debug knobs (e.g., `GoDebug=schedtrace=1000` for scheduler tracing) which Delve will honor when launching the target.
- **No remote-code-execution by default** — the JSON-RPC API does not expose primitives for arbitrary memory writes or arbitrary code execution; only well-typed operations are exposed. (Of course, "set variable to value" with a pointer type is very nearly arbitrary memory write.)

These are not security boundaries — they are guardrails to prevent accidents.

## Idioms

A few patterns recur in well-run Delve workflows:

- **Build with `-gcflags='all=-N -l'` for debuggable binaries.** This is the single most common cause of "value optimized away" or "function inlined" frustration. For development builds, just add `-gcflags='all=-N -l'` to your build commands.
- **Use `--headless` for editor integration.** The interactive REPL is fine for command-line debugging, but IDEs need the network API. Always run dlv with `--headless` from within IDEs.
- **Use `trace` for printf-debugging at runtime.** Adding `fmt.Println` statements requires recompiling and changes the binary. `dlv trace` adds tracepoints without recompilation and removes them when you exit.
- **DAP, not JSON-RPC2, for VS Code.** VS Code's official Go extension uses the DAP backend (via the `dlv-dap` binary). If you see odd VS Code behavior, ensure the extension is using DAP mode.
- **Use `dlv test` for tests.** `dlv debug` runs the package's main; for tests you need `dlv test`.
- **Use `dlv core` for post-mortem.** When a Go program panics in production, configure `GOTRACEBACK=crash` and `ulimit -c unlimited`, capture the core, then debug at leisure with `dlv core`.
- **Use `--accept-multiclient` for long-lived debugging sessions.** Without it, an accidental disconnect kills the session.
- **Use `--substitute-path` for cross-machine debugging.** When the binary was built elsewhere, source paths will not match; substitute.
- **Set conditional breakpoints sparingly in hot loops.** Each evaluation costs microseconds; in a tight loop that fires millions of times per second, you can slow the program by orders of magnitude. Tracepoints with capture are a better fit.
- **Use `goroutines -running` to find the goroutines actually doing work.** In a program with thousands of parked goroutines, the ones that are running are usually what you want.
- **Use `bp` (with `--stack=N`) on errors to capture context without stopping.** A breakpoint on `errors.New` or your custom error constructor with stack capture builds an ad hoc trace of where errors originate.

## See Also

- gdb (sheets/system) — the C-model debugger Delve was built to replace for Go.
- lldb (sheets/system) — Apple's debugger; Delve uses lldb-server as its lower layer on macOS.
- pdb (sheets/system) — Python's debugger; analogous in spirit (language-aware debugger using the runtime's introspection).
- go (sheets/languages) — the language Delve targets.

## References

- github.com/go-delve/delve — the canonical source repository.
- github.com/go-delve/delve/tree/master/Documentation — user documentation including command reference and architectural notes.
- github.com/go-delve/delve/tree/master/Documentation/internal — internal design documents covering the proc layer, DWARF handling, and Go-specific decisions.
- The JSON-RPC2 service spec lives in `service/rpc2/server.go` (godoc-extractable).
- Microsoft's Debug Adapter Protocol specification at microsoft.github.io/debug-adapter-protocol/.
- JetBrains GoLand documentation on debugger configuration at jetbrains.com/help/go/.
- The VS Code Go extension repository at github.com/golang/vscode-go for the DAP-mode launch configuration reference.
- Mozilla rr documentation at rr-project.org for replay debugging.
- The DWARF Debugging Information Format Standard at dwarfstd.org for the DWARF reference.
- Linux ptrace(2) and process_vm_readv(2) man pages for the Linux process control primitives.
- Apple's "Mach Concepts" technical note for the macOS debugging API model.
- Microsoft's "Debugging Functions" documentation at learn.microsoft.com for the Win32 debug API.
- The Go runtime source at github.com/golang/go/tree/master/src/runtime, especially `runtime2.go` (struct definitions for `g`, `m`, `p`, `hmap`, `hchan`, `sched`), `proc.go` (scheduler), and `map.go`/`map_swiss.go` (map implementation) for the structures Delve walks.
- The Go pclntab format documented in `runtime/symtab.go` and the `debug/gosym` package.
