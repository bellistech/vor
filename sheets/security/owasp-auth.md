# OWASP Authentication

Verifying who-you-are without becoming the breach. Argon2id, sessions, JWT, MFA, OAuth/OIDC, CSRF, federation, recovery — all of it in-terminal so you never have to alt-tab to a browser at 2am.

## Setup

- **Authentication (AuthN)** — verifying *identity*: are you who you claim to be? (login, MFA, signature)
- **Authorization (AuthZ)** — checking *permission*: can this verified principal perform this action on this resource? (RBAC, ABAC, policy)
- **The conflation that bites every newcomer** — "logged in == allowed" is wrong; a logged-in user is still an attacker against everything they shouldn't access. AuthN happens once per session; AuthZ happens on every request.
- **OWASP ASVS** — Application Security Verification Standard, v4.0.3, sections V2 (Authentication), V3 (Session), V4 (Access Control). Treat L2 as the floor for any internet-exposed app.
- **Cheatsheet sources** — OWASP Cheat Sheet Series: Authentication, Session_Management, Password_Storage, Multifactor_Authentication, JSON_Web_Token_for_Java, OAuth, Forgot_Password.
- **Mental model** — the auth subsystem is the front door; everything behind it trusts the assertion it produces. Bugs here are crown-jewel bugs.
- **Never roll your own** — use the language's library (passlib, golang.org/x/crypto, password_hash, Spring Security). Auth bugs are subtle, exploits are not.

```bash
# Levels of OWASP ASVS coverage
# L1 — opportunistic protection, automated tooling can verify
# L2 — most apps with sensitive data (default for public web app)
# L3 — high-value (military, medical, finance)
```

## Threat Model

- **What attackers want**
  - **Account takeover (ATO)** — own a single account, drain it, pivot
  - **Credential reuse / lateral movement** — same password unlocks 12 other services
  - **Privilege escalation** — go from `user` to `admin` via impersonation, broken AuthZ, or role-mapping flaw
  - **Mass compromise** — breach the password store and offline-crack everyone
  - **Persistent access** — install attacker MFA device, attacker SSH key, refresh-token grant
- **Attack vectors (memorize this list)**
  - **Credential stuffing** — replay leaked passwords across services (most common ATO in 2024)
  - **Brute force** — top-100k password list against a target username
  - **Password spraying** — top-10 passwords against millions of usernames (evades per-account lockout)
  - **Session hijack** — steal cookie via XSS / network / malware
  - **Session fixation** — set victim's session ID before they log in
  - **MFA bypass** — push-bombing, social-engineering the help desk, SIM swap, response-step skipping
  - **Password reset abuse** — predictable tokens, host-header injection in reset email, lockout-via-reset DoS
  - **Account enumeration** — different responses for "valid email, wrong password" vs "no such email"
  - **OAuth misconfig** — open `redirect_uri`, missing `state`, mixing access/id tokens
  - **JWT alg=none** — verifier accepts unsigned tokens
  - **JWT algorithm confusion** — HS256 verify with RS256 public key as HMAC secret
  - **CSRF** — authenticated browser tricked into making side-effect request
  - **Phishing** — fake login page, fake OAuth consent
  - **Reverse-tabnabbing / clickjacking** — overlay your form
  - **Cookie theft via XSS** — non-HttpOnly cookie + a stored-XSS = ATO
  - **Replay** — capture and resend an auth artifact
- **Threat modeling questions to ask** — what's the highest-value account class? what mass-attack would unlock $X? where do we trust assertions we shouldn't?

```bash
# STRIDE applied to auth
# S poofing      — defeated by AuthN
# T ampering     — defeated by signed/HMACed tokens
# R epudiation   — defeated by audit logging
# I nfo disclosure — defeated by const-time compare, generic errors
# D oS           — defeated by per-user rate limit (NOT per-IP only)
# E levation     — defeated by AuthZ on every request
```

## Password Storage — Argon2id

- **Argon2id** — winner of the 2015 Password Hashing Competition; OWASP's first-choice algorithm.
- **Why id** — Argon2i is data-independent (side-channel resistant), Argon2d is data-dependent (faster). `id` interleaves both — best of both worlds.
- **OWASP 2024 minimum parameters** (use higher if your hardware allows):
  - `memoryCost` (m) — **≥ 19456 KiB (19 MiB)**, prefer 64 MiB if under 100ms budget
  - `timeCost` (t) — **≥ 2 iterations**
  - `parallelism` (p) — **≥ 1 thread**
  - `hashLen` — **32 bytes** output (256-bit)
  - `saltLen` — **16 bytes** (128-bit) random per password from CSPRNG
- **Use defaults from your library** — most libs ship sane defaults; bench locally and bump until ~100ms per hash on your prod CPU.
- **Encoded format** — `$argon2id$v=19$m=19456,t=2,p=1$<base64-salt>$<base64-hash>` — store this whole string in a single column.
- **Tuning** — measure on your prod CPU; if a single hash takes <50ms, double `m` until it sits at 100-500ms (interactive) or 1s (batch).

```python
from argon2 import PasswordHasher
ph = PasswordHasher(time_cost=2, memory_cost=19456, parallelism=1, hash_len=32, salt_len=16)
encoded = ph.hash("correct horse battery staple")
ph.verify(encoded, "correct horse battery staple")  # True or raises
ph.check_needs_rehash(encoded)  # True if params upgraded
```

```go
// golang.org/x/crypto/argon2
salt := make([]byte, 16)
if _, err := rand.Read(salt); err != nil { panic(err) }
hash := argon2.IDKey([]byte(pw), salt, 2, 19456, 1, 32)
// Encode: $argon2id$v=19$m=19456,t=2,p=1$<b64salt>$<b64hash>
```

## Password Storage — bcrypt

- **bcrypt** — Blowfish-based, from 1999; still acceptable in 2024 if Argon2id unavailable (legacy systems, FIPS environments).
- **Cost factor** — **≥ 12 (OWASP 2024 minimum)**; cost is exponential (cost 12 = 2^12 = 4096 rounds).
- **Aim for 250-500ms** per hash on your prod CPU — bench it.
- **72-byte input limit** — bcrypt silently truncates passwords beyond 72 bytes. Long passphrases get clipped.
- **Pre-hash + bcrypt pattern** — `bcrypt(base64(sha256(password)))` — sidesteps the 72-byte limit and pepper conflict.
  - **DON'T** use raw `sha256(password)` because the resulting bytes can contain NUL — bcrypt stops at first NUL on some libraries. **Always base64-encode before bcrypt.**
- **bcrypt-on-bcrypt anti-pattern** — `bcrypt(bcrypt(pw))` does NOT increase security meaningfully and makes verification impossible without the inner work factor. Don't.
- **Salt** — bcrypt embeds a 128-bit salt automatically; no manual salt management.

```python
import bcrypt
hashed = bcrypt.hashpw(b"correct horse battery staple", bcrypt.gensalt(rounds=12))
bcrypt.checkpw(b"correct horse battery staple", hashed)  # True

# Pre-hash for >72 byte input
import base64, hashlib
pre = base64.b64encode(hashlib.sha256(pw.encode()).digest())
hashed = bcrypt.hashpw(pre, bcrypt.gensalt(rounds=12))
```

```go
import "golang.org/x/crypto/bcrypt"
hashed, _ := bcrypt.GenerateFromPassword([]byte(pw), 12)
err := bcrypt.CompareHashAndPassword(hashed, []byte(pw))
```

## Password Storage — scrypt

- **scrypt** — memory-hard KDF from Colin Percival (2009); used by Bitcoin's Litecoin variant and 1Password.
- **Parameters**
  - `N` — CPU/memory cost factor; **≥ 2^17 (131072)** for OWASP 2024 (interactive auth)
  - `r` — block size; **8** is canonical
  - `p` — parallelism; **1** for auth (>1 only for batch derivation)
  - `dkLen` — output length; **32 bytes**
  - `salt` — **16+ bytes** from CSPRNG
- **Memory cost** ≈ `128 * N * r` bytes — at N=131072, r=8 that's ~128 MiB per hash. Plan capacity.
- **scrypt vs argon2id** — argon2id is newer, better-analyzed, more configurable; pick argon2id by default. scrypt only if your stack mandates it (Erlang, some Bitcoin-adjacent codebases) or you're already running it.

```python
import hashlib, os
salt = os.urandom(16)
hashed = hashlib.scrypt(pw.encode(), salt=salt, n=2**17, r=8, p=1, dklen=32)
# Store: salt + hashed (or use passlib's scrypt scheme)
```

```go
// golang.org/x/crypto/scrypt
hash, err := scrypt.Key([]byte(pw), salt, 1<<17, 8, 1, 32)
```

## Password Storage — PBKDF2

- **PBKDF2** — RFC 8018; FIPS 140-validated; the boring choice for FIPS / .gov / regulated stacks.
- **Iterations (OWASP 2024)**
  - **PBKDF2-HMAC-SHA256** — **≥ 600,000** iterations (OWASP 2023 update bumped from 310k)
  - **PBKDF2-HMAC-SHA512** — **≥ 210,000** iterations
  - **PBKDF2-HMAC-SHA1** — **≥ 1,300,000** iterations (legacy; prefer SHA-256)
- **Salt** — **16+ bytes** random from CSPRNG; per-password.
- **Output (dkLen)** — match HMAC output (32 bytes for SHA-256) — asking for more triggers re-derivation = no extra security but more CPU.
- **Why PBKDF2 is "OK but not great"** — not memory-hard; GPU/ASIC crackers chew it. Use only if compliance demands it.

```python
import hashlib, os
salt = os.urandom(16)
dk = hashlib.pbkdf2_hmac("sha256", pw.encode(), salt, iterations=600_000, dklen=32)
```

