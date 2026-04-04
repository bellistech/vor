# Kerberos (Network Authentication Protocol)

Authenticate users and services across untrusted networks using Kerberos V5's ticket-based protocol with KDC architecture, keytab management, cross-realm trusts, and SPNEGO negotiation via MIT Kerberos and Active Directory implementations.

## KDC Architecture

### Components

```
┌─────────────────────────────────────────────┐
│           Key Distribution Center           │
│                                             │
│  ┌──────────────┐  ┌────────────────────┐   │
│  │     AS       │  │       TGS          │   │
│  │ (Authenticate│  │ (Ticket-Granting   │   │
│  │   Service)   │  │    Service)        │   │
│  └──────────────┘  └────────────────────┘   │
│         │                    │               │
│         └────────┬───────────┘               │
│                  │                           │
│         ┌────────┴────────┐                  │
│         │   Principal DB  │                  │
│         │   (MIT krb5db / │                  │
│         │    AD NTDS.dit) │                  │
│         └─────────────────┘                  │
└─────────────────────────────────────────────┘
```

### Authentication Flow

```bash
# Step 1: AS-REQ — client requests TGT
#   Client → KDC: {principal, timestamp encrypted with client key}

# Step 2: AS-REP — KDC returns TGT
#   KDC → Client: {TGT encrypted with krbtgt key,
#                   session key encrypted with client key}

# Step 3: TGS-REQ — client requests service ticket
#   Client → KDC: {TGT, authenticator, target service SPN}

# Step 4: TGS-REP — KDC returns service ticket
#   KDC → Client: {service ticket encrypted with service key,
#                   service session key encrypted with TGT session key}

# Step 5: AP-REQ — client authenticates to service
#   Client → Service: {service ticket, authenticator}
```

## krb5.conf Configuration

### Client Configuration

```ini
# /etc/krb5.conf
[libdefaults]
    default_realm = EXAMPLE.COM
    dns_lookup_realm = false
    dns_lookup_kdc = true
    ticket_lifetime = 10h
    renew_lifetime = 7d
    forwardable = true
    rdns = false
    default_ccache_name = KEYRING:persistent:%{uid}

[realms]
    EXAMPLE.COM = {
        kdc = kdc1.example.com
        kdc = kdc2.example.com
        admin_server = kdc1.example.com
        default_domain = example.com
    }
    PARTNER.ORG = {
        kdc = kdc.partner.org
        admin_server = kdc.partner.org
    }

[domain_realm]
    .example.com = EXAMPLE.COM
    example.com = EXAMPLE.COM
    .partner.org = PARTNER.ORG

[capaths]
    PARTNER.ORG = {
        EXAMPLE.COM = .
    }
```

### KDC Configuration

```ini
# /var/kerberos/krb5kdc/kdc.conf (MIT) or /etc/krb5kdc/kdc.conf
[kdcdefaults]
    kdc_ports = 88
    kdc_tcp_ports = 88

[realms]
    EXAMPLE.COM = {
        kadmind_port = 749
        max_life = 12h
        max_renewable_life = 7d
        master_key_type = aes256-cts-hmac-sha384-192
        supported_enctypes = aes256-cts-hmac-sha384-192:normal aes256-cts-hmac-sha1-96:normal
        default_principal_flags = +preauth
    }
```

## Ticket Management

### kinit / klist / kdestroy

```bash
# Obtain TGT with password
kinit jdoe@EXAMPLE.COM

# Obtain TGT with keytab (service accounts, cron jobs)
kinit -kt /etc/krb5.keytab HTTP/web.example.com@EXAMPLE.COM

# Obtain renewable ticket
kinit -r 7d jdoe@EXAMPLE.COM

# Renew ticket before expiry
kinit -R

# List cached tickets
klist
# Output:
# Ticket cache: KEYRING:persistent:1000:1000
# Default principal: jdoe@EXAMPLE.COM
#
# Valid starting       Expires              Service principal
# 04/03/2026 09:00:00  04/03/2026 19:00:00  krbtgt/EXAMPLE.COM@EXAMPLE.COM
#     renew until 04/10/2026 09:00:00

# List all cached credentials (verbose)
klist -e -f

# Destroy all cached tickets
kdestroy

# Destroy a specific cache
kdestroy -c KEYRING:persistent:1000:1000
```

### Keytab Management

```bash
# Create keytab for a service principal
kadmin -q "ktadd -k /etc/krb5.keytab HTTP/web.example.com"

# Create keytab with specific encryption types
kadmin -q "ktadd -k /etc/krb5.keytab -e aes256-cts-hmac-sha1-96 HTTP/web.example.com"

# List keytab contents
klist -kt /etc/krb5.keytab
# Output:
# Keytab name: FILE:/etc/krb5.keytab
# KVNO Timestamp           Principal
# ---- ------------------- ------------------------------------------------------
#    3 04/03/2026 08:00:00 HTTP/web.example.com@EXAMPLE.COM

# Merge keytabs
ktutil
# ktutil: read_kt /etc/old.keytab
# ktutil: read_kt /etc/new.keytab
# ktutil: write_kt /etc/merged.keytab
# ktutil: quit

# Remove old keys from keytab (keep only latest KVNO)
kadmin -q "ktremove -k /etc/krb5.keytab HTTP/web.example.com old"
```

## kadmin — KDC Administration

### Principal Management

