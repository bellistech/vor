# Kerberos

Network authentication protocol — symmetric crypto + trusted third-party KDC (RFC 4120).

## Setup

### Debian / Ubuntu

```bash
# Client (workstation) packages
sudo apt update
sudo apt install -y krb5-user libpam-krb5 libpam-ccreds auth-client-config

# Server (KDC) packages
sudo apt install -y krb5-kdc krb5-admin-server krb5-config

# AD-integration stack (sssd / realmd)
sudo apt install -y sssd sssd-tools sssd-ad realmd adcli libnss-sss libpam-sss \
    samba-common-bin oddjob oddjob-mkhomedir packagekit
```

### RHEL / CentOS / Rocky / AlmaLinux

```bash
# Client
sudo dnf install -y krb5-workstation krb5-libs

# Server
sudo dnf install -y krb5-server krb5-libs krb5-workstation

# AD integration
sudo dnf install -y sssd sssd-tools sssd-ad realmd adcli oddjob oddjob-mkhomedir \
    samba-common-tools authselect

# FreeIPA / Red Hat IdM (managed Kerberos)
sudo dnf install -y ipa-server ipa-server-dns       # server
sudo dnf install -y ipa-client                      # client
```

### Arch Linux

```bash
# Client + server in one package
sudo pacman -S krb5

# AD integration
sudo pacman -S sssd realmd adcli samba
```

### Alpine

```bash
apk add krb5 krb5-pkinit krb5-server-ldap
```

### macOS

```bash
# Built in via Heimdal — kinit, klist, kdestroy already on PATH
which kinit                # /usr/bin/kinit
# Brew alternative for MIT tooling
brew install krb5
```

### Windows

```text
# Built in — runas /netonly, klist (since Vista), ksetup
# AD provides KDC service automatically on every domain controller
# Microsoft tooling: setspn.exe, ktpass.exe, ksetup.exe
```

### MIT vs Heimdal vs Microsoft AD

```text
MIT Kerberos (krb5):
  Reference implementation, BSD-style license
  Default on RHEL, Debian, most Linux distros
  Tools: kinit, klist, kadmin, kadmin.local, kdb5_util, ktutil, kprop, kpropd
  Library: libkrb5, libgssapi_krb5

Heimdal:
  BSD-licensed clean-room reimplementation, Sweden
  Default on FreeBSD, macOS
  Tools: kinit (different flags!), klist, kadmin, ktutil
  Library: libkrb5 (incompatible ABI with MIT), libgssapi (Heimdal)

Microsoft AD (Active Directory):
  Proprietary KDC built into every domain controller
  Wire-compatible with RFC 4120
  Adds: PAC (Privilege Attribute Certificate), S4U2Self/S4U2Proxy
  Tools: setspn, ktpass, ksetup, klist (Windows), Active Directory Users and Computers

FreeIPA / Red Hat IdM:
  Bundles MIT Kerberos + 389-DS LDAP + Dogtag CA + BIND DNS + ntpd
  Web UI + ipa CLI
  realm = uppercase DNS domain (commonly EXAMPLE.LOCAL)
  Single command install: ipa-server-install
```

## Concepts

```text
KDC (Key Distribution Center) = AS + TGS
  AS  (Authentication Service): issues TGT after pre-auth
  TGS (Ticket Granting Service): issues service tickets given a TGT

Principal: globally unique identifier in a realm
  User principal:    alice@EXAMPLE.COM
  Service principal: HTTP/web.example.com@EXAMPLE.COM
  Host principal:    host/server.example.com@EXAMPLE.COM
  Admin principal:   alice/admin@EXAMPLE.COM   (instance = "admin")

Realm: Kerberos administrative domain (UPPERCASE), e.g., EXAMPLE.COM
  Each realm has exactly one master KDC (read-write database)
  Plus zero or more replica/slave KDCs (read-only via kprop replication)

Keytab: file holding long-term keys for service principals
  Default: /etc/krb5.keytab (mode 0600, owned by root)
  Lets a service decrypt service tickets without typing a password

TGT (Ticket Granting Ticket): proves identity to TGS
  Encrypted with TGS key (krbtgt/REALM@REALM)
  Contains session key for client-TGS communication

Service Ticket: proves identity to a specific service
  Encrypted with service's key (from keytab)
  Contains session key for client-service communication

Ticket Lifetime: time the ticket is valid
  Typical default: 10 hours (36000 seconds)
  Encoded in ticket; KDC will not exceed max_life of principal

Renewability: extend lifetime without re-typing password
  Renewable until renew_lifetime (typical: 7 days)
  Client uses kinit -R to renew (must happen before expiry)

The trust chain:
  Client trusts KDC because they share the long-term password
  Service trusts KDC because they share the keytab key
  Client and service trust each other because both trust KDC
  All trust depends on KDC's secret database
```

## Realm Naming

```text
Convention: UPPERCASE, usually mirrors DNS domain
  example.com (DNS)  ->  EXAMPLE.COM (realm)
  corp.acme.io        ->  CORP.ACME.IO

Why uppercase: distinguishes realm from DNS in mixed contexts and matches RFC 1510 history.
Lowercase realms work but break some clients and SPN canonicalization.

Active Directory:
  AD domain = DNS domain (e.g., corp.example.com)
  Kerberos realm = uppercase AD domain (CORP.EXAMPLE.COM)
  NetBIOS name (legacy) = the SHORTNAME, e.g., CORP

FreeIPA:
  Often <organization>.LOCAL, .IDM, or your real DNS domain in caps
  Convention: realm matches DNS, principal local-part matches uid

Multi-realm trees:
  Parent: EXAMPLE.COM
  Child:  CHILD.EXAMPLE.COM
  Sibling: SIBLING.COM (no domain relationship)
  AD constructs transitive trust automatically inside a forest.
```

## Auth Flow Diagram

```text
                        7-MESSAGE CANONICAL EXCHANGE

  CLIENT (alice@EXAMPLE.COM)                      KDC                       SERVICE (HTTP/web.example.com)
  ---------------------------                     ----                      -------------------------------

  1) AS-REQ  ----------------------------------> [AS]
        principal=alice@EXAMPLE.COM
        request TGT
        (pre-auth: PA-ENC-TIMESTAMP encrypted with user's long-term key)

  2) AS-REP <----------------------------------- [AS]
        TGT (encrypted with krbtgt key, opaque to client)
        Session key TGS_session_key (encrypted with user's long-term key)

        Client decrypts session key with password-derived key.

  3) TGS-REQ ----------------------------------> [TGS]
        TGT
        Authenticator (timestamp encrypted with TGS_session_key)
        Server name = HTTP/web.example.com@EXAMPLE.COM

  4) TGS-REP <---------------------------------- [TGS]
        Service ticket (encrypted with service's key from keytab)
        New session key Service_session_key

  5) AP-REQ ----------------------------------------------------------> [SERVICE]
        Service ticket
        Authenticator (timestamp encrypted with Service_session_key)

        Service decrypts ticket with keytab, extracts session key,
        decrypts authenticator, validates timestamp within clock skew.

  6) AP-REP (optional, for mutual auth) <----------------------------- [SERVICE]
        Timestamp from authenticator + 1, encrypted with Service_session_key

  7) Application data flows over GSS-API context, integrity-protected
     and optionally privacy-protected, using Service_session_key.

  Re-auth: client reuses TGT until expiry (no password re-prompt).
  TGS-REQ to multiple services reuses same TGT.
```

## krb5.conf [libdefaults]

Path: `/etc/krb5.conf` (system) or `~/.krb5.conf` (user override). INI-style.

```ini
[libdefaults]
    # The realm to use when none specified in a principal name
    default_realm = EXAMPLE.COM

    # Use DNS SRV records to find KDCs (recommended for large/dynamic deployments)
    dns_lookup_kdc = true

    # Use DNS TXT records to map hostnames -> realms
    # (Often disabled in practice; prefer explicit [domain_realm])
    dns_lookup_realm = false

    # Default TGT lifetime requested by kinit (seconds, or 1d, 10h, etc.)
    ticket_lifetime = 24h

    # Maximum lifetime over which TGT can be renewed (must be >= ticket_lifetime)
    renew_lifetime = 7d

    # TGT can be forwarded to remote service (e.g., ssh -K with GSSAPIDelegateCredentials)
    forwardable = true

    # TGT can be used by a proxy (deprecated; prefer constrained delegation)
    proxiable = false

    # Allow legacy DES-style enctypes (DANGEROUS — leave disabled in 2024+)
    allow_weak_crypto = false

    # Where credential caches live; %{uid} expands to user's UID
    # FILE (default), DIR, KEYRING, KCM, MEMORY, API
    default_ccache_name = FILE:/tmp/krb5cc_%{uid}
    # default_ccache_name = KEYRING:persistent:%{uid}
    # default_ccache_name = DIR:/var/krb5/users
    # default_ccache_name = KCM:                       # RHEL 8+ default

    # Default keytab file
    default_keytab_name = FILE:/etc/krb5.keytab

    # Enctypes the client will accept in TGS-REP (preference order)
    default_tgs_enctypes = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96 \
                           aes256-cts-hmac-sha384-192 aes128-cts-hmac-sha256-128

    # Enctypes the client will request in AS-REQ
    default_tkt_enctypes = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96 \
                           aes256-cts-hmac-sha384-192 aes128-cts-hmac-sha256-128

    # Enctypes acceptable for any session key
    permitted_enctypes = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96 \
                        aes256-cts-hmac-sha384-192 aes128-cts-hmac-sha256-128

    # Don't enforce that AP-REQ acceptor hostname matches keytab
    # (Useful for HTTP behind a load balancer with multiple SPNs)
    ignore_acceptor_hostname = false

    # Require reverse DNS lookup of service hostname
    # Set to false in containers / cloud where rDNS is unreliable
    rdns = true

    # Switch from UDP to TCP if request would exceed this byte count
    # 0 = always TCP; useful with very large PAC blobs in AD
    udp_preference_limit = 1465

    # Cap on clock skew tolerance in seconds (default: 300)
    clockskew = 300

    # Replay cache type: dfl (default file), none (disabled — risky)
    default_rcache_name = dfl:

    # Force canonicalization of client principal in AS-REQ
    canonicalize = true

    # Disable PAC inclusion (set to false to strip AD authdata)
    include_pac = true

    # Verify SAS / KDC certificate chain on PKINIT
    pkinit_anchors = FILE:/etc/pki/tls/certs/ca-bundle.crt

    # Disable PA-FX-COOKIE behavior on legacy KDCs
    disable_encrypted_timestamp = false

    # Send Kerberos requests over a SPECIFIC interface
    # (rare, useful for multihomed hosts)
    # kdc_default_options = 0x40000000

    # Logging — see [logging] section below
    # forwardable_for_pwchange = false
    # extra_addresses = ...                         # historical, IPv6 era
```

