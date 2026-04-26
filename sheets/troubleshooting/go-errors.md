# Go Errors

Every Go runtime panic, compile error, module error, race-detector output verbatim with cause and fix. The terminal-bound debugging companion: paste the error message, find the exact line, get the canonical fix.

## Setup

Go has two error-signaling mechanisms: explicit `error` returns (idiomatic, recoverable) and `panic` (exceptional, unrecoverable by default). Most application code returns errors. Panics are reserved for programmer errors (nil deref, out-of-bounds), unrecoverable conditions, or as a fallback when an interface contract cannot be honored.

### The error interface

```go
// builtin
type error interface {
    Error() string
}
```

Any type implementing `Error() string` satisfies `error`. Conventional usage:

```go
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("divide by zero")
    }
    return a / b, nil
}

result, err := divide(10, 0)
if err != nil {
    log.Fatalf("divide failed: %v", err)
}
```

### Sentinel errors

Package-level error variables compared with `==` or `errors.Is`:

```go
package mylib

import "errors"

var (
    ErrNotFound  = errors.New("mylib: not found")
    ErrConflict  = errors.New("mylib: conflict")
    ErrForbidden = errors.New("mylib: forbidden")
)

// Caller:
if errors.Is(err, mylib.ErrNotFound) {
    // handle 404
}
```

Stdlib examples: `io.EOF`, `sql.ErrNoRows`, `os.ErrNotExist`, `context.Canceled`, `context.DeadlineExceeded`.

### Error types

A type implementing `error`, examined with `errors.As`:

```go
type ValidationError struct {
    Field string
    Msg   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: %s: %s", e.Field, e.Msg)
}

// Caller:
var verr *ValidationError
if errors.As(err, &verr) {
    fmt.Println(verr.Field)
}
```

Stdlib examples: `*os.PathError`, `*net.OpError`, `*url.Error`, `*json.SyntaxError`.

### Opaque errors

Errors whose only useful operation is `.Error()` for logging — the caller cannot discriminate beyond the surface text. Most `fmt.Errorf` results without `%w` are opaque.

### Wrapping with %w

`fmt.Errorf` with `%w` wraps an underlying error so `errors.Is`/`errors.As` can reach it through the chain:

```go
func loadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("loadConfig %q: %w", path, err)
    }
    // ...
}

// Caller:
_, err := loadConfig("/etc/app.toml")
if errors.Is(err, os.ErrNotExist) {
    // file missing — provision defaults
}
```

Multiple wraps are allowed (Go 1.20+):

```go
return fmt.Errorf("step %d failed: %w; rolled back: %w", n, errA, errB)
```

`errors.Is` and `errors.As` walk the entire DAG.

### errors.Is, errors.As, errors.Unwrap

```go
// errors.Is — sentinel match anywhere in the chain
if errors.Is(err, fs.ErrNotExist) { ... }

// errors.As — first error of given type, fills target
var pathErr *fs.PathError
if errors.As(err, &pathErr) {
    fmt.Println(pathErr.Path, pathErr.Op)
}

// errors.Unwrap — single step (rarely needed in app code)
inner := errors.Unwrap(err)
```

Custom `Is`/`As` methods let your error types match by predicate:

```go
func (e *MyErr) Is(target error) bool {
    t, ok := target.(*MyErr)
    return ok && t.Code == e.Code
}
```

## Runtime Panics

Verbatim panic text with cause and fix. All examples are reproducible — paste, run, see the panic, apply the fix.

### panic: runtime error: invalid memory address or nil pointer dereference

```text
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x...]

goroutine 1 [running]:
main.main()
        /tmp/main.go:8 +0x18
```

Cause — dereferencing a nil pointer, calling a method on a nil receiver that touches fields, accessing a nil map by literal slice/array deref through pointer.

```go
// BROKEN
type User struct{ Name string }
var u *User
fmt.Println(u.Name) // panic
```

```go
// FIXED — initialize before use
u := &User{Name: "alice"}
fmt.Println(u.Name)

// or guard
if u != nil {
    fmt.Println(u.Name)
}
```

Common subtle case — interface holding a typed nil:

```go
// BROKEN
type errReader struct{}
func (errReader) Read(p []byte) (int, error) { return 0, nil }

var r io.Reader
var er *errReader // nil pointer of concrete type
r = er            // r is non-nil interface holding nil *errReader
r.Read(nil)       // panics inside Read if it touches fields
```

### panic: runtime error: index out of range [N] with length M

```text
panic: runtime error: index out of range [3] with length 3

goroutine 1 [running]:
main.main()
        /tmp/main.go:6 +0x39
```

Cause — slice/array index `>= len`. The error reports both the index attempted and the actual length.

```go
// BROKEN
s := []int{1, 2, 3}
fmt.Println(s[3]) // panic
```

```go
// FIXED — bounds check
if i < len(s) {
    fmt.Println(s[i])
}
```

### panic: runtime error: slice bounds out of range [:N] with capacity M

```text
panic: runtime error: slice bounds out of range [:5] with capacity 3
```

Cause — slice expression exceeds capacity (`s[low:high:max]`). Slice bounds cannot exceed cap, not just len.

```go
// BROKEN
s := make([]int, 3, 3)
_ = s[:5] // panic — high > cap
```

```go
// FIXED — grow the slice
s = append(s, 0, 0)
_ = s[:5]
```

Other variants:

```text
panic: runtime error: slice bounds out of range [-1:]
panic: runtime error: slice bounds out of range [:5] with length 3
panic: runtime error: slice bounds out of range [3:1]   // low > high
```

### panic: runtime error: integer divide by zero

```text
panic: runtime error: integer divide by zero
```

Cause — `/` or `%` with integer denominator equal to zero. Constants are caught at compile time; the runtime panic only fires for variable denominators.

```go
// BROKEN
n := 0
_ = 10 / n
```

```go
// FIXED
if n != 0 {
    _ = 10 / n
}
```

Floats produce `+Inf`/`NaN` instead of panicking:

```go
fmt.Println(1.0 / 0.0) // +Inf
```

### panic: runtime error: floating point error

Rare; usually surfaces from CGo or signal-driven contexts. Treat like the integer case but check for inputs producing `NaN` or sub-normal underflow that downstream code mishandles.

### fatal error: concurrent map writes

```text
fatal error: concurrent map writes

goroutine 7 [running]:
main.writer(...)
        /tmp/main.go:11
```

Cause — two goroutines mutate the same map without synchronization. The runtime detects this and aborts (un-recoverable, even with `recover`).

```go
// BROKEN
m := map[string]int{}
go func() { m["a"] = 1 }()
go func() { m["b"] = 2 }()
```

```go
// FIXED — sync.Mutex
var mu sync.Mutex
m := map[string]int{}
go func() { mu.Lock(); m["a"] = 1; mu.Unlock() }()
go func() { mu.Lock(); m["b"] = 2; mu.Unlock() }()

// or sync.Map for read-heavy or set-once-read-many workloads
var sm sync.Map
sm.Store("a", 1)
v, _ := sm.Load("a")
```

### fatal error: concurrent map iteration and map write

```text
fatal error: concurrent map iteration and map write
```

Cause — one goroutine ranges, another writes. Same fix.

```go
// BROKEN
m := map[int]int{1: 1, 2: 2, 3: 3}
go func() { for k := range m { _ = m[k] } }()
go func() { m[4] = 4 }()
```

```go
// FIXED — guard with RWMutex
var mu sync.RWMutex
go func() {
    mu.RLock(); defer mu.RUnlock()
    for k := range m { _ = m[k] }
}()
go func() {
    mu.Lock(); m[4] = 4; mu.Unlock()
}()
```

### fatal error: all goroutines are asleep - deadlock!

```text
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [chan receive]:
main.main()
        /tmp/main.go:7 +0x57
```

Cause — every goroutine is blocked. The runtime detects whole-program deadlock and exits. (It does NOT detect partial deadlock if even one goroutine remains live, e.g. a network listener.)

```go
// BROKEN
ch := make(chan int)
<-ch // nothing will ever send
```

```go
// FIXED — ensure a sender exists
ch := make(chan int)
go func() { ch <- 42 }()
<-ch
```

Common variants — unbuffered channel send with no receiver, two goroutines waiting on each other's channels, sync.WaitGroup with Add() forgotten before Wait().

### panic: runtime error: makeslice: cap out of range

```text
panic: runtime error: makeslice: cap out of range
```

Cause — `make([]T, len, cap)` with `cap < 0`, `cap < len`, or `cap` overflowing platform `int`/`size_t`.

```go
// BROKEN
n := -1
_ = make([]byte, 0, n)
```

```go
// FIXED — validate user-supplied size
if n < 0 || n > 1<<30 {
    return errors.New("bad size")
}
_ = make([]byte, 0, n)
```

### panic: runtime error: makeslice: len out of range

Same family — len negative or absurdly large. Same fix.

### panic: send on closed channel

```text
panic: send on closed channel

goroutine 1 [running]:
main.main()
        /tmp/main.go:8 +0x6e
```

Cause — `ch <- v` after `close(ch)`.

```go
// BROKEN
ch := make(chan int, 1)
close(ch)
ch <- 1
```

```go
// FIXED — only the sender closes; close in defer; never close from receiver
ch := make(chan int, 1)
defer close(ch)
ch <- 1
```

Idiom — for fan-in from multiple producers, use a sync.WaitGroup and a single closer goroutine:

```go
var wg sync.WaitGroup
out := make(chan int)
for _, p := range producers {
    wg.Add(1)
    go func(p Producer) {
        defer wg.Done()
        for v := range p.C {
            out <- v
        }
    }(p)
}
go func() { wg.Wait(); close(out) }()
```

