# SOPS — Secrets OPerationS

Mozilla/getsops SOPS: leaf-value encryption for YAML/JSON/TOML/INI/dotenv/binary with multi-recipient envelopes (age, PGP, AWS KMS, Azure KV, GCP KMS, HC Vault). Git-friendly diffs, structure-preserving, MAC-protected.

## Setup

Install on macOS via Homebrew, the canonical path. Homebrew tracks `getsops/sops` (the active fork) since the Mozilla repository was archived in 2023.

```bash
brew install sops
sops --version
```

Install on Debian/Ubuntu via apt or the release `.deb`. The distro packages tend to lag — prefer the GitHub release for current features.

```bash
sudo apt-get install sops

SOPS_VERSION=3.9.4
curl -L "https://github.com/getsops/sops/releases/download/v${SOPS_VERSION}/sops_${SOPS_VERSION}_amd64.deb" -o /tmp/sops.deb
sudo dpkg -i /tmp/sops.deb
```

Install on Fedora/RHEL/Rocky via dnf or the release `.rpm`.

```bash
SOPS_VERSION=3.9.4
sudo rpm -i "https://github.com/getsops/sops/releases/download/v${SOPS_VERSION}/sops-${SOPS_VERSION}-1.x86_64.rpm"
```

Install on Alpine (musl static binary).

```bash
SOPS_VERSION=3.9.4
curl -L "https://github.com/getsops/sops/releases/download/v${SOPS_VERSION}/sops-v${SOPS_VERSION}.linux.amd64" -o /usr/local/bin/sops
chmod +x /usr/local/bin/sops
```

Install via Go (builds head-of-tree from source; requires Go 1.21+).

```bash
go install github.com/getsops/sops/v3/cmd/sops@latest
which sops
```

Install on Arch (community).

```bash
sudo pacman -S sops
```

Install on NixOS or via Nix.

```bash
nix-env -iA nixpkgs.sops
nix run nixpkgs#sops -- --version
```

The repository move: `mozilla/sops` was archived in late 2023; the active fork lives at `github.com/getsops/sops`. The binary, CLI flags, and file format are identical between the two — `mozilla/sops` 3.7.3 and `getsops/sops` 3.9.x can read each other's encrypted files. Migration is a no-op beyond updating the install source.

Version compatibility within the 3.x line.

```bash
sops --version
```

3.0+ introduced the recipient-set design used today. 3.5+ added age recipient support (the recommended path forward). 3.7+ added key_groups / Shamir threshold groups in the CLI. 3.8+ added hc_vault_transit_uri in .sops.yaml schema. 3.9+ introduced improved YAML 1.2 handling and JSON pointer extract. Files encrypted with 3.5+ remain readable by 3.9+; downgrade-only paths may lose features (e.g., reopening a hc_vault-only file in 3.6 will fail).

Verify by running `sops --help`. The first time you run it on a new system, set `EDITOR` so the in-place editing flow works.

```bash
export EDITOR=vim
sops --help
```

Confirm dependencies are present.

```bash
which gpg
gpg --version
which age
age --version
which aws
aws --version
```

## Why SOPS

SOPS encrypts only the values inside structured files, leaving the keys, structure, and comments visible in plaintext. A YAML file with a hundred config knobs and three secrets becomes a YAML file with a hundred config knobs and three encrypted strings — diffable, mergeable, comprehensible.

Native parsers for the common configuration formats.

```bash
sops -e config.yaml > config.enc.yaml
sops -e config.json > config.enc.json
sops -e config.toml > config.enc.toml
sops -e .env > .env.enc
sops -e --input-type binary keystore.jks > keystore.jks.enc
```

Encrypted-fields versus full-file. Full-file encryption (gpg, age) makes the entire file opaque to git: every change rewrites every byte, every diff is meaningless. SOPS encrypts only the leaf values, so edits to one secret produce a one-line diff:

```bash
sops -e secrets.yaml > secrets.enc.yaml
git add secrets.enc.yaml
git commit -m 'initial secrets'

sops -i secrets.enc.yaml
git diff secrets.enc.yaml
```

Multi-recipient envelopes. The dataKey (a 256-bit AES key SOPS generates per file) is wrapped under each recipient's public key. Any single recipient can decrypt; revocation is a `sops updatekeys` away.

```bash
sops -e \
  --age age1u9q6r5d8j3p2k4lvwetszqyt7gpz0fvy6lnnxh3v9k02j5fa6vys9zrf6h \
  --pgp 85D77543B3D624B63CEA9E6DBC17301B491B3F21 \
  --kms arn:aws:kms:us-east-1:111111111111:key/abc-123 \
  config.yaml > config.enc.yaml
```

Git-friendly diffs. Because keys and structure stay plaintext, code review can confirm that a PR added an `aws_secret` field without revealing the value. Reviewers see the field name and an `ENC[...]` payload, can ask "Why is this added? What rotation policy?" — without the value being on screen.

Leaf-value-only encryption preserves structure for tools that read the file. A Helm chart's `values.yaml` keeps its tree shape; a Terraform `terraform.tfvars` keeps its variable layout; a Kubernetes Secret manifest keeps the API kind, version, and metadata visible to `kubectl explain` even while encrypted at rest.

Format coverage at a glance.

```bash
sops -e file.yaml
sops -e file.json
sops -e file.toml
sops -e --input-type ini file.ini
sops -e --input-type dotenv file.env
sops -e --input-type binary file.bin
```

## Architecture

The recipient model is the heart of SOPS. Every encrypted file carries a metadata block (in YAML, the `sops:` key; in JSON, the `sops` field at the root; in binary, a base64 envelope) that lists every recipient and the dataKey wrapped to each.

```bash
sops -d --extract '["sops"]' config.enc.yaml
grep -A 50 '^sops:' config.enc.yaml
```

The dataKey envelope. SOPS generates a fresh 256-bit AES-GCM key per file at encrypt time. That key is wrapped under each recipient's public key (or KMS-encrypted blob) and stored in the metadata. Every leaf value is encrypted with the same dataKey, so adding a recipient never re-encrypts the data — only the wrapped dataKey list changes.

Conceptual envelope (simplified) inside an encrypted YAML file shows an `age` block listing each recipient and a wrapped block, a `pgp` block with fingerprints and PGP-armored wrapped keys, a `kms` block with ARNs and base64 ciphertexts, plus `lastmodified`, `mac`, and `version` fields.

```bash
sops -d --extract '["sops"]["age"]' config.enc.yaml
sops -d --extract '["sops"]["pgp"]' config.enc.yaml
sops -d --extract '["sops"]["kms"]' config.enc.yaml
```

MAC over plaintext for tampering detection. SOPS computes a MAC (HMAC-SHA-512) over all the plaintext leaf values concatenated in canonical order, then encrypts that MAC with the dataKey. On decrypt, SOPS recomputes the MAC from the decrypted plaintext and compares — any tampering with ciphertexts (or with the metadata) trips a "MAC mismatch" error.

```bash
sops -d config.enc.yaml > /dev/null && echo OK

sed -i 's/data:abc/data:abd/' config.enc.yaml
sops -d config.enc.yaml
```

Controlling which keys get encrypted. Four mechanisms select which leaves go through encryption; the rest stay plaintext.

```bash
sops --encrypted-regex '^(password|secret|key|token)' -e config.yaml > out.yaml
sops --unencrypted-regex '^(public|hostname)' -e config.yaml > out.yaml
sops --encrypted-suffix '_enc' -e config.yaml > out.yaml
sops --unencrypted-suffix '_unenc' -e config.yaml > out.yaml
```

Pick exactly one of these per file; mixing leads to surprising behaviour. The `.sops.yaml` configuration file (next sections) lets you set these per-path.

The dataKey lifecycle.

```bash
sops -e config.yaml > config.enc.yaml
sops --add-age age1NEW... -i config.enc.yaml
sops --rotate -i config.enc.yaml
sops updatekeys -y config.enc.yaml
```

The metadata block in YAML mode is a sibling of your top-level keys; in JSON it lives at the root as `"sops": {...}`; in binary mode it lives outside the encrypted blob in the YAML wrapper. The block is critical — losing it makes the file undecryptable even with the right keys.

## Threat Model

What SOPS protects.

- Confidentiality of values. Every encrypted leaf is AES-256-GCM under a per-file dataKey. Without a recipient private key, the values are computationally indistinguishable from random.
- Integrity of values. The MAC over plaintext means a tampered ciphertext (or tampered metadata) fails decrypt with a verifiable error.
- Recipient revocation (with rotation). Removing a recipient from `.sops.yaml` and running `updatekeys` strips the wrapped dataKey for that recipient; combined with `--rotate`, the dataKey itself changes so the old recipient's old wrapped key is useless.

What SOPS does NOT protect.

- Structure and keys. A YAML key named `prod_db_password` reveals what kind of secret you have, even if the value is encrypted. Treat key names as public.
- Presence of secret. If a config file exists, an attacker knows secrets exist there. Use `.sops.yaml` to scope encryption away from non-secret files, but the encrypted file's existence is itself information.
- Comment confidentiality. Comments in YAML/TOML are NOT encrypted — never put secret material in a comment.
- Side-channel and timing. SOPS uses Go's standard crypto; attacker access to the host or memory while running SOPS exposes plaintext.

The comment-leakage caveat. YAML comments live outside the value tree, so SOPS leaves them plaintext. Repeat: do not write comments that contain rotated values, account hints, or anything that would help an attacker.

The key-leakage caveat. SOPS does not encrypt mapping keys; only values. If an attacker controlling the repository renames a key from `db_password` to `aws_secret`, your downstream tooling reading the decrypted file may behave unexpectedly. The MAC catches value tampering, not key renames at the YAML structural level (though the MAC includes the keys in its input on most format versions — check version-specific behaviour).

```bash
sops -d --extract '["sops"]["mac"]' config.enc.yaml
```

Defense in depth.

```bash
git diff secrets.enc.yaml
git log --oneline -- secrets.enc.yaml
git blame secrets.enc.yaml
```

Combine SOPS with code review (no PR can land without two approvals on a `secrets.enc.yaml` change), branch protection, and signed commits to raise the cost of malicious tampering even by an authorized recipient.

## File Formats — YAML deep

Default and the most fully-supported format. SOPS uses an in-house YAML parser that preserves comments, key ordering, and indentation across the encrypt-decrypt round trip — critical when the file is also human-edited.

Sample plaintext config.yaml has an `api` section with a `token` plus URL, a `database` section with `password` plus `host`, and inline comments throughout.

```bash
sops -e config.yaml > config.enc.yaml
```

After encryption the structure stays identical — `api.token` becomes `ENC[AES256_GCM,data:...]`, `api.url` becomes another ENC string if it matched the encrypt rule, and the `sops:` metadata appears as a sibling to the existing top-level keys.

