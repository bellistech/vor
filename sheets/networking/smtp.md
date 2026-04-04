# SMTP (Simple Mail Transfer Protocol)

Application-layer protocol for sending email between mail servers and from clients to servers, using a text-based command/response model over TCP port 25 (relay), 587 (submission), or 465 (implicit TLS).

## SMTP Transaction Flow

```
Client                                    Server
  |                                         |
  |  --- TCP connect (port 25/587) -------> |
  |  <-- 220 mail.example.com ESMTP ------- |
  |  --- EHLO client.example.com ---------> |
  |  <-- 250-mail.example.com ------------- |
  |       250-STARTTLS                      |
  |       250-AUTH PLAIN LOGIN              |
  |       250 SIZE 52428800                 |
  |  --- STARTTLS ------------------------> |
  |  <-- 220 Ready to start TLS ---------- |
  |  <<< TLS handshake >>>                  |
  |  --- EHLO client.example.com ---------> |
  |  --- AUTH PLAIN <base64> --------------> |
  |  <-- 235 Authentication successful ---- |
  |  --- MAIL FROM:<alice@example.com> ----> |
  |  <-- 250 OK --------------------------- |
  |  --- RCPT TO:<bob@target.com> ---------> |
  |  <-- 250 OK --------------------------- |
  |  --- DATA -----------------------------> |
  |  <-- 354 Start mail input ------------- |
  |  --- headers + body + CRLF.CRLF ------> |
  |  <-- 250 OK: queued as ABC123 --------- |
  |  --- QUIT -----------------------------> |
  |  <-- 221 Bye -------------------------- |
```

## MX Record Lookup

```bash
# Find mail servers for a domain
dig MX example.com +short
# 10 mail1.example.com.
# 20 mail2.example.com.

# Verify MX reachability
dig MX example.com +short | awk '{print $2}' | while read mx; do
  echo "=== $mx ==="
  dig A "$mx" +short
done

# Check reverse DNS (required by many servers)
dig -x 203.0.113.10 +short
```

## Testing SMTP with Telnet/OpenSSL

```bash
# Plain text SMTP test (port 25)
telnet mail.example.com 25

# TLS-wrapped connection (port 465)
openssl s_client -connect mail.example.com:465

# STARTTLS on submission port (port 587)
openssl s_client -starttls smtp -connect mail.example.com:587

# Send test email via openssl
openssl s_client -starttls smtp -connect mail.example.com:587 -quiet << 'EOF'
EHLO test.local
AUTH PLAIN AGFsaWNlQGV4YW1wbGUuY29tAHNlY3JldA==
MAIL FROM:<alice@example.com>
RCPT TO:<bob@example.com>
DATA
Subject: Test
From: alice@example.com
To: bob@example.com

Hello from SMTP test.
.
QUIT
EOF
```

## SMTP Status Codes

```
2xx — Success
  211  System status
  220  Service ready
  221  Closing connection
  235  Authentication successful
  250  Requested action OK
  251  User not local; will forward
  252  Cannot VRFY but will accept

3xx — Intermediate
  334  AUTH continuation (server challenge)
  354  Start mail input, end with <CRLF>.<CRLF>

4xx — Temporary Failure (retry later)
  421  Service not available, closing channel
  450  Mailbox unavailable (busy/policy)
  451  Local error in processing
  452  Insufficient storage

5xx — Permanent Failure
  500  Syntax error / command not recognized
  501  Syntax error in parameters
  502  Command not implemented
  503  Bad sequence of commands
  504  Parameter not implemented
  530  Authentication required
  535  Authentication failed
  550  Mailbox unavailable (not found/policy)
  551  User not local
  552  Exceeded storage allocation
  553  Mailbox name not allowed
  554  Transaction failed
```

## SPF, DKIM, and DMARC

```bash
# Check SPF record
dig TXT example.com +short | grep "v=spf1"
# "v=spf1 include:_spf.google.com ip4:203.0.113.0/24 -all"

# SPF qualifiers: + pass, - fail, ~ softfail, ? neutral

# Check DKIM record (selector from email header)
dig TXT selector1._domainkey.example.com +short
# "v=DKIM1; k=rsa; p=MIIBIjANBg..."

# Check DMARC policy
dig TXT _dmarc.example.com +short
# "v=DMARC1; p=reject; rua=mailto:dmarc@example.com; pct=100"

# DMARC policies: none (monitor), quarantine (spam folder), reject (bounce)
```

## Postfix Configuration

```bash
# Main config: /etc/postfix/main.cf
# Key settings for a sending relay

# Identity
myhostname = mail.example.com
mydomain = example.com
myorigin = $mydomain

# Network
inet_interfaces = all
inet_protocols = ipv4
mynetworks = 127.0.0.0/8, 10.0.0.0/8

# TLS (outbound)
smtp_tls_security_level = may
smtp_tls_loglevel = 1
smtp_tls_CAfile = /etc/ssl/certs/ca-certificates.crt

# TLS (inbound)
smtpd_tls_security_level = may
smtpd_tls_cert_file = /etc/letsencrypt/live/mail.example.com/fullchain.pem
smtpd_tls_key_file = /etc/letsencrypt/live/mail.example.com/privkey.pem

# Authentication (submission)
smtpd_sasl_auth_enable = yes
smtpd_sasl_type = dovecot
smtpd_sasl_path = private/auth

# Restrictions
smtpd_relay_restrictions = permit_mynetworks,
    permit_sasl_authenticated,
    reject_unauth_destination

# Reload after changes
sudo postfix reload
```

