# Linux Automation and Scripting

Task scheduling, bash scripting patterns, error handling, signal traps, file locking, automation best practices.

## Cron

### Crontab Syntax

```bash
# ┌───────────── minute (0-59)
# │ ┌───────────── hour (0-23)
# │ │ ┌───────────── day of month (1-31)
# │ │ │ ┌───────────── month (1-12 or jan-dec)
# │ │ │ │ ┌───────────── day of week (0-7, 0 and 7 = Sunday)
# │ │ │ │ │
# * * * * *  command

# Edit crontab
crontab -e

# List crontab
crontab -l

# Edit another user's crontab
crontab -e -u username

# Remove crontab
crontab -r

# Examples:
# Every 5 minutes
*/5 * * * *  /usr/local/bin/check.sh

# Daily at 2:30 AM
30 2 * * *  /usr/local/bin/backup.sh

# Mon-Fri at 9:00 AM
0 9 * * 1-5  /usr/local/bin/report.sh

# First day of month at midnight
0 0 1 * *  /usr/local/bin/monthly.sh

# Every Sunday at 3 AM
0 3 * * 0  /usr/local/bin/weekly.sh

# Every 15 minutes during business hours
*/15 9-17 * * 1-5  /usr/local/bin/monitor.sh
```

### Cron Environment

```bash
# Cron runs with minimal environment
# Set PATH explicitly in crontab:
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
SHELL=/bin/bash
MAILTO=admin@example.com

# Use full paths in scripts
# Redirect output to prevent mail
0 2 * * * /usr/local/bin/backup.sh >> /var/log/backup.log 2>&1

# Or discard output
0 2 * * * /usr/local/bin/backup.sh > /dev/null 2>&1
```

### Cron Directories

```bash
# System cron directories (scripts dropped in, no crontab syntax)
/etc/cron.hourly/
/etc/cron.daily/
/etc/cron.weekly/
/etc/cron.monthly/

# System crontab with user field
/etc/crontab
# min hour dom mon dow user command

# Drop-in directory (crontab syntax with user field)
/etc/cron.d/

# Access control
/etc/cron.allow    # only these users can use cron
/etc/cron.deny     # these users cannot use cron
```

### Anacron

```bash
# /etc/anacrontab — runs missed jobs (laptops, servers that aren't 24/7)
# period  delay  job-id     command
1         5      cron.daily   nice run-parts /etc/cron.daily
7         25     cron.weekly  nice run-parts /etc/cron.weekly
@monthly  45     cron.monthly nice run-parts /etc/cron.monthly

# period: days between runs
# delay: minutes to wait after anacron starts
# Timestamps stored in /var/spool/anacron/
```

## at and batch

### One-Time Scheduling

```bash
# Schedule a job
at 2:30 AM tomorrow
at> /usr/local/bin/backup.sh
at> <Ctrl-D>

# Schedule from string
echo "/usr/local/bin/backup.sh" | at 2:30 AM tomorrow

# Time formats
at now + 1 hour
at now + 30 minutes
at 14:00 2024-12-25
at noon
at midnight
at teatime          # 4:00 PM

# List pending jobs
atq

# Show job content
at -c <job_number>

# Remove job
atrm <job_number>

# batch — runs when load average drops below 1.5
echo "/usr/local/bin/heavy-task.sh" | batch
```

## systemd Timers

### Timer Unit

```bash
# /etc/systemd/system/backup.timer
[Unit]
Description=Daily backup timer

[Timer]
OnCalendar=*-*-* 02:30:00
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target

# Corresponding service unit
# /etc/systemd/system/backup.service
[Unit]
Description=Daily backup

[Service]
Type=oneshot
ExecStart=/usr/local/bin/backup.sh
```

### OnCalendar Syntax

```bash
# Calendar expressions:
OnCalendar=daily                  # 00:00:00
OnCalendar=weekly                 # Mon 00:00:00
OnCalendar=monthly                # 1st 00:00:00
OnCalendar=*-*-* 02:30:00        # daily at 2:30 AM
OnCalendar=Mon..Fri *-*-* 09:00:00  # weekdays at 9 AM
OnCalendar=*-*-01 00:00:00       # first of month
OnCalendar=*:0/15                # every 15 minutes

# Monotonic timers (relative):
OnBootSec=5min                   # 5 minutes after boot
OnUnitActiveSec=1h               # 1 hour after last activation
OnStartupSec=10min               # 10 min after systemd started

# Persistent=true — catch up missed runs after downtime

# Test calendar expression
systemd-analyze calendar "Mon..Fri *-*-* 09:00:00"
```

