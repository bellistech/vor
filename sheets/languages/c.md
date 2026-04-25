# C (Programming Language)

> Low-level systems language, ISO/IEC 9899, foundation of operating systems, runtimes, and most language implementations.

## Setup

### Compilers

```bash
# GCC (GNU Compiler Collection) — Linux default
# gcc --version  -> e.g. gcc (Ubuntu 13.2.0-23ubuntu4)
# Install: apt install build-essential | dnf install gcc make | brew install gcc

# Clang (LLVM) — macOS default, also strong on Linux
# clang --version  -> e.g. Apple clang version 15.0.0
# Install: apt install clang | brew install llvm

# MSVC (cl.exe) — Windows. Requires "Developer Command Prompt for VS"
# cl.exe -nologo file.c       (similar to gcc/clang but different flag syntax)

# tcc (Tiny C Compiler) — fast, tiny, scriptable
# tcc -run hello.c   (compile + execute in one shot)

# pcc (Portable C Compiler), icx (Intel oneAPI), zig cc (Zig as C compiler)
```

### Standards and -std flag

```bash
# C89 / C90 / ANSI C — original ISO standard
gcc -std=c89 file.c          # Strict C89 (no //-comments by default)
gcc -ansi   file.c           # Equivalent to -std=c89

# C99 — bool, // comments, designated init, VLA, restrict, intptr_t, snprintf
gcc -std=c99 file.c

# C11 — _Generic, _Atomic, _Thread_local, anonymous structs/unions, aligned_alloc
gcc -std=c11 file.c

# C17 / C18 — bug-fix release of C11 (same features, fewer DRs)
gcc -std=c17 file.c

# C23 — typeof, constexpr, nullptr, true/false keywords, [[attributes]], _BitInt
gcc -std=c23 file.c          # GCC 14+ / Clang 18+
gcc -std=c2x file.c          # Older alias

# GNU dialect (default for GCC) adds extensions
gcc -std=gnu11 file.c        # C11 + GNU extensions (statement exprs, ({...}), nested fns)

# Default if -std missing: gcc 5+ uses -std=gnu17. Always set it explicitly.
```

### Hello World

```bash
# hello.c
# #include <stdio.h>
# int main(void) {
#     printf("Hello, world!\n");
#     return 0;
# }

gcc -std=c11 -Wall -Wextra -o hello hello.c
./hello                       # Hello, world!
```

## Compile/Link

### Translation Units

```bash
# A C program is one or more .c "translation units" → .o objects → linked binary

gcc -c main.c -o main.o       # Compile only — produce main.o
gcc -c util.c -o util.o       # util.o
gcc main.o util.o -o app      # Link → app
gcc main.o util.o -o app -lm  # Link with libm (math)

# One-step (same as above but no separate .o files kept):
gcc main.c util.c -o app -lm
```

### Common flags

```bash
-c              # Compile to .o, do not link
-o <file>       # Output filename
-E              # Stop after preprocessing — print to stdout
-S              # Stop after compilation — emit assembly (.s)
-I <dir>        # Add include search path
-L <dir>        # Add library search path
-l <name>       # Link libname (libname.so / libname.a)
-D NAME=VAL     # Define macro (-DDEBUG, -DBUFSZ=4096)
-U NAME         # Undefine macro
-MMD -MP        # Emit .d Makefile dependency files
-pipe           # Use pipes between cc1/as instead of temp files
-v              # Verbose (show subprocess commands)
```

### Warnings

```bash
gcc -Wall                     # Most useful warnings (NOT all)
gcc -Wall -Wextra             # More
gcc -Wall -Wextra -Werror     # Treat warnings as errors
gcc -Wpedantic                # Strict ISO C
gcc -Wshadow                  # Variable shadowing
gcc -Wconversion              # Implicit narrowing
gcc -Wsign-conversion         # Signed↔unsigned implicit
gcc -Wcast-align              # Pointer alignment changes
gcc -Wstrict-prototypes       # K&R-style declarations
gcc -Wmissing-prototypes      # Non-static fn without prior decl
gcc -Wnull-dereference        # Likely NULL deref (GCC ≥6)
gcc -Wformat=2                # printf format checking, strict
gcc -Wfloat-equal             # `==` on float
gcc -Wdouble-promotion        # float → double silently
gcc -Wundef                   # `#if FOO` when FOO undefined
```

## Preprocessor

### Includes and Macros

```bash
# #include <stdio.h>          // angle brackets — system search path (-I dirs after)
# #include "myutil.h"         // quotes — current dir first, then system

# Object-like macro
# #define PI 3.14159
# #define MAX_BUF 4096

# Function-like macro — ALWAYS parenthesize args and the whole expansion
# #define SQUARE(x) ((x) * (x))
# #define MAX(a, b) ((a) > (b) ? (a) : (b))
# #define ARRAY_LEN(a) (sizeof(a) / sizeof((a)[0]))

# Multi-line macro — backslash-continuation, do/while(0) for statement-safety
# #define LOG_ERR(fmt, ...) do {                          \
#     fprintf(stderr, "[ERR] %s:%d: " fmt "\n",           \
#             __FILE__, __LINE__, ##__VA_ARGS__);         \
# } while (0)
```

### Conditionals

```bash
# #if EXPR / #elif / #else / #endif
# #ifdef NAME      // shorthand for #if defined(NAME)
# #ifndef NAME     // shorthand for #if !defined(NAME)

# #ifdef __linux__
#     /* Linux only */
# #elif defined(__APPLE__)
#     /* macOS only */
# #elif defined(_WIN32)
#     /* Windows */
# #endif

# #if __STDC_VERSION__ >= 201112L
#     /* C11 or newer */
# #endif

# #error "message"       // halt with diagnostic
# #warning "message"     // GCC/Clang extension
```

### Stringification (#) and Token-paste (##)

```bash
# #define STR(x) #x                  // STR(hello)   -> "hello"
# #define XSTR(x) STR(x)             // expand-then-stringify
# #define VER 12
# STR(VER)   -> "VER"
# XSTR(VER)  -> "12"

# #define CAT(a, b) a##b
# #define VAR(n) CAT(var_, n)
# int VAR(3) = 0;                    // -> int var_3 = 0;
```

### X-Macros

```bash
# A list defined once, reused for enum, names, dispatch:

# #define COLORS \
#     X(RED,   "red"  ) \
#     X(GREEN, "green") \
#     X(BLUE,  "blue" )

# enum Color { 
# #define X(name, str) name,
#     COLORS
# #undef X
# };

# const char *color_name(enum Color c) {
#     switch (c) {
# #define X(name, str) case name: return str;
#         COLORS
# #undef X
#     }
#     return "?";
# }
```

### Header Guards

```bash
# Classic include guard:
# #ifndef MYLIB_FOO_H
# #define MYLIB_FOO_H
# /* declarations */
# #endif /* MYLIB_FOO_H */

# Modern (GCC, Clang, MSVC, all major):
# #pragma once
```

### Predefined Macros

```bash
__FILE__         # Source file path (string literal)
__LINE__         # Line number (int)
__func__         # Function name (C99, char[])
__DATE__         # "Mmm dd yyyy"
__TIME__         # "hh:mm:ss"
__STDC__         # 1 if conforming
__STDC_VERSION__ # 199901L / 201112L / 201710L / 202311L
__GNUC__ / __clang__ / _MSC_VER  # compiler detection
__cplusplus      # set under C++ — guard C headers with `extern "C"`
```

## Integer Types

### Standard integer types

```bash
# Signed (default for char is implementation-defined!)
char        # 1 byte; CHAR_BIT bits (almost always 8)
short       # ≥16 bits
int         # ≥16 bits (typically 32)
long        # ≥32 bits (32 on Win64, 64 on Linux/macOS x86_64)
long long   # ≥64 bits

# Unsigned
unsigned char, unsigned short, unsigned int, unsigned long, unsigned long long

# Signedness modifiers
signed char    # always signed
unsigned int   # always unsigned

