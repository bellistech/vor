# PostgreSQL — ELI5 (The Warehouse with Strict Rules)

> PostgreSQL is a giant warehouse with strict rules: every box has a label, every label points to a shelf, and a careful librarian fetches things for you so the warehouse never gets messy.

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` helps for the storage / fsync side)

This sheet is the very first stop for databases. You do not need to know SQL. You do not need to know what a database is. You do not need to know what a "table" is. By the end, you will know all of those things in plain English, and you will have typed real commands into a real `psql` terminal and watched real things happen.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your shell." If you see `mydb=#` at the start of a line, that means "type the rest of this line into the `psql` prompt." You do not type the prefix. The lines underneath that don't have a prefix are what your computer prints back at you. We call that "output."

## What Even Is PostgreSQL?

### Imagine a giant warehouse

Picture an enormous warehouse, the kind with rows and rows of shelves stretching out as far as you can see. Each shelf has a number painted on it. Each shelf is divided into bays. Each bay has labeled boxes. Inside each box is exactly one thing.

The warehouse has strict rules:

- **Every box must have a label.** No anonymous boxes.
- **Every label must be unique on its shelf.** You can't have two boxes labeled "Order #42" on the same shelf.
- **You cannot put a banana on a shelf marked "Books."** Each shelf only holds boxes of one kind.
- **You cannot just walk in and grab things.** You ask the librarian. The librarian fetches.
- **If you start rearranging boxes, you put up a "do not disturb" sign.** When you take the sign down, the new arrangement is visible to everyone. Until then, everyone else sees the old arrangement.

That warehouse is PostgreSQL.

The librarian is the **query planner**. Your shelves are **tables**. Your boxes are **rows**. The labels on the boxes are **columns**. The "do not disturb" signs are **transactions**. The kind-of-thing-on-the-shelf rules are **types** and **constraints**.

If you have ever organized a kitchen pantry — flour in one canister, sugar in another, beans in a third, all clearly labeled, all in matching containers, none mixed together — you have already understood the core of how PostgreSQL works.

### Imagine a really fussy filing cabinet

Here is another way to think about it. Imagine a filing cabinet at a doctor's office. There are dozens of drawers. Each drawer is for one kind of paperwork: patient records, billing, prescriptions, lab results. Each folder inside the drawer has the same fields filled in: name, date of birth, address, phone, allergies. Every folder is the same shape. You always know exactly where the "phone number" line is, because every folder follows the same template.

That template is a **schema**. The drawers are **tables**. The folders are **rows**. The fields on the folder are **columns**. The labels on the drawers and the rules about which folder shape goes in which drawer are **constraints**.

PostgreSQL is the most strict, most disciplined filing cabinet you have ever seen. It will refuse to file a folder if it doesn't match the template. It will refuse to put a folder in the wrong drawer. It will refuse to let you remove a folder if some other folder is pointing at it ("Hey, that allergy record references this patient — you can't delete the patient yet").

This strictness is a feature, not a bug. It means once data is in PostgreSQL, you can trust it. Nothing snuck in that doesn't fit. Nothing references something that doesn't exist. Nothing is half-written. The strictness is what makes the database a place you can trust.

### Why "relational"?

PostgreSQL is called a **relational database**. The "relational" part means rows in one table can relate to rows in another table.

Imagine an `orders` table and a `customers` table. Each order belongs to a customer. Each customer can have many orders. Instead of writing the customer's full name and address inside every single order, you write the customer's ID number on the order. The order says "I belong to customer #42." If you want the customer's name and address, you go look up customer #42 in the customers table.

That little ID number that points to another table is called a **foreign key**. It is the relationship. It is the "relational" in "relational database."

Picture two filing cabinets next to each other. The first has a folder for every customer, sorted by ID. The second has a folder for every order, and each order folder has a sticky note that says "see customer 42" or "see customer 87." When you want to know who ordered what, you grab the order folder, read the sticky note, walk to the other cabinet, and pull the customer folder. That walk-and-look-up is what databases call a **join.** PostgreSQL is extremely good at joins. It can link folders across many cabinets in a fraction of a second.

### Why "ACID"?

You will hear people say "PostgreSQL is ACID-compliant" like it's a magic spell. ACID stands for four properties:

- **A — Atomicity.** Every transaction is all-or-nothing. Either all your changes happen, or none of them do. There is no halfway state. Picture a money transfer: take $100 out of account A, put $100 into account B. If the database crashes between those two steps, you do not want $100 to disappear or $100 to appear from nowhere. ACID guarantees that either both changes happen, or neither does.
- **C — Consistency.** The database always moves from one valid state to another. If a constraint says "this column cannot be negative," there is no moment, ever, where a negative number sits in that column. The database refuses any change that would break a rule.
- **I — Isolation.** When two people are changing the database at the same time, neither sees the other's half-finished work. Your changes are like a private workspace until you say "commit," and only then do they become visible to everyone else.
- **D — Durability.** Once you commit, the change is written to disk and will survive a power loss. You can pull the plug on the server, and when it boots back up, your committed changes are still there.

PostgreSQL takes all four of these very seriously. It is famous for being the most strict, most rigorous, most paranoid mainstream relational database. People joke that "PostgreSQL won't let you shoot yourself in the foot" because of how many sanity checks it does.

### Picture: the warehouse from above

```
+---------------------------------------------------------------+
|                       The PostgreSQL Warehouse                |
+---------------------------------------------------------------+
|                                                               |
|   Loading dock (clients connect here):                        |
|     [psql] [pgAdmin] [your app] [pg_dump]                     |
|                                                               |
|   Front desk (the postmaster process):                        |
|     "Hi, who are you? OK, here's a worker."                   |
|                                                               |
|   Workers (one process per connection):                       |
|     [worker 1] [worker 2] [worker 3] ... [worker N]           |
|         |          |          |             |                |
|         v          v          v             v                |
|   Librarian (the query planner):                              |
|     "Best route to fetch what you asked for? This way."       |
|                                                               |
|   Shelves (tables):                                           |
|     +----------+  +----------+  +----------+  +----------+    |
|     | users    |  | orders   |  | products |  | logs     |    |
|     | rows...  |  | rows...  |  | rows...  |  | rows...  |    |
|     +----------+  +----------+  +----------+  +----------+    |
|                                                               |
|   Cross-reference cards (indexes):                            |
|     [users.email_idx] [orders.created_at_idx] ...             |
|                                                               |
|   Janitor (autovacuum):                                       |
|     "Cleaning up dead tuples ... done."                       |
|                                                               |
|   Logbook (write-ahead log = WAL):                            |
|     [every change written here BEFORE the data files change]  |
+---------------------------------------------------------------+
```

Keep this picture in your head. Every concept later in the sheet maps to one of those rooms.

## Tables, Rows, Columns, Types

### The shelf, the box, the label

A **table** is a shelf. Picture a shelf labeled `users`. On that shelf, every box has the same fixed shape. Every box is a `user`-sized box. You cannot put a `product`-sized box on the `users` shelf.

A **row** is a single box on the shelf. One row, one user. If you have ten thousand users, you have ten thousand boxes on the `users` shelf.

A **column** is a label on the box. Every box on the `users` shelf has the same labels: `id`, `email`, `created_at`, `is_active`. You can read the `email` label to get the user's email. You can read the `id` label to get the user's ID number.

A **type** is the shape of what fits behind a label. The `id` label only accepts whole numbers. The `email` label only accepts text up to some length. The `created_at` label only accepts a date and time. The `is_active` label only accepts true or false.

### Built-in types you will use every day

Here is the short list of types that cover 95% of everything you will ever do:

- **`integer`** (32-bit whole number, range about ±2.1 billion) — for IDs and counts.
- **`bigint`** (64-bit whole number, range up to 9 quintillion) — when 2.1 billion isn't enough.
- **`smallint`** (16-bit whole number, range ±32k) — for small enums or status codes.
- **`numeric(p, s)`** — exact decimal numbers. Use this for money. Never use `float` for money.
- **`real` / `double precision`** — floating-point. Fast, inexact. Use for scientific data, never for money.
- **`text`** — string of any length. The standard string type in PostgreSQL.
- **`varchar(n)`** — string up to `n` characters. Almost the same as `text` in PostgreSQL; use `text` unless you really need the length cap.
- **`char(n)`** — fixed-length string, padded with spaces. You probably don't want this. Use `text`.
- **`boolean`** — `true`, `false`, or `null`.
- **`date`** — calendar date with no time (e.g., `2026-04-27`).
- **`time`** — time of day with no date (e.g., `14:30:00`).
- **`timestamp`** — date and time without time zone.
- **`timestamptz`** — date and time **with** time zone. **Use this 99% of the time.** It stores everything in UTC under the hood and converts to your session's time zone when displayed.
- **`interval`** — a duration (e.g., `3 days 4 hours`).
- **`uuid`** — a 128-bit globally unique identifier (e.g., `550e8400-e29b-41d4-a716-446655440000`).
- **`bytea`** — raw bytes. For storing binary blobs (avoid this if you can; usually files belong on disk, not in the database).
- **`json` / `jsonb`** — JSON data. **Almost always use `jsonb`** (the binary form). It is faster to query and supports indexes. `json` is just text-stored-as-text and is slower.
- **`array`** — any type can be made into an array. `text[]` is an array of strings. `integer[]` is an array of integers.

### Custom types

You can also make your own types. Two ways:

- **`CREATE DOMAIN`** — give a base type a name and some constraints. Example: `CREATE DOMAIN positive_int AS integer CHECK (VALUE > 0);` and now you can use `positive_int` as a type and the database will refuse negative values.
- **`CREATE TYPE`** — make a fully new composite type or enum. Example: `CREATE TYPE order_status AS ENUM ('pending', 'paid', 'shipped', 'delivered', 'cancelled');` and now `order_status` is a type that only accepts those five values.

### Picture: a row inside a table

