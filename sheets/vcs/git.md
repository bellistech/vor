# Git (Version Control System)

> Distributed version control — track changes, branch, merge, recover. The reference that keeps you out of the browser.

## Setup

### Install

```bash
# macOS (Homebrew)
brew install git
brew upgrade git

# macOS (Xcode CLT — minimum bundled)
xcode-select --install

# Debian / Ubuntu
sudo apt update
sudo apt install -y git

# RHEL / Fedora / Rocky / Alma
sudo dnf install -y git

# Arch
sudo pacman -S git

# Alpine
apk add git

# Windows
winget install --id Git.Git -e --source winget
choco install git -y

# from source (when distro is ancient)
sudo apt install -y dh-autoreconf libcurl4-gnutls-dev libexpat1-dev gettext libz-dev libssl-dev
git clone https://github.com/git/git.git
cd git && make prefix=/usr/local all && sudo make prefix=/usr/local install
```

### Version

```bash
git --version                          # e.g. git version 2.46.0
git version --build-options            # show build features (sha1 backend, etc)
```

Most modern features assume Git 2.40+. Anything older is a hazard. Notable cutoffs:
- `git switch` / `git restore` — stable since 2.23 (2019)
- `--force-with-lease=ref:expect` — needs 2.30+
- `safe.directory` — 2.35.2+
- `merge.conflictStyle=zdiff3` — 2.35+
- `git maintenance` — 2.31+
- SHA-256 repos — experimental, 2.29+

### Identity

```bash
git config --global user.name "Alice Smith"
git config --global user.email "alice@example.com"

# verify
git config --global --get user.name
git config --global --get user.email

# per-repo override (e.g. work email)
cd ~/code/work-repo
git config --local user.email "alice@work.com"
```

### Help

```bash
git help                               # top-level summary
git help -a                            # list every git command
git help <cmd>                         # man page (e.g. git help commit)
git <cmd> -h                           # one-screen synopsis
git help --web rebase                  # open browser man page (terminal-only? avoid)
git help everyday                      # essential everyday commands
git help workflows                     # recommended workflows
git help glossary                      # terminology
```

## Configuration

### Scopes

Three scopes, evaluated lowest to highest precedence: system → global → local.

```bash
git config --system <key> <value>      # /etc/gitconfig (all users on host)
git config --global <key> <value>      # ~/.gitconfig or ~/.config/git/config (per-user)
git config --local  <key> <value>      # .git/config inside this repo
git config --worktree <key> <value>    # per-worktree (needs extensions.worktreeConfig=true)
```

Inspect the merged view:

```bash
git config --list                       # everything that applies here
git config --list --show-origin         # which file each setting came from
git config --list --show-scope          # tag with scope (system/global/local)
git config --get-all remote.origin.url  # all values for multivar key
git config --get-regexp '^alias\.'      # all aliases
git config --unset user.signingkey      # remove a single key
git config --unset-all url.<base>.insteadOf
```

### Highly recommended baseline

```bash
git config --global init.defaultBranch main
git config --global pull.rebase true
git config --global push.autoSetupRemote true
git config --global push.default current
git config --global fetch.prune true
git config --global rerere.enabled true
git config --global rebase.autostash true
git config --global rebase.autosquash true
git config --global merge.conflictstyle zdiff3
git config --global diff.algorithm histogram
git config --global diff.colorMoved zebra
git config --global color.ui auto
git config --global commit.verbose true
git config --global help.autocorrect prompt
git config --global core.editor "vim"
git config --global core.pager "less -FRX"
git config --global protocol.version 2
git config --global transfer.fsckobjects true
git config --global fetch.fsckobjects true
git config --global receive.fsckobjects true
git config --global tag.sort -version:refname
git config --global branch.sort -committerdate
git config --global log.date iso-local
```

What these do:

- `init.defaultBranch main` — new repos start with `main`, not `master`.
- `pull.rebase true` — `git pull` becomes `fetch + rebase`, never creating noise merge commits on your feature branches.
- `push.autoSetupRemote true` — first push of a new branch doesn't need `-u origin <branch>`.
- `fetch.prune true` — `git fetch` automatically deletes remote-tracking refs whose remote branch was deleted.
- `rerere.enabled true` — Reuse Recorded Resolution: git remembers how you solved a conflict and replays the resolution next time.
- `rebase.autostash true` — `git rebase main` on dirty tree auto-stashes and re-applies.
- `rebase.autosquash true` — `git rebase -i` automatically reorders `fixup!` / `squash!` commits.
- `merge.conflictstyle zdiff3` — conflict markers include the merge base, dramatically clearer than `merge`.
- `diff.algorithm histogram` — better minimal diffs than the default Myers.
- `diff.colorMoved zebra` — moved blocks are coloured separately from added/removed.
- `commit.verbose true` — `git commit` shows the staged diff in the editor for review.
- `protocol.version 2` — modern smart-protocol, faster ref negotiation on huge repos.
- `transfer.fsckobjects true` — verify object integrity on transfer (slows clones, prevents corruption).

### Conditional includes

`includeIf` switches sections based on the worktree path. Perfect for separating personal and work identity without per-repo config.

```bash
# in ~/.gitconfig
[user]
    name = Alice Smith
    email = alice@personal.com

[includeIf "gitdir:~/code/work/"]
    path = ~/.gitconfig-work

[includeIf "gitdir:~/code/oss/"]
    path = ~/.gitconfig-oss

[includeIf "hasconfig:remote.*.url:git@github.com:work-org/**"]
    path = ~/.gitconfig-work
```

```bash
# ~/.gitconfig-work
[user]
    email = alice@work.com
    signingkey = ABC123DEF456
[commit]
    gpgsign = true
```

`hasconfig:` matcher requires Git 2.36+.

### Editor and pager

```bash
git config --global core.editor "vim"
git config --global core.editor "nano"
git config --global core.editor "code --wait"             # VS Code
git config --global core.editor "subl -n -w"              # Sublime
git config --global core.editor "nvim"
git config --global core.editor "emacs -nw"

git config --global core.pager "less -FRX"                # F=quit if one screen, R=raw colours, X=no clear screen
git config --global core.pager "delta"                    # https://github.com/dandavison/delta — pretty diffs
git config --global pager.diff false                      # disable pager for diff specifically
```

Per-command pager toggle:

```bash
git --no-pager log
git --paginate log
```

### Color

```bash
git config --global color.ui auto                         # default; colourize when output is a TTY
git config --global color.ui always                       # force colour even when piped
git config --global color.ui never                        # disable
git config --global color.diff.meta "yellow bold"
git config --global color.diff.frag "magenta bold"
git config --global color.diff.old "red bold"
git config --global color.diff.new "green bold"
git config --global color.status.added "green"
git config --global color.status.changed "yellow"
git config --global color.status.untracked "red"
```

### Credentials

```bash
git config --global credential.helper cache               # in-memory, 15 min default
git config --global credential.helper "cache --timeout=3600"
git config --global credential.helper store               # plaintext ~/.git-credentials — avoid
git config --global credential.helper osxkeychain         # macOS Keychain
git config --global credential.helper manager             # Git Credential Manager (cross-platform)
git config --global credential.helper "/usr/libexec/git-core/git-credential-libsecret"   # GNOME

# per-host helper
git config --global credential.https://github.com.helper osxkeychain
git config --global credential.https://gitlab.com.helper "cache --timeout=7200"
```

## Aliases

Aliases live in `[alias]` and are invoked as `git <alias>`.

```bash
git config --global alias.co checkout
git config --global alias.br branch
git config --global alias.ci commit
git config --global alias.st 'status -sb'
git config --global alias.unstage 'reset HEAD --'
git config --global alias.last 'log -1 HEAD'
git config --global alias.amend 'commit --amend --no-edit'
git config --global alias.wip '!git add -A && git commit -m "WIP"'
git config --global alias.save '!git add -A && git commit -m "SAVEPOINT"'
git config --global alias.undo 'reset HEAD~1 --mixed'
git config --global alias.nuke '!git reset --hard && git clean -fdx'
git config --global alias.aliases 'config --get-regexp ^alias\\.'
git config --global alias.contributors 'shortlog -sne --no-merges'
git config --global alias.recent 'for-each-ref --sort=-committerdate --format="%(refname:short) %(committerdate:relative)" refs/heads/'
```

The classic pretty log:

```bash
git config --global alias.lg "log --graph --abbrev-commit --decorate --format=format:'%C(bold blue)%h%C(reset) - %C(bold green)(%ar)%C(reset) %C(white)%s%C(reset) %C(dim white)- %an%C(reset)%C(auto)%d%C(reset)' --all"
git config --global alias.ll "log --pretty=format:'%C(yellow)%h%C(reset) %C(blue)%ad%C(reset) %C(red)%d%C(reset) %s %C(green)[%an]%C(reset)' --decorate --date=short"
git config --global alias.hist "log --pretty=format:'%h %ad | %s%d [%an]' --graph --date=short"
```

Shell aliases (prefix with `!`):

```bash
git config --global alias.s '!git status -sb'
git config --global alias.publish '!git push -u origin "$(git symbolic-ref --short HEAD)"'
git config --global alias.unpublish '!git push origin :"$(git symbolic-ref --short HEAD)"'
git config --global alias.cleanup '!git branch --merged | grep -v "\\*\\|main\\|master\\|develop" | xargs -n 1 git branch -d'
git config --global alias.root 'rev-parse --show-toplevel'
git config --global alias.churn '!git log --all -M -C --name-only --format=format: "$@" | sort | grep -v "^$" | uniq -c | sort -nr'
git config --global alias.restore-staged 'reset HEAD --'
```

List your aliases:

```bash
git config --get-regexp ^alias\.
git aliases                            # if you defined the alias above
```

Useful trick — alias for help:

```bash
git config --global alias.h '!git help'
```

## Repository Init / Clone

### init

```bash
git init                               # in current directory (creates .git)
git init my-project                    # creates ./my-project/.git
git init --initial-branch=main         # explicit default branch
git init -b main                       # short form (Git 2.28+)
git init --bare repo.git               # bare repo for serving (no working tree)
git init --shared=group repo.git       # group-writable bare repo
git init --separate-git-dir=/var/git/foo.git ~/work/foo
git init --template=~/.git-templates   # use a template (hooks, config, etc.)
```

