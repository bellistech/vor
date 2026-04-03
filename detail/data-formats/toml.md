# The Theory of TOML — Grammar, Type System, and Table Semantics

> *TOML (Tom's Obvious Minimal Language) is a configuration file format with a simple, unambiguous grammar. Unlike YAML, every TOML document maps to a unique hash table with typed values. There is no implicit type coercion — every value's type is syntactically determined. The table/array-of-tables system provides structured nesting without indentation.*

---

## 1. TOML Grammar — Key Properties

### Grammar Classification

| Property | TOML | JSON | YAML |
|:---------|:-----|:-----|:-----|
| Grammar class | Context-free | Context-free (LL1) | Context-sensitive |
| Ambiguity | Unambiguous | Unambiguous | Ambiguous (implicit types) |
| Whitespace-sensitive | No (except multi-line strings) | No | Yes (indentation) |
| Type determination | Syntactic (from value syntax) | Syntactic | Implicit (heuristic) |

### Type System

TOML has exactly 8 types:

| Type | Syntax | Example |
|:-----|:-------|:--------|
| String | `"..."`, `'...'`, `"""..."""`, `'''...'''` | `"hello"` |
| Integer | Decimal, hex, octal, binary | `42`, `0xFF`, `0o77`, `0b1010` |
| Float | IEEE 754 | `3.14`, `1e10`, `inf`, `nan` |
| Boolean | `true`, `false` | `true` |
| Offset Date-Time | RFC 3339 | `2024-01-15T09:30:00-05:00` |
| Local Date-Time | RFC 3339 (no offset) | `2024-01-15T09:30:00` |
| Local Date | ISO 8601 | `2024-01-15` |
| Local Time | ISO 8601 | `09:30:00` |
| Array | `[...]` | `[1, 2, 3]` |
| Inline Table | `{...}` | `{key = "value"}` |

### No Null Type

TOML has **no null/nil/none type**. If a key is absent, it's absent — there's no way to represent "key present but no value." This is a deliberate design choice to reduce ambiguity.

---

## 2. Tables — The Nesting Mechanism

### Table Headers

```toml
[server]
host = "localhost"
port = 8080

[server.tls]
cert = "/path/to/cert"
key = "/path/to/key"
```

Maps to:
```json
{
  "server": {
    "host": "localhost",
    "port": 8080,
    "tls": {
      "cert": "/path/to/cert",
      "key": "/path/to/key"
    }
  }
}
```

### Dotted Keys vs Table Headers

These are equivalent:

```toml
# Table header style:
[a]
[a.b]
c = 1

# Dotted key style:
a.b.c = 1
```

Both produce: `{"a": {"b": {"c": 1}}}`

### Super Tables — Implicit Creation

```toml
[a.b.c]
d = 1
```

Tables `a` and `a.b` are **implicitly created** as empty tables. You can later add keys to them:

```toml
[a.b.c]
d = 1

[a]
e = 2
```

Result: `{"a": {"e": 2, "b": {"c": {"d": 1}}}}`

### The No-Redefine Rule

A table header can only appear once. A key can only be defined once.

```toml
[a]
b = 1

[a]      # ERROR: table [a] already defined
b = 2    # ERROR: key 'b' already defined
```

This prevents configuration files from having conflicting definitions.

---

## 3. Array of Tables

### Syntax

```toml
[[products]]
name = "Hammer"
price = 9.99

[[products]]
name = "Nail"
price = 0.05
```

Maps to:
```json
{
  "products": [
    {"name": "Hammer", "price": 9.99},
    {"name": "Nail", "price": 0.05}
  ]
}
```

### Nesting Arrays of Tables

```toml
[[fruits]]
name = "apple"

[[fruits.varieties]]
name = "red delicious"

[[fruits.varieties]]
name = "granny smith"

[[fruits]]
name = "banana"

[[fruits.varieties]]
name = "plantain"
```

Result:
```json
{
  "fruits": [
    {
      "name": "apple",
      "varieties": [
        {"name": "red delicious"},
        {"name": "granny smith"}
      ]
    },
    {
      "name": "banana",
      "varieties": [
        {"name": "plantain"}
      ]
    }
  ]
}
```

Each `[[fruits]]` creates a new element. Subsequent `[[fruits.varieties]]` append to the **most recently defined** `[[fruits]]` element.

---

## 4. String Types — Four Kinds

### String Comparison

| Kind | Delimiters | Escapes | Multi-line | Use Case |
|:-----|:-----------|:--------|:-----------|:---------|
| Basic | `"..."` | Yes (`\n`, `\t`, `\uXXXX`) | No | Most strings |
| Multi-line basic | `"""..."""` | Yes | Yes | Long text |
| Literal | `'...'` | No (raw) | No | Regexes, paths |
| Multi-line literal | `'''...'''` | No (raw) | Yes | Raw blocks |

### Escape Sequences

| Sequence | Character |
|:---------|:----------|
| `\b` | Backspace |
| `\t` | Tab |
| `\n` | Line feed |
| `\f` | Form feed |
| `\r` | Carriage return |
| `\"` | Quotation mark |
| `\\` | Backslash |
| `\uXXXX` | Unicode (BMP) |
| `\UXXXXXXXX` | Unicode (full range) |

### Multi-Line String Trimming

```toml
str = """\
  The quick brown \
  fox jumps over \
  the lazy dog."""
```

A `\` at end of line trims the newline and leading whitespace of the next line. Result: `"The quick brown fox jumps over the lazy dog."`

---

## 5. Integer Representation

### Supported Bases

| Base | Prefix | Example | Value |
|:-----|:-------|:--------|:------|
| Decimal | (none) | `42` | 42 |
| Hexadecimal | `0x` | `0xDEAD_BEEF` | 3735928559 |
| Octal | `0o` | `0o755` | 493 |
| Binary | `0b` | `0b1010_0001` | 161 |

### Underscore Separators

Underscores between digits improve readability:

```toml
population = 8_000_000_000
color = 0xFF_CC_00
permissions = 0o755
```

### Integer Range

TOML integers are 64-bit signed:

$$-2^{63} \leq n \leq 2^{63} - 1$$
$$-9,223,372,036,854,775,808 \leq n \leq 9,223,372,036,854,775,807$$

---

## 6. Date and Time Types

### RFC 3339 / ISO 8601

```toml
odt = 2024-01-15T09:30:00-05:00     # Offset Date-Time
ldt = 2024-01-15T09:30:00           # Local Date-Time (no timezone)
ld  = 2024-01-15                     # Local Date
lt  = 09:30:00                       # Local Time
```

### Separators

The `T` between date and time can be replaced with a space:

```toml
dt = 2024-01-15 09:30:00-05:00      # also valid
```

### Fractional Seconds

```toml
precise = 2024-01-15T09:30:00.123456789Z
```

---

## 7. Inline Tables and Arrays

### Inline Tables

```toml
point = {x = 1, y = 2}
```

Constraints:
- Must be on a single line
- Cannot be extended after definition (no adding keys later)
- No trailing commas

### Arrays

```toml
# Homogeneous (recommended):
ports = [8001, 8002, 8003]

# Can span lines:
hosts = [
    "alpha.example.com",
    "beta.example.com",
]

# Trailing comma allowed in arrays (not inline tables)
```

### Mixed-Type Arrays

TOML 1.0 technically allows mixed-type arrays, but many implementations restrict to homogeneous arrays for practical reasons.

---

## 8. TOML vs Alternatives

### When to Use TOML

| Scenario | Best Choice | Why |
|:---------|:------------|:----|
| App configuration | TOML | Unambiguous, typed, comments |
| Data interchange | JSON | Universal parser support |
| Complex configuration | YAML | Anchors, multi-doc, mature ecosystem |
| Deep nesting (>3 levels) | JSON or YAML | TOML headers become verbose |
| Flat key-value config | TOML | Perfect fit |

### The Nesting Problem

TOML becomes verbose with deep nesting:

```toml
[servers.production.us-east-1.web.frontend]
port = 8080
```

Each level of nesting extends the table header. At 4+ levels, YAML or JSON may be clearer.

---

## 9. Summary of Key Properties

| Property | Value |
|:---------|:------|
| Grammar | Context-free, unambiguous |
| Types | 8 (string, int, float, bool, 3 date/time variants, array) |
| No null | Absent key = not defined |
| No implicit coercion | Type is always syntactically determined |
| String types | 4 (basic, literal, each with multi-line) |
| Integer range | 64-bit signed |
| Table uniqueness | Each table defined at most once |
| Key uniqueness | Each key defined at most once per table |
| Comments | `#` to end of line |

---

*TOML exists because INI files have no standard, JSON has no comments, and YAML has 80 pages of specification with implicit type coercion that turns your country code into a boolean. TOML trades expressiveness for predictability — what you write is what you get, every time, in every parser.*
