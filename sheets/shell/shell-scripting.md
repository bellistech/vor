# Shell Scripting (Portable Patterns)

> Writing robust, portable shell scripts with proper error handling and argument parsing.

## Shebang and Basics

```bash
#!/usr/bin/env bash           # portable bash (finds bash in PATH)
#!/bin/bash                   # explicit path (faster, no env lookup)
#!/bin/sh                     # POSIX sh only — most portable

# make executable
chmod +x script.sh
./script.sh
```

## Strict Mode

### set -euo pipefail

```bash
#!/usr/bin/env bash
set -euo pipefail             # the "unofficial strict mode"
# -e  exit immediately on non-zero exit
# -u  treat unset variables as errors
# -o pipefail  pipe returns rightmost non-zero exit code

# optional: debug trace
set -x                        # print each command before execution
set +x                        # turn off trace
```

### Handling -e Edge Cases

```bash
# command allowed to fail: use || true
grep "maybe" file.txt || true

# capture exit code without triggering -e
if ! output=$(some_command 2>&1); then
    echo "failed: $output"
fi

# or explicitly check
set +e
risky_command
rc=$?
set -e
```

## Exit Codes

```bash
exit 0                   # success
exit 1                   # general error
exit 2                   # misuse of shell command (convention)
exit 126                 # command not executable
exit 127                 # command not found
exit 128+N               # killed by signal N (e.g., 130 = SIGINT)

# custom exit codes
readonly EX_OK=0
readonly EX_USAGE=64
readonly EX_DATAERR=65
readonly EX_NOINPUT=66
readonly EX_CONFIG=78
```

## Argument Parsing

### With getopts (POSIX)

```bash
#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo "Usage: $0 [-v] [-o output] [-n count] file..."
    exit 64
}

verbose=false
output="/dev/stdout"
count=1

while getopts ":vo:n:h" opt; do
    case $opt in
        v) verbose=true ;;
        o) output="$OPTARG" ;;
        n) count="$OPTARG" ;;
        h) usage ;;
        :) echo "Option -$OPTARG requires an argument" >&2; usage ;;
        \?) echo "Unknown option -$OPTARG" >&2; usage ;;
    esac
done
shift $((OPTIND - 1))

# remaining args are positional
files=("$@")
[[ ${#files[@]} -eq 0 ]] && usage
```

### Manual Long Options

```bash
while [[ $# -gt 0 ]]; do
    case "$1" in
        -v|--verbose)  verbose=true; shift ;;
        -o|--output)   output="$2"; shift 2 ;;
        --output=*)    output="${1#*=}"; shift ;;
        -n|--count)    count="$2"; shift 2 ;;
        -h|--help)     usage ;;
        --)            shift; break ;;      # end of options
        -*)            echo "Unknown: $1" >&2; usage ;;
        *)             break ;;             # start of positional args
    esac
done
```

## Temporary Files

```bash
# mktemp — safe temp file creation
tmpfile=$(mktemp)                        # /tmp/tmp.XXXXXXXXXX
tmpdir=$(mktemp -d)                      # temp directory
tmpfile=$(mktemp /tmp/myapp.XXXXXX)      # custom prefix

# always clean up
trap 'rm -f "$tmpfile"' EXIT

# or for temp directories
cleanup() {
    rm -rf "$tmpdir"
}
trap cleanup EXIT
```

## Signal Handling

```bash
#!/usr/bin/env bash
set -euo pipefail

pidfile="/var/run/myapp.pid"
logfile="/var/log/myapp.log"

cleanup() {
    local rc=$?
    rm -f "$pidfile"
    echo "Exiting with code $rc" >> "$logfile"
    exit $rc
}

on_sigint() {
    echo "Interrupted by user" >> "$logfile"
    exit 130
}

on_sighup() {
    echo "Reloading config..." >> "$logfile"
    # re-read config here
}

trap cleanup EXIT
trap on_sigint INT
trap on_sighup HUP
trap 'echo "Terminated" >> "$logfile"; exit 143' TERM

echo $$ > "$pidfile"

# main loop
while true; do
    do_work
    sleep 60
done
```

## Error Handling Patterns

### Die Function

```bash
die() {
    echo "ERROR: $*" >&2
    exit 1
}

[[ -f "$config" ]] || die "Config file not found: $config"
```

### Logging

```bash
readonly LOG_FILE="/var/log/myapp.log"

log() {
    local level="$1"; shift
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*" | tee -a "$LOG_FILE" >&2
}

log INFO "Starting process"
log WARN "Disk usage above 80%"
log ERROR "Connection failed"
```

### Retry Logic

