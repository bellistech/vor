# Polyglot Landmines — Where Translation Burns You

> *Five languages, five worldviews. Naming a thing the same does not make it the same. This page catalogues the places a working idiom in language A produces a subtly broken program in language B — equality, nullness, truthiness, numbers, strings, errors, concurrency, and memory.*

---

## 1. The Five Models at a Glance

| Axis | Rust | Go | Python | TypeScript | Bash |
|:-----|:-----|:----|:-------|:-----------|:------|
| **Memory** | Ownership / borrow / lifetimes | GC + escape analysis | Reference-counted GC + cycle detector | V8 / engine GC | Process forks; subshell scope |
| **Types** | Static, nominal, sound | Static, structural-ish (interfaces) | Dynamic, gradually typed via hints | Static structural (erased at runtime) | Untyped strings; integers in `$(( ))` |
| **Errors** | `Result<T, E>` + `?` | `(T, error)` returns | Exceptions | Exceptions + Promise rejection | `$?` + `set -euo pipefail` + `trap` |
| **Concurrency** | Threads + `Send`/`Sync`; async via runtime | Goroutines + channels | GIL + threads / asyncio / multiprocessing | Single-threaded event loop + Workers | `&` / `wait` / `xargs -P` |
| **Null** | `Option<T>` (no nulls) | nil pointers + zero values | `None` | `null` and `undefined` (different) | unset vs empty string |
| **Package mgmt** | Cargo (`Cargo.toml`) | Modules (`go.mod`) | pip / poetry / uv | npm / pnpm / yarn | none (sourced files) |

---

## 2. Null / Empty / Zero — Five Concepts Wearing One Name

### Bash: unset ≠ empty ≠ "0" ≠ 0

```bash
unset x;          [[ -z "${x:-}" ]] && echo unset
y="";             [[ -z "$y" ]]    && echo empty
z="0";            [[ -n "$z" ]]    && echo "non-empty (even though zero)"
declare -p x 2>/dev/null || echo "x truly does not exist"
```

The `${var:-default}` form treats unset and empty as the same; `${var-default}` distinguishes them. Most production bash bugs in this area come from forgetting which form you used.

### Go: nil and zero values are different

```go
var s []int        // nil slice  — len=0, cap=0, == nil
s = []int{}        // empty slice — len=0, cap=0, != nil  ← gotcha for JSON marshalling
var m map[string]int  // nil map — reads OK, writes panic!
m["k"] = 1         // panic: assignment to entry in nil map
```

A nil slice marshals to `null`; an empty slice marshals to `[]`. APIs care.

### Python: None vs falsy collections

```python
x = None
if x:           # False
if x is None:   # idiomatic check (don't use ==)
y = []
if y:           # False — empty list is falsy
if y is None:   # False — y is an empty list, not None
```

Also: **mutable default arguments** are evaluated once at function definition. `def f(xs=[]): xs.append(1); return xs` returns `[1]`, then `[1, 1]`, then `[1, 1, 1]`. Use `xs=None` and `xs = xs if xs is not None else []` inside.

### TypeScript: null vs undefined

```ts
let a: number | null = null;
let b: number | undefined;          // undefined
console.log(a == b);    // true  — loose equality treats null == undefined
console.log(a === b);   // false — strict; they ARE different
console.log(a ?? 0);    // 0 — nullish coalescing covers BOTH null and undefined
console.log(a || 0);    // 0 — but also fires for 0, "", false (truthy soup)
```

`strictNullChecks` is non-negotiable in production code.

### Rust: Option<T> — null is a type, not a runtime trap

```rust
let x: Option<i32> = None;
match x {
    Some(v) => println!("{v}"),
    None    => println!("missing"),
}
let y = x.unwrap_or(0);            // safe default
let z = x.expect("must exist");    // panic with message — debug aid only
```

There is no null-pointer-deref class of bug in safe Rust because the type system makes you handle the absent case.

---

## 3. Equality — `==` Means Different Things in Each Language

