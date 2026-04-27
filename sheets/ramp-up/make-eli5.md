# Make — ELI5

> Make is a smart recipe-runner. You tell it every dish you can cook and what ingredients each dish needs. When you ask for a dish, make only does the cooking that is actually out of date.

## Prerequisites

You should already feel comfortable in a terminal. If you don't, read `cs ramp-up bash-eli5` first. That sheet teaches you what `$` means, how to type commands, what files and folders look like, and how to move around. Come back here once you can list files, read files, and run a program.

You will get the most out of this sheet if you have used `git` at least once (read `cs ramp-up git-eli5` if not), because we are going to talk about how make and git both deal with files changing over time. They are different tools but they share the idea of "look at what changed."

You do **not** need to know C, C++, Go, Rust, or any compiled language. We will use tiny examples. The whole point of make is that it works no matter what is on the recipe card. The recipe could be "compile this C file" or "convert this picture" or "render this Markdown to HTML" or "run these tests." Make does not care. Make only cares about timestamps.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is Make

### Imagine your kitchen has a really lazy cook

Picture a chef in a small restaurant. The chef is great at cooking but really, really lazy. The chef hates wasted effort. If the chef can avoid cooking something, the chef will avoid cooking it.

You are the customer. You walk in and say, "I want a cake." The chef is too lazy to just start baking. The chef looks around first. The chef checks: is there already a cake on the counter? If yes, the chef says, "Here, take the cake that is already there." Done. No work.

But you are picky. You say, "Wait — when did you bake that cake?" Maybe the cake is two days old. Maybe the eggs the chef would use today are fresher than the cake on the counter. So the chef does another check: "When did I bake the cake on the counter? When did I last get fresh eggs? Are the eggs newer than the cake?" If the eggs are newer than the cake, the cake on the counter is **stale**. The chef has to bake a fresh cake using the new eggs.

The chef checks every ingredient one by one. Eggs newer than cake? Stale. Flour newer than cake? Stale. Sugar newer than cake? Stale. If even one ingredient is newer than the cake, the cake on the counter is stale and the chef has to start over.

That is exactly what make does. The cake is the file you want. The ingredients are the files it depends on. The "newer than" check is a comparison of file timestamps. Make is the lazy chef.

**That lazy chef is `make`.**

The recipe book the chef reads from is called the **Makefile**. The dishes in the book are called **targets**. The ingredients each dish needs are called **prerequisites**. The cooking steps for each dish are called the **recipe**. We will use those four words a thousand times in this sheet.

### Imagine your homework folder

Here is another way to think about it. Pretend you are a kid with a folder of school assignments. Every assignment has a final paper that you have to turn in. To make the final paper, you have to do some research, write a draft, get feedback, and then print the paper. Each of those steps is a separate file: `research.txt`, `draft.txt`, `feedback.txt`, `final.pdf`.

Now imagine you have a list pinned to the fridge that says:

```
final.pdf needs draft.txt and feedback.txt
draft.txt needs research.txt
research.txt needs nothing
```

When the teacher says, "Hand in the final paper," you don't sit down and redo every single step. You look at the list. You ask: "Is `final.pdf` already there? Is it newer than `draft.txt` and `feedback.txt`?" If yes, you just hand in `final.pdf`. If no, you redo only the steps you need.

That is the make idea. A file you want, a list of files it depends on, and a single rule: only redo work if some ingredient is newer than the result. Make turns this kind of "what depends on what" thinking into something a computer does for you automatically.

### Why does this even exist?

In 1976, a programmer named **Stuart Feldman** at Bell Labs got tired of accidentally compiling the wrong files. Back then, building a program meant typing a bunch of commands by hand. If you forgot one, your program had old code mixed with new code, and weird bugs would show up. Stuart wrote a tool that read a file (the Makefile) listing every output file, every input file, and the command to make the output from the inputs. The tool would figure out which outputs were stale and run only the commands that were actually needed.

That tool was called `make`. It is now **fifty years old**. Every version of every operating system in the world ships with some version of `make`. The Linux kernel is built with `make`. Most of GNU is built with `make`. The Go compiler used `make` for years. Docker images are built by something `make`-flavored. Almost every C program in existence was at some point built with a Makefile.

Why has it lasted so long? Because the idea is so simple. **Don't redo work that is already done.** That idea applies to compiling code, but it also applies to rendering Markdown, building containers, running tests, generating documentation, processing data, training models, and ten thousand other things. Anywhere you have a "this depends on that" relationship, `make` can save you time.

### The one trick make does

Make has exactly one job. Here it is in one sentence:

> Given a target, look at every prerequisite. If the target file does not exist, run the recipe. If any prerequisite is newer than the target file, run the recipe. Otherwise, do nothing.

That's it. Everything else in this sheet is a variation on that idea. Variables make recipes shorter. Patterns let one rule cover many files. Functions let you compute lists. Phony targets let you have "verbs" like `clean` and `install` that don't correspond to real files. But the core idea is always the same: **look at timestamps and only do what's stale.**

### Imagine a giant family tree of files

Here is the picture you should have in your head:

```
                        +------------+
                        |  program   |   <- target you asked for
                        +-----+------+
                              |
              +---------------+----------------+
              |               |                |
              v               v                v
           main.o         parser.o          util.o     <- intermediate targets
              |               |                |
              v               v                v
           main.c         parser.c          util.c     <- source files (leaves)
              |               |                |
              v               v                v
           parser.h       parser.h          util.h     <- shared headers
```

That is a **dependency graph**. The thing at the top is what you want. The leaves at the bottom are what you actually have on disk. Make walks this tree, looking at every node, and decides which branches need rebuilding. If you change `parser.h`, then `main.o`, `parser.o`, and `util.o` are all stale because each of them depends (somehow) on `parser.h`. Once those `.o` files are rebuilt, the final program is also stale (because its prerequisites changed), so make rebuilds the program too.

If you change only `util.c`, only `util.o` and the final program are rebuilt. `main.o` and `parser.o` are untouched. That is the magic of make: it does the **minimum work needed** to bring everything up to date.

### Why so many pictures?

You might wonder why we have a chef picture, a homework picture, and a family-tree picture. The answer is: nobody can see make working. Make is invisible. So we have to use pictures to imagine what it is doing. Different pictures help with different ideas.

The **chef** picture is best for understanding "stale" and "fresh."

The **homework** picture is best for understanding "this depends on that."

The **family tree** picture is best for understanding the whole graph at once.

If one picture is not clicking, switch to another. Whichever one feels right is the one you should keep in your head.

## The Mental Model: Targets, Prerequisites, Recipes

A Makefile is just a list of **rules**. A rule looks like this:

```makefile
target: prerequisite1 prerequisite2
	recipe-line-1
	recipe-line-2
```

Three parts:

- **Target** — the file (or "verb") you want to make.
- **Prerequisites** — the files the target depends on. If any of them changes, the target is stale.
- **Recipe** — the shell commands that turn the prerequisites into the target. Each recipe line **must start with a tab character.** Not eight spaces. Not four spaces. A tab. We will scream about this many times because it is the single biggest source of make errors.

Read it as English: "To make the target, you need these prerequisites, and here is how you do it."

Example:

```makefile
hello: hello.c
	cc -o hello hello.c
```

That says: "To make the file `hello`, I need `hello.c`. Here is how: run `cc -o hello hello.c`." When you type `make hello`, make checks: does `hello` exist? Is `hello` newer than `hello.c`? If `hello` is missing or older than `hello.c`, run the recipe.

A Makefile can have as many rules as you want. The first rule listed is the **default goal** — the one that runs if you type just `make` with no arguments.

```makefile
all: program

program: main.o util.o
	cc -o program main.o util.o

main.o: main.c
	cc -c main.c

util.o: util.c
	cc -c util.c
```

Now `make` (with no argument) builds `all`, which depends on `program`, which depends on `main.o` and `util.o`, which each depend on a `.c` file. Make figures out the right order automatically.

## The Stale-File Logic

Make decides "is this stale" by comparing **modification times** (sometimes called **mtime**). Every file on a Unix-like system has three timestamps: when it was last accessed (atime), when its contents were last modified (mtime), and when its metadata was last changed (ctime). Make only cares about mtime.

