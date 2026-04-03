# The Internals of GNU Screen — Terminal Multiplexing and Session Management

> *GNU Screen is the original terminal multiplexer (1987). It creates a layer between the user's terminal and their shell processes, providing session persistence (detach/reattach), multiple virtual terminals (windows), and copy/scrollback. While largely superseded by tmux, Screen remains ubiquitous on legacy systems and has unique features (serial port access, multi-user sessions with ACLs).*

---

## 1. Architecture

### Process Model

```
User's terminal
    │
    └── screen (manager process)
           │
           ├── Window 0: pty ←→ bash
           ├── Window 1: pty ←→ vim
           └── Window 2: pty ←→ ssh
```

Unlike tmux's client-server model, Screen uses a **single process** that forks child processes for each window. The Screen process itself is what you detach from and reattach to.

### Session Persistence

```bash
screen -S work          # Create named session
# ... do work ...
Ctrl-a d                # Detach (screen keeps running)

# Later (even from a different machine):
screen -r work          # Reattach
```

### Socket Files

Screen sessions are tracked via socket files:

```
/run/screen/S-username/PID.sessionname
/tmp/screens/S-username/PID.sessionname  (alternative location)
```

---

## 2. The Escape Key

### Default: Ctrl-a

All Screen commands begin with the **escape character** (default `Ctrl-a`):

$$\text{C-a} + \text{key} \to \text{screen command}$$

### Conflict with Bash

`Ctrl-a` is "beginning of line" in Bash/Emacs mode. In Screen:
- `C-a a` sends a literal `Ctrl-a` to the terminal
- This is the #1 complaint about Screen (tmux chose `Ctrl-b` to avoid this)

### Changing the Escape Key

```screenrc
# In ~/.screenrc:
escape ^Oo       # use Ctrl-o as escape key
```

---

## 3. Essential Commands

### Window Management

| Keys | Command | Action |
|:-----|:--------|:-------|
| `C-a c` | `screen` | Create new window |
| `C-a n` | `next` | Next window |
| `C-a p` | `prev` | Previous window |
| `C-a 0-9` | `select N` | Jump to window N |
| `C-a "` | `windowlist` | Show window list |
| `C-a A` | `title` | Rename window |
| `C-a k` | `kill` | Kill current window |
| `C-a '` | — | Jump to window by number/name |

### Session Management

| Keys | Command | Action |
|:-----|:--------|:-------|
| `C-a d` | `detach` | Detach from session |
| `C-a D D` | `pow_detach` | Detach and logout |
| `C-a \` | `quit` | Kill all windows and exit |
| `C-a ?` | `help` | Show key bindings |
| `C-a :` | — | Command prompt |

### Split Windows (Regions)

| Keys | Action |
|:-----|:-------|
| `C-a S` | Split horizontally (top/bottom) |
| `C-a \|` | Split vertically (left/right) |
| `C-a Tab` | Switch to next region |
| `C-a X` | Close current region |
| `C-a Q` | Close all regions except current |

Note: After splitting, the new region is **empty** — you must select/create a window in it with `C-a n` or `C-a c`.

---

## 4. Copy and Scrollback Mode

### Enter Copy Mode

`C-a [` or `C-a Esc` enters **copy/scrollback mode**.

### Navigation (Vi-Like)

| Key | Action |
|:----|:-------|
| `h/j/k/l` | Move cursor |
| `Ctrl-u/d` | Half-page up/down |
| `Ctrl-b/f` | Full page up/down |
| `0` / `$` | Beginning/end of line |
| `H/M/L` | Top/middle/bottom of screen |
| `/pattern` | Search forward |
| `?pattern` | Search backward |
| `n` / `N` | Next/previous match |

### Copy Operation

1. Press `Space` to set start mark
2. Move cursor to end
3. Press `Space` to copy to Screen buffer

### Paste

`C-a ]` pastes the Screen buffer.

### Scrollback Buffer Size

```screenrc
defscrollback 10000     # 10,000 lines per window (default: 100)
```

---

## 5. Status Line (Hardstatus)

### Configuration

```screenrc
hardstatus alwayslastline
hardstatus string '%{= kw}%-w%{= BW}%n %t%{-}%+w %-= %D %c'
```

### Format Escapes

| Escape | Meaning |
|:-------|:--------|
| `%n` | Window number |
| `%t` | Window title |
| `%w` | All window numbers/names |
| `%-w` | Windows before current |
| `%+w` | Windows after current |
| `%D` | Day of week |
| `%c` | Current time (HH:MM) |
| `%H` | Hostname |
| `%l` | Load average |
| `%{= XY}` | Color: X=background, Y=foreground |

### Color Codes

| Code | Color |
|:-----|:------|
| k | Black |
| r | Red |
| g | Green |
| y | Yellow |
| b | Blue |
| m | Magenta |
| c | Cyan |
| w | White |
| Capital = bold/bright |

---

## 6. Multi-User Sessions

### Screen's Unique Feature

Screen supports **multi-user mode** — multiple users can view and interact with the same session:

```bash
# User A creates session:
screen -S shared

