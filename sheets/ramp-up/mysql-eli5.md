# MySQL — ELI5

> MySQL is a giant filing cabinet at the office, organized into drawers and folders, with a clerk at the front who finds files for everyone who asks, writes down every change in a journal, and keeps a backup clerk in another room with the exact same cabinet.

## Prerequisites

It helps if you already know what a database is — go read **ramp-up/postgres-eli5** first if "database" is a fuzzy word for you. The Postgres sheet explains what tables, rows, and columns are using ELI5 pictures, and that picture transfers almost perfectly to MySQL. Once you've read that, come back here.

If you don't want to read the Postgres sheet first, that's fine too — this sheet defines every term as it appears, just at a slightly faster pace. The very first **What Even Is MySQL** section will get you up to speed.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

If you see a `mysql>` at the start of a line, that means "type the rest of this line into the MySQL command line." You do not type the `mysql>`. The MySQL command line is what you get when you log in to the database — we'll show you how to do that in a few minutes.

If a word feels weird, scroll down to the **Vocabulary** table near the bottom. Every weird word in this sheet has a one-line plain-English definition there.

## What Even Is MySQL

### The filing cabinet picture

Picture a giant office. In the middle of the office there is a filing cabinet. The filing cabinet has lots of **drawers**. Each drawer has lots of **folders**. Each folder has lots of **sheets of paper** with strict rules about what can go on each sheet.

A drawer might be labeled "CUSTOMERS." Inside that drawer there is a folder called "people" with a sheet of paper for every customer. Each sheet has the same boxes printed on it: a name box, an email box, a birthday box, a number box. If you tried to write a phone number in the birthday box, the clerk at the front desk would yell at you and refuse to file the paper, because birthdays go in the birthday box.

Now imagine lots of people walking into the office at the same time wanting things. One person wants to look up a customer. Another person wants to add a new customer. Another wants to delete an old one. Another wants to make a list of all customers born in May.

If everybody just walked into the cabinet and grabbed papers themselves, it would be chaos. Papers would get lost. Two people would grab the same paper and tear it in half fighting over it. Some folders would empty out, others would never be used.

So the office has a **clerk**. The clerk stands between you and the cabinet. You walk up to the clerk and say, "Please give me the paper for customer #42." The clerk goes back to the cabinet, opens drawer CUSTOMERS, opens folder people, finds paper #42, copies it onto a sheet for you, and hands you the sheet. You never touch the cabinet.

The clerk also writes down every change in a big book called the **journal**. Every time somebody adds a paper, the clerk writes "added paper #43 to folder people in drawer CUSTOMERS at 3:42 pm." Every time somebody changes a paper, the clerk writes that down too. The journal is a complete record of every single thing that ever happened to the cabinet.

In another room there is a **second cabinet** that is supposed to look exactly like the first one. There is also a second clerk standing in front of it. The second clerk's only job is to read the first clerk's journal, page by page, and copy every change into the second cabinet. That way, if the first cabinet catches fire, the second cabinet still has everything.

That is MySQL.

- The filing cabinet is the **database server**.
- The drawers are **databases** (sometimes called **schemas** in other systems).
- The folders are **tables**.
- The sheets of paper are **rows**.
- The boxes on each sheet are **columns**.
- The strict rules about what goes in each box are **data types**.
- The clerk at the front is the **server process** (`mysqld`).
- The people walking up to the clerk are **clients**.
- The journal is the **binary log** (binlog).
- The second clerk in another room with a copy of the cabinet is a **replica**.
- The original cabinet is the **source** (older docs say "master").

### What does "MySQL" mean

The name "MySQL" is a mash-up. The "My" is the name of one of the founder's daughters (her name is My — pronounced like "me" with a softer sound). The "SQL" is short for **Structured Query Language**, which is the special language you use to ask the clerk for things. SQL is its own little language with words like `SELECT`, `INSERT`, `UPDATE`, `DELETE`, and so on. You'll see lots of it in this sheet.

So "MySQL" is "My's SQL clerk." That's it. Just a name.

### Why people use MySQL

MySQL became famous because it was free, fast, and easy to set up. In the early 2000s, every popular website you can think of (Facebook, YouTube, Wikipedia, Twitter when it started, WordPress) was running on MySQL or something derived from it. It still powers a huge fraction of the internet today. If you've ever used a site with logins and posts and comments, there's a decent chance MySQL or one of its forks is the cabinet keeping it all straight.

### How big can the cabinet get

Pretty big. A single MySQL server can hold tens of millions to billions of rows in a single table. There are public examples of single MySQL servers handling **hundreds of terabytes** of data. The clerks in MySQL are very fast — they can answer thousands of questions a second on a normal laptop, and tens of thousands a second on a real server.

When one cabinet isn't enough, you set up many cabinets and split the data between them. That is called **sharding**, but we won't worry about that yet.

## MySQL vs MariaDB vs Percona Server

### Three cabinets that look almost the same

Once upon a time there was just MySQL. It was made by a small Swedish company called MySQL AB. In 2008, Sun Microsystems bought MySQL AB. In 2010, Oracle bought Sun. So now Oracle owns MySQL.

A bunch of the original developers, including the founder Monty Widenius, were nervous about Oracle owning MySQL. So they made a fork. A **fork** is when you take all the code, copy it, and start a new project with the copy. They called the fork **MariaDB**, which is named after another of Monty's daughters (Maria — see a pattern?).

A different company called Percona made a fork too, called **Percona Server**. Percona Server is much closer to MySQL — it tries to be a drop-in replacement that's just a bit faster and more transparent about what's going on inside.

So today there are three filing cabinets that look almost the same on the outside but have slightly different clerks inside:

| Name | Maker | Vibe |
|---|---|---|
| MySQL | Oracle | The original. Most popular. Closed-source enterprise version on top of free community. |
| MariaDB | The MariaDB Foundation | Fork. Diverging more every year. Has unique features like the Aria storage engine. |
| Percona Server | Percona | Closer to MySQL. Adds visibility and tooling. Used heavily in industry. |

### Which clerk talks the same language

All three speak SQL. Most simple commands work the same on all three. Where they differ:

- Replication: MySQL has Group Replication, MariaDB has Galera (built in), Percona Server bundles Percona XtraDB Cluster which is also Galera-based.
- Storage engines: MySQL ships InnoDB. MariaDB ships InnoDB plus its own Aria engine plus more. Percona Server ships InnoDB plus optionally MyRocks (RocksDB).
- Default password plugin: MySQL 8.0 uses caching_sha2_password. MariaDB still uses mysql_native_password.
- JSON: MySQL has a real JSON type. MariaDB stores JSON as LONGTEXT with helper functions.
- Window functions, CTEs: All three have them now.
- System tables: Slightly different layouts.

In this sheet, when we say "MySQL" we mean Oracle's MySQL 8.0. We'll point out where MariaDB or Percona behave differently.

### Why does this matter for me right now

Probably not at all. If you're reading an ELI5 sheet, you're going to install whatever is easiest, and that's fine. But six months from now when you're searching for help online and somebody says "use this command," remember to check whether they meant MySQL, MariaDB, or Percona, because the command might not exist on your version.

## The Storage Engines

### What is a storage engine

The clerk at the desk doesn't actually walk into the cabinet to grab papers. The clerk hands the request to a **back-room clerk** whose job is to physically open drawers and pull paper. There are different back-room clerks with different personalities. Some are careful and write everything down twice. Some are fast but forget things if the building loses power. Some only let you read papers, not write new ones.

The back-room clerk is called the **storage engine**. MySQL is unusual because it lets you pick a different storage engine for each table. The same database can have one careful table and one fast-but-forgetful table.

### The lineup

Here are the storage engines you might bump into:

```
+---------------+-----------------------------------------------------------+
| Engine        | What it does                                              |
+---------------+-----------------------------------------------------------+
| InnoDB        | The default since MySQL 5.5. Careful. Transactions, row-  |
|               | level locking, foreign keys, crash recovery.              |
| MyISAM        | The old default before 5.5. Fast for reads. No trans-     |
|               | actions. Table-level locking. Crash-prone. Legacy.        |
| MEMORY (HEAP) | Lives only in RAM. Vanishes on restart. Great for temp.   |
| ARCHIVE       | Compressed. Append-only. No updates. Used for old logs.   |
| CSV           | Stores rows as a literal CSV file on disk. No indexes.    |
| BLACKHOLE     | Accepts data and throws it away. Like /dev/null. Used     |
|               | as a relay for replication.                               |
| FEDERATED     | Forwards queries to a remote MySQL server. Like a tunnel. |
| NDB           | The clustered engine for MySQL Cluster. Distributed.      |
| MyRocks       | RocksDB-backed. Used by Percona Server and MariaDB. LSM   |
|               | tree, great compression, write-heavy workloads.           |
+---------------+-----------------------------------------------------------+
```

### Why InnoDB won

Before MySQL 5.5 (released in 2010), the default was MyISAM. MyISAM was fast for reads, but it had two huge problems:

1. **No transactions.** If you tried to move money from one bank account to another, and the power went out halfway through, MyISAM would happily leave the money in mid-air, gone from one account but never arriving in the other. Disaster.
2. **Table-level locking.** When somebody wrote to a table, the entire table was locked for everyone else. Even readers. Imagine the clerk telling everybody waiting in line, "sorry, the customer drawer is closed, somebody is filing a paper." For a busy website with many writers, this was a nightmare.

InnoDB fixed both problems:

1. InnoDB has **transactions**: a way to group several changes together so they all happen or none of them do. (More on this in **Transactions and Isolation**.)
2. InnoDB does **row-level locking**: only the specific paper being changed is locked, not the whole folder. Hundreds of people can write to different rows at the same time.

InnoDB also added **foreign keys** (rules saying "this paper must point at a real paper in another folder, you can't just make up a fake reference"), **crash recovery** (the clerk can rebuild the cabinet from the journal if the building catches fire), and a smart cache called the **buffer pool**.

The trade-off is that InnoDB is a bit slower for some pure-read tasks. For 99% of workloads in 2026, you want InnoDB. Unless you're maintaining a 20-year-old website, you should never see MyISAM in real life.

### When you'd use the others

- **MEMORY**: temp tables for fast lookups. Goes away on restart, so don't store anything important.
- **ARCHIVE**: write-once data like audit logs or telemetry where you'll rarely read individual rows.
- **BLACKHOLE**: replication trick — a server with BLACKHOLE tables accepts data, writes the binlog, and forwards it to replicas without keeping anything itself. A relay.
- **FEDERATED**: legacy way of querying a remote MySQL. Mostly replaced by application-level joins or CONNECT engine in MariaDB.
- **NDB**: shared-nothing distributed MySQL Cluster. Niche, telco-grade.

For everything else: **InnoDB**.

## A Hello-World CREATE / INSERT / SELECT

Time to actually use the thing. We'll assume you have MySQL installed somewhere. If not, on macOS:

```bash
$ brew install mysql
$ brew services start mysql
$ mysql -u root
```

On Debian/Ubuntu:

```bash
$ sudo apt install mysql-server
$ sudo mysql
```

Once you're in, you'll see this prompt:

```
mysql>
```

Now type these in one at a time:

```sql
mysql> CREATE DATABASE hello;
Query OK, 1 row affected (0.01 sec)

mysql> USE hello;
Database changed

mysql> CREATE TABLE pets (
    ->   id   INT AUTO_INCREMENT PRIMARY KEY,
    ->   name VARCHAR(50) NOT NULL,
    ->   kind VARCHAR(20),
    ->   age  INT
    -> );
Query OK, 0 rows affected (0.03 sec)

mysql> INSERT INTO pets (name, kind, age) VALUES ('Rex', 'dog', 4);
Query OK, 1 row affected (0.01 sec)

mysql> INSERT INTO pets (name, kind, age) VALUES ('Whiskers', 'cat', 7);
Query OK, 1 row affected (0.00 sec)

mysql> SELECT * FROM pets;
+----+----------+------+------+
| id | name     | kind | age  |
+----+----------+------+------+
|  1 | Rex      | dog  |    4 |
|  2 | Whiskers | cat  |    7 |
+----+----------+------+------+
2 rows in set (0.00 sec)
```

What just happened, line by line:

1. `CREATE DATABASE hello` made a new drawer called "hello."
2. `USE hello` told the clerk "from now on, all my requests are about the hello drawer, until I say otherwise."
3. `CREATE TABLE pets (...)` made a new folder called "pets" inside the hello drawer. The folder has 4 boxes printed on every paper: id, name, kind, age. The id box auto-fills with the next number. The name box must always have something in it (NOT NULL). The id box is the **primary key** — the unique label for each paper.
4. The two `INSERT` lines added two papers to the folder.
5. `SELECT * FROM pets` told the clerk "show me every paper in the pets folder, every box." `*` means "all columns."

Now you have a working filing cabinet with two pets in it. Welcome to MySQL.

### Adding, changing, deleting

```sql
mysql> INSERT INTO pets (name, kind, age) VALUES ('Goldie', 'fish', 1);
Query OK, 1 row affected (0.00 sec)

mysql> UPDATE pets SET age = 5 WHERE name = 'Rex';
Query OK, 1 row affected (0.00 sec)
Rows matched: 1  Changed: 1  Warnings: 0

mysql> DELETE FROM pets WHERE kind = 'fish';
Query OK, 1 row affected (0.00 sec)

mysql> SELECT * FROM pets;
+----+----------+------+------+
| id | name     | kind | age  |
+----+----------+------+------+
|  1 | Rex      | dog  |    5 |
|  2 | Whiskers | cat  |    7 |
+----+----------+------+------+
2 rows in set (0.00 sec)
```

Notice that `DELETE FROM pets;` (with no `WHERE`) would delete every row. **Always think before typing DELETE without a WHERE clause.** That's the kind of mistake that gets people fired.

## Data Types

Every box on every paper has a strict rule about what can go in it. The rule is the **data type**. MySQL has a lot of types because it has to handle numbers, text, dates, blobs of binary data, JSON, geographic shapes, and more.

### The number types

```
+------------+-------------+-------------------------------------------+
| Type       | Bytes       | Range (signed)                            |
+------------+-------------+-------------------------------------------+
| TINYINT    | 1           | -128 to 127                               |
| SMALLINT   | 2           | -32,768 to 32,767                         |
| MEDIUMINT  | 3           | -8,388,608 to 8,388,607                   |
| INT        | 4           | -2.1 billion to 2.1 billion               |
| BIGINT     | 8           | -9.2 quintillion to 9.2 quintillion       |
| FLOAT      | 4           | ~7 decimal digits of precision            |
| DOUBLE     | 8           | ~15 decimal digits of precision           |
| DECIMAL(M,D)| varies     | Exact decimal, M total digits, D after .  |
+------------+-------------+-------------------------------------------+
```

Use INT for normal counts. Use BIGINT for very large numbers (like Twitter status IDs). Use DECIMAL for **money** — never use FLOAT or DOUBLE for currency, because floats can't exactly represent 0.10 + 0.20.

There's also `UNSIGNED` for non-negative versions: `INT UNSIGNED` runs 0 to 4.2 billion.

Note: in MySQL 8.0, the **display width** in `INT(11)` is just decoration — it doesn't change the storage. People used to think `INT(11)` meant a different size; it doesn't. `INT` and `INT(11)` are identical.

### The text types

```
+------------+----------------------------------------------------------+
| Type       | What                                                     |
+------------+----------------------------------------------------------+
| CHAR(N)    | Fixed length. Always N chars. Padded with spaces.        |
| VARCHAR(N) | Variable length. Up to N chars. Length-prefixed.         |
| TINYTEXT   | Up to 255 bytes.                                         |
| TEXT       | Up to 65,535 bytes (~64 KB).                             |
| MEDIUMTEXT | Up to 16 MB.                                             |
| LONGTEXT   | Up to 4 GB.                                              |
+------------+----------------------------------------------------------+
```

Use `VARCHAR` for most strings. Use `CHAR` only for fixed-width fields like 2-letter country codes. Use `TEXT` for blog posts. Use `LONGTEXT` for whole books.

The `CHAR` vs `VARCHAR` thing trips everyone up. `CHAR(10)` always uses 10 characters even if you only put 3 in there — the rest is padded with spaces. `VARCHAR(10)` uses just enough room for 3 plus a small length header. For almost everything, `VARCHAR` is what you want.

### The blob types

```
+--------------+--------------------------------------------------------+
| Type         | What                                                   |
+--------------+--------------------------------------------------------+
| TINYBLOB     | Up to 255 bytes.                                       |
| BLOB         | Up to 65 KB.                                           |
| MEDIUMBLOB   | Up to 16 MB.                                           |
| LONGBLOB     | Up to 4 GB.                                            |
+--------------+--------------------------------------------------------+
```

BLOB is for raw binary like images or files. It's like TEXT but doesn't try to interpret the bytes as text.

**Note:** Don't store huge files in BLOB columns if you can help it. Store them on disk or in object storage and put a path/URL in the database.

### The date and time types

```
+-----------+-----------------------------------------------------------+
| Type      | What                                                      |
+-----------+-----------------------------------------------------------+
| DATE      | YYYY-MM-DD. 1000-01-01 to 9999-12-31. 3 bytes.            |
| TIME      | HH:MM:SS. -838:59:59 to 838:59:59. 3 bytes.               |
| DATETIME  | YYYY-MM-DD HH:MM:SS. 1000-01-01 to 9999-12-31. 5-8 bytes. |
| TIMESTAMP | Same look, but stores UTC seconds since 1970. 4 bytes.    |
| YEAR      | YYYY. 1901 to 2155. 1 byte.                               |
+-----------+-----------------------------------------------------------+
```

The big confusion: **DATETIME vs TIMESTAMP.** They look identical when you `SELECT` them. The differences:

- TIMESTAMP is stored as seconds since 1970-01-01 UTC. It auto-converts to your session's time zone when you read it.
- DATETIME is stored literally — no time zone conversion.
- TIMESTAMP only goes up to 2038-01-19 03:14:07 UTC. After that, it overflows. (The "Year 2038 problem" in old systems.)
- DATETIME goes up to year 9999.

For new code, prefer `DATETIME` unless you specifically want time-zone-aware storage. If you do want time-zone-aware storage, set `time_zone = '+00:00'` on the server and store everything in UTC.

### JSON

Since MySQL 5.7 (2015), there's a real `JSON` type:

```sql
mysql> CREATE TABLE doc (id INT PRIMARY KEY, body JSON);
mysql> INSERT INTO doc VALUES (1, '{"name":"Rex","tags":["dog","good boy"]}');
mysql> SELECT body->'$.name' FROM doc WHERE id = 1;
+----------------+
| body->'$.name' |
+----------------+
| "Rex"          |
+----------------+
```

The `->` operator extracts a path. `->>` does the same but unquotes strings. There's `JSON_EXTRACT`, `JSON_TABLE` (since 8.0) for turning JSON arrays into rows, and a big pile of helper functions.

Note: in MariaDB, `JSON` is just an alias for `LONGTEXT` with a CHECK constraint — it doesn't get the binary-encoded fast access that MySQL has.

### GEOMETRY

Spatial types: `POINT`, `LINESTRING`, `POLYGON`, `MULTIPOINT`, etc. Used for maps. You'd index them with a `SPATIAL INDEX` (R-tree). Niche unless you're building map software.

### ENUM and SET

```sql
mysql> CREATE TABLE shirt (
    ->   id   INT PRIMARY KEY,
    ->   size ENUM('S','M','L','XL'),
    ->   tags SET('cotton','wool','silk')
    -> );
```

`ENUM` lets you pick exactly one of a fixed list. `SET` lets you pick zero or more. They're stored as small integers internally so they're space-efficient.

People love and hate ENUM. Love because it documents the valid values right in the schema. Hate because adding a new value requires `ALTER TABLE`, which can be slow on big tables.

### NOT NULL, DEFAULT, AUTO_INCREMENT

These are not types but **column attributes**:

```sql
CREATE TABLE accounts (
  id     INT AUTO_INCREMENT PRIMARY KEY,
  email  VARCHAR(255) NOT NULL,
  name   VARCHAR(100) NOT NULL DEFAULT 'Anonymous',
  joined DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

- `NOT NULL`: this box must always have something in it.
- `DEFAULT`: if you don't say what to put in this box, fill in this value automatically.
- `AUTO_INCREMENT`: every new paper gets the next integer. Only one column per table can be `AUTO_INCREMENT`, and it has to be indexed (usually it's the primary key).

**Gotcha:** `NOT NULL DEFAULT '0'` on an INT column means "if you don't say anything, store 0." It does NOT mean "0 is now treated as null." Easy to confuse.

## Indexes

### What is an index

An index is the table-of-contents-with-page-numbers at the back of a textbook. Without an index, finding "Napoleon" in a 1000-page history book means flipping through every single page. With an index, you flip to the back, see "Napoleon — pages 412, 489, 720," and jump directly there.

Without an index on a column, MySQL has to read every row of the table to find matches. That's called a **full table scan**. With an index, MySQL can jump straight to the rows. The bigger the table, the bigger the difference — full scan of a billion-row table can take minutes; an index lookup takes a millisecond.

### The B+ tree

Every InnoDB index is a **B+ tree**. A B+ tree is a balanced tree where:

- Each node holds many keys (not just one or two), so the tree stays shallow.
- Only the **leaves** (the bottom layer) actually hold data; the upper layers are just signposts pointing to leaves.
- Leaves are linked together left-to-right, so range scans (`WHERE x BETWEEN 100 AND 200`) walk along the leaf chain.

```
                   [50 | 100]
                   /     |     \
              [10|30] [60|80] [120|150]
              /  |  \   ...      ...
           leaves with actual rows, linked sideways:
           [...] <-> [11,12,...] <-> [...] <-> [...]
