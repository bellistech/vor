# Unicode (Universal Character Encoding)

> Encode every writing system on Earth as numeric codepoints, with a small family of byte encodings (UTF-8, UTF-16, UTF-32) and a set of normalization rules so that different byte sequences can compare equal.

## Setup

```bash
# Unicode itself is a standard, not a tool.
# What you actually install are tools that read/write/inspect Unicode-encoded text.

# 1. iconv — universal encoding converter (always present on Linux/macOS via glibc/libiconv)
iconv --version
iconv -l | head                 # list supported encodings (~1200 of them)

# 2. recode — alternative converter, more aliases
brew install recode             # macOS
sudo apt-get install recode     # Debian/Ubuntu

# 3. hexdump / xxd — view raw bytes
hexdump -C file.txt | head      # canonical hex+ASCII dump
xxd file.txt | head             # similar; xxd -ps for pure hex
xxd -r -p hex.txt > bin.bin     # reverse: hex back to bytes

# 4. file — guess MIME type and charset
file -i  file.txt               # file.txt: text/plain; charset=utf-8
file -bi file.txt               # short form: text/plain; charset=utf-8

# 5. uniname (uniutils) — codepoint inspector
brew install uniutils           # macOS
sudo apt-get install uniutils   # Debian/Ubuntu
echo -n "café" | uniname        # one line per codepoint with name

# 6. uconv — ICU's transliteration tool
brew install icu4c              # macOS
sudo apt-get install icu-devtools libicu-dev   # Debian/Ubuntu
echo "café" | uconv -x any-ascii   # cafe (transliterate)

# 7. locale — current locale settings (controls many runtime behaviours)
locale                          # all LC_* + LANG
locale -a | grep -i utf         # available locales
echo $LANG                      # current setting; want en_US.UTF-8 or C.UTF-8

# 8. chardet / uchardet — encoding guesser when file -i lies
pip install chardet
sudo apt-get install uchardet
uchardet file.txt
chardet file.txt

# 9. ftfy — fix mojibake (Python)
pip install ftfy
python3 -c "import ftfy; print(ftfy.fix_text('CafÃ©'))"   # Café

# Recommended baseline locale (works almost anywhere)
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
```

## Codepoints and Code Units

```bash
# A codepoint is an abstract integer assigned to a character.
# Range:           U+0000 .. U+10FFFF        (1,114,112 possible values)
# Notation:        U+ followed by 4-6 hex digits
# Example:         U+0041 = 'A'   U+00E9 = 'é'   U+1F600 = '😀'

# A code unit is a fixed-width chunk used by a specific encoding.
#   UTF-8   uses  8-bit code units  (one or more units per codepoint)
#   UTF-16  uses 16-bit code units  (one or two units per codepoint)
#   UTF-32  uses 32-bit code units  (always exactly one unit per codepoint)

# A grapheme cluster is what a user perceives as "one character".
# Examples where these counts differ:
#   "é" written as U+00E9         → 1 codepoint, 2 UTF-8 bytes,  1 UTF-16 unit, 1 grapheme
#   "é" written as e + combining  → 2 codepoints, 3 UTF-8 bytes, 2 UTF-16 units, 1 grapheme
#   "😀"                          → 1 codepoint,  4 UTF-8 bytes, 2 UTF-16 units, 1 grapheme
#   "👨‍👩‍👧‍👦" family emoji         → 7 codepoints, 25 UTF-8 bytes, 11 UTF-16 units, 1 grapheme

# Always ask: "length in WHAT?" before writing length-sensitive code.
```

### Length comparison across languages

```bash
# Same string, six measurements, six languages.
# String: "👨‍👩‍👧‍👦" (family: man, woman, girl, boy joined by ZWJ U+200D)

# Bytes (UTF-8):                  25
# UTF-16 code units:              11
# Codepoints (runes/chars):        7
# Grapheme clusters:               1

# Python
python3 -c '
s="👨‍👩‍👧‍👦"
print(len(s.encode()))            # 25
print(len(s))                     # 7
import regex
print(len(regex.findall(r"\X", s)))  # 1
'

# JavaScript
node -e '
const s = "\u{1F468}‍\u{1F469}‍\u{1F467}‍\u{1F466}";
console.log(Buffer.byteLength(s, "utf8"));  // 25
console.log(s.length);                       // 11  (UTF-16 units!)
console.log([...s].length);                  // 7   (codepoints)
console.log([...new Intl.Segmenter("en", {granularity:"grapheme"}).segment(s)].length);  // 1
'

# Go
go run /tmp/x.go <<<'
package main
import ("fmt"; "unicode/utf8")
func main() {
    s := "\xf0\x9f\x91\xa8‍\xf0\x9f\x91\xa9‍\xf0\x9f\x91\xa7‍\xf0\x9f\x91\xa6"
    fmt.Println(len(s))                    // 25 (bytes)
    fmt.Println(utf8.RuneCountInString(s)) // 7  (runes)
}'
```

## Codepoint Ranges

```bash
# Total codespace: U+0000 .. U+10FFFF
# 17 planes of 65,536 codepoints each, but not all assigned.

# Plane  0 — BMP (Basic Multilingual Plane)         U+0000   .. U+FFFF
#   Most everyday scripts: Latin, Greek, Cyrillic, Arabic, Hebrew, CJK common, Hangul, etc.
#   Surrogate range (RESERVED, never standalone):    U+D800   .. U+DFFF
#     High surrogates:                                U+D800   .. U+DBFF
#     Low surrogates:                                 U+DC00   .. U+DFFF
#   Private Use Area (PUA):                          U+E000   .. U+F8FF
#   Specials block (BOM, replacement char, etc.):    U+FFF0   .. U+FFFF
#     U+FFFD = REPLACEMENT CHARACTER (the "")
#     U+FEFF = ZERO WIDTH NO-BREAK SPACE (BOM)
#
# Plane  1 — SMP (Supplementary Multilingual Plane) U+10000  .. U+1FFFF
#   Emoji, math alphanumerics, ancient scripts, music symbols.
# Plane  2 — SIP (Supplementary Ideographic Plane)  U+20000  .. U+2FFFF
#   Rare CJK ideographs (Extension B/C/D/E/F).
# Plane  3 — TIP (Tertiary Ideographic Plane)       U+30000  .. U+3FFFF
#   Even rarer CJK (Extension G, H).
# Planes 4-13 — currently unassigned.
# Plane 14 — SSP (Supplementary Special-purpose)    U+E0000  .. U+EFFFF
#   Tag characters, variation selectors VS17-VS256.
# Plane 15-16 — Supplementary Private Use Areas     U+F0000  .. U+10FFFF
#   Reserved for application-specific characters.

# Codepoints categorically NOT assignable to characters:
#   Surrogates           U+D800   .. U+DFFF   (used only inside UTF-16)
#   Noncharacters        U+FDD0   .. U+FDEF
#                        U+xxFFFE / U+xxFFFF for any plane (e.g. U+FFFE, U+FFFF, U+1FFFE…)

# Why U+10FFFF and not 2^32?
#   UTF-16 surrogate pairs encode at most (0xDBFF-0xD800+1) * (0xDFFF-0xDC00+1) = 1,048,576
#   codepoints above the BMP. Plus 0xFFFF in the BMP minus the 2,048 surrogates.
#   Total addressable = 0x10FFFF + 1 = 1,114,112.
```

## Encoding — UTF-8

```bash
# UTF-8 (RFC 3629) — variable-width 1..4 bytes per codepoint.
# ASCII compatible: U+0000..U+007F encode to a single byte equal to the ASCII value.

# Bit pattern by length:
#   U+0000   .. U+007F      0xxxxxxx                                              1 byte
#   U+0080   .. U+07FF      110xxxxx 10xxxxxx                                     2 bytes
#   U+0800   .. U+FFFF      1110xxxx 10xxxxxx 10xxxxxx                            3 bytes
#   U+10000  .. U+10FFFF    11110xxx 10xxxxxx 10xxxxxx 10xxxxxx                   4 bytes

# Length detection from first byte:
#   0xxxxxxx (0x00-0x7F)  → 1-byte sequence (ASCII)
#   10xxxxxx (0x80-0xBF)  → continuation byte (NOT a start)
#   110xxxxx (0xC0-0xDF)  → start of 2-byte sequence
#   1110xxxx (0xE0-0xEF)  → start of 3-byte sequence
#   11110xxx (0xF0-0xF7)  → start of 4-byte sequence
#   11111xxx (0xF8-0xFF)  → INVALID in UTF-8

# Self-synchronising:
#   Continuation bytes always start with 10. From any random byte you can find
#   the start of the current codepoint by scanning back at most 3 bytes.

# Examples:
echo -n "A"     | xxd       # 41                    U+0041 = 1 byte
echo -n "é"     | xxd       # c3 a9                 U+00E9 = 2 bytes
echo -n "€"     | xxd       # e2 82 ac              U+20AC = 3 bytes
echo -n "😀"    | xxd       # f0 9f 98 80          U+1F600 = 4 bytes

# Invalid byte sequences you will see in real life:
#   0xC0, 0xC1                         — overlong encoding of ASCII (forbidden)
#   0xF5..0xFF                         — would encode beyond U+10FFFF
#   0xED 0xA0..0xBF ...                — encoding of UTF-16 surrogate (forbidden)
#   start byte without enough continuation bytes (truncation)
#   continuation byte without start byte (corruption)

# Validate a file is UTF-8:
iconv -f UTF-8 -t UTF-8 file.txt > /dev/null && echo OK || echo BAD
python3 -c "open('file.txt',encoding='utf-8').read()" && echo OK
```

## Encoding — UTF-16

