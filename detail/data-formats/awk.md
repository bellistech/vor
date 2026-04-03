# The Theory of AWK — Pattern-Action Language and Record Processing

> *AWK is a data-driven programming language based on the pattern-action paradigm: for each input record, evaluate all patterns and execute the associated actions. It implements a complete programming language (variables, arrays, functions, printf) within a streaming record-processing framework. Its computational model is a finite transducer over line-oriented input.*

---

## 1. The Processing Model

### The AWK Cycle

```
BEGIN { initialization }

For each input record (line):
    Split record into fields ($1, $2, ..., $NF)
    For each pattern-action rule:
        If pattern matches:
            Execute action

END { finalization }
```

### Formal Model

AWK is a **finite transducer**: it reads input, maintains state (variables, arrays), and produces output. The processing is:

$$\text{output} = \bigcup_{r \in \text{records}} \bigcup_{(p_i, a_i) \in \text{rules}} \begin{cases} a_i(r) & \text{if } p_i(r) = \text{true} \\ \emptyset & \text{otherwise} \end{cases}$$

### Execution Order

1. Execute `BEGIN` block (before any input)
2. Read input record by record (split by `RS`, default `\n`)
3. Split each record into fields (split by `FS`, default whitespace)
4. Evaluate all pattern-action pairs in order
5. Execute `END` block (after all input)

---

## 2. Records and Fields

### Built-in Variables

| Variable | Default | Meaning |
|:---------|:--------|:--------|
| `RS` | `\n` | Record separator (input) |
| `ORS` | `\n` | Output record separator |
| `FS` | whitespace | Field separator (input) |
| `OFS` | space | Output field separator |
| `NR` | — | Current record number (across all files) |
| `FNR` | — | Record number in current file |
| `NF` | — | Number of fields in current record |
| `$0` | — | Entire current record |
| `$n` | — | n-th field |
| `FILENAME` | — | Current input filename |

### Field Splitting Algorithm

1. If `FS` is a single space (default): split on runs of whitespace, ignore leading/trailing
2. If `FS` is a single character: split on that exact character
3. If `FS` is a multi-character string: treat as regex
4. If `FS` is empty string: split each character into a field

### Reassigning Fields

Assigning to a field reconstructs `$0`:

```awk
{ $2 = "REDACTED"; print $0 }
# Fields are rejoined using OFS
```

Assigning to `$0` re-splits into fields using current `FS`.

---

## 3. Pattern Types

### Six Pattern Types

| Pattern | Syntax | Matches When |
|:--------|:-------|:-------------|
| Expression | `$3 > 100` | Expression is true (non-zero/non-empty) |
| Regex | `/pattern/` | `$0 ~ /pattern/` |
| Field regex | `$2 ~ /pat/` | Specific field matches |
| Range | `/start/,/stop/` | From start match through stop match |
| BEGIN | `BEGIN` | Before any input |
| END | `END` | After all input |
| Empty | (omitted) | Every record |

### Range Pattern Semantics

```awk
/START/,/END/ { print }
```

State machine:
```
         ┌──── /START/ matches ────┐
         │                          ▼
    [OFF] ◄── /END/ matches ── [ON: print]
                                    │
                             (also prints the
                              /END/ record)
```

The range is **inclusive** of both start and end records.

### Compound Patterns

```awk
$3 > 100 && $5 ~ /active/    # AND
$1 == "error" || $1 == "fatal" # OR
!($3 > 100)                    # NOT
```

---

## 4. Associative Arrays

### Hash Tables

AWK arrays are **associative** (hash tables) — keys are always strings:

```awk
count["apple"]++
count["banana"] += 3

for (key in count) {
    print key, count[key]
}
```

### Multi-Dimensional Arrays

AWK simulates multi-dimensional arrays using `SUBSEP` (default `\034`, ASCII FS character):

```awk
matrix[1, 2] = "value"
# Actually stores: matrix["1\0342"] = "value"

# Test membership:
if ((1, 2) in matrix) print "exists"
```

### Deletion

```awk
delete array[key]     # delete single element
delete array          # delete entire array (gawk extension)
```

### Array Iteration Order

`for (key in array)` iterates in **unspecified order** (hash table order). To sort:

```awk
# gawk extension:
PROCINFO["sorted_in"] = "@val_num_asc"
for (key in array) print key, array[key]
```

---

## 5. String Functions

### Core Functions

