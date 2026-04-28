# htop (interactive process viewer)

Interactive process viewer with color, tree view, and mouse support. Also covers top and btop basics.

## Launching

### Start htop

```bash
htop

# Show only processes for a user
htop -u deploy

# Show specific PID and its threads
htop -p 1234

# Start in tree mode
htop -t
```

## Navigation (Interactive Keys)

### Sorting

```bash
# F6 or >          Choose sort column
# P                Sort by CPU%
# M                Sort by MEM%
# T                Sort by TIME+
# I                Invert sort order
```

### Filtering and Searching

```bash
# F3 or /          Incremental search by name
# F4 or \          Filter — only show matching processes
# Esc              Clear filter
```

### Tree View

```bash
# F5 or t          Toggle tree/list view
# + / -            Expand/collapse tree branches (in tree mode)
```

### Process Actions

```bash
# F9 or k          Kill process (select signal from menu)
# F7 / F8          Renice: decrease/increase nice value
# s                Strace the selected process
# l                lsof the selected process
# Space            Tag process (for batch kill)
# U                Untag all
# F2               Setup (configure columns, colors, meters)
```

## Setup and Columns

### F2 Setup Menu

```bash
# Columns:  Add/remove PID, USER, CPU%, MEM%, IO_READ, IO_WRITE, etc.
# Meters:   Configure header — CPU bars, memory, load, uptime, hostname
# Display:  Show threads, kernel threads, custom thread names
# Colors:   Choose color scheme
```

## top Basics

### Common top Usage

```bash
top

# Batch mode (for scripting)
top -b -n 1

# Sort by memory
top -o %MEM

# Show specific user
top -u deploy
```

### top Interactive Keys

```bash
# 1         Toggle per-CPU display
# c         Show full command path
# H         Toggle threads
# k         Kill a process (type PID then signal)
# M         Sort by memory
# P         Sort by CPU
# q         Quit
```

## btop Basics

### btop Usage

```bash
btop

# btop has a TUI with tabs for CPU, memory, network, disk
# Navigate with arrow keys or mouse
# Esc for options menu
# q to quit
```

## Tips

- htop reads from `/proc` and requires no special privileges, but renice and kill need appropriate permissions.
- `F2` setup is saved to `~/.config/htop/htoprc` and persists across sessions.
- htop's filter (`F4`) is more useful than search (`F3`) -- filter hides non-matching lines.
- In tree mode, killing a parent sends the signal to that process only (children may be reparented to init).
- `htop -C` starts without color (useful for screen readers or monochrome terminals).
- btop is the modern successor with GPU, disk, and network panels but is not installed by default.

## Complete Key Reference

The full table of htop keybindings (from `htop --help` and the man page):

```
Navigation
  ↑/k        previous process
  ↓/j        next process
  PgUp       page up
  PgDn       page down
  Home       jump to first process
  End        jump to last process
  ←/→        scroll columns horizontally
  H          toggle hide/show user threads
  K          toggle hide/show kernel threads
  J          jump to highlighted process's thread group
  Ctrl-L     redraw screen

Sorting
  F6 or >    choose sort column from a menu
  P          sort by CPU%
  M          sort by MEM%
  T          sort by TIME+
  N          sort by PID
  I          invert sort order
  +/-        expand/collapse a tree branch (tree mode)

Tree / Affinity
  F5 or t    toggle tree view
  Ctrl+f     forward to /proc tree
  Shift+M    set CPU affinity (taskset live; needs root for other users)

Filter / Search
  F3 or /    incremental search by command name
  F4 or \    persistent filter (hide non-matching)
  n          next match
  N          previous match
  Esc        clear search/filter
  u          show only one user's processes (prompts for username)

Tag / Multi-select
  Space      tag/untag the highlighted process
  c          tag process and all its children
  U          untag all

Action
  F7         decrease nice value (NEEDS ROOT for negative)
  F8         increase nice value
  F9 or k    send signal (menu — TERM, KILL, HUP, USR1, USR2, etc.)
  F2         setup screen (columns, meters, colors)
  F1 or h    help screen
  F10 or q   quit
  s          strace the highlighted process
  l          lsof the highlighted process (open files / sockets)
  L          show recent FDs the process touched (htop 3.x)
  Shift+E    show environment variables of the process
  Shift+I    invert sort
  Shift+J    set scheduling policy (SCHED_OTHER, FIFO, RR, BATCH, IDLE)

Display Toggles (one-shot)
  z          pause refresh (toggle)
  x          show in megabytes (some columns)
```

## Setup Screen (F2) Walkthrough

The four sub-screens of `F2 Setup`:

### Meters

The colored bars at the top. Each meter can be left/right column, with sub-style:

- **Bar** — single bar with %
- **Text** — numeric values
- **Graph** — small ASCII graph over time
- **LED** — minimal indicator

