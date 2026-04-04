# The Internals of tmux — Client-Server Architecture, Window Management, and Multiplexing

> *tmux is a terminal multiplexer with a client-server architecture. The server process manages sessions, windows, and panes as a tree structure. The client connects via a Unix domain socket. This separation means sessions persist when the client disconnects — the fundamental feature that makes tmux indispensable for remote work.*

---

## 1. Architecture — Client-Server Model

### Process Separation

```
Terminal emulator
    │
    └── tmux client (thin — just sends keys, receives display)
           │
           └── Unix domain socket (/tmp/tmux-UID/default)
                  │
                  └── tmux server (long-lived daemon)
                         │
                         ├── Session 1
                         │    ├── Window 1
                         │    │    ├── Pane 1 (pty → bash)
                         │    │    └── Pane 2 (pty → vim)
                         │    └── Window 2
                         │         └── Pane 1 (pty → htop)
                         └── Session 2
                              └── Window 1
                                   └── Pane 1 (pty → ssh)
```

### Why Client-Server Matters

| Event | Without tmux | With tmux |
|:------|:-------------|:----------|
| SSH disconnects | All processes killed (SIGHUP) | Server keeps running |
| Terminal closes | Processes terminated | Server keeps running |
| Reconnect | Start over | `tmux attach` — everything intact |

### The Server Lifecycle

1. First `tmux` command: server starts (forks to background)
2. Client connects via Unix socket
3. Multiple clients can connect to the same session
4. Server exits when all sessions are closed (no windows/panes)

---

## 2. The Session-Window-Pane Hierarchy

### Data Model

```
Server
 └── Session (named, 0+)
      └── Window (numbered, 1+)
           └── Pane (within window layout, 1+)
                └── Pseudo-terminal (pty)
                     └── Shell or command
```

### Relationships

| Entity | Contains | Identity | Persistence |
|:-------|:---------|:---------|:------------|
| Server | Sessions | Socket path | Until last session dies |
| Session | Windows | Name (string) | Until explicitly killed |
| Window | Panes | Index (0-based) | Until last pane closes |
| Pane | One pty | Index (0-based) | Until process exits |

### Pseudo-Terminal (pty) Pairs

Each pane creates a **pty pair**: master + slave.

```
tmux server ←→ pty master ←→ pty slave ←→ bash/vim/etc.
```

The pty emulates a terminal — the child process (bash) thinks it's talking to a real terminal. tmux interprets terminal escape sequences from the child and renders them to the client.

---

## 3. Key Bindings and Prefix Key

### The Prefix Model

All tmux commands start with a **prefix key** (default: `Ctrl-b`):

$$\text{Prefix} + \text{Key} \to \text{tmux command}$$

### Essential Bindings

| Keys | Command | Action |
|:-----|:--------|:-------|
| `C-b c` | `new-window` | Create new window |
| `C-b ,` | `rename-window` | Rename current window |
| `C-b n` / `C-b p` | `next-window` / `previous-window` | Navigate windows |
| `C-b 0-9` | `select-window -t N` | Jump to window N |
| `C-b %` | `split-window -h` | Split pane horizontally |
| `C-b "` | `split-window -v` | Split pane vertically |
| `C-b o` | `select-pane -t +1` | Cycle to next pane |
| `C-b x` | `kill-pane` | Close current pane |
| `C-b d` | `detach-client` | Detach from session |
| `C-b [` | `copy-mode` | Enter copy/scroll mode |
| `C-b :` | command prompt | Enter tmux command |
| `C-b z` | `resize-pane -Z` | Zoom pane (toggle fullscreen) |

### Custom Bindings

```tmux
# In ~/.tmux.conf:
set -g prefix C-a                    # change prefix to Ctrl-a
bind -n M-h select-pane -L           # Alt-h: move left (no prefix)
bind -n M-j select-pane -D           # Alt-j: move down
bind r source-file ~/.tmux.conf      # prefix + r: reload config
```

---

## 4. Layout Algorithms

### Five Built-in Layouts

| Layout | Description | Key |
|:-------|:-----------|:----|
| `even-horizontal` | Equal-width columns | `C-b M-1` |
| `even-vertical` | Equal-height rows | `C-b M-2` |
| `main-horizontal` | Large top pane + row below | `C-b M-3` |
| `main-vertical` | Large left pane + column right | `C-b M-4` |
| `tiled` | Equal-sized grid | `C-b M-5` |

### Layout String Format

tmux encodes layouts as a string:

```
checksum,WxH,xoff,yoff{sublayouts}
```