### panic: close of closed channel

```text
panic: close of closed channel
```

Cause — `close(ch)` called twice.

```go
// BROKEN
ch := make(chan int)
close(ch)
close(ch)
```

```go
// FIXED — close exactly once; sync.Once if multiple paths can close
var once sync.Once
closeFn := func() { once.Do(func() { close(ch) }) }
```

### panic: close of nil channel

```text
panic: close of nil channel
```

Cause — `var ch chan T; close(ch)` — channel was never `make`'d.

```go
// BROKEN
var ch chan int
close(ch)
```

```go
// FIXED
ch := make(chan int)
close(ch)
```

### panic: assignment to entry in nil map

```text
panic: assignment to entry in nil map
```

Cause — declared map without `make` or literal initialization.

```go
// BROKEN
var m map[string]int
m["a"] = 1 // panic
```

```go
// FIXED
m := make(map[string]int)
m["a"] = 1

// or
m := map[string]int{"a": 1}
```

Note — reading from a nil map is safe and returns the zero value; only writes panic.

### panic: interface conversion: X is Y, not Z

```text
panic: interface conversion: *main.Cat is *main.Animal, not *main.Dog

goroutine 1 [running]:
main.main()
        /tmp/main.go:18 +0x77
```

Cause — single-value type assertion `v.(T)` where dynamic type is not `T`.

```go
// BROKEN
var i interface{} = "hello"
n := i.(int) // panic
```

```go
// FIXED — comma-ok form
n, ok := i.(int)
if !ok {
    n = 0 // or handle
}

// or type switch
switch v := i.(type) {
case int:
    fmt.Println("int:", v)
case string:
    fmt.Println("string:", v)
default:
    fmt.Println("other")
}
```

### panic: runtime error: hash of unhashable type X

```text
panic: runtime error: hash of unhashable type []int
```

Cause — using a slice, map, or function as a map key, or in a `==` comparison where it falls back to runtime hashing through `interface{}`.

```go
// BROKEN
m := map[interface{}]int{}
m[[]int{1, 2}] = 1 // panic
```

```go
// FIXED — convert to a hashable representation
key := fmt.Sprint([]int{1, 2})
m[key] = 1

// or use array (fixed-size, comparable)
var k [3]int = [3]int{1, 2, 3}
m[k] = 1
```

Compile-time variant — if the type is statically known to be unhashable, the compiler rejects the map declaration: `invalid map key type []int`.

### panic: reflect: call of reflect.Value.X on zero Value

```text
panic: reflect: call of reflect.Value.Field on zero Value
```

Cause — calling a method on a zero `reflect.Value` (returned, e.g., from `reflect.ValueOf(nil)` or after a failed lookup).

```go
// BROKEN
v := reflect.ValueOf(nil)
fmt.Println(v.Field(0)) // panic
```

```go
// FIXED — guard with IsValid
v := reflect.ValueOf(x)
if v.IsValid() && v.Kind() == reflect.Struct {
    fmt.Println(v.Field(0))
}
```

## Compile Errors

Verbatim compiler messages with cause and fix.

### cannot use X (type T) as type Y in argument to Z

```text
./main.go:7:13: cannot use 1 (untyped int constant) as string value in argument to fmt.Println
```

Cause — type mismatch. Go has no implicit numeric-to-string or numeric-to-numeric conversions.

```go
// BROKEN
var s string = 42
```

```go
// FIXED
s := strconv.Itoa(42)
// or
s := fmt.Sprintf("%d", 42)
```

Message variants in newer Go: `cannot use 42 (untyped int constant) as type string`.

### cannot use X (type T) as type Y in field value

Same family for struct literals:

```go
// BROKEN
type P struct{ Name string }
p := P{Name: 42}
```

```go
// FIXED
p := P{Name: strconv.Itoa(42)}
```

### X declared but not used

```text
./main.go:5:6: x declared but not used
```

Cause — local variable bound but never read. Go forbids dead variables. Most often caused by a typo (you read a different variable) or shadowing inside a block.

```go
// BROKEN
func f() {
    x := compute()
    log.Println(y) // typo
}
```

```go
// FIXED
func f() {
    x := compute()
    log.Println(x)
}

// If intentionally unused:
_ = x
```

Shadowing trap:

```go
// BROKEN
err := load()
if err != nil {
    err := fix(err) // shadows outer err — compiler complains about THIS err
    if err != nil {
        return err
    }
}
return err
```

```go
// FIXED
err := load()
if err != nil {
    err = fix(err) // assignment, not declaration
    if err != nil {
        return err
    }
}
return err
```

### X imported and not used

```text
./main.go:4:2: "fmt" imported and not used
```

Cause — package imported but no symbol from it referenced. Same dead-code rule.

```go
// BROKEN
import "fmt"
func main() { println("hi") }
```

```go
// FIXED — remove the import, or use it
import "fmt"
func main() { fmt.Println("hi") }

// Or for side-effect only (init):
import _ "net/http/pprof"
```

### undefined: X

```text
./main.go:5:9: undefined: bar
```

