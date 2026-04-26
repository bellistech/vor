# TLS Errors

Every TLS handshake failure, browser-side error, OpenSSL alert and verify code with cause and fix — the searchable catalog for terminal-bound TLS debugging.

## Setup

TLS exists to provide confidentiality, integrity, and authentication on top of TCP. The handshake is the part that fails most often, and almost every error you see — whether from `curl`, `openssl s_client`, a browser, or your application logs — maps back to a specific step in that handshake or to certificate validation.

### TLS 1.2 Handshake (RFC 5246)

```text
Client                                               Server

ClientHello                  -------->
                                                ServerHello
                                               Certificate*
                                         ServerKeyExchange*
                                        CertificateRequest*
                             <--------      ServerHelloDone
Certificate*
ClientKeyExchange
CertificateVerify*
[ChangeCipherSpec]
Finished                     -------->
                                         [ChangeCipherSpec]
                             <--------             Finished
Application Data             <------->     Application Data
```

Steps:

1. **ClientHello** — client offers a random nonce, supported TLS versions, cipher suites, compression methods, and extensions (SNI, ALPN, signature algorithms, supported curves, etc.).
2. **ServerHello** — server picks one TLS version, one cipher suite, sends its random nonce.
3. **Certificate** — server sends its X.509 cert chain (leaf + intermediates).
4. **ServerKeyExchange** — for ephemeral key exchanges (ECDHE/DHE), server sends its ephemeral public params signed with its long-term private key.
5. **CertificateRequest** (optional, mTLS) — server asks client for a cert.
6. **ServerHelloDone** — server done with its side of hello.
7. **Certificate** (optional, mTLS) — client sends its cert.
8. **ClientKeyExchange** — client contributes pre-master secret (RSA: encrypted with server pub key; ECDHE: client ephemeral public).
9. **CertificateVerify** (optional, mTLS) — client signs handshake transcript with private key to prove possession.
10. **ChangeCipherSpec / Finished** — both sides switch to the negotiated keys and verify the handshake transcript hash.

### TLS 1.3 Handshake (RFC 8446)

```text
Client                                               Server

Key  ^ ClientHello
Exch | + key_share*
     | + signature_algorithms*
     | + psk_key_exchange_modes*
     v + pre_shared_key*       -------->
                                                  ServerHello  ^ Key
                                                 + key_share*  | Exch
                                            + pre_shared_key*  v
                                        {EncryptedExtensions}  ^  Server
                                        {CertificateRequest*}  v  Params
                                               {Certificate*}  ^
                                         {CertificateVerify*}  | Auth
                                                   {Finished}  v
                               <--------  [Application Data*]
     ^ {Certificate*}
Auth | {CertificateVerify*}
     v {Finished}              -------->
       [Application Data]      <------->  [Application Data]
```

- **1-RTT** — full handshake completes in one round trip, application data flows on the second flight.
- **0-RTT** — if client and server share a PSK from a prior session, client can send early data with the ClientHello (replay risk; use only for idempotent requests).
- TLS 1.3 encrypts the certificate and most extensions immediately after the ServerHello (`{}` braces above mean encrypted).
- Key exchange is always ephemeral DH; RSA key transport is gone.

### Certificate Chain Validation

When a client receives a server cert, it walks up the chain:

1. Parse the leaf cert; check `notBefore` ≤ now ≤ `notAfter`.
2. Verify the leaf signature using the issuer's public key (next cert up).
3. Repeat for each intermediate.
4. The top of the chain must be signed by a cert in the client's trust store (a trusted root CA), or be the trusted root itself.
5. Check the leaf's subject (CN or SAN) matches the hostname the client connected to.
6. Check Key Usage / Extended Key Usage extensions allow `serverAuth`.
7. Optionally check OCSP / CRL for revocation.
8. Optionally check Certificate Transparency SCTs (Chrome enforces this for public certs since 2018).

### SNI — Server Name Indication

```bash
# Without SNI, server has to guess which cert to send for IP-shared hosts:
openssl s_client -connect 1.2.3.4:443

# With SNI (the right way):
openssl s_client -connect 1.2.3.4:443 -servername example.com
```

SNI is a TLS extension (RFC 6066) that lets the client tell the server the hostname it's trying to reach in the ClientHello, before any certificate is sent. Without it, the server will send the default cert (often the wrong one for vhost setups).

### ALPN — Application-Layer Protocol Negotiation

```bash
# Test if a server supports HTTP/2:
openssl s_client -connect example.com:443 -alpn h2,http/1.1

# Output includes:
# ALPN protocol: h2
```

ALPN (RFC 7301) lets the client list app-layer protocols (`h2`, `http/1.1`, `h3` for HTTP/3 over QUIC) it can speak; the server picks one from that list. Negotiated during the handshake — before any HTTP request — to avoid a round trip.

## How to Read a TLS Error

Three questions:

1. **Which side detected the error?** — client (your `curl`/browser/app) or server (logs in nginx/Apache/your service)?
2. **Which step failed?**
   - **Handshake** — anything before the encrypted Application Data phase. Most common.
   - **Certificate verification** — chain trust, hostname match, expiry, revocation.
   - **Record-level** — bad MAC, oversized record, decryption failure (rare; usually MITM or version confusion).
3. **What's the alert level?**
   - `warning` (level 1) — connection may continue (rare in practice).
   - `fatal` (level 2) — connection terminates immediately.

A TLS alert looks like this on the wire:

```text
struct {
  AlertLevel   level;       // 1 = warning, 2 = fatal
  AlertDescription description;  // 0..255 — see catalog below
} Alert;
```

In OpenSSL output you'll see lines like:

```text
SSL alert read 0:close notify
SSL alert write 0:close notify
SSL3 alert read:fatal:handshake failure
```

The format is:

```text
<protocol family> alert <read|write>:<level>:<description>
```

`read` means the alert came from the peer; `write` means we sent it.

### Anatomy of a Common Error

```text
curl: (60) SSL certificate problem: unable to get local issuer certificate
```

- **Detector**: client (curl).
- **Step**: certificate verification (chain).
- **Cause**: server didn't send the intermediate cert; client doesn't have it locally.
- **Fix**: add the missing intermediate to the server's bundle (concatenate `cert.pem intermediate.pem` and serve that).

```text
SSL3 alert write:fatal:bad certificate
```

- **Detector**: server (sent the alert).
- **Step**: handshake — server is rejecting client's cert.
- **Cause**: mTLS configured; client's cert is malformed, expired, or signed by an unknown CA.

```text
tls: handshake failure
```

- **Detector**: usually a Go service (this is the literal `crypto/tls` error string).
- **Step**: handshake.
- **Cause**: no overlap between client's offered cipher suites/versions and what the server allows. Check both ends.

## TLS Alert Codes

Verbatim from RFC 8446 (TLS 1.3) and RFC 5246 (TLS 1.2). Code in parentheses; "1.3-only" / "1.2-only" notes where applicable.

### close_notify (0)

Notifies the recipient that the sender will not send any more on this connection. Either side may send. Not a failure — graceful shutdown. Required (in 1.2) before TCP close, though many implementations now permit close-without-close-notify.

```text
SSL alert read 0:close notify
```

**Cause**: peer closed the TLS session cleanly.
**Fix**: nothing to fix; informational.

### unexpected_message (10)

A message was received that was inappropriate to the current state.

**Cause**: protocol confusion — e.g., HTTP server speaking HTTP when client expects TLS, or message ordering broken by a middlebox.
**Fix**: confirm the server is actually a TLS server on the port; check for transparent proxies that may corrupt the framing.

### bad_record_mac (20)

Record's MAC didn't verify. Always fatal.

**Cause**: corruption (rare), wrong keys (a bug or downgrade attack), or a middlebox that decrypts and re-encrypts without proper key sync.
**Fix**: check for SSL-intercepting proxies; ensure both ends use the same TLS implementation versions; rule out hardware/network faults.

### record_overflow (22)

A TLS record was received with more than 2^14 + 2048 bytes (1.2) or 2^14 + 256 (1.3).

**Cause**: misbehaving peer, or version mismatch where one side thinks it's compressing and the other doesn't.
**Fix**: capture pcap, identify the offender; usually fixed by upgrading the peer.

### handshake_failure (40)

Sender unable to negotiate an acceptable set of security parameters.

**Cause**: very common catch-all. Usually means no shared cipher suite, no shared TLS version, or no acceptable curve/group.
**Fix**: capture ClientHello and ServerHello (use `openssl s_client -msg` or Wireshark); reconcile the cipher/version lists; remove restrictions or update one side.

```bash
openssl s_client -connect example.com:443 -msg -tls1_2
```

### bad_certificate (42)

Certificate was corrupt, contained signatures that did not verify correctly, etc.

**Cause**: malformed cert, signature mismatch, mTLS rejection of a syntactically broken client cert.
**Fix**: re-issue cert; verify with `openssl x509 -in cert.pem -text -noout` first.

### unsupported_certificate (43)

Certificate was of an unsupported type.

**Cause**: server sent a cert with an algorithm the client doesn't support (e.g., Ed25519 cert to a client that only supports RSA/ECDSA).
**Fix**: align the cert algorithm with what the client accepts.

### certificate_revoked (44)

Certificate was revoked by its signer.

**Cause**: CA published a revocation; client checked OCSP or CRL.
**Fix**: get a new cert.

### certificate_expired (45)

Certificate has expired or is not currently valid.

**Cause**: leaf or intermediate cert outside its validity window.
**Fix**: renew the cert; check system clock on both ends.

### certificate_unknown (46)

Some unspecified issue arose in processing the certificate, rendering it unacceptable.

**Cause**: catch-all for cert processing failures (revocation lookup failure, policy mismatch, etc.).
**Fix**: enable verbose logging on the rejecting side; inspect the cert chain.

### illegal_parameter (47)

A field in the handshake was out of range or inconsistent with other fields.

**Cause**: malformed handshake message; could be a buggy client/server or fuzz testing.
**Fix**: pcap it; report to vendor.

### unknown_ca (48)

Valid cert chain or partial chain was received, but the CA could not be located or could not be matched with a known, trusted CA.

**Cause**: client's trust store doesn't have the root that signed the chain; or server didn't send the intermediate that connects leaf to a trusted root.
**Fix**: install the missing root CA into the client trust store; or fix the server bundle to include all intermediates.

