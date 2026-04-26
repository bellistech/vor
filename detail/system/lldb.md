# LLDB Internals — Deep Dive

A mechanism-and-algorithm focused tour of LLDB's internals: how the debugger is layered on LLVM, how it talks to processes, how it parses debug information, how it places breakpoints, how it unwinds stacks, and why each design choice was made. The companion sheet at `sheets/system/lldb.md` covers the user-facing HOW. This page covers the WHY.

## Setup

LLDB is the debugger of the LLVM project. It lives in the `lldb/` subtree of the `llvm-project` monorepo and shares its CMake build, its compiler infrastructure, its testing harness, and crucially its libraries (libLLVM for disassembly, libClang for the C/C++/Objective-C type system, libSwift for Swift type system on Apple platforms). LLDB does not link statically against the system DWARF parser, the system C++ demangler, or the system disassembler — it brings its own. This is intentional: a debugger that depends on the very compiler infrastructure being debugged risks cross-tool drift; LLDB closes that loop by living in the same tree.

The relationship to Clang is the deepest. LLDB does not implement a parser for C, C++, or Objective-C. When you type `expr -- (int)foo->bar + 3` at an LLDB prompt, what happens is: LLDB asks Clang to parse the expression as if it were a tiny translation unit, asks Clang to type-check it against an AST (Abstract Syntax Tree) reconstructed from DWARF debug info, asks Clang to lower it to LLVM IR, asks the LLVM JIT (Just-In-Time compiler — currently MCJIT or ORCv2) to materialize machine code, and then injects that code into the inferior process and runs it on a stolen thread. The expression evaluator is a full mini-compiler. This is why LLDB ships with a Clang AST as its primary type system rather than a hand-rolled type model — the moment you want to evaluate `foo[i].member` against a real C++ object with templates and inheritance, you need a real C++ frontend, and LLDB just borrows the one already next door in the source tree.

The Swift toolchain integration is the other deep linkage. On Apple platforms, LLDB embeds a complete Swift compiler in its address space (the same way it embeds Clang). The Swift type system is layered as a `TypeSystemSwift` plugin parallel to `TypeSystemClang`, and a Swift module's binary `.swiftmodule` files (essentially a serialized Swift AST) are deserialized at debug time so the debugger can resolve generic types, protocol conformances, and metadata of objects whose static type is `Any` or `AnyObject`. This is why the LLDB shipped with Xcode is much larger than upstream LLDB — it carries the entire Swift compiler. The standalone open-source LLDB does not include Swift support unless built from the swift-lldb fork at `github.com/apple/llvm-project`, branch `swift/release/X.Y`, which is the canonical Swift+LLDB combined branch.

On macOS, the runtime hookups are subtle. LLDB interacts with the dynamic linker via the `_dyld_debugger_notification` interface — a callback registered with dyld through the `DYLD_INTERPOSE` machinery. When the inferior loads a new dylib, dyld calls a synchronous notification function which LLDB has set as its breakpoint trampoline; LLDB stops the process, walks the dyld image-info array (anchored by `_dyld_all_image_infos` exported from `/usr/lib/dyld`), discovers the new module, parses its Mach-O load commands, locates its UUID, finds the matching dSYM bundle, and resumes. The same channel is used for unloads and (on iOS) for shared cache notifications. CrashReporter sits parallel: when a process faults, the kernel sends a Mach exception to the host's exception port; if no debugger is attached, the bootstrap server routes the exception to ReportCrash which generates an `.ips` file in `~/Library/Logs/DiagnosticReports/`. If LLDB is attached, it claims the exception port via `task_set_exception_ports` first, and the kernel never reaches ReportCrash.

How `lldb.framework` loads inside Xcode: Xcode does not embed LLDB as a static library; instead, Xcode's `DebuggerLLDB.ideplugin` (an Xcode plugin bundle) `dlopen()`s `/Applications/Xcode.app/Contents/SharedFrameworks/LLDB.framework/LLDB`. The framework exposes the public SBAPI (Script Bridge API) C++ symbols. Xcode talks to LLDB only through SBAPI — never through internal C++ classes. This is the contract that lets LLDB ship internal refactors every release without breaking the IDE.

## Architecture

LLDB is structured in three concentric rings. The innermost ring is the private internal C++: classes named without the `SB` prefix (e.g., `Process`, `Thread`, `StackFrame`, `Target`, `ValueObject`, `SymbolContext`, `Module`, `CompileUnit`, `Variable`, `Function`, `Block`). These classes are not stable API; they change every release; they live in `lldb/source/` and are linked into `liblldbCore.a`, `liblldbSymbol.a`, `liblldbBreakpoint.a`, etc. They are virtual-dispatch-heavy, ref-counted via `std::shared_ptr`, and freely thread-cooperative.

The middle ring is the SBAPI — the Script Bridge API. Every internal class has a thin facade in `lldb/source/API/` named with an `SB` prefix: `SBProcess`, `SBThread`, `SBFrame`, `SBValue`, `SBTarget`, `SBBreakpoint`, `SBModule`, `SBCompileUnit`, `SBSymbolContext`, etc. Each `SBxxx` is a value type holding only an opaque `std::shared_ptr` (or `std::weak_ptr`) to the corresponding internal object. The SBAPI is intentionally minimal — methods like `SBProcess::GetSelectedThread()`, `SBThread::GetFrameAtIndex(idx)`, `SBValue::GetChildAtIndex(idx)`. Critically, the SBAPI is the only ABI-stable surface in LLDB. It is what Xcode, Visual Studio Code, the Python interpreter, and the `lldb` command-line driver itself all consume.

The outermost ring is the various drivers. The `lldb` binary in `bin/` is a small command-line driver that links against `liblldb` (the SBAPI shared library) and embeds a Python interpreter or Lua interpreter for scripting. Xcode is another driver. The Python module `lldb` (importable with `import lldb` in any CPython process) is yet another driver — same SBAPI, exposed via SWIG-generated bindings.

The hierarchy of objects observed by a user:

```
Debugger
 └─ Target* (one per executable being debugged; often just one)
     ├─ Module* (one per loaded library + the main exec)
     │   ├─ ObjectFile (Mach-O / ELF / COFF parser)
     │   ├─ SymbolFile (DWARF / CodeView / Symtab parser)
     │   └─ Symtab (the unified symbol table)
     ├─ Breakpoint* (a logical breakpoint)
     │   └─ BreakpointLocation* (a concrete address; one BP can match many)
     ├─ Watchpoint*
     └─ Process (zero or one — created once you `run` or `attach`)
         ├─ Thread*
         │   ├─ StackFrame*
         │   │   └─ Variable* (locals, params, statics)
         │   ├─ ThreadPlan* (the action the thread is currently executing)
         │   └─ RegisterContext (the per-thread registers)
         ├─ DynamicLoader (the runtime image loader plugin)
         ├─ JITLoader (per-runtime JIT debug interface)
         └─ OperatingSystem (kernel-mode plugin for kernel debugging)
```

This hierarchy is reflected one-to-one in SBAPI. `SBDebugger.GetSelectedTarget().GetProcess().GetSelectedThread().GetSelectedFrame().FindVariable("argv")` is a typical chain, and each call is a single `std::shared_ptr` deref.

The `Target/Breakpoint/Watchpoint` registry is owned by a `Target`. A `Target` maps 1:1 to a binary on disk — when you `lldb a.out`, you create one `Target`. A target persists across `run` cycles: when the process exits, the breakpoints, watchpoints, and source-mapping state are retained, and the next `run` reattaches them. This is why typing `b main; r; ... exit; r` re-hits `main` without re-typing `b main`.

A `Breakpoint` is a logical specification ("stop when execution reaches `foo.cpp:42`" or "stop on entry to function `MyClass::do_thing`"). A breakpoint is materialized into one or more `BreakpointLocation` instances, each of which is a real address in the inferior's memory. One logical breakpoint can spawn many locations because of inlining (one source line lives at multiple machine addresses), templates (one source function instantiated for many types), and shared libraries (the same function in two dylibs). When a location's image is not yet loaded, the breakpoint is "pending" — it waits in the breakpoint registry, and `DynamicLoader` notifies it on each load event so it can scan the new module for matches.

## The Plugin Manager

Almost every concrete behavior in LLDB is implemented as a plugin. Plugins are not dynamically loaded `.so`/`.dylib` files (almost always); they are statically linked into `liblldb` and registered at startup via static initializers (the `LLDB_PLUGIN_DEFINE` macro). The `PluginManager` maintains a registry of plugin instances per category, and each plugin exposes a static `CreateInstance` method that the manager invokes when the right conditions are met.

The plugin categories, roughly in the order LLDB exercises them on a typical session:

**Process** — the engine that controls a running inferior. Two main plugins:
- `process.gdb-remote` — the canonical out-of-process plugin. Communicates with `lldb-server` (or `debugserver` on macOS, or `gdbserver` for foreign hosts) over a TCP/serial/UNIX-socket connection using the GDB remote serial protocol. Used for remote debugging, iOS device debugging, and even local debugging on Linux through `lldb-server gdbserver`.
- `ProcessNative` (per OS) — `ProcessFreeBSD`, `ProcessLinux`, `ProcessMacOSXKernel` etc. — drives the inferior in-process via the host's native debugging API (`ptrace` on Linux/BSD, Mach exceptions on Darwin). On Linux, `ProcessLinux` is itself layered over `NativeProcessLinux` which speaks `ptrace` directly.
- Specialized: `ProcessElfCore`, `ProcessMachCore`, `ProcessMinidump` for postmortem core/minidump files; `ProcessKDP` for macOS kernel debugging; `ProcessWindows` for Windows.

**Disassembler** — the `disassembler.llvm-mc` plugin wraps LLVM's `MC` (Machine Code) library, which is the same disassembler used by `llvm-objdump`. There is no in-house LLDB disassembler. Architecture-specific behavior (e.g. Thumb/ARM mode, Power VLE, MIPS branch-delay slots) is provided by the LLVM target.

