# Regex (Regular Expressions)

> Pattern matching language for searching, validating, and transforming text — covering POSIX BRE/ERE, PCRE/PCRE2, RE2, JavaScript, Python, Go, Rust, Java, .NET, Ruby Onigmo, Vim, sed, awk, and grep dialects.

## Setup — Dialects

### The Dialect Landscape

```bash
# Major regex dialects you will encounter:
#
# POSIX BRE  — grep (default), sed (default), ed, vi (default)
# POSIX ERE  — grep -E, egrep, awk, sed -E, lex
# PCRE/PCRE2 — grep -P, pcregrep, PHP preg_*, nginx, Apache, Nim
# JavaScript — ECMAScript ES2018+ (V8, SpiderMonkey, JavaScriptCore)
# Python re  — CPython re module (Python 3)
# Python regex — third-party regex module (more features)
# Go RE2     — Go regexp package, RE2/C++, Cloud Bigtable
# Rust regex — Rust regex crate (RE2-like, no backrefs/lookaround)
# Java       — java.util.regex (java.util.regex.Pattern)
# .NET       — System.Text.RegularExpressions
# Ruby Onigmo — Ruby 1.9+ regex engine (formerly Oniguruma)
# Vim        — Vim's own dialect (4 magic levels: nomagic, magic, very magic, very nomagic)
# sed BRE    — sed default (POSIX BRE plus GNU extensions)
# awk ERE    — awk uses POSIX ERE
# grep -P    — PCRE2 wrapper, only on Linux GNU grep (not BSD/macOS)
# tcl        — Tcl's ARE (advanced) and BRE/ERE
# Hyperscan  — Intel's high-performance regex (subset of PCRE)
# RE/flex    — fast NFA-based engine
```

### Compatibility Matrix Quick Reference

```bash
# Feature              BRE  ERE  PCRE  JS   Py   Go   Rust Java .NET Ruby
# ----------------------------------------------------------------------
# + ? quantifiers      no   yes  yes   yes  yes  yes  yes  yes  yes  yes
# {n,m} quantifier     \{\} {n,m} yes  yes  yes  yes  yes  yes  yes  yes
# Capturing (...)      \(\) ()   yes   yes  yes  yes  yes  yes  yes  yes
# Alternation a|b      no   yes  yes   yes  yes  yes  yes  yes  yes  yes
# Lookahead (?=)       no   no   yes   yes  yes  no   no   yes  yes  yes
# Lookbehind (?<=)     no   no   yes   yes* yes  no   no   yes  yes  yes
# Backreferences       \1   no** yes   yes  yes  no   no   yes  yes  yes
# Named groups         no   no   yes   yes  yes  yes  no***yes  yes  yes
# Atomic groups (?>)   no   no   yes   no   no   no   no   yes  yes  yes
# Possessive *+ ++     no   no   yes   no   no   no   no   yes  no   yes
# Unicode \p{L}        no   no   yes   yes  yes  no****yes yes  yes  yes
# Conditional (?(...)) no   no   yes   no   no   no   no   no   yes  no
#
# *  JavaScript got lookbehind in ES2018 (Node 10+, Safari 16.4+)
# ** Some ERE implementations add backrefs as a non-POSIX extension
# *** Rust regex has no named-group backrefs, but does have named captures
# **** Go regex does support \p{} but no named property aliases like \p{Latin}
```

### Picking the Right Tool

```bash
# Matching trusted patterns? — any backtracking engine is fine
# Matching hostile/user input? — use RE2 (Go), Rust regex, or Hyperscan
# Need backreferences or lookaround? — use PCRE/Java/Python/.NET/Ruby
# CLI search?                    — ripgrep (rg) > grep -P > grep -E
# In-shell scripting?            — sed -E, awk, grep -E
# Embedded in C?                 — link libpcre2 or RE2
# Embedded in firmware?          — re2c (compiles regex to C source)
```

## Anchors

### Standard Anchors

```bash
^         # Start of string (or line in /m multiline mode)
$         # End of string (or line in /m multiline mode)
\A        # Absolute start of input — ignores multiline flag (PCRE/Python/Java/Ruby)
\Z        # End of input — before final newline (PCRE/Python/Java/Ruby)
\z        # Absolute end of input — after final newline (PCRE/Python/Java/Ruby)
\G        # Where last match ended (PCRE/Java/.NET) — for iterative matching
```

### Word Boundaries

```bash
\b        # Word boundary — between \w and \W (or string edge)
\B        # Non-word boundary — between \w-\w or \W-\W
\<        # Start of word (GNU grep, gawk, Vim, sed GNU)
\>        # End of word (GNU grep, gawk, Vim, sed GNU)
\m        # Start of word (Ruby Onigmo)
\M        # End of word (Ruby Onigmo)

# Examples
\bfoo\b   # Match "foo" as a whole word
\Bfoo\B   # Match "foo" only inside another word, like "barfoobaz"
\bcat     # Match "cat" or "cats" — not "scatter"
```

### Multiline Mode Behaviour

```bash
# Default mode (/m off):
^abc$     # Matches if entire string is exactly "abc"

# Multiline mode (/m on):
^abc$     # Matches "abc" on any line of multi-line input

# Python explicit:
re.search(r'^abc$', text, re.MULTILINE)

# JavaScript:
/^abc$/m

# Go (RE2):
regexp.MustCompile(`(?m)^abc$`)
```

## Character Literals and Escapes

### Literal Special Characters

```bash
\.        # Literal dot
\\        # Literal backslash
\/        # Literal slash (often unnecessary — but required in JS literals)
\(        # Literal parenthesis (in PCRE/ERE; in BRE the bare ( is literal)
\)
\[
\]        # Literal bracket — required in PCRE; sometimes optional inside [...]
\{
\}
\+        # Literal plus (required in BRE; in ERE/PCRE bare + is the metachar)
\?
\|        # Literal pipe (required in BRE)
\^        # Literal caret — only required at start of pattern or [...]
\$        # Literal dollar — only required at end
```

### Control Characters

```bash
\n        # Newline (0x0A)
\r        # Carriage return (0x0D)
\t        # Tab (0x09)
\f        # Form feed (0x0C)
\v        # Vertical tab (0x0B)
\0        # Null character (0x00) — danger: treated as literal '0' in some dialects
\a        # Bell (0x07)
\e        # ESC (0x1B) — PCRE/Ruby/Java/.NET, NOT in JavaScript or Go
```

### Numeric Escapes

```bash
\xHH         # Hex byte: \x41 == "A"
\x{HHHH}     # Extended hex: \x{1F600} == U+1F600 (PCRE, Ruby, Python, Java with /u, JS with /u)
\uHHHH       # 4-digit Unicode: é == "é" (JS, Java, .NET, Python)
\u{HHHHH}    # Variable-width Unicode escape (JS with /u, PCRE)
\cX          # Control character: \cA == 0x01, \cZ == 0x1A
\NNN         # Octal byte (sed, awk): \101 == "A"
\o{HHH}      # Explicit octal escape (Perl 5.14+)
```

### Common Confusion

```bash
# JS WITHOUT /u flag — \u{1F600} is interpreted as "u{1F600}" (literal u and curly braces)
# Always use /u flag for Unicode literals in modern JS:
/^\u{1F600}$/u   # works
/^\u{1F600}$/    # SyntaxError or wrong match in older engines

# Python — must use raw strings to preserve escapes:
re.search(r'\d+', text)    # CORRECT
re.search('\d+', text)     # works but emits DeprecationWarning since 3.6
re.search('\\d+', text)    # equivalent verbose form
```

## Character Classes — Ranges

### The Universal Wildcard

```bash
.         # Any character except newline (default)
.         # Any character including newline (in /s singleline / DOTALL mode)
[\s\S]    # Any character including newline — works in ALL dialects
[\d\D]    # Same idea using digit classes
[^]       # Any character including newline — JavaScript ONLY (don't use elsewhere)
```

### Custom Character Classes

```bash
[abc]        # Match a, b, or c
[^abc]       # Match anything EXCEPT a, b, c
[a-z]        # Lowercase letter (ASCII range U+0061..U+007A)
[A-Z]        # Uppercase letter
[0-9]        # Digit
[a-zA-Z0-9]  # Alphanumeric
[a-fA-F0-9]  # Hex digit
[!-~]        # Any printable ASCII (U+0021..U+007E)
```

### Hyphen and Bracket Quirks

```bash
# Hyphen: literal at start, end, or escaped
[-abc]       # Hyphen, a, b, c
[abc-]       # a, b, c, hyphen
[a\-c]       # a, hyphen, c
[a-c]        # range a..c (a, b, c)

# Right bracket: literal at the start (after ^ if negated)
[]abc]       # Right bracket, a, b, c — POSIX form
[\]abc]      # Same, with explicit escape (PCRE/JS/Python form)
[^]abc]      # NOT right bracket, a, b, c — POSIX
[^\]abc]     # Same, escaped form

# Caret: literal anywhere except first character
[^abc]       # NOT a, b, c
[a^bc]       # a, ^, b, c
```

### Range Pitfalls

```bash
# Wrong order errors:
[z-a]        # ERROR: "Range out of order in character class" (Python re.error)
             # PCRE: "range out of order in character class"
             # Java: PatternSyntaxException "Illegal character range"

# Locale ranges differ across systems:
[a-Z]        # Probably not what you want — depends on collation
[A-z]        # Includes [, \, ], ^, _, ` — DON'T DO THIS
[A-Za-z]     # Correct way to match any ASCII letter

