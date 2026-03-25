# Bash (Bourne Again Shell)

> GNU's Unix shell and command language — the default on most Linux systems.

## Variables

### Assignment and Export

```bash
name="world"                    # no spaces around =
export PATH="$HOME/bin:$PATH"  # available to child processes
readonly DB_HOST="localhost"    # immutable
unset name                     # delete variable
```

### Special Variables

```bash
echo $0          # script name
echo $1 $2       # positional args
echo $#          # number of args
echo $@          # all args (preserves quoting)
echo $*          # all args (single string)
echo $?          # last exit code
echo $$          # current PID
echo $!          # last background PID
echo $_          # last argument of previous command
echo $LINENO     # current line number in script
```

## Arrays

### Indexed Arrays

```bash
fruits=(apple banana cherry)
fruits[3]="date"
echo "${fruits[0]}"            # apple
echo "${fruits[@]}"            # all elements
echo "${#fruits[@]}"           # length: 4
echo "${!fruits[@]}"           # indices: 0 1 2 3
echo "${fruits[@]:1:2}"        # slice: banana cherry
fruits+=(elderberry)           # append
unset 'fruits[1]'              # remove element (leaves gap)
```

### Associative Arrays

```bash
declare -A config
config[host]="localhost"
config[port]=5432
echo "${config[host]}"
echo "${!config[@]}"           # keys: host port
echo "${config[@]}"            # values
```

## Parameter Expansion

### Defaults and Substitution

```bash
echo "${name:-default}"        # use default if unset/empty
echo "${name:=default}"        # assign default if unset/empty
echo "${name:+alternate}"      # use alternate if set and non-empty
echo "${name:?error msg}"      # exit with error if unset/empty
```

### String Operations

```bash
path="/home/user/docs/file.tar.gz"
echo "${#path}"                # string length: 30
echo "${path#*/}"              # remove shortest prefix: home/user/docs/file.tar.gz
echo "${path##*/}"             # remove longest prefix: file.tar.gz
echo "${path%.*}"              # remove shortest suffix: /home/user/docs/file.tar
echo "${path%%.*}"             # remove longest suffix: /home/user/docs/file
echo "${path/docs/files}"      # replace first: /home/user/files/file.tar.gz
echo "${path//o/0}"            # replace all o with 0
echo "${path^^}"               # uppercase
echo "${path,,}"               # lowercase
echo "${path:6:4}"             # substring: user
```

## Conditionals

### Test Expressions

```bash
# file tests
[[ -f /etc/hosts ]]            # regular file exists
[[ -d /tmp ]]                  # directory exists
[[ -r file.txt ]]              # readable
[[ -w file.txt ]]              # writable
[[ -x script.sh ]]             # executable
[[ -s file.txt ]]              # non-empty file
[[ -L /usr/bin/python ]]       # symlink
[[ file1 -nt file2 ]]         # file1 newer than file2

# string tests
[[ -z "$var" ]]                # empty string
[[ -n "$var" ]]                # non-empty string
[[ "$a" == "$b" ]]             # equal
[[ "$a" != "$b" ]]             # not equal
[[ "$a" == *.txt ]]            # glob match
[[ "$a" =~ ^[0-9]+$ ]]        # regex match

# arithmetic
(( x > 5 ))
(( x >= 5 && x <= 10 ))
```

### If / Case

```bash
if [[ -f config.yml ]]; then
    echo "found"
elif [[ -f config.json ]]; then
    echo "json fallback"
else
    echo "no config"
fi

case "$1" in
    start|run)   do_start ;;
    stop)        do_stop ;;
    restart)     do_stop; do_start ;;
    *)           echo "Usage: $0 {start|stop|restart}" ;;
esac
```

## Loops

```bash
# iterate list
for f in *.log; do
    gzip "$f"
done

# C-style
for ((i=0; i<10; i++)); do
    echo "$i"
done

# while read (line by line)
while IFS= read -r line; do
    echo ">> $line"
done < input.txt

# infinite loop with break
while true; do
    read -rp "Continue? [y/n] " ans
    [[ "$ans" == "n" ]] && break
done

# iterate array
for item in "${fruits[@]}"; do
    echo "$item"
done
```

## Functions

