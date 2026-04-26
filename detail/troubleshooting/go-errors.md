# Go Errors — Runtime, Panic, and Error-Interface Internals

A deep dive into Go's two error mechanisms: the explicit `error` interface and the exceptional `panic`/`recover` flow. This is not a catalog of error messages — for that, see `sheets/troubleshooting/go-errors.md`. This document is about the runtime and the design rationale.

## Setup

Go has two error mechanisms, deliberately separated:

1. **`error` interface** — the *expected* failure path. Returned alongside other return values; the caller decides how to handle it. Used for "this might not work" cases: I/O, parsing, network, etc.
2. **`panic` / `recover`** — the *exceptional* path. For programmer errors (out-of-bounds, nil deref) and unrecoverable runtime conditions. Should rarely be caught.

The split is deliberate: making errors explicit return values forces the caller to handle them, but making panic separate prevents the trap of every function being potentially exception-raising. Most Go code never uses `panic`/`recover`; library boundaries should not propagate panics.

## The error Interface

```go
type error interface {
    Error() string
}
```

That's it. One method, returns a string. The minimalism is intentional:

- Any type can be an error by implementing `Error()`.
- `nil` is a valid `error` value (the zero value of the interface).
- The returned string is the canonical human-readable description.

Construction:

```go
errors.New("file not found")
fmt.Errorf("invalid input %q", s)
fmt.Errorf("upload failed: %w", err)  // wrapping
```

`errors.New` returns a `*errors.errorString` containing the message. Two calls with the same string return distinct values — `errors.New("x") != errors.New("x")`. This is why sentinel errors are package-level vars (one allocation, comparable identity).

## errors.Is / errors.As / errors.Unwrap (1.13+)

Pre-1.13, error inspection was string comparison or type switches. Both are fragile. 1.13 added structured chaining:

```go
errors.Is(err, target)   // does err or any wrapped error == target?
errors.As(err, &target)  // does err or any wrapped error match target's type?
errors.Unwrap(err)       // get the immediate wrapped error, or nil
```

`Unwrap` returns the next link in the chain. Implementations:

```go
type wrapError struct {
    msg string
    err error
}

func (e *wrapError) Error() string { return e.msg }
func (e *wrapError) Unwrap() error { return e.err }
```

`errors.Is` walks `Unwrap()` chain calling `==` at each level (or calling `target.Is(err)` if target implements `Is`). `errors.As` walks calling `errors.As` semantics (interface check) at each level.

Properties:

- `errors.Is(err, nil)` returns `err == nil` — supports nil cleanly.
- `errors.As(nil, &x)` returns false; doesn't panic.
- The chain is followed via `Unwrap()`. If a custom error doesn't implement `Unwrap`, the chain ends there.

```go
var ErrNotFound = errors.New("not found")

func find() error {
    return fmt.Errorf("lookup user: %w", ErrNotFound)
}

err := find()
fmt.Println(errors.Is(err, ErrNotFound))  // true
```

## fmt.Errorf %w Wrapping

```go
fmt.Errorf("context: %w", err)
```

The `%w` verb produces an error that wraps `err`. Implementation: `fmt.Errorf` returns a `*fmt.wrapError` with `msg` (the formatted string) and `err` (the wrapped error). `Unwrap` returns `err`.

Multiple `%w` (1.20+): `fmt.Errorf("a: %w, b: %w", err1, err2)` — returns an error wrapping both. `Unwrap` returns `[]error{err1, err2}`.

`%s` and `%v` produce the same string but *don't* preserve the chain — the returned error has no `Unwrap`. So:

```go
// Loses chain — caller can't use errors.Is/As to detect ErrNotFound
return fmt.Errorf("lookup: %s", err)

// Preserves chain
return fmt.Errorf("lookup: %w", err)
```

The distinction matters for libraries: if you want callers to be able to detect specific underlying errors, wrap with `%w`. If you want to opaque-ize (deliberately hide the underlying error), use `%s`.

## errors.Join (1.20+)

```go
err1 := io.ErrClosedPipe
err2 := os.ErrPermission
joined := errors.Join(err1, err2, nil)
// joined is an error whose Error() is err1's + "\n" + err2's
// nils are skipped; if all nil, returns nil
```

`Unwrap` (the multi-error variant) returns `[]error{err1, err2}`. `errors.Is` and `errors.As` walk all branches, so:

```go
errors.Is(joined, io.ErrClosedPipe)  // true
errors.Is(joined, os.ErrPermission)  // true
```

Useful for cleanup paths:

```go
func (m *Manager) Close() error {
    var errs []error
    for _, c := range m.closers {
        if err := c.Close(); err != nil {
            errs = append(errs, err)
        }
    }
    return errors.Join(errs...)
}
```

