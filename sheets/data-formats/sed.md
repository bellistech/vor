# sed (Stream Editor)

Line-oriented text transformation tool for substitution, deletion, and insertion.

## Substitute

### Basic substitution

```bash
sed 's/old/new/' file.txt                  # first occurrence per line
sed 's/old/new/g' file.txt                 # all occurrences per line
sed 's/old/new/2' file.txt                 # second occurrence per line
sed 's/old/new/gi' file.txt               # case-insensitive, global
```

### In-place editing

```bash
sed -i 's/old/new/g' file.txt              # Linux (GNU sed)
sed -i '' 's/old/new/g' file.txt           # macOS (BSD sed)
sed -i.bak 's/old/new/g' file.txt         # create backup first
```

### Alternative delimiters

```bash
sed 's|/usr/local|/opt|g' file.txt         # use | when / is in pattern
sed 's#http://#https://#g' file.txt        # use # as delimiter
```

### Capture groups

```bash
sed 's/\(.*\)=\(.*\)/\2=\1/' file.txt     # swap around =
sed -E 's/([a-z]+)=([0-9]+)/\2=\1/' file.txt   # extended regex (-E)
sed -E 's/^(.{20}).*/\1.../' file.txt      # truncate to 20 chars + ellipsis
```

### Using & (matched text)

```bash
sed 's/[0-9]*/(&)/' file.txt               # wrap first number in parens
sed -E 's/[A-Z][a-z]+/"&"/g' file.txt     # quote capitalized words
```

## Delete

### Delete lines

```bash
sed '5d' file.txt                          # delete line 5
sed '1d' file.txt                          # delete first line (header)
sed '$d' file.txt                          # delete last line
sed '3,7d' file.txt                        # delete lines 3-7
sed '/pattern/d' file.txt                  # delete lines matching pattern
sed '/^$/d' file.txt                       # delete empty lines
sed '/^#/d' file.txt                       # delete comment lines
sed '1,/^$/d' file.txt                     # delete from start to first blank line
```

## Insert & Append

### Insert before a line

```bash
sed '3i\New line before line 3' file.txt
sed '/pattern/i\Inserted before match' file.txt
```

### Append after a line

```bash
sed '3a\New line after line 3' file.txt
sed '/pattern/a\Appended after match' file.txt
```

### Change (replace) a line

```bash
sed '3c\Replacement line' file.txt
sed '/old_line/c\new_line' file.txt
```

## Ranges

### Line number ranges

```bash
sed '10,20s/old/new/g' file.txt            # substitute only on lines 10-20
sed '1,5d' file.txt                        # delete first 5 lines
sed '10,$d' file.txt                       # delete from line 10 to end
```

### Pattern ranges

```bash
sed '/START/,/END/d' file.txt              # delete between patterns (inclusive)
sed '/START/,/END/s/foo/bar/g' file.txt    # substitute only between patterns
sed -n '/START/,/END/p' file.txt           # print only between patterns
```

### Negate

```bash
sed '/pattern/!d' file.txt                 # delete lines NOT matching (same as grep)
sed '1,5!s/old/new/g' file.txt            # substitute except lines 1-5
```

## Print

### Print specific lines

```bash
sed -n '5p' file.txt                       # print line 5
sed -n '5,10p' file.txt                    # print lines 5-10
sed -n '/pattern/p' file.txt              # print matching lines (like grep)
sed -n '1p' file.txt                       # first line (like head -1)
sed -n '$p' file.txt                       # last line (like tail -1)
```

### Print with line numbers

```bash
sed -n '=' file.txt                        # print line numbers only
sed '=' file.txt | sed 'N;s/\n/\t/'       # line numbers with content
```

## Hold Space

### Swap pattern and hold space

```bash
sed -n 'x;p' file.txt                     # shift lines by one (empty first line)
```

### Reverse file

```bash
sed -n '1!G;h;$p' file.txt                # reverse line order (like tac)
```

### Join pairs of lines

```bash
sed 'N;s/\n/ /' file.txt                   # join every two lines
```

## Common Patterns

### Remove trailing whitespace

```bash
sed 's/[[:space:]]*$//' file.txt
```

### Remove leading whitespace

```bash
sed 's/^[[:space:]]*//' file.txt
```

### Remove blank lines

```bash
sed '/^$/d' file.txt
sed '/^[[:space:]]*$/d' file.txt           # includes whitespace-only lines
```

### Extract email addresses

```bash
sed -nE 's/.*([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}).*/\1/p' file.txt
```

### Add prefix/suffix to lines

```bash
sed 's/^/PREFIX: /' file.txt               # add prefix
sed 's/$/ :SUFFIX/' file.txt               # add suffix
```

### Replace nth line

```bash
sed '5s/.*/replacement text/' file.txt
```

### Comment/uncomment lines

```bash
sed '/pattern/s/^/# /' file.txt            # comment matching lines
sed 's/^# //' file.txt                     # uncomment all lines
```

### Convert DOS to Unix line endings

```bash
sed 's/\r$//' file.txt
```

### Print line count

```bash
sed -n '$=' file.txt
```

## Multiple Commands

### Chain with -e

```bash
sed -e 's/foo/bar/g' -e 's/baz/qux/g' file.txt
```

### Chain with semicolons

```bash
sed 's/foo/bar/g; s/baz/qux/g' file.txt
```

### Use a sed script file

```bash
sed -f commands.sed file.txt
```

## Regex Reference

```bash
# .     any character
# *     zero or more of previous
# +     one or more (with -E)
# ?     zero or one (with -E)
# ^     start of line
# $     end of line
# [abc] character class
# [^a]  negated class
# \b    word boundary (GNU sed)
# \1    back-reference to capture group
# ()    capture group (with -E, or \(\) without)
```

## Tips

- Use `-i.bak` to create a backup before in-place edits. Verify, then delete the backup.
- macOS ships BSD sed. `-i` requires an argument (use `-i ''` for no backup). Install GNU sed via `brew install gnu-sed` for `gsed`.
- `-E` (or `-r` on GNU) enables extended regex, avoiding excessive backslash escaping of `()`, `+`, `?`.
- `sed -n` suppresses default output. Combine with `p` to print only what you want.
- When the replacement contains `/`, switch delimiters: `s|old|new|g`.
- `&` in the replacement refers to the entire matched text.
- sed processes line by line. For multi-line operations, use `N` to pull the next line into the pattern space.
- For complex transformations, consider `awk` or `perl -pe` instead of fighting with hold space.

## References

- [GNU Sed Manual](https://www.gnu.org/software/sed/manual/) -- complete reference for GNU sed
- [man sed](https://man7.org/linux/man-pages/man1/sed.1.html) -- sed man page
- [POSIX sed Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/sed.html) -- portable sed behavior
- [Sed One-Liners Explained](https://catonmat.net/sed-one-liners-explained-part-one) -- practical recipes with explanations
- [Sed FAQ](https://sed.sourceforge.io/sedfaq.html) -- common questions and solutions
- [Grymoire Sed Tutorial](https://www.grymoire.com/Unix/Sed.html) -- in-depth tutorial with examples
- [GNU Sed Addresses](https://www.gnu.org/software/sed/manual/html_node/Addresses-overview.html) -- line and pattern address syntax
- [regex(7) Man Page](https://man7.org/linux/man-pages/man7/regex.7.html) -- POSIX regex used by sed
