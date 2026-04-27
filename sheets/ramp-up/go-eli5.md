# Go — ELI5

> Go is C with seatbelts, a built-in engine, and a factory full of tiny workers who pass notes to each other on a conveyor belt.

## Prerequisites

(basic programming helps — knowing what a variable, a function, and a loop are)

If you have never written a single line of code in your life, this sheet will still mostly make sense, because we explain every word as it shows up. But it will be a lot easier if you have at least poked at one other programming language for an hour or so. If the words "variable" and "function" do not feel like aliens to you, you are ready.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is Go

### Imagine a really stubborn boring bicycle

Picture the world's most boring bicycle. It has two wheels. It has handlebars. It has pedals. It has a seat. That is it. No fancy gears. No electric motor. No suspension. No leather grips. No carbon-fibre nonsense.

A lot of people would look at this bicycle and say "this is too plain, give me something cool." But this bicycle has one secret superpower: **it never breaks.** You can leave it in the rain. You can drop it. You can hand it to a five-year-old. It just keeps going. It is the most boring and the most reliable bicycle on Earth.

**Go is that bicycle.**

Go is a programming language that on purpose has fewer features than its competitors. It does not have classes the way Java has classes. It does not have templates the way C++ has templates. It does not have macros the way Rust has macros. It does not have decorators the way Python has decorators. It does not even have a ternary operator like `x ? a : b`. It is plain. It is boring. It is small. And on purpose.

### Why would anyone want a boring language?

Three engineers at Google made Go because they were tired. They were tired of waiting for C++ programs to compile. They were tired of wading through giant Java codebases nobody could understand. They were tired of fighting their tools to do simple things.

They wanted a language that:

- Compiles in **seconds**, not minutes or hours.
- Produces a **single binary** you can copy onto a server with `scp` and run, no runtime required.
- Has **garbage collection** so you don't have to remember to free memory.
- Has **goroutines** (lightweight threads) and **channels** so writing programs that do many things at once feels easy.
- Has a **giant standard library** so you don't have to download a hundred packages just to make an HTTP server.
- Has **one and only one** way to format your code, enforced by a tool called `gofmt`.

That last one is huge. In other languages people argue for hours about whether to use tabs or spaces, where to put braces, how long lines should be. In Go those debates do not happen. The tool `gofmt` reformats your code for you. Everyone's code looks the same. You stop arguing and start shipping.

### The five-second elevator pitch

Go is **C with seatbelts and a built-in engine.**

- Like C, it is small and fast and produces a tight static binary.
- Unlike C, it has garbage collection, so you don't have to call `malloc` and `free` yourself and panic when you forget.
- Like C, it gives you pointers, so you can be efficient when you need to.
- Unlike C, it gives you slices, maps, strings, channels, and goroutines built right into the language.
- Unlike C++, it compiles in seconds even on huge codebases. The Kubernetes codebase is about two million lines of Go and it compiles in a couple of minutes from scratch and in seconds when you change one file.

If you remember nothing else, remember this: **Go is the language people pick when they want to ship a fast, simple, reliable backend service that they don't have to babysit.** Most of the cloud infrastructure tools you have heard of — Docker, Kubernetes, Terraform, Prometheus, Grafana, etcd, Caddy, Hugo, CockroachDB, InfluxDB, Vault, Consul — are written in Go. That is not an accident. They picked Go because Go is good at exactly that job.

### The conveyor belt and the tiny workers

Here is the picture you should keep in your head for Go:

```
                            +-----------+
       you put a thing -->  |           |  --> some other worker
                            | goroutine |      picks it up
       in one end of  -->   |           |
       the channel          +-----------+
                            (the conveyor belt)
```

A **goroutine** is a tiny worker. Tiny. Each one starts with about 2 kilobytes of memory, and grows only if it needs more. You can have **millions** of goroutines running at the same time on one computer. Compare that to a regular operating-system thread, which usually uses 1 to 8 megabytes of memory each. You could not have a million OS threads. Your computer would explode. But a million goroutines is fine.

A **channel** is a conveyor belt between goroutines. One worker drops a thing on the belt. Another worker picks it up. Nobody steps on each other. Nobody fights. The conveyor belt handles the synchronisation for you.

That is the soul of Go. The rest of the language exists to make this picture work.

## Why Go Is Simple

### One way to do things

In Python you can write a loop with `for`, with a list comprehension, with `map`, with `filter`, with a generator, with `itertools`, with `functools.reduce`. In Go you write a loop with `for`. That's it. There is no `while`. There is no `do-while`. There is no `foreach`. There is no list comprehension. There is just `for`. If you want a `while`, you write `for cond { }`. If you want an infinite loop, you write `for { }`. If you want to iterate, you write `for i := range thing { }`.

This is on purpose. Less ways to write the same thing means less to argue about, less to learn, less to misread.

```go
// classic three-clause for
for i := 0; i < 10; i++ { }

// while
for x < 100 { }

// infinite
for { }

// range over a slice
for i, v := range mySlice { }

// range over a map
for k, v := range myMap { }

// range over a channel
for msg := range ch { }

// range over an integer (Go 1.22+)
for i := range 10 { } // i goes 0..9
```

That is **every single loop you will ever write in Go.** All other languages have at least three loop statements. Go has one.

### gofmt — the style police

In every other language people fight about formatting. Tabs or spaces? Where do braces go? How wide is a line? How many blank lines between functions? Should structs be aligned?

In Go, you don't fight. You run `gofmt`. It reformats your code. Everyone's code looks the same. End of debate.

```bash
$ cat hello.go
package main
import "fmt"
func main(){
fmt.Println(   "hi"   )
}

$ gofmt -w hello.go
$ cat hello.go
package main

import "fmt"

func main() {
	fmt.Println("hi")
}
```

The `-w` flag means "write the changes back to the file." Without it, `gofmt` just prints the formatted version to your terminal. Run `gofmt` before every commit. Most editors run it on save automatically. You will never need to think about formatting again.

### Small spec

The Go language spec is about 90 pages of HTML. It is short. You can read it in an afternoon. The C++ spec is over 1500 pages. The Rust reference is also enormous. The Java spec is hundreds of pages. The Go spec is **small and complete.** A senior Go engineer can hold the entire language in their head. That is not a thing you can say about C++.

The result: when you read Go code by someone you have never met, you can almost always figure out what it does. There are not many secret syntax tricks. There is no operator overloading. There are no implicit conversions between types. The code does what it looks like it does.

## A Hello-World Go program

Let's actually run something. Make a folder. Put one file in it. Run it.

```bash
$ mkdir hello
$ cd hello
$ cat > main.go <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("hi")
}
EOF
$ go run main.go
hi
```

That is it. Six lines (well, the heredoc trick takes a few more). Let's break down every part.

- `package main` — every Go file starts with a `package` declaration. The package called `main` is special: it means "this is an executable program, build me an exe." Other packages are libraries. They have other names like `fmt` or `net/http` or `mylib`.
- `import "fmt"` — bring in another package. `fmt` is short for "format," and it has functions for printing and formatting strings. Almost every Go program uses it.
- `func main() { ... }` — the function called `main` in package `main` is where the program starts. It takes no arguments and returns nothing. When `main` returns, the program exits.
- `fmt.Println("hi")` — call the function `Println` from the `fmt` package, passing it the string `"hi"`. `Println` prints its argument followed by a newline.

Now let's build it instead of running it directly.

```bash
$ go build
$ ls
hello   main.go
$ ./hello
hi
```

`go build` compiled the program into an executable file called `hello` (named after the folder). You can copy this file to any other Linux machine of the same architecture and run it. It has no runtime dependency. There is no `python3` to install. There is no `java` to install. It is one file. That is one of the best things about Go.

Let's see how big it is.

```bash
$ ls -lh hello
-rwxr-xr-x  1 you  staff   1.8M Apr 27 14:23 hello
```

