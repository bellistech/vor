# tmux (Terminal Multiplexer)

> Persistent terminal sessions, windows, panes, copy-mode, scripting, plugins — survives SSH drops and shell exits.

## Setup

### Install

```bash
# macOS — IMPORTANT: system tmux is ancient. Always brew install.
brew install tmux

# Debian/Ubuntu
sudo apt install tmux

# RHEL/Fedora
sudo dnf install tmux

# Arch
sudo pacman -S tmux

# Alpine
apk add tmux

# from source (latest features)
git clone https://github.com/tmux/tmux.git
cd tmux && sh autogen.sh && ./configure && make && sudo make install
```

### Version Check

```bash
tmux -V                           # prints e.g. "tmux 3.5a"
tmux -V | awk '{print $2}'        # 3.5a
```

### macOS Gotcha

```bash
# WHY: macOS often ships a stale tmux 2.x via /usr/bin/tmux which lacks:
# - true-color (RGB) support
# - modern hooks (pane-died, session-created)
# - copy-mode-vi keybind table
# - the @plugin syntax used by tpm

# diagnose
which tmux                        # /usr/bin/tmux  ← BAD
tmux -V                           # tmux 2.x       ← BAD

# fix
brew install tmux
which tmux                        # /opt/homebrew/bin/tmux  (Apple Silicon)
                                  # /usr/local/bin/tmux     (Intel)
tmux -V                           # tmux 3.5a or newer

# put brew path before /usr/bin in $PATH (zsh)
echo 'export PATH="/opt/homebrew/bin:$PATH"' >> ~/.zshrc
```

### Config File Path

```bash
~/.tmux.conf                      # primary user config
~/.config/tmux/tmux.conf          # alternate XDG-style location (3.1+)
/etc/tmux.conf                    # system-wide
```

### Version Feature Matrix

```bash
# tmux 3.0 (2020)  — popup menus (display-popup), focus-events
# tmux 3.1 (2020)  — XDG config dir, command-prompt -F
# tmux 3.2 (2021)  — extended keys (Ctrl-Shift, etc.) via "extended-keys"
# tmux 3.3 (2022)  — empty windows OK, "next-prompt"/"previous-prompt" copy-mode jumps
# tmux 3.4 (2024)  — popup borders, "menu-style", improved styles
# tmux 3.5 (2024)  — server-access ACL, "remain-on-exit-format", more 256-color choices
# tmux 3.5a (2024) — bugfix release, current stable as of 2026
```

## Conceptual Model

### The Hierarchy

```bash
# server   — one daemon per user; lives at /tmp/tmux-$UID/default (a unix socket)
#   |
#   +-- session    — a workspace; named or numbered
#         |
#         +-- window   — a tab; one or more panes
#               |
#               +-- pane    — a pseudo-terminal (PTY) running a shell or program

# clients attach to sessions, not directly to windows or panes
# multiple clients can attach to the same session — they share view & input
# detaching a client leaves the session running on the server
# killing the server kills every session, every window, every pane
```

### The Prefix Key

```bash
# the prefix is the universal escape sequence that says "the next key is for tmux"
# default: Ctrl-b
# common rebind: Ctrl-a (screen-style)

# every binding in tmux except those bound with `bind -n` requires the prefix first
# example: prefix then c creates a window
# example: prefix then % vertical-splits

# prefix key is configurable — see "The Prefix Key" section below
```

### Targets

```bash
# tmux commands take -t TARGET to specify which session/window/pane
# target syntax:
#   session                       just session by name or index
#   session:window                window by index or name within session
#   session:window.pane           pane by index within window within session
#   :window                       current session, named window
#   .pane                         current window, named pane

# examples
tmux send-keys -t work 'date' Enter            # current pane in session "work"
tmux send-keys -t work:0 'date' Enter          # pane 0 of window 0
tmux send-keys -t work:editor 'date' Enter     # pane 0 of window named "editor"
tmux send-keys -t work:editor.1 'date' Enter   # pane 1 of window "editor"
```

## Starting and Attaching

### Start

```bash
tmux                              # start anonymous session (named "0", "1", ...)
tmux new                          # same as above
tmux new -s work                  # named session "work"
tmux new -s work -n editor        # named session, first window named "editor"
tmux new -s work -d               # create detached (no attach)
tmux new -s work -c /path/to/dir  # start session with given working directory
tmux new -A -s work               # attach if exists, else create — VERY USEFUL
tmux new -t work                  # group with existing session "work" (linked windows)
tmux new -x 200 -y 50             # explicit width x height for detached session
```

### Attach

```bash
tmux a                            # attach to last/most-recent session (alias for attach)
tmux at                           # same
tmux attach                       # full form
tmux a -t work                    # attach to specific session
tmux a -d                         # attach AND detach all other clients (force claim)
tmux a -d -t work                 # attach to "work", kicking other clients off
tmux a -r                         # attach read-only (no input)
tmux a -t work -E                 # don't run update-environment hook on attach
```

### List

```bash
tmux ls                           # list sessions
tmux list-sessions                # full form
tmux list-windows -t work         # list windows in session
tmux list-panes -t work:0         # list panes in window 0 of "work"
tmux list-clients                 # list attached clients
tmux list-sessions -F '#{session_name}'   # custom format, just names
```

### Kill

```bash
tmux kill-session -t work         # kill session "work" (windows + panes go too)
tmux kill-session -a              # kill all OTHER sessions, keep current
tmux kill-session -a -t work      # kill all except "work"
tmux kill-window -t work:logs     # kill specific window
tmux kill-pane -t work:0.1        # kill specific pane
tmux kill-server                  # NUKE — every session for this user gone
```

## The Prefix Key

### Default

```bash
# tmux ships with prefix = Ctrl-b
# to send a key bound to prefix, you press: Ctrl-b, release, then the key
# example: prefix d → press Ctrl-b, release, press d → detach
```

### Common Rebind to Ctrl-a

```bash
# in ~/.tmux.conf
unbind C-b                        # remove default binding
set -g prefix C-a                 # set new prefix
bind C-a send-prefix              # let inner programs see Ctrl-a via double-tap

# WHY Ctrl-a beats Ctrl-b for most people:
# - Ctrl-b is bind-key for "back-one-char" in bash readline (annoying conflict)
# - Ctrl-a is the home-of-line key in bash, more easily replaced
# - GNU screen used Ctrl-a for decades; finger memory transfers
# - Ctrl-b in vim is "page up" — annoying inside vim-in-tmux

# COUNTERARGUMENT: emacs users live on Ctrl-a as line-start. They keep Ctrl-b.
```

### Double-Tap to Send Literal Prefix

```bash
# scenario: you rebound to Ctrl-a, but the program inside the pane (bash, vim)
# also wants Ctrl-a (line-start). Pressing Ctrl-a alone is captured by tmux.

# fix: bind the prefix back to itself with send-prefix
bind C-a send-prefix
# now pressing Ctrl-a Ctrl-a sends a literal Ctrl-a to the inner program
```

### Send Prefix to Outer (Nested)

