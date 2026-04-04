# Unicode (Universal Character Encoding)

> Represent, encode, and manipulate text from all writing systems using code points, UTF encodings, and normalization forms.

## Concepts

### Code Points and Planes

```
# Code point — unique number for each character: U+0041 = 'A'
# Range: U+0000 to U+10FFFF (1,114,112 possible code points)

# Planes:
#   BMP  (Plane 0)   U+0000..U+FFFF     — most common characters
#   SMP  (Plane 1)   U+10000..U+1FFFF   — emoji, historic scripts, math
#   SIP  (Plane 2)   U+20000..U+2FFFF   — CJK ideographs extension B
#   TIP  (Plane 3)   U+30000..U+3FFFF   — CJK extension G/H
#   SSP  (Plane 14)  U+E0000..U+EFFFF   — tags, variation selectors
#   PUA  (Planes 15-16) — private use areas
```

### UTF-8 Encoding

```
# Variable-width encoding: 1 to 4 bytes per code point
# Backward-compatible with ASCII (bytes 0x00-0x7F)

# Byte sequences:
#   U+0000..U+007F     1 byte    0xxxxxxx                        (ASCII)
#   U+0080..U+07FF     2 bytes   110xxxxx 10xxxxxx
#   U+0800..U+FFFF     3 bytes   1110xxxx 10xxxxxx 10xxxxxx
#   U+10000..U+10FFFF  4 bytes   11110xxx 10xxxxxx 10xxxxxx 10xxxxxx

# Example: U+00E9 (e with acute) = C3 A9 in UTF-8
# Example: U+1F600 (grinning face) = F0 9F 98 80 in UTF-8
```

### UTF-16 and UTF-32

```
# UTF-16 — 2 or 4 bytes per code point
#   BMP characters:  single 16-bit code unit
#   Non-BMP:         surrogate pair (high + low surrogate)
#     High surrogate: 0xD800..0xDBFF
#     Low surrogate:  0xDC00..0xDFFF
#   Used internally by: Windows, Java, JavaScript, .NET

# UTF-32 — fixed 4 bytes per code point
#   Simple indexing but wastes space
#   Used internally by: Python (some builds)
```

### BOM (Byte Order Mark)

```
# BOM is U+FEFF placed at the start of a file to indicate encoding/endianness
#   UTF-8:     EF BB BF (optional, often discouraged)
#   UTF-16 BE: FE FF
#   UTF-16 LE: FF FE
#   UTF-32 BE: 00 00 FE FF
#   UTF-32 LE: FF FE 00 00

# Remove UTF-8 BOM
sed -i '1s/^\xEF\xBB\xBF//' file.txt
```

## Normalization Forms

### NFC, NFD, NFKC, NFKD

```
# NFD  (Canonical Decomposition)
#   Decomposes characters: e with acute -> e + combining acute
#   U+00E9 -> U+0065 U+0301

# NFC  (Canonical Decomposition + Composition)
#   Decomposes then recomposes: e + combining acute -> e with acute
#   U+0065 U+0301 -> U+00E9
#   Most common form for interchange (W3C, HTML5)

# NFKD (Compatibility Decomposition)
#   Like NFD but also decomposes compatibility chars: fi ligature -> f + i

# NFKC (Compatibility Decomposition + Composition)
#   Like NFC but includes compatibility decomposition
#   Used for identifiers, search, comparison

# macOS uses NFD for filenames; Linux/Windows typically use NFC
# This causes problems when moving files between systems
```

## Inspection and Conversion

### Command Line Tools

```bash
# Detect file encoding
file -bi document.txt                    # outputs charset info
file --mime-encoding document.txt        # just the encoding

# Hex dump to see raw bytes
hexdump -C document.txt | head -5
xxd document.txt | head -5

# Show Unicode code points
echo -n "cafe" | iconv -f utf-8 -t utf-32be | xxd

# Convert between encodings
iconv -f ISO-8859-1 -t UTF-8 input.txt > output.txt
iconv -f UTF-16 -t UTF-8 input.txt > output.txt
iconv -f UTF-8 -t ASCII//TRANSLIT input.txt > output.txt   # transliterate
iconv -l                                  # list supported encodings

# Normalize filenames (macOS NFD -> NFC)
convmv -f utf8 -t utf8 --nfc -r --notest .
```

### Common Symbols Table