```bash
sops -d config.enc.yaml | head -20
```

Comment preservation. A comment like `# rotated quarterly` placed above a key stays exactly where it was, byte-for-byte. After `sops config.enc.yaml` to edit, then save, comments survive.

Multi-document YAML support. The `---` document separator is honoured; each document gets its own encryption pass but they share the file-level metadata.

```bash
sops -e multi.yaml > multi.enc.yaml
sops -d multi.enc.yaml
```

Both documents encrypt; the metadata block lives at the end and applies across all documents.

Inline comments. Comments at end-of-line are also preserved. A line `token: secret  # rotate quarterly` becomes `token: ENC[...]  # rotate quarterly`.

Key ordering preserved. SOPS uses an order-preserving YAML decoder, so the encrypted file presents the same key order as the input. This matters for git diffs and for human readers.

Merge-key references. YAML `<<: *anchor` merge keys are honoured at parse time. SOPS encrypts the merged result; the anchor itself is not preserved across encrypt — the file is materialised. If you rely on YAML anchors for DRY, encrypt the source-of-truth and template-render the merged form.

```bash
sops -e config-with-anchors.yaml > config.enc.yaml
sops -d config.enc.yaml
```

YAML 1.1 vs 1.2 booleans. The bareword `yes`, `no`, `on`, `off` are booleans in YAML 1.1 (default) but strings in YAML 1.2. SOPS treats whatever the parser emits — if you depend on the literal string "on", quote it: `"on"`.

Long string handling. YAML supports `|` (literal) and `>` (folded) block scalars. SOPS preserves the chosen form when possible.

```bash
sops -d config.enc.yaml | grep -A 5 'cert:'
```

Tagged scalars. YAML tags like `!!str`, `!!int`, `!!binary` are preserved through encryption — SOPS records the type in the ENC string's `type:` suffix (`type:str`, `type:int`, etc.) so that decryption returns the original type.

```bash
sops -d --extract '["int_value"]' config.enc.yaml
```

## File Formats — JSON

JSON has no comment support, so SOPS metadata lives in a top-level `sops` key.

```bash
sops -e config.json > config.enc.json
cat config.enc.json | jq '.sops | keys'
```

After encryption a config.json with `{"api": {"token": "secret"}}` becomes `{"api": {"token": "ENC[AES256_GCM,...]"}, "sops": {...}}` with the metadata in the reserved `sops` key.

The `sops` key is reserved at the top level — if your application has a `sops:` field for unrelated reasons, rename it before encrypting or use a JSON wrapper.

JSON ordering. JSON has no spec-mandated key order, but SOPS preserves insertion order for diff stability. Most `jq` filters break this; pipe through `jq -S` only if you accept whole-file diffs.

JSON arrays. Arrays are encrypted element-by-element. The array length is visible (it's the structure), but element values are not.

```bash
sops -d --extract '["servers"][0]' config.enc.json
sops -d --extract '["servers"][1]' config.enc.json
```

Less metadata friendly. Without comments, JSON SOPS files are harder for humans to annotate. Some teams use a parallel `README.md` next to the `.enc.json` file to describe rotation policy and field meaning.

```bash
sops -d config.enc.json | jq .
sops -d --output-type yaml config.enc.json
```

Decrypt to a different output format for human reading.

## File Formats — TOML

TOML is parsed via go-toml; tables, sub-tables, and arrays-of-tables are preserved.

```bash
sops -e config.toml > config.enc.toml
sops -d config.enc.toml
```

TOML inline tables `{ x = 1, y = 2 }` are flattened on encrypt — be aware if you have downstream parsers that distinguish inline from block tables.

TOML arrays-of-tables `[[server]]` work cleanly. The structure is preserved; only the values inside each `[[server]]` block are encrypted.

```bash
sops -d --extract '["server"][0]["pass"]' config.enc.toml
```

TOML comments are preserved similar to YAML; place comments above the key they describe.

```bash
sops --encrypted-regex '^(pass|secret|token)' -e config.toml > config.enc.toml
```

TOML datetime values are preserved as their native type through encryption.

## File Formats — INI / dotenv

Line-oriented formats. Each `KEY=VALUE` line is treated independently — the value is encrypted, the key remains plaintext.

```bash
sops -e --input-type dotenv .env > .env.enc
cat .env.enc
```

The encrypted `.env.enc` shows `KEY=ENC[AES256_GCM,...]` for each line, plus a series of `sops_*` lines at the bottom containing the metadata flattened into the dotenv namespace (`sops_age__list_0__map_recipient`, `sops_lastmodified`, `sops_mac`, `sops_version`).

Encrypted-line vs key-encrypted. In dotenv, the entire VALUE side of `KEY=VALUE` is encrypted; the KEY name is plaintext. There is no concept of a fully-encrypted line in dotenv mode — line structure is what makes the file parseable.

INI mode is similar. SOPS treats `[section]` headers as plaintext structure and encrypts only the values.

```bash
sops -e --input-type ini config.ini > config.enc.ini
sops -d --input-type ini config.enc.ini
```

Auto-detection. SOPS picks the format from the file extension (`.yaml`, `.json`, `.toml`, `.ini`, `.env`). For files without recognised extensions, pass `--input-type` explicitly.

```bash
sops -e --input-type dotenv production > production.enc
sops -e --input-type ini app.conf > app.conf.enc
```

Dotenv quoting. Values containing spaces, equals, or special characters should be quoted in plaintext; SOPS preserves the quote style.

```bash
sops -d --input-type dotenv .env.enc | grep DATABASE_URL
```

## File Formats — binary

For files that aren't structured (binary blobs, JKS keystores, opaque ciphertexts, .pem files you want to treat as opaque). SOPS wraps the bytes in a YAML envelope.

```bash
sops -e --input-type binary --output-type yaml keystore.jks > keystore.enc.yaml
```

Inside `keystore.enc.yaml` is a `data: ENC[AES256_GCM,data:<base64>,iv:...,tag:...,type:str]` line plus the standard `sops:` metadata block.

Decrypting binary.

```bash
sops -d --input-type yaml --output-type binary keystore.enc.yaml > keystore.jks
```

Or the shorter form, leveraging the `binary` extension hint via `.binary`.

```bash
mv keystore.jks keystore.binary
sops -e keystore.binary > keystore.binary.enc
sops -d keystore.binary.enc > keystore.binary
```

The base64-armored output. Inside the YAML wrapper, the binary content is base64-encoded then encrypted, then placed in a `data:` ENC string. The wrapper is ASCII-safe and git-storable.

The binary-ish file in YAML wrapper. A common pattern is to keep the encrypted file with extension `.yaml` for `.sops.yaml` rule-matching simplicity, but tag with `--input-type binary` so SOPS treats the source as opaque.

```bash
sops -e --input-type binary --output-type yaml server.pem > server.pem.enc.yaml
sops -d --input-type yaml --output-type binary server.pem.enc.yaml > server.pem
```

This pattern is essential for TLS certificates, SSH keys, keystores, and any file where line-by-line encryption would corrupt the format.

```bash
file keystore.jks
sops -e --input-type binary keystore.jks > keystore.jks.enc.yaml
sops -d --output-type binary keystore.jks.enc.yaml > /tmp/decrypted.jks
diff keystore.jks /tmp/decrypted.jks && echo round-trip-clean
```

## .sops.yaml — Top-Level

The `.sops.yaml` file lives at the root of a repository (or in any ancestor directory of the file being encrypted). SOPS walks up from the file location looking for `.sops.yaml`; the first one found is used.

```bash
ls -la .sops.yaml
find . -name '.sops.yaml'
```

A typical repository layout has `.sops.yaml` at the root, with optional overrides in subdirectories like `apps/api/.sops.yaml` for files under that path.

The `creation_rules` array. The fundamental schema: a list of rules, each with a `path_regex` and a recipient set. First matching rule wins; rules are tried in order.

A reasonable production `.sops.yaml` has rules ordered most-specific-first.

```bash
cat .sops.yaml
```

Other top-level options at rule scope.

- `path_regex` — Go regex matched against the file path relative to `.sops.yaml`. Without anchors, partial matches; use `^...$` for full anchoring.
- `encrypted_regex` — only keys matching encrypt; non-matching keys stay plaintext.
- `unencrypted_regex` — only keys NOT matching encrypt.
- `encrypted_suffix` — keys ending in this suffix encrypt; others plaintext.
- `unencrypted_suffix` — keys ending in this suffix stay plaintext; others encrypt.
- `mac_only_encrypted: true` — MAC computed only over encrypted values, not all values. Lets you edit unencrypted parts without retripping the MAC. Default: false.
- `key_groups` — Shamir threshold groups (see Key Groups section).

A reasonable production `.sops.yaml` template.

```bash
sops -e prod/secrets.yaml > prod/secrets.enc.yaml
sops -e dev/secrets.yaml > dev/secrets.enc.yaml
```

Each rule applies its own `encrypted_regex` and recipient set; SOPS picks the first match by traversing `creation_rules` in order.

## .sops.yaml — creation_rules

Recipient blocks. Each rule can declare any combination of recipient types. SOPS encrypts the dataKey to all of them; any one suffices for decryption.

PGP recipients are listed as comma-separated 40-hex fingerprints in the `pgp:` field. Multiple fingerprints can be joined on one line or split across multiple lines using YAML block scalar syntax.

```bash
gpg --list-secret-keys --keyid-format LONG
```

age recipients are comma-separated bech32 strings beginning with `age1...` in the `age:` field. The age public key is short (62 chars), so most teams put each recipient on its own line for readability.

```bash
age-keygen -y < ~/.config/sops/age/keys.txt
```

AWS KMS recipients are comma-separated ARNs in the `kms:` field. In older SOPS docs you'll see `aws_kms`; both work in 3.7+.

```bash
aws kms list-keys --region us-east-1
aws kms describe-key --key-id arn:aws:kms:us-east-1:111:key/abc-123
```

Azure Key Vault recipients are comma-separated key URLs in `azure_keyvault:`. The URL has the form `https://<vault>.vault.azure.net/keys/<keyname>/<keyversion>`.

```bash
az keyvault key show --vault-name my-vault --name sops-key --query key.kid
```

GCP KMS recipients are comma-separated CryptoKey resource names in `gcp_kms:`. The format is `projects/<PROJECT>/locations/<LOCATION>/keyRings/<RING>/cryptoKeys/<KEY>`.

```bash
gcloud kms keys list --keyring sops --location global
```

HashiCorp Vault Transit recipients are comma-separated transit URIs in `hc_vault_transit_uri:`. The URI ends with `/v1/sops/keys/<keyname>` (or whatever path the transit engine is mounted at).

```bash
vault list sops/keys
vault read sops/keys/firstkey
```

A fully-loaded `.sops.yaml` mixing all recipient types covers a single rule with `age`, `pgp`, `kms`, `gcp_kms`, `azure_keyvault`, and `hc_vault_transit_uri` fields. Any single recipient suffices for decrypt; redundancy survives single-provider outages.

`key_groups` for threshold scenarios — covered in the next section.

## .sops.yaml — Key Groups

Shamir-style threshold. Instead of a single recipient set where any one recipient can decrypt, split the dataKey into N parts and require K-of-N parts (threshold K) to reconstitute.

A `.sops.yaml` rule with `shamir_threshold: 2` and three `key_groups` entries (each containing its own age/pgp/kms recipients) requires any 2 of the 3 groups to decrypt.

```bash
sops -d highly-sensitive/secret.enc.yaml
```

Decrypt mechanics. SOPS attempts each group; if K succeed, the dataKey reconstitutes and decryption proceeds. If fewer than K succeed, you get an error like `Error getting data key: 0 successful groups required, got 1`.

Single-recipient compromise resistance. Without key_groups, leaking one recipient key compromises every file readable by that key. With a 2-of-3 threshold, a single key leak is recoverable: rotate the leaked key and `updatekeys` all files; the other two groups still allow legitimate decrypt during the transition.

Each group can have its own recipient mix. Group A might be the security team with PGP smartcards or age keys; group B SREs with age; group C a KMS-backed break-glass. Threshold 2 means any two of (security, SRE, KMS) can decrypt.

Encrypting with key_groups from CLI (rather than .sops.yaml).

```bash
sops -e \
  --shamir-secret-sharing-threshold 2 \
  --shamir-secret-sharing-quorum 3 \
  --age age1A...,age1B... \
  --pgp 85D7... \
  --kms arn:aws:kms:us-east-1:111:key/break-glass \
  config.yaml > config.enc.yaml
```

The CLI groups all recipients into a single key_group when using only flags. For true multi-group at create time, use `.sops.yaml` (which has full schema support including grouped key_groups).

```bash
sops -d --extract '["sops"]["key_groups"]' config.enc.yaml
```

Inspecting an encrypted file to see its key groups and threshold.

```bash
sops -d --extract '["sops"]["shamir_threshold"]' config.enc.yaml
```

## Encrypting

Basic encrypt to stdout.

```bash
sops -e file.yaml > file.enc.yaml
```

In-place encrypt (overwrites the file). Useful when you've created a plaintext file and want it encrypted in place.

```bash
sops -e -i file.yaml
ls -la file.yaml
```

After this, file.yaml is encrypted; the plaintext is gone.

Override recipients on the command line, ignoring `.sops.yaml`.

```bash
sops -e --age age1u9q6r5d8j3p2k4lvwetszqyt7gpz0fvy6lnnxh3v9k02j5fa6vys9zrf6h config.yaml > config.enc.yaml
sops -e --pgp 85D77543B3D624B63CEA9E6DBC17301B491B3F21 config.yaml > config.enc.yaml

sops -e \
  --age age1u9q6r5d8j3p2k4lvwetszqyt7gpz0fvy6lnnxh3v9k02j5fa6vys9zrf6h \
  --pgp 85D77543B3D624B63CEA9E6DBC17301B491B3F21 \
  config.yaml > config.enc.yaml
```

AWS KMS specifics with profile and region.

```bash
sops -e --aws-profile prod --aws-region us-east-1 \
  --kms arn:aws:kms:us-east-1:111111111111:key/abc-123 \
  config.yaml > config.enc.yaml
```

Cross-account KMS via role assumption with the `+arn:aws:iam::...` extension.

```bash
sops -e \
  --kms 'arn:aws:kms:us-east-1:111111111111:key/abc-123+arn:aws:iam::111111111111:role/sops-encryptor' \
  config.yaml > config.enc.yaml
```

GCP KMS.

```bash
sops -e \
  --gcp-kms projects/my-proj/locations/global/keyRings/sops/cryptoKeys/prod \
  config.yaml > config.enc.yaml
```

Azure Key Vault.

```bash
sops -e \
  --azure-kv https://my-vault.vault.azure.net/keys/sops-key/abc123def456 \
  config.yaml > config.enc.yaml
```

HashiCorp Vault Transit.

```bash
sops -e \
  --hc-vault-transit https://vault.example.com:8200/v1/sops/keys/firstkey \
  config.yaml > config.enc.yaml
```

Selective encryption with regex on the command line.

```bash
sops -e \
  --encrypted-regex '^(password|secret|key|token)' \
  --age age1A... \
  config.yaml > config.enc.yaml
```

Encrypted-suffix.

```bash
sops -e \
  --encrypted-suffix '_enc' \
  --age age1A... \
  config.yaml > config.enc.yaml
```

Only keys like `password_enc:`, `token_enc:` encrypt; others stay plaintext.

Input/output type override.

```bash
sops -e --input-type yaml --output-type json config.txt > config.enc.json
sops -e --input-type binary --output-type yaml blob.yaml > blob.enc.yaml
```

Encrypt from stdin.

```bash
echo "password: hunter2" | sops -e --input-type yaml /dev/stdin > pw.enc.yaml
```

The `--unencrypted-comment-regex` flag (3.8+) strips secrets from comments before encrypting.

```bash
sops -e --unencrypted-comment-regex '^# rotated' config.yaml > config.enc.yaml
```

Encrypt a file under a non-default `.sops.yaml` location.

```bash
sops --config /path/to/.sops.yaml -e file.yaml > file.enc.yaml
```

## Decrypting

Basic decrypt to stdout.

```bash
sops -d file.enc.yaml
```

Decrypt in-place (replaces encrypted file with plaintext — usually you want stdout instead).

```bash
sops -d -i file.enc.yaml
```

Now file.enc.yaml contains plaintext. Use with care — the encrypted form is gone unless you have it in git.

Selective decryption with `--extract`. The path syntax is JSON-pointer-ish: each segment in `[""]` brackets, integer indices unbracketed quotes.

```bash
sops -d --extract '["api"]["token"]' config.enc.yaml
sops -d --extract '["servers"][0]["host"]' config.enc.yaml
```

Combined extract + bash assignment (the most common pattern).

```bash
TOKEN=$(sops -d --extract '["api"]["token"]' config.enc.yaml)
curl -H "Authorization: Bearer $TOKEN" https://api.example.com
```

Recipient discovery and first-success behavior. SOPS reads the metadata, lists every recipient (age public keys, PGP fingerprints, KMS ARNs, etc.), and tries each with the keys available locally. The first to succeed yields the dataKey; all others are skipped.

Local key discovery order for age.

```bash
export SOPS_AGE_KEY='AGE-SECRET-KEY-1...'
export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt
ls ~/.config/sops/age/keys.txt
```

The order is: 1) `SOPS_AGE_KEY` env (literal contents), 2) `SOPS_AGE_KEY_FILE` env (path), 3) the default age key file at `~/.config/sops/age/keys.txt` (Linux/macOS) or `%AppData%\sops\age\keys.txt` (Windows).

