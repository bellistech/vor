# SQL Advanced

Advanced SQL techniques for analytical queries, performance optimization, and complex data manipulation across PostgreSQL, MySQL, and standard SQL.

## Window Functions

```sql
-- ROW_NUMBER: unique sequential number per partition
SELECT
    department,
    name,
    salary,
    ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) AS rank_in_dept
FROM employees;

-- RANK and DENSE_RANK: handle ties differently
SELECT
    name,
    score,
    RANK() OVER (ORDER BY score DESC) AS rank,           -- gaps after ties
    DENSE_RANK() OVER (ORDER BY score DESC) AS dense_rank -- no gaps
FROM results;

-- LAG / LEAD: access previous/next rows
SELECT
    date,
    revenue,
    LAG(revenue, 1) OVER (ORDER BY date) AS prev_day,
    LEAD(revenue, 1) OVER (ORDER BY date) AS next_day,
    revenue - LAG(revenue, 1) OVER (ORDER BY date) AS daily_change
FROM daily_sales;

-- Running totals and moving averages
SELECT
    date,
    amount,
    SUM(amount) OVER (ORDER BY date ROWS UNBOUNDED PRECEDING) AS running_total,
    AVG(amount) OVER (ORDER BY date ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) AS moving_avg_7d
FROM transactions;

-- NTILE: distribute rows into N buckets
SELECT
    name,
    salary,
    NTILE(4) OVER (ORDER BY salary) AS salary_quartile
FROM employees;

-- FIRST_VALUE / LAST_VALUE / NTH_VALUE
SELECT
    department,
    name,
    salary,
    FIRST_VALUE(name) OVER (
        PARTITION BY department ORDER BY salary DESC
        ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING
    ) AS highest_paid
FROM employees;

-- Named window definitions
SELECT
    name,
    salary,
    RANK() OVER w AS salary_rank,
    SUM(salary) OVER w AS running_salary
FROM employees
WINDOW w AS (PARTITION BY department ORDER BY salary DESC);
```

## Common Table Expressions (CTEs)

```sql
-- Basic CTE
WITH active_users AS (
    SELECT user_id, name, email
    FROM users
    WHERE last_login > CURRENT_DATE - INTERVAL '30 days'
),
user_orders AS (
    SELECT u.user_id, u.name, COUNT(*) AS order_count, SUM(o.total) AS total_spent
    FROM active_users u
    JOIN orders o ON u.user_id = o.user_id
    GROUP BY u.user_id, u.name
)
SELECT * FROM user_orders WHERE total_spent > 1000 ORDER BY total_spent DESC;

-- Recursive CTE: organizational hierarchy
WITH RECURSIVE org_tree AS (
    -- Base case: CEO (no manager)
    SELECT id, name, manager_id, 1 AS depth, ARRAY[name] AS path
    FROM employees
    WHERE manager_id IS NULL

    UNION ALL

    -- Recursive step
    SELECT e.id, e.name, e.manager_id, t.depth + 1, t.path || e.name
    FROM employees e
    JOIN org_tree t ON e.manager_id = t.id
)
SELECT depth, repeat('  ', depth - 1) || name AS org_chart, path
FROM org_tree
ORDER BY path;

-- Recursive CTE: generate date series
WITH RECURSIVE dates AS (
    SELECT DATE '2024-01-01' AS dt
    UNION ALL
    SELECT dt + INTERVAL '1 day'
    FROM dates
    WHERE dt < DATE '2024-12-31'
)
SELECT dt FROM dates;

-- Recursive CTE: graph traversal (shortest path)
WITH RECURSIVE paths AS (
    SELECT target AS node, 1 AS hops, ARRAY[source, target] AS path
    FROM edges
    WHERE source = 'A'

    UNION ALL

    SELECT e.target, p.hops + 1, p.path || e.target
    FROM paths p
    JOIN edges e ON p.node = e.source
    WHERE e.target != ALL(p.path)  -- prevent cycles
      AND p.hops < 10             -- depth limit
)
SELECT DISTINCT ON (node) node, hops, path
FROM paths
ORDER BY node, hops;
```

## EXPLAIN / EXPLAIN ANALYZE

