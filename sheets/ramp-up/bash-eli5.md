# Bash — ELI5

> Bash is a recipe language for your computer. You write commands one after another, and bash reads them and runs them.

## Prerequisites

(none, but the linux-kernel ELI5 sheet helps a lot)

You do not need to know how to program to read this sheet. You do not need to have written code before. You do need a computer with a terminal on it. If you are on a Mac, the terminal is called "Terminal" and lives in `Applications/Utilities/`. If you are on Linux, your terminal is in your applications menu and is probably called "Terminal" or "Konsole" or "GNOME Terminal." If you are on Windows, install WSL (Windows Subsystem for Linux) and you will get a terminal that runs Ubuntu inside Windows.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

It will help to read the linux-kernel-eli5 sheet first, because bash is the program you talk to in order to talk to the kernel. Bash is the front door. The kernel is the building behind the door.

## Plain English

### Imagine bash is a really patient cook

Picture a cook in a kitchen. The cook is very, very patient. The cook will do exactly what you tell them to do, in the exact order you tell them to do it, and they will not complain. The cook does not improvise. The cook does not guess. The cook will not say, "I think you meant to say carrots, not potatoes." The cook will hear "potatoes" and bring you potatoes.

You give the cook a list of instructions. We call that list a **script**. The cook reads each line of the script, one at a time, and does what it says. If a line says "chop the onion," the cook chops the onion. If the next line says "fry the onion," the cook fries the onion. If a line is wrong — like "chopp the onion" with two p's — the cook freezes and says, "I do not understand 'chopp.' I quit." The cook is very literal.

That cook is bash.

The list of instructions is your shell script. Each instruction is a **command**. Bash reads commands one at a time, runs them, and waits for the next one. When you sit at your terminal and type, you are giving bash one command at a time, live. When you save a list of commands to a file and run that file, bash reads the file from top to bottom and does each one. Either way, the cook is doing what the recipe says.

### Imagine bash is a foreman on a construction site

Here is another picture. Picture a foreman on a busy construction site. There are workers all around: a digger, a bricklayer, a painter, a plumber. The foreman doesn't dig holes. The foreman doesn't lay bricks. The foreman just tells workers what to do.

You walk up to the foreman and say, "Tell the digger to dig a hole, then tell the bricklayer to fill it with bricks, then tell the painter to paint the bricks." The foreman blows their whistle, points at the digger, and says, "Go." When the digger is done, the foreman points at the bricklayer. When the bricklayer is done, the foreman points at the painter.

The foreman is bash. The workers are the actual programs on your computer. Bash itself doesn't list files. Bash itself doesn't search text. Bash itself doesn't even print a date. There are little programs for each of those things. `ls` lists files. `grep` searches text. `date` prints a date. Bash is just the foreman that tells these little programs to run, in the right order, with the right input, and routes their output where you want it.

So when you type `ls` in bash, bash isn't listing files. Bash is finding a program called `ls` somewhere on your computer, running it, waiting for it to finish, and showing you what it printed. Bash is the middleman. Bash is the orchestrator.

### Imagine bash is an assembly line

Bash has a really beautiful trick. It can connect programs together so the output of one becomes the input of the next, like an assembly line. The car body comes off station 1, rolls into station 2, gets paint, rolls into station 3, gets wheels, rolls into station 4, gets a windshield, rolls out the door.

We call that connection a **pipe**, and we write it with the `|` symbol. Bash sets up the conveyor belt. Each program is a station. The output of station 1 flows into station 2. Station 2's output flows into station 3. And so on. Each station only knows how to do one job, and does it well. Bash glues them together.

A real example: `ls | grep cat | wc -l`. That is three stations.
- Station 1: `ls` lists files.
- Station 2: `grep cat` keeps only the lines that have "cat" in them.
- Station 3: `wc -l` counts the lines.

So that pipeline answers, "How many files do I have with 'cat' in the name?" And every station only had to do one tiny job. The genius of bash is that you can chain little tools together to answer huge questions.

### Why so many pictures?

Different pictures help with different ideas. The **cook** picture is best for understanding bash reads commands one by one and does exactly what you say. The **foreman** picture is best for understanding bash itself doesn't do most of the work — little programs do. The **assembly line** picture is best for understanding pipes. Pick whichever feels right.

## What Even Is Bash

Bash stands for "Bourne-Again SHell." It is a pun. The first big shell on UNIX was called the **Bourne shell**, written by Stephen Bourne in 1977. Bash is a remake (a "born-again" version) with more features. Bash was first released in 1989 by Brian Fox, working for the GNU project. It has been the default shell on most Linux systems for thirty years.

A **shell** is the program you talk to when you open a terminal. The shell takes what you type, figures out what you want, runs the right program, and shows you the result. Without a shell, the terminal would just be a black box that did nothing.

There are several shells. Here is the family tree, in plain English:

- **sh** — the original POSIX shell. Tiny. Standard. Boring on purpose. If a script must run on every UNIX in the world, write it in sh.
- **bash** — sh plus a thousand features. The most common shell on Linux. The default on macOS until 2019 (when Apple switched to zsh).
- **dash** — a tiny, fast version of sh. On Ubuntu, `/bin/sh` is actually dash, not bash. Dash starts faster, which makes boot scripts faster.
- **ksh** — the Korn shell. Big in the 80s and 90s. Still used on some commercial UNIX systems like AIX. Mostly retired.
- **zsh** — bash plus even more features, plus a much better autocomplete and themes. Default on macOS since macOS Catalina (10.15). Most "fancy prompt" pictures online are zsh with the `oh-my-zsh` framework.
- **fish** — "the friendly interactive shell." Different syntax from bash. Easier for humans. Annoying when you want bash compatibility, because scripts that work in bash do not work in fish without translation.
- **nushell** — newer, treats shell output as structured data (tables, JSON) instead of plain text. Cool idea, very different from bash.

The main thing to remember: **most scripts in the world are bash or sh.** If you learn bash, you can read 95% of the shell scripts you will encounter. zsh and fish are for sitting at a terminal interactively. bash and sh are for writing scripts.

There is one big trap. On macOS, the default `/bin/bash` is **bash 3.2 from 2007.** That is twenty years out of date. Apple stopped updating it for licensing reasons (bash 4.0 switched to GPL v3, and Apple does not ship GPL v3 software). If you are on a Mac, do `brew install bash` to get a modern bash, then point your scripts at `#!/usr/bin/env bash` so they pick up the new one. We will say more about this later.

## The Three Streams

Every program that bash runs has three magic pipes attached to it. Three streams of bytes flow in and out of every program. They have boring grown-up names but the idea is simple.

```
              +----------------+
   stdin  --> |                | --> stdout
   (fd 0)     |    program     |     (fd 1)
              |                | --> stderr
              +----------------+     (fd 2)
```

- **stdin** (file descriptor 0) — the input pipe. Stuff comes in here. Like a person reading a book: this is the page they are looking at. By default, stdin is connected to your keyboard.
- **stdout** (file descriptor 1) — the output pipe. The program prints stuff here. By default, stdout is connected to your terminal screen.
- **stderr** (file descriptor 2) — the error pipe. The program prints errors here. By default, stderr is also connected to your terminal screen, but it is a different pipe than stdout. This is on purpose.

Why two output pipes? Because you want to be able to send the normal output one place and the errors a different place. Imagine running a program that prints a list of files (good output) but also occasionally prints "warning: I couldn't read this one" (errors). If both went down the same pipe, the warnings would get mixed in with the file list. With two pipes, you can save the file list to disk and print the warnings on screen.

### File descriptors

Every open file or pipe a program has is given a number, called a **file descriptor** (fd for short). The kernel hands these out. fd 0, 1, and 2 are special: they are stdin, stdout, and stderr. Fd 3, 4, 5, etc. are anything else the program opens (a file you opened with `open()`, a network socket, etc.).

When you write `2>&1` in bash (you will see this a lot), the `2` means fd 2 (stderr) and the `&1` means "fd 1 (stdout)." So `2>&1` means "send stderr to wherever stdout is going." We will explain redirection in detail in a moment.

## Pipes

A pipe connects the stdout of one program to the stdin of another. We write a pipe with `|`.

```
   ls            grep cat        wc -l
+--------+   +-------------+   +-------+
| stdin  |   | stdin       |   | stdin |
| stdout |---| stdout      |---| stdout|--> terminal
| stderr |   | stderr      |   | stderr|
+--------+   +-------------+   +-------+
              ^                   ^
              |                   |
        all stderr goes to terminal directly
```

When you write `ls | grep cat | wc -l`:

1. Bash starts `ls`, `grep cat`, and `wc -l` all at the same time.
2. Bash connects ls's stdout to grep's stdin.
3. Bash connects grep's stdout to wc's stdin.
4. Bash connects wc's stdout to your terminal.
5. As ls produces output, it flows into grep. As grep produces output, it flows into wc. As wc produces output, it shows on your screen.

All three programs run **at the same time**. They are not serialized. Bash uses kernel pipes (created with the `pipe()` syscall) and just lets the kernel buffer between them. If grep is slower than ls, the pipe buffer fills up and ls is paused by the kernel until grep catches up. If grep is faster than ls, grep blocks waiting for more input. The kernel handles all of this for you.

