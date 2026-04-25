# SQL (Structured Query Language)

The lingua franca of relational databases — declarative queries, schema definition, and transactional data manipulation across PostgreSQL, MySQL, SQLite, SQL Server, and Oracle.

## Setup & Dialects

The "SQL standard" (ISO/IEC 9075) is real but no major engine fully implements it. Every production database is a dialect.

```bash
# PostgreSQL — Postgres.app, brew install postgresql, apt install postgresql
# psql is the canonical client
psql --version                       # psql (PostgreSQL) 16.x

# MySQL / MariaDB
mysql --version                       # Ver 8.x for MySQL or 10.x for MariaDB
mariadb --version                     # MariaDB

# SQLite — embedded, file-based, no server
sqlite3 --version                     # 3.4x.x
sqlite3 mydb.db                       # opens or creates the file

# SQL Server (Linux/macOS via mssql-tools)
sqlcmd -S localhost -U sa -P 'Pass!'  # interactive shell

# Oracle (rarely used in modern dev — included for completeness)
sqlplus user/pass@//host:1521/SID
```

Dialect-soup reality:

```bash
# Same idea, four spellings:
# PostgreSQL:        SELECT NOW();
# MySQL:             SELECT NOW();
# SQLite:            SELECT datetime('now');
# SQL Server:        SELECT GETDATE();
# Oracle:            SELECT SYSDATE FROM dual;

# Auto-incrementing primary key — five different syntaxes:
# PostgreSQL:        id SERIAL PRIMARY KEY        (or GENERATED ALWAYS AS IDENTITY)
# MySQL:             id INT AUTO_INCREMENT PRIMARY KEY
# SQLite:            id INTEGER PRIMARY KEY        (rowid alias)
# SQL Server:        id INT IDENTITY(1,1) PRIMARY KEY
# Oracle:            id NUMBER GENERATED ALWAYS AS IDENTITY
```

Standard SQL vs vendor extensions:

```bash
# Standard:        OFFSET 10 FETCH FIRST 5 ROWS ONLY
# Postgres/MySQL: LIMIT 5 OFFSET 10                  # vendor extension, ubiquitous
# SQL Server:     OFFSET 10 ROWS FETCH NEXT 5 ROWS ONLY
# Oracle 12c+:    OFFSET 10 ROWS FETCH NEXT 5 ROWS ONLY
```

Rule of thumb: write standard SQL where it works, document dialect-specific bits, and never assume your query ports cleanly.

## Connecting & psql / mysql / sqlite3 CLI

```bash
# psql — PostgreSQL
# psql [-h host] [-p port] [-U user] [-d database] [-W prompt for password]
psql -h db.example.com -p 5432 -U alice -d analytics
psql postgres://alice:secret@db.example.com:5432/analytics
psql                                  # uses PGUSER, PGDATABASE env vars / unix socket

# mysql — MySQL / MariaDB
# mysql [-h host] [-P port] [-u user] [-p password] [database]
mysql -h db.example.com -P 3306 -u alice -p analytics
mysql --defaults-file=~/.my.cnf       # credentials file (mode 600)

# sqlite3 — file-based; "connecting" means opening a file
sqlite3 ~/data/app.db
sqlite3 :memory:                      # transient in-memory database

# sqlcmd — SQL Server
sqlcmd -S tcp:db,1433 -U sa -P 'Pass!' -d analytics

# Common: read SQL from a file
psql -d analytics -f schema.sql
mysql -u alice -p analytics < schema.sql
sqlite3 app.db < schema.sql
sqlcmd -i schema.sql
```

psql meta-commands (backslash commands):

```bash
# \?            help on backslash commands
# \h SELECT     help on SQL command (parser-level)
# \q            quit
# \l            list databases
# \c dbname     connect to a different db
# \dt           list tables in current schema
# \dt+          with size
# \dt schema.*  list all tables in schema
# \d table      describe table (columns, indexes, FKs)
# \d+ table     verbose (storage, comments)
# \di           list indexes
# \dv           list views
# \df           list functions
# \du           list roles/users
# \dn           list schemas
# \timing on    time every query
# \x            toggle expanded display (records → vertical)
# \x auto       expanded only when wider than terminal
# \pset null '<NULL>'   show NULLs explicitly
# \e            edit query in $EDITOR
# \i file.sql   execute file
# \o out.txt    redirect output to file
# \copy t TO 'f.csv' WITH CSV HEADER     client-side COPY (no superuser)
# \watch 5      re-run last query every 5s
# \conninfo     show current connection
```

mysql shell shortcuts:

```bash
# help              list commands
# exit / quit       (or \q)
# show databases;
# use dbname;       (no semicolon needed for use, but accepted)
# show tables;
# describe tablename;     (or DESC)
# show create table t\G   \G prints rows vertically (one column per line)
# status;           connection info
# \! ls             shell escape
# source file.sql   run a script
# pager less        pipe long output through less
```

sqlite3 dot-commands:

```bash
# .help
# .quit / .exit
# .tables                       list tables
# .schema [table]               CREATE statements
# .databases                    attached files
# .mode column                  pretty columns (vs default 'list')
# .mode csv                     CSV output
# .mode markdown                | columns | tables |
# .headers on                   include column names
# .width 20 30                  set column widths for column mode
# .timer on                     time queries
# .read file.sql                run a file
# .dump                         emit SQL to recreate everything
# .backup main backup.db        online backup
# .open new.db                  switch databases
```

Why three different worlds: psql uses backslash, mysql uses bare words, sqlite3 uses dots. None of them are SQL — they are client features. The actual SQL is roughly the same.

## Data Types — Numeric

```bash
# Exact integers (signed)
# SMALLINT      2 bytes        -32768            32767
# INT/INTEGER   4 bytes        -2147483648       2147483647
# BIGINT        8 bytes        -9.2e18           9.2e18

# Exact decimal (arbitrary precision)
# NUMERIC(p, s) / DECIMAL(p, s)   p=total digits, s=digits after decimal
# Example: NUMERIC(10,2) holds -99,999,999.99 .. 99,999,999.99
# Use DECIMAL/NUMERIC for money — never FLOAT.

# Approximate (binary floating point)
# REAL              4 bytes  ~6  decimal digits precision
# DOUBLE PRECISION  8 bytes  ~15 decimal digits precision
# (SQL Server: REAL, FLOAT(24)/FLOAT(53). MySQL: FLOAT, DOUBLE.)

# Auto-incrementing
# Postgres:    id SERIAL          (alias for INT + sequence + DEFAULT nextval)
#              id BIGSERIAL       (BIGINT version)
#              id INT GENERATED ALWAYS AS IDENTITY    -- modern, SQL standard
# MySQL:       id INT AUTO_INCREMENT
# SQLite:      id INTEGER PRIMARY KEY        -- becomes ROWID alias, auto-fills
# SQL Server:  id INT IDENTITY(1,1)          -- start, increment
# Oracle:      id NUMBER GENERATED ALWAYS AS IDENTITY
```

```bash
CREATE TABLE prices (
  id        SERIAL       PRIMARY KEY,        -- Postgres
  amount    NUMERIC(12,2) NOT NULL,           -- exact money
  qty       INT          NOT NULL CHECK (qty >= 0),
  weight_kg DOUBLE PRECISION,
  big_id    BIGINT
);
```

Storage rules of thumb:

```bash
# Don't use BIGINT when INT fits. 4 bytes vs 8 across millions of rows matters.
# Don't use SMALLINT just to "save space" — alignment usually negates it.
# Don't store currency as REAL/DOUBLE. 0.1 + 0.2 = 0.30000000000000004.
# Use NUMERIC(p, s) — slower than INT, but exact.
```

## Data Types — String

```bash
# CHAR(n)       fixed-length, RIGHT-padded with spaces to n
# VARCHAR(n)    variable-length up to n; n is a constraint, not storage
# TEXT          variable-length, unbounded (in Postgres; MySQL has TEXT/MEDIUMTEXT/LONGTEXT)
# CLOB          standard SQL "character large object"

# Postgres: VARCHAR(n) and TEXT have identical performance — TEXT is preferred.
# MySQL:    VARCHAR(n) up to 65535 (row size), TEXT family for longer.
# SQLite:   types are advisory; everything is essentially TEXT.
# SQL Server: NVARCHAR(n) for unicode, VARCHAR(n) ASCII; NVARCHAR(MAX) up to 2GB.
```

```bash
CREATE TABLE users (
  id     SERIAL PRIMARY KEY,
  email  VARCHAR(320) UNIQUE NOT NULL,        -- 320 = SMTP max
  bio    TEXT,
  code   CHAR(3) NOT NULL                     -- e.g. 'USA' (always 3 chars)
);
```

Collation and encoding:

```bash
# Encoding: prefer UTF-8 everywhere.
# Postgres:  initdb --encoding=UTF8 --locale=en_US.UTF-8
# MySQL:     CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    -- NOT 'utf8' (3 bytes only)
# SQL Server: NVARCHAR for unicode; default collation set per database.

# Per-column collation (Postgres):
ALTER TABLE users ALTER COLUMN email TYPE TEXT COLLATE "C";   -- byte-wise, fastest

# Case-insensitive comparison:
# Postgres CITEXT extension:
CREATE EXTENSION IF NOT EXISTS citext;
CREATE TABLE accounts (email CITEXT UNIQUE);   -- comparisons ignore case automatically

# MySQL: utf8mb4_unicode_ci is case-insensitive by default
# SQL Server: collation determines it (e.g. SQL_Latin1_General_CP1_CI_AS — CI = case insensitive)
```

## Data Types — Date/Time

```bash
# DATE                          calendar date, no time
# TIME [WITHOUT TIME ZONE]      wall-clock time
# TIME WITH TIME ZONE           rarely useful — TZ alone is ambiguous
# TIMESTAMP [WITHOUT TIME ZONE] date + time, naive (no zone)
# TIMESTAMP WITH TIME ZONE      stored as UTC; converted on input/output (Postgres: TIMESTAMPTZ)
# INTERVAL                      duration: '1 day', '2 months', '03:30:00'
```

```bash
# Postgres: prefer TIMESTAMPTZ for any user-facing time.
CREATE TABLE events (
  id        SERIAL PRIMARY KEY,
  occurred  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  duration  INTERVAL
);

INSERT INTO events (occurred, duration) VALUES
  ('2026-04-25 10:30:00+00', INTERVAL '1 hour 30 minutes');

# MySQL: use TIMESTAMP (UTC, range 1970-2038) or DATETIME (no zone, larger range).
# SQLite: no real date type — TEXT 'YYYY-MM-DD HH:MM:SS' or REAL Julian day or INTEGER unix epoch.
# SQL Server: DATETIME2 (preferred), DATETIMEOFFSET (with TZ), DATE, TIME.

# Epoch handling
# Postgres:   EXTRACT(EPOCH FROM ts)             returns double — seconds since 1970-01-01
#             TO_TIMESTAMP(1714060800)            epoch → timestamptz
# MySQL:      UNIX_TIMESTAMP(ts)                  seconds since epoch
#             FROM_UNIXTIME(1714060800)
# SQLite:     strftime('%s', 'now')               text seconds-since-epoch
#             datetime(1714060800, 'unixepoch')
# SQL Server: DATEDIFF(SECOND, '1970-01-01', GETUTCDATE())
```

