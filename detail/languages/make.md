# The Internals of GNU Make — Evaluation, Dependency Graph, and Performance

> *Make is older than most languages still in use, but it is also more subtle than most. The surface looks like a list of recipes; the engine underneath is a two-phase evaluator over a directed acyclic graph, with a string-rewriting macro system, a job-token protocol, an implicit-rule chain search, and a set of expansion semantics that depend on which side of `:=` and `=` you stand on. This document is the deep dive: how GNU Make actually decides what to build, when variables expand, why your recipes run in their own subshells, how the jobserver passes tokens through pipes, and where the historical paper "Recursive Make Considered Harmful" still applies thirty years later.*

---

## 1. Make's Conceptual Model — Dependency Graph

### 1.1 The Build IS the Graph Traversal

Make is not a scripting language with build features bolted on. It is a **dependency-graph evaluator** whose only job is to walk a DAG (directed acyclic graph) of targets and prerequisites and execute recipes for nodes that are missing or out of date. Everything else — variables, functions, conditionals, includes — exists to construct that graph.

A rule:

```makefile
target: prereq1 prereq2 prereq3
	recipe-line-1
	recipe-line-2
```

declares three directed edges in the graph: `prereq1 -> target`, `prereq2 -> target`, `prereq3 -> target`. The recipe is the **action** associated with the target node — it is run only when at least one prerequisite is newer than the target, or the target file does not exist.

### 1.2 Topological Execution

Given a goal target $G$ on the command line (or the default goal — the first non-dot target in the file), Make performs a **post-order depth-first traversal** of the subgraph reachable from $G$:

```
build(t):
    for each prerequisite p of t:
        build(p)                 # recurse before acting on t
    if t does not exist OR
       any prereq mtime > t mtime:
        execute t's recipe
```

This produces a valid topological sort: every prerequisite is fully built before its dependents. Two nodes that are mutually independent (neither is an ancestor of the other) can be built **in parallel** — that is the entire mechanism behind `make -j`.

### 1.3 The "Minimum Recompilation" Claim

Make is often advertised as performing the minimum work necessary. The claim is precise but conditional. Make rebuilds target $t$ if and only if:

1. $t$ does not exist on disk, **or**
2. some prerequisite $p$ has `mtime(p) > mtime(t)`.

Equality (`mtime(p) == mtime(t)`) does **not** trigger a rebuild. Make uses strict greater-than. On filesystems with second-resolution timestamps (HFS+, FAT32, some NFS mounts), two files written within the same second can compare equal, and a "stale" target may be silently skipped.

The minimum-recompilation claim further requires:

| Requirement | Consequence if violated |
|:-----------|:-----------------------|
| **Complete edges** — every header included by a `.c` file is a prereq of the resulting `.o` | Edits to a header don't trigger recompilation |
| **Acyclic graph** | Make errors out: `Circular X <- Y dependency dropped` |
| **Stable timestamps** | Touching a file (without changing it) triggers a rebuild; reproducible-build tools fight this with `SOURCE_DATE_EPOCH` |
| **No side effects in recipes that escape the graph** | Recipe writes a sibling file Make doesn't know about; that sibling never re-derives |

Auto-generated dependency files (see Section 11) exist precisely to satisfy requirement 1 without manually listing every header.

### 1.4 Phony Vertices

Some vertices have no corresponding file on disk: `clean`, `install`, `test`, `all`. They are declared with `.PHONY:` to tell Make that no `stat()` should be performed and the recipe always runs. Internally, a phony target is treated as if its mtime were $-\infty$ — it is always older than any prereq, always triggering its recipe.

### 1.5 The Default Goal

Make builds the **default goal** when invoked without target arguments. The default goal is:

1. Whatever `.DEFAULT_GOAL := name` is set to, **or**
2. The first target encountered in the parse that is not a special target (does not start with `.` and is not a pattern rule).

A common idiom is to put `all:` first in the Makefile so `make` with no arguments builds everything:

```makefile
.DEFAULT_GOAL := all

all: lib bin docs
```

---

## 2. Phases of Evaluation

### 2.1 The Two Phases

Every `make` invocation runs in two sharply distinct phases:

| Phase | What happens |
|:------|:-------------|
| **Phase 1 — Read-in** | Parse all Makefiles, expand simple variables, evaluate conditionals, build the rule database, snapshot the dependency graph |
| **Phase 2 — Build** | Walk the graph from the goal, expand recursive variables and recipes lazily, run sub-shells |

This separation explains nearly every "weird" Make behavior. During Phase 1, recipes are **not** executed and most variables are **not** expanded. During Phase 2, the graph structure is **frozen** — adding a rule via `$(eval ...)` after Phase 1 does nothing.

### 2.2 What Gets Expanded When

The rule "expansion happens at variable USE, not at variable SET" applies to recursive variables (`=`). For simply-expanded variables (`:=`), the right-hand side is expanded at SET time.

```makefile
# Phase 1 — both lines parsed
NOW_LATE  = $(shell date)         # captured but not yet expanded
NOW_EARLY := $(shell date)         # expanded NOW, frozen as a string

# Phase 2 — recipe executes
print:
	@echo late=$(NOW_LATE)         # date runs HERE — fresh
	@echo early=$(NOW_EARLY)       # value frozen at parse time
```

If you run this two seconds after parsing finishes, `NOW_LATE` shows the recipe-execution time, and `NOW_EARLY` shows the parse time. The seconds will differ.

### 2.3 Why Recipe Lines Are Expanded Last

A target's recipe is stored as a list of strings during Phase 1. When Make finally decides to run that target in Phase 2, it expands each recipe line **at that moment**, with `$@`, `$<`, `$^`, target-specific variables, and any other deferred references resolved. This is why automatic variables work — they are bound only when the recipe runs.

### 2.4 The `define` / `endef` Block

Multi-line variable definitions use `define`:

```makefile
define HEADER_TEMPLATE
#ifndef $(1)_H
#define $(1)_H
extern void $(1)_init(void);
#endif
endef
```

Like `=`, `define` is recursively expanded — its body is captured verbatim and expanded at use time. `define VAR :=` (with the `:=` modifier, GNU 3.82+) creates a simply-expanded multi-line variable.

---

## 3. Variable Forms — Recursive vs Simple

### 3.1 The Five Assignment Operators

| Operator | Name | Expansion Time | Behavior |
|:---------|:-----|:---------------|:---------|
| `=` | Recursive | At each USE | Right-hand side stored verbatim, expanded fresh on every reference |
| `:=` | Simply-expanded | At SET | Right-hand side expanded once, result stored as static string |
| `::=` | POSIX simply-expanded | At SET | POSIX 2024 spelling for `:=`; identical semantics in GNU Make |
| `?=` | Conditional | At each USE (if expanded) | Assigns only if variable is currently undefined |
| `+=` | Append | Inherits flavor | Appends to existing variable, preserving recursive-vs-simple flavor |

### 3.2 Worked Example — The Shell-Out Difference

```makefile
SOURCES_LATE  = $(shell find src -name '*.c')
SOURCES_EARLY := $(shell find src -name '*.c')

OBJS_LATE  := $(SOURCES_LATE:.c=.o)
OBJS_EARLY := $(SOURCES_EARLY:.c=.o)
```

Both `OBJS_*` look identical, but `SOURCES_LATE` will re-run `find` **every time** it is referenced — once for the `OBJS_LATE` definition above, again if a recipe references it, again if a function call evaluates it, and so on. Each reference forks a shell. On a 10000-file source tree this is the difference between a 50ms parse and a 5-second parse.

The rule: **always use `:=` for `$(shell ...)` results unless you explicitly want re-evaluation.**

### 3.3 The `?=` Idiom

```makefile
CC ?= gcc          # use gcc unless CC came from environment or command line
PREFIX ?= /usr/local
```

`?=` checks whether the variable already has any origin other than `default` or `undefined`. Environment variables, command-line arguments, and earlier file assignments all suppress `?=`.

### 3.4 The `+=` Flavor Inheritance

```makefile
CFLAGS = -Wall                    # recursive
CFLAGS += -O2                     # still recursive: stored as "-Wall -O2"

WARNINGS := -Wall                 # simple
WARNINGS += $(EXTRA)              # simple — $(EXTRA) is expanded NOW
```

