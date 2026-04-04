# Make (Build Automation)

Build tool that uses dependency graphs to compile targets from source files via rules.

## Targets & Dependencies

### Basic rule

```bash
# target: dependency1 dependency2
# 	command1
# 	command2
```

### Example: compile C program

```bash
# app: main.o utils.o
# 	gcc -o app main.o utils.o
#
# main.o: main.c main.h
# 	gcc -c main.c
#
# utils.o: utils.c utils.h
# 	gcc -c utils.c
```

### Run make

```bash
make                   # build default (first) target
make app               # build specific target
make -j4               # parallel build (4 jobs)
make -n                # dry run (print commands without running)
make -B                # force rebuild all targets
make -f custom.mk      # use a different Makefile
```

## Variables

### Define and use

```bash
# CC = gcc
# CFLAGS = -Wall -O2
# LDFLAGS = -lm
# SRC = main.c utils.c
# OBJ = $(SRC:.c=.o)        # substitution: main.o utils.o
#
# app: $(OBJ)
# 	$(CC) $(CFLAGS) -o $@ $^ $(LDFLAGS)
```

### Override from command line

```bash
make CC=clang CFLAGS="-g -O0"
```

### Variable flavors

```bash
# VAR = value              # recursively expanded (evaluated on use)
# VAR := value             # simply expanded (evaluated on assignment)
# VAR ?= default           # set only if not already defined
# VAR += more              # append
```

### Environment and shell

```bash
# COMMIT := $(shell git rev-parse --short HEAD)
# DATE := $(shell date +%Y-%m-%d)
```

## Automatic Variables

```bash
# $@    target name
# $<    first dependency
# $^    all dependencies (deduplicated)
# $?    dependencies newer than target
# $*    stem of pattern match (e.g., "main" from "main.o")
# $(@D) directory of target
# $(@F) file part of target
```

### Example

```bash
# %.o: %.c
# 	$(CC) $(CFLAGS) -c $< -o $@
# # $< = the .c file, $@ = the .o file
```

## .PHONY

### Declare non-file targets

```bash
# .PHONY: all clean test install lint
#
# all: app
#
# clean:
# 	rm -f *.o app
#
# test:
# 	go test ./...
#
# install: app
# 	cp app /usr/local/bin/
```

Without `.PHONY`, if a file named `clean` exists, `make clean` would do nothing.

## Pattern Rules

### Compile any .c to .o

```bash
# %.o: %.c
# 	$(CC) $(CFLAGS) -c $< -o $@
```

### Compile any .go to binary

```bash
# bin/%: cmd/%/main.go
# 	go build -o $@ ./$(<D)
```

### Static pattern rule

```bash
# OBJECTS = main.o utils.o
# $(OBJECTS): %.o: %.c
# 	$(CC) -c $< -o $@
```

## Conditionals

```bash
# ifdef DEBUG
# CFLAGS += -g -DDEBUG
# else
# CFLAGS += -O2
# endif
#
# ifeq ($(OS),Windows_NT)
# EXE = app.exe
# else
# EXE = app
# endif
#
# ifneq ($(CC),gcc)
# $(warning Not using gcc)
# endif
```

## Functions

### String functions

```bash
# FILES = main.c utils.c test.c
# $(filter %.c, $(FILES))          # main.c utils.c test.c
# $(filter-out test.c, $(FILES))   # main.c utils.c
# $(patsubst %.c, %.o, $(FILES))   # main.o utils.o test.o
# $(subst .c,.o,$(FILES))          # same as above (simpler)
# $(strip  hello  )                # "hello" (trim whitespace)
# $(words $(FILES))                # 3
# $(firstword $(FILES))            # main.c
# $(lastword $(FILES))             # test.c
# $(sort c b a)                    # a b c (also deduplicates)
# $(wildcard src/*.c)              # glob expansion
```

### File functions

```bash
# $(dir src/main.c)                # src/
# $(notdir src/main.c)             # main.c
# $(suffix main.c)                 # .c
# $(basename main.c)               # main
# $(addsuffix .o, main utils)      # main.o utils.o
# $(addprefix src/, main.c)        # src/main.c
```

### Shell function

```bash
# GIT_SHA := $(shell git rev-parse --short HEAD)
# UNAME := $(shell uname -s)
```

## Include

### Include other Makefiles

```bash
# include config.mk
# -include optional.mk    # no error if missing
```

## Common Makefile Patterns

### Go project

```bash
# .PHONY: build test lint clean run
# BINARY = myapp
# VERSION := $(shell git describe --tags --always)
#
# build:
# 	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) .
#
# test:
# 	go test -race -count=1 ./...
#
# lint:
# 	golangci-lint run
#
# clean:
# 	rm -f $(BINARY)
#
# run: build
# 	./$(BINARY)
```

### Docker project

```bash
# .PHONY: build push deploy
# IMAGE = myapp
# TAG := $(shell git rev-parse --short HEAD)
#
# build:
# 	docker build -t $(IMAGE):$(TAG) .
#
# push: build
# 	docker push $(IMAGE):$(TAG)
#
# deploy: push
# 	kubectl set image deployment/myapp myapp=$(IMAGE):$(TAG)
```

### Self-documenting help

```bash
# .PHONY: help
# help: ## Show this help
# 	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
# 		awk 'BEGIN {FS = ":.*## "}; {printf "  %-15s %s\n", $$1, $$2}'
#
# build: ## Build the binary
# 	go build -o app .
#
# test: ## Run tests
# 	go test ./...
```

## Tips

- Recipes (commands) must be indented with a real TAB character, not spaces.
- `.PHONY` prevents conflicts with files that share a target name. Always declare non-file targets.
- `make -n` (dry run) shows what would execute without running anything.
- `$` in shell commands must be escaped as `$$` (e.g., `$$HOME`, `$$variable`).
- `:=` (simply expanded) is usually what you want. `=` (recursively expanded) can cause infinite loops.
- `@` before a command suppresses printing it: `@echo "quiet"`.
- `-` before a command ignores its exit status: `-rm -f maybe_missing`.
- `$(MAKE)` should be used instead of `make` for recursive calls to inherit flags like `-j`.

## See Also

- c, go, bash, shell-scripting, docker, git

## References

- [GNU Make Manual](https://www.gnu.org/software/make/manual/) -- complete reference for GNU Make
- [GNU Make Quick Reference](https://www.gnu.org/software/make/manual/html_node/Quick-Reference.html) -- one-page summary of directives and functions
- [man make](https://man7.org/linux/man-pages/man1/make.1.html) -- make man page
- [POSIX make Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/make.html) -- portable make behavior
- [GNU Make Automatic Variables](https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html) -- `$@`, `$<`, `$^`, etc.
- [GNU Make Functions](https://www.gnu.org/software/make/manual/html_node/Functions.html) -- `$(wildcard)`, `$(patsubst)`, `$(shell)`, etc.
- [Remake](http://bashdb.sourceforge.net/remake/) -- GNU Make with debugger and improved error reporting
- [BSD Make (bmake)](https://www.crufty.net/help/sjg/bmake.html) -- NetBSD make, used on FreeBSD/macOS