```bash
# UTF-16 (RFC 2781) — variable-width 2 or 4 bytes per codepoint.

# BMP codepoint U+0000..U+FFFF (excluding surrogates) → single 16-bit unit equal to the codepoint.
# Codepoint U+10000..U+10FFFF → two 16-bit units (a "surrogate pair"):
#
#   c' = c - 0x10000              (0..0xFFFFF, 20 bits)
#   high = 0xD800 | (c' >> 10)    (top 10 bits, range 0xD800..0xDBFF)
#   low  = 0xDC00 | (c' & 0x3FF)  (bottom 10 bits, range 0xDC00..0xDFFF)

# Example: U+1F600 (😀)
#   c' = 0x0F600
#   high = 0xD800 | 0x3D = 0xD83D
#   low  = 0xDC00 | 0x200 = 0xDE00
#   UTF-16 BE bytes: D8 3D DE 00
#   UTF-16 LE bytes: 3D D8 00 DE

# Endianness:
#   UTF-16BE — big-endian, no BOM required.
#   UTF-16LE — little-endian, no BOM required.
#   UTF-16   — endianness signalled by BOM (FE FF = BE, FF FE = LE) or platform default.

# Used internally by Java (char[] is UTF-16), JavaScript, .NET, Windows API (W functions),
# Qt QString, ICU UChar.

# The BMP gotcha: "𝓗".length === 2 in JavaScript because that's two UTF-16 units, even
# though it's one codepoint. Same for almost all emoji.

echo -n "😀" | iconv -f UTF-8 -t UTF-16BE | xxd   # d8 3d de 00
echo -n "😀" | iconv -f UTF-8 -t UTF-16LE | xxd   # 3d d8 00 de
echo -n "😀" | iconv -f UTF-8 -t UTF-16   | xxd   # fe ff d8 3d de 00 (with BOM)
```

## Encoding — UTF-32

```bash
# UTF-32 — fixed 4 bytes per codepoint, value equals the codepoint number.
# Sometimes called UCS-4.
# Range that is actually used: 0x00000000 .. 0x0010FFFF (top 11 bits always zero).

# Endianness: UTF-32BE, UTF-32LE, or UTF-32 with BOM (00 00 FE FF / FF FE 00 00).

# Pros:  O(1) random access by codepoint index.
# Cons:  4x the size of UTF-8 for ASCII text. Almost never used on the wire or on disk.

# Where you do see it:
#   Linux  wchar_t           is 32-bit  → wide-character strings are effectively UTF-32.
#   Python (Py 3.3+ flexible string repr) may store as 32-bit per character internally.
#   ICU UChar32 / Go rune    are 32-bit Unicode scalar types in memory.

echo -n "A"   | iconv -f UTF-8 -t UTF-32BE | xxd   # 00 00 00 41
echo -n "😀"  | iconv -f UTF-8 -t UTF-32BE | xxd   # 00 01 f6 00
echo -n "😀"  | iconv -f UTF-8 -t UTF-32LE | xxd   # 00 f6 01 00
```

## Byte Order Mark (BOM)

```bash
# The BOM is codepoint U+FEFF "ZERO WIDTH NO-BREAK SPACE" placed at the start of a file
# to signal byte order and (loosely) encoding.

# Encoded forms:
#   UTF-8     EF BB BF        (signature only — UTF-8 has no byte order!)
#   UTF-16BE  FE FF
#   UTF-16LE  FF FE
#   UTF-32BE  00 00 FE FF
#   UTF-32LE  FF FE 00 00     (also matches UTF-16LE prefix → ambiguous!)

# Add UTF-8 BOM:
printf '\xEF\xBB\xBF' > with-bom.txt
cat input.txt          >> with-bom.txt

# Remove UTF-8 BOM:
sed -i '1s/^\xEF\xBB\xBF//' file.txt                          # GNU sed
LANG=C sed -i '' '1s/^\xEF\xBB\xBF//' file.txt                # BSD/macOS sed
perl -i -pe 'tr/\x{feff}//d' file.txt                         # any platform

# Detect BOM:
head -c3 file.txt | xxd
# ef bb bf  → UTF-8 BOM
# fe ff     → UTF-16BE BOM
# ff fe 00  → UTF-32LE BOM (and continuing 00 confirms vs UTF-16LE)

# Why UTF-8 BOM is controversial:
#   - UTF-8 has fixed byte order so BOM is redundant.
#   - Many Unix tools choke: bash scripts with BOM fail "command not found" on '#!/...'.
#   - PHP outputs the BOM as page content before headers — header-already-sent error.
#   - JSON parsers per RFC 8259 must NOT emit BOM and MAY reject one.
#   - Windows Notepad historically wrote them; modern Notepad lets you choose.
# General advice: don't write a UTF-8 BOM unless required by a Microsoft tool that demands it.
```

## Detecting Encoding

```bash
# 1. Trust the metadata if you have it (HTTP Content-Type, XML declaration, etc.).
# 2. Otherwise: assume UTF-8 first (the modern default).
# 3. If that fails, escalate to chardet/uchardet/file -i, then human inspection.

file -i  file.txt           # file.txt: text/plain; charset=utf-8
file -bi file.txt           # text/plain; charset=utf-8
file -b  file.txt           # UTF-8 Unicode text, with no line terminators

# uchardet — Mozilla universal charset detector
uchardet file.txt           # WINDOWS-1252

# chardet (Python)
chardet file.txt            # file.txt: utf-8 with confidence 0.99

# enca — strong on East European encodings
enca -L none file.txt

# Roll-your-own check: is every byte valid UTF-8?
iconv -f UTF-8 -t UTF-8 file.txt > /dev/null

# Symptoms of wrong encoding:
#   "café"        as UTF-8 read as Latin-1     → "cafÃ©"      (mojibake)
#   "café"        as Latin-1 read as UTF-8     → byte 0xE9 followed by space → 'é' invalid
#   "Привет"      as UTF-8 read as CP1252      → "Привет"
#   text starts with "" (U+FFFD)               → previously decoded with errors='replace'
```

## Inspecting Bytes and Codepoints

```bash
# Raw bytes:
hexdump -C file.txt | head
xxd        file.txt | head
xxd -ps    file.txt          # pure hex, no offset/ASCII column
od -An -c  file.txt | head   # bytes as C-style chars

# Codepoints with names (uniname from uniutils):
echo -n "café" | uniname
#  character      byte  UTF-32   encoded as glyph  name
#         0          0  00000063 63                LATIN SMALL LETTER C
#         1          1  00000061 61                LATIN SMALL LETTER A
#         2          2  00000066 66                LATIN SMALL LETTER F
#         3          3  000000E9 C3 A9             LATIN SMALL LETTER E WITH ACUTE

# Codepoints inline:
python3 -c 'import sys; [print(hex(ord(c)), c) for c in sys.argv[1]]' "café"
node     -e   'for (const c of process.argv[2]) console.log(c.codePointAt(0).toString(16), c)' "café"
perl     -CS -e 'for (split //, $ARGV[0]) { printf "%X %s\n", ord, $_ }' "café"

# Search for a literal codepoint:
grep -P "\x{2014}" file.txt              # find every EM DASH
LC_ALL=C grep -P "[\x80-\xff]" file.txt  # find every non-ASCII byte
rg --pcre2 "\p{Cc}" file.txt             # find every control character

# Per-language codepoint dump:
python3 -c 's="café"; print([hex(ord(c)) for c in s])'
node     -e 'console.log([..."café"].map(c=>c.codePointAt(0).toString(16)))'
go run -  <<<'package main; import "fmt"; func main(){ for _, r := range "café" { fmt.Printf("%U\n", r) } }'
ruby     -e 'puts "café".each_char.map{|c| "U+%04X" % c.ord}'
```

## Codepoint to Character

```bash
# Bash printf supports \xHH (raw byte), \uHHHH (BMP codepoint), \UHHHHHHHH (any codepoint):
printf '\xc3\xa9\n'            # é   (raw UTF-8 bytes)
printf 'é\n'              # é   (codepoint, bash 4.2+)
printf '\U0001F600\n'          # 😀
echo -e 'é'               # é   (echo -e in bash)

# zsh: $'é' inside a $'…' string.
echo $'é'                 # é   (zsh and bash)

# Python
python3 -c 'print(chr(0xE9))'                  # é
python3 -c 'print(chr(0x1F600))'              # 😀
python3 -c 'print("\N{LATIN SMALL LETTER E WITH ACUTE}")'  # é   (by name!)

# JavaScript / Node
node -e 'console.log(String.fromCodePoint(0xE9))'      # é
node -e 'console.log(String.fromCodePoint(0x1F600))'   # 😀
node -e 'console.log("é")'                         # é
node -e 'console.log("\u{1F600}")'                      # 😀  (ES6 brace form)

# Go
go run - <<<'package main; import "fmt"; func main(){ fmt.Println(string(rune(0x1F600))) }'

# Rust
echo 'fn main(){println!("{}", char::from_u32(0x1F600).unwrap());}' > /tmp/r.rs && rustc /tmp/r.rs -o /tmp/r && /tmp/r

# Java
jshell -<<<'System.out.println(new String(Character.toChars(0x1F600)));'

# Ruby
ruby -e 'puts 0xE9.chr("UTF-8")'                # é
ruby -e 'puts "\u{1F600}"'                       # 😀

# C (C11 with <uchar.h>)
#   char32_t c = U'😀';   // 0x0001F600
```

## Character to Codepoint

```bash
# Python
python3 -c 'print(hex(ord("é")))'                           # 0xe9
python3 -c 'print(hex(ord("😀")))'                          # 0x1f600

# JavaScript
node -e 'console.log("é".codePointAt(0).toString(16))'      # e9
node -e 'console.log("😀".codePointAt(0).toString(16))'    # 1f600
node -e 'console.log("😀".charCodeAt(0).toString(16))'     # d83d  (just the high surrogate!)

# Go — range yields runes, NOT bytes
cat <<'EOF' > /tmp/cp.go
package main
import "fmt"
func main() {
    for i, r := range "café" {
        fmt.Printf("byte=%d  U+%04X  %c\n", i, r, r)
    }
}
EOF
go run /tmp/cp.go

# Rust — .chars() iterates Unicode scalar values
cat <<'EOF' > /tmp/cp.rs
fn main() {
    for c in "café".chars() { println!("U+{:04X} {}", c as u32, c); }
}
EOF
rustc /tmp/cp.rs -o /tmp/cp && /tmp/cp

# Java
jshell <<<'"😀".codePointAt(0)'                              # 128512 = 0x1F600

# Bash (single ASCII or first byte only)
printf '%d\n' "'A"                                          # 65
printf '%x\n' "'A"                                          # 41
# For real codepoint extraction, drop into Python or od:
echo -n "😀" | iconv -t UTF-32BE | od -An -tx4 -v          # 0001f600
```

