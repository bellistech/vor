# The Internals of JavaScript — V8 Engine, Event Loop, and Type Coercion

> *JavaScript is a single-threaded, prototype-based, dynamically-typed language with a JIT-compiled runtime. V8's hidden classes, inline caches, and tiered compilation (Ignition interpreter + TurboFan optimizing compiler) make it fast. The event loop with its microtask and macrotask queues defines its concurrency model.*

---

## 1. V8 Compilation Pipeline

### Tiered Compilation

```
Source code
    │ Parser
    ▼
AST (Abstract Syntax Tree)
    │ Ignition (bytecode compiler)
    ▼
Bytecode (register-based VM)
    │ Sparkplug (non-optimizing, 1:1 bytecode→machine code)
    ▼
Baseline machine code
    │ TurboFan (optimizing compiler, triggered by hotness counter)
    ▼
Optimized machine code
    │ Deoptimization (type assumption violated)
    ▼
Back to bytecode
```

### Hot Function Detection

V8 tracks invocation count and loop iterations. When a function exceeds the **hotness threshold** (~1000 invocations or ~10000 loop iterations), it's queued for TurboFan optimization.

### Deoptimization

If TurboFan assumed `x` is always an integer but it receives a string, it **deoptimizes** — discards optimized code and falls back to Ignition bytecode. This is expensive and should be avoided by keeping types stable.

---

## 2. Hidden Classes (Maps/Shapes)

### The Problem

JavaScript objects are dictionaries — properties can be added/removed dynamically. Naive implementation: hash table per object. Slow.

### V8's Solution: Hidden Classes

V8 assigns a **hidden class** (called "Map" in V8, "Shape" in SpiderMonkey) to every object. Objects with the same property names in the same order share the same hidden class.

```javascript
function Point(x, y) {
    this.x = x;    // Transition: Map0 → Map1 (adds .x at offset 0)
    this.y = y;    // Transition: Map1 → Map2 (adds .y at offset 8)
}

let p1 = new Point(1, 2);   // Hidden class: Map2
let p2 = new Point(3, 4);   // Hidden class: Map2 (same!)
```

Property access becomes a fixed-offset load — same as C structs. $O(1)$ instead of hash table $O(1)$ amortized with worse constants.

### Transition Trees

Hidden classes form a **tree** of transitions:

```
Map0 (empty)
  │ add .x
  ▼
Map1 {x: offset 0}
  │ add .y
  ▼
Map2 {x: offset 0, y: offset 8}
```

### What Breaks Hidden Classes

```javascript
// BAD: different property order → different hidden classes
let a = {}; a.x = 1; a.y = 2;   // Map: x→y
let b = {}; b.y = 2; b.x = 1;   // Map: y→x (different!)

// BAD: deleting properties invalidates hidden class
delete a.x;  // Falls back to dictionary mode (slow)
```

---

## 3. Inline Caches (ICs)

### Monomorphic, Polymorphic, Megamorphic

At each property access site, V8 caches the hidden class and property offset:

| IC State | Hidden Classes Seen | Lookup Cost |
|:---------|:---:|:------|
| Uninitialized | 0 | Full lookup |
| Monomorphic | 1 | Compare hidden class, load offset — fastest |
| Polymorphic | 2-4 | Linear search through cache entries |
| Megamorphic | >4 | Fall back to hash table — slowest |

### Worked Example

```javascript
function getX(obj) { return obj.x; }

getX({x: 1, y: 2});   // IC becomes monomorphic (Map: {x, y})
getX({x: 3, y: 4});   // Same map — cache HIT

getX({x: 5, z: 6});   // Different map — IC becomes polymorphic
getX({x: 7});          // Different map — still polymorphic (3 entries)
// After 4+ different maps: megamorphic — full lookup every time
```

---

## 4. The Event Loop

### Single Thread, Multiple Queues

```
┌─────────────────────────────────────┐
│           Call Stack                │
│  (synchronous execution)           │
└──────────┬──────────────────────────┘
           │ empty?
           ▼
┌─────────────────────────────────────┐
│       Microtask Queue               │  ← Promise callbacks, queueMicrotask
│  (drain ALL before next macrotask)  │
└──────────┬──────────────────────────┘
           │ empty?
           ▼
┌─────────────────────────────────────┐
│       Macrotask Queue               │  ← setTimeout, setInterval, I/O
│  (process ONE, then check micro)    │
└─────────────────────────────────────┘
```

### Execution Order Rules

1. Execute synchronous code to completion (call stack empties)
2. Drain **all** microtasks (Promise `.then`, `queueMicrotask`, `MutationObserver`)
3. Execute **one** macrotask (`setTimeout`, `setInterval`, I/O callback)
4. Drain **all** microtasks again
5. Render (in browser: `requestAnimationFrame`, layout, paint)
6. Repeat from step 3

### Worked Example

