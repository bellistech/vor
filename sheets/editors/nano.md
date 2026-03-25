# Nano (GNU nano)

> Simple terminal text editor with on-screen shortcut hints and minimal learning curve.

## Notation

```
^X     = Ctrl + X
M-X    = Alt + X (or Esc then X)
```

## Navigation

### Moving Around

```
^F / ^B            forward/back one character
^N / ^P            next/previous line
^A / ^E            beginning/end of line
^V / ^Y            page down/up
M-\ / M-/         beginning/end of file
^_                 go to line number (and column)
M-G                go to line number (alternative)
```

### Scrolling

```
M-( / M-)          scroll up/down one line (without moving cursor)
```

## Editing

### Basic Editing

```
^D                 delete character under cursor
^H                 delete character before cursor (backspace)
M-T                cut from cursor to end of file
^K                 cut current line (or marked region)
^U                 paste (uncut)
M-6                copy current line (or marked region)
M-U                undo
M-E                redo
^J                 justify paragraph
^T                 run spell checker
```

### Selection (Marking)

```
M-A                start/stop marking text
^K                 cut marked region
M-6                copy marked region
^U                 paste last cut/copied text
```

## Search and Replace

```
^W                 search forward
M-W                repeat last search
^W ^R              search backward (inside search prompt)
^\                 search and replace
M-R                search and replace (alternative)
```

### Search Prompt Options

```
M-C                toggle case sensitivity
M-R                toggle regex mode
M-B                search backward direction
```

## Files

```
^O                 save (write out)
^X                 exit (prompts to save if modified)
^R                 insert another file at cursor
^T                 open file browser (when available)
```

## Cut and Paste

```
^K                 cut current line
M-6                copy current line
^U                 paste
M-A then ^K        mark region, then cut
M-A then M-6       mark region, then copy
^K ^K ^K           cut multiple lines (repeat)
```

## Configuration (.nanorc)

### Location

```bash
~/.nanorc                              # user config
/etc/nanorc                            # system-wide config
```

### Common Settings

```bash
# ~/.nanorc
set autoindent                         # auto-indent new lines
set tabsize 4                          # tab width
set tabstospaces                       # convert tabs to spaces
set linenumbers                        # show line numbers
set mouse                              # enable mouse support
set softwrap                           # wrap long lines visually
set atblanks                           # wrap at word boundaries
set backup                             # create backup files (~)
set casesensitive                      # case-sensitive search
set constantshow                       # always show cursor position
set nohelp                             # hide shortcut hints (more screen space)
set smooth                             # smooth scrolling
set zap                                # delete/backspace erases marked region
set indicator                          # show scroll position indicator
```

### Syntax Highlighting

```bash
# include default syntax files
include "/usr/share/nano/*.nanorc"
include "/usr/share/nano/extra/*.nanorc"

# custom syntax example
syntax "conf" "\.conf$"
color green "^#.*"
color yellow "\<(yes|no|true|false)\>"
color brightred "="
```

### Custom Keybindings

```bash
# rebind keys
bind ^S savefile main                  # Ctrl+S to save
bind ^Q exit main                      # Ctrl+Q to quit
bind ^Z undo main                      # Ctrl+Z to undo
bind ^Y redo main                      # Ctrl+Y to redo
```

## Command-Line Options

```bash
nano file.txt                          # open file
nano +10 file.txt                      # open at line 10
nano +10,5 file.txt                    # open at line 10, column 5
nano -B file.txt                       # create backup before editing
nano -l file.txt                       # show line numbers
nano -m file.txt                       # enable mouse
nano -i file.txt                       # auto-indent
nano -E file.txt                       # convert tabs to spaces
nano -w file.txt                       # disable long-line wrapping
nano -Y sh file.txt                    # force syntax highlighting type
```

## Tips

- The bottom two lines always show available shortcuts -- `^G` (Help) lists all of them.
- Nano reads `/usr/share/nano/*.nanorc` for syntax highlighting -- include them in your `.nanorc` for color support.
- Use `set tabstospaces` and `set tabsize 4` in `.nanorc` to match common project conventions.
- `^K` without a mark cuts the entire current line -- repeat it to cut multiple consecutive lines, then `^U` to paste them all.
- `M-A` (mark) combined with arrow keys lets you select arbitrary regions for cut/copy.
- Nano supports regex search: press `M-R` in the search prompt to toggle regex mode.
- Use `nano +N file` to open a file at a specific line -- useful from compiler error output.
- The `-w` flag prevents line wrapping, which is important when editing config files.

## References

- [GNU nano Documentation](https://www.nano-editor.org/docs.php) -- official docs and FAQ
- [GNU nano Manual](https://www.nano-editor.org/dist/latest/nano.html) -- full user manual
- [nanorc Manual](https://www.nano-editor.org/dist/latest/nanorc.5.html) -- configuration file reference
- [man nano](https://man7.org/linux/man-pages/man1/nano.1.html) -- nano man page
- [man nanorc](https://man7.org/linux/man-pages/man5/nanorc.5.html) -- nanorc man page
- [nano Syntax Highlighting](https://github.com/scopatz/nanorc) -- community syntax files for many languages
- [GNU nano News](https://www.nano-editor.org/news.php) -- release history and changelogs