## Data Types — Boolean / JSON / Array / Special

```bash
# BOOLEAN
# Postgres: BOOLEAN with literals TRUE/FALSE/NULL — first-class.
# MySQL:    BOOLEAN is an alias for TINYINT(1). 0=false, 1=true. No real bool type.
# SQLite:   no BOOLEAN — store 0/1 as INTEGER.
# SQL Server: BIT (0/1, NULL) — closest equivalent.

# JSON
# Postgres: JSON (text) and JSONB (binary, indexable, deduped, the one you want)
# MySQL 5.7+: JSON (binary, validated)
# SQLite 3.45+: JSON1 extension; JSONB introduced 3.45+ (mostly compat shape)
# SQL Server: NVARCHAR + ISJSON()/JSON_VALUE()/JSON_QUERY() — no dedicated type until 2025

# Arrays
# Postgres: native arrays — INT[], TEXT[], or JSONB for nested structures
CREATE TABLE tags_demo (
  id   SERIAL PRIMARY KEY,
  tags TEXT[] NOT NULL DEFAULT '{}'
);
INSERT INTO tags_demo (tags) VALUES (ARRAY['sql','postgres']);
INSERT INTO tags_demo (tags) VALUES ('{"a","b"}');                -- alt literal
SELECT * FROM tags_demo WHERE 'sql' = ANY(tags);
SELECT * FROM tags_demo WHERE tags && ARRAY['sql','redis'];        -- overlap
# MySQL/SQLite/SQL Server: no native arrays — use JSON or a join table.

# UUID
# Postgres: native UUID type; uuid-ossp or pgcrypto for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE u (id UUID PRIMARY KEY DEFAULT gen_random_uuid());
# MySQL 8: UUID() function returns string; use BINARY(16) for compact storage.
# SQLite: store as TEXT or BLOB.
# SQL Server: UNIQUEIDENTIFIER + NEWID()/NEWSEQUENTIALID()

# ENUM
# Postgres: CREATE TYPE color AS ENUM ('red','green','blue');
# MySQL:    color ENUM('red','green','blue') NOT NULL
# SQLite:   no ENUM — use CHECK (color IN ('red','green','blue'))
# SQL Server: same — CHECK constraint.

# SQLite ROWID / _rowid_ / oid: every regular table has an implicit 64-bit
# integer rowid. Selecting rowid, oid, or _rowid_ all return it. Becomes
# alias for INTEGER PRIMARY KEY column. Use WITHOUT ROWID for table-clustered.
```

## CREATE TABLE

```bash
# Anatomy of a column definition:
#   name TYPE [DEFAULT expr] [NOT NULL] [UNIQUE] [PRIMARY KEY]
#   [REFERENCES other(col) ON DELETE action ON UPDATE action]
#   [CHECK (expr)] [COLLATE name] [GENERATED ...]

CREATE TABLE IF NOT EXISTS departments (
  id        SERIAL       PRIMARY KEY,
  name      VARCHAR(100) NOT NULL UNIQUE,
  created   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS employees (
  id          SERIAL        PRIMARY KEY,
  email       CITEXT        NOT NULL UNIQUE,
  first_name  VARCHAR(100)  NOT NULL,
  last_name   VARCHAR(100)  NOT NULL,
  salary      NUMERIC(12,2) NOT NULL CHECK (salary >= 0),
  dept_id     INT           NOT NULL,
  manager_id  INT,
  hired_at    DATE          NOT NULL DEFAULT CURRENT_DATE,
  active      BOOLEAN       NOT NULL DEFAULT TRUE,

  -- table-level constraints (NAMED — important for ALTER later)
  CONSTRAINT fk_employees_dept    FOREIGN KEY (dept_id)    REFERENCES departments(id)
    ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT fk_employees_manager FOREIGN KEY (manager_id) REFERENCES employees(id)
    ON DELETE SET NULL,
  CONSTRAINT chk_name_not_blank   CHECK (length(trim(first_name)) > 0)
);
```

Always name your constraints. `pg_dump`-style auto-generated names like `employees_dept_id_fkey` are fine until you need `ALTER TABLE ... DROP CONSTRAINT` and have to look them up.

```bash
# IF NOT EXISTS — idempotent creation (Postgres, MySQL, SQLite, SQL Server 2016+)
CREATE TABLE IF NOT EXISTS staging (...);

# Temporary tables — session-local
CREATE TEMP TABLE tmp_import (LIKE employees INCLUDING DEFAULTS);    -- Postgres
CREATE TEMPORARY TABLE tmp_import (id INT) ENGINE=Memory;            -- MySQL
CREATE TEMP TABLE tmp_import AS SELECT * FROM employees WHERE 0;     -- SQLite

# Generated/computed columns
# Postgres 12+:
ALTER TABLE employees
  ADD COLUMN full_name TEXT GENERATED ALWAYS AS (first_name || ' ' || last_name) STORED;
# MySQL 5.7+:   full_name VARCHAR(200) AS (CONCAT(first_name,' ',last_name)) STORED
# SQLite 3.31+: full_name TEXT GENERATED ALWAYS AS (first_name || ' ' || last_name) VIRTUAL
# SQL Server:   full_name AS (first_name + ' ' + last_name)
```

## ALTER TABLE

```bash
# Add column
ALTER TABLE employees ADD COLUMN phone VARCHAR(20);
ALTER TABLE employees ADD COLUMN phone VARCHAR(20) DEFAULT '' NOT NULL;

# Drop column
ALTER TABLE employees DROP COLUMN phone;
ALTER TABLE employees DROP COLUMN IF EXISTS phone;                 -- Postgres

# Rename column / table
ALTER TABLE employees RENAME COLUMN dept TO dept_id;
ALTER TABLE employees RENAME TO staff;
# MySQL alt:  ALTER TABLE employees CHANGE COLUMN dept dept_id INT;
# SQLite:     ALTER TABLE employees RENAME COLUMN dept TO dept_id;     (3.25+)

# Change column type
# Postgres:
ALTER TABLE employees ALTER COLUMN salary TYPE NUMERIC(14,2)
  USING salary::NUMERIC(14,2);            -- USING when conversion is non-trivial
# MySQL:
ALTER TABLE employees MODIFY COLUMN salary DECIMAL(14,2) NOT NULL;
# SQL Server:
ALTER TABLE employees ALTER COLUMN salary DECIMAL(14,2) NOT NULL;
# SQLite: limited — typical workaround is CREATE new + INSERT SELECT + RENAME.

# Set / drop default
ALTER TABLE employees ALTER COLUMN active SET DEFAULT TRUE;
ALTER TABLE employees ALTER COLUMN active DROP DEFAULT;
# MySQL:    ALTER TABLE employees ALTER active SET DEFAULT TRUE;

# NOT NULL toggling
# Postgres / SQL Server:
ALTER TABLE employees ALTER COLUMN phone SET NOT NULL;
ALTER TABLE employees ALTER COLUMN phone DROP NOT NULL;
# MySQL: must redeclare full type — MODIFY COLUMN phone VARCHAR(20) NOT NULL;

# Add / drop constraints
ALTER TABLE employees ADD CONSTRAINT uq_employees_email UNIQUE (email);
ALTER TABLE employees DROP CONSTRAINT uq_employees_email;
# MySQL drops UNIQUE differently:  ALTER TABLE employees DROP INDEX uq_employees_email;
# MySQL drops FK:                  ALTER TABLE employees DROP FOREIGN KEY fk_emp_dept;
```

Locking behavior — the surprise that takes prod down:

```bash
# PostgreSQL:
#   - ADD COLUMN with no default            -> brief ACCESS EXCLUSIVE, fast
#   - ADD COLUMN with VOLATILE default      -> rewrites entire table (slow + AEL)
#   - ADD COLUMN with constant default      -> fast since PG11 (metadata-only)
#   - ALTER COLUMN TYPE (non-binary-coercible) -> ACCESS EXCLUSIVE + full rewrite
#   - CREATE INDEX                          -> blocks writes; use CREATE INDEX CONCURRENTLY
# MySQL (InnoDB):
#   - Most online via ALGORITHM=INPLACE / INSTANT (8.0); some still copy
#   - Add column: 8.0 INSTANT for trailing nullable cols
#   - Add index: ALGORITHM=INPLACE LOCK=NONE (depends)
# SQL Server:
#   - Online operations require Enterprise edition for some ops
```

Sketch a migration plan. Don't ALTER live tables blindly during peak.

## DROP / TRUNCATE

```bash
DROP TABLE employees;                          -- error if not exists
DROP TABLE IF EXISTS employees;                -- idempotent
DROP TABLE employees CASCADE;                  -- also drop dependent objects (Postgres)
DROP TABLE employees RESTRICT;                 -- default — fail if any dep

DROP INDEX idx_employees_email;
DROP VIEW IF EXISTS active_employees;
DROP SCHEMA reporting CASCADE;                 -- nuke schema and contents
DROP DATABASE staging;                         -- can't drop the one you're connected to

# TRUNCATE — fast empty
TRUNCATE TABLE staging;                                       -- standard
TRUNCATE TABLE staging RESTART IDENTITY CASCADE;              -- Postgres: reset sequences + cascade FK
TRUNCATE TABLE staging1, staging2;                            -- multiple at once (Postgres/MySQL)

# Why TRUNCATE > DELETE FROM:
#   - TRUNCATE drops storage; DELETE rewrites every row's xmin (Postgres MVCC)
#   - TRUNCATE is DDL — implicit commit on MySQL; rolls back on Postgres if in TX
#   - DELETE fires triggers; TRUNCATE typically doesn't (Postgres can BEFORE/AFTER TRUNCATE)
#   - Foreign keys referring to the table can prevent TRUNCATE without CASCADE
```

## SELECT — Basic

```bash
# SELECT [DISTINCT] expr [, expr ...]
# FROM table
# [WHERE cond]
# [GROUP BY ...]
# [HAVING ...]
# [ORDER BY ...]
# [LIMIT n [OFFSET m]]

SELECT first_name, last_name FROM employees;

# SELECT * — fine for ad-hoc, avoid in production code:
#   - Breaks when columns are added/reordered
#   - Pulls more data than needed (network + memory)
#   - Defeats covering indexes
SELECT * FROM employees;

# DISTINCT — removes duplicate rows after the projection
SELECT DISTINCT department FROM employees;
SELECT DISTINCT department, country FROM employees;     -- combination must be distinct

# Postgres-only: DISTINCT ON (col) — keeps first row per col-value (with ORDER BY)
SELECT DISTINCT ON (dept_id) dept_id, name, salary
FROM employees
ORDER BY dept_id, salary DESC;                          -- top-paid per dept

# Aliases
SELECT first_name AS fname, salary * 12 AS annual_salary FROM employees;
SELECT first_name fname FROM employees;                  -- AS optional for column aliases
SELECT e.first_name FROM employees AS e;                 -- AS optional for table alias too

# LIMIT / OFFSET
SELECT * FROM employees ORDER BY id LIMIT 10;            -- first 10 by id
SELECT * FROM employees ORDER BY id LIMIT 10 OFFSET 20;  -- rows 21..30
SELECT TOP 10 * FROM employees ORDER BY id;              -- SQL Server alt
SELECT * FROM employees ORDER BY id
  OFFSET 0 ROWS FETCH FIRST 10 ROWS ONLY;                -- standard SQL form
```

