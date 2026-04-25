# Go (Programming Language)

Statically typed, compiled language with built-in concurrency, garbage collection, fast builds, and a deliberately small surface — the standard library is the framework.

## Setup

### Install Go

```bash
# macOS:    brew install go         # or pkg from go.dev/dl
# Linux:    tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz   # then add /usr/local/go/bin to PATH
# Windows:  msi installer from go.dev/dl
# Verify:   go version              # go version go1.23.4 darwin/arm64
# Check env: go env GOROOT GOPATH GOMODCACHE
```

### Project init (modules)

```bash
# mkdir myapp && cd myapp
# go mod init github.com/user/myapp     # creates go.mod (module path is import-friendly URL)
# touch main.go                          # package main with func main() runs as a binary
# go run .                               # compile + run the package in cwd
# go build -o myapp .                    # produce a static binary "myapp"
```

### GOPATH vs modules

```bash
# Pre-1.11:   code lived under $GOPATH/src/<import-path> — directory layout WAS the import.
# Modern:     modules — go.mod anywhere on disk, imports resolved via go.mod paths.
# Status:     $GOPATH still exists ($HOME/go by default) for downloaded module cache + installed binaries.
# go env GOPATH                         # default ~/go
# go env GOMODCACHE                     # ~/go/pkg/mod   (read-only cache)
# go env -w GOFLAGS=-mod=mod            # opt out of vendor lookup
```

### Version pinning

```bash
# go.mod toolchain directive (1.21+):
#     go 1.23
#     toolchain go1.23.4               // pinned compiler — `go` will download if missing
# go install cmd@v1.2.3                  # install a tool at a specific version
# go get example.com/lib@v1.5.0          # pin dependency version in go.mod
# go get example.com/lib@latest          # newest tagged release
# go get -u ./...                        # update all dependencies (minor/patch)
```

### Editor & tooling

```bash
# gopls         — official LSP (autocomplete, hover, refs, rename)
# go install golang.org/x/tools/gopls@latest
# dlv           — Delve debugger:  go install github.com/go-delve/delve/cmd/dlv@latest
# staticcheck   — best-in-class linter:  go install honnef.co/go/tools/cmd/staticcheck@latest
# golangci-lint — meta-linter, runs many at once (separate distribution, see references)
```

## Variables & Constants

### Variable declaration

```bash
# var x int                   // zero value (0)
# var s string                // zero value ("")
# var p *Foo                  // zero value (nil)
# var n int = 42              // explicit type + value
# var n = 42                  // type inferred — n is int
# n := 42                     // short decl — function scope only
# x, y := 1, "two"            // multiple assign with mixed types
# var ( a int; b string )     // grouped declaration
```

### The `:=` rule of thumb

```bash
# Use := inside functions for new locals.
# Use var at package level (no := allowed at top level).
# Mixing: a, err := f(); a, err = g()    // = re-uses existing a/err; at least one new var on LHS to use :=
```

### Constants

```bash
# const Pi = 3.14159              // untyped constant — adapts to context
# const MaxRetries int = 5         // typed constant
# const (
#     KB = 1 << 10                 // 1024
#     MB = 1 << 20
#     GB = 1 << 30
# )
# Constants are compile-time. They cannot reference function calls, and only basic types are allowed.
```

### iota — auto-incrementing enums

```bash
# const (
#     Sunday = iota               // 0
#     Monday                      // 1
#     Tuesday                     // 2
# )
# const (
#     _  = iota                   // skip 0
#     KB = 1 << (10 * iota)       // 1<<10
#     MB                          // 1<<20
#     GB                          // 1<<30
# )
# type Color int
# const ( Red Color = iota; Green; Blue )    // typed iota — gives you a typed enum
```

## Types

### Basic types

```bash
# Booleans:  bool                                    // true | false
# Numerics:  int int8 int16 int32 int64
#            uint uint8 uint16 uint32 uint64 uintptr
#            float32 float64 complex64 complex128
# Aliases:   byte == uint8           rune == int32
# Strings:   string                                  // immutable UTF-8 bytes
# int and uint are 32-bit on 32-bit platforms, 64-bit on 64-bit. For specific widths use intN/uintN.
```

### Type conversions

```bash
# i := 42
# f := float64(i)                              // explicit; no implicit int->float
# n, err := strconv.Atoi("42")                 // string -> int
# s := strconv.Itoa(i)                         // int -> string
# bs := []byte("hello")                        // string -> byte slice (copies)
# rs := []rune("héllo")                        // string -> rune slice (decodes UTF-8)
# back := string(bs)                            // bytes -> string (copies)
# // strconv.FormatFloat / ParseFloat / FormatBool / ParseBool for the rest
```

### Type aliases (`type X = Y`)

```bash
# type ByteSlice = []byte                      // EXACT same type, just a new name
# var b ByteSlice = []byte("ok")               // assignable both ways with no conversion
# Used for migration (rename without breaking callers) and for re-exporting types across packages.
```

### Type definitions (`type X Y`)

```bash
# type UserID int                               // NEW DISTINCT type — UserID and int are NOT interchangeable
# var id UserID = 42                            // ok — untyped literal
# var n int = 5
# id = n                                        // COMPILE ERROR
# id = UserID(n)                                // explicit conversion required
# // Methods can attach to defined types but NOT to aliases.
```

## Pointers

### Basics

```bash
# x := 42
# p := &x                                       // p is *int
# fmt.Println(*p)                               // 42 — dereference
# *p = 99                                        // mutate through pointer
# fmt.Println(x)                                 // 99
# var q *int                                     // q == nil (zero value)
# if q != nil { fmt.Println(*q) }               // dereferencing nil panics
```

### No pointer arithmetic

```bash
# Go has pointers but NOT pointer arithmetic. You cannot do p++ to walk an array.
# Use slices for sequence access; use unsafe.Pointer for FFI; use sync/atomic for atomic ops.
# This is intentional — it preserves memory safety and lets the GC track all references.
```

### `new` vs `make`

```bash
# new(T)        — allocates a zeroed T, returns *T               (works for any type)
# make(T, ...)  — initializes slice/map/chan and returns T (NOT *T) — required for those three
#
# p := new(int)                                  // *int pointing to 0
# s := make([]int, 0, 10)                        // []int  len=0 cap=10
# m := make(map[string]int)                      // empty map, ready for writes
# c := make(chan int, 5)                         // buffered channel
#
# new(T) is rare in idiomatic Go — &T{} is more common for structs:
# u := &User{Name: "Ada"}                        // preferred over new(User) + field assigns
```

## Strings & Runes

### Strings are immutable UTF-8 byte sequences

```bash
# s := "héllo"                                  // bytes: 0x68 0xC3 0xA9 0x6C 0x6C 0x6F (6 bytes)
# len(s)                                         // 6 — byte length, NOT character count
# s[0]                                            // byte: 'h' (uint8 / byte)
# s[1]                                            // byte: 0xC3 (start of é) — DOES NOT give 'é'
# // Indexing always returns a byte. Slicing by byte can split a codepoint mid-encoding.
```