## krb5.conf [realms]

```ini
[realms]
    EXAMPLE.COM = {
        # KDC address(es) — port 88 is default; can specify port
        kdc = kdc1.example.com
        kdc = kdc2.example.com:88
        kdc = tcp/kdc3.example.com:88               # force TCP

        # Master KDC (writable; for kpasswd, kadmin)
        master_kdc = kdc1.example.com

        # Admin server for kadmin (default port 749)
        admin_server = kdc1.example.com:749

        # Default DNS domain to append to short hostnames in this realm
        default_domain = example.com

        # Password change service (kpasswd, default port 464)
        kpasswd_server = kdc1.example.com:464

        # auth_to_local: map foreign principals to local UNIX accounts
        # RULE:[<numcomp>:<format>](regex)s/match/replace/
        # DEFAULT: drop @REALM if matches default_realm; otherwise reject
        auth_to_local = RULE:[1:$1@$0](^.*@CORP\.EXAMPLE\.COM$)s/@.*//
        auth_to_local = RULE:[2:$1](^.*/admin$)s/^.*$/admin/
        auth_to_local = DEFAULT

        # PKINIT
        pkinit_anchors = FILE:/etc/pki/tls/certs/ca-bundle.crt
        pkinit_kdc_hostname = kdc1.example.com
        pkinit_eku_checking = kpKDC
    }

    CORP.EXAMPLE.COM = {
        kdc = ad-dc1.corp.example.com
        kdc = ad-dc2.corp.example.com
        admin_server = ad-dc1.corp.example.com
        default_domain = corp.example.com
    }

    AD.EXAMPLE.COM = {
        # Active Directory specific
        kdc = ad-dc1.ad.example.com
        kdc = ad-dc2.ad.example.com
        admin_server = ad-dc1.ad.example.com
        default_domain = ad.example.com
        # AD doesn't support kadmin protocol; use samba-tool / RSAT instead
    }
```

## krb5.conf [domain_realm]

```ini
[domain_realm]
    # Map a specific FQDN -> realm
    web.example.com = EXAMPLE.COM
    db.example.com  = EXAMPLE.COM

    # Map an entire DNS subtree -> realm
    # Leading dot means "any host under this domain"
    .example.com = EXAMPLE.COM
    example.com  = EXAMPLE.COM

    # Cross-realm mapping
    .corp.example.com = CORP.EXAMPLE.COM
    corp.example.com  = CORP.EXAMPLE.COM

    .ad.example.com = AD.EXAMPLE.COM
    ad.example.com  = AD.EXAMPLE.COM

    # Idiom: always pair "domain" and ".domain" so single-host and subtree both match
```

## krb5.conf [appdefaults]

```ini
[appdefaults]
    # Per-application overrides for libdefaults values

    pam = {
        ticket_lifetime = 1d
        renew_lifetime  = 1d
        forwardable     = true
        proxiable       = false
        retain_after_close = false
        minimum_uid     = 1000
        debug           = false
        external        = sshd
        use_shmem       = true
        krb4_convert    = false
    }

    telnet = {
        ticket_lifetime = 1h
        forwardable     = false
    }

    login = {
        krb4_convert    = false
        krb4_get_tickets = false
    }

    httpd = {
        # Apache mod_auth_gssapi reads this
        forwardable = false
    }

    sshd = {
        forwardable = true
    }
```

## krb5.conf [logging]

```ini
[logging]
    # Daemon-specific log destinations
    default      = FILE:/var/log/krb5libs.log
    kdc          = FILE:/var/log/krb5kdc.log
    admin_server = FILE:/var/log/kadmind.log

    # Multiple destinations allowed
    # default = FILE:/var/log/krb5lib.log
    # default = SYSLOG:NOTICE:DAEMON
    # default = STDERR
    # default = CONSOLE
    # default = DEVICE=/dev/tty01

    # Per-component log level (MIT 1.18+)
    # debug = false
```

## DNS SRV Records

```text
With dns_lookup_kdc=true, the libkrb5 resolver uses these RRs to locate KDCs:

  _kerberos._tcp.EXAMPLE.COM.   IN SRV  0 100 88   kdc1.example.com.
  _kerberos._tcp.EXAMPLE.COM.   IN SRV  10 100 88  kdc2.example.com.
  _kerberos._udp.EXAMPLE.COM.   IN SRV  0 100 88   kdc1.example.com.

  _kerberos-master._tcp.EXAMPLE.COM. IN SRV 0 100 88 kdc1.example.com.
  _kerberos-adm._tcp.EXAMPLE.COM.    IN SRV 0 100 749 kdc1.example.com.

  _kpasswd._udp.EXAMPLE.COM.   IN SRV  0 100 464  kdc1.example.com.
  _kpasswd._tcp.EXAMPLE.COM.   IN SRV  0 100 464  kdc1.example.com.

  ; Optional realm discovery via TXT (used when dns_lookup_realm=true)
  _kerberos.example.com.        IN TXT "EXAMPLE.COM"
```

```bash
# Verify with dig
dig +short SRV _kerberos._tcp.EXAMPLE.COM
dig +short SRV _kerberos-master._tcp.EXAMPLE.COM
dig +short SRV _kpasswd._udp.EXAMPLE.COM

# AD makes these automatically; check with
dig +short SRV _kerberos._tcp.dc._msdcs.corp.example.com
dig +short SRV _ldap._tcp.dc._msdcs.corp.example.com
```

## kinit

```bash
# Most basic — prompt for password and obtain a TGT
kinit alice@EXAMPLE.COM

# Verbose, very useful for debugging
kinit -V alice@EXAMPLE.COM

# Specify principal explicitly (-p)
kinit -p alice/admin@EXAMPLE.COM

# Lifetime (-l) and renewable lifetime (-r)
kinit -l 12h -r 7d alice@EXAMPLE.COM

# Use a keytab instead of typing a password
kinit -k -t /etc/krb5.keytab host/server.example.com@EXAMPLE.COM

# Equivalent shorthand: -k uses default_keytab_name
kinit -k host/server.example.com

# Specify ccache destination
kinit -c FILE:/tmp/krb5cc_alice alice@EXAMPLE.COM
kinit -c KEYRING:persistent:1000 alice@EXAMPLE.COM

# Renew an existing TGT (must still be valid)
kinit -R

# Forwardable / proxiable per-call override
kinit -f alice@EXAMPLE.COM           # request forwardable
kinit -F alice@EXAMPLE.COM           # NOT forwardable
kinit -p alice@EXAMPLE.COM           # request proxiable
kinit -P alice@EXAMPLE.COM           # NOT proxiable
kinit -A alice@EXAMPLE.COM           # request addressless ticket

# Anonymous PKINIT
kinit -n @EXAMPLE.COM

# Validate (renew but don't get new tickets — checks renewability)
kinit -v

# X kdc options (request specific KDC behavior)
kinit -X X509_user_identity=FILE:/path/cert.pem,/path/key.pem alice@EXAMPLE.COM

# Heimdal differences (NOT MIT)
#   -t                  ccache type
#   --keytab=PATH       (long form)
#   --renew             (instead of -R)
#   --validate          (instead of -v)
```

### Pre-auth Flag

```text
Most modern KDCs require pre-authentication. The first kinit attempt sends
AS-REQ with NO pa-data; KDC responds with KRB5KDC_ERR_PREAUTH_REQUIRED and
hints which pa-data types it accepts. Client retries with PA-ENC-TIMESTAMP
or other supported method. Set requires_preauth=true on the principal to
enforce this and disable AS-REP roasting.
```

```bash
# Verify pre-auth is enforced for a principal
kadmin.local -q "getprinc alice"
# Look for "Attributes: REQUIRES_PRE_AUTH" in output
```

## klist

```bash
# Show current default ccache
klist

# Show encryption types of each ticket
klist -e

# Show ticket flags (FRIA: forwardable, renewable, initial, addressless, etc.)
klist -f

# All caches in the collection (KEYRING/DIR/KCM)
klist -A

# List collection caches by name
klist -l

# Quiet — exit code only (0=valid TGT, 1=none)
klist -s && echo "have tgt" || echo "no tgt"

# Inspect a keytab instead of a ccache
klist -k                                  # /etc/krb5.keytab
klist -k /etc/httpd/conf/krb5.keytab
klist -kte /etc/krb5.keytab               # show enctypes too
klist -kt                                 # show key version + timestamps

# Specify a non-default ccache
klist -c FILE:/tmp/krb5cc_alice
klist -c KEYRING:persistent:1000

# Sample output:
#   Ticket cache: FILE:/tmp/krb5cc_1000
#   Default principal: alice@EXAMPLE.COM
#
#   Valid starting       Expires              Service principal
#   2024-04-25T10:00:00  2024-04-26T10:00:00  krbtgt/EXAMPLE.COM@EXAMPLE.COM
#       renew until 2024-05-02T10:00:00, Flags: FPRIA
#   2024-04-25T10:01:00  2024-04-26T10:00:00  HTTP/web.example.com@EXAMPLE.COM
#       Flags: FAT
```

### Ticket Flag Letters

```text
F   Forwardable
f   forwarded (already)
P   Proxiable
p   proxy (already)
D   may be postDated
d   postdated (already)
R   Renewable
I   Initial (TGT direct from AS-REQ; not from TGS)
i   invalid
H   Hardware-authenticated
A   pre-Authenticated
T   Transit-policy-checked
O   OK as delegate (S4U2Proxy permitted target)
a   anonymous
```

## kdestroy

```bash
# Destroy default ccache
kdestroy

# Destroy ALL caches in the collection
kdestroy -A

# Destroy a specific ccache
kdestroy -c FILE:/tmp/krb5cc_1000
kdestroy -c KEYRING:persistent:1000

# Quiet (no error if nothing to destroy)
kdestroy -q

# Idiom: log out script
kdestroy -A 2>/dev/null || true
```

## kpasswd

```bash
# Change own password (uses kpasswd_server / SRV record)
kpasswd

# Change another principal's password (must have admin TGT)
kpasswd alice@EXAMPLE.COM

# Behavior: connects to kpasswd_server (port 464),
# authenticates with current password, sets new password.
# Password policy is enforced server-side: minlength, mindiff, history, etc.

# Sample interaction:
#   Password for alice@EXAMPLE.COM: ********
#   Enter new password: ********
#   Enter it again: ********
#   Password changed.
```

### Common kpasswd Errors

```text
"Password is too short"                  -> increase length, see policy
"Password does not contain enough character classes"   -> mix upper/lower/digit/special
"Password is too recent"                 -> wait min_lifetime (often 1 day)
"Password has been used too recently"    -> hit history (e.g., last 10 forbidden)
"Soft error from server"                 -> typically a policy violation
"Generic error (see e-text)"             -> server-side; check kadmind log
```