```bash
# scenario: you have local tmux, ssh into remote, run remote tmux. Both use C-a.
# pressing C-a now triggers LOCAL tmux. The remote one is invisible to your input.

# option A — double-tap pattern
bind C-a send-prefix              # C-a C-a sends prefix to inner

# option B — visual indicator + conditional rebind
bind -T root F12 \
  set prefix None \;\
  set key-table off \;\
  set status-bg red \;\
  refresh-client -S
bind -T off F12 \
  set -u prefix \;\
  set -u key-table \;\
  set -u status-bg \;\
  refresh-client -S
# F12 toggles "off" mode where outer ignores all bindings, passing through to inner
```

## Sessions

### Create / Rename / Switch

```bash
# from shell
tmux new -s work
tmux rename-session -t old-name new-name
tmux switch-client -t other       # only works from inside an attached client

# from inside tmux (prefix then key)
prefix d                          # detach current session
prefix D                          # choose client to detach (interactive)
prefix s                          # interactive session list (choose-tree)
prefix $                          # rename current session
prefix (                          # switch to PREVIOUS session
prefix )                          # switch to NEXT session
prefix L                          # switch to last (most-recent) client/session
prefix C-z                        # suspend current client (fg in shell to resume)
prefix Ctrl-s                     # if tmux-resurrect installed: save session
prefix Ctrl-r                     # if tmux-resurrect installed: restore session
```

### List + Attach Pattern

```bash
# the canonical "what sessions do I have?" workflow
tmux ls                           # see them
tmux a -t work                    # attach by name

# attach-or-create idiom
tmux a -t work || tmux new -s work
# OR with -A (preferred, single command)
tmux new -A -s work
```

### Nested Sessions

```bash
# you ssh to a server, start tmux. Now you have local-tmux and remote-tmux.
# both use Ctrl-a. The first Ctrl-a is captured by LOCAL.

# strategies:
# 1. double-tap: bind C-a send-prefix → C-a C-a forwards to remote
# 2. F12 mode: see "Send Prefix to Outer" above
# 3. different prefix in remote ~/.tmux.conf  (e.g. C-q for remote)
# 4. iTerm2 tmux integration: tmux -CC on remote → iTerm draws windows natively
```

## Windows

### Create / Rename / Kill

```bash
# from shell
tmux new-window -t work
tmux new-window -t work -n logs
tmux new-window -t work -n logs -c /var/log         # cwd
tmux rename-window -t work:0 editor
tmux kill-window -t work:logs

# from inside tmux
prefix c                          # create window (named after running program)
prefix ,                          # rename window
prefix &                          # kill window (with confirmation prompt)
prefix .                          # move-window — prompt for new index
prefix :                          # opens command prompt — type "kill-window" etc.
```

### Navigation

```bash
prefix n                          # next window
prefix p                          # previous window
prefix l                          # last (toggle between two recent)
prefix 0..9                       # window 0..9 directly
prefix '                          # prompt for window index (3.0+)
prefix w                          # interactive window picker (choose-tree)
prefix f                          # find window by name (prompt)
```

### Move / Swap

```bash
prefix .                          # move window to target index (prompt)
prefix M-n                        # next window with bell or activity
prefix M-p                        # previous window with bell or activity

# move-window between sessions
tmux move-window -s work:logs -t play:1
tmux swap-window -s work:0 -t work:3
```

### Workflow: rename window 0 to project name

```bash
# default: tmux names window after current command (zsh, bash, vim, etc.)
# habit: rename window 0 to your project so prefix w picker is meaningful

tmux new -s repo-cs -c ~/code/cs   # session "repo-cs"
prefix ,                           # rename window
# type: cs
prefix c                           # new window (becomes window 1)
prefix ,                           # rename to "tests"
# now prefix w shows: cs, tests, ... not zsh, zsh, ...

# auto-rename windows after current cmd
setw -g automatic-rename on        # default on
setw -g automatic-rename off       # turn off if you rename manually
```

## Panes — Splitting

### Basics

```bash
prefix "                          # split horizontally — 2 ROWS (top + bottom)
prefix %                          # split vertically — 2 COLUMNS (left + right)

# mnemonic (NOT obvious):
# % is "two columns" because the symbol has a vertical-ish slash      ← memorize
# " is "two rows"    because the symbol is horizontal-ish quotes       ← memorize
```

### Canonical Re-Bind

```bash
# the % and " keys are unmemorable. The community standard is | and -:
unbind '"'
unbind %
bind | split-window -h            # | for vertical-split (two columns)
bind - split-window -v            # - for horizontal-split (two rows)

# variant: use \| and -, keep cwd of source pane
bind | split-window -h -c "#{pane_current_path}"
bind - split-window -v -c "#{pane_current_path}"
# WHY: without -c, new pane opens in HOME, not where you were
```

### split-window Flags

```bash
tmux split-window                 # split current pane (default vertical, top/bottom)
tmux split-window -h              # horizontal direction → two columns
tmux split-window -v              # vertical direction → two rows (default)
tmux split-window -p 30           # new pane = 30% of parent
tmux split-window -l 40           # new pane = 40 cells
tmux split-window -b              # before (above/left of) source pane, not after
tmux split-window -d              # do not move focus to new pane
tmux split-window -c '#{pane_current_path}'  # cwd from source pane
tmux split-window 'top'           # run command in new pane
```

## Panes — Navigation

### Move Focus

```bash
prefix Up Down Left Right         # arrow keys move between panes
prefix o                          # cycle to next pane
prefix ;                          # toggle to last (most-recent) pane
prefix q                          # show pane numbers; type N to jump
prefix q 0                        # show numbers, jump to pane 0
```

### Vim-Style hjkl

```bash
# canonical re-bind
bind h select-pane -L
bind j select-pane -D
bind k select-pane -U
bind l select-pane -R
```

### Smart Vim+tmux Pane Switch

```bash
# the famous Christoomey vim-tmux-navigator:
# C-h/j/k/l moves between vim splits AND tmux panes seamlessly,
# without prefix, without thinking about which is which.

# in ~/.tmux.conf
is_vim="ps -o state= -o comm= -t '#{pane_tty}' \
        | grep -iqE '^[^TXZ ]+ +(\\S+\\/)?g?(view|n?vim?x?)(diff)?$'"

bind-key -n 'C-h' if-shell "$is_vim" 'send-keys C-h'  'select-pane -L'
bind-key -n 'C-j' if-shell "$is_vim" 'send-keys C-j'  'select-pane -D'
bind-key -n 'C-k' if-shell "$is_vim" 'send-keys C-k'  'select-pane -U'
bind-key -n 'C-l' if-shell "$is_vim" 'send-keys C-l'  'select-pane -R'

# in vim/.vimrc
let g:tmux_navigator_no_mappings = 1
nnoremap <silent> <C-h> :TmuxNavigateLeft<cr>
nnoremap <silent> <C-j> :TmuxNavigateDown<cr>
nnoremap <silent> <C-k> :TmuxNavigateUp<cr>
nnoremap <silent> <C-l> :TmuxNavigateRight<cr>
```

### Swap Panes

```bash
prefix {                          # swap with PREVIOUS pane
prefix }                          # swap with NEXT pane
prefix C-o                        # rotate panes upward
prefix M-o                        # rotate panes downward
```

## Panes — Resize

### Default Bindings