### Range gives runes

```bash
# for i, r := range "héllo" {
#     fmt.Printf("%d: %c (%U)\n", i, r, r)      // i is BYTE offset, r is rune
# }
# // 0: h
# // 1: é   (i jumps from 1 to 3 because é took 2 bytes)
# // 3: l
# // 4: l
# // 5: o
```

### Counting characters / iterating safely

```bash
# import "unicode/utf8"
# n := utf8.RuneCountInString("héllo")          // 5 — codepoints, not bytes
#
# // Convert to []rune for random-access by codepoint:
# rs := []rune("héllo")                          // []rune of length 5
# rs[1]                                           // 'é'
#
# // Convert to []byte if you need to mutate (strings are immutable):
# b := []byte(s); b[0] = 'H'; s2 := string(b)
```

## Arrays vs Slices

### Arrays — fixed length, value type

```bash
# var a [3]int                                  // [0 0 0]
# a := [3]int{1, 2, 3}
# a := [...]int{1, 2, 3}                        // ... lets compiler count
# len(a)                                         // 3 — part of the type
# var b [3]int = a                               // COPY — arrays are value types
# // [3]int and [4]int are distinct types. Rarely used directly — slices are the workhorse.
```

### Slices — header over an array

```bash
# s := []int{1, 2, 3}                            // slice literal
# s := make([]int, 3)                            // [0 0 0]   len=3 cap=3
# s := make([]int, 0, 100)                       // len=0 cap=100   (preallocated)
# // A slice value is a 3-word struct: {pointer, len, cap}.
# // Multiple slices can share the same backing array.
```

## Slice Mechanics

### len, cap, append, copy

```bash
# s := []int{1, 2, 3}
# len(s); cap(s)                                // 3, 3
# s = append(s, 4, 5)                            // append may grow backing array if cap exceeded
# s2 := make([]int, len(s))
# copy(s2, s)                                    // src->dst by value; returns count copied
# // append returns a new slice header; ALWAYS reassign: s = append(s, x)
```

### Slice-of-slice

```bash
# s := []int{0, 1, 2, 3, 4, 5}
# s[2:5]                                          // [2 3 4]   len=3
# s[:3]                                           // [0 1 2]
# s[3:]                                           // [3 4 5]
# s[:]                                            // full slice (new header, same array)
# s[2:5:5]                                        // 3-index form: [low:high:max] sets cap explicitly
# s[2:5:5][:0]                                    // emptied with bounded cap — protects callers from append
```

### Slice aliasing trap

```bash
# // BUG: append into a sub-slice can stomp the parent's data when cap > len.
# orig := []int{1, 2, 3, 4}
# sub := orig[:2]                                // shares backing — len=2 cap=4
# sub = append(sub, 99)                          // writes index 2 of orig — orig now [1 2 99 4]!
# // FIX: clone or use 3-index slicing to limit cap.
# safe := append([]int(nil), orig[:2]...)        // independent copy
# safe = append(safe, 99)                        // orig unchanged
```

## Maps

### Construction and access

```bash
# m := map[string]int{"alice": 1, "bob": 2}
# m["carol"] = 3                                 // insert / update
# v := m["alice"]                                // 1
# v := m["zzz"]                                  // 0 — ZERO VALUE for missing keys (no error)
# v, ok := m["zzz"]                              // comma-ok: ok=false if missing
# delete(m, "bob")                               // no-op if missing
# len(m)                                          // count
# m2 := make(map[string]int, 100)                // hint capacity to avoid rehashes
```

### Iteration is randomized

```bash
# for k, v := range m { fmt.Println(k, v) }
# // Order is RANDOMIZED on every range. Don't rely on it.
# // To emit deterministically, sort keys:
# keys := make([]string, 0, len(m))
# for k := range m { keys = append(keys, k) }
# sort.Strings(keys)
# for _, k := range keys { fmt.Println(k, m[k]) }
```

### Nil-map writes panic

```bash
# var m map[string]int                            // nil map (zero value)
# v := m["x"]                                    // OK — reads return zero value
# m["x"] = 1                                     // PANIC: assignment to entry in nil map
# // FIX: always initialize before write.
# m = make(map[string]int)                        // now safe
```

## Control Flow

### if / else

```bash
# if x > 0 { ... } else if x < 0 { ... } else { ... }
# // Init clause is idiomatic — confines the variable to the if-block.
# if v, err := f(); err != nil { return err } else { use(v) }
# // Parens around the condition are NOT used; braces are mandatory.
```

### switch — no fallthrough by default

```bash
# switch x {
# case 1, 2, 3:                                   // multi-value case
#     fmt.Println("small")
# case 4:
#     fmt.Println("four")
#     fallthrough                                 // explicit opt-in to next case
# case 5:
#     fmt.Println("five (fell through)")
# default:
#     fmt.Println("other")
# }
# // switch with no expr behaves like if/else if chain:
# switch { case x > 100: ...; case x > 10: ...; default: ... }
```

### Type switch

```bash
# func describe(v any) string {
#     switch t := v.(type) {                     // .(type) only valid inside switch
#     case nil:    return "nil"
#     case int:    return fmt.Sprintf("int %d", t)
#     case string: return "string " + t
#     case []byte: return fmt.Sprintf("bytes len=%d", len(t))
#     default:     return fmt.Sprintf("unknown %T", t)
#     }
# }
```

## For Loops

### The only loop construct

```bash
# for i := 0; i < 10; i++ { ... }                // C-style three-clause
# for i < 10 { ... }                             // condition-only — like while
# for { break }                                  // infinite loop — like while(true)
```

### range — over slice / array / map / string / chan

```bash
# for i, v := range slice { ... }                // i = index, v = element copy
# for i := range slice { ... }                   // index only
# for _, v := range slice { ... }                // value only
# for k, v := range mapVal { ... }               // randomized order
# for k := range mapVal { ... }                  // keys only
# for v := range chanVal { ... }                 // receive until channel closed
# for i, r := range "héllo" { ... }              // i=byte offset, r=rune
```

### Labels for nested break / continue

```bash
# OUTER:
# for i := 0; i < 10; i++ {
#     for j := 0; j < 10; j++ {
#         if grid[i][j] == target {
#             break OUTER                         // break the outer loop
#         }
#     }
# }
# // continue LABEL similarly skips to the next iteration of the labeled loop.
```

## Functions

### Definition

```bash
# func add(a, b int) int { return a + b }       // shared type — (a, b int) means both int
# func divide(a, b float64) (float64, error) {  // multiple return values
#     if b == 0 { return 0, fmt.Errorf("divide by zero") }
#     return a / b, nil
# }
```