```go
import "crypto/sha256"
import "golang.org/x/crypto/pbkdf2"
dk := pbkdf2.Key([]byte(pw), salt, 600_000, 32, sha256.New)
```

## Password Storage — Pepper

- **Pepper** — a server-side secret added to every password hash. Database leak alone ≠ crackable hashes (attacker also needs pepper).
- **Pattern (OWASP 2024)** — pre-hash with HMAC, then KDF:
  ```
  intermediate = HMAC-SHA256(key=pepper, msg=password)
  stored = Argon2id(intermediate, salt, params)
  ```
- **Why HMAC-then-KDF** — keeps the pepper out of the slow KDF (cheap), bounds the input (sidesteps bcrypt's 72-byte limit cleanly).
- **Pepper storage**
  - **Yes** — HSM (AWS CloudHSM, Azure Dedicated HSM, YubiHSM)
  - **Yes** — secret manager (HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager, Doppler)
  - **Yes** — env var injected by orchestrator (k8s Secret, ECS task-role)
  - **No** — alongside the password column in DB (defeats the point)
  - **No** — git repo (even private — use `vault` skill)
- **Rotation** — track `pepper_version` per row; on next successful login, re-derive with current pepper and update.
- **Tradeoff** — pepper rotation is hard; lose the pepper = lose all logins. Back it up to an offline encrypted medium.

```python
import hmac, hashlib
pepper = os.environ["AUTH_PEPPER"].encode()  # 32+ random bytes
intermediate = hmac.new(pepper, pw.encode(), hashlib.sha256).digest()
stored = ph.hash(intermediate)  # argon2id
```

## Password Storage — Don'ts

- **NEVER plaintext** — even "internal-only", even "dev-only", even "behind a VPN". A db dump = total compromise.
- **NEVER reversible encryption** — if your "forgot password" emails the *current* password, your design is broken. Reset → new password, full stop.
- **NEVER MD5 / SHA-1 / SHA-256 / SHA-512 alone** — not slow, not memory-hard. A modern GPU cracks 100B+ SHA-256/sec. Salting helps against rainbow tables but does nothing against per-account brute force.
- **NEVER your own scheme** — `sha256(salt + pw + secret)` is not an algorithm, it's a CV-bug factory.
- **NEVER design a system that can email you your password** — implies you can decrypt it, implies the attacker can after a breach. Reset only.
- **NEVER log the password** — not in access logs, not in stack traces, not in the request body capture.
- **NEVER compare with `==`** — timing attack. Use library `verify()` or `hmac.compare_digest`.
- **NEVER share password hashes across services** — same password across two services with the same hash scheme = pivot.

## Password Storage — Library Recipes

- **Python**
  - `passlib` — high-level multi-scheme; `CryptContext(schemes=["argon2"], deprecated="auto")`
  - `argon2-cffi` — direct argon2id binding; `PasswordHasher()`
  - `bcrypt` — `bcrypt.hashpw / checkpw`
  - `hashlib.scrypt`, `hashlib.pbkdf2_hmac` for stdlib-only
- **Go**
  - `golang.org/x/crypto/argon2` — `argon2.IDKey`
  - `golang.org/x/crypto/bcrypt` — `bcrypt.GenerateFromPassword`
  - `golang.org/x/crypto/scrypt`, `golang.org/x/crypto/pbkdf2`
- **Node.js**
  - `@node-rs/argon2` — fastest argon2id for Node (Rust-backed)
  - `argon2` (npm) — pure-N-API binding, also good
  - `bcrypt` or `bcryptjs` (avoid bcryptjs in prod — pure JS, slow)
- **Ruby** — `bcrypt-ruby` (`Password.create`); `argon2` gem
- **PHP** — `password_hash($pw, PASSWORD_ARGON2ID)`, verify with `password_verify`; `password_needs_rehash` for upgrades
- **Java**
  - Spring Security — `Argon2PasswordEncoder`, `BCryptPasswordEncoder`, `Pbkdf2PasswordEncoder`, `DelegatingPasswordEncoder` for migration
  - `de.mkammerer:argon2-jvm` for direct argon2
- **Rust** — `argon2` crate (`Argon2::default().hash_password()`); `bcrypt` crate
- **Erlang/Elixir** — `argon2_elixir`; `bcrypt_elixir`
- **.NET** — `Konscious.Security.Cryptography.Argon2` for argon2id; `BCrypt.Net-Next`; `Microsoft.AspNetCore.Identity` uses PBKDF2 by default

```python
# Python migration pattern via passlib
from passlib.context import CryptContext
ctx = CryptContext(schemes=["argon2", "bcrypt", "pbkdf2_sha256"], deprecated="auto")
ok, new_hash = ctx.verify_and_update(plain, stored)
if new_hash:
    db.update_user(user_id, password_hash=new_hash)
```

```php
$hash = password_hash($pw, PASSWORD_ARGON2ID);
if (password_verify($pw, $hash)) {
    if (password_needs_rehash($hash, PASSWORD_ARGON2ID)) {
        $hash = password_hash($pw, PASSWORD_ARGON2ID);
        db_update_password($user_id, $hash);
    }
}
```

## Password Policy

- **Minimum length** — **8 chars (OWASP 2024 floor)**; **prefer 14+** for human-chosen; **64 max** so passphrases work.
- **No composition rules** — NIST SP 800-63B revision 3 forbids "must have uppercase + symbol + digit" requirements. They lower entropy (everyone does `Password1!`).
- **No periodic rotation** — NIST 800-63B forbids forced periodic resets. Rotate only on suspected compromise.
- **Allow all printable Unicode** — including spaces, emoji. Length is what matters.
- **Check against breached-password lists** — use HIBP Pwned Passwords API (k-anonymity model: send first 5 chars of SHA-1 hash, get list of suffixes).
- **Strength meter** — `zxcvbn` (Dropbox) gives realistic entropy estimates; show as feedback, do NOT block submission unless score < 3.
- **Reject obvious anti-patterns** — `Password`, `P@ssw0rd`, `Welcome1`, the username, the email local-part, the company name.
- **Don't truncate silently** — if you have a length cap, error visibly above it (and don't have one below 64).
- **Don't strip whitespace** — leading/trailing spaces matter; the user typed them.

```bash
# HIBP API — k-anonymity
echo -n "correct horse battery staple" | sha1sum | tr '[:lower:]' '[:upper:]'
# Take first 5 chars, GET https://api.pwnedpasswords.com/range/<5chars>
# Search response for the remaining 35 chars; if present, password is breached
```

```python
import hashlib, requests
def is_breached(pw):
    h = hashlib.sha1(pw.encode()).hexdigest().upper()
    prefix, suffix = h[:5], h[5:]
    r = requests.get(f"https://api.pwnedpasswords.com/range/{prefix}", timeout=3)
    return any(line.split(":")[0] == suffix for line in r.text.splitlines())
```

## Password Reset

- **Token generation** — **32 bytes** from CSPRNG; URL-safe encode (base64url or hex).
  - Python — `secrets.token_urlsafe(32)`
  - Go — `crypto/rand.Read(buf[:32])` + base64url
  - Node — `crypto.randomBytes(32).toString("base64url")`
  - Ruby — `SecureRandom.urlsafe_base64(32)`
- **TTL** — **15 minutes** for high-value, **30-60 min** for typical. Long TTL = larger window for token theft via email forwarding.
- **Single-use** — mark `used=true` on first verification; reject thereafter.
- **One outstanding token per account** — invalidate prior tokens when issuing a new one.
- **Token via email** — primary channel for most apps; assumes email account is at least as secure as the app.
- **Token via SMS** — vulnerable to SIM swap; avoid for high-value resets.
- **Email is the de-facto root account** — own the email, own all resets. Encourage users to MFA their email and beware the recursive recovery problem.
- **Don't put the token in the URL fragment** — fragments aren't sent to server but can leak via Referer/Analytics.
- **Account-Lockout-via-Reset attack** — attacker triggers reset on victim's account, victim's password keeps getting "expired" as new tokens issue. Mitigate: don't expire the active password until the new one is set.
- **Host header injection in reset emails** — never use `Host` header to build reset URLs; pin the canonical domain in config.

## Password Reset — Implementation

- **DB schema**
  ```
  reset_tokens(id PK, user_id FK, token_hash bytea NOT NULL, created_at, expires_at, used_at)
  ```
