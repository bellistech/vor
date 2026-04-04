# bat (cat with syntax highlighting)

A cat clone with syntax highlighting, line numbers, git integration, and automatic paging that serves as a drop-in replacement for cat while providing a much richer reading experience for source code, config files, and structured text in the terminal.

## Basic Usage

### Viewing Files

```bash
# Display a file with syntax highlighting
bat main.go

# Display multiple files
bat src/*.go

# Display with explicit language
bat -l json data.txt
bat -l yaml < config.txt

# Read from stdin
curl -s https://example.com/api | bat -l json
echo '{"key": "value"}' | bat -l json

# Concatenate files (like cat)
bat header.txt body.txt footer.txt
```

### Plain Mode (cat Replacement)

```bash
# Plain output — no line numbers, no decorations, no paging
bat -p file.txt

# Completely plain (identical to cat, but with syntax highlighting)
bat -pp file.txt

# No syntax highlighting (truly identical to cat)
bat --style=plain --paging=never --color=never file.txt
```

## Line Range Selection

### Viewing Specific Lines

```bash
# Show lines 10-20
bat --line-range 10:20 main.go

# Show from line 50 to end
bat --line-range 50: main.go

# Show first 30 lines
bat --line-range :30 main.go

# Show last 20 lines (negative not supported — use tail + bat)
tail -20 main.go | bat -l go

# Highlight specific lines
bat --highlight-line 15 main.go
bat --highlight-line 10:20 main.go
```

## Decorations and Style

### Style Components

```bash
# Control decorations (comma-separated)
bat --style=numbers file.go              # line numbers only
bat --style=grid file.go                 # grid lines only
bat --style=header file.go               # filename header only
bat --style=changes file.go              # git changes only
bat --style=numbers,changes file.go      # combine
bat --style=full file.go                 # everything

# Available components:
#   auto, full, plain, numbers, changes, header, header-filename,
#   header-filesize, grid, rule, snip

# Default style
export BAT_STYLE="numbers,changes,header,grid"
```

### Git Integration

```bash
# Show git diff markers in the gutter (default when in git repo)
bat --diff main.go

# Diff markers: + added, ~ modified, - deleted (in left gutter)
# This works automatically — bat detects git repos

# Use bat as git diff pager
git diff | bat -l diff

# Use bat as git show pager
git show HEAD:main.go | bat -l go
```

## Paging

### Pager Control

```bash
# Auto paging (default — pages if output exceeds terminal)
bat file.go

# Always page
bat --paging=always file.go

# Never page (for piping)
bat --paging=never file.go

# Custom pager
bat --pager="less -RF" file.go

# Set default pager
export BAT_PAGER="less -RF"
```

## Themes

### Theme Management

```bash
# List available themes
bat --list-themes

# Preview themes on a file
bat --list-themes | fzf --preview='bat --theme={} --color=always main.go'

# Use a specific theme
bat --theme="Dracula" file.go

# Set default theme
export BAT_THEME="Dracula"

# Popular themes:
#   Dracula, Monokai Extended, Nord, OneHalfDark, OneHalfLight,
#   Solarized (Dark), Solarized (Light), Catppuccin Mocha,
#   gruvbox-dark, ansi

# Add custom themes
mkdir -p "$(bat --config-dir)/themes"
cp mytheme.tmTheme "$(bat --config-dir)/themes/"
bat cache --build
```

## Configuration

### Config File

```bash
# Config file location
bat --config-file                        # shows path
# Typically: ~/.config/bat/config

# Example config file:
# --theme="Dracula"
# --style="numbers,changes,header,grid"
# --pager="less -RF"
# --map-syntax "*.conf:INI"
# --map-syntax ".ignore:Git Ignore"
# --italic-text=always
```

### Syntax Mapping

```bash
# Map file extensions to languages
bat --map-syntax '*.ino:C++' sketch.ino
bat --map-syntax '.env:Bash' .env
bat --map-syntax 'Jenkinsfile:Groovy' Jenkinsfile

# List supported languages
bat --list-languages

# Add custom syntaxes
mkdir -p "$(bat --config-dir)/syntaxes"
cp mylang.sublime-syntax "$(bat --config-dir)/syntaxes/"
bat cache --build
```

