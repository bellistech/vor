# AWK (Pattern Scanning Language)

A line-oriented programming language for extracting and transforming structured text — fields, records, regex patterns, and associative arrays.

## Setup

POSIX awk is the portable subset specified by IEEE Std 1003.1. Three major implementations dominate, plus a couple of minor ones.

```bash
awk --version 2>/dev/null || awk -W version 2>/dev/null
which awk
```

```bash
gawk --version       # GNU awk — most features, default on most Linux distros
mawk -W version      # mawk — fast, smaller feature set, default on Debian/Ubuntu
nawk -V              # original "new awk" — Plan 9 / some BSDs
```

```bash
brew install gawk    # macOS — install GNU awk alongside BSD awk
apt install gawk     # Debian/Ubuntu
dnf install gawk     # Fedora/RHEL
apk add gawk         # Alpine
```

On macOS, `awk` is BSD awk derived from the original Kernighan codebase. It supports POSIX features and a few extensions but lacks gawk-only goodies (FPAT, networking, namespaces, gensub). Install gawk via Homebrew if you need them — invoke as `gawk` to avoid clobbering BSD awk.

On most Linux distros `awk` is symlinked to either gawk (Fedora/RHEL/Arch) or mawk (Debian/Ubuntu). Check:

```bash
readlink -f $(which awk)
ls -l /usr/bin/awk /etc/alternatives/awk 2>/dev/null
```

```bash
update-alternatives --config awk    # Debian/Ubuntu — switch system awk
```

If you want behavior consistent across machines, call gawk explicitly rather than `awk`.

## Anatomy of an AWK Program

An AWK program is a sequence of `pattern { action }` rules. The program runs once per input record (line), trying each rule top-to-bottom; if the pattern matches, the action runs.

```awk
BEGIN { ... }              # runs once before any input is read
pattern { action }         # runs per record where pattern matches
pattern { action }
END   { ... }              # runs once after all input is exhausted
```

```bash
awk '
BEGIN { print "header" }
NR == 1 { print "first line:", $0 }
$1 == "ERROR" { errors++ }
END   { print "total errors:", errors }
' file.txt
```

Comments start with `#` and run to end of line. Statements are separated by `;` or newlines.

```awk
# this is a comment
{ print $1 ; print $2 }    # two statements on one line
{ print $1                  # newline ends the statement too
  print $2 }
```

A bare action `{ ... }` runs on every record. A bare pattern (no action) prints `$0` when it matches.

```bash
awk '/error/' file.txt           # equivalent to /error/ { print }
awk '{ print }' file.txt         # equivalent to 1 { print } — runs on every line
awk '1' file.txt                 # idiom: pattern "1" is always true → print every line
```

## Invocation

```bash
awk 'program' file...                       # inline program, one or more files
awk 'program' < file                        # via stdin
cmd | awk 'program'                         # via pipe
awk -f script.awk file                      # program from file
awk -f a.awk -f b.awk file                  # multiple program files concatenated
awk -F'\t' 'program' file                   # set FS=tab
awk -F: '{ print $1 }' /etc/passwd          # FS=':'
awk -v var=value 'program' file             # set awk var before program runs
awk -v thresh=$N '$1 > thresh' file         # interpolate shell var
```

```bash
gawk -e 'program1' -e 'program2' file       # gawk: multiple inline programs
gawk -E script.awk file                     # gawk: -E like -f but stops option processing (good for shebangs)
gawk -O 'program' file                      # gawk: enable optimizations
gawk -i csv 'program' file                  # gawk: load extension library (CSV here)
gawk -L invalid 'program' file              # gawk: --lint, warn about non-portable usage
gawk --posix 'program' file                 # gawk: strict POSIX mode, disable extensions
gawk --traditional 'program' file           # gawk: original awk compat mode
mawk -W exec script.awk                     # mawk: similar to -f
```

Shebang scripts:

```bash
#!/usr/bin/awk -f
BEGIN { print "running as a script" }
{ print NR, $0 }
```

```bash
#!/usr/bin/env -S gawk -E       # gawk-only; -E swallows args correctly in shebangs
BEGIN { for (i = 1; i < ARGC; i++) print ARGV[i] }
```

`--` ends option processing — useful when filenames start with `-`.

```bash
awk 'program' -- -weirdfile.txt
```

`-v` assignments happen before BEGIN runs. Inline `var=value` arguments between filenames are evaluated when awk reaches that argument:

```bash
awk '{ print kind, $0 }' kind=alpha a.txt kind=beta b.txt
```

## Records and Fields

A record is, by default, one line of input (RS="\n"). Fields are split from each record by FS (default: whitespace).

```awk
$0          # the entire current record
$1, $2, $3  # individual fields
$NF         # last field
$(NF-1)     # second-to-last field
NR          # global record number (1-based, increments across all input files)
NF          # number of fields in the current record
FNR         # record number within the current file (resets per file)
FILENAME    # name of the current input file ("" before first file, "-" for stdin)
```

```bash
awk '{ print NR, NF, $1, $NF }' file.txt    # line#, fieldcount, first, last
awk 'END { print NR }' file.txt             # count lines (like wc -l)
awk '{ print $0 }' file.txt                 # cat
awk 'NR == 1 { print }' file.txt            # head -n 1
awk 'NR > 1' data.csv                       # skip header
awk 'NR % 2 == 0' file.txt                  # even-numbered lines
```

Assigning to a field (or `$0`) reparses the record. Assigning to `$N` for `N > NF` extends the record with empty fields.

```bash
awk '{ $2 = "REDACTED"; print }' file.txt   # forces $0 rebuild using OFS
awk '{ $5 = "fifth"; print NF, $0 }' file   # if NF was 3, NF becomes 5 with $4=""
awk '{ $1 = $1; print }' file.txt           # idiom: rebuild $0 to apply OFS
```

The default FS treats *runs* of whitespace (spaces and tabs) as a single separator and *strips* leading/trailing whitespace from `$0` when computing fields. Setting FS to a single space preserves that behavior. Setting FS to anything else (e.g. `","`) changes splitting to "exactly one separator between fields" — empty fields between consecutive separators are preserved.

```bash
printf 'a,,c\n' | awk -F, '{ print NF, "[" $1 "][" $2 "][" $3 "]" }'
# → 3 [a][][c]
printf '  a  b  \n' | awk '{ print NF, "[" $1 "][" $2 "]" }'
# → 2 [a][b]
```

## Field Separators

```bash
awk -F: '{ print $1 }' /etc/passwd            # single-char FS
awk -F'\t' '{ print $2 }' file.tsv            # tab; quote because shell expands \t differently
awk 'BEGIN { FS = "\t" } { print $2 }' file
awk -F'[,;]' '{ print $1 }' file              # regex FS — comma OR semicolon
awk -F'[[:space:]]+' '{ print $2 }' file      # explicit whitespace regex
awk 'BEGIN { OFS = "|" } { $1 = $1; print }' f.txt   # change output separator
```

```bash
awk -F: -v OFS=, '{ $1 = $1; print $1, $3, $7 }' /etc/passwd  # convert to CSV
awk 'BEGIN { FS = OFS = "\t" } { gsub(/x/, "y", $3); print }' f.tsv
```

gawk extensions for non-separator-based field parsing:

```bash
gawk 'BEGIN { FIELDWIDTHS = "5 3 8" } { print $2 }' fixedwidth.txt
gawk 'BEGIN { FPAT = "([^,]*)|(\"[^\"]*\")" } { print $1 }' quoted.csv
```

`FIELDWIDTHS` makes fields fixed-width (no FS at all). `FPAT` says "match the field" instead of "match the separator" — essential for quoted CSV. POSIX awk supports neither.

## Built-in Variables

Most are set by awk; some you set yourself.

```awk
NR           # number of records read so far
NF           # number of fields in current record
FNR          # record number in current file (resets per file)
FILENAME     # current input file ("" before first record, "-" for stdin)
FS           # input field separator (default " " — special whitespace mode)
OFS          # output field separator (default " ")
RS           # input record separator (default "\n")
ORS          # output record separator (default "\n")
SUBSEP       # multi-dim array key separator (default "\034")
ENVIRON      # array of environment vars: ENVIRON["HOME"]
RSTART       # set by match() — index of first match (0 if none)
RLENGTH      # set by match() — length of match (-1 if none)
CONVFMT      # number→string conversion format (default "%.6g")
OFMT         # output format for print of numbers (default "%.6g")
ARGC, ARGV   # command-line arg count and array
ARGIND       # gawk: index of current file in ARGV
IGNORECASE   # gawk: nonzero → regex/string compares are case-insensitive
FIELDWIDTHS  # gawk: space-separated widths for fixed-width parsing
FPAT         # gawk: regex describing what a field LOOKS like
PROCINFO     # gawk: array of process metadata (uid, pid, version, etc.)
AWK_LIBPATH  # gawk: path for -i and @load extension lookup
```

Examples:

```bash
awk 'BEGIN { for (k in ENVIRON) print k, ENVIRON[k] }' | head
gawk 'BEGIN { print PROCINFO["version"], PROCINFO["pid"] }'
gawk 'BEGIN { IGNORECASE = 1 } /error/' file.txt
awk 'BEGIN { OFMT = "%.2f" } { print $1 + 0 }' nums.txt
```

`OFMT` only affects `print` — `printf` always honors its explicit format string.

## Patterns

A pattern can be:

- A regex literal: `/pat/` — matches if the regex finds anything in `$0`
- An expression: `$3 > 100`, `$1 == "alice"`, `length($0) > 80`
- A range: `pat1, pat2` — true from line matching pat1 through line matching pat2 (inclusive)
- `BEGIN`, `END` — special, run before/after input (not on records)
- gawk: `BEGINFILE`, `ENDFILE` — run before/after each file
- A combined boolean: `pat1 && pat2`, `pat1 || pat2`, `! pat1`, `(pat1)`

```bash
awk '/error/' /var/log/syslog                   # regex pattern
awk '!/comment/' file                           # negated
awk '$3 > 100' data                             # expression
awk '$1 == "alice" && $2 ~ /admin/' data        # combined
awk '/start/,/end/' file                        # range
awk 'NR > 1 && NF >= 3' csv                     # skip header, ensure 3+ fields
awk 'length($0) > 80' file                      # long lines
gawk 'BEGINFILE { print "==", FILENAME, "==" } { print }' a.txt b.txt
```

`/regex/` as a top-level pattern matches against `$0`. To match a regex against another field, use `~` or `!~`:

```bash
awk '$2 ~ /^[0-9]+$/ { print }' file            # field 2 is all digits
awk '$3 !~ /^#/' file                           # field 3 doesn't start with #
```

Range patterns are inclusive on both ends and re-arm: a range can match more than once in a file.

```bash
awk '/^=== START ===$/,/^=== END ===$/' multi-section.txt
```

## Actions

Inside `{ ... }` you can use any of:

```awk
print expr_list           # print, comma-separated → joined with OFS
printf "fmt", args...     # formatted print, no automatic newline
getline                   # read next record (advances NR)
next                      # skip remaining patterns, go to next record
nextfile                  # skip rest of current file, advance to next ARGV file
exit [code]               # terminate, run END, exit with optional status
return expr               # return from a user-defined function
delete arr[k]             # remove element from array
delete arr                # remove all elements (gawk + most modern awks)
if (c) ... else ...
while (c) ...
do ... while (c)
for (init; c; step) ...
for (k in arr) ...
break ; continue
```

```bash
awk '/skip/ { next } { print }' file            # skip, then default-print rest
awk 'NR > 100 { exit }' file                    # bail out after 100 lines
awk '{ delete seen[$1]; seen[$1] = 1 }' file
```

## Print and Printf

`print` is comma-separated and uses `OFS` to join arguments. Concatenated arguments (no commas) are joined directly.

```bash
awk '{ print $1, $2 }' f.txt                    # joined by OFS
awk '{ print $1 $2 }' f.txt                     # concatenated, no separator
awk 'BEGIN { OFS = " | " } { print $1, $2 }' f
awk '{ print "name:", $1 ; print "age:", $2 }' f
awk '{ print "[" $1 "]" }' f                    # explicit brackets
```

`print` automatically appends `ORS` (default newline). To suppress, use `printf`.

`printf` takes a format string and arguments. It does NOT add a newline — you must include `\n`. It does NOT use OFS — you control all spacing.

```awk
printf "%-10s %5d %.2f\n", $1, $2, $3
```

Format specifiers:

```
%s      string
%d, %i  integer (decimal)
%u      unsigned decimal (gawk treats numbers as signed; equivalent to %d)
%x, %X  hex (lower / upper)
%o      octal
%c      character — int → that codepoint; string → first character
%f      float (default 6 decimals)
%e, %E  scientific
%g, %G  shorter of %e or %f
%%      literal percent sign
```

Flags / widths / precision:

```
%-10s   left-align in 10 chars
%10s    right-align in 10 chars
%010d   zero-pad to 10 digits
%+d     always print sign
%.3f    3 decimal places
%*d, %.*f   width/precision from argument (gawk supports this)
```

```bash
awk '{ printf "%-20s %10.2f\n", $1, $2 }' data
awk '{ printf "%05d  %s\n", NR, $0 }' file       # zero-padded line numbers
awk 'BEGIN { printf "%-8s %-8s %-6s\n", "name", "kind", "size" }
     { printf "%-8s %-8s %-6d\n", $1, $2, $3 }' data
awk 'BEGIN { printf "%c\n", 65 }'                # → A
awk 'BEGIN { printf "%x %o %b\n", 255, 255, 255 }'   # %b is gawk-only; %x %o portable
```

Use `sprintf` to capture formatted output as a string:

```bash
awk '{ s = sprintf("[%05d] %s", NR, $0); print s }' file
```

## String Functions

POSIX-standard:

```awk
length(s)               # length of s in chars (default: $0)
length()                # same as length($0)
length(arr)             # gawk: number of elements in arr
substr(s, i)            # substring from index i (1-based) to end
substr(s, i, n)         # substring of length n starting at i
index(s, t)             # 1-based position of t in s, or 0 if absent
match(s, regex)         # sets RSTART, RLENGTH; returns RSTART (0 if no match)
split(s, arr, sep)      # split s into arr[1..n] by sep (string or regex); returns n
split(s, arr)           # uses FS as separator
split(s, arr, sep, seps)# gawk: also fills seps[] with separator strings between fields
sub(regex, repl, s)     # replace FIRST match in s with repl; returns 1/0
sub(regex, repl)        # operates on $0
gsub(regex, repl, s)    # GLOBAL replace; returns count of replacements
gsub(regex, repl)       # operates on $0
sprintf(fmt, args...)   # like printf, but returns the string
toupper(s)              # uppercase
tolower(s)              # lowercase
```

