# Make (Build Automation Language)

GNU Make is a dependency-driven build automation tool. You write rules — target plus prerequisites plus recipe — and `make` decides what is out-of-date and runs the minimum work to rebuild it. Recipes execute in a sub-shell. Indentation is a literal TAB. The default executable name is `make` on Linux, but on macOS/BSD it is `bmake` or `pmake` and the GNU implementation is shipped as `gmake` from Homebrew.

## Setup

GNU Make is the de facto standard. BSD make (NetBSD/FreeBSD) and Microsoft NMAKE are different dialects with overlapping syntax — most non-trivial Makefiles target GNU Make explicitly.

### Linux

```bash
sudo apt install build-essential       # Debian/Ubuntu — pulls make + gcc
sudo dnf install make                  # Fedora/RHEL
sudo pacman -S make                    # Arch
make --version                         # confirm GNU Make 4.x
```

### macOS

```bash
xcode-select --install                 # ships GNU Make 3.81 (very old)
brew install make                      # installs GNU Make 4.4 as `gmake`
gmake --version                        # GNU Make 4.4.x
echo 'export PATH="/opt/homebrew/opt/make/libexec/gnubin:$PATH"' >> ~/.zshrc
```

### Windows

```bash
choco install make                     # via Chocolatey
scoop install make                     # via Scoop
winget install GnuWin32.Make           # via winget
# Or use WSL2: sudo apt install make
```

### The canonical first Makefile

```bash
# file: Makefile

hello:
	echo "Hello, world"
```

```bash
$ make
echo "Hello, world"
Hello, world
```

The recipe line MUST be indented with a TAB character. If your editor inserts spaces, you get the legendary error: `Makefile:3: *** missing separator (did you mean TAB instead of 8 spaces?).  Stop.`

## Hello World

A minimal Makefile demonstrates the core triad: target, prerequisites, recipe.

```bash
# file: Makefile

hello: hello.c
	gcc -o hello hello.c

clean:
	rm -f hello
```

```bash
$ make           # builds the first target (hello)
gcc -o hello hello.c

$ make hello     # explicit target name
make: 'hello' is up to date.

$ make clean     # named target
rm -f hello
```

Run `make` with no argument and it builds the first non-special rule (here `hello`). Run `make <target>` to build a specific named target.

## Anatomy of a Rule

Every rule has three parts:

```bash
target: prerequisites
	recipe-line-1
	recipe-line-2
```

- **target** — the file to build (or a phony name)
- **prerequisites** — files (or other targets) the target depends on
- **recipe** — shell commands to run, each line indented by a literal TAB

The recipe runs only if any prerequisite is newer than the target (mtime comparison) OR the target does not exist.

```bash
app.o: app.c app.h
	gcc -c -o app.o app.c
```

Multiple prerequisites are space-separated. Multiple recipe lines each get their own sub-shell:

```bash
build:
	cd src
	make            # this make runs in the parent dir, NOT src/
```

Each TAB-indented line is a fresh shell. The `cd src` does not persist. Either join with `&&` plus backslash, or use `.ONESHELL`.

### The TAB requirement

Recipes MUST start with a literal TAB (`\t`, 0x09). Spaces will not work.

```bash
target:
    echo "this is 4 spaces, not TAB"
# Makefile:2: *** missing separator (did you mean TAB instead of 8 spaces?).  Stop.
```

GNU Make 4.0+ allows overriding via `.RECIPEPREFIX`:

```bash
.RECIPEPREFIX := >
target:
> echo "now indented with > instead of TAB"
```

## The Default Target

Without `.DEFAULT_GOAL`, Make builds the first non-special rule it finds:

```bash
all: app           # this is the default goal because it's first
app: app.o
	gcc -o app app.o
```

To override, set `.DEFAULT_GOAL`:

```bash
.DEFAULT_GOAL := test

build:
	go build .

test:
	go test ./...
```

Now `make` runs `make test` by default.

`MAKECMDGOALS` holds whatever the user typed: `make foo bar` -> `MAKECMDGOALS = foo bar`.

## Variables — Forms

GNU Make has multiple assignment operators with subtly different semantics:

```bash
VAR  = value             # recursively expanded — re-evaluated on each use
VAR := value             # simply expanded — evaluated ONCE at parse time
VAR ::= value            # GNU 4.0+ alias for := (POSIX-blessed)
VAR ?= value             # set only if VAR not yet defined
VAR += more              # append (preserves flavor — recursive stays recursive)
VAR != cmd               # GNU 4.0+ shorthand for VAR := $(shell cmd)
```

```bash
NAME := app
SRC  := main.c util.c
OBJ  := $(SRC:.c=.o)            # main.o util.o (substitution reference)
CFLAGS ?= -O2 -Wall              # honor user-set CFLAGS
CFLAGS += -g                     # always add -g
```

Override on command line — command-line values beat in-file values:

```bash
make CFLAGS="-O0 -g -DDEBUG"
```

Environment variables are imported automatically (use `$(origin VAR)` to detect source). Use `override VAR := ...` to force the in-Makefile value.

## Recursive vs Simple Variables

This is the single biggest landmine in GNU Make. Read carefully.

```bash
A = $(B)                # recursive — B referenced lazily
B = hello
$(info A is $(A))       # prints: A is hello

C := $(D)               # simple — D evaluated NOW
D = hello
$(info C is $(C))       # prints: C is   (empty — D was undefined when C was assigned)
```

Use `:=` for anything you want to evaluate once (commands, paths, sums of other vars). Use `=` only when you genuinely want late binding.

The classic recursive landmine — infinite recursion:

```bash
# BROKEN
CC = $(CC) -Wall          # *** Recursive variable 'CC' references itself (eventually).  Stop.
```

