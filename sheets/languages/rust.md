# Rust (Programming Language)

Systems language with ownership-based memory safety, zero-cost abstractions, fearless concurrency, and no garbage collector — the borrow checker is your linter, your reviewer, and your test suite.

## Setup

### Install via rustup

```bash
# macOS/Linux:  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
# Windows:      download rustup-init.exe from https://rustup.rs
# Verify:       rustc --version            # rustc 1.83.0 (90b35a623 2024-11-26)
#               cargo --version            # cargo 1.83.0 (5ffbef321 2024-10-29)
# Update:       rustup update              # pulls latest stable
# Multiple toolchains:
#   rustup toolchain install nightly       # nightly for unstable features
#   rustup toolchain install 1.75.0        # pin a specific version
#   rustup default stable                  # set default toolchain
#   rustup show                            # show installed toolchains
```

### Cargo — the only build tool you need

```bash
# cargo new myapp                          # creates ./myapp with Cargo.toml + src/main.rs
# cargo new --lib mylib                    # creates a library (src/lib.rs)
# cargo init                               # initialize a project in cwd (no new dir)
# cargo build                              # debug build → target/debug/myapp
# cargo build --release                    # optimized → target/release/myapp
# cargo run -- arg1 arg2                   # build + run with args after --
# cargo check                              # type-check without producing a binary (FAST)
# cargo clean                              # delete target/
```

### Edition: 2021 vs 2024

```bash
# Editions are opt-in syntax/lint changes — NOT a backward-incompatible language fork.
# A 2024 crate can depend on a 2018 crate; the compiler handles both at once.
# Cargo.toml:
#     [package]
#     name    = "myapp"
#     version = "0.1.0"
#     edition = "2024"                     # current default for new projects
# Notable edition features:
#     2018: module path system overhaul, async/await reservation
#     2021: closure capture disjointness, IntoIterator for arrays, panic ABI
#     2024: never type stabilization paths, lifetime captures (impl Trait), gen blocks
# Migrate:  cargo fix --edition  --edition-idioms
```

### MSRV (Minimum Supported Rust Version)

```bash
# Cargo.toml:
#     [package]
#     rust-version = "1.75"                # cargo will refuse to compile on older toolchains
# This is library hygiene: pick the oldest Rust you genuinely test against in CI.
# Tools:
#     cargo install cargo-msrv             # find the actual minimum
#     cargo msrv find                      # bisects via toolchain installs
# Rule of thumb: stable - 6 months for libraries; latest stable for apps.
```

### Editor & tooling

```bash
# rust-analyzer    — official LSP (autocomplete, goto-def, type hover, inlay hints)
#                    rustup component add rust-analyzer    OR install via your editor
# rustfmt          — formatter; rustup component add rustfmt; cargo fmt
# clippy           — linter (~700 lints); rustup component add clippy; cargo clippy
# cargo-edit       — cargo add / cargo rm dependencies (now built into cargo 1.62+)
# cargo-watch      — auto-rebuild on file change: cargo install cargo-watch; cargo watch -x run
# cargo-nextest    — much faster, better-output test runner
# cargo-expand     — show macro expansions: cargo install cargo-expand; cargo expand
# cargo-asm        — view emitted assembly for a function
```

## Variables, Mutability, Shadowing

### let — immutable by default

```bash
# let x = 5;                               // x is i32 by inference
# let y: i64 = 10;                         // explicit annotation
# x = 6;                                   // ERROR: cannot assign twice to immutable variable
# let mut z = 0;                           // mut → reassignable
# z = 99;                                  // OK
# const MAX: u32 = 1000;                   // compile-time constant; type required
# static GREETING: &str = "hello";         // 'static lifetime, single instance
```

### Shadowing (re-binding)

```bash
# // Shadowing creates a NEW binding; the old one drops at the end of its scope.
# let x = 5;
# let x = x + 1;                           // x is now 6 (immutable, fresh binding)
# let x = "six";                           // type can change — this is shadowing, not mutation
# // Useful for type conversions you only need once:
# let spaces = "   ";                      // &str
# let spaces = spaces.len();               // now usize
# // Mutation does NOT allow type change:
# let mut s = "hi"; s = 42;                // ERROR: expected &str, found integer
```

### const vs static

```bash
# const  — value inlined at every use site; type required; only basic exprs allowed.
# static — single memory location; 'static lifetime; can be `static mut` (unsafe to access).
# const PI: f64 = 3.14159;
# static VERSION: &str = env!("CARGO_PKG_VERSION");   // computed at compile time
# // Use const for true compile-time literals; use static when you need a single address
# // (e.g., for FFI, large lookup tables, or `lazy_static!`/`once_cell` cells).
```

## Primitive Types

### Integer family

```bash
# Signed:    i8 i16 i32 i64 i128 isize     // isize is pointer-width (32 or 64-bit)
# Unsigned:  u8 u16 u32 u64 u128 usize     // usize is pointer-width — used for indexing
# Defaults:  integer literal → i32 unless context forces otherwise
#            float literal   → f64
# Suffix literals: 42i64    1_000_000u32    0xff_u8    0b1010_u16    0o755_u32
# Underscores for readability:  1_000_000  ==  1000000
# // Conversions are EXPLICIT — there is no implicit narrowing or widening.
# let a: i32 = 100;  let b: i64 = a as i64;     // widening cast
# let c: u8  = 257 as u8;                       // wrap → 1 (truncation)
# let d: u8 = i32::try_into(a).unwrap();        // checked conversion, returns Result
```

### Floating point

```bash
# f32, f64                                 // f64 is the default literal type
# let x: f32 = 1.5_f32;
# let y = 2.5;                             // y: f64
# // No implicit conversion between f32 and f64 — use `as`.
# // f64::INFINITY, f64::NAN, f64::EPSILON, f64::MAX, f64::MIN_POSITIVE
# // NaN never equals itself: f64::NAN == f64::NAN  →  false
```

### bool

```bash
# let t: bool = true;
# let f: bool = false;
# // No truthy coercion — `if 1` is a compile error. Use comparisons.
# // Booleans are 1 byte. They are NOT integers; you must cast: (b as u8).
```

### char — 4-byte Unicode scalar value

```bash
# let c: char = 'A';
# let heart: char = '❤';
# let emoji: char = '\u{1F600}';           // 4 bytes, holds any Unicode scalar (not surrogates)
# // A Rust char is NOT a byte. Use u8 (or b'A') for ASCII byte literals.
# let b: u8 = b'A';                        // 65
# // String iteration:
# for c in "héllo".chars() { println!("{c}"); }   // h, é, l, l, o
# // For raw bytes:
# for b in "héllo".bytes() { println!("{b:#x}"); } // 0x68 0xc3 0xa9 0x6c 0x6c 0x6f
```

### Unit type and never type

```bash
# ()           — the unit type, single value `()`. Functions returning nothing return ().
# !            — the never type, no values exist. Functions like panic! return !.
# fn forever() -> ! { loop {} }             // never returns
# fn doom() -> ! { panic!("kaput") }
# // ! coerces to any type — that's why `panic!()` is valid in any expression position.
```

## Numeric Overflow

### Default behavior — debug vs release

```bash
# Debug builds:    arithmetic OVERFLOW PANICS at runtime (overflow checks on).
# Release builds:  arithmetic WRAPS around silently (two's complement).
# let x: u8 = 255;
# let y = x + 1;          // debug: panic!("attempt to add with overflow")
#                          // release: y == 0  (silent wrap — possible bug source)
# // Always test in release mode before shipping numeric code.
```

### Explicit overflow methods (use these in production)

```bash
# checked_*    — returns Option<T>: None on overflow
# wrapping_*   — wraps explicitly (no UB, no panic)
# saturating_* — clamps to MIN/MAX
# overflowing_* — returns (T, bool) — value plus did_overflow flag
#
# let x: u8 = 250;
# x.checked_add(10)                        // None  (would overflow)
# x.wrapping_add(10)                       // 4     (250 + 10 = 260, wraps mod 256)
# x.saturating_add(10)                     // 255   (clamped at u8::MAX)
# x.overflowing_add(10)                    // (4, true)
#
# // Same for sub, mul, div, neg, pow, shl, shr.
# // Always pick one explicitly when the math is at the edge of the type's range.
```

### Strict mode (1.79+ stable)

```bash
# // strict_* methods always panic on overflow regardless of profile:
# let x: i32 = i32::MAX;
# // x.strict_add(1)  →  panics in debug AND release
# // Use when a wrong answer is worse than crashing (security, accounting).
```

## Tuples

### Construction and destructuring

```bash
# let t: (i32, f64, &str) = (42, 3.14, "hi");
# let (a, b, c) = t;                       // destructure
# let first = t.0;                         // index access
# let second = t.1;
# let third = t.2;
# // Empty tuple is the unit type:
# let unit: () = ();
# // Nested:
# let nested = ((1, 2), (3, 4));
# let ((a, b), (c, d)) = nested;
```

### Tuples vs structs

```bash
# // Tuples are anonymous — fine for short return values; bad for public APIs.
# fn min_max(xs: &[i32]) -> (i32, i32) { ... }
# // Prefer named structs once you have 3+ fields or any ambiguity:
# struct Range { min: i32, max: i32 }
```

## Arrays vs Slices vs Vec

### Array — fixed size, on stack

```bash
# let a: [i32; 5] = [1, 2, 3, 4, 5];        // length is part of the type
# let zeros = [0u8; 32];                   // [T; N] init shorthand
# a.len()                                   // 5 — known at compile time
# a[2]                                      // bounds-checked; panics on out-of-range
# // Arrays are Copy if T: Copy. They live on the stack.
# // [i32; 5] and [i32; 6] are DISTINCT types — generic over const N for variable lengths.
```

### Slice — `&[T]` and `&mut [T]`

```bash
# let a = [1, 2, 3, 4, 5];
# let s: &[i32] = &a[1..4];                // borrowed view: [2, 3, 4]
# s.len(); s.iter(); s.first(); s.last();
# s.split_at(2)                            // (&[2], &[3, 4])
# // Slices are fat pointers: (data_ptr, length). They borrow from the source.
# fn sum(xs: &[i32]) -> i32 { xs.iter().sum() }   // accepts arrays, Vecs, sub-slices
```

### Vec — heap-allocated, growable