Example for 2 vertical panes in 200x50 terminal:
```
a]e1,200x50,0,0[200x25,0,0,200x24,0,26]
```

### Manual Resizing

```
C-b C-arrow    # resize pane by 1 cell
C-b M-arrow    # resize pane by 5 cells
```

Or via command: `resize-pane -D 10` (down 10 rows).

---

## 5. Copy Mode and Scrollback

### The Scrollback Buffer

Each pane maintains a **history buffer** (default: 2000 lines):

```tmux
set -g history-limit 50000    # increase to 50K lines
```

### Copy Mode

`C-b [` enters copy mode (vi or emacs keybindings):

| vi Binding | Action |
|:-----------|:-------|
| `Space` | Start selection |
| `Enter` | Copy selection + exit |
| `q` / `Esc` | Exit copy mode |
| `/` | Search forward |
| `?` | Search backward |
| `v` | Toggle rectangle selection |

### Clipboard Integration

```tmux
# On macOS:
bind -T copy-mode-vi Enter send -X copy-pipe-and-cancel "pbcopy"

# On Linux (X11):
bind -T copy-mode-vi Enter send -X copy-pipe-and-cancel "xclip -selection clipboard"
```

---

## 6. Scripting and Automation

### tmux Commands

Every operation has a corresponding command:

```bash
# Create session:
tmux new-session -d -s work

# Create windows:
tmux new-window -t work:1 -n editor
tmux new-window -t work:2 -n server

# Send keystrokes:
tmux send-keys -t work:1 "vim ." Enter
tmux send-keys -t work:2 "npm start" Enter

# Split panes:
tmux split-window -t work:2 -h

# Attach:
tmux attach -t work
```

### Target Syntax

```
session:window.pane
```

| Target | Meaning |
|:-------|:--------|
| `work` | Session named "work" |
| `work:1` | Window 1 in session "work" |
| `work:1.0` | Pane 0 of window 1 in session "work" |
| `:1` | Window 1 of current session |
| `:.1` | Pane 1 of current window |

### Hooks

```tmux
set-hook -g after-new-window "send-keys 'echo hello' Enter"
set-hook -g session-created "display-message 'New session!'"
```

---

## 7. Environment and Terminal Interaction

### Terminal Emulation

tmux emulates a **VT100/xterm** terminal for child processes:

```tmux
set -g default-terminal "tmux-256color"
```

### The `TERM` Variable Chain

```
Real terminal (e.g., iTerm2: xterm-256color)
  └── tmux (sets TERM=tmux-256color for children)
       └── bash (sees TERM=tmux-256color)
            └── vim (uses terminfo for tmux-256color)
```

### True Color (24-bit)

```tmux
set -ag terminal-overrides ",xterm-256color:RGB"
set -ag terminal-overrides ",*-256color:Tc"
```

### Undercurl and Other Features

```tmux
set -as terminal-overrides ',*:Smulx=\E[4::%p1%dm'
set -as terminal-overrides ',*:Setulc=\E[58::2::%p1%{65536}%/%d::%p1%{256}%/%{255}%&%d::%p1%{255}%&%d%;m'
```

---

## 8. tmux vs screen

| Feature | tmux | screen |
|:--------|:-----|:-------|
| Architecture | Client-server (clean) | Monolithic |
| Pane splits | Horizontal + vertical | Vertical only (recent: both) |
| Scripting | Rich command language | Limited |
| Status bar | Highly customizable | Basic |
| Copy mode | vi or emacs bindings | Screen-specific |
| Mouse support | Yes | Limited |
| Configuration | `~/.tmux.conf` | `~/.screenrc` |
| Active development | Yes | Minimal |
| True color | Yes | No |

---

## 9. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Architecture | Client-server via Unix domain socket |
| Persistence | Server survives client disconnect |
| Hierarchy | Server → Session → Window → Pane → pty |
| Prefix | `C-b` (default), then command key |
| Layouts | 5 built-in + custom resize |
| Scrollback | Per-pane history buffer (default 2000 lines) |
| Terminal | Emulates `tmux-256color` for children |
| Scripting | Full command language, target syntax `session:window.pane` |

---

*tmux's value proposition is simple: your work survives network disconnections, terminal crashes, and going home for the night. The client-server split makes this possible — your shells, editors, and running processes live in the server, not in your terminal. Everything else (pane layouts, copy mode, scripting) is a bonus built on top of that fundamental architectural decision.*

## Prerequisites

- Terminal emulation and PTY (pseudo-terminal) concepts
- Client-server architecture (Unix domain sockets)
- Shell session management (jobs, signals, process groups)
- Key binding and prefix key conventions
