# zoxide (smarter cd)

A smarter cd command that learns your habits, ranking directories by frecency (frequency + recency) so you can jump to deeply nested paths with just a few characters instead of typing full paths.

## Installation

### Package Managers

```bash
# macOS
brew install zoxide

# Arch Linux
pacman -S zoxide

# Ubuntu/Debian (via cargo)
cargo install zoxide --locked

# Nix
nix-env -iA nixpkgs.zoxide

# Windows (scoop)
scoop install zoxide
```

### Shell Integration

```bash
# Add to ~/.bashrc
eval "$(zoxide init bash)"

# Add to ~/.zshrc
eval "$(zoxide init zsh)"

# Add to ~/.config/fish/config.fish
zoxide init fish | source

# Add to Nushell config
zoxide init nushell | save -f ~/.cache/zoxide/init.nu

# Add to PowerShell profile
Invoke-Expression (& { (zoxide init powershell | Out-String) })

# Use --cmd to change the command name (default: z)
eval "$(zoxide init bash --cmd cd)"   # replaces cd entirely
eval "$(zoxide init zsh --cmd j)"     # use j/ji instead of z/zi
```

## Core Commands

### z -- Jump to Directory

```bash
# Jump to the highest-ranked directory matching "proj"
z proj

# Jump to a path matching multiple keywords (AND logic)
z foo bar            # matches /home/user/foo/project/bar

# Jump to an exact path (bypasses ranking)
z /usr/local/bin

# Jump to home directory
z

# Jump to previous directory (like cd -)
z -

# Jump to the highest-ranked subdirectory of current dir
z proj .             # "." restricts to children
```

### zi -- Interactive Selection

```bash
# Interactively select from matching directories (requires fzf)
zi proj

# Browse all tracked directories
zi

# Interactive with multiple keywords
zi docs work
```

## Database Management

### Query the Database

```bash
# List all tracked directories, sorted by score
zoxide query --list

# List with scores visible
zoxide query --score --list

# Search for specific pattern
zoxide query proj

# Search with multiple keywords
zoxide query foo bar

# Show all entries (including excluded)
zoxide query --all --list

# Show where the database is stored
zoxide query --list 2>/dev/null; echo "$ZOXIDE_DATA_DIR"
```

### Add and Remove Entries

```bash
# Manually add a directory
zoxide add /path/to/project

# Add current directory
zoxide add .

# Remove a directory from the database
zoxide remove /path/to/old/project

# Remove current directory
zoxide remove .

# Remove interactively
zoxide remove --interactive proj
```

## Environment Variables

### Configuration

```bash
# Database location (default: platform-specific data dir)
export _ZO_DATA_DIR="$HOME/.local/share/zoxide"

# Custom fzf options for zi
export _ZO_FZF_OPTS="--height 40% --layout=reverse --preview 'ls -la {2..}'"

# Exclude directories from tracking (regex patterns)
export _ZO_EXCLUDE_DIRS="/tmp/*:/private/*:$HOME/Downloads/*"

# Maximum number of entries in the database
export _ZO_MAXAGE=10000

# Resolve symlinks before adding to database
export _ZO_RESOLVE_SYMLINKS=1
```

### Database Location by Platform

```bash
# Linux: $XDG_DATA_HOME/zoxide/db.zo  (default: ~/.local/share/zoxide/db.zo)
# macOS: ~/Library/Application Support/zoxide/db.zo
# Windows: %LOCALAPPDATA%\zoxide\db.zo

# Override with environment variable
export _ZO_DATA_DIR="$HOME/.zoxide"
```

## Frecency Algorithm

### How Scoring Works

```bash
# Score = frequency_weight * recency_weight
# Recency decays over time:
#   last 1 hour:   score *= 4
#   last 1 day:    score *= 2
#   last 1 week:   score *= 1
#   after 1 week:  score *= 0.5

# View current scores
zoxide query --score --list | head -20

# Aging: when total score exceeds _ZO_MAXAGE (default 10000),
# all scores are multiplied by 0.9 (decay)
# Entries with score < 1 are pruned automatically
```

## Shell Completions

### Tab Completion Setup