# Limits — <limits.h>
INT_MIN, INT_MAX, UINT_MAX, LONG_MIN, LONG_MAX,
SCHAR_MIN, SCHAR_MAX, UCHAR_MAX, CHAR_BIT, LLONG_MIN, LLONG_MAX
```

### Fixed-width — `<stdint.h>` (C99)

```bash
int8_t   int16_t   int32_t   int64_t          # exact width signed
uint8_t  uint16_t  uint32_t  uint64_t         # exact width unsigned
int_fast8_t  int_fast32_t  ...                # at least N, fastest
int_least8_t int_least16_t ...                # at least N bits
intmax_t   uintmax_t                          # widest available

intptr_t   uintptr_t                          # holds a void* losslessly
size_t                                        # unsigned, sizeof()/strlen() return type
ptrdiff_t                                     # signed, pointer subtraction result
ssize_t                                       # POSIX, signed size

# Limits — INT8_MIN, UINT64_MAX, SIZE_MAX, PTRDIFF_MAX, INTPTR_MIN ...

# Literal suffixes — <stdint.h>
INT32_C(42)       # Right suffix for int32_t literal
UINT64_C(0xDEADBEEF)
```

### printf format macros — `<inttypes.h>`

```bash
# Don't guess %d vs %ld vs %lld — use the macros:
# printf("%" PRId32 "\n", val32);     # signed 32-bit
# printf("%" PRIu64 "\n", val64);     # unsigned 64-bit
# printf("%" PRIx32 "\n", val32);     # hex 32-bit
# printf("%zu\n", sizeof(x));         # size_t
# printf("%td\n", ptr2 - ptr1);       # ptrdiff_t
# scanf("%" SCNd64, &x);              # input version
```

## Type Conversions & Integer Promotion

### Integer promotion rules

```bash
# Operands narrower than int are promoted to int (or unsigned int) before
# arithmetic. This causes surprises:

# unsigned char a = 0xFF, b = 0x01;
# int c = a + b;            // a,b promoted to int -> c = 256, NOT 0
# printf("%d\n", c);        // 256

# Signed/unsigned mixed: signed converted to unsigned of same rank
# int i = -1;
# unsigned u = 1;
# if (i < u) puts("yes"); else puts("no");
#   // prints "no" — i becomes UINT_MAX, which is > 1
```

### Implicit narrowing

```bash
# BROKEN — silent truncation:
# uint64_t big = 0x1FFFFFFFF;
# uint32_t small = big;     // becomes 0xFFFFFFFF; data lost; -Wconversion warns

# FIXED — explicit cast + bounds check:
# if (big > UINT32_MAX) { /* error */ }
# uint32_t small = (uint32_t)big;
```

### Common conversion gotchas

```bash
# strlen returns size_t (unsigned). Never compare with -1:
# BROKEN:    if (strlen(s) - n < 0)        // always false — unsigned never < 0
# FIXED:     if ((size_t)n > strlen(s))

# char vs int from getchar — getchar returns int (so it can return EOF == -1):
# BROKEN:    char c = getchar(); if (c == EOF) ...   // can't store EOF in char
# FIXED:     int c = getchar(); if (c == EOF) ...
```

## Floating Point

### Types

```bash
float        # ≥6 decimal digits, IEEE 754 single
double       # ≥15 decimal digits, IEEE 754 double
long double  # ≥double; x86_64 Linux/macOS = 80-bit, MSVC = 64-bit

# <float.h> limits
FLT_EPSILON  # smallest x such that 1.0 + x != 1.0  (~1.19e-7)
DBL_EPSILON  # ~2.22e-16
FLT_MAX, FLT_MIN, DBL_MAX, DBL_MIN
FLT_DIG, DBL_DIG  # decimal digits of precision
FLT_RADIX         # usually 2
```

### Comparing floats

```bash
# BROKEN — never use ==:
# if (x == 0.1) ...

# FIXED — epsilon comparison:
# #include <math.h>
# int near(double a, double b, double eps) {
#     return fabs(a - b) <= eps * fmax(fabs(a), fabs(b));
# }
```

### `<math.h>` essentials (link with `-lm`)

```bash
sqrt, cbrt, pow, exp, exp2, log, log2, log10
sin, cos, tan, asin, acos, atan, atan2
sinh, cosh, tanh
floor, ceil, round, trunc, nearbyint
fabs, fmax, fmin, fmod, remainder, fma
hypot(x,y)        # sqrt(x*x + y*y), no overflow
copysign, signbit
isnan(x), isinf(x), isfinite(x), isnormal(x)
INFINITY, NAN, HUGE_VAL, M_PI (POSIX, _USE_MATH_DEFINES on MSVC)
```

## Pointers

### Basics

```bash
# int x = 42;
# int *p = &x;          // p stores address of x
# int v = *p;           // dereference -> 42
# *p = 100;             // x is now 100
# int **pp = &p;        // pointer to pointer
# int v2 = **pp;        // double deref

# NULL check
# if (p == NULL) ...
# if (!p) ...           // idiomatic
```

### void* — generic pointer

```bash
# void *raw = malloc(100);
# int *ip = raw;        // implicit; cast not required in C (but is in C++)
# int *ip = (int*)raw;  // explicit (some style guides require this)

# void* arithmetic is undefined in standard C (GCC allows as extension).
```

### Function pointers

```bash
# Declaration:
# int (*cmp)(const void*, const void*);

# Assign + call:
# int my_cmp(const void *a, const void *b) { ... }
# cmp = my_cmp;        // (or &my_cmp — equivalent)
# int r = cmp(p1, p2); // (or (*cmp)(p1, p2) — equivalent)

# Typedef for sanity:
# typedef int (*compare_fn)(const void*, const void*);
# compare_fn cmp = my_cmp;

# Returning function pointer:
# int (*get_handler(int kind))(int, int);
# // typedef helps a lot here.
```

### Pointer arithmetic

```bash
# arr + n advances n * sizeof(*arr) bytes:
# int a[5];
# int *p = a;
# *(p + 2);             // == a[2]
# p[2];                 // identical
# p - q;                // ptrdiff_t = element distance
# Pointer arithmetic only legal within an array (or one-past-the-end).
```

## Arrays

### Declaration and decay

```bash
# int a[5] = {1,2,3,4,5};
# int b[] = {1,2,3};                // size deduced -> 3
# int c[100] = {0};                 // first 0, rest zero-initialized
# int d[100] = {[50]=1, [99]=2};    // designated init (C99)

# Array decay: in most expressions, `a` becomes a pointer to a[0].
# Exceptions: sizeof(a), &a, _Alignof, string literal init.

# Length:
# size_t n = sizeof(a) / sizeof(a[0]);  // ONLY works on actual array, not pointer
```

### Multi-dimensional

```bash
# int grid[3][4] = { {1,2,3,4}, {5,6,7,8}, {9,10,11,12} };
# grid[i][j];                          // row-major

# Pass to function:
# void f(int g[3][4]);                 // size of inner dims required
# void g(int rows, int cols, int (*g)[/*cols*/]);  // VLA-as-param (C99)
```

### VLA (Variable Length Array, C99)

```bash
# int n = read_n();
# int buf[n];                          // stack-allocated, size at runtime

# Caveats:
#   - Not available in C++; optional in C11+ (__STDC_NO_VLA__ macro).
#   - Stack overflow risk for large n. Prefer malloc beyond a few KB.
#   - MSVC does not support VLAs.
```

## Strings

### Null-terminated

```bash
# A C string is a char array ending in '\0'.
# char s[] = "hello";       // 6 bytes: 'h','e','l','l','o','\0'
# char *p = "hello";        // pointer to read-only string literal — modifying is UB
# char m[6] = "hello";      // mutable copy
```

### `<string.h>` reference

```bash
size_t strlen(const char *s);
char  *strcpy(char *dst, const char *src);            # UNSAFE — no bound
char  *strncpy(char *dst, const char *src, size_t n); # may not null-term if src ≥ n
char  *strcat(char *dst, const char *src);            # UNSAFE
char  *strncat(char *dst, const char *src, size_t n);
int    strcmp(const char *a, const char *b);          # <0, 0, >0
int    strncmp(const char *a, const char *b, size_t n);
int    strcasecmp(const char *a, const char *b);      # POSIX
char  *strchr(const char *s, int c);                  # first c
char  *strrchr(const char *s, int c);                 # last c
char  *strstr(const char *h, const char *needle);
char  *strtok(char *s, const char *delim);            # MUTATES s, NOT thread-safe
char  *strtok_r(char *s, const char *d, char **save); # POSIX, reentrant
char  *strdup(const char *s);                         # POSIX, malloc'd copy
size_t strspn / strcspn / strpbrk
char  *strerror(int errno_val);
void  *memcpy(void *dst, const void *src, size_t n);  # MUST NOT overlap
void  *memmove(void *dst, const void *src, size_t n); # overlap OK
void  *memset(void *p, int byte, size_t n);
int    memcmp(const void *a, const void *b, size_t n);
void  *memchr(const void *p, int c, size_t n);
```

### strtok pitfalls

```bash
# strtok modifies its input and uses static state — not reentrant.
# char line[] = "a,b,c";
# for (char *t = strtok(line, ","); t; t = strtok(NULL, ","))
#     puts(t);