**Important:** pipes only connect stdout, not stderr. If `ls` prints an error to stderr, that error goes straight to your terminal — it does NOT go into grep. If you want errors to also flow through the pipe, you have to merge them first: `ls 2>&1 | grep error`.

## Redirection

Redirection is how you move a stream somewhere other than the default. You change the destination of stdout, stderr, or stdin.

The redirect operators:

- `>` — redirect stdout to a file (overwrite). `cmd > out.txt` writes cmd's output to out.txt, replacing whatever was there.
- `>>` — redirect stdout to a file (append). `cmd >> out.txt` adds cmd's output to the end of out.txt.
- `<` — redirect stdin from a file. `cmd < in.txt` reads in.txt as cmd's input.
- `<<EOF` — heredoc. Reads literal text until a closing tag. `cat <<EOF` then text then `EOF` on its own line.
- `<<<` — herestring. A single-string version of stdin. `grep cat <<< "the cat sat"` is equivalent to `echo "the cat sat" | grep cat`.

You can also redirect specific file descriptors:

- `2>` — redirect stderr to a file. `cmd 2> errors.log`.
- `2>&1` — redirect stderr to wherever stdout currently is.
- `1>&2` — redirect stdout to wherever stderr currently is.
- `&>` — redirect both stdout and stderr to a file (bash 4+). `cmd &> all.log`.
- `2>/dev/null` — throw away stderr.
- `>/dev/null` — throw away stdout. `cmd > /dev/null 2>&1` throws away everything.

### The 2>&1 ordering trap

This is the single most-asked bash question on Stack Overflow. The order of redirections matters because bash applies them left to right.

```bash
cmd > file 2>&1     # both stdout AND stderr go to file
cmd 2>&1 > file     # only stdout goes to file; stderr goes to terminal
```

ASCII for the first one (`cmd > file 2>&1`):

```
Step 1: cmd > file
  fd 1 (stdout) --> file
  fd 2 (stderr) --> terminal

Step 2: 2>&1
  fd 2 = duplicate of fd 1 = file
  
Result:
  fd 1 --> file
  fd 2 --> file
```

ASCII for the second one (`cmd 2>&1 > file`):

```
Step 1: 2>&1
  fd 2 = duplicate of fd 1 = terminal
  fd 1 --> terminal
  fd 2 --> terminal

Step 2: > file
  fd 1 --> file
  fd 2 --> still terminal (it was set to a copy of fd 1's old value)

Result:
  fd 1 --> file
  fd 2 --> terminal
```

Order matters. The trick: read redirections left to right and remember `2>&1` copies the **current** target of fd 1, not the future one.

The shorthand `&>` (bash 4+) avoids this footgun. `cmd &> file` always sends both to the file. Use `&>` when you can.

### Heredocs

A heredoc is a way to type multi-line text directly into a command's stdin without making a file.

```bash
cat <<EOF
This is line one.
This is line two.
EOF
```

That sends the two lines into cat, which prints them. The "EOF" on its own line is the closing tag. You can use any word, not just EOF, but EOF is traditional.

Variations:

- `<<EOF` — variables and command substitutions are expanded inside the heredoc.
- `<<'EOF'` — quoted: NO expansion, the text is literal.
- `<<-EOF` — leading tabs (only tabs, not spaces) are stripped, so you can indent the heredoc with the surrounding code.

```bash
name="world"
cat <<EOF
hello, $name
EOF
# prints: hello, world

cat <<'EOF'
hello, $name
EOF
# prints: hello, $name (literal)
```

### Herestrings

A herestring (`<<<`) is a one-line shortcut for piping a string into stdin.

```bash
grep cat <<< "the cat sat on the mat"
# prints: the cat sat on the mat
```

It is exactly equivalent to `echo "the cat sat on the mat" | grep cat`, but a tiny bit faster (no fork) and easier to type.

## Variables and Quoting

A **variable** is a sticky note. You write a name on it and you write a value on it. Later, when you say the name, bash hands you the value.

```bash
name="world"
echo "hello, $name"
# prints: hello, world
```

The dollar sign tells bash, "look this up." `$name` becomes whatever is on the sticky note named "name." Without the dollar sign, `name` is just the literal four letters n-a-m-e.

**Important rules:**

1. **No spaces around `=`.** `name="world"` works. `name = "world"` does NOT work — bash thinks you are trying to run a program called `name` with arguments `=` and `world`.
2. **Use `$NAME` or `${NAME}`.** `$NAME` is short for `${NAME}`. The braces are needed when the variable name touches other letters: `${NAME}_extra` is the variable NAME followed by `_extra`. `$NAME_extra` is the variable `NAME_extra` (a totally different variable).

### Single vs double quotes

Bash has two main quote styles, and they behave very differently.

- **Double quotes** `"..."` — variables and command substitutions ARE expanded. `\"` is a literal double quote.
- **Single quotes** `'...'` — NOTHING is expanded. The string is fully literal. You can't even put a single quote inside.
- **Backticks** `` `...` `` — old way to do command substitution. Don't use anymore. Use `$(...)`.

```bash
name="world"
echo "hello, $name"   # hello, world
echo 'hello, $name'   # hello, $name (literal, no expansion)
```

Rule of thumb: **always use double quotes** when you are using a variable. The exceptions are when you literally want a `$` to be a `$` (rare) or when you want a multi-line literal block (use single quotes or a heredoc).

### Why backticks are deprecated

Backticks (`` `cmd` ``) are an old syntax for "run cmd and put the output here." Do not use them. Use `$(cmd)` instead.

```bash
# old, do not use
files=`ls`

# modern, use this
files=$(ls)
```

Reasons backticks are bad:
- They cannot nest cleanly. `` `cmd1 `cmd2`` `` is broken; you have to escape: `` `cmd1 \`cmd2\`` ``. Awful.
- They are easy to confuse with single quotes visually.
- They are not POSIX-required (POSIX explicitly recommends `$(...)` instead).

### $() vs ${}

Two completely different things. People confuse these constantly.

- `$(cmd)` — **command substitution.** Runs cmd, replaces with cmd's stdout.
- `${var}` — **parameter expansion.** Looks up the variable.

```bash
date_now=$(date)        # runs `date`, stores output
echo "${date_now}"      # looks up the variable date_now
```

## Word Splitting and Globbing

After bash expands variables and command substitutions, it does one more pass: **word splitting** and **glob expansion.**

**Word splitting** is bash chopping a string into separate words wherever whitespace appears. The whitespace characters bash uses for splitting are stored in the special variable `IFS` (Internal Field Separator). The default value of IFS is space, tab, newline.

```bash
greeting="hello world"
some_cmd $greeting        # bash splits into TWO arguments: "hello" "world"
some_cmd "$greeting"      # bash passes ONE argument: "hello world"
```

The single most common bash bug: **forgetting to quote variables.** If a filename has a space in it, and you forget to quote, bash will split it into two arguments and your program will try to open two files that don't exist.

```bash
file="my report.txt"
rm $file       # broken: tries to rm "my" and "report.txt"
rm "$file"    # correct: removes the file with the space
```

**Quote your variables.** Always. Unless you have a very specific reason not to. The reason not to is when you actively want word splitting (rare).

**Globbing** (also called "filename expansion" or "pathname expansion") is bash expanding patterns like `*` and `?` into filenames.

- `*` — matches any number of any characters except `/`. `*.txt` matches all files ending in `.txt`.
- `?` — matches exactly one character. `file?.txt` matches `file1.txt`, `fileA.txt`, etc.
- `[abc]` — matches one character that is a, b, or c.
- `[a-z]` — matches one lowercase letter.
- `[!abc]` — matches one character that is NOT a, b, or c.
- `**` — matches recursively, but only when `globstar` is enabled (`shopt -s globstar`). `**/*.txt` then matches every .txt under the current dir, at any depth.

If a glob doesn't match anything, by default bash leaves the literal pattern in place:

```bash
ls *.nonexistent
ls: *.nonexistent: No such file or directory
```

You can change this with `shopt -s nullglob` (no match = empty), `shopt -s failglob` (no match = error), `shopt -s nocaseglob` (case insensitive), `shopt -s dotglob` (also match dotfiles).

## Brace Expansion {a,b,c}

Brace expansion is bash generating multiple strings from one pattern. It happens BEFORE variable expansion and BEFORE globbing.

```bash
echo {a,b,c}
# prints: a b c

echo file{1,2,3}.txt
# prints: file1.txt file2.txt file3.txt

mkdir -p project/{src,test,docs}
# creates project/src, project/test, project/docs

echo {1..5}
# prints: 1 2 3 4 5

echo {a..e}
# prints: a b c d e

echo {0..10..2}
# prints: 0 2 4 6 8 10 (step 2)
```

Brace expansion is purely textual — it does NOT check whether files exist. `echo file{1,2,3}.txt` always prints those three names whether the files exist or not. That is the difference between brace expansion and globbing.

## Tilde Expansion ~

The tilde `~` is shorthand for your home directory.

- `~` — your home dir, e.g. `/Users/govan` on Mac, `/home/govan` on Linux.
- `~/Documents` — the Documents folder in your home dir.
- `~someuser` — that user's home dir.
- `~+` — current dir (same as `$PWD`).
- `~-` — previous dir (same as `$OLDPWD`).

Tilde expansion only happens at the start of a word, and only when the word is unquoted or starts the unquoted portion. `echo "~"` prints the literal tilde, because of the quotes.

## Parameter Expansion