Almost two megabytes for "hi"? Yes. The Go runtime, the garbage collector, the goroutine scheduler, and a chunk of the standard library are all linked statically into every Go binary. That is the price of zero runtime dependencies. We can shrink it with build flags later (we'll cover `ldflags` and `-trimpath`).

## The Type System

Go is **statically typed.** That means every variable has a type, and the compiler checks that you only do legal things with each type. The compiler catches a huge number of bugs before your program ever runs. If you have used Python or JavaScript and been bitten by `undefined is not a function` at 2 a.m., this will feel like a hug.

But Go's type system is also **simple and practical.** It is not Haskell. There are no monads, no functors, no higher-kinded types. There are basic types, structs, interfaces, slices, maps, channels, and pointers. That's it.

### Basic types

```go
var i int = 42                  // signed integer (64-bit on modern machines)
var u uint64 = 42               // unsigned 64-bit
var f float64 = 3.14            // 64-bit float
var s string = "hello"          // immutable string
var b bool = true               // bool
var r rune = 'A'                // a Unicode code point (alias for int32)
var by byte = 0xff              // a byte (alias for uint8)
```

All numeric types in Go have explicit sizes: `int8`, `int16`, `int32`, `int64`, `uint8`, ..., `uint64`, `float32`, `float64`, `complex64`, `complex128`. There is also `int` and `uint` which are 32 or 64 bits depending on your platform (almost always 64 these days). Use `int` for general counting unless you have a specific reason to pick a sized type.

Go does **not** do implicit conversions between numeric types. This is on purpose:

```go
var i int32 = 5
var j int64 = i         // ERROR: cannot use i (type int32) as type int64
var j2 int64 = int64(i) // OK — explicit conversion
```

This catches a huge class of bugs in code that mixes sizes (network protocols, file formats, low-level work). If you have ever lost an afternoon to a silent overflow in C, you will appreciate it.

### Zero values

Every type in Go has a **zero value.** That is the value a variable of that type has if you don't initialise it. Zero values are the secret weapon of Go. They make a lot of code "just work" without ceremony.

```go
var i int       // 0
var f float64   // 0.0
var s string    // ""
var b bool      // false
var p *int      // nil
var sl []int    // nil
var m map[string]int // nil
var ch chan int // nil
var fn func()   // nil
```

In C, an uninitialised `int` is whatever happened to be in that piece of memory. It might be 0. It might be 4,294,967,295. It might be a value left over from somebody else's program. This causes endless bugs. In Go, an uninitialised `int` is **always** 0. End of story.

### Struct

A `struct` is a record. A bag of named fields.

```go
type Person struct {
	Name string
	Age  int
}

func main() {
	var p Person
	fmt.Println(p)        // {  0}    -- zero value: empty name, zero age
	p.Name = "Alice"
	p.Age = 30
	fmt.Println(p)        // {Alice 30}

	// or all at once:
	q := Person{Name: "Bob", Age: 25}
	fmt.Println(q)        // {Bob 25}
}
```

Note the use of `:=` here. That is the **short variable declaration.** `q := Person{...}` means "make a new variable `q`, infer its type from the right-hand side, and assign." Use `:=` inside functions. Use `var` at the top level (outside functions) where `:=` is not allowed.

Structs in Go are **value types.** When you assign one struct to another or pass a struct to a function, you get a **copy.** Not a reference. A copy. If you want a reference, take a pointer (we'll get there).

```go
p := Person{Name: "Alice", Age: 30}
q := p             // q is a *copy* of p
q.Name = "Bob"     // does NOT change p
fmt.Println(p.Name) // Alice
fmt.Println(q.Name) // Bob
```

### Interface

An `interface` is a list of method signatures. Any type that has all those methods automatically satisfies the interface. There is **no `implements` keyword.** This is called **structural typing.** It is one of Go's superpowers.

```go
type Greeter interface {
	Greet() string
}

type Cat struct{}
func (c Cat) Greet() string { return "meow" }

type Dog struct{}
func (d Dog) Greet() string { return "woof" }

func sayHi(g Greeter) {
	fmt.Println(g.Greet())
}

func main() {
	sayHi(Cat{}) // meow
	sayHi(Dog{}) // woof
}
```

Notice that neither `Cat` nor `Dog` says "I implement Greeter." They just both happen to have a `Greet() string` method. The compiler figures it out. This means you can write a function that takes an interface, and people can pass in their own types without modifying the interface or the type. Decoupling for free.

The most famous interface in Go is `io.Reader`:

```go
type Reader interface {
	Read(p []byte) (n int, err error)
}
```

Anything with a `Read` method is a `Reader`. Files. Network connections. Strings. Compressed streams. Encrypted streams. They all just work with each other because they all satisfy the same one-method interface.

### Slice

A `slice` is a window into an array. It looks like a dynamic array, but under the hood it is three things stuck together: a pointer, a length, and a capacity.

```
slice header:
+---------+--------+----------+
| pointer | length | capacity |
+----|----+--------+----------+
     |
     v
     [a][b][c][d][e][f][g][h]    underlying array (capacity = 8)
     ^-- length = 4
```

```go
sl := []int{10, 20, 30}
fmt.Println(sl)        // [10 20 30]
fmt.Println(len(sl))   // 3
fmt.Println(cap(sl))   // 3

sl = append(sl, 40)    // append a new value
fmt.Println(sl)        // [10 20 30 40]
fmt.Println(len(sl))   // 4
fmt.Println(cap(sl))   // 6 (Go doubled the capacity)
```

When you `append` and there is no spare capacity, Go allocates a new bigger underlying array (usually double the size), copies the old data over, and returns a slice pointing to the new array. This is why `append` returns a new slice — you must always re-assign:

```go
sl = append(sl, 50) // CORRECT
append(sl, 50)      // WRONG — the result is thrown away
```

We will come back to slice gotchas in the slice/map section. They are a top source of bugs for new Go programmers.

### Map

A `map` is a hash table. Lookup, insert, and delete are all (on average) constant time.

```go
m := map[string]int{
	"alice": 30,
	"bob":   25,
}
fmt.Println(m["alice"])     // 30
fmt.Println(m["nobody"])    // 0  -- the zero value of int!

// is the key actually there?
v, ok := m["nobody"]
fmt.Println(v, ok)          // 0 false

m["carol"] = 40             // insert
delete(m, "alice")          // remove

for k, v := range m {       // iterate
	fmt.Println(k, v)       // order is RANDOM
}
```

Two things to remember:

1. Reading a missing key returns the **zero value** of the value type. Use the `v, ok := m[k]` form to tell missing from "actually zero."
2. Map iteration order is **deliberately randomised** by the Go runtime. Do not depend on a particular order. If you need order, sort the keys first.

You **must** initialise a map before you write to it. The zero value of a map is `nil`, and writing to a nil map panics:

```go
var m map[string]int
m["alice"] = 30  // panic: assignment to entry in nil map

m = make(map[string]int)
m["alice"] = 30  // OK
```

### Channel

A channel is the conveyor belt we drew above. We will spend a whole section on channels later. For now:

```go
ch := make(chan int)   // unbuffered channel of ints
go func() {
	ch <- 42           // send 42 into ch
}()
v := <-ch              // receive a value from ch
fmt.Println(v)         // 42
```

### Pointer

A pointer is an address — a number that says where in memory another value lives. In Go, `*T` means "pointer to T," `&x` means "address of x," and `*p` means "the value at address p."

```go
x := 10
p := &x        // p is a *int pointing at x
fmt.Println(*p) // 10
*p = 20        // change the value at p
fmt.Println(x)  // 20  -- x changed!
```

Pointers are how you pass things by reference in Go, and how you let a function modify its caller's variables. Unlike C, Go pointers cannot do arithmetic. There is no `p++`. You can only point and dereference. This kills a huge category of bugs.

### nil

`nil` is the zero value of pointers, slices, maps, channels, functions, and interfaces. It means "this thing points to nothing." Reading or writing through a `nil` pointer is a runtime panic. Sending or receiving on a `nil` channel blocks forever. Calling a method on a `nil` interface is a panic. Calling a method on a non-nil interface that wraps a nil pointer **might** be fine — see the famous nil-interface-vs-nil-concrete confusion in the Common Confusions section.

## Functions and Methods

### Functions

A function in Go takes some arguments and returns zero or more values.

```go
func add(a, b int) int {
	return a + b
}

func divmod(a, b int) (int, int) {
	return a / b, a % b
}

q, r := divmod(17, 5)
fmt.Println(q, r) // 3 2
```

Note that **Go can return multiple values.** This is used everywhere — almost every operation that can fail returns `(result, error)`:

```go
f, err := os.Open("/etc/hosts")
if err != nil {
	return err
}
defer f.Close()
// ... use f
```

You can also name your return values, which is occasionally useful:

```go
func divmod(a, b int) (q, r int) {
	q = a / b
	r = a % b
	return // "naked return" — uses the named values
}
```

Most style guides say avoid naked returns except for very short functions, because they are easy to misread. Just use the explicit `return q, r`.

### Variadic functions

A function whose last parameter is `...T` accepts any number of `T` values:

```go
func sum(xs ...int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}

sum(1, 2, 3)          // 6
sum()                 // 0
xs := []int{1,2,3}
sum(xs...)            // 6 — spread a slice as variadic args
```

`fmt.Printf` is the most famous variadic function:

```go
fmt.Printf("hello %s, you are %d\n", "alice", 30)
```

### Methods

A **method** is a function with a special parameter at the front called the **receiver.** The method belongs to the type of the receiver.

```go
type Counter struct {
	n int
}

func (c *Counter) Inc() {
	c.n++
}

func (c Counter) Value() int {
	return c.n
}

func main() {
	var c Counter
	c.Inc()
	c.Inc()
	c.Inc()
	fmt.Println(c.Value()) // 3
}
```

### Value receiver vs pointer receiver

Look closely at the two methods above.

`Inc` has a **pointer receiver** `(c *Counter)`. That means `Inc` gets a *pointer* to the original Counter. When `Inc` does `c.n++`, it changes the original.

`Value` has a **value receiver** `(c Counter)`. That means `Value` gets a *copy* of the Counter. It cannot change the original. It can only read it.

The rule of thumb:

- If the method **modifies** the receiver, use a pointer receiver.
- If the receiver is **large** (a struct with many big fields), use a pointer receiver to avoid copying.
- If the type contains a `sync.Mutex` or any other thing you must not copy, use a pointer receiver.
- Otherwise, value receiver is fine.
- **Be consistent** within one type — don't mix value and pointer receivers on the same type unless you have a reason.

## Interfaces

We met interfaces above. Now let's go deeper, because they are the most powerful and most subtle thing in Go.

### Structural typing

In Java, to implement an interface you must say so:

```java
class Cat implements Greeter { ... }
```

In Go, you do not. If your type has all the methods, you implement the interface, full stop. The compiler figures it out at compile time.

This sounds like a small change but it has huge consequences. It means you can write a function that takes some interface, and **other people can write types that satisfy your interface without ever importing your package's interface declaration.** They just write the methods. The compiler does the matching. This means Go libraries can decouple from each other in ways that are awkward in nominally-typed languages.

### The empty interface — `interface{}` and `any`

`interface{}` is the empty interface. It has no methods. **Every** Go type satisfies it. It is the closest thing Go has to "anything."

Since Go 1.18 there is an alias `any` that means exactly the same thing. Use `any`. It reads better.

```go
var x any = 42
var y any = "hello"
var z any = []int{1, 2, 3}
```

You can put anything into an `any`, but to do something with it you must do a **type assertion** or a **type switch:**

```go
func describe(x any) {
	switch v := x.(type) {
	case int:
		fmt.Println("int:", v)
	case string:
		fmt.Println("string:", v)
	default:
		fmt.Println("something else:", v)
	}
}

describe(42)        // int: 42
describe("hello")   // string: hello
describe(3.14)      // something else: 3.14
```

A type assertion: `v, ok := x.(int)`. If `x` is an `int`, `ok` is true and `v` is the int. If not, `ok` is false and `v` is the zero value. Without `, ok` it panics on mismatch, so use the comma-ok form unless you are 100% sure.

### Small interfaces are best

The Go standard library has lots of one-method interfaces: `io.Reader`, `io.Writer`, `io.Closer`, `fmt.Stringer`, `error`. The wisdom is "the bigger the interface, the weaker the abstraction." A one-method interface is implemented everywhere; a 30-method interface is hard to mock and hard to swap.

```go
// THE most useful pattern in Go
type Stringer interface {
	String() string
}
```

Implement `String()` on your type, and `fmt.Println` automatically uses it.

```go
type Person struct{ Name string; Age int }
func (p Person) String() string {
	return fmt.Sprintf("%s (%d)", p.Name, p.Age)
}

fmt.Println(Person{"Alice", 30}) // Alice (30)
```

## Error Handling

Go does **not** have exceptions. There is no `try` / `catch`. Functions that can fail return an `error` as the last return value. You check it. If it is `nil`, all is well. If it is not, something went wrong.

```go
f, err := os.Open("/etc/hosts")
if err != nil {
	return fmt.Errorf("open hosts: %w", err)
}
defer f.Close()
```

The `error` is just an interface:

```go
type error interface {
	Error() string
}
```

Any type with an `Error() string` method is an error.

### errors.New and fmt.Errorf

To make an error you usually call `errors.New` or `fmt.Errorf`:

```go
var ErrNotFound = errors.New("not found")

func find(k string) (Thing, error) {
	if !exists(k) {
		return Thing{}, ErrNotFound
	}
	// ...
}
```

`fmt.Errorf` lets you format the error message and, importantly, **wrap** another error with `%w`:

```go
data, err := os.ReadFile(path)
if err != nil {
	return fmt.Errorf("read config %s: %w", path, err)
}
```

The `%w` verb is special — it preserves the original error so callers can inspect it with `errors.Is` and `errors.As`. (Don't confuse `%w` with `%v` and `%s`, which just embed the message and lose the error chain.)

### errors.Is and errors.As

`errors.Is(err, target)` walks the wrap chain looking for `target`:

```go
_, err := os.Open("/no/such/file")
if errors.Is(err, os.ErrNotExist) {
	fmt.Println("file is missing") // matches even through wrappers
}
```

`errors.As(err, &target)` walks the chain looking for an error of a particular **type** and assigns it to `target`:

```go
var pathErr *os.PathError
if errors.As(err, &pathErr) {
	fmt.Println("path was:", pathErr.Path)
}
```

### errors.Join (Go 1.20+)

`errors.Join` combines multiple errors into one:

```go
err := errors.Join(err1, err2, err3)
```

The combined error matches all the originals through `errors.Is` / `errors.As`. Use it when a function does several things and you want to report all the failures, not just the first.

### Sentinel errors and error types

A **sentinel error** is a known error value at package level you compare against:

```go
var ErrClosed = errors.New("foo: connection closed")

if errors.Is(err, ErrClosed) {
	// reconnect
}
```

A **typed error** is a struct with extra context:

```go
type ValidationError struct {
	Field string
	Reason string
}
func (v *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Reason)
}
```

Use sentinels when callers need to compare. Use typed errors when callers need extra info.

### Wrapping rules

- Use `%w` once per `fmt.Errorf` call (you can wrap multiple via `errors.Join` since 1.20).
- Add **context**: where, what, with which input.
- Don't wrap and re-format the same message twice — that's just noise.
- At the **boundary** of your program (e.g., `main`), you log or report; everywhere inside, you wrap and bubble up.

## Slices and Maps

Slices and maps are where most Go beginners stub their toes. Three things to know.

### The slice header sneak-attack

A slice is a header (pointer, length, capacity) into an underlying array. **Slicing does not copy.** Two slices can share the same backing array.

```go
a := []int{1, 2, 3, 4, 5}
b := a[1:3]            // b shares storage with a
b[0] = 999
fmt.Println(a)         // [1 999 3 4 5]   <-- a was changed!
fmt.Println(b)         // [999 3]
```

This is fast, but surprising. If you want a real copy, do:

```go
c := append([]int(nil), a[1:3]...)
// or use the built-in copy:
c := make([]int, 2)
copy(c, a[1:3])
```

### append and capacity doubling

`append` is your only way to grow a slice. If there is spare capacity, `append` writes in place and returns a slice with a longer length. If not, `append` allocates a new (usually 2x) backing array, copies the old data, and returns a slice pointing to the new array.

```go
a := make([]int, 0, 4)   // len=0, cap=4
fmt.Println(len(a), cap(a)) // 0 4
a = append(a, 1, 2, 3, 4)
fmt.Println(len(a), cap(a)) // 4 4
a = append(a, 5)            // need more room — new backing array
fmt.Println(len(a), cap(a)) // 5 8
```

The doubling means amortised O(1) appends, but it also means **the storage can move out from under you** if other slices were sharing it:

```go
a := []int{1, 2, 3, 4}
b := a[:2]               // b shares storage with a
a = append(a, 5)         // may or may not move depending on cap
b = append(b, 99)        // does this clobber a[2]? depends on whether `a` moved
```

Rule: if you are passing slices around between goroutines or returning them from functions, be very careful about who owns the backing array. When in doubt, copy.

### Map iteration order is randomised

```go
m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
for k, v := range m {
	fmt.Println(k, v)
}
```

This prints in a different order every run. The Go authors did this on purpose. It stops people from accidentally relying on insertion order or hash order. If you need a specific order, sort the keys:

```go
keys := make([]string, 0, len(m))
for k := range m {
	keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
	fmt.Println(k, m[k])
}
```

### `clear` (Go 1.21+)

The built-in `clear` empties a slice or a map:

```go
clear(s)  // sets all elements of slice s to zero (length stays the same)
clear(m)  // removes all keys from map m
```

Useful for reusing buffers in a hot loop.

## Goroutines and Channels

Now we get to the heart of Go.

### Goroutines

A **goroutine** is a function running on its own. You start one with the `go` keyword.

```go
go someFunc(a, b, c)
```

That's it. The goroutine starts in the background. The current function continues immediately. If you want anonymous functions:

```go
go func() {
	fmt.Println("hi from a goroutine")
}()
```

Goroutines are **multiplexed** onto a small number of OS threads by the Go runtime. The runtime uses a model called **GMP:** goroutines (G), OS threads (M), and processors (P). The default is one P per CPU core.

```
+--------+   +--------+   +--------+
|   P    |   |   P    |   |   P    |   <-- one per CPU core (GOMAXPROCS)
+----|---+   +----|---+   +----|---+
     |            |            |
     v            v            v
+--------+   +--------+   +--------+
|   M    |   |   M    |   |   M    |   <-- OS threads, one bound to each P
+--------+   +--------+   +--------+
     |            |            |
   runs        runs         runs
     |            |            |
     v            v            v
+--------+   +--------+   +--------+
| G G G  |   | G G G  |   | G G G  |   <-- thousands of goroutines
+--------+   +--------+   +--------+
```

When a goroutine makes a blocking call (network read, channel receive, sleep), the scheduler parks it and puts another goroutine on that M instead. This is why one Go service can handle hundreds of thousands of TCP connections without breaking a sweat.

### The famous deadlock

```go
func main() {
	ch := make(chan int)
	ch <- 42        // blocks forever — nobody is receiving
}
```

This program runs and panics:

```
fatal error: all goroutines are asleep - deadlock!
```

Go is smart enough to notice that every goroutine is blocked and yells at you. The fix is to either receive from another goroutine or use a buffered channel:

```go
ch := make(chan int, 1)  // buffer of 1
ch <- 42                 // doesn't block (buffer has room)
```

### Send and receive

```go
ch <- v          // send v on ch (blocks until someone receives, on unbuffered)
v := <-ch        // receive a value from ch (blocks until someone sends)
v, ok := <-ch    // receive; ok is false if ch was closed and drained
```

### Buffered vs unbuffered

```
unbuffered channel:
  sender blocks until receiver picks up.
  send and receive happen simultaneously — a "rendezvous."
  use when you want to synchronise two goroutines.

buffered channel of capacity N:
  sender does not block as long as buffer has room.
  receiver does not block as long as buffer has items.
  use when you want a small queue between producer and consumer.
```

### close

A sender may close a channel to say "no more values are coming." Receivers see this:

```go
close(ch)
v, ok := <-ch    // ok = false once ch is closed and drained
```

Iterating with `for v := range ch` ends when the channel is closed.

**Two rules of close:**
1. Only the **sender** closes a channel. Never the receiver. (How would the receiver know if more sends are coming?)
2. Closing an already-closed channel **panics.**
3. Sending on a closed channel **panics.**

### select

`select` is like a `switch` for channel operations. It blocks until one of its cases can proceed.

```go
select {
case v := <-ch1:
	fmt.Println("from ch1:", v)
case v := <-ch2:
	fmt.Println("from ch2:", v)
case ch3 <- 42:
	fmt.Println("sent to ch3")
case <-time.After(1 * time.Second):
	fmt.Println("timeout")
default:
	fmt.Println("nothing ready")
}
```

The `default` branch runs immediately if no other case is ready (turning the select non-blocking). The `time.After` trick gives you a timeout. `select` is the swiss-army knife of channels.

### Channel send/recv blocking diagram

```
unbuffered:
  goroutine A:    ch <- v  -->  [BLOCKS until B receives]  --> proceeds
  goroutine B:    v := <-ch -->  [BLOCKS until A sends]    --> v is delivered

buffered (cap 2):
  goroutine A:    ch <- v1  -->  [v1 in buf, A continues]
  goroutine A:    ch <- v2  -->  [v2 in buf, A continues]
  goroutine A:    ch <- v3  -->  [BLOCKS until B receives one]
  goroutine B:    <-ch     -->  [v1 popped, A unblocks if blocked]
```

## Sync Primitives

Channels are the preferred way to synchronise in Go, but the `sync` package has the classic primitives for when channels would be overkill or wrong.

### sync.Mutex

A mutex protects a critical section. Only one goroutine can hold the lock at a time.

```go
var (
	mu    sync.Mutex
	count int
)

func inc() {
	mu.Lock()
	defer mu.Unlock()
	count++
}
```

Always pair `Lock` and `Unlock`. The `defer` form is bulletproof — it unlocks even if the function panics.

### sync.RWMutex

A reader-writer mutex. Many readers OR one writer.

```go
var (
	mu sync.RWMutex
	m  = map[string]int{}
)

func get(k string) int {
	mu.RLock()
	defer mu.RUnlock()
	return m[k]
}

func set(k string, v int) {
	mu.Lock()
	defer mu.Unlock()
	m[k] = v
}
```

Use `RWMutex` only when reads vastly outnumber writes. If contention is light, a plain `Mutex` is faster because RWMutex has more overhead.

### sync.WaitGroup

Wait for a bunch of goroutines to finish.

```go
var wg sync.WaitGroup
for _, url := range urls {
	wg.Add(1)
	go func(u string) {
		defer wg.Done()
		fetch(u)
	}(u)
}
wg.Wait()
```

`Add(n)` adds n to the counter. `Done()` subtracts one. `Wait()` blocks until the counter is zero. **Always call `Add` before starting the goroutine,** never inside the goroutine — otherwise `Wait` might race past zero.

### sync.Once

Run a function exactly once, ever, even with many callers.

```go
var (
	once   sync.Once
	config *Config
)

func GetConfig() *Config {
	once.Do(func() {
		config = loadConfigFromDisk()
	})
	return config
}
```

In Go 1.21 there are `sync.OnceFunc`, `sync.OnceValue`, and `sync.OnceValues` that wrap this pattern more conveniently.

### atomic

For simple counters and flags, the `sync/atomic` package is faster than a mutex.

```go
var counter atomic.Int64
counter.Add(1)
n := counter.Load()
```

The `atomic.Int32`, `atomic.Int64`, etc. types arrived in Go 1.19. Before that you used the lower-level `atomic.AddInt64(&x, 1)` etc. (still works, but the typed variants are nicer).

`atomic.Value` and `atomic.Pointer[T]` (Go 1.19+) let you atomically swap a whole struct, useful for lock-free configuration reloads.

## Context Package

Imagine you start a request to fetch a webpage. The webpage takes too long. You want to cancel. You also want to cancel **everything that webpage's handler started** — database queries, downstream HTTP calls, goroutines.

That is what `context.Context` is for. A context propagates **cancellation, deadlines, and a small amount of request-scoped data** down a tree of function calls.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := http.DefaultClient.Do(req)
```

If the timeout fires, the HTTP call aborts. If you call `cancel()`, every operation watching `ctx.Done()` aborts. The cancellation propagates down a tree:

```
ctx (root)
 |
 +-- ctx1 (with timeout 5s)
 |    |
 |    +-- ctx1a (db query)
 |    +-- ctx1b (downstream HTTP)
 |
 +-- ctx2 (with cancel)
      |
      +-- ctx2a (background worker)

cancel ctx -> cancels ctx1, ctx1a, ctx1b, ctx2, ctx2a
cancel ctx1 -> cancels ctx1a, ctx1b only
```

### context.Background and context.TODO

- `context.Background()` is the **root** context. Use it in `main`, in tests, when you really do mean "no parent."
- `context.TODO()` means "I don't know yet what to use here, fix me later." Useful as a placeholder during refactors. Functionally identical to `Background()`.

### WithCancel, WithTimeout, WithDeadline

```go
ctx, cancel := context.WithCancel(parent)        // cancel manually
ctx, cancel := context.WithTimeout(parent, dur)  // cancels after dur
ctx, cancel := context.WithDeadline(parent, t)   // cancels at time t
```

Always `defer cancel()` — even with timeouts. It frees resources and is a safety net.

### WithValue

`context.WithValue(parent, key, value)` adds a value to the context. Use it for **request-scoped** data: request ID, user ID, trace span. Do **not** use it as a back-door to pass function arguments. Don't put your database handle in here.

```go
type ctxKey int
const reqIDKey ctxKey = 0

ctx := context.WithValue(parent, reqIDKey, "req-123")
id, _ := ctx.Value(reqIDKey).(string)
```

The conventional key type is your own unexported type (here `ctxKey`) so it cannot collide with another package's keys.

### Listening for cancellation

In your worker:

```go
for {
	select {
	case <-ctx.Done():
		return ctx.Err()    // context.Canceled or context.DeadlineExceeded
	case msg := <-incoming:
		handle(msg)
	}
}
```

### Go 1.21 additions

- `context.WithCancelCause(parent)` returns a cancel function that records *why* you cancelled, retrievable via `context.Cause(ctx)`.
- `context.AfterFunc(ctx, f)` runs `f` on its own goroutine when ctx is cancelled.

## The Standard Library Tour

Go's standard library is *enormous.* You can build production services with **only** the stdlib — no third-party packages — and that is normal. Here is the tour.

### net/http — full HTTP client and server

```go
http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hi")
})
log.Fatal(http.ListenAndServe(":8080", nil))
```

That is a real working HTTP server. No frameworks. As of Go 1.22 the default `ServeMux` supports method-and-pattern matching:

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /users/{id}", getUser)
mux.HandleFunc("POST /users", createUser)
```

Client side:

```go
resp, err := http.Get("https://example.com/")
if err != nil { return err }
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
```

### encoding/json

```go
type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age,omitempty"`
}