```bash
# let mut v: Vec<i32> = Vec::new();
# let v = vec![1, 2, 3];                   // macro shorthand
# let v = Vec::with_capacity(100);         // pre-allocate
# v.push(4); v.pop();                       // O(1) amortized
# v.len(); v.capacity();
# v.insert(0, 99); v.remove(0);             // O(n) — shifts elements
# v.extend([7, 8, 9]);                      // append from iterator
# v.iter().sum::<i32>();
# // &Vec<T> coerces to &[T] automatically; prefer &[T] in function signatures.
# fn process(xs: &[i32]) { ... }            // works with both arrays AND Vecs
```

### When to use which

```bash
# Array  [T; N]:   small, fixed-size, no heap, e.g., [u8; 32] for hashes, RGB pixel.
# Slice  &[T]:    function parameter for "any contiguous sequence of T".
# Vec    Vec<T>:   owned, growable storage; the workhorse heap container.
# // Box<[T]> is a heap slice without growth capacity — fixed after construction.
```

## Strings

### `&str` and `String`

```bash
# &str            — borrowed string slice; fat pointer (ptr, len); UTF-8.
# String          — owned, growable, heap-allocated UTF-8 buffer.
# let lit: &str = "hello world";           // string literal: &'static str baked into binary
# let owned: String = String::from("hi");
# let owned: String = "hi".to_string();
# let owned: String = "hi".to_owned();
# let borrowed: &str = &owned;              // Deref coercion → &str view of String
# let borrowed: &str = owned.as_str();      // explicit form
# // Function param convention: take &str unless you NEED ownership.
# fn greet(name: &str) -> String { format!("hello {name}") }
```

### Indexing rules — there is no `s[0]`

```bash
# let s = "héllo";
# // s[0]  → COMPILE ERROR. Strings are UTF-8 byte sequences; bytes ≠ characters.
# s.len()                                   // 6 — byte length, NOT char count
# s.chars().count()                         // 5 — Unicode scalar count (slower)
# s.chars().nth(2)                          // Some('l') — O(n)
# s.as_bytes()[0]                           // 104 (the byte 'h')
# &s[0..1]                                  // "h" — slice by BYTE indices; PANICS if it splits a codepoint
# &s[1..3]                                  // "é" — bytes 1..=2 form é
# &s[1..2]                                  // PANIC: byte index 2 is not a char boundary
```

### Building strings

```bash
# let mut s = String::new();
# s.push('a');                              // single char
# s.push_str("bc");                         // append &str
# s += "de";                                // also push_str
# let f = format!("{}-{}", "x", 42);        // returns String — most common
# // Avoid `s = s + "..."` in loops — repeated reallocation. Use String::with_capacity + push_str.
# let mut buf = String::with_capacity(1024);
# for chunk in chunks { buf.push_str(chunk); }
```

### `OsString` / `OsStr` — paths and arguments

```bash
# // OS strings can contain bytes that are not valid UTF-8 (Unix paths) or UTF-16 (Windows paths).
# use std::ffi::{OsStr, OsString};
# let arg: OsString = std::env::args_os().nth(1).unwrap();
# // Convert to str only if you can prove it's UTF-8:
# arg.to_str()                              // Option<&str>
# arg.to_string_lossy()                     // Cow<str> — replaces invalid bytes with U+FFFD
```

### `CString` / `CStr` — null-terminated FFI strings

```bash
# use std::ffi::{CStr, CString};
# let c = CString::new("hello").unwrap();   // appends \0; errors if input contains \0
# let p: *const i8 = c.as_ptr();            // pass to extern "C" fn taking const char*
# // Receiving a C string:
# let cstr = unsafe { CStr::from_ptr(ptr) };
# let s: &str = cstr.to_str().unwrap();     // validated UTF-8
```

### `Path` and `PathBuf`

```bash
# use std::path::{Path, PathBuf};
# let p: &Path = Path::new("/etc/hosts");
# let mut owned: PathBuf = PathBuf::from("/etc");
# owned.push("hosts");                      // "/etc/hosts"
# p.extension()                             // Option<&OsStr>
# p.file_name()                             // Option<&OsStr> — "hosts"
# p.parent()                                // Option<&Path>  — "/etc"
# p.is_file(); p.is_dir(); p.exists();
# p.to_str()                                // Option<&str> — None on non-UTF-8
# // Don't concatenate paths with format! — use .join() to handle separators.
```

## Collections

### `HashMap` and `BTreeMap`

```bash
# use std::collections::HashMap;
# let mut m: HashMap<String, i32> = HashMap::new();
# m.insert("a".into(), 1);
# m.get("a")                                // Option<&i32>
# m.contains_key("a")
# m.entry("b".into()).or_insert(0) += 1;    // upsert pattern
# for (k, v) in &m { ... }                  // iteration order is RANDOM (DoS-resistant)
#
# use std::collections::BTreeMap;
# let mut bm: BTreeMap<&str, i32> = BTreeMap::new();
# // BTreeMap iterates in SORTED key order. Slightly slower than HashMap on hash hits.
# // Use BTreeMap when you need ordered iteration or range queries: bm.range("a".."m")
```

### `HashSet` and `BTreeSet`

```bash
# use std::collections::{HashSet, BTreeSet};
# let mut s: HashSet<i32> = HashSet::new();
# s.insert(1); s.insert(2); s.insert(1);    // duplicate ignored
# s.contains(&1);
# let a: HashSet<_> = [1,2,3].into();
# let b: HashSet<_> = [2,3,4].into();
# a.intersection(&b).copied().collect::<Vec<_>>();   // [2, 3]
# a.union(&b); a.difference(&b); a.symmetric_difference(&b);
```

### `VecDeque` — double-ended queue

```bash
# use std::collections::VecDeque;
# let mut q: VecDeque<i32> = VecDeque::new();
# q.push_back(1); q.push_back(2);
# q.push_front(0);
# q.pop_front();                            // Some(0) — O(1)
# q.pop_back();                             // Some(2) — O(1)
# // Use for FIFO queues, BFS, sliding windows. Backed by a ring buffer.
```

### `BinaryHeap` — max-heap priority queue

```bash
# use std::collections::BinaryHeap;
# let mut h = BinaryHeap::new();
# h.push(3); h.push(1); h.push(2);
# h.peek();                                 // Some(&3) — max
# h.pop();                                  // Some(3)
# // For min-heap, wrap with std::cmp::Reverse:
# use std::cmp::Reverse;
# let mut min_h = BinaryHeap::new();
# min_h.push(Reverse(3)); min_h.push(Reverse(1));
# min_h.peek();                             // Some(&Reverse(1))
```

## Option<T>

### Construction and consumption

```bash
# let x: Option<i32> = Some(42);
# let y: Option<i32> = None;
# // Pattern matching:
# match x {
#     Some(v) => println!("{v}"),
#     None    => println!("nothing"),
# }
# if let Some(v) = x { println!("{v}"); }
# while let Some(v) = iter.next() { ... }
# // Conversions / shortcuts:
# x.is_some(); x.is_none();
# x.unwrap();                               // panics on None — use only when impossible
# x.expect("must have a value");            // panics with custom message
# x.unwrap_or(0);                           // default
# x.unwrap_or_else(|| compute());           // lazy default
# x.unwrap_or_default();                    // T::default()
```

### Combinators

```bash
# x.map(|v| v + 1);                         // Option<i32> → Option<i32>
# x.and_then(|v| if v > 0 { Some(v) } else { None });   // monadic bind
# x.or(Some(0));                            // first Some wins
# x.filter(|&v| v > 10);
# x.as_ref();                               // Option<&T>  (don't move the inner value)
# x.as_mut();                               // Option<&mut T>
# x.take();                                 // replaces with None, returns old
# x.replace(99);                            // returns old, sets new
# x.zip(y);                                 // Option<(T, U)> — both Some or None
```

### The `?` operator on Option

```bash
# // ? on Option short-circuits with None (only valid in fns returning Option).
# fn first_word(s: &str) -> Option<&str> {
#     let bytes = s.as_bytes();
#     let space = bytes.iter().position(|&b| b == b' ')?;   // None propagates
#     Some(&s[..space])
# }
# // To bridge between Option and Result use .ok_or() / .ok_or_else():
# fn parse(s: &str) -> Result<i32, String> {
#     let n = s.parse::<i32>().ok().ok_or("not a number".to_string())?;
#     Ok(n)
# }
```

## Result<T, E>

### Construction and pattern

```bash
# fn divide(a: f64, b: f64) -> Result<f64, String> {
#     if b == 0.0 { return Err("div by zero".into()); }
#     Ok(a / b)
# }
# match divide(10.0, 2.0) {
#     Ok(v)  => println!("{v}"),
#     Err(e) => eprintln!("error: {e}"),
# }
# // Same combinators as Option:
# r.is_ok(); r.is_err(); r.unwrap(); r.unwrap_or(0.0);
# r.unwrap_or_else(|_| 0.0); r.expect("expected ok");
# r.map(|v| v * 2);                         // Result<U, E>
# r.map_err(|e| MyErr::from(e));            // Result<T, F>
# r.and_then(|v| more_work(v));             // chain
# r.ok();                                   // Option<T>
# r.err();                                  // Option<E>
```

### The `?` operator + From conversion

```bash
# // ? unwraps Ok, returns Err (after From conversion) from the enclosing function.
# fn read_config() -> Result<Config, MyError> {
#     let data = std::fs::read_to_string("config.json")?;   // io::Error → MyError via From
#     let cfg: Config = serde_json::from_str(&data)?;       // serde error → MyError via From
#     Ok(cfg)
# }
# // For ? to work: the enclosing fn returns Result<_, E> where E: From<EachInnerError>.
# // The thiserror / anyhow crates make this nearly automatic.
```

### main can return Result

```bash
# fn main() -> Result<(), Box<dyn std::error::Error>> {
#     let data = std::fs::read_to_string("input.txt")?;
#     println!("{data}");
#     Ok(())
# }
# // Process exits 0 on Ok, 1 on Err (with Debug-printed error to stderr).
```

## Control Flow

### if as expression

```bash
# let x = 5;
# // if-else is an expression — no ternary needed.
# let label = if x > 0 { "positive" } else if x < 0 { "negative" } else { "zero" };
# // Both branches must produce the SAME TYPE.
# let y = if cond { 1 } else { 2 };          // OK — both i32
# // let y = if cond { 1 } else { "two" };   // ERROR: incompatible arms
# // No parentheses required. Braces are mandatory.
```

### match — exhaustive by default

```bash
# // The compiler enforces total coverage. Every variant must be handled.
# match dir {
#     Direction::Up    => 0,
#     Direction::Down  => 1,
#     Direction::Left  => 2,
#     Direction::Right => 3,
# }
# // Adding a new variant elsewhere causes a compile error here — you cannot forget.
# // Use _ as a catch-all:
# match n {
#     0 => "zero",
#     1 | 2 | 3 => "small",
#     _ => "other",
# }
```

