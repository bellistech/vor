# gopass

Go reimplementation and superset of `pass(1)` — the standard Unix password manager. Encrypted secrets in a git-versioned tree, GPG or age crypto, multi-store mounts, OTP, fuzzy search, REST/JSON-API, browser bridge, and a hardened CLI for terminal-bound workflows.

## Setup

```bash
# macOS — Homebrew
brew install gopass
brew install gopass-jsonapi              # browser bridge (optional)
brew install gnupg pinentry-mac          # GPG backend + GUI passphrase prompt
brew install age                          # age backend (optional, modern)
```

```bash
# Debian / Ubuntu
sudo apt update
sudo apt install gopass gnupg2 git
sudo apt install rng-tools                # entropy for key generation
```

```bash
# Fedora / RHEL / Rocky
sudo dnf install gopass gnupg2 git
```

```bash
# Arch / Manjaro
sudo pacman -S gopass gnupg git
yay -S gopass-jsonapi-bin                 # AUR
```

```bash
# Alpine
apk add gopass gnupg git
```

```bash
# From release (any Linux/BSD)
GP_VERSION=$(curl -fsSL https://api.github.com/repos/gopasspw/gopass/releases/latest | jq -r .tag_name)
curl -fsSL "https://github.com/gopasspw/gopass/releases/download/${GP_VERSION}/gopass-${GP_VERSION#v}-linux-amd64.tar.gz" \
  | sudo tar -xz -C /usr/local/bin gopass
gopass version
```

```bash
# From source (Go 1.22+)
go install github.com/gopasspw/gopass@latest
which gopass                              # ~/go/bin/gopass on default GOBIN
```

```bash
# Verify install
gopass version                            # gopass 1.15.x go1.22.x linux amd64
gopass --help
gopass completion bash > /etc/bash_completion.d/gopass
gopass completion zsh   > "${fpath[1]}/_gopass"
gopass completion fish  > ~/.config/fish/completions/gopass.fish
```

```bash
# Confirm the upstream repo + lineage
gopass version --json | jq .
# gopass is a Go reimplementation/superset of `pass(1)` — the bash
# "standard unix password manager" (passwordstore.org). Reads/writes
# the same ~/.password-store layout, adds mounts, age, OTP, fuzzy
# search, JSON-API, audit, fsck, templates, aliases, recipients UX.
```

## Why gopass

```bash
# vs pass(1) — the bash original
#   * Single static Go binary; no bash + 12 coreutils dependency soup
#   * Multiple stores (mounts) — separate work / personal / clients
#   * age crypto support (gpg or age, per-store)
#   * Built-in fuzzy search (gopass find / search)
#   * Auto-push to git remote on every change (configurable)
#   * Audit (weak/duplicate/age) and fsck (recipient drift) commands
#   * OTP (TOTP/HOTP) generation built in (no pass-otp extension)
#   * Templates for new-secret scaffolding
#   * JSON-API + browser bridge (gopass-bridge) without shell hacks
#   * Per-folder recipients with .gpg-id propagation UX
#   * Faster cold start (~30ms vs ~150ms) — Go vs bash + GPG fork
```

```bash
# vs Bitwarden / 1Password / Dashlane (SaaS)
#   * Terminal-native — no Electron app, no browser-only UI
#   * Encrypted-at-rest in YOUR own git repo (self-hosted)
#   * No SaaS dependency, no monthly fee, no vendor lock-in
#   * Works offline; sync is plain git push/pull over SSH/HTTPS
#   * Crypto you can audit (GPG / age) — no proprietary blob format
#   * Plain-text + git history = grep-able audit trail
#   * Works with smartcards (Yubikey/Nitrokey) via gpg-agent
#   * No browser extension required (but available as gopass-bridge)
#   * pass(1) compatible — escape hatch is "use any other tool that
#     reads ~/.password-store"
```

```bash
# vs HashiCorp Vault / AWS Secrets Manager / GCP Secret Manager
#   * No server to run, patch, or pay for
#   * No HSM, no Raft cluster, no policy engine to misconfigure
#   * Static binary on a laptop — appropriate for single user / small team
#   * Vault wins for: dynamic secrets (DB creds, AWS STS), revocation,
#     leases, ACLs, audit log; gopass is for human-edited static secrets
```

```bash
# When NOT to use gopass
#   * Dynamic secrets that rotate per-request   → Vault
#   * Service accounts in Kubernetes pods       → SOPS + age + sealed-secrets
#   * High-frequency machine-to-machine secrets → Vault / cloud KMS
#   * Need browser autofill UX for non-tech users → Bitwarden
```

## Architecture

```bash
# Default store layout (one mount, GPG)
~/.password-store/
├── .gpg-id                   # GPG recipient key IDs, one per line
├── .gopass.yml               # gopass per-store config (optional)
├── .public-keys/             # exported public keys for collaborators
│   └── 0xDEADBEEFCAFEBABE.asc
├── .git/                     # git repo for sync + history
├── .gitattributes
├── .gitignore
├── personal/
│   ├── .gpg-id               # per-folder override (optional)
│   ├── email/
│   │   └── gmail.gpg         # encrypted secret file (file-per-secret)
│   └── bank.gpg
└── work/
    ├── .gpg-id               # different recipient set for shared work secrets
    ├── ssh/
    │   └── prod-key-pass.gpg
    └── aws/
        └── root-account.gpg
```

```bash
# age-crypto store layout
~/.password-store/
├── .age-recipients           # age recipient lines (age1xyz... or ssh-ed25519...)
├── .age-ids                  # local age private keys (encrypted)
├── personal/
│   └── bank.age              # encrypted with age
└── ...
```

```bash
# Multi-store (mounts)
~/.password-store/            # default mount, "" (root)
~/.password-store-work/       # mounted as "work"
~/.password-store-clients/    # mounted as "clients"

# Each mount has its own:
#   * .gpg-id / .age-recipients
#   * .git/ remote
#   * crypto backend choice (gpg or age)
#   * recipients (different teams = different keys)
```

```bash
# File-per-secret rationale
# Each secret = one file → one git diff per change → easy audit
# Decrypt is per-file → no monolithic blob to load
# Per-folder .gpg-id → recipient-aware encryption boundaries
# Filename = path → secret/foo/bar.gpg shown as "secret/foo/bar"
```

```bash
# Internal data model — what's INSIDE a decrypted .gpg/.age file
<password-as-first-line>
<blank line>
<arbitrary YAML / KEY: value lines or freeform notes>
url: https://example.com
user: alice
otpauth://totp/Example:alice?secret=JBSWY3DPEHPK3PXP&issuer=Example
notes: |
  Account opened 2024-01-15.
  Recovery codes:
    - aaaa-bbbb
```

## Initialization

```bash
# Interactive setup wizard — recommended first run
gopass setup
# Prompts:
#   1. Crypto backend: gpg or age?
#   2. Storage: filesystem (fs) [default]
#   3. Existing GPG key? Or generate one?
#   4. Initialize git? Add remote?
#   5. Auto-sync? Auto-clip? Notifications?
```

```bash
# Explicit GPG init — use existing GPG key
gpg --list-secret-keys --keyid-format=long
# /home/alice/.gnupg/secring.gpg
# sec   rsa4096/0xDEADBEEFCAFEBABE 2024-01-01 [SC]
gopass init 0xDEADBEEFCAFEBABE
# Initializes ~/.password-store with that key as recipient
```

```bash
# Multiple recipients at init (shared store)
gopass init 0xDEADBEEFCAFEBABE 0xCAFED00D12345678
# Both keys can decrypt every secret
```

```bash
# age + filesystem (modern, no GPG)
gopass init --crypto age --storage fs
# Generates an age identity at ~/.config/gopass/age/identities
# Adds the corresponding recipient to .age-recipients
```

