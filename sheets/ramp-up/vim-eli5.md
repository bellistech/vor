# Vim — ELI5

> Vim is a piano for text. Most editors are typewriters: press a letter, the letter shows up. Vim is different: in one mode, your keys are commands, and in another mode, your keys are letters. Learning the chords takes a while. Once your fingers know the chords, editing is faster than thinking.

## Prerequisites

You should have already read **`ramp-up/bash-eli5`** and **`ramp-up/git-eli5`**, because we are going to be living inside a terminal in this sheet, and we are going to assume you can move around the file system and run commands. You do not need to know anything else. You do not need to know what an "editor" is. You do not need to know what a "mode" is. You do not need to have ever used vim. By the end of this sheet you will have edited real files, recorded a real macro, jumped around a buffer, and written your first `.vimrc`.

If a word is weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English explanation. If a key is weird, look it up. If an error message looks scary, find it in **Common Errors**. The exact text is there with the cause and the fix.

If you see a `$` at the start of a line in a code block, you type the rest of that line into your terminal — you do not type the `$`. If you see a `:` at the start of a line in a code block, you press the `:` key inside vim, then you type the rest of the line, then you press Enter. If you see something like `dw` in a code block, you press `d` then `w` while in normal mode. We will explain what "normal mode" is in a minute. Hold tight.

## What Even Is Vim

### A typewriter vs a piano

Imagine you sit down at a typewriter. You press the `H` key. The letter `H` appears on the page. You press the `e` key. The letter `e` appears on the page. You press space. A space appears. Every key on the typewriter does one job: it puts a letter on the page. The typewriter has a few special keys, like Backspace and Shift, but mostly the keys all do the obvious thing.

Almost every text editor on your computer works like a typewriter. Notepad on Windows works like a typewriter. TextEdit on macOS works like a typewriter. The little notes app on your phone works like a typewriter. You press a letter, the letter shows up. If you want to do something fancy like delete a word or jump to the end of the file, you have to take your hands off the home row, grab the mouse, drag-select, then press Delete. It works. It is fine. It is also slow if you spend hours every day editing text, and a programmer spends hours every day editing text.

Now imagine you sit down at a piano. You press the `H` key. Wait. There is no `H` key on a piano. There are 88 keys, and each one makes a sound, and to play a song you have to learn which keys to press in which order, and to play a beautiful song you have to press several keys at once — a chord. The piano is much harder than the typewriter. Nobody sits down at a piano for the first time and plays a song. You have to practice. For weeks. For months. You have to learn that this finger goes here and that finger goes there and these three fingers come down at the same time to make this chord.

But once you know the piano, you can play any song. You can write your own songs. Your hands move so fast you do not even think about which keys you are pressing — your fingers know. The piano can do things the typewriter cannot. The piano is more expressive. The piano is faster, once you have paid the price.

**Vim is the piano. Notepad is the typewriter.**

### So what does that actually mean

In a typewriter editor, every key prints a letter. The `j` key prints the letter `j`. The `w` key prints the letter `w`. The `d` key prints the letter `d`. There is one mode, and in that one mode, keys mean letters.

In vim, the `j` key does not print the letter `j`. Not by default. By default, vim is in **normal mode**, and in normal mode the `j` key means **"move the cursor down one line."** The `w` key means **"jump to the next word."** The `d` key means **"delete (something)."** The `h`, `j`, `k`, `l` keys are arrow keys. The `0` key jumps to the start of the line. The `$` key jumps to the end of the line. The keys are commands.

How do you actually type a letter, then? You press `i`, which means **"go into insert mode."** Insert mode is the typewriter mode. In insert mode, the keys behave like a normal editor: press `H`, get `H`. When you are done typing, you press the **Escape** key, and you are back in normal mode, where the keys are commands again.

That is the whole trick. Vim has different **modes**. The same key does different things in different modes. In normal mode you are giving the editor orders. In insert mode you are typing letters. Most of your time is spent in normal mode, even though that feels backwards when you first hear it. You enter insert mode briefly to type a few words, then you escape back to normal mode to navigate, search, delete, copy, paste, save, and so on.

### Why on earth would anyone want this

Because in normal mode, every key on the keyboard is doing useful work. You are not pressing arrow keys. You are not reaching for the mouse. You are not holding down Ctrl and Shift to do shortcuts. Your fingers are on the home row, where they always are, and your fingers are running the editor with single keystrokes.

Once you know the chords, this is so fast that nothing else feels right. Want to delete the word your cursor is sitting on? `daw` — three letters: delete-around-word. Want to delete from here to the end of the line? `D`. Want to change the contents of the parentheses your cursor is in? `ci(` — change-inner-parenthesis. Want to jump to the line where you last edited? `'.` — apostrophe-period.

In a typewriter editor, those operations are: select with the mouse, delete, retype. Or: hold Shift and Ctrl and arrow-key your way through. Or: open a menu, find the right command.

In vim, those operations are: three keys. Done. Hands never moved.

This is the deal. The investment is upfront. You will be slow for a week. You will be slow for two weeks. Then one day you will realize your fingers are doing things without you telling them to, and from that day on, editing text is faster than thinking, forever.

### Vim, vi, neovim — what's the difference

You will see all three names. They are all the same family.

**vi** is the original. It was written in 1976 by Bill Joy at Berkeley. `vi` stands for "visual," because the original Unix editor before vi was a line editor called `ed` that did not show you the file you were editing, and `vi` was a huge improvement: you could actually see the file. It was on every Unix system from then on. The `vi` you find on a modern macOS or Linux machine is usually a thin compatibility shim — it runs a stripped-down version of vim in `vi` compatibility mode.

**vim** is "vi improved." Bram Moolenaar wrote it in 1991. It is vi with everything added: syntax highlighting, plugins, multiple windows, undo, scripting, the works. Vim 9 (released 2022) is the current vim. Vim has its own scripting language called Vimscript, and Vim 9 added a faster compiled version called Vim9script.

**neovim** (or `nvim`) is a fork of vim that started in 2014. The fork happened because vim's codebase had grown old and creaky and Bram, who maintained vim almost alone for 30 years, was not accepting big changes. Neovim's goals: clean up the code, add a real plugin API, embed Lua as the configuration language, support asynchronous I/O, and ship a built-in language server protocol (LSP) client and tree-sitter parser. Neovim is what most new vim users today are actually running. If you are starting now, install neovim.

For everything in this sheet, "vim" means the family. Where there is a real difference between vim 9 and neovim, we will say so.

There is also **gvim** (vim with a graphical window — like a normal app, not in a terminal), **mvim** (the Mac graphical version), and various plugins that embed vim keybindings into VS Code, JetBrains IDEs, Emacs (`evil-mode`), and even web browsers (Vimium). The keybindings are universal. Once you learn vim, you bring it everywhere.

## The Mode System

This is the most important section of the sheet. If you do not understand modes, you will not understand vim. Read it twice. Maybe three times.

### The modes

Vim has these modes:

- **Normal mode** — keys are commands. This is where you start. This is where you spend most of your time. Press `Esc` to get here from anywhere.
- **Insert mode** — keys are letters. Press `i` from normal mode to get here. Press `Esc` to get back.
- **Visual mode** — character-by-character selection. Press `v` from normal mode. Move with motions. Apply an operator (like `d` to delete) to the selection.
- **Visual-line mode** — line-by-line selection. Press `V` (capital V).
- **Visual-block mode** — rectangular selection. Press `Ctrl-v` (or `Ctrl-q` on Windows). Lets you edit columns.
- **Command-line mode** — for running ex commands. Press `:` from normal mode. Type a command (like `:w` to save), press Enter. Also reachable with `/` (search forward) and `?` (search backward).
- **Replace mode** — typed letters overwrite existing letters instead of inserting before them. Press `R` (capital) from normal mode. Press `Esc` to leave.
- **Virtual replace mode** — like replace, but tabs are treated as spaces. Press `gR`. Less common.
- **Terminal mode** (neovim only) — when you have a terminal open inside neovim with `:term`, that buffer has its own mode where keys go to the shell. Press `Ctrl-\ Ctrl-n` to escape back to normal mode.

### The mode state machine (ASCII diagram)