Cause — symbol not declared, not imported, mis-capitalized (Go's export rule: capital first letter is exported, lowercase is package-private), or shadowed.

```go
// BROKEN — wrong case
import "strings"
strings.toLower("X") // undefined: strings.toLower
```

```go
// FIXED
strings.ToLower("X")
```

Other causes — typo, deleted function, imported the wrong package version, build tags exclude the file containing the definition.

### missing return at end of function

```text
./main.go:9:1: missing return at end of function
```

Cause — function declares non-empty return, but a path doesn't return. Go's reachability analysis is conservative.

```go
// BROKEN
func f(x int) int {
    if x > 0 {
        return x
    }
    // falls off end
}
```

```go
// FIXED
func f(x int) int {
    if x > 0 {
        return x
    }
    return 0
}

// Alternative — terminating panic counts as not falling off
func g(x int) int {
    if x > 0 {
        return x
    }
    panic("negative")
}
```

### multiple-value Y in single-value context

```text
./main.go:5:13: multiple-value strconv.Atoi() (value of type (int, error)) in single-value context
```

Cause — assigning a multi-return function to a single value, or passing it as a single argument.

```go
// BROKEN
n := strconv.Atoi("42")
```

```go
// FIXED
n, err := strconv.Atoi("42")
if err != nil { /* ... */ }
```

### syntax error: unexpected X

```text
./main.go:5:6: syntax error: unexpected =, expecting :=
```

Cause — wrong operator at top level (`=` instead of `:=` for new var), missing brace, comma, or semicolon — but Go's auto-semicolon insertion makes some of these surprising.

```go
// BROKEN — opening brace must be on same line
func f()
{
    return
}
```

```go
// FIXED
func f() {
    return
}
```

### no new variables on left side of :=

```text
./main.go:6:5: no new variables on left side of :=
```

Cause — every variable on the left of `:=` is already declared in the same block. `:=` requires at least one new variable.

```go
// BROKEN
x := 1
x := 2
```

```go
// FIXED
x := 1
x = 2

// Or with mixed new/existing:
y, x := 3, 4 // y is new, x is reused
```

### non-name X on left side of :=

```text
./main.go:6:11: non-name p.x on left side of :=
```

Cause — `:=` requires plain identifiers. Selector expressions (`p.x`), index expressions (`s[0]`), and the like must use `=`.

```go
// BROKEN
type P struct{ X int }
p := P{}
p.X := 1
```

```go
// FIXED
p.X = 1
```

### X is not a type

```text
./main.go:8:2: foo is not a type
```

Cause — using a variable, constant, or function name where a type was expected — usually a typo for the type name.

```go
// BROKEN
type point struct{ X, Y int }
var p Point // typo: Point doesn't exist; point does
```

```go
// FIXED
var p point
```

### X is not exported by package Y

```text
./main.go:6:9: cannot refer to unexported name pkg.foo
```

(Also rendered: `pkg.foo undefined (cannot refer to unexported name pkg.foo)`.)

Cause — symbol starts with a lowercase letter and is package-private.

```go
// BROKEN
package zoo
func bark() {} // lowercase

// in another package:
zoo.bark() // error
```

```go
// FIXED — export with a capital
func Bark() {}
zoo.Bark()
```

### cannot range over X

```text
./main.go:6:11: cannot range over n (variable of type int)
```

Cause — `range` works over arrays, slices, strings, maps, channels, and (Go 1.22+) integers and `func(yield func(...))` iterators. Other types are rejected.

```go
// BROKEN (pre-1.22)
for i := range 10 { _ = i }
```

```go
// FIXED — explicit loop, or 1.22+
// 1.22+:
for i := range 10 { _ = i }

// classic:
for i := 0; i < 10; i++ { _ = i }
```

### cannot assign to struct field X.Y in map

```text
./main.go:7:6: cannot assign to struct field m["a"].X in map
```

Cause — values stored in maps are not addressable. You cannot set a single field of a struct directly through a map index expression.

```go
// BROKEN
type P struct{ X int }
m := map[string]P{"a": {}}
m["a"].X = 1
```

```go
// FIXED — replace whole value, or store pointers
p := m["a"]
p.X = 1
m["a"] = p

// or
m2 := map[string]*P{"a": {}}
m2["a"].X = 1
```

### cannot take the address of X

```text
./main.go:5:9: cannot take the address of f()
```

Cause — `&` requires an addressable expression: variable, indirection (`*p`), slice index, struct field of an addressable struct. Map indices, function results, and constants are not addressable.

```go
// BROKEN
m := map[string]int{}
p := &m["a"]
```

```go
// FIXED
v := m["a"]
p := &v
```

### ambiguous selector X.Y

```text
./main.go:13:14: ambiguous selector s.Name
```

Cause — `s` embeds two types that both have a method or field `Name`. The compiler cannot disambiguate.

```go
// BROKEN
type A struct{ Name string }
type B struct{ Name string }
type S struct{ A; B }

var s S
fmt.Println(s.Name) // ambiguous
```

```go
// FIXED — qualify
fmt.Println(s.A.Name)
```

### duplicate field name X

```text
./main.go:4:14: duplicate field Name
```

Cause — same field name listed twice in a struct literal/declaration.

```go
// BROKEN
type P struct {
    Name string
    Name string
}
```

### duplicate method X

```text
./main.go:7:6: method redeclared: P.M
```

Cause — same method declared twice on a type, or two methods only differing in receiver pointer-ness.

```go
// BROKEN
type P struct{}
func (P) M()  {}
func (*P) M() {}  // duplicate
```

### function ends without a return statement

Synonym/older form of "missing return at end of function". Same fix.

### expected expression

```text
./main.go:7:18: expected expression
```

Cause — usually a stray operator or empty argument slot.

```go
// BROKEN
fmt.Println(,)
```

### missing ',' before newline in argument list

```text
./main.go:8:14: missing ',' before newline in argument list
```

Cause — Go's auto-semicolon insertion rules. When a function call spans lines, every argument except the last must end with a comma — including the last visible one before the closing `)`.

```go
// BROKEN
fmt.Println(
    1,
    2  // missing trailing comma
)
```

```go
// FIXED
fmt.Println(
    1,
    2,
)
```

`gofmt` enforces this automatically.

### implicit conversion of untyped int X to string

```text
./main.go:6:14: conversion from untyped int to string yields a string of one rune, not a string of digits (did you mean fmt.Sprint(x)?)
```

Cause (Go 1.15+ this is a vet warning; later treated as an error in some forms) — `string(65)` returns `"A"` (rune at codepoint 65), not `"65"`. Compiler nudges you to be explicit.

```go
// BROKEN intent
n := 65
s := string(n) // s == "A", probably not what you meant
```

```go
// FIXED
s := strconv.Itoa(n)
// or
s := fmt.Sprint(n)
// or, if you really want the rune:
s := string(rune(n))
```

## Type Assertion Errors

The two assertion forms behave very differently.

### Single-return form panics

```go
v := i.(T) // panics if dynamic type != T
```

```text
panic: interface conversion: interface {} is string, not int
```

### Two-return safe form

```go
v, ok := i.(T) // ok=false if mismatch; v is zero value of T
```

### Error type-assert idiom (pre-errors.As)

```go
// pre-1.13 (still works)
if perr, ok := err.(*os.PathError); ok {
    fmt.Println("path:", perr.Path, "op:", perr.Op)
}
```

Modern equivalent:

```go
var perr *fs.PathError
if errors.As(err, &perr) {
    fmt.Println("path:", perr.Path, "op:", perr.Op)
}
```

`errors.As` is preferred — it walks the wrap chain.

### Type switch

```go
switch v := i.(type) {
case nil:
    fmt.Println("nil")
case string:
    fmt.Println("string", v)
case int, int64:
    fmt.Println("int-like")
case io.Reader:
    fmt.Println("reader")
default:
    fmt.Println("other")
}
```

Inside each case, `v` has the case's type (or, for multi-type cases, the original interface type).

## Module / Go Mod Errors

### go: module declares its path as: X but was required as: Y

```text
go: github.com/foo/bar@v0.1.0: parsing go.mod:
        module declares its path as: github.com/foo/bar
                but was required as: github.com/oldfoo/bar
```

Cause — the module's `go.mod` declares one path, your `go.mod` requires another. Happens after a fork or rename.

```bash
# FIXED — replace directive (local), or update require
go mod edit -replace=github.com/oldfoo/bar=github.com/foo/bar@v0.1.0
go mod tidy

# or just
go mod edit -droprequire=github.com/oldfoo/bar
go get github.com/foo/bar
```

### ambiguous import: found package X in multiple modules

```text
ambiguous import: found package github.com/foo/bar/qux in multiple modules:
        github.com/foo/bar v1.0.0 (/.../bar@v1.0.0/qux)
        github.com/foo/bar/qux v0.1.0 (/.../qux@v0.1.0)
```

Cause — two modules both publish the same import path. Pick one with `replace`/`exclude` or a more specific require.

```bash
# FIXED — pin the version you want
go mod edit -require=github.com/foo/bar@v1.0.0
go mod edit -exclude=github.com/foo/bar/qux@v0.1.0
go mod tidy
```

### unknown revision X

```text
go: github.com/foo/bar@v9.9.9: invalid version: unknown revision v9.9.9
```

Cause — tag doesn't exist on the upstream repo, you're behind a proxy, or the tag exists only on a non-default branch.

```bash
# Check available tags
git ls-remote --tags https://github.com/foo/bar

# Use commit SHA pseudo-version
go get github.com/foo/bar@abc1234

# Bypass proxy if firewalled
GOPROXY=direct go get github.com/foo/bar@latest
```

### invalid version: X

```text
go: github.com/foo/bar@1.0.0: invalid version: must be of the form v1.2.3
```

Cause — version must be valid semver with a leading `v`.

```bash
# FIXED
go get github.com/foo/bar@v1.0.0
```

### verifying module: checksum mismatch

```text
verifying github.com/foo/bar@v1.0.0: checksum mismatch
        downloaded: h1:aaaa...
        go.sum:     h1:bbbb...

SECURITY ERROR
This download does NOT match an earlier download recorded in go.sum.
The bits may have been replaced on the origin server, or an attacker may
have intercepted the download attempt.
```

Cause — bytes published at the upstream version differ from the hash recorded in `go.sum`. Could be malicious; could be a force-pushed tag (legitimate but bad-practice); could be a proxy cache desync.

```bash
# Investigate FIRST. If you trust the upstream change:
go clean -modcache
rm go.sum
go mod tidy

# Verify against sum.golang.org
GOSUMDB=sum.golang.org go mod tidy
```

### module github.com/X/Y@vN: invalid version: module contains a go.mod file, so major version must be compatible

```text
module github.com/foo/bar@v2.0.0: invalid version: module contains a go.mod file,
so major version must be compatible: should be v0 or v1, not v2
```

Cause — Go's "semantic import versioning" rule. Modules at major version 2 or above must include the major version in their import path: `github.com/foo/bar/v2`, with `module github.com/foo/bar/v2` in their `go.mod`.

```bash
# FIXED — for the module author:
# 1. Create a v2 branch (or directory)
# 2. Update go.mod: module github.com/foo/bar/v2
# 3. Tag v2.0.0

# FIXED — for the consumer:
go get github.com/foo/bar/v2@v2.0.0
# imports change from "github.com/foo/bar" to "github.com/foo/bar/v2"
```

### module github.com/X/Y/v2 found, but does not contain package github.com/X/Y/v2/Z

```text
github.com/foo/bar/v2 found, but does not contain package github.com/foo/bar/v2/qux
```

Cause — package didn't exist (or had a different name) at the v2 tag; or you accidentally added `/v2` to a sub-package import.

```bash
# Check what packages the version actually contains
go list -m -json github.com/foo/bar/v2
```

### missing go.sum entry for module providing package X

```text
missing go.sum entry for module providing package github.com/foo/bar; to add:
        go mod download github.com/foo/bar
```

Cause — you added or updated an import but didn't run `go mod tidy`/`go mod download`.

```bash
go mod tidy
# or
go mod download github.com/foo/bar
```

### go: cannot find main module, but found .git/config in /path

```text
go: cannot find main module, but found .git/config in /home/u/code
        to create a module there, run:
        go mod init
```

Cause — running `go build`/`go run` outside any `go.mod` tree.

```bash
go mod init github.com/me/project
go mod tidy
```

### go: modules disabled by GO111MODULE=off

Cause — environment forces GOPATH mode.

```bash
unset GO111MODULE
# or
export GO111MODULE=on
```

### go: cannot use path@version syntax in GOPATH mode

Same root cause; same fix.

### go: github.com/X/Y@vN: reading X.go at revision vN: unknown revision X

Tag exists on a branch other than the default; the proxy refuses to materialize it. Use a commit SHA via pseudo-version, or upstream the tag onto the default branch.

### go: github.com/X/Y@vN: invalid pseudo-version

```text
go: github.com/foo/bar@v0.0.0-20240101000000-abc: invalid pseudo-version: timestamp X is before commit time Y
```

Cause — you constructed a pseudo-version by hand and it doesn't match Go's rules (timestamp must precede commit time, the SHA prefix must be 12 chars, etc.).

```bash
# FIXED — let Go compute it
go get github.com/foo/bar@abc1234
# or
go get github.com/foo/bar@<branchname>
```

### build constraints exclude all Go files in /path

```text
package github.com/foo/bar: build constraints exclude all Go files in /path
```

Cause — every `.go` file in the directory has a `//go:build` line that excludes the current GOOS/GOARCH. Or all files have a non-matching `_GOOS.go` suffix.

```bash
# Check active build context
go env GOOS GOARCH

# Inspect file headers
head -5 /path/*.go
```

```go
// BROKEN — only Linux
//go:build linux
package foo

// FIXED — add other tags or remove constraint
//go:build linux || darwin || windows
package foo
```

## go test Errors

### FAIL package N.Ms

```text
FAIL    github.com/foo/bar    0.123s
```

Cause — at least one test in the package failed. Run with `-v` for per-test output.

```bash
go test -v ./...
go test -v -run TestSpecific ./...
go test -count=1 ./...   # disable cache
```

### panic: test timed out after 10m0s

```text
panic: test timed out after 10m0s
        running tests:
                TestSlow (10m0s)

goroutine 18 [chan receive, 10 minutes]:
...
```

Cause — a single test exceeded `-timeout`. Default is 10 minutes; CI often forces shorter.

```bash
go test -timeout 60s ./...
go test -timeout 0 ./...  # no limit
```

```go
// In tests, also use t.Deadline / context for fine control
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

### no test files

```text
?       github.com/foo/bar    [no test files]
```

Cause — the package has no `*_test.go`. The `?` is informational; not an error.

### go test: -run regexp invalid: error parsing regexp

```text
go test: -run regexp invalid: error parsing regexp: missing closing ): `(foo`
```

Cause — `-run` argument is a Go regexp; unbalanced parens or unescaped meta characters.

```bash
go test -run "TestFoo$" ./...
go test -run "TestParent/SubTest$" ./...
go test -run "TestA|TestB" ./...
```

### TestX failed

```text
--- FAIL: TestParse (0.00s)
    parse_test.go:42: expected 1, got 2
