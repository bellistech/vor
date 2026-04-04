# Dovecot (IMAP/POP3 Server)

Secure, high-performance IMAP and POP3 server with Sieve filtering, quotas, and Postfix integration.

## Core Configuration

### Protocols and Listeners

```bash
# /etc/dovecot/dovecot.conf
# protocols = imap pop3 lmtp sieve

# /etc/dovecot/conf.d/10-master.conf
# service imap-login {
#   inet_listener imap {
#     port = 143
#   }
#   inet_listener imaps {
#     port = 993
#     ssl = yes
#   }
# }
# service pop3-login {
#   inet_listener pop3 {
#     port = 110
#   }
#   inet_listener pop3s {
#     port = 995
#     ssl = yes
#   }
# }
# service lmtp {
#   unix_listener /var/spool/postfix/private/dovecot-lmtp {
#     mode = 0600
#     user = postfix
#     group = postfix
#   }
# }
```

### Mailbox Formats

```bash
# Maildir (one file per message, recommended)
mail_location = maildir:~/Maildir

# mbox (single file per folder)
mail_location = mbox:~/mail:INBOX=/var/mail/%u

# dbox - Dovecot's high-performance native format
# sdbox: single-dbox (one file per message, like Maildir)
mail_location = sdbox:~/dbox

# mdbox: multi-dbox (multiple messages per file)
mail_location = mdbox:~/mdbox

# Convert between formats
dsync mirror maildir:~/Maildir mdbox:~/mdbox
```

## Authentication

### PAM Authentication

```bash
# /etc/dovecot/conf.d/10-auth.conf
# auth_mechanisms = plain login

# /etc/dovecot/conf.d/auth-system.conf.ext
# passdb {
#   driver = pam
# }
# userdb {
#   driver = passwd
# }

# Test authentication
doveadm auth test username password
doveadm auth login username password
```

### LDAP Authentication

```bash
# /etc/dovecot/conf.d/auth-ldap.conf.ext
# passdb {
#   driver = ldap
#   args = /etc/dovecot/dovecot-ldap.conf.ext
# }
# userdb {
#   driver = ldap
#   args = /etc/dovecot/dovecot-ldap.conf.ext
# }

# /etc/dovecot/dovecot-ldap.conf.ext
# hosts = ldap.example.com
# dn = cn=dovecot,ou=services,dc=example,dc=com
# dnpass = secret
# base = ou=users,dc=example,dc=com
# user_filter = (&(objectClass=posixAccount)(uid=%u))
# pass_filter = (&(objectClass=posixAccount)(uid=%u))
# pass_attrs = uid=user,userPassword=password
# user_attrs = homeDirectory=home,uidNumber=uid,gidNumber=gid
```

### SQL Authentication

```bash
# /etc/dovecot/conf.d/auth-sql.conf.ext
# passdb {
#   driver = sql
#   args = /etc/dovecot/dovecot-sql.conf.ext
# }
# userdb {
#   driver = sql
#   args = /etc/dovecot/dovecot-sql.conf.ext
# }

# /etc/dovecot/dovecot-sql.conf.ext (MySQL example)
# driver = mysql
# connect = host=localhost dbname=maildb user=dovecot password=secret
# default_pass_scheme = SHA512-CRYPT
# password_query = SELECT email as user, password FROM users WHERE email='%u'
# user_query = SELECT home, uid, gid FROM users WHERE email='%u'

# Password file (simple flat file auth)
# passdb {
#   driver = passwd-file
#   args = scheme=SHA512-CRYPT /etc/dovecot/users
# }
# Format: user@domain:{SHA512-CRYPT}$6$salt$hash::::::
```

## SSL/TLS Configuration