Common meter additions:
- `Battery` (laptop)
- `Hostname` (so the screenshot tells you which host)
- `LoadAverage` (1/5/15 min)
- `Temperature` (CPU thermal sensor; needs `lm-sensors`)
- `Memory` separate from `Swap`
- `Tasks` (running / sleeping / stopped count)
- `DiskIO` (htop 3+)
- `NetworkIO` (htop 3+)

### Display Options

Toggles you'll actually flip:

- `Tree view is always sorted by PID (htop 3+)` — recommended for stable tree layout
- `Show kernel threads` — off by default; enable when chasing kthread CPU
- `Hide userland process threads` — off by default; enabling collapses Java/Postgres etc.
- `Show CPU usage on Meters in colors` — yes
- `Show program path` — yes (full /usr/bin/foo not just foo)
- `Highlight base name of program` — yes
- `Highlight large numbers` — yes
- `Detailed CPU time (System/IO-Wait/Hard-IRQ/Soft-IRQ/Steal/Guest)` — yes for kernel debugging
- `Show custom thread names` — yes (matches what `prctl(PR_SET_NAME)` set)
- `Update process names on every refresh` — costly on busy boxes

### Columns

Add/remove and reorder. Useful additions:

- `IO_READ_RATE` / `IO_WRITE_RATE` — per-process disk bytes (needs `/proc/<pid>/io`, kernel >= 2.6.20)
- `IO_PRIORITY` — ionice class
- `OOM_SCORE` — likelihood of being OOM-killed (higher = first)
- `CTID` — container ID (LXC / OpenVZ)
- `CGROUP` — full cgroup path of the process
- `PROCESSOR` — last CPU run on
- `STARTTIME` — wall-clock start time
- `M_VIRT` / `M_RESIDENT` / `M_SHARE` — memory breakdown

### Colors

Six built-in schemes:
- Default
- Monochromatic (no color)
- Black on White
- Light Terminal
- MC (Midnight Commander)
- Black Night
- Broken Gray (htop 3+)

You can also hand-edit `~/.config/htop/htoprc` for custom colors not exposed in the UI.

## Signal Cheat Sheet (F9 menu)

When you hit F9 or `k`, htop shows a signal menu. The most useful:

| # | Name | Effect |
|---|------|--------|
| 1 | SIGHUP | "config changed" — many daemons reload, others die |
| 2 | SIGINT | Ctrl-C equivalent; graceful interrupt |
| 3 | SIGQUIT | core dump (if enabled), used to be Ctrl-\ |
| 9 | SIGKILL | uncatchable; immediate kill — last resort |
| 15 | SIGTERM | default; graceful, "please stop" |
| 17 | SIGSTOP | pause (uncatchable) |
| 18 | SIGCONT | resume from STOP |
| 19 | SIGTSTP | Ctrl-Z equivalent; can be caught |

Use Space to tag multiple processes, then F9 once for the same signal to all.

## htoprc — Persistent Config

```bash
~/.config/htop/htoprc

# A few interesting keys you might want to tweak by hand:
fields=0 48 17 18 38 39 40 2 46 47 49 1   # column IDs in display order
sort_key=46                                # 46=PERCENT_CPU
sort_direction=-1                          # -1 = descending
hide_kernel_threads=1
hide_userland_threads=0
shadow_other_users=0
show_thread_names=1
show_program_path=1
highlight_base_name=1
highlight_megabytes=1
tree_view=0
header_margin=1
detailed_cpu_time=1
cpu_count_from_one=0
update_process_names=0
account_guest_in_cpu_meter=0
color_scheme=0          # 0=Default, 1=Monochromatic, 2=Black-on-White,
                         # 3=Light Terminal, 4=MC, 5=Black Night, 6=Broken Gray
delay=15                # refresh tenths of a second (15 = 1.5s)
left_meters=AllCPUs Memory Swap
left_meter_modes=1 1 1
right_meters=Tasks LoadAverage Uptime
right_meter_modes=2 2 2
```

## top Basics

### Common top Usage

```bash
top

# Batch mode (for scripting)
top -b -n 1

# Sort by memory
top -o %MEM

# Show specific user
top -u deploy

# One snapshot to a file (every column)
top -b -n 1 -w 512 > /tmp/top.txt

# Only processes matching a name
top -p $(pgrep -d, postgres)
```

### top Interactive Keys

```bash
1         toggle per-CPU display
c         show full command path
e         scale memory units (KB → MB → GB → TB)
H         toggle threads
k         kill a process (type PID then signal)
M         sort by memory
P         sort by CPU
T         sort by TIME+
W         write current configuration to ~/.toprc
V         tree view
Z         change color scheme
f         field management — like htop's F2 Columns
o         add a filter (e.g. COMMAND=httpd)
q         quit
```