### Manage Timers

```bash
# Enable and start
systemctl enable --now backup.timer

# List all timers
systemctl list-timers --all

# Check timer status
systemctl status backup.timer

# Trigger immediately (run the service now)
systemctl start backup.service

# Show next trigger time
systemctl list-timers backup.timer
```

## Expect Scripts

### Basic Expect

```bash
#!/usr/bin/expect -f

# SSH with password (when keys aren't available)
set timeout 30
spawn ssh user@server.example.com

expect "password:"
send "mypassword\r"

expect "$ "
send "hostname\r"

expect "$ "
send "exit\r"

expect eof
```

### Expect within Bash

```bash
#!/bin/bash

# Inline expect
expect <<'EOF'
spawn ssh user@server
expect "password:"
send "mypassword\r"
expect "$ "
send "uptime\r"
expect "$ "
send "exit\r"
expect eof
EOF
```

## Heredocs

### Heredoc Patterns

```bash
# Basic heredoc (variable expansion happens)
cat <<EOF
Hello, $USER
Current date: $(date)
Home: $HOME
EOF

# Quoted heredoc (no expansion — literal)
cat <<'EOF'
This $variable is not expanded
Neither is $(this command)
EOF

# Heredoc to file
cat > /tmp/config.conf <<EOF
server=example.com
port=8080
EOF

# Heredoc to command stdin
mysql -u root <<EOF
CREATE DATABASE mydb;
GRANT ALL ON mydb.* TO 'user'@'localhost';
EOF

# Indented heredoc (strips leading tabs)
cat <<-EOF
	This is indented with tabs
	Tabs are stripped from output
	EOF

# Here string
grep "pattern" <<< "search in this string"
```

## Bash Functions

### Function Definition and Scope

```bash
# Define function
backup_database() {
    local db_name="$1"
    local backup_dir="${2:-/var/backups}"
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)

    if [[ -z "$db_name" ]]; then
        echo "Usage: backup_database <db_name> [backup_dir]" >&2
        return 1
    fi

    local backup_file="${backup_dir}/${db_name}_${timestamp}.sql"
    pg_dump "$db_name" > "$backup_file" 2>/dev/null

    if [[ $? -eq 0 ]]; then
        echo "$backup_file"
        return 0
    else
        echo "Backup failed for $db_name" >&2
        return 1
    fi
}

# Call function
result=$(backup_database mydb /tmp/backups)
echo "Backup: $result"

# Local variables: visible only within function
# Return codes: 0 = success, 1-255 = failure
# Capture output: var=$(function_name args)
```

### Function Libraries

```bash
# /usr/local/lib/bash/logging.sh
log_info()  { echo "[INFO]  $(date +%Y-%m-%dT%H:%M:%S) $*"; }
log_warn()  { echo "[WARN]  $(date +%Y-%m-%dT%H:%M:%S) $*" >&2; }
log_error() { echo "[ERROR] $(date +%Y-%m-%dT%H:%M:%S) $*" >&2; }

die() {
    log_error "$@"
    exit 1
}

# Source in scripts
source /usr/local/lib/bash/logging.sh
log_info "Starting backup"
```

## Bash Arrays

### Indexed Arrays

```bash
# Declare
declare -a fruits=("apple" "banana" "cherry")

# Append
fruits+=("date")

# Access
echo "${fruits[0]}"        # first element
echo "${fruits[-1]}"       # last element
echo "${fruits[@]}"        # all elements
echo "${#fruits[@]}"       # count

# Iterate
for fruit in "${fruits[@]}"; do
    echo "$fruit"
done

# Slice
echo "${fruits[@]:1:2}"   # elements 1-2

# Delete element
unset 'fruits[1]'

# Find index
for i in "${!fruits[@]}"; do
    echo "$i: ${fruits[$i]}"
done
```

### Associative Arrays

```bash
# Declare (must use declare -A)
declare -A config
config[host]="example.com"
config[port]="8080"
config[protocol]="https"

# Or inline
declare -A config=(
    [host]="example.com"
    [port]="8080"
    [protocol]="https"
)

# Access
echo "${config[host]}"

# All keys
echo "${!config[@]}"

# All values
echo "${config[@]}"

# Iterate
for key in "${!config[@]}"; do
    echo "$key = ${config[$key]}"
done

# Check key exists
if [[ -v config[host] ]]; then
    echo "host is set"
fi
```

