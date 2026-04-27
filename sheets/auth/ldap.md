# LDAP (Lightweight Directory Access Protocol, RFC 4511 / 4519)

> Hierarchical directory protocol for authentication, authorization, and identity data — querying, modifying, and replicating entries in a Directory Information Tree across OpenLDAP, 389 Directory Server, Active Directory, ApacheDS, and OpenDJ backends with LDIF, search filters, SASL, TLS, ACLs, syncrepl, sssd integration, schema design, and operational tooling for production directories.

## Quick Reference

```bash
# Search a subtree, anonymous, plain LDAP
ldapsearch -x -H ldap://ldap.example.com -b 'dc=example,dc=com' '(uid=alice)'

# Authenticated search, prompt for password, LDAPS
ldapsearch -x -H ldaps://ldap.example.com:636 \
  -D 'cn=admin,dc=example,dc=com' -W \
  -b 'ou=People,dc=example,dc=com' '(objectClass=inetOrgPerson)'

# Add entries from LDIF
ldapadd -x -H ldap://localhost -D 'cn=admin,dc=example,dc=com' -W -f users.ldif

# Reset a user password
ldappasswd -x -H ldap://localhost -D 'cn=admin,dc=example,dc=com' -W \
  -s 'NewSecret123!' 'uid=alice,ou=People,dc=example,dc=com'

# Test bind / identity
ldapwhoami -x -H ldaps://ldap.example.com \
  -D 'uid=alice,ou=People,dc=example,dc=com' -W
```

## LDAP in 60 Seconds

LDAP is a stateful TCP/389 (or TCP/636 for LDAPS) protocol for reading and writing entries in a Directory Information Tree (DIT). Each entry is identified by a Distinguished Name (DN) and contains typed attributes governed by an object-class schema. Clients establish a session with `BIND` (anonymous, simple, or SASL), perform `SEARCH` / `ADD` / `MODIFY` / `DELETE` / `MODIFYDN` / `COMPARE` operations, and tear down with `UNBIND`. The wire encoding is BER (ASN.1), but operators interact via LDIF (a text serialization) and the `ldap*` command-line tools.

The standard backends are OpenLDAP (`slapd`), Red Hat 389 Directory Server (`ns-slapd`), Microsoft Active Directory (`lsass`/NTDS), Apache Directory Server (`apacheds`), and ForgeRock OpenDJ. They all speak RFC 4511 LDAPv3 with vendor extensions (e.g., AD's range retrieval, OpenLDAP's `cn=config`, 389-DS replication agreements, schema definitions in OID arcs `1.3.6.1.4.1.*`).

LDAP is the operating-system substrate for centralized POSIX accounts (RFC 2307 schema → `sssd`/`nslcd`), enterprise authentication (AD domain join via Kerberos and LDAP), and a fading legacy of in-app authentication that has largely been displaced by OAuth/OIDC, SAML, and SCIM. It still anchors most "single source of truth" identity stores in 2026.

## DIT Structure

The DIT is a strict tree. Every entry has exactly one parent, identified by the DN of that parent. The leaf of every DN is the entry's RDN (Relative Distinguished Name).

```
                       dc=com
                          |
                    dc=example,dc=com
       __________________|__________________
      |              |          |           |
ou=People       ou=Groups   ou=Services  ou=Hosts
   |                |           |            |
uid=alice       cn=admins   cn=postgres   cn=web01
uid=bob         cn=devs     cn=mail       cn=db01
uid=charlie     cn=ops      cn=ldap-ro
```

### Common Roots

```
dc=example,dc=com    # Domain-component (most common; mirrors DNS)
o=Example,c=US       # Country/organization (X.500 legacy)
o=Example            # Single-org (ApacheDS sample)
```

### Standard Suborganizational Units

```
ou=People,dc=example,dc=com           # Human users
ou=Groups,dc=example,dc=com           # Posix and access groups
ou=Services,dc=example,dc=com         # Service accounts (bind DNs, app users)
ou=Hosts,dc=example,dc=com            # Workstation/server entries
ou=Roles,dc=example,dc=com            # RBAC role objects
ou=Sudoers,dc=example,dc=com          # sudo rules (sudo-ldap schema)
ou=AutomountMaps,dc=example,dc=com    # autofs maps
ou=DNS,dc=example,dc=com              # bind9-dlz / 389-DS DNS plugin
ou=System,dc=example,dc=com           # Replication agreements, control entries
```

### Active Directory Default Layout

```
DC=example,DC=com
├── CN=Users               (default container, NOT an OU — cannot apply GPO)
├── CN=Computers           (joined workstations)
├── CN=Builtin             (BUILTIN\Administrators etc.)
├── CN=System              (DNS zones, DFSR, replication metadata)
├── OU=Domain Controllers  (DCs — required exact name)
└── OU=Corp                (admin-created hierarchy)
    ├── OU=Users
    ├── OU=Groups
    └── OU=Servers
```

## Distinguished Names

A DN is the complete leaf-to-root path expressed as comma-separated RDNs. RFC 4514 defines the string form, including escaping rules for `,`, `=`, `+`, `<`, `>`, `;`, `\`, `"`, leading/trailing whitespace, and NUL.

```
DN:   uid=alice,ou=People,dc=example,dc=com
RDN:  uid=alice                 # leftmost component
parent DN: ou=People,dc=example,dc=com
```

### RDN Variants

```
uid=alice                                # single-valued RDN
cn=Alice Smith+uid=alice                 # multi-valued RDN (uncommon, both unique together)
cn=Smith\, John                          # comma in CN, escaped
cn=#34616263                             # binary RDN, hex-encoded
```

### Reserved Characters

```
,  -> \,        =  -> \=        +  -> \+
<  -> \<        >  -> \>        ;  -> \;
\  -> \\        "  -> \"
leading space    -> \ x
trailing space   -> x\
leading #        -> \#
```

### Normalization

Most servers normalize DNs case-insensitively for the attribute name and use the attribute's matching rule for the value. `uid=Alice` and `UID=alice` typically refer to the same entry; do not rely on this in scripts — store and compare DNs as your server emits them.

## Object Classes and Attributes

Every entry must have a `structural` object class plus zero-or-more `auxiliary` object classes. Object classes are inherited from `top` through a directed acyclic graph.

```
                    top
                     |
                +----+----+
                |         |
              person   organizationalUnit
                |
        organizationalPerson
                |
           inetOrgPerson           (RFC 2798 — adds mail, etc.)
                |
       (auxiliary) posixAccount     (RFC 2307 — adds uidNumber, etc.)
                |
       (auxiliary) shadowAccount    (RFC 2307 — adds shadowExpire, etc.)
```

### Required vs Optional Per Class

```
person              MUST: cn, sn
                    MAY:  userPassword, telephoneNumber, seeAlso, description

organizationalPerson  MAY: title, ou, postalAddress, l, st, postalCode

inetOrgPerson       MAY: mail, employeeNumber, displayName, givenName,
                         labeledURI, mobile, jpegPhoto, manager, ...

posixAccount        MUST: cn, uid, uidNumber, gidNumber, homeDirectory
                    MAY:  loginShell, gecos, description

posixGroup          MUST: cn, gidNumber
                    MAY:  memberUid, userPassword, description

groupOfNames        MUST: cn, member          (1+ DNs)
                    MAY:  owner, description

groupOfUniqueNames  MUST: cn, uniqueMember    (DN + UID number)

shadowAccount       MUST: uid
                    MAY:  shadowMin, shadowMax, shadowWarning, shadowExpire,
                          shadowFlag, shadowInactive, shadowLastChange
```

### Common Attributes

```
# Naming
cn          common name              "Alice Smith"
sn          surname                  "Smith"
givenName   first name               "Alice"
displayName preferred display        "Alice S."
initials    middle initials          "Q"

# Identity
uid         user ID (unix)           "alice"
mail        primary email            "alice@example.com"
mailAlternate   secondary mail
employeeNumber  HRIS ID
employeeType    contractor/full-time

# POSIX (RFC 2307)
uidNumber       integer UID          1001
gidNumber       integer GID          1001
homeDirectory   path                 /home/alice
loginShell      shell                /bin/bash
gecos           legacy GECOS         "Alice Smith,,,,"

# Group membership
memberOf        DN (computed by overlay/AD; reverse pointer)
member          DN (groupOfNames)
uniqueMember    DN (groupOfUniqueNames)
memberUid       string (posixGroup)

# Auth
userPassword            hashed password ({SSHA}, {ARGON2}, {CRYPT})
userCertificate;binary  X.509 cert
krbPrincipalName        Kerberos principal (mit-kerberos schema)

# AD-specific
sAMAccountName          legacy NetBIOS-era login (alice)
userPrincipalName       UPN (alice@example.com)
objectGUID              128-bit binary unique ID
objectSid               binary SID
distinguishedName       full DN (auto)
userAccountControl      bitmask (0x0202 = disabled)
pwdLastSet              FILETIME
accountExpires          FILETIME (0 / 9223372036854775807 = never)
servicePrincipalName    SPN list (Kerberos)
```

### AD `userAccountControl` Bits

