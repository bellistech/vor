# Burp Suite (Web Application Security Testing Proxy)

> For authorized security testing, CTF competitions, and educational purposes only.

Burp Suite is the industry-standard platform for web application security testing.
This sheet covers Proxy, Repeater, Intruder, Scanner, and key extensions.

---

## Proxy Setup

### Browser Configuration

```
# Manual proxy settings (apply in browser or system network settings)
HTTP Proxy:  127.0.0.1    Port: 8080
HTTPS Proxy: 127.0.0.1    Port: 8080

# Install Burp CA certificate for HTTPS interception
1. With proxy running, browse to http://burpsuite (or http://127.0.0.1:8080)
2. Click "CA Certificate" to download cacert.der
3. Import into browser:
   - Firefox: Settings > Privacy & Security > Certificates > Import
   - Chrome:  Settings > Privacy > Security > Manage certificates > Import
   - System:  Add to OS trust store (Keychain on macOS, update-ca-certificates on Linux)

# FoxyProxy (recommended Firefox extension)
# Create a profile pointing to 127.0.0.1:8080
# Toggle proxy on/off easily during testing
```

### Proxy Settings in Burp

```
# Proxy > Options (or Proxy > Proxy settings in newer versions)

# Listener configuration
Bind to address: 127.0.0.1 (or 0.0.0.0 for remote clients)
Bind to port: 8080

# Invisible proxying (for non-proxy-aware clients)
Proxy > Options > Request handling > Support invisible proxying

# TLS pass-through (skip interception for specific hosts)
Proxy > Options > TLS Pass Through > Add: *.google.com, *.gstatic.com

# Response interception (off by default)
Proxy > Options > Intercept Server Responses > Enable
```

---

## Intercepting & Modifying Requests

### Intercept Controls

```
# Toggle interception
Proxy > Intercept > Intercept is on/off
Shortcut: Ctrl+I (Cmd+I on macOS)

# When a request is intercepted:
Forward     — send to server (with any modifications)
Drop        — discard the request
Action      — send to Repeater/Intruder/other tools
```

### Match & Replace Rules

```
# Proxy > Options > Match and Replace
# Automatically modify requests/responses passing through the proxy

# Example rules:
# Replace User-Agent
Type: Request header
Match: ^User-Agent:.*$
Replace: User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0

# Add custom header to all requests
Type: Request header
Match: ^$  (empty — adds new header)
Replace: X-Forwarded-For: 127.0.0.1

# Remove security headers from responses (for testing)
Type: Response header
Match: ^Content-Security-Policy:.*$
Replace: (empty — removes header)

Type: Response header
Match: ^X-Frame-Options:.*$
Replace: (empty)

# Modify response body
Type: Response body
Match: <input type="hidden" name="csrf_token" value="([^"]+)"
Replace: <input type="text" name="csrf_token" value="$1"
# Makes hidden fields visible for inspection
```

---

## Scope Configuration

```
# Target > Scope (or Target > Scope settings)

# Add to scope
Target > Site map > right-click host > Add to scope

# Manual scope rules
Include in scope:
  Protocol: Any
  Host: ^example\.com$
  Port: Any
  File: ^/app/.*

# Exclude from scope (noisy paths, third-party)
Exclude:
  Host: ^.*\.google\.com$
  File: ^/static/.*
  File: ^.*\.(jpg|png|gif|css|js|woff)$

# Filter proxy history to only show in-scope items
Proxy > HTTP history > Filter bar > Show only in-scope items

# Restrict Intruder/Scanner to scope
Project options > Scope > Use advanced scope control
```

---

## Repeater

```
# Send any request to Repeater for manual manipulation
# Right-click request > Send to Repeater
# Shortcut: Ctrl+R (Cmd+R on macOS)

# Usage workflow:
1. Send intercepted or history request to Repeater
2. Modify parameters, headers, body as needed
3. Click "Send" (or Ctrl+Space)
4. Analyze the response
5. Iterate — modify and resend

# Tips for Repeater:
# - Rename tabs (right-click tab) to organize tests
# - Use Ctrl+Shift+R to send and follow redirects
# - Toggle "Follow redirections" dropdown: Never / On-site only / In-scope only / Always
# - Use "Render" tab to see HTML response visually
# - Right-click > "Change request method" to toggle GET/POST

# Example: testing SQLi in Repeater
# Original: GET /user?id=5 HTTP/1.1
# Modified: GET /user?id=5' OR '1'='1 HTTP/1.1
# Send and compare response to original
```

---

## Intruder

### Attack Types