# Reentrant version (POSIX):
# char *save = NULL;
# for (char *t = strtok_r(line, ",", &save); t; t = strtok_r(NULL, ",", &save))
#     puts(t);
```

## Structs, Unions, Enums

### Structs

```bash
# struct Point { int x; int y; };           // tag form; declare with `struct Point p;`
# typedef struct { int x, y; } Point;       // tag-less + typedef
# typedef struct Node { int v; struct Node *next; } Node;  // self-referential needs tag

# Designated initializers (C99):
# Point p = { .x = 1, .y = 2 };

# Compound literal (C99):
# update(&(Point){ .x = 1, .y = 2 });

# Anonymous struct/union members (C11):
# struct S {
#     int kind;
#     union { int i; float f; };           // access as s.i / s.f directly
# };
```

### Unions

```bash
# union U { int i; float f; char b[4]; };
# Only one member is "active" at a time. Reading another is implementation-defined
# (commonly used for type-punning). Strict aliasing rules still apply.

# Tagged variant pattern:
# typedef enum { V_INT, V_STR } Kind;
# typedef struct {
#     Kind kind;
#     union { int i; const char *s; };
# } Var;
```

### Bit fields

```bash
# struct Flags {
#     unsigned in_use : 1;
#     unsigned kind   : 3;
#     unsigned        : 4;   // unnamed padding
#     unsigned id     : 24;
# };
# Layout/order is implementation-defined — DO NOT use for wire formats.
# Use explicit shifts/masks when crossing the wire.
```

### Enums

```bash
# enum Day { MON, TUE, WED, THU, FRI, SAT, SUN };  // 0..6
# enum HTTP { OK = 200, NOT_FOUND = 404 };
# typedef enum { LEFT, RIGHT } Side;

# C23: enum with explicit underlying type:
# enum Op : uint8_t { OP_A = 1, OP_B = 2 };
```

## typedef + Opaque Types

### Aliasing types

```bash
# typedef unsigned long u64;
# typedef int (*compare_fn)(const void*, const void*);
# typedef struct Slab Slab;       // forward decl + typedef in one
```

### Opaque handles

```bash
# Header (slab.h):
# typedef struct Slab Slab;        // incomplete type — size unknown to callers
# Slab *slab_create(size_t n);
# void  slab_free(Slab *s);

# Source (slab.c):
# struct Slab { size_t n; void *data; };
# Slab *slab_create(size_t n) { ... }

# Callers cannot see the layout — true encapsulation.
```

## Storage Classes

```bash
auto             # default for block-scope (rarely written)
register         # hint to keep in register; cannot take &; deprecated in C23
static
    # at file scope: internal linkage (private to .c file)
    # at block scope: lifetime = whole program, value persists
extern           # declares an object/fn defined elsewhere; no allocation
_Thread_local    # C11; one instance per thread (also: thread_local in C23 / <threads.h>)

# Examples:
# static int counter = 0;          // file-private
# void f(void) { static int n = 0; n++; }   // persistent counter
# extern int errno;                // (actually a thread-local macro on Linux)
```

## const / volatile / restrict

### const

```bash
# const int x = 5;             // x cannot be assigned
# const char *p;               // p points to const chars (can repoint p, can't modify *p)
# char *const p;               // p is const pointer to mutable chars
# const char *const p;         // both const
# Rule of thumb: const binds to whatever is on its LEFT (or to the right if leftmost).
```

### volatile

```bash
# Tells compiler: this object may change outside program flow (MMIO, signal handler).
# volatile uint32_t *reg = (uint32_t*)0x40021000;
# *reg = 1;                    // compiler must NOT optimize-away
# Does NOT make access atomic. Use _Atomic for that.
```

### restrict (C99)

```bash
# Promise: only this pointer (or one derived from it) accesses the object.
# Enables aliasing-based optimizations (vectorization, fewer reloads).
#
# void copy(int *restrict dst, const int *restrict src, size_t n) {
#     for (size_t i = 0; i < n; i++) dst[i] = src[i];
# }
# Lying = UB. memcpy is restrict, memmove is NOT.
```

## Control Flow

### if / else

```bash
# if (cond) { ... } else if (cond2) { ... } else { ... }
# Always brace, even one-liners — Apple "goto fail" CVE-2014-1266.
```

### switch

```bash
# switch (x) {
#     case 1:
#     case 2: do_low(); break;
#     case 3: do_three(); /* fallthrough */
#     case 4: do_four(); break;
#     default: do_other();
# }
#
# C23: [[fallthrough]] attribute (GCC: __attribute__((fallthrough));).
# Cases must be integer constant expressions. No range cases (GCC ext: case 1 ... 5).
```

### goto + labels

```bash
# Useful idiom: cleanup on error.
# int f(void) {
#     int rc = -1;
#     FILE *fp = fopen("a", "r"); if (!fp) goto out;
#     char *buf = malloc(1024);   if (!buf) goto close_fp;
#     /* work */
#     rc = 0;
#     free(buf);
# close_fp:
#     fclose(fp);
# out:
#     return rc;
# }
# goto can only jump within a function and not into a VLA scope.
```

## Loops

### for / while / do-while

```bash
# for (int i = 0; i < n; i++) { ... }   // C99 allows decl in init clause
# while (cond) { ... }
# do { ... } while (cond);              // body always runs at least once
#
# break    — exit innermost loop or switch
# continue — skip to next iteration
#
# Infinite loop idioms:
# for (;;) { ... }
# while (1) { ... }
```

### Common pitfalls

```bash
# Off-by-one with size_t reverse:
# BROKEN:  for (size_t i = n - 1; i >= 0; i--)   // infinite loop, unsigned never <0
# FIXED:   for (size_t i = n; i-- > 0; )

# Trailing semicolon:
# BROKEN:  for (i = 0; i < 10; i++);             // empty body — easy to miss
#             do_thing(i);                       // runs once with i=10
# FIXED:   for (i = 0; i < 10; i++)
#              do_thing(i);
```

## Functions

### Declaration vs Definition

```bash
# Declaration (prototype) — usually in header:
# int add(int a, int b);

# Definition — in .c:
# int add(int a, int b) { return a + b; }

# Old K&R / no-prototype declarations are deprecated.
# C23 makes `int f();` mean `int f(void);` (matches C++).
```

### void-arg vs no-arg

```bash
# int f(void);    // takes NO arguments (always write this)
# int g();        // pre-C23: unspecified args; C23: same as (void)
```

### inline (C99)

```bash
# In header:
# static inline int square(int x) { return x * x; }
# (`static inline` is the safest portable form — no linkage drama.)
```

### _Noreturn (C11)

```bash
# #include <stdnoreturn.h>           // noreturn keyword
# noreturn void die(const char *msg) {
#     fputs(msg, stderr);
#     exit(1);
# }
# C23: use [[noreturn]] attribute instead.
```

### Variadic — `<stdarg.h>`

```bash
# int sum(int n, ...) {
#     va_list ap;
#     va_start(ap, n);
#     int s = 0;
#     for (int i = 0; i < n; i++) s += va_arg(ap, int);
#     va_end(ap);
#     return s;
# }
# Each va_arg promotes — ask for `int` not `short`, `double` not `float`.
# Wrap printf-style with vprintf / vsnprintf.
# Annotate format-string fns:  __attribute__((format(printf, 1, 2)))
```

## Function Pointers & Callbacks

### qsort

```bash
# int cmp_int(const void *a, const void *b) {
#     int x = *(const int*)a, y = *(const int*)b;
#     return (x > y) - (x < y);          // safe — no overflow
# }
# qsort(arr, n, sizeof(arr[0]), cmp_int);
#
# Don't use `return *a - *b` — overflow on (INT_MIN, 1).
```

### Dispatch table

```bash
# typedef int (*op_fn)(int, int);
# int op_add(int a, int b) { return a + b; }
# int op_sub(int a, int b) { return a - b; }
# op_fn ops[] = { op_add, op_sub };
# int r = ops[kind](x, y);
```

## Dynamic Memory

### malloc / calloc / realloc / free

```bash
# void *malloc(size_t n);                // n bytes, uninitialized (junk)
# void *calloc(size_t cnt, size_t sz);   // cnt*sz bytes, zeroed
# void *realloc(void *p, size_t n);      // resize; may return NEW pointer
# void  free(void *p);                   // free(NULL) is OK; free same ptr twice = UB
# void *aligned_alloc(size_t align, size_t sz);  // C11; sz must be multiple of align

