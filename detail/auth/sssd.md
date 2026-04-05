# SSSD — Process Architecture, Provider Internals, and Offline Authentication Design

> *SSSD (System Security Services Daemon) implements a modular multi-process architecture where a monitor daemon supervises independent responder and backend processes. Each identity domain runs in its own backend process with pluggable providers for identity, authentication, access control, and password changes. The LDB cache provides LDAP-like local storage with offline credential verification, enabling authentication when network connectivity is lost.*

---

## 1. Process Architecture

### Multi-Process Design

SSSD runs as multiple cooperating processes rather than a single monolithic daemon. This provides fault isolation -- a crash in the PAM responder does not affect NSS lookups.

```
Process Tree:
  sssd (monitor)                    PID 1 of SSSD — watchdog + config
  ├── sssd_nss                      NSS responder (getpwnam, getgrnam)
  ├── sssd_pam                      PAM responder (authenticate, acct_mgmt)
  ├── sssd_sudo                     Sudo responder (sudo rule lookups)
  ├── sssd_ssh                      SSH responder (authorized keys from LDAP)
  ├── sssd_autofs                   Autofs responder (automount maps)
  ├── sssd_ifp                      InfoPipe responder (D-Bus interface)
  └── sssd_be (per domain)          Backend process for domain "example.com"
      ├── id_provider plugin        Identity lookups
      ├── auth_provider plugin      Authentication
      ├── access_provider plugin    Authorization
      └── chpass_provider plugin    Password changes
```

### Monitor Process

The monitor (`sssd` binary) is the parent process responsible for:

```
Monitor responsibilities:
  1. Parse /etc/sssd/sssd.conf at startup
  2. Fork and exec each responder process (nss, pam, sudo, etc.)
  3. Fork and exec each backend process (one per [domain/...] section)
  4. Health-check child processes via periodic ping (every 10s default)
  5. Restart crashed children (up to restart limit, then disable domain)
  6. Handle SIGHUP for config reload (limited — most changes need restart)
  7. Manage shared memory maps used by fast NSS cache (memcache)

Restart policy:
  Child crash → immediate restart
  3 crashes in 60s → disable that responder/domain, log error
  Manual restart: systemctl restart sssd
```

### Inter-Process Communication

Responders and backends communicate through UNIX domain sockets:

```
Communication paths:
  Application → NSS/PAM → sssd_nss/sssd_pam (via NSS module / PAM module)
                                │
                    UNIX socket │ (D-Bus-like protocol over socket)
                                │
                                ▼
                          sssd_be (backend)
                                │
                    Network     │ (LDAP, Kerberos, etc.)
                                ▼
                        Remote server (AD, LDAP, IPA)

Socket locations:
  /var/lib/sss/pipes/nss          — NSS responder
  /var/lib/sss/pipes/pam          — PAM responder (privileged)
  /var/lib/sss/pipes/sudo         — sudo responder
  /var/lib/sss/pipes/private/     — backend communication (root only)
```

### Fast NSS Cache (memcache)

For frequently looked-up users and groups, SSSD maintains a shared memory cache that the NSS module reads directly without contacting the sssd_nss process:

```
Memcache architecture:
  Application calls getpwnam("user1")
      │
      ▼
  libnss_sss.so (loaded into application address space)
      │
      ├── Check shared memory map (/var/lib/sss/mc/passwd)
      │   └── HIT → return immediately (no IPC, no context switch)
      │
      └── MISS → send request to sssd_nss over UNIX socket
          └── sssd_nss checks LDB cache
              ├── HIT → return from cache, update memcache
              └── MISS → forward to sssd_be → LDAP/AD lookup

Memcache files:
  /var/lib/sss/mc/passwd   — passwd entries (hash table, mmap'd)
  /var/lib/sss/mc/group    — group entries
  /var/lib/sss/mc/initgr   — initgroups results (user → group list)

Performance impact:
  With memcache:    ~1 microsecond per lookup (mmap read)
  Without memcache: ~100 microseconds (IPC to sssd_nss)
  Remote lookup:    ~5-50 milliseconds (network round-trip)
```

## 2. Identity / Auth / Access Provider Model