| Function | Description | Complexity |
|:---------|:-----------|:-----------|
| `length(s)` | String length | $O(n)$ |
| `substr(s, start, len)` | Substring | $O(\text{len})$ |
| `index(s, target)` | Find first occurrence | $O(nm)$ |
| `split(s, array, sep)` | Split string into array | $O(n)$ |
| `sub(regex, replacement, target)` | Replace first match | $O(n)$ |
| `gsub(regex, replacement, target)` | Replace all matches | $O(n)$ per match |
| `match(s, regex)` | Find regex match, set RSTART/RLENGTH | $O(nm)$ |
| `sprintf(fmt, args...)` | Format string | $O(n)$ |
| `tolower(s)` / `toupper(s)` | Case conversion | $O(n)$ |

### The `sub`/`gsub` Replacement

The replacement string can use `&` to reference the matched text:

```awk
{ gsub(/[0-9]+/, "[&]"); print }
# Input:  "port 8080 and 443"
# Output: "port [8080] and [443]"
```

---

## 6. Numeric Processing

### AWK as a Calculator

AWK handles floating-point arithmetic natively:

```awk
{ sum += $1; count++ }
END { print "Average:", sum / count }
```

### Mathematical Functions

| Function | Description |
|:---------|:-----------|
| `int(x)` | Truncate to integer |
| `sqrt(x)` | Square root |
| `exp(x)` | $e^x$ |
| `log(x)` | Natural logarithm |
| `sin(x)` / `cos(x)` | Trigonometric |
| `atan2(y, x)` | Arc tangent |
| `rand()` | Random in $[0, 1)$ |
| `srand(seed)` | Seed random generator |

### Worked Example: Standard Deviation

```awk
{ sum += $1; sumsq += $1 * $1; n++ }
END {
    mean = sum / n
    variance = (sumsq / n) - (mean * mean)
    stddev = sqrt(variance)
    printf "Mean: %.2f, StdDev: %.2f\n", mean, stddev
}
```

$$\sigma = \sqrt{\frac{\sum x_i^2}{n} - \left(\frac{\sum x_i}{n}\right)^2}$$

---

## 7. Output and Formatting

### Three Output Mechanisms

| Statement | Description |
|:----------|:-----------|
| `print` | Print with `OFS` and `ORS` |
| `printf fmt, args` | C-style formatted output |
| `print > file` | Redirect to file |
| `print >> file` | Append to file |
| `print \| cmd` | Pipe to command |

### Printf Formats

| Format | Type | Example |
|:-------|:-----|:--------|
| `%d` | Integer | `printf "%d", 42` |
| `%f` | Float | `printf "%.2f", 3.14159` → `3.14` |
| `%e` | Scientific | `printf "%e", 1234` → `1.234000e+03` |
| `%s` | String | `printf "%-20s", "left-aligned"` |
| `%x` | Hex | `printf "%x", 255` → `ff` |
| `%o` | Octal | `printf "%o", 255` → `377` |

---

## 8. User-Defined Functions

```awk
function factorial(n) {
    if (n <= 1) return 1
    return n * factorial(n - 1)
}

{ print $1, factorial($1) }
```

### Scope Rules

- All variables are **global** by default
- Function parameters are **local**
- Local variables are declared as extra parameters (convention):

```awk
function myfunc(param,    local1, local2) {
    # local1 and local2 are local (extra params, never passed)
}
```

The extra spaces before `local1` are a convention to mark them as locals.

---

## 9. AWK Implementations

| Implementation | Features | Speed |
|:---------------|:---------|:------|
| `awk` (POSIX) | Minimal, portable | Baseline |
| `gawk` (GNU) | Networking, regex extensions, `@include` | Good |
| `mawk` | Minimal, optimized interpreter | Fastest |
| `goawk` | Go implementation, CSV support | Good |

### Complexity Analysis

For $n$ input records, $p$ pattern-action rules, and $f$ fields per record:

$$\text{Time} = O(n \times p \times C_{\text{pattern}})$$

Where $C_{\text{pattern}}$ is the cost of evaluating each pattern (typically $O(f)$ for field comparisons, $O(f \times |\text{regex}|)$ for regex patterns).

---

## 10. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Processing model | Record-oriented pattern-action |
| Record splitting | `RS` (default newline) |
| Field splitting | `FS` (default whitespace) |
| Arrays | Associative (hash tables), string keys |
| Scope | Global by default, locals via parameter hack |
| Pattern types | Expression, regex, range, BEGIN, END |
| Output | `print`, `printf`, redirection, piping |

---

*AWK occupies a unique niche: more powerful than sed, simpler than Perl, faster to write than Python for columnar data. Its pattern-action model maps naturally to log analysis, CSV processing, and report generation. When your problem is "for each line, check something, print something," AWK is the right tool.*