# Always check for NULL:
# int *a = malloc(n * sizeof *a);
# if (!a) { perror("malloc"); exit(1); }

# realloc gotcha:
# BROKEN:  p = realloc(p, n2);    // if NULL, original leaks
# FIXED:   void *q = realloc(p, n2);
#          if (!q) { /* original p still valid */ free(p); err(); }
#          p = q;
```

### Pitfalls

```bash
# Double free:
# free(p); free(p);                  // UB, often heap corruption
# Defense: free(p); p = NULL;         // free(NULL) is a no-op

# Use-after-free:
# free(p); p->x = 1;                 // UB

# Memory leak (sanitizer with -fsanitize=leak or LSAN catches):
# p = malloc(n); p = malloc(m);      // first allocation leaked

# Misalignment of size:
# malloc(sizeof(p));                 // BUG — sizeof pointer, not pointee
# FIXED: malloc(sizeof *p) or malloc(N * sizeof *p);
```

## File I/O

### `<stdio.h>` essentials

```bash
FILE *fopen(const char *path, const char *mode);
    # modes: "r" "w" "a" "r+" "w+" "a+"   — append "b" for binary
int   fclose(FILE *fp);
int   fflush(FILE *fp);

size_t fread(void *buf, size_t sz, size_t cnt, FILE *fp);
size_t fwrite(const void *buf, size_t sz, size_t cnt, FILE *fp);

int   fgetc(FILE *fp);          # returns int (so EOF = -1 fits)
int   fputc(int c, FILE *fp);
char *fgets(char *buf, int sz, FILE *fp);   # reads ≤ sz-1, null-terms, keeps '\n'
int   fputs(const char *s, FILE *fp);
int   ungetc(int c, FILE *fp);

int   fseek(FILE *fp, long off, int whence);   # whence: SEEK_SET, SEEK_CUR, SEEK_END
long  ftell(FILE *fp);
void  rewind(FILE *fp);

int   feof(FILE *fp);
int   ferror(FILE *fp);
void  clearerr(FILE *fp);

int   remove(const char *path);
int   rename(const char *old, const char *new);
FILE *tmpfile(void);
char *tmpnam(char *buf);        # avoid — race condition; use mkstemp on POSIX
```

### NEVER use gets

```bash
# gets() removed from the standard in C11 — unbounded read, classic stack-smash.
# Use fgets:
# char line[256];
# if (fgets(line, sizeof line, stdin) == NULL) { /* EOF or error */ }
# // Strip trailing newline:
# size_t L = strlen(line);
# if (L && line[L-1] == '\n') line[L-1] = '\0';
```

### EOF handling

```bash
# Loop pattern for char-by-char:
# int c;
# while ((c = fgetc(fp)) != EOF) putchar(c);
#
# Distinguish EOF vs error:
# if (feof(fp))  /* clean EOF */
# if (ferror(fp))/* read/write error — check errno */
```

## Streams

### stdin / stdout / stderr

```bash
# Three pre-opened FILE* in <stdio.h>:
#   stdin   — file descriptor 0
#   stdout  — fd 1, line-buffered when connected to a terminal, fully-buffered to pipe
#   stderr  — fd 2, UNBUFFERED by default
#
# fprintf(stderr, "warning: %s\n", msg);
# Always flush before fork/exec or before exiting on a crash:
# fflush(stdout);
```

### setvbuf — control buffering

```bash
# setvbuf(stream, NULL, _IONBF, 0);          // unbuffered
# setvbuf(stream, NULL, _IOLBF, BUFSIZ);     // line-buffered
# setvbuf(stream, NULL, _IOFBF, BUFSIZ);     // fully buffered (default)
# Common need: when piping output, force line-buffer:
# setvbuf(stdout, NULL, _IOLBF, 0);
```

## Formatted I/O

### printf format specifiers

```bash
# %[flags][width][.precision][length]conversion
#
# Conversion (selected):
#   d, i  — signed decimal int
#   u     — unsigned decimal
#   o, x, X — unsigned octal / hex (lower/upper)
#   c     — char (int promoted)
#   s     — null-terminated string
#   p     — pointer (implementation-defined)
#   e, E  — scientific (1.23e+05)
#   f, F  — fixed-point (123.456)
#   g, G  — %e or %f, whichever shorter, trims zeros
#   a, A  — hex float
#   %     — literal '%'
#   n     — store chars-written count (DANGEROUS, format-string-attack target;
#           glibc requires _FORTIFY_SOURCE or read-only format)
#
# Flags:
#   -    left-justify
#   +    show sign on positive
#   ' '  leading space on positive
#   0    zero-pad
#   #    alt form (0x for %#x, decimal point for %#g)
#
# Width:   minimum field width (or '*' to read from arg)
# Precision: '.' followed by digits (or '.*'); for %s = max chars, %f = decimals,
#            %d = minimum digits.
#
# Length modifiers:
#   hh   char        h    short
#   l    long        ll   long long
#   j    intmax_t    z    size_t        t    ptrdiff_t   L    long double
#
# Examples:
# printf("%-20s %5d\n",   name, age);
# printf("%08x\n",        crc);
# printf("%.3f\n",        3.14159);     // 3.142
# printf("%*.*f\n", 10, 3, x);          // width 10, prec 3
# printf("%zu / %td\n",   sizeof(s), p2-p1);
# printf("%" PRId64 "\n", big);
```

### scanf perils

```bash
# %s — UNBOUNDED, just like gets. ALWAYS specify max width:
# char buf[64];
# scanf("%63s", buf);                    // leaves 1 byte for '\0'

# Mismatched specifier and arg type = UB.
# Always check return value (number of conversions):
# if (scanf("%d", &x) != 1) { /* parse error or EOF */ }