## WHERE Clause

```bash
# Equality / inequality
SELECT * FROM employees WHERE salary = 50000;
SELECT * FROM employees WHERE salary <> 50000;             -- standard
SELECT * FROM employees WHERE salary != 50000;             -- accepted everywhere
SELECT * FROM employees WHERE salary > 50000;
SELECT * FROM employees WHERE salary >= 50000;

# Boolean composition
SELECT * FROM employees WHERE salary > 50000 AND department = 'Eng';
SELECT * FROM employees WHERE department IN ('Sales','Mktg') OR active = FALSE;
SELECT * FROM employees WHERE NOT active;

# Set membership
SELECT * FROM employees WHERE department IN ('Sales','Mktg','HR');
SELECT * FROM employees WHERE department NOT IN ('Sales','Mktg');     -- NULL trap below

# Range (inclusive on both ends)
SELECT * FROM employees WHERE salary BETWEEN 40000 AND 80000;
-- equivalent to: salary >= 40000 AND salary <= 80000

# NULL — always IS / IS NOT, never =
SELECT * FROM employees WHERE manager_id IS NULL;
SELECT * FROM employees WHERE manager_id IS NOT NULL;
SELECT * FROM employees WHERE manager_id = NULL;          -- BUG: returns 0 rows always

# Pattern matching with LIKE
#   %  zero or more chars
#   _  exactly one char
SELECT * FROM employees WHERE last_name LIKE 'Sm%';        -- starts with Sm
SELECT * FROM employees WHERE last_name LIKE '%son';       -- ends with son
SELECT * FROM employees WHERE last_name LIKE '_a%';        -- second char a

# Case-insensitive
SELECT * FROM employees WHERE LOWER(last_name) LIKE 'sm%';   -- portable
SELECT * FROM employees WHERE last_name ILIKE 'sm%';         -- Postgres only
SELECT * FROM employees WHERE last_name LIKE 'Sm%' COLLATE Latin1_General_CI_AS;  -- SQL Server

# SIMILAR TO — regex-like, less common (Postgres standard SQL extension)
SELECT * FROM employees WHERE last_name SIMILAR TO '(Sm|Ma)%';

# Regex (dialect-specific)
# Postgres: ~ case-sensitive, ~* case-insensitive, !~ negation
SELECT * FROM employees WHERE last_name ~ '^Sm';
SELECT * FROM employees WHERE last_name ~* '^sm';
# MySQL:    REGEXP / RLIKE
SELECT * FROM employees WHERE last_name REGEXP '^Sm';
# SQLite:   REGEXP via extension (often disabled)
# SQL Server: LIKE only — use CLR or pattern hacks

# ESCAPE — match literal % or _
SELECT * FROM logs WHERE msg LIKE '100\%' ESCAPE '\';     -- matches '100%'
```

## ORDER BY / GROUP BY / HAVING

```bash
# ORDER BY — applied after WHERE/GROUP BY/HAVING; before LIMIT
SELECT * FROM employees ORDER BY salary DESC, last_name ASC;

# NULL ordering
SELECT * FROM employees ORDER BY manager_id NULLS LAST;     -- Postgres / Oracle
-- MySQL: NULLs sort first ASC by default; trick:
SELECT * FROM employees ORDER BY manager_id IS NULL, manager_id;     -- non-nulls first

# Position-based ORDER BY (shorthand only — fragile if columns reorder)
SELECT first_name, last_name, salary FROM employees ORDER BY 3 DESC;

# Order by an expression
SELECT * FROM employees ORDER BY LENGTH(last_name);

# GROUP BY — collapses rows; non-aggregated cols in SELECT must be in GROUP BY (in standard SQL)
SELECT department, COUNT(*) AS headcount, AVG(salary) AS avg_sal
FROM employees
GROUP BY department;

# Multiple grouping keys
SELECT department, country, COUNT(*) FROM employees GROUP BY department, country;

# GROUPING SETS / ROLLUP / CUBE — multiple groupings in one query
SELECT department, country, COUNT(*)
FROM employees
GROUP BY GROUPING SETS ((department), (country), ());        -- per dept, per country, total

SELECT department, country, COUNT(*)
FROM employees
GROUP BY ROLLUP (department, country);                       -- subtotals per dept + grand total

# HAVING — filter AFTER aggregation; WHERE filters BEFORE
SELECT department, AVG(salary) AS avg_sal
FROM employees
WHERE active = TRUE                          -- pre-agg filter
GROUP BY department
HAVING AVG(salary) > 60000                   -- post-agg filter
ORDER BY avg_sal DESC;
```

## JOINs

```bash
# INNER JOIN — only rows present in BOTH tables
SELECT e.first_name, d.name
FROM employees e
INNER JOIN departments d ON e.dept_id = d.id;
-- INNER keyword optional: JOIN === INNER JOIN

# LEFT [OUTER] JOIN — all rows from left; NULLs for missing right
SELECT e.first_name, d.name AS dept
FROM employees e
LEFT JOIN departments d ON e.dept_id = d.id;

# RIGHT [OUTER] JOIN — symmetric to LEFT
SELECT e.first_name, d.name
FROM employees e
RIGHT JOIN departments d ON e.dept_id = d.id;
-- (rarely needed — easier to just swap sides and use LEFT)

# FULL [OUTER] JOIN — union of both LEFT and RIGHT
SELECT e.first_name, d.name
FROM employees e
FULL OUTER JOIN departments d ON e.dept_id = d.id;
-- MySQL has no FULL JOIN — emulate via LEFT UNION RIGHT.

# CROSS JOIN — cartesian product (every left × every right)
SELECT a.color, b.size FROM colors a CROSS JOIN sizes b;
-- equivalent to:  FROM colors a, sizes b      (implicit cross — discouraged, easy mistake)

# NATURAL JOIN — join on every same-named column. DISCOURAGED.
-- Adding a column later silently changes the join. Avoid.

# USING — when join columns share a name
SELECT first_name, name FROM employees JOIN departments USING (dept_id);
-- USING(dept_id) folds the two columns into one (no e.dept_id vs d.dept_id ambiguity)

# Self join
SELECT e.first_name AS employee, m.first_name AS manager
FROM employees e
LEFT JOIN employees m ON e.manager_id = m.id;

# Multi-table joins
SELECT e.first_name, d.name AS dept, c.name AS country
FROM employees e
JOIN departments d ON e.dept_id = d.id
JOIN countries  c ON d.country_id = c.id;

# Filtering in JOIN vs WHERE — different for OUTER joins!
-- Keeps unmatched employees, but filters the right side:
SELECT * FROM employees e LEFT JOIN departments d
  ON e.dept_id = d.id AND d.active = TRUE;
-- Loses unmatched employees (because the WHERE makes the LEFT effectively INNER):
SELECT * FROM employees e LEFT JOIN departments d ON e.dept_id = d.id
WHERE d.active = TRUE;
```

## Subqueries

```bash
# Scalar subquery — returns one row, one column
SELECT name, salary,
  (SELECT AVG(salary) FROM employees) AS company_avg
FROM employees;

# Row subquery — returns one row, multiple columns
SELECT * FROM employees
WHERE (dept_id, salary) = (SELECT dept_id, MAX(salary) FROM employees WHERE id = 42);

# Table subquery — used as a derived table in FROM
SELECT dept, avg_sal FROM (
  SELECT department AS dept, AVG(salary) AS avg_sal
  FROM employees GROUP BY department
) AS dept_stats
WHERE avg_sal > 50000;
-- Postgres requires the alias (`AS dept_stats`); MySQL does too in modern versions.

# IN / NOT IN
SELECT * FROM employees WHERE dept_id IN (SELECT id FROM departments WHERE country = 'US');

# EXISTS / NOT EXISTS — generally faster than IN for large lists; NULL-safe
SELECT * FROM departments d
WHERE EXISTS (SELECT 1 FROM employees e WHERE e.dept_id = d.id);

# Correlated subquery — references outer row; runs per outer row in worst case
SELECT e.first_name, e.salary
FROM employees e
WHERE e.salary > (SELECT AVG(salary) FROM employees e2 WHERE e2.dept_id = e.dept_id);

# LATERAL (Postgres / Oracle / SQL Server CROSS APPLY) —
# RHS subquery sees columns of LHS row-by-row.
SELECT e.first_name, last3.amount
FROM employees e,
LATERAL (
  SELECT amount FROM payroll p
  WHERE p.employee_id = e.id
  ORDER BY paid_on DESC LIMIT 3
) AS last3;
-- SQL Server equivalent:
-- FROM employees e CROSS APPLY (SELECT TOP 3 amount FROM payroll WHERE employee_id = e.id ORDER BY paid_on DESC) AS last3
```

## Set Operations

```bash
# UNION — combine; remove duplicates
SELECT name FROM employees
UNION
SELECT name FROM contractors;

# UNION ALL — keep duplicates; faster (no dedup pass)
SELECT name FROM employees
UNION ALL
SELECT name FROM contractors;

# INTERSECT — rows in BOTH queries
SELECT email FROM customers
INTERSECT
SELECT email FROM newsletter_subs;

# INTERSECT ALL — preserve dup counts (e.g. 2 in left + 3 in right -> 2 in result)

# EXCEPT — rows in left but NOT in right (Oracle uses MINUS)
SELECT email FROM customers
EXCEPT
SELECT email FROM unsubscribed;

# Rules:
#   - Each SELECT must have the same number/type of columns
#   - Result column names come from the FIRST query
#   - ORDER BY / LIMIT apply to the FINAL combined result; wrap a single query in
#     parens to order it locally:
(SELECT name FROM e ORDER BY name LIMIT 5)
UNION ALL
(SELECT name FROM c ORDER BY name LIMIT 5);
```

## Aggregates

```bash
# Counting
SELECT COUNT(*)        FROM employees;             -- total rows (NULLs included)
SELECT COUNT(manager_id) FROM employees;           -- non-NULL count
SELECT COUNT(DISTINCT department) FROM employees;  -- distinct values

# Numeric
SELECT SUM(salary), AVG(salary), MIN(salary), MAX(salary), STDDEV(salary), VARIANCE(salary)
FROM employees;

# Strings
# Postgres / Standard:
SELECT department, STRING_AGG(last_name, ', ' ORDER BY last_name) AS members
FROM employees GROUP BY department;
# MySQL:
SELECT department, GROUP_CONCAT(last_name ORDER BY last_name SEPARATOR ', ') FROM employees GROUP BY department;
# SQLite:
SELECT department, GROUP_CONCAT(last_name, ', ') FROM employees GROUP BY department;
# SQL Server 2017+:
SELECT department, STRING_AGG(last_name, ', ') WITHIN GROUP (ORDER BY last_name) FROM employees GROUP BY department;

# Arrays / JSON
SELECT department, ARRAY_AGG(last_name ORDER BY last_name) FROM employees GROUP BY department;     -- Postgres
SELECT department, JSON_AGG(row_to_json(employees)) FROM employees GROUP BY department;             -- Postgres
SELECT department, JSONB_AGG(row_to_json(employees)) FROM employees GROUP BY department;            -- Postgres

# FILTER (WHERE) — conditional aggregation; standard SQL, not in MySQL/SQLite older
SELECT
  COUNT(*) FILTER (WHERE active),
  COUNT(*) FILTER (WHERE NOT active)
FROM employees;
-- Portable equivalent:
SELECT
  COUNT(CASE WHEN active     THEN 1 END),
  COUNT(CASE WHEN NOT active THEN 1 END)
FROM employees;

# DISTINCT inside aggregates
SELECT COUNT(DISTINCT dept_id), AVG(DISTINCT salary) FROM employees;
```

