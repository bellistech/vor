# OpenSSL (Cryptographic Swiss Army Knife)

Manage certificates, keys, encryption, and TLS connections from the command line.

## Key Generation

### Generate RSA Private Key

```bash
# 4096-bit RSA key (no passphrase)
openssl genrsa -out server.key 4096

# RSA key with AES-256 passphrase protection
openssl genrsa -aes256 -out server.key 4096
```

### Generate EC Private Key

```bash
# List available curves
openssl ecparam -list_curves

# P-256 key (most common)
openssl ecparam -genkey -name prime256v1 -noout -out server-ec.key

# P-384 key
openssl ecparam -genkey -name secp384r1 -noout -out server-ec.key
```

### Extract Public Key

```bash
openssl rsa -in server.key -pubout -out server.pub
openssl ec -in server-ec.key -pubout -out server-ec.pub
```

## Certificate Signing Requests (CSR)

### Generate CSR from Existing Key

```bash
openssl req -new -key server.key -out server.csr \
  -subj "/C=US/ST=California/L=San Francisco/O=Acme Inc/CN=acme.com"
```

### Generate Key + CSR in One Shot

```bash
openssl req -newkey rsa:4096 -nodes -keyout server.key -out server.csr \
  -subj "/CN=acme.com"
```

### CSR with Subject Alternative Names

```bash
openssl req -new -key server.key -out server.csr \
  -subj "/CN=acme.com" \
  -addext "subjectAltName=DNS:acme.com,DNS:*.acme.com,IP:10.0.0.1"
```

### Inspect a CSR

```bash
openssl req -in server.csr -noout -text
openssl req -in server.csr -noout -subject -nameopt multiline
```

## Certificates

### Self-Signed Certificate (Quick Dev Cert)

```bash
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout dev.key -out dev.crt -days 365 \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

### Sign a CSR with a CA

```bash
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out server.crt -days 825 -sha256
```

### Inspect a Certificate

```bash
openssl x509 -in server.crt -noout -text
openssl x509 -in server.crt -noout -dates        # just expiry
openssl x509 -in server.crt -noout -subject -issuer
openssl x509 -in server.crt -noout -fingerprint -sha256
```

### Verify Certificate Chain

```bash
openssl verify -CAfile ca.crt server.crt
openssl verify -CAfile ca.crt -untrusted intermediate.crt server.crt
```

## TLS Connections (s_client / s_server)

### Test TLS Connection to a Server

```bash
openssl s_client -connect acme.com:443 -servername acme.com </dev/null

# Show full certificate chain
openssl s_client -connect acme.com:443 -showcerts </dev/null

# Check expiry of remote cert
echo | openssl s_client -connect acme.com:443 -servername acme.com 2>/dev/null \
  | openssl x509 -noout -dates

# Test specific TLS version
openssl s_client -connect acme.com:443 -tls1_2
openssl s_client -connect acme.com:443 -tls1_3
```

### Run a Test TLS Server

```bash
openssl s_server -cert server.crt -key server.key -accept 4433 -www
```

## Encryption and Decryption

### Symmetric Encryption

```bash
# Encrypt a file with AES-256-CBC
openssl enc -aes-256-cbc -salt -pbkdf2 -in secret.txt -out secret.enc

# Decrypt
openssl enc -d -aes-256-cbc -pbkdf2 -in secret.enc -out secret.txt
```

### Asymmetric Encryption (RSA)

```bash
# Encrypt with public key (small data only)
openssl rsautl -encrypt -pubin -inkey server.pub -in msg.txt -out msg.enc

# Decrypt with private key
openssl rsautl -decrypt -inkey server.key -in msg.enc -out msg.txt
```

## Hashing and Digests

```bash
openssl dgst -sha256 file.tar.gz
openssl dgst -sha512 file.tar.gz
openssl dgst -md5 file.tar.gz           # legacy, avoid for security

# HMAC
openssl dgst -sha256 -hmac "mysecret" file.tar.gz

# Generate random bytes (hex / base64)
openssl rand -hex 32
openssl rand -base64 32
```

## PKCS12 / PFX

### Create PKCS12 Bundle

```bash
openssl pkcs12 -export -out bundle.p12 \
  -inkey server.key -in server.crt -certfile ca.crt
```

### Extract from PKCS12

```bash
# Private key
openssl pkcs12 -in bundle.p12 -nocerts -nodes -out extracted.key

# Certificate
openssl pkcs12 -in bundle.p12 -clcerts -nokeys -out extracted.crt

# CA certs
openssl pkcs12 -in bundle.p12 -cacerts -nokeys -out ca-chain.crt
```

## Diffie-Hellman Parameters

```bash
# Generate DH params (can take minutes)
openssl dhparam -out dhparam.pem 4096

# Use pre-defined groups instead (faster, equally secure)
openssl genpkey -genparam -algorithm DH -pkeyopt dh_paramgen_prime_len:4096 \
  -out dhparam.pem
```

## Format Conversion

```bash
# PEM to DER
openssl x509 -in cert.pem -outform DER -out cert.der

# DER to PEM
openssl x509 -in cert.der -inform DER -outform PEM -out cert.pem

# PEM key to PKCS8
openssl pkcs8 -topk8 -nocrypt -in server.key -out server-pkcs8.key
```

## Tips

- Chrome and Safari require SAN (Subject Alternative Name) even for self-signed certs -- CN alone is not enough
- Max certificate lifetime for public CAs is 397 days (since 2020); use `-days 397` or less
- Always use `-nodes` during development to skip passphrase prompts (means "no DES")
- `dhparam 2048` is the practical minimum; 4096 is preferred but slow to generate
- Use `-pbkdf2` with `enc` to avoid the "deprecated key derivation" warning
- `rsautl` can only encrypt data smaller than the key size minus padding; use symmetric encryption for large files
- PEM files are base64 with `-----BEGIN/END-----` headers; DER is raw binary
- When debugging TLS, `-servername` flag is required for SNI-enabled servers

## See Also

- tls, pki, certbot, cryptography, gpg

## References

- [OpenSSL Documentation](https://www.openssl.org/docs/)
- [openssl(1ssl) Man Page](https://man7.org/linux/man-pages/man1/openssl.1ssl.html)
- [OpenSSL Man Pages (3.0)](https://www.openssl.org/docs/man3.0/)
- [openssl-req(1ssl) Man Page](https://man7.org/linux/man-pages/man1/openssl-req.1ssl.html)
- [openssl-x509(1ssl) Man Page](https://man7.org/linux/man-pages/man1/openssl-x509.1ssl.html)
- [openssl-s_client(1ssl) Man Page](https://man7.org/linux/man-pages/man1/openssl-s_client.1ssl.html)
- [OpenSSL Cookbook (feisty duck)](https://www.feistyduck.com/library/openssl-cookbook/)
- [Arch Wiki — OpenSSL](https://wiki.archlinux.org/title/OpenSSL)
- [RFC 5280 — X.509 PKI Certificate and CRL Profile](https://www.rfc-editor.org/rfc/rfc5280)
