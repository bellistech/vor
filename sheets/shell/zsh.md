# Zsh (Z Shell)

> Extended Bourne shell with powerful globbing, completion, and prompt — default on macOS 10.15+, popular on Linux.

## Setup

Install zsh on common platforms.

```bash
brew install zsh                               macOS via Homebrew (newer than system zsh)
sudo apt install zsh                           Debian / Ubuntu
sudo dnf install zsh                           Fedora / RHEL
sudo pacman -S zsh                             Arch
```

Set zsh as the default login shell.

```bash
which zsh                                      print absolute path, e.g. /opt/homebrew/bin/zsh
sudo sh -c "echo $(which zsh) >> /etc/shells"  whitelist non-system zsh on macOS
chsh -s $(which zsh)                           change YOUR default shell (logout to take effect)
echo $SHELL                                    confirm after re-login
```

Run zsh from inside another shell without changing the login default.

```bash
zsh                                            spawn zsh as subprocess; exit returns to parent
zsh -l                                         start as login shell (sources .zprofile + .zlogin)
zsh -i                                         start as interactive shell (sources .zshrc)
exec zsh                                       replace current shell with zsh (no fork)
```

Detect that you are inside zsh (works in scripts too).

```bash
echo $ZSH_VERSION                              "5.9" — empty in bash/sh
echo $ZSH_NAME                                 "zsh"
[[ -n $ZSH_VERSION ]] && echo "running zsh"    portable test
```

Find the version's feature gate quickly.

```bash
autoload -Uz is-at-least
is-at-least 5.8 && echo "modern enough"
zsh --version                                  shell-out version line
```

## Configuration files

Zsh sources files in a deterministic order. Each is optional; missing files are skipped silently. Get this wrong and you will edit the wrong file forever.

```bash
.zshenv          ALWAYS sourced (login, interactive, scripts, even non-interactive); keep TINY
.zprofile        login shells only, after .zshenv (bash equivalent: .bash_profile)
.zshrc           interactive shells only, after .zprofile
.zlogin          login shells only, after .zshrc (rare; use .zprofile instead)
.zlogout         when a login shell exits
```

Canonical "what goes where":

```bash
.zshenv          PATH, ENV vars used by GUI apps and scripts. KEY: GUI apps inherit only this.
.zprofile        one-time login setup: ssh-agent start, tmux attach, MOTD
.zshrc           aliases, functions, completion, prompt, key bindings, plugins
.zlogin          end-of-login banners; almost always empty
.zlogout         clear screen, log session, kill ssh-agent
```

Relocate dotfiles to keep $HOME tidy (zsh 5.0+ honors $ZDOTDIR).

```bash
ZDOTDIR=$HOME/.config/zsh; export ZDOTDIR     put it in /etc/zshenv to take global effect
ls $ZDOTDIR/.zshrc                             zsh now reads from here
```

System-wide config lives in /etc and is sourced first by login shells.

```bash
/etc/zshenv      sourced before user .zshenv
/etc/zprofile    macOS sets PATH here via path_helper
/etc/zshrc       /etc/zlogin /etc/zlogout
```

A common mistake: putting PATH in .zshrc breaks GUI apps and cron. Put PATH in .zshenv.

```bash
.zshrc           export PATH=...           BROKEN: cron, launchd, LSP servers don't see it
.zshenv          export PATH=...           FIXED: every spawned shell inherits PATH
```

## Frameworks

Vanilla zsh is fast and capable; frameworks trade startup time for batteries-included plugins.

```bash
Oh My Zsh       biggest community, easy themes/plugins, slowest (300-1500ms boot typical)
Prezto          modular, fast, "fork of Oh My Zsh that fixed the speed"
zinit           plugin manager with turbo (lazy) mode; fastest if tuned (<50ms)
antidote        successor to antibody, simple bundle file, fast
zplug           plugin manager with parallel installs, mostly stable
```

Minimal vanilla zsh setup (no framework — ~20ms startup):

```bash
.zshrc:
autoload -Uz compinit && compinit
autoload -Uz colors && colors
autoload -Uz vcs_info
setopt AUTO_CD AUTO_PUSHD PUSHD_IGNORE_DUPS PUSHD_SILENT
setopt SHARE_HISTORY HIST_IGNORE_DUPS HIST_REDUCE_BLANKS HIST_VERIFY
setopt EXTENDED_GLOB GLOB_DOTS NULL_GLOB
HISTSIZE=50000 SAVEHIST=50000 HISTFILE=~/.zsh_history
PROMPT='%F{green}%n@%m%f:%F{blue}%~%f %# '
RPROMPT='${vcs_info_msg_0_}'
zstyle ':vcs_info:git:*' formats ' (%b)'
precmd() { vcs_info }
```

Install Oh My Zsh.

```bash
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
ZSH_THEME="robbyrussell"
plugins=(git docker kubectl z fzf history-substring-search)
source $ZSH/oh-my-zsh.sh
```

Install zinit (turbo mode lets you defer plugins past the first prompt).

```bash
bash -c "$(curl --fail --show-error --silent --location https://raw.githubusercontent.com/zdharma-continuum/zinit/HEAD/scripts/install.sh)"
zinit ice wait lucid                           load AFTER first prompt
zinit light zsh-users/zsh-autosuggestions
zinit light zsh-users/zsh-syntax-highlighting
```

## Variables

Bash-style assignment with no spaces around =.

```bash
NAME=stevie                                    BROKEN if you write NAME = stevie (zsh tries to RUN "NAME")
greeting="Hello, $NAME"
echo $greeting                                 Hello, stevie
echo ${greeting}                               same; braces avoid ambiguity in $varSomething
```

Read-only and typed with typeset.

```bash
typeset -r PI=3.14159                          readonly; PI=2 errors with "read-only variable: PI"
typeset -i count=0                             integer; count=count+1 works without (( ))
typeset -i 16 hex=0xff                         display in base 16
typeset -F 4 ratio=0.3333                      float, 4 decimals
typeset -a names=(alice bob carol)             indexed array
typeset -A colors=(red ff0000 green 00ff00)    associative array
typeset -U path                                "unique" — dedupes path automatically
```

Locals and privates inside functions.

```bash
greet() {
    local name=$1                              scoped to this function
    typeset -L 8 padded=$name                  left-padded to 8 chars
    print -- "Hi, $padded"
}
```

Export to subprocesses.

```bash
export EDITOR=vim
typeset -gx EDITOR=vim                         long form: -g global -x exported
unset NAME                                     remove
```

Special parameters worth knowing.

```bash
$$        PID of shell
$!        PID of last backgrounded job
$?        exit status of last command (0 = success)
$#        argc — number of positional parameters
$0        script name (or shell name in interactive)
$@        all positional params, properly quoted as separate words ("$@" only)
$*        all positional params, joined by IFS first char
$_        last argument of last command
$PIPESTATUS  array of exit codes from each command in the most recent pipeline
$pipestatus  same, lowercase variant (zsh prefers lower)
```

## Quoting

Word splitting and glob expansion only happen on UNQUOTED expansions. Zsh DIFFERS from bash here.

```bash
files="a.txt b.txt c.txt"
for f in $files; do echo "<$f>"; done          zsh: <a.txt b.txt c.txt>  ONE iteration
for f in $files; do echo "<$f>"; done          bash: <a.txt> <b.txt> <c.txt>  THREE iterations
for f in ${=files}; do echo "<$f>"; done       zsh, force splitting via the (=) flag
```

Quote types.

```bash
'literal'                                      no expansion at all (NOT EVEN backslash)
"double $VAR"                                  parameter, command, arithmetic expansion
$'ANSI-C \n \x1b[31mred\x1b[0m'                C-style escapes: \n \t \xHH \uHHHH
$"localized"                                   gettext localization (rare)
```

Critical "$@" expansion subtlety.

```bash
"$@"          BROKEN often misunderstood — actually expands to "$1" "$2" "$3" (separate words)
"$*"          one word with $1$IFS$2$IFS$3
$@            unquoted: still SEPARATE WORDS in zsh (in bash, splits each one further!)
```

In zsh you almost always still want "$@" anyway — defensive habit, identical bash behavior.

```bash
my_wrapper() {
    git "$@"                                   forwards all args including ones with spaces
}
```

## Parameter Expansion — Standard

Defaults, errors, alternates.