```

A typical B+ tree with 3-4 levels can find a row in a billion-row table with just 3-4 disk reads. Beautiful.

### The clustered index

Here is the part that bites people. In InnoDB, the **table itself is the primary key index.** That is, the leaves of the primary key B+ tree literally are the rows. There's no separate "table file" with a separate "index pointing into it" — the index IS the table. This is called a **clustered index**.

Consequences:
- Rows are physically stored in primary-key order.
- Looking up a row by primary key is one tree walk and you're done — no second jump.
- If you don't define a primary key, InnoDB picks one for you (a hidden 6-byte rowid).
- The primary key should be small, because every secondary index stores a copy of it.

### Secondary indexes

A **secondary index** is any non-primary index. In InnoDB, the leaves of a secondary index don't contain the row data — they contain the **primary key value** of the matching row. So a lookup goes: walk the secondary index to find the PK, then walk the primary index to find the row.

```
SELECT * FROM users WHERE email = 'rex@dog.com';

  Secondary index on email:
    [..., 'rex@dog.com' -> PK 42, ...]
                                |
                                v
  Primary (clustered) index on id:
    [..., 42 -> {id:42, email:'rex@dog.com', name:'Rex', age:5}]
```

This is why a small primary key matters: every secondary index has a copy of it, multiplied by every row.

### Covering indexes

A **covering index** is when the secondary index has every column you need, so MySQL doesn't have to do the second lookup.

```sql
CREATE INDEX idx_email_name ON users (email, name);
SELECT name FROM users WHERE email = 'rex@dog.com';  -- covered, fast
SELECT name, age FROM users WHERE email = 'rex@dog.com'; -- not covered (needs age)
```

`EXPLAIN` shows `Using index` in the Extra column when an index covers a query. This is one of the fastest things MySQL can do.

### FULLTEXT

For text search:

```sql
CREATE FULLTEXT INDEX idx_body ON posts(body);
SELECT * FROM posts WHERE MATCH(body) AGAINST('cat dog');
```

FULLTEXT indexes words and allows ranked text search. Not as powerful as a real search engine like Elasticsearch but built right in.

### SPATIAL

For geometry types: `CREATE SPATIAL INDEX idx_geo ON places(loc);`. Uses an R-tree internally — a tree where each node bounds a rectangle of space.

### HASH (MEMORY only)

The MEMORY engine supports hash indexes — constant-time lookup but useless for range queries. Just trivia for non-MEMORY tables.

## Transactions and Isolation

### What is a transaction

A **transaction** is a group of changes that all happen together or none of them happen. The classic example is moving money:

```sql
START TRANSACTION;
UPDATE accounts SET balance = balance - 100 WHERE id = 1;  -- Alice
UPDATE accounts SET balance = balance + 100 WHERE id = 2;  -- Bob
COMMIT;
```

If the power fails between the two UPDATEs, InnoDB will undo the first one when it restarts. Either Alice loses $100 and Bob gains $100, or nothing happens. Never something in between.

If you decide you don't want the transaction to take effect, you can `ROLLBACK` instead of `COMMIT`:

```sql
START TRANSACTION;
UPDATE accounts SET balance = 0 WHERE id = 1;
SELECT balance FROM accounts WHERE id = 1;  -- shows 0
ROLLBACK;
SELECT balance FROM accounts WHERE id = 1;  -- shows the old value
```

### ACID

Transactions give you four guarantees, the famous ACID:

- **Atomicity**: all or nothing.
- **Consistency**: the rules (constraints, foreign keys) hold before and after.
- **Isolation**: concurrent transactions don't see each other's half-finished work.
- **Durability**: once you commit, it survives a power outage.

InnoDB gets you all four. MyISAM gets you essentially none.

### Isolation levels

Isolation is the tricky one. When two transactions run at the same time, how much can they see of each other? MySQL gives you four levels:

```
+------------------+--------+------+----------+--------------+
| Level            | Dirty  | Non- | Phantom  | Performance  |
|                  | reads  | rep. | reads    |              |
+------------------+--------+------+----------+--------------+
| READ UNCOMMITTED |  YES   | YES  |   YES    | Fastest      |
| READ COMMITTED   |   no   | YES  |   YES    | Fast         |
| REPEATABLE READ  |   no   |  no  |   no*    | InnoDB default|
| SERIALIZABLE     |   no   |  no  |   no     | Slowest      |
+------------------+--------+------+----------+--------------+
```

The asterisk on REPEATABLE READ phantoms: in InnoDB, REPEATABLE READ blocks phantoms via **next-key locking**, which is stronger than the SQL standard requires. So InnoDB's REPEATABLE READ is actually pretty close to SERIALIZABLE for most workloads.

What the words mean:
- **Dirty read**: you see another transaction's uncommitted changes.
- **Non-repeatable read**: you read a row, somebody else commits a change, you read it again and it's different.
- **Phantom read**: you run `SELECT * WHERE x > 5` once, somebody inserts a new matching row, you run it again and a new row appears.

To set:

```sql
SET TRANSACTION ISOLATION LEVEL READ COMMITTED;
START TRANSACTION;
-- ...
COMMIT;
```

Or globally: `SET GLOBAL TRANSACTION ISOLATION LEVEL READ COMMITTED;`.

### REPEATABLE READ vs Postgres

Heads up if you've used Postgres: Postgres's REPEATABLE READ is actually **snapshot isolation**, slightly different from MySQL's. MySQL's REPEATABLE READ uses MVCC with consistent reads but allows write skew anomalies in some cases. Postgres's SERIALIZABLE uses SSI (Serializable Snapshot Isolation). The names are the same, the guarantees differ a little.

For day to day, MySQL's REPEATABLE READ is fine. If you need ironclad guarantees, use SERIALIZABLE.

## Locks

### What's a lock

A **lock** is a "do not disturb" sign on a row, page, or table. Other transactions that want to touch the same thing have to wait.

InnoDB uses **row-level locks**. MyISAM uses **table-level locks**. The difference matters.

### Shared vs exclusive

- **Shared lock (S)**: "I'm reading this row, others can read but nobody can write."
- **Exclusive lock (X)**: "I'm changing this row, nobody can touch it."

```
+----+----+----+
|    | S  | X  |
+----+----+----+
| S  | OK | NO |
| X  | NO | NO |
+----+----+----+
```

`SELECT ... FOR SHARE` takes shared locks. `SELECT ... FOR UPDATE` takes exclusive locks. `INSERT/UPDATE/DELETE` take exclusive locks on the rows they touch.

### Intention locks

InnoDB also has table-level **intention locks** that say "somewhere in this table, somebody has a row lock." They're held alongside the row locks so quick conflict detection works. You usually don't think about them — they're internal.

- IS = Intention Shared
- IX = Intention Exclusive

### Gap and next-key locks

When InnoDB is at REPEATABLE READ, it locks not just the row but also the **gap** before it, to prevent phantoms. A **next-key lock** is a row lock plus the gap immediately before it.

Example: rows with id 10, 20, 30. If you `SELECT * FROM t WHERE id BETWEEN 11 AND 25 FOR UPDATE`, InnoDB locks the row at 20 plus the gaps (10,20) and (20,30), so nobody can `INSERT` an id of 15 or 22 while you're working.

This is what makes InnoDB's REPEATABLE READ phantom-free.

### Deadlocks

When two transactions wait for each other in a circle:

```
T1: locks row 1, wants row 2
T2: locks row 2, wants row 1
```

InnoDB detects this almost instantly and kills one of them with:

```
ERROR 1213 (40001): Deadlock found when trying to get lock; try restarting transaction
```

Your application should catch this and retry. It's normal to see occasional deadlocks under load.

### Lock wait timeout

If a transaction can't get a lock within `innodb_lock_wait_timeout` seconds (default 50), it gets:

```
ERROR 1205 (HY000): Lock wait timeout exceeded; try restarting transaction
```

Tune this in `my.cnf`:

```ini
[mysqld]
innodb_lock_wait_timeout = 10
```

For diagnostic info:

```sql
SHOW ENGINE INNODB STATUS\G
```

(The `\G` instead of `;` makes the output a vertical format. Easier to read for big reports.)

## Replication

### The two cabinets, in detail

Recall the two cabinets in two rooms. Every change to the first cabinet (the **source**, formerly **master**) is recorded in the journal (the **binary log** or **binlog**). The second cabinet (the **replica**, formerly **slave**) reads the journal and applies every change to itself.

```
   client writes
        |
        v
+--------------+
|   SOURCE     |        binlog flow
|   (master)   |  ----------->   +--------------+
|              |                 |   REPLICA    |
| writes data  |                 |   (slave)    |
| writes binlog|                 | reads binlog |
+--------------+                 | applies      |
                                 | writes its   |
                                 | own copy     |
                                 +--------------+
```

The terminology change: in MySQL 8.0.22+, the official names are **source** and **replica**. Older docs and many production systems still say **master** and **slave**. Don't be confused.

### Binlog formats

The journal can be written in three styles:

- **STATEMENT**: writes the SQL statement itself. "INSERT INTO t VALUES (1, 'a')." Compact but unreliable for non-deterministic statements (like NOW() or UUID() or LIMIT without ORDER BY).
- **ROW**: writes the actual row data that changed. "Row 1 in table t went from {a,1} to {b,2}." Reliable but bigger.
- **MIXED**: uses STATEMENT for safe statements, ROW for unsafe ones.

Default since MySQL 5.7: ROW. Recommended.

```sql
mysql> SHOW VARIABLES LIKE 'binlog_format';
+---------------+-------+
| Variable_name | Value |
+---------------+-------+
| binlog_format | ROW   |
+---------------+-------+
```

### Async vs semi-sync vs sync

- **Async**: source commits, then sends to replicas in the background. Fastest. Default. Risk: if source dies before sending, replica missed transactions.
- **Semi-sync**: source waits for at least one replica to **acknowledge receipt** (not apply) before reporting commit success. Better safety, slight latency cost.
- **Sync (synchronous)**: source waits for replicas to fully apply. Used by Group Replication and Galera Cluster. Highest safety, highest latency.

Set up semi-sync by loading plugins:

```sql
INSTALL PLUGIN rpl_semi_sync_source SONAME 'semisync_source.so';
INSTALL PLUGIN rpl_semi_sync_replica SONAME 'semisync_replica.so';
SET GLOBAL rpl_semi_sync_source_enabled = ON;
```

### GTID — global transaction identifiers

A **GTID** is a unique label for every transaction in the replication topology. It looks like:

```
3E11FA47-71CA-11E1-9E33-C80AA9429562:23
\__________ source UUID __________/  : tx number
```

GTIDs replace the older "binlog file + position" coordinate, which was fragile during failovers.

Enable:

```ini
[mysqld]
gtid_mode = ON
enforce_gtid_consistency = ON
log_bin = /var/log/mysql/binlog
server_id = 1
```

With GTIDs, switching a replica to a new source is one command:

```sql
CHANGE REPLICATION SOURCE TO
  SOURCE_HOST='new-source.example.com',
  SOURCE_USER='repl',
  SOURCE_PASSWORD='secret',
  SOURCE_AUTO_POSITION=1;
