# OWASP Authentication — Theory and Cryptographic Foundations

A deep dive into the math, threat models, and protocol design that justify the recommendations in `sheets/security/owasp-auth.md`. This is the *why* page: why Argon2id beats bcrypt for new systems, why PKCE was bolted onto OAuth, why SameSite=Lax broke a class of CSRF attacks, why magic-link emails are token-bearer authentication and not magic.

## Setup

Authentication is the cryptographic act of *proving identity* to a verifier. The verifier (the server) starts with a public claim ("I am Alice") and must end either accepting it as Alice or rejecting it. The proof must be:

- **Sound** — an adversary who is *not* Alice cannot make the verifier accept.
- **Complete** — Alice (with her secret) can always make the verifier accept.
- **Resistant** — soundness must hold against the adversary's full toolkit: passive eavesdropping, active man-in-the-middle, replay, phishing, server-side compromise of stored verifier material, side-channel timing leaks, and offline brute force after a database breach.

Every authentication mechanism we will examine is judged against that bar. A password-hash scheme that leaks information when the database is dumped fails *resistance*. An OAuth flow that lets an attacker swap out the redirect URI fails *soundness*. A WebAuthn ceremony that signs a challenge without binding it to the origin fails *resistance against active phishing*.

The threat model has six adversary classes worth naming:

1. **Passive network adversary** — sees ciphertext, can't modify. Defeated by TLS.
2. **Active network adversary** — can drop, modify, replay packets. Defeated by TLS + cert pinning + signed protocol messages.
3. **Phishing adversary** — controls a domain that looks like the real one. Defeated only by origin-bound credentials (WebAuthn passkeys).
4. **Server-DB-breach adversary** — has stolen the password verifier table. Defeated by slow-hash + salt + pepper.
5. **Client-XSS adversary** — has script execution in the victim's origin. Defeated by HttpOnly cookies + CSP.
6. **Client-CSRF adversary** — controls a different origin in the victim's browser. Defeated by SameSite + CSRF tokens + Origin checks.

Authentication design is the discipline of constructing protocols where, for each class, the cryptographic cost of breaking *resistance* exceeds the value of the protected resource. The math below is how we prove that.

## Argon2id

Argon2 won the Password Hashing Competition (PHC) in 2015 and was standardized as **RFC 9106** in 2021. It comes in three variants:

- **Argon2d** — data-dependent memory access. Maximum resistance to GPU/ASIC time-memory tradeoff attacks. Vulnerable to side-channel timing attacks on the host where it runs (memory access pattern depends on the password).
- **Argon2i** — data-independent memory access. Side-channel resistant (memory pattern is fixed, doesn't leak password info). Slightly weaker against tradeoff attacks because predictable access can be precomputed.
- **Argon2id** — hybrid. The first half-pass uses Argon2i (side-channel safe while the password is "fresh" in memory), the remaining passes use Argon2d (max GPU resistance). This is the recommended default for password hashing.

The function takes:

- `tCost` — number of iterations (passes over memory). Linear in cost.
- `mCost` — memory in KiB. Linear in attacker memory required.
- `parallelism` — number of independent lanes. Each lane gets `mCost / parallelism` memory.
- `salt` — at least 16 bytes, random per password.
- `secret` (optional) — the *pepper*, server-side global secret.
- `associated_data` (optional) — extra context binding (user_id, etc).

Output is a 32-byte (default) tag.

The defender's wall-clock cost is roughly:

```
defender_time = tCost × passes_over_mCost_bytes / memory_bandwidth
```

The attacker's per-guess cost on commodity GPU hardware is bounded below by the memory-bandwidth wall. A GPU with 500 GB/s memory bandwidth running 1000 parallel guess threads needs to feed each thread ~`mCost` bytes per pass. If `mCost = 19 MiB` and `tCost = 2`, each guess requires ~38 MiB of memory streaming. At 500 GB/s, the GPU caps at about 13,000 guesses/sec across all threads — vs ~10^10 guesses/sec for unsalted SHA-256. That five-orders-of-magnitude gap is the entire point of memory-hardness.

Formally, Argon2's security proof (Biryukov, Dinu, Khovratovich, 2016) shows that any attacker computing the function with less than `mCost` memory must spend `tCost × N^2` time where `N = mCost / block_size`, due to the cascading data-dependency in the SMix-style mixing function. ASIC attackers cannot meaningfully reduce this because silicon area scales linearly with memory, and the attacker's economic budget is dollars-per-guess, not joules-per-hash.

The 64-byte input block is filled with pseudorandom data via BLAKE2b in counter mode keyed by (password || salt || params || secret || ad). Then for each of `tCost` passes, every block is overwritten with `G(prev_block, ref_block)` where:

- `prev_block` is the immediately previous block (sequential dependency).
- `ref_block` is selected from earlier blocks via the variant's index function (data-dependent for d, data-independent for i).
- `G` is a permutation built from BLAKE2b's round function applied to the XOR of inputs.

The data-dependency forces sequential evaluation; the random reference forces wide memory; the multiple passes (`tCost`) compound both costs. After all passes complete, the final block is folded into the output tag.

## Argon2id Parameter Tuning

OWASP 2024 (and RFC 9106) recommend, for *interactive* logins:

- `tCost = 2`
- `mCost = 19 MiB` (19456 KiB)
- `parallelism = 1`

This budget targets ~50ms wall-clock on a 2020-era server CPU. Doubling `mCost` doubles attacker cost while only moderately increasing defender wall-clock (memory bandwidth, not CPU, dominates). Doubling `tCost` doubles both equally.

The *parallelism* parameter is subtle. Increasing `parallelism` from 1 to 4 splits the memory across 4 lanes that can run on 4 cores in parallel — defender wall-clock drops by ~4×, but the *total work* and *total memory* are unchanged. So an attacker with a 4-wide-SIMD GPU still needs the full `mCost` per guess. The intuition is "parallelism cuts defender latency without cutting attacker cost," which is why interactive flows benefit from `parallelism > 1` only when latency budget is tight.

Tuning recipe:

1. Pick a latency budget. Interactive login → 50–250 ms. Batch / once-per-day → up to 1 second.
2. Start with `mCost = 19 MiB`, `tCost = 2`, `parallelism = 1`.
3. Time it on production hardware. If under budget, double `mCost`. If over budget and `parallelism = 1`, increase `parallelism` to use more cores.
4. Once `mCost` is at the limit of what your service-RAM allows under concurrent login load (don't OOM under traffic spikes), bump `tCost` instead.
5. Re-tune annually. Hardware gets faster; what was 50ms in 2024 will be 25ms by 2027 and a halving of attacker cost.

A common mistake is "let's make it 5 seconds, that's super secure." This denial-of-services your own login flow: every concurrent login holds 19 MiB × tCost passes of bandwidth, and a flood of login attempts becomes a self-DoS. The right answer is rate-limiting + reasonable cost, not infinite cost per attempt.

## bcrypt

bcrypt was designed by Niels Provos and David Mazières in 1999, predating the modern memory-hard generation. It uses a modified Blowfish cipher's *EksBlowfishSetup* — an expensive key schedule designed to be slow even with optimization.

The cost factor is `2^N` rounds of key schedule:

- `cost=10` → 1024 rounds (~10ms)
- `cost=12` → 4096 rounds (~250ms) — current OWASP minimum
- `cost=14` → 16384 rounds (~1s)

Each `+1` to cost doubles the work. This log-linear scaling means it's easy to bump every couple of years.

bcrypt's inputs:

- 16-byte salt (raw bytes, encoded as 22 chars in the bcrypt-flavour Base64 alphabet)
- Up to 72 bytes of password (anything past byte 72 is silently truncated)
- Cost factor

Output is a 23-byte hash, encoded as 31 chars, prefixed with `$2b$NN$` where `NN` is the cost.

**The 72-byte limit is a footgun.** bcrypt was built when "long passwords" meant 16 chars. Today users paste 100-char passphrases or password-manager-generated 64-char strings. bcrypt silently ignores everything past byte 72.

The naive fix — "we'll pre-hash with SHA-256 and feed the 64-byte hex digest to bcrypt" — has a worse failure mode. If the SHA-256 output contains a NUL byte, some bcrypt implementations (notably the original OpenBSD one) treat the NUL as end-of-string and only feed bcrypt the prefix up to the NUL. A 1/256 chance per password of dramatically reduced entropy. Mitigation:

- Pre-hash with HMAC-SHA-256 using a server-side pepper, then base64-encode the 32-byte tag (no NULs, fits in 44 chars, well under 72).
- Or just use Argon2id for new systems.

There's also a *type-2y / type-2a / type-2b* mess. Original implementations had a sign-extension bug in the password loop. `$2b$` is the post-fix variant. Always use `$2b$` and reject `$2a$` for new hashes.

bcrypt has held up reasonably well — Blowfish itself isn't broken in the bcrypt usage, and 2^N scaling means the algorithm ages gracefully — but it has *no* memory-hardness. A modern GPU farm can bcrypt at billions of guesses per second. For new code, prefer Argon2id.

## scrypt

**RFC 7914** (Colin Percival, 2016, originally 2009 paper). Designed explicitly to be memory-hard.

Parameters:

- `N` — CPU/memory cost. Must be a power of 2. Memory used = `128 × r × N` bytes.
- `r` — block size factor. Memory and bandwidth scale linearly.
- `p` — parallelization factor. Mostly used for hardware-tradeoff exploration, not parallelism per se.

The core is the **SMix** function:

1. Initialize a sequence `V[0..N-1]` where `V[0] = X` (derived from password+salt via PBKDF2-HMAC-SHA-256 and then BlockMix), and `V[i+1] = BlockMix(V[i])`.
2. For `N` iterations: pick `j = integerify(X) mod N`, set `X = BlockMix(X xor V[j])`.

The first phase fills a memory array of size `128 × r × N` bytes. The second phase makes `N` random-access reads into that array. An attacker who tries to skip the array (recompute V[j] on demand) pays `O(N)` time per access × `N` accesses = `O(N^2)` — a quadratic blowup. So the rational attacker stores the full array, paying the memory cost.

OWASP-ish reasonable defaults:

- `N = 2^17 = 131072`, `r = 8`, `p = 1` → 128 × 8 × 131072 = ~134 MiB memory, ~1 second on commodity CPU.
- For interactive: `N = 2^15 = 32768`, `r = 8`, `p = 1` → ~32 MiB, ~250ms.

scrypt was state-of-the-art memory-hard until Argon2 won PHC. It still works fine; it's just less battle-hardened than Argon2id and lacks the side-channel-mitigation hybrid mode. New systems should pick Argon2id; existing scrypt deployments are fine staying put.

The `p` parameter exists because Percival anticipated attackers running scrypt on hardware where they could afford `p` parallel pipelines but each pipeline had limited memory. By choosing `p > 1`, defenders force the attacker to pay `p × memory` total. In practice, `p = 1` is fine.

## PBKDF2

**RFC 2898** (PKCS#5 v2.0, 2000). The simplest of the slow-hash family: just iterate HMAC many times.

```
DK = T_1 || T_2 || ... || T_dkLen/hLen
T_i = F(P, S, c, i)
F(P, S, c, i) = U_1 xor U_2 xor ... xor U_c
U_1 = HMAC(P, S || INT(i))
U_j = HMAC(P, U_{j-1})
```

Where `c` is the iteration count, `P` is the password, `S` is the salt, `i` is the block index, and `dkLen` is the desired output length.

OWASP 2024 minimums:

- HMAC-SHA-256: 600,000 iterations (older guidance) → 1,300,000 (2024 update)
- HMAC-SHA-512: 210,000 iterations
- HMAC-SHA-1: 1,300,000 iterations (allowed for legacy compat only; SHA-1 collision-broken but PBKDF2 doesn't depend on collision resistance)

PBKDF2's problem is **GPU friendliness**. HMAC-SHA-256 is a few hundred ALU ops per iteration with tiny memory footprint. A modern GPU can run hundreds of thousands of parallel HMAC streams, each at ~10^7 hashes/sec, for 10^11+ guesses/sec aggregate. ASICs do even better. Memory-hard alternatives push this floor down by 4–5 orders of magnitude.

PBKDF2 remains acceptable when:

- FIPS 140 compliance forces an approved KDF and your toolchain doesn't ship Argon2.
- You're upgrading legacy systems and can re-hash on next login (store both, replace on success).
- You're using it for *key derivation* (encrypting blobs at rest), not password verification, where the cost is paid once per session.

For new password verification: Argon2id > scrypt > bcrypt > PBKDF2. PBKDF2 is the bottom of the recommended set, not because it's broken, but because it doesn't make GPU attacks expensive.

## Pepper Theory

A **pepper** is a server-side global secret added to passwords before hashing, kept *outside* the user database. Threat model: attacker exfiltrates the password DB (e.g., SQL injection) but not the application server's secrets (e.g., environment variables, HSM keys). Without the pepper, every offline crack attempt fails because the attacker can't compute the right HMAC input.

Canonical pattern:

```
intermediate = HMAC-SHA-256(pepper, password)
verifier = Argon2id(salt=user_salt, password=intermediate, params=...)
```

The HMAC step has constant work (~microseconds), so it doesn't slow the legitimate flow. But it adds an unknown 256-bit key to the attacker's computation. If the attacker doesn't have the pepper, they cannot brute force at any cost.

Where to store the pepper:

- **HSM (Hardware Security Module)** — canonical placement. The HSM exposes only an HMAC operation; the key never leaves the device. SQL injection cannot reach an HSM.
- **Environment variable in the app process** — easier, defends against DB-only breach but not server-RCE.
- **AWS KMS / GCP KMS** — `Encrypt`/`Decrypt` operations, the key material is in the cloud HSM. Uses an IAM boundary instead of physical isolation.

Rotation: peppers should rotate occasionally. The trick is you can't re-pepper existing hashes without the original password. Pattern: include a `pepper_version` in the stored verifier; on next successful login, re-hash with the new pepper and update.

The pepper does **not** replace per-user salt. Salt prevents rainbow-table attacks across users; pepper prevents brute force across the table. Both are required.

## Password Strength Math

Password entropy is the Shannon entropy of the *generation distribution*, not of the resulting string. A truly random password chosen uniformly from an alphabet of size `A` and length `L` has entropy:

```
H = log2(A^L) = L × log2(A)
```

Common alphabets:

| Alphabet | Size | log2(size) |
|----------|------|------------|
| digits | 10 | 3.32 |
| lowercase | 26 | 4.70 |
| alphanumeric | 62 | 5.95 |
| ASCII printable | 95 | 6.57 |
| Diceware (EFF list) | 7776 | 12.92 (per word) |

Examples:

- 8 random ASCII printable: `8 × 6.57 = 52.6 bits`
- 12 random ASCII printable: `12 × 6.57 = 78.8 bits`
- 16 random ASCII printable: `105 bits`
- 4 Diceware words: `4 × 12.92 = 51.7 bits`
- 6 Diceware words: `77.5 bits`
- 8 Diceware words: `103 bits`

Now the attacker math. Suppose the attacker has GPU resources costing $0.50/hour delivering 10^12 unsalted MD5 guesses/sec (or 10^7 Argon2id-default guesses/sec):

| Scheme | Hashes/sec | Time to crack 53-bit | Time to crack 80-bit |
|--------|-----------|----------------------|----------------------|
| Unsalted SHA-256 | 10^11 | 25 hours | 380 years |
| bcrypt cost-12 | 10^4 | 30,000 years | 4 × 10^12 years |
| Argon2id (19 MiB, 2t) | 10^4 | 30,000 years | 4 × 10^12 years |

The "53 bits = 1.4 hours" claim from the prompt assumes unsalted MD5 / fast hash + 10^12 guesses/sec. On a properly-configured Argon2id, even 53 bits resists practical brute force. The slow hash is doing 10^7 work units per attacker-hour.

**Length matters more than complexity** because:

1. Each character of length adds `log2(A)` bits regardless of `A`. Each "complexity rule" (must have uppercase, must have digit, etc.) restricts the alphabet *but rarely increases the chosen alphabet by users*.
2. Empirical studies (NIST 800-63B, Carnegie Mellon studies) show users respond to "must have uppercase + digit + symbol" by appending `1!` or capitalizing the first letter, both predictable transforms with negligible entropy gain.
3. A 16-char passphrase with no rules has more entropy than a 10-char with all the rules.

The right user-facing guidance: minimum 12 chars, no upper-bound below 64, no composition rules, blocklist breached passwords.

## Why NIST Removed Composition Rules (NIST 800-63B)

NIST SP 800-63B (2017, updated 2024) deliberately *removed* the historic composition requirements (must have uppercase, must have digit, etc.) and the periodic-rotation requirement.

Two empirical findings drove this:

1. **Composition rules don't add entropy.** Studies (e.g., Komanduri 2011, "Of Passwords and People") showed users defeat the rules with predictable transforms. `password` → `Password1!` adds ~5 bits of entropy in theory but ~0 in practice because attackers' wordlists already include those transforms.
2. **Periodic rotation hurts security.** When forced to change passwords every 90 days, users pick `Spring2024!` → `Summer2024!` → `Autumn2024!`. The *new* password is trivially derivable from the *old*, so any breach exposes the entire sequence.

The replacement guidance:

- Length minimum 8 (NIST), 12 (OWASP).
- No composition rules required.
- No periodic rotation unless evidence of compromise.
- Check candidate passwords against a breached-password list (HIBP, Pwned Passwords).
- Allow all printable ASCII + Unicode + spaces.
- No upper limit below 64 characters.
- No password hints, no security questions.

Breached-list checking is dramatically more effective than rules. The top-1M breached passwords cover ~50% of all human-chosen passwords. Blocking that list in the registration flow is a single check that rejects half the bad choices users would otherwise make.

## Have-I-Been-Pwned API

Troy Hunt's HIBP service exposes a *k-anonymity* API for the Pwned Passwords corpus (~600M unique passwords from breach dumps).

Naive design: client sends password to HIBP, server returns "leaked" or not. Privacy disaster — every login form just sent your password to a third party.

K-anonymity design:

1. Client computes `h = SHA-1(password)`, takes the first 5 hex characters as the *prefix*.
2. Client requests `GET https://api.pwnedpasswords.com/range/<prefix>`.
3. Server returns ~500 entries: `<rest_of_hash>:<count>` lines for every breached password sharing that prefix.
4. Client searches for `h[5:]` in the response. Match → password is in the breach corpus.

Privacy property: the server learns only the 20-bit prefix. Any prefix matches ~1/2^20 of the corpus, ~500 hashes, so the server can't tell *which* password the client was checking. The client never sends the rest of the hash.

SHA-1 is not chosen for cryptographic strength here — it's just a stable bucket function. Collision attacks on SHA-1 don't matter because HIBP uses it as a partition.

In code, the entire flow is one HTTP GET and a string search. There's no excuse not to integrate it into registration and password-change flows.

## Session Cookie Theory

HTTP is stateless. After authentication, the server needs a way to recognize subsequent requests as belonging to the authenticated user. The dominant mechanism is the **session cookie**: a server-set token the browser auto-attaches to every request to the origin.

The cookie attribute set is critical. A misconfigured cookie loses entire authentication guarantees.

```
Set-Cookie: __Host-session=eyJhbGc...; Path=/; Secure; HttpOnly; SameSite=Lax; Max-Age=3600
```

Each attribute:

- **HttpOnly** — JavaScript cannot read the cookie via `document.cookie`. Defends against XSS theft. An attacker who runs script in your origin can still issue authenticated requests *from* the page (because the browser auto-attaches the cookie), but cannot exfiltrate the token to their server. Defense-in-depth, not a complete XSS defense — assume XSS = full compromise of the session, but HttpOnly prevents the token from being trivially stolen and replayed elsewhere.
- **Secure** — cookie only sent over HTTPS. Without this, an attacker on the network can read the cookie on any plaintext HTTP request to your domain (even subdomains, even `/favicon.ico`). Always set this in production.
- **SameSite** — controls cross-site request behavior:
  - `Strict` — cookie *only* sent on same-site navigations. Even clicking a link from `gmail.com` to your site will *not* send the cookie. Maximum CSRF protection, but breaks "logged in across tabs" and external referrals.
  - `Lax` — cookie sent on top-level GET navigations from other sites, but not on cross-site POST/iframe/fetch. Modern browser default since 2020.
  - `None` — cookie sent on all cross-site requests. Required for embeds (e.g., your widget on a third-party site). Must combine with `Secure`.
- **Path** — cookie only sent on requests where the URL path starts with the configured value. Limits scope inside one origin. `Path=/` is the common default.
- **Domain** — explicit domain scope. *Omitting* this restricts to the current host. Setting `Domain=example.com` extends to all subdomains. Subdomain creep is a footgun: a vulnerable subdomain can read or set cookies for the whole apex.
- **Max-Age** / **Expires** — lifetime. Without one, cookie is "session" (lasts until browser close). With one, the browser persists it across restarts. Authentication cookies typically use `Max-Age` to bound idle session length.
- **`__Secure-` prefix** — cookie name starting with `__Secure-` is only accepted by the browser if it has the `Secure` attribute. A misconfigured backend that forgets `Secure` will fail to set the cookie, surfacing the bug instead of silently exposing it.
- **`__Host-` prefix** — stricter: requires `Secure`, `Path=/`, and *no* `Domain` attribute. The cookie is bound to *exactly* this hostname, with no subdomain leakage. Recommended for new auth cookies.

The composition `__Host-session; Path=/; Secure; HttpOnly; SameSite=Lax` is the conservative default for a session cookie in 2024.

## Session ID Generation

The session ID is a *bearer token*: anyone who has it is, to the server, the authenticated user. It must be unguessable.

Requirements:

- **CSPRNG** — cryptographically secure PRNG. On Linux, `/dev/urandom` (Go's `crypto/rand`, Python's `secrets`, Node's `crypto.randomBytes`). Never `Math.random()`, `rand()`, or anything called "fast random."
- **At least 128 bits of entropy.** A 64-bit token can be brute-forced online by a determined attacker (10^11 guesses, plausible at 10^4 attempts/sec for 4 months); 128 bits requires 10^28 guesses, infeasible.
- **No structure that leaks information.** The token should look uniformly random. Don't include user IDs, timestamps, or counters.

Encoding: 16 random bytes → 32 hex chars or 22 base64url chars. Either works. Length isn't a security property; entropy is.

Storage on the server:

- **In-memory store (Redis)** — fast lookup, supports immediate revocation, supports per-session attributes (last-seen IP, role, etc.).
- **Signed cookie** — the cookie *is* the data, signed with HMAC. No server lookup needed. But revocation requires a denylist or a reduced TTL.

Bearer-token property: if you log session IDs in access logs, anyone with read access to the logs has the user's session. Either don't log the cookie value, or hash it before logging (HMAC-SHA-256 with a server secret).

## Session Fixation Attack

Pre-2010s vulnerability that still appears in custom auth.

**Attack:** Attacker sets a known session ID before the victim authenticates. Two common mechanisms:

1. **URL-based session** — `https://example.com/?sessionid=abc123`. Attacker emails this link to victim; victim clicks, logs in, server promotes the same `abc123` session to authenticated. Attacker now uses `abc123` and is logged in as victim.
2. **Cookie-based session with attacker pre-set** — attacker uses a vulnerability (XSS on a sibling subdomain, or just a shared domain attribute) to set the victim's session cookie to a value the attacker knows. Victim authenticates, attacker reuses the cookie.

**Defense:** **Always rotate the session ID at the moment of authentication.** Whatever pre-auth session existed, throw it away. Generate a fresh ID, set it as the cookie, invalidate the old one server-side. The attacker's pre-set ID is now dead.

This is a one-line fix in any modern framework (Django: `request.session.cycle_key()`; Express-session: `req.session.regenerate()`; Rails: `reset_session`). The bug only appears in hand-rolled session managers that don't know about it.

Equally important: rotate on privilege escalation (admin login, role change) and on logout (invalidate, don't just clear).

## JWT Theory

**RFC 7519** (JSON Web Token, 2015). A JWT is three base64url-encoded segments separated by dots:

```
<header>.<payload>.<signature>
```

Header (JSON):

```json
{"alg": "RS256", "typ": "JWT", "kid": "key1"}
```

Payload (JSON):

```json
{"iss": "https://idp.example.com", "sub": "user-123", "aud": "api", "exp": 1714000000, "iat": 1713996400, "jti": "uuid-..."}
```

Signature: `Sign(alg, header_b64 + "." + payload_b64)` using the algorithm specified in the header.

Standard claims (RFC 7519):

- `iss` — issuer (the identity provider URL).
- `sub` — subject (the user identifier within the issuer).
- `aud` — audience (which API/service is allowed to accept this token).
- `exp` — expiration time (Unix seconds).
- `iat` — issued-at time.
- `nbf` — not-before time.
- `jti` — unique token ID, used for replay tracking.

Algorithms:

- `HS256` — HMAC-SHA-256, symmetric. Issuer and verifier share a secret.
- `RS256` — RSA-PKCS1v1_5-SHA-256. Asymmetric. Verifier needs only the public key.
- `ES256` — ECDSA-P-256-SHA-256. Asymmetric, smaller keys/sigs.
- `EdDSA` — Ed25519. Modern, deterministic, fast.

**Historical bugs:**

- **`alg=none`** — early JWT libraries respected an `alg: "none"` header that meant "no signature." A token with `{"alg":"none"}` and an empty signature would be accepted as valid. Fix: hardcode the expected algorithm in the verifier; never read `alg` from the token to choose verification.
- **Algorithm-confusion attack** — server is configured to accept RS256, holds the public key. Attacker crafts a token with `{"alg":"HS256"}`, signs it with the server's *public key as the HMAC secret*, sends it. A naive verifier reads `alg=HS256`, looks up the configured key (the public one), runs HMAC-SHA-256, validates. The defender exposed their public key (publicly), the attacker forged a valid HS256 signature. Fix: pin the algorithm at verification time, never accept whatever the token claims.

**Verification recipe (pseudocode):**

```
fn verify_jwt(token, expected_alg, key, expected_iss, expected_aud):
    parts = token.split(".")
    assert len(parts) == 3
    header = json_parse(b64url_decode(parts[0]))
    assert header["alg"] == expected_alg
    payload = json_parse(b64url_decode(parts[1]))
    sig = b64url_decode(parts[2])
    assert verify(expected_alg, key, parts[0] + "." + parts[1], sig)
    assert payload["iss"] == expected_iss
    assert payload["aud"] == expected_aud
    assert now() < payload["exp"]
    assert now() >= payload.get("nbf", 0)
    return payload
```

Every assertion is a security property. Skipping `aud` lets a token issued for service A be replayed against service B. Skipping `exp` lets stolen tokens live forever.

## JWT vs Session

The choice between server-session-cookie and JWT is a *stateful vs stateless* trade-off.

**Server-session (cookie + Redis):**

- Pro: Immediate revocation. Delete the session row, the user is logged out instantly.
- Pro: Tiny token (16 random bytes). Bandwidth-light.
- Pro: Server can attach arbitrary attributes (last_active, role_at_login, ip_pinning) without bloating the token.
- Con: Every request hits the session store. Adds a dependency and latency.
- Con: Cross-service auth requires shared session storage.

**JWT (signed token):**

- Pro: Stateless. Verifier needs only the issuer's public key. No DB hit.
- Pro: Cross-service: any service that trusts the issuer can verify the token.
- Pro: Self-contained: claims travel with the token.
- Con: Cannot revoke before `exp`. The user "logs out" but a stolen token still works until expiry.
- Con: Larger (typically 500–2000 bytes).
- Con: Re-signing on role change requires a fresh token issue + waiting for old one to expire, or a denylist.

The hybrid pattern: JWT access tokens (short TTL, 15 min) + opaque refresh tokens (server-stored). The access token is stateless and fast; revocation works because access tokens expire quickly and refresh tokens (which can be revoked) gate the issuance of new access tokens. We'll examine refresh-token rotation below.

For first-party web apps (your-domain.com calling your-api.your-domain.com), session cookies usually win on simplicity and revocation. For multi-service architectures or third-party API access, JWT is the better fit. Pick deliberately.

## JWT Storage Trade-offs

Where does the browser keep the JWT?

**localStorage:**

- Persists across tabs and browser restarts.
- Readable by any JavaScript on the origin → trivially exfiltrated by XSS.
- Survives the user closing all tabs (might not be what you want).
- Programmatically attached: every fetch needs `Authorization: Bearer <token>` manually.
- Not auto-sent on cross-origin requests (good, no CSRF).

**sessionStorage:**

- Per-tab, gone when tab closes.
- Same XSS vulnerability as localStorage.
- Awkward UX: opening a new tab requires re-login.

**HttpOnly cookie:**

- Browser-managed; JavaScript cannot read.
- Auto-attached on requests to the origin (including cross-origin if `SameSite=None`).
- XSS-safe for token theft (script can't read it).
- CSRF-prone if the app uses fetch with credentials and doesn't have CSRF tokens.
- Can use `__Host-` prefix for hardening.

The 2024 consensus for first-party web apps:

- HttpOnly + Secure + SameSite=Lax cookie holding the access token (or session ID).
- CSRF token (synchronizer or double-submit) for state-changing requests.
- Refresh token also in HttpOnly cookie, with a separate path or different cookie.

For SPAs talking to APIs on different origins where you cannot set cookies for the API origin, you're forced into Authorization headers + localStorage, with the XSS exposure that implies. Mitigations: strict CSP, no third-party scripts, regular CSP audits, short token TTLs.

## Refresh Token Rotation

The standard short-access + long-refresh pattern:

- **Access token** — JWT, ~15 minute TTL. Carried in every API request.
- **Refresh token** — opaque random string, ~7 day TTL. Used only to mint new access tokens.

Flow:

```
POST /token (with refresh_token=R1)
→ server validates R1
→ server issues access_token AT2 + new refresh_token R2
→ server invalidates R1
```

Each refresh consumes one refresh token and produces the next. R1 can never be used again.

**Replay detection:** if the server sees R1 used a *second* time after it was rotated to R2, this is a strong signal that R1 was stolen. The user might have R2 (legit) and the attacker has R1 (stolen). Defense: invalidate the entire token family — both R1 and R2, plus all access tokens issued from them. Force re-authentication.

This is RFC 6749 OAuth 2.0 + RFC 6819 + the OAuth 2.1 draft consolidating the rotation and replay-detection guidance.

Token-family tracking can be implemented as a chain ID stored alongside each refresh token; when one is reused after rotation, walk the chain and revoke everything.

Refresh tokens should be:

- Stored as a hash (HMAC-SHA-256 with a server secret, plus a per-token salt) in the database, not plaintext.
- Marked with a timestamp, IP, and user-agent for forensics.
- Bound to a client_id (the OAuth client that minted them) — prevents token leakage across apps.

## PKCE

**RFC 7636** (Proof Key for Code Exchange, 2015). Designed to fix a specific OAuth weakness for *public clients* (mobile apps, SPAs) that cannot keep a client_secret confidential.

The classic OAuth Authorization Code flow:

1. Client redirects to authorization server with `client_id` + `redirect_uri`.
2. User authenticates, authorization server redirects back to `redirect_uri` with `?code=ABC`.
3. Client exchanges the code at `/token` endpoint with `client_id` + `client_secret`.

For mobile/SPA clients, the `client_secret` is shipped in the binary or downloaded JS, so it's not actually secret. An attacker who intercepts the redirect (say, a malicious app registering the same `myapp://` scheme) gets the code and can exchange it.

PKCE adds a per-flow secret known only to the client instance.

1. Client generates `code_verifier` — 43–128 random base64url chars (≥256 bits of entropy).
2. Client computes `code_challenge = base64url(SHA-256(code_verifier))`.
3. Client redirects with `?code_challenge=<C>&code_challenge_method=S256`.
4. Authorization server stores `code_challenge` against the issued `code`.
5. After redirect-back, client exchanges code with `?code_verifier=<V>`.
6. Server checks `SHA-256(V) == stored C`. Mismatch → reject.

The `code_verifier` never leaves the client until the token-exchange step, which is HTTPS to the authorization server. An attacker who intercepts the redirect (with the code) does not have `code_verifier` and cannot complete the exchange.

OAuth 2.1 (draft) makes PKCE *mandatory for all clients*, public and confidential alike. There's no good reason not to use it, and there are still subtle attacks (mix-up attacks, etc.) that PKCE closes for confidential clients too.

## OAuth 2.1 Authorization Code Flow

OAuth 2.1 is the in-progress consolidation of OAuth 2.0 + the various BCPs and corrections. The Authorization Code Flow with PKCE is the canonical flow for any client:

```
[1] Client → AS: GET /authorize?
        response_type=code
       &client_id=APP
       &redirect_uri=https://app.example.com/cb
       &scope=openid profile
       &state=<csrf-token>
       &code_challenge=<C>
       &code_challenge_method=S256

[2] AS authenticates user, requests consent.

[3] AS → Client: 302 to redirect_uri?code=AUTH_CODE&state=<csrf-token>
        Client validates state matches.

[4] Client → AS: POST /token
        grant_type=authorization_code
       &code=AUTH_CODE
       &redirect_uri=https://app.example.com/cb
       &client_id=APP
       &code_verifier=<V>

[5] AS → Client: {access_token, refresh_token, id_token, token_type: "Bearer", expires_in: 900}
```

Key properties:

- **Code is one-time use, short TTL.** Authorization servers MUST reject second use of an authorization code, and codes typically expire in 60–600 seconds.
- **Redirect URI exact match.** AS must verify the `redirect_uri` in the token request matches the registration AND the one provided in the authorize request.
- **State parameter** — opaque random value the client generates per flow. AS echoes it back on redirect. Client validates. Defeats CSRF on the redirect step.
- **Scope parameter** — what the client is allowed to do. Server displays this in consent screen.

Anti-patterns OAuth 2.1 *forbids*:

- The Implicit Flow (`response_type=token`) — token returned in URL fragment, no PKCE, leaked everywhere. Banned.
- Resource Owner Password Credentials grant — client collects username/password, sends to AS. Defeats the entire point of OAuth (delegated authentication). Banned.

## OIDC ID Token

**OpenID Connect** is OAuth 2.0 plus an identity layer. The key addition: the **ID Token**, a JWT signed by the IdP that asserts the user's identity to the *client*.

When the client's `scope` includes `openid`, the token response includes:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "id_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 900
}
```

The `id_token` payload (after JWT decode):

```json
{
  "iss": "https://idp.example.com",
  "sub": "user-uuid-12345",
  "aud": "client-app-id",
  "exp": 1714000000,
  "iat": 1713996400,
  "nonce": "<from-authorize-request>",
  "auth_time": 1713996300,
  "email": "alice@example.com",
  "email_verified": true,
  "name": "Alice Example"
}
```

The crucial distinction:

- **`id_token` is for the client.** The client verifies it (using the IdP's public key from `/.well-known/jwks.json`) to learn *who logged in*. The client should never forward an ID token to a backend API expecting it to be authenticated by it.
- **`access_token` is for the API.** The client attaches it to API requests; the API verifies it (often opaque introspection, or as a JWT) to authorize the call.

Confusing the two leads to *audience confusion* attacks: an ID token for client A, sent to API B, with API B failing to check `aud`, accepted as authorization. The mitigation is the standard JWT verification recipe — always check `aud`.

The `nonce` claim ties the ID token back to a specific authorization request. The client generates a random nonce, sends it in `/authorize`, the IdP includes it in the ID token, the client verifies the match. Defeats replay of ID tokens captured from another flow.

## WebAuthn / Passkeys

**W3C Web Authentication** specification, paired with **CTAP2** (Client to Authenticator Protocol) from FIDO Alliance. Standardized by WebAuthn Level 2 (2021), Level 3 in progress. Real-world deployment as "passkeys" (a UX-friendly branding for resident, syncable WebAuthn credentials).

The model: each user has a **public-key credential** per *origin*, generated and held by an *authenticator* (Touch ID, Face ID, Windows Hello, YubiKey, Android keystore). The browser is a relayed messenger between the relying party (your website) and the authenticator.

### Registration ceremony

1. Server generates a random `challenge` (≥16 bytes).
2. Server sends `PublicKeyCredentialCreationOptions` to the browser:
   - `challenge`
   - `rp` (relying party): `{id: "example.com", name: "Example"}`
   - `user`: `{id, name, displayName}`
   - `pubKeyCredParams`: `[{type: "public-key", alg: -7}]` (ES256, for example)
   - `authenticatorSelection`: `{authenticatorAttachment: "platform", userVerification: "required", residentKey: "required"}`
   - `attestation`: `"direct"` or `"none"`
3. Browser calls `navigator.credentials.create({publicKey: ...})`, which forwards to the authenticator.
4. Authenticator generates a fresh keypair scoped to this `(rp.id, user.id)` pair, prompts the user to verify (PIN/biometric/touch).
5. Authenticator returns:
   - `attestationObject` — CBOR-encoded structure containing the public key, attestation statement, and authenticator data.
   - `clientDataJSON` — JSON with the challenge, origin, and "type":"webauthn.create".
6. Server verifies:
   - `clientDataJSON.challenge` matches the issued challenge.
   - `clientDataJSON.origin` matches the expected origin.
   - `clientDataJSON.type == "webauthn.create"`.
   - Attestation signature (if `attestation != "none"`) traces to a known trust anchor.
   - User-verification flag (UV) is set in authenticatorData (if required).
7. Server stores the public key and credential ID against the user.

### Assertion ceremony (login)

1. Server generates a random `challenge`.
2. Server sends `PublicKeyCredentialRequestOptions`:
   - `challenge`
   - `rpId: "example.com"`
   - `allowCredentials: [{id: <credential-id>, type: "public-key"}]` (optional with discoverable credentials)
   - `userVerification: "required"`
3. Browser calls `navigator.credentials.get({publicKey: ...})`.
4. Authenticator finds the credential, verifies the user, signs `(authenticatorData || SHA-256(clientDataJSON))` with the private key.
5. Browser returns the assertion to the server.
6. Server verifies the signature using the stored public key, and verifies challenge, origin, type, UV flag.

### Flags in authenticatorData

- **UP (User Presence)** — bit 0. The user touched the authenticator.
- **UV (User Verification)** — bit 2. The user proved identity to the authenticator (PIN/biometric).
- **AT (Attested credential data)** — bit 6. Credential data is included (registration only).
- **ED (Extension data)** — bit 7. Extensions are included.

The `signCount` in authenticatorData is a counter the authenticator increments per assertion. Servers compare it across logins; a counter going *backwards* indicates the credential may be cloned. Modern syncable passkeys often use 0 for `signCount` (skipping the check), which is a deliberate trade-off for sync ergonomics.

## WebAuthn Phishing Resistance

The cryptographic property that makes WebAuthn unphishable:

The signed payload includes `clientDataJSON.origin`, which is the origin *as seen by the browser at the moment of the assertion*. The server verifies this origin matches the expected one.

Concretely: an attacker phishing site `examp1e.com` can serve a login form, capture credentials (passwords, TOTP codes), and replay them to `example.com`. With WebAuthn, the attacker's site has origin `examp1e.com`. When the browser triggers `navigator.credentials.get`, it includes the *current page's origin* in the clientDataJSON. The authenticator signs it. The attacker forwards the signed assertion to `example.com`. The legitimate server checks `clientDataJSON.origin == "https://example.com"` — fails — rejects.

The attacker cannot tamper with `clientDataJSON.origin` because the signature covers it. The attacker cannot get the authenticator to sign a different origin because the browser refuses to lie about the origin to the authenticator (origins are enforced at the WebAuthn API surface).

Even if the attacker proxies the entire site (Evilginx-style), the signed origin is *the proxy's origin*, not the legit one. WebAuthn turns origin into a cryptographic primitive.

The key system requirements: (1) the relying party verifies origin, (2) the browser-authenticator path is integrity-protected (it is, because the authenticator's CTAP2 binding ties to the platform's WebAuthn implementation), (3) the user does not deliberately register a credential on the phishing site (which is just normal phishing of a fresh credential, not a way to compromise an existing one).

## TOTP

**RFC 6238** (Time-Based One-Time Password, 2011), built on **RFC 4226** (HOTP). The math is simple:

```
T = floor((now - T0) / period)
HOTP(K, T) = truncate(HMAC-SHA-1(K, T))
TOTP = HOTP(K, T)
```

Where:

- `K` — shared secret (typically 20 bytes, base32-encoded as ~32 chars).
- `T0` — Unix epoch (almost always 0).
- `period` — 30 seconds (default).
- `now` — current Unix time.

The truncate function:

```
offset = HMAC[19] & 0x0F
P = (HMAC[offset]   & 0x7F) << 24
  | (HMAC[offset+1] & 0xFF) << 16
  | (HMAC[offset+2] & 0xFF) << 8
  | (HMAC[offset+3] & 0xFF)
TOTP = P mod 10^digits   // typically digits = 6
```

6 digits = ~20 bits per attempt (`log10(10^6) / log10(2) ≈ 19.93`). 30-second window = a successful brute-force needs the right 6 digits within a 30-second window. At 1 attempt/sec the success probability per window is `30 / 10^6 = 3 × 10^{-5}`. Combined with rate-limiting (lockout after 5 wrong codes), TOTP is solid against online brute force.

The QR-code provisioning URI:

```
otpauth://totp/Example:alice@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Example&period=30&algorithm=SHA1&digits=6
```

Apps (Google Authenticator, Authy, 1Password) parse this and store the secret. The secret never leaves the user's device after setup.

TOTP weaknesses:

- **Phishable.** A phishing proxy collects the code in real time and replays. Time window is generous.
- **Replayable within a window** unless the server tracks last-used codes.
- **Clock drift.** Most servers accept ±1 step (90 seconds total) to tolerate drift.
- **HMAC-SHA-1.** Not broken in HMAC mode, but newer guidance (RFC 6238 allows SHA-256/512) is to upgrade. Most authenticator apps still default SHA-1 for compat.

TOTP is a strong second factor against credential stuffing and password breaches; it's a weak factor against active phishing. WebAuthn is the upgrade path.

## SMS-MFA Threat Model

SMS as a second factor was the dominant 2FA from ~2010 to ~2020. It's now actively recommended *against* by NIST (SP 800-63B) and OWASP for high-value accounts.

Three attacker techniques:

**SIM swap.** Attacker calls the carrier's customer service, impersonates the victim (often using personal info from data breaches or social media), convinces the agent to issue a new SIM with the victim's number. Victim's phone goes dark; attacker now receives all SMS, including 2FA codes. Reset flows that fall back to "we'll text you a recovery code" fail catastrophically.

**SS7 protocol weaknesses.** SS7 is the global telecom signaling protocol. It was designed in 1975 with no authentication between providers; any provider can request a "redirect SMS for this number" and adjacent providers comply. Researchers (Tobias Engel 2014, Karsten Nohl 2016) demonstrated remote SMS interception via SS7 from anywhere in the world for ~$500 in service fees.

**Port-out fraud.** Variant of SIM swap targeting number-portability flows. Attacker initiates a port-out from the victim's carrier to a carrier the attacker controls. Some carriers' port-out flows are weak (just need the number and a guessable PIN).

For low-stakes accounts (forum login, retail), SMS 2FA still raises the bar above no-2FA. For high-value accounts (banking, email, work IdP, password manager), SMS is below the bar — push-MFA, TOTP-app, or WebAuthn are required.

## Push-MFA + Number Matching

The "tap to approve" notification model from 2018-ish (Duo, Microsoft Authenticator, Okta Verify) replaced SMS for many enterprises. The user receives a notification on a registered phone, sees "Login attempt from <ip>, <location>?", and taps Approve or Deny.

This was an upgrade over SMS — no phone-number redirection attacks — but introduced a new failure: **MFA fatigue (push bombing)**. Attacker who has the password just hammers the login form; the user receives push after push, eventually taps Approve out of annoyance or reflex. The 2022 Uber breach (Lapsus$) used exactly this technique against an Uber contractor.

**Number matching** is the 2023–2024 mitigation now mandated by Microsoft, Okta, Duo, and most major IdPs:

1. Login form displays a 2-digit number (e.g., "27").
2. The push notification displays the same number plus an entry field.
3. User must *type* the number, not just tap.

This breaks reflexive approval. A user who taps Approve out of habit hits a number-entry screen and has to think. A user who is being phished and sees "type 42" but the phishing site shows "27" sees the mismatch.

Number matching adds ~20 bits of entropy per approval (well, ~6.6 bits for a 2-digit number, but the friction is the security feature, not the entropy). It does not defend against a determined real-time MitM proxy that displays the same number to the victim in the phishing UI, but it does defend against the broad class of "user habitually approves all pushes."

## CSRF Theory

Cross-Site Request Forgery exploits the browser's behavior of attaching cookies to *any* request to the cookie's origin, regardless of which site initiated the request.

Classic attack:

```
Victim is logged in to bank.com (session cookie set).
Victim visits attacker-blog.com.
Attacker's page contains:
  <form action="https://bank.com/transfer" method="POST">
    <input name="to" value="attacker-account">
    <input name="amount" value="10000">
  </form>
  <script>document.forms[0].submit()</script>
Browser sends POST to bank.com with the victim's session cookie attached.
bank.com sees an authenticated request, executes the transfer.
```

The attack works because:

1. Cookies are auto-attached to requests to the cookie's origin.
2. Cross-origin form POSTs are allowed by the browser (legacy behavior, predates the same-origin policy of fetch/XHR).
3. The attacker's script is in *attacker-blog.com*'s context, which can construct and submit forms targeting other origins.

**SameSite=Lax** changed this. Since 2020 (Chrome 80, Firefox 96, Safari ~14), the browser default for cookies *without* an explicit SameSite attribute is `Lax`, which means cookies are NOT sent on cross-site POSTs (or iframe loads, or fetch requests). The attack above silently fails because the cookie isn't attached.

So with `SameSite=Lax` (default) or `SameSite=Strict`:

- Cross-site POST: cookie not sent. Attack neutralized.
- Cross-site GET top-level navigation: cookie sent (Lax) or not (Strict). For GETs that don't change state, this is fine.
- Cross-site iframe / fetch / image: cookie not sent in either Lax or Strict.

This made many CSRF attacks impossible without any CSRF-token machinery. But:

- `SameSite=None` sites (cross-origin embeds, federated auth) still need CSRF tokens.
- Older browsers (Safari 12, IE) didn't honor SameSite.
- Method-based attacks via GET still possible if your server accepts state changes via GET (don't!).

CSRF tokens remain a defense-in-depth standard for any state-changing request. The 2024 best practice is *both* SameSite cookies *and* CSRF tokens *and* Origin/Referer checks.

## CSRF Token Pattern

Two implementation patterns dominate.

### Synchronizer Token

Server generates a random token per session (or per form), stores it server-side, embeds it in forms:

```html
<form action="/transfer" method="POST">
  <input type="hidden" name="csrf_token" value="<random-128-bit>">
  ...
</form>
```

On POST, server checks:

```
assert request.form["csrf_token"] == session.csrf_token
```

The attacker cannot read the token (same-origin policy on the page), so cannot forge the POST.

### Double-Submit Cookie

Server sends a CSRF token as both a cookie and an HTML field. On POST, server checks they match.

```
Set-Cookie: csrf=ABC123; Path=/; Secure; SameSite=Lax
<input type="hidden" name="csrf_token" value="ABC123">
```

```
assert request.cookie["csrf"] == request.form["csrf_token"]
```

The attacker's site can set the cookie value (but only on its own domain) and can include any value in the form. But it can't read the legitimate cookie from `bank.com` to put it in the form (same-origin policy on cookies). So the form value doesn't match the legit cookie, and the POST fails.

The double-submit pattern is *stateless* (no server-side session storage of the token), which is sometimes attractive for serverless. But it's slightly more fragile (subdomain takeover can defeat it; requires careful path / domain scoping).

### Origin / Referer Check

Defense-in-depth: server reads `Origin` header (if present) or `Referer` (fallback) and checks it matches an allowed origin.

```
allowed_origins = {"https://example.com"}
origin = request.headers.get("Origin") or extract_origin(request.headers.get("Referer", ""))
assert origin in allowed_origins
```

This is cheap to add and catches cases where SameSite or CSRF tokens are misconfigured. Not sufficient on its own (some CDNs strip Referer; old browsers omit Origin) but adds a layer.

## SAML Authentication Flow

Brief mention; cross-link to `detail/security/saml.md` if it exists.

SAML 2.0 (OASIS, 2005) is the older federation protocol, dominant in enterprise SSO. The Web Browser SSO Profile flow:

1. User accesses Service Provider (SP).
2. SP redirects to Identity Provider (IdP) with `SAMLRequest` (XML, base64+deflate-encoded, optionally signed).
3. User authenticates at IdP.
4. IdP returns an HTML form auto-submitting to SP's ACS URL, with a `SAMLResponse` (XML containing assertions about the user, signed by IdP).
5. SP validates the signature, extracts claims, establishes session.

SAML's pain points:

- **XML Signature canonicalization bugs.** XML signatures over a substring of the document have led to many authentication-bypass CVEs. The classic "XSW" (XML Signature Wrapping) attack reorders elements to fool naive verifiers.
- **XSLT in metadata.** SAML metadata can include XSLT processing instructions, weaponized for code execution in some libraries.
- **Verbose, hard to debug.** XML-encoded, base64-armored, deflate-compressed; debugging requires careful unpacking.

Modern enterprise auth is gradually shifting to OIDC. SAML remains entrenched in legacy IdPs (especially on-prem Active Directory Federation Services) and won't fully disappear for years.

## Federation Trust

Federation (SAML, OIDC) shifts the *authentication* burden from the SP to the IdP. The SP no longer stores passwords; it accepts assertions from a trusted IdP.

The trust model has three pieces:

1. **Metadata exchange.** SP and IdP exchange:
   - Issuer URLs.
   - Public keys (or X.509 certs) for signature verification.
   - Endpoints (authorization, token, logout, ACS).
   - Supported claims and bindings.

2. **Cryptographic root.** SP verifies IdP-signed assertions using the IdP's public key. IdP verifies SP requests via shared secret or signed JWTs.

3. **Operational trust.** SP trusts that IdP performs adequate authentication. SP trusts that IdP correctly identifies the subject. SP trusts that IdP rotates keys in time. None of this is cryptographic; it's contractual.

**Cert rotation challenges:**

- IdP rotates signing key. If SP didn't fetch updated metadata, all IdP assertions fail until SP refreshes.
- SP rotates encryption key. If IdP doesn't refresh, encrypted assertions fail.
- Best practice: publish JWKS / metadata at a stable URL, refresh on a regular interval *or* on signature failure.
- Always support multiple keys simultaneously during rotation: publish key1 + key2, sign with key2, give relying parties time to fetch, then retire key1.

## Magic Link Theory

Passwordless email-based authentication. The user enters their email, the server sends a single-use URL containing a token; clicking the link logs the user in.

```
GET /auth/magic?token=<random-256-bit>
```

The mechanism is *email-channel-as-second-factor*: the user demonstrates control of the email address by clicking the link, which proves ownership.

Critical security properties:

- **Single-use.** Token must be invalidated after first use. Without this, anyone who later finds the link in email history can replay.
- **Short TTL.** 5–15 minutes typical. Long TTLs increase the exposure window.
- **Bound to user-agent or IP.** Defeats forwarded-link attacks: if Bob asks Alice to check his email and forwards the link to her, the server should detect the mismatch and either re-prompt or refuse.
- **High entropy.** ≥128 bits of CSPRNG.
- **HTTPS only.** Token in URL means it appears in browser history, server logs, referer headers — Secure flag and HTTPS reduce some of this.

The trust model:

- Email account compromise → magic-link account takeover. So magic links inherit email security as the security ceiling.
- An attacker who pwns the email account can perform full account recovery on every site that does magic-link or email-based recovery. This is a real risk; password manager + MFA on the email account is a force multiplier.

Magic links are convenient and avoid the "store the password securely" problem, but they're not a security upgrade over a strong password + MFA. They shift, rather than reduce, the attack surface.

## Account Recovery

Account recovery is the back door of every authentication system. An attacker who can recover an account bypasses all the password/MFA cryptography.

The dominant flows:

1. **Email reset.** Server emails a reset link. User clicks, sets new password.
2. **Phone reset.** Server texts a code. User enters code, sets new password.
3. **Security questions.** "What was your first pet's name?" — user answers.
4. **Trusted device + biometric.** "Approve from another device you're already logged in on."
5. **Trusted-IdP recovery.** "Recover via Google/Apple/Microsoft you've previously linked."

**Email is root.** Recovery flows that go through email mean email account compromise is account compromise everywhere. This is why GMail's anti-phishing and recovery hardening is so aggressive — it's the recovery target for ~3 billion downstream accounts.

**Security questions are anti-security.** They're user-friction with negative security: the answers are often:

- Public (mother's maiden name from public records).
- Guessable (favorite color: blue).
- Findable on social media (high school name).
- Predictable (pet name: a few common ones).

NIST SP 800-63B explicitly recommends *against* knowledge-based authentication for recovery. Modern systems don't ship them.

**OAuth-recovery via trusted providers** is the leading alternative for high-value accounts:

- User links Google/Apple/Microsoft at registration.
- On recovery, user authenticates to the trusted IdP, IdP asserts identity, the original site allows recovery.
- This delegates the recovery burden to the IdP's hardened auth (MFA, anomaly detection, well-funded fraud team).

Best practice: *multiple* recovery options that the user pre-configures, no security questions, mandatory delay window before recovery completes (24–48 hours, with email notifications and a "wasn't me" cancel link).

## Account-Linking Trap

A subtle vulnerability in any system that combines password-based signup with SSO.

**Setup:**

- Service example.com supports both password-based signup and "Sign in with Google."
- The system identifies users by email address.

**Attack:**

1. Attacker signs up with email `victim@gmail.com` using a password. They never verify the email (or the system doesn't require verification).
2. Real user (victim) later visits example.com and clicks "Sign in with Google," signing in with `victim@gmail.com`.
3. The naive system finds the existing account with that email and *attaches* the SSO link. Victim now logs in normally.
4. But the attacker's password still works on the same account. Victim's data is silently shared with the attacker.

**Defenses:**

1. **Always verify email at signup.** If verification is required before account creation, the attacker can't claim `victim@gmail.com`.
2. **On SSO with a matching email, require additional proof.** Don't auto-link; show "an account with this email exists, please log in with password to link."
3. **Trust IdP-asserted email only when `email_verified: true`.** Some IdPs return unverified emails; don't auto-link those either.
4. **On link, immediately invalidate any non-IdP credentials** (passwords, tokens) and require the user to re-set them.

This vulnerability has a generic name in the security literature: *pre-account hijacking*. Microsoft's MSRC published a 2022 paper documenting variants across major sites. Audit any system that combines multiple auth mechanisms.

## Audit Logging Schema

Authentication events MUST be logged. The minimum schema for every auth event:

| Field | Type | Notes |
|-------|------|-------|
| `timestamp` | ISO-8601 UTC | Server-clock, not client-claimed |
| `event_type` | enum | login_success, login_failure, mfa_challenge, mfa_success, mfa_failure, logout, password_change, password_reset_request, password_reset_complete, session_revoke, account_locked, etc. |
| `user_id` | string | Internal user UUID (NOT email — email can change) |
| `ip` | string | Source IP, IPv4 or IPv6 |
| `user_agent` | string | Raw User-Agent header (truncated to 1024) |
| `mechanism` | enum | password, totp, sms, push, webauthn, oauth, saml, magic_link, recovery |
| `outcome` | enum | success, failure_bad_password, failure_locked, failure_mfa, failure_disabled, failure_unknown_user |
| `additional_info` | JSON | Mechanism-specific (e.g., {"webauthn_credential_id": "..."}) |
| `request_id` | string | For correlating with HTTP access logs |
| `geoip_country` | string | Optional, derived from IP |

**Never log:**

- Plaintext passwords (even on failure).
- Session tokens, JWTs, or refresh tokens (HMAC-hash them if you need correlation).
- WebAuthn assertions in raw form.
- TOTP codes (logged or not).
- Private keys.

Log retention: typically 90 days for general access, 1 year for security events, indefinite for "account compromise suspected" events. Ensure logs are immutable (append-only) or shipped to a SIEM in real-time so an attacker who pwns the app can't erase tracks.

## Account Enumeration

Account enumeration is a vulnerability class: distinguishing valid usernames/emails from invalid ones via observable behavior.

**Common leaks:**

1. **Login error messages.** "Invalid password" vs "User not found" — directly enumerates valid usernames.
2. **Timing.** Server bcrypt-verifies real users (250ms) vs short-circuits unknown users (5ms) — the response time difference reveals which is which.
3. **Password reset.** "Email sent" vs "Email not found" — same problem.
4. **Registration.** "Username already taken" — reveals registered usernames.
5. **Profile pages.** `GET /users/<username>` returning 200 vs 404.

**Defenses:**

- **Constant message at the login endpoint.** "Invalid email or password," same string for both cases.
- **Constant timing.** When the user is unknown, run a *dummy* password verification against a fixed-but-realistic hash, so the response time is similar.

```python
DUMMY_HASH = argon2.hash("$2024-dummy-password")  # constant

def login(email, password):
    user = db.find_user(email)
    if user is None:
        argon2.verify(DUMMY_HASH, password)  # waste time
        return error("invalid email or password")
    if not argon2.verify(user.password_hash, password):
        return error("invalid email or password")
    return success(user)
```

- **Recovery: same response regardless.** "If an account exists for that email, a reset link has been sent." Send no email if the account doesn't exist; respond identically; the user with no account just sees the message and never receives email.

- **Registration: rate-limit + bot detection.** Don't fix this with timing; you can't avoid telling the user their username is taken. Mitigate with rate limits and bot detection.

Some products deliberately accept enumeration (e.g., Twitter/X) because the friction of "if your email matches" messages is bad UX and the threat model accepts username-knowledge. That's a deliberate choice; for high-security apps, close it.

## Brute Force Defense Math

The threat: attacker submits passwords in sequence against the login endpoint, hoping to hit a valid one.

Without defense, at 100 req/sec to a single endpoint, an attacker covers the top-1000-passwords list in 10 seconds. With password reuse from breaches, even a ~10% hit rate against random user accounts is realistic.

**Per-username exponential backoff:**

```
attempts = 1: no delay
attempts = 2: delay 1s
attempts = 3: delay 2s
attempts = 4: delay 4s
attempts = 5: delay 8s
...
attempts = N: delay 2^(N-2)s, capped
```

After 5 failures in 15 minutes, lock the account for 15 minutes. After more, escalate.

**Per-IP rate limit:** 60 login attempts per minute per IP. Caps password-spraying from a single IP.

**Why per-username, not just per-IP:**

- Attackers use botnets and rotate IPs.
- Per-IP-only lets a botnet of 10,000 IPs attempt 600,000 passwords/minute against one account.
- Per-username caps the attempts on any single account regardless of source.

**Why not just per-username:**

- Attackers spray *one* password against millions of accounts (credential stuffing). Per-username triggers nothing because each account sees just one attempt.
- Per-IP catches the spraying pattern.

You need both.

**The DoS trade-off:** Aggressive lockouts let an attacker DoS legitimate users. "After 3 failures, lock for 24 hours" lets an attacker enter wrong passwords and lock targets out indefinitely. Mitigations:

- Lockouts time out (15 min, not 24h).
- Lockouts notify the user via email, with a "this wasn't me, recover account" link.
- Successful login from a known-good device (cookie or device fingerprint) skips lockout.

**CAPTCHA:** triggered after N failures, defends against automated tools. But CAPTCHAs have severe accessibility issues (visually impaired users, screen readers, automated browsers). Use sparingly; prefer other signals (device reputation, behavioral analysis).

## Privilege Escalation

Two related concerns:

1. **Re-authentication for sensitive operations.** Even after login, certain actions (changing password, adding 2FA device, transferring funds, deleting account) should require re-typing the password (or providing a fresh MFA).
2. **Admin "act as user" features.** Customer support tools that let an admin impersonate a user for debugging.

**Re-auth pattern:**

- Sensitive endpoint checks: "was the user authenticated within the last N minutes (e.g., 5)?"
- If not, redirect to a re-auth page; user provides password (or MFA) again.
- On success, mark the session as "reauthenticated" with a fresh timestamp.

This bounds the damage if a session token is briefly stolen: the attacker can read but cannot escalate to changing the password.

**Admin act-as:**

- Admin authenticates with their own credentials + MFA.
- Admin requests "view as user X."
- System creates a *separate* session, scoped as user X, with explicit indication ("ACTING AS USER X — admin action").
- All actions taken under act-as are logged with both the admin's identity AND the target user's identity.
- Sensitive operations (changing user's password, sending email, deleting account) are *forbidden* in act-as mode; admin must take the action explicitly via admin tooling, which audit-logs differently.
- Act-as session has a tight TTL (5–30 min).
- A banner is always visible to the admin, preventing accidental confusion about whose data they're acting on.

The audit log entries are critical:

```json
{"event": "act_as_start", "admin_id": "A1", "target_user_id": "U2", "reason": "support ticket #1234"}
{"event": "act_as_action", "admin_id": "A1", "target_user_id": "U2", "action": "viewed_orders"}
{"event": "act_as_end", "admin_id": "A1", "target_user_id": "U2", "duration_seconds": 600}
```

Compliance regimes (SOC 2, ISO 27001, PCI-DSS) typically require this level of admin-action visibility.

## Cryptographic Theory

**Properties of a good password hash:**

- **Slow.** Wall-clock time per evaluation is high enough to make brute force expensive. Tunable upward as hardware improves.
- **Memory-hard.** Each evaluation requires significant RAM (10s of MiB). Defeats GPU/ASIC parallelism, which is bandwidth-bound, not compute-bound.
- **Salt-mandatory.** Per-user random salt (≥16 bytes). Defeats rainbow tables and prevents identical passwords producing identical hashes.
- **Parallel-resistant.** No way to evaluate the function with less work by parallelizing internally.
- **Versioned output.** The stored hash includes algorithm + parameters, so the verifier can identify what scheme + cost was used. Enables seamless migration: on each successful login, re-hash with current parameters if the stored ones are below threshold.

**Properties of a good MAC:**

- **HMAC-SHA-256 minimum.** SHA-1 HMAC isn't broken (HMAC doesn't depend on collision resistance), but new code uses SHA-256 or larger.
- **Constant-time verification.** Comparing two MAC values byte-by-byte and short-circuiting on first mismatch leaks the position of the first differing byte via timing. Use `hmac.compare_digest`-style functions.
- **Key length ≥ 256 bits.** Random, from CSPRNG. Long enough to resist exhaustive search.
- **Domain separation.** Reusing a MAC key for multiple purposes (authenticating cookies AND CSRF tokens AND signed URLs) creates cross-protocol attack surface. Use HKDF-derived per-purpose keys, or distinct keys.

**Properties of a good random source:**

- CSPRNG (e.g., `/dev/urandom`, `crypto.randomBytes`, `secrets.token_bytes`).
- Never `Math.random()`, `rand()`, `mt_rand()` — these are PRNGs designed for simulation, not security. Their state is recoverable from a few outputs.
- Don't seed with predictable values (timestamp, PID).
- Reseed on fork (long-running daemons that fork need to reinitialize the RNG state in children).

## Constant-Time Comparison

The classic timing-attack threat model:

```python
def compare(a: bytes, b: bytes) -> bool:
    if len(a) != len(b):
        return False
    for i in range(len(a)):
        if a[i] != b[i]:
            return False
    return True
```

This function returns *faster* when the mismatch is at index 0 than at index 5. Over many calls, an attacker can measure the response time and *infer the prefix* of the secret one byte at a time.

In practice, the timing differences are tiny (nanoseconds) and usually swamped by network jitter. But for attackers in the same data center, with high-bandwidth low-jitter connections, or for attackers measuring *many* requests to extract statistical signal, the attack works. Coppersmith and Bernstein have demonstrated practical timing attacks on RSA private keys via remote timing.

The fix: a constant-time comparison that does the same work regardless of input.

```python
def compare_ct(a: bytes, b: bytes) -> bool:
    if len(a) != len(b):
        return False
    result = 0
    for x, y in zip(a, b):
        result |= x ^ y
    return result == 0
```

Every byte is examined; mismatches don't short-circuit. The total work is identical for all inputs of the same length.

Standard libraries provide:

- **Python:** `hmac.compare_digest(a, b)`
- **Go:** `subtle.ConstantTimeCompare([]byte, []byte)`
- **Java:** `MessageDigest.isEqual(byte[], byte[])` (since Java 6 update 17)
- **JavaScript (Node):** `crypto.timingSafeEqual(Buffer, Buffer)`
- **Rust:** `subtle::ConstantTimeEq`

Use them for:

- HMAC verification.
- Comparing CSRF tokens.
- Comparing session IDs (if doing manual lookup).
- Comparing API keys.
- Comparing reset tokens.

Anywhere a secret is on the right side of an `==` against user input, the comparison should be constant-time.

The same principle applies inside cryptographic operations. RSA-PKCS1v1_5 padding-oracle attacks (Bleichenbacher) exploit non-constant-time padding validation; the modern fix is to do the entire decryption + padding check in constant time, regardless of validity. Modern AEAD constructions (AES-GCM, ChaCha20-Poly1305) are designed to be naturally constant-time.

## Argon2id Parameter Math: Concrete Attacker Cost

OWASP 2024 baseline: tCost=2, mCost=19MiB, parallelism=1. Wall-clock target: ~50ms.

Attacker cost analysis at this baseline:

```text
Single GPU (NVIDIA RTX 4090, 24GB VRAM):
  Memory pool: 24 GB / 19 MiB = ~1290 concurrent Argon2id operations
  Per-operation latency: ~80ms (CPU) but ~120ms on GPU due to memory-bandwidth ceiling
  Throughput: ~1290 / 0.12s = ~10,750 hashes/sec on a single high-end GPU

  At $1/hour cloud GPU:
    Cost per hash: $1 / 3600 / 10750 = 2.6e-8 dollars
    Cost per 10^10 guesses (full English-word + suffix space): 10^10 * 2.6e-8 = $258
    Cost per 10^14 guesses (full lowercase 8-char): $2.58 million

Bumping to mCost=64 MiB:
  Concurrent ops on 24GB GPU: 24/64 * 1024 = 384 concurrent
  Throughput: ~384 / 0.12s = ~3200 hashes/sec
  Cost per 10^10 guesses: $25,800 — 100x worse for attacker
  Cost per 10^14 guesses: $258 million — practically intractable

Bumping to tCost=4 + mCost=128 MiB:
  Memory: 24/128*1024 = 192 concurrent
  Wall-clock: ~250ms
  Throughput: ~192/0.25 = 770 hashes/sec
  Cost per 10^14 guesses: ~$1 billion
```

The takeaway: **memory cost dominates GPU economics**. Doubling mCost roughly halves attacker throughput because GPU VRAM is the bottleneck, not compute.

For service-accounts where you control hardware, mCost=512MiB or even 1GiB is feasible; for user-facing login where 50ms is the budget, tCost=2/mCost=19MiB is the floor.

## HIBP API Protocol Walkthrough

The k-anonymity scheme works as follows:

```python
import hashlib, requests

def hibp_check(password):
    # Step 1: SHA-1 hash the password (on client side)
    sha1 = hashlib.sha1(password.encode("utf-8")).hexdigest().upper()
    prefix, suffix = sha1[:5], sha1[5:]

    # Step 2: Send only the first 5 hex chars (20 bits) to HIBP
    # The server returns ALL hashes that share this prefix (~500 entries on average)
    response = requests.get(f"https://api.pwnedpasswords.com/range/{prefix}")

    # Step 3: Search the returned list locally for the rest of the SHA-1
    for line in response.text.splitlines():
        h, count = line.strip().split(":")
        if h == suffix:
            return int(count)  # password seen this many times in breaches
    return 0

# Privacy property: HIBP server learns the 20-bit prefix only.
# Even with the full 65,536 prefix space, the server can't determine
# which exact password the user is checking — only that it's one of
# ~30 trillion / 65,536 ≈ 460 million possible 160-bit suffixes per prefix.
# In practice, the server sees ~500 candidate hashes per request.
```

The bandwidth cost: ~30KB per check (500 entries × 60 bytes each). HIBP serves billions of these checks per month and is hosted on Cloudflare for caching.

## WebAuthn Ceremony Detailed Walk

### Registration ceremony — server's challenge generation

```python
import secrets, base64

def make_registration_options(user_id, user_email, user_display_name):
    challenge = secrets.token_bytes(32)
    return {
        "challenge": base64.urlsafe_b64encode(challenge).rstrip(b"=").decode(),
        "rp": {"name": "Example App", "id": "app.example.com"},
        "user": {
            "id": base64.urlsafe_b64encode(user_id.bytes).rstrip(b"=").decode(),
            "name": user_email,
            "displayName": user_display_name,
        },
        "pubKeyCredParams": [
            {"type": "public-key", "alg": -7},   # ES256 (preferred)
            {"type": "public-key", "alg": -257}, # RS256 (fallback)
        ],
        "timeout": 60000,
        "attestation": "none",   # or "indirect"/"direct" if you need attestation cert
        "authenticatorSelection": {
            "authenticatorAttachment": "platform",  # or "cross-platform"
            "userVerification": "required",
            "residentKey": "required",  # discoverable credential (passkey)
        },
        "excludeCredentials": [],  # list of already-registered credentials
    }
```

### Registration ceremony — verifying the response

```python
def verify_registration(response, expected_challenge, expected_origin):
    # Parse clientDataJSON (browser-generated)
    client_data = json.loads(base64url_decode(response["response"]["clientDataJSON"]))

    # CRITICAL VERIFICATIONS:
    assert client_data["type"] == "webauthn.create"
    assert client_data["challenge"] == base64url_encode(expected_challenge)
    assert client_data["origin"] == expected_origin  # ← phishing-resistance anchor

    # Parse attestationObject (CBOR-encoded)
    attestation_object = cbor.decode(base64url_decode(response["response"]["attestationObject"]))
    auth_data = parse_authenticator_data(attestation_object["authData"])

    # Verify rpIdHash matches SHA-256(rp.id)
    assert auth_data.rp_id_hash == sha256(b"app.example.com")

    # Verify flags: User Present (UP), User Verified (UV)
    assert auth_data.flags & 0x01  # UP
    assert auth_data.flags & 0x04  # UV

    # Extract the credential public key (COSE-encoded)
    # Store in DB: credential_id, public_key, sign_count, user_id
    return {
        "credential_id": auth_data.credential_id,
        "public_key": auth_data.credential_public_key,
        "sign_count": auth_data.sign_count,
    }
```

### Assertion ceremony — verifying the signature

```python
def verify_assertion(response, expected_challenge, expected_origin,
                     stored_credential):
    client_data = json.loads(base64url_decode(response["response"]["clientDataJSON"]))
    assert client_data["type"] == "webauthn.get"
    assert client_data["challenge"] == base64url_encode(expected_challenge)
    assert client_data["origin"] == expected_origin

    auth_data_bytes = base64url_decode(response["response"]["authenticatorData"])
    signature = base64url_decode(response["response"]["signature"])

    # The signed data is auth_data || SHA-256(clientDataJSON)
    client_data_hash = sha256(base64url_decode(response["response"]["clientDataJSON"]))
    signed_bytes = auth_data_bytes + client_data_hash

    # Verify signature with stored public key
    pub_key = cose_to_pem(stored_credential["public_key"])
    if not verify_signature(pub_key, signature, signed_bytes):
        raise AuthenticationFailed()

    # Anti-rollback: sign_count must increase
    new_sign_count = parse_sign_count(auth_data_bytes)
    if new_sign_count <= stored_credential["sign_count"] and new_sign_count != 0:
        raise PossibleClonedCredential()  # The "duplicate authenticator" attack

    update_credential_sign_count(stored_credential, new_sign_count)
    return True
```

The sign_count anti-rollback protects against an attacker cloning the authenticator (e.g., by extracting the private key from a compromised hardware token).

## TOTP Algorithm Walkthrough

RFC 6238 defines TOTP as `HOTP(K, T)` where T is the time step:

```python
import hmac, hashlib, struct, time

def totp(secret_b32, period=30, digits=6, algorithm="sha1"):
    # 1. Decode base32 secret
    secret = base64.b32decode(secret_b32, casefold=True)

    # 2. Compute time step T (Unix time / period)
    T = int(time.time() / period)
    msg = struct.pack(">Q", T)  # 8-byte big-endian time counter

    # 3. HMAC the time counter with the secret
    h = hmac.new(secret, msg, hashlib.sha1).digest()  # 20 bytes

    # 4. Dynamic truncation (RFC 4226 §5.3)
    offset = h[-1] & 0x0F  # use last 4 bits as offset (0-15)
    code = (struct.unpack(">I", h[offset:offset+4])[0] & 0x7FFFFFFF)
    # ↑ mask off MSB to handle signed-int conversion in Java/JS implementations

    # 5. Modulo to get 6 (or 7/8) digits
    return str(code % (10 ** digits)).zfill(digits)

# Usage:
print(totp("JBSWY3DPEHPK3PXP"))  # standard test secret
```

The 30-second window means a code is valid for at most 30s, with ±1 step tolerance for clock skew (so effectively up to 90s). The 6-digit truncation reduces the brute-force space to 10^6 = ~20 bits per attempt.

## References

- **RFC 9106** — Argon2 Memory-Hard Function for Password Hashing and Proof-of-Work Applications.
- **RFC 7914** — The scrypt Password-Based Key Derivation Function.
- **RFC 2898** — PKCS #5: Password-Based Cryptography Specification (PBKDF2).
- **NIST SP 800-63B** — Digital Identity Guidelines: Authentication and Lifecycle Management (rev 4, 2024).
- **OWASP Authentication Cheat Sheet** — https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
- **OWASP Password Storage Cheat Sheet** — https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- **OWASP Session Management Cheat Sheet** — https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
- **OWASP Cross-Site Request Forgery Prevention Cheat Sheet** — https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
- **RFC 7519** — JSON Web Token (JWT).
- **RFC 7515** — JSON Web Signature (JWS).
- **RFC 7517** — JSON Web Key (JWK).
- **RFC 7518** — JSON Web Algorithms (JWA).
- **RFC 6749** — The OAuth 2.0 Authorization Framework.
- **RFC 6819** — OAuth 2.0 Threat Model and Security Considerations.
- **RFC 7636** — Proof Key for Code Exchange by OAuth Public Clients (PKCE).
- **OAuth 2.1 draft** — https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/
- **OpenID Connect Core 1.0** — https://openid.net/specs/openid-connect-core-1_0.html
- **W3C Web Authentication: An API for accessing Public Key Credentials Level 2** — https://www.w3.org/TR/webauthn-2/
- **FIDO2 / CTAP2** — https://fidoalliance.org/specs/fido-v2.0-rd-20180702/fido-client-to-authenticator-protocol-v2.0-rd-20180702.html
- **RFC 6238** — TOTP: Time-Based One-Time Password Algorithm.
- **RFC 4226** — HOTP: An HMAC-Based One-Time Password Algorithm.
- **NIST SP 800-63B (SMS deprecation guidance)** — https://pages.nist.gov/800-63-3/sp800-63b.html
- **Have I Been Pwned: Pwned Passwords API** — https://haveibeenpwned.com/API/v3#PwnedPasswords
- **Microsoft MSRC: Pre-Account Hijacking research** — https://msrc-blog.microsoft.com/2022/05/24/pre-hijacking-attacks-on-web-user-accounts/
- **PortSwigger Web Security Academy: Authentication** — https://portswigger.net/web-security/authentication
- **Google Online Security Blog: number matching for MFA** — https://security.googleblog.com/
- **Bleichenbacher's CRYPTO '98 paper on PKCS#1 v1.5 padding oracle.**
- **Coppersmith, Bernstein — papers on timing attacks against RSA.**
- **Biryukov, Dinu, Khovratovich (2016) — "Argon2: the memory-hard function for password hashing and other applications" (PHC submission).**
- **Provos, Mazières (1999) — "A Future-Adaptable Password Scheme" (bcrypt).**
- **Percival (2009) — "Stronger Key Derivation via Sequential Memory-Hard Functions" (scrypt).**
- **Engel (2014), Nohl (2016) — SS7 protocol attacks (Chaos Communication Congress talks).**
- **Komanduri et al. (2011) — "Of Passwords and People: Measuring the Effect of Password-Composition Policies" CHI.**
- **CMU Password Research Group** — https://www.ece.cmu.edu/~lbauer/proj/passwords.php
- **OAuth 2.0 Security Best Current Practice (RFC 9700, 2024)** — https://datatracker.ietf.org/doc/rfc9700/
- **Cure53 / various WebAuthn deployment audits** — public reports on production WebAuthn integrations.
- **draft-ietf-oauth-security-topics-26** — incremental updates to OAuth 2.0 BCP.

## Production Threat Models (Extended)

### Threat 1: Credential Stuffing at Scale

**Attack model:** attacker has a leaked database of (email, password) pairs from a breach of Service A. They replay those pairs against Service B's login endpoint, expecting ~1-2% reuse rate.

**Math:** with a 1% reuse rate against a database of 10M leaked credentials, an attacker probing at 100 attempts/sec finds ~1000 valid logins per hour at Service B. Over a week: ~168,000 takeovers.

**Defenses, in priority order:**
1. **HIBP check on login** via k-anonymity API.
2. **Per-IP rate limit**: ≤5 failures/15min, 1h cooldown.
3. **Per-account rate limit**: ≤5 failures/15min per email regardless of IP.
4. **CAPTCHA escalation**: after 3 failures from any (account, IP), CAPTCHA. After 10, require email verification.
5. **MFA for all accounts**: SMS-MFA cuts credential-stuffing success ~99%; WebAuthn 100%.
6. **Detection signals**: 401 spike, geographic clustering, user-agent monoculture.

### Threat 2: Phishing → Session Hijack

Phishing site captures password AND live MFA code; backend logs in as victim within seconds.

**Defenses:**
1. **WebAuthn**: domain-bound credentials cannot be replayed against the wrong RP. The phishing site at `g0ogle.com` cannot use a credential issued for `google.com`.
2. **DPoP / mTLS-bound tokens (RFC 9449)**: tokens useless without DPoP nonce dance.
3. **Continuous Access Evaluation (CAEP)**: re-prompt for MFA on impossible travel / new device.
4. **Refresh token rotation**: detect reuse; force re-auth (RFC 6749 §6).
5. **Email/SMS notification on new login**.

### Threat 3: Insider with Database Access

DBA can read password-hash column. Weak hashes (MD5, SHA-1, unsalted, bcrypt-cost-4) crackable offline. Even Argon2id crackable if cost too low.

**Defenses:**
1. **Argon2id calibrated cost**: ≥250ms per hash on prod. GPU at 10M H/s on Argon2 cannot keep up.
2. **Pepper**: server-side secret added to hash, stored OUTSIDE database (HSM, Vault).
3. **Periodic re-hashing**: when login succeeds, transparently rehash with current cost.
4. **Hash rotation**: HMAC the hash with a versioned KMS key; rotate periodically.

## Argon2id Parameter Calibration (Worked Example)

Goal: ≥250ms per hash on production hardware.

```python
from argon2 import PasswordHasher
import time

# Start: time_cost=2, memory_cost=64MB, parallelism=1
hasher = PasswordHasher(time_cost=2, memory_cost=65536, parallelism=1, hash_len=32, salt_len=16)
start = time.perf_counter()
hasher.hash("test password 12345")
elapsed = time.perf_counter() - start
print(f"hash time: {elapsed*1000:.1f}ms")
```

Increase `memory_cost` rather than `time_cost` (better resistance to GPU). Goal: 64-128MB per hash, 250-500ms.

| Hardware | Recommended params |
|----------|-------------------|
| Modern x86 server (16+ cores) | t=2, m=131072 (128MB), p=4 |
| Modest cloud instance | t=3, m=65536 (64MB), p=2 |
| Edge / Lambda | t=2, m=32768 (32MB), p=1 (250ms warm, 500ms cold) |

OWASP minimum 2024: `m=46MB, t=1, p=1` Argon2id, OR `t=2, m=19MB, p=1` Argon2id, OR bcrypt cost 12.

## WebAuthn Ceremony Step-by-Step

### Registration

1. **Server**: generate `challenge` (random 32 bytes), `userId` (stable handle), `rpId` (your domain). Store challenge in session.
2. **Server → Browser**: `navigator.credentials.create({publicKey: {challenge, rp, user, pubKeyCredParams: [{alg: -7, type: "public-key"}]}})`.
3. **Browser → Authenticator**: prompts user (touch yubikey / face ID).
4. **Authenticator**: generates new key pair scoped to (rpId, userId). Returns `(publicKey, attestation, signature)`.
5. **Browser → Server**: posts the credential.
6. **Server**: verifies attestation chain (optional in non-FIDO-AAGUID mode), stores `(userId, credentialId, publicKey)`.

### Authentication

1. Server generates challenge, looks up known credentialIds.
2. Browser → Authenticator → user touches device.
3. Authenticator signs `(authenticatorData ‖ clientDataHash)` with the credential's private key.
4. Server verifies signature against stored publicKey, verifies challenge match, verifies `origin` match. Issues session.

The signature COVERS the rpId hash — a phishing site at a different domain literally cannot produce a valid signature, even if the user touches the YubiKey.

## TOTP RFC 6238 Math

```
counter = floor((current_unix_timestamp - T0) / X)   # T0=0, X=30s default
HOTP = HMAC-SHA-1(secret, counter)                    # 20-byte HMAC
offset = HOTP[19] & 0x0F                               # last nibble
truncated = (HOTP[offset] & 0x7F) << 24
          | HOTP[offset+1] << 16
          | HOTP[offset+2] << 8
          | HOTP[offset+3]
code = truncated mod 10^6                              # 6 digits
```

Server-side allow ±1 window (90s drift tolerance). Shared secret is 160 bits (HMAC-SHA-1 native), encoded as base32 for QR transport: `otpauth://totp/Issuer:account?secret=BASE32SECRET&issuer=Issuer&algorithm=SHA1&digits=6&period=30`.
