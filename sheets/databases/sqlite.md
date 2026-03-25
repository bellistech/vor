# SQLite (Embedded Database)

Self-contained, serverless, zero-configuration SQL database engine stored in a single file.

## Shell Commands

### Open a database

```bash
sqlite3 mydb.db
sqlite3 :memory:                   # in-memory database
```

### Dot commands

```bash
.tables                            # list all tables
.schema                            # show all CREATE statements
.schema users                      # show CREATE for one table
.headers on                        # show column headers
.mode column                       # columnar output
.mode csv                          # CSV output
.mode json                         # JSON output
.mode table                        # pretty table output
.separator "\t"                    # set separator for csv/list mode
.databases                         # list attached databases
.indices users                     # list indexes on table
.dump                              # dump entire database as SQL
.dump users                        # dump one table
.read script.sql                   # execute SQL file
.output results.csv                # redirect output to file
.output stdout                     # back to terminal
.quit                              # exit
.help                              # list all dot commands
```

### One-liner from shell

```bash
sqlite3 mydb.db "SELECT * FROM users;"
sqlite3 -header -csv mydb.db "SELECT * FROM users;" > users.csv
sqlite3 -json mydb.db "SELECT * FROM users;"
sqlite3 mydb.db ".dump" > backup.sql
```

## CREATE

```bash
# CREATE TABLE users (
#     id INTEGER PRIMARY KEY AUTOINCREMENT,
#     email TEXT UNIQUE NOT NULL,
#     name TEXT NOT NULL,
#     active INTEGER DEFAULT 1,
#     data TEXT,   -- JSON stored as text
#     created_at TEXT DEFAULT (datetime('now'))
# );

# CREATE TABLE IF NOT EXISTS orders (
#     id INTEGER PRIMARY KEY,
#     user_id INTEGER NOT NULL REFERENCES users(id),
#     amount REAL NOT NULL,
#     created_at TEXT DEFAULT (datetime('now'))
# );
```

### ALTER

```bash
# ALTER TABLE users ADD COLUMN phone TEXT;
# ALTER TABLE users RENAME TO customers;
# ALTER TABLE users RENAME COLUMN name TO full_name;
# ALTER TABLE users DROP COLUMN phone;   -- SQLite 3.35+
```

## CRUD

### INSERT

```bash
# INSERT INTO users (email, name) VALUES ('alice@example.com', 'Alice');
# INSERT OR REPLACE INTO users (id, email, name) VALUES (1, 'alice@example.com', 'Alice');
# INSERT OR IGNORE INTO users (email, name) VALUES ('alice@example.com', 'Alice');
# INSERT INTO users (email, name) VALUES ('bob@example.com', 'Bob')
#     RETURNING id, email;   -- SQLite 3.35+
```

### SELECT

```bash
# SELECT * FROM users WHERE active = 1 ORDER BY created_at DESC LIMIT 10;
# SELECT * FROM users LIMIT 10 OFFSET 20;
# SELECT u.name, SUM(o.amount) AS total
#     FROM users u JOIN orders o ON u.id = o.user_id
#     GROUP BY u.name HAVING total > 100;
```

### UPDATE

```bash
# UPDATE users SET name = 'Alice Smith' WHERE id = 1;
# UPDATE users SET active = 0
#     WHERE id IN (SELECT user_id FROM sessions WHERE last_seen < datetime('now', '-1 year'));
```

### DELETE

```bash
# DELETE FROM sessions WHERE expires_at < datetime('now');
```

## Indexes

```bash
# CREATE INDEX idx_users_email ON users (email);
# CREATE UNIQUE INDEX idx_users_email ON users (email);
# CREATE INDEX idx_orders_user_date ON orders (user_id, created_at);
# DROP INDEX idx_users_email;
```

### Check index usage

```bash
# EXPLAIN QUERY PLAN SELECT * FROM users WHERE email = 'alice@example.com';
```

## Backup

### SQL dump backup

```bash
sqlite3 mydb.db ".dump" > backup.sql
sqlite3 mydb.db ".dump" | gzip > backup.sql.gz
```

### Restore from dump

```bash
sqlite3 newdb.db < backup.sql
```

