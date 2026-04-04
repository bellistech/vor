# Emacs (GNU Emacs)

> Extensible, self-documenting text editor with Lisp-based configuration and Org-mode.

## Notation

```
C-x    = Ctrl + x
M-x    = Alt + x (or Esc then x)
C-M-x  = Ctrl + Alt + x
RET    = Enter
SPC    = Space
```

## Movement

### Basic Movement

```
C-f / C-b          move forward/back one character
M-f / M-b          move forward/back one word
C-n / C-p          next/previous line
C-a / C-e          beginning/end of line
M-a / M-e          beginning/end of sentence
C-v / M-v          scroll down/up one page
M-< / M->          beginning/end of buffer
M-g M-g            go to line number
C-l                recenter screen (cycle top/center/bottom)
```

### Search

```
C-s                incremental search forward
C-r                incremental search backward
C-s C-s            repeat last search forward
M-%                query replace
C-M-s              regex search forward
C-M-%              regex query replace
M-s o              occur (list all matches)
```

## Editing

### Basic Editing

```
C-d                delete character forward
M-d                delete word forward
C-k                kill to end of line
M-k                kill to end of sentence
C-w                kill region (cut)
M-w                copy region
C-y                yank (paste)
M-y                cycle through kill ring (after C-y)
C-/                undo
C-x u              undo (alternative)
C-g                cancel current command
```

### Selection and Region

```
C-SPC              set mark (start selection)
C-x C-x            exchange point and mark
C-x h              select entire buffer
M-h                mark paragraph
C-M-SPC            mark sexp (expression)
```

### Transposing and Case

```
C-t                transpose characters
M-t                transpose words
C-x C-t            transpose lines
M-u                uppercase word
M-l                lowercase word
M-c                capitalize word
C-x C-u            uppercase region
C-x C-l            lowercase region
```

## Buffers

```
C-x C-f            find (open) file
C-x C-s            save buffer
C-x C-w            save as (write file)
C-x b              switch buffer
C-x C-b            list buffers
C-x k              kill buffer
C-x s              save some buffers (prompt for each)
```

## Windows

```
C-x 2              split window horizontally
C-x 3              split window vertically
C-x 0              close current window
C-x 1              close all other windows
C-x o              switch to other window
C-x ^              enlarge window vertically
C-x {              shrink window horizontally
C-x }              enlarge window horizontally
```

## Frames

```
C-x 5 2            create new frame
C-x 5 0            close current frame
C-x 5 o            switch to other frame
C-x 5 f            find file in new frame
```

## Dired (Directory Editor)

```
C-x d              open dired
RET                open file/directory
d                  mark for deletion
u                  unmark
x                  execute deletions
R                  rename/move
C                  copy
+                  create directory
g                  refresh
^                  go up one directory
m                  mark file
% m                mark by regex
```

## Shell

```
M-!                run shell command
M-&                run shell command async
M-|                pipe region through command
M-x shell          open shell buffer
M-x eshell         open emacs lisp shell
M-x term           open terminal emulator
C-x C-z            suspend emacs (fg to resume)
```

## Packages (use-package / straight.el)

### use-package

```elisp
;; ~/.emacs.d/init.el or ~/.emacs
(require 'package)
(setq package-archives
      '(("melpa" . "https://melpa.org/packages/")
        ("gnu"   . "https://elpa.gnu.org/packages/")))
(package-initialize)

;; install use-package if missing
(unless (package-installed-p 'use-package)
  (package-refresh-contents)
  (package-install 'use-package))

;; example package declarations
(use-package magit
  :ensure t                              ; auto-install from melpa
  :bind ("C-x g" . magit-status))       ; keybinding

(use-package company
  :ensure t
  :hook (prog-mode . company-mode)       ; enable in programming modes
  :config
  (setq company-idle-delay 0.2))

(use-package which-key
  :ensure t
  :defer 1                               ; load after 1 second
  :config (which-key-mode))
```

### straight.el