```javascript
console.log("1");                          // sync
setTimeout(() => console.log("2"), 0);     // macrotask
Promise.resolve().then(() => console.log("3")); // microtask
console.log("4");                          // sync

// Output: 1, 4, 3, 2
```

### `setTimeout(fn, 0)` is NOT Immediate

Minimum delay is **clamped to 4ms** after 5 nested `setTimeout` calls (HTML spec). The callback enters the macrotask queue, which is only processed after all microtasks drain.

---

## 5. Prototype Chain and Inheritance

### Prototype Lookup

Property access `obj.prop` follows the **prototype chain**:

```
obj → obj.__proto__ → obj.__proto__.__proto__ → ... → null
```

$$\text{lookup}(o, p) = \begin{cases}
o.\text{own}[p] & \text{if } p \in o.\text{ownProperties} \\
\text{lookup}(o.\text{\_\_proto\_\_}, p) & \text{if } o.\text{\_\_proto\_\_} \neq \text{null} \\
\text{undefined} & \text{otherwise}
\end{cases}$$

### The Prototype DAG

```
instance ──▶ Constructor.prototype ──▶ Object.prototype ──▶ null
                     │
            Constructor ──▶ Function.prototype ──▶ Object.prototype ──▶ null
```

---

## 6. Type Coercion Rules

### Abstract Equality (`==`) Algorithm

The `==` operator follows a **23-step algorithm** (ECMA-262 7.2.16). Key rules:

| Comparison | Coercion | Result |
|:-----------|:---------|:-------|
| `null == undefined` | Special case | `true` |
| `null == 0` | `null` only equals `undefined` | `false` |
| `"" == false` | Both → Number: `0 == 0` | `true` |
| `"0" == false` | Both → Number: `0 == 0` | `true` |
| `[] == false` | `[] → "" → 0`, `false → 0` | `true` |
| `[] == ![]` | `![] = false`, then `[] == false` | `true` |

### `ToPrimitive` Algorithm

When an object must become a primitive:
1. If hint is "number": try `valueOf()`, then `toString()`
2. If hint is "string": try `toString()`, then `valueOf()`
3. If hint is "default": same as "number" (except `Date` uses "string")

### `+` Operator Overloading

```
a + b:
  1. ToPrimitive(a), ToPrimitive(b)
  2. If EITHER result is a string → string concatenation
  3. Otherwise → numeric addition
```

This is why `[] + [] = ""` and `[] + {} = "[object Object]"` and `{} + [] = 0`.

---

## 7. Closures and Scope Chains

### Lexical Scoping

Each function creates a **lexical environment** — a chain of variable bindings:

```
Global Env: { x: 1 }
  └── outer() Env: { y: 2, parent: Global }
        └── inner() Env: { z: 3, parent: outer }
```

Variable lookup walks up the scope chain: $O(d)$ where $d$ is scope depth.

### Closure Memory

A closure captures its **entire lexical environment**, not just the variables it uses. V8 optimizes this: only captured variables are kept alive (context allocation).

```javascript
function outer() {
    let big = new Array(1000000);  // NOT captured — will be GCed
    let x = 42;                     // captured — kept alive
    return function inner() { return x; };
}
```

---

## 8. Garbage Collection in V8

### Generational Collection

| Heap Space | Objects | Collector | Algorithm |
|:-----------|:--------|:----------|:----------|
| Young generation (nursery) | New objects | Scavenger (Minor GC) | Semi-space copying |
| Old generation | Survived 2 minor GCs | Major GC | Mark-sweep-compact |

### Semi-Space Copying (Young Gen)

Two equal-sized spaces: **from-space** and **to-space**. Allocation in from-space. On GC:
1. Copy live objects from from-space to to-space
2. Swap spaces
3. Old from-space is now free

Cost proportional to **live objects** (not total heap). Fast because most young objects die quickly (generational hypothesis).

### Young Gen Size

Default: 1-8 MB per semi-space. Small size → frequent minor GC → low pause times.

---

## 9. Summary of Key Internals

| Concept | Mechanism | Key Number |
|:--------|:----------|:-----------|
| Hidden classes | Property offset caching | Monomorphic = fastest |
| Inline caches | Per-site type specialization | >4 types = megamorphic |
| TurboFan trigger | Invocation/loop counter | ~1000 calls |
| Event loop | Single thread + task queues | Microtasks drain fully |
| `setTimeout` min delay | HTML spec clamp | 4 ms (nested) |
| Young gen GC | Semi-space copying | ~1-8 MB |
| Prototype chain | Linked list of objects | Terminates at `null` |

---

*JavaScript's performance story is about predictability: keep your types stable (monomorphic ICs), avoid deoptimization (consistent shapes), and understand the event loop (microtasks before macrotasks). The engine does the rest.*

## Prerequisites

- Event loop and asynchronous programming (callbacks, promises, microtasks)
- Prototypal inheritance and object model
- Closures, scoping rules, and execution contexts
- JIT compilation basics (inline caches, hidden classes, deoptimization)
