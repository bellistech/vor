# GPG / OpenPGP (Encryption, Signing, Web of Trust)

The canonical OpenPGP implementation: long-term identity keys, encryption to recipients, detached signatures, gpg-agent for caching, smartcard/Yubikey integration, WKD for modern key discovery — covering GnuPG 2.4+ with notes on 2.2 LTS.

## Setup

On modern systems, `gpg` and `gpg2` refer to the same binary — GnuPG 2.x. The legacy 1.4 line is unmaintained. If your distribution still ships `gpg1`, that is the legacy line; `gpg2` (often symlinked to `gpg`) is what you want.

```bash
gpg --version
# gpg (GnuPG) 2.4.5
# libgcrypt 1.10.3
# Copyright (C) 2024 g10 Code GmbH
# License GNU GPL-3.0-or-later
# Home: /Users/alice/.gnupg
# Supported algorithms:
# Pubkey: RSA, ELG, DSA, ECDH, ECDSA, EDDSA
# Cipher: IDEA, 3DES, CAST5, BLOWFISH, AES, AES192, AES256, TWOFISH, CAMELLIA128, CAMELLIA192, CAMELLIA256
# Hash: SHA1, RIPEMD160, SHA256, SHA384, SHA512, SHA224
# Compression: Uncompressed, ZIP, ZLIB, BZIP2
```

The `Home:` line shows where keyrings live. Override via `GNUPGHOME`:

```bash
export GNUPGHOME=/path/to/alt/.gnupg
gpg -k                           # uses the alternate homedir
```

Install on the major platforms:

```bash
# macOS (recommended)
brew install gnupg pinentry-mac

# Debian/Ubuntu
sudo apt install gnupg gnupg-agent scdaemon pcscd

# Arch
sudo pacman -S gnupg pcsclite ccid

# Fedora
sudo dnf install gnupg2 gnupg2-smime pcsc-lite

# Alpine
apk add gnupg gnupg-scdaemon
```

LTS vs current:

- GnuPG 2.2.x is the LTS line (security updates only). Many enterprise distros pin to 2.2 — it lacks Argon2 and some 25519 ergonomics but is otherwise fully featured.
- GnuPG 2.4.x is the current series. Argon2 is the default for symmetric KDF, AEAD-OCB and chacha20 are first-class, ed25519/cv25519 generation is fast and clean.
- GnuPG 2.5/2.6 is the development tree leading toward RFC 9580 (the OpenPGP "crypto refresh").

Homedir layout:

```bash
ls ~/.gnupg/
# gpg.conf                  user config
# gpg-agent.conf            agent config
# pubring.kbx               keybox: public keys (modern format)
# trustdb.gpg               trust database
# private-keys-v1.d/        directory: one file per private key (keygrip-named)
# crls.d/                   certificate revocation lists
# openpgp-revocs.d/         auto-generated revocation certificates per key
# random_seed               PRNG state
# S.gpg-agent               UNIX socket to gpg-agent
# S.gpg-agent.ssh           UNIX socket for ssh-agent emulation (if enabled)
# S.gpg-agent.extra         remote-friendly socket
```

Permissions matter. The `~/.gnupg` directory must be `0700` and files must be `0600` or gpg will warn or refuse:

```bash
chmod 700 ~/.gnupg
find ~/.gnupg -type f -exec chmod 600 {} \;
find ~/.gnupg -type d -exec chmod 700 {} \;
```

## The OpenPGP Conceptual Model

OpenPGP (RFC 4880, updated by RFC 9580) defines a packet-based message format and a long-term identity key model. The pieces:

- **Primary key** — the long-term identity. Fingerprint identifies you. Capability is typically Certify (`C`), sometimes also Sign.
- **Subkeys** — short-term operational keys bound to the primary by a binding signature. Capabilities: Sign (`S`), Encrypt (`E`), Authenticate (`A`).
- **User IDs** — `Name (Comment) <email>` strings bound to the primary by self-signatures.
- **Keyrings** — `pubring.kbx` holds public keys; `private-keys-v1.d/` holds secret material; `trustdb.gpg` holds the WoT graph.
- **Keyserver vs WKD** — keyservers are bulletin boards; WKD (Web Key Directory) publishes via HTTPS at `.well-known` paths and is the modern recommendation.
- **Web of Trust** — bidirectional certifications: Alice signs Bob's UID after verifying his fingerprint in person.

Modern recommended layout for a new key:

```text
ed25519 primary key       cap: cert (and often sign)     1y expiry
 ├─ cv25519 subkey         cap: encrypt                   1y expiry
 ├─ ed25519 subkey         cap: sign                      1y expiry
 └─ ed25519 subkey         cap: authenticate              1y expiry (for SSH)
```

The primary stays offline (USB or paper). The three subkeys live on the daily-driver host or, better, on a Yubikey 5. Yearly subkey rotation forces a regular health check and limits blast radius if a host is compromised.

Fingerprint formats:

- 20-byte SHA-1 — classic v4 fingerprint, 40 hex chars (`AB12 CD34 ...`).
- 32-byte SHA-256 — v5/v6 fingerprint per RFC 9580, 64 hex chars. Not yet universal as of GnuPG 2.4.

Always exchange and verify the full fingerprint, never the short keyid (last 8 hex chars) or even the long keyid (last 16 hex chars) — both are collidable.

## Key Generation — Quick

The `--quick-generate-key` family is non-interactive and the right tool for scripted or repeatable setups. Generate the primary, then add subkeys explicitly:

```bash
# Primary: ed25519, certify-only, 1-year expiry
gpg --quick-generate-key "Alice Smith <alice@example.com>" ed25519 cert 1y

# After generation, capture the fingerprint
FPR=$(gpg --list-keys --with-colons alice@example.com \
  | awk -F: '/^fpr:/ {print $10; exit}')

# Encryption subkey (cv25519)
gpg --quick-add-key "$FPR" cv25519 encr 1y

# Signing subkey (ed25519)
gpg --quick-add-key "$FPR" ed25519 sign 1y

# Authentication subkey (ed25519, for SSH)
gpg --quick-add-key "$FPR" ed25519 auth 1y
```

Capability strings for `--quick-generate-key` and `--quick-add-key`:

- `cert` — certify other keys (the default for primary keys; use `cert,sign` if you also want the primary to sign data)
- `sign` — sign data
- `encr` — encrypt data
- `auth` — authenticate (used by ssh-agent emulation)

Passphrase prompts: `--quick-generate-key` will pop a pinentry prompt. To pre-supply (CI, batch use):

```bash
gpg --batch --pinentry-mode loopback \
    --passphrase 'correct horse battery staple' \
    --quick-generate-key "Alice Smith <alice@example.com>" ed25519 cert 1y
```

For an unprotected key (testing only — never for real identity keys):

```bash
gpg --batch --pinentry-mode loopback --passphrase '' \
    --quick-generate-key "CI Bot <ci@example.com>" ed25519 cert,sign 1y
```

## Key Generation — Full

`--full-generate-key` is the interactive ceremony with all knobs exposed:

```bash
gpg --full-generate-key
```

The legacy menu:

```text
Please select what kind of key you want:
   (1) RSA and RSA
   (2) DSA and Elgamal
   (3) DSA (sign only)
   (4) RSA (sign only)
   (7) DSA (set your own capabilities)
   (8) RSA (set your own capabilities)
   (9) ECC and ECC                     <-- pick this
  (10) ECC (sign only)
  (11) ECC (set your own capabilities)
  (13) Existing key
  (14) Existing key from card
Your selection? 9
```

After choosing `9` (ECC and ECC):

```text
Please select which elliptic curve you want:
   (1) Curve 25519                    <-- pick this
   (3) NIST P-256
   (4) NIST P-384
   (5) NIST P-521
   (6) Brainpool P-256
   (7) Brainpool P-384
   (8) Brainpool P-512
   (9) secp256k1
Your selection? 1
```

Then expiry:

```text
Please specify how long the key should be valid.
         0 = key does not expire
      <n>  = key expires in n days
      <n>w = key expires in n weeks
      <n>m = key expires in n months
      <n>y = key expires in n years
Key is valid for? (0) 1y
```

Recommended: 1y or 2y, never 0. Expiry is renewable; if you lose the primary, expiry self-revokes the key.

User ID:

```text
Real name: Alice Smith
Email address: alice@example.com
Comment:                              <-- usually leave blank; comments confuse verifiers
You selected this USER-ID:
    "Alice Smith <alice@example.com>"
```

Pinentry will prompt for the passphrase. Choose a long, memorable passphrase; gpg-agent will cache it.