This is the secret weapon of bash. There are about twenty forms of parameter expansion, and most people only know two. Here are the important ones.

### Defaults

```bash
echo "${var:-default}"     # if var is unset OR empty, use "default"
echo "${var:=default}"     # same, AND assign default to var
echo "${var:+alternate}"   # if var IS set and non-empty, use "alternate"
echo "${var:?error msg}"   # if var is unset or empty, exit with error
```

Real example:

```bash
name="${1:-anonymous}"     # use first arg, or "anonymous" if none given
```

### Substring removal

```bash
# remove SHORTEST prefix matching pattern
echo "${var#prefix}"

# remove LONGEST prefix matching pattern
echo "${var##prefix}"

# remove SHORTEST suffix matching pattern
echo "${var%suffix}"

# remove LONGEST suffix matching pattern
echo "${var%%suffix}"
```

The mnemonic: `#` is at the top of a US keyboard (left) — prefix. `%` is at the top of a US keyboard (right) — suffix. (You also see this in `zsh`.)

```bash
file="/path/to/my-document.tar.gz"
echo "${file##*/}"        # my-document.tar.gz (strip everything up to last /)
echo "${file%/*}"         # /path/to        (strip last / and after)
echo "${file%.gz}"        # /path/to/my-document.tar
echo "${file%.tar.gz}"    # /path/to/my-document
echo "${file%.*}"         # /path/to/my-document.tar (strip last .ext)
echo "${file%%.*}"        # /path/to/my-document (strip first .ext onward)
```

### Substring replacement

```bash
echo "${var/old/new}"     # replace first "old" with "new"
echo "${var//old/new}"    # replace ALL "old" with "new"
echo "${var/#old/new}"    # replace if "old" is at the START
echo "${var/%old/new}"    # replace if "old" is at the END
```

Example:

```bash
greeting="hello, world, world"
echo "${greeting/world/earth}"     # hello, earth, world
echo "${greeting//world/earth}"    # hello, earth, earth
```

### Length

```bash
var="hello"
echo "${#var}"     # 5
```

### Slice

```bash
var="hello, world"
echo "${var:7}"      # world (offset 7 to end)
echo "${var:7:3}"    # wor   (offset 7, length 3)
echo "${var: -5}"    # world (last 5 chars; mind the space before -)
echo "${var: -5:3}"  # wor
```

### Indirect reference

```bash
greeting="hello"
varname="greeting"
echo "${!varname}"     # hello (looks up the variable named in varname)
```

This is the "old style" indirect reference. Modern bash (4.3+) prefers `declare -n` for namerefs:

```bash
declare -n ref=greeting
echo "$ref"     # hello
ref="goodbye"
echo "$greeting"   # goodbye (because ref is a nameref to greeting)
```

### Case modification

```bash
var="hello"
echo "${var^}"     # Hello (uppercase first letter)
echo "${var^^}"    # HELLO (uppercase all)
echo "${var,}"     # hello (lowercase first letter — no-op here)
echo "${var,,}"    # hello (lowercase all)
```

### Default value tricks

A clean idiom for "first arg, or default":

```bash
target="${1:-localhost}"   # use $1, or "localhost" if $1 is empty/unset
```

A clean idiom for "fail loudly if a required env var is missing":

```bash
api_key="${API_KEY:?must set API_KEY}"
```

That second one will exit the script with an error if `API_KEY` isn't set. Way better than running and breaking later.

## Arithmetic $((expr))

Bash can do integer math (no floats — for floats you call `bc` or `awk`).

```bash
echo $(( 2 + 3 ))         # 5
echo $(( 10 / 3 ))        # 3 (integer division)
echo $(( 10 % 3 ))        # 1 (modulo)
echo $(( 2 ** 10 ))       # 1024 (power)
echo $(( 0xff ))          # 255 (hex)
echo $(( 010 ))           # 8 (octal — leading zero!)
echo $(( 1 << 8 ))        # 256 (bit shift left)
echo $(( 1 & 3 ))         # 1 (bitwise AND)
echo $(( 1 | 2 ))         # 3 (bitwise OR)

count=10
(( count++ ))
echo "$count"             # 11
```

The double-paren `(( ))` form is a "command form" that returns 0 if the result is non-zero, and 1 if zero. So you can use it in conditionals:

```bash
if (( count > 5 )); then
    echo "big"
fi
```

Inside `(( ))`, you do NOT need `$` before variable names — bash already knows you're in arithmetic mode.

**Octal trap:** `010` is 8, not 10. Leading zero means octal. To force decimal: `10#10`. Real bug:

```bash
hour="08"
echo $((hour + 1))   # error! 08 isn't valid octal
echo $((10#$hour + 1))  # works, prints 9
```

This bites people writing log scripts that parse `date +%H`.

## Conditionals

### if / then / elif / else

```bash
if [[ $name == "world" ]]; then
    echo "hi world"
elif [[ $name == "moon" ]]; then
    echo "hi moon"
else
    echo "hi unknown"
fi
```

The `;` after the test condition can also be a newline. The `then` MUST come before any commands. The `fi` (`if` spelled backwards) closes the block.

### [ vs [[ vs ((

This is bash's most confusing area. There are three "test" forms:

- `[ test ]` — old POSIX form. Same as the program `test`. Available in every shell. Limited.
- `[[ test ]]` — bash-specific. More powerful. Doesn't word-split or glob inside. Use this in bash scripts.
- `(( math ))` — arithmetic test. Use for number comparisons.

```bash
# string equality
[ "$a" = "$b" ]            # POSIX form
[[ $a == $b ]]             # bash form (no quotes needed)
[[ $a == foo* ]]           # bash supports glob match!
[[ $a =~ ^foo[0-9]+$ ]]    # bash supports regex match!

# integer equality (use arithmetic, not [[ ]] — see common confusions)
(( a == b ))
(( a > b ))
(( a >= 5 && a < 10 ))

# old-style integer in [ ] uses -eq -ne -lt -le -gt -ge
[ "$a" -eq "$b" ]
[ "$a" -gt 5 ]
```

**Trap:** `[[ $a > $b ]]` is a STRING comparison, not numeric. So `[[ 10 > 9 ]]` is FALSE because "10" comes alphabetically before "9". For numbers always use `(( ))`.

### Test operators (for [ ] and [[ ]])

File tests:
- `-e file` — exists
- `-f file` — is a regular file
- `-d file` — is a directory
- `-L file` — is a symlink
- `-r file` — is readable
- `-w file` — is writable
- `-x file` — is executable
- `-s file` — exists and is not empty
- `file1 -nt file2` — file1 is newer than file2
- `file1 -ot file2` — file1 is older than file2
- `file1 -ef file2` — same inode (same file)

String tests:
- `-z $s` — string is empty
- `-n $s` — string is non-empty
- `s1 = s2` — equal (POSIX)
- `s1 == s2` — equal (bash)
- `s1 != s2` — not equal
- `[[ s1 =~ regex ]]` — regex match (bash)

Integer tests (in `[ ]`):
- `-eq` `-ne` `-lt` `-le` `-gt` `-ge`

Logical:
- `!` — not
- `-a` `-o` — and/or in `[ ]` (avoid; use `&&` `||` between brackets instead)
- `&&` `||` — and/or in `[[ ]]` (use these)

## Loops

### for loop

```bash
for fruit in apple banana cherry; do
    echo "$fruit"
done

# loop over files
for f in *.txt; do
    echo "$f"
done

# C-style
for (( i = 0; i < 10; i++ )); do
    echo "$i"
done

# range
for i in {1..10}; do
    echo "$i"
done
```

### while loop

```bash
i=0
while (( i < 5 )); do
    echo "$i"
    (( i++ ))
done

# read a file line by line (THE canonical idiom)
while IFS= read -r line; do
    echo "got: $line"
done < some_file.txt
```

Why `IFS= read -r`?
- `IFS=` (empty) means: don't strip leading/trailing whitespace.
- `-r` means: don't process backslash escapes (raw mode).

Without those, your read will silently mangle lines with leading spaces or backslashes.

### until loop

```bash
i=0
until (( i >= 5 )); do
    echo "$i"
    (( i++ ))
done
```

`until` is just `while not`. It loops until the condition becomes true. Most people use `while` and never use `until`.

### select loop (interactive menus)

```bash
select choice in apple banana cherry quit; do
    case "$choice" in
        quit) break ;;
        *) echo "you chose $choice" ;;
    esac
done
```

`select` prints a numbered menu and reads the user's choice. Useful for quick CLI menus.

## Functions and Local Variables

```bash
greet() {
    local name="$1"
    echo "hello, $name"
}

greet "world"      # hello, world
```

A function is a named piece of code you can call later. Inside the function, `$1` is the first argument, `$2` the second, etc. — same as a script.

`local name="$1"` makes `name` a function-local variable. Without `local`, `name` would be a global variable that leaks out and clobbers anything else with the same name. **Always use `local` for function variables.**

Functions return an exit code (0 = success, non-zero = failure) via `return`. They don't "return" values like in other languages — to return data, print to stdout and use command substitution to capture.

```bash
add() {
    echo $(( $1 + $2 ))
}

result=$(add 3 4)
echo "$result"     # 7
```

If you have to return many values, print them or use a global associative array.

## Arrays

Bash has two array types: indexed (numeric keys) and associative (string keys, bash 4+).

### Indexed arrays

