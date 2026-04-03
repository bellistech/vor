# The Theory of Make — Dependency Graphs, Topological Sort, and Evaluation

> *Make is a build automation tool based on directed acyclic graphs (DAGs). It performs topological sorting of dependency edges, compares file modification timestamps to determine staleness, and uses pattern rules as a rewrite system. Understanding Make means understanding graph theory and term rewriting.*

---

## 1. The Dependency DAG

### Formalization

A Makefile defines a **directed acyclic graph** $G = (V, E)$ where:
- $V$ = set of targets and prerequisites (files or phony names)
- $E$ = directed edges from prerequisite to target: $(p, t)$ means "$t$ depends on $p$"

A rule `target: prereq1 prereq2` creates edges:

$$(\text{prereq1}, \text{target}), \quad (\text{prereq2}, \text{target})$$

### Worked Example

```makefile
app: main.o utils.o
	$(CC) -o app main.o utils.o

main.o: main.c defs.h
	$(CC) -c main.c

utils.o: utils.c defs.h
	$(CC) -c utils.c
```

The DAG:
```
  main.c   defs.h   utils.c
    │        │ │       │
    ▼        ▼ ▼       ▼
  main.o ◄───┘ └──► utils.o
    │                  │
    └────► app ◄───────┘
```

---

## 2. Build Decision — Timestamp Comparison

### The Staleness Rule

A target $t$ is **stale** (needs rebuilding) if:

$$\exists \ p \in \text{prereqs}(t) : \text{mtime}(p) > \text{mtime}(t)$$

Or if $t$ does not exist.

### The Algorithm

```
function build(target):
    for each prerequisite p of target:
        build(p)                          // recursive — depth-first
    if target does not exist OR
       any prereq is newer than target:
        execute recipe
```

This is a **post-order depth-first traversal** of the DAG.

### Timestamp Resolution

File system timestamp granularity matters:

| Filesystem | Resolution | Problem |
|:-----------|:-----------|:--------|
| ext4 | 1 nanosecond | No issue |
| HFS+ (macOS) | 1 second | Fast builds may miss changes |
| FAT32 | 2 seconds | Frequent false negatives |

If two files have the same mtime, Make considers the target **up to date** (not stale).

---

## 3. Topological Sort and Build Order

### Definition

A **topological sort** of a DAG is a linear ordering of vertices such that for every edge $(u, v)$, $u$ comes before $v$.

### Make's Build Order

Make doesn't explicitly topologically sort. Instead, the recursive depth-first build function implicitly produces a valid topological order. The first target in the Makefile is the **default goal**.

### Parallel Make (`-j`)

With `make -j N`, Make can execute recipes for independent targets simultaneously. Two targets are **independent** if neither is an ancestor of the other in the DAG.

Maximum parallelism is bounded by the **width** of the DAG:

$$\text{max\_parallel} = \max_{\text{level}} |\{v : \text{depth}(v) = \text{level}\}|$$

### Cycle Detection

If the dependency graph has a cycle, Make reports an error. A cycle means no valid topological sort exists:

```makefile
a: b
b: a
# ERROR: circular dependency
```

---

## 4. Variable Expansion — Two Flavors

### Recursive vs Simple Variables

| Syntax | Name | Expansion Time | Behavior |
|:-------|:-----|:---------------|:---------|
| `VAR = value` | Recursive | At **use** time | Re-evaluated each time referenced |
| `VAR := value` | Simple | At **definition** time | Evaluated once, stored as string |
| `VAR ?= value` | Conditional | At use time | Set only if not already set |
| `VAR += value` | Append | Depends on original type | Append to existing value |

### Expansion Order

Variable references `$(VAR)` are expanded according to a precise order:

1. **Automatic variables** (`$@`, `$<`, `$^`, etc.) — bound during rule execution
2. **Target-specific variables** — set with `target: VAR = value`
3. **Pattern-specific variables** — set with `%.o: VAR = value`
4. **File-level variables** — set in the Makefile
5. **Environment variables** — inherited from shell
6. **Default variables** — Make's built-in defaults

### Automatic Variables

| Variable | Expansion | Description |
|:---------|:----------|:------------|
| `$@` | Target name | The file being built |
| `$<` | First prerequisite | First dependency |
| `$^` | All prerequisites | All dependencies (deduplicated) |
| `$+` | All prerequisites | All dependencies (with duplicates) |
| `$*` | Stem | The `%` match in pattern rules |
| `$(@D)` | Directory of `$@` | Directory part of target |
| `$(@F)` | File of `$@` | File part of target |

---

## 5. Pattern Rules — Term Rewriting