```
0x0001  SCRIPT
0x0002  ACCOUNTDISABLE       (most-used: 514 = normal+disabled)
0x0008  HOMEDIR_REQUIRED
0x0010  LOCKOUT
0x0020  PASSWD_NOTREQD
0x0040  PASSWD_CANT_CHANGE
0x0080  ENCRYPTED_TEXT_PWD_ALLOWED
0x0100  TEMP_DUPLICATE_ACCOUNT
0x0200  NORMAL_ACCOUNT       (most-used: 512 = normal user)
0x0800  INTERDOMAIN_TRUST_ACCOUNT
0x1000  WORKSTATION_TRUST_ACCOUNT
0x2000  SERVER_TRUST_ACCOUNT
0x10000 DONT_EXPIRE_PASSWORD
0x20000 MNS_LOGON_ACCOUNT
0x40000 SMARTCARD_REQUIRED
0x80000 TRUSTED_FOR_DELEGATION
0x100000 NOT_DELEGATED
0x200000 USE_DES_KEY_ONLY
0x400000 DONT_REQ_PREAUTH
0x800000 PASSWORD_EXPIRED
0x1000000 TRUSTED_TO_AUTH_FOR_DELEGATION
```

## Schema Files

### Standard Schemas

```
core.schema           RFC 4519   top, person, organizationalUnit, ...
cosine.schema         RFC 4524   COSINE Pilot Service (host, document, ...)
inetorgperson.schema  RFC 2798   inetOrgPerson
nis.schema            RFC 2307   posixAccount, posixGroup, shadowAccount
misc.schema                      misc utilities
java.schema                      JNDI / Java directory entries
sudo.schema                      sudo-ldap rules
openssh-lpk.schema               sshPublicKey attr (AuthorizedKeysCommand)
ppolicy.schema                   password policy (RFC draft)
collective.schema     RFC 3671   collective attributes
```

### Schema Hierarchy ASCII

```
ATTRIBUTE TYPES                 OBJECT CLASSES
---------------                 ---------------
1.3.6.1.4.1.1466.115.121.1.*    top (abstract)
   syntaxes                       |
        |                         +-- person (structural)
   matchingRules                  |     |
        |                         |     +-- organizationalPerson
   attributeType                  |           |
   (NAME, OID, SYNTAX,            |           +-- inetOrgPerson
    EQUALITY, ORDERING,           |
    SUBSTR, SINGLE-VALUE)         +-- groupOfNames (structural)
                                  +-- posixAccount (auxiliary)
                                  +-- shadowAccount (auxiliary)
                                  +-- top  -> dcObject -> domain
```

### Custom Schema (OpenLDAP cn=schema)

```ldif
dn: cn=acme,cn=schema,cn=config
objectClass: olcSchemaConfig
cn: acme
olcAttributeTypes: ( 1.3.6.1.4.1.99999.1.1
  NAME 'acmeBadgeId'
  DESC 'Physical badge identifier'
  EQUALITY caseIgnoreMatch
  SUBSTR caseIgnoreSubstringsMatch
  SYNTAX 1.3.6.1.4.1.1466.115.121.1.15
  SINGLE-VALUE )
olcObjectClasses: ( 1.3.6.1.4.1.99999.2.1
  NAME 'acmeEmployee'
  DESC 'ACME employee auxiliary'
  SUP top
  AUXILIARY
  MAY ( acmeBadgeId $ acmeOffice ) )
```

## Search Filters (RFC 4515)

### Filter Syntax Cheat Sheet

```
(attr=value)                  equality
(attr=val*ue)                 substring (initial, any, final)
(attr=*)                      presence
(attr~=value)                 approximate (soundex/metaphone, optional)
(attr>=value)                 greaterOrEqual (lexical, not numeric)
(attr<=value)                 lessOrEqual
(&(f1)(f2)(f3))               AND
(|(f1)(f2)(f3))               OR
(!(f1))                       NOT
(attr:dn:caseExactMatch:=v)   extensible
(attr:1.2.840.113556.1.4.1941:=v)   AD LDAP_MATCHING_RULE_IN_CHAIN
(attr:1.2.840.113556.1.4.803:=2)    AD bitwise AND
(attr:1.2.840.113556.1.4.804:=2)    AD bitwise OR
```

### Worked Examples

```bash
# All inetOrgPerson entries with email
ldapsearch -x -b 'dc=example,dc=com' '(&(objectClass=inetOrgPerson)(mail=*))'

# Users with surname Smith OR Jones
ldapsearch -x -b 'ou=People,dc=example,dc=com' '(|(sn=Smith)(sn=Jones))'

# Users not in /sbin/nologin
ldapsearch -x -b 'ou=People,dc=example,dc=com' '(!(loginShell=/sbin/nologin))'

# All members of a static group
ldapsearch -x -b 'cn=admins,ou=Groups,dc=example,dc=com' \
  -s base '(objectClass=*)' member

# Reverse: groups that contain alice
ldapsearch -x -b 'ou=Groups,dc=example,dc=com' \
  '(member=uid=alice,ou=People,dc=example,dc=com)'

# AD: nested group membership (LDAP_MATCHING_RULE_IN_CHAIN)
ldapsearch -x -b 'DC=example,DC=com' \
  '(memberOf:1.2.840.113556.1.4.1941:=CN=Domain Admins,CN=Users,DC=example,DC=com)'

# AD: disabled accounts (UAC bit 0x2)
ldapsearch -x -b 'DC=example,DC=com' \
  '(userAccountControl:1.2.840.113556.1.4.803:=2)'

# AD: never-expire passwords (UAC bit 0x10000)
ldapsearch -x -b 'DC=example,DC=com' \
  '(userAccountControl:1.2.840.113556.1.4.803:=65536)'

# Users created in last 7 days (AD whenCreated, GeneralizedTime)
# 2026-04-20 00:00:00 UTC
ldapsearch -x -b 'DC=example,DC=com' '(whenCreated>=20260420000000.0Z)'

# uidNumber range (RFC 2307 POSIX UIDs 1000-1999)
ldapsearch -x -b 'ou=People,dc=example,dc=com' \
  '(&(uidNumber>=1000)(uidNumber<=1999))'

# Email domain match
ldapsearch -x -b 'dc=example,dc=com' '(mail=*@example.com)'

# Empty groups
ldapsearch -x -b 'ou=Groups,dc=example,dc=com' \
  '(&(objectClass=groupOfNames)(!(member=*)))'

# Locked accounts (OpenLDAP ppolicy)
ldapsearch -x -b 'ou=People,dc=example,dc=com' \
  '(pwdAccountLockedTime=*)'

# AD: stale accounts (lastLogonTimestamp older than 90d)
# 2026-01-27 in FILETIME (100ns since 1601-01-01)
ldapsearch -x -b 'DC=example,DC=com' \
  '(&(objectClass=user)(lastLogonTimestamp<=133458912000000000))'
```

### Filter Parsing Tips

- Filters are LISP-like: every `(` must have a matching `)`.
- The outer parens are required. `uid=alice` without parens is a syntax error in `ldapsearch`.
- Special characters in values must be backslash-hex escaped:
  `( -> \28`, `) -> \29`, `* -> \2a`, `\ -> \5c`, NUL -> `\00`.

```bash
# Search for a literal asterisk in cn (escape)
ldapsearch -x -b 'dc=example,dc=com' '(cn=*\2a*)'
```

## Operations (RFC 4511)

```
BIND       authenticate (anonymous, simple DN+pw, or SASL)
UNBIND     terminate session (no response)
SEARCH     query the DIT
COMPARE    test attribute=value (returns true/false without disclosing value)
ADD        create new entry
DELETE     remove entry (must be leaf or use subtree-delete control)
MODIFY     change attributes (add/replace/delete on existing entry)
MODIFYDN   rename / move (changes RDN, optionally moves under new parent)
ABANDON    cancel in-flight async operation
EXTENDED   vendor/standard extensions (StartTLS, Password Modify, Cancel,
           Who Am I, Notice of Disconnection, ...)
```

### Search Scope (RFC 4511 §4.5.1.2)

```
base / 0       only the base DN entry itself
one  / 1       immediate children only
sub  / 2       full subtree from base
children / 3   subordinate (subtree minus base) — RFC 5805
```

```bash
# base
ldapsearch -x -b 'uid=alice,ou=People,dc=example,dc=com' -s base '(objectClass=*)'

# one
ldapsearch -x -b 'ou=People,dc=example,dc=com' -s one '(objectClass=*)'

# sub (default)
ldapsearch -x -b 'dc=example,dc=com' -s sub '(uid=alice)'
```

### Dereferencing Aliases (`-a`)

```
never    do not dereference (default)
search   dereference only when searching subordinates
find     dereference base only
always   dereference everywhere
```

## Authentication

### BIND Methods

```
Anonymous          no DN, no password         BIND with empty values
Simple             DN + cleartext password    requires TLS for safety
SASL EXTERNAL      X.509 client certificate   over TLS, no password
SASL PLAIN         RFC 4616, cleartext        TLS-only (effectively the same as simple)
SASL DIGEST-MD5    RFC 2831, deprecated       MD5 challenge-response
SASL CRAM-MD5      RFC 2195, deprecated       challenge-response
SASL SCRAM-SHA-1   RFC 5802                   modern challenge-response
SASL SCRAM-SHA-256 RFC 7677                   stronger SCRAM
SASL GSSAPI        Kerberos v5 ticket         enterprise (AD, FreeIPA)
SASL GS2-KRB5      RFC 5801, modern GSSAPI    Kerberos with channel binding
```

### LDAP Bind Flow (Simple over TLS)

