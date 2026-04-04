# The Theory of YAML — Data Model, Parsing, and Type Resolution

> *YAML is a data serialization language with a three-stage processing model: parsing (character stream to events), composing (events to node graph), and constructing (nodes to native types). Its data model supports anchors/aliases (graph structures), multi-line strings with precise whitespace control, and implicit type resolution that can cause surprising type coercion.*

---

## 1. The YAML Data Model

### Three-Stage Processing

```
Character Stream
    │ Parse (Scanner + Parser)
    ▼
Event Stream (stream-start, mapping-start, scalar, ...)
    │ Compose
    ▼
Representation Graph (nodes + anchors/aliases)
    │ Construct
    ▼
Native Data Structures (dicts, lists, strings, ints, ...)
```

### Node Types

| Node | JSON Equivalent | YAML Examples |
|:-----|:---------------|:-------------|
| Mapping | Object | `key: value` |
| Sequence | Array | `- item` |
| Scalar | String/Number/Bool/Null | `hello`, `42`, `true`, `~` |

### The Representation Graph

Unlike JSON's tree, YAML's data model is a **directed graph** — anchors and aliases allow shared references and even cycles:

```yaml
defaults: &defaults
  timeout: 30
  retries: 3

production:
  <<: *defaults         # merge key — copies defaults
  timeout: 60           # override

# Graph: production node references defaults node
```

---

## 2. Indentation as Syntax — Formal Grammar

### The Off-Side Rule

YAML uses indentation to denote structure (like Python, Haskell). The grammar is **context-sensitive** — the meaning of indentation depends on the current nesting level.

$$\text{child\_indent} > \text{parent\_indent}$$

This makes YAML **not context-free** — a PDA cannot parse it. The parser maintains an **indentation stack**.

### Indentation Rules

| Rule | Description |
|:-----|:-----------|
| Spaces only | Tabs are **forbidden** for indentation |
| Consistent per level | All siblings at same indent level |
| Amount is flexible | 1+ spaces per level (2 is conventional) |
| Flow style exempt | `{key: value}` and `[a, b]` ignore indentation |

### Why Tabs Are Forbidden

Tabs have ambiguous visual width (2, 4, 8 spaces depending on terminal settings). Since YAML's semantics depend on visual alignment, tabs would make the grammar ambiguous.

---

## 3. Scalar Types and Implicit Resolution

### The Type Resolution Problem

YAML scalars are unquoted by default. The parser must determine the type:

```yaml
port: 8080        # integer
host: localhost   # string
debug: true       # boolean
version: 1.0      # float
```

### YAML 1.1 Implicit Type Resolution (The Dangerous One)

| Pattern | Resolved Type | Surprise |
|:--------|:-------------|:---------|
| `true`, `yes`, `on`, `True`, `YES` | Boolean true | `country: NO` → `false` |
| `false`, `no`, `off`, `False`, `NO` | Boolean false | Norway problem |
| `0o17` | Octal integer (15) | `permissions: 0755` → `493` |
| `0x1F` | Hex integer (31) | |
| `1_000` | Integer (1000) | |
| `~`, `null`, `Null`, `NULL` | Null | |
| `.inf`, `.NaN` | Float | |
| `2023-01-15` | Date | `version: 2023-01-15` → date object |

### The Norway Problem

```yaml
countries:
  - GB    # string "GB"
  - IE    # string "IE"
  - NO    # boolean false!  (YAML 1.1)
```

### YAML 1.2 — Reduced Ambiguity

YAML 1.2 (2009) reduced the boolean values to only `true` and `false` (case-sensitive). Many parsers still default to YAML 1.1.

### Explicit Typing with Tags

```yaml
country: !!str NO        # force string
port: !!str 8080         # force string
flag: !!bool true        # explicit boolean
```

### The Quoting Solution

```yaml
country: "NO"            # always a string
version: "1.0"           # always a string
```

**Rule of thumb:** When in doubt, quote it.

---

## 4. Multi-Line Strings — Block Scalars

### Two Block Scalar Styles

| Style | Indicator | Newlines | Trailing Newline |
|:------|:----------|:---------|:----------------|
| Literal | `|` | Preserved | Single `\n` |
| Folded | `>` | Folded to spaces | Single `\n` |

### Chomping Indicators

| Indicator | Name | Trailing Newlines |
|:----------|:-----|:-----------------|
| (none) | Clip | Single trailing `\n` |
| `-` | Strip | No trailing `\n` |
| `+` | Keep | All trailing `\n` preserved |

### Worked Example

```yaml
literal: |
  line one
  line two

  line four

folded: >
  line one
  line two

  line four
```

Results:
```
literal: "line one\nline two\n\nline four\n"
folded:  "line one line two\nline four\n"
```

