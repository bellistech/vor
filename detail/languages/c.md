# The Internals of C — Abstract Machine, Linkage, Memory, and ABI

> *C is not a portable assembler — it is a contract with the compiler about an abstract machine. The standard tells you what observable behavior must occur; everything else is fair game for optimization. This deep dive maps that contract: the abstract machine, the linkage model, the preprocessor's expansion algorithm, the type system, struct layout, the strict aliasing rule, the taxonomy of undefined behavior, the allocator, inline assembly, the System V AMD64 ABI, the linker and ELF format, signal-handler restrictions, the compilation pipeline, and the modern C surface through C23.*

---

## 1. The C Abstract Machine

The C standard (ISO/IEC 9899) does not describe execution on real hardware. It describes execution on an *abstract machine* whose semantics the implementation must preserve as observed via *observable behavior*. Everything else — register allocation, instruction order, even whether a variable exists in memory at all — is up to the compiler under the **as-if rule** (C17 §5.1.2.3 ¶6).

### 1.1 Observable Behavior

The standard defines exactly three classes of observable behavior:

1. **Accesses to volatile-qualified objects** (their reads and writes must occur in source order)
2. **Data written to files at program termination** (the contents must match the abstract machine's writes)
3. **Prompt input/output via interactive devices** (with the host environment's definition of "prompt")

Anything not observable is fungible. The compiler can reorder, fold, eliminate, vectorize, inline, devirtualize, or replace code wholesale, provided the observable trace matches.

```c
int x = 0;
for (int i = 0; i < 1000000; i++) x += i;
printf("%d\n", x);
```

A compiler at `-O2` typically replaces the loop with `printf("499999500000\n")` — observable behavior is identical, so the loop need not run.

### 1.2 Sequence Points and the Sequenced-Before Relation

C99 introduced *sequence points* (C99 §5.1.2.3); C11 reformulated this as the **sequenced-before** partial order (C11 §5.1.2.3, §6.5). At a sequence point, all side effects of prior evaluations are complete and no later evaluations have begun.

Sequence points (or equivalently, sequenced-before edges) occur at:

- The end of a full expression (statement-terminating `;`)
- The `&&`, `||`, `?:`, and `,` operators (left operand sequenced-before right)
- After evaluation of function arguments and before the call (C11)
- After a function's return value is determined and before any expression in the calling function continues
- Between a function-pointer evaluation and the call through it

### 1.3 Unsequenced, Indeterminately-Sequenced, and Sequenced-Before

C11 distinguishes three relations:

| Relation | Meaning | Example |
|:---------|:--------|:--------|
| Sequenced-before | A finishes before B begins | `;` between statements |
| Indeterminately-sequenced | A or B first, but not interleaved | Function arguments |
| Unsequenced | May interleave, in any order | Subexpressions of `+`, `*` |

Two unsequenced side effects on the same scalar object are *undefined behavior* (C11 §6.5 ¶2). This is what makes `i = i++;` UB — the assignment and the post-increment are unsequenced.

```c
int i = 0;
i = i++;        // UB — unsequenced modification of i
a[i] = i++;     // UB — same scalar modified twice
int x = i + i++; // UB pre-C++17, still UB in C
printf("%d %d\n", i++, i++); // UB — function args indeterminately sequenced, but each arg's side effects on i are unsequenced relative to the other
```

### 1.4 Implementation-Defined, Unspecified, and Undefined

These three terms are not synonyms. The standard defines them precisely (C17 §3.4):

| Term | Definition | Documentation Required | Predictability |
|:-----|:-----------|:----------------------:|:--------------:|
| **Implementation-defined** | A choice the implementation makes | Yes — must be in the docs | Predictable on a given compiler+platform |
| **Unspecified** | A choice the implementation makes | No | Per-invocation may vary |
| **Undefined (UB)** | The standard imposes no requirements | No | Anything: crash, garbage, time travel |

```c
// Implementation-defined: sizeof(int), right-shift of negative
int x = -1 >> 1;          // -1 on most compilers (arithmetic shift), but standard does not require it

// Unspecified: order of function-argument evaluation
int a = f() + g();        // f() may run first, or g() may — the implementation need not be consistent

// Undefined: signed overflow
int y = INT_MAX + 1;      // UB — anything can happen
```

The critical insight: UB is not a runtime concept. If the compiler can *prove* a program path contains UB, it may eliminate that path entirely — even paths that haven't executed yet ("time-travel UB"). This is why `-fsanitize=undefined` instruments potential UB sites at runtime, and why static analyzers chase UB paths aggressively.

### 1.5 Trap Representations

A *trap representation* is a bit pattern that doesn't represent a value of any object type. Reading one (other than via `unsigned char`) is UB. Padding bytes in structs may hold trap representations; that's why `memcmp` on two structs with the same field values can return non-zero — padding bytes differ.

---

## 2. The Linkage Model

C compiles each `.c` file independently into a **translation unit**. The linker stitches translation units together by resolving *symbols*. The linkage model determines which names are visible across translation units.

### 2.1 Linkage Categories

C defines three linkage levels (C17 §6.2.2):

| Linkage | Visibility | Declared with |
|:--------|:-----------|:--------------|
| **External** | All translation units | File-scope without `static`, or `extern` |
| **Internal** | One translation unit | File-scope with `static` |
| **None** | One block | Block-scope variables; function parameters |

```c
// File scope
int g_external = 0;        // external linkage
static int g_internal = 0; // internal linkage
extern int g_decl;         // external linkage, declaration only

void f(void) {
    int local = 0;          // no linkage (block scope)
    static int s = 0;       // no linkage, but static storage duration
}
```

### 2.2 Declarations, Tentative Definitions, and Definitions

C distinguishes three states for a file-scope identifier (C17 §6.9.2):

- **Declaration**: introduces a name, may not allocate storage
- **Tentative definition**: a declaration without an initializer that *may* become a definition at end of TU
- **Definition**: the actual storage allocation

```c
extern int x;     // declaration only
int y;            // tentative definition — becomes int y = 0; if no other definition appears
int y = 5;        // definition (if this and `int y;` both appear, the definition wins)
int z = 10;       // definition
```

Multiple tentative definitions in the same TU are merged. Across TUs, multiple definitions of the same external symbol violate the **One Definition Rule (ODR)** and produce a linker error (or, with `-fcommon` and historical Unix behavior, silently merge them — a long-standing footgun that GCC 10+ disabled by default).

### 2.3 The `inline` + `extern` Dance (C99)

C99 introduced `inline`, but with semantics that surprise even experienced C programmers. There are three states:

```c
// Pattern 1: inline definition (inline keyword in header, no extern)
inline int square(int x) { return x * x; }
// In each TU including this header, the compiler may inline-expand it.
// No external symbol is emitted. If the compiler decides not to inline,
// you get a linker error unless one TU also provides an external definition.

// Pattern 2: external inline definition (header has inline, one .c file has extern inline)
// In header.h:
inline int square(int x) { return x * x; }
// In one .c file:
extern inline int square(int x);   // forces emission of an external definition

// Pattern 3: static inline (the safest, most common)
static inline int square(int x) { return x * x; }
// Each TU gets its own copy; no linker conflict; no externally visible symbol.
```

The C99 model is the *opposite* of C++'s: in C++, `inline` allows multiple definitions and merges them. In C99, `inline` *suppresses* the external definition. This is why `static inline` is the de-facto idiom in headers — it sidesteps the trap entirely.

### 2.4 Weak Symbols

`__attribute__((weak))` (GCC/Clang extension) allows multiple definitions of the same symbol; the linker picks one, with strong definitions winning over weak. Used for default implementations, library hooks, and symbol versioning.

```c
__attribute__((weak)) void on_init(void) { /* default no-op */ }

// In a user's TU, this strong definition overrides:
void on_init(void) { printf("hooked\n"); }
```

### 2.5 Common Symbols

A historical Unix feature: uninitialized globals (`int x;`) became "common" symbols, allowing multiple translation units to declare the same global without linker error. GCC 10+ defaults to `-fno-common`, treating tentative definitions as ordinary definitions. This caught a generation of bugs (silent symbol collision) and broke a generation of legacy code.

### 2.6 Symbol Visibility (ELF/Mach-O)

Beyond C's linkage model, ELF adds visibility attributes:

```c
__attribute__((visibility("hidden"))) void internal(void);  // not exported from .so
__attribute__((visibility("default"))) void api_func(void); // exported (default)
__attribute__((visibility("protected"))) void p(void);     // exported, but not preemptable
```

`-fvisibility=hidden` makes hidden the default; you opt into export. This shrinks `.dynsym`, speeds up dynamic linking, and prevents symbol interposition surprises.

---

## 3. Preprocessor Evaluation Model

The preprocessor (cpp) is a separate language with its own grammar, executed before the C compiler proper sees the source. Understanding its evaluation model explains every bizarre macro behavior.

### 3.1 Translation Phases

C17 §5.1.1.2 defines **eight translation phases**, executed in order:

1. Physical source character mapping (e.g., trigraph replacement — deprecated in C23)
2. Line splicing (backslash-newline removal)
3. Tokenization and comment-to-space replacement
4. Preprocessing directives executed; macros expanded; `#include` files inserted
5. Source character → execution character set conversion in literals
6. Adjacent string literal concatenation
7. The C compiler proper: parsing and semantic analysis
8. Linking

This order is observable. For example, `\` at end of line splices *before* tokenization, so:

```c
// This is a co\
mment   // <- still a comment! line-splicing happens before tokens form
```

### 3.2 Macro Expansion Algorithm

C macros are not simple text substitution. The expansion algorithm (C17 §6.10.3) is:

1. Identify the invocation token sequence
2. Substitute arguments (with full macro replacement applied to arguments first, *unless* the parameter appears with `#` or `##`)
3. Apply `#` (stringize) and `##` (concatenation) operators
4. **Rescan** the result for further replacement, with the **further-replacement-prevention rule**: a macro currently being expanded is "painted blue" — if encountered during rescanning, it is *not* re-expanded

```c
#define A B
#define B A
A    // expands to B (A is painted blue), then B paints itself blue, then no more expansion → "B"
```

This rule prevents infinite recursion. It also means macros can be self-referential without looping:

```c
#define puts puts_logged(__FILE__, __LINE__, puts)
// invoking puts(s) expands to puts_logged(__FILE__, __LINE__, puts)(s)
// (but only if you take the address — calling needs more care)
```

### 3.3 The `#` (Stringize) and `##` (Token Paste) Operators

`#param` produces a string literal of the argument's text (with escapes inserted for `"` and `\`):

```c
#define STR(x) #x
STR(foo bar)   // "foo bar"
STR("a")       // "\"a\""
```

`a ## b` produces a single token from concatenating `a` and `b` (must form a valid token):

```c
#define CONCAT(a, b) a ## b
int CONCAT(my_, var) = 0;  // expands to: int my_var = 0;
```

A subtle gotcha: `#` and `##` operate on *unexpanded* arguments. To get expansion before stringize/paste, use a two-level macro:

```c
#define STR_IMPL(x) #x
#define STR(x) STR_IMPL(x)
#define VERSION 42
STR(VERSION)       // "42" (two-level forces expansion)
STR_IMPL(VERSION)  // "VERSION" (single-level, no expansion)
```

### 3.4 Variadic Macros (C99)

```c
#define LOG(fmt, ...) fprintf(stderr, fmt, __VA_ARGS__)
LOG("hello %s\n", name);
```

`__VA_ARGS__` holds the variadic tokens. For zero arguments, GCC's `, ##__VA_ARGS__` extension swallows the leading comma; C20 standardizes this as `__VA_OPT__(,)`:

```c
#define LOG(fmt, ...) fprintf(stderr, fmt __VA_OPT__(,) __VA_ARGS__)
LOG("hello\n");        // fprintf(stderr, "hello\n")
LOG("hello %s\n", x);  // fprintf(stderr, "hello %s\n", x)
```

### 3.5 Predefined Macros

Mandated by the standard (C17 §6.10.8):

| Macro | Value |
|:------|:------|
| `__FILE__` | Current source filename (string) |
| `__LINE__` | Current line number (integer) |
| `__DATE__` | Compile date `"Mmm dd yyyy"` |
| `__TIME__` | Compile time `"hh:mm:ss"` |
| `__func__` | Current function name (C99, *identifier* not macro) |
| `__STDC__` | `1` if conforming hosted implementation |
| `__STDC_VERSION__` | `199901L`, `201112L`, `201710L`, `202311L` |
| `__STDC_HOSTED__` | `1` if hosted, `0` if freestanding |

GCC/Clang add many more (`__GNUC__`, `__clang__`, `__x86_64__`, etc.). To list them all:

```bash
gcc -E -dM -x c /dev/null | sort
```

---

## 4. Type System and Promotion

C's type system is small and unforgiving. The conversion rules are formally specified but rarely fully internalized.

### 4.1 The Integer Rank Ladder

Every integer type has a **conversion rank** (C17 §6.3.1.1). Roughly:

```
_Bool < char ≤ short ≤ int ≤ long ≤ long long ≤ intmax_t
```

Ranks are unique per signedness pair: `signed int` and `unsigned int` have the same rank.

### 4.2 Integer Promotion

Anything *narrower* than `int` (or whose entire value range fits in `int`) is promoted to `int` (or `unsigned int` if `int` cannot hold the value range).

```c
unsigned char c = 255;
int x = c + 1;   // c is promoted to int (value 255), result 256, no overflow

unsigned short s = 65535;
int y = s * s;   // PROMOTED TO int: signed overflow is UB on 16-bit-short systems!
                 // on x86-64 where int is 32-bit, s promotes to int, then s*s = 4294836225,
                 // which overflows int (INT_MAX = 2147483647) → UB
```

This is one of the most surprising UB sources. Promotion converts unsigned types to *signed* int when int can hold them, and signed overflow is UB.

### 4.3 Usual Arithmetic Conversions

For binary operators, after integer promotion, both operands are converted to a common type:

1. If either is `long double` → `long double`
2. Else if either is `double` → `double`
3. Else if either is `float` → `float`
4. Otherwise apply integer promotions, then:
   - If both same type → done
   - If both same signedness → wider rank wins
   - If unsigned has rank ≥ signed → unsigned wins
   - If signed can represent all unsigned values → signed wins
   - Else → unsigned version of signed type wins

This is why mixing signed and unsigned often produces surprising results:

```c
int  i = -1;
unsigned u = 1;
if (i < u) puts("yes");   // does NOT print — i is converted to unsigned (UINT_MAX), and UINT_MAX > 1
```

### 4.4 Sized Integer Type Aliases

`<stdint.h>` provides exact-width types. Their typedef definitions vary by platform:

| Type | x86-64 Linux (LP64) | x86-64 Windows (LLP64) | i386 (ILP32) |
|:-----|:---|:---|:---|
| `int8_t` / `uint8_t` | `signed char` / `unsigned char` | same | same |
| `int16_t` / `uint16_t` | `short` / `unsigned short` | same | same |
| `int32_t` / `uint32_t` | `int` / `unsigned int` | same | same |
| `int64_t` / `uint64_t` | `long` / `unsigned long` | `long long` / `unsigned long long` | `long long` |
| `intptr_t` / `uintptr_t` | 8-byte | 8-byte | 4-byte |
| `size_t` | `unsigned long` (8 bytes) | `unsigned long long` (8 bytes) | `unsigned int` (4 bytes) |
| `ptrdiff_t` | `long` (signed, 8 bytes) | `long long` (signed, 8 bytes) | `int` (4 bytes) |

The takeaway: `long` is 8 bytes on Linux LP64 but 4 bytes on Windows LLP64. Always use sized types in cross-platform code.

### 4.5 Integer Conversion Truncation

Converting a wider integer to a narrower one truncates the high bits. For unsigned destinations, the result is well-defined: $value \bmod 2^N$ where $N$ is the destination's bit width:

$$\text{result} = \text{src} \bmod 2^{\text{width}(\text{dst})}$$

For signed destinations, if the value doesn't fit, the result is **implementation-defined** (or raises an implementation-defined signal — C17 §6.3.1.3 ¶3). In practice, every modern compiler does two's-complement wrap, but it's not standard-mandated until C23.

---

## 5. Struct Layout, Alignment, Padding

Struct layout in C is mostly mechanical, with one twist: members are laid out *in declaration order* (C17 §6.7.2.1), with padding inserted to satisfy each member's alignment, and tail padding to satisfy the struct's overall alignment.

### 5.1 Alignment per Type

Typical alignments on x86-64 SysV (`_Alignof(T)`):

| Type | Size | Alignment |
|:-----|:----:|:---------:|
| `char`, `_Bool` | 1 | 1 |
| `short` | 2 | 2 |
| `int`, `float` | 4 | 4 |
| `long`, `double`, `void *` | 8 | 8 |
| `long long` | 8 | 8 |
| `long double` | 16 | 16 |
| `__m128`, `__m256`, `__m512` | 16/32/64 | 16/32/64 |

`malloc` returns memory aligned to `alignof(max_align_t)` — typically 16 bytes on x86-64 SysV (enough for `long double` and SSE).

### 5.2 The Padding Insertion Algorithm

For each member in declaration order:
1. Round up the current offset to the member's alignment
2. Place the member, advance the offset by the member's size
3. After the last member, round the size up to the struct's overall alignment (max of all member alignments)

```c
struct bad {
    char  a;   // offset 0, size 1
               // 3 bytes padding to align int
    int   b;   // offset 4, size 4
    char  c;   // offset 8, size 1
               // 3 bytes tail padding for struct alignment of 4
};             // sizeof(struct bad) == 12

struct good {
    int   b;   // offset 0, size 4
    char  a;   // offset 4, size 1
    char  c;   // offset 5, size 1
               // 2 bytes tail padding
};             // sizeof(struct good) == 8
```

**Optimization heuristic**: order members by alignment requirement, descending. This minimizes interior padding (tail padding may still apply, but is smaller).

### 5.3 `_Alignas` and `_Alignof` (C11)

C11 added explicit alignment control:

```c
#include <stdalign.h>

_Alignas(64) char cacheline[64];   // align to 64-byte cache line
alignof(double) == _Alignof(double);

struct __attribute__((aligned(64))) hot {
    int counter;
};   // structure aligned to 64 bytes

// Cache-line-padded counter to avoid false sharing:
struct {
    _Alignas(64) atomic_int counter;
    char padding[64 - sizeof(atomic_int)];
} per_cpu_counter[NCPUS];
```

### 5.4 Bit-Fields

Bit-fields pack sub-byte values into a struct, but their layout is *implementation-defined* (C17 §6.7.2.1):

```c
struct flags {
    unsigned a : 1;
    unsigned b : 3;
    unsigned c : 4;
    unsigned d : 24;
};
```

Issues that vary across compilers:

- Allocation order within a "storage unit" (LSB-first or MSB-first)
- Whether a bit-field can straddle storage-unit boundaries
- Whether unnamed `:0` actually forces alignment
- Sign-extension behavior of plain `int` bit-fields (implementation-defined whether signed or unsigned)

For wire protocols, **never use bit-fields**. Use explicit shifts and masks; bit-fields' layout is not portable and not even guaranteed across GCC versions.

### 5.5 Flexible Array Members (C99)

A struct's last member may be an array of unspecified length:

```c
struct buf {
    size_t len;
    char data[];   // flexible array member (FAM)
};

struct buf *b = malloc(sizeof(*b) + 1024);
b->len = 1024;
// b->data is now usable as char[1024]
```

The FAM contributes nothing to `sizeof(struct buf)`. Pre-C99 code used `data[1]` ("struct hack") as a poor substitute, which formally invokes UB by indexing past the array bound.

### 5.6 Cache-Line Awareness

A modern x86-64 CPU's cache line is 64 bytes. Two threads writing to fields in the same cache line cause **false sharing** — every write invalidates the other CPU's cache, costing 50-200ns per round-trip:

```c
struct counters {
    atomic_long a;     // both fit in same 64B line
    atomic_long b;
};
// Thread 1 increments a; Thread 2 increments b — bouncing the line forever.
```

The fix is to pad each hot field to its own cache line (see §5.3). Tools like `perf c2c` and `intel-pcm` detect false sharing in production.

---

## 6. The Strict Aliasing Rule and `restrict`

The strict aliasing rule (C17 §6.5 ¶7) restricts how an object's stored value may be accessed. Violating it produces UB and miscompilation at `-O2`.

### 6.1 The Formal Rule

An object's stored value may be accessed only through an lvalue of:

- A **compatible** type (e.g., the declared type itself, or `signed`/`unsigned` variants)
- A type that **agrees in qualification** modulo `const`/`volatile`
- An aggregate or union type that **contains** one of the above
- A **character type** (`char`, `signed char`, `unsigned char`)

`char *` is the universal alias — it can read any object's bytes. But `int *` can only access `int` (or `unsigned int`) objects.

```c
int x = 0x41424344;
char *cp = (char *)&x;    // OK — char aliases anything
short *sp = (short *)&x;  // UB to read *sp — short does not alias int
float *fp = (float *)&x;  // UB to read *fp
```

### 6.2 Type-Punning Mechanisms

Sometimes you genuinely need to reinterpret bytes. The legal mechanisms:

**1. `memcpy`** (always portable, optimized to a load by every modern compiler):

```c
float f = 1.5f;
uint32_t u;
memcpy(&u, &f, sizeof(u));   // reinterpret f's bits as uint32_t
```

**2. Union type-punning** (UB pre-C99, defined in C99 with a footnote, fully defined in C11 §6.5.2.3 footnote 95):

```c
union {
    float f;
    uint32_t u;
} pun = { .f = 1.5f };
uint32_t bits = pun.u;   // C99+: defined; C++ remains UB
```

**3. `char *` aliasing** — read/write byte-by-byte:

```c
unsigned char *p = (unsigned char *)&f;
uint32_t u = (p[0]) | (p[1] << 8) | (p[2] << 16) | ((uint32_t)p[3] << 24);
```

**4. `__attribute__((may_alias))`** (GCC/Clang escape hatch):

```c
typedef int __attribute__((may_alias)) int_alias_t;
int_alias_t *aliasing = (int_alias_t *)&f;   // explicitly opts out of strict aliasing
```

### 6.3 `restrict` (C99)

The `restrict` qualifier (C17 §6.7.3.1) is a *promise* by the programmer that, during the pointer's lifetime, all accesses to the pointed-to object go through that pointer (or pointers derived from it). Violating the promise is UB.

```c
void daxpy(size_t n, double a,
           const double *restrict x,
           double *restrict y) {
    for (size_t i = 0; i < n; i++) y[i] += a * x[i];
}
```

Without `restrict`, the compiler must assume `x` and `y` may overlap, which prevents:

- Register hoisting of `y[i]` reads
- SIMD vectorization (loads and stores might race)
- Loop interchange / unrolling reorderings

With `restrict`, the compiler emits AVX vector loads and stores. The `memcpy` standard signature uses `restrict` for exactly this reason:

```c
void *memcpy(void *restrict dest, const void *restrict src, size_t n);
```

`memmove` does *not* use `restrict` — it must handle overlapping ranges, accepting slower code.

---

## 7. Undefined Behavior Taxonomy

The standard lists ~200 distinct undefined behaviors (C17 Annex J.2). The most operationally important are below, with ISO C citations and remediation.

### 7.1 Signed Integer Overflow

C17 §6.5 ¶5: "If during the evaluation of an expression the result is not mathematically defined or not in the range of representable values for its type, the behavior is undefined."

```c
int x = INT_MAX + 1;   // UB
int y = -INT_MIN;      // UB (INT_MIN's negation overflows)
```

**Remediation**:
- `-fwrapv` makes signed overflow wrap (two's-complement); inhibits some optimizations
- `-ftrapv` traps on overflow at runtime
- `-fsanitize=signed-integer-overflow` runtime detection
- Use `__builtin_add_overflow`, `__builtin_mul_overflow` for checked arithmetic
- C23: `<stdckdint.h>` provides `ckd_add`, `ckd_sub`, `ckd_mul`

### 7.2 Shift by ≥ Width or Negative

C17 §6.5.7 ¶3: "If the value of the right operand is negative or is greater than or equal to the width of the promoted left operand, the behavior is undefined."

```c
uint32_t x = 1U << 32;   // UB — x is 32 bits, shift count is 32
int      y = 1 << 31;    // UB on platforms where int is 32 bits (overflow into sign bit)
int      z = -1 >> 1;    // implementation-defined (not UB), typically arithmetic shift
```

**Remediation**: always use `1U << n` (unsigned), and assert `n < 32` (or 64).

### 7.3 NULL Pointer Dereference

```c
int *p = NULL;
*p = 5;   // UB
```

In practice on Linux/macOS, dereferencing NULL faults via the unmapped zero page, but the *standard* makes no such guarantee. The compiler may eliminate code based on the assumption that NULL is never dereferenced. For instance:

```c
int *p = lookup(key);
*p = 5;
if (p == NULL) abort();   // dead code at -O2 — compiler reasons "if *p succeeded, p was not NULL"
```

This is the "time-travel UB" pattern: the compiler reorders or eliminates the NULL check based on the dereference.

### 7.4 Dangling Pointer

Reading or writing through a pointer to freed memory or an out-of-scope automatic variable is UB.

```c
int *escape(void) {
    int x = 42;
    return &x;   // UB to dereference the returned pointer
}
```

**Detection**: AddressSanitizer marks freed memory as poisoned; Valgrind tracks use-after-free.

### 7.5 Modifying String Literals

```c
char *s = "hello";
s[0] = 'H';   // UB — string literals are in .rodata
```

Most platforms place string literals in a read-only segment, so the write SIGSEGV's at runtime. Use `char s[] = "hello";` for a writable copy.

### 7.6 Unsequenced Modifications

```c
i = i++;        // UB
a[i] = i++;     // UB
f(i++, i++);    // UB — function args indeterminately sequenced, but each arg's side effect on i is unsequenced relative to the other
```

### 7.7 Strict Aliasing Violation

See §6. Reading an object through an incompatible-type pointer is UB.

### 7.8 Out-of-Bounds Access

Past the end (or before the start) of an array. Forming a pointer one-past-the-end is allowed (`&arr[N]` for `arr[N]`), but dereferencing it is UB.

```c
int a[10];
int *p = &a[10];   // OK — one past the end
int x = *p;         // UB
int y = a[10];      // UB
```

### 7.9 Divide by Zero

```c
int x = 1 / 0;        // UB
double y = 1.0 / 0.0;  // implementation-defined: typically returns +inf with IEEE 754
```

### 7.10 Modifying a `const`-Qualified Object

```c
const int x = 5;
int *p = (int *)&x;   // cast strips const
*p = 10;               // UB to modify if x was actually declared const
```

The compiler may have placed `x` in `.rodata`, or constant-propagated `5` everywhere it's read.

### 7.11 Inactive Union Member (C11 Type-Punning)

C99 introduced an explicit allowance for union type-punning (footnote 82, then promoted to body in C11 §6.5.2.3). Reading from a non-active member is *defined* in C11+ — the bytes are reinterpreted as the destination type. C++ has not adopted this; in C++ it remains UB.

---

## 8. The Memory Allocator

`malloc` is the most frequently called runtime function in most programs. Understanding its internals explains performance, fragmentation, and security.

### 8.1 Allocator Implementations

| Allocator | Used By | Notable Features |
|:----------|:--------|:-----------------|
| **dlmalloc** | original BSDs, embedded | Single-arena, simple, classic |
| **ptmalloc / glibc malloc** | Linux glibc | dlmalloc + per-thread arenas |
| **jemalloc** | FreeBSD, Firefox, Redis, Rust default | Size-class arenas, low fragmentation |
| **tcmalloc** | Google, Chrome | Per-thread caches, central heap |
| **mimalloc** | Microsoft, default in some Rust profiles | Free-list sharding, security focus |
| **scudo** | Android, Fuchsia | Hardened allocator with quarantine |

### 8.2 ptmalloc (glibc) Architecture

```
       Process
          ├── Main arena (uses brk for heap growth)
          ├── Arena 1 (uses mmap chunks)
          ├── Arena 2
          └── ...
              Each arena has:
              ├── Fastbins (size 16-80 bytes, LIFO)
              ├── Smallbins (size 16-512 bytes)
              ├── Largebins (> 512 bytes, sorted)
              ├── Unsorted bin (transient)
              ├── Top chunk (wilderness)
              └── tcache (per-thread cache, 7 entries × 64 sizes)
```

The number of arenas is bounded by `M_ARENA_MAX` (env: `MALLOC_ARENA_MAX`), defaulting to `8 * num_cpus`. Threads stick to an arena (via thread-local pointer) but can fall back to others under contention.

### 8.3 brk vs mmap Heuristic

Small allocations come from the heap (extended via `brk`/`sbrk`). Large allocations bypass the heap and go directly to `mmap`. The threshold is `M_MMAP_THRESHOLD` (env: `MALLOC_MMAP_THRESHOLD_`), default 128 KB but adaptive:

- Above threshold → `mmap` an anonymous region; `free` calls `munmap`
- Below threshold → split a free chunk from a bin or extend the heap via `brk`

`mmap` allocations are returned to the OS on `free`. Heap allocations may not be — `free` puts them in a free list, and only the *top chunk* shrinks the heap (via `sbrk` with a negative argument). This is why long-running C processes' RSS grows: a single live chunk near the top prevents heap shrinkage.

### 8.4 Fastbins, Smallbins, Largebins, Unsorted Bin

- **Fastbins**: 10 bins for sizes 16, 24, 32, ..., 80 bytes. LIFO — last freed, first allocated. No coalescing (adjacent free chunks are *not* merged). Fast but fragmenting.
- **Smallbins**: 62 bins, one per 16-byte size class up to 512 bytes. Doubly linked, FIFO.
- **Unsorted bin**: a temporary buffer for freed chunks before they're sorted into smallbins/largebins.
- **Largebins**: 63 bins for sizes > 512 bytes, each holding a *range* of sizes, sorted by size.
- **Top chunk** (wilderness): the topmost free region of the heap; used when no bin satisfies the request.

### 8.5 tcache (since glibc 2.26)

A per-thread cache of free chunks, 64 size classes × 7 entries. Allocations check tcache first, avoiding arena lock contention. The price: tcache poisoning attacks where a freed chunk's `next` pointer is overwritten to point at attacker-controlled memory, returning that pointer from the next `malloc` of the same size.

### 8.6 Tunables

`mallopt(3)` and `MALLOC_*` env vars:

| Tunable | Purpose |
|:--------|:--------|
| `M_MMAP_THRESHOLD` | Above → mmap; below → heap |
| `M_MMAP_MAX` | Max simultaneous mmap regions |
| `M_TRIM_THRESHOLD` | Free heap space before sbrk(-) |
| `M_ARENA_MAX` | Max arenas |
| `M_PERTURB` | Fill freed memory with byte (debug) |
| `MALLOC_CHECK_` | Detect simple heap corruption |

### 8.7 Security Hardening

Modern glibc adds:

- Tcache key: detects double-free of tcached chunk
- Fastbin double-free detection
- Chunk size sanity checks
- Pointer mangling (XOR with random key) to make heap pointers harder to forge
- `glibc.malloc.check=3` for paranoid mode

---

## 9. Inline Assembly and Compiler Intrinsics

When you need control the C abstract machine doesn't expose — atomic operations, vector instructions, special-purpose registers — drop into asm.

### 9.1 Basic Inline Assembly

```c
asm("nop");
asm volatile ("rdtsc");   // volatile: don't reorder, don't dead-code-eliminate
```

Basic inline asm is a black box to the compiler; it has no idea what registers are touched. Use sparingly — extended asm is almost always better.

### 9.2 Extended Inline Assembly (GCC/Clang)

```c
uint64_t rdtsc(void) {
    uint32_t lo, hi;
    asm volatile ("rdtsc" : "=a"(lo), "=d"(hi));
    return ((uint64_t)hi << 32) | lo;
}
```

Syntax: `asm [volatile] ( template : outputs : inputs : clobbers )`. Constraint codes:

| Code | Meaning |
|:-----|:--------|
| `"=r"` | Output to any general register |
| `"=a"` | Output to RAX/EAX |
| `"=d"` | Output to RDX/EDX |
| `"=m"` | Output to memory |
| `"=&r"` | Earlyclobber output (modified before all inputs are consumed) |
| `"r"` | Input from any general register |
| `"m"` | Input from memory |
| `"i"` | Input is an immediate |
| `"+r"` | In-out register |
| `"memory"` clobber | Tells GCC the asm reads/writes arbitrary memory |
| `"cc"` clobber | Condition codes (flags) modified |

Example with multiple constraints — atomic CAS:

```c
int cas(int *p, int old, int new) {
    int prev;
    asm volatile (
        "lock cmpxchgl %2, %1"
        : "=a"(prev), "+m"(*p)
        : "r"(new), "0"(old)
        : "cc"
    );
    return prev;
}
```

### 9.3 Intel Intrinsics

`<immintrin.h>` exposes SIMD vector instructions as C functions. Each intrinsic maps to one machine instruction:

```c
#include <immintrin.h>

void add_vec(float *a, const float *b, size_t n) {
    for (size_t i = 0; i < n; i += 8) {
        __m256 va = _mm256_loadu_ps(&a[i]);
        __m256 vb = _mm256_loadu_ps(&b[i]);
        __m256 vc = _mm256_add_ps(va, vb);
        _mm256_storeu_ps(&a[i], vc);
    }
}
```

| Intrinsic Family | ISA | Width |
|:-----------------|:----|:------|
| `_mm_*` | SSE/SSE2/SSE3/SSE4 | 128-bit |
| `_mm256_*` | AVX/AVX2 | 256-bit |
| `_mm512_*` | AVX-512 | 512-bit |
| `_pdep_u64` / `_pext_u64` | BMI2 | scalar |
| `_lzcnt_u64` | LZCNT | scalar |

### 9.4 GCC `__builtin_*` Family

```c
// Branch-prediction hints
if (__builtin_expect(x == 0, 0)) abort();   // tells compiler the branch is unlikely

// Bit-counting
int leading  = __builtin_clz(x);    // count leading zeros
int trailing = __builtin_ctz(x);    // count trailing zeros
int bits     = __builtin_popcount(x);

// Endianness
uint32_t swapped = __builtin_bswap32(x);

// Memory hints
__builtin_prefetch(addr, 0 /*read*/, 3 /*high locality*/);

// Unreachable
if (impossible) __builtin_unreachable();   // says: this point is never reached

// Atomic operations (also C11 stdatomic.h)
int v = __atomic_load_n(&x, __ATOMIC_ACQUIRE);
__atomic_store_n(&x, 1, __ATOMIC_RELEASE);
int old = __atomic_fetch_add(&x, 1, __ATOMIC_RELAXED);
```

`__builtin_expect` informs the layout pass; the unlikely path goes to a cold section, improving I-cache locality. `__builtin_unreachable` enables UB-based dead-code elimination on the impossible path — use carefully; if the path *is* reached, you're in UB territory.

---

## 10. The System V AMD64 ABI

The System V AMD64 ABI defines how C code interoperates on Linux/macOS x86-64. (Windows x64 uses a different convention.)

### 10.1 Argument Registers

Integer/pointer arguments use these registers, in order:

| Arg # | Integer | Float (xmm) |
|:-----:|:-------:|:-----------:|
| 1 | RDI | XMM0 |
| 2 | RSI | XMM1 |
| 3 | RDX | XMM2 |
| 4 | RCX | XMM3 |
| 5 | R8  | XMM4 |
| 6 | R9  | XMM5 |
| 7+ | (stack) | XMM6, XMM7, then stack |

Mnemonic: "Diane's Silk Dress Costs $8.99" → DI, SI, D, C, 8, 9.

### 10.2 Return Values

| Type | Register |
|:-----|:---------|
| Integer/pointer up to 64 bits | RAX |
| Integer 65-128 bits | RAX (low), RDX (high) |
| Float/double | XMM0 |
| Two doubles (struct) | XMM0, XMM1 |
| Struct > 16 bytes | Caller passes hidden pointer in RDI; struct written there; RAX = same pointer |

### 10.3 Caller-Saved vs Callee-Saved

| Caller-saved (volatile) | Callee-saved (preserved) |
|:------------------------|:-------------------------|
| RAX, RCX, RDX | RBX, RBP, R12, R13, R14, R15 |
| RSI, RDI, R8-R11 | (and RSP) |
| XMM0-XMM15 | (no callee-saved XMMs in SysV) |

Caller must save these around a call if it needs them; callee must preserve callee-saved across its execution.

### 10.4 Stack Alignment

The stack must be 16-byte aligned at the point of a `CALL` instruction (so the callee sees `(RSP+8) % 16 == 0` upon entry, since `CALL` pushed 8 bytes). With AVX, certain operations require 32-byte alignment, but the ABI mandates only 16.

### 10.5 The Red Zone

The 128 bytes below RSP is reserved for the current function's use without adjusting RSP — provided the function makes no calls (a "leaf" function). The kernel does not stomp on the red zone via signals (in user space). Kernels disable the red zone (`-mno-red-zone`) because interrupt handlers run on the same stack.

### 10.6 Eightbyte Classification (Struct Passing)

To decide how a struct is passed, the ABI classifies each 8-byte chunk:

- **INTEGER**: an integer/pointer field straddles this 8-byte
- **SSE**: a float/double straddles this 8-byte
- **MEMORY**: any field crosses an 8-byte boundary, or the struct is > 16 bytes
- Combination rules: INTEGER + SSE → INTEGER; etc.

A struct of two ints fits in two INTEGER eightbytes → passed in two registers (RDI, RSI). A 24-byte struct → MEMORY → passed via stack.

### 10.7 Variadic Functions

For variadic calls, RAX holds the number of XMM registers used (0-8). This lets the callee know which xmm registers to spill to the va_list area. This is why a variadic function compiled without `<stdarg.h>` may misbehave even if the call types match.

### 10.8 Windows x64 ABI Differences

A common cross-platform footgun:

| Aspect | SysV (Linux/macOS) | Windows x64 |
|:-------|:-------------------|:------------|
| Args | RDI, RSI, RDX, RCX, R8, R9 | RCX, RDX, R8, R9 |
| Float args | XMM0-7 | XMM0-3 (interleaved with int args) |
| Shadow space | None | 32 bytes on stack for callee |
| Red zone | 128 bytes | None |
| Stack alignment | 16 bytes | 16 bytes |
| Callee-saved XMM | None | XMM6-XMM15 |

---

## 11. The Linker

The linker (ld, gold, lld, mold) takes object files (`.o`) and shared/static libraries and produces an executable or library by *resolving symbols* and applying *relocations*.

### 11.1 Symbol Resolution Algorithm

The linker walks input files in command-line order, maintaining:

- A set of *defined* symbols (provided by some object)
- A set of *undefined* symbols (referenced but not yet provided)

For each input:
- Object files (`.o`): always linked; their definitions resolve undefineds; their references add to undefineds
- Static archives (`.a`): linked *only if* the archive provides at least one currently-undefined symbol

This is why archive order matters: `cc main.o -lfoo -lbar` searches `libfoo.a`, then `libbar.a` — if `libbar.a` references `libfoo.a`, the linker won't go back to `libfoo.a` unless you write `-lfoo -lbar -lfoo` or use `--start-group`/`--end-group`.

### 11.2 Static vs Dynamic Linking

| Aspect | Static (.a) | Dynamic (.so) |
|:-------|:------------|:--------------|
| Resolution time | Link time | Load time |
| Binary size | Larger (libs included) | Smaller |
| Memory share | No (each process has its own) | Yes (shared across processes) |
| Update | Recompile | Replace .so |
| Symbol search | Linker walks .a | Dynamic loader walks DT_NEEDED |

### 11.3 PLT and GOT

For shared libraries, calls to dynamically-linked functions go through:

- **PLT** (Procedure Linkage Table): a stub for each external function. First call resolves the address; subsequent calls jump directly.
- **GOT** (Global Offset Table): holds the actual function addresses; modified by the dynamic linker on first call (lazy binding) or at load time (`-Wl,-z,now`).

```
caller -> PLT stub -> jmp *GOT[n]
                       │
                       ├─ first call: GOT[n] points to resolver
                       └─ later calls: GOT[n] points to real function
```

### 11.4 Hardening: RELRO, NOW, NX

| Flag | Purpose |
|:-----|:--------|
| `-Wl,-z,relro` | Mark GOT read-only after relocation (defends against GOT overwrite) |
| `-Wl,-z,now` | Resolve all symbols at load time (full RELRO) |
| `-Wl,-z,noexecstack` | Mark stack non-executable (NX) |
| `-fstack-protector-strong` | Stack canaries (compiler-side) |
| `-fPIE -pie` | Position-independent executable (ASLR) |
| `-D_FORTIFY_SOURCE=2` | Source-level overflow checking via libc wrappers |

### 11.5 Link-Time Optimization (LTO)

`-flto` makes the compiler emit bitcode (LLVM IR or GIMPLE) instead of machine code. The linker invokes the optimizer across the whole program:

- Cross-TU inlining
- Cross-TU dead-code elimination
- Cross-TU constant propagation
- Devirtualization (more relevant for C++)

Tradeoff: link time grows substantially. ThinLTO (LLVM) parallelizes across TUs to mitigate.

### 11.6 Static Archives and Shared Libraries

```bash
# Static archive
ar rcs libfoo.a foo.o bar.o
ar t libfoo.a       # list contents
ar x libfoo.a       # extract

# Shared library
gcc -fPIC -c foo.c bar.c
gcc -shared -Wl,-soname,libfoo.so.1 -o libfoo.so.1.0 foo.o bar.o
ln -s libfoo.so.1.0 libfoo.so.1
ln -s libfoo.so.1 libfoo.so
```

The three-level naming for shared libraries:

- **linker name**: `libfoo.so` (used by `-lfoo`)
- **soname**: `libfoo.so.1` (encoded in DT_SONAME, used by dynamic loader)
- **real name**: `libfoo.so.1.0` (the actual file)

Soname versioning lets you ship `libfoo.so.1.5` to replace `libfoo.so.1.0` without recompilation.

### 11.7 Linker Speed Comparison

| Linker | Project | Speed (relative) |
|:-------|:--------|:-----------------|
| ld (GNU bfd) | Original GNU | 1× (baseline, slow) |
| gold | Google, in binutils | 3-5× faster |
| lld | LLVM project | 5-10× faster |
| mold | Rui Ueyama | 10-100× faster (incremental) |

For a 50 MB Chromium link, ld takes 60s, lld takes 8s, mold takes 1.5s. Use `-fuse-ld=mold` to switch.

---

## 12. ELF Format

ELF (Executable and Linkable Format) is the file format on Linux and most modern Unix systems. (macOS uses Mach-O.)

### 12.1 ELF Structure: Sections vs Segments

ELF has *two views* of the file:

- **Section view** (linker view): for static linking; described by section headers
- **Segment view** (loader view): for execution; described by program headers

A segment may span multiple sections (e.g., the `R+X` text segment holds `.text`, `.init`, `.fini`, `.rodata`).

### 12.2 Common Sections

| Section | Contents |
|:--------|:---------|
| `.text` | Executable code |
| `.rodata` | Read-only data (string literals, const globals) |
| `.data` | Initialized read-write data |
| `.bss` | Zero-initialized data (no file space — just `MemSize`) |
| `.symtab` | Symbol table (linker; stripped from production binaries) |
| `.strtab` | String table for symbol names |
| `.dynsym` | Dynamic symbol table (used at runtime) |
| `.dynstr` | Dynamic string table |
| `.dynamic` | Dynamic linking metadata (DT_NEEDED, DT_RPATH, DT_SONAME, ...) |
| `.got` | Global Offset Table |
| `.got.plt` | GOT entries used by PLT |
| `.plt` | Procedure Linkage Table |
| `.init_array` | Constructors run before `main` |
| `.fini_array` | Destructors run after `exit` |
| `.eh_frame` | DWARF unwind info (exception handling, backtraces) |
| `.note.ABI-tag` | Indicates required kernel/ABI |
| `.gnu.hash` | Faster hash for symbol lookup than `.hash` |

### 12.3 Common Segments

| Segment Type | Permissions | Contains |
|:-------------|:-----------:|:---------|
| PT_LOAD (text) | R+X | .text, .init, .fini, .rodata, .eh_frame |
| PT_LOAD (data) | RW | .data, .bss |
| PT_DYNAMIC | RW | .dynamic |
| PT_INTERP | R | Path to dynamic loader |
| PT_GNU_RELRO | R after relocation | .got, .init_array (with -z relro) |
| PT_GNU_STACK | RW (or RWX) | Marker for stack permissions |

### 12.4 The Dynamic Linker

The dynamic loader (`/lib64/ld-linux-x86-64.so.2` on Linux x86-64) is itself an ELF binary, named in the executable's PT_INTERP. The kernel maps the binary into memory and jumps to the loader, which:

1. Reads `DT_NEEDED` entries → loads each `.so`
2. Walks the search path: LD_LIBRARY_PATH → DT_RPATH → DT_RUNPATH → /etc/ld.so.cache → default paths
3. Performs relocations (R_X86_64_GLOB_DAT, R_X86_64_JUMP_SLOT, etc.)
4. Calls `.init_array` for each .so, then for the executable
5. Jumps to `_start` → `__libc_start_main` → `main`

### 12.5 RPATH vs RUNPATH

Both encode library search paths in the binary, but:

- **DT_RPATH**: searched *before* LD_LIBRARY_PATH (older, deprecated)
- **DT_RUNPATH**: searched *after* LD_LIBRARY_PATH (newer, since glibc 2.2)

```bash
gcc -Wl,-rpath,'$ORIGIN/../lib' main.c   # DT_RPATH/RUNPATH with $ORIGIN expansion
```

### 12.6 Hardening Flags Summary

```bash
gcc -fPIE -pie \                                  # ASLR for executable
    -fstack-protector-strong \                    # canaries
    -fstack-clash-protection \                    # stack-clash mitigation
    -D_FORTIFY_SOURCE=2 -O2 \                     # libc overflow checks
    -fcf-protection=full \                        # CET (Intel control-flow)
    -Wl,-z,relro,-z,now \                         # full RELRO
    -Wl,-z,noexecstack \                          # NX stack
    -Wl,-z,separate-code \                        # separate text/rodata mapping
    main.c -o main
```

---

## 13. Signals and Async-Signal-Safe Functions

A signal handler interrupts the program at an arbitrary instruction boundary. Calling a non-async-signal-safe function from a handler is *undefined behavior* (POSIX, not C standard, but observed across implementations).

### 13.1 Async-Signal-Safe Functions

POSIX defines a list. The shortlist:

| Safe | Why |
|:-----|:----|
| `_exit`, `_Exit` | No stdio buffers to flush |
| `write`, `read` | Direct syscall, no internal locks |
| `sigaction`, `sigprocmask` | Async-safe by design |
| `kill`, `raise` | Async-safe |
| `time`, `clock_gettime` | Read-only timers |
| `dup`, `dup2`, `fcntl`, `close`, `open` (without O_CREAT) | Direct fd ops |

| Unsafe | Why |
|:-------|:----|
| `printf`, `fprintf`, `puts` | Hold stdio locks; can deadlock if handler interrupts a printing thread |
| `malloc`, `free` | Hold allocator locks |
| `pthread_*` | Hold pthread internals |
| `exit` (lowercase) | Runs atexit handlers, flushes buffers |
| `localtime`, `strftime` | Use static buffers; not reentrant |

### 13.2 Signal-Safe Communication

The portable way to communicate between handler and main code is `volatile sig_atomic_t`:

```c
#include <signal.h>

static volatile sig_atomic_t shutdown_requested = 0;

void handle_term(int sig) { shutdown_requested = 1; }

int main(void) {
    struct sigaction sa = { .sa_handler = handle_term };
    sigemptyset(&sa.sa_mask);
    sigaction(SIGTERM, &sa, NULL);

    while (!shutdown_requested) {
        // do work
    }
    cleanup();
    return 0;
}
```

`sig_atomic_t` is the only type the C standard guarantees can be read/written atomically with respect to signals. Using anything else (even `int`) is technically UB, though in practice modern CPUs make int load/store atomic.

For richer signal-to-main-thread communication, use:

- `signalfd(2)` (Linux): convert signals to readable file descriptors
- `pipe(2)` self-pipe trick: handler writes one byte to a pipe, main loop selects on it
- `eventfd(2)` (Linux): like pipe but more efficient

### 13.3 sigaction vs signal

`signal()` has historical SysV vs BSD differences (does it auto-reset? does it interrupt syscalls?). `sigaction()` is portable and explicit:

```c
struct sigaction sa = {0};
sa.sa_handler = handler;
sigemptyset(&sa.sa_mask);
sa.sa_flags = SA_RESTART;   // restart interrupted syscalls
sigaction(SIGINT, &sa, NULL);
```

`SA_RESTART` makes interrupted syscalls (like `read` blocked on a fifo) restart automatically. Without it, they return -1 with `errno = EINTR` and you must retry manually.

---

## 14. Compilation Pipeline

```
.c source
   │
   │ ├─ trigraph replacement (deprecated, removed C23)
   │ ├─ line splicing (\\\n)
   │ └─ tokenization
   ▼
preprocessing (cpp / cc1 -E)     # macros, #include, #if
   ▼ translation unit
parser → AST                      # syntactic analysis
   ▼
semantic analyzer / type checker  # name resolution, type inference
   ▼
LLVM IR / GIMPLE                  # mid-level intermediate representation
   ▼
optimization passes               # mem2reg, GVN, LICM, vectorize, inline, ...
   ▼
backend: instruction selection    # IR → MachineIR
        register allocation       # graph coloring or linear scan
        instruction scheduling    # reorder for pipeline
        code emission             # → .s assembly
   ▼
assembler (as)                    # .s → .o (ELF object)
   ▼
linker (ld/gold/lld/mold)         # → executable / .so
```

### 14.1 GCC Stages

| Flag | Stops after | Output |
|:-----|:-----------:|:-------|
| `-E` | Preprocessing | `.i` (preprocessed source) |
| `-S` | Compilation | `.s` (assembly) |
| `-c` | Assembly | `.o` (object file) |
| (none) | Linking | executable |
| `-x c` | Force C language regardless of extension | |

### 14.2 AT&T vs Intel Asm Syntax

GCC defaults to AT&T (`source, dest`); Intel uses (`dest, source`):

```asm
# AT&T (GCC default)
movq %rax, %rbx       ; rbx = rax
addq $5, %rax         ; rax += 5
movq (%rdi, %rsi, 4), %rax   ; rax = *(rdi + 4*rsi)

; Intel
mov rbx, rax
add rax, 5
mov rax, [rdi + rsi*4]
```

`gcc -masm=intel` switches to Intel syntax. Use `-fverbose-asm` to add comments showing source variables.

### 14.3 Optimization Levels

| Level | What's Enabled |
|:------|:---------------|
| `-O0` | None. Each variable lives in memory. Debugger-friendly. |
| `-O1` | Basic: dead code elim, common subexpr, basic block reordering, register allocation. ~30% speedup typical. |
| `-O2` | Inlining, loop unrolling (small), constant prop, GVN, partial redundancy elimination, schedule. ~50% speedup. Default for most builds. |
| `-O3` | Adds: aggressive vectorization, larger inlining heuristic, function cloning, ipa-cp. May *increase* code size. |
| `-Os` | `-O2` minus size-bloating optimizations. |
| `-Oz` | (Clang) Aggressively shrink size. |
| `-Ofast` | `-O3` plus `-ffast-math` (relax IEEE 754) plus other "unsafe" relaxations. Breaks NaN handling. |
| `-Og` | `-O1` minus optimizations that hurt debugging. |

### 14.4 Profile-Guided Optimization (PGO)

Three-step process:

```bash
# 1. Compile with instrumentation
gcc -O2 -fprofile-generate -o app app.c

# 2. Run on representative workload
./app < typical_input.txt

# 3. Recompile using the profile
gcc -O2 -fprofile-use -o app app.c
```

Typical gains: 5-15% speedup. Modern variants (LLVM IRPGO, AutoFDO) reduce the workflow burden.

---

## 15. Optimization Examples

### 15.1 Loop Unrolling

Source:

```c
for (int i = 0; i < 4; i++) sum += a[i];
```

`-O3` unrolls:

```asm
movl    (%rdi), %eax
addl    4(%rdi), %eax
addl    8(%rdi), %eax
addl    12(%rdi), %eax
```

Eliminates the loop counter, comparison, branch — pure straight-line code.

### 15.2 Auto-Vectorization

Source:

```c
void scale(float *a, float k, size_t n) {
    for (size_t i = 0; i < n; i++) a[i] *= k;
}
```

`-O3 -mavx2`:

```asm
vbroadcastss %xmm0, %ymm0      ; replicate k into all 8 lanes
.L_loop:
    vmulps (%rdi), %ymm0, %ymm1 ; 8 floats × scalar
    vmovups %ymm1, (%rdi)
    addq $32, %rdi
    cmpq %rdi, %rsi
    jne .L_loop
```

8× speedup on the hot loop. Add `restrict` to enable this without runtime alias checks.

### 15.3 Dead Code Elimination

```c
int f(int x) {
    if (x > 0) return 1;
    return 1;
}
```

Compiles to `mov eax, 1; ret` — the comparison is gone.

### 15.4 Constant Folding and Propagation

```c
int x = 3, y = 4;
int z = x * y;
return z;
```

Compiles to `mov eax, 12; ret` — all arithmetic at compile time.

### 15.5 Common Subexpression Elimination

```c
int z1 = (a + b) * c;
int z2 = (a + b) * d;
```

The compiler hoists `t = a + b` and computes `z1 = t*c, z2 = t*d`.

### 15.6 Function Inlining Heuristics

The compiler inlines a callee into its caller when:

- Callee is `static inline` or marked `__attribute__((always_inline))`
- Callee is small (cost-vs-benefit analysis)
- Caller is hot (PGO-informed)
- Inlining enables further optimization (constant prop into callee, devirtualization, ...)

`-finline-limit=N` tunes the heuristic; `-Winline` warns when inlining was requested but not performed.

---

## 16. Testing Strategy at Internals Depth

### 16.1 Valgrind memcheck

Valgrind runs the program under a CPU emulator (Vex IR), tracking every byte's *defined* and *addressable* state. Detects:

- Uninitialized reads (V-bits track defined-ness)
- Heap out-of-bounds (red zones around malloc)
- Use-after-free (free zones quarantined)
- Memory leaks (reachable, possibly-lost, definitely-lost)

Cost: 20-50× slowdown. False positives mostly come from custom allocators (Valgrind sees only `malloc`/`free`); suppress via Valgrind suppression files. Valgrind also models thread synchronization (`drd`, `helgrind` tools).

### 16.2 AddressSanitizer (ASan)

Compile with `-fsanitize=address`. Adds:

- Shadow memory: 1 byte of metadata for every 8 bytes of program memory (one bit per byte, plus a marker for poisoned-byte counts)
- Each load/store is instrumented to check the shadow byte
- `malloc` returns chunks surrounded by *redzones* (poisoned shadow bytes)
- `free` quarantines memory (delayed reuse) for use-after-free detection

Performance: 2× slowdown, 3× memory. Detects same bugs as Valgrind but ~10× faster.

The shadow-memory algorithm:

$$\text{shadow}(\text{addr}) = (\text{addr} \gg 3) + \text{offset}$$

where `offset` is `0x7fff8000` on x86-64 Linux. To check an 8-byte load: read shadow byte, ensure it's zero. For smaller accesses: shadow byte = N means "first N bytes valid, rest poisoned."

### 16.3 UndefinedBehaviorSanitizer (UBSan)

`-fsanitize=undefined` instruments common UB sources:

- `signed-integer-overflow`
- `shift` (count out of range)
- `null` (NULL deref)
- `bounds` (array index out of range)
- `alignment` (misaligned pointer)
- `vptr` (C++ vtable mismatch)

Each check is a few inline instructions plus a call to a runtime stub on failure. ~5-10% slowdown.

### 16.4 Kernel ASan (kASan)

A version of ASan for the Linux kernel, using a separate shadow region. Drops the user-space red zone trick (kernel mallocs are different) and uses `quarantine_lists` for slab freed objects. Critical for finding kernel use-after-frees.

### 16.5 Fuzzing

The modern stack:

```
libFuzzer / AFL++  (engine: coverage-guided mutation)
       │
       ├── Compiled with -fsanitize=address,undefined,fuzzer
       ├── Reads bytes from stdin/buffer, calls LLVMFuzzerTestOneInput
       └── On any sanitizer report → minimize crash, save to corpus
```

Example harness:

```c
#include <stddef.h>
#include <stdint.h>

extern int parse(const uint8_t *data, size_t size);

int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
    parse(data, size);
    return 0;
}
```

Compile: `clang -fsanitize=address,undefined,fuzzer harness.c parse.c -o fuzz`.

Run: `./fuzz corpus/` — runs forever, saving new inputs that hit new code paths.

OSS-Fuzz runs hundreds of open-source projects continuously, finding ~25,000 bugs to date.

---

## 17. C and C++ Compatibility Surface

### 17.1 `extern "C"`

C++ name-mangles function names (encoding signature, namespace, etc.) for overloading and template support. C does not mangle. To call C code from C++, declare the function `extern "C"`:

```c
// header.h — usable from C and C++
#ifdef __cplusplus
extern "C" {
#endif

void c_function(int x);

#ifdef __cplusplus
}
#endif
```

### 17.2 ABI Matching

C and C++ ABIs are *mostly* compatible on the same platform. Trouble spots:

- C++ passes `bool` differently (LLVM/GCC pass it in low byte of a register, but the high bits are undefined on input)
- `enum class` (C++) vs `enum` (C): C++ guarantees underlying type is `int` unless specified
- Struct ABI: C and C++ usually agree, but C++ adds vtables to polymorphic classes

### 17.3 `_Bool` vs `bool`

C99 has `_Bool`. `<stdbool.h>` provides:

```c
typedef _Bool bool;
#define true  1
#define false 0
```

C++ has `bool` as a built-in keyword. C23 promotes `bool`, `true`, `false` to keywords as well, removing the need for `<stdbool.h>`.

### 17.4 `__cplusplus` Guard

```c
#ifdef __cplusplus
// C++ code path
#else
// C code path
#endif
```

`__cplusplus` is `199711L` (C++98), `201103L` (C++11), `201402L`, `201703L`, `202002L`.

---

## 18. Modern C Surface

### 18.1 C11 Features

**`_Generic` — type-generic macros**:

```c
#define abs(x) _Generic((x), \
    int: abs,                \
    long: labs,              \
    long long: llabs,        \
    float: fabsf,            \
    double: fabs,            \
    long double: fabsl       \
)(x)

abs(-3);     // calls abs
abs(-3.0);   // calls fabs
```

**`_Atomic` and `<stdatomic.h>`**:

```c
#include <stdatomic.h>

atomic_int counter;
atomic_fetch_add_explicit(&counter, 1, memory_order_relaxed);

int v = atomic_load_explicit(&counter, memory_order_acquire);
atomic_store_explicit(&counter, 42, memory_order_release);

// Memory orderings (weakest to strongest):
// memory_order_relaxed — atomicity only, no ordering
// memory_order_consume — (deprecated/effectively unused) data-dependency ordering
// memory_order_acquire — ordering: this load + later loads/stores
// memory_order_release — ordering: earlier loads/stores + this store
// memory_order_acq_rel — both
// memory_order_seq_cst — sequentially consistent (default)
```

**Threads (optional)**: `<threads.h>` with `thrd_create`, `mtx_lock`, `cnd_signal`. Optional in C11 — many implementations don't ship it (glibc didn't until 2.28).

**`_Static_assert`**:

```c
_Static_assert(sizeof(int) == 4, "int must be 4 bytes");
```

### 18.2 C17

C17 (formerly C18) is essentially C11 with bug fixes. No new features; just defect resolutions to the standard text.

### 18.3 C23 Highlights

**`typeof` and `typeof_unqual`**: standard versions of GCC's extension.

```c
#define swap(a, b) do { typeof(a) t = a; a = b; b = t; } while (0)
```

**`constexpr`**:

```c
constexpr int N = 100;   // true compile-time constant, usable in array bounds
int arr[N];
```

**Enum with explicit underlying type**:

```c
enum color : uint8_t { RED, GREEN, BLUE };   // sizeof(enum color) == 1
```

**Standard attributes**:

```c
[[deprecated("use new_func instead")]] void old_func(void);
[[nodiscard]] int must_check_return(void);
[[maybe_unused]] static int debug_only;
[[fallthrough]];   // in switch case
[[noreturn]] void die(void);
```

**`nullptr` and `nullptr_t`**: a typed null. `(nullptr_t)nullptr == NULL` but `nullptr` is not implicitly an int 0.

**`true`, `false`, `bool` as keywords**: no more `<stdbool.h>` needed.

**`_BitInt(N)`**: arbitrary-width integers.

```c
_BitInt(7) seven_bit;     // 7-bit signed
unsigned _BitInt(128) huge;  // 128-bit unsigned
```

**`#embed`**: include binary data at compile time.

```c
const unsigned char img[] = {
    #embed "logo.png"
};
```

**Other**: digit separators (`1'000'000`), binary literals (`0b1010`) finally standardized, `unreachable()` macro, `<stdckdint.h>` for checked arithmetic.

---

## 19. Performance Tips

### 19.1 Branch Prediction

```c
if (__builtin_expect(error_condition, 0)) goto error;
```

Modern CPUs predict branches with hardware history. Static hints (`__builtin_expect`) influence basic-block layout — the unlikely path is moved to a cold section, improving I-cache locality. PGO does this automatically with profile data.

### 19.2 Prefetching

```c
for (size_t i = 0; i < n; i++) {
    __builtin_prefetch(&a[i + 16], 0, 3);  // read, high temporal locality
    process(a[i]);
}
```

Compilers auto-prefetch simple stride patterns. Manual prefetch helps with pointer-chasing (linked lists, hash tables) where the access pattern is unpredictable from memory.

### 19.3 Alignment

```c
#include <stdalign.h>

_Alignas(64) struct hot_data {
    atomic_int counters[16];
};
```

Cache-line-align hot structures. For SIMD, align to 32 (AVX) or 64 (AVX-512) for aligned loads (`vmovaps` vs slower `vmovups`).

### 19.4 Profile-Guided Optimization

`-fprofile-generate` → run → `-fprofile-use`. Three-step. Typical gain: 5-15%. The profile informs:

- Function inlining decisions
- Basic-block layout (hot path on fall-through)
- Loop unroll factors
- Register allocation (hot variables in callee-saved registers)

### 19.5 Structure of Arrays vs Array of Structures

```c
// AoS — bad for SIMD
struct vec3 { float x, y, z; };
struct vec3 points[N];
for (size_t i = 0; i < N; i++) points[i].x += 1.0f;
// Cache loads y and z too — 67% wasted bandwidth.

// SoA — good for SIMD
struct points { float x[N], y[N], z[N]; } pts;
for (size_t i = 0; i < N; i++) pts.x[i] += 1.0f;
// Loads only x — 100% useful bandwidth, easy to vectorize.
```

### 19.6 False Sharing

Two threads writing different fields in the same cache line cause contention. Detect with `perf c2c`, fix with cache-line padding (see §5).

### 19.7 Streaming Stores

For write-only large outputs (where the data won't be read again before eviction), bypass the cache:

```c
#include <emmintrin.h>

void zero(int *p, size_t n) {
    __m128i z = _mm_setzero_si128();
    for (size_t i = 0; i + 4 <= n; i += 4)
        _mm_stream_si128((__m128i *)&p[i], z);   // non-temporal store
    _mm_sfence();   // flush write buffer
}
```

Streaming stores write directly to memory, freeing cache space for the data the program *will* re-read.

### 19.8 The `perf` + Flame Graph Workflow

```bash
# Record
perf record -F 99 -g ./app

# Print top functions
perf report

# Generate flame graph (Brendan Gregg's tool)
perf script | stackcollapse-perf.pl | flamegraph.pl > graph.svg
```

The flame graph shows time per call stack — wide bars are hot. Useful complement to `perf annotate` for line-level hotness.

---

## 20. Prerequisites

- Familiarity with K&R-level C: pointers, structs, the standard library, function pointers
- Computer organization basics: registers, cache hierarchy, virtual memory, system calls
- Number representation: two's complement, IEEE 754 floats, endianness
- Comfort reading the companion sheet `sheets/languages/c.md`
- An operating systems mental model: processes, threads, signals, scheduling
- Some assembly literacy (x86-64 preferred; ARMv8 ports of these concepts are direct)

## 21. Complexity

This is a comparative-reference deep dive — calibrated for readers who already write C and want internals. If you want a hands-on tour, start with the companion sheet first; treat this file as the *why* behind every `--help` flag, every `-O2` optimization, every UB sanitizer report. The material here is the working content of *Modern C* (Gustedt), CSAPP (Bryant & O'Hallaron), and the implementation chapters of *Linkers and Loaders* (Levine), distilled to what matters for production work.

## 22. See Also

- [c (companion sheet)](../../sheets/languages/c.md) — practical reference: types, memory functions, preprocessor recipes
- [polyglot](../../sheets/languages/polyglot.md) — survey of C against contemporaries
- [rust](../../sheets/languages/rust.md) — what C looks like with ownership/borrowing instead of UB
- [go](../../sheets/languages/go.md) — what C looks like with garbage collection and goroutines

## 23. References

- *cppreference.com — C section* (https://en.cppreference.com/w/c) — practical, accurate, up-to-date through C23
- ISO/IEC 9899:2018 (C17) and the C23 final draft (N3220) — the actual standard
- *The GNU C Library Reference Manual* — definitive on Linux's libc behavior
- Jens Gustedt, *Modern C* (free online: https://hal.inria.fr/hal-02383654) — best modern C tutorial through C17
- Bryant & O'Hallaron, *Computer Systems: A Programmer's Perspective* (CSAPP) — links C to hardware, ABI, linking
- Kernighan & Ritchie, *The C Programming Language*, 2nd ed. — the original; still relevant
- Levine, *Linkers and Loaders* — the linker book
- Brian "Beej" Hall, *Beej's Guide to C Programming* — gentle re-entry
- Linux Kernel Newbies — `kernelnewbies.org` for kernel-style C
- Agner Fog, *Optimization manuals* (https://agner.org/optimize/) — microarchitecture-level performance, calling conventions, instruction tables
- *System V ABI x86-64 supplement* (https://gitlab.com/x86-psABIs/x86-64-ABI) — the official ABI document
- *Dwarf Standard* (https://dwarfstd.org/) — for understanding `.eh_frame`, debugging info
- Drepper, *What Every Programmer Should Know About Memory* — long-form on caches, NUMA, prefetching

---

*The C abstract machine is a contract: the compiler delivers observable behavior; you deliver UB-free code. Every section above is a clause of that contract — break it and the compiler's optimizer is no longer your ally. Read it as a discipline, not a bag of tricks.*