## Integration with Other Tools

### As a Manpager

```bash
# Use bat as the man page viewer
export MANPAGER="sh -c 'col -bx | bat -l man -p'"
export MANROFFOPT="-c"

# Or in .bashrc / .zshrc
export MANPAGER="sh -c 'col -bx | bat -l man -p'"
```

### As a Previewer

```bash
# fzf preview
fzf --preview 'bat --color=always --style=numbers --line-range=:500 {}'

# fzf with bat and line highlighting
rg --line-number "pattern" | fzf --delimiter=: \
  --preview 'bat --color=always --highlight-line {2} {1}'

# git log preview
git log --oneline | fzf --preview 'git show {1} | bat -l diff --color=always'
```

### Aliases and Helpers

```bash
# Alias cat to bat
alias cat='bat --paging=never'

# Pretty help pages
alias bathelp='bat --plain --language=help'
help() { "$@" --help 2>&1 | bathelp ; }

# Diff with bat
batdiff() { diff -u "$1" "$2" | bat -l diff ; }

# Show file with line numbers for code review
review() { bat --style=full --highlight-line "$2" "$1" ; }

# Preview YAML/JSON
alias yamlview='bat -l yaml'
alias jsonview='bat -l json'
```

## Supported Formats

### Common Languages and Formats

```bash
# Programming languages
bat -l go main.go
bat -l python script.py
bat -l rust lib.rs
bat -l javascript app.js
bat -l typescript index.ts
bat -l c main.c
bat -l cpp main.cpp

# Config and data formats
bat -l json config.json
bat -l yaml deploy.yaml
bat -l toml config.toml
bat -l ini config.ini
bat -l xml pom.xml
bat -l dockerfile Dockerfile
bat -l nginx nginx.conf

# Special formats
bat -l diff patch.diff
bat -l man man_page
bat -l csv data.csv
bat -l sql schema.sql
bat -l markdown README.md
```

## Tips

- Set `alias cat='bat -pp'` for a drop-in replacement that adds syntax highlighting while keeping cat behavior.
- Use `bat --line-range 10:20` to view specific lines without loading the whole file into a pager.
- `export BAT_THEME` and `export BAT_STYLE` in your shell profile to avoid typing theme and style flags every time.
- bat works seamlessly as fzf's `--preview` command -- add `--color=always` and `--line-range=:500` to keep previews fast.
- The `--diff` flag shows git change markers in the gutter, making bat useful for quick code review without running git diff.
- Map non-standard extensions to languages with `--map-syntax '*.conf:INI'` in your config file.
- Use bat as `MANPAGER` for colorized, searchable man pages with syntax highlighting.
- `bat --list-themes | fzf --preview='bat --theme={} file'` is the fastest way to find a theme you like.
- bat detects piped output and automatically disables paging and decorations -- safe to use in scripts.
- Custom syntaxes and themes use TextMate/Sublime format -- thousands are available from the Sublime community.
- The `-A` flag shows non-printable characters (tabs, newlines, spaces) -- useful for debugging whitespace issues.

## See Also

- cat, less, jless, fzf, ripgrep, fd, git, vim

## References

- [bat GitHub Repository](https://github.com/sharkdp/bat)
- [bat README — Usage](https://github.com/sharkdp/bat#how-to-use)
- [bat README — Configuration](https://github.com/sharkdp/bat#configuration-file)
- [bat README — Adding Custom Themes](https://github.com/sharkdp/bat#adding-new-themes)
- [bat README — Adding Custom Syntaxes](https://github.com/sharkdp/bat#adding-new-syntaxes--language-definitions)
- [Arch Wiki — bat](https://wiki.archlinux.org/title/Bat_(cat_clone))
- [Sublime Syntax Definitions](https://www.sublimetext.com/docs/syntax.html)
