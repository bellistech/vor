# Rust Errors

Verbatim Rust compiler errors, borrow checker diagnostics, lifetime issues, type errors, cargo failures, and clippy lints with exact text, root cause, and the fix.

## Setup

Rust's error model is two-tiered: **panics** for unrecoverable bugs (out-of-bounds index, integer overflow in debug, `unwrap` on `None`) and **`Result<T, E>`** for recoverable errors (I/O, parse failures, business-logic failures). Panics unwind the stack by default and abort the thread; `Result` is handled with `match`, `?`, `if let`, `unwrap_or_else`.

```rust
use std::fs::File;
use std::io::{self, Read};

// Result<T, E>: recoverable
fn read_config() -> Result<String, io::Error> {
    let mut f = File::open("config.toml")?; // ? propagates io::Error
    let mut s = String::new();
    f.read_to_string(&mut s)?;
    Ok(s)
}

// Panic: unrecoverable bug
fn first(v: &[i32]) -> i32 {
    v[0] // panics if v is empty — programmer error to call on empty slice
}
```

The `std::error::Error` trait is the standard error abstraction. Errors implement `Display + Debug` and optionally `source()` for chained causes.

```rust
use std::error::Error;
use std::fmt;

#[derive(Debug)]
struct MyError {
    msg: String,
    source: Option<Box<dyn Error + Send + Sync>>,
}

impl fmt::Display for MyError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{}", self.msg)
    }
}

impl Error for MyError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        self.source.as_deref().map(|e| e as _)
    }
}
```

The `?` operator unwraps `Ok(v)` to `v` or returns `Err(e.into())` from the enclosing function. It works with `Result<T, E>` and `Option<T>`. The `From` impl chain lets `?` convert error types implicitly.

```rust
fn parse_then_double(s: &str) -> Result<i32, std::num::ParseIntError> {
    let n: i32 = s.parse()?; // ParseIntError → Result::Err early return
    Ok(n * 2)
}
```

Rust's "fail fast" stance: don't paper over invariant violations. If something is genuinely a bug (negative array length, exhausted iterator that should have items), panic loudly so it's caught in test/dev. Reserve `Result` for situations the caller can reasonably recover from.

The compiler's diagnostic format is rich: `error[ECCCC]: <summary>` headers, file/line spans with caret markers, `help:` suggested fixes, `note:` explanatory context, and frequently a copy-pasteable suggestion. Read every line — the first error often dominates the rest.

## How to Read a Rust Error

A typical error has structure:

```text
error[E0382]: borrow of moved value: `s`
  --> src/main.rs:5:20
   |
 3 |     let s = String::from("hi");
   |         - move occurs because `s` has type `String`, which does not implement the `Copy` trait
 4 |     let t = s;
   |             - value moved here
 5 |     println!("{}", s);
   |                    ^ value borrowed here after move
   |
   = note: in the call to `s` after move
help: consider cloning the value if the performance cost is acceptable
   |
 4 |     let t = s.clone();
   |              ++++++++
```

Anatomy:

- `error[E0382]` — the error code; every code is documented at <doc.rust-lang.org/error-index.html>.
- `borrow of moved value: \`s\`` — one-line summary.
- `--> src/main.rs:5:20` — primary span.
- `|` and line numbers — context lines.
- `-`, `^` markers — secondary and primary spans pointing at the offending tokens. `-` is supporting context, `^` is the focal point.
- `= note:` — additional explanation.
- `help: consider …` — actionable hint, often with a code suggestion you can paste.
- `++++` and `----` — added/removed text in suggestions.

Read errors top-down. The first error is the cause; subsequent errors are often cascade failures. Fix one, recompile, repeat. Use `cargo check` to iterate fast (no codegen).

Multi-line spans look like:

```text
error[E0277]: the trait bound `MyType: Display` is not satisfied
  --> src/main.rs:8:20
   |
 8 |     println!("{}", x);
   |                    ^ `MyType` cannot be formatted with the default formatter
   |
   = help: the trait `Display` is not implemented for `MyType`
   = note: in format arguments, you may be able to use `{:?}` (which uses the `Debug` trait) instead
```

Always read `help:` and `note:` — they often spell out the fix verbatim. The `consider X` hints are reliable: if rustc says "consider borrowing", borrow.

## Borrow Checker Errors

The borrow checker enforces ownership: a value has exactly one owner; references are either one mutable XOR many immutable, and never outlive the value.

### error[E0382]: use of moved value

```text
error[E0382]: use of moved value: `s`
 --> src/main.rs:5:20
  |
3 |     let s = String::from("hello");
  |         - move occurs because `s` has type `String`, which does not implement the `Copy` trait
4 |     let t = s;
  |             - value moved here
5 |     println!("{}", s);
  |                    ^ value used here after move
```

Cause: ownership of `s` transferred into `t`; `s` is invalidated. Same applies to passing into a function or capturing in a closure.

Fix:

```rust
// Option 1: clone (if cheap or required)
let t = s.clone();
println!("{}", s);

// Option 2: borrow instead
let t = &s;
println!("{}", s);

// Option 3: design with Copy if value is small (i32, bool, char, etc.)
#[derive(Copy, Clone)]
struct Point { x: i32, y: i32 }
```

### error[E0382]: borrow of moved value

```text
error[E0382]: borrow of moved value: `v`
 --> src/main.rs:5:14
  |
3 |     let v = vec![1, 2, 3];
4 |     consume(v);
  |             - value moved here
5 |     println!("{:?}", v);
  |              ^^^^^^^^^^^ value borrowed here after move
```

Cause: passed `v` by value into `consume`; can't print after.

Fix: `consume(&v)` if signature allows, or pass `v.clone()` if not.

### error[E0507]: cannot move out of `X` which is behind a shared reference

```text
error[E0507]: cannot move out of `*self.name` which is behind a shared reference
 --> src/lib.rs:8:9
  |
8 |         self.name
  |         ^^^^^^^^^ move occurs because `self.name` has type `String`, which does not implement the `Copy` trait
```

Cause: trying to move a non-`Copy` field out of `&self`.

Fix:

```rust
// Borrow the field
fn name(&self) -> &str { &self.name }
// Or clone
fn name(&self) -> String { self.name.clone() }
// Or use .take() if the field is Option<T>
fn take_name(&mut self) -> Option<String> { self.name.take() }
// Or std::mem::take to swap with default
let name = std::mem::take(&mut self.name);
```

### error[E0507]: cannot move out of borrowed content

```text
error[E0507]: cannot move out of borrowed content
 --> src/main.rs:5:13
  |
5 |     let n = (*r).name;
  |             ^^^^^^^^^ cannot move out of borrowed content
```

Cause: dereferencing a `&T` and trying to move a non-`Copy` field.

Fix: borrow the field (`&(*r).name` or `&r.name`), or use `r.name.clone()`.

### error[E0506]: cannot assign to `X` because it is borrowed

```text
error[E0506]: cannot assign to `v` because it is borrowed
 --> src/main.rs:6:5
  |
4 |     let r = &v;
  |             -- borrow of `v` occurs here
5 |     v = vec![10, 20, 30];
  |     ^^^^^^^^^^^^^^^^^^^^ assignment to borrowed `v` occurs here
6 |     println!("{:?}", r);
  |                      - borrow later used here
```

Cause: `r` is an immutable borrow of `v`; assigning to `v` invalidates `r`.