Local key discovery for PGP uses the system gpg agent; no env var needed if your private key is in the gpg keyring.

```bash
gpg --list-secret-keys
sops -d file.enc.yaml
```

Local key discovery for KMS uses the standard AWS SDK credential chain.

```bash
export AWS_PROFILE=prod
sops -d file.enc.yaml
```

The chain is: `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` env, `~/.aws/credentials` (profile from `AWS_PROFILE` or `default`), then EC2 instance / EKS pod / Lambda role.

Decrypt with a specific output format.

```bash
sops -d --output-type json config.enc.yaml > config.json
sops -d --output-type binary keystore.enc.yaml > keystore.jks
```

Decrypt with a specific recipient profile (rare; usually managed by .sops.yaml).

```bash
sops -d --aws-profile prod file.enc.yaml
```

`SOPS_AGE_KEY` env var with literal key. The contents are the bech32 secret key starting with `AGE-SECRET-KEY-1...`. Useful in CI runners where you want zero-disk persistence.

```bash
SOPS_AGE_KEY='AGE-SECRET-KEY-1...' sops -d secrets.enc.yaml
```

`AGE_KEY_FILE` env var (alias used by some integrations). SOPS itself prefers `SOPS_AGE_KEY_FILE`, but several tools (sops-nix, KSOPS) also honour `AGE_KEY_FILE`. Set both for maximum compatibility.

```bash
export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt
export AGE_KEY_FILE=~/.config/sops/age/keys.txt
```

Decrypt to a specific path.

```bash
sops -d file.enc.yaml > /tmp/file.yaml
chmod 600 /tmp/file.yaml
```

## Editing

In-place edit. SOPS launches `$EDITOR` on a temp file containing the decrypted plaintext; on save and exit, SOPS re-encrypts and writes back.

```bash
sops file.enc.yaml
```

`$EDITOR` opens with plaintext. Edit, save, quit. SOPS re-encrypts and replaces file.enc.yaml.

Choosing an editor.

```bash
export EDITOR='vim'
EDITOR='code -w' sops file.enc.yaml
EDITOR='nano' sops file.enc.yaml
EDITOR='nvim' sops file.enc.yaml
EDITOR='micro' sops file.enc.yaml
```

The temp file lifecycle. SOPS creates a temp file under the system temp dir (`/tmp` on Linux, `$TMPDIR` on macOS), writes plaintext, runs `$EDITOR`, reads the modified plaintext on editor exit, encrypts to the original path, and unlinks the temp. If `$EDITOR` crashes mid-edit, the temp may persist — SOPS warns but cannot guarantee cleanup.

```bash
ls -la $TMPDIR/sops*
```

Editor save mechanics. SOPS detects "modified vs unmodified" by hashing the temp file's contents on entry and exit. If you save without changes, no re-encryption happens (the dataKey and ciphertexts stay identical, preserving git diffs at zero).

The `SOPS_PGP_FP` env override.

```bash
export SOPS_PGP_FP='85D77543B3D624B63CEA9E6DBC17301B491B3F21,1A2B3C4D5E6F7A8B9C0D1E2F3A4B5C6D7E8F9A0B'
sops -e file.yaml > file.enc.yaml
```

Now sops -e uses these PGP fingerprints regardless of `.sops.yaml`.

`--enable-local-keyservice` flag. Routes key operations through a local SOPS keyservice (gRPC daemon), useful for environments where the user's gpg-agent is on a different host or for delegated access.

```bash
sops --enable-local-keyservice file.enc.yaml
```

`--keyservice` flag for delegated key ops. Point SOPS at a remote keyservice instance; the keyservice handles age/pgp/kms operations on behalf of the SOPS process.

```bash
sops keyservice --network tcp --address 0.0.0.0:5000 &
sops --keyservice tcp://10.0.0.5:5000 file.enc.yaml
```

