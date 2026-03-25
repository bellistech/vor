# TLS (Transport Layer Security)

> Cryptographic protocol securing communications over networks; successor to SSL.

## Handshake

### TLS 1.2 Handshake (RSA Key Exchange)

```
Client                              Server
  |--- ClientHello ------------------>|   # Supported ciphers, random, session ID
  |<-- ServerHello -------------------|   # Chosen cipher, random, session ID
  |<-- Certificate -------------------|   # Server's X.509 certificate chain
  |<-- ServerHelloDone ---------------|
  |--- ClientKeyExchange ------------>|   # Pre-master secret encrypted with server pubkey
  |--- ChangeCipherSpec ------------->|   # Switch to encrypted
  |--- Finished --------------------->|   # Verify handshake integrity
  |<-- ChangeCipherSpec --------------|
  |<-- Finished ----------------------|
```

### TLS 1.3 Handshake (1-RTT)

```
Client                              Server
  |--- ClientHello ------------------>|   # Supported groups, key shares, ciphers
  |<-- ServerHello -------------------|   # Chosen group, key share
  |<-- EncryptedExtensions -----------|   # All encrypted from here
  |<-- Certificate -------------------|
  |<-- CertificateVerify -------------|
  |<-- Finished ----------------------|
  |--- Finished --------------------->|
```

### TLS 1.3 0-RTT Resumption

```
Client                              Server
  |--- ClientHello + EarlyData ----->|   # PSK identity + application data
  |<-- ServerHello ------------------|   # Accept/reject early data
```

## Cipher Suites

### TLS 1.3 Cipher Suites (Mandatory)

```
TLS_AES_256_GCM_SHA384         # Preferred
TLS_AES_128_GCM_SHA256         # Most common
TLS_CHACHA20_POLY1305_SHA256   # Mobile-friendly (no AES-NI needed)
```

### TLS 1.2 Cipher Suite Format

```
# Format: TLS_KeyExchange_WITH_Cipher_Hash
TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256

# Avoid these (insecure)
TLS_RSA_WITH_AES_128_CBC_SHA         # No forward secrecy
TLS_RSA_WITH_RC4_128_SHA             # RC4 broken
TLS_RSA_WITH_3DES_EDE_CBC_SHA        # 3DES weak
```

## OpenSSL s_client

### Connect and Inspect

```bash
# Basic connection
openssl s_client -connect example.com:443

# Force TLS version
openssl s_client -connect example.com:443 -tls1_2
openssl s_client -connect example.com:443 -tls1_3

# Show full certificate chain
openssl s_client -connect example.com:443 -showcerts

# With SNI (Server Name Indication)
openssl s_client -connect example.com:443 -servername example.com

# Check specific cipher
openssl s_client -connect example.com:443 -cipher ECDHE-RSA-AES256-GCM-SHA384

# Check STARTTLS protocols
openssl s_client -connect mail.example.com:587 -starttls smtp
openssl s_client -connect mail.example.com:143 -starttls imap
```

### Inspect Certificate Details

```bash
# Download and decode server certificate
openssl s_client -connect example.com:443 </dev/null 2>/dev/null \
  | openssl x509 -noout -text

# Check expiration
openssl s_client -connect example.com:443 </dev/null 2>/dev/null \
  | openssl x509 -noout -dates

# Check SANs (Subject Alternative Names)
openssl s_client -connect example.com:443 </dev/null 2>/dev/null \
  | openssl x509 -noout -ext subjectAltName

# Verify certificate chain
openssl verify -CAfile ca-bundle.crt server.crt
```

## Certificate Chains

### Chain Order

```
End-entity (server) certificate     # Leaf — matches domain
  └── Intermediate CA certificate   # Signed by root
        └── Root CA certificate     # Self-signed, in trust store
```

### Verify Chain

```bash
# Check chain completeness
openssl s_client -connect example.com:443 -partial_chain </dev/null 2>&1 \
  | grep -E "Verify|depth"

# Concatenate chain for server config
cat server.crt intermediate.crt > fullchain.pem
```

## HSTS (HTTP Strict Transport Security)

### Header Format

```
# Basic — enforce HTTPS for 1 year
Strict-Transport-Security: max-age=31536000

# Include subdomains
Strict-Transport-Security: max-age=31536000; includeSubDomains

# Preload list eligible
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
```

### Nginx Configuration

```nginx
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
```

## Certificate Pinning

### HTTP Public Key Pinning (Deprecated — Use CT Instead)

```
# HPKP header (deprecated in favor of Certificate Transparency)
Public-Key-Pins: pin-sha256="base64=="; max-age=5184000
```

### Generate Pin Hash

```bash
# From certificate
openssl x509 -in cert.pem -pubkey -noout \
  | openssl pkey -pubin -outform DER \
  | openssl dgst -sha256 -binary \
  | base64
```

## Testing

### Test with curl

```bash
# Force TLS version
curl --tlsv1.2 --tls-max 1.2 https://example.com
curl --tlsv1.3 https://example.com

# Specify CA bundle
curl --cacert /path/to/ca-bundle.crt https://example.com

# Skip verification (testing only)
curl -k https://example.com

# Show TLS handshake details
curl -v https://example.com 2>&1 | grep -E "SSL|TLS|subject|issuer"
```

### Scan with nmap

```bash
# Enumerate supported ciphers
nmap --script ssl-enum-ciphers -p 443 example.com

# Check for known vulnerabilities
nmap --script ssl-cert,ssl-heartbleed -p 443 example.com
```

### Test with testssl.sh

```bash
# Full scan
./testssl.sh https://example.com

# Check specific protocols
./testssl.sh --protocols https://example.com

# Check vulnerabilities
./testssl.sh --vulnerabilities https://example.com
```

## Tips

- TLS 1.3 removes RSA key exchange, static DH, and CBC ciphers entirely.
- Always prefer ECDHE for forward secrecy; if server is compromised later, past sessions remain safe.
- OCSP stapling reduces latency by letting the server prove its certificate is not revoked.
- Use `ssl_session_tickets off;` in nginx when forward secrecy is critical.
- Certificate Transparency (CT) logs are now preferred over pinning for certificate trust.
- 0-RTT in TLS 1.3 is vulnerable to replay attacks; do not use for non-idempotent requests.

## References

- [RFC 8446 — TLS 1.3](https://www.rfc-editor.org/rfc/rfc8446)
- [RFC 5246 — TLS 1.2](https://www.rfc-editor.org/rfc/rfc5246)
- [RFC 6797 — HTTP Strict Transport Security (HSTS)](https://www.rfc-editor.org/rfc/rfc6797)
- [RFC 6066 — TLS Extensions: Extension Definitions (SNI)](https://www.rfc-editor.org/rfc/rfc6066)
- [RFC 7301 — TLS ALPN Extension](https://www.rfc-editor.org/rfc/rfc7301)
- [RFC 7525 — Recommendations for Secure Use of TLS and DTLS](https://www.rfc-editor.org/rfc/rfc7525)
- [RFC 8740 — Using TLS 1.3 with HTTP/2](https://www.rfc-editor.org/rfc/rfc8740)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [Mozilla Server Side TLS Guidelines](https://wiki.mozilla.org/Security/Server_Side_TLS)
- [testssl.sh — Testing TLS/SSL Encryption](https://testssl.sh/)
- [Qualys SSL Labs — SSL Server Test](https://www.ssllabs.com/ssltest/)
- [Arch Wiki — Transport Layer Security](https://wiki.archlinux.org/title/Transport_Layer_Security)
