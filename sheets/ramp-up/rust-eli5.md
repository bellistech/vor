# Rust — ELI5

> Rust is C with a strict librarian standing at the door. Every book you borrow must be checked out properly, returned on time, and never handed to a friend while you're still reading it. The librarian reads every page of your code before the library opens. If anything is wrong, the doors stay shut.

## Prerequisites

You will get the most out of this sheet if you have already typed a few commands at a terminal and seen a programming language compile something. If you have read `ramp-up/linux-kernel-eli5` and `ramp-up/git-eli5`, you have plenty. If you have read `languages/c` you will recognize a lot of vocabulary fast — Rust borrows almost every dirty trick from C and then puts a bouncer in front of each one.

You do **not** need to know what "ownership" or "borrowing" or "lifetimes" mean. Those are the things this sheet teaches. You do not need to have used a "compiler" before — we will explain what one is. You do not need to know what "memory" really is past "the place where running programs keep their stuff." That is enough.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word is in there with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you.

## What Even Is Rust

### A language with a compiler that says "no"

Rust is a programming language. A programming language is just a set of words and rules you use to write down instructions for a computer. You write the instructions in a text file. Then a special program called a **compiler** reads that file and turns it into a runnable program your computer can actually execute.

Most languages run their programs first and crash later if you made a mistake. Rust runs the compiler first and refuses to make a runnable program if the compiler doesn't like what it sees. The compiler is grumpy. The compiler is famous for being grumpy. It will scold you. It will reject your work. It will print three pages of red text explaining exactly why it refuses to build your program.

This sounds bad. It is actually amazing. Because if the compiler accepts your program, that program is **almost certainly free of a whole pile of nasty bugs** that haunt other languages. The Rust community calls this "fearless concurrency" and "memory safety without garbage collection," but at the level of this sheet, the right phrase is: **the compiler said yes, so the program is probably good.**

### The librarian picture

Imagine a giant library. Every book in the library is a piece of data your program is using. A web browser has millions of books. A calculator has a handful. The kernel from the previous sheet manages whose desk gets to hold which book; Rust manages whose **hand** is currently holding which book.

In this library, there is a strict librarian sitting at the front desk. The librarian has three rules:

1. **Every book has exactly one owner.** Only one person can own a book. If you give the book to your friend, your friend now owns it and you don't anymore. You can't read it. You can't even point at it. It's gone.
2. **You can lend a book out, but only one of two ways.** Either many people can read it at once (everybody gets to look but nobody writes), **or** exactly one person can write in it (and nobody else even gets to look). Never both.
3. **The library closes the second a book is dropped.** When the owner of a book leaves the library, that book is destroyed, and any loan paperwork that points to it is automatically torn up.

The library only opens — that is, your program only compiles — if every loan slip is provably legal under those three rules. If even one loan would let two people write in the same book at the same time, or let somebody read a book that is about to be destroyed, the librarian refuses to open the doors. You go home. You fix the loan slips. You come back. The librarian checks again.

This is the **borrow checker.** It is the heart of Rust. Everything else in Rust is shaped by it. The whole language is designed so the librarian can do its job at compile time, without ever running the program.

### The grown-up picture

Now the grown-up version, in case you skipped to this section. Rust is a systems programming language. It compiles ahead-of-time to native machine code. It has no runtime garbage collector — memory is freed automatically when the variable that owns it goes out of scope, exactly like C++ destructors but with stronger guarantees. The compiler enforces a set of rules called **ownership** and **borrowing** that rule out, statically, an entire family of bugs:

- Use-after-free.
- Double-free.
- Iterator invalidation.
- Most data races.
- Most null pointer dereferences (Rust has no null; it has `Option<T>` instead).
- A surprising amount of buffer overrun.

You pay for those guarantees with a steep learning curve. The first month of writing Rust feels like fighting the compiler. The fight ends. After a while you start writing programs that compile on the first try and then run forever without segfaults. People say Rust gives you the speed of C with the safety of a high-level language. That isn't marketing — it's roughly true.

## Why Rust Got Famous

C and C++ have been the kings of fast software for fifty years. Operating systems, browsers, game engines, databases, anything that has to be fast — written in C or C++. They are also infamous bug factories. Memory corruption is the single biggest source of security vulnerabilities in shipped software. Microsoft published a study showing about **70%** of their CVEs were memory-safety bugs. Google published the same number for Chromium.

Rust came along and said: what if we kept all the speed but ruled out the bugs? That is the entire pitch. The compiler proves at build time that your program does not have those bugs. It does it through ownership and borrowing.

The other modern way to rule out memory bugs is **garbage collection** — the language has a runtime that periodically pauses your program and walks through memory to find unreferenced objects and free them. Java, Go, Python, JavaScript, C# all do this. It works great except: the runtime has overhead, the pauses can be unpredictable, and you can't easily use a garbage-collected language to write a kernel or an embedded device. Rust does not have a garbage collector. It freezes the right answers at compile time so it doesn't need one at run time.

That combination — C-level performance, no GC, memory safety — is why Rust is now used by Linux (kernel modules can be written in Rust as of kernel 6.1), by Microsoft (parts of Windows), by Amazon (Firecracker, the engine under Lambda and Fargate), by Cloudflare, by Discord, by Mozilla (where it was born), by Dropbox, by 1Password, by Figma, by Google's Android, and on and on. It is not a niche thing anymore. It is the new default for "we need this to be fast and we cannot afford a security bug."

### The first stable release

Rust 1.0 shipped in **May 2015.** Before that it was a research project at Mozilla. The language has had many versions since, but the rule is **stability without stagnation**: code that compiled on Rust 1.0 still compiles today. New features ship behind **editions** (like 2015, 2018, 2021, 2024) so the language can evolve without breaking old code. Pick an edition in `Cargo.toml`, the language gives you the matching set of features.

## Ownership

This is the first big idea. Once you have it, half of Rust makes sense.

### Every value has exactly one owner

In Rust, every piece of data that lives on the heap (or on the stack, with caveats) has a single **owner**. The owner is a variable. When that variable goes out of scope, the data is destroyed. There is no garbage collector. There is no `free()` to remember. The compiler inserts the cleanup automatically, at the end of the variable's scope, every time, perfectly.

Picture giving a kid a balloon. The kid is the owner. When the kid leaves the room, the balloon pops. Nobody else can be holding the balloon when the kid leaves, because the balloon belongs to **that kid.** If the kid hands it to another kid, the new kid is the owner now. The old kid no longer has a balloon. The balloon does not pop until the new kid leaves.

```rust
fn main() {
    let balloon = String::from("red");  // 'balloon' owns this String
    println!("{}", balloon);
}                                       // 'balloon' goes out of scope here
                                        // The String is dropped automatically
```

When the closing brace of `main` runs, Rust calls `drop` on `balloon`. Memory is freed. There is no leak.

### Ownership transfers on assignment, function calls, and return

If you write `let b = a;` where `a` is a `String`, **`a` is no longer usable.** `b` is the new owner. `a` has been **moved**. Try to use `a` after that and the compiler refuses to build:

```rust
let a = String::from("hi");
let b = a;
println!("{}", a);  // error[E0382]: borrow of moved value: `a`
```

Same thing happens when you pass a `String` to a function: that function takes ownership unless you say otherwise. Same thing when a function returns a value: ownership of the return value moves to the caller.

This is wild the first time. You go from "variables hold values, who cares" to "every assignment is a move with consequences." You feel like a librarian yourself, scratching out names on cards. Eventually the picture clicks: **the program tracks who is responsible for cleaning up each piece of memory, exactly once, with no ambiguity.**

### Why this rules out a huge class of bugs

In C, you might do `char *p = malloc(10); free(p); printf("%s", p);` — you used `p` after freeing it. That's **use-after-free.** In Rust, after the owner is dropped or moved, the compiler refuses to let you use the old name. It's a compile-time error, not a runtime crash, not a security hole.

You might also do `free(p); free(p);` in C — that's a **double-free**, often a security bug. In Rust, drop happens exactly once because there's exactly one owner. Double-free is impossible without `unsafe`.

### The ownership tree

Picture every value as a node in a tree. The root is some local variable. Children are values owned **by** the root. For example, a `Vec<String>` is a vector that owns its strings. The vector is the root; each string is a child node. When the root drops, every child drops, recursively, automatically.