# Inside screen:
C-a :multiuser on
C-a :acladd bob         # grant access to user bob

# User B attaches:
screen -x alice/shared   # attach to alice's session named "shared"
```

### Access Control Lists (ACLs)

| Command | Effect |
|:--------|:-------|
| `aclchg user perms list` | Change permissions |
| `acladd user` | Grant full access |
| `acldel user` | Revoke access |

Permissions: `+rwx` (read, write, execute per window).

### Use Cases

- **Pair programming:** Two developers in the same session
- **Support:** Admin watching user's terminal
- **Teaching:** Instructor demonstrates in shared session

---

## 7. Serial Port Access

### Screen as Serial Terminal

```bash
screen /dev/ttyUSB0 115200
```

This connects Screen directly to a serial port at 115200 baud. Screen acts as a terminal emulator for serial communication.

| Parameter | Syntax |
|:----------|:-------|
| Device | `/dev/ttyUSB0`, `/dev/ttyS0`, `/dev/cu.usbserial` |
| Baud rate | `9600`, `115200`, etc. |
| Data bits | `cs8` (8-bit, default) |
| Parity | `parenb` (even), `-parenb` (none) |
| Stop bits | `cstopb` (2), `-cstopb` (1) |

### Exit Serial Session

`C-a k` (kill window) or `C-a \` (quit Screen).

This feature is why Screen is still used by embedded systems engineers, even when tmux is preferred for general use.

---

## 8. screenrc Configuration

### Essential Configuration

```screenrc
# Disable startup message:
startup_message off

# Visual bell instead of audible:
vbell on

# Scrollback buffer:
defscrollback 10000

# Status line:
hardstatus alwayslastline
hardstatus string '%{= kw}%-w%{= BW}%n %t%{-}%+w'

# Default shell:
shell /bin/bash

# Automatically detach on hangup:
autodetach on

# UTF-8 support:
defutf8 on

# Window creation:
screen -t shell 0
screen -t editor 1 vim
screen -t logs 2 tail -f /var/log/syslog
```

### Startup Windows

```screenrc
screen -t main 0 bash
screen -t code 1 bash
screen -t logs 2 bash
select 0                # start in window 0
```

---

## 9. Screen vs tmux

| Feature | Screen | tmux |
|:--------|:-------|:-----|
| First release | 1987 | 2007 |
| Architecture | Monolithic | Client-server |
| Pane splits | Horizontal + vertical | Horizontal + vertical |
| Pane persistence | Split regions lost on detach (pre-4.9) | Preserved |
| Multi-user | Built-in with ACLs | Via socket permissions |
| Serial port | Built-in | Not built-in |
| Scripting | Limited | Rich command language |
| True color | No | Yes |
| Active development | Minimal | Active |
| Installed by default | Many Linux distros | macOS (via Homebrew) |
| Configuration | `~/.screenrc` | `~/.tmux.conf` |

### When Screen is Still the Right Choice

| Scenario | Why |
|:---------|:----|
| Legacy system with no tmux | Screen is more widely pre-installed |
| Serial port communication | Built-in serial terminal |
| Multi-user pair programming | ACL-based multi-user sessions |
| Simple detach/reattach only | Screen is simpler for basic use |

---

## 10. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Architecture | Single process with child ptys (not client-server) |
| Escape key | `C-a` (conflicts with Bash beginning-of-line) |
| Session persistence | Detach: `C-a d`, Reattach: `screen -r` |
| Windows | Virtual terminals within a session |
| Regions | Split screen areas (lost on detach in older versions) |
| Scrollback | Copy mode with vi-like navigation |
| Multi-user | Built-in ACL system (unique feature) |
| Serial port | `screen /dev/ttyUSB0 115200` (unique feature) |

---

*Screen's place in computing history is secure: it invented terminal multiplexing. Its multi-user ACL system and serial port support remain unique features that tmux doesn't replicate. For everything else — pane management, scripting, modern terminal features — tmux has surpassed it. Know both: Screen for legacy systems and serial work, tmux for daily use.*