This pattern was previously hand-rolled with `multierror` libraries (HashiCorp's `go-multierror`). 1.20 made it standard.

## Sentinel Errors

```go
package mypkg

import "errors"

var ErrNotFound = errors.New("not found")
var ErrTimeout  = errors.New("timeout")
```

Properties:

- Allocated once. Pointer comparison is exact identity.
- Public API: callers can `errors.Is(err, mypkg.ErrNotFound)` to detect.
- Exhaustive: a package's sentinels form its public error API.

Limits:

- No per-call-site context. Same `ErrNotFound` for every miss.
- Wrap to add context: `return fmt.Errorf("user %d: %w", id, ErrNotFound)`.

The standard library is full of sentinels: `io.EOF`, `io.ErrUnexpectedEOF`, `sql.ErrNoRows`, `http.ErrServerClosed`, `context.Canceled`, `context.DeadlineExceeded`. Comparing with `==` is correct only when no wrapping is involved; use `errors.Is` for safety.

## Error Types vs Sentinel

When sentinel isn't enough — you need fields on the error — define a struct:

```go
type ParseError struct {
    Line int
    Col  int
    Msg  string
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("line %d:%d: %s", e.Line, e.Col, e.Msg)
}
```

Caller:

```go
var pe *ParseError
if errors.As(err, &pe) {
    fmt.Println("error at line", pe.Line)
}
```

Trade-offs:

- **Sentinel:** simple, comparable, no fields. Good for "category" errors.
- **Type:** carries context. Caller can extract structured data. Forms larger public API surface (every field is documented behavior).

A common pattern is to have *both*:

```go
var ErrParseFailed = errors.New("parse failed")

type ParseError struct {
    Line int
    Col  int
}

func (e *ParseError) Error() string { return fmt.Sprintf("line %d:%d", e.Line, e.Col) }
func (e *ParseError) Is(t error) bool { return t == ErrParseFailed }
```

Now `errors.Is(err, ErrParseFailed)` works *and* `errors.As(err, &pe)` extracts fields.

## Panic Internals

`runtime.gopanic` (in `src/runtime/panic.go`) is called when:

- Explicit `panic(v)`.
- Runtime checks fail (nil deref, divide-by-zero, slice OOB, type assertion failure, send on closed channel, etc.).

Pseudocode:

```go
func gopanic(v interface{}) {
    p := &_panic{arg: v, link: gp._panic}
    gp._panic = p

    for {
        d := gp._defer
        if d == nil { break }
        // run deferred function
        gp._defer = d.link
        reflectcall(...)
        if recovered { return /* via deferreturn */ }
    }
    // no recover — print stack trace and exit
    fatalpanic(p)
}
```

Each goroutine has a `_panic` linked-list (panics can nest if a deferred function itself panics) and a `_defer` linked-list. `gopanic` walks deferreds in LIFO order, executing each. If any calls `recover()`, the panic is consumed and execution resumes after the deferred function's caller.

The `recover()` returns the `panic` argument. After return, the `gopanic` returns normally (rather than calling `fatalpanic`), and execution continues at the deferred function's return site.

If no defer calls `recover()`, `fatalpanic` walks all goroutines, prints stacks of each, and exits with status 2.

## Defer Mechanism

```go
defer fmt.Println("done")
```

Compiler generates a call to `runtime.deferproc`:

```go
func deferproc(siz int32, fn *funcval) {
    d := newdefer()
    d.fn = fn
    d.sp = getsp()
    d.pc = getcallerpc()
    d.link = gp._defer
    gp._defer = d
}
```

A `_defer` struct is allocated, the function pointer recorded, and the struct linked into the goroutine's defer chain. At function return, `runtime.deferreturn` walks the chain.

In Go 1.14+, the compiler optimizes "open-coded" defers: if a function has at most 8 defers and they're not in loops, the deferred call is generated inline and the heap allocation skipped. This reduces defer overhead from ~50ns to ~1-2ns.

The famous loop-capture gotcha (pre-1.22):

```go
for _, v := range items {
    defer fmt.Println(v)  // all defers captured the same v
}
// Pre-1.22: prints last v, n times
// 1.22+: prints each v correctly (loop-var-per-iter change)
```

Same gotcha for goroutines. 1.22 fixed it by giving each iteration its own variable. Pre-1.22 idiom: `v := v` inside the loop.

## Recover Semantics

```go
defer func() {
    if r := recover(); r != nil {
        log.Printf("recovered: %v", r)
    }
}()
```

`recover()` is *only* meaningful when called inside a deferred function. Outside a deferred function (or outside a panic), it returns `nil`.

The check:

- `recover()` looks at `gp._panic`.
- If non-nil and the panic hasn't been recovered yet, marks it recovered and returns the panic value.
- Otherwise returns `nil`.

The panic value is `interface{}`. Type-assert to inspect:

```go
defer func() {
    if r := recover(); r != nil {
        switch v := r.(type) {
        case error:
            log.Printf("error panic: %v", v)
        case string:
            log.Printf("string panic: %s", v)
        default:
            log.Printf("unknown panic: %v", v)
        }
    }
}()
```

`panic(nil)` was historically a special case — `recover()` returned `nil` even though a panic occurred, making it impossible to detect. Go 1.21 changed this: `panic(nil)` now panics with a `*runtime.PanicNilError`, and `recover()` returns that. For backward compat, set `GODEBUG=panicnil=1` to restore old behavior.

## Stack Unwinding

When `gopanic` walks the defer chain and no `recover` fires, control reaches `fatalpanic` which prints stacks. The stacks are walked using `runtime.Callers` which uses architecture-specific link-register or return-address conventions:

- **AMD64:** return address pushed on stack by `CALL`. Walker reads `*rbp`, then `*(rbp+8)` for the return address, recursing.
- **ARM64:** link register `LR` holds the return address; saved to stack on function prologue. Walker reads `LR` of current frame, then walks via frame pointer.

The runtime uses a `_func` table (built into the binary) to map PC values to function info: name, source file, line, parameter info. `runtime.FuncForPC(pc).Name()` is the public API.

For each goroutine, the runtime knows:

- Status (`_Grunning`, `_Gwaiting`, etc.).
- Wait reason (channel receive, mutex lock, GC assist, etc.).
- Stack PC and SP.

`runtime.Stack(buf, all=true)` walks `runtime.allgs` and dumps each. SIGQUIT (default Ctrl+\) triggers this on a live process.

## The runtime Package's Panic List

These types implement `error` and may appear as the panic value:

- `runtime.errorString` — generic wrapped string.
- `runtime.TypeAssertionError` — fields `_interface`, `concrete`, `asserted`, `missingMethod`.
- `runtime.divideError` — division by zero.
- `runtime.boundsError` — slice/array index OOB. Has `x` (index), `y` (length), `code`.
- `runtime.PanicNilError` — `panic(nil)` (1.21+).

The interface `runtime.Error` extends `error`:

```go
type Error interface {
    error
    RuntimeError()  // marker method
}
```

So you can detect runtime panics specifically:

```go
defer func() {
    r := recover()
    if r == nil { return }
    if _, ok := r.(runtime.Error); ok {
        // it's a runtime panic (nil deref, etc.)
        log.Print("runtime panic")
    } else {
        // it's a deliberate panic(value)
    }
}()
```

## Goroutine Safety + Panic

A panic in goroutine A does not propagate to goroutine B. If goroutine A doesn't recover, the *whole process* exits.

```go
go func() {
    panic("boom")  // crashes the entire program
}()
```

There's no main goroutine "owner" of child goroutines. The runtime treats them as peers; any unrecovered panic terminates everything.

The pattern: every goroutine started by your code should recover at the top:

```go
func safeGo(fn func()) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("goroutine panic: %v\n%s", r, debug.Stack())
            }
        }()
        fn()
    }()
}
```

For HTTP servers, the stdlib `net/http` server already does this per-request. Your handlers can panic and only that connection is closed; the server continues. But goroutines you spawn yourself (background workers) need their own recover.

## Runtime Race Detector

Build with `-race`:

```bash
go build -race ./...
go test -race ./...
```

Implementation:

- `runtime/race` package wraps Google's ThreadSanitizer (TSan) C library.
- Compiler instruments every memory access (`MOVQ` becomes a TSan-tracked access).
- TSan tracks happens-before relationships via vector clocks.
- On detected race, prints WARNING with both stacks.

Cost:

- ~5-10x slower runtime.
- ~2x memory overhead.
- Binary ~2x larger.

Don't ship `-race` to production; use it in CI and development.

False positives are extremely rare; TSan's algorithm is sound (no false alarms) but incomplete (may miss races that didn't happen this run). If `-race` reports something, it's real.

