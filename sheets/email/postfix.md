# Postfix (Mail Transfer Agent)

Full-featured MTA for routing, delivering, and relaying email on Unix systems.

## Core Configuration Files

### main.cf - Global Settings

```bash
# View effective configuration (non-default values)
postconf -n

# View all configuration with defaults
postconf

# View a specific parameter
postconf mydomain myhostname

# Set a parameter (persists to main.cf)
postconf -e "myhostname = mail.example.com"
postconf -e "mydomain = example.com"
postconf -e "myorigin = \$mydomain"
postconf -e "mydestination = \$myhostname, localhost.\$mydomain, localhost"
postconf -e "inet_interfaces = all"
postconf -e "inet_protocols = ipv4"

# Reload after changes
postfix reload
```

### master.cf - Service Definitions

```bash
# Standard SMTP on port 25
# smtp      inet  n  -  n  -  -  smtpd

# Submission on port 587
# submission inet  n  -  n  -  -  smtpd
#   -o syslog_name=postfix/submission
#   -o smtpd_tls_security_level=encrypt
#   -o smtpd_sasl_auth_enable=yes
#   -o smtpd_reject_unlisted_recipient=no
#   -o smtpd_recipient_restrictions=permit_sasl_authenticated,reject

# SMTPS on port 465
# smtps     inet  n  -  n  -  -  smtpd
#   -o syslog_name=postfix/smtps
#   -o smtpd_tls_wrappermode=yes
#   -o smtpd_sasl_auth_enable=yes
```

## Virtual Domains and Mailboxes

```bash
# Virtual mailbox domains (not in mydestination)
postconf -e "virtual_mailbox_domains = example.com, example.org"
postconf -e "virtual_mailbox_base = /var/mail/vhosts"
postconf -e "virtual_mailbox_maps = hash:/etc/postfix/vmailbox"
postconf -e "virtual_uid_maps = static:5000"
postconf -e "virtual_gid_maps = static:5000"

# /etc/postfix/vmailbox
# user@example.com    example.com/user/
# info@example.org    example.org/info/

# Virtual alias maps
postconf -e "virtual_alias_maps = hash:/etc/postfix/virtual"

# /etc/postfix/virtual
# postmaster@example.com  admin@example.com
# @example.org            catchall@example.com

# Rebuild hash maps after editing
postmap /etc/postfix/vmailbox
postmap /etc/postfix/virtual
```

## Transport Maps

```bash
# Route mail for specific domains to different transports
postconf -e "transport_maps = hash:/etc/postfix/transport"

# /etc/postfix/transport
# internal.corp    smtp:[10.0.0.5]:25
# archive.com      lmtp:unix:/var/run/dovecot/lmtp
# .example.com     smtp:[relay.example.com]
# *                smtp:[gateway.isp.com]

postmap /etc/postfix/transport
```

## Milter Integration

```bash
# OpenDKIM milter
postconf -e "milter_protocol = 6"
postconf -e "milter_default_action = accept"
postconf -e "smtpd_milters = inet:localhost:8891, inet:localhost:8893"
postconf -e "non_smtpd_milters = inet:localhost:8891"

# OpenDMARC milter (after OpenDKIM)
# smtpd_milters = inet:localhost:8891, inet:localhost:8893

# SpamAssassin via milter
# spamass-milter typically on inet:localhost:783
```

## Relay Controls and Restrictions

```bash
# Trusted networks allowed to relay
postconf -e "mynetworks = 127.0.0.0/8, 10.0.0.0/24"

# Relay domains (domains we accept mail for relay)
postconf -e "relay_domains = partner.com"

# Recipient restrictions (evaluated in order)
postconf -e "smtpd_recipient_restrictions = \
  permit_mynetworks, \
  permit_sasl_authenticated, \
  reject_unauth_destination, \
  reject_invalid_hostname, \
  reject_non_fqdn_hostname, \
  reject_non_fqdn_sender, \
  reject_non_fqdn_recipient, \
  reject_unknown_sender_domain, \
  reject_unknown_recipient_domain, \
  reject_rbl_client zen.spamhaus.org, \
  permit"

# Sender restrictions
postconf -e "smtpd_sender_restrictions = \
  reject_non_fqdn_sender, \
  reject_unknown_sender_domain"

# HELO restrictions
postconf -e "smtpd_helo_required = yes"
postconf -e "smtpd_helo_restrictions = \
  permit_mynetworks, \
  reject_invalid_helo_hostname, \
  reject_non_fqdn_helo_hostname"
```

## Queue Management

```bash
# List queued messages
mailq
# or
postqueue -p

# Flush the queue (attempt delivery of all)
postqueue -f

# Flush deferred queue only
postqueue -f

# Process a specific queue ID
postqueue -i QUEUE_ID

# Delete a specific message
postsuper -d QUEUE_ID

# Delete ALL queued messages
postsuper -d ALL

# Delete all from deferred queue
postsuper -d ALL deferred

# Hold a message
postsuper -h QUEUE_ID

# Release a held message
postsuper -H QUEUE_ID

# Requeue messages (re-resolve recipients)
postsuper -r QUEUE_ID

# View a queued message
postcat -q QUEUE_ID

# Queue statistics
qshape        # show queue shape by domain
qshape deferred
```

## Header and Body Checks

```bash
# Header checks (regexp-based)
postconf -e "header_checks = regexp:/etc/postfix/header_checks"

# /etc/postfix/header_checks
# /^Subject:.*viagra/i    REJECT spam subject
# /^X-Mailer: EvilBot/    DISCARD
# /^From:.*spammer/i      REJECT known spammer
# /^Received:/            IGNORE   (strip received headers)

# Body checks
postconf -e "body_checks = regexp:/etc/postfix/body_checks"

# /etc/postfix/body_checks
# /click here to unsubscribe.*free/i   REJECT body spam pattern

# MIME header checks
postconf -e "mime_header_checks = regexp:/etc/postfix/mime_header_checks"

# /etc/postfix/mime_header_checks
# /name=[^>]*\.(exe|bat|cmd|scr)/  REJECT dangerous attachment
```

