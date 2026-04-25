# Polyglot (Rust, Go, Python, Bash, TypeScript)

Side-by-side idiom map across five working languages — find the equivalent fast.

## Variables, Constants & Scope

### Declaration

```bash
# Rust:    let x: i32 = 42;          let mut y = 0;          // immutable by default
# Go:      var x int = 42            x := 42                  // short-decl in func only
# Python:  x: int = 42               x = 42                   # type hints optional
# TS:      const x: number = 42      let y = 0                // const = no rebind
# Bash:    x=42                      readonly PI=3.14         # NO spaces around =
```

### Mutability default

```bash
# Rust:    immutable; opt in with `mut`. Compile error if you forget.
# Go:      mutable; const for compile-time literals only.
# Python:  mutable; convention SCREAMING_CASE for constants (no enforcement).
# TS:      `const` rebinding-immutable but mutates inner objects; `as const` deep-freezes literals.
# Bash:    mutable; `readonly` makes the binding immutable for the rest of the shell.
```

### Constants

```bash
# Rust:    const MAX_USERS: u32 = 1000;                       // compile-time, inlined
# Go:      const MaxUsers = 1000                              // typed or untyped
# Python:  MAX_USERS: Final[int] = 1000   # PEP 591 — checker-enforced only
# TS:      const MAX_USERS = 1000 as const                    // narrow literal type
# Bash:    readonly MAX_USERS=1000                            # set -o nounset to catch typos
```

### Shadowing

```bash
# Rust:    let x = 5;        let x = "hi";  // legal — new binding, type can change
# Go:      x := 5;           x = "hi"        // ILLEGAL — type cannot change
# Python:  x = 5;            x = "hi"        # legal — names hold whatever
# TS:      let x = 5;        x = "hi"        // illegal under strict typing
# Bash:    x=5;              x="hi"          # all values are strings anyway
```

### Scope rules

```bash
# Rust:    block-scoped {}; lexical; let drops at `}`.
# Go:      block-scoped {}; package-level for top-decls; capitalized = exported.
# Python:  function-scoped (LEGB: Local, Enclosing, Global, Built-in).
# TS:      block-scoped (let/const) or function-scoped (var, avoid).
# Bash:    global by default; `local x` inside func; subshells get a copy.
```

## Primitive Types

### Integer types

```bash
# Rust:    i8 i16 i32 i64 i128 isize / u8 u16 u32 u64 u128 usize        // explicit width
# Go:      int int8 int16 int32 int64 / uint*; int is 64-bit on 64-bit OS
# Python:  int — arbitrary precision, no overflow                         # 2**1000 is fine
# TS:      number is float64; bigint for arbitrary-precision int          // 9007199254740993n
# Bash:    untyped strings; arithmetic in $(( )) is 64-bit signed         # x=$(( 2**62 ))
```

### Float types

```bash
# Rust:    f32, f64                          // f64 is the default literal type
# Go:      float32, float64                  // float64 is the default
# Python:  float == 64-bit IEEE-754
# TS:      number == 64-bit IEEE-754 (only)
# Bash:    no native floats; use `bc -l` or `awk` or python3 -c
```

### Boolean

```bash
# Rust:    bool — true | false               // no truthy coercion
# Go:      bool — true | false               // no truthy coercion
# Python:  bool — True | False               # subclass of int; True == 1
# TS:      boolean — true | false            // beware == coercion; use ===
# Bash:    no native bool; use `true` / `false` commands and exit codes; `[[ -n "$x" ]]`
```

### Character

```bash
# Rust:    char — 4-byte Unicode scalar value           // 'A', '\u{1F44B}'
# Go:      rune — alias for int32, a Unicode codepoint  // 'A'
#          byte — alias for uint8                       // 'A' if you mean ASCII
# Python:  no separate char type — single-char str
# TS:      no separate char type — single-char string
# Bash:    no chars; substring with ${var:0:1}
```

### Type conversion

```bash
# Rust:    let n: i32 = 42;   let f = n as f64;   let s: String = n.to_string();
# Go:      n := 42;            f := float64(n);    s := strconv.Itoa(n)
# Python:  n = 42;             f = float(n);       s = str(n)
# TS:      const n = 42;       const f = n;        const s = String(n)   // Number(s) reverse
# Bash:    n=42;               # arithmetic context auto-converts; printf for format
```

### Type checking at runtime

```bash
# Rust:    no runtime type info by default; use `std::any::Any` or enums for sum types
# Go:      v, ok := x.(MyType)            // type assertion; `reflect.TypeOf(x)` for full info
# Python:  isinstance(x, int);            type(x) is int                # type() vs isinstance
# TS:      typeof x === "number";         x instanceof MyClass          // erased generics
# Bash:    [[ "$x" =~ ^[0-9]+$ ]]         # only string pattern checks
```

## Strings & Formatting

### Literal forms

```bash
# Rust:    "hello" (&str — borrowed)       String::from("hi")        r"raw \n"        b"bytes"
# Go:      "hello"                          `raw multi-line`           // no built-in raw escapes
# Python:  "hello"   'hi'   r"raw \n"      f"interp"   b"bytes"      """triple"""
# TS:      "hello"   'hi'   `template`     // no raw form — use String.raw\`...\`
# Bash:    "double quoted $var"   'single — no expansion'   $'C-escapes \n'   <<EOF heredoc EOF
```

### Interpolation / formatting

```bash
# Rust:    format!("{name} is {age}", name=n, age=a)        // capture from scope (1.58+)
# Go:      fmt.Sprintf("%s is %d", name, age)               // %v %s %d %f %q %+v %T
# Python:  f"{name} is {age:>5}"                             # also "{}".format(...)  and  % style
# TS:      `${name} is ${age}`                               // template literal
# Bash:    printf "%s is %d\n" "$name" "$age"                # printf is portable; echo is not
```

### Multi-line strings