## kvno

```bash
# Query Key Version Number for a service principal
kvno HTTP/web.example.com@EXAMPLE.COM
# Output: HTTP/web.example.com@EXAMPLE.COM: kvno = 4

# Multiple at once
kvno HTTP/web.example.com host/db.example.com

# Specify enctype (-e)
kvno -e aes256-cts-hmac-sha1-96 HTTP/web.example.com

# Use a non-default ccache
kvno -c FILE:/tmp/krb5cc_alice HTTP/web.example.com

# Specify a different output ccache (-S)
kvno -S host HTTP/web.example.com

# Idiom: validate that a keytab decrypts a real service ticket
kinit -k -t /etc/krb5.keytab HTTP/web.example.com
kvno HTTP/web.example.com
# If kvno succeeds, keytab works.
```

## ksu

```bash
# Switch user with Kerberos auth instead of /etc/shadow
ksu alice
# Reads ~/.k5users on target user; if alice has TGT and is listed, allowed.

# Inherit cache instead of reusing
ksu -n alice

# Pass through environment
ksu -p alice

# Specify ccache
ksu -c FILE:/tmp/krb5cc_alice alice

# Drop privileges back
exit
```

## kadmin

```bash
# Remote kadmin (network protocol, requires admin TGT)
kadmin -p alice/admin@EXAMPLE.COM
kadmin: addprinc bob

# Local kadmin (direct database access; runs as root on the KDC)
sudo kadmin.local
kadmin.local: addprinc bob

# One-shot from CLI (-q "command")
kadmin.local -q "listprincs"
kadmin.local -q "addprinc -randkey HTTP/web.example.com"
kadmin.local -q "ktadd -k /tmp/web.keytab HTTP/web.example.com"
```

### Core kadmin Commands

```text
addprinc <principal>          create new principal (prompts for password)
delete_principal <principal>  delete principal           (alias: delprinc)
modify_principal <principal>  modify principal attrs     (alias: modprinc)
rename_principal <old> <new>  rename principal           (alias: renprinc)
listprincs [pattern]          list all principals (or matching glob)
get_principal <principal>     show principal details     (alias: getprinc)
change_password <principal>   set new password for principal  (alias: cpw)
ktadd -k <keytab> <princ>...  extract keys to keytab
ktremove -k <keytab> <princ>  remove entry from keytab
get_policy <name>             show password policy       (alias: getpol)
add_policy <name>             create password policy     (alias: addpol)
modify_policy <name>          modify password policy     (alias: modpol)
delete_policy <name>          delete password policy     (alias: delpol)
list_policies                 list all policies          (alias: listpols)
get_strings <principal>       show string attributes
set_string <princ> <key> <v>  set string attribute
del_string <princ> <key>      delete string attribute
purgekeys [-keepkvno N] <princ>  remove old keys from KDB
lock <principal>              prevent further authentication
unlock <principal>            re-enable authentication
?                             list commands
quit | exit | q               exit kadmin
```

## kadmin Commands Deep

```bash
# Add a service principal with a random key (no password)
kadmin.local -q "addprinc -randkey HTTP/web.example.com"

# Add a user with a policy attached
kadmin.local -q "addprinc -policy default alice"

# Add with explicit max ticket lifetime + renew lifetime
kadmin.local -q "addprinc -maxlife 24h -maxrenewlife 7d alice"

# Add with explicit enctype list (overrides default supported_enctypes)
kadmin.local -q "addprinc -randkey -e aes256-cts-hmac-sha1-96:normal,aes128-cts-hmac-sha1-96:normal HTTP/web.example.com"

# Force pre-auth
kadmin.local -q "modprinc +requires_preauth alice"

# Allow forwardable tickets
kadmin.local -q "modprinc +allow_forwardable alice"

# Force password change at next login
kadmin.local -q "modprinc +needchange alice"

# Disable password expiry warning
kadmin.local -q "modprinc -needchange alice"

# Common +/- attributes (+ enable, - disable)
#   allow_postdated, allow_forwardable, allow_renewable, allow_proxiable,
#   allow_dup_skey, allow_tix, requires_preauth, requires_hwauth,
#   needchange, allow_svr, password_changing_service, ok_as_delegate

# Set service-key kvno explicitly
kadmin.local -q "setpw -randkey HTTP/web.example.com"

# Extract to keytab (creates entries for ALL of principal's keys, all enctypes)
kadmin.local -q "ktadd -k /etc/krb5.keytab HTTP/web.example.com"

# Extract specific enctypes only
kadmin.local -q "ktadd -k /tmp/web.keytab -e 'aes256-cts-hmac-sha1-96:normal aes128-cts-hmac-sha1-96:normal' HTTP/web.example.com"

# Don't increment kvno on extract (-norandkey)
kadmin.local -q "ktadd -k /tmp/web.keytab -norandkey HTTP/web.example.com"

# Remove from keytab (all enctypes for principal)
kadmin.local -q "ktremove -k /etc/krb5.keytab HTTP/web.example.com all"

# Remove specific kvno
kadmin.local -q "ktremove -k /etc/krb5.keytab HTTP/web.example.com 3"

# Add password policy
kadmin.local -q "add_policy -minlength 12 -minclasses 3 -history 10 -maxlife 90d -minlife 1d strict"

# Apply policy to principal
kadmin.local -q "modprinc -policy strict alice"

# Show policy
kadmin.local -q "get_policy strict"
```

## ktutil

```bash
# Interactive shell for keytab manipulation
ktutil
ktutil:  rkt /etc/krb5.keytab                  # read keytab
ktutil:  list                                  # list slots
ktutil:  list -k -e                            # list with kvno + enctype
ktutil:  add_entry -password -p alice@EXAMPLE.COM -k 3 -e aes256-cts-hmac-sha1-96
            Password for alice@EXAMPLE.COM: ********
ktutil:  add_entry -key -p HTTP/web.example.com@EXAMPLE.COM -k 3 -e aes256-cts-hmac-sha1-96
            Key: <hex-key>
ktutil:  delete_entry 1                        # delete slot 1
ktutil:  wkt /etc/krb5.keytab                  # write keytab
ktutil:  rkt second.keytab                     # also read second
ktutil:  wkt merged.keytab                     # write merged
ktutil:  change_passwd                         # change password for all entries
ktutil:  clear                                 # clear keylist
ktutil:  quit
```

```text
ktutil sub-commands (MIT):
  read_kt | rkt <file>              read keytab into list
  write_kt | wkt <file>             write list to keytab
  add_entry [-password|-key] [-p] [-k kvno] [-e enctype] [-f] [-s salt] [-K hex]
  delete_entry <slot>               delete slot N
  list | l [-t] [-k] [-e]           list entries
  list_requests | lr | ?            list commands
  clear_list | clear                clear in-memory list
  quit | exit | q                   exit
  change_passwd                     change password for all entries

Heimdal ktutil sub-commands differ slightly:
  list, add, get, remove, change, copy, rename, srvconvert, srv2keytab, purge
```

## The SPN Format

```text
Service Principal Name format:

   <service>/<host.fqdn>@<REALM>

Components:
  service   short name of the service (lowercase by convention)
  host.fqdn fully-qualified DNS name of the host (lowercase)
  REALM     uppercase realm name

Examples:
  HTTP/web.example.com@EXAMPLE.COM        Apache, nginx, IIS
  host/server.example.com@EXAMPLE.COM     ssh, login
  ldap/ad.example.com@EXAMPLE.COM         OpenLDAP, AD LDAP
  cifs/file.example.com@EXAMPLE.COM       SMB/CIFS
  nfs/nfs.example.com@EXAMPLE.COM         Kerberized NFSv4
  postgres/db.example.com@EXAMPLE.COM     PostgreSQL gss auth
  imap/mail.example.com@EXAMPLE.COM       Cyrus, Dovecot
  smtp/mail.example.com@EXAMPLE.COM       Postfix, sendmail
  ftp/files.example.com@EXAMPLE.COM       Kerberized FTP
  termsrv/desktop.example.com@EXAMPLE.COM AD RDP

Critical rule: the FQDN in the SPN MUST exactly match what the client uses to
connect to the service AND the canonical DNS-resolved name (subject to rdns
setting). A short hostname or wrong domain breaks auth silently.

Active Directory creates HOST/<computer>@DOMAIN.COM automatically when a
machine joins. AD's "HOST/" SPN doubles as alias for many service classes
(cifs, http, etc.) — but only if explicitly declared.

To verify which SPN a client requests, use:
  KRB5_TRACE=/dev/stderr klist <service>
  tcpdump -i any -nn port 88 -w kerb.pcap   # then inspect with Wireshark
```

## Pre-authentication

```text
PA-ENC-TIMESTAMP (RFC 4120, most common)
  Client encrypts a current timestamp with their long-term key,
  sends as pa-data in AS-REQ. KDC decrypts, validates timestamp
  within clock skew, then issues TGT.
  Defeats AS-REP roasting (offline attack on TGT enc).

FAST (Flexible Authentication Secure Tunneling, RFC 6113)
  Wraps AS-REQ inside an armored channel using an existing TGT
  (typically a host TGT). Hides pa-data from passive observers.
  Required by some hardened KDCs and AD with FAST-only mode.
  Configure with [libdefaults] enable_fast = true and provide an
  armor ccache: kinit -T <armor-ccache> alice

PKINIT (RFC 4556)
  Certificate-based pre-auth. Client signs AS-REQ with X.509
  private key; KDC verifies signature against pkinit_anchors.
  Use case: smart cards, CAC/PIV cards, machine certificates.
  AD: Windows 10/11 Hello-for-Business, RHEL/IPA: ipa-getcert.

OTP (RFC 6560)
  One-time-password pre-auth. Combines password + token (e.g., HOTP).
  Less common; supported by FreeIPA, RSA-style integrations.

SAM (Single-use Auth Mechanism)
  Legacy challenge-response; obsolete.

S4U2Self / S4U2Proxy (Service-for-User extensions)
  Constrained delegation in AD; not pre-auth strictly but related.
  Service obtains a ticket for itself on behalf of a user (S4U2Self),
  then exchanges it for a service ticket to a downstream service
  (S4U2Proxy) — without the user's password.
```

## Encryption Types