```
Client                                    Server
  |  TCP SYN ----------------------------->|
  |<------------------ SYN-ACK ------------|
  |  ACK --------------------------------->|
  |  StartTLS extended request ----------->|     [or TCP/636 LDAPS]
  |<--------------- StartTLS success ------|
  |  TLS ClientHello -------------------->|
  |<------------- TLS ServerHello, cert ---|
  |  TLS key exchange ------------------->|
  |<------------- TLS Finished ------------|
  |  BindRequest(simple, DN, password) -->|
  |<------------- BindResponse(success) ---|
  |  SearchRequest --------------------->| ... |
  |<------------- SearchResult -----------|
  |  UnbindRequest ---------------------->|
  |  TCP FIN --------------------------->|
```

### Anonymous Bind

```bash
ldapsearch -x -H ldap://localhost -b '' -s base '(objectClass=*)' \
  namingContexts subschemaSubentry supportedLDAPVersion supportedSASLMechanisms
```

### LDAPS vs StartTLS

```
ldaps://host:636     TLS wraps the entire TCP session (TLS-on-connect)
ldap://host:389  +   StartTLS extended op upgrades cleartext to TLS
```

Both are equivalent for confidentiality once TLS is up. StartTLS lets you use a single port (389) for cleartext, opportunistic TLS, and anonymous probes. LDAPS is preferred for load balancers that cannot rewrite the StartTLS extended op.

## OpenLDAP CLI Tooling

### ldapsearch

```bash
# Common flags
ldapsearch \
  -x                    # simple bind (no SASL)
  -H ldaps://host       # URI (use ldap:// + -ZZ for StartTLS)
  -ZZ                   # require StartTLS, fail otherwise
  -D 'cn=admin,...'     # bind DN
  -w 'secret'           # password literal (insecure on shared hosts)
  -W                    # prompt for password
  -y /path/to/pwfile    # read password from file
  -b 'dc=example,dc=com'      # search base
  -s sub                # scope: base | one | sub | children
  -l 30                 # time limit (seconds)
  -z 1000               # size limit (entries)
  -L                    # LDIFv1 (no comments)
  -LL                   # LDIFv1 (no comments, no version)
  -LLL                  # LDIFv1 (no comments, no version, no metadata)
  -E pr=500/noprompt    # paged results, page size 500
  -E sss=cn             # server-side sort by cn
  '(objectClass=*)'     # filter (mandatory)
  cn mail uidNumber     # attributes to return (default: all user attrs)
```

```text
# Expected output (LDIFv1 with -LLL)
dn: uid=alice,ou=People,dc=example,dc=com
cn: Alice Smith
mail: alice@example.com
uidNumber: 1001
```

### ldapadd / ldapmodify

```bash
# Add from LDIF
ldapadd -x -H ldap://localhost \
  -D 'cn=admin,dc=example,dc=com' -W \
  -f new-users.ldif

# Modify from LDIF (replace, add, delete operations)
ldapmodify -x -H ldap://localhost \
  -D 'cn=admin,dc=example,dc=com' -W \
  -f changes.ldif

# Continue on error (-c) and verbose (-v)
ldapmodify -cv -x -D 'cn=admin,...' -W -f changes.ldif

# Read LDIF from stdin
ldapmodify -x -D 'cn=admin,...' -W <<'EOF'
dn: uid=alice,ou=People,dc=example,dc=com
changetype: modify
replace: mail
mail: alice@newdomain.com
EOF
```

```text
# Successful add
adding new entry "uid=alice,ou=People,dc=example,dc=com"

# Successful modify
modifying entry "uid=alice,ou=People,dc=example,dc=com"
```

### ldapdelete

```bash
ldapdelete -x -D 'cn=admin,dc=example,dc=com' -W \
  'uid=bob,ou=People,dc=example,dc=com'

# Recursive subtree delete (server must support tree-delete control 1.2.840.113556.1.4.805)
ldapdelete -x -D 'cn=admin,...' -W \
  -M -E 1.2.840.113556.1.4.805 \
  'ou=Defunct,dc=example,dc=com'
```

### ldapmodrdn

```bash
# Rename (change the RDN of an entry, keep parent)
ldapmodrdn -x -D 'cn=admin,dc=example,dc=com' -W \
  'uid=bob,ou=People,dc=example,dc=com' 'uid=robert'

# Rename and remove the old RDN attribute (-r)
ldapmodrdn -r -x -D 'cn=admin,...' -W \
  'uid=bob,ou=People,dc=example,dc=com' 'uid=robert'

# Move + rename (-s newSuperior)
ldapmodrdn -r -s 'ou=Alumni,dc=example,dc=com' \
  -x -D 'cn=admin,...' -W \
  'uid=alice,ou=People,dc=example,dc=com' 'uid=alice'
```

### ldappasswd

```bash
# Reset to a specified password
ldappasswd -x -H ldaps://ldap.example.com \
  -D 'cn=admin,dc=example,dc=com' -W \
  -s 'NewSecret123!' \
  'uid=alice,ou=People,dc=example,dc=com'

# Generate a random password (server returns it)
ldappasswd -x -D 'cn=admin,...' -W \
  'uid=alice,ou=People,dc=example,dc=com'

# User changes their own password
ldappasswd -x -D 'uid=alice,ou=People,dc=example,dc=com' \
  -w 'OldSecret' \
  -s 'NewSecret123!'
```

```text
# Expected on random-password response
New password: 9bX3uL7eK!q2Vw
```

### ldapwhoami

```bash
ldapwhoami -x -H ldaps://ldap.example.com \
  -D 'uid=alice,ou=People,dc=example,dc=com' -W
```

```text
dn:uid=alice,ou=People,dc=example,dc=com
```

### ldapcompare

```bash
# Returns TRUE/FALSE/UNKNOWN — useful for password verification without disclosing
ldapcompare -x -D 'cn=admin,dc=example,dc=com' -W \
  'uid=alice,ou=People,dc=example,dc=com' 'uid:alice'
```

```text
TRUE
```

### ldapurl

```bash
# Build a URL
ldapurl -h ldap.example.com -p 636 -S ldaps \
  -b 'ou=People,dc=example,dc=com' \
  -s sub -f '(uid=alice)' -a cn,mail
```

```text
ldaps://ldap.example.com:636/ou=People,dc=example,dc=com?cn,mail?sub?(uid=alice)
```

### slapcat / slapadd / slapindex / slapdn / slapauth

```bash
# Export the directory (offline, fast, ignores ACLs)
slapcat -n 1 -l backup-$(date -u +%Y%m%d).ldif

# Bulk import (must be offline; faster than ldapadd)
systemctl stop slapd
slapadd -n 1 -l import.ldif
chown -R openldap:openldap /var/lib/ldap
systemctl start slapd

# Rebuild indexes after schema/index change
slapindex -n 1
chown -R openldap:openldap /var/lib/ldap

# Parse and normalize a DN
slapdn 'UID=alice,OU=People,DC=example,DC=com'
# -> uid=alice,ou=People,dc=example,dc=com

# Check what bind the SASL authcid maps to
slapauth -X 'u:alice'
```

## LDIF Format (RFC 2849)

LDIF is line-oriented UTF-8. Lines beginning with a space are continuations. Blank lines separate records. Values containing non-printable bytes use `attr:: <base64>`.

### Add Entry

```ldif
version: 1

dn: uid=alice,ou=People,dc=example,dc=com
changetype: add
objectClass: top
objectClass: inetOrgPerson
objectClass: posixAccount
objectClass: shadowAccount
uid: alice
cn: Alice Smith
sn: Smith
givenName: Alice
displayName: Alice Smith
mail: alice@example.com
employeeNumber: 8842
uidNumber: 1001
gidNumber: 1001
homeDirectory: /home/alice
loginShell: /bin/bash
gecos: Alice Smith
userPassword: {SSHA}xQ6mlbvP6vXx5lz7zYUR8E7F8gS6sdfQ
shadowLastChange: 19840
shadowMin: 0
shadowMax: 90
shadowWarning: 7
```

### Modify Entry (Multiple Changes)

```ldif
dn: uid=alice,ou=People,dc=example,dc=com
changetype: modify
replace: mail
mail: alice@newdomain.com
-
add: telephoneNumber
telephoneNumber: +1-555-0100
telephoneNumber: +1-555-0101
-
delete: description
-
add: objectClass
objectClass: shadowAccount
-
add: shadowExpire
shadowExpire: 19999
```

### Delete Entry

```ldif
dn: uid=charlie,ou=People,dc=example,dc=com
changetype: delete
```

### Rename / Move

```ldif
dn: uid=alice,ou=People,dc=example,dc=com
changetype: modrdn
newrdn: uid=alice.smith
deleteoldrdn: 1
newsuperior: ou=Alumni,dc=example,dc=com
```

### Add Group with Members

```ldif
dn: cn=admins,ou=Groups,dc=example,dc=com
changetype: add
objectClass: top
objectClass: groupOfNames
cn: admins
description: Administrators
member: uid=alice,ou=People,dc=example,dc=com
member: uid=bob,ou=People,dc=example,dc=com
```

### Add Member to Existing Group

```ldif
dn: cn=admins,ou=Groups,dc=example,dc=com
changetype: modify
add: member
member: uid=charlie,ou=People,dc=example,dc=com
```

### Add posixGroup

```ldif
dn: cn=engineers,ou=Groups,dc=example,dc=com
changetype: add
objectClass: top
objectClass: posixGroup
cn: engineers
gidNumber: 2001
memberUid: alice
memberUid: bob
```

### Add Sudoer

```ldif
dn: cn=alice,ou=Sudoers,dc=example,dc=com
changetype: add
objectClass: top
objectClass: sudoRole
cn: alice
sudoUser: alice
sudoHost: ALL
sudoCommand: ALL
sudoOption: !authenticate
```

### Binary Attribute (jpegPhoto)

