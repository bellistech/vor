# Xargs (Build and Execute Commands from Stdin)

Xargs reads items from standard input and executes a command with those items as arguments, enabling efficient pipeline composition for batch processing, parallel execution, and overcoming shell argument length limits.

## Basic Usage

```bash
# Pass stdin as arguments to a command
echo "file1.txt file2.txt file3.txt" | xargs rm

# Default behavior: appends all items as arguments
echo "one two three" | xargs echo
# echo one two three  -> "one two three"

# Read from a file
xargs < filelist.txt rm

# Pipe from find
find . -name "*.log" | xargs rm

# Pipe from grep
grep -rl "TODO" src/ | xargs grep -l "FIXME"
```

## Null-Terminated Input (-0)

```bash
# Handle filenames with spaces, quotes, newlines
find . -name "*.txt" -print0 | xargs -0 rm

# Combine with grep -Z (null output)
grep -rlZ "pattern" . | xargs -0 sed -i 's/old/new/g'

# Null-terminated input from other sources
printf "file one.txt\0file two.txt\0" | xargs -0 cat

# CRITICAL: Always use -print0/-0 for filenames
# Without -0, this BREAKS on "my file.txt":
find . -name "*.txt" | xargs rm       # WRONG
find . -name "*.txt" -print0 | xargs -0 rm  # CORRECT
```

## Replacement String (-I)

```bash
# Use {} as placeholder for each input item
echo "file.txt" | xargs -I{} cp {} /backup/{}

# Rename files
ls *.txt | xargs -I{} mv {} {}.bak

# Complex substitution
cat urls.txt | xargs -I{} curl -o "{}.html" "{}"

# Multiple uses of placeholder in one command
cat hosts.txt | xargs -I{} ssh {} "hostname && uptime"

# Custom placeholder string
echo "myfile" | xargs -Ifile cp file /dest/file

# -I implies -L 1 (one line at a time)
cat list.txt | xargs -I{} echo "Processing: {}"
```

## Batch Size (-n)

```bash
# Process N items at a time
echo "a b c d e f" | xargs -n 2 echo
# echo a b  -> "a b"
# echo c d  -> "c d"
# echo e f  -> "e f"

# Useful when command takes a specific number of args
echo "src1 dst1 src2 dst2" | xargs -n 2 cp

# Single item at a time
echo "a b c" | xargs -n 1 echo
# echo a  -> "a"
# echo b  -> "b"
# echo c  -> "c"

# With -0 for safety
find . -name "*.jpg" -print0 | xargs -0 -n 10 mogrify -resize 800x600
```

## Parallel Execution (-P)

```bash
# Run up to 4 processes in parallel
find . -name "*.png" -print0 | xargs -0 -P 4 -I{} convert {} {}.webp

# Parallel with batch size
cat urls.txt | xargs -P 8 -n 1 wget -q

# Use all CPU cores
CORES=$(nproc)
find . -name "*.gz" -print0 | xargs -0 -P "$CORES" -n 1 gunzip

# Parallel SSH commands
cat hosts.txt | xargs -P 10 -I{} ssh {} "apt update && apt upgrade -y"

# Parallel make-like pattern
seq 1 100 | xargs -P 8 -I{} bash -c 'process_item {}'

# Combine -P with -n for parallel batches
find . -name "*.csv" -print0 | xargs -0 -P 4 -n 5 process_batch
# 4 parallel processes, each handling 5 files at a time
```

## Lines Mode (-L)

```bash
# Process N lines at a time (instead of N words)
cat commands.txt | xargs -L 1 bash -c

# Two lines at a time
printf "src1\ndst1\nsrc2\ndst2\n" | xargs -L 2 cp

# -L 1 is equivalent to one logical line per command
# (lines ending with space continue to next line)
```

## Delimiter (-d)

```bash
# Use custom delimiter
echo "one:two:three" | xargs -d: echo
# echo one two three

# Comma delimiter
echo "a,b,c,d" | xargs -d, -n 1 echo

# Newline delimiter (default on some systems)
echo -e "a\nb\nc" | xargs -d'\n' echo
```

## Prompt and Verbose (-p, -t)

```bash
# Prompt before each execution
find . -name "*.tmp" -print0 | xargs -0 -p rm
# rm ./old.tmp ./cache.tmp? y

# Show command before executing (verbose)
echo "a b c" | xargs -t -n 1 echo
# echo a
# a
# echo b
# b

# Dry run: show what would be executed without running
echo "a b c" | xargs -t -n 1 echo 2>&1 | grep "^echo"
```