# Bracket-class inside bracket-class — only POSIX classes work:
[[:alpha:]_]   # Letters and underscore
[[a]]          # ERROR or weird match — not a "set of sets"
```

## POSIX Character Classes

### The Classes

```bash
[[:alpha:]]   # Letters         [a-zA-Z] in C locale
[[:alnum:]]   # Letters+digits  [a-zA-Z0-9]
[[:digit:]]   # Digits           [0-9]
[[:upper:]]   # Uppercase
[[:lower:]]   # Lowercase
[[:space:]]   # Whitespace      [ \t\n\r\f\v]
[[:blank:]]   # Space and tab   [ \t]
[[:punct:]]   # Punctuation
[[:print:]]   # Printable
[[:graph:]]   # Printable, no space
[[:cntrl:]]   # Control characters
[[:xdigit:]]  # Hex digit       [0-9a-fA-F]
[[:word:]]    # Word character — GNU grep, PCRE — same as \w
[[:ascii:]]   # ASCII chars 0-127 — PCRE only
```

### Usage Rules

```bash
# CORRECT — POSIX classes only work INSIDE [...]
grep '[[:digit:]]+' file       # Match digits
grep -E '^[[:upper:]][[:lower:]]+$' file

# WRONG — outside brackets, [:digit:] is literal characters
grep ':digit:' file            # Matches literal ":digit:"

# Negated class inside POSIX:
[^[:digit:]]                   # Anything NOT a digit

# Combining:
[[:alpha:][:digit:]_-]         # Letter, digit, underscore, hyphen
```

### Locale Effects

```bash
# POSIX classes are LOCALE-AWARE:
LC_ALL=C grep '[[:alpha:]]'    # ASCII letters only
LC_ALL=en_US.UTF-8 grep '[[:alpha:]]'   # Includes é, ü, etc.

# \w, \d, [a-z] are NOT locale-aware in most engines (ASCII only by default)
# Use \p{L} \p{N} for Unicode-aware matching
```

## Shorthand Character Classes

### The Common Set

```bash
\d        # Digit
\D        # Non-digit
\w        # Word character (letter, digit, underscore)
\W        # Non-word character
\s        # Whitespace
\S        # Non-whitespace
\h        # Horizontal whitespace (PCRE, Ruby) — [ \t]
\H        # Non-horizontal-whitespace
\v        # Vertical whitespace (PCRE) — \n\r\f\v
\V        # Non-vertical-whitespace
\R        # Any line break (PCRE, Java) — \r\n|[\r\n\v\f\x{85}\x{2028}\x{2029}]
\N        # Any character except newline (PCRE) — like . regardless of /s flag
```

### Engine-Specific Definitions

```bash
# \d means:
# - PCRE/Java/Python (default):       [0-9]
# - PCRE with (*UCP):                  Unicode digit category Nd
# - Python with re.UNICODE (default):  Unicode digit category Nd
# - JavaScript without /u:             [0-9]
# - JavaScript with /u (no Unicode):   [0-9]
# - Go RE2 (default):                  [0-9]
# - .NET (default):                    Unicode digit category Nd
# - Ruby (default):                    [0-9] in ASCII strings, Unicode in others

# \w means:
# - Most engines (default): [A-Za-z0-9_]
# - .NET (default):          Unicode letter, digit, mark, underscore, connector punctuation
# - Python (Unicode mode):   Unicode letter, digit, underscore (not all marks)
```

### Unicode Property Classes

```bash
\p{L}       # Any letter
\p{Letter}  # Same — long form
\p{Ll}      # Lowercase letter
\p{Lu}      # Uppercase letter
\p{Lt}      # Titlecase letter
\p{Lm}      # Modifier letter
\p{Lo}      # Other letter (CJK, Hebrew, etc.)
\p{N}       # Any number
\p{Nd}      # Decimal digit
\p{Nl}      # Letter number (Roman numerals)
\p{No}      # Other number (¼, ½, etc.)
\p{P}       # Punctuation
\p{Pc}      # Connector punctuation (_)
\p{Pd}      # Dash punctuation
\p{S}       # Symbol
\p{Sc}      # Currency
\p{Sm}      # Math symbol
\p{Z}       # Separator
\p{Zs}      # Space separator
\p{Zl}      # Line separator
\p{Zp}      # Paragraph separator
\p{C}       # Control / other
\p{Cc}      # Control character
\p{Cf}      # Format
\p{M}       # Mark
\p{Mn}      # Non-spacing mark (combining accents)
\p{Mc}      # Spacing mark

\P{L}       # NOT a letter (capital P negates)

# Script properties (PCRE, Java, Python regex module, .NET):
\p{Latin}     # Latin script
\p{Greek}     # Greek script
\p{Han}       # Chinese characters
\p{Hiragana}  # Japanese hiragana
\p{Katakana}
\p{Cyrillic}
\p{Arabic}
\p{Hebrew}
```

### Cross-Engine Unicode

```bash
# JavaScript needs the /u flag:
/\p{L}/u            # CORRECT — Unicode letter
/\p{L}/             # SyntaxError: Invalid regular expression: /\p{L}/: Invalid escape

# Go RE2 supports \p but with limited categories:
regexp.MustCompile(`\p{L}+`)         # Works
regexp.MustCompile(`\p{Latin}+`)     # Works in Go 1.x

# Python's stdlib re — no \p{} support:
re.search(r'\p{L}', text)            # error: bad escape \p
# Use the third-party 'regex' module:
import regex
regex.search(r'\p{L}', text)         # Works
```

## Quantifiers

### Greedy Quantifiers (Default)

```bash
*           # 0 or more — match as many as possible
+           # 1 or more
?           # 0 or 1 (optional)
{n}         # Exactly n
{n,}        # At least n
{n,m}       # Between n and m (inclusive)
{,m}        # Up to m — PCRE/Java/Python; NOT in older JS
{0,m}       # Same, portable form
```

### Lazy / Reluctant Quantifiers

```bash
*?          # 0 or more — match as few as possible
+?          # 1 or more, lazy
??          # 0 or 1, lazy (prefer 0)
{n}?        # Exactly n (no laziness, but valid)
{n,}?       # At least n, lazy
{n,m}?      # Between n and m, lazy
```

### Possessive Quantifiers (PCRE/Java/Ruby Onigmo)

```bash
*+          # 0 or more, possessive (no backtracking)
++          # 1 or more, possessive
?+          # 0 or 1, possessive
{n,m}+      # Between n and m, possessive

# Possessive prevents backtracking — useful for catastrophic backtracking fixes
# Equivalent to atomic group: (?>X*) is the same as X*+

# Engines:
# PCRE          ✓
# Java          ✓
# Ruby          ✓
# Python re     ✗  (planned for 3.11+, partial support)
# Python regex  ✓  (third-party module)
# JavaScript    ✗
# Go            ✗
# Rust          ✗
# .NET          ✗  (use atomic groups (?>...))
```

### Examples

```bash
\d{3}                 # Exactly 3 digits
\d{3,5}               # 3 to 5 digits
\d{3,}                # 3 or more digits
[a-z]+                # 1+ lowercase letters
"[^"]*"               # Empty or non-empty quoted string (no embedded quotes)
".*?"                 # Quoted string — lazy match (one tag at a time)
".*"                  # Quoted string — greedy (matches across multiple tokens)
```

## Greedy vs Lazy

### The Canonical Example

```bash
# Input: '<b>bold</b> and <i>italic</i>'

# Greedy:
<.*>                  # Matches: <b>bold</b> and <i>italic</i>
                      # The whole span — `.*` consumes everything, then backtracks
                      # to find the last `>`.

# Lazy:
<.*?>                 # Matches: <b>
                      # First match. Use a global flag to find each tag.

# Negated character class — usually best:
<[^>]*>               # Matches: <b>, </b>, <i>, </i>
                      # No backtracking, no laziness, single pass.
```

### The Dot-Star Problem

```bash
# Input: 'key="hello" key2="world"'

# Wrong (greedy):
key="(.*)"            # Captures: hello" key2="world
                      # Greedy `.*` eats too much.

# Better (lazy):
key="(.*?)"           # Captures: hello

# Best (negated class — fastest, no backtracking):
key="([^"]*)"         # Captures: hello

# Lazy can still be too aggressive across newlines:
<.*?>                 # Lazy, but may not span lines without /s flag
<[^>]*>               # Negation works regardless of /s flag
```

### Greedy Across Lines

```bash
# Default: . does NOT match newline
.*                    # Stops at first newline

# Singleline / DOTALL mode:
(?s).*                # . matches newline too — match runs across lines
[\s\S]*               # Portable — any char including newline (no flag needed)
```

## Groups

### Capturing Groups

```bash
(abc)                 # Group 1 captures "abc"
(a)(b)(c)             # Groups 1, 2, 3
(a(b)(c))             # Group 1: "abc"; Group 2: "b"; Group 3: "c"
                      # Outer groups numbered before inner
\1                    # Backreference to group 1
\2                    # Backreference to group 2
```

### Non-Capturing Groups

```bash
(?:abc)               # Group, but no capture (saves overhead)
(?:a|b|c)             # Alternation without capture
```

### Named Groups — The Portability Mess

```bash
# Python original (works in Python 3, PCRE):
(?P<name>pattern)
(?P=name)             # Backreference

# .NET / PCRE7+ / JavaScript ES2018+ / Java / Ruby:
(?<name>pattern)
\k<name>              # Backreference

# Both forms work in PCRE/PCRE2:
(?<name>...) or (?P<name>...)     # Both legal

# Go RE2 — only the Python form:
(?P<name>pattern)     # Works
(?<name>pattern)      # ERROR

