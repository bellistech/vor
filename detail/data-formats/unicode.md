# The Theory of Unicode — Encoding, Normalization, and Collation

> *Unicode is a character encoding standard that maps every human writing system to a codespace of 1,114,112 code points. UTF-8 is a variable-length encoding that packs code points into 1-4 bytes using a bit-packing scheme. Normalization (NFC/NFD/NFKC/NFKD) resolves equivalent representations. The Unicode Collation Algorithm (UCA) defines culturally-correct sorting using multi-level comparison.*

---

## 1. The Unicode Codespace

### Structure

| Property | Value |
|:---------|:------|
| Total code points | 1,114,112 ($\text{0x000000}$ to $\text{0x10FFFF}$) |
| Assigned characters | ~150,000 (Unicode 16.0) |
| Planes | 17 (0-16), each with 65,536 code points |
| BMP (Plane 0) | $\text{U+0000}$ to $\text{U+FFFF}$ — most common characters |
| SMP (Plane 1) | $\text{U+10000}$ to $\text{U+1FFFF}$ — emoji, historic scripts |
| Private Use | $\text{U+E000}$-$\text{U+F8FF}$, $\text{U+F0000}$-$\text{U+10FFFF}$ |
| Surrogates | $\text{U+D800}$-$\text{U+DFFF}$ — reserved for UTF-16 |

### Code Point Notation

$\text{U+0041}$ = LATIN CAPITAL LETTER A = `A`

A code point is **not** a character. A visible character (grapheme cluster) may be:
- One code point: `A` = U+0041
- Multiple code points: `e` = U+0065 + U+0301 (e + combining acute)
- Many code points: some emoji sequences are 7+ code points

---

## 2. UTF-8 Encoding Algorithm

### Bit Packing Rules

| Code Point Range | Bytes | Bit Pattern | Available Bits |
|:---:|:---:|:---|:---:|
| U+0000 - U+007F | 1 | `0xxxxxxx` | 7 |
| U+0080 - U+07FF | 2 | `110xxxxx 10xxxxxx` | 11 |
| U+0800 - U+FFFF | 3 | `1110xxxx 10xxxxxx 10xxxxxx` | 16 |
| U+10000 - U+10FFFF | 4 | `11110xxx 10xxxxxx 10xxxxxx 10xxxxxx` | 21 |

### Encoding Algorithm

Given code point $U$:

1. Determine byte count from range
2. Fill bits from **right to left** into the template
3. Each continuation byte holds 6 bits (prefix `10`)
4. The lead byte's prefix indicates byte count (number of leading 1s)

### Worked Example: U+00E9 (e with acute accent)

$$U = \text{0x00E9} = 233_{10} = 11101001_2$$

Range: $\text{0x0080}$-$\text{0x07FF}$ → 2 bytes → template: `110xxxxx 10xxxxxx`

Need 11 bits: $00011101001$

Split: $00011 | 101001$

```
Byte 1: 110|00011  = 0xC3
Byte 2: 10|101001  = 0xA9
```

UTF-8: `C3 A9`

### Worked Example: U+1F600 (grinning face emoji)

$$U = \text{0x1F600} = 000|011111|011000|000000_2$$

Range: $\text{0x10000}$-$\text{0x10FFFF}$ → 4 bytes

Split 21 bits: $000 | 011111 | 011000 | 000000$

```
Byte 1: 11110|000  = 0xF0
Byte 2: 10|011111  = 0x9F
Byte 3: 10|011000  = 0x98
Byte 4: 10|000000  = 0x80
```

UTF-8: `F0 9F 98 80`

### Key Properties of UTF-8

| Property | Consequence |
|:---------|:-----------|
| ASCII-compatible | All valid ASCII is valid UTF-8 |
| Self-synchronizing | Can find character boundaries from any byte |
| No embedded NULs | Safe for C strings (except U+0000 itself) |
| Sortable | Byte-order sorting = code point order |
| Overlong sequences invalid | `C0 80` is not valid UTF-8 for U+0000 |

---

## 3. UTF-16 Encoding and Surrogate Pairs

### BMP Characters (U+0000 - U+FFFF)

Encoded as a single 16-bit code unit: $\text{U+0041} \to \text{0x0041}$.

### Supplementary Characters (U+10000 - U+10FFFF)

Encoded as a **surrogate pair** — two 16-bit code units:

$$U' = U - \text{0x10000} \quad (0 \leq U' \leq \text{0xFFFFF}, \text{20 bits})$$
$$\text{High surrogate} = \text{0xD800} + (U' >> 10) \quad (\text{0xD800}-\text{0xDBFF})$$
$$\text{Low surrogate} = \text{0xDC00} + (U' \mathbin{\&} \text{0x3FF}) \quad (\text{0xDC00}-\text{0xDFFF})$$

### Decoding Surrogate Pair

$$U = (\text{high} - \text{0xD800}) \times \text{0x400} + (\text{low} - \text{0xDC00}) + \text{0x10000}$$

### Encoding Comparison

| Encoding | Min Bytes | Max Bytes | BMP Overhead | ASCII Overhead |
|:---------|:---:|:---:|:---:|:---:|
| UTF-8 | 1 | 4 | 1-3 | 1 (optimal) |
| UTF-16 | 2 | 4 | 2 (fixed) | 2 (wasteful) |
| UTF-32 | 4 | 4 | 4 (wasteful) | 4 (wasteful) |

---

## 4. Normalization Forms

### The Problem

Some characters have multiple representations:

```
"e" = U+00E9 (precomposed)
"e" = U+0065 U+0301 (decomposed: e + combining acute)
```

These look identical but are **different byte sequences**. String comparison fails without normalization.

### Four Normalization Forms

| Form | Decompose | Compose | Compatibility |
|:-----|:----------|:--------|:-------------|
| NFD | Canonical decomposition | No | No |
| NFC | Canonical decomposition | Then canonical composition | No |
| NFKD | Compatibility decomposition | No | Yes |
| NFKC | Compatibility decomposition | Then canonical composition | Yes |

### Canonical vs Compatibility Decomposition

**Canonical:** Different representations of the **same character**:
- $\text{U+00E9}$ (e) $\xrightarrow{\text{NFD}}$ $\text{U+0065 U+0301}$ (e + combining accent)

**Compatibility:** Characters that are **semantically similar** but visually different:
- $\text{U+FB01}$ (fi ligature) $\xrightarrow{\text{NFKD}}$ $\text{U+0066 U+0069}$ (f + i)
- $\text{U+2126}$ (ohm sign) $\xrightarrow{\text{NFKD}}$ $\text{U+03A9}$ (Greek capital omega)

### Canonical Ordering

When multiple combining marks are present, they're sorted by **Canonical Combining Class** (CCC):

$$\text{CCC}(\text{U+0301 acute}) = 230$$
$$\text{CCC}(\text{U+0327 cedilla}) = 202$$

Lower CCC comes first. Marks with the same CCC maintain relative order.

### Which Form to Use

| Use Case | Recommended Form |
|:---------|:-----------------|
| String comparison | NFC (most compact) |
| Search/indexing | NFKC (broadest matching) |
| Security (IDN) | NFKC (prevents homograph attacks) |
| Storage (general) | NFC (web standard, W3C recommendation) |

---

## 5. Grapheme Clusters

### What Users See vs What Unicode Stores

A **grapheme cluster** is what a user perceives as a single character. It may span multiple code points:

| Visual | Code Points | Count |
|:-------|:------------|:------|
| `e` | U+0065 U+0301 | 2 |
| `ga` (Hangul) | U+1100 U+1161 | 2 |
| Flag emoji | U+1F1FA U+1F1F8 (regional indicators) | 2 |
| Family emoji | U+1F468 U+200D U+1F469 U+200D U+1F467 | 5 |
| Skin tone emoji | U+1F44B U+1F3FD | 2 |

### String Length Confusion

```python
s = "e\u0301"           # é (decomposed)
len(s)                   # 2 (code points)
len(s.encode('utf-8'))   # 3 (bytes)
# Visual characters:      1 (grapheme clusters)
```

Three different "lengths" — code units, code points, grapheme clusters.

---

## 6. The Unicode Collation Algorithm (UCA)

### Multi-Level Comparison

UCA compares strings at multiple levels:

| Level | Distinguishes | Example |
|:------|:-------------|:--------|
| L1: Base | Different letters | a vs b |
| L2: Accents | Same letter, different diacritics | a vs a |
| L3: Case | Same letter+accent, different case | a vs A |
| L4: Punctuation | Ties at L3 | "co-op" vs "coop" |

### Algorithm

1. Map each character to a **collation element** (sequence of weights)
2. Compare L1 weights of all characters first
3. If equal, compare L2 weights
4. If equal, compare L3 weights
5. Continue until a difference is found

### Locale-Specific Ordering

| Language | Order |
|:---------|:------|
| English | a < b < c < ... < z |
| Swedish | ... < z < a < a < o |
| German (phonebook) | a = ae, o = oe, u = ue |
| Spanish (traditional) | ... < c < ch < d < ... < l < ll < m |

The same characters sort differently depending on locale.

---

## 7. Security Considerations

### Homograph Attacks

Visually identical characters from different scripts:

| Looks Like | Actually | Code Point |
|:-----------|:---------|:-----------|
| `a` | Cyrillic a | U+0430 |
| `o` | Greek omicron | U+03BF |
| `p` | Cyrillic er | U+0440 |

`paypal.com` could be spoofed with Cyrillic characters: `pаypal.com` (U+0430 instead of U+0061).

**Defense:** NFKC normalization + script mixing detection (IDNA 2008, UTS #39).

### Invisible Characters

| Character | Code Point | Risk |
|:----------|:-----------|:-----|
| Zero-width space | U+200B | Breaks string comparison |
| Zero-width joiner | U+200D | Changes emoji rendering |
| Right-to-left override | U+202E | Reverses text direction |
| Byte order mark | U+FEFF | Breaks file parsing |

---

## 8. Summary of Key Formulas

| Concept | Formula/Rule |
|:--------|:-------------|
| UTF-8 byte count | 1 byte: $U < 128$, 2: $U < 2048$, 3: $U < 65536$, 4: otherwise |
| Surrogate pair encode | $\text{high} = \text{0xD800} + ((U - \text{0x10000}) >> 10)$ |
| Surrogate pair decode | $U = (\text{high} - \text{0xD800}) \times \text{0x400} + (\text{low} - \text{0xDC00}) + \text{0x10000}$ |
| Normalization | NFD → decompose; NFC → decompose then compose |
| Collation | L1 (base) > L2 (accent) > L3 (case) > L4 (punct) |
| Total code points | $17 \times 65536 = 1,114,112$ |

---

*Unicode is not "ASCII but bigger." It's a complex system of encoding (UTF-8/16/32), equivalence (normalization), ordering (collation), segmentation (grapheme clusters), and security (homograph detection). Getting "string length" right requires specifying which of the three lengths you mean. Getting "string equality" right requires specifying a normalization form. Getting "string sorting" right requires specifying a locale.*