### Provider Plugin Architecture

Each domain configures independent providers for different functions. Providers are compiled as shared libraries loaded by the backend process:

```
Provider types and their responsibilities:
  ┌────────────────┬───────────────────────────────────────────────────┐
  │ id_provider    │ User/group identity: getpwnam, getgrnam,         │
  │                │ getpwuid, getgrgid, initgroups                   │
  ├────────────────┼───────────────────────────────────────────────────┤
  │ auth_provider  │ Password verification, Kerberos ticket           │
  │                │ acquisition, OTP validation                      │
  ├────────────────┼───────────────────────────────────────────────────┤
  │ access_provider│ Login authorization: is this user allowed to     │
  │                │ access this host? (HBAC, GPO, simple, LDAP)      │
  ├────────────────┼───────────────────────────────────────────────────┤
  │ chpass_provider│ Password change operations (passwd command)      │
  ├────────────────┼───────────────────────────────────────────────────┤
  │ sudo_provider  │ Sudo rule retrieval from LDAP/IPA                │
  ├────────────────┼───────────────────────────────────────────────────┤
  │ autofs_provider│ Automount map retrieval from LDAP/IPA            │
  └────────────────┴───────────────────────────────────────────────────┘

Provider implementations:
  ldap  — Generic LDAP (OpenLDAP, 389DS, etc.)
  ad    — Active Directory (extends ldap with AD-specific logic)
  ipa   — FreeIPA (extends ldap with IPA-specific logic)
  krb5  — Kerberos-only auth (paired with ldap for identity)
  proxy — Proxy to another NSS/PAM module
  local — Deprecated local user database
  files — /etc/passwd and /etc/group (SSSD as caching layer)
```

### Provider Mixing

Providers can be mixed within a single domain:

```
Common combinations:
  id_provider = ldap + auth_provider = krb5
    → Users from LDAP, authentication via Kerberos (common in university environments)

  id_provider = ad + auth_provider = ad + access_provider = ad
    → Full AD integration (most common enterprise setup)

  id_provider = ldap + auth_provider = ldap + access_provider = simple
    → LDAP for identity/auth, simple allow/deny list for access control
```

## 3. LDAP Provider Internals

### Connection Management

The LDAP provider manages connections with automatic failover and TLS:

```
Connection lifecycle:
  1. Backend starts → resolve LDAP URI (DNS SRV if _srv_)
  2. Establish TCP connection to first available server
  3. STARTTLS or LDAPS (per ldap_uri scheme)
  4. Bind (simple bind with DN/password, or SASL/GSSAPI)
  5. Begin operations
  6. On connection loss:
     a. Mark server as failed
     b. Try next server in ldap_uri list
     c. Retry with exponential backoff
  7. Periodic keepalive (ldap_connection_expire_timeout)

Failover state machine:
  ACTIVE → (connection lost) → FAILED → (backoff timer) → RETRY → ACTIVE
                                                        → FAILED (all servers)
                                                          → OFFLINE MODE
```

### Search Operations

SSSD translates NSS/PAM requests into LDAP searches:

```
getpwnam("jdoe") translates to:
  Base:   ou=People,dc=example,dc=com (ldap_user_search_base)
  Scope:  subtree
  Filter: (&(objectClass=posixAccount)(uid=jdoe))
  Attrs:  uid, uidNumber, gidNumber, homeDirectory, loginShell, cn, ...

initgroups("jdoe") translates to:
  RFC2307:    search ou=Groups for (memberUid=jdoe)
  RFC2307bis: search ou=Groups for (member=uid=jdoe,ou=People,dc=example,dc=com)
              + recursive nested group resolution

Performance optimization:
  - SSSD batches multiple pending requests for the same user
  - Results cached in LDB with entry_cache_timeout TTL
  - Negative cache (nonexistent entries) prevents repeated failed lookups
    entry_cache_nowait_percentage = 50  (refresh at 50% of TTL, serve stale)
```

## 4. AD Provider Internals

### Global Catalog Lookups

The AD provider uses the Global Catalog (GC) for cross-domain operations:

```
Global Catalog (port 3268/3269):
  - Read-only replica of ALL domains in the forest
  - Contains partial attribute set (key identity attributes)
  - Single search covers entire forest (no referral chasing)
  - Used for: cross-domain group membership, universal group resolution

SSSD GC usage:
  1. User login: jdoe@child.ad.example.com
  2. SSSD connects to GC in forest root
  3. Single search finds user + all universal/global group memberships
  4. For domain-local groups: targeted search against user's domain DC
  5. Cache all results locally

  ad_server = dc1.ad.example.com           # domain-specific LDAP (port 389/636)
  # GC server auto-discovered via DNS SRV:
  #   _gc._tcp.ad.example.com → GC servers
```

### SID-to-UID/GID Mapping

When `ldap_id_mapping = true`, SSSD algorithmically converts Active Directory SIDs to POSIX UIDs/GIDs without requiring POSIX attributes in AD:

```
SID structure:
  S-1-5-21-<DomainID1>-<DomainID2>-<DomainID3>-<RID>
  Example: S-1-5-21-123456789-987654321-111222333-1104

ID mapping algorithm:
  1. Domain SID → assigned a "slice" of UID/GID space
     Default range: 200000 - 2000200000 (ldap_idmap_range_min/max)
     Slice size: 200000 (ldap_idmap_range_size)

  2. First domain encountered gets slice [200000, 400000)
     Second domain gets slice [400000, 600000)

  3. Within a slice, UID = slice_start + RID
     User with RID 1104 → UID = 200000 + 1104 = 201104

  Deterministic: same SID always maps to same UID on any host
  (as long as domain discovery order is consistent)

Configuration:
  ldap_id_mapping = true                    # enable algorithmic mapping
  ldap_idmap_range_min = 200000             # start of ID space
  ldap_idmap_range_max = 2000200000         # end of ID space
  ldap_idmap_range_size = 200000            # IDs per domain slice

Alternative: ldap_id_mapping = false
  → Requires uidNumber/gidNumber populated in AD (via Identity Management for UNIX)
  → More control but more admin overhead
```

### AD-Specific Features

```
Token group resolution:
  SSSD uses LDAP extended operation to fetch tokenGroups attribute
  → Returns all transitive group SIDs (nested groups resolved server-side)
  → Much faster than recursive LDAP searches for nested groups

Site discovery:
  1. SSSD performs LDAP search for subnet-to-site mapping
  2. Matches client IP to AD site
  3. Prioritizes DCs in the same site
  4. Falls back to other sites if local DCs unavailable

  ad_site = BranchOffice1                    # override auto-detection

POSIX attribute source priority:
  1. AD attributes (if ldap_id_mapping = false)
  2. Algorithmic mapping (if ldap_id_mapping = true)
  3. Override: sss_override user-add jdoe@ad.example.com --uid=5000
```

## 5. Kerberos Authentication Flow Through SSSD

### Authentication Sequence

```
User types password at login prompt:
  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
  │  login    │───▶│ PAM      │───▶│ sssd_pam │───▶│ sssd_be  │
  │  (getty)  │    │ module   │    │          │    │          │
  └──────────┘    └──────────┘    └──────────┘    └────┬─────┘
                                                       │
  Step 1: pam_authenticate() called                    │
  Step 2: PAM module sends password to sssd_pam        │
  Step 3: sssd_pam forwards to sssd_be                 │
                                                       ▼
                                                  ┌──────────┐
  Step 4: sssd_be performs kinit with password     │   KDC    │
          (AS-REQ with PA-ENC-TIMESTAMP)          │          │
  Step 5: KDC returns TGT (AS-REP)               └──────────┘
  Step 6: sssd_be stores TGT in ccache
          (/tmp/krb5cc_<UID> or KCM)
  Step 7: If cache_credentials=true, store
          password hash in LDB cache
  Step 8: Return PAM_SUCCESS to login

Credential cache types:
  FILE:/tmp/krb5cc_%{uid}         — traditional file-based (default)
  KCM:                            — kernel credential manager (preferred)
  KEYRING:persistent:%{uid}       — kernel keyring (Linux-specific)
  DIR:/tmp/krb5cc_%{uid}_dir/     — directory collection (multiple TGTs)
```