```text
Recommended (RFC 4120 + RFC 8009):
  aes256-cts-hmac-sha1-96         (alias: aes256-cts)            FIPS-OK
  aes128-cts-hmac-sha1-96         (alias: aes128-cts)            FIPS-OK
  aes256-cts-hmac-sha384-192      (RFC 8009, MIT 1.15+)          FIPS-OK
  aes128-cts-hmac-sha256-128      (RFC 8009, MIT 1.15+)          FIPS-OK
  camellia256-cts-cmac            (rare, JP government)
  camellia128-cts-cmac

Legacy / weak (DISABLE in production):
  des3-cbc-sha1                   3DES — slow, deprecated
  arcfour-hmac                    RC4 — broken; AD legacy
  arcfour-hmac-md5
  arcfour-hmac-exp                export-grade RC4 — never enable
  des-cbc-md5                     56-bit DES — broken
  des-cbc-crc                     56-bit DES, CRC integrity — broken
  des-cbc-md4                     56-bit DES — broken

  These all imply allow_weak_crypto=true; DON'T.

Configure on KDC (kdc.conf):
  supported_enctypes = aes256-cts-hmac-sha1-96:normal \
                       aes128-cts-hmac-sha1-96:normal \
                       aes256-cts-hmac-sha384-192:normal \
                       aes128-cts-hmac-sha256-128:normal

Configure on client (krb5.conf):
  default_tkt_enctypes = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96
  default_tgs_enctypes = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96
  permitted_enctypes   = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96

The :normal salt suffix means "default salt"; alternatives include
:v4, :norealm, :onlyrealm, :afs3, :special — virtually never used outside
historical Kerberos 4 transitions.

To inspect a ticket's enctype:
  klist -e
    Etype (skey, tkt): aes256-cts-hmac-sha1-96, aes256-cts-hmac-sha1-96
    AD-types: pa-pac
```

## Cross-realm Trust

```text
One-way trust (REALM_A trusts REALM_B but not the reverse):
  Users in REALM_B can access services in REALM_A.
  Users in REALM_A canNOT access services in REALM_B.

  KDC of REALM_A must have principal:    krbtgt/REALM_B@REALM_A
  KDC of REALM_B must have principal:    krbtgt/REALM_B@REALM_A
  Both copies share the SAME password (= shared trust secret).

Two-way (mutual) trust:
  Add the inverse pair too:
  KDC of REALM_A:    krbtgt/REALM_A@REALM_B
  KDC of REALM_B:    krbtgt/REALM_A@REALM_B
  Same password on both sides.
```

```bash
# Setup steps (MIT to MIT):

# On REALM_A KDC:
kadmin.local -q "addprinc -e aes256-cts-hmac-sha1-96:normal -pw <SECRET> krbtgt/REALM_B@REALM_A"

# On REALM_B KDC, same password:
kadmin.local -q "addprinc -e aes256-cts-hmac-sha1-96:normal -pw <SECRET> krbtgt/REALM_B@REALM_A"

# Then verify with kvno cross-realm
# On a client with TGT in REALM_B:
kvno HTTP/web.realm-a.example@REALM_A

# The kvno of both copies MUST match. After password rollover,
# both sides must re-key together.
```

## Transitive Trust

```text
If REALM_A <-> REALM_B and REALM_B <-> REALM_C exist (and [capaths] permits),
a user in REALM_C can access a service in REALM_A by traversing REALM_B.

The trust path is encoded by [capaths] in krb5.conf:
```

```ini
[capaths]
    REALM_A = {
        REALM_C = REALM_B          # path from A to C goes via B
    }
    REALM_C = {
        REALM_A = REALM_B          # path from C to A goes via B
    }
    REALM_B = {
        REALM_A = .                # direct trust
        REALM_C = .                # direct trust
    }
```

```text
A "." means direct trust (no intermediate). Multiple intermediates are
listed space-separated:
```

```ini
[capaths]
    REALM_A = {
        REALM_D = REALM_B REALM_C  # A -> B -> C -> D
    }
```

```text
Each TGS-REQ along the chain is sent to a different KDC. The client's
ticket cache will contain referral tickets like:

  krbtgt/REALM_B@REALM_A
  krbtgt/REALM_C@REALM_B
  HTTP/svc.realm-c.example@REALM_C

AD: forest-wide trust is automatic; cross-forest requires explicit forest
trust object (one-way or two-way) configured via Active Directory Domains
and Trusts MMC.
```

## PKINIT

```ini
# krb5.conf — client-side PKINIT
[libdefaults]
    default_realm = EXAMPLE.COM
    pkinit_anchors = FILE:/etc/pki/tls/certs/ca-bundle.crt
    pkinit_kdc_hostname = kdc.example.com
    pkinit_eku_checking = kpKDC
    pkinit_require_eku = true
    pkinit_pool = FILE:/etc/pki/intermediate-ca.pem
    pkinit_revoke = FILE:/etc/pki/crl.pem
    pkinit_identities = PKCS11:/usr/lib64/libcoolkeypk11.so
    # Or
    # pkinit_identities = FILE:/etc/pki/user-cert.pem,/etc/pki/user-key.pem
    # Or for smartcard
    # pkinit_identities = PKCS11:/usr/lib64/opensc-pkcs11.so

[realms]
    EXAMPLE.COM = {
        pkinit_anchors = FILE:/etc/pki/example-ca.pem
        pkinit_kdc_hostname = kdc.example.com
    }
```

```bash
# Use a smartcard for kinit (PKCS#11 via OpenSC)
kinit -X X509_user_identity=PKCS11:/usr/lib64/opensc-pkcs11.so alice@EXAMPLE.COM

# Use a file-based cert
kinit -X X509_user_identity=FILE:/etc/pki/alice.crt,/etc/pki/alice.key alice@EXAMPLE.COM

# Anonymous PKINIT (FAST armor without a user identity)
kinit -n @EXAMPLE.COM
```

```text
PKINIT use cases:
  Smartcards (PIV / CAC) — eliminate password
  Hardware tokens (YubiKey with PKCS#11)
  Pre-staged machine certs for unattended kinit
  AD: certificate-based logon when password is unavailable

KDC-side requirements:
  KDC's certificate must have id-pkinit-KPKdc EKU (1.3.6.1.5.2.3.5)
  Client cert must have id-pkinit-KPClientAuth EKU (1.3.6.1.5.2.3.4)
  pkinit_anchors must include CA chain that issued KDC cert
```

## GSSAPI

```text
Generic Security Services Application Programming Interface (RFC 2743)
A standard C API layered ABOVE Kerberos (and other mechanisms like SPNEGO).

libgssapi_krb5 (MIT)            on Linux
libgssapi (Heimdal)             on FreeBSD/macOS
SSPI (Security Support Provider Interface)  on Windows — same wire format

Key API surface:
  gss_acquire_cred()       grab credentials (e.g., from default ccache)
  gss_init_sec_context()   client side: produce AP-REQ token
  gss_accept_sec_context() server side: consume AP-REQ, produce AP-REP
  gss_get_mic() / gss_verify_mic()    integrity-protect a message
  gss_wrap() / gss_unwrap()           confidentiality + integrity
  gss_delete_sec_context() tear down
  gss_release_cred()
  gss_release_buffer()
  gss_display_status()    decode minor status (e.g., krb5 error code)
```

```bash
# Test commands
klist                                 # use it to verify TGT exists
ssh -K user@host                      # delegate via GSSAPI
curl --negotiate -u : https://web.example.com/   # HTTP SPNEGO
```

```text
// Skeleton: client-side GSSAPI authentication
#include <gssapi/gssapi.h>
gss_buffer_desc out_token = GSS_C_EMPTY_BUFFER;
gss_ctx_id_t   ctx = GSS_C_NO_CONTEXT;
OM_uint32 maj, min;
maj = gss_init_sec_context(&min, GSS_C_NO_CREDENTIAL, &ctx,
                           target_name, GSS_C_NO_OID,
                           GSS_C_MUTUAL_FLAG | GSS_C_DELEG_FLAG,
                           0, GSS_C_NO_CHANNEL_BINDINGS, GSS_C_NO_BUFFER,
                           NULL, &out_token, NULL, NULL);
// Send out_token.value bytes to server, etc.
```

## SPNEGO

```text
Simple and Protected GSSAPI Negotiation Mechanism (RFC 4178)
Wraps any GSSAPI mechanism for negotiation. In practice on the web,
SPNEGO selects Kerberos and falls back to NTLM in AD environments.

HTTP Negotiate (RFC 4559) — the dance:

  Client                         Server
  ------                         ------
  GET /private             ----->
                           <----- 401 Unauthorized
                                  WWW-Authenticate: Negotiate

  GET /private             ----->
  Authorization: Negotiate <base64-spnego-blob>
                           <----- 200 OK
                                  WWW-Authenticate: Negotiate <base64-AP-REP>

The blob is a SPNEGO-wrapped GSSAPI Kerberos AP-REQ. The server delegates
to GSSAPI which validates the AP-REQ against its keytab, then returns AP-REP.
```

```ini
# Apache (mod_auth_gssapi, current; mod_auth_kerb is deprecated)
# /etc/httpd/conf.d/auth_kerberos.conf
<Location /private>
    AuthType GSSAPI
    AuthName "Kerberos Login"
    GssapiCredStore keytab:/etc/httpd/conf/krb5.keytab
    GssapiAllowedMech krb5
    GssapiUseSessions On
    Session On
    SessionCookieName gssapi_session path=/private;httponly;secure
    GssapiDelegCcacheDir /run/httpd/clientcaches
    Require valid-user
</Location>
```

```ini
# Older mod_auth_kerb (deprecated, but still seen)
<Location /private>
    AuthType Kerberos
    AuthName "Kerberos Login"
    KrbAuthRealms EXAMPLE.COM
    KrbServiceName HTTP/web.example.com
    Krb5Keytab /etc/httpd/conf/krb5.keytab
    KrbMethodNegotiate On
    KrbMethodK5Passwd Off
    Require valid-user
</Location>
```

```text
nginx: no native module — use lua-resty-gssapi, or nginx-spnego-http-auth
(Stoke MIT plugin), or sidecar proxy (Apache reverse-proxying nginx).
```

```bash
# Curl client side
# Get a TGT first
kinit alice@EXAMPLE.COM

# Then negotiate
curl --negotiate -u : -b /tmp/cookies -c /tmp/cookies https://web.example.com/private

# --negotiate enables SPNEGO; -u : provides empty user (cred from ccache)
```

## Active Directory Integration

```ini
# /etc/sssd/sssd.conf
[sssd]
config_file_version = 2
domains = corp.example.com
services = nss, pam, ssh, sudo

[domain/corp.example.com]
id_provider = ad
auth_provider = ad
chpass_provider = ad
access_provider = ad

ad_domain = corp.example.com
ad_server = dc1.corp.example.com, dc2.corp.example.com
ad_backup_server = dc3.corp.example.com
ad_site = London
ad_hostname = workstation.corp.example.com

krb5_realm = CORP.EXAMPLE.COM
krb5_server = dc1.corp.example.com, dc2.corp.example.com
krb5_canonicalize = false
krb5_keytab = /etc/krb5.keytab

ldap_id_use_start_tls = false
ldap_id_mapping = true
ldap_schema = ad
ldap_idmap_range_size = 200000
ldap_idmap_range_min = 200000
ldap_idmap_range_max = 2000200000
ldap_idmap_default_domain_sid = S-1-5-21-...

cache_credentials = true
fallback_homedir = /home/%u@%d
default_shell = /bin/bash
override_homedir = /home/%u@%d
use_fully_qualified_names = true
case_sensitive = false

dyndns_update = true
dyndns_refresh_interval = 43200
dyndns_update_ptr = true

debug_level = 6
```