### Named returns

```bash
# func split(sum int) (x, y int) {              // named — pre-declared as zero values
#     x = sum * 4 / 9
#     y = sum - x
#     return                                     // "naked return" — returns x, y as named
# }
# // Use sparingly. Named returns + naked return + defer is a footgun.
```

### Multiple return — comma-ok / value-error

```bash
# v, ok := m["key"]                              // comma-ok idiom — common for maps, type assertions
# data, err := os.ReadFile("/tmp/x")            // value-error idiom — convention for fallible funcs
# if err != nil { return fmt.Errorf("read: %w", err) }
```

## Variadic Functions

```bash
# func sum(nums ...int) int {                    // nums is []int inside the function
#     total := 0
#     for _, n := range nums { total += n }
#     return total
# }
# sum()                                          // 0
# sum(1, 2, 3)                                   // 6
# nums := []int{4, 5, 6}
# sum(nums...)                                   // SPREAD — pass slice as variadic args
# // Common idiom: forwarding to fmt.Sprintf
# func wrap(format string, args ...any) string {
#     return "ERR: " + fmt.Sprintf(format, args...)
# }
```

## Closures & Function Values

```bash
# // Functions are first-class values.
# add := func(a, b int) int { return a + b }
# add(2, 3)                                      // 5
#
# // Closures capture enclosing variables BY REFERENCE.
# func counter() func() int {
#     n := 0
#     return func() int { n++; return n }
# }
# c := counter()
# c(); c(); c()                                  // 1, 2, 3 — n persists, captured by reference
#
# // Pass functions as parameters:
# func apply(f func(int) int, x int) int { return f(x) }
# apply(func(n int) int { return n*2 }, 5)       // 10
```

## Defer / Panic / Recover

### defer — LIFO cleanup

```bash
# func main() {
#     defer fmt.Println("1")
#     defer fmt.Println("2")
#     defer fmt.Println("3")
#     fmt.Println("hi")
# }
# // Output:
# //   hi
# //   3
# //   2
# //   1
# // Defers run in LIFO order at function return — even on panic.
# // Common use: defer file.Close(); defer mu.Unlock(); defer cancel()
```

### Deferred argument evaluation

```bash
# // Arguments to deferred calls are EVALUATED IMMEDIATELY at defer time, not at call time.
# i := 0
# defer fmt.Println(i)                           // captures 0 right now
# i = 99
# // prints 0, not 99
#
# // To defer the LATEST value, wrap in a closure:
# defer func() { fmt.Println(i) }()              // reads i at call time → prints 99
```

### panic and recover

```bash
# func safeDiv(a, b int) (result int, err error) {
#     defer func() {
#         if r := recover(); r != nil {
#             err = fmt.Errorf("recovered: %v", r)
#         }
#     }()
#     return a / b, nil                          // panics if b == 0
# }
# // recover only works inside a deferred function. It catches a panicking goroutine.
# // recover OUTSIDE a defer returns nil. Recover from a parent goroutine cannot catch panics
# // in a child goroutine — each goroutine must recover itself.
```

## Methods

### Value vs pointer receivers

```bash
# type Counter struct { n int }
#
# func (c Counter) Get() int   { return c.n }    // VALUE receiver — works on a copy
# func (c *Counter) Inc()      { c.n++ }         // POINTER receiver — mutates the caller
#
# c := Counter{}
# c.Inc()                                        // Go auto-takes &c — mutation is visible
# c.Get()                                        // 1
```

### When to use which

```bash
# Use POINTER receiver when:
#   • the method modifies the receiver
#   • the struct is large (avoid copying on every call)
#   • the type contains sync.Mutex or other non-copyable fields
#   • OTHER methods on the type use pointer receivers (consistency!)
#
# Use VALUE receiver when:
#   • the type is small (a few words)
#   • the method does not need to mutate
#   • the type is intended to be used as an immutable value (like time.Time)
#
# Rule of thumb: pick one and use it for ALL methods on the type. Mixing causes confusion.
```

## Interfaces

### Implicit satisfaction

```bash
# type Stringer interface { String() string }
#
# type User struct { Name string }
# func (u User) String() string { return "user: " + u.Name }
#
# // No "implements Stringer" keyword — User satisfies Stringer because it has String() string.
# var s Stringer = User{Name: "Ada"}
# fmt.Println(s.String())
```

### Empty interface — `any`

```bash
# // any is an alias for interface{} (since Go 1.18).
# var v any                                       // can hold a value of any type
# v = 42; v = "hi"; v = []int{1,2}
# // Useful for fmt.Println, json.Unmarshal, etc.
# // Lose all static type info — use type assertions to recover.
```

### Type assertions

```bash
# var v any = "hello"
# s := v.(string)                                // PANIC if v is not string
# s, ok := v.(string)                            // safe form — ok=false on mismatch
# if !ok { return fmt.Errorf("not a string") }
```

## Embedding

### Struct embedding (composition)

```bash
# type Animal struct { Name string }
# func (a Animal) Speak() string { return a.Name + " makes a sound" }
#
# type Dog struct {
#     Animal                                     // embedded — fields/methods promoted
#     Breed string
# }
#
# d := Dog{Animal: Animal{Name: "Rex"}, Breed: "Lab"}
# d.Name                                         // "Rex" — field PROMOTION
# d.Speak()                                      // "Rex makes a sound" — method PROMOTION
# d.Animal.Name                                  // explicit access still works
```

### Interface embedding

```bash
# type Reader interface { Read(p []byte) (n int, err error) }
# type Closer interface { Close() error }
#
# type ReadCloser interface {                    // composed — implementers must satisfy BOTH
#     Reader
#     Closer
# }
# // io.ReadCloser, io.ReadWriter, io.ReadWriteCloser are all defined this way in stdlib.
```

## Generics (1.18+)

### Type parameters

```bash
# func Max[T cmp.Ordered](a, b T) T {            // 1.21+: cmp.Ordered = ints, floats, strings
#     if a > b { return a }
#     return b
# }
# Max(3, 5)                                      // T inferred: int
# Max[string]("a", "b")                          // explicit instantiation
```

### Constraints

```bash
# import "constraints"   // OR write your own:
# type Number interface {
#     ~int | ~int32 | ~int64 | ~float32 | ~float64    // ~ allows defined types whose underlying is int etc.
# }
# func Sum[T Number](xs []T) T {
#     var s T
#     for _, x := range xs { s += x }
#     return s
# }
# Sum([]int{1, 2, 3})                            // 6
```

### slices and maps packages

```bash
# import "slices"                                // 1.21+ — moved out of x/exp
# slices.Contains([]int{1,2,3}, 2)               // true
# slices.Index([]int{1,2,3}, 3)                  // 2
# slices.Sort([]int{3,1,2})                      // in-place sort using cmp.Ordered
# slices.Reverse(s)
# slices.Clone(s)                                // independent copy
# slices.Equal(a, b)
# slices.BinarySearch(sorted, target)
#
# import "maps"                                   // 1.21+
# maps.Keys(m); maps.Values(m)                   // iterators in 1.23+
# maps.Clone(m); maps.Equal(m1, m2); maps.Copy(dst, src)
```