```bash
${var:-default}        if var unset/empty: use "default" (var unchanged)
${var:=default}        if unset/empty: ASSIGN "default" to var, then use it
${var:?error msg}      if unset/empty: print "error msg" and exit
${var:+alt}            if SET and non-empty: use "alt", else empty (inverse of :-)
```

Length and substring.

```bash
${#var}                length in characters
${#array[@]}           number of elements
${var:offset:length}   substring (0-indexed for substring offset, even though arrays are 1-indexed)
${var:5}               from byte 5 to end
${var: -3}             last 3 chars (note the SPACE before the negative number)
```

Pattern strip.

```bash
${var#prefix}          remove SHORTEST prefix matching glob
${var##prefix}         remove LONGEST prefix
${var%suffix}          remove SHORTEST suffix
${var%%suffix}         remove LONGEST suffix

path=/usr/local/bin/zsh
echo ${path##*/}                               zsh           (basename)
echo ${path%/*}                                /usr/local/bin (dirname)
```

Pattern substitution.

```bash
${var/old/new}         replace FIRST match
${var//old/new}        replace ALL matches
${var/#old/new}        replace if old matches at START
${var/%old/new}        replace if old matches at END
${var:gs/old/new/}     all-substitute (zsh "gs" form, ksh-compat)
```

Case modification.

```bash
${var:u}               UPPERCASE entire value
${var:l}               lowercase entire value
${(U)var}              same, flag form
${(L)var}              same, lowercase flag
${(C)var}              Capitalize words
```

## Parameter Expansion — Zsh-Specific

Path modifiers (the colon-letter forms; ZSH SUPERPOWER).

```bash
file=/home/stevie/projects/cheat_sheet/README.md
echo ${file:h}                                 /home/stevie/projects/cheat_sheet  (head/dirname)
echo ${file:t}                                 README.md                         (tail/basename)
echo ${file:r}                                 /home/.../README                  (root, no ext)
echo ${file:e}                                 md                                (extension)
echo ${file:h:t}                               cheat_sheet                       (chained)
echo ${file:A}                                 absolute, resolving symlinks
echo ${file:a}                                 absolute, NOT resolving symlinks
```

Array slicing (1-indexed).

```bash
arr=(zero one two three four)
echo ${arr[1]}                                 zero — first element (NOT zero index!)
echo ${arr[-1]}                                four — last
echo ${arr[2,4]}                               one two three (inclusive range)
echo ${arr[2,-1]}                              one two three four (to end)
```

Splitting and joining (the (s) and (j) flags).

```bash
csv="alpha,beta,gamma"
parts=("${(s:,:)csv}")                         split on comma into array
echo ${parts[1]} ${parts[2]} ${parts[3]}       alpha beta gamma

words=(one two three)
joined=${(j:,:)words}                          one,two,three
joined_pipe=${(j:|:)words}                     one|two|three
```

Common parameter expansion flags (the (X) prefix forms).

```bash
${(P)name}             "indirect" — value of variable whose name is in $name
${(e)var}              expand again (re-evaluate $vars inside the value)
${(q)var}              quote value safe for re-eval (single quote)
${(qq)var}             quote double
${(qqq)var}            quote dollar-single
${(Q)var}              REMOVE one level of quoting
${(k)assoc}            KEYS of associative array
${(v)assoc}            VALUES of associative array
${(kv)assoc}           interleaved keys+values (good for re-creating)
${(i)var}              case-insensitive sort
${(o)var}              sort ascending
${(O)var}              sort descending
${(u)var}              unique (dedupe, preserve order)
${(z)cmd}              tokenize like the shell would (great for parsing)
${(L)var}              lowercase
${(U)var}              uppercase
${(C)var}              Capitalize
${(@)var}              array context (force one-elem-per-word in "")
${(f)var}              split on newlines (great for command output)
${(0)var}              split on null byte
```

Multiple flags compose.

```bash
files=$(find . -name "*.go" -print0)
list=("${(0)files}")                           split a NUL-delimited stream into array
sorted=("${(io)list[@]}")                      case-insensitive sorted
```

## Globbing — Basic

The same patterns bash uses, with differences in error behavior.

```bash
*                ANY chars except /
?                exactly one char
[abc]            one of a, b, c
[a-z]            range
[^abc] [!abc]    negation
{a,b,c}          brace expansion (NOT a glob — happens before globbing)
{1..10}          numeric range
{01..10}         zero-padded range
```

KEY DIFFERENCE: zsh defaults to NOMATCH error if a glob matches nothing.

```bash
ls *.xyz                                       BROKEN if no .xyz files: "zsh: no matches found: *.xyz"
ls *.xyz(N)                                    FIXED: (N) = nullglob qualifier (silent empty expand)
setopt NULL_GLOB; ls *.xyz                     FIXED: globally; missing globs vanish
setopt NO_NOMATCH; ls *.xyz                    FIXED: pass literal "*.xyz" through (bash-style)
```

Brace expansion fires before globbing — useful for repeating prefixes.

```bash
mv old/{config,data,logs} new/                 mv old/config old/data old/logs new/
echo file{1..3}.txt                            file1.txt file2.txt file3.txt
echo {a..z}                                    a b c ... z
echo {01..05}                                  01 02 03 04 05  (zero-padded)
```

## Globbing — Extended

Activate with setopt EXTENDED_GLOB. Then unlock the superpowers.

```bash
setopt EXTENDED_GLOB

ls ^*.log                                      negation: ALL files EXCEPT .log
ls *.txt~README.txt                            "minus": .txt files except README.txt
ls (foo|bar)*.sh                               alternation: starts with foo or bar
ls *(#i)readme*                                case-insensitive flag (#i)
ls *.go(#qN)                                   query-form with N qualifier (no-match-empty)
```

Repetition and exact counts.

```bash
ls ab#c                                        a, ab, abc, abbc... (b zero or more)
ls ab##c                                       abc, abbc, abbbc... (b one or more)
```

Glob qualifiers — append (qualifiers) to a pattern. The big list:

```bash
*(.)             regular files
*(/)             directories
*(@)             symlinks
*(*)             executable
*(=)             socket
*(p)             named pipe (FIFO)
*(b)             block device
*(c)             character device
*(.r-)           regular files NOT readable by owner
*(r) (w) (x)     readable / writable / executable by owner
*(R) (W) (X)     by world (others)
*(U)             owned by current user
*(G)             owned by current user's primary group
*(u:user:)       owned by named user
*(g:group:)      owned by named group
*(D)             include dotfiles in match
*(N)             nullglob — empty expansion if no match (no error)
*(L+100)         size > 100 bytes (Lk = KB, Lm = MB, Lg = GB)
*(L-50)          size < 50 bytes
*(m+7)           modified > 7 DAYS ago (M = months, w = weeks, h = hours, m = minutes, s = seconds)
*(mh-1)          modified within last 1 hour
*(om)            sort by modification time, NEWEST first
*(Om)            sort by modification time, OLDEST first
*(oc)            sort by inode-change time
*(oa)            sort by access time
*(on)            sort by name (default but explicit)
*(oL)            sort by size, smallest first
*(OL)            sort by size, largest first
*(od)            sort by directory depth
*([1,5])         first 5 results
*([2])           the 2nd result only
```

Combine fearlessly.

```bash
ls **/*.log(.mw+4Lk+100)                       log files older than 4 weeks AND > 100 KB
ls -la *(.om[1,10])                            10 most recently modified files
print -l **/*(.x)                              all executable regular files in tree
ls *(*) | xargs -n1 file                       run `file` on every executable
```

## Recursive Globbing

The /**/ pattern walks the tree. No shopt globstar needed — built in.

```bash
ls **/*.go                                     all .go files in tree (excludes dotdirs by default)
ls -la **/*                                     enumerate everything
ls **/*(.)                                      all regular files, recursive
ls **/*(.x)                                     all executable files, recursive
ls **/.git(/N)                                  every .git directory (N = nullglob in case none)
print -l **/node_modules(/N) | head             find every node_modules quickly
rm -f **/*.tmp(N)                               nuke all *.tmp safely (N = no error if none)
```

Symlink handling defaults to NOT following; use ***/ to follow.

```bash
ls **/*                                        does NOT descend into symlinked dirs
ls ***/*                                        does descend (rare, dangerous if cycles)
```

## Arrays — Indexed

Arrays are 1-INDEXED in zsh. This is the #1 porting bug from bash.