```
# Intruder > Positions tab
# Highlight parameters and click "Add" to mark injection points

# Attack Types:

# Sniper — single payload set, tests one position at a time
# Use for: testing each parameter individually
# Positions: §param1§ and §param2§
# Payload list: [a, b, c]
# Requests: param1=a&param2=original, param1=b&param2=original, ...
#            param1=original&param2=a, param1=original&param2=b, ...

# Battering Ram — single payload set, same value in all positions simultaneously
# Use for: same payload everywhere at once
# Requests: param1=a&param2=a, param1=b&param2=b, ...

# Pitchfork — multiple payload sets, parallel iteration (1:1 mapping)
# Use for: username:password pairs from a credential list
# Set 1: [user1, user2, user3]
# Set 2: [pass1, pass2, pass3]
# Requests: user1&pass1, user2&pass2, user3&pass3

# Cluster Bomb — multiple payload sets, all combinations (cartesian product)
# Use for: brute force all username + password combinations
# Set 1: [user1, user2]
# Set 2: [pass1, pass2, pass3]
# Requests: user1&pass1, user1&pass2, user1&pass3, user2&pass1, user2&pass2, user2&pass3
```

### Payload Configuration

```
# Intruder > Payloads tab

# Payload types:
Simple list      — manually entered or loaded from file
Runtime file     — stream from file (memory efficient for large lists)
Numbers          — sequential/random numbers (e.g., IDOR testing: 1-10000)
Dates            — date range generation
Brute forcer     — charset + length-based generation
Null payloads    — empty payloads (for repeated requests, e.g., race conditions)
Character frobber — modify original value one char at a time

# Payload processing (transforms applied in order):
Add prefix       — prepend string
Add suffix       — append string
Match/Replace    — regex substitution
Encode           — URL-encode, Base64, HTML-encode
Hash             — MD5, SHA-1, SHA-256
Case modification — uppercase, lowercase

# Payload encoding
# By default, Burp URL-encodes special chars
# Uncheck "Payload Encoding" at bottom if sending raw payloads

# Example: IDOR enumeration
# Payload type: Numbers
# From: 1, To: 10000, Step: 1
# Min/Max integer digits: 1
```

### Analyzing Intruder Results

```
# Key columns to sort/filter by:
Status code   — look for 200 vs 403/401/404
Length         — different length = different response = potential hit
Response time  — slower responses may indicate blind injection

# Grep - Match (Options tab)
# Flag responses containing specific strings
Add: "Welcome"         # successful login indicator
Add: "Invalid"         # failed login indicator
Add: "error"           # error messages

# Grep - Extract
# Extract data from responses using regex or delimiter
# Useful for extracting CSRF tokens, usernames, data from each response

# Grep - Payloads
# Flag responses that reflect the payload (XSS testing)
```

---

## Scanner

```
# Target > Site map > right-click > Scan / Active scan (Pro only)
# Or: right-click any request > Do active scan

# Scan types:
Crawl only          — discover content without testing vulnerabilities
Audit only          — test already-discovered content for vulnerabilities
Crawl and Audit     — full scan

# Scan configuration:
# Dashboard > New scan > Scan configuration

# Crawl settings:
Max crawl depth: 8-12
Max unique locations: 5000
Handle forms: Submit with default values

# Audit settings:
# Light — fast, fewer checks
# Normal — balanced
# Deep — thorough, slower

# Issue types to check:
# SQL injection, XSS, SSRF, file path traversal, command injection,
# open redirects, header injection, etc.

# Scan specific insertion points:
# Right-click request > Scan defined insertion points
# Manually select which parameters to test

# Viewing results:
# Target > Site map > select host > Issues tab
# Dashboard > Issue activity
# Each issue includes: severity, confidence, evidence, remediation
```

---

## Decoder

```
# Decoder tab — encode/decode/hash data

# Decode:
# URL, HTML, Base64, ASCII hex, Hex, Octal, Binary, Gzip

# Encode:
# Same formats as decode

# Hash:
# MD5, SHA-1, SHA-256, SHA-384, SHA-512

# Smart Decode — auto-detect and decode layered encoding
# Useful for multi-encoded payloads: base64(url(payload))

# Example workflow:
# 1. Paste: dGVzdEBleGFtcGxlLmNvbQ==
# 2. Decode as Base64: test@example.com
# 3. Or: Paste URL-encoded string, decode, then base64-decode inner value
```

---

## Comparer

```
# Comparer tab — diff two responses side by side

# Send items to Comparer:
# Right-click any request/response > Send to Comparer

# Use cases:
# - Compare response with valid vs invalid input (SQLi detection)
# - Compare response with vs without authentication
# - Compare response from different user contexts (IDOR)
# - Spot subtle differences in error messages

# Comparison modes:
Words    — highlights word-level differences
Bytes    — highlights byte-level differences
```

