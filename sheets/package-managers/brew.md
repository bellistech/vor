# Homebrew (brew)

> Package manager for macOS (and Linux) — install CLI tools, apps, and manage services.

## Install and Uninstall

### Formulae (CLI Tools)

```bash
brew install ripgrep                   # install formula
brew install gcc@13                    # install specific version
brew install --HEAD neovim             # install from latest git commit
brew uninstall ripgrep                 # remove formula
brew reinstall ripgrep                 # reinstall
brew install --build-from-source vim   # compile from source
```

### Casks (GUI Applications)

```bash
brew install --cask firefox            # install GUI app
brew install --cask visual-studio-code
brew uninstall --cask firefox          # remove app
brew install --cask --appdir=~/Apps firefox  # custom install location
```

## Update and Upgrade

```bash
brew update                            # update Homebrew + tap metadata
brew outdated                          # list outdated packages
brew upgrade                           # upgrade all packages
brew upgrade ripgrep                   # upgrade specific formula
brew upgrade --cask                    # upgrade all casks
brew upgrade --cask --greedy           # upgrade casks that auto-update too
```

## Search and Info

```bash
brew search nginx                      # search formulae and casks
brew search --cask chrome              # search casks only
brew search /^vim/                     # regex search
brew info ripgrep                      # show formula info
brew info --cask firefox               # show cask info
brew info --json=v2 ripgrep            # JSON output
brew list                              # list installed formulae
brew list --cask                       # list installed casks
brew list ripgrep                      # list files installed by formula
brew deps ripgrep                      # show dependencies
brew deps --tree ripgrep               # dependency tree
brew uses --installed gcc              # what depends on gcc (installed only)
```

## Taps (Third-Party Repos)

```bash
brew tap                               # list taps
brew tap homebrew/services             # add tap
brew untap homebrew/services           # remove tap
brew tap user/repo https://github.com/user/homebrew-repo.git  # custom tap
```

## Services

```bash
brew services list                     # list managed services
brew services start postgresql@16      # start service (and enable on boot)
brew services stop postgresql@16       # stop service
brew services restart nginx            # restart service
brew services run redis                # start without boot launch (one-time)
brew services info nginx               # show service status
```

## Pinning

```bash
brew pin node                          # prevent upgrades
brew unpin node                        # allow upgrades again
brew list --pinned                     # list pinned formulae
```

## Cleanup

```bash
brew cleanup                           # remove old versions and cache
brew cleanup -n                        # dry run: show what would be removed
brew cleanup -s                        # also remove downloads for latest versions
brew cleanup --prune=7                 # remove cache entries older than 7 days
brew autoremove                        # remove unused dependencies

# check disk usage
du -sh $(brew --cache)                 # cache size
du -sh $(brew --prefix)                # total install size
```

## Diagnostics

```bash
brew doctor                            # check for issues
brew missing                           # check for missing dependencies
brew config                            # show Homebrew configuration
brew --prefix                          # show install prefix (/opt/homebrew on arm64)
brew --cellar                          # show cellar path
brew --cache                           # show download cache path
```

## Bundle (Brewfile)

```bash
# generate Brewfile from currently installed packages
brew bundle dump                       # creates ./Brewfile
brew bundle dump --force               # overwrite existing

# install from Brewfile
brew bundle                            # install everything in Brewfile
brew bundle --file=~/dotfiles/Brewfile

# check what's missing
brew bundle check
brew bundle cleanup                    # show packages not in Brewfile
brew bundle cleanup --force            # actually uninstall them
```

### Brewfile Format

```bash
# taps
tap "homebrew/services"

# formulae
brew "ripgrep"
brew "node@20"
brew "go"
brew "neovim", args: ["HEAD"]

# casks
cask "firefox"
cask "iterm2"
cask "docker"

# Mac App Store (requires mas CLI)
mas "Xcode", id: 497799835
```

## Linking

```bash
brew link node                         # create symlinks in /opt/homebrew/bin
brew link --overwrite node             # force link (overwrite existing)
brew unlink node                       # remove symlinks
brew link --overwrite --force node@20  # link keg-only formula
```

## Editing and Development

```bash
brew edit ripgrep                      # edit formula
brew create https://example.com/tool-1.0.tar.gz  # create new formula
brew audit --new ripgrep               # check formula for issues
brew test ripgrep                      # run formula tests
```

## Tips

- On Apple Silicon (M1+), Homebrew installs to `/opt/homebrew`. On Intel, it uses `/usr/local`.
- `brew upgrade --greedy` is needed for casks that have auto-update mechanisms (Chrome, Firefox, etc.).
- `brew bundle dump` is the best way to reproduce your setup on a new machine. Check the Brewfile into your dotfiles.
- Keg-only formulae (like `openssl`, `python@3.12`) are installed but not linked. Use `brew link --force` if needed.
- `brew pin` prevents accidental upgrades of critical tools. Always pin database servers in development.
- `HOMEBREW_NO_AUTO_UPDATE=1 brew install pkg` skips the auto-update check for faster installs.
- `brew cleanup` can free gigabytes of disk space. It removes old versions and cached downloads.
- `brew services` manages launchd plists -- services persist across reboots unless started with `run`.
- `brew deps --tree --installed` shows the full dependency tree of everything installed.
- Avoid `sudo brew`. Homebrew is designed to run as your user. If it asks for sudo, something is misconfigured.