```bash
# FIXED
CC := gcc
CC := $(CC) -Wall
```

`$(shell ...)` is especially dangerous in `=`. Every reference re-runs the command:

```bash
# BROKEN — runs `git rev-parse` on every reference, slow
SHA = $(shell git rev-parse --short HEAD)
```

```bash
# FIXED — runs once
SHA := $(shell git rev-parse --short HEAD)
```

## Automatic Variables

Available inside every recipe — they reference the current rule context:

```bash
$@      # the target file name
$<      # first prerequisite
$^      # ALL prerequisites, deduplicated, space-separated
$?      # prerequisites NEWER than the target (out-of-date)
$*      # the stem matched by % in a pattern rule
$+      # ALL prerequisites WITH duplicates (rare)
$|      # order-only prerequisites
$(@D)   # directory part of $@ (no trailing slash)
$(@F)   # file part of $@
$(<D)   # directory part of $<
$(<F)   # file part of $<
$(^D)   # directory parts of $^
$(^F)   # file parts of $^
$(*D)   # directory part of stem
$(*F)   # file part of stem
$(?D)   # directory parts of $?
$(?F)   # file parts of $?
```

Demo:

```bash
build/app: src/main.o src/util.o include/app.h | build
	gcc -o $@ $(filter %.o,$^)
# $@   = build/app
# $<   = src/main.o
# $^   = src/main.o src/util.o include/app.h
# $?   = whichever prereqs are newer than build/app
# $|   = build
# $(@D)= build, $(@F) = app
# $(<D)= src, $(<F) = main.o
```

## Built-in Variables

Make pre-defines many variables for common toolchains. You can override any of them.

```bash
CC          # C compiler, default: cc
CXX         # C++ compiler, default: g++
CPP         # C preprocessor, default: $(CC) -E
AS          # assembler, default: as
AR          # archiver for static libs, default: ar
LD          # linker, default: ld
RM          # remove command, default: rm -f
MAKE        # path to make itself — USE THIS, never `make`
MAKEFLAGS   # flags passed to current make invocation
SHELL       # the shell — DEFAULT IS /bin/sh, NOT /bin/bash
.SHELLFLAGS # flags to SHELL, default: -c
CFLAGS      # C compiler flags, default empty
CXXFLAGS    # C++ compiler flags
CPPFLAGS    # preprocessor flags (-I, -D)
LDFLAGS     # linker flags (-L, -Wl,...)
LDLIBS      # libs to link (-l)
TARGET_ARCH # arch flags for cross-compile
ARFLAGS     # ar flags, default: rv
.RECIPEPREFIX  # GNU 4.0+ — override TAB recipe indent
```

The SHELL gotcha:

```bash
# BROKEN — bashisms in /bin/sh
target:
	arr=(a b c); echo "${arr[0]}"
# /bin/sh: 1: Syntax error: "(" unexpected
```

```bash
# FIXED — explicit bash
SHELL := /bin/bash
target:
	arr=(a b c); echo "$${arr[0]}"
```

## Pattern Rules

Pattern rules use `%` as a wildcard. The matched portion (the stem) is `$*`.

```bash
%.o: %.c
	$(CC) $(CPPFLAGS) $(CFLAGS) -c -o $@ $<
```

This rule says: to build any `.o` file, look for the matching `.c` file, then run the recipe. The `%` matches the same stem on both sides.

Multiple pattern targets:

```bash
%.so %.dylib: %.o
	$(CC) -shared -o $@ $<
```

Static pattern rules — apply pattern only to specific targets:

```bash
OBJECTS := main.o util.o parse.o
$(OBJECTS): %.o: %.c
	$(CC) $(CFLAGS) -c -o $@ $<
```

The form is `targets: target-pattern: prereq-patterns`. Only `main.o util.o parse.o` are affected; other `.o` files are unaffected.

Pattern rule with multiple stems is not allowed — you cannot have two `%` in one target.

## Implicit Rules

GNU Make ships a built-in catalogue of implicit rules. The most common:

```bash
%.o:  %.c          # uses $(CC) -c $(CPPFLAGS) $(CFLAGS)
%.o:  %.cc         # uses $(CXX) -c $(CPPFLAGS) $(CXXFLAGS)
%.o:  %.cpp        # ditto
%.o:  %.cxx        # ditto
%.o:  %.s          # uses $(AS) $(ASFLAGS)
%:    %.c          # link single-file C program
%:    %.o          # link from object
%.a:  %.o          # archive
%:    %.sh         # cp + chmod +x
```

Inspect the full list:

```bash
make -p | less     # dump database including all implicit rules
make -r            # disable built-in rules (recommended for clean builds)
make -R            # disable built-in variables AND rules
```

For reproducible builds, many projects start with:

```bash
MAKEFLAGS += --no-builtin-rules --no-builtin-variables
```

## Suffix Rules

Legacy POSIX form, predates pattern rules:

```bash
.SUFFIXES:                 # clear default suffix list
.SUFFIXES: .c .o

.c.o:                      # equivalent to %.o: %.c
	$(CC) $(CFLAGS) -c $<
```

Use pattern rules in new code. Suffix rules exist for compatibility with antique Makefiles and POSIX make.

## .PHONY Targets

A phony target is a name that does NOT correspond to a file. Without `.PHONY`, Make checks whether a file with the target name exists, and if it is up to date the recipe is skipped.

```bash
.PHONY: all clean install test

all: app
clean:
	rm -f *.o app
```

If a directory or file named `clean` ever appears, `make clean` silently does nothing — until you `.PHONY` it.

Common phony targets to declare: `all`, `clean`, `distclean`, `install`, `uninstall`, `test`, `check`, `dist`, `lint`, `fmt`, `help`, `release`, `docs`.

## Special Targets