## Window Functions

```bash
# Pattern: function() OVER (PARTITION BY ... ORDER BY ... [ROWS|RANGE BETWEEN ...])
# Window functions DO NOT collapse rows like aggregates do — every row is preserved.

# ROW_NUMBER — unique sequential, no ties
SELECT first_name, dept_id, salary,
       ROW_NUMBER() OVER (PARTITION BY dept_id ORDER BY salary DESC) AS rn
FROM employees;

# RANK / DENSE_RANK — handle ties differently
#   RANK:        1, 2, 2, 4   (gaps after ties)
#   DENSE_RANK:  1, 2, 2, 3   (no gaps)
SELECT first_name, salary,
       RANK()       OVER (ORDER BY salary DESC) AS r,
       DENSE_RANK() OVER (ORDER BY salary DESC) AS dr
FROM employees;

# PERCENT_RANK / CUME_DIST — relative position 0..1
SELECT first_name, salary,
       PERCENT_RANK() OVER (ORDER BY salary) AS pct_rank,
       CUME_DIST()    OVER (ORDER BY salary) AS cume_dist
FROM employees;

# NTILE(n) — divide into n approximately-equal buckets
SELECT first_name, salary, NTILE(4) OVER (ORDER BY salary) AS quartile FROM employees;

# LAG / LEAD — peek backward / forward
SELECT first_name, hired_at, salary,
       LAG(salary, 1)  OVER (ORDER BY hired_at) AS prev_salary,
       LEAD(salary, 1) OVER (ORDER BY hired_at) AS next_salary,
       LAG(salary, 1, 0) OVER (ORDER BY hired_at) AS prev_or_zero    -- default 0
FROM employees;

# FIRST_VALUE / LAST_VALUE / NTH_VALUE — careful with frame!
SELECT first_name, salary,
  FIRST_VALUE(salary) OVER (PARTITION BY dept_id ORDER BY salary DESC
                             ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) AS top,
  LAST_VALUE(salary)  OVER (PARTITION BY dept_id ORDER BY salary DESC
                             ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) AS bottom
FROM employees;

# Cumulative / running aggregates
SELECT hired_at, salary,
       SUM(salary) OVER (ORDER BY hired_at ROWS UNBOUNDED PRECEDING) AS running_total,
       AVG(salary) OVER (ORDER BY hired_at ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) AS ma7
FROM employees;

# Frames:
#   ROWS BETWEEN n PRECEDING AND m FOLLOWING       physical row offsets
#   RANGE BETWEEN INTERVAL '7 days' PRECEDING AND CURRENT ROW    value-based (Postgres)
#   GROUPS BETWEEN ...                              peer-group based (Postgres 11+)

# Reusable WINDOW clause
SELECT first_name, salary,
       RANK()       OVER w AS rk,
       DENSE_RANK() OVER w AS drk
FROM employees
WINDOW w AS (PARTITION BY dept_id ORDER BY salary DESC);
```

## Common Table Expressions (CTEs)

```bash
# Basic CTE — improves readability vs nested subqueries
WITH high_earners AS (
  SELECT * FROM employees WHERE salary > 80000
)
SELECT department, COUNT(*) FROM high_earners GROUP BY department;

# Multiple CTEs (comma-separated)
WITH dept_stats AS (
  SELECT department, AVG(salary) AS avg_sal FROM employees GROUP BY department
),
high_paying AS (
  SELECT department FROM dept_stats WHERE avg_sal > 70000
)
SELECT e.* FROM employees e
WHERE e.department IN (SELECT department FROM high_paying);

# Recursive CTE — reachability, hierarchies, sequences
# Anchor (non-recursive) UNION ALL recursive part referencing the CTE
WITH RECURSIVE org AS (
  -- anchor: top-level managers
  SELECT id, first_name, manager_id, 0 AS depth
  FROM employees WHERE manager_id IS NULL
  UNION ALL
  -- recursive step
  SELECT e.id, e.first_name, e.manager_id, o.depth + 1
  FROM employees e
  JOIN org o ON e.manager_id = o.id
)
SELECT * FROM org ORDER BY depth, first_name;

# Generate a series of dates (Postgres) — without recursion, use generate_series:
SELECT * FROM generate_series('2026-01-01'::date, '2026-12-31'::date, INTERVAL '1 day');

# Materialized vs not (Postgres):
#   - Pre-12: CTEs were ALWAYS materialized (optimizer fence) — sometimes slow
#   - 12+:    CTE is inlined unless you write WITH foo AS MATERIALIZED (...)
WITH foo AS MATERIALIZED (SELECT ...) SELECT ... FROM foo;
WITH foo AS NOT MATERIALIZED (SELECT ...) SELECT ... FROM foo;
```

## CASE Expression

```bash
# Searched CASE — independent WHEN conditions
SELECT first_name, salary,
  CASE
    WHEN salary >= 100000 THEN 'Senior'
    WHEN salary >=  60000 THEN 'Mid'
    ELSE                       'Junior'
  END AS band
FROM employees;

# Simple CASE — equality match against a single expression
SELECT first_name,
  CASE department
    WHEN 'Eng'    THEN 'Tech'
    WHEN 'Sales'  THEN 'Revenue'
    ELSE               'Other'
  END AS division
FROM employees;

# Use in WHERE / ORDER BY / GROUP BY
SELECT * FROM employees
ORDER BY CASE WHEN department = 'Eng' THEN 0 ELSE 1 END, last_name;

# Combine with aggregates for pivots
SELECT
  SUM(CASE WHEN department='Eng'   THEN salary ELSE 0 END) AS eng_total,
  SUM(CASE WHEN department='Sales' THEN salary ELSE 0 END) AS sales_total
FROM employees;
```

## INSERT

```bash
# Single row — explicit column list
INSERT INTO employees (first_name, last_name, salary, dept_id)
VALUES ('Jane', 'Doe', 75000, 3);

# Single row — positional (fragile; breaks if table changes)
INSERT INTO employees VALUES (DEFAULT, 'jane@x.com', 'Jane', 'Doe', 75000, 3, NULL, '2026-04-25', TRUE);

# Multiple rows — one round trip
INSERT INTO employees (first_name, last_name, salary) VALUES
  ('Jane', 'Doe',   75000),
  ('John', 'Smith', 68000),
  ('Mei',  'Lin',   92000);

# INSERT ... SELECT — copy from another table
INSERT INTO archive_employees (id, first_name, last_name, terminated_at)
SELECT id, first_name, last_name, NOW()
FROM employees
WHERE active = FALSE;

# RETURNING (Postgres / Oracle / SQLite 3.35+) — get back generated columns
INSERT INTO employees (first_name, last_name, salary)
VALUES ('Sam', 'Jones', 70000)
RETURNING id, hired_at;

# SQL Server: OUTPUT
INSERT INTO employees (first_name, last_name, salary)
OUTPUT INSERTED.id, INSERTED.hired_at
VALUES ('Sam', 'Jones', 70000);

# MySQL: LAST_INSERT_ID()
INSERT INTO employees (first_name, last_name, salary) VALUES ('Sam','Jones', 70000);
SELECT LAST_INSERT_ID();
```

## UPDATE

```bash
# Single column
UPDATE employees SET salary = salary * 1.10 WHERE department = 'Eng';

# Multiple columns
UPDATE employees SET department = 'Product', title = 'PM' WHERE id = 42;

# Update with subquery
UPDATE employees
SET salary = salary * 1.05
WHERE id IN (SELECT employee_id FROM kpi_top_quartile);

# UPDATE ... FROM (Postgres) — join semantics for UPDATE
UPDATE employees e
SET    salary = e.salary * (1 + r.raise_pct / 100.0)
FROM   raise_table r
WHERE  e.id = r.employee_id;

# UPDATE with JOIN (MySQL / SQL Server):
# MySQL:
UPDATE employees e JOIN raise_table r ON e.id = r.employee_id
SET e.salary = e.salary * (1 + r.raise_pct/100.0);
# SQL Server:
UPDATE e SET e.salary = e.salary * (1 + r.raise_pct/100.0)
FROM employees e JOIN raise_table r ON e.id = r.employee_id;

# RETURNING (Postgres / SQLite 3.35+)
UPDATE employees SET salary = salary * 1.10
WHERE department = 'Eng'
RETURNING id, salary;
```

Safety: ALWAYS dry-run UPDATE/DELETE in a transaction first.

```bash
BEGIN;
UPDATE accounts SET balance = 0;        -- forgot WHERE!
SELECT COUNT(*) FROM accounts WHERE balance = 0;     -- 1.5M rows? oh.
ROLLBACK;
```

## DELETE

```bash
DELETE FROM employees WHERE active = FALSE;

# DELETE ... USING (Postgres) — multi-table semantics
DELETE FROM employees e
USING   departments d
WHERE   e.dept_id = d.id AND d.closed = TRUE;

# MySQL multi-table delete:
DELETE e FROM employees e JOIN departments d ON e.dept_id = d.id WHERE d.closed = TRUE;

# RETURNING (Postgres / SQLite 3.35+)
DELETE FROM employees WHERE active = FALSE RETURNING id, email;

# DELETE everything — slow because per-row MVCC
DELETE FROM staging;          -- writes a tombstone for every row
TRUNCATE TABLE staging;        -- DDL: instant

# Soft delete (preferred for audit)
UPDATE employees SET deleted_at = NOW() WHERE id = 42;
-- queries: WHERE deleted_at IS NULL
```

## UPSERT / MERGE

```bash
# PostgreSQL: ON CONFLICT
INSERT INTO inventory (sku, qty) VALUES ('A1', 10)
ON CONFLICT (sku) DO UPDATE
SET qty = inventory.qty + EXCLUDED.qty;
-- EXCLUDED refers to the would-be-inserted row; inventory.qty is the existing row.

INSERT INTO inventory (sku, qty) VALUES ('A1', 10)
ON CONFLICT (sku) DO NOTHING;                     -- skip if already present

# Multiple constraints: target by constraint name
INSERT INTO ... ON CONFLICT ON CONSTRAINT inventory_sku_key DO UPDATE ...

# MySQL: ON DUPLICATE KEY UPDATE — fires for ANY unique violation
INSERT INTO inventory (sku, qty) VALUES ('A1', 10)
ON DUPLICATE KEY UPDATE qty = qty + VALUES(qty);
-- VALUES(col) refers to inserted; in MySQL 8.0.20+ use aliasing:
INSERT INTO inventory (sku, qty) VALUES ('A1', 10) AS new
ON DUPLICATE KEY UPDATE qty = inventory.qty + new.qty;

# SQLite: REPLACE (delete-then-insert; loses other column data) OR ON CONFLICT
INSERT INTO inventory (sku, qty) VALUES ('A1', 10)
ON CONFLICT(sku) DO UPDATE SET qty = qty + excluded.qty;

# Standard SQL MERGE (SQL Server, Oracle, Postgres 15+):
MERGE INTO inventory AS t
USING (VALUES ('A1', 10), ('B2', 5)) AS s(sku, qty)
   ON t.sku = s.sku
WHEN MATCHED      THEN UPDATE SET qty = t.qty + s.qty
WHEN NOT MATCHED  THEN INSERT (sku, qty) VALUES (s.sku, s.qty);
```