```
Table: users (one shelf)
+------+----------------------+---------------------+-----------+
|  id  | email                | created_at          | is_active |
+------+----------------------+---------------------+-----------+
|   1  | alice@example.com    | 2024-01-15 09:30:00 |   true    |
|   2  | bob@example.com      | 2024-01-16 11:42:11 |   true    |
|   3  | carol@example.com    | 2024-02-01 14:10:00 |   false   |
|   4  | dave@example.com     | 2024-02-15 16:55:42 |   true    |
+------+----------------------+---------------------+-----------+
                ^                     ^                  ^
                |                     |                  |
              column              column             column
              (text)              (timestamptz)     (boolean)

Each horizontal line above is one row (one box). Each vertical
column is one labeled drawer of every box.
```

## Primary Keys, Foreign Keys, Constraints

### The primary key — the one true label

Every box on a shelf needs a unique identifier. The unique identifier is the **primary key**. On the `users` shelf, the primary key is usually `id`. There can never be two rows with the same `id`. The database enforces this. If you try to insert a second row with `id = 5` and there is already an `id = 5`, the database refuses with an error: `ERROR: duplicate key value violates unique constraint "users_pkey"`.

Most tables have a primary key called `id` of type `bigint` or `uuid`. PostgreSQL has a special trick called a **sequence** (also called `serial` or `bigserial`) that auto-generates ascending integers for you. UUIDs are randomly generated and are great when you want to merge data from many sources without worrying about ID collisions.

### Foreign keys — pointers to another shelf

A **foreign key** is a column that says "I point at the primary key of another row in another table." Example: in the `orders` table, the column `user_id` is a foreign key pointing at `users.id`. If you try to insert an order with `user_id = 99999` and there is no user with id 99999, the database refuses: `ERROR: insert or update on table "orders" violates foreign key constraint`.

This means **the database guarantees referential integrity**. You can't have orphan orders. You can't have an order pointing to a deleted user. The database enforces the relationship.

You can also tell the database what to do when the parent row is deleted:

- **`ON DELETE CASCADE`** — if the user is deleted, also delete all their orders.
- **`ON DELETE SET NULL`** — if the user is deleted, set `user_id` to null on their orders (orphan them but keep them).
- **`ON DELETE RESTRICT`** (the default) — refuse to delete the user if any orders reference them.

### Other constraints

- **`UNIQUE`** — no two rows can have the same value in this column. Example: `email` should be unique on `users`.
- **`NOT NULL`** — this column must always have a value; null is forbidden.
- **`CHECK`** — a custom rule. Example: `CHECK (price > 0)` means price must be positive. Example: `CHECK (status IN ('pending', 'paid', 'shipped'))` means status must be one of three values.
- **`EXCLUSION`** — a generalized uniqueness rule. Example: "no two reservations can overlap in time for the same room." Used with range types and GiST indexes.

### Picture: the relationship

```
users                          orders
+----+----------------+        +----+----------+--------+
| id | email          |        | id | user_id  | total  |
+----+----------------+        +----+----------+--------+
|  1 | alice@...      |<-------|  1 |    1     |  19.99 |
|  2 | bob@...        |<-------|  2 |    1     |  42.50 |  (alice has 2 orders)
|  3 | carol@...      |<-------|  3 |    2     |   9.00 |  (bob has 1)
|  4 | dave@...       |        |  4 |    3     | 100.00 |  (carol has 1)
+----+----------------+        +----+----------+--------+
                                        ^
                                  foreign key
                              (orders.user_id -> users.id)
```

The arrows are not real columns; they represent the foreign key constraint. The database enforces them invisibly.

## Indexes

### The cross-reference cards

Imagine you have a million boxes on the `users` shelf and you want to find the user with `email = 'alice@example.com'`. Without an index, the librarian has to walk down the entire shelf, opening every single box, reading every single email, and comparing it to `alice@example.com`. That's a million boxes opened. We call that a **sequential scan**, or **Seq Scan**. For small shelves (a few hundred rows), it's fine. For a million rows, it's slow.

An **index** is a separate set of cross-reference cards. There is one card per row, sorted by the indexed column (e.g., `email`). The cards are organized into a tree. To find `alice@example.com`, the librarian walks the tree from top to bottom in maybe 20 steps and lands exactly on alice's card, which says "alice's box is on shelf X at position Y." The librarian walks straight there. 20 lookups instead of a million. We call that an **Index Scan**.

Indexes are how databases make queries fast. They are also how they make INSERT/UPDATE/DELETE slow, because every change to the table must also update every index. So you only build indexes you actually need.

### B-tree (the default, the workhorse)

A **B-tree** index is a balanced search tree. Picture a phone book: the names are sorted, and you can flip to "M" instantly without reading every page. B-trees do that for any sortable type. B-trees are great for:

- Equality lookups: `WHERE email = 'alice@...'`
- Range scans: `WHERE created_at > '2024-01-01'`
- Sorted output: `ORDER BY email` can use the index directly.
- Inequality (less commonly): `WHERE id > 100`.

99% of the indexes you will create are B-trees. The default index type if you don't specify.

### Picture: a B-tree

```
                              [ M ]
                              /   \
                           [F]     [T]
                          /  \    /   \
                       [B]   [I][P]   [W]
                      / | \  ...
                  Adams Brown Carter ...
                  ^
              actual row pointers (pointing at boxes on the shelf)
```

Each internal node has a small number of keys and pointers down to the next layer. Tree depth grows logarithmically: a billion-row index is only about 4-5 levels deep. That is why even huge tables stay fast.

### Hash

A **hash index** is great for one thing only: equality (`=`). It hashes the value and uses the hash as a direct address. Hash indexes do not support range scans, ordering, or anything other than `=`. Modern PostgreSQL hash indexes are crash-safe (since 10) but B-tree is almost always good enough. Use hash only if you have measured and seen a real benefit.

### GIN (Generalized Inverted Index)

**GIN** is for when one row contains many things and you want to search by one of them. Examples:

- An `array` column. You want to find rows where the array contains `42`.
- A `jsonb` column. You want to find rows where the JSON has `"role": "admin"`.
- A full-text search column (`tsvector`). You want to find rows containing the word `kernel`.
- A `tsvector` column. You want to find rows containing `'cat & mouse'` as a query.

GIN builds an inverted index: for each searchable element, it lists all rows that contain it. Like the index in the back of a textbook: "kernel — pages 12, 47, 91, 132."

### GiST (Generalized Search Tree)

**GiST** is a framework for tree-based indexes over weird data. Used for:

- Geometric data (with PostGIS).
- Full-text search.
- Range types (`tstzrange`, `int4range`).
- Exclusion constraints (the "no overlapping reservations" trick).

GiST is more flexible than B-tree but slower for plain ordered data. Use GiST when B-tree can't do what you need.

### BRIN (Block Range Index)

**BRIN** is for absolutely huge tables where the data is naturally ordered on disk. Example: a `logs` table where rows are inserted in order of `created_at` and never updated. BRIN stores a tiny summary per block range (e.g., "blocks 0-127 contain timestamps from 2024-01-01 to 2024-01-15"). To find a specific timestamp, BRIN narrows down to maybe a hundred blocks instead of billions.

BRIN indexes are tiny — kilobytes for a billion-row table. They are great for time-series or append-only tables. They are useless for randomly distributed data.

### SP-GiST (Space-Partitioned GiST)

**SP-GiST** is for unbalanced tree structures: quadtrees, radix trees. You probably won't reach for this until you do something specialized like IP-prefix lookup.

### Partial indexes (the elegant trick)

A **partial index** indexes only some of the rows, using a `WHERE` clause:

```sql
CREATE INDEX users_active_idx ON users (email) WHERE is_active = true;
```

Now the index only contains active users. It is smaller, faster to maintain, and the planner uses it whenever your query also filters on `is_active = true`. This is one of the most powerful tools in PostgreSQL.

### Expression indexes

You can index the result of a function:

```sql
CREATE INDEX users_lower_email_idx ON users (lower(email));
```

Now `WHERE lower(email) = 'alice@...'` can use the index. Without it, the planner can't use a normal `email` index because of the `lower()` call.

### Covering indexes (`INCLUDE`)

A **covering index** stores extra columns alongside the indexed key, so the database can answer the query without ever touching the actual row:

```sql
CREATE INDEX users_email_inc_idx ON users (email) INCLUDE (created_at, is_active);
```

When a query says `SELECT email, created_at FROM users WHERE email = '...'`, the database finds the entry in the index, reads `email` and `created_at` directly from the index entry, and returns. It never visits the heap (the actual table). This is called an **Index Only Scan** and it is the fastest kind of read.

### Picture: index scan vs sequential scan

```
Sequential scan (no index):
  Shelf:   [box][box][box][box][box][box][box][box][box][box]...
  Walker:   1    2    3    4    5    6    7    8    9    10
  Each box opened, read, compared. Linear in shelf size.

Index scan (B-tree on email):
  Index tree:        [M]
                     / \
                   [F] [T]
                   ...
                  /
  Lookup walk:    -> [E] -> [Eli] -> alice@... -> shelf pointer
  About log2(N) steps. For a billion rows, that's about 30 steps.
```

## Transactions and ACID

### BEGIN, COMMIT, ROLLBACK

A **transaction** is a chunk of work that should happen all-or-nothing. You start one with `BEGIN`. You make whatever changes you want. If everything looks good, you say `COMMIT` and the changes become permanent and visible to everyone. If something went wrong, you say `ROLLBACK` and all changes are undone as if they never happened.

```sql
BEGIN;
UPDATE accounts SET balance = balance - 100 WHERE id = 1;
UPDATE accounts SET balance = balance + 100 WHERE id = 2;
COMMIT;
```

If the database crashes between the two `UPDATE`s, both updates are rolled back when the database restarts. There is no halfway state. That is **atomicity**.

### Savepoints

A **savepoint** is a marker inside a transaction. You can roll back to a savepoint without rolling back the whole transaction:

```sql
BEGIN;
INSERT INTO log (msg) VALUES ('starting');
SAVEPOINT mid;
DELETE FROM users WHERE id = 7;  -- oops, didn't mean it
ROLLBACK TO SAVEPOINT mid;
INSERT INTO log (msg) VALUES ('phew');
COMMIT;
```

The `INSERT INTO log` calls both committed; the `DELETE` was undone because of the rollback to savepoint.

### Isolation levels

When two transactions run at the same time, they can see each other's work in different ways. PostgreSQL supports four levels:

- **READ UNCOMMITTED** — would let you see another transaction's uncommitted changes. PostgreSQL does not actually have this level; if you ask for it you get READ COMMITTED.
- **READ COMMITTED** (the default) — each statement in your transaction sees a snapshot taken at the moment that statement starts. So if you run `SELECT` twice, you might see different results because another transaction committed in between.
- **REPEATABLE READ** — your whole transaction sees a snapshot taken at its start. Every `SELECT` in the transaction sees the same world. New rows committed by others don't show up.
- **SERIALIZABLE** — strongest. The database guarantees that your transaction's outcome is the same as if all transactions had run one at a time, in some order. Sometimes the database has to abort your transaction with a `serialization_failure` error and you have to retry. Use this when correctness matters more than throughput.

You set the isolation level at the start of a transaction:

```sql
BEGIN ISOLATION LEVEL SERIALIZABLE;
-- ... work ...
COMMIT;
```

Most apps use READ COMMITTED by default and only reach for stronger when needed.

### Why this matters

Picture two cashiers at a bank. Both try to transfer money from account A at the same instant. Without isolation, both might read "account A has $200," both subtract $100, both think the new balance is $100, and now $100 has been stolen by the universe.

With isolation, the database makes one cashier wait, or aborts one of them, or somehow ensures the math comes out right. Serializable says "the result is as if the cashiers had taken turns one at a time."

## MVCC (Multi-Version Concurrency Control)

### The "do not disturb" sign explained

PostgreSQL never overwrites a row in place. When you update a row, PostgreSQL writes a brand new row right next to the old one and leaves the old one alone. The old row is marked "valid until transaction X." The new row is marked "valid from transaction Y."

Each row has two hidden columns:

- **`xmin`** — the transaction ID that created this version.
- **`xmax`** — the transaction ID that deleted this version (or 0 if still alive).

When a transaction reads, it filters: "show me rows where `xmin` is committed and is `<=` my transaction ID, AND (xmax is 0 OR xmax is greater than my ID OR xmax is not yet committed)."

The result: every transaction sees a consistent snapshot of the world at the moment it started, even while other transactions are inserting/updating/deleting. Readers never block writers. Writers never block readers (except for the same row). That is **MVCC** — Multi-Version Concurrency Control.

### Picture: the version chain

```
Time --->

Transaction 100: INSERT user alice
   row [xmin=100, xmax=0,    name='alice', email='alice@...']

Transaction 105: UPDATE alice's email to alice2@...
   row [xmin=100, xmax=105,  name='alice', email='alice@...']  <-- old version
   row [xmin=105, xmax=0,    name='alice', email='alice2@...'] <-- new version

Transaction 110: DELETE alice
   row [xmin=100, xmax=105,  name='alice', email='alice@...']  <-- old version
   row [xmin=105, xmax=110,  name='alice', email='alice2@...'] <-- now-deleted

Transaction 102 (started before 105) reading users:
   Sees the row with xmin=100, xmax=0 (because xmax was set after T102 started)
   Does not see the new version (xmin=105, after T102 started)

Transaction 108 (started after 105 committed) reading users:
   Sees the row with xmin=105, xmax=110 -- wait, xmax is committed too? No,
   T108 started before T110 committed, so it sees the live version.

Transaction 115 (started after 110 committed):
   Sees no live row for alice; she's deleted from this transaction's view.
```

This little number-juggling is the entire reason readers and writers don't block each other.

### The cost: dead tuples and bloat

The dead row versions don't disappear automatically. They sit in the table taking up space. We call them **dead tuples**. Over time, dead tuples make the table bigger than the live data needs to be. We call this **bloat**.

That is what `VACUUM` is for. It cleans up.

## VACUUM and Autovacuum

### The cleanup goblin

`VACUUM` walks through a table, finds rows that are dead (no transaction can see them anymore), and marks their space as reusable. Future `INSERT`s can reuse those slots instead of growing the file.

By default, `VACUUM` does **not** return disk space to the OS. It just makes the space within the file reusable. To return space to the OS, you need `VACUUM FULL`, which rewrites the whole table and **takes an exclusive lock**. Don't run `VACUUM FULL` on a busy table during business hours.

### Autovacuum

You almost never run `VACUUM` by hand. PostgreSQL has an **autovacuum** process that watches your tables and vacuums them automatically when bloat passes a threshold. You can tune the thresholds in `postgresql.conf`:

- `autovacuum_vacuum_threshold` — base minimum number of dead tuples (default 50).
- `autovacuum_vacuum_scale_factor` — fraction of table size that must be dead before vacuuming (default 0.2 = 20%).
- `autovacuum_naptime` — how often the autovacuum daemon wakes up to check (default 1 minute).

If your autovacuum is falling behind, you will see bloat grow. Symptoms: tables seem bigger than expected, queries get slower, `pg_stat_user_tables.n_dead_tup` is high.

### Variants of VACUUM

- **`VACUUM`** — cleans up dead tuples, allows space reuse.
- **`VACUUM FULL`** — rewrites the table, returns space to OS, locks the table.
- **`VACUUM (FREEZE)`** — freezes old transaction IDs to prevent xid wraparound (more on this in a sec).
- **`VACUUM (ANALYZE)`** — also updates statistics.
- **`VACUUM (VERBOSE)`** — prints what it did.

### Why FREEZE?

Transaction IDs in PostgreSQL are 32-bit. They wrap around every 4 billion transactions. To prevent old data from looking like future data after a wrap, PostgreSQL **freezes** old rows by stamping them with a special "this is older than everything" marker. Autovacuum does this in the background. If you have a database that has been running for years and never been vacuumed, you can hit the dreaded "transaction wraparound" emergency, and PostgreSQL will refuse new writes until you vacuum. Modern PostgreSQL does this in the background safely; you just need autovacuum to be working.

### ANALYZE

`ANALYZE` is a separate command that updates the table's statistics — histograms of column values, frequency of common values, density of distinct values. The query planner uses these to estimate how many rows a query will return and pick the right plan. Out-of-date stats lead to bad plans.

You almost never run `ANALYZE` by hand either; autovacuum does it. But after a big data load, manually running `ANALYZE` immediately is a good practice.

## EXPLAIN and Query Planning

### How to read a query plan

`EXPLAIN` shows you the planner's intended path. `EXPLAIN ANALYZE` actually runs the query and shows the real path.

```sql
EXPLAIN SELECT * FROM users WHERE email = 'alice@example.com';
```

Output:

```
                            QUERY PLAN
-------------------------------------------------------------------
 Index Scan using users_email_idx on users  (cost=0.42..8.44 rows=1 width=64)
   Index Cond: (email = 'alice@example.com'::text)
```

Read this from the bottom up:

- The bottom-most node is the data source. Here it's an `Index Scan` on `users_email_idx`.
- `Index Cond` shows what the index is doing.
- `cost=0.42..8.44` is the planner's cost estimate (start cost..total cost). Lower is faster.
- `rows=1` is the planner's estimate of how many rows will come out.
- `width=64` is the average width of a row in bytes.

### EXPLAIN ANALYZE (actually runs!)

```sql
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'alice@example.com';
```

```
                                 QUERY PLAN
-----------------------------------------------------------------------------
 Index Scan using users_email_idx on users
   (cost=0.42..8.44 rows=1 width=64)
   (actual time=0.041..0.042 rows=1 loops=1)
   Index Cond: (email = 'alice@example.com'::text)
 Planning Time: 0.123 ms
 Execution Time: 0.067 ms
```

Now you also get **actual** time and **actual** rows. If estimated rows = 1 but actual rows = 1000, your stats are out of date or the planner is confused.

### EXPLAIN (BUFFERS, FORMAT JSON)

For deeper insight:

```sql
EXPLAIN (ANALYZE, BUFFERS, VERBOSE, FORMAT TEXT)
  SELECT * FROM users WHERE email = 'alice@example.com';
```

`BUFFERS` shows how many disk blocks were read from the cache (`shared hit`) vs from disk (`shared read`). If you see `shared read=12000`, that's 12000 blocks pulled from disk and the query is slow because of it.

### Common plan node names

- **`Seq Scan`** — read every row in the table. Linear time. Bad for big tables, fine for small ones.
- **`Index Scan`** — walk an index, then fetch each matching row from the heap.
- **`Index Only Scan`** — walk the index and never touch the heap (because the index covers all needed columns and the visibility map says the row is fresh).
- **`Bitmap Heap Scan`** — first scan the index to build a bitmap of matching rows, then fetch them from the heap in physical order. Faster than Index Scan when many rows match.
- **`Bitmap Index Scan`** — the index half of a bitmap heap scan.
- **`Nested Loop`** — for each row in A, look up matching rows in B. Fast when one side is tiny.
- **`Hash Join`** — build a hash table of B's rows in memory, then probe it for each row in A. Fast when B fits in memory.
- **`Merge Join`** — both sides are pre-sorted; walk them in lockstep. Fast when both sides are big and sorted.
- **`Sort`** — sort the rows. Uses memory up to `work_mem`, spills to disk if bigger.
- **`Materialize`** — cache an inner subplan in memory because we'll re-scan it.
- **`Aggregate`** — `COUNT`, `SUM`, `AVG`, etc.
- **`HashAggregate`** — group rows using an in-memory hash table.
- **`GroupAggregate`** — group rows in pre-sorted order.
- **`Limit`** — stop after N rows.

