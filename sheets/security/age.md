# age

Modern, simple, secure file encryption — small attack surface, X25519 + ChaCha20-Poly1305 + scrypt, no PGP cruft.

## Setup

```bash
# macOS — Homebrew
brew install age

# Debian / Ubuntu (12+, 22.04+)
sudo apt update
sudo apt install age

# Fedora
sudo dnf install age

# Arch / Manjaro
sudo pacman -S age

# Alpine
apk add age

# openSUSE
sudo zypper install age

# FreeBSD
pkg install age

# OpenBSD
pkg_add age

# From source — Go reference implementation
go install filippo.io/age/cmd/...@latest
# Installs both `age` and `age-keygen` into $GOBIN (default ~/go/bin)

# Specific version pinning
go install filippo.io/age/cmd/...@v1.2.0

# Rust port — rage (fully compatible)
cargo install rage
# Installs `rage`, `rage-keygen`, `rage-mount`

# Static binary (no Go toolchain) — GitHub releases
curl -LO https://github.com/FiloSottile/age/releases/download/v1.2.0/age-v1.2.0-linux-amd64.tar.gz
tar -xzf age-v1.2.0-linux-amd64.tar.gz
sudo install age/age age/age-keygen /usr/local/bin/

# Verify
age --version            # 1.2.0
age-keygen --version
rage --version           # only if rage installed

# Shell completion (Go age)
age completion bash > /etc/bash_completion.d/age
age completion zsh  > "${fpath[1]}/_age"
age completion fish > ~/.config/fish/completions/age.fish
```

### Version differences

| Version  | Date     | Notable                                                        |
|----------|----------|----------------------------------------------------------------|
| v1.0.0   | 2021-12  | First stable spec freeze; format `age-encryption.org/v1`       |
| v1.1.0   | 2023-08  | Plugin protocol stabilised; `--decrypt` errors clarified       |
| v1.1.1   | 2023-12  | Security release; bumped Go minimum                            |
| v1.2.0   | 2024-09  | `age-keygen -y` reads from stdin; performance improvements     |
| v1.2.1   | 2025-04  | Bugfixes, additional plugin protocol robustness                |

```bash
# Where age lives — distinguish flavours
which age                      # /opt/homebrew/bin/age (brew) or ~/go/bin/age
age --version | head -1        # Go: "1.2.0"; rage: "rage 0.10.0"
file $(which age)              # Mach-O / ELF; both binaries are tiny (<6 MB)
```

## Why age

age is by Filippo Valsorda (former Go security lead), shipped in 2019, frozen at v1 in 2021. Design goals:

- **Tiny.** ~3,000 LoC of Go for the reference; auditable in an afternoon.
- **Modern primitives only.** X25519 (asymmetric), ChaCha20-Poly1305 (AEAD), HKDF-SHA-256, scrypt (passphrase KDF). No RSA, no DSA, no MD5, no CAST5, no IDEA.
- **No options.** No cipher selection, no algorithm negotiation, no compression, no signing — every age file is encrypted the same way.
- **Encryption only.** Signing is a separate concern (`minisign`, `signify`, `ssh-keygen -Y sign`).
- **No keyring.** Recipients are public strings. Identities are files. No `~/.gnupg` mess.
- **SSH key support.** Reuse `~/.ssh/id_ed25519` / `~/.ssh/id_rsa` directly; teams can fan-out via `authorized_keys`.
- **Plugin protocol.** Hardware tokens (YubiKey, TPM, FIDO2, Secure Enclave) via separate binaries.

```bash
# The aesthetic in one line
age-keygen -o key.txt && age -e -i key.txt < secret > secret.age
```

vs gpg (for the same task):

```bash
# gpg path of pain
gpg --quick-generate-key 'me@example.com' default default 1y
gpg --list-keys
gpg -e -r me@example.com secret             # writes secret.gpg
gpg -d secret.gpg                            # uses default keyring
```

`age` ships *one* mode; `gpg` ships dozens of cipher/digest/compression toggles, MDC vs SEIPDv2, OpenPGP vs PGP/MIME, web-of-trust signatures, key servers, pinentry — almost none of which most people need. age trades flexibility for auditability.

## Format Spec

```
age-encryption.org/v1
-> X25519 SflpOoFkHtxYkR04u6JhzgDp3uSmTKMPV0kn05j+8Tg
QyZ5n6DJ7HKGwXFtLrHhcdkHL98BXz0bj+rE8m5Ye80
-> X25519 ttp3RcAfH0NcW3hF7v4cXk0wCydNL0lcgDFRSp7XHnA
n8yStb2D1+9GlCzS+5G73oo36G+3UTyT8FugDAaNuAU
--- ZQ8X9CmLkQTyV+t4XXdvUM5cIXBLbJrMaPqL7n/D6dY
<encrypted payload — ChaCha20-Poly1305 over 64-KiB chunks>
```

- Line 1 — magic header `age-encryption.org/v1` (LF-terminated). No leading BOM.
- Recipient stanzas — one per recipient, beginning `-> <type> <args>` then base64-no-padding body. `\n` indicates end of stanza body when a short line (<64 chars) appears.
- HMAC-SHA-256 line `--- <base64>` over `header[:hmac]` (the magic line, all stanzas, and the trailing `---`). Key is HKDF(file-key, "header").
- Payload — encrypted with ChaCha20-Poly1305, key = HKDF(file-key, nonce, "payload"), 16-byte random nonce. Payload split into 64-KiB chunks; each chunk has a 16-byte Poly1305 tag, last chunk has its high bit set in counter.
- ARMOR — strict PEM-style: `-----BEGIN AGE ENCRYPTED FILE-----` ... base64 lines (64 cols) ... `-----END AGE ENCRYPTED FILE-----`.

```bash
# Inspect a binary age file
xxd test.age | head -10
# 00000000: 6167 652d 656e 6372 7970 7469 6f6e 2e6f  age-encryption.o
# 00000010: 7267 2f76 310a 2d3e 2058 3235 3531 3920  rg/v1.-> X25519

# Inspect ARMOR
head -1 test.age.asc
# -----BEGIN AGE ENCRYPTED FILE-----
tail -1 test.age.asc
# -----END AGE ENCRYPTED FILE-----

# Count recipients in a file (each `-> ` line is one)
grep -c '^-> ' test.age
```

The format **does not seek** — payload is a stream. You cannot decrypt the middle of a 1 GiB file without decrypting the prefix.

## Recipient Types

| Type            | Recipient prefix       | Identity prefix             | Plugin?       |
|-----------------|------------------------|-----------------------------|---------------|
| X25519 native   | `age1...` (Bech32)     | `AGE-SECRET-KEY-1...`       | No            |
| scrypt          | n/a (passphrase only)  | n/a                         | No            |
| ssh-rsa         | `ssh-rsa AAAA...`      | `~/.ssh/id_rsa`             | No            |
| ssh-ed25519     | `ssh-ed25519 AAAA...`  | `~/.ssh/id_ed25519`         | No            |
| YubiKey (PIV)   | `age1yubikey1...`      | `AGE-PLUGIN-YUBIKEY-1...`   | yes           |
| TPM             | `age1tpm1...`          | `AGE-PLUGIN-TPM-1...`       | yes           |
| FIDO2-HMAC      | `age1fido2hmac1...`    | `AGE-PLUGIN-FIDO2-HMAC-1`   | yes           |
| Secure Enclave  | `age1se1...`           | `AGE-PLUGIN-SE-1...`        | yes (macOS)   |