```bash
prefix Ctrl-Up    Ctrl-Down    Ctrl-Left    Ctrl-Right    # resize 1 cell
prefix Alt-Up     Alt-Down     Alt-Left     Alt-Right     # resize 5 cells
```

### Vim-Style Resize Re-Bind

```bash
# the -r flag makes the binding "repeatable" — within "repeat-time" ms (500ms default)
# you can press the second key without re-pressing prefix
bind -r C-h resize-pane -L 5
bind -r C-j resize-pane -D 5
bind -r C-k resize-pane -U 5
bind -r C-l resize-pane -R 5
bind -r H   resize-pane -L 1
bind -r J   resize-pane -D 1
bind -r K   resize-pane -U 1
bind -r L   resize-pane -R 1

set -g repeat-time 500            # how long the repeatable window stays open (ms)
```

### Zoom and Break

```bash
prefix z                          # toggle zoom (current pane fills the window)
                                  # status bar shows "Z" suffix while zoomed
prefix !                          # break-pane: turn current pane into its own window
prefix Space                      # cycle through built-in layouts
```

### resize-pane Flags

```bash
tmux resize-pane -L 10            # resize left 10 cells
tmux resize-pane -R 10            # right
tmux resize-pane -U 5             # up
tmux resize-pane -D 5             # down
tmux resize-pane -x 80 -y 24      # absolute size
tmux resize-pane -Z               # toggle zoom (same as prefix z)
```

## Panes — Layouts

### Built-in Layouts

```bash
# 5 layouts, all selectable from inside tmux:
prefix Alt-1                      # even-horizontal — panes side by side, equal width
prefix Alt-2                      # even-vertical   — panes stacked, equal height
prefix Alt-3                      # main-horizontal — big pane on top, small below
prefix Alt-4                      # main-vertical   — big pane on left, small right
prefix Alt-5                      # tiled           — grid arrangement

prefix Space                      # cycle through layouts in the order above
```

### Custom Layouts

```bash
# from shell
tmux select-layout even-horizontal
tmux select-layout main-vertical
tmux select-layout tiled

# main-* layouts have a "main-pane-width" / "main-pane-height" option
set -g main-pane-width 120
set -g main-pane-height 30
```

### Save and Restore Layout

```bash
# layouts are encoded as a string — capture it for reuse
tmux display-message -p '#{window_layout}'
# example output: 4f1d,206x50,0,0{103x50,0,0,0,102x50,104,0[102x24,104,0,1,102x25,104,25,2]}

tmux select-layout '4f1d,206x50,0,0{...}'    # apply saved layout
```

## Panes — Kill

### Bindings

```bash
prefix x                          # kill-pane (with confirmation prompt y/n)
prefix : kill-pane                # via command prompt, no prompt
prefix : kill-pane -t :.+         # kill the next pane in this window
prefix : kill-pane -a -t :.       # kill ALL panes except current
```

### Killing Last Pane

```bash
# IMPORTANT: when you kill the LAST pane in a window, the WINDOW also dies
# (because a window with zero panes makes no sense)
# similarly: kill the last window in a session → SESSION dies

# safer: detach instead of kill
prefix d                          # leaves session running for later attach
```

### tmux kill-pane CLI

```bash
tmux kill-pane                    # kill current pane (no prompt)
tmux kill-pane -t work:0.1
tmux kill-pane -a -t work:0       # kill all except current in window 0
tmux kill-pane -a                 # kill all panes except current pane
```

## Copy Mode

### Enter and Exit

```bash
prefix [                          # enter copy mode
q                                 # exit copy mode (when inside)
Esc                               # exit copy mode (some configs)
prefix PgUp                       # enter copy mode AND scroll up immediately
```

### Vi-Keys Setup

```bash
# put in ~/.tmux.conf
setw -g mode-keys vi              # vi-style copy-mode keys (default is emacs)
```

### Vi-Mode Key Reference

```bash
# inside copy mode (after prefix [):

# movement
h j k l                           # left, down, up, right
w b                               # next/prev word
W B                               # next/prev WORD (whitespace-delimited)
e E                               # end-of-word / end-of-WORD
0 $                               # start / end of line
^                                 # first non-blank
g G                               # top / bottom of buffer
H M L                             # top / middle / bottom of visible
{ }                               # prev / next paragraph
( )                               # prev / next sentence
Ctrl-u Ctrl-d                     # half-page up / down
Ctrl-b Ctrl-f                     # full-page up / down

# search
/pattern Enter                    # forward
?pattern Enter                    # backward
n N                               # next / prev match

# selection
v                                 # begin selection (character-wise)
V                                 # begin LINE selection
Ctrl-v                            # begin BLOCK (rectangular) selection
y                                 # yank selection AND exit copy mode
Enter                             # yank AND exit (default)
Esc                               # cancel selection (stay in copy mode)
```

### System Clipboard Integration

```bash
# macOS — pbcopy
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "pbcopy"
bind -T copy-mode-vi MouseDragEnd1Pane send -X copy-pipe-and-cancel "pbcopy"

# Linux X11 — xclip (install: apt install xclip)
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "xclip -selection clipboard -i"
bind -T copy-mode-vi MouseDragEnd1Pane send -X copy-pipe-and-cancel "xclip -selection clipboard -i"

# Linux X11 — xsel alternative
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "xsel -i --clipboard"

# Wayland — wl-copy (install: apt install wl-clipboard)
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "wl-copy"

# WSL2 — clip.exe
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "clip.exe"

# OSC 52 (works over SSH! tmux 3.2+)
set -s set-clipboard on           # let tmux send OSC 52 escape sequences
# requires terminal that supports OSC 52: iTerm2, kitty, alacritty, wezterm, foot
```

## Buffers

### Default Buffer

```bash
prefix ]                          # paste latest buffer (paste-buffer)
prefix =                          # interactive buffer picker
prefix #                          # list buffers
prefix -                          # delete most recent buffer
```

### CLI Buffer Commands

```bash
tmux list-buffers                 # show all buffers (newest first)
tmux show-buffer                  # print contents of latest buffer
tmux show-buffer -b mybuf         # specific buffer by name
tmux save-buffer ~/clip.txt       # write latest buffer to file
tmux save-buffer -b mybuf ~/clip.txt
tmux load-buffer ~/file.txt       # load file into a new buffer
tmux load-buffer -b mybuf ~/file.txt
tmux delete-buffer                # delete latest
tmux delete-buffer -b mybuf
tmux paste-buffer                 # paste latest into current pane
tmux paste-buffer -t work:0.1     # paste into specific pane
tmux paste-buffer -d              # paste AND delete buffer
```

### Canonical "load file → paste" Pattern

```bash
# put a file's contents into a tmux buffer, then paste it
tmux load-buffer ~/snippets/long-cmd.sh
# in tmux: prefix ]                # paste the loaded buffer

# one-liner
tmux load-buffer ~/snippets/long-cmd.sh \; paste-buffer
```

## Status Bar

### Default

```bash
# bottom row showing: [session] window-list [time]
# can be moved to top:
set -g status-position top
set -g status-position bottom     # default
```

### Style Options