# Rust regex — supports both Python and angle-bracket forms:
(?P<name>pattern)
(?<name>pattern)
```

### Group Replacement

```bash
# Replacement syntax differences:
# Perl/PCRE/sed-GNU/JavaScript:   $1 or \1
# Python:                          \1 or \g<1> or \g<name>
# Go:                              ${1} or ${name}
# Java:                            $1 or ${name}
# .NET:                            $1 or ${name}
# Ruby:                            \1 or \k<name>
```

## Alternation

### Basic Alternation

```bash
a|b                   # Match "a" or "b"
cat|dog               # Match "cat" or "dog"
red|green|blue        # Multi-way alternation
```

### Precedence Pitfall

```bash
# COMMON MISTAKE:
ABC|DEF               # Means: "ABC" OR "DEF" (NOT "AB" + "C|D" + "EF")

# What you might actually mean:
AB(C|D)EF             # "ABCEF" or "ABDEF"
AB[CD]EF              # Same, but faster (no group)

# Anchored alternation:
^cat|dog$             # "cat" at start OR "dog" at end — likely not intended
^(cat|dog)$           # Either "cat" or "dog" as whole string — likely intended
```

### Alternation Performance

```bash
# Slow — repeated common prefix:
http://www\.example\.com|http://www\.example\.org|http://www\.example\.net

# Fast — factor common prefix:
http://www\.example\.(?:com|org|net)

# Even faster — character class:
http://www\.example\.[a-z]{3}      # Matches three letters
```

### Order Matters in Backtracking Engines

```bash
# Backtracking engines try alternatives left-to-right and stop at first match.
# Put longer/more specific alternatives FIRST to avoid surprises.

