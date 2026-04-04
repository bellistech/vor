# Zsh (Z Shell)

> Extended Bourne shell with powerful completion, globbing, and customization — default on macOS.

## Key Differences from Bash

```bash
# arrays are 1-indexed in zsh (0-indexed in bash)
arr=(a b c)
echo $arr[1]              # a (zsh) vs ${arr[0]} in bash

# word splitting is off by default
name="hello world"
for w in $name; do echo $w; done   # prints "hello world" as one item
for w in ${=name}; do echo $w; done # force splitting: prints hello, then world

# glob failures are errors by default (no unmatched glob passed literally)
ls *.xyz                  # error if no match (bash passes "*.xyz" literally)
ls *.xyz(N)               # (N) = nullglob, silently expand to nothing
```

## Glob Qualifiers

### File Type and Attributes

```bash
ls *(.)                   # regular files only
ls *(/)                   # directories only
ls *(@)                   # symlinks only
ls *(*)                   # executable files
ls *(r)                   # readable by owner
ls *(w)                   # writable by owner
ls *(R)                   # readable by world
ls *(U)                   # owned by current user
```

### Size, Time, and Sorting

```bash
ls *(Lk+100)              # files larger than 100 KB
ls *(Lm-5)                # files smaller than 5 MB
ls *(mh-1)                # modified in last hour
ls *(mw+2)                # modified more than 2 weeks ago
ls *(om)                  # sort by modification time (newest first)
ls *(Om)                  # sort by modification time (oldest first)
ls *(oL)                  # sort by size (smallest first)
ls *(.om[1,5])            # 5 most recently modified files
```

### Combining Qualifiers

```bash
ls **/*.log(.mw+4Lk+100)  # log files older than 4 weeks AND larger than 100K
ls *(.)                    # regular files
ls *(-.)                   # regular files, follow symlinks
```

## Extended Globbing

```bash
ls **/*.go                 # recursive glob (built-in, no shopt needed)
ls **/*test*(.on)          # all test files recursively, sorted by name
ls *.{jpg,png,gif}         # brace expansion
ls ^*.log                  # negation: everything except .log files (with EXTENDED_GLOB)
ls *.txt~README.txt        # all .txt except README.txt
ls (foo|bar)*.sh           # alternation
```

## Completion System

### Setup

```bash
autoload -Uz compinit && compinit     # initialize completion
zstyle ':completion:*' menu select     # arrow-key menu selection
zstyle ':completion:*' matcher-list 'm:{a-z}={A-Z}'  # case-insensitive
zstyle ':completion:*' list-colors ${(s.:.)LS_COLORS} # colored completions
zstyle ':completion:*' completer _expand _complete _correct _approximate
```

### Custom Completions

```bash
# complete hostnames for ssh
zstyle ':completion:*:ssh:*' hosts $(awk '/^Host / {print $2}' ~/.ssh/config)

# complete with descriptions
zstyle ':completion:*' format '%B%d%b'  # bold group descriptions
zstyle ':completion:*' group-name ''    # group by type

# cache completions (faster startup)
zstyle ':completion:*' use-cache on
zstyle ':completion:*' cache-path ~/.zsh/cache
```

## Options (setopt / unsetopt)

```bash
setopt AUTO_CD               # cd by typing directory name alone
setopt AUTO_PUSHD            # cd pushes onto directory stack
setopt PUSHD_IGNORE_DUPS     # no dupes on directory stack
setopt CORRECT               # suggest corrections for commands
setopt CORRECT_ALL           # suggest corrections for arguments too
setopt EXTENDED_GLOB         # enable ^, ~, # in globs
setopt GLOB_DOTS             # include dotfiles in globs
setopt HIST_IGNORE_DUPS      # no consecutive duplicate history entries
setopt HIST_IGNORE_SPACE     # ignore commands starting with space
setopt SHARE_HISTORY         # share history across sessions
setopt INC_APPEND_HISTORY    # append to history immediately
setopt NO_BEEP               # silence terminal bell
```

## History Configuration

```bash
HISTFILE=~/.zsh_history
HISTSIZE=50000
SAVEHIST=50000
setopt HIST_EXPIRE_DUPS_FIRST
setopt HIST_FIND_NO_DUPS
setopt HIST_REDUCE_BLANKS    # trim whitespace
setopt HIST_VERIFY           # show expanded history before running
```

## Prompt Customization

```bash
# built-in prompt escapes
autoload -Uz promptinit && promptinit
PROMPT='%F{green}%n@%m%f:%F{blue}%~%f %# '
# %n = username, %m = hostname, %~ = cwd, %# = # for root, % for user
# %F{color}...%f = foreground color
# %B...%b = bold, %U...%u = underline

# vcs_info for git branch in prompt
autoload -Uz vcs_info
precmd() { vcs_info }
zstyle ':vcs_info:git:*' formats ' (%b)'
RPROMPT='${vcs_info_msg_0_}'    # right-side prompt with branch
```