## Goroutines

### Launching

```bash
# go fetch(url)                                   // run fetch in a new goroutine
# go func() {
#     defer wg.Done()
#     work()
# }()
# // A goroutine is a lightweight thread managed by the Go runtime, not the OS.
# // Initial stack: 2 KB. Grows/shrinks dynamically. Millions per process is normal.
```

### The GMP model

```bash
# G — Goroutine (your code, with its stack/state)
# M — Machine (an OS thread)
# P — Processor (a logical scheduler — owns a runnable queue)
#
# At any moment: GOMAXPROCS Ps are bound to Ms running Gs.
# Default GOMAXPROCS == NumCPU.
# Blocking syscalls park the M and another M takes over the P, so other Gs keep running.
# Channel ops, defer, GC are scheduler safe-points.
```

### GOMAXPROCS

```bash
# import "runtime"
# runtime.GOMAXPROCS(0)                          // returns current value (no change)
# runtime.GOMAXPROCS(4)                          // limit to 4 logical processors
# // Or via env: GOMAXPROCS=4 go run .
# // Pinning helps in containers where /sys/fs reports host CPUs but cgroup limits you to 2.
# // Use go.uber.org/automaxprocs to read cgroup quota.
```

## Channels

### Construction

```bash
# ch := make(chan int)                           // UNBUFFERED — send blocks until recv
# ch := make(chan int, 10)                       // BUFFERED capacity 10 — send blocks only when full
# var ch chan int                                // NIL channel — send/recv block forever
```

### Send / receive / close

```bash
# ch <- 42                                       // send (blocks if full / unbuffered with no recv)
# v := <-ch                                      // recv (blocks if empty)
# v, ok := <-ch                                  // ok=false if channel closed AND drained
# close(ch)                                      // signal "no more sends" — closing twice or sending after close PANICS
# for v := range ch { ... }                      // recv until close — drains then exits
```

### Direction

```bash
# func send(out chan<- int)    { out <- 1 }      // chan<- = send-only
# func recv(in  <-chan int)    { v := <-in }    // <-chan = recv-only
# // Use directional types to enforce role at compile time.
```

### One-shot vs streaming

```bash
# // ONE-SHOT: result + done signal
# done := make(chan struct{})
# go func() { work(); close(done) }()
# <-done                                          // wait
#
# // STREAMING: producer/consumer
# events := make(chan Event, 64)                 // buffered to absorb bursts
# go produce(events)                              // closes when source exhausted
# for ev := range events { handle(ev) }          // exits when channel closed
```

### Nil channel idiom

```bash
# // Setting a channel to nil DISABLES that case in a select. Useful to "turn off" a source.
# var in chan Event = events
# for in != nil {
#     select {
#     case ev, ok := <-in:
#         if !ok { in = nil; continue }           // source exhausted — disable this case
#         process(ev)
#     case <-ctx.Done():
#         return
#     }
# }
```

## Select

### Multiplexing

```bash
# select {
# case msg := <-ch1:
#     handle(msg)
# case ch2 <- result:
#     // sent successfully
# case <-time.After(5 * time.Second):            // timeout via channel from time package
#     return errTimeout
# case <-ctx.Done():
#     return ctx.Err()
# }
# // If multiple cases are ready, one is chosen at RANDOM. Otherwise blocks until one becomes ready.
```

### Default case — non-blocking

```bash
# select {
# case ev := <-events:
#     handle(ev)
# default:
#     // no event ready — don't block
# }
# // Use default for try-recv / try-send semantics.
```

## Sync Primitives

### Mutex / RWMutex

```bash
# var mu sync.Mutex
# mu.Lock()
# defer mu.Unlock()
# counter++
#
# var rw sync.RWMutex
# rw.RLock(); v := cache[k]; rw.RUnlock()       // many readers
# rw.Lock();  cache[k] = v; rw.Unlock()         // exclusive writer
# // Don't COPY a Mutex value — its internal state would split.
```

### WaitGroup

```bash
# var wg sync.WaitGroup
# for i := 0; i < 10; i++ {
#     wg.Add(1)                                  // increment BEFORE the goroutine
#     go func(id int) {
#         defer wg.Done()                        // decrement at exit
#         work(id)
#     }(i)
# }
# wg.Wait()                                      // blocks until counter == 0
```

### Once

```bash
# var (
#     once  sync.Once
#     conn  *DB
# )
# func GetDB() *DB {
#     once.Do(func() { conn = openDB() })        // runs exactly once across all goroutines
#     return conn
# }
# // Standard for one-time init. Safer than ad-hoc booleans.
```

### Pool — reusable temp objects

```bash
# var bufPool = sync.Pool{
#     New: func() any { return new(bytes.Buffer) },
# }
# b := bufPool.Get().(*bytes.Buffer)
# b.Reset()
# // ... use b ...
# bufPool.Put(b)
# // Items can be reclaimed by GC. Don't put state-bearing objects here without reset.
```

### Cond — rare; usually channels are better

```bash
# var (
#     mu   sync.Mutex
#     cond = sync.NewCond(&mu)
#     items []int
# )
# // Producer:
# mu.Lock(); items = append(items, x); cond.Signal(); mu.Unlock()
# // Consumer:
# mu.Lock()
# for len(items) == 0 { cond.Wait() }            // releases mu, blocks, re-acquires on wake
# x := items[0]; items = items[1:]
# mu.Unlock()
```

## Atomic

### sync/atomic — for lock-free counters & flags

```bash
# import "sync/atomic"
# var n atomic.Int64                              // 1.19+ — type-safe wrapper
# n.Add(1)
# n.Load()
# n.Store(42)
# n.CompareAndSwap(42, 99)                        // CAS for lock-free updates
# // Older API still works:   atomic.AddInt64(&counter, 1)   atomic.LoadInt64(&counter)
# // Use atomics ONLY for primitive counters/flags. For complex state, use a mutex.
```

## Context

### Background, TODO

```bash
# import "context"
# ctx := context.Background()                    // root — for main, init, tests
# ctx := context.TODO()                          // placeholder when unsure — same as Background but greppable
```

### WithCancel

```bash
# ctx, cancel := context.WithCancel(parent)
# defer cancel()                                  // ALWAYS call cancel — even on success — to free resources
# go work(ctx)                                   // work returns when ctx.Done() fires
```

### WithTimeout / WithDeadline

```bash
# ctx, cancel := context.WithTimeout(parent, 5*time.Second)
# defer cancel()
# req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
# resp, err := http.DefaultClient.Do(req)        // canceled at 5s; err includes context.DeadlineExceeded
```