## Parameter Expansion

### Common Expansions

```bash
name="hello_world.tar.gz"

# Default values
echo "${var:-default}"        # use default if unset/empty
echo "${var:=default}"        # assign default if unset/empty
echo "${var:+alternate}"      # use alternate if set
echo "${var:?error message}"  # error if unset/empty

# String operations
echo "${name#*.}"             # remove shortest prefix match: tar.gz
echo "${name##*.}"            # remove longest prefix match:  gz
echo "${name%.*}"             # remove shortest suffix match: hello_world.tar
echo "${name%%.*}"            # remove longest suffix match:  hello_world

# Substitution
echo "${name/world/earth}"    # replace first: hello_earth.tar.gz
echo "${name//l/L}"           # replace all:   heLLo_worLd.tar.gz

# Length
echo "${#name}"               # string length

# Substring
echo "${name:0:5}"            # first 5 chars: hello
echo "${name: -6}"            # last 6 chars: tar.gz

# Case conversion (bash 4+)
echo "${name^^}"              # uppercase: HELLO_WORLD.TAR.GZ
echo "${name,,}"              # lowercase
echo "${name^}"               # capitalize first letter
```

## Process Substitution

### Usage Patterns

```bash
# Compare two command outputs
diff <(ls /dir1) <(ls /dir2)

# Feed command output as file to program
paste <(cut -f1 file1) <(cut -f2 file2)

# Process while reading (avoids subshell pipe issue)
while IFS= read -r line; do
    echo "Processing: $line"
done < <(find /var/log -name "*.log" -mtime -1)

# Write to multiple destinations
tee >(gzip > backup.gz) >(sha256sum > backup.sha256) < data.raw > /dev/null
```

## Signal Handling (trap)

### Common Traps

```bash
#!/bin/bash

# Cleanup on exit
cleanup() {
    echo "Cleaning up..."
    rm -f "$TMPFILE"
    [[ -n "$PID" ]] && kill "$PID" 2>/dev/null
}
trap cleanup EXIT    # runs on any exit (normal or error)

# Handle Ctrl-C gracefully
trap 'echo "Interrupted"; exit 1' INT

# Handle TERM signal
trap 'echo "Terminated"; exit 1' TERM

# Ignore HUP (keep running after terminal close)
trap '' HUP

# Trap multiple signals
trap 'echo "Signal received"; cleanup; exit 1' INT TERM HUP

# Signal reference:
# SIGHUP  (1)  — terminal hangup
# SIGINT  (2)  — Ctrl-C
# SIGQUIT (3)  — Ctrl-\
# SIGTERM (15) — graceful termination
# SIGKILL (9)  — cannot be trapped
# EXIT        — pseudo-signal, any exit

TMPFILE=$(mktemp)
echo "Working with $TMPFILE"
# ... script logic ...
```

## File Locking (flock)

### Prevent Concurrent Execution

```bash
#!/bin/bash

# Method 1: flock wrapper (recommended)
LOCKFILE="/var/lock/myscript.lock"
exec 200>"$LOCKFILE"
flock -n 200 || { echo "Already running"; exit 1; }

# Script logic here...
echo "Running exclusively"
sleep 60
# Lock released automatically when script exits

# Method 2: flock subshell
(
    flock -n 9 || { echo "Already running"; exit 1; }
    # script logic
) 9>/var/lock/myscript.lock

# Method 3: flock command wrapper
flock -n /var/lock/myscript.lock /usr/local/bin/myscript.sh

# Timeout (wait up to 10 seconds for lock)
flock -w 10 /var/lock/myscript.lock /usr/local/bin/myscript.sh

# Shared lock (multiple readers, exclusive writers)
flock -s /var/lock/data.lock cat /data/file      # reader
flock -x /var/lock/data.lock update_data.sh       # writer
```

## Script Logging

### Logging Patterns

```bash
#!/bin/bash

LOGFILE="/var/log/myscript.log"

# Redirect all output to log and terminal
exec > >(tee -a "$LOGFILE") 2>&1

# Or log only (no terminal output)
exec >> "$LOGFILE" 2>&1

# Timestamped logging function
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [$1] ${*:2}"
}

log INFO "Script started"
log WARN "Disk usage above 80%"
log ERROR "Backup failed"

# Log rotation with logrotate
# /etc/logrotate.d/myscript
# /var/log/myscript.log {
#     daily
#     rotate 7
#     compress
#     missingok
#     notifempty
# }
```