```
                       ┌─────────────────┐
                       │                 │
                       │   NORMAL MODE   │  <-- start here, return here
                       │  (commands)     │
                       │                 │
                       └────────┬────────┘
                                │
        ┌──────────┬────────────┼────────────┬──────────┐
        │          │            │            │          │
   i,a,o,O,A,I    v,V,Ctrl-v   :, /, ?       R          gR
        │          │            │            │          │
        ▼          ▼            ▼            ▼          ▼
   ┌────────┐  ┌────────┐  ┌──────────┐  ┌────────┐  ┌────────┐
   │ INSERT │  │ VISUAL │  │ COMMAND- │  │REPLACE │  │VIRTUAL │
   │ (typing│  │ (select│  │  LINE    │  │(over-  │  │REPLACE │
   │  text) │  │  text) │  │ (ex cmd) │  │ write) │  │        │
   └────┬───┘  └────┬───┘  └────┬─────┘  └────┬───┘  └────┬───┘
        │          │           │             │           │
        └──────────┴───────────┴─────────────┴───────────┘
                          Esc returns
                       to NORMAL MODE
```

So you are always pressing `Esc` to get back home. Home is normal mode. Always. If you are not sure what mode you are in, press `Esc`. It is free. You cannot break anything. After enough `Esc` presses, you are in normal mode.

(Look at the bottom of the screen — vim shows the mode there: `-- INSERT --`, `-- VISUAL --`, `-- REPLACE --`. If nothing shows, you are in normal mode.)

### Ways to enter insert mode

There are several ways to switch from normal to insert mode. Each one starts inserting at a different place. This matters more than it seems — if you pick the right one, you save a movement.

```
i   insert before the cursor
a   append after the cursor (insert after this character)
I   insert at the start of the line (after leading whitespace)
A   append at the end of the line
o   open a new line below and start inserting
O   open a new line above and start inserting
gi  resume insert at the last place you left insert mode
s   delete the character under the cursor and insert
S   delete the whole line and insert
cc  change line (= S)
C   change to end of line
```

Beginners use `i` for everything. Then they reach for the arrow keys. Don't. Use `A` to append at the end of a line. Use `o` to open a new line below. Use `I` to insert at the start. Less movement = faster.

### Why Modal Editing

Once more, with feeling: in a typewriter editor, the keys mean letters. To do anything useful, you have to combine keys with modifiers — Shift, Ctrl, Alt, Cmd. So your shortcut to delete a word is something like `Ctrl-Shift-Right, Delete`. Three keys, two modifiers, fingers contorted across the keyboard.

In vim, the keys mean commands. So the shortcut to delete a word is `dw`. Two keys. No modifiers. Both keys on the home row.

Multiply that by every operation you do all day. Hundreds of edits. Thousands of cursor movements. Modal editing is faster because the alphabet is the menu. Every letter on the keyboard is a command. You do not need modifiers because you are already in "command mode."

The price you pay: you have to learn which key does what. The reward you get: every key does something useful, all the time.

## Movement (Motions)

In normal mode, your job is to move the cursor and tell vim what to do. Movement is the foundation. If you can move precisely, you can edit precisely.

### The four arrow keys are on the home row

```
h   move cursor LEFT
j   move cursor DOWN
k   move cursor UP
l   move cursor RIGHT
```

Yes, your hand never moves. `j` is on the home row. So is `k`. So is `l`. So is `h` (one key to the left of `j`). The right hand sits on the home row and the four most-used keys are right there.

(The reason `h, j, k, l` and not the obvious arrow keys is that the original vi was written on an ADM-3A terminal where the arrows were *printed on those four keys.* No, really, look up a picture of an ADM-3A keyboard. The arrows are on `h, j, k, l`.)

### Word motions

```
w   forward to next WORD start (word = letters/digits/underscore)
b   backward to previous word start
e   forward to end of word
ge  backward to end of previous word
W   forward to next big-WORD start (big-WORD = whitespace-separated)
B   backward big-WORD
E   forward end of big-WORD
gE  backward end of big-WORD
```

The lowercase versions stop at punctuation. Uppercase versions only stop at whitespace. So in `hello-world`, lowercase `w` would treat `hello`, `-`, and `world` as three separate things. Uppercase `W` would treat the whole `hello-world` as one big-WORD.

### Line motions

```
0    jump to column 0 (very start of line, before whitespace)
^    jump to first non-whitespace character
$    jump to end of line
g_   jump to last non-whitespace character of line
|    jump to column 1 (or :  17|  jumps to column 17)
```

### File motions

```
gg   jump to the top of the file (line 1)
G    jump to the bottom of the file
17G  jump to line 17  (also: :17 then Enter)
%    jump to matching bracket (), {}, []
H    jump to top of visible screen
M    jump to middle of visible screen
L    jump to bottom of visible screen
zt   scroll so cursor is at TOP of screen
zz   scroll so cursor is at MIDDLE of screen
zb   scroll so cursor is at BOTTOM of screen
```

`%` is brilliant. Cursor on a `(`, press `%`, cursor jumps to the matching `)`. Same for `{}` and `[]`. Same for `/* ... */` C comments. Same for `#if/#endif`. Saves your eyes from scanning.

### Find character

```
f<char>  jump forward to next occurrence of <char> on this line
F<char>  jump backward
t<char>  jump forward to just BEFORE next <char> (till)
T<char>  jump backward to just after
;        repeat last f/F/t/T forward
,        repeat last f/F/t/T backward
```

Want to jump to the next semicolon? `f;`. Need to delete everything up to (but not including) the next quote? `dt"`. These are surgical.

### Search

```
/pattern   search forward for pattern, press Enter
?pattern   search backward
n          go to next match (in same direction)
N          go to next match (in opposite direction)
*          search for the word under the cursor (forward)
#          search for the word under the cursor (backward)
gd         go to local definition (often jumps to first occurrence in scope)
gD         go to global definition
:noh       turn off the highlighted matches
```

Search is a motion. That means you can combine it with operators (which we will get to). `d/foo<Enter>` deletes from here to the next `foo`.

### Jump list and change list

Every time you make a "big" jump (search, `gg`, `G`, jumps to a tag, etc.), vim records it in the **jump list**.

```
Ctrl-o   jump back to PREVIOUS location in jump list
Ctrl-i   jump forward (Ctrl-i is the same key as Tab on most terminals)
:jumps   show the jump list
```

This is your "back" button. Way better than trying to remember where you came from. Pressed `gg`, looked at line 1, want to go back? `Ctrl-o`.

There is also a **change list** — a list of every place you made a change.

```
g;       go to the previous change in this file
g,       go to the next change
:changes show the change list
'.       jump to the line where you last made a change
`.       jump to the exact position of the last change
```

The `` `. `` (backtick-period) one is gold. You just made an edit. Now you scrolled away. Want to come back to where you were typing? `` `. ``. Done.

### File jumps

```
gf       go to file under cursor (opens the path you're sitting on)
Ctrl-]   jump to tag definition (when ctags index is loaded)
Ctrl-t   pop back from a tag jump
gx       open URL under cursor in default browser
```

## Operators + Motions = Edits

This is the core grammar of vim, and it is what makes vim feel like a language instead of a list of shortcuts.

### The grammar

```
[count] operator [count] motion
```

You can read it left-to-right: "do this *operator* over this *motion*, this many times."

The operators are:

```
d   delete (and yank to register)
c   change (= delete + go to insert mode)
y   yank (copy)
>   indent (shift right)
<   outdent (shift left)
=   auto-indent
gU  uppercase
gu  lowercase
g~  swap case
gq  format (re-wrap text to 'textwidth')
!   filter through external command
```

The motions are everything from the previous section: `w`, `b`, `e`, `0`, `$`, `gg`, `G`, `%`, `f<char>`, `t<char>`, `/pattern`, etc.

### Diagram: operator + motion

```
                  ┌───────────────────────┐
                  │     OPERATOR          │
                  │   d  c  y  >  <  =    │
                  │   gU gu g~ gq !       │
                  └───────────┬───────────┘
                              │ followed by
                              ▼
                  ┌───────────────────────┐
                  │       MOTION          │
                  │  w  b  e  0  $  gg G  │
                  │  f<c>  t<c>  /pat     │
                  │  %  H  M  L  '<mark>  │
                  └───────────┬───────────┘
                              │ together they form
                              ▼
                  ┌───────────────────────┐
                  │   ONE EDIT COMMAND    │
                  │       e.g.  dw        │
                  │       e.g.  c$        │
                  │       e.g.  y%        │
                  │       e.g.  >G        │
                  └───────────────────────┘