In `repl`, an `&` is replaced by the matched text. Use `\&` for a literal ampersand.

```bash
awk '{ gsub(/[0-9]+/, "[&]"); print }' file        # wrap numbers in brackets
awk '{ sub(/error/, "ERROR"); print }' file
awk '{ print substr($0, 1, 80) }' file             # first 80 chars
awk '{ if (match($0, /[A-Z][a-z]+/)) print substr($0, RSTART, RLENGTH) }' file
awk 'BEGIN { n = split("a:b:c:d", a, ":"); for (i=1;i<=n;i++) print i, a[i] }'
awk '{ print toupper($1), tolower($2) }' file
awk 'BEGIN { print sprintf("%05d", 42) }'          # → 00042
```

gawk-only:

```awk
gensub(regex, repl, how, s)
                        # how="g" replaces all; how=N replaces only Nth match
                        # repl supports backreferences \\1, \\2 ...
                        # returns NEW string (does not mutate s)
patsplit(s, arr, fpat, seps)
                        # split s into arr[] using fpat as field-pattern (like FPAT)
strtonum(s)             # parse "0x1F", "0777", "0b101" — POSIX awk does not
```

```bash
gawk '{ print gensub(/(\w+)@(\w+)/, "\\2/\\1", "g", $0) }' email.txt
gawk 'BEGIN { print strtonum("0xFF"), strtonum("0755"), strtonum("0b1010") }'
```

`sub` and `gsub` MUTATE their target. `gensub` does not. POSIX awk has no `gensub` and no backreferences in `sub`/`gsub` — use gawk if you need them.

## Numeric Functions

```awk
int(x)                  # truncate toward zero
sqrt(x)                 # square root
exp(x)                  # e^x
log(x)                  # natural log
sin(x), cos(x)          # trig (radians)
atan2(y, x)             # arctan(y/x), in correct quadrant
rand()                  # uniform [0, 1)
srand(seed)             # seed RNG; returns previous seed
srand()                 # seed from current time; returns previous seed
```

```bash
awk 'BEGIN { print int(3.7), int(-3.7) }'          # → 3 -3
awk 'BEGIN { print sqrt(2), exp(1), log(2.71828) }'
awk 'BEGIN { srand(); for (i=0;i<5;i++) print rand() }'
awk 'BEGIN { srand(42); print int(rand()*100) }'    # reproducible random 0..99
```

gawk also provides `strtonum()` (above) which respects `0x`, `0`, `0b` prefixes — handy when dealing with hex/octal in input.

## I/O Functions

`getline` is the workhorse for reading additional input. Six forms, with different effects on `NR`, `FNR`, `NF`, and `$0`:

```awk
getline                 # read next record into $0; updates NF, NR, FNR
getline var             # read next record into var; updates NR, FNR (NOT NF, not $0)
getline < "file"        # read next record from file into $0; updates NF (NOT NR/FNR)
getline var < "file"    # read next record from file into var; updates nothing else
"cmd" | getline         # read one line from cmd's stdout into $0; updates NF
"cmd" | getline var     # read one line from cmd into var; updates nothing else
```

Return values: 1 on success, 0 on EOF, -1 on error.

```bash
awk 'BEGIN { while ((getline line < "/etc/hostname") > 0) print line }'
awk 'BEGIN { "date" | getline now; print "time:", now }'
awk '/INCLUDE / { while ((getline l < $2) > 0) print l ; next } { print }' main.txt
```

Always `close()` files and pipes you opened, especially in long-running scripts:

```awk
close("cmd")
close("file")
fflush("file")     # flush buffered writes (gawk: fflush() with no arg flushes all)
```

```bash
awk 'BEGIN { for (i=0;i<3;i++) { print "hi" | "tee /tmp/log" } close("tee /tmp/log") }'
```

`system(cmd)` runs a shell command synchronously and returns its exit status. Output goes wherever the command sends it.

```bash
awk '{ if (system("test -f " $1) == 0) print $1, "exists" }' files.txt
awk 'BEGIN { rc = system("ls /nope 2>/dev/null"); print "rc:", rc }'
```

Output redirection in awk:

```awk
print "x" > "file"              # truncate-and-write (only on first use within program)
print "x" >> "file"             # append
print "x" | "cmd"               # pipe to cmd's stdin
```

Once a file/cmd is opened with `>` or `|`, repeated `>` writes APPEND to that already-open stream; awk does not truncate again on each `>`. Close it to start over.

```bash
awk '{ if ($1 == "ERROR") print > "errors.txt"; else print > "rest.txt" }' input
```

## Time Functions (gawk)

POSIX awk has no time functions — these are gawk extensions.

```awk
systime()                       # current Unix time (seconds since epoch)
mktime("YYYY MM DD HH MM SS")   # build epoch time from broken-down spec
strftime(fmt, t)                # format epoch t per fmt; default fmt is locale; default t is now
strftime()                      # current time, default format
```

```bash
gawk 'BEGIN { print systime(), strftime("%Y-%m-%d %H:%M:%S", systime()) }'
gawk 'BEGIN { t = mktime("2024 01 15 12 00 00"); print strftime("%a %F", t) }'
gawk '{ print strftime("%H:%M:%S", $1), $2 }' epoch_log.txt
```

`mktime()` interprets fields as local time. Add `"UTC"` as the optional second arg in newer gawk for UTC.

## Arrays

All AWK arrays are associative — keys are strings (numeric keys get converted to strings via CONVFMT). There is no "array length" built-in in POSIX awk; gawk supports `length(arr)`.

```awk
arr[k] = v               # assign
arr[k]                   # read; nonexistent keys get auto-created with empty string
                         # (this matters! see below)
if (k in arr) ...        # SAFE existence test — does NOT create the key
delete arr[k]            # remove single element
delete arr               # remove all elements (gawk + mawk; not strict POSIX 2001)
for (k in arr) ...       # iterate, in IMPLEMENTATION-DEFINED ORDER
```

```bash
awk '{ count[$1]++ } END { for (k in count) print count[k], k }' access.log | sort -rn
awk -F, '{ sum[$1] += $2 } END { for (k in sum) print k, sum[k] }' sales.csv
awk '!seen[$0]++' file                  # de-dup keeping order
awk '{ if (!($1 in first)) first[$1] = NR }
     END { for (k in first) print first[k], k }' file
```

The auto-create-on-read behavior is a common bug source:

```bash
# BROKEN — this creates count["alice"] with value "" then tests if !"" (true)
awk '{ if (count[$1] == "") seen_new = 1; count[$1]++ }' file

# FIXED — use `in`
awk '{ if (!($1 in count)) seen_new = 1; count[$1]++ }' file
```

Iteration order is unspecified in POSIX. gawk respects `PROCINFO["sorted_in"]`:

```bash
gawk 'BEGIN { PROCINFO["sorted_in"] = "@ind_str_asc" }
      { count[$1]++ } END { for (k in count) print k, count[k] }' f
```

Sort modes include `@ind_str_asc`, `@ind_num_asc`, `@val_str_desc`, etc.

## Multi-Dimensional Arrays

POSIX awk has no real multi-dim arrays. `arr[i,j]` is sugar for `arr[i SUBSEP j]` — a single-dim array with synthesized keys.