These start with `.` and have built-in meaning. Declare them at the top of the Makefile.

```bash
.PHONY:                  # marks targets as not files (see above)
.SUFFIXES:               # clear suffix list — `.SUFFIXES:` alone removes defaults
.DEFAULT_GOAL := test    # override "first rule wins"
.DEFAULT:                # recipe to run for any target with no rule
	@echo "no rule for $@"
.INTERMEDIATE: %.o       # treat as intermediate — Make deletes after build
.SECONDARY: %.o          # like intermediate but Make does NOT delete
.DELETE_ON_ERROR:        # delete target file if recipe fails — VERY USEFUL
.NOTPARALLEL:            # disable parallelism for this Makefile
.ONESHELL:               # run all recipe lines in ONE shell
.POSIX:                  # request POSIX-conformant behavior
.SECONDEXPANSION:        # enable second expansion of prerequisites
.EXPORT_ALL_VARIABLES:   # export all variables to recipe environment
.IGNORE:                 # ignore errors from recipes (per-target or global)
.SILENT:                 # silence all recipes (per-target or global)
.LOW_RESOLUTION_TIME:    # treat seconds-resolution timestamps as equal
```

`.DELETE_ON_ERROR` deserves a callout — it is almost always what you want:

```bash
.DELETE_ON_ERROR:

%.gz: %
	gzip -c $< > $@      # if gzip fails, partial $@ would be left behind
```

Without `.DELETE_ON_ERROR`, a failed recipe can leave a corrupted output file that Make considers up-to-date next run.

## Functions — Text

```bash
$(subst from,to,text)              # straight string substitution
$(patsubst pattern,replacement,text)  # pattern-based, % is the wildcard
$(strip text)                      # collapse whitespace, trim
$(findstring find,in)              # returns find if substring, else empty
$(filter pattern,text)             # keep words matching any pattern
$(filter-out pattern,text)         # remove words matching any pattern
$(sort list)                       # sort + dedupe
$(word n,text)                     # nth word (1-indexed)
$(words text)                      # word count
$(wordlist start,end,text)         # slice (1-indexed inclusive)
$(firstword text)                  # first word
$(lastword text)                   # last word
```

```bash
SRC := main.c util.c parse.c test.c
OBJ := $(patsubst %.c,%.o,$(SRC))           # main.o util.o parse.o test.o
SRC_NO_TEST := $(filter-out test.c,$(SRC))  # main.c util.c parse.c
SORTED := $(sort $(SRC))                    # main.c parse.c test.c util.c
COUNT := $(words $(SRC))                    # 4
THIRD := $(word 3,$(SRC))                   # parse.c
```

## Functions — File Names

```bash
$(dir names)         # directory part(s), with trailing /
$(notdir names)      # file part(s)
$(suffix names)      # extension(s) including the dot
$(basename names)    # name(s) without extension
$(addsuffix s,names) # append s to each
$(addprefix p,names) # prepend p to each
$(join l1,l2)        # pairwise concat
$(wildcard pattern)  # glob expansion at parse time
$(realpath names)    # resolve symlinks + . + .. — empty if not exist
$(abspath names)     # absolute path WITHOUT resolving symlinks/existence
```

```bash
SRCS := $(wildcard src/*.c src/**/*.c)
$(dir src/main.c)         # src/
$(notdir src/main.c)      # main.c
$(suffix src/main.c.tar)  # .tar
$(basename src/main.c)    # src/main
$(addsuffix .o,a b c)     # a.o b.o c.o
$(addprefix obj/,a.o b.o) # obj/a.o obj/b.o
```

`$(wildcard ...)` returns empty if nothing matches — silent failure mode. Recursive wildcards require a helper (see Recursive Wildcard Pattern below).

## Functions — Conditional

```bash
$(if condition,then-part,else-part)   # condition non-empty -> then, else else
$(or  cond1,cond2,...)                # first non-empty
$(and cond1,cond2,...)                # last non-empty if all non-empty, else empty
```

```bash
DEBUG ?= 0
OPT := $(if $(filter 1,$(DEBUG)),-O0 -g,-O2)

CFLAGS := $(or $(CUSTOM_CFLAGS),-O2 -Wall)
```

These run at parse time, not at recipe time — they cannot test runtime state.

## Functions — Foreach and Call

`$(foreach var,list,text)` — iterate over a list, expanding `text` for each element with `$(var)` bound:

```bash
DIRS := src lib bin
PATHS := $(foreach d,$(DIRS),$(d)/.dirstamp)
# PATHS = src/.dirstamp lib/.dirstamp bin/.dirstamp
```

`$(call var,a,b,...)` — invoke a parametric variable. Inside the callee, `$(0)` is the function name, `$(1)..$(9)` are arguments:

```bash
define greet
@echo "Hello, $(1) — you are $(2) years old"
endef

target:
	$(call greet,Alice,30)
	$(call greet,Bob,25)
```

`define ... endef` declares multi-line variables (recursive flavor by default; `define VAR :=` for simple).

## Functions — Shell

`$(shell cmd)` runs a shell command at parse time and substitutes the trimmed stdout.

```bash
SHA := $(shell git rev-parse --short HEAD)
DATE := $(shell date +%Y-%m-%d)
NCPU := $(shell nproc 2>/dev/null || sysctl -n hw.ncpu)
```

CRITICAL: every reference to a recursive variable holding `$(shell ...)` re-runs the command. Always use `:=` to cache.

```bash
# BROKEN — git runs once per reference (could be hundreds of times)
SHA = $(shell git rev-parse --short HEAD)
```

```bash
# FIXED
SHA := $(shell git rev-parse --short HEAD)
```

`$(shell ...)` has no error handling — failures yield empty output. Capture exit status via the `.SHELLSTATUS` variable (GNU 4.2+):

