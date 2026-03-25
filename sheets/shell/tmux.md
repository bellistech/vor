# tmux (Terminal Multiplexer)

> Persistent terminal sessions with windows, panes, and detach/reattach support.

## Sessions

### Management

```bash
tmux new -s work                   # new session named "work"
tmux new -s work -d                # new session, detached
tmux ls                            # list sessions
tmux attach -t work                # attach to session
tmux attach -t work -d             # attach and detach other clients
tmux kill-session -t work          # kill session
tmux kill-server                   # kill all sessions
tmux rename-session -t old new     # rename session
tmux switch-client -t other        # switch to another session
```

### Key Bindings (prefix = Ctrl-b)

```bash
# prefix then:
d                  # detach from session
s                  # list/switch sessions (interactive)
$                  # rename current session
(                  # switch to previous session
)                  # switch to next session
```

## Windows

### Management

```bash
tmux new-window -t work            # new window in session
tmux new-window -n logs            # new window named "logs"
tmux select-window -t 2            # switch to window 2
tmux kill-window -t 3              # kill window 3
tmux move-window -t 5              # move window to index 5
tmux swap-window -s 2 -t 4        # swap windows 2 and 4
```

### Key Bindings

```bash
# prefix then:
c                  # create window
&                  # kill window (with confirmation)
,                  # rename window
n                  # next window
p                  # previous window
0-9                # switch to window 0-9
w                  # list windows (interactive picker)
l                  # toggle last active window
f                  # find window by name
```

## Panes

### Splitting

```bash
# prefix then:
%                  # split vertically (left/right)
"                  # split horizontally (top/bottom)
```

### Navigation and Resizing

```bash
# prefix then:
o                  # cycle through panes
;                  # toggle last active pane
q                  # show pane numbers (press number to switch)
arrow keys         # move between panes
z                  # toggle pane zoom (fullscreen)
{                  # swap pane left
}                  # swap pane right
!                  # break pane into its own window
x                  # kill pane (with confirmation)
space              # cycle through layouts

# resize (prefix then hold):
Ctrl+arrow         # resize by 1 cell
Alt+arrow          # resize by 5 cells
```

### Layouts

```bash
# prefix then:
Alt+1              # even-horizontal
Alt+2              # even-vertical
Alt+3              # main-horizontal
Alt+4              # main-vertical
Alt+5              # tiled
```

## Copy Mode

### Vi-Style (recommended)

```bash
# enable in .tmux.conf
set-window-option -g mode-keys vi

# prefix then:
[                  # enter copy mode
q                  # exit copy mode

# in copy mode (vi keys):
/                  # search forward
?                  # search backward
n                  # next search match
N                  # previous search match
v                  # begin selection
y                  # yank (copy) selection
Space              # begin selection (alternative)
Enter              # copy selection and exit
g                  # go to top
G                  # go to bottom
```

### Clipboard Integration

```bash
# macOS — copy to system clipboard
bind-key -T copy-mode-vi y send -X copy-pipe-and-cancel "pbcopy"

# Linux — copy to system clipboard (needs xclip)
bind-key -T copy-mode-vi y send -X copy-pipe-and-cancel "xclip -selection clipboard"

# paste
# prefix then:
]                  # paste buffer
=                  # choose paste buffer from list
```

## Configuration (~/.tmux.conf)

### Common Settings

```bash
# remap prefix to Ctrl-a
unbind C-b
set -g prefix C-a
bind C-a send-prefix

# start window/pane indices at 1
set -g base-index 1
setw -g pane-base-index 1

# renumber windows when one is closed
set -g renumber-windows on

# increase scrollback buffer
set -g history-limit 50000

# reduce escape delay (important for vim)
set -sg escape-time 10

# enable mouse support
set -g mouse on

# true color support
set -g default-terminal "tmux-256color"
set -ag terminal-overrides ",xterm-256color:RGB"

# vi keys in copy mode
setw -g mode-keys vi

# reload config
bind r source-file ~/.tmux.conf \; display "Config reloaded"
```

### Pane Navigation (vim-style)

```bash
bind h select-pane -L
bind j select-pane -D
bind k select-pane -U
bind l select-pane -R

# resize with Ctrl+hjkl
bind -r C-h resize-pane -L 5
bind -r C-j resize-pane -D 5
bind -r C-k resize-pane -U 5
bind -r C-l resize-pane -R 5
```

### Status Bar

```bash
set -g status-position top
set -g status-interval 5
set -g status-left-length 40
set -g status-left '#[fg=green]#S #[fg=white]| '
set -g status-right '#[fg=yellow]%H:%M #[fg=white]%d-%b-%Y'
set -g status-style 'bg=black,fg=white'
setw -g window-status-current-style 'bg=blue,fg=white,bold'
```

## Scripting

### Automating Layouts

```bash
#!/usr/bin/env bash
SESSION="dev"

tmux new-session -d -s "$SESSION" -n editor
tmux send-keys -t "$SESSION:editor" "vim ." Enter

tmux new-window -t "$SESSION" -n server
tmux send-keys -t "$SESSION:server" "npm run dev" Enter

tmux new-window -t "$SESSION" -n logs
tmux split-window -h -t "$SESSION:logs"
tmux send-keys -t "$SESSION:logs.0" "tail -f /var/log/app.log" Enter
tmux send-keys -t "$SESSION:logs.1" "tail -f /var/log/error.log" Enter

tmux select-window -t "$SESSION:editor"
tmux attach -t "$SESSION"
```

### Useful Commands

```bash
# send keys to a specific pane
tmux send-keys -t work:0.1 "ls -la" Enter

# capture pane contents
tmux capture-pane -t work:0.0 -p > output.txt

# pipe pane output to file
tmux pipe-pane -t work:0.0 -o 'cat >> ~/tmux-log.txt'

# display a message
tmux display-message "Build complete"

# wait for a specific event
tmux wait-for -S build-done
```

## Tips

- The prefix key is Ctrl-b by default. Most users remap to Ctrl-a (screen-style).
- `set -sg escape-time 10` is essential if you use vim inside tmux. The default 500ms delay makes Esc sluggish.
- `tmux attach -d` detaches other clients -- useful when a session is stuck at a smaller terminal size.
- Use `tmux list-keys` to see all current key bindings.
- `tmux show-options -g` and `tmux show-window-options -g` dump all settings.
- Mouse mode (`set -g mouse on`) enables scroll, pane select, and resize with the mouse.
- For true color, you need both the `default-terminal` and `terminal-overrides` settings.
- `prefix z` zooms a pane to fullscreen and back -- invaluable for small panes.
- `tmux source-file ~/.tmux.conf` reloads config without restarting.
- Consider tmux plugin manager (tpm) for plugins like tmux-resurrect (persist sessions across reboots).

## References

- [tmux Wiki](https://github.com/tmux/tmux/wiki) -- official wiki with FAQ, guides, and recipes
- [man tmux](https://man7.org/linux/man-pages/man1/tmux.1.html) -- tmux man page
- [tmux GitHub Repository](https://github.com/tmux/tmux) -- source code, issues, and releases
- [tmux Plugin Manager (tpm)](https://github.com/tmux-plugins/tpm) -- plugin manager for tmux
- [tmux-resurrect](https://github.com/tmux-plugins/tmux-resurrect) -- persist and restore tmux sessions
- [tmux-sensible](https://github.com/tmux-plugins/tmux-sensible) -- universal set of reasonable defaults
- [The Tao of tmux](https://leanpub.com/the-tao-of-tmux/read) -- free online book
- [Awesome tmux](https://github.com/rothgar/awesome-tmux) -- curated list of tmux resources and plugins