## Listing Keys

The most-used incantations:

```bash
gpg -k                                   # list public keys
gpg --list-keys                          # same
gpg -K                                   # list secret keys
gpg --list-secret-keys                   # same

# Show full 16-char (long) keyids in addition to UIDs
gpg --list-keys --keyid-format long

# Show fingerprints alongside keys (canonical full info)
gpg --list-keys --keyid-format long --with-fingerprint

# Verbose: show subkeys, expiry, capabilities, keygrip
gpg -v -k
gpg -vv -k                               # full subkey detail

# Machine-readable, colon-separated
gpg --list-keys --with-colons

# Show only fingerprints (one per key)
gpg --list-keys --with-colons | awk -F: '/^fpr:/ {print $10}'
```

`--keyid-format` accepts:

- `none` — hide keyids entirely (clean UI)
- `0xshort` — 8-hex-char short ID prefixed with `0x` (collidable, do not use)
- `0xlong` — 16-hex-char long ID prefixed with `0x` (still collidable in theory, but rarely)
- `long` — 16-hex-char long ID without prefix
- `short` — 8-hex-char short ID without prefix

Sample output:

```text
$ gpg --list-keys --keyid-format long
/Users/alice/.gnupg/pubring.kbx
-------------------------------
pub   ed25519/0x1A2B3C4D5E6F7A8B 2026-04-01 [C] [expires: 2027-04-01]
      0123456789ABCDEF0123456789ABCDEF12345678
uid                   [ultimate] Alice Smith <alice@example.com>
sub   cv25519/0xAABBCCDDEEFF0011 2026-04-01 [E] [expires: 2027-04-01]
sub   ed25519/0x2233445566778899 2026-04-01 [S] [expires: 2027-04-01]
sub   ed25519/0xCCDDEEFFAABBCCDD 2026-04-01 [A] [expires: 2027-04-01]
```

Capability flags in brackets: `C`=certify, `S`=sign, `E`=encrypt, `A`=authenticate. Trust column shows `[ultimate]`, `[full]`, `[marginal]`, `[unknown]`, or `[never]`.

The `--with-colons` machine output is what every script should parse; the human format is unstable across versions. Field reference:

```bash
gpg --list-keys --with-colons alice@example.com
# tru::1:1714000000:0:3:1:5
# pub:u:255:22:1A2B3C4D5E6F7A8B:1711929600:1743465600::u:::cC::::::ed25519::0:
# fpr:::::::::0123456789ABCDEF0123456789ABCDEF12345678:
# uid:u::::1711929600::ABCDEF...::Alice Smith <alice@example.com>::::::::::0:
# sub:u:255:18:AABBCCDDEEFF0011:1711929600:1743465600:::::e::::::cv25519::
```

Per `man gpg`, the field meanings are: type, validity, key-length, pubkey-algo, keyid, creation-date, expiry-date, certificate, owner-trust, user-id, signature-class, key-capabilities, issuer, flag, token, hash-algo, curve-name, ... .

## Key Inspection

Inspect a key file before importing to learn what's inside:

```bash
gpg --show-keys public.asc
# pub   ed25519 2026-04-01 [C] [expires: 2027-04-01]
#       0123456789ABCDEF0123456789ABCDEF12345678
# uid                      Alice Smith <alice@example.com>
# sub   cv25519 2026-04-01 [E] [expires: 2027-04-01]
```

Show fingerprint(s):

```bash
gpg --fingerprint alice@example.com
gpg --fingerprint --fingerprint alice@example.com    # also show subkey fingerprints
```

Raw packet structure (the deepest debug tool):

```bash
gpg --list-packets public.asc
# :public key packet:
#         version 4, algo 22, created 1711929600, expires 0
#         pkey[0]: ED25519
#         pkey[1]: ...
#         keyid: 1A2B3C4D5E6F7A8B
# :user ID packet: "Alice Smith <alice@example.com>"
# :signature packet: algo 22, keyid 1A2B3C4D5E6F7A8B
#         version 4, created 1711929600, md5len 0, sigclass 0x13
#         digest algo 10, begin of digest e1 a4
#         hashed subpkt 33 len 21 (issuer fpr v4 0123...)
#         hashed subpkt 2 len 4 (sig created 2026-04-01)
#         ...
```

Export-then-inspect (handy when a key is in your keyring but you want the wire form):

```bash
gpg --export alice@example.com | gpg --list-packets
```

Show all of gpg's compiled-in defaults:

```bash
gpg --list-config
gpg --list-config sigexpire,certexpire,group,curve,pubkey,cipher,digest,compress,digestname,ciphername
```

Strip third-party signatures during export (clean key for upload):

```bash
gpg --export-options export-minimal --export alice@example.com > alice-clean.gpg
```

## Key Editing

`--edit-key` drops you into an interactive shell with full surgical access:

```bash
gpg --edit-key alice@example.com
```

The prompt is `gpg>`. Common commands:

```text
list      show key(s) and subkeys
fpr       show full fingerprint
uid N     select user ID number N (toggle)
key N     select subkey number N (toggle); * marker means selected
expire    change expiry of selected key/subkey (or primary if none selected)
passwd    change passphrase
addkey    add a subkey
addrevoker  designate another key that can revoke this one
adduid    add a new user ID
deluid    delete the selected user ID
revuid    revoke the selected user ID (preserves history)
sign      sign (certify) selected user IDs (exportable)
lsign     local signature, non-exportable
tsign     trust signature (delegated trust to a CA-style key)
trust     set owner trust (1-5 menu)
toggle    swap between public-key and secret-key view
showpref  show preference list of selected UID
setpref   set algorithm preferences
pref      show packed preference list
clean     remove unusable signatures and unselected UIDs
minimize  same as clean but more aggressive
revsig    revoke a signature on the selected UID
revkey    revoke a subkey
addphoto  add a JPEG photo UID
showphoto display attached photo UIDs
save      write changes and exit
quit      discard and exit (will warn first)
```

The "rotate subkey expiry" workflow:

```bash
gpg --edit-key alice@example.com
gpg> list
gpg> key 1                       # select subkey 1 (asterisk appears)
gpg> key 2                       # also select subkey 2
gpg> key 3                       # also select subkey 3
gpg> expire
Key is valid for? (0) 1y
# (passphrase prompt for primary)
gpg> save
```

Then re-export and republish:

```bash
gpg --armor --export alice@example.com > alice-pub.asc
gpg --keyserver hkps://keys.openpgp.org --send-keys "$FPR"
```

## Subkey Strategy

The canonical OpenPGP layout in 2026:

1. **Primary key**: ed25519 with `cert` capability only (or `cert,sign` if you want to sign with the primary occasionally). Lives on offline media — encrypted USB stick, paper backup, air-gapped laptop. Used for: certifying other keys, extending subkey expiry, revocation.
2. **Encryption subkey**: cv25519, capability `encr`. Lives on daily host or Yubikey.
3. **Signing subkey**: ed25519, capability `sign`. Lives on daily host or Yubikey.
4. **Authentication subkey**: ed25519, capability `auth`. Used by gpg-agent's ssh-agent emulation.

Why bother:

- If your laptop is stolen, attacker gets only subkeys. You boot the offline primary, revoke the subkeys, generate fresh ones, republish. Identity preserved.
- Certification is the long-term commitment ("this UID belongs to me"). Encryption and signing are operational concerns that should rotate.

Yearly cadence: every spring, mount the offline primary, extend (or rotate) subkeys, re-export, push to keyservers + WKD, distribute. Make it a calendar event.

Yubikey upgrade path:

- Generate primary on offline laptop.
- Generate subkeys (or `keytocard` existing ones) directly to Yubikey.
- Backup Yubikey: a second Yubikey with identical subkeys, locked in a safe.

## Export and Backup

Public key export — distribute freely:

```bash
gpg --armor --export alice@example.com > alice-pub.asc
gpg --export alice@example.com > alice-pub.gpg          # binary form
```

Private key export — guard at all costs:

```bash
# ALL secret material including primary
gpg --armor --export-secret-keys alice@example.com > alice-secret-FULL.asc

# Secret subkeys WITHOUT primary -- the canonical daily-driver export
gpg --armor --export-secret-subkeys alice@example.com > alice-secret-sub.asc
```

Owner-trust DB (separate from key material):

```bash
gpg --export-ownertrust > alice-trust.txt
```

Re-import on a new machine:

```bash
gpg --import alice-pub.asc
gpg --import alice-secret-sub.asc
gpg --import-ownertrust < alice-trust.txt
```