```ldif
dn: uid=alice,ou=People,dc=example,dc=com
changetype: modify
add: jpegPhoto
jpegPhoto:< file:///tmp/alice.jpg
```

```ldif
# Or inline base64
dn: uid=alice,ou=People,dc=example,dc=com
changetype: modify
add: jpegPhoto
jpegPhoto:: /9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQg...
```

### Folded Long Lines

```ldif
description: This is a very long description that exceeds 76 columns so we
 fold it with a leading space on continuation lines per RFC 2849 section 2.
```

### Comments

```ldif
# Anything after a leading '#' is a comment
# until end of line. Comments are ignored.
```

## OpenLDAP Configuration (cn=config)

OpenLDAP 2.4+ stores configuration as LDAP entries under `cn=config`. The legacy `slapd.conf` is supported but deprecated.

### Browse cn=config

```bash
# List all config DNs (root via ldapi:// + EXTERNAL)
ldapsearch -Y EXTERNAL -H ldapi:/// -b cn=config dn

# Inspect a specific entry
ldapsearch -Y EXTERNAL -H ldapi:/// -b 'olcDatabase={1}mdb,cn=config'
```

```text
dn: cn=config
dn: cn=module{0},cn=config
dn: cn=schema,cn=config
dn: cn={0}core,cn=schema,cn=config
dn: cn={1}cosine,cn=schema,cn=config
dn: cn={2}nis,cn=schema,cn=config
dn: cn={3}inetorgperson,cn=schema,cn=config
dn: olcBackend={0}mdb,cn=config
dn: olcDatabase={-1}frontend,cn=config
dn: olcDatabase={0}config,cn=config
dn: olcDatabase={1}mdb,cn=config
```

### Set Log Level

```bash
ldapmodify -Y EXTERNAL -H ldapi:/// <<'EOF'
dn: cn=config
changetype: modify
replace: olcLogLevel
olcLogLevel: stats sync
EOF
```

```
none      0       no logging
trace     1       function call trace
packets   2       packet handling
args      4       heavy trace + args
conns     8       connection management
BER       16      BER encoding/decoding
filter    32      search filter processing
config    64      configuration file processing
ACL       128     access control list processing
stats     256     stats log connections/operations/results
stats2    512     stats log entries sent
shell     1024    print communication with shell backends
parse     2048    entry parsing
sync      16384   syncrepl consumer
```

### Add Index

```bash
ldapmodify -Y EXTERNAL -H ldapi:/// <<'EOF'
dn: olcDatabase={1}mdb,cn=config
changetype: modify
add: olcDbIndex
olcDbIndex: mail eq,sub
-
add: olcDbIndex
olcDbIndex: memberOf eq
-
add: olcDbIndex
olcDbIndex: uidNumber eq
EOF
```

### Index Types

```
pres       presence (attr=*)
eq         equality (attr=value)
approx     approximate (~=, soundex)
sub        substring (initial, any, final)
sub_initial   only (val*)
sub_any       only (*val*)
sub_final     only (*val)
```

### Set Size / Time Limits

```bash
ldapmodify -Y EXTERNAL -H ldapi:/// <<'EOF'
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcSizeLimit
olcSizeLimit: 5000
-
replace: olcTimeLimit
olcTimeLimit: 30
EOF
```

### ACLs

```ldif
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcAccess
olcAccess: {0}to attrs=userPassword,shadowLastChange
  by self =xw
  by anonymous auth
  by dn.exact="cn=admin,dc=example,dc=com" write
  by * none
olcAccess: {1}to dn.subtree="ou=People,dc=example,dc=com" attrs=mail,telephoneNumber
  by self write
  by users read
  by * none
olcAccess: {2}to *
  by self read
  by dn.exact="cn=admin,dc=example,dc=com" write
  by users read
  by * none
```

### ACL Privileges

```
=     replace (clear and set)
+     add to existing
-     remove from existing

w     write
r     read
s     search
c     compare
x     authenticate (bind)
d     disclose
0     none
auth  search + compare + authenticate
read  + read
write + write
manage  + delete + entry creation/deletion
```

### Backends

```
mdb      LMDB-backed (default in 2.4+, recommended; mmap'd, fast, transactional)
hdb      hash-DB (deprecated, stuck on bdb 4.x)
bdb      Berkeley DB (deprecated; pre-mdb)
ldif     flat-file LDIF backend (slow; for read-only or test)
sql      ODBC bridge to RDBMS
monitor  cn=monitor (operational stats)
config   cn=config itself
null     /dev/null backend (test)
```

### Module Loading

```ldif
dn: cn=module{0},cn=config
changetype: modify
add: olcModuleLoad
olcModuleLoad: memberof.la
-
add: olcModuleLoad
olcModuleLoad: refint.la
-
add: olcModuleLoad
olcModuleLoad: ppolicy.la
```

### Overlays

```
memberof    Maintain reverse memberOf attribute on user entries
refint      Referential integrity (delete user -> remove from groups)
ppolicy     Password policy (lockout, complexity, expiry)
syncprov    Syncrepl provider (replication producer)
unique      Enforce attribute uniqueness across subtree
accesslog   Audit/changelog database
auditlog    Append-only audit LDIF
constraint  Regex/uri/count constraints on attribute values
chain       Chain referrals on writes
collect     Collective attribute service
dynlist     Dynamic group expansion
glue        Subordinate database stitching
nestgroup   Nested groupOfNames expansion
pcache      Proxy cache
ppolicy     Password policies
refint      Referential integrity
retcode     Force return codes for testing
rwm         Rewrite middleware
slapo-pbind protocol bind passthrough
syncprov    Replication
translucent Overlay one DB on another
unique      Uniqueness enforcement
valsort     Value-ordering hints
```

## Active Directory Specifics

### Ports

```
389/tcp     LDAP / StartTLS to LDAP
636/tcp     LDAPS
3268/tcp    Global Catalog (read-only forest-wide search, partial attrs)
3269/tcp    Global Catalog over TLS
88/tcp+udp  Kerberos KDC
464/tcp+udp kpasswd
135/tcp     RPC endpoint mapper (DCOM, replication)
445/tcp     SMB / CIFS / DFSR
49152-65535 Dynamic RPC range
```

### Schema Differences

```
sAMAccountName      legacy NT4-era login (max 20 chars)
userPrincipalName   modern UPN (alice@example.com)
displayName         "Alice Smith"
distinguishedName   auto-populated DN
objectGUID          binary 16-byte UUID
objectSid           binary SID
userAccountControl  bitmask (see above)
pwdLastSet          FILETIME (100-nanosecond intervals since 1601-01-01 UTC)
accountExpires      FILETIME (0 / 0x7FFFFFFFFFFFFFFF = never)
memberOf            computed by AD; do NOT modify directly (use member on group)
tokenGroups         operational; expanded transitive group SIDs (constructed)
primaryGroupID      RID of primary group (default 513 Domain Users)
```

### DNS SRV Records (Required for AD Discovery)

```
_ldap._tcp.example.com                       LDAP DCs
_ldap._tcp.dc._msdcs.example.com             writable DCs
_ldap._tcp.gc._msdcs.example.com             Global Catalogs
_ldap._tcp.<sitename>._sites.example.com     site-specific DCs
_kerberos._tcp.example.com                   Kerberos KDCs
_kpasswd._tcp.example.com                    password change service
_gc._tcp.example.com                         Global Catalogs (alt form)
_ldap._tcp.pdc._msdcs.example.com            PDC emulator
```

### Authenticate to AD via Kerberos

```bash
# 1. Obtain a TGT
echo 'NotMyRealPassword' | kinit alice@EXAMPLE.COM

# 2. Verify
klist

# 3. Search using GSSAPI
ldapsearch -Y GSSAPI -H ldap://dc01.example.com \
  -b 'DC=example,DC=com' '(sAMAccountName=alice)'
```

```text
# klist output
Ticket cache: KCM:1000
Default principal: alice@EXAMPLE.COM

Valid starting     Expires            Service principal
04/27/26 09:14:00  04/27/26 19:14:00  krbtgt/EXAMPLE.COM@EXAMPLE.COM
        renew until 05/04/26 09:14:00
```

### AD vs OpenLDAP vs 389-DS Side-by-Side

```
Feature              | OpenLDAP            | 389-DS              | Active Directory
---------------------|---------------------|---------------------|-----------------------
Config storage       | cn=config / mdb     | cn=config / dse.ldif| NTDS.dit / SYSVOL
Default port         | 389 / 636           | 389 / 636           | 389 / 636 / 3268 / 3269
Replication          | syncrepl (RFC 4533) | MMR (multi-master)  | DCDIAG / MMR with USN
ACL syntax           | olcAccess           | aci (per-entry)     | NTSecurityDescriptor
Schema files         | LDIF in cn=schema   | LDIF in cn=schema   | OID arc, baked in
Password policy      | ppolicy overlay     | password-policy plugin | Default Domain Policy
Group types          | groupOfNames,       | groupOfNames,       | security/distribution
                     | posixGroup          | groupOfUniqueNames  | global/universal/domain-local
SASL                 | Cyrus SASL          | Cyrus SASL          | SSPI (NTLM/Kerberos)
Reverse memberOf     | overlay             | virtual attr        | computed (back-link)
TLS implementation   | OpenSSL/GnuTLS      | NSS                 | Schannel
GC port              | n/a                 | n/a                 | 3268/3269
DNS integration      | optional            | optional            | required (AD DS)
Tooling              | ldap*, slap*        | dsconf, dsctl       | ADUC, dsa.msc, ldp.exe,
                                                                   dsquery, dsget, PowerShell
```