## Rate Limiting

```bash
# Anvil-based client rate limiting
postconf -e "smtpd_client_connection_rate_limit = 50"
postconf -e "smtpd_client_connection_count_limit = 20"
postconf -e "smtpd_client_message_rate_limit = 100"
postconf -e "smtpd_client_recipient_rate_limit = 200"

# Exempt trusted networks from rate limits
postconf -e "smtpd_client_event_limit_exceptions = \$mynetworks"

# Anvil status window
postconf -e "anvil_rate_time_unit = 60s"
postconf -e "anvil_status_update_time = 600s"

# Default delivery concurrency
postconf -e "default_destination_concurrency_limit = 20"
postconf -e "smtp_destination_concurrency_limit = 20"
postconf -e "default_destination_rate_delay = 1s"
```

## TLS Configuration

```bash
# Inbound TLS (server)
postconf -e "smtpd_tls_cert_file = /etc/letsencrypt/live/mail.example.com/fullchain.pem"
postconf -e "smtpd_tls_key_file = /etc/letsencrypt/live/mail.example.com/privkey.pem"
postconf -e "smtpd_tls_security_level = may"
postconf -e "smtpd_tls_protocols = >=TLSv1.2"
postconf -e "smtpd_tls_mandatory_protocols = >=TLSv1.2"
postconf -e "smtpd_tls_loglevel = 1"
postconf -e "smtpd_tls_session_cache_database = btree:\${data_directory}/smtpd_scache"

# Outbound TLS (client)
postconf -e "smtp_tls_security_level = may"
postconf -e "smtp_tls_protocols = >=TLSv1.2"
postconf -e "smtp_tls_loglevel = 1"
postconf -e "smtp_tls_session_cache_database = btree:\${data_directory}/smtp_scache"

# Enforce TLS to specific destinations
postconf -e "smtp_tls_policy_maps = hash:/etc/postfix/tls_policy"
# /etc/postfix/tls_policy
# example.com     encrypt
# .gov            encrypt protocols=TLSv1.2
```

## Multi-Instance Postfix

```bash
postmulti -e init                                    # enable multi-instance
postmulti -I postfix-out -G mta -e create            # create instance
postmulti -i postfix-out -x postconf -e "myhostname = out.example.com"
postmulti -i postfix-out -x postconf -e "inet_interfaces = 10.0.0.2"
postmulti -i postfix-out -e enable                   # enable instance
postmulti -i postfix-out -p start                    # start instance
postmulti -l                                         # list instances
```

## Aliases and Canonical Maps

```bash
# Local alias database
postconf -e "alias_maps = hash:/etc/aliases"
postconf -e "alias_database = hash:/etc/aliases"

# /etc/aliases
# postmaster:    root
# root:          admin
# abuse:         admin
# mailer-daemon: postmaster

# Rebuild aliases
newaliases
# or
postalias /etc/aliases

# Canonical maps (rewrite envelope/header addresses)
postconf -e "sender_canonical_maps = hash:/etc/postfix/sender_canonical"
postconf -e "recipient_canonical_maps = hash:/etc/postfix/recipient_canonical"

# /etc/postfix/sender_canonical
# @oldname.com   @newname.com

postmap /etc/postfix/sender_canonical
```

## Diagnostics

```bash
# Check configuration for errors
postfix check

# Test address rewriting
postmap -q "user@example.com" hash:/etc/postfix/virtual

# Trace mail delivery for a specific address
sendmail -bv user@example.com

# Watch the mail log
tail -f /var/log/mail.log

# Connection test
openssl s_client -connect mail.example.com:465
openssl s_client -starttls smtp -connect mail.example.com:587
```

## Tips

- Always run `postfix check` after editing main.cf or master.cf before reloading.
- Use `postconf -n` to see only non-default settings -- much easier to audit than `postconf` alone.
- Keep `mydestination` minimal when using virtual domains; overlap causes confusion.
- Set `smtpd_tls_security_level = may` for opportunistic TLS on port 25; use `encrypt` only on submission (587).
- Use `reject_unauth_destination` early in recipient restrictions to prevent open relay.
- Map files (hash, regexp) must be rebuilt with `postmap` after every edit; aliases need `newaliases`.
- Check `qshape deferred` regularly to spot delivery problems by destination domain.
- Run `postsuper -d ALL deferred` with caution -- legitimate retries will be lost.
- Use `smtp_tls_policy_maps` to enforce TLS per-destination rather than globally.
- Set `smtpd_helo_required = yes` to block poorly-written spam bots that skip HELO.
- Use separate Postfix instances (postmulti) to isolate inbound vs outbound traffic.
- Monitor `anvil` rate-limit stats via `postconf -e "anvil_status_update_time = 600s"` and check logs.

## See Also

- email-security (SPF, DKIM, DMARC, ARC configuration)
- dovecot (IMAP/POP3 server, LMTP delivery integration)
- wireshark (SMTP packet capture and analysis)

## References

- [Postfix Official Documentation](http://www.postfix.org/documentation.html)
- [Postfix Configuration Parameters](http://www.postfix.org/postconf.5.html)
- [Postfix Virtual Domain Hosting](http://www.postfix.org/VIRTUAL_README.html)
- [Postfix TLS Support](http://www.postfix.org/TLS_README.html)
- [Postfix SASL Howto](http://www.postfix.org/SASL_README.html)
- [Postfix Rate Limiting](http://www.postfix.org/TUNING_README.html)