## Transactions

```bash
# Begin / commit / rollback
BEGIN;                                  -- Postgres / SQLite
START TRANSACTION;                       -- MySQL standard
BEGIN TRANSACTION;                       -- SQL Server (or just BEGIN TRAN)

UPDATE accounts SET balance = balance - 500 WHERE id = 1;
UPDATE accounts SET balance = balance + 500 WHERE id = 2;
COMMIT;                                  -- persist

-- if anything failed:
ROLLBACK;                                -- discard

# Savepoints — nested rollback
BEGIN;
UPDATE inventory SET qty = qty - 1 WHERE sku = 'A1';
SAVEPOINT before_ship;
INSERT INTO shipments (sku, qty) VALUES ('A1', 1);
ROLLBACK TO SAVEPOINT before_ship;       -- undo just the shipment
RELEASE SAVEPOINT before_ship;           -- discard the savepoint
COMMIT;

# Isolation levels (lowest → highest):
#   READ UNCOMMITTED  — dirty reads possible (most engines silently treat as RC)
#   READ COMMITTED    — Postgres default; no dirty reads, non-repeatable allowed
#   REPEATABLE READ   — MySQL InnoDB default; rows seen once won't change
#   SERIALIZABLE      — full isolation; may abort with serialization_failure
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;          -- per-tx
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;     -- session

# Read-only / deferrable
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY DEFERRABLE;   -- Postgres reporting

# Locking reads (Postgres / MySQL InnoDB)
SELECT * FROM accounts WHERE id = 1 FOR UPDATE;            -- exclusive row lock
SELECT * FROM accounts WHERE id = 1 FOR SHARE;             -- shared lock
SELECT ... FOR UPDATE SKIP LOCKED;                         -- queue worker pattern
SELECT ... FOR UPDATE NOWAIT;                              -- error if locked

# Autocommit
# Postgres / SQL Server: each statement auto-commits unless inside BEGIN
# MySQL: SET autocommit = 0;  -- start a session-wide transaction
# JDBC/ODBC drivers often default to autocommit = false
```

## Indexes

```bash
# Default index = B-tree
CREATE INDEX idx_emp_dept ON employees (dept_id);

# Composite (multi-column) — order matters!
#   Index on (a,b) supports queries on (a) and (a,b), NOT (b) alone
CREATE INDEX idx_emp_dept_salary ON employees (dept_id, salary DESC);

# UNIQUE — also enforces a constraint
CREATE UNIQUE INDEX idx_emp_email ON employees (email);

# Partial index (Postgres / SQL Server filtered index) — only index matching rows
CREATE INDEX idx_emp_active ON employees (last_name) WHERE active = TRUE;
CREATE INDEX idx_orders_pending ON orders (created_at) WHERE status = 'pending';

# Expression / function index — needed when WHERE uses a function
CREATE INDEX idx_emp_lower_email ON employees (LOWER(email));
-- now: WHERE LOWER(email) = 'jane@x.com' can use the index.

# Covering index — INCLUDE non-key columns to make index-only scans (Postgres 11+, SQL Server)
CREATE INDEX idx_emp_dept_cover ON employees (dept_id) INCLUDE (first_name, last_name, salary);

# Hash, GIN, GiST, BRIN (Postgres specialized)
CREATE INDEX idx_emp_data ON events USING GIN (data jsonb_path_ops);
CREATE INDEX idx_logs_ts  ON logs   USING BRIN (created_at);     -- huge tables, ranges

# Concurrent / online build — does not block writes
CREATE INDEX CONCURRENTLY idx_emp_dept ON employees (dept_id);   -- Postgres
ALTER TABLE employees ADD INDEX idx_dept (dept_id) ALGORITHM=INPLACE LOCK=NONE;  -- MySQL

# Drop
DROP INDEX idx_emp_dept;
DROP INDEX CONCURRENTLY IF EXISTS idx_emp_dept;                  -- Postgres
DROP INDEX idx_emp_dept ON employees;                            -- MySQL syntax

# When NOT to index
#   - Tiny tables (full scan is faster than index lookup)
#   - Very low-cardinality columns (e.g. boolean) without partial filter
#   - Columns rarely used in WHERE/JOIN/ORDER BY
#   - Heavy write tables where the index cost outweighs the read benefit
```

## Views

```bash
# Logical view — query expanded each time it is read
CREATE VIEW active_employees AS
  SELECT id, first_name, last_name, dept_id, salary
  FROM employees WHERE active = TRUE;

SELECT * FROM active_employees WHERE dept_id = 3;

# Replace (or create) atomically
CREATE OR REPLACE VIEW active_employees AS ... ;       -- Postgres / MySQL

# Drop
DROP VIEW IF EXISTS active_employees;
DROP VIEW active_employees CASCADE;                     -- drop dependent views/objects too

# Updatable views
#   Simple views (one table, no DISTINCT/GROUP BY/JOIN/UNION/window) are usually updatable
INSERT INTO active_employees (first_name, last_name, dept_id, salary) VALUES (...);

# WITH CHECK OPTION — enforce that updates stay in the view's filter
CREATE VIEW active_employees AS
  SELECT * FROM employees WHERE active = TRUE
  WITH CHECK OPTION;
-- INSERT/UPDATE that produces active=FALSE will be rejected.

# security_barrier — Postgres flag preventing leaky predicates around the view
CREATE VIEW v WITH (security_barrier) AS SELECT * FROM employees WHERE id = current_user_id();
```

## Materialized Views

```bash
# Pre-computed snapshot — physically stored
# Postgres:
CREATE MATERIALIZED VIEW mv_dept_stats AS
  SELECT department, COUNT(*) AS headcount, AVG(salary) AS avg_sal
  FROM employees GROUP BY department
WITH DATA;                              -- WITH NO DATA = create empty, populate later

REFRESH MATERIALIZED VIEW mv_dept_stats;                    -- blocks reads (ACCESS EXCLUSIVE)
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_dept_stats;       -- requires UNIQUE index, no blocking

CREATE UNIQUE INDEX ON mv_dept_stats (department);          -- needed for CONCURRENTLY

# Oracle: MATERIALIZED VIEW with REFRESH ON COMMIT / FAST refresh
# SQL Server: indexed views (CREATE VIEW ... WITH SCHEMABINDING + CREATE UNIQUE CLUSTERED INDEX)
# MySQL / SQLite: no native materialized views — emulate with a real table + scheduled INSERT/REPLACE
```

## Stored Procedures and Functions

```bash
# Postgres function (SQL body)
CREATE OR REPLACE FUNCTION add(a INT, b INT) RETURNS INT
LANGUAGE sql IMMUTABLE AS $$
  SELECT a + b;
$$;
SELECT add(2, 3);    -- 5

# Postgres function (PL/pgSQL — procedural)
CREATE OR REPLACE FUNCTION fn_raise(emp_id INT, pct NUMERIC) RETURNS NUMERIC
LANGUAGE plpgsql AS $$
DECLARE
  new_salary NUMERIC;
BEGIN
  UPDATE employees SET salary = salary * (1 + pct/100.0)
  WHERE id = emp_id
  RETURNING salary INTO new_salary;
  RETURN new_salary;
END;
$$;

# Function returning a set of rows
CREATE OR REPLACE FUNCTION top_earners(n INT) RETURNS SETOF employees
LANGUAGE sql STABLE AS $$
  SELECT * FROM employees ORDER BY salary DESC LIMIT n;
$$;
SELECT * FROM top_earners(5);

# Postgres procedure (no return value; can manage transactions in newer versions)
CREATE OR REPLACE PROCEDURE refresh_stats()
LANGUAGE plpgsql AS $$
BEGIN
  REFRESH MATERIALIZED VIEW mv_dept_stats;
END;
$$;
CALL refresh_stats();

# MySQL function / procedure
DELIMITER $$
CREATE FUNCTION add(a INT, b INT) RETURNS INT
DETERMINISTIC
BEGIN
  RETURN a + b;
END$$
DELIMITER ;
SELECT add(2, 3);

# SQL Server T-SQL
CREATE FUNCTION dbo.add(@a INT, @b INT) RETURNS INT AS
BEGIN
  RETURN @a + @b;
END;
SELECT dbo.add(2, 3);
```

OUT parameters and table-valued returns let procedures emit complex results without SELECT.

## Triggers

```bash
# Postgres — trigger function + trigger binding
CREATE OR REPLACE FUNCTION fn_set_updated_at() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at := NOW();
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_employees_updated_at
BEFORE UPDATE ON employees
FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

# MySQL
DELIMITER $$
CREATE TRIGGER trg_employees_updated_at
BEFORE UPDATE ON employees FOR EACH ROW
BEGIN
  SET NEW.updated_at = NOW();
END$$
DELIMITER ;

# Drop
DROP TRIGGER trg_employees_updated_at ON employees;       -- Postgres
DROP TRIGGER trg_employees_updated_at;                     -- MySQL

# When to use triggers
#   - Audit / change capture
#   - Maintaining denormalized columns (sum, count, last_updated)
#   - Enforcing complex invariants impossible via CHECK
# When to avoid
#   - Business logic — hidden, hard to test
#   - Cross-row computations on hot tables (lock contention)
#   - Anywhere debuggability matters more than DRY
```

## Constraints — Foreign Keys & Cascades

```bash
# Inline definition
CREATE TABLE orders (
  id          SERIAL PRIMARY KEY,
  customer_id INT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
  ...
);

# Table-level (preferred — easier to name)
CREATE TABLE orders (
  id          SERIAL PRIMARY KEY,
  customer_id INT NOT NULL,
  ...
  CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id)
    REFERENCES customers(id)
    ON DELETE CASCADE
    ON UPDATE NO ACTION
);

# Cascade options on parent change
#   ON DELETE / ON UPDATE
#     CASCADE       — propagate change to children
#     SET NULL      — clear FK column (must be nullable)
#     SET DEFAULT   — set FK column to its default
#     RESTRICT      — block immediately
#     NO ACTION     — block at constraint check (deferrable)

# Deferrable constraints — check at COMMIT instead of statement end (Postgres)
CREATE TABLE t (
  id INT PRIMARY KEY,
  parent_id INT REFERENCES t(id) DEFERRABLE INITIALLY DEFERRED
);
BEGIN;
SET CONSTRAINTS ALL DEFERRED;
INSERT INTO t (id, parent_id) VALUES (1, 2), (2, 1);     -- self-referencing
COMMIT;
```

## Schemas / Namespaces

```bash
# Postgres — schemas inside a database
CREATE SCHEMA reporting;
SET search_path TO reporting, public;     -- session-level resolution order
CREATE TABLE reporting.events (...);
SELECT * FROM reporting.events;

# Show / change current schema
SHOW search_path;
ALTER ROLE alice SET search_path = analytics, public;     -- persistent for that role

# MySQL / SQL Server — "schema" is essentially a database
USE analytics;            -- both
SELECT * FROM analytics.events;     -- fully qualified

# SQLite — no schemas, but ATTACH multiple databases as namespaces
ATTACH DATABASE '/path/other.db' AS other;
SELECT * FROM other.events;

# Move a table to another schema (Postgres)
ALTER TABLE old.events SET SCHEMA reporting;
```

