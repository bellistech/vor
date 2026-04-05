# Linux Automation and Scripting — Theory and Design

> *Automation transforms manual procedures into reliable, repeatable processes. Understanding the execution model, scheduling architecture, and design principles behind scripting ensures automation that is robust, idempotent, and maintainable.*

---

## 1. Cron Daemon Architecture

### How crond Works

The cron daemon (`crond`) runs as a persistent process that wakes up **every minute** to check for scheduled jobs:

```
crond starts → loads all crontabs into memory
    ↓
Every 60 seconds:
    ↓
Check current minute against all crontab entries
    ↓
For each matching entry:
    fork() → setuid(user) → exec(shell -c "command")
    ↓
Capture stdout/stderr → mail to MAILTO (if any output)
```

### Crontab Storage

| Type | Location | User Field |
|:---|:---|:---|
| Per-user | `/var/spool/cron/<username>` | No (implicit) |
| System | `/etc/crontab` | Yes (6th field) |
| Drop-in | `/etc/cron.d/*` | Yes (6th field) |
| Run-parts | `/etc/cron.{hourly,daily,weekly,monthly}/` | N/A (scripts) |

### Execution Environment

Cron jobs run in a **minimal environment**. Key differences from interactive shells:

| Variable | Interactive Shell | Cron |
|:---|:---|:---|
| PATH | Full user PATH | `/usr/bin:/bin` (minimal) |
| HOME | User's home | User's home |
| SHELL | User's login shell | `/bin/sh` |
| TERM | xterm, etc. | Not set |
| DISPLAY | Set if X11 | Not set |
| USER | Set | Set |

This is the #1 source of "works in terminal, fails in cron" issues.

### Timing Precision

Cron has **minute-level granularity**. The daemon checks once per minute, and all entries matching the current minute fire simultaneously.

$$resolution = 60 \text{ seconds}$$

For sub-minute scheduling, use systemd timers with `OnUnitActiveSec=` or a loop within a cron job.

### Anacron vs Cron

| Feature | cron | anacron |
|:---|:---|:---|
| Granularity | Minutes | Days |
| Requires uptime | Yes (misses jobs if off) | No (catches up on boot) |
| Daemon | crond (persistent) | Run on boot/daily |
| Use case | Servers (24/7) | Laptops, workstations |
| Timestamps | N/A | `/var/spool/anacron/` |

Anacron records the **last run date** for each job. On startup, it compares the current date against the last run date and executes any jobs whose period has elapsed.

---

## 2. systemd Timer vs Cron Comparison

### Feature Comparison

| Feature | cron | systemd timer |
|:---|:---|:---|
| Calendar scheduling | crontab syntax | OnCalendar (richer) |
| Monotonic scheduling | No | OnBootSec, OnUnitActiveSec |
| Missed run catch-up | anacron (daily) | Persistent=true (any granularity) |
| Randomized delay | No | RandomizedDelaySec |
| Dependencies | No | After=, Requires= |
| Resource limits | No | CPUQuota=, MemoryMax= |
| Logging | Mail or redirect | journalctl -u unit |
| Concurrency control | None (manual flock) | Built-in (one instance per timer) |
| Monitoring | `crontab -l`, check mail | `systemctl list-timers` |
| Transactional | No | Atomic enable/disable |
| User timers | Per-user crontab | `systemctl --user` |

### When to Use Which

**Use cron when:**
- Simple, well-understood scheduling
- Legacy systems, compatibility required
- Quick one-liners, temporary jobs

**Use systemd timers when:**
- Need resource limits (cgroups integration)
- Need dependency ordering
- Need persistent (catch-up) scheduling
- Want unified logging via journalctl
- Need randomized delay (thundering herd prevention)
- Want automatic concurrency protection

### Timer Accuracy

systemd timers have configurable accuracy:

$$actual\_time = scheduled\_time + random(0, AccuracySec)$$

Default `AccuracySec=1min`. For time-critical jobs:

```
[Timer]
AccuracySec=1us
```

### Persistent Timer Behavior

When `Persistent=true`, systemd stores the last trigger time. On boot:

$$\text{if } time_{now} - time_{last\_trigger} > period \implies \text{fire immediately}$$

This is equivalent to anacron but works at any granularity, not just daily.

---

## 3. Expect/TCL Automation Model

### How Expect Works

Expect is built on TCL (Tool Command Language) and operates by:

1. **Spawning** a child process (pseudo-terminal allocation)
2. **Expecting** specific patterns in the output (regex matching)
3. **Sending** input when patterns match

```
expect process ←→ pseudo-terminal (pty) ←→ child process
    ↕                                          ↕
pattern match                              stdin/stdout
send response                              (thinks it's interactive)
```

### The PTY Trick

The key innovation: expect allocates a **pseudo-terminal** (pty) for the child process. The child process sees a real terminal (not a pipe), so programs that check `isatty()` behave interactively:

- `ssh` prompts for passwords (not with pipes)
- `passwd` accepts input
- `sudo` prompts for credentials

### Limitations

| Limitation | Explanation |
|:---|:---|
| Timing-dependent | Patterns must match before timeout |
| Brittle | Output format changes break scripts |
| Security | Passwords often embedded in script |
| No parallelism | Sequential expect/send model |

### Alternatives

| Tool | Advantage | Use Case |
|:---|:---|:---|
| SSH keys | No passwords needed | Remote automation |
| Ansible expect module | Declarative, idempotent | Config management |
| pexpect (Python) | Full Python ecosystem | Complex automation |
| sshpass | Simple password piping | Quick scripts (less secure) |

---

## 4. Bash Execution Model

### Fork, Exec, and Subshells

Every external command in bash follows the **fork-exec** model:

```
Parent bash process
    ↓ fork()
Child process (copy of parent)
    ↓ exec()
Replaces child with external command
    ↓ wait()
Parent waits for child to exit
```

### Subshell Creation

Subshells are created by:

| Construct | Creates Subshell |
|:---|:---|
| `(commands)` | Yes — explicit subshell |
| `command \| command` | Yes — each pipeline stage |
| `$(command)` | Yes — command substitution |
| `<(command)` | Yes — process substitution |
| `command &` | Yes — background job |
| `{ commands; }` | **No** — group in current shell |

**Critical implication:** Variables set in a subshell are **not visible** in the parent:

```bash
echo "hello" | while read line; do
    var="$line"    # set in subshell (pipe)
done
echo "$var"        # empty! subshell is gone

# Fix: use process substitution
while read line; do
    var="$line"    # set in current shell
done < <(echo "hello")
echo "$var"        # "hello"
```

### Exit Code Propagation

```
Command exit code: $?
Pipeline exit code: last command (or PIPESTATUS array)
    with pipefail: first non-zero in pipeline
Subshell exit code: last command in subshell
Function return code: return N (0-255)
Script exit code: exit N or last command
```

$$\text{with } \texttt{set -o pipefail}: \quad exit_{pipeline} = \text{first non-zero } exit_i$$

### Builtin vs External Commands

Builtins execute **within the shell process** (no fork):

| Builtin | External Equivalent |
|:---|:---|
| `echo` | `/bin/echo` |
| `printf` | `/usr/bin/printf` |
| `test` / `[` | `/usr/bin/test` |
| `cd` | None (must be builtin) |
| `read` | None |
| `type` | `/usr/bin/type` |

Builtins are faster (no fork overhead) and can modify shell state (cd, export, read).

---

## 5. Signal Delivery and Handling

### Signal Delivery Model

Signals are **asynchronous notifications** delivered to processes:

```
Signal generated (kill, kernel, terminal)
    ↓
Delivered to target process
    ↓
Process signal disposition:
    ├── Default action (term, core, stop, ignore)
    ├── Custom handler (trap in bash)
    └── Ignored (trap '' SIGNAL)
```