FAIL
```

Cause — `t.Errorf`/`t.Fatalf` called. Use `cmp.Diff` for readable struct diffs:

```go
if diff := cmp.Diff(want, got); diff != "" {
    t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
}
```

## Race Detector Output Anatomy

Build/run with `-race` (requires CGO_ENABLED=1):

```bash
CGO_ENABLED=1 go run -race ./...
go test -race ./...
go build -race ./cmd/server
```

Sample output:

```text
==================
WARNING: DATA RACE
Read at 0x00c0000a8000 by goroutine 7:
  main.reader()
      /tmp/main.go:14 +0x44

Previous write at 0x00c0000a8000 by goroutine 6:
  main.writer()
      /tmp/main.go:9 +0x4c

Goroutine 7 (running) created at:
  main.main()
      /tmp/main.go:21 +0x88

Goroutine 6 (finished) created at:
  main.main()
      /tmp/main.go:20 +0x66
==================
Found 1 data race(s)
exit status 66
```

Decoding —

- The shared address (`0x00c0000a8000`) — same in both events.
- The read site (file:line) and the write site.
- The originating goroutines (with creation stacks) — trace where the goroutine started, often the missing synchronization point.
- The exit status `66` is conventional for race-detector-flagged tests.

Triggering race causes —

- Unprotected mutation of shared state (vars, slice elements, struct fields, map entries).
- Reading a value while another goroutine writes — even if both are atomic on hardware (`int64` on 64-bit), Go's memory model demands explicit synchronization.
- Closing over a loop variable in pre-1.22 code (each goroutine sees the latest, racing with the loop's increment).

Fix tools —

- `sync.Mutex`, `sync.RWMutex` for guarded sections.
- `sync/atomic` for primitives.
- channels for ownership transfer.
- `sync.Map` for set-once-read-many maps.

CGO_ENABLED=1 + -race is mandatory; `-race` cannot be combined with `-msan` or `-asan`. The race binary runs ~5–10x slower and uses ~5–10x more memory — never ship to prod.

## CGO Errors

### cgo: command not found

```text
exec: "gcc": executable file not found in $PATH
```

Cause — building a CGo-using package without a C toolchain.

```bash
# macOS
xcode-select --install

# Debian/Ubuntu
sudo apt install build-essential

# Alpine
apk add gcc musl-dev
```

If you don't actually need CGo:

```bash
CGO_ENABLED=0 go build ./...
```

### ld: warning: ignoring file X.dylib, building for macOS-arm64 but attempting to link with file built for unknown-x86_64

Cause — architecture mismatch on macOS (M-series vs Intel) — your CGo dependency was prebuilt for the other arch.

```bash
# Rebuild the dep, or:
GOARCH=amd64 go build ./...     # match the dylib

# Or for fat binaries:
brew reinstall <pkg>            # often provides universal2
```

### linker command failed with exit code 1

Generic — the line above usually has the real cause (undefined symbol, missing lib, arch mismatch).

```bash
# Verbose linker output
go build -ldflags="-v" ./...
```

### could not determine kind of name for C.X

```text
./main.go:10:12: could not determine kind of name for C.foo
```

Cause — `C.foo` referenced in Go but not declared in the preceding `import "C"` comment block.

```go
// BROKEN
/*
#include <stdlib.h>
*/
import "C"
_ = C.atoi // OK
_ = C.bar  // not declared anywhere
```

```go
// FIXED — declare it
/*
#include <stdlib.h>
extern int bar(int);
*/
import "C"
_ = C.bar
```

## database/sql Errors

### sql: no rows in result set

```text
sql: no rows in result set
```

Cause — `QueryRow().Scan(...)` found zero rows. Idiomatically detected via `errors.Is`:

```go
err := db.QueryRow("SELECT id FROM users WHERE email = ?", e).Scan(&id)
if errors.Is(err, sql.ErrNoRows) {
    return ErrNotFound
}
if err != nil { return err }
```

### sql: Scan error on column index N, name X: converting NULL to int is unsupported

```text
sql: Scan error on column index 0, name age: converting NULL to int is unsupported
```

Cause — column is nullable, scan target isn't.

```go
// BROKEN
var age int
db.QueryRow("...").Scan(&age)
```

```go
// FIXED — sql.NullInt64 (or null-aware driver type)
var age sql.NullInt64
db.QueryRow("...").Scan(&age)
if age.Valid { fmt.Println(age.Int64) }
```

### sql: expected N destination arguments in Scan, not M

```text
sql: expected 3 destination arguments in Scan, not 2
```

Cause — column count in SELECT doesn't match Scan args.

```go
// BROKEN
db.QueryRow("SELECT id, name, email FROM u").Scan(&id, &name)
```

```go
// FIXED
var email string
db.QueryRow("SELECT id, name, email FROM u").Scan(&id, &name, &email)
```

### sql: connection is already closed

Cause — using a `*sql.Conn`/`*sql.Tx` after `Close()`/`Commit()`/`Rollback()`. Often a defer ordering bug.

### sql: database is closed

Cause — using `*sql.DB` after `db.Close()`. Make sure shutdown runs after all consumers finish.

### Error 1062: Duplicate entry 'X' for key 'Y' (MySQL)

```text
Error 1062 (23000): Duplicate entry 'alice@example.com' for key 'users.email_unique'
```

Cause — unique constraint violation. Detect by code:

```go
import "github.com/go-sql-driver/mysql"

if me, ok := err.(*mysql.MySQLError); ok && me.Number == 1062 {
    return ErrConflict
}
```

### pq: relation "X" does not exist (PostgreSQL)

```text
pq: relation "users" does not exist
```

Cause — table missing, schema not on `search_path`, or you're connected to the wrong database.

```sql
SELECT current_database(), current_schema();
SHOW search_path;
```

### ERROR: column "X" does not exist (SQLSTATE 42703) (pgx)

Cause — column reference misspelled or quoting issue. Postgres folds unquoted identifiers to lowercase; `"FirstName"` is a different column from `firstname`.

### context deadline exceeded with sql.QueryContext

```text
context deadline exceeded
```

Cause — query took longer than the context allowed. Tune the timeout, add an index, or break the query up.

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
rows, err := db.QueryContext(ctx, q)
```

### driver: bad connection

Cause — underlying network connection died (idle timeout on the server, network partition). The standard library retries automatically once for `Query` but not for transactions.

```go
db.SetConnMaxLifetime(5 * time.Minute) // shorter than server's wait_timeout
db.SetConnMaxIdleTime(1 * time.Minute)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
```

## http / net Errors

### http: connection has been hijacked

Cause — calling `Write`/`WriteHeader` on a `ResponseWriter` after `Hijacker.Hijack()` returned the underlying conn (e.g., for WebSocket upgrade).

```go
// FIXED — once hijacked, write to the net.Conn directly
hj, _ := w.(http.Hijacker)
conn, brw, err := hj.Hijack()
if err != nil { return }
defer conn.Close()
fmt.Fprintf(conn, "...")
brw.Flush()
```

### http: superfluous response.WriteHeader call from main.X

```text
http: superfluous response.WriteHeader call from main.handler (main.go:23)
```

Cause — `WriteHeader` called more than once. Common when you call `WriteHeader(200)` and then a middleware/error path also writes a header.

```go
// BROKEN
w.WriteHeader(http.StatusOK)
fmt.Fprintln(w, "ok")
w.WriteHeader(http.StatusInternalServerError) // ignored, warns
```

```go
// FIXED — write once, branch first
if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
}
w.WriteHeader(http.StatusOK)
fmt.Fprintln(w, "ok")
```

### http: response.Write on hijacked connection

Cause — same as the hijack one. Don't write through `w` after hijacking.

### http: server closed

```text
http: Server closed
```

This is `http.ErrServerClosed`, returned from `srv.ListenAndServe` after `srv.Shutdown` succeeds. Treat as benign:

```go
if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
    log.Fatal(err)
}
```

### Get http://X: context deadline exceeded

Cause — request exceeded `Client.Timeout` or the request `Context`'s deadline.

```go
client := &http.Client{Timeout: 10 * time.Second}
```

For finer control, use `http.NewRequestWithContext` and a per-request context.

### Get http://X: dial tcp X.X.X.X:Y: connect: connection refused

Cause — nothing listening on that host:port. DNS resolved fine; TCP got RST.

```bash
# Verify
ss -tnlp | grep :8080
nc -vz host 8080
```

### Get http://X: x509: certificate signed by unknown authority

Cause — server's TLS cert chain isn't rooted in your system's CA bundle. Common with self-signed certs or private CAs.