The rule is:

```
if (target does not exist) -> stale, rebuild
if (any prerequisite has mtime newer than target's mtime) -> stale, rebuild
otherwise -> up to date, do nothing
```

A picture of the comparison:

```
target:        2026-04-27 14:00:00   (cake on counter, baked at 2pm)
prereq A:      2026-04-27 13:30:00   (eggs delivered at 1:30pm)  -> older, fine
prereq B:      2026-04-27 14:30:00   (flour delivered at 2:30pm) -> NEWER, stale!

Result: rebuild the target.
```

It does not care **how much** newer. One nanosecond newer is enough. That can lead to weird situations where a tool changes a file's content without changing its mtime (because it wrote the same bytes back) and make doesn't notice. We will mention this again under "common confusions."

The mtime comparison is also why **`touch`** is a useful command with make. `touch foo.c` updates `foo.c`'s mtime to right now without changing its contents. That fakes a "this changed" signal and forces make to rebuild anything that depends on `foo.c`. Many real Makefiles use `touch` as a sentinel trick: a file whose only purpose is to be the timestamp of a thing that has no real output file.

## A Hello-World Makefile

Let's build the smallest useful Makefile. Make a folder, put a tiny C program in it:

```bash
$ mkdir hello
$ cd hello
$ cat > hello.c <<'EOF'
#include <stdio.h>
int main(void) { puts("hello, make"); return 0; }
EOF
```

Now write the Makefile:

```makefile
hello: hello.c
	cc -o hello hello.c
```

Save that as `Makefile` in the same folder. Make sure the indentation in front of `cc -o hello hello.c` is **a real tab character**, not spaces.

Run it:

```bash
$ make
cc -o hello hello.c
$ ./hello
hello, make
```

Run it again right away:

```bash
$ make
make: 'hello' is up to date.
```

Make is being lazy. It saw that `hello` exists and is newer than `hello.c`, so it did nothing. Now change `hello.c`:

```bash
$ touch hello.c
$ make
cc -o hello hello.c
```

You forced `hello.c`'s mtime to be newer than `hello`, so make ran the recipe again. That is the entire idea of make in one experiment.

## Variables: $(VAR), := vs = vs ?= vs +=

Repeating yourself is annoying. Make has variables.

```makefile
CC = cc
CFLAGS = -O2 -Wall
SRC = hello.c
OUT = hello

$(OUT): $(SRC)
	$(CC) $(CFLAGS) -o $(OUT) $(SRC)
```

A variable is set with `=` and used with `$(NAME)` or `${NAME}`. Either spelling works; pick one and be consistent.

There are **four ways** to assign a variable, and they have different timing semantics. This trips up beginners constantly.

### `=` (recursive / lazy)

```makefile
A = hello
B = $(A) world
A = goodbye
```

What is `B` now? `$(B)` expands to `$(A) world`, and **`$(A)` is re-evaluated every time you use `B`**. So `B` is `goodbye world`, not `hello world`. The right-hand side of a `=` assignment is **re-expanded every time the variable is used**. Lazy. Late-binding.

### `:=` (simple / immediate)

```makefile
A := hello
B := $(A) world
A := goodbye
```

Now `$(A)` is **expanded right at the moment of assignment**. `B` is set to `hello world` and stays `hello world` forever, no matter what happens to `A` later. Eager. Early-binding.

Use `:=` whenever you can. It is faster (the right-hand side is expanded once instead of every time) and it avoids confusing late-binding bugs. Use `=` only when you specifically need lazy expansion (rare).

### `?=` (only if not already set)

```makefile
CC ?= cc
```

Means: "If `CC` is not already defined, set it to `cc`." Otherwise leave it alone. This is how you let users override variables from the environment or command line. If they typed `make CC=clang`, your `CC ?= cc` does nothing. If they didn't, `CC` becomes `cc`.

### `+=` (append)

```makefile
CFLAGS = -O2
CFLAGS += -Wall
CFLAGS += -Wextra
```

Equivalent to `CFLAGS = -O2 -Wall -Wextra`. Useful for collecting flags from many places.

### Where do variables come from?

Variables can be set in five places, in order of priority (highest wins):

1. **Command line:** `make CC=clang` — overrides everything in the Makefile.
2. **Environment:** `export CC=clang; make` — set in the parent shell.
3. **Makefile itself:** `CC = cc`.
4. **Implicit defaults:** `CC` defaults to `cc` even if you never set it.
5. **`override` directive:** `override CC = gcc` forces it from the Makefile, beating the command line. Use sparingly.

## Automatic Variables

Every recipe gets a handful of magic variables that make sets for you. They have weird single-character names. Memorize them. They are everywhere.

| Variable | Meaning |
|----------|---------|
| `$@` | The target name. |
| `$<` | The first prerequisite. |
| `$^` | All prerequisites, space-separated, duplicates removed. |
| `$+` | All prerequisites, space-separated, duplicates kept. |
| `$?` | All prerequisites that are newer than the target. |
| `$*` | The "stem" — the part of a pattern rule that matched `%`. |
| `$%` | The archive member name (only useful with `.a` archives). |
| `$|` | The order-only prerequisites. |

Examples:

```makefile
hello: hello.c utils.c
	cc -o $@ $^
# expands to: cc -o hello hello.c utils.c

%.o: %.c
	cc -c $< -o $@
# for a target foo.o made from foo.c:
# expands to: cc -c foo.c -o foo.o
```

`$<` is "the first ingredient." `$@` is "what we're making." `$^` is "all ingredients."

There are also `D` and `F` suffixes that grab the directory or filename part:

- `$(@D)` — directory of the target.
- `$(@F)` — filename of the target.
- `$(<D)` — directory of the first prereq.
- `$(<F)` — filename of the first prereq.

Useful when targets live in subdirs.

## Functions

Make has built-in functions that operate on strings and lists. They are written `$(funcname args)`. Make is not a real programming language, but with functions you can do most simple computation you need.

### `$(wildcard PATTERN)`

Returns the list of files matching a glob pattern.

```makefile
SRCS := $(wildcard *.c)
# SRCS = "main.c parser.c util.c" if those exist
```

### `$(patsubst PATTERN,REPLACEMENT,TEXT)`

Pattern substitution. Replace one pattern with another in each word.

```makefile
OBJS := $(patsubst %.c,%.o,$(SRCS))
# main.c parser.c util.c -> main.o parser.o util.o
```

There is also a shorthand: `$(SRCS:.c=.o)` does the same thing.

### `$(foreach VAR,LIST,BODY)`

Loop over a list, expanding `BODY` once per item with `VAR` set to the current item.

```makefile
DIRS := src lib bin
ALL_SRCS := $(foreach d,$(DIRS),$(wildcard $(d)/*.c))
```

### `$(shell COMMAND)`

Runs a shell command and substitutes the output. Use sparingly — every `$(shell ...)` slows down Makefile parsing.

```makefile
GIT_SHA := $(shell git rev-parse --short HEAD)
DATE := $(shell date +%Y-%m-%d)
```

### `$(if CONDITION,THEN,ELSE)`

Returns `THEN` if `CONDITION` expands to non-empty, else `ELSE`.

```makefile
VERBOSE := $(if $(V),--verbose,)
# if V is set, VERBOSE = --verbose, else empty
```

### `$(or A,B,C)` and `$(and A,B,C)`

`or` returns the first non-empty argument; `and` returns the last argument if all are non-empty, else empty.

### `$(call FUNC,ARG1,ARG2,...)`

Invokes a user-defined function. You define the function with `=` and reference its arguments as `$(1)`, `$(2)`, etc.

```makefile
greet = hello, $(1)!
$(info $(call greet,world))
# prints: hello, world!
```

### Other useful functions