## 389 Directory Server

### Install and Create Instance

```bash
dnf install -y 389-ds-base

# Inline instance config
cat >/tmp/instance.inf <<'EOF'
[general]
config_version = 2

[slapd]
instance_name = example
root_password = AdminSecret123
self_sign_cert = True

[backend-userroot]
sample_entries = yes
suffix = dc=example,dc=com
EOF

dscreate from-file /tmp/instance.inf
```

### Lifecycle

```bash
dsctl slapd-example status
dsctl slapd-example start
dsctl slapd-example stop
dsctl slapd-example restart
dsctl slapd-example backup create
dsctl slapd-example backup list
dsctl slapd-example dbverify
dsctl slapd-example fsck
```

### Common dsconf

```bash
dsconf slapd-example backend list
dsconf slapd-example backend create --suffix=dc=example,dc=com --be-name=userRoot
dsconf slapd-example plugin list
dsconf slapd-example plugin memberof enable
dsconf slapd-example plugin memberof set --groupattr member --memberofattr memberOf
dsconf slapd-example replication enable --suffix=dc=example,dc=com --role=supplier --replica-id=1
dsconf slapd-example replication agreement create \
  --host=ds2.example.com --port=389 --conn-protocol=LDAP \
  --bind-dn='cn=replmgr,cn=config' --bind-passwd='ReplSecret' \
  --suffix=dc=example,dc=com --init agreement-1
dsconf slapd-example pwpolicy set --pwdmin=12 --pwdmax=90 --pwdwarn=7 --pwdlockout=on
dsconf slapd-example monitor server
```

### 389-DS ACI Example

```ldif
dn: ou=People,dc=example,dc=com
changetype: modify
add: aci
aci: (targetattr="userPassword || shadowLastChange")
     (version 3.0; acl "Self password write";
      allow (write) userdn = "ldap:///self";)
```

## Common Errors and Diagnostics

### LDAP Result Codes (RFC 4511 §A)

```
 0  success                          Operation succeeded
 1  operationsError                  Server operations error
 2  protocolError                    Bad protocol version, malformed PDU
 3  timeLimitExceeded                Server-side time limit
 4  sizeLimitExceeded                Server-side entry limit (use paging!)
 5  compareFalse                     Compare returned false
 6  compareTrue                      Compare returned true
 7  authMethodNotSupported           Bind method unsupported
 8  strongerAuthRequired             Server requires SASL/TLS
10  referral                         Refer client to another server
11  adminLimitExceeded               Admin-imposed limit
12  unavailableCriticalExtension     Critical control unsupported
13  confidentialityRequired          TLS required
14  saslBindInProgress
16  noSuchAttribute                  Attribute missing on modify/delete
17  undefinedAttributeType           Schema lookup failed
18  inappropriateMatching            Matching rule mismatch
19  constraintViolation              Value out of range / pwd policy
20  attributeOrValueExists           Add of duplicate attribute value
21  invalidAttributeSyntax           Value fails syntax check
32  noSuchObject                     DN does not exist
33  aliasProblem
34  invalidDNSyntax                  DN string malformed
36  aliasDereferencingProblem
48  inappropriateAuthentication      Anonymous required, etc.
49  invalidCredentials               Wrong password OR wrong DN
50  insufficientAccessRights         ACL denial
51  busy                             Server temporarily unable
52  unavailable                      Server shutting down
53  unwillingToPerform               Generic refusal (RO db, schema, ...)
54  loopDetected                     Alias/referral loop
64  namingViolation                  RDN violates schema/structure
65  objectClassViolation             Required attr missing or extra
66  notAllowedOnNonLeaf              Delete/modrdn on non-leaf without control
67  notAllowedOnRDN                  Cannot delete RDN attribute
68  entryAlreadyExists               Add over existing DN
69  objectClassModsProhibited        Cannot change structural class
71  affectsMultipleDSAs              Distributed op spans servers
80  other                            Vendor-specific catch-all
```

### Verbatim Errors and Fixes

```
ldap_bind: Invalid credentials (49)
        additional info: 80090308: LdapErr: DSID-0C0903A9, comment:
        AcceptSecurityContext error, data 52e, v23f0
```

`data 52e` = wrong password. Other AD subcodes:
```
525  user not found
52e  invalid credentials (wrong password)
530  not permitted to logon at this time
531  not permitted to logon at this workstation
532  password expired
533  account disabled
701  account expired
773  user must reset password
775  account locked
```

```
ldap_search_ext: No such object (32)
        matched DN: dc=example,dc=com
```
Fix: the DN portion below `matched DN` does not exist. Verify with:
```bash
ldapsearch -x -H ldap://localhost -b 'dc=example,dc=com' -s one dn
```

```
ldap_modify: Object class violation (65)
        additional info: attribute 'gidNumber' not allowed
```
Fix: add the auxiliary class that defines the attribute:
```ldif
dn: uid=alice,ou=People,dc=example,dc=com
changetype: modify
add: objectClass
objectClass: posixAccount
```

```
ldap_add: Constraint violation (19)
        additional info: Password fails quality checking policy
```
Fix: choose a longer password (≥ ppolicy `pwdMinLength`), more distinct chars, or unset history conflicts.

```
ldap_search_ext: Size limit exceeded (4)
```
Fix: use paging:
```bash
ldapsearch -x -E pr=500/noprompt -b 'dc=example,dc=com' '(objectClass=person)'
```

```
ldap_bind: Strong(er) authentication required (8)
        additional info: TLS confidentiality required
```
Fix: use `-ZZ` or switch to `ldaps://`.

```
ldap_search_ext: Referral (10)
        matched DN: DC=example,DC=com
        additional info: 0000202B: RefErr: DSID-0310083F, data 0,
        1 access points
                ref 1: 'corp.example.com'
```
Fix: AD chase referrals with `-C` or query the GC on 3268.

```
TLS: hostname does not match CN in peer certificate
```
Fix: ensure the URL host matches the cert's SAN/CN, or set:
```
TLS_REQCERT demand
TLS_CACERT /etc/pki/tls/certs/ca-bundle.crt
```
in `/etc/openldap/ldap.conf` and reissue cert with the right SAN.

```
ldap_sasl_interactive_bind_s: Local error (-2)
        additional info: SASL(-1): generic failure: GSSAPI Error:
        Unspecified GSS failure.  Minor code may provide more information
        (Server not found in Kerberos database)
```
Fix: missing SPN. On AD verify with:
```bash
setspn -L dc01
```
or for Linux KDC, `kadmin -q 'addprinc -randkey ldap/dc01.example.com'` and `ktutil` to extract the keytab.

```
ldapsearch: Invalid DN syntax (34)
        additional info: invalid DN
```
Fix: escape comma or special characters in the DN, quote in single quotes.

```
ldap_modify: Insufficient access (50)
```
Fix: bind as a DN that the ACL grants `write` for the targeted attrs/subtree, or escalate via `cn=admin`.

```
ldap_add: Already exists (68)
```
Fix: the DN exists. `ldapsearch -b '<dn>' -s base` first; use `ldapmodify` instead of `ldapadd`.

## Sample Workflows

### 1. Add a New User End-to-End

```bash
# Generate a salted-SHA password hash
slappasswd -h '{SSHA}' -s 'Welcome2026!'
# -> {SSHA}9e9c2c8f....

# Build LDIF
cat >/tmp/alice.ldif <<'EOF'
dn: uid=alice,ou=People,dc=example,dc=com
changetype: add
objectClass: top
objectClass: inetOrgPerson
objectClass: posixAccount
objectClass: shadowAccount
uid: alice
cn: Alice Smith
sn: Smith
givenName: Alice
mail: alice@example.com
uidNumber: 1001
gidNumber: 1001
homeDirectory: /home/alice
loginShell: /bin/bash
userPassword: {SSHA}9e9c2c8f....
EOF

# Apply
ldapadd -x -H ldap://localhost \
  -D 'cn=admin,dc=example,dc=com' -W \
  -f /tmp/alice.ldif

# Confirm
ldapsearch -x -LLL -b 'uid=alice,ou=People,dc=example,dc=com' -s base \
  '(objectClass=*)' dn cn mail uidNumber
```

### 2. Reset Password

```bash
# Admin-reset
ldappasswd -x -H ldaps://ldap.example.com \
  -D 'cn=admin,dc=example,dc=com' -W \
  -s 'Welcome2026!' \
  'uid=alice,ou=People,dc=example,dc=com'

# User self-change (ppolicy allows)
ldappasswd -x -D 'uid=alice,ou=People,dc=example,dc=com' \
  -w 'OldSecret' \
  -s 'BrandNew!2026'
```

### 3. Move a User Between OUs

```bash
ldapmodrdn -r -s 'ou=Alumni,dc=example,dc=com' \
  -x -H ldap://localhost \
  -D 'cn=admin,dc=example,dc=com' -W \
  'uid=alice,ou=People,dc=example,dc=com' 'uid=alice'
```

### 4. Find Inactive Users

```bash
# OpenLDAP with accesslog overlay logs all writes; pwdChangedTime works for ppolicy.
# Last bind requires a "lastbind" overlay (slapo-lastbind, OpenLDAP-contrib).
# AD: lastLogonTimestamp updates ~14 days; lastLogon is per-DC and must be aggregated.

# AD: stale users (lastLogonTimestamp older than 90 days)
# Compute FILETIME for cutoff:
#   epoch_seconds = $(date -d '90 days ago' +%s)
#   filetime = (epoch_seconds + 11644473600) * 10000000
ldapsearch -x -H ldap://dc01.example.com \
  -D 'EXAMPLE\admin' -W \
  -b 'DC=example,DC=com' \
  '(&(objectClass=user)(objectCategory=person)
    (lastLogonTimestamp<=133458912000000000))' \
  sAMAccountName lastLogonTimestamp
```