```go
// PROD FIX — add your CA to a custom RootCAs pool
caData, _ := os.ReadFile("/etc/ssl/private-ca.pem")
pool := x509.NewCertPool()
pool.AppendCertsFromPEM(caData)
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{RootCAs: pool},
    },
}

// DEV ONLY — disable verification (NEVER in prod)
&tls.Config{InsecureSkipVerify: true}
```

### Get http://X: tls: failed to verify certificate: x509: certificate has expired

Renew the cert. If a fixed historical date is required (e.g. testing past data), set `Time` in `tls.Config`:

```go
tls.Config{Time: func() time.Time { return time.Date(2024,1,1,...) }}
```

### EOF reading response body

Cause — server closed the TCP connection mid-response, often due to a panic or timeout on the server side, an idle-timeout closing a connection just as the client started a request, or a proxy (HAProxy, nginx) terminating early.

```go
// Mitigations
client.Transport = &http.Transport{
    DisableKeepAlives: true, // avoid reusing dead conns
}
// or
http.DefaultTransport.(*http.Transport).IdleConnTimeout = 90*time.Second
```

### use of closed network connection

```text
write tcp X->Y: use of closed network connection
```

Cause — write attempted on a `net.Conn` that's been `Close`d, or after the listener accepted-and-aborted.

### http: request body too large

Cause — `MaxBytesReader` returned its sentinel; payload > limit.

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
```

### http2: stream stopped

Cause — peer reset the HTTP/2 stream (RST_STREAM). Often a server-side cancellation. Retry idempotent requests; check the server.

## encoding/json Errors

### json: cannot unmarshal number into Go struct field X.Y of type string

```text
json: cannot unmarshal number into Go struct field User.Age of type string
```

Cause — JSON `number` does not match the Go `string`-typed target. Either change the field type, or add a custom UnmarshalJSON, or use `json.Number`.

```go
// FIX 1 — match types
type User struct{ Age int `json:"age"` }

// FIX 2 — accept string OR number
type LooseInt int
func (l *LooseInt) UnmarshalJSON(b []byte) error {
    s := strings.Trim(string(b), `"`)
    n, err := strconv.Atoi(s)
    if err != nil { return err }
    *l = LooseInt(n); return nil
}
```

### json: unsupported type: X

```text
json: unsupported type: chan int
json: unsupported type: func()
json: unsupported value: NaN
```

Cause — channels, functions, complex numbers, and `NaN`/`Inf` floats cannot be marshaled.

### json: unknown field "X" (with DisallowUnknownFields)

```go
dec := json.NewDecoder(r)
dec.DisallowUnknownFields()
err := dec.Decode(&v) // returns "json: unknown field \"foo\""
```

Useful for strict APIs that should reject typos.

### json: invalid UTF-8 in string

```text
json: invalid UTF-8 in string: "..."
```

Cause — JSON spec requires UTF-8. Source data has stray bytes.

### json: error calling MarshalJSON for type X

The custom marshaler returned an error; check it.

### unexpected end of JSON input

```text
unexpected end of JSON input
```

Cause — input truncated. Maybe you read 0 bytes (read failed), or the stream wasn't fully consumed before unmarshal.

### invalid character 'X' looking for beginning of value

```text
invalid character '<' looking for beginning of value
```

Cause — input isn't JSON. The classic case is a server returning HTML error page on a JSON endpoint; the leading `<` from `<html>` triggers this.

```go
// Always check Content-Type and status before unmarshalling
if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
    body, _ := io.ReadAll(resp.Body)
    return fmt.Errorf("non-JSON response: %d: %s", resp.StatusCode, body)
}
```

### invalid character 'X' after object key:value pair

Cause — missing comma between fields, trailing comma (illegal in JSON), or stray whitespace inside a string.

## encoding/xml + YAML Errors

### XML

```text
XML syntax error on line 5: element <foo> closed by </bar>
```

Cause — mismatched tags. XML has no auto-close.

### yaml: line N: did not find expected key

```text
yaml: line 3: did not find expected key
```

Cause — usually mixed tabs/spaces (YAML disallows tabs in indentation), or an unquoted value containing `:`.

```yaml
# BROKEN — tab indented
foo:
\tbar: 1
```

```yaml
# FIXED — spaces only
foo:
  bar: 1
```

### yaml: unmarshal errors: line N: cannot unmarshal !!str X into int

Same as JSON's number-vs-string issue. Fix by matching field types or adding a custom unmarshaler.

## context Errors

### context canceled

`context.Canceled` — returned when `cancel()` is called, the parent was canceled, or `context.WithoutCancel` was not used and shutdown propagated.

### context deadline exceeded

`context.DeadlineExceeded` — `WithTimeout`/`WithDeadline` fired.

### Distinguishing the two

```go
err := ctx.Err()
switch {
case errors.Is(err, context.Canceled):
    // user canceled, parent shut down
case errors.Is(err, context.DeadlineExceeded):
    // timeout
}
```

Note — many wrapping errors include "context deadline exceeded" or "context canceled" as their message; always use `errors.Is` rather than string match.

### context.WithoutCancel (Go 1.21+)

Detaches a child context from the parent's cancellation but preserves Values:

```go
// Common pattern: handler ctx canceled when client disconnects,
// but you want to finish the audit log write even so.
ctx2 := context.WithoutCancel(ctx)
go writeAuditLog(ctx2, evt)
```

Other 1.21 helpers — `context.AfterFunc(ctx, f)`, `context.WithDeadlineCause`, `context.WithTimeoutCause`, `context.Cause(ctx)`.

## time / Duration Errors

### time: missing unit in duration X

```text
time: missing unit in duration "5"
```

Cause — `time.ParseDuration` requires a unit suffix.

```go
d, _ := time.ParseDuration("5")     // error
d, _ := time.ParseDuration("5s")    // 5 seconds
d, _ := time.ParseDuration("1h30m") // 1.5 hours
d, _ := time.ParseDuration("250ms")
d, _ := time.ParseDuration("2.5h")
```

Valid units — `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`. No days, weeks, or months (variable length).

### time: invalid duration X

Same family — string parsed but couldn't make sense of the value.

### parsing time "X" as "Y": cannot parse "X" as "Z"

```text
parsing time "2024-01-15T12:00:00Z" as "2006-01-02": cannot parse "T12:00:00Z" as ""
```

Cause — the layout doesn't match the input. Go's time format uses a *reference time* — every value is shorthand for a part of `Mon Jan 2 15:04:05 MST 2006` (which numerically is 01/02 03:04:05PM '06 -0700).

```go
// Common layouts
time.Parse(time.RFC3339, "2024-01-15T12:00:00Z")
time.Parse("2006-01-02", "2024-01-15")
time.Parse("2006-01-02 15:04:05", "2024-01-15 12:00:00")
time.Parse("01/02/2006", "01/15/2024")
time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Mon, 15 Jan 2024 12:00:00 UTC")
```

Why so weird — the reference is engineered so each component takes a unique numeric value: Jan=1, day=2, 03=hour-12, 04=minute, 05=second, 06=year, 7-zone-offset (in -0700 form).

Predefined layouts in `time/`:

```go
time.Layout      // "01/02 03:04:05PM '06 -0700"
time.ANSIC       // "Mon Jan _2 15:04:05 2006"
time.UnixDate    // "Mon Jan _2 15:04:05 MST 2006"
time.RubyDate    // "Mon Jan 02 15:04:05 -0700 2006"
time.RFC822
time.RFC822Z
time.RFC850
time.RFC1123
time.RFC1123Z
time.RFC3339     // "2006-01-02T15:04:05Z07:00"
time.RFC3339Nano // "2006-01-02T15:04:05.999999999Z07:00"
time.Kitchen     // "3:04PM"
time.DateTime    // "2006-01-02 15:04:05" (1.20+)
time.DateOnly    // "2006-01-02"
time.TimeOnly    // "15:04:05"
```

## os / file I/O Errors

### open X: no such file or directory

```text
open /etc/missing.conf: no such file or directory
```

Cause — path doesn't exist. Detect with `errors.Is`:

```go
_, err := os.Open(p)
if errors.Is(err, fs.ErrNotExist) {
    // handle missing
}
```

### open X: permission denied

```text
open /root/secret: permission denied
```

Cause — current user lacks read/write/execute on the file or any parent directory.

```bash
ls -ld /root /root/secret
namei -l /root/secret
```

`errors.Is(err, fs.ErrPermission)`.

### open X: is a directory

```text
open /etc: is a directory
```

Cause — `os.Open` (which calls `OpenFile` with read mode) on a directory works for listing; `os.ReadFile` and most read-as-file operations don't. Use `os.ReadDir` / `os.Stat`.

### open X: too many open files

```text
open /tmp/file: too many open files
```

Cause — process hit `RLIMIT_NOFILE`. Check with `ulimit -n`. Common when not closing files/connections in a hot path.

```go
// Always close
f, err := os.Open(p)
if err != nil { return err }
defer f.Close()
```

```bash
# Inspect
lsof -p <pid> | wc -l
cat /proc/<pid>/limits

# Bump for current shell
ulimit -n 65536

# Persistent (Linux): /etc/security/limits.conf, systemd LimitNOFILE
```

### remove X: directory not empty

Cause — `os.Remove` doesn't recurse. Use `os.RemoveAll`.

```go
os.RemoveAll("/tmp/build") // removes recursively
```

### rename X Y: cross-device link

```text
rename /tmp/foo /var/data/foo: invalid cross-device link
```

Cause — `rename(2)` is atomic only within the same filesystem. Across filesystems (e.g., `/tmp` is tmpfs, `/var` is ext4) you must copy then delete.

```go
import "io"