## Match guards, bindings, or-patterns

### Guards (`if`)

```bash
# match pt {
#     (0, 0)            => "origin",
#     (x, _) if x > 0   => "right half",
#     (x, _) if x < 0   => "left half",
#     _                 => "axis",
# }
# // Guard expressions can reference matched bindings.
```

### `@` bindings

```bash
# match age {
#     n @ 0..=12  => println!("kid {n}"),
#     n @ 13..=19 => println!("teen {n}"),
#     n           => println!("adult {n}"),
# }
# // Binds the matched value to a name AND requires it match the pattern.
```

### Or-patterns

```bash
# match n {
#     0 | 1 | 2 => "small",
#     3..=10    => "medium",
#     _         => "big",
# }
# // 1.53+: or-patterns can appear in any position:
# if let Some(0 | 1 | 2) = x { ... }
```

## Loops

### `loop` — returns a value via `break`

```bash
# // The only loop in Rust that can produce a value.
# let result = loop {
#     let n = next();
#     if n.is_done() { break n.value(); }
# };
# // result has the type of the break expression. Forever-loops have type !.
```

### `while` and `while let`

```bash
# while x > 0 { x -= 1; }
# // while let: loop until pattern fails:
# while let Some(top) = stack.pop() { use_it(top); }
```

### `for` — over iterators

```bash
# for i in 0..10        { ... }              // 0..10  is Range<i32>; 0..=10 is inclusive
# for x in &v           { ... }              // &x: borrow each element
# for x in &mut v       { *x += 1; }        // mutable borrow
# for x in v            { ... }              // CONSUMES v (moves elements out)
# for (i, v) in xs.iter().enumerate() { ... } // (index, &element)
# for (k, v) in &map     { ... }
# for line in std::io::stdin().lock().lines() { let line = line?; ... }
```

### Labels for nested loops

```bash
# 'outer: for i in 0..n {
#     for j in 0..m {
#         if grid[i][j] == target {
#             break 'outer;                  // breaks outer loop
#         }
#         if skip(i, j) { continue 'outer; } // continues outer loop
#     }
# }
# // Label is `'name:` before the loop, `break 'name` or `continue 'name` jumps to it.
```

## Functions

### Definition and call

```bash
# fn add(a: i32, b: i32) -> i32 { a + b }   // last expression is the return (no semicolon)
# fn greet(name: &str) -> String { format!("hi {name}") }
# fn noop() {}                              // returns () implicitly
# // Type inference does NOT cross function boundaries — signatures are explicit.
# add(1, 2);
```

### Early return

```bash
# fn first_positive(xs: &[i32]) -> Option<i32> {
#     for &x in xs {
#         if x > 0 { return Some(x); }      // explicit return + semicolon
#     }
#     None                                  // fallthrough — no semicolon, this IS the return
# }
```

### Divergent functions (`-> !`)

```bash
# fn doom(msg: &str) -> ! {
#     eprintln!("{msg}");
#     std::process::exit(1);                 // never returns; type is !
# }
# // ! coerces to any type, so:
# let x: i32 = if cond { 42 } else { doom("required") };
```

## Closures

### Syntax and inference

```bash
# let add = |a, b| a + b;                   // types inferred from use site
# let add: fn(i32, i32) -> i32 = |a, b| a + b;     // function pointer (no captures)
# add(2, 3);                                 // 5
# // Block body:
# let f = |x: i32| -> i32 {
#     let y = x * 2;
#     y + 1
# };
```

### `Fn`, `FnMut`, `FnOnce`

```bash
# // Fn      — read-only capture; can call repeatedly through &.
# // FnMut   — mutable capture; needs &mut access; callable multiple times.
# // FnOnce  — consumes captures; callable at most once.
# // The compiler picks the most permissive automatically. To force one:
# fn run<F: Fn(i32) -> i32>(f: F)    -> i32 { f(1) + f(2) }
# fn run<F: FnMut(i32)>     (mut f: F)        { f(1); f(2); }
# fn run<F: FnOnce() -> Vec<i32>>(f: F) -> Vec<i32> { f() }
# // Hierarchy: Fn ⊂ FnMut ⊂ FnOnce. A Fn satisfies an FnMut bound.
```

### `move` — transfer ownership of captures

```bash
# // Default capture is BY REFERENCE. `move` forces by-value (ownership transfers in).
# let data = vec![1, 2, 3];
# let task = move || println!("{:?}", data);   // data moves into the closure
# std::thread::spawn(task);                    // closure can outlive caller's frame
# // Without `move`, spawning a thread that captures `&data` is a compile error
# // because the thread might outlive `data`.
```

## Ownership

### The three rules

```bash
# 1. Each value has a single owner.
# 2. There is exactly one owner at a time.
# 3. When the owner goes out of scope, the value is dropped (memory freed).
```

### Move semantics

```bash
# let s1 = String::from("hi");
# let s2 = s1;                              // MOVE — s1 is no longer valid
# // println!("{s1}");                      // ERROR: borrow of moved value
# let s3 = s2.clone();                      // explicit deep copy — both alive
# // Function calls also move (or borrow):
# fn take(s: String) { /* drops s here */ }
# take(s2);                                 // s2 moved into take, gone after
```

### Copy vs Clone

```bash
# // Copy: implicit bitwise duplicate. For types where it's cheap and safe.
# // Implemented for: all integers, floats, bool, char, fixed-size arrays of Copy, tuples of Copy, &T.
# // Clone: explicit (.clone()), can do arbitrary work (deep copy heap memory).
# let x: i32 = 5;
# let y = x;                                // both alive — i32 is Copy
# let s: String = "hi".into();
# let t = s.clone();                        // explicit; both alive
# // Implementing Copy requires Clone (Copy is a marker trait that implies trivial Clone).
# #[derive(Copy, Clone, Debug)]
# struct Point { x: i32, y: i32 }
```

## Borrowing

### Shared (`&T`) vs exclusive (`&mut T`)

```bash
# fn read(s: &String) -> usize { s.len() }
# fn write(s: &mut String) { s.push('!'); }
# let mut s = String::from("hi");
# read(&s);                                 // immutable borrow
# write(&mut s);                            // mutable borrow
```

### The borrowing rules

```bash
# // At any point in time, you can have EITHER:
# //   • any number of shared (&T) borrows  OR
# //   • exactly one exclusive (&mut T) borrow.
# // No two mutable borrows. No mutable plus shared.
#
# let mut v = vec![1, 2, 3];
# let r1 = &v;                              // ok
# let r2 = &v;                              // ok — multiple shared
# // let m  = &mut v;                       // ERROR: cannot borrow `v` as mut while shared exists
#
# let m = &mut v;
# // let s = &v;                            // ERROR: cannot have shared while mut exists
# m.push(4);                                // ok via the mut borrow
```

### Why these rules

```bash
# // Prevents data races at compile time. Prevents iterator invalidation.
# // The mutex you'd add in C++/Java is unnecessary because the type system already enforces
# // the invariant: at most one writer OR many readers.
```

## Lifetimes

### Annotations

```bash
# // 'a is a lifetime parameter — a scope. Annotations describe RELATIONSHIPS between scopes,
# // they do not create scopes.
# fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
#     if x.len() > y.len() { x } else { y }
# }
# // The compiler can verify: the returned &str does not outlive 'a — the shorter of the inputs.
```

### Elision rules (when you can omit annotations)

```bash
# // 1. Each input ref gets its own lifetime parameter.
# // 2. If exactly one input lifetime, it's assigned to all output refs.
# // 3. If &self or &mut self, that lifetime is assigned to outputs.
# // Hence:
# fn first(s: &str) -> &str { ... }                 // unambiguous — rule 2
# fn name(&self) -> &str    { &self.name }          // unambiguous — rule 3
# fn pair(a: &str, b: &str) -> &str                  // AMBIGUOUS — must annotate
```

### `'static` — lives for the whole program

```bash
# let s: &'static str = "embedded in binary";   // string literals are 'static
# // 'static can also mean "owned data with no borrows" (Box<T>, Vec<T>, etc.).
# fn make() -> Box<dyn Fn() -> i32 + 'static> { Box::new(|| 42) }
```

### Structs holding references

```bash
# struct Reader<'a> {
#     buffer: &'a [u8],
#     pos: usize,
# }
# // The struct cannot outlive 'a — that's enforced by the compiler.
# impl<'a> Reader<'a> {
#     fn peek(&self) -> u8 { self.buffer[self.pos] }
# }
```

## Structs

### Three flavors

```bash
# // Named-field struct
# struct User {
#     name: String,
#     email: String,
#     active: bool,
# }
#
# // Tuple struct (positional)
# struct Point(f64, f64);
# struct Wrapper(String);                   // common newtype pattern
#
# // Unit struct (no fields)
# struct Marker;
```

### Construction and update syntax

```bash
# let u1 = User { name: "Ada".into(), email: "ada@x".into(), active: true };
# // Struct update syntax — fill remaining fields from another instance:
# let u2 = User { name: "Bob".into(), ..u1 };
# // u1.email is MOVED into u2 (if not Copy). After this, u1 is partially moved.
```

### `impl` blocks and methods

```bash
# impl User {
#     fn new(name: &str, email: &str) -> Self {
#         Self { name: name.into(), email: email.into(), active: true }
#     }
#     fn deactivate(&mut self) { self.active = false; }
#     fn domain(&self) -> Option<&str> { self.email.split('@').nth(1) }
# }
# // Self == User inside the impl block — handy for renaming and generics.
```

### `Default` and `derive`

```bash
# #[derive(Default, Debug, Clone)]
# struct Config {
#     host: String,                         // ""
#     port: u16,                            // 0
#     verbose: bool,                        // false
# }
# let c = Config::default();
# // Override one field while taking defaults for the rest:
# let c = Config { port: 8080, ..Default::default() };
```

## Enums

### Variants — sum types done right

```bash
# enum Shape {
#     Circle(f64),
#     Rectangle(f64, f64),
#     Triangle { base: f64, height: f64 },  // struct-like variant
#     Empty,                                 // no payload
# }
# // Each variant can carry different data — enum is a tagged union.
```

### Methods on enums