- `$(filter PATTERN,LIST)` — keep only words matching the pattern.
- `$(filter-out PATTERN,LIST)` — drop words matching the pattern.
- `$(sort LIST)` — sort and de-duplicate.
- `$(strip STR)` — remove leading/trailing whitespace.
- `$(subst FROM,TO,STR)` — plain text substitution.
- `$(words LIST)` — count words.
- `$(word N,LIST)` — Nth word.
- `$(firstword LIST)`, `$(lastword LIST)` — first/last word.
- `$(dir NAMES)`, `$(notdir NAMES)` — directory part / filename part.
- `$(basename NAMES)` — name without final suffix.
- `$(suffix NAMES)` — final suffix.
- `$(addprefix PRE,LIST)`, `$(addsuffix SUF,LIST)` — prepend/append to each word.
- `$(realpath NAMES)`, `$(abspath NAMES)` — resolve to absolute path.
- `$(error MESSAGE)` — abort with an error.
- `$(warning MESSAGE)` — print a warning, keep going.
- `$(info MESSAGE)` — print to stdout, keep going.
- `$(eval STRING)` — re-parse `STRING` as Makefile syntax. Powerful and dangerous.
- `$(value VAR)` — the unexpanded text of `VAR`.
- `$(origin VAR)` — where `VAR` was defined (file, environment, command line, etc.).
- `$(flavor VAR)` — `simple`, `recursive`, or `undefined`.
- `$(file >>X,content)` — write content to file X (GNU make 4.0+).

## Pattern Rules

A **pattern rule** lets one rule handle a whole class of files. Use `%` as a wildcard.

```makefile
%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@
```

That says: "For any file `foo.o`, if `foo.c` exists, here is how to make `foo.o` from `foo.c`." The `%` is called the **stem**. In this case the stem is `foo` for the target `foo.o`. You can refer to the stem inside the recipe with `$*`.

You can have multiple `%`s in a target as long as they all match the same stem:

```makefile
build/%.o: src/%.c
	$(CC) -c $< -o $@
```

Pattern rules let you write a Makefile for ten thousand C files in three lines.

## Implicit Rules (the built-in catalog)

GNU make ships with a built-in catalog of implicit rules. If you don't write your own pattern rules, make tries to apply these. The most important built-in rules are:

- `%.o: %.c` — compile a C file with `$(CC) $(CPPFLAGS) $(CFLAGS) -c`.
- `%.o: %.cc` and `%.o: %.cpp` — compile C++ with `$(CXX) $(CPPFLAGS) $(CXXFLAGS) -c`.
- `%.o: %.s` — assemble with `$(AS) $(ASFLAGS)`.
- `(%): %` — add object to archive with `$(AR) $(ARFLAGS) $@ $%`.
- A linker rule for making an executable from a single `.o` file.

This is why a one-line Makefile like:

```makefile
hello: hello.o
```

actually works — make uses its built-in `%.o: %.c` rule to make `hello.o` from `hello.c`, and then a built-in link rule to make `hello` from `hello.o`. You don't have to write either rule yourself.

You can see every implicit rule make knows by running:

```bash
$ make -p -f /dev/null | less
```

Set `-p` ("print database") and `-f /dev/null` (read no Makefile) and make will dump every built-in rule, every default variable, every built-in function. It is a lot. It is a great way to learn make.

To turn off implicit rules entirely (sometimes useful when debugging), use `make -r`.

## Phony Targets

A **phony target** is a target that does not correspond to a real file. It is just a name for a sequence of commands.

```makefile
.PHONY: clean install all test

clean:
	rm -rf build *.o

install:
	cp hello /usr/local/bin/

test:
	./run-tests.sh
```

By declaring these targets `.PHONY`, you tell make: "Don't ever look for a file called `clean`. Just always run the recipe when somebody asks for it."

If you forget `.PHONY` and someone happens to create a file called `clean` in your directory, your `make clean` rule will silently stop working because make will think the file `clean` is up to date. **Always mark your verbs phony.**

The conventional set of phony targets every project has:

- `all` — build everything (default goal).
- `clean` — delete build artifacts.
- `install` — install the built program.
- `uninstall` — remove the installed program.
- `test` or `check` — run the test suite.
- `dist` — make a release tarball.
- `help` — print available targets.

## Suffix Rules (legacy)

Before pattern rules existed, there were **suffix rules**. They look like:

```makefile
.SUFFIXES: .c .o

.c.o:
	$(CC) -c $<
```

`.c.o:` means "to make a `.o` from a `.c`." It is the old-school version of `%.o: %.c`. You will see this in old Makefiles and in BSD make. Modern GNU make supports both, but pattern rules are clearer and more flexible. Use pattern rules in new code.

## Multiple Outputs (grouped targets, GNU make 4.3+)

Sometimes a single command produces several output files at once. For example, a parser generator might emit both `parser.c` and `parser.h` in one invocation. Until GNU make 4.3, this was awkward (you had to use stamp files or awkward tricks). GNU make 4.3 added the **grouped target** syntax, written with `&:`:

```makefile
parser.c parser.h &: parser.y
	bison -o parser.c -d parser.y
```

The `&:` says: "These targets are produced **together** by one invocation of the recipe. Don't run the recipe once per target." Without `&:`, make would think running the recipe produced only the first listed target and would run the recipe again for the second.

Older Makefiles emulate this with a stamp file:

```makefile
parser.c parser.h: parser.stamp
parser.stamp: parser.y
	bison -o parser.c -d parser.y
	touch parser.stamp
```

GNU make 4.4 added `.NOTINTERMEDIATE` to keep stamp/grouped files from being auto-deleted as intermediates.

## Order-Only Prerequisites

Sometimes you want a prerequisite that must exist before the target is built, but whose mtime should not trigger a rebuild. Classic example: an output directory.

```makefile
build/hello: hello.c | build
	cc -o $@ $<

build:
	mkdir -p build
```

Everything after the `|` is **order-only**. The `build` directory must exist before `build/hello` is built, but if `build`'s mtime changes (which happens whenever you create files in it!), it will not cause `build/hello` to rebuild. Without the `|`, every time you put a new file in `build/`, make would rebuild `build/hello` because the directory's mtime got newer. With the `|`, that doesn't happen.

Order-only prereqs are the cleanest way to handle "make sure this directory exists" without breaking incremental builds.

## Recursive Make Considered Harmful

In big projects, people are tempted to give every subdirectory its own Makefile and call them recursively from a top-level Makefile, like:

```makefile
all:
	cd src && $(MAKE)
	cd lib && $(MAKE)
	cd tests && $(MAKE)
```

In 1998, Peter Miller wrote a famous paper called **"Recursive Make Considered Harmful."** The argument: when you split your build into multiple Makefile invocations, each child make has only a partial view of the dependency graph. It does not know about files in sibling directories. So it cannot detect when a header in `lib/` changes that affects something in `src/`. You either get incorrect builds (stale outputs) or you have to over-rebuild to be safe (slow). Either way, parallel `-j` is bottlenecked because each child make finishes before the next one starts.

The fix: **one Makefile that knows about everything.** Use `include` to pull in fragments from each subdirectory. The top-level make sees the whole graph and can parallelize correctly.

That said, recursive make is still everywhere because it's easy to write and easy to reason about per-directory. For small projects it's fine. For big projects where build times matter, look at single-Makefile patterns or move to a modern build system (CMake, Meson, Ninja, Bazel). The Linux kernel uses a famous variant called "Kbuild" that is recursive but heavily tuned to avoid the worst issues.

The original Miller paper is short and worth reading: search "recursive make considered harmful Miller" or check `cs build-systems make` for a longer summary.

## Single Big Makefile vs Many Small (the inclusion pattern)

The "single big Makefile" pattern looks like this:

```makefile
# top-level Makefile
include src/rules.mk
include lib/rules.mk
include tests/rules.mk

all: $(SRC_BIN) $(LIB_TARGETS)
```

Each `rules.mk` contributes variables and rules, but there is only one make process and one dependency graph. This is the modern way to handle big projects without recursing.

A typical layout:

```
project/
+-- Makefile          <- top level, includes everything
+-- src/
|   +-- rules.mk      <- "here are the source files in src/"
|   +-- main.c
+-- lib/
|   +-- rules.mk      <- "here are the library files"
|   +-- foo.c
+-- tests/
    +-- rules.mk
    +-- test_foo.c
```

The `rules.mk` files declare paths relative to the project root, not relative to themselves, so a single make from the top sees a consistent global picture.

## Conditional Directives

Make supports `if`/`else` directives that change which lines are read. They look like Bourne shell but are not.