```bash
# Re-init / additional store
gopass init --store work 0xDEADBEEFCAFEBABE
# Initializes a separate mount named "work"
```

```bash
# Generate a new GPG key first (if you don't have one)
gpg --full-generate-key
# Choose: (1) RSA and RSA, 4096 bits, 2y expiry
# Real name: Alice Example
# Email: alice@example.com
# Strong passphrase!
gpg --list-secret-keys --keyid-format=long
gopass init alice@example.com
```

```bash
# Recipients file inspection
cat ~/.password-store/.gpg-id
# 0xDEADBEEFCAFEBABE
# 0xCAFED00D12345678
```

```bash
# Convert an existing pass(1) store
ls ~/.password-store         # already exists from pass(1)?
gopass setup --remote        # gopass detects existing store, adopts it
gopass list                  # confirm secrets visible
```

## Stores / Mounts

```bash
# List all mounts
gopass mounts list
# Mount       Path                                 Type
# <root>      /home/alice/.password-store          fs+gpg
# work        /home/alice/.password-store-work     fs+gpg
# clients     /home/alice/.password-store-clients  fs+age
```

```bash
# Add a new mount (clone or local path)
gopass mounts add work /home/alice/.password-store-work
# If the path doesn't exist yet, gopass init the new store
gopass init --store work alice@example.com bob@example.com
```

```bash
# Add a mount from an existing remote (clone)
gopass clone git@github.com:acme/secrets.git work
# Clones to a default path and mounts as "work"

gopass clone --path /opt/secrets git@github.com:acme/secrets.git work
# Custom path
```

```bash
# Remove a mount (does NOT delete files on disk)
gopass mounts remove work
ls ~/.password-store-work    # still there — manually rm -rf if you want
```

```bash
# Address secrets in a non-default mount
gopass show work/aws/prod-key
gopass show clients/acme/db-password
gopass insert work/new/secret
```

```bash
# Per-store crypto choice
gopass init --crypto age --storage fs --store personal
gopass init --crypto gpg --storage fs --store work
# personal/ uses age, work/ uses gpg, simultaneously
```

```bash
# Multi-tenant separation rationale
# Use mounts when:
#   * Recipients differ (work team vs personal vs client A vs client B)
#   * Git remotes differ (private GH repo vs corporate GitLab)
#   * Crypto differs (legacy GPG store, new age store)
#   * Backup policy differs (personal local-only vs work pushed remote)
```

## Inserting Secrets

```bash
# Single-line password (interactive prompt)
gopass insert personal/email/gmail
# Enter password for personal/email/gmail:
# Retype password: 
# Pasted from clipboard? Use --force / -f to overwrite existing
```

```bash
# Multi-line — opens a buffer for entire body
gopass insert -m personal/email/gmail
# Stdin until Ctrl-D, OR opens $EDITOR if -e
# Convention:
#   line 1: password
#   line 2: blank
#   line 3+: freeform key: value or notes
```

```bash
# Editor mode
gopass insert -e personal/email/gmail
# $EDITOR opens with empty buffer; save+quit to commit
```

```bash
# Force overwrite existing secret
gopass insert -f personal/email/gmail
# Without -f, gopass refuses to overwrite (returns "secret exists")
```

```bash
# Pipe a file in as the entire body
gopass insert -m personal/notes/recovery-codes < codes.txt

# Pipe a single-line password from another tool
echo -n 'hunter2' | gopass insert -m personal/throwaway
# -m is required even for single line when piping
```

```bash
# Append to existing multi-line secret
gopass show personal/email/gmail | { cat; echo "new: line"; } | \
  gopass insert -mf personal/email/gmail
# Re-encrypts the whole secret with the appended line
```

```bash
# Copy from clipboard (avoid passwords in shell history)
xclip -o -selection clipboard | gopass insert -m personal/email/gmail
# Or pbpaste on macOS:
pbpaste | gopass insert -m personal/email/gmail
```

```bash
# Prevent the secret being printed back after creation
GOPASS_NO_NOTIFY=1 gopass insert personal/secret
```

## Generating

```bash
# Generate a 32-char random password
gopass generate personal/email/gmail 32
# 32 is the password LENGTH (default 24)
# Generated and stored; printed unless --no-print-password
```

```bash
# XKCD-style passphrase (correct horse battery staple)
gopass generate --xkcd personal/wifi/home 5
# 5 = WORDS, not characters
# Default 4 words, dash-separated, lowercase
```

```bash
# XKCD options
gopass generate --xkcd --xkcd-sep '_'        personal/secret 4
gopass generate --xkcd --xkcd-lang en        personal/secret 4   # de, fr, es, it
gopass generate --xkcd --xkcd-capitalize     personal/secret 4   # Capitalize-Each-Word
gopass generate --xkcd --xkcd-numbers        personal/secret 4   # append digits
```

```bash
# Symbols / no symbols
gopass generate -s personal/secret 32       # include !@#$%^&* etc.
gopass generate -n personal/secret 32       # no symbols (alphanumeric only)
gopass generate    personal/secret 32       # default: alphanumeric
```

```bash
# Length flag (alternative to positional arg)
gopass generate -p 40 personal/secret       # -p = print/length (alias)
gopass generate --print personal/secret 32  # also print to stdout
```

```bash
# Force overwrite existing
gopass generate -f personal/secret 32
```

```bash
# Generate WITHOUT printing to stdout (prints by default to clipboard)
gopass generate --clip personal/secret 32   # -c also works
gopass generate -c -s personal/secret 32

# Some versions: --no-print-password
gopass generate --no-print-password personal/secret 32
```

```bash
# Generate into a multi-line secret (overwrite first line only)
echo "old body" | gopass insert -mf personal/secret
gopass generate personal/secret 32
# First line replaced with new password, rest preserved
```

```bash
# Custom character set (regex/charset, gopass 1.15+)
gopass generate --generator memorable personal/secret 24
# generators: cryptic (default), memorable, xkcd, external
```

## Showing

```bash
# Print the entire decrypted secret to stdout
gopass show personal/email/gmail
# password
# 
# url: https://gmail.com
# user: alice@gmail.com

# Note: gopass warns when stdout is a terminal (you might shoulder-surf)
```

```bash
# Password only — first line only
gopass show -o personal/email/gmail
gopass show --password personal/email/gmail
# Useful for piping into other tools
```

```bash
# Copy password to clipboard (auto-clears after 45s by default)
gopass show -c personal/email/gmail
gopass show --clip personal/email/gmail

# Copy with explicit clear timeout (seconds)
gopass show -C 30 personal/email/gmail
# Some versions: --clip-timeout 30
```

```bash
# Show a specific YAML key from the body
gopass show personal/email/gmail user
# alice@gmail.com
# (parses lines after the blank line as KEY: VALUE)
```

```bash
# Suppress the "we are not in a TTY" / "stdout-not-tty" warnings
gopass show --unsafe -o personal/email/gmail
# --unsafe disables the safety guards — only use in scripts you trust
```

```bash
# Git history of a secret (revisions)
gopass history personal/email/gmail
# 0  2025-01-15 abcd1234  alice@example.com  Update gmail password
# 1  2024-09-01 ef567890  alice@example.com  Add gmail

gopass show --revision=ef567890 personal/email/gmail
gopass show --revision=HEAD~3   personal/email/gmail
gopass show --revision=1        personal/email/gmail   # by index in `history`
```

```bash
# QR code output (great for phone-to-laptop transfer)
gopass show --qr personal/email/gmail
# Renders a terminal QR of the password
```

```bash
# Password-only vs first-line semantics
# `gopass show -o` => exactly the first line, raw
# `gopass show`    => entire body (password + blank + freeform)
# `gopass show <path> <key>` => parses freeform as YAML, returns one key
```

## Listing