## Pipeline Composition Patterns

### find + xargs

```bash
# Delete files older than 30 days
find /tmp -type f -mtime +30 -print0 | xargs -0 rm -f

# Change permissions on directories only
find . -type d -print0 | xargs -0 chmod 755

# Count lines in all Python files
find . -name "*.py" -print0 | xargs -0 wc -l

# Find and replace across files
find . -name "*.go" -print0 | xargs -0 sed -i 's/oldFunc/newFunc/g'

# Archive specific files
find . -name "*.log" -mtime +7 -print0 | xargs -0 tar -czf old_logs.tar.gz
```

### grep + xargs

```bash
# Find files containing pattern, then process them
grep -rlZ "deprecated" src/ | xargs -0 -I{} echo "TODO: update {}"

# Count matches across files
grep -rl "TODO" . | xargs grep -c "TODO" | sort -t: -k2 -rn

# Extract and process
grep -oP 'https?://\S+' urls.txt | xargs -n 1 -P 4 curl -sI
```

### Other Combinations

```bash
# Docker: remove all stopped containers
docker ps -aq --filter "status=exited" | xargs docker rm

# Git: checkout files matching pattern
git diff --name-only | xargs -I{} git checkout -- {}

# Kill processes by pattern
pgrep -f "worker" | xargs kill -TERM

# Bulk file operations with awk output
df -h | awk '$5 > "80%" {print $6}' | xargs -I{} echo "WARNING: {} is nearly full"

# Process JSON with jq + xargs
cat config.json | jq -r '.servers[]' | xargs -I{} ping -c 1 {}
```

## Handling Edge Cases

```bash
# Empty input: prevent running command with no args
echo "" | xargs --no-run-if-empty rm
# GNU xargs: --no-run-if-empty or -r
# BSD xargs: default behavior (does not run on empty)

# Max command length check
getconf ARG_MAX
# 2097152 (2MB on Linux)

# Force xargs to respect ARG_MAX
find / -name "*.log" -print0 | xargs -0 -s 131072 rm
# -s limits command line to 131072 bytes

# Handle command failure
find . -name "*.dat" -print0 | xargs -0 -n 1 process_file || true

# Exit on first failure
find . -name "*.dat" -print0 | xargs -0 -n 1 sh -c 'process_file "$1" || exit 255' _
# xargs exits immediately if child returns 255

# Stdin conflict: when command also reads stdin
find . -name "*.sh" -print0 | xargs -0 -I{} bash -c 'shellcheck "{}" < /dev/null'
```

## GNU vs BSD xargs Differences

```bash
# GNU xargs (Linux)
xargs --no-run-if-empty    # Skip on empty input
xargs -r                   # Same as above (short form)
xargs -P 0                 # As many as possible in parallel
xargs -d '\n'              # Custom delimiter

# BSD xargs (macOS)
xargs                      # Already skips on empty input
xargs -P 0                 # As many as possible
xargs -0                   # Null delimiter (same on both)
# BSD lacks -d flag; use -0 with null-terminated input

# Portable: always use -0 with find -print0
```

## Tips

- Always pair `find -print0` with `xargs -0` to handle filenames containing spaces, quotes, or newlines
- Use `-P $(nproc)` for embarrassingly parallel workloads like image conversion or file compression
- `-I{}` implies `-L 1` and `-n 1`, processing one item at a time; do not combine `-I` with `-n`
- Use `--no-run-if-empty` (or `-r`) on GNU xargs to prevent running commands when stdin is empty
- `-n 1` with `-P N` is the standard pattern for parallel per-item processing
- Check `getconf ARG_MAX` to understand the maximum command line length on your system
- xargs is faster than `find -exec` with `+` for most workloads because it batches arguments
- When the target command also reads stdin (like `ssh`), redirect stdin from `/dev/null` inside xargs
- Return code 255 from a child process causes xargs to stop immediately; use this for fail-fast behavior
- For complex per-item logic, use `xargs -I{} bash -c 'commands here' _` with the trailing underscore as $0
- Use `-t` (trace) during development to see exactly what commands xargs is building
- Combine `-P` for parallelism with `-n` for batch size to control both concurrency and granularity

## See Also

proc-sys, signals, inotify

## References

- [xargs(1) Man Page](https://man7.org/linux/man-pages/man1/xargs.1.html)
- [GNU findutils Manual](https://www.gnu.org/software/findutils/manual/html_mono/find.html)
- [POSIX xargs Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/xargs.html)
- [ShellCheck: Safe xargs Usage](https://www.shellcheck.net/wiki/SC2038)