### Signal Table

| Signal | Number | Default | Can Trap | Common Source |
|:---|:---:|:---|:---:|:---|
| SIGHUP | 1 | Terminate | Yes | Terminal close |
| SIGINT | 2 | Terminate | Yes | Ctrl-C |
| SIGQUIT | 3 | Core dump | Yes | Ctrl-\ |
| SIGKILL | 9 | Terminate | **No** | kill -9 |
| SIGTERM | 15 | Terminate | Yes | kill (default) |
| SIGSTOP | 19 | Stop | **No** | Ctrl-Z (via SIGTSTP) |
| SIGTSTP | 20 | Stop | Yes | Ctrl-Z |
| SIGCHLD | 17 | Ignore | Yes | Child exits |
| SIGUSR1 | 10 | Terminate | Yes | User-defined |
| SIGUSR2 | 12 | Terminate | Yes | User-defined |

### Signal Propagation in Process Groups

When Ctrl-C is pressed, SIGINT is sent to the **entire foreground process group**:

$$SIGINT \to \text{process group} = \{parent, child_1, child_2, ...\}$$

Background processes (`&`) are in a different process group and don't receive the signal.

### Trap Execution Order

```bash
trap 'handler_A' EXIT
trap 'handler_B' INT

# On Ctrl-C:
# 1. handler_B runs (INT trap)
# 2. If handler_B calls exit:
#    → handler_A runs (EXIT trap)
# 3. Process terminates

# EXIT trap ALWAYS runs on exit (normal, error, signal)
# Except: SIGKILL (cannot be trapped)
```

---

## 6. File Locking Theory

### Advisory vs Mandatory Locking

| Type | Enforcement | Linux Support |
|:---|:---|:---|
| Advisory | Cooperative — processes must check | `flock()`, `fcntl()` |
| Mandatory | Kernel-enforced — blocks I/O | Deprecated in Linux 5.15+ |

Advisory locking only works when **all processes** cooperate by checking the lock. A process that ignores the lock can freely access the file.

### flock vs fcntl

| Feature | flock | fcntl (POSIX) |
|:---|:---|:---|
| Granularity | Whole file | Byte range |
| NFS support | No | Yes |
| Inheritance | Shared across fork (same fd) | Per-process |
| API | Simple | Complex |
| Use case | Script mutex | Database, multi-process file I/O |

### Lock Types

$$\text{Shared (read) lock:} \quad N_{readers} \geq 0, N_{writers} = 0$$
$$\text{Exclusive (write) lock:} \quad N_{readers} = 0, N_{writers} = 1$$

Multiple shared locks can coexist. An exclusive lock requires no other locks.

### Deadlock Prevention

Two processes deadlock when each holds a lock the other needs:

$$P_1 \text{ holds } L_A, \text{ wants } L_B$$
$$P_2 \text{ holds } L_B, \text{ wants } L_A$$

Prevention strategies:
- **Lock ordering:** Always acquire locks in the same order
- **Timeout:** Use `flock -w N` to fail after N seconds
- **Non-blocking:** Use `flock -n` to fail immediately

---

## 7. Idempotent Script Design

### The Idempotency Principle

An operation is **idempotent** if running it multiple times produces the same result as running it once:

$$f(f(x)) = f(x)$$

### Idempotent Patterns

| Operation | Non-Idempotent | Idempotent |
|:---|:---|:---|
| Create directory | `mkdir /data` (fails if exists) | `mkdir -p /data` |
| Create user | `useradd bob` (fails if exists) | `id bob \|\| useradd bob` |
| Append to file | `echo "line" >> file` (duplicates) | `grep -q "line" file \|\| echo "line" >> file` |
| Set config | `echo "key=val" >> config` | `sed -i 's/^key=.*/key=val/' config` |
| Start service | `systemctl start svc` | Already idempotent (no-op if running) |
| Install package | `dnf install pkg` | Already idempotent (no-op if installed) |