```bash
# Tree view (default)
gopass ls
# gopass
# ├── personal/
# │   ├── email/
# │   │   └── gmail
# │   └── bank
# └── work/
#     └── aws/
#         └── prod-key

gopass list                                # alias
gopass tree                                # explicit tree view
```

```bash
# Limit depth
gopass ls --depth 2
gopass ls -d 2                             # short form (when supported)
```

```bash
# Strip the prefix (clean output for piping)
gopass ls --strip-prefix personal/
# email/gmail
# bank
```

```bash
# Flat output — newline-delimited paths (script-friendly)
gopass ls --flat
# personal/email/gmail
# personal/bank
# work/aws/prod-key
```

```bash
# List a subfolder
gopass ls personal/email
# gopass
# └── personal/email/
#     └── gmail
```

```bash
# Show only folders / only secrets
gopass ls --folders                        # directories only
gopass ls --no-folders                     # secrets only
```

```bash
# Combine with shell tools
gopass ls --flat | grep -i aws
gopass ls --flat | wc -l                   # total secret count
```

## Searching

```bash
# Substring search across paths (case-insensitive)
gopass find gmail
# personal/email/gmail

gopass search gmail                        # alias of find
```

```bash
# Search across decrypted bodies (slow! decrypts everything matching path filter)
gopass grep 'TODO'
gopass grep --regexp 'pwd_[0-9]+'          # regex mode
```

```bash
# Fuzzy / interactive search (TUI picker)
gopass select                              # newer versions
gopass find -i pattern                     # interactive in some versions
```

```bash
# Case sensitivity — find is case-INSENSITIVE by default
gopass find GMAIL                          # matches personal/email/gmail
gopass find --regexp '^Personal/'          # regex flag enables case-sensitive
```

```bash
# Pipe to fzf for full fuzzy UX
gopass ls --flat | fzf | xargs -r gopass show -c
# Pick interactively, copy to clipboard
```

## Editing

```bash
# Edit the secret — opens $EDITOR with current decrypted contents
gopass edit personal/email/gmail
# $EDITOR (vim, nvim, nano, code -w, helix...) opens
# Save+quit → re-encrypts and commits
# Quit without saving → no change
```

```bash
# Convention preserved on edit
# Line 1 = password (do not put a label on this line!)
# Line 2 = blank
# Line 3+ = arbitrary YAML key: value or freeform notes
```

```bash
# Set the editor explicitly for one invocation
EDITOR='code --wait' gopass edit personal/email/gmail
EDITOR='nvim'        gopass edit personal/email/gmail
```

```bash
# Edit a specific revision (rare — see history + show --revision instead)
# gopass cannot edit a past revision directly; use:
gopass show --revision=HEAD~1 personal/email/gmail | gopass insert -mf personal/email/gmail
# Restores yesterday's value as a NEW commit
```

```bash
# Aborted edit recovery
# If $EDITOR crashes mid-edit, gopass leaves a temp file; subsequent
# `gopass edit` warns and offers recovery. Manual:
ls ~/.cache/gopass/edit-*.tmp
# rm to discard, cat to recover
```

## Copying

```bash
# Copy a secret to a new path (re-encrypts under target folder's recipients)
gopass copy personal/old/secret personal/new/secret
gopass cp   personal/old/secret personal/new/secret
# Original remains; new file created

# Move a secret (rename)
gopass move personal/old/secret personal/new/secret
gopass mv   personal/old/secret personal/new/secret
# git rename semantics — preserves history under new path
```

```bash
# Copy across mounts
gopass copy personal/notes/x work/notes/x
# Re-encrypts with WORK recipients, not personal
```

```bash
# Force overwrite at destination
gopass copy -f source dest
gopass move -f source dest
```

```bash
# Copy a folder recursively
gopass copy personal/old-project/ personal/archived/old-project/
# Folders ending with / are treated as folders
```

## Deleting

```bash
# Delete a single secret
gopass rm personal/email/gmail
gopass delete personal/email/gmail
# Prompts for confirmation
```

```bash
# Force (no confirmation)
gopass rm -f personal/email/gmail
```

```bash
# Recursive (folder delete)
gopass rm -r personal/old-project/
gopass rm --recursive personal/old-project/

# Recursive + force (DANGEROUS)
gopass rm -rf personal/old-project/
gopass rm --force-recursive personal/old-project/
```

```bash
# Note: deleted secrets remain in git history!
# To purge from history (rare), use git filter-repo:
cd ~/.password-store
git filter-repo --path personal/email/gmail.gpg --invert-paths
git push --force                          # rewrite remote (coordinate with team!)
```

## Recipients

```bash
# Show recipients for the default store
gopass recipients show
# gopass
# └── <root>
#     ├── 0xDEADBEEFCAFEBABE - Alice Example <alice@example.com>
#     └── 0xCAFED00D12345678 - Bob Example   <bob@example.com>
```

```bash
# Show recipients for a specific mount
gopass recipients show --store work
gopass recipients show work
```

```bash
# Add a recipient (re-encrypts everything they should be able to read)
gopass recipients add 0xCAFED00D12345678
gopass recipients add bob@example.com         # by email if key is in keyring
gopass recipients add --store work bob@example.com
```

```bash
# Remove a recipient (re-encrypts to exclude them — they can still read PRIOR secrets in git history!)
gopass recipients remove 0xCAFED00D12345678
gopass recipients remove --store work bob@example.com
# CRITICAL: rotate any secrets they had access to AFTER removal
```

```bash
# Per-folder recipients (override .gpg-id)
gopass recipients add --store '' --path personal/private 0xDEADBEEFCAFEBABE
# Adds Alice as the ONLY recipient for ~/.password-store/personal/private/
# Subfolders inherit unless they also have a .gpg-id

cat ~/.password-store/personal/private/.gpg-id
# 0xDEADBEEFCAFEBABE
```

```bash
# .gpg-id propagation rules
# When encrypting personal/private/secret.gpg, gopass walks UP the
# directory tree and uses the FIRST .gpg-id it finds:
#   personal/private/.gpg-id   ← used (closest)
#   personal/.gpg-id           ← skipped
#   .gpg-id                    ← skipped
```

```bash
# Audit recipients vs actual encryption (drift detection)
gopass fsck                                # checks every secret can be decrypted by every recipient

# After adding a recipient, force re-encrypt everything
gopass recipients update                    # re-encrypts to current .gpg-id list
```

```bash
# Trusted GPG keys must be in your keyring
gpg --recv-keys 0xCAFED00D12345678
gpg --import bob-public.asc
gpg --edit-key 0xCAFED00D12345678 trust quit   # set ownertrust
```

## OTP Codes

```bash
# Generate the current OTP code for a secret containing an otpauth:// URI
gopass otp personal/email/gmail
# 539281
# valid for 22s

# Copy to clipboard
gopass otp -c personal/email/gmail
gopass otp --clip personal/email/gmail
```

```bash
# Insert an OTP secret — the otpauth:// URI from a QR code
gopass insert -m personal/email/gmail
# Type/paste the password line, blank, then:
# otpauth://totp/Gmail:alice@gmail.com?secret=JBSWY3DPEHPK3PXP&issuer=Gmail&period=30&digits=6&algorithm=SHA1
```

```bash
# Decode a QR code from a screenshot to extract otpauth://
zbarimg --raw qr.png
# otpauth://totp/Acme:alice@acme.com?secret=ONSWG4TFOQ====&issuer=Acme

# Then add it to the existing secret
gopass otp --qr=qr.png personal/acme            # some versions support direct QR import
```