### WithValue

```bash
# type ctxKey string
# const userIDKey ctxKey = "userID"
# ctx := context.WithValue(parent, userIDKey, 42)
# id, _ := ctx.Value(userIDKey).(int)
# // Use sparingly — for request-scoped data only (auth, traceID). Don't pass deps via context.
```

### Propagation idioms

```bash
# // 1. ctx is the FIRST parameter, always named ctx.
# func Fetch(ctx context.Context, url string) ([]byte, error) { ... }
# // 2. Don't store ctx in structs — pass it through.
# // 3. Always check ctx.Err() at boundaries:
# select {
# case <-ctx.Done():
#     return nil, ctx.Err()
# case data := <-resultCh:
#     return data, nil
# }
```

## Errors

### Creating

```bash
# import "errors"
# err := errors.New("not found")
# err := fmt.Errorf("invalid id: %d", id)        // formatted
# err := fmt.Errorf("read %s: %w", path, ioErr)  // %w wraps — preserves chain
```

### errors.Is and errors.As

```bash
# // Is: walks the chain looking for a matching SENTINEL.
# var ErrNotFound = errors.New("not found")
# if errors.Is(err, ErrNotFound) { ... }
#
# // As: walks the chain looking for a matching TYPE; assigns to target.
# var perr *os.PathError
# if errors.As(err, &perr) { fmt.Println(perr.Path) }
```

### Custom error types

```bash
# type NotFoundError struct { ID string }
# func (e *NotFoundError) Error() string { return fmt.Sprintf("not found: %s", e.ID) }
#
# // Use a pointer receiver so two NotFoundError{} values compare by identity, not by ID.
# // Or implement Is(target error) bool for value-based equality.
```

### Wrapping convention

```bash
# // Wrap with operation context as you bubble up:
# func loadConfig(path string) (*Config, error) {
#     data, err := os.ReadFile(path)
#     if err != nil { return nil, fmt.Errorf("loadConfig %s: %w", path, err) }
#     ...
# }
# // Never just return raw err from a deep callsite — wrap it for traceability.
# // Don't double-prefix: "loadConfig: read x: open x: ENOENT" — pick the layer that adds value.
```

## Panics in Production

```bash
# // Don't.
# // Panics are for unrecoverable bugs (nil dereference, index OOB, impossibility).
# // Library code should return errors, not panic.
# // Recover only at goroutine ROOTS:
# go func() {
#     defer func() {
#         if r := recover(); r != nil {
#             log.Printf("worker panic: %v\n%s", r, debug.Stack())
#         }
#     }()
#     work()
# }()
# // Never let panics cross goroutine boundaries — the parent CANNOT catch them.
```

## JSON

### Encode / decode

```bash
# import "encoding/json"
# data, err := json.Marshal(user)                // []byte
# data, err := json.MarshalIndent(user, "", "  ")
# err := json.Unmarshal(data, &user)             // pass POINTER
#
# // Streaming (preferred for HTTP / files):
# err := json.NewDecoder(r.Body).Decode(&user)
# err := json.NewEncoder(w).Encode(user)
```

### Struct tags

```bash
# type User struct {
#     Name  string `json:"name"`
#     Email string `json:"email,omitempty"`      // omit if zero
#     Pwd   string `json:"-"`                     // never marshal
#     Age   int    `json:"age,string"`            // marshal as string "42"
# }
# // Field name is exact-match on input. Tags are required for non-Go-idiomatic field names.
```

### Decoder vs Unmarshal

```bash
# // Use Decoder for streams and large payloads — avoids buffering the whole body.
# dec := json.NewDecoder(r.Body)
# dec.DisallowUnknownFields()                    // reject extra fields — strict mode
# for dec.More() {
#     var item Item
#     if err := dec.Decode(&item); err != nil { return err }
#     handle(item)
# }
# // Use Unmarshal when you already have a []byte in memory.
```

## File I/O

### Read / write whole files

```bash
# data, err := os.ReadFile("config.json")        // []byte — preferred over ioutil
# err := os.WriteFile("out.txt", data, 0644)
```

### Open / Create / OpenFile

```bash
# f, err := os.Open("in.txt")                    // read-only
# defer f.Close()
# f, err := os.Create("out.txt")                 // create or truncate
# f, err := os.OpenFile("log.txt",
#     os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // append mode
```

### Buffered I/O

```bash
# import "bufio"
# r := bufio.NewReader(f)                        // buffered reader
# line, err := r.ReadString('\n')                // until delimiter
#
# // Scanner — line-at-a-time is the default.
# s := bufio.NewScanner(f)
# for s.Scan() { line := s.Text() }              // returns line without trailing \n
# if err := s.Err(); err != nil { ... }
# // Default token size 64 KB — bump for huge lines: s.Buffer(make([]byte,1<<20), 1<<20)
#
# w := bufio.NewWriter(f)
# defer w.Flush()                                 // CRITICAL — buffered writes lost without Flush
# fmt.Fprintln(w, "line")
```

## Stdio / Args / Env

```bash
# // Args
# os.Args                                        // []string — Args[0] is program name
# // Env
# os.Getenv("HOME")                              // "" if unset
# os.Setenv("KEY", "value")
# os.LookupEnv("KEY")                            // value, found bool — distinguishes empty vs unset
#
# // Stdin
# var name string
# fmt.Scan(&name)                                // whitespace-separated tokens
# fmt.Scanln(&name)                              // until newline
# r := bufio.NewReader(os.Stdin)
# line, _ := r.ReadString('\n')                 // including \n
#
# // Stdout / Stderr
# fmt.Println("stdout")                          // to os.Stdout
# fmt.Fprintln(os.Stderr, "stderr msg")
# // For args parsing, use stdlib `flag` package or third-party (cobra, kong).
```

## Subprocess

### os/exec

```bash
# import "os/exec"
# cmd := exec.Command("git", "log", "-1", "--oneline")
#
# out, err := cmd.Output()                       // stdout only; stderr discarded
# out, err := cmd.CombinedOutput()               // merged stdout+stderr
# err := cmd.Run()                               // run + wait, output goes to /dev/null unless wired up
#
# // Stream stdout:
# cmd.Stdout = os.Stdout
# cmd.Stderr = os.Stderr
# err := cmd.Run()
#
# // Pipe to/from:
# stdout, _ := cmd.StdoutPipe()
# cmd.Start()
# // ... read from stdout ...
# cmd.Wait()
#
# // ExitError gives access to ProcessState:
# var exitErr *exec.ExitError
# if errors.As(err, &exitErr) { fmt.Println(exitErr.ExitCode()) }
```

## Date & Time

### Now / construction

```bash
# import "time"
# t := time.Now()                                 // local time
# t := time.Now().UTC()                           // UTC
# t := time.Date(2025, time.March, 15, 10, 30, 0, 0, time.UTC)
```