```bash
awk 'BEGIN { SUBSEP = "/" }                     # change for human-readable keys
     { count[$1, $2]++ }
     END { for (k in count) {
              split(k, a, SUBSEP);
              print a[1], a[2], count[k]
           } }' file
```

Test membership of a tuple:

```awk
if ((i, j) in arr) ...
```

gawk supports true arrays-of-arrays:

```awk
gawk '{ data[$1][$2] = $3 }
      END { for (k in data)
              for (k2 in data[k])
                print k, k2, data[k][k2] }' file
```

Mixing the two styles within one variable in gawk causes errors — pick one shape per variable.

## Array Sorting (gawk)

POSIX awk does not sort arrays — pipe to `sort` instead. gawk has `asort` and `asorti`.

```awk
asort(in)                   # sort by VALUE; renumbers keys 1..n IN-PLACE; original keys lost
asort(in, out)              # sort in[] by value; result goes to out[1..n]; in[] untouched
asort(in, out, "@val_num_desc")
asorti(in)                  # sort by KEY; renumbers IN-PLACE
asorti(in, out)             # sort keys, write to out[1..n]
```

```bash
gawk '{ count[$1]++ }
      END { n = asorti(count, sorted_keys);
            for (i=1;i<=n;i++) print sorted_keys[i], count[sorted_keys[i]] }' f
```

For deterministic `for (k in arr)` iteration without copying:

```bash
gawk 'BEGIN { PROCINFO["sorted_in"] = "@val_num_desc" }
      { count[$1]++ }
      END { for (k in count) print k, count[k] }' f
```

## Control Flow

```awk
if (cond) stmt
if (cond) stmt else stmt
if (cond) { ... } else if (cond) { ... } else { ... }

while (cond) stmt
do stmt while (cond)
for (init; cond; step) stmt
for (k in arr) stmt

break       # exit innermost loop
continue    # next iteration of innermost loop
next        # skip remaining patterns, advance to next input record
nextfile    # advance to next ARGV file
exit code   # terminate; runs END unless already in END
return val  # from a user function
```

```bash
awk 'BEGIN { for (i=1; i<=10; i++) print i, i*i }'
awk '{ if ($3 > 100) print $0 }' f
awk '/^---$/ { next } { print }' f
awk 'NR > 1000 { nextfile } { count++ } END { print count }' big*.txt
awk '$2 == "" { exit 1 } { print }' f
```

`exit` followed by `code` becomes the awk process exit status. END still runs; another `exit` inside END skips no further END logic.

## Functions

User-defined functions:

```awk
function name(arg1, arg2,    local1, local2) {
    # by convention, EXTRA "args" after a stretch of whitespace are LOCAL variables.
    # awk has no other way to declare locals.
    ...
    return value
}
```

```bash
awk '
function abs(x) { return x < 0 ? -x : x }
function max(a, b) { return a > b ? a : b }
function sum_array(a,    k, s) { for (k in a) s += a[k]; return s }

{ vals[NR] = $1 + 0 }
END {
    print "max abs:", max(abs(vals[1]), abs(vals[NR]))
    print "sum:", sum_array(vals)
}
' nums.txt
```

Arrays are passed by REFERENCE; scalars by VALUE. Recursion works.

```awk
function fact(n) { return n <= 1 ? 1 : n * fact(n - 1) }
```

Forgetting the local-var spacing convention is a classic bug:

```bash
# BROKEN — k and s are GLOBAL (no extra-spacing separator)
function bad(a, k, s) { for (k in a) s += a[k]; return s }

# FIXED — k and s are local
function good(a,     k, s) { for (k in a) s += a[k]; return s }
```

gawk has indirect function calls via `@`:

```bash
gawk 'function add(a,b){return a+b} function mul(a,b){return a*b}
      BEGIN { f = "add"; print @f(2, 3) }'
```

## Pattern Matching with Regex

AWK uses Extended Regular Expressions (ERE). No `-E` flag needed — that's grep. All standard ERE constructs are supported:

```
.           any single char (except newline)
*           0 or more
+           1 or more
?           0 or 1
|           alternation
^ $         start/end of STRING (not line, when matching multi-line content)
( )         grouping (POSIX awk: no backreferences)
[ ]         character class
[^ ]        negated class
{n}, {n,m}  bounded repetition (gawk supports; some old awks don't)
\           escape — to match a literal regex metachar
```

```bash
awk '/^[A-Z][a-z]+/' file               # starts with capitalized word
awk '$1 ~ /^[0-9]+$/' file              # field 1 is all digits
awk '/foo|bar/' file                    # alternation
awk '/x{3,5}/' file                     # 3 to 5 x's (POSIX awk + gawk + mawk)
```

gawk extensions:

```
\<, \>      word boundaries (start, end of word)
\y          word boundary (either)
\B          non-boundary
[[:alpha:]] POSIX character classes — also work in many other awks
[[:digit:]]
[[:space:]]
[[:upper:]] [[:lower:]] [[:alnum:]] [[:punct:]] [[:xdigit:]] [[:cntrl:]]
```

```bash
gawk '/\<the\>/' file                   # whole word "the"
awk '/[[:alpha:]]+/' file               # POSIX class
```

`~` and `!~` test a string against a regex. The right-hand side can be a regex literal `/foo/` or a string — but a string is interpreted as a regex pattern at runtime ("dynamic regex"):

```bash
awk -v pat='er+or' '$0 ~ pat' file
```

In a dynamic regex, you must double-escape backslashes: `"\\."` for a literal dot.

## Common One-Liners

```bash
awk 'END { print NR }' file                                     # count lines
awk '{ s += $2 } END { print s }' file                          # sum column 2
awk '{ s += $2; n++ } END { print s/n }' file                   # mean of column 2
awk '{ print $NF }' file                                        # last field
awk 'NR == 1 || NR == 10' file                                  # specific lines
awk 'NR % 5 == 0' file                                          # every 5th line
awk '{ for (i=NF;i>0;i--) printf "%s%s", $i, (i>1?OFS:ORS) }' f # reverse fields
awk '{ a[NR] = $0 } END { for (i=NR;i>0;i--) print a[i] }' f    # tac (reverse lines)
awk '!seen[$0]++' file                                          # dedup, preserve order
awk '/start/,/end/' file                                        # block between markers
awk 'NF' file                                                   # drop blank lines
awk 'NF { print NR": "$0 }' file                                # nl-style numbering
awk 'length > 80' file                                          # long lines
awk '{ gsub(/[ \t]+$/,""); print }' file                        # strip trailing whitespace
awk 'BEGIN { for (i=1; i<=10; i++) print i }'                   # seq 1 10
awk '{ print length, $0 }' file | sort -n                       # sort by length
awk '$1=="ERROR"{e++} $1=="WARN"{w++} END{print e, w}' log
awk '{ a[$1]++ } END { for (k in a) if (a[k]>1) print k, a[k] }' f  # only duplicates
```

## Splitting and Joining

`split(s, arr, sep)` returns the number of pieces and fills `arr[1..n]`.

```bash
awk '{ n = split($0, a, ":"); for (i=1;i<=n;i++) print i, a[i] }' file
awk -F: '{ split($7, parts, "/"); print parts[length(parts)] }' /etc/passwd  # basename of shell
```