```bash
# Inspect what the server is actually sending:
openssl s_client -connect example.com:443 -showcerts
```

### access_denied (49)

A valid certificate was received, but when access control was applied, the sender decided not to proceed with negotiation.

**Cause**: cert is technically valid but server policy rejects it (mTLS allow-list, etc.).
**Fix**: get a cert from an issuer the server's policy permits, or update the server policy.

### decode_error (50)

A message could not be decoded because some field was out of the specified range or the length was incorrect.

**Cause**: peer sent a syntactically invalid message.
**Fix**: pcap; report.

### decrypt_error (51)

A handshake cryptographic operation failed, including being unable to correctly verify a signature, decrypt a key exchange, or validate a Finished message.

**Cause**: the peer's signature on CertificateVerify or Finished didn't verify. Often a bug or a wrong private key paired with a cert. In mTLS, client's signature didn't match its cert.
**Fix**: confirm cert and key on each side actually match (`openssl x509 -modulus -in cert | openssl md5` vs `openssl rsa -modulus -in key | openssl md5`).

### protocol_version (70)

Protocol version the client has attempted to negotiate is recognized but not supported.

**Cause**: client offers TLS 1.0/1.1; server requires TLS 1.2+. Or client offers SSLv3 to a TLS-only server.
**Fix**: upgrade the client; or — only if you must — re-enable older versions on the server (security risk).

```text
tlsv1 alert protocol version
```

### insufficient_security (71)

Returned in lieu of `handshake_failure` when a negotiation has failed specifically because the server requires ciphers more secure than those supported by the client.

**Cause**: server requires AEAD/PFS ciphers; client only offers weak ones (RC4, 3DES, export ciphers, static RSA).
**Fix**: upgrade client; enable strong ciphers on the client side.

### internal_error (80)

Internal error unrelated to the peer or the correctness of the protocol (memory failure, etc.).

**Cause**: bug or resource exhaustion on the sending side.
**Fix**: check sender's logs; restart; file bug.

### inappropriate_fallback (86)

Sent by a server in response to an invalid connection retry attempt from a client (RFC 7507).

**Cause**: client included `TLS_FALLBACK_SCSV` in its cipher list, indicating a downgrade attempt; server detected that it could speak a higher version and refused. Defends against POODLE-style downgrade.
**Fix**: don't downgrade unnecessarily — most clients only do this if the previous handshake failed; investigate why the higher-version handshake fails first.

### user_cancelled (90)

Sent if the user cancels the handshake for some reason unrelated to a protocol failure.

**Cause**: human action — e.g., user declined a client cert prompt.
**Fix**: usually informational.

### missing_extension (109, 1.3-only)

Sent when an extension that is mandatory was missing.

**Cause**: TLS 1.3 requires `supported_versions`, `key_share`, and (for cert auth) `signature_algorithms`. A peer didn't include one.
**Fix**: upgrade peer; ensure extension support.

### unsupported_extension (110)

Sent when peer included an extension in a message that was not permitted.

**Cause**: server returned an extension the client didn't ask for; or vice versa.
**Fix**: bug in implementation; upgrade.

### unrecognized_name (112)

Server name in SNI doesn't correspond to a name the server knows.

**Cause**: SNI value sent by client doesn't match any vhost on the server.
**Fix**: use the right hostname; or stop sending SNI if the server doesn't expect it.

```bash
openssl s_client -connect 1.2.3.4:443 -servername wrong.example.com
# May yield:
# tlsv1 unrecognized name
```

### bad_certificate_status_response (113)

Sent when an invalid or unacceptable OCSP response is received.

**Cause**: stapled OCSP response is expired, malformed, or signed by the wrong CA.
**Fix**: refresh OCSP staple on server; check cert's AIA URL points at a working OCSP responder.

### unknown_psk_identity (115, 1.3-only)

PSK identity provided is not known.

**Cause**: client offered a session resumption ticket the server doesn't recognize (rotated, expired, server restarted).
**Fix**: client should fall back to full handshake (most do).

### certificate_required (116, 1.3-only)

Server requires a client cert (mTLS) and the client did not send one.

**Cause**: mTLS server enforces client cert; client didn't present one.
**Fix**: configure client with `--cert` and `--key`.

### no_application_protocol (120)

Sent by servers when a client `application_layer_protocol_negotiation` extension advertises only protocols that the server does not support.

**Cause**: ALPN mismatch — client wants `h2` only; server only supports `http/1.1`.
**Fix**: align ALPN lists; add fallback.

```bash
openssl s_client -connect example.com:443 -alpn h2
# If server doesn't support h2:
# tlsv1 alert no application protocol
```

## Common Handshake Failures

Verbatim error strings, cause, and fix.

### "no shared cipher"

```text
SSL_R_NO_SHARED_CIPHER:no shared cipher
```

**Cause**: client and server share no common cipher suite. Often happens when:

- Client only offers TLS 1.3 ciphers but server is TLS 1.2 max.
- Server policy is "only ECDHE-ECDSA" but client only has an RSA cert.
- Client offers only ChaCha20 but server has it disabled.
- One side has FIPS mode constraining ciphers.

**Fix**: dump both sides' offered ciphers:

```bash
# Client offer (visible in ClientHello):
openssl s_client -connect host:443 -msg | grep -A20 ClientHello

# Server's accepted set (try a wide cipher list):
openssl s_client -connect host:443 -cipher 'ALL'

# What does a cipher string expand to?
openssl ciphers -v 'ECDHE+AESGCM:ECDHE+CHACHA20'
```

Then fix the policy on whichever side is too narrow.

### "wrong version number"

```text
SSL_R_WRONG_VERSION_NUMBER:wrong version number
```

**Cause**: TLS records start with a version byte; this error means the byte was nonsense. Almost always:

- HTTPS client connected to an HTTP-only port.
- Plain TCP service on the port (server doesn't speak TLS at all).
- TLS hello fragmented across packets in a way that caused parsing to fail.

**Fix**: confirm the port speaks TLS:

```bash
openssl s_client -connect host:port < /dev/null
# If output is "HTTP/1.1 400 Bad Request" — it's HTTP, not HTTPS.
# If output is "wrong version number" — port isn't TLS.
```

Also check for STARTTLS-style protocols (SMTP, IMAP, FTP) where TLS is upgraded after a plain greeting:

```bash
openssl s_client -connect smtp.example.com:587 -starttls smtp
```

### "decryption failed or bad record mac"

```text
SSL_R_DECRYPTION_FAILED_OR_BAD_RECORD_MAC:decryption failed or bad record mac
```

**Cause**: a record didn't decrypt cleanly. In modern TLS this is one alert (combined to prevent padding-oracle distinguishing). Common causes:

- TLS-intercepting middlebox doing MITM badly.
- Network corruption (rare with TCP, but possible with bad NICs / cables).
- Application reused a `SSL_CTX` across processes via `fork()` without re-initializing.

**Fix**:

```bash
# Bypass any intercepting proxy:
unset https_proxy http_proxy
curl --noproxy '*' https://example.com/

# Capture and look at sequence numbers:
sudo tcpdump -i any -w tls.pcap port 443
```

### "tlsv1 alert protocol version"

```text
SSL_R_TLSV1_ALERT_PROTOCOL_VERSION:tlsv1 alert protocol version
```

**Cause**: server received a `protocol_version` (70) alert. The client offered a TLS version the server doesn't allow — usually TLS 1.0 or 1.1 against a server that requires 1.2+.

**Fix**: upgrade the client; or use `--tlsv1.2` flag explicitly:

```bash
curl --tlsv1.2 https://example.com/
```

If you absolutely cannot upgrade the client, temporarily allow older versions on the server (and plan to remove that):

```text
# nginx
ssl_protocols TLSv1.2 TLSv1.3;  # add TLSv1.1 only if you must
```

### "certificate verify failed: unable to get local issuer certificate"

```text
verify error:num=20:unable to get local issuer certificate
```

**Cause**: client built a partial chain — it can't find the cert that signed the leaf (or an intermediate). Usually:

- Server didn't send the intermediate(s) in its bundle.
- Client trust store is missing the root.
- Client's CA bundle is empty / wrong path.

**Fix**:

```bash
# Find what the server actually sends:
openssl s_client -connect example.com:443 -showcerts < /dev/null \
  | awk '/-----BEGIN CERTIFICATE-----/,/-----END CERTIFICATE-----/'

# How many certs? Should be 2+ (leaf + intermediate, root often omitted):
openssl s_client -connect example.com:443 -showcerts < /dev/null \
  | grep -c 'BEGIN CERTIFICATE'
```

If only 1 cert returned, server is missing the intermediate. Concatenate the right intermediate (download from CA's site or extract from the cert's AIA extension):

```bash
# Inspect AIA URL:
openssl x509 -in cert.pem -text -noout | grep -A1 'CA Issuers'

# Fetch and append:
curl -s http://ca.example.com/intermediate.crt -o intermediate.pem
cat leaf.pem intermediate.pem > fullchain.pem
# Reconfigure server to use fullchain.pem
```

### "certificate verify failed: certificate has expired"

```text
verify error:num=10:certificate has expired
```

**Cause**: clock check `now > notAfter` failed for some cert in the chain.

**Fix**:

```bash
# Show expiry of all certs in server chain:
openssl s_client -connect example.com:443 -showcerts < /dev/null \
  | openssl x509 -noout -dates

# Or for a local file:
openssl x509 -in cert.pem -noout -enddate

# What's "now" on this box?
date -u
```

If the cert really has expired: renew it.
If the cert hasn't but the box says it has: fix the system clock (usually `chronyd` or `systemd-timesyncd`).

```bash
# macOS
sudo sntp -sS time.apple.com

# Linux
sudo timedatectl set-ntp true
```

### "certificate verify failed: hostname mismatch"

```text
verify error:num=50:hostname mismatch
```

**Cause**: cert is otherwise valid but neither CN nor any SAN matches the hostname the client connected to.

**Fix**:

```bash
# What names does the cert cover?
openssl s_client -connect example.com:443 < /dev/null 2>/dev/null \
  | openssl x509 -noout -subject -ext subjectAltName

# Output:
#   subject= CN = example.com
#   X509v3 Subject Alternative Name:
#       DNS:example.com, DNS:www.example.com
```

Either reissue the cert with the right SANs, or connect to the right hostname. Modern Chrome/Firefox ignore CN entirely — only SAN matters. See SAN vs CN below.

### "certificate verify failed: self signed certificate"

```text
verify error:num=18:self signed certificate
```

**Cause**: leaf cert is self-signed (issuer == subject == leaf), and the leaf is not in the client's trust store.

**Fix**:

- For a real public site: get a proper cert from a public CA.
- For an internal/dev site: either trust the self-signed cert in the client store, or use mkcert / a private CA for development.

```bash
# Trust a self-signed cert (curl):
curl --cacert self-signed.pem https://internal.example.com/

# Or, for dev only:
curl -k https://internal.example.com/

# Or import to system trust store (macOS):
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain self-signed.pem

# Linux (Debian/Ubuntu):
sudo cp self-signed.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

### "certificate verify failed: self signed certificate in certificate chain"

```text
verify error:num=19:self signed certificate in certificate chain
```

**Cause**: somewhere up the chain there's a self-signed cert (a root) that isn't in the client's trust store.

**Fix**: install the root in the client's trust store, or change the chain to be signed by a publicly trusted root.

### "certificate verify failed: unable to verify the first certificate"

```text
verify error:num=21:unable to verify the first certificate
```

**Cause**: same family as #20; client can't connect the leaf to anything trusted. Often means the server only sent the leaf with no intermediates and the client doesn't have them locally.

**Fix**: same as "unable to get local issuer certificate" — fix the server bundle.

### "certificate verify failed: certificate signed by unknown authority"

```text
x509: certificate signed by unknown authority
```

**Cause**: Go's standard library phrasing of "unknown CA" — chain doesn't lead to anything in the trust store.

**Fix**: either trust the CA, or use one that's already trusted.

```go
// Go: explicitly trust a custom CA:
caCert, _ := os.ReadFile("ca.pem")
pool := x509.NewCertPool()
pool.AppendCertsFromPEM(caCert)
config := &tls.Config{RootCAs: pool}
```

### "certificate verify failed: certificate has expired or is not yet valid"

```text
verify error:num=9:certificate is not yet valid
verify error:num=10:certificate has expired
```

**Cause**: cert's validity window doesn't include "now" — either expired or `notBefore` is in the future.

**Fix**: clock or cert. `not yet valid` is almost always a clock issue (newly-minted cert + box with skewed clock).

### "alert handshake failure"

```text
SSL3 alert read:fatal:handshake failure
```

**Cause**: peer rejected the handshake. Usually:

- No shared cipher.
- No cert at all (server sent nothing for SNI it didn't know).
- mTLS: server requires client cert and client offered none.

**Fix**: see `handshake_failure (40)` above; pcap it.

### "tls: handshake failure"

The Go `crypto/tls` flavor of the same message.

```text
remote error: tls: handshake failure
```

**Cause / Fix**: identical to alert 40.

### "tls: oversized record received with length N"

```text
http: TLS handshake error from 10.0.0.1:5432: tls: oversized record received with length 21536
```

**Cause**: someone connected to a TLS port speaking a non-TLS protocol — what arrived didn't parse as a TLS record. The "length" is whatever bytes 4 and 5 of the input happened to be. If you see lengths like ~21536, that's `0x5420` which is `"T "` (HTTP method `GET ` parsed as record length) — a plain HTTP client hit the TLS port.

**Fix**: the client is wrong; redirect HTTP to HTTPS at a different port, or fix the client.

```bash
# Listening for plain HTTP that should be TLS? Reject early:
nginx: listen 443 ssl;
# Add explicit redirect for HTTP:
# listen 80; return 301 https://$host$request_uri;
```

## OpenSSL s_client Output Anatomy

```bash
openssl s_client -connect example.com:443 -servername example.com
```

Output, line by line:

### "CONNECTED(00000003)"

```text
CONNECTED(00000003)
```

TCP-level connect succeeded. The number is the file descriptor. If you don't see this, the failure is at the TCP/network layer, not TLS.

### "depth=N CN=X"

```text
depth=2 C = US, O = ..., CN = ISRG Root X1
depth=1 C = US, O = ..., CN = R3
depth=0 CN = example.com
```

`depth=0` is the leaf. Higher numbers are further from leaf and closer to root. If you only see `depth=0`, the server isn't sending intermediates.

### "verify error:num=NN:"

```text
verify error:num=20:unable to get local issuer certificate
```

This is the trigger for the verify-codes catalog (next section). The numeric code is the load-bearing part — the text is for humans.

### "verify return:1"

```text
verify return:1
```

Confirms verification succeeded for that step. `verify return:0` means failure was tolerated (because we didn't pass `-verify_return_error`).

### "Certificate chain"

```text
Certificate chain
 0 s:CN = example.com
   i:C = US, O = Let's Encrypt, CN = R3
 1 s:C = US, O = Let's Encrypt, CN = R3
   i:C = US, O = Internet Security Research Group, CN = ISRG Root X1
```

`s:` = subject, `i:` = issuer. Read up the chain — each cert's issuer should match the next cert's subject.

### "Server certificate"

```text
-----BEGIN CERTIFICATE-----
MIIE...
-----END CERTIFICATE-----
subject=CN = example.com
issuer=C = US, O = Let's Encrypt, CN = R3
```

The leaf cert in PEM form.

### "subject=" / "issuer="

Distinguished Names of the leaf cert. Also accessible via:

```bash
openssl x509 -in cert.pem -noout -subject -issuer
```

### "SSL handshake has read N bytes and written M bytes"

```text
SSL handshake has read 5417 bytes and written 396 bytes
```

Useful sanity check — if `read` is suspiciously small (<200), the server probably aborted before sending its cert. If `read` is huge (>10kb), there are large intermediates / lots of SCTs.

### "Verification: OK"

```text
Verification: OK
```

The chain validated end-to-end. `Verification error: <text>` means it didn't.

### "Cipher    : TLS_AES_256_GCM_SHA384"

```text
New, TLSv1.3, Cipher is TLS_AES_256_GCM_SHA384
```

The negotiated cipher suite. TLS 1.3 names look like `TLS_<aead>_<hash>`. TLS 1.2 names look like `ECDHE-RSA-AES256-GCM-SHA384`.

### "Server Temp Key: X25519, 253 bits"

```text
Server Temp Key: X25519, 253 bits
```

The ephemeral key exchange group/curve. `X25519` is best; `prime256v1` (NIST P-256) is fine; anything DH-only with bits < 2048 is bad.

### "Peer signing digest: SHA256"

The hash used by the server in its handshake signature. Anything below SHA256 is bad.

### "Peer signature type: RSA-PSS"

```text
Peer signature type: RSA-PSS
```

In TLS 1.3, RSA signatures must use PSS, not the older PKCS#1 v1.5. Older servers that haven't been updated will fail TLS 1.3 here.

### "Session ID:" vs "Session ID-ctx:" vs Session-Ticket

```text
Session-ID: A1B2C3...
Session-ID-ctx:
Session-Ticket: ...
```

- **Session ID** — server-side session cache key; client presents it on resumption.
- **Session ID-ctx** — context binding (which app context the session belongs to).
- **Session-Ticket** — RFC 5077; server-encrypted blob the client stores and presents on resumption (no server-side state).

Empty `Session-ID` + non-empty `Session-Ticket` is normal for ticket-based resumption.

### The `-msg` Flag

Adds a wire-level dump:

```bash
openssl s_client -connect example.com:443 -msg < /dev/null
```

Output includes lines like:

```text
>>> TLS 1.2  [length 0200]
    01 00 01 fc 03 03 ...
<<< TLS 1.2  [length 0061]
    02 00 00 5d 03 03 ...
```

Direction (`>>>` sent, `<<<` received), TLS version, length, and hex bytes. Compare with Wireshark for full decoding.

Other useful debugging flags:

```bash
-debug         # raw bytes hex dump (very noisy)
-state         # state machine transitions
-trace         # decoded handshake messages (best for human reading)
-status        # request OCSP staple
```

## OpenSSL Verify Codes

When the chain check fails, OpenSSL prints `verify error:num=N:<text>` where N is one of the codes below. Use `-verify_return_error` to make `s_client` exit non-zero on verify failure (useful in scripts).

### 0 ok

Verification succeeded.

### 2 unable to get issuer certificate

The cert that issued the cert under inspection couldn't be found, but it's not the top of the chain (the root). This is for missing intermediates.

**Fix**: server should send the missing intermediate.

### 3 unable to get certificate CRL

The CRL of a cert couldn't be found.

**Fix**: usually you don't actually want to fetch CRLs (they're big and slow); turn off CRL checking, or ensure CRL DPs are reachable.

### 4 unable to decrypt certificate's signature

The cert signature can't be decrypted (likely wrong public key for the issuer).

**Fix**: chain is broken — wrong intermediate is being matched.

### 5 unable to decrypt CRL's signature

CRL's signature can't be decrypted. Same logic as 4 but for CRL.

### 6 unable to decode issuer public key

The public key in the SubjectPublicKeyInfo of the issuer cert is unreadable.

**Fix**: corrupt cert; replace.

### 7 certificate signature failure

The signature of the cert doesn't verify against the issuer's public key. Could be wrong issuer cert, corrupted cert, or signature was made with a key different from the issuer's.

### 8 CRL signature failure

CRL signature didn't verify.

### 9 certificate is not yet valid

`now < notBefore`. Check the system clock.

### 10 certificate has expired

`now > notAfter`. Renew the cert.

### 11 CRL is not yet valid

CRL's `thisUpdate` is in the future. Clock issue or freshly-issued CRL not propagated.

### 12 CRL has expired

CRL's `nextUpdate` has passed. CA failed to issue a new one in time.

### 13 format error in certificate's notBefore field

Malformed time in cert. Reissue.

### 14 format error in certificate's notAfter field

Same.

### 15 format error in CRL's lastUpdate field

Same for CRL.

### 16 format error in CRL's nextUpdate field

Same for CRL.

### 17 out of memory

OpenSSL OOM during verify. Restart; investigate memory pressure.

### 18 self signed certificate

Leaf cert is self-signed and is not trusted in the client store.

### 19 self signed certificate in certificate chain

A self-signed cert is somewhere in the chain (a root); it isn't in the client trust store.

### 20 unable to get local issuer certificate

Issuer of the leaf or an intermediate isn't in the trust store and wasn't provided in the chain.

### 21 unable to verify the first certificate

Couldn't verify the first cert in the chain (the leaf). Either no chain was sent or the chain is broken.

### 22 certificate chain too long

Chain length exceeds `verify_depth` (default 100; usually plenty unless a CA has gone wild with cross-signs).

### 23 certificate revoked

CRL or OCSP says this cert is revoked.

### 24 invalid CA certificate

Some prerequisite for a CA cert isn't met (e.g., basicConstraints CA bit not set).

### 25 path length constraint exceeded

A CA cert in the chain set a `pathLenConstraint`; the chain has more intermediates below it than allowed.

### 26 unsupported certificate purpose

Cert's `extendedKeyUsage` or `keyUsage` doesn't permit how it's being used (e.g., a code-signing cert used for TLS server auth).

### 27 certificate not trusted

Cert is in the trust store but isn't explicitly trusted for the relevant purpose.

### 28 certificate rejected

Cert is in the trust store but explicitly marked as rejected.

### 29 subject issuer mismatch

Issuer name in a cert doesn't match the subject name in the cert claimed to be its issuer.

### 30 authority and subject key identifier mismatch

`authorityKeyIdentifier` in the cert doesn't match the `subjectKeyIdentifier` of the candidate issuer.

### 31 authority and issuer serial number mismatch

`authorityKeyIdentifier`'s serial number doesn't match the issuer's serial.

### 32 key usage does not include certificate signing

A CA cert in the chain has a `keyUsage` extension that doesn't include `keyCertSign`.

### 33 unable to get CRL issuer certificate

Couldn't find the cert that issued a CRL.

### 34 unhandled critical extension

Cert has a critical extension that OpenSSL doesn't understand.

### 35 key usage does not include CRL signing

CRL issuer's cert lacks `cRLSign`.

### 36 key usage does not include digital signature

Some cert in the chain lacks `digitalSignature` where it's required (rare).

### 37 signing not permitted by name constraints

`nameConstraints` extension on a CA disallowed signing for this name.

### 38 X509v3 extensions: not yet supported

Some unsupported extension was encountered.

### 39 application verification failure

Custom verify callback failed.

### 40 OCSP verification needed

Trust requires OCSP check.

### 41 OCSP verification failed

OCSP check failed (couldn't reach responder, signature bad, etc.).

### 42 OCSP unknown cert

OCSP says the responder doesn't know the cert.

### 43 invalid Signed Certificate Timestamp (SCT)

Cert's SCT didn't validate.

### 44 unsupported precertificate signing algorithm

Precertificate (CT) used an unsupported sig algorithm.

### 45 OCSP response expired

Stapled OCSP response is past its `nextUpdate`.

### 50 hostname mismatch

Connected hostname doesn't match leaf cert's CN/SAN.

### 51 email address mismatch

Email address from cert doesn't match (S/MIME, not TLS server).

### 52 IP address mismatch

Cert's IP SAN doesn't match the connected IP. Useful for IP-only certs.

```bash
# Verify a chain manually:
openssl verify -CAfile root.pem -untrusted intermediate.pem leaf.pem

# Add -verbose for more detail:
openssl verify -CAfile root.pem -verbose leaf.pem
```

## Browser-Specific Error Strings

### Chrome / Edge / Chromium

#### ERR_CERT_AUTHORITY_INVALID

**Cause**: cert chain doesn't lead to a CA in the system trust store.
**Fix**: install missing root, get cert from trusted CA, or fix server bundle to include intermediates.

#### ERR_CERT_DATE_INVALID

**Cause**: cert is expired or not yet valid.
**Fix**: renew cert; check system clock.

#### ERR_CERT_COMMON_NAME_INVALID

**Cause**: hostname doesn't match leaf cert. Chrome since 58 ignores CN entirely; only SAN matters.
**Fix**: reissue cert with proper SAN entries.

#### ERR_CERT_REVOKED

**Cause**: CRL/OCSP indicates revocation.
**Fix**: get new cert; investigate why old one was revoked.

#### ERR_CERT_WEAK_KEY

**Cause**: cert uses an RSA key < 2048 bits or an EC key on a deprecated curve.
**Fix**: reissue with `RSA-2048` or `ECDSA P-256`.

#### ERR_CERT_WEAK_SIGNATURE_ALGORITHM

**Cause**: cert is signed with SHA-1 or MD5.
**Fix**: reissue with SHA-256 (or SHA-384).

#### ERR_CERT_NAME_CONSTRAINT_VIOLATION

**Cause**: an intermediate CA has `nameConstraints` and the leaf's name violates them. Common with corporate CAs.
**Fix**: get a cert from a CA whose constraints permit the name; or update CA constraints.

#### ERR_CERT_VALIDITY_TOO_LONG

**Cause**: leaf cert's lifetime exceeds CA/Browser Forum maximum (398 days since 2020, soon 90).
**Fix**: reissue with shorter validity.

#### ERR_SSL_PROTOCOL_ERROR

**Cause**: malformed TLS record or unexpected message — could be middlebox interference, server bug, or TLS-incompatible service on the port.
**Fix**: capture pcap, check intercepting proxies, test with `openssl s_client`.

#### ERR_SSL_VERSION_OR_CIPHER_MISMATCH

**Cause**: no overlap between client's offered TLS versions/ciphers and server's. Most often: server is TLS 1.0/1.1 only (deprecated), or only RC4 ciphers, or client has TLS 1.3 disabled and server requires it.
**Fix**: enable TLS 1.2/1.3 and modern ciphers on server; or update client.

#### ERR_TLS_CERTIFICATE_KEY_TYPE_NOT_SUPPORTED

**Cause**: cert uses a key type the client doesn't support (e.g., DSA, certain Ed-curves on older Chromium).
**Fix**: reissue with RSA or ECDSA P-256/P-384.

#### NET::ERR_CERT_DATE_INVALID

The `NET::` prefix shows up on Chrome's interstitial page; same root cause as `ERR_CERT_DATE_INVALID`.

#### NET::ERR_CERT_AUTHORITY_INVALID

Same as `ERR_CERT_AUTHORITY_INVALID` from interstitial.

### Firefox

#### SEC_ERROR_UNKNOWN_ISSUER

**Cause**: Firefox's NSS can't find the issuer in its bundled trust store.
**Fix**: install root in Firefox's certificate manager; or fix server to send intermediates.

#### SEC_ERROR_EXPIRED_CERTIFICATE

**Cause**: cert expired.
**Fix**: renew.

#### MOZILLA_PKIX_ERROR_MITM_DETECTED

**Cause**: Firefox detected a TLS-intercepting product (corporate proxy, AV, etc.). Firefox specifically calls this out because it has its own trust store distinct from the OS.
**Fix**: import the proxy's CA into Firefox; or set `security.enterprise_roots.enabled = true` in `about:config` to use the OS trust store; or stop the interception.

#### SSL_ERROR_BAD_CERT_DOMAIN

**Cause**: hostname mismatch.
**Fix**: reissue cert with right SANs.

#### SSL_ERROR_RX_RECORD_TOO_LONG

**Cause**: same as `tls: oversized record` — connected to a non-TLS port.
**Fix**: confirm port speaks TLS.

#### SSL_ERROR_NO_CYPHER_OVERLAP

**Cause**: no shared cipher.
**Fix**: align cipher suites.

### Safari

#### Cannot Verify Server Identity

Generic Safari error covering most cert validation failures (unknown CA, hostname mismatch, expired). Tap "Details" or click "Show Certificate" for specifics.

**Fix**: investigate which specific issue using `openssl s_client`; same fixes apply.

#### Safari Cannot Open Page

Often shows TLS-related sub-error in the details. Common sub-errors:

- "TLS 1.0 is no longer supported" — upgrade server.
- "The certificate for this server is invalid" — same family as above.

### Edge

Edge is Chromium-based and shares Chrome's `ERR_*` strings exactly. Anything in the Chrome section applies.

## Curl Exit Codes (TLS-related)

`curl --help all | grep -i ssl` lists relevant flags. Exit codes you'll see for TLS:

### 35 SSL connect error

The SSL/TLS handshake failed. Generic — check `-v` for the specific cause.

```bash
curl -v https://example.com 2>&1 | grep -E '(SSL|TLS|certificate)'
```

### 51 The server's SSL/TLS certificate or SSH fingerprint failed verification

Certificate failed pinning check (`--pinnedpubkey` or SSH known_hosts).

### 52 The server didn't reply anything

Often after handshake failure where server tore down the connection silently.

### 56 Failure with receiving network data

Connection closed mid-stream. Could be TLS abort (alert), TCP reset, or timeout.

### 58 Local certificate problem

Client cert (`--cert`) couldn't be loaded.

```bash
# Check format:
openssl x509 -in client.pem -noout -text

# Common fix — convert from PKCS#12:
openssl pkcs12 -in client.p12 -out client.pem -nodes
```

### 59 Couldn't use specified SSL cipher

`--ciphers` value didn't resolve to anything. Use `openssl ciphers -v 'string'` to validate.

### 60 Peer certificate cannot be authenticated with given CA certificates

The most common TLS curl error. Server cert chain doesn't validate against `--cacert` or system trust store.

```bash
# Find current trust store:
curl-config --ca

# Use a specific bundle:
curl --cacert /path/to/bundle.pem https://example.com/

# Skip verification (testing only!):
curl -k https://example.com/
```

### 77 Problem with reading the SSL CA cert

CA cert file (`--cacert`) exists but couldn't be read — permissions or wrong format.

```bash
ls -la $(curl-config --ca)
file $(curl-config --ca)
# Should be PEM (ASCII).
```

### 82 Could not load CRL file

`--crlfile` value can't be parsed.

### 83 Issuer check against peer certificate failed

`--cert-status` or similar check failed.

## OpenSSL Library Error Codes

OpenSSL's C-level reason codes — what shows up in application logs that link against libssl. The string form is `SSL_R_<NAME>`.

### SSL_R_SSL3_GET_SERVER_CERTIFICATE

Pre-OpenSSL-3.0 family error during cert receipt. Subclassed by the verify code (e.g., `:certificate verify failed`).

### SSL_R_NO_CIPHERS_AVAILABLE

The local cipher list is empty after applying configuration. Misconfigured cipher string or restrictive policy.

```bash
# What does my string parse to?
openssl ciphers -v 'HIGH:!aNULL:!MD5'
```

### SSL_R_NO_CIPHER_MATCH

Older form of `NO_SHARED_CIPHER` (OpenSSL 1.0.x and earlier).

### SSL_R_NO_SHARED_CIPHER

Local list and peer's offered list have no overlap. See "no shared cipher" above.

### SSL_R_DECRYPTION_FAILED_OR_BAD_RECORD_MAC

Combined error — record didn't decrypt or MAC didn't verify. Always fatal.

### SSL_R_CERTIFICATE_VERIFY_FAILED

The cert chain didn't validate. The actual reason is in the `verify error:num=N` line.

### SSL_R_TLSV1_ALERT_PROTOCOL_VERSION

Peer sent `protocol_version (70)`. See above.

### SSL_R_TLSV1_ALERT_HANDSHAKE_FAILURE

Peer sent `handshake_failure (40)`. See above.

### SSL_R_WRONG_VERSION_NUMBER

TLS record version byte was nonsense. See "wrong version number" above.

### SSL_R_UNEXPECTED_RECORD

Got a TLS record we didn't expect for the current state machine state.

### SSL_R_INAPPROPRIATE_FALLBACK

Server detected client's `TLS_FALLBACK_SCSV` and rejected the downgrade.

### SSL_R_LENGTH_MISMATCH

A length field in a handshake message disagreed with the actual content length.

### SSL_R_BAD_PROTOCOL_VERSION_NUMBER

The advertised version doesn't match what was negotiated. Buggy peer or middlebox.

## Common Misconfigurations and Their Symptoms

### SNI not set → wrong cert returned

```bash
# BROKEN:
openssl s_client -connect 1.2.3.4:443
# Returns whatever the server chose as default — likely wrong vhost.

# FIXED:
openssl s_client -connect 1.2.3.4:443 -servername example.com
```

Symptom: hostname mismatch error even though the cert exists on the server.

### Intermediate cert missing from server bundle

```text
verify error:num=20:unable to get local issuer certificate
```

Symptom: works in Firefox (which caches intermediates from prior visits) but fails in curl/openssl/some browsers.

```bash
# BROKEN (only leaf):
ssl_certificate /etc/ssl/example.com.crt;

# FIXED (leaf + intermediate):
ssl_certificate /etc/ssl/fullchain.pem;
```

```bash
# Build fullchain:
cat leaf.pem intermediate.pem > fullchain.pem
```

### Mismatched CN/SAN

```text
verify error:num=50:hostname mismatch
```

Symptom: cert is for `example.com` but you connected as `www.example.com` (or vice versa).

```bash
# Inspect what names are covered:
openssl x509 -in cert.pem -noout -ext subjectAltName
```

### Non-trusted root

```text
unable to get local issuer certificate
self signed certificate in certificate chain
```

Symptom: works on a colleague's machine, fails on yours. Different trust stores.

### ChaCha20 disabled when client only supports it

```text
no shared cipher
```

Mostly historical — modern OpenSSL has ChaCha20 enabled by default. Affects locked-down profiles.

### TLS 1.0/1.1 disabled when client requires it

```text
SSL_R_TLSV1_ALERT_PROTOCOL_VERSION
```

Common when modernizing a server with embedded device clients in the field.

### DH key too short (< 2048 bits)

```text
dh key too small
```

Modern clients reject DH parameters under 2048 bits (Logjam mitigation).

```bash
# Generate strong DH params:
openssl dhparam -out dhparams.pem 2048

# In nginx:
ssl_dhparam /etc/ssl/dhparams.pem;
```

Better: avoid plain DHE; use ECDHE.

### SHA-1 cert signature

```text
ERR_CERT_WEAK_SIGNATURE_ALGORITHM
```

Modern clients reject SHA-1-signed leaves and intermediates. Reissue with SHA-256.

### Cert with no extendedKeyUsage

```text
unsupported certificate purpose
```

Some strict clients require `serverAuth` EKU explicitly.

```bash
# Check EKU:
openssl x509 -in cert.pem -noout -text | grep -A2 'Extended Key Usage'
```

### Self-signed without CA chain bundle

Symptom: works locally with `-k` / `--insecure` but fails everywhere else without explicit trust.

```bash
# Production-like fix: stand up a private CA, sign certs from it,
# and distribute the CA's root to clients.
# Or use mkcert (see below).
```

## Cert Chain Construction

Order matters in concatenated PEM bundles.

```text
[ Leaf cert       ]   <-- closest to client connection
[ Intermediate 1  ]
[ Intermediate 2  ]   (if cross-signed or multi-tier)
                      (root usually NOT included; client trusts it)
```

Server SHOULD send leaf + all intermediates needed to reach a trusted root, but NOT the root itself (clients have it; sending it just wastes bytes).

### Reorder Gotcha

```bash
# BROKEN order:
cat intermediate.pem leaf.pem > bundle.pem

# CORRECT order:
cat leaf.pem intermediate.pem > bundle.pem
```

Many TLS implementations are strict; some are forgiving. Don't rely on forgiveness.

### `-CAfile` vs `-CApath`

```bash
# Single file with concatenated trusted roots:
openssl verify -CAfile /etc/ssl/cert.pem leaf.pem

# Directory of hashed-named cert files (for c_rehash):
openssl verify -CApath /etc/ssl/certs leaf.pem

# Build the hashed names:
c_rehash /etc/ssl/certs
```

### Inspecting a Chain

```bash
# Extract all certs from a PEM bundle:
openssl crl2pkcs7 -nocrl -certfile bundle.pem | openssl pkcs7 -print_certs -text -noout

# Or for a P7B/PKCS#7 file:
openssl pkcs7 -in chain.p7b -inform DER -print_certs -text -noout

# Walk the chain manually:
openssl s_client -connect example.com:443 -showcerts < /dev/null \
  | awk 'BEGIN{i=0} /BEGIN CERT/{i++} {print > "cert_"i".pem"}'

# Then inspect each:
for f in cert_*.pem; do echo "=== $f ==="; openssl x509 -in $f -noout -subject -issuer; done
```

## SAN vs CN

Modern Chrome (since 58, 2017) requires the leaf cert to have a Subject Alternative Name (SAN) extension matching the hostname. The legacy CN field is ignored entirely.

```text
ERR_CERT_COMMON_NAME_INVALID
```

This error usually means: the cert has the right name in CN but no SAN at all.

### Verify SAN

```bash
openssl x509 -in cert.pem -text -noout | grep -A1 'Subject Alternative Name'
```

Output:

```text
X509v3 Subject Alternative Name:
    DNS:example.com, DNS:www.example.com
```

If you don't see the section at all, the cert has no SAN.

### Issuing a Cert With SAN

```bash
# CSR config (san.cnf):
cat > san.cnf <<'EOF'
[req]
distinguished_name = req_dn
req_extensions = req_ext
prompt = no

[req_dn]
CN = example.com

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = example.com
DNS.2 = www.example.com
EOF

openssl req -new -key key.pem -out csr.pem -config san.cnf
```

When signing, ensure SAN is preserved:

```bash
openssl x509 -req -in csr.pem -CA ca.pem -CAkey ca.key \
    -CAcreateserial -out cert.pem -days 365 \
    -extensions req_ext -extfile san.cnf
```

## SNI Workflow

```bash
# Test a specific vhost on a multi-host server:
openssl s_client -connect 1.2.3.4:443 -servername example.com

# Without -servername you'll get the default cert,
# which often produces hostname mismatch errors.
```

### The "no shared cipher" Gotcha When -servername Is Wrong

If the server is configured to reject unknown SNI values entirely, you'll get a `handshake_failure` or `unrecognized_name` rather than a mismatched cert.

```bash
openssl s_client -connect 1.2.3.4:443 -servername wrong.example.com
# tlsv1 unrecognized name
```

### Wildcard Cert SNI Behavior

`*.example.com` covers `foo.example.com` but NOT `foo.bar.example.com` (single label only) and NOT `example.com` (the apex needs its own SAN).

### SNI-Aware vs SNI-Blind Clients

Older clients (Java 6, IE on XP) don't send SNI. Modern servers handle this by sending a default cert; if your server is configured to require SNI, those old clients break.

```bash
# Test SNI-blind behavior:
openssl s_client -connect host:443 -noservername
```

## ALPN

```bash
# Test HTTP/2 support:
openssl s_client -connect example.com:443 -alpn h2

# Output line:
# ALPN protocol: h2
```

If the server doesn't speak HTTP/2 it returns no ALPN selection (some) or sends `no_application_protocol` (strict implementations).

```bash
# Multiple options, in preference order:
openssl s_client -connect example.com:443 -alpn h2,http/1.1

# Test HTTP/3 (QUIC, not TCP — won't work directly via s_client):
# Use a QUIC-aware tool like quiche-client or curl built with QUIC.
```

The `no_application_protocol` alert (alert 120) fires when client offers only protocols server doesn't support. Adding `http/1.1` as fallback nearly always fixes it.

## TLS Renegotiation Attacks

In 2009, Marsh Ray showed that an attacker could splice plaintext into the start of a TLS session by abusing the renegotiation feature (CVE-2009-3555). The fix is the **Secure Renegotiation extension** (RFC 5746): both peers attach the previous handshake's "verify_data" to subsequent renegotiations, binding old and new sessions cryptographically.

Modern OpenSSL refuses to renegotiate without this extension.

```text
SSL: error:140940E5:SSL routines:ssl3_read_bytes:
ssl handshake failure (reason 1040: insufficient security)
```

Or sometimes:

```text
unsafe legacy renegotiation disabled
```

**Cause**: connecting to an old server that doesn't support secure renegotiation.

**Fix**:

- Best: upgrade the server.
- If you must connect: pass `-legacy_renegotiation` (s_client) or set `SSL_OP_LEGACY_SERVER_CONNECT` (in code). NEVER do this on the public internet.

```bash
openssl s_client -connect old-server.internal:443 -legacy_renegotiation
```

```text
# OpenSSL 3.x config to allow legacy renegotiation globally (do not do this):
[openssl_init]
ssl_conf = ssl_sect

[ssl_sect]
system_default = ssl_default_sect

[ssl_default_sect]
Options = UnsafeLegacyRenegotiation
```

## OCSP & OCSP Stapling

OCSP (Online Certificate Status Protocol, RFC 6960) lets a client ask "is this cert revoked?" by querying the URL in the cert's AIA extension.

```bash
# Find the OCSP responder URL:
openssl x509 -in cert.pem -noout -ocsp_uri
# Example output: http://ocsp.int-x3.letsencrypt.org

# Or:
openssl x509 -in cert.pem -noout -text | grep 'OCSP - URI'
```

### Manual OCSP Query

```bash
# Prepare:
ISSUER=intermediate.pem
CERT=leaf.pem
URL=$(openssl x509 -in $CERT -noout -ocsp_uri)

# Query:
openssl ocsp -issuer $ISSUER -cert $CERT -url $URL -resp_text -noverify
```

Output includes the cert status: `good`, `revoked`, or `unknown`.

### OCSP Stapling

The server fetches the OCSP response itself and includes it in the TLS handshake. Saves the client a round trip and a privacy leak (CA learns who's visiting).

```bash
# Check if a server staples:
openssl s_client -connect example.com:443 -status < /dev/null 2>&1 | grep -A20 'OCSP'
```

If stapled, you'll see:

```text
OCSP Response Data:
    OCSP Response Status: successful (0x0)
    Response Type: Basic OCSP Response
    Version: 1 (0x0)
    ...
    Cert Status: good
```

### MUST_STAPLE

A cert can carry the `id-pe-tlsfeature` extension with value `status_request` — meaning clients MUST require a stapled OCSP response. If the server doesn't staple, modern clients refuse the connection.

```bash
# Check for MUST_STAPLE:
openssl x509 -in cert.pem -noout -text | grep -A1 '1.3.6.1.5.5.7.1.24'
```

### Common OCSP Errors

#### "OCSP responder unreachable"

The CA's OCSP server isn't responding. If using soft-fail OCSP (most clients), the connection succeeds but OCSP isn't checked. With MUST_STAPLE, connection fails.

#### "OCSP signature verify failed"

Stapled response was signed by a key that doesn't match what the client expects.

### Soft-Fail vs Hard-Fail

- **Soft-fail (default)**: if OCSP can't be checked, the cert is accepted. Browsers do this; it's why OCSP is criticized as security theater.
- **Hard-fail**: any OCSP failure rejects the connection. Available with MUST_STAPLE.

## CRL (Certificate Revocation List)

CRLs are signed lists of revoked cert serial numbers, published periodically by CAs. The cert's `crlDistributionPoints` extension says where to fetch them.

```bash
# Find CRL URL:
openssl x509 -in cert.pem -noout -text | grep -A4 'CRL Distribution'

# Fetch and decode:
curl -o crl.der http://crl.example.com/intermediate.crl
openssl crl -inform DER -in crl.der -text -noout
```

### Why CRLs Are Slow

- Single CRL can be MBs (every revoked cert from that CA, ever, until expiry).
- Cached CRL goes stale.
- Most clients skip CRL checking entirely.

### CRLite

Mozilla's CRLite uses bloom filters to ship a compressed view of all revocations to clients. Eliminates the network round trip and the privacy leak. Currently shipped in Firefox.

### "unable to get certificate CRL"

```text
verify error:num=3:unable to get certificate CRL
```

**Cause**: CRL checking enabled but the CRL DP is unreachable, or the file is missing.

**Fix**: usually disable CRL checking entirely (it's not what protects you in 2024); rely on OCSP / cert lifetime / revocation lists baked into clients.

## Certificate Transparency

CT (RFC 6962) requires CAs to log every issued cert in publicly auditable Merkle-tree logs. Each cert carries one or more **Signed Certificate Timestamps (SCTs)** as evidence it was logged.

Chrome enforces CT for all publicly-trusted certs since April 2018: certs without sufficient SCTs are rejected with:

```text
ERR_CERTIFICATE_TRANSPARENCY_REQUIRED
```

### Inspect SCTs

```bash
openssl x509 -in cert.pem -text -noout | grep -A5 'CT Precertificate'

# Or use ctutil / ct-honest:
ct-honest --cert cert.pem
```

### Searching crt.sh

```bash
# Find all certs issued for a domain:
curl -s "https://crt.sh/?q=example.com&output=json" | jq '.[].name_value' | sort -u
```

Use this to detect unauthorized issuance — if you see a cert for your domain you didn't request, that's a problem.

### When You'll See ERR_CERTIFICATE_TRANSPARENCY_REQUIRED

- Internal CA whose intermediates aren't in Chrome's "CT-not-required" list.
- Misconfigured CA that didn't include SCTs.
- Cert older than April 2018 that was reissued from a CA that doesn't log.

**Fix**: get a cert from a properly logging CA (any reputable public CA today); for internal CAs, configure `EnableEnterpriseClientCT` policy if appropriate.

## Let's Encrypt / ACME Common Errors

Let's Encrypt (and other ACME providers) return JSON Problem Details on errors. The `type` field is the canonical error code.

### urn:ietf:params:acme:error:badNonce

**Cause**: client used an expired or already-used nonce, or its clock is skewed too far from the ACME server.

**Fix**: sync system clock; ensure the ACME client retrieves a fresh nonce per request:

```bash
sudo timedatectl set-ntp true
```

### urn:ietf:params:acme:error:rateLimited

**Cause**: hit a rate limit (50 certs/registered domain/week is the most-hit one for Let's Encrypt).

**Fix**: use the staging environment for testing:

```bash
certbot certonly --staging -d example.com
```

The staging API has its own much-higher limits and produces non-trusted certs perfect for testing.

### urn:ietf:params:acme:error:rejectedIdentifier

**Cause**: domain is on a block list (high-risk TLD, recently abused, etc.).

**Fix**: contact the CA; consider a different domain.

### urn:ietf:params:acme:error:unauthorized

**Cause**: the challenge response was wrong — server didn't show the right token, or the token URL didn't match what was provisioned.

**Fix**: verify the challenge file is reachable:

```bash
# For HTTP-01:
curl http://example.com/.well-known/acme-challenge/<token>
# Should return the expected key authorization (no redirects, no auth).
```

### "Unable to fulfill HTTP-01 challenge"

**Cause**: the ACME server tried to fetch `http://example.com/.well-known/acme-challenge/<token>` and failed. Reasons:

- Web root doesn't include `/.well-known/acme-challenge/`.
- Web server returns 4xx/5xx for that path.
- DNS doesn't resolve.
- Firewall blocks port 80 from outside.

**Fix**:

```bash
# Test from outside (use a third-party tool or another network):
curl -v http://example.com/.well-known/acme-challenge/test
```

Make sure port 80 is open even if your site is HTTPS-only — Let's Encrypt's HTTP-01 challenge uses port 80.

### "DNS problem: NXDOMAIN looking up X"

**Cause**: domain doesn't resolve at all (or DNS hasn't propagated yet).