## Permissions

```bash
# Roles / users
CREATE ROLE analytics_ro NOLOGIN;                                  -- group role
CREATE ROLE alice WITH LOGIN PASSWORD 's3cret!' IN ROLE analytics_ro;
GRANT analytics_ro TO bob;                                          -- add bob to the group

# Object permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON employees TO analytics_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA reporting TO analytics_ro;
GRANT USAGE ON SCHEMA reporting TO analytics_ro;
GRANT EXECUTE ON FUNCTION fn_raise TO alice;

# Default privileges — apply to FUTURE objects
ALTER DEFAULT PRIVILEGES IN SCHEMA reporting
  GRANT SELECT ON TABLES TO analytics_ro;

# Revoke
REVOKE INSERT ON employees FROM analytics_ro;
REVOKE ALL ON SCHEMA reporting FROM PUBLIC;        -- strip public default

# Inspect
\du                          -- psql: list roles
\dp                          -- psql: list table privileges
SHOW GRANTS FOR alice;       -- MySQL

# Postgres Row Level Security (RLS)
ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
CREATE POLICY emp_self ON employees
  USING (id = current_setting('app.user_id')::INT);     -- read filter
GRANT SELECT ON employees TO app_user;
-- now app_user only sees their own rows.
```

## EXPLAIN / EXPLAIN ANALYZE

```bash
# Postgres
EXPLAIN SELECT * FROM employees WHERE dept_id = 3;
EXPLAIN ANALYZE SELECT * FROM employees WHERE dept_id = 3;          -- actually runs it
EXPLAIN (ANALYZE, BUFFERS, VERBOSE, FORMAT JSON) SELECT ...;
-- look for: Seq Scan vs Index Scan vs Index Only Scan
--           rows= (estimate) vs actual rows= (real)
--           cost=startup..total
--           Buffers: shared hit/read
--           Hash Join / Merge Join / Nested Loop

# MySQL
EXPLAIN SELECT * FROM employees WHERE dept_id = 3;
EXPLAIN FORMAT=JSON SELECT ...;
EXPLAIN ANALYZE SELECT ...;          -- 8.0.18+
-- columns: type (ALL=full scan, ref/range/eq_ref=using index), key, rows, Extra (Using where/Using index)

# SQLite
EXPLAIN QUERY PLAN SELECT * FROM employees WHERE dept_id = 3;
-- look for: SCAN (full) vs SEARCH (using index)

# SQL Server
SET SHOWPLAN_TEXT ON; SELECT ...; SET SHOWPLAN_TEXT OFF;
SET STATISTICS IO, TIME ON; SELECT ...;

# Reading plans:
#   Seq Scan / Table Scan        — full table read; fine for small tables
#   Index Scan                   — uses index but reads rows from heap
#   Index Only Scan / Using index — answer entirely from index (covering)
#   Nested Loop                  — good for small inner side
#   Hash Join                    — good for big tables; hash build cost
#   Merge Join                   — both sides sorted; great with index order
```

## Common Functions — String

```bash
# Concatenation — three flavors:
# Standard:    'Hello' || ' ' || 'World'        (Postgres, SQLite, Oracle)
# CONCAT():    CONCAT('Hello', ' ', 'World')    (Postgres, MySQL, SQL Server, Oracle)
# +:           'Hello' + ' ' + 'World'          (SQL Server only)
# MySQL:       CONCAT() yes; || means OR by default (set sql_mode=PIPES_AS_CONCAT to enable)

# Length
SELECT LENGTH('héllo');        -- BYTES in MySQL, CHARACTERS in Postgres/SQLite/SQL Server (mostly)
SELECT CHAR_LENGTH('héllo');   -- characters everywhere — preferred

# Substring
SUBSTRING('hello world' FROM 1 FOR 5);   -- standard
SUBSTRING('hello world', 1, 5);          -- accepted everywhere
SUBSTR('hello world', 1, 5);             -- SQLite, Oracle, MySQL

# Trim
TRIM('  hello  ');                       -- both ends
LTRIM('  hello');
RTRIM('hello  ');
TRIM(BOTH 'x' FROM 'xxhelloxx');         -- standard, removes 'x' chars
TRIM(LEADING '0' FROM '0042');           -- '42'

# Case
UPPER('hello');                           -- HELLO
LOWER('HELLO');                           -- hello
INITCAP('hello world');                   -- 'Hello World' (Postgres, Oracle)

# Search & replace
POSITION('lo' IN 'hello');               -- 4 (1-indexed)
INSTR('hello', 'lo');                    -- MySQL/SQLite/Oracle
CHARINDEX('lo', 'hello');                -- SQL Server
REPLACE('hello', 'l', 'r');              -- 'herro'
TRANSLATE('hello', 'el', 'ip');          -- 'hippo' — char-by-char map

# Padding
LPAD('42', 5, '0');                      -- '00042'
RPAD('42', 5, '.');                      -- '42...'

# Splitting / parsing
SPLIT_PART('a,b,c', ',', 2);             -- 'b'        (Postgres)
SUBSTRING_INDEX('a,b,c', ',', 2);         -- 'a,b'      (MySQL)
STRING_SPLIT('a,b,c', ',');               -- table     (SQL Server 2016+)

# Regex match (dialect-specific)
'hello world' ~ 'world'                   -- Postgres boolean
REGEXP_LIKE('hello', '^h');                -- Oracle / Postgres / MySQL 8+
REGEXP_REPLACE('foo123', '[0-9]+', 'X');   -- 'fooX'
```

## Common Functions — Date

```bash
# Now / today
SELECT CURRENT_DATE, CURRENT_TIMESTAMP, NOW();        -- Postgres / standard
SELECT CURRENT_TIMESTAMP, GETDATE(), SYSDATETIME();    -- SQL Server
SELECT NOW(), CURDATE(), CURTIME();                    -- MySQL
SELECT date('now'), datetime('now'), datetime('now','localtime');     -- SQLite
SELECT SYSDATE, CURRENT_DATE FROM dual;                -- Oracle

# Truncation — round to a unit
DATE_TRUNC('month', occurred);                         -- Postgres
DATE_TRUNC('hour',  occurred);
DATE_FORMAT(occurred, '%Y-%m-01');                     -- MySQL (string back)
strftime('%Y-%m', occurred);                           -- SQLite
DATEFROMPARTS(YEAR(occurred), MONTH(occurred), 1);     -- SQL Server

# Extract part
EXTRACT(YEAR FROM occurred);                            -- standard / Postgres
DATE_PART('year', occurred);                            -- Postgres helper
YEAR(occurred);                                          -- MySQL / SQL Server / Oracle
strftime('%Y', occurred);                                -- SQLite

# Intervals / arithmetic
SELECT NOW() + INTERVAL '1 day';                        -- Postgres / standard
SELECT DATE_ADD(NOW(), INTERVAL 1 DAY);                  -- MySQL
SELECT DATEADD(day, 1, GETDATE());                       -- SQL Server
SELECT date('now', '+1 day');                            -- SQLite
SELECT SYSDATE + 1 FROM dual;                            -- Oracle (days)

# Difference
SELECT AGE(end_at, start_at);                            -- Postgres — INTERVAL
SELECT DATEDIFF(end_at, start_at);                       -- MySQL — days
SELECT DATEDIFF(day, start_at, end_at);                   -- SQL Server — pick unit
SELECT julianday(end_at) - julianday(start_at);           -- SQLite — days

# Parse / format
SELECT TO_DATE('2026-04-25', 'YYYY-MM-DD');              -- Postgres / Oracle
SELECT STR_TO_DATE('2026-04-25', '%Y-%m-%d');             -- MySQL
SELECT CAST('2026-04-25' AS DATE);                        -- standard
SELECT TO_CHAR(NOW(), 'YYYY-MM-DD HH24:MI:SS');           -- Postgres / Oracle
SELECT DATE_FORMAT(NOW(), '%Y-%m-%d %H:%i:%s');           -- MySQL
SELECT FORMAT(GETDATE(), 'yyyy-MM-dd HH:mm:ss');          -- SQL Server

# Epoch
EXTRACT(EPOCH FROM NOW())                                 -- Postgres seconds (float)
UNIX_TIMESTAMP()                                          -- MySQL
DATEDIFF(SECOND, '1970-01-01', GETUTCDATE())               -- SQL Server
strftime('%s', 'now')                                      -- SQLite (text)
```

## Common Functions — Numeric

```bash
ABS(-42)                  -- 42
ROUND(3.567, 2)           -- 3.57
ROUND(3.567)              -- 4 (some dialects need explicit 0)
FLOOR(3.9)                -- 3
CEIL(3.1)  /  CEILING(3.1)
TRUNC(3.9)                -- 3 (Postgres, Oracle); MySQL TRUNCATE(3.9, 0)

MOD(10, 3)                -- 1
10 % 3                     -- works in Postgres, MySQL, SQL Server (NOT SQLite/Oracle)

POWER(2, 10)              -- 1024
SQRT(16)                  -- 4
EXP(1)                    -- 2.71828...
LN(2.71828)               -- ~1   (natural log)
LOG(10, 1000)             -- 3    (Postgres: LOG(base, x))
LOG10(1000)               -- 3    (MySQL / SQL Server)

PI()
RANDOM()                  -- Postgres: 0..1
RAND()                    -- MySQL / SQL Server
ROUND(RANDOM() * 100)::INT

GREATEST(3, 7, 2)         -- 7
LEAST(3, 7, 2)            -- 2

SIGN(-5)                  -- -1
```

## Common Functions — Type Conversion

```bash
# Standard CAST
SELECT CAST('42'  AS INT);
SELECT CAST(NOW() AS DATE);
SELECT CAST('2026-04-25' AS TIMESTAMPTZ);

# Postgres shorthand
SELECT '42'::INT;
SELECT NOW()::DATE;
SELECT '{"a":1}'::JSONB;

# SQL Server / MySQL
SELECT CONVERT(INT, '42');
SELECT CONVERT(VARCHAR, GETDATE(), 23);                -- 23 = ISO yyyy-mm-dd

# Oracle / Postgres parsing
SELECT TO_NUMBER('1,234.56', '9,999.99');
SELECT TO_DATE('2026-04-25', 'YYYY-MM-DD');
SELECT TO_TIMESTAMP('2026-04-25 10:30', 'YYYY-MM-DD HH24:MI');

# Failed cast behavior
# Postgres: ERROR: invalid input syntax — query aborts
# MySQL:    truncates and warns by default; STRICT_ALL_TABLES sql_mode raises an error
# SQLite:   silently coerces — '12abc' as INT becomes 12; non-numeric becomes 0
# SQL Server: ERROR: Conversion failed
# Postgres 16+ safe cast:
SELECT pg_input_is_valid('not a number','integer');      -- false (no error)
```

## NULL Handling