START REPLICA;
```

### Group Replication

In MySQL 8.0, you can set up a group of three or more nodes that all see writes (multi-primary) or one node sees writes and the others are read-only (single-primary). Group Replication uses Paxos under the hood for consensus on every write. Strong consistency, automatic failover, no data loss.

The MySQL InnoDB Cluster product is Group Replication + MySQL Router + MySQL Shell, all bundled.

### NDB Cluster

Different beast. **MySQL Cluster** uses the NDB engine — a fully distributed in-memory database. Rows are sharded across data nodes with synchronous replication between them. Used in telecom for things like phone subscriber databases. Not the same as Group Replication.

## Query Optimization

### EXPLAIN

The first tool to reach for when a query is slow:

```sql
mysql> EXPLAIN SELECT * FROM pets WHERE name = 'Rex';
+----+-------------+-------+------+---------------+------+---------+------+------+-------------+
| id | select_type | table | type | possible_keys | key  | key_len | ref  | rows | Extra       |
+----+-------------+-------+------+---------------+------+---------+------+------+-------------+
|  1 | SIMPLE      | pets  | ALL  | NULL          | NULL | NULL    | NULL |    2 | Using where |
+----+-------------+-------+------+---------------+------+---------+------+------+-------------+
```

Read it: type=ALL means full table scan. We have two rows so that's fine. With a million rows, ALL is bad.

After adding `CREATE INDEX idx_name ON pets(name);`:

```
| id | select_type | table | type | possible_keys | key      | key_len | ref   | rows | Extra |
| 1  | SIMPLE      | pets  | ref  | idx_name      | idx_name | 202     | const |    1 |       |
```

type=ref means index lookup. rows=1 means it expects to read 1 row. Much better.

The most important `type` values, best to worst:
- `system`, `const`: 1 row, primary or unique key.
- `eq_ref`: 1 row per join from the other table.
- `ref`: index lookup, multiple rows possible.
- `range`: index range scan.
- `index`: full index scan (better than table scan).
- `ALL`: full table scan. Bad for big tables.

### EXPLAIN FORMAT=JSON

More detail, harder to read by eye:

```sql
EXPLAIN FORMAT=JSON SELECT * FROM pets WHERE name = 'Rex';
```

Lots of cost estimates in there. Use it when EXPLAIN's traditional output isn't enough.

### EXPLAIN ANALYZE

Since 8.0.18, this **actually runs the query** and reports timing per node:

```sql
EXPLAIN ANALYZE SELECT * FROM pets WHERE name = 'Rex';
-> Index lookup on pets using idx_name (name='Rex')
   (cost=0.35 rows=1) (actual time=0.012..0.015 rows=1 loops=1)
```

The `actual time` and `actual rows` numbers are gold — they show what really happened, not just what the optimizer guessed.

### optimizer_trace

For deep diving:

```sql
SET optimizer_trace = "enabled=on";
SELECT * FROM pets WHERE name = 'Rex';
SELECT * FROM information_schema.optimizer_trace\G
```

Output is a big JSON blob showing every decision the optimizer made. Useful for "why did it pick this index" questions.

### Index hints

Sometimes the optimizer is wrong. You can force its hand:

- `FORCE INDEX (idx_name)`: prefer this index.
- `USE INDEX (idx_name)`: only consider this index (and full scan).
- `IGNORE INDEX (idx_name)`: don't use this index.
- `STRAIGHT_JOIN`: force the join order to match the SQL order.

Example:

```sql
SELECT * FROM pets FORCE INDEX (idx_name) WHERE name = 'Rex' AND age > 5;
```

Hints are a last resort. If you need them often, your stats are probably stale (`ANALYZE TABLE`) or your indexes are wrong.

## Common Index Pitfalls

### Leftmost-prefix rule

A composite index on `(a, b, c)` works for queries on:

- `WHERE a = ?`
- `WHERE a = ? AND b = ?`
- `WHERE a = ? AND b = ? AND c = ?`

It does NOT work for:

- `WHERE b = ?`
- `WHERE c = ?`
- `WHERE b = ? AND c = ?`

Because the index is sorted by a first, then b inside each a, then c inside each b. You can't binary-search by b without knowing a.

### Function on a column kills the index

```sql
SELECT * FROM users WHERE LOWER(email) = 'rex@dog.com';     -- no index used
SELECT * FROM users WHERE email = LOWER('Rex@Dog.Com');     -- index used
SELECT * FROM users WHERE YEAR(created_at) = 2026;           -- no index used
SELECT * FROM users WHERE created_at >= '2026-01-01'         -- index used
                       AND created_at <  '2027-01-01';
```

Wrap a column in a function and the index becomes useless. Move the function to the constant side, or use a generated column with an index.

In MySQL 8.0.13+, you can have **functional indexes**:

```sql
CREATE INDEX idx_email_lower ON users ((LOWER(email)));
```

Then `WHERE LOWER(email) = '...'` does use the index.

### OR conditions

```sql
SELECT * FROM users WHERE email = 'a' OR username = 'b';
```

If only `email` has an index, MySQL does a full scan. Either index both columns and hope for **index merge**, or rewrite as `UNION`:

```sql
SELECT * FROM users WHERE email = 'a'
UNION
SELECT * FROM users WHERE username = 'b';
```

### IS NULL vs = NULL

```sql
SELECT * FROM users WHERE email = NULL;     -- always returns 0 rows
SELECT * FROM users WHERE email IS NULL;    -- correct
```

`= NULL` is never true, never false — it's NULL, which is falsy in WHERE. So you get nothing back.

### Implicit type conversion

```sql
SELECT * FROM users WHERE phone = 5551234567;  -- phone is VARCHAR
```

If `phone` is VARCHAR, MySQL converts every row's phone to a number to compare. Index is dead. Always quote: `WHERE phone = '5551234567'`.

### Wildcard at the start of LIKE

```sql
SELECT * FROM users WHERE name LIKE '%son';   -- can't use index
SELECT * FROM users WHERE name LIKE 'John%';  -- can use index
```

A leading wildcard means the index can't help (the index is sorted left-to-right). Trailing wildcards are fine.

## Slow Query Log + pt-query-digest

### Turn on the slow log

```ini
[mysqld]
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 1
log_queries_not_using_indexes = 1
```

Now every query taking longer than 1 second gets logged.

```bash
$ sudo tail -f /var/log/mysql/slow.log
# Time: 2026-04-27T15:30:10.123456Z
# User@Host: app[app] @ localhost []
# Query_time: 1.234567  Lock_time: 0.000012 Rows_sent: 1000  Rows_examined: 1000000
SET timestamp=1745765410;
SELECT * FROM events WHERE created_at > '2026-01-01';
```

### Aggregate with pt-query-digest

The slow log is raw. Aggregate it with `pt-query-digest` from Percona Toolkit:

```bash
$ pt-query-digest /var/log/mysql/slow.log
# 89.6s user time, 110ms system time, 28.50M rss, 50.50M vsz
# Current date: Mon Apr 27 15:35:00 2026
# Hostname: web1.example.com
# Files: /var/log/mysql/slow.log
# Overall: 12.5k total, 25 unique, 14.3 QPS, 0.18x concurrency

# Profile
# Rank Query ID                     Response time   Calls   R/Call
# ==== ============================ =============== ======= =======
#    1 0xABC123...                  450.2s 25.3%    1234    0.365
#    2 0xDEF456...                  380.1s 21.4%    5678    0.067
#    3 0x789012...                  220.5s 12.4%    400     0.551
```

You get a ranked list of the queries doing the most damage. Fix #1 first.

## The Query Cache

For ten years MySQL had a **query cache** that remembered the result of recent SELECTs. If the same query came in and no underlying table had changed, it returned the cached answer.

In theory: brilliant. In practice: a disaster. The cache had a single global mutex. On busy servers with lots of writes, the cache was constantly being invalidated, and the contention on that mutex slowed everything down.

**MySQL removed the query cache entirely in 8.0.** Don't look for it. It's gone.

If you want a query cache, use ProxySQL or an application-level cache like Redis. Don't try to reproduce the old behavior in 8.0 — it's not possible and that's by design.

MariaDB still has the query cache (improved). Percona Server still has it. Most production deployments turn it off anyway.

## Backup

A database with no backups is a ticking time bomb. Here are your tools.

### mysqldump

The classic. Single-threaded, produces a SQL text file:

```bash
$ mysqldump --single-transaction --routines --triggers --events \
            --master-data=2 --databases hello > hello.sql
$ ls -lh hello.sql
-rw-r--r-- 1 me me 12K Apr 27 15:42 hello.sql
```

Important flags:
- `--single-transaction`: take a consistent InnoDB snapshot via `START TRANSACTION` (no locking).
- `--routines`: include stored procedures.
- `--triggers`: include triggers.
- `--events`: include scheduled events.
- `--master-data=2`: write the binlog coordinates as a comment, useful for setting up replicas.

Restore with:

```bash
$ mysql < hello.sql
```

mysqldump is fine for small DBs. It's slow for big ones.

### mysqlpump

Multi-threaded version of mysqldump:

```bash
$ mysqlpump --default-parallelism=4 --databases hello > hello.sql
```

Faster but **deprecated in 8.0**. Use `mysqlsh util.dumpInstance()` instead.

### mydumper

Third-party tool, very fast, multi-threaded:

```bash
$ mydumper -B hello -o /backup -t 8
```

Pairs with `myloader` to restore. Used at huge scale.

### Percona XtraBackup

Hot, physical backup. Copies the data files while the server runs:

```bash
$ xtrabackup --backup --target-dir=/backup --user=root --password=...
$ xtrabackup --prepare --target-dir=/backup
```

To restore: stop MySQL, copy /backup over the data directory, start MySQL.

XtraBackup is what you use for production-grade backups of multi-terabyte databases.

### MySQL Shell util.dumpInstance

The official 8.0 tool:

```bash
$ mysqlsh root@localhost -- util dump-instance /backup --threads=8 --compatibility=strip_definers
$ mysqlsh root@localhost -- util load-dump /backup --threads=8
```

Multi-threaded, supports cloud storage, has a load tool that adapts the dump to the target server's settings. Recommended for new setups.

### Binary log backup

Don't forget to back up your binlogs alongside your data. The binlogs let you do **point-in-time recovery** — replay the journal up to the second before the bad query hit:

```bash
$ mysqlbinlog --start-datetime='2026-04-27 14:00:00' \
              --stop-datetime='2026-04-27 14:59:59' \
              binlog.000123 | mysql
```

## Common Configuration

The MySQL config file is `my.cnf` (Linux) or `my.ini` (Windows). Located at `/etc/mysql/my.cnf`, `/etc/my.cnf`, or `~/.my.cnf`. The most important knobs:

```ini
[mysqld]

# Memory
innodb_buffer_pool_size = 8G       # ~70-80% of RAM on dedicated server
innodb_log_file_size = 1G          # bigger = better write throughput
innodb_log_files_in_group = 2

# Connections
max_connections = 500              # how many clients can be open at once
thread_cache_size = 16             # cache idle threads

# Durability
innodb_flush_log_at_trx_commit = 1 # 1=ACID, 2=lose 1s on crash, 0=lose more
sync_binlog = 1                    # flush binlog on commit

# Logging
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 1
log_bin = /var/log/mysql/binlog
server_id = 1

# Character set
character_set_server = utf8mb4
collation_server     = utf8mb4_0900_ai_ci

# Time zone
default_time_zone = '+00:00'