```bash
fruits=(apple banana cherry)
echo "${fruits[0]}"       # apple
echo "${fruits[1]}"       # banana
echo "${fruits[@]}"       # apple banana cherry (all elements)
echo "${#fruits[@]}"      # 3 (number of elements)
echo "${!fruits[@]}"      # 0 1 2 (indices)

fruits+=("date")          # append
echo "${fruits[3]}"       # date

unset 'fruits[1]'         # delete index 1
echo "${fruits[@]}"       # apple cherry date

# loop
for f in "${fruits[@]}"; do
    echo "$f"
done
```

`"${fruits[@]}"` (with quotes!) is the safe way to expand an array as multiple words. Without quotes, word splitting kicks in. Without `[@]`, you only get the first element.

`"${fruits[*]}"` (with `*` instead of `@`) joins all elements with the first character of `IFS`. Subtle. Use `[@]` 99% of the time.

### Associative arrays (bash 4+)

```bash
declare -A colors
colors[apple]="red"
colors[banana]="yellow"
colors[cherry]="red"

echo "${colors[apple]}"        # red
echo "${colors[@]}"            # red yellow red (values)
echo "${!colors[@]}"           # apple banana cherry (keys)

for key in "${!colors[@]}"; do
    echo "$key: ${colors[$key]}"
done
```

**You MUST `declare -A` before using.** If you skip it, bash silently makes an indexed array and `colors[apple]` becomes `colors[0]` (because `apple` evaluates to 0 in arithmetic context). Awful gotcha.

**macOS bash 3.2 does NOT have associative arrays.** This is the #1 reason to install a modern bash on Mac.

## Heredocs <<EOF

Already covered above. Quick reminder of the variants:

- `<<EOF` ... `EOF` — expansions happen.
- `<<'EOF'` ... `EOF` — literal, no expansions.
- `<<-EOF` ... `EOF` — leading tabs stripped (only tabs!).
- `<<<"string"` — herestring; one-line input.

## Process Substitution <(cmd) >(cmd)

This is one of bash's most powerful features. It lets you treat a command's output (or input) as if it were a file.

```bash
# diff the output of two commands as if they were files
diff <(ls dir1) <(ls dir2)

# tee output to two pipelines at once
echo "hello" | tee >(grep h) >(grep e)
```

How does it work? Bash creates a named pipe (or `/dev/fd/N`), runs the inner command in the background writing to it, and replaces `<(cmd)` with the path to the pipe. The outer command sees a "file" path it can read from.

`<(cmd)` is "read from cmd's stdout as if it were a file." `>(cmd)` is "write to cmd's stdin as if it were a file."

You can't seek in the resulting file (it's a pipe, not a real file). Most things don't seek anyway.

**Process substitution is a bash extension.** It does NOT work in plain sh.

## Subshells vs Command Groups

Two ways to group commands:

- `( cmd1; cmd2 )` — runs in a **subshell.** Variable assignments don't escape.
- `{ cmd1; cmd2; }` — runs in the **current shell.** Variable assignments stay.

The braces form requires a space after `{` and a `;` (or newline) before `}`. Easy to mess up.

```bash
x=1
( x=2 )
echo "$x"     # 1 (subshell change is gone)

x=1
{ x=2; }
echo "$x"     # 2 (group change persists)
```

```
+--------- current shell ---------+
|  x=1                            |
|                                 |
|  +--- subshell () ---+          |
|  |  x=2  (lost!)     |          |
|  +-------------------+          |
|                                 |
|  echo "$x" --> 1                |
+---------------------------------+
```

Pipelines also create subshells (in bash, by default — see `lastpipe` shopt). This is why this fails:

```bash
count=0
echo -e "a\nb\nc" | while read line; do
    (( count++ ))
done
echo "$count"     # 0! the while ran in a subshell
```

The fix is process substitution, which runs the loop in the parent:

```bash
count=0
while read line; do
    (( count++ ))
done < <(echo -e "a\nb\nc")
echo "$count"     # 3
```

Or set `shopt -s lastpipe`.

## Job Control

When you run a command with `&` at the end, it runs in the **background.** You get your prompt back immediately while the command keeps running.

```bash
sleep 100 &
[1] 12345        # job 1, PID 12345
```

Job control commands:

- `jobs` — list current jobs.
- `fg %1` — bring job 1 to foreground.
- `bg %1` — resume job 1 in background.
- `kill %1` — kill job 1.
- `disown %1` — detach job 1 from the shell (it survives shell exit).
- `wait` — wait for all background jobs to finish.
- `wait %1` — wait for job 1 specifically.
- `nohup cmd &` — run cmd immune to hangup signals; output goes to nohup.out.

In an interactive shell, you can press Ctrl+Z to suspend the foreground job. It moves to the background as a stopped job. `bg` resumes it. `fg` brings it back.

`%1` is "job 1." `%%` is "current job." `%-` is "previous job." Don't confuse `%1` (job number) with `$1` (positional arg) or PID 1 (init).

## Trap (signal handling)

A **signal** is a kernel message to a process, like "please stop" or "config reloaded." Bash can catch signals with `trap`.

```bash
trap 'echo "caught Ctrl+C"; exit 1' INT
```

Now if you press Ctrl+C, bash runs the echo and exits.

Signals you'll see:

- `INT` (2) — interrupt. Sent by Ctrl+C.
- `TERM` (15) — politely terminate. Default for `kill`.
- `HUP` (1) — hang up. Sent when terminal closes. Often used to mean "reload config" by daemons.
- `KILL` (9) — kill. **Cannot be trapped or ignored.** Use as last resort.
- `STOP` (19) — stop. **Cannot be trapped.**
- `CONT` (18) — continue (resume after STOP).

Pseudo-signals trap supports:

- `EXIT` — runs when the script exits, no matter how.
- `ERR` — runs when any command fails (when `set -e` is on).
- `DEBUG` — runs before every command.
- `RETURN` — runs when a function returns.

The most useful pattern: clean up temp files on exit.

```bash
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
# ... use $tmp ...
# the cleanup runs even if the script crashes
```

## Set Options

`set` and `shopt` are bash's "switches." They turn options on and off. Some options are POSIX (set), some are bash-only (shopt).

The big four for safer scripts:

- `set -e` — **errexit.** Exit immediately if any command exits non-zero. Lots of caveats — see below.
- `set -u` — **nounset.** Treat unset variables as an error.
- `set -o pipefail` — make pipelines fail if ANY stage fails (not just the last).
- `set -x` — **xtrace.** Print every command before running it. Great for debugging.

The "unofficial bash strict mode," from Aaron Maxwell:

```bash
#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'
```

That last line sets IFS to only newline and tab (not space), which prevents word splitting on spaces in filenames. Most modern bash scripts start this way.

**What `set -e` doesn't catch:**

- Failures inside `if`, `&&`, `||` chains.
- Failures of commands inside command substitution (until bash 4.4ish).
- Failures of intermediate stages of a pipeline (use `pipefail`).
- A function that returns non-zero — sometimes ignored, sometimes not, depends on context.

`set -e` is full of subtle gotchas. It is better than nothing but do not assume it catches every failure. Always test your scripts.

Other useful options:

- `set -v` — verbose. Print each line as it's read (before running).
- `shopt -s nullglob` — globs that match nothing become empty (rather than literal).
- `shopt -s globstar` — enables `**` recursion.
- `shopt -s dotglob` — globs match dotfiles too.
- `shopt -s lastpipe` — last stage of a pipeline runs in current shell, not a subshell.
- `shopt -s extglob` — extended glob patterns: `?(p)`, `*(p)`, `+(p)`, `@(p)`, `!(p)`.

## Shellcheck and Shellharden

You will write bugs in bash. Everyone does. Use these tools.

- **shellcheck** — a linter for shell scripts. Catches >100 common mistakes. Use it on every script. `brew install shellcheck` or `apt install shellcheck`. Then `shellcheck script.sh`.
- **shellharden** — auto-fixes some shellcheck warnings, mainly missing quotes around variables. `cargo install shellharden`.

Even seasoned bash people use shellcheck. There is no shame.

```bash
shellcheck script.sh
```

It will tell you things like "SC2086: Double quote to prevent globbing and word splitting" with line numbers.

VS Code and most editors have shellcheck integration. Set it up. Save yourself hours.

## Bash vs POSIX

POSIX is the standard that defines what `sh` does. Plain `sh` features are guaranteed to work in any POSIX shell: bash, dash, ksh, busybox sh, etc.

When to use which shebang:

- `#!/bin/sh` — write POSIX-only. Will run anywhere. Lowest common denominator.
- `#!/bin/bash` — bash-specific. Common, but the path varies on some BSDs.
- `#!/usr/bin/env bash` — finds bash via PATH. Most portable for "I want bash specifically."

**Trap:** on Ubuntu/Debian, `/bin/sh` is dash, not bash. If you use bash extensions (like `[[`, arrays, `<()`) in a script with `#!/bin/sh`, it will work on RHEL (where /bin/sh is bash) and break on Ubuntu (where /bin/sh is dash). If you use bash features, USE THE BASH SHEBANG.

Bash extensions to know about (these are NOT in plain sh):

