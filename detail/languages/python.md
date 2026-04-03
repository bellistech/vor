# The Internals of CPython — GIL, Memory, and Bytecode

> *CPython is not just "Python" — it's a specific implementation with a Global Interpreter Lock, a three-tier memory allocator (arenas/pools/blocks), and a stack-based bytecode VM. Understanding these internals explains why Python behaves the way it does under load.*

---

## 1. The Global Interpreter Lock (GIL)

### What It Is

The GIL is a **mutex** that protects access to Python objects. Only one thread can execute Python bytecode at a time, even on multi-core machines.

$$\text{Threads executing Python bytecode} \leq 1 \quad \text{(at any instant)}$$

### Why It Exists

CPython uses **reference counting** for memory management. Every object has a reference count:

```c
typedef struct {
    Py_ssize_t ob_refcnt;    // reference count
    PyTypeObject *ob_type;    // type pointer
} PyObject;
```

Without the GIL, every `ob_refcnt` increment/decrement would need to be atomic, which has significant overhead on every object operation.

### GIL Release Schedule

The GIL is released:
1. Every **N bytecode instructions** (default: 5ms interval, `sys.setswitchinterval()`)
2. During **I/O operations** (file read, network, sleep)
3. During **C extension calls** that explicitly release it (`Py_BEGIN_ALLOW_THREADS`)

### Practical Impact

| Workload | Multi-threading Benefit | Why |
|:---------|:-----------------------|:----|
| I/O-bound | Yes | GIL released during I/O |
| CPU-bound | **No** (may be slower) | GIL contention overhead |
| C extensions (NumPy) | Yes | Extensions release GIL |
| `multiprocessing` | Yes | Separate processes, separate GILs |

### GIL-Free Python (PEP 703, Python 3.13+)

The `--disable-gil` build makes `ob_refcnt` atomic and adds per-object locks. Overhead on single-threaded code: approximately 5-10%.

---

## 2. CPython Memory Architecture

### Three-Tier Allocator

```
Level 3:  Object-specific allocators (list, dict, tuple freelists)
Level 2:  pymalloc (arenas → pools → blocks)         ← objects ≤ 512 bytes
Level 1:  Platform malloc (libc)                       ← objects > 512 bytes
Level 0:  OS virtual memory (mmap/brk)
```

### pymalloc: Arenas, Pools, and Blocks

| Unit | Size | Contains |
|:-----|:-----|:---------|
| Arena | 256 KB | 64 pools |
| Pool | 4 KB (one page) | Blocks of one size class |
| Block | 8 to 512 bytes (step 8) | One Python object |

### Size Classes

There are **64 size classes**, each a multiple of 8 bytes:

$$\text{size\_class}(n) = \lceil n / 8 \rceil \times 8$$

| Request Size | Size Class | Blocks per Pool |
|:---:|:---:|:---:|
| 1-8 bytes | 8 | 512 |
| 9-16 bytes | 16 | 256 |
| 17-24 bytes | 24 | 170 |
| 25-32 bytes | 32 | 128 |
| ... | ... | ... |
| 505-512 bytes | 512 | 8 |

### Memory Fragmentation

Arenas are sorted by **utilization** — the most-full arena is used first. Empty arenas are returned to the OS. However, a single live object in an arena prevents its release — this is why long-running Python processes can appear to "leak" memory.

### Object Freelists

Common types maintain **freelists** — pre-allocated pools of recently freed objects for reuse:

| Type | Freelist Size | Why |
|:-----|:---:|:------|
| `int` | Small ints [-5, 256] cached permanently | Extremely common |
| `float` | 100 | Avoid frequent alloc/free |
| `tuple` | 2000 (per size 0-19) | Tuple-heavy internal usage |
| `list` | 80 | Common container |
| `dict` | 80 | Common container |

### Worked Example: Small Integer Caching

```python
a = 256
b = 256
a is b       # True — same cached object

a = 257
b = 257
a is b       # False — different objects (outside cache range)
```

---

## 3. CPython Bytecode VM

### The Evaluation Loop

CPython compiles source to **bytecode** (`.pyc` files), then executes it in a **stack-based virtual machine** (`ceval.c`).

```
Source code (.py)
    │ compile()
    ▼
Code object (co_code = bytecode, co_consts, co_names, ...)
    │
    ▼
Frame object (f_code, f_locals, f_globals, f_stack)
    │ _PyEval_EvalFrameDefault()
    ▼
Execution (fetch-decode-execute loop)
```

### Key Bytecode Instructions

```python
import dis
dis.dis(lambda x, y: x + y * 2)
```

Output:
```
  0 LOAD_FAST    0 (x)
  2 LOAD_FAST    1 (y)
  4 LOAD_CONST   1 (2)
  6 BINARY_MULTIPLY
  8 BINARY_ADD
 10 RETURN_VALUE
```

### Frame Object Structure

Each function call creates a **frame object**:

| Field | Purpose |
|:------|:--------|
| `f_code` | Code object (bytecode + constants + names) |
| `f_locals` | Local variable dict |
| `f_globals` | Module global dict |
| `f_builtins` | Built-in functions dict |
| `f_back` | Caller's frame (call stack) |
| `f_lasti` | Last bytecode instruction index |
| `f_stack` | Evaluation stack (operand stack) |

### Specializing Adaptive Interpreter (3.11+)

CPython 3.11 introduced **inline caching** and **instruction specialization**:

```
LOAD_ATTR generic → LOAD_ATTR_INSTANCE_VALUE  (known attribute offset)
BINARY_ADD generic → BINARY_ADD_INT            (both operands are int)
CALL generic      → CALL_PY_EXACT_ARGS        (known Python function)
```

Each instruction starts generic, then **specializes** after a few executions based on observed types. If the type changes, it **de-specializes** back to generic.

---

## 4. Dictionary Implementation

### Hash Table with Open Addressing

Python dicts use a **compact hash table** (since 3.6):

```
Indices array:  [_, 0, _, _, 2, _, 1, _]  ← sparse, stores index into entries
Entries array:  [(hash, key, value),        ← dense, insertion-ordered
                 (hash, key, value),
                 (hash, key, value)]
```

### Hash Collision Resolution

Python uses **open addressing** with a **perturbation probe**:

$$j = ((5 \times j) + 1 + \text{perturb}) \mod 2^k$$
$$\text{perturb} \mathrel{>>}= 5$$

This explores all slots and avoids clustering better than linear probing.

### Load Factor

Dict resizes when load factor exceeds $\frac{2}{3}$:

$$\text{resize when } \frac{\text{used} + \text{deleted}}{\text{table\_size}} > \frac{2}{3}$$

New size: next power of 2 that gives load factor $\leq \frac{1}{3}$ (roughly $4 \times \text{used}$).

---

## 5. Object Model — Everything Is an Object

### Type Hierarchy

```
type (metaclass — its own type!)
  │
  ├── object (base of all classes)
  │     ├── int
  │     ├── str
  │     ├── list
  │     ├── dict
  │     ├── function
  │     └── ...
  │
  └── type itself inherits from object
```

The circular dependency: `type` is an instance of itself, and `type` inherits from `object`, but `object` is an instance of `type`.

### Method Resolution Order (MRO)

Python uses the **C3 linearization** algorithm for multiple inheritance:

$$\text{MRO}(C) = C + \text{merge}(\text{MRO}(B_1), \ldots, \text{MRO}(B_n), [B_1, \ldots, B_n])$$

The merge operation takes the first element from the first list that doesn't appear in the tail of any other list.

### Descriptor Protocol

Attribute lookup follows:
1. **Data descriptor** on type (has `__set__` or `__delete__`) — e.g., `property`
2. **Instance dict** (`obj.__dict__`)
3. **Non-data descriptor** on type (has only `__get__`) — e.g., methods

This is why `property` overrides instance attributes but plain methods don't.

---

## 6. Reference Counting + Cycle Collector

### Reference Counting

Every assignment, argument pass, or container insertion increments the refcount. When refcount hits 0, the object is immediately freed.

$$\text{DECREF}(o): \quad o.\text{refcnt} -= 1; \quad \text{if } o.\text{refcnt} = 0 \text{ then free}(o)$$

### The Cycle Problem

Reference counting cannot collect cycles:

```python
a = []
a.append(a)   # a → a (refcount = 1, but unreachable after del a)
del a          # refcount drops to 1, not 0 — LEAKED
```

### Generational Cycle Collector

Three generations with different collection frequencies:

| Generation | Contains | Threshold | Collection Frequency |
|:---:|:---------|:---:|:------|
| 0 | Newly created objects | 700 | Most frequent |
| 1 | Survived gen 0 | 10 gen-0 collections | Medium |
| 2 | Survived gen 1 | 10 gen-1 collections | Least frequent |

The collector uses a **mark-and-sweep** algorithm on containers only (ints, strings can't form cycles). It detects cycles by temporarily decrementing refcounts for internal references and checking if any reach 0.

---

## 7. Summary of Key Internals

| Concept | Mechanism | Key Number |
|:--------|:----------|:-----------|
| GIL | Mutex on bytecode execution | 1 thread at a time |
| pymalloc cutoff | Objects routed to pymalloc vs libc | 512 bytes |
| Arena size | Unit of OS memory allocation | 256 KB |
| Pool size | Unit of size-class allocation | 4 KB |
| Small int cache | Permanently cached integers | [-5, 256] |
| Dict load factor | Resize threshold | 2/3 |
| Cycle collector | Generational mark-sweep | 3 generations |
| Bytecode switch interval | GIL release frequency | 5 ms |

---

*CPython is a reference-counted, GIL-protected, bytecode-interpreted runtime with a three-tier memory allocator. Every "why is Python slow?" question has a specific answer in these internals — and every performance optimization (NumPy, asyncio, multiprocessing) is a strategy for working around one of them.*