Fix: drop `r` (let scope end) before reassigning, or restructure so the borrow ends before reassignment. With NLL the borrow ends at last use, so simply not using `r` after the assignment can work.

### error[E0499]: cannot borrow `X` as mutable more than once at a time

```text
error[E0499]: cannot borrow `v` as mutable more than once at a time
 --> src/main.rs:5:14
  |
4 |     let a = &mut v;
  |             ------ first mutable borrow occurs here
5 |     let b = &mut v;
  |             ^^^^^^ second mutable borrow occurs here
6 |     a.push(1);
  |     - first borrow later used here
```

Cause: Rust forbids two simultaneous `&mut` to the same place.

Fix:

```rust
// Use one borrow at a time
{
    let a = &mut v;
    a.push(1);
}
let b = &mut v;
b.push(2);

// Or split borrows: split_at_mut, two halves
let (left, right) = v.split_at_mut(mid);

// Or use indices instead of references
for i in 0..v.len() {
    v[i] += 1;
}
```

### error[E0502]: cannot borrow `X` as mutable because it is also borrowed as immutable

```text
error[E0502]: cannot borrow `v` as mutable because it is also borrowed as immutable
 --> src/main.rs:5:5
  |
4 |     let r = &v[0];
  |             -- immutable borrow occurs here
5 |     v.push(4);
  |     ^^^^^^^^^ mutable borrow occurs here
6 |     println!("{}", r);
  |                    - immutable borrow later used here
```

Cause: holding `&v[0]` while mutating `v` would invalidate the reference if the Vec reallocates.

Fix: copy/clone the value out before mutating, or finish using the immutable borrow first.

```rust
let val = v[0]; // copy out
v.push(4);
println!("{}", val);
```

### error[E0500]: closure requires unique access to `X` but it is already borrowed

```text
error[E0500]: closure requires unique access to `v` but it is already borrowed
 --> src/main.rs:5:14
  |
4 |     let r = &v;
  |             -- borrow occurs here
5 |     let mut c = || v.push(1);
  |                 ^^ - second borrow occurs due to use of `v` in closure
  |                 |
  |                 closure construction occurs here
6 |     println!("{:?}", r);
  |                      - first borrow later used here
```

Cause: the closure captures `v` mutably; an outstanding `&v` conflicts.

Fix: let the immutable borrow end before constructing the closure, or restructure.

### error[E0503]: cannot use `X` because it was mutably borrowed

```text
error[E0503]: cannot use `v` because it was mutably borrowed
 --> src/main.rs:5:20
  |
4 |     let r = &mut v;
  |             ------ borrow of `v` occurs here
5 |     println!("{:?}", v);
  |                    ^ use of borrowed `v`
6 |     r.push(1);
  |     - borrow later used here
```

Cause: while `r` is live, `v` itself is unusable.

Fix: finish with `r` first, or use `r` directly: `println!("{:?}", r);`.

### error[E0501]: cannot borrow `X` as mutable because previous closure requires unique access

```text
error[E0501]: cannot borrow `v` as mutable because previous closure requires unique access
 --> src/main.rs:5:14
  |
4 |     let mut c = || v.push(1);
  |                 -- - first borrow occurs due to use of `v` in closure
  |                 |
  |                 closure construction occurs here
5 |     let r = &mut v;
  |             ^^^^^^ second borrow occurs here
6 |     c();
  |     - first borrow later used here
```

Cause: same as E0500 mirror; closure captures uniquely while you also try `&mut v`.

Fix: drop the closure first, or rework so the captures don't overlap.

### error[E0381]: borrow of possibly-uninitialized variable

```text
error[E0381]: borrow of possibly-uninitialized variable: `x`
 --> src/main.rs:4:20
  |
2 |     let x: i32;
3 |     if cond { x = 1; }
4 |     println!("{}", x);
  |                    ^ use of possibly-uninitialized `x`
```

Cause: at least one branch leaves `x` uninitialized.

Fix: initialize on every path, or assign a default first.

```rust
let x: i32 = if cond { 1 } else { 0 };
```

## Lifetime Errors

Lifetimes are compile-time scopes for references. Most are elided; explicit ones (`'a`) name them.

### error[E0106]: missing lifetime specifier

```text
error[E0106]: missing lifetime specifier
 --> src/lib.rs:1:33
  |
1 | fn longest(x: &str, y: &str) -> &str {
  |               ----     ----     ^ expected named lifetime parameter
  |
  = help: this function's return type contains a borrowed value, but the signature does not say whether it is borrowed from `x` or `y`
help: consider introducing a named lifetime parameter
  |
1 | fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
  |           ++++     ++          ++          ++
```

Cause: lifetime elision can't infer the return reference's source.

Fix: name the lifetime as the help suggests.

### error[E0495]: cannot infer an appropriate lifetime

```text
error[E0495]: cannot infer an appropriate lifetime for borrow expression due to conflicting requirements
```

Cause: a borrow's required scope conflicts with an inferred constraint, often from a trait bound or HRTB.

Fix: add explicit `'a` annotations. Try `for<'a>` higher-ranked bounds for closures over references.

### error[E0597]: `X` does not live long enough

```text
error[E0597]: `s` does not live long enough
 --> src/main.rs:6:13
  |
4 |     let r;
5 |     {
6 |         let s = String::from("hi");
  |             - binding `s` declared here
7 |         r = &s;
  |             ^^ borrowed value does not live long enough
8 |     }
  |     - `s` dropped here while still borrowed
9 |     println!("{}", r);
  |                    - borrow later used here
```

Cause: `s` goes out of scope while a reference to it (`r`) is still live.

Fix: extend `s`'s scope or own the data:

```rust
let s = String::from("hi"); // outer scope
let r = &s;
println!("{}", r);

// Or own the data
let r: String = String::from("hi");
```

### error[E0621]: explicit lifetime required in the type of `X`

```text
error[E0621]: explicit lifetime required in the type of `arg`
 --> src/lib.rs:2:5
  |
1 | fn foo(arg: &str) -> &'static str {
  |             ---- help: add explicit lifetime `'static` to the type of `arg`: `&'static str`
2 |     arg
  |     ^^^ lifetime `'static` required
```

Cause: returning `&'static str` but accepting `&str` (any lifetime).

Fix: change return to `&str`, accept `&'static str`, or return an owned `String`.

### error[E0623]: lifetime mismatch

```text
error[E0623]: lifetime mismatch
 --> src/lib.rs:5:5
  |
3 | fn foo<'a, 'b>(x: &'a str, y: &'b str) -> &'a str {
  |                            -------         -------
  |                            |
  |                            this parameter and the return type are declared with different lifetimes...
5 |     y
  |     ^ ...but data from `y` is returned here
```

Cause: returning `y` (lifetime `'b`) but signature promises `'a`.

Fix: unify lifetimes (`'a: 'b` outlives bound, or one named lifetime), or return the right value.

### error[E0700]: hidden type for `impl Trait` captures lifetime that does not appear in bounds

```text
error[E0700]: hidden type for `impl Iterator<Item = &i32>` captures lifetime that does not appear in bounds
 --> src/lib.rs:1:30
  |
1 | fn it(v: &Vec<i32>) -> impl Iterator<Item = &i32> {
  |          ---------     ^^^^^^^^^^^^^^^^^^^^^^^^^^
  |          |
  |          hidden type `Iter<'_, i32>` captures the anonymous lifetime defined here
help: to declare that `impl Iterator<Item = &i32>` captures `'_`, you can add an explicit `'_` lifetime bound
  |
