# awk (Text Processing Language)

Pattern-scanning and processing language for extracting and transforming columnar text data.

## Basic Usage

### Print specific fields

```bash
awk '{print $1}' file.txt                  # first field
awk '{print $1, $3}' file.txt              # first and third fields
awk '{print $NF}' file.txt                 # last field
awk '{print $(NF-1)}' file.txt             # second to last
```

### Print entire line

```bash
awk '{print $0}' file.txt                  # same as cat
awk '{print}' file.txt                     # shorthand
```

### Pipe input

```bash
ps aux | awk '{print $1, $2, $11}'         # user, pid, command
df -h | awk '{print $1, $5}'              # filesystem, use%
```

## Field Separators

### Input separator (-F)

```bash
awk -F: '{print $1, $3}' /etc/passwd       # colon-separated
awk -F, '{print $1, $2}' data.csv          # comma-separated
awk -F'\t' '{print $1}' data.tsv           # tab-separated
awk -F'[,;]' '{print $1}' data.txt         # regex separator
```

### Output separator (OFS)

```bash
awk -F: -v OFS=',' '{print $1, $3, $7}' /etc/passwd    # CSV output
awk 'BEGIN{OFS="\t"} {print $1, $2}' file.txt           # tab output
```

## Patterns

### Match lines

```bash
awk '/error/' /var/log/syslog              # lines containing "error"
awk '/^root/' /etc/passwd                  # lines starting with "root"
awk '!/comment/' file.txt                  # lines NOT matching
awk '/start/,/end/' file.txt              # range (from start to end)
```

### Condition-based

```bash
awk '$3 > 100' data.txt                    # third field > 100
awk '$1 == "alice"' data.txt               # exact match
awk 'length > 80' file.txt                 # lines longer than 80 chars
awk 'NF > 3' file.txt                      # lines with more than 3 fields
awk 'NR > 1' data.csv                      # skip header line
awk 'NR >= 10 && NR <= 20' file.txt        # lines 10-20
```

## BEGIN / END

### Header and footer

```bash
awk 'BEGIN{print "Name\tScore"} {print $1, $2} END{print "---done---"}' data.txt
```

### Set variables in BEGIN

```bash
awk 'BEGIN{FS=":"; OFS=","} {print $1, $3}' /etc/passwd
```

### Summary at end

```bash
awk '{sum += $2} END{print "Total:", sum}' data.txt
awk '{sum += $2; count++} END{print "Average:", sum/count}' data.txt
```

## Built-in Variables

```bash
# NR     current record (line) number (across all files)
# FNR    record number in current file
# NF     number of fields in current record
# FS     input field separator (default: whitespace)
# OFS    output field separator (default: space)
# RS     input record separator (default: newline)
# ORS    output record separator (default: newline)
# FILENAME  current input filename
# $0     entire current line
# $1..$n individual fields
```

### Using NR and NF

```bash
awk '{print NR, $0}' file.txt             # add line numbers
awk 'END{print NR}' file.txt              # count lines
awk '{print NF}' file.txt                 # fields per line
awk 'NR==1{print}' file.txt              # first line only
awk 'NR%2==0' file.txt                   # even lines
```

## Arrays

### Associative arrays

```bash
awk '{count[$1]++} END{for (k in count) print k, count[k]}' access.log
```

### Word frequency

```bash
awk '{for (i=1; i<=NF; i++) freq[$i]++} END{for (w in freq) print freq[w], w}' file.txt | sort -rn
```

### Group and sum

```bash
awk -F, '{sum[$1] += $2} END{for (k in sum) print k, sum[k]}' sales.csv
```

### Delete array element

```bash
# delete arr["key"]
```

## Functions

### Built-in string functions

```bash
awk '{print toupper($1)}' file.txt         # uppercase
awk '{print tolower($0)}' file.txt         # lowercase
awk '{print length($1)}' file.txt          # string length
awk '{print substr($1, 1, 3)}' file.txt   # substring (start, length)
awk '{gsub(/old/, "new"); print}' file.txt # global substitution
awk '{sub(/first/, "replaced"); print}' file.txt  # first occurrence
awk '{n = split($0, arr, ":"); print arr[1]}' file.txt  # split into array
awk 'match($0, /[0-9]+/) {print substr($0, RSTART, RLENGTH)}' file.txt
```

### Math functions

```bash
awk '{print int($1)}' file.txt             # truncate to integer
awk '{print sqrt($1)}' file.txt
awk 'BEGIN{print sin(3.14159/2)}'
awk 'BEGIN{srand(); print rand()}'         # random 0-1
awk '{printf "%.2f\n", $1}' file.txt       # formatted output
```

### printf formatting

```bash
awk '{printf "%-20s %10.2f\n", $1, $2}' data.txt    # left-align name, right-align number
awk '{printf "%05d %s\n", NR, $0}' file.txt          # zero-padded line numbers
```

## Common One-Liners

### Sum a column

```bash
awk '{sum += $1} END{print sum}' numbers.txt
```

### Count unique values

```bash
awk '{a[$1]++} END{for (k in a) print k, a[k]}' file.txt | sort
```

### Remove duplicate lines

```bash
awk '!seen[$0]++' file.txt
```

### Print between patterns

```bash
awk '/BEGIN_SECTION/,/END_SECTION/' config.txt
```

### Join lines

```bash
awk '{printf "%s ", $0} END{print ""}' file.txt
```

### Swap columns

```bash
awk '{print $2, $1}' file.txt
```

### Add column headers to CSV

```bash
awk 'NR==1{print "id,name,email"} NR>0{print}' data.csv
```

### Extract IPs from log

```bash
awk '{print $1}' /var/log/nginx/access.log | sort | uniq -c | sort -rn | head -20
```

### Convert CSV to TSV

```bash
awk -F, -v OFS='\t' '{$1=$1; print}' data.csv
```

## Tips

- Field numbering starts at `$1`. `$0` is the whole line.
- `$1=$1` forces awk to reconstruct the line with OFS. Useful when changing the output separator.
- `awk` uses extended regex by default. No need for `-E` like grep.
- Single quotes around the awk program prevent shell expansion of `$`.
- `print` with no arguments prints `$0` (the full line).
- Arrays in awk are associative (hash maps). They are always string-keyed.
- `OFMT` controls default number formatting for print. Use `printf` for explicit control.
- `-v var=value` passes shell variables into awk. For dynamic values: `awk -v threshold="$THRESH" '$3 > threshold'`.

## References

- [GNU AWK Manual (GAWK)](https://www.gnu.org/software/gawk/manual/) -- complete gawk reference
- [man gawk](https://man7.org/linux/man-pages/man1/gawk.1.html) -- gawk man page
- [POSIX awk Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html) -- portable awk behavior
- [The AWK Programming Language (book)](https://awk.dev/) -- by Aho, Weinberger, and Kernighan (2nd edition)
- [AWK One-Liners Explained](https://catonmat.net/awk-one-liners-explained-part-one) -- practical recipes with explanations
- [mawk](https://invisible-island.net/mawk/mawk.html) -- fast awk interpreter (default on Debian/Ubuntu)
- [GNU AWK Built-in Functions](https://www.gnu.org/software/gawk/manual/html_node/Built_002din.html) -- string, math, I/O, and time functions
- [GAWK Extensions](https://www.gnu.org/software/gawk/manual/html_node/Extension-Samples.html) -- loadable extensions for gawk
