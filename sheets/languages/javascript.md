# JavaScript (Programming Language)

The dynamic, multi-paradigm language of the web — runs in every browser, on Node.js, Deno, and Bun, with a 25-year history of accumulated quirks and a modern ECMAScript spec that ships new features every June.

## Setup & Runtimes

### Node.js (the workhorse)

```bash
# Install (use a version manager, never the system package):
# brew install fnm                                  // fast Node manager (Rust)
# curl -fsSL https://fnm.vercel.app/install | bash
# fnm install 22                                    // Node 22 — current LTS family
# fnm use 22
# node --version                                    // v22.x.x
#
# Or nvm:
# curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/master/install.sh | bash
# nvm install --lts
#
# What Node 20+ ships natively (no polyfills needed):
# - fetch / Request / Response / Headers
# - AbortController / AbortSignal / AbortSignal.timeout
# - Web Streams (ReadableStream, WritableStream, TransformStream)
# - structuredClone, queueMicrotask, performance.now
# - test runner (node:test), watch mode (--watch)
# - --experimental-strip-types (22.6+) — run .ts files with types erased
```

### Deno (TS-first secure runtime)

```bash
# curl -fsSL https://deno.land/install.sh | sh
# deno --version
#
# deno run script.js                                 // permissions OFF by default
# deno run --allow-net --allow-read script.js        // explicit grants
# deno run -A script.js                              // grant all (lazy mode)
#
# Deno ships:
# - TypeScript without configuration
# - npm: and jsr: imports (no node_modules)
# - deno fmt / deno lint / deno test / deno bench
# - deno compile script.js -o myapp                  // single binary
# - Deno.serve, Deno.readTextFile, Deno.env — Web standards-aligned APIs
```

### Bun (the fast all-in-one)

```bash
# curl -fsSL https://bun.sh/install | bash
# bun --version
#
# bun run script.js                                  // launches in ~2ms
# bun script.js                                      // shorthand
# bun --hot script.js                                // hot reload
# bun test                                           // built-in test runner
# bun build ./src/index.js --outdir ./dist           // bundler
# bun install                                        // npm-compatible installer
#
# Bun.file, Bun.serve, Bun.write, Bun.password, Bun.spawn — native fast APIs
```

### Browsers (the original target)

```bash
# Modern evergreen browsers (Chrome, Firefox, Safari, Edge) ship ES2023+ natively.
# Module syntax in <script>:
# <script type="module" src="app.js"></script>
# // app.js can use import/export with no bundler in dev
#
# Older browser support is a build-tool problem (esbuild, Vite, webpack) — modern code
# is written ESM and downleveled at build time.
```

### ESM is the default

```bash
# Modern: package.json "type": "module"  → all .js files are ESM (import/export).
# Legacy: package.json "type": "commonjs" or omitted → .js files are CommonJS (require).
#
# File-extension override:
# - .mjs        always ESM
# - .cjs        always CommonJS
# - .js         depends on nearest package.json "type"
#
# Initialize a modern Node project:
# mkdir my-app && cd my-app
# npm init -y
# npm pkg set type=module
# echo 'console.log("hello, esm")' > index.js
# node index.js
```

## Variables

### const / let / var

```bash
# const PI = 3.14;               // block-scoped, immutable BINDING (object contents still mutable)
# let count = 0;                 // block-scoped, mutable
# count = 1;                     // OK
#
# var legacy = 1;                // function-scoped, hoisted, no temporal dead zone — avoid
#
# // const objects: the binding is immutable, the contents are not.
# const cfg = { port: 80 };
# cfg.port = 443;                // OK — mutating contents
# // cfg = {};                   // ERROR — reassigning binding
#
# // Use Object.freeze to prevent contents mutation (shallow):
# const FROZEN = Object.freeze({ port: 80 });
# // FROZEN.port = 443;          // silently fails in non-strict; throws in strict
```

### Hoisting and the Temporal Dead Zone

```bash
# // var declarations hoist with undefined initialization:
# console.log(x);                // undefined — NOT a ReferenceError
# var x = 1;
#
# // let / const hoist but stay in the TDZ until the line that initializes them:
# console.log(y);                // ReferenceError — TDZ
# let y = 1;
#
# // Function declarations hoist FULLY (callable above their definition):
# foo();                         // OK
# function foo() { return 1; }
#
# // Function expressions and arrow functions do NOT (they hoist as variables):
# bar();                         // TypeError: bar is not a function (var) or TDZ (let/const)
# const bar = () => 1;
```

### Block scoping

```bash
# {
#   const a = 1;                 // a only exists inside this block
#   let b = 2;
# }
# // console.log(a, b);          // ReferenceError
#
# // var leaks out — historical footgun:
# function leak() {
#   if (true) { var leaked = 1; }
#   return leaked;               // 1 — var ignored the block
# }
```

## Primitive Types

### The seven primitives

```bash
# typeof "hello"                 // "string"     — UTF-16 code units, immutable
# typeof 42                      // "number"     — IEEE-754 float64
# typeof 9007199254740993n       // "bigint"     — arbitrary-precision integer
# typeof true                    // "boolean"
# typeof undefined               // "undefined"  — uninitialized / missing
# typeof null                    // "object"     — historical bug, kept for compat
# typeof Symbol("id")            // "symbol"     — unique opaque identifier
#
# // Wrapper objects exist (String, Number, Boolean) but ALMOST NEVER use them:
# const bad = new String("x");   // typeof bad === "object"  — surprises everywhere
# const good = "x";              // typeof good === "string" — what you want
```

### null vs undefined

```bash
# // Convention:
# // - undefined: the runtime did not give a value (uninitialized var, missing arg)
# // - null:      explicitly "no value here, on purpose"
#
# function f(a, b) { console.log(a, b); }
# f(1);                          // 1 undefined  — b never given
#
# const obj = { x: null };       // null = "x exists, but the value is intentionally absent"
#
# // == treats them as equal; === does not:
# null == undefined              // true
# null === undefined             // false
#
# // The classic "is this nullish" check:
# if (x == null) { /* x is null OR undefined */ }
```

## Numbers

### Only float64 (until BigInt)

```bash
# // Every "number" is an IEEE-754 double (64-bit float).
# // Integers up to 2^53 - 1 are exact; beyond that, you lose precision.
#
# Number.MAX_SAFE_INTEGER        // 9007199254740991  (2^53 - 1)
# Number.MIN_SAFE_INTEGER        // -9007199254740991
# Number.EPSILON                 // 2.220446049250313e-16  — smallest 1+x distinguishable
# Number.MAX_VALUE               // 1.7976931348623157e+308
# Number.MIN_VALUE               // 5e-324  — smallest positive (subnormal)
#
# // The classic float trap:
# 0.1 + 0.2                      // 0.30000000000000004
# 0.1 + 0.2 === 0.3              // false
#
# // Compare with epsilon for "almost equal":
# Math.abs(0.1 + 0.2 - 0.3) < Number.EPSILON  // true
```

### BigInt — arbitrary precision integers

```bash
# const big = 9007199254740993n;          // n suffix = bigint literal
# const bigger = BigInt("12345678901234567890");
# big + 1n                                // 9007199254740994n
#
# // CANNOT mix bigint and number:
# // big + 1                              // TypeError: Cannot mix BigInt and other types
# big + BigInt(1)                         // OK
#
# // typeof bigint === "bigint"
# // No Math.* support: Math.sqrt(4n) → TypeError
#
# // JSON.stringify can't serialize bigint by default:
# // JSON.stringify({ n: 1n })            // TypeError
# // Workaround:
# JSON.stringify({ n: 1n }, (k, v) => typeof v === "bigint" ? v.toString() : v);
```

### isNaN vs Number.isNaN

```bash
# // The legacy global isNaN coerces first, which gives wrong answers:
# isNaN("hello")                 // true   — "hello" coerces to NaN, then is NaN
# isNaN(undefined)               // true   — same trap
#
# // Number.isNaN is type-strict — only true for actual NaN:
# Number.isNaN("hello")          // false  — string is not NaN, it's a string
# Number.isNaN(NaN)              // true
# Number.isNaN(0/0)              // true
#
# // Use Number.isNaN. The global isNaN is broken by design.
# // Same for Number.isFinite vs the global isFinite — always prefer the static method.
#
# // NaN is not equal to itself — the only such value:
# NaN === NaN                    // false
# // So use Number.isNaN(x) instead of x === NaN.
```

### Numeric helpers

```bash
# // Rounding family — pick by intent, not "what feels right":
# Math.floor(2.7)                // 2     — toward -Infinity
# Math.ceil(-2.3)                // -2    — toward +Infinity
# Math.round(2.5)                // 3     — half to even? NO — half AWAY from zero (positive)
# Math.round(-2.5)               // -2    — note asymmetry
# Math.trunc(2.7)                // 2     — drop fraction
# Math.trunc(-2.7)               // -2    — toward zero
# Math.sign(-5)                  // -1
# Math.sign(0)                   // 0
# Math.sign(NaN)                 // NaN
#
# // Integer / float parsing:
# parseInt("42px")               // 42    — stops at first non-digit
# parseInt("0x10")               // 16    — hex auto-detected
# parseInt("0o10")               // 0     — DOES NOT recognize 0o; prefer Number()
# parseInt("10", 2)              // 2     — explicit radix; ALWAYS pass it
# Number.parseInt === parseInt   // true  — same function, prefer Number.parseInt for clarity
# Number("42px")                 // NaN   — strict; the right answer for validation
# +"42"                          // 42    — unary plus = Number(s)
# +"42px"                        // NaN
# parseFloat("3.14abc")          // 3.14
```

## Strings

### Template literals