```bash
arr=(alpha beta gamma delta)
echo $arr                                      alpha beta gamma delta (joined with space)
echo $arr[1]                                   alpha   (NOT zero — KEY DIFF FROM BASH)
echo $arr[-1]                                  delta   (last)
echo "${arr[@]}"                               every elem as separate quoted word
echo "${arr[*]}"                               every elem joined by IFS first char
echo $#arr                                     4       (length)
echo ${#arr[@]}                                4       (length, bash-compatible form)
```

Mutate.

```bash
arr+=(epsilon)                                 append
arr=(prepend $arr)                              prepend (rebuild)
arr[5]=epsilon                                  assign by index
arr[2]=()                                       delete element 2 (replace with empty list)
unset 'arr[2]'                                 also delete element 2 (note quoting)
arr=(${arr:#beta})                              remove all "beta" entries
arr=(${(u)arr})                                 dedupe in place
```

Slice.

```bash
arr[2,3]                                       beta gamma (RANGE; both inclusive)
arr[2,-1]                                      beta gamma delta (to end)
arr[1,-2]                                      alpha beta gamma (drop last)
"${arr[@]:0:2}"                                bash-compatible slice (alpha beta)
```

Iterate cleanly.

```bash
for x in $arr; do print -- "$x"; done           good in zsh — no word-splitting needed
for x in "${arr[@]}"; do print -- "$x"; done   bash-compatible form
```

## Arrays — Associative

Hash maps. Declare with typeset -A.

```bash
typeset -A user
user[name]=stevie
user[shell]=zsh
user[host]=workstation

echo ${user[name]}                              stevie
echo ${(k)user}                                 keys: name shell host
echo ${(v)user}                                 values: stevie zsh workstation
echo ${(kv)user}                                interleaved: name stevie shell zsh host workstation
echo ${#user}                                   3 (key count)
```

Build from interleaved list.

```bash
typeset -A colors=(red ff0000 green 00ff00 blue 0000ff)
for k v in ${(kv)colors}; do
    print "$k -> $v"
done
```

Iterate keys and values.

```bash
for k in ${(k)user}; do print "$k = ${user[$k]}"; done
for k v in ${(@kv)user}; do print "$k = $v"; done    @ flag forces array context
```

Membership and deletion.

```bash
(( ${+user[name]} ))                           1 if key exists, 0 otherwise
unset 'user[name]'                             delete a key (mind the quoting)
user=()                                        empty the map
```

## Conditionals

Bash-style if works fine.

```bash
if [[ -f $file ]]; then
    print "exists"
elif [[ -d $file ]]; then
    print "directory"
else
    print "missing"
fi
```

[[ ... ]] is the modern test (no globbing/word-splitting inside; safe with empty vars).

```bash
[[ -f $f ]]            regular file
[[ -d $d ]]            directory
[[ -L $l ]]            symlink
[[ -e $p ]]            exists (any type)
[[ -r -w -x ]]         readable / writable / executable
[[ -s $f ]]            non-empty file
[[ -z $s ]]            string empty
[[ -n $s ]]            string non-empty
[[ $a == $b ]]         string equal (== is GLOB MATCH if pattern on right)
[[ $a = $b ]]          same as ==
[[ $a != $b ]]         not equal
[[ $a < $b ]]          lexicographic less than
[[ $n -eq 5 ]]         integer equal (numeric comparisons use -eq -ne -lt -le -gt -ge)
[[ $n -lt 10 ]]
[[ $a == foo* ]]       glob match (== on right is a GLOB pattern)
[[ $a =~ ^[0-9]+$ ]]   regex match; captures land in $MATCH and $match[1..]
[[ $a -ot $b ]]        $a is older than $b (mtime)
[[ $a -nt $b ]]        $a is newer than $b
```

Logical operators inside [[ ]].

```bash
[[ -f $f && -r $f ]]   AND
[[ $x = a || $x = b ]] OR
[[ ! -d $f ]]          NOT
```

Arithmetic conditional.

```bash
(( count > 0 ))        true if positive
(( x = y + 1 ))        evaluate AND assign (truthy if non-zero result)
(( x++ ))              post-increment
((  ))                 ZERO-LENGTH expression — BROKEN, errors "(:0: bad math expression"
```

## Loops

For loop, multiple syntaxes.

```bash
for i in 1 2 3; do print $i; done
for i in {1..10}; do print $i; done
for i ({1..10}) print $i                         compact zsh syntax
foreach i (1 2 3)                                csh-style alternative
    print $i
end                                              foreach...end uses 'end' not 'done'

for arg; do print $arg; done                     implicit list = "$@"
for f in **/*.md; do print $f; done              glob in for
for f in **/*.md(.); do wc -l $f; done            with qualifier (regular files only)
```

C-style for.

```bash
for ((i = 0; i < 10; i++)); do
    print $i
done
```

While, until, repeat.

```bash
while read -r line; do
    print "got: $line"
done < input.txt

until [[ -f /tmp/ready ]]; do sleep 1; done

repeat 5 print "hello"                           run command 5 times (zsh-only construct)
repeat 3 do print "hi"; print "bye"; done
```

select for menu prompts.

```bash
select fruit in apple banana cherry; do
    print "you picked $fruit"
    break
done
```

Break and continue.

```bash
for i (1 2 3 4 5) {
    (( i == 3 )) && continue
    (( i == 5 )) && break
    print $i
}                                                 brace form: { ... }
```

## Functions

Two syntaxes; pick one and stick.

```bash
function greet {
    print "hello, $1"
}

greet() { print "hello, $1"; }                   POSIX form, also valid

function greet() { ... }                          combined (works but redundant)
```

Locals.

```bash
mything() {
    local name=$1                                  scope to function
    local -a items                                  local array
    local -A map                                    local assoc
    integer count=0                                 typed local; same as local -i
    items=(a b c)
    print "name=$name items=$items count=$count"
}
```

Caller info.

```bash
who_called_me() {
    print "caller: ${funcstack[2]}"               funcstack[1] is current; [2] is caller
    print "files:  ${funcfiletrace[@]}"
    print "lines:  ${funcsourcetrace[@]}"
}
```

Return values.

```bash
add() {
    local sum=$(( $1 + $2 ))
    print $sum                                     STDOUT: capture with $(add 1 2)
    return 0                                       exit STATUS only (0-255), NOT the result
}

result=$(add 3 4)
print $result                                      7
```

Anonymous functions (zsh-only) — useful for one-shot scoped blocks.

```bash
() { local foo=bar; print $foo } arg1 arg2       declare and immediately run
```

Autoload pattern (lazy load functions from FPATH).

```bash
fpath=(~/.zfunctions $fpath)
mkdir -p ~/.zfunctions
print 'print "hi from autoloaded"' > ~/.zfunctions/myfunc
autoload -Uz myfunc                               -U: no aliases expanded; -z: zsh-style
myfunc                                             auto-loaded on first call
```

## Aliases

Three flavors: regular, global, suffix.

```bash
alias ll='ls -lah'                                regular: replaces command at start of line
alias g='git'
alias gst='git status -sb'

alias -g G='| grep'                               global: replaces ANYWHERE in line
ls -la G zsh                                       expands to: ls -la | grep zsh
alias -g L='| less'
alias -g NUL='2>/dev/null'

alias -s md=glow                                   suffix: extension auto-runs command
alias -s py=python3                                ./script.py runs python3 ./script.py
./README.md                                        runs: glow ./README.md
```

Manage aliases.

```bash
alias                                              list all
alias g                                            show one
unalias g                                          remove
alias -L                                           list in re-executable form
\g                                                  escape (run "g" command, not alias)
command g                                           same — bypass alias
```

## History

Configure size, file, and behavior.

```bash
HISTFILE=$HOME/.zsh_history
HISTSIZE=50000                                    in-memory size
SAVEHIST=50000                                    on-disk size

setopt EXTENDED_HISTORY                           store timestamp + duration: ": 1700000000:5;cmd"
setopt SHARE_HISTORY                              all sessions share one history file (live)
setopt INC_APPEND_HISTORY                         append on each command (instead of on exit)
setopt HIST_IGNORE_DUPS                           don't store consecutive duplicates
setopt HIST_IGNORE_ALL_DUPS                       drop older duplicate when adding new
setopt HIST_SAVE_NO_DUPS                          drop dupes when writing the file
setopt HIST_FIND_NO_DUPS                          skip dupes during search
setopt HIST_IGNORE_SPACE                          don't store commands that start with space
setopt HIST_REDUCE_BLANKS                         collapse multiple spaces to one
setopt HIST_VERIFY                                show !! / !N expansion before running
setopt HIST_NO_STORE                              don't store history-related commands themselves
```