```bash
# global status style (background + foreground)
set -g status-style "bg=black,fg=white"
set -g status-bg black            # legacy, equivalent
set -g status-fg white            # legacy, equivalent

# left/right segments + length
set -g status-left "#[fg=green]#S #[fg=white]| "
set -g status-left-length 40
set -g status-right "#[fg=yellow]%H:%M #[fg=white]%d-%b-%Y"
set -g status-right-length 80

# justify center / left / right / absolute-centre
set -g status-justify centre

# update interval (seconds)
set -g status-interval 5          # 5s — easier on CPU than 1s default

# window status styles
setw -g window-status-style "bg=black,fg=white"
setw -g window-status-current-style "bg=blue,fg=white,bold"
setw -g window-status-format " #I:#W "
setw -g window-status-current-format " #I:#W "
```

### Status Variables

```bash
# session/window/pane info
#S            session name
#H            full hostname
#h            short hostname (no domain)
#I            window index
#W            window name
#F            window flags (- * # ! Z M)
#P            pane index
#D            pane unique id
#T            pane title
#{pane_current_path}    cwd of the pane
#{pane_current_command} running command (e.g. "vim", "ssh")

# time strftime-style
%H            hour (24h, zero-padded)
%M            minute
%S            second
%Y            year 4-digit
%m            month numeric
%d            day-of-month
%a %A         abbrev / full day name
%b %B         abbrev / full month name
%T            same as %H:%M:%S

# colors (basic)
#[fg=red]
#[bg=#1e1e2e]
#[bold]
#[reverse]
#[default]    reset to default style
```

### Shell Output in Status

```bash
# run shell command, embed output (cached for status-interval seconds)
set -g status-right "#(uptime | awk -F'load average:' '{print $2}') | %H:%M"
set -g status-right "#(git -C #{pane_current_path} branch --show-current) | %H:%M"

# use #(cmd) for shell, #{var} for tmux var, #[fmt] for style
```

## .tmux.conf — Essential Options

### File Format

```bash
# ~/.tmux.conf — read by tmux on server start
# also: tmux source-file ~/.tmux.conf reloads on the fly

# basic syntax
set        OPTION VALUE           # session-level option
setw       OPTION VALUE           # window-level option
set -g     OPTION VALUE           # global (server-wide)
set -ga    OPTION VALUE           # APPEND to existing value
set -u     OPTION                 # UNSET (revert to default)
unbind     KEY                    # remove a binding
bind       KEY COMMAND            # bind key (after prefix)
bind -n    KEY COMMAND            # bind WITHOUT prefix
bind -r    KEY COMMAND            # repeatable binding
bind -T tablename KEY COMMAND     # bind in a custom key-table
```

## .tmux.conf — Modern Defaults

### Recommended Boilerplate

```bash
# ~/.tmux.conf — modern, sane defaults

# === prefix ===
unbind C-b
set -g prefix C-a
bind C-a send-prefix

# === reload config ===
bind r source-file ~/.tmux.conf \; display "config reloaded"

# === terminal + colors ===
set -g default-terminal "tmux-256color"
set -ag terminal-overrides ",xterm-256color:RGB"
set -ag terminal-overrides ",alacritty:RGB"
set -ag terminal-overrides ",foot:RGB"
set -ag terminal-overrides ",ghostty:RGB"
set -ag terminal-overrides ",wezterm:RGB"

# === scrollback + escape delay ===
set -g history-limit 50000        # lines per pane
set -sg escape-time 10            # CRITICAL: vim ESC delay; default 500 is awful
set -g focus-events on            # let vim/neovim see focus changes
set -g display-time 2000          # how long display-message shows (ms)
set -g display-panes-time 2000    # prefix q timeout (ms)

# === indices ===
set -g base-index 1               # windows start at 1, not 0
setw -g pane-base-index 1         # panes start at 1
set -g renumber-windows on        # close window 2 → window 3 becomes 2

# === mouse ===
set -g mouse on

# === activity ===
setw -g monitor-activity on
set -g visual-activity off        # don't beep on activity

# === copy mode ===
setw -g mode-keys vi
bind -T copy-mode-vi v send -X begin-selection
bind -T copy-mode-vi V send -X select-line
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "pbcopy"  # macOS

# === splits with cwd ===
unbind '"'
unbind %
bind | split-window -h -c "#{pane_current_path}"
bind - split-window -v -c "#{pane_current_path}"

# === vim-style pane nav ===
bind h select-pane -L
bind j select-pane -D
bind k select-pane -U
bind l select-pane -R

# === resize ===
bind -r C-h resize-pane -L 5
bind -r C-j resize-pane -D 5
bind -r C-k resize-pane -U 5
bind -r C-l resize-pane -R 5

# === status bar ===
set -g status-position top
set -g status-interval 5
set -g status-style "bg=#1e1e2e,fg=#cdd6f4"
set -g status-left "#[fg=green,bold] #S #[default]"
set -g status-left-length 30
set -g status-right "#[fg=yellow]%H:%M #[fg=white]%d-%b"
set -g status-right-length 60
setw -g window-status-current-style "bg=blue,fg=white,bold"
setw -g window-status-format " #I:#W "
setw -g window-status-current-format " #I:#W "

# === pane borders ===
set -g pane-border-style "fg=#45475a"
set -g pane-active-border-style "fg=#89b4fa"

# === message bar ===
set -g message-style "bg=#f9e2af,fg=#1e1e2e,bold"
```

## Key Bindings

### Forms of `bind`

```bash
bind KEY CMD                      # bind in prefix table; press prefix then key
bind -n KEY CMD                   # bind in root table; press key without prefix
bind -r KEY CMD                   # repeatable; can press key multiple times in repeat-time
bind -T tablename KEY CMD         # bind in custom table (e.g. copy-mode-vi)

unbind KEY                        # remove from prefix table
unbind -n KEY                     # from root table
unbind -T copy-mode-vi v          # from copy-mode-vi table

list-keys                         # show all bindings
list-keys -T copy-mode-vi         # specific table
list-keys -t prefix               # alias for prefix table
```

### Useful Custom Bindings

```bash
# copy mode entry on prefix Esc (if you don't like [)
bind Escape copy-mode

# clear screen + scrollback (kills scrollback history!)
bind C-l send-keys 'C-l' \; clear-history

# join the previous pane as a horizontal split into current window
bind J join-pane -h -s !

# break pane into named window
bind B command-prompt -p "Break to window:" "break-pane -n '%%'"

# kill session shortcut
bind X confirm-before -p "kill-session #S? (y/n)" kill-session

# new window/session in current path
bind c new-window -c "#{pane_current_path}"
bind C new-session

# synchronize-panes toggle
bind y setw synchronize-panes \; display "sync #{?pane_synchronized,on,off}"
```

### Key Notation

```bash
C-x        Ctrl + x
M-x        Alt + x  (Meta)
S-x        Shift + x  (with extended-keys 3.2+)
F1..F12    function keys
Up Down Left Right    arrows
PageUp PageDown Home End Insert Delete
BSpace     backspace
Space      space
Tab        tab
Enter      return
Escape     escape
NPage PPage    page down / up (alternate names)
```

## Mouse Mode

### Enable

```bash
set -g mouse on                   # one knob, full feature

# what you get:
#  - scroll wheel enters copy mode and scrolls history
#  - click pane to focus it
#  - drag pane border to resize
#  - click window in status bar to switch
#  - drag-select text in copy mode
```