**Fix**: wait for propagation; verify with:

```bash
dig example.com
# Should return an A or AAAA record.
```

### Certbot Common Arguments and Errors

```bash
# Standard request:
certbot certonly --standalone -d example.com -d www.example.com

# Webroot mode (don't stop the web server):
certbot certonly --webroot -w /var/www/html -d example.com

# DNS challenge (works behind firewalls / for wildcard):
certbot certonly --manual --preferred-challenges dns -d '*.example.com'

# Renew (run from cron):
certbot renew --quiet

# Common errors:
# - "Could not bind to IPv4 or IPv6"  --> port 80 in use
# - "Failed authorization procedure"  --> challenge verification failed
# - "An unexpected error occurred: HTTPSConnectionPool"  --> ACME API unreachable
# - "Too many failed authorizations recently"  --> rate-limited; wait an hour
```

## Mutual TLS (mTLS)

In mTLS, both client and server present certs. Server's `CertificateRequest` triggers the client to send its cert.

### "no client certificate sent"

```text
SSL3_READ_BYTES: tlsv13 alert certificate required
```

Or, in TLS 1.2:

```text
SSL3_READ_BYTES: peer did not return a certificate
```

**Cause**: server requires client cert (`SSL_VERIFY_PEER` + `SSL_VERIFY_FAIL_IF_NO_PEER_CERT`); client didn't send one.

