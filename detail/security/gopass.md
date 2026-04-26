# gopass internals

Deep dive into gopass — the team password manager built on `pass(1)`, git, and your existing GPG/age tooling. The why behind a deceptively simple file-per-secret store.

## Setup

`gopass` is a pure-Go reimplementation and superset of Jason Donenfeld's `pass(1)`. Conceptually it inherits three design choices from `pass`:

1. **Filesystem-as-database**. Every secret is a separate file under `~/.password-store/`. There is no database, no sqlite, no JSON blob. `tree ~/.password-store` shows your password tree exactly as it is.
2. **GPG-as-crypto**. Encryption is delegated to `gpg-agent`. There is no in-process crypto. This means crypto bugs, key management, smartcard integration, and PIN caching are all GPG's problem (and benefit).
3. **git-as-replication**. Synchronisation is `git push`/`git pull`. There is no central server. Conflicts are git conflicts. History is git history.

`gopass` extends `pass` along several axes:

- **Multi-mount stores** — work secrets and personal secrets and the homelab share the same binary, with separate roots, separate `.gpg-id` files, and (optionally) separate crypto backends.
- **age backend** — instead of GPG, use Filippo Valsorda's `age` for modern, low-fuss crypto. No web of trust, no expiring subkeys.
- **Plugin protocol** — JSON-API for browser integrations, Summon provider for secret injection, FUSE filesystem mount.
- **Templates** — scaffold a new secret from a template file in the parent directory.
- **OTP** — store `otpauth://` URIs, generate TOTP codes on demand.
- **Audit & fsck** — verify the store's structural and cryptographic integrity.
- **Init wizard** — interactive recipient selection, git remote setup, mount registration.

The North Star of gopass is *operational simplicity*: a Linux user with `gpg --gen-key` already done is one `gopass init` away from a working store, and a developer with a USB YubiKey is one `gpg --card-status` away from passwordless smartcard use.

The downside, which we'll explore in detail in §16, is that gopass inherits all of GPG's foot-guns and offers no protection against the larger threats: a compromised desktop, a malicious clipboard reader, or a screen-recording attacker.

## Architecture

The core data structure is a single directory tree:

```
~/.password-store/
├── .gpg-id             # one fingerprint per line, recipients
├── .gpg-id.sig         # optional: detached signature of .gpg-id
├── .public-keys/       # exported recipient pubkeys (for non-keyring use)
├── personal/
│   ├── .gpg-id         # overrides parent for this subtree
│   ├── github.com/
│   │   ├── alice.gpg   # one secret = one file
│   │   └── bob.gpg
│   └── bank.gpg
└── work/
    ├── .gpg-id         # work-only recipients
    └── prod/
        ├── .gpg-id     # tighter recipient list for prod
        └── api-key.gpg
```

A "secret" is a single `.gpg` (or `.age` for age stores) file. The format inside is application-defined: gopass uses a YAML-front-matter style by default, but any text works.

By convention, the first line of a decrypted secret is the password; subsequent lines are arbitrary metadata. This was inherited from `pass` and is preserved for compatibility:

```
super-secret-password-here
url: https://example.com
user: alice
notes: |
  recovery codes
  - 1234
  - 5678
```

Tools that consume secrets typically read line 1 only; tools that need metadata parse YAML.

The `.gpg-id` file is the recipient list. Each non-comment line is a GPG fingerprint (or short keyid, but always prefer fingerprints). Encryption walks up the directory tree looking for `.gpg-id`; the deepest match wins. This makes per-team recipient policies trivial:

```
~/.password-store/.gpg-id           # everyone
~/.password-store/work/.gpg-id      # work team only
~/.password-store/work/prod/.gpg-id # SREs only
```

The `.public-keys/` directory is an optional escape hatch for environments where users may not have all recipient keys in their local keyring. gopass can `gpg --import` from this directory transparently before encrypting, so a new team member who joins by adding their fingerprint to `.gpg-id` doesn't need everyone to manually import their key.

A multi-mount store lets you have multiple roots. The default mount is `~/.password-store`; additional mounts live wherever you point them and are registered in `~/.config/gopass/config.yml`:

```yaml
mounts:
  default: /home/alice/.password-store
  work: /home/alice/work/secrets
  homelab: /home/alice/homelab/store
```

You access them with prefix: `gopass show work/prod/api-key`.

## Encryption Workflow

The classic `gopass insert foo/bar` flow:

1. gopass walks up from `foo/` to find the nearest `.gpg-id`.
2. It reads the fingerprints listed there.
3. It optionally imports any keys present under `.public-keys/`.
4. It collects the new secret bytes (from stdin, or by spawning `$EDITOR` against a temp file in `tmpfs` if available).
5. It calls `gpg --encrypt --recipient FP1 --recipient FP2 ... --output foo/bar.gpg.tmp`.
6. It atomically renames `foo/bar.gpg.tmp` to `foo/bar.gpg`.
7. It runs `git add foo/bar.gpg && git commit`.
8. If auto-push is enabled, `git push`.