p := Person{Name: "Alice", Age: 30}
data, _ := json.Marshal(p)
fmt.Println(string(data)) // {"name":"Alice","age":30}

var q Person
json.Unmarshal(data, &q)
```

Tag options: `omitempty` skips zero values; `-` skips the field; `string` forces serialise-as-string. As of Go 1.24, `omitzero` is a stricter sibling of `omitempty` (covers more zero-value cases).

For streaming, use `json.Decoder` and `json.Encoder` — they read/write line by line and don't buffer everything in memory.

### encoding/xml, encoding/csv, encoding/gob, encoding/binary

The same shape as `encoding/json`. `xml.Marshal`, `csv.NewWriter`, etc. `gob` is Go's native binary format — fast for Go-to-Go RPC. `binary` reads and writes fixed-width integers.

### io and io/fs

`io.Reader` and `io.Writer` are the two most important interfaces in Go. Lots of helper functions:

```go
io.Copy(dst, src)         // copy until EOF
io.MultiWriter(a, b)      // tee writes to multiple writers
io.TeeReader(r, w)        // every read also writes to w
io.Pipe()                 // in-memory pipe (reader, writer)
io.ReadAll(r)             // slurp everything (careful with size)
```

`io/fs` (Go 1.16+) is an interface for read-only file systems. `embed.FS` satisfies it.

### os

```go
os.Args                   // []string of command-line args
os.Getenv("HOME")
os.Setenv("FOO", "bar")
os.Open("file")           // returns *os.File (an io.Reader)
os.Create("file")
os.ReadFile("file")       // slurp a small file
os.WriteFile("f", b, 0644)
os.Exit(1)
os.Stdin, os.Stdout, os.Stderr
```

### fmt

```go
fmt.Println(args...)
fmt.Printf("%s %d\n", s, i)
fmt.Sprintf("...")        // returns the string instead of printing
fmt.Errorf("ctx: %w", err) // returns an error with wrap

