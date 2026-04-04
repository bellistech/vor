# delta (syntax-highlighting pager for git)

A viewer for git and diff output that provides syntax highlighting, line numbers, side-by-side layout, and intelligent word-level diff highlighting as a drop-in replacement for less in the git pager pipeline.

## Installation

### Package Managers

```bash
# macOS
brew install git-delta

# Arch Linux
pacman -S git-delta

# Ubuntu/Debian
dpkg -i git-delta_x.x.x_amd64.deb   # from GitHub releases

# Cargo
cargo install git-delta

# Nix
nix-env -iA nixpkgs.delta
```

### Git Configuration

```bash
# Set delta as git pager (required)
git config --global core.pager delta

# Set delta for interactive diffs
git config --global interactive.diffFilter 'delta --color-only'

# Set delta for merge conflict display
git config --global merge.conflictStyle zdiff3
```

## .gitconfig Setup

### Minimal Configuration

```ini
# ~/.gitconfig
[core]
    pager = delta

[interactive]
    diffFilter = delta --color-only

[delta]
    navigate = true
    side-by-side = false
    line-numbers = true

[merge]
    conflictStyle = zdiff3
```

### Full-Featured Configuration

```ini
[delta]
    navigate = true
    side-by-side = true
    line-numbers = true
    syntax-theme = Dracula
    plus-style = "syntax #003800"
    minus-style = "syntax #3f0001"
    plus-emph-style = "syntax #004000"
    minus-emph-style = "syntax #5f0000"
    line-numbers-left-format = "{nm:>4} "
    line-numbers-right-format = "{np:>4} | "
    file-style = "bold yellow ul"
    file-decoration-style = "yellow ol"
    hunk-header-decoration-style = "blue box"
    hunk-header-style = "file line-number syntax"
    hyperlinks = true
    hyperlinks-file-link-format = "vscode://file/{path}:{line}"
    max-line-length = 512
    wrap-max-lines = unlimited
    tabs = 4
    true-color = always
    diff-so-fancy = false
```

## Display Modes

### Side-by-Side

```bash
# Enable side-by-side in config
git config --global delta.side-by-side true

# One-off side-by-side
git diff | delta --side-by-side

# Set side-by-side width
git diff | delta --side-by-side --width 200

# Auto side-by-side only if terminal is wide enough
git config --global delta.side-by-side true
git config --global delta.width variable
```

### Line Numbers

```bash
# Enable line numbers
git config --global delta.line-numbers true

# Customize line number format
git config --global delta.line-numbers-left-format "{nm:>4} "
git config --global delta.line-numbers-right-format "{np:>4} | "

# Line number styling
git config --global delta.line-numbers-minus-style "red"
git config --global delta.line-numbers-plus-style "green"
git config --global delta.line-numbers-zero-style "dim"
```

### Navigate Mode

```bash
# Enable navigate mode (n/N to jump between files in diff)
git config --global delta.navigate true

# In the pager, press:
#   n    jump to next file
#   N    jump to previous file
```

## Syntax Themes

### Available Themes

```bash
# List all available themes
delta --list-syntax-themes

# Preview themes side by side
delta --show-syntax-themes

# Popular themes
git config --global delta.syntax-theme "Dracula"
git config --global delta.syntax-theme "Nord"
git config --global delta.syntax-theme "Monokai Extended"
git config --global delta.syntax-theme "gruvbox-dark"
git config --global delta.syntax-theme "OneHalfDark"
git config --global delta.syntax-theme "Solarized (dark)"
git config --global delta.syntax-theme "ansi"

# Use for light terminals
git config --global delta.syntax-theme "GitHub"
git config --global delta.syntax-theme "Solarized (light)"

# Dark mode toggle (based on terminal)
git config --global delta.dark true
git config --global delta.light false
```

## Color and Style Customization

### Style String Syntax

```bash
# Style format: "foreground-color background-color attributes"
# Colors: named (red, green), hex (#ff0000), RGB (rgb(255,0,0)), ANSI (ansi-red)
# Attributes: bold, italic, ul (underline), ol (overline), blink, strike

# Plus/minus styles (added/removed lines)
git config --global delta.plus-style "syntax #003800"
git config --global delta.minus-style "syntax #3f0001"

# Emphasis styles (changed words within a line)
git config --global delta.plus-emph-style "syntax bold #004000"
git config --global delta.minus-emph-style "syntax bold #5f0000"

# File header
git config --global delta.file-style "bold yellow ul"
git config --global delta.file-decoration-style "yellow ol"

# Hunk headers
git config --global delta.hunk-header-style "file line-number syntax"
git config --global delta.hunk-header-decoration-style "blue box"

# Word-level diff (inline)
git config --global delta.inline-hint-style "syntax dim"
```

## Merge Conflicts

### Conflict Display