```bash
# Rust:    let s = "line1\n\
#                  line2";   //    or    let s = r#"line1\nline2"#;
# Go:      s := `line1
#                line2`                       // raw — no \n interpretation
# Python:  s = """line1
#                 line2"""                    # triple-quote
# TS:      const s = `line1
#                     line2`;                 // template literal
# Bash:    s=$(cat <<'EOF'                    # 'EOF' (quoted) disables variable expansion
# line1
# line2
# EOF
# )
```

### Length & substring

```bash
# Rust:    s.len();   &s[0..5];                 // bytes; panic if cuts a codepoint
# Go:      len(s);    s[0:5]                     // bytes; utf8.RuneCountInString for chars
# Python:  len(s);    s[0:5]                     # codepoints
# TS:      s.length;  s.slice(0, 5)              // UTF-16 code units (emoji bite)
# Bash:    ${#s};     ${s:0:5}                    # locale-dependent
```

### Search / replace / split / join

```bash
# Rust:    s.contains("x"); s.replace("a","b"); s.split(",").collect::<Vec<_>>(); v.join(",")
# Go:      strings.Contains(s,"x"); strings.Replace(s,"a","b",-1); strings.Split(s,","); strings.Join(v,",")
# Python:  "x" in s;    s.replace("a","b");    s.split(",");        ",".join(v)
# TS:      s.includes("x");  s.replaceAll("a","b");  s.split(",");  v.join(",")
# Bash:    [[ "$s" == *x* ]]; s="${s//a/b}";   IFS=, read -ra arr <<<"$s";  s="$(IFS=,; echo "${arr[*]}")"
```

### Case conversion

```bash
# Rust:    s.to_uppercase();  s.to_lowercase()
# Go:      strings.ToUpper(s); strings.ToLower(s)
# Python:  s.upper();          s.lower();         s.title();   s.casefold()
# TS:      s.toUpperCase();    s.toLowerCase()
# Bash:    "${s^^}";           "${s,,}"            # bash 4+
```

### Trim & pad

```bash
# Rust:    s.trim();    s.trim_start();   format!("{:>10}", s);   format!("{:0>5}", n)
# Go:      strings.TrimSpace(s); strings.TrimLeft(s," "); fmt.Sprintf("%10s", s); fmt.Sprintf("%05d", n)
# Python:  s.strip();   s.lstrip();        f"{s:>10}";              f"{n:05d}"
# TS:      s.trim();    s.trimStart();     s.padStart(10);          n.toString().padStart(5,"0")
# Bash:    s="${s##*( )}";   printf "%10s" "$s";   printf "%05d" "$n"
```

### String builder / efficient concat

```bash
# Rust:    let mut s = String::with_capacity(64);  s.push_str("a");   s.push('b');
# Go:      var b strings.Builder; b.Grow(64); b.WriteString("a"); b.WriteByte('b'); s := b.String()
# Python:  parts: list[str] = []; parts.append("a"); s = "".join(parts)   # join is the idiom
# TS:      const parts: string[] = []; parts.push("a"); const s = parts.join("");
# Bash:    s=""; for x in "${arr[@]}"; do s+="$x"; done                    # quadratic; usually fine
```

## Collections — List / Array

### Construction

```bash
# Rust:    let v: Vec<i32> = vec![1, 2, 3];   let v: [i32; 3] = [1, 2, 3];   // fixed array
# Go:      v := []int{1, 2, 3}                v := [3]int{1, 2, 3}             // fixed array
# Python:  v = [1, 2, 3]                       v = list(range(3))
# TS:      const v: number[] = [1, 2, 3]       const v = Array.of(1, 2, 3)
# Bash:    v=(1 2 3)                           v=({1..3})                       # brace expansion
```

### Append / insert / remove

```bash
# Rust:    v.push(4);     v.insert(0, 0);     v.remove(1);     v.pop();   // pop returns Option
# Go:      v = append(v, 4); v = append([]int{0}, v...); v = append(v[:1], v[2:]...); v = v[:len(v)-1]
# Python:  v.append(4);   v.insert(0, 0);      del v[1] / v.pop(1);  v.pop()
# TS:      v.push(4);     v.unshift(0);        v.splice(1, 1);       v.pop()
# Bash:    v+=(4);        v=(0 "${v[@]}");     unset 'v[1]'; v=("${v[@]}");   unset 'v[-1]'
```

### Index & slicing

```bash
# Rust:    v[0];   v.get(0)  // returns Option;   &v[1..4];   &v[..3];   &v[2..]
# Go:      v[0]    // panic on OOB;               v[1:4];     v[:3];      v[2:]
# Python:  v[0];   v[-1]  // negative indexes;    v[1:4];     v[:3];      v[::2]   # step
# TS:      v[0];   v.at(-1)  // ES2022 negative;  v.slice(1, 4)
# Bash:    "${v[0]}"; "${v[-1]}"  # bash 4.3+;   "${v[@]:1:3}"
```

### Length & iterate

```bash
# Rust:    v.len();  for x in &v { ... }           // for x in v consumes
# Go:      len(v);   for i, x := range v { ... }
# Python:  len(v);   for x in v: ...               for i, x in enumerate(v): ...
# TS:      v.length; for (const x of v) { ... }    v.forEach((x, i) => ...)
# Bash:    "${#v[@]}"; for x in "${v[@]}"; do ...; done
```

### Map / filter / reduce

```bash
# Rust:    v.iter().map(|x| x*2).filter(|x| *x > 0).sum::<i32>()
# Go:      // no stdlib — write a loop, or use slices.Map (golang.org/x/exp/slices) in 1.21+
# Python:  [x*2 for x in v if x > 0]              # comprehension preferred over map/filter
#          sum(x*2 for x in v if x > 0)            # generator — lazy
# TS:      v.map(x => x*2).filter(x => x > 0).reduce((a, b) => a + b, 0)
# Bash:    s=0; for x in "${v[@]}"; do (( x>0 )) && s=$(( s + x*2 )); done; echo "$s"
```

### Sort / reverse

```bash
# Rust:    v.sort();        v.sort_by(|a, b| b.cmp(a));     v.reverse()
# Go:      sort.Ints(v);    sort.Slice(v, func(i,j int) bool { return v[i] > v[j] });   slices.Reverse(v)
# Python:  v.sort();        v.sort(key=lambda x: -x);        v.reverse()
# TS:      v.sort((a,b) => a-b);   v.sort((a,b) => b-a);     v.reverse()   // mutates!
# Bash:    sorted=( $(printf '%s\n' "${v[@]}" | sort -n) ) // word-split safe for numbers
```

### Search / contains

```bash
# Rust:    v.contains(&3);  v.iter().position(|x| *x == 3);  v.binary_search(&3)
# Go:      slices.Contains(v, 3);   slices.Index(v, 3);   sort.SearchInts(v, 3)   // requires sorted
# Python:  3 in v;          v.index(3)  // ValueError if missing;      bisect.bisect_left(v, 3)
# TS:      v.includes(3);   v.indexOf(3);  v.findIndex(x => x === 3)
# Bash:    [[ " ${v[*]} " == *" 3 "* ]]                                # crude — beware of substrings
```

### Concatenate

```bash
# Rust:    let c = [a, b].concat();   let c = a.iter().chain(b.iter()).copied().collect::<Vec<_>>();
# Go:      c := append(a, b...)        // mutates a's backing if cap allows!
# Python:  c = a + b                    # creates new list
#          a.extend(b)                  # mutates
# TS:      const c = [...a, ...b];     const c = a.concat(b)
# Bash:    c=("${a[@]}" "${b[@]}")
```

### Transpose / zip

```bash
# Rust:    a.iter().zip(b.iter())         // Iterator of (&A, &B)
# Go:      // no zip — write the loop with shared index
# Python:  zip(a, b);   zip(*matrix)     # transpose
# TS:      a.map((x, i) => [x, b[i]] as const)    // no built-in zip
# Bash:    # no zip — paste(1) is the closest:  paste -d, <(printf '%s\n' "${a[@]}") <(printf '%s\n' "${b[@]}")
```

## Collections — Map / Dict / HashMap

### Construction

```bash
# Rust:    let m: HashMap<&str, i32> = HashMap::from([("k1", 1), ("k2", 2)]);
# Go:      m := map[string]int{"k1": 1, "k2": 2}
# Python:  m = {"k1": 1, "k2": 2}                                  # also dict(k1=1, k2=2)
# TS:      const m: Record<string, number> = {k1: 1, k2: 2}        // or new Map([["k1", 1]])
# Bash:    declare -A m=([k1]=1 [k2]=2)                            # bash 4+
```

### Get / set / delete

```bash
# Rust:    m.insert("k", 3);   m.get("k");  // Option<&V>     m.remove("k");   // Option<V>
# Go:      m["k"] = 3;          v, ok := m["k"];               delete(m, "k")
# Python:  m["k"] = 3;          m.get("k", default);            del m["k"]   # KeyError if missing
# TS:      m.k = 3;             m.k ?? default;                 delete m.k
# Bash:    m[k]=3;              "${m[k]:-default}";              unset 'm[k]'
```

### Iterate

```bash
# Rust:    for (k, v) in &m { ... }                     // unsorted
# Go:      for k, v := range m { ... }                  // RANDOMIZED iteration order — deliberate!
# Python:  for k, v in m.items(): ...                   # insertion order since 3.7
# TS:      for (const [k, v] of Object.entries(m)) ...  // string keys only
#          for (const [k, v] of mapObj) ...             // for Map<K, V>
# Bash:    for k in "${!m[@]}"; do v="${m[$k]}"; done
```

### Default values & contains-key

```bash
# Rust:    *m.entry("k").or_insert(0) += 1;             // upsert pattern
#          m.contains_key("k")
# Go:      v, ok := m["k"];   if !ok { m["k"] = 0 }
# Python:  m.setdefault("k", 0);   m.get("k", 0);   "k" in m
#          from collections import defaultdict;   d = defaultdict(int);   d["k"] += 1
# TS:      m.k ??= 0;   "k" in m;   Object.hasOwn(m, "k")
# Bash:    [[ -v 'm[k]' ]] && echo present;   m[k]="${m[k]:-0}"
```

### Order

```bash
# Rust:    HashMap is unordered. Use BTreeMap for sorted-by-key, IndexMap (crate) for insertion order.
# Go:      maps are unordered AND randomized per iteration. Sort keys explicitly when emitting.
# Python:  dict preserves insertion order since 3.7. OrderedDict for explicit semantics.
# TS:      Object key order: integer-like ascending, then strings in insertion order, then symbols.
#          Map preserves insertion order (use Map for ordered keys).
# Bash:    associative array order is unspecified.
```

## Collections — Set

### Construction

```bash
# Rust:    let s: HashSet<&str> = HashSet::from(["a", "b", "c"]);
# Go:      s := map[string]struct{}{"a": {}, "b": {}, "c": {}}        // idiom: map to empty struct
# Python:  s = {"a", "b", "c"};   s = set(["a", "b"])                  # frozenset for immutable
# TS:      const s = new Set<string>(["a", "b", "c"])
# Bash:    declare -A s=([a]=1 [b]=1 [c]=1)                            # use assoc array as set
```

### Add / contains / remove

```bash
# Rust:    s.insert("d");        s.contains("a");        s.remove("a")
# Go:      s["d"] = struct{}{};  _, ok := s["a"];        delete(s, "a")
# Python:  s.add("d");           "a" in s;                s.discard("a")  # no error if missing
# TS:      s.add("d");           s.has("a");              s.delete("a")
# Bash:    s[d]=1;               [[ -v 's[a]' ]];          unset 's[a]'
```

### Set algebra

```bash
# Rust:    a.union(&b);  a.intersection(&b);  a.difference(&b);  a.symmetric_difference(&b)
# Go:      // write loops over map keys
# Python:  a | b;        a & b;                  a - b;            a ^ b
#          a.union(b);   a.intersection(b);      a.difference(b);  a.symmetric_difference(b)
# TS:      new Set([...a, ...b]);  new Set([...a].filter(x => b.has(x)));  // build manually pre-2024
#          a.union(b);   a.intersection(b);  a.difference(b);   a.symmetricDifference(b)   // ES2025
# Bash:    # comm(1) on sorted streams:    comm -12 <(sort a) <(sort b)
```

## Tuples / Records / Structs

### Tuple

```bash
# Rust:    let t: (i32, &str) = (1, "a");        let (n, s) = t;   t.0
# Go:      // no tuples; multiple returns: func() (int, string) { return 1, "a" }
# Python:  t = (1, "a");   n, s = t;             t[0]              # also NamedTuple
# TS:      const t: [number, string] = [1, "a"]; const [n, s] = t;  t[0]
# Bash:    # no tuples — return space-separated strings or use globals
```

### Struct / record

```bash
# Rust:    struct User { name: String, age: u32 }    let u = User { name: "Ada".into(), age: 30 };
# Go:      type User struct { Name string; Age int } u := User{Name: "Ada", Age: 30}
# Python:  @dataclass class User: name: str; age: int;   u = User("Ada", 30)
# TS:      type User = { name: string; age: number };    const u: User = { name: "Ada", age: 30 }
# Bash:    # no structs — assoc array or naming convention:    user_name=Ada; user_age=30
```

### Pattern destructuring

```bash
# Rust:    let User { name, age } = u;       let (a, b) = pair;   let [first, .., last] = arr;
# Go:      // limited — multi-assign: a, b := pair()
# Python:  u_name, u_age = u.name, u.age;    a, b, *rest = lst;    match u: case User(name=n): ...
# TS:      const { name, age } = u;          const [a, ...rest] = lst;   const [, b] = pair
# Bash:    read -r a b <<<"$pair";           IFS=, read -ra arr <<<"$csv"
```

## Optional / Maybe / Nullable

### Express absence

```bash
# Rust:    let x: Option<i32> = Some(5);     let n: Option<i32> = None;
# Go:      var p *int = nil;                  p := &v;                   // pointers; or (T, ok) pattern
# Python:  x: Optional[int] = None             # == int | None
# TS:      let x: number | undefined;         let y: number | null;      // distinct concepts
# Bash:    unset x   # vs    x=""             # ${x:-} treats both as same
```

### Default value when absent

```bash
# Rust:    x.unwrap_or(0);     x.unwrap_or_else(|| compute_default())
# Go:      if p != nil { v = *p } else { v = 0 }
# Python:  v = x if x is not None else 0
# TS:      const v = x ?? 0;                  // null AND undefined → use default; 0 / "" do NOT
# Bash:    v="${x:-0}"                        # unset OR empty
#          v="${x-0}"                         # unset only (empty string passes through)
```

### Map / and_then chain

```bash
# Rust:    user.and_then(|u| u.email).map(|e| e.to_lowercase()).unwrap_or_default()
# Go:      // verbose — guard at every step
# Python:  email = user.email.lower() if user and user.email else ""
# TS:      const email = user?.email?.toLowerCase() ?? "";              // optional chaining
# Bash:    email="${user_email:-}";  email="${email,,}"
```

### Convert to error

```bash
# Rust:    let v = x.ok_or("missing")?;       // Option -> Result, propagate with ?
# Go:      if p == nil { return fmt.Errorf("missing") }
# Python:  if x is None: raise ValueError("missing")
# TS:      if (x == null) throw new Error("missing");   // == covers null + undefined
# Bash:    [[ -n "${x:-}" ]] || { echo "missing" >&2; exit 1; }
```

## Control Flow

### if / else / ternary

```bash
# Rust:    let v = if cond { a } else { b };   // if is an expression
# Go:      var v T; if cond { v = a } else { v = b }   // statement only — no ternary
# Python:  v = a if cond else b
# TS:      const v = cond ? a : b
# Bash:    [[ cond ]] && v=a || v=b   # subtle bug if a is falsy; safer:  if [[ cond ]]; then v=a; else v=b; fi
```

### Pattern match / switch

```bash
# Rust:    match x { 0 => "zero", 1..=10 => "small", n if n > 100 => "big", _ => "other" }
# Go:      switch x { case 0: "zero"; case 1, 2: "few"; default: "other" }   // no fallthrough by default
# Python:  match x:                                                              # 3.10+
#              case 0: ...
#              case [_, *_] as lst: ...
#              case _: ...
# TS:      switch (x) { case 0: r = "z"; break; default: r = "o" }              // FALLS THROUGH without break!
# Bash:    case "$x" in 0) ...;; [12]) ...;; *) ...;; esac
```

### Loops — for / while / loop

```bash
# Rust:    for i in 0..10 {}      while cond {}       loop { if done { break; } }   // loop returns value via break
# Go:      for i := 0; i < 10; i++ {}    for cond {}    for {}                       // single keyword for all
# Python:  for i in range(10): ...         while cond: ...                            # else clause runs if not broken
# TS:      for (let i=0; i<10; i++) {}     while (cond) {}      do { } while (cond)
# Bash:    for i in {0..9}; do ...; done   while cond; do ...; done                   # also: until / select
```

### Break / continue / labels

```bash
# Rust:    'outer: for ... { for ... { break 'outer; } }
# Go:      OUTER: for ... { for ... { break OUTER }   continue OUTER }
# Python:  no labels — restructure with functions or a flag
# TS:      outer: for ... { for ... { break outer } }
# Bash:    break 2;   continue 2                                              # numeric — break N levels
```

### Early return / guard

```bash
# Rust:    if x.is_none() { return Err("missing".into()); }
# Go:      if x == nil { return ErrMissing }
# Python:  if x is None: return None                                            # or raise
# TS:      if (x == null) return undefined;
# Bash:    [[ -z "${x:-}" ]] && return 1
```

## Functions & Closures

### Definition

```bash
# Rust:    fn add(a: i32, b: i32) -> i32 { a + b }                       // last expr returned
# Go:      func add(a, b int) int { return a + b }
# Python:  def add(a: int, b: int) -> int: return a + b
# TS:      function add(a: number, b: number): number { return a + b; }
# Bash:    add() { echo $(( $1 + $2 )); }                                # use stdout for return values
```

### Default & keyword args

```bash
# Rust:    // none built-in; use Option<T> or builder pattern
# Go:      // none built-in; use functional options or a config struct
# Python:  def f(x: int = 10, *, name: str = "default"): ...               # * forces keyword-only
# TS:      function f(x: number = 10, opts: { name?: string } = {}) { ... }
# Bash:    f() { local x="${1:-10}";  local name="${2:-default}"; }
```

### Variadic args

```bash
# Rust:    fn f(args: &[i32]) {}                                          // pass slice
# Go:      func f(xs ...int) {}                                            // f(1, 2, 3) or f(slice...)
# Python:  def f(*args, **kwargs): ...                                     # args=tuple, kwargs=dict
# TS:      function f(...args: number[]): void {}                          // also: f(a, ...rest)
# Bash:    f() { for a in "$@"; do echo "$a"; done; }                      # $@ is variadic by nature
```

### Closures & capture

```bash
# Rust:    let f = |x| x * 2;             let f = move |x| x + captured;    // move takes ownership
# Go:      f := func(x int) int { return x * 2 }                            // closes over enclosing scope
# Python:  f = lambda x: x * 2                                              # closes by reference (late-binding!)
# TS:      const f = (x: number) => x * 2;                                  // arrow inherits this/lexical
# Bash:    # no closures — use functions + globals or nameref
```

### Higher-order functions

```bash
# Rust:    fn apply<F: Fn(i32) -> i32>(f: F, x: i32) -> i32 { f(x) }
# Go:      func apply(f func(int) int, x int) int { return f(x) }
# Python:  def apply(f, x): return f(x)
# TS:      const apply = <T, R>(f: (x: T) => R, x: T): R => f(x);
# Bash:    apply() { local f="$1"; shift; "$f" "$@"; }                       # call by name
```

### Method receivers / self

```bash
# Rust:    impl Foo { fn bar(&self) {} fn baz(&mut self) {} fn quux(self) {} }   // borrow / mut / consume
# Go:      func (f *Foo) Bar() {}     // pointer receiver mutates
#          func (f  Foo) Bar() {}     // value receiver — copy
# Python:  class Foo:  def bar(self): ...    @classmethod def baz(cls): ...    @staticmethod def qux(): ...
# TS:      class Foo {  bar() { return this.x; }  static baz() {} }
# Bash:    # no methods — use functions with $1=instance prefix:  user_get_name() { echo "${1}_name"; }
```

### Generics

```bash
# Rust:    fn max<T: Ord>(a: T, b: T) -> T { if a > b { a } else { b } }
# Go:      func max[T cmp.Ordered](a, b T) T { if a > b { return a }; return b }   // 1.18+
# Python:  def first[T](xs: list[T]) -> T: return xs[0]                              # 3.12+ syntax
# TS:      function max<T>(a: T, b: T, cmp: (a: T, b: T) => number): T { ... }
# Bash:    # no generics — bash is untyped; functions accept any string
```

## Error Handling

### Throw / return / propagate

```bash
# Rust:    fn read() -> Result<String, io::Error> { let s = fs::read_to_string("p")?; Ok(s) }
# Go:      data, err := os.ReadFile("p"); if err != nil { return nil, fmt.Errorf("read: %w", err) }
# Python:  data = open("p").read()        # raises OSError on failure
# TS:      const data = await fs.readFile("p", "utf8")    // throws on failure
# Bash:    set -euo pipefail; data=$(< p) || { echo "read failed: $?" >&2; exit 1; }
```

### Catch / handle

```bash
# Rust:    match read() { Ok(s) => use(s), Err(e) => log(e) }
# Go:      if err != nil { log.Println(err); return }
# Python:  try: ...
#          except FileNotFoundError as e: log(e)
#          except Exception as e: log(e); raise
# TS:      try { ... } catch (e) { if (e instanceof Error) log(e.message); }
# Bash:    if ! cmd; then echo "cmd failed: $?" >&2; fi
```

### Custom error types

```bash
# Rust:    #[derive(thiserror::Error, Debug)] enum MyErr { #[error("bad: {0}")] Bad(String) }
# Go:      type MyErr struct{ Msg string }
#          func (e *MyErr) Error() string { return e.Msg }
# Python:  class MyErr(Exception): pass
# TS:      class MyErr extends Error { constructor(public code: string) { super(code); } }
# Bash:    # no types — encode in exit codes (1-125 user-defined) and stderr messages
```

### Wrapping & cause chain

```bash
# Rust:    Err(MyErr::ParseFailed)?     // From<io::Error> for MyErr handles chain
# Go:      return fmt.Errorf("parse %s: %w", path, err)    // %w for chain;  errors.Is / errors.As
# Python:  raise MyErr("parse failed") from e              # __cause__ chain
# TS:      throw new Error("parse failed", { cause: e })   // ES2022
# Bash:    echo "parse failed at $path: $err" >&2;  exit 2
```

### Panic / abort / fatal

```bash
# Rust:    panic!("unrecoverable");      // unwinds (default) or aborts
# Go:      log.Fatal("unrecoverable")    // os.Exit(1) — does NOT run defers
#          panic("unrecoverable")        // runs defers, can be recover()'d
# Python:  raise SystemExit(1)           # also: sys.exit(1) or os._exit(1)
# TS:      throw new Error("...")        // unhandled = process exits
#          process.exit(1)               // immediate, skips finally/cleanup in async
# Bash:    exit 1;       die() { echo "$@" >&2; exit 1; }
```

### Cleanup / finally / defer

```bash
# Rust:    // RAII — Drop trait runs on scope exit
# Go:      defer file.Close()           // LIFO order; runs even on panic (unless os.Exit)
# Python:  with open("p") as f: ...     # __enter__/__exit__ context manager
#          try: ... finally: ...
# TS:      try { ... } finally { ... }
#          using f = openFile()         // ES2024 explicit resource management
# Bash:    trap 'rm -f "$tmp"' EXIT     # signal/exit hook
```

## Iteration & Iterators

### For-each

```bash
# Rust:    for x in &v { ... }      for (i, x) in v.iter().enumerate() { ... }
# Go:      for _, x := range v {}    for i, x := range v {}
# Python:  for x in v: ...           for i, x in enumerate(v): ...
# TS:      for (const x of v) {}     v.forEach((x, i) => {})
# Bash:    for x in "${v[@]}"; do echo "$x"; done
```

### Take / skip / chunk

```bash
# Rust:    v.iter().take(5);   v.iter().skip(5);     v.chunks(3)
# Go:      // no built-in; slice manually:  v[:5], v[5:], or write helper
# Python:  itertools.islice(v, 5);  itertools.islice(v, 5, None);  itertools.batched(v, 3)  # 3.12+
# TS:      v.slice(0, 5);     v.slice(5);            // no built-in chunk — write a loop
# Bash:    "${v[@]:0:5}";     "${v[@]:5}";            # no built-in chunk
```

### Zip / enumerate

```bash
# Rust:    a.iter().zip(b.iter());    v.iter().enumerate()
# Go:      // no zip — manual loop with shared index
# Python:  zip(a, b);                 enumerate(v)
# TS:      a.map((x, i) => [x, b[i]] as const);    v.entries()
# Bash:    # no zip — paste(1) on streams; for (( i=0; i<${#a[@]}; i++ )); do ...; done
```

### Generators / lazy sequences

```bash
# Rust:    iter::successors(Some(0u64), |&n| n.checked_add(1))   // infinite Iterator
# Go:      // no generators pre-1.23; range-over-func iterators in 1.23+
# Python:  def fib():
#              a, b = 0, 1
#              while True: yield a; a, b = b, a + b
# TS:      function* fib() { let a = 0, b = 1; while (true) { yield a; [a, b] = [b, a + b]; } }
# Bash:    # no generators — use coproc or named pipes for streaming
```

### Infinite sequences

```bash
# Rust:    iter::repeat(0).take(10);   (0..).filter(|n| n % 7 == 0)
# Go:      // 1.23+:  range-over-func generators
# Python:  itertools.count(1, 2);      itertools.cycle([1, 2, 3])
# TS:      function* nat() { let i = 0; while (true) yield i++; }
# Bash:    yes "$x" | head -10                                                # closest equivalent
```

## Concurrency

### Spawn (thread / goroutine / task)

```bash
# Rust:    let h = std::thread::spawn(|| work());                  // OS thread, ~2 MB stack
#          let t = tokio::spawn(async { fetch().await });           // async task, ~kb
# Go:      go work()                                                // goroutine, 2 KB initial stack
# Python:  threading.Thread(target=work).start()                    # GIL-bound for CPU
#          asyncio.create_task(fetch())
#          multiprocessing.Process(target=work).start()             # real parallelism
# TS:      // event loop only — for parallelism, use Workers:
#          new Worker(new URL("./w.ts", import.meta.url))
# Bash:    work &  pid=$!                                           # background process
```

### Wait / join

```bash
# Rust:    h.join().unwrap();              tokio::join!(t1, t2)
# Go:      var wg sync.WaitGroup; wg.Add(1); go func(){ defer wg.Done(); ... }(); wg.Wait()
# Python:  t.join();        await asyncio.gather(t1, t2)
# TS:      await Promise.all([f1(), f2()]);                          // for Promises
#          worker.terminate();   // for Workers — no implicit join
# Bash:    wait "$pid";       wait                                   # waits for ALL bg jobs
```

### Channels / message passing

```bash
# Rust:    let (tx, rx) = std::sync::mpsc::channel();   tx.send(1).unwrap();   rx.recv()
# Go:      ch := make(chan int, 10);   ch <- 1;   v := <-ch;   close(ch)
# Python:  q = queue.Queue();   q.put(1);   q.get()                  # thread-safe
#          q = asyncio.Queue();   await q.put(1);   await q.get()    # async
# TS:      // no native channels — use async generator + AbortController, or 'comlink' for workers
# Bash:    mkfifo /tmp/fifo;    echo hi > /tmp/fifo &    read line < /tmp/fifo    # named pipe
```

### Mutex / lock

```bash
# Rust:    let m = Arc::new(Mutex::new(0));   *m.lock().unwrap() += 1;
# Go:      var mu sync.Mutex;   mu.Lock(); defer mu.Unlock();   counter++
# Python:  lock = threading.Lock();   with lock: counter += 1
# TS:      // single-threaded JS — no mutex needed for in-process data
#          // for SharedArrayBuffer + Atomics, use Atomics.wait/notify
# Bash:    flock /tmp/lock -c 'cmd-needing-lock'                     # advisory file lock
```

### Atomic ops

```bash
# Rust:    use std::sync::atomic::{AtomicI32, Ordering};   counter.fetch_add(1, Ordering::SeqCst);
# Go:      atomic.AddInt32(&counter, 1);   atomic.LoadInt32(&counter)
# Python:  # no atomics on regular ints — GIL makes simple ops atomic; use threading.Lock for compound
# TS:      Atomics.add(int32Array, 0, 1)                              // SharedArrayBuffer required
# Bash:    # no atomics — flock or single-writer pattern
```

### Async / await

```bash
# Rust:    async fn fetch(u: &str) -> Result<R, E> { ... }    let r = fetch(u).await?;   // needs runtime
# Go:      // no async/await — goroutines + channels:
#          ch := make(chan Result, 1);   go func() { ch <- fetch(u) }();   r := <-ch
# Python:  async def fetch(u): ...;        r = await fetch(u);            # asyncio.run(main())
# TS:      async function fetch(u: string) { ... }   const r = await fetch(u);
# Bash:    # no async — background jobs:    fetch "$u" > /tmp/r &
```

### Cancellation / context / timeout

```bash
# Rust:    tokio::select! { _ = work() => ..., _ = tokio::time::sleep(Duration::from_secs(5)) => ... }
# Go:      ctx, cancel := context.WithTimeout(ctx, 5*time.Second);   defer cancel()
# Python:  await asyncio.wait_for(coro(), timeout=5.0)                # raises TimeoutError
# TS:      const ac = new AbortController();  setTimeout(() => ac.abort(), 5000);  fetch(u, { signal: ac.signal })
# Bash:    timeout 5 cmd                                               # GNU coreutils
```

## Modules / Imports

### Import a module

```bash
# Rust:    use std::collections::HashMap;          mod helpers;          use helpers::tool;
# Go:      import "fmt"                             import "github.com/x/y"
# Python:  from collections import OrderedDict     import json
# TS:      import { Foo } from "./foo";            import * as fs from "node:fs"
# Bash:    source ./lib.sh         or       . ./lib.sh                     # POSIX form
```

### Aliases

```bash
# Rust:    use std::collections::HashMap as Map;
# Go:      import f "fmt"                          // f.Println(...)
# Python:  import numpy as np
# TS:      import { veryLongName as v } from "./foo";
# Bash:    alias short=very_long_name              # not recommended for scripts (interactive only)
```

### Re-export / public visibility

```bash
# Rust:    pub use crate::inner::Foo;              // re-export at this level
#          pub fn / pub(crate) fn / fn            // visibility ladder
# Go:      capitalized identifiers are exported; lowercase = package-private
# Python:  __all__ = ["Foo", "bar"]                # convention; underscore prefix = private
# TS:      export { Foo };                          export default ...;     export * from "./foo"
# Bash:    export VAR                               # passes to subprocesses; no module visibility
```

### Path / file resolution

```bash
# Rust:    Cargo.toml deps + src/main.rs / src/lib.rs;   mod foo; loads src/foo.rs
# Go:      go.mod path + filesystem; folder = package
# Python:  sys.path search; __init__.py marks a package
# TS:      tsconfig.json paths + moduleResolution + node_modules; baseUrl + paths for aliases
# Bash:    relative or absolute path; PATH for executables; BASH_SOURCE for current file
```

## JSON

### Encode / decode

```bash
# Rust:    let v: T = serde_json::from_str(&s)?;     let s = serde_json::to_string(&v)?;
# Go:      var v T; err := json.Unmarshal(b, &v);    b, err := json.Marshal(v)
# Python:  v = json.loads(s);                         s = json.dumps(v, indent=2)
# TS:      const v: T = JSON.parse(s);               const s = JSON.stringify(v, null, 2)
# Bash:    v=$(jq -r '.field' <<<"$s");              s=$(jq -n --arg n "$x" '{name: $n}')
```

### Custom (de)serialization

```bash
# Rust:    #[derive(Serialize, Deserialize)] #[serde(rename_all = "camelCase")] struct T { ... }
# Go:      type T struct { Name string `json:"name,omitempty"` }
# Python:  class T:  def to_json(self): ...     # or pydantic / dataclasses-json
# TS:      JSON.stringify(v, (key, val) => ...)                        // replacer function
# Bash:    # no custom serialization — build with jq filter expressions
```

### Stream / parse partial

```bash
# Rust:    serde_json::Deserializer::from_reader(r).into_iter::<T>()
# Go:      dec := json.NewDecoder(r);  for dec.More() { dec.Decode(&t) }
# Python:  ijson.items(stream, "item")                                  # streaming parser
# TS:      // streaming JSON parsers are 3rd party (e.g., 'stream-json')
# Bash:    jq -c '.items[]' <<<"$json" | while read -r item; do ...; done
```

## File I/O

### Read whole file

```bash
# Rust:    let s = std::fs::read_to_string("p")?;            let b = std::fs::read("p")?;
# Go:      data, err := os.ReadFile("p")
# Python:  data = open("p").read()                            # closes on GC; prefer with: open(...) as f
# TS:      const data = await fs.readFile("p", "utf8")        // node: fs/promises
# Bash:    data=$(<p)              # bash; cat is a fork
```

### Write whole file

```bash
# Rust:    std::fs::write("p", contents)?;
# Go:      err := os.WriteFile("p", data, 0644)
# Python:  open("p", "w").write(s)                            # again, prefer with-statement
# TS:      await fs.writeFile("p", data, "utf8")
# Bash:    echo "$data" > p          printf '%s' "$data" > p
```

### Append

```bash
# Rust:    use std::io::Write;
#          let mut f = OpenOptions::new().append(true).open("p")?;
#          writeln!(f, "{line}")?;
# Go:      f, _ := os.OpenFile("p", os.O_APPEND|os.O_WRONLY, 0644);  fmt.Fprintln(f, line)
# Python:  with open("p", "a") as f: f.write(line + "\n")
# TS:      await fs.appendFile("p", line + "\n")
# Bash:    echo "$line" >> p
```

### Read line-by-line

```bash
# Rust:    use std::io::BufRead;
#          for line in BufReader::new(File::open("p")?).lines() { let line = line?; ... }
# Go:      f, _ := os.Open("p");  s := bufio.NewScanner(f);  for s.Scan() { line := s.Text() }
# Python:  with open("p") as f:
#              for line in f: process(line.rstrip())
# TS:      // node: stream + readline
#          const rl = readline.createInterface({ input: fs.createReadStream("p") });
#          for await (const line of rl) { ... }
# Bash:    while IFS= read -r line; do ...; done < p
```

### Binary

```bash
# Rust:    let bytes: Vec<u8> = std::fs::read("p")?
# Go:      data, _ := os.ReadFile("p")        // []byte
# Python:  data = open("p", "rb").read()      # bytes object
# TS:      const buf = await fs.readFile("p"); // Buffer
# Bash:    data=$(xxd -p < p);  printf '%s' "$bin_str" | xxd -r -p > p
```

## Process / Environment

### CLI args

```bash
# Rust:    let args: Vec<String> = std::env::args().collect();   // args[0] = program name
# Go:      args := os.Args                                        // os.Args[0] = program name
# Python:  import sys;  args = sys.argv                            # argv[0] = script path
# TS:      const args = process.argv;                              // [node, script, ...real args]
# Bash:    "$0" (script) "$1" "$2" ... "$@" (all)   "$#" (count)
```

### Environment variables

```bash
# Rust:    let v = std::env::var("KEY").unwrap_or_default();   std::env::set_var("KEY", "val");
# Go:      v := os.Getenv("KEY");                                os.Setenv("KEY", "val")
# Python:  v = os.environ.get("KEY", "");                         os.environ["KEY"] = "val"
# TS:      const v = process.env.KEY ?? "";                       process.env.KEY = "val"
# Bash:    v="${KEY:-}"                                            export KEY=val
```

### Stdin / stdout / stderr

```bash
# Rust:    let mut s = String::new();  std::io::stdin().read_line(&mut s)?;
#          println!("out");   eprintln!("err");
# Go:      r := bufio.NewReader(os.Stdin);   line, _ := r.ReadString('\n')
#          fmt.Println("out");   fmt.Fprintln(os.Stderr, "err")
# Python:  line = input();   print("out");   print("err", file=sys.stderr)
# TS:      // node: readline or process.stdin events;  console.log("out");  console.error("err")
# Bash:    read -r line;     echo "out";    echo "err" >&2
```

### Exit code

```bash
# Rust:    std::process::exit(2);                                // skips Drop!  prefer return from main
# Go:      os.Exit(2)                                            // skips defers
# Python:  sys.exit(2)
# TS:      process.exit(2)
# Bash:    exit 2
```

### Subprocess / shell out

```bash
# Rust:    let out = Command::new("git").args(["log", "-1"]).output()?;   // captures stdout/stderr
# Go:      out, err := exec.Command("git", "log", "-1").Output()
# Python:  out = subprocess.run(["git", "log", "-1"], capture_output=True, text=True, check=True).stdout
# TS:      const { stdout } = await execFile("git", ["log", "-1"])     // util.promisify(child_process.execFile)
# Bash:    out=$(git log -1)                                            # native shell-out
```

### Working directory

```bash
# Rust:    let cwd = std::env::current_dir()?;     std::env::set_current_dir("/tmp")?;
# Go:      cwd, _ := os.Getwd();                    os.Chdir("/tmp")
# Python:  cwd = os.getcwd();                       os.chdir("/tmp")
# TS:      const cwd = process.cwd();               process.chdir("/tmp")
# Bash:    cwd=$(pwd);                              cd /tmp
```

## Date & Time

### Now

```bash
# Rust:    let t = std::time::SystemTime::now();                          // chrono::Utc::now() for richer
# Go:      t := time.Now()
# Python:  t = datetime.datetime.now(tz=timezone.utc)
# TS:      const t = new Date()                                            // Date.now() for ms epoch
# Bash:    date "+%Y-%m-%dT%H:%M:%S%z"   # epoch:  date +%s
```

### Parse / format

```bash
# Rust:    let t: DateTime<Utc> = "2024-01-15T10:30:00Z".parse()?;        // chrono
# Go:      t, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")        // reference: Mon Jan 2 15:04:05 MST 2006
# Python:  t = datetime.fromisoformat("2024-01-15T10:30:00+00:00")
# TS:      const t = new Date("2024-01-15T10:30:00Z")
# Bash:    t=$(date -d "2024-01-15T10:30:00Z" +%s)                         # GNU date
```

### Arithmetic (add/subtract)

```bash
# Rust:    use chrono::Duration;   t + Duration::days(7)
# Go:      t.Add(7 * 24 * time.Hour)                                       // also AddDate(0, 0, 7)
# Python:  t + datetime.timedelta(days=7)
# TS:      new Date(t.getTime() + 7 * 86400 * 1000)                         // ms math
# Bash:    date -d "$t + 7 days" +%s                                        # GNU date
```

### Sleep / delay

```bash
# Rust:    std::thread::sleep(Duration::from_secs(1));     tokio::time::sleep(...).await
# Go:      time.Sleep(time.Second)
# Python:  time.sleep(1.0);   await asyncio.sleep(1.0)
# TS:      await new Promise(r => setTimeout(r, 1000))
# Bash:    sleep 1   # accepts 0.5 / 1m / 2h on GNU
```

## Numbers & Math

### Conversion

```bash
# Rust:    let n: i32 = "42".parse()?;     let s = n.to_string();
# Go:      n, _ := strconv.Atoi("42");     s := strconv.Itoa(n)
# Python:  n = int("42");                   s = str(n)
# TS:      const n = parseInt("42", 10);    const s = n.toString()
# Bash:    n="$num"             # already string; arith via $(( $num + 0 ))
```

### Powers / roots / log

```bash
# Rust:    x.powi(2);    x.sqrt();    x.ln();    x.log10()
# Go:      math.Pow(x, 2);  math.Sqrt(x);  math.Log(x);  math.Log10(x)
# Python:  x ** 2;          math.sqrt(x);  math.log(x);  math.log10(x)
# TS:      Math.pow(x, 2);  Math.sqrt(x);  Math.log(x);  Math.log10(x)
# Bash:    bc <<<"x^2";     bc -l <<<"sqrt($x)";  bc -l <<<"l($x)/l(10)"
```

### Random

```bash
# Rust:    use rand::Rng;     let n: u32 = rand::thread_rng().gen_range(0..100)
# Go:      n := rand.Intn(100)                                              // math/rand;  crypto/rand for cryptographic
# Python:  random.randint(0, 99);   secrets.randbelow(100)                  # secrets for cryptographic
# TS:      Math.floor(Math.random() * 100);   crypto.randomInt(0, 100)       // crypto.* for cryptographic
# Bash:    echo $(( RANDOM % 100 ));   shuf -i 0-99 -n 1
```

### Constants

```bash
# Rust:    std::f64::consts::PI;  std::f64::consts::E;  i32::MAX;  i32::MIN
# Go:      math.Pi;  math.E;       math.MaxInt32;        math.MinInt32
# Python:  math.pi;  math.e;       sys.maxsize;          float('inf')
# TS:      Math.PI;  Math.E;       Number.MAX_SAFE_INTEGER;  Number.NEGATIVE_INFINITY
# Bash:    pi=$(bc -l <<<"4*a(1)")                                          # arctan(1)*4
```

## Regex

### Compile / match

```bash
# Rust:    use regex::Regex;     let re = Regex::new(r"\d+").unwrap();   re.is_match(s)
# Go:      re := regexp.MustCompile(`\d+`);   re.MatchString(s)
# Python:  re.match(r"\d+", s);   re.search(...);    re.compile(r"\d+")
# TS:      /\d+/.test(s);          new RegExp("\\d+").test(s)
# Bash:    [[ "$s" =~ ^[0-9]+$ ]]                                         # ERE; no PCRE; ${BASH_REMATCH}
```

### Capture groups

```bash
# Rust:    let caps = re.captures(s).unwrap();   &caps[1]
# Go:      m := re.FindStringSubmatch(s);   m[1]
# Python:  m = re.search(r"(\d+)", s);   m.group(1)
# TS:      const m = s.match(/(\d+)/);   m && m[1]
# Bash:    [[ "$s" =~ ([0-9]+) ]] && echo "${BASH_REMATCH[1]}"
```

### Replace

```bash
# Rust:    re.replace_all(s, "X")
# Go:      re.ReplaceAllString(s, "X")
# Python:  re.sub(r"\d+", "X", s)
# TS:      s.replace(/\d+/g, "X")            // 'g' flag for all matches
# Bash:    sed 's/[0-9]\+/X/g' <<<"$s"
```

### Split

```bash
# Rust:    re.split(s).collect::<Vec<_>>()
# Go:      re.Split(s, -1)
# Python:  re.split(r"\s+", s)
# TS:      s.split(/\s+/)
# Bash:    awk '{ for (i=1;i<=NF;i++) print $i }' <<<"$s"
```

## HTTP Client

### GET / POST

```bash
# Rust:    let body = reqwest::blocking::get(url)?.text()?;                 // async: reqwest::Client
#          let res = client.post(url).json(&body).send().await?;
# Go:      resp, err := http.Get(url);   defer resp.Body.Close();   body, _ := io.ReadAll(resp.Body)
#          resp, err := http.Post(url, "application/json", bytes.NewReader(j))
# Python:  r = requests.get(url, timeout=10);   r.raise_for_status();   data = r.json()
#          r = requests.post(url, json=payload, timeout=10)
# TS:      const r = await fetch(url);   if (!r.ok) throw ...;   const data = await r.json()
#          await fetch(url, { method: "POST", body: JSON.stringify(p), headers: { "content-type": "application/json" } })
# Bash:    curl -fsSL "$url"                                                  # -f fails on 4xx/5xx
#          curl -X POST -H 'content-type: application/json' -d "$json" "$url"
```

### Headers & timeout

```bash
# Rust:    reqwest::Client::builder().timeout(Duration::from_secs(10)).build()?.get(u).header(...).send()
# Go:      client := &http.Client{Timeout: 10 * time.Second};  req, _ := http.NewRequest("GET", u, nil)
#          req.Header.Set("Authorization", "Bearer ...");      resp, _ := client.Do(req)
# Python:  requests.get(u, headers={"Authorization": "..."}, timeout=10)
# TS:      const ac = new AbortController();  setTimeout(() => ac.abort(), 10000)
#          fetch(u, { headers: { authorization: "..." }, signal: ac.signal })
# Bash:    curl -m 10 -H "Authorization: Bearer $tok" "$url"
```

## Type System

### Interfaces / traits / protocols

```bash
# Rust:    trait Display { fn fmt(&self, f: &mut Formatter) -> Result; }
#          impl Display for Foo { fn fmt(&self, f: &mut Formatter) -> Result { ... } }
# Go:      type Stringer interface { String() string }
#          // implicit satisfaction — no `implements` keyword
# Python:  from typing import Protocol
#          class Drawable(Protocol):  def draw(self) -> None: ...           # structural
# TS:      interface Drawable { draw(): void }                                // structural
# Bash:    # no type system
```

### Sum types / variants

```bash
# Rust:    enum Result<T, E> { Ok(T), Err(E) }                  // tagged union; exhaustive match
# Go:      // closest:  iota constants + type switch on interface
#          type Event interface { isEvent() }
# Python:  Status = Literal["pending", "active", "done"]
#          @dataclass class Click: x: int                       # match on dataclass
# TS:      type Event = { kind: "click"; x: number } | { kind: "key"; code: string }   // discriminated union
# Bash:    # no sum types — use string constants and case
```

### Generics

```bash
# Rust:    fn max<T: Ord>(a: T, b: T) -> T { if a > b { a } else { b } }
# Go:      func Max[T cmp.Ordered](a, b T) T { ... }                                     // 1.18+
# Python:  def first[T](xs: list[T]) -> T: return xs[0]                                  # 3.12+
# TS:      function max<T extends number>(a: T, b: T): T { ... }
# Bash:    # untyped — parametricity is automatic but unchecked
```

### Variance / subtyping

```bash
# Rust:    Lifetimes are covariant in some positions, contravariant/invariant in others. Compiler enforces.
# Go:      No subtype variance. Interface satisfaction is structural.
# Python:  TypeVar(bound=...);  TypeVar(covariant=True)                                  # mostly type-checker only
# TS:      Function parameters are bivariant under default; --strictFunctionTypes enables contravariance
# Bash:    n/a
```

## Build & Run

### Run a script / file

```bash
# Rust:    cargo run                          // compiled
# Go:      go run main.go                     // compiles + runs
# Python:  python3 script.py
# TS:      tsx script.ts                       // also: ts-node, deno run, bun run
# Bash:    bash script.sh    or    ./script.sh (with shebang + exec bit)
```

### Compile / build

```bash
# Rust:    cargo build --release              // target/release/<name>
# Go:      go build -o bin/app ./cmd/app      // -ldflags "-s -w" to strip
# Python:  # interpreted; pyinstaller / nuitka for binaries
# TS:      tsc --build                         // emits .js;  esbuild / swc / tsup for bundles
# Bash:    # interpreted; shc -f script.sh for an obfuscated binary (rarely a good idea)
```

### Add a dependency

```bash
# Rust:    cargo add reqwest --features rustls-tls
# Go:      go get github.com/foo/bar@latest
# Python:  pip install requests       # or: poetry add  /  uv add
# TS:      npm install foo            # or: pnpm add foo  /  yarn add foo  /  bun add foo
# Bash:    # none — call out to system tools (jq, curl, awk)
```

### Format / lint

```bash
# Rust:    cargo fmt;       cargo clippy -- -D warnings
# Go:      gofmt -w .;       go vet ./...;     golangci-lint run
# Python:  ruff format .;    ruff check .;      mypy .
# TS:      prettier --write .;     eslint .;     tsc --noEmit
# Bash:    shfmt -w .;       shellcheck script.sh
```

### Run tests

```bash
# Rust:    cargo test
# Go:      go test ./... -race
# Python:  pytest;   python -m unittest
# TS:      vitest;   jest;   node --test
# Bash:    bats test/;       shunit2 (lightweight)
```

## Testing primitives (bare minimum)

### Assert equal

```bash
# Rust:    assert_eq!(actual, expected);     assert!(cond, "msg");
# Go:      if got != want { t.Errorf("got %v want %v", got, want) }                    // stdlib testing
# Python:  assert actual == expected, f"got {actual!r}, want {expected!r}"
# TS:      expect(actual).toBe(expected)     // jest/vitest;  assert.strictEqual(...)  // node:assert
# Bash:    [[ "$actual" == "$expected" ]] || { echo "got $actual"; exit 1; }
```

### Setup / teardown

```bash
# Rust:    // each #[test] is independent; use fixtures crate or builder pattern
# Go:      func TestMain(m *testing.M) { setup(); code := m.Run(); teardown(); os.Exit(code) }
# Python:  @pytest.fixture def db(): yield connect(); db.close()
# TS:      beforeEach(() => ...);  afterEach(() => ...)        // jest/vitest
# Bash:    setUp() { ...; };  tearDown() { ...; }              # bats / shunit2 conventions
```

## Common Gotchas (cross-language)

```bash
# Numbers:     Rust panics on int overflow in debug, wraps in release. Go silently wraps.
#              Python ints are bigints. TS only has float64 (no real int — Number.MAX_SAFE_INTEGER 2^53-1).
#              Bash $(( )) is 64-bit signed and silently wraps.

