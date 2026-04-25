# The Internals of Unicode — Encoding, Normalization, and Segmentation Algorithms

> *Unicode is not "ASCII but bigger." It is a 1,114,112-codepoint code space partitioned across 17 planes, served by three primary encoding forms (UTF-8, UTF-16, UTF-32), three families of equivalence (canonical, compatibility, case), three families of segmentation (grapheme, word, line), and a property database with hundreds of attributes per character. Every "string" question — length, equality, ordering, slicing, lowercasing, searching — fragments into "in which code unit?", "under which equivalence?", "under which locale?". This deep dive walks the algorithms, byte patterns, and gotchas that the practical sheet glosses past.*

---

## 1. The Unicode Codepoint Space

### The Code Space

Unicode reserves a fixed code space of 1,114,112 codepoints, numbered U+0000 through U+10FFFF. The cap of U+10FFFF is not arbitrary: it is the largest codepoint that can be encoded in UTF-16 with a single surrogate pair (high surrogate U+DBFF + low surrogate U+DFFF), and the entire architecture is designed to keep all three encoding forms — UTF-8, UTF-16, UTF-32 — round-trip equivalent. Any value above U+10FFFF is, by definition, not a Unicode codepoint and must not appear in conformant data.

### The Seventeen Planes

The code space is divided into 17 planes of 65,536 codepoints each. A plane is just the high four hex digits of a codepoint (treated as a number from 0 to 16).

| Plane | Range | Name | Contents |
|:-----:|:------|:-----|:---------|
| 0 | U+0000–U+FFFF | BMP (Basic Multilingual Plane) | Most living scripts, common symbols, CJK common ideographs |
| 1 | U+10000–U+1FFFF | SMP (Supplementary Multilingual Plane) | Historic scripts, music, math, emoji |
| 2 | U+20000–U+2FFFF | SIP (Supplementary Ideographic Plane) | Rare CJK ideographs (CJK Ext B/C/D/E/F) |
| 3 | U+30000–U+3FFFF | TIP (Tertiary Ideographic Plane) | Oracle bone, even rarer CJK (Ext G/H) |
| 4–13 | U+40000–U+DFFFF | Unassigned | Reserved, all mostly empty |
| 14 | U+E0000–U+EFFFF | SSP (Supplementary Special-purpose Plane) | Tag characters, variation selectors supplement |
| 15 | U+F0000–U+FFFFF | SPUA-A (Supplementary Private Use Area A) | Vendor-defined |
| 16 | U+100000–U+10FFFF | SPUA-B (Supplementary Private Use Area B) | Vendor-defined |

Most "ordinary" text — Latin, Cyrillic, Greek, Arabic, Hebrew, Devanagari, Tamil, Thai, the most common 20,000 CJK ideographs — lives entirely on the BMP. Emoji and the long tail of historic scripts (Cuneiform, Egyptian Hieroglyphs, Linear B, Phoenician, Gothic, Old Italic, Tangut, Khitan Small Script) live on the SMP. Rare CJK characters that scholars need but most users will never see live on the SIP and TIP.

### Surrogates: The Reserved-and-Invalid Range

Codepoints U+D800–U+DFFF are permanently reserved and **never represent characters**. They exist only as a side effect of UTF-16's encoding scheme: U+D800–U+DBFF are "high surrogates" and U+DC00–U+DFFF are "low surrogates," and a pair of one of each encodes a single supplementary codepoint. In any other context — alone in UTF-8, alone in UTF-32, in a string-comparison algorithm, in a normalization algorithm — they are invalid.

```python
>>> chr(0xD800)
'\ud800'
>>> chr(0xD800).encode('utf-8')
Traceback (most recent call last):
  ...
UnicodeEncodeError: 'utf-8' codec can't encode character '\ud800' in position 0: surrogates not allowed
>>> chr(0xD800).encode('utf-8', errors='surrogatepass')
b'\xed\xa0\x80'
```

Python's `surrogatepass` error handler exists precisely because some real-world data (most often Windows filenames or JSON from buggy producers) contains lone surrogates, and the only way to round-trip it is to permit the technically invalid encoding. Rust solves the same problem with WTF-8 (see Section 5).

### Noncharacters

A small set of codepoints is permanently reserved as "noncharacters." They are valid Unicode codepoints (legal in any encoding form, can be normalized, can be transmitted) but they are **guaranteed never to be assigned a meaning**. Programs are explicitly permitted to use them as internal sentinels.

| Range | Count | Notes |
|:------|:-----:|:------|
| U+FDD0–U+FDEF | 32 | Original noncharacter block |
| U+xFFFE, U+xFFFF (one of each per plane) | 34 | The last two codepoints of every plane |

Total: 66 noncharacters. Common pattern: `U+FFFE` and `U+FFFF` appear as terminators in some legacy formats; `U+FDD0–U+FDEF` is sometimes used as an internal "this is not real text" marker in Unicode-aware algorithms.

### Private Use Area

Three blocks are designated for vendor-, application-, or user-assigned characters that the Unicode Consortium will never standardize.

| Block | Range | Codepoints |
|:------|:------|:----------:|
| BMP PUA | U+E000–U+F8FF | 6,400 |
| Plane 15 PUA | U+F0000–U+FFFFD | 65,534 |
| Plane 16 PUA | U+100000–U+10FFFD | 65,534 |

Apple's logo (U+F8FF in macOS), the Klingon alphabet (informally allocated in the BMP PUA by the ConScript Unicode Registry), and many enterprise glyphs sit here. The contract is: by mutual agreement between sender and receiver, any codepoint in these ranges can mean anything. The Unicode Consortium will never assign a standard meaning, so these are safe forever.

### Blocks

The Unicode Character Database organizes assigned codepoints into named "blocks." A block is just a contiguous range with a name. Blocks have no semantic weight in algorithms — they're navigation aids for `Blocks.txt`. Compare with the **Script** property (Section 11), which is what algorithms actually look at.

```python
import unicodedata
for cp in [0x0041, 0x03A9, 0x05D0, 0x4E2D, 0x1F600, 0x1F1FA]:
    print(f"U+{cp:04X}  {chr(cp)}  {unicodedata.name(chr(cp))}")
```

Output:
```
U+0041  A     LATIN CAPITAL LETTER A
U+03A9  Ω     GREEK CAPITAL LETTER OMEGA
U+05D0  א     HEBREW LETTER ALEF
U+4E2D  中    CJK UNIFIED IDEOGRAPH-4E2D
U+1F600 😀    GRINNING FACE
U+1F1FA 🇺    REGIONAL INDICATOR SYMBOL LETTER U
```

---

## 2. UTF-8 Encoding — Algorithm and Properties

UTF-8 is the dominant encoding on the modern web, in modern operating systems, and in modern protocols. It encodes each codepoint in 1 to 4 bytes using a self-synchronizing, ASCII-compatible variable-length scheme.

### The Byte Patterns

| Codepoint range | Bytes | Bit pattern |
|:----------------|:-----:|:------------|
| U+0000–U+007F | 1 | `0xxxxxxx` |
| U+0080–U+07FF | 2 | `110xxxxx 10xxxxxx` |
| U+0800–U+FFFF | 3 | `1110xxxx 10xxxxxx 10xxxxxx` |
| U+10000–U+10FFFF | 4 | `11110xxx 10xxxxxx 10xxxxxx 10xxxxxx` |

The leading bits of the first byte tell you the length of the sequence:

- `0xxxxxxx` — 1-byte sequence (ASCII).
- `110xxxxx` — first byte of a 2-byte sequence.
- `1110xxxx` — first byte of a 3-byte sequence.
- `11110xxx` — first byte of a 4-byte sequence.
- `10xxxxxx` — continuation byte (cannot be a start byte).
- `11111xxx` — invalid (used to be defined for 5- and 6-byte sequences, since restricted).

### ASCII Compatibility

The 1-byte form covers exactly U+0000–U+007F, which is byte-for-byte identical with ASCII. Every valid ASCII file is a valid UTF-8 file with the same bytes. This means **every byte-oriented program that does not interpret high-bit characters works correctly on UTF-8**: tokenizers that split on ASCII whitespace, Makefiles, JSON parsers (when they use only ASCII syntax), C compilers, even `grep`-on-byte-patterns. This single property is why UTF-8 won.

### Self-Synchronization

Because continuation bytes are uniquely tagged with `10xxxxxx`, you can locate a codepoint boundary from any random byte position by walking forward (or backward) until you find a non-continuation byte. This is why UTF-8 is robust to truncation and concatenation. Compare with Shift-JIS, where you must scan from the beginning to disambiguate single- vs double-byte characters.

```c
// Find the start of the codepoint containing byte position i
size_t codepoint_start(const unsigned char *s, size_t i) {
    while (i > 0 && (s[i] & 0xC0) == 0x80) i--;
    return i;
}
```

### Canonical Encoder

