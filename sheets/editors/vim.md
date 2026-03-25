# Vim (Vi IMproved)

> Modal text editor — fast, ubiquitous, and endlessly configurable.

## Modes

```bash
i          # insert mode (before cursor)
a          # insert mode (after cursor)
I          # insert at beginning of line
A          # insert at end of line
o          # open line below and insert
O          # open line above and insert
v          # visual mode (character)
V          # visual mode (line)
Ctrl-v     # visual block mode
R          # replace mode
Esc        # return to normal mode
:          # command-line mode
```

## Motions

### Character and Word

```bash
h j k l          # left, down, up, right
w                # next word start
W                # next WORD start (whitespace-delimited)
b                # previous word start
B                # previous WORD start
e                # next word end
E                # next WORD end
0                # start of line
^                # first non-blank character
$                # end of line
f{char}          # forward to char (inclusive)
F{char}          # backward to char
t{char}          # forward to char (exclusive)
;                # repeat last f/F/t/T
,                # repeat last f/F/t/T (reverse)
```

### Line and Document

```bash
gg               # go to first line
G                # go to last line
42G              # go to line 42
{                # previous paragraph
}                # next paragraph
%                # matching bracket
Ctrl-d           # half page down
Ctrl-u           # half page up
Ctrl-f           # full page down
Ctrl-b           # full page up
H                # top of screen
M                # middle of screen
L                # bottom of screen
```

## Text Objects

```bash
# used with operators (d, c, y, v, etc.)
iw               # inner word
aw               # a word (includes surrounding space)
is               # inner sentence
ip               # inner paragraph
i"               # inside double quotes
a"               # around double quotes (includes quotes)
i(  or  ib       # inside parentheses
a(  or  ab       # around parentheses
i{  or  iB       # inside braces
it               # inside HTML/XML tag
i[               # inside brackets
```

### Common Combinations

```bash
diw              # delete inner word
ci"              # change inside quotes
yap              # yank around paragraph
da(              # delete around parentheses
vi{              # select inside braces
dit              # delete inside tag
```

## Operators

```bash
d                # delete
c                # change (delete + insert mode)
y                # yank (copy)
>                # indent right
<                # indent left
=                # auto-indent
gq               # format/wrap text
gu               # lowercase
gU               # uppercase
~                # toggle case (single char)
```

### Operator + Motion Examples

```bash
dd               # delete line
yy               # yank line
cc               # change entire line
D                # delete to end of line
C                # change to end of line
dw               # delete word
d$               # delete to end of line
d0               # delete to start of line
dG               # delete to end of file
dgg              # delete to start of file
3dd              # delete 3 lines
```

## Registers

```bash
"ayy             # yank line into register a
"ap              # paste from register a
"Ayy             # append yank to register a
"+y              # yank to system clipboard
"+p              # paste from system clipboard
"0p              # paste last yank (not delete)
""               # unnamed register (default)
"_dd             # delete to black hole (don't save)
:reg             # show all registers
```

## Search and Replace

```bash
/pattern         # search forward
?pattern         # search backward
n                # next match
N                # previous match
*                # search word under cursor (forward)
#                # search word under cursor (backward)

# substitute
:s/old/new/      # replace first on current line
:s/old/new/g     # replace all on current line
:%s/old/new/g    # replace all in file
:%s/old/new/gc   # replace all with confirmation
:5,20s/old/new/g # replace in lines 5-20
:'<,'>s/old/new/g # replace in visual selection

# regex
:%s/\v(\w+)/\U\1/g     # uppercase all words (very magic mode)
:%s/\s\+$//e            # remove trailing whitespace
```

## Macros

```bash
qa               # start recording macro into register a
q                # stop recording
@a               # play macro a
@@               # replay last macro
5@a              # play macro a 5 times
```

## Marks

```bash
ma               # set mark a at cursor
'a               # jump to mark a (line)
`a               # jump to mark a (exact position)
'.               # last change
''               # position before last jump
:marks           # list all marks

# special marks
`.               # position of last change
`"               # position when last exited file
`[               # start of last yank/change
`]               # end of last yank/change
```

## Splits and Tabs

### Splits

```bash
:split file      # horizontal split
:vsplit file     # vertical split
Ctrl-w s         # split current window horizontal
Ctrl-w v         # split current window vertical
Ctrl-w h/j/k/l  # navigate between splits
Ctrl-w H/J/K/L  # move split to edge
Ctrl-w =         # equalize split sizes
Ctrl-w _         # maximize height
Ctrl-w |         # maximize width
Ctrl-w q         # close split
Ctrl-w o         # close all splits except current
Ctrl-w r         # rotate splits
```

### Tabs

```bash
:tabnew file     # open file in new tab
:tabn            # next tab
:tabp            # previous tab
:tabclose        # close tab
gt               # next tab
gT               # previous tab
2gt              # go to tab 2
```

## Visual Mode

```bash
v                # character-wise selection
V                # line-wise selection
Ctrl-v           # block selection
gv               # reselect last visual selection

# after selecting:
d                # delete selection
c                # change selection
y                # yank selection
>                # indent
<                # outdent
:                # command on selection
U                # uppercase
u                # lowercase
J                # join lines
```

### Block Mode Tricks

```bash
# insert text on multiple lines:
Ctrl-v           # enter block mode
jjj              # select lines
I                # insert at block start
# type text
Esc              # apply to all lines

# append to multiple lines:
Ctrl-v jjj $     # select to end of lines
A                # append
# type text
Esc
```

## Folds

```bash
zf{motion}       # create fold
zo               # open fold
zc               # close fold
za               # toggle fold
zR               # open all folds
zM               # close all folds
zd               # delete fold
:set foldmethod=indent   # fold by indentation
:set foldmethod=syntax   # fold by syntax
```

## Ex Commands

```bash
:w               # save
:q               # quit
:wq              # save and quit
:x               # save and quit (only writes if changes)
:q!              # quit without saving
:e file          # edit file
:r file          # read file into buffer
:r !cmd          # read command output into buffer
:!cmd            # run shell command
:.!cmd           # replace line with command output
:%!sort          # sort entire file
:set nu          # show line numbers
:set rnu         # relative line numbers
:set paste       # paste mode (no auto-indent)
:noh             # clear search highlighting
:earlier 5m      # undo to 5 minutes ago
:later 5m        # redo to 5 minutes later
:sort            # sort selected/all lines
:sort u          # sort and remove duplicates
```

## File Navigation

```bash
:e .             # open file browser (netrw)
gf               # open file under cursor
Ctrl-^           # toggle between last two files
:ls              # list buffers
:b2              # switch to buffer 2
:bn              # next buffer
:bp              # previous buffer
:bd              # delete buffer (close file)
```

## Tips

- Use `.` to repeat the last change. Structure edits as repeatable operations.
- `ciw` (change inner word) is usually better than `diwi` -- fewer keystrokes, one undo step.
- `Ctrl-o` and `Ctrl-i` jump backward and forward through the jump list.
- `:set hlsearch` highlights matches; `:noh` clears until next search.
- In insert mode, `Ctrl-r "` pastes from the unnamed register. `Ctrl-r =` evaluates an expression.
- `g;` and `g,` move through the change list -- jump to where you last edited.
- `ZZ` is equivalent to `:wq`, `ZQ` is equivalent to `:q!`.
- Use `Ctrl-a` and `Ctrl-x` to increment/decrement numbers under the cursor.
- `:%!jq .` formats JSON in the current buffer (requires jq).
- `:set scrolloff=5` keeps 5 lines of context above/below the cursor.
