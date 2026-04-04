# The Theory of jq — Functional JSON Processing Language

> *jq is a functional, lazy, streaming JSON processor. Its core abstraction is the filter: a function from JSON values to streams of JSON values. Filters compose via pipes, branch via conditionals, and recurse via recursive descent. The language is Turing complete with lexical scoping, closures, and a rich set of built-in combinators.*

---

## 1. The Filter Model

### Core Abstraction

A jq **filter** is a function:

$$f: \text{JSON} \to \text{Stream}(\text{JSON})$$

Every filter takes one JSON input and produces zero or more JSON outputs.

### The Identity Filter

$$\text{.} : v \mapsto v$$

The simplest filter — passes input through unchanged.

### Generator Filters

Some filters produce multiple outputs:

```
.[] : [1, 2, 3] → 1, 2, 3    (three separate outputs)
```

### The Pipe Operator

Pipe connects filters by feeding each output of the left filter as input to the right:

$$f \mid g : v \mapsto \bigcup_{u \in f(v)} g(u)$$

```
.[] | . * 2 : [1, 2, 3] → 2, 4, 6
```

### The Comma Operator

Comma concatenates output streams:

$$f, g : v \mapsto f(v) \cup g(v)$$

```
.name, .age : {"name": "Alice", "age": 30} → "Alice", 30
```

---

## 2. Path Expressions

### Object Access

| Expression | Result | Note |
|:-----------|:-------|:-----|
| `.foo` | Value of key "foo" | `null` if absent |
| `.foo.bar` | Nested access | Chained |
| `.foo?` | Optional — no error if absent | Suppresses errors |
| `."key with spaces"` | Quoted key access | For non-identifier keys |

### Array Access

| Expression | Input | Result |
|:-----------|:------|:-------|
| `.[0]` | `[10, 20, 30]` | `10` |
| `.[-1]` | `[10, 20, 30]` | `30` |
| `.[1:3]` | `[10, 20, 30, 40]` | `[20, 30]` |
| `.[]` | `[10, 20, 30]` | `10`, `20`, `30` (stream) |

### Recursive Descent

`..` recursively descends into all values:

$$.. : v \mapsto v, \text{children}(v), \text{grandchildren}(v), \ldots$$

```
.. | numbers : {"a": {"b": 1}, "c": [2, 3]} → 1, 2, 3
```

---

## 3. Constructors and Collectors

### Array Construction

Square brackets **collect** a stream into an array:

$$[f] : v \mapsto [\text{all outputs of } f(v)]$$

```
[.[] | . * 2] : [1, 2, 3] → [2, 4, 6]
```

### Object Construction

```
{name: .user, id: .uid} : {"user": "Alice", "uid": 42} → {"name": "Alice", "id": 42}
```

Dynamic keys:

```
{(.key): .value} : {"key": "name", "value": "Alice"} → {"name": "Alice"}
```

### String Interpolation

```
"Hello, \(.name)!" : {"name": "Alice"} → "Hello, Alice!"
```

---

## 4. Conditionals and Comparison

### If-Then-Else

```
if .age >= 18 then "adult" else "minor" end
```

jq's truthiness: `false` and `null` are falsy. Everything else (including `0`, `""`, `[]`) is truthy.

### Comparison Operators

| Operator | Description |
|:---------|:-----------|
| `==`, `!=` | Equality (deep comparison) |
| `<`, `>`, `<=`, `>=` | Ordering |
| `and`, `or`, `not` | Boolean logic |

### Alternative Operator

```
.name // "unknown"    # if .name is null or false, use "unknown"
```