func crossFsRename(src, dst string) error {
    in, err := os.Open(src); if err != nil { return err }
    defer in.Close()
    out, err := os.Create(dst); if err != nil { return err }
    if _, err := io.Copy(out, in); err != nil { out.Close(); os.Remove(dst); return err }
    if err := out.Close(); err != nil { os.Remove(dst); return err }
    return os.Remove(src)
}
```

### errors.Is(err, fs.ErrNotExist) family

```go
fs.ErrInvalid    // "invalid argument"
fs.ErrPermission // "permission denied"
fs.ErrExist      // "file already exists"
fs.ErrNotExist   // "file does not exist"
fs.ErrClosed     // "file already closed"
```

The historical `os.ErrNotExist` etc. are aliases.

## crypto / TLS Errors

### crypto/rsa: verification error

Cause — signature didn't verify. Wrong key, wrong hash algorithm, or tampered data.

### x509: certificate signed by unknown authority

See HTTP section. Add CA to trust pool, or fix the chain.

### x509: certificate has expired or is not yet valid

Renew the cert. Or, if testing, use a fixed `tls.Config.Time`.

### x509: cannot validate certificate for X because it doesn't contain any IP SANs

```text
x509: cannot validate certificate for 192.0.2.1 because it doesn't contain any IP SANs
```

Cause — modern Go (1.15+) requires the cert's SAN (Subject Alternative Name) extension to list the hostname/IP. CN-only certs are rejected.

```bash
# Inspect
openssl x509 -in cert.pem -noout -text | grep -A1 'Subject Alternative Name'

# Reissue with SANs
openssl req -new -x509 -addext 'subjectAltName = DNS:host.example.com, IP:192.0.2.1' ...
```

Workaround for legacy systems (DEPRECATED, removed in newer Go):

```bash
GODEBUG=x509ignoreCN=0 ./app
```

### x509: certificate relies on legacy Common Name field, use SANs instead

Same root cause; same fix — reissue with SANs.

### tls: failed to verify certificate

Wrap of the above. Always include the inner error.

### tls: bad record MAC

Cause — encrypted-record integrity check failed. Possible causes: pre-shared key mismatch (rare), middleware corruption, or version downgrade attacks. Most commonly a misconfigured load balancer.

### tls: handshake failure

Cause — protocol negotiation failed. Mismatched TLS versions, cipher suites, or curve sets.

```bash
openssl s_client -connect host:443 -tls1_2
```

### tls: oversized record received with length N

Cause — peer is not actually speaking TLS, or sent a non-TLS preamble (e.g., HTTP error page). Typical with HTTP-on-HTTPS-port mistakes.

## net Errors

### dial tcp X: connect: connection refused

See HTTP section. Nothing listening, or firewall RST.

### dial tcp X: i/o timeout

Cause — TCP SYN unanswered. Network drop, packet filter, or wrong route.

```bash
mtr host
traceroute host
tcpdump -i any host X and port Y
```

### lookup X: no such host

Cause — DNS resolver returned NXDOMAIN.

```bash
dig +short X
getent hosts X
cat /etc/resolv.conf
```

### lookup X on Y:53: read udp X->Y: i/o timeout

Cause — DNS UDP packet to nameserver `Y` timed out.

### bind: address already in use

```text
listen tcp :8080: bind: address already in use
```

Cause — another process holds the port. Or the same process restarting before TIME_WAIT clears (use `SO_REUSEADDR`).

```bash
ss -tnlp | grep :8080
lsof -i :8080
```

```go
// Go enables SO_REUSEADDR on Listen by default; manual fix:
import "syscall"
lc := net.ListenConfig{Control: func(network, address string, c syscall.RawConn) error {
    return c.Control(func(fd uintptr) {
        syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
    })
}}
ln, _ := lc.Listen(ctx, "tcp", ":8080")
```

### accept tcp X: too many open files

Same root cause as os/file's variant — RLIMIT_NOFILE.

## Reflect Errors

### reflect: call of reflect.Value.X on zero Value

Always guard:

```go
v := reflect.ValueOf(x)
if !v.IsValid() { return }
```

### reflect.Value.Interface: cannot return value obtained from unexported field or method

```text
reflect.Value.Interface: cannot return value obtained from unexported field or method
```

Cause — Go's reflection respects export rules; you can read unexported fields with `Value.Field`, but you cannot pass them through `.Interface()` (which would defeat the rule).

```go
// Workaround — unsafe.Pointer (DANGEROUS, breaks invariants)
import "unsafe"

f := v.Field(0)
ptr := unsafe.Pointer(f.UnsafeAddr())
real := reflect.NewAt(f.Type(), ptr).Elem()
val := real.Interface()
```

### reflect: call using interface as type X

Cause — calling a reflect-bound method/function with the wrong concrete arg.

## Goroutine Leaks

### Detection

Import the pprof handlers:

```go
import _ "net/http/pprof"

go http.ListenAndServe("localhost:6060", nil)
```

Snapshot the goroutine profile:

```bash
curl -s 'http://localhost:6060/debug/pprof/goroutine?debug=2' > goroutines.txt
```

Or use `runtime.NumGoroutine()` to track count over time.

### Common patterns

Goroutine waiting on channel that never receives:

```go
// BROKEN
ch := make(chan int)
go func() {
    v := <-ch          // waits forever if no one sends
    process(v)
}()
return // sender path never executed
```

Fix — close the channel from the producer side, or use a context to bail.

```go
// FIXED
go func() {
    select {
    case v := <-ch:
        process(v)
    case <-ctx.Done():
        return
    }
}()
```

Goroutine on response body Read after the request was canceled:

```go
// Always drain and close, ideally with a timeout
defer func() {
    io.Copy(io.Discard, resp.Body)
    resp.Body.Close()
}()
```

Goroutine waiting on context that's never canceled — the parent forgot `defer cancel()`:

```go
// BROKEN
ctx, _ := context.WithCancel(parent) // discarded cancel = leak
go work(ctx)

// FIXED
ctx, cancel := context.WithCancel(parent)
defer cancel()
```

`go vet` flags discarded cancel functions when they're returned from `WithCancel`/`WithTimeout`/`WithDeadline`.

## go vet Common Findings

### X passes lock by value: T contains sync.Mutex

```text
./main.go:12:6: passes lock by value: main.Counter contains sync.Mutex
```

Cause — copying a struct containing `sync.Mutex` (or any field that is or embeds one). The copy has its own zeroed mutex and protects nothing.

```go
// BROKEN
type Counter struct {
    sync.Mutex
    n int
}
func bump(c Counter) { c.Lock(); c.n++; c.Unlock() }
```

```go
// FIXED — pass pointer
func bump(c *Counter) { c.Lock(); c.n++; c.Unlock() }
```

### Errorf format %X has arg of wrong type Y

```text
./main.go:6:13: Errorf format %d has arg "x" of wrong type string
```

Cause — `%d` got a string. Match verbs to types.

```go
fmt.Errorf("count = %d", n)        // n must be int-like
fmt.Errorf("name = %q", s)         // s must be string
fmt.Errorf("err: %w", err)         // err must be error
fmt.Errorf("got = %v, want = %v", a, b) // anything
```

### the loop variable X has the same address each iteration (pre-1.22)

```text
./main.go:8:8: the loop variable i has the same address each iteration
```

Cause (before Go 1.22) — `for i := range xs` reused a single `i`. Capturing `&i` in closures launched as goroutines was a sharp edge.

```go
// BROKEN (pre-1.22)
for i := 0; i < 5; i++ {
    go func() { fmt.Println(i) }() // all see final i
}
```

```go
// FIXED (any version)
for i := 0; i < 5; i++ {
    i := i // shadow
    go func() { fmt.Println(i) }()
}

// Or pass as argument
for i := 0; i < 5; i++ {
    go func(i int) { fmt.Println(i) }(i)
}

// Or upgrade — Go 1.22+ scopes loop vars per iteration automatically
```

### result of fmt.Sprintf may be discarded

Direct `fmt.Sprintf("...")` with no use is dead code.

### unreachable code

Code after `return`, `panic`, `os.Exit`, or an infinite `for {}` is unreachable.

## golangci-lint / staticcheck Findings

### SA1019 — using deprecated API

```text
SA1019: io/ioutil has been deprecated since Go 1.16 and an alternative has been available since Go 1.16: ...
```

Replacements — `ioutil.ReadFile` → `os.ReadFile`, `ioutil.ReadAll` → `io.ReadAll`, `ioutil.NopCloser` → `io.NopCloser`, `ioutil.TempFile` → `os.CreateTemp`, `ioutil.TempDir` → `os.MkdirTemp`.

### S1000 — should use a simple channel send/receive

```text
S1000: should use a simple channel send/receive instead of select with a single case
```

```go
// Found
select {
case msg := <-ch:
    process(msg)
}

// Replace
msg := <-ch
process(msg)
```

### U1000 — unused

`U1000: func unused is unused`. Delete or use.

### typecheck errors

staticcheck won't run if the package doesn't compile. Fix `go build` errors first.

## go build / go run / go install Errors

### go: cannot find main module

Same as the mod-error; `go mod init` first.

### build constraints exclude all Go files in /path

See module section.

### package X is not in std (in /usr/local/go/src/X)

```text
package github.com/foo/bar is not in std (in /usr/local/go/src/github.com/foo/bar)
```

Cause — running outside a module, falling back to GOPATH. The path doesn't match an installed package.

```bash
cd /your/module
go mod init example.com/me/proj
go get github.com/foo/bar
```

### no Go files in /path

```text
no Go files in /path/to/dir
```

Cause — directory exists but contains no `.go` files (or only `_test.go` files when not running `go test`).

### import cycle not allowed

```text
import cycle not allowed
        package github.com/me/a
                imports github.com/me/b
                imports github.com/me/a
