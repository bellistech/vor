# The Internals of Rust — Ownership, Lifetimes, and Zero-Cost Abstractions

> *Rust's type system encodes memory safety at compile time through an affine type system (ownership), a borrow checker (shared/exclusive references), and lifetime annotations as type-level constraints. Zero-cost abstractions are achieved through monomorphization — generics are compiled to concrete types with no runtime dispatch.*

---

## 1. Ownership as an Affine Type System

### Affine Types

In an **affine type system**, each value can be used **at most once**. Rust enforces this: every value has exactly one owner, and when that owner goes out of scope, the value is dropped.

$$\text{Linear}: \text{use exactly once} \quad \text{Affine}: \text{use at most once}$$

Rust is affine (not linear) because you can drop a value without using it.

### Move Semantics

Assignment transfers ownership (a **move**):

```rust
let s1 = String::from("hello");
let s2 = s1;        // s1 is MOVED to s2
// s1 is now invalid — compile error if used
```

After a move, the source binding is **statically invalidated**. No runtime check.

### The Copy Trait

Types implementing `Copy` are duplicated on assignment instead of moved. Requirements:
- Type must be trivially copyable (memcpy-safe)
- All fields must also be `Copy`
- Cannot implement `Drop` (custom destructor)

| Copy Types | Non-Copy Types |
|:-----------|:---------------|
| `i32`, `f64`, `bool`, `char` | `String`, `Vec<T>`, `Box<T>` |
| `(i32, i32)` | `(i32, String)` |
| `[i32; 5]` | `[String; 5]` |
| `&T` (shared references) | `&mut T` (mutable references) |

### Drop Order

Values are dropped in **reverse declaration order** within a scope. Struct fields are dropped in **declaration order**:

```rust
{
    let a = Foo::new();  // dropped 3rd
    let b = Bar::new();  // dropped 2nd
    let c = Baz::new();  // dropped 1st
}
```

---

## 2. The Borrow Checker

### Two Reference Types

| Reference | Syntax | Aliasing | Mutation | Analogy |
|:----------|:------:|:--------:|:--------:|:--------|
| Shared | `&T` | Multiple | No | Read lock (RwLock::read) |
| Mutable | `&mut T` | Exclusive | Yes | Write lock (RwLock::write) |

### The Fundamental Rule

At any given time, you can have **either**:
- Any number of shared references `&T`, **or**
- Exactly one mutable reference `&mut T`

**Never both simultaneously.** This is the **readers-writers invariant** enforced at compile time.

Formally, for a value `x` at any program point:

$$\text{shared}(x) > 0 \implies \text{mutable}(x) = 0$$
$$\text{mutable}(x) = 1 \implies \text{shared}(x) = 0$$

### Non-Lexical Lifetimes (NLL)

Since Rust 2018, the borrow checker uses **non-lexical lifetimes**: a borrow ends at the last use, not at the end of the scope.

```rust
let mut v = vec![1, 2, 3];
let r = &v[0];       // shared borrow starts
println!("{}", r);   // shared borrow ENDS here (last use)
v.push(4);           // mutable borrow OK — no conflict
```

---

## 3. Lifetime Annotations as Type Constraints

### What Lifetimes Are

A lifetime `'a` is a **constraint on how long a reference is valid**. It's part of the type:

$$\text{\&'a T} \neq \text{\&'b T} \quad \text{(different types if } a \neq b\text{)}$$

### The Subtyping Rule

Lifetime `'a` is a subtype of `'b` if `'a` **outlives** `'b`:

$$'a : 'b \iff 'a \supseteq 'b$$

A longer-lived reference can be used where a shorter-lived one is expected (covariance).

### Lifetime Elision Rules

The compiler infers lifetimes in function signatures using three rules:

1. Each reference parameter gets its own lifetime: `fn f(x: &T, y: &T)` → `fn f<'a, 'b>(x: &'a T, y: &'b T)`
2. If there's exactly one input lifetime, it's assigned to all output lifetimes
3. If one parameter is `&self` or `&mut self`, its lifetime is assigned to all output lifetimes

### Worked Example

```rust
// The compiler sees:
fn first_word(s: &str) -> &str { ... }

// After elision (rule 1 + rule 2):
fn first_word<'a>(s: &'a str) -> &'a str { ... }

// Meaning: returned reference lives as long as input
```

### Variance Table

| Type Constructor | Variance in `'a` | Variance in `T` |
|:-----------------|:-----------------|:----------------|
| `&'a T` | Covariant | Covariant |
| `&'a mut T` | Covariant | **Invariant** |
| `fn(T) -> U` | — | **Contravariant** in T, covariant in U |
| `Cell<T>` | — | **Invariant** |

Invariance of `&mut T` prevents a `&mut Vec<&'static str>` from being used as `&mut Vec<&'short str>` (which could insert a short-lived reference into a container expecting static ones).

---

## 4. Zero-Cost Abstractions — Monomorphization

### How Generics Compile

