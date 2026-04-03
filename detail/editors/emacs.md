# The Theory of Emacs — Lisp Machine, Buffer Model, and Extensibility

> *Emacs is not an editor — it's a Lisp interpreter that happens to edit text. Its core is a single-threaded Emacs Lisp (Elisp) VM with cooperative multitasking. Every key press, menu item, and mode is a Lisp function. Buffers are gap-buffered text objects with overlays and text properties. The extensibility model is total: every behavior is a function that can be advised, replaced, or composed.*

---

## 1. Architecture — A Lisp Machine

### The Core Loop

```
┌──────────────────────────────────────────┐
│              Emacs Lisp VM               │
│                                          │
│  Event Loop:                             │
│    1. Read key/event from input          │
│    2. Look up binding in current keymaps │
│    3. Call bound Lisp function            │
│    4. Redisplay (update screen)          │
│    5. Repeat                             │
└──────────────────────────────────────────┘
```

### C Core vs Lisp Layer

| Layer | Lines of Code | Provides |
|:------|:-------------|:---------|
| C core | ~300K | Lisp interpreter, display engine, I/O, GC |
| Elisp | ~1.5M+ | Everything else: modes, keymaps, UI, packages |

The C core provides **primitives** (subroutines). Everything built on top is Lisp.

### The `command-execute` Chain

```
key press → key-binding-lookup → interactive function → side effects → redisplay
```

Every command is a Lisp function marked `(interactive)`. Even `self-insert-command` (typing a character) is a function call.

---

## 2. Buffer Model — Gap Buffer

### Data Structure

Emacs buffers use a **gap buffer**: a contiguous array with a gap at the cursor position.

```
Text: "Hello World"
             ^cursor here (after "Hello ")

Memory: [H|e|l|l|o| |_|_|_|_|_|W|o|r|l|d]
                      ^^^^^^^^^^^
                      gap (unused space)
```

### Operations and Complexity

| Operation | Complexity | Why |
|:----------|:-----------|:----|
| Insert at cursor | $O(1)$ amortized | Insert into gap |
| Delete at cursor | $O(1)$ | Expand gap |
| Move cursor by $k$ | $O(k)$ | Move gap (copy characters) |
| Move cursor to distant point | $O(n)$ | Move entire gap |
| Search | $O(n)$ | Must handle gap boundary |

### Gap Buffer vs Alternatives

| Structure | Insert | Delete | Random Access | Memory |
|:----------|:-------|:-------|:-------------|:-------|
| Gap buffer | $O(1)$* | $O(1)$* | $O(1)$ | $O(n)$ |
| Rope | $O(\log n)$ | $O(\log n)$ | $O(\log n)$ | $O(n)$ |
| Piece table | $O(1)$ | $O(1)$ | $O(\log n)$ | $O(n + m)$ |

*Amortized at cursor; $O(n)$ if cursor jumps.

Gap buffers are optimal for sequential editing (typing, deleting at the cursor) which is the dominant use case.

### Buffer Properties

Each buffer carries metadata:

| Property | Description |
|:---------|:-----------|
| Major mode | One per buffer (defines language/behavior) |
| Minor modes | Zero or more (add features) |
| Local variables | Per-buffer variable bindings |
| Keymaps | Local keymap (overrides global) |
| Text properties | Per-character metadata (font, color, read-only) |
| Overlays | Regions with display properties |
| Markers | Positions that track text movement |

---

## 3. Emacs Lisp — The Extension Language

### Key Characteristics

| Feature | Elisp | Common Lisp | Scheme |
|:--------|:------|:------------|:-------|
| Scoping | Dynamic (default) + lexical (opt-in) | Lexical | Lexical |
| Typing | Dynamic | Dynamic | Dynamic |
| Tail call optimization | No | Implementation-dependent | Required |
| Concurrency | Cooperative (single-threaded) | Varies | Varies |
| GC | Mark-and-sweep | Varies | Varies |

### Dynamic vs Lexical Scoping

```elisp
;; Dynamic scoping (traditional):
(defvar x 10)
(defun show-x () (message "%d" x))
(let ((x 20)) (show-x))   ; prints 20 (dynamic lookup)

;; Lexical scoping (modern, per-file opt-in):
;;; -*- lexical-binding: t -*-
(let ((x 20))
  (lambda () x))           ; captures x = 20 (closure)
```

Dynamic scoping enables the **advice** system: any function can be modified by temporarily rebinding variables it uses.

### Advice System

```elisp
;; Add behavior before a function:
(advice-add 'save-buffer :before
  (lambda (&rest _)
    (delete-trailing-whitespace)))

;; Add behavior after:
(advice-add 'find-file :after
  (lambda (&rest _)
    (message "Opened: %s" (buffer-file-name))))
```

Advice types: `:before`, `:after`, `:around`, `:override`, `:filter-args`, `:filter-return`.

---

## 4. Keymaps — Hierarchical Lookup

### Keymap Precedence

Key lookup searches keymaps in order:

```
1. Overriding terminal local map (highest priority)
2. overriding-local-map (if set)
3. Text property keymap at point
4. Minor mode keymaps (in reverse activation order)
5. Buffer-local keymap (major mode)
6. Global keymap (lowest priority)
```

### Keymap Data Structure