A bare repo has no working tree — only the contents normally inside `.git`. It's what servers expose.

### clone

```bash
git clone https://github.com/user/repo.git
git clone git@github.com:user/repo.git              # SSH
git clone https://github.com/user/repo.git mydir    # custom directory
git clone --branch v2.5.0 repo-url                  # specific branch or tag (== --branch == -b)
git clone --single-branch --branch main repo-url    # only that branch's history
git clone --depth 1 repo-url                        # shallow — latest commit only
git clone --depth 50 --no-single-branch repo-url    # last 50 commits, all branches
git clone --filter=blob:none repo-url               # partial clone — fetch trees, lazy fetch blobs
git clone --filter=tree:0 repo-url                  # even more aggressive — only commits
git clone --recurse-submodules repo-url             # also clone submodules
git clone --recurse-submodules -j8 repo-url         # parallel submodule clone
git clone --bare repo-url                           # mirror skeleton
git clone --mirror repo-url                         # like --bare + all refs (useful for migration)
git clone --reference ~/cache/repo.git repo-url     # share objects with local cache (saves bandwidth)
git clone --dissociate --reference ~/cache repo-url # use cache during clone, then unlink
git clone --separate-git-dir=/var/git/foo.git url   # split .git from working tree
```

Shallow clones cannot push or fetch from older history (the missing commits aren't there). Convert with:

```bash
git fetch --unshallow                  # backfill all history
git fetch --depth=1000                 # extend depth
```

## Status / Diff / Log

### status

```bash
git status                             # full output
git status -s                          # short format (XY filename)
git status -sb                         # short + branch info
git status --porcelain                 # stable machine-readable v1
git status --porcelain=v2 --branch     # stable v2, with branch info
git status --ignored                   # also list ignored files
git status -uno                        # don't show untracked files (faster)
git status --untracked-files=normal    # default
git status --untracked-files=all       # show contents of untracked dirs
```

Short format glyphs (XY = staged/unstaged):

```
M  modified
A  added
D  deleted
R  renamed
C  copied
U  updated but unmerged
?? untracked
!! ignored
```

### diff

```bash
git diff                                # working tree vs index (unstaged)
git diff --staged                       # index vs HEAD (what `git commit` will record)
git diff --cached                       # alias for --staged
git diff HEAD                           # working tree vs HEAD (everything uncommitted)
git diff main..feature                  # tip-of-main vs tip-of-feature (set diff)
git diff main...feature                 # changes on feature since it diverged from main
git diff HEAD~3 HEAD                    # last 3 commits
git diff --stat                         # summary: files changed, insertions, deletions
git diff --shortstat                    # one-line summary
git diff --numstat                      # machine-readable: added\tremoved\tfile
git diff --name-only                    # filenames only
git diff --name-status                  # filenames + status (A/M/D/R)
git diff --check                        # warn about whitespace errors
git diff --word-diff                    # inline word-level diff
git diff --word-diff=color              # use colour to mark adds/dels
git diff --color-words                  # word diff, colour only
git diff --color-words='[A-Za-z_]+'     # custom word regex (per-language)
git diff --no-color                     # force off
git diff -w                             # ignore all whitespace
git diff -b                             # ignore whitespace within lines
git diff --ignore-blank-lines
git diff --diff-filter=ACDMR            # only Added/Copied/Deleted/Modified/Renamed
git diff --find-renames=90%             # detect renames (% similarity)
git diff -- '*.go' ':!vendor/'          # pathspec — match *.go but exclude vendor
git diff --submodule=log                # readable submodule diffs
git diff --binary > patch.diff          # include binary changes (apply with `git apply`)
git diff > patch.diff && git apply patch.diff   # roundtrip
```

`a..b` vs `a...b`:
- `a..b` — commits reachable from `b` but not `a`. Set difference.
- `a...b` — commits reachable from either, but not both. Symmetric difference. For diffs, this means "what `b` did since the merge base", which is usually what you want when reviewing a feature branch.

### log

```bash
git log                                 # full log
git log --oneline                       # one line per commit
git log --oneline --decorate            # show ref names (HEAD, branches, tags)
git log --oneline --decorate --graph    # ASCII graph
git log --oneline --decorate --graph --all   # all refs, not just current branch
git log --oneline -n 20                 # last 20
git log -p                              # show patches
git log --stat                          # show file change stats
git log --shortstat                     # condensed stats
git log --name-only                     # filenames only
git log --name-status                   # filenames with status
git log --since="2 weeks ago"
git log --until="2024-01-01"
git log --after="2024-01-01" --before="2024-04-01"
git log --author="Alice"                # commits by Alice (regex)
git log --author="^Alice$"
git log --committer="Alice"             # vs author (e.g. when rebased by someone else)
git log --grep="fix:"                   # commit message regex
git log --grep="bug" -i                 # case-insensitive
git log --invert-grep --grep="WIP"      # exclude WIP
git log --all-match --grep=foo --grep=bar  # both terms
git log --merges                        # only merge commits
git log --no-merges                     # exclude merge commits
git log --first-parent                  # follow only first parent (linear branch view)
git log -- src/auth.go                  # history of one file
git log --follow -- renamed.go          # follow across renames
git log -L :myFunc:src/foo.go           # function history (needs hunk header)
git log -L 10,30:src/foo.go             # line-range history
git log -S "needle"                     # pickaxe: commits where "needle" count changed
git log -G "regex"                      # commits whose diff matches regex
git log -p -S 'API_KEY'                 # accidental secret hunt
git log --reverse                       # oldest first
git log HEAD..origin/main               # what upstream has that you don't
git log origin/main..HEAD               # what you have that upstream doesn't (ahead)
git log --left-right HEAD...origin/main # both directions, marked < or >
git log --pretty=fuller                 # show author + committer date and identity
git log --pretty=oneline                # SHA + subject only
git log --pretty=format:'%h %ad | %s%d [%an]' --date=short
git log --decorate=full                 # full ref names
git log --abbrev-commit                 # short SHA
git log --abbrev=12                     # 12-char SHA
git log --topo-order                    # topological order (no chrono interleave)
git log --date-order
git log --reverse --pretty=format:'%h %s' main..feature
git log --merge                         # commits relevant to current merge conflict
git log --cherry-pick --left-right A...B   # find unique commits, ignoring cherry-picks
```

Format placeholders for `--pretty=format:`:

```
%H   full SHA            %h   short SHA
%T   tree SHA            %t   short tree SHA
%P   parent SHAs         %p   short parents
%an  author name         %ae  author email
%ad  author date         %ar  author date relative
%aI  author date ISO 8601 strict
%cn  committer name      %ce  committer email
%cd  committer date      %cr  committer date relative
%s   subject             %b   body
%d   ref decoration (parens)  %D   without parens
%G?  signature status    %GS  signer
%n   newline             %%   literal percent
%C(red)  colour          %C(reset)
```

`git shortlog`:

```bash
git shortlog                           # group commits by author
git shortlog -sn                       # count + name, summary
git shortlog -sne                      # include emails
git shortlog -sn --no-merges
git shortlog -sn --since="1 year ago"
```

## Stage / Unstage / Reset

### add

```bash
git add file.go                        # stage one file
git add src/                           # everything under src/
git add -A                             # stage everything in working tree (incl. deletes)
git add --all                          # same as -A
git add .                              # stage current dir + below (also captures deletes 2.0+)
git add -u                             # update tracked files only (incl. deletes), skip new
git add -p                             # interactive — review hunks
git add -i                             # interactive menu UI
git add -N file.go                     # intent-to-add — track file but stage no content (lets diff show new file)
git add -f ignored.log                 # force add an ignored file
git add ':(exclude)vendor/' .          # pathspec — add everything except vendor
```

Inside `git add -p`, the prompt accepts:

```
y - stage this hunk
n - skip this hunk
q - quit; do not stage this hunk or any remaining
a - stage this hunk and all later hunks in the file
d - skip this hunk and all later hunks in the file
g - select a hunk to go to
/ - search for a hunk matching regex
j - leave undecided, see next undecided hunk
J - leave undecided, see next hunk
k - leave undecided, see previous undecided hunk
K - leave undecided, see previous hunk
s - split current hunk into smaller hunks
e - manually edit the current hunk
? - help
```

### unstage

```bash
git restore --staged file.go           # modern; unstage but keep changes (Git 2.23+)
git reset HEAD file.go                 # legacy way to unstage one file
git reset                              # unstage everything (--mixed default)
git reset --                           # explicit
```

### reset semantics

`git reset` rewrites HEAD, optionally rewriting the index and working tree.

```bash
git reset --soft <commit>              # move HEAD; keep index + working tree
git reset --mixed <commit>             # move HEAD; reset index; keep working tree (default)
git reset <commit>                     # same as --mixed
git reset --hard <commit>              # move HEAD; reset index AND working tree (destructive)
git reset --keep <commit>              # like --hard but abort if uncommitted changes would be lost
git reset --merge <commit>             # like --keep but reset paths that differ between index and target
```

| Mode      | Moves HEAD | Resets Index | Resets Worktree |
|-----------|------------|--------------|-----------------|
| `--soft`  | yes        | no           | no              |
| `--mixed` | yes        | yes          | no              |
| `--hard`  | yes        | yes          | yes             |

`--hard` is the most-feared command in Git. Always verify your reflog escape hatch before pulling the trigger:

```bash
git reflog                             # last 90 days of HEAD movement
git reset --hard HEAD@{1}              # back one move
git reset --hard ORIG_HEAD             # back to pre-reset HEAD
```

## Commit

```bash
git commit                              # opens editor
git commit -m "Add user auth"           # one-line message
git commit -m "Subject" -m "Body line 1" -m "Body line 2"  # multi-paragraph
git commit -a -m "msg"                  # auto-stage tracked files (NOT new files)
git commit -am "msg"                    # short
git commit --amend                      # rewrite tip commit (NEVER on shared branches)
git commit --amend --no-edit            # add to last commit, keep message
git commit --amend -m "new message"     # change message only
git commit --amend --reset-author       # reset author/date to now
git commit --allow-empty -m "Trigger CI"
git commit --allow-empty-message -m ""  # really? but allowed
git commit --signoff                    # appends "Signed-off-by: Name <email>" (DCO)
git commit -s                           # short for --signoff
git commit -S                           # GPG-sign (uses user.signingkey)
git commit -S -s                        # signed off + GPG-signed
git commit --no-verify                  # skip pre-commit + commit-msg hooks
git commit -n                           # short for --no-verify
git commit --fixup=<sha>                # creates "fixup! <subject>" commit
git commit --squash=<sha>               # creates "squash! <subject>" commit
git commit --fixup=amend:<sha>          # 2.32+: fixup that ALSO edits message
git commit --fixup=reword:<sha>         # 2.32+: only edits message
git commit -C HEAD~                     # reuse message from another commit
git commit -c HEAD~                     # reuse with editor
git commit --date="2024-01-15T10:00:00" # override author date
git commit --author="Alice <a@x.com>"   # override author
git commit -F message.txt               # message from file
git commit --cleanup=strip              # strip blank lines and trailing whitespace
git commit --cleanup=verbatim           # don't touch message at all
git commit --interactive                # interactive staging then commit
git commit -p                           # patch-mode commit (add -p + commit)
```

After `commit --fixup=<sha>`, run `git rebase -i --autosquash <sha>~` (or set `rebase.autosquash=true` and run any interactive rebase that includes that commit) to fold the fixup back in.

Commit messages: 50-char subject, blank line, 72-col-wrapped body. Imperative mood ("Fix" not "Fixed").

## Branch

```bash
git branch                             # list local branches; * marks current
git branch -a                          # all (local + remote-tracking)
git branch -r                          # remote-tracking only
git branch -vv                         # verbose: SHA, upstream, ahead/behind
git branch --list 'feat*'              # glob filter
git branch --merged                    # branches whose tip is reachable from HEAD (safe to delete)
git branch --no-merged                 # branches with unmerged work
git branch --merged main               # branches merged into main
git branch --contains <sha>            # branches containing this commit
git branch --no-contains <sha>
git branch --points-at <sha>           # branches whose tip IS this commit
git branch --sort=-committerdate       # by recent activity
git branch --format='%(refname:short) %(committerdate:relative)'

git branch feature/auth                # create from HEAD
git branch feature/auth main           # create from main
git branch feature/auth <sha>          # create from specific commit

git branch -d feature/auth             # delete (refuses if unmerged)
git branch -D feature/auth             # force delete
git branch --delete --force feature/auth   # long form

git branch -m new-name                 # rename current branch
git branch -m old-name new-name        # rename a branch
git branch -M new-name                 # force rename (overwrite if exists)

git branch -c new-copy                 # copy current branch
git branch -c old new                  # copy old to new

git branch -u origin/main              # set upstream for current
git branch --set-upstream-to=origin/main  # long form
git branch --unset-upstream            # remove tracking

git branch --edit-description          # opens editor — written to branch.<name>.description
```

Cleanup merged branches except main/develop:

```bash
git branch --merged main | grep -v -E '(\*|main|master|develop)$' | xargs -r git branch -d
```

## Switch / Checkout

`git switch` and `git restore` (Git 2.23+) split the overloaded `git checkout` into two: switching branches and restoring files.

### switch

```bash
git switch main                        # switch to existing branch
git switch -c feature/auth             # create + switch
git switch -C feature/auth             # create + switch, reset if it exists
git switch -c feature/auth main        # create from main
git switch --detach <sha>              # detached HEAD at sha
git switch -                           # switch to last branch (like `cd -`)
git switch --orphan new-branch         # new branch with no parent (great for gh-pages)
git switch --discard-changes main      # blow away local changes (DANGEROUS)
git switch -t origin/feat              # create local tracking branch from remote
```

### restore

```bash
git restore file.go                    # discard unstaged changes (working tree -> index)
git restore --staged file.go           # unstage (index -> HEAD)
git restore --source=HEAD~3 file.go    # restore file as of 3 commits ago
git restore --source=main -- file.go   # restore from main
git restore --worktree --staged file.go   # discard staged + unstaged
git restore -p                         # patch-mode discard
git restore .                          # discard all unstaged in cwd
```

### checkout (legacy but still supported)

```bash
git checkout main                      # switch branch
git checkout -b new                    # create + switch
git checkout -B new                    # create or reset to HEAD + switch
git checkout <sha>                     # detached HEAD
git checkout --orphan new              # new branch with no parent
git checkout -- file.go                # restore file (legacy `git restore`)
git checkout HEAD~3 -- file.go         # checkout file from 3 commits ago
git checkout --theirs file.go          # take "their" version during conflict
git checkout --ours file.go            # take "our" version during conflict
git checkout -                         # last branch
```

Detached HEAD is fine for inspection, but commits there are unreachable once you switch away — except via reflog. Always make a branch if you want to keep work:

```bash
git switch -c my-experiment
```

## Merge

```bash
git merge feature/auth                 # merge feature/auth INTO current branch
git merge --no-ff feature/auth         # always create a merge commit, even if FF possible
git merge --ff-only feature/auth       # refuse if non-fast-forward
git merge --squash feature/auth        # bring in changes as a single un-committed change
git merge --no-commit feature/auth     # merge but pause before committing
git merge --no-edit feature/auth       # skip editor on merge commit
git merge -X ours feature/auth         # in conflicts, prefer our side
git merge -X theirs feature/auth       # in conflicts, prefer their side
git merge -X ignore-space-change feature/auth
git merge -X ignore-all-space feature/auth
git merge -X renormalize feature/auth  # apply line-ending normalization
git merge -X patience feature/auth     # use patience diff algorithm
git merge -s recursive feature/auth    # default merge strategy
git merge -s ort feature/auth          # 2.34+ default; faster recursive replacement
git merge -s octopus feat1 feat2 feat3 # merge multiple branches at once (no conflicts)
git merge -s ours unwanted-branch      # mark merged but keep our content (NOT -X ours)
git merge -s subtree feature/auth      # subtree merge (rarely needed)
git merge --allow-unrelated-histories  # combine repos that share no commits
git merge --abort                      # cancel in-progress merge
git merge --continue                   # finish merge after resolving (Git 2.12+)
git merge --quit                       # leave merge state but keep working tree changes (2.11+)
```

`-s ours` is wildly different from `-X ours`. The strategy `-s ours` discards the other side entirely; the option `-X ours` only auto-resolves conflicts in our favour.

Fast-forward (FF) merge: when current is an ancestor of feature, "merging" just moves the pointer. No merge commit. Some teams use `--no-ff` to preserve branch shape; trunk-based teams prefer FF.

## Rebase

```bash
git rebase main                        # replay current branch onto main
git rebase main feature                # replay feature onto main
git rebase --onto main old-base feature   # replay commits in (old-base..feature) onto main
git rebase -i HEAD~5                   # interactive: edit last 5 commits
git rebase -i main                     # interactive from merge base with main
git rebase --autosquash -i HEAD~10     # auto-handle fixup!/squash! commits
git rebase --autostash main            # stash dirty tree, rebase, unstash
git rebase --no-verify                 # skip pre-rebase hook
git rebase --exec="make test" main     # run command after each replayed commit
git rebase -x "make test" main         # short
git rebase --root                      # rebase from the very first commit
git rebase --keep-empty                # preserve empty commits
git rebase --rebase-merges             # preserve merge commits
git rebase --preserve-merges           # legacy, replaced by --rebase-merges
git rebase --update-refs               # 2.38+: update intermediate branches that point inside the range
git rebase --signoff                   # add Signed-off-by to all replayed commits
git rebase --gpg-sign                  # sign replayed commits

# during a rebase
git rebase --continue                  # after resolving conflicts
git rebase --skip                      # skip current commit
git rebase --abort                     # cancel rebase, restore original HEAD
git rebase --edit-todo                 # re-open the todo list mid-rebase
git rebase --quit                      # stop rebasing but DO NOT restore HEAD
```

The interactive rebase todo file:

```
pick   abc123 add login form
reword def456 fix typo in subject
edit   ghi789 stop here to amend
squash jkl012 merge into previous
fixup  mno345 like squash but discard message
fixup -C mno345  like squash but USE this commit's message
drop   pqr678 throw away
exec   make test    run shell command
break              stop and let me poke around
label  branch-A    label current commit
reset  branch-A    reset HEAD to label
merge -C abcd branch-A    re-create a merge commit
```

The `--autosquash` workflow:

```bash
# you find a typo in commit abc1234
git add fix.go
git commit --fixup=abc1234
git rebase -i --autosquash abc1234~
# the editor opens with the fixup pre-positioned and pre-marked. Just save.
```

`--update-refs` is a Git 2.38+ killer feature: when a branch passes through commits being rebased, those branches automatically follow.

## Cherry-pick

```bash
git cherry-pick <sha>                  # apply a single commit on top of current
git cherry-pick <sha1> <sha2> <sha3>   # apply several
git cherry-pick A..B                   # exclusive range (A not included)
git cherry-pick A^..B                  # inclusive of A
git cherry-pick -x <sha>               # append "(cherry picked from commit ...)" to message
git cherry-pick --no-commit <sha>      # apply but don't commit
git cherry-pick -n <sha>               # short for --no-commit
git cherry-pick -e <sha>               # edit message
git cherry-pick -s <sha>               # add Signed-off-by
git cherry-pick -S <sha>               # GPG-sign
git cherry-pick -m 1 <merge-sha>       # cherry-pick a merge commit, mainline = 1st parent
git cherry-pick --strategy=recursive -X theirs <sha>
git cherry-pick --abort
git cherry-pick --continue
git cherry-pick --quit                 # forget cherry-pick state without restoring
git cherry-pick --skip                 # 2.27+
```

To find commits not yet on the target branch, use `git cherry`:

```bash
git cherry main feature                # commits in feature not yet in main
git cherry -v main feature             # with subject
```

## Reflog and Recovery

The reflog is your safety net. Every change to HEAD (or branch tips) is logged for 90 days by default. As long as you act before expiry, almost nothing in Git is unrecoverable.

```bash
git reflog                              # HEAD reflog
git reflog show HEAD                    # explicit
git reflog show feature/auth            # branch reflog
git reflog show --all                   # every ref
git reflog --date=iso                   # human dates
git reflog -n 50                        # last 50 entries
git reflog expire --expire=90.days.ago --all
git reflog expire --expire-unreachable=30.days.ago --all
git reflog delete HEAD@{5}              # remove a single entry
```

Recover after `git reset --hard`:

```bash
git reflog                              # find pre-reset HEAD
git reset --hard HEAD@{1}               # go back one move
git reset --hard ORIG_HEAD              # ORIG_HEAD is set by reset/merge/rebase
```

Recover a deleted branch:

```bash
git reflog                              # find the SHA where the branch was
git checkout -b feature/auth <sha>      # restore it
git switch -c feature/auth <sha>        # modern
```

Recover a stash that was popped:

```bash
git fsck --no-reflogs --lost-found      # list dangling commits
ls .git/lost-found/commit/              # candidates
git show <dangling-sha>                 # inspect
git stash apply <dangling-sha>          # bring it back
```

Reflog refers to commits with `<ref>@{<n>}` syntax:

```bash
git show HEAD@{1}                       # what HEAD was 1 move ago
git show HEAD@{2.hours.ago}
git show HEAD@{yesterday}
git diff main@{1} main                  # what main was last move
```

## Bisect

`git bisect` does a binary search over history to find the commit that introduced a regression.

```bash
git bisect start                        # begin
git bisect bad                          # current is bad
git bisect bad HEAD                     # explicit
git bisect good v1.2.0                  # v1.2.0 was good

# git checks out the midpoint; test
make test
git bisect good                         # this midpoint is good
# or
git bisect bad                          # this midpoint is bad

# repeat until git announces:
# abc1234 is the first bad commit
git bisect reset                        # back to your branch
```

Automated:

```bash
git bisect start HEAD v1.0.0           # bad...good
git bisect run ./regression-test.sh    # script: exit 0=good, 1=bad, 125=skip
```

Skip an untestable commit (build broken etc.):

```bash
git bisect skip
git bisect skip <sha>
```

Replay a previous bisect:

```bash
git bisect log > bisect.txt            # save the session
git bisect replay bisect.txt           # restore it later
```

Visualize the remaining range:

```bash
git bisect visualize
git bisect view --oneline
```

Use terms other than good/bad (e.g. fast/slow, working/broken):

```bash
git bisect start --term-old=fast --term-new=slow
git bisect fast v1.0
git bisect slow HEAD
```

## Worktree

A worktree is a checked-out copy of the repo on a different branch, sharing one `.git` directory. No clone needed.

```bash
git worktree add ../hotfix hotfix/123          # checkout existing branch in new dir
git worktree add -b new-feature ../wt main     # create new branch from main, check out
git worktree add --detach ../inspect <sha>     # detached HEAD worktree
git worktree add --lock ../release release     # mark locked (won't be auto-pruned)

git worktree list                              # show worktrees
git worktree list --porcelain                  # machine-readable
git worktree list --verbose

git worktree lock ../release                   # prevent prune
git worktree unlock ../release
git worktree move ../old ../new                # relocate
git worktree remove ../hotfix                  # delete worktree
git worktree prune                             # clean stale entries (e.g. dirs deleted manually)
git worktree repair                            # 2.30+: fix broken links after moves
```

A given branch can be checked out in only one worktree at a time.

## Stash

```bash
git stash                                   # stash tracked + indexed changes
git stash push                              # explicit
git stash push -m "wip auth refactor"       # named
git stash push -u                           # include untracked files
git stash push --include-untracked          # long form
git stash push -a                           # also include ignored files
git stash push --all
git stash push --keep-index                 # stash but leave staged changes alone
git stash push -p                           # patch mode — choose hunks
git stash push -- src/auth.go               # stash only specific paths
git stash push --staged                     # 2.35+: stash only staged changes

git stash list                              # show stashes
git stash list --stat
git stash show                              # short summary of latest
git stash show -p                           # full diff of latest
git stash show -p stash@{2}                 # full diff of specific
git stash show stash@{1} --name-only

git stash pop                               # apply latest + drop
git stash pop stash@{1}
git stash apply                             # apply latest, keep stash
git stash apply --index                     # restore staged state too
git stash drop                              # delete latest
git stash drop stash@{0}
git stash clear                             # delete all stashes (DANGEROUS)
git stash branch new-branch stash@{1}       # apply stash on new branch
git stash create                            # make stash commit but don't add to stash list
git stash store -m "rescue" <sha>           # add a stash commit to the stash list
```

`stash branch` is the safe play when a stash conflicts with the current branch — applying it on a fresh branch from the stash's parent guarantees no conflict.

## Submodules

Submodules embed another repo at a fixed commit inside yours. They are notoriously sharp.

```bash
git submodule add https://github.com/lib/lib.git vendor/lib
git submodule add -b main https://github.com/lib/lib.git vendor/lib
git submodule init                         # populate .git/config from .gitmodules
git submodule update                       # check out the recorded commit
git submodule update --init                # combined
git submodule update --init --recursive    # plus nested submodules
git submodule update --remote              # update to latest of submodule's tracked branch
git submodule update --remote --merge      # merge upstream into local submodule
git submodule update --remote --rebase
git submodule status
git submodule status --recursive
git submodule summary
git submodule foreach 'git pull origin main'
git submodule foreach --recursive 'git status'
git submodule deinit vendor/lib            # unregister
git submodule deinit -f --all              # nuke registration; do not remove .gitmodules
git rm vendor/lib                          # remove submodule from tracking
```

Clone with submodules:

```bash
git clone --recurse-submodules <url>
git clone --recurse-submodules -j8 <url>   # parallel
```

`.gitmodules` lives in the superproject and stores URL + path. The actual SHA is recorded in the parent commit's tree as a special "gitlink" entry.

Diff in a submodule:

```bash
git diff --submodule=log                   # readable
git diff --submodule=diff                  # full diff into the submodule
git config --global diff.submodule log
git config --global status.submoduleSummary true
```

## Subtrees

Subtrees vendor another repo's contents into yours as regular files. No `.gitmodules`, no init step. Tradeoff: bigger history.

```bash
git subtree add --prefix=vendor/lib https://github.com/lib/lib.git main --squash
git subtree pull --prefix=vendor/lib https://github.com/lib/lib.git main --squash
git subtree push --prefix=vendor/lib https://github.com/lib/lib.git my-changes
git subtree split --prefix=vendor/lib --branch lib-only-history
```

Submodules vs subtrees:

| Aspect              | Submodule                          | Subtree                              |
|---------------------|------------------------------------|--------------------------------------|
| History             | Pointer to external repo           | Vendored, full inline                |
| Clone               | `--recurse-submodules` required    | Just works                           |
| Update              | `submodule update`                 | `subtree pull`                       |
| Push back upstream  | Native                             | `subtree push` (rewrites)            |
| Repo size           | Small                              | Larger                               |
| Beginner-friendly?  | No                                 | Yes                                  |

## Tags

Lightweight tag = a ref that points to a commit. Annotated tag = a real Git object with author, message, optional signature.

```bash
git tag                                     # list
git tag -l 'v1.*'                           # filter
git tag --sort=-v:refname                   # newest first
git tag --contains <sha>                    # tags containing this commit
git tag --points-at HEAD                    # tags exactly here

git tag v1.0.0                              # lightweight at HEAD
git tag v1.0.0 <sha>                        # lightweight at specific commit
git tag -a v1.0.0 -m "Release 1.0"          # annotated
git tag -a v1.0.0 -F message.txt
git tag -s v1.0.0 -m "Release 1.0"          # GPG-signed annotated
git tag -d v1.0.0                           # delete locally
git tag -f v1.0.0                           # overwrite (DANGEROUS for shared tags)

git push origin v1.0.0                      # push one tag
git push origin --tags                      # push all tags
git push --follow-tags                      # push reachable annotated tags only (best default)
git push origin --delete v1.0.0             # delete on remote
git push origin :refs/tags/v1.0.0           # legacy delete

git verify-tag v1.0.0                       # check signature
git show v1.0.0                             # show tag + tagged commit
git describe                                # describe HEAD by closest tag (e.g. v1.2.3-7-gabc123)
git describe --tags                         # include lightweight tags
git describe --abbrev=0                     # just the tag name
git describe --dirty                        # add -dirty if working tree is dirty
git describe --match 'v*' --exclude '*-rc*'
```

Use annotated tags for releases (they have a date and author). Lightweight tags are private bookmarks.

## Remote Operations

```bash
git remote                                  # list remote names
git remote -v                               # with URLs (fetch + push)
git remote show origin                      # detailed info
git remote add origin git@github.com:user/repo.git
git remote add upstream https://github.com/upstream/repo.git
git remote rename origin upstream
git remote remove upstream                  # alias: rm
git remote set-url origin git@github.com:user/repo.git
git remote set-url --push origin git@github.com:user/repo.git
git remote set-url --add --push origin git@gitlab.com:user/repo.git   # push to multiple
git remote set-head origin main             # set the HEAD ref of a remote
git remote set-head origin --auto           # query the remote
git remote prune origin                     # remove stale remote-tracking refs
git remote update                           # fetch from all remotes
```

### fetch

```bash
git fetch                                   # default remote of current branch
git fetch origin                            # all branches from origin
git fetch origin main                       # one branch
git fetch --all                             # every configured remote
git fetch --prune                           # delete remote-tracking refs whose remote is gone
git fetch -p                                # short
git fetch --prune-tags                      # also prune deleted tags
git fetch --tags                            # also fetch all tags
git fetch --no-tags                         # skip tags
git fetch --depth=10                        # limit history depth (shallow)
git fetch --unshallow                       # backfill history (after a shallow clone)
git fetch --filter=blob:none                # partial fetch (lazy blobs)
git fetch --recurse-submodules
git fetch --jobs=8                          # parallel fetch
git fetch origin pull/42/head:pr-42         # GitHub PR ref into local branch
```

### pull

```bash
git pull                                    # fetch + merge (or rebase if pull.rebase=true)
git pull --rebase                           # explicit rebase
git pull --rebase=interactive               # rebase -i
git pull --ff-only                          # refuse non-fast-forward
git pull --no-rebase                        # force merge even if pull.rebase=true
git pull --no-edit                          # skip merge commit editor
git pull --autostash                        # stash before, unstash after
git pull --recurse-submodules
git pull origin main
git pull --tags
```

### push

```bash
git push                                    # to upstream of current branch
git push origin main                        # explicit
git push -u origin feature                  # push and set upstream
git push --set-upstream origin feature      # long form
git push origin HEAD                        # current branch
git push origin HEAD:other-branch           # push current to a different remote name
git push origin :delete-me                  # delete remote branch (legacy)
git push origin --delete delete-me          # delete remote branch (modern)
git push --tags                             # push all tags
git push --follow-tags                      # push commits + reachable annotated tags
git push --force                            # DANGER: overwrite remote
git push --force-with-lease                 # safer force: refuse if remote moved
git push --force-with-lease=branch:expected-sha   # 2.30+: explicit expected
git push --force-if-includes                # 2.30+: refuse if local doesn't include all your remote commits
git push --no-verify                        # skip pre-push hook
git push --dry-run                          # show what would happen
git push -n                                 # short
git push --atomic origin main feature       # all-or-nothing
git push --signed                           # signed push (server must allow)
git push --recurse-submodules=on-demand     # push submodule changes too
```

`--force-with-lease` is what you want 99% of the time. It refuses to push if someone else's commits are on the remote tip you didn't see.

`--force-with-lease --force-if-includes` (Git 2.30+) is the gold standard — it also makes sure your local has fetched everything you've already pushed, defending against double-rewrites.

## Hooks

Hooks live in `.git/hooks/`. Files are NOT versioned by default. To version them, store in `.githooks/` (or any path) and configure:

```bash
git config core.hooksPath .githooks         # repo-local versioned hooks
git config --global core.hooksPath ~/.githooks   # personal hooks for all repos
chmod +x .githooks/pre-commit
```

Common hooks:

```
applypatch-msg          before applypatch-msg accepts message
pre-applypatch          before applypatch commits
post-applypatch         after applypatch commits
pre-commit              before commit message editor runs (linters, tests)
prepare-commit-msg      modify the message template before editor
commit-msg              validate / reformat the message
post-commit             after commit (notifications, etc.)
pre-rebase              before rebase begins
post-checkout           after checkout (LFS, etc.)
post-merge              after merge completes
pre-push                before push (final tests)
pre-receive             on server, before any ref updates
update                  on server, per ref update
post-receive            on server, after all ref updates
post-update             on server, after refs update
push-to-checkout        on server, allow non-bare push
fsmonitor-watchman      file-system monitor integration
p4-pre-submit           perforce bridge
post-rewrite            after amend / rebase rewrites commits
sendemail-validate      validate before send-email
```

A pre-commit hook to run gofmt:

```bash
#!/usr/bin/env bash
# .githooks/pre-commit
files=$(git diff --cached --name-only --diff-filter=ACMR | grep '\.go$')
[ -z "$files" ] && exit 0
unformatted=$(gofmt -l $files)
if [ -n "$unformatted" ]; then
    echo "gofmt issues:"
    echo "$unformatted"
    exit 1
fi
```

Skip hooks (test-only):

```bash
git commit --no-verify
git push --no-verify
```

The `pre-commit` framework (https://pre-commit.com/) manages hooks across repos with a `.pre-commit-config.yaml` and shared community hooks (lint, format, secret-scan, etc.).

## Conflict Resolution

When a merge or rebase hits a conflict, Git pauses with markers in the file:

```
<<<<<<< HEAD
console.log("ours");
=======
console.log("theirs");
>>>>>>> feature/auth
```

With `merge.conflictstyle=zdiff3`, you also see the merge base:

```
<<<<<<< HEAD
console.log("ours");
||||||| merged common ancestors
console.log("original");
=======
console.log("theirs");
>>>>>>> feature/auth
```

```bash
git status                                  # see conflicted files
git diff                                    # see conflicts
git diff --base file.go                     # diff vs merge base
git diff --ours file.go                     # diff vs our side
git diff --theirs file.go                   # diff vs their side
git checkout --ours file.go                 # take our version wholesale
git checkout --theirs file.go               # take their version wholesale
git restore --source=HEAD --staged --worktree file.go   # reset to ours

git mergetool                               # launch configured merge tool (vimdiff, kdiff3, etc.)
git mergetool --tool=vimdiff
git mergetool --tool=meld

# resolve, then
git add file.go
git merge --continue                        # finish merge
git rebase --continue                       # finish rebase
git cherry-pick --continue                  # finish cherry-pick

git merge --abort                           # bail out
git rebase --abort
git cherry-pick --abort
```

### rerere — Reuse Recorded Resolution

Git records how you resolve a conflict and replays the resolution next time the same conflict appears (especially useful during long-running rebases or repeated merges).

```bash
git config --global rerere.enabled true
git config --global rerere.autoupdate true     # auto-stage rerere'd resolutions
git rerere status                               # what rerere thinks is conflicted
git rerere diff                                 # what rerere is recording
git rerere remaining                            # files still conflicting
git rerere clear                                # forget current resolutions
git rerere forget file.go                       # forget specific file
```

## Filter-repo and History Surgery

`git filter-branch` is deprecated. Use `git filter-repo` (https://github.com/newren/git-filter-repo) — fast, safe, replaces BFG.

```bash
brew install git-filter-repo
pip install git-filter-repo
```

```bash
git filter-repo --path src/                          # keep only src/
git filter-repo --path config.yml --invert-paths     # remove a file
git filter-repo --path-glob '*.log' --invert-paths   # remove all .log
git filter-repo --strip-blobs-bigger-than 10M        # purge huge blobs
git filter-repo --replace-text replacements.txt      # redact secrets (file is regex==>replace)
git filter-repo --mailmap mailmap.txt                # rewrite author/committer identities
git filter-repo --subdirectory-filter sub/           # promote sub/ to root
git filter-repo --refs main feature/x                # restrict to specific refs
git filter-repo --commit-callback 'commit.message = commit.message.replace(b"old", b"new")'
```

Always run filter-repo on a fresh `--mirror` clone:

```bash
git clone --mirror git@host:user/repo.git
cd repo.git
git filter-repo ...
git push --force --mirror                            # to a NEW remote, not the original
```

Force-push aftermath: every collaborator has to either re-clone or rebase their work. Communicate.

For redacting `.env` and similar:

```
# replacements.txt
PASSWORD=secret==>PASSWORD=REDACTED
literal:my-actual-token==>REDACTED
regex:apikey-[A-Z0-9]{32}==>REDACTED
```

## Sparse Checkout

Check out only a subset of paths without cloning the rest. Pairs beautifully with partial clone for monorepos.

```bash
git clone --filter=blob:none --no-checkout https://host/big-repo.git
cd big-repo
git sparse-checkout init --cone
git sparse-checkout set apps/web libs/shared
git checkout main
git sparse-checkout list
git sparse-checkout add apps/api
git sparse-checkout reapply
git sparse-checkout disable                  # restore full checkout
```

Cone mode (`--cone`, default since 2.27) is the optimized fast path: paths must be directories and use simple prefixes. Non-cone mode supports gitignore-style patterns:

```bash
git sparse-checkout set --no-cone '/*' '!/heavy/'
```

## Bundle

A bundle is a single-file packfile containing commits + refs. Useful for airgapped transfer.

```bash
git bundle create repo.bundle --all                  # all refs
git bundle create main.bundle main                   # one branch
git bundle create incr.bundle main ^last-shipped     # incremental since tag
git bundle verify repo.bundle                        # check it's well-formed
git bundle list-heads repo.bundle                    # what's inside
git clone repo.bundle dest-dir                       # clone from bundle
git fetch repo.bundle main                           # fetch from bundle
```

Workflow: airgapped sync.

```bash
# on online machine
git bundle create snapshot.bundle main develop --tags
# transfer snapshot.bundle via media

# on airgapped machine
git fetch ../snapshot.bundle main:main develop:develop
```

## Archive

Export a tree without `.git`:

```bash
git archive HEAD                                     # tar to stdout
git archive --format=tar --output=src.tar HEAD
git archive --format=tar.gz --output=src.tgz HEAD
git archive --format=zip --output=src.zip HEAD
git archive --format=tar --prefix=myproj-1.0/ HEAD | gzip > myproj-1.0.tar.gz
git archive --format=tar v1.2.3 | tar -xC /tmp/checkout
git archive --remote=git@github.com:user/repo.git HEAD | tar -x -C dest
```

Honour `export-ignore` in `.gitattributes` to exclude files from archives:

```
# .gitattributes
docs/  export-ignore
test/  export-ignore
.gitignore export-ignore
.github/   export-ignore
```

## Notes

`git notes` attach metadata to a commit without rewriting it.

```bash
git notes add -m "Reviewed by: Bob"
git notes add -m "Note for HEAD~3" HEAD~3
git notes append -m "Additional comment"
git notes show
git notes show <sha>
git notes list
git notes remove <sha>
git notes prune                                       # remove notes for deleted commits
git notes copy <src-sha> <dst-sha>                    # copy notes (e.g. after cherry-pick)

git log --show-notes
git log --show-notes=*                                # show all note refs
git config --global notes.displayRef 'refs/notes/*'   # show all by default
```

Notes are stored in `refs/notes/commits` (or any namespace). They are second-class — most workflows ignore them — and they don't transfer by default:

```bash
git push origin refs/notes/*:refs/notes/*
git fetch origin refs/notes/*:refs/notes/*
git config --global remote.origin.fetch '+refs/notes/*:refs/notes/*'
```

## LFS — Large File Storage

```bash
brew install git-lfs
sudo apt install git-lfs
git lfs install                                       # one-time per machine

git lfs track '*.psd'                                 # adds entry to .gitattributes
git lfs track '*.bin' '*.zip'
git lfs untrack '*.psd'

git add .gitattributes
git add design.psd
git commit -m "Add design"

git lfs ls-files                                       # list LFS-tracked files
git lfs status                                         # current LFS state
git lfs migrate import --include='*.psd'               # convert past commits
git lfs migrate export --include='*.psd' --everything
git lfs prune                                          # delete unreferenced LFS blobs locally
git lfs fetch origin main                              # fetch LFS objects for a branch
git lfs pull
git lfs push origin main
git lfs locks                                          # see locks
git lfs lock design.psd
git lfs unlock design.psd
```

## Performance

```bash
git gc                                                  # garbage collect: pack loose objects, prune
git gc --aggressive                                     # heavier optimization (slow, occasional)
git gc --prune=now                                      # prune all unreachable objects immediately
git gc --auto                                           # only run if needed (called automatically)
git repack -a -d                                        # repack everything, drop redundant
git repack -a -d --depth=250 --window=250               # tighter packs (slower)
git pack-refs --all                                     # pack refs into packed-refs file
git prune                                               # remove unreachable objects
git fsck                                                # check repo integrity
git fsck --full --strict
git count-objects -v                                    # detailed size info

git maintenance start                                    # 2.31+: schedule background maintenance
git maintenance stop
git maintenance run --task=gc
git maintenance run --task=commit-graph
git maintenance run --task=incremental-repack

git commit-graph write --reachable                       # speeds up log/graph queries
git multi-pack-index write                               # multi-pack index speeds large repos
```

Tuning for huge repos:

```bash
git config --global core.preloadIndex true               # parallel index ops
git config --global core.fsmonitor true                  # filesystem monitor (built-in 2.36+)
git config --global core.untrackedCache true
git config --global gc.auto 1024                          # less frequent auto-gc
git config --global gc.autoPackLimit 50
git config --global pack.threads 0                        # auto = NCPU
git config --global pack.windowMemory 256m
git config --global pack.deltaCacheSize 2g
git config --global fetch.parallel 0                      # parallel fetch (auto NCPU)
git config --global submodule.fetchJobs 8
git config --global protocol.version 2                    # faster ref negotiation
git config --global feature.manyFiles true                # 2.34+: enable multiple optimisations
git config --global index.version 4                       # smaller, newer index format
```

## Searching

### grep

```bash
git grep "needle"                                       # search tracked content
git grep -i "needle"                                    # case-insensitive
git grep -n "needle"                                    # show line numbers
git grep -l "needle"                                    # filenames only
git grep -L "needle"                                    # filenames NOT matching
git grep -c "needle"                                    # count per file
git grep -E "regex"                                     # extended regex
git grep -F "literal"                                   # fixed string (no regex)
git grep -P "perl"                                      # PCRE
git grep -w "word"                                      # whole word
git grep -p "needle"                                    # show enclosing function (like diff -p)
git grep -W "needle"                                    # show full enclosing block
git grep --break                                        # blank line between files
git grep --heading                                      # filename above matches
git grep -e foo -e bar                                  # OR
git grep --and -e foo -e bar                            # AND
git grep --not -e foo                                   # NOT
git grep "needle" -- '*.go'                             # restrict by pathspec
git grep "needle" v1.0                                  # search a tag
git grep "needle" $(git rev-list --all)                 # search ALL history (slow)
git grep --all-match -e foo -e bar
```

### log pickaxe vs regex

```bash
git log -S "API_KEY"                # commits where number of "API_KEY" occurrences changed (pickaxe)
git log -G "API_KEY"                # commits whose patch matches regex
git log -S "func Foo" --all -p
git log -G "TODO|FIXME" --all
```

`-S` is faster and finds add/remove. `-G` finds any change touching the regex (including line moves).

## Blame

```bash
git blame file.go                                      # who last touched each line
git blame -L 10,30 file.go                             # only lines 10-30
git blame -L :myFunc:file.go                           # function blame (needs hunk header)
git blame -w file.go                                   # ignore whitespace changes
git blame -M file.go                                   # detect moved lines within file
git blame -C file.go                                   # detect copied lines from other files
git blame -CCC file.go                                 # full copy detection (slow)
git blame --since="2024-01-01" file.go
git blame -e file.go                                   # show emails
git blame -p file.go                                   # porcelain format
git blame --incremental file.go                        # streaming output
git blame -- file.go HEAD~10                           # blame at a previous commit
git blame --ignore-rev <noise-sha> file.go             # ignore a specific commit
git blame --ignore-revs-file .git-blame-ignore-revs file.go
```

`.git-blame-ignore-revs` (lines = SHAs) lets you mass-ignore mass-format commits:

```bash
git config --global blame.ignoreRevsFile .git-blame-ignore-revs
```

GitHub honours this file too.

## Diff Tools

```bash
git difftool                                            # interactive
git difftool --tool=vimdiff
git difftool -t meld
git difftool -t code                                    # VS Code (after configure)
git difftool --no-prompt
git difftool --dir-diff                                 # tree-level rather than file-by-file
git difftool main..feature

git mergetool
git mergetool --tool=vimdiff
git mergetool --tool=meld
git mergetool --tool=kdiff3
```

Configure VS Code:

```bash
git config --global diff.tool vscode
git config --global difftool.vscode.cmd 'code --wait --diff $LOCAL $REMOTE'
git config --global merge.tool vscode
git config --global mergetool.vscode.cmd 'code --wait $MERGED'
```

Configure delta (https://github.com/dandavison/delta):

```bash
git config --global core.pager delta
git config --global interactive.diffFilter 'delta --color-only'
git config --global delta.navigate true
git config --global delta.line-numbers true
git config --global delta.side-by-side true
git config --global merge.conflictstyle zdiff3
```

## Submodule + Worktree + Sparse interactions

The matrix is surprising:

- Submodules + worktree: a submodule is shared across worktrees; updating a submodule in one worktree affects all.
- Worktrees + sparse-checkout: each worktree has its own sparse-checkout config, but they share the object database. Use `extensions.worktreeConfig=true` and `git worktree add --sparse`.
- Sparse + submodule: sparse-checkout interacts oddly with submodules; ensure submodules are inside sparse-included paths or use `git sparse-checkout set --skip-checks`.
- Partial clone + sparse: works great. `--filter=blob:none` + cone-mode sparse is the recommended pattern for monorepo subset checkout.

```bash
git config extensions.worktreeConfig true
git worktree add --no-checkout ../wt main
cd ../wt
git sparse-checkout init --cone
git sparse-checkout set apps/web
git checkout
```

## .gitignore Patterns

Pattern syntax (gitignore(5)):

- Lines starting with `#` are comments. `\#literal-hash` to start with `#`.
- Trailing slash `dir/` matches directories only.
- Leading slash `/foo` is anchored to the root.
- `*` matches anything except `/`.
- `**` matches across `/`.
- `**/foo` matches `foo` anywhere.
- `foo/**` matches everything inside foo.
- `a/**/b` matches a, a/x/b, a/x/y/b.
- `!` negates — re-includes a previously excluded path. Cannot un-ignore a file inside an ignored directory.
- Trailing whitespace ignored unless escaped with `\`.

Example:

```
# Build artifacts
bin/
build/
dist/
*.exe
*.dll
*.so
*.dylib

# IDE
.idea/
.vscode/
!.vscode/settings.json
!.vscode/launch.json

# OS
.DS_Store
Thumbs.db

# Secrets
.env
.env.*
!.env.example

# Logs
*.log
logs/

# Anywhere
**/node_modules/
**/__pycache__/
```

Precedence (later beats earlier within one file; closer to file beats farther):

1. command-line patterns
2. patterns from `.gitignore` (per-directory, deeper wins)
3. patterns from `.git/info/exclude` (repo-local, not versioned)
4. patterns from `core.excludesFile` (`~/.config/git/ignore` by default)

```bash
git config --global core.excludesFile ~/.gitignore_global
```

Check why a file is ignored:

```bash
git check-ignore -v build/output.exe        # which pattern is matching?
# .gitignore:5:bin/    build/output.exe
git check-ignore -v --no-index path
```

## Gitignore Bypass and Untrack

A `.gitignore` only ignores untracked files. Already-tracked files are still tracked.

```bash
# Stop tracking a file but keep it locally
git rm --cached secrets.env
echo secrets.env >> .gitignore
git add .gitignore
git commit -m "Stop tracking secrets.env"

# Stop tracking a directory but keep it locally
git rm -r --cached node_modules/
echo 'node_modules/' >> .gitignore

# Don't show local-only changes (e.g. machine-specific config)
git update-index --assume-unchanged config.local.yml
git update-index --no-assume-unchanged config.local.yml
git ls-files -v | grep '^h '                # 'h' prefix = assume-unchanged

# Sparse / partial intent — pretend the file isn't there
git update-index --skip-worktree config.local.yml
git update-index --no-skip-worktree config.local.yml
git ls-files -v | grep '^S '                # 'S' prefix = skip-worktree
```

`assume-unchanged` is a performance optimization — Git is allowed to assume the file hasn't changed. If anything updates the file (like `git pull`), the assumption is broken.

`skip-worktree` is the right choice for "I want my local edit to persist no matter what." It survives most operations but not all (rebase can clobber it).

## Range and Selection Syntax

Revision parameters (gitrevisions(7)):

```
HEAD                tip of current branch
@                   alias for HEAD
HEAD~               first parent of HEAD
HEAD~3              third ancestor (HEAD~1~1~1)
HEAD^               first parent of HEAD
HEAD^2              second parent (only meaningful for merge commits)
HEAD^^              HEAD's grandparent (HEAD~2)
HEAD@{1}            HEAD as it was 1 reflog entry ago
HEAD@{2.weeks.ago}  HEAD as of two weeks ago
HEAD@{u}            upstream of HEAD's branch
HEAD@{push}         where push goes (may differ from upstream)
HEAD^{tree}         tree object of HEAD
HEAD^{commit}       force-resolve to commit (deref tag)
HEAD^{}             dereference to non-tag
:/fix typo          search commit messages
:1:file             stage 1 of file in conflict (1=base, 2=ours, 3=theirs)
ORIG_HEAD           previous HEAD (set by reset/merge/rebase)
FETCH_HEAD          last fetched ref
MERGE_HEAD          merge target during merge
CHERRY_PICK_HEAD    during cherry-pick
REBASE_HEAD         during rebase

a..b                commits in b not in a
a...b               commits in either, not both
a^@                 all parents of a
a^!                 a but not its parents (single commit)
^a                  exclude commits reachable from a
git log ^a b c      commits in b or c, not a
```

Examples:

```bash
git log HEAD~5..HEAD                # last 5 commits
git log main..HEAD                  # what's on this branch since main
git log HEAD..origin/main           # what's on remote you don't have
git diff main...feature             # changes ON feature since branching from main
git log --left-right A...B          # marks each commit < or > based on side
git show :1:file.txt                # base version during conflict
git show :2:file.txt                # our version
git show :3:file.txt                # their version
git show HEAD@{yesterday}:file.txt
git show v1.0:src/main.go           # that file at v1.0
```

## Configuration Tricks

```bash
# Show staged diff in commit editor
git config --global commit.verbose true

# Edit message template (every commit pre-fills with this)
git config --global commit.template ~/.gitmessage.txt

# Sign all commits with GPG
git config --global commit.gpgsign true
git config --global user.signingkey ABCD1234
git config --global gpg.program gpg

# Sign with SSH (Git 2.34+)
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global gpg.ssh.allowedSignersFile ~/.config/git/allowed_signers

# Show signature status in log
git log --show-signature

# Pretty default branch handling
git config --global init.defaultBranch main
git config --global push.default current

# Faster status
git config --global feature.manyFiles true

# Auto-correct typos with confirmation
git config --global help.autocorrect prompt          # 2.30+
git config --global help.autocorrect 30              # auto-correct after 3.0 sec

# Better diffs
git config --global diff.algorithm histogram
git config --global diff.colorMoved zebra
git config --global diff.colorMovedWS allow-indentation-change

# Remember conflict resolutions
git config --global rerere.enabled true
git config --global rerere.autoupdate true

# Push only the current branch
git config --global push.default current

# Default to rebase on pull
git config --global pull.rebase true
git config --global pull.ff only

# Trailers
git config --global trailer.signedoffby.key "Signed-off-by"

# url rewrites
git config --global url.git@github.com:.insteadOf https://github.com/
git config --global url.https://github.com/.pushInsteadOf git@github.com:
```

## CI / CD Patterns

Shallow clones in CI:

```bash
git clone --depth=1 --no-tags https://host/repo.git
```

GitHub Actions checkout:

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0          # full history (default 1)
    submodules: recursive
    lfs: true
    ref: ${{ github.head_ref || github.ref }}
```

Monorepo subset CI checkout:

```bash
git clone --filter=blob:none --no-checkout https://host/big-repo.git
cd big-repo
git sparse-checkout init --cone
git sparse-checkout set apps/web
git checkout main
```

Bisect a regression in CI:

```bash
git bisect start --no-checkout
git bisect bad HEAD
git bisect good v1.0.0
git bisect run docker run --rm -v "$PWD:/src" myimg ./test.sh
```

GitLab CI shallow + LFS:

```yaml
variables:
  GIT_DEPTH: 1
  GIT_LFS_SKIP_SMUDGE: 0
  GIT_SUBMODULE_STRATEGY: recursive
```

## Common Error Messages and Fixes

### `fatal: not a git repository (or any of the parent directories): .git`

**Cause:** you're outside a Git repo.

**Fix:**
```bash
pwd
git rev-parse --show-toplevel        # finds repo root
git init                              # if you meant to create one
cd $(git rev-parse --show-toplevel)   # back to root
```

### `fatal: refusing to merge unrelated histories`

**Cause:** two branches have no common ancestor (e.g. fresh clone of one repo into another, or after `--orphan`).

**Fix:**
```bash
git pull --allow-unrelated-histories origin main
git merge --allow-unrelated-histories other-branch
```

### `! [rejected] main -> main (non-fast-forward) error: failed to push some refs to 'origin' hint: Updates were rejected because the remote contains work that you do not have locally`

**Cause:** someone else pushed since your last fetch.

**Fix (preferred):**
```bash
git fetch origin
git rebase origin/main
git push
```

**Fix (if you really must rewrite remote):**
```bash
git push --force-with-lease
```

NEVER `git push --force` on a shared branch.

### `fatal: Not possible to fast-forward, aborting.`

**Cause:** `pull.ff=only` is set and the merge isn't fast-forward.

**Fix:**
```bash
git pull --rebase
# or
git config pull.ff false
```

### `error: Your local changes to the following files would be overwritten by checkout:`
### `error: Your local changes to the following files would be overwritten by merge:`

**Cause:** uncommitted changes to a file the operation wants to update.

**Fix:**
```bash
git stash
git checkout other-branch       # or merge / pull
git stash pop
```

Or commit:
```bash
git add -A && git commit -m "WIP"
```

Or discard if you don't need the changes:
```bash
git restore .
git checkout -- .              # legacy
```

### `CONFLICT (content): Merge conflict in <file>`

**Cause:** the obvious one.

**Fix:**
```bash
git status                      # see conflicted files
# edit each, remove markers
git add file
git merge --continue            # or rebase / cherry-pick --continue
```

If unresolvable:
```bash
git merge --abort
git rebase --abort
```

### `fatal: a branch named 'X' already exists`

**Fix:** delete first or rename:
```bash
git branch -D X
git checkout -b X
# or
git branch -M X new-name
```

### `fatal: Authentication failed for 'https://github.com/...'`

**Cause:** GitHub deprecated password auth in 2021.

**Fix (HTTPS):** use a Personal Access Token (PAT). Or switch to SSH:
```bash
git remote set-url origin git@github.com:user/repo.git
ssh -T git@github.com           # test SSH
```

Configure credential cache:
```bash
git config --global credential.helper osxkeychain    # macOS
git config --global credential.helper manager        # cross-platform
```

### `fatal: detected dubious ownership in repository at '/path'`

**Cause:** Git 2.35.2+ refuses repos owned by a different user (CVE-2022-24765).

**Fix:**
```bash
git config --global --add safe.directory /path
git config --global --add safe.directory '*'         # allow all (riskier)
```

### `warning: ignoring broken ref refs/heads/X`

**Cause:** a ref file is corrupted or empty.

**Fix:**
```bash
git fsck --full
cat .git/refs/heads/X            # is it empty / garbage?
git update-ref -d refs/heads/X   # delete bad ref
git reflog | grep X              # find a recent SHA
git update-ref refs/heads/X <sha>
```

### `error: insufficient permission for adding an object to repository database`

**Cause:** mixed UID ownership in `.git/objects/`.

**Fix:**
```bash
sudo chown -R "$(whoami)" .git/
chmod -R u+rwX .git/
# for shared repos
git init --bare --shared=group repo.git
```

### `fatal: bad object <sha>`

**Cause:** missing or corrupted object in pack/loose.

**Fix:**
```bash
git fsck --full
# if you have a healthy clone elsewhere
git fetch other-clone <sha>
# or restore from a backup mirror
```

### `Auto packing the repository for optimum performance.`

Not an error — it's `gc --auto` running. To suppress:

```bash
git config --global gc.auto 0          # disable auto-gc (run manually)
git config --global gc.autoDetach true # run gc in background (default)
```

### `fatal: The current branch X has no upstream branch.`

**Fix:**
```bash
git push -u origin X
git config --global push.autoSetupRemote true     # so this never happens again
```

### `error: pathspec 'X' did not match any file(s) known to git`

**Cause:** typo, wrong cwd, or file isn't tracked.

**Fix:**
```bash
git ls-files | grep X            # is it tracked?
ls X                              # is it on disk?
git status                        # was it added?
```

### `fatal: cannot lock ref 'refs/heads/X': reference already exists`

**Fix:**
```bash
git update-ref -d refs/heads/X
# or
git branch -D X
```

### `error: cannot lock ref 'HEAD': Unable to create '.git/HEAD.lock': File exists.`

**Cause:** another Git process crashed mid-operation, leaving `.lock` files.

**Fix:**
```bash
ps -ef | grep git                 # nothing running?
rm .git/index.lock .git/HEAD.lock .git/refs/heads/*.lock
```

### `fatal: Could not read from remote repository. Please make sure you have the correct access rights and the repository exists.`

**Causes:** SSH key not added, host not in known_hosts, repo URL typo, permissions denied.

**Fix:**
```bash
ssh -vT git@github.com                       # test
ssh-add -l                                    # list loaded keys
ssh-add ~/.ssh/id_ed25519                     # load
git remote -v                                  # check URL
```

### `Updates were rejected because a pushed branch tip is behind its remote counterpart.`

Same as the non-fast-forward error above.

## Recovery Recipes

### Undo last commit, keep changes staged

```bash
git reset --soft HEAD~1
```

### Undo last commit, keep changes unstaged

```bash
git reset --mixed HEAD~1
git reset HEAD~1               # --mixed is default
```

### Undo last commit AND discard changes

```bash
git reset --hard HEAD~1        # DANGEROUS — only if you really mean it
```

### Unstage everything

```bash
git reset                      # alias: git restore --staged .
git restore --staged .
```

### Discard all local changes (working tree)

```bash
git restore .                  # safer
git stash push -m "discardable" && git stash drop   # safest
git checkout -- .              # legacy
git reset --hard HEAD          # also nukes staged
```

### Discard untracked files too

```bash
git clean -n                   # dry run
git clean -fd                  # delete files + dirs
git clean -fdx                 # also delete .gitignored
```

### Undo a public push (already on remote)

NEVER reset on a shared branch. Revert instead:

```bash
git revert <bad-sha>
git push
```

For multiple commits:
```bash
git revert -n <sha1> <sha2> <sha3>
git commit -m "Revert series"
git push
```

### Recover from `git reset --hard`

```bash
git reflog                     # find pre-reset SHA (HEAD@{1} or further)
git reset --hard HEAD@{1}
```

### Wrong branch — moved commits to the right one

```bash
# you committed to main but meant feature
git log -2                      # find the commits
git checkout -b feature         # if feature doesn't exist
git checkout main
git reset --hard origin/main   # roll main back
git checkout feature           # commits are already there if you switched before reset
```

Or with cherry-pick:

```bash
git checkout feature
git cherry-pick <sha>
git checkout main
git reset --hard HEAD~1
```

### Accidentally rebased onto stale upstream

```bash
git reflog                     # find the commit BEFORE the rebase
git reset --hard <pre-rebase-sha>
git fetch origin
git rebase origin/main
```

### Recover a popped stash

```bash
git fsck --no-reflog --lost-found
# look at .git/lost-found/commit/* — they're stash-style merges
git show <sha>
git stash apply <sha>
```

### Restore a deleted file

```bash
git log --oneline -- path/to/file       # find the commit that deleted it
git checkout <sha>~ -- path/to/file     # checkout from before deletion
```

### Restore a deleted branch

```bash
git reflog                              # find the SHA where it was
git switch -c branch-name <sha>
```

### Undo `git add` (unstage)

```bash
git restore --staged file.go
git reset HEAD file.go         # legacy
```

### Restore an `--amend`ed-away commit

```bash
git reflog                     # the previous tip is in reflog
git reset --hard HEAD@{1}
```

### Resolve a "stuck" rebase

```bash
git rebase --abort             # bail out clean
# or
git status                     # see what's blocking
git add .
git rebase --continue
```

## Performance Tips

```bash
# commit-graph (faster log/blame/contains queries)
git config --global core.commitGraph true
git config --global gc.writeCommitGraph true
git commit-graph write --reachable --changed-paths

# Multi-pack-index (multiple packs without slowdown)
git multi-pack-index write
git config --global core.multiPackIndex true

# Background maintenance (Git 2.31+)
git maintenance start
git maintenance stop
git maintenance run --task=commit-graph --task=incremental-repack --task=loose-objects

# Partial clone (huge repos)
git clone --filter=blob:none --no-checkout host:path
git -C cloned sparse-checkout init --cone
git -C cloned sparse-checkout set my/subdir
git -C cloned checkout

# fsmonitor (built-in, much faster status on big trees)
git config --global core.fsmonitor true
git config --global core.untrackedCache true

# Reduce fetch refspecs (don't fetch refs you don't need)
git config remote.origin.fetch '+refs/heads/main:refs/remotes/origin/main'

# Pack tuning
git config --global pack.threads 0          # auto NCPU
git config --global pack.windowMemory 256m
git config --global pack.deltaCacheSize 2g
git config --global core.bigFileThreshold 64m
git config --global core.compression 0      # store loose objects uncompressed (CPU vs disk)
```

## Idioms and Workflows

### Conventional commits

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`. Append `!` after type or `BREAKING CHANGE:` footer for major bumps.

```
feat(auth): add OAuth2 PKCE flow

Adds the PKCE extension to the authorization code flow per RFC 7636.

Closes #123
Signed-off-by: Alice <alice@example.com>
```

### Trunk-based development

- One long-lived branch (`main`).
- Short-lived feature branches merged or rebased back daily.
- Heavy use of feature flags.
- CI gates every push.

### GitHub Flow

1. Branch from `main`.
2. Push, open PR.
3. Review.
4. Merge to `main`.
5. Deploy `main`.

### GitFlow

- `main` always production.
- `develop` integration branch.
- `feature/*` from develop.
- `release/*` from develop, into main + develop.
- `hotfix/*` from main, into main + develop.

Heavyweight; falling out of fashion.

### Rebase vs merge religious war (summarized)

- **Rebase:** linear history, easy `git log` reading, harder for newcomers, never on shared branches.
- **Merge:** preserves true history, accurate "what happened when", noisy graph.
- Common compromise: rebase your feature branch off `main` before merge; merge the feature with `--no-ff` so the branch's existence is recorded.

### DCO Sign-off workflow

```bash
git config --global format.signoff true       # not effective for commits but for format-patch
git commit -s -m "msg"                         # signs off
git rebase --signoff main                      # add sign-off retroactively
git rebase --exec 'git commit --amend --no-edit -s' main
```

The `Signed-off-by:` trailer is a developer's attestation under the Linux Developer Certificate of Origin (https://developercertificate.org/).

## Tips and one-liners

```bash
# Undo last commit, keep changes
git reset --soft HEAD~

# Amend without changing date
git commit --amend --no-edit --date="$(git log -1 --format=%aD)"

# Show files changed in a commit
git show --stat <sha>
git show --name-only <sha>

# List all contributors with email and count
git shortlog -sne

# Find largest objects in history
git rev-list --objects --all \
  | git cat-file --batch-check='%(objecttype) %(objectname) %(objectsize) %(rest)' \
  | awk '$1=="blob"' \
  | sort -k3 -n \
  | tail -20

# GPG-sign all commits forever
git config --global commit.gpgsign true
git config --global user.signingkey ABCD1234

# SSH-sign commits (Git 2.34+, no GPG dance)
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global commit.gpgsign true
git config --global tag.gpgsign true

# What changed in this PR (against base)
git diff $(git merge-base HEAD origin/main)..HEAD

# Number of commits on current branch since main
git rev-list --count main..HEAD

# Most-touched files (churn)
git log --pretty=format: --name-only | sort | uniq -c | sort -rn | head -20

# Word count of all commit messages
git log --pretty=format:%s | wc -w

# Last commit subject line
git log -1 --pretty=%s

# Branches sorted by last commit date
git for-each-ref --sort=-committerdate refs/heads/ \
  --format='%(committerdate:short) %(refname:short)'

# Show commits unique to a branch
git log main..feature --oneline

# Show all branches a commit is on
git branch --contains <sha>

# Find when a function disappeared
git log -L:funcName:file.go

# Search history for a string
git log -S "secret_key" --all --source --remotes

# What did each contributor change in the last week
git shortlog --since="1 week ago" --no-merges -nse

# Quickly stash one file
git stash push -m "wip" -- path/to/file

# Reapply the most recent stash on a new branch
git stash branch new-branch

# Drop ALL stashes (verify first!)
git stash list
git stash clear

# Find commits not yet pushed
git log @{u}..

# Find commits on remote but not local
git log ..@{u}

# Show staged diff
git diff --staged

# Show what `git push` will push
git diff origin/main..HEAD

# Reset only the index (don't move HEAD or worktree)
git reset                              # alias for `reset HEAD`
git reset HEAD .

# Get the SHA of the merge base
git merge-base main feature
git merge-base --is-ancestor <sha> main && echo "yes"

# Pretty list of files modified per commit on this branch
git log --pretty=format:'%h %s' --name-only main..HEAD

# Quickly create an empty commit with no editor
git commit --allow-empty -m "chore: trigger CI"

# Stage everything except one path
git add -A && git restore --staged secret.key

# Forced clean of everything (untracked, ignored)
git clean -fdx

# Show file from a different branch without checking out
git show main:src/foo.go > /tmp/foo.go.main

# Diff a file between two branches
git diff main..feature -- src/foo.go

# Run command on every commit in a range
git rebase -x "make test" main

# Replay one commit by SHA into the current branch
git cherry-pick <sha>

# What was HEAD an hour ago?
git rev-parse 'HEAD@{1.hour.ago}'

# Auto-correct typos
git config --global help.autocorrect prompt

# Display the merge base (common ancestor) of two refs
git merge-base main feature

# Find the smallest set of commits between two refs
git log --reverse --first-parent main..feature

# Apply a patch
git apply patch.diff
git apply --check patch.diff           # dry run
git am < patch.eml                     # apply mailbox patch
git am --abort
git am --continue
git am --skip

# Format a series as patches for email
git format-patch origin/main..HEAD     # one .patch per commit
git format-patch -1 HEAD               # just the tip

# Send patches via email
git send-email *.patch
```

## See Also

- bash
- zsh
- polyglot

## References

- [Pro Git Book (free, 2nd ed)](https://git-scm.com/book/en/v2) — Scott Chacon and Ben Straub, the canonical reference.
- [Git Reference Documentation](https://git-scm.com/docs) — every git command, every flag.
- [git-scm.com](https://git-scm.com/) — official site, downloads, news.
- [man git](https://man7.org/linux/man-pages/man1/git.1.html) — top-level man page.
- [man gitrevisions](https://man7.org/linux/man-pages/man7/gitrevisions.7.html) — revision and range syntax (HEAD~2, @{upstream}, etc.).
- [man gitworkflows](https://man7.org/linux/man-pages/man7/gitworkflows.7.html) — recommended workflows.
- [man gitattributes](https://man7.org/linux/man-pages/man5/gitattributes.5.html) — per-path settings (LFS, diff drivers, merge drivers).
- [man gitignore](https://man7.org/linux/man-pages/man5/gitignore.5.html) — pattern syntax.
- [man gitglossary](https://man7.org/linux/man-pages/man7/gitglossary.7.html) — terminology reference.
- [man githooks](https://man7.org/linux/man-pages/man5/githooks.5.html) — every hook and its environment.
- [Git Internals (Pro Git Ch. 10)](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain) — objects, refs, packfiles, the wire protocol.
- [Oh Shit, Git!?!](https://ohshitgit.com/) — by Katie Sylor-Miller; recipes for common disasters.
- [Git Flight Rules](https://github.com/k88hudson/git-flight-rules) — symptom-driven recovery handbook.
- [Atlassian Git Tutorials](https://www.atlassian.com/git/tutorials) — visual explanations of merge vs rebase.
- [Think Like (a) Git](http://think-like-a-git.net/) — mental model for the reachability graph.
- [git-tips](https://github.com/git-tips/tips) — practical one-liners.
- [git-filter-repo](https://github.com/newren/git-filter-repo) — modern history-rewrite tool.
- [pre-commit framework](https://pre-commit.com/) — community hook manager.
- [delta](https://github.com/dandavison/delta) — better pager and diff renderer.
- [tig](https://jonas.github.io/tig/) — ncurses Git interface.
- [lazygit](https://github.com/jesseduffield/lazygit) — TUI Git frontend.
- [Conventional Commits](https://www.conventionalcommits.org/) — message format spec.
- [Developer Certificate of Origin](https://developercertificate.org/) — what `--signoff` attests.
- [Git Magic](http://www-cs-students.stanford.edu/~blynn/gitmagic/) — Ben Lynn's free intro book.
- [Git from the bottom up](https://jwiegley.github.io/git-from-the-bottom-up/) — John Wiegley, builds intuition from objects up.
- [GitHub: First-time contributor's guide](https://docs.github.com/en/get-started/quickstart) — for tooling around git but worth knowing.
- [Linux kernel Git workflow](https://www.kernel.org/doc/html/latest/process/submitting-patches.html) — the most demanding workflow in the wild.