```bash
# otpauth:// URI parameters that gopass parses
# otpauth://TYPE/LABEL?PARAMS
#   TYPE     = totp | hotp
#   LABEL    = issuer:account (e.g. Gmail:alice@gmail.com)
#   secret   = base32-encoded shared secret (REQUIRED)
#   issuer   = display name
#   algorithm= SHA1 (default) | SHA256 | SHA512
#   digits   = 6 (default) | 8
#   period   = 30 seconds (default, TOTP only)
#   counter  = HOTP counter (HOTP only)
```

```bash
# HOTP (counter-based) vs TOTP (time-based)
# TOTP: rotates every period seconds; gopass otp returns current code + remaining time
# HOTP: counter-based; gopass otp INCREMENTS the counter on each call (writes back)
gopass otp personal/work/hotp-token         # increments counter on each call
```

```bash
# Show the otpauth URI without the code (for migration)
gopass show personal/email/gmail otpauth
gopass show personal/email/gmail | grep otpauth
```

```bash
# Render the otpauth URI as a QR code (for re-pairing a phone)
gopass otp --qr=- personal/email/gmail | open -fa Preview      # macOS
gopass otp --qr=- personal/email/gmail | feh -                 # Linux
gopass otp --qr=qr.png personal/email/gmail                    # write to file
```

## Audit / Health

```bash
# Audit — checks for weak / duplicate / outdated passwords
gopass audit
# WARNING: personal/old/forgotten - last modified 2019-03-04 (5 years old)
# WARNING: personal/email/gmail   - duplicate password used by personal/email/yahoo
# WARNING: work/test/temp         - weak password (entropy 28 bits, recommended 80+)
```

```bash
# Audit in parallel (faster on big stores)
gopass audit --jobs 8
gopass audit -j 8
```

```bash
# Audit specific subtree
gopass audit personal/
```

```bash
# JSON output for piping into reporting tools
gopass audit --format json | jq '.findings'
```

```bash
# fsck — verify integrity of every secret
gopass fsck
# Walks every secret, verifies it parses correctly,
# checks recipients vs .gpg-id manifest

# Decrypt every secret as part of fsck (slow, exhaustive)
gopass fsck --decrypt

# Specific store
gopass fsck --store work

# Auto-fix recipient drift (re-encrypt to current .gpg-id)
gopass fsck --decrypt --fix
```

```bash
# health — overall store health summary
gopass health
# Store: gopass (default)
#   secrets:        427
#   recipients:     2 (Alice, Bob)
#   git remote:     git@github.com:alice/secrets.git
#   last sync:      2 hours ago
#   crypto backend: gpg
#   issues:         0
```

## Sync

```bash
# Sync — pull, then push, every store
gopass sync
# default behavior: git pull --rebase, then git push, every mount

# Specific store
gopass sync --store work
gopass sync work

# Specific remote (if multiple)
gopass sync --remote origin
```

```bash
# Auto-sync on every change
gopass config autosync true
# Now every gopass insert/edit/rm pushes immediately

# Disable auto-sync (you'll sync manually)
gopass config autosync false
GOPASS_NO_AUTOSYNC=1 gopass insert ...      # one-off
```

```bash
# Conflict resolution
# gopass uses git rebase by default. Conflict = manual:
cd ~/.password-store
git status
git mergetool                               # or edit .gpg files manually
git rebase --continue
gopass sync                                 # resume
```

```bash
# Push without pull (rare)
gopass git push --store work

# Pull without push
gopass git pull --store work
```

## Git Operations

```bash
# All gopass git commands operate on the underlying ~/.password-store git repo
gopass git status
gopass git log --oneline -20
gopass git pull
gopass git push
gopass git diff
```

```bash
# Per-store git (specify mount)
gopass git --store work status
gopass git --store work log -10
```

```bash
# Add / remove remote
gopass git remote add origin git@github.com:alice/secrets.git
gopass git remote add --store work origin git@gitlab.com:acme/secrets.git
gopass git remote remove origin
gopass git remote -v                        # list remotes
```

```bash
# Init git in an existing store that doesn't have it
gopass git init
gopass git init --store work
```

```bash
# Per-store git config
gopass git config user.name  "Alice Example"
gopass git config user.email "alice@example.com"
gopass git config --store work user.email "alice@acme.com"
```

```bash
# Auto-push behavior
# By default, after every insert/edit/generate/rm/move:
#   git add <changed> && git commit -m "<action>" && git push
# Disable with `gopass config autosync false`
# Or per-invocation: GOPASS_NO_AUTOSYNC=1 gopass insert ...
```

```bash
# View who changed what (full audit log)
cd ~/.password-store
git log --all --pretty='%h %an %s' --follow personal/email/gmail.gpg
```

## Templates

```bash
# Templates scaffold new secrets with a structure
# A .template file in a folder applies to all `gopass insert` operations
# under that folder (and gopass generate with --template)

# Example template at personal/email/.template
{{ .Pass | repeat 1 }}

url: {{ .Name | trimPrefix "personal/email/" | printf "https://%s.com" }}
user: {{ env "USER" }}@example.com
notes: |
  Created on {{ now | date "2006-01-02" }}
```

```bash
# List templates
gopass templates
gopass template ls

# Show a template
gopass template show personal/email
gopass template show -- personal/email

# Edit (creates if missing)
gopass template edit personal/email

# Remove
gopass template remove personal/email
gopass template rm personal/email
```

```bash
# Available template fields:
#   {{ .Pass }}        the password (generated or entered)
#   {{ .Name }}        the full secret path
#   {{ now }}          time.Time of insert
#   {{ env "VAR" }}    environment variable
# Plus all sprig functions: trim, replace, upper, lower, repeat, ...
```

```bash
# spec.tmpl format — older / alternate template name
~/.password-store/.gopass/templates/email.tmpl
# Same Go template syntax
```

```bash
# Apply a template to a one-off insert
gopass generate --template email personal/email/new-account 32
```

## Aliases

```bash
# Per-store alias registry — short forms for common commands
gopass alias add ll  "list -d 1"
gopass alias add cp  "copy"
gopass alias add cl  "show -c"
gopass alias add gen "generate -s"
```

```bash
# Use alias
gopass ll                                   # expands to: gopass list -d 1
gopass cl personal/email/gmail              # expands to: gopass show -c personal/email/gmail
gopass gen personal/new 32                  # expands to: gopass generate -s personal/new 32
```

```bash
# List aliases
gopass alias ls
gopass alias list

# Remove an alias
gopass alias remove ll
gopass alias rm ll
```

```bash
# Aliases are stored in gopass config — synced via git? NO, they're local
# Per-user, per-machine. Use shell aliases for portable shortcuts.
```

## gopass-jsonapi

```bash
# Install the JSON-API helper (browser bridge)
brew install gopass-jsonapi
# Or:
go install github.com/gopasspw/gopass-jsonapi@latest
```

```bash
# Configure native messaging host (one-time)
gopass-jsonapi configure
# Prompts: which browser? (chromium, firefox, brave, vivaldi, edge)
# Writes a JSON manifest into ~/Library/Application Support/Google/Chrome/NativeMessagingHosts/
# (or the equivalent dir for your browser)
```

```bash
# Install the gopass-bridge browser extension
# Chrome:  https://chrome.google.com/webstore/detail/gopass-bridge/...
# Firefox: https://addons.mozilla.org/en-US/firefox/addon/gopass-bridge/
# The extension talks to gopass-jsonapi over native messaging stdin/stdout
```

```bash
# Protocol — JSON over stdin/stdout (length-prefixed)
# Browser extension sends:
#   {"type": "query", "query": "github.com"}
# gopass-jsonapi responds:
#   {"results": ["personal/github/web", "work/github/enterprise"]}
# Browser extension sends:
#   {"type": "getLogin", "entry": "personal/github/web"}
# gopass-jsonapi responds:
#   {"username": "alice", "password": "hunter2"}
```