```sql
-- Query plan (estimated costs)
EXPLAIN SELECT * FROM orders WHERE user_id = 42;

-- Actual execution stats (runs the query)
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT o.*, u.name
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE o.created_at > '2024-01-01';

-- JSON format for programmatic analysis
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) SELECT ...;

-- Key metrics to watch:
-- Seq Scan vs Index Scan
-- actual rows vs estimated rows (bad estimates = stale stats)
-- Sort Method: quicksort vs external merge (memory spill)
-- Hash Join vs Nested Loop vs Merge Join
-- Buffers: shared hit (cache) vs read (disk)

-- Update statistics
ANALYZE orders;
ANALYZE users;
```

## Index Types (PostgreSQL)

```sql
-- B-tree (default): equality and range queries
CREATE INDEX idx_orders_date ON orders (created_at);
CREATE INDEX idx_orders_user_date ON orders (user_id, created_at DESC);

-- Partial index: index only relevant rows
CREATE INDEX idx_orders_pending ON orders (created_at)
    WHERE status = 'pending';

-- Covering index (INCLUDE): avoid heap lookups
CREATE INDEX idx_orders_cover ON orders (user_id)
    INCLUDE (total, status);

-- Hash index: equality only, faster than B-tree for =
CREATE INDEX idx_users_email_hash ON users USING hash (email);

-- GIN (Generalized Inverted Index): arrays, JSONB, full text
CREATE INDEX idx_tags ON articles USING gin (tags);
CREATE INDEX idx_doc_search ON articles USING gin (to_tsvector('english', body));
CREATE INDEX idx_jsonb ON events USING gin (metadata jsonb_path_ops);

-- GiST (Generalized Search Tree): geometry, range types, nearest-neighbor
CREATE INDEX idx_location ON stores USING gist (coordinates);
CREATE INDEX idx_time_range ON bookings USING gist (during);

-- BRIN (Block Range INdex): large naturally-ordered tables
CREATE INDEX idx_logs_time ON logs USING brin (created_at) WITH (pages_per_range = 32);

-- Expression index
CREATE INDEX idx_lower_email ON users (lower(email));
```

## Query Optimization Patterns

```sql
-- Avoid: SELECT * with unused columns
-- Prefer: select only needed columns
SELECT id, name, email FROM users WHERE active = true;

-- Avoid: correlated subquery (runs per row)
SELECT u.*, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS order_count
FROM users u;

-- Prefer: JOIN with aggregation
SELECT u.*, COALESCE(o.order_count, 0) AS order_count
FROM users u
LEFT JOIN (SELECT user_id, COUNT(*) AS order_count FROM orders GROUP BY user_id) o
    ON u.id = o.user_id;

-- Use EXISTS instead of IN for large subqueries
SELECT * FROM users u
WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id AND o.total > 1000);

-- Batch operations with VALUES
INSERT INTO tags (name)
SELECT unnest AS name FROM unnest(ARRAY['a', 'b', 'c'])
ON CONFLICT (name) DO NOTHING;

-- Avoid functions on indexed columns in WHERE
-- Bad: WHERE YEAR(created_at) = 2024
-- Good: WHERE created_at >= '2024-01-01' AND created_at < '2025-01-01'
```

## LATERAL Joins

```sql
-- LATERAL: reference earlier tables in subquery (like a for-each loop)
-- Top 3 orders per user
SELECT u.name, top_orders.*
FROM users u
CROSS JOIN LATERAL (
    SELECT id, total, created_at
    FROM orders
    WHERE user_id = u.id
    ORDER BY total DESC
    LIMIT 3
) AS top_orders;

-- LATERAL with set-returning functions
SELECT u.name, t.tag
FROM users u,
LATERAL unnest(u.tags) AS t(tag);

-- LATERAL for dependent aggregation
SELECT d.name AS department, stats.*
FROM departments d
CROSS JOIN LATERAL (
    SELECT
        COUNT(*) AS employee_count,
        AVG(salary) AS avg_salary,
        MAX(salary) AS max_salary
    FROM employees
    WHERE department_id = d.id
) AS stats;
```