```bash
# X25519 — 62-character Bech32 recipient
age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p

# X25519 secret — 74-char Bech32 with prefix AGE-SECRET-KEY-1
AGE-SECRET-KEY-1QXEAQ52AKR8ZYE65XTRF6QDD6WX8FQTSGW5ME5HQHYPN7TT8EMHQXQHGN8

# SSH recipient (one line of authorized_keys)
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBhpV5...
ssh-rsa     AAAAB3NzaC1yc2EAAAADAQABAAABAQ...
```

### Threat model — SSH keys as encryption keys

Reusing an SSH key for age means the same private material now decrypts files **and** authenticates SSH sessions. Implications:

- A leaked SSH key now leaks past correspondence too.
- Hardware-backed SSH keys (yubikey-agent, ssh-sk) can NOT be used directly with `age -i` — use `age-plugin-yubikey` instead.
- ssh-rsa requires a 2048+ bit key; age refuses smaller.
- ssh-ed25519 is preferred (smaller stanzas, faster, simpler).
- agent forwarding does NOT extend to age — age reads the raw key file.

## age-keygen

```bash
# Generate a fresh X25519 identity to stdout
age-keygen
# # created: 2024-04-01T08:30:00Z
# # public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
# AGE-SECRET-KEY-1QXEAQ52AKR8ZYE65XTRF6QDD6WX8FQTSGW5ME5HQHYPN7TT8EMHQXQHGN8

# Write to a file (mode auto-set to 0600)
age-keygen -o ~/.config/age/key.txt
# Public key: age1...
ls -l ~/.config/age/key.txt
# -rw------- 1 me me 189 Apr  1 08:30 /home/me/.config/age/key.txt

# Derive the public key from an existing identity
age-keygen -y ~/.config/age/key.txt
# age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p

# v1.2.0+ — read identity from stdin
cat ~/.config/age/key.txt | age-keygen -y -
# age1...

# Convert an SSH key to an age identity (NOT supported in upstream — common confusion)
# age-keygen does NOT have a --convert flag.
# Instead: pass the SSH key directly to age -i / age -R.
# Some out-of-tree wrappers (age-keygen-rs etc.) do offer conversion.
```

### Generated key file format

```
# created: 2024-04-01T08:30:00Z
# public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
AGE-SECRET-KEY-1QXEAQ52AKR8ZYE65XTRF6QDD6WX8FQTSGW5ME5HQHYPN7TT8EMHQXQHGN8
```

- Two `#`-prefixed comment lines (timestamp, public key — for human reference).
- Single non-comment line — the secret key, Bech32-encoded `AGE-SECRET-KEY-1...`.
- File permissions default to `0600`. On creation `age-keygen -o` aborts if the target exists (no clobber).
- Multiple identities per file are allowed — just append. age tries each until one decrypts.

```bash
# Multi-identity file
{
  age-keygen
  age-keygen
} > keys.txt

# Verify both pubkeys
grep '^# public key:' keys.txt

# Permissions sanity
chmod 600 keys.txt
stat -c '%a %n' keys.txt           # 600 keys.txt    (Linux)
stat -f '%Lp %N' keys.txt          # 600 keys.txt    (macOS)
```

### Default identity locations

age does not look for default identity files automatically; you MUST pass `-i`. Exception: when decrypting a file with SSH recipient stanzas and no `-i` is given, age will try `~/.ssh/id_ed25519` and `~/.ssh/id_rsa` in that order.

```bash
# Personal convention — where to put it
mkdir -m 700 -p ~/.config/age
age-keygen -o ~/.config/age/key.txt

# Then alias for ergonomics
alias enc='age -e -i ~/.config/age/key.txt'
alias dec='age -d -i ~/.config/age/key.txt'
```

## Encrypting

```bash
# Single recipient
age -e -r age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p \
  -o secret.age secret.txt

# Multiple -r (encrypt-once-decrypt-by-anyone)
age -e \
  -r age1abc... \
  -r age1xyz... \
  -r ssh-ed25519\ AAAAC3...\ alice@host \
  -o secret.age secret.txt

# From a recipients file (one recipient per line; # starts a comment)
age -e -R recipients.txt -o secret.age secret.txt

# Mixed flags
age -e -R team.txt -r age1emergency... -o secret.age secret.txt

# Passphrase mode (interactive only)
age -e -p -o secret.age secret.txt
# Enter passphrase (leave empty to autogenerate a secure one):
# Confirm passphrase:

# Identity-based recipient encryption
# A "recipient" derived from your own identity — useful for symmetric-style flows
age -e -i ~/.config/age/key.txt -o backup.age backup.tar
# This means: take the public-key pair of the identity, encrypt to it.
# Anyone WITH the identity file can decrypt; anyone with only the public key cannot decrypt
# without also having the identity (since the recipient is derived).

# ARMOR (ASCII-only) output
age -e -a -r age1... -o secret.age.asc secret.txt
cat secret.age.asc
# -----BEGIN AGE ENCRYPTED FILE-----
# YWdlLWVuY3J5cHRpb24ub3JnL3Yx...
# -----END AGE ENCRYPTED FILE-----

# Stdin → stdout (the canonical pipe form)
echo 'hello' | age -e -r age1... > hi.age
tar c data/ | age -e -R recipients.txt > backup.tar.age
```

### All `age -e` flags

| Flag                    | Purpose                                                  |
|-------------------------|----------------------------------------------------------|
| `-e, --encrypt`         | Encrypt mode (default if `-d` not specified)             |
| `-d, --decrypt`         | Decrypt mode                                             |
| `-o, --output FILE`     | Write to FILE instead of stdout                          |
| `-r, --recipient X`     | Add recipient X (may repeat)                             |
| `-R, --recipients-file` | File of recipients, one per line                         |
| `-i, --identity FILE`   | Identity file (may repeat); used as recipient on encrypt |
| `-p, --passphrase`      | Encrypt with a passphrase (interactive)                  |
| `-a, --armor`           | ARMOR (PEM-like ASCII) output                            |
| `-j, --plugin-name X`   | Force plugin lookup name (rare)                          |
| `--version`             | Print version and exit                                   |
| `-h, --help`            | Help                                                     |

## Decrypting

```bash
# Identity file
age -d -i ~/.config/age/key.txt secret.age > secret.txt
age -d -i ~/.config/age/key.txt -o secret.txt secret.age

# Stdin → stdout
age -d -i ~/.config/age/key.txt < secret.age > secret.txt

# Multiple identities — first one that matches a stanza wins
age -d -i alice.txt -i bob.txt -i shared.txt secret.age

# SSH default identity (no -i; only works when file has SSH recipients)
age -d secret.age
# tries ~/.ssh/id_ed25519 then ~/.ssh/id_rsa

# Passphrase decrypt
age -d secret.age
# Enter passphrase:

# ARMOR auto-detected — no flag needed on decrypt
age -d -i key.txt secret.age.asc

# rage variant — same flags
rage -d -i key.txt secret.age
```