```bash
greet() {
    local name="${1:?name required}"   # local scope, required arg
    local greeting="${2:-Hello}"
    echo "$greeting, $name!"
    return 0
}

greet "Alice"                          # Hello, Alice!
greet "Bob" "Hey"                      # Hey, Bob!

# capture output
result=$(greet "World")
```

## Redirection and Process Substitution

```bash
cmd > out.txt                  # stdout to file (overwrite)
cmd >> out.txt                 # stdout to file (append)
cmd 2> err.txt                 # stderr to file
cmd &> all.txt                 # stdout + stderr to file
cmd 2>&1                       # stderr to stdout
cmd > /dev/null 2>&1           # silence everything
cmd1 | cmd2                    # pipe stdout
cmd1 |& cmd2                   # pipe stdout + stderr

# process substitution
diff <(sort file1) <(sort file2)
while read -r line; do
    echo "$line"
done < <(find . -name "*.go")

# here document
cat <<EOF
Hello $name
Today is $(date)
EOF

# here string
grep "error" <<< "$log_output"
```

## Traps and Signal Handling

```bash
cleanup() {
    rm -f "$tmpfile"
    echo "cleaned up"
}
trap cleanup EXIT              # run on script exit
trap 'echo interrupted' INT    # Ctrl-C
trap '' TERM                   # ignore SIGTERM
trap - INT                     # reset to default
```

## Set Options

```bash
set -e              # exit on error
set -u              # error on undefined variable
set -o pipefail     # pipe fails if any command fails
set -x              # print commands before execution (debug)
set -f              # disable globbing
set -euo pipefail   # common combo for scripts
```

## History

```bash
history              # show history
!!                   # repeat last command
!42                  # run history entry 42
!grep                # last command starting with grep
!$                   # last argument of previous command
!*                   # all arguments of previous command
^old^new             # replace in last command and run
ctrl-r               # reverse search
history -c           # clear history
HISTSIZE=10000       # entries in memory
HISTFILESIZE=20000   # entries in file
HISTCONTROL=ignoreboth  # ignore dupes and space-prefixed
```

## Arithmetic

```bash
echo $((3 + 5))           # 8
echo $((10 / 3))          # 3 (integer division)
echo $((2 ** 10))         # 1024
((count++))               # increment
((total = price * qty))   # assign
```

## Tips

- Always quote `"$variables"` to prevent word splitting and glob expansion.
- Use `[[ ]]` over `[ ]` in bash -- it handles spaces and has regex support.
- `local` variables in functions prevent polluting the global scope.
- `set -euo pipefail` should be at the top of every script.
- `$@` preserves argument quoting; `$*` smashes everything into one string.
- Arrays are 0-indexed. `unset 'arr[i]'` leaves a gap -- it does not re-index.
- Use `mapfile -t lines < file.txt` to read a file into an array (bash 4+).
- `${var:-}` is the safe way to reference a possibly-unset variable under `set -u`.
- Process substitution `<()` creates a temporary file descriptor -- not available in POSIX sh.
- Heredocs with `<<'EOF'` (quoted) disable variable expansion inside the block.

## References

- [Bash Reference Manual](https://www.gnu.org/software/bash/manual/) -- complete GNU Bash reference
- [man bash](https://man7.org/linux/man-pages/man1/bash.1.html) -- bash man page
- [Bash Hackers Wiki (archived)](https://web.archive.org/web/2023*/https://wiki.bash-hackers.org/) -- in-depth articles and scripting patterns
- [POSIX Shell Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html) -- portable shell behavior
- [ShellCheck](https://www.shellcheck.net/) -- online shell script linter and analyzer
- [Bash Pitfalls](https://mywiki.wooledge.org/BashPitfalls) -- common mistakes and how to avoid them
- [Bash FAQ](https://mywiki.wooledge.org/BashFAQ) -- answers to frequently asked questions
- [Bash Guide (Wooledge)](https://mywiki.wooledge.org/BashGuide) -- comprehensive beginner-to-advanced guide
- [GNU Readline Library](https://tiswww.case.edu/php/chet/readline/rltop.html) -- line editing and key bindings used by Bash
- [Bash Changes (NEWS)](https://tiswww.case.edu/php/chet/bash/NEWS) -- changelog for every Bash release