1 | fn it(v: &Vec<i32>) -> impl Iterator<Item = &i32> + '_ {
  |                                                   ++++
```

Cause: returned `impl Trait` borrows from input but doesn't say so.

Fix: `+ '_` or name the lifetime: `impl Iterator<Item = &'a i32>`.

### error[E0759]: `X` has lifetime `'a` but it needs to satisfy a `'static` lifetime requirement

```text
error[E0759]: `data` has lifetime `'a` but it needs to satisfy a `'static` lifetime requirement
 --> src/main.rs:5:24
  |
3 | fn spawn<'a>(data: &'a [u8]) {
  |                    -------- this data with lifetime `'a`...
4 |     std::thread::spawn(move || {
5 |         println!("{:?}", data);
  |                          ^^^^ ...is captured here, requiring it to live as long as `'static`
```

Cause: `thread::spawn` requires `'static` closures; borrowed data won't satisfy.

Fix: own the data (`data.to_vec()`), use scoped threads (`std::thread::scope`), or `Arc`.

### error[E0716]: temporary value dropped while borrowed

```text
error[E0716]: temporary value dropped while borrowed
 --> src/main.rs:2:14
  |
2 |     let s = String::from("hi").as_str();
  |             ^^^^^^^^^^^^^^^^^^^         - temporary value is freed at the end of this statement
  |             |
  |             creates a temporary value which is freed while still in use
3 |     println!("{}", s);
  |                    - borrow later used here
help: consider using a `let` binding to create a longer lived value
  |
2 |     let binding = String::from("hi");
2 |     let s = binding.as_str();
  |
```

Cause: a temporary (the `String`) is dropped at end of statement, but `s` borrows it.

Fix: bind the temporary to a `let` so it lives.

## Type Errors

### error[E0308]: mismatched types

```text
error[E0308]: mismatched types
 --> src/main.rs:2:18
  |
2 |     let n: i32 = "42";
  |            ---   ^^^^ expected `i32`, found `&str`
  |            |
  |            expected due to this
```

Cause: declared/expected `i32`, supplied `&str`.

Fix: parse: `"42".parse::<i32>().unwrap()` or use literal `42`.

Common causes: returning `()` from a function expected to return `T` (missing trailing expression), supplying `String` where `&str` is expected (use `&s`), supplying `Option<T>` where `T` is expected (`.unwrap()` or pattern-match).

### error[E0277]: the trait bound `X: Y` is not satisfied

```text
error[E0277]: the trait bound `MyStruct: Display` is not satisfied
 --> src/main.rs:8:20
  |
8 |     println!("{}", x);
  |                    ^ the trait `Display` is not implemented for `MyStruct`
  |
  = note: in format arguments, you may be able to use `{:?}` (which uses the `Debug` trait) instead
```

Cause: Type doesn't implement the required trait.

Fix: implement the trait, derive it (`#[derive(Debug)]`), or use a different formatter (`{:?}` instead of `{}`).

### error[E0599]: no method named `X` found for type `Y`

```text
error[E0599]: no method named `pus` found for struct `Vec<i32>` in the current scope
 --> src/main.rs:3:7
  |
3 |     v.pus(1);
  |       ^^^ help: there is a method with a similar name: `push`
```

Cause: typo or missing trait import.

Fix: take the help suggestion. If the method is from a trait, `use the_trait::TheTrait;`.

### error[E0601]: `main` function not found in crate `X`

```text
error[E0601]: `main` function not found in crate `mybin`
 --> src/main.rs:1:1
  |
1 | fn run() {}
  | ^^^^^^^^^^^ consider adding a `main` function to `src/main.rs`
```

Cause: binary crate without `fn main()`.

Fix: add `fn main() { … }` to `src/main.rs`. For a library, name the file `src/lib.rs` and remove the binary build expectation.

### error[E0606]: casting `X` as `Y` is invalid

```text
error[E0606]: casting `&str` as `i32` is invalid
 --> src/main.rs:2:13
  |
2 |     let n = "42" as i32;
  |             ^^^^^^^^^^^
```

Cause: `as` only does numeric/pointer casts.

Fix: `"42".parse::<i32>().unwrap()` for strings; for safe numeric narrowing use `TryFrom`.

### error[E0658]: ... is unstable

```text
error[E0658]: use of unstable library feature 'never_type'
 --> src/main.rs:2:13
  |
2 |     let x: ! = panic!();
  |            ^
  = note: see issue #35121 <https://github.com/rust-lang/rust/issues/35121> for more information
  = help: add `#![feature(never_type)]` to the crate attributes to enable
```

Cause: feature gated to nightly.

Fix: switch to nightly toolchain (`rustup default nightly`) and add the feature attribute, or rewrite using stable alternatives.

### error[E0282]: type annotations needed

```text
error[E0282]: type annotations needed
 --> src/main.rs:2:9
  |
2 |     let v = Vec::new();
  |         ^ consider giving `v` an explicit type
```

Cause: rustc can't infer `Vec<T>`'s `T`.

Fix: annotate: `let v: Vec<i32> = Vec::new();` or `let v = Vec::<i32>::new();` or push something to constrain inference.

### error[E0284]: type annotations needed: cannot satisfy `_: Y`

```text
error[E0284]: type annotations needed: cannot satisfy `_: From<i32>`
```

Cause: ambiguous trait selection. Multiple types could satisfy.

Fix: turbofish or annotation to pin the concrete type.

### error[E0271]: type mismatch resolving `<X as Iterator>::Item == Y`

```text
error[E0271]: type mismatch resolving `<std::vec::IntoIter<i32> as Iterator>::Item == String`
```

Cause: trait associated type doesn't match.

Fix: Convert items (`.map(|i| i.to_string())`) or fix the consuming side's expectation.

### error[E0521]: borrowed data escapes outside of function

```text
error[E0521]: borrowed data escapes outside of function
 --> src/lib.rs:5:5
  |
3 | fn store(s: &str) {
  |          - `s` is a reference that is only valid in the function body
4 |     STORE.lock().unwrap().push(s);
  |     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ `s` escapes the function body here
```

Cause: putting a borrowed `&str` into a `'static` container.

Fix: `s.to_string()` to own; or change container to hold `&'static str`.

### error[E0599]: no variant or associated item named `X` found for enum `Y`

```text
error[E0599]: no variant or associated item named `Greenn` found for enum `Color`
 --> src/main.rs:3:20
  |
3 |     let c = Color::Greenn;
  |                    ^^^^^^ help: there is a variant with a similar name: `Green`
```

Cause: typo'd variant name.

Fix: use the correct variant.

## Pattern Matching Errors

### error[E0004]: non-exhaustive patterns

```text
error[E0004]: non-exhaustive patterns: `&None` not covered
 --> src/main.rs:3:11
  |
3 |     match opt {
  |           ^^^ pattern `&None` not covered
  |
  = note: the matched value is of type `&Option<i32>`
help: ensure that all possible cases are being handled by adding a match arm with a wildcard pattern, or an explicit pattern as shown
  |
5 ~         Some(n) => println!("{}", n),
6 ~         &None => todo!(),
```

Cause: `match` doesn't cover every variant.

Fix: add the missing arm, or use `_ => …` catch-all (last resort — exhaustive matching catches future variants).

### error[E0530]: match bindings cannot shadow X

```text
error[E0530]: match bindings cannot shadow constants
 --> src/main.rs:5:9
  |
2 | const X: i32 = 5;
  | -------------------- the constant `X` is defined here
...
5 |         X => println!("matched"),
  |         ^ cannot be named the same as a constant
```

Cause: a match arm binding has the same name as an existing constant; Rust treats it as the constant.

Fix: rename the binding (`x`) or the constant.

### error[E0532]: expected tuple struct or tuple variant

```text
error[E0532]: expected tuple struct or tuple variant, found struct variant `Color::Rgb`
 --> src/main.rs:3:9
  |
3 |     Color::Rgb(r, g, b) => {}
  |     ^^^^^^^^^^ help: use struct pattern syntax: `Color::Rgb { r, g, b }`
```

Cause: variant uses named fields `Rgb { r, g, b }`, not tuple `Rgb(...)`.

Fix: use struct pattern as suggested.

### error[E0596]: cannot borrow `X` as mutable, as it is not declared as mutable

```text
error[E0596]: cannot borrow `v` as mutable, as it is not declared as mutable
 --> src/main.rs:3:5
  |
2 |     let v = vec![1];
  |         - help: consider changing this to be mutable: `mut v`
3 |     v.push(2);
  |     ^^^^^^^^^ cannot borrow as mutable
```

Cause: `let v` instead of `let mut v`.

Fix: `let mut v = …`.

## Trait Errors

### error[E0277]: the trait bound `X: Y` is not satisfied (revisited)

```text
error[E0277]: the trait bound `MyType: Clone` is not satisfied
 --> src/main.rs:5:13
  |
5 |     let b = a.clone();
  |               ^^^^^ the trait `Clone` is not implemented for `MyType`
help: consider annotating `MyType` with `#[derive(Clone)]`
  |
1 + #[derive(Clone)]
2 | struct MyType { … }
```

Fix: derive or implement `Clone`. Same pattern for `Debug`, `PartialEq`, `Default`, `Hash`.

### error[E0119]: conflicting implementations of trait `X` for type `Y`

```text
error[E0119]: conflicting implementations of trait `MyTrait` for type `Foo`
 --> src/main.rs:7:1
  |
4 | impl MyTrait for Foo { … }
  | -------------------- first implementation here
...
7 | impl MyTrait for Foo { … }
  | ^^^^^^^^^^^^^^^^^^^^ conflicting implementation for `Foo`
```

Cause: two `impl` blocks overlap.

Fix: merge them; or use marker traits / type wrappers (newtype) to differentiate.

### error[E0117]: only traits defined in the current crate can be implemented for arbitrary types

```text
error[E0117]: only traits defined in the current crate can be implemented for arbitrary types
 --> src/main.rs:1:1
  |
1 | impl Display for Vec<u8> {
  | ^^^^^^^^^^^^^^^^^^^^^^^^
  | |
  | impl doesn't use only types from inside the current crate
  | `Vec` is not defined in the current crate
```

Cause: orphan rule — you can't implement an external trait on an external type.

Fix: wrap with a newtype:

```rust
struct MyBytes(Vec<u8>);
impl std::fmt::Display for MyBytes { … }
```

### error[E0220]: associated type `X` not found for `Y`

```text
error[E0220]: associated type `Item` not found for `MyIter`
```

Cause: trait bound mentions an associated type the type doesn't have.

Fix: implement the trait (or its associated type), or fix the bound.

### error[E0107]: this struct takes N generic argument but M generic arguments were supplied

```text
error[E0107]: this struct takes 1 generic argument but 2 generic arguments were supplied
 --> src/main.rs:3:13
  |
3 |     let v: Vec<i32, u8> = Vec::new();
  |            ^^^      --- help: remove this generic argument
  |            |
  |            expected 1 generic argument
```

Fix: supply the right count. (Allocator parameter is unstable.)

### error[E0277]: `X` cannot be sent between threads safely

```text
error[E0277]: `Rc<i32>` cannot be sent between threads safely
 --> src/main.rs:5:18
  |
5 |     thread::spawn(move || println!("{}", r));
  |     ------------- ^^^^^^^^^^^^^^^^^^^^^^^^^^ `Rc<i32>` cannot be sent between threads safely
  |     |
  |     required by a bound introduced by this call
  = help: the trait `Send` is not implemented for `Rc<i32>`
```

Cause: `Rc` is single-threaded; `thread::spawn` requires `Send`.

Fix: use `Arc<T>` instead. Same applies to `Cell`/`RefCell` → use `Mutex`/`RwLock`/`AtomicXxx`.

### error[E0277]: `X` cannot be shared between threads safely

```text
error[E0277]: `Cell<i32>` cannot be shared between threads safely
  = help: the trait `Sync` is not implemented for `Cell<i32>`
```

Fix: use `AtomicI32` or `Mutex<i32>`.

## async Errors

### error[E0277]: `()` is not a future

```text
error[E0277]: `()` is not a future
 --> src/main.rs:5:5
  |
5 |     do_thing().await;
  |     ^^^^^^^^^^ ----- `()` is not a future
  |     |
  |     `()` is not a future
  = help: the trait `Future` is not implemented for `()`
```

Cause: calling `.await` on a non-async function.

Fix: make the callee `async fn`, or remove the `.await`.

### error[E0277]: future cannot be sent between threads safely

```text
error[E0277]: future cannot be sent between threads safely
  --> src/main.rs:7:18
   |
7  |     tokio::spawn(do_thing());
   |     ------------ ^^^^^^^^^^^ future returned by `do_thing` is not `Send`
   = help: within `impl Future`, the trait `Send` is not implemented for `Rc<i32>`
note: future is not `Send` as this value is used across an await
```

Cause: holding `!Send` (e.g. `Rc`, `RefCell`) across `.await`.

Fix: drop the `!Send` value before awaiting (`drop(rc); call().await;`), restructure so it's in a separate scope, or use `Send` equivalents.

### error[E0728]: `await` is only allowed inside `async` functions and blocks

```text
error[E0728]: `await` is only allowed inside `async` functions and blocks
 --> src/main.rs:3:5
  |
2 | fn main() {
  | --------- this is not `async`
3 |     fetch().await;
  |     ^^^^^^^^^^^^^ only allowed inside `async` functions and blocks
```

Fix: `#[tokio::main] async fn main()` or wrap in `async { … }.await` inside a runtime, or call `block_on`.

### error[E0277]: the trait `Future` is not implemented for `X`

Cause: passing a non-future to a future-expecting position.

Fix: ensure the value is a future (`async fn` call, `async {}` block, struct implementing `Future`).

### Send-bound on spawned tasks

`tokio::spawn` (multi-threaded runtime) requires `F: Future + Send + 'static`. If your future isn't `Send`, options:

```rust
// 1. Use a current-thread runtime and tokio::task::spawn_local
let local = tokio::task::LocalSet::new();
local.spawn_local(async { … }).await;

// 2. Refactor to avoid !Send across .await

// 3. Use Arc<Mutex<T>> instead of Rc<RefCell<T>>
```

### "captured variable cannot escape `FnMut` closure body"

```text
error: captured variable cannot escape `FnMut` closure body
```

Cause: a closure with `FnMut` semantics (called multiple times) tries to move out of a capture.

Fix: use `FnOnce` (consumes the closure once) or clone inside.

## Module / Visibility Errors

### error[E0432]: unresolved import

```text
error[E0432]: unresolved import `crate::utils::helper`
 --> src/main.rs:1:5
  |
1 | use crate::utils::helper;
  |     ^^^^^^^^^^^^^^^^^^^ no `helper` in `utils`
```

Cause: typo, missing module declaration, or item not pub.

Fix: check `mod utils;` exists, the file is `src/utils.rs` or `src/utils/mod.rs`, and `helper` is `pub`.

### error[E0433]: failed to resolve: use of undeclared crate or module

```text
error[E0433]: failed to resolve: use of undeclared crate or module `serde_json`
 --> src/main.rs:1:5
  |
1 | use serde_json::Value;
  |     ^^^^^^^^^^ use of undeclared crate or module `serde_json`
```

Fix: add to `Cargo.toml`:

```toml
[dependencies]
serde_json = "1"
```

### error[E0603]: function `X` is private

```text
error[E0603]: function `inner` is private
 --> src/main.rs:3:13
  |
3 |     mymod::inner();
  |            ^^^^^ private function
note: the function `inner` is defined here
```

Fix: `pub fn inner()` in the source module, or expose via a `pub` wrapper.

### error[E0463]: can't find crate for `X`

```text
error[E0463]: can't find crate for `core`
  = note: the `wasm32-unknown-unknown` target may not be installed
  = help: consider downloading the target with `rustup target add wasm32-unknown-unknown`
```

Fix: install the target. For `core`/`std`, install the toolchain target. For external crates, ensure dependency in `Cargo.toml`.

### error[E0658]: use of unstable library feature

When stable rustc encounters a feature you used. Fix: switch to nightly + `#![feature(...)]`, or use a stable workaround.

### error[E0432]: unresolved import (typo example)

```text
error[E0432]: unresolved import `std::collections::HashMpa`
 --> src/main.rs:1:5
  |
1 | use std::collections::HashMpa;
  |     ^^^^^^^^^^^^^^^^^^^^^^^^^ help: a similar name exists in the module: `HashMap`
```

Fix: correct the typo.

## Macro Errors

### error: cannot find macro `X` in this scope

```text
error: cannot find macro `prinln` in this scope
 --> src/main.rs:2:5
  |
2 |     prinln!("hi");
  |     ^^^^^^ help: a macro with a similar name exists: `println`
```

Fix: correct typo, or `use crate::module::macro_name;` if user-defined.

### error: macro `X` is not exported

```text
error: macro `my_macro` is not exported
```

Cause: `macro_rules!` defined without `#[macro_export]`.

Fix: `#[macro_export] macro_rules! my_macro { … }` and import via `use crate::my_macro;`.

### error: expected expression, found `X`

```text
error: expected expression, found `;`
 --> src/main.rs:2:21
  |
2 |     let x = my_mac!(;);
  |                     ^
```

Cause: invalid token sequence inside macro invocation.

Fix: supply the macro's expected syntax. Use `cargo expand` to inspect.

### error: recursion limit reached while expanding `X`

```text
error: recursion limit reached while expanding `vec`
  = help: consider increasing the recursion limit by adding a `#![recursion_limit = "256"]` attribute to your crate (`mycrate`)
```

Fix: add `#![recursion_limit = "256"]` (or larger) at the crate root. Default is 128.

### macro_rules! recursion limit reached

Same as above; or rewrite the macro to be tail-recursive / iterative.

## Cargo Errors

### error: failed to compile X (lib)

```text
error: failed to compile `mycrate v0.1.0 (/path)`, intermediate artifacts can be found at `/path/target`.
```

Cause: the actual error is above this summary. Read the full output.

### error: failed to run custom build command for `X`

```text
error: failed to run custom build command for `openssl-sys v0.9.85`

Caused by:
  process didn't exit successfully: exit status: 101
  --- stderr
  thread 'main' panicked at 'Could not find directory of OpenSSL installation'
```

Cause: `build.rs` script failed.

Fix: read the stderr from the build script. For openssl-sys see the linker section.

### error: linker `cc` not found

```text
error: linker `cc` not found
  |
  = note: No such file or directory (os error 2)
```

Fix:
- Linux: `sudo apt install build-essential` or `gcc`.
- macOS: `xcode-select --install`.
- Windows: install Visual Studio Build Tools (MSVC) or use the `gnu` toolchain.

### error: could not find `Cargo.toml` in `/path` or any parent directory

Fix: cd into a Rust project, or `cargo init`.

### error: failed to download X v0.0.0

```text
error: failed to download from `https://crates.io/api/v1/crates/serde/1.0.0/download`

Caused by:
  failed to get successful HTTP response from `...`, got 503
```

Fix: retry; check proxy (`CARGO_HTTP_PROXY`), check network. For corp networks try `cargo --offline` if cache has it.

### error: could not find `Cargo.lock` for the workspace

Fix: `cargo generate-lockfile` or run any cargo command that produces it (`cargo build`, `cargo check`).

### error: the lock file ... needs to be updated but --locked was passed to prevent this

```text
error: the lock file /path/Cargo.lock needs to be updated but --locked was passed to prevent this
If you want to try to generate the lock file without accessing the network, remove the --locked flag and use --offline instead.
```

Cause: `Cargo.toml` changed; lock file out of date; CI uses `--locked`.

Fix: locally `cargo update -p <pkg>` (or `cargo build`), commit `Cargo.lock`. In CI, ensure `Cargo.lock` is in version control and matches.

### error: failed to select a version for the requirement `X`

```text
error: failed to select a version for the requirement `tokio = "^99.0"`
candidate versions found which didn't match: 1.32.0, 1.31.0, ...
```

Fix: pick an actual published version. `cargo search tokio`.

### error: package `X` cannot be tested because it requires dev-dependencies

Fix: add the dev-deps to `[dev-dependencies]` in `Cargo.toml`.

### error: target may not be specified in this context

Fix: typically a `Cargo.toml` mistake — `target.cfg(...).dependencies` in wrong scope.

### error: package `X` v0.0.0 cannot be built because it requires rustc Y.Z

```text
error: package `tokio v1.30.0` cannot be built because it requires rustc 1.63 or newer, while the currently active rustc version is 1.60.0
```

Fix: `rustup update stable`. Or pin an older crate version.

### error: feature `X` is required

```text
error[E0432]: unresolved import `serde_json::from_str`
  = note: enable the `std` feature in `Cargo.toml`
```

Fix: enable the feature:

```toml
serde_json = { version = "1", features = ["std"] }
```

## Linker / Build Errors

### error: linking with `cc` failed: exit status: 1

```text
error: linking with `cc` failed: exit status: 1
  |
  = note: ld: library not found for -lssl
          clang: error: linker command failed with exit code 1
```

Cause: missing native library.

Fix: install the system library (`apt install libssl-dev`, `brew install openssl`).

### ld: library not found for -l<X>

Fix: install the library. For `openssl`, either install via system pkg manager or use rustls (a pure-Rust alternative).

### undefined reference to `X`

```text
undefined reference to `pthread_atfork'
```

Cause: missing link flag.

Fix: `build.rs` with `println!("cargo:rustc-link-lib=pthread");` or set `RUSTFLAGS="-C link-arg=-lpthread"`.

### error: failed to run custom build command for `openssl-sys v0.0.0`

The classic OpenSSL on macOS/Windows. Workaround:

```bash
# macOS
export OPENSSL_DIR=$(brew --prefix openssl@3)
export OPENSSL_LIB_DIR=$OPENSSL_DIR/lib
export OPENSSL_INCLUDE_DIR=$OPENSSL_DIR/include

# or use vendored
```

```toml
[dependencies]
openssl = { version = "0.10", features = ["vendored"] }
# Or switch ecosystem:
reqwest = { version = "0.11", default-features = false, features = ["rustls-tls"] }
```

## Cross-Compilation Errors

### error[E0463]: can't find crate for `core`

```text
error[E0463]: can't find crate for `core`
  = note: the `aarch64-unknown-linux-gnu` target may not be installed
  = help: consider downloading the target with `rustup target add aarch64-unknown-linux-gnu`
```

Fix: `rustup target add <triple>`.

### error: linker `aarch64-linux-gnu-gcc` not found

Fix: install the cross linker (`apt install gcc-aarch64-linux-gnu`) and configure `.cargo/config.toml`:

```toml
[target.aarch64-unknown-linux-gnu]
linker = "aarch64-linux-gnu-gcc"
```

### error: failed to find tool. Is `cc` installed?

Cause: build.rs scripts need a C compiler.

Fix: install one for the target. See the `cc` crate docs.

### the cross crate workaround

[cross](https://github.com/cross-rs/cross) wraps cargo with Docker images that pre-install cross toolchains:

```bash
cargo install cross --git https://github.com/cross-rs/cross
cross build --target aarch64-unknown-linux-gnu --release
```

## Clippy Warnings

Run with `cargo clippy -- -W clippy::pedantic` to see more.

### clippy::missing_safety_doc

```text
warning: docstring for unsafe trait `Foo` does not have a `# Safety` section
```

Fix: add `/// # Safety` doc comment explaining the invariants the caller must uphold.

### clippy::needless_borrow

```text
warning: this expression borrows a value the compiler would automatically borrow
   |
   |     foo(&v.bar());
   |         ^^^^^^^^^ help: change this to: `v.bar()`
```

Fix: drop the `&`.

### clippy::single_match

```text
warning: you seem to be trying to use `match` for an equality check. Consider using `if`
help: try
   |
   |     if let Some(x) = opt { println!("{x}") }
```

Fix: use `if let` for one-arm matches.

### clippy::collapsible_match

Two nested `match` expressions where the outer's bound name is matched again. Fix: combine into one with a nested pattern.

### clippy::or_fun_call

```text
warning: use of `unwrap_or` followed by a function call
help: try this: `.unwrap_or_else(|| compute())`
```

Cause: `unwrap_or(compute())` evaluates `compute()` even on `Some/Ok`.

Fix: `unwrap_or_else(|| compute())` for lazy evaluation.

### clippy::needless_collect

```text
warning: avoid using `collect()` when not needed
   |     let v: Vec<_> = it.collect();
   |     for x in v { … }
```

Fix: iterate directly: `for x in it { … }`.

### clippy::redundant_clone

```text
warning: redundant clone
   |     let s = s.clone();
help: this clone is unnecessary
```

Fix: drop the `.clone()`.

### clippy::expect_used / clippy::unwrap_used

Deny in production code:

```toml
[lints.clippy]
unwrap_used = "deny"
expect_used = "deny"
```

Fix: handle the error properly with `?`, `match`, or `unwrap_or`.

### clippy::too_many_arguments

```text
warning: this function has too many arguments (8/7)
```

Fix: bundle into a struct, builder, or break the function up.

### clippy::cyclomatic_complexity

```text
warning: the function has a cyclomatic complexity of (15/10)
```

Fix: extract helpers, replace nested ifs with early returns or table-driven dispatch.

### clippy::shadow_unrelated

```text
warning: `x` is shadowed by `x` and they have unrelated types
```

Fix: pick distinct names. Shadowing is fine when types are related (transformations).

## Compiler Suggestions

Rust's compiler is unusually proactive. Patterns to recognise:

- **"consider adding a `pub`"** — item private; expose it.
- **"consider adding a lifetime parameter"** — function returns a borrow but signature can't infer source.
- **"consider using `?`"** — simplify `match` on `Result`.
- **"consider borrowing here: `&X`"** — value moved when reference would suffice.
- **"consider boxing the type"** — recursive type with infinite size; wrap in `Box<T>`.
- **"consider importing one of these items"** — missing `use`.
- **"help: a method with this name exists in trait Y; consider importing it"** — trait method not in scope.

```text
help: there is a method with a similar name
   |
   |     v.lenght
   |       ^^^^^^ help: there is a method with a similar name: `len`
```

Take the suggestion almost always.

## Test Errors

### error: test failed, to rerun pass `--lib`

```text
error: test failed, to rerun pass `--lib`

Caused by:
  process didn't exit successfully: ... (exit status: 101)
```

Fix: read the test output above. Run `cargo test --lib -- --nocapture` to see println, or `cargo test name_of_test -- --exact --nocapture`.

### thread 'main' panicked at '...'

```text
thread 'main' panicked at 'index out of bounds: the len is 3 but the index is 5', src/main.rs:6:13
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
```

Fix: `RUST_BACKTRACE=1 cargo run` for stack trace; `RUST_BACKTRACE=full` for full frames.

### thread 'tests::X' panicked at 'assertion failed'

```text
thread 'tests::add_works' panicked at 'assertion `left == right` failed
  left: 5
 right: 4', src/lib.rs:12:5
```

Fix: read the values; fix the code or the assertion.

### tests run failed: signal: 6, SIGABRT

A test hit `abort()` (panic with `panic = "abort"`, or C code aborting). Run with backtrace.

## unsafe Errors

### error[E0133]: call to unsafe function is unsafe

```text
error[E0133]: call to unsafe function is unsafe and requires unsafe function or block
 --> src/main.rs:3:5
  |
3 |     dangerous();
  |     ^^^^^^^^^^^ call to unsafe function
```

Fix: wrap in `unsafe { dangerous(); }` (and document why it's safe).

### error[E0133]: dereference of raw pointer is unsafe

```text
error[E0133]: dereference of raw pointer is unsafe and requires unsafe function or block
 --> src/main.rs:4:13
  |
4 |     let v = *p;
  |             ^^ dereference of raw pointer
```

Fix: `unsafe { *p }`. Verify the pointer is valid, aligned, non-null, and the data is initialized.

### error[E0133]: use of mutable static is unsafe

```text
error[E0133]: use of mutable static is unsafe and requires unsafe function or block
```

Fix: prefer `OnceCell`, `OnceLock`, `LazyLock`, or `Mutex<T>` instead. `static mut` is nearly always wrong.

### error[E0133]: access to union field is unsafe

```text
error[E0133]: access to union field is unsafe and requires unsafe function or block
```

Fix: `unsafe { u.field }`. Ensure you read the variant that was last written.

## const Errors

### error[E0080]: evaluation of constant value failed

```text
error[E0080]: evaluation of constant value failed
 --> src/main.rs:2:23
  |
2 | const X: u32 = 1 / 0;
  |                ^^^^^ attempt to divide `1_u32` by zero
```

Fix: don't divide by zero (compiler caught it at const-eval time).

### error[E0019]: constant contains unimplemented expression type

Cause: feature in const that isn't supported yet (heap allocation, trait dispatch, etc.).

Fix: rewrite to use only const-supported constructs, or move out of const context.

### error[E0277]: the trait `const_eval` is not implemented for X

Cause: const fn uses non-const trait methods.

Fix: use `const fn`-compatible operations only; await further stabilization or use `lazy_static!` / `OnceCell` for runtime-initialized statics.

### error[E0658]: trait methods cannot be stably called on const traits

Fix: switch to nightly with `#![feature(const_trait_impl)]`, or evaluate at runtime.

## Std Library Errors

### thread 'main' panicked at 'index out of bounds: the len is N but the index is M'

```text
thread 'main' panicked at 'index out of bounds: the len is 3 but the index is 5', src/main.rs:6:13
```

Fix: bounds check (`if i < v.len()`) or use `.get(i)` which returns `Option<&T>`.

### thread 'main' panicked at 'attempt to subtract with overflow'

```text
thread 'main' panicked at 'attempt to subtract with overflow', src/main.rs:4:13
```

Cause: in debug, integer overflow panics; in release, it wraps (silently!).

Fix: use checked/saturating/wrapping methods explicitly:

```rust
a.checked_sub(b).ok_or("underflow")?;
a.saturating_sub(b);
a.wrapping_sub(b);
```

### thread 'main' panicked at 'attempt to divide by zero'

Fix: guard with `if b != 0` or use `checked_div`.

### thread 'main' panicked at 'called `Result::unwrap()` on an `Err` value: ...'

```text
thread 'main' panicked at 'called `Result::unwrap()` on an `Err` value: Os { code: 2, kind: NotFound, message: "No such file or directory" }', src/main.rs:3:34
```

Fix: handle the error: `?`, `match`, `unwrap_or_else`.

### thread 'main' panicked at 'called `Option::unwrap()` on a `None` value'

```text
thread 'main' panicked at 'called `Option::unwrap()` on a `None` value', src/main.rs:5:21
```

Fix: pattern match or `unwrap_or` / `ok_or` to convert to `Result`.

### thread 'main' panicked at 'capacity overflow'

Cause: `Vec::with_capacity(usize::MAX)` or similar absurd capacity.

Fix: provide a reasonable capacity.

### thread 'X' panicked at 'PoisonError ...'

```text
thread 'main' panicked at 'PoisonError { .. }: Mutex poisoned'
```

Cause: another thread panicked while holding the mutex; Rust marks it poisoned.

Fix: `lock().unwrap_or_else(|e| e.into_inner())` to recover, or fix the panicking thread.

## tokio / async-runtime Errors

### thread 'main' panicked at 'there is no reactor running, must be called from the context of a Tokio 1.x runtime'

Cause: calling `tokio::spawn` / awaiting a tokio future outside a runtime.

Fix: use `#[tokio::main]` or `tokio::runtime::Runtime::new()?.block_on(async { … })`.

### panicked at 'Cannot start a runtime from within a runtime'

Cause: calling `block_on` from within an async context.

Fix: just `.await` directly. If you must bridge sync/async, use `tokio::task::block_in_place`.

### JoinError

```text
JoinError::Panic(...) — the spawned task panicked
JoinError::Cancelled  — the task was aborted
```

Fix: handle the join: `match task.await { Ok(v) => …, Err(e) if e.is_panic() => … }`.

### #[tokio::main] macro requirement

```rust
#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    do_async().await
}
```

Without it, `main` can't be `async`.

## serde Errors

### error: missing field `X` at line N column M

```text
Error("missing field `name`", line: 3, column: 1)
```

Fix: include the field, mark optional with `Option<String>` and `#[serde(default)]`, or rename/alias.

### error: invalid type: integer `X`, expected a string

```text
Error("invalid type: integer `42`, expected a string", line: 2, column: 9)
```

Fix: fix the JSON, or use `#[serde(deserialize_with = "...")]` to coerce.

### error: trailing characters at line N column M

```text
Error("trailing characters", line: 5, column: 1)
```

Cause: invalid JSON (extra commas, comments, etc.) — JSON proper doesn't allow either.

Fix: clean the input, or use `serde_json5` for relaxed JSON.

### error: data did not match any variant of untagged enum X

```text
Error("data did not match any variant of untagged enum MyEnum", line: 0, column: 0)
```

Cause: untagged enums fall through every variant.

Fix: check input. Switch to tagged representation if practical.

### Error("missing field `X`", line: N, column: M)

Same as the first; with `#[serde(default)]` or `Option<T>` it becomes optional.

## Common Warnings That Are Errors-In-Disguise

### warning: unused import: X

```text
warning: unused import: `std::io::Write`
 --> src/main.rs:1:5
  |
1 | use std::io::Write;
  |     ^^^^^^^^^^^^^^
```

Fix: remove the import. With `-D warnings` (CI), this fails the build.

### warning: unused variable: `X` → prefix _X to silence

```text
warning: unused variable: `result`
 --> src/main.rs:3:9
  |
3 |     let result = compute();
  |         ^^^^^^ help: if this is intentional, prefix it with an underscore: `_result`
```

Fix: prefix with `_` if intentional, otherwise consume the value.

### warning: variable does not need to be mutable

```text
warning: variable does not need to be mutable
 --> src/main.rs:2:9
  |
2 |     let mut x = 5;
  |         ----^
  |         |
  |         help: remove this `mut`
```

Fix: drop `mut`.

### warning: function `X` is never used

```text
warning: function `helper` is never used
```

Fix: delete the function, or `#[allow(dead_code)]` if it's intentionally there (testing, future use).

### -D warnings escalation in CI

Many projects build with `RUSTFLAGS="-D warnings"`. Every warning becomes an error. Treat warnings as errors locally with:

```bash
cargo clippy -- -D warnings
RUSTFLAGS="-D warnings" cargo build
```

## Common Gotchas

### Cloning to satisfy borrow checker when ownership rethink would be better

Broken:
```rust
fn process(v: Vec<i32>) {
    do_thing(v.clone()); // why clone?
    do_other(v.clone());
}
```
Fixed:
```rust
fn process(v: &[i32]) {
    do_thing(v);
    do_other(v);
}
```

### .unwrap() in production code

Broken:
```rust
let user = db.find_user(id).unwrap();
```
Fixed:
```rust
let user = db.find_user(id)?; // or .ok_or(MyError::NotFound)?
```

### String vs &str confusion

Broken:
```rust
fn greet(name: String) { println!("hi {}", name); }
greet(my_string.clone()); // wasteful clone
```
Fixed:
```rust
fn greet(name: &str) { println!("hi {}", name); }
greet(&my_string);
greet("literal");
```

Use `&str` for parameters unless you need ownership; use `String` for return values where the caller wants ownership.

### Vec<i32> vs &[i32] in function parameters

Broken:
```rust
fn sum(v: Vec<i32>) -> i32 { v.iter().sum() }
```
Fixed:
```rust
fn sum(v: &[i32]) -> i32 { v.iter().sum() }
// Now accepts &Vec<i32>, &[i32], or &[i32; N]
```

### println! capturing borrows in non-obvious ways

Broken:
```rust
let s = String::from("hi");
let r = &s;
let m = &mut s; // E0502
println!("{} {}", r, m);
```

Fixed: end the immutable borrow first, or restructure.

### Returning &str from function that owns String

Broken:
```rust
fn name() -> &str {
    let s = String::from("Alice");
    &s // E0515: returns reference to local
}
```
Fixed:
```rust
fn name() -> String { String::from("Alice") }
fn name() -> &'static str { "Alice" } // string literal
```

### Closure capturing by reference when move was needed

Broken:
```rust
fn spawn_task<F: FnOnce()>(_: F) { /* ... */ }
let s = String::from("hi");
spawn_task(|| println!("{}", s)); // captures &s
drop(s); // borrow check error
```
Fixed:
```rust
spawn_task(move || println!("{}", s));
```

### Mutex poisoning ignored

Broken:
```rust
let g = mutex.lock().unwrap(); // panics if poisoned
```
Fixed:
```rust
let g = mutex.lock().unwrap_or_else(|e| e.into_inner());
```

Or use `parking_lot::Mutex` which doesn't poison.

### Async fn returning Future that doesn't outlive caller

Broken:
```rust
fn run<'a>(s: &'a str) -> impl Future<Output = ()> + 'a {
    async move { println!("{}", s); }
}
let fut = run(&temp_string()); // temp dropped
```
Fixed: own the data, or ensure source outlives the future.

### Box<dyn Error> vs concrete type

Broken: throwing `Box<dyn Error>` everywhere loses information.

Fixed for libraries: define a concrete error enum with `thiserror`:

```rust
#[derive(thiserror::Error, Debug)]
pub enum MyError {
    #[error("io: {0}")]
    Io(#[from] std::io::Error),
    #[error("parse: {0}")]
    Parse(#[from] std::num::ParseIntError),
}
```

Fixed for applications: `anyhow::Result<T>` for ergonomic catch-all.

### thread::spawn requiring 'static + Send

Broken:
```rust
let data = vec![1, 2, 3];
let r = &data;
thread::spawn(move || println!("{:?}", r)); // r is &Vec, not 'static
```
Fixed:
```rust
let data = vec![1, 2, 3];
thread::spawn(move || println!("{:?}", data)); // owns data

// Or scoped threads (no 'static needed)
thread::scope(|s| {
    s.spawn(|| println!("{:?}", &data));
});
```

### static mut requiring unsafe (now nearly always wrong)

Broken:
```rust
static mut COUNT: u32 = 0;
unsafe { COUNT += 1; } // race conditions galore
```
Fixed:
```rust
use std::sync::OnceLock;
use std::sync::atomic::{AtomicU32, Ordering};

static COUNT: AtomicU32 = AtomicU32::new(0);
COUNT.fetch_add(1, Ordering::Relaxed);

// Or for non-trivial init:
static CONFIG: OnceLock<Config> = OnceLock::new();
let cfg = CONFIG.get_or_init(|| load_config());
```

## Debugging

```bash
# Stack traces on panic
RUST_BACKTRACE=1 cargo run
RUST_BACKTRACE=full cargo run     # full frames including std

# Logging
RUST_LOG=debug cargo run          # with env_logger
RUST_LOG=mycrate=trace cargo run  # filter by module
```

```rust
// env_logger
fn main() {
    env_logger::init();
    log::info!("starting");
}

// tracing
fn main() {
    tracing_subscriber::fmt::init();
    tracing::info!(user_id = 42, "logged in");
}
```

```bash
# Macro expansion
cargo install cargo-expand
cargo expand                       # whole crate
cargo expand path::to::module      # one module

# Codegen inspection
cargo install cargo-asm
cargo asm crate::function::name

# Binary size
cargo install cargo-bloat
cargo bloat --release --crates
cargo bloat --release -n 10        # top 10 functions

# Flamegraph (perf-based)
cargo install flamegraph
cargo flamegraph --bin myapp
# produces flamegraph.svg

# Disassembly via objdump
cargo build --release
objdump -d target/release/myapp | less

# rust-gdb / rust-lldb (pretty-prints stdlib types)
rust-gdb target/debug/myapp
rust-lldb target/debug/myapp

# Miri — UB checker (nightly)
rustup +nightly component add miri
cargo +nightly miri test

# Sanitizers (nightly)
RUSTFLAGS="-Z sanitizer=address" cargo +nightly run

# Verbose cargo
cargo build --verbose
cargo build -vv                    # very verbose, including build script output

# Faster iteration
cargo check                        # type-check, no codegen
cargo test --no-run                # compile tests, don't run
cargo test -- --nocapture          # show println in tests
cargo test name -- --exact         # run only "name" test
```

## NLL (Non-Lexical Lifetimes)

Pre-2018 (lexical lifetimes): borrows lasted from declaration to end of block. The 2018 edition introduced **NLL**: a borrow ends at its last *use*.

This used to error:

```rust
let mut v = vec![1, 2, 3];
let r = &v[0];
println!("{}", r);
v.push(4); // pre-NLL: error, r still in scope to end of block
```

Now allowed because `r`'s last use is the `println!` line.

The Polonius next-gen borrow checker is in development for even smarter analysis (still nightly). Most codebases written in 2015 edition style can simply be migrated:

```bash
cargo fix --edition          # update Cargo.toml edition
cargo fix --edition-idioms   # apply idiom updates
```

Common 2015→2018 migrations:
- `extern crate foo;` → just use `foo::…` (path syntax).
- `use crate::…` instead of bare `use …`.
- Async/await syntax (2018 edition only).
- Module path: `super::super::` chains often shortened.

## Idioms

### Result<T, E> for fallible operations

```rust
fn parse_port(s: &str) -> Result<u16, ParseIntError> {
    s.trim().parse()
}
```

### ? for short-circuit

```rust
fn read_port() -> Result<u16, Box<dyn std::error::Error>> {
    let s = std::fs::read_to_string("port.txt")?;
    let n = s.trim().parse::<u16>()?;
    Ok(n)
}
```

`?` chains through any `From` impl, so different error types compose.

### thiserror for library errors

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum DbError {
    #[error("connection failed: {0}")]
    Connect(#[from] std::io::Error),
    #[error("query timed out after {0:?}")]
    Timeout(std::time::Duration),
    #[error("not found: {id}")]
    NotFound { id: u64 },
}
```

### anyhow for application errors

```rust
use anyhow::{Context, Result};

fn run() -> Result<()> {
    let s = std::fs::read_to_string("config.toml")
        .context("reading config.toml")?;
    let cfg: Config = toml::from_str(&s)
        .context("parsing config.toml")?;
    Ok(())
}
```

`anyhow::Error` is `Box<dyn Error>` with backtrace and context chains.

### From impl pattern for error conversion

```rust
impl From<std::io::Error> for MyError {
    fn from(e: std::io::Error) -> Self { MyError::Io(e) }
}
```

Then `?` converts io::Error → MyError automatically. `thiserror`'s `#[from]` does this for you.

### Never `panic!()` in libraries

Libraries should return `Result`. Reserve panics for invariant violations the *caller* could verify ahead of time (out-of-bounds index, NaN where ordered values expected). Document any panic conditions.

```rust
/// Returns the element at `index`.
///
/// # Panics
///
/// Panics if `index >= len()`.
pub fn at(&self, index: usize) -> &T { … }
```

## See Also

- rust
- cargo
- polyglot
- troubleshooting/python-errors
- troubleshooting/go-errors
- troubleshooting/javascript-errors

## References

- doc.rust-lang.org/error-index.html — every E-code with examples and explanations
- doc.rust-lang.org/book — The Rust Programming Language
- doc.rust-lang.org/nomicon — The Rustonomicon (unsafe Rust)
- rust-lang.github.io/rust-clippy/master — every clippy lint with what/why
- github.com/rust-lang/rust/tree/master/compiler/rustc_error_codes/src/error_codes — error code source explanations
- doc.rust-lang.org/std — standard library reference
- doc.rust-lang.org/cargo — cargo reference
- rust-lang.github.io/api-guidelines — API design checklist (including error guidelines)
- rust-lang.github.io/async-book — async programming in Rust
- tokio.rs/tokio/topics — tokio runtime docs
- serde.rs — serde framework docs