`~/.toprc` persists settings between runs.

## btop Basics

`btop` is the modern Rust-rewritten cousin of htop. Better aesthetics, more meters out of the box, GPU support.

```bash
btop

# Common keys:
m         menu
o         options
M         show memory
P         show processes
N         show network
t         show CPU
+/-       sort by next/prev column
f         filter
h         help (full key list)
q         quit
```

Config: `~/.config/btop/btop.conf`. Themes: `/usr/share/btop/themes/`. Multiple panels can be shown at once via `o` → preset config.

## Worked Comparisons

### "What's eating my CPU?"

```bash
htop          # F6 → choose CPU%, then watch the top of the list
# OR
top -o %CPU
# OR scripted:
ps aux --sort=-%cpu | head -5
```

### "What's eating my memory?"

```bash
htop          # F6 → choose MEM%
# Look at RES (resident set), not VIRT (virtual address space).
# VIRT can be huge for processes that mmap large files without faulting them in.

# Watch RSS over time
ps -eo pid,rss,comm --sort=-rss | head -10
watch -n 1 'ps -eo pid,rss,comm --sort=-rss | head -10'
```

### "What's hammering disk?"

```bash
# htop won't tell you — use iotop or pidstat
sudo iotop -o -P -d 1
sudo pidstat -d 1 5
```

### "Why is my system so laggy?"

```bash
# Step through layer by layer:
htop                  # CPU%, MEM%, swap usage
sudo iotop -o -P      # disk pressure
vmstat 1 5            # swap, IO wait, blocked processes
sar -P ALL 1 5        # per-CPU breakdown over time
```

## Common Errors and Fixes

```bash
# "htop: command not found"
sudo apt install htop                 # Debian/Ubuntu
sudo dnf install htop                 # RHEL/Fedora
sudo apk add htop                     # Alpine
brew install htop                     # macOS

# F-keys don't work (e.g. SSH from terminal that intercepts F-keys)
# Fix: use the letter shortcut: t for tree, k for kill, q for quit, etc.
#      Most F-keys have letter equivalents documented in `htop --help`.

# Colors look wrong / monochrome
# Fix: ensure $TERM is set correctly (typically xterm-256color)
echo $TERM
TERM=xterm-256color htop

# Columns truncated on narrow terminal
# Fix: F2 → Columns → remove ones you don't need; or just resize wider.

# Process renice fails: "Operation not permitted"
# Fix: htop sees what it sees — root needed for negative nice or other users.
sudo htop

# "couldn't open /proc/<pid>/io" — io stats columns blank
# Cause: container without /proc:rw (Docker default). Or kernel built without
#        CONFIG_TASK_IO_ACCOUNTING.
# Fix: docker run --pid=host or grant CAP_SYS_PTRACE; OR rebuild the kernel.

# btop crashes on launch with "Could not init terminal"
# Cause: locale/terminal mismatch.
# Fix: LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8 btop
```

## Tips

- htop reads from `/proc` and requires no special privileges, but renice and kill need appropriate permissions.
- `F2` setup is saved to `~/.config/htop/htoprc` and persists across sessions.
- htop's filter (`F4`) is more useful than search (`F3`) — filter hides non-matching lines.
- In tree mode, killing a parent sends the signal to that process only (children may be reparented to init).
- `htop -C` starts without color (useful for screen readers or monochrome terminals).
- btop is the modern successor with GPU, disk, and network panels but is not installed by default.
- If running over SSH, set `TERM=xterm-256color` for full color/Unicode (some pre-2010 terminfo entries lack it).
- `htop -d 50` increases refresh interval to 5 seconds — useful when the htop process itself is the load.
- Alongside htop, `iotop` for disk and `nethogs` for per-process network usage round out the live-perf trio.
- Never use SIGKILL (9) as a first attempt — many programs have shutdown hooks that flush data on SIGTERM (15) but not on KILL.

## See Also

- system/iostat, system/vmstat, system/lsof, troubleshooting/linux-errors, process/nice

## References

- [man htop(1)](https://man7.org/linux/man-pages/man1/htop.1.html)
- [man top(1)](https://man7.org/linux/man-pages/man1/top.1.html)
- [man proc(5) — /proc filesystem](https://man7.org/linux/man-pages/man5/proc.5.html)
- [htop Project Site](https://htop.dev/)
- [htop GitHub — README and FAQ](https://github.com/htop-dev/htop)
- [Arch Wiki — htop](https://wiki.archlinux.org/title/Htop)
- [Kernel /proc Documentation](https://www.kernel.org/doc/html/latest/filesystems/proc.html)
- [Ubuntu Manpage — htop](https://manpages.ubuntu.com/manpages/noble/man1/htop.1.html)