## Race Detector Output Format

```text
WARNING: DATA RACE
Read at 0x00c000094090 by goroutine 7:
  main.reader()
      /home/x/main.go:15 +0x44

Previous write at 0x00c000094090 by main goroutine:
  main.main()
      /home/x/main.go:9 +0x68

Goroutine 7 (running) created at:
  main.main()
      /home/x/main.go:11 +0x53
```

Three sections:

1. The conflicting access (read or write).
2. The previous conflicting access (write — there must be at least one write for it to be a race).
3. Where the racing goroutine was created (helps trace lifecycle).

The address is the literal memory location, useful for tracking which variable. Match against your binary's symbol table if needed: `go tool nm ./binary | grep variable`.

## The runtime Type Checker

Interface values in Go are two words: a type pointer and a data pointer.

```go
type iface struct {
    tab  *itab    // pointer to interface table
    data unsafe.Pointer
}
```

The `itab` records the concrete type and the methods it satisfies for the interface. Type assertions check the `itab`:

```go
v, ok := x.(*MyType)
```

Compiles to:

```go
itab_check(x.tab, MyType_typeptr) -> ok
if ok { v = x.data }
```

If the assertion fails and the form is `v := x.(*MyType)` (no `ok`), a `runtime.TypeAssertionError` panic is raised. With `, ok` form, no panic — `ok` is false.