```
fn main() {
  vec_of_names: Vec<String>     <- root, lives until end of main()
   ├── "alice".to_string()      <- owned by vec
   ├── "bob".to_string()        <- owned by vec
   └── "carol".to_string()      <- owned by vec
  config: Config                <- another root
   ├── name: String
   └── port: u16                <- u16 lives inline, not heap
}
```

When `main()` returns, Rust walks both trees, runs `drop` for every node, frees every byte. No leaks, no manual `free`, no GC.

## Borrowing

Owning every value all the time would be a pain. You'd be moving things constantly. So Rust lets you **borrow** — take a temporary loan on a value without taking ownership. A borrow is called a **reference**, written with an ampersand: `&value`.

There are two kinds of references: shared and exclusive.

### `&T` — shared reference (a.k.a. immutable borrow)

A `&T` lets you **read** a `T` but not modify it. You can have **as many shared references as you want** to the same value at the same time.

```rust
let s = String::from("hello");
let r1 = &s;
let r2 = &s;
let r3 = &s;
println!("{} {} {}", r1, r2, r3);  // fine: three readers
```

Picture a book on a coffee table with three friends reading over each other's shoulders. Nobody writes. Everybody reads. No conflict.

### `&mut T` — exclusive reference (a.k.a. mutable borrow)

A `&mut T` lets you **modify** a `T`. You can have **only one** `&mut T` to a value at any given moment, and **no shared references at the same time.** It's an exclusive lock.

```rust
let mut s = String::from("hello");
let r1 = &mut s;
r1.push_str(" world");     // fine: one writer
println!("{}", r1);
```

But:

```rust
let mut s = String::from("hello");
let r1 = &mut s;
let r2 = &mut s;          // error[E0499]: cannot borrow `s` as mutable more than once
println!("{} {}", r1, r2);
```

Picture one person holding the book and writing in it. Nobody else can read or write while the writer has it. When the writer puts it down, anybody can pick it up.

### The one-line rule

> **At any time, you can have one mutable reference OR any number of immutable references — never both.**

This is "the rule" Rust people talk about. Internalize it. Every borrow checker error will eventually trace back to it.

### Why this rules out data races

Two threads each holding a `&mut T` to the same `T` would let both threads write at the same time — that's a data race. Rust's borrow checker says you can't even make that happen on **one** thread, let alone two. Combined with `Send` and `Sync` (the auto-traits, more later), Rust makes data races a compile error. Other languages catch them with luck and tooling. Rust catches them with the type system.

### The borrow ends, the value lives on

A borrow doesn't take ownership. When the borrow ends — when the variable holding the reference goes out of scope — the value is still owned by whoever owned it before. No drop happens. The borrow was just a loan.

```rust
let s = String::from("hi");
{
    let r = &s;            // r borrows s
    println!("{}", r);
}                          // r goes out of scope; s is still alive
println!("{}", s);         // fine
```

## Lifetimes

Now the tricky part. References in Rust always have a **lifetime** — a region of code during which the reference is guaranteed to be valid. Most of the time the compiler figures lifetimes out automatically. Sometimes you have to write them down.

### Lifetimes in pictures

A lifetime is just a name for "this reference lives at least this long." We write lifetime names with a tick: `'a`, `'b`, `'static`. They look like type parameters but they refer to durations of validity, not types.

```
let s = String::from("hi");     // s starts living
let r = &s;                     // r borrows s; lifetime starts
println!("{}", r);              // r is used
                                // r ends here (last use)
                                // s ends here (end of scope)
```

The lifetime of `r` is the region from where it's declared to its last use. Rust's borrow checker knows: `r` must live no longer than `s`. If `s` was dropped first, `r` would be a dangling reference. The compiler refuses.

### `'static` is the longest lifetime

`'static` means "this reference is valid for the entire run of the program." String literals like `"hello"` have type `&'static str` because they're baked into your binary. They live as long as the binary runs. You can't dangle from a string literal.

### Lifetime elision rules

You don't usually write lifetimes by hand. Rust's **elision** rules fill them in. The three rules:

1. Each `&` parameter gets its own lifetime.
2. If there's exactly one input lifetime, it's assigned to every output reference.
3. If there are multiple input lifetimes but one is `&self` or `&mut self`, the lifetime of `self` is assigned to outputs.

So this:

```rust
fn first_word(s: &str) -> &str { ... }
```

means the same as:

```rust
fn first_word<'a>(s: &'a str) -> &'a str { ... }
```

Compiler fills in the `'a` for you. You only have to write lifetimes when elision can't figure it out — typically when a function takes multiple references and returns one of them.

### When you must write lifetimes

```rust
fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() { x } else { y }
}
```

The compiler needs to know that the return value lives at least as long as both inputs. You tell it by giving them all the same lifetime name `'a`. Without that, the compiler doesn't know which input the output came from.

### Structs with references need lifetimes

```rust
struct Snippet<'a> {
    text: &'a str,
}
```

The struct holds a reference, so it's parameterized by the lifetime of that reference. The struct cannot outlive the data it borrows.

### NLL (Non-Lexical Lifetimes)

In old Rust (before 2018), a reference's lifetime extended to the end of its scope. So `let r = &s; println!("{}", r); /* r still alive here */`. That made many borrow checker errors annoying. Rust 2018 introduced **NLL**: a reference's lifetime ends at its **last use**, not at the closing brace. So in the example, `r`'s lifetime ends after the `println!`, freeing `s` for other borrows. Modern Rust is much friendlier because of this.

## The Borrow Checker

This is the part of the compiler that enforces the ownership and borrowing rules. It's also the part beginners spend a lot of time fighting.

### Decision flow

```
                  +-----------------------------+
                  |  You wrote: let r = &x;     |
                  +--------------+--------------+
                                 |
                                 v
                  +-----------------------------+
                  | Is x in scope here?         |
                  +------+------------+---------+
                       no |          | yes
                          v          v
                 [E0425  ]   +-------------------+
                              | Is there an      |
                              | active &mut x    |
                              | already?         |
                              +-----+-----+------+
                                 yes|     |no
                                    v     v
                          [E0502/E0499]   +------------------+
                                          | Is r supposed to |
                                          | outlive x?       |
                                          +-----+------+-----+
                                              yes|      |no
                                                 v      v
                                          [E0716/   [borrow OK]
                                           E0597]
```

### What good error messages look like

The borrow checker prints what's wrong, where, and usually how to fix it:

```
error[E0382]: borrow of moved value: `s`
   --> src/main.rs:5:20
    |
3   |     let s = String::from("hi");
    |         - move occurs because `s` has type `String`, which does not implement the `Copy` trait
4   |     let t = s;
    |             - value moved here
5   |     println!("{}", s);
    |                    ^ value borrowed here after move
    |
help: consider cloning the value if the performance cost is acceptable
    |
4   |     let t = s.clone();
    |              ++++++++
```

Read the message carefully every time. The compiler will literally tell you where the move happened, why it happened, and suggest a fix.

## Move Semantics vs Copy Trait

So far we said "assignment moves." That's true for most types. But some types — small, simple, fixed-size things like integers, floats, booleans, characters, and tuples of `Copy` types — implement a special trait called **`Copy`**. When a `Copy` type is assigned, the bytes are duplicated and the original stays usable.

```rust
let a: i32 = 5;
let b = a;
println!("{} {}", a, b);  // fine: i32 is Copy
```

vs.

```rust
let a: String = String::from("hi");
let b = a;
println!("{} {}", a, b);  // error: String is not Copy; a was moved into b
```

### Why integers are Copy and Strings aren't

An `i32` is just 4 bytes on the stack. Copying it is cheap and harmless — there's no heap allocation to share, no resource to manage. A `String` owns a heap allocation. If you "copied" a String just by duplicating its three-word handle, you'd have two Strings pointing at the same heap buffer — and when both go out of scope, you'd free the buffer twice. Disaster.

So: **types that own no resource can be `Copy`. Types that own a resource cannot.** Move is the default; `Copy` is opt-in for the trivial cases.

### `Clone` is the manual version

`Clone` is "explicit deep copy." Any type that wants to support being duplicated implements `Clone`, and you call `.clone()` to make a copy:

```rust
let a = String::from("hi");
let b = a.clone();        // explicit deep copy; both a and b are usable
println!("{} {}", a, b);
```

Idiom: `Copy` is automatic, `Clone` is manual. `Copy` implies `Clone` (you can always derive both). `Clone` doesn't imply `Copy`.