History expansion (csh-style).

```bash
!!                  previous command
!$                  last word of previous command
!^                  first word of previous command
!*                  all words of previous command
!-2                 two commands back
!42                 command #42
!ssh                most recent line starting with "ssh"
!?fish              most recent line containing "fish"
^old^new            replace "old" with "new" in previous command, run it
```

Inspect with fc.

```bash
fc -l                                             list recent history (default 16 lines)
fc -l -100                                        list last 100
fc -l -50 -1                                      list 50..most recent
fc -l ssh                                         starting from most recent "ssh*"
fc -e vim 100                                     edit history line 100 in vim, then run
fc -e -                                            re-run last command
fc -li -10                                         list with timestamps
```

Substring search hotkeys (with plugin).

```bash
bindkey '^[[A' history-substring-search-up        up arrow searches matching history
bindkey '^[[B' history-substring-search-down      down arrow
```

## Tab Completion

The crown jewel of zsh. Powered by the compsys system.

```bash
autoload -Uz compinit
compinit                                          rebuilds completion dump (~/.zcompdump)
compinit -C                                       skip security audit (faster, less safe)
```

Configure via zstyle (the canonical settings).

```bash
zstyle ':completion:*' menu select                arrow-key menu navigation
zstyle ':completion:*' matcher-list \
    'm:{a-zA-Z}={A-Za-z}' \
    'r:|[._-]=* r:|=*' \
    'l:|=* r:|=*'                                  case-insensitive + partial-word + fuzzy
zstyle ':completion:*' list-colors ${(s.:.)LS_COLORS}
zstyle ':completion:*' completer _expand _complete _correct _approximate
zstyle ':completion:*:approximate:*' max-errors 1 numeric
zstyle ':completion:*' format '%B%F{yellow}%d%f%b'
zstyle ':completion:*' group-name ''
zstyle ':completion:*:descriptions' format '%B-- %d --%b'
zstyle ':completion:*:warnings'     format 'no matches: %B%d%b'
zstyle ':completion:*' use-cache on
zstyle ':completion:*' cache-path ~/.zcache
```

Speed up first-day-of-month rebuild.

```bash
autoload -Uz compinit
() {
  local zcd=${ZDOTDIR:-$HOME}/.zcompdump
  if [[ -n $zcd(#qN.mh+24) ]]; then              older than 24 hours
      compinit
  else
      compinit -C
  fi
}
```

Trigger completion manually with Tab. Special keys:

```bash
Tab                first / next match
Shift-Tab          previous match
^X^L               complete a long line (history)
^X?                show possible completions
^Xh                show all "help" for the current spot
```

## Prompt — Basics

Build with %-escapes.

```bash
PROMPT='%F{cyan}%n%f@%F{green}%m%f %F{blue}%~%f %# '
```

Common escapes:

```bash
%n           username
%m           short hostname (up to first .)
%M           full hostname
%~           PWD with $HOME → ~ and named dirs (~proj)
%/           PWD without ~ replacement
%d           same as %/
%c %1d       last component of PWD
%2~          last TWO components
%h %!         history number
%?           exit status of last command
%#           "%" for user, "#" for root
%T           24-hour time HH:MM
%t %@         12-hour time
%D           date YY-MM-DD
%D{format}    custom strftime: %D{%H:%M:%S}
%F{color}    foreground color start; %f = end
%K{color}    background color
%B %b        bold start / end
%U %u        underline
%S %s        standout (reverse)
%(?.✓.✗)     conditional: shows ✓ if last cmd succeeded, ✗ if failed
```

Color names: black red green yellow blue magenta cyan white. Or 0-255 numbers.

```bash
PROMPT='%F{208}%n%f%F{240}@%f%F{75}%m%f %F{green}%~%f %# '
```

Multiline prompts use $'\n' or %nl.

```bash
PROMPT=$'%F{green}%n@%m %F{blue}%~%f\n%# '       newline before $ on a fresh line
```

## Prompt — Themes / Right Prompt

Right side prompt.

```bash
RPROMPT='%F{240}[%D{%H:%M}]%f'                     a clock on the right
RPROMPT='${vcs_info_msg_0_}'                       git branch
```

vcs_info — built-in version-control awareness (no plugin needed).

```bash
autoload -Uz vcs_info
zstyle ':vcs_info:*' enable git hg svn
zstyle ':vcs_info:git:*' formats       ' %F{yellow}(%b)%f'
zstyle ':vcs_info:git:*' actionformats ' %F{red}(%b|%a)%f'
zstyle ':vcs_info:git:*' check-for-changes true
zstyle ':vcs_info:git:*' stagedstr     '+'
zstyle ':vcs_info:git:*' unstagedstr   '*'
zstyle ':vcs_info:git:*' formats       ' %F{yellow}(%b%c%u)%f'
precmd() { vcs_info }
setopt PROMPT_SUBST                                  REQUIRED to evaluate ${vcs_info_msg_0_}
RPROMPT='${vcs_info_msg_0_}'
```

Powerlevel10k — fastest popular theme (instant prompt).

```bash
git clone --depth=1 https://github.com/romkatv/powerlevel10k.git \
  ${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}/themes/powerlevel10k
ZSH_THEME="powerlevel10k/powerlevel10k"
p10k configure                                       interactive setup wizard
```

pure — minimalist alternative.

```bash
fpath+=($HOME/.zsh/pure)
autoload -U promptinit; promptinit
prompt pure
```

starship — third-party, cross-shell.

```bash
brew install starship
eval "$(starship init zsh)"                           goes in .zshrc
```

## ZLE — Zsh Line Editor

ZLE is the line editor with bindable widgets. Switch keymaps, define widgets, rebind keys.

```bash
bindkey -e                                            emacs keymap (default)
bindkey -v                                            vi keymap
bindkey -l                                            list all keymaps
bindkey -lL main                                       which keymap is "main" aliased to
bindkey -A vicmd main                                  promote vicmd as main keymap
```

Useful widgets to remember.

```bash
backward-kill-word              kill back one word
forward-word                    move forward
beginning-of-line               move to col 0
clear-screen                    Ctrl-L
push-line-or-edit               Ctrl-Q (save line, return after typing more)
expand-or-complete              Tab default
history-incremental-search-backward   Ctrl-R
history-search-backward         arrow up after starting line
quoted-insert                   Ctrl-V (literal next char)
self-insert                     just type
copy-prev-shell-word            ESC-., yank last word
```

Bind keys (use bindkey).

```bash
bindkey '^R' history-incremental-search-backward     Ctrl-R
bindkey '^[[A' up-line-or-beginning-search           up-arrow with substring search
bindkey '^[[B' down-line-or-beginning-search         down-arrow
bindkey ' ' magic-space                              space expands history (e.g. !!)
bindkey -M viins 'jk' vi-cmd-mode                    map jk in viins to ESC equivalent
```

Define your own widget.

```bash
my-prepend-sudo() {
    LBUFFER="sudo $LBUFFER"
}
zle -N my-prepend-sudo                                register as a widget
bindkey '^X^S' my-prepend-sudo                        Ctrl-X Ctrl-S to prepend sudo
```

Find the keycode for any key.

```bash
cat -v                                                press the key, see escape sequence
^[[A         up arrow            (escape "[A")
^[[B         down arrow
^?           backspace
^[[Z         shift-tab
```

## Options

Toggle options with setopt and unsetopt. Names are case-insensitive and accept underscores.