```python
def utf8_encode(cp: int) -> bytes:
    if cp < 0:
        raise ValueError("negative codepoint")
    if 0xD800 <= cp <= 0xDFFF:
        raise ValueError("surrogates are not encodable in UTF-8")
    if cp > 0x10FFFF:
        raise ValueError("beyond Unicode range")
    if cp < 0x80:
        return bytes([cp])
    if cp < 0x800:
        return bytes([0xC0 | (cp >> 6),
                      0x80 | (cp & 0x3F)])
    if cp < 0x10000:
        return bytes([0xE0 | (cp >> 12),
                      0x80 | ((cp >> 6) & 0x3F),
                      0x80 | (cp & 0x3F)])
    return bytes([0xF0 | (cp >> 18),
                  0x80 | ((cp >> 12) & 0x3F),
                  0x80 | ((cp >> 6) & 0x3F),
                  0x80 | (cp & 0x3F)])
```

### Canonical Decoder

```python
def utf8_decode_one(buf: bytes, i: int) -> tuple[int, int]:
    """Decode one codepoint starting at i. Return (codepoint, next_index)."""
    b0 = buf[i]
    if b0 < 0x80:
        return b0, i + 1
    if b0 < 0xC2:                      # 0x80-0xBF stray continuation, 0xC0-0xC1 overlong
        raise ValueError("invalid start byte")
    if b0 < 0xE0:
        if i + 1 >= len(buf):
            raise ValueError("truncated")
        b1 = buf[i + 1]
        if (b1 & 0xC0) != 0x80:
            raise ValueError("expected continuation byte")
        cp = ((b0 & 0x1F) << 6) | (b1 & 0x3F)
        if cp < 0x80:
            raise ValueError("overlong 2-byte form")
        return cp, i + 2
    if b0 < 0xF0:
        if i + 2 >= len(buf):
            raise ValueError("truncated")
        b1, b2 = buf[i + 1], buf[i + 2]
        if (b1 & 0xC0) != 0x80 or (b2 & 0xC0) != 0x80:
            raise ValueError("expected continuation byte")
        cp = ((b0 & 0x0F) << 12) | ((b1 & 0x3F) << 6) | (b2 & 0x3F)
        if cp < 0x800:
            raise ValueError("overlong 3-byte form")
        if 0xD800 <= cp <= 0xDFFF:
            raise ValueError("surrogate in UTF-8")
        return cp, i + 3
    if b0 < 0xF5:
        if i + 3 >= len(buf):
            raise ValueError("truncated")
        b1, b2, b3 = buf[i + 1], buf[i + 2], buf[i + 3]
        if any((b & 0xC0) != 0x80 for b in (b1, b2, b3)):
            raise ValueError("expected continuation byte")
        cp = ((b0 & 0x07) << 18) | ((b1 & 0x3F) << 12) | ((b2 & 0x3F) << 6) | (b3 & 0x3F)
        if cp < 0x10000:
            raise ValueError("overlong 4-byte form")
        if cp > 0x10FFFF:
            raise ValueError("beyond Unicode range")
        return cp, i + 4
    raise ValueError("invalid start byte (>= 0xF5)")
```

### Validity Rules

A strict UTF-8 decoder rejects:

1. **Overlong forms.** The codepoint U+0000 *can* technically be encoded as `C0 80` (2-byte form storing all zeros), but the spec mandates the shortest possible encoding. Overlong encodings of `/` (`C0 AF` instead of `2F`) were a real path-traversal exploit vector in the early 2000s.
2. **Surrogate codepoints (U+D800–U+DFFF).** They cannot appear in valid UTF-8.
3. **Codepoints > U+10FFFF.** A 4-byte sequence with first byte ≥ `F5` is invalid (originally UTF-8 supported up to `FD` for 6-byte sequences, but RFC 3629 capped it at 4 bytes for U+10FFFF).
4. **Lone continuation bytes.** A byte in `80–BF` not preceded by a valid start byte.
5. **Truncated sequences.** A start byte followed by fewer continuation bytes than its length indicates.

### The UTF-8 BOM

UTF-8 has no byte order. There is no high or low byte to swap. Yet some Microsoft tools prepend `EF BB BF` (the UTF-8 encoding of U+FEFF, the BOM) to files as a "this is UTF-8" marker. This is **rarely useful and frequently harmful**: it breaks Unix shebangs, JSON parsers, XML parsers (that already self-identify), and anything that assumes the file starts with its actual content. Every modern best-practice doc says: do not emit a UTF-8 BOM. If you receive one, strip it.

```bash
# Detect UTF-8 BOM
head -c 3 file.txt | xxd

# Strip UTF-8 BOM in place
sed -i '1s/^\xEF\xBB\xBF//' file.txt
```

---

## 3. UTF-16 Encoding — Surrogate Pairs

UTF-16 encodes each codepoint as one or two 16-bit code units.

### BMP: One Code Unit

Codepoints U+0000–U+FFFF (the BMP) encode directly as the 16-bit value of the codepoint, *except* that the surrogate range U+D800–U+DFFF is excluded — those are reserved precisely so they can be used as the markers for the supplementary form.

### Supplementary: A Surrogate Pair

For codepoints U+10000–U+10FFFF:

```
U' = U - 0x10000          # 20-bit value, 0..0xFFFFF
high = 0xD800 | (U' >> 10)        # 0xD800..0xDBFF
low  = 0xDC00 | (U' & 0x3FF)      # 0xDC00..0xDFFF
```

The decode is the inverse:

```
codepoint = 0x10000 + ((high - 0xD800) << 10) + (low - 0xDC00)
```

### Canonical Encoder and Decoder

```python
def utf16_encode_codepoint(cp: int) -> list[int]:
    if 0xD800 <= cp <= 0xDFFF:
        raise ValueError("surrogate")
    if cp < 0x10000:
        return [cp]
    if cp > 0x10FFFF:
        raise ValueError("beyond range")
    u = cp - 0x10000
    return [0xD800 | (u >> 10), 0xDC00 | (u & 0x3FF)]

def utf16_decode(units: list[int]) -> list[int]:
    out, i, n = [], 0, len(units)
    while i < n:
        u = units[i]
        if 0xD800 <= u <= 0xDBFF:
            if i + 1 >= n:
                raise ValueError("truncated surrogate pair")
            lo = units[i + 1]
            if not 0xDC00 <= lo <= 0xDFFF:
                raise ValueError("high surrogate not followed by low")
            cp = 0x10000 + ((u - 0xD800) << 10) + (lo - 0xDC00)
            out.append(cp)
            i += 2
        elif 0xDC00 <= u <= 0xDFFF:
            raise ValueError("lone low surrogate")
        else:
            out.append(u)
            i += 1
    return out
```

### UTF-16BE vs UTF-16LE

A 16-bit code unit must be serialized to bytes in some order:

- **UTF-16BE (big-endian):** high byte first. `0x0041` → `00 41`.
- **UTF-16LE (little-endian):** low byte first. `0x0041` → `41 00`.

When the byte order is unspecified, a **BOM** (Byte Order Mark, U+FEFF) at the start of the stream tells the decoder which order:
- `FE FF` → big-endian.
- `FF FE` → little-endian.

The BOM character is U+FEFF "ZERO WIDTH NO-BREAK SPACE" — chosen because U+FFFE is a noncharacter (so it cannot occur naturally and seeing it after byte-swapping immediately reveals the wrong order).

### Languages That Use UTF-16 Internally

| System | String type | Internal encoding |
|:-------|:------------|:------------------|
| Java | `String`, `char[]` | UTF-16 |
| JavaScript / ECMAScript | strings | UTF-16 (with lone surrogates allowed) |
| .NET / C# | `String`, `char` | UTF-16 |
| Windows API (`-W` functions) | `WCHAR*` | UTF-16 |
| ICU | `UChar` arrays | UTF-16 |

This is a historical artifact: in the early 1990s, when Java and Windows NT were designed, Unicode was thought to be a 16-bit fixed-width encoding (UCS-2). Supplementary planes were added later, and these systems retrofitted surrogate pairs. Today this means JavaScript's `string.length` returns the count of UTF-16 code units, so `"😀".length === 2` even though it is one user-visible character.

```javascript
"a".length        // 1
"😀".length       // 2 (one surrogate pair)
[..."😀"].length  // 1 (codepoint iteration, ES2015+)
```

```java
"a".length()         // 1
"😀".length()        // 2
"😀".codePointCount(0, "😀".length())  // 1
```

---

## 4. UTF-32 — Fixed-Width Codepoint

UTF-32 stores each codepoint as a single 32-bit code unit. There is exactly one code unit per codepoint, so `string.length` in UTF-32 equals the codepoint count.

| Encoding | Bytes per codepoint | Endianness | BOM |
|:---------|:-------------------:|:-----------|:---:|
| UTF-32BE | 4 | big-endian | `00 00 FE FF` |
| UTF-32LE | 4 | little-endian | `FF FE 00 00` |
| UTF-32 (with BOM) | 4 | self-identifying | one of the above |

### When UTF-32 Is Used