The keyservice never sees plaintext values; it only sees the dataKey wrap/unwrap requests.

```bash
sops --enable-local-keyservice --keyservice unix:///tmp/sops.sock file.enc.yaml
```

Sockets work too for local-only keyservice flows.

## Updating Keys

`sops updatekeys` re-encrypts the dataKey to the current recipient list from `.sops.yaml` without touching the value ciphertexts.

```bash
sops updatekeys file.enc.yaml
```

Mechanics. SOPS reads the file's current metadata, decrypts the dataKey using one available recipient, then re-encrypts that same dataKey to the new recipient list (as resolved from `.sops.yaml`). The value ciphertexts are unchanged because the dataKey is unchanged. Git sees only the metadata diff.

```bash
git diff file.enc.yaml
```

Idempotent. Running `updatekeys` twice in a row is a no-op (after the first run the recipient list matches `.sops.yaml`; the second run sees no diff).

Bulk update.

```bash
find . -name '*.enc.yaml' -exec sops updatekeys -y {} \;
```

The `-y` flag means "yes, skip confirmation prompts." By default `updatekeys` shows the diff and asks Y/n before writing.

`--input-type` for non-YAML extensions.

```bash
sops updatekeys --input-type json secrets.json
```

Safe across `git status`. Because only metadata changes, you can run `updatekeys` on a clean working tree and the resulting diff is contained, reviewable, and reversible.

```bash
git stash
sops updatekeys -y file.enc.yaml
git diff
git checkout file.enc.yaml
git stash pop
```

Roll forward only the recipient envelope without touching value ciphertexts.

## Rotating

`sops --rotate -i file.enc.yaml` generates a new dataKey and re-encrypts every value with it.

```bash
sops --rotate -i file.enc.yaml
```

All ciphertexts change; all wrapped dataKeys change; the MAC changes.

When to rotate the dataKey:

- You suspect compromise of any single recipient key.
- A team member with access has left.
- Routine rotation policy (quarterly is common).
- Before removing a recipient from the file (rotate, then `--rm-pgp` or edit `.sops.yaml` + `updatekeys`).

Combined with `updatekeys` for full rotation.

```bash
sops --rotate -i file.enc.yaml
sops updatekeys -y file.enc.yaml
```

Rotate-only without recipient change.

```bash
sops --rotate -i file.enc.yaml
```

Same recipients, brand-new dataKey.

Bulk rotation across a tree.

```bash
find . -name '*.enc.yaml' -exec sops --rotate -i {} \;
find . -name '*.enc.yaml' -exec sops updatekeys -y {} \;
```

The git-diff after `--rotate` is huge — every value's ciphertext changes. Plan rotation as its own commit ("rotate dataKey for prod/secrets.enc.yaml") to keep diffs reviewable.

```bash
git diff --stat file.enc.yaml
```

## Add / Remove Keys

Per-file recipient changes without touching `.sops.yaml`.

```bash
sops --add-pgp 85D77543B3D624B63CEA9E6DBC17301B491B3F21 -i file.enc.yaml
sops --rm-pgp 1A2B3C4D5E6F7A8B9C0D1E2F3A4B5C6D7E8F9A0B -i file.enc.yaml

sops --add-age age1NEW... -i file.enc.yaml
sops --rm-age age1OLD... -i file.enc.yaml

sops --add-kms arn:aws:kms:us-east-1:111:key/new -i file.enc.yaml
sops --rm-kms arn:aws:kms:us-east-1:111:key/old -i file.enc.yaml

sops --add-gcp-kms projects/p/locations/global/keyRings/r/cryptoKeys/new -i file.enc.yaml
sops --rm-gcp-kms projects/p/locations/global/keyRings/r/cryptoKeys/old -i file.enc.yaml

sops --add-azure-kv https://v.vault.azure.net/keys/k/new -i file.enc.yaml
sops --rm-azure-kv https://v.vault.azure.net/keys/k/old -i file.enc.yaml

sops --add-hc-vault-transit https://vault:8200/v1/sops/keys/new -i file.enc.yaml
sops --rm-hc-vault-transit https://vault:8200/v1/sops/keys/old -i file.enc.yaml
```

These add/remove flags modify the recipient set without rotating the dataKey. The same dataKey is re-wrapped to the new recipient list. This is NOT a true rotation — if the removed recipient already saved the dataKey somewhere, they retain access to the value ciphertexts.

Pair with `--rotate` for true rotation.

```bash
sops --rotate --rm-pgp 1A2B3C4D5E6F7A8B9C0D1E2F3A4B5C6D7E8F9A0B -i file.enc.yaml
```

Old recipient: had dataKey, but it no longer matches the new ciphertexts.

Multiple add/remove in one invocation.

```bash
sops \
  --add-age age1NEW... \
  --rm-age age1OLD... \
  --add-kms arn:aws:kms:us-east-1:111:key/new \
  --rotate \
  -i file.enc.yaml
```

The flag list applies in order: add new recipients, remove old recipients, rotate dataKey. The result is a re-encrypted file with the new recipient list and a fresh dataKey.

## exec-env / exec-file

Run a command with secrets injected as environment variables.

```bash
sops exec-env file.enc.yaml 'cmd $VAR1 $VAR2'
```

Mechanics. SOPS decrypts `file.enc.yaml`, walks the leaves, sets each as an env var (key = uppercased version, value = decrypted plaintext), then exec's the given command. Plaintext never touches disk.

```bash
sops exec-env secrets.enc.yaml 'env | grep DATABASE_URL'
sops exec-env secrets.enc.yaml 'my-app --port 8080'
```

Run a command with a temp file path containing decrypted secrets.

```bash
sops exec-file file.enc.yaml 'cmd {}'
```

The `{}` placeholder. SOPS decrypts to a temp file (or FIFO on Unix), substitutes `{}` in the command with the temp file path, runs the command, then deletes the temp file on exit.

```bash
sops exec-file config.enc.yaml 'kubectl apply -f {}'
sops exec-file ansible-vars.enc.yml 'ansible-playbook -e @{} site.yml'
sops exec-file values.enc.yaml 'helm upgrade my-release ./chart -f {}'
```

`--user` flag drops privileges before exec'ing.

```bash
sops exec-env --user www-data secrets.enc.yaml 'my-server'
```

`my-server` runs as `www-data` with secrets in env.

`--no-fifo` for tmpfile mode. By default `exec-file` uses a named pipe (FIFO) on Unix to avoid plaintext on disk; `--no-fifo` falls back to a real temp file.

```bash
sops exec-file --no-fifo secrets.enc.yaml 'cat {}'
```

Useful when the consuming command can't read from a FIFO (e.g., tools that mmap or seek).

`--background` flag. Runs the command in the background; SOPS waits for it to exit before cleaning up the temp file.

```bash
sops exec-file --background config.enc.yaml 'long-running-server --config {}'
```

Combining exec-env with downstream tooling.

```bash
sops exec-env .env.enc 'docker compose up -d'
sops exec-env vars.enc.yaml 'terraform apply -auto-approve'
sops exec-env prod.enc.yaml 'kubectl rollout restart deployment/api'
```

## Selective Decryption

`sops -d --extract '["path"]["to"]["value"]' file.enc.yaml`

Single-value retrieval. The bracket-quote syntax mirrors JSON pointer with array indices unbracketed.

```bash
sops -d --extract '["api"]["token"]' file.enc.yaml
sops -d --extract '["api"]["endpoints"][0]["prod"]' file.enc.yaml
```

The JSONPath-ish syntax. Each path segment is `[ "key" ]` for map keys and `[ N ]` for array indices. Quotes around keys are required (single or double).

```bash
sops -d --extract '["a"]["b"]' file.enc.yaml
sops -d --extract "['a']['b']" file.enc.yaml
```

Edge cases.

```bash
sops -d --extract '["api.example.com"]["token"]' file.enc.yaml
sops -d --extract '["weird[key]"]' file.enc.yaml
```

Combined extract + bash variable.

```bash
DB_PASS=$(sops -d --extract '["database"]["password"]' file.enc.yaml)
PGPASSWORD=$(sops -d --extract '["database"]["password"]' file.enc.yaml) psql -U user -d mydb -h db.internal
```

Extract and pipe.

```bash
sops -d --extract '["tls"]["cert"]' file.enc.yaml | openssl x509 -text
sops -d --extract '["ssh"]["private_key"]' file.enc.yaml > /tmp/sshkey
chmod 600 /tmp/sshkey
ssh -i /tmp/sshkey user@host
shred -u /tmp/sshkey
```

Extract a list of values for a comma-separated env.

```bash
sops -d --extract '["allowed_ips"]' file.enc.yaml | tr '\n' ','
```

The output for arrays is one element per line; pipe through `tr`/`paste`/`jq` to reshape.

## Integration with age

The age-as-recipient flow. age (`github.com/FiloSottile/age`) is a modern, simple alternative to PGP, with no web-of-trust, no subkeys, no expiration metadata. SOPS supports age natively.

Generate an age keypair.

```bash
mkdir -p ~/.config/sops/age
age-keygen -o ~/.config/sops/age/keys.txt
chmod 600 ~/.config/sops/age/keys.txt
```

The generated file contains a `# public key:` comment followed by `AGE-SECRET-KEY-1...`. The public key (starting `age1...`) is what you put in `.sops.yaml` and share with the team.

Add the public key to `.sops.yaml` under the `age:` field of a `creation_rules` entry.

```bash
age-keygen -y < ~/.config/sops/age/keys.txt
```

The above prints the public key derived from the secret key file.

`SOPS_AGE_KEY_FILE` env. The default location SOPS reads is `~/.config/sops/age/keys.txt`; override per-process.

```bash
export SOPS_AGE_KEY_FILE=/secure/age/keys.txt
sops -d file.enc.yaml
```

The recipient-rotation flow.

```bash
age-keygen -o ~/.config/sops/age/keys.new.txt
age-keygen -y < ~/.config/sops/age/keys.new.txt

find . -name '*.enc.yaml' -exec sops updatekeys -y {} \;

SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.new.txt sops -d file.enc.yaml > /dev/null

find . -name '*.enc.yaml' -exec sops --rotate -i {} \;
```

After verifying, update `.sops.yaml` to remove the old recipient, run `updatekeys` again, and optionally rotate dataKeys to invalidate any saved old wrappings.

Multi-recipient age.

```bash
sops -e --age 'age1A...,age1B...,age1C...' file.yaml > file.enc.yaml
```

Comma-separated public keys; SOPS wraps the dataKey to each.

The keys file format. `~/.config/sops/age/keys.txt` is one age private key per line, comments allowed.

```bash
cat ~/.config/sops/age/keys.txt
```

Output shows lines like `AGE-SECRET-KEY-1...` separated by `# public key: age1...` comment lines.