`+=` does **not** flatten. Its rule: if the variable was recursive, the appended text is also recursive (deferred); if simple, the appended text is expanded immediately. If the variable was previously undefined, `+=` creates a recursive variable.

### 3.5 Practical Implications

The recursive-vs-simple distinction matters most in three places:

1. **Shell results.** `$(shell ...)` should almost always be `:=`. Otherwise the shell runs every time the variable is read.
2. **`$(wildcard ...)` calls.** Same reasoning — `:=` to snapshot the directory listing.
3. **Recipe-context variables.** Variables intended to use `$@` or `$<` MUST be recursive (`=`), because automatic variables don't exist at parse time:
    ```makefile
    OUT = $(@D)/$(@F).log     # recursive: $(@D) and $(@F) resolved per-target
    ```

The opposite mistake — `:=` for a variable meant to use `$@` — gives an empty `$@` because Phase 1 has no current target.

### 3.6 The `override` Directive

When the user invokes `make CFLAGS=-O0`, command-line variables clobber file assignments. To force the file's value to win:

```makefile
override CFLAGS += -fPIC
```

`override` is required to modify a command-line variable from inside the Makefile.

---

## 4. Variable Scope and Origin

### 4.1 The Six Origins

Every variable has an **origin** — where its current value came from. `$(origin VAR)` returns one of:

| Origin | Meaning |
|:-------|:--------|
| `undefined` | Variable is not set |
| `default` | Value comes from Make's built-in default rules (e.g. `CC = cc`) |
| `environment` | Value inherited from the calling shell environment |
| `environment override` | Environment value, but `make -e` is in effect (env beats file) |
| `file` | Set by an assignment in a Makefile |
| `command line` | Set on the `make` command line: `make CC=clang` |
| `override` | Set with the `override` directive |
| `automatic` | Bound during recipe execution: `$@`, `$<`, etc. |

Use case — only set `CC` if the user hasn't already chosen one:

```makefile
ifeq ($(origin CC),default)
    CC := clang
endif
```

This is more precise than `?=` because `?=` treats `default`-origin variables as already-set (it WILL override `CC = cc`), whereas the explicit `origin` check lets you distinguish.

### 4.2 The Three Flavors

`$(flavor VAR)` returns:

| Flavor | Meaning |
|:-------|:--------|
| `undefined` | Not set |
| `recursive` | Set with `=` or `define` |
| `simple` | Set with `:=` or `::=` |

### 4.3 Scope Hierarchy

Variables in Make are global by default — there is no lexical scope. But Make supports **target-specific** and **pattern-specific** overrides:

```makefile
# Target-specific
debug: CFLAGS := -O0 -g
debug: app

# Pattern-specific
%.fast.o: CFLAGS := -O3
%.fast.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@
```

When Make builds `debug`, it sees `CFLAGS = -O0 -g` for that target's recipe and **all of its prerequisites**. The override propagates down the dependency graph but not up.

### 4.4 The `export` Keyword

Recipes are executed by sub-shells. Make passes a controlled subset of variables to those shells. By default:

- Variables defined on the command line are exported.
- The `MAKE`, `MAKELEVEL`, `MAKEFLAGS`, `MFLAGS`, `MAKECMDGOALS` are exported.
- Other variables are **not** exported.

To force export:

```makefile
export PATH := /opt/toolchain/bin:$(PATH)
export CC

unexport SECRET
```

`export` with no value tells Make "export this name". `unexport` removes a name from the export set. The `.EXPORT_ALL_VARIABLES:` special target exports everything (rarely a good idea — pollutes the recipe environment).

### 4.5 Inheritance into `$(MAKE)` Sub-invocations

Recursive Make passes variables via the environment plus `MAKEFLAGS`. Command-line and exported variables are visible to sub-makes; file-only variables are not. This is why `make CC=clang` in the top-level passes `CC` down, but `CC = clang` in the Makefile does not (unless you `export` it).

---

## 5. Pattern Rules and the Implicit Rule Chain

### 5.1 The `%` Wildcard

A pattern rule uses `%` as a single-stem placeholder:

```makefile
%.o: %.c
	$(CC) -c $< -o $@
```

Given a goal `parser.o`, Make matches `%.o` against `parser.o`, binds `% = parser`, and substitutes into the prerequisite list to derive `parser.c`. The match is **non-greedy and non-overlapping** — exactly one `%` per side.

### 5.2 Implicit Rule Chain Search

Make ships with a database of built-in rules: `%.o: %.c`, `%.o: %.s`, `%.tex: %.dvi`, etc. When Make needs to build `foo.o` and you didn't write a rule, it tries each implicit rule in turn. If none produce a buildable prereq, Make can **chain** rules — derive `foo.c` from `foo.y` via `%.c: %.y`, then derive `foo.o` from `foo.c`.

Chain length is bounded (default: 1 intermediate file via implicit rules) to prevent runaway derivation.

The full implicit-rule database is visible via:

```bash
make -p -f /dev/null | less       # dump database without running anything
```

This prints every built-in rule, every variable's default value, and every suffix rule. Essential reading once in your career.

### 5.3 Static Pattern Rules

`%` in a regular pattern rule applies to **all** matching targets globally. To restrict the pattern to a specific list of targets, use a **static pattern rule**:

```makefile
OBJS := main.o utils.o parser.o

$(OBJS): %.o: %.c
	$(CC) -c $< -o $@
```

Format: `targets : target-pattern : prereq-patterns`. Only `main.o`, `utils.o`, `parser.o` get the rule — no other `.o` matches it. Static patterns are stricter and avoid surprising matches in mixed projects.

### 5.4 The Stem `$*`

The stem is the text matched by `%`. If `%.o: %.c` matches `src/main.o`:

```
$*  = src/main           # stem
$@  = src/main.o         # full target
$<  = src/main.c         # first prereq
```

The stem retains its directory component. Use `$(notdir $*)` for filename-only.

### 5.5 Double-Colon Rules

A regular rule can have only **one** recipe per target. Double-colon rules allow multiple **independent** recipes:

```makefile
build/log:: 
	@echo "first recipe"

build/log::
	@echo "second recipe"
```

Each `::` rule is independent — Make runs them all (in declaration order) when the target is out of date, but each one's prereqs are checked separately. Used rarely; the most common case is generators that append to a log on each invocation.

### 5.6 Grouped Targets `&:` (GNU 4.3+)

A normal rule with multiple targets:

```makefile
foo.h foo.c: foo.y
	bison foo.y -o foo.c        # WRONG: rule runs once per target
```

Make 4.3 added grouped targets — a rule whose recipe is run **once** to produce all listed targets:

```makefile
foo.h foo.c &: foo.y
	bison foo.y -o foo.c        # runs once; produces both
```

Before 4.3, the workaround was a stamp file or a lock-file pattern. Mark this as 4.3+ in any Makefile that uses it.

---

## 6. Automatic Variables — Full Reference and Semantic Subtleties

### 6.1 The Core Set

| Variable | Meaning | Available |
|:---------|:--------|:----------|
| `$@` | Target name | Always in recipe |
| `$<` | First prerequisite | Always in recipe |
| `$^` | All prerequisites, deduplicated, space-separated | Always in recipe |
| `$+` | All prerequisites, **with duplicates**, in order | Always in recipe |
| `$?` | Prerequisites newer than target | Always in recipe |
| `$*` | Stem (the `%` match) | Pattern rules only; static-pattern rules; suffix rules |
| `$\|` | Order-only prerequisites | Always in recipe |
| `$%` | Archive member name (when target is `archive(member)`) | Archive rules only |

### 6.2 The Directory and Filename Variants

Each automatic variable has `D` (directory) and `F` (file) variants:

| Form | Equivalent | Example |
|:-----|:-----------|:--------|
| `$(@D)` | `$(dir $@)` | `src/main.o` -> `src` |
| `$(@F)` | `$(notdir $@)` | `src/main.o` -> `main.o` |
| `$(<D)` | `$(dir $<)` | First prereq's directory |
| `$(<F)` | `$(notdir $<)` | First prereq's filename |
| `$(^D)` | Directory parts of `$^` (space-separated) | All prereq directories |
| `$(^F)` | Filename parts of `$^` | All prereq filenames |
| `$(?D)`, `$(?F)` | Same for `$?` | |
| `$(*D)`, `$(*F)` | Stem split | |

