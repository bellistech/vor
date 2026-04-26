# Git Errors

Verbatim error messages, root causes, and exact recovery commands for every common (and uncommon) Git failure mode.

## Setup

Git's mental model:

- **Working tree** — the files you see on disk; your editable copy.
- **Index / staging area** — the snapshot you're building for the next commit (`.git/index` is a binary file).
- **HEAD** — a symbolic ref pointing to the current branch (e.g. `refs/heads/main`); when detached, points directly at a commit.
- **Branches** — files under `.git/refs/heads/<name>` (or in `.git/packed-refs`) holding a single commit SHA.
- **Refs** — named pointers under `.git/refs/`; tags live in `refs/tags/`, remote-tracking branches in `refs/remotes/`.
- **Objects** — content-addressed by SHA-1 (or SHA-256 with `--object-format=sha256`):
  - **blob** — file content (no name, no metadata).
  - **tree** — directory listing (names + modes + blob/tree SHAs).
  - **commit** — tree SHA + parent SHAs + author + committer + message.
  - **tag** — annotated tag wrapping a commit SHA + message + signature.
- **Packfiles** — `.git/objects/pack/*.pack` + `*.idx`; deltified storage of many objects.
- **Reflog** — `.git/logs/HEAD` and `.git/logs/refs/heads/*`; per-ref local history of every move. **Your safety net** — anything HEAD ever pointed at survives until `gc.reflogExpire` (default 90 days).

`.git/` directory layout:

```text
.git/
  HEAD                    # ref: refs/heads/main
  config                  # local config (remotes, branches, aliases)
  index                   # binary staging area
  description             # gitweb only; ignore
  hooks/                  # client-side scripts (pre-commit, post-merge, ...)
  info/exclude            # repo-local gitignore (not committed)
  logs/                   # reflog data
    HEAD
    refs/heads/<branch>
  objects/                # the object database
    <sha[0:2]>/<sha[2:]>  # loose objects
    pack/*.pack           # packed objects
    pack/*.idx            # pack indexes
    info/packs            # list of packs
  packed-refs             # snapshotted refs (branches, tags, remotes)
  refs/
    heads/                # local branches
    tags/                 # tags
    remotes/<remote>/     # remote-tracking branches
  ORIG_HEAD               # previous HEAD before reset/merge/rebase
  FETCH_HEAD              # last-fetched commits
  MERGE_HEAD              # commit being merged in (during merge)
  CHERRY_PICK_HEAD        # commit being cherry-picked
  REVERT_HEAD             # commit being reverted
  rebase-merge/           # state for an interactive rebase
  rebase-apply/           # state for non-interactive rebase
```

Useful inspection commands:

```bash
git rev-parse HEAD                  # SHA of current commit
git rev-parse --abbrev-ref HEAD     # current branch name
git rev-parse --git-dir              # path to .git
git rev-parse --show-toplevel        # path to working tree root
git rev-parse --is-inside-work-tree  # true/false
git cat-file -t <sha>                # object type (blob/tree/commit/tag)
git cat-file -p <sha>                # pretty-print object
git ls-tree HEAD                     # tree of current commit
git ls-files --stage                 # index contents
git reflog                           # HEAD's history
git reflog show <branch>             # branch's history
```

Reflog is the single most important Git survival tool. Almost every "I lost a commit" recovery starts with `git reflog`.

## Push Errors

### `! [rejected] <branch> -> <branch> (non-fast-forward)`

**Cause:** the remote branch has commits you don't have. A fast-forward push would lose them.

```text
To github.com:org/repo.git
 ! [rejected]        main -> main (non-fast-forward)
error: failed to push some refs to 'github.com:org/repo.git'
hint: Updates were rejected because the tip of your current branch is behind
hint: its remote counterpart. Integrate the remote changes (e.g.
hint: 'git pull ...') before pushing again.
```

**Fix:**

```bash
git fetch origin
git log HEAD..origin/main --oneline   # what you don't have
git pull --rebase origin main         # replay your work on top
git push origin main
# Or merge:
git pull origin main
git push origin main
```

### `! [rejected] <branch> -> <branch> (fetch first)`

**Cause:** the remote ref exists but your local doesn't know about it. Run fetch.

```bash
git fetch origin
git pull --rebase origin main
git push origin main
```

### `Updates were rejected because the remote contains work that you do not have locally`

**Cause:** same as non-fast-forward — remote has commits you lack.

**Fix:** identical to above. Never use `--force` on a shared branch without team coordination; use `--force-with-lease`.

### `Updates were rejected because the tip of your current branch is behind`

**Cause:** local branch is strictly behind the remote.

```bash
git pull --ff-only origin main        # cleanest fast-forward
git push origin main
```

### `error: failed to push some refs to 'X'`

Generic wrapper around any of the above. Read the lines above it for the actual reason.

```bash
git push origin main 2>&1 | head -20  # full output
```

### `fatal: refusing to update branch 'X'`