### Trade-Off: Native Selection

```bash
# WITH mouse on, dragging selects INTO TMUX BUFFER, not native terminal selection.
# you cannot Cmd-C to system clipboard from a native drag anymore.

# escape hatches:
#  - hold Shift while dragging — most terminals bypass mouse-tracking
#    iTerm2, Terminal.app, Alacritty, kitty, foot, Ghostty: Shift+drag works
#  - use copy-mode + y bound to pbcopy/xclip/wl-copy
#  - tmux 3.2+: set -s set-clipboard on uses OSC 52, works over SSH
```

### Mouse Bindings

```bash
# the default mouse bindings (don't usually need to touch these)
bind -n MouseDown1Pane select-pane -t = \; send-keys -M
bind -n WheelUpPane if-shell -F -t = "#{mouse_any_flag}" \
    "send-keys -M" "if -Ft= '#{pane_in_mode}' 'send-keys -M' 'copy-mode -e'"

# common customization: don't immediately exit copy mode on drag-end
unbind -T copy-mode-vi MouseDragEnd1Pane
bind -T copy-mode-vi MouseDragEnd1Pane send -X copy-pipe "pbcopy"
```

## Hooks

### Available Hooks (3.0+)

```bash
alert-activity           # window-monitor-activity fires
alert-bell               # window bell received
alert-silence            # window-monitor-silence
client-attached
client-detached
client-resized
client-session-changed
pane-died                # process in pane terminated
pane-exited              # remain-on-exit pane closed
pane-focus-in
pane-focus-out
session-closed
session-created
session-renamed
session-window-changed
window-pane-changed
window-renamed
window-linked
window-unlinked
```

### Setting Hooks

```bash
set-hook -g session-created 'display "new session #S"'
set-hook -g pane-died 'display "pane #P died"'
set-hook -g client-attached 'display "client #{client_name} attached"'

# array form for multiple commands
set-hook -ga session-created 'run-shell "echo $(date) #S >> ~/.tmux-log"'
```

### Useful Hook Patterns

```bash
# auto-rename window after pane command exits
set-hook -g pane-exited 'set -w automatic-rename on'

# log session lifecycle to file
set-hook -g session-created 'run "echo created #S >> ~/.tmux-log"'
set-hook -g session-closed  'run "echo closed  #S >> ~/.tmux-log"'

# auto-resize on client attach
set-hook -g client-resized 'refresh-client'
```

## Sessions in Scripts

### Detached Session Setup

```bash
#!/usr/bin/env bash
# dev-env-up.sh — set up a 4-pane workspace for the current project

SESSION="dev"

# kill any existing
tmux kill-session -t "$SESSION" 2>/dev/null

# create detached
tmux new-session -d -s "$SESSION" -n editor -c "$PWD"
tmux send-keys -t "$SESSION:editor" 'vim .' Enter

# split editor window: 70% editor, 30% bottom for shell
tmux split-window -t "$SESSION:editor" -v -p 30 -c "$PWD"
tmux send-keys -t "$SESSION:editor.2" 'clear; echo "ready"' Enter

# server window
tmux new-window -t "$SESSION" -n server -c "$PWD"
tmux send-keys -t "$SESSION:server" 'npm run dev' Enter

# logs window with two panes
tmux new-window -t "$SESSION" -n logs -c "$PWD"
tmux split-window -t "$SESSION:logs" -h -c "$PWD"
tmux send-keys -t "$SESSION:logs.1" 'tail -f /var/log/app.log' Enter
tmux send-keys -t "$SESSION:logs.2" 'tail -f /var/log/error.log' Enter

# focus editor and attach
tmux select-window -t "$SESSION:editor"
tmux attach -t "$SESSION"
```

### Common Script Building Blocks

```bash
tmux new-session -d -s NAME                          # create detached
tmux new-window -t NAME -n WIN -c CWD                # add named window
tmux split-window -t NAME:WIN -h                     # horizontal split
tmux split-window -t NAME:WIN.0 -v -p 30             # vertical 30% under pane 0
tmux send-keys -t NAME:WIN.PANE 'cmd' Enter          # send command + enter
tmux send-keys -t NAME:WIN.PANE 'cmd' C-m            # C-m == Enter (alt syntax)
tmux select-window -t NAME:WIN
tmux select-pane -t NAME:WIN.PANE
tmux resize-pane -t NAME:WIN.PANE -y 20
tmux set-option -t NAME status-style 'bg=red'
tmux attach -t NAME                                  # finally, attach
```

### Wait For

```bash
# rendezvous between scripts
tmux wait-for build-done                  # blocks until signaled
tmux wait-for -S build-done               # signals (other side unblocks)

# example: run heavy build in detached pane, signal when done
tmux new-window -d -t work \
  'make build && tmux wait-for -S build-done'
tmux wait-for build-done && say "build finished"
```

## tmuxinator / smug / tmux-resurrect

### tmuxinator (Ruby)

```bash
gem install tmuxinator

# create config
tmuxinator new myproject          # opens $EDITOR on ~/.config/tmuxinator/myproject.yml

# typical YAML
# name: myproject
# root: ~/code/myproject
# windows:
#   - editor:
#       layout: main-vertical
#       panes:
#         - vim
#         - guard
#   - server: bundle exec rails s
#   - logs: tail -f log/development.log

# launch
tmuxinator start myproject
mux start myproject               # alias
mux myproject                     # shorter
mux ls                            # list configs
mux stop myproject
```

### smug (Go)

```bash
# https://github.com/ivaaaan/smug
brew install smug                 # or: go install github.com/ivaaaan/smug@latest

# YAML at ~/.config/smug/myproject.yml — similar shape to tmuxinator
smug start myproject
smug stop myproject
smug list
```

### tmux-resurrect

```bash
# save sessions to disk, restore after reboot
# install via tpm: set -g @plugin 'tmux-plugins/tmux-resurrect'

# default keybinds
prefix Ctrl-s                     # save
prefix Ctrl-r                     # restore

# save extra: vim/nvim sessions, pane contents
set -g @resurrect-strategy-vim 'session'
set -g @resurrect-strategy-nvim 'session'
set -g @resurrect-capture-pane-contents 'on'
set -g @resurrect-dir '~/.tmux/resurrect'
```

### tmux-continuum

```bash
# auto-save resurrect state every N min
set -g @plugin 'tmux-plugins/tmux-continuum'
set -g @continuum-save-interval '15'   # minutes
set -g @continuum-restore 'on'         # auto-restore on tmux start
```

## Plugin Management

### tpm — Tmux Plugin Manager

```bash
# install
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm

# in ~/.tmux.conf
set -g @plugin 'tmux-plugins/tpm'
set -g @plugin 'tmux-plugins/tmux-sensible'
set -g @plugin 'tmux-plugins/tmux-resurrect'
set -g @plugin 'tmux-plugins/tmux-continuum'
set -g @plugin 'tmux-plugins/tmux-yank'
set -g @plugin 'tmux-plugins/tmux-pain-control'
set -g @plugin 'tmux-plugins/tmux-prefix-highlight'

# initialize tpm — MUST be the LAST line of ~/.tmux.conf
run '~/.tmux/plugins/tpm/tpm'

# in-tmux keys
prefix I                          # install plugins (capital I)
prefix U                          # update plugins
prefix Alt-u                      # uninstall removed plugins
```