```bash
# realmd (high-level wrapper)
sudo realm discover corp.example.com
sudo realm join --user=domain-admin corp.example.com
sudo realm list
sudo realm leave corp.example.com

# adcli (lower-level, more flexible)
sudo adcli join --domain=corp.example.com --domain-controller=dc1.corp.example.com \
                --domain-ou='OU=Linux,DC=corp,DC=example,DC=com' \
                --login-user=domain-admin
sudo adcli show-computer --domain=corp.example.com
sudo adcli update --domain=corp.example.com   # rotate machine password
sudo adcli delete-computer --domain=corp.example.com

# What realm join produces
ls -l /etc/krb5.keytab          # machine keytab — mode 0600
sudo klist -kt /etc/krb5.keytab # shows host/<fqdn>@REALM and HOST/<short>@REALM
ls -l /etc/sssd/sssd.conf       # mode 0600
sudo systemctl restart sssd

# Verify
id alice@corp.example.com
getent passwd alice@corp.example.com
sudo systemctl status sssd
sssctl domain-list
sssctl domain-status corp.example.com
sssctl logs-fetch /tmp/sssd.tar.gz
```

## AD Service Principal Patterns

```text
On join, AD auto-creates these for the computer object:
  HOST/<computer-shortname>@DOMAIN.COM
  HOST/<computer.fqdn>@DOMAIN.COM

The "HOST/" alias resolves implicitly to:
  cifs/, http/, ldap/, host/, etc. (per AD's SPN aliasing)

For specific services you typically register:
  HTTP/web.corp.example.com@CORP.EXAMPLE.COM       IIS / Apache / nginx
  MSSQLSvc/sql.corp.example.com:1433@CORP.EXAMPLE.COM  SQL Server
  ldap/ad-dc1.corp.example.com@CORP.EXAMPLE.COM    LDAP (auto on DC)
  cifs/file.corp.example.com@CORP.EXAMPLE.COM      SMB/CIFS
```

```text
Windows tooling:

  setspn -L <computer>            list SPNs on computer
  setspn -A HTTP/web.corp.example.com WEB1   add SPN
  setspn -D HTTP/web.corp.example.com WEB1   delete SPN
  setspn -Q HTTP/web.corp.example.com        query who has it
  setspn -X                       find duplicate SPNs (corruption check)
  setspn -F -A HTTP/...           add even if duplicate (force)
  setspn -S HTTP/web.corp.example.com WEB1  smart add (checks dup first)

  ktpass -princ HTTP/web.corp.example.com@CORP.EXAMPLE.COM ^
         -mapuser web-svc@corp.example.com ^
         -pass <strong-password> ^
         -ptype KRB5_NT_PRINCIPAL ^
         -crypto AES256-SHA1 ^
         -out C:\temp\web.keytab

  ksetup /addkdc CORP.EXAMPLE.COM dc1.corp.example.com
  ksetup /domain CORP.EXAMPLE.COM
  ksetup /setdomain CORP.EXAMPLE.COM
  klist                            list cached tickets (Windows)
  klist purge                      destroy cached tickets
```

```bash
# Linux equivalent — generate keytab from AD without ktpass

# On a Linux client joined to AD
sudo net ads keytab create -k        # samba-tool / net
sudo net ads keytab add HTTP -U domain-admin
sudo net ads keytab list
klist -kt /etc/krb5.keytab
```

## Kerberized Services

### SSH

```ini
# /etc/ssh/sshd_config (server side)
GSSAPIAuthentication yes
GSSAPIKeyExchange yes
GSSAPICleanupCredentials yes
GSSAPIStrictAcceptorCheck yes
KerberosAuthentication no
KerberosOrLocalPasswd no
KerberosTicketCleanup yes
```

```ini
# /etc/ssh/ssh_config (client side)
Host *.example.com
    GSSAPIAuthentication yes
    GSSAPIKeyExchange yes
    GSSAPIDelegateCredentials yes
    PreferredAuthentications gssapi-with-mic,publickey,password
    HostbasedAuthentication no
```

```bash
# Use it
kinit alice@EXAMPLE.COM
ssh -K alice@server.example.com           # -K = enable GSSAPI delegation
ssh -o GSSAPIAuthentication=yes alice@server.example.com

# Verify on server
who                                       # alice ... gssapi
sudo journalctl _SYSTEMD_UNIT=ssh.service | grep -i gss

# Server-side keytab requirement
sudo klist -kt /etc/krb5.keytab           # must contain host/<fqdn>@REALM
```

### HTTP (Apache mod_auth_gssapi)

```ini
# /etc/httpd/conf.d/kerberos.conf
LoadModule auth_gssapi_module modules/mod_auth_gssapi.so

<Location />
    AuthType GSSAPI
    AuthName "Kerberos Login"
    GssapiCredStore keytab:/etc/httpd/conf/krb5.keytab
    GssapiUseS4U2Self On
    GssapiDelegCcacheDir /run/httpd/clientcaches
    GssapiSSLonly On
    GssapiUseSessions On
    Session On
    SessionCookieName gssapi_session path=/;httponly;secure
    Require valid-user
</Location>
```

```text
# nginx via auth_request to a sidecar
location / {
    auth_request /negotiate;
    proxy_pass http://upstream;
}
location = /negotiate {
    internal;
    proxy_pass http://localhost:8081;   # sidecar that does SPNEGO
}
```

### NFS

```bash
# /etc/exports (server)
# /data  client.example.com(rw,sec=krb5,no_root_squash)
# /data  *(ro,sec=krb5p)            # privacy

# sec= variants:
#   krb5   authentication only (no integrity, no privacy)
#   krb5i  authentication + integrity (HMAC-SHA1 per packet)
#   krb5p  authentication + integrity + privacy (encryption)

# Mount (client)
sudo mount -t nfs4 -o sec=krb5p server.example.com:/data /mnt/data

# /etc/fstab
# server.example.com:/data /mnt/data nfs4 sec=krb5p,vers=4.2 0 0

# nfs-utils requires keytab:
sudo klist -kt /etc/krb5.keytab           # must have nfs/<fqdn>@REALM

# Tracing
sudo rpc.gssd -vvv -f                          # foreground client daemon
sudo rpc.svcgssd -vvv -f                       # foreground server daemon
```

### CIFS / SMB

```bash
# Linux client mounting Windows share with Kerberos
sudo mount -t cifs //file.corp.example.com/share /mnt/share \
    -o sec=krb5,multiuser,vers=3.1.1,cruid=$(id -u)

# /etc/fstab
# //file.corp.example.com/share /mnt/share cifs sec=krb5,multiuser,vers=3.1.1,_netdev 0 0

# sec= variants for CIFS:
#   krb5         Kerberos auth
#   krb5i        Kerberos + signing
#   ntlmssp      legacy fallback
```

### LDAP (SASL/GSSAPI)

```bash
# OpenLDAP server: requires ldap/<fqdn>@REALM in keytab
ldapsearch -Y GSSAPI -H ldap://ad-dc1.corp.example.com -b "dc=corp,dc=example,dc=com"

# Verbose
ldapsearch -Y GSSAPI -v -H ldap://ad-dc1.corp.example.com -b "dc=corp,dc=example,dc=com" \
    "(objectClass=person)"

# Idiom: use ldapwhoami to verify GSSAPI
ldapwhoami -Y GSSAPI -H ldap://ad-dc1.corp.example.com
# Output: dn:cn=alice,cn=Users,dc=corp,dc=example,dc=com
```

### PostgreSQL

```ini
# postgresql.conf
krb_server_keyfile = '/etc/postgresql/krb5.keytab'
krb_caseins_users = on
```

```text
# pg_hba.conf
# TYPE   DATABASE  USER  ADDRESS       METHOD
hostgssenc all     all   0.0.0.0/0     gss include_realm=0 krb_realm=EXAMPLE.COM
```

```bash
# Client (with TGT)
psql "host=db.example.com user=alice dbname=app gssencmode=require"

# Without GSS encryption but Kerberos auth
psql "host=db.example.com user=alice dbname=app krbsrvname=postgres"
```

## The 5-Minute Clock-Skew Window

```text
Kerberos rejects authenticators with timestamps outside +/- clockskew (default
300 seconds = 5 minutes) of the KDC's clock. This is the SINGLE most common
"Kerberos doesn't work" cause.

Symptoms:
  "kinit: Clock skew too great while getting initial credentials"
  ssh GSSAPI fails with "Server not yet available"
  Existing tickets work, new ones don't.
```

```bash
# Fix (always-on time sync)

# chrony (default on RHEL 8+, Ubuntu 22+)
sudo systemctl enable --now chronyd
chronyc tracking
chronyc sources
sudo chronyc makestep                  # force step now
```

```ini
# /etc/chrony.conf
pool 2.pool.ntp.org iburst
makestep 1.0 3
rtcsync
```

```bash
# systemd-timesyncd
sudo systemctl enable --now systemd-timesyncd
timedatectl status
sudo timedatectl set-ntp true

# Active Directory (every domain controller is an NTP server)
# Point chrony at the closest DC:
#   server dc1.corp.example.com iburst prefer
#   server dc2.corp.example.com iburst

# Validate skew vs KDC explicitly:
ntpdate -q kdc.example.com             # rdate-style query
chronyc tracking | grep 'Last offset'

# Bigger window? (NOT recommended; fix the time instead)
# /etc/krb5.conf:
# [libdefaults]
#     clockskew = 600
```

## ccache Types