## Normalization Forms

```bash
# UAX #15 defines four normalization forms:
#
#   NFD   Canonical Decomposition
#         Splits precomposed characters into base + combining marks.
#         "é" U+00E9 → U+0065 U+0301 (e + combining acute)
#
#   NFC   Canonical Composition (decompose, then recompose)
#         Most compact form. Default for the web (HTML5/W3C recommend NFC).
#         U+0065 U+0301 → U+00E9
#
#   NFKD  Compatibility Decomposition
#         NFD plus replacement of "compatibility characters" with their canonical equivalents.
#         "ﬁ"   (U+FB01 LATIN SMALL LIGATURE FI) → "fi"
#         "①"  (U+2460 CIRCLED DIGIT ONE)        → "1"
#         "Ⅻ"  (U+216B ROMAN NUMERAL TWELVE)     → "XII"
#         Loses semantic distinctions — use only for search/identifier folding.
#
#   NFKC  Compatibility Composition
#         NFKD then NFC-style recomposition. Used by IDNA, Python identifiers (PEP 3131).

# Why this matters:
#   Filesystems, APIs, and languages disagree about which form they use:
#     macOS HFS+        stored filenames as NFD
#     APFS              preserves whatever form you give it (mostly)
#     Linux ext4        stores raw bytes, no normalisation
#     NTFS              UTF-16, no normalisation
#     Web forms         most browsers send NFC, but not all
#   Two strings can look identical and pass `==` only after normalising both.

# Always normalise BEFORE comparing user input or storing in a database.
```

## Normalization Examples

```bash
# Demonstrate that "café" can be either 4 codepoints (NFC) or 5 codepoints (NFD).

python3 - <<'EOF'
import unicodedata as u
nfc = "café"          # 4 codepoints: c a f é
nfd = "café"          # 5 codepoints: c a f e ́
print(nfc == nfd)           # False  — bytes differ
print(len(nfc), len(nfd))   # 4 5
print(u.normalize("NFC", nfc) == u.normalize("NFC", nfd))  # True
print(u.normalize("NFD", nfc) == u.normalize("NFD", nfd))  # True
# Inspect codepoints:
for s in (nfc, nfd):
    print([f"U+{ord(c):04X}" for c in s])
EOF

# Quick CLI normalization:
echo "cafe$(printf '́')" | uconv -x any-NFC | xxd       # final bytes c3 a9
echo "café"                    | uconv -x any-NFD | xxd       # final bytes 65 cc 81

# Compatibility decomposition kills information:
python3 -c 'import unicodedata; print(unicodedata.normalize("NFKD", "ﬁle"))'   # "file"
python3 -c 'import unicodedata; print(unicodedata.normalize("NFKD", "H₂O"))'   # "H2O"
python3 -c 'import unicodedata; print(unicodedata.normalize("NFKD", "①②③"))' # "123"

# Same precaution for filenames:
ls -la                     # see exact bytes per filename
ls | iconv -f UTF-8-MAC -t UTF-8     # macOS NFD → standard NFC
```

## Combining Characters

```bash
# Combining marks are codepoints that visually attach to the preceding base codepoint.
# Most live in:
#   U+0300 .. U+036F   Combining Diacritical Marks      (Latin/Greek/Cyrillic accents)
#   U+1AB0 .. U+1AFF   Combining Diacritical Marks Extended
#   U+1DC0 .. U+1DFF   Combining Diacritical Marks Supplement
#   U+20D0 .. U+20FF   Combining Diacritical Marks for Symbols
#   U+FE20 .. U+FE2F   Combining Half Marks

# They have General_Category Mn (Mark, Nonspacing) or Mc (Mark, Spacing combining).
# A user-perceived character can have arbitrarily many combining marks stacked on it.

# Build "Z̵̧͚̗͍̩̱̦̳̮̬̜͚̘̭̝̲̭̑̇̇̌̆̈́̃̃̾̏̾̌̆͊̕͝͝͝" (Zalgo text):
python3 -c '
import random
out = "Z" + "".join(chr(random.randint(0x300, 0x36F)) for _ in range(20))
print(out, "len(codepoints)=", len(out))
'

# Strip combining marks ("remove diacritics") to fold "café" to "cafe":
python3 -c '
import unicodedata as u
s = "Café Über naïve résumé"
print("".join(c for c in u.normalize("NFD", s) if u.category(c) != "Mn"))
# Cafe Uber naive resume
'

# In Go:
cat <<'EOF' > /tmp/d.go
package main
import (
    "fmt"
    "unicode"
    "golang.org/x/text/runes"
    "golang.org/x/text/transform"
    "golang.org/x/text/unicode/norm"
)
func main() {
    t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
    out, _, _ := transform.String(t, "café")
    fmt.Println(out)   // cafe
}
EOF
```

## Grapheme Clusters

```bash
# A "user-perceived character" is a grapheme cluster, defined by UAX #29.
# A single grapheme can span many codepoints:
#   - Base + N combining marks
#   - Hangul syllable made of L + V + T jamos
#   - Emoji + variation selector + skin tone modifier
#   - Multiple emoji joined by ZWJ (U+200D) — family, profession, gender variants
#   - Regional Indicator pair forming a flag

# Most languages do NOT segment by grapheme out of the box.
# You need:

# Python:
pip install regex
python3 -c 'import regex; print(regex.findall(r"\X", "👨‍👩‍👧‍👦café"))'
# ['👨‍👩‍👧‍👦', 'c', 'a', 'f', 'é']

# JavaScript (Node 16+, modern browsers):
node -e '
const seg = new Intl.Segmenter("en", {granularity:"grapheme"});
console.log([...seg.segment("👨‍👩‍👧‍👦café")].map(s=>s.segment));
'
# [ '👨‍👩‍👧‍👦', 'c', 'a', 'f', 'é' ]

# Rust:
#   cargo add unicode-segmentation
#   use unicode_segmentation::UnicodeSegmentation;
#   for g in "👨‍👩‍👧‍👦café".graphemes(true) { println!("{}", g); }

# Go (community library; std lib only does codepoints):
#   go get github.com/rivo/uniseg
#   for s := uniseg.NewGraphemes("…"); s.Next(); { fmt.Println(s.Str()) }

# Why this is critical:
#   "👨‍👩‍👧‍👦"[0:1] in JS    yields the high surrogate of 👨 alone — invalid UTF-16.
#   substr(0, 5) of "café" in C may chop "é" mid-byte → invalid UTF-8.
#   Truncating user names by code-unit count produces broken text and bug reports.

# Always segment by graphemes before truncating, reversing, or counting "characters" for users.
```

## Properties — General Category

```bash
# Every codepoint has exactly one General_Category. Two letters: top-level + sub.
#
#   L  Letter
#     Lu  Uppercase Letter         A B C
#     Ll  Lowercase Letter         a b c
#     Lt  Titlecase Letter         ǅ ǈ ǋ (used by some Slavic digraphs)
#     Lm  Modifier Letter          ʰ ʲ
#     Lo  Other Letter             ا 中 (no case distinction)
#
#   M  Mark
#     Mn  Nonspacing Mark          ́ ̈ ̃   (combining accents)
#     Mc  Spacing Mark             ा ी (Devanagari vowel signs)
#     Me  Enclosing Mark           ⃝ ⃞
#
#   N  Number
#     Nd  Decimal_Digit            0 ١ ๑
#     Nl  Letter_Number            Ⅰ Ⅱ Ⅲ
#     No  Other_Number             ¼ ½ ²
#
#   P  Punctuation
#     Pc  Connector                _ ‿
#     Pd  Dash                     - – —
#     Ps  Open                     ( [ {
#     Pe  Close                    ) ] }
#     Pi  Initial Quote            « " '
#     Pf  Final Quote              » " '
#     Po  Other                    . , ; : ! ?
#
#   S  Symbol
#     Sm  Math                     + < = ≠
#     Sc  Currency                 $ € ¥ ₹
#     Sk  Modifier                 ^ `
#     So  Other                    © ™ ☃ 😀
#
#   Z  Separator
#     Zs  Space Separator          U+0020 U+00A0 U+2003
#     Zl  Line Separator           U+2028
#     Zp  Paragraph Separator      U+2029
#
#   C  Other
#     Cc  Control                  \t \n \r U+0000..U+001F U+007F..U+009F
#     Cf  Format                   U+200B ZWSP, U+200D ZWJ, U+FEFF BOM
#     Cs  Surrogate                U+D800..U+DFFF (never standalone!)
#     Co  Private Use              U+E000..U+F8FF and Plane 15-16
#     Cn  Unassigned               (reserved or noncharacter)

# Look up a codepoint's category:
python3 -c 'import unicodedata as u; print(u.category("é"))'    # Ll
python3 -c 'import unicodedata as u; print(u.category("😀"))'  # So
python3 -c 'import unicodedata as u; print(u.category("​"))'  # Cf

# Match by category in regex (PCRE/Python re/Rust regex/Java):
grep -P "\p{Lu}" file.txt           # any uppercase letter
grep -P "\p{Sc}" file.txt           # any currency symbol
grep -P "\p{Cc}" file.txt           # any control character (find rogue \r)
grep -P "[\p{L}\p{Nd}]" file.txt    # letters or digits, in any script
```

## Properties — Script

```bash
# Each codepoint has a Script property assigning it to a writing system.
# Common values:
#   Latin     (Latn)   English, French, German, Vietnamese, Hausa, etc.
#   Greek     (Grek)
#   Cyrillic  (Cyrl)   Russian, Ukrainian, Bulgarian, Serbian, Mongolian, etc.
#   Arabic    (Arab)   Arabic, Persian, Urdu, Pashto
#   Hebrew    (Hebr)
#   Devanagari(Deva)   Hindi, Sanskrit, Marathi
#   Bengali   (Beng)
#   Tamil     (Taml)
#   Thai      (Thai)
#   Han       (Hani)   Chinese, Japanese kanji, Korean hanja
#   Hiragana  (Hira)   Japanese
#   Katakana  (Kana)   Japanese
#   Hangul    (Hang)   Korean
#   Ethiopic  (Ethi)   Amharic
#   Common    (Zyyy)   shared across scripts: digits, punctuation, ASCII letters
#   Inherited (Zinh)   combining marks that take their script from the base char