```bash
# Bash completions (add to .bashrc after init)
eval "$(zoxide init bash)"
# completions are auto-registered

# Zsh completions
eval "$(zoxide init zsh)"
# completions work with zsh-autocomplete or built-in

# Fish completions (auto-installed)
zoxide init fish | source

# Verify completions are working
z <TAB>              # shows matching directories
zi <TAB>             # shows matching directories for interactive
```

## Advanced Usage

### Integration with Other Tools

```bash
# Use with fzf preview
export _ZO_FZF_OPTS="
  --height=50%
  --layout=reverse
  --border
  --preview='eza --tree --level=2 --color=always {2..}'
  --preview-window=right:40%
"

# Pipe zoxide query into other tools
zoxide query --list | fzf --preview 'ls -la {}' | xargs cd

# Use in scripts (raw query without shell hooks)
target=$(zoxide query project 2>/dev/null) && cd "$target"

# Combine with tmux sessionizer
tmux_session() {
  local dir=$(zoxide query --list | fzf)
  local name=$(basename "$dir")
  tmux new-session -d -s "$name" -c "$dir" 2>/dev/null
  tmux switch-client -t "$name"
}
```

### Importing from Other Tools

```bash
# Import from autojump
zoxide import --from autojump "$HOME/.local/share/autojump/autojump.txt"

# Import from z.sh / z.lua
zoxide import --from z "$HOME/.z"

# Import from fasd
zoxide import --from autojump <(fasd -l)

# Merge databases
zoxide import --merge --from z /path/to/old/.z
```

### Hooks

```bash
# zoxide hooks into cd to automatically track directories
# On every directory change, it runs: zoxide add <new_dir>

# In bash/zsh, this is done via:
#   - bash: PROMPT_COMMAND or precmd
#   - zsh: chpwd hook
#   - fish: --on-variable PWD

# Verify hook is running
cd /tmp && zoxide query --score tmp
# Score should increase after cd
```

## Troubleshooting

### Common Issues

```bash
# "z: command not found" -- shell init not loaded
# Fix: ensure eval "$(zoxide init <shell>)" is in your rc file

# zi not working -- fzf not installed
# Fix: install fzf
brew install fzf    # macOS
apt install fzf     # Debian/Ubuntu

# Database not updating -- hook not firing
# Debug: run with RUST_LOG
RUST_LOG=debug z proj

# Exclude not working -- check regex syntax
# _ZO_EXCLUDE_DIRS uses regex, not glob
export _ZO_EXCLUDE_DIRS="/tmp.*:/proc.*"

# Reset database
rm "$(zoxide query --list 2>/dev/null; echo "${_ZO_DATA_DIR:-$HOME/.local/share/zoxide}")/db.zo"
```

## Tips

- Use `z foo bar` with multiple keywords to narrow matches -- each keyword must appear in the path in order.
- Set `_ZO_FZF_OPTS` with `--preview 'eza --tree {2..}'` to see directory contents when using `zi`.
- Run `zoxide query --score --list` periodically to see which paths have the highest frecency scores.
- Use `eval "$(zoxide init zsh --cmd cd)"` to replace `cd` entirely so you never think about it.
- Import your existing z.sh or autojump database with `zoxide import --from z ~/.z` to avoid a cold start.
- The frecency algorithm means recently visited dirs rank higher than old favorites -- recency decays hourly.
- Set `_ZO_EXCLUDE_DIRS` to keep `/tmp`, downloads, and other transient directories out of the database.
- Tab completion with `z` works in all shells and shows matching directories ranked by score.
- The `zi` command requires fzf installed separately -- it will not work without it.
- Use `z -` to jump back to the previous directory, just like `cd -`.
- Database auto-prunes entries with scores below 1, so abandoned directories fade out naturally.

## See Also

- fzf, fish, bash, zsh, tmux

## References

- [zoxide GitHub Repository](https://github.com/ajeetdsouza/zoxide)
- [zoxide Wiki](https://github.com/ajeetdsouza/zoxide/wiki)
- [Frecency Algorithm (Mozilla)](https://developer.mozilla.org/en-US/docs/Mozilla/Tech/Places/Frecency_algorithm)
- [zoxide man page](https://github.com/ajeetdsouza/zoxide/blob/main/man/man1/zoxide.1)