# Prefer fgets + strtol for line-based input:
# char line[256];
# if (fgets(line, sizeof line, stdin)) {
#     errno = 0;
#     char *end; long v = strtol(line, &end, 10);
#     if (end == line || (errno == ERANGE)) { /* error */ }
# }
```

## errno + `<errno.h>`

```bash
# errno is a thread-local int set by many libc/POSIX functions on failure.
# Save it before any other library call (which might clobber it).
#
# Common values:
#   EACCES    permission denied
#   ENOENT    no such file or directory
#   ENOMEM    out of memory
#   EINTR     interrupted system call
#   EAGAIN    try again (= EWOULDBLOCK on most systems)
#   EINVAL    invalid argument
#   EEXIST    file exists
#   EIO       I/O error
#   EBADF     bad file descriptor
#
# Reporting:
# perror("fopen");                            -> fopen: No such file or directory
# fprintf(stderr, "open: %s\n", strerror(errno));
# strerror_r(errno, buf, sizeof buf);         // POSIX, thread-safe
#
# Always set errno=0 BEFORE strtol/strtod when you want to detect ERANGE.
```

## Signals

### `<signal.h>`

```bash
# Common signals:
#   SIGINT    Ctrl-C
#   SIGTERM   default kill
#   SIGKILL   forced kill (cannot catch)
#   SIGSTOP   stop (cannot catch)
#   SIGQUIT   Ctrl-\, core dump
#   SIGSEGV   invalid memory access
#   SIGBUS    misaligned access
#   SIGFPE    arithmetic error (div by zero)
#   SIGILL    illegal instruction
#   SIGPIPE   write to pipe with no readers — IGNORE in servers
#   SIGCHLD   child changed state
#   SIGALRM   alarm() timer
#   SIGUSR1, SIGUSR2  user-defined
#   SIGHUP    terminal hangup / reload config convention
```

### signal vs sigaction

```bash
# signal() is portable but legacy and underspecified — semantics differ across
# systems. Prefer sigaction (POSIX):
#
# void on_int(int sig) { /* set a sig_atomic_t flag */ }
#
# struct sigaction sa = {0};
# sa.sa_handler = on_int;
# sigemptyset(&sa.sa_mask);
# sa.sa_flags = SA_RESTART;
# sigaction(SIGINT, &sa, NULL);
#
# Ignore SIGPIPE (servers writing to closed sockets):
# signal(SIGPIPE, SIG_IGN);
```

### Async-signal-safety

```bash
# Inside a signal handler you may ONLY call async-signal-safe functions:
#   write, _exit, signal, sigaction, kill, raise, abort, time,
#   read, open, close, fork, execve, dup2, ...
# NOT safe: printf, malloc, free, fprintf, exit (note: _exit is OK)
#
# Pattern: set a `volatile sig_atomic_t flag = 1;` and check it in main loop.
```

## POSIX Threads

### `pthread_create / join / detach`

```bash
# Compile + link with -pthread (use -pthread on the COMPILE line too on Linux —
# it sets _REENTRANT and links libpthread).
#
# #include <pthread.h>
# void *worker(void *arg) {
#     int n = *(int*)arg;
#     return (void*)(intptr_t)(n*n);
# }
#
# pthread_t t;
# int n = 7;
# pthread_create(&t, NULL, worker, &n);    // returns 0 on success, errno-style code on fail
# void *ret;
# pthread_join(t, &ret);                   // returns 0 on success
# // or: pthread_detach(t)  -> resources reclaimed automatically
```

### Mutexes

```bash
# pthread_mutex_t m = PTHREAD_MUTEX_INITIALIZER;
# pthread_mutex_lock(&m);
# /* critical section */
# pthread_mutex_unlock(&m);
# pthread_mutex_destroy(&m);
#
# Attributes (recursive, error-checking):
# pthread_mutexattr_t a;
# pthread_mutexattr_init(&a);
# pthread_mutexattr_settype(&a, PTHREAD_MUTEX_ERRORCHECK);
# pthread_mutex_init(&m, &a);
```

### Condition variables

```bash
# pthread_cond_t  cv = PTHREAD_COND_INITIALIZER;
# pthread_mutex_t mu = PTHREAD_MUTEX_INITIALIZER;
# int ready = 0;
#
# // Waiter:
# pthread_mutex_lock(&mu);
# while (!ready)
#     pthread_cond_wait(&cv, &mu);          // ALWAYS in a loop — spurious wakeups
# pthread_mutex_unlock(&mu);
#
# // Signaler:
# pthread_mutex_lock(&mu);
# ready = 1;
# pthread_cond_signal(&cv);                 // or _broadcast
# pthread_mutex_unlock(&mu);
```

## C11 Atomics

### `<stdatomic.h>`

```bash
# atomic_int counter = 0;                       // or _Atomic(int)
# atomic_fetch_add(&counter, 1);                // returns old value
# atomic_store(&counter, 0);
# int v = atomic_load(&counter);
# int desired = 5, expected = 0;
# atomic_compare_exchange_strong(&counter, &expected, desired);
#
# Standard typedefs: atomic_bool, atomic_char, atomic_int,
# atomic_uint, atomic_long, atomic_size_t, atomic_uintptr_t, atomic_ptrdiff_t, ...
#
# Spin lock with atomic_flag:
# atomic_flag spin = ATOMIC_FLAG_INIT;
# while (atomic_flag_test_and_set(&spin)) /* spin */;
# /* critical */
# atomic_flag_clear(&spin);
```

### Memory ordering

```bash
# memory_order_relaxed  — only atomicity, no ordering
# memory_order_acquire  — load: no later ops reorder before
# memory_order_release  — store: no earlier ops reorder after
# memory_order_acq_rel  — RMW
# memory_order_seq_cst  — total order (default; safest, slowest)
# memory_order_consume  — discouraged, treated like acquire on most compilers
#
# atomic_load_explicit(&x, memory_order_acquire);
# atomic_store_explicit(&y, 1, memory_order_release);
```

## `<time.h>`

```bash
# time_t t = time(NULL);                       // seconds since epoch
# struct tm tm; localtime_r(&t, &tm);          // POSIX, thread-safe
# struct tm utm; gmtime_r(&t, &utm);
# char buf[64];
# strftime(buf, sizeof buf, "%Y-%m-%d %H:%M:%S", &tm);
#
# Format specifiers (subset):
#   %Y year (4)   %m month (01)  %d day (01)
#   %H hour 24    %I hour 12     %M min   %S sec
#   %A weekday    %B month name  %Z tz   %z +0100
#   %s epoch (GNU)            %F == %Y-%m-%d
#
# Monotonic vs realtime:
# struct timespec t;
# clock_gettime(CLOCK_REALTIME,  &t);          // wall clock; can jump
# clock_gettime(CLOCK_MONOTONIC, &t);          // never goes backwards
# clock_gettime(CLOCK_MONOTONIC_RAW, &t);      // Linux, untouched by NTP
# clock_gettime(CLOCK_PROCESS_CPUTIME_ID, &t); // CPU time
#
# Sleep:
# nanosleep(&(struct timespec){.tv_sec=1, .tv_nsec=500*1000*1000}, NULL);
# struct timespec rem;  while(nanosleep(&req, &rem) == -1 && errno == EINTR) req = rem;
```

## `<stdlib.h>`

```bash
# Environment:
char *getenv(const char *name);                # do not free
int   setenv(const char *name, const char *val, int overwrite);  // POSIX
int   unsetenv(const char *name);                                // POSIX

# Process control:
void  exit(int status);                        # runs atexit handlers, flushes stdio
void  _Exit(int status);                       # no atexit, no flush
int   atexit(void (*fn)(void));                # max ATEXIT_MAX (≥32) handlers
int   system(const char *cmd);                 # SHELL INJECTION DANGER — avoid

# Conversion (NOT atoi family — they have no error reporting):
long  strtol (const char *s, char **end, int base);  # 0=auto, 8/10/16
unsigned long  strtoul(...);
long long      strtoll(...);
unsigned long long strtoull(...);
double          strtod (const char *s, char **end);

# Algorithms:
void  qsort (void *base, size_t n, size_t sz, int (*cmp)(const void*, const void*));
void *bsearch(const void *key, const void *base, size_t n, size_t sz,
              int (*cmp)(const void*, const void*));

# Random (NOT cryptographic):
int   rand(void);                              # RAND_MAX-bounded, weak
void  srand(unsigned seed);
# Use arc4random / getrandom / /dev/urandom for security-grade.