### Guard Patterns

```bash
# State check before action
ensure_user_exists() {
    id "$1" &>/dev/null || useradd "$1"
}

# Atomic file replacement (never half-written)
update_config() {
    local tmpfile
    tmpfile=$(mktemp "${target}.XXXXXX")
    generate_config > "$tmpfile"
    mv "$tmpfile" "$target"    # atomic on same filesystem
}

# Marker files for one-time operations
run_once() {
    local marker="/var/lib/myapp/.migration_v2_done"
    [[ -f "$marker" ]] && return 0
    do_migration_v2
    touch "$marker"
}
```

---

## 8. Automation Anti-Patterns

### Common Mistakes

| Anti-Pattern | Problem | Fix |
|:---|:---|:---|
| No error handling | Silent failures | `set -euo pipefail` |
| Parsing `ls` output | Breaks on spaces, special chars | Use globs, `find -print0` |
| Unquoted variables | Word splitting, globbing | Always quote: `"$var"` |
| `cat file \| grep` | Useless use of cat | `grep pattern file` |
| Hardcoded paths | Breaks across environments | Config files, variables |
| No logging | Can't debug failures | Log to file with timestamps |
| No lock file | Race conditions, double-runs | `flock` |
| `eval "$user_input"` | Arbitrary code execution | Never eval untrusted input |
| `sleep` polling | Wastes resources, slow reaction | inotifywait, systemd path units |
| Password in script | Security vulnerability | Vault, keyring, env vars |

### The "Works on My Machine" Problem

Scripts that work interactively but fail in automation typically suffer from:

1. **Environment differences** — different PATH, missing variables
2. **Terminal assumptions** — interactive prompts, colors, tty checks
3. **Timing assumptions** — services not yet started, DNS not resolved
4. **Permission differences** — running as different user, no sudo
5. **Working directory** — script assumes specific cwd

---

## 9. Script Testing Methodology

### Testing Levels

| Level | What | Tool |
|:---|:---|:---|
| Syntax check | Parse errors | `bash -n script.sh` |
| Static analysis | Common bugs, style | `shellcheck script.sh` |
| Unit testing | Individual functions | bats-core, shunit2 |
| Integration testing | Full script execution | Docker containers |
| Idempotency testing | Run twice, same result | Automated re-run |

### ShellCheck

ShellCheck detects common bash pitfalls statically:

```bash
# Install
dnf install ShellCheck    # or: apt install shellcheck

# Run
shellcheck script.sh

# Common findings:
# SC2086 — Double quote to prevent globbing and word splitting
# SC2046 — Quote this to prevent word splitting
# SC2034 — Variable appears unused
# SC2155 — Declare and assign separately
# SC2164 — Use cd ... || exit in case cd fails
```

### bats-core Testing

```bash
# test_backup.bats
@test "backup creates file with timestamp" {
    run backup_database testdb /tmp
    [ "$status" -eq 0 ]
    [[ "$output" =~ /tmp/testdb_[0-9]+\.sql ]]
    [ -f "$output" ]
}

@test "backup fails on missing database" {
    run backup_database nonexistent /tmp
    [ "$status" -ne 0 ]
}

# Run tests
bats test_backup.bats
```

### Testing in Isolation

Use containers or VMs to test destructive scripts safely:

```bash
# Test in Docker
docker run --rm -v "$(pwd):/scripts:ro" centos:stream9 bash /scripts/deploy.sh

# Test with dry-run mode
DRY_RUN=1 ./deploy.sh
```

---

## References

- GNU Bash Manual (gnu.org/software/bash/manual/)
- Advanced Bash-Scripting Guide (tldp.org)
- ShellCheck wiki (github.com/koalaman/shellcheck/wiki)
- bats-core documentation (bats-core.readthedocs.io)
- man cron, crontab, anacrontab, at, flock, systemd.timer
- Expect documentation (expect.sourceforge.net)
- POSIX Shell Command Language (pubs.opengroup.org)
