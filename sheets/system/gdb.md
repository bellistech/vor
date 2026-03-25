# GDB (GNU Debugger)

> Source-level debugger for C, C++, and other compiled languages; inspect running programs, set breakpoints, and analyze core dumps.

## Starting GDB

### Launch Options

```bash
# Debug an executable
gdb ./program

# With arguments
gdb --args ./program arg1 arg2

# Attach to running process
gdb -p <pid>
gdb ./program <pid>

# Load a core dump
gdb ./program core

# Quiet mode (skip banner)
gdb -q ./program

# Execute commands on start
gdb -ex "break main" -ex "run" ./program
```

### Compile for Debugging

```bash
# Include debug symbols, disable optimization
gcc -g -O0 -o program main.c

# Debug + address sanitizer
gcc -g -O0 -fsanitize=address -o program main.c
```

## Running

### Run and Control

```
run                  # Start program (with args if set)
run arg1 arg2        # Start with arguments
set args arg1 arg2   # Set arguments for next run
show args            # Show current arguments
kill                 # Kill running program
quit                 # Exit GDB (or Ctrl+D)
```

## Breakpoints

### Setting Breakpoints

```
break main                    # Break at function
break file.c:42               # Break at file:line
break file.c:42 if x > 10    # Conditional breakpoint
break *0x00400520             # Break at address
tbreak main                   # Temporary breakpoint (hit once)
rbreak ^process_              # Regex breakpoint (all funcs matching)
```

### Managing Breakpoints

```
info breakpoints              # List all breakpoints
info break                    # Short form
disable 2                     # Disable breakpoint 2
enable 2                      # Enable breakpoint 2
delete 2                      # Delete breakpoint 2
delete                        # Delete all breakpoints
clear file.c:42               # Clear breakpoint at location
condition 2 x == 5            # Add condition to existing breakpoint
ignore 2 10                   # Skip breakpoint 2 for 10 hits
commands 2                    # Run commands when breakpoint 2 hits
  print x
  continue
end
```

### Watchpoints

```
watch x                       # Break when x changes (write)
rwatch x                      # Break when x is read
awatch x                      # Break on read or write
watch *(int *)0x7fffffffe000  # Watch memory address
info watchpoints              # List watchpoints
```

### Catchpoints

```
catch throw                   # Break on C++ throw
catch catch                   # Break on C++ catch
catch syscall write           # Break on system call
catch signal SIGSEGV          # Break on signal
```

## Stepping

### Step Controls

```
next         (n)    # Step over (execute line, skip into functions)
step         (s)    # Step into (enter function calls)
finish       (fin)  # Run until current function returns
continue     (c)    # Continue to next breakpoint
until 50            # Run until line 50
advance file.c:50   # Continue to specific location
stepi        (si)   # Step one machine instruction
nexti        (ni)   # Step over one machine instruction
```

## Examining State

### Print and Display

```
print x              (p x)     # Print variable
print *ptr                     # Dereference pointer
print arr[0]@10                # Print 10 elements of array
print/x val                    # Print in hex
print/t val                    # Print in binary
print/o val                    # Print in octal
print/d val                    # Print as signed decimal
print/u val                    # Print as unsigned decimal
print/c val                    # Print as character
print/s ptr                    # Print as string
print/f val                    # Print as float
print (struct foo)*ptr         # Cast and print

# Auto-display (print on every stop)
display x                     # Add auto-display
display/x val                 # Auto-display in hex
undisplay 1                   # Remove display 1
info display                  # List auto-displays
```

### Backtrace

```
backtrace        (bt)    # Full call stack
bt full                  # Stack with local variables
bt 5                     # Top 5 frames
frame 3          (f 3)   # Select frame 3
up                       # Move up one frame
down                     # Move down one frame
info frame               # Detailed frame info
info locals              # Local variables in current frame
info args                # Function arguments in current frame
```

### Memory Examination