```makefile
ifeq ($(OS),Linux)
    LIBS = -lpthread -ldl
endif

ifneq ($(DEBUG),)
    CFLAGS += -g -O0
else
    CFLAGS += -O2
endif

ifdef VERBOSE
    Q =
else
    Q = @
endif
```

- `ifeq (A,B)` — true if A and B are equal.
- `ifneq (A,B)` — true if A and B are not equal.
- `ifdef VAR` — true if `VAR` is defined (even if empty? subtle: true if `$(origin VAR)` is not `undefined`).
- `ifndef VAR` — opposite of `ifdef`.

These are processed at parse time, before any rule runs. They affect which lines are part of the Makefile, not which recipes execute.

## Include Directive

You can split your Makefile into pieces and pull them together with `include`:

```makefile
include common.mk
include $(wildcard *.d)
-include optional.mk
```

- `include` — include the named file; error if it doesn't exist.
- `-include` (or `sinclude`) — include the file if it exists; silently skip if not.

The `-include $(wildcard *.d)` line is a famous idiom: include any auto-generated dependency files, but don't fail on the first build when none exist yet. We will see how `.d` files are generated below.

## Self-Documenting Makefile (help target)

A great convention: write a `help` target that prints every documented target.

```makefile
.PHONY: help
help:  ## Print this help.
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the binary.
	go build .

test:  ## Run tests.
	go test ./...

clean: ## Remove build artifacts.
	rm -f myapp
```

Run `make help` and you get:

```
  help                Print this help.
  build               Build the binary.
  test                Run tests.
  clean               Remove build artifacts.
```

The trick: the `awk` script reads the Makefile itself (via `$(MAKEFILE_LIST)`), grabs every line that starts with a target name followed by `## `, and pretty-prints them. Free documentation that can never go out of sync.

## Common Patterns

### Auto-generated header dependencies (C/C++)

This is the killer pattern for serious C/C++ Makefiles. The compiler can emit a `.d` file describing every header your `.c` file actually included, and make can include those `.d` files to track header dependencies automatically.

```makefile
CC = cc
CFLAGS = -O2 -Wall -MMD -MP
SRCS := $(wildcard *.c)
OBJS := $(SRCS:.c=.o)
DEPS := $(SRCS:.c=.d)

myapp: $(OBJS)
	$(CC) -o $@ $^

%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

-include $(DEPS)

clean:
	rm -f myapp $(OBJS) $(DEPS)
```

The flags do the heavy lifting:

- `-MMD` — emit a `.d` file alongside the `.o` listing every header included (skipping system headers).
- `-MP` — also emit phony targets for each header. This prevents make errors if a header is later renamed or deleted.

The `-include $(DEPS)` pulls in all the `.d` files. After the first build, make knows that `main.o` depends on `main.c` and every header `main.c` included transitively. Edit a header and only the affected `.o` files rebuild.

### Hash-based out-of-date detection

Pure mtime comparison breaks if a tool rewrites a file with the same content (mtime advances, content unchanged) or if you copy/move files in ways that scramble timestamps. Modern build systems like Bazel, Buck, ccache, and tup use **content hashes** instead of mtimes. Plain make does not, but you can fake it with a stamp file that's only updated when content actually changes:

```makefile
config.h: config.h.in config.stamp
config.stamp: config.h.in
	./regen-config && cmp -s config.h.tmp config.h || mv config.h.tmp config.h
	touch config.stamp
```

For really hash-based behavior, use ccache (caches compiler outputs by source hash) or move to a hash-based build system.

## GNU Make vs BSD Make vs POSIX Make

There are three main flavors of make in the wild:

- **GNU make** — the most popular; default on Linux. Many features documented in this sheet are GNU extensions: pattern rules with `%`, `:=`, `?=`, `+=`, all the function calls, conditional directives, `.PHONY`, grouped targets, and so on.
- **BSD make (`bmake`)** — default on FreeBSD, NetBSD, OpenBSD. Different syntax for some features. Notably uses `.if`/`.endif` instead of `ifeq`/`endif`. Has its own variable modifiers like `${VAR:S/x/y/}` for substitution.
- **POSIX make** — the standardized minimum. Just suffix rules, basic variables, and recursive expansion. Almost nobody writes only POSIX make today.