```bash
# Test the bridge manually
echo '{"type":"query","query":"gmail"}' | \
  gopass jsonapi listen
# Returns matching secret paths

# Listen mode (development)
gopass jsonapi listen
```

```bash
# Run jsonapi as a foreground daemon (alt to native messaging)
gopass jsonapi listen --socket /tmp/gopass.sock
```

## gopass-summon-provider

```bash
# Install
brew install gopass-summon-provider
# Or:
go install github.com/gopasspw/gopass-summon-provider@latest
```

```bash
# What it does — HashiCorp Summon plugin
# Summon launches subprocess with secrets injected as env vars
# !var:secret/path  in secrets.yml resolves via gopass-summon-provider
```

```bash
# Example secrets.yml
DB_PASSWORD: !var:work/db/prod-password
API_KEY:     !var:work/api/stripe-key
SECRET_TOKEN: !var:work/internal/token
```

```bash
# Run a command with secrets injected
summon -p gopass-summon-provider --yaml secrets.yml -- ./run-app
# DB_PASSWORD, API_KEY, SECRET_TOKEN populated in the subprocess env
# Cleared from RAM when subprocess exits
```

```bash
# Useful for:
#   * Local dev with prod secrets without a `.env` on disk
#   * CI runners with gopass installed (decrypt at run time)
#   * Replacing 12-factor `.env` files with encrypted source-controlled equivalents
```

## Config

```bash
# Show full config
gopass config

# Show specific key
gopass config show
gopass config get autoclip
```

```bash
# Set a value
gopass config autoclip      true            # auto-copy first-line on `gopass show`
gopass config autoimport    true            # auto-import unknown GPG keys from server
gopass config autosync      true            # auto-pull/push on every change
gopass config cliptimeout   45              # seconds before clipboard is cleared
gopass config exportkeys    true            # export public keys into store on recipient add
gopass config nopager       false           # disable auto-paging on large output
gopass config notifications true            # OS notifications (macOS / libnotify)
gopass config parsing       true            # parse YAML body for `show <path> <key>`
gopass config path          ~/.password-store
gopass config safecontent   true            # mask the password in `show` (show only freeform body)
```

```bash
# Per-store config — set with --store flag
gopass config --store work autosync false
```

```bash
# Per-mount overrides — ~/.password-store-work/.gopass.yml
cat <<'EOF' > ~/.password-store-work/.gopass.yml
core:
  autosync: false
  notifications: false
  cliptimeout: 30
EOF
```

```bash
# Default location of the global config
ls ~/.config/gopass/config.yml
cat ~/.config/gopass/config.yml
```

```bash
# Reset config to defaults
rm ~/.config/gopass/config.yml
gopass config                               # regenerates with defaults
```

## env Vars

```bash
# Suppress OS notifications (no popups when secrets are copied)
export GOPASS_NO_NOTIFY=1

# Disable auto-sync (no auto-push) for this shell
export GOPASS_NO_AUTOSYNC=1

# Verbose debugging — logs to stderr (NOT to a file)
export GOPASS_DEBUG=true
gopass show personal/email/gmail 2> gopass-debug.log

# Override the gopass home directory (config + state)
export GOPASS_HOMEDIR=/secure/disk/gopass
gopass setup                                # uses /secure/disk/gopass instead of ~/.config/gopass

# Path to a specific config file
export GOPASS_CONFIG=/etc/gopass/config.yml

# age master passphrase (UNATTENDED MODE — use with extreme care)
export GOPASS_AGE_PASSWORD='hunter2'
gopass show personal/secret                 # decrypts age secret without prompt

# $EDITOR — what `gopass edit` opens
export EDITOR='nvim'
export EDITOR='code --wait'
export EDITOR='hx'

# GPG agent settings (for pinentry-curses on TTY)
export GPG_TTY="$(tty)"
export GNUPGHOME=~/.gnupg
```

```bash
# Less-known env vars
export GOPASS_NO_REMINDER=1                 # disable update reminders
export GOPASS_NOPAGER=1                     # disable auto-pager
export GOPASS_UMASK=077                     # secret file permissions
export GOPASS_GPG_OPTS='--use-agent'        # extra args to gpg
export GOPASS_EXTERNAL_PWGEN='/usr/local/bin/pwgen -s 32 1'   # custom generator
```

```bash
# Unsafe env vars — disable warnings (dangerous)
export GOPASS_UNCLIP_CHECKSUM=1             # always overwrite clipboard
export GOPASS_FORCE=1                       # bypass confirmation prompts
```

## pass Compatibility

```bash
# gopass reads/writes the SAME ~/.password-store layout as pass(1)
# You can have BOTH installed and use them on the same store

pass insert email/gmail                     # bash pass(1)
gopass show email/gmail                     # gopass reads it correctly

gopass insert email/yahoo
pass show email/yahoo                       # bash pass(1) reads it correctly
```

```bash
# Differences gopass adds (still pass(1)-compatible):
#   * .gopass.yml per-store config — pass(1) ignores it
#   * .public-keys/ folder — pass(1) ignores it
#   * gopass commit messages have a different format
#   * gopass-templates → .gopass/templates/ folder pass(1) ignores
#   * Multi-mount layout — pass(1) only knows the default store
```

```bash
# .gpg-id propagation
# pass(1) and gopass both walk UP the tree to find .gpg-id
# Same semantics — closest .gpg-id wins
# Verified by `pass insert work/aws/key` and `gopass insert work/aws/key`
# producing identical recipient sets
```

```bash
# Hierarchy / addressing — IDENTICAL between pass and gopass
~/.password-store/personal/email/gmail.gpg → "personal/email/gmail"
```

```bash
# Migrating to gopass when you already have pass
which pass                                  # /usr/bin/pass
gopass setup                                # detects ~/.password-store, asks to adopt
gopass list                                 # see all your existing pass(1) secrets
# Now safe to `apt remove pass` if you want
```

## age Crypto Backend

```bash
# age vs gpg
# age:  modern, simple, X25519/ssh-ed25519 keys, no web of trust, no expiry
# gpg:  RFC 4880, 30+ years of features (smartcards, subkeys, signing, web of trust)
# Choose:
#   age  → solo / small team / no smartcard / want simplicity
#   gpg  → existing GPG infrastructure / Yubikey / corporate compliance
```

```bash
# Init a store with age
gopass init --crypto age --storage fs

# Generates an age identity at:
ls ~/.config/gopass/age/identities          # encrypted with passphrase
ls ~/.config/gopass/age/keyring             # public side
```

```bash
# .age-recipients file
cat ~/.password-store/.age-recipients
# age1abcdefghijklmnopqrstuvwxyz0123456789...
# ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... alice@laptop
# ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAA... bob@laptop
```

```bash
# Add a recipient (age public key)
gopass recipients add age1bobpublickeyxxxxxxxxxxxxxxxxxxxx
gopass recipients add 'ssh-ed25519 AAAAC3... bob'    # SSH keys also accepted!
```

```bash
# Generate a new age key
age-keygen -o ~/.config/gopass/age/identities
# Public key: age1qz...
# Add to .age-recipients of stores you want to read
```

```bash
# Migrate from gpg to age (in place)
# 1. Init a fresh store with age
gopass init --crypto age --storage fs --store age-store
# 2. Read each secret from gpg store, insert into age store
gopass ls --flat | while read p; do
  gopass show "$p" | gopass insert -m "age-store/$p"
done
# 3. Verify, then `gopass mounts remove` the old gpg store
```

```bash
# age + ssh keys — use your existing ~/.ssh/id_ed25519
# Add the PUBLIC key to .age-recipients
ssh-keygen -y -f ~/.ssh/id_ed25519 >> ~/.password-store/.age-recipients
# Add the PRIVATE key as an age identity
echo "AGE-IDENTITY-FILE: $HOME/.ssh/id_ed25519" > ~/.config/gopass/age/identities
# Now `gopass show` decrypts using your SSH key + ssh-agent
```