- `[[ ... ]]`
- arrays
- `(( ... ))`
- process substitution `<()`
- `{a,b}` brace expansion (mostly)
- `$(<file)` (dash supports it but POSIX doesn't require it)
- `local`
- `select`

If you must run on plain sh, write `[ ... ]`, no arrays, use backticks or `$(...)`, no `(( ))`. It is uglier but it runs everywhere. Start every sh script by mentally subtracting these features.

## Common Errors (verbatim)

Here is the exact text bash prints for the errors you'll see most, with the canonical fix.

### `bash: somecmd: command not found`

You typed a command bash can't find. Either it's not installed, or it's not in your PATH. Check `echo $PATH`. Check `which somecmd` and `type somecmd`. Maybe you meant a different name.

### `bash: ./script.sh: Permission denied`

The script doesn't have the executable bit set. Fix:

```bash
chmod +x script.sh
./script.sh
```

Or run it explicitly: `bash script.sh`.

### `bash: ./script.sh: /bin/bash^M: bad interpreter: No such file or directory`

The script has Windows line endings (CRLF). The `^M` is the carriage return. Fix:

```bash
dos2unix script.sh
# or
sed -i 's/\r$//' script.sh
```

### `bash: somefile: ambiguous redirect`

You used a redirection where the target was a variable that expanded to multiple words.

```bash
out="my file.log"
cmd > $out          # broken: bash tries to redirect to "my" and uses "file.log" as next thing
cmd > "$out"        # fixed
```

### `bash: ${foo}: bad substitution`

You used a parameter expansion bash didn't understand. Often caused by:
- Running in `sh` instead of `bash` (sh doesn't support all expansions).
- Typo in the expansion.
- Trying `${foo:0:5}` or similar in a shell that doesn't support it.

Fix: shebang to bash, double-check syntax.

### `bash: [: var: integer expression expected`

You used `-eq` or `-gt` etc. on a value that wasn't a number.

```bash
val=""
[ "$val" -gt 5 ]    # error: "" is not an integer
```

Fix: default first: `val="${val:-0}"`. Or use `[[ -n $val ]]` to check non-empty before.

### `bash: syntax error: unexpected end of file`

You have an unclosed `if`, `for`, `while`, function, heredoc, or quote. Bash hit the end of the file looking for `fi`, `done`, `}`, or the heredoc terminator.

Fix: count your `if/fi`, `for/done`, `function {/}` pairs. Check heredocs. Check that all quotes are paired.

### `bash: command substitution: line N: syntax error near unexpected token`

Inside a `$(...)`, you have unbalanced quotes or parens. The error message will give the line number INSIDE the substitution.

Fix: add a missing `)` or `'` or `"` inside the substitution.

### `bash: printf: format reused with arguments`

You gave `printf` more arguments than placeholders. printf will reuse the format string for the extras. Sometimes this is intentional, sometimes a typo.

```bash
printf "%s\n" a b c       # prints a, b, c each on a line — INTENTIONAL reuse
printf "name=%s age=%d\n" alice 30 bob 40   # reuses format for second pair
```

### `bash: !: event not found`

You used `!` in an interactive shell and bash tried to do history expansion. Inside double quotes, `!foo` is "the most recent command starting with foo." Avoid `!` in interactive double quotes, or use single quotes:

```bash
echo "hello!"     # may break on some bashes
echo 'hello!'     # always works
```

Or `set +H` to disable history expansion entirely.

## Hands-On

Open a terminal. Type these. Watch what happens.

### 1. echo and printf

```bash
echo "hello, world"
printf "%s, %s!\n" "hello" "world"
printf "%-10s %5d\n" "apples" 42      # left-pad and right-pad
```

`printf` is more precise than `echo`. Use printf when formatting matters.

### 2. read

```bash
read -p "what's your name? " name
echo "hi, $name"
```

`-p` shows a prompt. `-r` (raw mode) is recommended.

### 3. command, type, which, hash, builtin

```bash
command -v ls       # prints path of ls
type ls             # ls is /bin/ls
type cd             # cd is a shell builtin
type ll             # ll might be an alias
which ls            # /bin/ls
hash                # show recently-used commands cache
builtin cd /tmp     # explicitly use builtin cd
```

`type` is the bash-native one. `which` is an external program. `command -v` is the most portable. Use `command -v` in scripts.

### 4. alias

```bash
alias ll='ls -alF'
alias                # list all aliases
unalias ll           # remove
```

Aliases live in your shell, not in scripts (by default). They are interactive helpers.

### 5. declare, local, export, readonly, unset

```bash
declare -i count=0          # integer
declare -r pi=3.14          # readonly
declare -a arr=(a b c)      # indexed array
declare -A map; map[k]=v    # associative array
declare -n ref=count        # nameref (4.3+)

readonly pi=3.14            # alternative readonly

myvar="hello"
export myvar                # propagate to child processes

unset myvar                 # remove the variable
```

Inside a function, use `local` instead of `declare` for the same effect. `local` is more idiomatic.

### 6. env

```bash
env                         # all exported variables
env | grep PATH             # just the PATH line
env VAR=value cmd           # run cmd with VAR set
```

### 7. source / dot

```bash
source ~/.bashrc            # re-read your bashrc
. ~/.bashrc                 # same thing, POSIX form
```

`source` runs the file in the CURRENT shell — variables persist. `bash file.sh` runs in a NEW shell — variables don't escape.

### 8. xargs

`xargs` reads stdin and turns each line into an argument to a command.

```bash
ls *.txt | xargs wc -l         # word-count all .txt files
find . -name '*.bak' | xargs rm
find . -name '*.bak' -print0 | xargs -0 rm    # safe with spaces
```

`-print0` and `-0` use NUL bytes as separators, which means filenames with spaces and newlines work. ALWAYS use `-print0` and `-0` when piping `find` output through `xargs`.

### 9. find

```bash
find . -name '*.txt'                    # by name
find . -type f -name '*.log'            # only files
find . -type d                          # only dirs
find . -mtime -7                        # modified in last 7 days
find . -size +100M                      # bigger than 100MB
find . -name '*.bak' -delete            # find and delete
find . -name '*.txt' -exec wc -l {} +   # run wc on each match
```

### 10. grep

```bash
grep "pattern" file.txt
grep -r "pattern" .              # recursive
grep -i "pattern" file.txt       # case insensitive
grep -v "pattern" file.txt       # invert (lines NOT matching)
grep -n "pattern" file.txt       # show line numbers
grep -c "pattern" file.txt       # count matches
grep -E "pat1|pat2" file.txt     # extended regex
grep -F "literal" file.txt       # fixed string (no regex)
```

### 11. sed

```bash
sed 's/old/new/' file.txt              # replace first per line
sed 's/old/new/g' file.txt             # replace all
sed -i 's/old/new/g' file.txt          # in-place edit (Linux)
sed -i '' 's/old/new/g' file.txt       # in-place edit (macOS — needs '')
sed -n '5p' file.txt                   # print line 5
sed -n '5,10p' file.txt                # print lines 5-10
sed '/^$/d' file.txt                   # delete empty lines
```

### 12. awk

```bash
awk '{print $1}' file.txt              # print first field
awk -F: '{print $1}' /etc/passwd       # delimiter is :
awk '$3 > 100' file.txt                # filter rows where col 3 > 100
awk 'NR==1' file.txt                   # first line (head -1)
awk '{sum += $1} END {print sum}' file.txt   # sum column 1
awk '/pattern/ {print $0}' file.txt    # like grep
```

### 13. cut

```bash
cut -d, -f1 file.csv             # first column of CSV
cut -c1-10 file.txt              # first 10 chars of each line
```

### 14. tr

```bash
echo "hello" | tr a-z A-Z         # HELLO
echo "abc123" | tr -d '0-9'       # abc (delete digits)
echo "hello world" | tr ' ' '_'   # hello_world
```

### 15. sort, uniq

```bash
sort file.txt                      # alphabetical
sort -n file.txt                   # numeric
sort -r file.txt                   # reverse
sort -k2 file.txt                  # sort by column 2
sort -u file.txt                   # unique
sort file.txt | uniq               # same (uniq needs sorted input!)
sort file.txt | uniq -c            # with count
sort file.txt | uniq -d            # only duplicates
```

### 16. head, tail

```bash
head file.txt                      # first 10 lines
head -n 20 file.txt                # first 20
tail file.txt                      # last 10 lines
tail -n 50 file.txt                # last 50
tail -f file.log                   # follow (live)
tail -F file.log                   # follow even through rotation
```

### 17. wc

```bash
wc file.txt                        # lines, words, bytes
wc -l file.txt                     # just lines
wc -w file.txt                     # just words
wc -c file.txt                     # just bytes
```

### 18. tee

```bash
echo "hello" | tee file.txt          # write to file AND stdout
echo "hello" | tee -a file.txt       # append
echo "hello" | sudo tee /etc/foo     # the canonical "sudo redirect"
```

`sudo cmd > file` does NOT work because the redirect happens in your shell, not in sudo. `sudo tee file` does work because tee runs as root.

### 19. watch

```bash
watch ls                           # rerun ls every 2 seconds
watch -n 0.5 'ps -ef | grep nginx' # every half-second
```

### 20. timeout

```bash
timeout 5 curl https://example.com    # kill after 5 seconds
timeout 5s sleep 100                  # kills sleep
```

### 21. parallel

```bash
parallel echo ::: a b c               # run echo for each
ls *.txt | parallel gzip              # gzip in parallel
parallel -j 4 'sleep 1; echo {}' ::: 1 2 3 4 5 6   # 4 at a time
```

### 22. getopts

`getopts` is the bash-builtin option parser.

```bash
while getopts "f:vh" opt; do
    case "$opt" in
        f) file="$OPTARG" ;;
        v) verbose=1 ;;
        h) echo "usage: ..."; exit 0 ;;
        \?) echo "bad option" >&2; exit 1 ;;
    esac
done
```

`f:` means `-f` takes an argument. `v` and `h` are flags. `OPTIND` tracks position.

For long options (`--verbose`), use the GNU `getopt` external program (note: `getopt` not `getopts`).

### 23. trap

```bash
trap 'echo INT received' INT
trap 'cleanup' EXIT
trap '' HUP                # ignore HUP entirely
trap - INT                 # restore default INT handler
```

### 24. mapfile / readarray

Read a file into an array, line by line.

```bash
mapfile -t lines < file.txt
echo "${lines[0]}"             # first line
echo "${#lines[@]}"            # how many lines

readarray -t lines < file.txt  # same thing, alias
```

`-t` strips the trailing newline from each line. Without it, every line has a `\n` at the end.

### 25. The unofficial bash strict mode

```bash
#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'
```

Put this at the top of every script. `-e` for exit on error. `-u` for nounset. `pipefail` so pipelines fail. IFS to only newline/tab.

### 26. shellcheck

```bash
shellcheck script.sh
shellcheck -x script.sh         # follow source statements
shellcheck -S error script.sh   # only errors, hide warnings
```

### 27. bash --version

```bash
bash --version
# GNU bash, version 5.2.15(1)-release (x86_64-apple-darwin22)
```

Check what bash you actually have. Apple ships 3.2. Modern is 5.x.

### 28. complete

`complete` registers a tab-completion function for a command.

```bash
complete -p              # list all completions
complete -W "alpha beta gamma" mycmd   # static word list
complete -F _git_completions git       # use a function
```

Most users never write completions, but you should know they exist.

### 29. exec (replacing the shell)

```bash
exec ls               # bash IS REPLACED by ls, terminal closes after
exec > log.txt        # redirect bash's stdout to log.txt FROM NOW ON
exec 2>&1             # merge stderr into stdout for the rest of the script
```

`exec` without a command just sets up redirections for the rest of the script.

### 30. history

```bash
history                  # list recent commands
history -c               # clear
!42                      # rerun command 42
!!                       # rerun last
!ssh                     # rerun last command starting with ssh
^old^new                 # rerun last command, replacing "old" with "new"
```

History is per-shell. Bash writes it to `~/.bash_history` on exit (configurable).

### 31. true, false, : (colon)

```bash
true                # always succeeds (exit 0)
false               # always fails (exit 1)
:                   # null command — does nothing, exits 0

while true; do      # infinite loop
    sleep 1
done

: ${VAR:=default}   # idiom: ensure VAR has a value
```

`:` is a real builtin. It evaluates arguments (so the expansion happens) but does nothing. Useful for setting defaults.

### 32. let and (( ))

```bash
let count=count+1
(( count++ ))             # same idea, more readable
(( count = count * 2 ))
```

Most people prefer `(( ))` over `let`. It's the same.

## Common Confusions

### 1. When to quote variables (rule: always)

Always quote variable expansions: `"$var"`. The only exception is when you specifically want word splitting and globbing, which is rare. If you don't quote, every variable that contains a space, `*`, or other shell metacharacter will silently misbehave.

### 2. The difference between [ and [[

`[` is the old POSIX `test` command, an external program. `[[` is a bash-specific keyword. Use `[[` in bash scripts. It is faster, doesn't word-split, supports `==` patterns and `=~` regex, and is harder to mess up.

### 3. "$@" vs $@ vs "$*" vs $*

- `"$@"` — expands to "$1" "$2" ... — preserves each arg as a separate word, including spaces. **This is what you want 99% of the time.**
- `$@` (no quotes) — gets word-split.
- `"$*"` — joins all args with the first character of IFS (default: space). Single string.
- `$*` (no quotes) — gets word-split, also.

Always: `"$@"`. To pass through arguments to another command: `cmd "$@"`.

### 4. Subshells lose variable changes

A subshell is a child process. Variable changes inside don't escape. This trips up `cmd | while read; do x=...; done` because the `while` runs in a subshell. See "subshells vs command groups."

### 5. cd in a script doesn't affect parent

When you run `bash script.sh` and the script `cd`s somewhere, that doesn't change YOUR shell. The script ran in a new bash, and that bash exited. To `cd` from a script, you must `source` the script: `source script.sh` (or `. script.sh`).

### 6. echo vs printf

`echo` has different behavior across platforms. macOS echo (BSD) interprets `-e` differently than Linux echo (GNU). `printf` is portable and predictable. **For scripts, prefer `printf`.** For interactive use, echo is fine.

### 7. How to test for empty string

```bash
[ -z "$var" ]         # POSIX: empty
[ -n "$var" ]         # POSIX: non-empty
[[ -z $var ]]         # bash
[[ -z ${var:-} ]]     # safe even with set -u (default to empty)
```

The last form survives `set -u` because `${var:-}` substitutes empty if unset.

### 8. Why backticks are deprecated

Backticks `` `cmd` `` cannot nest cleanly and are visually confusing. Use `$(cmd)`. Modern bash, modern style.

### 9. What set -e DOESN'T catch

`set -e` does NOT exit on:
- Failures inside `if` conditions, `while` conditions, `&&` chains, `||` chains.
- Failures inside command substitution (until late bash, and even then it's complicated).
- Failures of intermediate stages of a pipe (use `set -o pipefail`).
- Functions that return nonzero (sometimes).

`set -e` is helpful but NOT a safety net. Verify your error handling explicitly.

### 10. Why pipefail matters

Without `pipefail`, a pipeline's exit code is just the LAST stage's exit code. So this is fine to bash:

```bash
cat /nonexistent | sort
```

`cat` fails, but `sort` succeeds (with empty input), so the pipeline exits 0. Adding `set -o pipefail` makes the pipeline exit nonzero if ANY stage fails.

### 11. What does $0 actually return

In a script, `$0` is the script name as it was invoked (so `bash ./foo.sh` sets `$0` to `./foo.sh`). In a function, `$0` stays as the script name (NOT the function name). Use `$FUNCNAME` for the function name.

In an interactive shell, `$0` is `-bash` or `bash`.

### 12. Difference between source and bash script.sh

- `source script.sh` (or `. script.sh`) — runs IN the current shell. Variable assignments persist. Used to load config files (`source ~/.bashrc`).
- `bash script.sh` — runs in a NEW bash subshell. The new bash exits when done. Variable changes do NOT persist.

If a script needs to change YOUR environment (export vars, change dir), it must be sourced.

### 13. Why `cmd | while read` runs in a subshell

In bash by default, every stage of a pipeline is a subshell. The `while read` loop is a stage, so it's a subshell. Variables changed inside don't escape.

Workarounds:
- Use process substitution: `while read; do ...; done < <(cmd)`.
- Use `shopt -s lastpipe` to make the last stage run in the current shell.
- Use a heredoc: `while read; do ...; done <<EOF\n$(cmd)\nEOF`.

### 14. Why `(( var > 5 ))` works but `[[ var > 5 ]]` does string compare

Inside `(( ))`, `>` is greater-than (numeric). Inside `[[ ]]`, `>` is greater-than as a STRING (lexicographic). So `[[ 10 > 9 ]]` is FALSE because "10" < "9" alphabetically. For numbers, ALWAYS use `(( ))`. For `[[ ]]`, use `-gt`, `-lt`, etc.

### 15. What does `2>&1` actually mean and why does order matter

`2>&1` means "make fd 2 (stderr) point at the same place fd 1 (stdout) currently points to." It is a SNAPSHOT, not a binding.

```bash
cmd > file 2>&1     # 1 -> file, then 2 = copy of 1 (= file). Both go to file.
cmd 2>&1 > file     # 2 = copy of 1 (= terminal). Then 1 -> file. So 2 still goes to terminal.
```

The shorthand `&> file` (bash 4+) avoids the issue.

### 16. echo's -n and -e flags differ across platforms

GNU echo treats `-e` as enable backslash escapes. BSD echo treats it differently. `printf` is consistent.

### 17. The difference between $1 and ${1}

`$1` is the first positional parameter. `${1}` is the same with explicit braces. The braces matter when followed by characters: `${1}suffix` versus `$1suffix` (which would be the variable named `1suffix`).

### 18. PATH and `command not found`

`PATH` is a colon-separated list of directories where bash looks for commands. `echo $PATH` shows it. To add a dir: `export PATH="$HOME/bin:$PATH"`. Note we put new dirs FIRST so they shadow system commands.

If you got `command not found`, either the command isn't installed, isn't in PATH, isn't executable, or has a typo.

### 19. exit code 0 = success, anything else = failure

Bash's exit codes are inverted from what you might expect. Zero is success, non-zero is failure. The reason: there's only one way to succeed, but many ways to fail (each non-zero code can mean a different error).

### 20. bash 3.2 vs bash 5.x

macOS ships bash 3.2 from 2007 because Apple won't ship GPLv3. macOS scripts that want modern bash features (associative arrays, `&>`, `mapfile`, etc.) must:

```bash
#!/usr/bin/env bash    # use the bash on PATH
```

And users install modern bash via Homebrew: `brew install bash`. Apple's `/bin/bash` is forever stuck at 3.2.

## ASCII Diagrams

### Process tree showing fd inheritance

```
   bash (parent)                
   fd 0 = terminal              
   fd 1 = terminal              
   fd 2 = terminal              
       |                        
       | fork()                 
       v                        
   bash (child)                 
   fd 0 = terminal (inherited)  
   fd 1 = terminal (inherited)  
   fd 2 = terminal (inherited)  
       |                        
       | redirect: > out.txt    
       | dup2(fd_outtxt, 1)     
       v                        
   bash (child)                 
   fd 0 = terminal              
   fd 1 = out.txt   <-- changed 
   fd 2 = terminal              
       |                        
       | execve("ls")           
       v                        
   ls (now this process)        
   fd 0 = terminal              
   fd 1 = out.txt               
   fd 2 = terminal              
```

The kernel inherits open fds across fork. Bash sets up redirections (using dup2) BEFORE calling execve. The new program just sees fd 1 already pointing at the file.

### Pipe between two processes

```
   bash creates pipe()           
   pipe = (read_end, write_end)  
                                 
       fork() twice              
                                 
   process A (ls)            process B (grep)
   ----------------          --------------------
   fd 1 = write_end          fd 0 = read_end
   fd 2 = terminal           fd 1 = terminal
                                                
   ls writes to fd 1   --->  grep reads from fd 0
                                                
   when A closes write_end,                     
   B reads EOF and exits                        
```

### Redirection ordering: 2>&1 > file vs > file 2>&1

```
   cmd > file 2>&1
   ----------------
   step 1: > file
     fd 1 -> file
     fd 2 -> terminal
   step 2: 2>&1
     fd 2 = (where fd 1 points now) = file
   FINAL:
     fd 1 -> file
     fd 2 -> file


   cmd 2>&1 > file
   ----------------
   step 1: 2>&1
     fd 2 = (where fd 1 points now) = terminal
   step 2: > file
     fd 1 -> file
     fd 2 -> terminal (already snapshotted)
   FINAL:
     fd 1 -> file
     fd 2 -> terminal   <-- did NOT go to file!
```

### Subshell isolation

```
+------- main shell -------+
|  x=1                     |
|                          |
|  +--- subshell () ---+   |
|  |  x=2              |   |
|  |  echo $x  --> 2   |   |
|  +-------------------+   |
|                          |
|  echo $x  --> 1          |
+--------------------------+
```

The subshell sees x=1 (inherited), changes to x=2 in its own copy. When the subshell exits, the change dies. Main shell still has x=1.

### Command-line parsing pipeline

```
   you type: echo "hi $name" *.txt
                |
                v
   1. Tokenize / split into words
                |
                v
   2. Brace expansion: {a,b}
                |
                v
   3. Tilde expansion: ~user
                |
                v
   4. Parameter expansion: $name -> "world"
   5. Arithmetic expansion: $((1+2))
   6. Command substitution: $(cmd)
                |
                v
   7. Word splitting (using IFS) — only on UNQUOTED expansions
                |
                v
   8. Pathname expansion (globbing): *.txt -> a.txt b.txt
                |
                v
   9. Quote removal: strip the literal quotes
                |
                v
   ARGV: ["echo", "hi world", "a.txt", "b.txt"]
                |
                v
   execute: echo with those args
```

This order matters. For example, brace expansion happens before parameter expansion, so `{a,b}$VAR` is `a$VAR b$VAR`, then both expand. Globbing happens after parameter expansion, so `$VAR` containing `*` expands to filenames.

## Vocabulary

| Word | Plain English |
|---|---|
| shell | The program you talk to in a terminal that runs commands. |
| login shell | A shell started when you log in. Reads `~/.bash_profile` (or similar). |
| interactive shell | A shell you type commands into. Reads `~/.bashrc`. |
| non-interactive shell | A shell running a script (no human typing). Doesn't read bashrc. |
| builtin | A command implemented inside bash itself (like `cd`, `echo`, `read`). |
| external command | A command that lives as a separate program file (like `ls`, `grep`). |
| alias | A shortcut name for a longer command. `alias ll='ls -alF'`. |
| function | A user-defined named block of bash code. |
| command | One unit of work bash runs — a builtin, alias, function, or external. |
| pipeline | Two or more commands joined by `|`, output flowing left to right. |
| redirection | Sending stdin/stdout/stderr to/from a file or another fd. |
| file descriptor | A number the kernel assigns to an open file/pipe (0, 1, 2, ...). |
| fd 0 | Standard input (stdin). |
| fd 1 | Standard output (stdout). |
| fd 2 | Standard error (stderr). |
| stdin | The input stream, fd 0. |
| stdout | The normal output stream, fd 1. |
| stderr | The error output stream, fd 2. |
| pipe | An anonymous OS pipe connecting one program's stdout to another's stdin. |
| named pipe | A pipe with a filename, made with `mkfifo`. Also called FIFO. |
| FIFO | First-in-first-out — another name for a named pipe. |
| heredoc | Multi-line input typed inline with `<<TAG ... TAG`. |
| herestring | One-line input with `<<<"string"`. |
| process substitution | `<(cmd)` or `>(cmd)` — treat a command's I/O as a file. |
| subshell | A child shell created by `(...)`, pipes (default), `$(...)`, `&`, etc. |
| command substitution | `$(cmd)` — replace with cmd's stdout. |
| parameter | A variable's name. `$name` looks up the parameter "name". |
| variable | A named value. Can be a string, number, or array. |
| environment variable | A variable that's been `export`ed and is visible to child processes. |
| special parameter | Built-in $-vars: `$0 $1 $@ $* $# $? $$ $! $-`. |
| positional parameter | `$1`, `$2`, etc. — the script/function arguments. |
| $0 | Script or shell name. |
| $1 .. $9 | First through ninth args. |
| $@ | All args, each as separate word (use quoted: `"$@"`). |
| $* | All args, joined into one word with IFS. |
| $# | Number of positional args. |
| $? | Exit code of the last command. |
| $$ | PID of the current shell. |
| $! | PID of the last background process. |
| $- | Current shell flag list. |
| IFS | Internal Field Separator — chars used for word splitting. Default: space, tab, newline. |
| OPTARG | The argument to the option getopts just parsed. |
| OPTIND | Index of the next argument getopts will look at. |
| exit status | The integer 0–255 a command returns. 0 = success. |
| return | Builtin to set a function's exit code. |
| signal | A short kernel message to a process. |
| SIGINT | "Interrupt" signal. Sent by Ctrl+C. Number 2. |
| SIGTERM | "Terminate" signal. Politely asks process to stop. Number 15. |
| SIGHUP | "Hangup" signal. Number 1. Sent when terminal closes. |
| SIGKILL | "Kill" signal. Cannot be caught. Number 9. |
| trap | Bash builtin to install handlers for signals. |
| job | A pipeline running under a shell, can be foreground or background. |
| foreground | The job currently bound to the terminal (you can type to it). |
| background | A job running with `&` — terminal is free. |
| disown | Remove a job from the shell's table so it survives shell exit. |
| nohup | "No hangup" — run command immune to SIGHUP, output to nohup.out. |
| history | List of past commands stored per-shell. |
| fc | "Fix command" — edit and rerun a recent command. |
| set | Builtin to toggle shell options (`set -e`, etc.). |
| shopt | Builtin to toggle bash-specific options (`shopt -s globstar`, etc.). |
| declare | Define a variable with attributes (integer, array, readonly). |
| typeset | Synonym for declare (ksh-style). |
| readonly | Make a variable unchangeable. |
| local | Inside a function, makes the variable scoped to the function. |
| export | Mark a variable as visible to child processes. |
| unset | Remove a variable or function. |
| hash | Bash's cache of recently-used command paths. |
| type | Builtin: tell what kind of thing a name is (alias, function, builtin, external). |
| command -v | Portable way to check if a name exists. |
| getopts | Bash builtin for parsing short options (-a -b). |
| getopt | External program for parsing long options (--all --bytes). |
| $BASH_SOURCE | Array of source filenames (for the running script and any sourced files). |
| $BASH_VERSION | The version string of the running bash. |
| $LINENO | Current line number in the script. |
| $FUNCNAME | Array of function call stack — `${FUNCNAME[0]}` is the current function. |
| $PIPESTATUS | Array of exit codes of every stage of the last pipeline. |
| complete | Builtin to register tab-completion for a command. |
| compgen | Builtin to generate completion candidates. |
| compopt | Builtin to modify completion options. |
| completion | The system that lets bash auto-finish your typing. |
| prompt | The text bash prints before each command. |
| $PS1 | The primary prompt string. |
| $PS2 | The continuation prompt (when a command spans lines). |
| $PS4 | The prompt printed by `set -x`. Default `+`. |
| RPROMPT | A right-side prompt (zsh feature, not bash). |
| DEBUG trap | A trap that runs before every command. |
| ERR trap | A trap that runs when a command fails (with `set -e`). |
| EXIT trap | A trap that runs when the script ends. |
| RETURN trap | A trap that runs when a function returns. |
| errexit | `set -e` — exit on any error. |
| nounset | `set -u` — unset variable is an error. |
| pipefail | `set -o pipefail` — pipeline fails if any stage fails. |
| xtrace | `set -x` — print each command before running. |
| verbose | `set -v` — print each line as it's read. |
| word splitting | Bash chopping a string into words at IFS chars. |
| glob | A filename pattern using `*`, `?`, `[...]`. |
| globbing | Expansion of glob patterns into filenames. |
| globstar | `**` — recursive glob. Enabled with `shopt -s globstar`. |
| dotglob | `shopt -s dotglob` — globs match dotfiles too. |
| nullglob | `shopt -s nullglob` — non-matching globs become empty. |
| nocaseglob | `shopt -s nocaseglob` — case-insensitive globs. |
| extglob | `shopt -s extglob` — extended globs like `?(p)`, `*(p)`. |
| brace expansion | `{a,b,c}` and `{1..5}` — generate strings. |
| tilde expansion | `~` becomes home dir, `~user` becomes user's home. |
| parameter expansion | All the `${var...}` forms. |
| arithmetic expansion | `$((expr))` — integer math. |
| command substitution | `$(cmd)` — replace with cmd output. |
| process substitution | `<(cmd)` `>(cmd)` — file-like wrapper. |
| history expansion | `!!`, `!42`, `^a^b` — dig into history. |
| here document | The `<<TAG` heredoc form. |
| here string | The `<<<"string"` form. |
| history file | `~/.bash_history` — where command history is saved. |
| .bashrc | Per-user file read by interactive non-login shells. |
| .bash_profile | Per-user file read by login shells. |
| .profile | Per-user file read by sh-compatible login shells. |
| /etc/bash.bashrc | System-wide file read by interactive bash. |
| /etc/profile | System-wide file read at login. |
| /etc/profile.d/ | Directory of files sourced by /etc/profile. |
| login vs non-login | Login: started by `login`/SSH/console. Non-login: started in a window. |
| POSIX | The standard that defines portable shell. |
| sh | The POSIX shell. On Ubuntu it's actually dash. |
| dash | A small fast POSIX-only shell. Ubuntu's `/bin/sh`. |
| ksh | The Korn Shell. Older bash competitor. |
| zsh | A bash-compatible shell with more features. macOS default. |
| fish | A shell with a different (friendlier) syntax. Not bash-compatible. |
| nushell | A shell where output is structured data. |
| shebang | The `#!/path/to/interpreter` first line of a script. |
| script | A file of shell commands you can run. |
| chmod | Change file permissions, including the executable bit. |
| sudo | Run a command as another user (usually root). |
| root | The superuser. Can do anything. |
| PID | Process ID. The kernel's badge number for a process. |
| process | A running program. |
| daemon | A long-running background process. |
| init | PID 1, the first process the kernel starts. |
| fork | Syscall to make a copy of the current process. |
| exec | Syscall to replace the current process's program. |
| wait | Syscall (and bash builtin) to wait for a child to finish. |
| zombie | A child process that finished but parent hasn't `wait()`ed. |
| orphan | A child whose parent died — adopted by init. |
| dup2 | Syscall to make one fd a copy of another. |
| pipe() | Syscall to create an anonymous pipe. |
| open() | Syscall to open a file. |
| read() | Syscall to read bytes from an fd. |
| write() | Syscall to write bytes to an fd. |
| close() | Syscall to close an fd. |
| EOF | End of file — fd has no more bytes. |
| nameref | A variable that is a reference to another variable. `declare -n`. |
| readarray | Builtin alias for mapfile. Reads stdin into an array. |
| mapfile | Builtin to read stdin into an array. |
| EPOCHSECONDS | Bash 5+ variable: current Unix epoch seconds. |
| EPOCHREALTIME | Bash 5+ variable: epoch with microseconds. |
| RANDOM | Builtin variable: a new random number each read. |
| BASHPID | The current bash process's PID (different from $$ in subshells). |
| SHELL | Env var holding the path of your login shell. |
| HOME | Env var holding your home directory path. |
| PWD | Current working directory. |
| OLDPWD | Previous working directory (use `cd -` to swap). |

## Try This

Open a terminal and try these in order. Each one teaches something.

```bash
# 1. Hello, world. The traditional first command.
echo "hello, world"

# 2. Variables.
name="alice"
echo "hi, $name"

# 3. The dangers of unquoted vars.
phrase="hi there"
echo $phrase     # one space between words
echo "$phrase"   # exact spacing

# 4. A simple pipeline.
ls | wc -l       # how many things in this directory

# 5. Redirect to a file.
date > today.txt
cat today.txt

# 6. Append to a file.
echo "later" >> today.txt
cat today.txt

# 7. Discard stdout.
ls /usr > /dev/null

# 8. Capture stderr.
ls /nonexistent 2> errors.log
cat errors.log

# 9. Combine streams.
ls /usr /nonexistent > out.log 2>&1
cat out.log

# 10. A loop.
for i in 1 2 3; do
    echo "step $i"
done

# 11. A loop over files.
for f in *.txt; do
    echo "file: $f"
done

# 12. A simple if.
if [ -f today.txt ]; then
    echo "yes, the file exists"
fi

# 13. Test for empty.
var=""
[ -z "$var" ] && echo "var is empty"

# 14. Compare numbers.
n=42
(( n > 10 )) && echo "big"

# 15. Strict mode test (in a script file).
cat > strict.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'
echo "strict mode running"
unknown_var=$DOES_NOT_EXIST
echo "you will not see this"
EOF
chmod +x strict.sh
./strict.sh

# 16. A function.
greet() {
    local who="${1:-world}"
    echo "hello, $who"
}
greet
greet alice

# 17. An array.
fruits=(apple banana cherry)
echo "${fruits[@]}"
echo "${#fruits[@]}"

# 18. An associative array (bash 4+).
declare -A age
age[alice]=30
age[bob]=40
echo "${age[alice]}"

# 19. A trap.
trap 'echo "exiting"' EXIT
echo "running"
# (script ends here, "exiting" prints)

# 20. Test a script with shellcheck.
shellcheck strict.sh

# 21. Read a file line by line.
seq 5 | while IFS= read -r line; do
    echo "got: $line"
done

# 22. Process substitution diff.
diff <(echo -e "a\nb\nc") <(echo -e "a\nx\nc")

# 23. Use parameter expansion.
file="report.tar.gz"
echo "${file%.tar.gz}"   # report
echo "${file##*.}"       # gz

# 24. Run something for every match.
find /tmp -name '*.tmp' -print0 2>/dev/null | xargs -0 echo "would delete:"

# 25. A simple menu with select.
select choice in red green blue quit; do
    case "$choice" in
        quit) break ;;
        *) echo "you picked $choice" ;;
    esac
done
```

## Where to Go Next

You now know the core of bash. Where next?

- **Read your shell's manual.** `man bash` is intimidating but exhaustive. Search inside with `/`. Look up specific topics: `man bash` then `/parameter expansion`.
- **Read the BashFAQ at mywiki.wooledge.org.** It is the highest-quality bash content on the internet. Pick a question that interests you and read the answer.
- **Read the BashPitfalls page (same site).** Top 50 mistakes bash beginners make. Avoid them.
- **Install shellcheck. Run it on every script you write.** It will teach you bash by complaining at you.
- **Write a real script.** A backup script. A log-rotator. An installer. Something useful. Real scripts teach more than tutorials.
- **Move on to `shell/bash`** — the canonical reference sheet — once the ELI5 picture is solid.
- **Move on to `shell/shell-scripting`** for portable POSIX patterns once you outgrow bash-only stuff.
- **Try zsh** — it is bash-compatible enough to ease into and the interactive UX (autocomplete, theming) is much better.
- **Try fish** — totally different syntax, but seeing a different shell makes you understand bash's choices better.
- **Read the source.** Bash is open source. The C source code is at `git://git.savannah.gnu.org/bash.git`. Skim `parse.y` to see how bash parses commands. Eye-opening.

## See Also

- shell/bash — the full reference cheatsheet
- shell/zsh — the macOS default shell
- shell/shell-scripting — portable shell patterns
- shell/fish — the friendly shell
- terminal/tmux — keep shells running across disconnects
- terminal/screen — the older alternative to tmux
- system/strace — see what syscalls bash and your scripts make
- system/gdb — debug compiled programs (and bash itself)
- ramp-up/linux-kernel-eli5 — what the kernel actually does for bash

## References

- **Bash Reference Manual** — gnu.org/software/bash/manual/bash.html — the official, exhaustive guide.
- **Bash Hackers Wiki (archive)** — wiki.bash-hackers.org/start (and web archive copies) — community deep dives. Site went down in 2023, mirrors live on.
- **Pro Bash Programming** by Chris F.A. Johnson — the textbook. POSIX-friendly, deep.
- **mywiki.wooledge.org/BashFAQ** — the highest-quality FAQ in shell-land.
- **mywiki.wooledge.org/BashPitfalls** — top 50 mistakes. Read this before writing your next script.
- **shellcheck.net** — try snippets in the browser, or install locally with `brew install shellcheck`.
- **`man bash`** — comprehensive but dense. Search with `/keyword`.
- **POSIX shell standard** — pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html — the formal spec for portable shell.
- **"Advanced Bash-Scripting Guide"** by Mendel Cooper — outdated in places but still a useful reference. tldp.org/LDP/abs/html/.
- **GNU `getopt` versus bash builtin `getopts`** — there are two; the article at mywiki.wooledge.org/BashFAQ/035 explains both.