## Error Handling

### set Options

```bash
#!/bin/bash
set -euo pipefail

# set -e: Exit immediately if a command exits with non-zero status
# set -u: Treat unset variables as an error
# set -o pipefail: Pipeline fails if any command in it fails

# Combined as shebang alternative:
#!/bin/bash -eu

# Safe patterns with set -e:

# Check return code without triggering -e
if ! grep -q "pattern" file; then
    echo "Pattern not found"
fi

# Allow specific commands to fail
command_that_might_fail || true

# Capture exit code
set +e
result=$(risky_command 2>&1)
rc=$?
set -e
if [[ $rc -ne 0 ]]; then
    echo "Failed with: $result"
fi
```

### Error Handling Patterns

```bash
#!/bin/bash
set -euo pipefail

# Error handler
on_error() {
    local line=$1
    local code=$2
    echo "ERROR on line $line (exit code $code)" >&2
}
trap 'on_error ${LINENO} $?' ERR

# Retry pattern
retry() {
    local max_attempts=$1
    shift
    local attempt=1
    while [[ $attempt -le $max_attempts ]]; do
        if "$@"; then
            return 0
        fi
        echo "Attempt $attempt/$max_attempts failed, retrying..." >&2
        ((attempt++))
        sleep $((attempt * 2))
    done
    return 1
}

retry 3 curl -sf http://example.com/health

# Validate prerequisites
require_command() {
    command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

require_command jq
require_command curl
```

## Script Security

### Best Practices

```bash
#!/bin/bash
set -euo pipefail

# Use absolute paths
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Secure temp files
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Validate input
validate_hostname() {
    local host="$1"
    if [[ ! "$host" =~ ^[a-zA-Z0-9._-]+$ ]]; then
        die "Invalid hostname: $host"
    fi
}

# Quote all variables (prevent word splitting and globbing)
cp "$source" "$dest"       # correct
# cp $source $dest         # WRONG — breaks on spaces

# Use [[ ]] instead of [ ] (no word splitting)
if [[ -f "$file" ]]; then
    echo "File exists"
fi

# Use arrays for commands with arguments
cmd=("rsync" "-avz" "--delete" "$source/" "$dest/")
"${cmd[@]}"

# Avoid eval
# eval "$user_input"       # NEVER — arbitrary code execution

# Drop privileges
if [[ $EUID -eq 0 ]]; then
    exec su -s /bin/bash nobody -c "$0 $*"
fi
```

## Common Automation Patterns

### Config File Processing

```bash
# Read key=value config
declare -A config
while IFS='=' read -r key value; do
    [[ "$key" =~ ^[[:space:]]*# ]] && continue   # skip comments
    [[ -z "$key" ]] && continue                    # skip empty
    key=$(echo "$key" | xargs)                     # trim
    value=$(echo "$value" | xargs)
    config["$key"]="$value"
done < /etc/myapp/config.conf
```

### Parallel Execution

```bash
# Run N tasks in parallel
MAX_PARALLEL=4
for host in "${hosts[@]}"; do
    while [[ $(jobs -rp | wc -l) -ge $MAX_PARALLEL ]]; do
        wait -n
    done
    ssh "$host" "uptime" &
done
wait

# Using xargs
echo "${hosts[@]}" | tr ' ' '\n' | xargs -P4 -I{} ssh {} "uptime"

# Using GNU parallel
parallel -j4 ssh {} uptime ::: "${hosts[@]}"
```

### Watchdog Script

```bash
#!/bin/bash
# Restart service if health check fails

HEALTH_URL="http://localhost:8080/health"
SERVICE="myapp"
MAX_FAILURES=3
failures=0

while true; do
    if curl -sf --max-time 5 "$HEALTH_URL" > /dev/null; then
        failures=0
    else
        ((failures++))
        echo "Health check failed ($failures/$MAX_FAILURES)"
        if [[ $failures -ge $MAX_FAILURES ]]; then
            echo "Restarting $SERVICE..."
            systemctl restart "$SERVICE"
            failures=0
        fi
    fi
    sleep 30
done
```

## See Also

- bash-scripting
- cron
- systemd-timers
- regex

## References

- man bash, crontab, at, flock, expect
- man systemd.timer, systemd.time
- GNU Bash Manual (gnu.org/software/bash/manual/)
- Advanced Bash-Scripting Guide (tldp.org)