age recipient lookup at decrypt time. SOPS tries each private key in `keys.txt` against each `age:` entry in the file metadata. The first match yields the dataKey.

```bash
sops --age 'age1A...,age1B...' --decrypt file.enc.yaml
```

ssh-to-age conversion. SSH ed25519 keys can be converted to age keys, useful for zero-setup recipient lists.

```bash
ssh-to-age < ~/.ssh/id_ed25519.pub
ssh-to-age -private-key < ~/.ssh/id_ed25519
```

The first prints the age public key derived from the SSH public key; the second prints the age private key from the SSH private key. Pipe into `~/.config/sops/age/keys.txt` to use SSH-derived age identities.

## Integration with PGP

The classic recipient type, supported since SOPS 1.x.

```bash
gpg --list-secret-keys --keyid-format LONG
```

Output shows a `sec` line with `rsa4096/B3D624B63CEA9E6D` and a 40-hex fingerprint on the next line. The 40-hex string is what SOPS uses.

Add to `.sops.yaml` under the `pgp:` field of `creation_rules`.

```bash
sops -e --pgp 85D77543B3D624B63CEA9E6DBC17301B491B3F21 file.yaml > file.enc.yaml
```

The gpg-agent role. SOPS does not implement PGP cryptography itself; it shells out to `gpg` (or `gpg2`). The local `gpg-agent` caches the passphrase, manages keyrings, and signs/decrypts. SOPS expects:

- `gpg` in `$PATH` — verify with `which gpg`.
- A running `gpg-agent` (started automatically on most distros).
- The private key for at least one recipient in your keyring.

```bash
gpg --version
echo test | gpg --encrypt -r 85D77543B3D624B63CEA9E6DBC17301B491B3F21 | gpg --decrypt
```

`SOPS_PGP_FP` env. Override the fingerprint list.

```bash
export SOPS_PGP_FP='85D77543B3D624B63CEA9E6DBC17301B491B3F21'
sops -e file.yaml > file.enc.yaml
```

Smartcard / Yubikey + gpg + sops flow.

```bash
gpg --card-status
sops -d file.enc.yaml
```

`gpg --card-status` shows the Application ID and key fingerprints on the card. SOPS uses the master key fingerprint; gpg-agent routes to the card and prompts for PIN/touch on each decrypt.

Long-form expired-key handling. PGP keys can expire; SOPS does not auto-detect this. If your subkey expires, decryption fails with `gpg: decryption failed: No secret key`. Extend the expiration or rotate to a new fingerprint and `updatekeys`.

```bash
gpg --edit-key 85D77543B3D624B63CEA9E6DBC17301B491B3F21
```

In the gpg> prompt, use `key 1` to select the encryption subkey, `expire` to extend, and `save` to commit.

Importing a teammate's PGP public key.

```bash
gpg --keyserver keys.openpgp.org --recv-keys 85D77543B3D624B63CEA9E6DBC17301B491B3F21
gpg --import < teammate.asc
```

After import, run `sops updatekeys -y file.enc.yaml` to add their fingerprint to the file's recipient list.

Trust level. SOPS only requires that gpg can encrypt to the key, not that you've signed it; trust level affects gpg's prompt-on-encrypt behavior.

```bash
gpg --edit-key 85D77543B3D624B63CEA9E6DBC17301B491B3F21
```

Use `trust` then `5` for ultimate (or `4` for full) trust to suppress gpg's "not certified with a trusted signature" warning.

## Integration with AWS KMS

The IAM permissions required.

```bash
aws iam get-policy-version --policy-arn arn:aws:iam::111:policy/sops-policy --version-id v1
```

A minimal SOPS policy needs `kms:Encrypt`, `kms:Decrypt`, `kms:GenerateDataKey` on the key resource. Encrypt operations need all three; decrypt needs only `kms:Decrypt`.

Encrypt with explicit KMS ARN.

```bash
sops -e \
  --kms arn:aws:kms:us-east-1:111111111111:key/abc-123 \
  --aws-profile prod \
  --aws-region us-east-1 \
  config.yaml > config.enc.yaml
```

Multi-region key replicas. KMS multi-region keys (`mrk-*`) replicate ciphertext across regions; SOPS treats each replica ARN as a separate recipient.

```bash
sops -e \
  --kms 'arn:aws:kms:us-east-1:111:key/mrk-abc,arn:aws:kms:eu-west-1:111:key/mrk-abc' \
  config.yaml > config.enc.yaml
```

Either region's KMS endpoint can decrypt.

KMS Grants for sub-account access. Rather than IAM policies, KMS Grants give time-bounded delegated access to a key, useful for sub-account or cross-account decrypt.

```bash
aws kms create-grant \
  --key-id arn:aws:kms:us-east-1:111:key/abc \
  --grantee-principal arn:aws:iam::222:role/sops-reader \
  --operations Decrypt
```

Cross-account encrypt with role assumption.

```bash
sops -e \
  --kms 'arn:aws:kms:us-east-1:111:key/abc+arn:aws:iam::111:role/sops-encryptor' \
  config.yaml > config.enc.yaml
```

The `+arn:aws:iam::...` after the KMS ARN tells SOPS to assume that role before calling KMS.

KMS encryption context. SOPS supports KMS encryption context (additional auth data for the wrapped dataKey). Set via the colon-delimited extension.

```bash
sops -e \
  --kms 'arn:aws:kms:us-east-1:111:key/abc:app=billing,env=prod' \
  config.yaml > config.enc.yaml
```

The `:app=billing,env=prod` part is the encryption context, baked into the wrapped dataKey. Decrypt requires IAM that allows Decrypt with matching context, otherwise KMS returns InvalidCiphertextException.

```bash
sops -d config.enc.yaml
```

Fails with `InvalidCiphertextException` if the IAM policy doesn't permit the context, or if the context was tampered.

EC2 / EKS / Lambda IAM. The instance role or pod identity needs `kms:Decrypt`; SOPS picks up the credentials from the SDK chain.

```bash
aws sts get-caller-identity
sops -d file.enc.yaml
```

## Integration with Azure Key Vault

The X.509 service-principal auth. Azure auth uses one of:

- `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` env vars (service principal).
- `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_CERTIFICATE_PATH` (cert auth).
- Managed Identity on Azure-hosted runners.
- `az login` for local dev.

```bash
export AZURE_TENANT_ID=11111111-2222-3333-4444-555555555555
export AZURE_CLIENT_ID=66666666-7777-8888-9999-000000000000
export AZURE_CLIENT_SECRET='your-secret'

sops -e \
  --azure-kv https://my-vault.vault.azure.net/keys/sops-key/abc123def456 \
  config.yaml > config.enc.yaml
```

The vault key URL format is `https://<vault-name>.vault.azure.net/keys/<key-name>/<key-version>`. The version is required at encrypt time; SOPS embeds it in metadata.

```bash
az keyvault key show --vault-name my-vault --name sops-key --query key.kid
```

Soft-delete + purge protection. Azure Key Vault has a soft-delete feature: deleted keys enter a 7-90 day retention window before permanent deletion. With purge protection enabled, deleted keys cannot be purged before the retention period — useful for accidentally-deleted SOPS recipients.

```bash
az keyvault key recover --vault-name my-vault --name sops-key
```

After recovery, `sops -d` works again.

Permissions on the key. The service principal needs `keys/decrypt` and `keys/encrypt` permissions, granted via Key Vault access policy or RBAC.

```bash
az keyvault set-policy \
  --name my-vault \
  --spn $AZURE_CLIENT_ID \
  --key-permissions encrypt decrypt
```

Or via RBAC.

```bash
az role assignment create \
  --role 'Key Vault Crypto User' \
  --assignee $AZURE_CLIENT_ID \
  --scope /subscriptions/.../resourceGroups/.../providers/Microsoft.KeyVault/vaults/my-vault
```

Managed Identity on AKS / Azure Container Apps. SOPS uses the SDK's default credential chain, which includes the metadata endpoint at `169.254.169.254`.

```bash
sops -d file.enc.yaml
```

If running with Azure Workload Identity, set `AZURE_CLIENT_ID` to the workload identity's client ID and SOPS picks it up automatically.

## Integration with GCP KMS

The auth flow.

```bash
gcloud auth application-default login
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/sa.json
```

Local development uses ADC (Application Default Credentials) at `~/.config/gcloud/application_default_credentials.json`. CI uses a service account JSON key file. GKE with Workload Identity is automatic (no env var).

Key-ring + key location.

```bash
gcloud kms keyrings create sops --location global
gcloud kms keys create prod --keyring sops --location global --purpose encryption
```

The resource name format is `projects/<PROJECT>/locations/<LOCATION>/keyRings/<KEYRING>/cryptoKeys/<KEY>`.

Encrypt.

```bash
sops -e \
  --gcp-kms projects/my-proj/locations/global/keyRings/sops/cryptoKeys/prod \
  config.yaml > config.enc.yaml
```

KMS-IAM bindings. The service account or user needs `roles/cloudkms.cryptoKeyEncrypterDecrypter` on the key.

```bash
gcloud kms keys add-iam-policy-binding prod \
  --keyring sops \
  --location global \
  --member 'serviceAccount:sops-sa@my-proj.iam.gserviceaccount.com' \
  --role roles/cloudkms.cryptoKeyEncrypterDecrypter
```

Per-key version. Unlike Azure, GCP KMS keys auto-rotate the primary version; SOPS works with the key resource (any active version decrypts ciphertext encrypted under any version).

```bash
gcloud kms keys versions list --key prod --keyring sops --location global
```

Workload Identity Federation. For CI without service account JSON keys, federate the CI provider's OIDC token to a GCP service account.

```bash
gcloud iam workload-identity-pools providers create-oidc github-actions \
  --location global \
  --workload-identity-pool ci-pool \
  --issuer-uri https://token.actions.githubusercontent.com
```

In the runner, `gcloud auth login` with the federated token, then `sops -d file.enc.yaml` works without a long-lived key.

## Integration with HC Vault Transit

Vault transit secrets engine as a SOPS recipient. The transit engine wraps and unwraps the dataKey on Vault's side; SOPS never sees the underlying key material.

```bash
vault secrets enable -path=sops transit
vault write -f sops/keys/firstkey
```

The path config in `.sops.yaml` is `hc_vault_transit_uri: https://vault.example.com:8200/v1/sops/keys/firstkey` under a `creation_rules` entry.

Encrypt.

```bash
vault login -method=oidc
sops -e \
  --hc-vault-transit https://vault.example.com:8200/v1/sops/keys/firstkey \
  config.yaml > config.enc.yaml
```

Required env for SOPS to talk to Vault.

```bash
export VAULT_ADDR=https://vault.example.com:8200
export VAULT_TOKEN='hvs....'
sops -d file.enc.yaml
```