The atomic rename is non-negotiable: a partial write to `foo/bar.gpg` followed by a crash would leave a truncated, undecryptable file. The temp-file-then-rename pattern guarantees that any reader either sees the old file or the fully-written new file, never a half-written one.

The temp file used for editing is the security-critical step. gopass tries hard to put it on `tmpfs` (which is RAM-backed and never hits disk):

- On Linux, `/dev/shm` if writable.
- On macOS, `$TMPDIR` if it's `/var/folders/...` (which is a per-user tmpfs).
- Falls back to `os.TempDir()`.

The temp file is unlinked immediately on close, but gopass also overwrites it with zeroes first as a defense against filesystem journals or recovery tools — though against a determined forensic attacker with disk access, the only real defense is encryption-at-rest on the device.

For the show path:

1. `gpg --decrypt foo/bar.gpg` is invoked.
2. gpg-agent prompts for passphrase or smartcard PIN if not cached.
3. The plaintext is streamed to gopass.
4. gopass either prints to stdout, copies to clipboard (and schedules a clear), or pipes to a consumer.

The PIN cache is gpg-agent's job. `default-cache-ttl 600` and `max-cache-ttl 7200` are typical. Smartcard-backed keys may have a touch policy that requires user interaction every time regardless of cache.

## Recipients

The `.gpg-id` file is the only authoritative source of who can decrypt. A common operational mistake is changing `.gpg-id` and assuming all existing files are now decryptable by the new recipient — they aren't. Each `.gpg` file was encrypted with a specific recipient list at creation time, and that list is baked into the file's PKESK packets.

To re-key existing files, you must run `gopass recipients update` (or `gopass sync`). This:

1. Decrypts every file in the affected subtree.
2. Re-encrypts to the *current* `.gpg-id` recipients.
3. Replaces the file in place (atomic rename).
4. Commits.

This is why removing a recipient from `.gpg-id` is necessary but not sufficient — the recipient must also no longer have access to git history, since old commits contain old encrypted blobs they can decrypt. We discuss this fully in §17.

Recipients are GPG fingerprints. The full 40-hex-character fingerprint is required (or strongly preferred). Short keyids (8-hex) are vulnerable to fingerprint collisions and gopass will warn or refuse to use them. Long keyids (16-hex) are also collisionable in principle though no public attacks exist.

A `.gpg-id.sig` detached signature is an optional integrity check on `.gpg-id` itself. Without it, an attacker with write access to your git remote could add their own fingerprint to `.gpg-id`, then on next encryption, all secrets would be encrypted to them as well. With the signature, gopass verifies `.gpg-id` against a trusted-signers list before honouring it.

Trust ultimately comes from the keyring: if your local GPG keyring trusts a key for the signature, gopass trusts `.gpg-id`. This pushes trust establishment back to standard GPG flows (key signing parties, fingerprint verification on first import, etc.).

## Multi-Mount

`gopass mounts add work /path/to/work-store` registers a new mount. The mount mechanism is purely a naming layer in gopass — each mount is its own independent password store with its own `.gpg-id`, its own git repo, its own crypto backend.

The cross-mount copy and move semantics are subtle:

- `gopass copy default/foo work/foo` — decrypts `default/foo`, re-encrypts to `work/.gpg-id` recipients, writes `work/foo`.
- `gopass move default/foo work/foo` — same as copy, then `gopass rm default/foo`.
- Both are atomic per-file; if encryption fails partway, neither store is mutated.

A common pattern is one mount per "trust domain":

- `default`: solo personal secrets (laptop only).
- `work`: shared with colleagues via git remote.
- `homelab`: shared with self across machines.
- `family`: shared with spouse via git on a self-hosted Gitea.

Each mount can use a different crypto backend. `default` might use age (fast, modern), `work` might use GPG with smartcards, `homelab` might use GPG without smartcards. The choice is per-store, recorded in `.gopass/store.cfg`.

## Git Integration

Auto-commit is the default. Every mutation produces a commit:

- `gopass insert foo` → "Add foo"
- `gopass edit foo` → "Edit foo"
- `gopass rm foo` → "Remove foo"

Commit messages are stable and short by design — they describe the path, not the content. The content is encrypted; you don't want to leak metadata about what changed.

Auto-push is opt-in per mount. When enabled, every commit is followed by a `git push` to the configured remote. A network failure here is non-fatal: the local store is consistent, the push is just deferred. `gopass sync` retries pending pushes.

Auto-pull is on `gopass sync` only; gopass does not pull on every show, because that would slow down trivial operations and require network connectivity.

Conflict resolution is a real problem. Two users editing the same secret on different machines produces a git merge conflict, and the conflict markers are inside encrypted bytes that look like random binary data. gopass's `.gitattributes` handling installs a custom merge driver:

```
*.gpg merge=gpg
*.age merge=age
```

