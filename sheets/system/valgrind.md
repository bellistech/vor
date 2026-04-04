# Valgrind (Dynamic Analysis Framework)

> Detect memory leaks, profile CPU and heap usage, and find threading errors in compiled programs using instrumented execution.

## Concepts

### Tools Overview

```
# memcheck   — memory error detector (default tool)
#               invalid reads/writes, use of uninitialized values, leaks
# callgrind  — call-graph profiling (instruction-level CPU profiling)
# cachegrind — cache and branch-prediction profiling
# massif     — heap memory profiler (allocation over time)
# helgrind   — thread error detector (races, lock order violations)
# drd        — alternative thread error detector
# lackey     — example tool for basic block/instruction counting
```

## Memcheck

### Basic Usage

```bash
# Run with default settings (memcheck)
valgrind ./myprogram arg1 arg2

# Full leak check with source locations
valgrind --leak-check=full --show-leak-kinds=all ./myprogram

# Track origins of uninitialized values (slower but more informative)
valgrind --leak-check=full --track-origins=yes ./myprogram

# Show reachable blocks (memory still pointed to at exit)
valgrind --leak-check=full --show-reachable=yes ./myprogram

# Generate XML output for CI integration
valgrind --xml=yes --xml-file=valgrind-report.xml ./myprogram

# Log to file instead of stderr
valgrind --log-file=valgrind.log --leak-check=full ./myprogram
```

### Interpreting Output

```
# Error types memcheck reports:

# Invalid read/write — accessing freed or out-of-bounds memory
#   Invalid read of size 4
#     at 0x4005F2: process_data (main.c:42)
#     Address 0x5204048 is 8 bytes inside a block of size 10 free'd

# Conditional jump on uninitialized value
#   Conditional jump or move depends on uninitialised value(s)
#     at 0x400610: check_flag (main.c:55)

# Memory leak categories:
#   "definitely lost"   — no pointer to block exists (real leak)
#   "indirectly lost"   — lost because the pointer to it was lost
#   "possibly lost"     — pointer exists but points to interior of block
#   "still reachable"   — pointer exists at exit (usually not a bug)

# LEAK SUMMARY:
#   definitely lost: 72 bytes in 3 blocks
#   indirectly lost: 0 bytes in 0 blocks
#   possibly lost: 0 bytes in 0 blocks
#   still reachable: 4,096 bytes in 1 blocks
#   suppressed: 0 bytes in 0 blocks

# Exit code: valgrind returns the program's exit code by default
# Use --error-exitcode=1 to return 1 if errors found (useful for CI)
valgrind --error-exitcode=1 --leak-check=full ./myprogram
```

### Common Flags

```bash
# Useful flag combinations
valgrind \
    --tool=memcheck \
    --leak-check=full \                  # full leak detail
    --show-leak-kinds=all \              # definite, indirect, possible, reachable
    --track-origins=yes \                # trace uninitialized values
    --track-fds=yes \                    # warn on unclosed file descriptors
    --num-callers=20 \                   # deeper stack traces (default 12)
    --verbose \                          # extra diagnostic info
    --suppressions=myapp.supp \          # apply suppression file
    ./myprogram

# Run with GDB server for interactive debugging
valgrind --vgdb=yes --vgdb-error=1 ./myprogram
# Then in another terminal: gdb ./myprogram -ex "target remote | vgdb"
```

## Callgrind

### CPU Profiling

```bash
# Profile with callgrind
valgrind --tool=callgrind ./myprogram

# Output: callgrind.out.<pid>

# Disable instrumentation at start, toggle with callgrind_control
valgrind --tool=callgrind --instr-atstart=no ./myprogram
# In another terminal:
callgrind_control -i on                  # start collecting
callgrind_control -i off                 # stop collecting

# Collect cache simulation data too
valgrind --tool=callgrind --cache-sim=yes ./myprogram

# Annotate source with costs
callgrind_annotate callgrind.out.12345
callgrind_annotate --auto=yes callgrind.out.12345   # annotate all source files

# Visualize with KCachegrind (GUI)
kcachegrind callgrind.out.12345          # Linux
qcachegrind callgrind.out.12345          # macOS (brew install qcachegrind)
```

## Massif

### Heap Profiling