Or use the Vault Agent for token caching.

```bash
export VAULT_AGENT_ADDR=http://127.0.0.1:8100
sops -d file.enc.yaml
```

Role-based access. The Vault token's policy must allow `update` on the transit `encrypt/<key>` path (for encrypt) and `decrypt/<key>` (for decrypt).

A minimal policy snippet has `path "sops/encrypt/firstkey" { capabilities = ["update"] }` and `path "sops/decrypt/firstkey" { capabilities = ["update"] }`.

```bash
vault policy write sops-app sops-policy.hcl
vault token capabilities sops/decrypt/firstkey
```

The transit key version. Vault transit keys support versioning; SOPS encrypts with the current key version, decryption works for any retained version.

```bash
vault read sops/keys/firstkey
vault write sops/keys/firstkey/rotate
```

After rotation, `sops --rotate -i file.enc.yaml` re-encrypts with the new version.

## Integration with Kubernetes — sops-secrets-operator

Decrypt at controller. Install the operator (Isindir/sops-secrets-operator), commit `SopsSecret` CRDs containing encrypted data; the operator decrypts on the cluster and creates standard `Secret` resources.

```bash
helm repo add isindir https://isindir.github.io/charts/
helm install sops-secrets-operator isindir/sops-secrets-operator \
  --set secretsAsFiles[0].name=sops-age \
  --set secretsAsFiles[0].mountPath=/etc/sops-age \
  --set secretsAsFiles[0].secretName=sops-age-key
```

The reconcile loop. The operator watches `SopsSecret` resources, decrypts them using the cluster-stored age/PGP/KMS key, and creates/updates `Secret` resources with the plaintext.

A SopsSecret CRD has `apiVersion: isindir.github.com/v1alpha3`, `kind: SopsSecret`, and a `spec.secretTemplates[]` array where each template defines a `name` and `data` map. The `data` values are encrypted leaves; the operator decrypts and produces a `Secret` with the same name in the same namespace.

```bash
kubectl apply -f mysecret.enc.yaml
kubectl get secrets
```

The CRD schema. `secretTemplates` is a list of named templates; each becomes a `Secret` resource. Encrypted leaves under `data` and `stringData` are decrypted by the operator.

```bash
kubectl logs -n sops-secrets-operator deployment/sops-secrets-operator
```

Watch the operator logs for decryption events and errors.

## Integration with Kubernetes — kustomize-sops

Kustomize generator plugin. KSOPS (`viaduct-ai/kustomize-sops`) is a kustomize plugin that decrypts SOPS-encrypted secrets at `kustomize build` time.

```bash
KSOPS_VERSION=4.3.2
curl -L "https://github.com/viaduct-ai/kustomize-sops/releases/download/v${KSOPS_VERSION}/ksops_${KSOPS_VERSION}_Linux_x86_64.tar.gz" | tar xz
mkdir -p ~/.config/kustomize/plugin/viaduct.ai/v1/ksops
mv ksops ~/.config/kustomize/plugin/viaduct.ai/v1/ksops/
```

The manifest flow. Encrypted secrets stay encrypted-at-rest in git; `kustomize build --enable-alpha-plugins` decrypts on-the-fly at apply time.

A kustomization.yaml lists generators including a `secret-generator.yaml`; that file has `apiVersion: viaduct.ai/v1`, `kind: ksops`, and a `files:` array pointing at `.enc.yaml` files.

```bash
kustomize build --enable-alpha-plugins . | kubectl apply -f -
```

Encrypted-at-rest manifests. The git repo only contains `secrets.enc.yaml`; plaintext only exists in cluster memory and on the apply-side host's stdout briefly.

```bash
kustomize build --enable-alpha-plugins ./prod
kustomize build --enable-alpha-plugins ./prod | grep -v ENC
```

The second command verifies that no `ENC[...]` strings leak into the rendered output (sanity check for properly-decrypted manifests).

## Integration with Helm — helm-secrets plugin

Encrypt `values.yaml`. The helm-secrets plugin (`jkroepke/helm-secrets`) wraps SOPS for Helm.

```bash
helm plugin install https://github.com/jkroepke/helm-secrets
helm secrets encrypt values.yaml > secrets.yaml
```

Install with `secrets://` URI scheme.

```bash
helm secrets install my-release ./chart \
  --values secrets://secrets.yaml
```

helm-secrets decrypts secrets.yaml on the fly, then passes to helm.

The wrapping behavior. Internally helm-secrets calls `sops -d` on the file, captures the plaintext, hands it to helm via a temp file (FIFO when supported), and removes the temp on exit.

```bash
helm secrets upgrade my-release ./chart -f secrets://prod-values.yaml
helm secrets template ./chart -f secrets://values.yaml
helm secrets diff upgrade my-release ./chart -f secrets://values.yaml
```

`HELM_SECRETS_DRIVER`. Defaults to `sops`; can be set to `vals` for an alternative.

```bash
export HELM_SECRETS_DRIVER=sops
helm secrets view secrets.yaml
helm secrets edit secrets.yaml
```

The `view` command decrypts to stdout; `edit` opens `$EDITOR` like `sops` does.

## Integration with Terraform

The `carlpett/sops` provider data source.

```bash
terraform init
terraform plan
```

A typical main.tf has a `terraform.required_providers` block declaring `sops = { source = "carlpett/sops", version = "~> 1.0" }`, then a `data "sops_file" "secrets"` block with `source_file = "secrets.enc.yaml"`. Resources reference `data.sops_file.secrets.data["database.password"]`.

The data source. `data.sops_file.NAME.data` is a map keyed by dotted-path of the YAML/JSON; values are the decrypted strings.

```bash
terraform console
```

In the console, `data.sops_file.secrets.data["api.token"]` returns the decrypted string.

For nested paths, use dotted access like `data.sops_file.secrets.data["api.token"]` or `data.sops_file.secrets.data["database.password"]`. For raw file content (binary), use `data.sops_file.cert.raw`.

Auth flow. The provider uses the same SOPS env vars: `SOPS_AGE_KEY_FILE`, `AWS_PROFILE`, etc. Plaintext lives in Terraform state — encrypt the state too (`backend "s3"` with KMS).

```bash
terraform state pull | jq '.resources[] | select(.type=="sops_file")'
```

The state file contains the decrypted values; protect it with state encryption and minimal access.

## Integration with NixOS — sops-nix

`Mic92/sops-nix` decrypts SOPS files at NixOS activation, dropping plaintext at known runtime paths owned by `root` or specific users.

```bash
nix flake update sops-nix
nixos-rebuild switch
```

A NixOS configuration imports `sops-nix.nixosModules.sops` and declares `sops.defaultSopsFile = ./secrets.enc.yaml;`, `sops.age.keyFile = "/var/lib/sops-nix/age-key.txt";`, and a `sops.secrets."path/in/file"` entry per secret.

The secret declaration. Each `sops.secrets."path/in/file"` becomes a file at `/run/secrets/<path>` (or a configurable mount).

Runtime decryption via systemd. sops-nix generates a systemd service unit (`sops-install-secrets.service`) that runs at boot before any service that depends on `/run/secrets/*`. The decryption happens once per boot; no plaintext in the nix store.

The activation hook. On `nixos-rebuild switch`, the activation script runs sops-install-secrets, which decrypts and writes to `/run/secrets/` with the configured ownership.

```bash
ls -la /run/secrets/
sudo cat /run/secrets/database/password
systemctl status sops-install-secrets
```

Per-key options control owner, group, mode, custom path, and `restartUnits` (services to restart when the secret changes after a `nixos-rebuild`).

```bash
nixos-rebuild switch --flake .#my-host
journalctl -u sops-install-secrets
```

Bootstrap key generation on a fresh NixOS host.

```bash
nix-shell -p age --run 'age-keygen -o /var/lib/sops-nix/age-key.txt'
chmod 600 /var/lib/sops-nix/age-key.txt
age-keygen -y < /var/lib/sops-nix/age-key.txt
```

The output public key goes into your repo's `.sops.yaml` as a recipient; run `sops updatekeys -y` on the encrypted files; the next `nixos-rebuild switch` will succeed.

## Integration with Flux CD

`decryption.provider sops` field on Kustomization. Flux's source-controller watches a git repo; the kustomize-controller renders manifests, and if `decryption` is set, calls SOPS on encrypted resources.

A Kustomization CR has `apiVersion: kustomize.toolkit.fluxcd.io/v1`, `spec.path`, `spec.sourceRef`, and `spec.decryption: { provider: sops, secretRef: { name: sops-age } }` to decrypt during reconciliation.

```bash
kubectl apply -f flux-system/kustomization.yaml
flux get kustomizations
```

The `sa.spec.decryption.secretRef` pattern. Create a Secret in the `flux-system` namespace with the age private key.

```bash
cat ~/.config/sops/age/keys.txt | \
  kubectl create secret generic sops-age \
  --namespace=flux-system \
  --from-file=age.agekey=/dev/stdin
```

Flux's kustomize-controller mounts this secret at `/var/run/age` and uses it to decrypt SOPS-encrypted resources during reconcile.

```bash
kubectl logs -n flux-system deployment/kustomize-controller -f
flux reconcile kustomization prod
```

Trigger a reconcile and watch the controller logs to confirm decryption succeeds.

## Integration with ArgoCD

The kustomize-sops plugin path. ArgoCD doesn't ship SOPS support; install KSOPS in the argocd-repo-server pod, configure it as a kustomize plugin.

The argocd-cm ConfigMap needs `data.kustomize.buildOptions: --enable-alpha-plugins` and a `configManagementPlugins` block declaring a `name: ksops` plugin.

```bash
kubectl edit configmap argocd-cm -n argocd
kubectl rollout restart deployment argocd-repo-server -n argocd
```

The repo-server needs the KSOPS binary plus the SOPS decryption key (mounted as a Secret).

The argocd-repo-server Deployment mounts `sops-age` secret at `/etc/sops/age` and sets `SOPS_AGE_KEY_FILE=/etc/sops/age/keys.txt` in its env.

```bash
kubectl apply -f argocd-repo-server-patch.yaml
kubectl rollout status deployment argocd-repo-server -n argocd
```

Verify the plugin works by syncing an Application that includes a SOPS-encrypted secret.

```bash
argocd app sync my-app
argocd app get my-app
```

## CI/CD Patterns

Decrypt-on-the-fly in CI. The single hard rule: never persist decrypted files. Decrypt to env or in-memory FIFO, run the consumer, exit.

A typical GitHub Actions step has `env: SOPS_AGE_KEY: ${{ secrets.SOPS_AGE_KEY }}` and `run: sops exec-env secrets.enc.yaml './deploy.sh'`.

```bash
sops exec-env secrets.enc.yaml ./deploy.sh
```