**Fix**: configure the client:

```bash
curl --cert client.pem --key client.key https://api.example.com/

# Or with a P12:
curl --cert client.p12 --cert-type P12 https://api.example.com/

# OpenSSL:
openssl s_client -cert client.pem -key client.key -connect api.example.com:443
```

### "wrong client certificate"

**Cause**: client presented a cert, but it doesn't validate against the server's client-CA trust store.

**Fix**: ensure the client cert is signed by a CA the server expects. The server's *client-CA* bundle is separate from its server-cert CA bundle — don't confuse them.

```text
# nginx: separate CA bundles for server cert vs client verification:
ssl_certificate         /etc/ssl/server-cert.pem;       # server's own cert
ssl_certificate_key     /etc/ssl/server-key.pem;
ssl_client_certificate  /etc/ssl/client-ca-bundle.pem;  # CAs we accept for clients
ssl_verify_client       on;
```

### Client Cert in OpenSSL

```bash
openssl s_client \
  -connect api.example.com:443 \
  -cert client.pem \
  -key client.key \
  -CAfile server-ca.pem
```

Add `-verify_return_error` to make verification errors fatal in scripts.

### Different CA Bundles

The server's "trust store for client certs" is a different bundle from the system trust store. They serve different purposes:

- **System trust store (e.g., `/etc/ssl/cert.pem`)** — what *clients* use to verify *server* certs.
- **Client CA bundle (e.g., `ssl_client_certificate`)** — what *server* uses to verify *client* certs.