`reflect.TypeOf(v)` accesses the `_type` pointer; `reflect.ValueOf(v)` wraps the data pointer. The cost of `reflect` calls is mainly the allocation of the `reflect.Value` struct and the indirection through `_type` metadata.

## Generics (1.18+) Errors

Type parameter inference happens at compile time. Common errors:

```text
cannot infer T
```

The compiler can't figure out the type parameter from arguments:

```go
func Identity[T any](x T) T { return x }

Identity(5)        // T inferred as int
Identity[int](5)   // explicit
```

```text
T does not satisfy comparable
```

A constraint isn't met:

```go
func Eq[T comparable](a, b T) bool { return a == b }

type S struct { F func() }
Eq(S{}, S{})  // ERROR: S contains func, not comparable
```

```text
T does not satisfy ~string | ~int
```

Type set constraint not met. The `~` means "underlying type"; without `~`, only the exact type matches.

These are pure compile-time errors; runtime doesn't see them.

## The cgo Runtime Bridge

Go's runtime is built around the assumption that goroutines run on its scheduler. C code doesn't know about goroutines. The cgo bridge:

1. On `C.foo()`: Go captures registers, switches to a system stack (separate from goroutine stack), enters C.
2. C runs as long as it wants; Go scheduler can't preempt it.
3. On C return: Go restores registers, returns to goroutine.

Signals: by default, the Go runtime installs handlers for SIGSEGV, SIGBUS, SIGFPE, SIGURG, etc. C libraries that install their own handlers can break this. Go provides `os/signal.Notify` for cooperation but cgo-heavy programs may need explicit handler management.

Panicking out of C is undefined behavior. If C calls back to a Go function, and that Go function panics, the panic propagates back through cgo only if the runtime can unwind C frames — which it can on most platforms via DWARF unwind info, but it's fragile. Best practice: don't panic across cgo boundaries; convert to `error` returns at the boundary.

## signal Package

```go
import "os/signal"

ch := make(chan os.Signal, 1)
signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
<-ch
log.Println("shutting down")
```

`signal.Notify` registers a forwarder. The runtime's signal handler runs in signal context (very limited; can't allocate, can't acquire locks), captures the signal number, and atomically pushes to a buffered channel.

Internally:

- `runtime/signal_unix.go`'s `sighandler` is the C-level handler.
- It copies signal info to a "signal queue" (per-M lock-free).
- A dedicated "signalwait" goroutine drains and dispatches.

SIGSEGV is special: the runtime catches it, examines the faulting address, and either:

- Treats as nil pointer deref → panic with `runtime.errorString("nil pointer dereference")`.
- Treats as memory boundary violation → panic similarly.
- Forwards to user (e.g., for cgo crash recovery).

SIGURG (signal 23 on Linux) is used internally for goroutine preemption since Go 1.14: when a goroutine has been running too long without yielding, the runtime sends SIGURG to its M, the signal handler sets a preemption flag, and the next safe point yields to the scheduler. This is why `os/signal` doesn't allow you to handle SIGURG.

## Stack Overflow vs Panic

Go uses split stacks: each goroutine starts with a small stack (~2 KB) that grows as needed. The function prologue checks `SP < g.stackguard0`; if so, it calls `runtime.morestack` which allocates a larger stack and copies frames over.

If allocation fails (extremely rare; only on OOM), runtime panics:

```text
runtime: stack overflow
```

This is distinct from infinite recursion: infinite recursion grows the stack until `runtime.maxstacksize` (default 1 GB on 64-bit) is reached, then panics. Set lower with `debug.SetMaxStack`.

`runtime.lessstack` shrinks stacks during GC if a goroutine's stack high-water mark is much smaller than current size — helpful in long-lived goroutines that briefly used deep stacks.

## Memory Errors

Out-of-memory triggers:

```text
runtime: out of memory
```

The runtime's `mheap_.allocSpan` can't get pages from the OS. The panic is unrecoverable — the runtime exits.

`GOMEMLIMIT` (1.19+) sets a soft heap limit:

```bash
GOMEMLIMIT=4GiB ./myprogram
```

The runtime targets keeping live heap below this. GC runs more aggressively as the limit approaches. Goroutines may also assist GC (do work for the collector inline). If the limit is unreachable due to live data exceeding it, the runtime *does not OOM* — it just runs GC continuously, which appears as 100% CPU. Set `GOMEMLIMIT` only when you have a clear understanding of your workload's working set.

`debug.SetMemoryLimit(n)` is the runtime API.

## Goroutine Leaks

A goroutine leak happens when a goroutine blocks forever. Common causes:

- Unbuffered channel send with no receiver.
- Channel receive from a channel never closed and never sent to.
- `sync.WaitGroup.Wait` when one of the workers exits early.
- `context.Context` that is never canceled and the goroutine waits on `ctx.Done()`.

Detection:

- `runtime.NumGoroutine()` — total count. If grows without bound, you have a leak.
- `pprof` goroutine profile:

```go
import _ "net/http/pprof"
http.ListenAndServe(":6060", nil)
```

Then:

```bash
go tool pprof http://localhost:6060/debug/pprof/goroutine
(pprof) top
(pprof) traces  # show all goroutine stacks
```

The profile groups goroutines by stack, so a leak shows up as N copies of the same stack (where N grows over time).

Each goroutine has ~2 KB stack initially; 100k leaked goroutines is 200 MB of memory invisible to the heap profile. Always profile goroutine count alongside heap.

`runtime.allgs` (via `go tool view-allg` or pprof) shows every goroutine's wait reason. "select (no cases)" or "chan receive" with no corresponding send is the classic leak signature.

## The error Wrapping Specification

Pre-1.13, `pkg/errors` (Dave Cheney's library) was the de-facto standard:

```go
errors.Wrap(err, "context")
errors.Cause(err)  // unwrap to root
```

Go 1.13 made wrapping standard but with different semantics:

- Used `%w` in `fmt.Errorf` instead of a `Wrap` function — fits Go's "format strings everywhere" idiom.
- Used `errors.Is`/`errors.As` instead of `Cause` — the design assumption is "you check for specific errors", not "you extract the root".
- The chain has no global root; each error adds its own context.

Why no `Wrap()` function? The proposal team wanted to make wrapping feel like normal error formatting, not a separate concept. `fmt.Errorf("op failed: %w", err)` reads as "the message includes the wrapped error", which is true.

Why `Is`/`As` instead of recursive `Cause`? Because errors can be matched at any level, not just the leaf. An HTTP error wrapping a DNS error wrapping a syscall error: you might want to detect "any DNS issue", which is a middle layer.

The std-library-first principle: the design is meant to be in `errors` and `fmt`, with no external package required. `pkg/errors` is now legacy.

## errgroup vs sync.WaitGroup

```go
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(parentCtx)
g.Go(func() error { return work1(ctx) })
g.Go(func() error { return work2(ctx) })
if err := g.Wait(); err != nil {
    log.Printf("error: %v", err)
}
```

Compared to `sync.WaitGroup`:

- `WaitGroup.Wait()` returns no error. Errors must be collected manually with a mutex or channel.
- `errgroup.Wait()` returns the first error, cancels the context, lets others observe cancellation.

The key feature is the linked context: when any worker returns non-nil, the context is canceled, signaling the others to stop.

`SetLimit(n)` (added later) caps concurrent workers — useful for bounded parallelism.

`errgroup.TryGo` — non-blocking schedule attempt; returns false if at limit.

## Error Inspection in Tests

```go
func TestThing(t *testing.T) {
    err := doThing()
    if !errors.Is(err, ErrExpected) {
        t.Fatalf("got %v, want %v", err, ErrExpected)
    }
}
```

Don't compare `err.Error()` strings — the format may change between versions or after wrapping. The wrapping/unwrapping semantics make string comparison fragile.

For type assertions:

```go
var pe *ParseError
if !errors.As(err, &pe) {
    t.Fatalf("expected ParseError, got %T", err)
}
if pe.Line != 5 {
    t.Errorf("line = %d, want 5", pe.Line)
}
```

`testify`'s `assert.ErrorIs` and `require.ErrorAs` wrap these for cleaner test code, but stdlib alone is sufficient.

## Performance

Cost model:

- **Sentinel error:** zero allocation. Compared by pointer.
- **`errors.New`:** one allocation per call. Avoid in hot paths; use sentinel.
- **`fmt.Errorf` without `%w`:** one allocation for the formatted string + one for the error.
- **`fmt.Errorf` with `%w`:** plus one for the wrap struct.
- **`runtime/debug.Stack`:** allocates ~64 KB buffer. Don't call in hot paths.
- **`recover`:** essentially free if no panic. The check is a load + branch.
- **`panic`:** O(stack depth) — walks frames building defer chain. ~µs per frame. Don't use as control flow.

Returning an error is fast because `error` is a 2-word interface; copying two words is cheap. The cost is in the *creation*, not the return.

For very-hot paths where you must indicate "no result", consider returning `(value, bool)` instead of `(value, error)` — `bool` is cheaper than `error` since no allocation is involved.

## Common Library Patterns

**database/sql:**

```go
err := db.QueryRow("SELECT ...").Scan(&x)
if errors.Is(err, sql.ErrNoRows) {
    // 0 rows returned
}
```

`sql.ErrNoRows` is a sentinel.

**net/http:**

```go
err := server.ListenAndServe()
if !errors.Is(err, http.ErrServerClosed) {
    log.Fatal(err)
}
```

`ErrServerClosed` indicates intentional shutdown via `Server.Shutdown()`. Anything else is a real error.

**io:**

```go
n, err := r.Read(buf)
if err == io.EOF {
    // end of stream — not really an error in most contexts
}
```

`io.EOF` is sentinel and is documented as "use `==`" since `Read` doesn't wrap. Be careful — some `io.Reader` implementations *do* wrap, in which case use `errors.Is`.

**context:**

```go
select {
case <-ctx.Done():
    err := ctx.Err()
    if errors.Is(err, context.Canceled) {
        // explicit cancel
    }
    if errors.Is(err, context.DeadlineExceeded) {
        // timeout
    }
}
```

Two sentinels for the two reasons. Wrap when propagating: `fmt.Errorf("step X: %w", ctx.Err())`.

**os/exec:**

```go
err := cmd.Run()
var ee *exec.ExitError
if errors.As(err, &ee) {
    fmt.Println("exit code:", ee.ExitCode())
    fmt.Println("stderr:", string(ee.Stderr))
}
```

`ExitError` wraps the process state; `As` extracts the typed error.

## Best Practices

**Wrap with `%w` to add context:**

```go
if err := doStep(); err != nil {
    return fmt.Errorf("step failed: %w", err)
}
```

Don't wrap if you're returning to the same caller — they have the same context. Wrap at module/package boundaries to add scope.

**Use sentinel for category, type for context-rich:**

- `var ErrNotFound = errors.New("not found")` — public API; callers detect with `errors.Is`.
- `type *ValidationError struct { Field string }` — caller extracts `Field` with `errors.As`.

Combine when you want both behaviors (see "Error Types vs Sentinel" above).

**Never panic in libraries:**

Library code should return errors. Panicking violates the implicit "I won't crash your program" contract. Exceptions:

- Programmer error in API misuse (e.g., `nil` to a function that documents `must not be nil`). Panic is acceptable here because it's a bug.
- Constructors of "must succeed at startup" types where failure means the program is unusable.

**Recover at goroutine top:**

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("worker panic: %v\n%s", r, debug.Stack())
            metrics.IncCounter("worker_panics")
        }
    }()
    runWorker()
}()
```

Without this, one bug in `runWorker` crashes the whole program. With it, the worker dies, you log it, and the rest of the program continues.

**Don't recover everywhere:**

Recovering inside business logic to "keep going" usually means the program is in an undefined state. Panic typically indicates a bug; suppressing it doesn't fix the bug, just makes it harder to find. Recover at architectural boundaries (request handlers, worker goroutines, background tasks) — places where one unit of work can fail without contaminating others.

**Provide an Is/As when you need it:**

```go
type NotFoundError struct {
    Resource string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found", e.Resource)
}

