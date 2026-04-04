# LDAP (Lightweight Directory Access Protocol)

Query, authenticate, and manage directory entries using LDAP's hierarchical Directory Information Tree, LDIF import/export format, and search filters across OpenLDAP, 389 Directory Server, and Active Directory backends with sssd integration for Linux host authentication.

## Directory Information Tree (DIT)

### DN and RDN Components

```bash
# Distinguished Name (DN) — full path to an entry
dn: uid=jdoe,ou=People,dc=example,dc=com

# Relative Distinguished Name (RDN) — leftmost component
# RDN: uid=jdoe

# Common DN attributes:
# dc  = domain component      (dc=example,dc=com)
# ou  = organizational unit    (ou=People, ou=Groups)
# cn  = common name           (cn=John Doe)
# uid = user ID               (uid=jdoe)
```

### Base DIT Structure

```ldif
# Root entry
dn: dc=example,dc=com
objectClass: top
objectClass: domain
dc: example

# Organizational units
dn: ou=People,dc=example,dc=com
objectClass: organizationalUnit
ou: People

dn: ou=Groups,dc=example,dc=com
objectClass: organizationalUnit
ou: Groups

dn: ou=Services,dc=example,dc=com
objectClass: organizationalUnit
ou: Services
```

## LDIF Format

### Adding Entries

```ldif
# user.ldif — add a user entry
dn: uid=jdoe,ou=People,dc=example,dc=com
changetype: add
objectClass: inetOrgPerson
objectClass: posixAccount
objectClass: shadowAccount
uid: jdoe
cn: John Doe
sn: Doe
givenName: John
mail: jdoe@example.com
uidNumber: 10001
gidNumber: 10001
homeDirectory: /home/jdoe
loginShell: /bin/bash
userPassword: {SSHA}hashed_password_here
```

### Modifying Entries

```ldif
# modify.ldif — change attributes
dn: uid=jdoe,ou=People,dc=example,dc=com
changetype: modify
replace: mail
mail: john.doe@example.com
-
add: telephoneNumber
telephoneNumber: +1-555-0199
-
delete: description
```

### Importing LDIF

```bash
# Add entries from LDIF file
ldapadd -x -D "cn=admin,dc=example,dc=com" -W -f user.ldif

# Apply modifications
ldapmodify -x -D "cn=admin,dc=example,dc=com" -W -f modify.ldif

# Delete an entry
ldapdelete -x -D "cn=admin,dc=example,dc=com" -W \
  "uid=jdoe,ou=People,dc=example,dc=com"
```

## Search Filters

### Basic Searches

```bash
# Search for a user by uid
ldapsearch -x -b "dc=example,dc=com" "(uid=jdoe)"

# Search with specific attributes returned
ldapsearch -x -b "dc=example,dc=com" "(uid=jdoe)" cn mail uidNumber

# Search with bind credentials
ldapsearch -x -D "cn=admin,dc=example,dc=com" -W \
  -b "dc=example,dc=com" "(objectClass=posixAccount)"

# Limit results
ldapsearch -x -b "dc=example,dc=com" -z 10 "(objectClass=person)"
```

### Filter Syntax (AND / OR / NOT)

```bash
# AND — all conditions must match
ldapsearch -x -b "ou=People,dc=example,dc=com" \
  "(&(objectClass=posixAccount)(uid=jdoe))"

# OR — any condition matches
ldapsearch -x -b "ou=People,dc=example,dc=com" \
  "(|(uid=jdoe)(uid=asmith))"

# NOT — exclude matches
ldapsearch -x -b "ou=People,dc=example,dc=com" \
  "(!(loginShell=/sbin/nologin))"

# Combined — active users in engineering
ldapsearch -x -b "dc=example,dc=com" \
  "(&(objectClass=posixAccount)(ou=Engineering)(!(loginShell=/sbin/nologin)))"

# Wildcard — all users whose cn starts with "John"
ldapsearch -x -b "dc=example,dc=com" "(cn=John*)"

# Presence — entries that have a mail attribute
ldapsearch -x -b "dc=example,dc=com" "(mail=*)"
```

### Search Scopes

```bash
# base — only the entry itself
ldapsearch -x -b "uid=jdoe,ou=People,dc=example,dc=com" -s base "(objectClass=*)"

# one — immediate children only
ldapsearch -x -b "ou=People,dc=example,dc=com" -s one "(objectClass=posixAccount)"

# sub — full subtree (default)
ldapsearch -x -b "dc=example,dc=com" -s sub "(uid=jdoe)"
```

## Schema and objectClass

### Core objectClasses

```bash
# List schema objectClasses
ldapsearch -x -b "cn=schema,cn=config" -s base "(objectClass=*)" objectClasses

# Common objectClasses:
# inetOrgPerson — email, phone, org attributes
# posixAccount  — uidNumber, gidNumber, homeDirectory (RFC 2307)
# shadowAccount — password expiry (shadow suite)
# groupOfNames  — groups with member DN list
# posixGroup    — groups with memberUid list (RFC 2307)
# organizationalUnit — ou containers
```