## Traits

A **trait** in Rust is like an interface in Java or a protocol in Swift: a named set of methods that types can implement. If a type implements a trait, you can use that trait's methods on it.

### Defining and implementing

```rust
trait Greet {
    fn hello(&self) -> String;
}

struct Cat { name: String }
impl Greet for Cat {
    fn hello(&self) -> String {
        format!("meow, I'm {}", self.name)
    }
}
```

Now any `Cat` has a `.hello()` method.

### Generics with trait bounds

You can write a function that works on **any type that implements a trait**:

```rust
fn shout<T: Greet>(thing: T) {
    println!("{}!", thing.hello().to_uppercase());
}
```

The `T: Greet` is a **trait bound**. It says: this function works for any `T`, but only if `T` implements `Greet`. The compiler **monomorphizes** — it generates a separate copy of `shout` for each concrete `T` you call it with. There's no runtime dispatch. It's as fast as if you'd written the function for that exact type.

### `impl Trait`

A shorter syntax for "some type that implements this trait":

```rust
fn make_greeter() -> impl Greet { Cat { name: "milo".into() } }
```

The caller doesn't see the concrete type — just that the return implements `Greet`.

### `dyn Trait` — trait objects

Sometimes you want to mix multiple types behind one trait at runtime — a list of greeters where some are `Cat` and some are `Dog`. You use a **trait object**:

```rust
let greeters: Vec<Box<dyn Greet>> = vec![
    Box::new(Cat { name: "milo".into() }),
    Box::new(Dog { name: "rex".into() }),
];
for g in &greeters {
    println!("{}", g.hello());
}
```

`dyn Greet` is a **trait object** — a fat pointer (data pointer + vtable pointer) that lets you call trait methods at runtime. Unlike `impl Trait`, `dyn Trait` uses **dynamic dispatch**: there's a virtual table lookup at every method call. Slightly slower; way more flexible.

Pick `impl Trait` when one concrete type comes out of the function. Pick `dyn Trait` when you genuinely need a heterogeneous collection.

### The orphan rule (a.k.a. coherence)

Big gotcha: you can only `impl Trait for Type` if **either the trait or the type is defined in your crate**. You can't add your own impl for a third-party trait on a third-party type. Otherwise two crates could define conflicting impls and the compiler couldn't tell which one wins.

Workaround: the **newtype pattern** — wrap the foreign type in a tuple struct in your crate and impl on that.

## Common Traits

Here is the cast of usual suspects you will see everywhere:

- **`Debug`** — `{:?}` printing. For developer output. `#[derive(Debug)]` to auto-derive.
- **`Display`** — `{}` printing. For user-facing output. Hand-written.
- **`Clone`** — `.clone()` for explicit duplication.
- **`Copy`** — implicit duplication on move. Marker trait; no methods.
- **`PartialEq` / `Eq`** — `==` operator. `PartialEq` allows weird cases (NaN); `Eq` says "true equivalence."
- **`PartialOrd` / `Ord`** — `<`, `<=`, sort.
- **`Hash`** — hashable; lets a value be a key in `HashMap`.
- **`Default`** — `T::default()` for an "empty" value (`0`, `""`, `[]`).
- **`Iterator`** — has a `.next()` method; everything iterable.
- **`IntoIterator`** — convertible into an `Iterator`. `for x in coll` calls `.into_iter()`.
- **`From<T>` / `Into<T>`** — conversion. Implementing `From` gets you `Into` for free.
- **`AsRef<T>`** — cheap reference conversion. `fn open<P: AsRef<Path>>(p: P)` accepts `&str`, `String`, `Path`, `PathBuf`...
- **`Drop`** — custom destructor. `fn drop(&mut self) { ... }` runs when the value is dropped.
- **`Send`** — value can be moved to another thread.
- **`Sync`** — value can be referenced from multiple threads.

Most of these you `#[derive]`:

```rust
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default)]
struct Point { x: i32, y: i32 }
```

That single line gives you printing, cloning, equality, hashing, and a default constructor.

## Error Handling

Rust has no exceptions. Instead, errors are **values returned by functions**. The compiler forces you to handle them.

### `Result<T, E>`

The standard error type. Functions that can fail return `Result<T, E>`:

```rust
enum Result<T, E> {
    Ok(T),
    Err(E),
}
```

`Ok(x)` means success with value `x`. `Err(e)` means failure with error `e`.

```rust
let f = std::fs::File::open("hello.txt");
match f {
    Ok(file)   => println!("opened: {:?}", file),
    Err(error) => println!("could not open: {}", error),
}
```

You **must** handle both arms — the compiler will warn if you ignore a `Result`.

### `Option<T>`