### Picture: a query plan tree

```
SELECT u.email, COUNT(o.id)
  FROM users u
  JOIN orders o ON o.user_id = u.id
  WHERE u.is_active = true
  GROUP BY u.email
  ORDER BY COUNT(o.id) DESC
  LIMIT 10;

Plan tree (read bottom-up):

         Limit (10)
            |
         Sort (by count desc)
            |
         HashAggregate (group by email)
            |
         Hash Join (u.id = o.user_id)
        /                       \
   Bitmap Heap Scan         Seq Scan
   on users                 on orders
   (filter: is_active)
        |
   Bitmap Index Scan
   on users_active_idx
```

The planner picks each node based on cost estimates. The cost model accounts for sequential page reads (`seq_page_cost`, default 1.0), random page reads (`random_page_cost`, default 4.0 — meaning random I/O is 4x more expensive than sequential), and CPU cost per tuple.

## The Statistics System

### Why the planner needs stats

The planner has many possible plans for any query. To pick the cheapest, it estimates cost, and to estimate cost it needs to know things like "how many rows match `is_active = true`?" The answer comes from statistics gathered by `ANALYZE`.

PostgreSQL stores these in two places:

- **`pg_statistic`** — the raw table. You usually don't read this directly.
- **`pg_stats`** — a friendly view on top of it. Use this.

Examples of what's in `pg_stats`:

- **`null_frac`** — fraction of values that are null.
- **`avg_width`** — average size of values in bytes.
- **`n_distinct`** — estimated number of distinct values (negative means "fraction of total rows").
- **`most_common_vals`** — the most frequent values.
- **`most_common_freqs`** — their frequencies.
- **`histogram_bounds`** — a histogram of the rest of the values.

### default_statistics_target

The number of histogram bins (and most-common-values) is controlled by `default_statistics_target`, default 100. For columns with weird distributions, you can crank this up per column:

```sql
ALTER TABLE users ALTER COLUMN email SET STATISTICS 1000;
ANALYZE users;
```

Now PostgreSQL keeps 1000 bins instead of 100 for that column, leading to better estimates on weird data.

### Correlation

`pg_stats.correlation` says how correlated the column's order is with the table's physical order on disk. Values range from -1 to 1. Near 1 means rows on disk are roughly sorted by this column (great for `BRIN` and for cheap range scans). Near 0 means the column's values are scattered randomly across the table.

## WAL (Write-Ahead Log)

### Every change written first

The cardinal rule of crash-safe databases: **never modify a data file before recording the intended change in a log.** That log is the **WAL** — Write-Ahead Log.

When you do `UPDATE users SET email = 'new@...' WHERE id = 5`, PostgreSQL:

1. Writes a record to the WAL: "in transaction T, page P of users had row at offset O updated; old email was X, new email is Y."
2. Flushes the WAL to disk (calls `fsync()` so it's actually on the platter, not just in OS cache).
3. Modifies the in-memory page.
4. Returns "ok" to the client.
5. Eventually (later) writes the modified page to the data file during a checkpoint.

If the database crashes between step 4 and step 5, the WAL has the change recorded. On restart, PostgreSQL replays the WAL and reapplies the change. The data is safe.

This is why **`fsync = on`** is critical (and why turning it off makes PostgreSQL much faster but unsafe).

### Picture: WAL → checkpoint flow

```
                       Transaction commits
                              |
                              v
              +----------------------------------+
              |  WAL record written + fsync'd    |
              +----------------------------------+
                              |
               (clients see "commit ok" here)
                              |
                              v
              +----------------------------------+
              |  Modified pages in shared_buffers|
              |  (still dirty, not yet on disk)  |
              +----------------------------------+
                              |
                       periodic checkpoint
                              |
                              v
              +----------------------------------+
              |  Dirty pages flushed to disk     |
              |  (background bgwriter or         |
              |   checkpointer process)          |
              +----------------------------------+
                              |
                              v
                  WAL files older than the
                  last checkpoint can be
                  recycled or archived.

If a crash happens before the checkpoint, on restart:
  - PostgreSQL reads WAL from the last checkpoint forward
  - Replays each WAL record (idempotently)
  - Database is exactly as committed
```

### Checkpoint settings

- **`checkpoint_timeout`** — max time between checkpoints (default 5 minutes). Longer means less I/O, longer recovery.
- **`checkpoint_completion_target`** — fraction of `checkpoint_timeout` to spread the checkpoint write over (default 0.9 = 90%). Smooths I/O spikes.
- **`max_wal_size`** — soft cap on WAL files between checkpoints. Going over it triggers an early checkpoint.

### wal_level

`wal_level` controls how much info is in the WAL:

- **`minimal`** — only what's needed for crash recovery. No replication possible.
- **`replica`** — adds info needed for streaming replication and backups. Default since PG 10.
- **`logical`** — adds enough info for logical replication (publishing specific tables, decoded).

Most production servers run with `replica` or `logical`.

## Replication

### Streaming replication (physical)

The primary streams its WAL to one or more standby servers. The standbys replay the WAL byte-for-byte. They are exact replicas of the primary. They can serve read-only queries (called **hot standby**). They can take over (be **promoted**) if the primary fails.

The standby connects to the primary, says "I'm at WAL position X," and the primary streams new WAL records as they're produced. Lag is usually milliseconds.

### Logical replication

Instead of byte-for-byte, **logical replication** decodes the WAL into row-level changes and ships them. You publish specific tables on the primary; subscribers subscribe to those publications. This is cross-version safe, lets you replicate subsets, and is great for upgrades (run a new version as a logical subscriber, then switch).

```sql
-- on primary
CREATE PUBLICATION mypub FOR TABLE users, orders;

-- on subscriber
CREATE SUBSCRIPTION mysub
  CONNECTION 'host=primary user=replicator dbname=mydb'
  PUBLICATION mypub;
```

### Synchronous vs asynchronous

- **Asynchronous** (default) — primary commits as soon as WAL is local. Standby is some milliseconds behind.
- **Synchronous** — primary waits for at least one named standby to confirm WAL receipt before commit. Stronger durability, slightly slower commits.

### Cascading replication

A standby can act as a primary for further standbys. Useful for fanning out reads geographically.

### Failover tools

If the primary dies, you need to promote a standby and update DNS / load balancer. Doing this by hand is error-prone, so people use:

- **Patroni** — built on top of etcd or Consul, very popular.
- **pg_auto_failover** — Citus's tool, simple two-node + monitor.
- **repmgr** — older, still widely used.

### Picture: streaming replication topology

```
                  Application
                       |
                  load balancer
                  /    |    \
            (writes) (reads) (reads)
                /      |      \
          [Primary]<--lag-->[Standby1]
              |               |
              |WAL stream     |WAL replay
              v               v
              +----[Standby2]
                       |
                  cascading

If primary dies:
  - Failover tool detects loss (heartbeat lost)
  - Picks the most up-to-date standby
  - Promotes it (it stops being read-only)
  - Updates DNS / VIP / connection string
  - Other standbys re-attach to the new primary
```

## Connection Pooling

### Why not just open more connections?

Each PostgreSQL connection is a separate **process** on the server. Each process uses about 5-10 MB of RAM and consumes a backend slot. Default `max_connections` is 100. Creating a connection takes a few milliseconds (forking a process). If your web app opens a fresh connection for every HTTP request, you're forking thousands of processes a second, which is terrible.

Solution: a **connection pool**. The pool keeps a small set of long-lived connections to PostgreSQL and hands them out to your app on demand.

### PgBouncer

**PgBouncer** is the most popular pooler. It is tiny, fast, and supports three pool modes:

- **Session mode** — your app's connection holds a server connection for its whole lifetime. Same as no pool, really.
- **Transaction mode** — your app's connection holds a server connection only during a transaction. After commit/rollback, the server connection goes back to the pool. Most common mode.
- **Statement mode** — server connection released after every statement. Strict; you can't use prepared statements or transactions properly.

In transaction mode, you can have thousands of app connections sharing a pool of a few dozen server connections. Massive efficiency win.

Caveats: in transaction mode, certain PostgreSQL features (session-level state, prepared statements made in one transaction and used in another, advisory locks, `LISTEN`/`NOTIFY`) don't work, because the server connection switches between transactions.

### Pgpool

**Pgpool-II** is a fancier pooler that also does query routing (sending reads to standbys, writes to primary), failover, and load balancing. More complex than PgBouncer; pick PgBouncer first unless you specifically need Pgpool's features.

## Roles, Privileges, Row-Level Security (RLS)

### Roles

A **role** is a user or group. PostgreSQL doesn't really distinguish them; a role can have a password (login role, like a user) or not (group role).

```sql
CREATE ROLE alice LOGIN PASSWORD 'sekrit';
CREATE ROLE app_readers;
GRANT app_readers TO alice;
```

Now alice inherits privileges granted to `app_readers`.

### GRANT / REVOKE

```sql
GRANT SELECT ON users TO app_readers;
GRANT INSERT, UPDATE ON orders TO app_writers;
REVOKE DELETE ON orders FROM bob;
```

Privileges flow: cluster → database → schema → table → column. Each level has its own grants. To read a table, you typically need:

1. `CONNECT` on the database.
2. `USAGE` on the schema (default `public`).
3. `SELECT` on the table.

### Row-Level Security (RLS)

A **policy** restricts which rows a role can see/modify:

```sql
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

CREATE POLICY orders_owner_policy ON orders
  FOR ALL TO app_user
  USING (user_id = current_setting('app.current_user_id')::int);
```

Now when `app_user` does `SELECT * FROM orders`, they only see rows where `user_id` matches the session's `app.current_user_id`. The database silently rewrites the query to add the policy. RLS is how multi-tenant apps can share a single table while ensuring tenant A never sees tenant B's rows.

### Picture: the privilege chain

```
              cluster
                 |
                 +-- CONNECT privilege
                 v
              database (mydb)
                 |
                 +-- USAGE privilege
                 v
              schema (public, app, audit, ...)
                 |
                 +-- SELECT, INSERT, UPDATE, DELETE
                 v
              table (users, orders, ...)
                 |
                 +-- column-level GRANT (rare)
                 v
              column (email, total, ...)
                 |
                 +-- row-level policy filters which rows
                 v
              row
```

## Common Extensions

PostgreSQL is famously extensible. Here are extensions you should know:

- **`pg_stat_statements`** — tracks every query, normalizes parameters, gives total/mean/calls/io stats. The single most useful telemetry extension. Always install this.
- **`pgvector`** — vector data type and similarity search (cosine, L2, inner product). The hottest extension since the LLM boom; powers most "semantic search" features built on Postgres.
- **`PostGIS`** — geographic data. Points, lines, polygons, distance, intersection, GIS projections. Industry standard.
- **`pg_trgm`** — trigram-based fuzzy text matching. Powers `LIKE '%foo%'` with index support, `similarity()` for typo-tolerant search.
- **`hstore`** — old-school key-value blob. Mostly superseded by `jsonb`, but still around.
- **`uuid-ossp`** / **`pgcrypto`** — generate UUIDs and hashes. (PG 13+ has `gen_random_uuid()` built in.)
- **`citext`** — case-insensitive `text` type. Useful for emails, usernames.
- **`timescaledb`** — time-series extension. Adds hypertables (transparently partitioned by time). External, but excellent.
- **`citus`** — distributed PostgreSQL. Sharding for OLTP. Acquired by Microsoft.
- **`postgrest`** — not exactly an extension, but a tool that turns your Postgres schema into a REST API automatically.

Install: `CREATE EXTENSION pg_stat_statements;` (some need pre-installation on the server).

## Common Errors

These are the errors you'll see again and again. Memorize the fix.

### `ERROR: relation "X" does not exist`

You queried a table that doesn't exist, or it exists in a different schema. Fix: check spelling; check `search_path`; qualify with schema name (e.g., `app.users`).

### `ERROR: column "X" does not exist`

Same idea but for a column. Check spelling. PostgreSQL is case-sensitive when you use double quotes (`"User"`); unquoted identifiers are folded to lowercase. So `SELECT Email FROM users` becomes `email`. If the column was created as `"Email"`, you must quote it.

### `ERROR: duplicate key value violates unique constraint "X_pkey"`

You tried to insert a row with a primary key value that already exists. Fix: either let the database auto-generate the ID (use `DEFAULT` or sequence), or `ON CONFLICT DO UPDATE` for upsert behavior.

### `ERROR: insert or update on table "X" violates foreign key constraint`

You tried to insert/update a row whose foreign key points to a row that doesn't exist. Fix: insert the parent row first, or check the value.

### `ERROR: deadlock detected`

Two transactions each hold a lock the other wants. PostgreSQL aborts one of them. Fix: order your locks consistently (e.g., always lock rows by ascending ID); shorten transactions; retry on serialization failure.

### `ERROR: canceling statement due to lock timeout`

Your statement was waiting for a lock longer than `lock_timeout`. Fix: figure out who holds the lock (`pg_locks`), shorten their work, or raise the timeout.

### `ERROR: canceling statement due to statement timeout`

Your statement ran longer than `statement_timeout`. Fix: optimize the query, raise the timeout for that session, or break the work into smaller pieces.

### `ERROR: too many connections for role "X"`

Either you've hit `max_connections` or a per-role connection limit. Fix: use a connection pooler (PgBouncer); raise `max_connections` (only if you really have RAM for it).

### `ERROR: out of shared memory; HINT: You might need to increase max_locks_per_transaction`

You hit a transaction that wanted to lock more objects than the lock table allows. Common cause: a transaction touching thousands of partitions. Fix: raise `max_locks_per_transaction` (requires restart).

### `WARNING: pg_dump: aborting because of server version mismatch`

Your `pg_dump` is older than the server. Fix: use the `pg_dump` from the same major version as the server (or newer).

### `FATAL: password authentication failed for user "X"`

Wrong password, or `pg_hba.conf` doesn't have a matching rule. Check both.

### `FATAL: database "X" does not exist`

Self-explanatory. `\l` from psql to list databases. Connect to one that exists, then `CREATE DATABASE` if needed.

### `FATAL: no pg_hba.conf entry for host "..."`

The server doesn't have a rule allowing your IP/user/database combo. Edit `pg_hba.conf`, reload (`SELECT pg_reload_conf();`).

### `ERROR: cannot truncate a table referenced in a foreign key constraint`

Some other table has a FK pointing at this one. Fix: `TRUNCATE ... CASCADE` (truncates the referencing tables too — be careful!) or drop the FK first.

### `WARNING: there is already a transaction in progress`

You ran `BEGIN` inside a transaction. Harmless but a sign your code is confused.

## Hands-On

These commands run against a real PostgreSQL. You can spin one up locally with `brew install postgresql` (macOS) or `sudo apt install postgresql` (Debian/Ubuntu). Or use Docker: `docker run -e POSTGRES_PASSWORD=secret -p 5432:5432 postgres:16`.

### Connecting

```
$ psql -h localhost -U postgres -d mydb
psql (16.2)
Type "help" for help.

mydb=#
```

The `mydb=#` prompt means you're in. The `#` indicates you're a superuser (regular users see `=>`).

### List databases

```
mydb=# \l
                                      List of databases
   Name    |  Owner   | Encoding | Collate | Ctype |   Access privileges
-----------+----------+----------+---------+-------+-----------------------
 mydb      | postgres | UTF8     | C.UTF-8 | C.UTF-8|
 postgres  | postgres | UTF8     | C.UTF-8 | C.UTF-8|
 template0 | postgres | UTF8     | C.UTF-8 | C.UTF-8| =c/postgres          +
           |          |          |         |       | postgres=CTc/postgres
 template1 | postgres | UTF8     | C.UTF-8 | C.UTF-8| =c/postgres          +
           |          |          |         |       | postgres=CTc/postgres
(4 rows)
```

### List schemas

```
mydb=# \dn
  List of schemas
  Name  |  Owner
--------+----------
 public | postgres
(1 row)
```

### List tables in current schema

```
mydb=# \dt
            List of relations
 Schema |  Name   | Type  |  Owner
--------+---------+-------+----------
 public | orders  | table | postgres
 public | users   | table | postgres
(2 rows)
```

### Describe a table

```
mydb=# \d users
                                     Table "public.users"
    Column    |           Type           | Collation | Nullable |              Default
--------------+--------------------------+-----------+----------+-----------------------------------
 id           | bigint                   |           | not null | nextval('users_id_seq'::regclass)
 email        | text                     |           | not null |
 created_at   | timestamp with time zone |           | not null | now()
 is_active    | boolean                  |           | not null | true
Indexes:
    "users_pkey" PRIMARY KEY, btree (id)
    "users_email_key" UNIQUE CONSTRAINT, btree (email)
```

### List functions

```
mydb=# \df
                          List of functions
 Schema |    Name    | Result data type | Argument data types | Type
--------+------------+------------------+---------------------+------
 public | get_userid | integer          | text                | func
(1 row)
```

### List users / roles

```
mydb=# \du
                                      List of roles
  Role name  |                         Attributes                         | Member of
-------------+------------------------------------------------------------+-----------
 alice       |                                                            | {}
 postgres    | Superuser, Create role, Create DB, Replication, Bypass RLS | {}
```

### Toggle query timing

```
mydb=# \timing
Timing is on.
mydb=# SELECT count(*) FROM users;
 count
-------
   100
(1 row)

Time: 1.234 ms
```

### Toggle expanded output

```
mydb=# \x
Expanded display is on.
mydb=# SELECT * FROM users LIMIT 1;
-[ RECORD 1 ]------+----------------------------
id                 | 1
email              | alice@example.com
created_at         | 2024-01-15 09:30:00+00
is_active          | t
```

### psql help

```
mydb=# \?
General
  \copyright             show PostgreSQL usage and distribution terms
  \crosstabview [...]    execute query and display result in crosstab
  \errverbose            show most recent error message at maximum verbosity
  \g [(OPTIONS)] [FILE]  execute query (and send result to file or |pipe);
  \gdesc                 describe result of query, without executing it
  ...
```

### Plan a query

```
mydb=# EXPLAIN SELECT * FROM users WHERE email='foo@bar.com';
                              QUERY PLAN
-----------------------------------------------------------------------
 Index Scan using users_email_key on users  (cost=0.28..8.29 rows=1 width=64)
   Index Cond: (email = 'foo@bar.com'::text)
(2 rows)
```

### Plan and run

```
mydb=# EXPLAIN ANALYZE SELECT * FROM users WHERE email='foo@bar.com';
                                              QUERY PLAN
-----------------------------------------------------------------------------------------------------
 Index Scan using users_email_key on users  (cost=0.28..8.29 rows=1 width=64)
                                            (actual time=0.045..0.046 rows=0 loops=1)
   Index Cond: (email = 'foo@bar.com'::text)
 Planning Time: 0.156 ms
 Execution Time: 0.082 ms
(4 rows)
```

### Database size

```
mydb=# SELECT pg_size_pretty(pg_database_size('mydb'));
 pg_size_pretty
----------------
 142 MB
(1 row)
```

### Single table size

```
mydb=# SELECT pg_size_pretty(pg_relation_size('users'));
 pg_size_pretty
----------------
 1248 kB
(1 row)
```

### Active sessions

```
mydb=# SELECT pid, usename, state, query
mydb-#   FROM pg_stat_activity
mydb-#   WHERE state != 'idle';
  pid  | usename  |        state        |              query
-------+----------+---------------------+----------------------------------
  1234 | postgres | active              | SELECT * FROM pg_stat_activity;
  1567 | app      | idle in transaction | UPDATE accounts SET ...
(2 rows)
```

### Locks not granted

```
mydb=# SELECT * FROM pg_locks WHERE NOT granted;
   locktype    | database | relation | ...
---------------+----------+----------+-----
 transactionid |          |          | ...
(1 row)
```

### Top queries by total time

```
mydb=# SELECT query, calls, total_exec_time, mean_exec_time
mydb-#   FROM pg_stat_statements
mydb-#   ORDER BY total_exec_time DESC
mydb-#   LIMIT 5;
                  query                   | calls | total_exec_time | mean_exec_time
------------------------------------------+-------+-----------------+----------------
 SELECT * FROM users WHERE email = $1     | 1024  |        12345.67 |          12.05
 INSERT INTO orders VALUES ($1, $2, $3)   |  512  |         8765.43 |          17.12
 ...
```

### Vacuum analyze a table

```
mydb=# VACUUM (VERBOSE, ANALYZE) users;
INFO:  vacuuming "public.users"
INFO:  index "users_pkey" now contains 100 row versions in 1 pages
INFO:  "users": found 0 removable, 100 nonremovable row versions in 1 out of 1 pages
INFO:  analyzing "public.users"
VACUUM
```

### Reindex a table

```
mydb=# REINDEX TABLE users;
REINDEX
```

### Update statistics manually

```
mydb=# ANALYZE users;
ANALYZE
```

### Dump database to a SQL file

```
$ pg_dump mydb > mydb.sql
```

The output is plain SQL: `CREATE TABLE`, `COPY`, etc. Restore with `psql -d mydb -f mydb.sql`.

### Custom-format dump (compressed, parallel-restore-capable)

```
$ pg_dump -Fc mydb > mydb.dump
```

The `-Fc` flag means "custom format." This is the format you should usually use because `pg_restore` can pick out individual tables, run in parallel, and reorder operations.

### Restore from custom dump

```
$ pg_restore -d mydb mydb.dump
```

### Restore from plain SQL

```
$ psql -d mydb -f mydb.sql
```

### Physical base backup (binary)

```
$ pg_basebackup -h primary -D /var/lib/pg/backup -X stream -P
26456/26456 kB (100%), 1/1 tablespace
```

The `-X stream` includes WAL streamed during the backup, so the backup is self-contained. `-P` shows progress.

### Read pg_hba.conf

```
$ cat /var/lib/postgresql/15/main/pg_hba.conf
# TYPE  DATABASE  USER  ADDRESS         METHOD
local   all       all                   peer
host    all       all   127.0.0.1/32    scram-sha-256
host    all       all   ::1/128         scram-sha-256
```

### Inspect important conf values

```
$ cat /etc/postgresql/15/main/postgresql.conf | grep -E '^(shared_buffers|work_mem|max_connections)'
shared_buffers = 2GB
work_mem = 16MB
max_connections = 200
```

### Show a config from psql

```
mydb=# SHOW shared_buffers;
 shared_buffers
----------------
 2GB
(1 row)
```

### Liveness check

```
$ pg_isready -h localhost -p 5432
localhost:5432 - accepting connections
```

### Benchmark

```
$ pgbench -i mydb        # initialize
$ pgbench -c 10 -j 2 -T 60 mydb
transaction type: <builtin: TPC-B (sort of)>
scaling factor: 1
query mode: simple
number of clients: 10
number of threads: 2
duration: 60 s
number of transactions actually processed: 12345
latency average = 4.86 ms
tps = 205.45 (including connections establishing)
tps = 205.50 (excluding connections establishing)
```

### Install pg_stat_statements

```
mydb=# CREATE EXTENSION pg_stat_statements;
CREATE EXTENSION
```

(Also requires `shared_preload_libraries = 'pg_stat_statements'` in `postgresql.conf` and a restart.)

### Check version

```
mydb=# SELECT version();
                                                 version
---------------------------------------------------------------------------------------------------------
 PostgreSQL 16.2 (Ubuntu 16.2-1.pgdg22.04+1) on x86_64-pc-linux-gnu, compiled by gcc 11.4.0, 64-bit
(1 row)
```

### See server logs

```
$ journalctl -u postgresql -n 50
Apr 27 09:30:00 mybox postgres[1234]: LOG:  database system is ready to accept connections
Apr 27 09:35:12 mybox postgres[1567]: LOG:  checkpoint starting: time
Apr 27 09:40:00 mybox postgres[1567]: LOG:  checkpoint complete: wrote 412 buffers (0.5%)
```

### Find unused indexes

```
mydb=# SELECT schemaname, relname, indexrelname, idx_scan
mydb-#   FROM pg_stat_user_indexes
mydb-#   WHERE idx_scan = 0;
 schemaname | relname | indexrelname | idx_scan
------------+---------+--------------+----------
 public     | logs    | logs_old_idx |        0
(1 row)
```

(Indexes with zero scans are candidates for dropping.)

### Find table bloat

```
mydb=# SELECT relname, n_dead_tup, n_live_tup,
mydb-#        round(n_dead_tup::numeric / NULLIF(n_live_tup, 0), 2) AS dead_ratio
mydb-#   FROM pg_stat_user_tables
mydb-#   ORDER BY n_dead_tup DESC LIMIT 5;
 relname  | n_dead_tup | n_live_tup | dead_ratio
----------+------------+------------+------------
 sessions |    234567  |    100000  |       2.35
 logs     |     45678  |  10000000  |       0.00
```

### Kill a running query

```
mydb=# SELECT pg_cancel_backend(1234);   -- gentle (cancels query)
mydb=# SELECT pg_terminate_backend(1234); -- harsher (kills connection)
```

## Common Confusions

### "I made an index but my query is still slow"

**Broken:** `CREATE INDEX ON users (email); SELECT * FROM users WHERE lower(email) = 'alice@...';`

**Fixed:** Either `CREATE INDEX ON users (lower(email));` (an expression index), or query without `lower()` if your data is consistently cased.

The planner can only use the index if the indexed expression matches what's in the WHERE clause exactly. `email` is not the same as `lower(email)`.

### "EXPLAIN says it's fast, but it's actually slow"

**Broken:** Trusting `EXPLAIN`'s cost estimates as runtime.

**Fixed:** Use `EXPLAIN ANALYZE` to get actual times. The cost is the planner's guess; ANALYZE shows reality. If estimates and reality diverge wildly, your stats are stale — run `ANALYZE`.

### "My UPDATE doesn't reduce the table size"

**Broken:** `UPDATE big_table SET col = 'x';` and expecting the disk file to shrink. It doesn't; MVCC keeps the old row versions, the new versions are appended.

**Fixed:** Run `VACUUM` to reclaim space within the file. To shrink the file on disk, run `VACUUM FULL` (warning: takes an exclusive lock).

### "I dropped a table but disk usage didn't go down"

**Broken:** Looking at `df` immediately after `DROP TABLE`.

**Fixed:** PostgreSQL files are unlinked but the WAL still has references; until the next checkpoint, space is held. Wait or `CHECKPOINT;`.

### "I added a column and now it's super slow on big tables"

**Broken:** `ALTER TABLE big_table ADD COLUMN x text NOT NULL DEFAULT 'foo';` rewrites every row to fill in the default. Locks the table for the duration.

**Fixed:** PostgreSQL 11+ supports fast `ADD COLUMN ... DEFAULT ...` for non-volatile defaults — the default is stored once and not materialized into every row. For older versions or volatile defaults, do it in two steps: `ADD COLUMN x text` (fast), then `UPDATE` in batches.

### "I dropped an index and queries got slower instead of faster"

**Broken:** Assuming any unused-looking index can be dropped.

**Fixed:** Check `pg_stat_user_indexes.idx_scan` for actual usage. Also, an index might be there for a constraint (UNIQUE, PRIMARY KEY) and dropping it breaks the constraint.

### "DELETE removes the row but the size grows"

**Broken:** `DELETE FROM big_table WHERE ...; -- table file got bigger?!`

**Fixed:** DELETE marks rows dead but writes WAL records and possibly inserts deletion-marker tuples. Run `VACUUM` to reclaim. The disk file doesn't shrink; only space within it becomes reusable.

### "ORDER BY indexed_column is slow"

**Broken:** Querying `SELECT * FROM big_table ORDER BY created_at DESC LIMIT 10;` and not using the index.

**Fixed:** Make sure you have an index on `created_at` (and that the index direction matches your query, or use `ASC` / `DESC` in the index for asymmetric performance). Also check that you're not preventing index usage with a function or transformation in the ORDER BY clause.

### "Why does my count(*) take forever?"

**Broken:** `SELECT count(*) FROM big_table;` on a billion-row table.

**Fixed:** PostgreSQL has to scan every row to count, because of MVCC (each transaction sees a different number). For a fast estimate use `SELECT reltuples FROM pg_class WHERE relname = 'big_table';` which is the planner's last-known estimate. For exact, you can keep a counter table updated by trigger.

### "CTE is slower than I expected"

**Broken:** Assuming `WITH foo AS (SELECT ...)` is just inlining.

**Fixed:** Before PostgreSQL 12, CTEs were always materialized (an "optimization fence"). PG 12+ inlines them when possible, but you can force materialization with `AS MATERIALIZED` or prevent it with `AS NOT MATERIALIZED`.

### "UPDATE with subquery is slow"

**Broken:** `UPDATE a SET x = (SELECT y FROM b WHERE b.id = a.id);` running once per row.

**Fixed:** `UPDATE a SET x = b.y FROM b WHERE b.id = a.id;` (the JOIN form) is much faster — PostgreSQL does a single join instead of a subquery per row.

### "ON CONFLICT DO UPDATE doesn't fire"

**Broken:** `INSERT ... ON CONFLICT DO UPDATE SET ...` and expecting it to work without specifying the conflict target.

**Fixed:** You must specify which constraint or column you're conflicting on: `ON CONFLICT (email) DO UPDATE ...`.

### "TRUNCATE is fast but ROLLBACK is slow"

**Broken:** `BEGIN; TRUNCATE big_table; ROLLBACK;` — surprisingly slow.

**Fixed:** TRUNCATE allocates new file IDs; rollback reverts to the old files. Both operations are mostly O(1) for the data but log a lot. Most of the time this is fine; just don't expect TRUNCATE to be free in long transactions.

### "psql script aborts mid-way"

**Broken:** Running a long script in psql; one statement fails; the script keeps going.

**Fixed:** Use `\set ON_ERROR_STOP on` at the top of your script, or pass `--set=ON_ERROR_STOP=1` to psql. Now an error halts the script.

### "Can't drop a database — it's in use"

**Broken:** `DROP DATABASE mydb;` when something is connected.

**Fixed:** PostgreSQL 13+ supports `DROP DATABASE mydb WITH (FORCE);` which kicks off all connections. Otherwise, manually kill connections with `pg_terminate_backend()` first.

## Vocabulary

| Word | Plain English |
|------|----------------|
| PostgreSQL | The full name of the database. |
| Postgres | Friendly nickname for PostgreSQL; same thing. |
| psql | The command-line client; talks to a Postgres server. |
| pgAdmin | A graphical (web/desktop) tool for managing Postgres. |
| cluster | A single Postgres server installation (one `data` directory, one port). Confusingly: not a sharded cluster; just one server's worth. |
| database | A named container inside a cluster. A cluster has many databases. |
| schema | A namespace inside a database. Like a folder for tables. Default is `public`. |
| table | A shelf. Holds rows of one shape. |
| row | One box on a shelf. |
| tuple | Same as row, but the word databases use internally. |
| column | A label on a box; one named field per row. |
| attribute | Same as column. |
| type | The shape behind a column label (integer, text, etc.). |
| domain | A type with extra constraints (`CREATE DOMAIN`). |
| sequence | An auto-incrementing counter, used for IDs. |
| view | A saved SELECT, queryable like a table. |
| materialized view | A view whose result is stored on disk; refresh manually. |
| function | Server-side code (SQL or PL/pgSQL or others) that returns a value. |
| procedure | Server-side code that doesn't return a value (since PG 11). |
| trigger | Code that runs automatically before/after INSERT/UPDATE/DELETE. |
| rule | An older mechanism for query rewriting; mostly avoid. |
| index | Cross-reference cards; speeds up lookups. |
| B-tree | The default index; balanced tree, good for sortable data. |
| hash | An index for equality only (no ranges). |
| GIN | Inverted index for arrays, jsonb, full text. |
| GiST | Generalized search tree for geometric/range/full-text data. |
| BRIN | Block range index; tiny, for huge ordered tables. |
| SP-GiST | Space-partitioned GiST; for unbalanced trees like quadtrees. |
| partial index | Index with a `WHERE` clause; only some rows. |
| expression index | Index on a function of a column (`lower(email)`). |
| covering index | Index that includes extra columns to allow Index Only Scans. |
| INCLUDE | Keyword that adds non-key columns to a covering index. |
| primary key | The unique label per row. Often `id`. |
| foreign key | A column that points at another table's primary key. |
| unique constraint | No two rows may share this value. |
| not-null constraint | This column must always have a value. |
| check constraint | A custom rule (e.g., `price > 0`). |
| exclusion constraint | A generalized uniqueness rule using GiST (e.g., no overlapping ranges). |
| transaction | A bundle of changes that commit or rollback together. |
| BEGIN | Start a transaction. |
| COMMIT | Make the transaction's changes permanent and visible. |
| ROLLBACK | Undo the transaction; nothing happened. |
| SAVEPOINT | A marker inside a transaction; can roll back to it. |
| RELEASE | Drop a savepoint without rolling back to it. |
| isolation level | How much each transaction sees of other concurrent ones. |
| READ COMMITTED | Default; each statement sees a fresh snapshot. |
| REPEATABLE READ | Whole transaction sees one snapshot. |
| SERIALIZABLE | Strictest; transactions appear to run one-at-a-time. |
| MVCC | Multi-version concurrency control; how Postgres keeps readers and writers from blocking each other. |
| xmin | Hidden column; transaction ID that created this row version. |
| xmax | Hidden column; transaction ID that deleted this row version (0 if alive). |
| ctid | Hidden column; physical position (block, offset) of a row. |
| dead tuple | A row version no transaction can see anymore; cleanup needed. |
| bloat | Excess space taken by dead tuples and unused space inside files. |
| VACUUM | The cleanup command; reclaims space from dead tuples. |
| autovacuum | Background process that runs VACUUM automatically. |
| ANALYZE | Updates planner statistics; runs as part of autovacuum too. |
| REINDEX | Rebuilds an index; useful after corruption or for shrinking. |
| EXPLAIN | Show the planner's intended plan for a query. |
| EXPLAIN ANALYZE | Run the query and show the actual plan with timings. |
| plan | The set of operations the database will use to answer a query. |
| Seq Scan | Read every row in a table. |
| Index Scan | Walk an index, then fetch matching rows from the heap. |
| Index Only Scan | Walk an index without touching the heap; possible if index covers and visibility map says rows are fresh. |
| Bitmap Heap Scan | Build a bitmap of matching rows from one or more indexes, then fetch in physical order. |
| Hash Join | Build a hash of one side, probe with the other. |
| Merge Join | Sort both sides, walk in lockstep. |
| Nested Loop | For each row in A, look up in B. |
| Sort | Sort rows in memory or on disk. |
| Materialize | Cache an inner subplan in memory. |
| CTE | "Common Table Expression," `WITH foo AS (...)`. |
| recursive CTE | A CTE that references itself; for trees and graphs. |
| window function | A function over a window of rows (`ROW_NUMBER()`, `RANK()`, `SUM() OVER (...)`). |
| OVER | Keyword introducing a window. |
| PARTITION BY | Split window into groups. |
| ROWS BETWEEN | Define the row frame within a window (e.g., `ROWS BETWEEN 1 PRECEDING AND CURRENT ROW`). |
| jsonb | Binary JSON; the right way to store JSON in Postgres. |
| json | Text JSON; usually use `jsonb` instead. |
| array | A column type that holds an ordered list (`text[]`, `integer[]`). |
| hstore | Old key-value type; mostly replaced by `jsonb`. |
| uuid | 128-bit globally unique identifier. |
| bytea | Raw bytes column. |
| citext | Case-insensitive text type (extension). |
| ltree | Hierarchical tree-path type (extension). |
| range type | A range of values (`int4range`, `tstzrange`). |
| enum | A user-defined type with a fixed list of values. |
| ROW | Construct a row literal (`ROW(1, 'foo')`). |
| RECORD | Untyped row variable in PL/pgSQL. |
| REFCURSOR | A reference to a cursor that can be passed around. |
| COMPOSITE TYPE | A user-defined type made of fields, like a row. |
| WAL | Write-ahead log; every change recorded before it hits the data files. |
| write-ahead log | Same as WAL. |
| checkpoint | Periodic flush of dirty pages from memory to data files. |
| fsync | A system call that forces buffered data to physical disk. |
| full_page_writes | Setting that forces full page in WAL after first change post-checkpoint, for crash safety. |
| wal_level | How much info the WAL contains: `minimal`, `replica`, `logical`. |
| replica | A standby that replicates WAL from a primary. |
| primary | The writable Postgres server. |
| standby | A read-only server replaying WAL. |
| hot standby | A standby that serves read queries while replaying. |
| streaming replication | Standbys connect and receive WAL as it's produced. |
| logical replication | Decoded row-level changes shipped to subscribers. |
| publication | A set of tables published for logical replication. |
| subscription | A receiver of a publication. |
| replication slot | A reserved spot on the primary that ensures WAL is kept until the subscriber consumes it. |
| recovery_target_time | "Roll the WAL forward up to this time" for point-in-time recovery. |
| PITR | Point-in-time recovery; restore base backup + WAL up to a chosen moment. |
| pg_basebackup | Tool to take a binary base backup of the cluster. |
| pg_dump | Tool to export a database to SQL or custom format. |
| pg_restore | Tool to restore from `pg_dump -Fc` custom-format dumps. |
| pg_upgrade | Tool to upgrade between major versions in place. |
| pg_stat_statements | Extension that tracks query statistics. |
| pg_stat_activity | View showing current sessions and their state. |
| pg_stat_user_tables | View showing per-table stats (rows scanned, dead tuples, etc.). |
| pg_locks | View showing all current locks. |
| deadlock | Two transactions each waiting on the other; one is aborted. |
| lock timeout | Max time a statement waits for a lock before erroring. |
| statement_timeout | Max time a statement runs before being canceled. |
| idle_in_transaction_session_timeout | Max time a connection is idle inside an open transaction. |
| max_connections | Cap on simultaneous client connections. |
| shared_buffers | Memory Postgres uses to cache pages; usually 25% of RAM. |
| work_mem | Memory each sort/hash node can use before spilling to disk. |
| maintenance_work_mem | Memory for VACUUM, CREATE INDEX, ALTER TABLE; bigger = faster. |
| effective_cache_size | Planner's estimate of OS file cache available; affects index plan choices. |
| random_page_cost | Planner's cost for a random page read. |
| seq_page_cost | Planner's cost for a sequential page read. |
| jit | Just-in-time compilation of expressions for speed (PG 11+). |
| parallel query | Splitting a query across multiple worker processes. |
| parallel_workers | Number of parallel workers a query may use. |
| autovacuum_vacuum_threshold | Min dead tuples before autovacuum kicks in. |
| autovacuum_naptime | How often autovacuum checks tables. |
| pg_hba.conf | Host-based authentication config; who can connect from where with what method. |
| pg_ident.conf | Maps OS users to Postgres roles. |
| postgresql.conf | Main server configuration file. |
| pgbouncer | Lightweight connection pooler. |
| pgpool | Heavier pooler with load balancing. |
| patroni | HA orchestration on top of Postgres + etcd/Consul. |
| repmgr | Older replication manager. |
| pg_auto_failover | Citus's small-footprint HA tool. |
| postgis | Geographic extension (points, polygons, distance). |
| pgvector | Vector similarity search extension. |
| timescaledb | Time-series extension with hypertables. |
| citus | Distributed Postgres extension; sharding for OLTP. |
| postgrest | Tool that turns a Postgres schema into a REST API. |

## Try This

These five experiments give you the muscle memory.

### Experiment 1: Watch a transaction roll back

In one psql session:

```sql
mydb=# CREATE TABLE notes (id serial PRIMARY KEY, msg text);
mydb=# BEGIN;
mydb=# INSERT INTO notes (msg) VALUES ('hello');
mydb=# SELECT * FROM notes;
 id |  msg
----+-------
  1 | hello
mydb=# ROLLBACK;
mydb=# SELECT * FROM notes;
 id | msg
----+-----
(0 rows)
```

Notice how the row was visible inside the transaction, then disappeared after rollback. That is atomicity in action.

### Experiment 2: Two sessions, see MVCC

Open two psql sessions. In session A:

```sql
mydb=# BEGIN;
mydb=# INSERT INTO notes (msg) VALUES ('from A');
mydb=# SELECT * FROM notes;
 id |   msg
----+----------
  1 | from A
```

In session B (without committing in A):

```sql
mydb=# SELECT * FROM notes;
 id | msg
----+-----
(0 rows)
```

Session B does not see A's uncommitted insert. Now in session A:

```sql
mydb=# COMMIT;
```

Now session B sees it (run the SELECT again). Two snapshots, isolated until commit.

### Experiment 3: Watch an index help

```sql
mydb=# CREATE TABLE bigtable AS
mydb-#   SELECT i AS id, md5(random()::text) AS data
mydb-#   FROM generate_series(1, 1000000) AS i;
SELECT 1000000

mydb=# EXPLAIN ANALYZE SELECT * FROM bigtable WHERE id = 500000;
                                                    QUERY PLAN
------------------------------------------------------------------------------------------------------------
 Seq Scan on bigtable  (cost=0.00..18334.00 rows=1 width=37) (actual time=42.110..82.340 rows=1 loops=1)
   Filter: (id = 500000)
   Rows Removed by Filter: 999999
 Planning Time: 0.123 ms
 Execution Time: 82.371 ms

mydb=# CREATE INDEX ON bigtable (id);
CREATE INDEX

mydb=# EXPLAIN ANALYZE SELECT * FROM bigtable WHERE id = 500000;
                                                       QUERY PLAN
---------------------------------------------------------------------------------------------------------------
 Index Scan using bigtable_id_idx on bigtable  (cost=0.42..8.44 rows=1 width=37) (actual time=0.045..0.046 rows=1)
 Planning Time: 0.245 ms
 Execution Time: 0.073 ms
```

From 82 milliseconds to 0.07 milliseconds. A thousand-fold speedup. That is what an index does.

### Experiment 4: See dead tuples accumulate

```sql
mydb=# CREATE TABLE churn (id int, val text);
mydb=# INSERT INTO churn SELECT i, 'x' FROM generate_series(1, 100000) i;
mydb=# UPDATE churn SET val = 'y';
mydb=# SELECT n_live_tup, n_dead_tup FROM pg_stat_user_tables WHERE relname = 'churn';
 n_live_tup | n_dead_tup
------------+------------
     100000 |     100000
mydb=# VACUUM churn;
mydb=# SELECT n_live_tup, n_dead_tup FROM pg_stat_user_tables WHERE relname = 'churn';
 n_live_tup | n_dead_tup
------------+------------
     100000 |          0
```

Before vacuum, 100k dead tuples. After, zero. The space is now reusable.

### Experiment 5: See a deadlock

Two psql sessions, two tables `a` and `b` with one row each.

Session 1:

```sql
mydb=# BEGIN;
mydb=# UPDATE a SET v = v + 1 WHERE id = 1;
```

Session 2:

```sql
mydb=# BEGIN;
mydb=# UPDATE b SET v = v + 1 WHERE id = 1;
mydb=# UPDATE a SET v = v + 1 WHERE id = 1;
-- now blocked, waiting for session 1
```

Session 1:

```sql
mydb=# UPDATE b SET v = v + 1 WHERE id = 1;
ERROR:  deadlock detected
DETAIL:  Process 1234 waits for ShareLock on transaction 5678; blocked by process 5678.
        Process 5678 waits for ShareLock on transaction 1234; blocked by process 1234.
```

Postgres detects the cycle and aborts one of you. Always lock rows in a consistent order to avoid this.

### Experiment 6: pg_dump + pg_restore round trip

```
$ pg_dump -Fc mydb > mydb.dump
$ createdb mydb_copy
$ pg_restore -d mydb_copy mydb.dump
$ psql mydb_copy -c '\dt'
```

You now have a copy of the database. This is your backup workflow.

### Experiment 7: Watch checkpoints in the log

In `postgresql.conf`, set `log_checkpoints = on` and reload (`SELECT pg_reload_conf();`). Then run a write-heavy workload and tail the log:

```
$ tail -f /var/log/postgresql/postgresql-*.log
LOG:  checkpoint starting: time
LOG:  checkpoint complete: wrote 412 buffers (0.5%); 0 WAL file(s) added, 0 removed, 1 recycled; write=2.012 s, sync=0.013 s, total=2.040 s; sync files=42, longest=0.005 s, average=0.001 s; distance=8192 kB, estimate=8192 kB
```

You can see the checkpoint timing, how many buffers it wrote, sync times. Useful for tuning.

### Experiment 8: Make a partial index

```sql
mydb=# CREATE INDEX users_active_idx ON users (email) WHERE is_active = true;
mydb=# EXPLAIN SELECT * FROM users WHERE email = 'alice@...' AND is_active = true;
```

You should see the partial index used. Now query without `is_active = true`:

```sql
mydb=# EXPLAIN SELECT * FROM users WHERE email = 'alice@...';
```

The partial index is not used (it doesn't cover inactive users). You'd need a different index for that. Partial indexes shine when you only ever query the subset.

## Where to Go Next

Once this sheet feels easy, the dense engineer-grade material is one command away. Stay in the terminal:

- **`cs databases postgresql`** — dense reference: every command, every flag, every catalog table.
- **`cs detail databases/postgresql`** — query planner cost model derivations, B-tree split math, MVCC visibility logic, WAL record format.
- **`cs databases mysql`**, **`cs databases redis`**, **`cs databases sqlite`** — the alternatives. Knowing two databases makes you a much better Postgres user.
- **`cs ramp-up linux-kernel-eli5`** — what `fsync()` actually does (which is what makes WAL durability real).

## See Also

- `databases/postgresql` — engineer-grade reference for Postgres.
- `databases/mysql` — the other big open-source RDBMS.
- `databases/sqlite` — embedded relational database; great mental contrast.
- `databases/redis` — key-value store; complement, not competitor.
- `databases/sql` — the query language itself, in detail.
- `databases/time-series` — when Postgres is enough vs when to reach for InfluxDB / TimescaleDB.
- `data-engineering/airflow` — orchestration for ETL pipelines that read/write Postgres.
- `ramp-up/linux-kernel-eli5` — the kernel behind fsync, page cache, and process management.
- `ramp-up/tcp-eli5` — what your `psql -h` connection actually does at the socket level.

## References

- **postgresql.org/docs** — the manual is gold. Searchable, accurate, exhaustive.
- **"PostgreSQL: Up and Running"** by Regina Obe and Leo Hsu — short, practical, modern.
- **"The Internals of PostgreSQL"** at interdb.jp/pg/ — free book; deep into MVCC, WAL, and the executor.
- **"PostgreSQL High Performance"** by Gregory Smith — older but the tuning chapters still apply.
- **`man psql`**, **`man pg_dump`**, **`man pg_restore`**, **`man pg_basebackup`** — official man pages, all in your terminal.
- **`man postgresql.conf`** — explanations of every config knob (also at postgresql.org/docs/current/runtime-config.html).
- **pgconf videos** — every year, hours of high-quality Postgres talks; YouTube has them all.
- **planet.postgresql.org** — aggregated Postgres community blogs.
- **pgvector** docs (github.com/pgvector/pgvector) — vector search.
- **PostGIS** docs (postgis.net) — geographic data.
- **TimescaleDB** docs (docs.timescale.com) — time-series.
- **PgBouncer** docs (pgbouncer.org) — pooling.

— End of ELI5 —

When this sheet feels boring (and it will, faster than you think), graduate to `cs databases postgresql` — the engineer-grade reference. After that, `cs detail databases/postgresql` gives you the academic underpinning. By the time you've read both, you will be reading `EXPLAIN ANALYZE` output without flinching, tuning `work_mem` and `maintenance_work_mem` from feel, and knowing exactly which knob to turn when the database is slow.

### One last thing before you go

Pick one command from the Hands-On section that you have not run yet. Run it right now. Read the output. Try to figure out what each part means, using the Vocabulary table as your dictionary. Don't just trust this sheet — see for yourself. PostgreSQL is a real thing. It is a process running on your computer (or someone's computer) right now, watching for connections, running queries, vacuuming dead tuples in the background. The commands in this sheet let you peek at every part.

Reading is good. Doing is better. Type the commands. Watch Postgres respond.

You are now officially started on your PostgreSQL journey. Welcome to the warehouse.

The whole point of the North Star for the `cs` tool is: never leave the terminal to learn this stuff. Everything you need is here, or one `man` page away, or one psql `\?` away. There is no Google search you need to do to start understanding PostgreSQL. You can sit at your terminal, type, watch, read, and learn forever.

Have fun. Postgres is happy to be poked at. Nothing in this sheet will break anything. Try things. Type queries. Read what comes back. The more you do, the more it all clicks into place.

— End of ELI5 — (really this time!)