```bash
RESULT := $(shell ls /nonexistent)
$(if $(filter-out 0,$(.SHELLSTATUS)),$(error ls failed))
```

## Functions — Origin and Flavor

`$(origin var)` returns where `var` came from:

```bash
undefined            # never defined
default              # built-in default like CC
environment          # from environment, not overridden by Makefile
environment override # from environment, overridden by Makefile (-e flag)
file                 # set in this Makefile
command line         # set on the make command line
override             # set with `override` directive
automatic            # automatic variable like $@
```

```bash
ifeq ($(origin CC),default)
$(warning CC is the default — set CC=clang explicitly)
endif
```

`$(flavor var)` returns:

```bash
undefined            # not defined
recursive            # = assignment
simple               # := assignment
```

## Conditionals

Make conditionals run at parse time. They control which lines of the Makefile are read.

```bash
ifeq (a,b)           # equal? — supports ($(VAR),"value") or "$(VAR)" "value"
ifneq ($(VAR),)      # not equal
ifdef VAR            # variable has non-empty value
ifndef VAR           # variable is empty/undefined
else                 # optional
else ifeq (...)      # chained
endif                # required
```

```bash
ifdef DEBUG
CFLAGS += -g -O0 -DDEBUG
else
CFLAGS += -O2
endif

ifeq ($(OS),Windows_NT)
EXE := app.exe
RM_F := del /f
else ifeq ($(shell uname -s),Darwin)
EXE := app
SHARED_EXT := .dylib
else
EXE := app
SHARED_EXT := .so
endif
```

These do NOT work inside recipes — recipes are shell, not Make. For runtime conditionals use shell `if`.

## Include directive

```bash
include other.mk         # error if missing
-include optional.mk     # silent if missing
sinclude optional.mk     # synonym for -include
```

Common idiom — include all auto-generated dependency files:

```bash
DEPS := $(OBJ:.o=.d)
-include $(DEPS)
```

If `make` cannot find an included file, it tries to build it as a target first, then re-reads the Makefile.

## Auto-generated Dependencies

Avoid maintaining header dependencies by hand. Have the compiler emit them:

```bash
DEPDIR := .deps
DEPFLAGS = -MMD -MP -MF $(DEPDIR)/$*.d -MT $@

%.o: %.c | $(DEPDIR)
	$(CC) $(DEPFLAGS) $(CFLAGS) -c -o $@ $<

$(DEPDIR):
	@mkdir -p $@

DEPS := $(SRC:%.c=$(DEPDIR)/%.d)
-include $(DEPS)
```

Flags explained:

```bash
-MMD     # emit .d alongside .o, ignoring system headers
-MP      # emit phony empty rule for each header — survives header deletions
-MF      # specify dependency output file
-MT      # set target name in the .d file (so it matches the .o)
```

This is the canonical idiom for any non-trivial C/C++ build.

## Multiple Targets per Rule

Two flavors with very different semantics.

### Same recipe runs ONCE PER TARGET

```bash
foo bar baz: input
	process $< -o $@
# `process input -o foo`, then `process input -o bar`, then `process input -o baz`
```

This is identical to writing three separate rules. Useful when each target is built independently from the same prerequisites.

### Grouped targets — recipe runs ONCE FOR ALL targets (GNU 4.3+)

```bash
foo bar baz &: input
	one-shot-tool input        # produces foo, bar, AND baz in a single invocation
```

The `&:` form tells Make that one recipe invocation produces ALL listed targets. Common when a tool emits multiple files (parser generators, codegen). Without `&:`, Make would call the recipe up to N times in parallel and trample the outputs.

## Double-Colon Rules

Allow multiple independent recipes for the same target:

```bash
log::
	@echo "first updater"

log::
	@date >> log
```

Each `log::` rule is independent — both run if `log` is requested. Single-colon rules disallow this.

Used for: log appends, daemon reload hooks, accumulating side effects.

## Order-Only Prerequisites

Place after `|` — Make ensures they exist but does NOT trigger rebuild on timestamp:

```bash
$(OBJDIR):
	mkdir -p $@

$(OBJDIR)/%.o: %.c | $(OBJDIR)
	$(CC) -c -o $@ $<
```

Without `|`, every change to `$(OBJDIR)`'s mtime would rebuild every `.o`. With `|`, Make merely guarantees `$(OBJDIR)` exists.

## Recipes — Per-Line Behavior

Each TAB-indented line in a recipe is its own sub-shell. Variables, `cd`, environment changes do not persist across lines.

```bash
# BROKEN — cd doesn't persist
build:
	cd src
	$(CC) -o app *.c
```

```bash
# FIXED — chain in one shell
build:
	cd src && $(CC) -o app *.c
```

```bash
# FIXED — line continuation
build:
	cd src && \
	$(CC) -o app *.c
```

```bash
# FIXED — .ONESHELL
.ONESHELL:
build:
	cd src
	$(CC) -o app *.c
```

### Recipe line prefixes

```bash
@cmd     # do not echo cmd before running (silent)
-cmd     # ignore cmd's exit status — keep going on failure
+cmd     # honor cmd even under -n / -t / -q (for $(MAKE))
```

```bash
@echo "configuring"        # quiet
-rm -f maybe_missing       # don't fail if file missing
+$(MAKE) -C subdir         # always invoke recursive make
```

Combine: `@-cmd` (silent + ignore-error). Order does not matter.

### .ONESHELL caveats

Under `.ONESHELL`, all lines pass to a single shell invocation. If the shell is `/bin/sh` and you set `.SHELLFLAGS := -ec`, every line errors out. Without `-e`, only the LAST line's exit status matters:

```bash
.ONESHELL:
.SHELLFLAGS := -ec     # -e: exit on error, -c: command from arg

target:
	false              # without -e, this is silently ignored
	echo "still ran"   # would print and rule succeeds
```