### 5. Sync POSIX Accounts via SSSD

```bash
realm join --user=admin EXAMPLE.COM    # joins AD with sssd
# or for bare LDAP, edit /etc/sssd/sssd.conf and:
systemctl enable --now sssd
authselect select sssd with-mkhomedir --force
getent passwd alice
id alice
sudo -u alice id
```

### 6. Disable AD Account

```bash
# Compute new UAC: existing | 0x2 (ACCOUNTDISABLE)
ldapmodify -x -H ldap://dc01.example.com \
  -D 'EXAMPLE\admin' -W <<'EOF'
dn: CN=Alice Smith,OU=Users,OU=Corp,DC=example,DC=com
changetype: modify
replace: userAccountControl
userAccountControl: 514
EOF
```

### 7. Find Empty Groups

```bash
ldapsearch -x -H ldap://localhost \
  -b 'ou=Groups,dc=example,dc=com' \
  '(&(objectClass=groupOfNames)(!(member=*)))' dn
```

### 8. List Groups a User Belongs To

```bash
# Forward (memberOf overlay or AD)
ldapsearch -x -b 'uid=alice,ou=People,dc=example,dc=com' \
  -s base '(objectClass=*)' memberOf

# Reverse lookup (works without memberOf overlay)
ldapsearch -x -b 'ou=Groups,dc=example,dc=com' \
  '(member=uid=alice,ou=People,dc=example,dc=com)' dn

# AD nested groups
ldapsearch -x -b 'DC=example,DC=com' \
  '(&(objectClass=group)(member:1.2.840.113556.1.4.1941:=
     CN=Alice Smith,OU=Users,DC=example,DC=com))' dn
```

### 9. Dump and Restore (OpenLDAP)

```bash
# Online via ldapsearch (respects ACLs; use a privileged DN)
ldapsearch -x -D 'cn=admin,dc=example,dc=com' -w "$BIND_PW" -LLL \
  -b 'dc=example,dc=com' -E pr=1000/noprompt \
  '(objectClass=*)' '*' '+' > backup.ldif

# Offline (fastest; ignores ACLs; requires slapd stopped)
systemctl stop slapd
slapcat -n 1 -l /var/backups/ldap-$(date +%F).ldif
systemctl start slapd

# Restore
systemctl stop slapd
rm -rf /var/lib/ldap/*
slapadd -n 1 -l /var/backups/ldap-2026-04-27.ldif
chown -R openldap:openldap /var/lib/ldap
systemctl start slapd
```

### 10. Bulk Add 10,000 Users

```bash
python3 - <<'PY' > /tmp/bulk.ldif
for i in range(10000):
    print(f"""dn: uid=user{i:05d},ou=People,dc=example,dc=com
objectClass: inetOrgPerson
objectClass: posixAccount
objectClass: shadowAccount
uid: user{i:05d}
cn: User {i:05d}
sn: User
givenName: {i:05d}
mail: user{i:05d}@example.com
uidNumber: {10000+i}
gidNumber: 10000
homeDirectory: /home/user{i:05d}
loginShell: /bin/bash
userPassword: {{SSHA}}placeholder
""")
PY

ldapadd -c -x -D 'cn=admin,dc=example,dc=com' -w "$ADMIN_PW" \
  -f /tmp/bulk.ldif
```

## SSSD Configuration

### sssd.conf (LDAP Provider)

```ini
[sssd]
config_file_version = 2
domains = example.com
services = nss, pam, ssh, sudo, autofs

[nss]
filter_groups = root
filter_users  = root
reconnection_retries = 3
homedir_substring = /home

[pam]
reconnection_retries = 3
offline_credentials_expiration = 7
offline_failed_login_attempts = 5
offline_failed_login_delay = 5

[domain/example.com]
id_provider = ldap
auth_provider = ldap
chpass_provider = ldap
sudo_provider = ldap
autofs_provider = ldap

ldap_uri = ldaps://ldap1.example.com,ldaps://ldap2.example.com
ldap_search_base = dc=example,dc=com
ldap_user_search_base = ou=People,dc=example,dc=com?subtree?
ldap_group_search_base = ou=Groups,dc=example,dc=com?subtree?
ldap_sudo_search_base = ou=Sudoers,dc=example,dc=com?subtree?

ldap_schema = rfc2307bis
ldap_user_object_class = posixAccount
ldap_user_principal = krbPrincipalName
ldap_group_object_class = groupOfNames
ldap_group_member = member

ldap_id_use_start_tls = false
ldap_tls_reqcert = demand
ldap_tls_cacert = /etc/pki/tls/certs/ca-bundle.crt

cache_credentials = true
entry_cache_timeout = 3600
enumerate = false

access_provider = ldap
ldap_access_filter = (&(objectClass=posixAccount)(memberOf=cn=linux-users,ou=Groups,dc=example,dc=com))
ldap_access_order = filter, expire
ldap_account_expire_policy = shadow
```

### sssd.conf (AD Provider)

```ini
[domain/example.com]
id_provider = ad
auth_provider = ad
access_provider = ad
chpass_provider = ad
sudo_provider = ad

ad_domain = example.com
ad_server = dc01.example.com,dc02.example.com
ad_gpo_access_control = enforcing

krb5_realm = EXAMPLE.COM
krb5_canonicalize = false
krb5_use_enterprise_principal = true

cache_credentials = true
ldap_id_mapping = true     # generate UIDs from SIDs
default_shell = /bin/bash
fallback_homedir = /home/%u@%d
use_fully_qualified_names = false
```

### Operational Commands

```bash
chmod 600 /etc/sssd/sssd.conf
chown root:root /etc/sssd/sssd.conf

systemctl enable --now sssd
authselect select sssd with-mkhomedir --force
sss_cache -E                          # invalidate cache
sssctl domain-list                    # show configured domains
sssctl domain-status example.com      # show online/offline state
sssctl user-checks alice              # diagnose lookup
getent passwd alice                   # NSS resolution
id alice
ssh alice@host                        # PAM resolution
```

```text
# sssctl user-checks output
user: alice
action: acct
service: system-auth

SSSD nss user lookup result:
 - user name: alice
 - user id: 1001
 - group id: 1001
 - gecos: Alice Smith
 - home directory: /home/alice
 - shell: /bin/bash
```

## Replication

### OpenLDAP syncrepl (RFC 4533)

```
                +--------+ syncrepl pull/push +--------+
                |  P1    | <----------------> |  P2    |
                | mdb    |                    | mdb    |
                +--------+                    +--------+
                    |  ^                          |  ^
                    v  |                          v  |
                +--------+ syncrepl pull/push +--------+
                |  P3    | <----------------> |  P4    |
                +--------+                    +--------+

   refreshAndPersist: long-lived TCP, server pushes csn changes
   refreshOnly: poll on interval
```

```ldif
# Provider (P1) — load syncprov overlay
dn: cn=module{0},cn=config
changetype: modify
add: olcModuleLoad
olcModuleLoad: syncprov.la

dn: olcOverlay=syncprov,olcDatabase={1}mdb,cn=config
changetype: add
objectClass: olcOverlayConfig
objectClass: olcSyncProvConfig
olcOverlay: syncprov
olcSpCheckpoint: 100 10
olcSpSessionLog: 1000

# Consumer (P2)
dn: olcDatabase={1}mdb,cn=config
changetype: modify
add: olcSyncRepl
olcSyncRepl: rid=001
  provider=ldaps://p1.example.com
  bindmethod=simple
  binddn="cn=replicator,dc=example,dc=com"
  credentials=ReplSecret
  searchbase="dc=example,dc=com"
  schemachecking=on
  type=refreshAndPersist
  retry="60 +"
  starttls=critical
  tls_reqcert=demand
-
add: olcMirrorMode
olcMirrorMode: TRUE
```

### 389-DS Replication

```
Multi-Supplier (MMR) topology:

   ds1 <----> ds2
    ^          ^
    |          |
    v          v
   ds3 <----> ds4

   Each supplier has a unique replicaID (1..65534).
   Conflicts resolved by CSN (Change Sequence Number).
```

```bash
# On supplier 1
dsconf slapd-ds1 replication enable \
  --suffix=dc=example,dc=com --role=supplier --replica-id=1

dsconf slapd-ds1 replication create-manager \
  --name=replmgr --passwd=ReplSecret

dsconf slapd-ds1 repl-agmt create \
  --suffix=dc=example,dc=com \
  --host=ds2.example.com --port=636 --conn-protocol=LDAPS \
  --bind-dn='cn=replmgr,cn=config' --bind-passwd='ReplSecret' \
  --bind-method=SIMPLE --init agreement-ds2

dsconf slapd-ds1 repl-agmt status \
  --suffix=dc=example,dc=com agreement-ds2

dsconf slapd-ds1 repl-agmt init \
  --suffix=dc=example,dc=com agreement-ds2
```

### AD Replication (USN-Based)

```
Up-to-Dateness Vector (UTDVector):
- Each DC tracks per-DC USN
- Replicate-Now via repadmin /syncall

Tooling:
  repadmin /showrepl                 # current replication state
  repadmin /replsummary              # summary
  repadmin /syncall /APed            # force pull push
  repadmin /removelingeringobjects   # tombstones
  dcdiag /v /e                       # comprehensive health
  ntdsutil "metadata cleanup"        # remove dead DC
```

