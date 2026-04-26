# Mozilla SOPS internals

A deep dive into `sops` (Secrets OPerationS) — Mozilla's editor-of-encrypted-files. The why behind selectively encrypting just the values in a structured config.

## Setup

`sops` solves a specific problem: storing secrets *in source control*, alongside the rest of your config, in a form that's diff-able, mergeable, and readable enough for humans, while still encrypting the sensitive bits.

The classical approaches don't fit:

- **Whole-file encryption** (`gpg`, `age`, `ansible-vault`): a single byte change produces an entirely different ciphertext. Diffs are useless. Reviewing PRs requires fully decrypting both versions and diffing the plaintexts. Merge conflicts are catastrophic.
- **External secret store** (Vault, AWS Secrets Manager, GCP Secret Manager): great, but now you need a network round-trip at runtime, you need to provision and authenticate to the store, and there's no obvious way to share dev secrets with a colleague without giving them store access.
- **Plaintext + .gitignore**: hopefully obvious why this is bad.

`sops` takes a middle path: the file's *structure* is plaintext, the *values* are encrypted individually. A YAML config like:

```yaml
db:
  host: db.example.com
  password: ENC[AES256_GCM,data:abc...,iv:def...,tag:ghi...,type:str]
api_key: ENC[AES256_GCM,data:jkl...,...]
```

Is exactly what's checked into git. The structure is visible. The keys are visible. The values that are "secret-like" (per a configurable filter) are individually encrypted, individually IV-randomised. Diffs show "this value changed" without revealing the new value. Reviewers can see structure changes without decrypting.

The role: file-level encryption with selective values. The supported formats:

- **YAML** — full support, including comments preservation, ordering preservation.
- **JSON** — full support, but no comments (JSON has none) and key ordering is per-platform (Go's encoding/json sorts).
- **TOML** — full support.
- **INI** — full support.
- **dotenv** (`.env`) — full support; line-oriented, no nesting.
- **binary** — opaque; the entire file is base64-armored and encrypted as one big value.

The crypto is hybrid: a per-file random *data key* encrypts the values; the data key is then wrapped under one or more recipient keys (KMS, PGP, age). This is the same pattern as age, gopass, and PGP itself — the "key encryption key" / "data encryption key" two-tier scheme.

The Mozilla provenance: sops was originally a Mozilla SecOps tool for managing Firefox release secrets. It was open-sourced and has since become the de facto standard for git-stored encrypted config in the Kubernetes/cloud-native ecosystem (Flux, ArgoCD, Helm-secrets, sops-nix all consume sops files). After Mozilla wound down the team, the project moved to GetSops/CNCF.

## Threshold Cryptography

The `key_groups` feature uses Shamir Secret Sharing for threshold decryption. The configuration:

```yaml
key_groups:
  - kms:
      - arn: arn:aws:kms:us-east-1:111111:key/aaa
    pgp:
      - "FINGERPRINT_A"
  - kms:
      - arn: arn:aws:kms:us-east-1:111111:key/bbb
    pgp:
      - "FINGERPRINT_B"
shamir_threshold: 2
```

This says: the data key is split into N shares (one per group); to decrypt, you need K shares (the `shamir_threshold`). Each share, in turn, is encrypted to *every* key in its group (so any member of group A can recover share A; any member of group B can recover share B; you need both groups to reconstruct the data key).

Shamir's scheme: a polynomial of degree K-1 over GF(2^8) is constructed with the data key as the constant term. N points on the polynomial are computed (one per share). Any K points uniquely determine the polynomial (Lagrange interpolation), so any K shares reconstruct the data key. Fewer than K shares give *zero* information about the data key (information-theoretic security, not just computational).

The use cases:

- **Two-person rule**: encrypt to two key groups, threshold 2. Neither person alone can decrypt; both must agree.
- **M-of-N quorum**: 5 ops engineers each have a key group; 3 of them together can decrypt. Resilient to one or two engineers being unavailable.
- **Region failover**: encrypt to KMS keys in 3 regions, threshold 2. Any 2-of-3 regions available is enough.

The threshold is configured at file creation. Changing it later requires re-encryption (`sops updatekeys` doesn't re-thresh; you must `sops -r` rotate).

The downside: Shamir is *stateful* in the sense that share assignment matters. Adding a new key group with `sops updatekeys` doesn't redo the share assignment. So practically, threshold cryptography is most useful when the topology is set once and remains stable.

## Data Key Envelope

The encryption flow for a single sops file:

1. **Generate** a random 32-byte AES key (the data key, `DK`).
2. For each top-level *recipient* (KMS, PGP, age, etc.):
   - Wrap `DK` under the recipient's key.
   - Store the wrapped `DK` and recipient metadata in the file's `sops` block.
3. For each "encryptable" value in the file (per regex, suffix, or default rules):
   - Generate a random 96-bit IV.
   - Encrypt the value with AES-GCM-256 using `DK` and the IV.
   - The 128-bit GCM tag goes into the same envelope.
   - The full envelope is a string: `ENC[AES256_GCM,data:base64(ciphertext),iv:base64(iv),tag:base64(tag),type:str|int|float|bool|bytes]`.
4. **MAC** the entire (encrypted) plaintext with HMAC-SHA-512 using a key derived from `DK`.
5. Write the file.

Decryption:

1. Read the `sops` block.
2. For each recipient: try to unwrap `DK`. If any succeeds, you have the data key.
3. For each encrypted value: parse the envelope, decrypt with `DK` and the value's IV, verify the GCM tag.
4. Recompute the HMAC-SHA-512 over the decrypted plaintext, compare with stored MAC.

The per-value AES-GCM is a fresh AEAD per value. Each gets its own IV (96 bits, random — collision-resistant for billions of values). The GCM tag authenticates each value individually.

The `type` field in the envelope is necessary because YAML and JSON have typed values (string vs int vs bool vs binary). When sops decrypts, it has to put the right type back. Without the type field, `password: 123` (the integer 123) would round-trip as the string `"123"` after encryption.

## The MAC

The `.sops.mac` field is an HMAC-SHA-512 over the *plaintext values* of every encrypted field, in canonical order. The HMAC key is derived:

```
mac_key = HMAC-SHA-512(DK, "sops" || lastmodified)
```

(The exact derivation has evolved across sops versions; check the source for current formula.)

The MAC catches:

- **Truncation** of the file.
- **Reordering** of encrypted values across keys (since the canonical order depends on key paths).
- **Substitution** of one encrypted value's ciphertext+IV+tag into another field's slot.

Without the MAC, an attacker with write access to the file could swap the encrypted values for `db.password` and `api_key`, and AES-GCM alone wouldn't notice (each value's GCM tag still validates against its own ciphertext+IV).

The "canonical-mac" handling is subtle. The plaintexts must be hashed in a deterministic order. For YAML, sops walks the document in document order; for JSON, sorts keys; for INI, line order. The serialisation is also fixed: integers as decimal, floats with full precision, strings as-is.

The `.mac` field is typically near the end of the `sops` block. It's stored hex-encoded.

## Encrypted Regex / Suffix

By default, sops encrypts *every* leaf value. This is rarely what you want — you don't need to encrypt `region: us-east-1` or `replicas: 3`.

Two filter mechanisms:

1. **`encrypted_regex`** — a regex over key names. Only matching keys are encrypted.
2. **`encrypted_suffix`** — only keys ending in this suffix are encrypted.

A typical `.sops.yaml`:

```yaml
creation_rules:
  - path_regex: '\.yaml$'
    encrypted_regex: '^(password|secret|api_key|token|cert|private_key)$'
    pgp: 'FINGERPRINT'
```

The matching is on *key names*, applied at every level of the document. So `db.password` matches because the leaf key `password` matches the regex.

For arrays of objects, each leaf inside each object is checked independently:

```yaml
users:
  - name: alice         # not encrypted (key 'name')
    password: hunter2   # encrypted (key 'password' matches)
  - name: bob
    password: secure!
```

The "leaf-only encryption preserves structure" property is what makes sops ergonomic: you can `kubectl apply -f` a sops-encrypted manifest without decryption (well, almost — you usually have a sops decryption webhook or `helm-secrets` plugin). The structure is visible to git diff, to YAML schema validators, to CI linters that don't need the values.

The `encrypted_suffix` variant is sometimes preferred:

```yaml
encrypted_suffix: '_enc'
```

Then keys ending in `_enc` are encrypted: `password_enc`, `api_key_enc`. This makes the intent obvious in the source file — anyone reading sees "this is meant to be encrypted." The downside: it pollutes key names.

There's also `unencrypted_regex` and `unencrypted_suffix` for the inverse: "encrypt everything except these." Useful when most fields are secret and only a few are not.

## AWS KMS Backend

The KMS backend uses AWS Key Management Service. The config:

```yaml
kms:
  - arn: arn:aws:kms:us-east-1:111111111111:key/abc-123
    role: arn:aws:iam::111111111111:role/sops-decrypt
    aws_profile: prod-profile
```

The IAM permissions required:

- **`kms:Encrypt`** — to wrap the data key (encryption-time).
- **`kms:Decrypt`** — to unwrap the data key (decryption-time).
- **`kms:GenerateDataKey`** — actually used by sops to ask KMS to generate-and-wrap in one call (more efficient than separate generate + encrypt).
- **`kms:DescribeKey`** — to validate the key exists and is enabled.

The `role` field tells sops to STS-assume that role before calling KMS. This enables cross-account use: the file is encrypted with a key in account A, but the user's credentials are in account B; the role in account A is assumable by B's principal.

Multi-region replicas: AWS supports replicating a KMS key across regions, so the same key (with the same `key-id`) exists in multiple regions. sops can be told to use any of them; failover is automatic.

KMS Grants are an alternative to IAM policies for fine-grained, time-limited access. A grant grants permission to encrypt/decrypt to a specific principal for a specific operation, without modifying the key policy. Useful for ephemeral CI/CD jobs.

KMS encryption context: KMS supports a "context" string that's bound to the encrypted data and must be supplied at decryption. sops uses encryption context to bind the data key to the file's path (or other identifier), so an attacker who steals one wrapped key can't use it to decrypt a different file. This is configured via `encryption_context: {key1: value1}` in the kms block.

The cost model: KMS charges per-call. Each sops encrypt is one `GenerateDataKey` call; each decrypt is one `Decrypt` call. At pennies per 10,000 calls, this is negligible for typical use, but bulk batch jobs that decrypt thousands of files in a tight loop can rack up small costs.

The latency: KMS calls are network round-trips, typically ~50ms to a regional endpoint. Encrypting a sops file is dominated by the KMS call, not the local AES-GCM. For interactive editing, this is fine; for high-frequency CI loops, batch operations or local caching is worth considering.

## Azure Key Vault Backend

Azure Key Vault uses HSM-backed RSA keys for wrapping. The config:

```yaml
azure_kv:
  - vaultUrl: https://my-vault.vault.azure.net
    name: sops-key
    version: abc123
```

Authentication: by default, the Azure SDK chain tries:

1. Environment variables (`AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_SECRET`).
2. Managed Identity (when running on an Azure VM with MSI).
3. Azure CLI (`az login` cached credential).
4. VS Code, IntelliJ IDEs.

For CI/CD, the recommended pattern is a service principal: create an SP in AAD, give it `Key Vault Crypto User` role on the vault, store credentials as CI secrets, sops uses them.

Vault key URL: each key has a versioned URL like `https://my-vault.vault.azure.net/keys/sops-key/abc123`. The version is required for stability — without it, the "current version" might change under you, and old sops files would fail to decrypt.

Soft-delete: enabled by default on Azure Key Vault. A deleted key is recoverable for 7-90 days (configurable). After purge, the key is gone forever, and any sops file encrypted to it is unrecoverable. **Always enable soft-delete and purge protection on production vaults.**

Purge protection: prevents purging deleted keys until soft-delete window expires. This is the difference between "I accidentally deleted, I have 7 days to undo" and "I can't even delete." For sops backends, enable both.

The wrap algorithm: AKV does RSA-OAEP-SHA-256 (for software keys) or RSA-OAEP-SHA-1 (older HSM keys). The wrapped data key is base64-encoded in the sops `azure_kv` block.

## GCP KMS Backend

GCP Cloud KMS works similarly to AWS:

```yaml
gcp_kms:
  - resource_id: projects/my-proj/locations/us/keyRings/sops-ring/cryptoKeys/sops-key
```

The `resource_id` follows the GCP-standard format: `projects/.../locations/.../keyRings/.../cryptoKeys/...`. The `locations/...` is a GCP region (`us-east1`) or a multi-region (`us`, `eu`, `global`).

Authentication: Application Default Credentials (`gcloud auth application-default login`). For CI/CD, use service account keys (downloaded JSON file via `GOOGLE_APPLICATION_CREDENTIALS` env) or Workload Identity (preferred — no static credentials).

IAM: the principal needs `roles/cloudkms.cryptoKeyEncrypterDecrypter` on the key.

GCP KMS uses AES-GCM for symmetric keys (not RSA wrap). The data key is wrapped with `Encrypt`, unwrapped with `Decrypt`. The key material itself is non-extractable (FIPS 140-2 Level 3 if you use HSM keys).

Multi-region keys: GCP's `global` location is a multi-region; the key is replicated across the entire global GCP fleet for highest availability. `us` is multi-region within North America.

Cost model: ~$0.03 per 10k calls. Cheaper than AWS KMS at scale.

## HC Vault Transit Backend

HashiCorp Vault's transit secrets engine is a "encryption-as-a-service" mode: Vault stores keys, your app sends plaintext, Vault returns ciphertext. sops uses this for wrapping the data key.

```yaml
hc_vault:
  - vaultAddress: https://vault.example.com
    enginePath: transit
    keyName: sops-key
```

The engine path: by default `transit`, but you can mount the engine at any path (`transit/team-a`, `transit/prod`).

The key name: a logical key within the engine. Has a current version + historical versions for rotation.

Vault auth: typically token-based (`VAULT_TOKEN` env). For CI/CD, AppRole auth is common (role-id + secret-id, exchanged for a token). For Kubernetes, Vault's K8s auth method binds to service accounts.

The role binding: the Vault role attached to the auth method must have the policy `path "transit/encrypt/sops-key" { capabilities = ["update"] }` and `path "transit/decrypt/sops-key" { capabilities = ["update"] }`.

The wrap is symmetric (Vault uses ChaCha20-Poly1305 or AES-GCM-256 internally). Vault tracks the key version used for each ciphertext, which is what enables Vault-managed rotation: you bump the key version in Vault, old ciphertexts still decrypt under the old version, new ones encrypt under the new version.

Performance: a Vault round-trip is a network call; for sops, this is one call per encryption, one per decryption. Vault is much faster than KMS in raw throughput (~1ms vs ~50ms).

## PGP Backend

The PGP backend uses GnuPG. Recipients are GPG fingerprints:

```yaml
pgp:
  - "ABCDEFABCDEFABCDEF1234567890ABCDEF12345678"
```

The fingerprint is the full 40-hex-char SHA-1 fingerprint of the OpenPGP public key. Short keyids (8 or 16 hex) are deprecated and dangerous (collision-prone).

To list available fingerprints:

```bash
gpg --list-secret-keys --keyid-format LONG
```

But the modern recommendation is `--keyid-format=0xLONG` or `--with-fingerprint` to get the full fingerprint.

gpg-agent integration: sops invokes `gpg --decrypt`, which prompts gpg-agent for the key. If the key is on a smartcard, the smartcard PIN is requested via pinentry. This is the standard GPG flow; sops does not invent its own protocol.

Smartcard pairing: the typical flow is `gpg --card-status` to verify the card is detected, then `gpg --card-edit` to associate with a key. Once paired, `gpg --decrypt` calls trigger the smartcard automatically.

The wrap algorithm depends on the key:

- RSA: RSAES-PKCS1-v1_5 (legacy) or RSAES-OAEP.
- ECC: ECDH-anonymous (Curve25519, P-256, etc.) wrapping AES-128/256-KEYWRAP.

sops doesn't care about the wrap algorithm; it just calls `gpg`.

The downside: PGP is the *most complex* of the backends, with the most foot-guns. Subkey expiration, web-of-trust, key-server pollution — all gotchas that don't apply to KMS/age. The trend, articulated by sops contributors and observable in real codebases, is to migrate from PGP to age.

## age Backend

The age backend uses age recipients:

```yaml
age:
  - recipient: age1abc...
```

For decryption, sops needs the age identity (private key). It looks for it in:

1. `SOPS_AGE_KEY` environment variable (literal key bytes).
2. `SOPS_AGE_KEY_FILE` env var pointing to a file.
3. `~/.config/sops/age/keys.txt` (default location).

The keys.txt is a simple format:

```
# created: 2024-01-01T00:00:00Z
# public key: age1abc...
AGE-SECRET-KEY-1XYZ...
```

Multiple identities can be in one file; age tries each.

The "age replaces PGP" trend: in late-2010s/early-2020s sops deployments, PGP was the dominant local backend (cloud KMS for production, PGP for dev). Around 2021-2023, age started replacing PGP for the local-dev case because:

- No subkey expiration grief.
- Simpler key generation (`age-keygen` vs the PGP wizard).
- Smaller binary footprint.
- Cleaner threat model.

For cloud production, KMS is still preferred (managed access control, audit logging, HSM backing). age is the local-dev companion to KMS.

## updatekeys

`sops updatekeys file.yaml` re-wraps the data key against the *current* `.sops.yaml` recipient list, without rotating the data key itself.

The flow:

1. Decrypt the data key using whichever current recipient the user has access to.
2. Read `.sops.yaml`, determine the new desired recipient list.
3. Re-wrap the same data key under the new recipient list.
4. Replace the `sops` block in the file with the new wrappings.
5. The encrypted *values* and the MAC are untouched.

The use case: adding or removing recipients without rotating the key. New team member joins → add their pubkey to `.sops.yaml`, run `sops updatekeys` on every encrypted file → they can now decrypt.

The result is idempotent: running updatekeys repeatedly produces the same file. This makes it safe to run in CI as a "ensure recipients are current" check.

The git diff after `updatekeys` is small: only the `sops` metadata block changes, not the encrypted values. This is reviewable.

But: a removed recipient who has a clone of the repo from before still has the old wrapped data key, and that wrap was to *their* pubkey. They can still decrypt. To actually remove access, you must rotate (`sops -r`).

## Rotate

`sops -r file.yaml` (or `--rotate`) generates a new data key and re-encrypts everything:

1. Decrypt the file with the old data key (via current recipients).
2. Generate a new random data key.
3. Re-encrypt every value with the new data key (new IVs, new GCM tags).
4. Wrap the new data key under all current recipients.
5. Recompute the MAC.
6. Write the file.

This is a complete data-key rotation. Every encrypted value changes (new IV, new ciphertext, new tag). The git diff is large (every encrypted line changes), which is ugly but unavoidable.

The cadence question: "how often should I rotate?" There's no general answer:

- After a recipient is removed: yes, rotate, otherwise the removed recipient retains decryption ability for git history.
- After a suspected compromise: yes, rotate.
- On a fixed schedule (quarterly, annually): probably yes for high-stakes secrets, debatable for low-stakes.

The "rotation needs rotation" insight: rotating the *data key* doesn't rotate the *secrets themselves*. If the database password leaked, you must change the password at the source (in the database) — re-encrypting the file with a new data key doesn't undo the leak. Rotation is a defence in depth, not a fix.

A common rotation hygiene: combine `sops -r` with a real secret rotation. Change the database password, update the file with the new password, encrypt with a new data key. Two birds.

## exec-env / exec-file

`sops exec-env file.yaml -- ./my-app` decrypts the file's leaf values, exports them as environment variables, and execs `./my-app`. The flow:

1. Decrypt the file (returns a YAML/JSON/etc structure).
2. Flatten the structure into env vars (key path becomes `UPPER_SNAKE_CASE`).
3. Set the env, exec the child process.
4. The child sees the variables; the parent (sops) exits.

`sops exec-file` does the same but writes the decrypted content to a temp file and substitutes the path:

```bash
sops exec-file file.yaml -- /usr/bin/my-app --config={}
```

The `{}` is replaced with the temp file path.

Security trade-offs:

**exec-env**:
- Pros: never writes plaintext to disk; OS-level isolation between processes.
- Cons: env vars are visible to anyone who can read `/proc/PID/environ`. On Linux, this is the same UID by default, but ptrace-capable processes can read other UIDs' environ. The variables are visible in the child's `argv`/env via tools like `ps eww` (depending on platform).

**exec-file**:
- Pros: file path can be passed to apps that only accept files (which is common for config tools like Helm, Kustomize).
- Cons: plaintext on disk, even if briefly. The temp file is created with mode 0600 in `os.TempDir()` (which on Linux is `/tmp` by default — multi-user). On systems with `/dev/shm` (tmpfs), preferring that is safer.

The general guidance: prefer exec-env for short-lived decryptions; prefer exec-file when the consumer requires file input; never write decrypted content to a persistent location.

The `$PROC` env var visibility concern: on most modern Linuxes, `/proc/PID/environ` is mode 0400 owned by the process UID, so only same-UID processes (and root) can read it. This is sufficient if you trust the user account. But containerised environments where multiple processes share a UID inside the container deserve consideration.

## Format-Specific Behavior

YAML preservation: sops uses a YAML library that preserves comments, key ordering, and (where possible) flow style. A diff between two sops-encrypted YAMLs shows clean changes in the right places. Anchors and aliases in YAML are *not* preserved — sops resolves them before encryption.

JSON: less metadata-friendly because JSON has no comments. Key ordering is preserved if the underlying parser is order-preserving (Go's encoding/json sorts; sops uses a custom order-preserving variant).

Binary: the entire file is base64-armored and stored as one big encrypted value. Use this for non-text files (TLS certs, certificate bundles, jpeg files, whatever). The diff is meaningless (a single byte change re-encrypts the entire file under a new IV), but the file is still git-friendly because git doesn't care about content.

INI / dotenv: line-oriented. Each value is encrypted independently. Comments are preserved. Section headers are not encrypted.

The format is auto-detected by file extension (`.yaml`, `.yml`, `.json`, `.toml`, `.ini`, `.env`, `.bin`). Override with `--input-type` and `--output-type`.

## Threat Model

sops protects:

- **Confidentiality of values** — encrypted at rest under AES-GCM-256 with a per-file random data key.
- **Integrity of values** — per-value GCM tag + global HMAC-SHA-512.
- **Recipient list integrity** — the data key is wrapped under each recipient; tampering with the recipient list breaks decryption (or at least is detectable, depending on order).

sops does not protect:

- **Structure** — key names are visible. An attacker reading the file learns "there's a `db.password`."
- **Number of values** — visible.
- **Presence** — the existence of a sops file is observable.
- **Lengths of values** — encrypted values' ciphertext lengths leak rough plaintext lengths (modulo padding, which sops does not do automatically).
- **Recipient-key resistance** — if any recipient's private key is compromised, all sops files encrypted to that recipient are decryptable.

The "recipient is the unit of trust" point is operationally important. A KMS key that's in many sops files is a single point of failure: if that KMS key is misconfigured (allowing too-broad access), every sops file using it is at risk.

## .sops.yaml creation_rules

The `.sops.yaml` config file in your repo defines *which keys to use for which paths*:

```yaml
creation_rules:
  - path_regex: ^secrets/dev/.*\.yaml$
    age: age1dev_recipient...

  - path_regex: ^secrets/staging/.*\.yaml$
    kms: arn:aws:kms:us-east-1:222222:key/staging
    age: age1staging_recipient...

  - path_regex: ^secrets/prod/.*\.yaml$
    kms: arn:aws:kms:us-east-1:333333:key/prod
    pgp: 'PROD_FINGERPRINT'
    encrypted_regex: ^(password|secret|token)$

  - path_regex: '\.yaml$'  # catchall, must be last
    age: age1default...
```

The matching is *first match wins*. The first rule whose `path_regex` matches the file path is used; subsequent rules are ignored. Order matters.

This is the standard "more specific first, catchall last" pattern. The catchall is essential — without it, sops fails to encrypt files that don't match a specific rule.

The `path_regex` matches relative paths (relative to the sops invocation directory or repo root). On Windows, slashes are normalised.

Each rule can specify:

- `kms`, `gcp_kms`, `azure_kv`, `hc_vault` — cloud KMS recipients.
- `age`, `pgp` — local recipients.
- `key_groups`, `shamir_threshold` — for threshold mode.
- `encrypted_regex`, `encrypted_suffix`, `unencrypted_regex`, `unencrypted_suffix` — value selection.

A team's `.sops.yaml` is itself committed to git. It's not encrypted (it doesn't contain secrets, just public keys and recipient identifiers). It's the operational source of truth for encryption policy.

## Migration Patterns

From gpg-encrypted files: decrypt the file with gpg, save the plaintext, re-encrypt with sops to multiple recipients (PGP for backward compat, age for forward, KMS for production). This is a one-shot migration, typically scripted.

From raw KMS: the previous pattern was often "encrypt the whole file with KMS via a custom wrapper script." Migration: decrypt with the old script, run `sops -e` with the same KMS key, commit the new file.

From ansible-vault: ansible-vault encrypts the entire file with a passphrase. Migration: decrypt with `ansible-vault decrypt`, re-encrypt with sops. The diff is large (entire file becomes structured encrypted YAML), but the long-term win is significant: now diffable, multi-recipient, granular.

Every migration is a one-way door for git history: old commits still contain the old encryption format. If you need to *purge* the old encrypted forms, you'd have to rewrite git history (`git filter-branch` or `git filter-repo`) — a heavy operation.

## Integration Architecture

The cloud-native ecosystem has built deep sops integration:

**kustomize-sops** — a kustomize generator that decrypts sops files at kustomize-build time, producing decrypted YAML for `kubectl apply`. The decryption happens at build time, in a controlled environment (CI runner, dev laptop), never in the cluster.

**helm-secrets** — a Helm plugin (`helm secrets enc`, `helm secrets dec`, `helm secrets install`) that wraps Helm with sops awareness. Charts can have `secrets.yaml` (sops-encrypted) alongside `values.yaml`.

**sops-nix** — Nix module that decrypts sops files at NixOS deployment time, placing decrypted content in `/run/secrets/` (tmpfs, root-owned, app-readable). Used by NixOS users who want the entire system declarative.

**Flux CD** (`decryption.provider: sops`) — Flux's Kustomize controller can decrypt sops files in-cluster using a key stored as a Kubernetes secret. The key never leaves the cluster; Flux fetches encrypted manifests from git, decrypts in-cluster, applies.

**ArgoCD** with sops support (via plugins or operators) — same pattern as Flux.

**Terraform** (`hashicorp/sops` provider) — read-only data source for decrypting sops files at terraform-plan time.

The pattern across all of these: keep the encrypted file in git, decrypt at deploy/apply/build time, never persist plaintext.

## CI/CD Patterns

Short-lived KMS credentials: CI/CD systems should not have long-lived AWS access keys with KMS permissions. Instead, use IAM role assumption: the CI runner has a base identity (GitHub Actions OIDC, Argo Workflows service account, etc.), assumes a role with KMS permissions for the duration of the job, gets temporary credentials.

The "avoid stash decrypted files" rule: never write decrypted content to a CI cache, artifact store, or workspace that persists beyond the immediate job. Decrypt to memory, use, discard.

The age-key-as-secret pattern: store the age private key as a CI secret (GitHub Actions secret, GitLab CI variable). At the start of each job, write it to a temp file, set `SOPS_AGE_KEY_FILE`, sops uses it. At job end, the runner is destroyed and the secret is gone with it.

For multi-environment workflows: the dev branch decrypts dev secrets; the staging branch decrypts staging; the prod branch decrypts prod. The CI's identity for each branch is different (via OIDC subject claims or environment-specific runners), and only the right identity has access to the right KMS keys.

The "no developer ever sees prod secrets" pattern: prod secrets are encrypted only to the production KMS key + the production CI identity. Developers can read structure (key names) but not values. Operations engineers have a separate emergency-access key (carefully audited) for break-glass scenarios.

The audit trail: KMS-backed sops gives you a CloudTrail entry for every decryption. You can answer "who decrypted this file when" with confidence. PGP-backed sops has no equivalent audit; the decryption is purely local.

The disaster-recovery key: every prod sops file should have a recovery recipient (a long-term offline key) in addition to the operational keys. If the operational KMS region goes down, or if an operator is unavailable, the offline key can restore access. The offline key is air-gapped, paper-printed, in a safe.

### Editing flow internals

`sops file.yaml` (with no other args) launches `$EDITOR` against a decrypted temp file. The flow:

1. Read the encrypted file from disk.
2. Decrypt the data key via available recipient identities.
3. Decrypt all encrypted values, producing the in-memory plaintext document.
4. Serialise the plaintext to a temporary file (in tmpfs if available).
5. Compute a hash of the temp file's content as the "before" reference.
6. Spawn `$EDITOR` against the temp file.
7. When the editor exits, re-read the temp file.
8. If unchanged (hash matches), abort — no encryption needed.
9. If changed, re-serialise the new structure, encrypt new/modified values (preserving IVs for unchanged ones if `mac_only_encrypted: true` isn't set... actually no, IVs are always fresh), recompute MAC, write back.
10. Securely delete the temp file (overwrite + unlink).

Step 9 is the subtle part. By default, every encrypted value gets a fresh IV at every save, which means the encrypted-file diff after editing is *every* encrypted line, even if the underlying value didn't change. This is operationally annoying.

The mitigation: `mac_only_encrypted: true` in `.sops.yaml`. With this flag, sops attempts to detect "unchanged values" by comparing decrypted-old to decrypted-new; unchanged values keep their original ciphertext+IV+tag. The MAC is computed only over actually-encrypted values. The diff after editing now only shows the values that changed.

This is recommended for source-controlled sops files. The downside: a determined attacker observing two versions of a file can correlate which values changed (which is leaked anyway by the diff in git).

### Temp file safety

The temp file in step 4 above contains plaintext. Handling:

- Created with mode 0600 (user-only read/write).
- Placed in `os.TempDir()`. On Linux, this is `$TMPDIR` (usually `/tmp`) by default. `/tmp` may be tmpfs (RAM-only) or persistent disk depending on distro.
- On macOS, `$TMPDIR` is per-user (`/var/folders/...`) which is itself a kind of tmpfs.
- Securely overwritten with zeros before unlinking, in best effort.

For high-stakes use, set `TMPDIR=/dev/shm` (Linux tmpfs) before running sops, ensuring the plaintext never touches a journaling filesystem.

### Multi-document YAML files

A YAML file can contain multiple documents separated by `---`:

```yaml
---
db:
  password: secret1
---
api:
  key: secret2
```

sops handles this transparently: each document is encrypted independently with the same data key, so they share the same recipient list and MAC scope. The format preserves document separators.

This pattern is common in Kubernetes manifests (one file containing multiple resources) and Helm value files (one file with environment-specific overrides).

### Anchor and alias handling

YAML anchors (`&name`) and aliases (`*name`) are *not* preserved. sops resolves them to their target values before encryption. This is because:

- The anchor/alias relationship is a YAML-syntactic concept, not a data concept.
- After encryption, the encrypted values are different per-position (different IVs), so an anchor would break.
- Resolving them upfront is the simplest correct behaviour.

The implication: a sops-encrypted YAML may be longer (in plaintext) than the original if the original used aliases. This is rarely a problem.

### Comments preservation

YAML comments are preserved through the encrypt-decrypt round trip. Comments attached to encrypted values stay in the file:

```yaml
# Production database — change quarterly
db:
  password: ENC[...]  # last rotated 2024-Q1
```

Both comments survive editing.

JSON has no comments, so this is moot. TOML and INI both preserve comments.

The benefit: documentation of why a value is what it is can live next to the value, encrypted file or not.

### Rotation in regulated environments

For regulated environments (PCI-DSS, HIPAA, SOC 2), the rotation cadence is dictated by compliance, not gut feeling:

- **PCI-DSS Requirement 3.6.4**: cryptographic key rotation at the end of cryptoperiod, defined by the organisation but typically annual.
- **HIPAA**: no specific cadence, but reasonable practice is annual + on personnel change.
- **NIST SP 800-57**: detailed cryptoperiod recommendations by key type.

The ops translation:

- Annual `sops -r --in-place file.yaml` for all production files.
- On-personnel-change `sops updatekeys file.yaml` for affected files.
- Quarterly review of `.sops.yaml` against personnel roster.

These cadences can be automated: a CI job runs quarterly, lists files needing rotation, opens PRs.

### Sops with binary files

For non-text files (TLS private keys, PFX bundles, JKS keystores, JPEG images), use `--input-type binary`:

```bash
sops --encrypt --input-type binary --output-type binary cert.p12 > cert.p12.sops
```

The internal representation is base64-armored ciphertext. The "values" are a single big blob.

The diff is meaningless: any byte change in the input produces an entirely different output. But git tracks it without choking on raw binary.

The MAC still works because the entire plaintext is one "value" being authenticated.

The downside: the encrypted file is 4/3 the size of the plaintext (base64 armoring), so multi-MB files balloon.

### Migration to age across an existing repo

A typical scenario: a team has dozens of sops-encrypted YAML files using PGP, wants to migrate to age. The brute-force approach:

```bash
for f in $(find . -name '*.yaml' -exec grep -l '^sops:' {} \;); do
  sops -d "$f" > /tmp/plain.yaml
  sops -e -i --age $AGE_RECIPIENT /tmp/plain.yaml
  mv /tmp/plain.yaml "$f"
done
```

But this has issues:

- Decrypting to a tempfile is unsafe if the tempfile location is unsafe.
- The data key is rotated (because we're calling `sops -e` fresh).
- The MAC is recomputed.
- The git diff is huge.

The cleaner approach: edit `.sops.yaml` to add age recipients alongside PGP, run `sops updatekeys` on every file (preserves data key, just adds new wraps). Files now decrypt with both PGP and age. Then, after a soak period, edit `.sops.yaml` to remove PGP recipients, run `sops updatekeys` again to drop the PGP wraps. Files now only decrypt with age.

This is the "additive then subtractive" rotation pattern, applied to backend migration.

### Sops in declarative deployment

Tools like Argo CD, Flux, and Pulumi treat sops files as just another input. The decryption happens at deploy time:

- **Flux**: the SourceController fetches the git repo, the KustomizeController runs kustomize-sops to decrypt as part of build, applies decrypted manifests to the cluster.
- **Argo CD**: a plugin (argocd-vault-plugin or similar) handles sops decryption at sync time. The SyncOption.SecretsFromVault directive points at sops files.
- **Pulumi**: read sops files in TypeScript/Python via a community provider, pass values into resource definitions.

The pattern across all of these: decrypt right before apply, never persist plaintext, rely on the cluster's RBAC for runtime secret access.

### Structured value encryption

Sometimes you want to encrypt a structured value (a whole dict, not a leaf string). sops supports this when the encrypted value is a YAML/JSON-serialised structure:

```yaml
config:
  database: ENC[AES256_GCM,data:base64-of-json,...,type:bytes]
```

The `type:bytes` tells sops to treat the encrypted value as raw bytes (not a string). The application reading this must know to deserialise.

This is rarely the cleanest approach — usually you'd have leaf-only encryption. But for cases like "encrypt the entire database connection block as one unit," it works.

### Threat model nuances

The MAC catches a specific class of attacks: rearranging or substituting encrypted values. But it does not catch:

- **Key substitution**: an attacker who controls the `sops` block can swap the wrapped data key for one of their own, then re-encrypt all values under their key. The result decrypts cleanly (the consumer uses the modified key), but the values are attacker-controlled. Mitigation: verify the recipient list (which `.sops.yaml` enforces).
- **Selective denial**: an attacker drops one encrypted value. The MAC catches this if it's keyed off "all encrypted values," but if `mac_only_encrypted` is misconfigured, dropping a value might still validate.
- **Replay**: an old version of the file (with old values) presented as current. Mitigation: out-of-band freshness checks (git timestamps, version comments).

For the highest-stakes use, sign the entire file with `minisign` or `ssh-keygen -Y sign` separately from sops, and verify the signature in the consuming pipeline.

### Performance

Sops encryption performance is dominated by:

1. KMS round-trip (if using KMS): ~50-100ms per call.
2. Per-value AES-GCM: microseconds per value.
3. YAML/JSON serialisation: microseconds per kilobyte.

For a typical 1KB sops file with 5 encrypted values and one KMS recipient, encryption takes ~100ms (dominated by the KMS call). Decryption is the same shape: one KMS call, fast local AES-GCM.

For high-frequency use cases (CI/CD pipelines decrypting many files), the KMS call dominates throughput. Mitigations:

- Cache the data key for the lifetime of the job (decrypt once, reuse).
- Use age or PGP for development files where KMS round-trips are unwelcome.
- Use a local Vault instance for low-latency wrap/unwrap.

### Why MAC over plaintext, not ciphertext?

A natural question: why does sops MAC the *plaintext* values, not the ciphertext?

Reasons:

- The MAC catches modifications to the plaintext semantics, not just byte-flips. If an attacker re-encrypts a value (which they could do if they have the data key), the ciphertext changes but the plaintext might be the same — MAC over plaintext detects neither (no change) nor false-positives (rotated IV).
- Computing the MAC over plaintext means the order of fields in the canonical serialisation matters; sops defines a canonical ordering that's deterministic.
- Encrypted values' ciphertexts include random IVs, so a MAC-over-ciphertext would change on every re-encryption even of unchanged values. MAC-over-plaintext is stable.

The trade-off: MAC over plaintext requires having the plaintext at MAC time, which means decrypting before MACing on the encrypt side, and decrypting before MACing on the verify side. This is fine but requires careful ordering.

### Comparison: sops vs Vault dynamic secrets

Both tools handle "secrets in code" but at different layers:

- **sops**: encrypted file at rest, decrypted at deploy/build time, consumed by the application.
- **Vault dynamic secrets**: app authenticates to Vault at runtime, Vault generates ephemeral credentials, app uses them, credentials expire automatically.

Trade-offs:

- sops is simpler operationally (no Vault to operate, no auth integration).
- Vault dynamic secrets are more secure for short-lived credentials (expire automatically, no human-managed rotation).
- sops works offline; Vault requires connectivity at runtime.
- sops has a Git audit trail of "who modified the encrypted secret"; Vault has an audit trail of "who used the secret."

Most production deployments use both: Vault for runtime credentials (database passwords, AWS API keys generated per-app), sops for static configuration (TLS certs, CA bundles, app config flags).

## References

- sops project — `https://github.com/getsops/sops`
- sops by Mozilla — original repo, archived at `mozilla/sops`
- CNCF sandbox — sops moved to CNCF sandbox after Mozilla
- "SOPS: Secrets management for the cloud era" — CNCF blog
- AWS KMS documentation — `https://docs.aws.amazon.com/kms/`
- AWS CloudTrail logging for KMS — `https://docs.aws.amazon.com/kms/latest/developerguide/logging-using-cloudtrail.html`
- Azure Key Vault — `https://learn.microsoft.com/en-us/azure/key-vault/`
- Azure soft-delete — `https://learn.microsoft.com/en-us/azure/key-vault/general/soft-delete-overview`
- GCP Cloud KMS — `https://cloud.google.com/security-key-management`
- GCP Cloud KMS resource hierarchy — keyring/cryptoKey
- HashiCorp Vault transit engine — `https://developer.hashicorp.com/vault/docs/secrets/transit`
- age — `https://age-encryption.org/v1`
- GnuPG — `https://gnupg.org/`
- Shamir Secret Sharing — original 1979 paper, "How to Share a Secret"
- RFC 5116 — Authenticated Encryption (AEAD) construction
- NIST SP 800-38D — AES-GCM
- HMAC RFC 2104 — Keyed-Hashing for Message Authentication
- kustomize — `https://kustomize.io/`
- KSOPS plugin — `https://github.com/viaduct-ai/kustomize-sops`
- helm-secrets — `https://github.com/jkroepke/helm-secrets`
- sops-nix — `https://github.com/Mic92/sops-nix`
- Flux CD sops integration — `https://fluxcd.io/flux/guides/mozilla-sops/`
- ArgoCD secrets plugin — `https://argo-cd.readthedocs.io/`
- Terraform sops provider — `https://registry.terraform.io/providers/carlpett/sops/`
- "Why we use sops" — various engineering blog posts
- "Secrets management at scale with sops and KMS" — talks
- sops-as-edit-mode — sops launches `$EDITOR` against decrypted, re-encrypts on save
- sops file format spec — `docs/` in the sops repo
- "Threshold cryptography in sops" — sops design docs
- OWASP secrets management cheat sheet — secrets in CI/CD
- CIS Benchmarks for KMS — AWS, Azure, GCP variants