## Oh My Zsh

### Installation and Basics

```bash
# install
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# ~/.zshrc configuration
ZSH_THEME="robbyrussell"
plugins=(git docker kubectl z fzf history-substring-search)
source $ZSH/oh-my-zsh.sh
```

### Useful Plugins

```bash
# git — aliases: gst, gco, gcm, gp, gl, gd, etc.
# z — frecency directory jumping: z projects
# fzf — fuzzy finder integration
# docker — completion for docker/docker-compose
# kubectl — completion + aliases: k, kgp, kgs, kdp
# history-substring-search — up/down arrow searches matching history
# zsh-autosuggestions — fish-style inline suggestions
# zsh-syntax-highlighting — command highlighting as you type
```

### Custom Plugins (external)

```bash
# zsh-autosuggestions
git clone https://github.com/zsh-users/zsh-autosuggestions ${ZSH_CUSTOM}/plugins/zsh-autosuggestions

# zsh-syntax-highlighting
git clone https://github.com/zsh-users/zsh-syntax-highlighting ${ZSH_CUSTOM}/plugins/zsh-syntax-highlighting

# then add to plugins=() in .zshrc
```

## Useful Builtins

```bash
# directory stack
dirs -v                    # show numbered directory stack
cd -2                      # go to stack entry 2
pushd /var/log             # push and cd
popd                       # pop and cd back

# zmv — regex renaming
autoload -Uz zmv
zmv '(*).txt' '$1.md'                 # rename all .txt to .md
zmv -W '*.jpeg' '*.jpg'              # simpler wildcard form
zmv '(**/)(*).JPG' '$1${2:l}.jpg'    # recursive lowercase rename

# zcalc — calculator
autoload -Uz zcalc
zcalc                     # interactive calculator
```

## Key Bindings

```bash
bindkey -v                          # vi mode
bindkey -e                          # emacs mode (default)
bindkey '^R' history-incremental-search-backward
bindkey '^[[A' history-substring-search-up      # up arrow
bindkey '^[[B' history-substring-search-down    # down arrow
bindkey '^ ' autosuggest-accept     # ctrl-space accept suggestion

# list all bindings
bindkey -L
```

## Tips

- Zsh arrays are 1-indexed. This is the #1 source of bugs when porting bash scripts.
- Use `${=var}` to force word splitting on a variable (off by default in zsh).
- `**/*.go` recursive glob works out of the box -- no `shopt -s globstar` needed.
- `setopt EXTENDED_GLOB` unlocks `^` (negation), `~` (exclusion), and `#` (repetition) in patterns.
- Glob qualifiers like `*(.)` and `*(/)` replace `find` for many simple use cases.
- `zmv` requires `autoload -Uz zmv` before first use.
- Oh My Zsh can slow startup. Profile with `zprof`: add `zmodload zsh/zprof` at top of .zshrc, then run `zprof`.
- For maximum compatibility, scripts should use `#!/bin/bash` -- zsh-specific syntax will break under bash.
- `RPROMPT` sets a right-aligned prompt -- great for git branch or timestamps.
- `hash -d proj=~/projects` creates a named directory: `cd ~proj`.

## See Also

- bash, shell-scripting, tmux, regex, vim, git

## References

- [Zsh Documentation](https://zsh.sourceforge.io/Doc/) -- official manual (all sections)
- [man zsh](https://man7.org/linux/man-pages/man1/zsh.1.html) -- zsh overview man page
- [man zshbuiltins](https://man7.org/linux/man-pages/man1/zshbuiltins.1.html) -- built-in commands reference
- [man zshexpn](https://man7.org/linux/man-pages/man1/zshexpn.1.html) -- parameter expansion and globbing
- [man zshcompsys](https://man7.org/linux/man-pages/man1/zshcompsys.1.html) -- completion system
- [Zsh FAQ](https://zsh.sourceforge.io/FAQ/) -- frequently asked questions
- [Zsh Guide (zsh.sourceforge.io)](https://zsh.sourceforge.io/Guide/) -- user-friendly guide from basics to advanced
- [Oh My Zsh](https://ohmyz.sh/) -- framework for managing Zsh configuration
- [Zsh GitHub Mirror](https://github.com/zsh-users/zsh) -- source code mirror
- [zsh-autosuggestions](https://github.com/zsh-users/zsh-autosuggestions) -- fish-like autosuggestions for Zsh
- [zsh-syntax-highlighting](https://github.com/zsh-users/zsh-syntax-highlighting) -- real-time syntax highlighting