```text
# repadmin /showrepl excerpt
Default-First-Site-Name\DC01
DSA Options: IS_GC
Site Options: (none)
DSA object GUID: 1f1b9df7-...
DSA invocationID: c4f6b1a4-...

==== INBOUND NEIGHBORS ======================================
DC=example,DC=com
   Default-First-Site-Name\DC02 via RPC
       DSA object GUID: 4e9b...
       Last attempt @ 2026-04-27 09:14:00 was successful.
```

## Performance Tuning

### Indexing

```ldif
# OpenLDAP: index every attribute used in filters
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcDbIndex
olcDbIndex: objectClass eq
olcDbIndex: cn eq,sub
olcDbIndex: sn eq,sub
olcDbIndex: uid eq
olcDbIndex: uidNumber eq
olcDbIndex: gidNumber eq
olcDbIndex: mail eq,sub
olcDbIndex: memberOf eq
olcDbIndex: member eq
olcDbIndex: entryCSN eq
olcDbIndex: entryUUID eq
```

```bash
# Force reindex (offline)
systemctl stop slapd
slapindex -n 1
chown -R openldap:openldap /var/lib/ldap
systemctl start slapd

# Detect unindexed searches
ldapmodify -Y EXTERNAL -H ldapi:/// <<'EOF'
dn: cn=config
changetype: modify
replace: olcLogLevel
olcLogLevel: stats stats2
EOF

journalctl -u slapd | grep -i 'index_param failed'
journalctl -u slapd | grep -i 'not indexed'
```

### LMDB Sizing

```bash
# olcDbMaxSize is per-database mmap budget
# Default 10 MiB is too small for production
ldapmodify -Y EXTERNAL -H ldapi:/// <<'EOF'
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcDbMaxSize
olcDbMaxSize: 8589934592    # 8 GiB
EOF
```

```
Rule of thumb:
  ~1 KB per entry  + indexes   ~ 250 B/entry/index
  Plan for 4x your live data size to allow growth
  LMDB is mmap'd; size it large; OS only commits what's used
```

### Connection Pooling (ldap.conf)

```
# /etc/openldap/ldap.conf
URI ldaps://ldap.example.com
BASE dc=example,dc=com
TLS_CACERT /etc/pki/tls/certs/ca-bundle.crt
TLS_REQCERT demand
TIMELIMIT 10
SIZELIMIT 1000
NETWORK_TIMEOUT 5
DEREF never
```

### Paged Results

```bash
ldapsearch -x -E pr=500/noprompt \
  -H ldap://localhost -b 'dc=example,dc=com' \
  '(objectClass=person)'
```

### Server-Side Sorting (RFC 2891)

```bash
ldapsearch -x -E sss=cn -b 'ou=People,dc=example,dc=com' \
  '(objectClass=inetOrgPerson)' cn
```

### Virtual List View (RFC 2891)

```bash
ldapsearch -x -E sss=cn -E vlv=0/9/1/0 \
  -b 'ou=People,dc=example,dc=com' '(objectClass=*)' cn
# (offset 1, before-count 0, after-count 9 — first 10 entries)
```

## Security Best Practices

### TLS Hardening

```ldif
dn: cn=config
changetype: modify
replace: olcTLSCertificateFile
olcTLSCertificateFile: /etc/openldap/certs/ldap.crt
-
replace: olcTLSCertificateKeyFile
olcTLSCertificateKeyFile: /etc/openldap/certs/ldap.key
-
replace: olcTLSCACertificateFile
olcTLSCACertificateFile: /etc/openldap/certs/ca.crt
-
replace: olcTLSCipherSuite
olcTLSCipherSuite: HIGH:!aNULL:!MD5:!RC4:!3DES:!CAMELLIA:!SHA1
-
replace: olcTLSProtocolMin
olcTLSProtocolMin: 3.3            # TLS 1.2 minimum
-
replace: olcTLSVerifyClient
olcTLSVerifyClient: try            # request optional client cert
```

### Disable Anonymous Bind

```ldif
dn: cn=config
changetype: modify
add: olcDisallows
olcDisallows: bind_anon
-
add: olcRequires
olcRequires: authc
```

### Password Policy (ppolicy)

```ldif
dn: cn=module{0},cn=config
changetype: modify
add: olcModuleLoad
olcModuleLoad: ppolicy.la

dn: olcOverlay=ppolicy,olcDatabase={1}mdb,cn=config
changetype: add
objectClass: olcOverlayConfig
objectClass: olcPPolicyConfig
olcOverlay: ppolicy
olcPPolicyDefault: cn=default,ou=PolicyConfig,dc=example,dc=com
olcPPolicyHashCleartext: TRUE
olcPPolicyUseLockout: TRUE

dn: cn=default,ou=PolicyConfig,dc=example,dc=com
objectClass: top
objectClass: device
objectClass: pwdPolicy
cn: default
pwdAttribute: userPassword
pwdMinLength: 12
pwdMinAge: 0
pwdMaxAge: 7776000           # 90 days
pwdInHistory: 5
pwdCheckQuality: 2           # require quality check
pwdMaxFailure: 5
pwdLockout: TRUE
pwdLockoutDuration: 1800     # 30 min
pwdFailureCountInterval: 600
pwdMustChange: TRUE
pwdAllowUserChange: TRUE
pwdSafeModify: FALSE
pwdExpireWarning: 604800     # 7 days
pwdGraceAuthnLimit: 3
```

### Audit Logging (auditlog overlay)

```ldif
dn: olcOverlay=auditlog,olcDatabase={1}mdb,cn=config
changetype: add
objectClass: olcOverlayConfig
objectClass: olcAuditLogConfig
olcOverlay: auditlog
olcAuditlogFile: /var/log/slapd-audit.log
```

```text
# Sample audit log line
# add 1714207020 cn=alice,...
dn: uid=alice,ou=People,dc=example,dc=com
changetype: add
...
# end
```

### Forward to syslog

```bash
# Edit /etc/sysconfig/slapd or systemd unit
# add SLAPD_OPTIONS="-d 256"
# In /etc/rsyslog.conf
local4.* /var/log/slapd.log
```

## ldap.conf Client Configuration

```
# /etc/openldap/ldap.conf
URI                ldaps://ldap1.example.com ldaps://ldap2.example.com
BASE               dc=example,dc=com
BINDDN             uid=svc-app,ou=Services,dc=example,dc=com
TLS_CACERT         /etc/pki/tls/certs/ca-bundle.crt
TLS_REQCERT        demand
TLS_CRLCHECK       none
TLS_PROTOCOL_MIN   3.3
TLS_CIPHER_SUITE   HIGH:!aNULL:!MD5:!3DES
TIMELIMIT          30
SIZELIMIT          1000
NETWORK_TIMEOUT    10
DEREF              never
SASL_MECH          GSSAPI
SASL_REALM         EXAMPLE.COM
```

### Per-User Override

```bash
# ~/.ldaprc has higher precedence than /etc/openldap/ldap.conf
cat >~/.ldaprc <<'EOF'
URI ldaps://lab.example.com
BASE dc=lab,dc=example,dc=com
EOF
```

## Common Modern Use-Cases

### POSIX Login via SSSD

```
NSS lookup ────► sssd ────► LDAP server
PAM auth   ────► sssd ────► LDAP bind
SSH AuthorizedKeysCommand ─► sssctl ssh-pubkeys ──► sshPublicKey attr
sudo -u   ────► sssd sudo provider ────► sudoRole entries
```

### Nginx LDAP Auth (ngx_http_auth_ldap_module)

```nginx
ldap_server example {
    url "ldaps://ldap.example.com:636/ou=People,dc=example,dc=com?uid?sub?(memberOf=cn=webusers,ou=Groups,dc=example,dc=com)";
    binddn "cn=svc-nginx,ou=Services,dc=example,dc=com";
    binddn_passwd "secret";
    group_attribute uniqueMember;
    group_attribute_is_dn on;
    require valid_user;
}

server {
    listen 443 ssl;
    location / {
        auth_ldap "Restricted";
        auth_ldap_servers example;
        proxy_pass http://backend;
    }
}
```

### Apache LDAP Auth (mod_authnz_ldap)

```apache
<Directory "/var/www/private">
    AuthType Basic
    AuthName "LDAP"
    AuthBasicProvider ldap
    AuthLDAPURL "ldaps://ldap.example.com/ou=People,dc=example,dc=com?uid?sub?(objectClass=inetOrgPerson)"
    AuthLDAPBindDN "cn=svc-apache,ou=Services,dc=example,dc=com"
    AuthLDAPBindPassword "secret"
    Require ldap-group cn=webusers,ou=Groups,dc=example,dc=com
</Directory>
```

### OpenSSH AuthorizedKeysCommand

```bash
# /etc/ssh/sshd_config
AuthorizedKeysCommand /usr/bin/sss_ssh_authorizedkeys
AuthorizedKeysCommandUser nobody
```

```bash
# Add openssh-lpk schema
ldapmodify -x -D 'cn=admin,...' -W <<'EOF'
dn: uid=alice,ou=People,dc=example,dc=com
changetype: modify
add: objectClass
objectClass: ldapPublicKey
-
add: sshPublicKey
sshPublicKey: ssh-ed25519 AAAAC3Nz... alice@laptop
EOF
```

### LDAP-Backed Postfix