// useful verbs:
%v      default
%+v     struct with field names
%#v     Go-syntax representation
%T      the type
%s %d %f %t %q %x %o %b %p
```

### strings, strconv

```go
strings.Contains("hello", "ell")    // true
strings.Split("a,b,c", ",")         // ["a" "b" "c"]
strings.ToLower(s)
strings.Trim(s, " ")
strings.NewReplacer("a","1","b","2").Replace("ab")
strings.Builder{}                   // efficient string concatenation

strconv.Atoi("42")                  // -> 42, nil
strconv.Itoa(42)                    // -> "42"
strconv.ParseFloat("3.14", 64)
strconv.FormatFloat(3.14, 'f', 2, 64)
```

### time

```go
time.Now()
time.Now().UnixNano()
time.Sleep(time.Second)
time.After(2 * time.Second)         // returns a chan that fires after 2s
time.Tick(time.Second)              // periodic chan (leaks if you stop using it)
time.NewTicker(d).C                 // safer; remember to .Stop()

t.Format("2006-01-02 15:04:05")     // the famous reference time
time.Parse("2006-01-02", "2024-04-27")
```

The format string in Go uses the **reference date** `Mon Jan 2 15:04:05 MST 2006`. It is weird. It is also memorable: 1, 2, 3, 4, 5, 6, 7 in order.

### regexp

```go
re := regexp.MustCompile(`^([a-z]+)\s+(\d+)$`)
m := re.FindStringSubmatch("alice 30")
// m == ["alice 30", "alice", "30"]
```

Go uses RE2 syntax — no backreferences, but linear-time and safe against catastrophic backtracking.

### sort

```go
sort.Ints([]int{3,1,2})
sort.Strings([]string{"b","a"})
sort.Slice(xs, func(i, j int) bool { return xs[i].Age < xs[j].Age })
```

Go 1.21 added `slices.Sort` which is type-safe via generics — prefer it.

### sync, sync/atomic

Covered above.

### testing

Built right into the language (next section).

### log/slog (Go 1.21+)

`log/slog` is the official structured logger.

```go
slog.Info("user logged in", "user", "alice", "ip", "1.2.3.4")
// {"time":"...","level":"INFO","msg":"user logged in","user":"alice","ip":"1.2.3.4"}

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
slog.SetDefault(logger)
```

Key/value pairs all the way down. Replaces ad hoc third-party loggers like zerolog and zap for most needs.

## Modules

A **module** is a unit of versioned source. Each module has a `go.mod` file declaring its name, Go version, and dependencies.

```bash
$ mkdir myapp
$ cd myapp
$ go mod init github.com/me/myapp
$ cat go.mod
module github.com/me/myapp

