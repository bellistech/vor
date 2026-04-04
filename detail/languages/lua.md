# The Internals of Lua — Tables, VM, and Metatables

> *Lua is a register-based VM with a single data structure (the table) that implements arrays, dictionaries, objects, modules, and environments. Its simplicity is not accidental — every design decision optimizes for embedding, small footprint (~250KB), and predictable performance.*

---

## 1. The Table — Lua's Universal Data Structure

### Dual Representation

Every Lua table has two parts:

```
Table
 ├── Array part:  sequential integer keys [1..n]  — contiguous C array
 └── Hash part:   all other keys                   — open-addressed hash table
```

### Array/Hash Boundary

Lua computes the optimal split between array and hash parts. The array part stores keys $1..n$ where:

$$n = \max\{k : \text{at least } k/2 \text{ of keys } 1..k \text{ are non-nil}\}$$

The goal: the array part is at least **50% utilized**. Keys beyond this, or non-integer keys, go to the hash part.

### Worked Example

```lua
t = {10, 20, nil, 40, nil, nil, nil, 80}
-- Keys 1-8: 4 non-nil values
-- Is 4 >= 8/2? Yes → array part = 8 slots
-- But keys 1-4: 3 non-nil, 3 >= 4/2? Yes → could also be 4

-- Lua picks the largest power of 2 that's ≥50% full:
-- 1-8: 4/8 = 50% → array part size = 8
```

### Hash Table Collision Resolution

The hash part uses **open addressing** with a twist: colliding elements are stored in a linked list **within the table** using free positions. No external chaining, no separate allocation.

$$\text{probe}(key) = \text{hash}(key) \mod 2^k$$

Resize when load factor reaches 100% (all slots full). New size is next power of 2.

---

## 2. Metatables — Operator Overloading and OOP

### Metamethod Dispatch

When an operation fails on a value (e.g., adding two tables), Lua checks the **metatable**:

$$\text{op}(a, b) = \begin{cases}
\text{raw\_op}(a, b) & \text{if } a, b \text{ support op natively} \\
\text{mt}(a).\text{\_\_op}(a, b) & \text{if metatable of } a \text{ has \_\_op} \\
\text{mt}(b).\text{\_\_op}(a, b) & \text{if metatable of } b \text{ has \_\_op} \\
\text{error} & \text{otherwise}
\end{cases}$$

### Key Metamethods

| Metamethod | Triggered By | Use Case |
|:-----------|:-------------|:---------|
| `__index` | `t[k]` when `k` not in `t` | Inheritance, default values |
| `__newindex` | `t[k] = v` when `k` not in `t` | Proxy tables, validation |
| `__add` | `a + b` | Operator overloading |
| `__call` | `t()` | Callable objects |
| `__len` | `#t` | Custom length |
| `__tostring` | `tostring(t)` | String representation |
| `__gc` | Garbage collection | Destructor/finalizer |

### Prototype-Based OOP via `__index`

```lua
Animal = {}
Animal.__index = Animal

function Animal:new(name)
    return setmetatable({name = name}, self)
end

function Animal:speak()
    return self.name .. " speaks"
end

-- Lookup chain: instance → Animal (via __index)
-- obj.speak → obj has no "speak" → check __index → find Animal.speak
```

The `__index` chain forms a **linked list of prototypes**, similar to JavaScript's prototype chain.

---

## 3. The Lua VM — Register-Based Bytecode

### Register vs Stack Based VMs

| Feature | Stack-based (JVM, CPython) | Register-based (Lua, Dalvik) |
|:--------|:--------------------------|:-----------------------------|
| Instruction format | 0 operands (implicit stack) | 2-3 operands (register indices) |
| Instructions needed | More (push/pop overhead) | Fewer (direct register access) |
| Code size | Smaller per instruction | Larger per instruction |
| Dispatch overhead | Higher (more instructions) | Lower (fewer instructions) |

### Instruction Format (Lua 5.4)

Each instruction is 32 bits:

```
iABC:   [opcode:7] [A:8] [k:1] [B:8] [C:8]
iABx:   [opcode:7] [A:8] [Bx:17]
iAsBx:  [opcode:7] [A:8] [sBx:17]     (signed)
iAx:    [opcode:7] [Ax:25]
```

### Example Bytecode

```lua
local x = 1 + 2 * 3
```

Compiles to:
```
LOADI    R0  1       ; R0 = 1
LOADI    R1  2       ; R1 = 2
LOADI    R2  3       ; R2 = 3
MUL      R1  R1  R2  ; R1 = R1 * R2 = 6
ADD      R0  R0  R1  ; R0 = R0 + R1 = 7
```

Note: register-based needs 5 instructions vs ~7 for a stack machine (no PUSH/POP).

### Constant Folding

The Lua compiler performs **constant folding** at compile time:

```lua
local x = 1 + 2 * 3
-- Actually compiles to:
-- LOADI R0 7
```

---

## 4. Closures and Upvalues

### The Problem

Lua functions can reference variables from enclosing scopes. When the enclosing function returns, those variables are on a dead stack frame.

### Upvalue Mechanism

An **upvalue** is a reference to a variable in an enclosing scope. While the variable is still on the stack, the upvalue points directly to the stack slot. When the enclosing function returns, the upvalue is **closed**: the value is moved from the stack into the upvalue object itself.

```
Before close:                After close:
┌─────────┐                 ┌─────────┐
│ Upvalue │──→ Stack slot   │ Upvalue │──→ Internal storage
└─────────┘                 └─────────┘    (value copied here)
```

### Upvalue Sharing

Multiple closures referencing the same variable share the **same upvalue object**:

```lua
function counter()
    local n = 0                    -- one upvalue object for n
    return {
        inc = function() n = n + 1 end,  -- shares upvalue
        get = function() return n end,    -- shares upvalue
    }
end
```

---

## 5. Garbage Collector — Incremental Tri-Color Mark-Sweep

### Algorithm

Lua uses an **incremental, non-moving, tri-color mark-and-sweep** GC:

| Phase | Action | Incremental? |
|:------|:-------|:-------------|
| Mark | Traverse object graph (white → grey → black) | Yes (interleaved with mutator) |
| Sweep | Free white objects, reset black to white | Yes |
| Finalize | Call `__gc` metamethods | Yes |

### GC Pacing

The GC is controlled by two parameters:

$$\text{pause} = \frac{\text{threshold}}{\text{live}} \times 100$$

- **gcpause** (default 200): How long to wait between cycles. At 200, GC starts when memory doubles.
- **gcstepmul** (default 100): GC speed relative to allocation speed. Higher = more aggressive.

### Weak Tables

Tables with weak keys/values allow the GC to collect entries:

| Mode | Syntax | Collected When |
|:-----|:-------|:---------------|
| Weak keys | `setmetatable(t, {__mode = "k"})` | Key is unreachable |
| Weak values | `setmetatable(t, {__mode = "v"})` | Value is unreachable |
| Both | `setmetatable(t, {__mode = "kv"})` | Either unreachable |

Use case: caches, memoization tables that don't prevent GC.

---

## 6. Coroutines — Cooperative Multitasking

### Coroutine vs Thread

| Feature | Coroutine | OS Thread |
|:--------|:----------|:----------|
| Scheduling | Cooperative (explicit yield) | Preemptive (OS scheduler) |
| Stack | ~1KB initial, grows | 1-8MB fixed |
| Context switch | ~100 ns | ~1-10 us |
| Parallelism | No (single thread) | Yes |

### State Machine

```
              create()           resume()
    [dead] ──────────► [suspended] ──────► [running]
                            ▲                  │
                            │    yield()        │
                            └──────────────────┘
                                               │
                            [dead] ◄───────────┘
                                    (function returns)
```

### Implementation

Each coroutine has its own **Lua stack** (separate from the C stack). `yield()` saves the instruction pointer and stack state. `resume()` restores them. No OS involvement.

---

## 7. String Interning

All strings in Lua are **interned**: stored in a global hash table. Two strings with the same content are the **same object**.

$$\text{str\_a} == \text{str\_b} \iff \text{ptr}(\text{str\_a}) = \text{ptr}(\text{str\_b})$$

Consequences:
- String comparison is $O(1)$ (pointer comparison)
- String creation is $O(n)$ (must hash and check table)
- Memory: no duplicate strings
- Strings are immutable (required for interning safety)

**Exception:** Lua 5.4 does not intern strings longer than 40 bytes (to avoid hashing cost).

---

## 8. Summary of Key Internals

| Concept | Mechanism | Key Detail |
|:--------|:----------|:-----------|
| Table array part | C array for keys 1..n | >=50% utilization rule |
| Table hash part | Open addressing, internal chaining | Resize at 100% load |
| VM architecture | Register-based, 32-bit instructions | Fewer dispatches than stack VM |
| Upvalues | Stack pointer → closed copy | Shared across closures |
| GC | Incremental tri-color mark-sweep | gcpause=200, gcstepmul=100 |
| String interning | Global hash table | O(1) comparison |
| Coroutine stack | Independent Lua stack per coroutine | ~1KB initial |

---

*Lua's genius is radical simplicity: one data structure (table), one number type (double, or integer+float in 5.3+), one extension mechanism (metatables). The entire source is ~30,000 lines of C. This is not a limitation — it's a design philosophy that makes Lua the most embeddable language ever created.*

## Prerequisites

- Hash table internals (array part vs hash part, resizing)
- C API fundamentals (stack-based argument passing, FFI)
- Coroutines and cooperative multitasking
- Metatables and metamethod dispatch