```bash
# impl Shape {
#     fn area(&self) -> f64 {
#         match self {
#             Shape::Circle(r) => std::f64::consts::PI * r * r,
#             Shape::Rectangle(w, h) => w * h,
#             Shape::Triangle { base, height } => 0.5 * base * height,
#             Shape::Empty => 0.0,
#         }
#     }
# }
```

### `Option`, `Result`, and friends are enums

```bash
# enum Option<T> { Some(T), None }
# enum Result<T, E> { Ok(T), Err(E) }
# // Custom error enums are the standard pattern (see Error Handling section).
```

## Pattern Matching

### In `let`, `if let`, `while let`

```bash
# // let with patterns:
# let (x, y) = (1, 2);
# let Point { x, y } = origin;
# let [first, .., last] = arr;              // slice pattern with ..
# let &val = reference;                     // dereference pattern
#
# // if let — match a single pattern, ignore others:
# if let Some(v) = result { use_it(v); }
# if let Ok(c) = parse(s) { ... }
#
# // let-else (1.65+) — destructure or diverge:
# let Some(v) = result else { return Err("missing".into()); };
# // v is in scope after this, fully unwrapped.
```

### Function arguments

```bash
# // Patterns also work in function parameters:
# fn print_pt(Point { x, y }: Point) { println!("({x}, {y})"); }
# fn first(&(a, _): &(i32, i32)) -> i32 { a }
```

## Traits

### Definition and impl

```bash
# trait Animal {
#     fn name(&self) -> &str;               // required method
#     fn legs(&self) -> u32 { 4 }           // default — override optional
#     fn describe(&self) -> String {
#         format!("{} ({} legs)", self.name(), self.legs())
#     }
# }
# struct Dog { name: String }
# impl Animal for Dog {
#     fn name(&self) -> &str { &self.name }
#     // Uses default legs() and describe()
# }
```

### Associated types

```bash
# trait Iterator {
#     type Item;
#     fn next(&mut self) -> Option<Self::Item>;
# }
# // Use associated types when each implementer has ONE specific Item type.
# // Use generics on the trait when an impl might exist for multiple types.
```

### Supertraits

```bash
# trait Animal: std::fmt::Debug {           // requires Debug
#     fn name(&self) -> &str;
# }
# // Anything implementing Animal must also implement Debug.
# // Multiple supertraits with `+`:
# trait Storable: Clone + Send + 'static { ... }
```

### `impl Trait` in argument and return position

```bash
# fn print_each(items: impl IntoIterator<Item = i32>) {
#     for x in items { println!("{x}"); }
# }
# fn make_counter() -> impl FnMut() -> u32 {
#     let mut n = 0;
#     move || { n += 1; n }
# }
# // impl Trait in args = generic shorthand. In return = "some single type implementing this trait".
```

## Trait Objects

### `dyn Trait`

```bash
# // Static dispatch (generics, monomorphized) vs dynamic dispatch (dyn, vtable):
# fn static_dispatch<T: Animal>(a: &T)    { a.name(); }    // resolved at compile time
# fn dynamic_dispatch(a: &dyn Animal)     { a.name(); }    // vtable lookup at runtime
# let zoo: Vec<Box<dyn Animal>> = vec![
#     Box::new(Dog { name: "Rex".into() }),
#     Box::new(Cat { name: "Mei".into() }),
# ];
# for a in &zoo { println!("{}", a.name()); }
# // Box<dyn Trait> = owned trait object; &dyn Trait = borrowed trait object.
```

### Object safety

```bash
# // A trait is object-safe iff every method has a Self that doesn't appear in:
# //   • return position (other than `Self` itself meaning the trait object)
# //   • generic parameters
# //   • a `where Self: Sized` clause
# // Examples of NOT object-safe:
# //   trait Clone { fn clone(&self) -> Self; }    // Self in return — not callable via vtable
# // Workaround: split into object-safe + extension trait, or use `&dyn` only for the safe parts.
```

## Derive Macros

### Built-in derives

```bash
# #[derive(Debug, Clone, PartialEq, Eq, Hash, Default)]
# struct Config {
#     host: String,
#     port: u16,
# }
# // Debug         — println!("{:?}", x) and {:#?} pretty form
# // Clone         — .clone()
# // Copy          — implicit duplicate (must also derive Clone, all fields Copy)
# // PartialEq/Eq  — == and !=  (Eq adds reflexivity for keys)
# // Hash          — usable as HashMap key (needs Eq)
# // Ord/PartialOrd — comparison ops
# // Default       — T::default()
```

### serde derives

```bash
# // In Cargo.toml:
# //   serde = { version = "1", features = ["derive"] }
# //   serde_json = "1"
# use serde::{Serialize, Deserialize};
# #[derive(Serialize, Deserialize, Debug)]
# struct User {
#     #[serde(rename = "userName")]
#     name: String,
#     #[serde(default)]
#     active: bool,
#     #[serde(skip_serializing_if = "Option::is_none")]
#     bio: Option<String>,
# }
# let u: User = serde_json::from_str(r#"{"userName":"Ada","active":true}"#).unwrap();
# let s = serde_json::to_string_pretty(&u).unwrap();
```

## Generics

### Type parameters

```bash
# fn largest<T: PartialOrd>(xs: &[T]) -> &T {
#     let mut max = &xs[0];
#     for x in &xs[1..] {
#         if x > max { max = x; }
#     }
#     max
# }
# // Bounds can be inline or in a where-clause for readability:
# fn long_signature<T, U>(a: T, b: U) -> T
# where
#     T: Clone + Default,
#     U: IntoIterator<Item = T>,
# { ... }
```

### Lifetime parameters

```bash
# fn pick_first<'a>(xs: &'a [i32]) -> &'a i32 { &xs[0] }
# struct Holder<'a, T> { inner: &'a T }
# // Lifetimes ALWAYS come before type parameters in the list.
```

### Const generics (1.51+)

```bash
# fn sum_array<const N: usize>(arr: [i32; N]) -> i32 {
#     arr.iter().sum()
# }
# sum_array([1, 2, 3]);                      // N inferred as 3
# // Useful for fixed-size buffer types and SIMD lanes.
# struct Matrix<const ROWS: usize, const COLS: usize> {
#     data: [[f64; COLS]; ROWS],
# }
```

### Trait bounds — multiple

```bash
# fn process<T: Clone + Send + 'static>(x: T) { ... }
# // Common idiom — the "where T: Send + Sync + 'static" thread-safe bound.
```

## Modules & Crates

### Module tree

```bash
# // src/main.rs
# mod config;                                // looks for src/config.rs OR src/config/mod.rs
# mod net {
#     pub mod tcp { pub fn listen() {} }    // inline submodule
# }
# fn main() {
#     net::tcp::listen();
#     config::load();
# }
```

### File layout

```bash
# // Two equivalent layouts for `mod foo`:
# //   src/foo.rs                                — single-file module
# //   src/foo/mod.rs                             — directory module (older style)
# //   src/foo.rs + src/foo/bar.rs                — modern: foo.rs declares `pub mod bar;`
# // Pick one style and stay consistent.
```

### Visibility

```bash
# pub             — public to anyone
# pub(crate)      — public within this crate only
# pub(super)      — public to the parent module
# pub(in path)    — public within a specific module path
# (no qualifier)  — private to the current module + descendants
# struct S { pub a: i32, b: i32 }            // a is public, b is private to the module
```

### `use` and re-exports

```bash
# use std::collections::HashMap;
# use std::collections::{HashMap, HashSet};
# use std::collections::HashMap as Map;
# pub use crate::internal::Thing;            // re-export for downstream users
# // Glob imports — usually only inside tests or prelude modules:
# use crate::prelude::*;
```

## Cargo

### Common commands

```bash
cargo new myapp                            # new binary crate
cargo new --lib mylib                      # new library
cargo build                                # debug build (target/debug)
cargo build --release                      # optimized (target/release)
cargo run -- arg1 arg2                     # build + run
cargo test                                 # run tests
cargo test --release                       # tests with optimizations
cargo test test_name                       # filter tests by name substring
cargo bench                                # nightly benchmarks
cargo doc --open                           # generate + open docs in browser
cargo check                                # type-check, no codegen (fastest)
cargo fmt                                  # format with rustfmt
cargo clippy                               # lint
cargo clippy -- -D warnings                # treat warnings as errors
cargo fix                                  # auto-apply compiler suggestions
cargo update                               # update Cargo.lock to latest semver
cargo tree                                 # dependency tree
cargo add serde --features derive          # add to Cargo.toml
cargo remove old-dep                       # remove from Cargo.toml
cargo install ripgrep                      # install a binary crate
```

### Cargo.toml

```bash
# [package]
# name    = "myapp"
# version = "0.1.0"
# edition = "2024"
# rust-version = "1.75"
#
# [dependencies]
# serde      = { version = "1", features = ["derive"] }
# tokio      = { version = "1", features = ["full"] }
# anyhow     = "1"
#
# [dev-dependencies]
# proptest   = "1"
#
# [build-dependencies]
# cc         = "1"
#
# [features]
# default = ["std"]
# std     = []
# alloc   = []
```

### Profiles

```bash
# [profile.release]
# opt-level     = 3                         # 0..=3, "s" (size), "z" (size+nopanic)
# lto           = "thin"                    # link-time optimization: false | true | "thin" | "fat"
# codegen-units = 1                         # fewer = better optimization, slower compile
# strip         = true                      # strip symbols
# panic         = "abort"                   # smaller binary, no unwind tables
#
# [profile.dev]
# opt-level = 0
# debug     = true
```

### Workspaces

```bash
# # workspace root Cargo.toml:
# [workspace]
# members  = ["crates/api", "crates/core", "crates/cli"]
# resolver = "2"
#
# [workspace.dependencies]
# serde = "1"                               # version once, use across members
#
# # crates/api/Cargo.toml:
# [dependencies]
# serde = { workspace = true }
```

## Macros

### Declarative — `macro_rules!`

```bash
# // Pattern-matching macros over token trees.
# macro_rules! min {
#     ($a:expr, $b:expr) => { if $a < $b { $a } else { $b } };
# }
# let x = min!(3, 5);                        // 3
#
# macro_rules! vec_of_squares {
#     ($($n:expr),*) => {
#         vec![$( $n * $n ),*]
#     };
# }
# vec_of_squares!(1, 2, 3);                  // vec![1, 4, 9]
# // Fragments: expr, ident, ty, pat, stmt, block, item, path, tt, literal, lifetime, vis, meta.
```

### Procedural — three flavors