### Parse and Format — the reference layout

```bash
# // Layout is the magic timestamp Mon Jan 2 15:04:05 MST 2006 (== 01/02 03:04:05PM '06 -0700)
# // Memorize: 1 2 3 4 5 6 7 = Jan 2 15:04:05 2006 -0700
# t, err := time.Parse(time.RFC3339, "2025-04-25T10:30:00Z")
# t, err := time.Parse("2006-01-02 15:04:05", "2025-04-25 10:30:00")
# s := t.Format("2006-01-02")                    // "2025-04-25"
# s := t.Format(time.RFC3339)                    // "2025-04-25T10:30:00Z"
```

### Durations

```bash
# d := 5 * time.Second
# d := 100 * time.Millisecond
# d := time.Hour + 30*time.Minute
# t.Add(d)                                       // shift forward
# t2.Sub(t1)                                     // returns time.Duration
# d.Seconds(); d.Milliseconds()
```

### Zero value gotcha

```bash
# var t time.Time                                 // ZERO VALUE — Jan 1, year 1 UTC
# t.IsZero()                                     // true
# // time.Time{} is NOT the same as time.Unix(0,0) (== Jan 1 1970 UTC).
# // Use IsZero() to check "unset" — never compare with == time.Time{}.
```

## Regex

### Compile and match

```bash
# import "regexp"
# re := regexp.MustCompile(`(\w+)@(\w+\.\w+)`)   // panic on bad pattern — fine for static
# re, err := regexp.Compile(pattern)             // dynamic
# re.MatchString("a@b.com")                      // true
# // RE2 syntax — no backreferences, no lookaround. Linear time guaranteed.
```

### Capture groups

```bash
# m := re.FindStringSubmatch("alice@example.com")
# // m[0]="alice@example.com", m[1]="alice", m[2]="example.com"
#
# // Named groups:
# re := regexp.MustCompile(`(?P<user>\w+)@(?P<domain>\S+)`)
# m := re.FindStringSubmatch("a@b.com")
# names := re.SubexpNames()                      // ["", "user", "domain"]
```

### Replace

```bash
# re.ReplaceAllString("a@b.com x@y.org", "EMAIL")
# // With backrefs:
# re.ReplaceAllString(s, "user=$1 domain=$2")
# // With function:
# re.ReplaceAllStringFunc(s, strings.ToUpper)
```

## HTTP

### Client — quick get

```bash
# import "net/http"
# resp, err := http.Get(url)
# defer resp.Body.Close()
# body, _ := io.ReadAll(resp.Body)
# // http.DefaultClient has NO timeout — easy to hang forever. Don't ship this in production.
```

### Client with timeout

```bash
# c := &http.Client{ Timeout: 10 * time.Second }
# req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
# req.Header.Set("Content-Type", "application/json")
# req.Header.Set("Authorization", "Bearer "+token)
# resp, err := c.Do(req)
```

### Server with ServeMux

```bash
# mux := http.NewServeMux()
# mux.HandleFunc("GET /api/users", listUsers)            // 1.22+ method patterns
# mux.HandleFunc("POST /api/users", createUser)
# mux.HandleFunc("GET /api/users/{id}", getUser)         // 1.22+ path params
# srv := &http.Server{
#     Addr:              ":8080",
#     Handler:           mux,
#     ReadHeaderTimeout: 5 * time.Second,                // production: ALWAYS set timeouts
#     ReadTimeout:       30 * time.Second,
#     WriteTimeout:      30 * time.Second,
#     IdleTimeout:       120 * time.Second,
# }
# log.Fatal(srv.ListenAndServe())
```

## Reflection

### When to use

```bash
# import "reflect"
# // reflect lets you inspect types and values at runtime.
# // Use cases: fmt.Printf %v, encoding/json, ORMs.
# // For application code, almost ALWAYS prefer interfaces / generics. Reflection is slow and brittle.
```

### TypeOf / ValueOf

```bash
# v := reflect.ValueOf(x)
# t := reflect.TypeOf(x)
# t.Kind()                                       // reflect.Int, reflect.Slice, ...
# v.Kind()                                       // same
# // Mutating via reflect requires reflect.ValueOf(&x).Elem() — pointer indirection.
# // Struct field tags accessed via t.Field(i).Tag.Get("json")
```

## Build / Run / Install

### Common commands

```bash
go run .                              # compile + execute the package in cwd
go run main.go                        # run a single file
go build -o myapp .                   # build current package, write to ./myapp
go build ./...                        # build every package in the module
go install ./cmd/myapp                # install to $GOBIN (defaults $GOPATH/bin)
go install golang.org/x/tools/gopls@latest    # install a remote tool at a version
```

### ldflags — strip and version-stamp

```bash
go build -ldflags="-s -w" -o myapp .                  # -s strip symbol, -w strip DWARF
go build -ldflags="-X main.version=1.2.3 -X main.commit=$(git rev-parse --short HEAD)" .
# In code: var version = "dev"   (overwritten by -X)
go build -trimpath -ldflags="-s -w" .                  # remove file paths from binary
```

### Cross-compile

```bash
GOOS=linux   GOARCH=amd64 go build -o myapp-linux .
GOOS=windows GOARCH=amd64 go build -o myapp.exe .
GOOS=darwin  GOARCH=arm64 go build -o myapp-mac-m1 .
CGO_ENABLED=0 go build -o myapp .                    # static binary, no libc dependency
go tool dist list                                     # all GOOS/GOARCH combinations
```

## Test

### go test basics

```bash
go test                              # run tests in current package
go test ./...                        # all packages in module
go test -v                           # verbose — print PASS/FAIL per test
go test -run TestSpecific            # filter by name (regex)
go test -count=1                     # disable test result caching
go test -timeout 30s ./...           # per-test timeout
```

### Table-driven tests

```bash
# func TestAdd(t *testing.T) {
#     tests := []struct {
#         name    string
#         a, b    int
#         want    int
#     }{
#         {"two positives", 2, 3, 5},
#         {"with zero",     0, 7, 7},
#         {"two negatives", -1, -2, -3},
#     }
#     for _, tt := range tests {
#         t.Run(tt.name, func(t *testing.T) {        // subtest — filterable, runs in own scope
#             if got := Add(tt.a, tt.b); got != tt.want {
#                 t.Errorf("Add(%d,%d) = %d, want %d", tt.a, tt.b, got, tt.want)
#             }
#         })
#     }
# }
```

### Parallel + Helper

```bash
# func TestX(t *testing.T) {
#     t.Parallel()                                  // run alongside other Parallel tests
#     // ...
# }
#
# func mustOpen(t *testing.T, path string) *os.File {
#     t.Helper()                                    // hide this frame from failure reports
#     f, err := os.Open(path)
#     if err != nil { t.Fatalf("open %s: %v", path, err) }
#     return f
# }
```