# Script vs language:
#   "中" is Script=Han whether used in Mandarin, Cantonese, Japanese, or Korean.
#   "a" is Script=Latin in English, Vietnamese, or Welsh.
# Script tells you nothing about pronunciation or orthographic rules. Use locale/lang for that.

# Match by script in regex:
grep -P "\p{Script=Han}"      file.txt   # Chinese characters
grep -P "\p{Script=Cyrillic}" file.txt
grep -P "\p{sc=Arabic}"       file.txt   # short alias

# Detect mixed-script text (homoglyph attack vector):
python3 -c '
import unicodedata as u
def script(c):
    return u.name(c, "").split()[0] if c.isalpha() else None
s = "аpple"        # first char is Cyrillic а (U+0430), not Latin a (U+0061)!
print({script(c) for c in s if c.isalpha()})
'

# Mixed scripts in a single identifier are a major phishing vector
# ("раypal.com" with Cyrillic а). Browsers and registrars apply policies (UTS #39).
```

## Case Folding

```bash
# Three mappings — they are NOT the same:
#   uppercase     "ß"     → "SS"        (string can grow!)
#   lowercase     "İ"     → "i̇" (i + combining dot above)
#   case fold     "ß"     → "ss"        intended for caseless comparison

# Locale-dependent gotchas:
#   Turkish / Azerbaijani:  "I".lower() == "ı"  (dotless), not "i"
#                           "i".upper() == "İ"  (dotted),  not "I"
#   German: ß ↔ ẞ (since 2017). Capital ẞ is rare and often unsupported.

python3 -c 'print("ß".upper())'           # SS
python3 -c 'print("ß".lower())'           # ß
python3 -c 'print("ß".casefold())'        # ss

python3 -c 'print("STRASSE".lower() == "straße".lower())'        # False
python3 -c 'print("STRASSE".casefold() == "straße".casefold())'  # True

# Java
jshell <<<'"ß".toUpperCase()'                            # "SS"
jshell <<<'"İ".toLowerCase()'                            # "i̇"  (default)
jshell <<<'"İ".toLowerCase(java.util.Locale.forLanguageTag("tr"))'  # "i"

# Ruby
ruby -e 'puts "ß".upcase'                  # SS
ruby -e 'puts "ß".downcase'                # ß
ruby -e 'puts "ß".unicode_normalize(:nfkc).downcase'    # depends on Ruby version

# Rule of thumb for caseless equality:
#   x.casefold() == y.casefold()  in Python/Java/etc.
#   Combine with normalisation:   nfkc_casefold(x) == nfkc_casefold(y)
```

## Encodings — Legacy

```bash
# Pre-Unicode encodings you still meet in the wild:
#
#   ASCII             7-bit, U+0000..U+007F. Always a valid UTF-8 prefix.
#   ISO-8859-1        Latin-1, 8-bit, Western European. Maps 1-1 to U+0000..U+00FF.
#   ISO-8859-2..16    Other Latin/Greek/Cyrillic/Hebrew/Arabic variants.
#   Windows-1252      Microsoft superset of Latin-1; bytes 0x80-0x9F are extra glyphs
#                     (smart quotes, em dash, €). Often mis-labelled as Latin-1.
#   Windows-1250..58  Microsoft variants for Eastern European, Greek, Turkish, etc.
#   KOI8-R / KOI8-U   Cyrillic encodings predominant in Russia/Ukraine pre-2010.
#   GB2312, GBK, GB18030  Chinese (mainland). GB18030 is a full Unicode transcoding.
#   Big5, Big5-HKSCS   Chinese (Taiwan, Hong Kong).
#   Shift-JIS, EUC-JP, ISO-2022-JP   Japanese. Shift-JIS dominates Windows; ISO-2022-JP email.
#   EUC-KR, ISO-2022-KR, UHC          Korean.
#   MacRoman, MacCentralEurope, ...  legacy classic Mac OS.
#   IBM-437, IBM-850, ...            DOS code pages.

# Convert legacy → UTF-8 (always a one-way trip; do it once on ingest):
iconv -f WINDOWS-1252 -t UTF-8 input.txt > output.txt
iconv -f SHIFT_JIS    -t UTF-8 input.txt > output.txt
iconv -f GB18030      -t UTF-8 input.txt > output.txt

# Lossy ASCII (last resort, drops non-Latin scripts):
iconv -f UTF-8 -t ASCII//TRANSLIT input.txt    # café → cafe
iconv -f UTF-8 -t ASCII//IGNORE   input.txt    # café → caf

# Why migrate everything to UTF-8:
#   - One encoding for every script, no escape sequences.
#   - ASCII-clean for every existing parser and protocol.
#   - Self-synchronising and easy to validate.
#   - Mandatory for JSON, XML, HTTP/1.1 (default), HTML5, modern terminals.
```

## iconv — Conversion CLI

```bash
# Basic syntax
iconv -f FROM -t TO file > out

# List supported encodings (very long)
iconv -l
iconv --list

# Common conversions
iconv -f LATIN1   -t UTF-8 in.txt > out.txt
iconv -f UTF-16   -t UTF-8 in.txt > out.txt
iconv -f UTF-8    -t UTF-16BE in.txt > out.txt
iconv -f UTF-8    -t ASCII//TRANSLIT in.txt > out.txt    # transliterate (lossy)
iconv -f UTF-8    -t ASCII//IGNORE   in.txt > out.txt    # silently drop unconvertible
iconv -f WINDOWS-1252 -t UTF-8 in.txt > out.txt          # most "Latin-1" files are really 1252

# In-place conversion (GNU iconv has no -i; use a temp file)
iconv -f WINDOWS-1252 -t UTF-8 file > file.tmp && mv file.tmp file

# Pipe usage
echo "café" | iconv -f UTF-8 -t LATIN1 | xxd     # 63 61 66 e9
echo "café" | iconv -f UTF-8 -t UTF-16LE | xxd

# Suppress conversion errors
iconv -c ...       # silently drop chars that can't be represented in target

# Alternative: recode (richer alias set, same idea)
recode UTF-8..LATIN1 file.txt
recode WINDOWS-1252..UTF-8 -d -i file.txt   # in-place, with diff

# Nice trick: BOM-strip while converting
iconv -f UTF-8 -t UTF-8 in.txt | sed '1s/^\xEF\xBB\xBF//'
```

## Unicode in Bash

```bash
# Bash 4+ handles UTF-8 reasonably IF the locale is set correctly.
# What "$LANG" affects:
#   - case statements with [[ "$x" == [a-z]* ]]
#   - sort, tr, sed character classes
#   - ${#var} length (codepoints in C.UTF-8/en_US.UTF-8, bytes in C/POSIX)
#   - readline command-line editing for multibyte input

export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8