Offline-primary backup bundle (the canonical "lock in safe" tarball):

```bash
mkdir -p alice-backup-$(date +%F)
cd alice-backup-$(date +%F)

gpg --armor --export "$FPR"                > pub.asc
gpg --armor --export-secret-keys "$FPR"    > secret-FULL.asc
gpg --armor --export-secret-subkeys "$FPR" > secret-sub.asc
gpg --export-ownertrust                    > ownertrust.txt
gpg --gen-revoke "$FPR"                    > revoke.asc

cd ..
tar czf alice-backup-$(date +%F).tar.gz alice-backup-$(date +%F)/
gpg --symmetric --cipher-algo AES256 alice-backup-$(date +%F).tar.gz
shred -u alice-backup-$(date +%F).tar.gz
rm -rf alice-backup-$(date +%F)
```

Keep `alice-backup-*.tar.gz.gpg` on two separate offline media (USB + paper QR), and keep the symmetric passphrase in a password manager or printed in a safe.

## Import

Import from a file:

```bash
gpg --import alice-pub.asc
gpg --import-options import-show alice-pub.asc       # preview before commit (older gpg)
gpg --show-keys alice-pub.asc                        # modern preview
gpg --import-options merge-only alice-pub.asc        # only update existing keys
gpg --import-options keep-ownertrust alice-pub.asc   # don't reset ownertrust
```

Fetch from a keyserver:

```bash
gpg --keyserver hkps://keys.openpgp.org --recv-keys "$FPR"
gpg --recv-keys "$FPR"                                # uses --keyserver from gpg.conf
```

Locate via the auto-key-locate chain (WKD → DANE → keyserver):

```bash
gpg --auto-key-locate wkd,dane,keyserver --locate-keys alice@example.com
gpg --locate-external-keys alice@example.com         # never look at local keyring
```

Refresh all keys against keyservers (catches revocations and new subkeys):

```bash
gpg --refresh-keys
gpg --keyserver hkps://keys.openpgp.org --refresh-keys
```

## Revocation Certificate

Generate this **immediately** after creating a key. It is the only way to revoke a key whose passphrase or secret material you have lost.

```bash
gpg --output revoke-alice.asc --gen-revoke alice@example.com
# > Create a revocation certificate for this key? (y/N) y
# > Please select the reason for the revocation:
# >   0 = No reason specified
# >   1 = Key has been compromised
# >   2 = Key is superseded
# >   3 = Key is no longer used
# >   Q = Cancel
# > Your decision? 0
# > Enter an optional description; end it with an empty line:
# > > (empty)
# > Is this okay? (y/N) y
```

Note: GnuPG 2.1+ also auto-generates a revocation cert during key creation, stored at `~/.gnupg/openpgp-revocs.d/<FPR>.rev`. Never lose it.

The recovery procedure:

```bash
# You lost everything except revoke.asc and the public key on keyservers
gpg --import revoke-alice.asc
gpg --keyserver hkps://keys.openpgp.org --send-keys "$FPR"
# The keyserver now propagates the revocation worldwide.
```

Print the revocation cert and store on paper in a safe. It is short enough for a single page when ASCII-armored.

## Trust

Owner-trust is your statement about how careful another person is with their certifications. Set via `--edit-key` then `trust`:

```text
gpg> trust
Please decide how far you trust this user to correctly verify
other users' keys (by looking at passports, checking fingerprints
from different sources, etc.)

  1 = I don't know or won't say
  2 = I do NOT trust
  3 = I trust marginally
  4 = I trust fully
  5 = I trust ultimately
  m = back to the main menu

Your decision? 5
```

Levels:

- `ultimate` — only your own keys. Anything signed by this key is treated as fully valid.
- `full` — you trust this person to certify others well. One signature from a `full` is enough to validate.
- `marginal` — you trust them somewhat. Multiple `marginal` signatures combine to validate.
- `unknown` — neutral.
- `never` — explicit distrust; signatures from this key never validate others.

The math is in `gpg.conf`:

```ini
marginals-needed 3       # need 3 marginal signatures to validate a key
completes-needed 1       # OR 1 fully-trusted signature
max-cert-depth   5       # max chain length
```

Alternative trust models (`--trust-model`):

- `pgp` — classic web of trust (default). Computes validity from signatures + ownertrust.
- `classic` — like `pgp` but ignores trust signatures.
- `direct` — only directly-set per-key trust matters; no transitive computation.
- `tofu` / `tofu+pgp` — Trust On First Use; track key-to-UID mappings over time and warn on changes.
- `always` — every key is fully valid. **Dangerous**: only for testing or non-security uses.
- `auto` — let gpg pick (currently equivalent to `pgp` unless `tofu` is configured).

Set per-invocation:

```bash
gpg --trust-model always --encrypt -r alice@example.com file.txt
```

## Signing Keys (Web of Trust Ceremony)

The classical key-signing ceremony:

1. Meet in person (or join a key-signing party).
2. Each person prints their fingerprint and a government-issued ID.
3. You verify their photo ID.
4. They read their fingerprint aloud; you compare against the printed copy.
5. Note their email address(es).

After the meeting, at home:

```bash
# Fetch their key
gpg --recv-keys 0123456789ABCDEF0123456789ABCDEF12345678

# Verify the fingerprint matches your printed copy
gpg --fingerprint bob@example.com

# Sign their UIDs (exportable certification)
gpg --edit-key bob@example.com
gpg> sign         # signs all UIDs; use uid N first to sign only specific UIDs
gpg> save

# Send the certified key back to them privately, OR upload to keyserver
gpg --armor --export bob@example.com | gpg -ear bob@example.com > bob-signed.asc.gpg
# (encrypt to bob; he imports and decides whether to publish)

# OR push to keyserver as community service
gpg --keyserver hkps://keys.openpgp.org --send-keys 0123456789ABCDEF0123456789ABCDEF12345678
```

Variants:

```bash
gpg --sign-key bob@example.com         # non-interactive sign-all
gpg --lsign-key bob@example.com        # local-only signature, never exported
```

`lsign` is the right choice when you trust a key for personal use but don't want to make a public statement. `tsign` (trust signature) is rare — used to delegate trust authority, e.g. "I trust the CA at Acme Corp to certify all *@acme.com UIDs":

```bash
gpg --edit-key acme-ca@acme.com
gpg> tsign
# > Please enter the depth of this trust signature: 2
# > Please enter a domain to restrict this signature: acme.com
gpg> save
```

Modern note: keys.openpgp.org **does not redistribute third-party signatures**. Your sign-key act creates a useful local cert in your keyring, but the keyserver will strip it. Distribute signed keys directly to the owner instead.

## Encryption

Encrypt to a single recipient (output is binary `file.gpg`):

```bash
gpg --encrypt --recipient bob@example.com file.txt
gpg -e -r bob@example.com file.txt                     # short form
gpg -er bob@example.com file.txt                       # shortest
```

ASCII-armored output (`file.txt.asc`, suitable for email/paste):

```bash
gpg --armor --encrypt --recipient bob@example.com file.txt
gpg -ea -r bob@example.com file.txt
gpg -ear bob@example.com file.txt
```

Recipient identifiers — try in this order:

```bash
gpg -er 'bob@example.com' file.txt                     # email (must be UID)
gpg -er 0xAABBCCDDEEFF0011 file.txt                    # 16-hex long keyid
gpg -er 0123456789ABCDEF0123456789ABCDEF12345678 file.txt   # full fingerprint (best)
gpg -er Bob file.txt                                   # name substring
```

Multi-recipient (any can decrypt):

```bash
gpg -e -r alice -r bob -r carol file.txt
```

Hide recipient keyids in the output (for traffic analysis resistance):

```bash
gpg --throw-keyids -e -r bob@example.com file.txt
# Recipients become "anonymous" in the packet header. Decryptors must try every secret key.
```

Encrypt to self automatically (always include yourself as recipient):

```bash
# In gpg.conf:
default-recipient-self
encrypt-to alice@example.com    # always add this recipient
```

Batch encrypt many files:

```bash
gpg --encrypt-files -r bob@example.com *.txt          # produces *.txt.gpg per file
```

Stream encryption from stdin to stdout:

```bash
echo "secret" | gpg -ear bob@example.com > secret.asc
tar czf - srcdir/ | gpg -e -r bob@example.com > srcdir.tar.gz.gpg
```

## Symmetric Encryption

No keys, just a passphrase:

```bash
gpg --symmetric file.txt                              # produces file.txt.gpg
gpg -c file.txt                                       # short form
gpg -ca file.txt                                      # ASCII-armored
gpg -c --cipher-algo AES256 file.txt                  # explicit cipher
```

GnuPG 2.4+ defaults to Argon2 KDF for symmetric encryption, which is excellent. To make it explicit (or to ensure modern parameters):

```bash
gpg --symmetric \
    --s2k-mode 4 \
    --s2k-cipher-algo AES256 \
    --s2k-digest-algo SHA512 \
    --s2k-count 65011712 \
    file.txt
```

S2K mode reference:

- `0` — simple iterated single-hash (legacy, weak)
- `1` — salted single-hash
- `3` — iterated and salted (the GnuPG 2.2 LTS default; with `--s2k-count` ≥ 65M)
- `4` — Argon2 (GnuPG 2.4+ default; modern memory-hard KDF)

Decrypt:

```bash
gpg --decrypt file.txt.gpg > file.txt                 # passphrase prompt
gpg -d file.txt.gpg                                   # to stdout
```

For new symmetric-encryption use cases (no signing, no recipients), prefer `age` — much simpler API, no UID/keyserver baggage, deterministic key files.

## Decryption

Decrypt to stdout:

```bash
gpg --decrypt file.txt.gpg
gpg -d file.txt.gpg
```

Decrypt to a specific output file:

```bash
gpg -o out.txt --decrypt file.txt.gpg
gpg -o out.txt -d file.txt.gpg
```

Redirect (works because `-d` writes to stdout when no output specified):

```bash
gpg -d file.txt.gpg > out.txt
```

Batch decrypt:

```bash
gpg --decrypt-files *.gpg                              # produces matching files without .gpg
```