```bash
setopt AUTO_CD                                        cd by typing dir name only
setopt AUTO_PUSHD PUSHD_IGNORE_DUPS PUSHD_SILENT      navigation magic
setopt CDABLE_VARS                                    cd $var if var contains a path
setopt EXTENDED_GLOB GLOB_DOTS NULL_GLOB              globbing power-ups
setopt NUMERIC_GLOB_SORT                              "file2.txt" before "file10.txt"
setopt NO_BEEP                                        silence terminal bell
setopt CORRECT CORRECT_ALL                            spelling correction
setopt INTERACTIVE_COMMENTS                           allow # comments in interactive shell
setopt PROMPT_SUBST                                   evaluate ${var} inside PROMPT
setopt RC_QUOTES                                      'don''t' = "don't"
setopt MULTIOS                                        multiple redirections (see Redirection)
setopt PIPE_FAIL                                      pipeline status = first nonzero (like bash set -o pipefail)
setopt NOTIFY                                         immediate notification of bg job exit
setopt NO_HUP                                         don't HUP background jobs on exit
setopt LONG_LIST_JOBS                                 verbose jobs listing
setopt HIST_VERIFY                                    show ! expansion before running
setopt SHARE_HISTORY EXTENDED_HISTORY                 history goodies
setopt HIST_IGNORE_DUPS HIST_REDUCE_BLANKS HIST_IGNORE_SPACE
setopt INC_APPEND_HISTORY                             append immediately, not on exit
setopt MENU_COMPLETE                                   tab inserts the FIRST match (off by default)
setopt AUTO_MENU                                       second tab cycles
setopt COMPLETE_IN_WORD                               complete at cursor (not just end of word)
setopt ALWAYS_TO_END                                   move cursor to end after completion
```

See current options.

```bash
setopt                                                 list set options
unsetopt                                                list options NOT set
print -l ${(k)options}                                 every option name
print $options[autocd]                                 "on" or "off"
```

The "good defaults" set most users want:

```bash
setopt AUTO_CD AUTO_PUSHD PUSHD_IGNORE_DUPS PUSHD_SILENT
setopt EXTENDED_GLOB GLOB_DOTS NULL_GLOB NUMERIC_GLOB_SORT
setopt SHARE_HISTORY EXTENDED_HISTORY HIST_VERIFY
setopt HIST_IGNORE_DUPS HIST_IGNORE_SPACE HIST_REDUCE_BLANKS
setopt INTERACTIVE_COMMENTS PROMPT_SUBST
setopt NO_BEEP NO_HUP
```

## Hooks

Zsh fires hooks at lifecycle moments. Register safely with add-zsh-hook.

```bash
chpwd          when PWD changes (cd, pushd, popd)
precmd         before each new prompt
preexec        after a command line is read but before it runs
periodic       every $PERIOD seconds (set PERIOD=300)
zshaddhistory  when a command is added to history (return non-zero to suppress)
zshexit        when the shell exits
```

