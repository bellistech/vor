# Regex (Regular Expressions)

> Pattern matching language for searching, validating, and transforming text.

## Character Classes

### Basic Classes

```regex
.           # Any character except newline
\d          # Digit [0-9]                    (PCRE/shorthand)
\D          # Non-digit [^0-9]
\w          # Word character [a-zA-Z0-9_]
\W          # Non-word character
\s          # Whitespace [ \t\n\r\f\v]
\S          # Non-whitespace
\b          # Word boundary
\B          # Non-word boundary
```

### Custom Classes

```regex
[abc]       # Match a, b, or c
[^abc]      # Match anything except a, b, c
[a-z]       # Lowercase letter
[A-Z]       # Uppercase letter
[0-9]       # Digit (same as \d)
[a-zA-Z]    # Any letter
[a-zA-Z0-9] # Alphanumeric
[-.]        # Literal hyphen and dot (hyphen first or last)
```

### POSIX Classes (BRE/ERE)

```regex
[:alnum:]   # Alphanumeric         [a-zA-Z0-9]
[:alpha:]   # Alphabetic            [a-zA-Z]
[:digit:]   # Digits                [0-9]
[:lower:]   # Lowercase             [a-z]
[:upper:]   # Uppercase             [A-Z]
[:space:]   # Whitespace            [ \t\n\r\f\v]
[:blank:]   # Space and tab         [ \t]
[:punct:]   # Punctuation
[:print:]   # Printable characters
[:graph:]   # Printable (excluding space)
[:cntrl:]   # Control characters
[:xdigit:]  # Hex digits            [0-9a-fA-F]

# Usage in bracket expressions: [[:alpha:]]
```

## Quantifiers

### Greedy (Match as Much as Possible)

```regex
*           # 0 or more
+           # 1 or more
?           # 0 or 1
{n}         # Exactly n
{n,}        # n or more
{n,m}       # Between n and m (inclusive)
```

### Non-Greedy / Lazy (Match as Little as Possible)

```regex
*?          # 0 or more (lazy)
+?          # 1 or more (lazy)
??          # 0 or 1 (lazy)
{n,m}?      # Between n and m (lazy)

# Example:
# <.*>  matches  <b>bold</b>        (entire string)
# <.*?> matches  <b>                (first tag only)
```

### Possessive (No Backtracking — PCRE Only)

```regex
*+          # 0 or more (possessive)
++          # 1 or more (possessive)
?+          # 0 or 1 (possessive)
```

## Anchors

```regex
^           # Start of string (or line in multiline mode)
$           # End of string (or line in multiline mode)
\A          # Start of string (absolute, ignores multiline)
\Z          # End of string (before final newline)
\z          # End of string (absolute)
```

## Groups and Backreferences

### Capturing Groups

```regex
(abc)       # Capture group 1
(a)(b)(c)   # Groups 1, 2, 3
\1          # Backreference to group 1

# Example: find repeated words
\b(\w+)\s+\1\b
```

### Named Groups (PCRE / Python / .NET)

```regex
(?P<name>pattern)    # Python syntax
(?<name>pattern)     # .NET / PCRE7+ syntax

# Backreference
(?P=name)            # Python
\k<name>             # .NET / PCRE
```

### Non-Capturing Groups

```regex
(?:abc)     # Group without capturing
(?:a|b|c)   # Alternation without capture
```

## Lookahead and Lookbehind

### Lookahead

```regex
(?=pattern)   # Positive lookahead — assert what follows
(?!pattern)   # Negative lookahead — assert what does NOT follow

# Example: match digits followed by "px"
\d+(?=px)     # Matches "12" in "12px" but not in "12em"

# Example: match word NOT followed by "ing"
\b\w+(?!ing)\b
```

### Lookbehind

```regex
(?<=pattern)  # Positive lookbehind — assert what precedes
(?<!pattern)  # Negative lookbehind — assert what does NOT precede

# Example: match digits preceded by "$"
(?<=\$)\d+    # Matches "50" in "$50" but not in "50"

# Example: match word NOT preceded by "un"
(?<!un)\w+
```

## Alternation and Conditional

```regex
a|b         # Match a or b
(cat|dog)   # Match "cat" or "dog"
```

## BRE vs ERE vs PCRE

### POSIX BRE (Basic Regular Expressions)