### Test fixtures

```bash
# // Files in testdata/ are ignored by go build but available at test time.
# data, _ := os.ReadFile("testdata/sample.json")
# // t.TempDir() gives an auto-cleaned dir per test:
# dir := t.TempDir()
```

## Race / Bench / Fuzz

### Race detector

```bash
go test -race ./...                  # run all tests with race detector
go run -race main.go                  # also works on `run`
go build -race -o myapp .             # binary with race checks (only for testing — slow)
# Race detector finds DATA races at runtime. CI MUST run -race or you'll ship races.
```

### Benchmarks

```bash
# func BenchmarkAdd(b *testing.B) {
#     for i := 0; i < b.N; i++ {
#         Add(2, 3)
#     }
# }
# // 1.24+ alternative:
# func BenchmarkAdd2(b *testing.B) { for b.Loop() { Add(2, 3) } }

go test -bench=.                      # run all benchmarks
go test -bench=BenchmarkAdd -benchmem # show allocs/op
go test -bench=. -benchtime=10s       # run each for 10s
go test -bench=. -count=10            # 10 runs — feed to benchstat
```

### Fuzz (1.18+)

```bash
# func FuzzReverse(f *testing.F) {
#     f.Add("hello")                                // seed corpus
#     f.Fuzz(func(t *testing.T, s string) {
#         r := Reverse(s)
#         if Reverse(r) != s {
#             t.Errorf("not symmetric: %q -> %q -> %q", s, r, Reverse(r))
#         }
#     })
# }

go test -fuzz=FuzzReverse -fuzztime=30s
# Found inputs saved to testdata/fuzz/<name>/ and become regression cases.
```

## Vet & Static Analysis

```bash
go vet ./...                          # built-in static checker — catches printf format bugs etc.
staticcheck ./...                     # honnef.co/go/tools/cmd/staticcheck — much deeper checks
golangci-lint run                     # meta-runner for many linters via .golangci.yml
gofmt -s -w .                         # canonical formatting; -s simplifies code
goimports -w .                        # gofmt + auto-manage import groups
```

## Build Tags

### //go:build constraints

```bash
# //go:build linux && amd64                      // file compiled only for linux/amd64
# //go:build !cgo                                 // exclude when cgo enabled
# //go:build integration                          // requires -tags=integration to compile
#
# package mypkg
# // Build tag MUST be the first non-blank, non-comment line, followed by a blank line.
# // Old syntax: // +build linux  — pre-1.17, still works but use //go:build going forward.
#
# go build -tags=integration ./...
```

## Modules — Deeper

### replace directive

```bash
# go.mod:
#     replace example.com/lib => ../lib                 // local path
#     replace example.com/lib v1.2.0 => example.com/fork v1.2.1
# // Useful for forks and local dev. Don't ship a release with replace pointing at local paths.
```

### vendor

```bash
go mod vendor                         # write deps into ./vendor
go build -mod=vendor                  # build using vendor only — no network
# Common in airgap / regulated environments. Larger repo, but reproducible.
```

### GOPRIVATE

```bash
go env -w GOPRIVATE='*.corp.example.com,github.com/myorg/*'
# Tells go: don't proxy these, don't checksum them. Required for private repos.
# Pair with GOPROXY (default https://proxy.golang.org,direct) and GONOSUMCHECK.
```

### Workspaces (go.work, 1.18+)

```bash
go work init ./moduleA ./moduleB      # create go.work covering multiple modules
go work use ./moduleC                 # add another
# go.work overrides go.mod replace directives during local dev.
# Don't commit go.work to release branches — use go.work.sum or .gitignore it.
```

## Standard Library Highlights

```bash
# fmt           — Print/Sprint/Fprintf, %v %s %d %q %T %+v %#v
# strings       — Contains, HasPrefix, Split, Join, ReplaceAll, ToLower, Builder
# strconv       — Atoi, Itoa, ParseFloat, ParseBool, FormatFloat, Quote
# bytes         — Buffer, NewReader, parallel API to strings for []byte
# io            — Reader/Writer/Closer interfaces, Copy, ReadAll, LimitReader, MultiReader
# bufio         — Scanner, NewReader/NewWriter, Reader.ReadString
# os            — Open, ReadFile, Args, Getenv, Exit, Stdin/Stdout/Stderr
# sort          — Slice, SliceStable, Ints/Strings, Search (sorted)
# slices        — Contains, Sort, Reverse, BinarySearch, Clone, Equal (1.21+)
# maps          — Keys, Values, Clone, Equal, Copy (1.21+)
# sync          — Mutex, RWMutex, WaitGroup, Once, Pool, Cond
# context       — Background, TODO, WithCancel, WithTimeout, WithValue
# encoding/json — Marshal, Unmarshal, Encoder, Decoder
# net/http      — Client, Server, ServeMux, Handler, NewRequestWithContext
# time          — Now, Parse, Format, Sleep, After, Tick, Duration
# errors        — New, Is, As, Unwrap, Join (1.20+)
# log/slog      — structured logging (1.21+) — logger, handlers, attrs
```

## Common Gotchas

### Slice aliasing on append

```bash
# orig := []int{1, 2, 3, 4}
# part := orig[:2]
# part = append(part, 99)        // CORRUPT — writes index 2 of orig if cap > 2
#
# // Fix — clone explicitly:
# part := append([]int{}, orig[:2]...)
# part = append(part, 99)        // safe
```

### Nil-map writes panic

```bash
# var m map[string]int
# m["x"] = 1                      // PANIC
#
# // Fix:
# m = make(map[string]int)
# m["x"] = 1                      // ok
```

### Range loop variable capture (pre-1.22)

```bash
# // Pre-1.22: loop variable is REUSED across iterations.
# var fns []func()
# for i := 0; i < 3; i++ {
#     fns = append(fns, func() { fmt.Println(i) })
# }
# for _, f := range fns { f() }   // prints 3 3 3 in pre-1.22, 0 1 2 in 1.22+
#
# // Pre-1.22 fix — shadow inside the loop:
# for i := 0; i < 3; i++ {
#     i := i                       // new variable per iteration
#     fns = append(fns, func() { fmt.Println(i) })
# }
# // Or pass as arg:
# fns = append(fns, func(i int) func() { return func() { fmt.Println(i) } }(i))
#
# // 1.22+ made this the default — each iteration has fresh per-loop variables.
# // Set go directive to 1.22 in go.mod to opt in.
```

### time.Time zero value is NOT Unix epoch

```bash
# var t time.Time
# t == time.Unix(0, 0)            // false — Unix(0,0) is 1970, t is year 1
# t.IsZero()                      // true
# // Always use IsZero() for "unset" checks.
```

### Interface nil != typed nil