Register multiple hooks (don't overwrite the function!).

```bash
autoload -Uz add-zsh-hook

show_dir_contents() { ls -la }
add-zsh-hook chpwd show_dir_contents

start_timer()  { _CMD_START=$EPOCHREALTIME }
end_timer()    {
    local elapsed=$(( EPOCHREALTIME - _CMD_START ))
    (( elapsed > 5 )) && print "took ${elapsed}s"
}
add-zsh-hook preexec start_timer
add-zsh-hook precmd  end_timer
```

UNREGISTER a hook.

```bash
add-zsh-hook -d chpwd show_dir_contents               -d for delete
```

## Process Substitution & Pipes

Process substitution puts a process where a file should be.

```bash
diff <(sort file1) <(sort file2)                       compare sorted streams
comm -12 <(sort a) <(sort b)                            common lines
wc -l <(grep error syslog) <(grep warn syslog)         per-stream line counts
```

The =( ... ) form returns a TEMP FILE path (cleaned up by zsh on exit).

```bash
vim =(curl -s https://example.com/config)              edit URL contents (then save? no — it's read-only ref to tmp)
ls -la =(echo hello)                                    show the temp filename
```

Pipe both stdout AND stderr.

```bash
cmd 2>&1 | grep ERROR                                   bash-compatible
cmd |& grep ERROR                                        zsh shorthand (also bash 4+)
```

Coprocess.

```bash
coproc bc                                                start bc in background, fd 0/1
print "1+1" >&p                                          send to coproc input
read -p answer                                            read from coproc output
```

## Redirection — MULTIOS

KEY DIFFERENCE FROM BASH: zsh by default writes to MULTIPLE targets in one redirect, no tee.

```bash
setopt MULTIOS                                            usually on by default

date > now.txt > also-now.txt                             writes to BOTH files
echo error >&2 > stderr.log                                stderr goes to console AND stderr.log

cat < file1 < file2                                        concatenates inputs (reads file1 then file2)
cat < <(echo hi) < <(echo bye)                             merge process-substituted inputs
```

Disable MULTIOS to get bash-style "last redirect wins."

```bash
unsetopt MULTIOS
date > a > b                                                only b gets the output (bash-style)
```

Append.

```bash
echo hi >> log                                              standard append
echo hi >>! log                                             "force append" — even if NOCLOBBER is set
echo hi >| log                                              "force overwrite" — even if NOCLOBBER set
```

NOCLOBBER protection.

```bash
setopt NO_CLOBBER                                          > on existing file errors with "file exists"
echo hi > existing                                          BROKEN: "zsh: file exists: existing"
echo hi >| existing                                         FIXED: pipe-bar bypasses NOCLOBBER
```

## Reading Input

read is a builtin with rich flags.

```bash
read -r line                                                always pass -r (no backslash escapes)
read -r line < file                                         from file
read -r -p "Name: " name                                     with prompt
read -r -t 5 line                                            timeout: 5 seconds, fail if no input
read -r -k 1 char                                            read just 1 keystroke (no Enter needed)
read -r -s pass                                              silent read (passwords)
read -r -A array                                             read whole line as ARRAY (split on IFS)
read -r -d '' content                                        read until NUL (slurp whole stdin)
read -r line0 line1 line2                                    bind first 3 fields to vars
```

Slurp from process.

```bash
output=("${(f)$(some-command)}")                           split command output on lines into array
content="$(<file.txt)"                                      read FILE into var (zsh shortcut)
content=$(<file.txt)                                        same, no quotes if single token
```

vared — edit a variable in place.

```bash
vared name                                                  pop up an editor (line) for $name
vared -p "edit: " text                                       with prompt
```

## Functions Library

Lazy-load functions from FPATH at first use.

```bash
mkdir -p ~/.zfunctions
fpath=(~/.zfunctions $fpath)                                add to search path FIRST

cat > ~/.zfunctions/upper << 'EOF'
upper() { print -- ${(U)1} }
EOF
chmod 644 ~/.zfunctions/upper

autoload -Uz upper                                           mark for lazy load
upper hello                                                   first call: file is read & function defined
                                                              HELLO
```

The -U flag suppresses alias expansion during loading (highly recommended). The -z flag selects zsh-style autoload (the default these days).

Run a function directly without typing its name (script-equivalent).

```bash
zsh -c 'autoload -Uz upper; upper hi'
```

Pre-compile autoloaded files for speed (.zwc = zsh word code).

```bash
zcompile ~/.zshrc                                            creates ~/.zshrc.zwc; sourced before .zshrc
zcompile -U ~/.zfunctions/*                                   precompile all funcs
```

## Job Control

Backgrounding, listing, foregrounding.

```bash
sleep 100 &                                                   run in background; prints job # and PID
jobs                                                           [1]  + running    sleep 100
jobs -l                                                        with PIDs
fg                                                              foreground last bg job
fg %1                                                           by job number
fg %sleep                                                       by command name prefix
fg %?100                                                        by command substring
bg %1                                                            resume stopped job in background
kill %1                                                          send TERM to job 1
kill -9 %1                                                       SIGKILL
disown %1                                                        detach from shell (won't get HUP at exit)
disown -h %1                                                     mark to ignore HUP without removing from job table
nohup long-running &                                              ignore HUP, redirect output to nohup.out
```

Suspend with Ctrl-Z; resume with fg/bg.

Built-in setopt for job behavior.

```bash
setopt MONITOR                                                  job control on (default for interactive)
setopt NOTIFY                                                   immediate exit notification
setopt NO_HUP                                                   don't kill background jobs on shell exit
setopt CHECK_JOBS                                                warn at exit if jobs are running
```

## Common Builtins

zsh has more powerful primitives than POSIX sh.

```bash
print            zsh's enhanced echo. ALWAYS prefer over echo.
printf            standard printf
echo              works but quirky; print/printf is better
type / where / whence    file/builtin/function lookup
builtin <name>    force run the builtin (skip alias/function)
command <name>    force run the external (skip function/alias)
hash              show / set hash table of commands
hash -d name=path  named directory: cd ~name later
hash -r           rebuild command hash table
hash -f           rebuild from PATH
alias / unalias    add / remove alias
fc                fix command — list/edit history
eval              parse and run a string as a command
exec              replace shell with command (no fork)
exit / return    leave shell / function with status
shift            drop $1; rest renumber
set              shell options + positional params
setopt / unsetopt    zsh options
source / .       run a file in current shell
typeset / declare    declare typed vars
local            local-scoped var
readonly         readonly var
true / false     exit 0 / 1
trap             handle signals
times            user/system time totals
ulimit           resource limits
umask            file creation mask
wait             wait for backgrounded job
```

print is significantly richer than echo.

```bash
print hello                           hello
print -l one two three                each on its own line (-l = list)
print -n no-newline                   suppress trailing newline (-n)
print -r 'raw \n no escape'            raw mode (-r); no escape interpretation
print -P '%F{red}red%f text'           prompt expansion (-P) in print output
print -- "$var"                        end-of-options before user data (good habit)
print -s 'cmd'                         add line to HISTORY without running
print -z 'edit me'                      put line into ZLE buffer (next prompt)
print -u 2 'to stderr'                  to FD 2
```

whence/where/type variants.

```bash
whence -v ls                            "ls is /bin/ls"
whence -ca git                           all matches (alias, function, file)
which ls                                  shows alias if any
where ls                                   all matches in PATH plus alias/function
type ls                                    shows builtin/function/alias/external
```

## zmv / zcp / zln

Pattern-based bulk file ops. Load once.

```bash
autoload -Uz zmv zcp zln
```

zmv basics.

```bash
zmv '(*).txt' '$1.md'                            rename .txt to .md
zmv -n '(*).txt' '$1.md'                         DRY RUN (-n) — show without doing
zmv -W '*.jpeg' '*.jpg'                           wildcard mode (no parens needed)
zmv -v '(*).log' 'archive/$1.log'                  verbose
zmv -i '(*)' '$1.bak'                             interactive: prompt per file
zmv '(**/)(*).JPG' '$1${2:l}.jpg'                 lowercase extension recursively
zmv -f '(*).c' '$1.cpp'                            force overwrite
zmv -s '(*).log' '$1.log.gz'                        symlink instead (zln-style)
```

Numbered captures.

```bash
zmv '([0-9]*)-([a-z]*)' '$2-$1'                  swap numeric and alpha parts
zmv '(*)/(*).log' '$1/old-$2.log'                 in subdirs, prefix with old-
```

## Zsh-specific patterns

The (#qN) qualifier — pack qualifiers into expansion expression.

```bash
files=(*.go(#qN))                                  array of .go files; empty if none (no error)
print -l *.txt(#qN)                                  for-loop with no-match-empty
```

Anonymous functions for scoped blocks.

```bash
() {
    local TMP=$(mktemp -d)
    cd $TMP
    do_work
    cd - > /dev/null
} arg1 arg2                                         args available as $1 $2
```

Read whole file into a variable.

```bash
content=$(<config.toml)                              fast — uses internal read, no /bin/cat
lines=("${(f)$(<config.toml)}")                     into array, split on \n
```

Nice numeric tricks.

```bash
(( x = RANDOM % 100 ))                               0-99
(( x = (RANDOM % (max - min + 1)) + min ))            inclusive range
typeset -F 4 piapprox=$(( 22.0 / 7.0 ))                 4 decimals
typeset -i 16 hex=255                                   prints as 16#ff
```

Iterate over both index and value.

```bash
arr=(a b c d)
for i in {1..${#arr}}; do
    print "$i: ${arr[i]}"
done

for i v in ${(kv)arr}; do                             k = index, v = value (associative-array style)
    print "$i = $v"
done
```

## Completion System

The completion system (compsys) is itself written in zsh. compdef glues completion functions to commands.

```bash
compdef _git mygit                                       use git's completion for "mygit"
compdef _files myhello                                    files completion
compdef '_files -g "*.md"' edit-md                         only .md files
```

Inspect what backs a completion.

```bash
which _ls                                                 shows the completion function file
echo $_comps[git]                                         which compdef function handles git
```

Write your own completion (toy example).

```bash
#compdef hello                                              FIRST LINE: tells compinit which command
_hello() {
    local -a opts
    opts=(
        '-h[show help]'
        '-v[verbose]'
        '--name[name]:string:'
        '*:filename:_files'
    )
    _arguments $opts
}
_hello "$@"                                               must call at end
```

Save to a file in fpath named "_hello", then run compinit.

The _arguments DSL.

```bash
_arguments \
    '-h[help]' \
    '-c[count]:count:(1 2 3)' \
    '-f[file]:file:_files -g "*.md"' \
    '*::cmd:->command'                                     state machine: changes $state on hit
```

State-driven completion (subcommands).

```bash
_mytool() {
    local state
    _arguments -C '1: :->cmds' '*::arg:->args'
    case $state in
        cmds) _values 'commands' build test deploy ;;
        args) [[ $words[2] == build ]] && _files ;;
    esac
}
```

## Plugin Managers

zinit (modern, fast, "turbo mode").

```bash
zinit ice wait lucid                                       wait until first prompt
zinit light zsh-users/zsh-syntax-highlighting

zinit ice wait'1' lucid atinit"zicompinit; zicdreplay"
zinit light zsh-users/zsh-history-substring-search

zinit ice wait lucid src"async.zsh"
zinit light mafredri/zsh-async
```

antidote (simple, declarative bundles).

```bash
brew install antidote
echo 'zsh-users/zsh-autosuggestions' >> ~/.zsh_plugins.txt
echo 'zsh-users/zsh-syntax-highlighting' >> ~/.zsh_plugins.txt
echo 'romkatv/powerlevel10k' >> ~/.zsh_plugins.txt
source $(brew --prefix)/opt/antidote/share/antidote/antidote.zsh
antidote load
```

zplug.

```bash
git clone https://github.com/zplug/zplug ~/.zplug
source ~/.zplug/init.zsh
zplug "zsh-users/zsh-syntax-highlighting"
zplug "zsh-users/zsh-autosuggestions"
zplug "romkatv/powerlevel10k", as:theme, depth:1
zplug load --verbose
```

## Keyboard Bindings

The bindkey command associates a key sequence with a widget.

```bash
bindkey -e                                                 emacs keymap (default)
bindkey -v                                                  vi keymap
bindkey -A vicmd main                                        promote a keymap as main
bindkey -L                                                    list current bindings
bindkey -M emacs                                              list emacs keymap bindings
bindkey -s '^X^P' 'paste this\n'                              string macro: types literal text
```

Common rebinds.

```bash
bindkey '^R' history-incremental-search-backward
bindkey '^[[A' up-line-or-beginning-search
bindkey '^[[B' down-line-or-beginning-search
bindkey '^[[H' beginning-of-line                              Home key
bindkey '^[[F' end-of-line                                    End key
bindkey '^[[3~' delete-char                                    Delete key
bindkey '^[.' insert-last-word                                  ESC-. insert last word
bindkey '^Z' undo                                                ctrl-Z (loses suspend; tradeoff)
bindkey '^I' expand-or-complete                                  Tab
```

Vi-mode ergonomics.

```bash
bindkey -v
export KEYTIMEOUT=1                                              ESC delay (1 = 10ms — instant feel)

bindkey -M viins 'jk' vi-cmd-mode                                jk in INS mode → command mode
bindkey -M vicmd 'H' beginning-of-line
bindkey -M vicmd 'L' end-of-line
bindkey -M vicmd 'u' undo
bindkey -M vicmd '^R' redo
bindkey -M vicmd '/' history-incremental-search-backward
```

Cursor shape per mode (terminal-dependent).

```bash
zle-keymap-select() {
    case $KEYMAP in
        vicmd) print -n '\e[2 q' ;;     block cursor
        viins|main) print -n '\e[6 q' ;;  bar cursor
    esac
}
zle -N zle-keymap-select

zle-line-init() { zle -K viins; print -n '\e[6 q' }
zle -N zle-line-init
```

## Common Error Messages

"command not found" — typo, missing PATH, or alias issue.

```bash
zsh: command not found: pip3
echo $PATH                                                       check PATH includes the right dirs
hash -r                                                          rebuild hash table after install
which pip3                                                       confirm visible
```

"no matches found" — glob without match in default mode.

```bash
ls *.xyz
zsh: no matches found: *.xyz
                                                                   FIX: pick one of these
ls *.xyz(N)                                                       inline nullglob qualifier
ls *.xyz(.N)                                                      regular files only
setopt NULL_GLOB                                                  globally: empty expansion on no match
setopt NO_NOMATCH                                                 globally: pass literal pattern (bash-style)
ls *.xyz 2>/dev/null                                                hide the error (less ideal)
```

"parse error" — usually a quoting or paren/brace mismatch.

```bash
echo "hello $name(world)"
zsh: parse error near `(world)'                                   ()s have meaning unless quoted
echo "hello $name\(world\)"                                        FIXED escape parens
echo 'hello $name(world)'                                          FIXED single-quote
```

"bad math expression" — = at line start, or empty arithmetic.

```bash
echo =1+2
zsh: bad math expression: operand expected at end of string
                                                                   FIX
echo "=1+2"                                                          quote the leading =
echo $((1 + 2))                                                       use $(()) for arithmetic
```

"file exists" — NOCLOBBER blocking overwrite.

```bash
echo hi > exists.txt
zsh: file exists: exists.txt                                        with NOCLOBBER
echo hi >| exists.txt                                                FIX: |  bypass clobber
```

"bad assignment" — equals with spaces.

```bash
NAME = stevie
zsh: command not found: NAME                                          ZSH SAW "NAME" AS CMD WITH ARGS = and stevie
NAME=stevie                                                            FIXED no spaces
```

"argument list too long" — kernel ARG_MAX.

```bash
rm *
zsh: argument list too long: rm
                                                                       FIXES
print -l *(.) | xargs rm                                                pipe through xargs
find . -maxdepth 1 -type f -delete                                       use find
for f in *(.); do rm $f; done                                            iterate
```

## Common Gotchas

Word splitting is OFF by default (DIFF FROM BASH).

```bash
files="a b c"
for f in $files; do print "<$f>"; done                        BROKEN expects 3 iters
                                                                <a b c>     one iter
for f in ${=files}; do print "<$f>"; done                       FIXED (=) flag forces splitting
                                                                <a> <b> <c>
```

Arrays are 1-indexed.

```bash
arr=(a b c)
echo $arr[0]                                                  BROKEN prints empty (no element 0)
echo $arr[1]                                                   FIXED
```

Glob with no match errors out.

```bash
ls *.foo                                                        BROKEN no matches found
ls *.foo(N)                                                     FIXED nullglob qualifier
setopt NULL_GLOB                                                 FIXED globally
```

= at start does math expansion (rare but jarring).

```bash
echo =1+2                                                        BROKEN bad math expression
echo "=1+2"                                                       FIXED quote it
```

(( )) requires non-empty expression.

```bash
((  ))                                                            BROKEN bad math expression
(( 1 ))                                                            FIXED
```

Forgetting $ in (( )) is FINE in zsh — names are auto-evaluated.

```bash
i=5
(( i + 1 ))                                                        works (no $ needed)
(( $i + 1 ))                                                       also works
```

print vs echo.

```bash
echo -e "hello\nworld"                                            sometimes BROKEN if echo doesn't honor -e
print "hello\nworld"                                              FIXED zsh's print honors escapes by default
print -r "raw\nliteral"                                            "raw" mode (no escapes)
```

Filename expansion in echo arguments.

```bash
echo *.txt                                                        BROKEN prints all .txt filenames
echo "*.txt"                                                       FIXED quote
```

The dollar-double-paren in array index — use plain brackets.

```bash
arr=(a b c)
echo ${arr[$((1+1))]}                                              works but verbose
echo $arr[2]                                                       cleaner; arithmetic implicit in subscript
```

## Vim mode + zle

Activate vi mode and tame the ESC delay.

```bash
bindkey -v
export KEYTIMEOUT=1                                                instant ESC (1 = 10ms)
                                                                    DEFAULT: 4 = 400ms = laggy feel
```

Why KEYTIMEOUT matters: ESC starts multi-char sequences (arrow keys send ESC[A). Zsh waits KEYTIMEOUT*10ms for the rest. With KEYTIMEOUT=1 the wait is 10ms.

Add common vi-mode niceties.

```bash
bindkey -M vicmd '^R' history-incremental-search-backward
bindkey -M vicmd 'k' up-line-or-beginning-search
bindkey -M vicmd 'j' down-line-or-beginning-search
bindkey -M vicmd 'H' beginning-of-line
bindkey -M vicmd 'L' end-of-line
bindkey -M viins '^A' beginning-of-line
bindkey -M viins '^E' end-of-line
bindkey -M viins '^R' history-incremental-search-backward
bindkey -M viins '^P' up-line-or-history
bindkey -M viins '^N' down-line-or-history
```

Cursor shape per mode (block in command, bar in insert).

```bash
function zle-keymap-select zle-line-init {
    case $KEYMAP in
        vicmd)      print -n '\e[2 q' ;;                          steady block
        main|viins) print -n '\e[6 q' ;;                          steady bar
    esac
}
zle -N zle-keymap-select
zle -N zle-line-init
print -n '\e[6 q'                                                    initial cursor
```

Show current mode in prompt.

```bash
function zle-keymap-select { zle reset-prompt }
zle -N zle-keymap-select
PROMPT='${${KEYMAP/vicmd/[CMD]}/(main|viins)/[INS]} %~ %# '
```

## Performance

Profile startup.

```bash
zsh -ixv 2>&1 | head -200                                          trace shell init verbosely
zsh -i -c exit                                                       cold benchmark; time it:
time zsh -i -c exit
                                                                       ~0.020s vanilla
                                                                       ~0.300s+ Oh My Zsh