**SymbolFile** — the parser for debug-info formats:
- `symbol-file.dwarf` — DWARF v2/3/4/5, including the split-DWARF variant where a `.dwo` or `.dwp` file holds the bulky `.debug_info` while the executable holds only a skeleton CU.
- `symbol-file.pdb` — PDB (Program Database, Microsoft's format), used on Windows.
- `symbol-file.native-pdb` — a native (non-DIA) PDB parser.
- `symbol-file.symtab` — fallback when only a symbol table is available (no DWARF). Yields functions but no line numbers, no types, no locals.
- `symbol-file.breakpad` — Breakpad symbol files (used by Mozilla, Chromium for crash collection).
- `symbol-file.json` — JSON-encoded symbol description, used for some embedded targets.

**TypeSystem** — language-aware type model:
- `TypeSystemClang` — Clang AST-based, covers C, C++, Objective-C, Objective-C++.
- `TypeSystemSwift` (Apple-tree only) — Swift AST + remote-mirror runtime introspection.
- `TypeSystemRust` — provided by the Rust LLDB fork.
- `TypeSystemPDB` — Microsoft PDB-derived types.

**ABI** — the calling convention used to read return values, set up function calls (for `expr` evaluation), and unwind:
- `ABISysV_x86_64` — System V AMD64 (Linux, BSD, macOS prior to ARM64).
- `ABIMacOSX_arm64` / `ABISysV_arm64` — AArch64 ABIs (Apple variant differs in pointer authentication and prologue conventions).
- `ABIWindows_x86_64` — Microsoft x64.
- `ABISysV_i386`, `ABISysV_arm`, `ABISysV_mips`, `ABISysV_ppc`, `ABISysV_riscv`, `ABISysV_s390x`, `ABISysV_hexagon` — many more.

**DynamicLoader** — the per-OS runtime loader plugin that watches dyld/ld.so for image-load events:
- `dynamic-loader.posix-dyld` — Linux/BSD glibc `_r_debug` / `r_debug_extended` interface.
- `dynamic-loader.macosx-dyld` — Apple `dyld_all_image_infos` and `dyld_process_info` (in modern macOS).
- `dynamic-loader.hexagon-dyld` — Qualcomm Hexagon DSP loader.
- `dynamic-loader.windows-dyld` — PE loader.
- `dynamic-loader.darwin-kernel` — XNU kernel images.
- `dynamic-loader.static` — for fully-static (no shared libs) executables.

**JITLoader** — watches for in-process JIT compilers registering code:
- `JITLoaderGDB` — the GDB JIT interface (the `__jit_debug_register_code` / `__jit_debug_descriptor` symbols and the breakpoint at `__jit_debug_register_code`).

**ScriptInterpreter** — embeds a scripting language inside LLDB:
- `ScriptInterpreterPython` — CPython 3 embedded; the `lldb` Python module exposing SBAPI.
- `ScriptInterpreterLua` — Lua embedded; lighter-weight alternative.
- `ScriptInterpreterNone` — no scripting (used in builds with no Python).

**OperatingSystem** — only loaded when debugging an OS kernel; provides synthetic threads from kernel thread structures (used by macOS kernel debugging and embedded RTOS support).

**InstrumentationRuntime** — the bridge to compiler-runtime sanitizers:
- `InstrumentationRuntimeASan` — AddressSanitizer.
- `InstrumentationRuntimeTSan` — ThreadSanitizer.
- `InstrumentationRuntimeUBSan` — UndefinedBehaviorSanitizer.
- `InstrumentationRuntimeMainThreadChecker` — Apple main-thread-only API checker.

**LanguageRuntime** — runtime-introspection plugins per language:
- `ObjCLanguageRuntime` (and v1/v2 subvariants) — Objective-C runtime metadata access.
- `CPlusPlusLanguageRuntime` — C++ runtime helpers (RTTI, exception unwinding).
- `SwiftLanguageRuntime` (Apple) — Swift runtime metadata, generic type resolution.
- `RenderScriptRuntime` — Android RenderScript (largely deprecated).

**StructuredData** — extracts structured event streams from the runtime (e.g. ASAN reports, DTrace event streams).

**SystemInitializer** / **SystemLifetimeManager** — orchestrate plugin discovery on `Debugger::Initialize()`.

The Plugin Manager is queried with `PluginManager::GetXxxPluginCreateCallbackForPluginName(name)` or `PluginManager::GetXxxPluginCreateCallbackAtIndex(i)`. Most plugin discovery is "ask each plugin: do you handle this?" — for example, when a `Module` is being parsed, each `ObjectFile` plugin is asked `CanHandle(this_data)` until one says yes. This is how the same LLDB binary handles ELF, Mach-O, COFF, and Minidump containers.

## SBAPI / Python Scripting

SBAPI is exposed to Python via SWIG (Simplified Wrapper and Interface Generator). The interface file `lldb/bindings/interface/SBxxx.i` describes how each C++ method maps to a Python method, with hints for argument conversion. SWIG generates a large `.cpp` shim (`LLDBWrapPython.cpp`) containing thunks that unbox `PyObject*`s into C++ values, call the C++ method, and rebox the result. The generated module is installed as `lldb/__init__.py` plus a native extension `_lldb.so`.

Importantly, the `lldb` Python module can be imported **outside** the interactive `lldb` driver. Any CPython process can `import lldb`, instantiate `lldb.SBDebugger.Create()`, and have full debugger functionality. This is how IDE integrations, automated debugging scripts, and kernel post-mortem tools work.

The SBAPI is intentionally a pull-style API: nothing is automatic. `SBDebugger::Create()` returns a debugger; you must call `CreateTarget()`, then `target.Launch()` or `target.AttachToProcess()`. Once the process is running, you query state by polling — `process.GetState()` returns one of `eStateStopped`, `eStateRunning`, `eStateExited`, etc. To wait for events, you use `SBListener`.

```python
import lldb

debugger = lldb.SBDebugger.Create()
debugger.SetAsync(False)              # synchronous mode: commands block
target = debugger.CreateTarget("a.out")
bp = target.BreakpointCreateByName("main")
process = target.LaunchSimple(None, None, ".")
# process is already stopped at main
thread = process.GetSelectedThread()
frame = thread.GetSelectedFrame()
for var in frame.GetVariables(True, True, True, True):
    print(var.GetName(), "=", var.GetValue())
process.Continue()
```

The synchronous-vs-async distinction matters. By default `SBDebugger` is asynchronous: `process.Continue()` returns immediately, the process runs, and stop events arrive on a background thread. To consume those events, you create an `SBListener`, register it on the process broadcaster (`process.GetBroadcaster().AddListener(listener, eBroadcastBitStateChanged)`), and pull events with `listener.WaitForEvent(timeout, event)`. Each event has a type (e.g. `SBProcess::eBroadcastBitStateChanged`) and a payload (typically the new state).

In synchronous mode, `Continue()` blocks until the next stop, and most scripts can ignore the listener machinery. Synchronous mode is `debugger.SetAsync(False)` and is what Python scripts almost always want.

The SBAPI thread-safety contract is: API calls are serialized internally via a per-Target mutex. You can call from any thread, but two threads calling SBAPI simultaneously will block one of them. Long-running operations (e.g. continuing a process) release the mutex while the process runs and re-acquire it on stop.

The `SBxxx` objects are themselves cheap: they hold only a `shared_ptr` and have no virtual functions. Copying an `SBFrame` is essentially copying a pointer plus a refcount bump. They are designed to be copied freely, used as Python locals, and discarded.

## Process Plugins

LLDB has two flavors of process control: the **gdb-remote** plugin and the **native** plugin. Both ultimately do `ptrace`/`mach`-style operations, but at very different layers.

The **native** plugin, on Linux, is `NativeProcessLinux`. It runs in the same address space as LLDB itself. It uses `ptrace(PTRACE_TRACEME)` (in the child) and `ptrace(PTRACE_ATTACH)` (in the parent) to gain debugging control, then uses `waitpid(WNOHANG)` and `ptrace(PTRACE_CONT)` to drive execution. For each thread, a Linux task ID is tracked. Reading/writing memory uses `process_vm_readv`/`process_vm_writev` (the modern Linux interface, which avoids the per-page expense of `PTRACE_PEEKDATA`/`POKEDATA`) when available, falling back to `/proc/PID/mem` and finally to ptrace for ancient kernels.

On macOS, the native plugin uses Mach exception ports rather than ptrace. `task_for_pid()` returns a Mach port representing the inferior's task, `mach_vm_read_overwrite()` reads memory, `thread_get_state()` reads registers. Crucially, `task_for_pid()` on modern macOS requires either root, the `com.apple.system-task-ports` entitlement, or the `get-task-allow` entitlement on the target. This is why you cannot debug system binaries on macOS without disabling SIP.

The **gdb-remote** plugin runs the process control logic out-of-process. The actual ptrace/Mach calls happen inside `lldb-server` (open-source) or `debugserver` (Apple-shipped, slightly different protocol extensions). LLDB itself talks to `lldb-server` over a socket using the GDB Remote Serial Protocol — the same wire format originally designed for GDB to talk to embedded-board ROM monitors over RS-232.

The key insight: on Linux, `lldb` typically launches `lldb-server gdbserver` as a subprocess on `localhost:randomport` and talks to it. This means even local Linux debugging goes through the gdb-remote protocol; the `native` plugin path is rarely taken on Linux. On macOS, by contrast, LLDB always uses gdb-remote (via `debugserver`) — there is no in-process Mach-exception-handling path, because Mach exception ports do not multiplex well with the LLDB main thread.

The **gdb-remote protocol** is a packet-based serial protocol. Each packet has the form `$<payload>#<checksum>` where `<checksum>` is a two-hex-digit modular sum of the payload bytes. The receiver acknowledges with `+` (ack) or `-` (retransmit). After the initial handshake, both sides typically request "no-ack mode" with `QStartNoAckMode` to skip the per-packet acknowledgments — this saves a round-trip per packet on slow links.

A typical session opens with:

```
GDB:    $qSupported:multiprocess+;swbreak+;hwbreak+;...
Server: $PacketSize=8000;qXfer:features:read+;QStartNoAckMode+;...
GDB:    $QStartNoAckMode
Server: $OK
GDB:    $vMustReplyEmpty
Server: $              (empty reply)
GDB:    $qXfer:features:read:target.xml:0,fff
Server: $l<?xml version="1.0"?>...<target>...</target>
```

Key packets:
- `qSupported` / `qXfer:features:read:target.xml` — feature negotiation; the target XML lists registers, register groups, and architecture details.
- `QStartNoAckMode` — turn off acknowledgments.
- `vMustReplyEmpty` — sent to verify the server reply behavior on unknown packets (must reply empty rather than `$E01`).
- `qfThreadInfo` / `qsThreadInfo` — first/subsequent thread info; returns a list of TIDs.
- `H` — set the "current thread" for subsequent operations (e.g. `Hg1234` sets thread 0x1234 for register-read operations).
- `g` / `G` — read/write all general registers.
- `p` / `P` — read/write a single register by index.
- `m<addr>,<len>` / `M<addr>,<len>:<data>` — read/write memory.
- `Z0,<addr>,<kind>` / `z0,<addr>,<kind>` — set/clear software breakpoint.
- `Z1` — hardware breakpoint. `Z2`/`Z3`/`Z4` — write/read/access watchpoint.
- `vCont;c` / `vCont;s` / `vCont;C09` — continue, step, continue-with-signal.
- `vCont;c:<thread>` — per-thread resume actions.
- `?` — query why the process stopped (returns a stop reply).
- `T05thread:1234;name:main;reason:breakpoint;` — a stop reply: signal 5 (SIGTRAP), thread 0x1234, named main, hit a breakpoint.
- `qXfer:libraries-svr4:read::0,fff` — read the loaded shared-library list (Linux); macOS uses `qShlibInfoAddr` and dyld walking instead.
- `qSymbol::` — server requests symbol resolution for runtime symbols (rarely used, but central for OS kernel debugging).

LLDB's gdb-remote implementation extends the protocol with Apple-specific packets: `jThreadsInfo` (returns JSON-serialized per-thread info), `QListThreadsInStopReply`, `qHostInfo` (returns target architecture, OS, vendor), `QEnableErrorStrings`. These are why `debugserver`'s wire format is not 100% compatible with vanilla `gdbserver`.

## Mach Tasks vs Linux Tasks

The fundamental difference: Linux uses `ptrace(2)`, a syscall-based per-thread model where the tracer interposes on the tracee at every signal/syscall boundary. Darwin uses Mach IPC, where the tracer captures the inferior's task port and receives exception messages on a Mach port.

On Linux:
- `ptrace(PTRACE_ATTACH, pid)` claims the process; subsequent `waitpid(pid)` returns when it stops.
- Each thread is a separate `tid` (kernel TID). All tids of a process share `/proc/PID/`.
- `/proc/PID/maps` lists memory mappings (incomparably more detailed than what Mach exposes).
- `/proc/PID/mem` is a seekable file backing the inferior's memory; readable/writable subject to `ptrace_scope` and capabilities.
- Stop notifications come via `waitpid(WNOHANG)` returning a status word; `WSTOPSIG(status)` reveals which signal.
- Resuming: `ptrace(PTRACE_CONT, tid, 0, sig)` — `sig` is delivered to the tracee on resume, or 0 to suppress.

On macOS:
- `task_for_pid(mach_task_self(), pid, &task_port)` claims the task. This requires entitlements/SIP exemptions.
- Each Mach thread has a `thread_port` obtained via `task_threads(task_port, &threads, &n)`.
- Memory access: `mach_vm_read_overwrite()`, `mach_vm_write()`, `mach_vm_protect()`, `mach_vm_region()` (the closest thing to `/proc/PID/maps`).
- Stop notifications come via Mach exception messages on a port LLDB has registered with `task_set_exception_ports(task, mask, port, behavior, flavor)`. The behavior is `EXCEPTION_DEFAULT | MACH_EXCEPTION_CODES`, the flavor is `THREAD_STATE_NONE` (so the exception message contains no register state — LLDB pulls registers explicitly).
- Resuming: `task_resume(task)` — but this resumes ALL threads; per-thread resume requires manipulating thread suspend counts with `thread_suspend()` / `thread_resume()`.

I/O redirection differs too. On Linux, LLDB inherits the inferior's stdin/stdout/stderr by default — running `program` in `lldb` shows program output inline. To redirect, LLDB replaces stdin/stdout/stderr file descriptors before `execve()`. On macOS, `debugserver` (or LLDB's launch path) sets up a `pty` and the inferior's stdin/stdout point at the slave side; LLDB reads/writes the master side. This is why `process launch -i /dev/tty -o /dev/tty -e /dev/tty` sometimes appears — you're requesting the pty replacement.