| Language | Operator | What it does | Footgun |
|:---------|:---------|:-------------|:--------|
| Bash | `=` (or `==` in `[[ ]]`) | String equality | `[ $a = $b ]` word-splits both sides if unquoted |
| Bash | `-eq` | Integer equality | `"10" -eq "10.0"` → error (not int) |
| Go | `==` | Value equality; for interfaces both type AND value | Panics on uncomparable dynamic types (slice, map, func) |
| Python | `==` | Value equality (calls `__eq__`) | NaN: `float('nan') == float('nan')` is False |
| Python | `is` | Identity (same object) | `x is y` for small ints / interned strings is *coincidentally* true; do NOT rely on it |
| JS / TS | `==` | Loose — coerces types | `[] == ![]` → true. Do not use. |
| JS / TS | `===` | Strict equality | `NaN === NaN` is false. Use `Object.is` or `Number.isNaN`. |
| Rust | `==` | Calls `PartialEq::eq` | Floats are `PartialEq` but NOT `Eq` (NaN reflexivity violation) |

### Concrete trap: the Go interface comparison panic

```go
var a interface{} = []int{1}
var b interface{} = []int{1}
_ = a == b   // runtime panic: comparing uncomparable type []int
```

Always check: would I store a slice/map/func behind this `interface{}`? If yes, write a comparator.

---

## 4. Numbers — Same Operation, Five Different Outcomes

### Integer overflow

```rust
let x: i32 = i32::MAX;
let y = x + 1;   // debug build: panic. release build: wraps to i32::MIN.
// Use checked_add / wrapping_add / saturating_add explicitly when intent matters.
```

```go
var x int32 = math.MaxInt32
y := x + 1   // silently wraps to MinInt32. No warning.
```

```python
x = 2**63
print(x + 1)   # 9223372036854775809 — int auto-promotes. No overflow.
```

```ts
const x = Number.MAX_SAFE_INTEGER  // 2^53 - 1
console.log(x + 1)  // 9007199254740992
console.log(x + 2)  // 9007199254740992  ← LOST PRECISION (silent)
// Use bigint when you mean integer: 9007199254740993n + 1n
```

```bash
x=$(( 2**62 ))
echo $(( x * 8 ))   # silently wraps on most shells (64-bit signed)
```

### Float precision

All five languages use IEEE-754 double (Python `float`, TS `number`, Rust `f64`, Go `float64`, bash via `bc`/`awk`). The mantissa is **52 bits** + implicit 1, so:

$$\text{exact integers representable} = [-2^{53}, 2^{53}]$$

`0.1 + 0.2 != 0.3` in **all five**. Use a fixed-point or decimal type (Python `Decimal`, Rust `rust_decimal`, JS `bigint`/`Decimal.js`, Go `shopspring/decimal`) when handling money.

---

## 5. String Indexing — Bytes vs Codepoints vs UTF-16 Units

```rust
let s = "héllo";
// s[0..2] — compile error if it cuts a UTF-8 codepoint
let first_char = s.chars().next();
```

Rust string slices are **bytes**, but the slicing operator panics rather than corrupt UTF-8 — pick `.chars()` or `.char_indices()` for codepoint work.

```go
s := "héllo"
fmt.Println(len(s))               // 6 — bytes
fmt.Println(utf8.RuneCountInString(s))   // 5 — codepoints
for i, r := range s { ... }       // range over runes (codepoints)
```

```python
s = "héllo"
print(len(s))      # 5 — codepoints (Python 3)
print(s[1])        # 'é'
```

```ts
const s = "héllo";
console.log(s.length);    // 5 — but UTF-16 code units, not codepoints
const emoji = "👋";
console.log(emoji.length);   // 2 (surrogate pair) — this is the bite
console.log([...emoji].length); // 1 — spread iterates codepoints
```

```bash
s="héllo"
echo "${#s}"      # 5 in UTF-8 locale; 6 in C locale. Locale matters.
```

### Bash word splitting (the canonical trap)

```bash
files="a.txt b c.txt"
rm $files          # WRONG — word-splits + globs; if "b" matches a dir, kaboom
# correct:
files=(a.txt "b" c.txt)
rm "${files[@]}"
```

`set -f` disables globbing; `IFS=` controls splitting. Most scripts get neither.

---

## 6. Truthiness Soup