## OpenLDAP slapd Configuration

### cn=config (OLC) Runtime Config

```bash
# Check slapd status
systemctl status slapd

# View current config
ldapsearch -Y EXTERNAL -H ldapi:/// -b "cn=config" "(objectClass=*)" dn

# Set log level
ldapmodify -Y EXTERNAL -H ldapi:/// <<EOF
dn: cn=config
changetype: modify
replace: olcLogLevel
olcLogLevel: stats
EOF

# Add an index for faster searches
ldapmodify -Y EXTERNAL -H ldapi:/// <<EOF
dn: olcDatabase={1}mdb,cn=config
changetype: modify
add: olcDbIndex
olcDbIndex: mail eq,sub
EOF

# Set size limit
ldapmodify -Y EXTERNAL -H ldapi:/// <<EOF
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcSizeLimit
olcSizeLimit: 5000
EOF
```

### Access Control

```bash
# View ACLs
ldapsearch -Y EXTERNAL -H ldapi:/// \
  -b "olcDatabase={1}mdb,cn=config" olcAccess

# Typical ACL set
ldapmodify -Y EXTERNAL -H ldapi:/// <<EOF
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcAccess
olcAccess: {0}to attrs=userPassword
  by self write
  by anonymous auth
  by * none
olcAccess: {1}to *
  by self read
  by users read
  by * none
EOF
```

## 389 Directory Server

### Basic Management

```bash
# Install 389DS
dnf install 389-ds-base

# Create instance
dscreate from-file instance.inf

# Instance control
dsctl slapd-localhost start
dsctl slapd-localhost stop
dsctl slapd-localhost status

# Interactive config
dsconf slapd-localhost backend list
dsconf slapd-localhost backend suffix list
dsconf slapd-localhost plugin list
```

## SSSD Integration

### sssd.conf for LDAP

```ini
# /etc/sssd/sssd.conf
[sssd]
domains = example.com
services = nss, pam, ssh

[domain/example.com]
id_provider = ldap
auth_provider = ldap
ldap_uri = ldaps://ldap.example.com
ldap_search_base = dc=example,dc=com
ldap_tls_reqcert = demand
ldap_tls_cacert = /etc/pki/tls/certs/ca-bundle.crt
ldap_id_use_start_tls = true

# User/group mapping
ldap_user_search_base = ou=People,dc=example,dc=com
ldap_group_search_base = ou=Groups,dc=example,dc=com
ldap_user_object_class = posixAccount
ldap_group_object_class = posixGroup

# Caching
cache_credentials = true
entry_cache_timeout = 600

# Enumeration (disable for large directories)
enumerate = false
```

### Enable SSSD

```bash
# Set permissions
chmod 600 /etc/sssd/sssd.conf

# Enable and start
systemctl enable --now sssd

# Configure NSS and PAM
authselect select sssd with-mkhomedir --force

# Test resolution
getent passwd jdoe
id jdoe
```

## Tips

- Always use LDAPS (port 636) or StartTLS (port 389 upgraded) in production -- never plain LDAP, as bind passwords are sent in cleartext
- Set `olcDbIndex` on attributes used in search filters (uid, mail, cn, memberOf) -- unindexed searches do full table scans on large directories
- Use `ldapsearch -LLL` to suppress comments and version lines for cleaner output suitable for scripting
- Prefer `cn=config` (OLC) over `slapd.conf` for OpenLDAP -- OLC changes take effect immediately without restarting slapd
- Set `enumerate = false` in sssd.conf for directories with more than a few thousand entries to avoid loading the entire directory into cache
- Use `{SSHA}` (salted SHA) or `{PBKDF2-SHA256}` for password hashing -- never store passwords in `{CLEAR}` or `{SHA}`
- Test LDAP connectivity with `ldapwhoami -x -D "cn=admin,dc=example,dc=com" -W -H ldaps://ldap.example.com` before debugging complex queries
- Back up the DIT regularly with `slapcat` (OpenLDAP) or `dsconf slapd-localhost backup create` (389DS)
- Use `memberOf` overlay in OpenLDAP to enable reverse group membership lookups without expensive search filters
- Set `olcSizeLimit` and `olcTimeLimit` to prevent runaway queries from exhausting server resources
- Use LDAP connection pooling in application code -- each LDAP bind involves a TCP handshake and potentially a TLS negotiation

## See Also

- kerberos, saml, sssd, pam, openssl, freeipa, active-directory

## References

- [OpenLDAP Admin Guide](https://www.openldap.org/doc/admin26/)
- [RFC 4511 — LDAPv3 Protocol](https://datatracker.ietf.org/doc/html/rfc4511)
- [RFC 4515 — LDAP Search Filters](https://datatracker.ietf.org/doc/html/rfc4515)
- [RFC 2307 — LDAP as Network Information Service](https://datatracker.ietf.org/doc/html/rfc2307)
- [389 Directory Server Documentation](https://www.port389.org/docs/389ds/documentation.html)
- [SSSD LDAP Provider](https://sssd.io/docs/users/ldap_provider.html)
