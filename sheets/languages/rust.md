# Rust (Programming Language)

Systems language with ownership-based memory safety, zero-cost abstractions, and no garbage collector.

## Ownership & Borrowing

### Ownership rules

```bash
# let s1 = String::from("hello");
# let s2 = s1;            // s1 is MOVED to s2 — s1 is no longer valid
# let s3 = s2.clone();    // deep copy — both s2 and s3 are valid
```

### Borrowing (references)

```bash
# fn len(s: &String) -> usize { s.len() }  // immutable borrow
# fn push(s: &mut String) { s.push('!'); }  // mutable borrow
#
# let s = String::from("hello");
# let r1 = &s;            // ok: multiple immutable borrows
# let r2 = &s;            // ok
# // let r3 = &mut s;     // ERROR: cannot mutably borrow while immutably borrowed
```

### Lifetimes

```bash
# fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
#     if x.len() > y.len() { x } else { y }
# }
# // 'a means: the returned reference lives at least as long as the shorter input
#
# struct Config<'a> {
#     name: &'a str,       // struct borrows data — must not outlive source
# }
```

## Types

### Primitives

```bash
# let x: i32 = 42;        // also i8, i16, i64, i128, isize
# let y: u64 = 100;       // also u8, u16, u32, u128, usize
# let f: f64 = 3.14;      // also f32
# let b: bool = true;
# let c: char = 'A';      // 4 bytes, Unicode scalar
# let t: (i32, f64) = (1, 2.0);  // tuple
# let arr: [i32; 5] = [1, 2, 3, 4, 5];
```

### Strings

```bash
# let s: &str = "hello";               // string slice (borrowed, on stack/binary)
# let s: String = String::from("hello"); // owned, heap-allocated
# let s = format!("{} {}", "hello", "world");
# s.push_str(" world");
# s.len(), s.is_empty(), s.contains("ell")
# let slice: &str = &s[0..5];
# for c in s.chars() { ... }
```

## Structs

```bash
# struct User {
#     name: String,
#     email: String,
#     active: bool,
# }
# impl User {
#     fn new(name: &str, email: &str) -> Self {
#         Self { name: name.into(), email: email.into(), active: true }
#     }
#     fn domain(&self) -> &str {
#         self.email.split('@').last().unwrap()
#     }
# }
# let u = User::new("Alice", "alice@example.com");
```

## Enums

```bash
# enum Shape {
#     Circle(f64),
#     Rectangle(f64, f64),
#     Triangle { base: f64, height: f64 },
# }
# fn area(s: &Shape) -> f64 {
#     match s {
#         Shape::Circle(r) => std::f64::consts::PI * r * r,
#         Shape::Rectangle(w, h) => w * h,
#         Shape::Triangle { base, height } => 0.5 * base * height,
#     }
# }
```

## Traits

```bash
# trait Summary {
#     fn summarize(&self) -> String;
#     fn preview(&self) -> String {           // default implementation
#         format!("{}...", &self.summarize()[..20])
#     }
# }
# impl Summary for User {
#     fn summarize(&self) -> String {
#         format!("{} <{}>", self.name, self.email)
#     }
# }
# fn notify(item: &impl Summary) { ... }     // trait bound shorthand
# fn notify<T: Summary + Display>(item: &T) { ... }  // multiple bounds
# fn notify(item: &(dyn Summary + Send)) { ... }      // trait object
```

### Common derive traits

```bash
# #[derive(Debug, Clone, PartialEq, Eq, Hash, Default, Serialize, Deserialize)]
# struct Config { ... }
```

## Option & Result

### Option

```bash
# let x: Option<i32> = Some(42);
# let y: Option<i32> = None;
# x.unwrap()                // panics on None
# x.unwrap_or(0)            // default value
# x.unwrap_or_else(|| compute())
# x.map(|v| v * 2)          // Some(84)
# x.and_then(|v| if v > 0 { Some(v) } else { None })
# if let Some(val) = x { ... }
# let val = x?;             // propagate None in functions returning Option
```

### Result

```bash
# fn read_file(path: &str) -> Result<String, io::Error> {
#     let content = fs::read_to_string(path)?;  // ? propagates Err
#     Ok(content)
# }
# match result {
#     Ok(val) => println!("{val}"),
#     Err(e) => eprintln!("error: {e}"),
# }
# result.unwrap_or_default()
# result.map_err(|e| CustomError::from(e))
```