To join an array back together, write a function — there is no built-in `join` in POSIX awk:

```awk
function join(arr, sep,    s, i) {
    for (i = 1; i in arr; i++) s = s (i==1 ? "" : sep) arr[i]
    return s
}
```

```bash
awk '
function join(arr, sep,    s, i) {
    for (i = 1; i in arr; i++) s = s (i==1 ? "" : sep) arr[i]
    return s
}
{ n = split($0, a, /,/); print join(a, " | ") }
' data.csv
```

For multi-dim, decompose with `split($0, a, SUBSEP)` after iterating with `for (k in arr)`.

## CSV Parsing

POSIX awk has no real CSV support. Splitting on `,` breaks the moment a field contains a quoted comma.

```bash
# BROKEN — fails on: alice,"smith, jr.",30
awk -F, '{ print $2 }' people.csv
```

gawk 5.3+ ships a CSV extension. Earlier gawk: use `FPAT`. Mawk/POSIX: don't write your own — it's quote-escape rules all the way down. Use `csvkit`, `mlr` (Miller), Python, or `xsv`.

```bash
gawk -i csv '{ print $2 }' people.csv                     # gawk 5.3+ CSV extension

gawk 'BEGIN { FPAT = "([^,]*)|(\"[^\"]+\")" }
      { for (i=1; i<=NF; i++) { gsub(/^"|"$/, "", $i); print i, $i } }' people.csv

mlr --csv cat people.csv                                  # better tool for the job
```

For TSV (no embedded tabs in fields), regular awk handles it fine:

```bash
awk -F'\t' -v OFS='\t' '{ print $2, $1 }' file.tsv
```

## JSON

AWK is a poor JSON tool. Hand-written awk JSON parsers exist but are fragile (escaping, nesting, unicode). Use `jq` for JSON processing; use awk for the line-/field-shaped output `jq` produces.

```bash
jq -r '.users[] | [.id, .name, .role] | @tsv' data.json |
    awk -F'\t' '$3 == "admin" { print $1, $2 }'

curl -s api.example.com | jq -r '.[] | "\(.id) \(.name)"' | awk '{ print $1 }'
```

gawk extension libs (`-i json`) exist for advanced cases but jq is more idiomatic and portable.

## Multiple Files

awk happily reads many files in sequence. `NR` keeps counting; `FNR` resets to 1 per file.

```bash
awk '{ print FILENAME, FNR, $0 }' a.txt b.txt c.txt
awk 'FNR == 1 { print "==", FILENAME, "==" } { print }' *.log
awk 'FNR == 1 { count[FILENAME] = 0 } { count[FILENAME]++ }
     END { for (f in count) print f, count[f] }' *.log
```

The `FNR == NR` idiom marks "we are still in the first file" — useful for join-like operations:

```bash
awk 'FNR == NR { keys[$1] = 1; next } $1 in keys' keys.txt data.txt    # filter data by keys
awk 'FNR == NR { name[$1] = $2; next } { print name[$1], $0 }' lookup.tsv records.tsv
```

gawk-only `BEGINFILE` and `ENDFILE` patterns let you take action between files:

```bash
gawk 'BEGINFILE { fcount = 0 } { fcount++ } ENDFILE { print FILENAME, fcount }' *.txt
```

`BEGINFILE` also lets you skip unreadable files gracefully:

```bash
gawk 'BEGINFILE { if (ERRNO) { print "skip:", FILENAME > "/dev/stderr"; nextfile } }
      { print }' *.log
```

POSIX awk has no BEGINFILE — emulate with `FNR == 1`.

`ARGV` / `ARGC` give programmatic access:

```bash
awk 'BEGIN { for (i=1; i<ARGC; i++) print i, ARGV[i] }' a b c
```

You can mutate ARGV inside BEGIN to inject or drop files.

## Pipes In and Out

Run an external command and read its output:

```bash
awk 'BEGIN { "uname -s" | getline os; print "os:", os; close("uname -s") }'
awk '{ ip = $1; ("dig +short -x " ip) | getline name; close("dig +short -x " ip)
       print ip, name }' ips.txt
```

Run an external command and feed it data:

```bash
awk '{ print | "sort -u" } END { close("sort -u") }' file
awk -F, '{ print $1, $3 | "column -t" }' data.csv
```

Multiple writes to the same `| "cmd"` reuse the same pipe — that's why you `close()` at the end.

`system(cmd)` runs synchronously without piping data. It returns exit status, not output.

```bash
awk '{ if (system("ping -c1 -W1 " $1 " > /dev/null 2>&1") != 0) print $1, "down" }' hosts
```

`fflush()` (gawk) forces buffered output to flush — important when piping to tail-like consumers.

```bash
gawk '{ print | "tee /tmp/live"; fflush() }' realtime.log
```

## AWK Variants Compared

POSIX awk (the spec): minimum guaranteed feature set. No `gensub`, no `FPAT`, no time functions, no networking, no arrays-of-arrays, no `length(arr)`, no `delete arr` (whole-array delete).

gawk (GNU awk): the most featureful. Adds `gensub`, `strtonum`, `FPAT`, `FIELDWIDTHS`, `BEGINFILE`/`ENDFILE`, time functions, multi-dim arrays, `PROCINFO`, `IGNORECASE`, `--bignum` (MPFR), namespaces, `@load` extensions, networking via `/inet/` special files, `--debug` interactive debugger, internationalization (`gettext`).

mawk: the speed champion — often 2-5x faster than gawk on simple programs. Keeps to a smaller, mostly POSIX feature set. No `gensub` (though has `sub`/`gsub`), no time functions, no `FPAT`, no networking. Default on Debian/Ubuntu.

BSD awk (a.k.a. "one true awk", `bwk` awk, or `nawk`): the original Kernighan codebase, descended from `nawk`. Lives at github.com/onetrueawk/awk. macOS ships this. Mostly POSIX with a few extensions; less featureful than gawk; usually faster than gawk, slower than mawk.

busybox awk: a minimal awk for embedded systems. Surprisingly capable but has subtle differences from gawk. Default on Alpine, OpenWrt.

If your script must run anywhere, restrict yourself to POSIX awk; if you can require gawk, you get the most power.

```bash
gawk --posix 'program' f          # gawk pretending to be strict POSIX
gawk --traditional 'program' f    # gawk pretending to be original v7 awk
mawk -W posix 'program' f         # mawk POSIX mode
```

## gawk-Specific

Loadable extension libraries with `-i` (or `@load` from inside a program):

```bash
gawk -i inet 'BEGIN { ... read/write tcp ... }'        # network I/O
gawk -i csv 'program' f.csv                            # 5.3+ CSV
gawk -i readfile 'BEGIN { x = readfile("/etc/hosts"); print x }'
gawk -i time 'BEGIN { print gettimeofday() }'
```

`@include` pulls in another awk source file at parse time:

```awk
@include "lib/funcs.awk"
@load "csv"

BEGIN { ... }
```

Namespaces (gawk 5.0+) prevent collisions in shared libraries:

```awk
@namespace "geom"
function area(r) { return 3.14159 * r * r }
```

```bash
gawk '@include "geom.awk"; BEGIN { print geom::area(5) }'
```

Arbitrary-precision math with `--bignum` / `-M` (gawk built with MPFR):

```bash
gawk -M 'BEGIN { PREC = 200; print 2 ** 200 }'
gawk -M 'BEGIN { print 1.0 / 3.0 }'
```