```bash
# Used by: grep, sed (default)
# Must escape: \( \) \{ \} \+ \? \|
grep 'abc\(def\)' file          # Capturing group
grep 'ab\{2,4\}' file           # Quantifier
```

### POSIX ERE (Extended Regular Expressions)

```bash
# Used by: grep -E (egrep), sed -E, awk
# Metacharacters are literal unless escaped
grep -E 'abc(def)' file         # Capturing group
grep -E 'ab{2,4}' file          # Quantifier
grep -E 'cat|dog' file          # Alternation
```

### PCRE (Perl-Compatible Regular Expressions)

```bash
# Used by: grep -P, most programming languages
# Supports: lookahead, lookbehind, named groups, possessive quantifiers
grep -P '(?<=\$)\d+' file       # Lookbehind
grep -P '\d{3}-\d{4}' file      # Standard PCRE
```

## Flags / Modifiers

```regex
i           # Case-insensitive
m           # Multiline (^ and $ match line boundaries)
s           # Dotall (. matches newline)
x           # Extended (ignore whitespace, allow comments)
g           # Global (all matches, not just first)
u           # Unicode support

# Inline flags (PCRE)
(?i)pattern       # Case-insensitive for this pattern
(?im)pattern      # Multiline + case-insensitive
(?-i)pattern      # Turn off case-insensitive
```

## Common Patterns

### Email (Simplified)

```regex
[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}
```

### IPv4 Address

```regex
\b(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\b
```

### IPv6 Address (Simplified)

```regex
(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}
```

### URL

```regex
https?://[^\s/$.?#].[^\s]*
```

### Date (YYYY-MM-DD)

```regex
\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])
```

### Phone (US)

```regex
(?:\+?1[-.\s]?)?\(?[2-9]\d{2}\)?[-.\s]?\d{3}[-.\s]?\d{4}
```

### Hex Color

```regex
#(?:[0-9a-fA-F]{3}){1,2}\b
```

### UUID

```regex
[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}
```

### Password Strength (Min 8 chars, upper, lower, digit, special)

```regex
^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$
```

### Blank Lines

```regex
^\s*$
```

### Strip HTML Tags

```regex
<[^>]+>
```

## Tool-Specific Usage

### grep

```bash
grep -E 'pattern' file             # ERE
grep -P 'pattern' file             # PCRE
grep -oP '(?<=key=)\S+' file      # Extract values
```

### sed

```bash
sed 's/old/new/g' file             # BRE
sed -E 's/(group)/\1/g' file      # ERE
```

### awk

```bash
awk '/pattern/ {print $0}' file    # ERE by default
```

## Tips

- Prefer non-greedy quantifiers (`*?`, `+?`) when matching delimited content like HTML tags or quoted strings.
- Use non-capturing groups `(?:...)` when you do not need the matched text, for better performance.
- Lookbehind in PCRE must be fixed-length; use `\K` as an alternative to reset the match start.
- Anchor patterns with `^` and `$` to avoid partial matches during validation.
- POSIX classes like `[:alpha:]` are locale-aware; `\w` and `[a-z]` are not.
- Test regex interactively at regex101.com (supports PCRE, Python, Go, Java).

## See Also

- sed, awk, grep, jq, python, javascript

## References

- [Regular-Expressions.info](https://www.regular-expressions.info/) -- comprehensive tutorial and reference
- [PCRE2 Specification](https://www.pcre.org/current/doc/html/) -- PCRE2 pattern syntax and API
- [regex(7) Man Page](https://man7.org/linux/man-pages/man7/regex.7.html) -- POSIX regex overview
- [regex101 -- Online Tester](https://regex101.com/) -- interactive tester with PCRE, Python, Go, Java flavors
- [POSIX Regular Expressions](https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap09.html) -- BRE and ERE specification
- [RE2 Syntax](https://github.com/google/re2/wiki/Syntax) -- Google RE2 engine (used in Go, Rust regex crate)
- [MDN Regular Expressions Guide](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Regular_expressions) -- JavaScript regex reference
- [Python re Module](https://docs.python.org/3/library/re.html) -- Python regex documentation
- [Debuggex](https://www.debuggex.com/) -- visual regex debugger with railroad diagrams
- [man grep](https://man7.org/linux/man-pages/man1/grep.1.html) -- grep regex usage and flags
