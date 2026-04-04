# lazygit (terminal UI for git)

A simple terminal-based UI for git commands that provides interactive staging, committing, branching, rebasing, and merge conflict resolution through keyboard-driven panels without memorizing complex git command syntax.

## Installation

### Package Managers

```bash
# macOS
brew install lazygit

# Arch Linux
pacman -S lazygit

# Ubuntu/Debian (via PPA)
LAZYGIT_VERSION=$(curl -s "https://api.github.com/repos/jesseduffield/lazygit/releases/latest" | grep -Po '"tag_name": "v\K[^"]*')
curl -Lo lazygit.tar.gz "https://github.com/jesseduffield/lazygit/releases/latest/download/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz"
tar xf lazygit.tar.gz lazygit && sudo install lazygit /usr/local/bin

# Go install
go install github.com/jesseduffield/lazygit@latest

# Nix
nix-env -iA nixpkgs.lazygit
```

## Panel Navigation

### Window Switching

```bash
# Main panels (press number or Tab/Shift-Tab):
#   1  Status
#   2  Files (staging area)
#   3  Branches
#   4  Commits
#   5  Stash

# Navigation within panels:
#   j/k or Up/Down    move up/down
#   h/l               collapse/expand or switch sub-panels
#   [ / ]              switch tabs within a panel
#   Enter              focus item / open detail
#   Escape             go back / close popup
#   ?                  show keybindings for current panel
#   q                  quit lazygit
```

## File Operations

### Staging and Unstaging

```bash
# In the Files panel (2):
#   Space        stage/unstage file
#   a            stage/unstage all files
#   Enter        view file diff, then stage individual hunks
#   d            discard changes (unstaged changes in file)
#   D            reset options menu (hard reset, mixed, soft)
#   e            open file in editor
#   o            open file in default application
#   i            add to .gitignore
#   c            commit staged changes
#   A            amend last commit

# Hunk staging (after Enter on a file):
#   Space        stage/unstage hunk
#   v            select lines mode
#   a            stage/unstage all hunks in file
#   Escape       back to files list
```

## Commits

### Creating and Managing Commits

```bash
# In Files panel:
#   c            commit (opens message editor)
#   C            commit with editor (full editor)
#   A            amend last commit
#   w            commit with pre-commit hook skip (--no-verify)

# In Commits panel (4):
#   Enter        view commit diff
#   Space        checkout commit
#   r            reword commit message
#   R            reword with editor
#   d            drop commit (during rebase)
#   e            edit commit (stop rebase at this commit)
#   s            squash commit into previous
#   f            fixup commit into previous (discard message)
#   p            pick commit (during rebase)
#   g            reset to this commit (soft/mixed/hard menu)
#   t            create tag at commit
#   T            tag with annotation
#   y            copy commit SHA to clipboard
#   o            open commit in browser
```

## Branches

### Branch Management

```bash
# In Branches panel (3):
#   n            new branch from current HEAD
#   Space        checkout branch
#   d            delete branch
#   D            force delete branch
#   r            rebase current branch onto selected
#   M            merge selected branch into current
#   R            rename branch
#   u            set/unset upstream
#   f            fast-forward branch to upstream
#   Enter        view branch commits

# Remote branches:
#   [ / ]        switch between Local / Remote / Tags tabs
#   f            fetch remote branch
#   n            new branch from remote
```

## Interactive Rebase

### Rebase Operations

```bash
# Start interactive rebase:
#   In Commits panel, press 'e' on the commit to stop at
#   Or press 'r' in Branches panel on the target branch

# During rebase (in Commits panel):
#   p            pick
#   s            squash
#   f            fixup
#   e            edit
#   d            drop
#   ctrl+j       move commit down
#   ctrl+k       move commit up

# Rebase controls:
#   m            view merge/rebase options
#   Enter        continue rebase
#   Escape       abort rebase (with confirmation)

# Fixup workflow:
#   1. Make your fix, stage it
#   2. Press 'F' to create fixup commit for selected commit
#   3. Press 'S' to squash all fixup commits (autosquash)
```

## Cherry-Pick

### Cherry-Pick Workflow

```bash
# Cherry-pick commits between branches:
#   1. Go to Commits panel (4)
#   2. Press 'C' (shift-c) to copy commit
#   3. Switch to target branch (Branches panel, Space)
#   4. Press 'V' (shift-v) to paste (cherry-pick)

# Cherry-pick range:
#   1. Press 'C' on first commit
#   2. Move to last commit, press 'C' again (adds to selection)
#   3. Switch branch and press 'V' to paste all
```

## Stash

### Stash Operations

```bash
# In Files panel:
#   s            stash all changes (with message prompt)
#   S            stash options (staged only, unstaged only, etc.)

# In Stash panel (5):
#   Space        apply stash entry
#   g            pop stash entry (apply + drop)
#   d            drop stash entry
#   Enter        view stash diff
#   n            new branch from stash
```

## Bisect

### Git Bisect in lazygit