The merge driver decrypts both sides (ours, theirs, base), runs a text merge on the plaintext (with conflict markers if there are real conflicts), then re-encrypts the result. This requires both branches to have valid keys for both files — which is normally true within a team.

For binary secrets (e.g. an SSH key file), there is no sensible auto-merge; the user has to pick one side.

## Templates

A `.template` file in a directory is a scaffold for new secrets created in that directory. For example, `~/.password-store/work/aws/.template`:

```
{{ .Password }}
account: {{ promptString "Account ID" }}
region: {{ promptString "Region" "us-east-1" }}
role: {{ promptString "Role" }}
mfa-arn: arn:aws:iam::{{ .Account }}:mfa/{{ .User }}
```

When you `gopass insert work/aws/foo`, gopass:

1. Finds `work/aws/.template`.
2. Generates a new password (length per `default_password_length`).
3. Runs the Go template engine over the file with `.Password` bound and prompts for the rest.
4. Hands the rendered text to the encryption pipeline.

This is a documentation pattern as much as a workflow: the template is the canonical answer to "what fields does an AWS account secret need."

## OTP

gopass stores OTP-token URIs as part of the secret body. The convention is:

```
my-password
otpauth://totp/Issuer:account?secret=BASE32SECRET&issuer=Issuer&algorithm=SHA1&digits=6&period=30
```

The `gopass otp foo` command parses the `otpauth://` URI and computes the current code:

- `totp`: TOTP per RFC 6238. Time-step `period` (default 30s), counter is `floor(unix_time / period)`.
- `hotp`: HOTP per RFC 4226. Counter is incremented on each use; the counter must be persisted back into the secret (and re-encrypted, and committed).

The HMAC algorithm is per the URI's `algorithm` parameter:

- `SHA1` — default; what Google Authenticator and most apps use.
- `SHA256`, `SHA512` — supported in spec but rare in practice.
- `BLAKE2b` — not in RFC 6238 but specified in some forks. gopass historically supported it; check current docs.

The truncation is HOTP's standard dynamic truncation: take the last 4 bits of the HMAC as offset, read 4 bytes starting at offset, mask the high bit, modulo `10^digits`.

The HOTP counter persistence is the gotcha: the counter must increment each `gopass otp` call, which means each call mutates the file, which means a git commit per OTP code request. This is why HOTP is rarely used in practice — TOTP is stateless on the client side.

## Audit / fsck

`gopass audit` runs heuristics over all decrypted secrets:

- Password length below threshold.
- Password reuse across secrets.
- Password in known-breach corpus (if HIBP integration enabled).
- Password older than N months (creation time from git log).

`gopass fsck` runs structural integrity checks on the *encrypted* store:

- Every file decrypts cleanly (catches corrupted files, missing recipients).
- Every file is encrypted to exactly the current `.gpg-id` set (catches stale recipients).
- No orphan files (files in git but not on disk, or vice versa).
- `.gpg-id` is consistent (no expired keys, no unknown fingerprints).
- The recipient list in each file's PKESK matches `.gpg-id`.

The recipient consistency check is the most operationally important: it catches cases where a recipient was added to `.gpg-id` but `gopass recipients update` was never run, leaving old files inaccessible to the new recipient.

`gopass fsck --decrypt` also catches files encrypted to subkeys that have since expired, where the parent key is still valid but the encryption subkey isn't.

## Sync

`gopass sync` is the one-shot reconciliation operation:

1. For each mount: `git pull --rebase`.
2. Resolve any merge conflicts (using the `.gitattributes` merge driver where possible).
3. For each mount: `git push`.
4. Optionally update plugins/templates.
5. Optionally re-key (`gopass recipients update`) if `.gpg-id` changed during the pull.

The order matters: pull first to incorporate remote changes, push second to broadcast local changes. If the remote diverged in a way the merge driver can't resolve, `gopass sync` aborts and the user must `cd` into the store and `git status` themselves.

For multi-remote setups (say, a personal mirror and a team mirror), gopass treats one remote as the upstream per mount. Other remotes are mirror-only.

## Aliases

The aliases registry maps short names to fully-qualified store paths:

```yaml
aliases:
  gh: personal/github.com
  aws: work/aws
  prod: work/aws/prod
```

`gopass show gh/alice` is rewritten to `gopass show personal/github.com/alice`. Aliases are local config (not synced via git); each user can have their own.

## Plugin Architecture

gopass plugins are external binaries that gopass invokes. Three official plugins:

**gopass-jsonapi** — a stdio-based JSON-RPC bridge. Browser extensions speak to gopass-jsonapi over a Native Messaging Host pipe; jsonapi answers password queries by shelling into the gopass binary. Each browser extension (Chrome, Firefox) installs a manifest pointing at gopass-jsonapi, plus an extension that sends `{"type":"query","query":"github"}` and receives matching credentials.