### Canonical Plugin Set

```bash
tmux-sensible            # universally good defaults
tmux-resurrect           # save/restore sessions
tmux-continuum           # auto save/restore
tmux-yank                # better copy-paste integration with system clipboard
tmux-pain-control        # | and - for splits, hjkl pane nav, HJKL resize
tmux-prefix-highlight    # status indicator when prefix is pressed
tmux-copycat             # regex search in copy mode
tmux-open                # o/O in copy mode to open URL/file
catppuccin/tmux          # theme
dracula/tmux             # theme
```

## Nested Sessions

### Detection

```bash
# inside a tmux pane the env var TMUX is set; use it to detect nesting
[ -n "$TMUX" ] && echo "in tmux"

# in ~/.zshrc — different prefix when SSHed inside tmux
if [ -n "$SSH_CONNECTION" ] && [ -n "$TMUX" ]; then
  # remote tmux uses different prefix
  : # noop here; prefix is set in remote ~/.tmux.conf
fi
```

### F12 Toggle Outer

```bash
# bind F12 to toggle "outer ignores all keys, pass through to inner"
bind -T root F12 \
  set prefix None \;\
  set key-table off \;\
  set status-style "fg=#cdd6f4,bg=#181825" \;\
  set window-status-current-format '#[fg=#1e1e2e,bg=#fab387] #I:#W ' \;\
  refresh-client -S

bind -T off F12 \
  set -u prefix \;\
  set -u key-table \;\
  set -u status-style \;\
  set -u window-status-current-format \;\
  refresh-client -S
```

### iTerm2 -CC Mode

```bash
# iTerm2 has native tmux integration — each tmux window becomes an iTerm2 window/tab
ssh server -t 'tmux -CC new -A -s remote'
# hit F2/F3 in iTerm2 to navigate as if local
```

## Sharing Sessions

### Multi-User on Same Host

```bash
# user A
tmux new -s shared

# user B (must read socket — needs perms)
tmux -S /tmp/tmux-1000/default a -t shared

# explicit shared socket
tmux -S /tmp/shared-sock new -s pair    # user A
sudo chgrp dev /tmp/shared-sock         # set group
sudo chmod 770 /tmp/shared-sock         # rwx for group

tmux -S /tmp/shared-sock a -t pair      # user B (must be in dev group)
```

### Detach Other Clients

```bash
tmux a -d -t shared               # attach AND boot all other clients
tmux a -t shared -d -X            # 3.2+ same effect
```

### tmate (easier ad-hoc sharing)

```bash
# tmate is a tmux fork with built-in pairing service
brew install tmate

tmate                             # starts session, prints SSH/web URLs
# share URL with collaborator → they get shared terminal

# read-only URL
tmate                             # then look at status for ssh-readonly link
```

## Logging Pane Output

### pipe-pane

```bash
# tee everything that prints in the pane to a file
tmux pipe-pane -o 'cat >> ~/log-#W-#P.log'
# the -o flag toggles: run again to stop

# from inside tmux: prefix : pipe-pane 'cat >> /tmp/out.log'
# stop: prefix : pipe-pane

# bind for convenience
bind P pipe-pane -o 'cat >> ~/tmux-#S-#W-#P.log' \; display 'logging toggled'
```

### Capture Pane

```bash
# snapshot CURRENT visible content + scrollback
tmux capture-pane -p                    # print to stdout (-p)
tmux capture-pane -p -S -                # include all scrollback (-S - = from start)
tmux capture-pane -p -S -3000            # last 3000 lines of scrollback
tmux capture-pane -p -t work:0.1 > out.txt
tmux capture-pane -e -p                  # include escape sequences (colors)
```

## Synchronized Panes

### Toggle

```bash
# every keystroke in current pane is sent to ALL panes in window
setw synchronize-panes on
setw synchronize-panes off

# from inside tmux
prefix : setw synchronize-panes
prefix : setw synchronize-panes on

# bind to a key
bind y setw synchronize-panes \; display "sync #{?pane_synchronized,ON,OFF}"
```

### Canonical Multi-Host Workflow

```bash
# split window N times, ssh each pane into a different host, then sync-panes:
tmux new -s sync
tmux split-window -h        # 2 panes
tmux split-window -v        # 3
prefix Space                # tile layout

# in each pane (manually): ssh host1, ssh host2, ssh host3
# enable sync
prefix : setw synchronize-panes on

# now `sudo apt update` runs on all 3 hosts
# disable when done
prefix : setw synchronize-panes off
```

## Choose Tree

### What It Is

```bash
# choose-tree is the modern interactive picker for sessions/windows/panes
prefix s                          # session-mode tree
prefix w                          # window-mode tree
prefix : choose-tree              # explicit
prefix : choose-tree -Z           # zoomed
prefix : choose-tree -s           # sessions only
prefix : choose-tree -w           # windows only
```

### Keys Inside choose-tree

```bash
j k or arrows   move
Enter           select
x               kill (with confirmation)
X               kill ALL marked
m               mark / unmark (then x to kill all marked)
<               <  /  >  collapse / expand
H L             top / bottom of list
Space           tag (multi-select)
/               filter
:               command prompt
Esc q           cancel
```

### Customize choose-tree

```bash
# custom format string for entries
set -g @prefix_highlight_show_copy_mode 'on'
# choose-tree itself:
bind s choose-tree -Zs -O time \; display 'sessions by time'
bind w choose-tree -Zw -O name
```

## Show Messages

### Command Prompt

```bash
prefix :                          # opens command prompt at bottom
                                  # type any tmux command (kill-window, source-file ~/.tmux.conf)
prefix ?                          # SHOW ALL KEY BINDINGS (in choose-tree-style picker)
prefix ~                          # show last server messages

# in command prompt:
new-window
kill-window
source-file ~/.tmux.conf
setw synchronize-panes
display 'hello'
```

### display-message

```bash
tmux display-message 'hello'
tmux display-message -p '#S'      # print, don't show; useful in scripts
tmux display-message -p '#{pane_current_path}'

# in config — message styling
set -g message-style "bg=#f9e2af,fg=#1e1e2e,bold"
set -g message-command-style "bg=#f9e2af,fg=#1e1e2e"
set -g display-time 2000          # ms before message disappears
```

## Common .tmux.conf Recipes

### Reload Config Without Restart

```bash
bind r source-file ~/.tmux.conf \; display "config reloaded"

# or from shell
tmux source-file ~/.tmux.conf
```

### Pane Border Status (3.2+)

```bash
set -g pane-border-status top
set -g pane-border-format " #P: #{pane_current_command} "
```

### Project-Specific Config

```bash
# auto-source ~/.tmux/projects/$PROJECT.conf when entering a session
# requires custom hook + scripting
set-hook -g session-created 'run "tmux source-file ~/.tmux/projects/#S.conf 2>/dev/null"'
```

### Smart "New Window In Same Path"

```bash
bind c new-window -c "#{pane_current_path}"
bind | split-window -h -c "#{pane_current_path}"
bind - split-window -v -c "#{pane_current_path}"
```

### Catppuccin-style Status