```bash
# Start bisect:
#   In Commits panel, press 'b' to start bisect
#   Mark commit as good: 'b' then select 'good'
#   Mark commit as bad:  'b' then select 'bad'
#   lazygit automatically checks out the midpoint

# Continue bisecting:
#   After testing, press 'b' and mark good/bad
#   Repeat until the culprit commit is found

# Abort bisect:
#   Press 'b' and select 'reset'
```

## Worktrees

### Worktree Support

```bash
# In the Status panel or Branches panel:
#   w            open worktree menu
#   n            create new worktree
#   Enter        switch to worktree
#   d            delete worktree

# Worktrees allow working on multiple branches simultaneously
# without stashing or switching
```

## Filtering and Search

### Filter and Find

```bash
# In any panel:
#   /            start filtering (type to filter list)
#   Escape       clear filter

# In diff view:
#   /            search within diff
#   n            next match
#   N            previous match

# Commit filtering:
#   ctrl+s       filter commits by path
#   Enter        apply filter, shows only commits touching that path
```

## Custom Commands

### config.yml Custom Commands

```yaml
# ~/.config/lazygit/config.yml
customCommands:
  - key: "C"
    context: "files"
    command: "git cz"
    description: "commit with commitizen"
    subprocess: true

  - key: "<c-p>"
    context: "global"
    command: "git push --force-with-lease"
    description: "force push with lease"
    loadingText: "Pushing..."

  - key: "b"
    context: "files"
    command: "git blame {{.SelectedFile.Name}}"
    description: "blame file"
    subprocess: true

  - key: "E"
    context: "commits"
    command: "git revert {{.SelectedLocalCommit.Sha}}"
    description: "revert commit"
    prompts:
      - type: "confirm"
        title: "Revert"
        body: "Are you sure you want to revert {{.SelectedLocalCommit.Sha}}?"
```

## Configuration

### General Config

```yaml
# ~/.config/lazygit/config.yml
gui:
  theme:
    activeBorderColor:
      - green
      - bold
    inactiveBorderColor:
      - white
    selectedLineBgColor:
      - reverse
  showFileTree: true
  showRandomTip: false
  nerdFontsVersion: "3"
  showBottomLine: false
  sidePanelWidth: 0.3333

git:
  paging:
    colorArg: always
    pager: delta --dark --paging=never
  autoFetch: true
  autoRefresh: true
  branchLogCmd: "git log --graph --color=always --abbrev-commit --decorate --date=relative --pretty=medium {{branchName}} --"
  allBranchesLogCmd: "git log --graph --all --color=always --abbrev-commit --decorate --date=relative --pretty=medium"

os:
  editPreset: "nvim"
  # or: editPreset: "vscode"
```

### Delta Integration

```yaml
# Use delta as the diff pager in lazygit
git:
  paging:
    colorArg: always
    pager: delta --dark --paging=never
    useConfig: false
```

## Bulk Operations

### Multi-Select

```bash
# Multi-select in Files panel:
#   v            toggle multi-select mode
#   Space        toggle item selection (in multi-select)
#   a            select/deselect all

# After selecting multiple files:
#   Space        stage/unstage all selected
#   d            discard all selected
#   e            open all in editor
```

## Tips

- Press `?` in any panel to see all available keybindings for that context -- this is the fastest way to learn.
- Use `e` on a commit in the Commits panel to start an interactive rebase at that point.
- Set `git.paging.pager` to `delta --dark --paging=never` for syntax-highlighted diffs inside lazygit.
- Press `Enter` on a file to view its diff, then `Space` to stage individual hunks -- far easier than `git add -p`.
- Use `ctrl+j` / `ctrl+k` during interactive rebase to reorder commits by moving them up and down.
- The cherry-pick flow (Copy with `C`, switch branch, Paste with `V`) works across branches seamlessly.
- Press `x` to open the command log panel to see exactly which git commands lazygit is running.
- Custom commands in `config.yml` can use template variables like `{{.SelectedFile.Name}}` and `{{.SelectedLocalCommit.Sha}}`.
- Use `ctrl+s` in the Commits panel to filter commits by file path -- great for tracking changes to a specific file.
- Set `gui.nerdFontsVersion: "3"` if you have Nerd Fonts installed for better icon rendering.
- Press `@` to open the command log filtering options to show/hide specific git operations.

## See Also

- delta, fzf, bat, zsh

## References

- [lazygit GitHub Repository](https://github.com/jesseduffield/lazygit)
- [lazygit Keybindings Docs](https://github.com/jesseduffield/lazygit/blob/master/docs/keybindings)
- [lazygit Custom Commands](https://github.com/jesseduffield/lazygit/blob/master/docs/Custom_Command_Keybindings.md)
- [lazygit Config Reference](https://github.com/jesseduffield/lazygit/blob/master/docs/Config.md)
- [lazygit Tutorial Video (YouTube)](https://www.youtube.com/watch?v=CPLdltN7wgE)