```bash
# // Procedural macros run as Rust code at compile time. Defined in their own crate
# // with crate-type = ["proc-macro"].
#
# // 1. Derive macros:        #[derive(MyTrait)]
# // 2. Attribute macros:     #[my_attr(args)] fn ...
# // 3. Function-like macros: my_macro!(args)
#
# // You'll typically CONSUME them rather than write them; common producers:
# // serde       — Serialize/Deserialize derives
# // thiserror   — Error derive
# // tokio       — #[tokio::main], #[tokio::test]
# // sqlx        — query!() compile-time-checked SQL
```

## Error Handling

### Custom error type with `thiserror`

```bash
# // Cargo.toml: thiserror = "1"
# use thiserror::Error;
# #[derive(Error, Debug)]
# pub enum AppError {
#     #[error("io: {0}")]
#     Io(#[from] std::io::Error),
#     #[error("parse: {0}")]
#     Parse(#[from] serde_json::Error),
#     #[error("not found: {id}")]
#     NotFound { id: String },
# }
# pub type Result<T> = std::result::Result<T, AppError>;
# fn load(path: &str) -> Result<Config> {
#     let s = std::fs::read_to_string(path)?;     // io::Error  → AppError::Io
#     let c = serde_json::from_str(&s)?;          // serde err  → AppError::Parse
#     Ok(c)
# }
```

### Quick errors with `anyhow`

```bash
# // Cargo.toml: anyhow = "1"
# // For applications (not libraries) where you want a single Box-like error type.
# use anyhow::{Context, Result, bail};
# fn run(path: &str) -> Result<()> {
#     let s = std::fs::read_to_string(path)
#         .with_context(|| format!("reading {path}"))?;
#     if s.is_empty() { bail!("empty file"); }
#     Ok(())
# }
# // anyhow::Error preserves a backtrace and a chain of contexts.
```

### Error chain inspection

```bash
# use std::error::Error;
# fn print_chain(err: &dyn Error) {
#     eprintln!("error: {err}");
#     let mut src = err.source();
#     while let Some(e) = src {
#         eprintln!("caused by: {e}");
#         src = e.source();
#     }
# }
```

## Smart Pointers

### `Box<T>` — heap allocation

```bash
# let b: Box<i32> = Box::new(42);
# // Use cases:
# //   1. Recursive types (linked lists, trees) — Box gives a known size.
# //   2. Trait objects: Box<dyn Trait>.
# //   3. Moving large values cheaply (only a pointer copies).
# enum List { Cons(i32, Box<List>), Nil }
```

### `Rc<T>` and `Arc<T>` — reference counting

```bash
# use std::rc::Rc;
# use std::sync::Arc;
# // Rc:  single-threaded shared ownership. NOT Send.
# // Arc: atomic reference counting; thread-safe. Send + Sync if T: Send + Sync.
# let a = Rc::new(vec![1, 2, 3]);
# let b = Rc::clone(&a);                    // bumps refcount; both point at same vec
# Rc::strong_count(&a);                     // 2
# // Both Rc and Arc give SHARED ACCESS only — wrap in RefCell/Mutex for mutation.
```

### `RefCell<T>`, `Cell<T>` — interior mutability (single-threaded)

```bash
# use std::cell::{Cell, RefCell};
# // RefCell: runtime-borrow-checked &T / &mut T through .borrow() / .borrow_mut().
# let r = RefCell::new(5);
# *r.borrow_mut() += 1;
# let n: i32 = *r.borrow();                  // 6
# // Borrow rules enforced at RUNTIME — panics on conflicting borrow:
# // let _b1 = r.borrow_mut();
# // let _b2 = r.borrow();                   // PANIC: already mutably borrowed
#
# // Cell: get/set for Copy types — no borrow tracking needed.
# let c = Cell::new(0);
# c.set(c.get() + 1);
```

### `Mutex<T>` and `RwLock<T>` — interior mutability (multi-threaded)

```bash
# use std::sync::{Mutex, RwLock, Arc};
# let counter = Arc::new(Mutex::new(0));
# {
#     let counter = Arc::clone(&counter);
#     std::thread::spawn(move || {
#         let mut n = counter.lock().unwrap();      // returns LockResult; unwrap on poison
#         *n += 1;
#     });
# }
# // RwLock: many readers OR one writer.
# let cache = Arc::new(RwLock::new(std::collections::HashMap::<String, i32>::new()));
# cache.read().unwrap().get("k");
# cache.write().unwrap().insert("k".into(), 1);
```

### Choosing a smart pointer

```bash
# Single ownership, heap         →  Box<T>
# Shared, single-thread, immutable →  Rc<T>
# Shared, single-thread, mutable   →  Rc<RefCell<T>>
# Shared, multi-thread, immutable  →  Arc<T>
# Shared, multi-thread, mutable    →  Arc<Mutex<T>>  or  Arc<RwLock<T>>
# // Reach for parking_lot::Mutex for faster locks (no poisoning, smaller).
```

## Interior Mutability

### When to break the rules

```bash
# // Sometimes you need to mutate through a shared reference — interior mutability.
# // Rust offers Cell, RefCell, Mutex, RwLock, atomics, and OnceCell for this.
# use std::cell::OnceCell;
# struct Lazy { cache: OnceCell<String> }
# impl Lazy {
#     fn get(&self) -> &str {
#         self.cache.get_or_init(|| compute_expensive())
#     }
# }
# // OnceCell: write-once read-many. OnceLock is the thread-safe variant.
```

## Iterators

### Lazy by default

```bash
# // Iterator adapters return iterators — they DO NOT execute until consumed.
# let v = vec![1, 2, 3, 4, 5];
# let _ = v.iter().map(|x| { println!("seen {x}"); x * 2 });   // prints NOTHING
# let doubled: Vec<i32> = v.iter().map(|x| x * 2).collect();   // collect drives the chain
```

### Common adapters

```bash
# v.iter().map(|x| x * 2)
# v.iter().filter(|&&x| x > 2)
# v.iter().take(3); v.iter().skip(2);
# v.iter().take_while(|&&x| x < 4);
# v.iter().enumerate();                     // (idx, &val)
# v.iter().zip(other.iter());
# v.iter().chain(other.iter());
# v.iter().flat_map(|x| vec![x, x]);
# v.iter().rev();
# v.iter().step_by(2);
# v.iter().peekable();
# // Eager (consumers):
# v.iter().sum::<i32>();
# v.iter().product::<i32>();
# v.iter().count();
# v.iter().min(); v.iter().max();
# v.iter().fold(0, |acc, &x| acc + x);
# v.iter().any(|&x| x > 4); v.iter().all(|&x| x > 0);
# v.iter().find(|&&x| x == 3);
# v.iter().position(|&x| x == 3);
# v.iter().collect::<Vec<_>>();
# v.iter().collect::<HashSet<_>>();
# v.iter().collect::<HashMap<_, _>>();      // pairs (K, V) → HashMap
```

### Building your own iterator

```bash
# struct Counter { n: u32 }
# impl Iterator for Counter {
#     type Item = u32;
#     fn next(&mut self) -> Option<u32> {
#         self.n += 1;
#         if self.n < 6 { Some(self.n) } else { None }
#     }
# }
# // You get all 70+ adapters for free once you implement next.
```

## Async / Futures

### `async fn` and `.await`

```bash
# // async fn returns a state machine that implements Future.
# async fn fetch(url: &str) -> Result<String, reqwest::Error> {
#     let resp = reqwest::get(url).await?;
#     resp.text().await
# }
# // .await yields control to the executor when the future isn't ready.
# // ASYNC IS LAZY — calling fetch(url) does nothing; you must await it (or pass to spawn).
```

### Executor model

```bash
# // Rust's standard library has the Future trait but NO built-in executor.
# // You pick a runtime:
# //   tokio       — most popular; multi-threaded; rich ecosystem
# //   async-std   — std-like API
# //   smol        — small, modular
# //   embassy     — embedded / no_std async
```

### async fn in trait — stable in 1.75

```bash
# // Pre-1.75 you needed `async-trait` crate (boxed futures) or manual impl Future.
# // 1.75+ stable:
# trait Repository {
#     async fn get(&self, id: u64) -> Option<User>;
# }
# // Caveat: not yet object-safe in trait objects without `Box<dyn>` boilerplate
# // — use the async-trait crate when you need `dyn Trait`.
```

## Tokio essentials

### `#[tokio::main]` and `tokio::spawn`

```bash
# // Cargo.toml: tokio = { version = "1", features = ["full"] }
# #[tokio::main]
# async fn main() -> anyhow::Result<()> {
#     let handle = tokio::spawn(async {
#         tokio::time::sleep(std::time::Duration::from_secs(1)).await;
#         "done"
#     });
#     let result = handle.await?;            // JoinHandle yields the task's value
#     println!("{result}");
#     Ok(())
# }
```

### `tokio::select!` — race futures

```bash
# tokio::select! {
#     x = recv() => println!("got {x}"),
#     _ = tokio::time::sleep(std::time::Duration::from_secs(5)) => println!("timeout"),
#     _ = ctrl_c() => println!("shutting down"),
# }
# // The first ready branch wins; all others are CANCELLED. Be careful with side effects
# // in branches you don't take — the future drop runs, but partial work may have happened.
```

### Time

```bash
# tokio::time::sleep(Duration::from_millis(100)).await;
# tokio::time::timeout(Duration::from_secs(5), some_future).await   // Result<T, Elapsed>
# let mut interval = tokio::time::interval(Duration::from_secs(1));
# loop {
#     interval.tick().await;
#     do_thing();
# }
```

### Tokio channels

```bash
# // mpsc — multi-producer single-consumer
# let (tx, mut rx) = tokio::sync::mpsc::channel::<Msg>(64);
# tokio::spawn(async move { tx.send(Msg::Hi).await.unwrap(); });
# while let Some(msg) = rx.recv().await { handle(msg); }
#
# // oneshot — single message
# let (tx, rx) = tokio::sync::oneshot::channel::<i32>();
# tokio::spawn(async move { tx.send(42).unwrap(); });
# rx.await.unwrap();
#
# // broadcast — multi-producer multi-consumer (each receiver sees every message)
# // watch — single-value latest-state
```

## Threads

### `std::thread::spawn` and `JoinHandle`

```bash
# use std::thread;
# let h = thread::spawn(|| {
#     println!("hi from thread");
#     42
# });
# let n: i32 = h.join().unwrap();           // blocks until done; unwrap panics on thread panic
# // Spawned closure must be 'static (no borrows of the parent stack) and Send.
```

### Scoped threads (1.63+)