go 1.24
```

Add a dependency by importing it:

```go
import "github.com/google/uuid"
```

Then:

```bash
$ go mod tidy
go: finding module for package github.com/google/uuid
go: downloading github.com/google/uuid v1.6.0
go: added github.com/google/uuid v1.6.0
```

`go.sum` records the cryptographic hashes of every dependency. Commit it. The `GOPROXY` setting (default `https://proxy.golang.org`) caches dependencies. The `GOSUMDB` (default `sum.golang.org`) verifies them.

### replace directive

For local dev or vendoring a fork:

```
replace github.com/foo/bar => ../bar-local
replace github.com/foo/bar => github.com/me/bar v0.0.0-20240101120000-deadbeefcafe
```

### Workspaces (`go.work`)

If you have multiple modules you are editing together:

```bash
$ go work init
$ go work use ./svc-a ./svc-b ./common
```

This produces a `go.work` file. Inside the workspace, `svc-a` will see local edits in `common` even though `common` is its own module.

### Semver

Go modules use semantic versioning: `vMAJOR.MINOR.PATCH`. Breaking changes go in a major version bump. **For v2 and up, the major version goes in the import path:**

```go
import "github.com/me/foo/v2"
```

This means `foo v1` and `foo v2` can coexist in the same build. Annoying but necessary.

## Build

```bash
go build                      # build current package; output binary in cwd
go build -o myapp             # name the output
go build ./...                # build everything in this module
go install ./cmd/myapp        # install to $GOPATH/bin (default ~/go/bin)
```

### ldflags

Inject values at build time via the linker:

```bash
go build -ldflags="-X main.version=v1.2.3 -s -w" -o myapp
```

`-X package.name=value` sets a string variable. `-s` strips the symbol table. `-w` strips DWARF debugging info. Together they shrink the binary by a lot and make it harder to reverse-engineer.

### Cross-compile

Set `GOOS` and `GOARCH`:

```bash
GOOS=linux   GOARCH=amd64 go build -o myapp-linux-amd64
GOOS=linux   GOARCH=arm64 go build -o myapp-linux-arm64
GOOS=darwin  GOARCH=arm64 go build -o myapp-mac-arm64
GOOS=windows GOARCH=amd64 go build -o myapp.exe
```

No cross-compiler toolchain required. This is one of Go's killer features.

### CGO_ENABLED

If you want a fully static binary with no `libc` link, set `CGO_ENABLED=0`:

```bash
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o myapp
```

The result is a single static binary you can drop into a `FROM scratch` Docker container.

`-trimpath` removes the build machine's filesystem paths from the binary. Useful for reproducibility and not leaking your `/Users/yourname/...` into the world.

### //go:embed

You can embed files into your binary at compile time:

```go
import _ "embed"

//go:embed assets/*
var assets embed.FS

//go:embed VERSION
var version string

//go:embed banner.txt
var banner []byte
```

The directive must be on the line immediately above the `var`. The path is **relative** to the source file. The variable type can be `string`, `[]byte`, or `embed.FS`. With `embed.FS` you get a read-only file system you can pass to `http.FileServer`.

## Testing

Go has built-in testing. No JUnit. No pytest. Just `go test`.

A test file is named `*_test.go` and lives **in the same package** as the code under test.

```go
// math.go
package mathx

func Add(a, b int) int { return a + b }

// math_test.go
package mathx

import "testing"

func TestAdd(t *testing.T) {
	got := Add(2, 3)
	if got != 5 {
		t.Errorf("Add(2,3) = %d, want 5", got)
	}
}
```

Run:

```bash
$ go test
PASS
ok      myapp/mathx     0.123s

$ go test -v
=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
PASS
ok      myapp/mathx     0.123s
```

### Table-driven tests

The signature Go test pattern:

```go
func TestAdd(t *testing.T) {
	cases := []struct {
		name    string
		a, b    int
		want    int
	}{
		{"both zero", 0, 0, 0},
		{"positive", 2, 3, 5},
		{"negative", -1, -2, -3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Add(tc.a, tc.b); got != tc.want {
				t.Errorf("Add(%d,%d) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
```

`t.Run` produces a named subtest. `go test -run TestAdd/positive` runs just that subtest. `t.Parallel()` inside the subtest runs cases in parallel.

### Helpers, cleanup, env

```go
t.Helper()                    // mark this fn as a helper so failures point at the caller
t.Cleanup(func() { /* ... */ })  // run at end of test/subtest, even on failure
t.Setenv("FOO", "bar")        // set env var for this test (Go 1.17+); auto-restored
t.TempDir()                   // creates a fresh temp dir; cleaned up automatically
```

### Benchmarks

```go
func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Add(2, 3)
	}
}
```

Run with `go test -bench=. -benchmem`.

`b.N` is set by the framework: it runs the loop more and more times until the run takes long enough to time accurately. Use `b.ResetTimer()` if you need to do setup that should not count.

Go 1.24 added `b.Loop()` which is a cleaner alternative to `for i := 0; i < b.N; i++` — it handles timer management automatically.

### Fuzz testing (Go 1.18+)

```go
func FuzzAdd(f *testing.F) {
	f.Add(2, 3)                // seed
	f.Fuzz(func(t *testing.T, a, b int) {
		got := Add(a, b)
		if got != a+b {
			t.Errorf("mismatch")
		}
	})
}
```

Run with `go test -fuzz=FuzzAdd`. The framework throws random inputs at your code looking for crashes or assertion failures.

### testify caveat

Lots of Go projects use `github.com/stretchr/testify` for assertions. It is fine. But for new code, **most projects do not need it.** The standard pattern is just `if got != want { t.Errorf(...) }`. Save the dependency. If you really love `assert.Equal`, fine, but know that you can ship without it.

## Profiling

Go has world-class profiling tools built in.

### -bench

```bash
go test -bench=. -benchmem -count=10 ./...
```

`-count=10` runs each benchmark 10 times so you can see variance. Pipe the output to `benchstat` to compare two runs:

```bash
go test -bench=. -count=10 ./... > old.txt
# make a change
go test -bench=. -count=10 ./... > new.txt
benchstat old.txt new.txt
```

### pprof

A live web server with `import _ "net/http/pprof"` exposes CPU and heap profiles at `/debug/pprof/`:

```go
import (
	_ "net/http/pprof"
	"net/http"
)

go func() { http.ListenAndServe("localhost:6060", nil) }()
```

Then:

```bash
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30
```

This grabs a 30-second CPU profile and opens an interactive web UI showing flame graphs, top functions, source code annotations.

For heap:

```bash
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

### trace

For deeper analysis (goroutine scheduling, syscall blocking, GC events):

```go
import "runtime/trace"