```

Read aloud: `dw` = "delete word." `c$` = "change to end of line." `y%` = "yank to matching bracket." `>G` = "indent from here to the bottom of the file." `gUw` = "uppercase the word." `=G` = "auto-indent everything from here to the bottom."

This composability is the secret. You learn 8 operators and 20 motions. That's 160 edits. Add counts (like `3dw` = "delete 3 words") and text objects (which we'll cover next), and the combinations explode. You don't memorize the combinations — you compose them on the fly.

### Doubled operator = whole line

A common shortcut: any operator typed twice acts on the current line.

```
dd   delete current line
cc   change current line
yy   yank current line   (also: Y = yy)
>>   indent current line
<<   outdent current line
==   auto-indent current line
gUU  uppercase current line
guu  lowercase current line
```

So `dd` is "delete line." `5dd` is "delete 5 lines."

### Counts

Put a number in front to repeat:

```
3w     forward 3 words
5j     down 5 lines
10dd   delete 10 lines
2dw    delete 2 words
d3w    delete 3 words   (same — count can be on either side)
```

### Common idioms

After a few weeks, your fingers will know these without thinking:

```
dw      delete word forward
db      delete word backward
diw     delete inner word (cursor anywhere in word)
daw     delete a word (including trailing space)
dd      delete line
D       delete to end of line (= d$)
dt,     delete up to (not including) the next comma
df,     delete up to and including the next comma
ci"     change inside the double-quoted string
ci(     change inside the parens
ca{     change around the braces (including the braces)
yi)     yank inside the parens
yy      yank the line
y$      yank to end of line
p       paste after cursor
P       paste before cursor
x       delete character under cursor
X       delete character before cursor
r<c>    replace single character with <c>
~       swap case of character
.       repeat last edit
u       undo
Ctrl-r  redo
```

The dot command (`.`) is the soul of vim. Did one thing. Want to do it again? Press `.`. Move somewhere else, press `.` again. Half of vim mastery is structuring your edits so `.` does the right thing on the next instance.

## Text Objects

Motions move you. Text objects describe a chunk of text you want to operate on. They make sentences in vim's grammar more useful, because instead of saying "from here to there" you can say "this thing I'm in the middle of."

A text object has two parts: an **adjective** (`i` for inner, `a` for around) and a **noun** (the kind of object).

```
iw   inner word
aw   a word (with surrounding whitespace)
iW   inner big-WORD
aW   a big-WORD
is   inner sentence
as   a sentence
ip   inner paragraph
ap   a paragraph
i(   inner parens   (also  i)  ib )
a(   a parens (with the parens)
i{   inner braces   (also  i}  iB )
a{   a braces
i[   inner brackets
a[   a brackets
i<   inner angle brackets
a<   a angle brackets
i"   inner double-quoted string
a"   a double-quoted string
i'   inner single-quoted string
a'   a single-quoted string
i`   inner backtick string
a`   a backtick string
it   inner tag (HTML/XML)
at   a tag (HTML/XML)
```

The cursor does not have to be at the start. It just has to be *inside* the object. Cursor anywhere inside `(hello world)`, type `ci(`, and you replace `hello world` with whatever you type next, leaving the parens intact.

`daw` deletes "a word" — the word AND the whitespace around it. `diw` only deletes the word itself, leaving the spaces. Tiny difference, huge in practice.

`ci"` is the most-used text object on the planet. You are inside a string, you want to change the string contents — `ci"`, type, `Esc`. Done.

`dap` deletes a whole paragraph (block of non-blank lines plus surrounding blank lines).

`yat` yanks an entire HTML/XML tag, including the open and close tags. Magic in HTML files.

## Insert Mode Tricks

You spend less time in insert mode than you think. But there are still some keystrokes worth knowing while you are in there.

```
Ctrl-h    backspace one character
Ctrl-w    delete the previous word   <-- huge
Ctrl-u    delete back to start of line
Ctrl-t    indent line one level
Ctrl-d    outdent line one level
Ctrl-r"   paste the contents of the unnamed register (no need to leave insert)
Ctrl-r=   evaluate an expression and paste the result
           (e.g. Ctrl-r= 2+2 Enter   -> inserts 4)
Ctrl-n    autocomplete with next match (built-in completion)
Ctrl-p    autocomplete with previous match
Ctrl-x Ctrl-l   line-completion (complete a whole line)
Ctrl-x Ctrl-f   filename completion
Ctrl-x Ctrl-o   omni completion (LSP-aware in nvim)
Ctrl-o<cmd>     run ONE normal-mode command, then return to insert
Ctrl-[    same as Esc (more reachable)
```

`Ctrl-w` to nuke the last word, `Ctrl-u` to nuke the line — these are also bash bindings. Same fingers, both places.

`Ctrl-o` is the "go normal for one command" key. Inside insert mode, type `Ctrl-o dd`, that deletes the current line and you stay in insert mode. Useful when you don't want to leave insert and come back.

## Registers

Registers are vim's clipboards. Yes, plural. There is more than one. This is one of vim's superpowers.

When you yank or delete, the text goes into a register. By default, into the **unnamed** register. But you can put text into any of dozens of named registers and recall it later.

### The registers

```
"    the unnamed register (default for d, c, y, p, x)
0    the yank register (last y went here, untouched by d/c)
1-9  the numbered registers (last 9 deletes — push down)
a-z  named registers (you choose where to put it)
A-Z  same as a-z but APPENDS instead of replacing
"_   the black hole — write here to throw text away
"+   the system clipboard (X11 CLIPBOARD)
"*   the primary selection clipboard (X11 PRIMARY, middle-click)
"=   the expression register — type an expression, get the result
":   the last command executed
"/   the last search pattern
"%   the current filename
"#   the alternate filename (last buffer)
".   the last inserted text
```

To use a register, prefix any operation with `"<register>`:

```
"ayy    yank line into register a
"ap     paste from register a
"Ayy    APPEND line to register a (capital A)
"+yy    yank line into system clipboard
"+p     paste from system clipboard
"_dd    delete line WITHOUT clobbering the unnamed register (black hole)
```

### Why this is great

You yank a line. You go delete some other stuff. In a typewriter editor, the deletion has overwritten your clipboard. In vim, your yank is still in register `0` even after the delete went into the unnamed register. So `"0p` pastes the original yank.

You can also build up text by appending. `"ayy` to start, then go to another line and `"Ayy` to append. Now register `a` has both lines. Paste with `"ap`.

### View what's in the registers

```
:registers       show every register's current contents
:reg a           show only register a
:reg "0+         show only specific registers
```

### Visual register layout

```
┌───────────────────────────────────────────────────┐
│  REGISTERS                                        │
├───────────────────────────────────────────────────┤
│  "    unnamed (last d/c/y/x went here)            │
│  0    yank-only (only y goes here)                │
│  1-9  delete history (1=most recent line-delete)  │
│  a-z  named, you control                          │
│  A-Z  same as a-z but APPENDS                     │
│  +    system clipboard (X11 CLIPBOARD)            │
│  *    selection clipboard (X11 PRIMARY)           │
│  =    expression — type an expression             │
│  /    last search pattern                         │
│  :    last ex command                             │
│  .    last inserted text                          │
│  %    current filename                            │
│  #    alternate filename                          │
│  _    black hole (toss text away)                 │
└───────────────────────────────────────────────────┘
```

### The system clipboard

Out of the box, vim does not share with your operating system clipboard. To paste from your OS, you reach for `Ctrl-Shift-v` in your terminal, which pastes characters one by one as if you were typing them — usually disastrous if you are in normal mode.

Two fixes:

1. **Build vim with clipboard support.** Run `vim --version | grep clipboard`. If you see `+clipboard` you have it. If you see `-clipboard` you do not. On macOS, `brew install vim` ships with `+clipboard`. On Linux, `vim-gtk3` or `vim-gtk` packages have it. Then yank with `"+y` and paste with `"+p`.

2. **Use neovim.** Neovim does not need a clipboard build flag — it uses your system's clipboard tool (`xclip`, `xsel`, `wl-copy`, `pbcopy`, etc.) on demand. Yank with `"+y`, paste with `"+p`.

Either way, you can also tell vim to use the clipboard register by default:

```vim
set clipboard=unnamedplus
```

Now `y` and `p` use the system clipboard automatically. Most people add this and forget about it.

## Macros

A macro is a recorded sequence of keystrokes you can play back. Whatever you can do by hand, you can record. This is vim's automation. It is shockingly powerful.

```
qa          start recording into register a
   ...      (do stuff — every keystroke is recorded)
q           stop recording
@a          play back the macro in register a
@@          replay the last macro
5@a         play the macro 5 times
```

The macro is just stored in register `a`. So `:reg a` shows you what's recorded. You can edit a macro by yanking text, pasting it, editing, and re-yanking back into register `a`.

### Example: number all lines

You have 10 lines, you want each to start with `1.`, `2.`, `3.`, ... .

1. Go to the first line. Type `1. ` at the start. Move to line 2.
2. Press `qa` to start recording.
3. Type `I2. <Esc>j` — that inserts `2. ` at line start, then moves down.

Wait — that hardcodes "2." Let's do it the smart way using `Ctrl-a` (which increments a number).

1. Go to line 1. Type `1. ` at the start, leave the cursor there.
2. `qa` (start recording into a).
3. `yy` yank line, `p` paste below, `Ctrl-a` increment the number, `q` stop.
4. Now `8@a` runs that macro 8 more times. You have 1-10.

`Ctrl-a` and `Ctrl-x` are vim's increment and decrement. Cursor on a number, press `Ctrl-a`, the number goes up by 1. `5Ctrl-a` adds 5. They are macro magic.

### Edit a macro

Macros are just text. To fix one without re-recording:

```
:put a       paste register a contents into the buffer
   ...       edit the line however you want
"add         delete that line back into register a
```

Or directly:

```
:let @a = 'I2. \<Esc>j'
```

(Note: `\<Esc>` inside a string is the escape character.)

## Search and Replace

The `:s` command (substitute) is global find-and-replace.

```
:s/old/new/        replace first 'old' on current line with 'new'
:s/old/new/g       replace ALL 'old' on current line
:%s/old/new/g      replace ALL 'old' in WHOLE FILE (% = whole file)
:%s/old/new/gc     same, but ASK before each replacement (c = confirm)
:%s/\<old\>/new/g  whole-word match (\< and \> are word boundaries)
:%s/old/new/gI     case-sensitive (I) override
:%s/old/new/gi     case-insensitive (i) override
:5,15s/old/new/g   only on lines 5 through 15
:'<,'>s/old/new/g  only on the visual selection (<,> are visual marks)
```

Without the `g` flag, only the FIRST occurrence on each line is replaced. This is one of the most common gotchas. We will hit it again in **Common Confusions**.

### Patterns

Vim's patterns are similar to grep's but have some quirks. By default many regex characters are literal. The fixes:

```
\v   "very magic" — most regex chars are special (like in PCRE/sed)
\V   "very nomagic" — most chars are literal
\m   "magic" — vim default
\M   "nomagic"
```

You usually want very-magic for sanity:

```
:%s/\v(\w+)\s+\1/\1/g    collapse repeated words (uses \v, capture groups)
```

### Global commands

`:g` runs an ex command on every line that matches a pattern. `:v` (or `:g!`) runs on every line that does NOT match.

```
:g/pattern/d         delete every line matching pattern
:v/pattern/d         delete every line NOT matching pattern
:g/TODO/p            print every line matching TODO (showing them at bottom)
:g/^$/d              delete every blank line
:g/^/m0              reverse the file (move every line to line 0)
:g/X/normal @a       on every line matching X, run macro a in normal mode
```

`:g` is a sleeper hit. Once you know it, you can do batch transformations no other editor can match.

## Buffers, Windows, Tabs

These three concepts are confused all the time. Hold on.

### Buffer

A **buffer** is a loaded file. When you open a file, vim reads it into memory — that in-memory version is the buffer. You can have many buffers loaded at once. Most of them are not visible.

```
:e file.txt       edit (open) file.txt — replaces current buffer's window
:badd file.txt    add file.txt to buffer list without showing it
:ls               list all buffers
:buffers          same
:b 3              switch to buffer #3
:b foo            switch to buffer whose name contains "foo"
:bn               next buffer
:bp               previous buffer
:bd               delete (close) the current buffer
:bd 3             delete buffer 3
:bufdo cmd        run cmd in every buffer
```

### Window

A **window** is a viewport that shows a buffer. You can have many windows on screen, each showing a different buffer (or the same buffer twice). Windows are the "split panes."

```
:sp file          horizontal split — open file in a new window above
:vsp file         vertical split — open file in a new window beside
Ctrl-w s          horizontal split (current buffer)
Ctrl-w v          vertical split
Ctrl-w h/j/k/l    move to the window in that direction
Ctrl-w w          cycle to next window
Ctrl-w c          close current window
Ctrl-w o          close all OTHER windows ("only")
Ctrl-w =          equalize window sizes
Ctrl-w +/-        grow/shrink height
Ctrl-w >/<        grow/shrink width
Ctrl-w T          move current window to its own tab
:windo cmd        run cmd in every window
```

### Tab page

A **tab page** is a layout of windows. Tabs are not "open files" like browser tabs are. They are *workspaces.* Each tab has its own arrangement of windows (and each window points to a buffer).

```
:tabnew           open a new tab
:tabnew file      open file in a new tab
:tabe file        same as :tabnew file
gt                go to next tab
gT                go to previous tab
2gt               go to tab 2
:tabclose         close current tab
:tabonly          close all OTHER tabs
:tabdo cmd        run cmd in every tab
```

### The hierarchy

```
┌─────────────────────────────────────────────────────┐
│                  VIM PROCESS                        │
│                                                     │
│   ┌─────────┐   ┌─────────┐   ┌─────────┐           │
│   │ TAB 1   │   │ TAB 2   │   │ TAB 3   │           │
│   │         │   │         │   │         │           │
│   │ ┌─────┐ │   │ ┌──┬──┐ │   │ ┌─────┐ │           │
│   │ │ Win │ │   │ │W │W │ │   │ │ Win │ │           │
│   │ │ A   │ │   │ │1 │2 │ │   │ │  D  │ │           │
│   │ └─────┘ │   │ ├──┴──┤ │   │ └─────┘ │           │
│   │         │   │ │ W3  │ │   │ ┌─────┐ │           │
│   │         │   │ └─────┘ │   │ │ Win │ │           │
│   │         │   │         │   │ │  E  │ │           │
│   │         │   │         │   │ └─────┘ │           │
│   └─────────┘   └─────────┘   └─────────┘           │
│                                                     │
│      Each Window points at a Buffer:                │
│                                                     │
│   ┌─────────────────────────────────────────────┐   │
│   │  BUFFER LIST (loaded files)                 │   │
│   │   1: main.go        4: utils.go             │   │
│   │   2: README.md      5: config.toml          │   │
│   │   3: notes.txt      6: [No Name]            │   │
│   └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

A **buffer** is the file in memory. A **window** is a viewport on a buffer. A **tab** is a layout of windows. Multiple windows can show the same buffer. Buffers exist independent of windows or tabs.

Most beginners over-use tabs because they map "tab" to their browser idea of tabs. In vim, just use buffers and windows. `:b foo` to switch buffers. `:vsp` to split. Tabs are for genuinely separate workspaces (different projects in the same vim session).

## Folding

Folding hides chunks of text so you can see the structure of a file at a glance. Vim has six fold methods:

```
:set foldmethod=manual    you create folds yourself
:set foldmethod=indent    fold by indent level
:set foldmethod=syntax    fold by syntax (language-aware)
:set foldmethod=marker    fold by marker comments  (default markers: {{{ }}})
:set foldmethod=expr      fold by an expression you write
:set foldmethod=diff      fold unchanged lines in a diff
```

In a Python file, `set foldmethod=indent` will collapse every function and block.

### Fold commands

```
zo    open fold under cursor
zc    close fold under cursor
za    toggle fold
zR    open ALL folds
zM    close ALL folds
zr    open one level
zm    close one level
zj    jump to next fold
zk    jump to previous fold
zf{motion}   create fold over motion (manual mode)
:5,15 fold   create fold from line 5 to 15
zd    delete the fold under cursor
zE    delete all folds
```

Hide a fold? `zc`. Open it? `zo`. Toggle? `za`. Want to see the whole file again? `zR`.

## Marks

A mark is a bookmark. You set it, then you can jump back to it.

```
ma          set local mark a (lowercase = file-local)
mA          set global mark A (uppercase = across all files)
'a          jump to start of line containing mark a
`a          jump to exact position of mark a
'A          jump to file containing global mark A, at that line
'.          jump to line of last change
'^          jump to position where you last left insert mode
''          jump to position before the last jump
`0          jump to position when last vim was closed (with viminfo)
:marks      list all marks
:delmarks a jump to delete mark a
```

Lowercase marks (`a-z`) are local to the file. Uppercase marks (`A-Z`) are global — `mA` puts a mark in this file at this line, and from any other file, `'A` will switch to this file at this line.

`''` (apostrophe-apostrophe) jumps you back to where you were before your last big jump. So you do `gg` to look at the top of the file, then `''` to come back to where you were. Different from `Ctrl-o` only in style.

## Quickfix and Location List

The **quickfix list** is vim's "list of things to deal with." When you run `:make`, the compiler errors go in the quickfix list. When you `:vimgrep`, the matches go in the quickfix list. You navigate the list with `:cnext` and `:cprev`.

```
:make             run 'makeprg' (default: make), capture errors into quickfix
:cnext  / :cn     go to next entry
:cprev  / :cp     go to previous
:copen            open the quickfix window
:cclose           close it
:cdo cmd          run cmd on every file/line in the quickfix list
```

There is also a **location list**, which is a quickfix list local to a window. LSP diagnostics often go here.

```
:lopen / :lclose  open/close location list
:lnext / :lprev   navigate
:ldo cmd          like :cdo but for the location list
```

### Search the project

```
:grep -rn pattern .         use external grep, fill the quickfix
:vimgrep /pat/ **/*.go      vim's built-in grep, fill quickfix
:Ggrep pattern              fugitive's git-grep wrapper
:Telescope live_grep        nvim plugin (fuzzy)
```

After a grep, `:copen` to see the list, `:cn`/`:cp` to walk through, or click-equivalent: jump in the quickfix window.

`:grep` runs `grepprg` which defaults to `grep -n`. Most folks set it to `rg --vimgrep`:

```vim
set grepprg=rg\ --vimgrep
```

## The .vimrc

Your config file. Lives at `~/.vimrc` (vim) or `~/.config/nvim/init.lua` (neovim). Loaded every time vim starts. This is where you make vim yours.

### A starter .vimrc (vim)

```vim
" basic options
set nocompatible              " don't bend over backwards for vi
set encoding=utf-8

" sane defaults
syntax on                     " syntax highlighting
filetype plugin indent on     " language-aware indent
set number                    " show line numbers
set relativenumber            " relative line numbers
set ruler                     " column position in status
set showcmd                   " show partial commands as you type
set wildmenu                  " tab-complete in command line
set incsearch                 " incremental search
set hlsearch                  " highlight matches
set ignorecase                " case-insensitive search...
set smartcase                 " ...unless you type a capital
set hidden                    " allow buffer switching without saving
set scrolloff=5               " keep 5 lines of context
set sidescrolloff=8

" indentation
set expandtab                 " spaces, not tabs
set tabstop=4                 " a tab is 4 columns wide
set softtabstop=4
set shiftwidth=4              " indent 4 spaces
set autoindent
set smartindent

" file safety
set undofile                  " persist undo across sessions
set undodir=~/.vim/undo
set backupdir=~/.vim/backup
set directory=~/.vim/swap
set updatetime=300

" clipboard
set clipboard=unnamedplus     " yank/paste use system clipboard

" mappings
let mapleader=" "             " spacebar = leader key
nnoremap <leader>w :w<CR>
nnoremap <leader>q :q<CR>
nnoremap <silent> <leader>h :nohlsearch<CR>
" jk to escape (a popular ergonomics hack)
inoremap jk <Esc>
```

### A starter init.lua (neovim)

```lua
-- options
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.expandtab = true
vim.opt.tabstop = 4
vim.opt.shiftwidth = 4
vim.opt.smartindent = true
vim.opt.ignorecase = true
vim.opt.smartcase = true
vim.opt.hlsearch = true
vim.opt.incsearch = true
vim.opt.hidden = true
vim.opt.undofile = true
vim.opt.scrolloff = 5
vim.opt.clipboard = "unnamedplus"

-- leader
vim.g.mapleader = " "

-- mappings
vim.keymap.set("n", "<leader>w", "<cmd>w<CR>")
vim.keymap.set("n", "<leader>q", "<cmd>q<CR>")
vim.keymap.set("n", "<leader>h", "<cmd>nohlsearch<CR>")
vim.keymap.set("i", "jk", "<Esc>")
```

### Reload

```
:source ~/.vimrc      reload your vimrc
:so $MYVIMRC          shortcut to reload whatever was loaded
:scriptnames          list every script vim has loaded (huge debug help)
:verbose set option?  see where an option was last set from
```

## Plugins

Vim has thousands of plugins. To install one, you need a **plugin manager**.

### vim-plug (vim and nvim, simple)

```vim
" in your vimrc
call plug#begin('~/.vim/plugged')
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-surround'
Plug 'tpope/vim-commentary'
Plug 'junegunn/fzf', { 'do': { -> fzf#install() } }
Plug 'junegunn/fzf.vim'
call plug#end()
```

Then `:PlugInstall` to install. `:PlugUpdate` to update. `:PlugClean` to remove ones not in your list.

### packer.nvim (neovim, Lua)

```lua
require('packer').startup(function(use)
  use 'wbthomason/packer.nvim'
  use 'tpope/vim-fugitive'
  use 'nvim-treesitter/nvim-treesitter'
end)
```

Then `:PackerSync`.

### lazy.nvim (neovim, modern, fast)

```lua
-- bootstrap
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
  vim.fn.system({"git", "clone", "--filter=blob:none",
    "https://github.com/folke/lazy.nvim.git", "--branch=stable", lazypath})
end
vim.opt.rtp:prepend(lazypath)

require("lazy").setup({
  "tpope/vim-fugitive",
  "tpope/vim-surround",
  { "nvim-treesitter/nvim-treesitter", build = ":TSUpdate" },
  { "neovim/nvim-lspconfig" },
})
```

Then `:Lazy sync`.

### Beloved plugins

- `tpope/vim-fugitive` — git from inside vim. `:Git status`, `:Git blame`, `:Gdiff`.
- `tpope/vim-surround` — change/add/delete surrounding pairs. `cs"'` changes `"...."` to `'...'`. `ds(` deletes parens around. `ysiw"` wraps word in quotes.
- `tpope/vim-commentary` — `gcc` comments line, `gc` comments motion, `gcap` comments paragraph.
- `tpope/vim-repeat` — makes plugin commands repeatable with `.`.
- `easymotion / leap.nvim / hop.nvim` — sub-second jumping by typing one or two chars.
- `nvim-treesitter` — better syntax highlighting via treesitter parser.
- `neovim/nvim-lspconfig` + `mason.nvim` — built-in language servers.
- `nvim-telescope/telescope.nvim` — fuzzy finder for everything.
- `harpoon` — bookmark a few key files for instant switching.
- `oil.nvim` / `nvim-tree` / `neo-tree` — file browsers.

## Neovim vs Vim 9

If you are starting today, install neovim. The big differences:

| | Vim 9 | Neovim |
| --- | --- | --- |
| Config | Vimscript or Vim9script | Lua (`init.lua`) or Vimscript |
| Plugin API | C with vim API | Lua + remote plugins via msgpack-rpc |
| Async | `:!cmd &`, channels (limited) | Native async via `vim.loop` (libuv) |
| Treesitter | not built-in | built-in |
| LSP client | not built-in (requires plugin) | built-in (`vim.lsp`) |
| Terminal | `:terminal` (vim 8.1+) | `:terminal` with terminal mode |
| GUI | gvim, mvim | embedders: VimR, neovide, etc. |
| Plugin language | Vimscript | Lua |
| Compiled config | Vim9script (fast) | Lua (fast) |

Vim 9 is fast and good. Neovim is the same idea but newer and easier to extend in a real programming language. Both are completely viable. The vim community is healthy and merging good ideas across.

In neovim, almost all options have a Lua equivalent:

```lua
vim.opt.number = true             -- :set number
vim.cmd("colorscheme habamax")    -- run any ex command from Lua
vim.api.nvim_create_autocmd(...)  -- set an autocommand
vim.keymap.set(...)               -- set a mapping
```

## Common Errors

Verbatim — these are the actual messages vim prints. Memorize the fixes.

- **`E37: No write since last change`** — you tried to switch buffers or quit, but your buffer has unsaved changes. Either `:w` to save, `:q!` to discard, or `:set hidden` so vim lets you change buffers without saving.
- **`E37: No write since last change (add ! to override)`** — same as above; the message includes the override hint.
- **`E20: Mark not set`** — you typed `'a` but never did `ma` to set mark `a` in this file. Set the mark first.
- **`E121: Undefined variable: g:foo`** — your vimrc references `g:foo` but the variable was never defined. Check spelling or initialize with `let g:foo = 1`.
- **`E319: Sorry, the command is not available in this version`** — you tried to use a feature your vim was not compiled with (e.g. `:python` on a vim built without `+python3`). Check `:version` to see your `+features` and rebuild or install a richer package.
- **`E486: Pattern not found`** — your `/search` did not match. Just informational. Press `Esc`.
- **`E72: Close error on swap file`** — vim could not write the `.swp` file. Permission problem in the directory, or disk full. Check `:set directory?` and disk space.
- **`W11: Warning: File "X" has changed since editing started`** — the file on disk changed under you (another editor modified it, or git checked out a different version). Vim asks if you want to reload. Use `:e!` to discard your buffer and re-read from disk, or `:w!` to overwrite the file with your buffer.
- **`E325: ATTENTION`** — vim found an existing swap file when opening this file. That means another vim is editing it, or another vim crashed mid-edit. Read the dialog, then choose: open Read-only `(O)`, Edit anyway `(E)`, Recover `(R)`, Delete it `(D)`, or Quit `(Q)`. If you are sure no other vim is running, deleting the swap is safe.
- **`E13: File exists (add ! to override)`** — you tried to `:saveas filename` to a path that already exists. `:saveas! filename` to overwrite.
- **`Not an editor command: XYZ`** — you typed `:XYZ` but vim doesn't know it. Either you misspelled it, or it's defined by a plugin that didn't load. Check `:scriptnames`.
- **`E212: Can't open file for writing`** — permission denied or directory missing. Check the path. Sudo-edit with `:w !sudo tee %` (or `:SudoWrite` from vim-eunuch).
- **`E471: Argument required`** — you typed an ex command that needs an argument and forgot it. e.g. `:e` with nothing.
- **`E492: Not an editor command`** — same family as the "Not an editor command" message.
- **`E464: Ambiguous use of user-defined command`** — two plugins define the same `:Cmd`. Use the long form, or rename one.

## Hands-On

Sit down, do these in order. None of them break anything.

```bash
# 1) install neovim (mac)
$ brew install neovim
# 1') install neovim (debian/ubuntu)
$ sudo apt install neovim
# 1'') install vim 9 (mac)
$ brew install vim

# 2) make a sandbox
$ mkdir ~/vim-playground && cd ~/vim-playground
$ printf 'hello\nworld\nfoo\nbar\nbaz\n' > sample.txt

# 3) open the file
$ nvim sample.txt
# (or: vim sample.txt)

# 4) get out without saving
:q!

# 5) open at the bottom of the file with a normal-mode command run
$ nvim +'normal G' sample.txt

# 6) open two files in vertical split
$ nvim -O sample.txt sample2.txt

# 7) open as tabs
$ nvim -p sample.txt sample2.txt

# 8) diff two files
$ nvim -d sample.txt sample2.txt
# (in nvim: ]c and [c jump between hunks; do = take change; dp = put change)

# 9) read-only mode (no accidental edits)
$ nvim -R /etc/hosts

# 10) clean start (no plugins, no rc) — useful to test if your config breaks something
$ nvim --clean

# 11) inside nvim: turn line numbers on
:set number
:set relativenumber

# 12) inside nvim: see what set 'expandtab' last
:verbose set expandtab?

# 13) list every script that loaded (huge debug aid)
:scriptnames

# 14) see your messages so far
:messages

# 15) syntax highlighting
:syntax on

# 16) show invisible characters
:set list

# 17) open the directory of the current file (built-in netrw)
:Ex
:e .

# 18) open netrw as a sidebar
:Lex

# 19) (nvim) open a terminal
:term
# back to normal mode in a terminal buffer:  Ctrl-\ Ctrl-n

# 20) (vim 8.1+) terminal sized 10 rows, auto-close on exit
:term ++rows=10 ++close

# 21) find a file by name in 'path'
:set path+=**
:find sample.txt

# 22) run an ex command across every file in the arglist
:argdo %s/foo/bar/g | update

# 23) every loaded buffer
:bufdo %s/foo/bar/g | update

# 24) every tab
:tabdo %s/foo/bar/g | update

# 25) every window
:windo set number

# 26) grep across the project
:grep -r foo .
:copen
:cnext

# 27) git grep (vim-fugitive)
:Ggrep TODO

# 28) (nvim, with telescope plugin) live grep
:Telescope live_grep

# 29) (nvim) what LSP servers are running on this buffer?
:LspInfo

# 30) (nvim, with mason.nvim) install a language server
:Mason

# 31) (nvim) check that your nvim is healthy
:checkhealth

# 32) install plugins (vim-plug)
:PlugInstall

# 33) sync plugins (packer.nvim)
:PackerSync

# 34) sync plugins (lazy.nvim)
:Lazy sync

# 35) macro practice — increment a number across 10 lines
# 1. Type a single number, e.g.: 1
# 2. With cursor on it, press qa  (record into a)
# 3. Press: yy p Ctrl-a q
# 4. Now press: 9@a
# You should now have 1 through 10 on consecutive lines.

# 36) text-object practice
# Open a file with: function foo("hello", "world") { return 42 }
# Cursor on hello: ci"   --> change inside double-quote
# Cursor on parens: ci(  --> change inside parens
# Cursor on whole call: caf  (with a treesitter-aware plugin) --> change around function

# 37) search and replace with confirmation
:%s/foo/bar/gc

# 38) delete every blank line
:g/^$/d

# 39) delete every line NOT containing TODO
:v/TODO/d

# 40) save the file
:w
# save and close
:wq
# (or: ZZ — save+quit, ZQ — quit no save)
```

## Common Confusions

These trip everyone up. Read each pair, understand which is which.

### 1. `x` vs `X` vs `dl`

`x` deletes the character UNDER the cursor. `X` deletes the character BEFORE the cursor. `dl` is "delete one character to the right" — same as `x`. They overlap; `x` is the shortcut.

### 2. `cw` vs `daw`

`cw` changes from the cursor to the end of the word. So if the cursor is in the middle of `hello`, `cw` only changes `llo`. `daw` deletes the entire word `hello` (and the trailing space). Use `ciw` to change the whole word the cursor is in.

### 3. `:w` vs `:w!`

`:w` writes (saves) the current buffer to its file. `:w!` forces the write, overriding read-only or "file changed" warnings. Use `:w!` when you mean it. Most people don't need it daily.

### 4. How to actually quit

```
:q       quit (fails if there are unsaved changes)
:q!      quit, discard changes
:wq      write and quit
:x       same as :wq but only writes if there were changes
ZZ       same as :x in normal mode
ZQ       same as :q!
:qa      quit ALL windows/tabs (fails on unsaved)
:qa!     quit all, discard everything
:wqa     write all and quit all
```

`:qa!` is the "I want out, drop everything" key.

### 5. Esc vs ergonomic alternatives

The `Esc` key is far from the home row. Common workarounds:

- Map `jk` (or `jj`, or `kj`) to Esc in insert mode. (`inoremap jk <Esc>`)
- Use `Ctrl-[` — same as Esc, on the home row.
- Use `Ctrl-c` — close, but not identical (it doesn't trigger `InsertLeave` autocmds).
- Remap your CapsLock key to Esc at the OS level (and free your fingers forever).

### 6. Register sync with the system clipboard

By default, vim's yanks do NOT go to your OS clipboard. Solutions:

- Use the `+` register: `"+y` to yank to system clipboard, `"+p` to paste from it.
- Set `:set clipboard=unnamedplus` so all yanks/pastes use the system clipboard automatically.
- Make sure your vim was built with `+clipboard` (`vim --version | grep clipboard`). Otherwise even `"+y` won't work.

### 7. `:s/X/Y/` only replaces the first match per line

By default `:s` is "substitute the first match on the line." Add `g` for "all matches on the line." Add `%` for "all lines." So `:%s/X/Y/g` is "all matches in the file."

### 8. Vimscript vs Lua in Neovim

Both work. Vimscript is the original, with weird syntax (`let g:foo = 1`, `if has('python3')`). Lua is a real programming language — `vim.opt.number = true`. New users should learn Lua. Most modern neovim plugins are Lua-only. Vim 9 cannot run Lua (use Vimscript or Vim9script there).

### 9. Buffer vs window

A buffer is a file in memory. A window is a viewport showing a buffer. Closing a window does NOT delete the buffer (use `:bd` for that). Hiding a buffer does NOT close it (it's still in `:ls`).

### 10. Quickfix vs location list

There is ONE quickfix list per vim instance — shared across windows. There is ONE location list per WINDOW — separate per window. Use quickfix for project-wide things (compile errors, grep results). Use location lists for window-local things (LSP diagnostics for the file in this window).

### 11. Why does my plugin manager need `:PlugInstall` (or `:PackerSync` or `:Lazy sync`)

Because the plugin manager is a vim plugin too. It reads your config, sees the list of plugins you want, and clones them. The first run, you have to *tell it* to clone them. Same after adding a new plugin. `lazy.nvim` and `packer.nvim` will auto-install on startup if you bootstrap them; vim-plug requires you to run `:PlugInstall` once.

### 12. The swap file dialog

Vim writes a `.swp` file beside the file you are editing as a crash-protection backup. If vim is killed (or you opened the same file in two vims), the second vim sees a swap file already there and shows the **E325: ATTENTION** dialog.

Options:
- `(O)pen Read-Only` — view but not edit.
- `(E)dit anyway` — open it; both vims now have stale swaps; bad idea unless you know it's stale.
- `(R)ecover` — load whatever was in the swap to recover unsaved work.
- `(D)elete it` — if you're sure the other vim is dead.
- `(Q)uit` — back out.

If you `(R)ecover`, save the file, then `:!rm sample.txt.swp` (or `:!rm .sample.txt.swp` depending on `directory` setting) to clear the swap.

### 13. Accidentally entered Replace mode

You hit `R` in normal mode — now every key OVERWRITES instead of inserting. The status bar says `-- REPLACE --`. Just press `Esc` to leave. If you typed over things you wanted to keep, `u` to undo.

### 14. Can't paste indented code

When you paste code into insert mode, vim's autoindent jumps in and double-indents. Fix: `:set paste` before pasting, `:set nopaste` after. Or set `:set pastetoggle=<F2>` to toggle quickly. Neovim with default settings often doesn't have this issue.

### 15. `dd` deletes a line — but it's still in the register

`dd` cuts the line. The line is in the unnamed register. So `p` will paste it. Vim has no separate "cut" and "delete" — every "delete" is a "cut." If you want to truly throw text away, use the black hole register: `"_dd`.

### 16. `:e file` blows up your splits

`:e file` replaces the buffer in the CURRENT window with `file`. If you wanted a split, use `:sp file` or `:vsp file`. If you wanted a tab, use `:tabnew file`.

### 17. Where did my command go in `q:`

If you press `q:` accidentally, vim opens the **command-line history window** — a buffer with every ex command you've run. You can edit and re-run. Press `Ctrl-c` or `:q` to leave. Same trick: `q/` opens search history.

### 18. Why is my colorscheme ugly

Make sure your terminal supports 256-color or true-color. In vim:

```vim
set termguicolors
```

In your shell, `echo $TERM` should be `xterm-256color`, `tmux-256color`, or similar. If not, set it.

## Vocabulary

| Term | Plain English |
| --- | --- |
| vi | The original 1976 visual editor that vim is based on. |
| vim | "vi improved" — the most common modern editor in the family. |
| neovim | Fork of vim from 2014 with cleaner code, async, Lua. Also `nvim`. |
| gvim | Graphical (window) version of vim. |
| mvim | macOS graphical version of vim. |
| nvim | Command name for neovim. |
| modal editing | Editor where keys do different things in different modes. |
| normal mode | The default mode where keys are commands, not letters. |
| insert mode | Mode where keys type letters (the typewriter mode). |
| visual mode | Mode for selecting text character by character. |
| visual-line mode | Like visual but selects whole lines. |
| visual-block mode | Like visual but selects rectangles (columns). |
| command-line mode | Mode for ex commands (`:`), search (`/`, `?`). |
| ex command | A command typed after `:` (like `:w`, `:q`, `:set`). |
| terminal mode | (nvim) When a terminal buffer is interactive, your keys go to the shell. |
| replace mode | Mode where typed letters overwrite existing letters. |
| virtual replace | Replace mode that treats tabs as spaces. |
| motion | A command that moves the cursor (e.g. `w`, `$`). |
| operator | A command that acts on text (e.g. `d`, `c`, `y`). |
| count | A number you put before a motion or operator to repeat (e.g. `3w`). |
| text object | A description of a chunk of text (e.g. `iw`, `ap`, `i"`). |
| character object | A text object that's a single character (`<>`, `()`, etc.). |
| line object | A text object that's a line. |
| paragraph | Block of non-blank lines. `ip` selects one. |
| sentence | Text up to a `. ! ?` followed by space. `is` selects one. |
| word | Letters/digits/underscore — bounded by other punctuation. |
| WORD | Whitespace-separated chunk (capital W in commands). |
| inner / a | Adjectives for text objects. `i` = inside; `a` = around (with whitespace/delimiters). |
| buffer | A file loaded into memory. |
| window | A viewport on a buffer (a "split"). |
| tab page | A layout of windows. |
| viewport | The visible part of a buffer in a window. |
| register | One of vim's many clipboards. Prefixed with `"`. |
| unnamed register `"` | The default register where deletes/yanks go. |
| yank register `0` | Holds the last yank, untouched by deletes. |
| numbered registers `1-9` | Hold the last 9 deletes, oldest at 9. |
| named register `a-z` | You choose what goes in. |
| capital named `A-Z` | Same as a-z but APPENDS to the existing contents. |
| clipboard register `+` | The system clipboard (X11 CLIPBOARD). |
| selection register `*` | The X11 PRIMARY selection (middle-click). |
| expression register `=` | Type an expression, get the result. |
| black hole `_` | Toss text away. Doesn't clobber the unnamed register. |
| last ex `:` | Holds the last ex command. |
| last search `/` | Holds the last search pattern. |
| filename `%` | Holds the current filename. |
| last insert `.` | Holds the last text you typed in insert mode. |
| macro | A recorded keystroke sequence stored in a register. |
| recording | The act of saving keystrokes into a register (`q<reg>...q`). |
| jump list | Vim's history of "big" cursor jumps. `Ctrl-o`/`Ctrl-i`. |
| change list | Vim's history of cursor positions where edits happened. `g;`/`g,`. |
| ChangeNr | The change number — vim tracks every change with a number. |
| marks | Bookmarks in a file. `m<x>` sets, `'<x>` jumps. |
| global mark | Uppercase mark (`A-Z`) — works across files. |
| local mark | Lowercase mark (`a-z`) — file-scoped. |
| swap file `.swp` | Crash-recovery snapshot vim writes during editing. |
| undo file `.un~` | Persistent undo history saved across sessions. |
| persistent undo | Undo that survives quitting and reopening (`set undofile`). |
| redo tree | Vim's undo is a tree, not a stack. `g-` and `g+` walk it. |
| folding | Hiding sections of text to see structure. |
| manual fold | Folds you create with `zf`. |
| syntax fold | Folds based on language syntax. |
| indent fold | Folds based on indent level. |
| marker fold | Folds based on `{{{ ... }}}` markers in comments. |
| expr fold | Folds defined by an expression you write. |
| foldcolumn | Margin where fold markers show. |
| `set foldmethod` | Option that picks one of the six fold modes. |
| `set foldlevel` | How deep folds open by default. |
| foldopen | Set of triggers that auto-open folds. |
| syntax | Vim's language-aware coloring engine. |
| syntax highlighting | Coloring keywords, strings, comments by language. |
| colorscheme | Your color theme (e.g. `:colorscheme habamax`). |
| treesitter | A modern parser that gives nvim better syntax info than regex. |
| LSP | Language Server Protocol — IDE-like features over RPC. |
| LSP client | The vim/nvim side that talks to a language server. |
| mason | A neovim plugin that installs/manages LSP servers, linters, formatters. |
| plugin | Extra code you load to add features. |
| vimscript | Vim's original scripting language. |
| Vim9script | Vim 9's faster, more modern scripting language. |
| Lua | The scripting language built into neovim. |
| init.lua | Neovim's Lua config file. |
| init.vim | Neovim's optional Vimscript config file. |
| `.vimrc` | Vim's main config file (in your home dir). |
| runtimepath | Vim's `$PATH` for plugins, syntax, and runtime files. |
| packpath | Like runtimepath, for `pack/` style plugins. |
| ftplugin | Filetype plugin — runs only for files of a given type. |
| autocmd | A command that runs on an event (file open, write, etc.). |
| augroup | A named group of autocommands you can clear together. |
| autocommand | Same as autocmd. |
| buffer-local | A setting that applies only to the current buffer. |
| window-local | A setting that applies only to the current window. |
| tab-local | A setting that applies only to the current tab. |
| scratch buffer | A throwaway buffer with no associated file. |
| terminal buffer | A buffer running an interactive shell (in nvim/vim8). |
| help buffer | The buffer opened by `:help`. |
| quickfix | Vim's per-instance "list of things to deal with." |
| location list | A per-window quickfix-style list. |
| error format | The pattern vim uses to parse compiler output (`errorformat`). |
| ripgrep / rg | A fast grep replacement many people use as `grepprg`. |
| fzf | A fuzzy finder, often integrated with vim plugins. |
| telescope | A neovim fuzzy finder plugin. |
| vim-plug | A simple, popular plugin manager. |
| packer.nvim | A neovim plugin manager written in Lua. |
| lazy.nvim | A modern neovim plugin manager that lazy-loads plugins. |
| paq-nvim | A minimal neovim plugin manager. |
| dein.vim | A older fast plugin manager. |
| vundle | An older plugin manager (legacy). |
| pathogen | An older plugin manager (legacy). |
| vim-fugitive | tpope's git-from-inside-vim plugin. |
| vim-surround | tpope's plugin for changing surrounding pairs. |
| vim-commentary | tpope's plugin for commenting code. |
| vim-easymotion | A plugin for jumping to any visible character with 2 keys. |
| leap.nvim | A modern alternative to easymotion. |
| hop.nvim | Another modern alternative to easymotion. |
| harpoon | A plugin for bookmarking a few files for instant switching. |
| oil.nvim | A neovim file browser that lets you edit the directory listing as text. |
| nvim-tree | A neovim file-tree sidebar plugin. |
| neo-tree | A modern alternative to nvim-tree. |
| dressing | A plugin that polishes vim's `vim.ui.input/select`. |
| lualine | A popular Lua-based statusline plugin. |
| statusline | The bottom bar showing mode, file, position. |
| tabline | The top bar showing tabs (with `:set showtabline=2`). |
| winbar | A bar at the top of each window (nvim 0.8+). |
| signcolumn | The leftmost column showing signs (errors, breakpoints). |
| listchars | Characters used to show whitespace when `:set list` is on. |
| fillchars | Characters used to fill window borders, fold lines, etc. |
| guicursor | Option controlling the cursor's shape per mode (in TUI/GUI). |
| conceal level | How aggressively to hide special syntax (e.g. markdown links). |
| modeline | A line in a file with vim options (e.g. `vim: ts=4 et`). |
| modelines | The number of lines vim checks for a modeline. |
| magic | Default regex flavor — most chars literal. |
| very magic `\v` | All meta chars are special (most regex-friendly). |
| no magic | Almost everything is literal. |
| `%` current file | The substitution that vim does for the current filename. |
| `#` alternate file | The other buffer you most recently visited. |
| `:registers` | Show all register contents. |
| `:marks` | Show all marks. |
| `:jumps` | Show the jump list. |
| `:changes` | Show the change list. |
| `:buffers` | Show the buffer list (also `:ls`). |
| `:ls` | Show the buffer list. |
| `:bdelete` | Delete a buffer (close it). |
| `:bunload` | Unload a buffer (drop from memory but keep in list). |
| `:tabnext` | Go to next tab page (also `gt`). |
| `gT` | Previous tab page. |
| q-mode | The recording state — `q<x>` starts, `q` stops. |
| q: command-history | The buffer of past ex commands (entered with `q:`). |
| q/ search-history | The buffer of past search patterns (entered with `q/`). |

## The Undo Tree

Most editors give you undo as a stack: undo goes back, redo goes forward, but if you undo and then make a new edit, the redo branch is gone. Vim does NOT do this.

Vim's undo is a tree.

```
        edit A
          │
        edit B
          │
        edit C
          │
   ┌──────┴──────┐
   │             │
 edit D        edit E   <-- you went back to C, made D, undone, then E
   │             │
 edit F        edit G
```

Both branches are preserved. Commands to walk the tree:

```
u            undo
Ctrl-r       redo
g-           go to OLDER state in time
g+           go to NEWER state in time
:earlier 5m  state from 5 minutes ago
:earlier 1h  state from 1 hour ago
:later 30s   state 30 seconds forward
:undolist    show undo branches
```

Plugins like `undotree` (mbbill/undotree) give you a visual browser. Set `:set undofile` and your tree persists across sessions — months of edits, all browsable.

## Try This

A 30-minute drill. Do this once, then again tomorrow, then again next week. Your fingers will catch on.

```vim
" 1) Open a fresh buffer
:enew

" 2) Type 5 lines of placeholder text:
i
This is line one.
This is line two.
Line three is here.
And line four.
Line five is the last.<Esc>

" 3) Practice motions
gg            " go to top
G             " go to bottom
3G            " go to line 3
$             " end of line
0             " start of line
^             " first non-whitespace
w             " next word
b             " back word
e             " end of word

" 4) Practice operators
yy            " yank line
p             " paste below
dd            " delete line (still in register, of course)
u             " undo
Ctrl-r        " redo

" 5) Combine operators with motions
3dd           " delete 3 lines
5yy           " yank 5 lines
d$            " delete to end of line
c$            " change to end of line
Esc

" 6) Text objects
ciw           " change inner word
ci"           " change inside double-quoted (place cursor in some quoted text first)
daw           " delete a word

" 7) Search and jump
/line<CR>     " search for 'line'
n             " next match
N             " previous match
*             " search for word under cursor

" 8) Macros — make every line uppercase
gg            " go to top
qa            " record into register a
gUU           " uppercase line
j             " next line
q             " stop recording
4@a           " run macro 4 more times -> all 5 lines uppercase

" 9) Replace
:%s/line/LINE/g

" 10) Marks
ma            " mark a here
G             " jump to bottom
'a            " come back to mark a

" 11) Splits
:vsp          " vertical split
Ctrl-w l      " move to right window
:b#           " switch to alternate buffer
Ctrl-w q      " close window

" 12) Save and quit
:w
:q
```

If you do this drill once a day for a week, you will feel competent. Two weeks: confident. A month: faster than your old editor.

## Where to Go Next

Once vim feels natural in normal/insert/visual, level up:

1. **Learn `:help`** — type `:help` for the index, `:help <topic>` for any topic. The help is excellent. Type `:help quickref` for the cheat sheet.
2. **Read "Practical Vim" by Drew Neil.** Best vim book ever written. Tip-format. You can read one tip per coffee break.
3. **Read "Modern Vim" by Drew Neil.** The follow-up. Covers neovim, async, terminals.
4. **Configure your `.vimrc` (or `init.lua`).** Build it slowly. Don't copy a 2000-line config from someone else; that won't help you learn.
5. **Add 5-10 plugins, no more.** Surround, commentary, fugitive, treesitter, telescope, lspconfig+mason. That's enough.
6. **Learn macros for real.** Spend an hour deliberately practicing macros. They feel awkward, then they feel essential.
7. **Learn the substitute command.** `:%s/.../.../g` with `\v` very-magic and a few captures unlocks bulk text transforms no IDE can match.
8. **Learn LSP.** In neovim, set up `nvim-lspconfig` + `mason.nvim`. You get rename, go-to-definition, hover, formatting, diagnostics.
9. **Learn the quickfix list.** `:grep`, `:make`, `:cdo`. Project-wide refactors become routine.
10. **Use vim everywhere you can.** Bash readline has vim mode (`set -o vi`). Many shells, browsers, and IDEs have vim keybinding plugins. Once your fingers know vim, bring it everywhere.

Welcome to the piano. The chords come slow. The music lasts forever.

## See Also

- editors/vim
- editors/neovim
- editors/emacs
- editors/nano
- terminal/tmux
- terminal/screen
- ramp-up/bash-eli5
- ramp-up/git-eli5
- ramp-up/linux-kernel-eli5

## References

- `:help` (the built-in help — start with `:help quickref` and `:help user-manual`)
- vimdoc.sourceforge.net — searchable HTML mirror of vim's docs
- "Practical Vim" — Drew Neil. The definitive intermediate-to-advanced book.
- "Modern Vim" — Drew Neil. Follow-up covering vim 8 and neovim.
- neovim.io/doc — neovim's official documentation
- Lua docs for Neovim — `:help lua-guide` (or online: neovim.io/doc/user/lua-guide.html)
- Vim Tips Wiki — vim.fandom.com (lots of community tips, varying quality)
- r/vim and r/neovim on Reddit — active communities
- "Learn Vimscript the Hard Way" — Steve Losh. Free online. Read after you can edit comfortably.