Signal handling: Linux delivers signals via the ptrace-stop mechanism; LLDB sees the signal, can replay it on resume, suppress it, or trigger user actions. Mach has no signals at the Mach layer — Unix signals are delivered by the BSD subsystem of XNU after Mach exception delivery. LLDB on macOS sees Mach exceptions first; only if it ignores them do they get translated to BSD signals.

## DWARF Symbol Resolution

DWARF (Debugging With Attributed Record Formats) is a format and not a single section. It is split across multiple ELF/Mach-O sections, each carrying a different aspect of the debug info:

- **`.debug_info`** — the trees of Debugging Information Entries (DIEs). One subtree per Compile Unit (CU). Holds types, functions, variables, namespaces. The largest section by far.
- **`.debug_abbrev`** — the abbreviation tables. To save space, every CU defines a small dictionary of "abbrevs" — each abbrev is a recipe (tag + list of (attribute, form)). DIEs in `.debug_info` reference an abbrev by number, then encode just the attribute values, drastically compacting the representation.
- **`.debug_line`** — the line-number program. Each CU has a small VM program that, when executed, emits a table of `(address, file, line, column, is_stmt, ...)` rows. The program is encoded as a stream of opcodes: `DW_LNS_set_file`, `DW_LNS_advance_pc`, `DW_LNS_advance_line`, `DW_LNS_copy`, plus extended opcodes like `DW_LNE_set_address`, `DW_LNE_end_sequence`.
- **`.debug_aranges`** — address ranges per CU. For each CU, a list of `(start, length)` pairs. Lets a debugger find which CU contains a given PC without parsing all CUs. Optional in DWARF v5; replaced by `.debug_rnglists`.
- **`.debug_pubnames`** / **`.debug_pubtypes`** — global name indexes. For each CU, a list of `(offset_within_cu, name)` pairs for globally-visible names (and types). Optional — Clang stopped emitting them by default in favor of `.debug_names` (the DWARF v5 accelerator), but many older binaries still have them.
- **`.debug_names`** — the DWARF v5 name accelerator. A hash-trie of all names (functions, types, namespaces) with offsets to their DIEs. Replaces `.debug_pubnames`/`pubtypes` and Apple's older `.apple_names`/`.apple_types`/`.apple_namespaces`/`.apple_objc`.
- **`.debug_loc`** / **`.debug_loclists`** — location lists. A variable's location can change as a function executes (in-register early, on-stack later, optimized out at the end). A loclist is a sequence of `(start_pc, end_pc, expression)` ranges, where the expression is a tiny stack-based VM (DW_OP_reg, DW_OP_breg, DW_OP_fbreg, DW_OP_lit, DW_OP_plus, ...).
- **`.debug_ranges`** / **`.debug_rnglists`** — range lists, used when a function or scope is non-contiguous (common with hot/cold splitting).
- **`.debug_frame`** — call frame information for unwinding. (See "Stack Unwinding".)
- **`.debug_str`** / **`.debug_line_str`** — string tables, deduplicated.
- **`.debug_macro`** / **`.debug_macinfo`** — preprocessor macro definitions.
- **`.debug_types`** (DWARF v4) — type units, deduplicated across CUs by signature.
- **`.debug_addr`** — address pool (DWARF v5).
- **`.debug_str_offsets`** — string offset table (DWARF v5).
- **`.debug_loclists`** / **`.debug_rnglists`** — DWARF v5 location/range lists with header.

A DIE (Debugging Information Entry) is the fundamental DWARF object. Each DIE has:
- A **tag** identifying its kind: `DW_TAG_compile_unit`, `DW_TAG_subprogram`, `DW_TAG_formal_parameter`, `DW_TAG_variable`, `DW_TAG_structure_type`, `DW_TAG_pointer_type`, `DW_TAG_inlined_subroutine`, `DW_TAG_lexical_block`, `DW_TAG_namespace`, `DW_TAG_class_type`, `DW_TAG_member`, `DW_TAG_array_type`, `DW_TAG_subrange_type`, `DW_TAG_enumeration_type`, `DW_TAG_enumerator`, `DW_TAG_typedef`, `DW_TAG_const_type`, `DW_TAG_volatile_type`, `DW_TAG_reference_type`, etc.
- A list of **attributes**: `DW_AT_name`, `DW_AT_low_pc`, `DW_AT_high_pc`, `DW_AT_type`, `DW_AT_byte_size`, `DW_AT_decl_file`, `DW_AT_decl_line`, `DW_AT_location`, `DW_AT_data_member_location`, `DW_AT_specification` (links a definition to a declaration), `DW_AT_abstract_origin` (links an inlined instance to its abstract subprogram).
- A list of **children** DIEs (forms a tree).

A typical function looks like:

```
DW_TAG_subprogram
  DW_AT_name ("foo")
  DW_AT_low_pc (0x1000)
  DW_AT_high_pc (0x10c0)
  DW_AT_type (->DIE for int)
  DW_AT_decl_file (1)
  DW_AT_decl_line (42)
  DW_AT_frame_base (DW_OP_reg6)             // %rbp
  DW_TAG_formal_parameter
    DW_AT_name ("x")
    DW_AT_type (->DIE for int)
    DW_AT_location (DW_OP_fbreg -4)          // [%rbp-4]
  DW_TAG_variable
    DW_AT_name ("y")
    DW_AT_type (->DIE for int)
    DW_AT_location (DW_OP_fbreg -8)          // [%rbp-8]
```

LLDB's DWARF parser is in `lldb/source/Plugins/SymbolFile/DWARF/`. It does **lazy** parsing: when a CU is needed, it's parsed; when a DIE is needed, it's parsed; types and functions are parsed on demand. This is the reason an LLDB session on a large binary can launch quickly and feel slow only when you ask for a symbol that's deep in an unparsed CU.

The **abbrev compaction** scheme is critical for size. Every distinct (tag, [(attribute, form)]) pattern in a CU gets one abbrev table entry (a number, the tag, then the list of attribute/form pairs). DIEs in `.debug_info` then reference the abbrev by ULEB128-encoded number and immediately follow with just the attribute values. A binary with 100,000 functions might have 30 abbrevs — every `DW_TAG_subprogram` with the same attribute shape uses one abbrev. Without abbrevs, DWARF would be vastly larger.

When LLDB resolves "set a breakpoint on foo()", the algorithm is:
1. Look up "foo" in `.debug_names` (or `.apple_names` or `.debug_pubnames` if older).
2. Get the offset(s) into `.debug_info`.
3. Parse the DIE at each offset (and its children) to get `DW_AT_low_pc`.
4. Translate the `low_pc` through the module's load-address bias into a process address.
5. Place a software breakpoint at that address (or multiple addresses for templates/inlined sites).

When LLDB resolves "what line is PC 0x10ab in?":
1. Find which CU contains 0x10ab via `.debug_aranges` (or by walking CUs if absent).
2. Run the line-number program in `.debug_line` for that CU until the row's `address` exceeds 0x10ab.
3. The previous row's (`file`, `line`, `column`) is the answer.

## dSYM Bundles

On macOS, the linker by default does **not** put DWARF in the final Mach-O binary. Instead, the linker leaves DWARF in the `.o` files, and a separate post-link step extracts it via `dsymutil` into a sidecar bundle named `MyApp.dSYM`. The bundle has the structure:

```
MyApp.dSYM/
└── Contents/
    ├── Info.plist
    └── Resources/
        └── DWARF/
            └── MyApp           ← Mach-O file containing only DWARF sections
```

The reason for this: linking on macOS used to be slow when the linker had to relocate DWARF references, and Apple's solution is to defer that work to `dsymutil`, which links DWARF separately with full knowledge of the final layout. The original `.o` files have DWARF references that point at sections, and `dsymutil` reads those, applies relocations, deduplicates types across CUs, and emits a single consolidated DWARF Mach-O.

The matching between an executable and its dSYM is by **UUID**. Every Mach-O has a `LC_UUID` load command containing a 16-byte UUID computed at link time as a hash of the binary contents. The dSYM's inner Mach-O has the *same* UUID. When LLDB loads a module, it reads the UUID from the executable, then searches for a dSYM with that UUID. The search paths:

1. The same directory as the executable: `MyApp.dSYM` next to `MyApp`.
2. The Spotlight index: `mdfind "com_apple_xcode_dsym_uuids == AAAA-...-FFFF"`.
3. `~/Library/Developer/Xcode/iOS DeviceSupport/<version>/Symbols/` for iOS device debugging.
4. `DEBUGINFOD_URLS` (newer LLDB) for HTTP-based debug-info servers.