## Recipe Variables in Recipes

Inside a recipe, `$` is Make's variable sigil. To get a literal `$` in the shell, use `$$`:

```bash
target:
	echo "PID is $$$$"            # bash: $$ is shell PID. Make sees $$$$ -> $$
	for f in *.c; do echo $$f; done
	echo "Make var: $(CC)"        # Make expands $(CC)
	echo "Shell var: $$HOME"      # passes literal $HOME to shell
```

Quick rules:

```bash
$@         # Make automatic var
$$         # literal $ in shell
$$$$       # literal $$ in shell (e.g., bash PID)
$(VAR)     # Make variable expansion
$${VAR}    # shell ${VAR} (escaped)
```

To export a Make variable into a recipe's environment:

```bash
export DESTDIR
install:
	./install.sh        # script sees $DESTDIR
```

Or set on the command line of the recipe:

```bash
target:
	DESTDIR=/tmp ./install.sh
```

## Variable Scoping

Target-specific variables override globals only inside that target's recipe (and its prereqs):

```bash
CFLAGS := -O2

debug: CFLAGS := -g -O0
debug: app

release: CFLAGS := -O3 -DNDEBUG
release: app
```

`make debug` builds `app` with `-g -O0`, `make release` with `-O3 -DNDEBUG`.

Pattern-specific:

```bash
%.fast.o: CFLAGS := -O3 -march=native
%.fast.o: %.c
	$(CC) $(CFLAGS) -c -o $@ $<
```

`export VAR` makes `VAR` an environment variable for child processes:

```bash
export PATH := $(PWD)/tools:$(PATH)

build:
	./build.sh        # sees the augmented PATH
```

`unexport VAR` removes from environment. `.EXPORT_ALL_VARIABLES:` exports everything.

## Recursive Make

Use `$(MAKE)` (not `make`) when invoking sub-makes — it preserves `MAKEFLAGS`, `-j`, `-n`, etc:

```bash
.PHONY: all
all:
	$(MAKE) -C src
	$(MAKE) -C lib
	$(MAKE) -C tests
```

`-C dir` changes directory before running. The `+` prefix on a recursive recipe ensures it still runs under `-n` (dry-run) so the parent can see what sub-makes would do.

### "Recursive Make Considered Harmful"

The famous Peter Miller paper (1998) identified problems:

- Sub-makes have incomplete dependency graphs — they can't see prereqs in sibling dirs
- Parallelism suffers — each sub-make has its own job server
- Total work is worst case multiple times the actual work

Modern alternatives — non-recursive Make using `include`:

```bash
# top-level Makefile
DIRS := src lib tests
SRC :=
include $(addsuffix /module.mk,$(DIRS))

# src/module.mk
SRC += src/main.c src/util.c
```

GNU Make 3.81+ has a job-server protocol that mostly addresses the parallelism issue in recursive scenarios, but the dependency-graph fragmentation remains.

## Parallel Make

```bash
make -j        # unlimited parallel jobs (DANGEROUS — fork bomb on big builds)
make -j8       # 8 parallel jobs
make -j$(nproc)
make -l 4.0    # respect 4.0 load avg — pause new jobs above
make -O        # serialize output per target (4.0+)
make -O=line   # output per line
make -O=target # output per target (default with -O)
make -O=recurse # serialize all output from sub-makes
```

`.NOTPARALLEL:` disables parallelism for the entire Makefile (rare — hurts throughput).

The output-interleaving problem: with `-j N`, two targets' stdout/stderr blend on screen. `-O` solves this by buffering and flushing per-target.

For correctness, every prerequisite must be declared. Missing prereqs work fine serially but corrupt under `-j`.

## Debugging

```bash
make -n                   # dry run — print what would run, do nothing
make -n target            # for a specific target
make -B                   # always-make — pretend everything is out of date
make -W src/main.c        # pretend src/main.c was just modified
make -d                   # full debug — VERY verbose
make --debug=basic        # only why-rebuilt info
make --debug=verbose      # parsing + chosen rules
make --debug=implicit     # show implicit rule choices
make --debug=jobs         # job-control info
make --debug=makefile     # parse-time info
make --debug=v            # variable assignments (4.4+)
make -p                   # print database — all rules + variables
make -p -f /dev/null      # built-in database only
make --trace              # GNU 4.0+ — show each recipe expansion
make -q target            # quiet — exit 0 if up-to-date, 1 if not
```

In the Makefile itself:

```bash
$(info CFLAGS=$(CFLAGS))           # print at parse time, continue
$(warning unusual condition)       # like info but with file:line prefix
$(error abort with this message)   # parse aborts
```

```bash
ifeq ($(strip $(CC)),)
$(error CC is empty)
endif
```

## Common Compile Targets — C/C++

Single-file C program:

```bash
hello: hello.c
	$(CC) $(CFLAGS) -o $@ $<
```

Multi-file C with separate objects:

```bash
CC      := gcc
CFLAGS  := -O2 -Wall -Wextra -std=c11
LDFLAGS :=
LDLIBS  := -lm

SRC := main.c util.c parse.c
OBJ := $(SRC:.c=.o)

app: $(OBJ)
	$(CC) $(LDFLAGS) -o $@ $^ $(LDLIBS)

%.o: %.c
	$(CC) $(CFLAGS) -c -o $@ $<

clean:
	rm -f $(OBJ) app

.PHONY: clean
```

Static library:

```bash
libfoo.a: $(OBJ)
	$(AR) rcs $@ $^
```

Shared library (Linux):

```bash
libfoo.so: $(OBJ)
	$(CC) -shared -fPIC -o $@ $^

%.o: %.c
	$(CC) -fPIC $(CFLAGS) -c -o $@ $<
```

