# Git (Version Control)

> Distributed version control system — track changes, branch, merge, and collaborate.

## Setup

### Init and Clone

```bash
git init                               # initialize new repo
git init --bare                        # bare repo (for servers)
git clone https://github.com/user/repo.git
git clone git@github.com:user/repo.git # SSH
git clone --depth 1 repo-url           # shallow clone (latest only)
git clone --branch v2.0 repo-url      # clone specific branch/tag
```

### Config

```bash
git config user.name "Alice Smith"
git config user.email "alice@example.com"
git config --global core.editor "vim"
git config --global init.defaultBranch main
git config --global pull.rebase true
git config --global push.autoSetupRemote true   # auto --set-upstream
git config --global rerere.enabled true          # remember merge resolutions
git config --list --show-origin                  # show all config with source
```

## Staging and Committing

```bash
git add file.go                        # stage file
git add src/                           # stage directory
git add -p                             # stage interactively (hunk by hunk)
git add -N file.go                     # track file but don't stage content
git reset HEAD file.go                 # unstage file
git restore --staged file.go           # unstage (modern)

git commit -m "Add user auth"
git commit -am "Fix typo"              # stage tracked + commit
git commit --amend                     # amend last commit
git commit --amend --no-edit           # amend without changing message
git commit --allow-empty -m "Trigger CI"
```

## Branching

```bash
git branch                             # list local branches
git branch -a                          # list all (including remote)
git branch feature/auth                # create branch
git branch -d feature/auth             # delete (safe — must be merged)
git branch -D feature/auth             # delete (force)
git branch -m old-name new-name        # rename branch
git branch --merged                    # branches merged into current

git switch feature/auth                # switch to branch
git switch -c feature/auth             # create and switch
git checkout feature/auth              # switch (older syntax)
git checkout -b feature/auth           # create and switch (older)
```

## Merging

```bash
git merge feature/auth                 # merge branch into current
git merge --no-ff feature/auth         # force merge commit
git merge --squash feature/auth        # squash all commits into one
git merge --abort                      # cancel in-progress merge

# resolve conflicts
git diff --name-only --diff-filter=U   # list conflicted files
# edit files, remove conflict markers
git add resolved-file.go
git merge --continue
```

## Rebasing

```bash
git rebase main                        # rebase current branch onto main
git rebase --onto main feature base    # rebase subset of commits
git rebase --abort                     # cancel rebase
git rebase --continue                  # after resolving conflicts
git rebase --skip                      # skip current commit

# interactive rebase (last 5 commits)
git rebase -i HEAD~5
# in editor: pick, reword, edit, squash, fixup, drop
```

## Cherry-Pick

```bash
git cherry-pick abc1234                # apply single commit
git cherry-pick abc1234 def5678        # apply multiple commits
git cherry-pick abc1234..def5678       # apply range (exclusive start)
git cherry-pick --no-commit abc1234    # apply without committing
git cherry-pick --abort                # cancel
```

## Stash

```bash
git stash                              # stash tracked changes
git stash -u                           # include untracked files
git stash push -m "wip: auth"          # named stash
git stash push src/auth.go             # stash specific files
git stash list                         # list stashes
git stash show -p stash@{0}            # show stash diff
git stash pop                          # apply and remove latest
git stash apply stash@{2}              # apply without removing
git stash drop stash@{0}              # delete stash
git stash clear                        # delete all stashes
```

## Log and History

```bash
git log --oneline                      # compact log
git log --oneline --graph --all        # branch graph
git log --stat                         # files changed per commit
git log -p                             # full diffs
git log -5                             # last 5 commits
git log --since="2 weeks ago"
git log --author="Alice"
git log -- src/auth.go                 # history of specific file
git log -S "functionName"              # search for string in diffs (pickaxe)
git log -G "regex"                     # search with regex
git log --follow -- renamed-file.go    # follow renames
git shortlog -sn                       # commit count by author
```

## Diff

```bash
git diff                               # unstaged changes
git diff --staged                      # staged changes
git diff HEAD                          # all uncommitted changes
git diff main..feature                 # between branches
git diff HEAD~3..HEAD                  # last 3 commits
git diff --stat                        # summary only
git diff --name-only                   # changed filenames only
git diff --word-diff                   # inline word-level diff
```

## Remote

```bash
git remote -v                          # list remotes
git remote add origin git@github.com:user/repo.git
git remote rename origin upstream
git remote remove upstream
git remote set-url origin new-url
git fetch origin                       # download without merging
git fetch --all                        # fetch all remotes
git fetch --prune                      # remove stale remote branches
```

## Push and Pull

```bash
git push origin main                   # push to remote
git push -u origin feature/auth        # push and set upstream
git push --force-with-lease            # safe force push
git push --tags                        # push all tags
git push origin --delete feature/auth  # delete remote branch

git pull                               # fetch + merge
git pull --rebase                      # fetch + rebase
git pull origin main                   # pull specific branch
```