For things that might be absent (Rust's answer to null):

```rust
enum Option<T> {
    Some(T),
    None,
}
```

```rust
let map = std::collections::HashMap::from([("alice", 30)]);
match map.get("alice") {
    Some(&age) => println!("age: {}", age),
    None       => println!("no alice"),
}
```

Rust **does not have null pointers** in safe code. Every "might be absent" value is `Option<T>`. The compiler forces you to think about the `None` case.

### The `?` operator

Writing match for every Result quickly gets tedious. The `?` operator unwraps an `Ok` or returns an `Err` early:

```rust
fn read_file(path: &str) -> Result<String, std::io::Error> {
    let mut s = String::new();
    std::fs::File::open(path)?.read_to_string(&mut s)?;
    Ok(s)
}
```

Each `?`: if `Ok`, take the inner value; if `Err`, return it from the function. Brilliant.

### `anyhow` and `thiserror`

Two extremely popular crates for error handling. They are not in the standard library — you add them via Cargo.

- **`anyhow`** is for **applications**. You use `anyhow::Result<T>` (which is `Result<T, anyhow::Error>`), and `anyhow::Error` swallows any error type. Easy. Loose. Great for binaries.
- **`thiserror`** is for **libraries**. You define your own error enum and derive `thiserror::Error` on it. Each variant gets a clean `Display` impl. Great for typed errors that callers can match on.

Rule of thumb: write `thiserror` enums in libraries, use `anyhow` in your `main.rs`.

## Pattern Matching

`match` is Rust's super-charged switch statement.

```rust
let x = 3;
match x {
    1       => println!("one"),
    2 | 3   => println!("two or three"),
    4..=10  => println!("four to ten"),
    n if n < 0 => println!("negative {}", n),
    _       => println!("something else"),
}
```

You can match enums, structs, tuples, references, ranges, literals, guards, you name it.

```rust
match point {
    Point { x: 0, y: 0 } => println!("origin"),
    Point { x, y: 0 }    => println!("on x-axis at {}", x),
    Point { x: 0, y }    => println!("on y-axis at {}", y),
    Point { x, y }       => println!("at ({}, {})", x, y),
}
```

### Exhaustiveness

The compiler **forces** match to cover every possible value. Miss a case and the program won't build. This is one of Rust's quiet superpowers — when you add a new variant to an enum, every `match` on that enum becomes a compile error until you handle the new variant. You literally cannot forget.

### `if let` and `while let`

Shorter syntax for one-arm matches:

```rust
if let Some(x) = maybe_something {
    println!("got {}", x);
}

while let Some(line) = lines.next() {
    println!("{}", line);
}
```

## Modules and Visibility

Code is organized into **modules**. A module groups types, functions, traits, other modules. By default, everything is **private** to the module that defines it. You must add `pub` to expose it.

```rust
mod math {
    pub fn add(a: i32, b: i32) -> i32 { a + b }
    fn secret_helper() {}    // private; visible only inside math
}

fn main() {
    math::add(2, 3);
    // math::secret_helper(); // error: private
}
```

### Visibility levels

- `pub` — visible everywhere it's exported.
- `pub(crate)` — visible anywhere inside this crate, but not exposed outside.
- `pub(super)` — visible to the parent module only.
- `pub(in path::to::mod)` — visible to a specific module.
- (no qualifier) — private to the defining module.

### Files and modules

`mod foo;` in `lib.rs` looks for `foo.rs` or `foo/mod.rs`. Each file is a module. Submodules nest. The full path is `crate::foo::bar::baz`.

### `use` for shortcuts

```rust
use std::collections::HashMap;
use std::io::{self, Read, Write};
```

Otherwise you'd type `std::collections::HashMap` every single time.

## Cargo

Cargo is Rust's package manager and build tool. It downloads dependencies, runs the compiler, runs tests, builds release binaries, generates docs, runs benchmarks. It is fantastic. Other languages have envied it for a decade.

### `Cargo.toml` — the manifest

Every Rust project has a `Cargo.toml` at its root:

```toml
[package]
name = "myproj"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = { version = "1", features = ["derive"] }
tokio = { version = "1", features = ["full"] }

[dev-dependencies]
criterion = "0.5"

[build-dependencies]
cc = "1"

[features]
default = ["json"]
json = []
yaml = ["dep:serde_yaml"]

[profile.release]
opt-level = 3
lto = true
```

### `Cargo.lock` — the snapshot

A file Cargo writes that records the exact version of every dependency. **Commit it** for binaries (so your CI builds the same thing as your dev box). **Don't commit it** for libraries (so downstream users can pick versions). This is the rule.

### Workspaces

A workspace is a folder with a top-level `Cargo.toml` that lists multiple crates as members:

```toml
[workspace]
members = ["crate-a", "crate-b", "crate-c"]
```

All members share one `target/` directory and one `Cargo.lock`. Faster builds, easier dependency management for big projects.

### Features

Optional bundles of behavior. You enable features at build time:

```bash
$ cargo build --features yaml
$ cargo build --no-default-features --features minimal
```

Crates expose features for things like async runtimes, serialization formats, OS-specific code paths.

### Profiles

Build configurations. Default ones are `dev`, `release`, `test`, `bench`. You can tune any of them in `Cargo.toml`.

## Testing

Testing is built into the language, not bolted on as a library.

### Unit tests live next to code

```rust
fn add(a: i32, b: i32) -> i32 { a + b }

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_add() {
        assert_eq!(add(2, 3), 5);
    }

    #[test]
    #[should_panic]
    fn test_panics() { panic!("ok"); }
}
```

`#[cfg(test)]` means "only compile this when running `cargo test`." `#[test]` marks a test function. Run with:

```bash
$ cargo test
```

### Integration tests live in `tests/`

Files in `tests/foo.rs` are compiled as separate crates and only see your library's public API:

```
myproj/
├── src/lib.rs
└── tests/
    ├── api.rs
    └── workflow.rs
```

### Doctests

Every code block in a `///` doc comment is also a test:

```rust
/// Adds two numbers.
///
/// # Examples
///
/// ```
/// assert_eq!(myproj::add(2, 3), 5);
/// ```
pub fn add(a: i32, b: i32) -> i32 { a + b }
```

`cargo test` runs unit tests, integration tests, **and** doctests. Documentation that lies cannot survive.

## Async Rust

Rust has built-in support for asynchronous programming via `async fn` and `.await`. It is famously slippery to learn but extremely powerful once it clicks.

### `async fn` returns a Future

```rust
async fn fetch_url(url: &str) -> String {
    // ... async I/O ...
}
```

When you call `fetch_url("...")` you do **not** run the code. You get back a `Future` — an object that, when "polled" by an executor, will eventually produce a `String`. Calling an async fn is roughly free; doing the work happens elsewhere.

### `.await` polls the Future

```rust
let body = fetch_url("https://example.com").await;
```

`.await` says: "drive this Future until it produces a value, and let me have the value." Crucially, while waiting, the current task can be **suspended** so the executor can run other tasks. That's the magic of async — concurrency without threads.

### You need an executor

Rust does not bundle an executor in the standard library. You pick one as a dependency. The most common is **tokio**:

```rust
#[tokio::main]
async fn main() {
    let body = fetch_url("https://example.com").await;
    println!("{}", body);
}
```

Other executors: **smol** (small, simple), **async-std** (mirrors `std`), **embassy** (embedded). They all run Futures; tokio is the de-facto choice for servers and CLIs.

### The `Pin<>` dance

Once a Future is being polled, it contains internal references to itself (state machines do this). It must not be moved in memory after polling starts. **`Pin<P>`** is a wrapper that says "this pointee will not move." Most of the time you don't need to think about Pin — `tokio::spawn` and `.await` handle it. When you write your own Future or use `Stream`, Pin shows up.

```
Future state machine, ASCII style:

  state 0: about to call fetch_url
  await: state 1: blocked on socket recv
  awakened: state 2: parse response
  return: state 3: done

Each transition saves enough state to resume in place. Pin keeps the
self-references inside the state machine valid across resumes.
```

### `tokio::select!` and friends

Run multiple Futures concurrently and act on the first one to finish:

```rust
tokio::select! {
    result = fetch_url("a") => println!("a: {}", result),
    result = fetch_url("b") => println!("b: {}", result),
    _ = tokio::time::sleep(Duration::from_secs(5)) => println!("timeout"),
}
```

Other tools: `join!` (wait for all), `try_join!` (wait for all, short-circuit on Err), `FuturesUnordered` (a streaming collection of Futures).

## Smart Pointers

Smart pointers wrap values and add a behavior. They're plain structs implementing `Deref` (and usually `Drop`).

### `Box<T>` — heap allocation

`Box<T>` puts a value on the heap and gives you a unique pointer to it. Use it when:

- A value is too big for the stack.
- You want a recursive type (a linked list of `Box<Node>`).
- You want a trait object (`Box<dyn Trait>`).

```rust
let big: Box<[u8; 1_000_000]> = Box::new([0; 1_000_000]);
```

### `Rc<T>` — reference-counted, single-threaded

Multiple owners! `Rc<T>` lets several variables share ownership of the same value. It tracks a count; when the count hits zero, the value drops.

```rust
use std::rc::Rc;
let a = Rc::new(String::from("hi"));
let b = Rc::clone(&a);     // both own the same String
let c = Rc::clone(&a);
println!("count: {}", Rc::strong_count(&a));  // 3
```

`Rc<T>` is **not thread-safe**. Use `Arc<T>` across threads.

### `Arc<T>` — atomically reference-counted, multi-threaded

Same idea but the count uses atomic operations so it's safe across threads.

```rust
use std::sync::Arc;
use std::thread;

let s = Arc::new(String::from("shared"));
for _ in 0..4 {
    let s = Arc::clone(&s);
    thread::spawn(move || println!("{}", s));
}
```

### `RefCell<T>` — interior mutability, single-threaded

Sometimes you have a `&T` and you actually need to mutate. `RefCell<T>` defers the borrow check from compile time to **runtime**. You call `.borrow()` for shared, `.borrow_mut()` for exclusive. If you violate the rules, your program panics.

```rust
use std::cell::RefCell;
let c = RefCell::new(0);
*c.borrow_mut() += 1;
*c.borrow_mut() += 1;
println!("{}", c.borrow());
```

### `Mutex<T>` and `RwLock<T>` — interior mutability, multi-threaded

`Mutex<T>` gives exclusive access; `RwLock<T>` gives many-reader-or-one-writer:

```rust
use std::sync::Mutex;
let m = Mutex::new(0);
{
    let mut g = m.lock().unwrap();
    *g += 1;
}                              // lock released when guard drops
```

Don't forget: lock guards drop releases the lock. Release order = reverse of acquire order. Avoid holding two locks at once.

### `Cow<T>` — clone on write

`Cow<T>` is "either a reference or an owned value." It's useful when you might or might not need to allocate:

```rust
use std::borrow::Cow;
fn maybe_lower(s: &str) -> Cow<str> {
    if s.chars().all(|c| c.is_lowercase()) {
        Cow::Borrowed(s)         // no allocation
    } else {
        Cow::Owned(s.to_lowercase())  // allocate when modifying
    }
}
```

## Unsafe Rust

Rust's safety guarantees come from the borrow checker. There are some things the borrow checker can't prove safe — talking to hardware, calling C, certain pointer manipulations. For those, you use **`unsafe`** blocks.

```rust
let mut x = 5;
let r = &mut x as *mut i32;     // raw pointer
unsafe {
    *r = 10;                    // raw deref, only allowed in unsafe block
}
```

Inside `unsafe`, you can:

- Dereference raw pointers (`*const T`, `*mut T`).
- Call unsafe functions and FFI functions.
- Implement unsafe traits.
- Access mutable static variables.
- Access fields of unions.

### The unsafe contract

`unsafe` doesn't turn off the borrow checker for safe code — it just lets you do certain things the safe checker can't verify. You're telling the compiler: **"I have manually verified that this code upholds Rust's invariants."** If you lie, your program has undefined behavior. The whole point of Rust is to minimize unsafe code and put a sound safe API on top of it. Use unsafe sparingly. Wrap it in a safe API. Document the invariants.

Tools: **miri** (interprets your program and detects undefined behavior). **loom** (concurrency model checker). **kani** (formal verification). **prusti** (pre/postconditions). Using these on unsafe code is gold.

## Macros

Rust has two flavors of macro: declarative and procedural.

### Declarative macros (`macro_rules!`)

Pattern-match on syntax and expand to other syntax:

```rust
macro_rules! square {
    ($x:expr) => { $x * $x };
}

let s = square!(3);   // expands to 3 * 3
```

Most common: `println!`, `vec!`, `format!`, `assert_eq!` are all declarative macros.

### Procedural macros

Procedural macros are mini-compilers that transform Rust code. Three kinds:

- **Derive macros** — `#[derive(MyTrait)]` generates impls.
- **Attribute macros** — `#[my_attr] fn foo() { ... }` rewrites the function.
- **Function-like macros** — `my_macro!(input)` like declarative but written in Rust code.

Examples in the wild: `serde::Serialize`, `tokio::main`, `sqlx::query!`. Procedural macros run at compile time; they're powerful and slow to compile.

## FFI

Rust talks to C with **FFI** (Foreign Function Interface).

### Calling C from Rust

```rust
extern "C" {
    fn abs(x: i32) -> i32;
}

fn main() {
    let n = unsafe { abs(-5) };
    println!("{}", n);
}
```

The `extern "C"` declares C-style calling convention. The call is `unsafe` because Rust can't verify what C does.

### Calling Rust from C

```rust
#[no_mangle]
pub extern "C" fn rust_add(a: i32, b: i32) -> i32 {
    a + b
}
```

`#[no_mangle]` keeps Rust from renaming the symbol. `extern "C"` makes it use C calling convention.

### Tools

- **bindgen** — generates Rust bindings from C headers automatically. Indispensable for big C libraries.
- **cbindgen** — generates C headers from Rust code.

## Common Errors

The Rust compiler errors look long the first time. They are not warnings. They are help. Each one tells you exactly where the bug is and usually how to fix it. Here are the famous ones:

```
error[E0382]: borrow of moved value: `s`
```
The value was moved (assigned, returned, passed to a function) and then used afterwards. Fix: clone the value, or use a reference, or restructure to not need the value twice.

```
error[E0499]: cannot borrow `x` as mutable more than once at a time
```
You tried to have two `&mut` to the same thing. Fix: end one borrow before starting the other (split scopes, restructure, or clone).

```
error[E0502]: cannot borrow `x` as mutable because it is also borrowed as immutable
```
A `&` is alive at the same time as a `&mut`. Fix: end the `&` before the `&mut`.

```
error[E0596]: cannot borrow `x` as mutable, as it is not declared as mutable
```
You forgot `let mut x = ...`. Add `mut`.

```
error[E0277]: the trait bound `T: U` is not satisfied
```
`T` does not implement trait `U`. Fix: implement the trait for `T`, or use a different type, or relax the bound.

```
error[E0308]: mismatched types
expected `String`, found `&str`
```
The type doesn't match the signature. Fix: convert with `.to_string()`, `.into()`, `String::from(...)`, or change the signature.

```
error[E0716]: temporary value dropped while borrowed
```
You took a reference to a temporary that doesn't outlive the borrow. Fix: bind the temporary to a `let`.

```
error[E0521]: borrowed data escapes outside of function
```
You returned a reference to something that didn't live long enough. Fix: return owned data, or restructure lifetimes.

```
error[E0119]: conflicting implementations of trait
```
Two impls overlap (orphan rule violation usually). Fix: only impl in the crate that defines either the trait or the type.

```
error[E0277]: `X` cannot be sent between threads safely
```
You tried to move a non-`Send` type to another thread. Fix: use `Send` types (e.g., `Arc<Mutex<T>>` instead of `Rc<RefCell<T>>`).

```
error[E0382]: use of moved value
```
Same family as the first one — used after move.

```
error[E0106]: missing lifetime specifier
```
Function returns a reference but the compiler can't elide which input it came from. Fix: write explicit `'a` lifetimes.

```
error[E0623]: lifetime mismatch
```
The lifetimes on inputs and outputs don't line up. Fix: make sure references have compatible lifetimes; consider returning owned data.

When in doubt: read every word of the error. Click any link the compiler prints. Run `rustc --explain E0382` for a long explanation.

## Hands-On

Type each line into your terminal. The point is to build muscle memory.

```bash
$ rustup --version                # tells you rustup is installed
$ rustup default stable           # use stable channel
$ rustup install nightly          # also install nightly
$ rustup target add wasm32-unknown-unknown
$ rustc --version                 # what compiler version
$ rustc --print target-list       # every platform Rust can build for

$ cargo new myproj                # makes new project folder + git repo
$ cd myproj
$ cargo init                      # init in existing folder

$ cargo build                     # debug build, fast compile, slow run
$ cargo build --release           # release build, slow compile, fast run
$ cargo run                       # build + run
$ cargo run -- --arg=value        # forward args after `--` to your program

$ cargo test                                          # run all tests
$ cargo test integration_test_name -- --nocapture     # run one test, show prints
$ cargo bench                                         # run benchmarks (nightly or criterion)

$ cargo doc --open                # build and open generated docs
$ cargo clippy                    # extra lints
$ cargo clippy --fix              # apply easy fixes
$ cargo fmt --check               # check formatting
$ cargo check                     # type-check without producing a binary (fast)

$ cargo update                    # bump dependency lockfile within semver
$ cargo tree                      # show dependency graph
$ cargo tree -d                   # only show duplicate deps

$ cargo audit                     # security advisories (cargo install cargo-audit first)
$ cargo outdated                  # show outdated deps
$ cargo expand                    # see proc-macro expansion (cargo install cargo-expand)
$ cargo asm myfn                  # see generated assembly for a function

$ cargo flamegraph                # build a flamegraph profile (cargo install flamegraph)
$ rust-gdb target/debug/myapp     # gdb with Rust pretty-printers
$ rust-lldb target/debug/myapp    # lldb with Rust pretty-printers

$ cargo install cargo-edit        # adds cargo add / cargo rm
$ cargo add tokio --features full
$ cargo remove anyhow

$ cargo workspaces list           # in a multi-crate repo (cargo install cargo-workspaces)

$ cargo +nightly fmt              # use nightly toolchain just for this command
$ cargo +nightly miri test        # run miri (UB detector) under nightly

$ cargo install cargo-watch && cargo watch -x test    # rerun tests on file change

$ cargo install cross && cross build --target aarch64-unknown-linux-gnu

$ RUST_BACKTRACE=1 cargo run                # show backtrace on panic
$ RUST_LOG=trace cargo run                  # set log level for env_logger / tracing
$ RUSTFLAGS='-C target-cpu=native' cargo build --release   # tune for your CPU

$ rustup component add miri                 # UB detector for unsafe code
$ rustup component add rust-src             # for std source
```

### Worked example: hello, world

```bash
$ cargo new hello && cd hello
$ cat src/main.rs
fn main() {
    println!("Hello, world!");
}
$ cargo run
   Compiling hello v0.1.0 (.../hello)
    Finished `dev` profile [unoptimized + debuginfo] target(s) in 0.42s
     Running `target/debug/hello`
Hello, world!
```

That's a complete Rust program: source file, `cargo new`, `cargo run`. The whole loop is two commands.

### Worked example: a tiny calculator

```bash
$ cargo new calc && cd calc
$ cat > src/main.rs <<'EOF'
use std::env;

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() != 4 {
        eprintln!("usage: calc <a> <op> <b>");
        std::process::exit(1);
    }
    let a: f64 = args[1].parse().expect("a not a number");
    let op = args[2].as_str();
    let b: f64 = args[3].parse().expect("b not a number");

    let r = match op {
        "+" => a + b,
        "-" => a - b,
        "x" => a * b,
        "/" => a / b,
        _   => { eprintln!("unknown op {}", op); std::process::exit(1); }
    };
    println!("{}", r);
}
EOF
$ cargo run -- 3 + 4
7
$ cargo run -- 12 / 5
2.4
```

`env::args()` returns command-line args. `parse()` returns a `Result` that we unwrap with `.expect(msg)` — fine for a tiny program, prefer `?` in larger code. `match` on `&str` covers each operator. `eprintln!` writes to stderr.

## Common Confusions

Things that trip up everyone the first time:

### `String` vs `&str`

`String` is an **owned, heap-allocated, growable UTF-8 string**. `&str` is a **borrowed slice into a UTF-8 string** — it points into a `String` or into a static literal. Both are guaranteed valid UTF-8. Function parameters should usually be `&str` (most flexible). Owned data goes in `String`.

```rust
fn greet(name: &str) {              // accepts &String, &str, literal, ...
    println!("hi {}", name);
}
let owned = String::from("alice");
greet(&owned);
greet("bob");
```

### `Vec<T>` vs `&[T]`

Same pattern: `Vec<T>` is owned; `&[T]` is a borrowed slice. Function params should usually take `&[T]`.

### `Box<T>` vs `Rc<T>` vs `Arc<T>`

- `Box<T>` — single owner, heap allocation.
- `Rc<T>` — multiple owners, single thread, cheap clone of the handle.
- `Arc<T>` — multiple owners, multiple threads, atomic counter.

If you don't need shared ownership, use `Box`. If you need shared ownership and you're single-threaded, use `Rc`. If you need shared ownership across threads, use `Arc`.

### `Mutex` vs `RwLock` vs `RefCell` vs `Cell`

- `Cell<T>` — interior mutability for `Copy` types only, single thread, no borrowing.
- `RefCell<T>` — interior mutability for any type, single thread, runtime borrow check.
- `Mutex<T>` — multi-thread, exclusive access.
- `RwLock<T>` — multi-thread, many readers or one writer.

### `async fn` vs `fn -> impl Future`

`async fn foo() -> T` is sugar for `fn foo() -> impl Future<Output = T>`. They're (almost) the same. Use `async fn` for readability; the desugared form is for traits and trait objects.

### What is `Pin` and why does Future need it

A `Future` is compiled to a state machine. The state often contains references that point inside itself ("hold this slice across the await"). If the Future were moved in memory, those self-references would point to the wrong place. `Pin<P>` is a wrapper that says "the pointee will not move." `.await` requires its operand to be Pinned. The executor handles Pinning behind the scenes; you only deal with Pin if you write your own Future or use `Stream` directly.

### `tokio` vs `smol` vs `async-std`

- `tokio` — biggest ecosystem, opinionated, used by hyper, axum, tonic.
- `smol` — minimal, fast, embeddable.
- `async-std` — mirrors `std`'s shape with async versions; smaller crowd.

Pick `tokio` unless you have a reason. The world ships on tokio.

### The orphan rule

You can only `impl Trait for Type` if either `Trait` or `Type` is local to your crate. This stops two crates from defining conflicting impls. The fix when you want to impl a foreign trait on a foreign type is the **newtype**:

```rust
struct MyVec(Vec<i32>);             // newtype wrapper
impl SomeForeignTrait for MyVec { ... }
```

### `?Sized` and DSTs

By default every generic parameter `T` requires `T: Sized` (the size is known at compile time). To allow unsized types like `[u8]` or `dyn Trait`, add `?Sized`:

```rust
fn print<T: ?Sized + Debug>(x: &T) { println!("{:?}", x); }
```

You'll mostly see `?Sized` on bounds for things you only ever take by reference.

### `dyn Trait` vs `impl Trait`

`impl Trait` — one concrete type, monomorphized, fast. `dyn Trait` — trait object, runtime dispatch, slower but heterogeneous. Use `impl` for "I return some type that does this." Use `dyn` for "I have a list of different types that all do this."

### `move` closures vs reference closures

By default, a closure captures variables by reference. `move ||` captures by value (moving them in). Spawning a thread or task usually requires `move` because the closure outlives the surrounding scope:

```rust
let s = String::from("hi");
thread::spawn(move || println!("{}", s));   // 's' is moved into the closure
```

### Lifetime elision

For `fn foo(s: &str) -> &str`, Rust assumes the return reference has the same lifetime as the input. If there are multiple inputs and one is `&self`, the return takes `self`'s lifetime. Otherwise you have to write lifetimes by hand.

### What `'static` actually means