```bash
# const name = "world";
# const greet = `Hello, ${name}!`;            // interpolation
# const multi = `line one
# line two`;                                  // newlines preserved
# const expr = `1 + 1 = ${1 + 1}`;            // any JS expression in ${}
#
# // Tagged templates — function gets raw chunks + interpolations:
# function html(strings, ...values) {
#   return strings.reduce((acc, s, i) => acc + s + (values[i] ?? ""), "");
# }
# const safe = html`<p>${name}</p>`;
#
# // String.raw — get the literal text without escape processing:
# console.log(String.raw`\n is two characters`);  // "\n is two characters" — backslash-n
# console.log(`\n is one character`);              // newline + " is one character"
```

### No char type — strings are UTF-16

```bash
# // JavaScript has no char/byte type. A single character is a 1-char string.
# const c = "a";                 // length 1
# typeof c                       // "string"
# c.charCodeAt(0)                // 97
# String.fromCharCode(97)        // "a"
# String.fromCodePoint(0x1F600)  // "😀" — full unicode
#
# // .length counts UTF-16 CODE UNITS, not characters:
# "café".length                  // 4 — OK, café fits in BMP
# "😀".length                    // 2 — surrogate pair! NOT 1
# "🇺🇸".length                    // 4 — flag = two regional indicators, each a surrogate pair
#
# // To count actual characters/grapheme clusters:
# [..."😀"].length               // 1 — spread iterates code points
# Array.from("😀").length        // 1
# // [..."🇺🇸"].length           // 2 — code points, not graphemes
#
# // Real grapheme cluster counting needs Intl.Segmenter:
# const seg = new Intl.Segmenter();
# [...seg.segment("🇺🇸")].length   // 1 — true grapheme count
```

### String methods

```bash
# // Test:
# "hello".includes("ell")        // true
# "hello".startsWith("he")       // true
# "hello".endsWith("lo")         // true
# "hello".indexOf("l")           // 2  — first match, -1 if missing
# "hello".lastIndexOf("l")       // 3
#
# // Slice / extract:
# "hello".slice(1, 4)            // "ell" — supports negatives: slice(-3) = "llo"
# "hello".substring(1, 4)        // "ell" — no negatives, swaps args if start > end
# "hello".at(-1)                 // "o"   — modern indexed access with negatives
#
# // Transform (always returns NEW string — strings are immutable):
# "Hello".toUpperCase()          // "HELLO"
# "HELLO".toLowerCase()          // "hello"
# "  hi  ".trim()                // "hi"
# "  hi  ".trimStart()           // "hi  "
# "  hi  ".trimEnd()             // "  hi"
# "ab".repeat(3)                 // "ababab"
# "5".padStart(3, "0")           // "005"
# "x".padEnd(4, "-")             // "x---"
#
# // Split / join / replace:
# "a,b,c".split(",")             // ["a","b","c"]
# "a,b,c".split(/,/g)            // same with regex
# ["a","b","c"].join("-")        // "a-b-c"
# "a.b.c".replace(".", "/")      // "a/b.c" — first only
# "a.b.c".replaceAll(".", "/")   // "a/b/c"
# "abc".replace(/./g, "*")       // "***" — replace + global regex
```

## Arrays

### Creation

```bash
# const xs = [1, 2, 3];
# const empty = [];
# const sized = new Array(5);                  // [<5 empty slots>] — sparse, weird
# const filled = Array(5).fill(0);             // [0, 0, 0, 0, 0]
# const range = Array.from({ length: 5 }, (_, i) => i);   // [0,1,2,3,4]
# const fromStr = Array.from("hi");            // ["h", "i"]
# const fromSet = Array.from(new Set([1, 1, 2]));  // [1, 2]
# const cloned = [...xs];                      // shallow copy
# const concat = [...xs, ...filled];           // merge
# const ofVals = Array.of(7);                  // [7]  — unlike Array(7) which is sparse
```

### Mutation methods (modify in place)

```bash
# const a = [1, 2, 3];
# a.push(4);                     // [1,2,3,4]    — appends, returns new length
# a.pop();                       // 4            — removes/returns last
# a.unshift(0);                  // [0,1,2,3]    — prepends, returns new length
# a.shift();                     // 0            — removes/returns first
# a.splice(1, 1);                // [2]          — removes 1 at index 1; mutates a
# a.splice(1, 0, "x");           // []           — inserts "x" at index 1
# a.reverse();                   // mutates and returns
# a.sort();                      // mutates — DEFAULT IS LEXICOGRAPHIC (string compare)!
#
# // Sort gotcha:
# [10, 2, 1].sort()              // [1, 10, 2]   — strings: "1" < "10" < "2"
# [10, 2, 1].sort((a, b) => a - b)   // [1, 2, 10]   — numeric ascending
# [10, 2, 1].sort((a, b) => b - a)   // [10, 2, 1]   — numeric descending
#
# // Non-mutating ES2023 alternatives — preserve original:
# const sorted = [10, 2, 1].toSorted((a, b) => a - b);   // [1,2,10], original unchanged
# const reversed = a.toReversed();
# const spliced = a.toSpliced(1, 1);
# const replaced = a.with(0, "X");                       // copy with index 0 = "X"
```

### Non-mutating transforms (return new arrays)

```bash
# const xs = [1, 2, 3, 4, 5];
#
# xs.slice(1, 3)                          // [2, 3]
# xs.concat([6, 7])                       // [1,2,3,4,5,6,7]
# xs.flat()                               // [1,2,3,4,5]
# [[1,2],[3,4]].flat()                    // [1,2,3,4]
# [[[1]],[[2]]].flat(Infinity)            // [1,2] — flatten any depth
# xs.flatMap(n => [n, n * 2])             // [1,2,2,4,3,6,4,8,5,10]
#
# xs.map(n => n * 2)                      // [2,4,6,8,10]
# xs.filter(n => n % 2 === 0)             // [2, 4]
# xs.reduce((acc, n) => acc + n, 0)       // 15  — sum
# xs.reduce((acc, n) => Math.max(acc, n), -Infinity)   // 5
# xs.reduceRight((acc, n) => acc + n, 0)  // 15  — right-to-left
#
# // Search:
# xs.find(n => n > 3)                     // 4   — first match (or undefined)
# xs.findLast(n => n > 3)                 // 5   — last match
# xs.findIndex(n => n > 3)                // 3
# xs.findLastIndex(n => n > 3)            // 4
# xs.includes(3)                          // true
# xs.indexOf(3)                           // 2   — strict equality only
#
# // Predicates:
# xs.some(n => n > 4)                     // true
# xs.every(n => n > 0)                    // true
# xs.length === 0                         // empty check
```

### Iteration

```bash
# // for-of — values, supports break/continue, the right default:
# for (const x of xs) console.log(x);
#
# // forEach — concise but: no break, no async/await:
# xs.forEach(x => console.log(x));
#
# // forEach with index:
# xs.forEach((x, i) => console.log(i, x));
#
# // Indexed for — fastest, full control:
# for (let i = 0; i < xs.length; i++) { /* ... */ }
#
# // Object.entries pattern for index + value:
# for (const [i, x] of xs.entries()) console.log(i, x);
#
# // NEVER for-in on arrays — iterates keys (strings) and includes inherited props:
# for (const k in xs) { /* DON'T */ }
```

## Objects

### Object literals

```bash
# const u = { name: "Alice", age: 30 };
# const empty = {};
#
# // Computed keys:
# const k = "status";
# const obj = { [k]: "active", [`${k}_at`]: Date.now() };
#
# // Shorthand (variable name → key):
# const name = "Alice", age = 30;
# const u2 = { name, age };               // { name: "Alice", age: 30 }
#
# // Method shorthand:
# const calc = {
#   add(a, b) { return a + b; },          // shorthand for: add: function(a, b) {...}
#   sub: (a, b) => a - b,
# };
#
# // Spread (shallow merge):
# const updated = { ...u, age: 31 };      // { name: "Alice", age: 31 }
# const merged = { ...defaults, ...overrides };
```

### Property access

```bash
# u.name                          // "Alice"   — dot notation (static keys)
# u["name"]                       // "Alice"   — bracket notation (dynamic keys)
# const k = "name"; u[k]          // "Alice"
#
# // Optional chaining (null/undefined-safe):
# u?.address?.city                // undefined — no TypeError
# u?.greet?.()                    // safe call
# arr?.[0]                        // safe indexed access
#
# // Existence:
# "name" in u                     // true   — checks prototype chain too
# Object.hasOwn(u, "name")        // true   — own properties only (modern)
# u.hasOwnProperty("name")        // true   — older, can be shadowed
#
# // Delete:
# delete u.age                    // removes the property
```

### Destructuring

```bash
# // Object destructuring:
# const { name, age } = u;
# const { name: userName } = u;            // rename: userName = u.name
# const { name, role = "user" } = u;       // default if undefined
# const { name, ...rest } = u;             // rest = everything else
#
# // Array destructuring:
# const [a, b, c] = [1, 2, 3];
# const [first, ...tail] = [1, 2, 3];      // tail = [2, 3]
# const [, , third] = [1, 2, 3];           // skip elements
# const [x = 0, y = 0] = [];               // defaults
#
# // Nested:
# const { user: { name }, items: [first] } = response;
#
# // Swap without temp:
# let p = 1, q = 2;
# [p, q] = [q, p];                         // p=2, q=1
```

### Object static methods

```bash
# Object.keys(u)                  // ["name", "age"]
# Object.values(u)                // ["Alice", 30]
# Object.entries(u)               // [["name","Alice"], ["age",30]]
# Object.fromEntries([["a",1]])   // { a: 1 }
#
# Object.assign({}, a, b, c)      // shallow merge into new object
# Object.freeze(u)                // SHALLOW immutability
# Object.isFrozen(u)              // true
# Object.seal(u)                  // existing keys mutable, no new keys
#
# Object.create(proto)            // new object with given prototype
# Object.getPrototypeOf(u)        // u's prototype
# Object.setPrototypeOf(u, p)     // change prototype (slow — avoid in hot paths)
#
# // structuredClone — deep clone with Date/Map/Set/Blob/etc support:
# const deep = structuredClone(u);
# // BEATS JSON.parse(JSON.stringify(u)) — handles cycles, dates, typed arrays.
# // Does NOT clone functions or class instances (DataCloneError).
#
# // Object.groupBy (ES2024) — group an array by a key function:
# Object.groupBy([{type:"a", n:1}, {type:"b", n:2}, {type:"a", n:3}], x => x.type);
# // { a: [{...},{...}], b: [{...}] }
```

## Map / Set / WeakMap / WeakSet

### Map — keyed collection, any key type

```bash
# const m = new Map();
# m.set("name", "Alice");
# m.set(42, "answer");
# m.set({ id: 1 }, "object key — only THIS object instance hits");
#
# m.get("name")                   // "Alice"
# m.has(42)                       // true
# m.delete(42)
# m.size                          // O(1)  — Object has no equivalent
# m.clear()
#
# // Init from entries:
# const m2 = new Map([["a", 1], ["b", 2]]);
# const m3 = new Map(Object.entries({ a: 1, b: 2 }));
#
# // Iteration is INSERTION ORDER (Object's order is loosely defined):
# for (const [k, v] of m) console.log(k, v);
# for (const k of m.keys()) {}
# for (const v of m.values()) {}
#
# // Convert back to object:
# const obj = Object.fromEntries(m);       // only works for string/symbol keys
#
# // Use Map when:
# // - keys are non-string (objects, instances, primitives mixed)
# // - you need .size frequently
# // - you'll iterate and want guaranteed insertion order
```

### Set — unique values

```bash
# const s = new Set([1, 2, 3, 2, 1]);     // Set(3) {1, 2, 3}
# s.add(4)
# s.has(2)                                 // true
# s.delete(1)
# s.size                                   // 3
# [...s]                                   // [2, 3, 4]
#
# // Dedupe pattern:
# const unique = [...new Set([1, 1, 2, 3])];   // [1, 2, 3]
#
# // ES2025 set algebra (Node 22+, Deno, Bun, Safari/Chrome/Firefox 2024+):
# const a = new Set([1, 2, 3]);
# const b = new Set([2, 3, 4]);
# a.union(b)                               // Set {1,2,3,4}
# a.intersection(b)                        // Set {2,3}
# a.difference(b)                          // Set {1}
# a.symmetricDifference(b)                 // Set {1,4}
# a.isSubsetOf(b)                          // false
# a.isSupersetOf(b)                        // false
# a.isDisjointFrom(b)                      // false
```

### WeakMap / WeakSet — GC-friendly metadata

```bash
# // Keys MUST be objects. Entries vanish when key is GC'd. NOT iterable. No .size.
# const meta = new WeakMap();
# function tag(obj, label) { meta.set(obj, label); }
# function getTag(obj) { return meta.get(obj); }
#
# {
#   const x = {};
#   tag(x, "temp");
#   // when x leaves scope and nothing references it, the WeakMap entry collects.
# }
#
# // Use cases:
# // - Attach private data to instances without preventing GC
# // - Cache derived values per object (lookups, parsed forms)
# // - Track "have I seen this DOM node" without leaking
#
# // WeakSet — same idea, just a set:
# const seen = new WeakSet();
# function visit(node) {
#   if (seen.has(node)) return;
#   seen.add(node);
#   // process(node);
# }
```

## Iterators & Iterables

### The protocol

```bash
# // An ITERABLE has [Symbol.iterator]() returning an ITERATOR.
# // An ITERATOR has next() returning { value, done }.
#
# const range = {
#   from: 1,
#   to: 5,
#   [Symbol.iterator]() {
#     let i = this.from;
#     const end = this.to;
#     return {
#       next() {
#         return i <= end ? { value: i++, done: false } : { value: undefined, done: true };
#       },
#     };
#   },
# };
#
# for (const n of range) console.log(n);   // 1 2 3 4 5
# [...range]                                // [1,2,3,4,5]
# Array.from(range)                         // [1,2,3,4,5]
# const [a, b, ...rest] = range;            // works because spread iterates
#
# // Built-in iterables:
# // Array, String, Map, Set, TypedArray, NodeList, HTMLCollection (in modern browsers).
# // Plain Object is NOT iterable — use Object.entries(obj) to iterate.
```

### Async iterables — for-await-of

```bash
# // The async cousin: [Symbol.asyncIterator]() returning a promise-based iterator.
# // Streams in Node implement this — you can for-await-of over a readable stream.
#
# import { createReadStream } from "node:fs";
# import readline from "node:readline";
#
# const rl = readline.createInterface({ input: createReadStream("data.txt") });
# for await (const line of rl) {
#   console.log(line);
# }
```

## Generators

### function* and yield

```bash
# // Generator function — produces values lazily:
# function* range(start, end) {
#   for (let i = start; i <= end; i++) yield i;
# }
#
# const g = range(1, 3);
# g.next()                       // { value: 1, done: false }
# g.next()                       // { value: 2, done: false }
# g.next()                       // { value: 3, done: false }
# g.next()                       // { value: undefined, done: true }
#
# // Spread / for-of work because generators are iterables:
# [...range(1, 5)]                // [1,2,3,4,5]
# for (const n of range(1, 5)) {}
#
# // yield* — delegate to another iterable:
# function* both() {
#   yield* range(1, 3);
#   yield* range(10, 12);
# }
# [...both()]                     // [1,2,3,10,11,12]
#
# // Two-way communication — value passed back into yield:
# function* echo() {
#   const x = yield "ready";
#   yield `got ${x}`;
# }
# const e = echo();
# e.next();                       // { value: "ready", done: false }
# e.next("hello");                // { value: "got hello", done: false }
```

### async function*

```bash
# // Async generator — yields promises that resolve to values.
# async function* fetchPages(url) {
#   let next = url;
#   while (next) {
#     const res = await fetch(next);
#     const page = await res.json();
#     yield page;
#     next = page.nextUrl;
#   }
# }
#
# for await (const page of fetchPages("/api/start")) {
#   console.log(page.items);
# }
```

## Control Flow & Switch

### if / else / ternary

```bash
# if (cond) { /* ... */ } else if (cond2) { /* ... */ } else { /* ... */ }
#
# // Ternary — single expression:
# const label = score > 50 ? "pass" : "fail";
#
# // Avoid nested ternaries — use early returns or extract a function.
#
# // Truthy/falsy controls:
# // Falsy: false, 0, -0, 0n, "", null, undefined, NaN
# // Everything else: TRUTHY (including [], {}, "0", "false")
# if ([]) console.log("empty array is truthy");      // logs!
# if ({}) console.log("empty object is truthy");     // logs!
```

### switch — DOES need break

```bash
# // Unlike Go, switch in JS falls through if you don't break:
# switch (status) {
#   case "active":
#     console.log("on");
#     break;                      // critical — without this, falls into "pending"
#   case "pending":
#     console.log("waiting");
#     break;
#   case "done":
#   case "complete":              // intentional fall-through stack
#     console.log("finished");
#     break;
#   default:
#     console.log("unknown");
# }
#
# // Use return inside case to avoid break:
# function label(s) {
#   switch (s) {
#     case "a": return 1;
#     case "b": return 2;
#     default:  return 0;
#   }
# }
#
# // Use eslint's no-fallthrough rule. The default is to allow fallthrough — turn it off.
# // ESLint comment to mark intentional:
# //   case "x": doX();
# //   // falls through
# //   case "y": doY();
```

## Loops

### for / while / do-while

```bash
# // Indexed for — most performant for arrays:
# for (let i = 0; i < arr.length; i++) { /* ... */ }
#
# // Cache length when array is large and not modified:
# for (let i = 0, n = arr.length; i < n; i++) { /* ... */ }
#
# // while — body runs zero or more times:
# while (queue.length) { process(queue.shift()); }
#
# // do-while — body runs at least once:
# do { ask(); } while (!valid);
#
# // Infinite loop with break:
# while (true) {
#   const tok = next();
#   if (tok === null) break;
#   handle(tok);
# }
```

### for-of (values)

```bash
# // The right default for arrays, strings, Map, Set, generators:
# for (const x of arr) console.log(x);
# for (const ch of "abc") console.log(ch);          // unicode-safe iteration
# for (const [k, v] of map) console.log(k, v);
# for (const v of set) console.log(v);
#
# // With index:
# for (const [i, x] of arr.entries()) console.log(i, x);
#
# // for-of supports break / continue / return (forEach does not).
```

### for-in — beware

```bash
# // for-in iterates ENUMERABLE STRING-KEYED properties, including inherited.
# // Original use case: object keys. NEVER use on arrays.
#
# const obj = { a: 1, b: 2 };
# for (const k in obj) console.log(k);              // "a", "b"
#
# // The trap with arrays:
# const arr = [1, 2, 3];
# arr.foo = "x";                                    // arrays are objects
# for (const i in arr) console.log(i);              // "0", "1", "2", "foo"
#
# // The right way to iterate object keys:
# for (const k of Object.keys(obj)) console.log(k);
# for (const [k, v] of Object.entries(obj)) console.log(k, v);
```

### .forEach quirks

```bash
# arr.forEach(x => console.log(x));
#
# // forEach DOES NOT WAIT for async callbacks — they all start in parallel and the
# // surrounding function continues immediately:
# async function broken() {
#   [1, 2, 3].forEach(async n => {
#     await save(n);                                // each starts; forEach returns immediately
#   });
#   console.log("done?");                           // logs BEFORE saves complete
# }
#
# // Fix — use for-of which awaits sequentially:
# async function fixed() {
#   for (const n of [1, 2, 3]) {
#     await save(n);                                // sequential, awaited
#   }
# }
#
# // Or Promise.all for parallel:
# await Promise.all([1, 2, 3].map(save));
```

## Functions

### Declaration / expression / arrow

```bash
# // Declaration — hoisted, named:
# function add(a, b) { return a + b; }
#
# // Expression — assigned to variable, NOT hoisted:
# const sub = function (a, b) { return a - b; };
# const sub2 = function namedSub(a, b) { return a - b; };  // name visible only inside
#
# // Arrow — concise, inherits this, no arguments object:
# const mul = (a, b) => a * b;
# const sq = n => n * n;                             // single arg: parens optional
# const id = x => x;
# const noop = () => {};
# const obj = () => ({ x: 1 });                      // wrap object in parens (else read as block)
# const longBody = (a, b) => {
#   const sum = a + b;
#   return sum * 2;                                  // explicit return required with braces
# };
```

### Default / rest / destructured params

```bash
# function greet(name = "world", greeting = "hello") {
#   return `${greeting}, ${name}`;
# }
# greet()                                  // "hello, world"
# greet("Ada")                             // "hello, Ada"
# greet(undefined, "hi")                   // "hi, world"  — undefined triggers default
# greet(null)                              // "hello, null" — null does NOT trigger default
#
# // Rest params (variadic):
# function sum(...nums) { return nums.reduce((a, b) => a + b, 0); }
# sum(1, 2, 3)                             // 6
#
# // Spread on call:
# const vals = [1, 2, 3];
# sum(...vals)                             // 6
#
# // Destructured / "named" params:
# function createUser({ name, age = 0, role = "user" } = {}) {
#   return { name, age, role };
# }
# createUser({ name: "Ada" })              // { name: "Ada", age: 0, role: "user" }
# createUser()                              // { name: undefined, age: 0, role: "user" }
#                                           // because of the = {} default
```

### IIFE — immediately invoked function expression

```bash
# // Pre-ESM idiom for module scope:
# (function () {
#   const private = "scoped";
#   // ...
# })();
#
# // Arrow form:
# (() => {
#   const x = 1;
# })();
#
# // Async IIFE — top-level await alternative for non-modules:
# (async () => {
#   const data = await fetch("/api").then(r => r.json());
#   console.log(data);
# })();
#
# // In ESM, you don't need IIFEs — the file IS the scope, and top-level await works.
```

## `this` Binding

### The four rules

```bash
# // 1. Default: `this` is undefined (strict) or globalThis (sloppy)
# function f() { return this; }
# f();                                     // undefined in strict mode (modules ARE strict)
#
# // 2. Implicit (method call): this = the object before the dot
# const o = { x: 1, get() { return this.x; } };
# o.get();                                 // 1 — `this` = o
#
# const lost = o.get;
# lost();                                  // undefined — `this` lost on detachment
#
# // 3. Explicit (call/apply/bind):
# function show() { return this.label; }
# show.call({ label: "A" });               // "A"
# show.apply({ label: "B" });              // "B"
# const bound = show.bind({ label: "C" });
# bound();                                 // "C" — bound permanently
#
# // 4. new — `this` = the freshly created object
# function Point(x, y) { this.x = x; this.y = y; }
# const p = new Point(1, 2);               // { x: 1, y: 2 }
```

### Arrow functions inherit `this`

```bash
# // Arrows DO NOT have their own `this`. They use the lexical scope's.
# class Counter {
#   constructor() { this.n = 0; }
#   incArrow = () => { this.n++; };        // arrow class field — bound to instance forever
#   incRegular() { this.n++; }             // regular method — `this` depends on call site
# }
# const c = new Counter();
# const f = c.incRegular;
# f();                                     // ERROR — this is undefined in strict
# const g = c.incArrow;
# g();                                     // OK — captured `this`
#
# // Common React/event handler bug:
# class Button {
#   constructor() { this.label = "go"; }
#   handle() { console.log(this.label); }
# }
# const b = new Button();
# document.addEventListener("click", b.handle);   // BROKEN — `this` is the document
# document.addEventListener("click", b.handle.bind(b));   // OK
# document.addEventListener("click", () => b.handle());   // OK — arrow captures b
```

## Closures

### Lexical scope

```bash
# // A closure is a function that remembers the scope where it was defined.
# function makeCounter() {
#   let n = 0;
#   return () => ++n;                      // captures `n`
# }
# const c = makeCounter();
# c(); c(); c();                           // 1, 2, 3 — `n` lives on
#
# // Each call creates a new closure:
# const a = makeCounter();
# const b = makeCounter();
# a(); a(); b();                           // a=2, b=1 — independent
```

### The classic loop-variable bug

```bash
# // Classic var-bug — all timeouts log 3:
# for (var i = 0; i < 3; i++) {
#   setTimeout(() => console.log(i), 0);
# }
# // Output: 3, 3, 3 — they all share the same `i`
#
# // Fix with let — block-scoped, new binding per iteration:
# for (let i = 0; i < 3; i++) {
#   setTimeout(() => console.log(i), 0);
# }
# // Output: 0, 1, 2
#
# // Pre-let workaround was IIFE per iteration:
# for (var i = 0; i < 3; i++) {
#   (function (j) {
#     setTimeout(() => console.log(j), 0);
#   })(i);
# }
```

### Module pattern (closure-based privacy)

```bash
# // Pre-class privacy via closure:
# const counter = (function () {
#   let n = 0;
#   return {
#     inc() { n++; },
#     get() { return n; },
#   };
# })();
# counter.inc();
# counter.get();                           // 1
# // counter.n is undefined — `n` is closed-over, inaccessible from outside.
```

## Prototypes & Inheritance

### The prototype chain

```bash
# // Every object has a prototype (except Object.prototype itself, whose proto is null).
# const o = {};
# Object.getPrototypeOf(o) === Object.prototype       // true
# Object.getPrototypeOf(Object.prototype) === null    // true
#
# // Lookups walk the chain:
# o.toString                                          // function — found on Object.prototype
# o.hasOwnProperty("toString")                        // false — inherited
# Object.hasOwn(o, "toString")                        // false — modern equivalent
#
# // Object.create — make an object with a specific prototype:
# const animal = { speak() { return `${this.name} makes a sound`; } };
# const dog = Object.create(animal);
# dog.name = "Rex";
# dog.speak()                                         // "Rex makes a sound"
#
# // Manipulating prototypes (slow — avoid in hot paths):
# Object.setPrototypeOf(o, anotherProto);
# o.__proto__                                         // legacy accessor — don't use
```

### Function constructors (pre-class)

```bash
# function Person(name) { this.name = name; }
# Person.prototype.greet = function () { return `Hi, I'm ${this.name}`; };
#
# const p = new Person("Ada");
# p.greet();                                          // "Hi, I'm Ada"
# p instanceof Person;                                // true
#
# // Inheritance the old way:
# function Employee(name, role) {
#   Person.call(this, name);                          // super-like
#   this.role = role;
# }
# Employee.prototype = Object.create(Person.prototype);
# Employee.prototype.constructor = Employee;
#
# // Modern code uses class — this is here so you can read older codebases.
```

## Classes

### Basics

```bash
# class Animal {
#   constructor(name) {
#     this.name = name;
#   }
#   speak() {
#     return `${this.name} makes a sound`;
#   }
# }
#
# const a = new Animal("Rex");
# a.speak();                              // "Rex makes a sound"
# a instanceof Animal;                    // true
# typeof Animal;                          // "function" — class is sugar over function constructor
```

### Class fields, private #fields, static

```bash
# class Counter {
#   n = 0;                                // public field (instance)
#   #secret = "shh";                      // private — runtime-private, # is part of name
#   static origin = { x: 0, y: 0 };       // static (class-level)
#   static #adminKey = "abc";             // static private
#
#   inc() { this.n++; return this.#secret; }
#
#   static create() { return new Counter(); }
# }
#
# const c = new Counter();
# c.n;                                    // 0
# c.inc();
# // c.#secret;                           // SyntaxError — # only inside the class
# Counter.origin;                         // { x: 0, y: 0 }
# Counter.create();
#
# // Underscore prefix is convention only — `_field` is NOT private. Use # for real privacy.
```

### Getters / setters

```bash
# class Temperature {
#   #celsius = 0;
#
#   get celsius() { return this.#celsius; }
#   set celsius(v) {
#     if (typeof v !== "number") throw new TypeError("number required");
#     this.#celsius = v;
#   }
#
#   get fahrenheit() { return this.#celsius * 9/5 + 32; }
#   set fahrenheit(f) { this.#celsius = (f - 32) * 5/9; }
# }
#
# const t = new Temperature();
# t.celsius = 100;
# t.fahrenheit;                           // 212
# t.fahrenheit = 32;
# t.celsius;                              // 0
```

### Inheritance, super, override

```bash
# class Animal {
#   constructor(name) { this.name = name; }
#   speak() { return `${this.name} makes a sound`; }
# }
#
# class Dog extends Animal {
#   constructor(name, breed) {
#     super(name);                        // MUST call super() before using `this`
#     this.breed = breed;
#   }
#   speak() {
#     return `${super.speak()} (woof)`;   // call parent method
#   }
# }
#
# const d = new Dog("Rex", "Husky");
# d.speak();                              // "Rex makes a sound (woof)"
# d instanceof Dog;                       // true
# d instanceof Animal;                    // true
```

## Modules

### ESM — import / export

```bash
# // math.js — named exports:
# export const PI = 3.14;
# export function add(a, b) { return a + b; }
# export class Vec2 { constructor(x, y) { this.x = x; this.y = y; } }
#
# // logger.js — default export:
# export default class Logger { log(m) { console.log(m); } }
#
# // mixed:
# // export default function main() {}
# // export const VERSION = "1.0";
#
# // consumer.js:
# import Logger from "./logger.js";                  // default
# import { PI, add } from "./math.js";               // named
# import { add as sum } from "./math.js";            // rename
# import * as math from "./math.js";                 // namespace
# import Logger, { VERSION } from "./module.js";     // default + named
#
# // Re-export:
# export { add } from "./math.js";
# export * from "./math.js";
# export { default } from "./logger.js";
#
# // Side-effect-only import:
# import "./register-globals.js";
```

### Dynamic import

```bash
# // import() returns a Promise — works at runtime, supports code splitting:
# async function loadHeavy() {
#   const mod = await import("./heavy.js");
#   mod.run();
# }
#
# // Conditional / lazy:
# if (needFeature) {
#   const { Feature } = await import("./feature.js");
#   new Feature();
# }
#
# // Dynamic specifier — the path can be a variable:
# const lang = "fr";
# const t = await import(`./locales/${lang}.js`);
```

### .mjs vs .cjs vs "type": "module"

```bash
# // package.json determines the default for .js files:
# // {
# //   "type": "module"     → .js is ESM (import/export)
# //   "type": "commonjs"   → .js is CJS (require/module.exports)
# //   omitted              → defaults to commonjs
# // }
#
# // Explicit overrides:
# // .mjs                     ALWAYS ESM
# // .cjs                     ALWAYS CommonJS
#
# // CommonJS — old style, still widespread:
# // const fs = require("fs");
# // module.exports = { add };
# // module.exports.add = (a, b) => a + b;
#
# // Interop notes:
# // - ESM can import CJS via default import: import pkg from "cjs-module"; // pkg = module.exports
# // - CJS cannot static-import ESM — must use dynamic await import("./esm.mjs")
# // - Top-level await works only in ESM
```

## Promises

### then / catch / finally

```bash
# // A Promise represents a future value with three states: pending, fulfilled, rejected.
#
# const p = new Promise((resolve, reject) => {
#   setTimeout(() => resolve(42), 100);
# });
#
# p.then(v => console.log(v))             // 42
#  .catch(e => console.error(e))
#  .finally(() => console.log("cleanup"));
#
# // .then can return a value, a thenable, or another promise — chains flatten:
# fetch("/users")
#   .then(r => r.json())                  // returns a promise; .then waits for it
#   .then(users => users.length)
#   .then(n => console.log(`got ${n}`))
#   .catch(err => console.error(err));    // catches any rejection in the chain
#
# // Throwing inside .then becomes a rejection:
# p.then(() => { throw new Error("nope"); })
#  .catch(e => console.error(e.message)); // "nope"
```

### Promise combinators

```bash
# // all — fail-fast: rejects on first rejection, resolves with array of values:
# const [a, b, c] = await Promise.all([fa(), fb(), fc()]);
#
# // allSettled — never rejects; per-result {status, value/reason}:
# const results = await Promise.allSettled([fa(), fb()]);
# for (const r of results) {
#   if (r.status === "fulfilled") console.log(r.value);
#   else                          console.error(r.reason);
# }
#
# // race — first SETTLED (fulfilled OR rejected) wins:
# const winner = await Promise.race([slow(), fast()]);
#
# // any — first FULFILLED wins; rejects only if ALL reject (with AggregateError):
# try {
#   const ok = await Promise.any([flakyA(), flakyB()]);
# } catch (e) {
#   // e is AggregateError with .errors = [reason, reason, ...]
# }
```

### Microtasks vs the message queue

```bash
# // Promise callbacks (then/catch/finally) run as MICROTASKS — they drain BEFORE
# // the next task (setTimeout, I/O, UI). This is the most important runtime detail.
#
# console.log("1");
# Promise.resolve().then(() => console.log("2"));
# setTimeout(() => console.log("3"), 0);
# console.log("4");
# // Output: 1, 4, 2, 3
# //   - "1", "4" run synchronously
# //   - "2" runs on the microtask queue (drains after current task)
# //   - "3" runs on the macrotask queue (next event loop tick)
```

## Async / Await

### The basics

```bash
# // async functions ALWAYS return a Promise. Inside, await pauses the function until
# // the awaited promise settles. Errors become rejections.
#
# async function load(url) {
#   const res = await fetch(url);
#   if (!res.ok) throw new Error(`HTTP ${res.status}`);
#   return res.json();
# }
#
# // Caller side:
# load("/api/users").then(users => render(users)).catch(showError);
# // Or:
# try {
#   const users = await load("/api/users");
#   render(users);
# } catch (err) {
#   showError(err);
# }
```

### Parallel vs sequential

```bash
# // SEQUENTIAL — each await blocks the next call:
# const a = await fetchA();
# const b = await fetchB();         // starts only after A resolves
#
# // PARALLEL — kick off both, then await:
# const pa = fetchA();              // returns a Promise immediately
# const pb = fetchB();              // also runs in parallel
# const a = await pa;
# const b = await pb;
#
# // Cleaner with Promise.all:
# const [a, b] = await Promise.all([fetchA(), fetchB()]);
#
# // Independent operations should ALWAYS use Promise.all unless ordering matters.
```

### Top-level await (ESM only)

```bash
# // In ES modules, you can await at the top level — no async wrapper needed:
# // config.js
# export const config = await fetch("/config.json").then(r => r.json());
#
# // Importing a module that uses TLA blocks the importer until the await resolves.
# // Use sparingly — TLA serializes module init.
```

### AbortController + signal

```bash
# // The standard way to cancel async work — fetch, streams, whatever accepts a signal.
# const ac = new AbortController();
#
# fetch("/big", { signal: ac.signal })
#   .then(r => r.text())
#   .then(t => console.log(t))
#   .catch(e => {
#     if (e.name === "AbortError") console.log("cancelled");
#   });
#
# setTimeout(() => ac.abort(), 1000);     // cancel after 1s
#
# // AbortSignal.timeout shortcut (Node 17.3+ / browsers):
# const r = await fetch("/api", { signal: AbortSignal.timeout(5000) });
#
# // Compose multiple signals:
# const merged = AbortSignal.any([ac.signal, AbortSignal.timeout(5000)]);
```

## Error Handling

### try / catch / finally

```bash
# try {
#   risky();
# } catch (err) {
#   console.error(err.message, err.stack);
# } finally {
#   cleanup();                              // runs whether or not an error was thrown
# }
#
# // Caught value is ALWAYS the thrown value — could be anything:
# try { throw "oops"; }                     // throwing a string is legal but bad
# catch (e) { console.error(e); }           // e is "oops" — no .message, no .stack
#
# // Convention: throw Error or Error subclasses. Discriminate in catch:
# try {
#   risky();
# } catch (e) {
#   if (e instanceof TypeError) handleType(e);
#   else if (e instanceof RangeError) handleRange(e);
#   else throw e;                           // re-throw unexpected
# }
```

### Custom Error subclasses

```bash
# class HttpError extends Error {
#   constructor(status, message, options) {
#     super(message, options);              // forward { cause } to parent
#     this.name = "HttpError";              // critical for error.name dispatch
#     this.status = status;
#   }
# }
#
# // ES2022 cause — the chain you've always wanted:
# try {
#   JSON.parse(badInput);
# } catch (e) {
#   throw new HttpError(400, "bad input", { cause: e });
# }
#
# // Walking the chain:
# try { /* ... */ }
# catch (e) {
#   for (let cur = e; cur; cur = cur.cause) {
#     console.error(cur.name, cur.message);
#   }
# }
```

### Throwing non-Errors — the trap

```bash
# // You CAN throw anything — but you'll regret it:
# throw "string";                           // no stack trace
# throw { code: 500 };                      // not instanceof Error
# throw 42;                                 // catch e is 42
#
# // Always:
# throw new Error("descriptive message");
#
# // Modern catch should defensively coerce:
# try { /* ... */ }
# catch (e) {
#   const err = e instanceof Error ? e : new Error(String(e));
#   logger.error(err);
# }
```

## Event Loop

### Call stack, task queue, microtask queue

```bash
# // 1. Run synchronous code on the call stack until empty.
# // 2. Drain the entire microtask queue (Promise callbacks, queueMicrotask).
# // 3. Run ONE task from the macrotask queue (setTimeout, I/O, UI events).
# // 4. Drain microtasks AGAIN.
# // 5. Repeat.
#
# // Implication: setTimeout(fn, 0) does NOT run before pending Promise.thens.
#
# console.log("A");
# setTimeout(() => console.log("B"), 0);
# Promise.resolve().then(() => console.log("C"));
# queueMicrotask(() => console.log("D"));
# console.log("E");
# // Output: A E C D B
# //   A, E synchronous
# //   C, D microtasks (in queue order)
# //   B macrotask (next tick)
```

### queueMicrotask

```bash
# // Schedule a microtask without creating a Promise:
# queueMicrotask(() => doWork());
#
# // Useful when you need "run after current sync code, before next task" without
# // the overhead of a Promise object.
```

### Why setTimeout(fn, 0) ≠ Promise.resolve().then(fn)

```bash
# // setTimeout(fn, 0)              — macrotask (next tick); minimum ~4ms in browsers
# // Promise.resolve().then(fn)     — microtask (this tick, after sync); zero delay
# // queueMicrotask(fn)             — same as Promise.resolve().then(fn), no Promise overhead
# // process.nextTick(fn)           — Node-only; runs BEFORE other microtasks (unique tier)
# // setImmediate(fn)               — Node-only; macrotask, runs after I/O
#
# // Rule: for "run after current code", prefer queueMicrotask. setTimeout(fn, 0) is
# // for explicitly yielding to the macrotask queue (e.g. allow rendering between
# // chunks of work in browsers).
```

## Date

### Date object — legacy and weird

```bash
# const now = new Date();
# Date.now()                                // unix ms — number
# now.toISOString()                         // "2026-04-25T10:30:00.000Z"
# now.getFullYear()                         // 2026
# now.getMonth()                            // 0-11 — JANUARY IS 0 (footgun)
# now.getDate()                             // 1-31 (day of month)
# now.getDay()                              // 0-6 (Sunday = 0)
# now.getTime()                             // unix ms — same as +now
#
# // Construction surprises:
# new Date(2026, 0, 1)                      // Jan 1 — months are 0-indexed
# new Date("2026-01-01")                    // UTC midnight (date-only string is UTC)
# new Date("2026-01-01T00:00:00")           // LOCAL midnight (no Z = local)
# new Date("2026-01-01T00:00:00Z")          // UTC midnight (Z = UTC)
#
# // Mutability:
# const d = new Date(2026, 0, 1);
# d.setMonth(d.getMonth() + 1);             // mutates d — surprising
# // Prefer immutable patterns or use a library.
```

### Intl — formatting and locale-aware

```bash
# new Intl.DateTimeFormat("en-GB", { dateStyle: "long" }).format(now);
# // "25 April 2026"
#
# new Intl.DateTimeFormat("en-US", {
#   year: "numeric", month: "short", day: "2-digit",
#   hour: "2-digit", minute: "2-digit", timeZone: "America/New_York",
# }).format(now);
# // "Apr 25, 2026, 06:30 AM"
#
# new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" }).format(1234.5);
# // "1.234,50 €"
#
# new Intl.RelativeTimeFormat("en", { numeric: "auto" }).format(-1, "day");
# // "yesterday"
#
# new Intl.ListFormat("en", { type: "conjunction" }).format(["a", "b", "c"]);
# // "a, b, and c"
```

### Temporal (TC39 stage 3)

```bash
# // The replacement for Date — finally a proper date/time API.
# // Polyfill until shipping: import "@js-temporal/polyfill"
#
# // Temporal.Now.plainDateISO()             // 2026-04-25 (no time, no zone)
# // Temporal.Now.zonedDateTimeISO()          // 2026-04-25T10:30:00Z
# // Temporal.PlainDate.from("2026-12-25")
# // const d = Temporal.PlainDate.from("2026-04-25");
# // d.add({ days: 7 })                       // 2026-05-02
# // d.until(Temporal.PlainDate.from("2026-12-25")).total("days")
#
# // For now: use date-fns or dayjs in production until Temporal ships native.
```

## Regex

### Literal vs constructor

```bash
# // Literal — compiled at parse time, fastest:
# const re = /\d+/g;
# const m = "abc123def".match(re);          // ["123"]
#
# // Constructor — for dynamic patterns:
# const pattern = "\\d+";                   // remember to escape backslashes!
# const re2 = new RegExp(pattern, "g");
#
# // String.raw avoids the double-escape mess:
# const re3 = new RegExp(String.raw`\d+`, "g");
```

### Flags

```bash
# // g  global       — match all (required for matchAll, replaceAll-with-regex)
# // i  insensitive  — case-insensitive
# // m  multiline    — ^/$ match line boundaries
# // s  dotAll       — . matches newline
# // u  unicode      — full Unicode mode
# // y  sticky       — match only at lastIndex (no scanning)
# // d  hasIndices   — match.indices populated with [start, end] pairs
# // v  unicodeSets  — extended Unicode (set algebra in char classes)
#
# // /abc/i.test("ABC") → true
# // /^a/m.test("x\na")  → true (with m)
```

### Named groups, matchAll, sticky

```bash
# // Named capture groups:
# const re = /(?<year>\d{4})-(?<month>\d{2})-(?<day>\d{2})/;
# const m = "2026-04-25".match(re);
# m.groups.year                             // "2026"
# m.groups.month                            // "04"
#
# // matchAll — iterator of all matches with groups:
# const text = "2026-04-25 and 2027-01-01";
# for (const hit of text.matchAll(/(?<y>\d{4})-(?<m>\d{2})-(?<d>\d{2})/g)) {
#   console.log(hit.groups.y);              // "2026", then "2027"
# }
#
# // Sticky (y) — match exactly at lastIndex:
# const sre = /\d+/y;
# sre.lastIndex = 3;
# sre.exec("abc123def");                    // matches "123" — but only because pos 3 starts there
#
# // Replace with named groups:
# "2026-04-25".replace(/(?<y>\d{4})-(?<m>\d{2})-(?<d>\d{2})/, "$<d>/$<m>/$<y>");
# // "25/04/2026"
```

## JSON

### parse / stringify

```bash
# JSON.parse('{"a": 1}')                    // { a: 1 }
# JSON.parse("[1, 2, 3]")                   // [1, 2, 3]
# JSON.parse('"hello"')                     // "hello"
#
# // Throws on invalid:
# try { JSON.parse("not json"); }
# catch (e) { console.error(e.message); }   // "Unexpected token o in JSON..."
#
# // CRITICAL: JSON.parse returns ANY shape — it lies about types. ALWAYS validate:
# const data = JSON.parse(input);
# // Don't trust data.user.name without checking it.
#
# // stringify — second arg is replacer, third is indent:
# JSON.stringify({ a: 1, b: 2 })            // '{"a":1,"b":2}'
# JSON.stringify({ a: 1, b: 2 }, null, 2)   // pretty with 2-space indent
# JSON.stringify({ a: 1, b: 2 }, null, "\t")  // tab indent
#
# // Replacer can filter keys:
# JSON.stringify({ a: 1, secret: "x" }, ["a"]);   // '{"a":1}' — array allowlist
# JSON.stringify({ a: 1, secret: "x" }, (k, v) => k === "secret" ? undefined : v);
# // '{"a":1}' — function form
#
# // Reviver (parse) — transform values:
# JSON.parse('{"d":"2026-04-25"}', (k, v) => k === "d" ? new Date(v) : v);
```

### Gotchas

```bash
# // BigInt — TypeError without custom replacer
# // JSON.stringify({ n: 1n })               // throws
# JSON.stringify({ n: 1n }, (k, v) => typeof v === "bigint" ? v.toString() : v);
#
# // undefined / functions / symbols are dropped:
# JSON.stringify({ a: undefined, b: () => 1, c: Symbol("x") });
# // "{}"
#
# // NaN / Infinity become null:
# JSON.stringify({ n: NaN, i: Infinity })   // '{"n":null,"i":null}'
#
# // Cyclic references throw:
# const o = {}; o.self = o;
# // JSON.stringify(o)                       // TypeError: cyclic
# // Use structuredClone for cycles, or a custom replacer that tracks seen.
#
# // Maps/Sets do NOT serialize as you'd expect:
# JSON.stringify(new Map([["a", 1]]))       // "{}"
# // Convert: JSON.stringify(Object.fromEntries(map));
```

## File I/O — Node fs/promises

```bash
# // Modern Node async file I/O — promise-based:
# import { readFile, writeFile, appendFile, mkdir, readdir, stat, rm, rename } from "node:fs/promises";
# import { createReadStream, createWriteStream } from "node:fs";
#
# // Read full file:
# const text = await readFile("data.txt", "utf8");
# const buffer = await readFile("image.png");           // no encoding → Buffer
#
# // Write:
# await writeFile("out.txt", "hello\n");
# await writeFile("data.bin", new Uint8Array([1, 2, 3]));
#
# // Append:
# await appendFile("log.txt", "new line\n");
#
# // Make dirs (recursive):
# await mkdir("a/b/c", { recursive: true });
#
# // List directory:
# const entries = await readdir("./", { withFileTypes: true });
# for (const e of entries) {
#   if (e.isFile()) console.log("file:", e.name);
#   if (e.isDirectory()) console.log("dir:", e.name);
# }
#
# // Stat:
# const s = await stat("data.txt");
# s.size; s.mtime; s.isFile(); s.isDirectory();
#
# // Streams (large files):
# const rs = createReadStream("big.txt", { encoding: "utf8" });
# for await (const chunk of rs) process(chunk);
#
# // Watch (Node 19+):
# import { watch } from "node:fs/promises";
# const watcher = watch("./src", { recursive: true });
# for await (const ev of watcher) {
#   console.log(ev.eventType, ev.filename);
# }
#
# // Remove:
# await rm("dir", { recursive: true, force: true });    // rm -rf
```

## File I/O — Deno & Bun

### Deno

```bash
# // Deno uses Web-standards-aligned APIs and explicit permissions.
# // Run with: deno run --allow-read --allow-write script.js
#
# const text = await Deno.readTextFile("data.txt");
# await Deno.writeTextFile("out.txt", text);
#
# // Bytes:
# const bytes = await Deno.readFile("data.bin");        // Uint8Array
# await Deno.writeFile("out.bin", bytes);
#
# // Streaming:
# const file = await Deno.open("big.txt");
# for await (const chunk of file.readable) process(chunk);
# file.close();
#
# // Stat / list:
# const info = await Deno.stat("data.txt");
# for await (const e of Deno.readDir("./")) console.log(e.name);
#
# // Watch:
# const watcher = Deno.watchFs("./src");
# for await (const ev of watcher) console.log(ev.kind, ev.paths);
```

### Bun

```bash
# // Bun.file — lazy file handle with a Web-aligned API:
# const file = Bun.file("data.txt");
# const text = await file.text();
# const json = await file.json();
# const bytes = await file.arrayBuffer();
#
# // Bun.write — accepts strings, buffers, Blobs, Responses, files:
# await Bun.write("out.txt", "hello");
# await Bun.write("copy.png", Bun.file("source.png"));
# await Bun.write("api.json", await fetch("https://api.example.com/data"));
#
# // Stream:
# const stream = Bun.file("big.txt").stream();
# for await (const chunk of stream) process(chunk);
#
# // Glob:
# const glob = new Bun.Glob("**/*.js");
# for (const path of glob.scanSync(".")) console.log(path);
```

## Browser File API

```bash
# // <input type="file"> gives you File objects (extends Blob):
# const input = document.querySelector("input[type=file]");
# input.addEventListener("change", async (ev) => {
#   const file = ev.target.files[0];
#   console.log(file.name, file.size, file.type);
#
#   // Read as text:
#   const text = await file.text();
#   // Read as bytes:
#   const buf = await file.arrayBuffer();
#   // Read as data URL (e.g., for img.src):
#   const url = URL.createObjectURL(file);
#   img.src = url;
#   // Remember to URL.revokeObjectURL(url) when done.
# });
#
# // Older callback-style (FileReader) — only when text/arrayBuffer aren't enough:
# const reader = new FileReader();
# reader.onload = () => console.log(reader.result);
# reader.readAsDataURL(file);
#
# // fetch + Response.text/json — the modern Web pattern:
# const res = await fetch("/api/data.json");
# const data = await res.json();
# const txt = await res.text();
# const blob = await res.blob();
# const ab = await res.arrayBuffer();
```

## HTTP fetch

### GET / POST / abort

```bash
# // fetch is global in Node 18+, Deno, Bun, all modern browsers.
#
# // GET:
# const res = await fetch("https://api.example.com/users/1");
# if (!res.ok) throw new Error(`HTTP ${res.status} ${res.statusText}`);
# const user = await res.json();
#
# // POST JSON:
# const r = await fetch("/api/users", {
#   method: "POST",
#   headers: { "content-type": "application/json", authorization: `Bearer ${token}` },
#   body: JSON.stringify({ name: "Ada" }),
# });
#
# // POST FormData (multipart):
# const fd = new FormData();
# fd.append("file", fileInput.files[0]);
# fd.append("name", "Ada");
# await fetch("/upload", { method: "POST", body: fd });
# // Don't set content-type — fetch sets multipart boundary automatically.
#
# // POST URL-encoded:
# await fetch("/login", {
#   method: "POST",
#   headers: { "content-type": "application/x-www-form-urlencoded" },
#   body: new URLSearchParams({ user: "ada", pass: "x" }),
# });
#
# // Abort + timeout:
# const r2 = await fetch(url, { signal: AbortSignal.timeout(5000) });
```

### Streaming responses

```bash
# // Stream a large response without buffering it all:
# const res = await fetch("/big-file.bin");
# for await (const chunk of res.body) {        // Web ReadableStream is async-iterable
#   process(chunk);                            // chunk is Uint8Array
# }
#
# // Or use the reader directly:
# const reader = res.body.getReader();
# while (true) {
#   const { done, value } = await reader.read();
#   if (done) break;
#   process(value);
# }
```

### Request / Response classes

```bash
# // You can construct Request / Response without making a network call —
# // useful for testing, service workers, mocking:
# const req = new Request("/api/x", {
#   method: "POST",
#   headers: { "x-trace": "abc" },
#   body: JSON.stringify({ a: 1 }),
# });
# fetch(req);
#
# const fakeRes = new Response(JSON.stringify({ ok: true }), {
#   status: 200,
#   headers: { "content-type": "application/json" },
# });
# await fakeRes.json();                        // { ok: true }
```

## Subprocess (Node child_process)

```bash
# // exec — spawns a shell, buffers all output. Easy but DANGEROUS with user input.
# import { exec, execFile, spawn, fork } from "node:child_process";
# import { promisify } from "node:util";
# const execAsync = promisify(exec);
#
# const { stdout, stderr } = await execAsync("ls -la");
# console.log(stdout);
#
# // execFile — no shell; safer for variable args:
# const execFileAsync = promisify(execFile);
# const { stdout: out } = await execFileAsync("git", ["status", "--short"]);
#
# // spawn — streaming I/O for long-running processes:
# const child = spawn("ffmpeg", ["-i", "in.mp4", "out.webm"]);
# child.stdout.on("data", chunk => console.log("out:", chunk.toString()));
# child.stderr.on("data", chunk => console.error("err:", chunk.toString()));
# child.on("close", code => console.log("exit", code));
#
# // fork — spawn a Node child with IPC channel:
# const worker = fork("./worker.js");
# worker.send({ cmd: "start" });
# worker.on("message", msg => console.log("from worker:", msg));
#
# // SECURITY: NEVER pass unvalidated user input to exec — shell injection.
# // Use execFile (no shell) and explicit argv arrays.
# // BAD:  exec(`ls ${userPath}`)
# // GOOD: execFile("ls", [userPath])
```

### Deno / Bun subprocess

```bash
# // Deno.Command (replaces Deno.run):
# const cmd = new Deno.Command("ls", { args: ["-la"], stdout: "piped" });
# const { stdout, code } = await cmd.output();
# console.log(new TextDecoder().decode(stdout));
#
# // Bun.spawn:
# const proc = Bun.spawn(["git", "status"], { stdout: "pipe" });
# const out = await new Response(proc.stdout).text();
# await proc.exited;
```

## Stdio / Args / Env

### Node

```bash
# process.argv                              // ["node", "/path/script.js", ...args]
# const args = process.argv.slice(2);       // user args only
#
# process.env.NODE_ENV                      // string | undefined
# process.env.DEBUG = "true";               // mutable
#
# process.cwd()                             // current working directory
# process.platform                          // "darwin" | "linux" | "win32"
# process.version                           // "v22.x.x"
# process.exit(1)                           // exit with code
#
# // stdin — readline for line-by-line:
# import readline from "node:readline";
# const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
# rl.on("line", line => console.log(`got: ${line}`));
# rl.on("close", () => console.log("done"));
#
# // Async iteration over stdin:
# for await (const line of readline.createInterface({ input: process.stdin })) {
#   console.log(line);
# }
#
# // Parse args (Node 18.3+):
# import { parseArgs } from "node:util";
# const { values, positionals } = parseArgs({
#   args: process.argv.slice(2),
#   options: {
#     verbose: { type: "boolean", short: "v" },
#     port:    { type: "string",  short: "p", default: "3000" },
#   },
#   allowPositionals: true,
# });
```

### Deno / Bun

```bash
# // Deno:
# Deno.args                                 // ["foo", "bar"]
# Deno.env.get("HOME")                      // string | undefined
# Deno.env.set("X", "y");
# Deno.cwd()
# Deno.exit(0);
#
# // Bun:
# Bun.argv                                  // same as process.argv
# Bun.env.HOME                              // string | undefined
# process.exit(0);                          // process is also available in Bun
```

## Streams

### Node streams (Readable, Writable, Transform, Duplex)

```bash
# import { Readable, Writable, Transform, pipeline } from "node:stream";
# import { pipeline as pipe } from "node:stream/promises";
# import { createReadStream, createWriteStream } from "node:fs";
# import { createGzip } from "node:zlib";
#
# // Readable from generator (modern):
# const numbers = Readable.from((async function* () {
#   for (let i = 0; i < 1_000_000; i++) yield `${i}\n`;
# })());
#
# // Pipeline (promises) — handles errors and cleanup:
# await pipe(
#   createReadStream("input.txt"),
#   createGzip(),
#   createWriteStream("output.gz"),
# );
#
# // Async iteration over readable:
# for await (const chunk of createReadStream("data.txt", { encoding: "utf8" })) {
#   process(chunk);
# }
#
# // Custom Transform:
# const upper = new Transform({
#   transform(chunk, _enc, cb) { cb(null, chunk.toString().toUpperCase()); },
# });
# await pipe(createReadStream("in.txt"), upper, createWriteStream("out.txt"));
```

### Web Streams (cross-runtime)

```bash
# // ReadableStream / WritableStream / TransformStream — work in browsers, Node 18+, Deno, Bun.
# const rs = new ReadableStream({
#   start(controller) {
#     controller.enqueue("hello\n");
#     controller.enqueue("world\n");
#     controller.close();
#   },
# });
#
# for await (const chunk of rs) console.log(chunk);
#
# // TransformStream:
# const upper = new TransformStream({
#   transform(chunk, controller) { controller.enqueue(chunk.toUpperCase()); },
# });
#
# const out = rs.pipeThrough(upper);
# for await (const chunk of out) console.log(chunk);
```

## Symbols & Well-Known Symbols

```bash
# // Symbols are unique, non-string, non-enumerable identifiers — perfect for hidden keys.
# const id = Symbol("description");          // description is for debugging only
# const id2 = Symbol("description");         // id !== id2 — every Symbol() call is unique
# const shared = Symbol.for("global-key");   // shared registry
# const same = Symbol.for("global-key");
# shared === same;                           // true
#
# // Well-known symbols — hooks for protocols:
# // Symbol.iterator        — make objects iterable (for-of, spread)
# // Symbol.asyncIterator   — make objects async-iterable (for-await-of)
# // Symbol.hasInstance     — customize instanceof
# // Symbol.toPrimitive     — customize coercion to primitive
# // Symbol.dispose         — using statement (TS 5.2+/ES proposal)
# // Symbol.asyncDispose    — await using statement
#
# class Counter {
#   constructor(n) { this.n = n; }
#   *[Symbol.iterator]() {
#     for (let i = 0; i < this.n; i++) yield i;
#   }
# }
# [...new Counter(3)]                        // [0, 1, 2]
#
# // Symbol.toPrimitive — controls type coercion:
# const t = {
#   [Symbol.toPrimitive](hint) {
#     if (hint === "number") return 42;
#     if (hint === "string") return "forty-two";
#     return "default";
#   },
# };
# +t                                         // 42
# `${t}`                                     // "forty-two"
# t + ""                                     // "default"
```

## Standard Built-ins

### Math

```bash
# Math.PI                                    // 3.141592653589793
# Math.E                                     // 2.718281828459045
# Math.LN2, Math.LN10, Math.LOG2E
#
# Math.abs(-3)                               // 3
# Math.min(1, 2, 3)                          // 1
# Math.max(...arr)                           // spread for arrays
# Math.pow(2, 10)                            // 1024
# 2 ** 10                                    // 1024 — modern operator
# Math.sqrt(16)                              // 4
# Math.cbrt(27)                              // 3
# Math.log(Math.E)                           // 1 (natural log)
# Math.log2(8)                               // 3
# Math.log10(1000)                           // 3
# Math.sin / cos / tan / atan2(y, x)
# Math.random()                              // [0, 1) — NOT cryptographically secure
# // For crypto random, use crypto.getRandomValues(new Uint32Array(1))[0]
#
# Math.hypot(3, 4)                           // 5 — sqrt(a^2 + b^2 + ...)
# Math.clz32(1)                              // 31 — count leading zeros (32-bit)
# Math.imul(a, b)                            // 32-bit integer multiplication
```

### Number

```bash
# Number.isInteger(42)                       // true
# Number.isInteger(42.0)                     // true (fractional part is 0)
# Number.isInteger(42.5)                     // false
# Number.isFinite(Infinity)                  // false
# Number.isFinite(42)                        // true
# Number.isNaN(NaN)                          // true (the only correct isNaN)
# Number.isSafeInteger(2 ** 53)              // false — beyond MAX_SAFE_INTEGER
# Number.parseInt("10", 2)                   // 2
# Number.parseFloat("3.14")                  // 3.14
#
# (3.14159).toFixed(2)                       // "3.14"        — string!
# (1234567).toLocaleString("en-US")          // "1,234,567"
# (255).toString(16)                         // "ff"          — radix
# (10).toString(2)                           // "1010"
```

### String, Array, Date, RegExp, Map, Set

```bash
# // String static:
# String.raw`\n is a backslash-n`            // "\n is a backslash-n"
# String.fromCharCode(72, 73)                // "HI"
# String.fromCodePoint(0x1F600)              // "😀"
#
# // Array static:
# Array.from("abc")                          // ["a","b","c"]
# Array.from({ length: 3 }, (_, i) => i)     // [0,1,2]
# Array.of(7)                                // [7] (vs Array(7) which is sparse)
# Array.isArray(x)                           // safer than instanceof Array (across realms)
#
# // Date static:
# Date.now()                                 // unix ms — fastest "now in ms"
# Date.parse("2026-04-25T00:00:00Z")         // unix ms
# Date.UTC(2026, 3, 25)                      // unix ms for Apr 25 2026 UTC
#
# // Object.* — already covered.
# // RegExp — covered in Regex section.
# // Map / Set — covered earlier.
```

## Reflection

### Reflect API

```bash
# // Reflect mirrors language operations as functions — useful for proxies and meta-programming.
# Reflect.has(obj, "key")                    // same as "key" in obj
# Reflect.get(obj, "key")                    // same as obj.key
# Reflect.set(obj, "key", val)               // same as obj.key = val
# Reflect.ownKeys(obj)                       // all own keys (including symbols, non-enumerable)
# Reflect.deleteProperty(obj, "key")         // same as delete obj.key
# Reflect.defineProperty(obj, "k", { value: 1, writable: false });
# Reflect.getPrototypeOf(obj)
# Reflect.setPrototypeOf(obj, proto)
# Reflect.construct(Date, [2026, 0, 1])      // new Date(2026, 0, 1)
# Reflect.apply(fn, this, [args])            // fn.apply(this, args)
```

### Proxy

```bash
# // A Proxy wraps an object and intercepts ALL operations via a handler:
# const target = { a: 1 };
# const proxy = new Proxy(target, {
#   get(t, k) { console.log("get", k); return t[k]; },
#   set(t, k, v) { console.log("set", k, v); t[k] = v; return true; },
#   has(t, k) { return k in t; },
#   deleteProperty(t, k) { delete t[k]; return true; },
# });
#
# proxy.a;                                   // logs "get a", returns 1
# proxy.b = 2;                               // logs "set b 2"
#
# // Common uses: validation, logging, virtualization, default values:
# const withDefaults = (obj, defaults) =>
#   new Proxy(obj, {
#     get(t, k) { return k in t ? t[k] : defaults[k]; },
#   });
#
# // Performance note: proxies are slower than direct access. Don't wrap hot paths.
```

### Property descriptors

```bash
# // Every property has a descriptor with: value, writable, enumerable, configurable.
# const o = {};
# Object.defineProperty(o, "x", {
#   value: 1,
#   writable: false,                         // assignment fails silently (or throws in strict)
#   enumerable: false,                       // hidden from for-in, Object.keys
#   configurable: false,                     // can't delete or redefine
# });
#
# Object.getOwnPropertyDescriptor(o, "x");
# // { value: 1, writable: false, enumerable: false, configurable: false }
#
# Object.getOwnPropertyDescriptors(o);       // all of them
# Object.defineProperties(o, { x: {...}, y: {...} });
#
# // Accessor descriptor — getter/setter:
# Object.defineProperty(o, "y", {
#   get() { return this._y; },
#   set(v) { this._y = v; },
# });
```

## Memory & GC

```bash
# // JavaScript is fully garbage collected. You CANNOT free memory manually.
# // The only handles you have are:
# //   - Drop references and let GC reclaim later (no guarantees on when).
# //   - Use WeakMap/WeakSet for "metadata that doesn't keep target alive".
# //   - WeakRef + FinalizationRegistry — RARE and DANGEROUS, see below.
#
# // WeakRef — hold a reference that doesn't prevent GC:
# const wr = new WeakRef(someObj);
# const obj = wr.deref();                    // obj or undefined (if collected)
# // Use sparingly: tickling GC behavior is non-deterministic across engines.
#
# // FinalizationRegistry — schedule cleanup AFTER the target is collected:
# const reg = new FinalizationRegistry(token => {
#   console.log(`cleanup ${token}`);
# });
# reg.register(someObj, "obj-id-123");
# // When someObj is GC'd, the callback runs eventually (no timing guarantee).
#
# // Reasons to AVOID WeakRef/FinalizationRegistry:
# // - Behavior depends on GC implementation (v8/SpiderMonkey/JSC differ).
# // - Callbacks may NEVER fire (e.g., on process exit).
# // - Encourages thinking about lifetime that JS doesn't actually expose.
# // Use only for: caches with weak values, polyfilling FinalizationRegistry use cases.
```

## Build / Run

### Run

```bash
# # Run plain JS — pick a runtime:
# node script.js                            # Node
# node --watch script.js                    # auto-restart on file change (Node 18.11+)
# node --env-file=.env script.js            # load .env (Node 20.6+)
# deno run --allow-net script.js
# bun run script.js
# bun --hot script.js                       # hot reload
#
# # Run as ESM specifically (when extension is ambiguous):
# node --input-type=module -e 'await import("./x.js")'
```

### Bundlers / transpilers

```bash
# # esbuild — Go, blazing fast:
# npx esbuild src/index.js --bundle --outfile=dist/bundle.js --target=es2022
#
# # swc — Rust, fast transpile, used by Next.js / Deno / Vitest:
# npx swc src -d dist
#
# # Vite — esbuild dev + rollup prod; the modern dev choice:
# npm create vite@latest my-app
# cd my-app && npm install && npm run dev
#
# # webpack — older, plugin-rich, slower:
# npx webpack --mode production
#
# # rollup — library-focused, tree-shaking pioneer:
# npx rollup -c
#
# # Bun build:
# bun build ./src/index.js --outdir ./dist
#
# # Deno bundle is deprecated — use deno_emit or esbuild instead.
# # Deno can compile a self-contained binary:
# deno compile --allow-net script.js
```

## Test

### vitest — recommended

```bash
# // npm install -D vitest
# // package.json: "scripts": { "test": "vitest" }
#
# // src/sum.test.js
# import { describe, it, expect } from "vitest";
# import { sum } from "./sum.js";
#
# describe("sum", () => {
#   it("adds two numbers", () => {
#     expect(sum(1, 2)).toBe(3);
#   });
#   it("rejects strings", () => {
#     expect(() => sum("1", 2)).toThrow();
#   });
# });
#
# // Run: npx vitest          (watch mode)
# //      npx vitest run      (single run, for CI)
# //      npx vitest --coverage
```

### node:test (built-in)

```bash
# // No dependencies. Built into Node 18+.
# // src/sum.test.js
# import { describe, it } from "node:test";
# import assert from "node:assert/strict";
# import { sum } from "./sum.js";
#
# describe("sum", () => {
#   it("adds", () => assert.equal(sum(1, 2), 3));
# });
#
# // Run: node --test
# //      node --test --watch
# //      node --test 'src/**/*.test.js'
```

### jest, deno test, bun test

```bash
# # jest — most popular legacy framework:
# npx jest
#
# # Deno test — built into Deno:
# deno test                                 # discovers *.test.js / *_test.js
# deno test --allow-read
#
# // src/sum_test.js
# import { assertEquals } from "jsr:@std/assert";
# Deno.test("sum adds", () => assertEquals(1 + 2, 3));
#
# # Bun test — Jest-compatible API, very fast:
# bun test
#
# // src/sum.test.js
# import { test, expect } from "bun:test";
# test("sum", () => expect(1 + 2).toBe(3));
```

## Lint / Format

```bash
# # ESLint — most popular linter:
# npm install -D eslint
# npx eslint --init                         # interactive setup
# npx eslint .                              # check
# npx eslint . --fix                        # auto-fix
#
# # Flat config (eslint.config.js, ESLint 9+):
# // import js from "@eslint/js";
# // export default [
# //   js.configs.recommended,
# //   { rules: { "no-unused-vars": "warn" } },
# // ];
#
# # Prettier — opinionated formatter:
# npm install -D prettier
# npx prettier --write .
# npx prettier --check .                    # CI mode
#
# # Biome — Rust-based all-in-one (lint + format), instant:
# npm install -D @biomejs/biome
# npx biome init
# npx biome check .                         # lint + format check
# npx biome check --apply .                 # auto-fix
#
# # Pick ONE formatter (Prettier OR Biome). Lint with ESLint OR Biome.
```

## Common Gotchas

### == vs ===

```bash
# 0 == "0"                                   // true   — string coerced to number
# 0 == false                                 // true   — both coerce to 0
# null == undefined                          // true   — only "useful" coercion
# null == 0                                  // false  — special case
# "" == false                                // true
# [] == false                                // true   — [] coerces to ""
# [] == ![]                                  // true   — never write this
#
# 0 === "0"                                  // false  — strict
# null === undefined                         // false
#
# // Convention: use === everywhere. The ONE exception is x == null which catches
# // both null and undefined cleanly.
```

### Falsy soup

```bash
# // Falsy values: false, 0, -0, 0n, "", null, undefined, NaN
# // Everything else is truthy — including [], {}, "0", "false", new Boolean(false).
#
# // The classic bug — || cannot distinguish "0" or "" from missing:
# function port(p) { return p || 3000; }
# port(0);                                   // 3000  — WRONG, 0 is a valid port
#
# // Use ?? for null/undefined ONLY:
# function port(p) { return p ?? 3000; }
# port(0);                                   // 0     — correct
# port(undefined);                           // 3000
```

### `this` in callbacks

```bash
# class Worker {
#   tasks = [];
#   start() {
#     this.tasks.forEach(function (t) {
#       this.run(t);                         // ERROR — `this` is undefined inside non-arrow callback
#     });
#   }
# }
#
# // Fix 1 — arrow function:
# this.tasks.forEach(t => this.run(t));
#
# // Fix 2 — bind:
# this.tasks.forEach(function (t) { this.run(t); }.bind(this));
#
# // Fix 3 — second arg of forEach is thisArg:
# this.tasks.forEach(function (t) { this.run(t); }, this);
```

### Hoisting bites

```bash
# // var declarations hoist; this is rarely what you want:
# function f() {
#   if (cond) { var x = 1; }
#   return x;                                // x is undefined when cond is false (var leaks out)
# }
#
# // let/const fix it (block-scoped):
# function g() {
#   if (cond) { const x = 1; return x; }
#   return undefined;
# }
#
# // Function declarations hoist — sometimes useful, sometimes confusing:
# greet();                                   // works
# function greet() { console.log("hi"); }
```

### NaN reflexivity

```bash
# // NaN is the ONLY value not equal to itself:
# NaN === NaN                                // false
# NaN == NaN                                 // false
# [NaN].includes(NaN)                        // true   — uses SameValueZero
# [NaN].indexOf(NaN)                         // -1     — uses ===
#
# // Detection — always Number.isNaN:
# Number.isNaN(NaN)                          // true
# Number.isNaN("hi")                         // false  — global isNaN coerces (BAD)
```

### .length on strings = UTF-16 units

```bash
# "😀".length                                 // 2     — surrogate pair
# "😀".charAt(0)                              // "\uD83D" — half a character
# [..."😀"].length                            // 1     — code points
# Array.from("😀").length                     // 1
#
# // For grapheme clusters (skin tones, flags), use Intl.Segmenter:
# const seg = new Intl.Segmenter();
# [...seg.segment("👨‍👩‍👧")].length              // 1 — one grapheme
```

### Async errors swallowed

```bash
# // Without await, a rejected promise can become an unhandled rejection:
# async function f() { throw new Error("nope"); }
# f();                                       // not awaited — unhandled rejection (depending on runtime)
# await f();                                 // throws normally
# f().catch(e => console.error(e));          // handled
#
# // forEach cannot await — see "Loops" section.
#
# // Promise constructor swallows synchronous throws into rejection:
# new Promise((res, rej) => { throw new Error("x"); });   // becomes rejection
# // Don't put async work inside Promise constructor — use async function.
```

### Automatic semicolon insertion (ASI)

```bash
# // ASI inserts semicolons in some places. The classic bite:
# function bad() {
#   return                                   // ASI inserts ; here!
#     { ok: true };                          // unreachable — function returns undefined
# }
#
# function good() {
#   return {                                 // brace on same line
#     ok: true,
#   };
# }
#
# // Other bites — lines starting with ( [ ` / + - need a leading ; after a previous statement:
# const a = 1
# (foo)()                                    // parsed as a(1)(foo)()  — surprise call
# // Fix: const a = 1;  or write   ;(foo)()
#
# // Easiest answer: always write semicolons (Prettier does this for you).
```

### for-in over arrays

```bash
# const a = [10, 20, 30];
# a.foo = "x";                               // arrays are objects
#
# for (const k in a) console.log(k);         // "0", "1", "2", "foo"  — strings!
# for (const v of a) console.log(v);         // 10, 20, 30            — values
# a.forEach((v, i) => console.log(i, v));    // proper iteration
#
# // Rule: never for-in arrays. Use for-of, forEach, or indexed for.
```

## Performance Tips

```bash
# # 1. Avoid for-in on arrays — use for-of or indexed for. Allocates strings, slow.
#
# # 2. Use indexed for for hot loops:
# for (let i = 0, n = arr.length; i < n; i++) { /* ... */ }
#
# # 3. Cache .length when iterating:
# for (let i = 0; i < arr.length; i++) {}    # OK — engines hoist for stable arrays
# for (let i = 0, n = arr.length; i < n; i++) {}  # safer if arr might change
#
# # 4. Use TypedArrays for binary data:
# const buf = new ArrayBuffer(1024);
# const view = new Uint8Array(buf);
# view[0] = 0xFF;
# # TypedArrays are contiguous, no boxing, much faster than Array<number>.
#
# # 5. Avoid synchronous fs in servers — blocks the event loop:
# # BAD:  fs.readFileSync("data.txt")  in a request handler
# # GOOD: await fs.promises.readFile("data.txt")  with async handler
#
# # 6. CPU-bound work? Use a worker:
# # Browser: new Worker("worker.js"), postMessage / onmessage
# # Node:    new Worker(...) from "node:worker_threads"
# # Bun:     Bun has identical Worker API
# # Deno:    new Worker("./w.js", { type: "module" })
#
# # 7. Avoid creating closures in hot paths if you can help it — each call allocates.
#
# # 8. Object shape stability matters — V8/SpiderMonkey de-optimize when shape changes.
# #    Add ALL fields in the constructor; don't add new properties later.
#
# # 9. Don't trigger deopts: avoid `delete obj.x` (use null), avoid `arguments`,
# #    avoid try/catch in tight loops (older engines deopt; less true now).
#
# # 10. Profile, don't guess: --inspect (chrome devtools), 0x flamegraphs,
# #     `node --cpu-prof`, deno test --bench, Bun.bench.
```

## Modern Idioms

### Optional chaining `?.`

```bash
# const city = user?.address?.city;          // undefined if any link is null/undefined
# const first = arr?.[0];                    // safe indexed access
# user?.greet?.();                            // safe method call
# // Short-circuits — RHS not evaluated if LHS is nullish.
```

### Nullish coalescing `??`

```bash
# const port = config.port ?? 3000;          // ONLY null/undefined trigger fallback
# // vs ||
# const x = "" || "default";                 // "default"  — empty string is falsy
# const y = "" ?? "default";                 // ""         — empty string is not nullish
```

### Logical assignment

```bash
# obj.x ??= "default";                       // assign if obj.x is null/undefined
# obj.y ||= "default";                       // assign if obj.y is falsy
# obj.z &&= "filled";                        // assign if obj.z is truthy
#
# // Equivalent expansions:
# // a ??= b;   →  a ?? (a = b)
# // a ||= b;   →  a || (a = b)
# // a &&= b;   →  a && (a = b)
```

### structuredClone for deep copy

```bash
# const deep = structuredClone({
#   d: new Date(),
#   m: new Map([["k", 1]]),
#   s: new Set([1, 2]),
#   buf: new Uint8Array([1, 2, 3]),
# });
# // Handles dates, maps, sets, regex, typed arrays, ArrayBuffer, cyclic refs.
# // Does NOT clone functions, class instances, DOM nodes (DataCloneError).
#
# // Why this beats JSON.parse(JSON.stringify(x)):
# //   - Preserves Date (JSON returns ISO string)
# //   - Preserves Map/Set (JSON drops them)
# //   - Handles cycles (JSON throws)
# //   - Faster for non-trivial trees
```

### Top-level await + AbortSignal.timeout

```bash
# // top-level await (ESM modules only):
# // config.js
# export const cfg = await fetch("/config.json").then(r => r.json());
#
# // AbortSignal.timeout — fluent timeout for fetch/streams (Node 17.3+, all browsers):
# const r = await fetch(url, { signal: AbortSignal.timeout(5000) });
#
# // Compose multiple aborts:
# const merged = AbortSignal.any([userAbort.signal, AbortSignal.timeout(5000)]);
# fetch(url, { signal: merged });
```

### Object.groupBy / Map.groupBy (ES2024)

```bash
# // Group an array by a key function:
# const orders = [{ type: "food", n: 1 }, { type: "drink", n: 2 }, { type: "food", n: 3 }];
# Object.groupBy(orders, o => o.type);
# // { food: [{...}, {...}], drink: [{...}] }
#
# Map.groupBy(orders, o => o.type);
# // Map(2) { "food" → [...], "drink" → [...] }
```

### Array.fromAsync (ES2024)

```bash
# // Build an array from an async iterable, awaiting each item:
# async function* nums() { for (let i = 0; i < 3; i++) yield i; }
# const arr = await Array.fromAsync(nums());           // [0, 1, 2]
# const fromPromises = await Array.fromAsync([fetchA(), fetchB(), fetchC()]);
# // Like Promise.all + Array.from in one.
```

### using / await using (TC39 stage 3 / Node 22+ behind flag)

```bash
# // Explicit resource management — auto-dispose at scope end:
# class FileHandle {
#   [Symbol.dispose]() { this.close(); }
#   close() { /* ... */ }
# }
# {
#   using f = openFile("data.txt");
#   // ... use f
# }                                          // f[Symbol.dispose]() runs here automatically
#
# // Async variant:
# {
#   await using db = await openDB();
# }                                          // db[Symbol.asyncDispose]() runs and is awaited
```

## Tips

- Always use `const` by default; switch to `let` only when reassignment is necessary. Never `var`.
- Use `===` everywhere except `x == null` (the one useful coercion — catches both null and undefined).
- Arrow functions inherit `this` from the enclosing scope; regular functions do not. Pick deliberately.
- For optional values, prefer `?.` and `??` over manual null checks and `||` (which conflates falsy with nullish).
- Use `structuredClone` for deep copies — never `JSON.parse(JSON.stringify(x))` (loses Date, Map, Set, cycles).
- Prefer `Number.isNaN` and `Number.isFinite` over the global versions, which lie via coercion.
- Use `for-of` for arrays, `for-of Object.entries(o)` for objects, and never `for-in` on arrays.
- `Promise.allSettled` when you want every result regardless of failures; `Promise.all` when first failure should abort.
- Always pass `signal: AbortSignal.timeout(ms)` to network fetches in production code — no naked unbounded waits.
- Throw `Error` (or subclasses with `cause`); never throw strings or plain objects — you lose the stack and structure.
- Type your boundaries — wrap `JSON.parse` in a guard or use Zod/Valibot/ArkType for runtime shape validation.
- Use private class fields `#field` for true privacy. Underscore prefixes are convention only.
- Prefer named exports over default exports for refactor-safety and clean tree-shaking.
- `forEach` cannot await — use `for-of` for sequential async, `Promise.all(arr.map(...))` for parallel.
- Cache rarely-changing values into a `const` outside hot loops; closures over let-vars cost shape changes.
- Always set `package.json` `"type": "module"` for new projects; ESM is the default in 2025+.
- Use `node:test` for zero-dep test runs; switch to vitest when you want richer assertions and snapshots.
- Run `node --watch` in dev for free hot-reload; pair with `--env-file=.env` for dotenv without a library.
- Profile before optimizing — V8 is much smarter than your micro-benchmark intuition.
- Read the TC39 proposals page periodically; new stage-3 features tend to land in Node within months.