// Allow errors.Is(err, ErrNotFound)
var ErrNotFound = errors.New("not found")
func (e *NotFoundError) Is(target error) bool {
    return target == ErrNotFound
}
```

Now both `errors.Is(err, ErrNotFound)` (category check) and `errors.As(err, &nfe)` (extract Resource) work.

**Don't compare error strings:**

```go
// WRONG
if err.Error() == "EOF" { ... }

// Right
if errors.Is(err, io.EOF) { ... }
```

**Format errors consistently:**

- Lowercase first letter (`fmt.Errorf("connection refused")` not `"Connection refused"`).
- No trailing punctuation (errors get wrapped — extra periods/colons cluster up).
- Include relevant fields: `fmt.Errorf("dial %s: %w", addr, err)` not just `"dial failed"`.

**Test error paths:**

If your function returns errors for specific conditions, write tests for those conditions. The test exercises the error construction (catches "I forgot to wrap" mistakes) and acts as documentation for callers.

## runtime.gopanic Internals

The `panic()` builtin calls `runtime.gopanic(v)` in `src/runtime/panic.go`. Simplified flow:

```go
// runtime.gopanic — when panic() is invoked
func gopanic(e any) {
    gp := getg()                  // current goroutine
    if gp.m.curg != gp { ... }    // sanity check

    // Build a _panic struct on the stack
    var p _panic
    p.arg = e
    p.link = gp._panic            // chain to existing panic (in case of nested)
    gp._panic = &p

    // Walk deferred functions in LIFO order, looking for recover()
    for {
        d := gp._defer
        if d == nil { break }

        // ... call deferred function ...
        d.started = true

        // Execute the deferred function. If it calls recover(), the recover
        // call sets p.recovered = true, and we'll bail out.
        reflectcall(nil, unsafe.Pointer(d.fn), nil, ...)

        gp._defer = d.link

        if p.recovered {
            // recover() succeeded — unwind to the deferred function's caller
            // by jumping to runtime.deferreturn-style continuation
            mcall(recovery)
            // Never returns here on success.
        }
    }

    // No deferred function recovered. Print stack trace and exit.
    fatalpanic(gp._panic)
}
```

The `_defer` struct is the linked-list node for deferred calls:

```go
type _defer struct {
    started bool
    heap    bool          // allocated on heap (post-1.13 escape) or stack?
    openDefer bool         // open-coded defer (1.14+ optimization)
    sp        uintptr     // sp at time of defer
    pc        uintptr     // pc at time of defer
    fn        *funcval
    _panic    *_panic
    link      *_defer
    rcvr      any          // receiver of recover()
}
```

## Open-Coded Defer (Go 1.14+)

For functions with at most 8 defers and no defers in loops, the compiler emits inline code instead of a runtime _defer chain:

```go
// Source:
func f() {
    defer cleanup()
    risky()
}