# Equality:    bash `=` (string) vs `-eq` (int). JS `==` coerces — always `===`. Python `is` is identity.
#              Go interface == panics on uncomparable dynamic types. Rust f64 NaN != NaN.

# Truthy:      bash `[[ -n "0" ]]` is TRUE (non-empty string). Python `[]`, `{}`, `""`, `0`, `None` are falsy.
#              JS `0`, `""`, `NaN`, `null`, `undefined`, `false` are falsy — also `[]` and `{}` are TRUTHY.
#              Go and Rust have NO implicit boolean conversion — must be `bool` explicitly.

# Strings:     Rust/Go index by BYTES (panic vs broken on multi-byte). Python by codepoint.
#              TS by UTF-16 code unit (emoji span 2). Bash word-splits unquoted vars — quote "$var" always.

# Errors:      Rust Result + ?. Go (T, error). Python/TS exceptions. Bash $? + set -e + trap.
#              Mixing styles via FFI is where prod breaks.

# Concurrency: Go goroutines auto-multiplex, no GIL. Rust threads enforce Send/Sync at compile time.
#              Python GIL serializes Python bytecode (3.12 and below). TS is single-threaded — Workers for parallelism.
#              Bash has no shared memory between & jobs.

# Mutable defaults (Python):  def f(xs=[]): is a CLASSIC bug — list is shared across calls.
#                              Use xs=None and assign inside.