- **Unix `wchar_t`** is 32-bit on Linux and most Unixes (`sizeof(wchar_t) == 4`), so `wchar_t*` is effectively UTF-32 internally. `mbstowcs` / `wcstombs` convert between locale encoding and `wchar_t`.
- **Python's internal `str` representation (PEP 393, since 3.3)** dynamically picks 1, 2, or 4 bytes per codepoint based on the highest codepoint in the string. A pure-ASCII string uses 1 byte; a string with a BMP character uses 2; a string with any supplementary character uses 4. So `len()` in Python 3 always returns codepoint count, regardless of actual storage.
- **Some text-processing libraries** convert internally to UTF-32 because random codepoint indexing is O(1).

### Trade-offs

UTF-32 is conceptually simple but wasteful: a megabyte of pure ASCII English becomes 4 megabytes. This is why no major file format or wire protocol stores UTF-32 — it's only seen as an in-memory representation. UTF-32 also still requires endianness handling, which surprises people.

```c
#include <wchar.h>
#include <locale.h>
#include <stdio.h>

int main(void) {
    setlocale(LC_ALL, "");
    wchar_t s[] = L"Hello, 世界! 😀";
    printf("wchar_t size: %zu bytes\n", sizeof(wchar_t));
    printf("array length: %zu wchar_t units\n", sizeof(s)/sizeof(wchar_t) - 1);
    return 0;
}
```

On Linux, `sizeof(wchar_t)` is 4 and the array length is the codepoint count (8 for "Hello, 世界! 😀"). On Windows, `sizeof(wchar_t)` is 2 and the array length is the UTF-16 code unit count (so the emoji counts as 2).

---

## 5. Validation Rules

The strict UTF-8 spec (RFC 3629) is intentionally narrower than the original UTF-8 (Pike & Thompson, 1992). Several variants exist for different ecosystems.

### Strict UTF-8 (RFC 3629)

- **No overlong forms.** The shortest possible encoding must be used.
- **No surrogates.** U+D800–U+DFFF cannot appear.
- **No values > U+10FFFF.** Maximum 4-byte sequences only.

### Modified UTF-8 (Java)

Java's `DataOutputStream.writeUTF` and JNI use a **modified UTF-8**:

- U+0000 is encoded as the overlong 2-byte sequence `C0 80` (so that NUL can be embedded in a length-prefixed string without terminating it).
- Supplementary codepoints are encoded as a UTF-16 surrogate pair, with each surrogate further encoded as a 3-byte UTF-8 sequence (this is **CESU-8**, see below).

So a single supplementary codepoint (e.g., 😀 = U+1F600) takes **6 bytes** in modified UTF-8 (3 bytes per surrogate × 2 surrogates) instead of 4 bytes in standard UTF-8.

### CESU-8

**Compatibility Encoding Scheme for UTF-16: 8-Bit.** Each UTF-16 code unit is independently encoded as UTF-8. Surrogates are *not* combined first. So U+10000–U+10FFFF takes 6 bytes (two 3-byte UTF-8-encoded surrogates) instead of 4. Created for systems that store UTF-16 code-unit indices in databases and need byte sorting to match those indices. **Not** standards-compliant UTF-8.

### WTF-8 (Rust, Unix `OsStr`, the Web)

**Wobbly Transformation Format — 8-bit.** Standard UTF-8, *except* that lone surrogates (U+D800–U+DFFF) are also allowed (each encoded as a 3-byte sequence as if it were any other codepoint). This is necessary because:

- **Windows filesystem strings** are UTF-16, but Windows does not enforce well-formedness, so a path can contain unpaired surrogates.
- **JavaScript strings** can contain lone surrogates (you can construct one with `String.fromCharCode(0xD800)`).
- Round-tripping these into a UTF-8 representation and back without loss requires WTF-8.

Rust's `OsString` and `OsStr` represent OS-given strings (filenames, environment variables, command-line arguments) using a WTF-8-like internal encoding precisely so that no information is lost when converting between native OS strings and Rust's `String`.

```rust
use std::ffi::OsString;
use std::os::unix::ffi::OsStringExt;

// On Unix, OsString is a sequence of bytes.
// A non-UTF-8 filename can be represented but cannot be losslessly
// converted to String.
let bytes: Vec<u8> = vec![0xff, 0xfe, b'h', b'i'];
let os_string = OsString::from_vec(bytes);
match os_string.into_string() {
    Ok(_)  => unreachable!(),
    Err(_) => println!("not valid UTF-8 — use OsString"),
}
```

| Variant | Overlong | Surrogates | > U+10FFFF | Notes |
|:--------|:--------:|:----------:|:----------:|:------|
| UTF-8 (RFC 3629) | forbidden | forbidden | forbidden | The standard |
| Modified UTF-8 (Java) | only `C0 80` for U+0000 | as CESU-8 pair | forbidden | Embedded-NUL safe |
| CESU-8 | forbidden | as separate 3-byte each | forbidden | UTF-16-binary-sort compat |
| WTF-8 | forbidden | each as 3-byte | forbidden | Round-trips ill-formed UTF-16 |

---

## 6. Normalization — The Four Forms

Two strings can look identical, print identically, mean identically, yet differ byte-for-byte. Normalization is the process of choosing a canonical representation so that equivalent strings become byte-equal.

### The Canonical Example

The character `é` has two representations:

| Form | Codepoints | Hex |
|:-----|:-----------|:----|
| Precomposed | U+00E9 | `c3 a9` (UTF-8) |
| Decomposed | U+0065 U+0301 | `65 cc 81` (UTF-8) |

```python
>>> a = "é"
>>> b = "é"
>>> a == b
False
>>> import unicodedata
>>> unicodedata.normalize("NFC", a) == unicodedata.normalize("NFC", b)
True
```

### Two Equivalences

**Canonical equivalence.** Two sequences that should be treated as identical for *all* text-processing purposes. `é` precomposed and `e` + combining acute are canonically equivalent.

**Compatibility equivalence.** Two sequences that have the same abstract character but differ in formatting. The fullwidth digit `２` (U+FF12) and the regular digit `2` (U+0032) are compatibility-equivalent. The ligature `ﬁ` (U+FB01) and the two-letter sequence `fi` are compatibility-equivalent. Compatibility equivalence is **lossy**: `２` is visually distinct from `2`, but they're "the same character" for many purposes (search, indexing, identifier comparison).

### The Four Forms

| Form | Decompose | Recompose | Equivalence |
|:-----|:----------|:----------|:------------|
| NFD | Canonical | No | Canonical |
| NFC | Canonical | Yes (canonical) | Canonical |
| NFKD | Compatibility | No | Compatibility |
| NFKC | Compatibility | Yes (canonical) | Compatibility |

### Trade-offs and Recommendations

| Use case | Recommended | Why |
|:---------|:-----------:|:----|
| Web pages, JSON output | NFC | W3C recommendation, most compact |
| HFS+ filesystem (legacy macOS) | NFD | The OS stored filenames in NFD |
| APFS (modern macOS) | "Normalization-insensitive" | Comparison-time NFD, storage as-given |
| Identifier comparison (login, IDN) | NFKC + case-fold | Defeat homograph variants |
| Searching | NFKC | Match `２` as `2`, `ﬁ` as `fi` |
| Display (line layout) | NFC | Engine pre-composes anyway |
| Cryptographic hashing of "the same text" | NFC (specify it!) | Otherwise same text → different hash |

The general rule: **NFC for storage, NFC for transmission, NFKC + case-fold for identifier comparison.**

```python
import unicodedata

samples = ["é", "é", "ﬁ", "２", "Ω", "Å"]
for s in samples:
    nfc = unicodedata.normalize("NFC", s)
    nfd = unicodedata.normalize("NFD", s)
    nfkc = unicodedata.normalize("NFKC", s)
    nfkd = unicodedata.normalize("NFKD", s)
    print(f"{s!r:10}  NFC={nfc!r:8}  NFD={nfd!r:12}  NFKC={nfkc!r:8}  NFKD={nfkd!r:12}")
```

---

## 7. Normalization Algorithm — UAX #15

The full normalization pipeline is documented in UAX #15 ("Unicode Normalization Forms"). The reference implementation is in ICU; pure-Python re-implementations (e.g., `unicodedata2`) match it byte-for-byte.

### Step 1 — Decompose

For each codepoint, look it up in `UnicodeData.txt`. If it has a *decomposition mapping*, recursively decompose into its mapped sequence.

There are two kinds of mapping:

- **Canonical decomposition** — used in NFD/NFC.
- **Compatibility decomposition** — used in NFKD/NFKC. (Compatibility decompositions include all canonical ones plus extras for ligatures, fullwidth/halfwidth, superscript/subscript, etc.)

The decomposition is recursive: if `é` decomposes to `e + ́`, and `ḗ` (e with macron and acute) decomposes to `ē + ́`, which itself decomposes to `e + ̄ + ́`, the algorithm fully unwinds.

Hangul syllables are special-cased. Every precomposed Hangul syllable in U+AC00–U+D7A3 algorithmically decomposes into a Lead+Vowel(+Tail) jamo sequence using arithmetic, not table lookup:

```python
SBASE, LBASE, VBASE, TBASE = 0xAC00, 0x1100, 0x1161, 0x11A7
LCOUNT, VCOUNT, TCOUNT = 19, 21, 28
NCOUNT = VCOUNT * TCOUNT          # 588
SCOUNT = LCOUNT * NCOUNT          # 11172

def hangul_decompose(s: int) -> list[int]:
    sindex = s - SBASE
    if not 0 <= sindex < SCOUNT:
        return [s]
    L = LBASE + sindex // NCOUNT
    V = VBASE + (sindex % NCOUNT) // TCOUNT
    T = TBASE + sindex % TCOUNT
    return [L, V] if T == TBASE else [L, V, T]
```

### Step 2 — Canonical Reorder

Adjacent combining marks are sorted in non-decreasing order of their **Canonical Combining Class (CCC)**. This is a tiny stable sort that runs only over runs of combining marks (CCC ≠ 0).

| Codepoint | Name | CCC |
|:----------|:-----|:---:|
| U+0301 | combining acute | 230 |
| U+0327 | combining cedilla | 202 |
| U+0316 | combining grave below | 220 |
| U+0308 | combining diaeresis | 230 |
| U+1AB0 | combining double inverted breve below | 220 |

The reorder makes equivalent sequences with marks in different orders yield identical output. `e` + cedilla (CCC 202) + acute (CCC 230) and `e` + acute (CCC 230) + cedilla (CCC 202) become the same output, with cedilla first (lower CCC).

CCC = 0 (most non-combining characters) acts as a barrier — the sort never crosses a CCC=0 boundary.

```python
def canonical_reorder(cps: list[int]) -> list[int]:
    out = list(cps)
    n = len(out)
    i = 0
    while i < n:
        # find a maximal run of CCC > 0
        j = i
        while j < n and ccc(out[j]) > 0:
            j += 1
        # stable sort the run by CCC
        out[i:j] = sorted(out[i:j], key=ccc)
        # advance past this run + the next non-combining character
        i = j + 1
    return out
```

### Step 3 — Compose (only for NFC/NFKC)

After decomposition and reordering, NFC/NFKC tries to recompose adjacent character pairs into precomposed forms by looking up the **canonical composition table** (the inverse of the canonical decomposition table, minus an explicit "Composition Exclusion" list — see UCD `CompositionExclusions.txt`).

Pseudocode:

```
starter = first codepoint
i = 1
while i < len(s):
    c = s[i]
    if can_compose(starter, c) and not blocked_by_intervening_marks:
        starter = compose(starter, c)
        s.delete(i)
    else:
        if ccc(c) == 0: starter = c
        i += 1
```

Hangul recomposition is again algorithmic, the inverse of decomposition.

### Composition Exclusions

Some canonical decompositions are *not* recomposed in NFC. This is to prevent NFC from changing visually-meaningful character choices:

- **Singletons** (e.g., `Ω` U+2126 OHM SIGN decomposes to `Ω` U+03A9 GREEK CAPITAL LETTER OMEGA — but the recomposition is intentionally excluded so OHM SIGN never re-appears).
- **Non-starter decompositions.**
- **Script-specific exclusions** (Hebrew points, Arabic shaping marks).

The exclusion list is short (~50 entries) and shipped as part of UCD.

### Performance

ICU's normalization is heavily optimized: most strings are already NFC and require zero allocation if validation reveals they are "Quick_Check=Yes". The `Quick_Check` UCD properties (`NFC_QC`, `NFD_QC`, `NFKC_QC`, `NFKD_QC`) are a fast-path filter; only when a string contains "maybe" or "no" characters does the full normalization run.

```python
def is_nfc(s: str) -> bool:
    return all(unicodedata.is_normalized("NFC", s) for _ in [None])
```

---

## 8. Grapheme Clusters and Segmentation — UAX #29

A "user-perceived character" is what a user clicks, deletes, or counts as one. It almost never lines up with one codepoint. Unicode formalizes this as a **grapheme cluster**, defined by UAX #29.

### Why It's Hard

| Visual | Codepoints | UTF-8 bytes |
|:-------|:-----------|:------:|
| `e` | 1 | 1 |
| `é` (precomposed) | 1 | 2 |
| `é` (decomposed) | 2 | 3 |
| `🇺🇸` (US flag) | 2 | 8 |
| `👨‍👩‍👧‍👦` (family) | 7 | 25 |
| `👋🏽` (waving hand, medium skin tone) | 2 | 8 |
| `한` (Hangul precomposed) | 1 | 3 |
| `한` (Hangul decomposed L+V+T) | 3 | 9 |

If you ask "give me the first 5 characters of this string," what you almost always mean is "give me the first 5 grapheme clusters." Slicing by codepoints will mid-split an emoji ZWJ sequence and corrupt the visual output.

### The Algorithm

UAX #29 defines a state machine: walk codepoints, classify each by its **Grapheme_Cluster_Break** property, and decide at every boundary whether to break.

The grapheme-break properties:

| Property | Examples |
|:---------|:---------|
| Other | Most letters, digits |
| CR, LF | Carriage return, line feed |
| Control | C0/C1 controls |
| Extend | Combining marks (most), variation selectors |
| ZWJ | Zero-width joiner U+200D |
| Regional_Indicator | U+1F1E6–U+1F1FF (flag halves) |
| Prepend | A few prefix marks |
| L, V, T, LV, LVT | Hangul jamo and syllables |
| SpacingMark | Vowel signs (Devanagari etc.) |
| Extended_Pictographic | All emoji |

The rules (paraphrased, see UAX #29 Table 1c):

- Break at start and end.
- Don't break between CR and LF (they're one cluster).
- Don't break between Hangul-syllable subparts (L+V or LV+T, etc.).
- Don't break before Extend, ZWJ, or SpacingMark.
- Don't break between two Regional_Indicators (flag emoji are pairs).
- Don't break across an emoji ZWJ sequence: `Pictographic Extend* ZWJ × Pictographic`.
- Otherwise, break.

### Reference Implementation

```python
# Pure-Python sketch of UAX #29 GB rules. Real implementations
# use a generated table; this is for clarity.
def grapheme_clusters(s: str) -> list[str]:
    if not s:
        return []
    clusters = []
    cur = s[0]
    for c in s[1:]:
        if not should_break(cur[-1], c, cur):
            cur += c
        else:
            clusters.append(cur)
            cur = c
    clusters.append(cur)
    return clusters
```

The real `should_break` requires lookahead/lookbehind for ZWJ sequences and the Regional_Indicator pairing rule (you must count consecutive RIs to know if a new one starts a new flag or pairs with the previous).

### Per-Language Libraries

| Language / lib | API |
|:---------------|:----|
| Python (stdlib) | None — must use `regex` package's `\X` |
| Python `regex` | `regex.findall(r"\X", s)` |
| JavaScript | `Intl.Segmenter("en", {granularity: "grapheme"})` |
| Rust crate | `unicode_segmentation::UnicodeSegmentation::graphemes` |
| Go | `golang.org/x/text/unicode/norm` + `runes`; or `github.com/rivo/uniseg` |
| Swift | `String.count` is grapheme count by default; iterate `for c in s` |
| Java | `java.text.BreakIterator.getCharacterInstance()` |
| ICU | `icu::BreakIterator::createCharacterInstance` |

```python
import regex
s = "👨‍👩‍👧‍👦 family"
print(len(s))                         # 9 (codepoints)
print(len(s.encode("utf-8")))         # 28 (bytes)
print(len(regex.findall(r"\X", s)))   # 8 (grapheme clusters: family + space + "family")
```

```javascript
const seg = new Intl.Segmenter("en", { granularity: "grapheme" });
const s = "👨‍👩‍👧‍👦 family";
console.log(s.length);                          // 18 (UTF-16 code units)
console.log([...seg.segment(s)].length);        // 8 (grapheme clusters)
```

```rust
use unicode_segmentation::UnicodeSegmentation;
fn main() {
    let s = "👨‍👩‍👧‍👦 family";
    println!("{}", s.len());                                 // 28 bytes
    println!("{}", s.chars().count());                        // 9 codepoints
    println!("{}", s.graphemes(true).count());                // 8 grapheme clusters
}
```

```go
import (
    "fmt"
    "github.com/rivo/uniseg"
)

func main() {
    s := "👨‍👩‍👧‍👦 family"
    fmt.Println(len(s))                       // 28 bytes
    fmt.Println(uniseg.GraphemeClusterCount(s)) // 8 grapheme clusters
}
```

```swift
let s = "👨‍👩‍👧‍👦 family"
print(s.count)                              // 8 (Swift Character == grapheme cluster)
print(s.unicodeScalars.count)               // 9 (codepoints)
print(s.utf16.count)                        // 18 (UTF-16 code units)
print(s.utf8.count)                         // 28 (bytes)
```

Note Swift gets this right by default — `String.count` counts grapheme clusters. Every other mainstream language requires opting in.