```bash
retry() {
    local max_attempts=$1; shift
    local delay=$1; shift
    local attempt=1

    while (( attempt <= max_attempts )); do
        if "$@"; then
            return 0
        fi
        echo "Attempt $attempt/$max_attempts failed. Retrying in ${delay}s..." >&2
        sleep "$delay"
        ((attempt++))
    done

    echo "All $max_attempts attempts failed." >&2
    return 1
}

retry 3 5 curl -sf https://api.example.com/health
```

## Input Validation

```bash
# check required commands exist
require_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed"
}
require_cmd jq
require_cmd curl

# validate numeric input
is_integer() {
    [[ "$1" =~ ^-?[0-9]+$ ]]
}
is_integer "$count" || die "count must be an integer: $count"

# validate file input
[[ -f "$input" ]] || die "File not found: $input"
[[ -r "$input" ]] || die "File not readable: $input"
```

## Portable Patterns

### POSIX-Safe Constructs

```bash
# use [ ] instead of [[ ]] for POSIX sh
[ -f "$file" ] && echo "exists"

# use $(command) not backticks
today=$(date +%Y-%m-%d)

# printf over echo for portability
printf '%s\n' "$message"

# use command -v over which
if command -v git >/dev/null 2>&1; then
    echo "git is installed"
fi
```

### Path Handling

```bash
# get script's own directory (handles symlinks)
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# resolve absolute path
abs_path="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"

# safe path joining
config_file="${script_dir}/config.yml"
```

### Lockfile

```bash
lockfile="/var/run/myapp.lock"

acquire_lock() {
    if ! mkdir "$lockfile" 2>/dev/null; then
        die "Another instance is running (lock: $lockfile)"
    fi
    trap 'rm -rf "$lockfile"' EXIT
}

acquire_lock
```

## Common Recipes

### Read Config File

```bash
# key=value config file
declare -A config
while IFS='=' read -r key value; do
    [[ "$key" =~ ^[[:space:]]*# ]] && continue   # skip comments
    [[ -z "$key" ]] && continue                    # skip empty lines
    config["${key// /}"]="${value// /}"             # trim spaces
done < config.ini
```

### Parallel Execution

```bash
# run jobs in parallel, limit concurrency
max_jobs=4
for file in *.csv; do
    process "$file" &
    # throttle: wait if too many background jobs
    while (( $(jobs -r | wc -l) >= max_jobs )); do
        sleep 0.1
    done
done
wait    # wait for all remaining jobs
```

### Yes/No Prompt

```bash
confirm() {
    local prompt="${1:-Are you sure?}"
    read -rp "$prompt [y/N] " answer
    [[ "$answer" =~ ^[Yy]$ ]]
}

confirm "Delete all logs?" || exit 0
```

## Tips

- Always use `set -euo pipefail` at the top of scripts. It catches 90% of bugs.
- Quote all variables: `"$var"`, `"$@"`, `"${array[@]}"`. Unquoted variables cause word splitting bugs.
- Use `local` in functions to avoid polluting the global namespace.
- `mktemp` + `trap cleanup EXIT` is the only safe way to handle temp files.
- `command -v` is POSIX; `which` is not. Always prefer `command -v`.
- `$BASH_SOURCE` is bash-specific. In POSIX sh, use `$0` (but it won't resolve symlinks).
- `mkdir` for lockfiles is atomic on all filesystems. `flock` is Linux-only but more robust.
- getopts only handles single-character options. For long options, parse manually or use `getopt` (GNU).
- Always use `read -r` to prevent backslash interpretation.
- `shellcheck` is the single best tool for catching shell scripting bugs. Install it and run it on every script.

## See Also

- bash, zsh, awk, sed, make, regex

## References

- [POSIX Shell Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html) -- portable shell command language
- [POSIX Utilities](https://pubs.opengroup.org/onlinepubs/9699919799/idx/utilities.html) -- standard utility index (test, printf, etc.)
- [Bash Reference Manual](https://www.gnu.org/software/bash/manual/) -- GNU Bash reference
- [Dash (POSIX sh)](https://manpages.debian.org/dash) -- lightweight POSIX shell man page
- [ShellCheck](https://www.shellcheck.net/) -- static analysis tool for shell scripts
- [ShellCheck Wiki](https://www.shellcheck.net/wiki/) -- explanations for every ShellCheck warning
- [Bash Pitfalls](https://mywiki.wooledge.org/BashPitfalls) -- common shell scripting mistakes
- [Bash FAQ](https://mywiki.wooledge.org/BashFAQ) -- practical answers to common questions
- [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html) -- conventions for production shell scripts
- [man test](https://man7.org/linux/man-pages/man1/test.1.html) -- conditional expressions and file tests