### Ticket Renewal

SSSD can automatically renew Kerberos tickets:

```
Configuration:
  krb5_renewable_lifetime = 7d    # request renewable tickets
  krb5_renew_interval = 3600      # renew every hour (seconds)

Renewal process:
  1. sssd_be sets timer for krb5_renew_interval
  2. On timer: TGS-REQ to KDC with renew flag
  3. KDC returns new TGT with extended lifetime
  4. sssd_be updates ccache
  5. Repeat until renewable_lifetime expires

  If renewal fails (network down):
    → Ticket expires naturally
    → User prompted for password on next interactive login
    → Cached credentials used for local authentication (if configured)
```

## 6. Offline Authentication

### Cached Credential Design

When `cache_credentials = true`, SSSD stores password verification data locally to allow authentication when the remote server is unreachable:

```
Credential storage in LDB cache:
  On successful online authentication:
    1. SSSD hashes the password: SHA-512 + salt (or bcrypt in newer versions)
    2. Stores hash in user's LDB cache entry:
       cachedPassword: {SSHA512}base64encodedHashAndSalt
       lastCachedPasswordChange: 20260405120000Z
       cachedPasswordExpire: 20260412120000Z (offline_credentials_expiration)
    3. Hash NEVER sent to remote server — derived locally from known-good password

  On offline authentication attempt:
    1. User provides password
    2. SSSD computes hash with stored salt
    3. Compares against cachedPassword in LDB
    4. Match → PAM_SUCCESS (offline)
    5. No match → PAM_AUTH_ERR

Security properties:
  - Password stored as salted hash (not reversible)
  - Expiration enforced (offline_credentials_expiration days)
  - No offline password CHANGES allowed (read-only cache)
  - Cached credentials do NOT provide Kerberos tickets
  - Access control still evaluated against cached group membership
```

### Online/Offline State Machine

```
State transitions:
  ONLINE
    │
    ├── LDAP connection fails → OFFLINE
    │   └── Periodic reconnect attempts (offline_timeout, exponential backoff)
    │       └── Success → ONLINE (refresh all stale cache entries)
    │       └── Fail → remain OFFLINE (extend backoff)
    │
    └── DNS SRV lookup fails → try next server → all failed → OFFLINE

  OFFLINE behavior:
    NSS lookups:  served from LDB cache (stale data acceptable)
    PAM auth:     cached credential verification
    PAM acct_mgmt: cached access rules
    Sudo:         cached sudo rules
    Password chg: DENIED (must be online)

  Cache staleness:
    entry_cache_timeout applies to online mode
    In offline mode: all cached entries served regardless of age
    On reconnect: SSSD marks all entries as stale, refreshes on demand
```

## 7. Cache Design (LDB / TDB)

### LDB Database

SSSD uses LDB (LDAP-like Database) as its primary cache. LDB is a library developed for Samba that provides an LDAP-like API over a local TDB (Trivial Database) file:

```
LDB characteristics:
  - LDAP-like interface (search with base, scope, filter)
  - Stored as TDB files on disk
  - Transaction support (atomic writes)
  - Indexed searches (configured per attribute)
  - Schema-less (no predefined schema required)

Cache files:
  /var/lib/sss/db/cache_example.com.ldb    — identity/auth cache per domain
  /var/lib/sss/db/sssd.ldb                 — SSSD configuration cache
  /var/lib/sss/db/timestamps_example.com.ldb — entry freshness timestamps

LDB entry example (user):
  dn: name=jdoe@example.com,cn=users,cn=example.com,cn=sysdb
  objectClass: user
  name: jdoe
  uidNumber: 201104
  gidNumber: 201100
  homeDirectory: /home/jdoe
  loginShell: /bin/bash
  cachedPassword: {SSHA512}...
  dataExpireTimestamp: 1712345678
  originalDN: uid=jdoe,ou=People,dc=example,dc=com

Cache invalidation:
  TTL-based: dataExpireTimestamp checked on every read
  Explicit: sss_cache -u jdoe (sets timestamp to 0)
  Nuclear: rm /var/lib/sss/db/* (full cache wipe)
```

### Timestamp Optimization