## Postfix Queue Management

```bash
# View mail queue
mailq
postqueue -p

# Flush the queue (retry all)
postqueue -f

# Flush specific domain
postqueue -s example.com

# Delete all queued mail
postsuper -d ALL

# Delete deferred mail only
postsuper -d ALL deferred

# Hold a message
postsuper -h QUEUE_ID

# Release a held message
postsuper -H QUEUE_ID

# View message content
postcat -q QUEUE_ID

# Check mail log
tail -f /var/log/mail.log
journalctl -u postfix -f
```

## Sendmail Compatibility

```bash
# Send email via sendmail interface (works with postfix too)
echo "Subject: Test" | sendmail -f sender@example.com recipient@example.com

# Send with full headers
sendmail -t << 'EOF'
From: alice@example.com
To: bob@example.com
Subject: Test Message
MIME-Version: 1.0
Content-Type: text/plain; charset=UTF-8

This is the body.
EOF

# Test delivery without actually sending
sendmail -bv recipient@example.com
```

## swaks (Swiss Army Knife for SMTP)

```bash
# Install
sudo apt install swaks    # Debian/Ubuntu
brew install swaks        # macOS

# Basic test
swaks --to bob@example.com --from alice@example.com \
  --server mail.example.com

# Authenticated submission
swaks --to bob@example.com --from alice@example.com \
  --server mail.example.com --port 587 \
  --auth LOGIN --auth-user alice@example.com \
  --tls

# Test with attachment
swaks --to bob@example.com --from alice@example.com \
  --server mail.example.com --port 587 --tls \
  --attach /path/to/file.pdf \
  --header "Subject: Report attached"

# DKIM-signed send (with opendkim)
swaks --to bob@example.com --from alice@example.com \
  --server localhost --port 10027
```

## Python smtplib

```python
import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

msg = MIMEMultipart()
msg["From"] = "alice@example.com"
msg["To"] = "bob@example.com"
msg["Subject"] = "Test from Python"
msg.attach(MIMEText("Hello from smtplib.", "plain"))

with smtplib.SMTP("mail.example.com", 587) as server:
    server.starttls()
    server.login("alice@example.com", "password")
    server.send_message(msg)
```

## SMTP Authentication Encoding

```bash
# AUTH PLAIN: base64("\0username\0password")
echo -ne '\0alice@example.com\0secret' | base64
# AGFsaWNlQGV4YW1wbGUuY29tAHNlY3JldA==

# AUTH LOGIN: base64(username) then base64(password) separately
echo -n 'alice@example.com' | base64
echo -n 'secret' | base64
```

## Tips

- Always use port 587 (submission) with STARTTLS for client-to-server; port 25 is for server-to-server relay
- Set up SPF, DKIM, and DMARC together; missing any one tanks deliverability to Gmail/Outlook
- Reverse DNS (PTR record) must match the HELO/EHLO hostname or many servers will reject
- Use `swaks` for SMTP debugging; it is far superior to raw telnet for testing
- Check blacklists at mxtoolbox.com/blacklists if mail is bouncing with 550 errors
- Rate-limit outbound mail (`smtpd_client_message_rate_limit` in Postfix) to avoid IP reputation damage
- DKIM key rotation: use selectors like `2026q1` so you can publish new keys before revoking old
- Monitor the deferred queue; a growing deferred queue usually means DNS, network, or reputation issues
- Set `message_size_limit` in Postfix to prevent abuse (default 10MB is often fine)
- Enable `smtp_tls_security_level = dane` for DANE/TLSA if your DNS supports DNSSEC
- Log and monitor DMARC aggregate reports (rua) to catch spoofing and misconfiguration early
- Never run an open relay; always set `smtpd_relay_restrictions` to reject unauthenticated relay

## See Also

- dns, tls, curl, postfix, dovecot, spf, dkim

## References

- [RFC 5321 — SMTP](https://datatracker.ietf.org/doc/html/rfc5321)
- [RFC 6409 — Message Submission (Port 587)](https://datatracker.ietf.org/doc/html/rfc6409)
- [RFC 8314 — Implicit TLS (Port 465)](https://datatracker.ietf.org/doc/html/rfc8314)
- [RFC 7208 — SPF](https://datatracker.ietf.org/doc/html/rfc7208)
- [RFC 6376 — DKIM](https://datatracker.ietf.org/doc/html/rfc6376)
- [RFC 7489 — DMARC](https://datatracker.ietf.org/doc/html/rfc7489)
- [Postfix Documentation](https://www.postfix.org/documentation.html)
- [swaks — SMTP Test Tool](https://www.jetmore.org/john/code/swaks/)