## Tags

```bash
git tag v1.0.0                         # lightweight tag
git tag -a v1.0.0 -m "Release 1.0"    # annotated tag
git tag -a v1.0.0 abc1234              # tag specific commit
git tag -l "v1.*"                      # list matching tags
git tag -d v1.0.0                      # delete local tag
git push origin v1.0.0                 # push tag
git push origin --tags                 # push all tags
git push origin --delete v1.0.0        # delete remote tag
```

## Reset and Revert

```bash
# reset (moves HEAD, rewrites history)
git reset --soft HEAD~1                # undo commit, keep staged
git reset HEAD~1                       # undo commit, keep unstaged (--mixed)
git reset --hard HEAD~1                # undo commit, discard changes
git reset --hard origin/main           # match remote exactly

# revert (creates new commit, safe for shared branches)
git revert abc1234                     # revert single commit
git revert abc1234..def5678            # revert range
git revert --no-commit abc1234         # revert without committing
```

## Restore and Clean

```bash
git restore file.go                    # discard unstaged changes
git restore --staged file.go           # unstage
git restore --source=HEAD~2 file.go    # restore from specific commit
git checkout -- file.go                # discard changes (older syntax)

git clean -n                           # dry run: show what would be deleted
git clean -fd                          # remove untracked files and dirs
git clean -fdx                         # also remove ignored files
```

## Bisect

```bash
git bisect start
git bisect bad                         # current commit is bad
git bisect good v1.0.0                 # known good commit
# git checks out middle commit — test it
git bisect good                        # or: git bisect bad
# repeat until found
git bisect reset                       # return to original branch

# automated bisect
git bisect start HEAD v1.0.0
git bisect run ./test.sh               # script exits 0=good, 1=bad
```

## Worktree

```bash
git worktree add ../hotfix hotfix/123  # new worktree with branch
git worktree add ../review feature     # checkout branch in new dir
git worktree list                      # list worktrees
git worktree remove ../hotfix          # remove worktree
```

## Reflog

```bash
git reflog                             # history of HEAD movements
git reflog show feature/auth           # reflog for specific branch
git checkout HEAD@{3}                  # go to 3 moves ago
git reset --hard HEAD@{5}              # recover lost commits
```

## Submodules

```bash
git submodule add https://github.com/lib/lib.git vendor/lib
git submodule init                     # initialize after clone
git submodule update --init --recursive
git submodule update --remote          # pull latest for all submodules
git submodule foreach git pull origin main
git submodule deinit vendor/lib        # unregister
```

## Tips

- `git push --force-with-lease` is always safer than `--force` -- it refuses if the remote has commits you haven't seen.
- `git add -p` is the best way to make clean, focused commits. Review every hunk.
- `git stash -u` includes untracked files. Without `-u`, new files are left behind.
- `git log -S "text"` (pickaxe) finds the commit that introduced or removed a specific string.
- `git reflog` is your safety net. Almost nothing in git is truly lost if you act within 90 days (the default reflog expiry).
- `git rebase -i` with `fixup` lets you squash commits without editing messages -- pair with `git commit --fixup=<sha>`.
- `git rerere` (reuse recorded resolution) remembers how you resolved conflicts and auto-applies them next time.
- `git diff --cached` is the same as `git diff --staged`.
- `git switch` and `git restore` are the modern replacements for `git checkout` -- they split its overloaded behavior.
- `git worktree` lets you work on multiple branches simultaneously without stashing or cloning again.

## See Also

- bash, vim, make, shell-scripting, github-actions, ssh

## References

- [Git Documentation](https://git-scm.com/doc) -- official docs, book, and videos
- [Git Reference](https://git-scm.com/docs) -- man pages for every git command
- [Pro Git Book](https://git-scm.com/book/en/v2) -- free comprehensive book by Scott Chacon and Ben Straub
- [man git](https://man7.org/linux/man-pages/man1/git.1.html) -- git man page
- [man gitrevisions](https://man7.org/linux/man-pages/man7/gitrevisions.7.html) -- revision and range syntax (HEAD~2, @{upstream}, etc.)
- [man gitworkflows](https://man7.org/linux/man-pages/man7/gitworkflows.7.html) -- recommended workflows
- [man gitattributes](https://man7.org/linux/man-pages/man5/gitattributes.5.html) -- per-path settings (LFS, diff, merge)
- [Git Internals (Pro Git Ch. 10)](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain) -- objects, refs, packfiles
- [git-tips](https://github.com/git-tips/tips) -- collection of practical git tips
- [Oh Shit, Git!?!](https://ohshitgit.com/) -- how to undo common git mistakes
- [Git Flight Rules](https://github.com/k88hudson/git-flight-rules) -- what to do when things go wrong