```text
FILE (default)
  /tmp/krb5cc_$UID, single-cache.
  Pros: simple, portable across krb5 implementations.
  Cons: world-readable on $TMPDIR if mode wrong (defaults to 0600).
  Setting: default_ccache_name = FILE:/tmp/krb5cc_%{uid}

DIR
  Multi-cache directory (e.g., /run/user/1000/krb5cc/).
  Each principal gets its own cache; one is "primary".
  Useful when a single user holds tickets for multiple principals.
  Setting: default_ccache_name = DIR:/run/user/%{uid}/krb5cc

KEYRING
  Stored in the kernel keyring; persists for session/user.
  Pros: not on disk, gone after reboot, isolated from other UIDs.
  Cons: limited size, fewer tools support inspecting it directly.
  Variants:
    KEYRING:thread:NAME       per-thread
    KEYRING:process:NAME      per-process
    KEYRING:session:NAME      per-login-session
    KEYRING:user:NAME         per-uid (cleared on logout)
    KEYRING:persistent:UID    persists until reboot (RHEL 7 default)
  Setting: default_ccache_name = KEYRING:persistent:%{uid}

KCM (Kerberos Credentials Manager)
  Daemon-managed cache (sssd-kcm or libkrb5's own kcm).
  Pros: containers can share, daemon brokers access.
  Cons: requires daemon (sssd-kcm.service).
  Default on RHEL 8+, Fedora 33+.
  Setting: default_ccache_name = KCM:
  Verify: systemctl status sssd-kcm.service

MEMORY
  In-process only; gone when process exits.
  Useful for ephemeral scripts.
  Setting: default_ccache_name = MEMORY:

API (macOS only)
  Mac OS X / Heimdal credential manager.
  Setting: default_ccache_name = API:
```

```bash
# Switch ccache for a single command
KRB5CCNAME=KEYRING:persistent:1000 kinit alice@EXAMPLE.COM
KRB5CCNAME=FILE:/tmp/alice.cc      klist

# Switch in shell
export KRB5CCNAME=KEYRING:persistent:$(id -u)

# List all caches in a collection (DIR / KCM / KEYRING)
klist -A
klist -l
```

## Common Errors

Verbatim error messages, cause, and fix:

```text
kinit: Clock skew too great while getting initial credentials
  Cause: client clock differs from KDC by > clockskew (default 300s).
  Fix:   sync NTP. chronyd / timesyncd / chronyc makestep.
```

```text
kinit: Cannot find KDC for requested realm
  Cause: dns_lookup_kdc=false AND no [realms] kdc=... entry,
         or DNS SRV records missing,
         or realm name typo.
  Fix:   Add kdc = kdc.example.com to [realms] EXAMPLE.COM = {...}
         OR enable dns_lookup_kdc=true and create _kerberos._tcp SRVs.
```

```text
kinit: Server not found in Kerberos database
  Cause: principal does not exist in KDC database (e.g., service principal
         not created, or user typo).
  Fix:   kadmin.local -q "listprincs" to verify; create principal if missing.
```

```text
kinit: Decrypt integrity check failed while getting initial credentials
  Cause: WRONG PASSWORD, OR enctype mismatch (KDC issued ticket in enctype
         the client doesn't list in default_tkt_enctypes), OR principal's
         long-term key was changed but ccache stale.
  Fix:   re-type password; confirm enctype overlap between
         default_tkt_enctypes (client) and supported_enctypes (KDC).
```

```text
kinit: Wrong principal in request
  Cause: case mismatch in principal name; KDC sees "Alice@EXAMPLE.COM" but
         AS-REQ is "alice@EXAMPLE.COM" (Kerberos is case-sensitive).
  Fix:   use exact principal as stored in KDB (kadmin.local listprincs).
```

```text
kinit: KDC has no support for encryption type
  Cause: client requested an enctype the KDC doesn't have keys for
         (e.g., AES256 client vs RC4-only KDC, or vice versa).
  Fix:   align supported_enctypes (KDC) with default_tkt_enctypes (client);
         re-key principals with the desired enctype:
         kadmin.local -q "cpw -randkey -e aes256-cts-hmac-sha1-96:normal alice"
```

```text
kinit: Preauthentication failed
  Cause: wrong password; OR locked-out account; OR pre-auth method mismatch.
  Fix:   try again; check kadmin.local "getprinc alice" for "Failed Auths"
         or expiration.
```

```text
kinit: Generic preauthentication failure
  Cause: catch-all from KDC; pre-auth attempt failed for an unspecified
         reason. Often pre-auth disabled on principal but client requires it.
  Fix:   kadmin.local -q "modprinc +requires_preauth alice"; retry.
```

```text
kinit: Client not found in Kerberos database while getting initial credentials
  Cause: user principal does not exist in KDB.
  Fix:   kadmin.local -q "addprinc alice"; OR confirm realm is correct.
```

```text
kinit: Realm not local to KDC
  Cause: trying to kinit into a remote realm but local KDC's principal db
         doesn't have a TGT principal for that realm.
  Fix:   Cross-realm trust must exist; OR connect to the foreign realm's
         KDC directly (kdc = ...) in [realms].
```

```text
Cannot determine realm for host (no default realm set)
  Cause: krb5.conf has no default_realm, AND no [domain_realm] mapping
         covers the FQDN being used.
  Fix:   set [libdefaults] default_realm = EXAMPLE.COM
         AND/OR add web.example.com = EXAMPLE.COM in [domain_realm].
```

```text
krb5_get_init_creds: KDC reply did not match expectations
  Cause: typically PKINIT misconfiguration, or principal canonicalization
         mismatch (alice vs ALICE, short vs FQDN), or replay protection.
  Fix:   set canonicalize=true; verify pkinit_anchors; check KDC log
         (/var/log/krb5kdc.log) for the corresponding AS-REQ.
```

```text
GSS major status: Unspecified GSS failure.  Minor code may provide more information
GSS minor status: No Kerberos credentials available (default cache: FILE:/tmp/krb5cc_1000)
  Cause: no TGT in default ccache; ccache expired; ccache moved.
  Fix:   kinit alice@EXAMPLE.COM; verify with klist; ensure KRB5CCNAME
         points to a valid cache.
```

```text
kvno: Key table entry not found while getting credentials for HTTP/web.example.com@EXAMPLE.COM
  Cause: keytab does not contain the requested SPN with a matching kvno,
         OR kvno mismatch between keytab (e.g., 3) and KDB (e.g., 5)
         after a rekey.
  Fix:   kadmin.local -q "ktadd -k /etc/krb5.keytab HTTP/web.example.com"
         to refresh keytab; verify with klist -kt.
```

```text
ssh: GSSAPI Error: Unspecified GSS failure.  Minor code may provide more information
  Cause: usually no TGT, OR forwardable=false TGT, OR server keytab missing
         host/<fqdn> SPN, OR rDNS broken so GSS-acceptor doesn't recognize
         server name.
  Fix:   kinit -f alice@EXAMPLE.COM; klist -kt /etc/krb5.keytab on server;
         verify reverse DNS resolves server's IP back to expected FQDN.
```

## Common Gotchas

```text
1. DNS SRV not configured + dns_lookup_kdc=false
   BROKEN: kinit: Cannot find KDC for requested realm
   FIXED:  Add kdc = kdc.example.com under [realms] OR set
           dns_lookup_kdc = true AND create _kerberos._tcp.EXAMPLE.COM SRV.

2. Time skew > 5 minutes
   BROKEN: kinit: Clock skew too great
   FIXED:  systemctl enable --now chronyd; chronyc makestep
           Persistent: configure NTP server in /etc/chrony.conf.

3. default_realm typo
   BROKEN: Cannot determine realm for host (no default realm set)
           OR sends AS-REQ to wrong realm and gets "Realm not local".
   FIXED:  Verify [libdefaults] default_realm = EXAMPLE.COM (uppercase!)

4. FQDN vs short hostname mismatch in SPN
   BROKEN: kvno HTTP/web (works); kvno HTTP/web.example.com fails
           OR client connects via "web" alias and AP-REQ uses
           HTTP/web@REALM but keytab has HTTP/web.example.com@REALM.
   FIXED:  Always use FQDN in SPNs; configure DNS so service is reached
           by canonical FQDN; or add multiple SPN variants to keytab.

5. Keytab readable by everyone
   BROKEN: anyone who reads /etc/krb5.keytab can impersonate the service.
   FIXED:  chmod 600 /etc/krb5.keytab; chown root:root /etc/krb5.keytab
           For non-root services (Apache): chmod 640, chgrp apache.

6. Reverse-DNS resolution required by default; rdns=true breaks LB
   BROKEN: rdns=true; client connects to LB IP, GSS-acceptor reverse-
           resolves IP -> "real-host-1.internal" -> not in keytab.
   FIXED:  rdns = false in [libdefaults] (modern best practice);
           OR add rdns target's SPN to keytab; OR pin client to the
           canonical FQDN that matches keytab.

7. kpropd not running on slave KDC
   BROKEN: master_kdc has new principal; slaves return "Server not found".
   FIXED:  systemctl enable --now kpropd; verify /var/kerberos/krb5kdc/kadm5.acl
           and /etc/krb5.conf [realms] master_kdc set; force replication
           with kprop -f /var/kerberos/krb5kdc/slave_datatrans slave.kdc

8. allow_weak_crypto=true accidentally left
   BROKEN: arcfour-hmac and DES enabled — vulnerable to AS-REP roasting,
           skeleton key, golden-ticket-with-weak-key.
   FIXED:  allow_weak_crypto = false; remove old enctypes from keytab:
           kadmin.local -q "cpw -randkey -e aes256-cts-hmac-sha1-96:normal,aes128-cts-hmac-sha1-96:normal alice"
           Re-extract clean keytab via ktadd.

9. kvno mismatch between keytab and KDC after password change
   BROKEN: kvno: Key table entry not found while getting credentials.
   FIXED:  Re-extract keytab AFTER each password rotation:
           kadmin.local -q "ktadd -k /etc/krb5.keytab <princ>"
           OR use -norandkey on ktadd to keep existing kvno.

10. canonicalize=true required for some clients (especially cross-realm)
    BROKEN: AD client sends "host/short@REALM"; expected canonical
            "host/short.fqdn@REALM"; KDC returns "Server not found".
    FIXED:  [libdefaults] canonicalize = true on the client.

11. capaths missing for transitive trust
    BROKEN: REALM_A user trying to reach REALM_C service via REALM_B
            gets "Realm not local"; KDCs don't know how to refer.
    FIXED:  Add explicit [capaths] entries on every KDC and client:
            REALM_A = { REALM_C = REALM_B }
            REALM_C = { REALM_A = REALM_B }

12. default_ccache_name pointing to read-only path
    BROKEN: kinit: cred cache I/O failure; or "permission denied" on
            /tmp/krb5cc_$UID when /tmp is mounted noexec,nosuid,ro.
    FIXED:  default_ccache_name = KEYRING:persistent:%{uid}
            OR KCM: (uses sssd-kcm); both avoid filesystem permissions.

13. PAM stack missing pam_krb5 line
    BROKEN: Kerberos password works for kinit but not for ssh/login;
            sssd offline credential cache empty.
    FIXED:  authselect select sssd with-mkhomedir
            OR /etc/pam.d/system-auth includes pam_krb5.so or pam_sss.so:
              auth        required      pam_env.so
              auth        sufficient    pam_unix.so try_first_pass nullok
              auth        sufficient    pam_krb5.so use_first_pass forwardable
              auth        required      pam_deny.so

14. /etc/hosts shadowing DNS (FQDN ambiguity)
    BROKEN: /etc/hosts has "192.168.1.10 server" without server.example.com;
            gethostbyaddr returns "server" not FQDN; rdns=true breaks SPN match.
    FIXED:  Order entries: "192.168.1.10 server.example.com server"
            OR set rdns=false; OR rely on DNS only.

15. SELinux blocking keytab read by Apache
    BROKEN: Apache logs "GSSAPI: Couldn't open file ... Permission denied"
            but keytab perms are 640 apache.
    FIXED:  semanage fcontext -a -t httpd_keytab_t /etc/httpd/conf/krb5.keytab
            restorecon -v /etc/httpd/conf/krb5.keytab
            setsebool -P httpd_use_kerberos 1

16. Long-running daemons holding stale TGTs
    BROKEN: TGT expired; daemon sees "GSS credentials expired" forever.
    FIXED:  k5start / kstart / kinit -R via cron; OR use a keytab and
            kinit -k inside the daemon's startup; OR sssd-keep-alive.
```