| Value | Bash `[[ -n ]]` | Python `bool(x)` | TS truthy | Go / Rust `if x` |
|:------|:---:|:---:|:---:|:---:|
| `""` (empty string) | false | false | false | compile error (must be `bool`) |
| `"0"` | **true** | true | true | n/a |
| `0` (number) | (n/a) | false | false | n/a |
| `[]` / empty slice | (n/a) | **false** | true | n/a |
| `{}` / empty object | (n/a) | **false** | true | n/a |
| `null` / `None` | (n/a) | false | false | n/a |
| `NaN` | (n/a) | true (!) | false | n/a |
| `"false"` | true | true | true | n/a |

The lesson: **never** rely on truthiness for non-boolean types when porting code. Always test the specific condition (`x is not None`, `x.length > 0`, `x !== undefined`).

---

## 7. Concurrency Models

### Go — goroutines + channels

```go
ch := make(chan int, 10)
for i := 0; i < 100; i++ {
    go func(i int) { ch <- work(i) }(i)
}
// Receive N results; loop body runs concurrently up to GOMAXPROCS.
```

Cheap (2 KB initial stack), preemptive (since 1.14), scheduled by the runtime onto OS threads.

### Rust — threads with `Send` + `Sync` proven by the compiler

```rust
use std::sync::{Arc, Mutex};
let counter = Arc::new(Mutex::new(0));
let mut handles = vec![];
for _ in 0..10 {
    let c = Arc::clone(&counter);
    handles.push(std::thread::spawn(move || { *c.lock().unwrap() += 1; }));
}
for h in handles { h.join().unwrap(); }
```

You cannot send a non-`Send` type across threads — the compiler refuses to build it. Data-race-free by construction.

For async, **pick a runtime**: Tokio, async-std, smol. The language doesn't ship one.

### Python — three competing stories

```python
# 1. threading + GIL: only one thread executes Python bytecode at once.
#    Good for I/O-bound. Useless for CPU-bound.
# 2. asyncio: cooperative single-thread, await yields.
# 3. multiprocessing: real parallelism, but processes don't share memory cheaply.
import asyncio
async def fetch(u): ...
results = asyncio.run(asyncio.gather(*[fetch(u) for u in urls]))
```

PEP 703 (Python 3.13+ `--disable-gil`) is changing the story but is not yet the default.

### TypeScript — single event loop + Workers

```ts
// Default: single thread. CPU-bound work blocks I/O.
const results = await Promise.all(urls.map(fetch));
// Real parallelism: worker_threads (Node) or Web Workers (browser).
```

`async`/`await` is sugar over `Promise`. Errors in unhandled promises terminate the process under default `--unhandled-rejections=throw`.

### Bash — processes + `wait` / `xargs -P`

```bash
for url in "${urls[@]}"; do
    fetch "$url" &
done
wait    # block until all backgrounded jobs finish

# Or, capped parallelism:
printf '%s\n' "${urls[@]}" | xargs -P 8 -I{} fetch {}
```

No shared memory. Communication via files, pipes, or a coprocess.

---

## 8. Error Propagation Patterns

### Rust — `Result` + `?` operator

```rust
fn read_config() -> Result<Config, ConfigError> {
    let raw = std::fs::read_to_string("config.toml")?;   // ? converts io::Error -> ConfigError via From
    let cfg: Config = toml::from_str(&raw)?;
    Ok(cfg)
}
```

### Go — explicit returns + `%w` wrapping

```go
func readConfig() (*Config, error) {
    raw, err := os.ReadFile("config.toml")
    if err != nil { return nil, fmt.Errorf("read config: %w", err) }
    var cfg Config
    if err := toml.Unmarshal(raw, &cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    return &cfg, nil
}
// Inspect chain: errors.Is(err, fs.ErrNotExist) / errors.As(err, &target)
```

### Python — exceptions + `from`

```python
try:
    raw = open("config.toml").read()
    cfg = tomllib.loads(raw)
except FileNotFoundError as e:
    raise ConfigError("missing config") from e
```

`raise X from Y` preserves the cause chain, visible via `__cause__`.

### TypeScript — try/catch + Promise rejection

```ts
async function readConfig() {
    try {
        const raw = await fs.readFile("config.toml", "utf8");
        return parseToml(raw);
    } catch (e) {
        throw new Error("read config", { cause: e });
    }
}
```