`$(@D)` is the canonical idiom for ensuring an output directory exists before writing into it (often combined with order-only prereqs).

### 6.3 `$^` vs `$+` — The Duplicate Subtlety

```makefile
prog: a.o b.o a.o
	$(CC) -o $@ $^         # a.o b.o (duplicates removed)
	$(CC) -o $@ $+         # a.o b.o a.o (duplicates preserved)
```

Linkers sometimes need duplicate prerequisites for circular library references: `gcc -o app a.o libfoo.a libbar.a libfoo.a`. Use `$+` in that case.

### 6.4 `$?` vs `$^` — Incremental Linking

```makefile
libfoo.a: $(OBJS)
	ar rcs $@ $?           # only the changed objects added/replaced
```

`$?` lists only prereqs newer than the target — useful for incremental archives or for tools that accept only the changed inputs. Most compilers want the full prereq list; archives want the changed list.

### 6.5 Position Subtleties

Automatic variables are bound when the recipe runs, but **target-specific variables that reference automatics** must be recursive:

```makefile
%.o: OUT_DIR = $(@D)         # WRONG with :=, OK with =
%.o: %.c
	$(CC) -c $< -o $(OUT_DIR)/$(@F)
```

With `:=`, `$(@D)` would be evaluated at parse time when no target exists, yielding empty.

Automatic variables are **not** available in the prereq list itself — only in the recipe. This is why the `.SECONDEXPANSION:` directive (Section 12) exists.

---

## 7. The Function Library — Internals

### 7.1 Text Functions

| Function | Effect |
|:---------|:-------|
| `$(subst FROM,TO,TEXT)` | Literal substring replacement |
| `$(patsubst PAT,REPL,TEXT)` | Pattern replacement (`%` is a wildcard) |
| `$(strip TEXT)` | Collapse runs of whitespace, trim leading/trailing |
| `$(findstring FIND,IN)` | Returns FIND if it appears in IN, else empty |
| `$(filter PAT...,LIST)` | Keep words in LIST that match any PAT |
| `$(filter-out PAT...,LIST)` | Remove words matching PAT |
| `$(sort LIST)` | Sort lexicographically AND remove duplicates |
| `$(word N,LIST)` | The Nth word (1-indexed) |
| `$(words LIST)` | Number of words |
| `$(wordlist S,E,LIST)` | Words from index S to E inclusive |
| `$(firstword LIST)` | First word |
| `$(lastword LIST)` | Last word |

`patsubst` is the workhorse — `%` matches a non-empty stem, captured for the replacement:

```makefile
SRCS := main.c utils.c parser.c
OBJS := $(patsubst %.c,build/%.o,$(SRCS))
# build/main.o build/utils.o build/parser.o
```

The substitution-reference shorthand `$(VAR:%.c=%.o)` is equivalent to `$(patsubst %.c,%.o,$(VAR))`.

### 7.2 File Functions

| Function | Effect |
|:---------|:-------|
| `$(dir NAMES...)` | Directory part (everything through final `/`) |
| `$(notdir NAMES...)` | Filename part (everything after final `/`) |
| `$(suffix NAMES...)` | Extension including dot |
| `$(basename NAMES...)` | Path without extension |
| `$(addsuffix SFX,NAMES)` | Append SFX to each word |
| `$(addprefix PFX,NAMES)` | Prepend PFX to each word |
| `$(join LIST1,LIST2)` | Pairwise concatenation |
| `$(wildcard PAT)` | Glob expansion (returns matching files) |
| `$(realpath NAMES)` | Canonical absolute path (resolves symlinks) |
| `$(abspath NAMES)` | Absolute path (no symlink resolution) |

### 7.3 Conditional Functions

```makefile
$(if CONDITION,THEN[,ELSE])     # CONDITION non-empty -> THEN
$(or A,B,C)                     # First non-empty
$(and A,B,C)                    # Last value if all non-empty, else empty
```

These are functions, not directives — they expand inline. Use `ifeq`/`ifneq` for parse-time conditionals.

### 7.4 `foreach`, `call`, `value`, `eval`

```makefile
$(foreach VAR,LIST,TEXT)        # Expand TEXT once per word in LIST
```

The classic loop:

```makefile
DIRS := src include test
HEADERS := $(foreach d,$(DIRS),$(wildcard $d/*.h))
```

`call` invokes a user-defined function:

```makefile
define greet
hello, $(1)! today is $(2).
endef

$(call greet,world,monday)      # hello, world! today is monday.
```

`value` retrieves a variable's **unexpanded** value (useful for inspecting `define` blocks):

```makefile
$(info $(value greet))          # prints the literal define body
```

`eval` is the meta-programming escape hatch — expand a string and re-parse it as Makefile syntax:

```makefile
$(eval $(call greet,foo,bar))   # parses the result as makefile lines
```

This is how you generate pattern rules dynamically. See Section 7.7.

### 7.5 The `shell` Function — Cost Model

```makefile
TODAY := $(shell date +%Y%m%d)
GIT_REV := $(shell git rev-parse --short HEAD)
```

Every `$(shell ...)` expansion is a `fork()` + `execve()` of `$(SHELL) -c ...` — typically `/bin/sh -c date`. On Linux this is roughly 1ms per call. On macOS with security policies it can be 5-10ms. With recursive variables (`=`), each reference re-forks. Thousand-call Makefiles with `SOURCES = $(shell find ...)` can spend seconds in shell setup before the first compile.

The `:=` snapshot pattern avoids this:

```makefile
SOURCES := $(shell find src -name '*.c')
```

### 7.6 The `file` Function (GNU 4.2+)

```makefile
$(file >output.txt,$(LARGE_VAR))     # write
$(file >>output.txt,more text)        # append
text := $(file <input.txt)            # read
```

Avoids the shell-out cost for writing large variables to disk. Marked 4.2+ — older systems must use `echo > file` in a recipe.

### 7.7 Code Generation with `eval`

The canonical pattern: build pattern rules in a loop:

```makefile
TARGETS := lib1 lib2 lib3

define LIB_RULE
$(1).a: $(1)/*.o
	ar rcs $$@ $$^
endef

$(foreach t,$(TARGETS),$(eval $(call LIB_RULE,$(t))))
```

The `$$@` inside the template becomes `$@` after the first expansion (by `call`), and is resolved as the automatic variable in the second expansion (parsing). This double-dollar pattern is the most common source of "why doesn't my eval work" bugs — count the expansion levels.

---

## 8. The Make Process Tree

### 8.1 Every Recipe Line Is Its Own Subshell

By default, Make runs each recipe line in a **fresh** subshell:

```makefile
target:
	cd build              # subshell 1: cd happens
	pwd                   # subshell 2: starts in original dir, prints original
```

Variables set in line 1 don't survive to line 2. This bites everyone once.

The fix is to chain commands with `&&` or `;` so they run in one shell:

```makefile
target:
	cd build && pwd
```

Or make the recipe a single logical line via backslash continuation:

```makefile
target:
	cd build && \
	pwd && \
	make all
```

### 8.2 `.ONESHELL` (GNU 3.82+)

The special target `.ONESHELL:` puts **every recipe line** of every target into a single subshell:

```makefile
.ONESHELL:

target:
	cd build
	pwd               # works — same shell
	make all
```

This changes a fundamental Make behavior — be aware that it interacts with the per-line prefixes (`@`, `-`, `+`) which now apply only to the first line of the recipe.

### 8.3 Recipe Line Prefixes

| Prefix | Meaning |
|:-------|:--------|
| `@` | Suppress echoing the command (Make would otherwise print it) |
| `-` | Ignore non-zero exit status (continue even on failure) |
| `+` | Honor `make -t/-q/-n/-s` flags — execute even in dry-run mode |

Combining: `@-rm -f target` runs `rm`, suppresses the echo, and ignores failure (for `clean` targets).

### 8.4 The `set -e` Trap