```

Cause — circular dependency between packages. Go forbids this. Refactor — extract the shared types into a third package both depend on, or move the cycling functions.

### package X imports Y: import cycle not allowed

Same family. The diagnostic shows the full cycle path.

## Generics (1.18+) Errors

### type X has no field or method Y

```text
./main.go:8:11: T does not satisfy interface {Method()}: T has no method Method
```

Cause — generic constraint requires a method/field the type argument lacks.

```go
// BROKEN
type Stringer interface{ String() string }
func Print[T Stringer](v T) { fmt.Println(v.String()) }

Print(42) // int has no String()
```

```go
// FIXED — pass a type that satisfies, or add a wrapper
type intStr int
func (i intStr) String() string { return strconv.Itoa(int(i)) }
Print(intStr(42))
```

### X does not satisfy comparable

```text
T does not satisfy comparable
```

Cause — using `comparable` constraint on a type that contains slices, maps, or functions.

### cannot use generic type X without instantiation

```text
cannot use generic type Stack without instantiation
```

```go
// BROKEN
type Stack[T any] struct{ ... }
var s Stack
```

```go
// FIXED
var s Stack[int]
```

### type parameters must be enclosed in [...]

Old typo for `[T any]` syntax. Use brackets, not parens.

### constraint type elements must be comparable

Type-set unions including non-comparable types can't combine with `comparable`.

## defer / Recover Patterns

### Captured panic via recover

```go
func safeCall(f func()) (err error) {
    defer func() {
        if r := recover(); r != nil {
            switch x := r.(type) {
            case error:
                err = x
            case string:
                err = errors.New(x)
            default:
                err = fmt.Errorf("panic: %v", r)
            }
        }
    }()
    f()
    return nil
}
```

Recover only works in a deferred function, on the same goroutine. Recover in `main` can't save you from a panic in another goroutine.

### deferred function must be called as a function

```text
./main.go:5:9: defer requires function call, not conversion
```

Cause — `defer T(x)` looks like a deferred function call but is a type conversion. Wrap in a function.

```go
// BROKEN
defer string(x)
```

```go
// FIXED
defer func() { _ = string(x) }()
```

### Defer in loop — closing all files at function end

```go
// BROKEN — closes all files only when function returns; OK for small loops, leaks for big ones
for _, p := range paths {
    f, err := os.Open(p)
    if err != nil { return err }
    defer f.Close()       // accumulates
    process(f)
}
```

```go
// FIXED — close per iteration via inner function
for _, p := range paths {
    if err := func() error {
        f, err := os.Open(p)
        if err != nil { return err }
        defer f.Close()
        return process(f)
    }(); err != nil {
        return err
    }
}
```

### No recover before panic

`recover()` only returns non-nil if it's called from a deferred function during a panic. Calling it elsewhere returns nil and is a no-op.

```go
// BROKEN — recover at top, no panic active
if r := recover(); r != nil { ... } // never fires
panic("oops")
```

```go
// FIXED — wrap in deferred fn
defer func() {
    if r := recover(); r != nil { handle(r) }
}()
panic("oops")
```

## Channel Errors / Patterns

### panic: send on closed channel

See runtime panics.

### panic: close of closed channel

See runtime panics.

### panic: close of nil channel

See runtime panics.

### panic: receive from nil channel — actually deadlocks

`var ch chan int; <-ch` doesn't panic; it blocks forever. If the whole program is blocked, the runtime emits the all-goroutines-asleep deadlock.

### Comma-ok receive

```go
v, ok := <-ch
if !ok {
    // channel closed and drained
}
```

Use this in pipeline stages to detect closure cleanly:

```go
for {
    v, ok := <-ch
    if !ok { return }
    process(v)
}

// Idiomatic — range
for v := range ch { process(v) }
```

### Fan-out / fan-in leak avoidance

```go
func fanIn(ctx context.Context, srcs ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    for _, src := range srcs {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for v := range c {
                select {
                case out <- v:
                case <-ctx.Done():
                    return
                }
            }
        }(src)
    }
    go func() { wg.Wait(); close(out) }()
    return out
}
```

Always include a `<-ctx.Done()` arm in any goroutine that might block on a slow receiver.

## sync Package Patterns

### sync.Mutex misuse

Copy-by-value:

```go
// BROKEN
type T struct{ mu sync.Mutex; n int }
t1 := T{}
t2 := t1            // t2.mu is a fresh, independent mutex
go func() { t1.mu.Lock(); ...; t1.mu.Unlock() }()
go func() { t2.mu.Lock(); ...; t2.mu.Unlock() }() // unrelated locks
```

Pass `*T` everywhere. `go vet` catches most of these via `copylocks`.

Double-Lock without Unlock — `mu.Lock(); mu.Lock()` deadlocks immediately (Go's mutex is non-reentrant).

```go
// FIXED — restructure to avoid recursion under lock
func (s *S) outer() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.innerLocked()
}

func (s *S) innerLocked() { /* assumes mu held */ }
```

### sync.Map gotchas

- Optimized for "set once, read many" or "disjoint key sets per goroutine". For frequent updates of shared keys, plain `sync.RWMutex + map` is faster.
- No `len()`. Iterate with `Range`, but Range is not a snapshot — concurrent stores may or may not be observed.
- Values are `any`; type-assert at read sites.

### sync.WaitGroup ordering rule

```text
sync: WaitGroup is reused before previous Wait has returned
```

`Add` must happen-before `Wait`. Specifically — `Add(N)` before launching goroutines, never `Add(1)` inside the goroutine body.

```go
// BROKEN
var wg sync.WaitGroup
go func() {
    wg.Add(1)         // RACE: Wait may already see counter==0
    defer wg.Done()
    work()
}()
wg.Wait()
```

```go
// FIXED
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    work()
}()
wg.Wait()
```

### sync.Once panic

If the function passed to `once.Do` panics, the panic propagates and `Once` is *still considered done* — subsequent `Do(f)` calls do nothing. If you want retries, build a custom struct with mutex and an "initialized" flag.

## Common Gotchas — broken→fixed pairs

### 1. Loop variable capture in goroutines (pre-1.22)

```go
// BROKEN
for i := 0; i < 5; i++ {
    go func() { fmt.Println(i) }() // prints 5 five times
}
```

```go
// FIXED
for i := 0; i < 5; i++ {
    i := i
    go func() { fmt.Println(i) }()
}
// or upgrade to Go 1.22+
```

### 2. Range loop with goroutines launching with stale i

```go
// BROKEN
for i, item := range items {
    go func() { handle(i, item) }()
}
```

```go
// FIXED
for i, item := range items {
    i, item := i, item
    go func() { handle(i, item) }()
}
```

### 3. Forgetting comma-ok on map access

```go
// BROKEN — silently uses zero value
v := m["nope"]
process(v)
```

```go
// FIXED
v, ok := m["nope"]
if !ok {
    return ErrMissing
}
process(v)
```

### 4. Using == on slices

```go
// BROKEN — compile error
a, b := []int{1,2}, []int{1,2}
if a == b { ... }
```

```text
./main.go:5:6: invalid operation: a == b (slice can only be compared to nil)
```

```go
// FIXED — slices.Equal (1.21+) or reflect.DeepEqual
import "slices"
if slices.Equal(a, b) { ... }
```

### 5. String conversion: []byte vs []rune

```go
s := "héllo"
fmt.Println(len(s))          // 6 (UTF-8 byte count)
fmt.Println(len([]byte(s)))  // 6
fmt.Println(len([]rune(s)))  // 5 (codepoint count)

// Iterating
for i, b := range []byte(s) { ... } // byte index, byte value
for i, r := range s         { ... } // byte index, rune
for i, r := range []rune(s) { ... } // rune index, rune
```

`utf8.RuneCountInString(s)` is the explicit codepoint count; cheaper than `len([]rune(s))`.

### 6. Implicit copy of mutex in struct literal

```go
// BROKEN — go vet: "passes lock by value"
type C struct { sync.Mutex; n int }
arr := []C{{}, {}, {}}
arr[0].Lock() // each element has its own mutex; OK
fn := func(c C) { c.Lock() } // BROKEN — copy
```

Fix — always `*C`.

### 7. for i := range slice vs for i, v := range slice

```go
for i := range s { ... }       // i = index
for i, v := range s { ... }    // i = index, v = value
for _, v := range s { ... }    // value only

// Maps
for k := range m { ... }
for k, v := range m { ... }

// Channels
for v := range ch { ... }      // until ch is closed

// Strings — yields (byteIndex, rune)
for i, r := range s { ... }
```

### 8. Returning pointer to local variable

```go
// FINE — escape analysis allocates on heap automatically
func newP() *int {
    x := 42
    return &x
}
```

C programmers rejoice — this is safe in Go. The compiler proves the variable escapes and allocates accordingly.

### 9. Comparing interface{} with ==

```go
var a interface{} = int(1)
var b interface{} = int64(1)
fmt.Println(a == b) // false — different concrete types

var c interface{} = []int{1}
var d interface{} = []int{1}
// fmt.Println(c == d) // PANIC: hash of unhashable type []int
```

Two interfaces compare equal iff *both* their concrete types and values are equal. With unhashable concrete types, `==` panics.

### 10. select { default: } burning CPU

```go
// BROKEN — busy loop
for {
    select {
    case v := <-ch:
        handle(v)
    default:
        // nothing — burns 100% CPU
    }
}
```

```go
// FIXED — block on receive
for v := range ch {
    handle(v)
}