The folded style joins consecutive non-blank lines with a space. Blank lines become `\n`.

### Indentation Indicator

When the first line of a block scalar starts with a space, you must specify the indentation level:

```yaml
code: |2
  def hello():
      print("hi")
```

The `2` means "content is indented 2 spaces from the indicator."

---

## 5. Anchors and Aliases — Graph Structures

### Syntax

```yaml
anchor: &name value
alias: *name
```

### Use Cases

**Shared configuration:**
```yaml
defaults: &defaults
  adapter: postgres
  host: localhost

development:
  database: dev_db
  <<: *defaults

production:
  database: prod_db
  <<: *defaults
  host: prod.example.com
```

### The Merge Key (`<<`)

The `<<` key with an alias performs a **shallow merge**: alias values are inserted into the mapping, but existing keys take precedence.

### Circular References

YAML allows circular references (unlike JSON):

```yaml
&a
- *a
```

This creates a list that contains itself. Most application-level parsers reject this, but it's valid in the YAML data model.

### Billion Laughs Attack

```yaml
a: &a ["lol"]
b: &b [*a, *a, *a, *a, *a]
c: &c [*b, *b, *b, *b, *b]
# Each level multiplies by 5
# Level 10: 5^10 = ~10 million elements
```

Exponential expansion from small input. Parsers must limit alias expansion depth.

---

## 6. Flow Style vs Block Style

### Two Syntaxes for the Same Data

| Structure | Block Style | Flow Style |
|:----------|:-----------|:-----------|
| Mapping | `key: value` (indented) | `{key: value, k2: v2}` |
| Sequence | `- item` (indented) | `[item1, item2]` |

Flow style is JSON-compatible. In fact, **every valid JSON document is valid YAML** (since YAML 1.2).

### Mixing Styles

```yaml
people:
  - name: Alice
    scores: [95, 87, 92]      # flow sequence in block mapping
  - {name: Bob, age: 30}      # flow mapping in block sequence
```

---

## 7. Multi-Document Streams

### Document Markers

```yaml
---                    # document start
document: one
...                    # document end (optional)
---                    # next document start
document: two
```

| Marker | Purpose |
|:-------|:--------|
| `---` | Start of document (required for multiple docs, optional for single) |
| `...` | End of document (optional, used before non-YAML content) |

### Directives

```yaml
%YAML 1.2             # YAML version directive
%TAG !e! tag:example.com,2000:  # tag prefix shorthand
---
```

---

## 8. YAML vs JSON vs TOML

### Feature Comparison

| Feature | JSON | YAML | TOML |
|:--------|:----:|:----:|:----:|
| Comments | No | `#` | `#` |
| Multi-line strings | No | `|`, `>` | `"""` |
| Anchors/aliases | No | Yes | No |
| Date/time type | No | Yes | Yes |
| Circular references | No | Yes | No |
| Trailing commas | No | N/A | No |
| Spec complexity | ~10 pages | ~80 pages | ~20 pages |
| Parsing complexity | $O(n)$ LL(1) | Context-sensitive | $O(n)$ |

### Common YAML Gotchas

| Input | Expected | Actual (YAML 1.1) |
|:------|:---------|:-------------------|
| `version: 3.10` | String "3.10" | Float 3.1 |
| `on: true` | Mapping | `{true: true}` |
| `NO` | String | Boolean false |
| `1_000` | String | Integer 1000 |
| `!!python/object` | Rejected | Arbitrary code execution |

### Arbitrary Code Execution

YAML tags like `!!python/object/apply:os.system` can execute code in unsafe parsers. **Always use safe loaders** (`yaml.safe_load()` in Python, not `yaml.load()`).

---

## 9. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Grammar | Context-sensitive (indentation-dependent) |
| Data model | Directed graph (not tree) via anchors/aliases |
| Node types | Mapping, Sequence, Scalar |
| Processing stages | Parse → Compose → Construct |
| Implicit typing | Source of bugs (Norway problem, float truncation) |
| Block scalars | Literal (`\|`), Folded (`>`), with chomping (`-`, `+`) |
| Security | Billion laughs (alias expansion), code execution (unsafe tags) |
| JSON superset | Every valid JSON is valid YAML (1.2) |

---

*YAML's power — comments, anchors, multi-line strings, implicit typing — is also its danger. The specification is 80 pages long. Most YAML bugs come from implicit type resolution (quote your strings), indentation errors (use a linter), or unsafe deserialization (use safe loaders). When simplicity matters more than features, use JSON or TOML.*

## Prerequisites

- Data serialization concepts (schema, typing, interchange)
- Indentation-sensitive parsing
- Anchors, aliases, and merge key semantics
- Unicode and encoding fundamentals
