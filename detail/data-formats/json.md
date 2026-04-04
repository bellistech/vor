# The Theory of JSON — Grammar, Parsing, and Schema Validation

> *JSON is defined by exactly 6 production rules and 4 primitive types. It's context-free, LL(1)-parsable in O(n) time, and its schema language (JSON Schema) forms a constraint satisfaction system. This simplicity is its greatest strength — and the source of its limitations.*

---

## 1. JSON Grammar — Formal Definition

### Production Rules (RFC 8259)

JSON's complete grammar in EBNF:

```
value     = object | array | string | number | "true" | "false" | "null"
object    = "{" [ member *( "," member ) ] "}"
member    = string ":" value
array     = "[" [ value *( "," value ) ] "]"
string    = '"' *char '"'
number    = [ "-" ] int [ frac ] [ exp ]
int       = "0" | ( digit1-9 *digit )
frac      = "." 1*digit
exp       = ( "e" | "E" ) [ "+" | "-" ] 1*digit
```

### Grammar Classification

JSON is a **context-free grammar** (CFG) — it can be described by production rules where the left-hand side is a single non-terminal. Specifically:

- **LL(1)**: parsable with 1 token of lookahead (the first character determines the type)
- **Unambiguous**: every valid JSON string has exactly one parse tree
- **Not regular**: nested structures (objects in arrays in objects) require a stack

### Lookahead Table

| First Character | Value Type |
|:----------------|:-----------|
| `{` | Object |
| `[` | Array |
| `"` | String |
| `-`, `0`-`9` | Number |
| `t` | `true` |
| `f` | `false` |
| `n` | `null` |

One character of lookahead suffices to determine the production rule — this is what makes JSON LL(1).

---

## 2. Parsing Complexity

### Two Parsing Strategies

| Strategy | Memory | Access Pattern | Complexity |
|:---------|:-------|:---------------|:-----------|
| DOM (tree) | $O(n)$ | Random access after parse | $O(n)$ time, $O(n)$ space |
| SAX/Streaming | $O(d)$ | Forward-only, event-driven | $O(n)$ time, $O(d)$ space |

Where $n$ = input size, $d$ = maximum nesting depth.

### DOM Parsing

Build a complete in-memory tree. Every value becomes a node:

```
{"name": "Alice", "scores": [95, 87]}

         Object
        /      \
   "name"    "scores"
     |          |
  "Alice"     Array
              /    \
            95      87
```

Memory overhead: each node requires a type tag, pointers, and value storage. Typical overhead: **3-10x** the raw JSON size.

### Streaming Parsing

Emit events as tokens are encountered:

```
START_OBJECT
  KEY "name"
  STRING "Alice"
  KEY "scores"
  START_ARRAY
    NUMBER 95
    NUMBER 87
  END_ARRAY
END_OBJECT
```

Memory: only the current path/depth needs to be stored. Ideal for large files.

### SIMD-Accelerated Parsing (simdjson)

Modern parsers use SIMD instructions to classify characters in parallel:

1. Load 32/64 bytes at once
2. Classify structural characters (`{}[],:`) using vector comparison
3. Build a **structural index** (positions of all structural characters)
4. Parse using the index

Throughput: **2-4 GB/s** on modern hardware (vs ~300 MB/s for traditional parsers).

---

## 3. Number Representation Limits

### IEEE 754 Double Precision

JSON numbers are arbitrary precision in the grammar, but most implementations use IEEE 754 doubles:

$$\text{double} = (-1)^s \times 2^{e-1023} \times (1 + f)$$

| Property | Value |
|:---------|:------|
| Significand bits | 52 (+ 1 implicit) |
| Max safe integer | $2^{53} - 1 = 9,007,199,254,740,991$ |
| Min safe integer | $-(2^{53} - 1)$ |
| Max value | $\approx 1.8 \times 10^{308}$ |
| Min positive | $\approx 5 \times 10^{-324}$ |
| Decimal precision | ~15.9 significant digits |

### The Integer Problem

```json
{"id": 9007199254740993}
```

JavaScript parses this as `9007199254740992` (rounds to nearest representable double). This is why Twitter IDs, Snowflake IDs, and database primary keys are often transmitted as **strings** in JSON APIs.

### Workaround

```json
{"id": "9007199254740993", "id_int": 9007199254740993}
```

Or use BigInt-aware parsers.

---

## 4. JSON Schema — Constraint System

### Schema as Type System

