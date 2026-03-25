# PostgreSQL (Relational Database)

Advanced open-source relational database with JSONB, full-text search, and extensibility.

## psql Client

### Connect

```bash
psql -U postgres -d mydb
psql -h db.example.com -p 5432 -U admin -d production
psql "postgresql://admin:pass@db.example.com:5432/mydb?sslmode=require"
```

### Meta-commands

```bash
\l                  # list databases
\c mydb             # connect to database
\dt                 # list tables
\dt+                # list tables with sizes
\d users            # describe table
\di                 # list indexes
\dv                 # list views
\du                 # list roles/users
\dn                 # list schemas
\df                 # list functions
\x                  # toggle expanded output
\timing             # toggle query timing
\i script.sql       # execute SQL file
\copy users TO '/tmp/users.csv' CSV HEADER
\q                  # quit
```

## DDL (CREATE / ALTER / DROP)

### Tables

```bash
# CREATE TABLE users (
#     id SERIAL PRIMARY KEY,
#     email VARCHAR(255) UNIQUE NOT NULL,
#     name TEXT NOT NULL,
#     data JSONB DEFAULT '{}',
#     created_at TIMESTAMPTZ DEFAULT NOW()
# );

# ALTER TABLE users ADD COLUMN active BOOLEAN DEFAULT true;
# ALTER TABLE users ALTER COLUMN name SET NOT NULL;
# ALTER TABLE users RENAME COLUMN name TO full_name;
# ALTER TABLE users DROP COLUMN active;

# DROP TABLE IF EXISTS users CASCADE;
```

### Common types

```bash
# INTEGER, BIGINT, SERIAL, BIGSERIAL
# TEXT, VARCHAR(n), CHAR(n)
# BOOLEAN
# TIMESTAMPTZ, DATE, INTERVAL
# JSONB, JSON
# UUID (with uuid-ossp or gen_random_uuid())
# NUMERIC(10,2), REAL, DOUBLE PRECISION
# BYTEA, INET, CIDR, MACADDR
# ARRAY (e.g., TEXT[])
```

## DML (CRUD)

### INSERT

```bash
# INSERT INTO users (email, name) VALUES ('alice@example.com', 'Alice');
# INSERT INTO users (email, name) VALUES ('bob@example.com', 'Bob')
#     RETURNING id, email;
# INSERT INTO users (email, name) VALUES ('alice@example.com', 'Alice')
#     ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name;
```

### SELECT

```bash
# SELECT * FROM users WHERE active = true ORDER BY created_at DESC LIMIT 10;
# SELECT name, COUNT(*) FROM orders GROUP BY name HAVING COUNT(*) > 5;
# SELECT u.name, o.total FROM users u
#     JOIN orders o ON u.id = o.user_id
#     WHERE o.created_at > NOW() - INTERVAL '30 days';
```

### UPDATE

```bash
# UPDATE users SET name = 'Alice Smith' WHERE email = 'alice@example.com';
# UPDATE users SET active = false WHERE last_login < NOW() - INTERVAL '1 year'
#     RETURNING id, email;
```

### DELETE

```bash
# DELETE FROM sessions WHERE expires_at < NOW();
# DELETE FROM users WHERE id = 42 RETURNING *;
```

## Indexes

```bash
# CREATE INDEX idx_users_email ON users (email);
# CREATE UNIQUE INDEX idx_users_email_unique ON users (email);
# CREATE INDEX idx_users_name_lower ON users (LOWER(name));   # expression index
# CREATE INDEX idx_orders_date ON orders (created_at DESC);
# CREATE INDEX idx_users_data_gin ON users USING GIN (data);  # for JSONB
# CREATE INDEX CONCURRENTLY idx_big_table ON big_table (col); # non-blocking
# DROP INDEX idx_users_email;
```

## Views

```bash
# CREATE VIEW active_users AS
#     SELECT id, name, email FROM users WHERE active = true;
# CREATE MATERIALIZED VIEW monthly_stats AS
#     SELECT date_trunc('month', created_at) AS month, COUNT(*)
#     FROM orders GROUP BY 1;
# REFRESH MATERIALIZED VIEW CONCURRENTLY monthly_stats;
```

## Users & Roles

```bash
# CREATE ROLE app_user LOGIN PASSWORD 'securepass';
# ALTER ROLE app_user SET search_path TO myapp, public;
# CREATE ROLE readonly;
# GRANT CONNECT ON DATABASE mydb TO readonly;
# GRANT USAGE ON SCHEMA public TO readonly;
# GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly;
# ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO readonly;
# GRANT readonly TO app_user;
# DROP ROLE readonly;
```

## Backup & Restore

### pg_dump

```bash
pg_dump -U postgres mydb > backup.sql
pg_dump -U postgres -Fc mydb > backup.dump         # custom format (compressed)
pg_dump -U postgres -t users mydb > users.sql       # single table
pg_dumpall -U postgres > all_databases.sql          # all databases + roles
```

### pg_restore

```bash
pg_restore -U postgres -d mydb backup.dump
pg_restore -U postgres -d mydb -t users backup.dump  # single table
pg_restore -U postgres --clean --if-exists -d mydb backup.dump
```

## Query Analysis

### EXPLAIN

```bash
# EXPLAIN SELECT * FROM users WHERE email = 'alice@example.com';
# EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'alice@example.com';
# EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) SELECT ...;
```

### Useful diagnostic queries

```bash
# SELECT pg_size_pretty(pg_database_size('mydb'));
# SELECT relname, n_tup_ins, n_tup_upd, n_tup_del
#     FROM pg_stat_user_tables ORDER BY n_tup_ins DESC;
# SELECT * FROM pg_stat_activity WHERE state = 'active';
# SELECT pg_cancel_backend(pid);       # cancel a query
# SELECT pg_terminate_backend(pid);    # kill a connection
```

## JSONB

```bash
# SELECT data->>'name' FROM users;                    # text value
# SELECT data->'address'->>'city' FROM users;          # nested text
# SELECT * FROM users WHERE data @> '{"role":"admin"}'; # containment
# SELECT * FROM users WHERE data ? 'email';             # key exists
# UPDATE users SET data = data || '{"verified":true}' WHERE id = 1;
# UPDATE users SET data = data - 'temp_key';            # remove key
# SELECT * FROM users WHERE data->>'age' IS NOT NULL;
```

## Common Functions

```bash
# NOW(), CURRENT_TIMESTAMP, CURRENT_DATE
# date_trunc('month', created_at)
# EXTRACT(YEAR FROM created_at)
# AGE(NOW(), created_at)
# COALESCE(name, 'Unknown')
# NULLIF(value, 0)
# string_agg(name, ', ')
# array_agg(id)
# generate_series(1, 100)
# gen_random_uuid()
```

## Tips

- Use `TIMESTAMPTZ` (not `TIMESTAMP`) for all time columns to avoid timezone bugs.
- `CREATE INDEX CONCURRENTLY` avoids locking the table during index creation.
- `EXPLAIN ANALYZE` actually runs the query. Do not use it on destructive statements without a transaction.
- `ON CONFLICT ... DO UPDATE` (upsert) is atomic and avoids race conditions.
- Use `pg_dump -Fc` (custom format) for backups. It compresses and allows selective restore.
- `\x auto` in psql switches to expanded display only when rows are wide.
- `VACUUM ANALYZE` updates statistics and reclaims space. Autovacuum handles most cases.
- `pg_stat_activity` is your first stop for debugging slow queries and connection issues.
- Use `GIN` indexes for JSONB containment queries (`@>`) and array membership.