```bash
# // Scoped threads can borrow non-'static data — they MUST join by scope end.
# let v = vec![1, 2, 3];
# thread::scope(|s| {
#     s.spawn(|| println!("{v:?}"));        // borrows v, no clone, no Arc
#     s.spawn(|| println!("len {}", v.len()));
# });   // all spawned threads joined here
```

## Send + Sync

### What they mean

```bash
# Send  — values of this type can be MOVED to another thread.
# Sync  — references &T can be SHARED across threads (i.e., T: Sync iff &T: Send).
# // Auto-traits: most types are Send + Sync automatically. Exceptions:
# // Rc<T>     — NOT Send (refcount isn't atomic)
# // RefCell   — NOT Sync (borrow check isn't atomic)
# // raw ptrs  — NOT Send/Sync by default
# // To send across threads, use Arc + Mutex/RwLock instead of Rc + RefCell.
```

### Why a non-Send error happens

```bash
# // Common case: you held a Rc across an .await:
# async fn bad() {
#     let r = Rc::new(0);
#     do_io().await;
#     // r still alive here — task may have hopped to another thread → not Send
# }
# // Fix: drop Rc before await, or replace with Arc.
```

## Channels

### `std::sync::mpsc` — stdlib channel

```bash
# use std::sync::mpsc;
# let (tx, rx) = mpsc::channel::<i32>();    // unbounded
# let (tx, rx) = mpsc::sync_channel::<i32>(64);   // bounded (back-pressure)
# std::thread::spawn(move || tx.send(42).unwrap());
# let v = rx.recv().unwrap();
# // Multi-producer: tx.clone() → another sender. Single consumer.
```

### `crossbeam-channel` — better stdlib alternative

```bash
# // Cargo.toml: crossbeam-channel = "0.5"
# use crossbeam_channel::{bounded, unbounded, select};
# let (tx, rx) = bounded::<i32>(64);
# select! {
#     recv(rx) -> msg => println!("{:?}", msg),
#     send(tx, 1) -> _ => {},
#     default(std::time::Duration::from_millis(100)) => {},
# }
# // Faster, supports multi-consumer, has select macro.
```

### Async channels

```bash
# // tokio::sync::mpsc, broadcast, oneshot, watch — all the variants you'd expect.
# // For sync↔async bridge: tokio::sync::mpsc has both blocking and async send/recv.
```

## Atomics

### `std::sync::atomic` — lock-free primitives

```bash
# use std::sync::atomic::{AtomicI64, AtomicBool, Ordering};
# static COUNTER: AtomicI64 = AtomicI64::new(0);
# COUNTER.fetch_add(1, Ordering::Relaxed);
# let n = COUNTER.load(Ordering::Acquire);
# COUNTER.store(0, Ordering::Release);
# COUNTER.compare_exchange(0, 1, Ordering::SeqCst, Ordering::SeqCst);
```

### Memory orderings

```bash
# Relaxed   — no ordering guarantees beyond atomicity. Use for counters where order doesn't matter.
# Acquire   — used on loads. Subsequent reads/writes see effects from the matching Release.
# Release   — used on stores. Earlier reads/writes are visible to a matching Acquire.
# AcqRel    — for read-modify-write ops. Combines Acquire + Release.
# SeqCst    — total global order across all SeqCst ops. Strongest, slowest.
# // Default to SeqCst when unsure; only weaken after careful reasoning.
# // Read https://marabos.nl/atomics/ if you're doing nontrivial work here.
```

## unsafe

### Five superpowers (and only five)

```bash
# // unsafe lets you:
# // 1. Dereference a raw pointer
# // 2. Call an unsafe function (FFI, intrinsics)
# // 3. Access or mutate a mutable static
# // 4. Implement an unsafe trait
# // 5. Access fields of a union
# // It does NOT turn off the borrow checker. Most rules still apply.
```

### Raw pointers

```bash
# let mut x = 5;
# let r1: *const i32 = &x;
# let r2: *mut   i32 = &mut x;
# unsafe {
#     *r2 = 42;                              // dereference requires unsafe
#     println!("{}", *r1);                   // 42
# }
# // Raw pointers don't implement Send/Sync — your responsibility to reason about safety.
```

### `Pin<T>` — guaranteed not to move

```bash
# // Pin is needed for self-referential types (notably async state machines).
# // 99% of users only encounter Pin via async fn return types; library authors
# // implementing Future or Stream learn it deeply.
# use std::pin::Pin;
# fn poll(self: Pin<&mut Self>, ...) { ... }
```

## FFI

### Calling C from Rust

```bash
# extern "C" {
#     fn abs(x: i32) -> i32;                // declares libc::abs
# }
# fn main() {
#     let n = unsafe { abs(-5) };
#     println!("{n}");                       // 5
# }
# // Link a library:
# //   #[link(name = "foo")] extern "C" { fn foo_init(); }
# // For sane wrappers, use `bindgen` to auto-generate Rust bindings from C headers.
```

### Exposing Rust to C

```bash
# #[no_mangle]                               // keep the symbol name
# pub extern "C" fn rust_add(a: i32, b: i32) -> i32 { a + b }
# // Build as a cdylib in Cargo.toml:
# //   [lib]
# //   crate-type = ["cdylib"]
# // For C headers, use cbindgen.
```

### ABI safety

```bash
# // Only #[repr(C)] structs and enums are layout-stable across the boundary.
# #[repr(C)]
# struct Point { x: i32, y: i32 }
# // Default Rust repr is unspecified — DO NOT pass plain Rust structs to C.
# // Avoid passing &str (no null terminator) or Box<T> to C — use *const c_char and *mut T.
```

## File I/O

### Whole-file helpers

```bash
# use std::fs;
# let s: String = fs::read_to_string("config.toml")?;        // text
# let bytes: Vec<u8> = fs::read("blob.bin")?;                 // binary
# fs::write("out.txt", "hello\n")?;                           // overwrite
```

### Streaming reads with `BufReader`

```bash
# use std::fs::File;
# use std::io::{BufRead, BufReader};
# let f = File::open("big.txt")?;
# let reader = BufReader::new(f);
# for line in reader.lines() {
#     let line = line?;
#     process(&line);
# }
# // BufReader<File> avoids syscall-per-byte. ALWAYS use it for line-oriented input.
```

### Buffered writes

```bash
# use std::fs::File;
# use std::io::{BufWriter, Write};
# let f = File::create("out.txt")?;
# let mut w = BufWriter::new(f);
# writeln!(w, "line one")?;
# writeln!(w, "line two")?;
# w.flush()?;                                // CRITICAL — drop alone may swallow errors
```

### Append vs truncate vs create_new

```bash
# use std::fs::OpenOptions;
# let mut f = OpenOptions::new().append(true).create(true).open("log.txt")?;
# writeln!(f, "appended")?;
# // .read(true) .write(true) .truncate(true) .create_new(true) (fail if exists)
```

## Stdio / Args / Env

### `std::env`

```bash
# // Args — first element is the program name.
# let args: Vec<String> = std::env::args().collect();
# // Use args_os() if any arg might be non-UTF-8.
#
# // Env vars
# match std::env::var("HOME") {
#     Ok(v)  => println!("{v}"),
#     Err(_) => println!("HOME unset"),
# }
# std::env::set_var("KEY", "value");
# let cwd = std::env::current_dir()?;
# std::env::set_current_dir("/tmp")?;
```

### `std::io` — stdin/stdout/stderr

```bash
# use std::io::{self, BufRead, Write};
# let stdin = io::stdin();
# for line in stdin.lock().lines() {
#     let line = line?;
#     println!("got: {line}");
# }
# let mut out = io::stdout().lock();
# writeln!(out, "to stdout")?;
# writeln!(io::stderr(), "to stderr")?;
# // Use println! / eprintln! for casual stdout/stderr; lock for hot loops to avoid per-call lock overhead.
```

## Subprocess

### `std::process::Command`

```bash
# use std::process::{Command, Stdio};
# // Capture stdout
# let out = Command::new("git")
#     .args(["log", "-1", "--oneline"])
#     .output()?;                            // Output { status, stdout, stderr }
# println!("{}", String::from_utf8_lossy(&out.stdout));
#
# // Stream output (inherits parent stdio):
# let status = Command::new("ls")
#     .arg("-l")
#     .status()?;
# assert!(status.success());
#
# // Pipe stdin in:
# let mut child = Command::new("grep").arg("foo")
#     .stdin(Stdio::piped()).stdout(Stdio::piped()).spawn()?;
# child.stdin.as_mut().unwrap().write_all(b"foo\nbar\n")?;
# let output = child.wait_with_output()?;
# println!("{}", String::from_utf8_lossy(&output.stdout));
```

## Date & Time

### `std::time::Instant` and `Duration`

```bash
# use std::time::{Instant, Duration};
# let start = Instant::now();
# do_work();
# let elapsed: Duration = start.elapsed();
# println!("{:?}", elapsed);                 // 1.234s
# // Instant is monotonic — never goes backwards. Use for benchmarks and timeouts.
# // SystemTime is wall-clock, can jump (NTP) — use for timestamps you record.
```

### `chrono` — calendar dates and parsing

```bash
# // Cargo.toml: chrono = "0.4"
# use chrono::{DateTime, Utc, Local, NaiveDate};
# let now: DateTime<Utc> = Utc::now();
# let local: DateTime<Local> = Local::now();
# let parsed = DateTime::parse_from_rfc3339("2025-04-25T10:30:00Z")?;
# let formatted = now.format("%Y-%m-%d %H:%M:%S").to_string();
# let d = NaiveDate::from_ymd_opt(2025, 12, 25).unwrap();
```

### Modern alternatives

```bash
# // jiff (1.0 in 2024) — designed by the regex author; replaces chrono+time for new code:
# //   jiff = "0.1"
# //   let z: jiff::Zoned = jiff::Zoned::now();
# // time crate — minimal dep, no_std friendly:
# //   time = { version = "0.3", features = ["formatting", "parsing"] }
```

## Regex

### Basic usage

```bash
# // Cargo.toml: regex = "1"
# use regex::Regex;
# let re = Regex::new(r"^\d{4}-\d{2}-\d{2}$").unwrap();
# re.is_match("2025-04-25");                 // true
# // Use raw strings r"..." to avoid double escaping.
# // The regex crate is RE2-style — linear-time, no backreferences, no lookaround.
```

### Captures and named groups