$$a \mathbin{//} b = \begin{cases} a & \text{if } a \neq \text{null} \land a \neq \text{false} \\ b & \text{otherwise} \end{cases}$$

### Try-Catch

```
try .foo.bar catch "default"    # suppress errors
try .foo.bar                     # suppress errors, produce no output
```

---

## 5. Reduction and Folding

### `reduce` — Left Fold

```
reduce .[] as $x (0; . + $x)
```

This is a **left fold**:

$$\text{reduce } f \text{ as } \$x (init; update) = \text{foldl}(\lambda \text{acc}, x. \text{update}[\$x/x, ./\text{acc}], \text{init}, f)$$

### Worked Example: Sum

```
reduce .[] as $x (0; . + $x) : [1, 2, 3, 4] → 10
```

Steps: `0 → 0+1=1 → 1+2=3 → 3+3=6 → 6+4=10`

### `foreach` — Streaming Fold

Like `reduce` but emits intermediate values:

```
[foreach .[] as $x (0; . + $x)]
: [1, 2, 3, 4] → [1, 3, 6, 10]    # running sum
```

### `limit` — Take First N

```
[limit(3; .[])] : [1, 2, 3, 4, 5] → [1, 2, 3]
```

Uses lazy evaluation — stops consuming input after 3 outputs.

---

## 6. Built-in Filters

### Array Functions

| Filter | Input | Output |
|:-------|:------|:-------|
| `length` | `[1, 2, 3]` | `3` |
| `sort` | `[3, 1, 2]` | `[1, 2, 3]` |
| `sort_by(.age)` | Array of objects | Sorted by `.age` field |
| `group_by(.type)` | Array of objects | Grouped arrays |
| `unique` | `[1, 2, 1, 3]` | `[1, 2, 3]` |
| `flatten` | `[[1, 2], [3]]` | `[1, 2, 3]` |
| `reverse` | `[1, 2, 3]` | `[3, 2, 1]` |
| `min`, `max` | `[3, 1, 2]` | `1`, `3` |
| `map(f)` | Array | Apply `f` to each element |
| `select(f)` | Value | Pass through if `f` is true |

### Object Functions

| Filter | Description |
|:-------|:-----------|
| `keys` | Array of keys (sorted) |
| `values` | Array of values |
| `has("key")` | Test key existence |
| `to_entries` | `[{"key": k, "value": v}, ...]` |
| `from_entries` | Inverse of `to_entries` |
| `with_entries(f)` | Transform key-value pairs |
| `+ {new: "val"}` | Merge objects |

### String Functions

| Filter | Description | Example |
|:-------|:-----------|:--------|
| `split(",")` | Split string | `"a,b,c"` → `["a","b","c"]` |
| `join(",")` | Join array | `["a","b"]` → `"a,b"` |
| `test("regex")` | Regex test (boolean) | `"foo" \| test("^f")` → `true` |
| `capture("(?<name>\\w+)")` | Named captures | Returns object |
| `ascii_downcase` | Lowercase | |
| `ltrimstr("prefix")` | Remove prefix | |

---

## 7. User-Defined Functions

### Definition

```
def double: . * 2;
def addN(n): . + n;

[.[] | double]           # [1, 2, 3] → [2, 4, 6]
[.[] | addN(10)]         # [1, 2, 3] → [11, 12, 13]
```

### Recursive Functions

```
def factorial:
  if . <= 1 then 1
  else . * ((. - 1) | factorial)
  end;

5 | factorial            # → 120
```

### Functions as Arguments

jq supports higher-order functions — filters can take filter arguments:

```
def mymap(f): [.[] | f];
def myselect(f): if f then . else empty end;

[1,2,3,4,5] | mymap(. * 2) | mymap(myselect(. > 4))
# → [6, 8, 10]
```

### `empty` — The Zero-Output Filter

$$\text{empty} : v \mapsto \emptyset$$

Produces no output. Used for filtering:

```
.[] | if . > 3 then . else empty end    # same as: .[] | select(. > 3)
```

---

## 8. Streaming and Performance

### Streaming Mode (`--stream`)

For large JSON files, `--stream` converts to a stream of path-value pairs:

```
echo '{"a":1,"b":[2,3]}' | jq --stream .

[[],{}]              # truncated start
[["a"],1]            # path: ["a"], value: 1
[["b",0],2]          # path: ["b",0], value: 2
[["b",1],3]          # path: ["b",1], value: 3
[["b"],true]         # truncated end of array
```

Memory: $O(d)$ where $d$ = maximum depth (not $O(n)$ like DOM).

### `tostream` / `fromstream`

Convert between tree and stream representations within a filter:

```
[tostream | select(.[0][-1] == "name") | .[1]]
```

---

## 9. The Type System

### jq Types

| Type | Examples | `type` output |
|:-----|:---------|:-------------|
| Null | `null` | `"null"` |
| Boolean | `true`, `false` | `"boolean"` |
| Number | `42`, `3.14` | `"number"` |
| String | `"hello"` | `"string"` |
| Array | `[1, 2]` | `"array"` |
| Object | `{"a": 1}` | `"object"` |

### Type-Based Dispatch

```
def process:
  if type == "array" then map(process)
  elif type == "object" then with_entries(.value |= process)
  elif type == "string" then ascii_downcase
  else .
  end;
```

---

## 10. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Core abstraction | Filter: JSON → Stream(JSON) |
| Composition | Pipe (`\|`), comma (`,`), collect (`[...]`) |
| Iteration | `.[]` (generator), `map(f)`, `reduce` |
| Truthiness | `null` and `false` are falsy; `0`, `""`, `[]` are truthy |
| Recursive descent | `..` traverses all nested values |
| Streaming | `--stream` for large files, $O(d)$ memory |
| Functions | First-class, recursive, higher-order |
| Turing complete | Yes (recursion + conditionals + variables) |

---

*jq is not "grep for JSON" — it's a complete functional programming language specialized for JSON transformation. Its filter-pipe-collect model maps naturally to data transformation pipelines: extract fields, filter records, reshape structures, aggregate values. Master the five core operations (`.`, `.[]`, `|`, `[]`, `{}`) and jq becomes the fastest path from raw JSON to the answer you need.*

## Prerequisites

- JSON structure and syntax
- Functional programming concepts (map, filter, reduce, pipes)
- Shell piping and stdin/stdout conventions
- Path expressions and tree traversal