```bash
# Set merge conflict style
git config --global merge.conflictStyle zdiff3

# Delta renders conflicts with color:
#   - "ours" in minus-style (red by default)
#   - "theirs" in plus-style (green by default)
#   - base (zdiff3) in dimmed style

# Example conflict in zdiff3:
# <<<<<<< HEAD
#   our_code();
# ||||||| parent
#   original_code();
# =======
#   their_code();
# >>>>>>> feature-branch
```

## Hyperlinks

### Editor Integration

```bash
# Enable clickable file paths
git config --global delta.hyperlinks true

# VSCode links
git config --global delta.hyperlinks-file-link-format "vscode://file/{path}:{line}"

# IntelliJ / JetBrains
git config --global delta.hyperlinks-file-link-format "idea://open?file={path}&line={line}"

# Sublime Text
git config --global delta.hyperlinks-file-link-format "subl://open?url=file://{path}&line={line}"

# Emacs
git config --global delta.hyperlinks-file-link-format "emacs://open?url=file://{path}&line={line}"
```

## Integration with Other Tools

### bat Integration

```bash
# Delta uses the same syntax highlighting as bat (via syntect)
# Themes from bat are available in delta

# List bat themes that delta can use
bat --list-themes

# Use bat theme in delta
git config --global delta.syntax-theme "$(bat --list-themes | fzf)"
```

### diff-so-fancy Migration

```bash
# Delta replaces diff-so-fancy entirely
# Remove old config:
git config --global --unset core.pager
git config --global --unset interactive.diffFilter

# Add delta:
git config --global core.pager delta
git config --global interactive.diffFilter 'delta --color-only'

# Emulate diff-so-fancy style
git config --global delta.diff-so-fancy true

# Or manually approximate the style:
git config --global delta.commit-decoration-style "bold yellow box ul"
git config --global delta.file-style "bold yellow ul"
git config --global delta.file-decoration-style "none"
git config --global delta.hunk-header-decoration-style "cyan box ul"
```

### Color-Moved Detection

```bash
# Git can detect moved lines -- delta respects this
git config --global diff.colorMoved default
git config --global diff.colorMovedWS ignore-space-change

# Delta styles for moved lines
git config --global delta.map-styles \
  "bold purple => syntax #330033, bold cyan => syntax #003333"
```

## Named Profiles (Features)

### Multiple Configurations

```ini
# ~/.gitconfig -- define named feature sets

[delta]
    features = decorations

[delta "decorations"]
    commit-decoration-style = bold yellow box ul
    file-style = bold yellow ul
    file-decoration-style = none
    hunk-header-decoration-style = cyan box ul

[delta "side-by-side-line-nums"]
    side-by-side = true
    line-numbers = true

[delta "minimal"]
    side-by-side = false
    line-numbers = false
    file-decoration-style = none
    hunk-header-decoration-style = none
```

```bash
# Use a specific feature
git diff | delta --features "side-by-side-line-nums"

# Combine features
git diff | delta --features "decorations side-by-side-line-nums"
```

## Usage Outside Git

### Standalone Diff

```bash
# Diff two files directly
delta file_a.py file_b.py

# Diff with specific language syntax
delta --language=python old.txt new.txt

# Pipe unified diff format
diff -u old.py new.py | delta

# Use with other diff tools
colordiff old.py new.py | delta --color-only
```

## Tips

- Set `navigate = true` to jump between files with `n`/`N` in the pager -- essential for large diffs.
- Use `side-by-side = true` for code review but switch to unified for narrow terminals.
- The `syntax-theme` option accepts any bat/syntect theme -- run `delta --list-syntax-themes` to browse.
- Enable `hyperlinks = true` with your editor's URL scheme to click file paths directly into your editor.
- Set `diff.colorMoved = default` in git config so delta can show moved blocks in distinct colors.
- Use `--color-only` for `interactive.diffFilter` so `git add -p` works correctly with delta.
- Create named feature profiles in `.gitconfig` to switch between display modes without editing config.
- The `max-line-length` option (default 512) truncates long lines -- increase it for minified files.
- Delta respects `BAT_THEME` environment variable if no `syntax-theme` is set in config.
- Word-level emphasis highlighting (`plus-emph-style` / `minus-emph-style`) shows exactly what changed within a line.

## See Also

- bat, fzf, lazygit, ripgrep

## References

- [delta GitHub Repository](https://github.com/dandavison/delta)
- [delta User Manual](https://dandavison.github.io/delta/)
- [delta Syntax Themes](https://dandavison.github.io/delta/choosing-a-syntax-theme.html)
- [bat Themes (compatible)](https://github.com/sharkdp/bat#adding-new-themes)
- [Git diff.colorMoved](https://git-scm.com/docs/git-diff#Documentation/git-diff.txt---color-movedltmodegt)
