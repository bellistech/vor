# The Theory of ASCII — Encoding, Control Characters, and Bit Layout

> *ASCII (American Standard Code for Information Interchange) is a 7-bit character encoding mapping 128 code points to characters. Its bit-level design is not arbitrary — arithmetic relationships between uppercase/lowercase letters, digit characters and their numeric values, and control character positions are deliberate and exploitable.*

---

## 1. The Encoding Space

### 7-Bit Layout

ASCII uses 7 bits, encoding exactly $2^7 = 128$ characters:

$$\text{Code point range: } 0x00 \text{ to } 0x7F \quad (0 \text{ to } 127)$$

| Range | Hex | Count | Category |
|:------|:----|:------|:---------|
| 0-31 | 0x00-0x1F | 32 | Control characters |
| 32-126 | 0x20-0x7E | 95 | Printable characters |
| 127 | 0x7F | 1 | DEL (control) |

### Bit Structure — The Two-Dimensional Table

ASCII is organized as a $2 \times 4$ grid of 32-character columns:

| Column (bits 6-5) | 00 | 01 | 10 | 11 |
|:-------------------|:---|:---|:---|:---|
| Range | 0x00-0x1F | 0x20-0x3F | 0x40-0x5F | 0x60-0x7F |
| Content | Control | Punctuation/Digits | Uppercase + symbols | Lowercase + symbols |

This structure means:
- **Bit 5** alone distinguishes uppercase from lowercase
- **Bit 6** separates control characters from printable
- **Bits 4-0** identify the character within its group

---

## 2. Arithmetic Properties of the Encoding

### Case Conversion via Bit 5

$$\text{uppercase}(c) = c \mathbin{\&} \text{0x5F} = c \mathbin{\&} \sim\text{0x20}$$
$$\text{lowercase}(c) = c \mathbin{|} \text{0x20}$$

| Character | Decimal | Binary | Bit 5 |
|:----------|:--------|:-------|:------|
| `A` | 65 | `100 0001` | 0 |
| `a` | 97 | `110 0001` | 1 |
| `Z` | 90 | `101 1010` | 0 |
| `z` | 122 | `111 1010` | 1 |

The difference is always exactly 32 ($2^5$):

$$\text{lowercase} = \text{uppercase} + 32$$

### Toggle Case via XOR

$$\text{toggle}(c) = c \oplus \text{0x20}$$

XOR with 0x20 flips bit 5, toggling between uppercase and lowercase.

### Digit Characters to Numeric Values

$$\text{digit\_value}(c) = c - \text{0x30} = c \mathbin{\&} \text{0x0F}$$

| Character | Code | $c - 48$ | $c \mathbin{\&} \text{0x0F}$ |
|:----------|:-----|:---------|:------------|
| `'0'` | 48 | 0 | 0 |
| `'5'` | 53 | 5 | 5 |
| `'9'` | 57 | 9 | 9 |

### Control Character from Letter

$$\text{Ctrl-}X = X \mathbin{\&} \text{0x1F}$$

| Key Combo | Letter Code | Result | Meaning |
|:----------|:---:|:---:|:------|
| Ctrl-A | 65 | 1 | SOH (Start of Heading) |
| Ctrl-C | 67 | 3 | ETX (interrupt) |
| Ctrl-D | 68 | 4 | EOT (end of transmission) |
| Ctrl-G | 71 | 7 | BEL (terminal bell) |
| Ctrl-H | 72 | 8 | BS (backspace) |
| Ctrl-I | 73 | 9 | HT (tab) |
| Ctrl-J | 74 | 10 | LF (line feed) |
| Ctrl-M | 77 | 13 | CR (carriage return) |

---

## 3. Control Characters

### The 33 Control Characters

| Dec | Hex | Abbr | Name | Modern Use |
|:---:|:---:|:-----|:-----|:-----------|
| 0 | 0x00 | NUL | Null | C string terminator |
| 7 | 0x07 | BEL | Bell | Terminal bell sound |
| 8 | 0x08 | BS | Backspace | Delete previous char |
| 9 | 0x09 | HT | Horizontal Tab | Indentation |
| 10 | 0x0A | LF | Line Feed | Unix newline |
| 13 | 0x0D | CR | Carriage Return | Windows newline (with LF) |
| 27 | 0x1B | ESC | Escape | Terminal escape sequences |
| 32 | 0x20 | SP | Space | Word separator |
| 127 | 0x7F | DEL | Delete | Delete character |

### Line Ending Conventions

| System | Sequence | Hex |
|:-------|:---------|:----|
| Unix/Linux/macOS | LF | `0x0A` |
| Windows | CR+LF | `0x0D 0x0A` |
| Classic Mac (pre-OS X) | CR | `0x0D` |

### ANSI Escape Sequences

ESC (0x1B) followed by `[` introduces **CSI** (Control Sequence Introducer) sequences:

```
ESC[31m    → red text
ESC[1m     → bold
ESC[0m     → reset
ESC[2J     → clear screen
ESC[H      → cursor home
ESC[10;20H → cursor to row 10, col 20
```

Format: `ESC [ <params> <command>`