## Cipher Suite Selection

```bash
# What does my cipher string parse to?
openssl ciphers -v 'HIGH:!aNULL:!MD5'

# Output columns:
# name | TLS version | Kx (key exchange) | Au (auth) | Enc | Mac
# Example:
# ECDHE-RSA-AES256-GCM-SHA384 TLSv1.2 Kx=ECDH Au=RSA Enc=AESGCM(256) Mac=AEAD
```

### TLS 1.3 Fixed Ciphers

TLS 1.3 only supports five cipher suites (versus dozens in 1.2):

- **TLS_AES_128_GCM_SHA256** — most common
- **TLS_AES_256_GCM_SHA384** — strongest AES
- **TLS_CHACHA20_POLY1305_SHA256** — best on devices without AES-NI (mobile)
- **TLS_AES_128_CCM_SHA256** — rare
- **TLS_AES_128_CCM_8_SHA256** — rare; for constrained devices

In TLS 1.3, the cipher selects only the AEAD; the key exchange (always ECDHE) and auth (signature) algorithm are negotiated separately via extensions.

### TLS 1.2 Cipher Naming

```text
ECDHE-RSA-AES128-GCM-SHA256
^ Kx ^ Au  ^ Enc      ^ Mac/PRF
```

- **Kx** — key exchange (`ECDHE`, `DHE`, `RSA`, `PSK`).
- **Au** — authentication (`RSA`, `ECDSA`, `DSS`).
- **Enc** — bulk cipher (`AES128`, `AES256`, `CHACHA20`, etc., with mode `GCM`, `CCM`, `CBC`).
- **Mac/PRF** — for AEAD (GCM/CCM), this is the SHA used as the PRF; for CBC modes, the actual MAC.

