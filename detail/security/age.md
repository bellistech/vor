# age cryptography internals

A deep dive into Filippo Valsorda's `age` — the file encryption tool that replaces GPG for the file-at-rest use case. The why behind a deliberately small format.

## Setup

`age` (pronounced like the Italian *aghe*, or just "age") was published in 2019 by Filippo Valsorda and Ben Cartwright-Cox as a corrective to PGP. The thesis is in the name of the spec: **age-encryption.org/v1**. Version one. Frozen. No version two coming. No flag soup. No web of trust. No subkeys. No mode bits. No compression options.

The design rationale, articulated by Filippo in talks and the design rationale doc:

1. **Modern primitives**. X25519 for ECDH, ChaCha20-Poly1305 for AEAD, HKDF-SHA-256 for key derivation, scrypt for passphrase hardening. These are the 2010s-era choices: fast, sidechannel-resistant on commodity hardware, unencumbered by patents, and with mature constant-time implementations.

2. **Small attack surface**. The reference implementations are a few thousand lines. The format spec is a few pages. There is no "encrypted body that contains compressed data containing PGP packets" recursion. There is no "session key wrapped in another session key" indirection.

3. **Auditable format**. The header is human-readable text (lines like `-> X25519 SshAFx...`) with a binary AEAD payload underneath. You can `head` an age file and see what's going on. PGP packets are TLV-encoded binary that requires `gpg --list-packets` to inspect.

4. **No PGP cruft**. The mistakes of PGP — encrypt-then-MAC done wrong (modification detection codes), variable-strength MDC, signing-then-encryption order, ASCII armor that doesn't authenticate, message segmentation that interacts with compression, key servers, web of trust — all gone. Each of these has a 20+ year history of CVEs and confusion.