# strtol idiom:
# errno = 0;
# char *end;
# long v = strtol(s, &end, 10);
# if (errno == ERANGE) /* overflow */;
# if (end == s)        /* no digits */;
# if (*end != '\0')    /* trailing junk */;
```

## POSIX Process & Env

### fork / exec / wait

```bash
# #include <unistd.h>  <sys/wait.h>
#
# pid_t pid = fork();
# if (pid == 0) {                              // child
#     execlp("ls", "ls", "-la", NULL);
#     _exit(127);                              // exec failed
# } else if (pid > 0) {                        // parent
#     int status;
#     waitpid(pid, &status, 0);
#     if (WIFEXITED(status))   printf("rc=%d\n", WEXITSTATUS(status));
#     if (WIFSIGNALED(status)) printf("signal %d\n", WTERMSIG(status));
# } else { perror("fork"); }
#
# exec family:
#   execl  (path, arg0, arg1, ..., NULL)
#   execlp (file, ...)               looks up PATH
#   execle (path, ...,  envp)
#   execv  (path, argv[])
#   execvp (file, argv[])
#   execvpe(file, argv[], envp)      GNU
```

### pipe / dup2

```bash
# int fds[2]; pipe(fds);                       // fds[0]=read, fds[1]=write
# pid_t p = fork();
# if (p == 0) {
#     dup2(fds[1], STDOUT_FILENO);             // child stdout -> pipe write
#     close(fds[0]); close(fds[1]);
#     execlp("ls", "ls", NULL);
# }
# close(fds[1]);
# char buf[1024]; ssize_t n = read(fds[0], buf, sizeof buf);
```

### environ

```bash
# extern char **environ;                       // NULL-terminated array of "KEY=VALUE"
# for (char **e = environ; *e; e++) puts(*e);
```

## Header Layout

### Conventions

```bash
# foo.h
# #ifndef FOO_H
# #define FOO_H
#
# #include <stddef.h>             // ONLY what foo.h needs (forward-declare otherwise)
#
# typedef struct Foo Foo;         // opaque
# Foo *foo_new(size_t n);
# void foo_free(Foo *f);
# extern int foo_count;           // declaration only
#
# #endif
#
# foo.c
# #include "foo.h"
# #include <stdlib.h>
#
# struct Foo { size_t n; };
# int foo_count = 0;              // single definition
# Foo *foo_new(size_t n) { ... }
```

### Rules of thumb

```bash
# - Headers contain DECLARATIONS, not DEFINITIONS (except inline / static).
# - Each .h is self-sufficient: compiles with no other includes preceding it.
# - extern "C" wrap only when supporting C++ callers:
#   #ifdef __cplusplus
#   extern "C" {
#   #endif
#   /* declarations */
#   #ifdef __cplusplus
#   }
#   #endif
```

## Linking

### Object files and archives

```bash
# .o   — object file (one translation unit, ELF/Mach-O/COFF)
# .a   — static archive (collection of .o); linker pulls in members it needs
# .so  — Linux shared object (dynamic library)
# .dylib — macOS dynamic library
# .dll — Windows DLL (with companion import lib .lib)
#
# Build a static lib:
# gcc -c a.c b.c
# ar rcs libmy.a a.o b.o
# gcc app.c -L. -lmy -o app                    // -lmy -> libmy.a or libmy.so
#
# Build a shared lib (Linux):
# gcc -fPIC -c a.c b.c
# gcc -shared -o libmy.so a.o b.o
# gcc app.c -L. -lmy -Wl,-rpath,'$ORIGIN' -o app
#
# Build dylib (macOS):
# clang -dynamiclib -o libmy.dylib a.c b.c
# clang app.c -L. -lmy -Wl,-rpath,@loader_path -o app
```

### Linker flags

```bash
-l<name>                # libname.{so|a}
-L<dir>                 # add to library search path
-static                 # link statically (libc.a — large, portable)
-shared                 # produce .so/.dylib
-Wl,<linker-arg>        # pass through to ld
-Wl,-rpath,<path>       # runtime lib search path
-Wl,-z,now              # bind all symbols at load (full RELRO)
-Wl,-z,relro            # mark GOT read-only after init
-Wl,--gc-sections       # drop unused sections (with -ffunction-sections)
-Wl,--as-needed         # only link libs actually used
```

### dlopen / dlsym

```bash
# #include <dlfcn.h>
# void *h = dlopen("libplugin.so", RTLD_NOW);
# if (!h) { fputs(dlerror(), stderr); exit(1); }
# typedef int (*hello_fn)(void);
# hello_fn f = (hello_fn)dlsym(h, "hello");
# if (!f) { fputs(dlerror(), stderr); exit(1); }
# f();
# dlclose(h);
# Link with -ldl on Linux.
```

## Compiler Flags Reference

### Optimization

```bash
-O0       # No optimization (default). Best for debugging.
-O1       # Basic.
-O2       # Standard release. No size-explosion options.
-O3       # Aggressive: vectorization, more inlining (sometimes slower!).
-Os       # Optimize for size.
-Oz       # Even smaller (Clang).
-Ofast    # -O3 + non-conforming math (-ffast-math). Avoid unless you mean it.
-Og       # Optimize for debugging — keep most -O1 transforms but preserve debug info.
-flto     # Link-time optimization (whole-program). Pair with -fuse-linker-plugin.
-fprofile-generate / -fprofile-use   # PGO (profile-guided opt).
```

### Debug info

```bash
-g            # default debug format (DWARF on Linux/macOS)
-g3           # include macros
-gdwarf-4     # specific DWARF version
-fno-omit-frame-pointer    # readable backtraces with perf, eBPF, ASan
```

### Warnings (full list)

```bash
-Wall                  # essentials
-Wextra                # additional
-Werror                # promote to errors
-Wpedantic / -pedantic # strict ISO
-Wshadow               # var shadows another
-Wundef                # #if X with X undefined
-Wmissing-prototypes   # non-static fn lacks prototype
-Wstrict-prototypes    # K&R-style decl
-Wold-style-definition # K&R-style def
-Wformat=2 -Wformat-security
-Wnull-dereference
-Wdouble-promotion
-Wfloat-equal
-Wcast-align=strict
-Wcast-qual            # cast away const
-Wlogical-op           # GCC: same expr on both sides of && / ||
-Wjump-misses-init     # GCC: goto skips initializer
-Wstack-usage=N        # warn fns whose stack frame > N
-Wvla                  # warn on VLAs (banned in many style guides)
-Wno-unused-parameter  # turn off a single warn (often useful)
```

### Sanitizers and hardening

```bash
-fsanitize=address              # ASan — buffer overflows, UAF, double-free
-fsanitize=undefined            # UBSan — signed overflow, shifts, alignment, ...
-fsanitize=thread               # TSan — data races
-fsanitize=memory               # MSan — uninitialized reads (Clang only; rebuild deps)
-fsanitize=leak                 # LSan — leaks (subset of ASan; runs at exit)
-fno-sanitize-recover=all       # abort on first finding instead of continuing
-fstack-protector-strong        # canary on functions with arrays/locals
-fstack-clash-protection        # incremental stack probes
-D_FORTIFY_SOURCE=2 -O1         # libc-side bounds checks for str/mem/printf fns
-fPIC                           # position-indep code (required for shared libs)
-pie -fPIE                      # position-indep executable (ASLR-friendly)
-fcf-protection=full            # CET (Intel branch tracking)
-fno-strict-aliasing            # disable type-based alias optimization (escape hatch)
-fno-common                     # one definition per global (default since GCC 10)
-march=native                   # tune for build host CPU
-mtune=native                   # same but no new ISA features
-static                         # static linking
```

## Sanitizers

### What each catches

```bash
# ASan (-fsanitize=address)
#   - Heap buffer overflow / underflow
#   - Stack buffer overflow (with -fstack-protector-* you also get canaries)
#   - Use-after-free, use-after-scope, use-after-return
#   - Double-free, invalid-free
#   - Slowdown ~2x, memory ~3x.
#   - Env: ASAN_OPTIONS=detect_leaks=1:abort_on_error=1:halt_on_error=1
#
# UBSan (-fsanitize=undefined)
#   - Signed integer overflow, divide by zero
#   - Shift by ≥ width
#   - Misaligned pointer load/store
#   - Null pointer deref
#   - Bool/enum out-of-range
#   - Env: UBSAN_OPTIONS=print_stacktrace=1:halt_on_error=1
#
# TSan (-fsanitize=thread)
#   - Data races
#   - 5–15x slowdown; rebuild EVERYTHING with TSan
#   - Env: TSAN_OPTIONS=halt_on_error=1
#
# MSan (-fsanitize=memory) — Clang only
#   - Reads of uninitialized memory
#   - Must rebuild dependencies (including libc++ if applicable)
#
# LSan (-fsanitize=leak)
#   - Memory leaks at exit
#   - Implicit in ASan with detect_leaks=1
```

## Debugging

### gdb essentials

```bash
gdb ./app                           # launch
gdb --args ./app arg1 arg2          # with args
gdb -p PID                          # attach to running process
gdb ./app core                      # post-mortem on coredump