The avoid-storing-decrypted-files rule.

```bash
sops exec-env secrets.enc.yaml ./deploy.sh
sops exec-file secrets.enc.yaml './deploy.sh --config {}'
```

The wrong path: `sops -d secrets.enc.yaml > secrets.yaml; ./deploy.sh; rm secrets.yaml` — too late if deploy.sh fails before rm.

GitHub Actions secret-as-AGE-KEY pattern.

A workflow has `env: SOPS_AGE_KEY: ${{ secrets.SOPS_AGE_KEY }}` at the job or step level, then runs SOPS commands. The repository secret SOPS_AGE_KEY contains the literal contents of `keys.txt`.

```bash
sops -d secrets.enc.yaml | kubectl apply -f -
```

Runner has age key, decrypts, never persists.

```bash
export SOPS_AGE_KEY="$CI_AGE_KEY_SECRET"
sops exec-env secrets.enc.yaml ./run-tests.sh
unset SOPS_AGE_KEY
```

The pattern in plain bash.

CI service-principal patterns.

```bash
aws sts get-caller-identity
sops -d secrets.enc.yaml > /dev/null
```

For AWS, configure OIDC trust between the CI provider (GitHub, GitLab, CircleCI) and an IAM role with `kms:Decrypt` permission. Use STS-issued creds for the SOPS call; they last only the workflow's lifetime.

For Azure, use federated identity (Workload Identity) — no client secrets in CI.

For GCP, use Workload Identity Federation — OIDC token exchange for service-account impersonation.

For Vault, use the JWT auth method against Vault, mapped to a policy with transit decrypt.

```bash
vault write auth/jwt/login role=sops-ci jwt="$ACTIONS_ID_TOKEN_REQUEST_TOKEN"
sops -d secrets.enc.yaml
```

Pre-commit hooks for safety.

```bash
pre-commit install
pre-commit run --all-files
```

A `.pre-commit-config.yaml` referencing `k8s-at-home/sops-pre-commit` adds a `forbid-secrets` hook that catches accidental commits of unencrypted files matching common secret patterns.

GitLab CI example.

```bash
sops --version
sops -d secrets.enc.yaml > /dev/null
```

In `.gitlab-ci.yml`, mask the `SOPS_AGE_KEY` variable, set it as a CI/CD variable in the project settings, and reference it as `$SOPS_AGE_KEY` in the job.

## Common Errors

`Failed to get the data key required to decrypt the SOPS file`

You have no key available that matches any recipient in the file's metadata.

```bash
sops -d --extract '["sops"]' file.enc.yaml 2>&1 | head -30
gpg --list-secret-keys --keyid-format LONG
ls ~/.config/sops/age/keys.txt
echo $AWS_PROFILE
```

Cross-check the file's recipients with locally available keys (gpg keyring, age key file, AWS profile credentials).

`config file not found, or has no creation rules, and no keys provided`

You're encrypting (`sops -e`) but `.sops.yaml` is missing or has no rule for this path AND no `--age`/`--pgp`/`--kms` flag was given.

```bash
ls -la .sops.yaml
sops -e --age age1A... file.yaml > file.enc.yaml
```

Either create `.sops.yaml` with a rule that matches the path, or specify recipients on CLI.

`could not load PGP key: gpg: decryption failed: No secret key`

The PGP fingerprint listed in the file isn't in your gpg keyring, or your subkey expired.

```bash
gpg --list-secret-keys --keyid-format LONG
gpg --import < private-key.asc
gpg --list-keys --with-subkey-fingerprint 85D77543B3D624B63CEA9E6DBC17301B491B3F21
```

Import the missing key, or extend the expiration.

`Failed to verify Message Authentication Code: MAC mismatch`

The file was tampered with after encryption, or someone edited a value outside SOPS, or the YAML formatting changed (e.g., re-indented) and the MAC is computed over canonical-formatted content.

```bash
git log --oneline -5 file.enc.yaml
git diff HEAD~1 file.enc.yaml
git checkout HEAD~1 -- file.enc.yaml
```

Recover by reverting to the last known-good commit.

`Error: cannot decrypt sops file: no key in keyring matches`

PGP-specific variant of "no key available".

```bash
gpg --list-secret-keys --keyid-format LONG
sops -d --extract '["sops"]["pgp"]' file.enc.yaml
```

Check gpg keyring vs file recipients.

`no creation rule for filename`

`.sops.yaml` exists but no `path_regex` matches the filename.

```bash
ls -la .sops.yaml
sops -e --age age1A... file.yaml > file.enc.yaml
```

Add a catch-all rule or pass `--age`/`--pgp` on CLI.

`metadata not found`

The file has no `sops:` metadata block. Either it's not a SOPS-encrypted file, or it's truncated.

```bash
tail -30 file.enc.yaml
grep -c '^sops:' file.enc.yaml
```

Should show a `sops:` block. If missing, file is plaintext or corrupt.

`ages.age decryption failed: no identity matched any of the recipients`

age-specific: the age private keys in `keys.txt` don't match any of the file's age public-key recipients.

```bash
sops -d --extract '["sops"]["age"]' file.enc.yaml
age-keygen -y < ~/.config/sops/age/keys.txt
```

Compare the file's age recipients with public keys derived from your private keys.

`Error getting data key: 0 successful groups required, got 0`

For Shamir threshold files: zero key groups returned a usable dataKey. Below the threshold; file undecryptable with your current keys.

```bash
sops -d --extract '["sops"]["key_groups"]' file.enc.yaml
sops -d --extract '["sops"]["shamir_threshold"]' file.enc.yaml
```

Verify you have the right keys for at least one group.

`Could not decrypt: encrypted_regex requires at least one match`

You set `encrypted_regex` (in CLI or `.sops.yaml`) but no key in the file matches the regex — there's nothing to encrypt.

```bash
grep -E '^[a-z_]+:' file.yaml
sops -e --encrypted-regex '^(password|secret|key|token)' file.yaml > file.enc.yaml
```

Adjust the regex to match actual key patterns.

`error connecting to vault: 403`

HC Vault Transit auth failed. `VAULT_TOKEN` invalid or lacks transit decrypt capability.

```bash
vault token lookup
vault token capabilities sops/decrypt/firstkey
```

Should include `update`. If not, get a new token or fix the policy.

`Could not generate data key: error encrypting with KMS`

KMS encrypt failed — usually IAM, region mismatch, or KMS key disabled.

```bash
aws kms describe-key --key-id arn:aws:kms:us-east-1:111:key/abc
aws sts get-caller-identity
```

Verify the key is enabled and your caller has `kms:Encrypt` and `kms:GenerateDataKey`.

`File hash mismatch`

The file changed between SOPS reading the metadata and reading the data — typically a concurrent edit. Retry the operation.

```bash
sops -d file.enc.yaml
```

`No such file or directory: '/path/to/file'`

Self-explanatory; the path is wrong or the file doesn't exist.

```bash
ls -la /path/to/file
realpath /path/to/file
```

## Common Gotchas

**Editing in non-sops editor → encrypted file corrupted.**

```bash
sops file.enc.yaml
git checkout file.enc.yaml
```

The wrong path: `vim file.enc.yaml` opens raw ciphertext, edits ENC[...] strings, and on save the file is unrecoverable; MAC mismatch on next `sops -d`. The right path: always use `sops file.enc.yaml`. If you accidentally vim'd an encrypted file, restore from git: `git checkout file.enc.yaml`. Without git history, the file is lost — there's no SOPS recovery from a hand-edited ciphertext.

**Forgetting `sops updatekeys` after rotating recipients in .sops.yaml.**

```bash
find . -name '*.enc.yaml' -exec sops updatekeys -y {} \;
git add .sops.yaml '*.enc.yaml'
git commit -m 'rotate recipients and update keys'
```

The wrong flow: edit `.sops.yaml`, commit, push — new team member tries to decrypt and fails because their key isn't in the existing files. The right flow: edit `.sops.yaml`, run `updatekeys` over every encrypted file, commit BOTH the `.sops.yaml` change AND the metadata diffs, push.

**`--extract` with JSON output instead of bash assignment.**

```bash
TOKEN=$(sops -d --extract '["api"]["token"]' file.enc.yaml)
echo "$TOKEN"
```

The wrong path: `sops -d --output-type json --extract '["api"]["token"]' file.enc.yaml` returns `"secret-value-with-quotes"` (JSON-encoded), and bash captures the quoted form. The right path: omit `--output-type json` so the leaf is returned as a raw string.

**Path-based recipient mismatch (creation_rules first-match).**

```bash
sops --config .sops.yaml -e prod/secrets.yaml > prod/secrets.enc.yaml
```

The wrong path: a catch-all `path_regex: .*` rule placed BEFORE a specific `prod/.*` rule means the catch-all matches first, and the specific rule never fires. The right path: order rules most-specific first, catch-all last.

**Comments in YAML changing whitespace → MAC mismatch on next edit.**

```bash
sops file.enc.yaml
```

Adding/removing leading whitespace on a comment line can shift the canonical-form MAC input. SOPS 3.8+ is more robust here, but older versions trip. The fix: use sops' built-in editor, never hand-edit indentation on encrypted files.

**Binary mode interpreting text as binary → unicode corruption.**

```bash
sops -e config.yaml > config.enc.yaml
sops -e --input-type binary opaque-blob.bin > opaque-blob.enc
```

The wrong path: `sops -e --input-type binary --output-type yaml utf8-config.txt > out.yaml` — bytes are preserved through the round trip, BUT you've lost the ability to do leaf-value diffs and you've added unnecessary base64 overhead. The right path: use `--input-type binary` only for genuinely opaque blobs (keystores, .pem with `-----BEGIN`, jks, image files).

**Decryption-only key vs encrypt+decrypt key permissions.**

```bash
sops -d file.enc.yaml
sops file.enc.yaml
sops -e new-file.yaml > new-file.enc.yaml
sops --rotate -i file.enc.yaml
```

An age private key always implies the public key (encrypt+decrypt). But for KMS, IAM policies can grant `kms:Decrypt` without `kms:Encrypt` or `kms:GenerateDataKey`. A user with decrypt-only KMS perms can run `sops -d` and edit existing files (re-encrypt uses existing dataKey), but `sops -e` on a new file fails because it needs `GenerateDataKey`. Likewise `sops --rotate` fails. Plan IAM to match expected operations.

**Forgetting to add new key to old encrypted files.**

```bash
find . -name '*.enc.yaml' -exec sops updatekeys -y {} \;
```

The wrong flow: add the new recipient to `.sops.yaml`, encrypt new files (which include the new recipient), but old files still only have old recipients. New team member can decrypt new files but not old ones. The right flow: always run `updatekeys` after `.sops.yaml` changes.

**Mixing encrypted_regex and encrypted_suffix in the same file.**