A lifetime, not a memory category. `'static` means "this reference is valid for the whole program." It does **not** mean "this thing is in a static memory region" (though string literals happen to be). In a generic bound `T: 'static`, it means "T contains no non-'static references" — which is true of all owned types.

### `Send` and `Sync` auto-traits

The compiler automatically implements `Send` and `Sync` for any type whose fields are all `Send`/`Sync`. `Rc<T>` is `!Send` because cloning the handle from two threads would race the count. `Arc<T>` is `Send` because the count uses atomics. Most everyday types are `Send + Sync` without you doing anything.

### Why does `Cell` exist if `RefCell` exists

`Cell<T>` is for `Copy` types and lets you `get` and `set` without borrowing. `RefCell<T>` uses runtime borrow checking. `Cell` is faster for the simple case (no borrow tracking); `RefCell` is more flexible.

### How lifetimes interact with async

Lifetime errors are nastier in async code because the Future captures references and may be moved across `.await` points. If a function returns `impl Future + 'a`, the future borrows for `'a`. Often the cleanest fix is to clone or own the data instead of borrowing across `.await`.

## Vocabulary

| Word | Plain English |
| --- | --- |
| Rust | A systems programming language with a strict compile-time librarian. |
| rustc | The Rust compiler. Turns `.rs` files into binaries. |
| cargo | Rust's package manager and build tool. Front door to almost everything. |
| rustup | Tool that installs and updates Rust toolchains and components. |
| crate | The smallest compile unit. A library or a binary. |
| package | A folder with a `Cargo.toml`; one or more crates. |
| workspace | A multi-package project sharing one `target/` and lockfile. |
| Cargo.toml | Manifest file that describes the package, deps, features, etc. |
| Cargo.lock | Recorded exact dependency versions; commit for binaries. |
| dependency | A crate your code uses. |
| dev-dependency | A crate used only during testing or examples. |
| build-dependency | A crate used only by `build.rs`. |
| target | A build output; or a CPU/OS target triple. |
| lib | Library crate. |
| bin | Binary crate (program). |
| example | A small executable in `examples/` for showing usage. |
| test | A test executable in `tests/` or a function annotated `#[test]`. |
| bench | A benchmark function. |
| profile | A build configuration: dev, release, test, bench. |
| feature | An optional bundle of behavior toggled at build time. |
| default-features | Features enabled by default; can be turned off with `--no-default-features`. |
| optional | A dependency that only links when a feature enables it. |
| no_std | Code that does not use the standard library. |
| std | The standard library. |
| core | The lowest-level subset of std that needs no allocator. |
| alloc | The middle layer that adds heap-allocating types but no OS. |
| compiler error | Build-time failure with `error[Exxxx]` code. |
| lint | A non-fatal warning about questionable code. |
| clippy | A lint tool with hundreds of extra checks. |
| rustfmt | The standard auto-formatter. |
| edition | A language version (2015, 2018, 2021, 2024). |
| MSRV | Minimum Supported Rust Version a crate promises to compile on. |
| stable | The release channel updated every six weeks. |
| beta | The next stable, in test for six weeks. |
| nightly | Daily snapshot with experimental features. |
| target triple | Identifier like `x86_64-unknown-linux-gnu`. |
| host | The platform the compiler is running on. |
| ownership | Each value has exactly one variable responsible for it. |
| borrow | Temporary access to a value without taking ownership. |
| reference | A pointer with lifetime tracking; `&T` or `&mut T`. |
| `&T` | Shared (immutable) reference. |
| `&mut T` | Exclusive (mutable) reference. |
| lifetime | A region of code during which a reference is valid. |
| `'a` | A named lifetime parameter. |
| `'static` | The lifetime of the whole program. |
| lifetime elision | Compiler-inferred lifetimes you don't have to write. |
| NLL | Non-Lexical Lifetimes; references end at last use, not scope end. |
| drop | Destructor; called when an owner goes out of scope. |
| Drop trait | The trait that lets you customize drop. |
| RAII | Resource Acquisition Is Initialization; resources tied to scope. |
| move | Ownership transfer that invalidates the old binding. |
| copy | Implicit duplication for `Copy` types. |
| Copy trait | Marker for types that can be copied bitwise. |
| Clone trait | Trait for explicit deep duplication via `.clone()`. |
| Send | Auto-trait: this type can be moved across threads safely. |
| Sync | Auto-trait: this type can be referenced from multiple threads. |
| !Send | Type that is not Send (e.g. `Rc<T>`). |
| !Sync | Type that is not Sync. |
| auto-trait | A trait that the compiler implements automatically based on fields. |
| marker trait | A trait with no methods, used to label a property. |
| trait | A named set of methods; like an interface. |
| impl | Implementation block; defines methods or implements a trait. |
| inherent impl | `impl Type { ... }` — methods on the type. |
| trait impl | `impl Trait for Type { ... }` — implements a trait. |
| generic | Code parameterized by type. |
| type parameter | A placeholder type like `T`. |
| trait bound | A constraint like `T: Display`. |
| where clause | A `where` chunk listing trait bounds (cleaner for many bounds). |
| associated type | A type member of a trait, set per impl. |
| associated const | A constant member of a trait, set per impl. |
| default method | A method with a body in the trait, optionally overridden. |
| supertrait | A trait that another trait requires. |
| blanket impl | An impl that covers many types (e.g. `impl<T: A> B for T`). |
| orphan rule | You can only impl a trait if you own the trait or the type. |
| coherence | The set of rules that prevent conflicting impls. |
| trait object | A pointer-with-vtable to something implementing a trait: `dyn Trait`. |
| `dyn Trait` | Spelled-out trait object form. |
| fat pointer | Pointer with extra metadata (data pointer + vtable or length). |
| vtable | Table of function pointers used for dynamic dispatch. |
| monomorphization | Compiler generates a copy of a generic per concrete type. |
| generic instantiation | The act of monomorphizing for one concrete type. |
| sized | Type whose size is known at compile time. |
| `?Sized` | Relaxed bound allowing unsized types. |
| DST | Dynamically Sized Type (e.g. `[u8]`, `dyn Trait`). |
| unsized type | Same as DST. |
| slice | A view into a contiguous sequence: `&[T]`, `&str`. |
| str | The string slice type, always behind a reference. |
| String | Owned, growable UTF-8 string. |
| `&str` | Borrowed UTF-8 string slice. |
| `[T]` | Slice of T; only used behind a reference. |
| `Vec<T>` | Owned, growable array. |
| `Box<T>` | Heap-allocated single-owner pointer. |
| `Rc<T>` | Single-thread reference-counted pointer. |
| `Arc<T>` | Atomic reference-counted pointer (multi-thread). |
| `Cell<T>` | Interior mutability for Copy types. |
| `RefCell<T>` | Interior mutability with runtime borrow checks. |
| `Mutex<T>` | Multi-thread exclusive lock. |
| `RwLock<T>` | Multi-thread reader/writer lock. |
| Atomic* | Lock-free integer/pointer types like `AtomicU32`. |
| `Cow<T>` | Clone-on-write: borrow if possible, allocate if needed. |
| smart pointer | Struct that wraps a value and adds behavior (Box, Rc, Arc, ...). |
| ownership tree | Conceptual tree of who owns what; drop walks it. |
| error type | A type that represents what went wrong. |
| `Result<T,E>` | Either an `Ok(T)` or an `Err(E)`. |
| `Option<T>` | Either `Some(T)` or `None`. |
| `?` operator | "Unwrap Ok or return Err" shortcut. |
| panic | A runtime failure that unwinds (or aborts) the thread. |
| panic_unwind | Default panic strategy: stack unwinds, drops run. |
| panic_abort | Alternate strategy: program aborts immediately. |
| std::panic | Module to control panicking. |
| catch_unwind | Catches a panic in a closure (rare). |
| anyhow | Crate for application error handling. |
| thiserror | Crate for library error enums. |
| eyre | Like anyhow with extras. |
| color-eyre | eyre with pretty colors. |
| miette | Pretty diagnostics for libraries and CLIs. |
| async | Keyword for asynchronous functions/blocks. |
| await | Keyword for polling a Future to completion. |
| Future | A value that will produce a result when polled. |
| `Pin<P>` | Wrapper saying "the pointee will not move." |
| Unpin | Marker that says it's safe to move out of Pin. |
| executor | The runtime that polls Futures. |
| runtime | Same as executor (informal). |
| tokio | The dominant async runtime. |
| smol | A small alternative async runtime. |
| async-std | Another async runtime mirroring std. |
| futures crate | Async traits and combinators outside std. |
| mpsc | Multi-producer, single-consumer channel. |
| oneshot | Single-shot channel; one send, one receive. |
| broadcast | Each receiver gets every message. |
| watch | A latest-value-only channel. |
| select! | Wait on whichever Future finishes first. |
| join! | Wait for all Futures. |
| try_join! | Wait for all, short-circuit on Err. |
| FuturesUnordered | Stream of Futures that yields whichever is ready. |
| FuturesOrdered | Stream that yields in the order they were inserted. |
| stream | Async iterator: produces values over time. |
| AsyncRead | Async version of `Read`. |
| AsyncWrite | Async version of `Write`. |
| tokio::spawn | Spawn a Future as a task on the runtime. |
| task::JoinHandle | Handle to await a spawned task's result. |
| structured concurrency | Spawn-and-await pattern keeping tasks scoped. |
| tokio::select! | Tokio's flavor of select. |
| scoped tasks | Tasks whose lifetime is bounded by their parent's scope. |
| AbortHandle | Handle that can cancel a spawned task. |
| Notify | One-shot signal between async tasks. |
| Semaphore | Counting limit on concurrent access. |
| Barrier | Sync point that releases when N tasks arrive. |
| Once | One-time initialization. |
| OnceCell | Cell that initializes once. |
| OnceLock | Thread-safe OnceCell in std. |
| lazy_static | Old crate for lazy globals; replaced by once_cell. |
| once_cell | Crate providing OnceCell types. |
| atomic ordering | Memory ordering for atomic ops: Relaxed/Acquire/Release/AcqRel/SeqCst. |
| MaybeUninit | Wrapper for memory that may not be initialized yet. |
| `MaybeUninit::uninit` | Make an uninitialized buffer. |
| `MaybeUninit::write` | Write a value into an uninitialized slot. |
| std::ptr | Module of raw pointer utilities. |
| raw pointer | `*const T` or `*mut T`; no lifetime, no safety, used in unsafe. |
| `*const T` | Immutable raw pointer. |
| `*mut T` | Mutable raw pointer. |
| `NonNull<T>` | Raw pointer that is statically known non-null. |
| std::mem | Module with `size_of`, `align_of`, `transmute`, `replace`, `swap`, `take`. |
| unsafe block | `unsafe { ... }` lets you do unsafe operations inside. |
| unsafe fn | Function whose caller must uphold extra invariants. |
| unsafe trait | A trait whose impls must uphold extra invariants. |
| unsafe impl | An impl that asserts those invariants. |
| miri | Interpreter that detects undefined behavior in unsafe code. |
| loom | Concurrency model checker for testing locks/atomics. |
| kani | Formal model checker for Rust. |
| prusti | Pre/postcondition verifier for Rust. |
| cargo install | Installs a crate's binary into `~/.cargo/bin`. |
| cargo doc | Builds HTML docs from `///` comments. |
| doctest | A test extracted from a doc comment code block. |
| std::process::Command | Spawn an external program. |
| std::env | Read environment variables and CLI args. |
| std::fs | Filesystem APIs. |
| std::io | I/O traits and types. |
| std::path | Path manipulation. |
| std::collections | Stdlib collections (Vec, HashMap, etc). |
| serde | Serialization framework. |
| serde_json | JSON via serde. |
| serde_derive | Derive macros for serde. |
| bincode | Compact binary format via serde. |
| postcard | No-std friendly binary format. |
| rmp-serde | MessagePack via serde. |
| hyper | Low-level HTTP. |
| reqwest | High-level HTTP client. |
| axum | Tokio-based web framework. |
| actix-web | Actor-style web framework. |
| warp | Filter-based web framework. |
| rocket | Macro-heavy web framework. |
| leptos | Full-stack web framework with reactivity. |
| dioxus | Cross-platform UI framework. |
| iced | Cross-platform GUI framework. |
| egui | Immediate-mode GUI. |
| gtk-rs | GTK bindings. |
| slint | Declarative UI toolkit. |
| tonic | gRPC framework on tokio. |
| prost | Protobuf code generation. |
| diesel | Compile-time-checked ORM. |
| sqlx | Async, compile-time-checked SQL. |
| sea-orm | Async ORM with relations. |
| tracing | Structured async-aware logging/tracing. |
| log | Lightweight logging facade. |
| env_logger | Simple logger for log crate. |
| pretty_env_logger | env_logger with colors. |
| tracing-subscriber | Subscriber/formatter for tracing. |
| criterion | Benchmark harness with statistics. |