## GUI Front-Ends

```bash
# gopass-ui (Electron desktop app)
# https://github.com/codecentric/gopass-ui
# brew install --cask gopass-ui              # macOS
# Cross-platform; offers fuzzy search, copy-to-clipboard, edit
```

```bash
# gopass-bridge (browser extension — Firefox / Chrome)
# Pairs with gopass-jsonapi (see above)
# Autofills login forms, generates passwords, copies OTP
```

```bash
# QtPass — Qt-based GUI (works with both pass and gopass)
brew install qtpass
# Or: sudo apt install qtpass
qtpass &
# Settings → Use gopass instead of pass
```

```bash
# gopass + rofi (Linux keystroke launcher)
# https://github.com/cdown/passmenu (works with gopass)
cat <<'EOF' > ~/.local/bin/gopass-rofi
#!/usr/bin/env bash
SECRET=$(gopass ls --flat | rofi -dmenu -i -p 'Secret:')
[ -z "$SECRET" ] && exit 0
gopass show -c "$SECRET"
notify-send "gopass" "Copied $SECRET to clipboard"
EOF
chmod +x ~/.local/bin/gopass-rofi
# Bind to Mod+P in your WM
```

```bash
# dmenu / wofi (Wayland) launcher
gopass ls --flat | wofi --dmenu | xargs -r gopass show -c
```

```bash
# fzf TUI launcher
gopass ls --flat \
  | fzf --preview 'gopass show {} | head -1' --preview-window=hidden \
  | xargs -r gopass show -c
```

```bash
# tmux popup launcher
bind-key 'g' display-popup -E '
  gopass ls --flat | fzf | xargs -r gopass show -c
'
```

## TOFU / SSH Workflow

```bash
# Use gopass to store the passphrase for an SSH private key
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -C 'alice@laptop'
# When prompted for passphrase, generate one with gopass first:
gopass generate -c ssh/id_ed25519-passphrase 32
# Then paste from clipboard at the ssh-keygen passphrase prompt

# Store the passphrase as a secret (use --xkcd if you'd ever type it)
```

```bash
# Inject the SSH passphrase via ssh-add (without typing)
gopass show -o ssh/id_ed25519-passphrase | \
  SSH_ASKPASS_REQUIRE=force SSH_ASKPASS=$(mktemp) \
  ssh-add ~/.ssh/id_ed25519 < /dev/null
# More robust: a wrapper script that pipes the passphrase
```

```bash
# Cleaner: a helper script as $SSH_ASKPASS
cat <<'EOF' > ~/.local/bin/ssh-askpass-gopass
#!/usr/bin/env bash
gopass show -o "ssh/id_ed25519-passphrase"
EOF
chmod +x ~/.local/bin/ssh-askpass-gopass

# In your shell rc:
export SSH_ASKPASS=~/.local/bin/ssh-askpass-gopass
export SSH_ASKPASS_REQUIRE=force
export DISPLAY=:0                           # SSH_ASKPASS requires a "display"
ssh-add ~/.ssh/id_ed25519 < /dev/null       # passphrase auto-pulled from gopass
```

```bash
# gpg-agent + ssh-agent + gopass (the triad)
# Use gpg-agent as your ssh-agent — Yubikey / smartcard becomes SSH key

# In ~/.gnupg/gpg-agent.conf:
enable-ssh-support
default-cache-ttl 600
max-cache-ttl 7200

# In your shell rc:
export SSH_AUTH_SOCK="$(gpgconf --list-dirs agent-ssh-socket)"
gpgconf --launch gpg-agent

# Now `ssh` uses the GPG-on-Yubikey for auth
# gopass uses the SAME gpg-agent for decryption
# One PIN entry → unlocks both signing/auth + secret decryption
```

## Migration

```bash
# From pass(1) → gopass — zero migration; same store
which pass
gopass setup                                # detects existing ~/.password-store
gopass list                                 # all secrets visible
```

```bash
# From KeePass → gopass — via pass-import
pip install --user pass-import
ls ~/.local/bin/pimport                     # provided by pass-import
pimport gopass passwords.kdbx --pass-prefix imported-keepass
# Reads .kdbx, inserts into gopass under `imported-keepass/...`
```

```bash
# From 1Password → gopass
# 1. Export 1Password vault as 1pif: 1Password → File → Export
# 2. Use pass-import:
pimport gopass exported.1pif --pass-prefix imported-1password

# Or 1Password CLI + gopass:
op signin
op item list --format=json | jq -r '.[].id' | while read id; do
  data=$(op item get "$id" --format=json)
  title=$(echo "$data" | jq -r .title | tr -cd '[:alnum:]-')
  pwd=$(echo "$data" | jq -r '.fields[]|select(.id=="password").value')
  printf '%s\n' "$pwd" | gopass insert -m "1pw/$title"
done
```

```bash
# From LastPass → gopass — via lpass + pass-import
lpass login alice@example.com
lpass export > lastpass.csv
pimport gopass lastpass.csv --pass-prefix imported-lastpass
```

```bash
# From Bitwarden → gopass
bw login
bw unlock
bw export --format=json > bitwarden.json
pimport gopass bitwarden.json --pass-prefix imported-bitwarden
```

```bash
# From Chrome / Firefox saved passwords
# 1. Export from chrome://settings/passwords (Chrome) or about:logins (Firefox)
# 2. CSV → pimport
pimport gopass passwords.csv --pass-prefix imported-browser
```

```bash
# Generic CSV migration
# CSV columns: name,url,username,password,extra
while IFS=, read -r name url user pass extra; do
  printf '%s\nurl: %s\nuser: %s\nnotes: %s\n' "$pass" "$url" "$user" "$extra" \
    | gopass insert -m "imported/$name"
done < passwords.csv
```

## Common Errors

```bash
# "no GPG key found" / "Error: no recipients available"
# Cause: no .gpg-id file, or all listed keys missing from local keyring
# Fix:
gopass recipients show
gpg --list-secret-keys
# If your key isn't there:
gpg --import private-key.asc
# If .gpg-id is missing:
gopass init <your-key-id>
```

```bash
# "gpg: decryption failed: No secret key"
# Cause: the secret was encrypted to a key you don't have the private half of
# Fix:
gpg --list-secret-keys                      # confirm you have a key at all
gopass recipients show                      # see who CAN decrypt
gpg --import your-private-key.asc           # if the key is missing
# Or ask a current recipient to re-encrypt for you:
gopass recipients add <your-new-key-id>
```

```bash
# "Failed to find any private keys"
# Cause: GPG_HOME unset, or ~/.gnupg permissions wrong, or no secret keys
# Fix:
export GNUPGHOME=~/.gnupg
chmod 700 ~/.gnupg
chmod 600 ~/.gnupg/*
gpg --list-secret-keys                      # must show at least one key
```

```bash
# "ERROR: Cannot read template" / "template: ...: invalid"
# Cause: malformed Go template syntax in .template file
# Fix:
gopass template show <folder>               # see the offending template
# Common issues: missing }}, unbalanced {{ if }}{{ end }}, undefined .Field
gopass template edit <folder>               # fix in $EDITOR
```

```bash
# "Error: store '<name>' not initialized"
# Cause: tried `gopass insert work/...` but `work` mount doesn't exist
# Fix:
gopass mounts list
gopass mounts add work /path/to/work-store
gopass init --store work <key-id>
```

```bash
# "Failed to push: rejected" (git remote diverged)
# Cause: someone else pushed before you; your local is behind
# Fix:
gopass git pull --rebase
# Resolve conflicts (rare on .gpg files since each secret is a distinct file)
gopass git push
gopass sync                                 # convenience for pull+push
```