# SQL mode
sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION'
```

### innodb_buffer_pool_size

The buffer pool is InnoDB's cache of data and index pages in RAM. It is the most important setting in MySQL. On a dedicated database server, set it to roughly 70-80% of total RAM. On a shared box, less.

### max_connections

How many client connections can be open simultaneously. Default 151. Each connection uses a thread and some memory. Don't set it absurdly high — use a connection pooler (ProxySQL, pgbouncer-equivalent like ProxySQL).

### sql_mode

Controls how strict MySQL is about bad data. The default since 5.7 is reasonably strict. The big modes:

- `STRICT_TRANS_TABLES`: refuse to silently truncate values for transactional tables.
- `STRICT_ALL_TABLES`: same but for all tables.
- `NO_ZERO_DATE`: refuse `'0000-00-00'` as a date.
- `ONLY_FULL_GROUP_BY`: every column in `SELECT` must be in `GROUP BY` or aggregated.
- `PIPES_AS_CONCAT`: make `||` mean string concat (like Postgres) instead of OR.

### character_set_server / collation_server

The default character set should be **utf8mb4**, not "utf8." More on this in **Common Confusions**.

A good modern collation: `utf8mb4_0900_ai_ci` (accent-insensitive, case-insensitive, Unicode 9.0).

### SET PERSIST

In 8.0, you can change variables and have them survive restart without editing the config file:

```sql
SET PERSIST max_connections = 1000;
```

This writes to `mysqld-auto.cnf` in the data directory, which gets read at startup.

## Authentication

### Users live at user@host

A MySQL user is **two parts**: a name and a host. `'app'@'localhost'` is a different user from `'app'@'%'`. Connection from a host has to match a `user@host` entry.

```sql
CREATE USER 'app'@'10.0.0.%' IDENTIFIED BY 'secretpassword';
GRANT SELECT, INSERT, UPDATE, DELETE ON hello.* TO 'app'@'10.0.0.%';
SHOW GRANTS FOR 'app'@'10.0.0.%';
+----------------------------------------------------------------+
| Grants for app@10.0.0.%                                        |
+----------------------------------------------------------------+
| GRANT USAGE ON *.* TO `app`@`10.0.0.%`                         |
| GRANT SELECT, INSERT, UPDATE, DELETE ON `hello`.* TO `app`@... |
+----------------------------------------------------------------+
```

`%` is a wildcard for any host. `10.0.0.%` is any IP in the 10.0.0.0/24 range.

### caching_sha2_password

In 8.0, the default password authentication plugin is `caching_sha2_password`. This is more secure than the older `mysql_native_password` but not all client libraries support it. If you get a "plugin not loaded" error from an older client, you can fall back:

```sql
ALTER USER 'app'@'%' IDENTIFIED WITH mysql_native_password BY 'secret';
```

Or in the config:

```ini
[mysqld]
default_authentication_plugin = mysql_native_password
```

The MariaDB-flavored servers still use `mysql_native_password` by default.

### REVOKE and FLUSH PRIVILEGES

To remove a permission:

```sql
REVOKE INSERT ON hello.* FROM 'app'@'10.0.0.%';
```

If you edit the `mysql.user` table directly (which you shouldn't), run:

```sql
FLUSH PRIVILEGES;
```

For `GRANT`/`REVOKE`/`CREATE USER`, the change takes effect immediately — no flush needed.

### Roles (8.0+)

Since 8.0 you can group permissions into roles:

```sql
CREATE ROLE 'app_read', 'app_write';
GRANT SELECT ON hello.* TO 'app_read';
GRANT INSERT, UPDATE, DELETE ON hello.* TO 'app_write';
GRANT 'app_read', 'app_write' TO 'app'@'%';
```

Then the user can pick which roles are active per session:

```sql
SET ROLE 'app_read';
```

## Common Errors

These are the messages you'll see when things go wrong, in roughly the order of frequency.

### `ERROR 1045 (28000): Access denied for user 'X'@'host' (using password: YES)`

The clerk doesn't recognize you. Check:
- Username typo.
- Password typo.
- The user exists at this host (remember `user@host`).
- The user has at least USAGE permission on the database.

### `ERROR 1062 (23000): Duplicate entry 'X' for key 'PRIMARY'`

You tried to insert a row with a primary key value that already exists. Either you have a real duplicate (data error) or you're trying to insert without an `AUTO_INCREMENT`-generated id.

### `ERROR 1146 (42S02): Table 'X.Y' doesn't exist`

The table isn't there. Check your `USE` database, check spelling, check `SHOW TABLES;`. Note: MySQL is case-sensitive about table names on Linux but not on macOS/Windows. The variable `lower_case_table_names` controls this.

### `ERROR 1064 (42000): You have an error in your SQL syntax`

Typo in your SQL. The error message shows where MySQL gave up parsing. Common causes: missing comma, missing semicolon, reserved word used as a column name without backticks.

### `ERROR 1213 (40001): Deadlock found when trying to get lock; try restarting transaction`

Two transactions waited for each other. InnoDB killed one. Retry it. If it happens often, you have a hot row or your transactions are taking too long.

### `ERROR 1205 (HY000): Lock wait timeout exceeded; try restarting transaction`

Some other transaction held a lock for longer than `innodb_lock_wait_timeout` seconds. Find the culprit with `SHOW ENGINE INNODB STATUS\G` and look for "TRANSACTIONS" with a long "history list length."

### `ERROR 1093 (HY000): You can't specify target table 'X' for update in FROM clause`

You can't `UPDATE t SET ... WHERE id IN (SELECT ... FROM t WHERE ...)`. MySQL refuses. Workaround: wrap the subquery in another subquery: `WHERE id IN (SELECT id FROM (SELECT id FROM t WHERE ...) sub)`.

### `ERROR 2002 (HY000): Can't connect to local MySQL server through socket '/var/run/mysqld/mysqld.sock'`

The server is not running, or it is running but the socket file is somewhere else. Run `systemctl start mysql` or check `/etc/mysql/my.cnf` for `socket = ...`.

### `ERROR 2003 (HY000): Can't connect to MySQL server on 'X' (111)`

TCP connection refused. The server isn't listening on that host/port, or a firewall is blocking it, or `bind-address` in `my.cnf` is set to localhost only. Check `bind-address = 0.0.0.0` to listen on all interfaces.

### `ERROR 1366 (HY000): Incorrect string value: '\xF0\x9F\x98\x80' for column 'X'`

You tried to store a 4-byte UTF-8 character (like an emoji) in a 3-byte `utf8` column. **Use `utf8mb4`**. See **Common Confusions**.

### `ERROR 1452 (23000): Cannot add or update a child row: a foreign key constraint fails`

You inserted a row pointing at a parent row that doesn't exist (or you're trying to delete a parent row that still has children). Either insert the parent first, set up `ON DELETE CASCADE`, or check your data.

### `ERROR 1227 (42000): Access denied; you need (at least one of) the SUPER privilege(s) for this operation`

Some commands (like `SET GLOBAL`, `CHANGE REPLICATION SOURCE TO`, `CREATE USER`) need elevated privileges. In 8.0, `SUPER` was split into many fine-grained privileges (`SYSTEM_VARIABLES_ADMIN`, `REPLICATION_SLAVE_ADMIN`, etc.). Grant the specific one needed.

### `ERROR 1356 (HY000): View 'X' references invalid table(s)`

A view's underlying table got dropped or altered out from under it. Recreate the view with `CREATE OR REPLACE VIEW`.

## Hands-On

Time to try things. Connect to your local MySQL and walk through these. The output is approximately what you should see (your numbers will differ a little).

### Connect

```bash
$ mysql -u root -p
Enter password:
Welcome to the MySQL monitor.  Commands end with ; or \g.
Server version: 8.0.36 MySQL Community Server - GPL
mysql>
```

### Connect to a specific database

```bash
$ mysql -h db.example.com -P 3306 -u app -p hello
Enter password:
mysql>
```

### Check the version

```bash
$ mysql --version
mysql  Ver 8.0.36 for Linux on x86_64 (MySQL Community Server - GPL)
```

### See the server's defaults

```bash
$ mysqld --verbose --help | grep -A 1 "Default options"
Default options are read from the following files in the given order:
/etc/my.cnf /etc/mysql/my.cnf /usr/etc/my.cnf ~/.my.cnf
```

### List databases

```sql
mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
| hello              |
+--------------------+
5 rows in set (0.00 sec)
```

### Switch into one and list tables

```sql
mysql> USE hello;
Database changed
mysql> SHOW TABLES;
+-----------------+
| Tables_in_hello |
+-----------------+
| pets            |
+-----------------+
```

### Describe a table

```sql
mysql> DESCRIBE pets;
+-------+-------------+------+-----+---------+----------------+
| Field | Type        | Null | Key | Default | Extra          |
+-------+-------------+------+-----+---------+----------------+
| id    | int         | NO   | PRI | NULL    | auto_increment |
| name  | varchar(50) | NO   |     | NULL    |                |
| kind  | varchar(20) | YES  |     | NULL    |                |
| age   | int         | YES  |     | NULL    |                |
+-------+-------------+------+-----+---------+----------------+
```

### Show the CREATE TABLE statement

```sql
mysql> SHOW CREATE TABLE pets\G
*************************** 1. row ***************************
       Table: pets
Create Table: CREATE TABLE `pets` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL,
  `kind` varchar(20) DEFAULT NULL,
  `age` int DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
```

### Show variables

```sql
mysql> SHOW VARIABLES LIKE 'innodb%' \G
*************************** 1. row ***************************
Variable_name: innodb_adaptive_flushing
        Value: ON
*************************** 2. row ***************************
Variable_name: innodb_adaptive_flushing_lwm
        Value: 10
... (lots more) ...
```

### Show status counters

```sql
mysql> SHOW STATUS LIKE 'Innodb_rows%';
+----------------------+--------+
| Variable_name        | Value  |
+----------------------+--------+
| Innodb_rows_deleted  | 0      |
| Innodb_rows_inserted | 3      |
| Innodb_rows_read     | 12     |
| Innodb_rows_updated  | 1      |
+----------------------+--------+
```

### Show what every connection is doing

```sql
mysql> SHOW PROCESSLIST;
+----+------+-----------+-------+---------+------+----------+------------------+
| Id | User | Host      | db    | Command | Time | State    | Info             |
+----+------+-----------+-------+---------+------+----------+------------------+
|  5 | root | localhost | hello | Query   |    0 | starting | SHOW PROCESSLIST |
+----+------+-----------+-------+---------+------+----------+------------------+
```

### Show internal InnoDB state

```sql
mysql> SHOW ENGINE INNODB STATUS\G
*************************** 1. row ***************************
  Type: InnoDB
  Name:
Status:
=====================================
2026-04-27 15:55:00 0x7fa1b8001700 INNODB MONITOR OUTPUT
=====================================
Per second averages calculated from the last 10 seconds
... (huge dump of internal state — transactions, locks, buffer pool, etc.) ...
```

This is the most important admin command. Every MySQL DBA stares at this output regularly.

### Replication status