Shared library (macOS):

```bash
libfoo.dylib: $(OBJ)
	$(CC) -dynamiclib -install_name @rpath/libfoo.dylib -o $@ $^
```

Linking against the lib:

```bash
LDFLAGS += -L. -Wl,-rpath,'$$ORIGIN'
LDLIBS  += -lfoo

app: app.o libfoo.so
	$(CC) $(LDFLAGS) -o $@ $< $(LDLIBS)
```

Note: pkg-config integration:

```bash
PKGS    := gtk+-3.0 sqlite3
CFLAGS  += $(shell pkg-config --cflags $(PKGS))
LDLIBS  += $(shell pkg-config --libs   $(PKGS))
```

## Cross-Platform Snippets

```bash
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
	OS := linux
	SHARED_EXT := .so
	SHARED_FLAGS := -shared -fPIC
endif

ifeq ($(UNAME_S),Darwin)
	OS := macos
	SHARED_EXT := .dylib
	SHARED_FLAGS := -dynamiclib
endif

ifeq ($(OS),Windows_NT)
	OS := windows
	SHARED_EXT := .dll
	EXE_EXT := .exe
	RM := del /f /q
	MKDIR := mkdir
else
	EXE_EXT :=
	RM := rm -f
	MKDIR := mkdir -p
endif

ifeq ($(UNAME_M),x86_64)
	ARCH := amd64
endif
ifeq ($(UNAME_M),arm64)
	ARCH := arm64
endif
ifeq ($(UNAME_M),aarch64)
	ARCH := arm64
endif
```

GNU vs BSD make portability — use `gmake` explicitly on BSD/macOS, and avoid GNU extensions when targeting both:

```bash
# GNU-only
$(patsubst %.c,%.o,$(SRC))
$(shell ...)
$(call ...)
$(eval ...)
```

```bash
# Portable (POSIX)
$(SRC:.c=.o)        # works in both
.SUFFIXES: .c .o    # works in both
.c.o:               # works in both
```

If portability matters, declare `SHELL := /bin/sh`, avoid `:=`, `$(shell)`, `$(call)`, and document GNU 3.81 as the floor.

## Common .PHONY Catalog

```bash
.PHONY: all clean distclean install uninstall check test dist lint fmt help docs release

all: $(BIN)                           # default goal — build everything

clean:                                # remove build artifacts (keep config)
	rm -f *.o $(BIN)

distclean: clean                      # clean + remove generated config
	rm -f config.h Makefile.local

install: $(BIN)                       # install to PREFIX (default /usr/local)
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BIN) $(DESTDIR)$(PREFIX)/bin/

uninstall:                            # remove what install put down
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BIN)

check test: $(BIN)                    # run tests — `check` is autoconf canon
	./test.sh

dist:                                 # build a release tarball
	git archive --format=tar.gz --prefix=$(NAME)-$(VERSION)/ HEAD > $(NAME)-$(VERSION).tar.gz

lint:                                 # static analysis
	clang-tidy $(SRC) -- $(CFLAGS)

fmt:                                  # autoformat
	clang-format -i $(SRC)

docs:                                 # generate documentation
	doxygen Doxyfile

release: clean test dist              # full release pipeline
	@echo "Released $(VERSION)"
```

Standard `PREFIX`/`DESTDIR` convention is sacred — package managers depend on it:

```bash
PREFIX  ?= /usr/local
DESTDIR ?=                            # set by `make install DESTDIR=/tmp/pkg`
```

## The 'help' Target Idiom

Self-documenting Makefile — annotate targets with `## description`:

```bash
.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -o app .

test: ## Run all tests
	go test ./...

clean: ## Remove build artifacts
	rm -f app
```

```bash
$ make help
  build                Build the binary
  test                 Run all tests
  clean                Remove build artifacts
  help                 Show this help
```

`MAKEFILE_LIST` contains all parsed Makefiles, so this works through includes.

## Generating From Templates

Multi-line `define ... endef` plus `$(call ...)`:

```bash
define cc_rule
$(1).o: $(1).c
	$(CC) $(CFLAGS) -c -o $$@ $$<
endef

MODULES := main util parse

$(foreach m,$(MODULES),$(eval $(call cc_rule,$(m))))
```

This generates three rules at parse time. `$(eval ...)` re-parses its argument as Makefile syntax. `$$@` becomes `$@` after the first expansion.

## Build Matrix Patterns

Build the same source for multiple targets:

```bash
ARCHES := amd64 arm64
OSES   := linux darwin

define build_target
bin/$(2)/$(1)/app: $(SRC)
	GOOS=$(2) GOARCH=$(1) go build -o $$@ ./cmd/app
endef

$(foreach arch,$(ARCHES),$(foreach os,$(OSES),$(eval $(call build_target,$(arch),$(os)))))

ALL_BINS := $(foreach arch,$(ARCHES),$(foreach os,$(OSES),bin/$(os)/$(arch)/app))

.PHONY: all
all: $(ALL_BINS)
```

`make` then builds: `bin/linux/amd64/app`, `bin/linux/arm64/app`, `bin/darwin/amd64/app`, `bin/darwin/arm64/app`.

## Recursive Wildcard Pattern

`$(wildcard ...)` does NOT recurse into subdirectories. Define a helper:

```bash
rwildcard = $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))

ALL_C_FILES := $(call rwildcard,src/,*.c)
```

Or shell out:

```bash
ALL_C_FILES := $(shell find src -name '*.c' -type f)
```

The shell version is simpler and faster for large trees but breaks if filenames contain spaces.

## Common Error Messages

### `*** missing separator (did you mean TAB instead of 8 spaces?).  Stop.`

Cause: recipe line is indented with spaces instead of TAB.