A keymap is a **nested alist** (association list):

```elisp
(keymap
  (?\C-x keymap
    (?\C-f . find-file)
    (?\C-s . save-buffer))
  (?\C-c keymap
    ...)
  (?a . self-insert-command))
```

### Prefix Keys

`C-x` is a **prefix key** — it starts a two-key sequence. The first key resolves to a keymap (not a command), and the second key is looked up in that keymap.

$$\text{C-x C-f} \to \text{keymap}[\text{C-x}][\text{C-f}] = \text{find-file}$$

---

## 5. Major and Minor Modes

### Major Mode

Each buffer has exactly **one** major mode that defines:
- Syntax table (what counts as a word, comment, string)
- Keymap (mode-specific bindings)
- Font-lock rules (syntax highlighting)
- Indentation function
- Hooks (functions run when mode activates)

### Mode Derivation

```elisp
(define-derived-mode go-mode prog-mode "Go"
  "Major mode for Go source code."
  (setq-local comment-start "// ")
  (setq-local indent-line-function #'go-indent-line))
```

Inheritance: `go-mode` → `prog-mode` → `text-mode` → `fundamental-mode`.

### Minor Modes

Independently toggleable features:

| Minor Mode | Purpose |
|:-----------|:--------|
| `display-line-numbers-mode` | Line numbers |
| `flycheck-mode` | On-the-fly syntax checking |
| `company-mode` | Completion |
| `undo-tree-mode` | Visual undo tree |
| `evil-mode` | Vim emulation |

---

## 6. Process and Async Model

### Single-Threaded with Cooperative Multitasking

Emacs runs on a **single thread**. Long-running operations block the UI. Solutions:

| Mechanism | Description | Use Case |
|:----------|:-----------|:---------|
| `start-process` | Async subprocess | Running compilers, linters |
| Process filters | Callbacks for subprocess output | Incremental output processing |
| Process sentinels | Callbacks for process state changes | Cleanup on exit |
| Timers | Deferred execution | Idle tasks, polling |
| `async.el` | Run Elisp in subprocess | CPU-intensive Elisp |

### Subprocess Communication

```elisp
(make-process
  :name "my-server"
  :command '("gopls")
  :connection-type 'pipe
  :filter (lambda (proc output)
            (process-output proc output))
  :sentinel (lambda (proc event)
              (message "Process %s: %s" proc event)))
```

### Native Compilation (Emacs 28+)

Emacs can compile Elisp to native code via **libgccjit**:

| Execution Mode | Speed |
|:---------------|:------|
| Interpreted Elisp | Baseline |
| Byte-compiled (`.elc`) | ~3-5x faster |
| Native-compiled (`.eln`) | ~10-20x faster |

---

## 7. Display Engine

### Redisplay Algorithm

After each command, Emacs redraws only changed portions:

1. Compare current buffer state with last display
2. Compute minimal set of changes
3. Update display (terminal escape sequences or GUI calls)

### Text Properties vs Overlays

| Feature | Text Properties | Overlays |
|:--------|:---------------|:---------|
| Attached to | Characters | Buffer regions (start-end) |
| Move with text | Yes | Start/end markers track |
| Performance | Fast (part of buffer) | Slower (separate list) |
| Use case | Font-lock highlighting | Highlights, annotations |

### Font-Lock (Syntax Highlighting)

Font-lock works by:
1. Defining **keywords** (regex → face mappings)
2. Running fontification on visible region (lazy)
3. Applying text properties for faces

```elisp
(font-lock-add-keywords 'python-mode
  '(("\\<TODO\\>" 0 'font-lock-warning-face t)))
```

---

## 8. Package System

### Package Sources

| Source | URL | Packages |
|:-------|:----|:---------|
| GNU ELPA | elpa.gnu.org | ~400 (FSF-approved) |
| NonGNU ELPA | elpa.nongnu.org | ~200 |
| MELPA | melpa.org | ~5500 (community) |

### Package Management

```elisp
;; Built-in package.el:
(package-install 'magit)

;; use-package (built-in since Emacs 29):
(use-package magit
  :ensure t
  :bind ("C-x g" . magit-status))
```

### `use-package` Lazy Loading

```elisp
(use-package python-mode
  :defer t           ; load only when needed
  :mode "\\.py\\'"   ; auto-activate for .py files
  :hook (python-mode . eglot-ensure)  ; attach LSP
  :config
  (setq python-indent-offset 4))
```

---

## 9. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Core model | Lisp interpreter + display engine (in C) |
| Buffer structure | Gap buffer (O(1) insert at cursor) |
| Extension language | Emacs Lisp (dynamic scoping, advice system) |
| Keymap lookup | Hierarchical: text property → minor mode → major mode → global |
| Concurrency | Single-threaded, async subprocesses |
| Modes | 1 major mode + N minor modes per buffer |
| Native compilation | Elisp → native via libgccjit (10-20x speedup) |
| Package ecosystem | ~6000 packages across ELPA + MELPA |

---

*Emacs' extensibility principle is absolute: if something happens in Emacs, a Lisp function did it, and you can replace that function. This is not a plugin API — it's the editor itself. The cost is complexity and a learning curve. The reward is an editor that adapts to you completely, not one that you adapt to.*