```
x/10xw 0x7fffffffe000    # 10 words in hex
x/20xb ptr               # 20 bytes in hex
x/s str                  # As null-terminated string
x/10i $pc                # 10 instructions at program counter

# Format: x/NFU address
# N = count, F = format (x,d,u,o,t,a,c,f,s,i), U = unit (b,h,w,g)
```

### Source Code

```
list                     # Show source around current line
list 42                  # Show source around line 42
list main                # Show source of function
list file.c:42           # Show source at file:line
list -                   # Show previous lines
set listsize 30          # Change number of lines shown
```

## TUI Mode

### Text User Interface

```
# Launch in TUI mode
gdb -tui ./program

# Toggle TUI in session
tui enable
tui disable
Ctrl+X A                 # Toggle TUI mode
Ctrl+X 2                 # Second window (assembly or registers)

# TUI layouts
layout src               # Source only
layout asm               # Assembly only
layout split             # Source + assembly
layout regs              # Source + registers

# Refresh display (if garbled)
Ctrl+L
refresh
```

## Core Dumps

### Analyze Core Dumps

```bash
# Enable core dumps
ulimit -c unlimited

# Generate core dump of running process
gcore <pid>

# Analyze in GDB
gdb ./program core
```

```
# In GDB with core loaded
bt                       # See crash backtrace
frame 0                  # Go to crash frame
info registers           # Register state at crash
print *ptr               # Examine variables
```

## Attach to Process

```bash
# Attach by PID
gdb -p 12345

# Or from within GDB
attach 12345
detach
```

```bash
# If attach fails (ptrace protection)
echo 0 | sudo tee /proc/sys/kernel/yama/ptrace_scope
```

## Pretty Printing

```
# Enable pretty printing for STL containers
set print pretty on
set print array on
set print array-indexes on
set print elements 0          # Print all elements (no truncation)

# Python pretty printers (usually auto-loaded for libstdc++)
info pretty-printer
```

## Remote Debugging

### GDB Server

```bash
# On target machine
gdbserver :1234 ./program
gdbserver :1234 --attach <pid>

# On host machine
gdb ./program
target remote 192.168.1.100:1234
```

## Configuration

### .gdbinit

```
# ~/.gdbinit — loaded on GDB startup
set history save on
set history size 10000
set print pretty on
set pagination off
set confirm off
set disassembly-flavor intel

# Auto-load project .gdbinit
set auto-load safe-path /home/user/project
```

## Tips

- Compile with `-g -O0` for best debugging experience; optimization reorders and eliminates code.
- Use `info threads` and `thread <n>` to switch between threads in multi-threaded programs.
- `set follow-fork-mode child` to debug child process after fork.
- Use `record` and `reverse-next`/`reverse-step` for reverse debugging (execution recording).
- Save breakpoints with `save breakpoints bp.txt` and reload with `source bp.txt`.
- Use `signal 0` to continue without delivering a signal when stopped by one.
- GDB supports Python scripting for custom commands: `python print(gdb.parse_and_eval("x"))`.

## References

- [man gdb(1)](https://man7.org/linux/man-pages/man1/gdb.1.html)
- [GDB User Manual](https://sourceware.org/gdb/current/onlinedocs/gdb/)
- [GDB Quick Reference Card](https://sourceware.org/gdb/current/onlinedocs/gdb/Summary.html)
- [GDB Command Index](https://sourceware.org/gdb/current/onlinedocs/gdb/Command-and-Variable-Index.html)
- [GDB Python API](https://sourceware.org/gdb/current/onlinedocs/gdb/Python-API.html)
- [GDB Remote Debugging](https://sourceware.org/gdb/current/onlinedocs/gdb/Remote-Debugging.html)
- [GDB to LLDB Command Map](https://lldb.llvm.org/use/map.html)
- [Arch Wiki — Debugging](https://wiki.archlinux.org/title/Debugging)
- [Red Hat — GDB Debugging Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/developing_c_and_cpp_applications_in_rhel_9/debugging-applications_developing-applications)
- [Kernel GDB Scripts](https://www.kernel.org/doc/html/latest/dev-tools/gdb-kernel-debugging.html)