A multi-command line with `;` does NOT abort on first failure unless `set -e` is in effect:

```makefile
target:
	cd build; broken-command; echo "still ran!"
```

`broken-command` fails, but `echo` still runs. To abort:

```makefile
target:
	set -e; cd build; broken-command; echo "won't print"
```

`&&` chaining is generally safer:

```makefile
target:
	cd build && broken-command && echo "won't print"
```

### 8.5 The `SHELL` Variable

Make uses `/bin/sh` by default. Override with:

```makefile
SHELL := /bin/bash
.SHELLFLAGS := -ec               # bash strict mode
```

This changes which shell interprets every recipe line. Bash's process-substitution `<(...)`, arrays, and `[[ ... ]]` become available; portability to dash-only systems suffers.

### 8.6 Recipe-Line Cost Model

Each recipe line costs:

1. `fork()` — copy-on-write address space (~30µs)
2. `execve()` — replace with shell binary (~1ms)
3. Shell startup — read profile, parse command (~5ms for bash)
4. Command execution — the actual work
5. `wait()` — collect exit status

For 10000 trivial recipes (e.g. `mkdir -p $(@D)`), startup overhead dominates. Combining lines with `&&` or using `.ONESHELL:` cuts that overhead.

---

## 9. Parallel Execution and Job Server

### 9.1 The `-j` Flag

```bash
make -j8                    # up to 8 concurrent recipes
make -j                     # unlimited (number of recipes)
make -l4.0                  # don't start a new job if loadavg > 4.0
```

Make schedules independent recipes onto a pool of worker slots. Two recipes are **concurrent-safe** if neither's prereqs include the other's target.

### 9.2 The Jobserver Protocol

When `make` recursively invokes another `$(MAKE)`, both processes need to coordinate so the total parallelism stays at `-j N`. GNU Make implements this via a **token pipe** — the jobserver:

1. Top-level Make creates a pipe (or, in 4.4+, a named FIFO) and writes `N - 1` bytes (tokens) into it.
2. Each Make invocation that wants to start a parallel recipe **reads one byte** from the pipe (acquires a token).
3. When the recipe finishes, Make **writes one byte** back (releases the token).
4. Sub-makes inherit the pipe via the `MAKEFLAGS` environment variable: `--jobserver-fds=3,4` (or `--jobserver-auth=fifo:/path` in 4.4+).

```c
// Conceptual jobserver acquire (from GNU Make sources, simplified)
char token;
if (read(jobserver_read_fd, &token, 1) == 1) {
    // got token — start recipe
} else {
    // pipe empty — wait
}
```

The "implicit token" — every Make invocation has one slot it always owns, allowing single-job builds with no synchronization. This is why `make -j1` and `make` with no `-j` behave the same.

### 9.3 The `+make` Recipe Convention

When a recipe explicitly invokes another make and you want it to participate in the jobserver, prefix with `+`:

```makefile
sub:
	+$(MAKE) -C subdir
```

The `+` tells Make to run the line even in dry-run mode, AND signals that the line is a make-aware command that should inherit the jobserver pipe. Without it, sub-makes get `-j1` and your parallel build serializes.

The variable `$(MAKE)` is automatically `+`-prefixed when used as the first word of a recipe, so:

```makefile
sub:
	$(MAKE) -C subdir          # implicitly +make
```

works correctly.

### 9.4 Deadlock Risks

If a recipe runs a script that itself wants to run things in parallel **without** participating in the jobserver, two failure modes appear:

1. **Over-subscription** — N tokens at the top, plus M parallel invocations from each recipe = N*M total parallelism, exceeding `-j`.
2. **Deadlock** — a recipe waits on a child that needs a token; no tokens left because all are held by sibling recipes also waiting.

Make 4.4 introduced a **named-pipe** jobserver that scripts can participate in by reading the `MAKEFLAGS` env var, parsing `--jobserver-auth=fifo:/path`, opening the FIFO, and acquiring tokens. The `make -j --jobserver-style=fifo` flag selects the new style; older sub-processes can still use the FD-based protocol.

### 9.5 Load-Aware Scheduling

```bash
make -l4.0 -j16
```

Allows up to 16 jobs but stops launching new ones if the system loadavg exceeds 4.0. Useful on shared build hosts; the loadavg threshold is checked only at job-launch time, so a build can briefly exceed it.

---

## 10. Recursive Make and the "Considered Harmful" Critique

### 10.1 The Traditional Pattern

```makefile
SUBDIRS := lib src test

all:
	for d in $(SUBDIRS); do $(MAKE) -C $$d; done
```

Each subdirectory has its own Makefile, and the top-level Makefile loops over them. Familiar from autoconf-era projects.

### 10.2 The Peter Miller Paper

Peter Miller's 1998 paper *"Recursive Make Considered Harmful"* identified the central flaw: each recursive Make invocation has its **own** dependency graph. A change in `lib/foo.h` that should trigger a recompile in `src/main.c` is invisible — the `src` Make has never seen `lib/foo.h`.

Workarounds (forced rebuilds, `make clean` everywhere, ad-hoc dependency files between directories) all fail to provide what a single Makefile gives for free: **one DAG with all edges**.

### 10.3 The Modern Non-Recursive Pattern

A single top-level Makefile includes per-directory rule fragments:

```makefile
# top-level Makefile
include lib/module.mk
include src/module.mk
include test/module.mk
```

Each `module.mk` contributes targets and rules:

```makefile
# lib/module.mk
LIB_SRCS := $(wildcard lib/*.c)
LIB_OBJS := $(LIB_SRCS:.c=.o)

lib/lib.a: $(LIB_OBJS)
	ar rcs $@ $^
```

```makefile
# src/module.mk
SRC_SRCS := $(wildcard src/*.c)
SRC_OBJS := $(SRC_SRCS:.c=.o)

src/app: $(SRC_OBJS) lib/lib.a
	$(CC) -o $@ $^
```

The top-level Make sees every rule, every prerequisite, and every header — a single DAG. Parallelism is correct (`-j N` actually saturates). Header changes propagate. Builds become reproducible.

### 10.4 `make -C dir target`

Sometimes you genuinely want to run Make in another directory:

```bash
make -C /src/project all
```

`-C` does `chdir` before reading the Makefile. The Makefile then references files relative to that directory. Combined with `MAKEFLAGS` propagation (parallelism, debug flags, etc.), recursive Make with `-C` is at least technically correct — the failure modes in the Miller paper apply only when the cross-directory dependency edges aren't expressible in any single graph.

### 10.5 `MAKELEVEL`

Make sets the environment variable `MAKELEVEL` to the recursion depth: 0 for the top-level invocation, 1 for the first sub-make, 2 for sub-sub, etc. Use it for top-level-only logic:

```makefile
ifeq ($(MAKELEVEL),0)
    @echo "starting build"
endif
```

`MAKEFLAGS` is similarly automatic — it carries `-j`, `-d`, and other flags through recursive invocations.

---

## 11. Auto-Generated Dependencies

### 11.1 The Problem

A correct Makefile needs to know that `main.o` depends on every header transitively included by `main.c`:

```c
// main.c
#include "config.h"
#include "lib/foo.h"      // includes "lib/foo_internal.h"
```

Manually listing those is brittle. Compilers can emit them.

### 11.2 GCC/Clang Flags

```bash
gcc -MMD -MP -MF main.d -MT main.o -c main.c -o main.o
```

| Flag | Effect |
|:-----|:-------|
| `-MMD` | Generate dependencies for user headers (skip system headers) |
| `-MD` | Generate dependencies for ALL headers including system |
| `-MP` | Add a phony target for each header (avoids "missing prereq" if a header is deleted) |
| `-MF FILE` | Output dependency info to FILE |
| `-MT TARGET` | Override the target name in the output |

The `.d` file Make includes looks like:

```makefile
main.o: main.c config.h lib/foo.h lib/foo_internal.h

config.h:
lib/foo.h:
lib/foo_internal.h:
```

The empty rules (from `-MP`) make each header a phony — if `lib/foo.h` is renamed and the `.d` file is stale, Make won't fail looking for the missing header.

### 11.3 The Canonical Pattern