```bash
# var p *Foo                       // p is nil
# var i interface{} = p            // i is NOT nil — it's (type=*Foo, value=nil)
# i == nil                         // FALSE
# // Fix — return a true nil interface value:
# func get() interface{} {
#     var p *Foo
#     if condition { return p }   // returns typed-nil — DANGEROUS
#     return nil                   // returns true nil
# }
```

### defer in a loop

```bash
# // BUG — defers don't run until the function returns.
# for _, path := range paths {
#     f, _ := os.Open(path)
#     defer f.Close()              // accumulates — only runs when caller returns
#     process(f)
# }
#
# // Fix — wrap in a helper or close manually:
# for _, path := range paths {
#     func() {
#         f, _ := os.Open(path)
#         defer f.Close()
#         process(f)
#     }()
# }
```

### `for _ = range` — discard syntax

```bash
# // 1.4+:
# for range time.Tick(time.Second) { /* every second */ }
# // Pre-1.4 you needed:
# for _ = range time.Tick(time.Second) { ... }
# // Modern Go: just `for range`. Drop the `_`.
```

## Performance Tips

### Pre-grow collections

```bash
# // Bad — grows multiple times.
# var xs []int
# for i := 0; i < 1_000_000; i++ { xs = append(xs, i) }
#
# // Better — one allocation.
# xs := make([]int, 0, 1_000_000)
# for i := 0; i < 1_000_000; i++ { xs = append(xs, i) }
#
# // strings.Builder — same idea:
# var b strings.Builder
# b.Grow(estimateBytes)
# for _, s := range parts { b.WriteString(s) }
# out := b.String()
```

### sync.Pool for hot temporaries

```bash
# var bufPool = sync.Pool{ New: func() any { return new(bytes.Buffer) } }
# // In a hot loop / handler:
# b := bufPool.Get().(*bytes.Buffer)
# b.Reset()
# // ... write to b ...
# bufPool.Put(b)
# // Reduces GC pressure for short-lived allocations like serialization buffers.
```

### Struct field ordering for alignment

```bash
# type Bad struct {
#     a bool   //  1 byte + 7 padding to align int64
#     b int64  //  8 bytes
#     c bool   //  1 byte + 7 padding for tail alignment
# }                                                // 24 bytes
#
# type Good struct {
#     b int64  //  8 bytes
#     a bool   //  1 byte
#     c bool   //  1 byte + 6 padding
# }                                                // 16 bytes
# // Order largest-to-smallest. Use `go vet -fieldalignment` from x/tools/go/analysis/passes.
```

### Avoid interface in hot paths

```bash
# // Calling a method through an interface is an indirect call — no inlining,
# // no escape-analysis simplification, and a small allocation if value-typed.
# // For numerical inner loops or sub-microsecond hot paths, use concrete types or generics.
```

## Idioms

### Accept interfaces, return structs

```bash
# // Accepting interfaces makes a function flexible:
# func Read(r io.Reader) ([]byte, error) { ... }
# // Returning concrete types preserves capability — callers can call all methods,
# // and you avoid leaking implementation as part of the API surface.
# func NewBuffer() *Buffer { return &Buffer{...} }
```

### Prefer errors.Is / errors.As

```bash
# // Don't compare with == when wrapping is in play.
# if err == ErrNotFound { ... }                 // BAD — misses wrapped errors
# if errors.Is(err, ErrNotFound) { ... }        // GOOD
```

### Context first arg

```bash
# // Always:
# func DoThing(ctx context.Context, x string) error { ... }
# // Never:
# func DoThing(x string, ctx context.Context) error { ... }
```

### Naming: getters without "Get"

```bash
# // BAD:
# user.GetName()
# // GOOD:
# user.Name()
# // Setters keep "Set":
# user.SetName(s)
# // Boolean getters: IsX, HasX, CanX.
```

## Tips

- Always check errors. `_` discarding errors leads to silent failures and ghost bugs.
- Use `defer` for cleanup (close files, unlock mutexes, cancel contexts). Defers run LIFO.
- `go test -race` catches data races. Run it in CI on every PR.
- Wrap errors with `fmt.Errorf("op: %w", err)` to build error chains. Use `errors.Is`/`errors.As` to inspect.
- Prefer `context.Context` as the first parameter of any function that does I/O or may be cancelled.
- Slices and maps are reference types under the hood. Passing them shares the underlying data.
- `sync.Once` is the safest way to do one-time initialization in concurrent code.
- `go vet` catches common mistakes (printf format errors, unreachable code, fieldalignment). Run alongside tests.
- `staticcheck` catches everything `go vet` misses. Wire it into CI.
- For HTTP servers, ALWAYS set `ReadHeaderTimeout` (and ideally Read/Write/IdleTimeout). The defaults are zero-valued, meaning no timeout.
- For HTTP clients, ALWAYS use a `*http.Client{Timeout:...}` or `context.WithTimeout`. `http.DefaultClient` has no timeout.
- Use `log/slog` (1.21+) over `log` for new code — structured, leveled, attribute-based.
- `go run -race main.go` is fine for local testing; never ship a binary built with `-race` to production (10x slower).
- Pin `toolchain` in `go.mod` to make builds reproducible across machines.
- `go mod tidy` removes unused deps and adds missing ones — run before commit.

## See Also

- polyglot
- rust
- python
- c
- javascript
- typescript
- java
- ruby
- lua
- bash
- make
- webassembly
- json

## References

- [Go Documentation](https://go.dev/doc/) -- getting started, tutorials, and guides
- [Go Language Specification](https://go.dev/ref/spec) -- formal language spec
- [Go Standard Library](https://pkg.go.dev/std) -- stdlib package reference
- [Effective Go](https://go.dev/doc/effective_go) -- idiomatic Go patterns and conventions
- [Go Memory Model](https://go.dev/ref/mem) -- happens-before relationships, atomics, sync primitives
- [Go Module Reference](https://go.dev/ref/mod) -- module system, go.mod, versioning
- [Go Blog](https://go.dev/blog/) -- official articles on features and best practices
- [Go Playground](https://go.dev/play/) -- run and share Go code online
- [Go Wiki](https://go.dev/wiki/) -- community-maintained guides and FAQs
- [pkg.go.dev](https://pkg.go.dev/) -- package discovery and documentation
- [Go Release History](https://go.dev/doc/devel/release) -- changelog for every Go release
- [Russ Cox blog](https://research.swtch.com/) -- deep dives by one of Go's lead designers
- [Go by Example](https://gobyexample.com/) -- short, runnable snippets per topic
- [Dave Cheney's blog](https://dave.cheney.net/) -- pragmatic Go articles, especially on errors and SOLID
- [Damian Gryski's "go-perfbook"](https://github.com/dgryski/go-perfbook) -- performance optimization handbook
- [Uber Go Style Guide](https://github.com/uber-go/guide) -- production-quality conventions
- [Awesome Go](https://github.com/avelino/awesome-go) -- curated list of Go libraries and frameworks