```bash
# BROKEN
target:
    echo "spaces"
```

```bash
# FIXED — use a literal TAB
target:
	echo "tab"
```

Editor settings matter. In Vim: `:set noexpandtab`. In VS Code: file-association `makefile -> tab`.

### `*** No rule to make target 'foo', needed by 'bar'.  Stop.`

Cause: `bar` lists `foo` as a prerequisite, but no rule (explicit or implicit) tells Make how to build `foo`, and `foo` does not exist as a file.

```bash
# BROKEN
app: util.o     # but util.o has no rule and no util.c exists
	$(CC) -o $@ $^
```

```bash
# FIXED — provide source or a rule
util.o: util.c
	$(CC) -c -o $@ $<
```

Or check for typos: is `util.o` actually named `utils.o`?

### `*** [target] Error N`

Cause: the recipe ran a command that exited with status N. The number is the exit code.

```bash
target:
	exit 1
# make: *** [Makefile:2: target] Error 1
```

Read the actual command output above the `Error N` line. Common causes:
- compiler error (Error 1)
- file not found (Error 127)
- permission denied (Error 13/126)

To investigate: `make -n target` (dry run), then run the command directly in your shell.

### `*** Recursive variable 'X' references itself (eventually).  Stop.`

Cause: a `=` assignment forms a cycle when expanded.

```bash
# BROKEN
CC = $(CC) -Wall
```

```bash
# FIXED — use :=
CC := gcc
CC := $(CC) -Wall
```

### `Makefile:N: warning: overriding recipe for target 'foo'`

Cause: two rules with the same target and recipe.

```bash
foo:
	echo a

foo:
	echo b
# Makefile:4: warning: overriding recipe for target 'foo'
```

Use `::` for double-colon if you want both, or merge.

### `*** mixed implicit and normal rules.  Stop.`

Cause: combining pattern and non-pattern targets in one rule.

```bash
# BROKEN
foo %.o: %.c
	...
```

Split into two rules.

### `make: Nothing to be done for 'target'.`

Not an error — informational. The target is up to date. Add `-B` to force, or `touch` the source.

### `Circular X <- Y dependency dropped.`

Cause: cycle in the dependency graph. Make breaks the cycle and warns.

```bash
a: b
b: c
c: a    # cycle
```

Refactor — usually a real bug.

## Common Gotchas

### TAB vs spaces

```bash
# BROKEN
target:
    echo "spaces — fails"
```

```bash
# FIXED
target:
	echo "tab — works"
```

### `=` vs `:=` (delayed expansion)

```bash
# BROKEN — re-runs git on every reference
SHA = $(shell git rev-parse HEAD)
log:
	@echo $(SHA) $(SHA) $(SHA)
```

```bash
# FIXED
SHA := $(shell git rev-parse HEAD)
```

### `$$` vs `$` in recipe

```bash
# BROKEN — Make expands $HOME first (probably empty)
target:
	echo $HOME
```

```bash
# FIXED — $$ becomes $ for the shell
target:
	echo $$HOME
```

### Rules failing silently

Default `/bin/sh -c` does NOT exit on first error. Multi-step recipes can fail invisibly:

```bash
# BROKEN — first command fails, second runs anyway
target:
	cd nonexistent; rm -rf /
```

```bash
# FIXED — chain with &&
target:
	cd src && rm -rf build
```

```bash
# FIXED — set -e in recipe
target:
	set -e; cd src; rm -rf build
```

```bash
# FIXED — .ONESHELL with -ec
.ONESHELL:
.SHELLFLAGS := -ec
target:
	cd src
	rm -rf build
```

### `&&` continuations and trailing whitespace

```bash
# BROKEN — backslash + trailing space comments out the continuation
target:
	cd src && \ 
	make
```

The trailing space after `\` makes the line continuation fail. Strip trailing whitespace.

### Whitespace eating in `\`

```bash
# Line continuation collapses leading whitespace of next line
target:
	echo a\
	echo b
# Recipe sees: echo a echo b (one shell command)
```

```bash
# To preserve newlines, use ; or && without \
target:
	echo a; echo b
```

### PHONY collision with directory

```bash
# BROKEN — if directory `build` exists, `make build` does nothing
build:
	go build -o app .
```

```bash
# FIXED
.PHONY: build
build:
	go build -o app .
```

### SHELL default is /bin/sh

```bash
# BROKEN — bashisms
target:
	if [[ "$$VAR" == "x" ]]; then echo yes; fi
# /bin/sh: 1: [[: not found
```

```bash
# FIXED
SHELL := /bin/bash
target:
	if [[ "$$VAR" == "x" ]]; then echo yes; fi
```

### Comments in recipes

```bash
# Make comment — fine
target:
	# This is a SHELL comment, runs `# ...` -> nothing
	echo "real work"
```

In a recipe, `#` is a shell comment, not a Make comment. To put a Make comment alongside a recipe, place it BEFORE the TAB:

```bash
target:
# Make comment about target — must NOT be TAB-indented
	echo "real work"
```

## Performance Tips

- Cache `$(shell ...)` into `:=` variables. Repeated `=` shell calls dominate parse time.
- Avoid recursive make where possible. Use `include` for sub-modules.
- Use grouped targets `target1 target2 &:` (GNU 4.3+) when one recipe produces multiple files.
- `make -j$(nproc)` — most builds scale near-linearly.
- Profile with `make --debug=v` (4.4+) or `make -d` for parsing diagnostics.
- For huge builds, `make -p -f /dev/null` shows just the built-in database (one-time cost analysis).
- Use auto-dep generation (`-MMD -MP`) — manual dependencies are slow to maintain and always out of sync.

## Idioms

Auto-create directories with order-only:

```bash
$(BUILDDIR):
	mkdir -p $@

$(BUILDDIR)/%.o: %.c | $(BUILDDIR)
	$(CC) $(CFLAGS) -c -o $@ $<
```

Out-of-tree builds via VPATH:

```bash
VPATH := src include
%.o: %.c
	$(CC) -Iinclude $(CFLAGS) -c -o $@ $<
```

Recursive clean (when source tree is messy):

```bash
.PHONY: clean
clean:
	find . -type f \( -name '*.o' -o -name '*.d' \) -delete
	rm -rf $(BUILDDIR)
```

Version stamping:

```bash
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS += -X main.version=$(VERSION)
```

Pinned dependency reload trigger:

```bash
go.mod-stamp: go.mod
	go mod download
	@touch $@

build: go.mod-stamp
	go build .
```

PHONY-first convention:

```bash
.PHONY: all
all: $(BINS)         # first non-special rule wins as default goal

# remaining rules below
```

## CMake / Meson Hint

GNU Make is fine for projects up to roughly 10k lines and a single platform. Beyond that, consider:

- **CMake** — generates Makefiles (or Ninja). Best for cross-platform C/C++ with complex link graphs, transitive deps, and IDE integration. Verbose DSL but ubiquitous.
- **Meson** — Python-like config, fast Ninja backend, opinionated defaults. Cleaner than CMake for new projects.
- **Ninja** — backend only, designed for speed. Almost no one writes `build.ninja` by hand; let CMake or Meson generate it.
- **Bazel/Buck/Pants** — monorepos with strict hermetic builds. Overkill for anything under a million lines.

If your Makefile crosses 1000 lines or you're doing per-target compile flags across 50+ files, you have outgrown plain Make.

## Make Versions and Compatibility

```bash
make --version
# GNU Make 4.4
# Built for x86_64-pc-linux-gnu
```

Version landmarks:

```bash
3.81  # macOS-shipped, lacks .ONESHELL, .RECIPEPREFIX, .SHELLSTATUS
3.82  # adds .RECIPEPREFIX
4.0   # .ONESHELL, --trace, !=, := alias ::=, GNU make jobserver improvements
4.2   # .SHELLSTATUS, $(file ...) function
4.3   # grouped targets &: , .EXTRA_PREREQS
4.4   # --debug=v, --shuffle, jobserver-style protocol overhaul
```

POSIX make is a strict subset: no `$(shell)`, no `$(call)`, no `:=` (POSIX 2024 added it), no `ifdef`, no `define`, no pattern rules (only suffix rules). Almost no real Makefile is POSIX-clean. GNU is the assumption.

macOS ships GNU Make 3.81 — install Homebrew's `make` (4.4) and use `gmake`, OR put the Homebrew `gnubin` first in `PATH`.

To detect GNU version inside a Makefile:

```bash
ifneq ($(filter 4.%,$(MAKE_VERSION)),)
	# GNU 4.x features available
	HAS_ONESHELL := 1
endif
```

## Tips

- Always set `.DELETE_ON_ERROR:` near the top — prevents corrupted partial outputs.
- Always declare `.PHONY` for non-file targets — avoids invisible breakage when filenames collide.
- Cache `$(shell ...)` with `:=` — never with `=`.
- Use `$(MAKE)` not `make` for recursive invocations.
- Use `-MMD -MP` for auto-deps in any C/C++ project beyond toy size.
- Order-only prereqs (`|`) are the right tool for "directory must exist."
- `make -p` dumps the full database — best debugging tool for "why is this rule firing?"
- `$(info ...)` is the printf of Make. Sprinkle liberally while debugging.
- Quote shell variables: `"$$FOO"` not `$$FOO`.
- Use `set -e` (or `.ONESHELL` + `-ec`) in any multi-line recipe doing real work.
- Prefer pattern rules over suffix rules in new code.
- Standardize on `PREFIX`/`DESTDIR` for `install` so packagers do not curse you.
- Keep targets idempotent — running `make` twice should be a no-op.

## See Also

- c, rust, go, python, bash, regex, polyglot

## References

- [GNU Make Manual](https://www.gnu.org/software/make/manual/) -- canonical reference for GNU Make 4.x
- [GNU Make 4.4 Release Notes](https://lists.gnu.org/archive/html/info-gnu/2022-10/msg00008.html) -- recent changes
- [GNU Make Quick Reference](https://www.gnu.org/software/make/manual/html_node/Quick-Reference.html) -- one-page directive/function index
- [GNU Make Automatic Variables](https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html) -- $@, $<, $^, $?, $*, etc.
- [GNU Make Functions](https://www.gnu.org/software/make/manual/html_node/Functions.html) -- $(patsubst), $(wildcard), $(shell), etc.
- [GNU Make Special Targets](https://www.gnu.org/software/make/manual/html_node/Special-Targets.html) -- .PHONY, .DELETE_ON_ERROR, .ONESHELL, etc.
- [Mr. Bookies' Makefile Tutorial](https://makefiletutorial.com/) -- concise pragmatic tutorial
- [mrbook.org Make tutorial](http://mrbook.org/blog/tutorials/make/) -- classic intro
- [POSIX make Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/make.html) -- portable subset
- [Recursive Make Considered Harmful (Peter Miller, 1998)](https://aegis.sourceforge.net/auug97.pdf) -- the classic critique
- [Managing Projects with GNU Make (Mecklenburg)](https://www.oreilly.com/library/view/managing-projects-with/0596006101/) -- O'Reilly book
- [BSD make (bmake)](https://www.crufty.net/help/sjg/bmake.html) -- NetBSD make, used on FreeBSD
- [man make(1)](https://man7.org/linux/man-pages/man1/make.1.html) -- system manual
- [Remake](http://bashdb.sourceforge.net/remake/) -- GNU Make with debugger and improved errors