### Implicit Rules as Rewrite System

A pattern rule like:

```makefile
%.o: %.c
	$(CC) -c $< -o $@
```

Is a **rewrite rule**: given a goal `foo.o`, Make unifies the pattern `%.o` with `foo.o`, binds `%` = `foo`, and substitutes to get prerequisite `foo.c`.

$$\text{match}(\%.o, \text{foo.o}) \implies \% = \text{foo} \implies \text{prereq} = \text{foo.c}$$

### Rule Chaining

Make can chain pattern rules:

```makefile
%.o: %.c      # Rule 1
%.c: %.y      # Rule 2
```

To build `parser.o`, Make chains: `parser.y → parser.c → parser.o`.

The chain length is bounded — Make limits chain depth to prevent infinite loops (typically 1 intermediate file).

### Static Pattern Rules

More explicit than implicit rules:

```makefile
objects = main.o utils.o

$(objects): %.o: %.c
	$(CC) -c $< -o $@
```

Only applies to the listed targets, not globally.

---

## 6. Functions — String Manipulation

### Key Built-in Functions

| Function | Syntax | Result |
|:---------|:-------|:-------|
| `$(subst from,to,text)` | Replace | Literal string replacement |
| `$(patsubst %.c,%.o,$(SRC))` | Pattern replace | `main.c utils.c` → `main.o utils.o` |
| `$(filter %.c,$(FILES))` | Filter | Keep matching words |
| `$(filter-out %.h,$(FILES))` | Exclude | Remove matching words |
| `$(sort $(LIST))` | Sort + dedup | Lexicographic, removes duplicates |
| `$(wildcard src/*.c)` | Glob | Expand file glob pattern |
| `$(shell command)` | Shell exec | Run shell command, capture stdout |
| `$(foreach var,list,text)` | Loop | Expand text for each word in list |
| `$(call func,arg1,arg2)` | Call | Invoke user-defined function |
| `$(if cond,then,else)` | Conditional | Non-empty string = true |

### Worked Example: Recursive Wildcard

```makefile
rwildcard = $(foreach d,$(wildcard $(1:=/*)),$(call rwildcard,$d,$2) $(filter $(subst *,%,$2),$d))

ALL_C := $(call rwildcard,src,*.c)
# Finds all .c files recursively under src/
```

---

## 7. Order-Only Prerequisites

### Syntax

```makefile
target: normal-prereqs | order-only-prereqs
```

Order-only prerequisites must **exist** before the target is built, but their **timestamp** is not checked. If they're newer than the target, no rebuild.

### Use Case: Directory Creation

```makefile
$(BUILDDIR)/%.o: %.c | $(BUILDDIR)
	$(CC) -c $< -o $@

$(BUILDDIR):
	mkdir -p $@
```

Without `|`, every object would rebuild whenever the directory's mtime changes (which happens whenever any file in it changes).

---

## 8. Recursive vs Non-Recursive Make

### Recursive Make (Traditional)

```makefile
# Top-level Makefile
subsystems:
	$(MAKE) -C lib
	$(MAKE) -C src
```

Each `$(MAKE)` spawns a **new Make process** with its own dependency graph. Problem: the sub-Makes can't see each other's dependencies, so parallelism is limited and builds may be incorrect.

### The Peter Miller Argument

"Recursive Make Considered Harmful" (1998): because sub-Makes have separate dependency graphs, they can't correctly determine build order across directories. The solution: a single top-level Makefile that includes all rules.

### Non-Recursive Make

```makefile
include lib/module.mk
include src/module.mk
```

Single dependency graph → correct parallel builds → optimal rebuild decisions.

$$\text{Correct build order} \iff \text{single DAG with all edges}$$

---

## 9. Summary of Key Concepts

| Concept | Formalization | Key Detail |
|:--------|:-------------|:-----------|
| Dependency graph | DAG $G = (V, E)$ | Targets + prerequisites |
| Build decision | $\text{mtime}(p) > \text{mtime}(t)$ | Timestamp comparison |
| Build order | Post-order DFS (topological sort) | Depth-first, leaves first |
| Pattern rules | Term rewriting (`%` = unification variable) | Chain depth limited |
| Variables | Recursive (lazy) vs Simple (eager) | `=` vs `:=` |
| Parallel build | Independent vertices in DAG | `-j N` flag |
| Correctness | Single DAG with complete edges | Non-recursive preferred |

---

*Make is graph theory applied to build systems. Every correct Makefile is a DAG, every build is a topological sort, and every bug is either a missing edge or a cycle. Understanding this transforms Make from a mysterious incantation file into a formal specification of your build.*