f, _ := os.Create("trace.out")
trace.Start(f)
defer trace.Stop()
// ... do work
```

Then:

```bash
go tool trace trace.out
```

Opens an interactive web view in your browser.

### Race detector

```bash
go test -race ./...
go run -race main.go
go build -race -o myapp
```

The `-race` flag adds runtime instrumentation that catches **data races** — places where two goroutines touch the same memory without synchronisation. It costs about 2x in CPU and 5–10x in memory, so you don't ship with it on, but you absolutely run your tests with `-race` in CI.

## Common Patterns

### Functional options

How to make a constructor that takes optional parameters in a clean way:

```go
type Server struct {
	addr    string
	timeout time.Duration
	tls     bool
}

type Option func(*Server)

func WithTimeout(d time.Duration) Option {
	return func(s *Server) { s.timeout = d }
}

func WithTLS(s *Server) { s.tls = true }

func NewServer(addr string, opts ...Option) *Server {
	s := &Server{addr: addr, timeout: 30 * time.Second}
	for _, o := range opts {
		o(s)
	}
	return s
}

NewServer(":8080", WithTimeout(5*time.Second), WithTLS)
```

Reads naturally, takes any number of options, doesn't break compatibility when you add new options later.

### The empty struct as a no-op

`struct{}` is a struct with no fields. It takes **zero bytes.** Use it as a value when you only care about presence:

```go
set := map[string]struct{}{}
set["alice"] = struct{}{}
set["bob"]   = struct{}{}
if _, ok := set["alice"]; ok { /* present */ }
```

A `chan struct{}` is a signal-only channel:

```go
done := make(chan struct{})
go func() { defer close(done); /* work */ }()
<-done
```

### iota — the enum

`iota` is the line counter inside a `const` block.

```go
type Day int
const (
	Sunday Day = iota   // 0
	Monday              // 1
	Tuesday             // 2
	Wednesday           // 3
	Thursday            // 4
	Friday              // 5
	Saturday            // 6
)
```

You can do bit flags too:

```go
type Perm uint
const (
	Read    Perm = 1 << iota   // 1
	Write                       // 2
	Execute                     // 4
)
```

### Type assertions

```go
var x any = "hello"
s, ok := x.(string)        // ok=true, s="hello"
n, ok := x.(int)           // ok=false, n=0 (no panic)
n := x.(int)               // PANIC: interface conversion
```

Always use the comma-ok form unless a panic is correct.

### any vs interface{}

Same thing since Go 1.18. Use `any` in new code; `interface{}` in old code is fine to leave alone.

## Generics

Go got generics in 1.18. They are deliberately minimal — type parameters and constraints only, no variance, no associated types, no overloading.

```go
func Map[T, U any](xs []T, f func(T) U) []U {
	out := make([]U, len(xs))
	for i, x := range xs {
		out[i] = f(x)
	}
	return out
}

doubled := Map([]int{1,2,3}, func(x int) int { return x * 2 })
// [2 4 6]
```

`[T, U any]` declares two type parameters, both constrained by `any` (which means "any type at all").

### Constraints

You can constrain a type parameter to be **comparable** (supports `==`):

```go
func Index[T comparable](xs []T, target T) int {
	for i, x := range xs {
		if x == target { return i }
	}
	return -1
}
```

You can constrain to a **type set:**

```go
type Number interface {
	int | int32 | int64 | float32 | float64
}

func Sum[T Number](xs []T) T {
	var sum T
	for _, x := range xs { sum += x }
	return sum
}
```

The `~` prefix means "any type whose underlying type is this":

```go
type Ordered interface {
	~int | ~int64 | ~float64 | ~string
}
```

Now `type MyInt int` satisfies `Ordered` because its underlying type is `int`.

The standard `golang.org/x/exp/constraints` package has handy ones (`Ordered`, `Integer`, etc.) and Go 1.21 added `cmp.Ordered` and `slices.*` / `maps.*` packages built on generics.

## Common Errors

The exact error messages Go prints, and what they mean:

```
cannot find package "foo" in any of: ...
```
Your import path is wrong, or you forgot `go mod tidy`, or the module is private and `GOPROXY` can't reach it.

```
undefined: X
```
You used a name the compiler doesn't know. Possibilities: typo, missing import, capitalisation (Go is case-sensitive — `os.open` is wrong, `os.Open` is right), or the symbol is unexported (lowercase) and you imported the package.

```
cannot use X (type T1) as type T2 in argument to ...
```
You passed the wrong type. Go does no implicit conversions. Convert explicitly: `T2(x)`.

```
non-name on left side of :=
```
You used `:=` with something that isn't a fresh variable name on the left. `:=` introduces new variables.

```
declared but not used
```
You declared a variable and never read it. Go will not let you. Either use it, rename to `_`, or delete the declaration.

```
imported and not used: "foo"
```
Same idea for imports. Delete the import or use it. Many editors do this for you on save.

```
argument is not a function
```
You wrote `x()` where `x` is a value, not a callable. Common when you forgot to declare `var f func()` and accidentally have `var f int`.

```
no Go files in /path
```
The directory you pointed `go build` at has no `.go` files (or the only files have a build constraint excluding them on this OS/arch).

```
cyclic import
```
Package A imports B and B imports A directly or transitively. Refactor: move shared code into a third package C that A and B both import.

```
missing go.sum entry for module providing package ...; to add: go mod download
```
Run `go mod tidy` (preferred) or `go mod download` to update `go.sum`.

```
panic: assignment to entry in nil map
```
You wrote to a map that is `nil`. Initialise it first: `m = make(map[K]V)`.

```
panic: runtime error: index out of range [N] with length M
```
Slice index past `len(slice)-1`. Bounds check.

```
panic: runtime error: invalid memory address or nil pointer dereference
```
You dereferenced a `nil` pointer. The stack trace shows where. Check for `nil` before `*p` or `p.Field`.

```
fatal error: concurrent map writes
```
Two goroutines wrote to the same map without synchronisation. Use a `sync.Mutex` or a `sync.Map` or a channel.

```
fatal error: all goroutines are asleep - deadlock!
```
Every goroutine is blocked. Most often: an unbuffered channel send with no receiver, or a `sync.WaitGroup.Wait()` whose `Done` never gets called.

```
panic: runtime error: ...
```
Generic runtime panic. Read the message; the cause is usually right there.

## Hands-On

```bash
# 1. Check version
$ go version
go version go1.24.2 darwin/arm64

# 2. See your environment
$ go env
GO111MODULE='on'
GOARCH='arm64'
GOOS='darwin'
GOROOT='/usr/local/go'
GOPATH='/Users/you/go'
GOMODCACHE='/Users/you/go/pkg/mod'
GOPROXY='https://proxy.golang.org,direct'
...

# 3. Run a one-file program
$ go run main.go
hi

# 4. Build current package
$ go build

# 5. Build with custom output name
$ go build -o myapp

# 6. Test everything
$ go test ./...

# 7. Verbose, run a specific test
$ go test -v -run TestAdd

# 8. Race-check tests
$ go test -race ./...

# 9. Run benchmarks
$ go test -bench=.

# 10. Coverage
$ go test -cover ./...
ok  myapp 0.142s coverage: 87.5% of statements

# 11. HTML coverage report
$ go test -coverprofile=cov.out && go tool cover -html=cov.out

# 12. Static analysis
$ go vet ./...

# 13. Format
$ go fmt ./...

# 14. Show formatting diff
$ gofmt -d -s .

# 15. Sort imports
$ goimports -l -d .

# 16. Install a binary into ~/go/bin
$ go install ./cmd/myapp

# 17. Update all dependencies
$ go get -u ./...

# 18. Init a module
$ go mod init github.com/me/myapp

# 19. Tidy go.mod and go.sum
$ go mod tidy

# 20. Just download deps to module cache
$ go mod download

# 21. Why is this dep here?
$ go mod why github.com/some/pkg

# 22. List all modules in this build
$ go list -m all

# 23. List with available updates
$ go list -m -u all

# 24. Workspace setup
$ go work init && go work use ./a ./b

# 25. Live CPU profile
$ go tool pprof http://localhost:6060/debug/pprof/profile

# 26. View a trace
$ go tool trace trace.out

# 27. Look up docs in your terminal
$ go doc fmt.Println

# 28. Cross-compile
$ GOOS=linux GOARCH=arm64 go build

# 29. Static, stripped
$ CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o myapp

# 30. Static analysis (separate tool)
$ staticcheck ./...

# 31. Aggregated linter
$ golangci-lint run

# 32. The Go LSP (used by editors)
$ gopls

# 33. Debugger
$ dlv debug

# 34. Debug tests
$ dlv test

# 35. Pretty test runner
$ gotestsum
```

## Common Confusions

### nil interface vs nil concrete

This is the #1 Go gotcha. An interface is two things internally: a type and a value. It is `nil` only when **both** are nil.

```go
func returnError() error {
	var p *MyError    // nil concrete pointer
	return p          // returns a non-nil interface wrapping a nil pointer!
}