**Cause:** server-side hook or `receive.denyCurrentBranch` rejected the push (default for non-bare repos: pushing to a non-bare repo's checked-out branch is dangerous).

**Fix on receiving repo:**

```bash
# On the server:
git config receive.denyCurrentBranch updateInstead   # working tree updated
# or:
git config receive.denyCurrentBranch warn            # allow with warning
# Best practice: push to a bare repo and have the server-side checkout it.
```

### `remote: error: GH006: Protected branch update failed for refs/heads/X`

**Cause:** GitHub branch protection rule blocks direct push (often blocks force-push, requires PR, requires status checks).

```text
remote: error: GH006: Protected branch update failed for refs/heads/main.
remote: error: At least 1 approving review is required by reviewers with write access.
To github.com:org/repo.git
 ! [remote rejected] main -> main (protected branch hook declined)
error: failed to push some refs to 'github.com:org/repo.git'
```

**Fix:** open a PR. If you own the repo, edit Settings → Branches → branch protection.

### `fatal: Authentication failed for 'X'`

**Cause:** wrong PAT/password (HTTPS), expired credential, or 2FA without token.

```text
fatal: Authentication failed for 'https://github.com/org/repo.git/'
```

**Fix:**

```bash
# Update credential cache:
git credential-cache exit             # clear in-memory cache
git credential-osxkeychain erase       # macOS Keychain helper
host=github.com
protocol=https
# (then Ctrl-D)
# Switch to SSH:
git remote set-url origin git@github.com:org/repo.git
# Or use a fresh PAT (GitHub: Settings → Developer settings → Tokens):
git remote set-url origin https://USERNAME:TOKEN@github.com/org/repo.git
```

### `fatal: unable to access 'X': SSL certificate problem`

**Cause:** corporate MITM proxy with custom CA, expired cert on self-hosted, system CA bundle stale.

```text
fatal: unable to access 'https://gitlab.example.com/group/repo.git/':
SSL certificate problem: unable to get local issuer certificate
```

**Fix (proper):**

```bash
git config --global http."https://gitlab.example.com/".sslCAInfo /path/to/corp-ca.pem
```

**Fix (last resort, INSECURE):**

```bash
git -c http.sslVerify=false clone https://gitlab.example.com/group/repo.git
# Persistent (don't):
git config --global http.sslVerify false
```

### `fatal: not currently on a branch` — detached HEAD push attempt

**Cause:** you're at a detached HEAD (e.g. after `git checkout <sha>`) and ran `git push`.

```bash
git switch -c new-branch              # name what HEAD points at
git push -u origin new-branch
```

### `fatal: The current branch X has no upstream branch`

**Cause:** local branch isn't tracking a remote.

```text
fatal: The current branch feature/x has no upstream branch.
To push the current branch and set the remote as upstream, use

    git push --set-upstream origin feature/x
```

```bash
git push -u origin HEAD               # push & set upstream in one shot
# Or set tracking without pushing:
git branch --set-upstream-to=origin/feature/x feature/x
# Configure default behavior:
git config --global push.default current   # always push to a same-named upstream
git config --global push.autoSetupRemote true  # auto -u on first push
```

## Pull / Fetch / Merge Errors

### `fatal: refusing to merge unrelated histories`

**Cause:** the two branches have no common ancestor (typical when you `git init` locally and then add a remote that already has commits).

```bash
git pull origin main --allow-unrelated-histories
# Then resolve conflicts and commit.
```

### `fatal: Not possible to fast-forward, aborting`

**Cause:** `pull.ff=only` (or `--ff-only`) and the branches have diverged.

```bash
git config pull.ff only               # current setting
git pull --rebase origin main         # replay local commits
# Or accept a merge:
git pull --no-ff origin main
```

### `Your branch is ahead of 'origin/X' by N commits`

Not an error — your local has commits the remote lacks.

```bash
git push origin HEAD
```

### `Your branch is behind 'origin/X' by N commits`

Not an error — remote has commits you lack.

```bash
git pull --ff-only origin main
```

### `Your branch and 'origin/X' have diverged`

```text
Your branch and 'origin/main' have diverged,
and have 2 and 3 different commits each, respectively.
  (use "git pull" to merge the remote branch into yours)
```

```bash
git log --oneline --graph --all --decorate
git pull --rebase origin main         # cleanest for feature branches
# or
git pull --no-rebase origin main      # merge commit
```

### `Already up to date`

Not an error — nothing to fetch.

### `Automatic merge failed; fix conflicts and then commit the result`

```bash
git status                            # see "Unmerged paths"
git diff                              # conflict markers
# Edit files: keep <<<<<<< / ======= / >>>>>>> blocks resolved
git add <resolved-file>
git commit                            # uses prepared merge message
# Abort and start over:
git merge --abort
```

### `CONFLICT (content): Merge conflict in X`

Standard textual conflict.

```text
<<<<<<< HEAD
your changes
=======
their changes
>>>>>>> branch-name
```

```bash
git mergetool                         # GUI tool (vimdiff/meld/kdiff3/vscode)
git checkout --ours <file>            # accept your side
git checkout --theirs <file>          # accept their side
git add <file>
git commit
```

### `CONFLICT (add/add): Merge conflict in X`

**Cause:** both branches added the same path independently.

```bash
git diff <file>
# Pick one or merge by hand, then:
git add <file>
git commit
```

### `CONFLICT (modify/delete): X deleted in Y and modified in HEAD`

**Cause:** one side modified a file, the other deleted it.

```bash
git status                            # shows "deleted by them" or "deleted by us"
git rm <file>                         # accept the deletion
# or
git add <file>                        # keep the modified version
git commit
```

### `CONFLICT (rename/rename): Rename X->Y in HEAD; rename X->Z in branch`

**Cause:** both branches renamed the same file to different targets.

```bash
git status
# Move file to chosen final name, delete other:
git mv Y final-name.go                 # or your choice
git rm Z
git add final-name.go
git commit
```

### `fatal: Need to specify how to reconcile divergent branches`

**Cause:** Git 2.27+ requires explicit `pull` strategy.

```text
hint: Pulling without specifying how to reconcile divergent branches is
hint: discouraged. You can squelch this message by running one of the
hint: following commands sometime before your next pull:
hint:
hint:   git config pull.rebase false  # merge (the default strategy)
hint:   git config pull.rebase true   # rebase
hint:   git config pull.ff only       # fast-forward only
```

```bash
git config --global pull.rebase true       # my default
git config --global pull.ff only           # safest for shared branches
```

## Branch Errors

### `fatal: A branch named 'X' already exists`

```bash
git branch -D X                       # delete (force, even if unmerged)
git checkout -B X                     # checkout, force-reset to current HEAD
# Or rename:
git branch -m X X-old
git checkout -b X
```

### `error: branch 'X' not found`

```bash
git branch -a | grep X                # confirm spelling, check remotes
git fetch --all
git checkout -b X origin/X            # create from remote
```

### `error: Cannot delete branch 'X' checked out at '/path'`

**Cause:** you can't delete the branch you're on, or the branch is checked out in a worktree.

```bash
git switch main
git worktree list                     # find checked-out worktrees
git worktree remove /path             # if needed
git branch -d X                       # safe delete (must be merged)
git branch -D X                       # force delete
```

### `error: The branch 'X' is not fully merged`

```text
error: The branch 'feature/x' is not fully merged.
If you are sure you want to delete it, run 'git branch -D feature/x'.
```

```bash
git log feature/x --not main --oneline   # see unmerged commits
git branch -D feature/x                  # force delete (commits remain in reflog 90d)
```

### `fatal: not a valid object name: 'X'`

**Cause:** ref/SHA doesn't exist (typo, or fetched but pruned).

```bash
git rev-parse X                        # confirm
git fetch --all
git reflog                             # was it ever real?
```

### `fatal: ambiguous argument 'X': both revision and filename`

**Cause:** `X` is both a branch name and a path on disk.

```bash
git log -- X                          # disambiguate as path
git log X --                          # disambiguate as revision
git log refs/heads/X                  # full ref path
```

## Repository / Workdir Errors

### `fatal: Not a git repository (or any of the parent directories): .git`

```bash
pwd
git rev-parse --show-toplevel          # find nearest repo
ls -la .git                            # inspect
git init                               # create one if appropriate
# Inside a worktree linked dir:
cat .git                               # may say: gitdir: /path/to/.git/worktrees/X
```

### `fatal: bad config file line N in /path/.git/config`

```bash
git config --edit                      # fix it (uses $GIT_EDITOR)
# Or by hand:
$EDITOR .git/config
git config --list --local              # validate
```

### `fatal: bad config file line N in $HOME/.gitconfig`

```bash
git config --global --edit
# Backup then reset if catastrophic:
mv ~/.gitconfig ~/.gitconfig.bak
git config --global user.name "Steve Bellis"
git config --global user.email stevie@bellis.tech
```

### `fatal: index file smaller than expected`

**Cause:** `.git/index` truncated/corrupted (e.g. process killed mid-write, disk full).

```bash
rm .git/index
git reset                              # rebuild index from HEAD
git status                             # working tree changes preserved
```

### `error: object file X is empty` — corrupted object

```bash
find .git/objects -type f -size 0      # find empty objects
git fsck --full                        # detailed integrity check
# Recover from a backup or fresh clone:
mv .git .git-broken
git clone <remote> /tmp/fresh
mv /tmp/fresh/.git .
git fsck                               # confirm clean
```

### `fatal: bad object X` — corruption or pruned reference

```bash
git fsck --full --strict
git cat-file -t <sha>                  # what type was it?
git fetch --all                        # may re-fetch from remote
# Surgical: copy missing objects from a clone:
cp clone/.git/objects/<aa>/<rest> .git/objects/<aa>/<rest>
```

### `warning: ignoring broken ref refs/X/Y`

**Cause:** ref file contains an invalid SHA or no longer points at a real object.

```bash
cat .git/refs/X/Y
# Fix:
git update-ref -d refs/X/Y             # delete ref
# Or repoint:
git update-ref refs/X/Y <good-sha>
```

### `fatal: detected dubious ownership in repository at '/path'`

```text
fatal: detected dubious ownership in repository at '/path'
To add an exception for this directory, call:

        git config --global --add safe.directory /path
```

**Cause:** Git 2.35.2+ refuses to operate on repos owned by another user (CVE-2022-24765 mitigation).

```bash
git config --global --add safe.directory /path
git config --global --add safe.directory '*'    # all dirs (less safe)
# Or fix ownership:
sudo chown -R $(id -u):$(id -g) /path
```

### `fatal: cannot use bare repository '/path' (safe.bareRepository is 'explicit')`

**Cause:** Git 2.38+ refuses bare repos discovered via parent traversal unless explicit (CVE-2022-39253).

```bash
git --git-dir=/path/to/bare.git log         # explicit
GIT_DIR=/path/to/bare.git git log           # explicit
git config --global safe.bareRepository all # globally allow (unsafe in untrusted dirs)
```

## Object Database Errors

### `error: insufficient permission for adding an object to repository database .git/objects`

**Cause:** mixed file ownership (often a previous `sudo git ...`).

```bash
ls -la .git/objects                    # spot the bad owner
sudo chown -R $(id -u):$(id -g) .git
chmod -R u+rwX .git
git fsck                               # confirm
```

### `fatal: pack-objects died of signal 9` — OOM

**Cause:** Linux OOM killer or container memory limit hit during repack.

```bash
git config --global pack.windowMemory 256m
git config --global pack.packSizeLimit 1g
git config --global pack.threads 1
git gc --auto                          # incremental
# Reduce blob limit:
git -c pack.window=10 -c pack.depth=10 gc
```

### `fatal: missing blob X` — corruption or shallow clone limitation

```bash
git fsck --full --no-reflogs --connectivity-only
# If shallow:
git fetch --unshallow
git fetch --depth=2147483647           # effectively full
# Otherwise, replace from a known-good clone (see above).
```

### `git fsck` output anatomy

```bash
git fsck --full --strict --unreachable --dangling
```

```text
Checking object directories: 100% (256/256), done.
dangling commit  abc123...           # not reachable from any ref, in reflog
dangling blob    def456...           # orphan blob (e.g. removed file)
missing blob     fff999...           # broken — fix immediately
broken link from   commit X
              to   blob fff999       # ditto
```

- **dangling** — survives reflog/gc grace period; recoverable.
- **missing** — DB is broken; restore from backup or clone.
- **broken link** — pointer exists but target doesn't.

## Remote Errors

### `fatal: remote origin already exists`

```bash
git remote -v
git remote set-url origin git@github.com:org/repo.git   # update
git remote remove origin                                  # nuke
git remote rename origin upstream                         # rename
```

### `fatal: No such remote 'X'`

```bash
git remote -v
git remote add X git@github.com:org/repo.git
```

### `fatal: 'X' does not appear to be a git repository`

**Cause:** wrong URL, server unreachable, repo private without auth, ref path typo.

```bash
git ls-remote https://github.com/org/repo.git    # smoke test
ssh -T git@github.com                             # SSH auth probe
ping github.com
nslookup github.com
```

### `fatal: Could not read from remote repository`

```text
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.
```

```bash
ssh -vT git@github.com 2>&1 | head -40
ssh-add -l                            # is the key loaded?
ssh-add ~/.ssh/id_ed25519
git remote -v                         # double-check URL
```

### `fatal: refspec X is not in the form 'refs/...:refs/...'`

```bash
git push origin refs/heads/main:refs/heads/main      # explicit
git push origin main:main                            # short form (works)
git push origin main:refs/heads/staging              # rename on push
```

### HTTPS vs SSH URL forms

```text
https://github.com/org/repo.git              # HTTPS — uses PAT/credential helper
git@github.com:org/repo.git                  # SSH — uses ~/.ssh keys
ssh://git@github.com/org/repo.git            # SSH (alternate)
git://github.com/org/repo.git                # git protocol (no auth, read-only — deprecated)
```

```bash
git remote set-url origin git@github.com:org/repo.git    # switch to SSH
git remote set-url origin https://github.com/org/repo.git # switch to HTTPS
# Force HTTPS for all GitHub:
git config --global url."https://github.com/".insteadOf git@github.com:
```

## Add / Commit Errors

### `Changes not staged for commit`

Not an error — just a status note.

```bash
git add <file>                        # stage
git add -p                            # interactive hunk-by-hunk
git add -u                            # stage tracked modifications, not new
git commit -a                         # auto-stage tracked + commit
```

### `Untracked files`

Not an error.

```bash
git add <file>
git status -uno                       # hide untracked
git config --global status.showUntrackedFiles no    # always hide (carefully)
```

### `nothing to commit, working tree clean`

Not an error.

```bash
git log -1                            # confirm what HEAD is
git commit --allow-empty -m "trigger CI"
```

### `fatal: pathspec 'X' did not match any files`

```text
fatal: pathspec 'README.md' did not match any files
```

```bash
ls                                    # spelling
git ls-files | grep -i X              # look for matches
git status                             # see what's there
# Often: you're in the wrong directory.
```

### `fatal: bad pathspec ':' in /path`

**Cause:** colon prefix in pathspec is a magic syntax (e.g. `:!`, `:(exclude)`, `:/`).

```bash
git add ':!secret/*' .                # exclude pattern
git add ':(exclude)secret' .          # equivalent
git add ':/'                          # everything from repo root
git add -- 'literal:colon'            # `--` ends options; literal colon ok
```

### `warning: LF will be replaced by CRLF` (Windows line endings)

```text
warning: in the working copy of 'src/main.go', LF will be replaced by CRLF
the next time Git touches it
```

```bash
# Per-repo .gitattributes:
echo '* text=auto eol=lf' >> .gitattributes
git add --renormalize .
git commit -m "Normalize line endings"
# Or:
git config core.autocrlf input         # convert to LF on commit, leave on checkout
git config core.autocrlf false         # leave alone (cross-platform projects)
```

### `fatal: empty ident name (for <X>) not allowed`

```text
fatal: empty ident name (for <stevie@bellis.tech>) not allowed
```

```bash
git config --global user.name "Steve Bellis"
git config --global user.email stevie@bellis.tech
```

### `Author identity unknown`

```text
*** Please tell me who you are.

Run

  git config --global user.email "you@example.com"
  git config --global user.name "Your Name"

to set your account's default identity.
Omit --global to set the identity only in this repository.

fatal: unable to auto-detect email address (got 'user@host.(none)')
```

```bash
git config --global user.name "Steve Bellis"
git config --global user.email stevie@bellis.tech
# Per-repo override (e.g. a different identity for work):
git config user.email work@company.com
# Conditional config by path (Git 2.13+):
cat >> ~/.gitconfig <<'EOF'
[includeIf "gitdir:~/work/"]
    path = ~/.gitconfig-work
EOF
```

## Stash Errors

### `fatal: --patch is incompatible with --all/--include-untracked`

```bash
git stash push -p                     # interactive hunks of tracked files
git stash push -u                     # include untracked
# Can't combine; pick one approach.
```

### `Cannot save the current worktree state` (with various reasons)

Common reasons: nothing to stash, files unmerged, paths conflict.

```bash
git status                            # see why
git stash --include-untracked         # often fixes "nothing to stash"
git merge --abort                     # if mid-merge
```

### `fatal: stash@{N} is not a stash reference`

```bash
git stash list                         # confirm available stashes
git show stash@{0}                     # default stash
git stash apply stash@{2}              # by index
```

### `error: Your local changes to the following files would be overwritten by merge`

```text
error: Your local changes to the following files would be overwritten by merge:
        src/main.go
Please commit your changes or stash them before you merge.
Aborting
```

```bash
git stash push -m "wip"
git pull
git stash pop                          # may produce conflicts; resolve
```

## Checkout / Switch / Restore Errors

### `error: Your local changes to the following files would be overwritten by checkout`

```bash
git stash push -m "wip"
git switch other-branch
git stash pop
# Or:
git checkout -m other-branch          # carry merging
git checkout --force other-branch     # DESTROYS uncommitted changes
```

### `error: pathspec 'X' did not match any file(s) known to git`

```bash
git ls-files | grep X
git fetch --all
git switch -c X origin/X              # likely a remote branch
```

### `fatal: invalid reference: X`

```bash
git branch -a | grep X
git tag | grep X
git rev-parse X
git fetch --all
```

### `Switched to a new branch 'X'`

Not an error — confirmation.

### `Already on 'X'`

Not an error — already there.

### `Note: switching to 'X'. You are in 'detached HEAD' state`

```text
Note: switching to 'abc123'.

You are in 'detached HEAD' state. You can look around, make experimental
changes and commit them, and you can discard any commits you make in this
state without impacting any branches by switching back to a branch.

If you want to create a new branch to retain commits you create, you may
do so (now or later) by using -c with the switch command. Example:

  git switch -c <new-branch-name>
```

```bash
git switch -c experiment              # name your work
git switch -                          # back to previous branch
```

## Reset / Revert / Cherry-pick Errors

### `error: cherry-pick failed: X`

```bash
git status                            # see conflicts
# Resolve files, then:
git cherry-pick --continue
git cherry-pick --abort               # bail out
git cherry-pick --skip                # skip this commit
```

### `CONFLICT (content): Merge conflict in X` (during cherry-pick)

Same resolution as merge: edit, `git add`, `git cherry-pick --continue`.

### `fatal: bad revision 'X'`

```bash
git rev-parse X                       # validate ref/SHA
git log --oneline | head              # eyeball recent SHAs
git reflog                            # in case it's expired locally
```

### `fatal: Could not parse object 'X'`

**Cause:** SHA exists in name only — object missing or path doesn't resolve.

```bash
git fsck --full
git cat-file -t X
```

### `fatal: --reverse is incompatible with --first-parent`

```bash
git log --first-parent main           # merges only
git log --reverse main                 # oldest first
# Pick one.
```

### `Could not apply X...` (cherry-pick or rebase)

```text
error: could not apply abc123... Add login flow
hint: Resolve all conflicts manually, mark them as resolved with
hint: "git add/rm <conflicted_files>", then run "git rebase --continue".
```

```bash
git status
# Resolve, then:
git add <files>
git rebase --continue       # OR  git cherry-pick --continue
git rebase --skip
git rebase --abort
```

## Rebase Errors

### `First, rewinding head to replay your work on top of it...`

Informational from older Git; modern Git is quieter.

### `error: could not apply X`

See "Could not apply X..." above.

### `When you have resolved this problem, run "git rebase --continue"`

```bash
git status
# Edit files, then:
git add <resolved>
git rebase --continue
git rebase --skip                     # skip this commit
git rebase --abort                    # back to where you started
```

### `fatal: cannot rebase: You have unstaged changes`

```bash
git stash push -u -m "rebase-wip"
git rebase main
git stash pop
# Or:
git rebase --autostash main           # auto stash/pop
git config --global rebase.autoStash true   # always
```

### `fatal: It seems that there is already a rebase-merge directory`

```text
fatal: It seems that there is already a rebase-merge directory, and
I wonder if you are in the middle of another rebase.  If that is the
case, please try
        git rebase (--continue | --abort | --skip)
If that is not the case, please
        rm -fr ".git/rebase-merge"
and run me again.  I am stopping in case you still have something
valuable there.
```

```bash
ls .git/rebase-merge/                 # check what's in flight
git rebase --abort                    # clean exit
# If certain it's stale:
rm -rf .git/rebase-merge .git/rebase-apply
```

### `Rebase in progress; onto X`

`git status` shows this when you're mid-rebase.

```bash
git status
git rebase --continue / --skip / --abort
```

### `Cannot rebase onto multiple branches`

**Cause:** ambiguous upstream (multiple branches contain HEAD).

```bash
git rebase origin/main                # explicit base
```

### `Could not execute editor`

```bash
echo $GIT_EDITOR $VISUAL $EDITOR
git config --global core.editor "vim"
git config --global core.editor "code --wait"
git config --global core.editor "nvim"
# Verify:
git config --get core.editor
```

## Submodule Errors

### `fatal: No url found for submodule path 'X' in .gitmodules`

```bash
cat .gitmodules                       # confirm entry exists
git submodule sync                    # copy URLs into .git/config
git submodule update --init --recursive
```

### `Submodule path 'X': checked out 'Y'`

Not an error — confirmation.

### `fatal: Submodule 'X' could not be updated`

**Cause:** submodule's pinned SHA isn't on the remote (force-push or removed branch).

```bash
git -C X fetch --all
git -C X log --all --oneline | head
git submodule update --init --remote X    # pull latest of tracked branch
```

### `warning: unable to rmdir 'X': Directory not empty`

**Cause:** removing a submodule but its files remain.

```bash
git submodule deinit -f X
git rm -f X
rm -rf .git/modules/X
git config -f .gitmodules --remove-section submodule.X
git commit -m "Remove submodule X"
```

## Worktree Errors

### `fatal: 'X' is already checked out at /path`

```bash
git worktree list
git worktree remove /path             # if no longer needed
git worktree prune                    # clean stale entries
git switch X                          # if you can use the existing worktree
```

### `fatal: 'X' already exists`

```bash
git worktree add /tmp/wt-feature -b feature/x main
# Path collision:
rm -rf /tmp/wt-feature
git worktree prune
git worktree add /tmp/wt-feature -b feature/x main
```

### `Removing worktrees/X: gitdir file points to non-existent location`

```text
Removing worktrees/X: gitdir file points to non-existent location
```

Run prune to clean up:

```bash
git worktree prune -v
```

### `fatal: '/path' already exists`

```bash
mv /path /path.bak
git worktree add /path feature/x
```

## Tag Errors

### `fatal: tag 'X' already exists`

```bash
git tag -d X                          # delete local
git push origin :refs/tags/X          # delete remote
git push --delete origin X            # equivalent
git tag -fa X                         # force overwrite
```

### `fatal: tag 'X' not found`

```bash
git fetch --tags
git tag -l 'v1.*'                     # glob
```

### `fatal: Cannot read object X (object missing)`

**Cause:** tag references a commit that's not in the object DB.

```bash
git fetch --tags --all
git fsck --full
```

### Annotated vs lightweight vs signed tags

```bash
git tag v1.0.0                        # lightweight (just a ref)
git tag -a v1.0.0 -m "Release 1.0.0"  # annotated (its own object: tagger, message)
git tag -s v1.0.0 -m "Release 1.0.0"  # signed (PGP)
git verify-tag v1.0.0                  # check signature
git tag -v v1.0.0                      # equivalent
git push origin v1.0.0                 # push one
git push origin --tags                 # push all
git push origin --follow-tags          # push reachable annotated tags
```

## Hook Errors

### `error: hook 'X' is not executable`

```bash
chmod +x .git/hooks/pre-commit
chmod +x .git/hooks/*
```

### `error: command for 'X' is not allowed in worktree`

**Cause:** `core.hooksPath` points outside the worktree and hooks contain commands `safe.directory` blocks.

```bash
git config --get core.hooksPath
git config --global --add safe.directory '*'
# Or fix the hook path:
git config core.hooksPath .githooks
```

### `pre-commit hook failed (add --no-verify to bypass)`

```bash
# Read the hook output above this line — that's your real error.
# Examples:
.git/hooks/pre-commit                  # run manually to see output
# Bypass (use sparingly):
git commit --no-verify -m "wip"
git commit -n -m "wip"                 # short
# Don't ship --no-verify in a script unless you really mean it.
```

### `commit-msg hook failed`

```text
commit-msg hook failed (add --no-verify to bypass)
```

```bash
cat .git/hooks/commit-msg              # inspect
# Often a commitlint / conventional-commits enforcer.
# Fix message format: "feat(scope): description"
git commit --amend
```

### Husky / pre-commit framework issues

```bash
# Husky v8+ — hooks live in .husky/
ls .husky/
chmod +x .husky/pre-commit
npx husky install                      # re-link hooks
# pre-commit framework (Python):
pre-commit install
pre-commit run --all-files
pre-commit clean                       # nuke caches
# Skipping hooks safely:
SKIP=eslint git commit -m "wip"        # pre-commit framework
HUSKY=0 git commit -m "wip"            # husky bypass
```

## LFS Errors (if Git LFS is in use)

### `Smudge error: Error downloading object: X`

```text
Downloading X (12 MB)
Error downloading object: X (sha256:...): Smudge error: Error downloading object
```

```bash
git lfs install
git lfs env                            # see config
git lfs fetch --all
git lfs pull
git lfs prune                          # gc local cache
```

### `smudge filter lfs failed`

**Cause:** LFS not installed, network failure, or auth issue with LFS server.

```bash
git lfs install --skip-smudge          # skip on clone:
git clone <repo>
cd <repo>
git lfs pull                           # fetch later
```

### `git-lfs: command not found`

```bash
brew install git-lfs                   # macOS
sudo apt install git-lfs               # Debian/Ubuntu
git lfs install                        # set up filters in ~/.gitconfig
```

### `Error: not a git repository` (LFS in non-git dir)

```bash
cd /path/to/your/repo                  # cd into the repo first
git lfs status
```

## Recovery Recipes

### Recover deleted commits via reflog

```bash
git reflog                                  # find HEAD@{N} of the lost commit
git reset --hard HEAD@{5}                   # go back 5 moves ago
# Cherry-pick instead (preserves current HEAD):
git cherry-pick <sha>
# Branch-specific reflog:
git reflog show feature/x
```

### Recover lost stash

```bash
git fsck --no-reflog | grep "dangling commit"
# Then:
git show -p <sha>                           # inspect
git stash apply <sha>                       # apply directly
git update-ref refs/stash <sha>             # restore as stash@{0}
```

### Undo a rebase

```bash
git reset --hard ORIG_HEAD                  # set just before the rebase
# If ORIG_HEAD is gone:
git reflog
git reset --hard HEAD@{N}                   # before "rebase finished"
```

### Undo a `git reset --hard`

```bash
git reflog
# Find the commit that was your HEAD before the reset:
git reset --hard HEAD@{1}
# Or:
git reset --hard <sha>
```

### Recover from `git push --force` overwrite

If your local has the old SHA:

```bash
git push origin <local-sha>:refs/heads/main --force-with-lease
```

If only the team has it: ask them to push their copy. Once GC has run on the server (typically days), it's gone.

### Recover from `git clean -df` of unstaged

**NOT recoverable.** `clean` deletes from the filesystem with no git history.

Mitigation going forward:

```bash
git clean -ndf                              # dry-run first
git config --global clean.requireForce true # always require -f
git stash --include-untracked               # safer than clean
```

### Recover from `rm -rf .git`

**NOT recoverable** unless:

- You have a clone elsewhere — `git clone` it back, then re-add the working tree.
- A Time Machine / backup snapshot exists — restore.
- You committed/pushed beforehand — `git clone <remote>`.

### Squash last N commits

```bash
git rebase -i HEAD~5
# In the editor: keep first as `pick`, change rest to `squash` (or `s`) or `fixup` (no message).
# Then save and rewrite the message.
# Non-interactive equivalent:
git reset --soft HEAD~5
git commit -m "Squashed 5 commits into one"
```

### Split a commit

```bash
git rebase -i <commit>~
# Mark target as `edit` (e), save.
git reset HEAD^                             # uncommit, keep changes staged
# Stage and commit pieces:
git add file1
git commit -m "Part 1"
git add file2
git commit -m "Part 2"
git rebase --continue
```

### Edit an old commit message

```bash
git rebase -i <commit>~
# Mark as `reword` (r), save.
# Editor opens with message; edit and save.
git rebase --continue
# Last commit only:
git commit --amend
```

### Reorder commits

```bash
git rebase -i <oldest-to-touch>~
# Reorder lines in the editor; save.
# Conflicts may appear if commits touch same lines — resolve and continue.
```

### Drop a commit

```bash
git rebase -i <commit>~
# Change `pick` to `drop` (d) or delete the line.
# Conflicts likely if later commits depend on it.
```

### Move commits to a different branch

```bash
git checkout target-branch
git cherry-pick A^..B                       # range
git checkout source-branch
git reset --hard <commit-before-A>          # remove from origin branch
git push --force-with-lease origin source-branch
```

## Force-push Safety

```bash
# DANGER — will overwrite even if remote moved:
git push --force origin main

# SAFE — refuses if remote moved since your last fetch:
git push --force-with-lease origin main

# Even safer — also requires you specify the expected old SHA:
git push --force-with-lease=main:abc123 origin main

# Make --force-with-lease the default for `--force`:
git config --global alias.pushf 'push --force-with-lease'
git config --global push.useForceIfIncludes true
```

Branch protection rules (GitHub): Settings → Branches → branch protection. Recommended for `main`:

- Require pull request reviews
- Require status checks
- Disallow force pushes
- Require linear history (blocks merge commits)
- Restrict who can push

**Never force-push to a shared branch** without telling the team. Anyone whose local is ahead of yours will see the message:

```text
hint: Updates were rejected because the tip of your current branch is behind
```

…and an inexperienced fix is `git push --force` again, escalating the loss.

Recovering from a force-push that ate someone's work:

```bash
# Whoever still has the old commits locally:
git reflog                                  # find the lost SHA
git push origin <sha>:refs/heads/main --force-with-lease
# Or push to a recovery branch first:
git push origin <sha>:recovery/lost-commits
```

## Authentication Errors

### `fatal: Authentication failed for 'https://...'`

```bash
# GitHub removed password auth in 2021 — must use PAT or SSH.
# Make a PAT: github.com → Settings → Developer settings → Tokens (classic or fine-grained)
# Then:
git remote set-url origin https://USERNAME:PAT@github.com/org/repo.git
# Or use credential helper:
git config --global credential.helper osxkeychain        # macOS
git config --global credential.helper manager            # Windows
git config --global credential.helper "cache --timeout=3600"   # Linux memory cache
git config --global credential.helper "store --file ~/.git-credentials"  # plaintext (DON'T)
git config --global credential.helper libsecret          # Linux desktop
```

### `Permission denied (publickey)` — SSH key not added or wrong key

```text
git@github.com: Permission denied (publickey).
fatal: Could not read from remote repository.
```

```bash
ssh -vT git@github.com 2>&1 | grep -i "offering\|accept\|reject"
ssh-add -l                            # list loaded keys
ssh-add ~/.ssh/id_ed25519             # load it
# Generate one:
ssh-keygen -t ed25519 -C "stevie@bellis.tech" -f ~/.ssh/id_ed25519
# Add public key on github.com → Settings → SSH and GPG keys
cat ~/.ssh/id_ed25519.pub | pbcopy    # macOS
# Per-host config:
cat >> ~/.ssh/config <<'EOF'
Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519
    IdentitiesOnly yes
EOF
```

### `ERROR: Repository not found`

**Cause:** wrong account in URL, repo private without auth, or repo deleted/renamed.

```bash
git remote -v
# Test with the right user:
ssh -T git@github.com                  # confirms which user GitHub thinks you are
# Multiple GitHub accounts via SSH:
cat >> ~/.ssh/config <<'EOF'
Host github-work
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_work
EOF
git remote set-url origin git@github-work:org/repo.git
```

### `remote: Invalid username or password`

PAT expired or wrong. Mint a new one:

```bash
git credential reject
protocol=https
host=github.com
username=USERNAME
# (Ctrl-D)
git credential-osxkeychain erase     # macOS Keychain
# Then push and enter the new PAT.
```

### `remote: Support for password authentication was removed on August 13, 2021. Please use a personal access token instead`

GitHub-only message. PAT or SSH; that's it.

```bash
# GitHub Personal Access Tokens (Settings → Developer settings → Tokens (classic)):
# Scopes for git push/pull: `repo` (private) or `public_repo` (public).
# 90-day default expiry; refresh in calendar.
# Fine-grained tokens (newer): repo-scoped, granular permissions.
```

### `credential.helper` config

```bash
git config --global credential.helper                      # show current
git config --global --unset credential.helper              # clear
# Per-host helper:
git config --global credential."https://github.com".helper osxkeychain
# Cache only memory:
git config --global credential.helper "cache --timeout=86400"
# Plain file (not recommended):
git config --global credential.helper "store --file ~/.git-credentials"
chmod 600 ~/.git-credentials
```

### GitHub PAT vs SSH-key vs OAuth-app

- **PAT (classic)** — opaque token, broad scopes. Treat like a password. Rotate quarterly.
- **PAT (fine-grained)** — repo-scoped, expirable, declarative permissions. Preferred for CI.
- **SSH key** — public-key auth, no password, no expiry (until revoked). Best for human dev machines.
- **OAuth app / GitHub App** — for hosted services authenticating on behalf of users.
- **Deploy key** — repo-specific SSH key, read or read+write, ideal for CI/CD per-repo.

### macOS Keychain / Windows Credential Manager / pass / libsecret

```bash
# macOS:
git config --global credential.helper osxkeychain

# Windows (Git for Windows ships this):
git config --global credential.helper manager

# Linux GNOME (libsecret):
sudo apt install libsecret-1-0 libsecret-1-dev
sudo make -C /usr/share/doc/git/contrib/credential/libsecret
git config --global credential.helper /usr/local/bin/git-credential-libsecret

# pass-based (Linux/macOS):
git config --global credential.helper '!f() { /usr/bin/pass-git-helper "$@"; }; f'
```

## Line Ending Issues

### `warning: LF will be replaced by CRLF in X`

```text
warning: in the working copy of 'X', LF will be replaced by CRLF
the next time Git touches it
```

This is a **warning, not an error** — but it's a flag that your project doesn't have line-ending policy.

### `.gitattributes` with `* text=auto` or `* text eol=lf`

```text
# .gitattributes
* text=auto

# Or force LF for everything:
* text eol=lf

# Per-extension:
*.go    text eol=lf
*.md    text eol=lf
*.bat   text eol=crlf
*.png   binary
*.jpg   binary
```

After adding or changing `.gitattributes`, renormalize:

```bash
git add --renormalize .
git commit -m "Renormalize line endings"
```

### `core.autocrlf` input/false/true behavior

- `core.autocrlf=true` (Windows default) — convert LF → CRLF on checkout, CRLF → LF on commit.
- `core.autocrlf=input` (macOS/Linux default in Git for Windows installer) — convert CRLF → LF on commit only; never modify on checkout.
- `core.autocrlf=false` — leave files alone. **Use with `.gitattributes` for explicit control.**

```bash
git config --global core.autocrlf input        # macOS/Linux
git config --global core.autocrlf false        # cross-platform with .gitattributes
git config core.eol lf                         # working tree EOL when text=auto
```

### "All files modified after clone" Windows symptom

**Cause:** `core.autocrlf=true` rewrote line endings; existing files in the repo are CRLF on disk and LF in the index.

```bash
git config core.autocrlf false        # stop the conversions
git rm --cached -r .
git reset --hard HEAD                 # reset checkout to match index
# Or normalize globally:
echo '* text=auto' > .gitattributes
git add --renormalize .
git commit -m "Normalize line endings"
```

## Performance / Repository Bloat

### `git gc` for cleanup

```bash
git gc                                # standard: pack loose objects, prune old
git gc --auto                         # only if needed (per gc.auto threshold)
git count-objects -v                  # before/after metrics
```

### `git gc --aggressive` for deeper repacking (slow)

```bash
git gc --aggressive --prune=now       # very slow on large repos; rarely worth it
```

In modern Git the `--aggressive` flag is rarely a win — repack with explicit window size:

```bash
git repack -ad --depth=50 --window=250
```

### `git repack -ad` for repacking all

```bash
git repack -ad                        # `-a` all, `-d` delete redundant packs
git repack -adfb                      # also build bitmap index
```

### `git prune` for removing unreachable

```bash
git prune --expire=now                # remove objects unreachable from refs
git prune --expire=2.weeks.ago        # cautious
git reflog expire --expire=now --all && git gc --prune=now --aggressive   # full nuke
```

### `git filter-repo` for surgical history rewrite

`git filter-repo` (replaces deprecated `git filter-branch`):

```bash
pip install git-filter-repo

# Remove a file from all history:
git filter-repo --path secret.env --invert-paths

# Replace a string everywhere:
git filter-repo --replace-text <(echo 'OLD_API_KEY==>REDACTED')

# Move subdirectory to root:
git filter-repo --subdirectory-filter src/

# Renaming everything:
git filter-repo --path-rename old/:new/
```

After rewrite, you must force-push and have everyone re-clone (their old SHAs are dead).

### Shallow-clone (--depth=N) tradeoffs

```bash
git clone --depth=1 <url>             # only HEAD; ~10x faster on big repos
git fetch --depth=10                  # extend
git fetch --unshallow                 # convert to full
git fetch --deepen=50                 # add 50 more
```

Caveats: can't push-force from shallow, can't blame deep history, some operations refuse.

### Partial-clone (--filter=blob:none) for blobless

```bash
git clone --filter=blob:none <url>             # all commits/trees, blobs on-demand
git clone --filter=tree:0 <url>                # commits only; trees on-demand
git clone --filter=blob:limit=10m <url>        # blobs <10MB, others on-demand
```

Modern alternative to shallow clones; preserves full history graph but minimizes transfer.

## Bisect Errors

### `fatal: bad revision 'X'` (during bisect)

```bash
git rev-parse X
git bisect start
git bisect bad HEAD
git bisect good v1.0
```

### `You need to start by 'git bisect start'`

```bash
git bisect start
git bisect bad                        # current HEAD is bad
git bisect good v1.0                  # known good
# Git checks out a midpoint; test, then:
git bisect bad / git bisect good
git bisect reset                      # done
```

### `Bisecting: N revisions left to test after this`

Informational — Git computes log2(N) more steps.

### `X is the first bad commit`

Result message — bisect found the offender.

```text
abc123def is the first bad commit
commit abc123def
Author: ...
Date:   ...

    Refactor auth flow

 src/auth.go | 42 ++++++++++++++++++--------------------------
 1 file changed, 17 insertions(+), 25 deletions(-)
```

### `git bisect run <script>` automation

Script must return:
- `0` for good
- non-zero (1-127, except 125) for bad
- `125` for skip (untestable commit)

```bash
git bisect start HEAD v1.0
git bisect run ./test.sh

# Or one-liner:
git bisect run sh -c 'go test ./...'
git bisect run sh -c '! grep -q OFFENDING_LINE src/main.go'
```

## Common Gotchas

### `git push --force` instead of `--force-with-lease`

**Broken:**

```bash
git push --force origin main          # overwrites teammates' work silently
```

**Fixed:**

```bash
git push --force-with-lease origin main
git config --global alias.pushf "push --force-with-lease"
```

### Committing with empty `user.email`

**Broken:**

```bash
git commit -m "..."                   # uses fallback like "user@host.(none)"
```

**Fixed:**

```bash
git config --global user.email stevie@bellis.tech
git log --format='%ae' | sort -u      # audit existing commits
git commit --amend --reset-author     # fix last commit
```

### Forgetting `git pull --rebase`

**Broken:**

```bash
git pull origin main                  # creates merge commit on every pull
```

**Fixed:**

```bash
git pull --rebase origin main
git config --global pull.rebase true
git config --global rebase.autoStash true
```

### `git checkout -- file` (lost changes)

**Broken:**

```bash
git checkout -- src/main.go           # overwrites uncommitted changes silently
```

**Fixed:**

```bash
git restore src/main.go               # equivalent, clearer name
git restore --staged src/main.go      # unstage only
git restore --source=HEAD~3 src/main.go   # restore to older version
# Always preview first:
git diff src/main.go
git stash push -- src/main.go
```

### `git reset` (mixed default) vs `git reset --hard`

**Broken:**

```bash
git reset --hard HEAD~1               # destroys working tree changes too
```

**Fixed:**

```bash
git reset --soft HEAD~1               # uncommit; keep index + working tree
git reset --mixed HEAD~1              # default; uncommit + unstage; keep working tree
git reset --hard HEAD~1               # ONLY when sure
git reset --keep HEAD~1               # like --hard but refuses if local changes conflict
```

### `git stash drop` after stash pop conflict

**Broken:**

```bash
git stash pop                         # conflicts; stash NOT auto-dropped
git stash drop                        # too eager; lost the stash
```

**Fixed:**

```bash
git stash pop
# Resolve conflicts:
git status
# Edit files, then:
git add <files>
git commit -m "..."
# Now safe:
git stash list                        # confirm stash@{0} is the popped one
git stash drop                        # if you really want it gone
# Recover if dropped:
git fsck --no-reflog | grep dangling
```

### Modifying merged history (the never-modify-public-history rule)

**Broken:**

```bash
git rebase -i HEAD~5                  # on a branch others have
git push --force origin main          # everyone's local breaks
```

**Fixed:** rewrite local-only or feature branches; never `main`/`master`/`develop`.

```bash
# If you accidentally rewrote shared history, the only safe response:
git push --force-with-lease origin main
# … and tell the team immediately so they can:
git fetch && git reset --hard origin/main
# (after stashing local work)
```

### LFS tracking added after the fact

**Broken:**

```bash
git lfs track "*.psd"
git add file.psd                       # already committed earlier as a regular blob
# History still has full PSDs; LFS only catches new commits.
```

**Fixed:**

```bash
git lfs migrate import --include="*.psd" --everything
git push --force-with-lease origin main
# Team must re-clone (history rewrite).
```

### `git rm -rf .` then commit with `--amend`

**Broken:**

```bash
git rm -rf .
git commit --amend                     # amends previous commit to be empty
git push --force                       # remote sees an empty repo
```

**Fixed:** **don't.** If you did:

```bash
git reflog                             # find pre-disaster SHA
git reset --hard HEAD@{N}
git push --force-with-lease origin main
```

### Submodule update without `--init`

**Broken:**

```bash
git clone <repo>                      # subdirs empty
ls submod/                            # nothing
```

**Fixed:**

```bash
git submodule update --init --recursive
# Or at clone time:
git clone --recurse-submodules <repo>
git config --global submodule.recurse true   # always
```

### SSH agent not running, key not loaded

**Broken:**

```bash
git push                              # "Permission denied (publickey)"
```

**Fixed:**

```bash
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
ssh -T git@github.com                 # confirm
# Persist on macOS:
ssh-add --apple-use-keychain ~/.ssh/id_ed25519
# In ~/.ssh/config:
cat >> ~/.ssh/config <<'EOF'
Host *
    UseKeychain yes
    AddKeysToAgent yes
EOF
```

### Stale credential cache after rotating PAT

**Broken:**

```bash
# Rotated PAT on GitHub. Local push still uses old one — auth fails.
```

**Fixed (macOS Keychain):**

```bash
git credential-osxkeychain erase
host=github.com
protocol=https
# (Ctrl-D)
# Or interactively:
printf "host=github.com\nprotocol=https\n\n" | git credential reject
git push                              # prompts for fresh PAT
```

## Configuration Errors

### `git config` invalid value formats

```text
fatal: bad numeric config value 'yes' for 'core.compression'
```

```bash
git config --get core.compression
git config core.compression 9          # 0–9 valid
# Boolean values:
git config --bool fetch.prune true     # true/false/yes/no/on/off/0/1
git config --int gc.autoPackLimit 50   # integer
```

### Local vs global vs system precedence

```bash
git config --list --show-origin       # see where each setting comes from
git config --list --local              # current repo only (.git/config)
git config --list --global             # ~/.gitconfig
git config --list --system             # /etc/gitconfig
git config --list --worktree           # current worktree only
```

Precedence (highest → lowest): worktree > local > global > system > default.

### `git config --global --edit` to fix syntax errors

```bash
git config --global --edit             # opens $EDITOR
git config --local --edit
# Validate after:
git config --list --show-origin
```

### `GIT_DIR` / `GIT_WORK_TREE` env var conflicts

```bash
echo $GIT_DIR
echo $GIT_WORK_TREE
unset GIT_DIR GIT_WORK_TREE
# Symptoms when set wrong: "fatal: not a git repository", commands operate on the wrong repo.
# Use them deliberately:
git --git-dir=/path/to/.git --work-tree=/path/to/wt status
```

## Idioms

```bash
git status -sb                              # short branch summary
# ## main...origin/main [ahead 2]
#  M README.md
# ?? new.txt

git log --oneline --graph --decorate --all  # tree visualization
git log --oneline --graph --all --simplify-by-decoration  # only branch tips/tags

git diff                                    # working tree vs index
git diff --staged                           # index vs HEAD (alias: --cached)
git diff HEAD                               # working tree vs HEAD
git diff main..feature                      # range diff (two-dot)
git diff main...feature                     # diff from merge-base (three-dot)

git push --force-with-lease                 # SAFE force-push
git pull --rebase                           # linear history
git rebase --autostash                      # preserve dirty tree across rebase

# Aliases worth keeping:
git config --global alias.st 'status -sb'
git config --global alias.lg "log --graph --pretty=format:'%C(yellow)%h%Creset -%C(red)%d%Creset %s %C(green)(%cr) %C(blue)<%an>%Creset' --abbrev-commit"
git config --global alias.pushf 'push --force-with-lease'
git config --global alias.unstage 'reset HEAD --'
git config --global alias.last 'log -1 HEAD'
git config --global alias.amend 'commit --amend --no-edit'
git config --global alias.aliases "config --get-regexp '^alias\.'"

# Fixup workflow:
git commit --fixup=<sha>                    # creates "fixup! Original message"
git rebase -i --autosquash <sha>~

# Find any string anywhere in history:
git log -S 'OFFENDING_STRING' --oneline    # commits that added/removed it
git log -G 'regex' --oneline                # regex-on-diff
git log --all --source --remotes -- path    # path across all refs

# Compact day-to-day status:
alias gst='git status -sb'
alias glg='git log --oneline --graph --decorate'
alias gd='git diff'
alias gds='git diff --staged'

# Useful one-shot:
git for-each-ref --sort=-committerdate refs/heads --format='%(committerdate:short) %(refname:short)'
git branch --merged main | grep -v '^\*' | xargs -n1 git branch -d  # cleanup merged branches
```

## See Also

- git
- github
- gitlab
- troubleshooting/ssh-errors
- polyglot

## References

- git-scm.com/docs — official Git reference manual
- git-scm.com/book/en/v2 — Pro Git book (free)
- github.com/git/git/tree/master/Documentation — raw man pages
- ohshitgit.com — recipe site for "I just made a horrible mistake"
- git-scm.com/docs/gitglossary — terminology
- git-scm.com/docs/git-config — every config option
- git-scm.com/docs/gitrevisions — `<rev>` syntax (HEAD~3, abc..def, etc.)
- git-scm.com/docs/giteveryday — common workflows
- git-scm.com/docs/gitfaq — official FAQ
- github.com/newren/git-filter-repo — modern history-rewrite tool
- github.com/git-lfs/git-lfs — Git LFS docs
- docs.github.com/en/authentication — GitHub auth (PATs, SSH, OAuth)
- docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/about-protected-branches
- docs.gitlab.com/ee/topics/git/ — GitLab's Git guide
- man git-rebase, man git-merge, man git-reset, man git-reflog — local manpages
- RFC 8174 (interpreting MUST/SHOULD in messages) — for those reading hook output literally
- CVE-2022-24765 — `safe.directory` introduction context
- CVE-2022-39253 — bare-repo embedded-config exploit
- CVE-2024-32002 — symlink-handling vulnerability (patch git ASAP if pre-2.45.1)
