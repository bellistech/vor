# Delve (Go Debugger)

Go-native source-level debugger with goroutine awareness, JSON-RPC API, and full Go type system support — the canonical debugger for Go programs.

## Setup

Install the latest Delve:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

The binary lands in `$GOBIN` (defaults to `$GOPATH/bin`, which is usually `$HOME/go/bin`). Make sure that directory is on `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Verify the install:

```bash
dlv version
```

Expected output:

```bash
Delve Debugger
Version: 1.22.1
Build: $Id: abc123... $
```

Update Delve when Go updates — newer Go versions sometimes break older Delve:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### macOS codesign requirement

On macOS, Delve must be codesigned to attach to running processes. Without signing, `dlv attach PID` returns `could not attach to pid X: operation not permitted`.

Apple-suggested self-sign workflow:

```bash
# 1. Open Keychain Access -> Certificate Assistant -> Create a Certificate
#    Name: dlv-cert
#    Identity: Self Signed Root
#    Certificate Type: Code Signing
# 2. Find the cert, set Trust -> Code Signing -> Always Trust
# 3. Sign the binary
codesign -s dlv-cert $(which dlv)
```

Or use the Delve-provided helper:

```bash
cd $(go env GOPATH)/src/github.com/go-delve/delve
make install
```

That Makefile signs with `dlv-cert` automatically.

### Linux ptrace permissions

Linux's Yama LSM restricts non-parent ptrace via `kernel.yama.ptrace_scope`:

```bash
cat /proc/sys/kernel/yama/ptrace_scope
```

Values: `0` = anything attaches, `1` = parent-only (default), `2` = admin-only, `3` = no ptrace.

Loosen for the session:

```bash
sudo sysctl kernel.yama.ptrace_scope=0
```

Or `sudo dlv attach PID` — the simpler answer.

## Why Delve

GDB has poor Go support. GDB:

- Confuses goroutines for OS threads — Go's M:N scheduler runs many goroutines on few threads.
- Doesn't understand Go's calling convention — registers, stack growth, panic propagation.
- Doesn't grok Go's runtime — channels, mutexes, the GC, deferred calls.
- Can't evaluate Go expressions — `print myMap["key"]` either fails or lies.
- Misreads Go's stack frames after stack growth — the runtime moves stacks; GDB's frame pointer arithmetic is wrong.

The canonical rule:

```bash
# DON'T:
gdb ./mybinary

# DO:
dlv exec ./mybinary
```

Delve speaks Go natively. It knows about goroutines, the runtime, Go types, and the calling convention. It evaluates real Go expressions. It tracks goroutine state across scheduler context switches.

Use GDB only for non-Go binaries (C, C++, Rust). Use Delve for everything Go.

## Modes — Starting Sessions

### dlv debug — compile and debug

```bash
dlv debug ./cmd/app -- --flag value
```

Builds with `-gcflags='all=-N -l'` (no optimization, no inlining) and starts a session. Args after `--` go to the binary.

### dlv exec — debug a pre-built binary

```bash
dlv exec ./bin/app -- --flag value
```

Use when you already built with debug info. Faster than rebuilding. Critical: build with `-gcflags='all=-N -l'` first.

### dlv test — debug tests

```bash
dlv test ./pkg/registry
```

Builds the test binary with debug flags, drops into Delve at the test entrypoint.

### dlv test — specific test

```bash
dlv test ./pkg/registry -- -test.run TestParseSheet -test.v
```

Args after `--` go to the test binary, prefixed `-test.`. Common flags:

- `-test.run REGEX` — filter test names
- `-test.v` — verbose
- `-test.timeout 30s` — kill if hung
- `-test.count 1` — disable cache

### dlv attach — running process

```bash
dlv attach 12345
```

Pause the process, drop into a session. PID from `pgrep`, `ps`, `ss`, etc.

```bash
dlv attach $(pgrep myapp)
```

When you `detach`, the process keeps running. `quit` with `--continue` does the same.

### dlv core — post-mortem

```bash
GOTRACEBACK=crash ./bin/app
# crashes, dumps core to ./core
dlv core ./bin/app ./core
```

Inspect a dead program's state. `GOTRACEBACK=crash` makes Go dump a core; otherwise it just prints a stack trace.

On macOS, core dumps are disabled by default. Linux needs `ulimit -c unlimited` first.

### dlv connect — headless server

```bash
dlv connect 127.0.0.1:2345
```

Connect to a running headless Delve. See Headless Mode.

## Headless Mode

Run a Delve server that editors connect to:

```bash
dlv exec --headless --listen=:2345 --api-version=2 --accept-multiclient ./bin/app
```

Flags:

- `--headless` — no interactive prompt; speak the API only
- `--listen=:2345` — TCP port (use `127.0.0.1:2345` to bind only to localhost)
- `--api-version=2` — current API; v1 is deprecated
- `--accept-multiclient` — allow reconnect (editors crash, you reconnect)
- `--continue` — auto-continue once a client connects (useful for kubernetes/CI)
- `--log` — enable debug logging
- `--log-output=debugger,gdbwire,lldbout,debuglineerr` — what to log

Editors that drive Delve via this API:

- VS Code (Go extension)
- GoLand and other JetBrains IDEs
- Neovim with `nvim-dap-go`
- Emacs with `dap-mode`

To connect a CLI client to the server:

```bash
dlv connect 127.0.0.1:2345
```

Multiple clients can connect with `--accept-multiclient`. Detach without killing the target:

```bash
(dlv) disconnect
```

## Editor Integration

### VS Code

Install the Go extension. The extension uses `dlv dap` (Debug Adapter Protocol) under the hood since VS Code 1.55.

Canonical `.vscode/launch.json`:

```bash
cat > .vscode/launch.json <<'EOF'
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Package",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${fileDirname}",
      "args": []
    },
    {
      "name": "Debug Test",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${fileDirname}"
    },
    {
      "name": "Attach to Process",
      "type": "go",
      "request": "attach",
      "mode": "local",
      "processId": 0
    },
    {
      "name": "Attach to Remote",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "${workspaceFolder}",
      "port": 2345,
      "host": "127.0.0.1"
    }
  ]
}
EOF
```

Key settings (`settings.json`):

```bash
{
  "go.delveConfig": {
    "debugAdapter": "dlv-dap",
    "showLog": false,
    "apiVersion": 2,
    "dlvLoadConfig": {
      "followPointers": true,
      "maxVariableRecurse": 1,
      "maxStringLen": 64,
      "maxArrayValues": 64,
      "maxStructFields": -1
    }
  }
}
```

`dlv-dap` is the new Debug Adapter mode (preferred). `dlv` (legacy) uses JSON-RPC.

### GoLand / IntelliJ

Built-in. Run -> Edit Configurations -> Go Build/Test. The IDE bundles its own Delve copy and codesigns it on macOS.

To attach to remote: Run -> Edit Configurations -> + -> Go Remote -> Host/Port.

### Neovim

`nvim-dap` + `nvim-dap-go`:

```bash
-- in your config
require('dap-go').setup({
  delve = {
    path = 'dlv',
    initialize_timeout_sec = 20,
    port = '${port}',
    args = {},
    build_flags = '',
  },
})