### Error handling pattern

```bash
# use thiserror::Error;
# #[derive(Error, Debug)]
# enum AppError {
#     #[error("io error: {0}")]
#     Io(#[from] io::Error),
#     #[error("parse error: {0}")]
#     Parse(#[from] serde_json::Error),
#     #[error("not found: {0}")]
#     NotFound(String),
# }
# type Result<T> = std::result::Result<T, AppError>;
```

## Iterators

```bash
# let v = vec![1, 2, 3, 4, 5];
# v.iter().map(|x| x * 2).collect::<Vec<_>>()
# v.iter().filter(|&&x| x > 2).collect::<Vec<_>>()
# v.iter().find(|&&x| x == 3)
# v.iter().any(|&x| x > 4)
# v.iter().all(|&x| x > 0)
# v.iter().sum::<i32>()
# v.iter().enumerate()                     // (index, &value) pairs
# v.iter().zip(other.iter())
# v.iter().flat_map(|x| vec![x, x])
# v.iter().fold(0, |acc, &x| acc + x)
# v.chunks(2)                              // iterate in chunks
# v.windows(3)                             // sliding window
```

## Cargo

```bash
cargo new myproject                        # new binary project
cargo new mylib --lib                      # new library
cargo build                                # debug build
cargo build --release                      # optimized build
cargo run                                  # build and run
cargo run -- arg1 arg2                     # pass args
cargo test                                 # run tests
cargo test -- --nocapture                  # show println output
cargo test test_name                       # run specific test
cargo clippy                               # lints
cargo fmt                                  # format code
cargo doc --open                           # generate and open docs
cargo add serde --features derive          # add dependency
cargo update                               # update deps
cargo bench                                # run benchmarks
```

## Common Patterns

### Builder pattern

```bash
# struct ServerBuilder { port: u16, host: String }
# impl ServerBuilder {
#     fn new() -> Self { Self { port: 8080, host: "0.0.0.0".into() } }
#     fn port(mut self, port: u16) -> Self { self.port = port; self }
#     fn host(mut self, host: &str) -> Self { self.host = host.into(); self }
#     fn build(self) -> Server { Server { port: self.port, host: self.host } }
# }
```

### Generics

```bash
# fn largest<T: PartialOrd>(list: &[T]) -> &T {
#     let mut max = &list[0];
#     for item in &list[1..] {
#         if item > max { max = item; }
#     }
#     max
# }
```

## Tips

- The borrow checker is strict but prevents data races and use-after-free at compile time.
- Use `&str` for function parameters, `String` for owned data stored in structs.
- `clone()` is explicit and visible in the source, unlike implicit copies in other languages.
- Prefer `?` over `.unwrap()` for error handling. Reserve `.unwrap()` for tests and cases you can prove never fail.
- `cargo clippy` catches far more issues than the compiler alone. Run it in CI.
- `#[derive(Debug)]` on all structs. You will need it for error messages.
- `impl From<X> for Y` enables the `?` operator and `.into()` conversions automatically.
- `Vec`, `HashMap`, and `String` allocate on the heap. Slices (`&[T]`, `&str`) are just views.

## See Also

- c, go, toml, make, cargo, typescript, webassembly

## References

- [The Rust Book](https://doc.rust-lang.org/book/) -- official tutorial and guide
- [Rust Standard Library](https://doc.rust-lang.org/std/) -- stdlib API reference
- [Rust Reference](https://doc.rust-lang.org/reference/) -- language syntax and semantics reference
- [Rust by Example](https://doc.rust-lang.org/rust-by-example/) -- learn Rust through annotated examples
- [The Cargo Book](https://doc.rust-lang.org/cargo/) -- package manager and build system guide
- [crates.io](https://crates.io/) -- Rust package registry
- [docs.rs](https://docs.rs/) -- auto-generated documentation for every crate
- [Rust Edition Guide](https://doc.rust-lang.org/edition-guide/) -- edition differences (2015, 2018, 2021, 2024)
- [Rustonomicon](https://doc.rust-lang.org/nomicon/) -- guide to unsafe Rust and advanced topics
- [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/) -- conventions for writing idiomatic Rust APIs
- [Rust Playground](https://play.rust-lang.org/) -- run and share Rust code online
- [This Week in Rust](https://this-week-in-rust.org/) -- weekly newsletter of Rust ecosystem updates