```

zprof — profile per-function time.

```bash
.zshrc:
zmodload zsh/zprof                                                    line 1
                                                                       ... rest of .zshrc ...
                                                                       (interactive prompt, then run)
zprof                                                                  prints time spent per function
zprof -c                                                                clear stats
zprof -m 50                                                              top 50 lines
```

Common wins (target <100ms total).

```bash
1. Don't run compinit -i (skip security audit) — compinit -C
2. Cache compinit result: only rebuild .zcompdump once per day (see Tab Completion)
3. Use zinit ice wait lucid for plugins (defer past first prompt)
4. Avoid `eval "$(starship init zsh)"`-style tools that fork on every prompt
5. Don't source large frameworks — Oh My Zsh adds 200-500ms typical
6. Precompile: `zcompile ~/.zshrc`  (creates .zwc, sourced first)
7. Trim PATH — every PATH entry costs lookup time
8. Defer NVM and similar with lazy-loading shims
```

zsh -ld (login + debug-mode init dumps).

```bash
zsh -lvi 2>&1 | grep -i source                                          which files were sourced?
echo $$ ; zsh -i 2>&1 | grep -E '\+|\-'                                  trace +/- option flips
```

## Compatibility

Emulate other shells inside zsh.

```bash
emulate sh                                                                strict POSIX behavior
emulate ksh                                                                ksh-style
emulate -L sh                                                              local to current function
emulate -R zsh                                                             reset to pure zsh
```

What bashisms WORK in zsh.

```bash
[[ $a == $b ]]                                                            yes (with == as glob match)
$(( ... ))                                                                 yes (full math)
function name { ... }                                                      yes
local var=value                                                            yes
arrays: arr=(a b c)                                                        yes (BUT 1-indexed!)
declare / typeset                                                          yes
&> file (stdout+stderr)                                                    yes
process substitution <(cmd)                                                  yes
```

What bashisms DON'T work or differ.

```bash
arr[0] is element 0 in bash                                               BROKEN; in zsh that's empty
$arr expands to ALL elements in bash                                        DIFFERENT in zsh: only $arr[1]
                                                                            WORKAROUND: $arr[@] both work