```bash
# let re = Regex::new(r"(?P<user>\w+)@(?P<domain>\S+)").unwrap();
# if let Some(caps) = re.captures("alice@example.com") {
#     println!("user={}", &caps["user"]);
#     println!("domain={}", &caps["domain"]);
# }
# // Numeric capture access: &caps[1], &caps[2]
# // Iterate over all matches:
# for m in re.find_iter(text) { println!("{}", m.as_str()); }
```

### Replace

```bash
# let r = re.replace_all("a@x.com b@y.com", "<email>");
# // With backrefs:
# let r = re.replace_all(text, "$user at $domain");
# // With closure:
# let r = re.replace_all(text, |c: &regex::Captures| c[1].to_uppercase());
```

## HTTP

### `reqwest` — the standard client

```bash
# // Cargo.toml: reqwest = { version = "0.12", features = ["json", "blocking"] }
# // tokio = { version = "1", features = ["full"] }            (for async)
#
# // Async (preferred):
# let resp: serde_json::Value = reqwest::Client::new()
#     .get("https://api.example.com/users")
#     .bearer_auth(&token)
#     .send().await?
#     .error_for_status()?
#     .json().await?;
#
# // Blocking (handy for scripts and tests):
# let body = reqwest::blocking::get("https://example.com")?
#     .text()?;
#
# // POST JSON:
# let resp = reqwest::Client::new()
#     .post("https://api.example.com/users")
#     .json(&User { name: "Ada".into() })
#     .send().await?;
```

### Servers

```bash
# // axum (built on tokio + tower) — most common modern choice:
# //   axum = "0.7"
# //   tokio = { version = "1", features = ["full"] }
# // Other choices: actix-web, rocket, warp, poem, salvo.
```

## JSON

### `serde_json`

```bash
# // Cargo.toml: serde = { version = "1", features = ["derive"] }
# //             serde_json = "1"
# use serde::{Serialize, Deserialize};
# #[derive(Serialize, Deserialize, Debug)]
# struct User { name: String, age: u32 }
#
# let u = User { name: "Ada".into(), age: 36 };
# let s = serde_json::to_string(&u)?;
# let s = serde_json::to_string_pretty(&u)?;
# let parsed: User = serde_json::from_str(&s)?;
# // Streaming for big files:
# let r = std::fs::File::open("big.json")?;
# let parsed: User = serde_json::from_reader(std::io::BufReader::new(r))?;
```

### Untyped JSON

```bash
# let v: serde_json::Value = serde_json::from_str(r#"{"a":1,"b":[2,3]}"#)?;
# v["a"].as_i64();                           // Option<i64>
# v["b"][0].as_i64();
# // Useful when the schema is dynamic, but prefer typed structs for known shapes.
```

### Common serde attributes

```bash
# #[serde(rename = "userName")]              // camelCase / snake_case override
# #[serde(rename_all = "camelCase")]         // applied to entire struct
# #[serde(default)]                          // use Default::default() if missing
# #[serde(skip)]                             // never (de)serialize
# #[serde(skip_serializing_if = "Option::is_none")]
# #[serde(flatten)]                          // inline child fields
# #[serde(tag = "type")]                     // discriminator on enums
```

## Standard Library Highlights

```bash
# std::collections     — HashMap, BTreeMap, HashSet, BTreeSet, VecDeque, BinaryHeap, LinkedList
# std::iter            — repeat, once, empty, from_fn, successors; Iterator trait + adapters
# std::sync            — Mutex, RwLock, Arc, atomic::*, Once, OnceLock, Barrier, Condvar
# std::sync::mpsc      — channel(), sync_channel()
# std::thread          — spawn, scope, sleep, current, JoinHandle, ThreadId, available_parallelism
# std::fs              — read, write, read_to_string, File, OpenOptions, copy, rename, remove_file
# std::io              — Read, Write, BufRead, Seek, Cursor, BufReader, BufWriter, stdin, stdout
# std::path            — Path, PathBuf, components, parents, extension, join
# std::process         — Command, exit, abort, id; Output, ExitStatus, Stdio
# std::env             — args, var, vars, set_var, current_dir, home_dir (deprecated; use dirs)
# std::time            — Instant, Duration, SystemTime, UNIX_EPOCH
# std::error::Error    — the trait every error type implements (source(), Display, Debug)
# std::fmt             — Display, Debug, Write, Formatter, format_args!
# std::num             — NonZeroU32, NonZeroI64 (niche-optimized), parse helpers
# std::mem             — swap, replace, take, drop, size_of, align_of, transmute (unsafe)
# std::ops             — Deref, Drop, Add, Sub, Index, RangeBounds, Fn/FnMut/FnOnce
# std::pin             — Pin, pin! macro (1.68+)
# std::cell            — Cell, RefCell, OnceCell
```

## Tests

### `#[test]` — unit tests

```bash
# // src/lib.rs
# pub fn add(a: i32, b: i32) -> i32 { a + b }
#
# #[cfg(test)]
# mod tests {
#     use super::*;
#
#     #[test]
#     fn add_works() {
#         assert_eq!(add(2, 3), 5);
#     }
#
#     #[test]
#     #[should_panic(expected = "div by zero")]
#     fn divide_panics_on_zero() {
#         let _ = 1 / 0;
#     }
#
#     #[test]
#     #[ignore = "slow"]
#     fn slow_test() { /* run with cargo test -- --ignored */ }
# }
# // assert_eq!, assert_ne!, assert!, dbg! macros are your toolkit.
```

### Integration tests in `tests/`

```bash
# // tests/api.rs — each file is a separate compilation unit, exercises the public API.
# // tests/common/mod.rs — shared helpers; declared from tests/api.rs as `mod common;`
# use mycrate::run;
# #[test]
# fn end_to_end() {
#     assert!(run().is_ok());
# }
```

### Doc tests

```bash
# /// Adds two numbers.
# ///
# /// # Examples
# ///
# /// ```
# /// use mycrate::add;
# /// assert_eq!(add(2, 3), 5);
# /// ```
# pub fn add(a: i32, b: i32) -> i32 { a + b }
# // cargo test compiles + runs all examples in /// ``` blocks.
```

### Running

```bash
cargo test                                 # all tests
cargo test add_                            # only tests matching "add_"
cargo test --release                       # with optimizations
cargo test -- --nocapture                  # show println from passing tests
cargo test -- --test-threads=1             # serial (for tests touching shared state)
cargo test --doc                           # only doc tests
# Or use cargo-nextest for faster execution + better output.
```

## Benchmarks

### `criterion` — stable benchmarks

```bash
# // Cargo.toml:
# //   [dev-dependencies]
# //   criterion = { version = "0.5", features = ["html_reports"] }
# //   [[bench]]
# //   name = "my_bench"
# //   harness = false
#
# // benches/my_bench.rs
# use criterion::{black_box, criterion_group, criterion_main, Criterion};
# fn bench(c: &mut Criterion) {
#     c.bench_function("add 2 3", |b| b.iter(|| add(black_box(2), black_box(3))));
# }
# criterion_group!(benches, bench);
# criterion_main!(benches);
# // Run: cargo bench
# // Generates target/criterion/.../report/index.html with regression detection.
```

### `cargo bench` (nightly only for stdlib `#[bench]`)

```bash
# #![feature(test)]                          // nightly
# extern crate test;
# #[bench]
# fn bench_add(b: &mut test::Bencher) {
#     b.iter(|| add(2, 3));
# }
# // Use criterion on stable instead — strictly better.
```

## Lint / Format

### `rustfmt`

```bash
cargo fmt                                  # format the whole crate
cargo fmt -- --check                       # CI mode — fail on diff
# rustfmt.toml controls style:
#   max_width            = 100
#   tab_spaces           = 4
#   imports_granularity  = "Crate"
#   group_imports        = "StdExternalCrate"
```

### `clippy`

```bash
cargo clippy                               # lint
cargo clippy --all-targets --all-features -- -D warnings    # CI strict mode
# Lint groups: clippy::correctness clippy::suspicious clippy::style clippy::complexity
#              clippy::perf clippy::pedantic clippy::nursery clippy::cargo
# Allow inline:
#   #[allow(clippy::needless_return)]
# Enable a group at the crate root:
#   #![warn(clippy::pedantic)]
```

### `deny(warnings)` at crate root

```bash
# // src/lib.rs OR src/main.rs:
# #![deny(missing_docs)]
# #![warn(clippy::all)]
# #![warn(rust_2018_idioms, unreachable_pub)]
# // deny(warnings) blanket is controversial — it locks you to a specific compiler version.
# // Better: deny specific lints that matter, warn for the rest, and run with -D warnings in CI only.
```

## no_std and Embedded

### `#![no_std]` crate

```bash
# // src/lib.rs
# #![no_std]
# // No std::*. You get core::* (intrinsics, basic types, iter) and optionally alloc::*
# // (Box, Vec, String — when an allocator is provided).
#
# extern crate alloc;
# use alloc::vec::Vec;
# use alloc::string::String;
```

### Panic handler

```bash
# // no_std binaries must define how panics behave:
# #![no_std]
# #![no_main]
# use core::panic::PanicInfo;
# #[panic_handler]
# fn panic(_info: &PanicInfo) -> ! {
#     loop {}                                // or reset the MCU
# }
# // For embedded, use crates like `panic-halt`, `panic-semihosting`, `panic-probe`.
# // Targets: cargo build --target thumbv7em-none-eabihf  (Cortex-M4F)
# //          cargo build --target riscv32imac-unknown-none-elf
```

## Common Gotchas

### Move-after-use

```bash
# let s = String::from("hi");
# let t = s;
# println!("{}", s);                         // ERROR: borrow of moved value `s`
# // Fix: clone, borrow, or restructure.
# let s = String::from("hi");
# let t = s.clone();
# println!("{s} {t}");                       // ok
```

### Borrow checker fights

```bash
# // Common shape: "cannot borrow as mutable because also borrowed as immutable"
# let mut v = vec![1, 2, 3];
# let first = &v[0];
# v.push(4);                                 // ERROR: v also borrowed immutably
# println!("{first}");                       // shared borrow used here
# // Fix — drop the borrow before mutating, or copy out:
# let first = v[0];                          // i32: Copy → no borrow
# v.push(4);
# println!("{first}");                       // ok
```

### Lifetime soup in returns

```bash
# // BUG — trying to return a reference to a local:
# fn dangling() -> &str {
#     let s = String::from("hi");
#     &s                                     // ERROR: s dropped at end of fn
# }
# // Fix — return owned data:
# fn ok() -> String { String::from("hi") }
# // Or have the caller pass in the buffer:
# fn fill<'a>(buf: &'a mut String) -> &'a str {
#     buf.push_str("hi"); buf.as_str()
# }
```