`error.cause` (ES2022) preserves the chain.

### Bash — `set -euo pipefail` + `trap`

```bash
set -euo pipefail        # -e: exit on error; -u: undefined vars; pipefail: pipeline rc = rightmost non-zero
trap 'rc=$?; echo "failed at line $LINENO with rc=$rc" >&2' ERR
config=$(cat config.toml)   # exits immediately if cat fails
```

Without `set -e`, every command's `$?` must be inspected manually. Without `pipefail`, `cat missing | grep x` returns 0.

---

## 9. Memory & Aliasing

### Rust — ownership prevents aliasing bugs

```rust
let mut v = vec![1, 2, 3];
let r = &v[0];
v.push(4);          // compile error: cannot borrow v as mutable while &v[0] is alive
```

Reallocation could move the vector — Rust refuses to compile the case. The borrow checker is doing exactly the work that produces the next bug in every other language.

### Go — slice aliasing trap

```go
s := []int{1, 2, 3}
t := s[:2]
t = append(t, 99)
fmt.Println(s)    // [1, 2, 99] — append wrote into s's backing array!
// Until the slice exceeds cap, append shares the underlying memory.
```

Use `slices.Clone` from `slices` package, or `append([]int(nil), s[:2]...)` to detach.

### Python — references everywhere; mutable default args

```python
def append_user(name, users=[]):    # ⚠ default list is shared across calls
    users.append(name)
    return users

append_user("a")  # ['a']
append_user("b")  # ['a', 'b']  — surprise
```

Idiom: `users=None` then `users = users or []` inside.

### TypeScript / JavaScript — shallow spreads

```ts
const a = { profile: { name: "x" } };
const b = { ...a };               // shallow
b.profile.name = "y";
console.log(a.profile.name);      // "y" — same nested object
// Use structuredClone(a) for deep copy.
```

### Bash — subshell scope

```bash
count=0
ls | while read f; do
    (( count++ ))         # subshell — increments local copy
done
echo "$count"             # still 0
# Fix: use process substitution to keep loop in the parent shell:
while read f; do (( count++ )); done < <(ls)
```

---

## 10. The "Same Concept" Vocabulary

| Concept | Rust | Go | Python | TS | Bash |
|:--------|:-----|:----|:-------|:----|:------|
| Module / namespace | `mod`, crate | package | module | namespace / module | sourced file |
| Sum type / variant | `enum` | iota or interface union | `Union` / `Literal` | union types | n/a |
| Trait / interface | `trait` | `interface` | abstract base / Protocol | `interface` | n/a |
| Generic | `<T>` | `[T any]` (1.18+) | `TypeVar` | `<T>` | n/a |
| Channel | `mpsc::channel` | `chan T` | `queue.Queue` / `asyncio.Queue` | not built-in | named pipe / fifo |
| Defer / cleanup | `Drop` impl / scope guard | `defer` | `with` / `try-finally` | `using` (5.2+) / `try-finally` | `trap EXIT` |
| Pattern match | `match` | type switch | `match` (3.10+) | `switch` / discriminated union | `case ... esac` |
| Iterator | `Iterator` trait | range, custom func | iterator protocol (`__iter__`) | `Iterable<T>` | `for ... in ...` |

---

## Prerequisites

- Working familiarity with at least one of the five languages
- General programming concepts: types, functions, closures, exceptions, threads
- For the concurrency section: thread vs process vs coroutine, shared memory vs message passing
- For the memory section: stack vs heap, reference vs value semantics, garbage collection vs manual management

## Complexity

This is a comparative reference, not a tutorial. Read it after you can write a correct program in *one* of these languages and want to understand why your idiom from there breaks here.

The **most-common production failures** when translating between these five:
1. **Bash word-splitting on unquoted variables** (~40% of bash bugs in shipped scripts)
2. **JS `==` coercion + falsy `0` / `""`** (countless `if (x)` bugs masking real values)
3. **Python mutable default arguments**
4. **Go nil-vs-zero-value, especially in JSON marshalling**
5. **Mismatched concurrency models** (assuming GIL semantics in TS, or async/await in Go)

The detail page above is organized to surface exactly these.