### Modern Recommendations

Mozilla SSL Config Generator levels:

- **Modern** (TLS 1.3 only): only TLS 1.3 fixed suites. Refuses any client < TLS 1.3.
- **Intermediate** (TLS 1.2+, default): broad compatibility while excluding weak crypto.
- **Old** (TLS 1.0+): for legacy clients only — explicitly insecure.

```text
# Mozilla intermediate cipher list (nginx):
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers off;
```

Get up-to-date config at https://ssl-config.mozilla.org

## Tools

### openssl s_client

The Swiss Army knife. Most useful flags:

```bash
# Basic:
openssl s_client -connect host:443

# With SNI (do this when the host has multiple vhosts):
openssl s_client -connect host:443 -servername example.com

# Show full cert chain the server sends:
openssl s_client -connect host:443 -showcerts

# Force a specific TLS version:
openssl s_client -connect host:443 -tls1_3
openssl s_client -connect host:443 -tls1_2
openssl s_client -connect host:443 -tls1_1   # if compiled in
openssl s_client -connect host:443 -tls1     # ditto

# Test ALPN:
openssl s_client -connect host:443 -alpn h2,http/1.1

# Request OCSP staple:
openssl s_client -connect host:443 -status

# Use a specific cipher list:
openssl s_client -connect host:443 -cipher 'ECDHE+AESGCM'

# mTLS:
openssl s_client -connect host:443 -cert client.pem -key client.key

# Don't read stdin (so it terminates after handshake):
echo Q | openssl s_client -connect host:443
# Or:
openssl s_client -connect host:443 < /dev/null

# Dump every protocol-level message:
openssl s_client -connect host:443 -msg

# Show TLS state machine transitions:
openssl s_client -connect host:443 -state

# Pretty-print the handshake:
openssl s_client -connect host:443 -trace

# Make verify errors fatal in scripts:
openssl s_client -connect host:443 -verify_return_error
```

### openssl x509

Inspect and manipulate certs:

```bash
# Print human-readable form:
openssl x509 -in cert.pem -text -noout

# Just the dates:
openssl x509 -in cert.pem -noout -dates

# Just subject and issuer:
openssl x509 -in cert.pem -noout -subject -issuer

# Specific extension:
openssl x509 -in cert.pem -noout -ext subjectAltName

# Fingerprint (SHA-256):
openssl x509 -in cert.pem -noout -fingerprint -sha256

# Convert DER to PEM:
openssl x509 -in cert.der -inform DER -out cert.pem -outform PEM

# Convert PEM to DER:
openssl x509 -in cert.pem -outform DER -out cert.der

# Extract just the public key:
openssl x509 -in cert.pem -pubkey -noout
```

### openssl verify

Standalone chain verification:

```bash
# Against system trust:
openssl verify cert.pem

# With explicit CA bundle:
openssl verify -CAfile bundle.pem cert.pem

# With intermediate file:
openssl verify -CAfile root.pem -untrusted intermediate.pem leaf.pem

# Verbose:
openssl verify -verbose -CAfile bundle.pem cert.pem
```

### sslyze (Python)

Batch testing tool. Faster and friendlier than `s_client`:

```bash
pip install sslyze

sslyze example.com
# Tests SSLv2/3, TLS 1.0/1.1/1.2/1.3, cert info, weak ciphers,
# heartbleed, robot attack, session resumption, etc.

sslyze --regular example.com
sslyze --json_out=results.json example.com
sslyze --tlsv1_3 example.com
sslyze --certinfo example.com
```

### testssl.sh

Pure-bash, requires only OpenSSL:

```bash
git clone https://github.com/drwetter/testssl.sh.git
./testssl.sh example.com

# Specific tests:
./testssl.sh --protocols example.com
./testssl.sh --ciphers example.com
./testssl.sh --vulnerable example.com   # heartbleed/POODLE/etc.

# JSON output:
./testssl.sh --jsonfile results.json example.com
```

### Qualys SSL Labs

Web-based, public sites only: https://www.ssllabs.com/ssltest/

Gives you the famous A+/A/B/F grade, plus a full breakdown of cipher suites, browser compatibility, certificate transparency, and known-vulnerability checks. Don't use for internal hostnames (they have to be reachable from Qualys's network).

### mkcert

Local CA + dev certs without the manual openssl dance:

```bash
# Install (macOS):
brew install mkcert

# One-time: install local CA in system + browser trust stores:
mkcert -install

# Generate a cert for localhost + custom host:
mkcert localhost 127.0.0.1 myapp.local '*.myapp.local'
# Outputs ./localhost+3.pem and ./localhost+3-key.pem

# Use in your dev server config — already trusted by browsers!
```

The "gotcha" with mkcert is that the local CA root is not in Firefox's trust store unless you've started Firefox at least once before running `mkcert -install`. Re-run `mkcert -install` after first Firefox launch if you see Firefox-only failures.

### certbot / lego / acme.sh

ACME clients for Let's Encrypt:

```bash
# certbot (the official Python client):
sudo certbot certonly --standalone -d example.com

# lego (Go, single binary):
lego --email you@example.com --domains example.com --http run

# acme.sh (pure bash, very portable):
acme.sh --issue -d example.com -w /var/www/html
```

`acme.sh` is great for systems that don't have Python; `lego` is great for embedding into deployment automation; `certbot` is the most-documented.

## Common Gotchas

### System time wrong → certs appear expired

```bash
# BROKEN — clock is in 2030:
$ date
Sun Mar 17 12:34:56 UTC 2030
$ curl https://example.com/
curl: (60) certificate has expired

# FIXED:
sudo timedatectl set-ntp true
sudo systemctl restart systemd-timesyncd
date
```

### SNI not sent → wrong cert returned

```bash
# BROKEN:
openssl s_client -connect 1.2.3.4:443 < /dev/null \
  | openssl x509 -noout -subject
# subject= CN = default.example.com   <- not what we wanted

# FIXED:
openssl s_client -connect 1.2.3.4:443 -servername api.example.com < /dev/null \
  | openssl x509 -noout -subject
# subject= CN = api.example.com
```

### Missing intermediate in server bundle → unable to verify

```bash
# BROKEN — only leaf served:
$ openssl s_client -connect example.com:443 -showcerts < /dev/null \
    | grep -c 'BEGIN CERT'
1

# FIXED — concat leaf + intermediate:
$ cat leaf.pem intermediate.pem > fullchain.pem
# Update server config to use fullchain.pem
$ openssl s_client -connect example.com:443 -showcerts < /dev/null \
    | grep -c 'BEGIN CERT'
2
```

### Renewed cert but didn't reload nginx/Apache → still serving old

```bash
# BROKEN — file on disk is new but server is still using old:
$ openssl x509 -in /etc/letsencrypt/live/example.com/fullchain.pem -noout -enddate
notAfter=Jul 15 12:00:00 2026 GMT
$ openssl s_client -connect example.com:443 < /dev/null 2>/dev/null \
    | openssl x509 -noout -enddate
notAfter=Apr 16 12:00:00 2026 GMT   # <- older!

# FIXED:
sudo systemctl reload nginx
# Or:
sudo nginx -t && sudo nginx -s reload
```

### Used HTTP redirect → stuck in HTTP-only after deploy

```bash
# BROKEN — server config:
server { listen 80; return 301 http://example.com$request_uri; }
# This redirects HTTP to HTTP, not HTTPS. Loop avoided only because
# 301 is cached and clients give up.

# FIXED:
server { listen 80; return 301 https://$host$request_uri; }
```

### Mixed TLS versions in cluster → some endpoints fail

```bash
# BROKEN — load balancer round-robins to two backends, one upgraded one not:
$ for i in 1 2 3 4; do
    openssl s_client -connect example.com:443 < /dev/null 2>&1 \
      | grep 'Protocol'
  done
Protocol  : TLSv1.3
Protocol  : TLSv1.2
Protocol  : TLSv1.3
Protocol  : TLSv1.2

# FIXED: roll out same TLS config across the entire fleet, or
# terminate TLS at the LB and use plain TCP to the backends.
```

### Client uses self-signed cert without --insecure flag → connect fails

```bash
# BROKEN:
$ curl https://internal.example.com/
curl: (60) SSL certificate problem: self-signed certificate

# FIXED — proper way (trust the CA):
$ curl --cacert internal-ca.pem https://internal.example.com/

# WRONG-BUT-WORKS (testing only):
$ curl -k https://internal.example.com/
```

### Certificate Pinning broke after rotation → app stuck on old cert

```bash
# BROKEN — app pins SHA-256 of the leaf cert. Cert rotated, pin not updated:
# Mobile app hardcodes:
# pin = "sha256/XXXXX..."
# After rotation, every connection fails with "pin verification failed".

# FIXED — pin to the public key (SPKI), not the cert.
# The public key survives cert rotation if you reuse the keypair.
$ openssl x509 -in cert.pem -pubkey -noout \
    | openssl pkey -pubin -outform der \
    | openssl dgst -sha256 -binary \
    | openssl enc -base64
# Use this hash as the pin.

# Better still: include a backup pin (the *next* key you'll rotate to)
# so you can rotate without app updates.
```

### HSTS preload submitted then can't roll back

HSTS (HTTP Strict Transport Security) tells browsers "always use HTTPS for this domain." Preload baking submits your domain into Chrome/Firefox source, where it can take *months* to remove.