# Slice aliasing (Go):       append may share backing array with the source slice — copy if you need
#                              independence:  out := append([]int{}, src...).

# Shallow spread (TS/JS):    {...obj} is shallow. Use structuredClone(obj) for deep copy.

# Subshell scope (Bash):     count=0; cmd | while read l; do (( count++ )); done; echo $count   # still 0
#                              Fix: use process substitution `done < <(cmd)`.

# Map iteration order:       Go map iteration is RANDOMIZED on purpose. Sort keys when emitting.
#                              Python dict and TS Map preserve insertion order. Rust HashMap is unordered;
#                              use BTreeMap for sorted-by-key.
```

## Tips

- **Quote your bash variables.** Always `"$var"` and `"${arr[@]}"`. Word-splitting + globbing on unquoted expansions is the #1 source of shipped bash bugs.
- **Read the detail page** (`cs -d polyglot`) for the full landmine catalogue: null/empty/zero, equality, numbers, strings, truthiness, concurrency, errors, memory.
- **`set -euo pipefail`** at the top of every bash script.
- **`strictNullChecks: true`** in `tsconfig.json` is non-negotiable.
- **`from __future__ import annotations`** + a type checker (mypy/pyright) for any Python codebase > a few files.
- **`#![deny(warnings)]`** + `cargo clippy -- -D warnings` for Rust.
- **`go vet`** + `golangci-lint` + `-race` for Go tests.
- **One language at a time.** When porting, write the test in the target language first, then make it pass. Don't read-translate idioms — re-derive them from the test.

## See Also

- rust
- go
- python
- typescript
- javascript
- bash
- regex
- json
- programming-paradigms

## References

- The Rust Programming Language https://doc.rust-lang.org/book/
- Rust by Example https://doc.rust-lang.org/rust-by-example/
- Effective Go https://go.dev/doc/effective_go
- Go by Example https://gobyexample.com/
- Python Language Reference https://docs.python.org/3/reference/
- Python Standard Library https://docs.python.org/3/library/
- TypeScript Handbook https://www.typescriptlang.org/docs/handbook/
- TypeScript Deep Dive https://basarat.gitbook.io/typescript/
- Bash Reference Manual https://www.gnu.org/software/bash/manual/bash.html
- BashGuide https://mywiki.wooledge.org/BashGuide
- Rosetta Code https://rosettacode.org/ — same task in 800+ languages
- Hyperpolyglot https://hyperpolyglot.org/ — original side-by-side reference (sparser, older)
- Learn X in Y minutes https://learnxinyminutes.com/ — tutorial-style per-language overview