```bash
# "Failed to lock secret store" / "another gopass instance is running"
# Cause: stale lock file, or actual concurrent gopass process
# Fix:
ps aux | grep gopass                        # check for hung process
ls ~/.password-store/.gopass.lock           # the lock file
rm ~/.password-store/.gopass.lock           # if no process owns it
```

```bash
# "Aborted: ..." (interactive prompt declined)
# Cause: you (or a script) said no/n at a confirmation prompt
# Fix:
# To bypass prompts in scripts, use -f or GOPASS_FORCE=1:
gopass rm -f path
GOPASS_FORCE=1 gopass recipients remove <key-id>
```

```bash
# "remote sync failed: not connected to internet"
# Cause: auto-sync triggered but no network
# Fix: gopass continues; the change is committed locally
# Sync later when online:
gopass sync
```

```bash
# "gpg: agent_genkey failed: No pinentry"
# Cause: missing pinentry program
# Fix:
brew install pinentry-mac                   # macOS
sudo apt install pinentry-curses            # Linux TTY
# In ~/.gnupg/gpg-agent.conf:
echo 'pinentry-program /opt/homebrew/bin/pinentry-mac' >> ~/.gnupg/gpg-agent.conf
gpgconf --kill gpg-agent                    # restart agent
```

```bash
# "inappropriate ioctl for device" (gpg on a script over ssh)
# Cause: GPG_TTY not set when running non-interactively
# Fix:
export GPG_TTY="$(tty)"
# Or in cron/systemd: explicitly set GPG_TTY=/dev/null and use --batch
```

```bash
# "Error: secret '...' not found"
# Cause: typo, or wrong store/mount, or secret was deleted
# Fix:
gopass find <substring>
gopass ls --flat | grep -i <pattern>
gopass mounts list                          # check you're in the right mount
```

```bash
# "skipped 1 invalid recipient" warning
# Cause: a key in .gpg-id is expired / revoked / no longer in keyring
# Fix:
gopass recipients show                      # see the warning detail
gpg --list-keys                             # check expiry
gopass recipients remove <expired-key-id>
gopass recipients update                    # re-encrypt
```

```bash
# "warning: trying to set option without value"
# Cause: gopass config <key> with no value
# Fix:
gopass config <key> <value>                 # provide the value
gopass config show                          # verify
```

## Common Gotchas

```bash
# GOTCHA 1: insert without -m loses multi-line content
# Broken:
echo -e "pwd\n\nuser: alice" | gopass insert path/to/secret
# Inserts ONLY "pwd" (single-line mode reads first line only)

# Fixed:
echo -e "pwd\n\nuser: alice" | gopass insert -m path/to/secret
# -m for multi-line preserves the blank line + freeform body
```

```bash
# GOTCHA 2: forgetting to add a recipient before pushing
# Broken:
gopass insert team/secret                   # encrypted to OLD recipients
git push
# Bob (newly-added recipient) still can't decrypt — he wasn't a recipient when this was encrypted

# Fixed:
gopass recipients add bob@example.com       # adds Bob AND re-encrypts ALL secrets
gopass sync
```

```bash
# GOTCHA 3: cliptimeout = 0 disables auto-clear
# Broken:
gopass config cliptimeout 0
gopass show -c personal/secret              # password sits in clipboard forever
# 0 means "no timeout", not "instant clear"

# Fixed:
gopass config cliptimeout 45                # default; clears after 45s
gopass show -c personal/secret
```

```bash
# GOTCHA 4: git remote misconfig — auto-push silently failing
# Broken:
gopass git remote -v
# (empty)
gopass insert personal/secret               # commits locally, "push" silently no-ops
# You think it synced; it didn't

# Fixed:
gopass git remote add origin git@github.com:alice/secrets.git
gopass git push -u origin main              # initial push
gopass sync                                 # verify
```

```bash
# GOTCHA 5: GPG_TTY unset breaking pinentry on TTY
# Broken (over SSH or in tmux):
gopass show personal/secret
# error: gpg: agent_genkey failed: Inappropriate ioctl for device
# pinentry can't find the TTY

# Fixed (add to ~/.bashrc or ~/.zshrc):
export GPG_TTY="$(tty)"
```

```bash
# GOTCHA 6: same path in two stores creates ambiguity
# Broken:
gopass mounts list
# <root>  ~/.password-store
# work    ~/.password-store-work
gopass insert email/gmail                   # ambiguous — root or work?
# gopass picks <root> by default; you may meant work

# Fixed:
gopass insert work/email/gmail              # explicit mount prefix
gopass insert <root>/email/gmail            # explicit root prefix (some versions)
gopass --store work insert email/gmail      # explicit --store flag
```

```bash
# GOTCHA 7: .gitignore not including build artifacts (custom hooks)
# Broken:
ls ~/.password-store/.gitignore
# (missing)
# Some users add scripts / temp files in the store; they get committed!

# Fixed:
cat > ~/.password-store/.gitignore <<'EOF'
*.swp
*.tmp
*.log
.DS_Store
node_modules/
EOF
gopass git add .gitignore
gopass git commit -m 'Add gitignore'
gopass sync
```

```bash
# GOTCHA 8: deleting a recipient does NOT remove their access to git history
# Broken:
gopass recipients remove evil-bob@example.com
gopass sync                                 # everyone re-encrypted, evil-bob can't decrypt new secrets
# But evil-bob has the OLD .gpg files in his clone — he can still decrypt those!

# Fixed:
# Treat evil-bob's removal as a key compromise:
gopass ls --flat | while read p; do
  pwd=$(gopass show -o "$p")                # current password
  gopass generate -f "$p" 32                # rotate every secret
done
# Force-push and have all OTHER recipients re-clone
```

```bash
# GOTCHA 9: pasting a multi-line password into single-line `insert`
# Broken:
gopass insert personal/secret
# Enter password: <PASTE>     # paste contains a newline mid-password
# Retype password: <PASTE>    # the second half lands in retype prompt
# error: passwords do not match

# Fixed:
gopass insert -m personal/secret            # multi-line mode tolerates newlines
# Or:
echo -n 'pasted password with newline...' | gopass insert -m personal/secret
```

```bash
# GOTCHA 10: $EDITOR must be a foreground command
# Broken:
EDITOR='code' gopass edit secret
# code launches asynchronously; gopass thinks edit is done immediately

# Fixed:
EDITOR='code --wait' gopass edit secret     # --wait keeps gopass blocked
EDITOR='subl --wait' gopass edit secret
EDITOR='nvim'        gopass edit secret     # vim/nvim/nano are foreground naturally
```

```bash
# GOTCHA 11: recipients update without sync = local-only re-encryption
# Broken:
gopass recipients add new-key@example.com
# Local files re-encrypted, but never pushed
# Other clones still have the old encryption

# Fixed:
gopass recipients add new-key@example.com
gopass sync                                 # explicitly push the re-encryption
```

```bash
# GOTCHA 12: piping `gopass show` through `cat` exposes the body
# Broken:
gopass show personal/secret | cat
# cat doesn't suppress; the entire body lands on stdout
# If your terminal scrollback is logged, the secret is logged

# Fixed:
gopass show -c personal/secret              # clipboard, not stdout
# Or for scripts:
gopass show -o personal/secret | program-stdin --no-log
```

## Threat Model

```bash
# What gopass PROTECTS
#   * Secrets at rest on disk (encrypted)
#   * Secrets at rest in git history (encrypted)
#   * Secrets in transit to git remote (encrypted; remote sees ciphertext only)
#   * Secrets after laptop theft (assuming GPG/age private key is also encrypted)
#   * Recipient management (only listed keys can decrypt new commits)
```