```bash
cat .sops.yaml
```

The wrong path: both `encrypted_regex` and `encrypted_suffix` set on the same rule — behaviour is undefined / regex wins. The right path: pick ONE mechanism per `.sops.yaml` rule and document the choice in a comment.

**Pre-commit hook that decrypts to validate — leaks plaintext.**

```bash
sops exec-file secrets.enc.yaml 'yamale -s schema.yaml {}'
```

The wrong path: a hook that runs `sops -d secrets.enc.yaml > /tmp/check.yaml; yamale -s schema.yaml /tmp/check.yaml; rm /tmp/check.yaml` — too late if the hook is killed mid-run. The right path: validate the encrypted form's structure (key names, types of `ENC[...]` strings) without decrypting, OR use `sops exec-file` with a FIFO so plaintext is never on disk.

**Forgetting to set EDITOR before `sops file.enc.yaml`.**

```bash
echo 'export EDITOR=vim' >> ~/.bashrc
export EDITOR=vim
sops file.enc.yaml
```

The wrong path: `$EDITOR` unset, SOPS falls back to `vi` (which may not exist on minimal containers), `sh: command not found`, file looks fine on disk because nothing wrote. The right path: set `EDITOR` globally in your shell profile.

**Committing the unencrypted intermediate file.**

```bash
echo 'secrets.yaml' >> .gitignore
echo '*.dec.yaml' >> .gitignore
echo '!*.enc.yaml' >> .gitignore
git add .gitignore
```

The wrong flow: `cp template.yaml secrets.yaml; fill in values; sops -e secrets.yaml > secrets.enc.yaml; git add .` — the unencrypted `secrets.yaml` is staged via `git add .`, then pushed with plaintext in the repo. The right flow: add to `.gitignore` so `git add` never picks up the plaintext intermediate.

**Forgetting AWS_REGION for KMS calls.**

```bash
export AWS_REGION=us-east-1
sops -d file.enc.yaml
```

If `AWS_REGION` is unset and the SOPS metadata's KMS ARN encodes a region, SOPS uses that region. But if the SDK chain has no region (no env, no profile region), KMS calls fail with `MissingRegion`. Always set `AWS_REGION` or `AWS_DEFAULT_REGION` for explicit control.

**Confusing `--encrypted-regex` and `--unencrypted-regex`.**

```bash
sops -e --encrypted-regex '^(password|secret)' file.yaml > file.enc.yaml
sops -e --unencrypted-regex '^(public|hostname)' file.yaml > file.enc.yaml
```

The first encrypts ONLY keys matching the regex; the second encrypts EVERYTHING EXCEPT keys matching the regex. Reading them backwards leads to either too much or too little encryption.

**Editing a SopsSecret CRD in kubectl.**

```bash
sops mysecret.enc.yaml
git add mysecret.enc.yaml
git commit -m 'rotate secret'
git push
```

The wrong path: `kubectl edit sopssecret mysecret -n default` — kubectl shows the encrypted YAML and lets you "edit" it, but on save the cluster sees an unchanged ciphertext (you can't edit the encrypted form usefully) or an invalid one (you tampered the ENC string). The right path: edit the source file with `sops`, commit to git, let GitOps roll it forward.

## Idioms

One-line decrypt to env.

```bash
sops exec-env config.enc.yaml 'echo $DATABASE_URL'
```

SOPS + git pre-commit hook (block plaintext commits). A `.git/hooks/pre-commit` script that loops over staged files matching `\.env|secrets|\.sops`, checks each for `ENC\[` or `^sops:`, and refuses commit if neither is present.

```bash
git diff --cached --name-only | xargs -I{} grep -l 'ENC\[' {}
```

Decrypt-and-pipe to kubectl.

```bash
sops -d secret.enc.yaml | kubectl apply -f -
```

SOPS in dev shell (nix devShell). A `flake.nix` `devShells.default` block lists `packages = with pkgs; [ sops age ssh-to-age ]` and a `shellHook` that exports `SOPS_AGE_KEY_FILE=$PWD/.age-key`.

```bash
nix develop
sops -d secrets.enc.yaml
```

Decrypt all secrets to env for a long-running session.

```bash
eval "$(sops -d --output-type dotenv .env.enc | sed 's/^/export /')"
```

All vars are now exported in the current shell.

SOPS-aware diff in git. A `.gitconfig` `[diff "sopsdiffer"]` with `textconv = sops -d` plus `.gitattributes` line `*.enc.yaml diff=sopsdiffer` makes `git diff` show plaintext diffs of encrypted files (assuming you have the keys locally).

```bash
git config --global diff.sopsdiffer.textconv 'sops -d'
echo '*.enc.yaml diff=sopsdiffer' >> .gitattributes
git diff secrets.enc.yaml
```

Wrap any command with secrets.

```bash
alias withsecrets='sops exec-env secrets.enc.yaml'
withsecrets ./run-app.sh
withsecrets terraform plan
```

Multi-file decrypt for local dev.

```bash
for f in api.enc.yaml db.enc.yaml; do
  eval "$(sops -d --output-type dotenv "$f" | sed 's/^/export /')"
done
```

Compare encrypted files across branches with sops-aware diff.

```bash
git diff main..feature -- secrets.enc.yaml
```

If you've configured the sopsdiffer textconv, this shows plaintext.

Decrypt for a single command, then unset.

```bash
SOPS_AGE_KEY="$AGE_KEY" sops exec-env secrets.enc.yaml './deploy.sh'
```

The age key is in env only for the SOPS process; it doesn't persist beyond the line.

Bulk add a recipient.

```bash
find . -name '*.enc.yaml' -exec sops --add-age age1NEW... -i {} \;
```

Useful when onboarding a new team member with an age key.

Rotate then update keys in a single git commit.

```bash
find . -name '*.enc.yaml' -exec sops --rotate -i {} \;
find . -name '*.enc.yaml' -exec sops updatekeys -y {} \;
git add '*.enc.yaml'
git commit -m 'quarterly rotation: new dataKey + updated recipient list'
```

Standard pattern for quarterly rotation.

## Migration

From gpg-encrypted files. Decrypt with gpg, re-encrypt with sops.

```bash
gpg --decrypt secrets.yaml.gpg > secrets.yaml.tmp
sops -e secrets.yaml.tmp > secrets.enc.yaml
shred -u secrets.yaml.tmp
rm secrets.yaml.gpg
```

The `shred -u` overwrites the file with random data before unlinking, mitigating filesystem recovery.

From raw KMS encryption (per-value envelope).

```bash
aws kms decrypt --ciphertext-blob fileb://secret.bin --output text --query Plaintext | base64 -d > plain.yaml
sops -e --kms arn:aws:kms:us-east-1:111:key/abc plain.yaml > config.enc.yaml
shred -u plain.yaml
```

Decrypt all values into a plain YAML file, then encrypt the whole file with sops using the same KMS key.

From ansible-vault.

```bash
ansible-vault decrypt vars.yml
sops -e vars.yml > vars.enc.yml
shred -u vars.yml
```

Then update playbooks to use the `community.sops.sops` lookup or pre-tasks to load decrypted vars.

```bash
ansible-playbook -e @<(sops -d vars.enc.yml) site.yml
```

The process substitution `<(...)` writes the decrypted content to a FIFO; ansible reads it as if it were a file; no plaintext on disk.

The rotate-then-encrypt-with-new-recipient pattern. Phased migration where old and new recipients coexist.

```bash
sops --add-age age1NEW... -i file.enc.yaml
SOPS_AGE_KEY_FILE=~/new-keys.txt sops -d file.enc.yaml > /dev/null
sops --rotate --rm-age age1OLD... -i file.enc.yaml
```

Phase 1: add new recipient alongside old. Phase 2: verify new recipient works (run as new user). Phase 3: remove old recipient, rotate dataKey. Now old recipient cannot decrypt new ciphertexts.

Bulk migration script.

```bash
mkdir -p secrets.enc
for f in secrets/*.yml; do
  sops -e "$f" > "secrets.enc/$(basename "$f" .yml).enc.yml"
done
```

Migrates every `.yml` under `secrets/` to `.enc.yaml` under `secrets.enc/`.

From SOPS+PGP to SOPS+age (the modern path).

```bash
age-keygen -o ~/.config/sops/age/keys.txt
find . -name '*.enc.yaml' -exec sops updatekeys -y {} \;
find . -name '*.enc.yaml' -exec sops --rotate -i {} \;
```

Generate age keys for everyone, edit `.sops.yaml` to add age recipients alongside PGP, re-wrap all files. After verification window, remove PGP from `.sops.yaml`, re-wrap again. Optionally rotate dataKeys if you suspect any PGP key leaked.

From git-crypt.

```bash
git-crypt unlock
sops -e -i secrets.yaml
git rm --cached secrets.yaml
git add secrets.enc.yaml
git commit -m 'migrate from git-crypt to sops'
```

Then remove git-crypt from `.gitattributes` and the repository.

From SealedSecrets (Bitnami).

```bash
kubectl get sealedsecret mysecret -o yaml > mysecret.sealed.yaml
kubectl get secret mysecret -o yaml > mysecret.plain.yaml
sops -e mysecret.plain.yaml > mysecret.enc.yaml
shred -u mysecret.plain.yaml
```

Then decommission the SealedSecret resource and adopt one of the SOPS-on-K8s patterns (operator, kustomize-sops, Flux, ArgoCD).

## See Also

- age — modern, simple file encryption used as a SOPS recipient.
- gpg — classic PGP-based file encryption, also a SOPS recipient.
- vault — HashiCorp Vault, the transit engine integrates as a SOPS recipient and Vault itself is an alternative for runtime secrets.
- openssl — for raw symmetric/asymmetric crypto outside of structured-file workflows.
- ssh — public-key auth and ssh-agent; ssh-to-age converts ssh-ed25519 keys to age recipients.

## References

- github.com/getsops/sops — active fork, source, releases.
- github.com/getsops/sops/blob/main/README.rst — canonical user guide and CLI reference.
- age-encryption.org — age spec and tooling.
- vaultproject.io/docs/secrets/transit — HC Vault transit engine docs.
- docs.aws.amazon.com/kms — AWS KMS reference.
- cloud.google.com/kms/docs — GCP KMS reference.
- learn.microsoft.com/azure/key-vault — Azure Key Vault reference.
- github.com/Mic92/sops-nix — NixOS integration.
- github.com/jkroepke/helm-secrets — Helm plugin.
- github.com/viaduct-ai/kustomize-sops — KSOPS for Kustomize/Argo.
- github.com/carlpett/terraform-provider-sops — Terraform provider.
- fluxcd.io/flux/guides/mozilla-sops — Flux CD integration.
- github.com/isindir/sops-secrets-operator — Kubernetes operator.