### Online backup (safe with WAL)

```bash
sqlite3 mydb.db ".backup backup.db"
```

### Copy while database is in use

```bash
sqlite3 mydb.db "VACUUM INTO 'backup.db';"
```

## WAL Mode

### Enable WAL (write-ahead logging)

```bash
# PRAGMA journal_mode=WAL;
```

### Check journal mode

```bash
# PRAGMA journal_mode;
```

WAL allows concurrent readers with one writer, much better than the default rollback journal.

## PRAGMAs

### Performance tuning

```bash
# PRAGMA journal_mode=WAL;           # concurrent reads + writes
# PRAGMA synchronous=NORMAL;         # safe with WAL, faster than FULL
# PRAGMA cache_size=-64000;          # 64MB page cache (negative = KB)
# PRAGMA mmap_size=268435456;        # 256MB memory-mapped I/O
# PRAGMA temp_store=MEMORY;          # temp tables in memory
# PRAGMA busy_timeout=5000;          # wait 5s on lock instead of failing
```

### Integrity check

```bash
# PRAGMA integrity_check;
# PRAGMA quick_check;
```

### Foreign keys (off by default)

```bash
# PRAGMA foreign_keys=ON;
```

### Database info

```bash
# PRAGMA table_info(users);
# PRAGMA database_list;
# PRAGMA page_size;
# PRAGMA page_count;
# PRAGMA compile_options;
```

## JSON Support

```bash
# SELECT json_extract(data, '$.name') FROM users;
# SELECT * FROM users WHERE json_extract(data, '$.role') = 'admin';
# SELECT json_each.value FROM users, json_each(users.tags);
# UPDATE users SET data = json_set(data, '$.verified', 1) WHERE id = 1;
# SELECT json_group_array(name) FROM users;
# SELECT json_group_object(name, email) FROM users;
```

## Date & Time

```bash
# datetime('now')                          # current UTC datetime
# datetime('now', 'localtime')             # local time
# date('now', '-7 days')                   # 7 days ago
# strftime('%Y-%m', created_at)            # format
# julianday('now') - julianday(created_at) # days between
```

## CLI Tricks

### Import CSV

```bash
sqlite3 mydb.db
# .mode csv
# .import users.csv users
```

### Export CSV

```bash
sqlite3 -header -csv mydb.db "SELECT * FROM users;" > users.csv
```

### Pretty output

```bash
sqlite3 mydb.db -header -table "SELECT * FROM users LIMIT 5;"
```

## Tips

- Always enable WAL mode (`PRAGMA journal_mode=WAL`) for any database accessed by more than one connection.
- Set `PRAGMA busy_timeout=5000` to avoid "database is locked" errors under contention.
- Foreign keys are disabled by default. Run `PRAGMA foreign_keys=ON` at the start of every connection.
- SQLite uses dynamic typing. A column declared `INTEGER` can hold text. Use `STRICT` tables (3.37+) to enforce types.
- `VACUUM INTO 'copy.db'` is the safest way to copy a live database.
- The `-json` flag on the CLI is handy for piping into `jq`.
- Use `EXPLAIN QUERY PLAN` (not `EXPLAIN`) to see whether queries use indexes.
- SQLite handles up to ~1TB databases and moderate write loads. For high write concurrency, consider PostgreSQL.

## References

- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [SQLite SQL Language Reference](https://www.sqlite.org/lang.html)
- [SQLite CLI (sqlite3)](https://www.sqlite.org/cli.html)
- [SQLite PRAGMA Statements](https://www.sqlite.org/pragma.html)
- [SQLite Write-Ahead Logging (WAL)](https://www.sqlite.org/wal.html)
- [SQLite STRICT Tables](https://www.sqlite.org/stricttables.html)
- [SQLite JSON Functions](https://www.sqlite.org/json1.html)
- [SQLite Full-Text Search (FTS5)](https://www.sqlite.org/fts5.html)
- [SQLite EXPLAIN QUERY PLAN](https://www.sqlite.org/eqp.html)
- [SQLite Limits](https://www.sqlite.org/limits.html)
- [SQLite GitHub Mirror](https://github.com/sqlite/sqlite)