s="café"
echo ${#s}                  # 4 in UTF-8 locale, 5 in C locale!

# Iterate per-character (substring expansion is by codepoint in UTF-8 locale):
for ((i=0; i<${#s}; i++)); do printf '[%s]' "${s:i:1}"; done
echo

# Generate a codepoint:
printf 'é\n'           # é (bash 4.2+)
printf '\U0001F600\n'       # 😀

# Get the codepoint of a literal:
LC_ALL=en_US.UTF-8 printf '%d\n' "'é"   # 233 (= 0xE9)

# Byte length vs char length:
echo -n "café" | wc -c                  # 5 (bytes)
echo -n "café" | LANG=en_US.UTF-8 wc -m # 4 (chars)

# tr will silently mangle UTF-8 in many locales. For non-ASCII transliteration,
# prefer iconv//TRANSLIT or sed with proper locale, or drop into Python.

# Read a file safely as UTF-8:
while IFS= read -r line; do
    printf '%s\n' "$line"
done < input.txt
```

## Unicode in Python

```python
# Python 3 fundamentals:
#   str        — sequence of Unicode codepoints (NOT bytes)
#   bytes      — sequence of 0..255 octets
# Default source encoding is UTF-8 (PEP 3120).

s = "café"
b = s.encode("utf-8")            # b'caf\xc3\xa9'
b.decode("utf-8")                # "café"
s.encode("latin-1")              # b'caf\xe9'
s.encode("ascii", errors="replace")    # b'caf?'
s.encode("ascii", errors="ignore")     # b'caf'
s.encode("ascii", errors="xmlcharrefreplace")    # b'caf&#233;'
s.encode("ascii", errors="backslashreplace")     # b'caf\\xe9'

# Length always counts codepoints:
len("café")          # 4
len("👨‍👩‍👧‍👦")    # 7  (codepoints — to count graphemes use the regex package)

# Codepoint → char and back
chr(0x1F600)         # '😀'
ord("😀")           # 128512
"\N{GREEK SMALL LETTER ALPHA}"   # 'α'  (lookup by name)

# Inspect properties
import unicodedata as u
u.name("é")          # 'LATIN SMALL LETTER E WITH ACUTE'
u.category("é")      # 'Ll'
u.bidirectional("ا") # 'AL'  (Arabic letter, right-to-left)
u.decomposition("é") # '0065 0301'
u.east_asian_width("中")  # 'W' (Wide)

# Normalise
u.normalize("NFC", x)
u.normalize("NFD", x)
u.normalize("NFKC", x)   # for identifiers, search

# Open files with explicit encoding (always!)
open("data.txt", encoding="utf-8")                   # default errors='strict'
open("data.txt", encoding="utf-8", errors="replace") # never raise; use 
open("data.txt", encoding="utf-8", errors="ignore")  # silently drop bad bytes
open("data.txt", encoding="utf-8", newline="")       # for csv module (don't translate \n)

# Common error
#   UnicodeDecodeError: 'utf-8' codec can't decode byte 0x9c in position 4: invalid start byte
# Fix: pass the actual encoding (often cp1252) or open as bytes and inspect.

# Grapheme-aware operations
import regex
regex.findall(r"\X", "👨‍👩‍👧‍👦café")   # ['👨‍👩‍👧‍👦', 'c', 'a', 'f', 'é']

# Casefolding for comparison
"Straße".casefold() == "STRASSE".casefold()   # True
```

## Unicode in JavaScript

```javascript
// JS strings are sequences of UTF-16 code units. This is the source of most surprises.

"𝓗".length;                 // 2  (UTF-16 surrogate pair, ONE codepoint!)
[..."𝓗"].length;            // 1  (spread iterates codepoints)
"𝓗".codePointAt(0);         // 119943 = 0x1D4D7
"𝓗".codePointAt(0).toString(16);  // "1d4d7"
"𝓗".charCodeAt(0).toString(16);   // "d835"  ← high surrogate alone

// Iterating by codepoint (not by code unit):
for (const c of "café") { console.log(c, c.codePointAt(0).toString(16)); }
// c 63
// a 61
// f 66
// é e9

// Building strings from codepoints
String.fromCodePoint(0x1F600);   // "😀"
String.fromCharCode(0xD83D, 0xDE00);  // "😀"  (manual surrogates — error-prone)
"é";                         // "é"  (BMP only, max 4 hex digits)
"\u{1F600}";                      // "😀"  (ES6 brace form, any codepoint)

// Regex Unicode awareness:
"𝓗".match(/./);                 // matches one UTF-16 unit (broken)
"𝓗".match(/./u);                // matches the codepoint with the u flag
/^\p{L}+$/u.test("café");        // true (\p{L} requires the u flag)

// Normalisation (built-in since ES6)
"é".normalize("NFC");
"é".normalize("NFD");
"é".normalize("NFKC");
"é".normalize("NFKD");
"é" === "é";                    // false if one is NFC and other NFD!
"é".normalize() === "é".normalize();   // true

// Grapheme segmentation (Intl.Segmenter, baseline modern Node and browsers)
const seg = new Intl.Segmenter("en", { granularity: "grapheme" });
[...seg.segment("👨‍👩‍👧‍👦café")].map(s => s.segment);

// Encoding bytes
new TextEncoder().encode("café");     // Uint8Array(5) [99,97,102,195,169]
new TextDecoder("utf-8").decode(buf); // "café"
```

## Unicode in Go

```go
// Go strings are immutable byte sequences. Source files are UTF-8 by spec.
// rune is an alias for int32, holding a Unicode codepoint.

s := "café"
len(s)                                  // 5 (bytes — é is 2 bytes in UTF-8)
utf8.RuneCountInString(s)              // 4 (codepoints)

// range yields (byte index, rune)
for i, r := range s {
    fmt.Printf("byte=%d  U+%04X  %c\n", i, r, r)
}

// Get the n-th rune (NOT s[n])
runes := []rune(s)
runes[3]                                // 'é'

// Decode/encode raw bytes
import "unicode/utf8"
r, size := utf8.DecodeRuneInString(s)   // first rune + its byte length
buf := make([]byte, 4)
n := utf8.EncodeRune(buf, '😀')         // n == 4

// Validation
utf8.ValidString(s)                     // true if all bytes form valid UTF-8
utf8.RuneError                          // 0xFFFD — replacement char

// Properties
import "unicode"
unicode.IsLetter('é')                  // true
unicode.IsDigit('1')                    // true
unicode.IsSpace(' ')                    // true
unicode.IsUpper('A')                    // true

// Normalization (sub-repo)
import "golang.org/x/text/unicode/norm"
norm.NFC.String("café")          // "café"
norm.NFD.String("café")                // "café"

// Casefolding
strings.EqualFold("Straße", "strasse")  // true (Go is unicode-aware)

// Grapheme clusters require a third-party lib (e.g., github.com/rivo/uniseg).
```

## Unicode in Rust

```rust
// &str and String are guaranteed to hold valid UTF-8.
// char is a 32-bit Unicode scalar value (codepoint excluding surrogates).

let s = "café";
s.len();                       // 5  (bytes)
s.chars().count();             // 4  (codepoints; O(n) walk)
s.chars().next().unwrap();     // 'c' as char (4 bytes wide internally)

// Indexing s[0..1] returns &str only if the slice ends on a UTF-8 boundary;
// otherwise it panics: "byte index 1 is not a char boundary".
&s[0..1];                       // "c"
&s[0..2];                       // "ca"
//&s[0..3];                     // "caf" — but 4 would be inside é, panic at runtime

// Iterators
s.bytes()         .collect::<Vec<u8>>();           // bytes
s.chars()         .collect::<Vec<char>>();         // codepoints
s.char_indices()  .collect::<Vec<(usize, char)>>(); // (byte_idx, char)

// Build chars from u32
char::from_u32(0x1F600);                            // Some('😀')

// Validate UTF-8 from raw bytes
let bytes: &[u8] = b"caf\xc3\xa9";
let s = std::str::from_utf8(bytes)?;                // err if invalid

// Normalisation (unicode-normalization crate)
use unicode_normalization::UnicodeNormalization;
"cafe\u{301}".nfc().collect::<String>();   // "café"

// Grapheme segmentation (unicode-segmentation crate)
use unicode_segmentation::UnicodeSegmentation;
"👨‍👩‍👧‍👦café".graphemes(true).collect::<Vec<&str>>();
// ["👨‍👩‍👧‍👦", "c", "a", "f", "é"]

// OsStr / OsString — for filesystem paths that are NOT guaranteed UTF-8.
// On Unix: arbitrary bytes; on Windows: ill-formed UTF-16 ("WTF-8" representation internally).
use std::ffi::OsStr;
let p = std::path::Path::new("naïve.txt");
p.to_str();                       // Option<&str> — None if path isn't valid UTF-8
p.to_string_lossy();              // Cow<str> with U+FFFD replacements
```

## Unicode in Java

```java
// Java strings are sequences of UTF-16 char (16-bit) values.
// Same surprise as JavaScript: emoji and non-BMP characters are surrogate pairs.

String s = "𝓗ello";
s.length();                       // 6  (chars = UTF-16 units; 𝓗 is two!)
s.codePointCount(0, s.length());  // 5  (codepoints)
s.codePointAt(0);                 // 119943 (0x1D4D7)
s.charAt(0);                      // '?'  high surrogate, not a complete char

// Iterate by codepoint
s.codePoints().forEach(cp -> System.out.println(Integer.toHexString(cp)));

// Build a String from a codepoint
new String(Character.toChars(0x1F600));        // "😀"
new String(new int[]{0x1F600}, 0, 1);          // "😀"

// Encoding
byte[] utf8 = s.getBytes(StandardCharsets.UTF_8);
String back = new String(utf8, StandardCharsets.UTF_8);

// File IO with explicit encoding (NEVER rely on Charset.defaultCharset())
Files.readString(Path.of("a.txt"), StandardCharsets.UTF_8);
Files.writeString(Path.of("a.txt"), s, StandardCharsets.UTF_8);

// Properties
Character.isLetter('é');           // true
Character.getType('é');            // Character.LOWERCASE_LETTER
Character.UnicodeBlock.of(0x1F600); // EMOTICONS

// Normalisation
import java.text.Normalizer;
Normalizer.normalize(s, Normalizer.Form.NFC);

// Locale-sensitive case
"İ".toLowerCase(Locale.ROOT);                // "i̇"
"İ".toLowerCase(Locale.forLanguageTag("tr"));// "i"

// Grapheme iteration via BreakIterator (built-in)
BreakIterator bi = BreakIterator.getCharacterInstance(Locale.ROOT);
bi.setText("👨‍👩‍👧‍👦café");
for (int s2 = bi.first(), e = bi.next(); e != BreakIterator.DONE; s2 = e, e = bi.next()) {
    System.out.println(text.substring(s2, e));
}

// Common exceptions
//   java.nio.charset.MalformedInputException: Input length = 1
//   java.io.UncheckedIOException: java.nio.charset.MalformedInputException
// Fix: pass the right Charset; don't use system default.
```

## Unicode in C

```c
// C's character types are a museum:
//   char        — byte (1 octet, signedness implementation-defined)
//   wchar_t     — wide character; 16-bit on Windows, 32-bit on Unix (Linux/macOS).
//                 Encoding is locale-dependent: UTF-32 on Linux, UTF-16 on Windows.
//   char16_t    — 16-bit, intended for UTF-16 (C11 <uchar.h>)
//   char32_t    — 32-bit, intended for UTF-32 (C11 <uchar.h>)
//   char8_t     — 8-bit unsigned for UTF-8 (C23 <uchar.h>); pre-C23 use uint8_t

// String literals
"hello"           // char[]    — basic execution charset
u8"café"          // char8_t[] (C23) or char[] (older) — guaranteed UTF-8
u"café"           // char16_t[]
U"café"           // char32_t[]
L"café"           // wchar_t[]

// Set the locale at program start or wide chars print as ?
#include <locale.h>
setlocale(LC_ALL, "");        // adopt the user's locale (LANG env var)

// Wide-char functions (declare with #include <wchar.h>, <wctype.h>):
wprintf(L"%ls\n", L"café");
size_t n = mbstowcs(wbuf, "café", 64);   // multi-byte → wide
size_t m = wcstombs(buf, L"café", 64);   // wide → multi-byte

// Byte-level UTF-8 manipulation needs a library (utf8proc, ICU)
//   #include <utf8proc.h>
//   utf8proc_NFC(...) etc.

// iconv from C
#include <iconv.h>
iconv_t cd = iconv_open("UTF-8", "WINDOWS-1252");
iconv(cd, &inbuf, &inleft, &outbuf, &outleft);
iconv_close(cd);

// Common pitfalls
//   strlen() returns bytes, not characters.
//   tolower()/toupper() are LOCALE-aware byte ops; safe only on ASCII.
//   strncpy(dst, src, n) may truncate inside a UTF-8 sequence — produces invalid bytes.
```

## Regex with Unicode

```bash
# Modern regex flavours support Unicode property escapes:
#   \p{L}, \p{Letter}            any letter
#   \p{Lu}, \p{Uppercase_Letter}
#   \p{Ll}, \p{Lowercase_Letter}
#   \p{N}, \p{Number}
#   \p{Nd}                       decimal digit (any script — 0-9, ٠-٩, ๐-๙ ...)
#   \p{P}, \p{Punctuation}
#   \p{S}, \p{Symbol}
#   \p{Sc}                       currency
#   \p{Z}                        separator
#   \p{Cc}                       control
#   \p{Cf}                       format (ZWJ, BOM, ZWSP)
#   \p{sc=Han}, \p{Script=Latin} script-based
#   \p{Block=Emoticons}          block-based (PCRE)
#   \X                           one grapheme cluster (PCRE/Java/Ruby/regex package)

# Engine-specific switches:
grep -P '\p{L}+' file.txt                    # GNU grep with PCRE
rg     '\p{L}+' file.txt                     # ripgrep, Unicode by default
perl -nE 'say if /\p{L}/' file.txt
python3 -c 'import re; print(re.findall(r"\w+", "café Привет 北京", flags=re.UNICODE))'
javascript: /\p{L}+/u.test("café")           // u flag REQUIRED

# Common surprises:
\d   in JS without /u            matches only ASCII 0-9 + nothing else
\d   in Python 3 (default)        matches ALL Nd codepoints (٠ ๑ ৬ ...)
\w   in JS without /u            matches [A-Za-z0-9_] only
\w   in PCRE without (*UTF) flag matches [A-Za-z0-9_] only
\b   word boundary               unicode-aware in modern engines, ASCII in old ones

# Force Unicode-aware matching:
PCRE:    (*UTF)(*UCP) at the start, or PCRE_UTF8 | PCRE_UCP flags.
ripgrep: --pcre2 for full property support; default is fast Rust regex (subset).
Python:  re works on str (Unicode) by default; for bytes use re.ASCII.
JavaScript: append /u to the regex literal.
Java:    Pattern.compile(p, Pattern.UNICODE_CHARACTER_CLASS).

# Examples
grep -P '\p{Sc}'  invoice.txt              # find any currency symbol
grep -P '\p{Cc}'  data.csv                 # find stray control chars
rg --pcre2 '\X{1,5}'  text.txt             # match 1-5 graphemes
```

## Sorting and Collation

```bash
# ASCII sort is byte-by-byte. Unicode sort uses the UCA (UAX #10).
# UCA assigns each codepoint a sequence of weights at multiple "levels":
#   primary    base letter         (a vs b)
#   secondary  diacritic           (a vs á)
#   tertiary   case                (a vs A)
#   quaternary punctuation, variant
# Locale tailorings change the order for specific languages.

# Locale-aware sort
LC_ALL=C       sort names.txt      # byte order: 'A' < 'B' < 'a' < 'b' < 'á'
LC_ALL=en_US.UTF-8 sort names.txt  # standard English: 'a' = 'A' < 'á' < 'b'
LC_ALL=sv_SE.UTF-8 sort names.txt  # Swedish: a < z < å < ä < ö
LC_ALL=de_DE.UTF-8 sort names.txt  # German: ä equivalent to a
LC_ALL=tr_TR.UTF-8 sort names.txt  # Turkish: dotless ı vs i, etc.

# Python
import locale
locale.setlocale(locale.LC_COLLATE, "sv_SE.UTF-8")
sorted(words, key=locale.strxfrm)

# ICU (preferred for production; locale-correct everywhere)
#   PyICU:  pip install PyICU
import icu
coll = icu.Collator.createInstance(icu.Locale("sv_SE"))
sorted(words, key=coll.getSortKey)

# Java
Collator c = Collator.getInstance(new Locale("sv", "SE"));
Collections.sort(list, c);

# Important gotchas:
#   - "ä" sorts after "z" in Swedish but is equal to "a" in German.
#   - Capital letters can sort BEFORE or AFTER lower depending on locale.
#   - Numbers in strings sort lexically by default ("10" < "2"); use Collator.NUMERIC if available.
#   - Punctuation may be ignored at primary level; "co-op" can sort with "coop".
#   - If reproducibility matters across machines, fix the locale explicitly (e.g., LC_ALL=C.UTF-8).
```

## Bidi (Bidirectional) Text

```bash
# Some scripts are written right-to-left:
#   Arabic, Hebrew, Syriac, Thaana, N'Ko, ...

# Each codepoint has a Bidirectional Class (UAX #9):
#   L     left-to-right strong
#   R     right-to-left strong (Hebrew)
#   AL    arabic letter (RTL)
#   EN    european number, AN arabic number
#   ES/ET/CS  separators
#   WS    whitespace
#   ON    other neutral
#   LRE/RLE/PDF/LRO/RLO/LRI/RLI/FSI/PDI  embedding/override controls

# Browsers and terminals run the Bidi algorithm to display mixed text correctly.
# The stored ("logical") order is the typing order; visual order can differ.

# HTML
<bdi>...</bdi>                       <!-- isolate user-supplied bidi text -->
<p dir="rtl">...</p>                 <!-- explicit base direction -->
<p dir="auto">...</p>                <!-- guess from first strong char -->

# CSS
direction: rtl;
unicode-bidi: isolate;               /* recommended for embedded RTL fragments */

# Bidi controls (rarely needed in source, dangerous if leaked):
U+202A  LRE  Left-to-Right Embedding
U+202B  RLE  Right-to-Left Embedding
U+202C  PDF  Pop Directional Formatting
U+202D  LRO  Left-to-Right Override        # CVE-2021-42574 "Trojan Source"
U+202E  RLO  Right-to-Left Override        # CVE-2021-42574 "Trojan Source"
U+2066  LRI  Left-to-Right Isolate (preferred over LRE)
U+2067  RLI  Right-to-Left Isolate
U+2068  FSI  First Strong Isolate
U+2069  PDI  Pop Directional Isolate

# Detect bidi controls in source code (security):
grep -P '[\x{202A}-\x{202E}\x{2066}-\x{2069}]' file.go
rg '[‪-‮⁦-⁩]' src/

# Many modern compilers (rustc, gcc) now warn on bidi controls in identifiers.
```

## Emoji

```bash
# Emoji are scattered across many blocks:
#   U+1F300 .. U+1F5FF   Miscellaneous Symbols and Pictographs
#   U+1F600 .. U+1F64F   Emoticons (smiley faces)
#   U+1F680 .. U+1F6FF   Transport and Map Symbols
#   U+1F900 .. U+1F9FF   Supplemental Symbols and Pictographs
#   U+1FA70 .. U+1FAFF   Symbols and Pictographs Extended-A
#   U+2600  .. U+26FF    Miscellaneous Symbols (older)
#   U+2700  .. U+27BF    Dingbats (older)
#   U+1F1E6 .. U+1F1FF   Regional Indicator Symbols (used in flag pairs)
#   U+1F3FB .. U+1F3FF   Skin Tone Modifiers (Fitzpatrick types 1-2 to 5-6)
#   U+E0020 .. U+E007F   Tag characters (used in subdivision flags)

# Building blocks:
U+200D    ZERO WIDTH JOINER (ZWJ)        # joins emoji into compound sequences
U+FE0F    VARIATION SELECTOR-16 (text→emoji presentation)
U+FE0E    VARIATION SELECTOR-15 (emoji→text presentation)
U+E0xx    tag chars (subdivision flags like 🏴󠁧󠁢󠁳󠁣󠁴󠁿 Scotland)

# Examples:
👨 + 🏿 (skin tone)               = 👨🏿 man dark skin tone   (2 codepoints, 1 grapheme)
👨 + ZWJ + 🍳                     = 👨‍🍳 man cook            (3 codepoints, 1 grapheme)
👨 + ZWJ + 👩 + ZWJ + 👧 + ZWJ + 👦 = 👨‍👩‍👧‍👦 family       (7 codepoints, 1 grapheme)
🇬 + 🇧 (regional indicators)       = 🇬🇧 UK flag             (2 codepoints, 1 grapheme)
🏴 + tag G B S C T (subdivision)  = 🏴󠁧󠁢󠁳󠁣󠁴󠁿 Scotland          (7+ codepoints, 1 grapheme)
✋ + VS-16                        = ✋️ raised hand emoji presentation
✋ + VS-15                        = ✋︎ raised hand text presentation

# Emoji sequences in code:
"\u{1F468}\u{200D}\u{1F373}"      // 👨‍🍳 in JS
"\U0001F468‍\U0001F373"      // 👨‍🍳 in Python
'\U0001F468' + '‍' + '\U0001F373'  // Go: build with rune literals

# Counting "emoji" correctly = counting grapheme clusters, not codepoints.
```

## Common Unicode Pitfalls

```bash
# 1. Length is not character count.
#    BROKEN  if user.length() > 50: reject(user)             # rejects "naïve"+lots of bytes too soon
#    FIXED   count graphemes via Intl.Segmenter / regex \X / unicode-segmentation

# 2. Substring slicing through a multi-byte character.
#    BROKEN  s[:5]                            # may chop é mid-UTF-8 in JS/Java/C
#    FIXED   slice by codepoints (Python str), or by graphemes (Intl.Segmenter, regex \X).

# 3. Comparing strings without normalising first.
#    BROKEN  if a == b: ...   when one is NFC and other NFD
#    FIXED   if unicodedata.normalize("NFC", a) == unicodedata.normalize("NFC", b):

# 4. Case-insensitive comparison without normalisation or casefolding.
#    BROKEN  a.toLowerCase() == b.toLowerCase()              # Turkish I problem
#    FIXED   a.casefold() == b.casefold(); locale-aware Collator at level=secondary; or
#            uconv -x "any-NFC; any-Lower" then compare.

# 5. Assuming text from "the database" is UTF-8.
#    BROKEN  open(file).read()                               # uses platform default
#    FIXED   open(file, encoding="utf-8")    # ALWAYS specify encoding

# 6. Double-encoding (UTF-8 bytes interpreted as Latin-1, encoded again as UTF-8).
#    Symptom: "café" → "Café"
#    BROKEN  data.encode("utf-8").decode("latin-1").encode("utf-8")     # the bug pattern
#    FIXED   1. decode the bytes with the correct legacy encoding ONCE
#            2. then encode as UTF-8 ONCE
#            ftfy.fix_text(s) is a one-shot repair for many forms of mojibake.

# 7. Hidden characters that change behaviour.
#    Zero-width: U+200B ZWSP, U+200C ZWNJ, U+200D ZWJ, U+FEFF BOM, U+2060 WORD JOINER
#    Bidi:       U+202A..U+202E, U+2066..U+2069
#    Soft hyphen: U+00AD (visible only when broken)
#    Inspect:    perl -CS -ne 'print if /[\x{200B}\x{200D}\x{FEFF}\x{202E}]/' file
#    Strip:      tr -d '\200\213\235\236' (won't work — multi-byte!) → use Python regex instead.

# 8. "It works on my machine."
#    BROKEN  python3 ... → encoding default = locale, which differs per OS
#    FIXED   Set PYTHONUTF8=1 or always pass encoding= explicitly.

# 9. Filesystem normalisation drift.
#    BROKEN  Path("café").exists() returns False even though `ls` shows the file (macOS NFD).
#    FIXED   Compare paths after normalising both sides.

# 10. Truncating user-visible names by char count "fits in 30 characters" → splits a flag.
#     FIXED  Use Intl.Segmenter / regex \X / unicode-segmentation; trim by graphemes.
```

## Common Error Messages

```bash
# Python
#   UnicodeDecodeError: 'utf-8' codec can't decode byte 0xff in position 0: invalid start byte
#     → bytes are not valid UTF-8. Pass the real encoding (probably cp1252).
#   UnicodeDecodeError: 'ascii' codec can't decode byte 0xc3 in position 3: ordinal not in range(128)
#     → wrong encoding=. Open with encoding='utf-8'.
#   UnicodeEncodeError: 'ascii' codec can't encode character 'é' in position 3: ordinal not in range(128)
#     → str.encode() defaulted to ascii. Pass encoding='utf-8' or set PYTHONIOENCODING=utf-8.
#   UnicodeError: encoding with 'idna' codec failed
#     → IDN domain has invalid characters; check for stray whitespace or BOM.

# Java
#   java.nio.charset.MalformedInputException: Input length = 1
#     → bytes do not match the declared charset.
#   java.nio.charset.UnmappableCharacterException
#     → tried to encode a char that the target charset can't represent.
#   sun.io.MalformedInputException
#     → legacy. Replace with NIO Files.readString(path, StandardCharsets.UTF_8).
#   "Invalid byte 1 of 1-byte UTF-8 sequence."  (XML parsers)
#     → file is mis-declared. Inspect with file -i and use the right charset attribute.

# Rust
#   error: stream did not contain valid UTF-8
#     → str::from_utf8 / read_to_string saw bad bytes. Use Vec<u8> + lossy.
#   thread 'main' panicked at 'byte index 1 is not a char boundary'
#     → tried to slice into the middle of a multi-byte char. Use char_indices().

# Go
#   utf8: invalid byte sequence
#     → bytes failed validation; check input encoding.
#   string contains the unicode replacement character U+FFFD
#     → earlier decoder swapped invalid bytes for FFFD silently.

# JavaScript
#   URIError: URI malformed
#     → decodeURIComponent saw an invalid %xx sequence (often a stray %).
#   "The string to be encoded contains characters outside of the Latin1 range."
#     → btoa("café") fails. Encode to UTF-8 bytes first, then base64.

# Bash
#   "command not found: ´foo'"
#     → script saved as UTF-8 with smart quotes; replace with ASCII or save as UTF-8.
#   "syntax error near unexpected token `\xef'"
#     → BOM at start of script. Strip it.

# C
#   stdout: Invalid or incomplete multibyte or wide character
#     → setlocale(LC_ALL, "") not called before wide-char output.
```

## Filenames and Paths

```bash
# Filesystems disagree about how to store filenames:
#
#   ext4 / xfs / btrfs (Linux)   raw byte sequences, no normalisation, no validation
#   APFS (macOS)                 preserves form you give it, but case-insensitive by default
#   HFS+ (older macOS)           NFD-normalised UTF-16
#   NTFS (Windows)               UTF-16, allows ill-formed (lone surrogates)
#   FAT/exFAT                    OEM code page or UTF-16 long names
#   ZFS                          configurable: utf8only=on for validation; normalisation=NFC|NFD|NFKC|NFKD

# Cross-platform consequences:
#   - Copy "café.txt" from Linux (NFC) to macOS HFS+ → stored as NFD → ls shows different bytes.
#   - Same name in different normalisation forms can coexist on Linux but collide on macOS HFS+.
#   - Lone-surrogate names from Windows can't be represented as UTF-8 strings.

# Inspect actual bytes:
ls       | xxd
ls       | uniname

# Normalise filenames in a tree (Linux: NFC, the modern default):
convmv -f utf8 -t utf8 --nfc -r --notest /path

# Per-language file paths
#   Rust         std::ffi::OsStr / OsString, Path / PathBuf — handle non-UTF-8 transparently.
#   Python       os.fsencode("café") → bytes; os.fsdecode(b) → str (using surrogateescape).
#   Go           string is bytes; pass to os.Open as-is; call utf8.ValidString to check.
#   Java         java.nio.file.Path — uses default Charset; specify with Files options.

# Sample bug:
#   Python script can't open files whose names contain accents on macOS because the user
#   typed an NFC name into a config file but `ls` returned NFD bytes.
import os, unicodedata
for name in os.listdir("."):
    nfc = unicodedata.normalize("NFC", name)
    if nfc != name:
        os.rename(name, nfc)
```

## JSON / XML / HTML Encoding

```bash
# JSON (RFC 8259):
#   - MUST be encoded as UTF-8, UTF-16, or UTF-32; UTF-8 is mandatory for interchange.
#   - MUST NOT emit a BOM; receivers MAY ignore one but are not required to.
#   - String contents are sequences of Unicode codepoints.
#   - Non-ASCII can be either:
#       a) raw UTF-8 bytes,                         e.g. "café"
#       b) \uXXXX escape (BMP only, surrogate pairs for non-BMP),
#                                                   e.g. "café", "😀"
#   - JSON has no \U{...} escape; non-BMP chars need surrogate pairs in escape form.
#   - Most modern parsers handle either form. Pretty printers usually emit raw UTF-8.

# Validate a JSON file is UTF-8 with no BOM:
head -c3 file.json | xxd       # NOT ef bb bf
python3 -c 'import json,sys; json.load(open(sys.argv[1], encoding="utf-8"))' file.json

# XML 1.0:
#   - Encoding declared in the XML declaration: <?xml version="1.0" encoding="UTF-8"?>
#   - If absent and no BOM, defaults to UTF-8 or UTF-16 depending on parser.
#   - Numeric character references: &#233; or &#xE9;  (decimal / hex)
#   - Named entities: only five required (&amp; &lt; &gt; &apos; &quot;).
#                     HTML adds &eacute;, &copy;, etc. — these are NOT XML by default.

# HTML5:
#   - Default encoding is UTF-8. Declare with <meta charset="utf-8"> early in <head>.
#   - Numeric refs: &#233; or &#xE9;
#   - Named entities: full HTML5 set (about 2200), e.g. &eacute; &amp; &copy; &mdash;
#   - Always escape & < > " in attributes; ' is good practice in attributes too.

# CSV:
#   - No standard charset; UTF-8 is the modern de facto choice.
#   - Some Excel locales prefer UTF-8 with BOM (otherwise non-ASCII shows as mojibake).
#   - Generators for Excel often emit CRLF + UTF-8 BOM as a compatibility kludge.
#       printf '\xEF\xBB\xBF'  > out.csv && cat data.csv >> out.csv

# YAML:
#   - YAML 1.2 streams are UTF-8/16/32. Tools default to UTF-8.
#   - BOM optional; many parsers tolerate it.
```

## Detecting and Fixing Mojibake

```bash
# Mojibake = visually garbled text from encoding mismatches.

# Recognisable patterns (UTF-8 read as Latin-1 / Windows-1252):
#   "é"       → "Ã©"
#   "ñ"       → "Ã±"
#   "🏴"     → "ð´"
#   "café"    → "café"      (most common — UTF-8 bytes interpreted as cp1252)
#   "naïve"   → "naÃ¯ve"
#   "—"       → "â"  (em dash)

# Recognisable patterns (Latin-1 / cp1252 read as UTF-8 then escaped):
#   bare 0xE9 in a UTF-8 string → "" (U+FFFD replacement char)
#   "Privet" → "?????"  if bytes were dropped during decode

# One-shot fix (Python):
pip install ftfy
python3 -c 'import ftfy; print(ftfy.fix_text("café"))'        # café
python3 -c 'import ftfy; print(ftfy.fix_text("naÃ¯ve"))'      # naïve

# Manual undo of the classic UTF-8→cp1252 double-encoding:
python3 -c 's="café"; print(s.encode("cp1252").decode("utf-8"))'  # café

# CLI workflow when ftfy isn't available:
iconv -f UTF-8 -t CP1252 broken.txt > round1.bin    # turn bad UTF-8 back into raw cp1252 bytes
iconv -f UTF-8 -t UTF-8  round1.bin  > fixed.txt    # confirm round1 is now valid UTF-8

# Find files affected by mojibake (heuristic: presence of "Ã©", "Ã ", "â"):
grep -rEl 'Ã[©°®¨º]|â€“|â€”|â€¢' /data

# Defensive ingest — never call .encode() twice:
#   1. read raw bytes
#   2. decode with the actual source encoding (detect once)
#   3. work with str/String the rest of the way
```

## Tools

```bash
# Inspection
hexdump -C file              # canonical hex+ASCII
xxd       file               # similar, more flexible
xxd -ps   file               # pure hex; xxd -r -p reverses
od  -An -c file              # bytes as printable+escapes
od  -An -tx1 file            # bytes as hex
od  -An -tx4 file            # 32-bit big-endian (handy for UTF-32)

# Codepoint info
uniname                      # codepoint per line with name (uniutils)
unifuzz                      # search by partial name
unidesc                      # describe a string
ucs                          # batch lookups

# Conversion
iconv -f F -t T file         # the workhorse
recode F..T file             # alternative
uconv -x "rule"              # ICU transliteration: any-NFC, any-ascii, latin-greek, ...

# Detection
file  -i  file               # MIME charset (often UTF-8 / us-ascii / unknown-8bit)
uchardet file                # Mozilla detector (best for non-Latin)
chardet  file                # Python detector
enca -L none file            # strong on East European

# Repair
ftfy                         # Python: fix double-encoding mojibake
unidecode                    # Python: lossy ASCII transliteration

# Search
rg --pcre2 '\p{Han}'         # ripgrep with full Unicode regex
grep -P '...'                # GNU grep with PCRE
ggrep                        # GNU grep on macOS via brew

# Strings extraction
strings           file       # ASCII strings (default)
strings -e l      file       # UTF-16LE strings
strings -e b      file       # UTF-16BE strings
strings -e L      file       # UTF-32LE
strings -e B      file       # UTF-32BE
strings -e S      file       # 8-bit strings (incl. high-bit)

# Perl unicode-mode flags
perl -CS    -ne '...'        # STDIN/OUT/ERR are UTF-8
perl -CSDA  -ne '...'        # IN/OUT/ARGV/files all UTF-8
perl -Mutf8 -e '...'         # source file is UTF-8

# Editor settings
# Vim:    set encoding=utf-8 fileencoding=utf-8 nobomb
# Emacs:  (prefer-coding-system 'utf-8-unix)
# VS Code: "files.encoding": "utf8", "files.autoGuessEncoding": false
```

## Idioms

```bash
# 1. Always specify encoding when opening files.
#    Python:  open(p, encoding="utf-8")
#    Java:    Files.readString(p, StandardCharsets.UTF_8)
#    Go:      os.ReadFile(p)            # bytes — convert with utf8.Valid before string()
#    Rust:    fs::read_to_string(p)?    # built-in UTF-8 validation
#    Node:    fs.readFileSync(p, "utf-8")
#    C#:      File.ReadAllText(p, Encoding.UTF8)

# 2. Never rely on the system default charset/locale.
#    Set explicitly. In Python: PYTHONUTF8=1 or always pass encoding=.
#    In Java: java -Dfile.encoding=UTF-8 ... (or use the explicit Charset overload).

# 3. Normalise before storing or comparing user text.
#    NFC for storage. NFKC + casefold for "search-friendly" indexes.

# 4. Casefold for case-insensitive comparison; not toLowerCase().
#    str.casefold() in Python; UCharacter.foldCase in ICU; Collator at level=secondary in Java.

# 5. Truncate by graphemes for user-visible strings.
#    Use Intl.Segmenter (JS), regex \X (Python regex/PCRE), unicode-segmentation (Rust).

# 6. Sort with an explicit locale or collator.
#    LC_ALL=C.UTF-8 sort -u   for byte-stable de-dupe across machines.
#    Use ICU Collator for human-friendly ordering.

# 7. Validate input early.
#    Reject bytes that aren't valid UTF-8 at the boundary; do NOT carry "raw" mystery bytes
#    through your code as strings.

# 8. Treat file paths as opaque OS-native types.
#    Rust OsStr/PathBuf, Python pathlib + bytes when needed, Go string of OS bytes.

# 9. Strip dangerous controls from untrusted input.
#    Bidi controls (U+202A..U+202E, U+2066..U+2069), zero-widths (U+200B..U+200D, U+FEFF),
#    and arbitrary U+2028/U+2029 line separators if you display in a terminal or HTML.

# 10. When in doubt, dump bytes.
#    `head -c 20 file | xxd`  is faster than guessing at encoding errors.
```

## Performance

```bash
# UTF-8 validation
#   Modern libraries (simdjson, simdutf, Rust std) validate UTF-8 at memory bandwidth using
#   SIMD (AVX2 / NEON). Validation is essentially free on multi-GB inputs.
#   Older byte-by-byte validators are 5-10x slower.

# Normalisation
#   Surprisingly expensive: requires table lookups and reordering combining marks.
#   Cache the result if you compare repeatedly. Most strings are already in NFC; a quick
#   "is_quick_check_NFC" pass returns YES/NO/MAYBE in O(n) without allocating.
#   Python: unicodedata.is_normalized("NFC", s)

# Codepoint iteration
#   UTF-8 is variable-width, so finding the n-th codepoint is O(n).
#   If you need O(1) random access, build an index of (byte_offset → codepoint_index)
#   or convert to []rune / Vec<char> / array of UTF-32 once.

# Grapheme segmentation
#   Per-codepoint state machine; ~50-100 ns per grapheme on a modern CPU.
#   Cache results. For most non-emoji ASCII text, segmentation is trivial.

# Collation
#   UCA generates sort keys that are 2-4x the size of the input string.
#   For repeated sorting, materialise the sort key once with collator.getSortKey()
#   then sort by the byte arrays. Massive speedup over recomputing per comparison.

# Locale-correct lower/upper
#   Locale-aware case mapping is 10-50x slower than ASCII tolower(). Hot loops should
#   handle ASCII inline and only fall back to ICU for non-ASCII bytes.

# Encoding conversion
#   iconv is byte-table based and very fast (≥1 GB/s for trivial pairs like Latin-1↔UTF-8).
#   GB18030 and Shift-JIS to/from UTF-8 use larger tables but are still cheap.
```

## Tips

```bash
# - "UTF-8 unless someone makes you stop" is the right default for text data on disk and on the wire.
# - Read once with the actual source encoding, decode to your language's string type,
#   keep it in memory in that form, and encode back to UTF-8 on output.
# - Set LANG=en_US.UTF-8 (or C.UTF-8) on every server. It silently fixes 80% of problems.
# - Avoid the BOM on UTF-8 unless a Microsoft tool requires it. It is the single most common
#   cause of "weird first character" bugs in shell scripts and JSON.
# - When you see U+FFFD () in your data, you decoded with errors='replace'; the raw bytes
#   are gone. Re-ingest from the source with the correct encoding.
# - When you see Ã©, ä¸­ or similar ASCII-looking junk, you have mojibake; one round of
#   ftfy.fix_text or iconv -f UTF-8 -t cp1252 | iconv -f UTF-8 will repair most cases.
# - Comparing usernames, passwords, or identifiers without NFKC + casefold is a security bug.
#   Two visually identical strings may have different bytes (homoglyph attacks).
# - Validate UTF-8 at trust boundaries. Once a string is a "valid UTF-8 string" type in your
#   language, the rest of your code can stay simple.
# - Test with non-Latin scripts (Chinese, Arabic, Devanagari) and emoji from day one. Bugs
#   that only appear in i18n testing tend to ship.
# - Keep a tiny test fixture that includes:
#       ASCII-only:           Hello world
#       Latin-1:              Café résumé
#       CJK:                  你好世界
#       Bidi:                 مرحبا بالعالم
#       Combining:            é (U+0065 U+0301)
#       Emoji ZWJ sequence:   👨‍👩‍👧‍👦
#       Flag:                 🇬🇧
#   It will catch 90% of regressions before users do.
```

## See Also

- regex, polyglot, awk, python, javascript, typescript, ruby, go, rust, java, c, bash

## References

- [The Unicode Standard](https://www.unicode.org/standard/standard.html) — full standard, current version
- [Unicode Glossary](https://www.unicode.org/glossary/) — definitive definitions of every term
- [Unicode Character Code Charts](https://www.unicode.org/charts/) — charts per block, with PDFs
- [UTF-8 Everywhere](https://www.utf8everywhere.org/) — manifesto and rationale
- [UAX #9 — Unicode Bidirectional Algorithm](https://unicode.org/reports/tr9/) — RTL/LTR layout
- [UAX #15 — Unicode Normalization Forms](https://unicode.org/reports/tr15/) — NFC/NFD/NFKC/NFKD
- [UAX #18 — Unicode Regular Expressions](https://unicode.org/reports/tr18/) — Unicode requirements for regex
- [UAX #29 — Unicode Text Segmentation](https://unicode.org/reports/tr29/) — grapheme/word/sentence boundaries
- [UAX #31 — Unicode Identifier and Pattern Syntax](https://unicode.org/reports/tr31/) — identifiers
- [UTS #10 — Unicode Collation Algorithm](https://unicode.org/reports/tr10/) — language-correct sorting
- [UTS #39 — Unicode Security Mechanisms](https://unicode.org/reports/tr39/) — homoglyph/spoofing detection
- [UTS #46 — IDNA Compatibility Processing](https://unicode.org/reports/tr46/) — internationalised domain names
- [Latest UCA data](https://unicode.org/Public/UCA/latest/) — collation tables
- [RFC 3629 — UTF-8](https://www.rfc-editor.org/rfc/rfc3629)
- [RFC 2781 — UTF-16](https://www.rfc-editor.org/rfc/rfc2781)
- [RFC 8259 — JSON](https://www.rfc-editor.org/rfc/rfc8259) — encoding requirements
- [RFC 5198 — Unicode Format for Network Interchange](https://www.rfc-editor.org/rfc/rfc5198)
- [ICU User Guide](https://unicode-org.github.io/icu/userguide/) — production-grade Unicode library
- [CLDR — Common Locale Data Repository](https://cldr.unicode.org/) — locale data
- [man iconv](https://man7.org/linux/man-pages/man1/iconv.1.html)
- [man locale](https://man7.org/linux/man-pages/man1/locale.1.html)
- [man uniname](https://manpages.debian.org/testing/uniutils/uniname.1.en.html)