```bash
# What gopass does NOT protect
#   * Secrets in CLIPBOARD — any process can read it
#   * Secrets in $EDITOR — vim swap files, undo files, .swp leak plaintext
#   * Secrets in TERMINAL SCROLLBACK — `gopass show` dumps to stdout
#   * Secrets in PROCESS RAM — gopass holds plaintext during decrypt
#   * Secrets in $HISTORY — `gopass show -o secret > /etc/...` lands in shell history
#   * Secrets after recipient removal — old git history still decryptable by removed key
#   * GPG/age private key compromise — game over for everything they could read
#   * Keylogger / shoulder surfer — gopass shows plaintext on the screen
```

```bash
# Hardening recommendations
#   1. Use a smartcard / Yubikey for the GPG private key
#      gpg --card-edit  → key generation on-device
#      Now decryption requires PIN + physical touch
#   2. Set GPG cache TTL low (force re-PIN frequently)
#      ~/.gnupg/gpg-agent.conf:
#        default-cache-ttl 60
#        max-cache-ttl 300
#   3. Use age + SSH ed25519 keys for simplicity (no GPG)
#   4. Mount ~/.password-store on encrypted disk (LUKS / FileVault / BitLocker)
#   5. Disable swap, or ensure encrypted swap (zram / plain dm-crypt swap)
#   6. Disable shell history for gopass commands:
#      alias gopass='HISTIGNORE=":gopass *" gopass'
#   7. Use `gopass show -c` (clipboard) over `gopass show` (stdout) wherever possible
#   8. Set cliptimeout aggressively low: `gopass config cliptimeout 15`
#   9. Run `gopass audit` weekly; rotate weak / old / duplicated passwords
#  10. Keep recipient list minimal; per-folder .gpg-id for sensitive subtrees
#  11. Monitor `gopass git log` for unexpected commits (audit log analysis)
#  12. For multi-team stores, use SEPARATE mounts (not one giant store)
```

```bash
# Smartcard / Yubikey setup
gpg --card-edit
# admin
# generate         (on-card key generation; secret never leaves the device)
# Save the public key
gpg --card-status
gopass init <new-card-key-id>
# Now `gopass show` blocks until the Yubikey is touched
```

```bash
# In-memory plaintext exposure
# When `gopass show secret`:
#   1. gpg/age decrypts the file in RAM
#   2. gopass holds the plaintext in a Go string
#   3. Go GC may not zero memory before reuse
# Mitigation: use mlock'd memory? gopass doesn't currently. For paranoia,
# wipe RAM after use:
gopass show -c secret                       # clipboard ages out
sudo sync && sudo sysctl vm.drop_caches=3   # flush page cache
```

## Idioms

```bash
# Idiom: pipe gopass output into another tool's stdin
# GitHub auth
gopass show -o git/gh-token | gh auth login --with-token

# Docker registry
gopass show -o docker/registry-pwd | \
  docker login --username alice --password-stdin registry.example.com

# AWS access key
gopass show work/aws/prod-access-key | aws configure import --csv -

# K8s secret creation
kubectl create secret generic db-creds \
  --from-literal=password="$(gopass show -o k8s/db-pwd)"

# Curl with bearer
curl -H "Authorization: Bearer $(gopass show -o api/stripe)" https://api.stripe.com/v1/...
```

```bash
# Idiom: gopass as ENV-var source for one command
DB_PASSWORD=$(gopass show -o work/db/prod) ./run-app
# Or use direnv:
cat <<'EOF' >> .envrc
export DB_PASSWORD=$(gopass show -o work/db/prod)
export API_KEY=$(gopass show -o work/api/stripe)
EOF
direnv allow
```

```bash
# Idiom: shell function to grab any secret
secret() { gopass show -o "$@"; }
# Usage:
curl -u "alice:$(secret personal/email/gmail)" https://imap.gmail.com
```

```bash
# Idiom: gopass-summon-provider for env-var injection (HashiCorp Summon)
# secrets.yml:
#   DB_PASSWORD: !var:work/db/prod-pwd
#   API_KEY:     !var:work/api/stripe
summon -p gopass-summon-provider --yaml secrets.yml -- ./run-app
# Subprocess sees DB_PASSWORD and API_KEY in env; they vanish on exit
```

```bash
# Idiom: team-shared store via shared git remote
# 1. Create a private GitHub/GitLab repo (e.g. acme/team-secrets)
# 2. Each developer:
gopass clone git@github.com:acme/team-secrets.git work
# 3. Lead adds each new dev's GPG key:
gopass recipients add --store work bob@acme.com
gopass sync --store work
# 4. New dev pulls, can now decrypt:
gopass sync --store work
gopass show work/db/prod
```

```bash
# Idiom: per-developer GPG keys (no shared keys ever)
# Each dev generates their own GPG key:
gpg --full-generate-key
gpg --export --armor alice@acme.com > alice.asc
# Send alice.asc to the team lead via secure channel
# Team lead imports + adds:
gpg --import alice.asc
gopass recipients add --store work alice@acme.com
gopass sync
```

```bash
# Idiom: rotate a secret + record old in history
old=$(gopass show -o personal/email/gmail)
gopass generate -f personal/email/gmail 32  # generate new
new=$(gopass show -o personal/email/gmail)
# Both old and new are in git history; history with --revision can recover
```

```bash
# Idiom: scripted bulk insert
while IFS=$'\t' read -r path pwd; do
  printf '%s\n' "$pwd" | gopass insert -m "$path"
done < secrets.tsv
```

```bash
# Idiom: bulk audit / health
gopass audit --jobs 8 --format json | jq '.findings[] | select(.severity=="high")'
gopass fsck --decrypt --jobs 8
```

```bash
# Idiom: backup the entire store as a tarball
tar -czf gopass-backup-$(date +%F).tar.gz \
  -C "$HOME" .password-store .gnupg \
  && age -e -p -o gopass-backup-$(date +%F).tar.gz.age gopass-backup-$(date +%F).tar.gz \
  && shred -u gopass-backup-$(date +%F).tar.gz
# Encrypted backup; decrypt with `age -d -o backup.tgz backup.tgz.age`
```

```bash
# Idiom: cron job — daily sync
crontab -l 2>/dev/null | { cat; echo '15 9 * * * gopass sync >> ~/.cache/gopass-sync.log 2>&1'; } | crontab -
# Daily 9:15am pull/push for any remote-side changes
```

```bash
# Idiom: emergency offline cleanup
# Stash gopass / pull from a key compromise:
mv ~/.password-store ~/.password-store.compromised.$(date +%s)
mv ~/.gnupg ~/.gnupg.compromised.$(date +%s)
# Generate new GPG key + re-init store from a known-clean backup
gpg --full-generate-key
gopass init <new-key-id>
# Restore from backup, re-encrypt to new key, push
```

## See Also

- gpg
- age
- sops
- vault
- ssh

## References

- gopass.pw — official site
- github.com/gopasspw/gopass — source, releases, issues
- github.com/gopasspw/gopass/blob/master/docs/commands/ — full command reference
- github.com/gopasspw/gopass-jsonapi — browser bridge / JSON-API
- github.com/gopasspw/gopass-summon-provider — HashiCorp Summon plugin
- github.com/gopasspw/gopass-bridge — browser extension source
- www.passwordstore.org — pass(1), the original "standard unix password manager"
- age-encryption.org — age crypto, the modern alternative to GPG
- github.com/FiloSottile/age — age reference implementation
- gnupg.org — GPG / GnuPG, the OpenPGP reference implementation
- pass-import (github.com/roddhjav/pass-import) — universal password manager importer
- summon (github.com/cyberark/summon) — env-var secret injection tool
- RFC 4226 — HOTP: HMAC-Based One-Time Password
- RFC 6238 — TOTP: Time-Based One-Time Password
- RFC 4880 — OpenPGP Message Format
- RFC 7748 — Elliptic Curves for Security (X25519, used by age)