---

## Essential Extensions

### Installing Extensions

```
# Extender > BApp Store (or Extensions > BApp Store)
# Search and install directly from within Burp

# Manual install:
# Extender > Extensions > Add
# Extension type: Java / Python (requires Jython)
# Select .jar or .py file
```

### Must-Have Extensions

```
# ActiveScan++ — enhanced active scanning checks
# Adds additional scan checks for edge cases and novel vulnerabilities

# Autorize — automatic authorization testing
# Configure low-privilege session cookies
# Browse as high-privilege user
# Autorize replays each request with low-priv cookies and without cookies
# Flags resources accessible without proper authorization (IDOR/broken access control)
# Setup: Autorize tab > paste low-priv cookies > start

# JWT Editor — JSON Web Token manipulation
# Decode, edit, and resign JWTs
# Test algorithm confusion (RS256 to HS256)
# Test "none" algorithm bypass
# Test key injection attacks

# Logger++ — enhanced HTTP logging
# Advanced filtering, color coding, and export
# Regex-based filters on any request/response field

# Param Miner — hidden parameter discovery
# Right-click request > Extensions > Param Miner > Guess params
# Discovers hidden GET/POST/header/cookie parameters

# Turbo Intruder — high-speed request engine
# Python scripting for complex attack patterns
# Race conditions, rate limit testing
# Handles thousands of requests per second

# Hackvertor — advanced encoding/decoding
# Tag-based encoding in requests: <@base64>payload<@/base64>
# Auto-encodes on send, supports nested encoding

# Collaborator Everywhere — inject Burp Collaborator payloads
# into all requests to detect out-of-band vulnerabilities (SSRF, blind XXE, etc.)

# Upload Scanner — automated file upload vulnerability testing
# Tests for various upload bypass techniques automatically

# InQL — GraphQL security testing
# Introspection query, schema analysis, batch query testing
```

---

## Useful Workflows

### Authentication Testing

```
# 1. Log in as User A, capture session cookie
# 2. Log in as User B in a different browser
# 3. In Burp, send User A's requests to Repeater
# 4. Replace User A's session cookie with User B's
# 5. If User B can access User A's resources = IDOR / broken access control

# Autorize extension automates this:
# 1. Set User B's cookies in Autorize config
# 2. Browse as User A
# 3. Autorize replays all requests with User B's cookies
# 4. Review "Enforced!" vs "Bypassed!" status
```

### Race Condition Testing

```python
# Turbo Intruder script for race conditions
def queueRequests(target, wordlists):
    engine = RequestEngine(endpoint=target.endpoint,
                          concurrentConnections=30,
                          requestsPerConnection=100,
                          pipeline=False)
    for i in range(50):
        engine.queue(target.req, gate='race')
    engine.openGate('race')  # send all at once

def handleResponse(req, interesting):
    table.add(req)
```

### CSRF Token Extraction

```
# Intruder > Options > Grep - Extract
# Define extraction rule for CSRF token in response
# Intruder uses the extracted value in the next request

# Or use macros:
# Project options > Sessions > Macros > Add
# Record the request that returns a CSRF token
# Session handling rules > Add > Use macro to update parameter
```

---

## Tips

- Use scope aggressively to filter out noise from third-party domains
- Rename Repeater tabs to track what you are testing (e.g., "SQLi user param", "IDOR order")
- Intruder in Community Edition is rate-limited; use Turbo Intruder extension for speed
- Save Burp project files (.burp) regularly to avoid losing work
- Use "Copy as curl command" (right-click) to reproduce requests outside Burp
- Filter proxy history by MIME type, status code, or search term to find relevant traffic
- Disable browser caching when testing (DevTools > Network > Disable cache)
- Use "Engagement tools > Find comments" to discover developer comments in responses
- Set up hotkeys in User Options > Misc for faster workflow
- Use "Passive scanning" even in Community Edition -- it catches low-hanging fruit automatically

---

## See Also

- web-attacks
- recon
- metasploit
- password-attacks
- html
- css

## References

- [PortSwigger Burp Suite Documentation](https://portswigger.net/burp/documentation)
- [PortSwigger Web Security Academy](https://portswigger.net/web-security)
- [Burp Suite BApp Store](https://portswigger.net/bappstore)
- [Turbo Intruder](https://github.com/PortSwigger/turbo-intruder)
- [Autorize Extension](https://github.com/PortSwigger/autorize)
- [JWT Editor](https://portswigger.net/bappstore/26aaa5ded2f74beea19e2ed8345a93dd)
- [HackTricks Burp Suite](https://book.hacktricks.xyz/network-services-pentesting/pentesting-web/burp-suite)