---

## 4. Printable Character Layout

### The 95 Printable Characters

```
 !"#$%&'()*+,-./0123456789:;<=>?
@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\]^_
`abcdefghijklmnopqrstuvwxyz{|}~
```

### Alphabetic Ranges

| Range | Characters | Formula |
|:------|:-----------|:--------|
| Uppercase | A-Z | $65 \leq c \leq 90$ |
| Lowercase | a-z | $97 \leq c \leq 122$ |
| Digits | 0-9 | $48 \leq c \leq 57$ |

### Character Classification Bitmasks

Fast classification using bit tricks:

$$\text{isAlpha}(c) = ((c \mathbin{|} \text{0x20}) - \text{'a'}) < 26$$

This works because:
1. `c | 0x20` forces lowercase
2. Subtracting `'a'` maps `a-z` to $0$-$25$
3. Unsigned comparison `< 26` catches the range

$$\text{isDigit}(c) = (c - \text{'0'}) < 10$$

$$\text{isAlnum}(c) = \text{isAlpha}(c) \lor \text{isDigit}(c)$$

---

## 5. Relationship to Other Encodings

### ASCII as a Subset

| Encoding | Relationship to ASCII |
|:---------|:---------------------|
| UTF-8 | ASCII is valid UTF-8 (first 128 code points identical) |
| ISO 8859-1 (Latin-1) | First 128 characters = ASCII, 128-255 = Latin extensions |
| Windows-1252 | Like Latin-1 but 0x80-0x9F have printable characters |
| EBCDIC | Completely different encoding (IBM mainframes) |

### The 8th Bit Problem

ASCII uses 7 bits. In an 8-bit byte, the high bit was used for:
- **Parity checking** (early serial communication)
- **Extended character sets** (code pages, ISO 8859-x)
- **UTF-8 multi-byte sequences** (continuation bytes start with `10`)

### Detecting ASCII

A string is pure ASCII if:

$$\forall b \in \text{bytes}: b \mathbin{\&} \text{0x80} = 0$$

If any byte has the high bit set, it's not ASCII. This check can be vectorized using SIMD:

```c
// Check 16 bytes at once using SSE2
__m128i chunk = _mm_loadu_si128(data);
__m128i high_bits = _mm_and_si128(chunk, _mm_set1_epi8(0x80));
int mask = _mm_movemask_epi8(high_bits);
bool all_ascii = (mask == 0);
```

---

## 6. Historical Design Decisions

### Why These Positions?

| Decision | Reason |
|:---------|:-------|
| Letters at 0x41/0x61 | Bit 5 toggles case — one instruction |
| Digits at 0x30 | Mask with 0x0F gives numeric value |
| Space at 0x20 | Lowest printable — simple `c >= 0x20` check |
| DEL at 0x7F | All bits set (1111111) — could be punched over any character on paper tape |
| NUL at 0x00 | No bits set — blank paper tape |
| Ctrl chars at 0x00-0x1F | Bit 6-5 = 00 — simple masking from letters |

### The Paper Tape Connection

ASCII was designed for **Teletype paper tape**:
- NUL (all holes unpunched) = leader/trailer tape
- DEL (all holes punched) = error correction (overpunch)
- Lower code points = fewer holes = less mechanical wear

---

## 7. ASCII Art and Box Drawing

### Box Drawing (Not ASCII — Extended)

True ASCII has no box-drawing characters. The commonly used ones are Unicode:

| Character | Code Point | Name |
|:----------|:-----------|:-----|
| `─` | U+2500 | Box drawings light horizontal |
| `│` | U+2502 | Box drawings light vertical |
| `┌` | U+250C | Box drawings light down and right |
| `┐` | U+2510 | Box drawings light down and left |
| `└` | U+2514 | Box drawings light up and right |
| `┘` | U+2518 | Box drawings light up and left |

ASCII approximations: `-`, `|`, `+`

---

## 8. Summary of Key Formulas

| Operation | Formula | Example |
|:----------|:--------|:--------|
| To lowercase | $c \mathbin{|} \text{0x20}$ | `'A'|0x20` = `'a'` |
| To uppercase | $c \mathbin{\&} \text{0x5F}$ | `'a'&0x5F` = `'A'` |
| Toggle case | $c \oplus \text{0x20}$ | `'A'^0x20` = `'a'` |
| Char to digit | $c \mathbin{\&} \text{0x0F}$ | `'7'&0x0F` = `7` |
| Letter to ctrl | $c \mathbin{\&} \text{0x1F}$ | `'C'&0x1F` = `3` (ETX) |
| Is printable | $32 \leq c \leq 126$ | |
| Is ASCII | $c \mathbin{\&} \text{0x80} = 0$ | |
| Is alpha | $((c \mathbin{|} \text{0x20}) - 97) < 26$ | |

---

*ASCII's design is a masterclass in bit-level engineering from 1963. Every character position was chosen so that common operations — case conversion, digit extraction, control character generation — could be done with a single bitwise operation. Sixty years later, UTF-8's backward compatibility with ASCII means these bit tricks still work on the most common characters in every text file on earth.*