```bash
# Connect to kadmin
kadmin -p admin/admin@EXAMPLE.COM

# List principals
kadmin -q "listprincs"
kadmin -q "listprincs *http*"

# Add user principal
kadmin -q "addprinc jdoe@EXAMPLE.COM"

# Add service principal (no password, keytab only)
kadmin -q "addprinc -randkey HTTP/web.example.com@EXAMPLE.COM"

# Change password
kadmin -q "cpw jdoe@EXAMPLE.COM"

# View principal details
kadmin -q "getprinc jdoe@EXAMPLE.COM"

# Modify principal attributes
kadmin -q "modprinc -maxlife 24h -maxrenewlife 7d jdoe@EXAMPLE.COM"

# Lock/unlock principal
kadmin -q "modprinc -allow_tix jdoe@EXAMPLE.COM"    # lock
kadmin -q "modprinc +allow_tix jdoe@EXAMPLE.COM"    # unlock

# Delete principal
kadmin -q "delprinc jdoe@EXAMPLE.COM"

# Local kadmin (no network, direct DB access)
kadmin.local -q "listprincs"
```

### Policy Management

```bash
# Create password policy
kadmin -q "addpol -minlength 12 -minclasses 3 -maxlife 90days -history 10 strong"

# Apply policy to principal
kadmin -q "modprinc -policy strong jdoe@EXAMPLE.COM"

# List policies
kadmin -q "listpols"

# View policy details
kadmin -q "getpol strong"
```

## Cross-Realm Trust

### Establishing Trust

```bash
# On EXAMPLE.COM KDC — create trust principal
kadmin.local -q "addprinc -pw 'shared_secret_here' krbtgt/PARTNER.ORG@EXAMPLE.COM"

# On PARTNER.ORG KDC — create matching principal
kadmin.local -q "addprinc -pw 'shared_secret_here' krbtgt/EXAMPLE.COM@PARTNER.ORG"

# Verify cross-realm — from EXAMPLE.COM client
kinit jdoe@EXAMPLE.COM
kvno host/server.partner.org@PARTNER.ORG
klist
# Should show:
# krbtgt/PARTNER.ORG@EXAMPLE.COM  (cross-realm TGT)
# host/server.partner.org@PARTNER.ORG  (service ticket)
```

## SPNEGO (HTTP Negotiate)

### Apache Configuration

```apache
# /etc/httpd/conf.d/kerberos.conf
<Location /protected>
    AuthType GSSAPI
    AuthName "Kerberos SSO"
    GssapiCredStore keytab:/etc/httpd/http.keytab
    GssapiAllowedMech krb5
    GssapiNegotiateOnce on
    GssapiUseSessions on
    Session On
    SessionCookieName gssapi_session path=/protected;httponly;secure
    Require valid-user
</Location>
```

### Testing SPNEGO

```bash
# Get ticket and test with curl
kinit jdoe@EXAMPLE.COM
curl --negotiate -u : https://web.example.com/protected

# Debug SPNEGO
KRB5_TRACE=/dev/stderr curl --negotiate -u : https://web.example.com/protected
```

## Diagnostics

### Debugging

```bash
# Enable trace logging
export KRB5_TRACE=/dev/stderr

# Test connectivity to KDC
echo "" | kinit -V jdoe@EXAMPLE.COM 2>&1

# Verify DNS SRV records
dig _kerberos._udp.example.com SRV
dig _kerberos-adm._tcp.example.com SRV

# Check clock skew (must be < 5 minutes)
ntpstat
date -u

# Test keytab authentication
kinit -kt /etc/krb5.keytab HTTP/web.example.com@EXAMPLE.COM
klist
```

## Tips

- Keep clock skew under 5 minutes between KDC and all clients -- Kerberos uses timestamps for replay protection and rejects tickets with excessive drift
- Use `aes256-cts-hmac-sha384-192` or `aes256-cts-hmac-sha1-96` encryption types -- disable RC4 (arcfour-hmac) and DES in production
- Set `default_ccache_name = KEYRING:persistent:%{uid}` to store tickets in the kernel keyring instead of `/tmp` files, preventing ticket theft via filesystem access
- Rotate keytabs regularly with `kadmin ktadd` -- each rotation increments the KVNO and invalidates old keys
- Set `+preauth` on all principals to require encrypted timestamp pre-authentication, preventing offline dictionary attacks against AS-REP
- Use `forwardable = true` only when delegation is needed (e.g., web servers accessing backend services on behalf of users)
- Configure at least two KDCs for high availability -- clients try each KDC listed in `krb5.conf` in order
- Use `kvno` to test service ticket acquisition without actually connecting to the service
- Set `rdns = false` in `krb5.conf` to prevent reverse DNS lookup issues from breaking service principal matching
- Never embed passwords in keytabs for user principals -- use `-randkey` for service principals and password-based kinit for users
- Monitor `krbtgt` account key version -- if compromised, an attacker can forge any ticket (Golden Ticket attack)
- Use cross-realm trust paths (`[capaths]`) to avoid transitive trust chains through untrusted intermediaries

## See Also

- ldap, saml, spnego, pam, sssd, freeipa, active-directory

## References

- [MIT Kerberos Documentation](https://web.mit.edu/kerberos/krb5-latest/doc/)
- [RFC 4120 — Kerberos V5](https://datatracker.ietf.org/doc/html/rfc4120)
- [RFC 4121 — Kerberos V5 GSSAPI](https://datatracker.ietf.org/doc/html/rfc4121)
- [RFC 4559 — SPNEGO HTTP Authentication](https://datatracker.ietf.org/doc/html/rfc4559)
- [Red Hat Kerberos Guide](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/managing_idm_users_groups_hosts_and_access_control_rules/)
- [Active Directory Kerberos](https://learn.microsoft.com/en-us/windows-server/security/kerberos/kerberos-authentication-overview)