if err := returnError(); err != nil {
	fmt.Println("got error") // prints!
}
```

The interface has a non-nil type (`*MyError`) so it isn't nil even though the pointer inside is nil. Fix: return `nil` explicitly instead of a typed nil.

### Slice header surprise with append

Two slices may share storage. `append` might or might not reallocate. This means writes through one can leak (or not) into the other. **Solution: copy when in doubt.**

### Map iteration order is randomised

By design. Sort keys if you need order.

### Goroutine leak

```go
go func() {
	v := <-ch    // ch never gets a value -> goroutine blocks forever
}()
```

That goroutine sits in memory until your program exits. Use `context.WithCancel` and `select` on `ctx.Done()` to give your goroutines an exit door.

### Buffered vs unbuffered channels

Unbuffered: send and receive synchronise. Buffered: capacity N. Default to unbuffered. Add a buffer only when you've thought about why.

### Channel close semantics

Close from sender side. Closing a closed channel panics. Sending on a closed channel panics. Receiving from a closed channel returns the zero value with `ok=false`.

### Never close from receiver side

Because how would the receiver know whether the sender will send again? Closing while the sender still wants to send leads to panic.

### context.Background vs context.TODO

Same behaviour. `Background` = "the root, intentionally." `TODO` = "I haven't decided yet." Cosmetic. Pick the one that documents your intent.

### defer order is LIFO

```go
defer fmt.Println("a")
defer fmt.Println("b")
defer fmt.Println("c")
// prints c, b, a
```

```
defer stack:
   ┌──────────────┐
   │ Println("c") │  <- runs first
   ├──────────────┤
   │ Println("b") │
   ├──────────────┤
   │ Println("a") │  <- runs last
   └──────────────┘
```

### deferred args evaluated immediately

```go
x := 1
defer fmt.Println(x)  // captures x=1 NOW
x = 2
// at return: prints 1
```

Want the late value? Wrap in a closure: `defer func() { fmt.Println(x) }()`.

### %w vs %v vs %s in errors

- `%w` wraps the error, preserving the chain for `errors.Is` / `errors.As`. Use **once** per `Errorf`.
- `%v` formats the error using its `Error()` method but does **not** wrap.
- `%s` is the same as `%v` for errors.

### Comparing structs with ==

Structs are `==` comparable only if **every field** is comparable. Slices, maps, and functions are not comparable, so structs containing them are not. Use `reflect.DeepEqual` or write a custom `Equal` method.

### Embedding an interface in a struct

```go
type ReadWriter struct {
	io.Reader
	io.Writer
}
```

You can call `rw.Read(...)` and `rw.Write(...)` directly. The struct **promotes** the embedded interface's methods. If both embeds have the same method, you get an ambiguity error.

### Passing interface vs concrete type

A function parameter typed as an interface accepts any type with the right methods. A function parameter typed as a concrete struct accepts only that struct. Prefer **interfaces** at API boundaries; prefer **concrete types** internally for clarity and performance.

### Goroutine ID is hidden by design

There is no public API to get a goroutine's ID. The Go authors intentionally hide it so you don't build thread-local storage hacks. If you need request-scoped state, use `context.Context`.

## Vocabulary

| Word | Meaning |
| --- | --- |
| Go | The language. |
| golang | The web-search-friendly nickname (because "go" is a common word). |
| gopher | The mascot — a small blue rodent designed by Renee French. |
| gc | Two meanings: the official Go compiler (called "gc," the c stands for "compiler"), and the garbage collector. Context tells which. |
| gccgo | An alternative Go front end built on top of GCC. Rarely used today. |
| tinygo | A LLVM-based Go compiler for embedded devices and WebAssembly. |
| GOPATH | Old-school workspace directory. Pre-modules. Mostly legacy. |
| GOROOT | Where Go itself is installed. |
| GOPROXY | The module proxy URL(s). Default `https://proxy.golang.org,direct`. |
| GONOSUMCHECK | Skip checksum verification for paths matching this pattern. Used for private modules. |
| GOFLAGS | Default flags applied to every `go` invocation. |
| GOMODCACHE | Where downloaded modules are cached. Default `$GOPATH/pkg/mod`. |
| go.mod | The module manifest. Names the module, declares Go version, lists dependencies. |
| go.sum | Cryptographic hashes of every dependency. Commit it. |
| replace directive | In go.mod, redirect a module path to a local path or a fork. |
| exclude directive | In go.mod, exclude a specific dependency version. |
| retract directive | In your own module's go.mod, mark a published version as unsafe. |
| vendor/ | Optional directory holding copies of your dependencies. `go mod vendor`. |
| workspace | Multi-module setup using `go.work`. |
| go.work | The workspace manifest. |
| module | A versioned bundle of packages. |
| package | A directory of `.go` files sharing a `package X` clause. |
| internal package | A package under a directory called `internal/` — can only be imported by code rooted at the parent of that internal/. |
| main package | The special package that makes an executable. |
| init function | `func init()` in any file runs at package load, before `main`. |
| blank import | `import _ "foo"` — pulls in a package only for its side effects (its `init()`). |
| import alias | `import f "fmt"` — give a different local name to a package. |
| dot import | `import . "fmt"` — pull names directly into the file scope. Avoid in production code. |
| interface | A set of method signatures. |
| struct | A named record type. |
| type alias | `type A = B` — A is literally another name for B. |
| type definition | `type A B` — A is a new type with the same underlying representation as B. |
| named type | A type given a name with `type X ...`. |
| unnamed type | A type written inline, like `struct{X int}`. |
| identical types | Two types with the same definition (matters for assignment). |
| assignable | Whether a value of one type can go in a variable of another. |
| convertible | Whether `T(x)` is legal. |
| comparable | Supports `==` and `!=`. |
| slice | A header (pointer, len, cap) into an array. |
| slice header | The 24-byte (on 64-bit) struct that *is* a slice value. |
| ptr/len/cap | The three fields of a slice header. |
| array | A fixed-size sequence of values, written `[N]T`. |
| map | A hash table. |
| channel | A typed conveyor belt between goroutines. |
| send statement | `ch <- v`. |
| receive statement | `<-ch`. |
| select | A switch over channel operations. |
| default branch | The case in a select that runs if nothing else is ready. |
| range | The keyword that lets `for` iterate. |
| for clause | The C-style three-part for: `for init; cond; post`. |
| infinite for | `for { }`. |
| break | Exit the innermost loop or switch. |
| continue | Skip to the next iteration. |
| goto | Jump to a label. Rarely used. |
| switch | Multi-way branch. |
| type switch | `switch x.(type) { ... }`. |
| fallthrough | Inside a switch case, continue to the next case. |
| defer | Schedule a call to run when the surrounding function returns. |
| panic | Abort the goroutine with an error, unwinding through deferreds. |
| recover | Inside a deferred function, stop a panic. |
| goroutine | A lightweight thread managed by the Go runtime. |
| scheduler | The runtime piece that maps goroutines to OS threads. |
| GMP | Goroutine, Machine (OS thread), Processor — the three things in the scheduler. |
| GOMAXPROCS | How many P's the scheduler uses. Default = number of CPUs. |
| runtime.LockOSThread | Pin this goroutine to its current OS thread. Needed for some C interop. |
| runtime.GC | Trigger a GC explicitly. Almost never use. |
| runtime.NumGoroutine | How many goroutines exist right now. |
| runtime.SetFinalizer | Run a function when an object is collected. Use carefully. |
| sync.Mutex | Mutual exclusion lock. |
| sync.RWMutex | Reader/writer lock. |
| sync.WaitGroup | Wait for a group of goroutines to finish. |
| sync.Once | Run a function exactly once. |
| sync.Cond | Condition variable. Rarely needed; channels are usually better. |
| sync.Pool | Pool of reusable objects to reduce allocations in hot paths. |
| sync.Map | A concurrent map; only better than a `Mutex+map` when keys rarely overlap between goroutines. |
| sync.OnceFunc | (Go 1.21+) wraps a function to be called once. |
| sync.OnceValue | (Go 1.21+) once-eval of a `func() T`. |
| sync.OnceValues | (Go 1.21+) once-eval of a `func() (T, U)`. |
| atomic.Int32 | Typed atomic int32. (Go 1.19+) |
| atomic.Int64 | Typed atomic int64. |
| atomic.Uint32 | Typed atomic uint32. |
| atomic.Uint64 | Typed atomic uint64. |
| atomic.Pointer | Typed atomic pointer. |
| atomic.Bool | Typed atomic bool. |
| atomic.Value | Holds an arbitrary value swapped atomically. |
| atomic.LoadInt32 | Legacy procedural atomic. Prefer the typed `atomic.Int32`. |
| context.Context | Carries deadlines, cancellation, and request-scoped data. |
| context.Background | The root context. |
| context.TODO | Same as Background; documents "I'll figure this out later." |
| context.WithCancel | Returns a child ctx and a cancel function. |
| context.WithCancelCause | (Go 1.20+) Cancel function takes an error explaining why. |
| context.WithTimeout | Auto-cancel after a duration. |
| context.WithDeadline | Auto-cancel at a wall time. |
| context.WithValue | Attach a key/value pair. |
| context.AfterFunc | (Go 1.21+) Run a function when ctx is cancelled. |
| io.Reader | One method `Read(p []byte) (n int, err error)`. |
| io.Writer | One method `Write(p []byte) (n int, err error)`. |
| io.Closer | One method `Close() error`. |
| io.ReadCloser | Reader + Closer. |
| io.WriterAt | Write at offset. |
| io.Seeker | Move the read/write position. |
| io.Pipe | An in-memory pipe (a Reader and a Writer connected). |
| io.Copy | Copy from a Reader to a Writer until EOF. |
| io.MultiWriter | One Writer that fans out to many. |
| io.TeeReader | A Reader whose Reads also write to a Writer. |
| io/fs.FS | (Go 1.16+) Generic read-only file system interface. |
| embed.FS | A filesystem embedded into the binary at compile time. |
| http.Handler | One method `ServeHTTP(ResponseWriter, *Request)`. |
| http.HandlerFunc | A function type adapting a func to a Handler. |
| http.ServeMux | Default HTTP request router. |
| http.NewServeMux | (Go 1.22 enhanced) supports method-and-pattern matching. |
| http.Request | An incoming or outgoing HTTP request. |
| http.ResponseWriter | The write side of an HTTP response. |
| http.Server | A configurable HTTP server. |
| http.Client | An HTTP client. Reuse one across requests. |
| http.Transport | Lower-level RoundTripper backing http.Client. |
| http.Cookie | An HTTP cookie. |
| http.Header | A map of header names to value lists. |
| encoding/json | JSON marshalling and unmarshalling. |
| json.Marshal | Convert a Go value to JSON bytes. |
| json.Unmarshal | Decode JSON bytes into a Go value via pointer. |
| json.Decoder | Streaming JSON reader. |
| json.Encoder | Streaming JSON writer. |
| json.RawMessage | Defers parsing of a JSON sub-tree. |
| json struct tag | The `json:"name,omitempty"` annotation on struct fields. |
| encoding/gob | Go's native binary encoding. |
| encoding/xml | XML encoding/decoding. |
| encoding/csv | CSV reader/writer. |
| encoding/binary | Read/write fixed-width integers. |
| encoding/base64 | Base-64 encoding. |
| encoding/hex | Hex encoding. |
| log | Standard library logger (basic). |
| log/slog | (Go 1.21+) Structured logger. |
| error | The built-in error interface. |
| errors.New | Make a new error from a string. |
| errors.Is | Walk the wrap chain checking for a sentinel match. |
| errors.As | Walk the wrap chain extracting an error of a given type. |
| errors.Unwrap | Pull the wrapped error out of an error. |
| errors.Join | (Go 1.20+) Combine multiple errors. |
| fmt.Errorf | Format-printf for errors; `%w` wraps. |
| panic | Abort flow. |
| recover | Trap a panic in a deferred call. |
| runtime.Goexit | Exit the current goroutine, running its deferreds. |
| build constraint | A `//go:build linux && amd64` line that limits which OS/arch a file compiles for. |
| //go:build | The modern build-constraint syntax (Go 1.17+). |
| //go:embed | Compile-time file embedding directive. |
| //go:generate | Annotate code with a command to run via `go generate`. |
| //go:noinline | Tell the compiler not to inline this function. |
| //go:linkname | Tell the compiler to link this symbol to another package's. Rarely safe. |
| cgo | The bridge between Go and C. |
| C.func | Calling a C function from Go via cgo. |
| import "C" | The magic import that turns on cgo. |
| CGO_ENABLED | Env var; set to 0 for fully static binaries. |
| syscall | Low-level OS calls. Mostly superseded by `golang.org/x/sys`. |
| golang.org/x/sys | The actively-maintained syscall layer. |
| x/sync | Extra concurrency primitives (errgroup, semaphore). |
| x/exp | Experimental packages. |
| x/tools | Compiler tools (goimports, gopls is built here). |
| gopls | The official Go language server (used by editors). |
| staticcheck | The leading static analysis tool. |
| golangci-lint | Aggregator linter. Runs many linters at once. |
| govulncheck | Reports known CVEs in your dependencies. |
| gofumpt | Stricter gofmt. |
| goimports | gofmt + auto-managed imports. |
| gofmt -s | gofmt with simplifications. |
| godoc | The classic doc tool. |
| pkgsite | The newer doc site (powers pkg.go.dev). |
| delve | The Go debugger. |
| dlv | The Delve binary. |
| dlv attach | Attach to a running Go process. |
| dlv core | Debug from a core dump. |
| escape analysis | The compiler decides whether a value lives on the stack or heap. |
| inline budget | The compiler inlines functions only up to a complexity limit. |
| stack growth | Goroutine stacks start small (~2KB) and grow as needed. |
| GC pauses | The brief stop-the-world phases of garbage collection. |
| garbage collector | Concurrent tri-color mark-sweep, low-pause. |
| generational hint | Go's GC is **not** generational (unlike Java's HotSpot). |
| pacer | The GC's algorithm for deciding when to start the next cycle. |
| GOGC | GC trigger threshold (default 100 = 100% growth). |
| GOMEMLIMIT | (Go 1.19+) Soft memory limit. |
| GODEBUG | Comma-separated debug knobs (gctrace=1, schedtrace=1000, ...). |
| gctrace | GODEBUG flag to print every GC cycle. |
| schedtrace | GODEBUG flag to print scheduler state periodically. |
| fastrand | Internal fast pseudo-random — exposed indirectly via math/rand/v2 (Go 1.22). |
| race detector | -race flag at build/test time. |
| -race | The flag that enables the race detector. |
| msan | Memory sanitiser (Linux/clang). |
| asan | Address sanitiser (Linux/clang). |
| cover | `-cover` enables coverage reporting. |
| profile | A sampled snapshot of execution. |
| pprof | The profile viewer tool. |
| trace | An execution trace, deeper than a profile. |
| benchmark | A `BenchmarkX(b *testing.B)` function. |
| b.N | Iteration count chosen by the framework. |
| b.ResetTimer | Reset the timer after expensive setup. |
| table test | A test driven by a slice of input/expected struct cases. |
| t.Run | Create a named subtest. |
| t.Parallel | Mark this test/subtest to run in parallel with others. |
| t.Cleanup | Register a cleanup callback. |
| t.Helper | Mark a fn as a test helper. |
| t.Setenv | (Go 1.17+) Set an env var for the test. |
| testing/iotest | Helper readers (one byte at a time, error after N bytes, etc.). |
| testing/quick | Property testing helpers (predates fuzz). |
| fuzz testing | (Go 1.18+) Random input generation. |
| F.Add | Seed a fuzz target with an example. |
| F.Fuzz | Run the fuzz target. |
| F.Skip | Skip a fuzz run. |
| generics | (Go 1.18+) Type parameters. |
| type parameter | A named placeholder for a type, declared in `[ ... ]`. |
| constraint | An interface restricting a type parameter. |
| ~T | "Any type whose underlying type is T." Used in constraints. |
| comparable | Built-in constraint for types supporting ==. |
| any | Alias for `interface{}`. |
| ordered | The constraint for `<`, `>` (in `cmp.Ordered`, Go 1.21). |
| iter.Seq | (Go 1.23+) The standard iterator type. |