If you write `make`, you mean GNU make on Linux and macOS (where it's installed as `make`). On BSD systems, `make` is BSD make and GNU make is `gmake`.

To make your Makefile work everywhere, stick to features POSIX guarantees, or document that you require GNU make and put `MAKEFLAGS += --no-builtin-rules` and `SHELL := /bin/sh` at the top.

## Common Errors

Verbatim error messages you will see and exactly how to fix each one:

### `*** missing separator. Stop.`

You used **eight spaces** (or four, or any spaces) instead of a real tab character on a recipe line. Make is yelling because it expected a tab. Open the file in your editor and configure it to use real tabs in Makefiles. In Vim: `:set noexpandtab`. In VS Code: there's a "render whitespace" option that shows tabs vs spaces, and the bottom-right corner lets you switch. **The original sin of make.** It has annoyed people for fifty years.

### `*** No rule to make target 'X'. Stop.`

You asked make to build something, but no rule produces it and no file by that name exists. Either you typed the target name wrong, or you forgot to write a rule for it, or the file is missing. Check spelling. Check that the prerequisite paths in your rules are correct.

### `*** Recursive variable 'X' references itself (eventually). Stop.`

You have something like `A = $(A) extra`. Because `=` is recursive, expanding `A` requires expanding `A` again, which requires... infinite recursion. Use `:=` instead: `A := $(A) extra` or just `A += extra`.

### `warning: overriding recipe for target 'X'`

You defined two recipes for the same target. Make uses the **later** one. Usually this is a bug — you have two rules trying to build the same file. If you genuinely want both recipes to run, use `::` (double-colon rules) or combine them into one recipe.

### `warning: ignoring old recipe for target 'X'`

The flip side of the previous one. Make is telling you which recipe got overridden.

### `*** target file 'X' has both : and :: entries. Stop.`

You used `target:` somewhere and `target::` somewhere else for the same target. Pick one style and stick with it.

### `make: 'all' is up to date.`

Not an error! It is make telling you that the requested target's prerequisites are all older than the target, so there is nothing to do. If you really want to rebuild from scratch, use `make -B` (force) or `make clean all`.

### `*** No targets specified and no makefile found. Stop.`

You ran `make` in a directory with no Makefile, or with a file named something other than `Makefile`/`makefile`/`GNUmakefile`. Use `-f Filename` to specify, or rename your Makefile.

### `*** [Makefile:42: target] Error 1`

A recipe command exited with non-zero status (a shell error code). The number after `Error` is the exit code (1, 2, 127, etc). The number `42` is the line in the Makefile where the rule for `target` started. Look at the previous output to see which command failed and why.

### `cannot find input file: X`

Often comes from a tool inside a recipe (compiler, linker), not from make itself. Means a prerequisite path is wrong or a file genuinely missing.

## Hands-On

These commands are safe to run in a scratch directory. Some are read-only (`-n`, `-q`, `-p`); a few will actually run a build, so do them in a folder you don't mind making messy. Most assume GNU make. On BSD systems substitute `gmake` for `make`.

### Setup

```bash
$ mkdir /tmp/make-eli5
$ cd /tmp/make-eli5
$ cat > hello.c <<'EOF'
#include <stdio.h>
int main(void) { puts("hi"); return 0; }
EOF
$ cat > Makefile <<'EOF'
CC = cc
CFLAGS = -O2 -Wall

.PHONY: all clean install

all: hello

hello: hello.c
	$(CC) $(CFLAGS) -o $@ $<

install: hello
	@echo "would install hello to $(DESTDIR)/usr/local/bin"

clean:
	rm -f hello
EOF
```

Now run experiments. Use a real **tab** in front of the recipe lines if you re-type the Makefile by hand.

### Experiment 1: Run the default goal

```bash
$ make
cc -O2 -Wall -o hello hello.c
```

### Experiment 2: Run again — see laziness

```bash
$ make
make: Nothing to be done for 'all'.
```

Make sees `hello` is newer than `hello.c`, refuses to do work.

### Experiment 3: Force a rebuild without changes

```bash
$ make -B
cc -O2 -Wall -o hello hello.c
```

`-B` is "always rebuild." Useful when you suspect the build state is wrong.

### Experiment 4: Dry run — see what would happen

```bash
$ make -n
cc -O2 -Wall -o hello hello.c
```

`-n` (or `--dry-run`, or `--just-print`) shows the commands but does not run them.

### Experiment 5: Query if a target is up to date

```bash
$ make -q hello && echo "up to date" || echo "stale"
up to date
$ touch hello.c
$ make -q hello && echo "up to date" || echo "stale"
stale
```

`-q` exits 0 if up to date, 1 if not. Good for scripts.

### Experiment 6: Build a specific target

```bash
$ make clean
rm -f hello
$ make hello
cc -O2 -Wall -o hello hello.c
```

You can name the target you want. `make` alone runs the first non-prefixed target (the default goal).

### Experiment 7: Override a variable

```bash
$ make CC=clang
clang -O2 -Wall -o hello hello.c
```

### Experiment 8: Override with extra flags

```bash
$ make CFLAGS='-O0 -g' LDFLAGS=-flto
cc -O0 -g -o hello hello.c
```

Variables on the command line beat anything in the Makefile.

### Experiment 9: Parallel build (`-j`)

```bash
$ make -j8 -B
cc -O2 -Wall -o hello hello.c
```

`-j8` means "run up to 8 recipes in parallel." On a real project with many sources, this is a huge speedup.

### Experiment 10: Auto-detect parallelism

```bash
$ make -B --jobs=$(nproc)        # Linux
$ make -B --jobs=$(sysctl -n hw.ncpu)  # macOS / BSD
```

`nproc` prints the number of CPUs.

### Experiment 11: Keep going on errors

```bash
$ make -k
```

`-k` makes make continue building unrelated targets even after one fails. Useful for getting the longest list of errors.

### Experiment 12: Debug — see decisions

```bash
$ make -d 2>&1 | less
```

`-d` floods you with information about every rule make considers. Pipe through `less` because the output is huge.

### Experiment 13: Verbose dependency tracing

```bash
$ make --debug=v
```

Shows which targets are considered, which prereqs are checked, why decisions were made. Less noisy than `-d`.

### Experiment 14: Job tracing (parallel debug)

```bash
$ make --debug=j -j4
```

Shows which jobs were started and finished, useful for diagnosing parallel build issues.

### Experiment 15: Print the database

```bash
$ make -p | less
```

`-p` dumps every rule, every variable, every implicit rule make knows about. Educational.

### Experiment 16: Search the database

```bash
$ make -p -f /dev/null | grep '^%\.o:'
```

See every implicit rule that produces `.o` files.

### Experiment 17: Warn on undefined variables

```bash
$ make --warn-undefined-variables
```

Catches typos: if you write `$(CCC)` instead of `$(CC)`, this option warns instead of silently expanding to empty.

### Experiment 18: Quiet "entering" / "leaving" messages

```bash
$ make --no-print-directory
```

When make recurses, it prints `Entering directory '...'` and `Leaving directory '...'` lines. This option suppresses them. Cleaner output.

### Experiment 19: Run from another directory

```bash
$ make -C /tmp/make-eli5 hello
```

`-C` is "change to this directory before doing anything." Useful in scripts and CI.

### Experiment 20: Use a different Makefile name

```bash
$ make -f Makefile.alt all
```

`-f` specifies a different Makefile. By default make looks for `GNUmakefile`, then `makefile`, then `Makefile` (in that order).

### Experiment 21: Pass DESTDIR for staging install

```bash
$ make install DESTDIR=/tmp/staging
would install hello to /tmp/staging/usr/local/bin
```

`DESTDIR` is the conventional variable for "install to a staging area instead of the real system." Used heavily by package builders.

### Experiment 22: BSD make variable inspection

```bash
$ bmake -V CC          # BSD only
$ gmake -p | grep '^CC = '   # GNU equivalent
```

### Experiment 23: Time a build

```bash
$ time make -B
real    0m0.157s
user    0m0.083s
sys     0m0.044s
```

### Experiment 24: Tee output to a log file

```bash
$ make 2>&1 | tee build.log
```

Combines stdout and stderr into one stream and saves a copy to `build.log` while you watch.

### Experiment 25: Generate compile_commands.json

```bash
$ bear -- make -B
```

`bear` (Build EAR) intercepts the compiler invocations and writes `compile_commands.json`, which is what `clangd` uses for IDE features (jump to definition, find references, etc.). Install with `apt install bear` or `brew install bear`.

### Experiment 26: Inspect MAKEFLAGS

```bash
$ MAKEFLAGS=-j4 make -p | grep MAKEFLAGS
MAKEFLAGS = -j4
```

`MAKEFLAGS` is how make passes flags to recursive sub-makes. Setting it in the environment is the way to globally enable, say, parallelism.

### Experiment 27: Run make's help target (if defined)

```bash
$ make help
```

Many projects define a `help` target. If you don't know what targets exist, try this first.

### Experiment 28: List every target

```bash
$ make -pn | grep -E '^[a-zA-Z0-9_-]+:' | grep -v -E '^(Makefile|.PHONY)' | sort -u
```

Crude but works. There are nicer scripts that read `MAKEFILE_LIST` from inside the Makefile.

### Experiment 29: Run a specific recipe in dry-run with full expansion

```bash
$ make -n --debug=v hello | less
```

Useful when you want to see exactly what command would run with all variable expansions resolved.

### Experiment 30: GNU make 4.4 — randomize execution

```bash
$ make --shuffle -B
```

`--shuffle` (added in GNU make 4.4) randomizes the order of independent build steps each run. Stress-tests your dependency declarations: if a build fails with `--shuffle` but passes without, you have missing prereqs.

### Experiment 31: See how long make takes to parse the Makefile

```bash
$ time make -p > /dev/null
```

If parsing your Makefile is slow, this tells you. Often `$(shell ...)` or `$(wildcard ...)` calls are the culprits.

### Experiment 32: Compare with ninja

If you have a ninja-based project (cmake -G Ninja, meson, etc.):

```bash
$ ninja -t commands       # show every command
$ ninja -t graph          # output dependency graph as graphviz
$ ninja -d explain        # explain why each target is being rebuilt
```

ninja was designed by ex-Chrome engineers as a faster make replacement. Same ideas, leaner format. If your make builds feel slow, this is the first alternative to try.

### Experiment 33: Look at MAKELEVEL during a recursive build

Add this to your Makefile temporarily:

```makefile
$(info MAKELEVEL = $(MAKELEVEL))
```

`MAKELEVEL` is 0 for the top-level make, 1 for a sub-make spawned by `$(MAKE)`, 2 for a sub-sub-make, and so on. Useful for debugging nested builds.

### Experiment 34: See which Makefile defined a variable

```makefile
$(info origin of CC = $(origin CC))
```

`$(origin VAR)` returns one of: `undefined`, `default`, `environment`, `environment override`, `file`, `command line`, `override`, `automatic`. Helpful when something has a value you didn't expect.

## Common Confusions

These are the things that bite people again and again. Each one is a misunderstanding the Makefile gods will absolutely punish you for.

### Tabs vs spaces — the original sin

The single biggest source of make errors. Recipe lines must start with a tab character. Not eight spaces. Not four. **A tab.** Modern editors usually default to "convert tabs to spaces," which silently breaks every Makefile you write. Configure your editor: in Vim, `autocmd FileType make setlocal noexpandtab`. In VS Code, the file-type detection should turn off "insert spaces" automatically; if it doesn't, fix the workspace settings. The error is `*** missing separator. Stop.`

GNU make 3.82 added `.RECIPEPREFIX := >` (or any single character) to let you replace tab with another character, but almost nobody uses it. Just use tabs.

### `:=` runs immediately, `=` runs lazily

```makefile
A := $(shell date)   # captures the time at parse, never changes
B = $(shell date)    # runs `date` every single time $(B) is expanded
```

If a heavy `$(shell ...)` ends up in a recursive variable, your build can become very slow because the shell runs on every reference. Use `:=` unless you specifically need late binding.

### `.PHONY` for "always run" targets

If you have a target `clean` and somebody creates a file called `clean` in your directory, your `make clean` rule silently stops working: make sees the file `clean` exists and is newer than nothing, so it's "up to date." Mark every verb target `.PHONY` and you avoid this trap forever.

### `make clean` can fail if `clean` is also a real file

Same idea as above. Phony targets are a *promise* to make that the target name is not a real file. If the file exists and the target isn't phony, weird stuff happens.

### Parallel make can race on shared `mkdir`

```makefile
build/main.o: main.c
	mkdir -p build
	cc -c main.c -o build/main.o

build/util.o: util.c
	mkdir -p build
	cc -c util.c -o build/util.o
```

Run with `-j8` and two recipes can race on the `mkdir`. The fix: use an order-only prerequisite to ensure the directory exists before either recipe runs.

```makefile
build/%.o: %.c | build
	cc -c $< -o $@

build:
	mkdir -p build
```

### Each recipe line runs in a fresh shell

```makefile
mytarget:
	cd /tmp
	pwd        # this is NOT in /tmp! It's wherever make was started.
```

Each line in a recipe is a separate `sh -c '...'` invocation. The `cd` happens in one shell that exits immediately; the next line is a brand-new shell starting from the current directory. Use backslash continuation:

```makefile
mytarget:
	cd /tmp && \
	pwd
```

Or use `.ONESHELL:` (GNU make 3.82+) to make the entire recipe run in one shell. Or use `;` chaining.

### Variables: command line beats env beats Makefile

```bash
$ export FOO=env
$ cat Makefile
FOO = makefile
print:
	@echo FOO=$(FOO)

$ make print              # prints "FOO=makefile" — Makefile wins by default
$ make FOO=cmd print      # prints "FOO=cmd" — command line beats Makefile
$ make -e print           # prints "FOO=env" — `-e` makes env beat Makefile
```

`?=` lets the Makefile yield to the environment without `-e`:

```makefile
FOO ?= default
```

### `Nothing to be done for X`

This means: you asked for `X`, no recipe needs to run because everything is up to date. It's a normal "everything's fine" message, not an error.

### A target appears to never rebuild even after sources change

Common causes: (1) target's mtime is somehow ahead of the source — usually because the build machine's clock is off, or files were copied with `cp -p` preserving mtimes; (2) target is phony but not declared `.PHONY`; (3) you have a rule `target:` with no recipe and another `target: deps` with a recipe — the no-recipe one sometimes wins.

Debug with `make --debug=v` to see exactly why each decision was made.

### `$$` is a literal dollar sign; `$` is a variable expansion

In a recipe line, `$` is special to make. To pass a literal `$` to the shell, double it: `$$`.

```makefile
print-pwd:
	echo $$PWD              # passes "$PWD" to the shell, shell expands it
	echo $(MAKEFILE_LIST)   # make expands the variable

# WRONG:
# echo $PWD                # make sees "$P" as a variable, then "WD" literal
```

### When to use `$(eval ...)`

`$(eval ...)` re-parses a string as Makefile syntax. It is powerful and rarely needed. Reach for it only when you want to *generate* rules dynamically, e.g.:

```makefile
define ADD_TARGET
$(1)_OBJS := $$(patsubst %.c,%.o,$$(wildcard $(1)/*.c))
$(1): $$($(1)_OBJS)
endef

PROGS := foo bar baz
$(foreach p,$(PROGS),$(eval $(call ADD_TARGET,$(p))))
```

This generates three rules at once, one per program. Powerful, hard to debug. Avoid until you have no other option.

### GNU-specific extensions vs portable POSIX make

If you say `make`, you mean GNU make on Linux/macOS. If you ship code that needs to build on FreeBSD, OpenBSD, Solaris, AIX, etc., either restrict yourself to POSIX make features (suffix rules, basic variables) or document that you require GNU make.

A portable Makefile starts with something like:

```makefile
SHELL := /bin/sh
.SUFFIXES:
.SUFFIXES: .c .o
```

and avoids `:=`, pattern rules, all functions, and most automatic variables.

### Why does `make` invent files I never asked for?

GNU make's implicit rule machinery sometimes makes intermediate files (like `.o` from `.c` even when you didn't ask). Worse: it auto-deletes them after the build, which can be surprising. If you want to keep an intermediate, mark it `.SECONDARY:` (keep silently) or `.PRECIOUS:` (don't delete on error). To turn implicit rules off entirely, use `make -r` or `MAKEFLAGS += --no-builtin-rules`.

### Why does my Makefile work in one terminal but not another?

Different shells. `SHELL` defaults to `/bin/sh`, which is dash on Debian/Ubuntu, ash on Alpine, zsh on macOS terminal, bash on most other places. If you wrote a recipe with `bash`-only features (`[[`, arrays, `<()`), it works in some places and fails in others. Always set `SHELL := /bin/bash` (or `/usr/bin/env bash`) at the top of the Makefile if you depend on bash features.

### Quoting hell

```makefile
greet:
	echo "hello, $(USER)"
```

Make first expands `$(USER)` to your username, then hands the line to the shell, which sees `echo "hello, alice"`. If your username has spaces (rare on Linux, possible on macOS) or special characters, the quoting can break. When in doubt, single-quote at the shell level: `echo 'hello, alice'`.

### Order matters with `?=` and overrides

```makefile
CC ?= cc
override CC = gcc
```

The `override` directive forces `CC = gcc` even if the user passed `make CC=clang`. The `?=` would normally yield to the command-line value, but `override` wins. Use `override` only when you really need to.

## Vocabulary

Words and symbols you will see in Makefiles, build logs, and discussions about make. Learn the ones that come up in your projects first; the rest are useful when you read other people's Makefiles.

| Term | Plain English |
|------|---------------|
| make | The tool itself. Reads a Makefile and runs the right recipes. |
| GNU make | The most popular implementation; default on Linux. Has many extensions over POSIX. |
| gmake | The name `GNU make` is installed under on systems where `make` is something else (BSD, often macOS). |
| BSD make | The make that ships with FreeBSD, NetBSD, OpenBSD. Different syntax in places. |
| bmake | Portable BSD make, often available on Linux as `bmake`. |
| POSIX make | The standardized minimum every make is supposed to support. |
| Makefile | The file make reads. Default names: `GNUmakefile`, `makefile`, `Makefile`. |
| GNUmakefile | A Makefile that uses GNU extensions; if present, GNU make prefers it. |
| makefile | Lower-case alternative file name. |
| target | The thing you want to make. A file name (or a phony name). |
| prerequisite | An ingredient the target depends on. Also called a dependency. |
| dependency | Same as prerequisite. |
| recipe | The shell commands that build a target from its prerequisites. |
| rule | A target plus its prereqs and recipe. |
| pattern rule | A rule using `%` to match many files at once: `%.o: %.c`. |
| implicit rule | A built-in rule make uses when you didn't write your own. |
| suffix rule | The legacy form of pattern rule, like `.c.o:`. |
| double-colon rule | `target:: prereqs ; recipe`. Lets you have multiple independent recipes for the same target. |
| .PHONY | Tells make a target name is not a real file; always run its recipe when asked. |
| .SUFFIXES | Lets you declare which suffixes are recognized for suffix rules. |
| .DEFAULT | A rule used when no other rule applies to a target. |
| .DELETE_ON_ERROR | If set, make deletes the target file when its recipe fails. Stops half-built outputs. |
| .NOTPARALLEL | Marks the entire Makefile (or specific targets) as not parallelizable. |
| .ONESHELL | Causes the whole recipe to run in a single shell invocation. |
| .SECONDARY | Targets that should not be auto-deleted as intermediates. |
| .INTERMEDIATE | Explicitly marks intermediate files (auto-deleted by default). |
| .PRECIOUS | Targets that should not be deleted even if their recipe fails or is interrupted. |
| .EXPORT_ALL_VARIABLES | Export every variable to the environment of every recipe. |
| .DEFAULT_GOAL | The target make builds when you don't name one. Defaults to the first target listed. |
| .RECIPEPREFIX | Lets you change the recipe-line prefix from tab to another character. |
| .NOTINTERMEDIATE | (GNU make 4.4+) Prevents specific files from being treated as intermediates. |
| variable | A named string in make. Set with `=`, `:=`, `?=`, `+=`, `define`. |
| simple variable | One assigned with `:=`. RHS expanded immediately at definition time. |
| recursive variable | One assigned with `=`. RHS re-expanded on every use. |
| conditional variable | One assigned with `?=`. Only sets if not already defined. |
| append variable | One modified with `+=`. Appends to existing value with a space. |
| override variable | One declared with `override`, immune to command-line changes. |
| environment variable | A variable inherited from the calling shell. |
| override directive | `override VAR = value` forces the value in the Makefile. |
| define directive | Multi-line variable definition: `define NAME` ... `endef`. |
| automatic variable | A variable make sets per-rule: `$@`, `$<`, `$^`, etc. |
| `$@` | The current target. |
| `$%` | Archive member name (for `.a` files). |
| `$<` | The first prerequisite. |
| `$?` | All prerequisites that are out of date relative to the target. |
| `$^` | All prerequisites, no duplicates. |
| `$+` | All prerequisites, with duplicates kept. |
| `$*` | The pattern stem matched by `%`. |
| `$|` | The order-only prerequisites. |
| `$(.SHELLSTATUS)` | Exit status of the most recent `$(shell ...)` call. |
| MAKEFLAGS | Flags passed implicitly to recursive sub-makes. |
| MAKELEVEL | How deeply nested the current sub-make is. 0 at the top level. |
| CURDIR | The current working directory of make. |
| MAKEFILE_LIST | List of all Makefiles read (top-level + included). |
| `$(MAKE)` | The name of the make executable, used for recursion. |
| include | Pull in another file as part of the Makefile. Errors if missing. |
| -include | Same as `include` but silent if file missing. |
| sinclude | Synonym for `-include`. |
| ifeq | Begin a block conditional on string equality. |
| ifneq | Begin a block conditional on string inequality. |
| ifdef | Begin a block conditional on a variable being defined. |
| ifndef | Begin a block conditional on a variable being undefined. |
| `$(if A,B,C)` | Function: if A non-empty return B else C. |
| `$(or A,B)` | First non-empty argument. |
| `$(and A,B)` | Last argument if all non-empty, else empty. |
| `$(filter X,Y)` | Words in Y matching pattern X. |
| `$(filter-out X,Y)` | Words in Y not matching pattern X. |
| `$(sort)` | Sort and de-duplicate. |
| `$(strip)` | Trim whitespace. |
| `$(subst A,B,STR)` | Plain text replace A with B. |
| `$(patsubst)` | Pattern-based substitution. |
| `$(words)` | Count words. |
| `$(word)` | Nth word. |
| `$(wordlist)` | Slice of words. |
| `$(firstword)` | First word. |
| `$(lastword)` | Last word. |
| `$(dir)` | Directory part of each path. |
| `$(notdir)` | Filename part of each path. |
| `$(suffix)` | Suffix of each path. |
| `$(basename)` | Path without final suffix. |
| `$(addsuffix)` | Add suffix to each word. |
| `$(addprefix)` | Add prefix to each word. |
| `$(join)` | Pairwise concat of two lists. |
| `$(realpath)` | Canonical path (follows symlinks). |
| `$(abspath)` | Absolute path (does not follow symlinks). |
| `$(error)` | Abort with a message. |
| `$(warning)` | Print a warning, keep going. |
| `$(info)` | Print a message, keep going. |
| `$(eval)` | Re-parse a string as Makefile syntax. |
| `$(value)` | Get a variable's unexpanded text. |
| `$(call F,A,B,C)` | Invoke a user function with arguments. |
| `$(foreach var,list,body)` | Loop, expanding body once per item in list. |
| `$(shell cmd)` | Run a shell command and substitute its output. |
| `$(origin VAR)` | Where the variable was defined. |
| `$(flavor VAR)` | `simple`, `recursive`, or `undefined`. |
| `$(file >>X,content)` | Write content to file X (GNU make 4.0+). |
| `$(guile)` | Embedded Guile Scheme support (GNU make built with Guile). |
| recursive make | The pattern of one make invoking another (`cd dir && $(MAKE)`). |
| single-makefile pattern | One Makefile that includes everything. The non-recursive style. |
| automake | GNU tool that generates `Makefile.in` from a higher-level `Makefile.am`. |
| autoconf | GNU tool that generates `configure` scripts from `configure.ac`. |
| libtool | GNU tool for portable shared library handling. |
| GNU build system | Autoconf + Automake + Libtool, the classic `./configure; make; make install` toolchain. |
| autoreconf | Re-runs the autotools to regenerate `configure` and `Makefile.in`. |
| configure | The script that probes the system and generates a Makefile from `Makefile.in`. |
| m4 | Macro processor used by autoconf. |
| scons | Python-based build system. Alternative to make. |
| waf | Another Python-based build system. |
| ninja | Fast minimal build system. Often used as a make replacement, generated by CMake/Meson. |
| meson | High-level build system. Generates ninja files. |
| bazel | Google's hermetic, scalable build system. |
| buck | Facebook's build system, similar idea to bazel. |
| pants | Twitter's monorepo build system. |
| redo | DJB's "make replacement done right." Uses scripts as recipes. |
| tup | Build system that watches the filesystem for changes. |
| mk | Plan 9 build system, simpler than make. |
| cmake | Generates Makefiles (or ninja files, or VS project files) from CMakeLists.txt. |
| ccache | Compiler cache. Wraps `cc` and reuses outputs based on source hash. |
| sccache | Mozilla's distributed compiler cache. |
| distcc | Distribute C compilation across machines. |
| icecream | Distributed compilation tool. |
| distmake | Distributed make. |
| -j | Parallelism flag. `make -j8` runs up to 8 recipes in parallel. |
| jobserver | Mechanism GNU make uses to coordinate parallelism across recursive makes. |
| GNU make jobserver | Token-passing system that limits total parallelism even across nested makes. |
| FIFO jobserver | Newer jobserver variant using a FIFO instead of pipes. |
| grouped target | Target syntax `a b &: c` (GNU make 4.3+) for "one recipe produces multiple files." |
| .NOTINTERMEDIATE | (GNU make 4.4) Keeps a target from being auto-deleted as intermediate. |
| --output-sync | Group output of parallel jobs by job, not interleaved. |
| `$(VAR:.x=.y)` | Substitution reference. Same as `$(patsubst %.x,%.y,$(VAR))`. |
| double-quote rules | Subtleties of how shells handle quoted recipe lines. |
| define-endef | Syntax for multi-line variable definitions. |
| .ONESHELL: | Make the whole recipe run in a single shell invocation. |
| recipe prefix character | The character that marks recipe lines (default: tab; configurable via `.RECIPEPREFIX`). |
| `@` recipe prefix | Suppress the echoing of the command before running. |
| `-` recipe prefix | Ignore non-zero exit status of this command. |
| `+` recipe prefix | Run this command even in dry-run (`-n`) mode. |
| `%` (stem) | The wildcard portion in a pattern rule. |
| vpath | Directive specifying search paths for prerequisites by pattern. |
| VPATH | Variable specifying global search paths for prerequisites. |
| Makefile.am | Automake input — high-level description. |
| Makefile.in | Automake output / configure input — template. |
| configure-generated Makefile | The actual Makefile that lands in your tree after `./configure`. |
| hidden ./.deps directory | Where automake puts auto-generated dependency files. |
| -MMD | Compiler flag (gcc/clang) to emit `.d` files for headers, skipping system headers. |
| -MP | Companion to `-MMD`: emit phony targets for each header to survive deletions. |
| .d files | Auto-generated dependency fragments included by `-include $(DEPS)`. |
| *.depend | Same idea as `.d`, older convention. |
| header tracking | The general practice of recording which headers a source file actually included. |
| stale | A target whose mtime is older than at least one of its prerequisites. |
| up to date | A target whose mtime is newer than all of its prerequisites. |
| dependency graph | The DAG of targets and prereqs that make walks. |
| DAG | Directed acyclic graph. |
| stem | The `%` portion of a pattern rule that matched. |
| dry run | `make -n` — print commands without executing. |
| force rebuild | `make -B` — pretend everything is stale. |

## Diagrams

### A target dependency DAG

```
       +------------+
       |   myapp    |   <- top target
       +-----+------+
             |
   +---------+---------+
   |         |         |
   v         v         v
  main.o   parser.o  util.o   <- intermediate targets
   |         |         |
   v         v         v
  main.c   parser.c  util.c   <- source files (leaves)
   |         |         |
   v         v         v
  app.h    app.h     app.h    <- shared headers
```

Make walks this graph from the top, recursively. For each node it asks: "Do I exist? Am I newer than all my children?" If no to either, run the recipe.

### Parallel build with the jobserver

```
        TOP-LEVEL MAKE
        +------------+
        | jobserver  |   <- holds N tokens (N = -j value)
        +-----+------+
              |
    +---------+---------+
    |         |         |
    v         v         v
  worker A   worker B  worker C
   takes      takes     takes
   token      token     no token (none left), waits
```

When you say `make -j4`, the top-level make creates a pool of 4 tokens. Every time a recipe wants to run, it grabs a token. When the recipe finishes, it returns the token to the pool. Sub-makes share the same pool through pipes/FIFOs, so you don't accidentally launch `j*j` jobs when nesting.

### mtime comparison flow

```
    +-------------------+
    | make hello called |
    +---------+---------+
              |
              v
    +-------------------+    no    +-----------------+
    | hello exists?     |--------->| run recipe      |
    +---------+---------+          +-----------------+
              | yes
              v
    +-------------------+    yes   +-----------------+
    | any prereq newer? |--------->| run recipe      |
    +---------+---------+          +-----------------+
              | no
              v
    +-------------------+
    | "up to date"      |
    +-------------------+
```

That single decision tree, applied recursively, is the entire algorithm.

### Recursive vs single-makefile project layout

```
RECURSIVE STYLE:                 SINGLE-MAKEFILE STYLE:

project/                         project/
+-- Makefile                     +-- Makefile               <- knows everything
+-- src/                         +-- src/
|   +-- Makefile (full)          |   +-- rules.mk           <- contributes to top
|   +-- main.c                   |   +-- main.c
+-- lib/                         +-- lib/
|   +-- Makefile (full)          |   +-- rules.mk
|   +-- foo.c                    |   +-- foo.c
+-- tests/                       +-- tests/
    +-- Makefile (full)              +-- rules.mk
    +-- test_foo.c                   +-- test_foo.c

Each subdir's Makefile           Top Makefile includes
runs as its own make             every rules.mk; one
process. Sees only its           graph, one make
own subdir.                      process, full visibility.
```

Recursive is easier to write, harder to scale. Single-makefile is harder to write, scales much better. For a project under 50 files, either works; over that, single-makefile or a modern build system pays off.

## Try This

1. Make `/tmp/make-eli5/`. Type the hello-world Makefile by hand. Run `make`. Then run `make` again. Watch the second one say "up to date." Use `touch hello.c` and run `make` once more. Watch it rebuild.
2. Add a `clean` target that removes `hello`. Mark it `.PHONY`. Run `make clean && make` a few times to feel the cycle.
3. Add `-MMD -MP` to `CFLAGS`, change `hello.c` to `#include <stdio.h>` (it already does), and compile. Look for the auto-generated `.d` file: `cat hello.d`. See how the compiler told you about every header.
4. Add a deliberately broken target — write a recipe with eight spaces instead of a tab. Run `make`. Read the `*** missing separator. Stop.` error. Now you have seen it once and you will never confuse it for anything else.
5. Override the compiler from the command line: `make CC=gcc-13` or `make CC=clang`. Watch the recipe pick up your value.
6. Run `make -p > /tmp/db.txt` and skim the output in `less`. Notice how many implicit rules and default variables there are.
7. Add a second source file (`util.c`, `util.h`) and a pattern rule. Build with `make -j2 -B` and watch make compile both `.o` files in parallel.
8. Try `make --debug=v -B`. Read the explanations of every decision make makes. This is the best way to internalize the algorithm.
9. Add a `help` target that uses `grep` and `awk` to print every documented target. Type comments after each target like `## description` and feel the documentation update itself.
10. (Optional) Compare with a `ninja` build. Install `ninja-build`, generate a `build.ninja` with `cmake -G Ninja .` (if your project uses cmake) and time both. Notice that ninja is faster, especially on incremental rebuilds.

## Where to Go Next

Once this sheet feels easy, the dense engineer-grade material is one command away. Stay in the terminal:

- **`cs languages make`** — quick reference card for everyday make commands.
- **`cs build-systems make`** — the dense reference. Real names of every directive, every special target, every variable.
- **`cs build-systems bazel`** — modern hermetic monorepo build system. Different model, similar problem.
- **`cs languages c`**, **`cs languages cpp`** — the two languages most commonly built with make.
- **`cs languages go`**, **`cs languages rust`** — modern languages that have their own build tools but often have a thin Makefile wrapper.
- **`cs ramp-up bash-eli5`** — back to basics if anything in the recipes felt mysterious.
- **`cs ramp-up git-eli5`** — git and make are both about "what changed since last time."
- **`cs ramp-up linux-kernel-eli5`** — what your computer is actually doing while make is running.

## See Also

- `languages/make` — quick reference card.
- `build-systems/make` — engineer-grade reference.
- `build-systems/bazel` — modern hermetic alternative.
- `languages/c` — the language make was originally built for.
- `languages/go` — Go's `go build` is the modern lazy-rebuild story for Go.
- `languages/rust` — Cargo plays the same role for Rust.
- `ramp-up/bash-eli5` — what's happening inside each recipe line.
- `ramp-up/git-eli5` — version control, often paired with make in real projects.
- `ramp-up/linux-kernel-eli5` — the kernel underneath your shell, your filesystem, and every `cc` invocation.

## References

- **gnu.org/software/make/manual** — the official GNU make manual. The single best free source of truth.
- **"Managing Projects with GNU Make" 3rd edition** by Robert Mecklenburg — the book on serious Makefile engineering. Available as a free PDF.
- **GNU Make 4.3 release notes** — introduces grouped targets (`&:`) and `--shuffle`.
- **GNU Make 4.4 release notes** — adds `.NOTINTERMEDIATE`, jobserver-on-FIFO, and tweaks for parallel correctness.
- **"Recursive Make Considered Harmful"** by Peter Miller (1998) — the classic essay arguing for single-Makefile builds.
- **NetBSD bmake manual** — the canonical BSD make documentation, often more readable than the FreeBSD pages.
- **POSIX make spec** — IEEE Std 1003.1, "make" utility section. The portable subset.
- **makefiletutorial.com** — friendly walkthrough of every concept with runnable examples.
- **`man 1 make`** — the manual page. Type `man make` in your terminal.
- **`info make`** — the GNU info pages, much more detailed than the man page. Type `info make`.

Tip: every reference above is reachable from your terminal. `man make` and `info make` both work offline. The Mecklenburg book is on the internet for free as a PDF; download it and read in `zathura`, `mupdf`, or just `less` for the plain-text bits. You really do not need to leave the terminal to learn make.

— End of ELI5 —

When this sheet feels boring (and it will, faster than you think), graduate to `cs build-systems make` — the engineer-grade reference. It uses real names for every directive, special target, variable, and idiom. After that, `cs detail build-systems/make` gives you the academic underpinning: complexity of dependency-graph traversal, why mtime comparison is unsound under some conditions, and how modern build systems improve on the model.

### One last thing before you go

Pick one experiment from the Hands-On section that you have not run yet. Run it right now. Read the output. Try to figure out what each part means, using the Vocabulary table as your dictionary. Don't just trust this sheet — see for yourself. Make is a real tool. It is on your computer (or one `apt install make` away). The commands in this sheet let you poke at it.

Reading is good. Doing is better. Type the commands. Watch make decide what to rebuild.

You are now officially started on your make journey. Welcome.

The whole point of the North Star for the `cs` tool is: never leave the terminal to learn this stuff. Everything you need is here, or one `man` page away, or one `info` page away. There is no Google search you need to do to start understanding make. You can sit at your terminal, type, watch, read, and learn forever.

Have fun. Make is happy to be poked at. Nothing on this sheet will break anything outside of `/tmp/make-eli5/`. Try things. Type commands. Read what comes back. The more you do, the more it all clicks into place.
