# The Internals of awk — Execution Model, Implementations, and Performance

> *awk is a tiny domain-specific language hiding a complete general-purpose programming language. Its execution model is a disciplined record-loop machine: read a line, split it into fields, evaluate every pattern, fire matching actions. Underneath the one-liner facade lives a hash-table runtime, a regex engine, an associative-array memory model, and a calling convention with no real local variables. Three serious implementations — bwk, gawk, and mawk — share a POSIX core but diverge by an order of magnitude in speed and feature set.*

---

## 1. The awk Execution Model

awk's surface syntax is a sequence of `pattern { action }` rules. Its execution model is the implicit main loop that ties them together.

### 1.1 The Three-Phase Pipeline

Every awk program executes in three phases, in this strict order:

```awk
BEGIN { initialization }      # Phase 1: before any input
pattern { action }            # Phase 2: per-record main loop
END { finalization }          # Phase 3: after all input
```

The `BEGIN` block runs before a single byte is read. The main rules form the body of an implicit loop the interpreter wraps around the input. The `END` block runs after EOF on the last input source.

You can have multiple `BEGIN` and `END` blocks; they execute in source order, concatenated as if they were one. Multiple `BEGIN` blocks allow library files (`-f lib.awk -f main.awk`) to each contribute initialization without trampling each other.

### 1.2 The Implicit Main Loop

If you stripped the syntactic sugar, awk's runtime is approximately this C-pseudocode:

```c
run_BEGIN_blocks();
while (read_next_record(&record)) {
    NR++;
    FNR++;
    split_into_fields(record, FS);   // populates $0, $1..$NF
    for (each pattern_action_pair p) {
        if (eval_pattern(p, record)) {
            execute_action(p);
        }
    }
}
run_END_blocks();
```

The `read_next_record` call respects `RS` (record separator, default `\n`). After splitting, `$0` holds the entire raw record and `$1..$NF` hold the fields. Every pattern is evaluated against this snapshot before the next record is read.

### 1.3 The Per-Record Pipeline

For each record the interpreter performs five micro-steps:

1. **Read** a record up to `RS`.
2. **Strip** the trailing `RS` (it's a separator, not a terminator).
3. **Split** the record into fields per `FS`, populating `$0`, `$1..$NF`, and `NF`.
4. **Match** every pattern in source order.
5. **Execute** every matched action; `next` aborts the rest of the pipeline for the current record; `exit` aborts the whole loop.

The control words `next`, `nextfile`, and `exit` short-circuit at different levels of this loop:

```awk
NR == 1 { next }              # skip header — abandon current record
$1 ~ /^#/ { next }            # skip comments
/STOP/   { exit }             # abort entire main loop, run END
FNR == 1 && NR > 1 { nextfile } # process only first file
```

### 1.4 Convergence with sed's Pattern Space

sed's model is `read into pattern space → apply commands → print → repeat`. awk's model is `read into $0 → split into fields → fire matching actions → repeat`. The difference is that sed's primary state is a single buffer (the pattern space) and its commands are byte-oriented; awk's primary state is the field vector plus arbitrary user variables and arrays, and its actions are full expressions.

| Concept | sed | awk |
|:--------|:----|:----|
| Per-record buffer | pattern space | `$0` plus `$1..$NF` |
| Auxiliary buffer | hold space | any variable or array |
| Action language | s/y/d/p/N/G/H/etc | full expression language |
| Pattern matching | address ranges, regex | regex, expressions, ranges |
| Auto-print | yes (default) | only via `print`/`printf` |

awk and sed implement the same abstract pipeline; awk's action grammar is a strict superset.

### 1.5 Auto-Print and the "Empty Action" Convention

A pattern with no action defaults to `{ print }`. An action with no pattern fires on every record.

```awk
/error/                        # equivalent to: /error/ { print }
{ print NR, $0 }               # equivalent to:    1   { print NR, $0 }
```

This is the conceptual basis for the awk one-liner: `awk '/pat/' file` is a complete program that prints matching lines.

---

## 2. Implementations Compared

There is no single "awk." There are several implementations descended from the original 1977 paper. They share a POSIX core; they diverge sharply on extensions, performance, and bug surface.

### 2.1 The Family Tree

| Year | Name | Source | Notes |
|:-----|:-----|:-------|:------|
| 1977 | awk | Aho–Weinberger–Kernighan | Original, in V7 Unix |
| 1985 | nawk | "new awk" | Rewrite, added user functions and dynamic regex |
| 1988 | gawk | GNU project | Free implementation, eventually full superset |
| 1991 | mawk | Mike Brennan | Pure-speed bytecode interpreter |
| 1996 | bwk | Brian Kernighan | "one true awk", the maintained nawk |
| 2018+ | goawk | Ben Hoyt | Go reimplementation, CSV mode |
| 2020+ | frawk | Eli Rosenthal | Rust implementation, JIT, parallel |

### 2.2 The "One True awk" — bwk

bwk is the ongoing maintenance of the original Bell Labs awk by Brian Kernighan (the K in AWK). Its source tree is `onetrueawk/awk` on GitHub. It is the reference implementation for POSIX awk and ships as `/usr/bin/awk` on macOS, the BSDs, and historically on most Unix systems. Conservative; small; portable; fast enough for everything except the largest inputs. Adds essentially nothing beyond POSIX, but Kernighan still merges fixes.

### 2.3 GNU awk — gawk

gawk is the GNU implementation. It's the default `awk` on most Linux distributions (often via the `/usr/bin/awk → gawk` symlink). It's a strict superset of POSIX awk and ships extensions in roughly four buckets: language extensions (multi-dim arrays, namespaces, indirect calls), regex extensions (`\<`, `\>`, `[[:class:]]`), I/O extensions (TCP/UDP via `/inet/`, two-way pipes via `|&`), and library/extension API (loadable C extensions, gettext support).

### 2.4 mawk

mawk by Mike Brennan compiles to a tight bytecode and uses a hand-tuned regex engine. It is the smallest, fastest awk — typically 2–5x faster than gawk on classic field-and-arithmetic workloads. Common on Debian/Ubuntu (`/etc/alternatives/awk → mawk` historically). It implements POSIX awk plus a few extensions but is famously *not* a gawk superset; many gawk programs do not run on mawk.

### 2.5 busybox awk

busybox bundles a tiny awk for embedded systems. It is roughly POSIX-compliant, tiny, and often the only awk on a router or initramfs. Slower than mawk, less complete than gawk; do not target it from desktop workflows.

### 2.6 What gawk Has That POSIX Doesn't

| Feature | POSIX | gawk | mawk | bwk |
|:--------|:-----:|:----:|:----:|:---:|
| `gensub()` (returns substituted string) | no | yes | no | no |
| True multi-dim arrays `a[i][j]` | no | yes | no | no |
| `length(array)` returns element count | no | yes | yes | yes |
| `delete array` (whole array) | yes (POSIX 2008) | yes | yes | yes |
| `\<` `\>` word anchors | no | yes | no | no |
| `[[:alpha:]]` character classes | yes | yes | yes | yes |
| `PROCINFO[]` | no | yes | no | no |
| Two-way pipe `\|&` | no | yes | no | no |
| `/inet/tcp/lport/host/rport` | no | yes | no | no |
| `@function` indirect call | no | yes | no | no |
| `@namespace` (gawk 5.0+) | no | yes | no | no |
| `@include` library files | no | yes | no | no |
| `@load` C extensions | no | yes | no | no |
| `systime()`, `strftime()`, `mktime()` | no | yes | partial | no |
| `--profile` / `pgawk` | no | yes | no | no |
| `--lint` strictness | no | yes | partial | no |
| Bignum integers (`-M`) | no | yes | no | no |

### 2.7 Picking an Implementation

| Goal | Pick |
|:-----|:-----|
| Portable script that must run anywhere | POSIX subset, test on bwk |
| Maximum throughput on big files | mawk |
| Sophisticated logic, time math, networking | gawk |
| CSV with quoted commas | gawk `--csv` (5.3+), goawk, frawk |
| Multi-core | frawk |
| Embedded/initramfs/busybox | busybox awk, but write portable code |

---

## 3. Field Splitting

Field splitting is the single most subtle part of awk's runtime. It happens for every record and is governed by `FS`, `FIELDWIDTHS`, or `FPAT`, in priority order from last set.

### 3.1 The Four FS Modes

```awk
FS=" "       # default: split on runs of whitespace, trim leading/trailing
FS="\t"      # single non-space char: split on each occurrence (no merging)
FS=","       # same — empty fields preserved
FS=":+"      # multi-char: treat FS as ERE regex
FS=""        # gawk: split each character into its own field
```

The default `FS=" "` is special: it merges runs of spaces and tabs into a single separator and trims leading/trailing whitespace. The moment you set `FS=" "` to anything else — even a single space written as `FS="[ ]"` — that special behavior is lost.

### 3.2 The Splitting Algorithm

```c
// pseudocode for a generic split
split_record(record, FS) {
    if (FS == " ")
        return whitespace_split(record);   // special path
    if (length(FS) == 1)
        return char_split(record, FS);     // exact-byte split
    return regex_split(record, FS);        // regex path
}
```

The whitespace path is hand-tuned in every implementation. The regex path constructs (or caches) a compiled regex and walks the record finding separator matches. Empty matches are pinned to advance the cursor by one; this is the rule that makes `FS=""` produce one field per character in gawk.

### 3.3 FIELDWIDTHS for Fixed-Width Records

gawk and frawk support `FIELDWIDTHS`, a whitespace-separated list of widths:

```awk
# Fixed-width: 5 cols name, 3 cols code, 7 cols value
BEGIN { FIELDWIDTHS = "5 3 7" }
{ print "name=" $1 ", code=" $2 ", value=" $3 }
```

A `*` width consumes the rest of the line. Use this for column-aligned output from older mainframe extracts where column boundaries don't have separators.

### 3.4 FPAT — "Match the Field Itself"

gawk's `FPAT` flips the model: instead of describing the separator, you describe the field. This is essential for CSV with quoted commas:

```awk
# Match either: a non-comma run, OR a quoted string with embedded commas
BEGIN { FPAT = "([^,]+)|(\"[^\"]+\")" }
{ for (i=1; i<=NF; i++) print i, $i }
```

When both `FS` and `FPAT` are usable, the last one set wins. gawk 5.3 added `--csv` which is even better than `FPAT` for proper RFC 4180 CSV, but `FPAT` remains the portable gawk solution.

### 3.5 The $0 Reconstruction Rule

Assigning to a field forces awk to rebuild `$0` using `OFS`:

```awk
{ $1 = $1; print }            # canonical "renormalize whitespace"
# Input:  "  hello   world  "
# Output: "hello world"
```

The trick is: even assigning a field to itself triggers reconstruction. This is the canonical way to convert tab-delimited input to space-delimited output, or to enforce a consistent `OFS`. The reconstruction runs in O(N) over the field count and concatenates fields with `OFS`.

Conversely, assigning to `$0` triggers a re-split. This pattern resets the field vector after manual mangling:

```awk
{ $0 = tolower($0); print $1, NF }   # lowercase, then re-split
```

### 3.6 NF Manipulation

`NF` is read/write. Assigning to `NF` truncates or extends the field vector:

```awk
{ NF = 3; print }             # keep only first three fields, re-emit with OFS
{ NF--; print }               # drop the last field
{ $5 = "extra"; print }       # auto-extends NF if needed; pads with empty
```

Increasing `NF` past the current end fills intermediate fields with empty strings. This is a portable way to pad records to a fixed width.

---

## 4. Regex Engine in awk

awk regex is POSIX Extended Regular Expressions (ERE) plus optional implementation extras.

### 4.1 The POSIX ERE Core

| Construct | Meaning |
|:----------|:--------|
| `.` | any single character |
| `^` `$` | anchors |
| `[abc]` `[^abc]` | character class / negated |
| `*` `+` `?` | greedy quantifiers |
| `{n}` `{n,}` `{n,m}` | bounded repetition |
| `\|` | alternation |
| `()` | grouping |
| `\1`..`\9` | back-reference (POSIX BRE only — ERE doesn't standardize) |

Note: POSIX ERE famously does not require backreferences. gawk supports them in the regex engine but the substitution string `\1` form is supported in `gensub()` only — `sub()` and `gsub()` use `&` for the whole match.

### 4.2 gawk Regex Extensions

| Construct | Meaning | gawk | mawk | bwk |
|:----------|:--------|:----:|:----:|:---:|
| `\<` `\>` | word boundaries (start/end) | yes | no | no |
| `\y` | word boundary (either side) | yes | no | no |
| `\B` | not-a-word-boundary | yes | no | no |
| `\s` `\S` | whitespace / non-whitespace | yes | no | no |
| `\w` `\W` | word char / non-word char | yes | no | no |
| `[[:alpha:]]` etc | POSIX classes | yes | yes | yes |

If you need word-boundary matching portably, fall back to `[^[:alnum:]_]` patterns instead of `\<`/`\>`.

### 4.3 Dynamic vs Literal Regex

A regex literal lives between slashes: `/error/`. Inside the program text it's compiled once.

A dynamic regex is a string used as a pattern: `match($0, "err" suffix)`. The string is compiled every time the expression is evaluated, then cached if the implementation is clever. mawk caches; gawk caches; bwk caches less aggressively.

```awk
# Literal — compiled once
$0 ~ /error/

# Dynamic — string-valued pattern
pat = "error|warn"
$0 ~ pat

# Anti-pattern: rebuilding the pattern every record
{ if ($0 ~ ("err" $1)) print }    # must compile per-record if $1 varies
```

Use literal regex when the pattern is fixed; use dynamic only when the pattern varies. For variable patterns inside hot loops, hoist the regex outside if possible:

```awk
BEGIN { pat = "^[0-9]{3}-[0-9]{4}$" }
$1 ~ pat { print }
```

### 4.4 The `/pat/` in Pattern Context

A bare `/pat/` in pattern position is shorthand for `$0 ~ /pat/`:

```awk
/error/ { print }              # equivalent to:
$0 ~ /error/ { print }
```

But a bare `/pat/` in *expression* position evaluates to the boolean `$0 ~ /pat/`:

```awk
{ if (/error/) print "found" } # /error/ here means $0 ~ /error/
```

Inside `match()`, `sub()`, `gsub()`, the regex argument is its own thing — it does not implicitly apply to `$0`. This is a frequent stumble for beginners.

### 4.5 Regex Performance

gawk's regex engine is general-purpose and uses an Aho–Corasick-style fast path for fixed strings; mawk's engine is hand-tuned for the common case. Pathological backtracking cases (nested quantifiers, alternation explosions) are rare in awk because POSIX ERE doesn't have features like lookahead, but they exist.

```awk
# Pathological: alternation with overlapping prefixes
$0 ~ /^(a|aa|aaa|aaaa|aaaaa)+b$/   # avoid

# Equivalent, faster
$0 ~ /^a+b$/
```

For raw substring search, `index()` is faster than a regex match because it skips compilation:

```awk
# Fast — no regex compile
index($0, "ERROR") > 0 { print }

# Slower — compiles a regex
$0 ~ "ERROR" { print }
```

---

## 5. Variable Types — Internals

awk has a single dynamic type called "scalar." A scalar is simultaneously a string and a number; conversion is implicit and context-dependent.

### 5.1 The Dual Representation

Internally, an awk scalar holds both representations or recomputes one on demand. The conceptual model:

```c
struct Cell {
    char *str;          // string form (or NULL until needed)
    double num;         // numeric form (or 0 until needed)
    enum { STR, NUM, BOTH } valid;
};
```

A value created by `x = "42"` starts as STR. When you do `x + 1`, it becomes BOTH (string `"42"` plus number `42`). Once both forms are valid, awk picks based on context.

### 5.2 The String-vs-Number Equality Trap

The canonical gotcha:

```awk
BEGIN {
    a = "10"
    b = 10
    print (a == b)            # 1 (true) — numeric comparison wins
    print (a == "10.0")       # 0 (false) — string compare
    print (10 == "10.0")      # 1 (true) — numeric

    # The booby trap:
    if (input == "0")  ...    # may fail if input is "00" or "0.0"
    if (input + 0 == 0) ...   # robust numeric test
}
```

The rule POSIX specifies: if **either** operand is a number with a strictly numeric provenance (literal, arithmetic result, field that looks numeric), the comparison is numeric; otherwise it's string. Fields read from input are "looks numeric" if they parse cleanly as a number. This rule is responsible for half of all awk surprises.

### 5.3 Forcing Numeric or String Context

```awk
n + 0              # force numeric — common idiom
"" n               # force string — concatenate empty string
n ""               # equivalent — concatenate empty string
sprintf("%d", n)   # force string with explicit format
```

The `+0` and `""` idioms are zero-cost in the sense that they don't allocate beyond the conversion. They are the canonical hammer for type ambiguity:

```awk
$1+0 < $2+0 { print }     # numeric compare even if fields look stringy
"" $3 == "" $4 { print }  # string compare even if fields look numeric
```

### 5.4 OFMT vs CONVFMT

`OFMT` controls number-to-string conversion in `print` (default `"%.6g"`). `CONVFMT` controls all *other* number-to-string conversions (also `"%.6g"`).

```awk
BEGIN {
    x = 1/3
    print x                  # uses OFMT     → "0.333333"
    s = "" x                 # uses CONVFMT  → "0.333333"
    OFMT = "%.2f"
    print x                  # → "0.33"
    CONVFMT = "%.10f"
    s = "" x                 # → "0.3333333333"
}
```

Both default to `"%.6g"`. POSIX requires that `OFMT` apply to `print` only, but mawk historically conflated them; rely on the spec for portability. Setting `OFMT = "%.20g"` is the canonical "stop losing precision" knob.

### 5.5 Integer vs Float

awk has no integer type; all numbers are IEEE 754 doubles. This means 53 bits of integer precision (up to `2^53 = 9007199254740992`). Beyond that, integer arithmetic loses precision:

```awk
BEGIN { print 9007199254740993 + 1 }     # likely 9007199254740993
BEGIN { print 9007199254740993 + 2 }     # 9007199254740994
```

gawk's `-M` (bignum) flag swaps the numeric representation for MPFR/GMP. Cost: 2-10x slowdown for arithmetic; benefit: arbitrary precision.

```bash
gawk -M 'BEGIN { print 2^200 }'
# 1606938044258990275541962092341162602522202993782792835301376
```

### 5.6 Uninitialized Variables

An uninitialized variable is the empty string `""`, which converts to 0 in numeric context and `""` in string context. This is why `count[key]++` works on first touch:

```awk
{ count[$1]++ }       # creates entry with value 1 the first time
END { for (k in count) print k, count[k] }
```

The `++` reads the (uninitialized → 0) value, increments it, stores 1.

---

## 6. Associative Arrays

Arrays are awk's only collection type. They are hash tables keyed by strings.

### 6.1 The Memory Model

```c
struct AwkArray {
    HashTable *table;     // string → Cell mapping
    size_t count;
};
```

All keys are stored as strings. If you write `arr[42]`, the key is the string `"42"`. If you set `OFMT = "%.2f"` and write `arr[3.14159]`, the key is `"3.14"` — the same `CONVFMT` rule as scalar conversion.

Arrays auto-vivify: reading `arr[k]` for a missing key materializes an empty entry in some implementations (gawk before 5.0 did this). In gawk 5.0+, `in` is the safe membership test that does not vivify:

```awk
# Safe — does not create entry
if (k in arr) print arr[k]

# Unsafe — may auto-vivify
if (arr[k] != "") print arr[k]
```

### 6.2 Multi-Dimensional Arrays

POSIX awk simulates multi-dim with `SUBSEP`:

```awk
arr[i, j] = v
# Stored as: arr[i SUBSEP j] = v
# Default SUBSEP = "\034" (FS, the ASCII record separator)
```

The membership test follows the same convention:

```awk
if ((i, j) in arr) ...
```

This breaks if your data contains `SUBSEP`. Override it for safety:

```awk
BEGIN { SUBSEP = "\x1f" }     # use ASCII unit separator
```

### 6.3 True Multi-Dim in gawk

gawk 4.0+ has true nested arrays — values can themselves be arrays:

```awk
BEGIN {
    a[1][1] = "x"
    a[1][2] = "y"
    a[2][1] = "z"
    for (i in a)
        for (j in a[i])
            print i, j, a[i][j]
    # delete sub-array
    delete a[1]
}
```

Nested arrays are first-class: you can pass them to functions, return them, and `delete` whole subtrees. Other awks have nothing equivalent.

### 6.4 The `in` Operator

```awk
if (k in arr)       { ... }    # membership
if (!(k in arr))    { ... }    # non-membership
if ((i, j) in arr)  { ... }    # multi-dim membership
for (k in arr)      { ... }    # iteration
```

`in` is the only safe way to test membership without auto-vivifying.

### 6.5 Iteration Order

POSIX explicitly leaves `for (k in arr)` order unspecified. In practice:

| awk | Default iteration order |
|:----|:------------------------|
| bwk | insertion-order-ish, hash-perturbed |
| gawk | hash-bucket order |
| mawk | hash-bucket order |

Hash-bucket order can change between awk runs and definitely changes between implementations. **Never rely on iteration order without sorting.**

gawk lets you fix the order via `PROCINFO`:

```awk
BEGIN { PROCINFO["sorted_in"] = "@val_num_desc" }
END   { for (k in count) print k, count[k] }
```

Available `sorted_in` values:

| Setting | Effect |
|:--------|:-------|
| `@unsorted` | default; implementation order |
| `@ind_str_asc` | by key, string-comparing, ascending |
| `@ind_str_desc` | by key, string-comparing, descending |
| `@ind_num_asc` / `_desc` | by key, numerically |
| `@val_str_asc` / `_desc` | by value, string-comparing |
| `@val_num_asc` / `_desc` | by value, numerically |
| custom function name | user-supplied comparator |

In bwk/mawk, you must pipe through `sort` or load into a numeric-keyed array and iterate by index:

```awk
END {
    n = 0
    for (k in count) keys[++n] = k
    # sort the keys array — but awk has no sort builtin in POSIX
    # so pipe through sort:
    for (i = 1; i <= n; i++) print keys[i] | "sort"
}
```

### 6.6 Deletion

```awk
delete arr[k]              # remove single entry
delete arr                 # remove every entry, keep the array
delete arr[i][j]           # gawk: delete leaf
delete arr[i]              # gawk: delete whole subtree
```

`delete arr` (whole-array form) was a gawk extension until POSIX 2008 standardized it. Old awks may reject it.

---

## 7. Control Flow and Function Calling

awk has C-style control flow (`if`, `while`, `for`, `do/while`, `break`, `continue`) plus its own pattern-matching control verbs (`next`, `nextfile`, `exit`).

### 7.1 The Operator-Stack Model

awk programs compile (in mawk and gawk) to a stack-based bytecode. Each expression evaluates left-to-right, pushing operands and applying operators. There's no register pressure — the model is the same as Forth or PostScript at this level.

This matters for one reason: function calls preserve the stack across recursive invocations. Recursion is supported and reasonably efficient.

```awk
function fact(n) {
    return (n <= 1) ? 1 : n * fact(n - 1)
}
BEGIN { print fact(20) }      # 2432902008176640000
```

### 7.2 The Function Definition Syntax

```awk
function name(arg1, arg2,    local1, local2) {
    # body
    return value
}
```

Variables listed before the extra-spaces convention are real parameters; variables listed after are locals — really they are *also* parameters, just ones you never pass at the call site, exploiting the rule that uninitialized parameters default to empty/zero.

```awk
# No "var" or "let" keyword exists. This is the convention:
function median(arr, n,         i, mid, sorted) {
    # i, mid, sorted are locals
    for (i = 1; i <= n; i++) sorted[i] = arr[i]
    asort(sorted)               # gawk
    mid = int(n/2)
    return (n % 2) ? sorted[mid+1] : (sorted[mid] + sorted[mid+1]) / 2
}
```

The "extra spaces" between the real and local parameters are a *convention* enforced only by humans reading the code. awk doesn't care; lint tools do.

### 7.3 Pass Semantics

Scalars are passed **by value**. Arrays are passed **by reference**. This is the only "complex" part of awk's calling convention:

```awk
function inc_scalar(x) { x++ }              # local x — caller unaffected
function fill(arr)     { arr[1] = "yes" }   # mutates caller's array

BEGIN {
    a = 10; inc_scalar(a); print a          # 10
    fill(b); print b[1]                     # yes
}
```

Functions cannot return arrays in POSIX awk; they can in gawk via nested arrays.

### 7.4 Recursion

awk recursion is straightforward. The interpreter maintains its own call stack. Stack depth is limited by the host C stack — typically thousands of frames. No tail-call optimization, but iterative rewrites are usually trivial.

```awk
function gcd(a, b) {
    return b == 0 ? a : gcd(b, a % b)
}
BEGIN { print gcd(252, 105) }    # 21
```

### 7.5 gawk's Indirect Calls — `@function`

gawk lets you call functions by name string:

```awk
function add(a, b) { return a + b }
function mul(a, b) { return a * b }

BEGIN {
    op = "add"
    print @op(2, 3)              # calls add(2, 3) → 5
    op = "mul"
    print @op(2, 3)              # calls mul(2, 3) → 6
}
```

This is gawk's only mechanism for first-class function references. Useful for dispatch tables; unavailable in mawk and bwk.

### 7.6 Built-in Control Verbs

| Verb | Effect |
|:-----|:-------|
| `next` | abort current record, read next |
| `nextfile` | abort current file, open next |
| `exit [n]` | abort main loop, run END, exit with status `n` |
| `getline` | (see I/O section) — read another record |
| `return [expr]` | return from function |
| `break` / `continue` | C-style loop control |

`exit` inside `BEGIN` skips the main loop entirely but still runs `END`. `exit` inside `END` skips remaining `END` blocks. A second `exit` inside `END` simply does nothing further.

---

## 8. I/O Internals

awk's I/O is built around `getline`, `print`, `printf`, redirection, and pipes. Each is a small primitive; combining them gives the full machinery.

### 8.1 The getline Forms

There are six distinct forms of `getline`. They differ in source and destination:

| Form | Source | Destination | Updates |
|:-----|:-------|:------------|:--------|
| `getline` | current input | `$0`, `NF` | `NR`, `FNR` |
| `getline var` | current input | `var` | `NR`, `FNR` |
| `getline < file` | file | `$0`, `NF` | nothing else |
| `getline var < file` | file | `var` | nothing else |
| `cmd \| getline` | command stdout | `$0`, `NF` | nothing else |
| `cmd \| getline var` | command stdout | `var` | nothing else |

Each form returns 1 (success), 0 (EOF), or -1 (error).

### 8.2 The Bare `getline`

`getline` with no operands reads the next record from the *current* input source (file or stdin), bypassing the main pattern-action loop.

```awk
# Skip the next line whenever a "begin block" marker is seen
/^BEGIN_BLOCK$/ { getline; next }
```

This advances `NR` and `FNR`, and re-splits into fields. The main loop continues from the record *after* the one fetched by `getline`.

### 8.3 Reading From a Specific File

```awk
# Read a sidecar file once in BEGIN
BEGIN {
    while ((getline line < "config.txt") > 0)
        cfg[++n] = line
    close("config.txt")
}
```

You must `close()` the file when you're done reading; otherwise the file descriptor leaks until awk exits. This matters for programs that read many sidecar files.

### 8.4 Reading From a Command

```awk
BEGIN {
    "date +%Y-%m-%d" | getline today
    close("date +%Y-%m-%d")
}
{ print today, $0 }
```

The string "date +%Y-%m-%d" is the pipe key — awk uses the exact command string as the dictionary key for the open pipe. If you call `close()`, you must use the same string.

### 8.5 The `print | "cmd"` Pipe

```awk
{ print $0 | "sort -u" }       # send every line to sort -u
END { close("sort -u") }       # without close(), output may be buffered
```

awk opens the pipe lazily on the first `print | "cmd"` and reuses it on subsequent calls. `close()` flushes and reaps the child. Without `close()`, awk closes pipes on exit, but output ordering may be confused if other writes happen after.

### 8.6 The `system()` Fire-and-Forget

```awk
{ system("logger " $0) }       # synchronous; awk waits for completion
```

`system()` runs a shell command, waits for it to finish, and returns the exit status. It does not capture output (use `cmd | getline` for that).

### 8.7 Two-Way Pipes — gawk `|&`

```awk
# Talk to a coprocess
cmd = "tr a-z A-Z"
print "hello" |& cmd
cmd |& getline result
print result                   # HELLO
close(cmd)
```

`|&` is gawk-only. The same string serves as both write target and read source. Pair with `close()` carefully — `close(cmd, "to")` half-closes the write side without closing the read side, which the coprocess may need to terminate gracefully.

### 8.8 Closing Files and Pipes

Every distinct redirection target is an entry in awk's "open files" table. Resources accumulate until you `close()` or the process exits. Implementations differ on the per-process file descriptor limit; mawk is stingier than gawk. Close pipes especially when you reuse them:

```awk
# Without close(), every value of $1 opens a new file handle and never frees it
{ print $0 > ("output_" $1 ".txt") }
$1 != prev { close("output_" prev ".txt"); prev = $1 }   # rotate cleanly
```

### 8.9 The "Read All Then Process" Pattern

Because awk only sees one record at a time, multi-pass algorithms either re-read the file or buffer it into memory. The classic approach:

```awk
# Pass 1: build the dictionary (in BEGIN, reading the whole file)
BEGIN {
    while ((getline line < "data") > 0) {
        n = split(line, f)
        rec[++count] = line
        sum += f[1]
    }
    close("data")
    avg = sum / count
}
# Pass 2: emit, comparing to avg
END {
    for (i = 1; i <= count; i++) {
        n = split(rec[i], f)
        if (f[1] > avg) print rec[i]
    }
}
```

Or use the "two arguments, same file" trick:

```awk
# awk -f prog.awk data data
NR == FNR { sum += $1; count++; next }       # first pass: accumulate
                                              # second pass: emit
$1 > sum/count { print }
```

The `NR == FNR` trick exploits the fact that `FNR` resets per file but `NR` doesn't — so on the first file `NR == FNR`, and on subsequent files it isn't.

---

## 9. Performance Characteristics

### 9.1 Implementation Speed

Rough order of magnitude on a column-extract benchmark over a 1 GB log file:

| Implementation | Time | Relative |
|:---------------|-----:|---------:|
| frawk | 6 s | 1.0x |
| mawk | 14 s | 2.3x |
| gawk | 38 s | 6.3x |
| bwk (one-true-awk) | 52 s | 8.7x |
| busybox awk | 110 s | 18x |

Numbers are illustrative; mileage varies by workload. Three patterns flip the ranking:

1. **Heavy regex.** gawk's regex engine is more sophisticated than mawk's; gawk can win on regex-heavy work.
2. **Massive arrays.** gawk has a better hash; mawk degrades on huge associative arrays.
3. **Math-heavy.** mawk's integer arithmetic path is hand-tuned; gawk wins only when bignums (`-M`) are off.

### 9.2 The Cost Model

For an input with N records each of L bytes and a program with P pattern-action rules:

| Operation | Cost |
|:----------|:-----|
| Read record | O(L) |
| Field split (whitespace) | O(L) |
| Field split (regex) | O(L · |regex|) |
| Pattern match (literal regex) | O(L) |
| Pattern match (compiled cached regex) | O(L) |
| Pattern match (uncached dynamic regex) | O(|regex|² + L) |
| Array `arr[k]` lookup | O(|k|) (hash) amortized |
| Array iteration | O(N) |
| `$0` reconstruction (after $i =) | O(NF · avg field len) |
| `getline` | O(L) |
| `print` to file | O(L) plus syscall |

The dominant cost on large inputs is usually the regex engine. mawk wins on raw throughput because its inner loop is tight; gawk wins on regex-heavy work because its engine is smarter.

### 9.3 The "use mawk for batch, gawk for one-off" Rule

For a script you'll run thousands of times in production: write portable POSIX awk, test against bwk, run under mawk in production. The 2–5x throughput is real and free.

For a script you'll run once to generate a report: gawk has time math, indirect calls, true multi-dim arrays, and a more forgiving runtime. The portability cost is irrelevant.

For ad-hoc one-liners on a known machine: use whatever `awk` is on the box; if performance matters, prepend `mawk` explicitly.

### 9.4 Practical Tuning Knobs

```awk
# Hoist regex out of inner loops
BEGIN { ipre = "^[0-9]{1,3}\\."; }
$1 ~ ipre { ... }

# Prefer index() over regex for fixed strings
index($0, "ERROR") > 0 { ... }      # faster than $0 ~ /ERROR/

# Avoid $0 reconstruction if you don't need it
{ if ($1 == "x") print $0 }         # cheap
{ $1 = $1; print $0 }               # forces O(NF) rebuild

# Avoid associative array growth in hot paths
# An array with millions of keys can exceed RAM in mawk
```

---

## 10. gawk Extensions

gawk's extensions divide into language, I/O, time, numeric, and structural buckets.

### 10.1 Time Functions

```awk
BEGIN {
    now = systime()                                   # epoch seconds
    print strftime("%Y-%m-%d %H:%M:%S", now)
    parsed = mktime("2026 04 25 14 30 00")
    print parsed - now, "seconds from now"
}
```

`mktime()` takes a space-separated `"YYYY MM DD HH MM SS [DST]"` string. `strftime()` follows C's strftime conventions. `systime()` returns the current epoch seconds.

mawk has none of these; bwk has none. For portable time math, shell out: `"date +%s" | getline now`.

### 10.2 Bignum Integers

```bash
gawk -M 'BEGIN { print 2^100, 100! }'
```

```awk
# Inside the program — gawk recognizes integer literals beyond 2^53
BEGIN {
    PREC = 100                # 100-bit precision
    print 2^200
}
```

`-M` enables MPFR/GMP arbitrary-precision math. `PREC` controls the precision of floats; `ROUNDMODE` controls the rounding mode.

### 10.3 Namespaces (gawk 5.0+)

```awk
@namespace "geometry"

function area(r) { return 3.14159 * r * r }

@namespace "awk"      # back to default

{ print geometry::area($1) }
```

Namespaces let you write libraries without name collisions. `awk::name` references the default namespace explicitly.

### 10.4 The `@load` Directive

```awk
@load "filefuncs"     # load the filefuncs extension

{ stat($1, info); print info["size"] }
```

`@load` loads a shared object that exposes new built-in functions. gawk ships `filefuncs`, `fnmatch`, `fork`, `inplace`, `intdiv`, `ordchr`, `readdir`, `readfile`, `revoutput`, `revtwoway`, `rwarray`, `time`. These are the official extensions; you can write your own using gawk's C API.

### 10.5 The `@include` Directive

```awk
@include "lib/json.awk"
@include "lib/utils.awk"

BEGIN { ... }
```

`@include` is awk's `#include`. The included file is parsed as if its text were inserted at the include point. Each include is processed at most once (idempotent).

### 10.6 Network I/O via `/inet/`

```awk
BEGIN {
    sock = "/inet/tcp/0/example.com/80"
    print "GET / HTTP/1.0\r\nHost: example.com\r\n\r" |& sock
    while ((sock |& getline line) > 0)
        print line
    close(sock)
}
```

gawk supports `/inet/tcp/`, `/inet/udp/`, `/inet4/`, `/inet6/`. The path components are `protocol/local-port/remote-host/remote-port`. Port `0` means OS-assigned. Combined with `|&` you have a tiny socket programming environment.

### 10.7 Database Integration

The PostgreSQL extension (`@load "pg"`) and MySQL extension exist via third-party builds; not in upstream gawk. For real database work, consider that you've outgrown awk.

### 10.8 Other Notable gawk Built-ins

| Function | Effect |
|:---------|:-------|
| `gensub(re, repl, how, target)` | substitute, *return* result without modifying target |
| `asort(src, dst)` | sort values into dst, return count |
| `asorti(src, dst)` | sort *indices* into dst |
| `mkbool(x)` | gawk 5.2: explicit boolean |
| `intdiv0(a, b, dst)` | integer division & modulo into dst |
| `typeof(x)` | "number", "string", "array", "regexp", "unassigned" |
| `bindtextdomain()`, `dcgettext()`, `dcngettext()` | gettext for i18n |

---

## 11. The Language as a Tiny Programming Language

awk is Turing-complete. Despite its DSL appearance, it has variables, conditionals, loops, recursion, hash tables, and I/O — sufficient for any computation.

### 11.1 Implementing a State Machine

```awk
BEGIN { state = "INIT" }
state == "INIT" && /BEGIN/ { state = "READING"; next }
state == "READING" && /END/ { state = "DONE"; next }
state == "READING" { buffer = buffer $0 ORS }
state == "DONE" { print "Captured:"; print buffer; state = "INIT"; buffer = "" }
```

This is a finite-state acceptor implemented in awk's pattern-action vocabulary. The same shape works for parsing log blocks, multi-line records, or simple grammars.

### 11.2 Aggregations and Group-By

```awk
# group-by-and-sum on column 1, summing column 2
{ sum[$1] += $2; count[$1]++ }
END {
    for (k in sum)
        printf "%-20s sum=%-10.2f avg=%-8.2f n=%d\n",
               k, sum[k], sum[k]/count[k], count[k]
}
```

This is exactly `SELECT k, SUM(v), AVG(v), COUNT(*) FROM t GROUP BY k`. awk's associative arrays are the entire aggregation engine.

### 11.3 Report Formatting

```awk
BEGIN { printf "%-30s %10s %10s\n", "Item", "Qty", "Total"
        print  "-------------------------------------------------" }
{ total += $3 * $4
  printf "%-30s %10d %10.2f\n", $1, $3, $3 * $4 }
END   { print  "-------------------------------------------------"
        printf "%-30s %10s %10.2f\n", "TOTAL", "", total }
```

Aligned tabular output with header, body, and footer. This is awk's traditional sweet spot — the original 1977 use case.

### 11.4 On-the-Fly Transformations

```awk
# Convert CSV to TSV, lowercase the headers, drop column 5
BEGIN { FS=","; OFS="\t" }
NR==1 { for (i=1; i<=NF; i++) $i = tolower($i) }
{ $5 = ""; for (i=5; i<NF; i++) $i = $(i+1); NF--; print }
```

This is a tiny ETL pipeline in 4 lines.

### 11.5 The "Right Tool for Column-Oriented Text" Pattern

If your problem is "tabular text with fields, do something per row, optionally aggregate," awk wins on lines-of-code, startup time, and portability over Python/Perl/Ruby. The moment your problem grows JSON, complex types, or non-trivial parsing, awk loses. The sweet spot is exactly: log files, CSV/TSV without quoting, fixed-width columnar data, /etc-style config files, build artifacts.

---

## 12. Common Anti-Patterns

A list of bad-then-better patterns you'll see in real awk code.

### 12.1 Useless Use of cat (UUOC)

```bash
# Bad — pipes a file through cat into awk
cat access.log | awk '{ print $1 }'

# Good — awk reads the file directly
awk '{ print $1 }' access.log
```

awk takes file arguments. The `cat` adds a process and a pipe; the savings on a 10 GB log are measurable.

### 12.2 Shell Loop Where awk Has BEGIN

```bash
# Bad — fork+exec per iteration
for i in $(seq 1 100); do
    awk -v n=$i 'BEGIN { print n*n }'
done

# Good — single awk invocation
awk 'BEGIN { for (i=1; i<=100; i++) print i*i }'
```

Each shell-driven `awk` invocation pays awk's startup cost (~5ms). 100 iterations is half a second; a million is an hour.

### 12.3 Subprocess-Heavy Pipelines

```bash
# Bad — sort, then awk, then sort again
sort data | awk '...' | sort

# Good — let awk do the grouping; sort once at the end
awk '...' data | sort
```

Many "sort first" patterns exist because the writer doesn't realize awk's hash table is exactly the abstraction needed.

### 12.4 Multi-Pass When Single-Pass Works

```bash
# Bad — three passes
awk '{print $1}' data | sort -u
awk '{print $2}' data | sort -u
awk '{print $3}' data | sort -u

# Good — one pass, three arrays
awk '{a[$1]++; b[$2]++; c[$3]++}
     END {
         for (k in a) print "a:", k
         for (k in b) print "b:", k
         for (k in c) print "c:", k
     }' data
```

Or run grouped output through `sort -u` after.

### 12.5 Relying on Iteration Order

```awk
# Bad — assumes hash order is meaningful
for (k in arr) print k                 # order is implementation-defined

# Good — explicit sort
n = 0
for (k in arr) keys[++n] = k
asort(keys)                            # gawk
for (i = 1; i <= n; i++) print keys[i]

# Or gawk-only:
PROCINFO["sorted_in"] = "@ind_str_asc"
for (k in arr) print k
```

Subtle bugs come from "it works on my machine" where the local awk happens to iterate in insertion order.

### 12.6 String Concatenation in a Hot Loop

```awk
# Bad — quadratic concatenation
{ buf = buf $0 "\n" }
END { printf "%s", buf }

# Good — print as you go
{ print }

# Good — buffer in an array, join at end
{ lines[++n] = $0 }
END { for (i = 1; i <= n; i++) print lines[i] }
```

awk's string concatenation in a tight loop forces repeated allocation. Either print directly or accumulate into an array.

### 12.7 Treating `$0` as Mutable When It Isn't Necessary

```awk
# Bad — modifies $0 just to inspect
{ gsub(/[^A-Z]/, ""); if (length() == 3) print original }

# Good — operate on a copy
{ s = $0; gsub(/[^A-Z]/, "", s); if (length(s) == 3) print }
```

`gsub` without an explicit target operates on `$0` and re-splits all fields. If you only need the transformed value for a check, copy first.

### 12.8 Forgetting close()

```awk
# Bad — opens a new pipe per record forever
{ print $0 | ("processor " $1) }

# Good — close when the key changes, or batch
{ print $0 | ("processor " $1) }
$1 != prev { close("processor " prev); prev = $1 }
END { close("processor " prev) }
```

Each unique pipe command is a child process that hangs around until awk closes it. Without `close`, you can run out of file descriptors.

---

## 13. Cost Model and Profiling

### 13.1 gawk's --profile Flag

gawk can produce a profiled trace showing how often each rule fires:

```bash
gawk --profile=prof.out -f prog.awk data
cat prof.out
```

The output annotates each line with execution count, e.g.:

```awk
    # gawk profile, created Sat Jan 25 14:32:01 2025

    # BEGIN rule(s)

      BEGIN {
   1            FS = ","
      }

    # Rule(s)

  10000  /error/ {
  10000          count++
   1234          print
      }

    # END rule(s)

      END {
   1            print "errors:", count
      }
```

Hot rules jump out. This is the awk equivalent of `cProfile` or `perf`.

### 13.2 The time(1) Wrapper

For implementation-comparison benchmarks:

```bash
# Compare awks on the same workload
time mawk -f prog.awk data
time gawk -f prog.awk data
time awk  -f prog.awk data        # whatever /usr/bin/awk is
```

Run each three times; take the median; `wall` time is what the user feels, `user`+`sys` exposes whether you're CPU- or I/O-bound.

### 13.3 Bench Patterns

```bash
# Generate predictable test data
seq 1 10000000 | awk 'BEGIN { srand(1) } { print rand(), rand(), $0 }' > big.dat

# Throughput baseline
time awk '{ s += $1 } END { print s }' big.dat

# Regex-heavy
time awk '/[0-9]\.[0-9]+/ { c++ } END { print c }' big.dat

# Array-heavy
time awk '{ k[int($1*1000)]++ } END { for (i in k) c++; print c }' big.dat
```

Run each test under mawk and gawk. The throughput baseline favors mawk; the regex test often goes to gawk; the array test depends on cardinality.

### 13.4 Reading the Profile

A useful rule: if a single inner loop is running > 90% of total time, look for a regex or `getline` that you can hoist or replace with `index()`. If function calls dominate, check whether you can inline.

---

## 14. Modern Replacements

awk is 50 years old. Several modern tools replace it for specific niches.

### 14.1 frawk

frawk is a Rust reimplementation. It JIT-compiles the program (LLVM or Cranelift), runs in parallel via SIMD CSV parsing, and is typically 5–10x faster than mawk on large inputs. Compatible with most awk programs; some incompatibilities around dynamic regex and edge-case I/O. Best when:
- You have multi-GB inputs.
- Your program is a clean "split, aggregate, print."
- You can install Rust binaries on the target machine.

```bash
frawk '{ s[$1] += $2 } END { for (k in s) print k, s[k] }' big.csv
```

### 14.2 Miller (mlr)

Miller is a structured-data command-line tool that speaks CSV, TSV, JSON-lines, and pretty-printed tables. It replaces awk for problems where the data is tabular but with field names you'd rather use:

```bash
# Group by status, sum bytes
mlr --csv stats1 -a sum -f bytes -g status access.csv

# Filter and project
mlr --tsv filter '$status == "200"' then cut -f path,bytes access.tsv

# Pretty-print
mlr --c2p cat data.csv
```

Miller wins on JSON-lines and CSV; awk wins on byte-level surgery and arbitrary text.

### 14.3 pandas / polars

For large-scale data work — joins, time series, statistical aggregations — awk is the wrong tool. Python's pandas, R's tidyverse, and Rust's polars handle that domain. The break-even is around "I need to join two tables" or "I want median, not just sum."

### 14.4 jq for JSON

awk has no JSON parser. The moment your input is JSON, switch to jq. Their philosophies are similar — pattern matching, transformation pipelines — but jq understands the structure.

### 14.5 Where awk Still Wins

Despite alternatives, awk dominates for:
- One-line text munging where startup time matters.
- Log slicing on machines where you can't install tools.
- Building/install scripts where awk is guaranteed present.
- Any problem solvable in five lines of `pattern { action }`.
- Pedagogy: awk is the cleanest example of pattern-action.

If the answer is "five lines of awk," it really is "five lines of awk." If the answer is "fifty lines of awk," consider Python.

---

## 15. Embedding awk in Pipelines

awk is Unix-pipeline-native. Most production awk lives between `sed` and `sort`.

### 15.1 The Canonical Pipeline

```bash
# Top 10 IPs by request count
awk '{print $1}' access.log | sort | uniq -c | sort -rn | head -10
```

`awk` extracts; `sort` clusters; `uniq -c` counts; `sort -rn` orders by count; `head` truncates. The same answer in pure awk:

```bash
awk '{c[$1]++} END {for (k in c) print c[k], k}' access.log | sort -rn | head -10
```

The pure-awk version reads the file once instead of streaming through a sort; on big files it wins easily.

### 15.2 awk as Filter Stage

```bash
# Extract Apache requests, normalize, group, count
sed -n 's/.*"\(GET [^"]*\)".*/\1/p' access.log \
  | awk '{print $2}' \
  | awk -F'?' '{print $1}' \
  | sort | uniq -c | sort -rn
```

Each tool does one thing. Awk shines as the "extract a field" stage even when the rest is sed and sort.

### 15.3 Heredoc Embedding

```bash
awk -f - data.csv <<'AWK'
BEGIN { FS = "," }
NR == 1 { for (i=1; i<=NF; i++) col[$i] = i; next }
$col["status"] == "ERROR" { print $col["timestamp"], $col["message"] }
AWK
```

The `-f -` reads the program from stdin; the heredoc supplies the program. Quoting `'AWK'` (single-quoted) prevents the shell from interpolating `$col["status"]`.

### 15.4 The -v Variable Injection

```bash
threshold=100
awk -v t="$threshold" '$3 > t { print }' data.tsv
```

`-v` sets variables before `BEGIN` runs. Use it for any shell-derived value to avoid string interpolation hell:

```bash
# Bad — shell expands inside awk's program
awk "{if (\$3 > $threshold) print}" data.tsv

# Good
awk -v t="$threshold" '$3 > t {print}' data.tsv
```

### 15.5 awk as Generator

```bash
# Generate a 10-line CSV from BEGIN
awk 'BEGIN {
    print "id,name,score"
    for (i = 1; i <= 10; i++)
        printf "%d,user%d,%.2f\n", i, i, rand()*100
}' > sample.csv
```

`BEGIN` with no main loop is awk's mode for "do this once and exit." Useful for sample data, fixtures, and configuration emission.

### 15.6 Multi-File Awareness with FILENAME

```bash
awk '{print FILENAME ":" FNR ":" $0}' a.txt b.txt c.txt
```

`FILENAME` is the current input filename. `FNR` resets per file; `NR` is cumulative. Together they give grep-like prefixes.

---

## 16. Idioms at the Internals Depth

Idioms you should recognize on sight, with internal-mechanics commentary.

### 16.1 Count Uniques

```awk
!seen[$0]++
```

This is equivalent to `if (!seen[$0]++) print`. The `!` makes the first appearance truthy; the `++` post-increments. The `print` is the implicit default action of a bare expression-pattern. It runs in O(N) time and O(unique-N) memory.

### 16.2 Sum a Column

```awk
{ s += $2 } END { print s }
```

Single accumulator, single end-print. This is the "Hello, World" of awk.

### 16.3 Print Specific Columns

```awk
{ print $3, $1, $5 }
```

Reordering. The `OFS` (default space) joins them.

### 16.4 Grep With Context

```awk
/pat/{ print prev "\n" $0 } { prev = $0 }
```

Saves the previous line; on a match, prints the previous line and the match. Effectively `grep -B1`.

### 16.5 Print Lines Between Markers

```awk
/^BEGIN/,/^END/
```

Range pattern. Inclusive of both endpoints. With an explicit action:

```awk
/^BEGIN/,/^END/ { print NR, $0 }
```

### 16.6 Last Field

```awk
{ print $NF }
```

`NF` is the field count; `$NF` indexes it. Same idiom for second-to-last:

```awk
{ print $(NF-1) }
```

### 16.7 Line Numbering

```awk
{ printf "%4d  %s\n", NR, $0 }
```

`cat -n` in one line.

### 16.8 Transpose Rows and Columns

```awk
{
    for (i = 1; i <= NF; i++)
        m[NR, i] = $i
    if (NF > maxf) maxf = NF
}
END {
    for (i = 1; i <= maxf; i++) {
        for (j = 1; j <= NR; j++)
            printf "%s%s", m[j, i], (j < NR ? OFS : ORS)
    }
}
```

Stores into a 2D associative array, then emits transposed. `SUBSEP` separates the keys internally.

### 16.9 Sum by Group

```awk
{ s[$1] += $2 } END { for (k in s) print k, s[k] }
```

GROUP BY and SUM. Pipe to `sort` for ordered output.

### 16.10 BEGIN-Only Calculator

```bash
awk 'BEGIN { print 2^32 }'
awk 'BEGIN { for (i=1;i<=20;i++) printf "%d! = %d\n", i, fact(i) }
     function fact(n) { return n<=1 ? 1 : n * fact(n-1) }'
```

awk with no main loop is a perfectly serviceable calculator. Useful when you don't want to fire up Python for `2^32`.

### 16.11 Reverse a File

```awk
{ a[NR] = $0 } END { for (i = NR; i >= 1; i--) print a[i] }
```

`tac` in awk. O(N) memory. (Real `tac` does cleverer things with seek().)

### 16.12 Remove Duplicate Adjacent Lines (`uniq`)

```awk
$0 != prev { print; prev = $0 }
```

`uniq` (without `-c`) in one line. Save the previous; print only when different.

### 16.13 Add Column Sum at the End

```awk
{ print; t += $NF }
END { print "TOTAL:", t }
```

Stream the rows, then emit a summary line.

### 16.14 Substring Search With Position

```awk
{ p = index($0, "ERROR"); if (p) print NR, p, substr($0, p) }
```

`index()` returns 1-based position or 0; `substr()` extracts from there.

### 16.15 Generate Random IDs

```awk
BEGIN {
    srand()
    for (i = 1; i <= 5; i++) {
        id = ""
        for (j = 1; j <= 8; j++) id = id sprintf("%x", int(rand()*16))
        print id
    }
}
```

awk's `rand()` is not cryptographically secure; for real ID generation use a system source.

### 16.16 In-Place Edit With gawk

```bash
gawk -i inplace '{ gsub(/old/, "new"); print }' file
```

The `inplace` extension rewrites the file. Equivalent to `sed -i 's/old/new/g'`. mawk and bwk have no equivalent.

---

## 17. Prerequisites

- Comfort with regular expressions, particularly POSIX ERE.
- Shell pipelines (stdin/stdout, redirection, exit codes).
- Field-delimited text formats: CSV (with reservations), TSV, whitespace-separated, fixed-width.
- Hash-table data structures and the meaning of "amortized O(1) lookup."
- Stack-based execution model (helpful but not required).

---

## 18. Complexity

For input of N records, mean record length L, P pattern-action rules, and a program touching K distinct array keys:

| Phase | Time | Space |
|:------|:-----|:------|
| BEGIN | O(BEGIN body) | O(1) |
| Per-record read | O(L) | O(L) |
| Per-record split | O(L) (whitespace) or O(L \cdot \|FS\|) (regex) | O(NF) |
| Per-record pattern eval | O(P \cdot L) | O(1) |
| Hash insert/lookup | O(\|key\|) amortized | O(1) per op |
| END | O(END body, e.g. K for array iteration) | O(K) |
| Total | O(N \cdot (L + P \cdot L)) plus O(K) array | O(L + K) |

For typical workloads, the dominant term is the regex evaluation in patterns: O(N · L) with a small constant. For workloads that build large hash tables, the dominant term is O(K · |key|) memory.

---

## 19. See Also

- awk
- bash
- regex
- sed
- polyglot

---

## 20. References

- POSIX awk specification — pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html
- GNU awk user manual — gnu.org/software/gawk/manual/
- Aho, Weinberger, Kernighan, *The AWK Programming Language*, Addison-Wesley, 1988 (the canonical reference)
- Robbins, *Effective awk Programming*, 5th ed., O'Reilly, 2023 (the gawk-flavored reference)
- one-true-awk on GitHub — github.com/onetrueawk/awk (Brian Kernighan's maintained nawk)
- gawk source — git.savannah.gnu.org/cgit/gawk.git
- mawk source — invisible-island.net/mawk/
- frawk — github.com/ezrosent/frawk
- Miller (mlr) — github.com/johnkerl/miller
- Brian Kernighan, "AWK — A Pattern Scanning and Processing Language," Bell Labs Technical Memorandum, 1977
- Ben Hoyt, "GoAWK: an AWK interpreter written in Go," 2018, benhoyt.com/writings/goawk/