## Try This

Bake one program from each of these:

1. A "hello world" — `go run main.go`.
2. A web server that responds with "hi" on port 8080.
3. A program that reads a JSON file and prints one field.
4. A program that fans out 10 goroutines to fetch 10 URLs concurrently and prints the response sizes.
5. A program with a `BenchmarkX` and run `go test -bench=.`.
6. A program with a `FuzzX` and run `go test -fuzz=FuzzX -fuzztime=10s`.
7. A cross-compiled binary: `GOOS=linux GOARCH=arm64 go build`.
8. A static binary: `CGO_ENABLED=0 go build -trimpath -ldflags='-s -w'`.
9. A module with a `replace` directive pointing at a local fork.
10. A program that uses `context.WithTimeout` to give a child operation 2 seconds.

## Where to Go Next

Read **Effective Go** at go.dev/doc/effective_go. Then **The Go Programming Language** by Donovan and Kernighan front-to-back. Then **100 Go Mistakes and How to Avoid Them** by Teiva Harsanyi. Then sample real Go: read the `net/http` source, read the standard library `errors` package, read a small project like `caddy` or `golang.org/x/sync/errgroup`.

After that, build something real. A CLI. A tiny REST service. A game with `ebitengine`. Whatever scratches an itch.

## See Also

- `languages/go` — the dense reference sheet.
- `languages/c` — the language Go was built to replace.
- `languages/rust` — a different answer to the same questions.
- `languages/typescript` — for the JS-side; another statically-typed friend.
- `languages/python` — for the script-side; great glue for a Go binary.
- `system/delve` — the Go debugger.
- `ramp-up/rust-eli5` — Rust from scratch in plain English.
- `ramp-up/python-eli5` — Python from scratch in plain English.
- `ramp-up/git-eli5` — Git from scratch in plain English.
- `ramp-up/linux-kernel-eli5` — what runs underneath your Go program.

## References

- The Go website — go.dev/doc — official docs, language spec, blog.
- *The Go Programming Language* by Alan A. A. Donovan and Brian W. Kernighan. The book.
- *100 Go Mistakes and How to Avoid Them* by Teiva Harsanyi. Read this once you have written a few hundred lines.
- *Go Crazy* and the Ardan Labs blog by Bill Kennedy — opinionated, deep posts on idiomatic Go.
- The Go FAQ — go.dev/doc/faq — answers to every "why doesn't Go have X" you will ever ask.
- Effective Go — go.dev/doc/effective_go — the unofficial style and idioms bible.
- The Go Blog — go.dev/blog — release notes, deep dives, and language history.