- **Store hash, not token** — `sha256(token)` is fine here (low entropy on the input is the threat — but we generated 256-bit, so no rainbow-table risk; we just don't want a DB read to ATO).
- **Verification** — find row by `token_hash`, check `expires_at > now`, check `used_at IS NULL`, **constant-time compare** the hash, mark `used_at = now` atomically.
- **Resolve user_id from token** — never accept `user_id` from the URL (attacker controls); the token *is* the credential.
- **Rate limit reset requests** — per-account (5/hour) and per-IP (50/hour).

```python
import secrets, hashlib, hmac
def issue_reset(user_id):
    token = secrets.token_urlsafe(32)
    th = hashlib.sha256(token.encode()).digest()
    db.insert_reset(user_id, th, expires_at=now()+15*minute)
    send_email(user_id, f"https://app.example.com/reset?t={token}")

def consume_reset(token):
    th = hashlib.sha256(token.encode()).digest()
    row = db.get_reset_by_hash(th)
    if not row or row.used_at or row.expires_at < now():
        return None
    if not hmac.compare_digest(row.token_hash, th):
        return None
    db.mark_used(row.id)
    return row.user_id
```

```go
token := make([]byte, 32)
rand.Read(token)
encoded := base64.RawURLEncoding.EncodeToString(token)
sum := sha256.Sum256(token)
db.Exec("INSERT INTO reset_tokens(user_id, token_hash, expires_at) VALUES($1,$2,$3)",
    userID, sum[:], time.Now().Add(15*time.Minute))
```

## Account Enumeration — General

- **The leak** — different responses for "valid email, wrong password" vs "no such user". Attackers build account lists, then targeted phish.
- **Always-same response on login**
  - **Bad** — "Invalid password" vs "User not found"
  - **Good** — "Invalid email or password" (single, generic)
- **Always-same response on registration / reset**
  - **Bad** — "Email already registered, please log in"
  - **Good** — "If an account with that email exists, we've sent a verification link"
- **Same response timing** — measure: a "user not found" path that skips bcrypt verify completes in microseconds; a real path takes ~250ms. Attackers diff this.
  - **Mitigation** — on miss, run a dummy hash with a constant fake-password against a constant fake-hash to consume the same time budget.
- **Same response size** — attackers also compare `Content-Length`. Make HTML/JSON identical.
- **HTTP status code** — same 200 (or same 401) for both branches.

```python
DUMMY_HASH = ph.hash("not-a-real-password")  # computed once at boot
def login(email, password):
    user = db.find_by_email(email)
    if user is None:
        ph.verify(DUMMY_HASH, password)  # consume time
        return generic_fail()
    try:
        ph.verify(user.pwd_hash, password)
    except VerifyMismatchError:
        return generic_fail()
    return success(user)
```

## Account Enumeration — Per-Flow

- **Registration flow**
  - **Bad** — "That username is taken" (oracle for stuffing)
  - **Good** — "We've sent a verification link to that email" — and ALSO send to the existing-account holder a "someone tried to register with your email" email with a "this was me / not me" link.
- **Password reset flow**
  - **Bad** — "No account with that email"
  - **Good** — "If an account exists, an email has been sent" — with constant timing.
- **Username vs email**
  - **Username-as-login** — usernames are public on most apps (mentions, profiles, URLs); enumeration is cheap regardless. Defense-in-depth still: don't confirm.
  - **Email-as-login** — emails are private; account-enumeration via email = privacy breach. Treat existence as a secret.
- **OAuth callback** — the redirect's URL params can leak ("error=user_not_found"); always normalize.
- **Forgot-username flow** — same generic response; deliver via email if any.

## Account Lockout / Brute Force

- **Lockout per username** — count failed attempts per account; **5 in 15 minutes** triggers temporary lock (15-30 min).
- **Why per-username not per-IP** — attackers rotate IPs (botnet, residential proxy); per-IP rate limiting alone fails. Per-username catches credential-stuffing too.
- **Per-IP rate limit** — defense-in-depth: 100 login attempts per IP per minute total. Throws away the unsophisticated.
- **Per-account-AND-IP combination** — separately track; alarm on either ceiling.
- **Exponential backoff** — failure 1: 0s; failure 2: 1s; failure 3: 2s; failure 4: 4s — slows interactive guessing without DoSing legit retries.
- **CAPTCHA after N failures** — show after 3 fails on an account. **Accessibility caveat** — image CAPTCHAs are barriers; prefer hCaptcha / Cloudflare Turnstile / invisible / audio fallback.
- **Lockout-as-DoS-vector** — attacker locks legit user out by spamming wrong passwords. Mitigations:
  - **Soft lock** — require CAPTCHA + delay, don't outright deny
  - **Notify owner** — email "5 failed login attempts on your account"
  - **Allow MFA bypass of lockout** — if user has valid second factor, let them through
- **Alert + audit** — every lockout = audit log entry; SOC reviews thresholds.
- **Stuffing-specific** — detect "many usernames, few attempts each" pattern (low rate per account, high IP volume); block IP + tag for review.

```bash
# Sliding-window rate limit (Redis)
KEY="login:fail:$user_id"
redis-cli ZREMRANGEBYSCORE $KEY 0 $(date -d '15 min ago' +%s)
COUNT=$(redis-cli ZCARD $KEY)
[ "$COUNT" -ge 5 ] && echo locked || redis-cli ZADD $KEY $(date +%s) "$(uuidgen)"
```

## Session Management — Cookie Attributes

- **HttpOnly** — cookie not readable from JavaScript (`document.cookie`). Defends against XSS-driven session theft. **Always set on session cookies.**
- **Secure** — cookie sent only over HTTPS. **Always set in prod.** Localhost dev: ok to relax conditionally.
- **SameSite**
  - `Strict` — never sent on cross-site requests (clicks from email kill session). Best for banking/admin.
  - `Lax` (browser default since 2021) — sent on top-level GET navigations cross-site, blocks POST/iframe. Good for typical web app.
  - `None` — sent everywhere; **REQUIRES `Secure`**. For genuine third-party-cookie use cases (cross-domain SSO, embedded iframes).
- **Path** — default `/`; restrict if cookie is route-scoped (`/admin`).
- **Domain** — **avoid wildcard** (`Domain=.example.com`) — sends cookie to every subdomain (including attacker.example.com if subdomain takeover). Prefer host-only (omit Domain).
- **Expires / Max-Age** — session cookies omit both (browser deletes on close); persistent need explicit `Max-Age=N` (seconds).
- **`__Host-` prefix** — requires `Secure`, no `Domain`, `Path=/`. Browser refuses to set otherwise. **Strongest cookie protection** — defends against subdomain cookie injection.
- **`__Secure-` prefix** — requires `Secure`. Weaker than `__Host-` but allows `Domain`.

```http
Set-Cookie: __Host-sid=8K1d0...; Path=/; Secure; HttpOnly; SameSite=Lax; Max-Age=28800
```

```bash
# Inspect cookies a server sets
curl -sI https://example.com/login | grep -i set-cookie
```

## Session Management — IDs & Lifecycle

- **Session ID generation** — CSPRNG, **128+ bits** of entropy (16+ random bytes), URL-safe encoded.
  - Python — `secrets.token_urlsafe(32)`
  - Go — `crypto/rand.Read(buf[:32])`
  - Node — `crypto.randomBytes(32)`
- **Don't use predictable IDs** — `user_id + timestamp + sha256` = predictable. Don't.
- **Don't expose user-id in cookie** — opaque random ID only; map to user server-side.
- **Rotation on auth** — issue NEW session ID after successful login (defends against session fixation: attacker sets victim's pre-auth ID, victim logs in; without rotation attacker rides the now-authed session).
- **Rotation on privilege change** — re-auth + new session for sudo/admin elevation.
- **Destruction on logout** — server invalidates session record; client-side `Set-Cookie: sid=; Max-Age=0` to remove cookie.
- **Idle timeout**
  - **15 min** — banking, admin, HIPAA-grade
  - **30 min** — most line-of-business apps
  - **24 hours** — low-sensitivity content sites (still set one; "forever" is the bug)
- **Absolute timeout** — **8 hours** for workday apps; force re-auth even with continuous activity. Caps impact of stolen session.
- **Slide-on-activity** — refresh idle TTL on each authed request, but never beyond absolute timeout.
- **"Remember me"** — separate long-lived token (30 days), single-purpose: re-issue session on next visit. Revocable independently.

```python
import secrets
sid = secrets.token_urlsafe(32)
redis.setex(f"session:{sid}", 28_800, json.dumps({"user_id": uid, "iat": int(time.time())}))
response.set_cookie("__Host-sid", sid, secure=True, httponly=True, samesite="Lax", max_age=28_800)
```

## Session Management — Server-Side vs JWT

- **Server-side sessions** — opaque ID in cookie, all state in Redis/DB.
  - **Pro** — instant revocation; small cookie; can change session contents server-side mid-flight.
  - **Pro** — compliance-friendly (no PII in cookie).
  - **Con** — needs a session store; horizontal scale needs sticky sessions or shared store.
- **Client-signed (JWT) sessions** — claims signed with HMAC/RSA; server verifies on each request, no DB lookup.
  - **Pro** — stateless, scales linearly.
  - **Pro** — works across services without a shared session store.
  - **Con** — **revocation is hard** — until the JWT expires, it's valid. Mitigations: short TTL + refresh, deny-list (defeats statelessness), token-versioning per-user.
  - **Con** — token bloat: stuff a user's profile in a JWT, every request now carries 2 KiB.
- **Storage backends for server-side** — Redis (fastest), Memcached (no persistence — survives restart? no), DB (slowest, simplest), Cookie itself (signed/encrypted, see below).
- **Encrypted-cookie sessions** — store state in cookie, encrypt with server secret (Flask `SESSION_TYPE=secure-cookie`, Rails `cookie_store`). Stateless; revocation via secret rotation only.
- **Recommendation** — server-side sessions for browser apps; JWTs only for service-to-service or where statelessness genuinely matters.

## Session Management — Concurrent Sessions

- **Allow multiple sessions** — typical SaaS UX (laptop + phone + tablet); track in DB by session_id with `device_label`, `created_at`, `last_seen_ip`, `last_seen_ua`.
- **Single-session enforcement** — banking-grade: starting a new session destroys the prior one ("you've been logged out — used elsewhere").
- **N-session cap** — limit to 5 active; oldest evicted.
- **"Sessions" page** — let user view all active sessions and revoke individually.
- **"Log out everywhere"** — bumps a `session_version` on the user row; every session ID embeds the version it was issued under; mismatch = invalid. One UPDATE atomically nukes them all.
- **On password change** — invalidate all *other* sessions; keep the one that just changed (UX: "stay logged in here").
- **On compromise indicator** — geo anomaly, new device — prompt re-auth without forcibly logging out.

```sql
CREATE TABLE sessions (
  sid           bytea PRIMARY KEY,
  user_id       int  NOT NULL REFERENCES users(id),
  user_session_version int NOT NULL,  -- compared to users.session_version
  created_at    timestamptz NOT NULL DEFAULT now(),
  last_seen_at  timestamptz NOT NULL DEFAULT now(),
  ip            inet,
  user_agent    text,
  device_label  text
);
```

## Session Management — Binding to IP / UA

- **IP binding** — refuse session if request IP differs from issued IP.
  - **Mobility breaks it** — phones flip cell tower → carrier-grade NAT → wifi → VPN. Locks legit users out constantly.
  - **Verdict** — **usually skip** for consumer apps; consider for admin / VPN gateways where IP is stable.
- **User-Agent binding** — refuse session if UA changes.
  - **Browser updates change UA** — every Chrome update, the version number in UA changes. Locks users out monthly.
  - **Verdict** — usually skip. *Hash-and-loose-match* the UA family if you must.
- **Better signal — anomaly detection** — score per session: ip-asn-jump, country-jump, ua-family-change → on high score, prompt re-auth or step-up MFA. Don't hard-deny.
- **Cloudflare-Turnstile / risk-engine** — bolt-on for SaaS that doesn't want to build this.

## JWT Pitfalls — Algorithm Attacks

- **`alg=none` attack** — RFC 7519 lists `none` as a valid algorithm. Some libraries verify successfully if header says `alg=none` with no signature. **Always pass an explicit allow-list to the verifier.**
- **Algorithm confusion (HS256 vs RS256)** — server expects RS256 (asymmetric: public key verifies). Attacker forges a token with `alg=HS256` (symmetric) and signs with the *public key as HMAC secret*. Naïve verifier loads the public key, sees HS256, uses it as HMAC key → token validates. **Pin the algorithm; don't trust header.**
- **`kid` header injection** — `kid` is "key id"; some servers use it to look up keys by file path or DB row → SQL injection / path traversal / SSRF. **Never feed user-controlled `kid` into a system call; map via allow-list.**
- **`x5u` / `jku` headers** — point to remote JWK URL; if verifier fetches blindly, attacker hosts their own keys → forged token validates. **Disable or pin to known issuers.**
- **Required claim verification**
  - `exp` — expiration; reject if past
  - `nbf` — not-before; reject if future
  - `iat` — issued-at; sanity-check against clock skew
  - `iss` — issuer; must equal expected
  - `aud` — audience; must include this service
  - `sub` — subject; usually user-id
  - `jti` — JWT ID; for replay-detection / deny-list

```python
import jwt
# WRONG — accepts anything
data = jwt.decode(token, key, algorithms=None)

# RIGHT — explicit
data = jwt.decode(
    token, public_key,
    algorithms=["RS256"],
    audience="api.example.com",
    issuer="https://auth.example.com",
    options={"require": ["exp","iat","iss","aud"]},
)
```

```go
// golang-jwt: pin the alg in the keyfunc
parsed, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
        return nil, fmt.Errorf("unexpected alg: %v", t.Header["alg"])
    }
    return rsaPublicKey, nil
})
```

## JWT Pitfalls — Browser Storage

- **localStorage**
  - **Pro** — easy API, persists across tabs/sessions
  - **Con** — readable by **any JS on the origin** → XSS = total token theft → ATO. CSP helps but never fully closes XSS.
- **sessionStorage** — per-tab, cleared on close. Same XSS problem; smaller blast radius.
- **HttpOnly cookie** — JS-invisible; XSS can't read it.
  - **Con** — auto-attached to cross-origin requests → **CSRF risk** unless `SameSite=Lax/Strict`.
  - **Con** — different origin scenarios (SPA at app.com hits api.com) require `SameSite=None; Secure` and cross-origin CSRF defenses (custom headers + CORS).
- **In-memory only (variable)** — lost on reload. Use for short-lived access token paired with HttpOnly refresh-token cookie.
- **Recommendation (OWASP 2024)** — for browser apps, use **HttpOnly cookies for the session/refresh token**; if you must use a header (`Authorization: Bearer ...`), keep the bearer in memory only and refresh from an HttpOnly cookie.
- **Mobile/native** — Keychain (iOS) / Keystore (Android); no localStorage equivalent in native.

## JWT Pitfalls — Refresh Tokens & Lifecycles

- **Short-lived access token** — **15 minutes** typical; the bearer for API calls. Compromise window = 15 min.
- **Long-lived refresh token** — **7-30 days**; trades for a new access token at `/oauth/token`.
- **Rotation** — every refresh issues a NEW refresh token, invalidates the old one. Use `jti` + a "current_jti" per user.
- **Reuse detection** — if an old refresh token is presented after rotation, **the previous holder may be an attacker** → invalidate the entire session family. Notify user.
- **Refresh-token storage**
  - Client: HttpOnly cookie (browser), Keychain (mobile)
  - Server: hashed in DB (don't store raw refresh tokens — db leak = ATO of every user)
- **Sliding expiration** — each rotation extends absolute expiry up to a cap (90 days).
- **Revocation list** — small, deny-list of compromised `jti`s; eviction when expiry passes; checked on each access token verify (cheap with Redis).

## JWT Pitfalls — When To Use

- **Use JWT** — **machine-to-machine across trust boundaries**, federated APIs, OIDC id_tokens.
  - Microservice A → Microservice B with claim "I am user X"
  - Mobile app → API where rotating server-side session cookies is awkward
  - Cross-domain SSO via id_token (OIDC)
- **Use sessions** — **browser → same-domain web app**.
  - Easier, smaller, instantly revocable, no cryptographic footguns.
- **Anti-pattern — JWT as a database**
  - Don't stuff `roles, permissions, profile, settings` into the JWT — bloats every request, can't change without re-auth, can't revoke a single role grant.
  - Put `sub` (user id) only; look up roles fresh from DB per request.

## JWT Pitfalls — Verification Discipline

- **Verify before trust** — never read claims (even `sub`) before signature verifies.
- **Pin algorithm explicitly** — see above.
- **Verify `aud` and `iss`** — a token issued by a sibling service is not valid for yours, even if same key.
- **Check expiry strictly** — minimal clock skew (30s); reject anything past `exp`.
- **No `kid`-based file lookups** — allow-list keys.
- **Reject unknown claims** that affect access decisions you didn't expect (`role`, `is_admin` smuggled into a token issued by a less-trusted service).

```python
# Defensive verify
try:
    claims = jwt.decode(
        tok, key=jwks_lookup, algorithms=["RS256"],
        audience=AUD, issuer=ISS,
        options={"require": ["exp","iat","iss","aud","sub"], "verify_signature": True},
        leeway=30,
    )
except jwt.PyJWTError:
    return 401
```

## MFA — TOTP

- **TOTP (RFC 6238)** — Time-based One-Time Password.
- **Construction** — `HOTP(key, T)` where `T = floor((now - T0) / step)`; default `step=30s`, `T0=0`.
- **Digits** — 6 (default); 8 acceptable.
- **HMAC algo** — **HMAC-SHA-1** per spec; not weak in this construction (the algo's weakness is collision resistance, irrelevant for HOTP). HMAC-SHA-256 / SHA-512 are also valid in RFC 6238 but Google Authenticator and most apps only do SHA-1.
- **Window** — accept current ±1 step (90s total) for clock skew.
- **Apps** — Google Authenticator, Microsoft Authenticator, Authy, 1Password, Bitwarden, Aegis (Android FOSS), Raivo (iOS).
- **Provisioning URI** — `otpauth://totp/Issuer:user@example.com?secret=BASE32SECRET&issuer=Issuer&algorithm=SHA1&digits=6&period=30`
- **QR code** — render the URI; users scan once.
- **Secret** — **160 bits** (20 bytes) random, base32-encoded for the URI.
- **Verification — store secret encrypted (or in HSM); verify with constant-time compare; mark used to prevent within-window replay.**

```python
import pyotp
secret = pyotp.random_base32()  # 32 base32 chars = 160 bits
uri = pyotp.totp.TOTP(secret).provisioning_uri(name="user@example.com", issuer_name="Acme")
# Show as QR
totp = pyotp.TOTP(secret)
totp.verify(user_input, valid_window=1)  # accepts ±30s
```

```bash
# Quick TOTP from CLI
oathtool --totp -b 'JBSWY3DPEHPK3PXP'
```

## MFA — WebAuthn / Passkeys

- **WebAuthn** — W3C standard browser API for public-key MFA & passwordless. **Phishing-resistant** because the credential is bound to the origin.
- **FIDO2 = WebAuthn (browser API) + CTAP2 (authenticator protocol)**.
- **Hardware-backed** — keys live in TPM, Secure Enclave, or external authenticator (YubiKey). Cannot be exfiltrated, only used.
- **Passkeys** — synced WebAuthn credentials (iCloud Keychain, 1Password, Google Password Manager). Same crypto, multi-device UX.
- **Registration ceremony** — server sends `challenge` + `rp.id` + `user`; authenticator generates keypair, returns `credentialId` + `publicKey` + attestation; server stores public key.
- **Assertion ceremony** — server sends `challenge` + `allowCredentials`; authenticator signs with private key (user verifies via biometric/PIN); server verifies signature with stored public key.
- **No shared secrets** — server only ever has public keys; breach = no impact.
- **Origin-binding** — phishing site can't use the credential because `origin` is part of the signed data and the browser supplies it.
- **Counters** — authenticator increments a counter each use; server detects clones.
- **Libraries**
  - Python — `webauthn` (Duo)
  - Go — `github.com/go-webauthn/webauthn`
  - Node — `@simplewebauthn/server`, `@simplewebauthn/browser`
  - Java — Yubico `webauthn-server-core`
- **UX win** — passkey replaces both password and second factor; users tap fingerprint, in.

```javascript
// Browser (registration)
const credential = await navigator.credentials.create({
  publicKey: {
    challenge: serverChallenge,                 // server-issued, base64url
    rp: { id: "example.com", name: "Acme" },
    user: { id: userId, name: email, displayName: email },
    pubKeyCredParams: [{ alg: -7, type: "public-key" }, { alg: -257, type: "public-key" }],
    authenticatorSelection: { residentKey: "preferred", userVerification: "preferred" },
    timeout: 60000,
    attestation: "none",
  },
});
// Send credential to server for verification + storage
```

## MFA — Recovery Codes

- **Generate at MFA setup** — **10 single-use codes**, each 8-12 chars, base32 or words.
- **Display once** — show in a printable format ("save these somewhere safe — each works once").
- **Allow N uses each** — most implementations: each code = one redemption. Some allow re-use within a generation (avoid; less secure).
- **Hashed storage** — bcrypt or argon2 each code; do NOT store plaintext.
- **Regenerate flow** — let user invalidate all and generate a fresh batch.
- **Audit** — every recovery-code use = email notification + audit log entry.
- **Don't email them** — defeats the air-gap. Show on-screen only; user copies/prints.

```python
import secrets
codes = [secrets.token_hex(5).upper() for _ in range(10)]  # 10-char hex
hashes = [ph.hash(c) for c in codes]
db.store_recovery_codes(user_id, hashes)
# Show codes once, never again
```

## MFA — SMS Is Bad

- **SIM-swap attack** — attacker socially engineers carrier into porting victim's number; SMS now goes to attacker. Documented in mainstream media; weekly events.
- **SS7 interception** — telco signaling protocol allows nation-state actors to redirect SMS without touching the carrier.
- **Voice-call OTP** — same threats; also vulnerable to call-forwarding hijack.
- **OWASP recommendation** — "SMS has known security issues, but it can serve as a step-up from password-only. Avoid for high-value accounts."
- **NIST SP 800-63B** — restricts SMS to "RESTRICTED" status; deprecation discouraged but warning users is required.
- **Better choices, in order**: WebAuthn/passkey > Hardware token (YubiKey) > TOTP > Push (with number-matching) > SMS > nothing.
- **Where SMS is still OK** — low-value consumer apps, fallback only when no other factor available, pair with anomaly detection.

## MFA — Push Notifications

- **Push apps** — Duo, Okta Verify, Microsoft Authenticator (push mode).
- **Phishing-resistance — depends on implementation**
  - **Old basic-push** — "Approve / Deny" prompt. **Vulnerable to MFA fatigue / push-bombing** — attacker initiates 50 logins, victim eventually taps Approve to silence.
  - **Number-matching (2024 default)** — login screen shows 2-digit number; user enters it on phone. Defeats blind approval.
  - **Geo + UA shown in prompt** — "from Lagos, Chrome on Windows" lets user spot anomaly.
- **Microsoft, Duo, Okta** all rolled out number-matching by default in 2023-2024.
- **Still not phishing-proof** — attacker phishes user, real auth flows; user does the number-match dance against the attacker's session. WebAuthn defeats this; push doesn't.
- **Verdict** — better than SMS, worse than WebAuthn. Acceptable for enterprise SSO with number-matching enabled.

## MFA — Hardware Tokens

- **YubiKey** — most popular FIDO2/WebAuthn hardware authenticator. Models: 5 NFC, 5C NFC, Bio.
- **SoloKey** — open-source FIDO2 alternative.
- **Titan Security Key** — Google's; FIDO2.
- **Attestation** — authenticator can prove it's a genuine YubiKey (signed by Yubico's CA). Useful for compliance to mandate "only YubiKey-class devices". Optional and privacy-tradeoff (linkability).
- **Lost-key recovery flow** — every account should have:
  - **2+ registered keys** (primary + spare in safe)
  - **Recovery codes** (printed)
  - **Account-recovery contact** for last-resort
- **Multi-device pattern** — register both your YubiKey 5 and your iPhone passkey; either gets you in.
- **Don't rely on a single key** — drop, lose, drown — bricked account.
- **Enterprise rollout** — distribute 2 keys per employee; require both registered before retiring password.

```bash
# Reset / list FIDO2 credentials on YubiKey
ykman fido credentials list
ykman fido reset    # WIPES ALL FIDO2 CREDS
```

## OAuth 2.1 / OIDC — Auth Code + PKCE

- **OAuth 2.1** — consolidation of best practices from OAuth 2.0 + 8 years of erratas. Drops implicit + ROPC flows.
- **OIDC (OpenID Connect)** — identity layer on top of OAuth. Adds `id_token` for AuthN.
- **The flow** (the only browser flow in 2024):
  1. Client generates `code_verifier` (43-128 chars, URL-safe random) and `code_challenge = base64url(sha256(verifier))`.
  2. Redirect user to AS: `GET /authorize?response_type=code&client_id=...&redirect_uri=...&scope=openid+profile&state=<csrf>&nonce=<replay>&code_challenge=...&code_challenge_method=S256`.
  3. User authenticates + consents at AS.
  4. AS redirects back: `GET <redirect_uri>?code=<auth_code>&state=<echo>`.
  5. Client validates `state` matches, sends `POST /token` with `code`, `code_verifier`, `redirect_uri`, `client_id`.
  6. AS verifies `sha256(code_verifier) == stored code_challenge`; returns `access_token`, `id_token`, `refresh_token`.
  7. Client validates `id_token` signature, `aud`, `iss`, `nonce`.
- **Implicit flow is dead** — fragments-with-tokens leak via Referer, browser history, analytics.
- **ROPC (resource owner password)** — DEAD; no client should ever ask for the user's password.
- **Client credentials flow** — still alive for machine-to-machine; not for user auth.
- **Device code flow** — for input-constrained devices (TV, CLI); see RFC 8628.

```bash
# Build an auth URL (browser flow)
verifier=$(openssl rand -base64 64 | tr -d '=+/' | cut -c1-64)
challenge=$(printf %s "$verifier" | openssl dgst -sha256 -binary | openssl base64 | tr -d '=+/\n' | tr '/+' '_-')
echo "https://auth.example.com/authorize?response_type=code&client_id=APP&redirect_uri=https://app.example.com/cb&scope=openid+profile+email&state=abc&nonce=def&code_challenge=$challenge&code_challenge_method=S256"
```

## OAuth 2.1 / OIDC — Refresh, Revoke, Introspect

- **Refresh token rotation** — `POST /token` with `grant_type=refresh_token&refresh_token=...&client_id=...`; AS returns new access + new refresh, old refresh dies.
- **Revocation endpoint (RFC 7009)** — `POST /revoke` with token; AS marks token invalid. Use on logout, password change, suspected compromise.
- **Introspection endpoint (RFC 7662)** — `POST /introspect` with token; AS returns `{active: true/false, sub, scope, exp, ...}`. Use for opaque-access-token validation by RS.
- **Userinfo endpoint** — OIDC-only: `GET /userinfo` with bearer access token; returns user claims.
- **JWKs endpoint** — `/.well-known/jwks.json` — RP fetches public keys for id_token verification. Cache + honor `Cache-Control` + key rotation.
- **Discovery endpoint** — `/.well-known/openid-configuration` — bootstrap for all the above URLs.

```bash
curl -s https://auth.example.com/.well-known/openid-configuration | jq .
# Returns {authorization_endpoint, token_endpoint, jwks_uri, ...}
```

## OAuth 2.1 / OIDC — Client Types

- **Confidential client** — server-side; can keep a secret. Authenticates to AS with `client_secret` or signed JWT (`client_assertion`).
  - Examples: server-rendered web app, backend service.
  - Use full client auth (secret/mtls/private_key_jwt).
- **Public client** — runs on user device, can't keep a secret. Browser SPA, mobile/native app.
  - **Must use PKCE** — code-injection defense replaces the missing client secret.
  - Don't ship a "client_secret" in your APK/JS bundle — it's not a secret.
- **Dynamic client registration (RFC 7591)** — for federated/per-tenant scenarios.
- **mTLS client auth (RFC 8705)** — banking-grade alternative to client_secret; client cert binds to access token.

## OAuth 2.1 / OIDC — Scopes

- **Least privilege** — request only what you need; `openid profile email` is a sane default for SSO.
- **Granular consent** — break "calendar.read" from "calendar.write"; users approve narrower grants.
- **UX implication** — every scope appears on the consent screen; bloat = more abandons + creepier feeling.
- **Incremental authorization** — request basic scopes at signup; ask for `drive.readonly` only when user clicks "import from Drive".
- **Naming convention** — `<resource>.<action>` (e.g., `mail.read`, `calendar.write`); some providers use URLs (`https://www.googleapis.com/auth/calendar`).
- **Scope ≠ Authorization** — scope is what the user delegated; the API still enforces ACLs ("user can read this calendar?"). Don't treat scope alone as the gate.

## OAuth 2.1 / OIDC — id_token vs access_token

- **id_token (OIDC)** — JWT for **the client**, proves "this user authenticated".
  - Verified by client; contains `sub`, `email`, `name`, `iat`, `exp`, `iss`, `aud=client_id`, `nonce`.
  - **Never send id_token to APIs** — it's not a credential for resource access; APIs probably won't validate `aud`, leading to confused-deputy attacks.
- **access_token (OAuth)** — bearer for **the resource server**, proves "client X is allowed scope Y on behalf of user Z".
  - Format: opaque (introspect) or JWT (verify locally).
  - `aud` = the API audience; `scope` = what's permitted; `sub` = end user.
- **refresh_token** — credential to mint new access tokens; never sent to RS.
- **Key invariants**
  - id_token: client validates → checks `aud == own client_id`
  - access_token: RS validates → checks `aud == own resource id`
  - Mixing them = bug: if RS accepts id_tokens, anyone with a Google id_token for any app can hit your API.

## OAuth 2.1 / OIDC — Provider Quirks

- **Google** — uses email as identifier; `sub` is stable, but verify `email_verified=true`. Discovery: `https://accounts.google.com/.well-known/openid-configuration`.
- **GitHub** — pre-OIDC OAuth 2.0; access token only, no id_token. Use `/user` and `/user/emails` endpoints. Scopes are coarse (`user:email`, `repo`).
- **Microsoft Entra ID (Azure AD)** — supports v1 and v2 endpoints; v2 is OIDC-compliant. Multi-tenant via `common` issuer; validate tenant id (`tid` claim) explicitly.
- **Okta** — full OIDC; per-tenant issuer URL. Use `okta-jwt-verifier-*` libraries.
- **Auth0** — full OIDC + many extension flows; `auth0` SDK or generic OIDC client.
- **Apple Sign In** — uniquely strict: nonce required, refresh token only on initial flow; private-relay emails (`@privaterelay.appleid.com`).
- **Facebook** — OAuth 2.0, not OIDC; their own `/me` endpoint.

## OAuth 2.1 / OIDC — state + nonce

- **state** — CSRF protection. Random per-flow value the client generates; AS echoes it; client checks match.
  - Generate: `secrets.token_urlsafe(32)`. Store in session cookie or PKCE session. **Required by OAuth 2.1.**
  - Without `state`, an attacker initiates an auth flow with their own AS account and tricks victim into completing it — victim's app session ends up linked to attacker's identity.
- **nonce** (OIDC only) — replay protection for id_token.
  - Generate similarly; client sends in `/authorize`; AS includes in id_token; client verifies.
  - Without `nonce`, an attacker who captures a valid id_token can replay against your client.
- **PKCE `code_verifier`** — different protection: ensures the code redeemed at the token endpoint matches the one that authorized — defends against code interception.
- **All three together** — `state` + `nonce` + PKCE = mandatory for browser auth flows.

```python
state = secrets.token_urlsafe(32)
nonce = secrets.token_urlsafe(32)
session["oauth_state"] = state
session["oauth_nonce"] = nonce
# Build /authorize URL with state, nonce, code_challenge

# On callback
assert request.args["state"] == session.pop("oauth_state")
# After id_token verify:
assert id_claims["nonce"] == session.pop("oauth_nonce")
```

## OAuth 2.1 / OIDC — Logout

- **RP-initiated logout (front-channel)** — RP redirects user to OP's `end_session_endpoint`; OP terminates SSO session, redirects back. User sees a flash; works in any browser.
- **Back-channel logout** — OP POSTs a `logout_token` JWT to each registered RP; RPs invalidate their local sessions server-side. No browser hop, but requires every RP to be reachable from OP.
  - `logout_token` claims: `iss`, `aud`, `iat`, `jti`, `events: {"http://schemas.openid.net/event/backchannel-logout": {}}`, `sub` and/or `sid`.
- **Front-channel logout** — OP renders an iframe pointing to each RP's logout URL; relies on third-party cookies (increasingly broken in browsers).
- **Best practice (2024)** — back-channel for trusted RPs; RP-initiated for browser-driven flows. Don't rely on front-channel iframes.
- **Local logout vs SLO (Single Logout)** — clarify with users: "log out of this app" vs "log out everywhere".

## CSRF — Concept & Defenses

- **What it is** — attacker tricks an authenticated user's browser into making a side-effect request to your app (POST /transfer, DELETE /post). Browser auto-attaches the user's cookies; server can't distinguish.
- **Primary defense — `SameSite` cookies**
  - `Strict` — fully blocks cross-site auto-send.
  - `Lax` — blocks cross-site POST/iframe but allows top-level GET (browser default since 2021).
  - **Cookie-CSRF is largely solved by `SameSite=Lax/Strict`**, but only against modern browsers. Defense-in-depth still needed for legacy + edge cases.
- **CSRF token (synchronizer)** — server generates per-session token, embeds in form, validates on submit. Server-stateful.
- **Double-submit cookie** — server sets random cookie + same value in form/header; on submit, server checks they match. Stateless.
  - Variant: signed double-submit (sign with server secret) — defends against subdomain-set cookies.
- **Origin / Referer header check** — defense-in-depth: reject if `Origin` or `Referer` doesn't match expected. Don't rely solely (some browsers strip).
- **Custom request header (e.g., `X-CSRF`)** — works because browsers can't make cross-origin custom-headered requests without CORS preflight. Pair with CORS config that doesn't allow the relevant origin.
- **GET should not have side effects** — basic REST hygiene; CSRF protection is harder for stateful GETs and they're not idempotent anyway.

## CSRF — Framework Patterns

- **Django** — `{% csrf_token %}` in form, `CsrfViewMiddleware` validates `csrfmiddlewaretoken`. AJAX: send `X-CSRFToken` header from cookie.
- **Rails** — `protect_from_forgery with: :exception` (default in 5.2+); `<%= csrf_meta_tags %>` for AJAX.
- **Express (Node)** — `csurf` (deprecated 2022; community forks); modern apps use double-submit via `cookie-parser` + custom middleware.
- **Flask** — `flask-wtf` `CSRFProtect`; per-form token.
- **Spring Security** — CSRF on by default for browser apps; disable for stateless APIs.
- **SPA + JWT in HttpOnly cookie + custom header pattern**
  - Server issues HttpOnly cookie with session/JWT.
  - Client also gets a non-HttpOnly CSRF cookie (or reads from meta tag).
  - Every state-changing request includes both: cookie (auto) + `X-CSRF` header (JS-set).
  - Server validates: cookie matches header (and matches stored value if synchronizer).
- **The `Authorization: Bearer X` exemption** — CSRF doesn't apply because browsers don't auto-attach Authorization headers to cross-origin requests. **But** only if your auth is *purely* the bearer header — if a session cookie is also set and accepted, you're CSRF-vulnerable.

```python
# Flask + WTF
from flask_wtf.csrf import CSRFProtect
csrf = CSRFProtect(app)

# Skip CSRF for true bearer-token API
@csrf.exempt
@app.route("/api/v1/orders", methods=["POST"])
def create_order():
    user = require_bearer_token()
    ...
```

## CSRF — Exemptions & Misconceptions

- **API-only endpoints with `Authorization: Bearer`** — exempt from CSRF (no cookies, no auto-attach).
- **API-only endpoints with cookie auth** — NOT exempt. The "I'm an API" claim doesn't matter; the browser sees a cookie.
- **"But it's POST with JSON, the browser can't forge that"** — **misconception**. An attacker can `<form enctype="text/plain" action="https://victim.com/api">` with one input named `{"a":"b","ignored":"` — produces a body that's *valid JSON* with a "junk" key. Servers that don't check `Content-Type` are CSRFable. Defenses:
  - Require `Content-Type: application/json` and reject anything else.
  - Add CSRF token / SameSite anyway.
- **Login CSRF** — attacker logs you into *their* account; subsequent activity (saved searches, payment methods) attaches to attacker. Defend by including CSRF token on the login form too.
- **CORS does NOT defend against CSRF** — CORS controls what JS can READ from cross-origin responses; nothing about whether the request is sent.
- **`SameSite=None` requires `Secure`** — if you intentionally set `None`, you've opted into CSRF risk; protect with token + custom header.

## Identity Federation — SAML

- **SAML 2.0** — XML-based federation protocol; pre-OIDC, still dominant in enterprise.
- **Roles**
  - **SP (Service Provider)** — your app
  - **IdP (Identity Provider)** — Okta, Microsoft AD FS, PingFederate, Auth0, Shibboleth
- **Flow (SP-initiated)**
  1. User hits SP, no session.
  2. SP redirects to IdP with SAMLRequest (signed, optionally encrypted).
  3. IdP authenticates user.
  4. IdP returns SAMLResponse via HTTP POST binding to SP's ACS URL.
  5. SP validates SAMLResponse signature, extracts assertion, creates session.
- **SAML response signing** — the assertion (or response) is XML-DSig signed by IdP private key; SP verifies with IdP's public cert from metadata.
- **XML signature wrapping (XSW) attack** — attacker re-orders/wraps signed elements so a different element is signed than the one parsed. Mitigations:
  - Use a SAML library that handles XSW correctly (`python3-saml`, `OneLogin SAML toolkits`).
  - Validate that the signed element is the asserted one (check XPath, not just "any signature is valid").
- **SAML vs OIDC**
  - SAML — XML, browser-only (HTTP Redirect/POST bindings), enterprise-heavy.
  - OIDC — JSON/JWT, works for browser + mobile + service-to-service, modern.
  - **Choose OIDC** for new builds; support SAML when you sell to enterprise.
- **NameID formats** — `emailAddress`, `persistent` (opaque pseudonym), `transient` (one-shot). Persistent is the right default.
- **Replay protection** — SAML assertions have `NotBefore`/`NotOnOrAfter`; nonce in `ID`; SP must track recent IDs to prevent replay.

## Identity Federation — JIT Provisioning

- **Just-in-time (JIT) provisioning** — on first SAML/OIDC login, auto-create a local account from the assertion claims.
- **Mapping**
  - `email` → users.email (verify the IdP asserted it as verified)
  - `given_name`, `family_name` → users.first_name, last_name
  - `groups` (array claim) → role mapping table
- **Group → Role mapping** — config table: `idp_group="dev-frontend" → role="developer"`. Reapply on every login (groups can shrink).
- **Default role** — if no group maps, assign least-privileged ("read-only") not most.
- **Attribute drift** — re-sync first/last name on every login; some users hate this (renames not respected). Provide override.
- **Don't auto-confirm email** — the IdP says they're trusted, but for security-sensitive emails (like recovery channel) require explicit verification on first link.

## Identity Federation — Deprovisioning & SCIM

- **Just-in-time deprovisioning is hard** — JIT is "user shows up, we create"; nothing fires "user left" client-side.
- **Patterns**
  - **Login-only check** — at next login, check IdP says user is still active. If not, deny. **But** they could already have a 24h session.
  - **Periodic IdP poll** — daily cron, hit IdP's "list users" API, deactivate locally if missing. Works only if IdP exposes that.
  - **SCIM (System for Cross-domain Identity Management, RFC 7643/7644)** — IdP **pushes** user lifecycle events to SP via REST: create, update, deactivate.
- **SCIM endpoints**
  - `POST /Users` — provision
  - `PATCH /Users/{id}` — update (incl. `active=false` deprovision)
  - `DELETE /Users/{id}` — hard delete
  - `GET /Users` — bulk sync
- **Session invalidation on deprovision** — receiving `active=false` must invalidate all sessions/tokens for that user immediately, not at next login.
- **OWASP — offboarding gaps are the most common breach vector for ex-employees.**
- **SCIM is the standard SaaS expectation** — Okta, OneLogin, Azure AD all push via SCIM if your endpoint speaks it.

```http
PATCH /scim/v2/Users/abc123
Content-Type: application/scim+json
Authorization: Bearer <scim-token>

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [{"op": "replace", "path": "active", "value": false}]
}
```

## Recovery Flows

- **The trade-off** — security ↔ usability. Make recovery easier and account takeover gets easier too.
- **Anti-pattern — secret questions**
  - "Mother's maiden name?" "First pet?" — public knowledge or guessable.
  - If you must, treat answers as passwords (case-sensitive, hashed) and let users invent question/answer pairs.
- **Recovery email** — de-facto root account. Treat with same rigor as the login itself: verify on add/change, MFA-protect the email account itself.
- **Recovery phone** — same caveats as SMS MFA — SIM swap risk.
- **Passwordless recovery via WebAuthn** — register a passkey; recovery becomes "use your passkey on a known device". Best-of-class.
- **Multi-channel recovery** — require both email-token AND phone-OTP for high-value resets.
- **Manual support recovery** — last resort, requires identity proof; document the SOP, audit every use, never let support staff have unilateral reset.
- **Recovery rate limit** — same as password reset: per-account + per-IP.
- **Cooldown after credential change** — block recovery for 24h after a password change ("change locks the account from being reset for a day").
- **Notify the user** — every recovery attempt = email to the account.

## Magic Link Login

- **Passwordless via email** — user enters email; gets a link with single-use token; clicking logs in.
- **Token TTL** — **<15 min**. Long-living magic links forwarded by users = future ATO.
- **Token entropy** — same as reset: 32 bytes from CSPRNG.
- **One-time-use** — invalidate on first redemption.
- **Rate limit** — per-email and per-IP; magic-link spam is a thing.
- **SameSite caveat** — clicking a link from email is a top-level navigation. `SameSite=Lax` cookies will be sent (by design); `SameSite=Strict` won't (the user shows up "logged out" because their existing session cookie isn't sent on cross-site nav). Magic-link flows generally need `Lax`.
- **Different-device problem** — user requests on phone, opens link on laptop. Pattern: link redeemed on whatever device opens it (preferred); or include a "device code" the requesting device polls.
- **Phishability** — magic links can be relayed by attackers (phish for the email, prompt-bomb until victim clicks). Pair with a per-device confirmation ("did you request this from <UA, IP, geo>?").
- **Don't combine with password** — pick one mode per account; mixing makes UX confusing and security harder.

## Account Linking

- **Use case** — user signs up with email/password, later wants to "Sign in with Google" mapped to same account.
- **The email-collision attack**
  1. Attacker signs up locally with `victim@gmail.com` + their own password (email unverified).
  2. Real victim later does "Sign in with Google" using the same address.
  3. SP's auto-link logic: "we already have a user with this email; link the Google identity to that account".
  4. Attacker now has both their own creds + victim's Google identity on the same account → ATO via attacker's password.
- **Defenses**
  - **Verify the local email** — never link to an unverified-email local account.
  - **Verify the IdP-asserted email** — Google's `email_verified=true`, Apple/Microsoft equivalents.
  - **Require re-auth** — when linking, prompt for the existing account's password / passkey.
  - **Notify both** — email both the local account and the IdP email of the linkage.
- **Unlinking** — UI to disconnect a federated identity; require alternative auth still works first.
- **Allow multiple IdPs per user** — store as `(user_id, provider, provider_user_id)` rows.

## Privilege Escalation Paths

- **Sudo / step-up auth** — for sensitive actions (delete account, change email, view PII), require fresh auth (last 5 min) even if logged in.
  - Prompt for password / passkey / MFA at action-time, not session-start.
- **"Act as user" (impersonation)** — admin views the app as another user.
  - **Require admin re-auth** — prompt for admin's password/MFA before starting impersonation, regardless of session age.
  - **Audit log entry** — `actor=admin@…, target=user@…, started_at=..., reason=...`. Reason field MANDATORY.
  - **Banner** — visible "you are impersonating X" bar throughout session; one-click stop.
  - **Session distinct** — separate session ID; on stop, return to admin's original session, don't dual-use.
  - **Time-box** — auto-expire impersonation after 30 min absent extension.
  - **Restricted actions** — even as user, certain actions blocked (changing user's password, exporting their data) without explicit user consent.
- **Privilege escalation via role-mapping flaws** — "groups: ['admin']" claim in a JWT issued by a less-trusted service; never trust unless issued by the role authority.
- **Vertical (user → admin) and horizontal (user A → user B)** — both are escalation; AuthZ-on-every-request defends both.

## Audit Logging

- **What to log (every auth event)**
  - Login success / failure (with reason: bad-password, account-locked, mfa-fail)
  - MFA challenge / success / failure / fallback to recovery code
  - Password changes / reset requested / reset completed
  - Email / phone changes (the recovery channel changes!)
  - Role / permission changes
  - Session creation / destruction / rotation
  - Token issuance / revocation
  - Federated login (IdP, sub, aud)
  - Account linking / unlinking
  - "Act as user" start / stop
  - Lockouts triggered
  - Suspicious-activity alerts
- **Fields per event** — `timestamp_iso8601`, `event_type`, `user_id`, `actor_id` (for impersonation), `ip`, `user_agent`, `outcome`, `reason`, `request_id`, `geo`, `risk_score`.
- **NEVER log**
  - Passwords (plain or hashed)
  - Session IDs / tokens / JWTs (full)
  - MFA codes (TOTP, SMS, recovery)
  - Secret answers
  - PAN / SSN / card numbers
- **Truncate or hash sensitive identifiers** — log `sha256(token)[:16]` if needed for correlation.
- **Structured logs (JSON)** — feed SIEM (Splunk, Datadog, ELK).
- **Retention** — minimum 90 days for forensics; 1 year for compliance (SOX, HIPAA depend on jurisdiction).
- **Tamper-evidence** — log to append-only sink (CloudWatch Logs immutability, AWS S3 Object Lock); for high stakes, sign each event chain (hash of prev + this).
- **Alerting** — push to SOC: high failure rate, login from new geo, impersonation start, role change.

```json
{
  "ts": "2024-01-15T14:23:01Z",
  "event": "auth.login.success",
  "user_id": 12345,
  "session_id_prefix": "9f2a...",
  "ip": "203.0.113.5",
  "ua": "Mozilla/5.0 (Macintosh; ...) Chrome/120",
  "geo": {"country": "US", "city": "Seattle"},
  "method": "password+totp",
  "request_id": "req-abc123"
}
```

## Common Errors — What Attackers Probe

- **Account-not-found timing** — "user not found" returns in 5ms; valid-user-bad-pw returns in 250ms. Diff = enumeration.
- **Response size diff** — same status code, but `Content-Length: 47` vs `Content-Length: 92`. Build user list via size oracle.
- **Error-message specificity**
  - "Wrong password for that account" → enumeration
  - "Your account has been locked" → enumeration + lockout state leak
  - **Always**: "Invalid email or password"
- **HTTP status code** differences — 200 OK vs 401 vs 404 between paths.
- **Redirects with parameters** — `?error=user_not_found&email=x@y` after callback.
- **Set-Cookie behavior** — only-on-success cookie reveals success.
- **Differences in security headers** — `WWW-Authenticate: Basic realm="users"` vs `realm="admins"`.
- **WAF / IDS detection ideas**
  - Rule: same source IP, >50 distinct usernames in 60s → block (credential stuffing).
  - Rule: >5 failed logins same username, distinct IPs (>3 ASNs) → flag (distributed brute force).
  - Rule: response payload size diff between login attempts from same IP → alert (enumeration probe).
  - Rule: any login from country X never seen before → step-up MFA.
  - Rule: JWT with `alg=none` or `alg=HS256` when expecting `RS256` → block + alert.
  - Rule: User-Agent containing known credential-stuffing tools (`OpenBullet`, `SilverBullet`, `Sentry MBA`) → block.

## Common Gotchas — Broken vs Fixed

- **Storing password with SHA-256 + salt**
  ```python
  # BAD
  stored = sha256(salt + pw).hexdigest()
  # GOOD
  stored = ph.hash(pw)  # argon2id
  ```

- **Comparing HMAC with `==` (timing attack)**
  ```python
  # BAD
  if computed_mac == provided_mac:
  # GOOD
  if hmac.compare_digest(computed_mac, provided_mac):
  ```

- **JWT verify with `algorithms=None`**
  ```python
  # BAD — accepts alg=none
  jwt.decode(token, key, algorithms=None)
  # GOOD
  jwt.decode(token, key, algorithms=["RS256"])
  ```

- **Cookie without `SameSite`**
  ```http
  # BAD
  Set-Cookie: sid=abc; HttpOnly; Secure
  # GOOD
  Set-Cookie: __Host-sid=abc; Path=/; HttpOnly; Secure; SameSite=Lax
  ```

- **Same response on valid+invalid login but timing diff**
  ```python
  # BAD — returns fast on missing user
  user = db.find(email)
  if not user: return fail()
  if not ph.verify(user.hash, pw): return fail()
  # GOOD — constant-time on miss
  user = db.find(email)
  ph.verify(user.hash if user else DUMMY_HASH, pw)
  if not user or not ok: return fail()
  ```

- **Predictable reset token**
  ```python
  # BAD
  token = str(random.random())
  # BAD
  token = hashlib.md5(f"{user_id}-{time.time()}".encode()).hexdigest()
  # GOOD
  token = secrets.token_urlsafe(32)
  ```

- **User-controlled redirect after login (open redirect)**
  ```python
  # BAD — phishing pivot
  return redirect(request.args["next"])
  # GOOD — allowlist
  next_url = request.args.get("next", "/")
  if not next_url.startswith("/") or next_url.startswith("//"):
      next_url = "/"
  return redirect(next_url)
  ```

- **2FA bypass on password-reset**
  ```python
  # BAD — reset path skips 2FA, attacker resets pw and is in
  def consume_reset(token, new_pw):
      user = lookup(token)
      user.set_password(new_pw)
      log_in(user)  # no MFA!
  # GOOD — reset only changes pw; user must complete login including MFA
  def consume_reset(token, new_pw):
      user = lookup(token)
      user.set_password(new_pw)
      return redirect("/login")  # MFA happens here
  ```

- **SMS for high-value MFA**
  ```text
  # BAD — SIM swap = ATO
  Send 6-digit code via SMS for bank login
  # GOOD
  TOTP / WebAuthn / hardware token; SMS only for low-value or step-up fallback
  ```

- **JWT in `localStorage` with no XSS defense**
  ```javascript
  // BAD — any XSS = total token theft
  localStorage.setItem("jwt", token);
  // GOOD — HttpOnly cookie set by server
  // Set-Cookie: __Host-sid=...; HttpOnly; Secure; SameSite=Lax
  // Plus strict CSP: default-src 'self'; script-src 'self'
  ```

- **Logging passwords / tokens**
  ```python
  # BAD
  log.info("login attempt", extra={"email": email, "password": pw})
  # GOOD
  log.info("login attempt", extra={"email": email, "outcome": "success"})
  ```

- **No rate limit on login**
  ```python
  # BAD
  @app.post("/login")
  def login(): ...
  # GOOD — per-username + per-IP
  @app.post("/login")
  @rate_limit(per_user=5, per_minute=15, per_ip=100, per_minute=1)
  def login(): ...
  ```

- **Trusting `Host` header for reset URLs**
  ```python
  # BAD — host-header injection redirects reset to attacker.com
  url = f"https://{request.host}/reset?t={token}"
  # GOOD — pin canonical
  url = f"{settings.BASE_URL}/reset?t={token}"
  ```

- **Failure to verify IdP-asserted email**
  ```python
  # BAD — trust whatever Google says
  user = User.objects.get_or_create(email=id_token["email"])
  # GOOD
  if not id_token.get("email_verified"):
      raise Forbidden()
  user = User.objects.get_or_create(email=id_token["email"])
  ```

- **OAuth callback without `state` validation**
  ```python
  # BAD — login CSRF
  code = request.args["code"]
  exchange_code(code)
  # GOOD
  if request.args["state"] != session.pop("oauth_state"):
      raise BadRequest()
  ```

- **Forgot to invalidate sessions on password change**
  ```python
  # BAD — old stolen sessions still valid
  user.set_password(new_pw)
  # GOOD
  user.set_password(new_pw)
  user.session_version += 1  # invalidates all sessions
  user.save()
  ```

## Recommended Stack

- **Password storage** — Argon2id with library defaults (memory ≥19 MiB, time ≥2, parallelism ≥1)
- **Password policy** — 14-char minimum, 64 max, no composition rules, HIBP check, `zxcvbn` strength meter (advisory only)
- **Pepper** — HMAC-SHA-256 with vault-stored 32-byte secret, applied before Argon2id
- **MFA** — WebAuthn/passkey primary; TOTP secondary; SMS only as last-resort step-up; 10 hashed recovery codes
- **Session** — server-side, opaque ID (32 random bytes), Redis-backed; idle timeout 30 min, absolute 8h; rotation on auth and on privilege change
- **Cookies** — `__Host-` prefix, `HttpOnly`, `Secure`, `SameSite=Lax` (or `Strict` for admin)
- **CSRF** — `SameSite=Lax` + double-submit token + custom-header check on JSON APIs
- **OAuth/OIDC** — auth-code + PKCE only; mandatory `state` + `nonce`; refresh token rotation with reuse-detection
- **JWT** (only when needed) — RS256 or EdDSA; pinned `alg`; verify `iss/aud/exp/nbf/iat`; 15-min access + 7-day refresh
- **Rate limiting** — per-username (5/15min), per-IP (100/min), exponential backoff, alert+lock on patterns
- **Audit logging** — structured JSON, every auth event, never passwords/tokens; append-only sink
- **Federation** — OIDC over SAML; SCIM for lifecycle if enterprise
- **Recovery** — verified email + recovery codes (hashed); WebAuthn passkey on a second device; never security questions
- **Headers** — `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload`, `Content-Security-Policy: default-src 'self'`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`

## Idioms

- **Boring secure defaults** — pick well-trodden libraries; resist the urge to "improve" auth.
- **Never roll your own crypto / auth** — every line of custom auth is a vulnerability waiting; use Spring Security, Devise, Auth.js, Authlib, ory/kratos, dex, keycloak, Auth0, Okta, Clerk, Supabase Auth.
- **Reach for OIDC + library** before building login forms.
- **Passkey-first UX** — let users skip passwords entirely if they have a passkey-capable device.
- **Make rotation cheap** — `password_needs_rehash`, `session_version`, JWT key rotation. The day you need to rotate a parameter, you'll thank past-you.
- **Defense in depth** — `SameSite` AND CSRF token AND custom header AND CORS. Layers.
- **Audit first** — turn on logging before turning on auth. The day after launch, you need to be able to reconstruct what happened.
- **Test the unhappy paths** — locked account, expired token, wrong-MFA, replay, race-conditions.
- **Threat-model in code review** — for any auth-touching PR, ask: which OWASP attack does this open or close?
- **Re-auth for sensitive actions** — sudo mode is non-negotiable for password change, email change, payout, "log out everywhere".
- **Document the model** — a one-page diagram of "session lifecycle" + "MFA flow" + "recovery flow" beats 200 wiki pages nobody reads.
- **Bug bounty + pen-test** — auth subsystems benefit from external eyes more than almost any other code.
- **Patch promptly** — auth lib CVEs ship monthly; you want CI to flag stale deps daily.

## See Also

- tls
- ssh
- openssl
- vault
- owasp-injection
- gpg
- polyglot
- javascript
- python
- go

## References

- OWASP Cheat Sheet Series — `https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html`
- OWASP Session Management — `https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html`
- OWASP Password Storage — `https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html`
- OWASP JWT — `https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html`
- OWASP OAuth 2.0 — `https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html`
- OWASP Multifactor Authentication — `https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html`
- OWASP Forgot Password — `https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html`
- OWASP Credential Stuffing Prevention — `https://cheatsheetseries.owasp.org/cheatsheets/Credential_Stuffing_Prevention_Cheat_Sheet.html`
- OWASP CSRF Prevention — `https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html`
- OWASP ASVS v4.0.3 — `https://owasp.org/www-project-application-security-verification-standard/`
- NIST SP 800-63B Digital Identity Guidelines — `https://pages.nist.gov/800-63-3/sp800-63b.html`
- RFC 6238 — TOTP — `https://datatracker.ietf.org/doc/html/rfc6238`
- RFC 4226 — HOTP — `https://datatracker.ietf.org/doc/html/rfc4226`
- RFC 7519 — JWT — `https://datatracker.ietf.org/doc/html/rfc7519`
- RFC 7515 — JWS — `https://datatracker.ietf.org/doc/html/rfc7515`
- RFC 7517 — JWK — `https://datatracker.ietf.org/doc/html/rfc7517`
- RFC 6749 — OAuth 2.0 — `https://datatracker.ietf.org/doc/html/rfc6749`
- OAuth 2.1 draft — `https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1`
- RFC 7636 — PKCE — `https://datatracker.ietf.org/doc/html/rfc7636`
- RFC 7009 — OAuth Token Revocation — `https://datatracker.ietf.org/doc/html/rfc7009`
- RFC 7662 — OAuth Token Introspection — `https://datatracker.ietf.org/doc/html/rfc7662`
- RFC 8628 — Device Authorization Grant — `https://datatracker.ietf.org/doc/html/rfc8628`
- OpenID Connect Core 1.0 — `https://openid.net/specs/openid-connect-core-1_0.html`
- OIDC Back-Channel Logout — `https://openid.net/specs/openid-connect-backchannel-1_0.html`
- WebAuthn L3 — `https://www.w3.org/TR/webauthn-3/`
- webauthn.guide — `https://webauthn.guide`
- FIDO Alliance — `https://fidoalliance.org`
- HIBP Pwned Passwords API — `https://haveibeenpwned.com/API/v3#PwnedPasswords`
- Argon2 RFC 9106 — `https://datatracker.ietf.org/doc/html/rfc9106`
- SCIM RFC 7644 — `https://datatracker.ietf.org/doc/html/rfc7644`
- SAML 2.0 Core — `http://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf`
