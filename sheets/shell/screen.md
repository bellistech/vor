# GNU Screen (Terminal Multiplexer)

> Persistent terminal sessions with window management and detach/reattach — the original multiplexer.

## Sessions

### Management

```bash
screen -S work                 # new session named "work"
screen -ls                     # list sessions
screen -r work                 # reattach to session
screen -r                      # reattach (if only one session)
screen -d -r work              # detach elsewhere, reattach here
screen -D -R work              # force detach + reattach (nuke other clients)
screen -x work                 # multi-display: attach without detaching others
screen -S work -X quit         # kill session from outside
```

### Key Bindings (prefix = Ctrl-a)

```bash
# Ctrl-a then:
d                  # detach from session
\                  # kill all windows and quit (with confirmation)
:sessionname new   # rename session
```

## Windows

### Management

```bash
# Ctrl-a then:
c                  # create new window
A                  # rename current window
k                  # kill current window (with confirmation)
n                  # next window
p                  # previous window
0-9                # switch to window 0-9
"                  # list windows (interactive)
w                  # show window bar
'                  # prompt for window number/name to switch to
```

### From Command Line

```bash
screen -S work -X screen -t logs          # create window named "logs"
screen -S work -p 0 -X stuff "ls\n"       # send command to window 0
screen -S work -X select 2                 # switch to window 2
```

## Split Regions

### Splitting

```bash
# Ctrl-a then:
S                  # split horizontal (top/bottom)
|                  # split vertical (left/right) — requires version 4.1+
tab                # move focus to next region
X                  # close current region
Q                  # close all regions except current
```

### Working with Regions

```bash
# after splitting, the new region is empty
# Ctrl-a then:
tab                # switch to new region
c                  # create window in it (or select existing with 0-9)
```

## Scrollback and Copy Mode

### Scrollback

```bash
# Ctrl-a then:
[                  # enter scrollback/copy mode
Escape             # exit copy mode

# in copy mode (vi-like by default):
h j k l            # navigate
Ctrl-u             # page up
Ctrl-d             # page down
/                  # search forward
?                  # search backward
```

### Copy and Paste

```bash
# in copy mode:
Space              # start selection
Enter              # copy selection and exit

# Ctrl-a then:
]                  # paste buffer
>                  # write paste buffer to file
<                  # read file into paste buffer
```

## Logging and Monitoring

```bash
# Ctrl-a then:
H                  # toggle logging to screenlog.N file
h                  # write hardcopy of current window to file
M                  # toggle activity monitoring (notify on output)
_                  # toggle silence monitoring (notify on no output)
```

### From Command Line

```bash
screen -L -S work              # start session with logging enabled
screen -S work -X hardcopy -h /tmp/screendump.txt  # capture with scrollback
```

## Configuration (~/.screenrc)

### Common Settings

```bash
# disable startup message
startup_message off

# large scrollback buffer
defscrollback 10000

# visual bell instead of audio
vbell on

# UTF-8 support
defutf8 on

# status line
hardstatus alwayslastline
hardstatus string '%{= kG}[ %{G}%H %{g}][ %{=kw}%?%-Lw%?%{r}(%{W}%n*%f%t%?(%u)%?%{r})%{w}%?%+Lw%?%?%= %{g}][ %{B}%Y-%m-%d %{W}%c %{g}]'

# default shell
shell -${SHELL}

# auto-detach on hangup
autodetach on

# no flow control (frees up Ctrl-s / Ctrl-q)
defflow off

# use vi keys in copy mode
markkeys "^M=y:$=A:^[=q"
```

### Key Remapping

```bash
# remap prefix to Ctrl-j
escape ^Jj

# common bindings
bind j focus down
bind k focus up
bind h focus left
bind l focus right
```

### Startup Windows

```bash
# auto-create windows on startup
screen -t editor 0 vim
screen -t shell  1 bash
screen -t logs   2 tail -f /var/log/syslog
select 0
```

## Multi-User Mode

```bash
# enable multi-user (in screenrc or runtime)
multiuser on
acladd alice                   # grant access to user alice
aclchg bob -w "#"              # bob can see but not write to any window
acldel carol                   # revoke access

# alice attaches
screen -x youruser/work
```

## Useful Commands

### Command Mode (Ctrl-a :)

```bash
:info                          # display terminal info
:number 5                      # move current window to position 5
:title newname                 # rename current window
:resize 20                     # resize region to 20 lines
:split                         # same as Ctrl-a S
:only                          # same as Ctrl-a Q
:quit                          # kill session
:source ~/.screenrc            # reload config
```

### From Command Line

```bash
# send a command to a running session
screen -S work -p 0 -X stuff "deploy.sh\n"

# set window title
screen -S work -p 0 -X title "builder"

# capture window output
screen -S work -p 0 -X hardcopy /tmp/window0.txt

# check if session exists
screen -ls | grep -q "work" && echo "running"
```

## 256 Color Support

```bash
# in .screenrc
term screen-256color
# or start with
screen -T screen-256color
```

## Tips

- The prefix is Ctrl-a by default, which conflicts with bash's "go to beginning of line". Use Ctrl-a a to send a literal Ctrl-a.
- `screen -D -R` is the most aggressive reattach -- it will detach and logout other attached clients.
- Always name sessions with `-S name`. Otherwise you get PIDs as identifiers.
- Screen splits (regions) are lost on detach. If you need persistent splits, use tmux instead.
- `screen -x` allows multiple terminals to share the same session simultaneously -- good for pair programming.
- The scrollback buffer default is only 100 lines. Set `defscrollback 10000` in .screenrc.
- Screen does not support true color (24-bit). If you need that, use tmux.
- `Ctrl-a :fit` resizes the current window to the terminal dimensions -- useful after reattaching from a different-sized terminal.
- To convert from screen to tmux: sessions map 1:1, and most concepts are similar, but the config syntax is completely different.
- If Ctrl-s freezes your terminal (flow control), Ctrl-q unfreezes it. Disable with `defflow off`.