# Wrong:
\b(do|don't|doing)\b           # Matches "do" in "don't" because "do" comes first

# Right:
\b(don't|doing|do)\b           # Tries longer first
```

## Lookahead and Lookbehind

### Lookahead (Zero-Width Assertion Forward)

```bash
(?=pattern)           # Positive lookahead
(?!pattern)           # Negative lookahead

# Examples:
\d+(?=px)             # Digits followed by "px" — "12" in "12px"
\d+(?!px)             # Digits NOT followed by "px"
foo(?=bar|baz)        # "foo" before "bar" or "baz"

# Multiple lookaheads at one point (AND logic):
(?=.*[A-Z])(?=.*\d)\w{8,}   # Word ≥8 chars containing uppercase AND digit
```

### Lookbehind (Zero-Width Assertion Backward)

```bash
(?<=pattern)          # Positive lookbehind
(?<!pattern)          # Negative lookbehind

# Examples:
(?<=\$)\d+            # Digits preceded by "$"
(?<!un)\w+            # Word NOT preceded by "un"
(?<=foo)bar           # "bar" after "foo"
```

### Variable-Width Lookbehind

```bash
# Java, .NET, PCRE2 (10.30+), Python regex module — variable-width supported
(?<=foo|prefix\d{1,5})bar       # OK in Java, .NET, PCRE2

# PCRE (older), JavaScript pre-2024 — fixed-width only
(?<=\d{3})bar         # OK
(?<=\d+)bar           # PCRE error: "lookbehind assertion is not fixed length"

# Go RE2, Rust regex — NO lookbehind at all
# Workarounds:
# 1. Restructure regex
# 2. Match more, then check programmatically
# 3. Use \K to reset match start (PCRE only — not real lookbehind)
```

### Width-Zero Behaviour

```bash
# Lookarounds DO NOT consume input. The match position does not advance.

# Pattern: (?=foo)foo
# Input: "foobar"
# Step 1: at position 0, lookahead asserts "foo" present — succeeds, position stays at 0
# Step 2: literal "foo" matches positions 0-2
# Final match: "foo" at 0-2
```

## Atomic Groups and Possessive Quantifiers

### Atomic Groups

```bash
(?>pattern)           # Atomic — once matched, no backtracking into this group

# Engines: PCRE, Java, Ruby, .NET, Python regex module
# NOT in: JavaScript, Python re (stdlib pre-3.11), Go, Rust

# Equivalences:
# (?>X*)  ≡  X*+
# (?>X+)  ≡  X++
# (?>X?)  ≡  X?+
```

### Why You Need Them

```bash
# Catastrophic backtracking pattern — exponential time on long input:
^(a+)+$               # On "aaaaaaaaaaX" — explodes

# Atomic-group fix:
^(?>a+)+$             # No backtracking — bounded time

# Possessive-quantifier fix (same idea, different syntax):
^a++$                 # Linear time
```

### Real-World Example

```bash
# Email validation written naively:
^([a-zA-Z0-9._%+-]+)@([a-zA-Z0-9.-]+)\.([a-zA-Z]{2,})$
# Catastrophic on "aaaaaaaaaaaaaaaaaaaa" (no @):
# ((+))(+) leaves many overlapping matches to retry.

# Defensive version with atomic groups:
^(?>[a-zA-Z0-9._%+-]+)@(?>[a-zA-Z0-9.-]+)\.(?>[a-zA-Z]{2,})$
```

## Conditional Patterns

### Syntax (PCRE / Perl Only)

```bash
(?(condition)yes|no)
(?(1)matched|not-matched)        # Branch on whether group 1 captured
(?(<name>)matched|not-matched)   # Branch on named group
(?(?=lookahead)yes|no)           # Branch on lookahead
```

### Use Cases (Rare)

```bash
# Match optional opening quote with matching closing quote:
^(['"])?(.*?)(?(1)\1|)$

# Engines: PCRE, Perl, Boost.Regex, .NET (limited)
# NOT in: most modern engines — usually replaced by simpler logic
```

## Modifiers / Flags

### Standard Flags

```bash
i         # Case-insensitive
m         # Multiline — ^ and $ match each line
s         # Singleline / DOTALL — . matches newline
x         # Extended — whitespace ignored, # introduces comments
g         # Global — find all matches (sed/JS/grep)
u         # Unicode — Unicode-aware matching
A         # Anchored — match must begin at start (PCRE)
U         # Ungreedy — swap default greediness (PCRE)
J         # Allow duplicate group names (PCRE)
n         # Explicit captures only — capture only named groups (.NET)
```

### Engine-Specific Flag Names

```bash
# Python re module:
re.IGNORECASE  / re.I
re.MULTILINE   / re.M
re.DOTALL      / re.S
re.VERBOSE     / re.X
re.UNICODE     / re.U   (default in Python 3 for str patterns)
re.ASCII       / re.A   (force ASCII-only \d \w \s)
re.LOCALE      / re.L   (legacy, use ASCII)

# JavaScript:
/pat/g    global
/pat/i    case-insensitive
/pat/m    multiline
/pat/s    DOTALL  (ES2018+)
/pat/u    unicode (ES2015+)
/pat/y    sticky  (ES2015+)
/pat/d    indices (ES2022+) — exposes match indices in result
/pat/v    Unicode set notation (ES2024+)

# Java:
Pattern.CASE_INSENSITIVE
Pattern.MULTILINE
Pattern.DOTALL
Pattern.UNICODE_CASE
Pattern.UNICODE_CHARACTER_CLASS
Pattern.COMMENTS              (extended mode)
Pattern.LITERAL               (treat pattern as literal)

# .NET:
RegexOptions.IgnoreCase
RegexOptions.Multiline
RegexOptions.Singleline
RegexOptions.IgnorePatternWhitespace
RegexOptions.ExplicitCapture
RegexOptions.Compiled
RegexOptions.RightToLeft
RegexOptions.NonBacktracking  (NEW in .NET 7+ — DFA mode)
```

## Inline Flags

### Pattern-Embedded Modifiers

```bash
(?i)abc               # Case-insensitive — applies to rest of pattern
(?im)abc              # i + m
(?-i)abc              # Turn OFF case-insensitive (PCRE/Java/Python)
(?i:abc)def           # Case-insensitive applies only to "abc"
(?-i:abc)def          # Case-sensitive only for "abc"
(?i:abc(?-i:def))     # Nested scoping
```

### Engine Differences

```bash
# PCRE / Java / Ruby — full inline flag support
# Python re — supported, but (?-i) requires position at start of pattern in old versions
# Go RE2:
(?i)abc               # OK — case-insensitive for rest
(?i:abc)              # OK — scoped
# JavaScript — NO inline flags (must use /pat/i externally)
# .NET — supported via (?imnsx-imnsx) syntax
```

### Ordering Caveat

```bash
# Some engines require flags at the START of the pattern:
(?i)hello world       # OK in Python, PCRE, Java, .NET
hello (?i)world       # In PCRE: applies from "world" onward
                      # In Python (old): error
                      # In Go: applies from that point onward
```

## Replacement Syntax

### Group References in Replacements

```bash
# Most languages:
$1, $2, $3            # Numbered groups (Perl/PCRE/JS/Java/.NET)
\1, \2, \3            # Numbered groups (sed/Python/Ruby/awk)
$&                    # Entire match (Perl/PCRE/JS/Java/.NET)
\&                    # Entire match (sed/awk)
$`                    # Text before match (Perl/JS)
$'                    # Text after match (Perl/JS)

# Named:
$<name>               # PCRE
${name}               # Java, Go, .NET, JavaScript ES2018+
\g<name>              # Python
\k<name>              # Ruby
```

### Case Conversion in Replacement (Perl/sed-GNU/Vim)

```bash
\U...\E               # Uppercase the enclosed text
\L...\E               # Lowercase
\u                    # Uppercase next char only
\l                    # Lowercase next char only

# Example (GNU sed):
sed 's/\([a-z]\+\)/\U\1\E/g'   # Uppercase each lowercase word
sed 's/\([A-Z]\)/_\l\1/g'      # CamelCase to snake_case (rough)
```

### sed Replacement Syntax

```bash
sed 's/old/new/g'         # Global within line
sed 's/old/new/'          # First occurrence per line
sed 's/old/new/2'         # Second occurrence
sed 's|old|new|'          # Different delimiter (any char allowed)
sed 's/\(foo\)/[\1]/'     # Backref \1 (BRE)
sed -E 's/(foo)/[\1]/'    # Same, ERE
sed 's/foo/&!/'           # & = whole match: "foo" → "foo!"
```

### Common Replacement Errors

```bash
# Python — unrecognized escape:
re.sub(r'(\w+)', r'\1!', text)    # OK
re.sub(r'(\w+)', '\1!', text)     # \1 is interpreted as 0x01 char (no raw string!)

# JavaScript — backreferences must be $:
text.replace(/(\w+)/, '\1!')      # WRONG — literal "\1!"
text.replace(/(\w+)/, '$1!')      # CORRECT

# Java — backslashes need double-escaping:
str.replaceAll("(\\w+)", "$1!")   # Pattern is \w+, replacement uses $1
str.replaceAll("(\\w+)", "\\1!")  # ERROR — InvalidEscapeException
```

## Regex Engine Types — Backtracking vs DFA

### Backtracking (NFA)

```bash
# Engines: PCRE, Perl, Java java.util.regex, Python re, JavaScript V8/SpiderMonkey,
#          Ruby, .NET (default), Boost
# Pros:
#   - Supports backreferences, lookaround, atomic groups, all PCRE features
#   - Easy to extend
# Cons:
#   - Worst-case exponential time ("ReDoS" attack)
#   - Performance unpredictable on hostile patterns/input
```

### DFA / Hybrid

```bash
# Engines: Go RE2, Rust regex, Hyperscan, re2c, .NET 7+ NonBacktracking
# Pros:
#   - Linear time guarantee O(n × m) — n input length, m pattern size
#   - Safe against ReDoS attacks
# Cons:
#   - No backreferences (math says they're equivalent to a context-free language)
#   - No lookaround in pure DFA — RE2 supports limited lookaround via tricks
#   - Compile-time can be larger
```

### Picking by Use Case

```bash
# User-supplied pattern OR user-supplied input → use RE2/Rust/Hyperscan
# Trusted patterns from your own code → backtracker is fine
# Hot path / huge input → benchmark both
# Need lookaround/backref → backtracker
```

## Catastrophic Backtracking

### The Classic Bombs

```bash
# Nested quantifiers on same character:
^(a+)+$                       # On "aaaaaaaaaaaaaaa!" → exponential
^(a*)*$                       # Same disease
^(a|a)*$                      # Worse — multiple paths

# Nested optional + repetition:
^(a?){15}a{15}$               # Polynomial-bombs on "aaaaaaaaaaaaaaa"

# Real-world example — email validation:
^([a-zA-Z]+)*$                # Hangs on "aaaaaaaaaaaaaaaa@" (no end match)

# Safe-Browser-Bypass-Spec breaker (real CVE):
^(([a-z])+\.)+[A-Z]([a-z])+$  # PoC for ReDoS on long lowercase input
```

### Diagnostic Patterns

```bash
# Ask yourself:
# 1. Can two quantifiers in this pattern try to consume the same character?
# 2. Is there ambiguity between alternatives? (e.g. a|a)
# 3. Are there nested groups with unbounded repetition?

# Look for patterns matching: (X+)+ or (X*)* or (X|Y)+ where X and Y overlap
```

### Fixes

```bash
# Use possessive quantifiers (PCRE/Java/Ruby):
^(a+)+$         →   ^a++$
^(\w+)*$        →   ^\w*+$

# Use atomic groups (PCRE/Java/.NET):
^(a+)+$         →   ^(?>a+)+$

# Restructure to be unambiguous:
^([a-zA-Z]+)*$  →   ^[a-zA-Z]*$

# Use non-overlapping classes:
^(\w+ )+\w+$    →   ^\w+( \w+)*$

# Bound the input length:
preg_match('/^.{1,100}$/', $input);  # Reject long inputs upfront
```

## JavaScript-Specific

### Literal vs Constructor

```bash
# Literal — preferred for static patterns:
const re = /^\d+$/;

# Constructor — for dynamic patterns:
const re = new RegExp('^\\d+$');     # Note: double-escape backslashes
const re = new RegExp(userInput, 'i');

# Tagged template (cleaner):
String.raw`^\d+$`                    # No double-escaping needed
new RegExp(String.raw`^\d+$`)
```

### Sticky Flag (y)

```bash
const re = /\d+/y;
re.lastIndex = 5;
re.exec("abc12345");        # Anchors at lastIndex=5 — "" (no match)
re.lastIndex = 3;
re.exec("abc12345");        # Matches "12345" starting exactly at index 3
```

### matchAll and Global Flag

```bash
"a1 b2 c3".match(/\w(\d)/g)
// → ['a1', 'b2', 'c3']  (no capture details with g + match)

[..."a1 b2 c3".matchAll(/\w(\d)/g)]
// → [['a1','1'], ['b2','2'], ['c3','3']]

// matchAll requires the /g flag:
"abc".matchAll(/b/);        // TypeError: must use /g
```

### Named Captures and Replacement (ES2018+)

```bash
const re = /(?<year>\d{4})-(?<month>\d{2})-(?<day>\d{2})/;
const m = "2024-04-25".match(re);
m.groups.year;              // "2024"
m.groups.month;             // "04"

"2024-04-25".replace(re, "$<day>/$<month>/$<year>");
// → "25/04/2024"
```

### replaceAll Quirk

```bash
"foofoofoo".replaceAll("foo", "X");           # OK
"foofoofoo".replaceAll(/foo/, "X");           # TypeError
"foofoofoo".replaceAll(/foo/g, "X");          # OK — g flag required
"foofoofoo".replace(/foo/g, "X");             # Alternative
```

### Common Errors (Exact Text)

```bash
# Constructing with bad pattern:
new RegExp('(unclosed')
# SyntaxError: Invalid regular expression: /(unclosed/: Unterminated group

# Bad escape:
/\p{L}/    // Without /u flag
# SyntaxError: Invalid regular expression: /\p{L}/: Invalid escape

# Lookbehind on old engine:
/(?<=foo)bar/   // Pre-ES2018 / Safari < 16.4
# SyntaxError: Invalid regular expression: /(?<=foo)bar/: Invalid group
```

## Python re Module

### Module Functions

```bash
import re

re.match(pat, string)          # Match at START only
re.search(pat, string)          # Find ANYWHERE
re.fullmatch(pat, string)       # Match ENTIRE string
re.findall(pat, string)         # List of all matches (strings or tuples for groups)
re.finditer(pat, string)        # Iterator of Match objects
re.sub(pat, repl, string)       # Substitute all
re.subn(pat, repl, string)      # Substitute, return (new_string, count)
re.split(pat, string)           # Split on pattern
re.compile(pat, flags)          # Pre-compile for reuse
re.escape(literal)              # Escape regex metacharacters in a string
```

### Match Object

```bash
m = re.search(r'(\w+) (\w+)', "Hello World")
m.group(0)                      # "Hello World"
m.group(1)                      # "Hello"
m.group(2)                      # "World"
m.groups()                      # ("Hello", "World")
m.start(), m.end()              # Match span
m.span()                        # (start, end)
m.span(1)                       # Span of group 1

# Named groups:
m = re.search(r'(?P<first>\w+) (?P<last>\w+)', "John Doe")
m.group('first')                # "John"
m.groupdict()                   # {'first': 'John', 'last': 'Doe'}
```

### Raw Strings — ESSENTIAL

```bash
# WRONG:
re.search('\d+', text)          # Works in modern Python but emits DeprecationWarning
re.search('\\d+', text)         # Verbose but legal

# CORRECT — always use raw strings:
re.search(r'\d+', text)
re.search(r'\\', text)          # Single literal backslash
```

### Substitution with Function

```bash
def upper(m):
    return m.group(1).upper()

re.sub(r'(\w+)', upper, "hello world")
# → "HELLO WORLD"

# With group reference:
re.sub(r'(\w+)\s(\w+)', r'\2 \1', "John Doe")
# → "Doe John"
```

### Common Errors (Exact Text)

```bash
re.compile(r'(unclosed')
# re.error: missing ), unterminated subpattern at position 0

re.compile(r'[z-a]')
# re.error: bad character range z-a at position 0

re.compile(r'*foo')
# re.error: nothing to repeat at position 0

re.compile(r'\p{L}')
# re.error: bad escape \p at position 0  (use third-party 'regex' module)
```

## Go RE2 / regexp Package

### Compilation

```bash
import "regexp"

re := regexp.MustCompile(`\d+`)            # Panics on bad pattern
re, err := regexp.Compile(`\d+`)           # Returns error
re := regexp.MustCompilePOSIX(`\d+`)       # POSIX leftmost-longest semantics
```

### Matching

```bash
re.MatchString("abc123")                   # true
re.FindString("abc123def456")              # "123"
re.FindStringIndex("abc123def456")         # [3, 6]
re.FindAllString("abc123def456", -1)       # ["123", "456"]
re.FindStringSubmatch("foo=42")            # entire match + captures
re.FindStringSubmatchIndex(...)            # indices
re.FindAllStringSubmatch(...)              # all matches with captures
```

### Replacement

```bash
re := regexp.MustCompile(`\b(\w+)\b`)
re.ReplaceAllString("hello world", "<$1>")
// → "<hello> <world>"

re.ReplaceAllStringFunc("hello world", strings.ToUpper)
// → "HELLO WORLD"

# Named:
re := regexp.MustCompile(`(?P<word>\w+)`)
re.ReplaceAllString("hello", "[$word]")
```

### Limitations (RE2)

```bash
# NOT supported:
# - Backreferences \1 in pattern (only in replacement)
# - Lookahead (?=) (?!)
# - Lookbehind (?<=) (?<!)
# - Atomic groups (?>)
# - Possessive quantifiers
# - Conditional patterns
# - Unicode property aliases (\p{Latin} works, but limited)

# Linear time guarantee: O(n × m) — safe against ReDoS

# Inline flags supported: i, m, s, U
re := regexp.MustCompile(`(?i)hello`)      # case-insensitive
```

## Rust regex Crate

### Basic Use

```bash
use regex::Regex;

let re = Regex::new(r"\d+").unwrap();
re.is_match("abc123");                     # true
re.find("abc123").unwrap().as_str();       # "123"
re.find_iter("a1 b2").map(|m| m.as_str()); # iter
re.captures("abc123").unwrap()[0];         # "123"
```

### Replacement

```bash
let re = Regex::new(r"(\w+) (\w+)").unwrap();
re.replace("John Doe", "$2 $1");           // "Doe John"
re.replace_all(text, "$1");                // All matches
re.replace_all(text, |caps: &Captures| { caps[1].to_uppercase() });
```

### Named Captures

```bash
let re = Regex::new(r"(?P<year>\d{4})-(?P<month>\d{2})").unwrap();
let caps = re.captures("2024-04").unwrap();
&caps["year"];                             // "2024"
&caps["month"];                            // "04"

# Both syntaxes work:
let re = Regex::new(r"(?<name>\w+)").unwrap();
```

### regex-lite for Smaller Binaries

```bash
# regex-lite — smaller, no Unicode, no SIMD optimizations
# Use when binary size matters and you don't need full Unicode

[dependencies]
regex-lite = "0.1"
```

### Limitations

```bash
# NOT supported (same RE2 family as Go):
# - Backreferences
# - Lookaround (look-ahead/behind)

# Supported features that Go RE2 lacks:
# - Some Unicode classes are richer
# - Better SIMD-accelerated literals matching
```

## Java java.util.regex

### Pattern and Matcher

```bash
import java.util.regex.*;

Pattern p = Pattern.compile("\\d+");
Matcher m = p.matcher("abc123def");
m.find();                    // true — finds "123"
m.group();                   // "123"
m.start(); m.end();          // 3, 6

# matches() vs find() vs lookingAt():
m.matches();                 // true if ENTIRE input matches
m.find();                    // true if ANY substring matches
m.lookingAt();               // true if BEGINNING matches (no anchor)
```

### Flags

```bash
Pattern p = Pattern.compile("hello",
    Pattern.CASE_INSENSITIVE | Pattern.MULTILINE);

# Inline equivalent:
Pattern p = Pattern.compile("(?im)hello");
```

### Replacement

```bash
String result = "John Doe".replaceAll("(\\w+) (\\w+)", "$2 $1");
// → "Doe John"

# With Matcher and StringBuilder for callbacks:
Pattern p = Pattern.compile("\\d+");
Matcher m = p.matcher("a1 b2");
StringBuilder sb = new StringBuilder();
while (m.find()) {
    int n = Integer.parseInt(m.group());
    m.appendReplacement(sb, String.valueOf(n * 2));
}
m.appendTail(sb);
// sb = "a2 b4"
```

### Common Errors (Exact Text)

```bash
Pattern.compile("(unclosed");
// java.util.regex.PatternSyntaxException: Unclosed group near index 9

Pattern.compile("(?<=\\w+)foo");
// PatternSyntaxException: Look-behind group does not have an obvious maximum length
// (Java 13+ relaxed this for some bounded patterns)

Pattern.compile("[z-a]");
// PatternSyntaxException: Illegal character range near index 3
```

## PCRE / pcregrep / grep -P

### PCRE2 Features Beyond PCRE1

```bash
# pcre2 has improvements:
# - \K — reset match start (already in pcre1)
# - (*MARK:NAME) — mark backtracking points
# - (*COMMIT) — commit to current path
# - (*PRUNE) — prune backtracking
# - (*SKIP) — skip alternatives
# - (*FAIL) — force fail
# - Variable-width lookbehind (10.30+)
# - (?(DEFINE)...) — define named subroutines
```

### Backtracking Verbs

```bash
(*PRUNE)             # Reject this path; try next at NEXT input position
(*COMMIT)            # Commit; failure means whole regex fails (no retry from later)
(*SKIP)              # Skip past last (*MARK), try from there
(*FAIL)              # Force failure at this point
(*ACCEPT)            # Accept the match immediately

# Example — match a quoted string but commit after first quote:
"(*COMMIT)[^"]*"
```

### \K Reset

```bash
foo\Kbar             # Match "foobar" but final match span only "bar"
                     # \K resets where the match starts in the output
                     # Like a lookbehind that doesn't have width limit

# Example — extract value from "key=value":
key=\K\S+
```

### UTF and Unicode

```bash
(*UTF8)              # Set UTF-8 mode (deprecated; use /u flag)
(*UCP)               # Unicode property categories
(*UTF)               # Auto-detect

# In pcre2grep:
pcre2grep -u 'pattern' file
```

### Defining Named Subroutines

```bash
# Reusable subroutine pattern:
(?(DEFINE)
  (?<dec_octet>(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d))
)
^(?&dec_octet)\.(?&dec_octet)\.(?&dec_octet)\.(?&dec_octet)$
```

## POSIX BRE

### What's Different

```bash
# Used by: grep (default), sed (default), ed, vi/Nano (default), expr, classic awk?
# (awk uses ERE, not BRE)

# In BRE, these ARE LITERAL by default — escape with \ to make them metacharacters:
( )      → \( \)        Capturing group
{ }      → \{ \}        Quantifier braces
+        → not standard — GNU sed/grep BRE supports \+
?        → not standard — GNU sed/grep BRE supports \?
|        → not standard — GNU only via \|

# These are STILL metacharacters in BRE:
. * [ ] ^ $ \
```

### BRE Examples

```bash
# Capture group:
grep '\(foo\)bar' file

# Quantifier:
grep 'a\{3,5\}' file

# Backreference (THIS works in BRE):
grep '\(a\)\1' file        # Match "aa", "bb", etc.

# Alternation — POSIX BRE has NO alternation
# GNU adds:
grep 'foo\|bar' file       # GNU only

# Plus / question:
grep 'foo\+' file          # GNU only — non-portable
```

### Backreferences in BRE

```bash
# IMPORTANT: BRE supports backreferences. ERE (POSIX) does not.
# So this works in BRE:
grep '\([abc]\)\1' file    # Match "aa", "bb", "cc"

# In strict POSIX ERE this is implementation-defined.
# GNU grep allows it; mawk does not; gawk allows.
```

## POSIX ERE

### What's Different from BRE

```bash
# Used by: grep -E, egrep, awk, sed -E (-r is GNU)
# Metacharacters work WITHOUT escaping:
( ) { } + ? |          # All metacharacters

# Escape to make literal:
\( \) \{ \} \+ \?

# Examples:
grep -E '(foo|bar)' file        # Alternation
grep -E '[a-z]{3,5}' file       # Quantifier
awk '/^[A-Z]+$/' file           # awk uses ERE
```

### BRE → ERE Conversion

```bash
# BRE                       # ERE
'\(foo\)\1'                  '(foo)\1'        # If backrefs supported
'a\{3,5\}'                   'a{3,5}'
'foo\|bar'                   'foo|bar'
'.\+'                        '.+'
```

### POSIX ERE Backreferences

```bash
# The POSIX standard says ERE has NO backreferences.
# Reality:
# - GNU grep -E:  \1 works
# - GNU awk:      \1 works in patterns (not all gawk versions)
# - mawk:         \1 does NOT work
# - BSD grep -E:  \1 may not work

# For portability, prefer BRE \(...\) \1 if you need backreferences in shell scripts
```

## sed Regex

### Default and Extended

```bash
# sed default — POSIX BRE
sed 's/foo/bar/' file
sed 's/\(foo\)/[\1]/' file

# Extended mode — POSIX ERE
sed -E 's/(foo)/[\1]/' file       # POSIX
sed -r 's/(foo)/[\1]/' file       # GNU sed (older)

# In-place edit:
sed -i 's/foo/bar/' file          # GNU
sed -i '' 's/foo/bar/' file       # BSD/macOS (REQUIRES empty extension arg)
sed -i.bak 's/foo/bar/' file      # Both — saves backup as file.bak
```

### Address Ranges

```bash
sed -n '5,10p' file               # Print lines 5-10
sed -n '/start/,/end/p' file      # Print lines between markers
sed '/^$/d' file                  # Delete empty lines
sed '/pattern/d' file             # Delete matching lines
sed '5,$d' file                   # Delete lines 5 to end
sed '2~3d' file                   # GNU — delete every 3rd line starting at 2
```

### Substitute Flags

```bash
sed 's/foo/bar/'                  # First occurrence per line
sed 's/foo/bar/g'                 # All occurrences
sed 's/foo/bar/2'                 # Second occurrence
sed 's/foo/bar/2g'                # 2nd onward
sed 's/foo/bar/I'                 # Case-insensitive (GNU)
sed 's/foo/bar/p'                 # Print modified lines (use with -n)
sed 's|/path|/new|'               # Custom delimiter
```

### Capturing and Special Replacements

```bash
sed 's/\([a-z]\+\)/(\1)/g' file       # BRE — wrap each lowercase word
sed -E 's/([a-z]+)/(\1)/g' file       # ERE
sed 's/foo/&!/' file                  # & = whole match → "foo!"
sed 's/foo/\&/' file                  # Literal & in replacement
sed 's/foo/\\/' file                  # Literal backslash in replacement
```

### Common sed Errors

```bash
sed 's/foo/bar' file
# sed: 1: "s/foo/bar": unterminated substitute pattern (missing /)

sed -E 's/(foo/bar/' file
# sed: -e expression #1, char 11: Unmatched ( or \(

sed -i 's/foo/bar/' file        # On macOS:
# sed: 1: "...": invalid command code .  (BSD sed needs -i '')
```

## awk Regex

### Default Use

```bash
# awk uses POSIX ERE
awk '/pattern/ { print }' file        # Print lines matching pattern
awk '$1 ~ /^foo/' file                # Print where field 1 starts with "foo"
awk '$1 !~ /pat/' file                # Negate match
awk 'BEGIN { RS = "//" } /foo/' file  # Custom record separator
```

### sub and gsub

```bash
awk '{ sub(/old/, "new"); print }' file       # First per line
awk '{ gsub(/old/, "new"); print }' file      # All per line
awk '{ gsub(/old/, "new", $2); print }' file  # On a specific field
```

### match — Capture Group Limit

```bash
# POSIX awk's match() returns position and length, but NOT capture groups
awk '{ if (match($0, /foo([0-9]+)/)) { print RSTART, RLENGTH } }'

# gawk extension — match() can return groups via array:
gawk '{ if (match($0, /foo([0-9]+)/, arr)) { print arr[1] } }'
```

### gawk Extensions

```bash
# Word boundaries:
gawk '/\<foo\>/'                  # Word "foo"
gawk '/[[:alpha:]]+/'             # POSIX classes

# Case-insensitive:
gawk 'BEGIN { IGNORECASE = 1 } /foo/'
```

### awk Pattern Form

```bash
awk '/start/,/end/'               # Range — like sed addresses
awk 'NR > 10 && /pattern/'        # Combined with conditions
```

## grep / egrep / fgrep / rg / ag

### grep

```bash
grep 'pattern' file               # POSIX BRE
grep -E 'pattern' file            # POSIX ERE (or use egrep)
grep -P 'pattern' file            # PCRE — GNU only (Linux), NOT BSD/macOS
grep -F 'literal' file            # Fixed string — fastest, no regex (or fgrep)
```

### Common Flags

```bash
-i        # Case-insensitive
-v        # Invert (non-matching)
-n        # Show line numbers
-c        # Count matches
-l        # List files with matches only
-L        # List files WITHOUT matches
-r        # Recursive
-R        # Recursive, follow symlinks
-w        # Whole word
-x        # Whole line
-o        # Print only matching part
-A 3      # 3 lines after
-B 3      # 3 lines before
-C 3      # 3 lines context (before+after)
-h        # Don't print filenames
-H        # Always print filenames
-q        # Quiet — exit code only
-e PAT    # Multiple patterns: grep -e foo -e bar
-f FILE   # Patterns from file
--include=GLOB
--exclude=GLOB
--exclude-dir=DIR
```

### ripgrep (rg)

```bash
rg 'pattern'                      # Recursive, fast, default
rg -P 'pattern'                   # Use PCRE2 (full PCRE features)
rg -F 'literal'                   # Fixed string
rg -i 'pattern'                   # Case-insensitive
rg -S 'Pattern'                   # Smart case (lower = insensitive, mixed = sensitive)
rg --hidden                       # Search hidden files
rg --no-ignore                    # Ignore .gitignore
rg -tpython 'def\s+\w+'           # File type filter
rg -g '*.md' 'pattern'            # Glob filter
rg -A 3 -B 1 'pattern'            # Context
rg --multiline 'foo.*\n.*bar'     # Match across lines (-U)
```

### Default Engines

```bash
# grep:    POSIX (BRE/ERE), with -P uses PCRE2 (Linux GNU only)
# rg:      Rust regex (RE2-like) by default; PCRE2 via -P
# ag:      PCRE-like (older)
# git grep: Same as grep, with -P for PCRE
# pcregrep: PCRE2 native
```

## Vim Regex

### Magic Levels

```bash
\v       # Very magic — POSIX-like, no escaping
\m       # Magic — Vim default
\M       # No magic — most chars literal
\V       # Very nomagic — only \ escapes

# Examples:
:%s/\v(\w+)\s+\1/&/g          # Very magic — clean syntax
:%s/\(\w\+\)\s\+\1/&/g        # Default magic — escape liberally
```

### Vim-Specific Atoms

```bash
\zs       # Start of match — like \K in PCRE
\ze       # End of match
\@=       # Lookahead (after the atom)
\@!       # Negative lookahead
\@<=      # Lookbehind
\@<!      # Negative lookbehind
\(...\)   # Group (in default magic)
\=        # Same as ? — 0 or 1
\<        # Word start
\>        # Word end
\.        # Literal dot (in default magic, . is metachar)
~         # Last replacement
\&        # Logical AND of two patterns
```

### Substitute and gn

```bash
:%s/old/new/g                     # All on all lines
:%s/old/new/gc                    # Confirm each
:5,10s/old/new/                   # Range
:'<,'>s/old/new/                  # Visual selection
gn                                # "Next match" operator — change next, etc.
cgn                               # Change next match
.                                 # Repeat — combined with gn for refactoring
```

### Vim Search

```bash
/pattern                          # Forward search
?pattern                          # Backward search
n                                 # Next match
N                                 # Previous match
*                                 # Search for word under cursor
:set ic                           # Case-insensitive
:set hls                          # Highlight matches
:noh                              # Clear highlight
```

## Common Patterns

### Email — A Pragmatic Regex

```bash
# RFC 5322 is monstrous (the "official" regex is ~6000 chars).
# Pragmatic:
[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}

# Better — Unicode-aware (PCRE):
[^\s@]+@[^\s@]+\.[^\s@]+

# Best practice: validate with a simple regex, then SEND a verification email.
```

### URL

```bash
# HTTP(S) URL, simple:
https?://[^\s/$.?#][^\s]*

# More thorough, with optional port and path:
https?://(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(?::\d{1,5})?(?:/[^\s]*)?

# Match any URL with scheme:
[a-zA-Z][a-zA-Z0-9+.-]*://[^\s]+
```

### Phone Numbers

```bash
# US — flexible:
(?:\+?1[-.\s]?)?\(?[2-9]\d{2}\)?[-.\s]?\d{3}[-.\s]?\d{4}

# E.164 (international):
^\+[1-9]\d{1,14}$

# UK:
^(?:0|\+?44)[1-9]\d{8,9}$

# IMPORTANT: phone validation is locale-specific. Use libphonenumber for production.
```

### IP Addresses

```bash
# IPv4 — strict (each octet 0-255):
\b(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.){3}(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\b

# IPv4 — loose (any 4 dot-separated 1-3 digit groups):
\b\d{1,3}(?:\.\d{1,3}){3}\b

# IPv6 — full form (no compression):
\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b

# IPv6 — with :: compression (complex):
(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,7}:|...

# IP CIDR:
\b\d{1,3}(?:\.\d{1,3}){3}/\d{1,2}\b
```

### UUIDs

```bash
# UUID v4 (any version):
[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}

# Version-specific (v4 only):
[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}

# Anchored:
^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$
```

### Dates

```bash
# ISO 8601 date (YYYY-MM-DD):
\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])

# ISO 8601 datetime:
\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})

# US date (MM/DD/YYYY):
(0[1-9]|1[0-2])/(0[1-9]|[12]\d|3[01])/\d{4}

# Year-only:
^(?:19|20)\d{2}$
```

### Semantic Versions

```bash
# SemVer 2.0:
^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$

# Pragmatic:
^v?\d+\.\d+\.\d+(?:-[a-zA-Z0-9.-]+)?(?:\+[a-zA-Z0-9.-]+)?$
```

### Hex Colours

```bash
# 3 or 6 digit hex:
^#([0-9a-fA-F]{3}){1,2}$

# 3, 4, 6, or 8 digit (with alpha):
^#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$
```

### Common Validators

```bash
# Slug (URL path):
^[a-z0-9]+(?:-[a-z0-9]+)*$

# Strong password (≥8 chars, upper, lower, digit, special):
^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$

# Username (3-32 chars, letter start, letter/digit/_/-):
^[a-zA-Z][a-zA-Z0-9_-]{2,31}$

# Credit card (no checksum — that's Luhn):
^(?:4\d{12}(?:\d{3})?|5[1-5]\d{14}|3[47]\d{13}|6(?:011|5\d{2})\d{12}|(?:2131|1800)\d{11})$
```

## Anti-Patterns and Better Tools

### Don't Parse HTML/XML with Regex

```bash
# Famous answer: https://stackoverflow.com/a/1732454
# HTML is not a regular language. You'll go mad.

# Instead:
# Python:        BeautifulSoup, lxml, html.parser
# JavaScript:    DOMParser, cheerio
# Go:            golang.org/x/net/html
# Ruby:          Nokogiri
# Java:          Jsoup
# CLI:           pup, htmlq, xmllint --xpath

# Bad:
re.findall(r'<a href="([^"]+)">([^<]+)</a>', html)
# Breaks on: <a href = 'x'>, attribute order, comments, scripts, malformed HTML

# Good:
soup.find_all('a')
```

### Don't Parse JSON with Regex

```bash
# JSON has nesting — regex fundamentally cannot parse balanced structures.
# Use jq, json.loads, JSON.parse, etc.

# Bad:
re.findall(r'"name"\s*:\s*"([^"]*)"', json_str)
# Breaks on: escaped quotes, nested objects, arrays, unicode escapes

# Good:
data = json.loads(json_str)
```

### Don't Parse CSV with Regex

```bash
# CSV with quoted fields containing commas, newlines, escaped quotes — regex can't handle it.

# Use:
# Python:     csv module
# JS:         papaparse, csv-parse
# Go:         encoding/csv
# Shell:      csvkit, miller (mlr), q
```

### Don't Reinvent

```bash
# Use built-in parsers for: JSON, XML, YAML, TOML, INI, URL, query strings, dates, IP addresses.
# Regex is a tool for finding/transforming TEXT — not a substitute for grammar.
```

## Performance — General

### Compile Once, Use Many Times

```bash
# Python:
pat = re.compile(r'\d+')
for line in lines:
    if pat.search(line):
        ...
# Don't:  re.search(r'\d+', line) inside a loop — recompiles each time

# Go:
re := regexp.MustCompile(`\d+`)   # Compile at startup
for _, line := range lines { re.FindString(line) }

# Java:
static final Pattern PAT = Pattern.compile("\\d+");

# JavaScript:
const re = /\d+/;     // Module-level
```

### Anchor When Possible

```bash
# Slow:    \d+\.\d+
# Faster:  ^\d+\.\d+$  (rejects non-matches early)
```

### Prefer Specific Classes Over .

```bash
# Slow:    "(.*)"
# Faster:  "([^"]*)"
# Why:     . matches anything; backtracking has many alternatives.
#          [^"] only matches non-quotes; one decision per character.
```

### Avoid Excessive Alternation

```bash
# Slow:    apple|application|apply
# Faster:  appl(?:e|ication|y)        (factor common prefix)
# Faster:  appl(?:e|ication|y)\b
# Even faster: app(?:le|lication|ly)
```

### Benchmark with Realistic Input

```bash
# Profile pattern + input together:
# - Python:      timeit, cProfile
# - JS:          performance.now()
# - Go:          testing.B
# - regex101.com shows step count
```

## Performance — DFA vs Backtracker

### Linear-Time Engines

```bash
# RE2 (Go, Rust regex): O(n × m) guaranteed
# Hyperscan (Intel):     SIMD-accelerated, scales to many patterns
# .NET 7+ NonBacktracking: opt-in via RegexOptions.NonBacktracking

# Use these when:
# - Patterns come from untrusted users
# - Input is enormous and adversarial
# - You can live without backreferences and complex lookaround
```

### Backtracking Engines

```bash
# PCRE, Perl, Python re, Java, JavaScript, Ruby
# Fast for typical patterns, but:
# - Vulnerable to ReDoS
# - Performance varies wildly with pattern shape
```

### Real Numbers

```bash
# Run "(a+)+$" against "aaa...aaa!" (length n):
#   Backtracker: O(2^n) — minutes on n=30
#   RE2:         O(n)   — microseconds

# Anchored exact match against literal string:
# All engines: O(n) — equal performance
```

### Mitigating ReDoS

```bash
# 1. Use linear-time engines for user input
# 2. Set timeouts on regex execution (.NET, Java, third-party)
# 3. Bound input length: limit to e.g. 1000 chars before applying
# 4. Use atomic groups / possessive quantifiers
# 5. Prefer character classes over alternation
# 6. Static analysis tools: ReDoS-checker, regex-static, safe-regex
```

## Common Error Messages and Fixes

### "Unmatched parenthesis" / "Missing )"

```bash
# Python:
re.error: missing ), unterminated subpattern at position 0

# Java:
PatternSyntaxException: Unclosed group near index 9

# JavaScript:
SyntaxError: Invalid regular expression: /(/: Unterminated group

# Go:
error parsing regexp: missing closing ): `(`

# Fix: balance the parens:
(foo            # Wrong
(foo)           # Right
\(foo           # If you meant a literal paren in PCRE/ERE
```

### "Unmatched brackets" / "Missing ]"

```bash
# Python:
re.error: unterminated character set at position 0

# Java:
PatternSyntaxException: Unclosed character class near index 4

# Fix:
[abc            # Wrong
[abc]           # Right
\[abc\]         # Literal brackets
```

### "Trailing backslash" / "Bad escape"

```bash
# Python:
re.error: bad escape (end of pattern) at position 4

# Java:
PatternSyntaxException: Unexpected internal error near index 4

# JavaScript:
SyntaxError: Invalid regular expression: /\: \ at end of pattern

# Fix:
foo\           # Wrong — trailing backslash
foo\\          # Right — escaped backslash
foo            # Right — no special char
```

### "Nothing to repeat"

```bash
# Python:
re.error: nothing to repeat at position 0
# Triggered by: *, +, ?, {n} with no preceding atom

# Examples:
*foo            # Wrong — nothing before *
?foo            # Wrong
{3}             # Wrong
\*foo           # Right — literal *
.*foo           # Right
```

### "Lookbehind requires fixed-width pattern"

```bash
# Java (pre-13):
PatternSyntaxException: Look-behind group does not have an obvious maximum length

# JavaScript pre-2018:
SyntaxError: Invalid regular expression: lookbehind not supported

# PCRE:
PCRE: lookbehind assertion is not fixed length

# Fix:
(?<=\w+)foo                    # Variable width — fails
(?<=\w{3})foo                  # Fixed — works
(?<=\w{1,3})foo                # PCRE2 10.30+ allows bounded variable width
```

### "Range out of order"

```bash
# Python:
re.error: bad character range z-a at position 0

# Java:
PatternSyntaxException: Illegal character range near index 3

# Fix:
[z-a]           # Wrong
[a-z]           # Right
[az]            # Right — literal a or z
```

### "Invalid escape"

```bash
# JavaScript:
SyntaxError: Invalid regular expression: /\p{L}/: Invalid escape

# Python:
re.error: bad escape \p at position 0

# Fix (JS): use /u flag
/\p{L}/u
# Fix (Python): use third-party 'regex' module, not stdlib 're'
```

## Common Gotchas

### Forgetting Raw Strings in Python

```bash
# Broken — \d is not a Python escape, becomes literal "\d" but other escapes break:
re.search('\\d+', text)         # works but verbose
re.search('\d+', text)          # works, DeprecationWarning
re.search('\b\w+\b', text)      # \b is BACKSPACE in Python strings — silently breaks!

# Fixed:
re.search(r'\b\w+\b', text)     # raw string preserves backslashes
```

### Greedy . in HTML/JSON Snippet

```bash
# Broken:
<.*>            # Matches "<b>foo</b>" (whole thing) — too greedy
"(.*)"          # Matches "key1=\"a\", key2=\"b\"" all at once

# Fixed (lazy):
<.*?>
"(.*?)"

# Better (negated class — no backtracking):
<[^>]*>
"([^"]*)"
```

### Case-Folding on Non-ASCII

```bash
# Broken: assuming \w covers Unicode letters
\w+             # In default mode: only [A-Za-z0-9_] — misses "café"

# Fixed:
[\wÀ-ſ]+        # Add Latin extended
\p{L}+                    # PCRE/Java/.NET — Unicode letters
(?u)\w+                   # Some engines — Unicode word
```

### Backref vs Replacement Confusion

```bash
# In PATTERN, backref is \1:
^(\w+)\s+\1$    # Same word twice

# In REPLACEMENT, syntax depends on language:
# Perl/sed/Python:    \1
# Most others:        $1

# Common bug:
re.sub(r'(\w+)', r'$1!', text)   # Python — replaces with literal "$1!"
# Fix:
re.sub(r'(\w+)', r'\1!', text)
```

### Hyphen at Wrong Place in Class

```bash
# Broken — - between A and Z is a range:
[A-z]           # Includes [, \, ], ^, _, ` — DON'T

# Fixed:
[A-Za-z]        # Letters only
[A-Z][a-z]      # Or as separate ranges
[-A-Z]          # Hyphen at start = literal
[A-Z-]          # Hyphen at end = literal
[A-Z\-a-z]      # Escaped hyphen = literal
```

### Multiline Mode Misunderstanding

```bash
# Broken:
text = "line1\nline2\nline3"
re.search(r'^line2$', text)      # No match — ^ and $ are start/end of string

# Fixed:
re.search(r'^line2$', text, re.MULTILINE)
re.search(r'(?m)^line2$', text)
```

### .NET Regex Options

```bash
# Forgotten flag:
Regex.IsMatch(input, "hello", RegexOptions.IgnoreCase)
# vs
Regex.IsMatch(input, "(?i)hello")
# Both work — but the second one applies to whole pattern

# Performance — Compiled flag:
new Regex(pattern, RegexOptions.Compiled)
# Compiles to IL once — much faster on hot loops
```

### Ruby's Onigmo Quirks

```bash
# Ruby supports POSIX classes AND Unicode AND backrefs:
str =~ /\p{Han}+/      # CJK characters
str.scan(/\w+/)        # Iterate all matches
"abc".gsub(/(\w)/) { |c| c.upcase }   # Block-style replace

# Pattern-as-string vs literal:
re = Regexp.new(user_input)           # Constructor
re = /literal/                        # Literal — preferred
```

### Java replaceAll and Backslashes

```bash
# DOUBLE-trouble — string-then-regex escaping:
"hello".replaceAll("\\.", ",")        # Pattern is \., literal dot
                                      # In source: \\. (escape \ once for string, once for regex)

# Replacement also has double-escaping:
"hello".replaceAll("(.)", "\\$1\\$1") # Each char doubled — $$$$ in source for literal $

# Use Pattern.quote for literal strings:
"hello".replaceAll(Pattern.quote("."), ",")
```

## Testing Tools

### Online Testers

```bash
# regex101.com
# - Supports: PCRE, PCRE2, Python, Go, JavaScript, Java
# - Step-by-step debugger
# - Explanation pane
# - Quick reference
# - Save and share via permalink

# rubular.com
# - Ruby flavour
# - Simpler interface

# debuggex.com
# - Visualizes regex as railroad diagram
# - PCRE, Python, JavaScript

# regexr.com
# - JavaScript flavour
# - Community patterns library
# - Cheatsheet sidebar

# pythex.org
# - Python re module specifically
```

### Local Tools

```bash
# grep --debug 'pattern'              # GNU grep debug
# rg --debug 'pattern'                # ripgrep debug
# python -c "import re; print(re.compile(r'pat').pattern)"
# perl -le 'use re "debug"; "abc" =~ /a(.)/'
```

### Static Analysis

```bash
# safe-regex (npm)         — checks for ReDoS risk
# rxxr2 (academic)         — formal ReDoS checker
# regex-static (Python)    — static analysis
# Hyperscan compiler       — refuses dangerous patterns
```

## Idioms

### Extract All Dates

```bash
\b\d{4}-\d{2}-\d{2}\b                   # YYYY-MM-DD
\b\d{1,2}/\d{1,2}/\d{2,4}\b             # M/D/YY or MM/DD/YYYY
\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},?\s+\d{4}\b
```

### Extract URLs

```bash
\bhttps?://[^\s<>"{}|\\^`[\]]+
\b(?:https?|ftp)://\S+
```

### Strip ANSI Escape Codes

```bash
# Most engines:
\x1b\[[0-9;]*[a-zA-Z]

# More complete (handles all CSI/OSC):
\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])

# Python:
re.sub(r'\x1b\[[0-9;]*m', '', text)

# In shell:
sed 's/\x1b\[[0-9;]*m//g'
```

### Find Leading/Trailing Whitespace

```bash
^\s+|\s+$              # Match either
^[ \t]+|[ \t]+$        # Tabs and spaces only
```

### Validate Hex Colour

```bash
^#(?:[0-9a-fA-F]{3}){1,2}$
^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$
```

### Parse Query String

```bash
# Match each key=value:
([^&=]+)=([^&]*)

# Iterate (Python):
re.findall(r'([^&=]+)=([^&]*)', query)
# But: prefer urllib.parse.parse_qs
```

### Simple INI Parser

```bash
# Section:           ^\[([^\]]+)\]$
# Key=Value:         ^([^=]+?)\s*=\s*(.*)$
# Comment:           ^[;#].*$
# Blank:             ^\s*$

# Combined (one-shot Python):
import re
pat = re.compile(r'''
    ^(?:
        \[(?P<section>[^\]]+)\] |
        (?P<key>[^=;#\s][^=]*?)\s*=\s*(?P<value>.*) |
        (?:[;#].*) |
        \s*
    )$
''', re.VERBOSE)
```

### Find Repeated Words

```bash
\b(\w+)\s+\1\b               # "the the" matches
\b(\w+)\b(?:\s+\1\b)+        # 2+ in a row
```

### Balanced Parentheses (Limited)

```bash
# Regex CANNOT match arbitrary nesting in classical theory.
# But PCRE/Perl/Ruby support recursion:

\((?:[^()]|(?R))*\)          # PCRE — balanced parens
\((?:[^()]|\g<0>)*\)         # Same, named recursion
```

### Match Whole Word Out of List

```bash
\b(?:apple|banana|cherry)\b
```

### Strip Multiple Spaces

```bash
# Replace runs of whitespace with single space:
\s+   →  ' '

# In Python:
re.sub(r'\s+', ' ', text).strip()
```

### Parse Log Line

```bash
# Apache common log:
^(\S+) (\S+) (\S+) \[([^\]]+)\] "(\S+) (\S+) (\S+)" (\d+) (\d+|-)$

# nginx access log:
^(\S+) - (\S+) \[([^\]]+)\] "(\S+) (\S+) (\S+)" (\d+) (\d+|-) "([^"]*)" "([^"]*)"$

# Use named groups for clarity:
^(?P<ip>\S+) - (?P<user>\S+) \[(?P<time>[^\]]+)\] ...
```

## Tips

### General

```bash
# 1. Start simple — add complexity only when necessary
# 2. Test on edge cases: empty string, whitespace-only, very long input
# 3. Anchor (^ and $) when validating; don't anchor when extracting
# 4. Use named captures for readability in long patterns
# 5. Use VERBOSE / x mode + comments for complex patterns
# 6. Keep patterns out of hot loops — compile once
# 7. Document non-obvious patterns with a comment explaining intent
# 8. Watch the dialect — what works in PCRE may not work in JS or RE2
# 9. When in doubt, prefer character classes over .
# 10. Validate input length BEFORE applying complex regex (prevent ReDoS)
```

### Debugging

```bash
# 1. Use regex101.com or similar to see step-by-step matches
# 2. Print intermediate matches:
#    Python: print(re.findall(...))
#    JS: console.log(text.match(...))
# 3. Build up: test the smallest piece, add layers
# 4. Strip flags one at a time to isolate flag-related issues
# 5. Use VERBOSE mode to add comments to long patterns
# 6. Test against your worst-case input — long, malformed, edge values
# 7. Profile pattern step counts — regex101 shows them
```

### Maintenance

```bash
# 1. Add unit tests for every regex in production code
# 2. Document pattern intent + which dialect
# 3. Avoid mega-regexes — split into smaller ones with code glue
# 4. Use named captures so refactoring doesn't shift group numbers
# 5. Quote literals with re.escape / Pattern.quote when interpolating
# 6. Beware of input with unexpected encodings (UTF-16, BOM, etc.)
# 7. Log pattern + input on regex error — saves debugging hours later
```

### Verbose / Extended Mode

```bash
# Most engines support 'x' flag to allow whitespace and comments:
(?x)
^                       # Anchor start
(?P<year>\d{4})         # Capture year
-                       # Separator
(?P<month>\d{2})        # Capture month
-                       # Separator
(?P<day>\d{2})          # Capture day
$                       # Anchor end

# In Python:
re.compile(r'''
    ^
    (?P<year>\d{4})         # Year
    -
    (?P<month>\d{2})        # Month
    -
    (?P<day>\d{2})          # Day
    $
''', re.VERBOSE)

# Whitespace inside [...] is still significant:
[ ]                          # Match space (use \s for any whitespace)
```

### When NOT to Use Regex

```bash
# 1. Parsing balanced/nested structures (XML, HTML, JSON, code) — use a parser
# 2. Validating dates beyond simple format (Feb 30 won't be caught) — use date library
# 3. Matching natural language — use NLP tools
# 4. Locale-sensitive comparison — use ICU or similar
# 5. Templating — use a template engine
# 6. SQL parsing — use a SQL parser
```

## See Also

- bash, polyglot, awk, python, javascript, typescript, ruby, go, rust, java, c

## References

- [regex101.com](https://regex101.com/) -- interactive tester for PCRE, PCRE2, Python, Go, Java, JavaScript flavors
- [regular-expressions.info](https://www.regular-expressions.info/) -- Jan Goyvaerts' canonical regex tutorial and reference
- [PCRE2 Pattern Documentation](https://www.pcre.org/current/doc/html/) -- PCRE2 pattern syntax, API, and verbs
- [Unicode Technical Standard #18](https://www.unicode.org/reports/tr18/) -- Unicode regex requirements (level 1/2/3 conformance)
- [Mastering Regular Expressions, 3rd ed.](https://www.oreilly.com/library/view/mastering-regular-expressions/0596528124/) -- Jeffrey Friedl, the canonical book
- [Regular Expression Matching Can Be Simple And Fast](https://swtch.com/~rsc/regexp/regexp1.html) -- Russ Cox on RE2's linear-time approach
- [POSIX Regular Expressions](https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap09.html) -- POSIX BRE and ERE specification (IEEE 1003.1-2017)
- [RE2 Syntax](https://github.com/google/re2/wiki/Syntax) -- Google RE2 engine reference
- [Go regexp package](https://pkg.go.dev/regexp/syntax) -- Go regex syntax (RE2 family)
- [Rust regex crate](https://docs.rs/regex/latest/regex/) -- Rust regex API and syntax
- [Python re module](https://docs.python.org/3/library/re.html) -- Python regex documentation
- [Python regex (third-party)](https://pypi.org/project/regex/) -- richer features than stdlib re
- [Java java.util.regex](https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/util/regex/Pattern.html) -- Java regex Pattern class
- [.NET Regular Expressions](https://learn.microsoft.com/dotnet/standard/base-types/regular-expressions) -- .NET regex reference
- [MDN Regular Expressions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Regular_expressions) -- JavaScript regex guide
- [ECMA-262 RegExp](https://tc39.es/ecma262/#sec-regexp-regular-expression-objects) -- JavaScript regex spec
- [Ruby Onigmo](https://github.com/k-takata/Onigmo/blob/master/doc/RE) -- Ruby regex engine syntax
- [Vim regex](https://vimhelp.org/pattern.txt.html) -- Vim pattern reference
- [GNU grep manual](https://www.gnu.org/software/grep/manual/grep.html) -- grep BRE/ERE/PCRE flags and usage
- [GNU sed manual](https://www.gnu.org/software/sed/manual/sed.html) -- sed regex and command reference
- [GNU awk manual](https://www.gnu.org/software/gawk/manual/gawk.html) -- gawk regex and pattern usage
- [ripgrep manual](https://github.com/BurntSushi/ripgrep/blob/master/GUIDE.md) -- rg command reference
- [pcre2grep](https://www.pcre.org/current/doc/html/pcre2grep.html) -- PCRE2-native grep
- [Debuggex](https://www.debuggex.com/) -- visual regex debugger with railroad diagrams
- [Regexr](https://regexr.com/) -- JavaScript-flavored online tester
- [Pythex](https://pythex.org/) -- Python-flavored online tester
- [Rubular](https://rubular.com/) -- Ruby regex tester
- [ReDoS Cheat Sheet (OWASP)](https://owasp.org/www-community/attacks/Regular_expression_Denial_of_Service_-_ReDoS) -- regex denial-of-service patterns
- [safe-regex](https://www.npmjs.com/package/safe-regex) -- npm tool to detect catastrophic patterns
- [Hyperscan](https://www.hyperscan.io/) -- Intel high-performance multi-pattern regex