### Identity precedence

1. Each `-i` file in order (multi-identity files iterate stanza-by-stanza).
2. If no `-i` given AND the file has SSH stanzas — `~/.ssh/id_ed25519`, `~/.ssh/id_rsa`.
3. If no `-i` AND the file has scrypt stanza — prompt for passphrase.
4. Otherwise — error: "no identity matched any of the recipients".

```bash
# Decrypt where you don't know which key works
for k in ~/keys/*.txt; do
  age -d -i "$k" secret.age 2>/dev/null && break
done

# Decrypt with passphrase-protected identity (manual two-step)
age -d -p < key.txt.age > /tmp/key && age -d -i /tmp/key secret.age
shred -u /tmp/key
```

## Working with SSH Keys

```bash
# Encrypt to your own SSH key
age -e -R ~/.ssh/id_ed25519.pub -o secret.age secret.txt

# Encrypt to a colleague's authorized_keys file (fan-out)
age -e -R /home/alice/.ssh/authorized_keys -o secret.age secret.txt

# Encrypt to a GitHub user's SSH keys
curl -s https://github.com/torvalds.keys > linus.keys
age -e -R linus.keys -o secret.age secret.txt
# torvalds.keys returns each public SSH key on a line — multiple recipients in one shot

# Decrypt with default SSH identity (no -i)
age -d secret.age > secret.txt

# Decrypt with a non-default SSH key
age -d -i ~/.ssh/id_ed25519_work secret.age

# Encrypted SSH key — age will prompt for the passphrase
age -d -i ~/.ssh/id_ed25519 secret.age
# Enter passphrase for "/home/me/.ssh/id_ed25519":
```

### Why this is dangerous

```
SSH key   ──┬── authenticates SSH login
            └── decrypts archived files (with age)
```

- Compromise of an SSH key now compromises every age file ever encrypted to it.
- Rotating an SSH key requires re-encrypting all archived files (`age -d` then `age -e -R newkeys`).
- For high-stakes secrets, use a dedicated `AGE-SECRET-KEY-1...` identity, not SSH.

```bash
# Convention: keep SSH-recipient files where they're easy to refresh
mkdir -p ~/.config/age/team
curl -s https://github.com/alice.keys > ~/.config/age/team/alice.keys
curl -s https://github.com/bob.keys   > ~/.config/age/team/bob.keys

# A recipients file for the whole team
cat ~/.config/age/team/*.keys > ~/.config/age/team/all.txt
age -e -R ~/.config/age/team/all.txt -o secret.age secret.txt
```

## Recipient Files / Multiple Recipients

```bash
# Format — one recipient per line, # for comments, blank lines OK
cat > recipients.txt <<'EOF'
# Production keys — January 2024 rotation
age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p   # alice
age1lggyhqrw2nlhcxprm67z43rta597azn8gknawjehu9d9dl0jq3yqqvfafg   # bob
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBhpV5... carol@laptop

# Offsite emergency key — physical paper backup, sealed
age1emergencyabcdefghijklmnopqrstuvwxyz0123456789abcdefghijkl
EOF

# Encrypt to all of them
age -e -R recipients.txt -o secret.age secret.txt

# Inspect how many recipient stanzas the file got
grep -c '^-> ' secret.age            # 4

# Combine sources
age -e \
  -R prod-keys.txt \
  -R staging-keys.txt \
  -r age1emergency... \
  -o secret.age secret.txt

# rotate-recipients pattern — re-encrypt with new list
age -d -i my.txt secret.age | age -e -R recipients.v2.txt -o secret.age.new
mv secret.age.new secret.age
```

### Encrypt-once-decrypt-by-anyone

age encrypts the actual payload **once** with a per-file key. Each recipient stanza wraps that key with its own KEM-style encapsulation. So:

- File grows by ~120 bytes per recipient (X25519) or ~600 bytes (ssh-rsa-4096).
- Decryption works for ANY listed recipient holding their identity.
- The list of recipients is INFERABLE from the file (each stanza names its type).

```bash
# Quick size demo
echo 'hi' | age -e -r age1... > 1r.age
echo 'hi' | age -e -r age1... -r age1... -r age1... > 3r.age
ls -l 1r.age 3r.age
# 3r.age is ~240 bytes larger than 1r.age
```

## Identity Files

```bash
# Identity file — one or more lines, # for comments
cat > identities.txt <<'EOF'
# personal
AGE-SECRET-KEY-1QXEAQ52AKR8ZYE65XTRF6QDD6WX8FQTSGW5ME5HQHYPN7TT8EMHQXQHGN8

# work — rotates 2024-Q3
AGE-SECRET-KEY-1ZJ7EA52GKR9ZYE76XTRF7QDD6XX9FQTSGW6ME6HQHYPN8TT9EMHQXQHGNX
EOF

chmod 600 identities.txt

# Use it
age -d -i identities.txt secret.age

# Multiple -i flags also fine
age -d -i ~/.config/age/personal.txt -i ~/.config/age/work.txt secret.age

# Stdin identity (Go age v1.2.0+; rage long supported)
cat ~/.config/age/key.txt | age -d -i - secret.age
```

### Passphrase-protecting an identity file

age has no native concept of an encrypted identity file, but the convention is to nest:

```bash
# Encrypt the identity itself with a passphrase
age -e -p -o key.txt.age key.txt
shred -u key.txt

# Decrypt-on-the-fly to use it
age -d key.txt.age | age -d -i - secret.age
# inner age prompts for the passphrase, outer age uses the resulting identity

# Or with a temp file (less secure — leaves the unwrapped key on disk briefly)
age -d -p -o /tmp/key.txt key.txt.age
age -d -i /tmp/key.txt secret.age
shred -u /tmp/key.txt
```

## ARMOR Mode

```bash
# Encrypt to ASCII (PEM-like)
age -e -a -r age1... -o secret.age.asc secret.txt

# Format
cat secret.age.asc
# -----BEGIN AGE ENCRYPTED FILE-----
# YWdlLWVuY3J5cHRpb24ub3JnL3Yx
# Cy0+IFgyNTUxOSBTZmxwT29Ga0h0eFlrUjA0dTZKaHpnRHAzdVNtVEtNUFYwa24w
# NWorOFRn
# UXlaNW42REo3SEtHd1hGdExySGhjZGtITDk4Qlh6MGJqK3JFOG01WWU4MAotLS0g
# WlE4WDlDbUxrUVR5Vit0NFhYZHZVTTVjSVhCTGJKck1hUHFMN24vRDZkWQ
# -----END AGE ENCRYPTED FILE-----

# Decrypt — no flag needed; auto-detected
age -d -i key.txt secret.age.asc

# Pipe-friendly
echo 'top secret' | age -e -a -r age1... | mail -s 'msg' you@example.com

# Lines wrap at 64 columns (RFC 7468 strict).
```

### When to use ARMOR

- Embedding in JSON / YAML / TOML / source code.
- Pasting into chat / email.
- Storing in environment variables.
- Avoid for large files — base64 inflates by ~33% (a 1 GiB tar becomes ~1.34 GiB).