```bash
set -g status-style "bg=#1e1e2e,fg=#cdd6f4"
set -g status-left "#[fg=#89b4fa,bold] #S #[default]"
set -g status-right "#[fg=#a6e3a1]#h #[fg=#cdd6f4]| #[fg=#fab387]%H:%M "
setw -g window-status-current-style "bg=#89b4fa,fg=#1e1e2e,bold"
```

## Common Errors and Fixes

### "no server running"

```bash
# error
no server running on /private/tmp/tmux-501/default

# meaning
# the tmux server isn't started yet. tmux ls / tmux a fails because there's
# nothing to attach to.

# fix
tmux                              # start the server (creates default session)
# or
tmux new -s work                  # start with named session
```

### "lost server" / SSH Drop

```bash
# error
[lost server]
[lost X11 forwarding connection]

# meaning
# your SSH connection died. The tmux server is still running on the remote box.

# fix
ssh remote                        # reconnect
tmux a                            # reattach to your session — work intact
```

### "open terminal failed: missing or unsuitable terminal"

```bash
# error
open terminal failed: missing or unsuitable terminal: tmux-256color
open terminal failed: missing or unsuitable terminal: alacritty

# meaning
# TERM (set by tmux's default-terminal option) is not in terminfo on this system.

# fixes
export TERM=xterm-256color        # cheap fix
# OR install proper terminfo entry (one-time on each new host)
infocmp -x tmux-256color | ssh remote -- 'tic -x -'
# OR change config
set -g default-terminal "screen-256color"
```

### "no current target"

```bash
# error
no current target

# meaning
# command needs -t SESSION/WINDOW/PANE but you're not in tmux and didn't pass one.

# fix
tmux send-keys -t work:0 'date' Enter   # explicit target
```

### "duplicate session" / "session already exists"

```bash
# error
duplicate session: work
session already exists: work

# meaning
# you ran `tmux new -s work` but session "work" exists.

# fix — attach if exists, else create
tmux new -A -s work               # the canonical idiom
# OR
tmux a -t work || tmux new -s work
```

### "can't find file"

```bash
# error
can't find file ~/.tmux.conf

# meaning
# tmux source-file path doesn't exist, OR your shell didn't expand ~.

# fix
tmux source-file "$HOME/.tmux.conf"
ls -la ~/.tmux.conf               # confirm path
```

### "unknown command"

```bash
# error
~/.tmux.conf:42: unknown command: terminal-features

# meaning
# config uses option/command added in a newer tmux than installed.

# fix
tmux -V                           # check version
brew upgrade tmux                 # update tmux
# OR remove the offending line
```

### "bad config file"

```bash
# error
~/.tmux.conf:17: bad atom: bg=#1e1e2e

# meaning
# tmux 2.x doesn't understand hex colors. Need 3.0+.

# fix
brew upgrade tmux
# OR use named/256-indexed colors
set -g status-style "bg=colour234,fg=colour252"
```

### "no space for new pane"

```bash
# error
no space for new pane

# meaning
# the window is too small to split further at requested size.

# fix
prefix z                          # zoom an existing pane to give yourself room
# resize parent / disable a smaller pane first
```

## Common Gotchas

### ESC Delay in vim

```bash
# bad
# vim's normal-mode ESC takes ~500ms inside tmux
# causes "ESC then o" to behave weird, slow normal-mode entry

# fix
set -sg escape-time 10            # 10ms is plenty
# or 0 if you NEVER paste raw escape sequences
set -sg escape-time 0
```

### Copy-Paste Doesn't Reach System Clipboard

```bash
# bad
# prefix [ then v y copies INTO TMUX BUFFER ONLY
# Cmd-V in browser pastes nothing

# fix (macOS)
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "pbcopy"
bind -T copy-mode-vi MouseDragEnd1Pane send -X copy-pipe-and-cancel "pbcopy"

# fix (Linux X11)
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "xclip -selection clipboard"

# fix over SSH (3.2+)
set -s set-clipboard on
```

### Nested Tmux Confusion

```bash
# bad
# you ssh into a remote host, run tmux there. Now you have local-tmux + remote-tmux.
# Ctrl-a always hits LOCAL. The remote one is mute.

# fix A — different prefix on remote (in remote's ~/.tmux.conf)
unbind C-b
set -g prefix C-q
bind C-q send-prefix

# fix B — F12 mode-toggle (see "Nested Sessions" section above)

# fix C — visual indicator
# show "REMOTE" in status bar of remote tmux to know which one you're driving
```

### Window Indices Off-by-One

```bash
# bad
# default base-index is 0 — so windows are 0, 1, 2, ...
# prefix 1 is first window, prefix 0 is... wait, where's window 0? oh.

# fix
set -g base-index 1
setw -g pane-base-index 1
```

### Renumbering Windows After Kill

```bash
# bad
# windows: 1 (editor), 2 (server), 3 (logs)
# kill window 2 → now you have 1, 3. Skipped index. Ugly.

# fix
set -g renumber-windows on
# now killing window 2 yields 1, 2 (the old 3 became 2)
```

### macOS Tmux Too Old

```bash
# bad
# /usr/bin/tmux is tmux 2.x — no RGB, no popups, no modern hooks
$ tmux -V
tmux 2.10                         # 6 years out of date

# fix
brew install tmux
echo 'export PATH="/opt/homebrew/bin:$PATH"' >> ~/.zshrc
exec zsh
tmux -V                           # tmux 3.5a
```

### True Color Doesn't Work in vim Inside tmux

```bash
# bad
# colorscheme in vim looks washed out / 256-color despite outer terminal being 24-bit

# fix
# in ~/.tmux.conf
set -g default-terminal "tmux-256color"
set -ag terminal-overrides ",xterm-256color:RGB"
set -ag terminal-overrides ",alacritty:RGB"
set -ag terminal-overrides ",foot:RGB"

# in vim/.vimrc
if exists('+termguicolors')
  let &t_8f = "\<Esc>[38;2;%lu;%lu;%lum"
  let &t_8b = "\<Esc>[48;2;%lu;%lu;%lum"
  set termguicolors
endif
```

### Scrollback Cuts Off

```bash
# bad
# scrolling up only goes ~2000 lines back, then nothing

# fix
set -g history-limit 50000        # plenty for most workflows
# 100000 if you compile/test heavily and want all the output
```

### Forgetting You're In tmux

```bash
# bad
# you exit your shell with Ctrl-d expecting to "leave tmux"
# instead, the pane closes, then the window closes, then the SESSION dies.
# all your other panes' state is gone.

# fix
prefix d                          # detach — leaves session running
# alias for habit
echo 'alias td="tmux detach"' >> ~/.zshrc
```

### Default Pane Path is HOME, Not CWD

```bash
# bad
prefix c                          # new window — opens in HOME, not project dir
prefix |                          # split — same thing

# fix
bind c new-window -c "#{pane_current_path}"
bind | split-window -h -c "#{pane_current_path}"
bind - split-window -v -c "#{pane_current_path}"
```

### Status Bar Burns CPU

```bash
# bad
set -g status-interval 1          # 1-second refresh keeps tmux awake constantly
# command in status-right runs every second
set -g status-right '#(curl -s api.example.com/status)'

# fix
set -g status-interval 5          # 5s is plenty
# cache expensive output to file
set -g status-right '#(cat /tmp/my-status-cache)'
# refresh cache via cron / systemd timer
```

