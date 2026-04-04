# fzf (fuzzy finder)

A general-purpose command-line fuzzy finder that reads lines from stdin or a file list, lets you interactively filter and select entries using fuzzy matching, and outputs the selection to stdout for use in pipelines, keybindings, and scripts.

## Basic Usage

### Simple Selection

```bash
# Find and open a file
vim $(fzf)

# Pipe any list into fzf
ps aux | fzf

# Find a process and kill it
kill -9 $(ps aux | fzf | awk '{print $2}')

# Select from command output
docker ps | fzf | awk '{print $1}'
```

### Search Syntax

```bash
# Fuzzy match (default)
fzf                           # type characters in any order

# Exact match (prefix with ')
# typing 'error matches "error" but not "err_or"

# Prefix match (^)
# ^main matches "main.go" but not "domain"

# Suffix match ($)
# .go$ matches "main.go" but not "gopher"

# Inverse match (!)
# !test excludes lines containing "test"

# Combine terms (space-separated AND, | for OR)
# ^src .go$ !test     matches src/**/*.go excluding test files
# main | init          matches lines with "main" or "init"
```

## Shell Integration

### Key Bindings

```bash
# Install shell integration (bash/zsh/fish)
# Add to .bashrc / .zshrc:
eval "$(fzf --bash)"          # bash
eval "$(fzf --zsh)"           # zsh (fzf 0.48+)
source <(fzf --zsh)           # zsh (older)

# CTRL-T  — paste selected file path onto command line
# CTRL-R  — search command history
# ALT-C   — cd into selected directory
```

### Customize Key Binding Behavior

```bash
# CTRL-T: preview file contents
export FZF_CTRL_T_OPTS="
  --preview 'bat --color=always --line-range :500 {}'
  --bind 'ctrl-/:change-preview-window(down|hidden|)'"

# CTRL-R: show full command, sort chronologically
export FZF_CTRL_R_OPTS="
  --preview 'echo {}' --preview-window up:3:hidden:wrap
  --bind 'ctrl-/:toggle-preview'"

# ALT-C: preview directory tree
export FZF_ALT_C_OPTS="
  --preview 'tree -C {} | head -200'"
```

## Preview Window

### Built-in Preview

```bash
# Preview file contents with bat
fzf --preview 'bat --color=always --style=numbers --line-range=:500 {}'

# Preview with syntax highlighting, positioned right
fzf --preview 'bat --color=always {}' --preview-window right:60%

# Preview directory contents
find . -type d | fzf --preview 'ls -la {}'

# Preview git log for files
git ls-files | fzf --preview 'git log --oneline -10 {}'

# Toggle preview with keybinding
fzf --preview 'bat {}' --bind 'ctrl-/:toggle-preview'
```

### Preview Window Layout

```bash
# Position: right (default), left, up, down
fzf --preview 'cat {}' --preview-window right:50%

# Hidden by default, toggle with ctrl-/
fzf --preview 'cat {}' --preview-window hidden

# Border style
fzf --preview 'cat {}' --preview-window border-left

# Follow mode (scroll preview to bottom)
fzf --preview 'cat {}' --preview-window follow
```

## fzf-tmux

### Tmux Integration

```bash
# Open fzf in a tmux pane (bottom 40%)
fzf-tmux -p 80%,60%          # popup (tmux 3.2+)
fzf-tmux -d 40%              # bottom pane, 40% height
fzf-tmux -u 30%              # top pane, 30% height
fzf-tmux -l 50%              # left pane, 50% width
fzf-tmux -r 40%              # right pane, 40% width

# Combine with preview
fzf-tmux -p 80%,60% --preview 'bat --color=always {}'
```

## Custom Commands

### File and Directory Finders

```bash
# Use fd as default source (faster, respects .gitignore)
export FZF_DEFAULT_COMMAND='fd --type f --hidden --follow --exclude .git'
export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
export FZF_ALT_C_COMMAND='fd --type d --hidden --follow --exclude .git'

# Use ripgrep as source
export FZF_DEFAULT_COMMAND='rg --files --hidden --follow --glob "!.git"'
```

### Git Integration

```bash
# Checkout branch interactively
git branch -a | fzf | xargs git checkout

# Interactive git log browser
git log --oneline | fzf --preview 'git show --color=always {1}'

# Stage files interactively
git diff --name-only | fzf -m --preview 'git diff --color=always {}' | xargs git add

# Interactive git stash
git stash list | fzf --preview 'git stash show -p {1}' | cut -d: -f1 | xargs git stash apply
```

