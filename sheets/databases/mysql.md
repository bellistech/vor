# MySQL (Relational Database)

Widely-used open-source relational database with InnoDB storage engine and replication support.

## Client Connection

### Connect

```bash
mysql -u root -p
mysql -u admin -p -h db.example.com -P 3306 mydb
mysql -u admin -p mydb < script.sql
mysql -u admin -p -e "SHOW DATABASES;"
```

### Useful client flags

```bash
mysql -u root -p --auto-rehash       # enable tab completion
mysql -u root -p -t                  # table format output
mysql -u root -p -N -B -e "SELECT 1" # no headers, tab-separated (scripting)
```

## DDL (CREATE / ALTER / DROP)

### Databases

```bash
# CREATE DATABASE mydb CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
# DROP DATABASE IF EXISTS mydb;
# USE mydb;
# SHOW DATABASES;
```

### Tables

```bash
# CREATE TABLE users (
#     id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
#     email VARCHAR(255) NOT NULL UNIQUE,
#     name VARCHAR(100) NOT NULL,
#     active TINYINT(1) DEFAULT 1,
#     created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
#     updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
#     INDEX idx_name (name)
# ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

# ALTER TABLE users ADD COLUMN phone VARCHAR(20) AFTER email;
# ALTER TABLE users MODIFY COLUMN name VARCHAR(200) NOT NULL;
# ALTER TABLE users DROP COLUMN phone;
# ALTER TABLE users RENAME TO customers;

# DROP TABLE IF EXISTS users;
# SHOW TABLES;
# DESCRIBE users;
# SHOW CREATE TABLE users;
```

## DML (CRUD)

### INSERT

```bash
# INSERT INTO users (email, name) VALUES ('alice@example.com', 'Alice');
# INSERT INTO users (email, name) VALUES
#     ('bob@example.com', 'Bob'),
#     ('carol@example.com', 'Carol');
# INSERT INTO users (email, name) VALUES ('alice@example.com', 'Alice')
#     ON DUPLICATE KEY UPDATE name = VALUES(name);
# REPLACE INTO users (id, email, name) VALUES (1, 'alice@example.com', 'Alice');
```

### SELECT

```bash
# SELECT * FROM users WHERE active = 1 ORDER BY created_at DESC LIMIT 10;
# SELECT name, COUNT(*) AS total FROM orders GROUP BY name HAVING total > 5;
# SELECT u.name, o.amount FROM users u
#     INNER JOIN orders o ON u.id = o.user_id
#     WHERE o.created_at > DATE_SUB(NOW(), INTERVAL 30 DAY);
# SELECT * FROM users LIMIT 10 OFFSET 20;
```

### UPDATE

```bash
# UPDATE users SET name = 'Alice Smith' WHERE email = 'alice@example.com';
# UPDATE users SET active = 0 WHERE last_login < DATE_SUB(NOW(), INTERVAL 1 YEAR);
```

### DELETE

```bash
# DELETE FROM sessions WHERE expires_at < NOW();
# TRUNCATE TABLE logs;   # fast, resets AUTO_INCREMENT
```

## Indexes

```bash
# CREATE INDEX idx_users_email ON users (email);
# CREATE UNIQUE INDEX idx_email ON users (email);
# CREATE INDEX idx_name_email ON users (name, email);     # composite
# CREATE FULLTEXT INDEX idx_ft_bio ON users (bio);        # full-text
# ALTER TABLE users ADD INDEX idx_name (name);
# DROP INDEX idx_users_email ON users;
# SHOW INDEX FROM users;
```

## Views

```bash
# CREATE VIEW active_users AS
#     SELECT id, name, email FROM users WHERE active = 1;
# DROP VIEW active_users;
# SHOW CREATE VIEW active_users;
```

## Users & Grants

```bash
# CREATE USER 'app'@'%' IDENTIFIED BY 'securepass';
# CREATE USER 'readonly'@'10.0.0.%' IDENTIFIED BY 'readpass';
# GRANT ALL PRIVILEGES ON mydb.* TO 'app'@'%';
# GRANT SELECT ON mydb.* TO 'readonly'@'10.0.0.%';
# REVOKE DELETE ON mydb.* FROM 'app'@'%';
# SHOW GRANTS FOR 'app'@'%';
# DROP USER 'app'@'%';
# FLUSH PRIVILEGES;
```

## Backup & Restore

### mysqldump

```bash
mysqldump -u root -p mydb > backup.sql
mysqldump -u root -p mydb users orders > tables.sql     # specific tables
mysqldump -u root -p --all-databases > all.sql
mysqldump -u root -p --single-transaction mydb > backup.sql  # consistent InnoDB
mysqldump -u root -p --routines --triggers mydb > full.sql
```

### Restore

```bash
mysql -u root -p mydb < backup.sql
```

### Binary log point-in-time recovery

```bash
mysqlbinlog --start-datetime="2025-01-15 10:00:00" binlog.000042 | mysql -u root -p
```

## Engine & Character Set

### Check engine

```bash
# SHOW TABLE STATUS WHERE Name = 'users';
# ALTER TABLE users ENGINE=InnoDB;
```

### Character set and collation

```bash
# SHOW VARIABLES LIKE 'character_set%';
# ALTER DATABASE mydb CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
# ALTER TABLE users CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

## Common Functions

```bash
# NOW(), CURDATE(), CURTIME()
# DATE_FORMAT(created_at, '%Y-%m-%d')
# DATE_ADD(NOW(), INTERVAL 7 DAY)
# DATE_SUB(NOW(), INTERVAL 1 HOUR)
# DATEDIFF(end_date, start_date)
# CONCAT(first_name, ' ', last_name)
# SUBSTRING(name, 1, 5)
# IFNULL(phone, 'N/A')
# COALESCE(a, b, c)
# GROUP_CONCAT(name SEPARATOR ', ')
# JSON_EXTRACT(data, '$.key')
# UUID()
```

## Diagnostics

```bash
# SHOW PROCESSLIST;
# KILL <process_id>;
# SHOW ENGINE INNODB STATUS\G
# EXPLAIN SELECT * FROM users WHERE email = 'alice@example.com';
# SHOW VARIABLES LIKE 'max_connections';
# SHOW STATUS LIKE 'Threads_connected';
# SELECT @@version;
```

## Tips

- Always use `utf8mb4` instead of `utf8` in MySQL. MySQL's `utf8` is only 3 bytes and cannot store emoji.
- `--single-transaction` in mysqldump gives a consistent backup for InnoDB without locking tables.
- `EXPLAIN` output: watch for `ALL` in type column (full table scan) and missing `Using index`.
- `ON DUPLICATE KEY UPDATE` is MySQL's upsert. Use it instead of check-then-insert patterns.
- InnoDB is the default and correct engine for almost all tables. MyISAM lacks transactions and crash safety.
- `TRUNCATE` is faster than `DELETE FROM table` for removing all rows, but cannot be rolled back.
- Set `innodb_buffer_pool_size` to 70-80% of available RAM on a dedicated database server.
- `SHOW CREATE TABLE` gives the exact DDL including indexes and constraints, more useful than `DESCRIBE`.