The `--debug` / `-D` flag launches an interactive debugger reminiscent of gdb:

```bash
gawk -D -f script.awk file
```

`gettext`-based i18n: `dcgettext("message")` to translate user-facing strings.

`/inet/PROTO/LPORT/RHOST/RPORT` special files for TCP/UDP I/O:

```bash
gawk 'BEGIN {
    s = "/inet/tcp/0/example.com/80"
    print "GET / HTTP/1.0\r\n\r\n" |& s
    while ((s |& getline line) > 0) print line
    close(s)
}'
```

`|&` is the gawk-specific coprocess operator — bidirectional pipe.

## Common Error Messages

Each entry: exact text (or close to it) → cause → fix.

`awk: syntax error at source line N` (POSIX awk) — most generic message; usually a missing `}`, stray quote, or a regex containing characters the shell ate.

```bash
# BROKEN — shell ate the parens
awk {print $1} file
# FIXED — quote the program
awk '{ print $1 }' file
```

`awk: cmd. line:1: error: cannot mix scalar values and arrays` (gawk) — using a name as both a scalar and an array.

```bash
# BROKEN — count is a scalar, then used as an array
awk 'BEGIN { count = 0 } { count[$1]++ }' file
# FIXED — drop the scalar init or rename
awk '{ count[$1]++ }' file
```

`awk: division by zero attempted` — `$0/$something` where the something is 0 or empty (parses as 0).

```bash
# BROKEN
awk '{ print $1 / $2 }' f                  # blows up if $2 is empty/0
# FIXED — guard
awk '$2 + 0 != 0 { print $1 / $2 }' f
```

`awk: cmd. line:N: ... is not allowed for getline` — usually mixing forms of getline.

```bash
# BROKEN — ambiguous: is this `getline var < file` or `(getline var) < file` ?
awk 'BEGIN { while (getline line < "x" > 0) print line }'
# FIXED — parenthesize
awk 'BEGIN { while ((getline line < "x") > 0) print line }'
```

`awk: command line:N: not enough arguments to function` — calling a user func with fewer args than declared, then trying to USE a missing one as a scalar after using the same name as an array.

```bash
# BROKEN
awk 'function f(a,b) { return a+b } BEGIN { print f(1) }'    # b is "", treated as 0; works
# but mixing array+scalar here is the deeper trap:
awk 'function f(a) { a[1]=1; return a[1] }
     BEGIN { x = f(); print x }'                              # passes uninitialized scalar
# FIXED — pass a real array
awk 'function f(a) { a[1]=1; return a[1] }
     BEGIN { f(arr); print arr[1] }'
```

`awk: warning: regexp constant /.../ looks like a C comment, but is not` — gawk lint catching `/*` inside regex.

```bash
gawk '/\/\*.*\*\//' file        # match /* ... */
```

`awk: NF in END action is sticky` — accessing `$0`, `NF`, `$N` inside END refers to the LAST record processed. There is no "current record" in END.

```bash
awk '{ last = $0 } END { print last }' file    # explicit
```

## Common Gotchas

Variables default silently. An unset variable is `""` as a string and `0` as a number. There is no error.

```bash
# BROKEN — typo, silently returns 0
awk '{ total += $amunt }' f                  # $amunt → $"" → $0 → whole line, then numeric coerce!
# FIXED
awk '{ total += $2 }' f
```

String vs numeric comparison. `==` comparison is numeric if BOTH sides look like numbers; otherwise string.

```bash
# BROKEN — looks like "filter where x is number 10"
awk '$1 == 10' f
# But on the line "010", $1=="010" string-compares to "10" → false. While
# $1 + 0 == 10 → numeric, true. POSIX awk fix:
awk '$1 + 0 == 10' f
# Or force string: awk '$1 == "10"' f
```

Assigning to `$N` rebuilds `$0` with `OFS`, NOT the original input separators.

```bash
# BROKEN — preserves spacing? No — OFS=" " by default, runs of spaces collapse
echo 'a    b    c' | awk '{ $2 = "B"; print }'
# → a B c        (single spaces; original double-space gone)
# FIXED — set OFS to whatever you need, accept that print rebuilds
echo 'a    b    c' | awk 'BEGIN{OFS=" "} { sub(/b/, "B"); print }'
# (use sub on $0 instead of assigning to $2)
```

`OFS` only affects `print`, not `printf`. `printf` uses your format string verbatim.

```bash
# Setting OFS=, has zero effect here:
awk 'BEGIN { OFS = "," } { printf "%s %s\n", $1, $2 }' f
# Use a comma in the format:
awk '{ printf "%s,%s\n", $1, $2 }' f
```

Regex constants vs string-as-regex. `/foo/` is a regex literal; `"foo"` is a string the engine compiles when used as a regex.

```bash
# BROKEN — literal backslash in dynamic regex needs doubling
awk -v p='\.' '$0 ~ p' f             # p ends up as ".", matches any char
# FIXED
awk -v p='\\.' '$0 ~ p' f
# OR — easier — use a regex literal
awk '$0 ~ /\./' f
```

Deleting array entries during `for (k in arr)` iteration. POSIX leaves this implementation-defined; it's safe in gawk and modern mawk, undefined in others.

```bash
# RISKY in old awks
awk '{ a[$1]=1 } END { for (k in a) if (length(k)<2) delete a[k] }' f
# SAFE in any awk — collect keys first
awk '{ a[$1]=1 }
     END { for (k in a) if (length(k)<2) toremove[k]=1
           for (k in toremove) delete a[k] }' f
```

Numbers with leading zeros. Most awks treat `010` as decimal 10, NOT octal 8. Hex `0x10` is parsed as 0 in POSIX awk.

```bash
# BROKEN expectation
awk 'BEGIN { print 010 + 0 }'             # → 10 (POSIX/gawk/mawk)  not 8
awk 'BEGIN { print 0x10 + 0 }'            # → 0 in POSIX/mawk; 16 only in gawk
# FIXED — use strtonum (gawk only) or parse manually
gawk 'BEGIN { print strtonum("0x10") }'   # → 16
```

mawk regex differences. mawk historically didn't support `\<` `\>` word boundaries or POSIX classes consistently. If your regex uses gawk-style features and your script runs on Debian, test on mawk too.

```bash
# Works in gawk, varies in mawk:
mawk '/[[:alpha:]]+/' file
# Portable alternative
mawk '/[A-Za-z]+/' file
```

Empty record at EOF. A trailing newline is normal — awk does NOT see an extra empty record. A file with no trailing newline still produces a final record (most awks).

`getline` and BEGIN. `getline` in BEGIN with no source reads from the FIRST file in ARGV. With `< file` it reads from file. Mixing them is confusing — keep them separate.

`printf` argument count mismatch. Too few args → empty / 0 substituted; too many → extras silently ignored.

```bash
# BROKEN
awk '{ printf "%s %s %s\n", $1 }'  f      # missing args → "$1 (empty) (empty)"
# FIXED — pass them all
awk '{ printf "%s %s %s\n", $1, $2, $3 }' f
```

## Performance

mawk is the fastest portable awk by a meaningful margin — often 2-5x faster than gawk on simple field-extraction tasks. gawk has more features and more aggressive optimization in newer versions, but is rarely faster than mawk on simple work. BSD awk falls in between.

General performance rules:

- Avoid `gsub` / `sub` / `match` in hot loops where a simple field test would do
- Filter early. `NR < 1000` short-circuits before a regex test
- Combine multiple awk pipelines into one program
- Prefer associative-array lookups (`if (k in seen)`) to recomputing
- Use mawk if every byte of throughput matters
- gawk's `--profile` (`-p`) tells you which lines are hot

```bash
mawk '$3 == "ERROR" { print }' big.log     # often fastest
gawk -p -f script.awk big.log              # produces awkprof.out
```

A common observation: AWK is much faster than people expect, and rewriting an AWK pipeline in Perl/Python rarely pays off unless the new version exploits richer data structures. For pure line-shape transforms, awk is usually the right answer.

## Replacing Common Tools

```bash
grep 'pat' f                ↔  awk '/pat/' f
grep -v 'pat' f             ↔  awk '!/pat/' f
grep -c 'pat' f             ↔  awk '/pat/{c++} END{print c}' f
grep -n 'pat' f             ↔  awk '/pat/{print NR ":" $0}' f
head -n 10 f                ↔  awk 'NR<=10' f
head -n 10 f                ↔  awk 'NR==11{exit} 1' f       # faster — bail early
tail -n +5 f                ↔  awk 'NR>=5' f
cut -f3 f                   ↔  awk -F'\t' '{print $3}' f
cut -d, -f1,3 f             ↔  awk -F, -v OFS=, '{print $1,$3}' f
tr -s ' '                   ↔  awk '{$1=$1; print}'         # collapse spaces
wc -l f                     ↔  awk 'END{print NR}' f
sort -u f                   ↔  awk '!seen[$0]++' f          # preserves first-seen order
paste -d, a b               ↔  awk 'NR==FNR{a[FNR]=$0;next} {print a[FNR] "," $0}' a b
```

`sort -n` is its own thing — awk does in-memory sort, but pipe to `sort` for large inputs.

## Output Formatting

Aligning columns:

```bash
printf 'name\tkind\tsize\n' > /tmp/h
awk 'BEGIN { printf "%-12s %-8s %6s\n", "name", "kind", "size" }
     NR>1 { printf "%-12s %-8s %6d\n", $1, $2, $3 }' data.tsv
```

Padded zeros for IDs:

```bash
awk '{ printf "%05d  %s\n", NR, $0 }' f
```

Comma-separated thousands (gawk `printf` flag):

```bash
gawk 'BEGIN { printf "%\047d\n", 1234567 }'      # → 1,234,567 in gawk
```

POSIX awk has no thousands-separator flag — do it yourself:

```bash
awk 'function commify(n,    s,r) {
        s = sprintf("%d", n)
        while (length(s) > 3) { r = "," substr(s, length(s)-2) r; s = substr(s, 1, length(s)-3) }
        return s r
     }
     { print commify($1) }' nums.txt
```

`column -t` does column-alignment using runtime width measurement. AWK can replicate it in a single pass per file:

```bash
awk 'NR == FNR {
        for (i = 1; i <= NF; i++) if (length($i) > w[i]) w[i] = length($i)
        n[FNR] = NF; for (i=1;i<=NF;i++) row[FNR,i] = $i
        next
     }
     END {
        for (r = 1; r <= NR-FNR; r++) {       # weirdness when re-reading
        }
     }' f f                                    # NB: needs file twice — easier: pipe to column -t
```

In practice: `awk '...' f | column -t`.

## Numeric Parsing Quirks

Numeric coercion is `+0` or unary minus or `* 1`. In POSIX awk, prefixes like `0x` and `0b` are NOT honored — the number ends at the first non-digit.

```bash
awk 'BEGIN { print "0x10" + 0 }'            # → 0 (POSIX) or 16 (gawk with strtonum awareness)
awk 'BEGIN { print "10abc" + 0 }'           # → 10
awk 'BEGIN { print "abc10" + 0 }'           # → 0
awk 'BEGIN { print "" + 0 }'                # → 0
awk 'BEGIN { print " 42 " + 0 }'            # → 42 (leading/trailing space tolerated)
```

Force coercion (numeric):

```bash
awk '{ x = $1 + 0; if (x > 100) print }' f
```

Force coercion (string) — concat with `""`:

```bash
awk '{ s = $1 ""; print length(s) }' f
```

`CONVFMT` controls how numbers are stringified for concat etc.; `OFMT` controls `print` output formatting of numbers. Default for both is `"%.6g"`.

```bash
awk 'BEGIN { CONVFMT = "%.2f"; x = 1/3; s = x ""; print s }'
awk 'BEGIN { OFMT = "%.2f"; print 1/3 }'                          # → 0.33
awk 'BEGIN { OFMT = "%.2f"; printf "%g\n", 1/3 }'                 # → 0.333333 (printf ignores OFMT)
```

`strtonum()` (gawk) for hex/oct/binary:

```bash
gawk 'BEGIN { print strtonum("0xFF"), strtonum("0755"), strtonum("0b1010") }'
# → 255 493 10
```

## AWK as a Calculator

```bash
awk 'BEGIN { print 2 + 3 * 4 }'                   # → 14
awk 'BEGIN { print (2 + 3) * 4 }'                 # → 20
awk 'BEGIN { print 2 ** 10 }'                     # → 1024  (^ also works in gawk)
awk 'BEGIN { print sqrt(2) }'
awk 'BEGIN { print log(2) }'
awk 'BEGIN { for (i=1;i<=5;i++) print i, i*i, i**3 }'
```

Pipe data through:

```bash
seq 1 100 | awk '{ s += $1 } END { print s }'                       # → 5050
seq 1 100 | awk '{ s += $1; n++ } END { print s/n }'                # → 50.5
seq 1 100 | awk '{ if ($1>m) m=$1; if (NR==1||$1<n) n=$1 } END { print n, m }'
```

Stats over a column:

```bash
awk '{
    n++
    s += $1
    s2 += $1 * $1
    if (NR == 1 || $1 < min) min = $1
    if ($1 > max) max = $1
}
END {
    mean = s / n
    var  = s2/n - mean*mean
    print "n=" n, "min=" min, "max=" max, "mean=" mean, "stddev=" sqrt(var)
}' nums.txt
```

Histogram bucket counts:

```bash
awk 'BEGIN { bw = 10 }
     { b = int($1/bw)*bw; hist[b]++ }
     END { for (b in hist) print b "-" b+bw-1, hist[b] }' nums.txt | sort -n
```

## AWK with Shell Composition

Heredoc form keeps shell variables out of the program text:

```bash
awk -v threshold="$THRESH" -v label="$NAME" '
$2 + 0 > threshold { print label, $0 }
' data.txt
```

Single quotes around the program prevent `$1` from being expanded by the shell. To embed a single quote inside the program, end the quoting and escape:

```bash
awk '{ print "it'"'"'s fine" }' f
awk "{ print \"shell substitutes \$HOME = $HOME\" }" f      # double-quoted form (rare)
```

Awk script in a file with `-f`:

```bash
cat > /tmp/p.awk <<'AWK'
BEGIN { OFS = "\t" }
{ count[$1]++ }
END { for (k in count) print k, count[k] }
AWK
awk -f /tmp/p.awk file
```

gawk allows mixing `-f` and `-e` and inline programs — useful when you want a library plus a one-off filter:

```bash
gawk -f lib.awk -e 'BEGIN { print library_func() }' f
```