## Diagnostic Tools

```bash
# MIT trace logging — most powerful debug tool
KRB5_TRACE=/dev/stderr kinit alice@EXAMPLE.COM
KRB5_TRACE=/tmp/krb.log kinit -V alice@EXAMPLE.COM

# Heimdal equivalent
KRB5_TRACE=/dev/stderr kinit alice@EXAMPLE.COM   # same env var

# Verbose flag
kinit -V alice@EXAMPLE.COM

# Inspect ticket enctypes (mismatch debugging)
klist -e

# Inspect keytab (verify SPNs and enctypes)
klist -kte /etc/krb5.keytab

# KDC logs (server-side)
sudo tail -f /var/log/krb5kdc.log
sudo tail -f /var/log/kadmind.log
sudo tail -f /var/log/krb5libs.log

# Network capture (Kerberos uses port 88 TCP/UDP, 749 TCP for kadmin, 464 for kpasswd)
sudo tcpdump -i any -nn -w kerb.pcap port 88 or port 464 or port 749
# Open in Wireshark — built-in Kerberos dissector decodes AS-REQ/AS-REP/etc.

# DNS sanity
dig +short SRV _kerberos._tcp.EXAMPLE.COM
dig +short -x 192.168.1.10                   # rDNS check
host kdc.example.com
host -t TXT _kerberos.example.com

# sssd debugging
sudo sssctl logs-fetch /tmp/sssd.tar.gz
sudo SSSD_KRB5_INCLUDE_PAC_ATTRIBUTE=1 systemctl restart sssd
sudo journalctl -u sssd --since "10 min ago"

# realmd debugging
sudo realm --verbose join corp.example.com

# adcli debugging
sudo adcli --verbose join corp.example.com

# Test a service principal end-to-end
kinit alice@EXAMPLE.COM
kvno HTTP/web.example.com@EXAMPLE.COM
klist
```

## AD-Specific Errors

```text
"wstr_to_str: Invalid argument"
  Cause: encoding issue — UTF-8 input where Windows expects UTF-16,
         or libsamba stack mismatch.
  Fix:   verify locale: locale; export LC_ALL=C.UTF-8;
         realm leave then realm join with --user explicit.

"Invalid signature was found in the message"
  Cause: mutual auth failed — client got AP-REP whose signature didn't
         verify, often because of clock skew, or kvno mismatch, or
         tampered token (MITM).
  Fix:   chronyc makestep; klist -e (verify enctype); rotate keytab.

"ERR: KRB5KDC_ERR_PREAUTH_REQUIRED"
  Cause: this is just a HINT — KDC tells client "include pa-data".
         The client retries automatically. NOT actually an error in
         steady state; only a problem if client never retries.
  Fix:   ignore in trace logs; verify subsequent AS-REQ has pa-data.

"KRB5KDC_ERR_C_PRINCIPAL_UNKNOWN"
  Cause: AD doesn't have the user / SPN you're requesting.
  Fix:   verify with: ldapsearch -Y GSSAPI ... "(samAccountName=alice)"
         OR setspn -L <computer> for service principals.

"KRB5KDC_ERR_S_PRINCIPAL_UNKNOWN"
  Cause: target service SPN doesn't exist in AD.
  Fix:   setspn -A HTTP/web.corp.example.com <computer-or-svc-account>

"KRB5KDC_ERR_BADOPTION"
  Cause: requested ticket option (forwardable/proxiable) not allowed
         on principal, e.g., user account "Account is sensitive and
         cannot be delegated".
  Fix:   in AD Users and Computers -> User properties -> Account
         tab -> uncheck "Account is sensitive and cannot be delegated".

"NT_STATUS_NO_LOGON_SERVERS"
  Cause: cifs/SMB couldn't reach a DC.
  Fix:   verify DNS SRV _ldap._tcp.dc._msdcs.<domain>; ping the DC;
         realm list to confirm join.

"NT_STATUS_LOGON_FAILURE"
  Cause: bad password or account lockout.
  Fix:   net rpc password ... ; OR Reset-ADAccountPassword on Windows.
```

## Hardening

```ini
# /etc/krb5.conf hardening template
[libdefaults]
    default_realm = EXAMPLE.COM
    dns_lookup_kdc = true
    dns_lookup_realm = false
    rdns = false
    canonicalize = true

    # Disable weak crypto entirely
    allow_weak_crypto = false

    # Strictly modern enctypes only (NO RC4, NO 3DES, NO DES)
    default_tgs_enctypes = aes256-cts-hmac-sha1-96 aes256-cts-hmac-sha384-192
    default_tkt_enctypes = aes256-cts-hmac-sha1-96 aes256-cts-hmac-sha384-192
    permitted_enctypes   = aes256-cts-hmac-sha1-96 aes256-cts-hmac-sha384-192

    # Tighter ticket lifetimes
    ticket_lifetime = 8h
    renew_lifetime = 1d

    # Force pre-auth, disable forwardable by default
    forwardable = true                         # required by ssh -K
    proxiable = false

    # FAST armoring for paranoid environments
    enable_fast = true

    # Use kernel keyring instead of filesystem ccache
    default_ccache_name = KEYRING:persistent:%{uid}

[realms]
    EXAMPLE.COM = {
        kdc = kdc1.example.com
        kdc = kdc2.example.com
        master_kdc = kdc1.example.com
        admin_server = kdc1.example.com
        # Restrict allowed keysalts globally
        allowed_keysalts = aes256-cts:normal aes128-cts:normal
    }
```

```ini
# /var/kerberos/krb5kdc/kdc.conf
[kdcdefaults]
    kdc_ports = 88
    kdc_tcp_ports = 88
    spake_preauth_kdc_challenge = edwards25519

[realms]
    EXAMPLE.COM = {
        master_key_type = aes256-cts-hmac-sha384-192
        supported_enctypes = aes256-cts-hmac-sha384-192:normal \
                             aes256-cts-hmac-sha1-96:normal \
                             aes128-cts-hmac-sha256-128:normal \
                             aes128-cts-hmac-sha1-96:normal
        max_life = 8h 0m 0s
        max_renewable_life = 1d 0h 0m 0s
        default_principal_flags = +preauth, -forwardable

        # FAST configuration
        encrypted_challenge_indicator = nip-fast

        # ACL files
        database_name = /var/kerberos/krb5kdc/principal
        acl_file = /var/kerberos/krb5kdc/kadm5.acl
        admin_keytab = /var/kerberos/krb5kdc/kadm5.keytab
        key_stash_file = /var/kerberos/krb5kdc/.k5.EXAMPLE.COM
        dict_file = /usr/share/dict/words
    }
```

```bash
# Strong password policy
kadmin.local <<'EOF'
add_policy -minlength 14 -minclasses 4 -history 24 -maxlife 90d -minlife 1d -maxfailure 5 -failurecountinterval 30m -lockoutduration 30m strict
modprinc -policy strict alice
EOF

# Disable arcfour-hmac on existing principals (re-key)
kadmin.local -q "cpw -randkey -e aes256-cts-hmac-sha1-96:normal,aes128-cts-hmac-sha1-96:normal HTTP/web.example.com"
kadmin.local -q "ktadd -k /etc/krb5.keytab HTTP/web.example.com"

# Audit
kadmin.local -q "listprincs" | while read p; do
    kadmin.local -q "getprinc $p" | grep -A1 "Number of keys"
done
```

## realmd / sssd

```bash
# Discovery before joining
sudo realm discover corp.example.com
# Output:
#   corp.example.com
#     type: kerberos
#     realm-name: CORP.EXAMPLE.COM
#     domain-name: corp.example.com
#     configured: no
#     server-software: active-directory
#     client-software: sssd
#     required-package: oddjob
#     required-package: oddjob-mkhomedir
#     required-package: sssd
#     required-package: adcli
#     required-package: samba-common-tools

# Join
sudo realm join --user=domain-admin corp.example.com
sudo realm join --user=domain-admin --computer-ou='OU=Linux,DC=corp,DC=example,DC=com' corp.example.com

# Membership check
sudo realm list

# Permit all logins
sudo realm permit --all

# Permit specific users / groups
sudo realm permit alice@corp.example.com
sudo realm permit -g linux-admins@corp.example.com

# Deny
sudo realm deny --all
sudo realm deny --realm corp.example.com

# Leave domain
sudo realm leave
sudo realm leave corp.example.com --remove

# sssd offline credential cache
# When network is down, sssd can authenticate with previously
# cached credentials (last successful login).
# Configure:
#   cache_credentials = true
#   krb5_store_password_if_offline = true
# In /etc/sssd/sssd.conf [domain/...]

# sssd debug levels (0=off, 9=maximum)
# Edit /etc/sssd/sssd.conf, add to [domain/<realm>] section:
#   debug_level = 9
# Then:
sudo systemctl restart sssd
sudo journalctl -u sssd -f

# Force cache flush
sudo sss_cache -E
sudo sss_cache -u alice@corp.example.com
sudo sss_cache -g linux-admins@corp.example.com

# Inspect cache directly
sudo ls -la /var/lib/sss/db/
sudo strings /var/lib/sss/db/cache_corp.example.com.ldb | head
```

## FreeIPA / Red Hat IdM

