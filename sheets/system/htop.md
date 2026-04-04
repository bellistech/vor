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

## See Also

- ps, vmstat, iostat, strace, lsof, kill

## References

- [man htop(1)](https://man7.org/linux/man-pages/man1/htop.1.html)
- [man top(1)](https://man7.org/linux/man-pages/man1/top.1.html)
- [man proc(5) — /proc filesystem](https://man7.org/linux/man-pages/man5/proc.5.html)
- [htop Project Site](https://htop.dev/)
- [htop GitHub — README and FAQ](https://github.com/htop-dev/htop)
- [Arch Wiki — htop](https://wiki.archlinux.org/title/Htop)
- [Kernel /proc Documentation](https://www.kernel.org/doc/html/latest/filesystems/proc.html)
- [Ubuntu Manpage — htop](https://manpages.ubuntu.com/manpages/noble/man1/htop.1.html)