```
# Symbol    Code Point   UTF-8 Bytes      Name
# -         U+2013       E2 80 93         EN DASH
# --        U+2014       E2 80 94         EM DASH
# '         U+2018       E2 80 98         LEFT SINGLE QUOTATION MARK
# '         U+2019       E2 80 99         RIGHT SINGLE QUOTATION MARK
# "         U+201C       E2 80 9C         LEFT DOUBLE QUOTATION MARK
# "         U+201D       E2 80 9D         RIGHT DOUBLE QUOTATION MARK
#           U+00A0       C2 A0            NO-BREAK SPACE
#           U+200B       E2 80 8B         ZERO WIDTH SPACE
#           U+FEFF       EF BB BF         BOM / ZERO WIDTH NO-BREAK SPACE
# ...       U+2026       E2 80 A6         HORIZONTAL ELLIPSIS
```

## Programming Language Handling

### Python

```python
# Strings are Unicode by default (Python 3)
s = "caf\u00e9"                          # 'cafe' with e-acute
len(s)                                    # 4 (code points, not bytes)
s.encode('utf-8')                         # b'caf\xc3\xa9' (5 bytes)

# Normalization
import unicodedata
unicodedata.normalize('NFC', s)           # canonical composition
unicodedata.normalize('NFD', s)           # canonical decomposition
unicodedata.name('\u00e9')                # 'LATIN SMALL LETTER E WITH ACUTE'
unicodedata.category('\u00e9')            # 'Ll' (lowercase letter)

# Detect and handle encoding
with open('file.txt', encoding='utf-8', errors='replace') as f:
    content = f.read()
```

### Go

```go
// Strings are UTF-8 byte sequences; rune = int32 = code point
s := "cafe\u0301"                         // NFD: e + combining acute
fmt.Println(len(s))                       // 6 (bytes)
fmt.Println(utf8.RuneCountInString(s))    // 5 (runes)

// Iterate over runes (not bytes)
for i, r := range s {
    fmt.Printf("index=%d rune=%U char=%c\n", i, r, r)
}

// Normalization (golang.org/x/text/unicode/norm)
import "golang.org/x/text/unicode/norm"
nfc := norm.NFC.String(s)                 // compose
nfd := norm.NFD.String(s)                 // decompose

// Validate UTF-8
utf8.ValidString(s)                       // true if valid UTF-8
```

## Locale and Terminal Settings

### System Locale

```bash
# Check current locale
locale                                    # show all LC_* variables
echo $LANG                               # e.g., en_US.UTF-8

# Set locale
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8

# List available locales
locale -a | grep -i utf

# Generate a locale (Debian/Ubuntu)
sudo locale-gen en_US.UTF-8
sudo update-locale LANG=en_US.UTF-8

# Verify terminal encoding
echo -e '\xC3\xA9'                       # should display: e with acute
printf '\u00e9\n'                         # same thing (bash 4.4+)
```

## Tips

- Always store and transmit text as UTF-8 unless a specific protocol requires otherwise.
- Normalize to NFC before comparing strings or storing in databases.
- String length in code points is not the same as display width (combining marks, CJK wide chars, emoji).
- Beware of homoglyph attacks: U+0410 (Cyrillic A) looks identical to U+0041 (Latin A).
- Use `//TRANSLIT` with iconv to approximate characters that have no equivalent in the target encoding.
- When grepping for non-ASCII, use `grep -P '[\x80-\xFF]'` or `grep '[^[:ascii:]]'`.

## See Also

- ascii, python, go, regex, sed, bash

## References

- [The Unicode Standard](https://www.unicode.org/versions/latest/) -- latest version of the full standard
- [Unicode Code Charts](https://unicode.org/charts/) -- character charts organized by block
- [Unicode Normalization Forms (UAX #15)](https://unicode.org/reports/tr15/) -- NFC, NFD, NFKC, NFKD
- [Unicode Text Segmentation (UAX #29)](https://unicode.org/reports/tr29/) -- grapheme, word, and sentence boundaries
- [Unicode Bidirectional Algorithm (UAX #9)](https://unicode.org/reports/tr9/) -- RTL and mixed-direction text
- [RFC 3629 -- UTF-8](https://www.rfc-editor.org/rfc/rfc3629) -- UTF-8 encoding specification
- [RFC 2781 -- UTF-16](https://www.rfc-editor.org/rfc/rfc2781) -- UTF-16 encoding specification
- [UTF-8 Everywhere](https://utf8everywhere.org/) -- rationale for UTF-8 as default encoding
- [man iconv](https://man7.org/linux/man-pages/man1/iconv.1.html) -- character set conversion utility
- [man locale](https://man7.org/linux/man-pages/man1/locale.1.html) -- locale settings and character maps
- [ICU Documentation](https://unicode-org.github.io/icu/userguide/) -- International Components for Unicode library
- [Unicode CLDR](https://cldr.unicode.org/) -- locale data for software internationalization