```makefile
SRCS := $(wildcard src/*.c)
OBJS := $(SRCS:.c=.o)
DEPS := $(SRCS:.c=.d)

%.o: %.c
	$(CC) $(CFLAGS) -MMD -MP -MF $*.d -MT $@ -c $< -o $@

-include $(DEPS)
```

Three pieces:

1. **The compile rule** generates the `.d` alongside the `.o`.
2. **`-include $(DEPS)`** — note the leading dash. This tells Make to silently include each `.d` file if it exists, and **silently skip** the include if it doesn't (which happens on the first compile, before any `.d` exists). Without the dash, Make would error on the missing files.
3. **The dep files don't need to be tracked as prereqs** — they're regenerated whenever the `.o` is built, which is the only time their content changes.

### 11.4 The `mkdir -p $(@D)` Idiom

When putting `.d` and `.o` files into a build directory:

```makefile
build/%.o: src/%.c
	@mkdir -p $(@D)
	$(CC) $(CFLAGS) -MMD -MP -MF build/$*.d -MT $@ -c $< -o $@
```

The `@mkdir -p $(@D)` ensures `build/sub/dir/` exists before the compiler tries to write into it. Order-only prereqs are a cleaner alternative:

```makefile
build/%.o: src/%.c | build
	$(CC) ...

build:
	mkdir -p $@ $(addprefix build/,$(SUBDIRS))
```

### 11.5 Second-Expansion Gotcha

If you try to use `$(@D)` in the prereq list (rather than the recipe), it doesn't work without `.SECONDEXPANSION:`:

```makefile
# WRONG — $(@D) is empty in prereq position
build/%.o: src/%.c | $(@D)/.dirstamp
	...

# RIGHT — see Section 12
.SECONDEXPANSION:
build/%.o: src/%.c | $$(@D)/.dirstamp
	...
```

---

## 12. Second Expansion

### 12.1 The Problem

In the prereq list, automatic variables aren't bound yet — they're set at recipe-execution time, but the prereq list is evaluated at parse time. So `$@` in a prereq is empty.

### 12.2 The `.SECONDEXPANSION:` Directive

```makefile
.SECONDEXPANSION:

build/%.o: src/%.c | $$(@D)
	$(CC) -c $< -o $@
```

The `.SECONDEXPANSION:` directive tells Make to **expand prereqs twice**:

1. First expansion: at parse time. Most `$(...)` references resolve normally; `$$` becomes `$` (preserving the second-expansion reference).
2. Second expansion: just before the rule fires, with automatic variables bound. `$(@D)` now evaluates correctly.

The `$$` syntax is required: `$$(@D)` literally means "give me a `$(@D)` to expand on the second pass."

### 12.3 Canonical Use Cases

**Per-target output directory:**

```makefile
.SECONDEXPANSION:
$(OBJDIR)/%.o: %.c | $$(@D)
	$(CC) -c $< -o $@

%/.dirstamp:
	@mkdir -p $(@D) && touch $@
```

**Stem-derived prereq:**

```makefile
.SECONDEXPANSION:
%.test: $$*.c $$*-data.txt
	./run-test.sh $* $^
```

Here `$$*` becomes `$*` after first expansion, then resolves to the stem on the second pass.

### 12.4 Multi-Stage Variable Derivation

For complex generators where the prereq list depends on the target's own variables:

```makefile
.SECONDEXPANSION:
$(OUTPUT_FILES): %: $$($(notdir %)_DEPS)
	build-from-deps.sh $< $@
```

Use sparingly — second expansion is expensive and confusing. Prefer `eval` for compile-time generation if possible.

---

## 13. .PHONY and Special Targets — Full Reference

### 13.1 The Master List

| Target | Effect |
|:-------|:-------|
| `.PHONY: name1 name2` | Targets that don't correspond to files; recipe always runs |
| `.SUFFIXES:` | List of recognized suffixes for old-style suffix rules; empty list disables built-in suffix rules |
| `.DEFAULT_GOAL := name` | Set the default goal explicitly |
| `.DEFAULT:` | Recipe used for any target without an explicit rule |
| `.INTERMEDIATE: name` | Mark name as intermediate — Make may delete it after the build |
| `.SECONDARY: name` | Like INTERMEDIATE but Make won't delete |
| `.PRECIOUS: name` | Don't delete name even on recipe failure or interrupt |
| `.DELETE_ON_ERROR:` | Delete target file if its recipe fails (avoids corrupted partial outputs) |
| `.NOTPARALLEL:` | Force serial execution of THIS Make invocation |
| `.ONESHELL:` | All recipe lines for every target share one shell |
| `.POSIX:` | Enable POSIX-conforming behavior (changes shell defaults, etc.) |
| `.SECONDEXPANSION:` | Enable second expansion of prereqs |
| `.EXPORT_ALL_VARIABLES:` | Pass all variables to recipe sub-shells |
| `.IGNORE: target` | Ignore errors in named target's recipe (or ALL if no targets given) |
| `.SILENT: target` | Suppress recipe echo for named target (or ALL if no targets given) |

### 13.2 The Critical Three

In practice, three are essential:

```makefile
.PHONY: all clean install test

.DELETE_ON_ERROR:
.SUFFIXES:
```

- `.PHONY` for non-file targets.
- `.DELETE_ON_ERROR` to avoid leaving truncated outputs (e.g. half-linked binaries) on disk where they'd appear up-to-date next time.
- `.SUFFIXES:` (empty) to disable the legacy suffix-rule database (significant parse-time speedup; otherwise Make tries pathways like `.c.o` for every `.o`).

### 13.3 .PHONY Pitfall

A target named `clean` that's also a real directory `./clean/` will surprise you. Without `.PHONY: clean`, Make sees the directory exists, considers it "up to date", and never runs the recipe. Always declare `.PHONY` explicitly.

### 13.4 .DELETE_ON_ERROR Pitfall

By default, if a recipe fails midway (e.g. the linker writes 10MB of an 80MB binary then segfaults), Make leaves the partial file. The next `make` sees a file newer than its prereqs and skips the rule, building on broken state. `.DELETE_ON_ERROR:` ensures the half-written file is deleted on failure.

This is one of the rare "no-argument" special targets — it applies globally to the Makefile.

### 13.5 .NOTPARALLEL Use Cases

Some build steps genuinely cannot run in parallel — global side effects, registry mutations, license-server checkouts. Mark just the affected target:

```makefile
license-checkout:
	flexlm-checkout
	build-thing
	flexlm-checkin

.NOTPARALLEL: license-checkout
```

Note: in current GNU Make, `.NOTPARALLEL:` is a global directive — all of THIS Makefile invocation runs serially. To serialize specific targets, use prerequisite ordering or explicit lock files.

---

## 14. Conditional Inclusion

### 14.1 The Four Conditional Forms

```makefile
ifdef VAR              # VAR has any non-empty value
ifndef VAR             # VAR is empty or undefined
ifeq (a,b)             # a and b are equal as strings
ifneq (a,b)            # a and b differ
```

Conditionals are **parse-time** — they affect what rules and variables are defined, not runtime behavior:

```makefile
ifeq ($(DEBUG),1)
    CFLAGS += -O0 -g
else
    CFLAGS += -O2 -DNDEBUG
endif
```

### 14.2 Nested Conditionals

```makefile
UNAME := $(shell uname)

ifeq ($(UNAME),Linux)
    CFLAGS += -DLINUX
    ifeq ($(shell uname -m),x86_64)
        CFLAGS += -m64
    endif
else ifeq ($(UNAME),Darwin)
    CFLAGS += -DMACOS
    ifeq ($(shell uname -m),arm64)
        CFLAGS += -mcpu=apple-m1
    endif
endif
```