The threat model is non-trivial: any browser process that can speak Native Messaging can ask gopass-jsonapi for any secret. Mitigation: jsonapi prompts (via gpg-agent's pinentry) for confirmation on each query, and the extension is sandboxed to specific origins.

**gopass-summon-provider** — adapts gopass to HashiCorp Summon's secret-injection convention. Summon reads a `secrets.yml`, looks up each secret via a provider, exports the resolved values as env vars, and execs the target program. The gopass provider answers "give me the password at path foo" with the first line of `gopass show foo`.

This is how gopass gets used in CI/CD: write `secrets.yml`, decrypt at build time via summon, inject as env, never write to disk.

**gopass-fuse** — mounts the store as a FUSE filesystem at `~/passwords/`. Reading a path reads the decrypted content. Writing replaces it. This provides app compatibility (any tool that reads files can read passwords) at the cost of caching plaintext in the kernel's VFS — it's a tradeoff, not a strict win.

## Migration

From `pass`: zero-effort. `pass` and `gopass` share the `~/.password-store` layout exactly. Run `gopass init --path ~/.password-store --remote-only`, point at your existing pass store, and you're done. Both tools can coexist on the same store indefinitely; they only differ in feature set.

From KeePass / 1Password / LastPass: use `pass-import` or `gopass-import`. These tools parse the export format (KeePass XML, 1Password 1pif/csv, LastPass CSV) and create one gopass secret per entry. Caveats:

- LastPass CSV is lossy — attachments and per-account notes might not round-trip.
- 1Password 1pif preserves more metadata; the importer maps custom fields to YAML.
- KeePass XML preserves group hierarchy as gopass directory structure.

Always import to a *new* mount for testing, audit it, then graduate to your main mount.

## age Crypto Backend

When initialised with `gopass init --crypto age`, the store uses age instead of GPG:

- Recipients are stored in `.age-recipients` (one X25519 recipient per line) instead of `.gpg-id`.
- Files are `.age` instead of `.gpg`.
- The agent is `age` itself (no daemon); identity files live at `~/.config/gopass/age-identities` or via `SOPS_AGE_KEY` / `RAGE_IDENTITY` env.
- No web of trust; recipients are just public keys.

The why: age is small, modern, audited, and has no PGP-era cruft. No subkey expiration, no signing/encryption distinction, no mode bits, no compression issues. The whole tool is two files of well-tested cryptography.

The downsides:

- No smartcard support out of the box (use age-plugin-yubikey for PIV slot integration).
- No revocation. If a private key leaks, you must rotate every secret.
- No third-party trust signals; you trust each recipient's pubkey on first use.

For a homelab or personal-multi-machine store, age is the obvious modern choice. For a team store with smartcards and an existing GPG investment, GPG remains practical.

## TOFU SSH Workflow

A common pattern: store SSH key passphrases in gopass, unlock SSH keys at session start.

```bash
ssh-add < <(gopass show ssh/work-laptop | head -1 | cat -)
```

But this leaks the passphrase to ssh-add via stdin (kernel buffer, possibly visible in `/proc`). Better:

```bash
SSH_ASKPASS=$(which gopass-ssh-askpass) DISPLAY=dummy ssh-add ~/.ssh/work-laptop
```

where `gopass-ssh-askpass` is a small helper script:

```bash
#!/bin/sh
gopass show ssh/work-laptop | head -1
```

This avoids any process-list visibility of the passphrase.

For SSH agent itself, a useful pattern is the gpg-agent SSH support. `enable-ssh-support` in `~/.gnupg/gpg-agent.conf` makes gpg-agent serve as an SSH agent, with smartcard-backed SSH keys. Combined with gopass, you get: gopass holds the inventory and metadata, gpg-agent holds the live keys, smartcards hold the private bits.

## Threat Model

gopass protects:

- **Confidentiality of stored secrets** — encrypted at rest under recipient keys.
- **Integrity of stored secrets** — GPG signature verification on decrypt; age MAC.
- **Authentication of contributors** — git commit signing (if enabled).
- **Replay resistance** — GPG's PKESK packets include a session key per file.

gopass does not protect:

- **Plaintext in memory** — when you `gopass show foo`, the plaintext is in the process memory, then written to your terminal's scrollback, possibly your shell history, and possibly paged to swap. There is no zeroisation guarantee.
- **Plaintext on clipboard** — `gopass show -c foo` copies to the system clipboard. The clipboard is process-wide on most OSes; any other running app can read it. gopass schedules a clipboard clear at 45 seconds (configurable), but a fast-running malicious app can still read.
- **Metadata** — secret *names* and *paths* are not encrypted. An attacker reading your git repo learns that you have a `work/aws/prod/db-password` even if they can't decrypt it.
- **Recipient list** — `.gpg-id` is plaintext. An attacker can enumerate who can decrypt.
- **Git commit history** — old encrypted blobs persist forever. A removed recipient who has a clone retains decryption ability for files at the time of the clone.
- **Endpoint compromise** — if your laptop is compromised, your secrets are gone. gopass doesn't and can't defend a compromised endpoint.

The clipboard threat in particular is operationally relevant. A common mitigation is the `gopass copy --browser-extension` flow, which uses the browser's password autofill instead of the system clipboard.

## Operational Patterns

The team-shared store pattern:

1. Create a git repo on a private remote (Gitea, GitLab, GitHub private).
2. Each team member has a GPG key (preferably YubiKey-backed).
3. Each team member's fingerprint goes in `.gpg-id`.
4. Each team member exports their pubkey to `.public-keys/`.
5. CI/CD has its own GPG key (long-lived, less ceremony) added to `.gpg-id` for production secrets only.
6. New team members are added by:
   a. They send their pubkey.
   b. An existing member adds their fingerprint to `.gpg-id`, runs `gopass recipients update`, commits, pushes.
   c. They `git clone`, `gopass init --remote-only`, decrypt as themselves.
7. Departing team members trigger:
   a. Their fingerprint is removed from `.gpg-id`.
   b. `gopass recipients update` re-encrypts everything.
   c. **Every secret they had access to is rotated** (changed at the source). This is the essential step — re-encryption alone doesn't help if they kept a clone.
   d. Their GPG key may be revoked publicly (depends on whether it was a personal key or a work-only key).

Step 7c is the painful but unavoidable step. There is no cryptographic shortcut: if someone cloned your encrypted store and held it, they have all the data they will ever have access to. Removing them from future updates only stops new secrets, not old ones.

The recipient-rotation flow for high-stakes environments (fintech, regulated industries) often has a quarterly cadence regardless of personnel changes:

- Rotate every secret quarterly.
- Re-key the data via `gopass recipients update` quarterly.
- Audit the recipient list against HR roster quarterly.

For lower-stakes setups, "rotate when there's a personnel change" is acceptable.

### Recipient management edge cases

A subtle edge case: a recipient with a *subkey* expiration. GPG keys often have a master key (used for signing pubkey certifications) and an encryption subkey with its own expiration. If the encryption subkey expires but the master is still valid, gopass cannot encrypt to that recipient (no valid encryption capability), but the recipient can still decrypt files encrypted before expiration (because the *private* subkey is still on their machine, even if the public expiration says it's invalid). This produces confusing UX: "I can read old secrets but not write new ones."

The fix is to rotate the subkey: `gpg --edit-key FINGERPRINT` then `addkey`, generate a new encryption subkey, distribute the updated pubkey, run `gopass recipients update`. This is mechanically straightforward but socially expensive — every team member with a stale subkey blocks the team's encryption ability.

A related issue: if a recipient's GPG keyring has multiple encryption-capable subkeys (intentional or accidental), gopass picks one at encrypt time. Decryption tries each until one works. This is invisible in normal operation but matters during smartcard rotations: a YubiKey replacement creates a new subkey, the old one stays in the keyring, and decryption may try the now-unavailable old subkey first. The fix: `gpg --edit-key`, set the old subkey to revoked or expired, force GPG to skip it.

### Per-store crypto fan-out

Multi-mount stores let you mix crypto. A pragmatic combination:

- **default** (`~/.password-store`): age-only, single recipient (you), fast.
- **work** (`~/work/secrets`): GPG, team recipients, smartcard-backed.
- **infra** (`~/work/infra-secrets`): GPG, narrower team, pin-cached.
- **share** (`~/share`): age, recipients = trusted family/friends.

Each mount has independent backups, independent git remotes, independent rotation cadences. Cross-mount references are by mount-prefixed path: `gopass show work/aws/prod/db`.

The discovery problem: tab-completion only works within the current mount unless you explicitly enable cross-mount completion. The shell completion script registers each mount as an alias — typing `gopa show wo<TAB>` expands to `gopass show work/`. Fancier shells (zsh + zsh-completions) support fuzzy-search across mounts.

### Performance on large stores

A typical gopass store has hundreds to low-thousands of secrets. At that scale, every operation is fast. But at 10,000+ secrets:

- `gopass list` enumerates the entire tree; `find ~/.password-store -name '*.gpg'` is the underlying op. Sub-second on SSDs.
- `gopass search foo` decrypts every secret in matched paths. Linear in store size.
- Auto-completion may pre-load the path set; large stores can produce slow shell startups.
- `gopass fsck --decrypt` is O(N) decryptions; on slow gpg-agent (e.g. smartcards with touch-required), this is hours.

Mitigations:

- Partition into multiple mounts (search scope is per-mount).
- Use full-content search sparingly; prefer path-based lookup.
- Cache common decryptions in gpg-agent (long `default-cache-ttl`).
- For YubiKey users: don't make every operation require touch — set touch-policy on the *encryption* subkey only, not on the SSH-auth subkey, so non-decryption operations don't prompt.

### CI/CD integration patterns

gopass-as-CI-backend works in a few configurations:

**Read-only access from CI**: a dedicated GPG keypair lives only in CI. Its fingerprint is in the relevant `.gpg-id` files. The private key is provided to CI as a secret env var (encrypted by the CI system's own KMS). At job start, CI imports the key into a temporary keyring, runs `gopass show` for needed secrets, exports them as env vars or to temporary files, then discards the keyring at job end.

**Read-write from CI** (for rotating-secret jobs): same as above, but the CI keypair has write capability (its fingerprint is in `.gpg-id` for relevant paths) and CI can `gopass insert` new versions. This requires the CI to push to git, which means the CI has a deploy key for the password-store git remote.

**No-CI-access (preferred for high-stakes)**: secrets that prod CI needs are *not* in gopass. They're in a real secret store (Vault, KMS, AWS Secrets Manager). gopass holds operational secrets for humans; runtime secrets for services live elsewhere.

The third option is increasingly the consensus: gopass is for humans, dedicated secret stores are for services. The boundary is often the laptop-vs-server line.

### Browser integration trade-offs

The gopass-jsonapi + browser extension flow is convenient but has a unique threat surface:

- The native messaging host process (gopass-jsonapi) runs as the user.
- The browser extension speaks to it via stdio.
- The extension is sandboxed by the browser to specific origins.
- Each query produces a pinentry prompt (or uses cached gpg-agent state).

The risk vector: a malicious browser extension that gains the right manifest origins can request any password. Mitigations:

- Limit which origins the extension responds to (firefox-sources or chrome web store extension review).
- Require pinentry confirmation per query (cannot be batched silently).
- Prefer browser-pass (the extension doesn't request, the user explicitly fills via a popup).

For high-stakes credentials (banking, work admin), many users disable browser integration entirely and only allow command-line `gopass copy` to clipboard.

### Migration: gopass → 1Password / Bitwarden

The reverse-migration is rarer but real: a team graduates from self-hosted gopass to a managed password manager.

The process:

1. Decrypt every gopass secret to a flat YAML/CSV.
2. Run the target's import tool (1Password CLI's `op item create`, Bitwarden's `bw create item`).
3. Verify a sample of items.
4. Wipe the local plaintext intermediate (overwrite + delete).

Caveats:

- Notes and metadata round-trip imperfectly. Custom fields may be flattened to a single notes blob.
- TOTP URIs are usually preserved.
- Recipient history (who could see what) is lost; rebuilt in the target's sharing model.

A common reason for this migration: regulatory requirements that demand a vendor with audit logs, MFA enforcement, and SSO integration that gopass doesn't provide.

### Shared password recovery scenario

Worst-case: a key team member is hit by a bus, taking their YubiKey with them. Their `gopass` access is gone, but the team needs the secrets they had access to.

If the team practiced multi-recipient encryption (every prod secret encrypted to the team-shared key plus individual keys), recovery is trivial: any other team member with the team key can decrypt.

If the team relied on the bus-victim's individual key alone for some secrets, those secrets are unrecoverable in cryptographic terms. Recovery options:

- Reset the secrets at the source (change DB password, regenerate API key, etc.) — for everything they were the only encryptor of.
- Restore from a non-gopass backup (yes, you should have those for irreplaceable secrets).
- Accept the loss if it's a non-critical credential.

The practice that prevents this: every secret has at least two human recipients, plus a recovery recipient (offline key in safe).

### Format peculiarities

The first-line-is-password convention is by-convention, not by-syntax. gopass parses YAML if it sees YAML syntax in the file; otherwise treats line 1 as password and the rest as freeform.

A common pitfall: a password starting with `---` (which is YAML document start) confuses the parser. The fix: start the file with a non-YAML first character, or wrap the password in YAML format explicitly:

```
password: |-
  ---this-password-starts-with-dashes
```

Another pitfall: Unicode passwords. Most CLI consumers handle them fine, but Windows clipboard utilities, older browser fields, and shell tools differ. If you use Unicode in passwords, test the consumer.

### The audit-log gap

gopass does not log who decrypted what when. The git history shows who *modified* the store (commits with timestamps and authors), but reads are invisible. There's no equivalent of CloudTrail.

For some compliance regimes (SOC2 with Trust Services Criteria around access logging, certain HIPAA configurations), this is a hard blocker. The mitigation patterns:

- Add a wrapper script that logs gopass invocations to a separate audit log (still local-only — not tamper-proof).
- Use OS-level audit (auditd, OSQuery) to log gpg invocations.
- Migrate high-stakes secrets to a tool with built-in audit (Vault, 1Password Business, etc.).

For most teams, the read-audit gap is acceptable — the threat model assumes trusted users, and the value of a read-audit is mostly forensic (figuring out what an attacker exfiltrated post-incident).

### Backup strategies

gopass's native backup is `git push` to a remote. This is sufficient for laptop-loss scenarios, but doesn't help in two cases:

1. **Remote compromise**: if your git remote is breached, attackers get the encrypted store. They can't decrypt without keys, but they can analyse metadata and structure.
2. **Catastrophic key loss**: every recipient's keys are lost simultaneously (extremely unlikely but possible — natural disaster, simultaneous YubiKey theft).

Mitigations:

- Multiple geographically-distributed git remotes (one self-hosted, one third-party).
- Encrypted offline backups (e.g. `tar c ~/.password-store | age -r BACKUP_KEY > backup.tar.age`) on detachable media.
- Paper-printed key recovery sheets in a fireproof safe.

The "encrypted offline backup" is itself a sops/age problem — see the sops detail page for canonical patterns.

### Why filesystem-as-database wins

People sometimes ask: "why isn't gopass a SQLite database?" The arguments for filesystem:

- **Diff-friendly**: `git diff` shows exactly which secrets changed.
- **Portable**: any tool that reads files reads gopass — `gpg`, `pass`, custom shell scripts, file managers.
- **Concurrent access**: filesystem locking is per-file, so two simultaneous edits to different secrets don't conflict.
- **No corruption surface**: a SQLite file corruption is total; a filesystem corruption is per-file and recoverable from git.
- **Auditable**: directory listings show structure trivially.

The arguments for SQLite (which gopass rejected):

- Faster bulk operations (one SQLite query vs N file decryptions).
- Atomic transactions (multiple secrets updated together).
- Better metadata storage (per-secret access counts, last-used timestamps).

For a personal/team password manager, filesystem wins. For a global enterprise secret store, the trade-offs flip.

### Templating with secrets in deployment configs

A common pattern: render a templated config (Helm values, Ansible vars, Terraform tfvars) by injecting gopass-stored secrets. The naive form:

```bash
DB_PASSWORD=$(gopass show prod/db | head -1) envsubst < template.tf > rendered.tf
```

This works but leaks: the rendered file has the password, may end up in CI artifacts, may be cached. Better:

```bash
gopass-summon-provider show prod/db | summon -p gopass-summon-provider terraform plan
```

Summon evaluates `secrets.yml` at runtime, sets env vars for the duration of the child process, never writes plaintext to disk. The rendered tfvars are computed in-memory by terraform from environment variables.

For Kubernetes specifically: the Helm pattern is:

```yaml
# values-secret.yaml (gopass-encrypted via plugin)
db:
  password: !gopass prod/db
```

helm-secrets resolves the `!gopass` tags at install time, hands the resolved values to Helm. The chart ships ready-to-go without containing real secrets.

### The "two laptops" sync problem

A user with two laptops (work + personal, or home + travel) shares a gopass store via git. Both laptops have GPG keys; both fingerprints are in `.gpg-id`.

The sync flow:

- Laptop A modifies a secret, commits, pushes.
- Laptop B pulls. gopass auto-pulls on `gopass sync`, but not on every `gopass show`.

The pitfall: laptop B has stale `gopass show` output until next sync. For high-frequency rotation (say, daily), `gopass sync` should be cron'd or run on shell startup.

A second pitfall: simultaneous modifications. If both laptops edit the same secret without syncing first, the second push hits a non-fast-forward error. Resolution requires `git pull --rebase` then re-applying the local change. The merge driver (see §6) helps for textual content but can't reconcile two different password choices — one wins, the other is reverted.

For high-collaboration use, the discipline is: pull before edit, push after edit, never leave uncommitted edits overnight.

### Performance implications of the merge driver

The custom git merge driver for `.gpg` files decrypts both sides for every merge, which means every `git pull` on a divergent branch may decrypt dozens of files. On smartcard-backed setups with touch-required, this is a touch fest.

Mitigation: avoid divergent branches. Pull frequently. Use `git pull --rebase` which converts cross-branch merges into linear rebases (still requires decryption per conflicted file, but less work overall).

For very-high-collaboration scenarios (10+ people on one store), the merge cost becomes operationally significant, and teams sometimes adopt a "single committer" rule where one engineer owns the main branch and merges others' patches.

### Comparison: gopass vs other team password managers

| Feature | gopass | 1Password Business | Bitwarden Org |
|---------|--------|-------------------|---------------|
| Self-hosted | yes (any git remote) | no (vendor cloud) | yes (Vaultwarden) or no |
| End-to-end encrypted | yes | yes | yes |
| Audit log of reads | no | yes | partial |
| MFA enforced | gpg key only | yes (TOTP, FIDO2) | yes (multiple) |
| SSO integration | no | yes (SAML, OIDC) | yes (Enterprise) |
| Browser autofill | via plugin | native | native |
| Mobile apps | no | yes | yes |
| Smartcard support | yes (gpg) | partial | partial |
| Open-source | yes | no | partial (server) |
| Cost | free | per-user | per-user (free for self-host) |

gopass wins on: self-host, OSS, smartcard depth, customisation, CLI-first workflow.
1Password/Bitwarden win on: audit logging, MFA enforcement, mobile UX, browser UX, SSO.

For a small ops team comfortable with CLI and GPG, gopass is excellent. For a larger company with non-technical users and compliance auditors, a managed solution is usually right.

### Customising the secret format

gopass treats the first line as password and the rest as freeform metadata. But for richer use, define your own structure:

```
hunter2
---
title: GitHub
url: https://github.com
username: alice
notes: |
  recovery codes are stored elsewhere
otpauth: otpauth://totp/GitHub:alice?secret=ABC&issuer=GitHub
fields:
  - name: backup-email
    value: backup@example.com
```

The YAML block after `---` is parsed by tools that understand it. The first line stays the password. Custom tooling can leverage this for richer integrations.

### Hardware-token rotation runbook

When a YubiKey is lost or rotated:

1. Generate a new GPG key on the new YubiKey (`gpg --card-edit`, then `admin`, `generate`).
2. Note the new fingerprint.
3. On a non-affected machine, decrypt every secret in the affected stores (or write a script that uses gopass internally).
4. Edit `.gpg-id` to add the new fingerprint and remove the old.
5. Run `gopass recipients update` (or equivalently `gopass sync`).
6. Push to git remotes.
7. Revoke the old GPG key publicly (publish revocation certificate).
8. **Reset all secrets the old YubiKey holder had access to** — re-encryption alone doesn't help if the YubiKey was stolen and wasn't PIN-protected.

Step 8 is the painful step. If the YubiKey had a strong PIN with retry-lockout, an attacker without the PIN can't extract anything; rotation is precautionary. If the PIN was weak or absent, every secret must be assumed compromised.

### Closing the loop

The gopass philosophy is "small tool, do one job." It's a frontend over `pass`-style stores, with team features added. It does not aspire to be the world's best password manager, but it's the world's best UNIX-philosophy password manager, and for the right workflow (CLI-heavy, git-fluent, GPG/age-comfortable users), nothing else matches its ergonomics.

### Final synthesis

gopass succeeds because it accepts limits. It doesn't try to be a SaaS, doesn't try to manage MFA, doesn't try to onboard non-technical users. Within the boundary it draws — terminal users, GPG/age familiar, git-comfortable, willing to manage their own keys — it's near-perfect.

The architecture is honest: filesystem-as-database means tools you already have (find, grep, ls, git) work. Encryption is delegated to gpg/age, which are themselves audited. Sync is delegated to git, which is bulletproof. Each layer is the right tool for that layer.

The cost is that gopass inherits the foot-guns of every layer: GPG's subkey-expiration confusions, git's merge-conflict subtleties, the filesystem's lack of audit logging. For the target user, these are acceptable; for a non-technical user, any one of them is a barrier.

The trend toward age (over GPG) and toward managed cloud secret stores (over self-hosted gopass) is real and accelerating. But for individual users, small ops teams, homelab enthusiasts, and infosec-conscious developers, gopass remains the best-fit tool: free, open, self-hosted, and built on primitives the user already understands.

The lesson gopass teaches: composition beats integration. A tool that combines filesystem + git + gpg loses to a tool that delegates each to the canonical implementation. Build interfaces, not implementations. Trust the ecosystem.

That ethos — composition over integration, simplicity over features, transparency over magic — is what makes gopass durable in a market full of flashier competitors. It will outlast many of them, because it has fewer moving parts to break, fewer abstractions to leak, and fewer business models to pivot.

## References

- gopass project — `https://www.gopass.pw/`
- gopass GitHub — `https://github.com/gopasspw/gopass`
- Original `pass(1)` — `https://www.passwordstore.org/`
- `pass(1)` man page — Jason Donenfeld
- GnuPG — `https://gnupg.org/`
- age — `https://age-encryption.org/v1`
- RFC 4880 — OpenPGP Message Format
- RFC 4226 — HOTP
- RFC 6238 — TOTP
- RFC 7253 — OCB Authenticated-Encryption (used by some GPG variants)
- RFC 5869 — HMAC-based Extract-and-Expand Key Derivation Function
- HashiCorp Summon — `https://github.com/cyberark/summon`
- Native Messaging — Mozilla MDN docs
- pass-import — `https://github.com/roddhjav/pass-import`
- age-plugin-yubikey — `https://github.com/str4d/age-plugin-yubikey`
- Filippo Valsorda, "age design" — `https://age-encryption.org/`
- gpg-agent man page — GnuPG project
- `~/.gnupg/gpg-agent.conf` reference — GnuPG manual
- "Modern key management with pass" — various blog posts in the password-store ecosystem
- "How to use gopass with YubiKey" — community guides
- HIBP API — `https://haveibeenpwned.com/API/v3`
- gopass JSON API protocol — gopass repository docs
- gopass-fuse — gopass repository docs
- Browser-pass — Native Messaging extension for Firefox/Chrome
- gpgsm — S/MIME variant of gpg, occasionally relevant
- libgcrypt — the underlying crypto library used by GPG
- "Why I use pass" — Jason Donenfeld blog post