```sql
mysql> SHOW REPLICA STATUS\G
*************************** 1. row ***************************
               Slave_IO_State: Waiting for source to send event
                  Source_Host: 10.0.0.5
                  Source_User: repl
                  Source_Port: 3306
                Connect_Retry: 60
              Source_Log_File: binlog.000023
          Read_Source_Log_Pos: 12345
            Replica_IO_Running: Yes
           Replica_SQL_Running: Yes
       Seconds_Behind_Source: 0
                ...
```

(Old syntax: `SHOW SLAVE STATUS\G`. Both still work.)

### Show binary logs

```sql
mysql> SHOW BINARY LOGS;
+---------------+-----------+-----------+
| Log_name      | File_size | Encrypted |
+---------------+-----------+-----------+
| binlog.000001 | 1572864   | No        |
| binlog.000002 | 891234    | No        |
+---------------+-----------+-----------+
```

### Show events from a binlog

```sql
mysql> SHOW BINLOG EVENTS IN 'binlog.000002' LIMIT 10\G
*************************** 1. row ***************************
   Log_name: binlog.000002
        Pos: 4
 Event_type: Format_desc
  Server_id: 1
End_log_pos: 124
       Info: Server ver: 8.0.36-0ubuntu0.22.04.1-log, Binlog ver: 4
... (more events) ...
```

### EXPLAIN a query

```sql
mysql> EXPLAIN SELECT * FROM pets WHERE name = 'Rex';
```

(See **Query Optimization**.)

### EXPLAIN FORMAT=JSON

```sql
mysql> EXPLAIN FORMAT=JSON SELECT * FROM pets WHERE name = 'Rex'\G
```

### EXPLAIN ANALYZE

```sql
mysql> EXPLAIN ANALYZE SELECT * FROM pets WHERE name = 'Rex';
```

### ANALYZE TABLE

Update the optimizer's statistics about row count and key distribution:

```sql
mysql> ANALYZE TABLE pets;
+-------------+---------+----------+----------+
| Table       | Op      | Msg_type | Msg_text |
+-------------+---------+----------+----------+
| hello.pets  | analyze | status   | OK       |
+-------------+---------+----------+----------+
```

### OPTIMIZE TABLE

Rebuild the table to defragment and reclaim space:

```sql
mysql> OPTIMIZE TABLE pets;
+------------+----------+----------+-------------------------------+
| Table      | Op       | Msg_type | Msg_text                      |
+------------+----------+----------+-------------------------------+
| hello.pets | optimize | note     | Table does not support optimize, doing recreate + analyze |
| hello.pets | optimize | status   | OK                            |
+------------+----------+----------+-------------------------------+
```

For InnoDB, this is essentially `ALTER TABLE ... ENGINE=InnoDB`.

### CHECK TABLE

```sql
mysql> CHECK TABLE pets;
+-------------+-------+----------+----------+
| Table       | Op    | Msg_type | Msg_text |
+-------------+-------+----------+----------+
| hello.pets  | check | status   | OK       |
+-------------+-------+----------+----------+
```

### REPAIR TABLE

For MyISAM tables that got corrupted:

```sql
mysql> REPAIR TABLE legacy_table;
```

InnoDB doesn't support REPAIR — InnoDB self-heals from the redo log on restart.

### FLUSH TABLES

Close all open tables and force them to be reopened:

```sql
mysql> FLUSH TABLES;
```

Useful before backups.

### FLUSH PRIVILEGES

Reload the grant tables:

```sql
mysql> FLUSH PRIVILEGES;
```

### SET a global variable

```sql
mysql> SET GLOBAL max_connections = 500;
mysql> SET PERSIST max_connections = 500;  -- 8.0+ survives restart
```

### Show grants for a user

```sql
mysql> SHOW GRANTS FOR 'app'@'%';
+-----------------------------------------------+
| Grants for app@%                              |
+-----------------------------------------------+
| GRANT USAGE ON *.* TO `app`@`%`               |
| GRANT SELECT, INSERT ON `hello`.* TO `app`@`%`|
+-----------------------------------------------+
```

### Quick health check

```bash
$ mysqladmin ping -u root -p
Enter password:
mysqld is alive

$ mysqladmin status -u root -p
Enter password:
Uptime: 1234  Threads: 2  Questions: 12345  Slow queries: 5
Opens: 200  Flush tables: 1  Open tables: 50  Queries per second avg: 9.999
```

### Auto-repair all databases

```bash
$ mysqlcheck --auto-repair --all-databases -u root -p
```

### Backup with mysqldump

```bash
$ mysqldump --single-transaction --routines --triggers --events \
            --master-data=2 hello > hello.sql
$ ls -lh hello.sql
-rw-r--r-- 1 me me 12K Apr 27 16:05 hello.sql
```

### Backup with mysqlpump

```bash
$ mysqlpump --default-parallelism=4 hello > hello.sql
```

### Backup with mydumper

```bash
$ mydumper -B hello -o /backup -t 8
$ ls /backup
hello-schema-create.sql  hello.pets-schema.sql  hello.pets.00000.sql.gz
```

### Backup with xtrabackup

```bash
$ xtrabackup --backup --target-dir=/backup -u root -p
$ xtrabackup --prepare --target-dir=/backup
```

### Read a binlog as text

```bash
$ mysqlbinlog binlog.000001 | head -20
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
DELIMITER /*!*/;
# at 4
#260427 15:30:00 server id 1  end_log_pos 124 CRC32 0xabc12345  Start: binlog v 4...
# at 124
#260427 15:30:01 server id 1  end_log_pos 250 CRC32 0xdef67890  Anonymous_GTID...
SET @@SESSION.GTID_NEXT='ANONYMOUS'/*!*/;
... (lots more) ...
```

### Print the resolved config

```bash
$ my_print_defaults mysqld
--datadir=/var/lib/mysql
--socket=/var/run/mysqld/mysqld.sock
--log-error=/var/log/mysql/error.log
--pid-file=/var/run/mysqld/mysqld.pid
--bind-address=0.0.0.0
```

### Aggregate slow queries

```bash
$ pt-query-digest /var/log/mysql/slow.log
```

### Online schema change

```bash
$ pt-online-schema-change --alter "ADD INDEX (name)" \
    D=hello,t=pets --execute
```

This rebuilds the table without locking it for writes. Used for big tables in production.

### Compare a source's tables with a replica's

```bash
$ pt-table-checksum --replicate=hello.checksums \
    --databases=hello h=source,u=root,p=...
```

Detects replicas that have drifted from the source.

## Common Confusions

### utf8 vs utf8mb4

This one breaks production every week somewhere. In MySQL, the character set named **`utf8`** is **NOT real UTF-8**. It's a 3-byte subset of UTF-8 that only covers the Basic Multilingual Plane. It cannot store emoji or many CJK characters or various other modern Unicode codepoints.

The real, full UTF-8 in MySQL is called **`utf8mb4`** (mb4 = multi-byte 4).

**Always use utf8mb4. Never use utf8.**

In 8.0, the default character set was changed to `utf8mb4` (it was `latin1` before that). If you're inheriting an old database, check:

```sql
SELECT @@character_set_server, @@collation_server;
```

If it says `utf8`, fix it. To convert a table:

```sql
ALTER TABLE pets CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

### CHAR padding

`CHAR(10)` always stores 10 characters. If you put `'hi'`, MySQL pads with 7 spaces and stores `'hi        '`. When you read it back, MySQL **strips the trailing spaces** by default (the `PAD CHAR TO FULL LENGTH` mode is OFF). So you get `'hi'` back.

This means `CHAR` and `VARCHAR` look the same when reading but differ on disk. Use VARCHAR for almost everything.

### NOT NULL DEFAULT '0' on an INT

Means "if not specified, set it to 0." It does **not** mean "0 is treated as NULL." 0 is a real number. If you want "0 is treated as NULL," use the `IFNULL()` function or store NULL.

### AUTO_INCREMENT gaps after rollback

```sql
START TRANSACTION;
INSERT INTO pets (name, kind) VALUES ('A', 'cat');
INSERT INTO pets (name, kind) VALUES ('B', 'cat');
ROLLBACK;
INSERT INTO pets (name, kind) VALUES ('C', 'cat');
SELECT id FROM pets WHERE name = 'C';
-- shows id = 3, not id = 1
```

The IDs from rolled-back inserts are not reused. This is by design — InnoDB doesn't want to scan the table to find gaps. If you absolutely need gap-free sequences, manage them yourself with a separate sequence table.

### InnoDB clustered PK design

Because InnoDB stores rows in PK order, the PK choice has a huge impact on performance:

- **Sequential PK (auto-increment INT/BIGINT)**: inserts always go to the end. Fast, no fragmentation.
- **Random PK (UUIDv4)**: inserts go all over the place. Slow, fragmented, makes the buffer pool sad.

If you need UUIDs, consider UUIDv7 (time-ordered) or store them as BINARY(16) instead of CHAR(36).

### InnoDB row format: DYNAMIC vs COMPACT vs REDUNDANT vs COMPRESSED

The default since 5.7 is **DYNAMIC**. It stores variable-length columns more efficiently and can put very long values entirely off-page.

- `COMPACT`: older default, slightly less efficient.
- `REDUNDANT`: ancient, don't use.
- `COMPRESSED`: zlib-compressed pages. Saves disk at CPU cost.

You probably don't need to touch this. DYNAMIC is fine.

### NULL vs '' vs 0 vs '0'

- `NULL` is "unknown" or "missing." Comparisons with NULL return NULL (not true, not false).
- `''` is an empty string. A real value of zero length.
- `0` is the integer zero.
- `'0'` is a string containing the character "0."

`'' = NULL` is NULL. `'' = 0` is true (because MySQL converts the empty string to 0 for comparison — yikes). `'0' = 0` is true. `0 IS NULL` is false.

Always use `IS NULL` and `IS NOT NULL` for null checks.

### LIMIT 5,10 vs LIMIT 10 OFFSET 5

These mean the same thing in MySQL:

```sql
SELECT * FROM pets LIMIT 5, 10;       -- skip 5, take 10
SELECT * FROM pets LIMIT 10 OFFSET 5; -- same
```

The first is MySQL-specific. The second is portable SQL. Prefer the second.

**Note:** the parameters in `LIMIT 5,10` are (offset, count), in that order. People mix this up with `(count, offset)` constantly.

### The optimizer doesn't always use your index

Sometimes you `CREATE INDEX` and `EXPLAIN` says it's not being used. Reasons:

- The index isn't selective enough — if 50% of rows match, a table scan is faster.
- Statistics are stale — run `ANALYZE TABLE`.
- The query has a function on the column, killing the index.
- The data type is wrong — string column being compared to a number.
- The index is too new — the optimizer hasn't decided to like it yet.

Use `FORCE INDEX` to test whether forcing the index helps. If yes, fix the underlying reason.

### SHOW WARNINGS

After many statements, MySQL silently coerces types or truncates values. Always run:

```sql
SHOW WARNINGS;
```

After a suspicious `INSERT` or `UPDATE`. You'll see messages like "Data truncated for column 'x' at row 1" that you'd otherwise miss.

### sql_mode strict_all_tables vs strict_trans_tables

- `STRICT_TRANS_TABLES`: be strict about bad data, but only for transactional tables (InnoDB). For non-transactional tables, give a warning instead of an error.
- `STRICT_ALL_TABLES`: be strict about bad data, always.

Modern best practice: `STRICT_ALL_TABLES` (you're not using non-transactional tables, right?). But `STRICT_TRANS_TABLES` is the default and good enough.

### MySQL REPEATABLE READ vs Postgres REPEATABLE READ

Same name, different behavior:

- MySQL InnoDB REPEATABLE READ uses MVCC consistent reads + next-key locking. Phantom-free for SELECT, allows write skew.
- Postgres REPEATABLE READ is snapshot isolation. Phantom-free for SELECT, also allows write skew. But Postgres SERIALIZABLE is SSI (Serializable Snapshot Isolation), which detects write skew.

Both have the same name and the same row-level guarantees for reads. Where they differ is in serialization conflicts. This trips people moving between the two.

### MyISAM doesn't support transactions or FK

If you `START TRANSACTION; INSERT INTO myisam_table ...; ROLLBACK;`, the insert is **not** rolled back. MyISAM ignores transaction commands. Don't expect it to behave like InnoDB.

MyISAM also doesn't enforce foreign keys — if you declare them, MySQL silently ignores the constraint.

### Group Replication vs old async

- Old async (and semi-sync): one source, many replicas, replicas might lag behind.
- Group Replication: a group of nodes with consensus on every write. Strongly consistent, no replica lag in the safe sense, automatic failover. Used in InnoDB Cluster.

GR has some restrictions (only InnoDB tables, primary keys required, no SERIALIZABLE isolation). Trade-offs.

### binlog position vs GTID

- Binlog position: file name + offset. e.g. `binlog.000023:1234567`. Fragile across restarts and across replicas.
- GTID: globally unique label. e.g. `3E11FA47-71CA-11E1-9E33-C80AA9429562:23`. Identifies a transaction no matter where it lands. Stable.

For new replication setups, always use GTIDs.

## Vocabulary

| Term | Plain English |
|---|---|
| MySQL | Open-source relational database, owned by Oracle. |
| mysqld | The MySQL server process (the "clerk in front"). |
| mysql (cli) | The MySQL command-line client. |
| mysqladmin | Admin command-line tool: ping, status, shutdown, flush. |
| mysqldump | Logical backup tool that produces SQL text. |
| mysqlpump | Newer multi-threaded version of mysqldump (deprecated 8.0). |
| mysqlbinlog | Tool to read and replay binary logs. |
| mysqlsh | MySQL Shell. JavaScript/Python REPL plus admin utilities. |
| mysql_config | Helper that prints compile flags for client libs. |
| my.cnf | The MySQL config file, usually at /etc/mysql/my.cnf. |
| mysqld --verbose --help | Dumps every config option and its current value. |
| MariaDB | Fork of MySQL by the original founders, MariaDB Foundation. |
| Percona Server | Closely-compatible enhanced fork of MySQL by Percona. |
| Percona XtraDB | Percona's fork of InnoDB with extra performance features. |
| Percona Toolkit | Suite of MySQL admin tools (pt-query-digest, pt-osc, etc.). |
| mydumper/myloader | Third-party fast multi-threaded backup pair. |
| sysbench | Standard MySQL/Postgres benchmarking tool. |
| mysqlslap | Built-in load-testing tool. Less popular than sysbench. |
| MySQL Workbench | GUI client for MySQL. |
| phpMyAdmin | Web-based MySQL admin GUI in PHP. |
| Adminer | Single-file PHP web admin tool. Lightweight. |
| MySQL Router | TCP proxy that auto-routes to InnoDB Cluster nodes. |
| Connector/J | The official Java JDBC driver for MySQL. |
| mysql-connector-python | Official Python driver for MySQL. |
| go-sql-driver | Popular Go MySQL driver (`github.com/go-sql-driver/mysql`). |
| libmysqlclient | C library that ships with MySQL for client apps. |
| MySQL Group Replication | Plugin for synchronous Paxos-based replication. |
| InnoDB Cluster | Group Replication + Router + Shell, all bundled. |
| NDB Cluster | Distributed shared-nothing engine, "MySQL Cluster." |
| InnoDB | The default storage engine since 5.5. ACID, row locks, FKs. |
| MyISAM | The legacy default. No transactions, table locks. |
| Aria | MariaDB's crash-safe MyISAM replacement. |
| MEMORY engine | Storage engine that lives in RAM only. |
| ARCHIVE | Compressed, append-only storage engine. |
| FEDERATED | Engine that forwards queries to a remote MySQL server. |
| BLACKHOLE | Engine that discards data; used as a replication relay. |
| ROCKSDB engine | RocksDB-backed engine in Percona/MariaDB; LSM tree. |
| TokuDB | Old fractal-tree engine, deprecated. |
| Engine | The actual storage backend for a table. |
| Storage engine | Same as engine. |
| Plugin | Loadable module that extends the server (auth, engines, etc.). |
| Table | A folder of rows (= papers). |
| Row | One paper in a folder, with values for each column. |
| Column | One labeled box on every paper. |
| Primary key | The unique label for each paper. Defines the clustered index. |
| Unique key | A constraint that no two papers share this column's value. |
| Foreign key | A pointer from one folder's paper to another folder's paper. |
| ON DELETE CASCADE | When parent is deleted, also delete children. |
| ON DELETE SET NULL | When parent is deleted, set child's FK to NULL. |
| ON DELETE RESTRICT | Refuse to delete if children exist. (Default.) |
| ON DELETE NO ACTION | Same as RESTRICT in MySQL. |
| ON UPDATE | Same idea but for UPDATEs to the parent's PK. |
| NOT NULL | Constraint: this box must have a value. |
| DEFAULT | The value used when an INSERT doesn't specify the column. |
| AUTO_INCREMENT | Column that auto-fills with the next integer. |
| GENERATED ALWAYS AS | A computed column based on an expression. |
| Virtual column | Generated column not stored on disk; computed on read. |
| Stored column | Generated column stored on disk and indexable. |
| JSON column | Native JSON type since 5.7 with binary encoding. |
| JSON_EXTRACT | Function to pull a value out of JSON by path. |
| JSON_TABLE | Turn JSON arrays into table rows (8.0+). |
| JSON path expression | A `$.field.array[0]` selector for JSON. |
| Geometry | Spatial data type for points, lines, polygons. |
| Spatial index | R-tree index for geometry columns. |
| R-tree | Tree where each node bounds a rectangle of space. |
| B+ tree | Tree where only leaves hold data and leaves are linked. |
| B-tree | Generic name for the tree family; B+ is most common in DBs. |
| Clustered index | Index whose leaves are the actual rows. (InnoDB PK.) |
| Secondary index | Non-PK index; leaves point at PK values. |
| Covering index | Index that has every column the query needs. |
| Unique index | Index that enforces uniqueness on the indexed columns. |
| FULLTEXT index | Index for word-based text search. |
| Hash index | Constant-time lookup, no range support; MEMORY only. |
| Composite index | Index on multiple columns. |
| Leftmost-prefix rule | Composite index works only for the leftmost column subset. |
| Index hint | Directive telling the optimizer which index to use. |
| FORCE INDEX | Hint: prefer this index strongly. |
| USE INDEX | Hint: only consider this index (or no index). |
| IGNORE INDEX | Hint: don't consider this index. |
| STRAIGHT_JOIN | Force the join order to match the SQL textual order. |
| Nested loop join | For each row in A, scan B for matches. |
| Block nested loop | Buffer many rows from A and scan B once. |
| Batched key access | Variant of NLJ that batches index lookups. |
| Hash join | Build a hash table from A, probe with B. (8.0+) |
| Index condition pushdown (ICP) | Apply WHERE filters at the engine level. |
| MRR (multi range read) | Optimize range reads by sorting page accesses. |
| Filesort | MySQL sorting result rows because no index helps. |
| Temporary table on disk | Intermediate result that overflowed RAM. |
| Derived table | Subquery in FROM clause acts as an inline table. |
| CTE | Common Table Expression; `WITH x AS (...) SELECT ...` (8.0+). |
| Recursive CTE | CTE that refers to itself; used for hierarchies. |
| Window function | Function over a set of rows defined by `OVER (...)` (8.0+). |
| Lateral derived table | Derived table that can reference outer columns (8.0.14+). |
| Materialized | Stored as a temp table, computed once. |
| optimizer_trace | Built-in trace showing the optimizer's decisions. |
| performance_schema | Schema with low-overhead instruments and counters. |
| sys schema | Pre-built views over performance_schema for humans. |
| INFORMATION_SCHEMA | Standard metadata views (tables, columns, indexes). |
| SHOW PROCESSLIST | List active connections and what they're doing. |
| SHOW ENGINE INNODB STATUS | Detailed InnoDB internal state report. |
| SHOW VARIABLES | List configuration variables. |
| SHOW STATUS | List runtime counters. |
| SHOW BINLOG EVENTS | List events in a binary log file. |
| SHOW MASTER STATUS | Show current binlog file and position (legacy term). |
| SHOW BINARY LOG STATUS | Modern equivalent of SHOW MASTER STATUS. |
| SHOW REPLICA STATUS | Show replication state on a replica (modern). |
| SHOW SLAVE STATUS | Same, legacy term. |
| GTID | Global Transaction Identifier. UUID:N format. |
| GTID_MODE | Server variable to enable/disable GTID. |
| ENFORCE_GTID_CONSISTENCY | Reject statements unsafe for GTID replication. |
| server_id | Unique numeric ID for each server in a topology. |
| log_bin | Path to the binary log files. |
| binlog_format | ROW (default), STATEMENT, or MIXED. |
| binlog_row_image | FULL (default), MINIMAL, or NOBLOB. |
| sync_binlog | How often to fsync the binlog. 1 = on every commit. |
| innodb_flush_log_at_trx_commit | 1 = ACID, 2/0 = faster but lossier. |
| innodb_buffer_pool_size | RAM for InnoDB's page cache. The biggest knob. |
| innodb_log_file_size | Size of each redo log file. Bigger = faster writes. |
| innodb_log_files_in_group | Number of redo log files (typically 2). |
| innodb_io_capacity | InnoDB's storage IOPS budget hint. |
| innodb_thread_concurrency | Limit InnoDB's concurrent threads. 0 = unlimited. |
| innodb_lock_wait_timeout | Max seconds to wait for a row lock. Default 50. |
| innodb_print_all_deadlocks | Log every deadlock to the error log. |
| table_open_cache | Cache of open table handles. |
| table_definition_cache | Cache of parsed CREATE TABLE definitions. |
| max_connections | Max simultaneous client connections. |
| thread_cache_size | Cache of idle worker threads. |
| query_cache | Old per-query result cache. Removed in 8.0. |
| tmp_table_size | Max in-memory size for internal temp tables. |
| max_heap_table_size | Same, for explicit MEMORY tables. |
| sort_buffer_size | Per-connection buffer for sorting. |
| join_buffer_size | Per-connection buffer for unindexed joins. |
| read_rnd_buffer_size | Buffer for sorted-then-randomly-read rows. |
| read_buffer_size | Buffer for sequential scans. |
| key_buffer_size | MyISAM index cache. Irrelevant for InnoDB. |
| max_allowed_packet | Max single packet size between client and server. |
| wait_timeout | Idle non-interactive connection timeout. |
| interactive_timeout | Idle interactive (mysql cli) connection timeout. |
| sql_mode | Strictness level for accepting bad data and behaviors. |
| character_set_server | Default character set for new databases. |
| collation_server | Default collation for new databases. |
| time_zone | Server's time zone setting. |
| lower_case_table_names | 0 = case-sensitive, 1 = lowercase, 2 = stored as-is. |
| Replication | The mechanism for keeping a replica in sync with a source. |
| Master | Old term for the source server. |
| Slave | Old term for the replica server. |
| Primary | New term for source (or for the writable node in GR). |
| Replica | The new term for slave. |
| Async | Replication where the source doesn't wait for replicas. |
| Semi-sync | Source waits for at least one replica's ack of receipt. |
| Sync | Source waits for replicas to fully apply (Group Replication). |
| Parallel replication | Multiple replica threads applying transactions in parallel. |
| MTS (Multi-Threaded Slave) | Old name for parallel replica apply. |
| GTID auto-positioning | Replica figures out where to start by GTID. |
| CHANGE REPLICATION SOURCE TO | Modern command to point a replica at a source. |
| CHANGE MASTER TO | Old equivalent of CHANGE REPLICATION SOURCE TO. |
| START REPLICA | Start the replication threads. |
| STOP REPLICA | Stop the replication threads. |
| RESET REPLICA | Erase replica state and restart. |
| SHOW REPLICAS | List replicas connected to this source. |
| RESET BINARY LOGS AND GTIDS | Wipe binlogs and GTID history (dangerous). |
| PURGE BINARY LOGS | Delete old binlog files older than X. |
| FLUSH BINARY LOGS | Close current binlog and start a new one. |
| Semi-sync plugin | Plugin pair (source + replica) for semi-sync replication. |
| group_replication plugin | Loadable plugin for MySQL Group Replication. |
| group_replication_primary_member | The current writable node in single-primary GR. |
| group_replication_consistency | Consistency level: EVENTUAL, BEFORE, AFTER, BEFORE_AND_AFTER. |
| mysqlrouter | Smart TCP proxy that knows InnoDB Cluster topology. |
| ProxySQL | Popular external MySQL proxy with caching, rewriting. |
| MaxScale | MariaDB's external proxy/load-balancer. |
| Vitess | YouTube's MySQL sharding system, now CNCF. |
| ssl_cert | Path to server's SSL certificate. |
| ssl_key | Path to server's SSL private key. |
| ssl_ca | Path to the CA cert that signed clients/server certs. |
| require_secure_transport | Require all connections to use TLS. |
| caching_sha2_password | Default 8.0 password authentication plugin. |
| mysql_native_password | Legacy password authentication plugin. |
| sha256_password | SHA-256-based plugin (predecessor to caching_sha2). |
| X Protocol port 33060 | MySQL Document Store port (X DevAPI / mysqlsh). |
| MySQL Document Store | NoSQL-style API to MySQL using JSON collections. |
| Buffer pool | InnoDB's RAM cache of pages. |
| Redo log | InnoDB's write-ahead log on disk for crash recovery. |
| Undo log | InnoDB's record of how to undo each transaction. |
| Doublewrite buffer | InnoDB's torn-page protection on disk. |
| Change buffer | InnoDB optimization for non-unique secondary index changes. |
| Adaptive hash index | InnoDB optimization that builds a hash on hot indexes. |
| Page | The 16 KB unit InnoDB reads and writes. |
| Tablespace | A file holding tables; default is `ibdata1` or per-table `.ibd`. |
| innodb_file_per_table | One `.ibd` file per InnoDB table (default). |
| ibdata1 | The system tablespace file (older, shared across tables). |
| Doublewrite | Writing every page twice for atomic page write protection. |
| Crash recovery | InnoDB rebuilds in-memory state from redo log on startup. |
| Read view | InnoDB's MVCC snapshot for a consistent read. |
| MVCC | Multi-Version Concurrency Control: keep old versions of rows. |
| Slow query log | Log of queries that took longer than long_query_time. |
| General query log | Log of every query (very verbose). |
| Error log | Log of server errors and warnings. |
| Audit log | Plugin-provided log of every connection and query. |
| Pluggable authentication | The plugin system for authentication methods. |

## Try This

Five experiments. Don't just read them — type them in. Watching the output happen is the whole point.

### Experiment 1: prove that InnoDB rolls back

```sql
mysql> CREATE DATABASE lab; USE lab;
mysql> CREATE TABLE t (id INT PRIMARY KEY, v VARCHAR(20)) ENGINE=InnoDB;
mysql> INSERT INTO t VALUES (1, 'first');
mysql> START TRANSACTION;
mysql> UPDATE t SET v = 'changed' WHERE id = 1;
mysql> SELECT * FROM t;
+----+---------+
| id | v       |
+----+---------+
|  1 | changed |
+----+---------+
mysql> ROLLBACK;
mysql> SELECT * FROM t;
+----+-------+
| id | v     |
+----+-------+
|  1 | first |
+----+-------+
```

The rollback brought back the original. Now try the same thing with `ENGINE=MyISAM`. The rollback does nothing. The change sticks.

### Experiment 2: see the index work

```sql
mysql> CREATE TABLE big (id INT PRIMARY KEY, val INT);
mysql> INSERT INTO big VALUES (1, 100), (2, 200), (3, 300);
-- ... insert a million rows however you want, e.g. with a stored procedure
mysql> EXPLAIN SELECT * FROM big WHERE val = 500;
| 1 | SIMPLE | big | ALL | NULL | NULL | NULL | NULL | 1000000 | Using where |
mysql> CREATE INDEX idx_val ON big (val);
mysql> EXPLAIN SELECT * FROM big WHERE val = 500;
| 1 | SIMPLE | big | ref | idx_val | idx_val | 5 | const | 1 | NULL |
```

Watch type go from ALL to ref. Watch rows go from a million to one.

### Experiment 3: cause and observe a deadlock

In session A:

```sql
mysql> START TRANSACTION;
mysql> UPDATE pets SET age = 99 WHERE id = 1;
```

In session B:

```sql
mysql> START TRANSACTION;
mysql> UPDATE pets SET age = 99 WHERE id = 2;
mysql> UPDATE pets SET age = 99 WHERE id = 1;  -- waits for A
```

Back in session A:

```sql
mysql> UPDATE pets SET age = 99 WHERE id = 2;  -- deadlock!
ERROR 1213 (40001): Deadlock found when trying to get lock; try restarting transaction
```

InnoDB picked one to kill. Run `SHOW ENGINE INNODB STATUS\G` and find the `LATEST DETECTED DEADLOCK` section.

### Experiment 4: utf8 vs utf8mb4

```sql
mysql> CREATE TABLE bad (s VARCHAR(20)) CHARACTER SET utf8;
mysql> INSERT INTO bad VALUES (UNHEX('F09F9880'));  -- the smiley emoji 0x1F600
ERROR 1366 (HY000): Incorrect string value: '\xF0\x9F\x98\x80' for column 's'