```bash
# /etc/dovecot/conf.d/10-ssl.conf
# ssl = required
# ssl_cert = </etc/letsencrypt/live/mail.example.com/fullchain.pem
# ssl_key = </etc/letsencrypt/live/mail.example.com/privkey.pem
# ssl_min_protocol = TLSv1.2
# ssl_cipher_list = ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384
# ssl_prefer_server_ciphers = yes

# Generate DH parameters
openssl dhparam -out /etc/dovecot/dh.pem 4096
# ssl_dh = </etc/dovecot/dh.pem

# Test TLS
openssl s_client -connect mail.example.com:993
openssl s_client -starttls imap -connect mail.example.com:143
```

## Sieve Filtering

```bash
# /etc/dovecot/conf.d/90-sieve.conf
# plugin {
#   sieve = ~/.dovecot.sieve
#   sieve_dir = ~/sieve
#   sieve_global_dir = /etc/dovecot/sieve
#   sieve_before = /etc/dovecot/sieve/before.d/
#   sieve_after = /etc/dovecot/sieve/after.d/
#   sieve_extensions = +vnd.dovecot.pipe +vnd.dovecot.execute
# }

# Example sieve rules: spam, list sorting, vacation
# require ["fileinto", "reject", "vacation", "imap4flags"];
# if header :contains "X-Spam-Flag" "YES" { fileinto "Junk"; stop; }
# if header :is "List-Id" "dev.lists.example.com" { fileinto "Lists.dev"; stop; }
# vacation :days 7 :subject "Out of Office" "I am currently out of the office.";

# Compile and test sieve scripts
sievec ~/.dovecot.sieve
sieve-test -t - ~/.dovecot.sieve test-message.eml
```

## Quotas

```bash
# /etc/dovecot/conf.d/90-quota.conf
# plugin {
#   quota = maildir:User quota    # or dict:User quota::file:%h/quota
#   quota_rule = *:storage=1G
#   quota_rule2 = Trash:storage=+100M
#   quota_rule3 = INBOX:storage=+200M
#   quota_grace = 10%%
#   quota_status_success = DUNNO
#   quota_status_nouser = DUNNO
#   quota_status_overquota = "552 5.2.2 Mailbox is full"
# }

# Check user quota
doveadm quota get -u user@example.com
doveadm quota recalc -u user@example.com
```

## Namespaces and Shared Folders

```bash
# Public shared namespace (/etc/dovecot/conf.d/10-mail.conf)
# namespace public { type=public; prefix=Public/; location=maildir:/var/mail/public }
# User-to-user: namespace shared { type=shared; prefix=shared/%%u/; location=maildir:%%h/Maildir }

# ACL plugin: mail_plugins = $mail_plugins acl
# plugin { acl = vfile }
doveadm acl set -u user@example.com "INBOX/Shared" user=other@example.com lookup read write
doveadm acl get -u user@example.com "INBOX/Shared"
```

## Replication

```bash
# Enable: mail_plugins = $mail_plugins notify replication
# plugin { mail_replica = tcp:replica.example.com:12345 }
# doveadm_password = shared-secret

# Manual sync
doveadm sync -u user@example.com tcp:replica.example.com:12345
dsync mirror -u user@example.com tcp:replica.example.com:12345

# Check replication status
doveadm replicator status '*'
```

## doveadm Commands

```bash
# User and mailbox management
doveadm user '*'                          # list all users
doveadm mailbox list -u user@example.com  # list mailboxes
doveadm mailbox create -u user@example.com "Archive/2024"
doveadm mailbox delete -u user@example.com "OldFolder"
doveadm mailbox status all -u user@example.com '*'

# Search and fetch
doveadm search -u user@example.com mailbox INBOX since 30d
doveadm fetch -u user@example.com "hdr subject" mailbox INBOX since 7d

# Expunge (delete messages matching search)
doveadm expunge -u user@example.com mailbox Trash savedbefore 30d
doveadm expunge -u user@example.com mailbox Junk savedbefore 14d

# Purge (physically remove deleted messages in mdbox)
doveadm purge -u user@example.com

# Force resync / fix index
doveadm force-resync -u user@example.com '*'
doveadm index -u user@example.com INBOX

# Import mail from another source
doveadm import -u user@example.com maildir:~/old-mail "" mailbox INBOX

# Who is connected
doveadm who
doveadm kick user@example.com
```