### Custom Key Actions

```bash
# Bind keys to actions within fzf
fzf --bind 'ctrl-y:execute-silent(echo {} | pbcopy)' \
    --bind 'ctrl-e:execute(vim {})' \
    --bind 'ctrl-d:preview-down' \
    --bind 'ctrl-u:preview-up'

# Multi-select with tab, confirm with enter
fzf --multi --bind 'ctrl-a:select-all,ctrl-d:deselect-all'
```

## Advanced Options

### Appearance and Behavior

```bash
# Layout: reverse puts prompt at top
fzf --layout=reverse --border --height=40%

# Exact match mode (no fuzzy)
fzf --exact

# Multi-select mode
fzf --multi

# Custom prompt and header
fzf --prompt='files> ' --header='Select files to edit'

# Delimiter and field selection (for structured input)
echo -e "1:foo:bar\n2:baz:qux" | fzf --delimiter=: --nth=2 --with-nth=2,3

# Custom colors
fzf --color='bg+:#3B4252,bg:#2E3440,spinner:#81A1C1,hl:#616E88'
```

### Output Formatting

```bash
# Print query if no match
fzf --print-query

# Print query and selection separated by newlines
fzf --print-query --expect=ctrl-v,ctrl-x

# Custom separator for multi-select
fzf --multi --bind 'enter:select-all+accept' | tr '\n' ' '

# Select specific field from delimiter-separated input
echo "id:name:path" | fzf --delimiter=: --with-nth=2
```

## Useful Aliases and Functions

### Shell Functions

```bash
# Interactive cd into any subdirectory
fcd() { cd "$(fd --type d | fzf --preview 'tree -C {} | head -50')" ; }

# Edit file found by fzf
fe() { vim "$(fzf --preview 'bat --color=always {}')" ; }

# Kill process interactively
fkill() {
  local pid
  pid=$(ps aux | fzf --header-lines=1 | awk '{print $2}')
  [ -n "$pid" ] && kill -${1:-9} "$pid"
}

# Search environment variables
fenv() { printenv | fzf --preview 'echo {2..}' --delimiter='=' ; }

# Docker container shell
fdocker() {
  local cid
  cid=$(docker ps | fzf --header-lines=1 | awk '{print $1}')
  [ -n "$cid" ] && docker exec -it "$cid" /bin/sh
}
```

## Tips

- Set `FZF_DEFAULT_COMMAND` to `fd` or `rg --files` for dramatically faster file listing that respects `.gitignore`.
- Use `--height=40%` to keep fzf inline in the terminal instead of taking the full screen.
- Press `TAB` in multi-select mode (`--multi`) to toggle individual items; `ctrl-a` to select all.
- Combine search tokens with spaces for AND logic (`^src .go$ !test`) and `|` for OR (`main | init`).
- Use `--preview` with `bat` for syntax-highlighted file previews; add `--line-range :500` to cap preview size.
- The `--bind` flag is incredibly powerful -- bind any key to actions like `execute`, `reload`, `toggle-preview`.
- `fzf-tmux -p` opens a centered popup window in tmux 3.2+ which is cleaner than a split pane.
- CTRL-R history search replaces the default reverse-i-search with a much more usable fuzzy interface.
- Use `--expect=ctrl-v,ctrl-x` to detect which key confirmed selection and branch on it in scripts.
- Pipe `fzf --print-query` output to capture what the user typed even if nothing matched.
- Add `--ansi` when piping colorized input (e.g., from `rg --color=always`) so fzf preserves colors.

## See Also

- ripgrep, fd, bat, tmux, bash, zsh

## References

- [fzf GitHub Repository](https://github.com/junegunn/fzf)
- [fzf Wiki — Examples](https://github.com/junegunn/fzf/wiki/examples)
- [fzf Wiki — Configuring Shell Integration](https://github.com/junegunn/fzf/wiki/Configuring-shell-key-bindings)
- [fzf README — Search Syntax](https://github.com/junegunn/fzf#search-syntax)
- [fzf-tmux Usage](https://github.com/junegunn/fzf#fzf-tmux)
- [Arch Wiki — fzf](https://wiki.archlinux.org/title/Fzf)