`else if` chains use the form `else ifeq (...)` (no nesting required — flat from the parser's perspective).

### 14.3 Quoting Subtleties

```makefile
ifeq ($(VAR), value)     # leading space included in comparison!
ifeq ($(VAR),value)      # what you usually want
ifeq "$(VAR)" "value"    # quoted form, immune to whitespace
```

`ifeq` strips a single leading space after the comma but not surrounding ones. The quoted form is safer.

### 14.4 The `shell` Pattern for Platform Detection

```makefile
UNAME := $(shell uname)
ARCH  := $(shell uname -m)
KERNEL_VERSION := $(shell uname -r)

ifeq ($(UNAME),Linux)
    LDFLAGS += -ldl -lrt
endif

ifeq ($(UNAME),Darwin)
    LDFLAGS += -framework CoreFoundation
endif
```

Always `:=` for `uname`-style detection — running it once at parse is enough.

---

## 15. The make -d / -p / --debug Output

### 15.1 The Debugging Flag Family

| Flag | Effect |
|:-----|:-------|
| `make -n` | Dry run — print recipes without executing |
| `make -d` | Debug mode — verbose decision tracing |
| `make --debug=FLAGS` | Selective debug: `b` (basic), `v` (verbose), `i` (implicit), `j` (jobs), `m` (makefile), `n` (none), `a` (all) |
| `make --trace` | Print each recipe before execution (4.0+) |
| `make -p` | Print the database (variables, rules, suffix list, files) |
| `make -q` | Question mode — exit 0 if up-to-date, 1 otherwise; runs nothing |
| `make -W FILE` | Pretend FILE has been modified (force rebuild of dependents) |
| `make -B` | Always build — ignore mtimes |
| `make -k` | Continue on errors (don't abort on first failure) |
| `make -s` | Silent — suppress recipe echoing |

### 15.2 Reading `-d` Output

```bash
make -d 2>&1 | head -100
```

Output excerpt:

```
Updating goal targets....
 Considering target file 'all'.
  File 'all' does not exist.
   Considering target file 'app'.
    File 'app' does not exist.
     Considering target file 'main.o'.
      File 'main.o' does not exist.
       Looking for an implicit rule for 'main.o'.
       Trying pattern rule with stem 'main'.
       Trying implicit prerequisite 'main.c'.
       Found an implicit rule for 'main.o'.
```

The "Trying implicit rule" / "Trying prerequisite" lines reveal the chain search. When Make can't build a target, this output shows every rule it considered and rejected.

### 15.3 `make -p` for Database Inspection

```bash
make -p -f /dev/null 2>/dev/null | grep -v '^#' | head -50
```

Dumps every variable, every implicit rule, every suffix rule. Useful for answering "what's the default value of `CFLAGS`?" (answer: empty in modern GNU Make; the default `CC` is `cc` and the default rule is `%.o: %.c` with `$(CC) $(CFLAGS) -c -o $@ $<`).

### 15.4 `--trace` (4.0+)

```bash
make --trace
```

Outputs each recipe with file and line number BEFORE it runs:

```
Makefile:23: target 'main.o' does not exist
gcc -c main.c -o main.o
Makefile:23: target 'utils.o' does not exist
gcc -c utils.c -o utils.o
```

Cleaner than `-d` for diagnosing "why is this rebuilding?"

### 15.5 `make -p | grep -A1 '^# Files'`

Inside `-p` output, the "Files" section lists every target Make knows about, its mtime, what depends on it, and what it depends on. Real-world example:

```
# Files
# Not a target:
.DEFAULT:
#  Builtin rule

# Not a target:
.SUFFIXES:
#  Builtin rule

main.o:
#  Phony target (prereq of .PHONY).
#  Implicit rule search has been done.
#  Implicit/static pattern stem: 'main'
#  Last modified 1714088421.0
#  File has been updated.
#  Successfully updated.
#  recipe to execute (from 'Makefile', line 23):
	gcc -c main.c -o main.o
```

### 15.6 The Implicit-Rule Attempt Trace

`--debug=i` shows only implicit-rule searches:

```bash
make --debug=i
```

Useful when "make says no rule to make target X" and you need to see which patterns Make tried.

---

## 16. POSIX Make vs GNU Make

### 16.1 The POSIX Subset

POSIX (IEEE Std 1003.1) defines a **minimal** make. Implementations: BSD make (used by FreeBSD/macOS), Solaris make, AIX make. The POSIX subset:

- `=` recursive variables
- `+=` append (POSIX 2024 only — earlier POSIX didn't have it)
- Suffix rules: `.c.o:` (instead of `%.o: %.c`)
- `.SUFFIXES:` to declare suffix order
- `$@`, `$<`, `$^` (POSIX 2024), `$?`
- Conditional via separate Makefile fragments + `include`

### 16.2 GNU Extensions

| Feature | GNU Status | Portability |
|:--------|:-----------|:------------|
| `:=` simply-expanded variables | GNU extension | Now in POSIX 2024 as `::=` |
| `?=` conditional assignment | GNU | POSIX 2024 |
| Pattern rules with `%` | GNU | NOT POSIX |
| Function library (`patsubst`, `shell`, etc.) | GNU | NOT POSIX |
| Conditional directives (`ifeq`, `ifdef`) | GNU | NOT POSIX |
| `$(eval ...)` | GNU | NOT POSIX |
| `&:` grouped targets (4.3+) | GNU | NOT POSIX |
| `.ONESHELL:` | GNU | NOT POSIX |
| Parallel jobserver | GNU | NOT POSIX |
| `define` / `endef` | GNU | NOT POSIX |

### 16.3 BSD Make (`bmake`) Differences

BSD make has its own extensions, mostly incompatible with GNU:

- `:=` means **modifier** (variable expansion modifier), NOT simply-expanded assignment.
- `.if` / `.elif` / `.endif` for conditionals (period prefix).
- `${VAR:M*.c}` modifier syntax for filtering.
- `.for` loops.
- No pattern rules; uses suffix-style and `:T:H:R:E` modifiers.

A Makefile portable across both is severely restricted. Most projects ship a `GNUmakefile` (which GNU Make picks up first) or a `BSDmakefile` (which BSD Make picks up first), with the platform-incompatible logic in the right one.

### 16.4 macOS Gotcha

macOS ships **GNU Make 3.81** — released 2006. Many modern features are missing:

- `.ONESHELL:` (3.82+)
- `&:` grouped targets (4.3+)
- `$(file ...)` (4.2+)
- Fixed jobserver (4.4+)

```bash
brew install make           # installs gmake at /opt/homebrew/bin/gmake
gmake --version             # GNU Make 4.4.x
```

The brew binary is named `gmake` to coexist with system make. Many projects detect this and use `MAKE := gmake` on macOS or fail loudly on 3.81.

---

## 17. Make Cost Model and Performance

### 17.1 Where Time Goes in a Slow Make Run

| Phase | Typical Cost | Source |
|:------|:-------------|:-------|
| Parse Makefiles | 50-500ms | Number of `include`s, `eval`s, `shell` calls |
| Build dependency database | 50-200ms | Number of rules and variables |
| Stat all prereqs | 1-5ms per file | `stat()` syscall per node in graph |
| Recipe execution | seconds-minutes | The actual work |
| Recipe shell startup | 1-5ms per line | fork+exec per recipe line |

For incremental builds where the actual work is small, parse + stat dominates. A 50000-target tree spends 250ms in `stat()` before doing any work.

### 17.2 The `:=` Caching Pattern

```makefile
# BAD: re-runs find every reference
SOURCES = $(shell find src -name '*.c')

# GOOD: snapshot once
SOURCES := $(shell find src -name '*.c')
```

For shell results, wildcards, and any computation that doesn't change during the build, prefer `:=`.

### 17.3 Avoiding Recursive Make

Recursive Make multiplies parse cost: `N` subdirectories means `N+1` parse phases. Worse, parallelism breaks down because each Make has only its own DAG. Switching to non-recursive (Section 10.3) often cuts build time by 50% on incremental rebuilds.

### 17.4 The `.PHONY-on-Clean` Stat Cost

If `clean` is NOT declared `.PHONY`, Make calls `stat("clean")` to check if a file by that name exists. On NFS or slow filesystems this matters in deeply-nested Makefiles where `clean` runs in many subdirectories.

```makefile
.PHONY: clean
```

Saves the stat call.

### 17.5 Profiling with `--debug=v`

```bash
make --debug=v 2>&1 | grep -E 'Considering|Trying|Found' | wc -l
```

Counts the number of consideration/attempt events. Compare across Makefile variants to see the impact of refactoring.

A more rigorous approach uses `time make`:

```bash
time make -n         # parse + plan only
time make -B         # full rebuild
time make            # incremental
```

The gap between `make -n` (parse + plan) and `make -B` (parse + plan + execute) shows execution cost; the gap between `make` and `make -B` shows the speedup from incremental builds.

### 17.6 ccache Integration

```bash
make CC="ccache gcc"
```

ccache caches preprocessor + compile output keyed on input content hash. Even when Make decides to rebuild a `.o` (because the `.c` mtime changed but content didn't — git checkout, for example), ccache returns the cached object in microseconds. Combine with auto-deps for fastest possible incremental builds.

### 17.7 Multi-Output Recipes

A common bug is invoking a multi-output tool with `make -j N`:

```makefile
foo.h foo.c: foo.y         # one rule, two targets
	bison foo.y
```

Make treats this as TWO independent rules. With `-j2`, both can run concurrently, calling `bison` twice (race condition; corrupted output). Fix with grouped targets (4.3+):

```makefile
foo.h foo.c &: foo.y       # GROUPED: one rule, runs once
	bison foo.y
```

Or with a single intermediate stamp file (pre-4.3 portable workaround):

```makefile
.foo.stamp: foo.y
	bison foo.y && touch $@

foo.h foo.c: .foo.stamp
```

---

## 18. Common Anti-Patterns (broken+fixed)

### 18.1 `=` for Shell Results

```makefile
# BAD
SOURCES = $(shell find src -name '*.c')

# Every reference re-runs find. With 100 references, 100 forks of find.
```

```makefile
# GOOD
SOURCES := $(shell find src -name '*.c')
```

### 18.2 PHONY Targets Named the Same as Real Files/Dirs

```makefile
# BAD
clean:
	rm -rf build

# Add a directory './clean/' and the rule never runs.
```

```makefile
# GOOD
.PHONY: clean

clean:
	rm -rf build
```

### 18.3 `$(shell ...)` That Mutates State at Parse Time

```makefile
# BAD — runs even on `make clean`, even on `make -n`
RESULT := $(shell rm -rf cache && build-cache)
```

`$(shell)` runs **at parse time**, before Make has decided whether the user even wants to run anything. `make -n` (dry run) still parses the Makefile and still runs the shell call. `make clean` similarly parses first. This idiom routinely deletes user data.

```makefile
# GOOD
.PHONY: rebuild-cache
rebuild-cache:
	rm -rf cache
	build-cache
```

Move side effects into recipes.

### 18.4 Nested-Make Where Include Would Suffice

```makefile
# BAD
all:
	$(MAKE) -C lib
	$(MAKE) -C src

# Sub-makes have no shared graph, parallelism breaks, headers don't trigger cross-dir rebuilds.
```

```makefile
# GOOD
include lib/module.mk
include src/module.mk
```

### 18.5 Relying on Order of Unrelated Prereqs

```makefile
# BAD
prog: a.o b.o c.o
	$(CC) -o $@ $^      # link order depends on parse order — fragile
```

If a future contributor reorders or `wildcard`-derives the prereq list, link order changes and may break linking against archives.

```makefile
# GOOD (when order matters explicitly)
prog: $(sort a.o b.o c.o)
	$(CC) -o $@ $^

# Or, when truly order-sensitive (with archives):
prog: a.o b.o c.o libfoo.a libbar.a libfoo.a
	$(CC) -o $@ $+      # $+ preserves duplicates and order
```

### 18.6 Forgetting `set -e` in Multi-Command Recipes

```makefile
# BAD
deploy:
	cd /var/www; rm -rf old; cp -r new old; restart-service
```

If `rm` fails, `cp` runs anyway, copying into wrong location, and `restart-service` runs against broken state.

```makefile
# GOOD
deploy:
	cd /var/www && rm -rf old && cp -r new old && restart-service

# OR
deploy:
	set -euo pipefail; \
	cd /var/www; \
	rm -rf old; \
	cp -r new old; \
	restart-service
```

### 18.7 Tab-vs-Space Confusion

Recipes MUST start with a literal tab. Editors that auto-indent with spaces silently break Make. The error is:

```
Makefile:5: *** missing separator.  Stop.
```

```makefile
# Verify by grepping for spaces at the start of recipe lines
.RECIPEPREFIX := >       # GNU 3.82+ — change recipe prefix to '>'
build:
> echo "no tabs needed"
```

`.RECIPEPREFIX` lets you redefine the prefix (rarely used; tab is universal).

---

## 19. The Make 4.x Improvements

### 19.1 Version Timeline

| Version | Year | Key Additions |
|:--------|:-----|:--------------|
| 3.81 | 2006 | The classic; macOS still ships this |
| 3.82 | 2010 | `.ONESHELL:`, `::=` (POSIX simply-expanded), `private` keyword, `.RECIPEPREFIX` |
| 4.0 | 2013 | `--trace`, `--output-sync`, `load` directive (loadable modules), `$(guile ...)` for embedded Guile |
| 4.1 | 2014 | Bug fixes, expanded `output-sync` |
| 4.2 | 2016 | `$(file ...)` function (read/write files without shell), `--shuffle` (4.4+ enhanced) |
| 4.3 | 2020 | Grouped targets `&:`, `.EXTRA_PREREQS` variable, fixed multiple-targets-with-recipe behavior |
| 4.4 | 2022 | Fixed jobserver deadlock with named-pipe (FIFO) protocol, `--shuffle=...` for build verification, `MAKEFLAGS` mode normalization |

### 19.2 Notable Per-Version Features

**3.82:** `.RECIPEPREFIX` lets a Makefile use `>` instead of TAB. `private` keyword on a variable assignment prevents inheritance to prereqs.

**4.0 — `--output-sync`:** Group recipe output so parallel builds don't interleave their stdout/stderr. Modes: `target`, `line`, `recurse`, `none`.

```bash
make -j8 --output-sync=target
```

**4.0 — `--trace`:** Print each recipe with file:line before execution. Debugger-grade.

**4.2 — `$(file ...)`:** Write large variables to a file without spawning a shell:

```makefile
$(file >cmdline.txt,$(LONG_LIST_OF_ARGS))
$(LD) @cmdline.txt
```

Avoids the shell command-line length limit (~128KB on Linux).

**4.3 — Grouped targets `&:`:** Already covered in Section 5.6. The cleanest fix for multi-output tools.

**4.4 — Jobserver fix:** Replaces the FD-based pipe with a named FIFO so non-Make-aware sub-processes can be patched to participate. Also reshapes `MAKEFLAGS` to be more parseable.

### 19.3 The macOS 3.81 Problem

Apple has not updated `/usr/bin/make` to 4.x for licensing reasons. Modern Make features fail silently on Mac unless users `brew install make` and use `gmake`:

```makefile
ifeq ($(shell uname),Darwin)
    MAKE_BIN := gmake
else
    MAKE_BIN := make
endif
```

Or reject ancient versions outright:

```makefile
ifneq (4,$(firstword $(subst ., ,$(MAKE_VERSION))))
$(error GNU Make 4.0+ required; you have $(MAKE_VERSION))
endif
```

### 19.4 The `--shuffle` Flag (4.4+ enhanced)

```bash
make --shuffle=random
make --shuffle=reverse
make --shuffle=42        # specific seed
```

Randomizes prereq order before scheduling. Surfaces bugs in Makefiles that accidentally depend on sequence (Section 18.5). Run shuffled builds in CI to catch ordering bugs.

---

## 20. Idioms at the Internals Depth

### 20.1 The Canonical Non-Recursive Skeleton

```makefile
# Top-level Makefile
SHELL    := /bin/bash
.SHELLFLAGS := -ec
.SUFFIXES:                     # disable legacy suffix rules
.DELETE_ON_ERROR:              # delete partial outputs on failure
.DEFAULT_GOAL := all

# Top-level config
PREFIX  ?= /usr/local
BUILD   ?= build
CC      ?= cc
CFLAGS  ?= -O2 -Wall -Wextra
LDFLAGS ?=

# Aggregator variables — modules append to these
ALL_OBJS  :=
ALL_BINS  :=
ALL_CLEAN :=

# Per-module .mk files contribute rules and variables
include lib/module.mk
include src/module.mk
include test/module.mk

# Top-level targets
.PHONY: all clean install test
all: $(ALL_BINS)

clean:
	rm -rf $(BUILD) $(ALL_CLEAN)

install: all
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(ALL_BINS) $(DESTDIR)$(PREFIX)/bin

test: $(ALL_BINS)
	$(MAKE) -C test run
```

```makefile
# lib/module.mk
LIB_DIR  := lib
LIB_SRCS := $(wildcard $(LIB_DIR)/*.c)
LIB_OBJS := $(LIB_SRCS:%.c=$(BUILD)/%.o)
LIB_DEPS := $(LIB_OBJS:.o=.d)

$(BUILD)/$(LIB_DIR)/%.o: $(LIB_DIR)/%.c
	@mkdir -p $(@D)
	$(CC) $(CFLAGS) -MMD -MP -MF $(@:.o=.d) -MT $@ -c $< -o $@

$(BUILD)/libfoo.a: $(LIB_OBJS)
	ar rcs $@ $^

ALL_OBJS  += $(LIB_OBJS)
ALL_CLEAN += $(BUILD)/libfoo.a

-include $(LIB_DEPS)
```

Each module fragment contributes to `ALL_*` aggregators. The single top-level Make sees the full DAG.

### 20.2 Auto-Deps + Per-Arch + Per-Config Matrix

```makefile
# Build a matrix: {linux, darwin} x {debug, release} x {x86_64, arm64}

PLATFORMS := linux darwin
CONFIGS   := debug release
ARCHS     := x86_64 arm64

# Generate the build directory tree
BUILD_DIRS := $(foreach p,$(PLATFORMS),\
              $(foreach c,$(CONFIGS),\
              $(foreach a,$(ARCHS),build/$(p)/$(c)/$(a))))

# Per-axis CFLAGS
CFLAGS_debug   := -O0 -g -DDEBUG
CFLAGS_release := -O2 -DNDEBUG
CFLAGS_x86_64  := -m64
CFLAGS_arm64   := -mcpu=apple-m1

# Generic compile pattern
.SECONDEXPANSION:
build/%/.dirstamp:
	@mkdir -p $(@D) && touch $@

# Use a function to generate per-cell rules
define COMPILE_TEMPLATE
build/$(1)/$(2)/$(3)/%.o: src/%.c | build/$(1)/$(2)/$(3)/.dirstamp
	$$(CC) $$(CFLAGS) $$(CFLAGS_$(2)) $$(CFLAGS_$(3)) \
	    -MMD -MP -MF $$(@:.o=.d) -MT $$@ -c $$< -o $$@
endef

$(foreach p,$(PLATFORMS),\
$(foreach c,$(CONFIGS),\
$(foreach a,$(ARCHS),\
$(eval $(call COMPILE_TEMPLATE,$(p),$(c),$(a))))))
```

The `$$(CC)` and `$$(@...)` in the template defer expansion to rule-firing time. Without the double-dollar, Make would expand the variables when `eval` runs and fail.

### 20.3 The `help` Target

A self-documenting Makefile:

```makefile
.PHONY: help

help:  ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	$(CC) $(CFLAGS) -o app main.c

test: ## Run tests
	./run-tests.sh

clean: ## Remove build artifacts
	rm -rf build app
```

`make help` greps the Makefile itself for `target: ## description` patterns and formats them. The `\033[36m` is ANSI cyan. Idiomatic in many open-source projects.

### 20.4 BUILDDIR + VPATH for Out-of-Tree Builds

Keep generated files in a separate directory tree:

```makefile
SRCDIR   := src
BUILDDIR := build

VPATH := $(SRCDIR)        # Make searches here for prereqs

SRCS := $(notdir $(wildcard $(SRCDIR)/*.c))
OBJS := $(SRCS:%.c=$(BUILDDIR)/%.o)

$(BUILDDIR)/%.o: %.c | $(BUILDDIR)
	$(CC) $(CFLAGS) -c $< -o $@

$(BUILDDIR):
	mkdir -p $@
```

`VPATH` tells Make where to look for prereqs that aren't in the current directory. With `VPATH := src`, the rule `%.o: %.c` finds `main.c` in `src/main.c`. `$<` expands to the resolved path, so `$(CC) -c $<` works correctly.

Useful when the same source tree is built into multiple `BUILDDIR`s (debug + release in parallel).

### 20.5 Print-VAR Debug Pattern

```makefile
print-%:
	@echo '$* = $($*)'
	@echo '  origin = $(origin $*)'
	@echo '  flavor = $(flavor $*)'
	@echo '  value  = $(value $*)'
```

Then:

```bash
make print-CFLAGS
# CFLAGS = -O2 -Wall -Wextra
#   origin = file
#   flavor = recursive
#   value  = -O2 -Wall -Wextra
```

Stem-based debugging — works for any variable name. Equivalent of a Python REPL in Makefile-land.

### 20.6 Grouped Outputs Idiom (Pre-4.3)

Before grouped targets:

```makefile
.foo.stamp: foo.y
	bison foo.y && touch $@

foo.h foo.c: .foo.stamp
	@true                     # no-op; the stamp's recipe already produced these
```

The stamp file is the "real" target; the headers/sources are siblings. Make sees foo.h depending on .foo.stamp, .foo.stamp depending on foo.y, and runs bison only when foo.y changes.

### 20.7 The `MAKECMDGOALS` Trick

```makefile
ifneq ($(MAKECMDGOALS),clean)
-include $(DEPS)
endif
```

Skip the dep-file include when the user is just cleaning. Avoids the cost of regenerating .d files just to delete them.

`MAKECMDGOALS` is the list of goals on the command line (or empty if user ran `make` with no args, in which case `.DEFAULT_GOAL` applies).

---

## 21. Prerequisites

- Shell scripting basics (variables, exit codes, subshell semantics, `set -e`).
- Compilation pipeline for at least one language (e.g., C: source -> object -> link).
- Filesystem timestamps and the `stat()` syscall.
- Directed acyclic graphs and topological sorting.
- Process forking and the cost of `fork()` + `execve()`.
- Pipes and FIFOs (for understanding the jobserver protocol).
- Pattern matching with `%` (similar to glob's `*` but anchored to one stem).

## Complexity

| Operation | Time | Notes |
|:----------|:-----|:------|
| Parse Makefile | O(N) | N = number of lines + included files |
| Build dependency database | O(R + V) | R = rules, V = variables |
| Topological sort + traversal | O(V + E) | V = targets, E = edges |
| Stat all prereqs | O(V) syscalls | One stat per node |
| Recipe execution | O(W) | W = real work; varies by recipe |
| Implicit rule chain search | O(P^k) | P = patterns, k = chain depth (bounded ~1) |
| `$(shell ...)` | O(1) fork+exec | ~1ms per call on Linux |
| `$(wildcard ...)` | O(F) | F = files in matched directories |
| Parallel build with `-j N` | O((V+E)/N) | Bounded by graph width and N |
| Jobserver acquire/release | O(1) | One read/write on the token pipe |

## See Also

- `make` (sheet) — practical command and syntax reference
- `polyglot` — using Make alongside Python, Go, Rust, JavaScript
- `c` — the language Make was designed to build
- `bash` — the shell that runs every recipe by default

## References

- *GNU Make Manual* — https://www.gnu.org/software/make/manual/
- *Managing Projects with GNU Make*, Robert Mecklenburg, O'Reilly, 3rd ed (2004) — the standard book.
- *Recursive Make Considered Harmful*, Peter Miller (1998) — http://aegis.sourceforge.net/auug97.pdf
- *mrbook.org Make Tutorial* — https://www.mrbook.org/blog/tutorials/make/
- *Makefile Tutorial by Example* — https://makefiletutorial.com/
- POSIX Make specification — IEEE Std 1003.1-2024, Utility Conventions, `make`.
- BSD Make (`bmake`) manpage — https://man.freebsd.org/cgi/man.cgi?make(1)
- GNU Make NEWS file (per-version changes) — https://git.savannah.gnu.org/cgit/make.git/tree/NEWS
- CppCon talks on build systems and the comparison with Bazel/Ninja — useful context for when Make's two-phase model breaks down at scale.