```bash
# Encrypt a config snippet you want to commit to git
age -e -a -R recipients.txt config.toml > config.toml.age
git add config.toml.age recipients.txt    # NEVER commit the identity file

# Embed in environment variable
export DB_CREDS="$(age -e -a -r age1... < creds.json)"
```

## Passphrase Mode

```bash
# Encrypt with a passphrase
age -e -p -o secret.age secret.txt
# Enter passphrase (leave empty to autogenerate a secure one):
# Confirm passphrase:

# Auto-generated passphrase (press enter twice at the prompt)
age -e -p -o secret.age secret.txt
# Using the autogenerated passphrase "garage-fellow-cinder-pony-blizzard-quote-iguana-vine-mantis-dwarf".

# Decrypt — age detects scrypt stanza and prompts
age -d secret.age > secret.txt

# Multi-stanza: passphrase + recipient (any can decrypt)
# age does NOT support mixing -p with -r/-R in one invocation. Use chained encryption instead.
```

### Why no scripting

`-p` reads from `/dev/tty` directly to defeat shell scrollback / process listing. Piping a passphrase via stdin is **not** supported and will fail:

```bash
echo 'hunter2' | age -e -p -o x.age x.txt
# age: error: failed to read passphrase: input does not look like a TTY
```

For automation, use a key file (`-i`) and protect the file with filesystem permissions.

### scrypt parameters

age uses scrypt with N=2^18 (≈256 MiB memory), r=8, p=1. Decryption takes ~1 second on modern hardware, ~5 on a Raspberry Pi. Tuning is **not** exposed — the format pins it.

```bash
# Force a TTY for testing (Linux)
script -qc 'age -e -p -o x.age x.txt' /dev/null

# Or use `expect` for one-shot pipelining (NOT for production — passphrase visible in process tree)
expect <<EOF
spawn age -e -p -o secret.age secret.txt
expect "passphrase"; send "hunter2\r"
expect "Confirm";    send "hunter2\r"
expect eof
EOF
```

## Plugins

age plugins extend the recipient/identity types. Installation is just a binary on `$PATH` named `age-plugin-NAME`.

```bash
# Plugin discovery
which age-plugin-yubikey       # /usr/local/bin/age-plugin-yubikey
echo $PATH | tr : '\n' | xargs -I@ ls @/age-plugin-* 2>/dev/null

# Install a plugin
brew install age-plugin-yubikey
# or:
go install filippo.io/yubikey-agent/cmd/age-plugin-yubikey@latest

# Available plugins (community)
# age-plugin-yubikey      — PIV slot on a YubiKey 5+
# age-plugin-tpm          — non-exportable identity bound to TPM 2.0
# age-plugin-fido2-hmac   — any FIDO2 token with hmac-secret extension
# age-plugin-se           — macOS Secure Enclave (Touch ID gated)
# age-plugin-tkey         — Tillitis TKey USB token
# age-plugin-sss          — Shamir Secret Sharing (m-of-n)
# age-plugin-ledger       — Ledger Nano hardware wallet

# Recipient/identity prefix → plugin name
# age1yubikey1...   → age-plugin-yubikey
# AGE-PLUGIN-X-1... → age-plugin-x  (lowercased after AGE-PLUGIN-)
```

### Plugin protocol (high level)

age communicates with the plugin over stdin/stdout in a stanza-oriented protocol:

- `recipient-v1` phase — age sends recipients + file key, plugin returns wrapped stanzas.
- `identity-v1` phase — age sends stanzas, plugin unwraps file key (may prompt user via age for PIN/touch).

Failures cascade up as `age: error: ...` messages.

## YubiKey Workflow

```bash
# Install
brew install age-plugin-yubikey

# Generate a slot-bound identity (uses PIV slot 1 by default)
age-plugin-yubikey --generate
# Slot:           1
# Touch policy:   cached      (touch needed once per 15s)
# PIN policy:     once        (PIN once per session)
# Public key:     age1yubikey1q...
# Identity:       AGE-PLUGIN-YUBIKEY-1q...

# Output to file — recommended
age-plugin-yubikey --generate --identity > ~/.config/age/yubikey.txt
# AGE-PLUGIN-YUBIKEY-1q...

# List slots / identities on the device
age-plugin-yubikey --list                          # public recipients
age-plugin-yubikey --list-all                      # full slot info
age-plugin-yubikey --identity                      # human-readable identity for current default slot

# Encrypt to the YubiKey identity
age -e -r age1yubikey1q... -o secret.age secret.txt

# Decrypt — plugin will prompt for PIN and touch
age -d -i ~/.config/age/yubikey.txt secret.age
# 🔓 Please insert YubiKey with serial 12345678 to decrypt
# 🔓 Please enter YubiKey PIN: ******
# 🔓 Please touch the YubiKey

# Set policies at generation time
age-plugin-yubikey --generate \
  --slot 2 \
  --pin-policy always \
  --touch-policy always \
  --name 'work-laptop'
```

### Backup considerations

A YubiKey can be lost / damaged / left at the office. Always encrypt to **two** recipients:

```bash
age -e \
  -r age1yubikey1q...                      \  # daily driver
  -r age1emergencyprintedonpaperabcdef... \  # offline backup
  -o secret.age secret.txt
```

Or use Shamir splitting (`age-plugin-sss`) over the offline keys.

## TPM Workflow

```bash
# Install (Linux)
sudo apt install age-plugin-tpm
# or:
go install github.com/Foxboron/age-plugin-tpm/cmd/age-plugin-tpm@latest

# Generate a TPM-bound identity (sealed to current PCRs by default)
age-plugin-tpm --generate -o ~/.config/age/tpm.txt
# # Recipient: age1tpm1...
# AGE-PLUGIN-TPM-1...

# Encrypt to the TPM recipient
age -e -r age1tpm1... -o secret.age secret.txt

# Decrypt — only works on this exact TPM, this exact OS install (if PCRs sealed)
age -d -i ~/.config/age/tpm.txt secret.age

# PCR sealing — e.g. only when Secure Boot enabled and bootloader unchanged
age-plugin-tpm --generate --pcrs 0,2,4,7 -o ~/.config/age/tpm-sealed.txt
```

### Sealing patterns

- Bind to PCR 0+7 (UEFI firmware + Secure Boot policy) — survives kernel updates.
- Bind to PCR 0+7+11 — invalidates on kernel upgrade (force re-encrypt on every kernel bump).
- Bind to PCR 14 (Microsoft-style "boot manager") — Windows-side parity.

Identity is **non-exportable**: cloning a disk image to another machine breaks decryption.

## Streaming / Pipes

```bash
# Encrypt stdin → stdout
echo 'hello' | age -e -r age1... > hi.age

# Pipeline: tar | age
tar czf - data/ | age -e -R recipients.txt > backup.tar.gz.age

# Decrypt + extract
age -d -i key.txt < backup.tar.gz.age | tar xzf -

# Keep going through ssh
tar czf - data/ | age -e -r age1... | ssh backup@host 'cat > backup.age'

# rsync-friendly
age -d -i key.txt backup.age | rsync --inplace -av - dest:/path/

# Cooperate with pv for progress
pv data.bin | age -e -r age1... > data.bin.age

# Compose with split for chunked transport
age -e -R r.txt big.iso | split -b 1G - big.iso.age.part-
cat big.iso.age.part-* | age -d -i key.txt > big.iso.recovered
```