List recipients without decrypting (handy when you don't have the secret key but want to know who can):

```bash
gpg --list-only --decrypt file.txt.gpg
gpg --list-packets file.txt.gpg                        # deepest detail
```

Non-interactive decrypt (CI):

```bash
gpg --batch --pinentry-mode loopback \
    --passphrase-file /path/to/passfile \
    --decrypt file.txt.gpg > out.txt
```

For asymmetric decrypt where the private key has a passphrase, the agent caches it after the first prompt. To force a fresh prompt:

```bash
gpgconf --reload gpg-agent          # gentler than --kill
```

## Detached Signatures

The canonical "release tarball + signature" pattern. The signature lives in a separate `.sig` (binary) or `.asc` (armored) file:

```bash
# Binary detached signature -> file.sig
gpg --sign --detach-sign file.tar.gz
gpg --detach-sign file.tar.gz
gpg -bs file.tar.gz                   # short form

# ASCII-armored detached signature -> file.asc
gpg --sign --detach-sign --armor file.tar.gz
gpg --armor --detach-sign file.tar.gz
gpg -bsa file.tar.gz
```

Pick the signing key explicitly when multiple secret keys exist:

```bash
gpg --local-user alice@example.com --detach-sign file.tar.gz
gpg -u alice@example.com -bsa file.tar.gz
```

Distribute the data file and the signature side by side. Verifiers run:

```bash
gpg --verify file.tar.gz.sig file.tar.gz
gpg --verify file.tar.gz.asc file.tar.gz
```

Note: when running `gpg --verify file.sig` without naming the data file, gpg looks for the data file by stripping `.sig`/`.asc`/`.gpg` from the signature filename.

## Inline Signatures

A clearsigned text file embeds the signature in human-readable form:

```bash
gpg --clear-sign message.txt          # produces message.txt.asc
gpg --clearsign message.txt           # same
```

The output:

```text
-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA512

This is the signed message body. Whitespace and line endings are
canonicalised before hashing.
-----BEGIN PGP SIGNATURE-----

iHUEARYKAB0WIQT...
=AbCd
-----END PGP SIGNATURE-----
```

Inline signed binary (signature wrapped around data):

```bash
gpg --sign file.bin                   # produces file.bin.gpg, binary
gpg --sign --armor file.bin           # produces file.bin.asc, armored
gpg -sa file.bin                      # short
```

Verify:

```bash
gpg --verify message.txt.asc          # clearsigned
gpg --verify file.bin.asc             # wrapped
gpg --decrypt file.bin.asc > file.bin # extract data and verify in one step
```

## Signature Verification

Detached:

```bash
gpg --verify file.tar.gz.sig file.tar.gz
```

Inline / clearsigned:

```bash
gpg --verify message.txt.asc
```

Batch verify many:

```bash
gpg --verify-files *.sig
```

Status output:

```text
gpg: Signature made Wed Apr  1 09:15:32 2026 PDT
gpg:                using EDDSA key 0123456789ABCDEF0123456789ABCDEF12345678
gpg: Good signature from "Alice Smith <alice@example.com>" [ultimate]
gpg:                 issuer "alice@example.com"
Primary key fingerprint: 0123 4567 89AB CDEF 0123  4567 89AB CDEF 1234 5678
```

Trust state in brackets:

- `[ultimate]` — your own key
- `[full]` — you have a fully-trusted certification path
- `[marginal]` — partial trust path
- `[unknown]` — no trust path (you should verify the fingerprint manually)
- `[never]` — explicit distrust

Exit codes:

- `0` — good signature with sufficient trust
- `1` — bad signature (cryptographic failure)
- `2` — error or untrusted but otherwise valid signature in some configurations

For machine consumption use the status FD (stable, documented in `doc/DETAILS`):

```bash
gpg --status-fd 1 --verify file.sig file 2>/dev/null
# [GNUPG:] NEWSIG
# [GNUPG:] SIG_ID xxxxxxx 2026-04-01 1712030132
# [GNUPG:] GOODSIG 1A2B3C4D5E6F7A8B Alice Smith <alice@example.com>
# [GNUPG:] VALIDSIG 0123...5678 2026-04-01 1712030132 0 4 0 22 10 00 0123...5678
# [GNUPG:] TRUST_ULTIMATE 0 pgp
```

Key statuses to grep for in scripts: `GOODSIG`, `VALIDSIG`, `TRUST_FULLY`, `TRUST_ULTIMATE`, `BADSIG`, `ERRSIG`, `EXPKEYSIG`, `EXPSIG`, `REVKEYSIG`.

## Signing Git Commits / Tags

Git natively supports OpenPGP commit/tag signatures. Configure once:

```bash
git config --global user.signingkey 0123456789ABCDEF0123456789ABCDEF12345678
git config --global commit.gpgsign true
git config --global tag.gpgSign true
git config --global gpg.program gpg
```

If you have multiple `gpg` binaries (Homebrew vs system), pin the path:

```bash
git config --global gpg.program /opt/homebrew/bin/gpg
```

Make a signed commit / tag:

```bash
git commit -S -m "feat: signed commit"             # -S forces signing if not default
git tag -s v1.0 -m "release v1.0"                  # signed annotated tag
```

Verify:

```bash
git log --show-signature
git verify-commit HEAD
git verify-tag v1.0
```

GitHub "Verified" badge: upload your **public** key to GitHub at *Settings → SSH and GPG keys → New GPG key*. The email on the GPG UID must match a verified email on the GitHub account.

Convenient: have ssh-agent emulation (see below) and use `gpg.format = ssh` if you'd rather sign commits with an SSH key — that's a Git-side feature, not GnuPG.

## gpg-agent

`gpg-agent` is the always-running daemon that holds unlocked secret keys, talks to pinentry, drives smartcard access, and (optionally) emulates `ssh-agent`. Configuration lives in `~/.gnupg/gpg-agent.conf`:

```ini
# How long an unlocked key stays cached after last use (seconds)
default-cache-ttl 600

# Hard maximum cache lifetime (seconds)
max-cache-ttl 7200

# SSH variants
default-cache-ttl-ssh 1800
max-cache-ttl-ssh 7200

# Which pinentry to use
pinentry-program /opt/homebrew/bin/pinentry-mac
# pinentry-program /usr/bin/pinentry-curses        # text terminal fallback
# pinentry-program /usr/bin/pinentry-gtk-2         # X11 GTK
# pinentry-program /usr/bin/pinentry-qt            # X11 Qt

# Allow batch use to bypass pinentry (DANGEROUS — only for sealed CI)
allow-loopback-pinentry

# Enable SSH agent emulation
enable-ssh-support

# Use the same passphrase cache for SSH and OpenPGP keys
# (rare; usually you want them separate)
# allow-emacs-pinentry

# Disable the scdaemon if you don't use smartcards
# disable-scdaemon
```

Lifecycle:

```bash
gpgconf --launch gpg-agent           # start if not running (idempotent)
gpgconf --kill gpg-agent             # stop
gpgconf --reload gpg-agent           # SIGHUP to re-read config
gpg-connect-agent reloadagent /bye   # alternative reload
gpg-connect-agent killagent /bye     # alternative kill
```

Send commands directly to the agent:

```bash
gpg-connect-agent
> getinfo version
D 2.4.5
OK
> keyinfo --list
S KEYINFO ABC...DEF D - - - P - - -
OK
> /bye
```

Force a passphrase clear:

```bash
echo RELOADAGENT | gpg-connect-agent
gpg-connect-agent reloadagent /bye
```

systemd user units (modern Linux):

```bash
systemctl --user status gpg-agent.socket
systemctl --user restart gpg-agent.service
systemctl --user enable --now gpg-agent.socket
```

macOS launchd: `gpgconf --launch gpg-agent` is enough; no plist required.

## gpg-agent SSH Integration

`gpg-agent` can replace `ssh-agent` and use your OpenPGP authentication subkey for SSH. Workflow:

1. **Add an authentication subkey** to your OpenPGP key:

```bash
gpg --edit-key alice@example.com
gpg> addkey
# > Please select what kind of key you want:
# >    (3) DSA (sign only)
# >    (4) RSA (sign only)
# >    (5) Elgamal (encrypt only)
# >    (6) RSA (encrypt only)
# >    (7) DSA (set your own capabilities)
# >    (8) RSA (set your own capabilities)
# >   (10) ECC (sign only)
# >   (11) ECC (set your own capabilities)
# >   (12) ECC (encrypt only)
# >   (14) Existing key from card
# > Your selection? 11
# > Possible actions for an EDDSA key: Sign Authenticate
# > Current allowed actions: Sign
# > Toggle: S          (turn off Sign)
# > Toggle: A          (turn on Authenticate)
# > Q                  (quit menu)
# > Your selection? S
# > Your selection? A
# > Your selection? Q
# > Please select which elliptic curve you want:
# >    (1) Curve 25519
# > Your selection? 1
# > Key is valid for? (0) 1y
gpg> save
```

2. **Enable ssh-support** in `~/.gnupg/gpg-agent.conf`:

```ini
enable-ssh-support
```

3. **Restart the agent**:

```bash
gpgconf --kill gpg-agent
gpgconf --launch gpg-agent
```

4. **Tell SSH to use the gpg-agent socket** (in `~/.bashrc` / `~/.zshrc`):

```bash
unset SSH_AGENT_PID
if [ "${gnupg_SSH_AUTH_SOCK_by:-0}" -ne $$ ]; then
    export SSH_AUTH_SOCK="$(gpgconf --list-dirs agent-ssh-socket)"
fi
export GPG_TTY="$(tty)"
gpg-connect-agent updatestartuptty /bye >/dev/null
```

5. **Authorize the key for SSH** by listing it in `sshcontrol` (older) or by exporting and using the key directly. Modern gpg-agent auto-presents the auth subkey via `ssh-add -L`.

```bash
ssh-add -L                            # list public SSH keys via gpg-agent
gpg --export-ssh-key alice@example.com > id_alice.pub  # OpenSSH-format public key
```

Add `id_alice.pub` content to remote `~/.ssh/authorized_keys`. SSH will now authenticate via gpg-agent, which will prompt for the OpenPGP passphrase (or hit the Yubikey) on first use.

The "one Yubikey for SSH + signing + encryption" workflow uses exactly this setup with the auth subkey pinned to the card slot.

## Yubikey / Smartcard Integration

The OpenPGP applet on Yubikey 5 (and similar OpenPGP cards) holds three slots: **Signature**, **Encryption**, **Authentication**. Once a private key is on the card, GnuPG keeps a **stub** in the keyring that says "use card X slot Y". The card never exports the private material.

Query card status:

```bash
gpg --card-status
# Reader ............: Yubico YubiKey OTP+FIDO+CCID
# Application ID ....: D2760001240103040006XXXXXXXX0000
# Application type ..: OpenPGP
# Version ..........: 3.4
# Manufacturer .....: Yubico
# Serial number ....: 12345678
# Name of cardholder: Alice Smith
# Language prefs ...: en
# Signature key ....: 0123 4567 89AB CDEF ...
#       created ....: 2026-04-01
# Encryption key....: AABB CCDD EEFF ...
#       created ....: 2026-04-01
# Authentication key: CCDD EEFF AABB ...
#       created ....: 2026-04-01
# General key info..: pub  ed25519/0x... 2026-04-01 ...
# PIN retry counter : 3 0 3
# Signature counter : 42
```

The interactive `--card-edit` shell:

```bash
gpg --card-edit
gpg/card> help
gpg/card> admin                       # enable admin commands
gpg/card> passwd                      # change PINs
# > 1 - change PIN
# > 2 - unblock PIN
# > 3 - change Admin PIN
# > 4 - set the Reset Code
gpg/card> name                        # set cardholder name
gpg/card> lang                        # set language
gpg/card> login                       # set login data (rarely used)
gpg/card> url                         # set fetch URL (for keyserver fallback)
gpg/card> key-attr                    # set key algorithms (defaults: rsa2048)
# Pick (2) ECC, then (1) Curve 25519 for sig, encr, auth slots
gpg/card> generate                    # generate keys ON CARD (private never leaves)
gpg/card> quit
```

Default PINs (Yubikey factory defaults — change immediately):

- User PIN: `123456`
- Admin PIN: `12345678`

Two ways to put keys on the card:

**(A) Generate on card** — strongest, but you cannot back up the private key:

```bash
gpg --card-edit
gpg/card> admin
gpg/card> key-attr           # set ECC + Curve 25519
gpg/card> generate
# > Make off-card backup of encryption key? (Y/n) Y
# (the encryption key has an off-card backup; sign and auth do not)
```

**(B) Move existing subkeys to card** — lets you keep an offline backup of the secret:

```bash
gpg --edit-key alice@example.com
gpg> key 1                   # select subkey 1 (encryption)
gpg> keytocard
# > Please select where to store the key:
# >    (2) Encryption key
# > Your selection? 2
gpg> key 1                   # deselect
gpg> key 2                   # select subkey 2 (signing)
gpg> keytocard
gpg> key 2
gpg> key 3                   # select subkey 3 (authentication)
gpg> keytocard
gpg> save
```

**WARNING**: `keytocard` is destructive on the host. The private subkey is replaced by a card stub in the keyring. Have a tested off-card backup before running it.

After moving, signing and decrypting will require the card to be inserted and the User PIN entered. PIN is cached per gpg-agent rules (`default-cache-ttl`).

Backup Yubikey: the canonical setup is two identical Yubikeys, one daily, one in a safe. Generate (or `keytocard`) the same subkeys onto each.

## Web Key Directory (WKD)

WKD is the modern, decentralized alternative to keyservers. You publish your public key at:

```text
https://example.com/.well-known/openpgpkey/hu/<HASHED-LOCAL-PART>
```

where `HASHED-LOCAL-PART` is the lowercased local part of your email, hashed with SHA-1, and Z-Base32 encoded.

Discover and fetch:

```bash
gpg --auto-key-locate wkd --locate-keys alice@example.com
gpg --locate-external-keys alice@example.com
```

Auto-resolve recipients during encryption:

```bash
gpg --auto-key-locate wkd,keyserver -ear alice@example.com file.txt
```

Configure in `gpg.conf`:

```ini
auto-key-locate wkd,keyserver
keyserver hkps://keys.openpgp.org
```

Publish your own WKD: a static HTTPS server at your domain. Tools:

```bash
gpg-wks-client --supported alice@example.com         # check if domain has WKD
gpg-wks-client --create alice@example.com            # generate a WKD blob for upload
```

For `example.com`, the directory layout is one of two variants:

```text
# Advanced (recommended; works for any subdomain)
https://openpgpkey.example.com/.well-known/openpgpkey/example.com/hu/<HASH>
https://openpgpkey.example.com/.well-known/openpgpkey/example.com/policy

# Direct
https://example.com/.well-known/openpgpkey/hu/<HASH>
https://example.com/.well-known/openpgpkey/policy
```

The `policy` file can be empty; it just signals that WKD is enabled.

To compute the hash for an arbitrary local part:

```bash
gpg-wks-client --print-wkd-hash alice
# kei1q4tipxxu1yj79k9kfukdhfy631xe
```

CORS / HTTPS notes: serve with proper TLS (Let's Encrypt is fine), no auth, no redirects.

## Keyservers

Historical context: in 2019, a flooding attack on the SKS keyserver network (used by `pgp.mit.edu`, `keys.gnupg.net`, etc.) made many user keys un-importable due to thousands of bogus signatures. Most SKS servers have shut down or stopped syncing.

Modern recommendation: **`keys.openpgp.org`**. It:

- Verifies email ownership before publishing UIDs (you receive a confirmation email).
- Strips third-party signatures during distribution (limits flooding).
- Offers `hkps://` (HTTPS) only.
- Does **not** federate with SKS.

Configure:

```ini
# In gpg.conf
keyserver hkps://keys.openpgp.org
```

Operations:

```bash
gpg --send-keys "$FPR"                           # publish (requires email confirmation first time)
gpg --recv-keys "$FPR"                           # fetch
gpg --search-keys alice@example.com              # search by UID
gpg --refresh-keys                               # refresh all keys in keyring
gpg --keyserver hkps://keys.openpgp.org --send-keys "$FPR"   # explicit keyserver
```

After `--send-keys`, check email for a verification link from `noreply@keys.openpgp.org`. Until you click it, the UID is not searchable by email.

Other surviving keyservers:

- `hkps://keyserver.ubuntu.com` — Ubuntu's, no email verification, hosts SKS-style data.
- `hkps://pgp.mit.edu` — historical, partially functional.

## DANE OPENPGPKEY

Publish OpenPGP keys via DNS using `OPENPGPKEY` records under `_openpgpkey.example.com.`. DNSSEC-protected; rarely used in practice but elegant.

Discover:

```bash
gpg --auto-key-locate dane --locate-keys alice@example.com
```

Generate the DNS record from a key:

```bash
gpg --export-options export-dane --export alice@example.com
# Outputs an RFC 7929 OPENPGPKEY record fragment
```

Operationally heavyweight: requires DNSSEC, large TXT-like RRs, and DNS hosting that supports the type. Most teams use WKD instead.

## Configuration File

`~/.gnupg/gpg.conf` — the canonical user-config. A sensible modern set:

```ini
# Display
keyid-format long
with-fingerprint
with-keygrip
no-emit-version
no-comments
list-options show-uid-validity
verify-options show-uid-validity

# Algorithms
personal-cipher-preferences AES256 AES192 AES
personal-digest-preferences SHA512 SHA384 SHA256
personal-compress-preferences ZLIB BZIP2 ZIP Uncompressed
default-preference-list SHA512 SHA384 SHA256 AES256 AES192 AES ZLIB BZIP2 ZIP Uncompressed
cert-digest-algo SHA512
s2k-cipher-algo AES256
s2k-digest-algo SHA512
s2k-mode 3
s2k-count 65011712

# Behavior
default-recipient-self
encrypt-to alice@example.com         # always cc: yourself
auto-key-locate wkd,keyserver
keyserver hkps://keys.openpgp.org
no-symkey-cache                       # don't cache passphrases for symmetric ops
require-cross-certification           # subkeys must back-sign primary
trust-model pgp
charset utf-8
# throw-keyids                         # uncomment to anonymize recipients
```

`~/.gnupg/dirmngr.conf` — the daemon that talks to keyservers and HTTPS:

```ini
keyserver hkps://keys.openpgp.org
hkp-cacert /etc/ssl/certs/ca-certificates.crt    # rare; only if hkps cert isn't in default truststore
```

Restart after edits:

```bash
gpgconf --reload
gpgconf --kill all && gpgconf --launch gpg-agent
```

## gpgconf

`gpgconf` is the canonical management tool — list components, options, defaults; mass-restart daemons.

```bash
# List GnuPG components
gpgconf --list-components
# gpg:OpenPGP:/usr/local/bin/gpg
# gpgsm:S/MIME:/usr/local/bin/gpgsm
# gpg-agent:Private Keys:/usr/local/bin/gpg-agent
# scdaemon:Smartcards:/usr/local/libexec/scdaemon
# dirmngr:Network:/usr/local/bin/dirmngr
# pinentry:PIN and Passphrase Entry:/opt/homebrew/bin/pinentry-mac

# All options for a component
gpgconf --list-options gpg
gpgconf --list-options gpg-agent
gpgconf --list-options dirmngr

# Show default + current value of one option
gpgconf --list-options gpg | grep -E '^default-key:|^keyserver:'

# Change options programmatically (each line is name:flags:value)
echo 'keyserver:0:"hkps://keys.openpgp.org' | gpgconf --change-options gpg

# Show directory paths
gpgconf --list-dirs
# sysconfdir:/etc/gnupg
# bindir:/usr/local/bin
# libexecdir:/usr/local/libexec
# datadir:/usr/local/share/gnupg
# homedir:/Users/alice/.gnupg
# socketdir:/Users/alice/.gnupg
# agent-socket:/Users/alice/.gnupg/S.gpg-agent
# agent-ssh-socket:/Users/alice/.gnupg/S.gpg-agent.ssh

# Daemon control
gpgconf --kill all                   # stop every GnuPG daemon
gpgconf --kill gpg-agent             # stop just the agent
gpgconf --launch gpg-agent           # start (idempotent)
gpgconf --reload gpg-agent           # SIGHUP, re-read config
gpgconf --reload                     # reload everything
```

## GnuPG 2.2 LTS vs 2.4+

| Feature                              | 2.2 LTS       | 2.4+                |
|--------------------------------------|---------------|---------------------|
| Default symmetric KDF                | iterated/salted (S2K mode 3) | Argon2 (S2K mode 4) |
| AEAD (modern auth-encrypt) defaults  | OCB available, opt-in        | OCB default for new keys |
| Curve 25519 ergonomics               | Works, more typing in `--full-generate-key` | First-class via `--quick-generate-key` |
| `--export-options export-clean`      | Works                         | Works               |
| RFC 9580 v6 keys                     | No                            | Partial / experimental |
| Argon2 packet support on decryption  | No                            | Yes                 |
| Yubikey OpenPGP 3.4+ ed25519         | Yes                           | Yes                 |

For maximum interop with old recipients (still on GnuPG 1.4 or 2.0), pass `--rfc4880`:

```bash
gpg --rfc4880 -e -r legacy@example.com file.txt
# Suppresses post-RFC4880 features (e.g., AEAD wrapping) that older clients can't parse.
```

For GnuPG 2.6/RFC 9580 mode (when available):

```bash
gpg --rfc9580 ...
```

## Common Errors and Fixes

Exact text of the most-seen errors and their fixes:

**`gpg: decryption failed: No secret key`**
The secret key is not in your keyring. Verify with `gpg --list-secret-keys`. Import with `gpg --import alice-secret-sub.asc`. If it should be on a Yubikey, `gpg --card-status` to confirm the card is detected and the stub is in your keyring.

**`gpg: signing failed: No agent running`**
Start the agent: `gpgconf --launch gpg-agent`. On systemd: `systemctl --user restart gpg-agent.socket`. If repeated, fix the `pinentry-program` path in `~/.gnupg/gpg-agent.conf`.

**`gpg: Sorry, no terminal at all requested - can't get input`**
gpg needs a TTY for pinentry-curses. Add to `~/.bashrc` / `~/.zshrc`:

```bash
export GPG_TTY=$(tty)
```

Reload your shell or `source ~/.bashrc`. For SSH sessions, add `SendEnv GPG_TTY` on the client and `AcceptEnv GPG_TTY` on the server.

**`gpg: invalid option "--list-keys"`** (but you typed it correctly)
Wrong gpg binary on PATH — likely GnuPG 1.4 from a stale install. Check:

```bash
which gpg
gpg --version
ls -la $(which gpg)
```

Fix: install GnuPG 2.x and reorder PATH so the modern binary is first.

**`gpg: WARNING: This key is not certified with a trusted signature!`**
You imported the key but haven't established trust. Either certify it (`gpg --edit-key bob@example.com`, then `sign`, `save`) or set ownertrust (`trust`, then choose level). For one-off non-security uses: `--trust-model always`.

**`gpg: WARNING: encrypting to alice@example.com that is not the registered uid of the key`**
The recipient string doesn't match a UID exactly. Use the fingerprint instead:

```bash
gpg --encrypt -r 0123456789ABCDEF0123456789ABCDEF12345678 file
```

Or fix the UID with `--edit-key + adduid`.

**`gpg: error retrieving 'alice@example.com' via WKD: General error`**
DNS or HTTPS unreachable. Test:

```bash
curl -v https://openpgpkey.example.com/.well-known/openpgpkey/example.com/hu/$(gpg-wks-client --print-wkd-hash alice)
```

Falls back to keyserver if `auto-key-locate wkd,keyserver` is set. Force keyserver:

```bash
gpg --auto-key-locate keyserver --locate-keys alice@example.com
```

**`gpg: skipping pubring.gpg, already migrated`**
Benign. GnuPG 2.1 migrated from `pubring.gpg` to `pubring.kbx` (the keybox format). The old file is preserved but no longer used. Delete it after confirming the new keybox has all keys.

**`gpg: There is no assurance this key belongs to the named user`**
Same family as the trust warning above. Either certify the key after verifying the fingerprint, or use `--trust-model always` for non-security verification (e.g., signed downloads where you trust the channel).

**`gpg: WARNING: cipher algorithm CAST5 not found in recipient preferences`**
The recipient's key advertises an outdated set of cipher prefs. The encryption still works using the lowest common denominator. Modern fix: ask the recipient to update their key to ECC + modern preferences. Workaround: `--cipher-algo AES256` to force.

**`gpg: encrypted with NNNN-bit RSA key, ID FPR, created Y`** followed by successful decryption
Not an error. Status info confirming which subkey decrypted the message.

**`gpg: WARNING: signature digest conflict in message`**
Mixed digest algorithms in a multi-signature object (rare). Usually still verifies. Inspect with `gpg --list-packets`.

**`gpg: keyserver receive failed: No name`**
DNS for the keyserver hostname failed. Test with `dig keys.openpgp.org`. Common cause: corporate DNS that blocks unknown TLDs. Switch to a different keyserver or fix DNS.

**`gpg: keyserver receive failed: Server indicated a failure`**
Keyserver responded with a 4xx/5xx. Try again later, or switch to another keyserver. `keys.openpgp.org` returns 404 for unverified UIDs — fingerprint lookup still works.

**`gpg: WARNING: standard input reopened`**
You piped data into a command that also wants a TTY for pinentry. Fix with `--pinentry-mode loopback --passphrase-file <file>` for batch use.

**`gpg: failed to start agent '/usr/local/bin/gpg-agent': No such file or directory`**
Path mismatch between `gpg.conf` `agent-program` and the installed binary. Remove the explicit `agent-program` line (let gpg find it) or fix the path.

**`gpg: signal Interrupt caught ... exiting`**
Ctrl-C during pinentry. Re-run.

**`gpg: problem with the agent: No pinentry`**
The pinentry binary listed in `gpg-agent.conf` doesn't exist. Set:

```ini
pinentry-program /opt/homebrew/bin/pinentry-mac
```

or use `pinentry-curses` for terminal-only systems. Restart the agent.

## Common Gotchas

For each: **bad** behavior, then the **fix**.

**bad**: committing `--export-secret-keys` output to a repo (or shipping it to a host that doesn't need the primary).
**fix**: use `--export-secret-subkeys` for daily-driver hosts. The primary stays offline:

```bash
gpg --armor --export-secret-subkeys alice@example.com > secret-sub.asc
```

**bad**: pinentry never appears in non-interactive shells (cron, CI, SSH without TTY).
**fix**: `export GPG_TTY=$(tty)` in shell rc, and pass `--pinentry-mode loopback --passphrase-file ...` in scripts.

**bad**: generating fresh keys as RSA-2048 in 2026.
**fix**: ed25519 (signing/auth/cert) + cv25519 (encryption). Faster, smaller, no parameter confusion:

```bash
gpg --quick-generate-key "Name <email>" ed25519 cert 1y
```

**bad**: keyserver of choice is offline; `--receive-keys` fails.
**fix**: explicit `--keyserver hkps://keys.openpgp.org`, or set `keyserver` and `auto-key-locate wkd,keyserver` in `gpg.conf` so failures fall through to WKD.

**bad**: pinentry doesn't pop up; gpg appears to hang.
**fix**: check `pinentry-program` in `~/.gnupg/gpg-agent.conf`. On macOS install `brew install pinentry-mac`. Restart the agent: `gpgconf --kill gpg-agent; gpgconf --launch gpg-agent`. On Linux X11: `pinentry-gtk-2` or `pinentry-qt`. Headless: `pinentry-curses`.

**bad**: SSH via gpg-agent doesn't see the key after you enabled `enable-ssh-support`.
**fix**: restart the shell, ensure `SSH_AUTH_SOCK="$(gpgconf --list-dirs agent-ssh-socket)"` is exported, run `gpg-connect-agent updatestartuptty /bye`. Confirm with `ssh-add -L`.

**bad**: `gpg --decrypt` blocks waiting for passphrase in a script.
**fix**: full batch incantation:

```bash
gpg --batch --pinentry-mode loopback \
    --passphrase-file /path/to/passfile \
    --decrypt file.gpg
```

Or, better, decrypt at a time when the agent already has the passphrase cached.

**bad**: Yubikey unplugged but the keyring still shows the key; signing fails with cryptic errors.
**fix**: insert the card and run `gpg --card-status`. If you've moved keys between cards, regenerate stubs:

```bash
gpg --card-status        # binds stubs to the inserted card
```

If the stub is permanently stale: `gpg --delete-secret-keys FPR` and re-import.

**bad**: keys you signed don't appear as trusted; signatures still show `[unknown]`.
**fix**: ownertrust is separate from certification. After signing a key, set ownertrust if you also want to trust them as a CA-of-sorts: `gpg --edit-key + trust + 4` (full) or `5` (ultimate, your own keys only).

**bad**: forgetting to generate a revocation certificate before going on vacation, then losing the laptop.
**fix**: at key creation, immediately:

```bash
gpg --output revoke-alice.asc --gen-revoke alice@example.com
```

Print, store in a safe. Also note `~/.gnupg/openpgp-revocs.d/<FPR>.rev` exists by default in GnuPG 2.1+.

**bad**: encrypting to a name substring like `-r Alice` and matching the wrong key.
**fix**: always use the fingerprint:

```bash
gpg -ear 0123456789ABCDEF0123456789ABCDEF12345678 file.txt
```

**bad**: passphrase change via `passwd` in `--edit-key` only changes one key (e.g., the primary), leaving subkeys with the old passphrase.
**fix**: GnuPG 2.x stores each key by keygrip and asks per-key. Use `passwd` repeatedly, or import after re-export to normalize.

**bad**: relying on short keyids (8 hex chars) for recipient selection.
**fix**: long keyids are also collidable in theory; the only safe identifier is the full fingerprint.

**bad**: assuming `keys.openpgp.org` will distribute your third-party signatures.
**fix**: it strips them. For full WoT signatures, distribute keys directly to the owner or use a SKS-clone server (rare).

**bad**: mismatched gpg vs gpg-agent binaries (system gpg + Homebrew agent or vice versa).
**fix**: verify with `which gpg` and `which gpg-agent`. Pin paths in PATH order. `gpgconf --list-components` shows what gpg thinks is installed.

## Migration to Modern Crypto

The "RSA-2048 → ed25519" walkthrough for an existing user.

1. **Generate a new ed25519 key** with the same UID:

```bash
gpg --quick-generate-key "Alice Smith <alice@example.com>" ed25519 cert 1y
NEW_FPR=$(gpg --list-keys --with-colons alice@example.com \
  | awk -F: '/^fpr:/ {print $10}' | tail -1)
gpg --quick-add-key "$NEW_FPR" cv25519 encr 1y
gpg --quick-add-key "$NEW_FPR" ed25519 sign 1y
gpg --quick-add-key "$NEW_FPR" ed25519 auth 1y
```

2. **Sign a transition statement** with both keys. Plain text, signed by both old and new:

```bash
cat > transition.txt <<EOF
I, Alice Smith, am transitioning my OpenPGP identity from
  OLD: 1234 5678 90AB CDEF 1234 5678 90AB CDEF 12345678 (RSA-2048)
to
  NEW: 0123 4567 89AB CDEF 0123 4567 89AB CDEF 12345678 (ed25519)

The new key supersedes the old. Please re-certify and re-encrypt to the new key.
Effective: 2026-04-25
EOF

gpg -u OLD_FPR --clearsign transition.txt
gpg -u NEW_FPR --clearsign transition.txt.asc -o transition.txt.asc.asc
mv transition.txt.asc.asc transition.txt.both.asc
```

3. **Cross-sign**: sign the new key with the old, and the old with the new:

```bash
gpg --default-key OLD_FPR --sign-key NEW_FPR
gpg --default-key NEW_FPR --sign-key OLD_FPR
```

4. **Dual-publish**: keep both keys live for ~6 months. Encrypt to both, sign with both for important documents.

5. **Revoke the old key** at end of overlap:

```bash
gpg --output revoke-old.asc --gen-revoke OLD_FPR    # if not already generated
gpg --import revoke-old.asc
gpg --keyserver hkps://keys.openpgp.org --send-keys OLD_FPR
```

6. **Republish the new key** to keyservers and WKD:

```bash
gpg --keyserver hkps://keys.openpgp.org --send-keys NEW_FPR
# Confirm via the email link from keys.openpgp.org
# Update WKD blob if you publish your own
```

## Modern Alternatives

OpenPGP is heavy. For new use cases, ask whether you actually need it.

- **age** (`age-encryption.org`) — file encryption with X25519 / scrypt. Tiny binary, no UID concept, no keyservers, deterministic key files. Use for: backups, file sharing, sealed envelopes between machines. Not a signing tool.
- **minisign** — Ed25519 signatures only. Two files: `key.pub` and `key.sec`. No keyservers. Use for: code signing, release signatures, automation. Compatible with `signify` (OpenBSD).
- **sops** + age/PGP/KMS — git-friendly encrypted secrets, encrypts only values not keys, structure-aware (YAML/JSON/INI). Use for: Kubernetes secrets, IaC config.
- **sequoia-pgp** (`sequoia-pgp.org`) — Rust implementation of OpenPGP. Wire-compatible with GnuPG; better library API; CLI named `sq`. Drop-in alternative to GnuPG for many tasks.
- **gpgme** / **rnp** — alternative OpenPGP libraries; rnp is BSD-licensed (used by Thunderbird).
- **signify** / **OpenBSD signify** — minimalist Ed25519 signatures.
- **HashiCorp Vault transit engine** — server-side encrypt/decrypt with rotated keys, audited. Use for: per-application encryption with a central audit log.

Rule of thumb: GnuPG is correct when you need OpenPGP-protocol interop (Debian package signing, git commit verified-via-OpenPGP, Yubikey-CCID, third-party WoT). For everything else, simpler tools are usually correct.

## Idioms

The patterns to internalize.

**Sign a release tarball**:

```bash
sha256sum mytool-1.0.tar.gz > mytool-1.0.tar.gz.sha256
gpg --armor --detach-sign mytool-1.0.tar.gz
gpg --armor --clearsign mytool-1.0.tar.gz.sha256
# Publish: tarball + .asc + .sha256.asc on the project release page
```

**Verify a release**:

```bash
gpg --recv-keys 0123...                              # fetch the project key
gpg --verify mytool-1.0.tar.gz.sha256.asc           # verify the checksum file
sha256sum -c mytool-1.0.tar.gz.sha256                # verify the tarball
gpg --verify mytool-1.0.tar.gz.asc mytool-1.0.tar.gz # belt and suspenders
```

**Encrypted backup**:

```bash
tar czf - srcdir/ | gpg -e -r alice@example.com -o backup-$(date +%F).tar.gz.gpg
# To restore:
gpg -d backup-2026-04-25.tar.gz.gpg | tar xzf -
```

**Send an encrypted file via email**:

```bash
gpg -ear bob@example.com message.txt
# Send message.txt.asc as attachment or paste inline
```

**Sign a git commit / verify**:

```bash
git commit -S -m "..."           # sign
git log --show-signature
git verify-commit HEAD
```

**Offline primary + Yubikey + WKD**: the canonical advanced setup.

1. Generate primary on offline laptop. Backup secret-FULL + revocation cert to two USB sticks + paper QR.
2. Generate three subkeys.
3. `keytocard` each subkey to Yubikey #1.
4. Repeat for Yubikey #2 (backup card).
5. Export `--export-secret-subkeys` and `--export` to public key.
6. Import public key on daily-driver machines.
7. Insert Yubikey #1 daily; #2 in safe.
8. Publish public key to `keys.openpgp.org` and to your domain's WKD.
9. Yearly: boot offline laptop, extend subkey expiry, re-export, re-publish.

**TOFU for casual peers**: when you can't meet in person but want some assurance:

```ini
# In gpg.conf
trust-model tofu+pgp
```

GnuPG records first-seen fingerprints per email and warns on changes. Not as strong as WoT but better than `--trust-model always`.

**Rotation calendar**: put a yearly reminder. The primary stays put; subkeys roll. The public key stays the same identity; the world re-fetches and notices the new subkeys via `--refresh-keys`.

## Tips

One-liners worth memorizing.

```bash
# List keys with full info
gpg --list-keys --keyid-format long --with-fingerprint

# Encrypt for one recipient, ASCII-armored
gpg -ear bob@example.com file.txt

# Decrypt to stdout
gpg -d file.gpg

# Verify a detached signature
gpg --verify file.sig file

# Generate a revocation certificate now
gpg --output revoke.asc --gen-revoke alice@example.com

# Show everything inside a packet
gpg --list-packets file.gpg

# Inspect a key file before import
gpg --show-keys keyfile.asc

# Auto-locate a recipient via WKD/keyserver
gpg --auto-key-locate wkd,keyserver --locate-keys alice@example.com

# Refresh all keys (catch revocations)
gpg --refresh-keys

# Change key expiry
gpg --edit-key alice@example.com
gpg> expire

# Send to keyserver
gpg --keyserver hkps://keys.openpgp.org --send-keys "$FPR"

# Receive from keyserver
gpg --keyserver hkps://keys.openpgp.org --recv-keys "$FPR"

# Export public key (armored)
gpg --armor --export alice@example.com > alice-pub.asc

# Export secret subkeys (armored, no primary)
gpg --armor --export-secret-subkeys alice@example.com > alice-secret-sub.asc

# Card status (Yubikey)
gpg --card-status

# Restart the agent
gpgconf --kill gpg-agent && gpgconf --launch gpg-agent

# Force a passphrase re-prompt
echo RELOADAGENT | gpg-connect-agent

# Set GPG_TTY for pinentry-curses
export GPG_TTY=$(tty)

# Use gpg-agent for SSH
export SSH_AUTH_SOCK="$(gpgconf --list-dirs agent-ssh-socket)"

# Strip third-party signatures during export
gpg --export-options export-minimal --export FPR > clean.gpg

# Quick ed25519 + subkeys
gpg --quick-generate-key "Name <email>" ed25519 cert 1y
gpg --quick-add-key FPR cv25519 encr 1y
gpg --quick-add-key FPR ed25519 sign 1y
gpg --quick-add-key FPR ed25519 auth 1y

# Sign a git tag
git tag -s v1.0 -m "release v1.0"

# Verify a git tag
git verify-tag v1.0

# Argon2 symmetric (modern KDF)
gpg --symmetric --s2k-mode 4 --cipher-algo AES256 file.txt

# Batch decrypt with passphrase file
gpg --batch --pinentry-mode loopback --passphrase-file pf -d file.gpg

# Show config
gpgconf --list-options gpg
```

## See Also

- openssl, ssh, vault, sops, polyglot

## References

- `man gpg`
- `man gpg-agent`
- `man gpgconf`
- `man dirmngr`
- [GnuPG Manual](https://www.gnupg.org/documentation/manuals/gnupg/)
- [GnuPG FAQ](https://www.gnupg.org/faq/gnupg-faq.html)
- [GnuPG mini-HOWTO](https://www.gnupg.org/howtos/en/mini-HOWTO.html)
- [Riseup OpenPGP Best Practices](https://riseup.net/en/security/message-security/openpgp/best-practices)
- [Debian Wiki — Subkeys](https://wiki.debian.org/Subkeys)
- [Arch Wiki — GnuPG](https://wiki.archlinux.org/title/GnuPG)
- [Yubico — Use OpenPGP with Yubikey](https://developers.yubico.com/PGP/)
- [keys.openpgp.org — about](https://keys.openpgp.org/about)
- [Web Key Directory draft](https://datatracker.ietf.org/doc/draft-koch-openpgp-webkey-service/)
- [RFC 4880 — OpenPGP Message Format](https://www.rfc-editor.org/rfc/rfc4880)
- [RFC 9580 — OpenPGP (crypto refresh)](https://www.rfc-editor.org/rfc/rfc9580)
- [RFC 7929 — DNS-Based Authentication of OpenPGP](https://www.rfc-editor.org/rfc/rfc7929)
- "OpenPGP for Application Developers" — https://openpgp.dev/book/
- [Sequoia-PGP](https://sequoia-pgp.org/)
- [age — A Simple, Modern File Encryption Tool](https://age-encryption.org/)