---

## 9. Word Segmentation and Line-Breaking

### Word Segmentation (UAX #29)

UAX #29 also defines word boundaries — useful for double-click-to-select, word counts, and tokenization. The rules are similar in spirit to grapheme rules but use the **Word_Break** property instead.

```javascript
const seg = new Intl.Segmenter("en", { granularity: "word" });
const s = "Hello, world! 你好世界。";
for (const part of seg.segment(s)) {
    console.log(`"${part.segment}"  isWord=${part.isWordLike}`);
}
```

Word segmentation is highly locale-sensitive: Thai and Khmer have no spaces, so word segmentation requires a dictionary or a statistical model. Chinese and Japanese also have no inter-word spaces but generally rely on character-class-based heuristics for "word-like" tokens.

### Line Breaking (UAX #14)

UAX #14 specifies where a line *may* be broken when text is wrapped to a fixed width. Each codepoint has a **Line_Break** property (CL, CP, QU, GL, NS, EX, SY, IS, PR, PO, NU, AL, ID, IN, HY, BA, BB, B2, ZW, CM, WJ, H2, H3, JL, JV, JT, RI, EB, EM, ZWJ, …). The algorithm is a state machine over pairs of LB classes producing one of **break opportunity**, **direct break**, **prohibited**, or **mandatory break**.

Why it's hard:

- Hyphens (`-`) usually allow breaks, but soft hyphens (U+00AD) only break when needed.
- CJK ideographs allow breaks before any character (no spaces), but not before closing punctuation.
- Numbers and units (`3 kg`) shouldn't be broken between number and unit.
- Multi-byte symbols (`«»`, `“”`) have asymmetric break rules.
- Word-joiner U+2060 prohibits breaks; ZWSP U+200B forces a break opportunity.
- Mandatory breaks: hard line feeds (LF/CR/CRLF), U+2028 LINE SEPARATOR, U+2029 PARAGRAPH SEPARATOR.

Production renderers (Pango, HarfBuzz, Chromium, WebKit, ICU) implement UAX #14. Pure-text wrapping with `textwrap` in Python is a *very* simplified approximation that doesn't handle CJK or RTL correctly.

```python
# ICU's line break iterator via PyICU
from icu import BreakIterator, Locale
text = "The quick brown fox 跳 over the 懒狗。"
it = BreakIterator.createLineInstance(Locale("en_US"))
it.setText(text)
breaks = list(it)
for a, b in zip([0] + breaks[:-1], breaks):
    print(repr(text[a:b]))
```

### Bidirectional Algorithm (UAX #9)

Right-to-left scripts (Hebrew, Arabic, Thaana, Syriac, NKo) need an algorithm to lay out runs of mixed-direction text. UAX #9 — the **Unicode Bidirectional Algorithm** — assigns each codepoint a *Bidi_Class* (L, R, AL, EN, AN, CS, ES, ET, BN, LRE, RLE, LRO, RLO, PDF, LRI, RLI, FSI, PDI, NSM, B, S, WS, ON) and applies a multi-stage process to determine the visual order of characters from logical (memory) order.

The high-level pipeline:

1. **Paragraph level** is determined (by the first strong character, or by an explicit override).
2. **Embedding levels** are assigned per-character based on directional formatting characters.
3. **Resolving weak types** — numbers and number-related characters take direction from context.
4. **Resolving neutrals** — punctuation and whitespace take direction from surrounding strong text.
5. **Resolving implicit levels** — final numerical levels for runs.
6. **Reordering** — runs at odd levels are reversed.

Practical impact: a Hebrew file containing `יום שני` and an English phone number `+1 800-555-1212` requires the bidi algorithm to lay out the phone number correctly within the surrounding Hebrew. Modern renderers do this transparently. UAX #9 also defines bidi-isolate characters (U+2066 LRI, U+2067 RLI, U+2068 FSI, U+2069 PDI) that explicitly bracket directional runs without leaking direction into surrounding text — added in Unicode 6.3 to fix isolation bugs in earlier embedding-only formats.

---

## 10. Case Folding — UAX #21

Case operations (lowercase, uppercase, titlecase) are frequently locale-dependent and frequently lossy.

### Locale Surprises

**Turkish dotless-i.** Turkish has *four* case relationships:

| Lower | Upper |
|:-----:|:-----:|
| `i` (U+0069 dotted small i) | `İ` (U+0130 capital I with dot above) |
| `ı` (U+0131 dotless small i) | `I` (U+0049 capital I, no dot) |

In Turkish, lowercasing `I` gives `ı` (dotless), not `i`. So `"TITLE".lower()` returns `"tıtle"` in a Turkish locale. This is why Java's `String.toLowerCase()` (no-arg) is famously a security bug in cross-locale code: it uses the JVM's default locale, so what runs in Istanbul does not match what runs in London.

```java
"TITLE".toLowerCase(Locale.US);         // "title"
"TITLE".toLowerCase(Locale.forLanguageTag("tr-TR"));  // "tıtle"
```

The **case-folding** API is locale-independent and intended for case-insensitive comparison. Java offers `String.equalsIgnoreCase` (a partial fix) and `String.toLowerCase(Locale.ROOT)` (a partial fix) but the most correct primitive is full case folding.

**German ß.** Lowercase `ß` (U+00DF, ESZETT) historically had no uppercase, so `"straße".toUpperCase()` gives `"STRASSE"` — a case-mapping that *changes the string length*. Unicode 5.1 added U+1E9E LATIN CAPITAL LETTER SHARP S, which is now the modern uppercase, but a great deal of software still uppercases to `SS`.

```python
"straße".upper()               # 'STRASSE'
"STRASSE".lower()              # 'strasse' — lost the ß
"straße".casefold()            # 'strasse' — for comparison
```

**Greek final sigma.** Greek lowercase sigma has two forms: `σ` (U+03C3) in the middle of words, `ς` (U+03C2) at the end. Uppercase is always `Σ` (U+03A3). So `"ΣΟΦΟΣ".lower()` should be `"σοφος"` *or* `"σοφoς"` depending on whether the algorithm is positional. UAX #21 defines a **Final_Sigma** condition for this exact case.

### Simple vs Full Case Mapping

- **Simple case mapping** is one codepoint to one codepoint. Fast, but incorrect for ß (gives `SS` requires two output codepoints) and for ligatures.
- **Full case mapping** allows one-to-many. `ß` → `SS`. `ﬁ` → `FI`. Necessary for correctness.

Unicode ships both: `UnicodeData.txt` has the simple mappings; `SpecialCasing.txt` has the full and locale-conditional ones.

### casefold() — The Comparison Primitive

`str.casefold()` (Python 3) implements UAX #21's *full case folding*. It is lossier than `lower()` but is the canonical primitive for case-insensitive comparison.

```python
def case_insensitive_equal(a: str, b: str) -> bool:
    import unicodedata
    return (unicodedata.normalize("NFKC", a).casefold()
            == unicodedata.normalize("NFKC", b).casefold())
```

Note the order: **NFKC first, then casefold**, then optionally NFKC again (the round-trip stabilizes).

In Rust:

```rust
use unicode_normalization::UnicodeNormalization;
use caseless::default_caseless_match_str;

fn ci_equal(a: &str, b: &str) -> bool {
    default_caseless_match_str(a, b)
}
```

In Go:

```go
import (
    "golang.org/x/text/cases"
    "golang.org/x/text/language"
)

func ciEqual(a, b string) bool {
    fold := cases.Fold()
    return fold.String(a) == fold.String(b)
}
```

---

## 11. Properties — General Category and Scripts

Every assigned codepoint has a **General Category** — a two-letter classification. This is the property regular-expression engines query when you write `\p{L}` or `\p{N}`.

### General Category