### No-seek constraint

age decrypts strictly forward — you cannot start mid-file. For random-access scenarios (huge VM disk images), wrap with a chunking layer (`split`, `tar | xs`) or use a different tool (`gocryptfs`, `cryptsetup`).

```bash
# rage-mount — Rust-only convenience for read-only random access
rage-mount -t age secret.age.tar /mnt/decrypted
# /mnt/decrypted now lets you cd / cat / ls; backed by FUSE
fusermount -u /mnt/decrypted
```

## Integration with sops

[sops](https://github.com/getsops/sops) (Mozilla / CNCF) uses age as one of several KEKs (alongside KMS, GCP KMS, Vault).

```bash
# .sops.yaml — declares which recipients can decrypt which paths
cat > .sops.yaml <<'EOF'
creation_rules:
  - path_regex: secrets/prod/.*
    age: >-
      age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p,
      age1emergencyabcdefghijklmnopqrstuvwxyz0123456789abcdefghijkl
  - path_regex: secrets/dev/.*
    age: age1devkeyabcdefghijklmnopqrstuvwxyz0123456789abcdefghijklm
EOF

# Encrypt — sops detects file extension (yaml, json, env, ini, binary)
sops -e -i secrets/prod/db.yaml          # in-place

# Decrypt
sops -d secrets/prod/db.yaml > /tmp/db.yaml

# Edit (decrypts → opens $EDITOR → re-encrypts)
sops secrets/prod/db.yaml

# Tell sops where the identity lives
export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt
# (older syntax: AGE_KEY_FILE — still respected by some plugins)

# Or via a literal key
export SOPS_AGE_KEY="AGE-SECRET-KEY-1QXEAQ..."

# Rotate — replace recipients in .sops.yaml then:
find secrets -type f -exec sops updatekeys {} \;
```

### Common sops + age idioms

```bash
# CI runner — pipe identity in, never write to disk
echo "$AGE_KEY_CI" | sops -d --input-type yaml --age-keys /dev/stdin secrets/prod/db.yaml

# Decrypt into env vars (sops --output-type dotenv → systemd EnvironmentFile)
sops -d --output-type dotenv secrets/prod/db.yaml > /run/secrets/db.env
```

## Integration with chezmoi / dotfiles

[chezmoi](https://www.chezmoi.io) — dotfile manager — encrypts secrets with age.

```bash
# One-time: tell chezmoi which recipient
chezmoi init --apply
# In ~/.config/chezmoi/chezmoi.toml:
# encryption = "age"
# [age]
#   identity = "/home/me/.config/age/key.txt"
#   recipient = "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

# Encrypt a file as you add it
chezmoi add --encrypt ~/.aws/credentials
# stored in $(chezmoi source-path)/encrypted_dot_aws/credentials.age

# Apply (decrypts on the fly)
chezmoi apply
```

The chezmoi source repo is **safe to push to public git** — every encrypted_*.age file is unreadable without the identity. The recipient line in `chezmoi.toml` is public.

## Integration with NixOS / agenix / sops-nix

### agenix

[agenix](https://github.com/ryantm/agenix) — secret management for NixOS using age + each host's SSH host key.

```nix
# secrets.nix — at the root of your flake
let
  alice = "ssh-ed25519 AAAAC3...";
  host1 = "ssh-ed25519 AAAAC3...";  # /etc/ssh/ssh_host_ed25519_key.pub of host1
in {
  "secrets/db.age".publicKeys = [ alice host1 ];
  "secrets/api.age".publicKeys = [ alice host1 ];
}
```

```bash
# Encrypt a secret (agenix CLI is a wrapper around age)
agenix -e secrets/db.age          # opens $EDITOR; re-encrypts on save

# Re-encrypt all secrets after rotating publicKeys
agenix -r

# In the host's NixOS config
{
  age.identityPaths = [ "/etc/ssh/ssh_host_ed25519_key" ];
  age.secrets.db.file = ./secrets/db.age;
  age.secrets.db.path = "/run/secrets/db";
  age.secrets.db.owner = "postgres";
}
```

### sops-nix

[sops-nix](https://github.com/Mic92/sops-nix) — sops-based secrets, age recipients listed in `.sops.yaml`, decrypted at boot using the host's SSH key.

```yaml
# .sops.yaml
keys:
  - &alice age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
  - &host1 age1lggyhqrw2nlhcxprm67z43rta597azn8gknawjehu9d9dl0jq3yqqvfafg
creation_rules:
  - path_regex: secrets/.*\.yaml
    key_groups:
      - age: [*alice, *host1]
```

### Host-key → age conversion

```bash
# Convert an SSH ed25519 host pubkey → age recipient
ssh-keygen -y -f /etc/ssh/ssh_host_ed25519_key | age-keygen-rs --to-recipient
# (Note: upstream age-keygen lacks --convert. Use ssh-to-age — github.com/Mic92/ssh-to-age)
ssh-to-age < /etc/ssh/ssh_host_ed25519_key.pub
# age1...
```

### Rotation

```bash
# Edit secrets.nix to add/remove a publicKey, then
agenix -r
git diff                                # all *.age files re-encrypted

# sops-nix
$EDITOR .sops.yaml
sops updatekeys secrets/db.yaml
```

## Integration with Kubernetes

Three common paths:

### sops + kustomize (ksops)

```bash
# Install ksops as a kustomize generator plugin
go install github.com/viaduct-ai/kustomize-sops/cmd/ksops@latest

# kustomization.yaml
generators:
  - secret-generator.yaml

# secret-generator.yaml
apiVersion: viaduct.ai/v1
kind: ksops
files:
  - ./db-creds.yaml         # sops + age encrypted

# Build
kustomize build --enable-alpha-plugins .
```

### helm-secrets

```bash
helm plugin install https://github.com/jkroepke/helm-secrets

# values.yaml.dec → values.yaml (encrypted in repo)
helm secrets enc values.yaml
helm secrets install myrelease ./chart -f values.yaml
```

### sealed-secrets vs age-encrypted manifests

| Aspect              | sealed-secrets               | sops + age                       |
|---------------------|------------------------------|----------------------------------|
| Decryption locus    | In-cluster controller        | At apply time (CI / kustomize)   |
| Key custody         | Controller's RSA key in K8s  | Each operator's age identity     |
| Rotation            | Re-seal each Secret          | Re-encrypt the file              |
| Multi-cluster       | One key per cluster          | Same age recipients for all      |
| GitOps friendliness | Native                       | Native via ksops/helm-secrets    |

```bash
# Decrypt in CI for kubectl apply
sops -d k8s/prod/secret.yaml | kubectl apply -f -
```

## rage — the Rust Implementation

[rage](https://github.com/str4d/rage) is the Rust port; it implements the same v1 spec and is byte-compatible with Go age.

```bash
# Install
cargo install rage          # rage, rage-keygen, rage-mount

# Generate a key — same format
rage-keygen -o key.txt

# Encrypt — ARMOR is the default in rage when stdout is a TTY
rage -e -r age1... secret.txt > secret.age

# Decrypt — same flags
rage -d -i key.txt secret.age > secret.txt

# rage-mount — read-only FUSE for tar.age archives
rage-mount -t tar -i key.txt big.tar.age /mnt/x
ls /mnt/x
fusermount -u /mnt/x

# Differences from go-age (1.x)
# - rage compiles to a smaller static binary (~2 MB vs 6 MB)
# - rage-mount provides FUSE mount for tar.age (no Go equivalent)
# - rage flag for ASCII output is `-a` / `--armor`, same as Go age
# - rage emits prettier multi-line errors with caret-pointers
# - rage refuses to write binary age to a TTY by default; pass --force
# - perf: rage is ~10–30% faster on encrypt for large files (libsodium-backed AEAD)
```

```bash
# Mix-and-match: encrypt with rage, decrypt with Go age (or vice versa)
rage -e -r age1... < big.bin > big.age
age -d -i key.txt big.age > big.bin     # works fine
```

## Comparison vs gpg

| Aspect                  | age                                   | gpg                                       |
|-------------------------|---------------------------------------|-------------------------------------------|
| Spec lines              | ~150                                  | thousands (RFC 4880, 9580)                |
| Binary size             | ~3–6 MB                               | ~10–50 MB (with libgcrypt etc.)           |
| Algorithms              | Frozen: X25519, ChaCha20-Poly1305     | RSA, DSA, ECDSA, EdDSA, IDEA, CAST5, ...  |
| Key sizes               | 256-bit only                          | 1024–8192 bit RSA, etc.                   |
| Format                  | One; binary or ARMOR                  | OpenPGP packets, MDC, SEIPDv2, SEIPDv1    |
| Keyring                 | None — files                          | `~/.gnupg/`                               |
| Web of trust            | None — explicit recipient lists       | Yes (signatures on keys)                  |
| Signing                 | NO — use `minisign`/`signify`/`ssh-keygen -Y sign` | Yes (detached, clear, inline)             |
| Compression             | None                                  | Yes (zlib, bzip2)                         |
| Plugin protocol         | Yes — minimal stdio                   | smartcard daemon (scdaemon)               |
| Audit history           | One major audit (NCC, 2021), clean    | Long history of CVEs (EFAIL, etc.)        |

```bash
# Same task in both
# age — encrypt to a friend (you have their public key as a string)
age -e -r age1theirkey... -o msg.age msg.txt

# gpg — same friend (you have their key in your keyring after signing)
gpg --encrypt --recipient friend@example.com --output msg.gpg msg.txt
```

### Why not signing?

age is intentionally encryption-only. To sign:

```bash
# minisign — small, recommended pairing with age
minisign -G -s ~/.minisign.key -p ~/minisign.pub
minisign -S -s ~/.minisign.key -m secret.age      # produces secret.age.minisig
minisign -V -p ~/minisign.pub -m secret.age

# signify (OpenBSD's tool)
signify -G -s ~/.signify.sec -p ~/signify.pub
signify -S -s ~/.signify.sec -m secret.age

# ssh-keygen as signer (using the same SSH key that's an age recipient)
ssh-keygen -Y sign -f ~/.ssh/id_ed25519 -n age secret.age
ssh-keygen -Y verify -f ~/.ssh/id_ed25519.pub -I name@host -n age -s secret.age.sig < secret.age
```

## Comparison vs openssl enc

```bash
# DON'T DO THIS — openssl enc is broken-by-default
openssl enc -aes-256-cbc -salt -in secret.txt -out secret.enc -k 'hunter2'
```

Why it's broken:

1. **No authentication.** AES-CBC is malleable; an attacker who flips ciphertext bytes flips plaintext bytes. No MAC = no integrity.
2. **Weak KDF.** The default key-derivation is a single MD5(password‖salt) iteration up to 1.0; PBKDF2 (`-pbkdf2 -iter 100000`) was added later but is not the default in many distros.
3. **No nonce/IV check.** Reuse leads to silent plaintext recovery on related messages.
4. **No format versioning.** Files have a `Salted__` magic but no algorithm tag; rotating ciphers is undefined.
5. **Default cipher in 1.0.x is RC4 / DES via aliases on some builds.**

age replaces all of the above with one fixed, authenticated stack: ChaCha20-Poly1305 + HKDF + (X25519 or scrypt) — no choices to get wrong.

```bash
# Equivalent in age — passphrase mode
age -e -p -o secret.age secret.txt
# Auth (Poly1305), strong KDF (scrypt N=2^18), versioned format. Done.
```

## Common Errors

```
age: error: no identity matched any of the recipients
```
Wrong identity for the file's recipient stanzas. Run `grep '^-> ' file.age` to see types; pass the matching `-i`.

```
age: error: failed to read header
```
File is truncated, not actually age, or the magic line is missing. Check `head -1 file.age` returns `age-encryption.org/v1`.

```
age: error: malformed recipient: "age1nope"
```
Typo in an `age1...` Bech32 string. Recipients are 62 chars, all lowercase, no `0/O/I/l` confusion possible (Bech32 alphabet excludes them).

```
age: error: malformed recipient: "ssh-ed25519 ..."
```
SSH-format key is missing the base64 part or has a wrong type. Test with `ssh-keygen -lf <(echo "<line>")`.

```
age: error: no recipients specified
```
You ran `age -e ... < input` without any of `-r`, `-R`, `-p`, or `-i` (when used as recipient).

```
age: error: identity file is malformed: line N
```
The identity file has a comment-misformatted line, or a non-AGE-SECRET-KEY line that isn't an SSH private key. Check `cat -A keyfile` for hidden chars.

```
age: warning: -a/--armor is set but the output is not a terminal
```
Cosmetic — you used `-a` but stdout was a pipe/file. Either add `-o file.asc` or drop `-a`.

```
age: error: failed to read passphrase: input does not look like a TTY
```
You piped into `age -p`. Passphrase mode requires `/dev/tty`. Use a key file instead, or wrap with `script -qc 'age -p ...' /dev/null`.

```
age: error: HMAC mismatch
```
Header was tampered with — file is not safe to decrypt; abort. Indistinguishable from random corruption.

```
age: error: invalid header
```
Header structure doesn't parse — usually means the file isn't age, or armor envelope was stripped.

```
age: error: unknown recipient type: "yubikey"
```
Plugin not on `$PATH`. The plugin binary must be named `age-plugin-yubikey` and be executable. Run `which age-plugin-yubikey` to verify.

```
age: error: plugin: age-plugin-yubikey: exit status 1
```
Plugin started but errored. Run the plugin manually (`age-plugin-yubikey --identity`) to see the diagnostic.

```
age: error: header has too many recipients
```
The file has an unreasonable number of `-> ` stanzas (limit ~256 in current implementations). Likely a corrupted or maliciously-crafted file.

```
age: error: payload nonce: short read
```
File truncated mid-payload — last write was incomplete (interrupted `tar | age >`).

```
age: error: failed to decrypt: chacha20poly1305: message authentication failed
```
Either tampered ciphertext or wrong file-key — usually means an SSH key matched a stanza but the file's per-stanza wrap doesn't validate (hash collisions on stanza key). Try other identities.

```
age: error: ssh: this private key is passphrase protected
```
Encrypted SSH key — provide the passphrase when prompted, or convert to unencrypted with `ssh-keygen -p -P 'old' -N '' -f key`.

```
age: error: ssh: no key found
```
SSH file isn't actually a private key (e.g. `id_ed25519.pub` instead of `id_ed25519`).

## Common Gotchas

### 1. Forgetting `-i` when decrypting an X25519 file

```bash
# BROKEN — file has X25519 stanzas, no SSH stanzas; default lookup finds nothing
age -d secret.age
# age: error: no identity matched any of the recipients

# FIXED
age -d -i ~/.config/age/key.txt secret.age
```

### 2. `-p` in a script with no TTY

```bash
# BROKEN
echo 'hunter2' | age -e -p -o secret.age secret.txt
# age: error: failed to read passphrase: input does not look like a TTY

# FIXED — use a key file (preferred)
age -e -i ~/.config/age/key.txt -o secret.age secret.txt

# OR — fake a TTY (works locally, awkward in CI)
script -qc "age -e -p -o secret.age secret.txt" /dev/null
```

### 3. Encrypting then losing the identity

```bash
# BROKEN — no recovery; age has no escrow
rm ~/.config/age/key.txt && age -d -i missing.txt secret.age
# age: error: failed to open identity: open missing.txt: no such file or directory

# FIXED — always encrypt to two recipients
age -e \
  -r age1daily... \
  -r age1paperbackup... \
  -o secret.age secret.txt
```

### 4. Committing a "public key" file that's actually a secret

```bash
# BROKEN — looks public-ish but the file IS the secret
cat key.txt
# AGE-SECRET-KEY-1QXEAQ52AKR8...                # <-- this is the SECRET
git add key.txt && git commit                  # CATASTROPHE

# FIXED — only commit the recipient (public)
age-keygen -y key.txt > key.pub
echo 'key.txt' >> .gitignore
git add key.pub                                 # safe — that's the public part
```

### 5. SSH key reuse threat model

```bash
# Risk — a stolen ~/.ssh/id_ed25519 = stolen SSH access AND every age file ever
# you encrypted to it.

# MITIGATE — use a dedicated age identity for high-stakes files
age-keygen -o ~/.config/age/highstakes.txt
chmod 600 ~/.config/age/highstakes.txt
# encrypt only to age1 from highstakes.txt; never to ssh-ed25519 lines
```

### 6. Plugin not on `$PATH`

```bash
# BROKEN — plugin installed in a non-PATH location
ls /opt/yubikey/bin/age-plugin-yubikey
age -d -i ~/.config/age/yubikey.txt secret.age
# age: error: unknown recipient type: "yubikey"

# FIXED
export PATH="/opt/yubikey/bin:$PATH"
age -d -i ~/.config/age/yubikey.txt secret.age
```

### 7. Multi-recipient file shrinks to 1 stanza when re-encrypting

```bash
# BROKEN — naively re-encrypting drops everyone but you
age -d -i my.txt secret.age | age -e -r age1mine... > secret.age.new
# Now ONLY you can decrypt; the team is locked out.

# FIXED — always pass the full recipients file when re-encrypting
age -d -i my.txt secret.age | age -e -R recipients.txt > secret.age.new
```

### 8. Missing `-R` when given a file

```bash
# BROKEN — age treats `recipients.txt` as a literal recipient string
age -e -r recipients.txt secret.txt
# age: error: malformed recipient: "recipients.txt"

# FIXED
age -e -R recipients.txt secret.txt
```

### 9. ARMOR roundtrip with trailing newlines

```bash
# BROKEN — embedding ARMOR in YAML drops the trailing newline → "invalid header"
yaml: "$(age -e -a -r age1... < x)"

# FIXED — use literal block (`|`) in YAML
key: |
  -----BEGIN AGE ENCRYPTED FILE-----
  YWdlLWVuY3J5cHRpb24ub3JnL3Yx
  ...
  -----END AGE ENCRYPTED FILE-----
```

### 10. Forgetting that ARMOR is auto-detected on decrypt

```bash
# Both work
age -d -i k.txt secret.age      # binary
age -d -i k.txt secret.age.asc  # ARMOR — no flag needed
# age: warning: -a/--armor is set but the output is not a terminal       (only on encrypt)
```

### 11. Stale recipients in a long-lived file

```bash
# BROKEN — Bob left the team a year ago; he still decrypts the archive
grep -c '^-> ' archive.age           # 5 recipients

# FIXED — periodic rotation
age -d -i shared.txt archive.age | age -e -R recipients.current.txt > archive.age.new
mv archive.age.new archive.age
```

### 12. Encrypted SSH key prompts on every file in a loop

```bash
# BROKEN — re-prompts for passphrase 100 times
for f in *.age; do age -d -i ~/.ssh/id_ed25519 "$f" > "${f%.age}"; done

# FIXED — temporarily unlock once
ssh-add -t 1h ~/.ssh/id_ed25519                    # not used by age, but the user is then aware
# OR copy key to memory
cp ~/.ssh/id_ed25519 /dev/shm/k && chmod 600 /dev/shm/k
for f in *.age; do age -d -i /dev/shm/k "$f" > "${f%.age}"; done
shred -u /dev/shm/k
```

## Idioms

```bash
# Quickest one-liner: encrypt a file to a single recipient
age -e -r "$(cat alice.pub)" -o secret.age secret.txt

# Encrypted backup of a directory tree
tar czf - my-dir/ | age -e -R recipients.txt > backup-$(date +%F).tar.gz.age

# Restore
age -d -i key.txt backup-2024-04-01.tar.gz.age | tar xzf -

# Team distribution via authorized_keys fan-out
age -e -R ~/.ssh/authorized_keys -o announcement.age announcement.txt

# Distribute via GitHub
curl -s https://github.com/alice.keys https://github.com/bob.keys > team.keys
age -e -R team.keys -o secret.age secret.txt

# Rotate recipients on every file in a directory
for f in secrets/*.age; do
  age -d -i my.txt "$f" | age -e -R recipients.v2.txt > "$f.new" && mv "$f.new" "$f"
done

# password-store + age combo (the `pass` family)
# Use https://github.com/FiloSottile/passage — drop-in fork of pass that uses age
passage init age1...
passage insert finance/bank
passage finance/bank
passage git push

# Encrypt + checksum
age -e -r age1... < secret.txt | tee secret.age | sha256sum > secret.age.sha256

# pipe a JSON secret into Vault for storage of the encrypted blob
age -e -a -r age1... < creds.json | vault kv put secret/team-creds content=-

# encrypted git-crypt replacement (manual)
# pre-commit hook encrypts; post-checkout decrypts via clean/smudge filter chain
git config filter.age.clean 'age -e -R .age-recipients --'
git config filter.age.smudge 'age -d -i ~/.config/age/key.txt --'

# Print the recipient (public) for an identity file
age-keygen -y ~/.config/age/key.txt

# Encrypted scratch buffer (vim)
vim +':set viminfo= bin noundofile noswapfile' \
    '+%!age -d -i ~/.config/age/key.txt' \
    '+set fenc=utf-8 binary nobinary' \
    secret.age
# On :wq, run `:%!age -e -r age1... -a` first

# Encrypted journal entry per day
ENTRY=~/journal/$(date +%F).md.age
$EDITOR /tmp/entry.md && age -e -r age1... -o "$ENTRY" /tmp/entry.md && shred -u /tmp/entry.md

# Read encrypted env into bash (NEVER `eval` untrusted output)
set -a; source <(age -d -i k.txt secrets.env.age); set +a

# Shell completion in scripts
type age >/dev/null 2>&1 || { echo "age not installed" >&2; exit 1; }
```

## Backup + Recovery

```
LOST IDENTITY = LOST DATA
                ↑
             no escrow, no key-server, no reset-by-email
```

### Two-recipient pattern

```bash
# Always have at least one offline recipient
age-keygen -o /mnt/usb/offline.txt          # generated on an air-gapped machine
age-keygen -y /mnt/usb/offline.txt > offline.pub
mv offline.pub repo/
# never plug the USB back into a networked host

# All future encryptions
age -e -R repo/recipients.txt -r "$(cat repo/offline.pub)" -o secret.age secret.txt
```

### Paper backup of an identity

```bash
# Identity is ~74 chars + comments; print it
age-keygen | tee >(qrencode -o backup.png)

# Or split with Shamir (m-of-n)
age-keygen | age-plugin-sss -t 3 -n 5 -o shares/    # 3 of 5 reconstruct
# distribute shares to 5 trusted parties
```

### Passphrase-encrypted identity file

```bash
# Encrypt the identity file at rest with a strong passphrase
age -e -p -o ~/.config/age/key.txt.age ~/.config/age/key.txt
shred -u ~/.config/age/key.txt

# Decrypt-on-the-fly
age -d ~/.config/age/key.txt.age | age -d -i - secret.age
```

### Verifying you can still decrypt

```bash
# Quarterly drill — pick three random files, decrypt them
find ~/encrypted -name '*.age' | shuf | head -3 | while read -r f; do
  age -d -i ~/.config/age/key.txt "$f" >/dev/null && echo "OK: $f" || echo "FAIL: $f"
done
```

### Hardware token spare

```bash
# Always provision two YubiKeys in parallel
age-plugin-yubikey --generate --slot 1 --serial 12345678 -o yk1.txt   # daily driver
age-plugin-yubikey --generate --slot 1 --serial 87654321 -o yk2.txt   # safe spare

# Add both as recipients on every file
age -e -r "$(grep '^# Recipient' yk1.txt | cut -d: -f2 | tr -d ' ')" \
       -r "$(grep '^# Recipient' yk2.txt | cut -d: -f2 | tr -d ' ')" \
       -R repo/recipients.txt -o secret.age secret.txt
```

## Threat Model

### What age PROTECTS

- **Confidentiality** — payload is unreadable without an authorised identity (X25519 + ChaCha20).
- **Integrity** — Poly1305 tag on every chunk; HMAC over header. Tampering = decrypt failure.
- **Forward integrity** of the chunk stream — chunk N+1's nonce is bound to N (counter), so chunk reordering, truncation, or duplication fails.

### What age does NOT protect

- **Metadata.** Filenames, sizes (within ~64-KiB chunks), timestamps, transport are visible. A 12 GiB age file is recognisably 12 GiB.
- **Recipient identity privacy.** Each `-> ` stanza announces its type + the X25519 wrap of the file key; an observer learns *how many* recipients and *what type*. (For X25519 the recipient pubkey is NOT in the stanza, but it can be tested against any candidate pubkey via the wrap — so you can confirm "is this recipient on the file" if you know the candidate.)
- **Traffic analysis** of when files are accessed.
- **Sender authentication.** Anyone can produce a valid age file claiming to come from anyone. Combine with `minisign` / `ssh-keygen -Y sign` for sender attribution.
- **Deniability.** age is NOT a deniable system; gpg's web-of-trust signatures are also non-deniable, but age has no built-in signature feature, leaving sender attribution to a separate tool.
- **Side channels.** Constant-time ChaCha20 + Curve25519 implementations are used, but the surrounding shell, editor, and disk paths can leak (swap, journals, undo files, less history).

### Algorithms (immutable)

| Layer       | Algorithm              | Notes                                |
|-------------|------------------------|--------------------------------------|
| Asymmetric  | X25519                 | RFC 7748                             |
| KDF         | HKDF-SHA-256           | RFC 5869                             |
| AEAD        | ChaCha20-Poly1305      | RFC 8439                             |
| Passphrase  | scrypt                 | N=2^18, r=8, p=1                     |
| ssh-rsa     | RSAES-OAEP-SHA-256     | with MGF1-SHA-256                    |
| ssh-ed25519 | X25519 (converted)     | derived via SHA-512 of seed          |
| Header MAC  | HMAC-SHA-256           | over magic + stanzas                 |

### Quantum considerations

age uses X25519, which is breakable by a sufficiently large quantum computer (Shor). For long-term archives that must remain confidential past ~2035, layer with a PQC scheme:

```bash
# Pre-encrypt with a PQ KEM (e.g. Kyber via openssl/oqs) then age-encrypt the result
oqs-encrypt --kem kyber768 -r kyber.pub < secret.txt | age -e -r age1... > secret.age
```

age has no PQC plugin yet (April 2026). See the `polyglot` sheet for hybrid encryption patterns.

## See Also

- gpg
- openssl
- ssh
- sops
- polyglot

## References

- age homepage — <https://age-encryption.org>
- age v1 spec — <https://age-encryption.org/v1>
- Filippo Valsorda's design notes — <https://words.filippo.io/dispatches/age-authentication/>
- Go reference implementation — <https://github.com/FiloSottile/age>
- rage (Rust port) — <https://github.com/str4d/rage>
- Plugin protocol spec — <https://github.com/C2SP/C2SP/blob/main/age-plugin.md>
- ssh-to-age — <https://github.com/Mic92/ssh-to-age>
- agenix (NixOS) — <https://github.com/ryantm/agenix>
- sops-nix — <https://github.com/Mic92/sops-nix>
- sops — <https://github.com/getsops/sops>
- chezmoi age docs — <https://www.chezmoi.io/user-guide/encryption/age/>
- age-plugin-yubikey — <https://github.com/str4d/age-plugin-yubikey>
- age-plugin-tpm — <https://github.com/Foxboron/age-plugin-tpm>
- age-plugin-fido2-hmac — <https://github.com/olastor/age-plugin-fido2-hmac>
- age-plugin-se (Secure Enclave) — <https://github.com/remko/age-plugin-se>
- passage (pass + age) — <https://github.com/FiloSottile/passage>
- minisign — <https://jedisct1.github.io/minisign/>
- signify — <https://man.openbsd.org/signify>
- NCC Group audit (2021) — <https://research.nccgroup.com/2021/02/22/public-report-age-security-review/>
- RFC 7748 (X25519) — <https://datatracker.ietf.org/doc/html/rfc7748>
- RFC 8439 (ChaCha20-Poly1305) — <https://datatracker.ietf.org/doc/html/rfc8439>
- RFC 5869 (HKDF) — <https://datatracker.ietf.org/doc/html/rfc5869>
- RFC 7468 (PEM/ARMOR) — <https://datatracker.ietf.org/doc/html/rfc7468>
- scrypt RFC 7914 — <https://datatracker.ietf.org/doc/html/rfc7914>