JSON Schema defines types using **constraints** (not classes). Each keyword is an independent assertion:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "name": { "type": "string", "minLength": 1 },
    "age": { "type": "integer", "minimum": 0, "maximum": 150 }
  },
  "required": ["name", "age"],
  "additionalProperties": false
}
```

### Validation as Constraint Satisfaction

A JSON value $v$ satisfies schema $S$ if all constraints in $S$ are satisfied:

$$\text{valid}(v, S) = \bigwedge_{c \in S.\text{constraints}} c(v)$$

### Composition Keywords

| Keyword | Logic | Meaning |
|:--------|:------|:--------|
| `allOf: [S1, S2]` | $S_1 \land S_2$ | Must match all schemas |
| `anyOf: [S1, S2]` | $S_1 \lor S_2$ | Must match at least one |
| `oneOf: [S1, S2]` | $S_1 \oplus S_2$ | Must match exactly one |
| `not: S` | $\lnot S$ | Must NOT match |
| `if/then/else` | $S_1 \implies S_2 \mid S_3$ | Conditional validation |

### Schema Validation Complexity

In general, JSON Schema validation is **polynomial** in the size of the document for a fixed schema. However, complex schemas with nested `allOf`/`anyOf` can lead to exponential validation time:

$$\text{Worst case}: O(n \times |S|^d)$$

Where $n$ = document size, $|S|$ = schema size, $d$ = nesting depth of composition keywords.

---

## 5. JSON Pointer and JSON Patch

### JSON Pointer (RFC 6901)

A string syntax for identifying a specific value within a JSON document:

```
/foo/bar/0 → root["foo"]["bar"][0]
```

Escaping: `~0` = `~`, `~1` = `/`.

### JSON Patch (RFC 6902)

An array of operations to apply atomically:

```json
[
  {"op": "add", "path": "/tags/0", "value": "new"},
  {"op": "remove", "path": "/obsolete"},
  {"op": "replace", "path": "/version", "value": 2},
  {"op": "move", "from": "/old", "path": "/new"},
  {"op": "test", "path": "/version", "value": 2}
]
```

### JSON Merge Patch (RFC 7396)

Simpler alternative — a partial document that describes changes:

```json
{
  "name": "New Name",
  "obsolete": null,
  "nested": {"added": true}
}
```

Rule: `null` means delete, present means replace/add, absent means keep.

---

## 6. JSON vs Related Formats

### What JSON Lacks

| Feature | JSON | YAML | TOML | JSON5 |
|:--------|:----:|:----:|:----:|:-----:|
| Comments | No | Yes | Yes | Yes |
| Trailing commas | No | N/A | No | Yes |
| Multi-line strings | No | Yes | Yes | Yes |
| Date type | No | Yes | Yes | No |
| NaN/Infinity | No | Yes | No | Yes |
| References/anchors | No | Yes | No | No |
| Binary data | No | No | No | No |

### JSON Lines (JSONL / NDJSON)

One JSON value per line, separated by `\n`:

```
{"event": "click", "time": 1234}
{"event": "scroll", "time": 1235}
```

Advantages: streamable, appendable, splittable, parallelizable. Used for log files, data pipelines.

---

## 7. String Encoding

### Escape Sequences

| Sequence | Character |
|:---------|:----------|
| `\"` | Quotation mark |
| `\\` | Reverse solidus |
| `\/` | Solidus (optional) |
| `\b` | Backspace |
| `\f` | Form feed |
| `\n` | Newline |
| `\r` | Carriage return |
| `\t` | Tab |
| `\uXXXX` | Unicode codepoint (BMP) |

### Surrogate Pairs for Non-BMP Characters

Characters outside the Basic Multilingual Plane (U+10000+) require **surrogate pairs**:

$$\text{U+1F600} \to \text{\textbackslash uD83D\textbackslash uDE00}$$

Formula for encoding codepoint $U$ (where $U \geq \text{0x10000}$):

$$U' = U - \text{0x10000}$$
$$\text{high} = \text{0xD800} + (U' >> 10)$$
$$\text{low} = \text{0xDC00} + (U' \mathbin{\&} \text{0x3FF})$$

---

## 8. Summary of Key Properties

| Property | Value |
|:---------|:------|
| Grammar class | Context-free, LL(1), unambiguous |
| Parse complexity | $O(n)$ time |
| Value types | 7: object, array, string, number, true, false, null |
| Max safe integer | $2^{53} - 1$ (IEEE 754 double) |
| Schema logic | Constraint satisfaction with boolean composition |
| String encoding | UTF-8 with `\uXXXX` escapes |
| MIME type | `application/json` |
| File extension | `.json` |

---

*JSON won not because it's the best serialization format — it lacks comments, dates, and binary support. It won because it's simple enough to parse in any language in an afternoon, human-readable enough to debug by eye, and precisely specified enough that interoperability issues are rare. The entire spec fits on a business card.*

## Prerequisites

- Context-free grammars and recursive descent parsing
- Unicode and UTF-8 encoding
- Data serialization concepts (schema, validation, interchange)
- IEEE 754 floating-point representation

## Complexity

| Operation | Time Complexity | Notes |
|---|---|---|
| Parse | O(n) | Single-pass recursive descent |
| Serialize | O(n) | Linear scan of data structure |
| JSONPath query | O(n) | Full document traversal in worst case |
| Schema validation | O(n * s) | n = document size, s = schema size |