# Inside gdb:
run / r                             # start program
break main / b file.c:42 / b func   # set breakpoint
break func if x > 10                # conditional bp
info breakpoints                    # list
delete N / clear                    # remove
continue / c
step / s                            # into call
next / n                            # over call
finish                              # run until return
until N                             # run until line N
backtrace / bt / bt full
frame N / f N                       # select stack frame
info locals / info args / info registers
print expr / p *p / p arr@10        # last form: print 10 elements of arr
ptype expr                          # show type
display expr                        # auto-print every step
watch var                           # break on write
rwatch var / awatch var
x/16xb &v                           # examine 16 bytes hex from &v
x/16xw $pc                          # 16 words at PC
disassemble / disas                 # show asm
set var x = 5                       # poke variable
set print pretty on
set logging on                      # log to gdb.txt
generate-core-file
quit / q
```

### lldb essentials (macOS / LLVM)

```bash
lldb ./app                          # launch
process launch -- arg1 arg2
breakpoint set -n main / b main.c:42
run / r
step / s     next / n     continue / c
frame variable / fr v
po expr                              # ObjC/Swift print-object
expression x = 5
bt    / image lookup -a $pc
quit
```

### Valgrind

```bash
valgrind --leak-check=full --show-leak-kinds=all --track-origins=yes ./app
# Tools: memcheck (default), helgrind (threading), cachegrind, callgrind, massif (heap profile)
valgrind --tool=helgrind ./app
valgrind --tool=callgrind ./app && callgrind_annotate callgrind.out.PID
```

## Static Analysis

```bash
# Clang
clang --analyze -Xanalyzer -analyzer-output=text file.c
scan-build make                         # wraps the build, runs analyzer

# GCC (10+)
gcc -fanalyzer -Wall -Wextra file.c

# Cppcheck
cppcheck --enable=all --inconclusive --std=c11 src/

# Other heavy tools: Coverity, PVS-Studio, Infer (Facebook), Frama-C (formal).
```

## Undefined Behavior catalog

```bash
# Signed integer overflow
# BROKEN:  int x = INT_MAX; x = x + 1;          // UB; -fsanitize=undefined catches
# FIXED:   if (x > INT_MAX - 1) /* error */; else x++;

# Shift count >= width
# BROKEN:  uint32_t v = 1u; v = v << 32;        // UB
# FIXED:   if (n < 32) v <<= n;

# Unsequenced modifications
# BROKEN:  i = i++;             a[i] = i++;     // UB pre-C11; unspecified C17+
# FIXED:   i++;                 int j = i; a[j] = j+1; i = j+1;

# Strict aliasing
# BROKEN:  float f; uint32_t bits = *(uint32_t*)&f;   // type-pun via pointer cast
# FIXED:   uint32_t bits; memcpy(&bits, &f, sizeof bits);

# NULL dereference
# BROKEN:  char *p = NULL; *p = 'x';
# FIXED:   if (p) *p = 'x';

# Out-of-bounds access
# BROKEN:  int a[10]; a[10] = 1;
# FIXED:   if (i < 10) a[i] = 1;

# Modifying string literal
# BROKEN:  char *s = "hi"; s[0] = 'H';          // UB — literal in .rodata
# FIXED:   char s[] = "hi"; s[0] = 'H';

# Reading uninitialized
# BROKEN:  int x; printf("%d\n", x);
# FIXED:   int x = 0;

# Calling fn through wrong type
# BROKEN:  int (*f)(int) = (int(*)(int))some_void_fn;
# FIXED:   match the actual signature.

# Division by zero (integer)
# BROKEN:  int q = a / 0;
# FIXED:   if (b == 0) /* error */; else q = a / b;
```

## Format String Vulnerabilities

```bash
# When user input becomes the format string, an attacker can read/write memory.
#
# BROKEN:  printf(user_input);
#          // Attack: input "%x %x %x %s" leaks stack, "%n" writes.
# FIXED:   printf("%s", user_input);
#          fputs(user_input, stdout);
#
# Same applies to syslog, fprintf, snprintf, dprintf — never let user data reach
# the format-string parameter.
#
# Compile defense:
#   -Wformat=2 -Wformat-security -Werror=format-security
# At runtime, FORTIFY_SOURCE blocks %n in writable format strings.
```

## Buffer Overflow Patterns

```bash
# strcpy / strcat — UNBOUNDED:
# BROKEN:  char buf[16]; strcpy(buf, user);
# FIXED:   char buf[16];
#          if (strlen(user) >= sizeof buf) /* error */;
#          memcpy(buf, user, strlen(user)+1);

# sprintf — UNBOUNDED:
# BROKEN:  sprintf(buf, "%s/%s", dir, name);
# FIXED:   if (snprintf(buf, sizeof buf, "%s/%s", dir, name) >= (int)sizeof buf)
#              /* truncated, treat as error */;

# strncpy — does NOT null-terminate if src is too long:
# BROKEN:  char dst[16]; strncpy(dst, src, sizeof dst);
#          // dst may be missing '\0'
# FIXED:   strncpy(dst, src, sizeof dst - 1);
#          dst[sizeof dst - 1] = '\0';
# Or: snprintf(dst, sizeof dst, "%s", src);

# gets — REMOVED in C11. Always use fgets:
# BROKEN:  gets(buf);
# FIXED:   fgets(buf, sizeof buf, stdin);

# strlcpy / strlcat — BSD/macOS, in glibc since 2.38: bounded, always null-term.
```

## Common Gotchas

### Integer promotion surprise

```bash
# BROKEN:  uint8_t a = 0xFF, b = 0xFF;
#          if (a + b > 0xFF) printf("yes\n");          // promoted to int -> 510 > 255 -> yes
# FIXED:   uint8_t r = (a + b) & 0xFF;                  // explicit narrow if you wanted u8
```

### Signed vs unsigned compare

```bash
# BROKEN:  int s = -1;
#          unsigned u = 1;
#          if (s < u) ... ;                             // s converted -> UINT_MAX, false
# FIXED:   if (s < 0 || (unsigned)s < u) ... ;
# Compile with -Wsign-compare to catch.
```

### sizeof on array vs pointer

```bash
# BROKEN:  void f(int a[10]) { size_t n = sizeof a / sizeof a[0]; } // a is int*, n=2
# FIXED:   pass length:
#          void f(int *a, size_t n) { ... }
# Inside the same scope where the array is declared, sizeof works as expected.
```

### Macro multiple-evaluation

```bash
# BROKEN:  #define MAX(a,b) ((a)>(b)?(a):(b))
#          int x = MAX(i++, j++);                  // i or j incremented twice
# FIXED:   inline function:
#          static inline int imax(int a, int b) { return a>b ? a : b; }
# Or GCC statement-expression (non-portable):
#          #define MAX(a,b) ({ __auto_type _a=(a); __auto_type _b=(b); _a>_b?_a:_b; })
```

### Struct padding

```bash
# struct S { char a; int b; char c; };
#   sizeof(struct S) often == 12 (not 6) on 64-bit due to alignment.
# Implication: do NOT memcmp two structs unless you memset to 0 first.
# struct S x = {0}; x.a = 1; x.b = 2; x.c = 3;
# struct S y = {0}; y.a = 1; y.b = 2; y.c = 3;
# memcmp(&x, &y, sizeof x);   // safe only because we zeroed padding
```

### Header order

```bash
# Some POSIX headers require feature-test macros set BEFORE any include:
# #define _POSIX_C_SOURCE 200809L
# #define _GNU_SOURCE
# #include <stdio.h>      // OK after macros
# Wrong order silently gives you a different prototype set.
```

## Performance Tips

```bash
# Profile, don't guess. Tools:
#   perf record -g ./app && perf report
#   gprof ./app gmon.out
#   callgrind + kcachegrind
#   Linux: perf stat -d -d -d ./app   (cache misses, branch misses, IPC)

# Compiler:
#   -O2 is the standard release level.
#   -O3 enables vectorization but can slow corner cases — measure.
#   -march=native + -mtune=native for build-host CPU.
#   -flto for cross-TU inlining (slower link, faster runtime).
#   PGO: -fprofile-generate, run, -fprofile-use (often +5–15%).

# Memory layout:
#   Prefer struct-of-arrays (SoA) over array-of-structs (AoS) for hot inner loops.
#   Pad hot atomics to 64 bytes (false-sharing). _Alignas(64).
#   Cache lines on x86 = 64 B. Use __builtin_prefetch(p, rw, locality) sparingly.

# restrict:
#   void axpy(size_t n, double a, double *restrict x, double *restrict y) {
#       for (size_t i=0;i<n;i++) y[i] += a * x[i];
#   }
#   Lets the compiler vectorize without alias-checking.

# SIMD:
#   -O3 -march=native enables auto-vectorization. Verify in -S output.
#   Manual: <immintrin.h> for SSE/AVX, <arm_neon.h> for NEON.