5. **No signing**. age does not sign. Authentication of the sender is punted to a separate tool (Filippo's `minisign`). This is the most controversial design decision: PGP conflated encryption and signing, and age explicitly separates them. The argument: encryption recipients are not the same as signature verifiers; the trust models are different; conflating them produces foot-guns.

6. **Recipients-only**. There is no implicit "who sent this." The decrypter learns only that *they* could decrypt — they learn nothing about the sender. If you want sender identity, you sign a separate file.

7. **Seekless format**. The file is meant for streaming. You can encrypt stdin to stdout, decrypt stdin to stdout. There is no "seek to chunk N" requirement. This rules out random-access encrypted filesystems but enables shell pipelines and tar streams.

8. **One spec, two implementations**. The Go reference (`filippo.io/age`) and the Rust port (`str4d/rage`). Both pin to the same spec. No proliferation of incompatible variants.

The result is a tool that does encryption-at-rest for files, period. It doesn't replace TLS, doesn't replace SSH, doesn't replace Signal, doesn't sign, doesn't manage keys for you. It does one thing.

## Format Spec Walkthrough

An age file has three regions in order:

```
age-encryption.org/v1                ← version line
-> X25519 SshAFx...                  ← recipient stanza 1
abc...                               ← stanza body (base64)
-> X25519 OtherP...                  ← recipient stanza 2
def...                               ← stanza body
-> scrypt MS9pe... 18                ← passphrase stanza (alternative)
ghi...                               ← stanza body
--- HMAC_OVER_HEADER_BASE64          ← HMAC line, terminates header
[BINARY PAYLOAD]                     ← encrypted body (ChaCha20-Poly1305 chunks)
```

The version line is exactly the bytes `age-encryption.org/v1\n`. This is the format identifier and the version. There is no v2. If a future variant exists, it would be `age-encryption.org/v2` and would be a different format entirely — no in-place upgrade path.

Each stanza is one or more lines:

```
-> TYPE arg1 arg2 ...
base64(stanza body, line-wrapped at 64 columns)
```

The `TYPE` is the recipient type: `X25519`, `scrypt`, `ssh-rsa`, `ssh-ed25519`, or a plugin-registered name like `yubikey`. The `arg1 arg2 ...` are type-specific positional arguments (typically the recipient's pubkey or salt or keyhandle).

The stanza body is base64-encoded, wrapped at 64 columns, no padding (i.e. raw base64 without trailing `=`). The body decodes to type-specific binary data, typically the wrapped *file key*.

The HMAC line starts with `---` followed by the base64-encoded HMAC-SHA-256 of all preceding bytes (including the version line and all stanzas). The HMAC key is derived from the file key.

After the HMAC line and its terminating newline, the binary payload begins immediately.

## X25519 Recipient Stanza

The X25519 stanza is the canonical recipient form. The encryption flow:

1. Generate a random 16-byte file key `K` (this is the symmetric key for the payload).
2. For each X25519 recipient:
   a. Generate an ephemeral X25519 keypair `(e_priv, e_pub)`.
   b. Compute the ECDH shared secret: `shared = X25519(e_priv, recipient_pub)`.
   c. Derive the wrap key: `wrap_key = HKDF-SHA-256(salt = e_pub || recipient_pub, ikm = shared, info = "age-encryption.org/v1/X25519")`.
   d. Encrypt `K` under `wrap_key` with ChaCha20-Poly1305, nonce = all-zero (it's safe because the wrap_key is unique per stanza thanks to the ephemeral pubkey in the salt).
   e. Emit stanza: `-> X25519 base64(e_pub)\nbase64(ciphertext_of_K)`.
3. After all stanzas, compute HMAC over the header.
4. Encrypt the payload under `K`.

The wrap is essentially HPKE (Hybrid Public Key Encryption) Mode-Base, but predates the RFC. The salt-binding-pubkeys construction is the key insight: even if the same recipient is encrypted multiple times with the same plaintext, each stanza gets a different wrap key (because `e_pub` is fresh each time), so stanzas are never byte-identical, and an attacker can't tell whether two age files have the same recipient.

The ephemeral X25519 keypair is critical: it ensures forward secrecy in a limited sense — if the recipient's long-term private key leaks *later*, an archived ciphertext is still decryptable (because the recipient's private key still works). But if the *ephemeral* key leaked at encryption time (which would mean a compromised RNG at encryption time), only that one file is at risk. Practical FS for archival files is impossible because the recipient must be able to decrypt at any future time.

The X25519 implementation is constant-time on every reasonable platform. Go uses `crypto/ecdh`. Rust uses `x25519-dalek`. Both are well-audited.

## scrypt Passphrase Stanza

The scrypt stanza is for *interactive* encryption: you run `age -p < file` and type a passphrase. There can be only one scrypt stanza per file (it's an exclusive recipient mode, mutually exclusive with X25519 stanzas).

The flow:

1. Generate a random 16-byte salt.
2. Choose `N = 2^18 = 262144` (memory cost), `r = 8`, `p = 1`.
3. Derive `wrap_key = scrypt(passphrase, "age-encryption.org/v1/scrypt" || salt, N, r, p, 32 bytes)`.
4. Encrypt the file key `K` with ChaCha20-Poly1305 under `wrap_key`, nonce = zero.
5. Emit stanza: `-> scrypt base64(salt) 18\nbase64(wrapped_K)`.

The third argument `18` is `log_2(N)`. This is the only knob exposed in the format. The decrypter validates that `log_2(N) <= 22` (i.e. N <= 4 million) to prevent a malicious file from making decryption take forever.

The parameters were chosen for "interactive" use: ~256MB of RAM, ~1 second per try on a modern laptop. The `r=8, p=1` choice mirrors the original scrypt paper's reference parameters. With these, the per-try cost is ~256MB-seconds.

The "can't be batched" property of scrypt is the design rationale. Unlike PBKDF2 (CPU-bound), scrypt is memory-bound: you can't crack a thousand passphrases in parallel on a single GPU because each one needs 256MB of RAM. The cost ratio of attacker-vs-defender is much closer to 1 than for hash-based KDFs. This makes scrypt the right choice for human-typed passphrases.

The interaction with hardware accelerators is interesting: GPUs have plenty of memory but slow random-access patterns; scrypt's `Mix` step is designed to be access-pattern-hostile to GPUs. ASICs can do better, and dedicated FPGA scrypt crackers exist (originally for Litecoin mining), but they're far less effective than the GPU-vs-CPU advantage you see with PBKDF2 or bcrypt.

The 256MB memory cost is also why scrypt-passphrase age files are interactive-only: you wouldn't want to decrypt a 100GB tar with a passphrase, because the *header* decryption takes a second. But the *payload* is symmetric ChaCha20-Poly1305 — fast as memcpy.

## SSH-RSA Recipient

age supports using an SSH RSA public key as a recipient. The convention: `~/.ssh/id_rsa.pub` is your age recipient. This was the second-most-controversial design decision (after no-signing). The argument: people have SSH keys; making them double as age keys avoids a parallel key infrastructure.

The wrap algorithm is RSAES-OAEP-SHA-256-MGF1. RFC 8017 §7.1. The label is the constant string `age-encryption.org/v1/ssh-rsa`. The output is the wrapped file key `K` (so the RSA modulus must be at least 2048 bits to fit a 16-byte K with OAEP padding overhead).

The stanza arg is the SHA-256 fingerprint of the SSH public key (truncated to 4 bytes, base64-encoded), which lets the decrypter quickly skip stanzas not for them.

The threat model implication is real. Your SSH key is now used in two contexts:

- **SSH authentication**: signs a challenge from the server.
- **age decryption**: unwraps a file key.

If the SSH key is on a YubiKey with PIV, and the YubiKey has been touch-required-for-signing-only (the common config), age decryption with that key will *not* require touch — because age is doing decryption (a separate PIV operation), not signing. This is a configuration trap.

If the SSH key is in `~/.ssh/id_rsa` with a passphrase, age has to prompt for the passphrase. There's no SSH agent for decryption.

The simpler rule: **for high-stakes decryption, generate a dedicated age X25519 key, don't reuse SSH**. The SSH-key support is a convenience for low-stakes operational use ("encrypt this config file to my laptop").

## SSH-Ed25519 Recipient

age also supports Ed25519 SSH keys. This is more interesting because Ed25519 is a *signing* algorithm, not a key-agreement algorithm — but X25519 (which is) shares the same underlying curve (Curve25519). The conversion is the Edwards-to-Montgomery birational map.

The flow:

1. Take the recipient's Ed25519 public key A.
2. Convert to Montgomery form: `u = (1 + y) / (1 - y)` where `y` is the Edwards y-coordinate. This gives the X25519 public key.
3. Now do the standard X25519 ECDH wrap (same as the X25519 recipient stanza).

For the *private* side, Ed25519 private keys are typically stored as a 32-byte seed. The Ed25519 scalar is `clamp(SHA-512(seed)[:32])`. This scalar can be used directly as an X25519 scalar (after applying the standard X25519 clamping bits, which Ed25519's clamping happens to match exactly modulo a sign bit).

The stanza arg is the SHA-256 fingerprint of the SSH key (4-byte truncated base64) plus the converted X25519 ephemeral pubkey.

The "magic" is that this works because Curve25519 was designed for both signing (via Edwards form, Ed25519) and key-agreement (via Montgomery form, X25519). The mapping between them is well-defined and lossless.

This is the killer feature for age: a developer with `~/.ssh/id_ed25519.pub` can be a recipient with zero setup. They send you their pubkey (e.g. `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5...`), you put it in your recipients list, you encrypt to them. They decrypt with the same key they SSH with.

## Plugin Stanza Format

The plugin protocol enables third-party recipient and identity types: hardware tokens, TPMs, KMS, password managers. The protocol is over stdio, with a discovery rule: a recipient prefixed `age1plugin1...` (or similar `age1<name>...`) is dispatched to a binary `age-plugin-<name>` found on `$PATH`.

Stanza format for a plugin:

```
-> piv-p256 SLOT_ID OTHER_ARGS
base64(plugin-defined body)
```

Where `piv-p256` is the plugin name. age treats the body as opaque; the plugin parses it.

The protocol between age and plugin is a simple length-prefixed message stream:

- age writes recipient/identity strings to plugin stdin.
- plugin responds with stanzas (encryption) or file-key bytes (decryption).
- A control channel handles UI (PIN prompts, touch prompts) by emitting `Msg`/`Confirm`/`Request` messages back to age.

age forwards UI messages to the user (via TTY or `pinentry` if non-interactive). The plugin doesn't need to know how to draw a UI — it just describes what it needs.

This is how `age-plugin-yubikey`, `age-plugin-tpm`, `age-plugin-fido2-hmac`, and others integrate without modifying age itself. Each plugin is its own binary, its own dependencies, its own audit trail.

The discovery is by PATH lookup, which means a malicious binary on PATH could intercept a plugin call. age does not currently verify plugin binaries; if you're worried, run age in a restricted PATH.

## ChaCha20-Poly1305 Payload

Once the header (with stanzas and HMAC) is written, the payload begins. The payload is encrypted under the file key `K` using ChaCha20-Poly1305, in 64KB chunks.

Each chunk is encrypted with a per-chunk key derived from `K`:

```
chunk_key = HKDF-SHA-256(salt = nonce_prefix, ikm = K, info = "payload")
```

where `nonce_prefix` is a per-file random 16 bytes written immediately after the header HMAC. Then each chunk's nonce within ChaCha20-Poly1305 is the 12-byte tuple:

```
nonce = counter (11 bytes, big-endian) || last_chunk_flag (1 byte)
```

The counter starts at 0 and increments per chunk. The last-chunk flag is 0x01 on the final chunk and 0x00 otherwise. This is the *length oracle* defense: an attacker can't truncate the file (drop a chunk from the end) without the receiver detecting that the supposed-last-chunk has flag 0x00, which is a different nonce than what was used at encryption time, so authentication fails.

Each chunk is up to 65535 bytes of plaintext. The ciphertext is 65535 + 16 (Poly1305 tag) = 65551 bytes. The receiver reads up to 65551 bytes, decrypts, advances the counter. If the chunk is short (less than 65535), the receiver knows it's the last chunk and verifies the last-chunk flag.

Why 64KB? It's a balance:

- Smaller chunks → more authentication overhead (each chunk has 16-byte tag).
- Larger chunks → more memory pressure (each chunk is buffered in full for AEAD).
- 64KB is large enough that overhead is negligible (16/65536 ≈ 0.024%) and small enough that buffering is fine.

The stream is ChaCha20-Poly1305 in chunked-AEAD mode, also known as STREAM. It's analogous to TLS 1.3's record layer, but without record types or version negotiation.

The per-chunk integrity is the key property: a multi-gigabyte file's chunks are individually authenticated. Truncation, reordering, or tampering with any chunk fails authentication. This contrasts with PGP's one-MAC-over-everything model, which had the modification detection code (MDC) bug where an attacker could downgrade a packet's MDC to skip integrity checking.

## HMAC over Header

The header HMAC line uses HMAC-SHA-256 over all bytes from the start of the file up to (but not including) the `---` of the HMAC line.

The HMAC key is derived:

```
hmac_key = HKDF-SHA-256(salt = nil, ikm = K, info = "header")
```

The HMAC ensures:

- The version line wasn't tampered with.
- No stanza was added, removed, or modified.
- The set of recipients (and their wrap data) is exactly what the encrypter wrote.

If an attacker added a stanza of their own (with a wrap of `K` to *their* pubkey), the HMAC would fail because they don't know `K`. This binds the recipient list to the file key.

The HMAC is also the only authentication of the header *prior* to payload integrity checks. Without it, an attacker could swap stanzas around, and even though the file key would still decrypt the payload, the recipient list would lie about who could decrypt.

## ARMOR ASCII

age's binary format can be wrapped in PEM-style ASCII armor:

```
-----BEGIN AGE ENCRYPTED FILE-----
YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUx...
...
-----END AGE ENCRYPTED FILE-----
```

The armor is:

- Standard base64 (with `=` padding) of the binary file.
- Line-wrapped at 64 columns.
- Wrapped in `-----BEGIN AGE ENCRYPTED FILE-----` / `-----END AGE ENCRYPTED FILE-----` markers.

The armor adds ~33% size overhead. It's used for environments that can't preserve binary safely: email, copy-paste, JSON config, env vars.

Critically, the armor is *not* authenticated separately. An attacker who modifies the base64 will produce a file that still decrypts cleanly if their modifications happen to round-trip through base64 — but the underlying binary integrity check (HMAC + per-chunk AEAD) catches modifications. The armor markers themselves are trivially modifiable but irrelevant: the decrypter only needs the binary, not the markers.

## Threat Model

age protects:

- **Confidentiality of file contents** under recipient-key compromise resistance for currently-uncompromised recipients.
- **Integrity of file contents** under HMAC-over-header + per-chunk AEAD.
- **Authentication of the recipient list** — the HMAC binds the stanzas.
- **Resistance to chosen-ciphertext attacks** — the AEAD construction is IND-CCA2.

age does not protect:

- **Sender identity**. Anyone who can encrypt to your pubkey can produce a file you'll decrypt. There's no built-in signature.
- **Forward secrecy in the strict sense**. The recipient's long-term private key, if compromised in the future, decrypts old files.
- **Recipient list privacy**. The number of recipients is visible; the wrap data is visible per recipient. A passive observer can count recipients.
- **Traffic analysis**. File sizes leak rough plaintext sizes (modulo padding, which age doesn't do automatically).
- **Timing side channels in plugins**. age itself is constant-time-ish, but plugin authors must be careful.
- **Endpoint compromise**. Same caveat as every encryption tool: if your laptop is compromised, your secrets are gone.

The "no signing" is a deliberate gap. To authenticate a sender, you sign the *plaintext* with `minisign` or `signify`, then encrypt the (plaintext + signature) bundle with age. The recipient decrypts with age, verifies with minisign. This is more steps than PGP's "sign and encrypt in one go," but the trust model is cleaner: encryption keys are not signing keys.

## age vs gpg

Algorithm modernity:

| Aspect | age | gpg |
|--------|-----|-----|
| Asymmetric | X25519 | RSA / ECDSA / EdDSA / ECC |
| Key derivation | HKDF-SHA-256 | various, depends on packet version |
| Symmetric | ChaCha20-Poly1305 | AES-128/192/256 + various MAC modes |
| Passphrase KDF | scrypt | S2K (iterated SHA-* by default) |

Format simplicity:

- age: text header + binary payload, ~150 lines of spec.
- gpg: nested OpenPGP packets, RFC 4880 + extensions, ~120 pages of spec.

The web-of-trust replacement: age has no WOT. Recipients are public keys. There's no key-signing party, no trust signal, no "ultimate / full / marginal trust" levels. You decide who's a valid recipient; the format doesn't help you decide.

Signing is delegated to `minisign` (or `signify`, `ssh-keygen -Y sign`). This is a deliberate functional split. PGP combined them and got both wrong; age does encryption only and gets that one job right.

Operational consequences:

- No key servers. You distribute pubkeys however you distribute them (git, email, chat).
- No subkeys. Your encryption key and your signing key are separate keys, possibly from different tools.
- No key expiration. age keys live forever unless you stop using them.
- No revocation. To "revoke," you stop trusting and re-encrypt outstanding files to a new recipient.

The "no revocation" is the same gap gopass has for cloned-store-departures: a recipient with the private key can decrypt files encrypted to them, forever. There's no cryptographic shortcut.

## age vs openssl enc

`openssl enc` is the historical Unix file-encryption tool. Its defects:

- **Key derivation**: by default, MD5-based PBKDF1 with one iteration. This is *trivially* brute-forceable with a GPU. Adding `-pbkdf2 -iter 1000000` helps but the default is broken.
- **No authentication**: `openssl enc -aes-256-cbc` is *unauthenticated* CBC. There is no MAC. An attacker can flip bits in the ciphertext, and the decrypter has no way to know. This has caused security incidents (the Padding Oracle attack family).
- **No format versioning**: an `openssl enc` output file has a fixed structure (`Salted__` + 8-byte salt + ciphertext) with no version field. If a future openssl release wants to change parameters, there's no way to indicate which file uses which.
- **Implicit cipher**: the cipher is encoded in the *command line*, not the file. The decrypter must know what cipher was used; the file doesn't say.

age fixes all of these:

- KDF is scrypt (passphrase) or HKDF (key wrap), both modern, neither trivially brute-forceable.
- AEAD throughout: header HMAC + ChaCha20-Poly1305 payload.
- Format-versioned: `age-encryption.org/v1`. A v2 file would be self-identifying.
- All parameters in the file: stanza types, args, sizes are explicit.

The migration path from `openssl enc` to age is straightforward: re-encrypt everything. There's no in-place upgrade because the formats are unrelated.

## rage (Rust port)

`rage` is the Rust implementation by Jack Grigg (`str4d`). It tracks the same spec and produces byte-compatible output.

Differences from go `age`:

- Built on the RustCrypto crate ecosystem (`x25519-dalek`, `chacha20poly1305`, `hkdf`).
- Smaller default static binary (~3MB stripped).
- Plugin protocol is identical, so plugins are language-agnostic.
- Some plugins are Rust-only (`age-plugin-yubikey`, `age-plugin-fido2-hmac`).
- Memory zeroisation is more aggressive — Rust's `zeroize` crate explicitly clears stack frames and structs; Go's GC-managed memory is harder to zero deterministically.

Performance characteristics:

- For typical files (~MB), throughput is dominated by I/O. Both implementations saturate disk on encryption.
- For micro-benchmarks, rage is sometimes faster on the X25519 step (dalek is heavily SIMD-optimised). go-age uses standard library `crypto/ecdh`.
- For multi-GB streams, both are pipelinable and process at ~1-2GB/s on modern hardware (ChaCha20-Poly1305 is the ratelimiter).

Both implementations are widely deployed in production. The choice between them is a function of which language ecosystem you're already in.

## YubiKey Plugin

`age-plugin-yubikey` integrates age with PIV (Personal Identity Verification) slots on a YubiKey 4/5. PIV slots:

- **9a (Authentication)**: typically used for SSH; touch policy on PIV doesn't always carry to ssh-agent.
- **9c (Signing)**: requires PIN every time by default.
- **9d (Key Management)**: typically used for decryption (PKCS#7); pin-cached after first use.
- **9e (Card Authentication)**: physical access; rarely used for crypto.

For age, the recommended slot is `9a` or `9d`, with policies:

- **PIN policy**: `default` (PIN once per session), `always` (PIN every operation), or `never` (no PIN).
- **Touch policy**: `default` (no touch), `always` (touch every operation), or `cached` (touch cached for 15s).

A high-stakes config:

- Slot `9a`, PIN policy `always`, touch policy `always`.
- Each age decryption requires PIN entry and a physical touch of the YubiKey.
- The YubiKey's PIV applet uses ECDH P-256 (or ECDSA P-256 for signing).

Wait — but age uses X25519, not P-256. How does this work?

The answer: `age-plugin-yubikey` uses the YubiKey's P-256 capability, not X25519 directly. The plugin generates a P-256 keypair on the YubiKey and stores it in the chosen slot. The age recipient string is `age1yubikey1...` with the encoded public key. The plugin protocol handles the wrap on the encryption side and the unwrap on the decryption side, using ECDH-P256 not X25519. This is invisible to age itself, which only sees the plugin-encoded stanza.

The downside: P-256 is not curve25519. Different mathematical structure, different threat history. P-256 is heavily standardised (NIST, FIPS 186-4) but has known concerns: the seed values for the curve parameters are unexplained ("nothing-up-my-sleeve" not satisfied), and the implementation is harder to make constant-time than 25519. In practice, the YubiKey hardware is a constant-time accelerator, so this concern is moot.

A reasonable threat model: if you trust the YubiKey's secure element (and you should, it's a Common Criteria EAL5+ chip), the algorithm differences are fine.

## TPM Plugin

`age-plugin-tpm` integrates age with a TPM 2.0 module. Most modern laptops have one (Intel PTT, AMD fTPM, or a discrete chip).

The plugin generates a sealed key in TPM NVRAM. "Sealed" means the key is bound to:

- **PCR state**: specific platform configuration registers (boot measurement, TPM EventLog).
- **Hierarchies**: typically Storage Hierarchy (SH), bound to the platform's storage owner.
- **Auth value**: optional PIN.

When you decrypt an age file with a TPM-sealed identity:

1. The plugin asks TPM to unseal the wrap key.
2. TPM checks current PCR state matches the sealed PCR state.
3. If they match, TPM emits the unsealed key.
4. If they don't match (e.g. you booted a different OS), TPM refuses.
5. The plugin uses the unsealed key to unwrap the age file key.

This means age files encrypted to a TPM identity are decryptable *only on that specific machine, with that specific boot state*. If you reflash firmware, change boot loader, change kernel — the PCR state changes — files become permanently undecryptable.

The non-exportability is the killer feature. An attacker who steals your laptop *and* your TPM can't extract the wrap key; the TPM only emits it during a successful unseal. They'd need to boot the same OS in the same state, which (in principle) requires breaking the secure boot chain.

The downside: hardware-bound encryption is fragile. Backups are essential. The recommended pattern is dual-recipient encryption: the file is encrypted to *both* a TPM identity (for fast/ergonomic local decryption) *and* an X25519 recipient stored offline (for disaster recovery).

## Backup + Recovery

The "lose key = lose data" reality is age's most operationally fraught aspect. Unlike a password (which you can write down or memorise), an age private key is 32 bytes of high-entropy data — practically impossible to memorise.

Backup strategies:

1. **Multi-recipient encryption**. Encrypt every important file to multiple recipients: your daily-use key, an offline-cold-storage key, a hardware-token key. Any one suffices to decrypt.

2. **Paper-printed identity**. The age private key is base64 (`AGE-SECRET-KEY-1...`). Print it on paper, store in a safe. Recovery requires typing it back in (or OCR + manual review).

3. **Shamir secret sharing**. Split the private key into N shares, threshold K. Distribute. (External tooling required; `ssss-split` or `tplsplit` work.)

4. **Hardware-token-with-backup-keys**. Use a YubiKey for daily decryption, and *also* generate a software age key kept offline.

5. **Encrypted backup of the key**. Encrypt your age private key under a memorable passphrase via age-itself: `age -p < age-secret-key > backup.age`. Store the passphrase-protected backup separately.

The fragility is sometimes overstated. With reasonable multi-recipient practice, the failure mode "all my keys are lost simultaneously" is rare. The common failure mode is "I lost my YubiKey," which is recoverable if you have a software-backup recipient.

## Streaming + Pipelines

age is designed for pipelines:

```bash
tar c /home/alice | age -r age1abc... > backup.tar.age
age -d -i ~/.config/age/key < backup.tar.age | tar x
```

The "no-seek" requirement of the format means age can encrypt stdin to stdout without buffering the entire input. This is essential for arbitrarily large inputs.

The concrete consequence: the format is sequential. The header is written first, then chunks in order, then end-of-file. There is no chunk index. There is no "skip to chunk 1000" capability.

This rules out random-access encrypted filesystems. If you need that, use `gocryptfs`, `cryfs`, or `fscrypt` — different tools for different jobs.

For streaming, the per-chunk authentication means a partial download can be decrypted up to the last fully-received chunk. The truncation is detected (last-chunk flag mismatch) only when the decrypter reaches what it thinks is the end. So you'll get a clean error rather than silent corruption.

## Format Stability

Filippo committed to the format being frozen on launch. The README says: "We do not anticipate any breaking changes to the format... We commit to keeping it backward-compatible for the foreseeable future."

The "10+ years" commitment is informal but credible: any change would require a `/v2` and a parallel implementation. The complexity cost of supporting two formats is enormous, and Filippo has been explicit that v1 is sufficient.

Practical implications:

- Files encrypted today will decrypt 10 years from now (assuming key custody).
- Implementations don't need to negotiate versions or feature flags.
- Tooling built on age (sops, gopass, terraform, kustomize) doesn't need to handle format upgrades.
- Long-term archival is plausible — store the file, store the key, recover any time.

The risk: cryptographic primitives age. ChaCha20-Poly1305 and X25519 are 2010s-era and considered safe today. If a fundamental flaw is found (very unlikely but possible), the format itself wouldn't change — the world would migrate to a new format, and old files would need re-encryption. age has no in-place upgrade story for primitives.

## Common Mistakes

The format-confusion gotcha: armored vs binary. An armored age file starts with `-----BEGIN AGE ENCRYPTED FILE-----`; a binary age file starts with `age-encryption.org/v1\n`. The wrong format passed to a tool that expects the other will fail with "incorrect format" or "no recipients." Always know which form you have.

The "I lost my key" reality: there is no recovery. age has no master key, no key escrow, no recovery seed. If you don't have the private key matching one of the recipients, the file is gone. Plan accordingly.

The "I have the key, but it's encrypted" recursion: people often passphrase-protect their age private key files (via age itself, recursively). This is fine, but you must remember the passphrase. Lost passphrase + passphrase-protected key = gone.

The "wrong recipient" mistake: encrypt to recipient A, tell decrypter "decrypt with key B." If A and B are different keys, decryption fails. Always confirm the recipient public key and the identity public key match before transmission.

The "age key in a backup" mistake: backing up `~/.config/age/key.txt` to a cloud service in plaintext is a single point of failure. The cloud provider's keychain or filesystem encryption is usually fine, but understand what you're trusting.

The "shared identity" anti-pattern: do not share the same age identity (private key) across multiple machines. Generate one per machine, encrypt to all of them. This way, losing one machine doesn't compromise the others.

The "age as TLS" mistake: age is for files-at-rest, not for transport. For transport, use TLS. age does not have forward secrecy at the transport level.

The "age for everything" mistake: age is for byte-streams. It's not a database encryptor (no random access), not a disk encryptor (no in-place updates), not a transport encryptor (no handshake). For each of those, use the right tool: `gocryptfs`/`fscrypt` for filesystems, LUKS/dm-crypt for block devices, TLS for transport, Signal/Olm for messaging.

The "I encrypted to the wrong key, can I re-encrypt?" recovery: yes, if you have the original plaintext or one of the matching identity files. Decrypt with what you have, encrypt with what you want, replace. There's no in-place re-key — every re-encryption produces a fresh nonce-prefix and per-chunk nonces, so the resulting file is byte-different.

The "where's my pubkey?" confusion: an age private key (identity) starts with `AGE-SECRET-KEY-1`. The corresponding public key (recipient) starts with `age1`. They derive from each other deterministically: `age-keygen -y identity.txt > recipient.txt`. People sometimes lose track of which file they have; the prefix tells you. Never share the secret-key form.

### Algorithm choice deep dive

Why X25519 over P-256? The arguments:

- X25519 is faster per operation on commodity hardware (~50% faster than P-256 in software).
- X25519 is easier to implement constant-time. P-256's prime order is sparse, requiring more careful scalar multiplication routines.
- X25519's parameter generation is fully transparent. P-256's seed values are unexplained NIST inputs, fueling concerns about possible backdoors (no public attack has materialised, but the trust posture differs).
- X25519 has fewer foot-guns: the Montgomery ladder doesn't expose intermediate values; P-256's projective representation requires care.

Why ChaCha20-Poly1305 over AES-GCM-256? The arguments:

- ChaCha20 is constant-time on every CPU. AES-GCM is constant-time on CPUs with AES-NI; without AES-NI (older ARM, embedded), table-based AES is timing-side-channel-vulnerable.
- ChaCha20 is faster than AES-GCM on CPUs without dedicated AES instructions (most ARM cores under power constraints).
- Poly1305 is a simpler MAC than GHASH; faster on the same hardware.
- The IETF and TLS 1.3 chose ChaCha20-Poly1305 as a co-equal alternative to AES-GCM, signalling its maturity.

Why HKDF over plain hashing? The argument:

- HKDF cleanly separates "extract" (concentrate entropy from a possibly-low-entropy input) from "expand" (generate as many bytes as needed). This decoupling is theoretically clean and avoids ad-hoc mistakes.
- Plain SHA-256(secret || context) has subtle issues with input concatenation (e.g. length-extension on some constructions, ambiguity if context is variable-length). HKDF formalises away these.

These choices — X25519, ChaCha20-Poly1305, HKDF — are essentially the IETF's "best modern crypto" set as of 2018. They've aged well; no flaws found.

### Format design choices revisited

Why text headers? The argument:

- Debuggable. Read the file, see what's going on. PGP packets require `gpg --list-packets`.
- Forward-extensible. New stanza types can be added; old implementations skip unknown ones (with proper handling).
- Tooling-friendly. Editors, less, head, all work on age files until the binary payload starts.

Why no compression? The argument:

- Compression-before-encryption leaks plaintext lengths via ciphertext lengths (CRIME, BREACH attacks for HTTPS demonstrated this).
- Compression complicates streaming.
- Users who want compression can pipe through `gzip` before age, which makes the trade-off explicit.

Why no chunked random access? The argument:

- Random access requires per-chunk metadata (offsets, lengths) which is either stored ahead-of-payload (forces full header buffering) or interspersed (complicates streaming).
- The use case (random access in encrypted bulk data) is a different tool's job.

Why HMAC over header before payload? The argument:

- An attacker modifying a stanza to redirect to their pubkey, without the HMAC, would produce a ciphertext that *they* can decrypt (because the encrypter would wrap K under their key) but the HMAC catches this because they don't know K to forge a valid HMAC.
- Without the header HMAC, even per-chunk AEAD doesn't catch stanza tampering: the chunks would still be authentic under K, but K's wrap to the legitimate recipient is missing.

Why per-chunk AEAD instead of one big MAC? The argument:

- Streaming. With one big MAC, you can't authenticate progressively — you must read the entire file before validating, which means buffering arbitrarily large inputs.
- With per-chunk AEAD, you authenticate as you go, which means a corrupted file fails fast.
- The 64KB chunk size balances per-chunk overhead (16-byte tag) against memory pressure (no need to hold a chunk longer than its decryption).

Why the last-chunk flag? The argument:

- Without it, an attacker who truncates the file (drops chunks from the end) produces a file that decrypts cleanly up to the truncation point. The receiver has no way to know there *was* a chunk N+1.
- With the flag, the supposed-last chunk has a different nonce than what the encrypter used, so it fails authentication. The receiver gets "authentication failure" rather than "looks valid but is incomplete."

### Comparisons to alternatives

vs **NaCl/libsodium** `crypto_box`: similar primitives, but `crypto_box` is for one-recipient direct encryption (no stanza format, no multi-recipient, no plugin protocol). It's a building block; age is the application.

vs **OpenPGP** (gpg, sequoia): age is dramatically simpler. PGP's format has 25+ packet types, a 30-year evolution, and incompatibility quirks between implementations. age has 1 format, 5-ish stanza types, and 2 implementations that match.

vs **encrypt-tar** patterns: `tar c | age | base64` for backups is a common substitute for `gpg --symmetric`. age's superior format gives confidence the backup is intact 10 years later.

vs **gocryptfs**: gocryptfs encrypts a *filesystem* (per-file headers, directory encryption, name encryption). age encrypts *files*. Different problems.

vs **veracrypt/luks**: block-device encryption. Different layer entirely.

vs **TLS pre-shared keys**: TLS-PSK is for two parties exchanging an active connection. age is for one party encrypting at rest for later retrieval. Different shapes.

### Operational deployment patterns

The "everyone on the team has age": each team member generates `age-keygen > ~/.config/age/me.key`, posts their pubkey to the team's wiki or the codebase, recipient lists are managed in `.sops.yaml` or `.gpg-id`-style files. Easy onboarding, easy offboarding (modify recipient list, re-encrypt outstanding files).

The "one age per service": each backend service has its own age identity, kept in the cloud secret store. Encrypted config artifacts in git are encrypted to the service identity. At startup, the service fetches its identity from the secret store, decrypts its config, runs.

The "age for backups, KMS for runtime": backup tarballs are age-encrypted offline (the backup operator has the only key). Live runtime config is KMS-encrypted (the cloud provider mediates access). Two layers, different threat models.

The "rotate age keys": age keys can be rotated, but it's a manual orchestration. Generate a new key, update recipients, re-encrypt outstanding files, retire the old key. There's no automatic rotation tooling in age itself; rotation is application-level (sops, gopass, custom scripts).

### Plugin protocol details

The plugin protocol over stdio is line-based, with a dialect:

```
-> recipient-v1 TYPE PUBKEY
-> identity-v1 IDENTITY
-> wrap-file-key
[base64 encoded file key]

-> ok
[response stanza body, base64]

-> done
```

(The exact wire format is specified in the age plugin spec; see references.)

The plugin discovery: age forks a subprocess `age-plugin-NAME` and pipes recipient/identity strings to it. The plugin can:

- Generate a new identity (during keygen).
- Encrypt a file key (encryption time).
- Decrypt a stanza body (decryption time).
- Display UI prompts (PIN, touch).

The UI prompts are streamed back to age, which forwards them to pinentry or the TTY. This means a YubiKey plugin doesn't need its own TUI — it just says "request touch" and age handles the user notification.

The implication: any cryptographic primitive can be added to age via a plugin. KMS-backed identities? Possible. Hardware wallet integration? Possible. Threshold cryptography? Possible (age-plugin-shamir exists). The format itself stays minimal; complexity moves to plugins.

### Why "just" 16 bytes for the file key?

The file key is 128 bits (16 bytes). Why not 256?

- ChaCha20-Poly1305 takes a 256-bit key. age uses HKDF to expand the 128-bit file key to a 256-bit ChaCha20 key. The security level is bounded by the 128-bit input.
- 128 bits is the post-quantum-secure level for symmetric crypto under Grover's algorithm. 256 bits doesn't help here unless you also have post-quantum public-key crypto (which age doesn't, intentionally).
- Smaller file keys mean smaller stanzas (less wrap overhead per recipient).
- 128 bits resists every classical attack indefinitely: a 2^128 brute force is infeasible with all current and projected computing.

So the choice is 128-bit symmetric security + classical (X25519) public-key, both at the same security level. Coherent design.

### The single-implementation risk

Both age and rage are by relatively small teams. There's no third independent implementation. If a serious flaw were found in both, the only options would be patch + recompile.

Mitigations:

- The format is simple enough that auditing it is tractable.
- Multiple security audits have been performed (informal community reviews, formal Trail of Bits review, etc.).
- The choice of well-known primitives (X25519, ChaCha20-Poly1305, HKDF) means flaws are likely to be caught at the primitive level by the broader cryptographic community before they affect age.

The risk is real but moderate. For most users, the simplicity benefits outweigh it.

### Long-term archival considerations

age claims format stability for 10+ years. For longer-term archival (decades), additional considerations:

- **Crypto migration**: at some point, ChaCha20-Poly1305 or X25519 may be deprecated. Migration is "decrypt with old age, re-encrypt with successor tool."
- **Implementation availability**: 30 years from now, is there still a working age binary that decrypts your files? The Rust and Go reference implementations should be buildable from preserved source for a long time, but operational support requires effort.
- **Key custody**: keys outlast tools. The 32-byte X25519 private key works for any X25519 implementation, today or tomorrow.

For digital archival serious enough to plan for decades, the practice is multi-format encryption: age + gpg + a third tool, with the redundancy that *some* combination will still be decryptable in any future state.

### Implementation gotchas

The `-i` and `-r` flag confusion: `-i` is identity (private key file, for decryption), `-r` is recipient (public key, for encryption). Mixing them up is a common error. The error messages are clear ("not an identity," "not a recipient"), but the reflexive habit of typing one or the other catches new users.

The `--passphrase` vs `-p` flag: identical, both invoke scrypt-passphrase mode. Using either with `-i` or `-r` is an error (can't combine recipient with passphrase mode). The format only allows one or the other per file.

Multi-recipient files: passing multiple `-r` produces a file with multiple X25519 stanzas. Any single recipient can decrypt independently. There is no quorum/threshold mode in age itself — for that, use age-plugin-shamir or sops's threshold mode.

The `-` (stdin/stdout) convention: `age -e -r RECIPIENT` reads stdin, writes stdout. `age -d -i KEY` reads stdin, writes stdout. This is fully composable with shell pipelines: `cat file | age -e -r KEY | base64 > encoded.txt`.

### Identity file management

An age identity file:

```
# created: 2024-01-15T12:00:00Z
# public key: age1abc...
AGE-SECRET-KEY-1XYZ...
```

The public key comment is informational; age doesn't parse it. The actual parsing relies on the `AGE-SECRET-KEY-1` prefix.

Multiple identities can be concatenated in one file. age tries each on decryption; the first that successfully unwraps wins. This is the multi-machine pattern: each machine has its own key, all keys are listed in the team's identities file.

Permissions: identity files should be `0400` or `0600`. Many age implementations warn if the file is group/world readable. The OS doesn't enforce this; it's the user's responsibility.

### age-keygen variants

`age-keygen` (no args) generates a new X25519 keypair, prints the identity to stdout, prints the recipient to stderr (so `age-keygen > key.txt` captures only the identity, recipient is shown to user).

`age-keygen -o file.txt` writes both to a file (header + identity).

`age-keygen -y file.txt` reads an identity, prints the matching recipient.

`age-keygen -y` (with stdin) is useful for "I have the secret key, what's the public key" computations.

`age-keygen --passphrase` (some forks) generates a passphrase-protected identity. Decrypting the identity file requires the passphrase. This is convenient for storing keys on a USB stick that might be lost.

### Plugin discovery resilience

Plugin binaries (e.g. `age-plugin-yubikey`) are looked up via `$PATH`. If multiple binaries with the same name exist, the first wins. age does not verify plugin signatures.

For high-stakes use, restrict `$PATH` to known-good directories before invoking age, or symlink the trusted plugin binary directly.

The plugin protocol is forward-compatible: a plugin that supports protocol v1 will work with future ages that support v1. Plugins that need v2 features fail cleanly on older ages.

### Comparison summary

| Feature | age | gpg | openssl enc |
|---------|-----|-----|-------------|
| Public-key | X25519 | RSA/ECDSA/EdDSA | (no) |
| Symmetric | ChaCha20-Poly1305 | AES + various MACs | AES-CBC (unauth) |
| KDF | HKDF / scrypt | S2K / various | PBKDF2 (with `-pbkdf2`) |
| Format | text+binary, simple | TLV packets | salt+ciphertext |
| Versioned | yes (v1 in name) | yes (packet versions) | no |
| Authenticated | yes (HMAC + AEAD) | yes (with MDC) | no (default), yes (-aead) |
| Multi-recipient | yes | yes | no |
| Streaming | yes | yes | yes |
| Signing | no (use minisign) | yes | no |
| Subkeys | no | yes | no |
| Web of trust | no | yes | no |

age's choice: do encryption excellently, do nothing else. Other tools handle signing, certificate management, key directory services. The Unix philosophy applied to crypto.

### Final synthesis

age represents a corrective movement in applied cryptography: take the lessons of 30 years of PGP, distill them into a small tool with strong primitives and a frozen format, and reject the temptation to add features.

The result is a tool whose surface area is small enough to audit, whose format is stable enough to trust for archival, whose primitives are modern enough to resist current attacks, and whose plugin architecture is flexible enough to support hardware tokens, TPMs, and KMS integration without bloating the core.

For the file-encryption-at-rest use case, age is the right answer in 2024+. It's the right answer for backup encryption, for sops's local backend, for gopass's modern crypto, for declarative deployments storing secrets in git.

For everything else (signing, transport, filesystem encryption, messaging), use the right tool for that job. age does one job well; the cryptographic ecosystem provides others.

The lesson age teaches: simplicity, when achievable, is itself a security property. Less code is fewer bugs. Less format is fewer edge cases. Less choice is fewer configuration mistakes. age maximises the things it doesn't do, and that's its primary contribution.

## References

- age homepage — `https://age-encryption.org/`
- age v1 spec — `https://age-encryption.org/v1`
- Filippo Valsorda's blog — `https://filippo.io/`
- age GitHub — `https://github.com/FiloSottile/age`
- rage (Rust port) — `https://github.com/str4d/rage`
- "Modern Cryptography" by Filippo (talk) — various recordings
- RFC 7539 — ChaCha20 and Poly1305 for IETF Protocols
- RFC 7748 — Elliptic Curves for Security (X25519)
- RFC 8032 — Edwards-Curve Digital Signature Algorithm (Ed25519)
- RFC 5869 — HMAC-based Extract-and-Expand Key Derivation Function (HKDF)
- RFC 7914 — The scrypt Password-Based Key Derivation Function
- RFC 8017 — PKCS #1 v2.2: RSA Cryptography Specifications (RSAES-OAEP)
- "The scrypt key derivation function" — Colin Percival, BSDCan 2009
- Curve25519 — Daniel J. Bernstein's original paper, 2006
- Ed25519 — Bernstein, Duif, Lange, Schwabe, Yang, 2011
- "The Salsa20 family of stream ciphers" — Bernstein
- "Why I plan to start using age" — various blog posts
- "Eliminating PGP" — Latacora blog
- minisign — `https://jedisct1.github.io/minisign/`
- signify — OpenBSD's signing tool
- age-plugin-yubikey — `https://github.com/str4d/age-plugin-yubikey`
- age-plugin-tpm — community-maintained
- age-plugin-fido2-hmac — `https://github.com/olastor/age-plugin-fido2-hmac`
- TPM 2.0 spec — Trusted Computing Group
- PIV NIST SP 800-73 — Personal Identity Verification
- HPKE RFC 9180 — Hybrid Public Key Encryption
- "STREAM: A Provable Security Treatment of the Use of Stream Ciphers in HTTPS" — Hoang, Reyhanitabar, Rogaway, Vizár
- libsodium — reference implementation of ChaCha20-Poly1305 and X25519
- monocypher — minimal C implementation
- "Audit of age" — informal community reviews
- "Age-encryption: Modern Cryptography Done Right?" — reviews and critiques
- gocryptfs — for the alternative random-access-encrypted-fs case
- ssh-keygen birational map — Curve25519 Edwards-to-Montgomery transformation
- DJB Curve25519 page — `https://cr.yp.to/ecdh.html`
- "The OpenPGP Format" RFC 4880 — for comparison with age's format goals