```bash
# NULL is "unknown" — never equals anything, including itself.
SELECT NULL = NULL;             -- NULL (not TRUE)
SELECT NULL = 1;                -- NULL
SELECT NULL <> 1;               -- NULL

# Use IS NULL / IS NOT NULL.
WHERE x IS NULL
WHERE x IS NOT NULL

# IS DISTINCT FROM — NULL-aware comparison (Postgres / standard)
WHERE a IS DISTINCT FROM b           -- TRUE if values differ OR exactly one is NULL
WHERE a IS NOT DISTINCT FROM b       -- TRUE if equal OR both NULL

# COALESCE — first non-NULL
SELECT COALESCE(nickname, first_name, 'Anonymous') FROM users;

# NULLIF — return NULL if two values match (avoid divide-by-zero, etc)
SELECT amount / NULLIF(qty, 0) FROM orders;     -- NULL instead of error when qty=0

# Dialect equivalents
ISNULL(x, 'default')               -- SQL Server
IFNULL(x, 'default')                -- MySQL / SQLite
NVL(x, 'default')                   -- Oracle

# NULL in arithmetic — anything OP NULL = NULL
SELECT 1 + NULL;        -- NULL
SELECT 'a' || NULL;      -- NULL (Postgres)   — note: CONCAT in MySQL skips NULLs

# NULL in aggregates — ignored, except COUNT(*)
COUNT(*)                  -- counts every row
COUNT(col)                -- counts non-NULL values
SUM(col)                  -- skips NULL; returns NULL if every input is NULL
AVG(col)                  -- skips NULL in numerator AND denominator

# NULL in DISTINCT / GROUP BY — treated as a single bucket (i.e. all NULLs together)
SELECT DISTINCT manager_id FROM employees;     -- one row for NULL
```

## JSON / JSONB Operations

```bash
# Postgres operators (JSONB recommended over JSON for storage + indexing)
SELECT data->'user'      FROM events;          -- json (object/array)
SELECT data->'user'->>'name' FROM events;      -- text
SELECT data#>'{user,addr,city}' FROM events;   -- nested path -> json
SELECT data#>>'{user,addr,city}' FROM events;  -- nested path -> text

# Containment / existence
SELECT * FROM events WHERE data @> '{"status":"ok"}';        -- contains
SELECT * FROM events WHERE data <@ '{"a":1,"b":2}';           -- contained
SELECT * FROM events WHERE data ?  'user';                     -- top-level key exists
SELECT * FROM events WHERE data ?| array['user','admin'];      -- any of these keys
SELECT * FROM events WHERE data ?& array['user','admin'];      -- all of these keys

# Build / modify
SELECT jsonb_build_object('a', 1, 'b', 'two');                -- {"a":1, "b":"two"}
SELECT jsonb_set(data, '{user,name}', '"Alice"', TRUE) FROM events;
SELECT data || '{"new":"key"}'::jsonb FROM events;            -- shallow merge
SELECT data - 'old_key' FROM events;                            -- remove key
SELECT data #- '{user,addr}' FROM events;                       -- remove path

# Iterate
SELECT key, value FROM jsonb_each(data);
SELECT * FROM jsonb_array_elements(data->'tags');

# Indexing JSONB (GIN)
CREATE INDEX idx_events_data ON events USING GIN (data jsonb_path_ops);   -- @>, ? queries
CREATE INDEX idx_events_user ON events ((data->>'user_id'));               -- exact field

# MySQL JSON
SELECT JSON_EXTRACT(data, '$.user.name');     SELECT data->'$.user.name';
SELECT data->>'$.user.name';                                           -- 5.7+ unquoted
SELECT JSON_SET(data, '$.user.name', 'Alice');
SELECT JSON_REMOVE(data, '$.old');
WHERE JSON_CONTAINS(data, '"ok"', '$.status')

# SQL Server / standard SQL
SELECT JSON_VALUE(data, '$.user.name');
SELECT JSON_QUERY(data, '$.user');               -- subobject/array
WHERE  JSON_VALUE(data, '$.status') = 'ok'

# SQLite JSON1
SELECT json_extract(data, '$.user.name'), data->'user'->>'name';        -- 3.38+ operators
```

## Common Errors and Fixes

Verbatim error messages and the actual cause + fix.

```bash
# 1) Postgres: ERROR: column "Email" does not exist
# Cause: unquoted identifiers are folded to lowercase. The column is "email".
# BAD:
SELECT Email FROM users;                  -- becomes lower: email — fine if column is email
SELECT "Email" FROM users;                -- looks for case-sensitive "Email"
# FIX: stop quoting identifiers, or rename the column.

# 2) Postgres: ERROR: relation "Users" does not exist
# Cause: someone created the table with double-quotes in pgAdmin etc., making it case-sensitive,
#        OR the table is in another schema not on search_path.
# FIX: SELECT * FROM "Users";    or   SET search_path = app, public;

# 3) duplicate key value violates unique constraint "users_email_key"
# DETAIL:  Key (email)=(jane@x.com) already exists.
# Cause: INSERT/UPDATE collides with a UNIQUE.
# FIX: use ON CONFLICT (email) DO NOTHING / DO UPDATE; or check before insert; or
#      let it fail and surface a 409 to the user.
INSERT INTO users (email) VALUES ('jane@x.com')
  ON CONFLICT (email) DO NOTHING;

# 4) insert or update on table "orders" violates foreign key constraint "fk_orders_customer"
# DETAIL: Key (customer_id)=(99) is not present in table "customers".
# Cause: parent row missing.
# FIX: insert parent first, or run inside one transaction with parent insert.

# 5) deadlock detected — Process X waits for ShareLock; Process Y waits for ShareLock
# Cause: two transactions taking row locks in opposite order.
# FIX: lock rows in a deterministic order (e.g. ORDER BY id FOR UPDATE) and keep transactions short.

# 6) ERROR: null value in column "email" of relation "users" violates not-null constraint
# Cause: omitted a NOT NULL column without a default.
# FIX: include the column or set a DEFAULT.

# 7) ERROR: division by zero
# FIX: guard with NULLIF — total / NULLIF(qty, 0).

# 8) ERROR: cannot drop table employees because other objects depend on it
# Cause: views / FKs reference it.
# FIX: DROP TABLE employees CASCADE — but make sure you understand the blast radius.

# 9) ERROR: syntax error at or near "USER"
# Cause: reserved word as identifier without quotes.
# FIX: rename the column, or quote it: "user". Better: never use reserved words as names.

# 10) MySQL: ERROR 1093 (HY000): You can't specify target table 't' for update in FROM clause
# Cause: subquery references the same table you're updating.
# FIX: wrap the subquery in another SELECT — MySQL won't see it as the same:
UPDATE t SET status='x'
WHERE id IN (SELECT id FROM (SELECT id FROM t WHERE created < NOW() - INTERVAL 30 DAY) AS s);

# 11) SQLite: Error: database is locked
# Cause: long-running transaction; second writer waits.
# FIX: shorten transactions; PRAGMA busy_timeout=5000; consider PRAGMA journal_mode=WAL.

# 12) SQL Server: Cannot insert explicit value for identity column in table 't' when IDENTITY_INSERT is set to OFF.
# FIX: don't supply the identity column, or:  SET IDENTITY_INSERT t ON;

# 13) Postgres: ERROR: more than one row returned by a subquery used as an expression
# Cause: scalar subquery yielded multiple rows.
# FIX: use LIMIT 1 with ORDER BY, or rewrite as JOIN.
```

## SQL Injection

```bash
# NEVER concatenate user input into SQL strings.

# BAD — Python f-string into SQL:
query = f"SELECT * FROM users WHERE email = '{user_email}'"
# attacker supplies:  ' OR '1'='1
# results in:  SELECT * FROM users WHERE email = '' OR '1'='1'

# GOOD — parameterized query (driver does the escaping; types preserved):
# Python (psycopg / sqlite3 / mysql-connector):
cur.execute("SELECT * FROM users WHERE email = %s", (user_email,))     -- psycopg / mysql
cur.execute("SELECT * FROM users WHERE email = ?",  (user_email,))      -- sqlite3
# Postgres native ($N):
cur.execute("SELECT * FROM users WHERE email = $1", (user_email,))
# Named (sqlalchemy / Oracle):
cur.execute("SELECT * FROM users WHERE email = :email", {"email": user_email})

# When the variable is an IDENTIFIER (table/column name) — bind params won't help:
# Postgres function:
EXECUTE format('SELECT %I FROM %I WHERE id = $1', col_name, tbl_name) USING the_id;
# format spec:   %I = identifier (quotes if needed),  %L = literal value,  %s = raw (DANGEROUS)
# Or quote_ident() and quote_literal() from PL/pgSQL.

# Least-privilege: app DB user should not be a superuser. SELECT/INSERT/UPDATE/DELETE on
# its tables only. No CREATE / DROP / ALTER. No access to pg_user / mysql.user.

# Detect / mitigate:
#   - Audit logs / pgAudit / MySQL audit plugin
#   - WAFs catch trivial cases — never rely on them
#   - Input validation is defense in depth, not a substitute for parameterization
```

## Common Gotchas

```bash
# 1) NULL in NOT IN excludes EVERYTHING
# BAD:
SELECT * FROM employees WHERE manager_id NOT IN (SELECT id FROM contractors);
-- if any contractor.id is NULL, the result is empty (NULL = unknown).
# FIX: use NOT EXISTS or filter out NULLs.
SELECT * FROM employees e
WHERE NOT EXISTS (SELECT 1 FROM contractors c WHERE c.id = e.manager_id);

# 2) GROUP BY without all non-aggregated columns
# BAD (ANSI):
SELECT department, last_name, AVG(salary) FROM employees GROUP BY department;
-- last_name is undetermined. Postgres / SQL Server reject this.
-- MySQL with default sql_mode allows it (returns ANY value!).
# FIX: include all non-aggregated cols or wrap in an aggregate / DISTINCT ON.
SELECT department, last_name, AVG(salary) OVER (PARTITION BY department) FROM employees;

# 3) LIMIT without ORDER BY is non-deterministic
# BAD:
SELECT * FROM events LIMIT 10;            -- different run can return different rows
# FIX: always pair LIMIT with ORDER BY on a stable key.
SELECT * FROM events ORDER BY id DESC LIMIT 10;

# 4) Case sensitivity rules differ
# Postgres:    unquoted identifiers fold to LOWERCASE; quoted preserve case.
# MySQL:       table names are case-sensitive on Linux (filesystem-driven), insensitive on macOS/Windows.
#              column names are insensitive everywhere by default.
# SQL Server:  default collation is case-INsensitive — 'Foo' = 'foo' returns true.
# SQLite:      identifiers insensitive; LIKE is case-insensitive for ASCII by default,
#              fix: PRAGMA case_sensitive_like = 1.

# 5) Implicit type conversion surprises
# BAD: indexed column on the right of a function — index ignored.
WHERE LOWER(email) = 'jane@x.com'         -- index on email won't help
# FIX: expression index, or store normalized form, or use case-insensitive collation.
CREATE INDEX idx_users_lower_email ON users (LOWER(email));

# 6) Integer division
SELECT 5 / 2;          -- 2 in Postgres (int/int), 2.5 in SQLite (silent float promotion in some)
SELECT 5.0 / 2;         -- 2.5 (Postgres)
# FIX: cast one side  ->  5::numeric / 2

# 7) String length surprise (MySQL):
SELECT LENGTH('héllo');         -- 6 (bytes, utf8mb4)
SELECT CHAR_LENGTH('héllo');     -- 5 (chars)
# Always use CHAR_LENGTH for character counts.

# 8) DATE arithmetic differs:
SELECT '2026-01-31'::DATE + 1;          -- '2026-02-01' (Postgres: integer = days)
SELECT '2026-01-31'::DATE + INTERVAL '1 month';      -- '2026-02-28' (clamped)
SELECT DATE_ADD('2026-01-31', INTERVAL 1 MONTH);     -- '2026-02-28' (MySQL clamps too)

# 9) Boolean coercion
WHERE active                  -- Postgres OK (boolean)
WHERE active = 1              -- MySQL TINYINT(1)
WHERE active = 'true'         -- string vs bool — DON'T

# 10) Unique violation under SERIALIZABLE
# Two transactions both INSERT same key — one will fail with serialization_failure
# under SERIALIZABLE. Retry the failing transaction.
```