```bash
# Profile heap allocations over time
valgrind --tool=massif ./myprogram

# Output: massif.out.<pid>

# Include stack memory in profiling
valgrind --tool=massif --stacks=yes ./myprogram

# Set snapshot frequency
valgrind --tool=massif --time-unit=B ./myprogram   # bytes allocated (not instructions)

# Visualize with ms_print
ms_print massif.out.12345

# Output shows an ASCII chart of heap usage over time:
#     MB
# 12.5^                                     #
#     |                                   ##:
#     |                              @@@###::
#     |                         @@@@@:: :::::
#     |                   @@@@@@::::::  :::::
#     |              @@@@@:::::::::::   :::::
#     0 +--------------------------------------------> time

# massif-visualizer for GUI (if available)
massif-visualizer massif.out.12345
```

## Helgrind

### Thread Error Detection

```bash
# Detect data races and lock misuse
valgrind --tool=helgrind ./myprogram

# Common errors reported:
#   - Data race: two threads access same memory, at least one writes, no sync
#   - Lock order violation: potential deadlock from inconsistent lock ordering
#   - Misuse of pthreads API: destroying locked mutex, unlocking unowned mutex

# Increase history for better race reports
valgrind --tool=helgrind --history-level=full ./myprogram

# Alternative: DRD (sometimes finds different races)
valgrind --tool=drd ./myprogram
```

## Suppression Files

### Creating and Using Suppressions

```bash
# Generate suppressions for known issues
valgrind --leak-check=full --gen-suppressions=all ./myprogram 2>&1 | \
    grep -A 20 '{' > generated.supp

# Apply suppressions
valgrind --suppressions=/usr/lib/valgrind/default.supp \
         --suppressions=myapp.supp \
         --leak-check=full ./myprogram
```

### Suppression File Format

```
# myapp.supp
{
   known_libc_leak
   Memcheck:Leak
   match-leak-kinds: reachable
   fun:malloc
   fun:_dl_signal_error
   ...
}

{
   ignore_openssl_init
   Memcheck:Leak
   match-leak-kinds: reachable
   fun:malloc
   ...
   fun:SSL_library_init
}

{
   ignore_cond_in_zlib
   Memcheck:Cond
   fun:inflateReset2
   fun:inflateInit2_
}

# Suppression types:
#   Memcheck:Leak        — memory leaks
#   Memcheck:Cond        — conditional on uninitialized
#   Memcheck:Value8      — uninitialized value (8 bytes)
#   Memcheck:Addr4       — invalid address (4 byte access)
#   Memcheck:Param       — uninitialized syscall param
```

## Build Recommendations

### Compiler Flags for Best Results

```bash
# Compile with debug info and no optimization for accurate line numbers
gcc -g -O0 -o myprogram myprogram.c

# -g    — include debug symbols (DWARF)
# -O0   — no optimization (source lines match execution)
# -O1   — acceptable; some variables may be optimized away

# For C++ with STL containers, consider:
gcc -g -O0 -fno-inline -o myprogram myprogram.cpp

# Valgrind slows execution ~10-50x (memcheck)
# Callgrind slows ~20-100x; massif ~20x; helgrind ~100x
```

## Tips

- Always compile with `-g -O0` for meaningful stack traces and line numbers.
- Use `--error-exitcode=1` in CI pipelines to fail builds on memory errors.
- Start with memcheck defaults, then add `--track-origins=yes` only when investigating uninitialized value errors (it is significantly slower).
- Use suppressions for known library issues rather than ignoring all errors.
- Callgrind + KCachegrind is one of the most powerful free profiling combos for C/C++.
- Valgrind does not support macOS well on newer versions; prefer Linux or use `leaks`/Instruments on macOS.
- For multi-threaded programs, run both helgrind and drd -- they use different algorithms and catch different issues.
- Use `RUNNING_ON_VALGRIND` macro from `valgrind/valgrind.h` to detect instrumented execution at runtime.

## See Also

- gdb, strace, perf, bpftrace, htop

## References

- [man valgrind(1)](https://man7.org/linux/man-pages/man1/valgrind.1.html)
- [Valgrind User Manual](https://valgrind.org/docs/manual/)
- [Memcheck Manual](https://valgrind.org/docs/manual/mc-manual.html)
- [Callgrind Manual](https://valgrind.org/docs/manual/cl-manual.html)
- [Massif Manual](https://valgrind.org/docs/manual/ms-manual.html)
- [Helgrind Manual](https://valgrind.org/docs/manual/hg-manual.html)
- [DRD Manual](https://valgrind.org/docs/manual/drd-manual.html)
- [Valgrind Core Options](https://valgrind.org/docs/manual/manual-core.html)
- [Valgrind Suppression Files](https://valgrind.org/docs/manual/manual-core.html#manual-core.suppress)
- [Arch Wiki — Valgrind](https://wiki.archlinux.org/title/Debugging#Valgrind)
- [KCachegrind — Callgrind Visualizer](https://kcachegrind.github.io/)