// Old (runtime):
//   pushdefer(cleanup)
//   risky()
//   // function epilogue calls deferreturn() which pops the chain

// 1.14+ open-coded:
//   var deferred uint8 = 0
//   risky()
//   if deferred & 1 { cleanup() }
```

This is much cheaper (no heap allocation for the _defer struct), making defer-in-hot-path acceptable in many cases.

## defer-in-loop Pitfall (and 1.22+ Fix)

```go
// PRE-1.22 BUG:
for _, file := range files {
    f, err := os.Open(file)
    if err != nil { continue }
    defer f.Close()  // ← all files held open until enclosing function returns!
    // ... process f ...
}
```

The fix pre-1.22 was to wrap the loop body in a closure:

```go
for _, file := range files {
    func() {
        f, err := os.Open(file)
        if err != nil { return }
        defer f.Close()
        // ... process f ...
    }()
}
```

Go 1.22 changed `for` loop variable semantics so each iteration gets its own variable, but the defer-FIFO-vs-iteration ordering issue remains for `defer`. The closure pattern is still the correct fix.

## Stack Unwinding Mechanics

When `gopanic` walks defers, it must unwind the stack:

```text
[ frame: f()             ] ← panic raised here
[ frame: g() defer h     ] ← _defer entry pointing at h
[ frame: main()          ]
```

The deferred `h` is invoked, then `g`'s frame is popped. If `h` recovers, control returns to `g`'s caller via `mcall(recovery)`, which sets up the goroutine's pc to resume at `g`'s deferreturn site.

The `lr` (link register on ARM) or return-address (x86) is used to find the caller's frame. The compiler emits frame-pointer info via `runtime.funcdata` (the `_FUNCDATA_LocalsPointerMaps` and `_FUNCDATA_StackObjects` sections of the binary).

## errors.Is / errors.As / errors.Unwrap Walk

```go
// The standard library implementation, simplified:

func Is(err, target error) bool {
    if target == nil { return err == target }

    isComparable := reflectlite.TypeOf(target).Comparable()
    for {
        if isComparable && err == target { return true }

        // Check if err implements Is(error) bool
        if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
            return true
        }

        // Walk to next
        switch x := err.(type) {
        case interface{ Unwrap() error }:
            err = x.Unwrap()
            if err == nil { return false }
        case interface{ Unwrap() []error }:
            for _, e := range x.Unwrap() {
                if Is(e, target) { return true }
            }
            return false
        default:
            return false
        }
    }
}
```

`errors.As(err, &target)` walks the same chain but checks whether `err` is assignable to `*target`'s type:

```go
func As(err error, target any) bool {
    val := reflectlite.ValueOf(target)
    if val.Kind() != reflectlite.Ptr || val.IsNil() {
        panic("errors: target must be a non-nil pointer")
    }

    targetType := val.Type().Elem()
    if targetType.Kind() != reflectlite.Interface && !targetType.Implements(errorType) {
        panic("errors: *target must be interface or implement error")
    }

    for err != nil {
        if reflectlite.TypeOf(err).AssignableTo(targetType) {
            val.Elem().Set(reflectlite.ValueOf(err))
            return true
        }
        if x, ok := err.(interface{ As(any) bool }); ok && x.As(target) {
            return true
        }
        err = Unwrap(err)
    }
    return false
}
```

## errors.Join (1.20+) Internals

```go
type joinError struct {
    errs []error
}

func (e *joinError) Error() string {
    var b []byte
    for i, err := range e.errs {
        if i > 0 { b = append(b, '\n') }
        b = append(b, err.Error()...)
    }
    return string(b)
}

func (e *joinError) Unwrap() []error { return e.errs }

func Join(errs ...error) error {
    n := 0
    for _, err := range errs {
        if err != nil { n++ }
    }
    if n == 0 { return nil }

    e := &joinError{errs: make([]error, 0, n)}
    for _, err := range errs {
        if err != nil { e.errs = append(e.errs, err) }
    }
    return e
}
```

`errors.Is(joined, target)` and `errors.As(joined, &target)` recursively check each wrapped error via the new `Unwrap() []error` interface.

## Race Detector ThreadSanitizer Internals

`-race` instruments every memory access with a call to runtime.racewrite or runtime.raceread. These calls update a per-goroutine "vector clock" tracking happens-before relationships:

```text
T1: write x  → racewrite(x, [T1: 5])
T2: read  x  → raceread(x, expected_clock=[T1: 5])
              → if T2's vector clock for T1 < 5, RACE detected
              (T2 doesn't have a synchronization edge from T1's write)