| Code | Name | Example |
|:----:|:-----|:--------|
| Lu | Uppercase letter | A, Σ, Б |
| Ll | Lowercase letter | a, σ, б |
| Lt | Titlecase letter | ǅ |
| Lm | Modifier letter | ʰ, ʷ |
| Lo | Other letter (most CJK, Arabic, Hebrew) | 中, א, ا |
| Nd | Decimal digit | 0–9, ٠–٩, ०–९ |
| Nl | Letter number | Ⅰ, Ⅱ, Ⅲ |
| No | Other number | ½, ², ❶ |
| Mn | Non-spacing mark | combining accents |
| Mc | Spacing mark | Devanagari vowel signs |
| Me | Enclosing mark | combining circle |
| Pc | Connector punct | _ |
| Pd | Dash | - – — |
| Ps | Open punct | ( [ { |
| Pe | Close punct | ) ] } |
| Pi | Initial quote | “ ‘ « |
| Pf | Final quote | ” ’ » |
| Po | Other punct | . , ; : ! ? |
| Sm | Math symbol | + − × ÷ |
| Sc | Currency symbol | $ € £ ¥ |
| Sk | Modifier symbol | ^ ` |
| So | Other symbol | © ® ™ |
| Zs | Space separator | regular space, NBSP |
| Zl | Line separator | U+2028 |
| Zp | Paragraph separator | U+2029 |
| Cc | Control | C0/C1 |
| Cf | Format | ZWJ, ZWNJ, BOM |
| Cs | Surrogate | U+D800–U+DFFF |
| Co | Private use | U+E000–U+F8FF, planes 15/16 |
| Cn | Unassigned | reserved |

### Script

The **Script** property identifies the writing system. Common values: Latin, Greek, Cyrillic, Han, Hiragana, Katakana, Hangul, Arabic, Hebrew, Devanagari, Bengali, Tamil, Thai, Tibetan, Cherokee, Mongolian, Georgian, Armenian, Ethiopic, Common, Inherited, Unknown.

Two special values:

- **Common** — punctuation and other characters used across multiple scripts (e.g., `.`, `,`, ASCII digits).
- **Inherited** — combining marks; they take their script from the base character they attach to.

### Script_Extensions

Some characters are used in *more than one* script. The **Script_Extensions** property lists all scripts a codepoint participates in. Example: U+0640 ARABIC TATWEEL is used in Arabic, Persian, Urdu, Pashto, etc.; its Script is `Arabic`, but its Script_Extensions includes `{Arab, Mand, Mani, Phlp, Rohg, Sogd, Syrc}`.

When a regex says `\p{Script=Latin}`, modern engines test against Script_Extensions, not just Script. Otherwise common punctuation in Latin text would fail to match.

```python
import unicodedata
for cp in [0x0041, 0x0301, 0x0640, 0x4E2D, 0x1F600]:
    c = chr(cp)
    print(f"U+{cp:04X}  {c!r}  cat={unicodedata.category(c)}  "
          f"name={unicodedata.name(c)}")
```

### Properties for Regex

Modern regex engines (`pcre2`, `re2`, `regex` in Python, JavaScript with `u` flag, Rust `regex`) support `\p{…}` and `\P{…}` to match by Unicode property:

| Pattern | Matches |
|:--------|:--------|
| `\p{L}` | Any letter |
| `\p{Lu}` | Uppercase letter |
| `\p{N}` | Any number |
| `\p{Sc}` | Currency symbol |
| `\p{Script=Greek}` | Greek-script characters |
| `\p{Block=Cyrillic}` | Codepoints in the Cyrillic block |
| `\p{ASCII}` | ASCII (U+0000–U+007F) |
| `\p{Emoji}` | Emoji |
| `\p{Emoji_Presentation}` | Default-emoji-presentation |

```python
import regex
text = "Hello 世界 ¡Hola! ١٢٣"
print(regex.findall(r"\p{L}+", text))      # all letter runs
print(regex.findall(r"\p{Han}+", text))    # only Han
print(regex.findall(r"\p{Nd}+", text))     # only decimal digits (matches Arabic-Indic ١٢٣)
```

---

## 12. Encodings — Legacy and Conversion

Before Unicode won, the world ran on a patchwork of single-byte and double-byte encodings. Most are still alive in legacy data, embedded systems, and East Asian web pages.

### Single-Byte Encodings

| Encoding | Range | Notes |
|:---------|:------|:------|
| ASCII | 0x00–0x7F | The original 7-bit set |
| ISO-8859-1 (Latin-1) | 0x00–0xFF | Western European; **its 256 codepoints are exactly U+0000–U+00FF** |
| ISO-8859-2 (Latin-2) | 0x00–0xFF | Central European |
| ISO-8859-5 | 0x00–0xFF | Cyrillic |
| ISO-8859-15 (Latin-9) | 0x00–0xFF | Latin-1 with € replacing ¤ |
| Windows-1252 | 0x00–0xFF | Latin-1 + curly quotes etc. in 0x80–0x9F (Latin-1 has C1 controls there) |
| Windows-1251 | 0x00–0xFF | Cyrillic, used by Russian Windows |
| KOI8-R | 0x00–0xFF | Russian, structured so that stripping the high bit gives ASCII transliteration |
| Mac Roman | 0x00–0xFF | Pre-OSX Macintosh |

Critical fact: **the first 256 codepoints of Unicode (U+0000–U+00FF) are exactly Latin-1.** This means a Latin-1 byte `0xE9` decodes to U+00E9, which in UTF-8 is `c3 a9`. This bijection is why mojibake from Latin-1 ↔ UTF-8 confusion is so common (Section 16).

### CJK Multi-Byte Encodings

| Encoding | Region | Bytes | Notes |
|:---------|:-------|:-----:|:------|
| GB2312 | Mainland China | 2 | Predates Unicode |
| GBK | Mainland China | 1–2 | Superset of GB2312 |
| GB18030 | Mainland China | 1, 2, or 4 | **Mandatory in PRC.** Covers all Unicode |
| Big5 | Taiwan, HK | 1–2 | Traditional Chinese |
| Shift-JIS | Japan | 1–2 | Mixes JIS X 0201 (single-byte) and JIS X 0208 (double-byte) |
| EUC-JP | Japan (Unix) | 1–3 | Unix-side counterpart to Shift-JIS |
| ISO-2022-JP | Japan email | variable | Stateful 7-bit, used by old email |
| EUC-KR | Korea | 1–2 | Korean Unix |
| Big5-HKSCS | Hong Kong | 1–2 | Big5 + HK extensions |

### Conversion: iconv

```bash
# Convert a file from Shift-JIS to UTF-8
iconv -f SHIFT_JIS -t UTF-8 input.sjis > output.utf8

# Detect encoding (rough)
file -i mystery.txt
chardet mystery.txt          # heuristic detector