Generics in Rust are **monomorphized**: for each concrete type a generic is instantiated with, the compiler generates a specialized copy.

```rust
fn max<T: Ord>(a: T, b: T) -> T { if a > b { a } else { b } }

// Used as:
max(1i32, 2i32);        // Generates: fn max_i32(a: i32, b: i32) -> i32
max(1.0f64, 2.0f64);    // Generates: fn max_f64(a: f64, b: f64) -> f64
```

**Cost:** Zero runtime overhead (no vtable, no boxing). **Tradeoff:** Larger binary size (code duplication).

### Static vs Dynamic Dispatch

| Mechanism | Syntax | Dispatch | Overhead |
|:----------|:-------|:---------|:---------|
| Monomorphization | `fn f<T: Trait>(x: T)` | Static (compile-time) | None |
| Trait objects | `fn f(x: &dyn Trait)` | Dynamic (vtable) | 1 indirect call |

### Trait Object Layout (Fat Pointer)

A `&dyn Trait` is a **fat pointer** — two words:

```
┌──────────────┬──────────────┐
│  data_ptr    │  vtable_ptr  │
│  (8 bytes)   │  (8 bytes)   │
└──────────────┴──────────────┘
```

The vtable contains: `drop`, `size`, `align`, and all trait method function pointers.

---

## 5. Memory Layout and Representations

### Enum Layout — Discriminated Unions

```rust
enum Option<T> {
    None,
    Some(T),
}
```

$$\text{sizeof}(\text{Option<T>}) = \text{sizeof}(T) + \text{sizeof}(\text{discriminant}) + \text{padding}$$

### Niche Optimization

For types with invalid bit patterns (niches), the compiler stores the discriminant **inside** the payload:

```rust
// NonZeroU32 can never be 0, so:
sizeof(Option<NonZeroU32>) == sizeof(NonZeroU32) == 4

// &T can never be null, so:
sizeof(Option<&T>) == sizeof(&T) == 8  // None is null pointer
```

### Struct Layout

By default, Rust structs use `repr(Rust)` — the compiler is free to reorder fields for optimal packing. For C interop:

| Repr | Behavior |
|:-----|:---------|
| `repr(Rust)` | Compiler may reorder fields, optimize padding |
| `repr(C)` | C-compatible layout, no reordering |
| `repr(transparent)` | Same layout as single field |
| `repr(packed)` | No padding (may cause unaligned access) |

---

## 6. Trait System — Type-Level Logic

### Trait Bounds as Constraints

Trait bounds form a **constraint system** that the compiler solves:

```rust
fn process<T>(x: T)
where
    T: Clone + Send + 'static,       // T must satisfy all three
    T: Into<String>,                   // AND be convertible to String
{
    // ...
}
```

### Coherence and the Orphan Rule

**Coherence:** For any type and trait, there is at most one implementation. Enforced by the **orphan rule**: you can implement trait `T` for type `S` only if:
- `T` is defined in your crate, **or**
- `S` is defined in your crate

This prevents conflicting implementations across crates.

### Associated Types vs Type Parameters

```rust
// Type parameter — caller chooses:
trait Iterator<Item> { fn next(&mut self) -> Option<Item>; }
// Problem: Vec<i32> could implement Iterator<i32> AND Iterator<String>

// Associated type — impl chooses (at most one per type):
trait Iterator { type Item; fn next(&mut self) -> Option<Self::Item>; }
// Vec<i32> implements Iterator with Item = i32, period.
```

---

## 7. Unsafe and the Safety Boundary

### What `unsafe` Enables

Exactly five things:
1. Dereference raw pointers (`*const T`, `*mut T`)
2. Call `unsafe` functions
3. Access mutable statics
4. Implement `unsafe` traits
5. Access fields of `union`s

### The Safety Contract

`unsafe` does not disable the borrow checker. It **extends trust**: the programmer asserts invariants the compiler cannot verify. The `unsafe` block is a **proof obligation** — you must ensure:

- No data races
- No dangling pointers
- No invalid bit patterns
- No aliasing violations

---

## 8. Summary of Key Concepts

| Concept | Formal Basis | Enforcement |
|:--------|:-------------|:------------|
| Ownership | Affine type system | Compile-time move checker |
| Borrowing | Readers-writers lock invariant | Borrow checker |
| Lifetimes | Subtyping constraints | Type inference + annotations |
| Generics | Parametric polymorphism | Monomorphization |
| Trait dispatch | Ad-hoc polymorphism | Static (generics) or dynamic (vtable) |
| Niche optimization | Sum type encoding | Compiler layout optimization |
| Variance | Category theory (functors) | Type system rules |
| `unsafe` | Proof obligations | Programmer contract |

---

*Rust's guarantee is not "your program has no bugs." It's "your program has no undefined behavior." Everything else — logic errors, deadlocks, memory leaks — is still your problem. But the class of bugs that corrupt memory, enable RCE, and crash production? Those are gone at compile time.*