-- keymaps
vim.keymap.set('n', '<F5>', require('dap').continue)
vim.keymap.set('n', '<F10>', require('dap').step_over)
vim.keymap.set('n', '<F11>', require('dap').step_into)
vim.keymap.set('n', '<S-F11>', require('dap').step_out)
vim.keymap.set('n', '<leader>b', require('dap').toggle_breakpoint)
vim.keymap.set('n', '<leader>dt', require('dap-go').debug_test)
```

### Emacs

`dap-mode`:

```bash
(use-package dap-mode
  :config
  (require 'dap-go)
  (dap-go-setup))
```

## Commands — Core Subset

When inside `(dlv)`:

### Breakpoints

```bash
(dlv) b main.main
(dlv) b ./main.go:42
(dlv) b github.com/go-delve/delve/cmd/dlv/cmds.New
(dlv) b 0x401234
```

Forms accepted:

- `package.function` — `main.main`, `fmt.Printf`, `(*MyType).Method`
- `file:line` — relative or absolute path
- `address` — hex address
- `+N` / `-N` — relative to current line

List breakpoints:

```bash
(dlv) bp
Breakpoint 1 at 0x401234 for main.main() ./main.go:10 (0)
Breakpoint 2 at 0x405678 for main.process() ./main.go:42 (0)
```

The `(0)` is the hit count.

Remove:

```bash
(dlv) clear 1
(dlv) clearall
```

Conditional breakpoint — only fire when expression is true:

```bash
(dlv) b main.process
Breakpoint 1 set at 0x405678 for main.process() ./main.go:42
(dlv) cond 1 i > 100 && err != nil
```

The condition is evaluated in the goroutine that hits the breakpoint. Cost: every hit evaluates the expression — high-rate breakpoints with conditions can slow execution dramatically.

### Execution

```bash
(dlv) c              # continue
(dlv) n              # next (step over)
(dlv) s              # step (step into)
(dlv) so             # stepout (run until current func returns)
(dlv) si             # stepi (single instruction)
(dlv) r              # restart
(dlv) q              # quit
```

`continue` runs to next breakpoint or program exit. `next` and `step` are line-granular; `stepi` is instruction-granular (assembly debugging).

Restart with new args:

```bash
(dlv) restart -- newarg1 newarg2
```

## Inspection

### print / p

Full Go expression syntax:

```bash
(dlv) p i
42
(dlv) p mySlice[0]
"first"
(dlv) p myMap["key"]
"value"
(dlv) p myStruct.Field
"field value"
(dlv) p len(mySlice)
3
(dlv) p &myStruct
*main.MyStruct {Field: "field value", N: 42}
(dlv) p *myPointer
main.MyStruct {Field: "field value", N: 42}
```

Also supports method calls, type assertions, slicing:

```bash
(dlv) p mySlice[1:3]
(dlv) p myInterface.(*main.Concrete)
(dlv) p myMap == nil
```

Function calls (with caveats — see below):

```bash
(dlv) call myFunc(42)
```

Function call gotchas: only safe when stopped at a safe-point, must be allowed via config, can't call functions that block on the goroutine you stopped.

### whatis

Show type without evaluating fully:

```bash
(dlv) whatis myVar
struct { ... }
(dlv) whatis myInterface
io.Reader
```

### locals / args / vars

```bash
(dlv) locals
i = 42
err = error nil
mySlice = []string len: 3, cap: 4, [...]
(dlv) args
ctx = context.Context(*context.cancelCtx) ...
input = "hello"
(dlv) vars main
main.GlobalVar = 0
main.config = ...
```

`vars` takes an optional package filter. Without arg, dumps all package vars (slow on big binaries).

### regs

```bash
(dlv) regs
   Rip = 0x0000000000401234
   Rsp = 0x000000c000040f60
   ...
(dlv) regs -a
   # all including FP/SSE
```

### display

Show expressions on every stop:

```bash
(dlv) display -a i
0: i = 42
(dlv) c
0: i = 43
> main.process() ./main.go:42 (hits 2)
(dlv) display -d 0
```

`-a` adds, `-d N` deletes by index.

## Stack

```bash
(dlv) bt
0  0x0000000000405678 in main.process at ./main.go:42
1  0x0000000000401234 in main.main at ./main.go:15
2  0x0000000000437890 in runtime.main at /usr/local/go/src/runtime/proc.go:250
3  0x0000000000457abc in runtime.goexit at /usr/local/go/src/runtime/asm_amd64.s:1571
```

Switch frame:

```bash
(dlv) frame 1
> main.main() ./main.go:15
(dlv) locals
# now shows main.main's locals
```

Or relative:

```bash
(dlv) up
(dlv) down
```

`up` and `down` walk the stack interactively without explicit frame numbers.

## Goroutines (the killer feature)

This is what GDB cannot do.

### List goroutines

```bash
(dlv) gr
* Goroutine 1 - User: ./main.go:15 main.main (0x401234) (thread 12345)
  Goroutine 2 - User: /usr/local/go/src/runtime/proc.go:367 runtime.gopark (0x437bb0)
  Goroutine 17 - Go: ./worker.go:5 main.worker (0x402345) (thread 12346)
  ...
```

The `*` marks the current goroutine. `User` is where the goroutine is currently executing user code; `Go` is where the goroutine was started.

### Verbose listing

```bash
(dlv) gr -t
# tab-formatted, easier to grep
```

### Switch to a goroutine

```bash
(dlv) goroutine 17
Switched from 1 to 17 (thread 12346)
(dlv) bt
# now shows goroutine 17's stack
```

### Filter by status

```bash
(dlv) goroutines -with running
(dlv) goroutines -with runnable
(dlv) goroutines -with waiting
(dlv) goroutines -with syscall
```

States:

- `running` — actually executing on a thread
- `runnable` — ready, waiting for a thread
- `waiting` — blocked (channel, mutex, network, etc.)
- `syscall` — in a syscall

Filter by user code:

```bash
(dlv) goroutines -with user main.worker
```

### Goroutine-specific commands

```bash
(dlv) goroutine 17 bt
# stack of goroutine 17 without switching to it
(dlv) goroutine 17 print myVar
# evaluate in goroutine 17's context
```

### Goroutine filtering toggle

```bash
(dlv) gr -on
(dlv) gr -off
```

Turn goroutine display filtering on/off when listing.

## Threads (the M's)

Go's M:N scheduler maps M goroutines onto N OS threads. Usually you care about goroutines, not threads.

```bash
(dlv) threads
* Thread 12345 at 0x405678 ./main.go:42 main.process
  Thread 12346 at 0x402345 ./worker.go:10 main.worker
  Thread 12347 at 0x437bb0 runtime.gopark
```

Switch:

```bash
(dlv) thread 12346
```

When you'd want this: debugging cgo, debugging the runtime itself, tracking down a CPU-spinning thread.

## Source

```bash
(dlv) list
> 41:    func process(input string) error {
  42:        if len(input) == 0 {
=>43:            return errors.New("empty input")
  44:        }
  45:        // ...
```

`=>` marks the current line.

```bash
(dlv) list main.main
(dlv) list ./main.go:50
(dlv) list +10        # 10 lines forward
(dlv) list -5         # 5 lines back
```

### funcs / types / sources

```bash
(dlv) funcs main\.
main.main
main.process
main.worker
(dlv) funcs ^net/http\.
net/http.Get
net/http.Post
...
(dlv) types
(dlv) types ^main\.
main.Config
main.Server
(dlv) sources main
./main.go
./worker.go
```

All accept Go regex.

## Memory and Examination

### examinemem

```bash
(dlv) examinemem 0xc000040f60 --size 64 --fmt hex
0xc000040f60: 0x00 0x00 0x00 0x00 ...
(dlv) examinemem 0xc000040f60 --size 32 --fmt octal
(dlv) examinemem &myStruct --size 16 --fmt bin
```

`--fmt` accepts: `hex`, `oct`, `bin`, `dec`, `ascii`, `addr`.

Shorter alias: `x`:

```bash
(dlv) x -fmt hex -count 16 0xc000040f60
```

### disassemble / disas

Current function:

```bash
(dlv) disassemble
TEXT main.process(SB) ./main.go
        main.go:41      0x405678        4154            push r12
        main.go:41      0x40567a        55              push rbp
        main.go:42 =>   0x40567b        488d6c2408      lea rbp, [rsp+0x8]
        ...
```

`=>` marks current PC.

Specific function:

```bash
(dlv) disas -l main.main
```

`-l` flag: `linear` ordering rather than control-flow.

By address range:

```bash
(dlv) disas 0x405678 0x405700
```

## Conditions and Tracepoints

### Conditional breakpoints

Set then condition:

```bash
(dlv) b main.process
Breakpoint 1 set at 0x405678 for main.process() ./main.go:42
(dlv) cond 1 i > 10 && err != nil
```

The breakpoint number from `bp`. The expression is full Go syntax — comparisons, function calls, struct field access.

Cost: every hit evaluates the expression. For very hot functions, this is expensive — Delve has to break, evaluate, and resume.

### trace — auto-continue with print

```bash
(dlv) trace main.process
Tracepoint 2 set at 0x405678 for main.process() ./main.go:42
(dlv) c
> main.process(input="hello") ./main.go:42 (hits goroutine(1):1 total:1)
> main.process(input="world") ./main.go:42 (hits goroutine(1):2 total:2)
```

`trace` is a breakpoint that auto-continues. Useful for "what's calling this 1000 times" without losing flow control.

`tracepoint` is the same; both names work.

To trace with custom expression:

```bash
(dlv) trace main.process
(dlv) on 2 print input
(dlv) c
> main.process(input="hello") ./main.go:42
input: "hello"
> main.process(input="world") ./main.go:42
input: "world"
```

`on N expr` runs `expr` every time breakpoint N is hit. Combine with `cond` to filter.

### The "trace where x is set without stopping" pattern

```bash
(dlv) trace main.setX
(dlv) on 1 print x
(dlv) c
# ... captures every call, every value ...
```

For a write to a specific variable, use a watchpoint (Delve >=1.21):

```bash
(dlv) watch -w x
Watchpoint 3 set at 0xc000016108 for x
```

`-w` write, `-r` read, `-rw` both. Works on global and local variables. Reset when the variable goes out of scope.

## JSON-RPC API

The headless server speaks JSON-RPC 2.0. This is the foundation of all editor integrations.

Connect:

```bash
nc 127.0.0.1 2345
```

Send a request — the wire format is JSON-RPC over a raw TCP stream:

```bash
{"method": "RPCServer.GetVersion", "params": [], "id": 1}
```

Response:

```bash
{"id": 1, "result": {"DelveVersion": "1.22.1", "APIVersion": 2}, "error": null}
```

### Common API methods

API v2 methods (prefixed `RPCServer.`):

- `GetVersion` — server info
- `SetBreakpoint` — `{Breakpoint: {File, Line, FunctionName, Cond, ...}}`
- `GetBreakpoint` — `{Id, Name}`
- `ListBreakpoints` — `{}`
- `ClearBreakpoint` — `{Id}`
- `Command` — `{name: "continue"|"next"|"step"|"stepOut"|"stepInstruction"|"halt"}`
- `State` — current state (running, halted, exited)
- `ListThreads` — all threads
- `ListGoroutines` — `{Start, Count}` paginated
- `Stacktrace` — `{Id: goroutineID, Depth, Full, Cfg}`
- `Eval` — `{Scope: {GoroutineID, Frame}, Expr, Cfg}`
- `Set` — assign to variable: `{Scope, Symbol, Value}`
- `ListLocalVars` / `ListFunctionArgs` / `ListPackageVars`
- `ListRegisters` — `{Threadid, IncludeFp}`
- `Restart` — `{Position, ResetArgs, NewArgs}`
- `Detach` — `{Kill: bool}`

Example — set breakpoint:

```bash
{
  "method": "RPCServer.CreateBreakpoint",
  "params": [{
    "Breakpoint": {
      "File": "/path/to/main.go",
      "Line": 42,
      "Cond": "i > 10"
    }
  }],
  "id": 2
}
```

Example — continue:

```bash
{
  "method": "RPCServer.Command",
  "params": [{"name": "continue"}],
  "id": 3
}
```

Example — eval:

```bash
{
  "method": "RPCServer.Eval",
  "params": [{
    "Scope": {"GoroutineID": -1, "Frame": 0},
    "Expr": "myVar",
    "Cfg": {"FollowPointers": true, "MaxVariableRecurse": 1}
  }],
  "id": 4
}
```

`GoroutineID: -1` means current.

### Debug Adapter Protocol (DAP)

`dlv dap` runs as a Debug Adapter — different protocol than JSON-RPC. DAP is what VS Code prefers.

```bash
dlv dap --listen=127.0.0.1:2345
```

DAP is documented at `microsoft.github.io/debug-adapter-protocol`. The `dlv-dap` mode is the modern path.

## .dlvrc Config

`~/.dlvrc` is sourced at startup. One command per line.

```bash
cat > ~/.dlvrc <<'EOF'
config alias bp breakpoints
config alias gr goroutines
config alias l list
config max-string-len 256
config max-array-values 128
config max-variable-recurse 2
config show-location-expr true
EOF
```

`config` settings (in-session or in `.dlvrc`):

- `aliases` / `alias` — define shortcuts
- `max-string-len N` — print at most N chars of strings
- `max-array-values N` — print at most N slice elements
- `max-variable-recurse N` — recurse N levels into nested structs/pointers
- `show-location-expr true` — print location expressions on stop
- `substitute-paths` — see next section

View current config:

```bash
(dlv) config -list
```

Set in session:

```bash
(dlv) config max-string-len 512
```

## Substitute Paths

When the binary was built on a different machine, source paths are wrong:

```bash
(dlv) list main.main
Command failed: open /build/host/path/main.go: no such file or directory
```

Map old paths to new:

```bash
(dlv) config substitute-path /build/host/path /home/me/myrepo
```

Or in `.dlvrc`:

```bash
config substitute-path /build/host/path /home/me/myrepo
```

Multiple mappings stack — apply in order. Use case: CI builds in `/builds/...`, you debug locally in `/home/...`.

For `dlv` invocation:

```bash
dlv exec --init <(echo "config substitute-path /build /home/me/repo") ./bin
```

`--init FILE` runs the file as if it were `.dlvrc`.

## Test Debugging

`dlv test ./pkg` builds the test binary with `-gcflags='all=-N -l'` automatically.

```bash
dlv test ./internal/registry
```

Drops into Delve at the test entry. Set breakpoints in tests or production code.

### Run specific test

```bash
dlv test ./internal/registry -- -test.run TestParseSheet -test.v
```

Common test flags:

- `-test.run REGEX` — only matching tests
- `-test.v` — verbose
- `-test.count 1` — disable cache
- `-test.timeout 30s` — kill if hung
- `-test.parallel 1` — disable parallelism (helps debugging)

### The "test panics, drop into delve" workflow

```bash
# Test panics in CI? Reproduce locally with delve.
dlv test ./pkg -- -test.run TestThatPanics
(dlv) c
# panics
(dlv) bt
# inspect the panic stack
(dlv) goroutines
# see what other goroutines were doing
```

Add a breakpoint before the panic:

```bash
(dlv) b ./pkg/buggy.go:42
(dlv) restart
(dlv) c
```

### Catching panics

Delve doesn't auto-break on panic by default. Set a breakpoint in `runtime.gopanic`:

```bash
(dlv) b runtime.gopanic
(dlv) c
```

Now any panic stops Delve at the runtime level. Walk up the stack with `bt` and `frame N` to see user code.

## Building Binaries with Optimization Off

The canonical Go build for debugging:

```bash
go build -gcflags='all=-N -l' -o bin/app ./cmd/app
```

Flags:

- `-N` — disable optimization
- `-l` — disable inlining
- `all=` — apply to all packages including dependencies

Without these, the compiler:

- Eliminates "unused" variables → `could not find symbol value for X`
- Inlines small functions → `f` doesn't appear on the stack
- Reorders instructions → stepping jumps unpredictably
- Folds constants → `print myConst` may show wrong value

Strip vs debug:

```bash
# Production (smaller binary):
go build -ldflags='-s -w' -o bin/app ./cmd/app
# -s strips symbol table, -w strips DWARF

# Debugging:
go build -gcflags='all=-N -l' -o bin/app ./cmd/app
# (no -ldflags strip)
```

For `dlv exec`, build with `-gcflags='all=-N -l'`. For `dlv debug`, Delve sets these automatically.

## Goroutine-Aware Debugging Patterns

### Find the deadlock

```bash
(dlv) goroutines
* Goroutine 1 - User: chan send ./main.go:42 main.producer
  Goroutine 2 - User: chan recv ./main.go:50 main.consumer
  Goroutine 3 - User: sync.Mutex.Lock ./worker.go:30 main.worker
  Goroutine 4 - User: sync.Mutex.Lock ./worker.go:30 main.worker
```

Multiple goroutines blocked on the same mutex — likely deadlock. Walk each:

```bash
(dlv) goroutine 3 bt
0  runtime.gopark
1  sync.runtime_SemacquireMutex
2  sync.(*Mutex).Lock
3  main.worker  ./worker.go:30
4  main.dispatch ./main.go:80
```

Goroutine 3 is waiting on mutex inside `worker`, called from `dispatch`. Same for 4 — they're contending. Add another level: who holds the lock? Find a goroutine *not* in `Lock` that has the mutex.

### Where is goroutine N stuck

```bash
(dlv) goroutine 17 bt -full
```

`-full` includes locals at each frame. See exactly what data the goroutine is sitting on.

### Channel inspection

```bash
(dlv) p myChan
chan int 0x0c000018040 {
    qcount: 0,
    dataqsiz: 5,
    elemsize: 8,
    closed: 0,
    sendq: waitq<int> {first: 0xc000040000, last: 0xc000040000},
    recvq: waitq<int> {first: 0x0, last: 0x0},
}
```

Fields:

- `qcount` — current items in buffer
- `dataqsiz` — buffer capacity
- `closed` — 0 open, 1 closed
- `sendq` — goroutines blocked on send
- `recvq` — goroutines blocked on recv

`sendq.first != 0` and `qcount == dataqsiz` → goroutines blocked because buffer is full.
`recvq.first != 0` and `qcount == 0` → goroutines blocked waiting for data.

### Mutex contention

```bash
(dlv) p myMutex
sync.Mutex {
    state: 1,    # 1 = locked
    sema: 0,
}
```

`state` bits: `1` locked, `2` woken, `4` starving. Anything > 0 means held or contended.

For RWMutex:

```bash
(dlv) p myRWMutex
sync.RWMutex {
    w: sync.Mutex {state: 0, sema: 0},
    writerSem: 0,
    readerSem: 0,
    readerCount: 3,    # 3 readers active
    readerWait: 0,
}
```

### Sanity check

```bash
(dlv) p runtime.NumGoroutine()
247
```

Way more than expected? Goroutine leak — find the source by listing and filtering:

```bash
(dlv) goroutines -with user main.leaky
```

## Tracing All Calls to a Function

When you need to see every invocation of a function (e.g., an HTTP handler called 1000x/sec):

```bash
(dlv) trace github.com/myorg/myrepo/internal/handler.ServeHTTP
Tracepoint 1 set at 0x402345 for handler.ServeHTTP() ./handler.go:42
(dlv) c
> handler.ServeHTTP(w=..., r=...) ./handler.go:42 (hits goroutine(7):1 total:1)
> handler.ServeHTTP(w=..., r=...) ./handler.go:42 (hits goroutine(8):1 total:2)
> handler.ServeHTTP(w=..., r=...) ./handler.go:42 (hits goroutine(7):2 total:3)
...
```

Now combine with conditions to narrow:

```bash
(dlv) cond 1 r.URL.Path == "/api/v1/users"
```

And add output:

```bash
(dlv) on 1 print r.Method
(dlv) on 1 print r.RemoteAddr
```

Result: every call to `/api/v1/users` prints method and remote addr without stopping.

To turn off:

```bash
(dlv) clear 1
```

## Conditional Breakpoint Patterns

### Break only after N hits

```bash
(dlv) b main.process
Breakpoint 1 set at 0x405678
(dlv) cond 1 runtime.Caller(0)
# this won't work - cond is more limited
```

Better: track via expression:

```bash
(dlv) b main.process
(dlv) cond 1 i > 100
```

Or set hit count:

```bash
(dlv) condhit 1 100
# break only on the 100th hit
```

### Break in specific goroutine

```bash
(dlv) b main.process
(dlv) cond 1 runtime.curg.goid == 17
```

### Break only in error path

```bash
(dlv) b main.process
(dlv) cond 1 err != nil
```

### Break on specific input

```bash
(dlv) b net/http.(*ServeMux).ServeHTTP
(dlv) cond 1 r.URL.Path == "/api/v1/users" && r.Method == "POST"
```

The expression is full Go — methods, field access, comparisons.

## Compile-Time Optimization Issues

### "could not find symbol value for X"

```bash
(dlv) p myVar
Command failed: could not find symbol value for myVar
```

Cause: compiler optimized the variable away — never stored in memory, lives only in a register that's been clobbered.

Fix: rebuild with `-gcflags='all=-N -l'`:

```bash
go build -gcflags='all=-N -l' -o bin/app ./cmd/app
dlv exec ./bin/app
```

Or use `dlv debug` (sets these flags automatically).

### Inlined functions don't appear

```bash
(dlv) bt
0  main.process ./main.go:42
1  main.main    ./main.go:15
```

Expected to see `main.helper` between them, but it was inlined. Same fix: `-gcflags='all=-N -l'`.

### Variables show wrong values

The compiler may reuse a register for two variables. The variable name in the debug info points at a stale value.

Fix: same `-gcflags='all=-N -l'`.

### Why doesn't dlv know about X

99% of the time: optimization. Rebuild without optimization. The remaining 1%: build with `-trimpath` (strips paths) or stripped binary (`-ldflags='-s -w'`) — Delve has nothing to work with.

Diagnose with:

```bash
go build -gcflags='all=-N -l' -trimpath=false -o bin/app ./cmd/app
file bin/app
# should say "not stripped"
```

## Goroutine Names

Go 1.22+ supports labeled goroutines via `runtime/pprof.SetGoroutineLabels`:

```bash
import "runtime/pprof"

ctx := pprof.WithLabels(context.Background(), pprof.Labels(
    "role", "kafka-consumer",
    "topic", "events",
))
pprof.SetGoroutineLabels(ctx)
```

Delve 1.21+ displays labels in `goroutines` output:

```bash
(dlv) goroutines
* Goroutine 1 - User: ./main.go:15 main.main (0x401234)
  Goroutine 17 - labels: [role=kafka-consumer topic=events] - User: ./consumer.go:42 main.consume
```

Filter:

```bash
(dlv) goroutines -l role=kafka-consumer
```

Massively useful for "which goroutine is the kafka consumer" — names are searchable, IDs are not.

## Profiling Integration

Delve does not replace pprof.

- **Delve** — correctness debugging (where is my code wrong)
- **pprof** — performance debugging (where is my code slow)

Canonical handoff:

```bash
# Profile first
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
(pprof) top
(pprof) list MyHotFunc

# Then debug correctness if needed
dlv test -- -test.run TestMyHotFunc
```

For runtime profiling of a live process:

```bash
import _ "net/http/pprof"
http.ListenAndServe(":6060", nil)
# then:
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

Delve and pprof don't conflict — you can attach Delve to a process serving pprof endpoints.

## Remote Debugging Workflow

### Target side

Start the binary under headless Delve:

```bash
dlv exec --headless --listen=:2345 --api-version=2 --accept-multiclient --continue ./bin/app
```

Flags reminder:

- `--headless` — no interactive prompt
- `--listen=:2345` — TCP port
- `--accept-multiclient` — allow editor reconnect
- `--continue` — auto-resume once first client connects (for production-like debugging without pause)

### Client side

```bash
dlv connect target-host:2345
```

Or in VS Code: launch.json with `"mode": "remote"`, `"port": 2345`, `"host": "target-host"`.

### Kubernetes pattern

Pod runs binary under `dlv exec --headless`. Port-forward and connect:

```bash
kubectl port-forward pod/myapp-xyz 2345:2345
dlv connect 127.0.0.1:2345
```

Dockerfile pattern:

```bash
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go build -gcflags='all=-N -l' -o /app ./cmd/app

FROM debian:bookworm-slim
COPY --from=build /app /app
COPY --from=build /go/bin/dlv /dlv
EXPOSE 2345
CMD ["/dlv", "exec", "--headless", "--listen=:2345", \
     "--api-version=2", "--accept-multiclient", "/app"]
```

Production tip: don't ship dlv-enabled containers. Build a separate `myapp:debug` image and only deploy when debugging.

### Source mapping

Remote Delve sees paths from the build environment. Local sources are at a different location. Use `substitute-path`:

```bash
(dlv) config substitute-path /go/src/myapp /home/me/myapp
```

## Common Errors

### "could not launch process: stat ./bin: no such file or directory"

Path typo. Verify:

```bash
ls -la ./bin/app
dlv exec ./bin/app
```

### "could not find symbol value for X"

Optimization stripped the variable. Rebuild:

```bash
go build -gcflags='all=-N -l' -o bin/app ./cmd/app
```

Or use `dlv debug` instead of `dlv exec`.

### "process X has exited with status N"

Program ended. To re-run:

```bash
(dlv) restart
(dlv) c
```

For `dlv attach`, you can't restart — the process is gone.

### "Internal debugger error: invalid recursive call to dlvbreak"

Delve hit a panic in itself. Bug. File at `github.com/go-delve/delve/issues`. Workarounds:

- Update to latest: `go install github.com/go-delve/delve/cmd/dlv@latest`
- Try without `-gcflags='all=-N -l'` (some Delve bugs only fire with debug info)
- Use a different breakpoint location

### "could not attach to pid X: this could be caused by a kernel security setting, try writing 0 to /proc/sys/kernel/yama/ptrace_scope"

Linux ptrace restriction. Fix:

```bash
sudo sysctl kernel.yama.ptrace_scope=0
# or
sudo dlv attach $PID
```

To make permanent:

```bash
echo "kernel.yama.ptrace_scope=0" | sudo tee /etc/sysctl.d/10-ptrace.conf
sudo sysctl -p /etc/sysctl.d/10-ptrace.conf
```

### "could not attach to pid X: operation not permitted"

Permissions. On Linux: `sudo`. On macOS: codesign:

```bash
codesign -s dlv-cert $(which dlv)
```

See Setup section.

### "no debug info found in binary"

Binary stripped. Rebuild without `-ldflags='-s -w'`:

```bash
go build -gcflags='all=-N -l' -o bin/app ./cmd/app
# (no -s -w)
```

Or:

```bash
file bin/app
# Should NOT say "stripped"
```

### "could not parse breakpoint location: cannot find file 'main.go'"

Working directory mismatch. Use absolute path:

```bash
(dlv) b /home/me/myrepo/main.go:42
```

Or `cd` to the project before launching `dlv`.

### "process spawn failed: exec format error"

Architecture mismatch — binary built for different OS/arch. Rebuild:

```bash
GOOS=linux GOARCH=amd64 go build -gcflags='all=-N -l' -o bin/app ./cmd/app
```

### "wait: waitid: no child processes"

`dlv` lost track of child. Restart Delve.

### "could not connect to debugger: dial tcp 127.0.0.1:2345: connect: connection refused"

Headless server not running. Verify:

```bash
ss -tlnp | grep 2345
```

Or check the process is alive.

## Common Gotchas

### Variables disappear

Bad:

```bash
go build -o bin/app ./cmd/app
dlv exec ./bin/app
(dlv) p myVar
Command failed: could not find symbol value for myVar
```

Fixed:

```bash
go build -gcflags='all=-N -l' -o bin/app ./cmd/app
dlv exec ./bin/app
(dlv) p myVar
"actual value"
```

### dlv attach denied without permission

Bad (Linux):

```bash
dlv attach 12345
could not attach to pid 12345: operation not permitted
```

Fixed:

```bash
sudo sysctl kernel.yama.ptrace_scope=0
dlv attach 12345
# or
sudo dlv attach 12345
```

Bad (macOS):

```bash
dlv attach 12345
could not attach to pid 12345: operation not permitted
```

Fixed:

```bash
codesign -s dlv-cert $(which dlv)
dlv attach 12345
```

### main.main not stopping

Bad: setting `b main.main` and not seeing it hit, because Go runs `runtime.main` first which initializes the runtime, then calls `main.main`.

```bash
(dlv) b main.main
(dlv) c
```

If that doesn't stop, Go's `init()` functions ran and there was an early exit. Set earlier:

```bash
(dlv) b runtime.main
(dlv) c
```

### Threads vs goroutines

Bad:

```bash
(dlv) threads
# only see 4 threads, but you have hundreds of goroutines
```

Fixed:

```bash
(dlv) goroutines
# see all goroutines, including blocked ones
```

Goroutines are M:N over threads. Threads = OS-level kernel threads. Goroutines = Go-level concurrency. For Go debugging, use `goroutines`. Use `threads` only when debugging cgo or the Go runtime itself.

### Breakpoint not hit in goroutine

Bad: setting a breakpoint, the goroutine exists, but Delve never stops there.

```bash
(dlv) b main.worker
(dlv) c
# program runs forever, never breaks
```

Fixed: the goroutine isn't currently scheduled — the M holding it might be in syscall. Pause:

```bash
^C  # Ctrl-C
(dlv) goroutines -with user main.worker
* Goroutine 17 - User: chan recv ./worker.go:42 main.worker
(dlv) goroutine 17
(dlv) bt
```

Or, the goroutine launched once and exited; the breakpoint will never hit again unless a new one starts.

### Function call from Delve hangs

Bad:

```bash
(dlv) call myBlockingFunc()
# hangs forever
```

`call` runs the function on the current goroutine. If the function blocks on something the current goroutine controls (like a channel that you're holding the only sender for), it deadlocks.

Fixed: don't call blocking functions from Delve. Or call them on a different goroutine via reflection (advanced).

### Headless with no `--accept-multiclient` disconnects on quit

Bad:

```bash
dlv exec --headless --listen=:2345 ./app
dlv connect :2345
(dlv) quit
# server exits — can't reconnect
```

Fixed:

```bash
dlv exec --headless --listen=:2345 --accept-multiclient ./app
dlv connect :2345
(dlv) disconnect
# server keeps running; reconnect later
```

Use `disconnect` (not `quit`) to leave without killing the server.

### `step` into runtime hell

Bad:

```bash
(dlv) s
# steps into runtime.morestack
(dlv) s
# steps into runtime.gosched
# ... endless runtime stepping
```

Fixed: use `next` (`n`) to step over, or set a breakpoint in user code and `c`:

```bash
(dlv) b main.userFunc
(dlv) c
```

Configure to skip the runtime:

```bash
(dlv) config skip-recur 1
```

### Debugging release builds

Bad: try to debug a stripped, optimized release binary.

```bash
go build -ldflags='-s -w' -o bin/app ./cmd/app
dlv exec ./bin/app
no debug info found in binary
```

Fixed: maintain a debug build alongside:

```bash
go build -gcflags='all=-N -l' -o bin/app-debug ./cmd/app
go build -ldflags='-s -w' -o bin/app ./cmd/app
# debug bin/app-debug, deploy bin/app
```

### Path mismatch on remote

Bad: remote built in `/build/path`, local sources in `/home/me/`.

```bash
(dlv) list main.main
Command failed: open /build/path/main.go: no such file
```

Fixed:

```bash
(dlv) config substitute-path /build/path /home/me/myrepo
(dlv) list main.main
```

### Conditional breakpoint with broken expression

Bad:

```bash
(dlv) cond 1 myMap[somekey] == "x"
# expression evaluates wrong because of nil maps
```

Fixed: guard:

```bash
(dlv) cond 1 myMap != nil && myMap[somekey] == "x"
```

## macOS Specifics

### Codesign

```bash
codesign -s dlv-cert $(which dlv)
```

`dlv-cert` is a self-signed cert from Keychain Access (see Setup).

### LLDB on macOS

LLDB has slightly better macOS-native integration but worse Go semantics. The canonical answer:

- macOS native binary, non-Go → LLDB
- Go on macOS → Delve (codesigned)

### SIP and `dlv attach`

System Integrity Protection blocks ptrace on system binaries. `dlv attach $(pgrep ssh)` will fail. Only attach to processes you built.

### Apple Silicon

Delve supports `darwin/arm64` (M1/M2/M3) since 1.7.0. `go env GOARCH` should show `arm64`. Cross-debugging is tricky:

```bash
# Built on amd64, debug on arm64?
GOOS=darwin GOARCH=arm64 go build -gcflags='all=-N -l' -o bin/app ./cmd/app
```

## Linux Permissions

### ptrace_scope

```bash
cat /proc/sys/kernel/yama/ptrace_scope
```

Values:

- `0` — anyone can ptrace anything they own
- `1` — only parent processes (default; `dlv debug` works, `dlv attach` doesn't)
- `2` — only `cap_sys_ptrace` (root or capability)
- `3` — no ptrace at all

To loosen:

```bash
sudo sysctl kernel.yama.ptrace_scope=0
```

Permanent:

```bash
echo "kernel.yama.ptrace_scope=0" | sudo tee /etc/sysctl.d/10-ptrace.conf
sudo sysctl --system
```

Or simpler: `sudo dlv attach $PID`.

### YAMA LSM

Yama is the Linux Security Module that enforces `ptrace_scope`. Some distros disable it; check:

```bash
ls /sys/module/yama
# if missing, yama isn't loaded - no restriction
```

### capabilities

Granting `CAP_SYS_PTRACE` to dlv:

```bash
sudo setcap cap_sys_ptrace=eip $(which dlv)
```

Now `dlv attach` works without sudo. Caveat: any user can now use `dlv` to attach to any process they own that doesn't drop privs.

### container debugging

Inside Docker:

```bash
docker run --cap-add SYS_PTRACE --security-opt seccomp=unconfined ...
```

Without these, ptrace is blocked by Docker's default seccomp profile.

In Kubernetes, you need `securityContext.capabilities.add: ["SYS_PTRACE"]` and possibly `securityContext.privileged: true` depending on the cluster's PSP.

## Idioms

### The canonical workflow

```bash
# 1. Build with debug flags
go build -gcflags='all=-N -l' -o bin/app ./cmd/app

# 2. Reproduce the bug under Delve
dlv exec ./bin/app -- --flag=value

# 3. Set breakpoint at suspect location
(dlv) b ./internal/foo/bar.go:42

# 4. Run, hit breakpoint, inspect
(dlv) c
(dlv) locals
(dlv) p myVar
```

### Trace + filter for high-rate

```bash
(dlv) trace github.com/myorg/myapp/handler.ServeHTTP
(dlv) cond 1 r.URL.Path == "/admin/users" && r.Method == "POST"
(dlv) on 1 print r.Header
(dlv) c
```

Captures only matching requests with full header dump.

### Goroutine-list-then-pick for deadlock

```bash
^C  # pause running program
(dlv) goroutines
# scan output, look for many in same lock or chan op
(dlv) goroutine N bt -full
# inspect full stack of suspect goroutine
```

### Sanity check goroutine count

```bash
(dlv) p runtime.NumGoroutine()
1247
```

If unexpectedly high → goroutine leak. Find:

```bash
(dlv) goroutines -with user main.suspicious
```

### pprof for performance, dlv for correctness

If your code is slow, `pprof`. If your code is wrong, `dlv`. Don't use the wrong tool.

### Save a session

```bash
dlv exec ./bin/app --log --log-output=debugger,rpc 2>session.log
```

The `--log` flag with comma-sep outputs preserves all RPC traffic. Replay later for diagnosis.

### Read transcript

```bash
(dlv) help
(dlv) help break
(dlv) help goroutines
```

Built-in help is exhaustive — better than guessing.

### Quickly view current stack

```bash
(dlv) bt 5
# only top 5 frames
(dlv) bt -full 3
# top 3 with locals
```

### Set a breakpoint while paused at one

```bash
(dlv) b main.process
(dlv) c
> main.process ./main.go:42
(dlv) b main.cleanup
(dlv) c
> main.cleanup ./main.go:80
```

You can manage breakpoints any time the process is paused.

### Examine all goroutines stuck on a specific call

```bash
(dlv) goroutines -with user runtime.gopark
# all blocked goroutines
(dlv) goroutines -with user sync.runtime_SemacquireMutex
# all goroutines waiting on mutexes
(dlv) goroutines -with user runtime.chanrecv
# all blocked on channel recv
```

### See what each thread is doing

```bash
(dlv) threads
# concise thread list
(dlv) thread N
(dlv) bt
# stack of that thread
```

Only useful when investigating cgo or runtime issues.

### Stop on next panic

```bash
(dlv) b runtime.gopanic
(dlv) c
# next panic stops here
(dlv) bt
# user code is up the stack
```

### Print into a slice

```bash
(dlv) config max-array-values 1000
(dlv) p mySlice
[1000-element slice]
```

Default is 64; bump for full inspection.

### Print deeply nested

```bash
(dlv) config max-variable-recurse 5
(dlv) p deeplyNestedStruct
# recurses 5 levels
```

Default is 1.

### Function calls in expressions

```bash
(dlv) call myFunc(42)
> Goroutine 1 returned from call
return: "result"
```

Prerequisites:

- Stopped at a safe-point (most line-level breakpoints qualify)
- Function doesn't block on the calling goroutine
- Build with `-gcflags='all=-N -l'`

Risky — can corrupt program state. Use sparingly.

### Modify a variable

```bash
(dlv) set i = 100
(dlv) set myStruct.Field = "newval"
```

Live modification. Useful for forcing a specific code path. Doesn't work for all types (slices, maps require careful handling).

## Tips

- Always build with `-gcflags='all=-N -l'` for debugging. Make a debug build target:

```bash
debug:
        go build -gcflags='all=-N -l' -o bin/app-debug ./cmd/app
```

- Use `dlv test` for test-specific debugging — it auto-applies the right flags.

- Keep a `.dlvrc` with your aliases and common config. Saves typing every session.

- For long sessions, set `max-string-len`, `max-array-values`, `max-variable-recurse` higher than defaults — defaults are conservative.

- Use `display` to watch a variable across multiple stops without re-typing `p`.

- Use `trace` instead of `b` when you want to see something fire many times without manual `c`.

- For panics, set `b runtime.gopanic` early in the session.

- For goroutine leaks, `p runtime.NumGoroutine()` regularly.

- For deadlocks, Ctrl-C and `goroutines` — patterns reveal themselves.

- Headless mode (`--headless --accept-multiclient`) for editor integration; reconnect freely.

- macOS: codesign once, forget about it.

- Linux: `sudo dlv attach` is fine for one-off debugging; `setcap cap_sys_ptrace` for repeated use.

- Substitute paths when debugging remote or container builds.

- Use `dlv core` for crashed programs — Go's `GOTRACEBACK=crash` produces useful cores.

- Don't use `step` into the runtime; use `next` and set user-code breakpoints.

- Panic in a test? `dlv test` then `c` to reproduce, `bt` and `goroutines` to inspect.

- Function call (`call`) is a foot gun. Use sparingly and never on blocking calls.

- Update Delve when Go updates. Release notes pair them.

- For VS Code, prefer `dlv-dap` over legacy `dlv` mode.

- For DAP debugging in CI containers, ensure `--cap-add SYS_PTRACE`.

- Use goroutine labels (`pprof.SetGoroutineLabels`) liberally — names are searchable.

- Build the dev binary with `-trimpath=false` so paths resolve. Strip in production only.

- Track Delve's own changelog: new features (watchpoints, labels, DAP improvements) ship regularly.

- The `(dlv) help` output is the source of truth — RTFM beats guessing.

- For protocol-level debugging, run `--log --log-output=debugger,rpc,gdbwire` and inspect the transcript.

- For headless Kubernetes, use `--continue` so the binary runs immediately on connect.

- When all else fails: smaller reproducer, simpler binary, fewer moving parts. Delve is good but it's not magic.

## See Also

- gdb — generic debugger; weaker on Go
- lldb — better on macOS for non-Go
- pdb — Python equivalent
- perf — Linux performance profiler
- polyglot — when debugging across language boundaries
- bash — shell for invoking dlv

## References

- github.com/go-delve/delve — source and issue tracker
- github.com/go-delve/delve/tree/master/Documentation — official docs
- github.com/go-delve/delve/blob/master/Documentation/cli/README.md — CLI reference
- github.com/go-delve/delve/blob/master/Documentation/api/json-rpc/README.md — JSON-RPC API
- github.com/go-delve/delve/blob/master/Documentation/api/dap/README.md — DAP support
- github.com/go-delve/delve/blob/master/Documentation/usage/dlv.md — `dlv` command reference
- github.com/go-delve/delve/blob/master/Documentation/faq.md — common questions
- pkg.go.dev/runtime/pprof — goroutine labels
- microsoft.github.io/debug-adapter-protocol — DAP spec
- github.com/golang/vscode-go — VS Code Go extension
- github.com/leoluk/perflock — useful adjunct for benchmarking
- "The Go Programming Language" by Donovan and Kernighan — chapter on debugging
- talks.golang.org — many Delve talks
- `dlv help` — built-in help, exhaustive
- `dlv help <command>` — per-command help
- `man dlv` — if installed via package manager