Pure BEGIN-only programs are useful as "awk scripts that don't read input":

```bash
awk 'BEGIN { for (i=1;i<=10;i++) print i }' </dev/null
```

In Makefiles, escape `$` as `$$`:

```makefile
sum:
\tawk '{ s += $$1 } END { print s }' nums.txt
```

## Idioms

```bash
awk '!seen[$0]++'                          # dedup, preserve order
awk '!seen[$1]++'                          # dedup by first field
awk 'NR==FNR{a[$1]=1; next} $1 in a'       # filter file2 by keys in file1
awk 'NR==FNR{a[$1]=$2; next} {print $0, a[$1]}' lookup data    # join/lookup
awk '{ tmp=$1; $1=$2; $2=tmp; print }'     # swap fields 1 and 2
awk '/start/,/end/'                        # block between markers, inclusive
awk '/start/{p=1; next} /end/{p=0; next} p'   # block between markers, exclusive
awk 'BEGIN{c=0} /pat/{c++} END{print c}'   # count matches
awk 'NR>1'                                 # skip header
awk 'NR==1{next} {print}'                  # also skip header
awk '{$1=$1; print}'                       # rebuild $0 with OFS
awk 'NF'                                   # drop blank lines
awk 'NF{print}'                            # same, explicit
awk '/./'                                  # drop blank lines (regex form)
awk 'length > 0'                           # drop truly empty lines
awk 'END{print NR}'                        # count lines
awk '{print NR, $0}'                       # number lines (no padding)
awk 'BEGIN{getline x<"/etc/hostname"; print x}'       # read one file in BEGIN
awk 'NR==FNR{n[$1]++; next} ($1 in n) && n[$1]==1' data data    # uniq by field
awk '{a[NR]=$0} END{for(i=NR;i>=1;i--) print a[i]}'   # tac
```

## Debugging

Print intermediate values:

```bash
awk '{ print "DBG nr="NR" nf="NF" $1="$1 > "/dev/stderr"; ... }' f
```

gawk has a real interactive debugger (`-D`, `--debug`):

```bash
gawk -D -f script.awk file
# (gawk) break main:5
# (gawk) run
# (gawk) print var
# (gawk) next
```

Lint warnings catch many bugs:

```bash
gawk --lint -f script.awk f                   # warn on iffy constructs
gawk --lint=invalid -f script.awk f           # warn only when output would be invalid
gawk --lint-old -f script.awk f               # warn on non-original-awk constructs
```

Dump variables at exit:

```bash
gawk --dump-variables=/tmp/vars.txt -f script.awk f
```

Dump the parsed program:

```bash
gawk -p/tmp/profile.txt -f script.awk f       # then read /tmp/profile.txt
```

Print a stack trace from awk-level functions: gawk's `--debug` lets you `backtrace`. POSIX awk: there's no built-in trace; instrument with `print "in", FUNCNAME` lines manually.

For mysterious "syntax error" messages:

- Comment out half the program until it parses (binary search)
- Make sure regex doesn't contain unescaped `/` inside `/.../`
- Watch for stray smart quotes if the program was pasted from a doc

## Tools

- gawk — gnu.org/software/gawk/, install: `apt install gawk`, `dnf install gawk`, `brew install gawk`
- mawk — invisible-island.net/mawk/, install: `apt install mawk`, `brew install mawk`
- bwk awk / one-true-awk — github.com/onetrueawk/awk, the historical reference impl, ships on macOS as `/usr/bin/awk`
- busybox awk — included in busybox, default on Alpine and OpenWrt
- nawk — variants on some BSDs
- gawkextlib — github.com/gawkextlib, extra extension libraries (XML, JSON, MPFR, etc.)

Helper / adjacent tools:

- mlr (Miller) — gawk-on-steroids for tabular data; understands CSV/TSV/JSON/PPRINT
- xsv — Rust CSV swiss-army knife
- jq — JSON processor; pair with awk for line-oriented output
- frawk — Rust-implemented awk dialect, blazingly fast
- dgsh — directed-graph shell where awk fits naturally

```bash
brew install gawk mawk miller jq                      # macOS dev environment
apt install gawk mawk miller jq                       # Debian/Ubuntu
```

## Tips

- Default-print the line with bare `1` at end of program — shorter than `{print}`.
- `{$1=$1}` reformats a record using `OFS`; useful before `print`.
- For "skip first N lines": `awk -v n=3 'NR>n'`.
- For "print last N lines": buffer in a circular array — `awk -v n=10 '{a[NR%n]=$0} END{for(i=NR-n+1;i<=NR;i++) print a[i%n]}'`. Or just use `tail -n10`.
- Two-file join: `awk 'NR==FNR{a[$1]=$0; next} $1 in a{print a[$1], $0}' f1 f2`.
- Inside awk, file paths are relative to AWK's cwd, not the script's location.
- `printf` does NOT add `\n` automatically — easy gotcha vs `print`.
- Avoid setting `RS = ""` (paragraph mode) if you want predictable behavior across awks — gawk and mawk differ on edge cases.
- For large inputs, prefer mawk for speed and gawk for features. Don't use `cat file | awk ...` — pass `file` as an argument so `FILENAME` is set.
- Quote your awk programs with single quotes to avoid `$` shell-expansion. Double quotes have their place but are noisier.
- `awk` sees `\t` as tab in regex constants and string literals. In single-quoted shell args, no further escaping needed.
- A solitary `1` (or any nonzero value) at the end of a program is a "true" pattern with no action — awk's default action is `{print}`. So `awk '... ; 1' f` means "do stuff, then print every line."
- `getline` increments NR and FNR (in some forms) — beware in counting loops.
- For arithmetic where precision matters, gawk's `--bignum` (`-M`) gives MPFR; otherwise expect IEEE 754 double.
- Printing arrays: there's no built-in. Loop with `for (k in arr) print k, arr[k]`.
- To pass shell array elements to awk, expand into `-v` flags: `awk -v a="$a" -v b="$b" '...'`.

## See Also

- bash, zsh, regex, sed, grep, sql, json

## References

- [POSIX awk specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html) — the portable spec
- [GNU AWK User's Guide (gawk manual)](https://www.gnu.org/software/gawk/manual/) — definitive gawk reference
- [GNU AWK Built-in Functions](https://www.gnu.org/software/gawk/manual/html_node/Built_002din.html)
- [GAWK Loadable Extensions](https://www.gnu.org/software/gawk/manual/html_node/Extension-Samples.html)
- [The AWK Programming Language (Aho, Weinberger, Kernighan)](https://awk.dev/) — the canonical book, 2nd edition
- [mawk home page](https://invisible-island.net/mawk/mawk.html) — the fast minimal awk
- [one-true-awk (bwk awk)](https://github.com/onetrueawk/awk) — the historical reference implementation
- [man gawk](https://man7.org/linux/man-pages/man1/gawk.1.html)
- [AWK One-Liners Explained (catonmat)](https://catonmat.net/awk-one-liners-explained-part-one) — practical recipes
- [gawkextlib](https://sourceforge.net/projects/gawkextlib/) — XML, JSON, MPFR, PostgreSQL extensions for gawk
- [Miller (mlr)](https://miller.readthedocs.io/) — awk for tabular data, CSV/TSV/JSON-aware
- [frawk](https://github.com/ezrosent/frawk) — Rust-implemented awk dialect, fast