# Convert a UTF-8 file with fallback for non-Latin-1 chars
iconv -f UTF-8 -t ISO-8859-1//TRANSLIT input.utf8 > output.lat1
```

### The "UTF-8 Everywhere" Migration

The "UTF-8 Everywhere" manifesto (utf8everywhere.org) advocates:

- Internal application strings: UTF-8.
- File I/O: UTF-8.
- Wire protocols: UTF-8.
- Cross-platform APIs: UTF-8 strings convert at the OS boundary (Windows: UTF-16 ↔ UTF-8 wrappers).

This is now the consensus for new code. C++17 deprecated `<codecvt>`; C++20 added `char8_t` to type-distinguish UTF-8; C23 added `u8` string literal support; Rust's `&str` is guaranteed UTF-8; Go strings are byte slices conventionally UTF-8.

---

## 13. Bidi Algorithm — UAX #9

Section 9 sketched the Bidirectional Algorithm; here are more details.

### Bidi Classes

Every codepoint has a Bidi Class (24 values). The strong types:

| Class | Meaning |
|:-----:|:--------|
| L | Strong left-to-right (Latin, Cyrillic, Greek, Han, Hangul, …) |
| R | Strong right-to-left (Hebrew, Thaana) |
| AL | Arabic letter (RTL with Arabic shaping rules) |

The weak types:

| Class | Meaning |
|:-----:|:--------|
| EN | European number (0–9 in Latin context) |
| AN | Arabic-Indic number (٠–٩) |
| ES | European number separator (`+`, `-` between digits) |
| ET | European number terminator (`%`, `°`) |
| CS | Common number separator (`,`, `.`, `:`, `/`) |
| NSM | Non-spacing mark |
| BN | Boundary neutral |

The neutrals:

| Class | Meaning |
|:-----:|:--------|
| WS | Whitespace |
| ON | Other neutrals (most punctuation) |
| B | Paragraph separator |
| S | Segment separator (tab) |

The explicit formatting characters:

| Class | Codepoints | Meaning |
|:-----:|:-----------|:--------|
| LRE | U+202A | Left-to-right embedding |
| RLE | U+202B | Right-to-left embedding |
| LRO | U+202D | Left-to-right override |
| RLO | U+202E | Right-to-left override |
| PDF | U+202C | Pop directional formatting |
| LRI | U+2066 | Left-to-right isolate |
| RLI | U+2067 | Right-to-left isolate |
| FSI | U+2068 | First-strong isolate |
| PDI | U+2069 | Pop directional isolate |

### Why Isolates Replaced Embeddings

Embeddings (LRE/RLE/LRO/RLO/PDF) leak directional context: an unbalanced LRE inside a tweet would corrupt every subsequent line. Isolates (LRI/RLI/FSI/PDI, added in Unicode 6.3) wrap a chunk in its own self-contained directional bubble — even if the inner content is malformed, the outer paragraph is unaffected. New code uses isolates exclusively.

### LRM and RLM

The LRM (Left-to-Right Mark, U+200E) and RLM (Right-to-Left Mark, U+200F) are **invisible strong-direction characters**. They take no space but force directional resolution. Use case: `Hello, world! (مرحبا)` with a trailing English period — without an LRM, the period might attach to the Arabic. Inserting LRM after the closing parenthesis pins the period to the Latin run.

### Trojan-Source

In 2021, researchers showed that bidi override characters (RLO etc.) can be inserted in source-code comments to make malicious code render harmless in editors and review tools. CVE-2021-42574 ("Trojan Source"). The fix in most languages and tooling: ban bidi formatting characters in source files outside string literals; warn at compile time. Rust 1.56 and many linters now reject these.

```rust
// This looks like "if access_level != 'user'" but the closing
// quotation actually contains an RLO that reorders characters.
// Modern rustc rejects this.
let access_level = "user‮ ⁦// Check if admin⁩ ⁦";
```

---

## 14. Emoji and ZWJ Sequences

### The Emoji Code Space

Emoji are scattered:

| Range | Notes |
|:------|:------|
| U+2600–U+26FF | Misc Symbols (older) |
| U+2700–U+27BF | Dingbats (older) |
| U+1F300–U+1F5FF | Misc Symbols and Pictographs |
| U+1F600–U+1F64F | Emoticons |
| U+1F680–U+1F6FF | Transport and Map |
| U+1F900–U+1F9FF | Supplemental Symbols and Pictographs |
| U+1FA70–U+1FAFF | Symbols and Pictographs Extended-A |
| U+1F000–U+1F02F | Mahjong tiles |
| U+1F0A0–U+1F0FF | Playing cards |

### Variation Selectors

VS-15 (U+FE0E) requests the *text* presentation of a character. VS-16 (U+FE0F) requests the *emoji* presentation. Many older symbols (`☺`, `❤`, `✈`) default to text; appending VS-16 promotes them to emoji form.

```python
heart = "❤"            # ❤ (text by default in many fonts)
heart_emoji = "❤️" # ❤️ (forced emoji presentation)
print(heart, heart_emoji)
```

### Skin-Tone Modifiers (Fitzpatrick)

U+1F3FB through U+1F3FF (5 tones) attach to a base emoji to set skin tone:

```
👋 + 🏽  =  👋🏽
U+1F44B  U+1F3FD
```

A skin-tone-modified emoji is two codepoints, one grapheme cluster, four UTF-8 bytes per codepoint.

### Regional Indicators (Flags)

Flag emoji are *pairs* of regional-indicator letters U+1F1E6–U+1F1FF (`A`–`Z`):

```
U+1F1FA U+1F1F8  =  🇺🇸  (US flag)
U+1F1EF U+1F1F5  =  🇯🇵  (Japan flag)
U+1F1EC U+1F1E7  =  🇬🇧  (UK flag)
```

A flag is two codepoints, one grapheme cluster, eight UTF-8 bytes. Subdivisions like Scotland and Wales use **tag sequences** (much longer).

### Zero-Width Joiner Sequences

The Zero-Width Joiner (ZWJ, U+200D) glues emoji into a single grapheme cluster. Examples:

| Sequence | Codepoints | Result |
|:---------|:-----------|:-------|
| Family | 👨 ZWJ 👩 ZWJ 👧 ZWJ 👦 | 👨‍👩‍👧‍👦 |
| Family with two children | 👨 ZWJ 👨 ZWJ 👦 ZWJ 👦 | 👨‍👨‍👦‍👦 |
| Female firefighter | 👩 ZWJ 🚒 | 👩‍🚒 |
| Couple with heart | 👩 ZWJ ❤ ZWJ 💋 ZWJ 👨 | 👩‍❤️‍💋‍👨 |
| Rainbow flag | 🏳 VS-16 ZWJ 🌈 | 🏳️‍🌈 |
| Trans flag | 🏳 VS-16 ZWJ ⚧ | 🏳️‍⚧️ |

Each ZWJ sequence is **one grapheme cluster** spanning many codepoints. Unicode publishes the canonical list of recommended ZWJ sequences (RGI) annually.

### New Emoji Per Unicode Version

Each year's Unicode release adds emoji. As of Unicode 16 (2024), there are over 4,000 emoji counting all skin-tone and gender variants. Vendors render based on the Common Locale Data Repository (CLDR) and Unicode's emoji-data.txt. An emoji that doesn't render correctly is usually a sign that the platform's emoji fonts are out of date.

---

## 15. The Implementation Cost Model

Different languages answer "what is `len(string)`?" differently. This causes endless cross-team confusion.

| Language | `len(s)` returns |
|:---------|:-----------------|
| Python 3 | codepoints |
| Python 2 (`str`) | bytes |
| Python 2 (`unicode`) | UTF-16 code units (narrow build) or codepoints (wide build) |
| Rust `s.len()` | bytes (UTF-8) |
| Rust `s.chars().count()` | codepoints |
| Rust `s.graphemes(true).count()` | grapheme clusters (with `unicode-segmentation`) |
| Go `len(s)` | bytes |
| Go `utf8.RuneCountInString(s)` | codepoints |
| Java `s.length()` | UTF-16 code units |
| Java `s.codePointCount(0, len)` | codepoints |
| JavaScript `s.length` | UTF-16 code units |
| JavaScript `[...s].length` | codepoints |
| Swift `s.count` | grapheme clusters |
| C `strlen` | bytes |
| Ruby `s.length` (UTF-8 string) | codepoints |
| Perl `length(s)` | codepoints if UTF-8 flag is on, bytes otherwise |
| C# `s.Length` | UTF-16 code units |

This mismatch is the source of countless real bugs. A Twitter "280 characters" limit was changed to "280 [grapheme clusters in some scripts] but [560 UTF-16 code units in CJK]" because their internal counter was UTF-16-based.

### Cost Hierarchy

| Operation | Cost on UTF-8 string |
|:----------|:---------------------|
| Byte length | O(1) |
| Iterate codepoints | O(n) |
| Codepoint count | O(n) |
| Random codepoint indexing | O(n) |
| Grapheme cluster iteration | O(n) but with state machine |
| Grapheme cluster count | O(n) |
| Random grapheme indexing | O(n) |
| Case fold | O(n) |
| Normalize | O(n) — but with Quick_Check fast path that often touches no bytes |
| Collation key | O(n) — with multi-level weights |

If you need O(1) random access to codepoints, you must convert to UTF-32 (or, equivalently, build an index of byte offsets to codepoint boundaries).

### The Canonical Pattern

For most applications:

1. **Store strings as UTF-8.** It's compact for most data and ASCII-compatible.
2. **Iterate by codepoints when you need to inspect individual characters** (parsers, validators).
3. **Iterate by grapheme clusters when you need to display, truncate, or count "characters" as a user perceives them** (UIs, text fields).
4. **Normalize once** when receiving data; rely on canonical form afterward.
5. **Case-fold + normalize for case-insensitive comparison** rather than calling `.lower()` per side.

---

## 16. Common Bugs and Their Causes

### Mojibake — Double-Encoded UTF-8

**Symptom:** Pure ASCII looks fine, but `é` displays as `Ã©`, `→` displays as `â†’`, `™` displays as `â„¢`.

**Cause:** UTF-8 bytes were decoded as Latin-1 (which silently maps any byte to a codepoint), and then re-encoded as UTF-8.

```python
original = "café"
utf8_bytes = original.encode("utf-8")           # b'caf\xc3\xa9'
mistakenly_latin1 = utf8_bytes.decode("latin-1") # "café" (note 'Ã©')
double_encoded = mistakenly_latin1.encode("utf-8")  # b'caf\xc3\x83\xc2\xa9'
print(double_encoded.decode("utf-8"))           # 'café' — broken
```

**The signature:** `Ã ` followed by another character that originally had its high bit set. `Ã©` for `é`. `Ã ` for `à`. `â€™` for `’`.

**Fix:** the `ftfy` library specifically:

```python
import ftfy
ftfy.fix_text("Itâ€™s café")  # "It's café"
```

### Mojibake — Wrong Source Encoding

**Symptom:** Cyrillic text reads as `Ð¿Ñ€Ð¸Ð²ÐµÑ‚`. Japanese reads as `テスト`.

**Cause:** UTF-8 bytes interpreted as Latin-1 (Cyrillic) or as Shift-JIS (Japanese).

**Fix:** `ftfy.fix_encoding`, `chardet` for detection, or explicit re-decode:

```python
broken = "Ð¿Ñ€Ð¸Ð²ÐµÑ‚"
fixed = broken.encode("latin-1").decode("utf-8")  # "привет"
```

### Locale + Filename Interaction

On Linux, filenames are byte strings with no defined encoding. The OS does not know whether `café` was created with UTF-8 or Latin-1. If the locale is mismatched, `ls` shows mojibake for filenames created under a different locale.

```bash
# Emergency: convert filenames in current dir from Latin-1 to UTF-8
convmv -f latin1 -t utf-8 --notest *
```

### Normalization Mismatch

**Symptom:** Two strings *look* identical, both contain `é`, but `==` says they're different.

**Cause:** One uses U+00E9; the other uses U+0065 + U+0301.

**Fix:**

```python
import unicodedata
def eq(a, b):
    return unicodedata.normalize("NFC", a) == unicodedata.normalize("NFC", b)
```

This bites hard on macOS, where the default `Files` app sometimes hands you NFD strings. `git` once hit this so frequently it grew a `core.precomposeunicode` config to NFC-ify filenames in the index.

### Emoji Length Surprises

```python
# A Twitter-style "280 character" limit
def too_long(post: str) -> bool:
    return len(post) > 280