## See Also

- typescript
- polyglot
- python
- ruby
- lua
- go
- rust
- c
- java
- bash
- make
- webassembly

## References

- [MDN JavaScript Reference](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference) -- comprehensive language and API docs
- [MDN JavaScript Guide](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide) -- tutorials from beginner to expert
- [ECMAScript Specification (TC39)](https://tc39.es/ecma262/) -- the living language standard
- [TC39 Proposals](https://github.com/tc39/proposals) -- upcoming features and their stages
- [You Don't Know JS (Yet)](https://github.com/getify/You-Dont-Know-JS) -- Kyle Simpson's deep-dive book series
- [JavaScript.info](https://javascript.info/) -- modern, thorough tutorial
- [Node.js Documentation](https://nodejs.org/docs/latest/api/) -- Node API reference
- [Deno Manual](https://docs.deno.com/) -- Deno runtime guide
- [Bun Documentation](https://bun.sh/docs) -- Bun runtime, bundler, test runner
- [V8 Blog](https://v8.dev/blog) -- V8 engine internals, optimizations, new features
- [Web Platform Docs](https://web.dev/) -- browser APIs, performance, modern web
- [Can I Use](https://caniuse.com/) -- browser feature compatibility
- [npm Registry](https://www.npmjs.com/) -- package registry
- [Node.js Best Practices](https://github.com/goldbergyoni/nodebestpractices) -- production-grade patterns
- [JSConf YouTube Channel](https://www.youtube.com/@jsconf) -- talks from the JS community
- [Web.dev Performance](https://web.dev/learn/performance) -- modern performance techniques
- [Vitest](https://vitest.dev/) -- fast Vite-native test runner
- [esbuild](https://esbuild.github.io/) -- Go-based bundler/transpiler
- [Vite](https://vitejs.dev/) -- modern dev server and build tool
- [ECMA-262 Index](https://tc39.es/ecma262/multipage/) -- multipage spec for searchable navigation