```bash
# BROKEN — HSTS preload was submitted with includeSubDomains during testing.
# Now subdomain test sites can't be reached over HTTP.

# FIXED — submit removal at https://hstspreload.org/removal/
# Wait 6-12 weeks for browsers to roll out the new list.
# Until then, every subdomain MUST have a working HTTPS endpoint.
```

Don't preload until you're certain. The preload header looks like:

```text
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
```

### HPKP (deprecated) bricked the site

HTTP Public Key Pinning was a header that pinned cert keys at the browser. If you rotated to a non-pinned key, every prior visitor's browser refused the new cert until the pin expired.

Removed from Chrome in 2018. Don't use it. If you somehow still have an `Public-Key-Pins` header:

```bash
# Remove:
# DELETE the header from your config.
# Then wait for max-age (originally set when you sent it) to elapse.
# There's no manual override.
```

### mTLS server expects client cert but client never sends → handshake_failure

```bash
# BROKEN:
$ curl https://api.example.com/
curl: (35) error:0A000172:SSL routines::ssl/tls alert handshake failure

# Server log:
# error:0A000418:SSL routines::tlsv1 alert no certificate

# FIXED:
$ curl --cert client.pem --key client.key https://api.example.com/
```

### Let's Encrypt rate limit (50/week per registered domain) hit during testing

```bash
# BROKEN — rapid iteration on certbot config blew through the limit:
urn:ietf:params:acme:error:rateLimited:
Error creating new order :: too many certificates already issued
for "example.com": see https://letsencrypt.org/docs/rate-limits/

# FIXED — switch to staging for development:
$ certbot certonly --staging -d example.com
# Staging has the same code path but with much higher limits and
# untrusted certs (trust the staging CA in your test env if needed).
```

## mkcert / Dev-Cert Workflow

For local development you almost certainly want `mkcert`. It:

1. Generates a local CA (root cert + key) on first run.
2. Installs that root into the OS trust store and major browsers.
3. Issues short-lived dev certs signed by the local CA.

```bash
# One-time setup:
brew install mkcert nss   # nss provides certutil for Firefox
mkcert -install

# Make a cert for typical dev hostnames:
cd ~/dev/myapp
mkcert localhost 127.0.0.1 ::1 myapp.local '*.myapp.local'
# Outputs:
#   localhost+4.pem
#   localhost+4-key.pem

# In your server (Go example):
http.ListenAndServeTLS(":8443", "localhost+4.pem", "localhost+4-key.pem", handler)
```

### Dev-Cert Gotchas

- **Firefox not trusting the cert** — the local CA needs to be in Firefox's NSS DB. `mkcert -install` does this if Firefox is installed AND `nss` (the certutil tool) is available. Run `mkcert -install` again after installing Firefox or `nss`.
- **Docker not trusting the cert** — copy the local CA root into the container:

```bash
docker cp "$(mkcert -CAROOT)/rootCA.pem" mycontainer:/usr/local/share/ca-certificates/
docker exec mycontainer update-ca-certificates
```

- **CI not trusting the cert** — same thing for CI runners; or use the staging CA approach (see above).
- **Cert appears valid in browser but `curl` fails** — `curl` uses the system OpenSSL trust store, which may differ from the system keychain. Add the local CA via `--cacert` or the `SSL_CERT_FILE` env var:

```bash
SSL_CERT_FILE="$(mkcert -CAROOT)/rootCA.pem" curl https://localhost:8443/
```

## Idioms

### Always specify -servername with openssl s_client when testing SNI hosts

```bash
# Default to including -servername — it's almost always correct:
openssl s_client -connect 1.2.3.4:443 -servername example.com
```

The legacy "you don't need it for IP-based hosts" advice is out of date; modern servers configured for SNI may not even respond without it.

### Validate full cert chain in CI

Every CI pipeline that produces a deployable artifact should include a TLS check before the deploy stage:

```bash
#!/usr/bin/env bash
set -euo pipefail
HOST=$1

# Hostname matches:
openssl s_client -connect $HOST:443 -servername $HOST -verify_return_error \
  < /dev/null > /dev/null

# Doesn't expire in next 30 days:
END_DATE=$(openssl s_client -connect $HOST:443 -servername $HOST < /dev/null 2>/dev/null \
  | openssl x509 -noout -enddate | cut -d= -f2)
END_EPOCH=$(date -d "$END_DATE" +%s 2>/dev/null || gdate -d "$END_DATE" +%s)
NOW_EPOCH=$(date +%s)
DAYS_LEFT=$(( (END_EPOCH - NOW_EPOCH) / 86400 ))
echo "Cert valid for $DAYS_LEFT more days"
[[ $DAYS_LEFT -gt 30 ]] || { echo "Cert expires in $DAYS_LEFT days"; exit 1; }

# TLS 1.2 minimum:
openssl s_client -connect $HOST:443 -servername $HOST -tls1_2 \
  < /dev/null > /dev/null
```

### Pin to public key (SPKI), not cert

Cert rotation breaks cert pinning unless you reissue with the same key. Public key (SPKI = SubjectPublicKeyInfo) pinning survives rotation if you keep the key.

```bash
# Compute SPKI hash:
openssl x509 -in cert.pem -pubkey -noout \
  | openssl pkey -pubin -outform der \
  | openssl dgst -sha256 -binary \
  | base64

# Use this in HTTP headers, mobile apps, IoT clients.
```

Always include a **backup pin** for the next rotation key.

### Use Let's Encrypt for public; private CA for internal

- **Public sites**: Let's Encrypt (or another ACME-supporting CA). Free, automated, ubiquitous trust.
- **Internal sites**: a private CA (`step certificates`, HashiCorp Vault PKI, AWS Private CA, smallstep, mkcert for dev). Why? Internal hostnames can't be validated by ACME (no public DNS), and you don't want internal hostnames in public CT logs.

Don't use a self-signed leaf for an internal site that more than two people use — it's a configuration headache that scales badly.

### Set --tls-min-version=1.2 always; 1.3 if all clients support

```bash
# In nginx:
ssl_protocols TLSv1.2 TLSv1.3;

# In Apache:
SSLProtocol TLSv1.2 +TLSv1.3

# Go application:
config := &tls.Config{
    MinVersion: tls.VersionTLS12,
    // For 1.3-only:
    // MinVersion: tls.VersionTLS13,
}

# curl:
curl --tls-max 1.3 --tlsv1.2 https://example.com/
```

TLS 1.0 and 1.1 are formally deprecated (RFC 8996, March 2021). Don't enable them unless you have an explicit, time-bounded reason.

## See Also

- `tls` — TLS protocol fundamentals and version differences.
- `openssl` — full OpenSSL CLI reference (genrsa, x509, req, ca, etc.).
- `troubleshooting/http-errors` — HTTP-layer debugging that often surfaces alongside TLS issues.
- `troubleshooting/dns-errors` — DNS resolution problems that masquerade as TLS errors (NXDOMAIN before handshake even starts).
- `age` — modern file encryption (different from TLS but uses similar key management ideas).
- `gpg` — OpenPGP for email and file encryption.

## References

- **RFC 8446** — *The Transport Layer Security (TLS) Protocol Version 1.3*. https://datatracker.ietf.org/doc/html/rfc8446
- **RFC 5246** — *The Transport Layer Security (TLS) Protocol Version 1.2*. https://datatracker.ietf.org/doc/html/rfc5246
- **RFC 8740** — *Using TLS 1.3 with HTTP/2*. https://datatracker.ietf.org/doc/html/rfc8740
- **RFC 5746** — *Transport Layer Security (TLS) Renegotiation Indication Extension*. https://datatracker.ietf.org/doc/html/rfc5746
- **RFC 6066** — *Transport Layer Security (TLS) Extensions: Extension Definitions* (includes SNI). https://datatracker.ietf.org/doc/html/rfc6066
- **RFC 7301** — *Transport Layer Security (TLS) Application-Layer Protocol Negotiation Extension*. https://datatracker.ietf.org/doc/html/rfc7301
- **RFC 6960** — *X.509 Internet Public Key Infrastructure Online Certificate Status Protocol - OCSP*. https://datatracker.ietf.org/doc/html/rfc6960
- **RFC 6962** — *Certificate Transparency*. https://datatracker.ietf.org/doc/html/rfc6962
- **RFC 7507** — *TLS Fallback Signaling Cipher Suite Value (SCSV) for Preventing Protocol Downgrade Attacks*. https://datatracker.ietf.org/doc/html/rfc7507
- **RFC 8996** — *Deprecating TLS 1.0 and TLS 1.1*. https://datatracker.ietf.org/doc/html/rfc8996
- **Mozilla SSL Configuration Generator** — https://ssl-config.mozilla.org
- **Qualys SSL Labs SSL Server Test** — https://www.ssllabs.com/ssltest/
- **badssl.com** — test cert anti-patterns (expired, wrong host, self-signed, weak DH, etc.). https://badssl.com
- **OpenSSL Manual — s_client(1)** — `man 1 s_client` or https://www.openssl.org/docs/manmaster/man1/openssl-s_client.html
- **OpenSSL Manual — verify(1)** — `man 1 verify` or https://www.openssl.org/docs/manmaster/man1/openssl-verify.html
- **OpenSSL Manual — x509(1)** — `man 1 x509` or https://www.openssl.org/docs/manmaster/man1/openssl-x509.html
- **OpenSSL Verify Codes List** — https://www.openssl.org/docs/manmaster/man1/openssl-verify.html
- **Let's Encrypt Rate Limits** — https://letsencrypt.org/docs/rate-limits/
- **Let's Encrypt Staging Environment** — https://letsencrypt.org/docs/staging-environment/
- **Chrome Net Errors List** — https://chromium.googlesource.com/chromium/src/+/HEAD/net/base/net_error_list.h
- **Firefox NSS Error Codes** — https://nss-crypto.org/reference/security/nspr/reference/pr_error_code_table/
- **CA/Browser Forum Baseline Requirements** — https://cabforum.org/baseline-requirements-documents/
- **smallstep step CLI** (private CA tooling) — https://smallstep.com/docs/step-cli/
- **mkcert** — https://github.com/FiloSottile/mkcert
- **testssl.sh** — https://testssl.sh
- **sslyze** — https://github.com/nabla-c0d3/sslyze
- **crt.sh — Certificate Search** — https://crt.sh