```elisp
;; bootstrap straight.el (add to early init)
(defvar bootstrap-version)
(let ((bootstrap-file (expand-file-name "straight/repos/straight.el/bootstrap.el"
                                         user-emacs-directory)))
  (unless (file-exists-p bootstrap-file)
    (url-retrieve-synchronously
     "https://raw.githubusercontent.com/radian-software/straight.el/develop/install.el"
     'silent 'inhibit-cookies)
    (goto-char (point-max))
    (eval-print-last-sexp)))

;; use with use-package
(straight-use-package 'use-package)
(use-package magit :straight t)
```

## Org-Mode Basics

```
* Heading 1                              top-level heading
** Heading 2                             sub-heading
- list item                             unordered list
1. numbered item                        ordered list

TAB                toggle fold heading
S-TAB              cycle all headings
M-RET              new heading/list item
M-left / M-right   promote/demote heading
M-up / M-down      move subtree up/down

C-c C-t            cycle TODO state
C-c C-s            schedule item
C-c C-d            set deadline
C-c .              insert timestamp
C-c C-c            toggle checkbox [ ] -> [X]
C-c C-l            insert/edit link
C-c C-o            open link at point
C-c C-e            export dispatcher (html, pdf, etc.)
C-c a              org-agenda
```

## TRAMP (Remote Editing)

```
C-x C-f /ssh:user@host:/path/file       edit file over SSH
C-x C-f /sudo::/etc/hosts               edit as root
C-x C-f /ssh:user@host|sudo::/etc/file  SSH then sudo
C-x C-f /docker:container:/path/file    edit inside docker
```

## Macros

```
C-x (              start recording macro
C-x )              stop recording macro
C-x e              execute last macro
C-u 10 C-x e       execute macro 10 times
C-x C-k n          name last macro
M-x insert-kbd-macro   save macro as elisp
```

## Help System

```
C-h t              tutorial
C-h k              describe key
C-h f              describe function
C-h v              describe variable
C-h m              describe current modes
C-h a              apropos (search commands)
C-h i              info manual
C-h b              list all keybindings
C-h w              where is command bound
C-h P              describe package
```

## Tips

- Emacs keybindings are everywhere: bash, zsh, readline, and macOS text fields all support C-a/C-e/C-k.
- `M-x` is the universal command launcher -- if you forget a keybinding, type `M-x` and the command name.
- Use `C-g` liberally to cancel anything -- stuck prompts, long operations, partial key sequences.
- `which-key` package shows available keybindings after a prefix -- invaluable for learning.
- Org-mode alone justifies learning Emacs: notes, task management, literate programming, and export to HTML/PDF/LaTeX.
- TRAMP lets you edit remote files transparently -- just change the path prefix.
- `M-x customize` provides a GUI for changing settings without writing elisp.
- Emacs server (`emacs --daemon`, `emacsclient`) gives instant startup for subsequent files.
- The help system (`C-h`) is comprehensive and self-documenting -- use `C-h k` to learn what any key does.

## See Also

- vim, neovim, tmux, git, regex, bash

## References

- [GNU Emacs Manual](https://www.gnu.org/software/emacs/manual/) -- complete user manual
- [Emacs Lisp Reference Manual](https://www.gnu.org/software/emacs/manual/html_node/elisp/) -- Elisp programming guide
- [Emacs Wiki](https://www.emacswiki.org/) -- community-maintained tips, packages, and configuration
- [MELPA](https://melpa.org/) -- package archive for Emacs (community packages)
- [GNU ELPA](https://elpa.gnu.org/) -- official GNU Emacs package archive
- [Org Mode Manual](https://orgmode.org/manual/) -- Org-mode documentation
- [Emacs Key Binding Reference](https://www.gnu.org/software/emacs/refcards/pdf/refcard.pdf) -- official reference card (PDF)
- [Mastering Emacs](https://www.masteringemacs.org/) -- practical articles and guides
- [man emacs](https://man7.org/linux/man-pages/man1/emacs.1.html) -- emacs man page
- [use-package Documentation](https://github.com/jwiegley/use-package) -- declarative package configuration