### Mouse Mode Steals Native Selection

```bash
# bad
# turning mouse on — drag now selects into TMUX BUFFER, not Cmd-C clipboard

# fix
# 1. hold Shift while dragging (works in iTerm2, Alacritty, kitty, foot, Ghostty, Terminal.app)
# 2. or live without mouse mode
# 3. or bind copy-pipe-and-cancel to pbcopy/xclip/wl-copy
```

### "command-prompt: too many arguments"

```bash
# bad
bind X command-prompt "rename-window %1"
# pressing X then typing a name with spaces fails

# fix — quote the arg
bind X command-prompt "rename-window '%1'"
```

### Escape-Sequence Glitches in vim

```bash
# bad
# vim renders weird color stripes / boxes when scrolling

# fix
set -ga terminal-overrides ",xterm-256color:RGB"
# in vim
set t_ut=                         # disable BCE (background color erase)
```

## Performance Tips

### history-limit

```bash
# memory ~ history-limit * panes * average_line_size
# 50000 lines * 100 bytes * 20 panes = ~100 MB — fine
# 1000000 lines * same = 2 GB — too much
set -g history-limit 50000        # the sweet spot
```

### status-interval

```bash
set -g status-interval 5          # not 1; less CPU
# expensive #(...) commands run on this interval
```

### Minimize Hooks

```bash
# every hook call is overhead; only hook things you use
# don't bind every available hook "just in case"
```

### Avoid Full IPv6 Hostname

```bash
# bad
set -g status-right '#H'          # might be very long: server.long.fqdn.example.com
# fix
set -g status-right '#h'          # short hostname only
```

### Lazy-Load Tmuxinator

```bash
# tmuxinator config files are read on every `mux` invoke
# avoid massive tmuxinator yaml; split per project
```

### Disable Visual Bell

```bash
set -g visual-bell off
set -g bell-action none
```

## Idioms

### "Session Per Project, Window Per Task, Pane Per Process"

```bash
# the canonical mental model:
# session    = a project (cs-cli, unheaded, dotfiles)
# window     = a task within the project (editor, server, tests, logs, scratch)
# pane       = one running process (vim, npm run dev, tail -f log)

tmux new -s cs-cli -c ~/code/cheat_sheet
prefix , cs-cli                            # rename window 1
prefix c                                   # new window
prefix , server
prefix c
prefix , tests
```

### dev-env-up.sh Pattern

```bash
# one shell script per project that creates the workspace
# checked into the repo or stored in ~/bin
# see "Sessions in Scripts" → "Detached Session Setup"
```

### tmux a || tmux

```bash
# canonical attach-or-create alias
echo 'alias t="tmux new -A -s main"' >> ~/.zshrc
# `t` always lands in session "main", creating if needed
```

### Clear-Screen-And-History

```bash
bind C-l send-keys 'C-l' \; clear-history
# C-l (in shell) just clears visible. With this binding, prefix C-l also wipes scrollback.
```

### "Send my last command to that other pane"

```bash
# in one pane, type cmd to test
# duplicate to another pane:
prefix : send-keys -t :.+ "$(history | tail -1 | sed 's/^[ 0-9]* //')" Enter
# wrap in a binding for repeat use
```

## Tips

### Quick One-Liners

```bash
tmux ls                           # list sessions
tmux a                            # attach to most recent
tmux new -s X                     # create session X
tmux new -A -s X                  # attach if exists, else create
tmux kill-session -t X
tmux kill-server                  # nuke everything
tmux source ~/.tmux.conf          # reload config (no restart)
```

### Inspection

```bash
tmux list-keys                    # all bindings
tmux list-keys -T copy-mode-vi
tmux show-options -g              # all global options
tmux show-options -gv prefix      # value of one option
tmux show-window-options -g
tmux info                         # server info dump
tmux display-message -p '#{version}'
```

### Sending Special Keys

```bash
# in send-keys, special tokens replace literal text
tmux send-keys 'echo hi' Enter
tmux send-keys 'C-c'              # Ctrl-C (interrupt)
tmux send-keys 'C-d'              # Ctrl-D (eof)
tmux send-keys 'Escape :w' Enter  # vim save
tmux send-keys -l 'literal text'  # -l = literal, no key parsing

# from a script: kill the foreground job in pane
tmux send-keys -t work:server.0 C-c
```

### display-popup (3.2+)

```bash
# floating popup window
prefix : display-popup -E 'btop'
prefix : display-popup -h 80% -w 80% -E 'lazygit'

# bind a popup
bind G display-popup -h 80% -w 80% -d '#{pane_current_path}' -E 'lazygit'
```

### Force Refresh

```bash
prefix : refresh-client                # repaint
prefix : refresh-client -S             # repaint status only
tmux refresh-client -t /dev/pts/3      # force a specific client
```

### Hidden Useful Commands

```bash
tmux clear-history                # wipe scrollback for current pane
tmux clock-mode                   # display clock in pane (toggle with q)
tmux rotate-window                # rotate panes
tmux respawn-pane -k              # restart dead pane (if remain-on-exit was on)
tmux select-pane -P 'bg=#1e1e2e'  # per-pane bg color override (3.2+)
tmux source-file ~/.tmux.conf -F  # 3.2+ : run as if in current target context
```

### Quick Layouts via Command

```bash
tmux select-layout main-horizontal
tmux select-layout main-vertical
tmux select-layout tiled
tmux select-layout even-horizontal
tmux select-layout even-vertical
```

## See Also

- bash, zsh, fish, screen, vim, neovim, polyglot

## References

- [man tmux](https://man7.org/linux/man-pages/man1/tmux.1.html) — the canonical reference
- [tmux Wiki](https://github.com/tmux/tmux/wiki) — official wiki with FAQ, recipes
- [tmux GitHub](https://github.com/tmux/tmux) — source, releases, issues
- [hamvocke — A Quick and Easy Guide to tmux](https://hamvocke.com/blog/a-quick-and-easy-guide-to-tmux/) — the canonical beginner intro
- ["tmux 2: Productive Mouse-Free Development" by Brian P. Hogan (PragProg)](https://pragprog.com/titles/bhtmux2/tmux-2/) — the book
- [tmux Plugin Manager (tpm)](https://github.com/tmux-plugins/tpm)
- [tmux-resurrect](https://github.com/tmux-plugins/tmux-resurrect)
- [tmux-continuum](https://github.com/tmux-plugins/tmux-continuum)
- [tmux-yank](https://github.com/tmux-plugins/tmux-yank)
- [tmux-sensible](https://github.com/tmux-plugins/tmux-sensible)
- [tmuxinator](https://github.com/tmuxinator/tmuxinator)
- [smug](https://github.com/ivaaaan/smug) — Go alternative to tmuxinator
- [tmate](https://tmate.io/) — tmux fork for instant terminal sharing
- [vim-tmux-navigator](https://github.com/christoomey/vim-tmux-navigator)
- [Awesome tmux](https://github.com/rothgar/awesome-tmux) — curated resources
- [The Tao of tmux](https://leanpub.com/the-tao-of-tmux/read) — free online book