## ASCII Diagrams

### Ownership tree

```
fn build_user_db() -> HashMap<u32, User> {
   HashMap                <- root, returned to caller (ownership moves out)
    ├── (1, User { ... })
    │       └── name: String       <- heap allocation A
    │       └── email: String      <- heap allocation B
    ├── (2, User { ... })
    │       └── name: String       <- heap allocation C
    │       └── email: String      <- heap allocation D
    └── (3, User { ... })
            └── name: String       <- heap allocation E
            └── email: String      <- heap allocation F

When the returned HashMap drops, every node walks down and frees A..F.
No leaks. No double-frees. Compiler arranged it.
```

### Borrow checker decision flow

```
                    let r = &x;     OR     let r = &mut x;
                          │                       │
                          ▼                       ▼
            +------------------------+   +-------------------------+
            | exists active &mut x?  |   | exists active & x?      |
            +------+----------+------+   +-----+-------------+-----+
                  Y            N               Y              N
                  │            │               │              │
                  ▼            ▼               ▼              ▼
              [E0502]      OK, add r      [E0502/E0499]    is x mut?
                            to live                      Y          N
                            borrows                      │          │
                                                         ▼          ▼
                                                  add &mut r   [E0596]
                                                  exclusive
```

### Async Future state machine

```
async fn fetch_two() -> (String, String) {
    let a = fetch("a").await;     // suspend point 1
    let b = fetch("b").await;     // suspend point 2
    (a, b)
}

   START
     │
     ▼
   poll(fetch("a"))  ──── NotReady ────► return NotReady (parked)
     │       ▲                                  │
     │       └──────── awakened ────────────────┘
     ▼
   got a; save a in state
     │
     ▼
   poll(fetch("b"))  ──── NotReady ────► return NotReady (parked)
     │       ▲                                  │
     │       └──────── awakened ────────────────┘
     ▼
   got b; build (a, b) and return Ready((a, b))
```