```

This counts codepoints. A post with 70 family-emoji ZWJ sequences (each 7 codepoints, one grapheme cluster) is 490 codepoints but only 70 user-visible characters. The user thinks they wrote a short post; the API rejects it. Real fix: count grapheme clusters with the `regex` module.

### Case-Insensitive Search False Negatives

```python
# Naive
def search(haystack: str, needle: str) -> bool:
    return needle.lower() in haystack.lower()

# Fails: "ﬁ" lowercase is "ﬁ", not "fi"
search("the file", "ﬁle")           # False — should be True
```

Better:

```python
def search_better(haystack: str, needle: str) -> bool:
    h = unicodedata.normalize("NFKC", haystack).casefold()
    n = unicodedata.normalize("NFKC", needle).casefold()
    return n in h
```

### Lone Surrogates in JSON

**Symptom:** A web service rejects valid-looking JSON with a "lone surrogate" error.

**Cause:** A JavaScript producer included a Windows filename (or a malformed UTF-16 string) that contained an unpaired surrogate. JSON specifies UTF-8, which cannot encode surrogates.

**Fix:** producer-side, sanitize with `WTF-8 → UTF-8` substitution (replace each lone surrogate with U+FFFD) before serializing.

---

## 17. Idioms at the Internals Depth

### Always Specify UTF-8

```python
# Python: open() defaults are platform-dependent on Windows.
# Always specify encoding.
with open("data.txt", encoding="utf-8") as f:
    text = f.read()

with open("out.txt", "w", encoding="utf-8", newline="\n") as f:
    f.write(text)
```

```c
// C: stop using char*-as-encoding-agnostic. Document UTF-8 explicitly.
void emit_utf8(const char *s);
```

```javascript
// Node: streams default to UTF-8. Explicit is better.
const text = await fs.promises.readFile("data.txt", "utf8");
```

```rust
// Rust: String is always UTF-8. Reading from a file checks validity.
let text = std::fs::read_to_string("data.txt")?;
```

```go
// Go: strings are byte slices. utf8.ValidString to check.
data, _ := os.ReadFile("data.txt")
if !utf8.Valid(data) { /* handle */ }
text := string(data)
```

### Encode Early, Decode Late

```
[network bytes] ──decode→ [internal Unicode strings] ──encode→ [network bytes]
                              ^
                              all string operations live here
```

Don't operate on bytes "as if they were strings" inside your application. Convert at the boundary. Internally, work with strings. On the way out, re-encode.

### Normalize on Ingest

```python
def ingest(text: str) -> str:
    return unicodedata.normalize("NFC", text)
```

Run this at every boundary where text enters your system. After this point, equality checks, sorts, and hashes are reliable.

### Casefold + Normalize for Comparison

```python
def compare_keys(a: str, b: str) -> bool:
    a = unicodedata.normalize("NFKC", a).casefold()
    b = unicodedata.normalize("NFKC", b).casefold()
    return a == b
```

### Segment by Grapheme for Truncation

```python
import regex

def truncate(s: str, n: int) -> str:
    clusters = regex.findall(r"\X", s)
    return "".join(clusters[:n])

truncate("👨‍👩‍👧‍👦 family time", 8)  # "👨‍👩‍👧‍👦 family"
```

A naive `s[:n]` on this string would split inside the family ZWJ sequence and produce malformed text.

### OsStr / OsString for Filesystems

```rust
use std::ffi::OsString;
use std::path::PathBuf;

fn read_filename(path: PathBuf) {
    // path may not be valid UTF-8 — use OsString
    let name: &std::ffi::OsStr = path.file_name().unwrap();
    if let Some(s) = name.to_str() {
        println!("UTF-8 name: {}", s);
    } else {
        println!("Non-UTF-8 filename, length {} bytes", name.len());
    }
}
```

```python
# Python: PathLike accepts both str and bytes; os.fsencode/fsdecode round-trip
import os
b = os.fsencode("café.txt")  # bytes appropriate for the OS
s = os.fsdecode(b)            # str, with surrogateescape for non-decodable bytes
```

The lesson: filesystem text is **not** guaranteed UTF-8 on Linux. The `OsStr`/`OsString` (Rust) and `os.fsencode`/`fsdecode` (Python) abstractions exist to round-trip non-UTF-8 paths losslessly.

### Use the System Library for Anything Hard

For any operation more complex than encode/decode, use a real Unicode library:

| Operation | Library |
|:----------|:--------|
| Normalization | `unicodedata` (Python), `unicode-normalization` (Rust), ICU (C/C++) |
| Grapheme clusters | `regex` (Python), `unicode-segmentation` (Rust), ICU, Swift native |
| Line breaking | ICU, Pango |
| Bidi | ICU, FriBidi |
| Collation | `pyuca`, `icu`, ICU4C |
| Case folding | ICU, `caseless` (Rust) |
| Locale-aware formatting | ICU, `babel` (Python), `Intl` (JavaScript) |

Re-implementing any of these from scratch is a significant project. ICU represents many person-decades of work and is the reference implementation for nearly every property in this document.

---

## Prerequisites

- ASCII (the 7-bit subset that all encodings preserve)
- Variable-length vs fixed-length encoding trade-offs
- Big-endian vs little-endian byte order
- The difference between a *codepoint* and a *codeunit* and a *grapheme cluster*
- C-string vs length-prefixed string semantics
- Bitwise operators (shifts and masks) for understanding UTF-8 byte patterns
- Familiarity with at least one regex engine that supports Unicode properties

---

## Complexity

| Operation | Time | Space |
|:----------|:-----|:------|
| UTF-8 encode (one codepoint) | O(1) | O(1) |
| UTF-8 decode (one codepoint) | O(1) amortized; up to 4 byte reads | O(1) |
| Encode/decode whole string | O(n) | O(n) |
| Validate UTF-8 (strict) | O(n) | O(1) |
| Surrogate pair encode/decode | O(1) | O(1) |
| Normalize (NFC/NFD/NFKC/NFKD) | O(n) worst case; O(n) Quick_Check fast path | O(n) |
| Canonical reorder | O(k log k) per run of k combining marks | O(k) |
| Hangul algorithmic decompose/compose | O(1) per syllable | O(1) |
| Grapheme cluster iteration | O(n) with state machine | O(1) extra |
| Word break / line break | O(n) with state machine | O(1) extra |
| Bidi resolution | O(n) per paragraph | O(n) |
| Case fold | O(n) | O(n) (one-to-many possible) |
| UCA collation key | O(n) per string; O(n) compare | O(n) |

---

## See Also

- unicode (sheet) — practical encode/decode and normalization commands
- regex — Unicode property classes (`\p{L}`, `\p{Sc}`, `\p{Script=Han}`)
- polyglot — comparing string semantics across multiple languages
- python — CPython's PEP 393 string representation, `unicodedata`
- javascript — UTF-16 code-unit strings, `Intl.Segmenter`
- go — UTF-8 byte slices, `golang.org/x/text`
- rust — UTF-8-guaranteed `&str`, `OsStr` for filesystems
- java — UTF-16 `String`, `Character.codePointAt`, `BreakIterator`
- c — `wchar_t` (UTF-32 on Linux, UTF-16 on Windows), `iconv`

## References

- Unicode Standard Core Specification — https://www.unicode.org/standard/standard.html
- Unicode Glossary — https://www.unicode.org/glossary/
- Unicode Charts — https://www.unicode.org/charts/
- UTF-8 Everywhere Manifesto — http://www.utf8everywhere.org
- UAX #15: Unicode Normalization Forms — https://www.unicode.org/reports/tr15/
- UAX #29: Unicode Text Segmentation — https://www.unicode.org/reports/tr29/
- UAX #18: Unicode Regular Expressions — https://www.unicode.org/reports/tr18/
- UAX #14: Unicode Line Breaking Algorithm — https://www.unicode.org/reports/tr14/
- UAX #9: Unicode Bidirectional Algorithm — https://www.unicode.org/reports/tr9/
- UAX #21: Case Mappings — https://www.unicode.org/reports/tr21/
- UAX #31: Unicode Identifier and Pattern Syntax — https://www.unicode.org/reports/tr31/
- UTS #10: Unicode Collation Algorithm — https://www.unicode.org/reports/tr10/
- UTS #39: Unicode Security Mechanisms — https://www.unicode.org/reports/tr39/
- UCA Default Allkeys — https://www.unicode.org/Public/UCA/latest/
- "Programming with Unicode" by Victor Stinner — https://unicodebook.readthedocs.io/
- RFC 3629: UTF-8, a transformation format of ISO 10646 — https://www.rfc-editor.org/rfc/rfc3629
- RFC 2781: UTF-16, an encoding of ISO 10646 — https://www.rfc-editor.org/rfc/rfc2781
- ICU User Guide — https://unicode-org.github.io/icu/userguide/
- Unicode Character Database — https://www.unicode.org/Public/UCD/latest/