If you build a binary with `-g`, then strip it and discard the dSYM, you cannot debug it — you'll see "(missing UUID 1234ABCD)" messages.

`dsymutil` also performs **type uniquing**. Because Clang emits a copy of every used type per CU, the same `std::vector<int>` DIE might appear in 1,000 CUs in a large project. `dsymutil` recognizes structural identity and emits one canonical DIE, replacing the others with forward references. Without this step, dSYMs would be 5–10× larger.

`atos(1)` is the macOS symbolicator that reads a dSYM and converts addresses to symbols+file+line. It uses the same DWARF parsing as LLDB but in a one-shot non-interactive form. Crash reports on macOS contain raw addresses + UUIDs, and `atos -arch arm64 -o MyApp.app/Contents/MacOS/MyApp -l 0x100000000 0x10001234` resolves a single address.

## Compile Units & Subprograms

A **Compile Unit (CU)** in DWARF terminology corresponds to a single translation unit — typically one `.c`, `.cpp`, `.m`, or `.swift` source file plus all its `#include`-d headers. Each CU produces:
- A top-level `DW_TAG_compile_unit` DIE.
- A line-number program in `.debug_line`.
- An entry in `.debug_aranges` (if emitted).

The CU DIE's attributes summarize the unit:
- `DW_AT_producer` — the compiler version string ("clang version 17.0.0").
- `DW_AT_language` — `DW_LANG_C99`, `DW_LANG_C_plus_plus_14`, `DW_LANG_ObjC`, `DW_LANG_Swift`, etc.
- `DW_AT_name` — the source filename ("foo.cpp").
- `DW_AT_comp_dir` — the directory the compiler was invoked from.
- `DW_AT_low_pc` / `DW_AT_high_pc` (or `DW_AT_ranges`) — the address range covered.
- `DW_AT_stmt_list` — offset into `.debug_line` for this CU's line program.

Inside the CU, **subprograms** (`DW_TAG_subprogram`) represent functions. A subprogram with `DW_AT_low_pc` and `DW_AT_high_pc` is a concrete function. A subprogram with no `low_pc` is an "abstract" subprogram — used for inlining: the abstract instance describes the parameter list and types once; each concrete inlined call site (`DW_TAG_inlined_subroutine`) refers back via `DW_AT_abstract_origin`.

The **line-number program** is a virtual machine that emits the line table. Its registers include `address` (current PC), `file` (current source file index), `line` (current line), `column` (current column), `is_stmt` (whether this is a recommended breakpoint location), `basic_block`, `end_sequence`. The opcodes:

- `DW_LNS_copy` — emit a row from current registers.
- `DW_LNS_advance_pc` — increment `address` by an LEB128 amount.
- `DW_LNS_advance_line` — increment `line` by a signed LEB128 amount.
- `DW_LNS_set_file` — set `file` to a new index.
- `DW_LNS_set_column` — set `column`.
- `DW_LNS_negate_stmt` — toggle `is_stmt`.
- `DW_LNS_set_basic_block` — mark next row as basic block start.
- `DW_LNS_const_add_pc` — fast-path PC advance.
- `DW_LNS_fixed_advance_pc` — uhalf-byte PC advance (used for line tables in non-standard alignment).
- `DW_LNS_set_prologue_end` / `DW_LNS_set_epilogue_begin` (DWARF 3+) — mark prologue/epilogue boundaries; lets debuggers skip prologue when stepping in.
- `DW_LNE_set_address` — extended opcode; set `address` to a fixed value (used after a discontiguous jump).
- `DW_LNE_end_sequence` — emit a row marking end-of-sequence; reset state.
- `DW_LNE_define_file` — define an additional file mid-sequence (rare).

There's also a **special opcode** range: opcodes from `opcode_base` (typically 13) up to 255 are encoded compactly to do `advance_pc(N) + advance_line(M) + copy()` in one byte. The factoring is `opcode = (line - line_base) + (opcode_base * minimum_instruction_length) + (advance_pc * line_range)`, decoded by the inverse formula. This is the densest encoding in DWARF and is why line tables can describe millions of source lines in a few KB.

Decoding the line program produces a sorted table of `(address, file, line, column, ...)`. To find "what line is PC X?", binary-search for X. To find "what addresses correspond to line Y in file F?", scan for matches.

For breakpoint resolution by source location, LLDB needs to find the address that corresponds to the *first* statement of a line (preferring rows with `is_stmt = true` and `prologue_end = true`). The "first column 0 is_stmt row at this line" is the standard target — it skips function prologue and lands on the first user statement.

## Type System & Clang

LLDB's type representation is **not** a hand-rolled struct. It is a Clang AST. The class `TypeSystemClang` (in `lldb/source/Plugins/TypeSystem/Clang/`) wraps a full `clang::ASTContext` plus the supporting `clang::SourceManager`, `clang::FileManager`, `clang::IdentifierTable`. Every C/C++/Objective-C type LLDB knows about is a `clang::QualType` referencing a `clang::Type` in this context.

The mapping from DWARF DIE to Clang type happens in `DWARFASTParserClang`. For each DIE, the parser:
1. Examines the tag (`DW_TAG_structure_type`, `DW_TAG_pointer_type`, `DW_TAG_class_type`, ...).
2. Reads the relevant attributes (`DW_AT_name`, `DW_AT_byte_size`, `DW_AT_type`).
3. Constructs a `clang::Type` via `ASTContext::getRecordType()`, `getPointerType()`, `getCXXRecordDecl()`, etc.
4. For records (structs/classes/unions), recursively parses children (`DW_TAG_member`) and inserts them into the `clang::CXXRecordDecl`'s `addDecl()` list.
5. Sets the size and alignment to match the DWARF info via the `clang::ExternalASTSource` extension hook.

The advantage is that LLDB now has full Clang type semantics: name lookup obeys C++ scoping rules, template instantiation works, member access checks are real, ADL works. The `expr` command can evaluate any expression Clang can parse.

