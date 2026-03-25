# C (Systems Programming Language)

> Low-level, general-purpose language with direct memory access, minimal runtime, and the foundation of most operating systems.

## Pointers

### Pointer Basics

```c
int x = 42;
int *p = &x;        // p holds the address of x
int val = *p;        // Dereference: val = 42
*p = 100;            // Modify x through p

int **pp = &p;       // Pointer to pointer
int val2 = **pp;     // Double dereference
```

### Pointer Arithmetic

```c
int arr[] = {10, 20, 30, 40, 50};
int *p = arr;        // Points to arr[0]

*(p + 2);            // arr[2] = 30 (advances by sizeof(int) * 2)
p++;                 // Now points to arr[1]
p[3];                // arr[4] = 50 (subscript notation)

// Difference between pointers (number of elements)
ptrdiff_t diff = &arr[4] - &arr[1];  // 3
```

### Function Pointers

```c
// Declaration
int (*compare)(int, int);

// Assignment
int max(int a, int b) { return a > b ? a : b; }
compare = max;

// Call through pointer
int result = compare(3, 7);  // 7

// Typedef for clarity
typedef int (*Comparator)(const void *, const void *);

// Callback pattern (qsort)
int cmp_int(const void *a, const void *b) {
    return *(const int *)a - *(const int *)b;
}
qsort(arr, 5, sizeof(int), cmp_int);
```

### Void Pointers

```c
void *generic = &x;           // Can hold any pointer type
int *ip = (int *)generic;     // Must cast before dereferencing

// Common in malloc
int *arr = (int *)malloc(10 * sizeof(int));
```

## Memory Management

### malloc / calloc / realloc / free

```c
#include <stdlib.h>

// malloc — uninitialized memory
int *a = malloc(10 * sizeof(int));
if (!a) { perror("malloc"); exit(1); }

// calloc — zero-initialized memory
int *b = calloc(10, sizeof(int));   // 10 ints, all zero

// realloc — resize allocation
int *c = realloc(a, 20 * sizeof(int));
if (!c) { perror("realloc"); exit(1); }
a = c;  // realloc may return a new address

// free — release memory
free(a);
a = NULL;  // Prevent dangling pointer

// Common pattern: allocate struct
typedef struct { int x, y; } Point;
Point *p = malloc(sizeof(Point));
p->x = 10;
p->y = 20;
free(p);
```

### Common Pitfalls

```c
// Double free
free(ptr);
free(ptr);          // Undefined behavior

// Use after free
free(ptr);
ptr->field = 1;     // Undefined behavior

// Memory leak
ptr = malloc(100);
ptr = malloc(200);  // Original 100 bytes leaked

// Buffer overflow
char buf[10];
strcpy(buf, "this is way too long");  // Overflow
```

## Strings

### String Functions

```c
#include <string.h>

strlen(s);                    // Length (excluding \0)
strcpy(dst, src);             // Copy (unsafe — no bounds check)
strncpy(dst, src, n);         // Copy at most n bytes
strcat(dst, src);             // Concatenate
strncat(dst, src, n);         // Concatenate at most n bytes
strcmp(a, b);                  // Compare: <0, 0, >0
strncmp(a, b, n);             // Compare first n bytes
strchr(s, c);                 // Find first occurrence of c
strrchr(s, c);                // Find last occurrence of c
strstr(haystack, needle);     // Find substring
strtok(s, delim);             // Tokenize (modifies string)

// Safer alternatives
snprintf(buf, sizeof(buf), "Hello %s", name);  // Bounded sprintf
```

### String Tokenization

```c
char line[] = "one,two,three";
char *tok = strtok(line, ",");
while (tok) {
    printf("%s\n", tok);
    tok = strtok(NULL, ",");   // NULL continues previous string
}
```

## Preprocessor

### Macros and Conditionals

```c
#define PI 3.14159
#define MAX(a, b) ((a) > (b) ? (a) : (b))   // Parenthesize everything
#define ARRAY_SIZE(arr) (sizeof(arr) / sizeof((arr)[0]))

#include <stdio.h>      // System header
#include "myheader.h"   // Local header

// Conditional compilation
#ifdef DEBUG
    printf("debug: x = %d\n", x);
#endif

#ifndef HEADER_H
#define HEADER_H
// ... header contents (include guard)
#endif

// Pragma once (non-standard but widely supported)
#pragma once

// Stringification and token pasting
#define STR(x) #x               // STR(hello) -> "hello"
#define CONCAT(a, b) a##b       // CONCAT(var, 1) -> var1

// Predefined macros
__FILE__    // Current filename
__LINE__    // Current line number
__func__    // Current function name
__DATE__    // Compilation date
__TIME__    // Compilation time
```

## Structs, Unions, Enums

### Structs

```c
struct Point {
    int x;
    int y;
};

// Typedef
typedef struct {
    char name[64];
    int age;
} Person;

// Designated initializers (C99)
Person p = { .name = "Alice", .age = 30 };

// Arrow operator (pointer to struct)
Person *pp = &p;
pp->age = 31;

// Nested structs
typedef struct {
    Point origin;
    int width, height;
} Rect;
```

### Unions

```c
typedef union {
    int i;
    float f;
    char bytes[4];
} Value;

Value v;
v.i = 42;
// v.f is now undefined — only one member valid at a time

// Tagged union pattern
typedef struct {
    enum { INT, FLOAT, STRING } type;
    union {
        int i;
        float f;
        char *s;
    } data;
} Variant;
```

### Enums

