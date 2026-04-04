# PKI (Public Key Infrastructure)

> Framework for managing digital certificates, certificate authorities, and trust hierarchies.

## Certificate Hierarchy

### Trust Model

```
Root CA (self-signed, offline)
  ├── Intermediate CA 1 (online, issues end-entity certs)
  │     ├── server.example.com
  │     └── client.example.com
  └── Intermediate CA 2
        └── internal.example.com
```

## CA Setup

### Create Root CA

```bash
# Generate root CA private key
openssl genrsa -aes256 -out rootCA.key 4096

# Create self-signed root certificate (10 years)
openssl req -x509 -new -nodes -key rootCA.key \
  -sha256 -days 3650 \
  -subj "/C=US/ST=State/O=MyOrg/CN=MyOrg Root CA" \
  -out rootCA.crt
```

### Create Intermediate CA

```bash
# Generate intermediate key
openssl genrsa -aes256 -out intermediate.key 4096

# Create CSR for intermediate
openssl req -new -key intermediate.key \
  -subj "/C=US/ST=State/O=MyOrg/CN=MyOrg Intermediate CA" \
  -out intermediate.csr

# Sign with root CA (5 years)
openssl x509 -req -in intermediate.csr \
  -CA rootCA.crt -CAkey rootCA.key -CAcreateserial \
  -sha256 -days 1825 \
  -extfile <(printf "basicConstraints=critical,CA:TRUE,pathlen:0\nkeyUsage=critical,keyCertSign,cRLSign") \
  -out intermediate.crt
```

## CSR Workflow

### Generate Key and CSR

```bash
# Generate private key (no passphrase for server use)
openssl genrsa -out server.key 2048

# Generate CSR
openssl req -new -key server.key \
  -subj "/C=US/ST=State/O=MyOrg/CN=example.com" \
  -out server.csr

# Generate key + CSR in one step
openssl req -new -newkey rsa:2048 -nodes \
  -keyout server.key -out server.csr \
  -subj "/C=US/ST=State/O=MyOrg/CN=example.com"

# With SANs (Subject Alternative Names)
openssl req -new -key server.key \
  -subj "/CN=example.com" \
  -addext "subjectAltName=DNS:example.com,DNS:www.example.com,IP:10.0.0.1" \
  -out server.csr
```

### Inspect CSR

```bash
# View CSR details
openssl req -in server.csr -noout -text

# Verify CSR signature
openssl req -in server.csr -verify -noout
```

### Sign CSR with CA

```bash
# Sign with extensions file
openssl x509 -req -in server.csr \
  -CA intermediate.crt -CAkey intermediate.key -CAcreateserial \
  -sha256 -days 365 \
  -extfile <(printf "subjectAltName=DNS:example.com,DNS:www.example.com\nkeyUsage=digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth") \
  -out server.crt
```

## X.509 Extensions

### Common Extensions

```
basicConstraints       = CA:FALSE                          # Not a CA
keyUsage               = digitalSignature, keyEncipherment # Key use restrictions
extendedKeyUsage       = serverAuth, clientAuth            # Purpose
subjectAltName         = DNS:example.com, IP:10.0.0.1     # Alternative names
authorityInfoAccess    = OCSP;URI:http://ocsp.example.com  # OCSP responder
crlDistributionPoints  = URI:http://crl.example.com/ca.crl # CRL location
subjectKeyIdentifier   = hash                              # Key ID
authorityKeyIdentifier = keyid,issuer                      # Issuer key ID
```

### Inspect Certificate

```bash
# Full details
openssl x509 -in cert.pem -noout -text

# Specific fields
openssl x509 -in cert.pem -noout -subject -issuer -dates -serial
openssl x509 -in cert.pem -noout -ext subjectAltName
openssl x509 -in cert.pem -noout -fingerprint -sha256
```

## OCSP (Online Certificate Status Protocol)

### Query OCSP Responder

```bash
# Extract OCSP URI from certificate
openssl x509 -in server.crt -noout -ocsp_uri

# Query OCSP status
openssl ocsp -issuer intermediate.crt -cert server.crt \
  -url http://ocsp.example.com -resp_text
```

### OCSP Stapling (Nginx)

```nginx
ssl_stapling on;
ssl_stapling_verify on;
ssl_trusted_certificate /etc/ssl/certs/fullchain.pem;
resolver 8.8.8.8;
```

## CRL (Certificate Revocation List)

### Revoke and Generate CRL

```bash
# Revoke a certificate
openssl ca -revoke server.crt -keyfile ca.key -cert ca.crt

# Generate CRL
openssl ca -gencrl -keyfile ca.key -cert ca.crt -out ca.crl

# Inspect CRL
openssl crl -in ca.crl -noout -text
```

## Certificate Formats

### Format Conversion

```bash
# PEM to DER
openssl x509 -in cert.pem -outform DER -out cert.der

# DER to PEM
openssl x509 -in cert.der -inform DER -outform PEM -out cert.pem

# PEM to PKCS12 (bundle key + cert + chain)
openssl pkcs12 -export -out cert.p12 \
  -inkey server.key -in server.crt -certfile intermediate.crt

# PKCS12 to PEM
openssl pkcs12 -in cert.p12 -out cert.pem -nodes

# PKCS12 extract key only
openssl pkcs12 -in cert.p12 -nocerts -nodes -out key.pem

# PKCS12 extract certs only
openssl pkcs12 -in cert.p12 -nokeys -out certs.pem
```

### Format Summary

```
PEM     Base64-encoded, -----BEGIN CERTIFICATE-----   .pem .crt .cer
DER     Binary ASN.1                                  .der .cer
PKCS7   Certs only (no key), used in S/MIME           .p7b .p7c
PKCS12  Binary bundle (key + cert + chain)            .p12 .pfx
```

## Tips

- Keep root CA keys offline and air-gapped; only use intermediates for day-to-day signing.
- Always include SANs; CN-only matching is deprecated in modern browsers.
- Set certificate lifetime to 90 days or less for automated renewal (Let's Encrypt default).
- Use ECDSA keys (P-256) over RSA for smaller certs and faster handshakes.
- OCSP stapling is preferred over CRL; clients do not need to contact the CA.
- Verify the full chain: `openssl verify -CAfile root.crt -untrusted intermediate.crt server.crt`.

## See Also

- tls, openssl, certbot, gpg, cryptography

## References

- [RFC 5280 — Internet X.509 PKI Certificate and CRL Profile](https://www.rfc-editor.org/rfc/rfc5280)
- [RFC 6960 — Online Certificate Status Protocol (OCSP)](https://www.rfc-editor.org/rfc/rfc6960)
- [RFC 6961 — TLS Multiple Certificate Status Extension](https://www.rfc-editor.org/rfc/rfc6961)
- [RFC 5652 — Cryptographic Message Syntax (CMS)](https://www.rfc-editor.org/rfc/rfc5652)
- [RFC 4210 — Certificate Management Protocol (CMP)](https://www.rfc-editor.org/rfc/rfc4210)
- [RFC 6844 — DNS CAA Resource Record](https://www.rfc-editor.org/rfc/rfc6844)
- [OpenSSL CA Documentation](https://www.openssl.org/docs/man3.0/man1/openssl-ca.html)
- [Red Hat RHEL 9 — Managing Certificates](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/securing_networks/using-shared-system-certificates_securing-networks)
- [Arch Wiki — Certificate Authority](https://wiki.archlinux.org/title/OpenSSL#Certificate_authority)
- [Let's Encrypt — Chain of Trust](https://letsencrypt.org/certificates/)