mysql> CREATE TABLE good (s VARCHAR(20)) CHARACTER SET utf8mb4;
mysql> INSERT INTO good VALUES (UNHEX('F09F9880'));
Query OK, 1 row affected (0.00 sec)
mysql> SELECT s FROM good;
+----+
| s  |
+----+
| 😀 |
+----+
```

Right there: the difference between fake utf8 and real utf8mb4.

### Experiment 5: replicate to yourself

This is more involved. Set up two MySQL servers (or one server with two configs). On the source:

```sql
SHOW BINARY LOG STATUS;
+---------------+----------+--------------+------------------+-------------------+
| File          | Position | Binlog_Do_DB | Binlog_Ignore_DB | Executed_Gtid_Set |
+---------------+----------+--------------+------------------+-------------------+
| binlog.000003 | 4567     |              |                  | UUID:1-42         |
+---------------+----------+--------------+------------------+-------------------+
```

On the replica:

```sql
CHANGE REPLICATION SOURCE TO
  SOURCE_HOST='127.0.0.1', SOURCE_PORT=3306,
  SOURCE_USER='repl', SOURCE_PASSWORD='replpass',
  SOURCE_AUTO_POSITION=1;
START REPLICA;
SHOW REPLICA STATUS\G
```

Make a change on the source. Watch it appear on the replica within a second.

## Where to Go Next

You've now seen MySQL from the outside (filing cabinet, clerk, journal, replica) and the inside (InnoDB engine, B+ tree indexes, MVCC, locks, replication). What next?

- For more depth on **SQL** itself (joins, subqueries, window functions, CTEs), read **databases/sql**.
- For deep dives on **MySQL** specifically — performance tuning, advanced replication, cluster topologies — read **databases/mysql**.
- For comparison with **Postgres**, read **databases/postgresql** and **ramp-up/postgres-eli5**. Postgres is MySQL's main rival; the differences in JSON, extensions, and isolation are worth knowing.
- For lightweight database use, read **databases/sqlite**.
- For caching and key-value stores, read **databases/redis** and **ramp-up/redis-eli5**.
- For high-volume time-series workloads (where MySQL struggles), read **databases/time-series**.
- The legendary book is **High Performance MySQL** by Schwartz, Zaitsev, and Tkachenko. Newer editions cover 8.0.
- Percona's blog (percona.com/blog) has a steady stream of practical MySQL tuning posts.
- The official manual at dev.mysql.com/doc is the canonical reference. It's well-written but very dense.

If you got nothing else from this sheet, get this: MySQL is a clerk in front of a giant filing cabinet, the cabinet has drawers and folders and papers, every change is journaled, and a backup clerk in another room keeps an identical cabinet. Everything else is an elaboration on that picture.

## See Also

- **databases/mysql** — the full MySQL reference sheet (deep dive on tuning, replication, ops).
- **databases/postgresql** — the rival reference sheet.
- **databases/sqlite** — single-file embedded database.
- **databases/redis** — in-memory key-value store.
- **databases/sql** — the SQL language itself, joins, subqueries, window functions.
- **databases/time-series** — when MySQL isn't the right tool.
- **ramp-up/postgres-eli5** — Postgres for absolute beginners.
- **ramp-up/redis-eli5** — Redis for absolute beginners.
- **ramp-up/linux-kernel-eli5** — what's underneath your server OS.
- **ramp-up/tcp-eli5** — how clients reach the database in the first place.
- **ramp-up/docker-eli5** — how to package and run MySQL in containers.

## References

- **dev.mysql.com/doc** — the official MySQL Reference Manual. Every command, every variable, every error code.
- **High Performance MySQL** by Baron Schwartz, Peter Zaitsev, Vadim Tkachenko — the canonical book. 4th edition covers 8.0.
- **Effective MySQL** series by Ronald Bradford — concise focused books on backup, replication, query tuning.
- **MariaDB Knowledge Base** at mariadb.com/kb — official MariaDB docs; also a great cross-reference for MySQL.
- **Percona Blog** at percona.com/blog — practical posts on tuning, backup, replication, troubleshooting.
- **Database Internals** by Alex Petrov — covers B+ trees, LSM trees, replication theory, and consensus. Background reading that makes everything click.