# Branch hints (use sparingly):
#   if (__builtin_expect(rare, 0)) { ... }
#   [[likely]] / [[unlikely]] — C23.
```

## Security Checklist

```bash
# - No gets, sprintf without bound, strcpy/strcat on user input.
# - Bound EVERY string copy (snprintf or memcpy with explicit length).
# - Check EVERY malloc / realloc / fopen / open / read / write.
# - Format strings always come from program text:
#     printf("%s", user) ✓     printf(user) ✗
# - Compile flags:
#     -O2 -D_FORTIFY_SOURCE=2 -fstack-protector-strong -fPIE -pie
#     -Wl,-z,relro -Wl,-z,now -Wl,--as-needed
#     -fcf-protection=full   (Intel CET)
#     -fstack-clash-protection
# - Run with sanitizers in CI: ASan + UBSan; TSan in a separate job for threaded code.
# - For randomness: getentropy / getrandom / arc4random — NEVER rand() for keys/tokens.
# - For crypto: use a library (libsodium, BoringSSL); do not roll your own.
# - Drop privileges (setuid/setgid) immediately after binding privileged port.
# - Use chroot, seccomp-bpf, capabilities, namespaces where appropriate.
```

## Idioms

### Defer-via-goto cleanup

```bash
# int load(const char *path, Buf *out) {
#     int rc = -1;
#     FILE *fp = NULL;
#     char *data = NULL;
#
#     fp = fopen(path, "rb");
#     if (!fp) goto out;
#
#     fseek(fp, 0, SEEK_END);
#     long sz = ftell(fp);
#     if (sz < 0) goto out;
#     fseek(fp, 0, SEEK_SET);
#
#     data = malloc((size_t)sz);
#     if (!data) goto out;
#     if (fread(data, 1, sz, fp) != (size_t)sz) goto out;
#
#     out->data = data; out->len = sz;
#     data = NULL;            // ownership transferred
#     rc = 0;
# out:
#     free(data);             // no-op if NULL
#     if (fp) fclose(fp);
#     return rc;
# }
```

### X-macros (already covered above) — single source of truth for parallel arrays.

### Opaque handle (already covered above) — strong encapsulation.

### Error via out-param

```bash
# typedef struct { int code; const char *msg; } Err;
# bool parse_int(const char *s, int *out, Err *err) {
#     errno = 0; char *end;
#     long v = strtol(s, &end, 10);
#     if (end == s || *end || errno == ERANGE || v > INT_MAX || v < INT_MIN) {
#         if (err) { err->code = -1; err->msg = "bad int"; }
#         return false;
#     }
#     *out = (int)v; return true;
# }
```

### Assert and static_assert

```bash
# #include <assert.h>
# assert(p != NULL);                          // disabled with -DNDEBUG
#
# C11 compile-time:
# _Static_assert(sizeof(int) == 4, "32-bit int required");
# C23: static_assert(sizeof(void*) >= 4);
```

## C23 Highlights

```bash
# typeof / typeof_unqual — like GCC's __typeof__ but standard.
#   typeof(expr) v = expr;

# constexpr — compile-time-evaluable named values:
#   constexpr int N = 32;

# enum with explicit underlying type:
#   enum E : uint16_t { A, B };

# nullptr keyword (replaces (void*)0 NULL):
#   void *p = nullptr;

# true / false / bool — keywords (no need for <stdbool.h>):
#   bool ok = true;

# [[attributes]] — standard attribute syntax:
#   [[nodiscard]] int must_use(void);
#   [[deprecated("use bar")]] void foo(void);
#   [[fallthrough]];
#   [[maybe_unused]] int x;
#   [[noreturn]] void die(void);

# _BitInt(N) — exact-width signed/unsigned integers (any N):
#   _BitInt(7) seven; unsigned _BitInt(128) bigu;

# #embed — embed binary at compile time:
#   const unsigned char favicon[] = {
#   #embed "favicon.ico"
#   };

# auto type-deduction (in declarations):
#   auto x = 1.0f;            // x is float

# u8'X' character literals; UTF-8 string literals u8"..." mandated as char8_t.
# binary literals: 0b1010_0011 (separators allowed: 1'000'000 — no, that's C++).
# Removal: K&R function definitions; trigraphs; gets (already C11); register storage class as keyword (still allowed).
```

## Tips

- Always set `-std=cNN` explicitly; do not depend on the compiler default.
- Compile with `-Wall -Wextra -Werror` plus targeted `-W` flags from day one.
- Run unit and integration tests under `-fsanitize=address,undefined` in CI.
- `static inline` for functions defined in headers — safe, no linker drama.
- Prefer `sizeof *p` over `sizeof(T)` so refactors do not silently break.
- Use `<stdint.h>` types for cross-platform: `uint32_t`, not `unsigned int`.
- `size_t` for sizes/indices/loop counts; never compare with `-1`.
- `free(NULL)` is legal; `fclose(NULL)` is UB — guard it.
- Never `return` a pointer to a local (`auto`) variable — its lifetime ended.
- `volatile` is for hardware/signal handlers; `_Atomic` is for inter-thread state.
- For wire formats, never use bit fields — use shifts/masks.
- Build a release binary with `-O2 -g -DNDEBUG` so production crashes still have symbols.
- `_FORTIFY_SOURCE=2` requires `-O1` or higher to take effect.
- Place static analysis (clang-tidy, cppcheck, gcc -fanalyzer) in CI.
- For new code, treat `gcc -std=c11 -Wall -Wextra -Werror -Wpedantic` as the floor.

## See Also

- rust, go, make, python, lua, bash, regex, polyglot, webassembly, java, javascript, typescript, ruby

## References

- [C Reference (cppreference.com)](https://en.cppreference.com/w/c) -- exhaustive language and library reference
- [The C Programming Language, 2nd ed. (K&R)](https://en.wikipedia.org/wiki/The_C_Programming_Language) -- Kernighan and Ritchie, the canonical introduction
- [Modern C, Jens Gustedt](https://hal.inria.fr/hal-02383654) -- free, comprehensive, current with C17/C23
- [ISO/IEC 9899:2018 (C17)](https://www.iso.org/standard/74528.html) -- current ratified standard
- [C23 Working Draft (N3220)](https://www.open-std.org/jtc1/sc22/wg14/www/docs/n3220.pdf) -- final WG14 draft
- [C11 Standard Draft (N1570)](https://www.open-std.org/jtc1/sc22/wg14/www/docs/n1570.pdf) -- ISO/IEC 9899:2011 final draft
- [GCC Manual](https://gcc.gnu.org/onlinedocs/gcc/) -- compiler flags, extensions, built-ins
- [Clang Documentation](https://clang.llvm.org/docs/) -- LLVM C compiler, diagnostics, sanitizers
- [GNU C Library (glibc) Manual](https://www.gnu.org/software/libc/manual/) -- POSIX and GNU libc functions
- [POSIX.1-2017 (IEEE Std 1003.1)](https://pubs.opengroup.org/onlinepubs/9699919799/) -- POSIX system interfaces
- [Linux man-pages](https://man7.org/linux/man-pages/) -- syscall and libc references
- [SEI CERT C Coding Standard](https://wiki.sei.cmu.edu/confluence/display/c/SEI+CERT+C+Coding+Standard) -- secure coding rules
- [MISRA C](https://misra.org.uk/) -- safety-critical C subset (auto/aerospace/medical)
- [Valgrind Documentation](https://valgrind.org/docs/manual/manual.html) -- memory debugger and profiler
- [AddressSanitizer Wiki](https://github.com/google/sanitizers/wiki/AddressSanitizer) -- ASan/UBSan/TSan/MSan documentation
- [Compiler Explorer (Godbolt)](https://godbolt.org/) -- interactive disassembly and multi-compiler comparison
- [man gcc](https://man7.org/linux/man-pages/man1/gcc.1.html) -- GCC manual
- [man ld](https://man7.org/linux/man-pages/man1/ld.1.html) -- GNU linker
- [man dlopen](https://man7.org/linux/man-pages/man3/dlopen.3.html) -- dynamic loading
- [man pthreads](https://man7.org/linux/man-pages/man7/pthreads.7.html) -- POSIX threads overview
- [man signal](https://man7.org/linux/man-pages/man7/signal.7.html) -- signal handling