unquoted $var word-splits in bash                                           DOES NOT in zsh; use ${=var}
$BASH_REMATCH                                                                BROKEN; zsh uses $MATCH and $match[]
shopt -s globstar                                                            BROKEN; zsh has ** built in
PROMPT_COMMAND                                                                BROKEN; zsh uses precmd hook
read into IFS-split var                                                       different; zsh more strict
trap 'cmd' DEBUG                                                                DIFFERENT signal name conventions
```

Run a bash script in compat mode.

```bash
emulate -L sh; setopt KSH_ARRAYS                                            arrays now 0-indexed in this function
emulate sh -c '. /tmp/old-bash-script.sh'                                    sandbox emulation
```

## Idioms

Robust prompt with vcs_info.

```bash
autoload -Uz vcs_info
zstyle ':vcs_info:git:*' formats       ' (%b)'
zstyle ':vcs_info:git:*' actionformats ' (%b|%a)'
zstyle ':vcs_info:git:*' check-for-changes true
zstyle ':vcs_info:git:*' stagedstr     '+'
zstyle ':vcs_info:git:*' unstagedstr   '*'
zstyle ':vcs_info:git:*' formats       ' (%b%c%u)'
precmd() { vcs_info }
setopt PROMPT_SUBST
PROMPT='%F{green}%n@%m%f:%F{blue}%~%f%F{yellow}${vcs_info_msg_0_}%f %# '
```

Conditional sourcing for portable dotfiles.

```bash
[[ -f ~/.localrc ]] && source ~/.localrc
[[ -f /opt/homebrew/etc/profile.d/z.sh ]] && source /opt/homebrew/etc/profile.d/z.sh
(( $+commands[fzf] )) && source <(fzf --zsh)                                 only if fzf is installed
```

Safe shared history.

```bash
HISTFILE=$HOME/.zsh_history
HISTSIZE=200000
SAVEHIST=200000
setopt SHARE_HISTORY                                                          live cross-session sharing
setopt HIST_VERIFY                                                             never auto-execute ! expansion
setopt HIST_IGNORE_ALL_DUPS                                                     dedupe across history
setopt HIST_REDUCE_BLANKS HIST_IGNORE_SPACE                                      cleanup
```

Directory stack with auto-pushd.

```bash
setopt AUTO_PUSHD                                                                cd auto-pushes
setopt PUSHD_IGNORE_DUPS                                                          no dupes
setopt PUSHD_SILENT                                                                quiet pushd
setopt PUSHD_TO_HOME                                                                pushd alone == pushd ~
DIRSTACKSIZE=20                                                                   bound it
alias d='dirs -v | head -20'                                                       quick inspect
                                                                                    cd -<TAB> shows the stack
```

Named directories (hash -d).

```bash
hash -d proj=~/projects/cheat_sheet                                                creates a named dir
cd ~proj                                                                            jump
print ~proj/sheets                                                                  use anywhere a path goes
```

Last-word-of-command shortcut.

```bash
mkdir /a/long/path
cd !$                                                                                expands to /a/long/path
                                                                                      OR with HIST_VERIFY: shows first
ESC-.                                                                                  insert previous last word interactively
```

Don't bother with ls aliases — use the global trick.

```bash
alias -g L='| less'
alias -g G='| grep -i'
alias -g NUL='2>/dev/null'
alias -g H='| head'
alias -g T='| tail'
                                                                                       command G pattern L
                                                                                       expands to: command | grep -i pattern | less
```

## Migrating from Bash

Most scripts run unchanged after these changes.

```bash
1. Quote $@: "$@" instead of $@ (defensive even though zsh handles this OK)
2. Convert ${arr[0]} indices: bash 0-indexed → zsh 1-indexed
   bash:  ${BASH_SOURCE[0]} → zsh: ${(%):-%x}    (current script)
   bash:  ${arr[0]}        → zsh: ${arr[1]}
3. Replace $BASH_REMATCH:
   bash:  [[ $s =~ ^([0-9]+)$ ]]; echo ${BASH_REMATCH[1]}
   zsh:   [[ $s =~ ^([0-9]+)$ ]]; echo $match[1]
4. Replace shopt:
   bash:  shopt -s globstar           → zsh: enabled by default
   bash:  shopt -s extglob              → zsh: setopt EXTENDED_GLOB
   bash:  shopt -s nullglob             → zsh: setopt NULL_GLOB
5. PROMPT_COMMAND → precmd() {}
6. read -p "prompt" var works the same
7. case statements identical
8. `$()' command substitution identical
9. test [ ... ] still works; [[ ... ]] preferred everywhere
```

Force zsh to behave like sh in a function.

```bash
my_compat_fn() {
    emulate -L sh                                                                       local to fn only
    setopt KSH_ARRAYS                                                                    0-indexed arrays
    setopt SH_WORD_SPLIT                                                                  word-split unquoted vars
                                                                                          ... bash-style code ...
}
```

Run a script with zsh's bash compatibility.

```bash
zsh -o BASH_REMATCH ./script.sh                                                          enable BASH_REMATCH array
zsh --emulate sh ./script.sh                                                               POSIX mode
```

IFS handling differences.

```bash
bash:  IFS=, read -ra parts <<< "a,b,c"; echo ${parts[0]}                                bash
zsh:   parts=("${(s:,:)1}"); print -l $parts                                              zsh-idiomatic split
zsh:   parts=("${(@s:,:)1}")                                                                @ ensures word-array
```

## Tips

- The `=( ... )` form gives you a TEMP FILE PATH; great for `diff =(cmd1) =(cmd2)` even when one or both produce huge output.
- `<<<` is a here-string: `grep error <<< "$LOG"`. Faster than `echo "$LOG" | grep error`.
- `print -l ${(ko)assoc}` lists keys SORTED, one per line — handy for stable diffs.
- `**/*(.)` is the zsh equivalent of `find . -type f`.
- `**/*(om[1,10])` is the 10 most recent files in a tree — no awk/sort/head pipeline.
- `cd -<TAB>` cycles through dirstack with completion. Pair with AUTO_PUSHD.
- `print -P '%F{red}%U%n%u%f'` quickly tests prompt escapes outside the prompt.
- `${(%):-%x}` gives the path of the CURRENT FILE inside a sourced script (zsh equivalent of bash $BASH_SOURCE).
- Command-not-found can be hooked: define `command_not_found_handler() { ... }` to install missing tools or suggest packages.
- Set `READNULLCMD=less` and you can do `< file.txt` to view a file without a command — zsh inserts $READNULLCMD.
- `noglob cmd ...` runs cmd with globbing disabled for that one invocation (no quoting needed).
- `env -i zsh -fi` runs zsh with ZERO env and skipping rcs (-f) — clean repro env.
- `alias -g ...` global aliases are footguns in scripts; use only interactively.
- `setopt warn_create_global` warns when you accidentally create a global var inside a function (forgot `local`).
- `print -z 'edit me'` puts text into the next prompt's buffer — useful for hooks that want to suggest a corrected line.
- The `(@)` flag forces array context inside double quotes: `"${(@)arr}"` keeps each element as its own word.

## See Also

- bash, fish, nushell, shell-scripting, polyglot, regex, awk, sql

## References

- [Zsh Documentation](https://zsh.sourceforge.io/Doc/) -- complete manual
- [Zsh source / wiki](https://www.zsh.org/) -- canonical home
- [zsh-users on GitHub](https://github.com/zsh-users) -- core plugins (autosuggestions, syntax-highlighting, completions)
- [man zsh](https://man7.org/linux/man-pages/man1/zsh.1.html) -- top-level overview
- [man zshall](https://man7.org/linux/man-pages/man1/zshall.1.html) -- the everything-page (massive)
- [man zshbuiltins](https://man7.org/linux/man-pages/man1/zshbuiltins.1.html) -- builtins
- [man zshexpn](https://man7.org/linux/man-pages/man1/zshexpn.1.html) -- expansion (params, globs)
- [man zshcompsys](https://man7.org/linux/man-pages/man1/zshcompsys.1.html) -- completion system
- [man zshparam](https://man7.org/linux/man-pages/man1/zshparam.1.html) -- parameters
- [man zshoptions](https://man7.org/linux/man-pages/man1/zshoptions.1.html) -- every setopt
- [man zshzle](https://man7.org/linux/man-pages/man1/zshzle.1.html) -- ZLE editor
- [Zsh FAQ](https://zsh.sourceforge.io/FAQ/) -- frequently asked questions
- [Zsh User's Guide](https://zsh.sourceforge.io/Guide/) -- approachable, basics-to-advanced
- [Oh My Zsh](https://ohmyz.sh/) -- popular framework
- [Prezto](https://github.com/sorin-ionescu/prezto) -- modular framework
- [zinit](https://github.com/zdharma-continuum/zinit) -- turbo-mode plugin manager
- [antidote](https://github.com/mattmc3/antidote) -- declarative plugin manager
- [Powerlevel10k](https://github.com/romkatv/powerlevel10k) -- fast theme with instant prompt
- [pure prompt](https://github.com/sindresorhus/pure) -- minimalist prompt
- [starship](https://starship.rs/) -- cross-shell prompt
- ["From Bash to Z Shell"](https://www.apress.com/gp/book/9781590593769) -- Kiddle, Peek, Stephenson — definitive book
- [Awesome Zsh](https://github.com/unixorn/awesome-zsh-plugins) -- curated plugin list