SSSD separates timestamps from data to optimize cache validation:

```
Problem: Updating dataExpireTimestamp on every cache hit writes to the main LDB,
         causing unnecessary I/O and TDB lock contention.

Solution: Separate timestamps database
  cache_example.com.ldb      — user/group data (written on identity changes)
  timestamps_example.com.ldb — expiry timestamps (written on every cache hit)

  Benefit: Main cache LDB mostly read-only → better concurrent access
  timestamps LDB handles high-write-rate TTL updates
```

## 8. SSSD vs Winbind vs nslcd Comparison

```
Feature              SSSD                    Winbind               nslcd
─────────────────────────────────────────────────────────────────────────────
Architecture         Multi-process           Single process +      Single process
                     (monitor + responders   child workers         (forking)
                     + backends)

AD support           Native (ad provider,    Native (Samba suite,  Via LDAP only
                     realmd integration)     net ads join)         (no native AD)

Kerberos             Built-in (krb5          Via Samba (winbindd   External (krb5
                     provider, ticket mgmt)  manages tickets)      not integrated)

Offline auth         Yes (cached passwords   Yes (winbindd         No
                     in LDB, configurable    offline logon)
                     expiration)

Cache                LDB + memcache          TDB (tdb files)       No persistent
                     (sophisticated TTL,                           cache (stateless)
                     negative cache)

Sudo rules           Yes (LDAP/IPA sudo)     No                    No

HBAC / GPO           Yes (IPA HBAC,          GPO (via Samba)       No
                     AD GPO)

Smart cards           Yes (pam_cert_auth)     Limited               No

SSH keys             Yes (ssh responder,      No                    No
                     authorized keys from
                     LDAP/IPA)

Autofs               Yes (autofs provider)   No                    No

Multi-domain         Yes (one backend         Yes (trusted          Limited
                     per domain, forest       domains via           (single LDAP
                     trust support)           wbinfo)               base)

D-Bus interface      Yes (InfoPipe)          No                    No

Recommended for      General Linux/AD/IPA    Samba file server     Simple LDAP-only
                     integration, modern      (domain member),      environments,
                     enterprise Linux         legacy environments   minimal overhead

Package size         ~15 MB installed         ~50 MB (full Samba)   ~1 MB
```

### Migration Path: nslcd to SSSD

```
nslcd users migrating to SSSD gain:
  - Offline authentication (biggest win for laptops/unreliable networks)
  - Integrated Kerberos (no separate kinit management)
  - Sudo rule centralization
  - SSH key distribution
  - Better failover and SRV discovery

Migration steps:
  1. Install SSSD packages
  2. Convert /etc/nslcd.conf → /etc/sssd/sssd.conf
     nslcd "uri" → sssd "ldap_uri"
     nslcd "base" → sssd "ldap_search_base"
     nslcd "binddn" → sssd "ldap_default_bind_dn"
  3. Update /etc/nsswitch.conf: nss → sss
  4. Update PAM: pam_ldap → pam_sss
  5. Disable nslcd, enable sssd
  6. Test: getent passwd, id, su
```

### Migration Path: Winbind to SSSD

```
Winbind users migrating to SSSD gain:
  - Sudo integration
  - HBAC / fine-grained access control
  - SSH key distribution
  - Better cache management and diagnostics
  - D-Bus interface for modern desktop integration

Winbind users LOSE:
  - Samba file server RPC authentication (Winbind required for smbd domain member)
  - Some edge-case NTLM authentication scenarios

Recommendation:
  Use SSSD for Linux workstation/server identity
  Keep Winbind ONLY if the host is a Samba file server domain member
  Hybrid possible: SSSD for NSS/PAM, Winbind for Samba file serving
```

## See Also

- ldap
- kerberos
- pam
- saml
- oidc

## References

- SSSD Design Documents: https://sssd.io/design_pages/
- SSSD Source Code: https://github.com/SSSD/sssd
- Red Hat Identity Management Guide (RHEL 9)
- Samba Wiki: Winbind Architecture
- LDB Library: https://ldb.samba.org/
- man sssd.conf(5), man sssd-ldap(5), man sssd-ad(5)