```bash
# Server install (interactive)
sudo ipa-server-install --realm=EXAMPLE.LOCAL --domain=example.local \
    --hostname=ipa.example.local --ds-password='SecretDS1' \
    --admin-password='AdminPwd1' --setup-dns --no-forwarders \
    --auto-reverse --unattended

# Client install (interactive)
sudo ipa-client-install --domain=example.local --server=ipa.example.local \
    --principal=admin --password='AdminPwd1' --mkhomedir --enable-dns-updates \
    --unattended

# Get a Kerberos ticket and authenticate ipa CLI
kinit admin
ipa user-add alice --first=Alice --last=Doe --email=alice@example.local --password
ipa user-find alice
ipa user-show alice --all

# Service principals
ipa service-add HTTP/web.example.local
ipa service-show HTTP/web.example.local

# Fetch a keytab for a service
sudo ipa-getkeytab -p HTTP/web.example.local -k /etc/httpd/conf/krb5.keytab \
    -s ipa.example.local

# Add to existing keytab without overwriting
sudo ipa-getkeytab -p HTTP/web.example.local -k /etc/krb5.keytab \
    -s ipa.example.local -e aes256-cts-hmac-sha1-96

# DNS records (built into FreeIPA)
ipa dnszone-add example.local
ipa dnsrecord-add example.local web --a-rec=192.168.1.10
ipa dnszone-find

# Host management
ipa host-add web.example.local --password='OnetimePwd1'
ipa host-show web.example.local

# HBAC (Host-Based Access Control)
ipa hbacrule-add allow_web_admins
ipa hbacrule-add-user --users=alice allow_web_admins
ipa hbacrule-add-host --hosts=web.example.local allow_web_admins
ipa hbacrule-add-service --hbacsvcs=sshd allow_web_admins
ipa hbacrule-enable allow_web_admins

# Sudo rules
ipa sudorule-add web_admin_full
ipa sudorule-add-user --users=alice web_admin_full
ipa sudorule-add-host --hosts=web.example.local web_admin_full
ipa sudorule-add-allow-command --sudocmds=ALL web_admin_full
ipa sudorule-mod web_admin_full --runasuser=root --runasgroup=root

# Password policy
ipa pwpolicy-mod global_policy --minlength=12 --history=24 --minclasses=3
ipa pwpolicy-add web_admins --minlength=14 --history=48 --minclasses=4 --maxlife=60 --minlife=1
ipa group-add-member web_admins --users=alice
```

## Active Directory + Linux Cookbook

```bash
# Recipe: a Linux box joined to AD, using Kerberos for SSH and HTTP

# 1. Pre-flight: NTP synced, DNS pointing at AD DCs, FQDN matches AD record.
sudo systemctl enable --now chronyd
echo "server dc1.corp.example.com iburst prefer" | sudo tee -a /etc/chrony.conf
echo "server dc2.corp.example.com iburst" | sudo tee -a /etc/chrony.conf
sudo systemctl restart chronyd
chronyc tracking
chronyc sources

sudo hostnamectl set-hostname workstation.corp.example.com
echo "nameserver 192.168.1.10" | sudo tee /etc/resolv.conf  # AD DC
echo "nameserver 192.168.1.11" | sudo tee -a /etc/resolv.conf
echo "search corp.example.com" | sudo tee -a /etc/resolv.conf

# 2. Discover and join
sudo realm discover corp.example.com
sudo realm join --user=domain-admin@CORP.EXAMPLE.COM corp.example.com

# 3. Verify
sudo realm list
klist -kt /etc/krb5.keytab            # should contain host/<fqdn>@CORP.EXAMPLE.COM
                                       # and HOST/<fqdn>@CORP.EXAMPLE.COM
                                       # and HOST/<short>@CORP.EXAMPLE.COM
sudo systemctl status sssd

# 4. Test login as AD user
id alice@corp.example.com
getent passwd alice@corp.example.com
ssh alice@corp.example.com@workstation.corp.example.com
# Inside that ssh session:
klist                                  # should show TGT for alice@CORP.EXAMPLE.COM

# 5. Add a service: web (Apache) auth
sudo dnf install -y httpd mod_auth_gssapi
sudo realm permit -g web-admins@corp.example.com

# Add HTTP SPN to AD computer object (run on Windows)
#   setspn -S HTTP/web.corp.example.com WEB1$
# Then refresh keytab on Linux:
sudo net ads keytab add HTTP -U domain-admin
sudo klist -kt /etc/krb5.keytab        # should now also show HTTP/<fqdn>

# 6. Apache config (see "HTTP" section above)
sudo systemctl restart httpd

# 7. Test from another joined client
kinit alice@CORP.EXAMPLE.COM
curl --negotiate -u : https://web.corp.example.com/private
# Expect 200 OK with X-Remote-User: alice@CORP.EXAMPLE.COM

# 8. Add NFSv4 with sec=krb5p
sudo dnf install -y nfs-utils
echo "/data *(rw,sec=krb5p)" | sudo tee -a /etc/exports
sudo exportfs -ra
sudo systemctl enable --now nfs-server rpc-gssd

# Add nfs/<fqdn>@REALM SPN
#   setspn -S nfs/web.corp.example.com WEB1$
sudo net ads keytab add nfs -U domain-admin

# 9. Mount on client
sudo mount -t nfs4 -o sec=krb5p web.corp.example.com:/data /mnt/data
mount | grep nfs4
ls /mnt/data
```

## Idioms

```text
"always run NTP"
  No NTP -> 5-min skew -> all auth eventually fails.

"klist before everything"
  When a Kerberized command fails, the FIRST command to run is klist.
  No TGT -> kinit. Expired -> kinit -R or kinit. Wrong realm -> klist
  shows it instantly.

"kvno your service to verify keytab"
  After ktadd, immediately:    kvno <service>/<fqdn>@REALM
  Success means keytab is correct AND KDB has the principal AND enctypes
  overlap. Failure tells you which is wrong.

"use KEYRING ccache on shared systems"
  default_ccache_name = KEYRING:persistent:%{uid}
  Avoids /tmp permission issues, leaks, and stale caches across reboots.

"FAST armor for paranoid"
  enable_fast = true; armor with a separate (machine) TGT.
  Hides pa-data from passive observers (defeats AS-REP roasting).

"AD realm = uppercase domain"
  corp.example.com (DNS) -> CORP.EXAMPLE.COM (Kerberos realm)
  Mismatched case is the most common typo.

"TGT renewability lets you avoid re-typing password"
  ticket_lifetime = 8h; renew_lifetime = 7d
  Run kinit -R hourly (cron) to keep TGT fresh; renew until renew_lifetime.

"FQDN, not short hostname, in SPNs"
  Always:  HTTP/web.example.com@EXAMPLE.COM
  Never:   HTTP/web@EXAMPLE.COM (unless explicitly aliased)

"keep keytabs at 0600"
  chmod 600 /etc/krb5.keytab; chown root:root /etc/krb5.keytab
  For service users: chmod 640, chgrp <service>.

"rdns=false in modern stacks"
  Cloud, containers, and load balancers break reverse DNS assumptions.

"canonicalize=true with cross-realm"
  AD and FreeIPA both expect KDC referrals; client canonicalize=true
  is required to follow them.

"rotate machine password"
  AD machine password ages out (default 30 days). adcli update
  rotates it; missing this rotation -> NT_STATUS_LOGON_FAILURE.

"audit weak enctypes annually"
  klist -kte /etc/krb5.keytab | grep -E 'arcfour|des-' && echo BAD

"replicate KDC databases"
  master KDC: kprop -f slave_dump slave.kdc
  slave: kpropd reads incoming dumps; restart krb5kdc.

"FAST + PKINIT + KEYRING = paranoid baseline"
  kdc.conf: spake_preauth_kdc_challenge = edwards25519
  krb5.conf: enable_fast = true, default_ccache_name = KEYRING:...
  client cert via pkinit_identities.

"don't use kinit -p in scripts; use -k -t"
  Script-friendly:  kinit -k -t /etc/krb5.keytab service/host@REALM
  No password prompt; pulls key from keytab.

"prefer KCM over FILE in containers"
  default_ccache_name = KCM:
  Daemon mediates access; survives bind-mount weirdness.

"GSSAPI, not Kerberos, is the API surface"
  Your code calls libgssapi_krb5; the library calls libkrb5.
  Apache, OpenSSH, Postfix, etc. integrate via GSSAPI.

"strict_acceptor_check on for SSH"
  GSSAPIStrictAcceptorCheck yes — prevents accepting tickets meant
  for a different SPN that happens to share keys.
```

## See Also

- ssh — GSSAPI delegation, GSSAPIAuthentication, host keytab integration
- openssl — X.509 issuance for PKINIT certificates and KDC anchors
- gpg — alternative PKI for code signing; complements Kerberos identity
- age — file encryption pairing with Kerberos-issued identity tokens
- vault — modern secrets manager; can issue dynamic Kerberos credentials
- sops — secret-file encryption that can use Kerberos-derived keys
- radius — alternate AAA protocol; often co-deployed via FreeRADIUS+krb5
- tacacs — Cisco AAA; often integrated via TACACS+ to Kerberos backend
- ldap — directory backing FreeIPA, AD, and SASL/GSSAPI bind
- polyglot — code patterns that span Kerberos-aware libraries

## References

- RFC 4120 — The Kerberos Network Authentication Service (V5)
- RFC 4121 — The Kerberos Version 5 GSS-API Mechanism: Version 2
- RFC 4178 — The Simple and Protected GSS-API Negotiation Mechanism (SPNEGO)
- RFC 4556 — Public Key Cryptography for Initial Authentication in Kerberos (PKINIT)
- RFC 4559 — SPNEGO-based Kerberos and NTLM HTTP Authentication
- RFC 6113 — A Generalized Framework for Kerberos Pre-Authentication (FAST)
- RFC 6560 — One-Time Password (OTP) Pre-Authentication
- RFC 6649 — Deprecate DES, RC4-HMAC-EXP, and Other Weak Cryptographic Algorithms in Kerberos
- RFC 8009 — AES Encryption with HMAC-SHA2 for Kerberos 5
- RFC 8636 — Public Key Cryptography for Initial Authentication in Kerberos (PKINIT) Algorithm Agility
- RFC 2743 — Generic Security Service Application Program Interface, Version 2
- RFC 2744 — Generic Security Service API Version 2: C-bindings
- web.mit.edu/kerberos — MIT Kerberos consortium, source, docs
- web.mit.edu/kerberos/krb5-latest/doc/ — current MIT Kerberos manual
- freeipa.org — FreeIPA / Red Hat IdM project home
- samba.org/docs — Samba integration with AD Kerberos
- learn.microsoft.com/windows-server/security/kerberos — Microsoft Kerberos docs
- man.archlinux.org/man/krb5.conf.5 — krb5.conf(5) manual
- man.archlinux.org/man/kdc.conf.5 — kdc.conf(5) manual
- man.archlinux.org/man/kadmin.1 — kadmin(1) manual
- man.archlinux.org/man/kinit.1 — kinit(1) manual
- man.archlinux.org/man/klist.1 — klist(1) manual
- man.archlinux.org/man/ktutil.1 — ktutil(1) manual
- sssd.io/docs — SSSD documentation
- access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_authentication_and_authorization_in_rhel — RHEL 9 authentication guide