```
# main.cf
virtual_alias_maps = ldap:/etc/postfix/ldap-aliases.cf

# /etc/postfix/ldap-aliases.cf
server_host = ldaps://ldap.example.com
search_base = ou=People,dc=example,dc=com
query_filter = (mail=%s)
result_attribute = uid
bind = yes
bind_dn = cn=svc-postfix,ou=Services,dc=example,dc=com
bind_pw = secret
```

## Migration Patterns

### LDAP → Keycloak (SAML/OIDC IdP)

```
Phase 1: Keycloak federates LDAP user store (LDAPFederationProvider)
Phase 2: Keycloak issues SAML/OIDC; apps move from direct LDAP to Keycloak
Phase 3: User self-service (password, MFA) moves to Keycloak
Phase 4: LDAP becomes write-back target only or is decommissioned
```

### LDAP → SCIM 2.0

```
SCIM endpoints:
  GET    /scim/v2/Users
  POST   /scim/v2/Users
  PATCH  /scim/v2/Users/<id>
  GET    /scim/v2/Groups
  ...

Common bridges:
  - Keycloak SCIM extension
  - midPoint (Evolveum)
  - SailPoint, Okta, OneLogin
```

### LDAP → SAML/OIDC

LDAP is for protocol/data; SAML/OIDC is for federation. Typical pattern: keep LDAP as the system-of-record, layer Keycloak/Okta/Azure AD on top to issue SAML/OIDC tokens to apps.

## ASCII Diagrams

### Bind Flow (Simple)

```
Client                                   Server
  | TCP SYN -----------------------------> |
  |<-------------- SYN/ACK ----------------|
  | ACK ----------------------------------> |
  | LDAP BindRequest(simple, DN, pwd) ----> |
  |<------- BindResponse(resultCode=0) ----|
  | SearchRequest -----------------------> |
  |<-------- SearchResultEntry × N --------|
  |<-------- SearchResultDone -------------|
  | UnbindRequest ----------------------->  |
  | TCP FIN ----------------------------->  |
```

### SASL GSSAPI Bind

```
Client                                   KDC                Server
  | AS-REQ ----------------------------->|
  |<------------ AS-REP (TGT) -----------|
  | TGS-REQ (ldap/srv@REALM) ----------->|
  |<--------- TGS-REP (service ticket) --|
  | LDAP Bind: SASL GSSAPI, Negotiate -------------------> |
  |<-------- Bind Response: SASL challenge ---------------|
  | LDAP Bind: SASL response (AP-REQ) -------------------> |
  |<-------- Bind Response: success ----------------------|
```

### Replication Topology

```
              ┌─────────────────┐
              │   Provider P1   │ rid=001
              └────┬───────┬────┘
                   │       │
       refreshAndPersist  │
                   │       │
              ┌────▼────┐  │
              │   P2    │  │ rid=002
              └────┬────┘  │
                   │       │
              ┌────▼────┐ ┌▼─────────┐
              │   P3    │ │   P4     │ rid=003,004
              └─────────┘ └──────────┘

  Mirror mode (P1 ↔ P2):  both writable, syncrepl in both directions
  Push (P1 → P3, P4):     P1 is single writer; consumers read-only
```

### Schema Hierarchy ASCII

```
                       top
                        │
           ┌────────────┼─────────────┐
           │            │             │
        person     organizationalUnit  dcObject
           │
   organizationalPerson
           │
     inetOrgPerson  ────────────┐  (auxiliary auxiliaries layered on top:)
                                │      posixAccount
                                │      shadowAccount
                                │      ldapPublicKey
                                │      ppolicyAccount
                                │
                              user (AD-only structural class)
```

## Tips

- Always use LDAPS or StartTLS in production. Cleartext passwords on the wire are exfiltrated by any host on the path.
- Index every attribute used in search filters. Unindexed lookups become full DB scans on large directories. Watch `journalctl -u slapd | grep 'not indexed'`.
- Use `cn=config` (OLC) over `slapd.conf`. OLC changes apply immediately and survive upgrades cleanly.
- `enumerate = false` in `sssd.conf` for directories larger than a few thousand entries. Enumeration loads everything into cache.
- Use `{SSHA}`, `{ARGON2}`, or `{PBKDF2-SHA512}` password hashes. Never `{CLEAR}`, `{CRYPT}` with DES, or `{MD5}`.
- Validate connectivity with `ldapwhoami -x -D '<dn>' -W -H ldaps://<host>` before debugging searches.
- Back up the DIT regularly with `slapcat` (OpenLDAP), `dsctl <inst> backup create` (389-DS), or `ntdsutil "ifm"` (AD).
- Use the `memberOf` overlay or 389-DS plugin to enable reverse group lookups without expensive `(member=...)` filters.
- Set `olcSizeLimit` and `olcTimeLimit` per database to prevent runaway queries from exhausting threads.
- Pool LDAP connections in apps. Each bind is a TCP+TLS handshake — at scale this dominates latency.
- For bulk loads, stop slapd and use `slapadd` (5–10× faster than `ldapadd`).
- Test ACLs with `slapacl -F /etc/openldap/slapd.d -D '<binddn>' -b '<targetdn>' '<attr>/<priv>'` before deploying.
- AD's `1.2.840.113556.1.4.1941` chain rule is O(n) over nested groups; avoid in hot paths.
- Keep `entryCSN` and `entryUUID` indexed (`eq`) — required for syncrepl performance.
- Use `-LLL` for scriptable output. Comments and version lines break naive parsers.
- Quote DNs in single quotes in shell. Spaces, commas, and backslashes are common.
- Use `-c` (`continue on error`) when applying large LDIFs so one bad entry does not abort the batch.
- Regenerate `slapd-config` certs every year and reload via `cn=config` modify (no restart needed).
- For AD lookups from Linux, prefer GSSAPI bind over simple bind. It avoids storing service-account passwords on disk and respects Kerberos delegation.
- When debugging schema violations, get the offending entry, then check each `objectClass`'s MUST/MAY against `attributeTypes` published in `cn=schema,cn=config`.

## See Also

- `security/oauth`
- `auth/oidc`
- `auth/saml`
- `auth/kerberos`
- `auth/sssd`
- `security/pam`
- `security/pki`
- `security/tls`
- `security/openssl`
- `network-os/cisco-ios`
- `ramp-up/tls-eli5`

## References

- RFC 4510 — LDAP: Technical Specification Road Map
- RFC 4511 — LDAP: The Protocol
- RFC 4512 — LDAP: Directory Information Models
- RFC 4513 — LDAP: Authentication Methods and Security Mechanisms
- RFC 4514 — LDAP: String Representation of Distinguished Names
- RFC 4515 — LDAP: String Representation of Search Filters
- RFC 4516 — LDAP: Uniform Resource Locator
- RFC 4517 — LDAP: Syntaxes and Matching Rules
- RFC 4518 — LDAP: Internationalized String Preparation
- RFC 4519 — LDAP: Schema for User Applications
- RFC 4520 — LDAP: IANA Considerations
- RFC 4522 — LDAP: Binary Encoding Option
- RFC 4523 — LDAP: X.509 Certificate Schema
- RFC 4524 — COSINE LDAP/X.500 Schema
- RFC 4525 — LDAP Modify-Increment Extension
- RFC 4526 — LDAP Absolute True and False Filters
- RFC 4527 — LDAP Read Entry Controls
- RFC 4528 — LDAP Assertion Control
- RFC 4529 — LDAP Requesting Attributes by Object Class
- RFC 4530 — LDAP entryUUID Attribute
- RFC 4532 — LDAP "Who Am I?" Extended Operation
- RFC 4533 — LDAP Content Synchronization (syncrepl)
- RFC 2696 — LDAP Simple Paged Results Control
- RFC 2891 — LDAP Server-Side Sorting
- RFC 3062 — LDAP Password Modify Extended Operation
- RFC 2798 — Definition of inetOrgPerson
- RFC 2307 — Approach for Using LDAP as a NIS
- RFC 4422 — Simple Authentication and Security Layer (SASL)
- RFC 4505 — Anonymous SASL Mechanism
- RFC 4616 — PLAIN SASL Mechanism
- RFC 5802 — SCRAM-SHA-1 SASL Mechanism
- RFC 7677 — SCRAM-SHA-256 SASL Mechanism
- RFC 4178 — GSS-API SPNEGO
- RFC 4559 — SPNEGO over HTTP (related)
- [OpenLDAP Admin Guide](https://www.openldap.org/doc/admin26/)
- [389 Directory Server Documentation](https://www.port389.org/docs/389ds/documentation.html)
- [SSSD LDAP Provider](https://sssd.io/docs/users/ldap_provider.html)
- [Microsoft AD Schema Reference](https://learn.microsoft.com/en-us/windows/win32/adschema/active-directory-schema)
- [Apache Directory Studio](https://directory.apache.org/studio/)
- man slapd, slapd.conf, slapd-mdb, slapd-config
- man ldap.conf, ldapsearch, ldapadd, ldapmodify, ldapdelete, ldapmodrdn, ldappasswd, ldapwhoami, ldapcompare, ldapurl
- man slapcat, slapadd, slapindex, slapdn, slapauth, slapacl, slappasswd
- man sssd.conf, sssd-ldap, sssd-ad, sssd-ipa
- "Understanding and Deploying LDAP Directory Services" — Howes, Smith, Good (2nd ed., Addison-Wesley)
- "Mastering OpenLDAP" — Matt Butcher (Packt)
- "LDAP System Administration" — Gerald Carter (O'Reilly)