```

Synchronization primitives (channels, mutex Lock/Unlock, atomic ops, sync.Once) emit happens-before edges that update vector clocks correctly. The "WARNING: DATA RACE" output formats:

```text
WARNING: DATA RACE
Read at 0xc0000180c0 by goroutine 7:
  main.main.func1()
      /home/x/main.go:15 +0x44

Previous write at 0xc0000180c0 by goroutine 6:
  main.main.func2()
      /home/x/main.go:21 +0x44

Goroutine 7 (running) created at:
  main.main()
      /home/x/main.go:14 +0x66
Goroutine 6 (finished) created at:
  main.main()
      /home/x/main.go:20 +0x99
```

The `-race` build adds ~5x memory overhead and ~2-20x runtime overhead. ThreadSanitizer is fork-of-LLVM's TSan; it shares logic with C/C++/Rust race detection.

## signal-as-panic Mechanism

When a goroutine triggers SIGSEGV, the runtime's signal handler converts it to a Go panic:

```go
// runtime.sigpanic in src/runtime/signal_unix.go
func sigpanic() {
    g := getg()
    if !canpanic(g) {
        throw("unexpected signal during runtime execution")
    }

    switch g.sig {
    case _SIGBUS:
        if g.sigcode0 == _BUS_ADRERR && ... {
            panicmem()       // → "invalid memory address or nil pointer dereference"
        }
    case _SIGSEGV:
        if (g.sigcode0 == 0 || ...) && ... {
            panicmem()
        }
    case _SIGFPE:
        switch g.sigcode0 {
        case _FPE_INTDIV: panicdivide()
        case _FPE_INTOVF: panicoverflow()
        case _FPE_FLTDIV: panicdivide()
        }
    }
    panic(errorString("runtime error: " + ...))
}
```

The `panicmem`, `panicdivide`, etc. helpers raise the canonical runtime errors. This is why a nil pointer dereference appears as a panic rather than crashing the process — the runtime catches the signal and converts it.

## Common Library Pattern: io.EOF Sentinel

`io.EOF = errors.New("EOF")`. The convention: `Read` returns `(0, io.EOF)` on clean stream end. Special: `errors.Is(err, io.EOF)` always works because EOF is a sentinel; never wrap it (always return io.EOF directly when stream ended normally).

## Common Library Pattern: context cancellation

`context.Canceled` and `context.DeadlineExceeded` are sentinels. Wrapping them via `fmt.Errorf("%w", ctx.Err())` is correct and `errors.Is(err, context.Canceled)` will still match — but be aware that the wrapping adds a frame to the chain.

## When recover() Doesn't Work

```go
// FAILS: recover called outside of deferred function
func f() {
    defer log.Println("done")
    if err := recover(); err != nil {  // always returns nil — not in deferred fn
        ...
    }
    panic("X")
}

// FAILS: recover in different goroutine
func main() {
    defer func() {
        if err := recover(); err != nil { log.Println(err) }
    }()
    go func() {
        panic("oh no")  // ← crashes whole process; main's recover doesn't see it
    }()
    time.Sleep(time.Second)
}

// CORRECT: every goroutine that might panic should have its own top-level recover
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("goroutine panic: %v\n%s", r, debug.Stack())
        }
    }()
    risky()
}()
```

## References

- Go Source `src/runtime/panic.go`: https://github.com/golang/go/blob/master/src/runtime/panic.go
- Go Source `src/errors/errors.go`: https://github.com/golang/go/blob/master/src/errors/errors.go
- Go Source `src/fmt/errors.go`: https://github.com/golang/go/blob/master/src/fmt/errors.go
- Go 1.13 Release Notes, "Error wrapping": https://go.dev/doc/go1.13#error_wrapping
- Go 1.20 Release Notes, "errors.Join": https://go.dev/doc/go1.20#errors
- Go Blog, "Working with Errors in Go 1.13" (Damien Neil, 2019)
- Go Blog, "Defer, Panic, and Recover" (Andrew Gerrand, 2010)
- Go Blog, "Error handling and Go" (Andrew Gerrand, 2011)
- Russ Cox, "Go's path-dependent design choices" (golang-nuts)
- Dave Cheney, "Don't just check errors, handle them gracefully"
- Dave Cheney, "Constant errors"
- Proposal: errors.Is/As/Unwrap (Marcel van Lohuizen, 2018)
- Proposal: errors.Join (Damien Neil, 2022)
- "The Go Memory Model" — happens-before semantics for race detection
- Google's ThreadSanitizer paper (PLDI 2009)
- Go 1.14 Release Notes, "Asynchronous preemption" (SIGURG)
- Go 1.21 Release Notes, "panic(nil) is now a runtime.PanicNilError"
- Go 1.22 Release Notes, "Loop variable scoping change"
- `runtime` package documentation: https://pkg.go.dev/runtime
- `errors` package documentation: https://pkg.go.dev/errors
- `golang.org/x/sync/errgroup` documentation
- See Also: `sheets/troubleshooting/go-errors.md`, `detail/go/go-runtime.md`, `detail/go/go-concurrency.md`