```c
enum Color { RED, GREEN, BLUE };          // 0, 1, 2
enum Status { OK = 200, NOT_FOUND = 404 };

// Typedef
typedef enum { FALSE, TRUE } Bool;
```

## Bitwise Operations

```c
a & b       // AND
a | b       // OR
a ^ b       // XOR
~a          // NOT (complement)
a << n      // Left shift (multiply by 2^n)
a >> n      // Right shift (divide by 2^n)

// Common patterns
x |= (1 << n);       // Set bit n
x &= ~(1 << n);      // Clear bit n
x ^= (1 << n);       // Toggle bit n
(x >> n) & 1;        // Check bit n
x & (x - 1);         // Clear lowest set bit
__builtin_popcount(x); // Count set bits (GCC)
```

## File I/O

```c
#include <stdio.h>

// Open / close
FILE *fp = fopen("data.txt", "r");    // r, w, a, rb, wb, ab, r+, w+, a+
if (!fp) { perror("fopen"); exit(1); }
fclose(fp);

// Character I/O
int ch = fgetc(fp);
fputc('A', fp);

// Line I/O
char line[256];
fgets(line, sizeof(line), fp);   // Reads up to sizeof(line)-1 chars
fputs("hello\n", fp);

// Formatted I/O
fprintf(fp, "value: %d\n", 42);
fscanf(fp, "%d", &val);

// Binary I/O
size_t n = fread(buf, sizeof(int), count, fp);
fwrite(buf, sizeof(int), count, fp);

// Seek
fseek(fp, 0, SEEK_SET);    // Beginning
fseek(fp, 0, SEEK_END);    // End
long pos = ftell(fp);       // Current position
rewind(fp);                 // Reset to beginning
```

## Common Patterns

### Error Handling

```c
// Return codes with goto cleanup
int process_file(const char *path) {
    int ret = -1;
    FILE *fp = fopen(path, "r");
    if (!fp) goto cleanup;

    char *buf = malloc(1024);
    if (!buf) goto close_file;

    // ... work ...
    ret = 0;

    free(buf);
close_file:
    fclose(fp);
cleanup:
    return ret;
}
```

### Flexible Array Member (C99)

```c
typedef struct {
    size_t len;
    char data[];    // Must be last member
} Buffer;

Buffer *buf = malloc(sizeof(Buffer) + 100);
buf->len = 100;
```

## Compilation

### GCC Flags

```bash
# Basic compilation
gcc -o program main.c

# Warnings (always use)
gcc -Wall -Wextra -Werror -o program main.c

# Debug build
gcc -g -O0 -Wall -Wextra -o program main.c

# Release build
gcc -O2 -DNDEBUG -o program main.c

# Aggressive optimization
gcc -O3 -march=native -o program main.c

# Multiple files
gcc -Wall -o program main.c utils.c -lm

# Compile to object, then link
gcc -c main.c -o main.o
gcc -c utils.c -o utils.o
gcc main.o utils.o -o program -lm

# Preprocessor output
gcc -E main.c                   # See preprocessor expansion

# Static analysis
gcc -fanalyzer main.c           # GCC 10+ static analyzer

# Address sanitizer (debug)
gcc -fsanitize=address -g -o program main.c
```

### Common Libraries

```bash
-lm         # Math library (sin, cos, sqrt)
-lpthread   # POSIX threads
-lrt        # Real-time extensions
-ldl        # Dynamic loading (dlopen)
-lcrypto    # OpenSSL crypto
-lssl       # OpenSSL SSL
```

## Tips

- Always check return values of `malloc`, `fopen`, `fread`, and system calls.
- Use `valgrind --leak-check=full ./program` to detect memory leaks and invalid accesses.
- Prefer `snprintf` over `sprintf` and `strncpy` over `strcpy` to prevent buffer overflows.
- Use `const` liberally: `const char *s` means the data is read-only; `char *const s` means the pointer is read-only.
- Compile with `-Wall -Wextra -Werror` during development to catch issues early.
- Use `sizeof(var)` instead of `sizeof(type)` to stay correct when the type changes.
- The `restrict` keyword (C99) tells the compiler pointers do not alias, enabling optimizations.

## References

- [C Reference (cppreference.com)](https://en.cppreference.com/w/c) -- standard library, language syntax, headers
- [C11 Standard Draft (N1570)](https://www.open-std.org/jtc1/sc22/wg14/www/docs/n1570.pdf) -- ISO/IEC 9899:2011 final draft
- [C23 Standard Draft (N3096)](https://www.open-std.org/jtc1/sc22/wg14/www/docs/n3096.pdf) -- latest C standard draft
- [GCC Manual](https://gcc.gnu.org/onlinedocs/gcc/) -- compiler flags, extensions, built-ins
- [Clang Documentation](https://clang.llvm.org/docs/) -- LLVM C compiler, diagnostics, sanitizers
- [GNU C Library (glibc) Manual](https://www.gnu.org/software/libc/manual/) -- POSIX and GNU libc functions
- [man gcc](https://man7.org/linux/man-pages/man1/gcc.1.html) -- GCC man page
- [man ld](https://man7.org/linux/man-pages/man1/ld.1.html) -- GNU linker
- [Valgrind Documentation](https://valgrind.org/docs/manual/manual.html) -- memory debugger and profiler
- [POSIX C Headers](https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/contents.html) -- POSIX standard headers and interfaces
- [SEI CERT C Coding Standard](https://wiki.sei.cmu.edu/confluence/display/c/SEI+CERT+C+Coding+Standard) -- secure coding rules
- [Compiler Explorer (Godbolt)](https://godbolt.org/) -- interactive disassembly and multi-compiler comparison