## Postfix Integration (LMTP/deliver)

```bash
# LMTP delivery (recommended for virtual users)
# In Postfix main.cf:
# virtual_transport = lmtp:unix:private/dovecot-lmtp

# Dovecot LDA (older method, single-user)
# In Postfix main.cf:
# mailbox_command = /usr/lib/dovecot/deliver -f "$SENDER" -a "$RECIPIENT"

# SASL authentication (Postfix uses Dovecot for SMTP AUTH)
# /etc/dovecot/conf.d/10-master.conf
# service auth {
#   unix_listener /var/spool/postfix/private/auth {
#     mode = 0660
#     user = postfix
#     group = postfix
#   }
# }

# Postfix main.cf:
# smtpd_sasl_type = dovecot
# smtpd_sasl_path = private/auth
# smtpd_sasl_auth_enable = yes
```

## Full-Text Search (FTS)

```bash
# /etc/dovecot/conf.d/90-fts.conf
# mail_plugins = $mail_plugins fts fts_solr
# or
# mail_plugins = $mail_plugins fts fts_xapian

# FTS with Xapian (built-in, no external service)
# plugin {
#   fts = xapian
#   fts_xapian = partial=3 full=20 attachments=no
#   fts_autoindex = yes
#   fts_autoindex_max_recent_msgs = 99
#   fts_enforced = yes
# }

# FTS with Solr (external service)
# plugin {
#   fts = solr
#   fts_solr = url=http://localhost:8983/solr/dovecot/ break-imap-search
# }

# Rebuild search index
doveadm fts rescan -u user@example.com
doveadm index -u user@example.com '*'

# Search using FTS
doveadm search -u user@example.com body "quarterly report"
```

## Tips

- Use `sdbox` or `mdbox` for production workloads -- they outperform Maildir on large mailboxes due to fewer filesystem operations.
- Always set `ssl = required` in production; `ssl = yes` still allows plaintext on non-SSL ports.
- Compile Sieve scripts with `sievec` after editing to catch syntax errors before they cause delivery failures.
- Use LMTP for Postfix integration over LDA -- LMTP supports multiple recipients per connection and is more efficient.
- Set `quota_grace = 10%%` so users can receive a final warning message even when quota is nearly full.
- Enable `fts_autoindex = yes` to build search indexes incrementally rather than at first search (which stalls the client).
- Use `doveadm who` to see connected users and `doveadm kick` to disconnect problem sessions.
- Run `doveadm force-resync` when mailbox indexes become corrupted rather than deleting index files manually.
- Set `sieve_before` for system-wide filtering rules (spam filing) that users cannot override.
- Use replication for high availability -- dsync handles conflict resolution automatically.
- Configure `auth_cache_size = 8192` and `auth_cache_ttl = 1 hour` to reduce LDAP/SQL load.
- Monitor with `doveadm mailbox status` to track per-user message counts and storage.

## See Also

- postfix (MTA configuration, SASL and LMTP integration)
- email-security (DKIM/SPF verification before delivery)
- wireshark (IMAP/POP3 protocol analysis)

## References

- [Dovecot Official Documentation](https://doc.dovecot.org/)
- [Dovecot Wiki - Configuration](https://wiki.dovecot.org/FrontPage)
- [Dovecot Sieve](https://doc.dovecot.org/configuration_manual/sieve/)
- [Dovecot FTS](https://doc.dovecot.org/configuration_manual/fts/)
- [Dovecot Replication](https://doc.dovecot.org/admin_manual/replication/)
- [Pigeonhole Sieve Reference](https://pigeonhole.dovecot.org/)