### `unwrap()` in production

```bash
# // unwrap() panics on None / Err. Acceptable in:
# //   • tests
# //   • prototypes
# //   • cases proven impossible by surrounding code (and even then prefer expect("why"))
# // NOT acceptable in library entry points or anywhere a user can reach it.
# // Replace with ?, .ok_or(), .unwrap_or_default(), or proper error propagation.
```

### Indexing into String

```bash
# let s = "héllo";
# // s[0]                                    // ERROR — String is not Index<usize>
# // Why? UTF-8 — index 0..=5 are bytes; characters span multiple bytes.
# &s[0..1]                                   // OK if byte 0 is a char boundary
# s.chars().nth(0)                           // O(n) — Option<char>
```

### `Rc` cycles leak

```bash
# // Rc/Arc count strong references. A cycle keeps refcount > 0 forever.
# // Break cycles with Weak<T> for the back-pointer:
# use std::rc::{Rc, Weak};
# struct Node { parent: Weak<Node>, children: Vec<Rc<Node>> }
# // Weak::upgrade() returns Option<Rc<T>> — None if dropped.
```

### async fn in trait — pre-1.75

```bash
# // Pre-1.75: async fn in trait was unstable. Workarounds:
# //   #[async_trait::async_trait]
# //   trait Repo { async fn get(&self) -> User; }
# // 1.75+: native, but trait objects (dyn Repo) still need workarounds for async methods —
# // use the `trait-variant` crate or design around it.
```

## Performance Tips

### Always benchmark in release mode

```bash
cargo run --release
cargo test --release
cargo bench
# Debug builds are 10-100x slower for numeric code due to overflow checks and no inlining.
```

### Compiler flags

```bash
# Cargo.toml:
# [profile.release]
# lto           = "fat"                     # whole-program optimization
# codegen-units = 1                          # single codegen unit (best inlining)
# strip         = "symbols"
# panic         = "abort"                    # smaller binary, faster (no unwind)
#
# Target-specific instructions:
#   RUSTFLAGS="-C target-cpu=native" cargo build --release
# Profile-guided optimization (PGO):
#   RUSTFLAGS="-Cprofile-generate=/tmp/pgo" cargo build --release
#   # ... run representative workload ...
#   llvm-profdata merge -o /tmp/pgo/merged.profdata /tmp/pgo
#   RUSTFLAGS="-Cprofile-use=/tmp/pgo/merged.profdata" cargo build --release
```

### Avoid `Box<dyn Trait>` in hot loops

```bash
# // Each call goes through the vtable — no inlining.
# // For numerical inner loops, prefer generics (monomorphized) or concrete types.
# fn hot<F: Fn(f64) -> f64>(f: F, x: f64) -> f64 { f(x) }   // inlined per call site
# fn slow(f: &dyn Fn(f64) -> f64, x: f64) -> f64  { f(x) }  // virtual call
```

### Reduce allocations

```bash
# // Pre-allocate vectors and strings:
# let mut v = Vec::with_capacity(n);
# let mut s = String::with_capacity(estimated_bytes);
# // Reuse buffers in loops instead of allocating fresh each iteration.
# // For numeric work, prefer iterators over collect() to avoid intermediate Vecs:
# let sum: i32 = (0..1_000_000).filter(|n| n % 2 == 0).sum();
```

### Use `cargo flamegraph` and `perf`

```bash
cargo install flamegraph                   # needs perf on Linux, dtrace on macOS
cargo flamegraph --bin myapp               # opens flamegraph.svg
# perf record / perf report works on the unstripped release binary directly.
```

## Idioms

### Newtype pattern

```bash
# // Wrap a type to give it new semantics, prevent mix-ups, and add methods.
# struct UserId(u64);
# struct OrderId(u64);
# fn ban(id: UserId) { ... }
# // ban(OrderId(5));                        // COMPILE ERROR — type-distinct
# // Add Display, From, etc. via derive or manual impl as needed.
```

### Typestate — encode states in the type

```bash
# struct Locked;  struct Unlocked;
# struct Door<S> { _phantom: std::marker::PhantomData<S> }
# impl Door<Locked>   { fn unlock(self) -> Door<Unlocked> { Door { _phantom: PhantomData } } }
# impl Door<Unlocked> { fn open(&self) {}  fn lock(self) -> Door<Locked> { Door { _phantom: PhantomData } } }
# let d = Door::<Locked> { _phantom: PhantomData };
# // d.open();                               // COMPILE ERROR — only Unlocked has open()
# d.unlock().open();                         // OK
```

### Builder pattern

```bash
# struct ServerBuilder { port: u16, host: String, tls: bool }
# impl ServerBuilder {
#     pub fn new() -> Self { Self { port: 8080, host: "0.0.0.0".into(), tls: false } }
#     pub fn port(mut self, p: u16) -> Self { self.port = p; self }
#     pub fn host(mut self, h: impl Into<String>) -> Self { self.host = h.into(); self }
#     pub fn tls(mut self, t: bool) -> Self  { self.tls = t; self }
#     pub fn build(self) -> Server { Server { /* ... */ } }
# }
# let server = ServerBuilder::new().port(9000).tls(true).build();
```

### `From` / `Into` duality

```bash
# // Implement From<X> for Y — you GET Into<Y> for X for free.
# impl From<u32> for MyId {
#     fn from(n: u32) -> Self { MyId(n) }
# }
# let id: MyId = 42_u32.into();              // uses From<u32>
# fn save(id: impl Into<MyId>) { let id = id.into(); ... }
# // ? operator relies on From: any inner err is converted via From into the outer error type.
```

### `Drop` for RAII cleanup

```bash
# struct File { handle: i32 }
# impl Drop for File {
#     fn drop(&mut self) {
#         println!("closing fd {}", self.handle);
#         // close(self.handle);
#     }
# }
# // Drop runs at end of scope — guaranteed even on panic (unless panic = "abort").
# // You CANNOT call .drop() yourself; use std::mem::drop(x) to drop early.
```

## Tips

- The borrow checker is strict but prevents data races, use-after-free, and iterator invalidation at compile time. Time spent fighting it is debugging time you don't pay later.
- `&str` for function parameters, `String` for owned data stored in structs. `&[T]` for read-only slice params, `Vec<T>` for owned.
- `clone()` is explicit and visible — unlike implicit copies in C++/Java. If you see a `.clone()` you don't like, examine the data flow before deleting it.
- Prefer `?` over `.unwrap()` for error handling. Reserve `.unwrap()` for tests and cases proven impossible.
- `cargo clippy --all-targets --all-features -- -D warnings` in CI catches more bugs than the compiler alone.
- Always `#[derive(Debug)]` on public types. You will need it for error messages, panic output, and `dbg!` debugging.
- `impl From<X> for Y` is the gateway drug that enables `?` and `.into()` conversions everywhere.
- `Vec`, `HashMap`, `String`, `Box` allocate on the heap. Slices (`&[T]`, `&str`) are just views — no allocation.
- For library APIs, return concrete types when possible. Use `impl Trait` for hidden iterator chains. Use `dyn Trait` only when you need heterogeneous storage.
- `cargo expand` reveals what your macros generate — invaluable when debugging derive output.
- For async: pick `tokio` first unless you have a reason otherwise. The ecosystem is largest, the docs best.
- `Arc<Mutex<T>>` is the workhorse shared-mutable state across threads. `parking_lot::Mutex` is faster.
- Never block on async tasks (no `std::thread::sleep` inside async fn) — use `tokio::time::sleep`.
- For CLI tools, `clap` is the standard arg parser; `tracing` is the standard structured logging crate.
- Pin your MSRV in CI on a stable channel — accidentally bumping it breaks downstream.
- `cargo doc --open --no-deps` after writing doc comments — Rust's docs are first-class.
- `#[must_use]` on Result-returning fns and lint-flagged types stops silent error swallowing.
- Use `std::mem::take` and `std::mem::replace` to move out of `&mut self` without partial-move pain.
- For zero-copy parsing, `Cow<'_, str>` lets you avoid allocations when input was already a borrowed string.
- `cargo install cargo-audit && cargo audit` checks dependencies against the RustSec advisory DB.

## See Also

- polyglot
- go
- python
- typescript
- javascript
- c
- java
- ruby
- lua
- bash
- make
- cargo
- toml
- webassembly

## References

- [The Rust Book](https://doc.rust-lang.org/book/) -- official tutorial and guide
- [Rust Standard Library](https://doc.rust-lang.org/std/) -- stdlib API reference
- [Rust Reference](https://doc.rust-lang.org/reference/) -- formal language syntax and semantics
- [Rust by Example](https://doc.rust-lang.org/rust-by-example/) -- learn through annotated examples
- [Rust Standard Library by Example (std-by-example)](https://github.com/rust-lang-nursery/rust-cookbook) -- cookbook of common tasks
- [Rustlings](https://github.com/rust-lang/rustlings) -- small interactive exercises to learn Rust
- [The Cargo Book](https://doc.rust-lang.org/cargo/) -- package manager and build system guide
- [The Rustonomicon](https://doc.rust-lang.org/nomicon/) -- guide to unsafe Rust and advanced topics
- [The Rust Edition Guide](https://doc.rust-lang.org/edition-guide/) -- edition differences (2015, 2018, 2021, 2024)
- [Rust Async Book](https://rust-lang.github.io/async-book/) -- async/await fundamentals
- [Tokio Tutorial](https://tokio.rs/tokio/tutorial) -- official tokio guide
- [crates.io](https://crates.io/) -- Rust package registry
- [docs.rs](https://docs.rs/) -- auto-generated documentation for every published crate
- [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/) -- conventions for idiomatic Rust APIs
- [Rust Playground](https://play.rust-lang.org/) -- run and share Rust code online
- [This Week in Rust](https://this-week-in-rust.org/) -- weekly newsletter of Rust ecosystem updates
- [Rust Forge](https://forge.rust-lang.org/) -- contributor and team documentation
- [Rust Blog](https://blog.rust-lang.org/) -- official release announcements and articles
- [Rust Atomics and Locks (Mara Bos)](https://marabos.nl/atomics/) -- definitive book on memory models and locking
- [Programming Rust (O'Reilly, 2nd ed.)](https://www.oreilly.com/library/view/programming-rust-2nd/9781492052586/) -- comprehensive reference
- [Zero To Production In Rust](https://www.zero2prod.com/) -- web service development end-to-end
- [Awesome Rust](https://github.com/rust-unofficial/awesome-rust) -- curated list of Rust libraries and resources