// Or include a timeout
for {
    select {
    case v := <-ch:
        handle(v)
    case <-time.After(100 * time.Millisecond):
        // tick
    case <-ctx.Done():
        return
    }
}
```

### 11. Multiple goroutines closing the same channel

```go
// BROKEN — racing closes
go func() { ...; close(ch) }()
go func() { ...; close(ch) }() // panic on second close
```

```go
// FIXED — sync.Once or single closer
var once sync.Once
go func() { ...; once.Do(func() { close(ch) }) }()
go func() { ...; once.Do(func() { close(ch) }) }()
```

### 12. log package vs custom loggers

`log.Print*` and `log.Logger` use an internal mutex — concurrent calls are safe. But many third-party loggers write to a shared `io.Writer` directly without locking; this corrupts output. Always check the doc for "safe for concurrent use".

### 13. Time-zone conversion

```go
// BROKEN — assumes local TZ
t, _ := time.Parse("2006-01-02 15:04:05", "2024-01-15 12:00:00")
// t.Location() is time.UTC, but you may have meant local

// FIXED — be explicit
loc, _ := time.LoadLocation("America/New_York")
t, _ := time.ParseInLocation("2006-01-02 15:04:05", "2024-01-15 12:00:00", loc)
```

### 14. Slice append aliasing

```go
// BROKEN — append may mutate the backing array
a := []int{1,2,3,4}
b := a[:2]
b = append(b, 99)        // overwrites a[2]
fmt.Println(a)           // [1 2 99 4]
```

```go
// FIXED — three-index slice to cap the new slice's capacity
b := a[:2:2]
b = append(b, 99)        // forced to allocate fresh
fmt.Println(a)           // [1 2 3 4]
```

### 15. nil interface vs interface holding nil

```go
type MyErr struct{ Msg string }
func (e *MyErr) Error() string { return e.Msg }

func compute() error {
    var e *MyErr // typed nil
    return e     // returns NON-NIL interface
}

if err := compute(); err != nil {
    // ALWAYS taken — typed nil != untyped nil interface
    fmt.Println("got error", err) // prints "<nil>" but enters branch
}
```

Fix — return untyped nil explicitly, or check `e != nil` before returning.

```go
func compute() error {
    var e *MyErr
    if shouldFail { e = &MyErr{"oops"} }
    if e != nil { return e }
    return nil // untyped nil
}
```

## Debugging

### delve

```bash
dlv debug ./cmd/server -- --flag=value
dlv test ./pkg/foo
dlv attach <pid>
dlv exec ./build/binary
dlv core ./build/binary core.dump
```

Inside delve — `b main.f` (breakpoint), `c` (continue), `n` (next), `s` (step), `p var` (print), `goroutines`, `goroutine N`, `bt` (backtrace). See the dedicated `delve` cheatsheet.

### pprof — CPU profile

```go
import _ "net/http/pprof"
go http.ListenAndServe("localhost:6060", nil)
```

```bash
go tool pprof -http=:8080 'http://localhost:6060/debug/pprof/profile?seconds=30'
go tool pprof -http=:8080 'http://localhost:6060/debug/pprof/heap'
go tool pprof -http=:8080 'http://localhost:6060/debug/pprof/goroutine'
go tool pprof -http=:8080 'http://localhost:6060/debug/pprof/block'
go tool pprof -http=:8080 'http://localhost:6060/debug/pprof/mutex'
go tool pprof -http=:8080 'http://localhost:6060/debug/pprof/allocs'
```

Inside pprof — `top`, `top10 -cum`, `list <fn>`, `web`, `peek <fn>`.

### GODEBUG flags

```bash
GODEBUG=gctrace=1 ./app          # GC trace per cycle to stderr
GODEBUG=schedtrace=1000 ./app    # scheduler dump every 1000ms
GODEBUG=allocfreetrace=1 ./app   # extreme — every alloc/free
GODEBUG=cgocheck=2 ./app         # CGo pointer-passing strict mode
GODEBUG=netdns=go ./app          # Force pure-Go DNS resolver
GODEBUG=netdns=cgo ./app         # Force libc resolver
GODEBUG=http2debug=2 ./app       # HTTP/2 frame logging
GODEBUG=tlsrsakex=1 ./app        # Restore RSA key exchange cipher (legacy)
GODEBUG=x509ignoreCN=0 ./app     # (deprecated) Allow legacy CN-only certs
```

### runtime/trace

```go
import "runtime/trace"

f, _ := os.Create("trace.out")
defer f.Close()
trace.Start(f)
defer trace.Stop()
// run workload
```

```bash
go tool trace trace.out
```

Renders goroutines, GC, syscalls, blocking — invaluable for latency questions.

### -race build

`go build -race`, `go test -race`. Requires CGO_ENABLED=1. ~5-10x slower / larger binary. Race-detector output is documented above.

### -msan build

Memory sanitizer (clang-style); detects uninitialized reads. Linux only, requires CGO_ENABLED=1 and clang.

## Idioms

### Wrap with %w

```go
// Always include context
return nil, fmt.Errorf("openConfig %q: %w", path, err)
```

### errors.Is / errors.As

```go
if errors.Is(err, fs.ErrNotExist) { ... }

var pe *fs.PathError
if errors.As(err, &pe) { ... }
```

### Sentinel errors

```go
var ErrNotFound = errors.New("not found")

if errors.Is(err, ErrNotFound) { ... }
```

### Custom error types for matching

```go
type ValidationError struct {
    Field string
    Reason string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

func (e *ValidationError) Is(t error) bool {
    _, ok := t.(*ValidationError)
    return ok
}
```

### Never silently swallow errors

```go
// BAD
result, _ := riskyCall()

// GOOD
result, err := riskyCall()
if err != nil {
    return fmt.Errorf("risky: %w", err)
}
```

### Defer for cleanup

```go
f, err := os.Open(p)
if err != nil { return err }
defer f.Close()

mu.Lock()
defer mu.Unlock()

start := time.Now()
defer func() { metrics.observe(time.Since(start)) }()
```

### Range-over-channel loops

```go
for v := range ch {
    process(v)
}
// loop exits cleanly when ch is closed; no comma-ok needed
```

### Context propagation

Every blocking call that takes a `context.Context` should receive one from the call chain — never `context.Background()` deep in business logic.

```go
func (s *Service) Query(ctx context.Context, id string) (*Row, error) {
    return s.db.QueryRowContext(ctx, "SELECT ... WHERE id=$1", id)
}
```

### Idiomatic structuring of error returns

```go
// Constructor pattern
func New(opts ...Option) (*T, error) {
    t := &T{}
    for _, o := range opts {
        if err := o(t); err != nil {
            return nil, fmt.Errorf("new: %w", err)
        }
    }
    return t, nil
}

// Multi-step pipeline
func process(ctx context.Context, in Input) (Output, error) {
    a, err := stepA(ctx, in)
    if err != nil { return Output{}, fmt.Errorf("stepA: %w", err) }

    b, err := stepB(ctx, a)
    if err != nil { return Output{}, fmt.Errorf("stepB: %w", err) }

    c, err := stepC(ctx, b)
    if err != nil { return Output{}, fmt.Errorf("stepC: %w", err) }

    return c, nil
}
```

### errors.Join (Go 1.20+)

Combine multiple non-nil errors:

```go
var errs []error
for _, p := range paths {
    if err := process(p); err != nil {
        errs = append(errs, fmt.Errorf("%s: %w", p, err))
    }
}
return errors.Join(errs...) // nil if errs empty
```

`errors.Is`/`errors.As` walk joined errors transparently.

### Logging errors at the boundary

Log once, at the outermost layer that owns the context:

```go
// Library code returns errors
func (s *Service) Do() error { ... }

// HTTP handler logs and translates to status
if err := svc.Do(); err != nil {
    log.Printf("svc.Do: %v", err)
    http.Error(w, "internal", 500)
    return
}
```

Avoid logging *and* returning the same error — duplicates spam.

### Panic with structure, not strings

```go
type InvariantError struct {
    Pkg, Func, Detail string
}
func (e *InvariantError) Error() string {
    return fmt.Sprintf("invariant: %s.%s: %s", e.Pkg, e.Func, e.Detail)
}

panic(&InvariantError{Pkg: "scheduler", Func: "Tick", Detail: "negative time"})
```

Recover sites can `errors.As` for typed handling.

## See Also

- go
- gomod
- polyglot
- delve
- troubleshooting/python-errors
- troubleshooting/javascript-errors
- troubleshooting/rust-errors

## References

- Go specification: https://go.dev/ref/spec
- Go Common Mistakes: https://go.dev/wiki/CommonMistakes
- The Go Blog (errors, modules, generics): https://go.dev/blog/
- Effective Go: https://go.dev/doc/effective_go
- Go Memory Model: https://go.dev/ref/mem
- The Go Programming Language Module Reference: https://go.dev/ref/mod
- Race Detector documentation: https://go.dev/doc/articles/race_detector
- Diagnostics: https://go.dev/doc/diagnostics
- pprof reference: https://github.com/google/pprof/blob/main/doc/README.md
- staticcheck checks: https://staticcheck.dev/docs/checks/
- net/http package: https://pkg.go.dev/net/http
- database/sql tutorial: https://go.dev/doc/tutorial/database-access
- crypto/tls: https://pkg.go.dev/crypto/tls
- context: https://pkg.go.dev/context
- errors: https://pkg.go.dev/errors
- runtime: https://pkg.go.dev/runtime
- runtime/trace: https://pkg.go.dev/runtime/trace
- sync: https://pkg.go.dev/sync
- reflect: https://pkg.go.dev/reflect
- time: https://pkg.go.dev/time