## Performance Patterns

```bash
# 1) Avoid SELECT * in queries that go through indexes.
#    A covering index on (a) INCLUDE (b, c) only helps if you SELECT a,b,c — not *.

# 2) Batch INSERTs — one round trip.
INSERT INTO t (col) VALUES (1),(2),(3),...,(1000);
-- vs 1000 separate INSERTs (way slower).

# 3) Bulk loads — bypass the SQL layer when you can.
# Postgres:   COPY t FROM STDIN WITH (FORMAT csv);    (or \copy from psql)
# MySQL:      LOAD DATA INFILE '/path/file.csv' INTO TABLE t FIELDS TERMINATED BY ',';
# SQLite:     .mode csv  +  .import file.csv t
# SQL Server: BULK INSERT t FROM 'file.csv' WITH (FIELDTERMINATOR=',');
#             BCP utility for the largest jobs.

# 4) LIMIT in subquery before joining big tables
# BAD: join 10M-row table to 100k-row table, then take top 10
# GOOD:
WITH top10 AS (
  SELECT id FROM big ORDER BY created DESC LIMIT 10
)
SELECT * FROM top10 t JOIN small s ON s.big_id = t.id;

# 5) Window function instead of self-join
# BAD: correlated subquery for "rank per dept"
# GOOD: ROW_NUMBER() OVER (PARTITION BY ...).

# 6) Partial indexes for hot subsets
CREATE INDEX idx_orders_open ON orders (created_at) WHERE status = 'open';
-- queries on open orders never touch closed-order index entries.

# 7) The N+1 anti-pattern (ORM)
# BAD:    for u in users: get_orders(u.id)             -- 1 + N queries
# GOOD:   single JOIN or "WHERE user_id IN (...)"      -- 1 query
SELECT u.id, u.name, o.id, o.total
FROM users u JOIN orders o ON o.user_id = u.id
WHERE u.id IN (1,2,3,...);

# 8) DON'T wrap indexed columns in functions
WHERE DATE(created_at) = '2026-04-25'              -- index can't help
WHERE created_at >= '2026-04-25' AND created_at < '2026-04-26'   -- index works

# 9) DON'T use OR across different columns — many planners can't combine indexes
WHERE email = 'x' OR phone = 'y'                   -- often slow
-- alt: UNION ALL of two indexed selects.

# 10) Statistics & ANALYZE
ANALYZE employees;                                 -- Postgres / SQLite refresh stats
ANALYZE TABLE employees;                            -- MySQL
UPDATE STATISTICS employees;                        -- SQL Server
-- queries depend on planner stats. After a big load, ANALYZE.
```

## Pagination

```bash
# OFFSET pagination — simple, but slows linearly with depth.
SELECT * FROM events ORDER BY id LIMIT 20 OFFSET 0;
SELECT * FROM events ORDER BY id LIMIT 20 OFFSET 20;
SELECT * FROM events ORDER BY id LIMIT 20 OFFSET 100000;     -- DB must skip 100k rows.

# Keyset / seek-method pagination — O(log n) per page, regardless of depth.
# Page 1:
SELECT * FROM events ORDER BY id DESC LIMIT 20;
# Page 2 (id_min = smallest id from page 1):
SELECT * FROM events WHERE id < :id_min ORDER BY id DESC LIMIT 20;

# Multi-column keyset (when ORDER BY isn't unique):
SELECT * FROM events
WHERE (created_at, id) < (:last_created, :last_id)
ORDER BY created_at DESC, id DESC
LIMIT 20;

# Cursor-based — server-side cursor (Postgres):
BEGIN;
DECLARE c CURSOR FOR SELECT * FROM events ORDER BY id;
FETCH FORWARD 20 FROM c;
FETCH FORWARD 20 FROM c;
CLOSE c;
COMMIT;
-- nice for stable iteration across long results; transaction must stay open.

# Total-count tradeoff:
-- Showing "Page 5 of 9,238" requires SELECT COUNT(*) — expensive on big tables.
-- Skip total counts; show "next" arrow + estimated count if needed.
```

## Idioms

```bash
# Top-N per group
# Portable (window function):
WITH ranked AS (
  SELECT *, ROW_NUMBER() OVER (PARTITION BY dept_id ORDER BY salary DESC) AS rn FROM employees
)
SELECT * FROM ranked WHERE rn = 1;

# Postgres shortcut:
SELECT DISTINCT ON (dept_id) * FROM employees ORDER BY dept_id, salary DESC;

# Running total
SELECT day, amount,
  SUM(amount) OVER (ORDER BY day ROWS UNBOUNDED PRECEDING) AS cumulative
FROM sales;

# Moving average
SELECT day, amount,
  AVG(amount) OVER (ORDER BY day ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) AS ma7
FROM sales;

# Gap-and-island detection — find consecutive runs
WITH grp AS (
  SELECT id, status, day,
         day - INTERVAL '1 day' * ROW_NUMBER() OVER (PARTITION BY status ORDER BY day) AS g
  FROM events
)
SELECT status, MIN(day), MAX(day), COUNT(*) AS run_length
FROM grp
GROUP BY status, g
ORDER BY status, MIN(day);

# Recursive hierarchy traversal (already covered above)

# Pivot via conditional aggregation (portable)
SELECT
  SUM(CASE WHEN year=2024 THEN revenue END) AS y2024,
  SUM(CASE WHEN year=2025 THEN revenue END) AS y2025,
  SUM(CASE WHEN year=2026 THEN revenue END) AS y2026
FROM finance;

# UNION ALL > UNION when you don't need dedup — saves a sort/hash pass.

# EXISTS > IN for big subqueries — short-circuits on first match.
WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)

# Anti-join via NOT EXISTS / LEFT JOIN ... IS NULL
SELECT u.* FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE o.id IS NULL;
```

## Tools

```bash
# Server CLIs
psql        # postgres official client; tab-completion, \-commands, \timing
mysql       # mysql official client
mariadb     # mariadb fork client
sqlite3     # bundled with sqlite
sqlcmd      # SQL Server
sqlplus     # Oracle (rare today)

# Friendlier CLIs (autocomplete, syntax highlight)
pgcli       # for postgres   — pip install pgcli
mycli       # for mysql      — pip install mycli
litecli     # for sqlite     — pip install litecli
mssql-cli   # for SQL Server — pip install mssql-cli

# GUIs
DBeaver     # free, JDBC, all dialects, ER diagrams
DataGrip    # JetBrains, paid, best DDL editor
TablePlus   # macOS / Windows, fast, paid
pgAdmin     # web UI for postgres
MySQL Workbench
Beekeeper Studio
Postico     # macOS / postgres, simple
DBVisualizer

# Linting / formatting
sqlfluff    # python; pluggable dialect — lints + auto-format
sqlfluff lint  myfile.sql --dialect postgres
sqlfluff fix   myfile.sql --dialect postgres
pgFormatter  # perl; nice for pg-flavored SQL
sql-formatter # node, dialect-aware

# Migrations
flyway / liquibase / sqitch / alembic / atlas / dbmate / goose / migrate
-- track schema state via versioned files; never edit prod schema by hand.

# ORMs (be SQL-aware regardless)
SQLAlchemy (Python), Django ORM, ActiveRecord (Rails), Prisma / TypeORM (Node),
GORM (Go), sqlx (Go raw), Diesel (Rust), Hibernate (Java), Entity Framework (.NET).
-- Always log generated SQL during development. ORMs make it easy to write N+1.
```

## Tips

- Parameterize every user-supplied value. Concatenation is the SQL injection path.
- Default to explicit column lists in production code. `SELECT *` is for the REPL.
- Always pair `LIMIT` with `ORDER BY`. Otherwise the order is "whatever the planner felt like."
- Wrap multi-statement changes in transactions and dry-run with `ROLLBACK` the first time.
- Index columns used in `WHERE`, `JOIN`, and `ORDER BY` — but never index just because.
- `EXPLAIN ANALYZE` is your friend. Compare estimated vs actual rows to find bad statistics.
- `DECIMAL/NUMERIC` for money; never `REAL/FLOAT/DOUBLE`.
- Use `TIMESTAMPTZ`/`DATETIMEOFFSET` and store UTC. Convert at the edges.
- Name your constraints. You will be grateful at 2am.
- Treat `NULL` as the third boolean state: TRUE / FALSE / unknown. Don't compare it with `=`.
- Migrations are forwards-only. Every change goes through a versioned migration file.
- Keep transactions short. Long transactions in MVCC databases bloat the WAL/undo segment.
- Prefer `EXISTS` over `IN` for large subqueries; prefer `UNION ALL` over `UNION` when dedup is unnecessary.
- For deep pagination, use keyset / seek pagination — `OFFSET 100000` is a tax on every page.
- The most efficient query is the one you don't run. Cache aggregates, denormalize hot reads, materialize views.
- Read the dialect docs before assuming portability — every database lies about being "standard SQL."

## See Also

- postgresql
- mysql
- sqlite
- redis
- dynamodb
- bash
- regex
- json
- awk
- polyglot

## References

- [PostgreSQL — SQL Reference](https://www.postgresql.org/docs/current/sql.html)
- [PostgreSQL — Functions and Operators](https://www.postgresql.org/docs/current/functions.html)
- [PostgreSQL — Indexes](https://www.postgresql.org/docs/current/indexes.html)
- [PostgreSQL — Using EXPLAIN](https://www.postgresql.org/docs/current/using-explain.html)
- [MySQL Reference Manual](https://dev.mysql.com/doc/refman/8.0/en/)
- [MySQL — SQL Statement Syntax](https://dev.mysql.com/doc/refman/8.0/en/sql-statements.html)
- [MariaDB Knowledge Base](https://mariadb.com/kb/en/)
- [SQLite — SQL Language Reference](https://www.sqlite.org/lang.html)
- [SQLite — Quirks, caveats, and gotchas](https://www.sqlite.org/quirks.html)
- [Microsoft — Transact-SQL Reference](https://learn.microsoft.com/en-us/sql/t-sql/language-reference)
- [Oracle SQL Language Reference](https://docs.oracle.com/en/database/oracle/oracle-database/19/sqlrf/index.html)
- [SQL Standard (ISO/IEC 9075)](https://www.iso.org/standard/76583.html)
- [Modern SQL — what's new across dialects](https://modern-sql.com/)
- [Use The Index, Luke! — Markus Winand on indexing and tuning](https://use-the-index-luke.com/)
- [SQL Style Guide — Simon Holywell](https://www.sqlstyle.guide/)
- [pgexercises.com — interactive PostgreSQL practice](https://pgexercises.com/)
- [SQLBolt — interactive SQL tutorial](https://sqlbolt.com/)