## Materialized Views

```sql
-- Create materialized view
CREATE MATERIALIZED VIEW monthly_revenue AS
SELECT
    date_trunc('month', created_at) AS month,
    product_id,
    SUM(quantity) AS total_units,
    SUM(total) AS total_revenue
FROM orders
WHERE status = 'completed'
GROUP BY 1, 2;

-- Add index on materialized view
CREATE UNIQUE INDEX idx_monthly_rev ON monthly_revenue (month, product_id);

-- Refresh (full)
REFRESH MATERIALIZED VIEW monthly_revenue;

-- Refresh concurrently (no lock, requires unique index)
REFRESH MATERIALIZED VIEW CONCURRENTLY monthly_revenue;
```

## Table Partitioning

```sql
-- Declarative partitioning (PostgreSQL 10+)
CREATE TABLE events (
    id          bigint GENERATED ALWAYS AS IDENTITY,
    event_type  text NOT NULL,
    payload     jsonb,
    created_at  timestamptz NOT NULL
) PARTITION BY RANGE (created_at);

-- Create partitions
CREATE TABLE events_2024_q1 PARTITION OF events
    FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');
CREATE TABLE events_2024_q2 PARTITION OF events
    FOR VALUES FROM ('2024-04-01') TO ('2024-07-01');

-- Default partition (catches unmatched rows)
CREATE TABLE events_default PARTITION OF events DEFAULT;

-- List partitioning
CREATE TABLE orders (
    id bigint, region text, total numeric
) PARTITION BY LIST (region);
CREATE TABLE orders_us PARTITION OF orders FOR VALUES IN ('us-east', 'us-west');
CREATE TABLE orders_eu PARTITION OF orders FOR VALUES IN ('eu-west', 'eu-central');

-- Hash partitioning (even distribution)
CREATE TABLE sessions (
    id uuid, user_id bigint, data jsonb
) PARTITION BY HASH (user_id);
CREATE TABLE sessions_0 PARTITION OF sessions FOR VALUES WITH (MODULUS 4, REMAINDER 0);
CREATE TABLE sessions_1 PARTITION OF sessions FOR VALUES WITH (MODULUS 4, REMAINDER 1);
```

## Tips

- Always run `EXPLAIN (ANALYZE, BUFFERS)` before and after optimization to verify improvements with real data
- Use partial indexes for queries that filter on a constant value (e.g., `WHERE status = 'active'`) to dramatically reduce index size
- Prefer `EXISTS` over `IN` for subqueries against large tables; the optimizer can short-circuit on first match
- Use BRIN indexes on append-only tables with naturally ordered columns (timestamps, serial IDs) for minimal storage overhead
- Name your CTEs descriptively and keep them focused; the optimizer may materialize them, preventing predicate pushdown
- Use `LATERAL` joins to replace correlated subqueries with clearer, often faster alternatives
- Refresh materialized views concurrently in production to avoid blocking reads during refresh
- Partition large tables (100M+ rows) by time range and ensure queries include the partition key in WHERE clauses
- Monitor `pg_stat_user_tables` for sequential scans on large tables; these often indicate missing indexes
- Use `pg_stat_statements` to identify the most expensive queries by total time, not just per-call latency
- Always include `ORDER BY` when using `ROW_NUMBER()`, `RANK()`, or `LEAD/LAG` -- window functions without explicit ordering produce nondeterministic results

## See Also

- PostgreSQL administration
- Database design and normalization
- Redis caching patterns
- TimescaleDB (time-series extension)
- Database replication and sharding

## References

- [PostgreSQL Window Functions](https://www.postgresql.org/docs/current/tutorial-window.html)
- [PostgreSQL CTE Documentation](https://www.postgresql.org/docs/current/queries-with.html)
- [Use The Index, Luke](https://use-the-index-luke.com/)
- [PostgreSQL EXPLAIN Documentation](https://www.postgresql.org/docs/current/sql-explain.html)
- [PostgreSQL Index Types](https://www.postgresql.org/docs/current/indexes-types.html)
- [PostgreSQL Table Partitioning](https://www.postgresql.org/docs/current/ddl-partitioning.html)
