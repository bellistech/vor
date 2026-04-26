# programming-paradigms

A paradigm is a *style* of building programs — a set of concepts that shape how problems are decomposed, how state is managed, and how computation is expressed. Most production languages are multi-paradigm: pick the style that fits the problem, not the one your language defaults to.

## Setup

A **paradigm** is not a feature checklist; it is a way of thinking about computation. *Imperative* says "do this, then that." *Functional* says "the answer is this expression." *Logic* says "here are the facts; find the answer." Two programs that compute the same result can look totally different across paradigms.

Most modern languages are **multi-paradigm**. Java is not "pure OO" — since Java 8 it has lambdas, streams, and (since 17) pattern matching. Python supports OO, procedural, and functional styles in the same file. C++ does almost everything: imperative, OO, generic, functional, concurrent. Even C, the canonical procedural language, can fake OO with structs of function pointers (look at the Linux kernel's VFS layer).

Why does the categorization matter?

- **Mental model.** Knowing the paradigm tells you what the *idiomatic* solution looks like. In Haskell, you transform data with pure functions; in Java, you mutate state through methods.
- **Trade-offs.** Each paradigm has strengths and pathologies. Functional is great for data pipelines but awkward for I/O. OO models domains well but invites coupling. Logic is concise for search problems but slow for arithmetic.
- **Polyglot programming.** A senior engineer recognizes the paradigm a codebase wants and follows it, even when the language allows alternatives.
- **Language design.** Understanding paradigms makes new languages cheap to learn — you map them onto familiar concepts (Rust's enum = ML's algebraic data types; Go's interface = Haskell's type class minus the inheritance).

A short taxonomy of the major axes:

```text
              Imperative ───── Declarative
                  │                 │
            Procedural        Functional ── Logic
                  │             │       │
                  OO          Pure   Datalog/Prolog
                  │           Impure
            Class-based / Prototype
```

Cross-cutting paradigms (orthogonal to the above): generic, metaprogramming, aspect-oriented, reactive, concurrent, parallel, dataflow, event-driven, array, concatenative, symbolic, differentiable, quantum.

The rest of this sheet walks each one with concrete code.

## Imperative Paradigm

The oldest and most direct style. A program is a sequence of commands that **change state** — variables, memory, I/O. The compiler/CPU model is itself imperative (load, store, jump), so this paradigm has the smallest abstraction gap.

Signature features:

- **Mutability** — variables are assigned and re-assigned.
- **Sequential execution** — statements run in order.
- **Explicit control flow** — `goto`, loops, conditionals.

Canonical examples: assembly, C, BASIC, FORTRAN-77.

Sum 1..n in C:

```c
int sum(int n) {
    int total = 0;
    for (int i = 1; i <= n; i++) {
        total += i;          // mutate state
    }
    return total;
}
```

Sum 1..n in x86-64 assembly (NASM):

```asm
; rdi = n, returns rax = sum
sum:
    xor     rax, rax        ; total = 0
    xor     rcx, rcx        ; i = 0
.loop:
    inc     rcx             ; i++
    cmp     rcx, rdi
    jg      .done
    add     rax, rcx        ; total += i
    jmp     .loop
.done:
    ret
```

The C version is imperative in *intent*; the assembly version is imperative in *implementation*. Same mental model: mutate accumulator, branch on counter.

Pure imperative drawbacks (which drove the next paradigms):

- Spaghetti code from `goto`.
- Hard to reason about state at any program point.
- No abstraction over repeated patterns beyond macros.

## Procedural Paradigm

Imperative + **procedure abstraction**. A procedure (function, subroutine) groups a sequence of statements behind a name and a parameter list. This is the first paradigm to introduce **modular decomposition** — break a program into named, callable units.

Languages: C, Pascal, FORTRAN-77 onward, Modula-2, Ada (mostly), early BASIC with `GOSUB`.

Same `sum` problem, refactored procedurally:

```c
// reusable procedure
int range_sum(int lo, int hi) {
    int total = 0;
    for (int i = lo; i <= hi; i++) total += i;
    return total;
}

int main(void) {
    printf("%d\n", range_sum(1, 100));
    printf("%d\n", range_sum(50, 60));
    return 0;
}
```

Procedural style scales by:

- **Top-down design** — start with `main`, decompose into procedures, decompose those, until each procedure is small enough to fit in a head.
- **Stepwise refinement** (Wirth) — write the algorithm in pseudocode, replace each step with a procedure call.
- **Stack-based locals** — each call gets its own activation record; no globals required.

Pascal exemplifies it (procedures *and* functions, type-safe records, no OO):

```pascal
program SumDemo;
var i, total: Integer;
begin
    total := 0;
    for i := 1 to 100 do
        total := total + i;
    writeln(total)
end.
```

Limit: procedural code with growing data still couples the data layout to every procedure that touches it. That's where OO comes in.

## Structured Programming

Edsger Dijkstra's 1968 letter **"Goto Considered Harmful"** triggered an industry-wide rejection of unrestricted jumps. Bohm and Jacopini had proven (1966) that any flowchart can be expressed using just three control structures:

1. **Sequence** — A; B
2. **Selection** — `if/then/else`, `switch`
3. **Iteration** — `while`, `for`

Structured programming = procedural + the discipline to use only these three (plus single-entry/single-exit functions). The ALGOL/Pascal/C lineage embeds this directly.

Before:

```c
    int x = 0;
loop:
    if (x >= 10) goto done;
    printf("%d\n", x);
    x++;
    goto loop;
done:
```

After:

```c
    for (int x = 0; x < 10; x++) {
        printf("%d\n", x);
    }
```

Same behavior, but the second is *locally* understandable — you see the bounds, the increment, the body, all in one place.

Modern languages effectively forbid `goto` (or scope it heavily — Go's `goto` cannot jump over variable declarations or into blocks). Even C's `goto` is mostly tolerated for one pattern: cleanup at function exit (the "goto fail" idiom, ironically named after the iOS bug it caused).

```c
int do_thing(void) {
    FILE *f = fopen("x", "r");
    if (!f) return -1;
    void *buf = malloc(1024);
    if (!buf) goto err_close;
    if (read_all(f, buf) < 0) goto err_free;
    process(buf);
    free(buf);
    fclose(f);
    return 0;
err_free:
    free(buf);
err_close:
    fclose(f);
    return -1;
}
```

Languages with RAII or `defer` (C++, Rust, Go, Swift) make even this `goto` unnecessary.

## Object-Oriented Paradigm

Combine **data** and the **operations on that data** into a single unit (object, class). The four pillars:

1. **Encapsulation** — internal state is hidden; access is via methods.
2. **Inheritance** — a subclass extends/refines a superclass.
3. **Polymorphism** — same call site, different runtime behaviour based on object type.
4. **Abstraction** — interfaces / abstract classes describe what an object *does*, not what it *is*.

Coined by Alan Kay (Smalltalk, 1972), popularized by C++, then Java, then C#/Python/Ruby. Smalltalk's slogan was **"everything is an object"**: integers, classes, even the runtime itself.

Class-based OO in Java:

```java
abstract class Shape {
    abstract double area();
    void describe() { System.out.println("area = " + area()); }
}
class Circle extends Shape {
    final double r;
    Circle(double r) { this.r = r; }
    double area() { return Math.PI * r * r; }
}
class Square extends Shape {
    final double s;
    Square(double s) { this.s = s; }
    double area() { return s * s; }
}
Shape[] shapes = { new Circle(1.0), new Square(2.0) };
for (Shape sh : shapes) sh.describe();   // dynamic dispatch
```

The same idea in Python:

```python
class Shape:
    def area(self): raise NotImplementedError
    def describe(self): print("area =", self.area())

class Circle(Shape):
    def __init__(self, r): self.r = r
    def area(self): return 3.14159 * self.r ** 2

class Square(Shape):
    def __init__(self, s): self.s = s
    def area(self): return self.s ** 2

for sh in [Circle(1.0), Square(2.0)]:
    sh.describe()
```

OO design ladder:

- **Abstract Data Types** (Liskov) — public ops + hidden representation.
- **Composition over inheritance** — has-a beats is-a for re-use; inheritance buys polymorphism but couples tightly.
- **Liskov Substitution Principle** — a subtype must be usable wherever its supertype is expected (the L in SOLID).
- **Open/Closed Principle** — open for extension, closed for modification.

Pathology: deep inheritance hierarchies, the **fragile base class problem**, the **diamond problem** (multiple inheritance), and *anaemic domain models* (data-only classes with logic in services — barely OO at all).

Smalltalk-style message-passing OO (Smalltalk, Ruby's `send`, Objective-C):

```ruby
[circle area]                  # Objective-C: send "area" message to circle
circle.send(:area)             # Ruby: same idea
```

vs C++/Java/C# class-based OO where method calls are looked up at compile/JIT time on a vtable.

## Functional Paradigm

A program is a **composition of pure functions**. Computation is the *evaluation of expressions*, not the execution of statements. Variables are immutable bindings, not memory cells.

Key tenets:

- **Pure functions** — same input → same output, no side effects.
- **Immutability** — data structures are not mutated; transformations return new values.
- **First-class functions** — functions can be passed, returned, stored.
- **Higher-order functions** — functions that take or return functions.
- **Referential transparency** — any expression can be replaced by its value without changing meaning.

Mathematical foundation: **lambda calculus** (Church, 1930s). `λx.x+1` is a function that adds 1. Function application is `(λx.x+1) 3 = 4`. That's it — the entire model of computation.

Languages: Haskell (pure), OCaml (impure ML), F#, Lisp/Scheme/Racket/Clojure, Erlang, Elm, Elixir, PureScript, ReScript.

Sum 1..n functionally (Haskell):

```haskell
sumTo :: Int -> Int
sumTo n = sum [1..n]
-- or
sumTo n = foldr (+) 0 [1..n]
-- or with fold over recursion
sumTo 0 = 0
sumTo n = n + sumTo (n - 1)
```

No loop counter, no mutable accumulator — just an expression that *equals* the answer.

Higher-order in OCaml:

```ocaml
let twice f x = f (f x)
let inc x = x + 1
let _ = twice inc 5            (* 7 *)
```

Functional patterns you'll see everywhere:

- `map`, `filter`, `reduce/fold` — operate on collections without explicit loops.
- **Function composition** — `(f . g) x = f (g x)`.
- **Recursion** — replaces iteration; tail-call optimization makes it cheap.
- **Closures** — functions that capture surrounding variables.
- **Currying** — `f(a, b, c)` ≡ `f(a)(b)(c)` — every multi-arg function is really a chain of single-arg functions.

## Pure-Functional vs Impure-Functional

A **pure** functional language guarantees referential transparency — there's no way to write a function that secretly does I/O, throws, or mutates global state. The compiler enforces it through the type system.

| Pure | Impure (allows mutation/IO) |
|------|----------------------------|
| Haskell | OCaml |
| Idris | F# |
| Elm | Lisp / Scheme / Racket |
| PureScript | Clojure (mostly functional, allows refs) |
| Agda | Standard ML |
|      | Erlang (functional but actors mutate) |

In Haskell, **side effects are encoded in the type system** through `IO` and other monads. A function with no `IO` in its type *cannot* perform I/O — guaranteed by the compiler:

```haskell
add :: Int -> Int -> Int          -- pure; cannot read a file
add x y = x + y

readConfig :: IO String           -- type says "this performs IO"
readConfig = readFile "config.txt"
```

The escape hatch is the `IO` monad — a type that *describes* an effect to be run by the runtime, rather than performing it inline. Impure languages let you do it freely:

```ocaml
let read_config () = open_in "config.txt" |> input_line   (* OCaml: no type tag *)
```

Clojure, Lisp, OCaml all let you mutate, print, throw, anywhere. Discipline is cultural, not enforced.

How **monads** sneak side effects back into Haskell:

```haskell
main :: IO ()
main = do
    name <- getLine                          -- read stdin
    putStrLn ("Hello, " ++ name)             -- print stdout
```

Under the hood, `do`-notation is sugar for `>>=` (bind). Monads sequence effects without breaking purity — they package effects as values that the runtime executes.

## Logic Paradigm

A program is a set of **facts** and **rules**; you ask **queries**, the engine searches for answers via **unification** and **backtracking**.

Languages: Prolog, Datalog, Mercury, Answer Set Programming (ASP), miniKanren (an embedded DSL).

Prolog example — the Royal Family puzzle:

```prolog
parent(elizabeth, charles).
parent(charles, william).
parent(charles, harry).
parent(william, george).
parent(william, charlotte).

grandparent(GP, GC) :- parent(GP, P), parent(P, GC).
ancestor(A, D) :- parent(A, D).
ancestor(A, D) :- parent(A, X), ancestor(X, D).

?- grandparent(elizabeth, X).
% X = william ; X = harry.

?- ancestor(elizabeth, george).
% true.
```

How it works:

1. The query `grandparent(elizabeth, X)` asks: find all `X` such that the rule succeeds.
2. The engine **unifies** `grandparent(GP, GC)` with `grandparent(elizabeth, X)`, binding `GP=elizabeth, GC=X`.
3. It tries `parent(elizabeth, P)` — finds `P=charles`.
4. Then `parent(charles, X)` — finds `X=william` (yields a solution), backtracks, finds `X=harry`.

**Datalog** is a restricted subset: no function symbols, guaranteed termination, used in static analysis (Soufflé, Datomic queries).

```datalog
parent(elizabeth, charles).
parent(charles, william).
ancestor(X, Y) :- parent(X, Y).
ancestor(X, Y) :- parent(X, Z), ancestor(Z, Y).
```

**ASP (Answer Set Programming)** — Clingo, DLV. Used for combinatorial search:

```text
% Map coloring
color(red). color(green). color(blue).
{ assigned(N, C) : color(C) } = 1 :- node(N).
:- edge(N1, N2), assigned(N1, C), assigned(N2, C).
```

Each "answer set" is a valid coloring.

Logic paradigm shines for:

- Symbolic reasoning, theorem proving.
- Declarative queries (Datalog ≈ recursive SQL).
- Constraint satisfaction.
- Compiler analyses (Datomic-style).

Weakness: arithmetic is awkward (no native floats), I/O bolted on, performance unpredictable.

## Declarative Paradigm

The umbrella term: programs say **WHAT** the answer is, not **HOW** to compute it. Logic and pure-functional are subsets.

Examples:

**SQL** — describe the result set, the engine plans the query:

```sql
SELECT customer_id, SUM(total) AS lifetime_value
FROM orders
WHERE created_at >= '2024-01-01'
GROUP BY customer_id
HAVING SUM(total) > 1000
ORDER BY lifetime_value DESC
LIMIT 10;
```

You did not write a join algorithm, a sort, or a hash table. You described the shape of the answer.

**HTML/CSS** — describe the rendered document; the browser computes layout:

```html
<div class="card">
  <h2>Title</h2>
  <p>Body</p>
</div>
<style>
  .card { display: grid; gap: 0.5rem; padding: 1rem; }
</style>
```

**Regular expressions** — describe a *language*, the engine builds an automaton:

```text
^([A-Z]{2,3})-(\d{4})$
```

**Make/Bazel** — describe targets and dependencies; the tool orders the build:

```makefile
out.bin: a.o b.o
	cc -o $@ $^
%.o: %.c
	cc -c $<
```

**Terraform / Kubernetes manifests** — describe desired infrastructure state, the controller reconciles:

```hcl
resource "aws_s3_bucket" "logs" {
  bucket = "my-logs"
  versioning { enabled = true }
}
```

**XSLT, JSONPath, GraphQL** — query/transform languages.

The win: declarative code is shorter, the engine optimises freely (SQL planner reorders joins), and intent is clearer. The cost: when the engine guesses wrong, debugging means understanding *its* model.

## Dataflow Paradigm

Computation is a **directed graph** of nodes; values flow along edges. A node fires when its inputs are ready. Sometimes called *pipeline* or *stream-processing* programming.

Languages: LabVIEW (graphical), Lustre, Esterel (synchronous, used in avionics), Lucid, Verilog/VHDL (hardware), TensorFlow's static graphs (TF1.x), Apache Beam.

Conceptual graph for `(a + b) * (a - b)`:

```text
   a ─┬───┐    ┌─── +
      │   ├──> │
   b ─┼───┘    └─── -  ───> *  ───> result
      │              │
      └──────────────┘
```

When `a` and `b` arrive, `+` and `-` fire in parallel; their outputs feed `*`.

Lustre (synchronous dataflow used in safety-critical systems):

```text
node Counter(reset: bool) returns (n: int);
let
    n = if reset then 0 else (0 -> pre n + 1);
tel
```

`->` is "initially / then"; `pre` is "previous value". Each cycle, `n` is the previous `n` plus 1, unless `reset`.

Modern reactive frameworks (RxJS, Reactor, Akka Streams) are dataflow at runtime. **Excel** is dataflow too — change a cell, dependent cells recompute.

```javascript
// RxJS dataflow
const a$ = source$.pipe(map(x => x.a));
const b$ = source$.pipe(map(x => x.b));
const sum$ = combineLatest([a$, b$]).pipe(map(([a, b]) => a + b));
const prod$ = combineLatest([a$, b$]).pipe(map(([a, b]) => a * b));
combineLatest([sum$, prod$]).subscribe(console.log);
```

## Reactive Paradigm

A specific flavour of dataflow centred on **observable streams of events** with declarative transformations: `map`, `filter`, `merge`, `combineLatest`, `debounceTime`, `flatMap`. Push-based: producers notify consumers when new values arrive.

Libraries: ReactiveX (RxJS, RxJava, RxSwift, RxKotlin), Project Reactor, Akka Streams, Combine (Apple).

Reactive search box (RxJS):

```javascript
import { fromEvent } from 'rxjs';
import { map, debounceTime, distinctUntilChanged, switchMap } from 'rxjs/operators';

fromEvent(input, 'input').pipe(
    map(e => e.target.value),
    debounceTime(300),
    distinctUntilChanged(),
    switchMap(q => fetch(`/api/search?q=${q}`).then(r => r.json()))
).subscribe(results => render(results));
```

No state machine, no manual cancellation — `switchMap` cancels in-flight requests when a new query arrives.

**React** the UI library is "reactive" in spirit: state changes → component re-renders. The view is a *function* of state. SwiftUI and Jetpack Compose adopt the same model.

```jsx
function Counter() {
    const [n, setN] = useState(0);
    return <button onClick={() => setN(n + 1)}>{n}</button>;
}
```

The Reactive Manifesto names four properties: **responsive, resilient, elastic, message-driven** — see the Reactive Streams section below.

## Event-Driven Paradigm

Code reacts to **events** rather than running top-down. The runtime owns the loop; you register callbacks/handlers.

Two flavours:

1. **Callback-based** — Node.js HTTP server, browser DOM events, GUI toolkits.
2. **Actor-based** — Erlang/Elixir, Akka, Microsoft Orleans. Each actor owns state, processes one message at a time from its mailbox.

Browser event:

```javascript
button.addEventListener('click', e => console.log('clicked'));
```

Erlang actor:

```erlang
-module(counter).
-export([start/0, loop/1]).

start() -> spawn(?MODULE, loop, [0]).

loop(N) ->
    receive
        {inc, From} -> From ! ok, loop(N + 1);
        {get, From} -> From ! N, loop(N);
        stop        -> ok
    end.
```

Each actor is its own concurrency unit. Messages are immutable. State changes only by recursing with new arguments. This is the OTP philosophy:

```text
send  : inbox grows
receive : pattern-match the next message
mutate state : recursive call with new args
```

Akka Typed (Scala):

```scala
sealed trait Msg
case object Inc extends Msg
case class Get(replyTo: ActorRef[Int]) extends Msg

def counter(n: Int): Behavior[Msg] = Behaviors.receiveMessage {
    case Inc       => counter(n + 1)
    case Get(rt)   => rt ! n; Behaviors.same
}
```

## Concurrent Paradigm

Multiple **logical** threads of execution, possibly interleaved on a single CPU. (Concurrency ≠ parallelism: concurrency is about *structure*, parallelism is about *simultaneous* execution.)

Three big approaches:

**Threads + locks** (Java, C++, POSIX threads):

```java
class Counter {
    private int n = 0;
    private final Object lock = new Object();
    void inc() {
        synchronized (lock) { n++; }
    }
    int get() {
        synchronized (lock) { return n; }
    }
}
```

Or `java.util.concurrent.atomic.AtomicInteger`, `ReentrantLock`, `Semaphore`, `CountDownLatch`, `ConcurrentHashMap`.

**Goroutines + channels** (Go's CSP-influenced model):

```go
ch := make(chan int)
go func() {
    for i := 1; i <= 10; i++ { ch <- i }
    close(ch)
}()
for v := range ch {
    fmt.Println(v)
}
```

Idiom: **don't communicate by sharing memory; share memory by communicating.**

**Ownership-based** (Rust): the borrow checker proves data-race freedom *at compile time*. A `&mut T` is exclusive; `&T` is shared but read-only; cross-thread requires `Send` + `Sync` traits.

```rust
use std::sync::{Arc, Mutex};
use std::thread;

let counter = Arc::new(Mutex::new(0));
let mut handles = vec![];
for _ in 0..10 {
    let c = Arc::clone(&counter);
    handles.push(thread::spawn(move || {
        let mut n = c.lock().unwrap();
        *n += 1;
    }));
}
for h in handles { h.join().unwrap(); }
println!("{}", *counter.lock().unwrap());
```

The compiler refuses code that would race. No silent data corruption.

## Parallel Paradigm

**Multiple physical processors** doing work simultaneously. Concurrency models can map onto parallel hardware, but parallelism is specifically about *speed-up* by using more cores.

Categories (Flynn's taxonomy):

- **SISD** — Single Instruction, Single Data — a regular scalar CPU.
- **SIMD** — Single Instruction, Multiple Data — vector ops, GPUs.
- **MIMD** — Multiple Instruction, Multiple Data — multi-core CPUs, distributed clusters.
- **MISD** — rare.

APIs:

**OpenMP** (C/C++/Fortran shared-memory):

```c
#pragma omp parallel for reduction(+:sum)
for (int i = 0; i < N; i++) sum += a[i];
```

**MPI** (distributed):

```c
MPI_Init(&argc, &argv);
MPI_Comm_rank(MPI_COMM_WORLD, &rank);
MPI_Reduce(&local, &global, 1, MPI_INT, MPI_SUM, 0, MPI_COMM_WORLD);
MPI_Finalize();
```

**CUDA** (NVIDIA GPUs, SIMT execution):

```c
__global__ void add(int *a, int *b, int *c) {
    int i = blockIdx.x * blockDim.x + threadIdx.x;
    c[i] = a[i] + b[i];
}
add<<<numBlocks, threadsPerBlock>>>(d_a, d_b, d_c);
```

**OpenCL** — vendor-neutral GPU/accelerator API.

**SIMD intrinsics** in C (AVX2):

```c
#include <immintrin.h>
__m256 va = _mm256_loadu_ps(a);
__m256 vb = _mm256_loadu_ps(b);
__m256 vc = _mm256_add_ps(va, vb);
_mm256_storeu_ps(c, vc);
```

Languages with parallel-as-paradigm: Chapel, X10, Fortran 2008 coarrays, Halide (image-processing DSL).

## Generic Paradigm

Write **once**, use with **many types**. The mechanism: parameterize over types.

C++ templates:

```cpp
template <typename T>
T max(T a, T b) { return a > b ? a : b; }

int x = max(3, 4);
double y = max(2.5, 3.1);
std::string z = max(std::string("ab"), std::string("cd"));
```

Java generics (erasure-based, runtime is `Object`):

```java
class Box<T> {
    private T value;
    public T get() { return value; }
    public void set(T v) { value = v; }
}
Box<Integer> bi = new Box<>();
```

Rust generics + traits (monomorphized, zero-cost):

```rust
fn max<T: PartialOrd>(a: T, b: T) -> T {
    if a > b { a } else { b }
}
```

C# generics (reified, runtime knows the type parameter):

```csharp
class Box<T> { public T Value { get; set; } }
var b = new Box<int> { Value = 5 };
```

Go generics (added in 1.18):

```go
func Max[T constraints.Ordered](a, b T) T {
    if a > b { return a }
    return b
}
```

Two implementation strategies:

- **Monomorphization** (C++, Rust, Go): the compiler generates a specialized copy of the function per type. Fast, code-size cost.
- **Erasure** (Java, Haskell): one implementation handles all types via boxing or dictionary-passing. Smaller binary, slight runtime cost.

Generic programming also shows up as **higher-kinded types** (Haskell, Scala) — parameterizing over *type constructors*: `Functor f`, where `f` itself takes a type argument.

## Metaprogramming

Programs that **read or write programs**. Several tiers:

1. **Macros** — text/AST rewriting at compile time (C preprocessor, Lisp `defmacro`, Rust `macro_rules!`, Scala macros).
2. **Reflection** — runtime introspection of types and members (Java reflection, C# reflection, Python `inspect`).
3. **Code generation** — generate source code from templates/specs (Go's `go generate`, Protocol Buffers).
4. **Eval / dynamic compilation** — Lisp `eval`, JavaScript `eval`, Ruby `instance_eval`.
5. **Method missing / `__getattr__`** — intercept calls to undefined names (Ruby `method_missing`, Python `__getattr__`, Smalltalk `doesNotUnderstand`).

**Lisp macros** — the gold standard, because *code is data* (homoiconicity):

```lisp
(defmacro unless (cond &rest body)
  `(if (not ,cond) (progn ,@body)))

(unless (zerop x)
  (print "x is non-zero"))
;; expands to: (if (not (zerop x)) (progn (print "x is non-zero")))
```

**Rust declarative macros**:

```rust
macro_rules! square {
    ($x:expr) => { $x * $x };
}
let n = square!(5);   // expands to 5 * 5
```

**Rust procedural macros** — derive serialization, generate code from attributes:

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
struct User { name: String, age: u32 }
```

**Python decorators**:

```python
def memoize(f):
    cache = {}
    def wrapper(*args):
        if args not in cache:
            cache[args] = f(*args)
        return cache[args]
    return wrapper

@memoize
def fib(n): return n if n < 2 else fib(n - 1) + fib(n - 2)
```

**Ruby `method_missing`**:

```ruby
class Builder
    def method_missing(name, *args, &block)
        puts "called #{name} with #{args.inspect}"
    end
end
Builder.new.foo("hi")    # called foo with ["hi"]
```

**C++ templates** as a Turing-complete metaprogramming language (sometimes accidentally so):

```cpp
template <int N> struct Fact { enum { value = N * Fact<N - 1>::value }; };
template <> struct Fact<0> { enum { value = 1 }; };
constexpr int x = Fact<5>::value;   // 120 at compile time
```

**Elixir AST manipulation**:

```elixir
defmacro my_if(condition, do: do_clause, else: else_clause) do
    quote do
        case unquote(condition) do
            true -> unquote(do_clause)
            _    -> unquote(else_clause)
        end
    end
end
```

## Aspect-Oriented Paradigm

Some concerns — logging, transactions, security checks, caching — naturally cross many classes/functions. Embedding them inline produces *tangled* code. AOP factors these **cross-cutting concerns** out into **aspects** that the compiler/weaver inserts where needed.

Vocabulary:

- **Joinpoint** — a point in execution where aspects can apply (method call, field access, exception throw).
- **Pointcut** — predicate selecting joinpoints.
- **Advice** — code to run at matched joinpoints (`before`, `after`, `around`).
- **Aspect** — a module containing pointcuts + advice.

**AspectJ** (Java):

```java
@Aspect
public class LoggingAspect {
    @Around("execution(* com.acme.service.*.*(..))")
    public Object log(ProceedingJoinPoint pjp) throws Throwable {
        long t = System.nanoTime();
        try {
            return pjp.proceed();
        } finally {
            System.out.println(pjp.getSignature() + " took " + (System.nanoTime() - t) + "ns");
        }
    }
}
```

Every method in `com.acme.service.*` is now logged without modifying its source.

**Spring AOP** (proxy-based, more limited but more common in practice):

```java
@Transactional
public void transferFunds(Account from, Account to, BigDecimal amt) {
    from.debit(amt);
    to.credit(amt);
}
```

The `@Transactional` advice wraps the call in begin/commit/rollback.

**Dependency injection containers** (Spring, Dagger, Guice) are a *degenerate form* of AOP — they intercept object construction to inject collaborators rather than inserting cross-cutting code.

PostSharp (C#), Python decorators-as-aspects, Ruby `before_action` filters are all in this family.

## Symbolic Paradigm

Programs **manipulate program-like data** — usually trees of symbols. Often paired with metaprogramming, but the paradigm is more about treating *expressions* (not just values) as the unit of computation.

Lisp's homoiconicity — code IS data (a list):

```lisp
(+ 1 2)                       ; evaluates to 3
'(+ 1 2)                      ; the list (+ 1 2), not evaluated
(eval '(+ 1 2))               ; 3
(let ((expr '(+ 1 2)))
    (eval (list (car expr) 10 20)))  ; rewrites to (+ 10 20), gives 30
```

**Mathematica** / **Wolfram Language** — symbolic algebra is the entire ecosystem:

```mathematica
Solve[x^2 + 5 x + 6 == 0, x]
(* {{x -> -3}, {x -> -2}} *)

D[Sin[x] x^2, x]
(* 2 x Sin[x] + x^2 Cos[x] *)

Integrate[Exp[-x^2], {x, -Infinity, Infinity}]
(* Sqrt[Pi] *)
```

The symbolic engine knows trigonometry, calculus, linear algebra — and rewrites expressions according to mathematical rules.

**Maxima**, **SymPy** (Python) bring symbolic computation to general-purpose languages:

```python
from sympy import symbols, solve, diff, sin, exp, oo, integrate
x = symbols('x')
solve(x**2 + 5*x + 6, x)        # [-3, -2]
diff(sin(x) * x**2, x)          # 2*x*sin(x) + x**2*cos(x)
```

Symbolic AI (older expression of the paradigm): rule-based expert systems, MYCIN, CLIPS — programs reason over symbolic facts and rules.

## Concatenative / Stack-Based

Functions take **stack inputs** and produce **stack outputs**; composition is concatenation. There are no named arguments at the function-definition level; data flows through an implicit stack.

Languages: Forth, Factor, PostScript, Joy, Cat.

**Forth**:

```forth
: square ( n -- n*n )  dup * ;
5 square .       \ prints 25

: cube ( n -- n^3 ) dup square * ;
3 cube .         \ prints 27
```

`dup` duplicates the top of stack; `*` pops two, pushes their product. The function `square` is the concatenation `dup *`.

**PostScript** is a stack-based DSL for printing:

```postscript
72 72 moveto
/Helvetica findfont 24 scalefont setfont
(Hello, world!) show
showpage
```

**Factor** is the modern descendant — a Forth with type inference, GC, and a rich library:

```factor
: fact ( n -- n! )  dup 0 = [ drop 1 ] [ dup 1 - fact * ] if ;
5 fact .          ! 120
```

Mental model: every function is a *transformation* on the stack. Composition is mere juxtaposition. There are no parameter names because positions are implicit.

Performance perk: very small interpreters (Forth implementations sometimes fit in 4KB). Drawback: code becomes terse to the point of being a write-only language without practice.

## Array Paradigm

The unit of computation is the **array** (vector, matrix, n-dim tensor), and operations are **whole-array primitives**: addition, slicing, broadcasting, reductions.

Languages: APL, J, K, Q (kdb+), R, Julia (with broadcasting), MATLAB, NumPy/pandas (Python), xtensor (C++).

**APL** — the original:

```apl
+/ 1 2 3 4 5         ⍝ sum: 15
(+/ ⍵) ÷ ≢⍵          ⍝ mean function
```

**J** (ASCII version of APL):

```j
+/ 1 2 3 4 5    NB. 15
(+/ % #) 1 2 3 4 5    NB. mean = 3
```

**K** (very terse):

```k
+/!100        / sum 0..99 = 4950
```

**NumPy** (Python):

```python
import numpy as np
a = np.array([1, 2, 3])
b = np.array([10, 20, 30])
a + b                # array([11, 22, 33])  -- elementwise
a * 2                # array([2, 4, 6])     -- broadcasting
(a ** 2).sum()       # 14
np.dot(a, b)         # 140
```

**Julia** broadcasting with `.`:

```julia
a = [1, 2, 3]
b = [10, 20, 30]
a .+ b               # 3-element Vector: [11, 22, 33]
a .^ 2 |> sum        # 14
```

**pandas** for tabular data:

```python
import pandas as pd
df = pd.read_csv("sales.csv")
df.groupby("region")["amount"].sum().sort_values(ascending=False)
```

The discipline is *vectorization*: explicit Python loops over arrays are slow; replace with NumPy/Pandas operations that push the loop down to C/SIMD.

Linear algebra notation maps directly: `A * B` in MATLAB is matrix multiplication; `A .* B` is element-wise. APL/J operators for inner product (`+.×`) and outer product (`∘.×`) read as the math.

## Prototypical OO

OO **without classes**. Objects are created by **cloning** (or extending) existing objects (prototypes). Methods are looked up on the prototype chain, not a class.

Origin: **Self** (Sun Microsystems, 1987) — the language that inspired JavaScript's design.

**JavaScript**:

```javascript
const animal = {
    name: "anon",
    greet() { return "Hi, I'm " + this.name; }
};
const dog = Object.create(animal);
dog.name = "Rex";
dog.bark = function() { return "Woof"; };

dog.greet();   // "Hi, I'm Rex"   - inherited from animal
dog.bark();    // "Woof"           - own method
```

ES2015 added `class` syntax, but it is *sugar over prototypes*:

```javascript
class Animal {
    greet() { return "Hi, I'm " + this.name; }
}
class Dog extends Animal {
    bark() { return "Woof"; }
}
```

Under the hood, `Dog.prototype.__proto__ === Animal.prototype`.

**Lua** — tables are everything; metatables provide method lookup:

```lua
Animal = {}
function Animal:greet() return "Hi, I'm " .. self.name end

Dog = setmetatable({}, { __index = Animal })
function Dog:bark() return "Woof" end

local rex = setmetatable({ name = "Rex" }, { __index = Dog })
print(rex:greet())   -- Hi, I'm Rex
print(rex:bark())    -- Woof
```

Strengths: extreme flexibility — change behaviour at runtime per-object. Weakness: no static type checks; refactoring is hard at scale.

## Constraint-Based

Declare **variables**, their **domains**, and **constraints**; a solver finds assignments satisfying all constraints. A specialization of declarative + logic.

Languages: MiniZinc, Gecode, Choco, Z3 (SMT solver), Prolog with CLP(FD), CHIP.

**MiniZinc** Sudoku skeleton:

```minizinc
int: N = 9;
array[1..N, 1..N] of var 1..N: x;

constraint forall (i in 1..N) (alldifferent(j in 1..N) (x[i,j]));
constraint forall (j in 1..N) (alldifferent(i in 1..N) (x[i,j]));
constraint forall (br, bc in 0..2)
    (alldifferent(i, j in 1..3) (x[br*3+i, bc*3+j]));

solve satisfy;
output [show(x[i,j]) ++ if j == N then "\n" else " " endif | i, j in 1..N];
```

You wrote no search algorithm.

**Z3** (SMT) in Python:

```python
from z3 import *
x, y = Ints("x y")
solve(x + 2*y == 7, x - y == 1)
# [y = 2, x = 3]
```

**Prolog CLP(FD)** — finite-domain constraint solving over integers:

```prolog
:- use_module(library(clpfd)).

sudoku(Rows) :-
    length(Rows, 9), maplist(same_length(Rows), Rows),
    append(Rows, Vs), Vs ins 1..9,
    maplist(all_distinct, Rows),
    transpose(Rows, Cols), maplist(all_distinct, Cols),
    Rows = [A,B,C,D,E,F,G,H,I],
    blocks(A,B,C), blocks(D,E,F), blocks(G,H,I),
    maplist(label, Rows).
```

Solvers handle propagation (if `x ∈ {1,2,3}` and `x ≠ 1`, propagate to `{2,3}`), backtracking, and search heuristics. You declare; they find.

## Differentiable Programming

Programs are written so that **derivatives** can be computed automatically. The model: any program is a chain of differentiable operations, and the framework constructs the gradient via reverse-mode autodiff.

Frameworks: PyTorch, JAX, TensorFlow, Julia (Zygote, Enzyme).

**PyTorch**:

```python
import torch
x = torch.tensor([2.0], requires_grad=True)
y = x ** 3 + 2 * x
y.backward()
print(x.grad)        # tensor([14.0])  -- dy/dx = 3x^2 + 2 = 14 at x=2
```

**JAX** with functional gradients:

```python
import jax
import jax.numpy as jnp

def loss(w, x, y):
    pred = jnp.dot(x, w)
    return jnp.mean((pred - y) ** 2)

grad_loss = jax.grad(loss)
g = grad_loss(w, x, y)              # gradient wrt w
```

Because `loss` is just a Python function and JAX traces it, you can compose: `jax.jit(jax.grad(jax.vmap(loss)))` — JIT-compile, vectorize, then differentiate.

**Julia Zygote**:

```julia
using Zygote
f(x) = sin(x) * x^2
f'(2.0)         # df/dx at x=2
```

Why a paradigm? Because writing programs *as if they were math expressions* (with autodiff free) changes how you structure code. You stop writing imperative loops over examples and start writing pure tensor expressions.

ML libraries like PyTorch's `nn.Module` blend OO + differentiable: layers are objects with state, but forward passes are pure tensor functions traced for autodiff.

## Quantum Paradigm

Computation in superpositions, gates, and measurement. State is a vector in a complex Hilbert space; operations are unitary matrices; the only way out is measurement, which collapses the state.

Languages: Q# (Microsoft), Qiskit (IBM, embedded in Python), Cirq (Google, embedded in Python), Quipper (embedded in Haskell), Silq.

**Qiskit** — Bell pair (the "Hello, world" of quantum):

```python
from qiskit import QuantumCircuit, transpile
from qiskit_aer import AerSimulator

qc = QuantumCircuit(2, 2)
qc.h(0)               # Hadamard on qubit 0 -> superposition
qc.cx(0, 1)           # CNOT entangles 0 and 1
qc.measure([0, 1], [0, 1])

sim = AerSimulator()
res = sim.run(transpile(qc, sim), shots=1000).result()
print(res.get_counts())   # ~ 50% '00' and 50% '11'
```

**Q#** — typed:

```qsharp
operation BellPair() : (Result, Result) {
    use (q1, q2) = (Qubit(), Qubit());
    H(q1);
    CNOT(q1, q2);
    return (M(q1), M(q2));
}
```

The paradigm forces a different mindset: no copying (no-cloning theorem), no in-place mutation of arbitrary state, only reversible operations until measurement. Algorithms (Shor's, Grover's, QAOA) exploit interference between amplitudes.

It may feel niche today, but it is firmly its own paradigm — gates, qubits, entanglement, measurement, superposition — none of which exist in any classical paradigm.

## Paradigm-to-Language Mapping

The cheat-sheet of which language is which:

| Language | Paradigms |
|----------|-----------|
| **C** | imperative + procedural |
| **C++** | imperative + OO + generic + functional (since C++11) |
| **Java** | OO + generic + functional (since 8) + pattern matching (since 17/21) |
| **C#** | OO + generic + functional + dataflow (LINQ) + async + pattern matching |
| **Go** | procedural + concurrent (goroutines/channels) + a touch of functional; generics since 1.18 |
| **Rust** | systems + concurrent + functional + traits-as-generics + ownership-as-paradigm |
| **Python** | procedural + OO + functional + metaprogramming + duck-typed (everything multi) |
| **JavaScript** | prototypical OO + functional + event-driven + reactive |
| **TypeScript** | JavaScript + structural types + generics |
| **Ruby** | OO (everything-is-an-object extreme) + metaprogramming + functional touches |
| **Haskell** | pure functional + lazy + ADTs + type classes |
| **OCaml** | ML-family functional + objects + modules + first-class polymorphism |
| **Lisp / Scheme / Racket** | functional + symbolic + metaprogramming + multi-paradigm via hygienic macros |
| **Clojure** | functional + dataflow (transducers) + concurrent (STM, atoms, agents) + symbolic (Lisp dialect) |
| **Erlang / Elixir** | functional + actor + concurrent (BEAM VM) |
| **Smalltalk** | pure OO + image-based + reflective |
| **Prolog** | logic |
| **Datalog** | logic restricted to bottom-up evaluation; no function symbols |
| **SQL** | declarative + relational |
| **APL / J / K** | array |
| **Forth / Factor** | concatenative + stack-based |
| **F#** | functional + OO + .NET interop |
| **Scala** | OO + functional + JVM interop |
| **Kotlin** | OO + functional + JVM/Android |
| **Swift** | OO + functional + protocol-oriented + value types |
| **Idris** | dependently-typed pure functional |
| **Coq / Agda / Lean** | dependently-typed proof assistants |
| **Solidity** | OO subset + EVM-targeted |

Pick a language by the paradigms you actually need. Web backend with rich domain modelling? Java, C#, Kotlin, Scala. High-throughput message-driven service? Erlang, Elixir, Go. Compiler? OCaml, Haskell, Rust. Numeric/array? Python+NumPy, Julia, R.

## Map / Filter / Reduce in 10 Languages

The canonical functional trio. Same task: from `[1..10]`, double the evens and sum.

**Python**:

```python
nums = range(1, 11)
result = sum(x * 2 for x in nums if x % 2 == 0)
# or with map/filter
from functools import reduce
result = reduce(lambda a, b: a + b, map(lambda x: x * 2, filter(lambda x: x % 2 == 0, nums)), 0)
```

**JavaScript**:

```javascript
const nums = [1,2,3,4,5,6,7,8,9,10];
const result = nums.filter(x => x % 2 === 0).map(x => x * 2).reduce((a,b) => a + b, 0);
```

**Go** (no built-in map/filter; manual loop is idiomatic):

```go
nums := []int{1,2,3,4,5,6,7,8,9,10}
sum := 0
for _, x := range nums {
    if x%2 == 0 { sum += x * 2 }
}
// Generic helpers with 1.18+ generics:
// func Filter[T any](xs []T, pred func(T) bool) []T { ... }
```

**Rust** (iterators):

```rust
let nums: Vec<i32> = (1..=10).collect();
let result: i32 = nums.iter().filter(|&&x| x % 2 == 0).map(|&x| x * 2).sum();
```

**Haskell** (lazy lists, list comprehension or pipeline):

```haskell
result = sum [x * 2 | x <- [1..10], even x]
-- pipeline form
result = sum . map (*2) . filter even $ [1..10]
```

**OCaml**:

```ocaml
let nums = [1;2;3;4;5;6;7;8;9;10]
let result = nums |> List.filter (fun x -> x mod 2 = 0)
                  |> List.map (fun x -> x * 2)
                  |> List.fold_left (+) 0
```

**Clojure** (transducer or ->>):

```clojure
(reduce + 0 (map #(* % 2) (filter even? (range 1 11))))
;; or with thread-last
(->> (range 1 11) (filter even?) (map #(* % 2)) (reduce + 0))
```

**Java** (streams):

```java
int result = IntStream.rangeClosed(1, 10)
                      .filter(x -> x % 2 == 0)
                      .map(x -> x * 2)
                      .sum();
```

**Ruby**:

```ruby
result = (1..10).select(&:even?).map { |x| x * 2 }.sum
```

**Swift**:

```swift
let result = (1...10).filter { $0 % 2 == 0 }.map { $0 * 2 }.reduce(0, +)
```

All produce `60`. Notice how similar the *shape* is across languages once you adopt the FP idiom.

## Currying Across Languages

**Currying** transforms `f(a, b, c)` into `f(a)(b)(c)` — every multi-arg function becomes a chain of single-arg functions returning functions. Enables **partial application**: fix some arguments, get a new function expecting the rest.

**Haskell** (curried by default):

```haskell
add :: Int -> Int -> Int      -- read: Int -> (Int -> Int)
add x y = x + y
add5 = add 5                  -- partial application is just leaving args off
add5 3                        -- 8
```

**OCaml** (also curried by default):

```ocaml
let add x y = x + y
let add5 = add 5
let _ = add5 3                (* 8 *)
```

**Python** via `functools.partial`:

```python
from functools import partial
def add(x, y): return x + y
add5 = partial(add, 5)
add5(3)                       # 8
```

Or using closures:

```python
def add(x):
    return lambda y: x + y
add(5)(3)                     # 8
```

**JavaScript** via closures:

```javascript
const add = x => y => x + y;
const add5 = add(5);
add5(3);                      // 8

// or curry helper
const curry = f => a => b => f(a, b);
const add2 = curry((x, y) => x + y);
add2(5)(3);
```

**Java** — needs explicit `Function<T,R>`:

```java
Function<Integer, Function<Integer, Integer>> add = x -> y -> x + y;
Function<Integer, Integer> add5 = add.apply(5);
int result = add5.apply(3);   // 8
```

**Go** via closures (no curry sugar):

```go
add := func(x int) func(int) int {
    return func(y int) int { return x + y }
}
add5 := add(5)
fmt.Println(add5(3))          // 8
```

**Rust** via closures returning `impl Fn`:

```rust
fn add(x: i32) -> impl Fn(i32) -> i32 { move |y| x + y }
let add5 = add(5);
println!("{}", add5(3));      // 8
```

**Scala**:

```scala
def add(x: Int)(y: Int) = x + y      // multiple parameter lists
val add5 = add(5) _
add5(3)                              // 8
```

Why bother? Currying makes **point-free** style natural — combine functions without naming intermediate values:

```haskell
sumOfSquares = sum . map (^2)
```

That `(^2)` is `\x -> x ^ 2` — a partially applied operator section.

## Pattern Matching

Deconstruct values by shape and bind variables in one step. Far more expressive than `if/else` ladders.

**Haskell**:

```haskell
data Shape = Circle Double | Rectangle Double Double | Triangle Double Double Double

area :: Shape -> Double
area (Circle r)         = pi * r * r
area (Rectangle w h)    = w * h
area (Triangle a b c)   =
    let s = (a + b + c) / 2
    in sqrt (s * (s-a) * (s-b) * (s-c))
```

**OCaml**:

```ocaml
type shape = Circle of float | Rect of float * float

let area = function
    | Circle r        -> Float.pi *. r *. r
    | Rect (w, h)     -> w *. h
```

**Rust**:

```rust
enum Shape { Circle(f64), Rect(f64, f64) }

fn area(s: &Shape) -> f64 {
    match s {
        Shape::Circle(r) => std::f64::consts::PI * r * r,
        Shape::Rect(w, h) => w * h,
    }
}
```

The Rust compiler **enforces exhaustiveness** — forget a variant, the build fails.

**Scala**:

```scala
sealed trait Shape
case class Circle(r: Double) extends Shape
case class Rect(w: Double, h: Double) extends Shape

def area(s: Shape): Double = s match {
    case Circle(r) => math.Pi * r * r
    case Rect(w, h) => w * h
}
```

**Python 3.10+** (`match` statement):

```python
def area(s):
    match s:
        case ("circle", r):       return 3.14159 * r * r
        case ("rect", w, h):      return w * h
        case ("triangle", a,b,c):
            s = (a + b + c) / 2
            return (s * (s-a) * (s-b) * (s-c)) ** 0.5
        case _:                   raise ValueError(s)
```

**C# 8+/9+** (pattern expressions):

```csharp
public static double Area(Shape s) => s switch {
    Circle (var r)      => Math.PI * r * r,
    Rectangle (var w, var h) => w * h,
    _                   => throw new ArgumentException()
};
```

**Erlang / Elixir** (clause matching is the calling convention):

```elixir
defmodule Shape do
    def area({:circle, r}),         do: :math.pi() * r * r
    def area({:rect,   w, h}),      do: w * h
    def area({:triangle, a, b, c}) do
        s = (a + b + c) / 2
        :math.sqrt(s * (s-a) * (s-b) * (s-c))
    end
end
```

**Java 17+** (sealed types + pattern matching for `switch`):

```java
sealed interface Shape permits Circle, Rect {}
record Circle(double r) implements Shape {}
record Rect(double w, double h) implements Shape {}

double area(Shape s) {
    return switch (s) {
        case Circle c -> Math.PI * c.r() * c.r();
        case Rect r   -> r.w() * r.h();
    };
}
```

**JavaScript** — no native pattern matching (proposal in flight). Libraries:

```javascript
import { match, P } from 'ts-pattern';
const result = match(shape)
    .with({ kind: "circle", r: P.number }, ({ r }) => Math.PI * r * r)
    .with({ kind: "rect", w: P.number, h: P.number }, ({ w, h }) => w * h)
    .exhaustive();
```

**Go** — no algebraic pattern matching; type switch is the closest thing:

```go
switch v := s.(type) {
case Circle: return math.Pi * v.R * v.R
case Rect:   return v.W * v.H
}
```

A pattern-matching proposal exists but has not landed.

## Algebraic Data Types

ADTs combine **product** (record/struct/tuple — `A AND B`) and **sum** (tagged union — `A OR B`) types. Together they express any data shape, exhaustively, with the type checker as guard rail.

**Sum** in Haskell:

```haskell
data Maybe a = Nothing | Just a
data Either l r = Left l | Right r
data Tree a = Leaf | Node (Tree a) a (Tree a)
```

**Sum** in OCaml:

```ocaml
type 'a option = None | Some of 'a
type ('a, 'b) result = Ok of 'a | Error of 'b
```

**Sum** in Rust:

```rust
enum Option<T> { None, Some(T) }
enum Result<T, E> { Ok(T), Err(E) }
enum Tree<T> { Leaf, Node(Box<Tree<T>>, T, Box<Tree<T>>) }
```

**Product** is just a struct/record:

```rust
struct Point { x: f64, y: f64 }
```

```haskell
data Point = Point { x :: Double, y :: Double }
```

**Swift** (enums with associated values are tagged unions):

```swift
enum Result<T, E: Error> {
    case success(T)
    case failure(E)
}
```

**Kotlin** sealed classes:

```kotlin
sealed class Result<out T>
data class Success<T>(val value: T) : Result<T>()
data class Failure(val error: Throwable) : Result<Nothing>()
```

**TypeScript** discriminated unions — tagged via a literal field:

```typescript
type Shape =
    | { kind: "circle"; r: number }
    | { kind: "rect"; w: number; h: number };

function area(s: Shape): number {
    switch (s.kind) {
        case "circle": return Math.PI * s.r ** 2;
        case "rect":   return s.w * s.h;
    }
}
```

The compiler narrows `s` inside each branch.

**Python** with `typing.Union` + `dataclasses`:

```python
from dataclasses import dataclass
from typing import Union

@dataclass
class Circle: r: float
@dataclass
class Rect:   w: float; h: float
Shape = Union[Circle, Rect]

def area(s: Shape) -> float:
    match s:
        case Circle(r): return 3.14159 * r * r
        case Rect(w, h): return w * h
```

**Slogan**: *make illegal states unrepresentable.* Instead of `class User { String email; boolean confirmed; ... }`, model the states as a sum: `data User = Pending Email | Confirmed Email Token`. The compiler catches "confirmed user without token" before runtime.

## Type Classes / Traits / Interfaces

Different ways languages express "this type supports these operations".

**Haskell type classes** (the original):

```haskell
class Eq a where
    (==) :: a -> a -> Bool

class Eq a => Ord a where
    compare :: a -> a -> Ordering

instance Eq Int where x == y = primEqInt x y
instance Ord Int where compare x y = ...
```

A type class is a **dictionary** of operations; instances provide the dictionary.

**Rust traits** (single-dispatch + dyn for trait objects):

```rust
trait Animal {
    fn name(&self) -> &str;
    fn sound(&self) -> String { format!("{} says generic noise", self.name()) }
}

struct Dog;
impl Animal for Dog {
    fn name(&self) -> &str { "Rex" }
    fn sound(&self) -> String { "Woof".into() }
}

fn describe(a: &dyn Animal) { println!("{}: {}", a.name(), a.sound()); }
```

**Scala** implicits / `given`:

```scala
trait Show[A]:
    def show(a: A): String

given Show[Int] with
    def show(x: Int) = x.toString

def display[A](a: A)(using s: Show[A]) = println(s.show(a))
```

**Java / C# interfaces** — narrower: only nominal subtyping, single inheritance of state, no operator overloading:

```java
interface Comparable<T> { int compareTo(T other); }
class Person implements Comparable<Person> { ... }
```

**Go interfaces** are **structural** — any type with the right methods *is* the interface, no `implements` declaration:

```go
type Stringer interface { String() string }

type Point struct{ x, y int }
func (p Point) String() string { return fmt.Sprintf("(%d,%d)", p.x, p.y) }

// Point satisfies Stringer automatically; no declaration needed
```

**Python protocols** (PEP 544, structural):

```python
from typing import Protocol

class Drawable(Protocol):
    def draw(self) -> None: ...

class Circle:
    def draw(self) -> None: print("circle")

def render(d: Drawable) -> None: d.draw()

render(Circle())   # type-checks; Circle satisfies Drawable structurally
```

**Swift protocols** with associated types and protocol extensions; **Kotlin** interfaces with default methods.

When choosing: type classes/traits give *ad-hoc polymorphism* (extend behaviour for types you don't own); Java-style interfaces are *nominal* (you must declare `implements`); Go/Python structural are between (no declaration, but limited to method-shape).

## Monads

A monad is an **interface for sequencing computations** that share a common context — a "result-with-effects". The textbook definition:

```haskell
class Monad m where
    return :: a -> m a                 -- wrap a value
    (>>=)  :: m a -> (a -> m b) -> m b -- sequence
```

Forget the burritos. The common useful instances:

**Maybe / Option** — for nullable / "computation may fail with no error":

```haskell
safeDiv :: Int -> Int -> Maybe Int
safeDiv _ 0 = Nothing
safeDiv x y = Just (x `div` y)

calc :: Int -> Int -> Int -> Maybe Int
calc a b c = do
    x <- safeDiv a b
    y <- safeDiv x c
    return (y + 1)
```

If any step yields `Nothing`, the whole expression short-circuits.

**Either / Result** — same idea, but carry error info:

```haskell
parseUser :: String -> Either String User
register :: User -> Either String UserId
notify :: UserId -> Either String ()

flow :: String -> Either String ()
flow input = do
    u   <- parseUser input
    uid <- register u
    notify uid
```

**Rust `?` operator** is Result-monad sugar:

```rust
fn flow(input: &str) -> Result<(), MyErr> {
    let u   = parse_user(input)?;
    let uid = register(&u)?;
    notify(uid)?;
    Ok(())
}
```

The `?` desugars to `match expr { Ok(v) => v, Err(e) => return Err(e.into()) }` — exactly the bind for `Result`.

**List** — for non-determinism:

```haskell
pairs = do
    x <- [1, 2, 3]
    y <- ['a', 'b']
    return (x, y)
-- [(1,'a'),(1,'b'),(2,'a'),(2,'b'),(3,'a'),(3,'b')]
```

**State** monad — thread state through pure functions:

```haskell
import Control.Monad.State
counter :: State Int Int
counter = do
    n <- get
    put (n + 1)
    return n

runState (replicateM 3 counter) 0
-- ([0,1,2], 3)
```

**IO** — describe side effects in pure Haskell:

```haskell
main :: IO ()
main = do
    name <- getLine
    putStrLn ("Hello, " ++ name)
```

The most useful gloss: **monads are interfaces for sequencing**. Each instance defines what "and then" means for its context — short-circuit on Nothing, accumulate state, log to a writer, dispatch to async runtime.

Other useful monads in industry: `Reader` (config/dependency injection), `Writer` (logging), `STM` (transactions), `Cont` (continuations), `Parser` (parser combinators).

## Actor Model

Carl Hewitt, 1973. Each **actor**:

1. Has a **mailbox** (FIFO queue of messages).
2. Has private **state**.
3. Processes one message at a time from its mailbox, deciding:
   - what messages to **send** to other actors,
   - what its **next state** should be,
   - what new actors to **spawn**.

There is no shared memory between actors. Concurrency is achieved by spawning more actors, not threads-with-locks.

**Erlang**:

```erlang
-module(bank).
-export([start/1, loop/1]).

start(Initial) -> spawn(?MODULE, loop, [Initial]).

loop(Balance) ->
    receive
        {deposit, Amount}   -> loop(Balance + Amount);
        {withdraw, Amount, From} when Amount =< Balance ->
            From ! {ok, Amount},
            loop(Balance - Amount);
        {withdraw, _, From} ->
            From ! {error, insufficient},
            loop(Balance);
        {balance, From}     -> From ! Balance, loop(Balance)
    end.
```

**Elixir** version:

```elixir
defmodule Bank do
    def start(initial), do: spawn(fn -> loop(initial) end)

    defp loop(balance) do
        receive do
            {:deposit, amount} -> loop(balance + amount)
            {:withdraw, amount, from} when amount <= balance ->
                send(from, {:ok, amount})
                loop(balance - amount)
            {:withdraw, _, from} ->
                send(from, {:error, :insufficient})
                loop(balance)
            {:balance, from} -> send(from, balance); loop(balance)
        end
    end
end
```

**Akka Typed** (Scala):

```scala
sealed trait BankMsg
case class Deposit(n: Int) extends BankMsg
case class Withdraw(n: Int, replyTo: ActorRef[Result]) extends BankMsg

def bank(balance: Int): Behavior[BankMsg] = Behaviors.receiveMessage {
    case Deposit(n)                => bank(balance + n)
    case Withdraw(n, rt) if n <= balance =>
        rt ! Result.Ok(n); bank(balance - n)
    case Withdraw(_, rt)           => rt ! Result.Insufficient; Behaviors.same
}
```

**Supervisor trees & let-it-crash**: an actor that fails is **restarted** by its supervisor, often to a known clean state. Errors aren't caught inline — they crash the actor and are handled at the supervisor level. This produces self-healing systems.

```text
                supervisor
                 / | \
                /  |  \
               A   B   C       <- workers
                       /\
                      /  \
                     D    E    <- C's workers
```

If `C` dies, supervisor restarts `C`, which respawns `D` and `E`. The OTP framework codifies this.

Microsoft **Orleans** does the same on .NET with "virtual actors" (grains) — actors that exist conceptually whether or not they are loaded.

## Reactive Streams

The **Reactive Manifesto** (2014) names four properties: responsive, resilient, elastic, message-driven. Reactive Streams is a JVM/JS spec that codifies a non-blocking, **back-pressure-aware** stream protocol.

The contract:

```text
Publisher --(onSubscribe)--> Subscriber
Subscriber --(request(n))--> Publisher       <- back-pressure
Publisher --(onNext, onNext, ...)--> Subscriber
Publisher --(onComplete | onError)--> Subscriber
```

The subscriber controls flow with `request(n)`; the publisher must not emit faster than requested. This avoids buffer blowup.

**Project Reactor** (Java):

```java
Flux<Integer> nums = Flux.range(1, 100)
    .filter(n -> n % 2 == 0)
    .map(n -> n * n)
    .onBackpressureBuffer(16);

nums.subscribe(System.out::println);
```

**RxJava** / **RxJS** / **RxSwift** — same idea, slightly different operator naming.

**Akka Streams** — graph-based:

```scala
Source(1 to 100)
    .filter(_ % 2 == 0)
    .map(n => n * n)
    .runWith(Sink.foreach(println))
```

**Cold vs hot streams**:

- **Cold** — produces values only when subscribed; each subscriber gets its own sequence (`Flux.range`, `Observable.interval`).
- **Hot** — produces regardless of subscribers; late subscribers miss earlier values (UI events, message buses).

Convert cold → hot with `share()`, `publish()`, `replay()`.

## Message Passing vs Shared Memory

Two ways concurrent code coordinates:

**Shared memory + locks** — multiple threads access the same memory, synchronization via mutexes, semaphores, atomics. Java/C++/POSIX threads. Failure modes: races, deadlocks, livelocks, priority inversion.

```java
class Counter {
    private int n = 0;
    synchronized void inc() { n++; }
    synchronized int get() { return n; }
}
```

**Message passing** — threads/processes communicate via messages over channels. No direct memory sharing. Hoare's **CSP** (Communicating Sequential Processes), Erlang's actors, Go's channels.

Go (CSP-style):

```go
ch := make(chan int)
go producer(ch)
for v := range ch { fmt.Println(v) }
```

Erlang (actor-style):

```erlang
Pid = spawn(fun() -> receive {msg, X} -> io:format("got ~p~n", [X]) end end),
Pid ! {msg, 42}.
```

**Rust** sits in between: shared memory is allowed, but the borrow checker statically prevents simultaneous mutable access. `Arc<Mutex<T>>` for shared mutation; `mpsc::channel` for message passing.

```rust
use std::sync::mpsc;
let (tx, rx) = mpsc::channel();
std::thread::spawn(move || tx.send(42).unwrap());
println!("{}", rx.recv().unwrap());
```

Trade-offs:

| | Shared memory + locks | Message passing |
|--|--|--|
| Performance | Fast (no copy) | Cost of message construction |
| Correctness | Hard — races invisible | Easier — explicit communication |
| Distribution | Same machine only | Naturally distributed (Erlang) |
| Reasoning | Local invariants weak | Each actor's state is private |

Rule of thumb: prefer message passing across module/process boundaries; use shared memory inside a tight loop where you've measured contention.

## Continuations / CPS

A **continuation** is "the rest of the program at this point" reified as a function. Continuation-Passing Style (CPS) means *every* function takes a callback (the continuation) and never returns directly.

Direct style:

```scheme
(define (square x) (* x x))
(define (f x) (+ (square x) 1))
```

CPS:

```scheme
(define (square-cps x k) (k (* x x)))
(define (f-cps x k)
    (square-cps x (lambda (sq) (k (+ sq 1)))))
(f-cps 3 display)   ; 10
```

**Scheme's `call/cc`** captures the current continuation as a first-class value:

```scheme
(+ 1 (call/cc (lambda (k)
    (k 10)              ; jump out with value 10
    99)))               ; never reached
;; result: 11
```

Used to implement: exceptions, generators, coroutines, backtracking, web continuations.

**Async/await** in JavaScript/Python/C#/Rust is implicitly **CPS-transformed** by the compiler. Each `await` is a call to "schedule the rest of this function as a continuation":

```javascript
async function f() {
    const x = await fetch("/a");   // await splits the function
    const y = await fetch("/b");
    return x + y;
}
```

Mentally:

```javascript
function f() {
    return fetch("/a").then(x =>
        fetch("/b").then(y => x + y));
}
```

The compiler does this rewrite for you. The implications:

- Stack traces are imperfect — the "stack" is a chain of continuations.
- Locals captured by continuations stay alive longer.
- Cancellation is hard — every continuation must check.

Generators (`yield`) are also a CPS-style mechanism: each `yield` saves the continuation; `next()` resumes it.

## Lazy vs Strict Evaluation

**Strict** (eager): arguments are evaluated **before** the call. Most languages — C, Java, Python, OCaml, Rust, Go, JavaScript.

**Lazy**: arguments are evaluated **only if and when used**. Haskell. Scala has opt-in `lazy val`. Python's generators give per-iteration laziness.

Strict (Python):

```python
def f(x, y): return x      # both x and y evaluated, even though y unused

f(1, expensive())          # expensive() runs
```

Lazy (Haskell):

```haskell
f x y = x

f 1 (expensive ())         -- expensive () NEVER runs; thunk discarded
```

Haskell evaluation:

- Each expression is a **thunk** — a suspended computation.
- Forcing a thunk evaluates it to **WHNF** (weak head normal form) — the outermost constructor.
- Thunks are memoized — forced once, the value is cached.

This enables **infinite data structures**:

```haskell
nats = [0..]                 -- infinite list of naturals
take 5 nats                  -- [0,1,2,3,4]
zip [1..] ["a", "b", "c"]    -- [(1,"a"),(2,"b"),(3,"c")]
```

You can fold over a "list" that would never finish if forced.

The **gotcha**: space leaks. Lazy thunks accumulate, holding references to large structures. Classic example:

```haskell
sum = foldl (+) 0 [1..10^6]   -- builds a thunk: 0+1+2+3+... -> stack overflow
sum = foldl' (+) 0 [1..10^6]  -- strict fold; uses constant memory
```

`foldl'` is strict; force the accumulator at each step.

**Python generators** as opt-in laziness:

```python
def squares():
    n = 0
    while True:
        yield n * n
        n += 1

import itertools
list(itertools.islice(squares(), 5))   # [0, 1, 4, 9, 16]
```

**Scala** `lazy val`:

```scala
lazy val expensive = {
    println("computed")
    42
}
println(expensive)   // "computed" then 42
println(expensive)   // 42 (cached)
```

**Streams** in many languages (Java Streams, Rust iterators) are *internally* lazy — operators chain without computing until a terminal operation runs.

## Memory Management Paradigms

How a language tracks the lifetime of allocated memory.

**Manual** — C/C++ legacy code: `malloc`/`free`, `new`/`delete`. Maximum control, maximum bug surface (use-after-free, double-free, leak).

```c
char *buf = malloc(1024);
if (!buf) return -1;
/* use buf */
free(buf);
```

**RAII** — Resource Acquisition Is Initialization. C++ destructors run when objects go out of scope; Rust's `Drop` trait does the same.

```cpp
{
    std::vector<int> v(1000);   // allocated
    /* ... */
}                                // destructor frees automatically
```

```rust
{
    let v: Vec<i32> = vec![0; 1000];
    // ...
}    // Drop runs here
```

**Garbage collection** — runtime tracks reachable objects, reclaims unreachable ones. Many flavours:

- **Mark-and-sweep** — Lisp, early Java.
- **Generational** — modern Java (G1, ZGC), .NET, V8.
- **Tracing** — Go's concurrent collector.
- **Incremental / concurrent** — minimize stop-the-world pauses.

```java
// Java: just allocate; the GC runs in the background
List<Integer> xs = new ArrayList<>();
xs.add(1);
// no free
```

**Reference counting** — each object has a count of references; when it hits 0, free. Swift (ARC), Objective-C (ARC), Python (CPython), some C++ smart pointers (`shared_ptr`). Cycles need help (weak refs, cycle collector).

```swift
class Node {
    weak var parent: Node?    // weak to break cycles
    var children: [Node] = [] // strong
}
```

**Ownership + borrow checker** — Rust. Each value has a single owner; borrows are tracked at compile time. No GC, no manual `free`, no use-after-free.

```rust
fn main() {
    let s = String::from("hi");   // s owns the String
    let r = &s;                    // borrow
    println!("{} {}", s, r);
}                                  // s dropped here; r already gone
```

Move semantics:

```rust
let s = String::from("hi");
let t = s;                // moved; s no longer usable
// println!("{}", s);     // compile error
```

**Region-based** — ML's region inference, Cyclone's regions, some research languages. Group allocations into regions, free the whole region at once.

## Type System Paradigms

Independent axes:

**Static vs Dynamic**

- **Static**: types checked at compile time. Java, C, C++, Rust, Haskell, OCaml, TypeScript, Go.
- **Dynamic**: types checked at run time. Python, JavaScript (without TS), Ruby, Lisp, Smalltalk.

**Strong vs Weak**

The terms are abused. The defensible distinction:

- **Strong**: implicit conversions are minimized; misuse usually errors. Python, Haskell, Rust.
- **Weak**: implicit conversions and reinterpretation are permitted. C (`(int)ptr`), JavaScript (`"5" - 1 === 4`), Perl.

**Structural vs Nominal**

- **Nominal**: types match by name. Java's `class Cat` is not a `class Dog` even with identical fields.
- **Structural**: types match by shape. Go's interfaces, TypeScript object types, OCaml's row polymorphism.

```typescript
type Point = { x: number; y: number };
function distance(p: Point) { return Math.hypot(p.x, p.y); }
distance({ x: 3, y: 4, z: 0 });   // OK in TS — extra fields fine
```

```go
type Stringer interface { String() string }
type Point struct { x, y int }
func (p Point) String() string { return "..." }
// Point structurally satisfies Stringer
```

**Gradual** — mix of static and dynamic. Python type hints + mypy/pyright; TypeScript (over JavaScript); Sorbet (Ruby); Hack (PHP); Typed Racket.

```python
def add(x: int, y: int) -> int:
    return x + y                    # checked by mypy; runtime ignores
```

**Dependent types** — types depend on values. The most expressive type systems.

```idris
-- Idris: a Vec carries its length in the type
data Vec : Nat -> Type -> Type where
    Nil  : Vec Z a
    (::) : a -> Vec n a -> Vec (S n) a

append : Vec n a -> Vec m a -> Vec (n + m) a
append Nil       ys = ys
append (x :: xs) ys = x :: append xs ys
```

The type *guarantees* the result vector has length `n + m`. Indexing past the end is a type error.

Coq, Agda, Lean, F* — proof assistants where you write theorems and proofs in the same language.

## Concurrency Paradigms

Overview of the styles in production:

| Style | Languages | Notes |
|--|--|--|
| Threads + locks | Java, C++, Pthreads | OS threads; preemptive; race-prone |
| Goroutines + channels | Go | M:N scheduler; CSP-influenced |
| Async/await | JavaScript, Python, Rust, C# | Cooperative; futures/promises; CPS-transformed |
| Actors | Erlang, Elixir, Akka, Orleans | Mailboxes; isolated state |
| STM | Clojure, Haskell | Atomic blocks compose |
| Coroutines | Lua, Kotlin, Python | Cooperative; explicit yield |
| Fibers | Ruby Fiber, JVM Project Loom | Lightweight cooperative threads |
| Effects systems | Eff, Koka | Effects in the type system |

**Threads + locks** (Java):

```java
ReentrantLock lock = new ReentrantLock();
lock.lock();
try { /* critical section */ } finally { lock.unlock(); }
```

**Goroutines + channels** (Go):

```go
done := make(chan struct{})
go func() { defer close(done); doWork() }()
<-done
```

**Async/await** (Rust):

```rust
async fn fetch(u: &str) -> String { /* ... */ }

#[tokio::main]
async fn main() {
    let (a, b) = tokio::join!(fetch("/a"), fetch("/b"));
    println!("{} {}", a, b);
}
```

**STM** (Clojure):

```clojure
(def account (ref 100))

(dosync
    (alter account - 30)
    (alter account + 10))
;; both alters atomic; retried on conflict
```

**Coroutines** (Kotlin):

```kotlin
suspend fun fetchUser(): User { /* ... */ }

runBlocking {
    val user = async { fetchUser() }
    println(user.await())
}
```

## Paradigm Choice for Common Tasks

Quick guide. The "best" paradigm is whichever your team can deliver maintainably; below are reasonable defaults.

| Task | Paradigm fit | Example stacks |
|--|--|--|
| Web backend, rich domain | OO + functional touches | Java/Spring, Kotlin/Ktor, C#/ASP.NET, Scala/Play, Python/Django |
| Data pipeline | Functional / dataflow | Python pandas, Spark, Apache Beam, Flink |
| High-concurrency network service | Actor or async | Erlang/OTP, Elixir/Phoenix, Go, Rust+Tokio, Akka |
| Systems programming | Procedural + RAII | C, C++ (modern), Rust |
| Type-driven business logic | ADTs + pattern matching | Haskell, Rust, F#, Scala |
| UI | Reactive | React (web), SwiftUI, Jetpack Compose, Elm |
| Game engine | Data-oriented + ECS | Unity (C#), Unreal (C++), Bevy (Rust) |
| Compiler / interpreter | Functional with ADTs | OCaml, Haskell, Rust |
| ML model training | Differentiable | PyTorch, JAX, TensorFlow |
| Theorem proving | Dependently typed | Lean, Coq, Agda, F* |
| Numerical computing | Array | NumPy, Julia, MATLAB, Fortran |
| Configuration / IaC | Declarative | Terraform, Kubernetes, Nix |
| Static analysis | Logic / Datalog | Soufflé, Datalog, CodeQL |

Start with the dominant paradigm for the domain; mix in others where they reduce code or risk. A web backend in Scala can be 80% OO with a functional core for the validation/transformation layer.

## Anti-Patterns

Paradigm mismatches that cost teams:

**Java-style classes everywhere in a functional codebase**:

```scala
// BAD: an OO Java translation in Scala
class UserValidator {
    def validate(u: User): Boolean = ???
}
val v = new UserValidator()
v.validate(user)

// GOOD: a function
val validate: User => Either[Error, User] = ???
```

**Lisp-style prefix everything in operator-rich languages**:

```lisp
(+ (* 2 3) 4)   ; idiomatic in Lisp
```

```javascript
add(multiply(2, 3), 4)   // BAD in JavaScript
2 * 3 + 4                // GOOD
```

**OO mocking where pure functions would do**:

```java
// BAD: mock the dependency
when(emailGen.subject(any())).thenReturn("Welcome!");

// GOOD: pass the function
fun welcomeEmail(genSubject: (User) -> String, u: User) = ...
welcomeEmail(_ -> "Welcome!", testUser)   // no mocking framework
```

**Functional puritanism that ignores reality**:

```haskell
-- BAD: fight IO endlessly to keep `main` "pure"
-- GOOD: keep core pure, edges in IO; that IS functional design
main = do
    cfg <- readConfig
    runApp (pureLogic cfg)
```

**Concurrent code with shared mutable state**:

```python
# BAD: globals shared across threads, no lock
counter = 0
def worker():
    global counter
    for _ in range(1000): counter += 1   # races

# GOOD: atomic / lock / message passing
from queue import Queue
q = Queue()
```

**"I'll just `gc.collect()`" in a scripting language**:

```python
# BAD: papering over a leak
gc.collect()
# GOOD: find the reference cycle (objgraph, tracemalloc) and break it
```

## Common Errors

Paradigm-induced bug categories. The first step in fixing is recognizing the family.

| Bug | Cause | Symptom |
|--|--|--|
| Race condition | Shared mutable state, no sync | Sporadic wrong values, "works in dev" |
| Memory leak (lazy) | Thunks holding refs | Heap grows; GC pauses lengthen |
| Deadlock | Locks acquired in different orders | Threads stuck forever |
| Stack overflow | Non-TCO recursion on big input | Crash; deep call stack |
| Heap fragmentation | Manual mgmt, mixed lifetimes | Allocator slows; OOM with free space |
| Capability leak | Object exposes mutator | Caller mutates "private" state |
| Unsafe cast | `Object`, `void*`, `any` | ClassCastException at runtime |

Concrete fixes:

```java
// Race
private final AtomicInteger n = new AtomicInteger();
n.incrementAndGet();

// Deadlock: always acquire in a global order
synchronized(lock1) { synchronized(lock2) { ... } }   // forbid lock2-then-lock1

// Stack overflow: rewrite as fold or trampoline
int sum = nums.stream().mapToInt(Integer::intValue).sum();
```

Rust eliminates several of these statically — race conditions through `Send`/`Sync`, use-after-free through ownership, double-free through move semantics.

## Common Gotchas

Eight broken→fixed pairs translating one paradigm into another.

**1) Go-style error returns shoehorned into a try/catch language**

```python
# BAD: imitating Go in Python
def divide(a, b):
    if b == 0: return None, "division by zero"
    return a / b, None

result, err = divide(1, 0)
if err: ...
```

```python
# GOOD: idiomatic Python
def divide(a, b):
    if b == 0: raise ValueError("division by zero")
    return a / b
```

The reverse is also a pitfall — wrapping every Go function in a `panic`/`recover` because you miss exceptions.

**2) Shared-memory synchronization in actor systems**

```elixir
# BAD: trying to use a Mutex alternative
defmodule BankBad do
    def init, do: :ets.new(:bank, [:public])
    def transfer(from, to, amt) do
        :ets.update_counter(:bank, from, -amt)   # race
        :ets.update_counter(:bank, to, +amt)
    end
end
```

```elixir
# GOOD: an actor owns the data
defmodule Bank do
    use GenServer
    def init(_), do: {:ok, %{}}
    def handle_call({:transfer, from, to, amt}, _, state) do
        # atomic by construction — single actor processes one msg at a time
        ...
    end
end
```

**3) For-loops in stream-processing systems**

```scala
// BAD: imperative loop in a streaming pipeline
val results = scala.collection.mutable.ListBuffer.empty[Int]
source.foreach(x => results += x * 2)
```

```scala
// GOOD
source.map(_ * 2).toList
```

**4) Inheritance hierarchies in functional codebases**

```scala
// BAD: deep class tree
abstract class Shape { def area: Double }
class Circle(r: Double) extends Shape { def area = math.Pi * r * r }
class FilledCircle(r: Double, c: Color) extends Circle(r) { ... }
```

```scala
// GOOD: ADT + functions
sealed trait Shape
case class Circle(r: Double) extends Shape
case class Filled(s: Shape, c: Color) extends Shape

def area(s: Shape): Double = s match {
    case Circle(r) => math.Pi * r * r
    case Filled(s, _) => area(s)
}
```

**5) String concatenation in pure-functional code**

```haskell
-- BAD: O(n^2) on long strings
slow xs = foldl (++) "" xs
```

```haskell
-- GOOD: use Builder / ByteString.Builder / Text.Builder
import qualified Data.Text.Lazy.Builder as B
fast xs = B.toLazyText (mconcat (map B.fromText xs))
```

```java
// BAD
String s = "";
for (var x : xs) s += x;       // O(n^2)
// GOOD
var sb = new StringBuilder();
for (var x : xs) sb.append(x);
String s = sb.toString();
```

**6) Mutable collections passed across actor boundaries**

```scala
// BAD: send a mutable buffer
val buf = mutable.ListBuffer(1, 2, 3)
otherActor ! Update(buf)        // both actors now alias buf
```

```scala
// GOOD: send immutable
otherActor ! Update(buf.toList)
```

**7) eval-style metaprogramming in static languages**

```javascript
// BAD: eval an expression at runtime
const fn = new Function("x", "return x * 2;");
fn(5);
```

```typescript
// GOOD: parameterize properly
const fn = (x: number) => x * 2;
```

If you really need codegen in a static language, prefer build-time generation (proc-macros, codegen scripts) over runtime `eval`.

**8) try/catch as control flow vs Result types**

```java
// BAD: exceptions for normal control flow
try { return Integer.parseInt(s); }
catch (NumberFormatException e) { return -1; }
```

```java
// GOOD (Java 8+ Optional)
public static Optional<Integer> tryParse(String s) {
    try { return Optional.of(Integer.parseInt(s)); }
    catch (NumberFormatException e) { return Optional.empty(); }
}
```

```rust
// In Rust, the type signature forces the choice
fn try_parse(s: &str) -> Option<i32> { s.parse().ok() }
```

## Evolution

Languages converge. The trend is **multi-paradigm by absorption** — the loudest "OO" language adds lambdas; the loudest "functional" language adds records and effect tracking.

**Java**:

- 8 (2014): lambdas, streams, `Optional`, default methods.
- 14 (2020): records.
- 16 (2021): pattern matching for `instanceof`.
- 17 (2021): sealed classes, switch expressions, pattern matching for switch (preview→).
- 21 (2023): record patterns, virtual threads (Project Loom), pattern matching for switch finalized.

**C#**:

- 3.0: LINQ, lambdas.
- 5.0: `async`/`await`.
- 7.x: tuples, pattern matching expressions, ref returns.
- 9.0: records, pattern improvements.
- 12: collection expressions, primary constructors.

**C++**:

- 11: `auto`, lambdas, move semantics, range-for.
- 14/17: structured bindings, `std::optional`, `std::variant`.
- 20: concepts (constraints on templates), ranges, coroutines, modules.
- 23: more ranges, `std::expected`.

**Python**:

- 3.5: `async`/`await`.
- 3.6: f-strings, type hints maturing.
- 3.7: dataclasses.
- 3.10: structural pattern matching (`match` statement).
- 3.11+: `Self`, `LiteralString`, `TypedDict` improvements, exception groups.

**JavaScript**:

- ES2015 (ES6): `class`, arrow functions, modules, `let/const`, destructuring.
- ES2017: `async`/`await`.
- ES2020: optional chaining `?.`, nullish coalescing `??`, BigInt.
- ES2022: top-level await, `at()`, public/private class fields.
- Stage proposals: pattern matching, pipe operator, decorators standardized.

**Go**:

- 1.18 (2022): generics.
- 1.21+: built-in `min`/`max`/`clear`, `for range int`, structured logging.

**TypeScript**: types-as-paradigm — discriminated unions, conditional types, mapped types, template literal types — push the boundary of what a structural type system can express.

**Rust**: established 2010s, language designed multi-paradigm from the start; ongoing additions are mostly refinements (async traits, GATs, const generics).

The trend is bidirectional: OO languages add functional features; functional languages add records and data classes. Pure paradigm wars are dated. **Polyglot teams pick the right paradigm per layer.**

## Idioms

Closing wisdom. Each phrase below carries enough collective wisdom to guide a design decision.

- **"Use the paradigm appropriate to the problem."** A purely functional implementation of a stateful UI is a slog; a procedural implementation of a tree transformation is verbose. Match the shape of the code to the shape of the data.
- **"You can do OO in C and FP in JavaScript."** Paradigms are *disciplines*, not language features. C with structs of function pointers is OO; JavaScript with `Object.freeze` and pure functions is FP.
- **"The paradigm of your team matters more than the paradigm of your language."** A consistent OO codebase outperforms a "clever" mix where every contributor brings their favourite paradigm.
- **"Reaching for inheritance is usually a code smell."** Composition is more flexible: pass collaborators in, don't extend.
- **"Favor composition over inheritance."** (GoF.)
- **"Make illegal states unrepresentable."** Use sum types; if a status can be `Pending | Active | Cancelled`, model the three cases — don't shove three booleans into a class and hope.
- **"Parse, don't validate."** (Alexis King.) Convert untrusted input into a domain type *once*, then carry the typed value forward. Don't sprinkle validation across the codebase.
- **"Premature optimization is the root of all evil."** (Knuth.)
- **"Avoid clever code; choose clear code."** Cleverness that requires paradigm gymnastics costs more than its terseness saves.
- **"Push effects to the edge."** Pure core, impure shell. Maximize the surface where reasoning is purely functional; isolate the I/O layer.
- **"Don't communicate by sharing memory; share memory by communicating."** (Go proverb.) Especially important when modeling concurrency.
- **"Concurrency is not parallelism."** (Rob Pike.) Concurrency structures programs; parallelism is a runtime property.
- **"Errors are values."** (Go.) Treating errors as data, not control flow, makes them composable.
- **"Code is data; data is code."** (Lisp.) Embrace this and metaprogramming becomes natural.

## See Also

- polyglot — the cross-language reference for syntax mappings
- c — imperative + procedural baseline
- cpp — multi-paradigm: imperative, OO, generic, functional
- java — class-based OO + generics + lambdas + pattern matching
- python — multi-paradigm scripting and dynamic typing
- javascript — prototypes + functional + event-driven
- typescript — JavaScript with structural types
- go — procedural + concurrent (CSP-style)
- rust — systems + ownership + traits + functional
- ruby — OO extreme + metaprogramming

## References

- Sebesta, *Concepts of Programming Languages* (12th ed.).
- Pierce, *Types and Programming Languages*.
- Abelson & Sussman, *Structure and Interpretation of Computer Programs* (SICP).
- Scott, *Programming Language Pragmatics*.
- Nystrom, *Crafting Interpreters*.
- Friedman & Byrd, *The Reasoned Schemer*.
- O'Sullivan, Stewart & Goerzen, *Real World Haskell*.
- Armstrong, *Programming Erlang*.
- Hewitt, Bishop & Steiger, "A Universal Modular ACTOR Formalism for Artificial Intelligence" (1973).
- Hoare, *Communicating Sequential Processes* (CSP, 1978/1985).
- Dijkstra, "Goto Considered Harmful" (CACM, 1968).
- Kay, "The Early History of Smalltalk" (HOPL II, 1993).
- Wadler, "Monads for Functional Programming" (1995).
- Reactive Manifesto v2.0, https://www.reactivemanifesto.org/.
- Reactive Streams Specification, https://www.reactive-streams.org/.
- The Rust Programming Language Book, https://doc.rust-lang.org/book/.
- The Erlang/OTP Documentation, https://www.erlang.org/docs.
- Haskell 2010 Language Report.
- ANSI Common Lisp standard (X3.226-1994).