### Pin vs Unpin

```
Box<MyFuture>         vs       Pin<Box<MyFuture>>
+---------------+              +----------------------+
| ptr -------+  |              | ptr -----+           |
+---------------+              +----------------------+
            │                            │
            ▼                            ▼
+----------------+             +-------------------+
|  state vars    |             |  state vars       |
|  internal      |             |  internal         |
|  references    |             |  references       |
| can be moved   |    ↔        |  CANNOT BE MOVED  |
| because nobody |             |  because Pin says |
| can promise    |             |  so, and Future   |
| references     |             |  contains self-   |
| stay valid     |             |  references       |
+----------------+             +-------------------+
```

### Atomic ordering memory model

```
Thread A:                Thread B:
  store(x, 1, Release) -----> load(x, Acquire)  // 1 visible? yes
                                  │
                                  ▼
                              load(y, Acquire)  // also sees what A did before x

Relaxed:  no ordering vs other ops, just atomicity
Acquire:  this load happens before all subsequent ops in this thread
Release:  this store happens after all preceding ops in this thread
AcqRel:   for read-modify-write — both ends
SeqCst:   single global order across all SeqCst ops everywhere

If unsure, use SeqCst. It's slow-ish but correct.
```

## Try This

If you have not installed Rust yet, the easiest path on macOS or Linux:

```bash
$ curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

That installs `rustup`, `rustc`, `cargo`. Restart your shell. Then:

```bash
$ rustc --version
$ cargo new ramp_up && cd ramp_up
$ cargo run
```

You just compiled and ran your first Rust program. Now do these in order:

1. Edit `src/main.rs` and add a function that takes a `String` and tries to use it twice. Watch the compiler refuse with `error[E0382]`. Read the message in full.
2. Add a function that returns `Result<i32, String>`. Use the `?` operator inside it.
3. Add `serde = { version = "1", features = ["derive"] }` and `serde_json = "1"` to `Cargo.toml`. Define a struct with `#[derive(Serialize, Deserialize)]` and serialize it to JSON.
4. Add `tokio = { version = "1", features = ["full"] }`. Make `main` `#[tokio::main] async fn`. Spawn three tasks that each sleep a different number of seconds and print their name.
5. Run `cargo clippy`. Fix every lint it complains about.
6. Run `cargo doc --open`. Browse your crate's docs. Add a `///` comment to your function and rebuild.
7. Add a `#[test] fn it_works() { ... }` and run `cargo test`.
8. Add `#[derive(Debug, Clone, PartialEq)]` to a struct and try to put it into a `HashMap` as a value. Now try as a key — see what error you get because `Hash` is missing.
9. Try `cargo install ripgrep`. Notice you just installed a real Rust binary the same way Rust devs install their tools.
10. Run `cargo audit` (after `cargo install cargo-audit`). See if any of your dependencies have advisories.

## Where to Go Next

When this sheet is comfortable, head for `languages/rust` for the full reference: every keyword, every standard library highlight, every gotcha. Then layer on:

- **`languages/c`** to see the world Rust replaced — pointers, malloc, undefined behavior. You will appreciate the borrow checker.
- **`languages/go`** for the other modern systems language; very different design (GC, simpler types).
- **`fundamentals/x86-64-assembly`** and **`fundamentals/arm64-architecture`** if you want to see what the compiler actually emits. `cargo asm` is great for this.
- **`fundamentals/risc-v`** if you want to play with embedded Rust on simple hardware.
- **`fundamentals/ebpf-bytecode`** for in-kernel programs; Rust has growing support.
- **`ramp-up/assembly-eli5`** if registers and instructions still feel mysterious.
- **`ramp-up/linux-kernel-eli5`** for the OS underneath everything we wrote.
- **`ramp-up/git-eli5`** so you can put your Rust code in version control.

## See Also

- `languages/rust`
- `languages/c`
- `languages/go`
- `fundamentals/x86-64-assembly`
- `fundamentals/arm64-architecture`
- `fundamentals/risc-v`
- `fundamentals/ebpf-bytecode`
- `ramp-up/assembly-eli5`
- `ramp-up/linux-kernel-eli5`
- `ramp-up/git-eli5`

## References

- The Rust Programming Language (free book): https://doc.rust-lang.org/book/
- Rust by Example: https://doc.rust-lang.org/rust-by-example/
- Rustlings (interactive exercises): https://github.com/rust-lang/rustlings
- Standard library docs: https://doc.rust-lang.org/std/
- The Rust Reference: https://doc.rust-lang.org/reference/
- The Rustonomicon (advanced unsafe): https://doc.rust-lang.org/nomicon/
- The Embedded Rust Book: https://docs.rust-embedded.org/book/
- "Programming Rust" — Blandy, Orendorff, Tindall (O'Reilly).
- "Rust for Rustaceans" — Jon Gjengset (No Starch Press).
- "Rust Atomics and Locks" — Mara Bos (O'Reilly, free online).
- crates.io / lib.rs — ecosystem search.
- rust-lang.org/learn — official starting point.

## Version Notes

- **Rust 1.0** — May 15, 2015. First stable release; ownership/borrowing/lifetimes all present.
- **Edition 2015** — the original edition; default for crates without an `edition` line.
- **Edition 2018** — December 2018. Module system clean-up; **Non-Lexical Lifetimes** (huge borrow-checker win); `?` for `Option`; first stable async/await syntax preview.
- **Edition 2021** — October 2021. Disjoint capture in closures (`move` borrows only the fields it uses); `IntoIterator` for arrays (`for x in [1,2,3]`); `or` patterns in macros.
- **Edition 2024** — early 2025. Lazy `LazyCell`/`LazyLock`; `let-else` mainstream; expanding `if let` chains; `gen` blocks (generators) coming behind unstable feature flag.
- **Async/await** — stabilized Rust 1.39 (November 2019).
- **`const fn`** — gradually expanded since 1.31; const traits and const generics keep landing release after release.
- **GATs (generic associated types)** — stabilized 1.65 (November 2022).
- **Linux kernel Rust support** — landed 6.1 (December 2022); growing every release.
- The compiler ships every six weeks. Run `rustup update` from time to time. Pin `rust-toolchain.toml` in your repo to keep CI and devs aligned.
