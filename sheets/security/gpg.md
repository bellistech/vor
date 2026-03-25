# GPG (GNU Privacy Guard)

Encrypt, decrypt, sign, and verify files and messages using public-key cryptography.

## Key Management

### Generate a New Key Pair

```bash
# Interactive (recommended for first key)
gpg --full-generate-key

# Quick generation with defaults (RSA 3072, no expiry prompt)
gpg --quick-generate-key "Alice Smith <alice@acme.com>" rsa4096 default 2y
```

### List Keys

```bash
gpg --list-keys                          # public keys
gpg --list-keys --keyid-format long      # show full key IDs
gpg --list-secret-keys                   # private keys
gpg --fingerprint alice@acme.com         # show fingerprint
```

### Delete Keys

```bash
gpg --delete-key alice@acme.com          # public key
gpg --delete-secret-key alice@acme.com   # private key first, then public
gpg --delete-secret-and-public-key alice@acme.com
```

## Import and Export

### Export Keys

```bash
# Public key (for sharing)
gpg --armor --export alice@acme.com > alice-pub.asc

# Private key (for backup -- guard this)
gpg --armor --export-secret-keys alice@acme.com > alice-priv.asc
```

### Import Keys

```bash
gpg --import alice-pub.asc
gpg --import alice-priv.asc
```

## Encryption and Decryption

### Encrypt for a Recipient

```bash
# Encrypt to a specific person (they decrypt with their private key)
gpg --armor --encrypt --recipient bob@acme.com secret.txt
# produces secret.txt.asc

# Encrypt to multiple recipients
gpg --encrypt --recipient bob@acme.com --recipient carol@acme.com secret.txt

# Encrypt with a passphrase (symmetric, no keys needed)
gpg --symmetric --cipher-algo AES256 secret.txt
```

### Decrypt

```bash
gpg --decrypt secret.txt.gpg > secret.txt
gpg --decrypt secret.txt.asc             # armor format
gpg -d secret.txt.gpg                    # short form, prints to stdout
```

## Signing and Verification

### Sign a File

```bash
# Detached signature (separate .sig file)
gpg --detach-sign --armor report.pdf     # produces report.pdf.asc

# Clearsign (inline signature, readable text)
gpg --clearsign message.txt              # produces message.txt.asc

# Embedded signature (binary, signature + data combined)
gpg --sign document.pdf                  # produces document.pdf.gpg
```

### Verify a Signature

```bash
# Detached signature
gpg --verify report.pdf.asc report.pdf

# Clearsigned or embedded
gpg --verify message.txt.asc
```

### Sign and Encrypt

```bash
gpg --sign --encrypt --recipient bob@acme.com secret.txt
```

## Keyservers

### Search and Fetch Keys

```bash
gpg --keyserver hkps://keys.openpgp.org --search-keys bob@acme.com
gpg --keyserver hkps://keys.openpgp.org --recv-keys 0xABCDEF1234567890
```

### Publish Your Key

```bash
gpg --keyserver hkps://keys.openpgp.org --send-keys 0xABCDEF1234567890
```

### Refresh Keys (Check for Revocations)

```bash
gpg --keyserver hkps://keys.openpgp.org --refresh-keys
```

## Trust Model

### Set Trust Level

```bash
gpg --edit-key bob@acme.com
# At the gpg> prompt:
#   trust    -> set trust level (1-5)
#   sign     -> sign their key (certify it)
#   save     -> save and exit
```

### Sign (Certify) a Key

```bash
# After verifying fingerprint in person
gpg --sign-key bob@acme.com

# Local signature only (not exportable)
gpg --lsign-key bob@acme.com
```

## Key Editing

### Add a Subkey or UID

```bash
gpg --edit-key alice@acme.com
# gpg> addkey    -> add a new subkey
# gpg> adduid    -> add another email/name
# gpg> expire    -> change expiration
# gpg> passwd    -> change passphrase
# gpg> save
```

### Revoke a Key

```bash
# Generate revocation certificate (do this when you create the key)
gpg --gen-revoke alice@acme.com > revoke.asc

# Apply it when needed
gpg --import revoke.asc
gpg --keyserver hkps://keys.openpgp.org --send-keys <KEY_ID>
```

## Batch and Scripting

```bash
# Encrypt without interactive prompts
gpg --batch --yes --trust-model always \
  --recipient bob@acme.com --encrypt secret.txt

# Decrypt with passphrase from file (symmetric)
gpg --batch --passphrase-file /path/to/passfile --decrypt secret.txt.gpg

# Verify exit code in scripts
gpg --verify report.pdf.asc report.pdf && echo "GOOD" || echo "BAD"
```

## Tips

- Always generate a revocation certificate immediately after key creation and store it offline
- Use `--armor` (`-a`) to produce ASCII output suitable for email; omit it for smaller binary files
- Key IDs can collide; always verify the full fingerprint, especially from keyservers
- `keys.openpgp.org` requires email verification before publishing; `keyserver.ubuntu.com` does not
- GPG agent caches passphrases; use `gpgconf --kill gpg-agent` to clear the cache
- For git commit signing, add `signingkey` to `~/.gitconfig` and set `commit.gpgsign = true`
- Subkeys are preferred for daily use -- keep the master key offline on a USB drive
- `--trust-model always` skips trust checks, useful in CI but dangerous for real verification
