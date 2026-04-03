# The Internals of C — Memory, Types, and Undefined Behavior

> *C is a thin abstraction over hardware. Understanding it means understanding memory layout, pointer arithmetic, alignment rules, and the taxonomy of undefined behavior that the standard deliberately leaves unspecified.*

---

## 1. Process Memory Layout

A compiled C program's virtual address space (low to high on most architectures):

```
+------------------+ 0x00000000
|      Text        |  Machine code (read-only, executable)
+------------------+
|      Rodata      |  String literals, const globals (read-only)
+------------------+
|      Data        |  Initialized globals/statics (read-write)
+------------------+
|      BSS         |  Uninitialized globals/statics (zero-filled)
+------------------+
|      Heap        |  malloc/calloc/realloc (grows upward)
|        |         |
|        v         |
|                  |
|        ^         |
|        |         |
|      Stack       |  Local variables, return addresses (grows downward)
+------------------+ 0xFFFFFFFF
```

### Segment Sizes

| Segment | Contents | Lifetime | Initialized |
|:--------|:---------|:---------|:------------|
| Text | Instructions | Process | By compiler |
| Rodata | `"hello"`, `const int x = 5` | Process | By compiler |
| Data | `int g = 42;` (file scope) | Process | By compiler |
| BSS | `int g;` (file scope) | Process | Zero-filled by OS |
| Heap | `malloc(n)` | Manual | Not initialized |
| Stack | Local vars, parameters | Scope | Not initialized |

### Worked Example

```c
const char *s = "hello";   // s in Data, "hello" in Rodata
int g = 10;                 // g in Data
int z;                      // z in BSS (zero-initialized)

void f(void) {
    int x = 5;              // x on Stack
    int *p = malloc(100);   // p on Stack, 100 bytes on Heap
    static int c = 0;       // c in Data (persists across calls)
}
```

---

## 2. Alignment and Padding Rules

### The Alignment Rule

Every type $T$ has an alignment requirement $a(T)$. A variable of type $T$ must be placed at an address $A$ such that:

$$A \mod a(T) = 0$$

Typical alignments (LP64 / x86-64):

| Type | Size (bytes) | Alignment |
|:-----|:---:|:---:|
| `char` | 1 | 1 |
| `short` | 2 | 2 |
| `int` | 4 | 4 |
| `float` | 4 | 4 |
| `double` | 8 | 8 |
| `long` | 8 | 8 |
| `void *` | 8 | 8 |

### Struct Padding Formula

For a struct, the compiler inserts padding bytes so each member satisfies alignment. The struct's own alignment is the maximum of its members':

$$a(\text{struct}) = \max(a(m_1), a(m_2), \ldots, a(m_n))$$

The struct's total size is rounded up to a multiple of $a(\text{struct})$:

$$\text{sizeof}(\text{struct}) = \lceil \text{last\_member\_end} / a(\text{struct}) \rceil \times a(\text{struct})$$

### Worked Example

```c
struct bad {       // Offset  Size  Padding
    char a;        //   0       1     3 bytes padding
    int b;         //   4       4
    char c;        //   8       1     3 bytes padding
};                 // Total: 12 bytes (not 6!)

struct good {      // Offset  Size  Padding
    int b;         //   0       4
    char a;        //   4       1
    char c;        //   5       1     2 bytes tail padding
};                 // Total: 8 bytes
```

**Rule of thumb:** sort members largest-to-smallest to minimize padding.

### The `offsetof` Macro

$$\text{offsetof}(T, m) = \text{byte offset of member } m \text{ within type } T$$

Defined in `<stddef.h>`. Implementation: `(size_t)&(((T *)0)->m)`.

---

## 3. Pointer Arithmetic

### The Fundamental Rule

For a pointer `p` of type `T *`, the expression `p + n` yields:

$$\text{addr}(p + n) = \text{addr}(p) + n \times \text{sizeof}(T)$$

This is why pointer arithmetic is type-aware. Subtracting two pointers of the same type:

$$p_2 - p_1 = \frac{\text{addr}(p_2) - \text{addr}(p_1)}{\text{sizeof}(T)}$$

The result type is `ptrdiff_t` (signed).

### Worked Example

```c
int arr[5] = {10, 20, 30, 40, 50};
int *p = arr;        // p points to arr[0], say address 0x1000

// p + 3 = 0x1000 + 3 * 4 = 0x100C → arr[3] = 40

int *q = &arr[4];
ptrdiff_t d = q - p; // (0x1010 - 0x1000) / 4 = 4
```

### Array-Pointer Duality

The expression `a[i]` is defined as `*(a + i)`, which means `i[a]` is also valid C (commutativity of addition). This is not a trick — it follows directly from the definition.

---

## 4. The C Type System — Formal View

### Type Categories

```
Types
 ├── Object types (have size and representation)
 │    ├── Scalar types
 │    │    ├── Arithmetic types
 │    │    │    ├── Integer types (char, short, int, long, _Bool, enum)
 │    │    │    └── Floating types (float, double, long double)
 │    │    └── Pointer types
 │    ├── Aggregate types
 │    │    ├── Array types
 │    │    └── Structure types
 │    └── Union types
 └── Function types (no size)
```

### Integer Promotion Rules

When operands have different types, C promotes the "narrower" type. The rule (simplified):

1. If either operand is `long double` → convert other to `long double`
2. Else if either is `double` → convert to `double`
3. Else if either is `float` → convert to `float`
4. Else apply **integer promotions**: anything narrower than `int` becomes `int`
5. Then if different signedness: unsigned wins if same or wider rank

### The `sizeof` Operator

Returns `size_t` (unsigned). Key identities:

$$\text{sizeof}(\text{char}) = 1 \quad \text{(always, by definition)}$$
$$\text{sizeof}(a) / \text{sizeof}(a[0]) = n \quad \text{(array length, only for true arrays)}$$

---

## 5. Undefined Behavior Taxonomy

The C standard defines three categories of non-portable behavior:

| Category | Definition | Example |
|:---------|:-----------|:--------|
| Implementation-defined | Compiler must document choice | `sizeof(int)`, right-shift of negative |
| Unspecified | Compiler may choose, no documentation needed | Evaluation order of function args |
| Undefined (UB) | Anything can happen | Signed overflow, null dereference |

### The Most Common UB Sources

| UB | Why It's UB | What Compilers Actually Do |
|:---|:------------|:--------------------------|
| Signed integer overflow | $a + b$ where $a + b > \text{INT\_MAX}$ | Assume it never happens; optimize accordingly |
| Null pointer dereference | `*((int *)0)` | May eliminate null checks |
| Use after free | `free(p); *p` | May reuse memory immediately |
| Buffer overflow | `arr[n]` where $n \geq \text{len}$ | Stack smashing, RCE |
| Double free | `free(p); free(p)` | Heap corruption |
| Uninitialized read | `int x; return x;` | May return anything, including "impossible" values |
| Strict aliasing violation | `*(float *)&int_val` | Miscompilation at `-O2` |

### Why UB Exists

UB is not a bug in the standard — it's a **deliberate optimization contract**. The compiler assumes UB never occurs, which enables:

- Loop optimization (signed overflow can't wrap → induction variable analysis)
- Dead code elimination (null deref is UB → code after null check is unreachable if pointer is null)
- Vectorization (strict aliasing → no need to worry about pointer aliasing)

---

## 6. The Compilation Pipeline

```
Source (.c)
    │
    ├─ 1. Preprocessing (cpp)      Macro expansion, #include, #ifdef
    │     Output: translation unit
    │
    ├─ 2. Compilation (cc1)        Lexing → Parsing → AST → IR → Optimization → Assembly
    │     Output: .s (assembly)
    │
    ├─ 3. Assembly (as)            Assembly → machine code
    │     Output: .o (object file, ELF/Mach-O)
    │
    └─ 4. Linking (ld)             Resolve symbols, relocations
          Output: executable or .so/.dylib
```

### Translation Units and Linkage

Each `.c` file is compiled independently into one **translation unit**. Symbols have linkage:

| Keyword | Linkage | Visibility |
|:--------|:--------|:-----------|
| (none, file scope) | External | All translation units |
| `static` (file scope) | Internal | This translation unit only |
| `extern` | External | Declaration only, defined elsewhere |
| `static` (block scope) | None | Local, but persistent storage |

### The One Definition Rule (ODR)

Each external symbol must be defined **exactly once** across all translation units. Multiple definitions → linker error (or worse, silent UB with `inline`).

---

## 7. `volatile`, `restrict`, and Memory Semantics

### `volatile`

Tells the compiler: "this variable may change outside the program's control." Prevents:
- Caching the value in a register
- Reordering reads/writes
- Eliminating "redundant" reads

**Use case:** memory-mapped I/O registers, signal handlers.

**Not a substitute for atomics.** `volatile` provides no inter-thread ordering guarantees.

### `restrict` (C99)

A promise to the compiler: "this pointer is the only way to access this memory." Enables alias analysis:

```c
void add(int *restrict a, int *restrict b, int *restrict c, int n) {
    for (int i = 0; i < n; i++)
        a[i] = b[i] + c[i];  // Compiler can vectorize — no aliasing possible
}
```

Without `restrict`, the compiler must assume `a`, `b`, `c` might overlap, preventing SIMD.

---

## 8. Function Calling Convention (x86-64 System V ABI)

| Argument # | Integer/Pointer | Floating Point |
|:---:|:---:|:---:|
| 1 | `rdi` | `xmm0` |
| 2 | `rsi` | `xmm1` |
| 3 | `rdx` | `xmm2` |
| 4 | `rcx` | `xmm3` |
| 5 | `r8` | `xmm4` |
| 6 | `r9` | `xmm5` |
| 7+ | Stack | `xmm6`-`xmm7`, then stack |

Return value: `rax` (integer), `xmm0` (float). Structs > 16 bytes returned via hidden pointer in `rdi`.

### Stack Frame Layout

```
High addresses
+------------------+
| Caller's frame   |
+------------------+
| Return address   |  ← pushed by CALL
+------------------+
| Saved rbp        |  ← pushed by callee (if frame pointer used)
+------------------+
| Local variables  |
+------------------+
| Red zone (128B)  |  ← leaf functions can use without adjusting rsp
+------------------+
Low addresses
```

---

## 9. Summary of Key Formulas

| Concept | Formula | Domain |
|:--------|:--------|:-------|
| Pointer arithmetic | $\text{addr}(p+n) = \text{addr}(p) + n \cdot \text{sizeof}(*p)$ | Memory addressing |
| Struct alignment | $a(S) = \max(a(m_i))$ | Data layout |
| Struct size | $\text{sizeof}(S) = \lceil \text{end} / a(S) \rceil \cdot a(S)$ | Data layout |
| Array length | $n = \text{sizeof}(a) / \text{sizeof}(a[0])$ | Compile-time only |
| Full mesh peering | Addresses in prefix: $2^{32-n}$ | CIDR subnetting |
| Stack growth | Downward from high addresses | Architecture ABI |

---

*C gives you a machine. The standard tells you which levers are guaranteed to work, which ones might work, and which ones will detach in your hand. Knowing the difference is the entire discipline.*