The disadvantage is bloat. A Clang `ASTContext` for a non-trivial C++ project has memory overhead in the hundreds of MB. LLDB mitigates this with lazy population: the `ExternalASTSource` only populates members of a record when Clang asks for them (e.g. when the user types `frame.member`, that's the moment Clang asks "what are the members of this `record_type`?", and LLDB consults DWARF and adds them).

`TypeSystemSwift` is a separate plugin doing the same trick with Swift's compiler infrastructure. Swift's type model is much richer (generics, protocols, extensions, value vs reference semantics), so the Swift type system imports the binary `.swiftmodule` files (serialized Swift ASTs) directly and mirrors them.

The "**synthetic types**" pattern is when LLDB needs to interpret a region of inferior memory as a structure that doesn't have a DWARF type. Examples: kernel stacks, JITed objects, raw protocol buffers. The `TypeSystemClang::CreateRecordType()` API lets you build a type from scratch in C++, give it members, byte sizes, and offsets, and the rest of LLDB will display values of that type as if they were real C structs. This is heavily used by Python pretty-printers for opaque runtime objects.

## Breakpoint Resolution

When you type `b foo.cpp:42` or `b MyClass::do_thing`, LLDB creates a `Breakpoint` with a **resolver** — a small object that knows how to translate the user's request into a list of addresses.

Resolver types:
- **`BreakpointResolverFileLine`** — match a source file and line. Walks every CU's line table searching for rows with the matching file (using basename or full-path match per setting) and line. Each match produces a location.
- **`BreakpointResolverAddress`** — exact address. Trivial — one location.
- **`BreakpointResolverName`** — match a function name. Uses the symbol table's accelerator (`.debug_names`, `.apple_names`, hash table) to find all DIEs with a matching `DW_AT_name`. Templates and overloads produce multiple locations.
- **`BreakpointResolverFileRegex`** — regex match against source line content (e.g., `b -p "TODO"` to break on every line containing "TODO").
- **`BreakpointResolverScripted`** — Python-coded resolver; the user implements `__callback__(target, name, locations)`.

A `Breakpoint` is created with one resolver and a **search filter** (which modules/CUs/functions to consider). The filter can be `Unconstrained` (everything), `ByModule` (only specific dylibs), `ByCompileUnit`, or scripted.

When a `Module` is loaded (image-load event from the dynamic loader), LLDB iterates all existing breakpoints and asks each resolver "do you match anything in this new module?". This is how breakpoints set on functions in not-yet-loaded shared libraries become **pending** and "fire" when the library loads.

Each match produces a `BreakpointLocation`. A location has:
- An address.
- A "shadow" — the original instruction byte(s) at that address (for software breakpoints).
- A reference back to the parent `Breakpoint`.
- An enabled/disabled state.
- An optional condition (an expression to evaluate; if false, silently resume).
- An optional ignore count (skip the first N hits).
- An optional Python callback (called on each hit).

The "**one breakpoint, many locations**" model is what makes `b foo` feel right when there are 50 inlined instances of `foo`. The user sees one numbered breakpoint (`#1`); internally there are 50 locations (`#1.1`, `#1.2`, ...). Disabling `#1` disables all 50; setting a condition on `#1` applies to all 50. This is also why LLDB occasionally reports "Breakpoint 1 (50 locations)".

## Hardware vs Software Breakpoints

A **software breakpoint** is implemented by overwriting the instruction at the target address with a "trap" instruction — `INT3` (`0xCC`) on x86, `BRK #imm` (`0xD4200000` or similar) on ARM64, `TRAP` on PowerPC, `BREAK` on MIPS. The original instruction byte(s) are saved in the `BreakpointLocation`'s shadow. When the inferior executes the trap, the kernel raises a debug trap exception (`SIGTRAP` on Linux/BSD, `EXC_BREAKPOINT` on Mach), which the debugger catches.

To resume past a software breakpoint, the debugger must:
1. Restore the original instruction(s) at the breakpoint address.
2. Set the PC back to the breakpoint address (the trap incremented it).
3. Single-step (one instruction).
4. Re-write the trap byte(s).
5. Resume normally.

This dance is "**breakpoint stepping**" and is invisible to the user. On variable-length-instruction CISC architectures (x86), an additional hazard is the boundary problem: if a multi-byte instruction crosses an instruction-cache line, you must replace exactly the right number of bytes. INT3 (`0xCC`) is 1 byte and on x86 is documented to work as a 1-byte breakpoint regardless of where the surrounding instruction starts.

ARM64 BRK is 4 bytes, fixed-width, aligned. Easy.

ARM (32-bit, A32 mode) is a hybrid: A32 instructions are 4 bytes, T32 (Thumb) instructions are 2 or 4 bytes (depending on first halfword). The breakpoint instruction differs (`BKPT` for A32, different encoding for T32). LLDB knows which mode the PC is in via the low bit of the address (T32 addresses have bit 0 set in the symbol table, even though the actual address is even).

A **hardware breakpoint** uses CPU debug registers to compare the PC against a watched address on every instruction fetch. The CPU traps if there's a match — no instruction patching, the original code is untouched.

x86: 4 hardware breakpoint slots in `DR0`-`DR3`. `DR7` is the control register specifying which slots are enabled, what conditions (execute, read, write, read/write), and what lengths (1/2/4/8 bytes). `DR6` is the status register reporting which slot fired. Hardware breakpoints are useful for **read-only memory** (where INT3 patching would fault), for **JITed code** that's been mprotected as PROT_READ|PROT_EXEC, and for **watchpoints** (which are essentially hardware breakpoints with read/write conditions).

ARM64: 6–16 hardware breakpoint slots in `DBGBVR0`-`DBGBVR15` paired with `DBGBCR0`-`DBGBCR15` (control registers). The exact count is implementation-defined; modern Apple Silicon has 6 BRPs (breakpoint registers) and 4 WRPs (watchpoint registers). Watchpoints use `DBGWVR0`-`DBGWVR15` paired with `DBGWCR0`-`DBGWCR15`. The control register specifies enable, privilege level, byte address select, length.

The 4-on-x86 (or 6-on-ARM64) hardware-breakpoint limit is hard. If a user requests a 5th hardware breakpoint or watchpoint, LLDB fails the request — there is no "fall back to software for the 5th". (Software watchpoints exist in some debuggers via single-stepping every instruction and checking the address, but LLDB does not do this; it would slow execution by 100,000×.)

## Watchpoints

A watchpoint stops the inferior when memory at a given address is read or written. There is no "software watchpoint" in LLDB — all watchpoints are hardware-backed and therefore subject to the per-CPU slot limit.

Configuring a watchpoint:
- Pick a slot (0-3 on x86, 0-3 on most ARM64).
- Write the watch address to the data-address register (`DR0-DR3` on x86, `DBGWVR_n` on ARM64).
- Configure the control register: enable, type (read=R/W=W/access=A), length (1/2/4/8 bytes).
- Resume the inferior.

When the inferior accesses the watched address, the CPU raises a debug exception. On Linux, this is `SIGTRAP` with `si_code = TRAP_HWBKPT`. On macOS, this is `EXC_BREAKPOINT` with code `EXC_ARM_DA_DEBUG` (data abort, debug). The debugger reads the status register (`DR6` on x86, `ESR_EL1` on ARM64) to determine which slot fired and what kind of access occurred.

**Length restrictions** matter. x86 hardware watchpoints allow lengths of 1, 2, 4, or 8 bytes (8 only on x86_64), and the address must be aligned to the length. To watch a 16-byte struct, you'd have to use 2 slots (8+8) or watch only part of it. ARM64 watchpoints support 1-, 2-, 4-, 8-byte lengths with byte-address-select for partial-word watches.

**Read-only watchpoints** are the most expensive in terms of triggering — every load is potentially a hit, and "every load" includes the CPU's own implicit reads (e.g., reading the return address from the stack on `ret`). This is why read watchpoints often trigger spuriously and immediately resume; LLDB must check that the access actually involved the watched bytes.

Mach exception delivery: when a watchpoint fires on macOS, the kernel sends `EXC_BREAKPOINT` to the task's exception port. LLDB's exception handler thread receives the message, identifies which watchpoint fired (by reading the debug registers), checks the optional condition, and decides whether to stop or resume.

ptrace delivery: on Linux, the watchpoint hit is reported via `waitpid()` returning with `WSTOPSIG == SIGTRAP`. The debugger reads `siginfo` via `ptrace(PTRACE_GETSIGINFO)` to get `si_code = TRAP_HWBKPT` and the watchpoint-specific data.

## Stepping Algorithms

Stepping is conceptually simple — "go until the user-visible state has advanced one unit" — but the implementation is layered for correctness.

**Single-instruction step (`stepi`)** is the primitive. It uses the CPU's single-step hardware:
- x86: set the trap flag (TF) bit in EFLAGS; the CPU traps after every instruction.
- ARM64: use the kernel's `PTRACE_SINGLESTEP` (Linux) or `thread_set_state` with the single-step bit (macOS). On bare ARM64 hardware, software single-stepping uses the MDSCR_EL1 single-step bit.
- Some architectures lack single-step entirely and require the debugger to disassemble the next instruction, place a breakpoint there, resume, and clean up.

**Step-in by line (`step`)**: enter the next source statement, descending into called functions.
1. Determine the current line from the line table (use the PC).
2. Set up a "step range plan" covering the address range of the current line.
3. While the PC is within the range and a step has not been "interrupted" by a function call or branch out, single-step.
4. If the PC leaves the range by entering a function (CALL on x86, BL on ARM), and that function has debug info, transfer to it (the user steps "in").
5. If the function has no debug info, fall through with step-out behavior (set a temporary breakpoint at the return address and continue).

**Step-over by line (`next`)**: stay in the current frame.
1. Same range setup as step-in.
2. On every CALL/BL within the range, set a breakpoint at the return address and continue (so the inferior runs the called function to completion at full speed); when the breakpoint fires, you're back at the next instruction in the current frame.
3. When the range is exited normally (the PC moves to a different line in the same frame), stop.
4. If the frame returns (the PC matches the return address of the original frame), stop one frame up.

**Step-out (`finish`)**: run until the current frame returns.
1. Read the current frame's return address (from `.eh_frame`/`.debug_frame` or a frame-pointer chain).
2. Set a temporary breakpoint at that address.
3. Continue.
4. When the breakpoint fires, stop and remove the breakpoint.
5. The return-value register (`%rax`/`%xmm0` on x86_64, `x0`/`v0` on ARM64) is captured and presented as the function's return value.

The line table is consulted constantly during stepping. After every single-step, LLDB looks up the new PC in the line table; if the line has not changed, single-step again. This loop is fast because the line table is sorted and binary-searchable.

The key optimization is **skipping prologues**. When a step-in lands at a function's first instruction, the PC is in the prologue (saving frame pointer, allocating stack). The user wants to be at the first user statement. LLDB consults the line table for `prologue_end = true` rows and jumps to the first such row. If there is no `prologue_end` info (older DWARF), LLDB uses heuristics or the debug info's listed first non-prologue address.

## Thread Plans

A **ThreadPlan** is the canonical synchronization mechanism in LLDB. Conceptually, every action on a thread (continue, step, run-to-address, run-until-return, evaluate-expression-on-thread) is represented as a thread plan, and thread plans stack: each plan has a "child" plan it pushed before yielding control.

The thread plan stack:

```
+--- Top of stack (currently in control)
|  ThreadPlanStepRange    (step over a line)
|  ThreadPlanCallFunction (called expression)
|  ThreadPlanRunToAddress (top-of-stack from before the stepover)
+-- Base
```

Each plan implements:
- `ShouldStop(event)` — given a stop event, should the thread stop or continue (popping or pushing plans as needed)?
- `ExplainsStop(event)` — does this plan account for the stop, or should we ask plans below?
- `WillStop()` — last chance to update state before stopping.
- `MischiefManaged()` — is this plan complete?
- `ShouldRunBeforePublicStop()` — should this plan be re-run inside a hidden continuation?

Concrete plans:
- `ThreadPlanStepInstruction` — single-step one instruction.
- `ThreadPlanStepInRange` — step in within an address range.
- `ThreadPlanStepOverRange` — step over within an address range.
- `ThreadPlanStepOut` — step out of current frame.
- `ThreadPlanStepThrough` — step through trampolines/PLT stubs/dyld stubs to the real function.
- `ThreadPlanStepUntil` — step until reaching one of several addresses.
- `ThreadPlanRunToAddress` — set a temporary breakpoint and continue.
- `ThreadPlanCallFunction` — push a fake stack frame, set up arguments per ABI, set return-trap address, and continue. Used for expression evaluation.
- `ThreadPlanCallUserExpression` — like CallFunction but for compiled user expressions.

Why plans matter: when a user types `expr foo()`, that's a `ThreadPlanCallUserExpression` pushed on top of whatever plan is currently active. While the expression runs, the user might hit a breakpoint inside `foo()`; the breakpoint fires, the plan stack is consulted, the call-function plan handles the trap by stopping (so the user can debug into the expression), and when the expression finally returns, the call-function plan unwinds back to the original state — restoring registers, popping the synthetic frame.

Plans also synchronize **across threads**. If you "step over" on thread 1, you don't want thread 2 to keep running and hit a breakpoint somewhere unrelated and confuse the step. LLDB's default policy is "**stop everyone on any stop**" via the `ProcessSync` mechanism, but plans can fine-tune this with `ShouldRunOnlyThreadPlans()` to suspend other threads during certain operations.

## Stack Unwinding

Unwinding the stack means going from `(PC, SP, FP, ...)` of the current frame to those of the caller. Three sources of information are consulted, in order of preference:

1. **`.eh_frame`** — the unwinding section emitted for C++ exception handling. Always present in modern binaries even without `-g`. Format: a series of CIE (Common Information Entry) records and FDE (Frame Description Entry) records.
2. **`.debug_frame`** — like `.eh_frame` but emitted only with debug info. Same format, slightly different semantics around personality routines.
3. **Symbol table + frame pointer** — fallback. If the function has a frame pointer (RBP/X29), follow the FP chain. If no FP, use the symbol table to locate the function and ABI-default register save layout.

A **CIE** describes the unwinding rules for a class of functions: code alignment factor, data alignment factor, return-address register, and a small program (CFI instructions) of "default" rules. A **FDE** describes one specific function: its address range and a sequence of CFI instructions that, when executed for a given PC within the range, produce the rules `(register N is at offset X from CFA, register M is in register Y, ...)`.

The CFI instructions form a tiny VM:
- `DW_CFA_advance_loc(delta)` — current location += delta.
- `DW_CFA_def_cfa(reg, offset)` — CFA (Canonical Frame Address, usually old SP) = reg + offset.
- `DW_CFA_def_cfa_register(reg)` — keep offset, change base register.
- `DW_CFA_def_cfa_offset(offset)` — keep base register, change offset.
- `DW_CFA_offset(reg, offset)` — register `reg` is saved at CFA + offset.
- `DW_CFA_register(reg, reg2)` — register `reg` lives in register `reg2`.
- `DW_CFA_restore(reg)` — restore register's rule from CIE default.
- `DW_CFA_remember_state` / `DW_CFA_restore_state` — push/pop the rule table (used in alternate prologues).
- `DW_CFA_nop` — padding.
- `DW_CFA_def_cfa_expression` — CFA is given by a DWARF expression (the full DW_OP VM).

To unwind, you execute the FDE's CFI program from the FDE start to the current PC offset. The result is a table of rules; you evaluate each rule against the current registers to recover the caller's registers. Then you replace `(PC, SP, ...)` with `(caller's PC, caller's SP, ...)` and repeat.

Performance note: parsing CIE/FDE is fast, but caching parsed unwind plans matters. LLDB caches per-function unwind plans in the `UnwindPlanCache` keyed on `(module, function_address)`. The first frame in a function pays the parsing cost; subsequent stack walks through the same function reuse the cached plan.

When `.eh_frame` is absent (rare, but happens with stripped binaries), LLDB falls back to the **symbol table + frame pointer** unwinder. If the function uses a frame pointer (the compiler emitted `push rbp; mov rbp, rsp` prologue), the chain is:

```
caller's RBP     ← [RBP+8]   (return address)
caller's frame   ← [RBP]
```

Walk RBP to walk frames. If no frame pointer (`-fomit-frame-pointer`, common in optimized release builds), this fallback fails and you get a corrupt backtrace. This is why release-build crashes on Linux often have `???` frames — no eh_frame, no FP.

On Apple platforms, an additional source is **compact unwind** in the `__unwind_info` section. This is a binary-encoded, more compact alternative to `.eh_frame` shipped on Mach-O. LLDB's unwinder handles it transparently.

## ASAN / TSAN / MSAN Integration

The compiler-runtime sanitizers (AddressSanitizer, ThreadSanitizer, MemorySanitizer) are libraries linked into the inferior at compile time. When a sanitizer detects an error (use-after-free, double-free, race, uninitialized-read), it normally prints a long report and aborts.

LLDB integrates with sanitizers via the `InstrumentationRuntimeASan` plugin (and friends). The sanitizers expose specific runtime symbols:
- `__asan_describe_address(addr)` — given an address, returns a description ("allocated by ...", "freed by ...", "stack of thread ...").
- `__asan_report_error(pc, bp, sp, addr, is_write, access_size)` — internally called when ASAN detects an error.
- `__asan_get_alloc_stack(addr, ...)` / `__asan_get_free_stack(addr, ...)` — recover the allocation/free call stacks.
- `__tsan_on_report(report)` — called by TSAN for each detected race.

When LLDB attaches and detects ASAN linkage (by checking for the symbol `__asan_init`), it sets a hidden breakpoint at `__asan_report_error`. When the breakpoint fires, LLDB extracts the arguments, calls `__asan_describe_address` via expression evaluation, displays the description, and presents the crash as a structured ASAN report.

Similarly, for TSAN, LLDB hooks `__tsan_on_report` and walks the report data structure.

The sanitizer reports are exposed via SBAPI as `SBProcess::GetExtendedCrashInformation()` returning an `SBStructuredData` tree.

## Python Pretty-Printers

Native LLDB display of a STL container shows the raw struct fields (`_M_start`, `_M_finish`, `_M_end_of_storage` for libstdc++ `std::vector`). Useless. Pretty-printers replace this with a child-list of conceptual elements.

Two pretty-printer mechanisms:
- **Type Summary** — a one-liner string. Format: `"$0.first = ${var.second}"`-style template, or a Python function that returns a string.
- **Type Synthetic Children** (a.k.a. synthetic providers) — a Python class implementing `num_children()`, `get_child_at_index(i)`, `get_child_index(name)`, `update()`. The synthetic children replace the real raw children for display purposes; the user sees `vector[0]`, `vector[1]` instead of `_M_start[0]`, `_M_start[1]`.

A synthetic provider for `std::vector`:

```python
class StdVectorProvider:
    def __init__(self, valobj, internal_dict):
        self.valobj = valobj
    def update(self):
        impl = self.valobj.GetChildMemberWithName('_M_impl')
        self.start = impl.GetChildMemberWithName('_M_start')
        self.finish = impl.GetChildMemberWithName('_M_finish')
        self.elem_type = self.start.GetType().GetPointeeType()
        self.elem_size = self.elem_type.GetByteSize()
        self.size = (self.finish.GetValueAsUnsigned() -
                     self.start.GetValueAsUnsigned()) // self.elem_size
    def num_children(self):
        return self.size
    def get_child_at_index(self, idx):
        offset = idx * self.elem_size
        return self.start.CreateChildAtOffset(f'[{idx}]', offset, self.elem_type)
    def get_child_index(self, name):
        try:
            return int(name.lstrip('[').rstrip(']'))
        except: return -1
```

Registered with `type synthetic add -P StdVectorProvider -x "^std::vector<.+>$"` — the regex matches type names. LLDB ships dozens of these in `examples/synthetic/` and `lldb/source/Plugins/Language/CPlusPlus/`.

## Type Summaries vs Type Synthetics

Summaries and synthetics are independent. A type can have both: synthetics for the children, summary for the one-line display.

Summary string format examples:
- `"size=${var._M_size}"` — direct member.
- `"${var.first} -> ${var.second}"` — pair.
- `"${var.x} + ${var.y}i"` — complex number.
- `"${var%S}"` — apply default summary recursively.
- `"${var.data%@}"` — print as objc object.

The `${...}` interpolation supports format specifiers: `%d` decimal, `%x` hex, `%s` string, `%c` character, `%@` Objective-C description, `%S` default summary, `%@@/N/` array of N. The order of the format string defines the visible left-to-right output.

Python summary functions get more flexibility:

```python
def MyClass_summary(valobj, internal_dict):
    name = valobj.GetChildMemberWithName('name').GetSummary() or "(none)"
    age = valobj.GetChildMemberWithName('age').GetValueAsUnsigned()
    return f'{name} ({age})'
```

Registered with `type summary add -F mymodule.MyClass_summary -x "^MyClass$"`.

The interplay: when `frame variable` displays `v`, LLDB consults the type summary (if any) for the one-line view. If the user expands `v` (e.g. via `--show-all-children` or by clicking in an IDE), LLDB consults the synthetic children for the member list.

## Command Aliases

LLDB's command system is hierarchical. The top-level commands include `breakpoint`, `target`, `process`, `thread`, `frame`, `register`, `memory`, `expression`, `script`, `settings`, `command`. Most have many subcommands (`breakpoint set`, `breakpoint list`, `breakpoint delete`, ...).

Typing the long form is verbose. LLDB ships with **command aliases** that map short forms to long ones:
- `b` → `_regexp-break` (regex-based breakpoint setter that figures out file:line vs function vs address)
- `bt` → `_regexp-bt` (regex-based backtrace that handles `bt 5`, `bt -c 5`, etc.)
- `c` → `process continue`
- `s` → `thread step-in`
- `n` → `thread step-over`
- `fin` → `thread step-out`
- `r` → `process launch`
- `p` → `expression --` (or sometimes `frame variable` depending on alias)
- `po` → `expression -O --` (Objective-C printObject)
- `up` → `frame select --relative=1`
- `down` → `frame select --relative=-1`
- `f` → `frame select`

User aliases use `command alias`:

```
command alias hello expression -- (void)NSLog(@"Hello, %@", $0)
```

Then `hello @"World"` expands to the full expression.

**Regex aliases** are more powerful. Each rule is a regex with capture groups and a substitution:

```
command regex git_log 's/(.+)/!git log --oneline -%1/'
```

Now `git_log 5` runs `!git log --oneline -5`. LLDB ships with regex aliases for `bt`, `b`, `j` (jump), and others, so that the same alias name accepts multiple input shapes. For example, `b` accepts:
- `b foo.cpp:42` — file:line
- `b foo`         — function name
- `b 0x1000`      — address
- `b /pattern/`   — file regex
And dispatches to `breakpoint set --file foo.cpp --line 42` / `breakpoint set --name foo` / `breakpoint set --address 0x1000` / `breakpoint set --source-pattern-regexp pattern` accordingly.

## Settings

LLDB has a hierarchical settings store, browsable with `settings list` and modified with `settings set`. Examples:
- `target.x86-disassembly-flavor` — `att` or `intel`.
- `target.process.thread.step-in-avoid-nodebug` — skip into functions without debug info.
- `target.run-args` — arguments for the next `run`.
- `target.env-vars` — environment variables.
- `target.source-map` — substitution rules for source paths (when sources moved between build and debug).
- `frame-format` / `thread-format` — printf-style templates for frame/thread display.
- `dwarf-stack-frame-format` — like frame-format but for unwound frames.
- `auto-confirm` — skip "are you sure?" prompts.
- `interpreter.prompt-on-quit` — prompt before exiting an interactive session.

Settings are scoped — most settings live on `Target` but some on `Debugger`. Setting a target-level setting before any target is created sets the default for new targets.

`~/.lldbinit` is a global LLDB init file, sourced on `lldb` startup. It contains command aliases, settings, and `command source` of script files. `~/.lldbinit-Xcode` is sourced *only* by Xcode's embedded LLDB (Xcode appends `-Xcode` to the lookup, so a single user can have different setups for command-line `lldb` vs IDE).

A per-project `.lldbinit` in the cwd is sourced if the user has set `target.load-cwd-lldbinit true` (default false for security: a malicious project could otherwise execute arbitrary script).

Platform-specific settings live under `platform.*` — e.g. `platform.plugin.linux.use-llgs` controls whether to use lldb-server vs gdbserver.

## Multi-Process Debugging

A single LLDB instance can debug multiple targets simultaneously, each with its own `Process`. The user switches between them with `target select N`.

`process attach -p PID` attaches to an existing process. `process attach --waitfor -n NAME` waits for a process matching NAME to be launched and then attaches. The waiting is implemented differently per platform:
- Linux: spin-loop on `/proc` looking for matching `/proc/*/comm` entries; once found, `ptrace(PTRACE_ATTACH)`.
- macOS: ask the kernel via `proc_listpids()` plus `proc_pidinfo()`; once found, `task_for_pid()`.

For multi-target debugging, `target create` creates additional targets. Each can be attached or launched independently. This is rarely used but is the foundation of "compare two builds" and "trace inter-process protocols" workflows.

The **platform abstraction** is what lets one LLDB on a developer's macOS host debug a process on an iOS device, a Linux server, or an Android emulator. Each platform plugin (`PlatformDarwinHost`, `PlatformDarwiniOS`, `PlatformDarwiniOSDevice`, `PlatformLinux`, `PlatformAndroid`, `PlatformWindows`, `PlatformRemoteAppleTV`) implements a shared interface:
- `LaunchProcess(target, args)` — start a new process on the platform.
- `Attach(pid)` — attach to existing.
- `RunShellCommand(...)` — execute a shell command on the platform (for inspection).
- `GetFile(remote_path)` / `PutFile(local_path, remote_path)` — file transfer.
- `ResolveExecutable(file_spec)` — find the exec file on the platform.

When debugging an iOS device, LLDB uses `PlatformRemoteiOS` which talks to `debugserver` running on the device (delivered to the device via Xcode's MobileDevice framework over USB or wifi).

## Remote Debugging Protocol

`lldb-server` runs in two modes:
- **gdbserver mode**: `lldb-server gdbserver localhost:1234 -- ./a.out arg1 arg2` — launches `a.out` and exposes the gdb-remote protocol on TCP port 1234. The client (`lldb`) connects with `(lldb) gdb-remote localhost:1234`.
- **platform mode**: `lldb-server platform --listen *:1234 --server` — exposes the platform interface (file transfer, shell exec, process listing). The client uses `platform select remote-linux; platform connect connect://host:1234` to use that platform.

The platform protocol is also gdb-remote-encoded but uses different packet types:
- `qLaunchSuccess` — did the recent launch succeed?
- `qPlatform_shell` — run a shell command on the remote.
- `qPlatform_mkdir`, `qPlatform_chmod`, `vFile:open`, `vFile:close`, `vFile:pread`, `vFile:pwrite`, `vFile:fstat`, `vFile:exists`, `vFile:unlink` — file operations.
- `qProcessInfoPID:<pid>` — info about a process by PID.
- `qfProcessInfo:matchall;` / `qsProcessInfo` — enumerate processes.

The **`qXfer:features:read:target.xml`** packet is the centerpiece of register negotiation. After connection, the client requests `qXfer:features:read:target.xml:0,fff` and the server returns an XML description of the target architecture. The XML lists each register: name, encoding (uint, float), bit-width, register group (general, float, vector), GCC reg num, DWARF reg num, semantic role (program counter, stack pointer, frame pointer, return-address). LLDB uses this to map between the gdb-remote register indices and the architecture's register set.

A snippet of `target.xml` for x86_64:

```xml
<target version="1.0">
  <architecture>i386:x86-64</architecture>
  <feature name="org.gnu.gdb.i386.core">
    <reg name="rax" bitsize="64" type="int64" regnum="0"/>
    <reg name="rbx" bitsize="64" type="int64" regnum="1"/>
    <reg name="rcx" bitsize="64" type="int64" regnum="2"/>
    ...
    <reg name="rip" bitsize="64" type="int64" regnum="16" altname="pc"/>
    <reg name="eflags" bitsize="32" type="int32" regnum="17"/>
    ...
  </feature>
</target>
```

LLDB's gdb-remote also exposes `qHostInfo` returning JSON-like key:value pairs about the target host (cpu_type, ostype, vendor, endian, ptr_size, watchpoint_exceptions_received) — these are LLDB extensions, not in the GDB spec.

## Core Files

A core file is a snapshot of process memory + registers + thread state at a moment in time. Different OSes use different formats.

**ELF core files** (Linux, BSD): an ELF file with type `ET_CORE`. The program headers list `PT_LOAD` segments (memory regions) and `PT_NOTE` segments (auxiliary info). Inside `PT_NOTE` are notes:
- `NT_PRSTATUS` — per-thread register set + signal info.
- `NT_PRPSINFO` — process info (pid, command-line, cwd).
- `NT_PRFPREG` — floating-point registers.
- `NT_X86_XSTATE` — x86 extended state (XMM, AVX).
- `NT_AUXV` — the kernel auxv vector (used by ld.so).
- `NT_FILE` — list of mapped files (path, address range).
- `NT_SIGINFO` — signal info.

LLDB's `ProcessElfCore` reads these notes, reconstructs the per-thread register state, and walks the `PT_LOAD` segments as the inferior memory. There is no real process — `Process::ReadMemory()` reads from the core file's `PT_LOAD`s.

**Mach-O core files** (macOS, iOS): a Mach-O file with type `MH_CORE`. The load commands include:
- `LC_SEGMENT_64` — memory regions (mapped from the original process).
- `LC_THREAD` — per-thread state. Contains a list of (`flavor`, `count`, `state[count]`) tuples — one for each register flavor (general, float, vector, exception state).
- `LC_NOTE` (recent macOS) — auxiliary metadata (image list, all-image-infos).

The `LC_THREAD` load command embeds the register state directly, unlike ELF where it's wrapped in a note. LLDB's `ProcessMachCore` parses `LC_THREAD` to reconstruct each thread.

**Minidumps** (Windows-style, used by Breakpad/Crashpad/Chromium): a structured binary file with a directory of streams (`MINIDUMP_THREAD_LIST_STREAM`, `MINIDUMP_MODULE_LIST_STREAM`, `MINIDUMP_MEMORY_LIST_STREAM`, etc.). LLDB's `ProcessMinidump` plugin reads these.

The "lldb just walks core memory" model: once the core is loaded, LLDB doesn't really care that there's no process. `Process::ReadMemory(addr)` works on the core's memory regions; symbol resolution works on the loaded modules; backtraces work via `.eh_frame`/`.debug_frame`. The only thing missing is `Continue()` — you can't run a core file. Some commands (like `process kill`, `process detach`) are also no-ops.

## JIT Debugging

A JIT compiler emits machine code into anonymous mmap'd memory. The debugger has no way to know about this code unless told. The standard mechanism is the **GDB JIT interface**, originally designed for GDB and adopted by LLDB.

The interface consists of two symbols in the inferior:
- `__jit_debug_register_code` — a function the JIT calls every time it adds or removes code.
- `__jit_debug_descriptor` — a global `struct jit_descriptor` containing a pointer to a linked list of `jit_code_entry` records.

```c
struct jit_code_entry {
    struct jit_code_entry *next;
    struct jit_code_entry *prev;
    const char *symfile_addr;     // pointer to an in-memory ELF file
    uint64_t symfile_size;
};

struct jit_descriptor {
    uint32_t version;
    uint32_t action_flag;          // JIT_REGISTER_FN or JIT_UNREGISTER_FN
    struct jit_code_entry *relevant_entry;
    struct jit_code_entry *first_entry;
};

void __attribute__((noinline)) __jit_debug_register_code(void) {
    asm volatile("" ::: "memory");
}
struct jit_descriptor __jit_debug_descriptor = { 1, 0, 0, 0 };
```

The JIT, when emitting code, builds an ELF file in memory containing the new code and its DWARF debug info, allocates a `jit_code_entry`, links it into the descriptor's list, sets `action_flag = JIT_REGISTER_FN`, and calls `__jit_debug_register_code()`. The function body is empty; it exists as a synchronization point. LLDB's `JITLoaderGDB` plugin sets a breakpoint at `__jit_debug_register_code`. When the breakpoint fires, LLDB reads `__jit_debug_descriptor`, walks `relevant_entry`, parses the in-memory ELF, registers it as a new `Module`, and continues.

This interface is implemented by:
- LLVM's MCJIT and ORC engines (used by LLDB's own expression evaluator).
- V8 (Chrome's JS engine).
- SpiderMonkey (Firefox's JS engine, with `--no-asm-jit-debug` to disable).
- HotSpot JVM (with `-XX:+ExposeJitDebug`-equivalent flags).
- CoreCLR (Microsoft's .NET runtime).
- Some game scripting engines.

The cost is significant: every JIT compilation must build an ELF + DWARF blob and call into the debugger. For latency-sensitive JITs, this is gated behind a flag.

## Swift Integration

The Swift debugger plugin embeds a complete Swift compiler. The interesting parts:

**TypeSystemSwift** is parallel to `TypeSystemClang`. It owns a `swift::ASTContext`. When a Swift type appears in DWARF (via `DW_AT_name = "Swift.Array<Foo>"` or a Swift mangled name), `DWARFASTParserSwift` constructs the corresponding `swift::Type` in the context.

**Module deserialization** is the heart of Swift type resolution. The `.swiftmodule` file is a serialized Swift AST — a binary representation of all public declarations in a module, including generic types, protocol conformances, and inlinable function bodies. LLDB on demand calls `swift::Serialization::Validation` to load a `.swiftmodule` into the AST context. This is expensive (the whole module's types are deserialized) but cached.

**Generic type resolution** at runtime is the magic. A Swift `Array<Foo>` value at runtime is laid out generically — the compiler doesn't know what `Foo` is until runtime. To display the array's elements, LLDB:
1. Reads the array's metadata pointer (a pointer in the array struct that points to type metadata).
2. Calls into the Swift runtime's reflection API (`swift_demangle`, `swift_getTypeByMangledName`) via expression evaluation.
3. Recovers the concrete type.
4. Uses TypeSystemSwift to interpret the array's storage.

**Demangling** Swift symbols is done by `swift::Demangle::demangleSymbolAsString()`. Swift's mangled names look like `$s4test1AVMa` and decode to `type metadata accessor for test.A`. LLDB embeds the demangler so that `nm`-output names become readable Swift names in backtraces.

The `swift::IRGenContext` is also embedded so that LLDB can compile Swift expressions just like Clang expressions — `expr -- self.foo + 3` in a Swift frame is a real Swift expression, type-checked against the surrounding scope, IR-gen'd, JIT-compiled, and run.

## Objective-C Integration

The Objective-C runtime is heavily reflective by design — almost every aspect of an object is queryable at runtime. LLDB's `AppleObjCRuntimeV2` plugin uses these runtime APIs (or, where possible, direct memory reads of the runtime's data structures) to introspect ObjC state:

- `object_getClass(obj)` — returns the Class of an object. Implemented as `obj->isa & ISA_MASK` (with pointer authentication on ARM64).
- `class_copyMethodList(cls, &count)` — returns the methods of a class. The method list is a `method_list_t` struct in the class's `class_rw_t`.
- `class_getName(cls)` — the class's name (a `char *` in `class_ro_t`).
- `class_getSuperclass(cls)` — the superclass.
- `protocol_getMethodDescription(proto, sel, ...)` — protocol introspection.

LLDB walks the class hierarchy by reading these structures directly from memory, avoiding round-trips through expression evaluation when possible. The trick is that the `class_t`/`class_ro_t`/`class_rw_t` layout is a private implementation detail that has changed between OS releases — `AppleObjCRuntimeV2` embeds version-specific layout knowledge for each Mac OS X / iOS version.

The Objective-C **method dispatch** is via `objc_msgSend(receiver, selector, args...)`. When a user steps into a method call, LLDB sees the call to `objc_msgSend`, inspects the receiver's class and the selector, looks up the actual implementation (via `class_getMethodImplementation`), and steps to it. This is the "step-through trampoline" mechanism — `objc_msgSend` is recognized as a trampoline by name, and stepping continues into the called method rather than into `objc_msgSend`'s assembly.

The **`po`** (`expression -O --`) command on an ObjC object calls `[obj description]` (or `[obj debugDescription]` if available) by synthesizing a tiny ObjC expression and evaluating it in the inferior. The result is a `NSString*`; LLDB extracts the C-string from it via `[result UTF8String]` and displays.

`@"NSString"` literals in expressions are compiled to runtime `NSString` constants by Clang's ObjC frontend (with the literal allocator routine, currently `__CFStringMakeConstantString`). The expression evaluator handles this transparently.

## Crash Report Routing

When a process on macOS crashes (uncaught Mach exception, signal), the kernel checks the task's exception ports. If a debugger is attached, the debugger gets the exception. If not, the exception goes to the bootstrap server, which routes to ReportCrash.

`ReportCrash` is a system daemon at `/System/Library/CoreServices/ReportCrash`. It receives the exception, samples the crashed process's threads (via Mach calls), generates an `.ips` file, and writes to `~/Library/Logs/DiagnosticReports/` (user crashes) or `/Library/Logs/DiagnosticReports/` (root/system crashes).

The **`.ips`** format is JSON-Lines: the first line is metadata (process name, version, OS version), and the second line is a JSON object with the full report (stack traces, register dumps, memory bands, image list with UUIDs and load addresses).

To symbolicate an `.ips` file:
1. Read the binary's UUID and load address from the report.
2. Find the matching dSYM (by UUID).
3. For each address in the backtrace, run `atos -arch arm64 -o /path/to/binary -l 0xLOADADDR 0xADDRESS` — `atos` consults the dSYM's DWARF and prints `function (file:line)`.

`atos` is itself implemented atop LLDB's symbol-resolution machinery. The tool is shipped as `/usr/bin/atos` and links `LLDB.framework`'s SBAPI internally.

LLDB's backtraces are formatted to be `atos`-compatible — each frame has the form `frame #N: 0xADDRESS module`function + offset at file:line` — and crash reports use the same convention.

## SIP / AMFI Restrictions

System Integrity Protection (SIP) is the kernel-level security feature on modern macOS that prevents even root from modifying system binaries or tracing system processes. AMFI (Apple Mobile File Integrity) is the kernel module that enforces SIP and code-signing.

The relevant restrictions:
- **`task_for_pid()` on Apple-signed binaries fails** unless the caller has the `com.apple.system-task-ports` entitlement (very rare). This means you cannot `lldb` attach to `/usr/bin/ssh`, `/System/Library/...`, or any Apple system tool.
- **Tracing your own binaries** requires the binary to have the `get-task-allow` entitlement. Xcode adds this automatically in Debug builds. Release builds do not have it; you can't `lldb` attach to a release-build app you ship.
- **Library injection** (DYLD_INSERT_LIBRARIES) is disabled for system binaries.
- **DTrace** is restricted: probes on system processes don't fire.

The `get-task-allow` entitlement is set in the binary's signed entitlements blob. To check: `codesign -d --entitlements - /path/to/binary`. To debug a binary that lacks it, you must either re-sign it with the entitlement (only possible if you have the developer cert), disable SIP (single-user mode + `csrutil disable`), or run on a developer-mode iOS device.

The list of SIP-protected paths is in `/System/Library/Sandbox/rootless.conf`. Roughly: `/System/`, `/usr/` (except `/usr/local/`), parts of `/bin/` and `/sbin/`, the kernel.

When LLDB fails with "the kernel returned EPERM (1)" on `task_for_pid`, the issue is almost always SIP/AMFI denying the request.

## Performance Characteristics

The dominant cost in an LLDB session on a real binary is **DWARF parsing**. Specifically:
- **Initial CU index build**: walking `.debug_info` to find every CU. Fast, just header parsing.
- **Symbol-name accelerator load**: the `.debug_names`/`.apple_names` hash table is loaded into memory. Fast.
- **Per-symbol DIE parse**: when you set a breakpoint, the DIE for the named function is fully parsed, including its children (parameters, local variables) and their types. Slow — pulls in the entire transitive type graph.
- **Per-frame variable enumeration**: when you `frame variable`, every local in the frame's scope is parsed. Each local's type triggers further parsing.
- **Per-CU line table decode**: when the user asks "what line is PC X?", the CU's line program is decoded. Cached after first decode.

Lazy DWARF parsing keeps cold CUs out of memory. On a binary with 10,000 CUs, only the dozen or so CUs containing the user's breakpoints and frame locations are fully decoded; the rest are skipped.

The `--target-debug-info-dir` (or per-target `target.debug-file-search-paths`) lets you specify additional paths where dSYMs/debug-info-only binaries live. This is especially useful in build-farm setups where the dSYM lives somewhere other than the developer's home directory.

The **dSYM cache** speeds up repeated invocations. `dsymForUUID` (a tool inside Xcode) caches dSYMs in `~/Library/Developer/Xcode/UserData/Symbols/<UUID>/` indexed by UUID. LLDB consults this cache before searching the filesystem broadly.

A modern LLDB session on a 100MB binary with full debug info typically:
- Starts in 100-300ms.
- Sets a function breakpoint in 10-50ms (mostly hash lookup + one DIE parse).
- Hits a breakpoint and prints a backtrace in 50-200ms (mostly unwinding 20-50 frames each consulting `.eh_frame` and the line table).
- Evaluates a non-trivial expression in 100-500ms (mostly Clang parsing, IRGen, JIT compile, function call setup).

Profiling LLDB's performance: `(lldb) log enable -t lldb dwarf` shows DWARF parse events. `(lldb) settings set target.show-progress true` shows progress meters during long DWARF operations.

## Common Internals Errors

**"the dSYM is for a different binary"** — UUID mismatch. The executable's `LC_UUID` does not match any dSYM's `LC_UUID`. Causes: rebuilt the binary without rebuilding the dSYM; archived from one Xcode version, opened in another; copying just the binary to another machine without the dSYM. Fix: regenerate the dSYM with `dsymutil /path/to/binary -o /path/to/binary.dSYM`, or find the correct dSYM in your build outputs. Verify with `dwarfdump --uuid /path/to/binary` and `dwarfdump --uuid /path/to/binary.dSYM`.

**"Compilation unit X has no aranges"** — when a CU lacks `.debug_aranges` (or it's incomplete/wrong), LLDB falls back to walking the CU's DIE tree to determine address coverage. This is much slower. Fix: rebuild with `-gpubnames` (Clang) or `-fdebug-types-section`, or use DWARF v5 which has better default accelerators.

**"Failed to launch process: 'A' (1)"** — fork failed (rare). Or, on macOS, the binary lacks `get-task-allow` and SIP is enabled. Check `codesign -d --entitlements -`.

**"failed to attach to process: status=-1 err=ptrace: Operation not permitted"** — Linux kernel `ptrace_scope` setting is restrictive. Cat `/proc/sys/kernel/yama/ptrace_scope`: 0 = unrestricted, 1 = parent-only, 2 = admin-only, 3 = no ptrace. Fix: `echo 0 | sudo tee /proc/sys/kernel/yama/ptrace_scope` (temporary), or edit `/etc/sysctl.d/`.

**"error: invalid frame address 0x..."** — Stack unwind failed because `.eh_frame` was missing or corrupt and frame-pointer fallback couldn't proceed. Common in heavily optimized release builds without `-fno-omit-frame-pointer`. Fix: build with frame pointers, or with full eh_frame.

**"warning: Module ... has been compiled with optimization (...). Stepping may behave oddly; variables may not be available"** — DWARF reports the CU is optimized (`-O1`+). LLDB warns because location lists may show variables as "optimized out" at certain PCs, and stepping may skip lines. Acceptable but expected.

**"error: Couldn't materialize: couldn't get the value of variable X"** — expression evaluator tried to read a variable's value, but the location list at the current PC says the variable is not in any location (optimized out). Workaround: rebuild with `-O0` (or `-Og`).

**"error: process exited with status -1 (lost connection)"** — gdb-remote socket dropped. Often happens when `lldb-server` or `debugserver` crashes. Check the system log; in many cases it's a fault in `lldb-server`'s own DWARF parsing.

## Idioms

**Use SBAPI for tooling, not commandline parsing.** If you're writing a tool that automates LLDB, do `import lldb` and use SBAPI directly — don't shell out to `lldb` and parse its output. Output formatting changes; the SBAPI is stable. Concretely: write `target = debugger.CreateTarget("a.out"); bp = target.BreakpointCreateByName("foo")`, not `subprocess.Popen(["lldb", ...])`.

**Respect the platform abstraction.** When debugging an iOS device, file paths on the device are not file paths on the host. `/var/mobile/Containers/Data/Application/UUID/Library/...` exists on the device; the host may map it as `/Users/me/Library/Developer/Xcode/iOS DeviceSupport/.../Symbols/var/mobile/...`. Use `SBPlatform`'s file APIs (`Get`, `Put`) rather than assuming host paths work.

**Thread plans for synchronization.** When automating a debugging session, use `process.Continue()` and event-based synchronization (or synchronous mode) rather than busy-polling `process.GetState()`. The thread-plan machinery is the canonical way to express "do this multi-step debugging operation atomically".

**DWARF parsing is the dominant startup cost.** When optimizing LLDB performance, focus on reducing DWARF size or improving DWARF accelerators. Use `-gsplit-dwarf` (split debug info into `.dwo` files), `-gpubnames`, `-fdebug-types-section`, or DWARF v5. Strip unused CUs from third-party libraries.

**Pretty-printers > raw structs.** For any complex C++ container or runtime object, ship a Python synthetic provider. Without it, debugging produces `_M_node = 0x12345678, _M_color = _S_red` confusion; with it, debugging produces `[ "key1": value1, "key2": value2 ]` clarity.

**Don't reach into private LLDB headers.** Even though they're in the source tree, `lldb/include/lldb/Core/`, `lldb/include/lldb/Symbol/`, etc., are private. The internal class layout changes with every release; depending on `Process` or `Module` directly will break. SBAPI is the contract.

**Use `--persistent-result false` for transient expressions.** Each `expr foo()` creates a `$0`, `$1`, ... persistent result accessible by name later. This holds memory references, can prevent GC, and clutters the namespace. For one-off expressions, `expr -X false -- foo()` skips persistence.

**Set `target.process.thread.step-avoid-libraries` to skip system frames.** When stepping in, you usually don't want to descend into `libc`, `libsystem_kernel`, `libobjc`. Set this to a regex of library names to skip — LLDB will set up step-out plans automatically.

**Use Python `lldb.debugger.HandleCommand("...")` for one-line scripting.** From a Python session, the simplest way to drive LLDB is to dispatch CLI commands. You don't have to use SBAPI for everything; the CLI is also an API.

## See Also

- [gdb (sheets/system/gdb.md)](../system/gdb.md) — the older, GPL'd debugger; many similar concepts but a different process model and a less consistent type system.
- [delve (sheets/system/delve.md)](../system/delve.md) — Go's debugger; goroutine-aware; different unwinder.
- [pdb (sheets/system/pdb.md)](../system/pdb.md) — Python's debugger; pure-Python; very different model.
- perf — Linux's performance counter framework; complementary to debugging.

## References

- LLDB Architecture Overview — https://lldb.llvm.org/use/architecture.html
- LLDB Source Tree — https://github.com/llvm/llvm-project/tree/main/lldb/source
- LLDB GDB-Remote Protocol Extensions — https://github.com/llvm/llvm-project/blob/main/lldb/docs/lldb-gdb-remote.txt
- DWARF Debugging Information Format Standard, Version 5 — https://dwarfstd.org/doc/DWARF5.pdf
- DWARF v4 — https://dwarfstd.org/doc/DWARF4.pdf
- GDB Remote Serial Protocol — https://sourceware.org/gdb/current/onlinedocs/gdb/Remote-Protocol.html
- Mach Exception Ports (XNU source) — https://github.com/apple-oss-distributions/xnu
- The Go runtime's GDB JIT interface — https://sourceware.org/gdb/current/onlinedocs/gdb/JIT-Interface.html
- Apple compact unwind format (libunwind source) — https://github.com/apple-oss-distributions/libunwind
- Mach-O File Format Reference — Apple Developer documentation, archive.org
- ELF and DWARF tooling: dwarfdump, eu-readelf, objdump --dwarf
- DWARF v5 split debug info — https://gcc.gnu.org/wiki/DebugFission
- LLDB Python Reference — https://lldb.llvm.org/python_api.html
- The LLVM MC project (disassembler) — https://llvm.org/docs/CodeGenerator.html#machine-code-description-classes
- ASAN runtime API — https://github.com/google/sanitizers/wiki/AddressSanitizer
- Swift LLDB integration (apple/llvm-project, swift/release/X.Y branch) — https://github.com/apple/llvm-project
- Apple System Integrity Protection — https://developer.apple.com/library/archive/documentation/Security/Conceptual/System_Integrity_Protection_Guide/
- Crashpad/Breakpad minidump format — https://chromium.googlesource.com/crashpad/crashpad/
- Mach-O LC_UUID and code signing — Apple's `man codesign`, `man dwarfdump`
