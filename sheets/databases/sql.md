# SQL (Structured Query Language)

A declarative language for managing and querying relational databases.

## Querying Data

### SELECT Basics

```sql
-- Select specific columns
SELECT first_name, last_name FROM employees;

-- Select all columns
SELECT * FROM employees;

-- Distinct values (remove duplicates)
SELECT DISTINCT department FROM employees;

-- Column aliases
SELECT first_name AS name, salary * 12 AS annual_salary FROM employees;

-- Table aliases
SELECT e.first_name, d.name FROM employees e JOIN departments d ON e.dept_id = d.id;
```

### Filtering with WHERE

```sql
-- Comparison operators
SELECT * FROM employees WHERE salary > 50000;
SELECT * FROM employees WHERE department != 'Sales';

-- Logical operators
SELECT * FROM employees WHERE salary > 50000 AND department = 'Engineering';
SELECT * FROM employees WHERE department = 'Sales' OR department = 'Marketing';
SELECT * FROM employees WHERE NOT active;

-- Pattern matching
SELECT * FROM employees WHERE last_name LIKE 'Sm%';    -- starts with Sm
SELECT * FROM employees WHERE last_name LIKE '%son';    -- ends with son
SELECT * FROM employees WHERE first_name LIKE '_a%';    -- second char is a

-- Range and set membership
SELECT * FROM employees WHERE salary BETWEEN 40000 AND 80000;
SELECT * FROM employees WHERE department IN ('Sales', 'Marketing', 'HR');

-- NULL checks
SELECT * FROM employees WHERE manager_id IS NULL;
SELECT * FROM employees WHERE manager_id IS NOT NULL;
```

### JOIN Types

```sql
-- Inner join (only matching rows)
SELECT e.name, d.name FROM employees e INNER JOIN departments d ON e.dept_id = d.id;

-- Left join (all from left table, matching from right)
SELECT e.name, d.name FROM employees e LEFT JOIN departments d ON e.dept_id = d.id;

-- Right join (all from right table, matching from left)
SELECT e.name, d.name FROM employees e RIGHT JOIN departments d ON e.dept_id = d.id;

-- Full outer join (all rows from both tables)
SELECT e.name, d.name FROM employees e FULL OUTER JOIN departments d ON e.dept_id = d.id;

-- Cross join (cartesian product)
SELECT e.name, d.name FROM employees e CROSS JOIN departments d;

-- Self join
SELECT e.name AS employee, m.name AS manager
FROM employees e LEFT JOIN employees m ON e.manager_id = m.id;
```

### Grouping and Aggregation

```sql
-- Aggregate functions
SELECT COUNT(*) AS total, AVG(salary), MIN(salary), MAX(salary), SUM(salary)
FROM employees;

-- GROUP BY
SELECT department, COUNT(*) AS headcount, AVG(salary) AS avg_salary
FROM employees GROUP BY department;

-- HAVING (filter groups after aggregation)
SELECT department, AVG(salary) AS avg_salary
FROM employees GROUP BY department HAVING AVG(salary) > 60000;

-- ORDER BY
SELECT * FROM employees ORDER BY salary DESC, last_name ASC;

-- LIMIT and OFFSET
SELECT * FROM employees ORDER BY salary DESC LIMIT 10;         -- top 10
SELECT * FROM employees ORDER BY salary DESC LIMIT 10 OFFSET 20; -- page 3
```

### Subqueries

```sql
-- Subquery in WHERE
SELECT * FROM employees WHERE salary > (SELECT AVG(salary) FROM employees);

-- Subquery in FROM (derived table)
SELECT dept_avg.department, dept_avg.avg_sal
FROM (SELECT department, AVG(salary) AS avg_sal FROM employees GROUP BY department) dept_avg
WHERE dept_avg.avg_sal > 50000;

-- EXISTS
SELECT * FROM departments d
WHERE EXISTS (SELECT 1 FROM employees e WHERE e.dept_id = d.id);

-- IN with subquery
SELECT * FROM employees WHERE dept_id IN (SELECT id FROM departments WHERE region = 'West');
```

## Common Table Expressions (CTEs)

### WITH Clause

```sql
-- Basic CTE
WITH high_earners AS (
    SELECT * FROM employees WHERE salary > 80000
)
SELECT department, COUNT(*) FROM high_earners GROUP BY department;

-- Multiple CTEs
WITH dept_stats AS (
    SELECT department, AVG(salary) AS avg_salary FROM employees GROUP BY department
),
high_paying_depts AS (
    SELECT department FROM dept_stats WHERE avg_salary > 70000
)
SELECT e.* FROM employees e WHERE e.department IN (SELECT department FROM high_paying_depts);

-- Recursive CTE (org chart traversal)
WITH RECURSIVE org_tree AS (
    SELECT id, name, manager_id, 0 AS depth FROM employees WHERE manager_id IS NULL
    UNION ALL
    SELECT e.id, e.name, e.manager_id, ot.depth + 1
    FROM employees e JOIN org_tree ot ON e.manager_id = ot.id
)
SELECT * FROM org_tree ORDER BY depth;
```

## Set Operations

```sql
-- Combine results (no duplicates)
SELECT name FROM employees UNION SELECT name FROM contractors;

-- Combine results (keep duplicates)
SELECT name FROM employees UNION ALL SELECT name FROM contractors;

-- Rows in both queries
SELECT email FROM customers INTERSECT SELECT email FROM subscribers;

-- Rows in first but not second
SELECT email FROM customers EXCEPT SELECT email FROM unsubscribed;
```

## CASE Expressions

```sql
-- Simple CASE
SELECT name, CASE department
    WHEN 'Engineering' THEN 'Tech'
    WHEN 'Sales' THEN 'Revenue'
    ELSE 'Other'
END AS division FROM employees;

-- Searched CASE
SELECT name, salary,
    CASE
        WHEN salary > 100000 THEN 'Senior'
        WHEN salary > 60000  THEN 'Mid'
        ELSE 'Junior'
    END AS band
FROM employees;
```

## Window Functions

```sql
-- ROW_NUMBER (unique sequential number)
SELECT name, department, salary,
    ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) AS rank_in_dept
FROM employees;

-- RANK (gaps after ties) and DENSE_RANK (no gaps)
SELECT name, salary,
    RANK() OVER (ORDER BY salary DESC) AS rank,
    DENSE_RANK() OVER (ORDER BY salary DESC) AS dense_rank
FROM employees;

-- LAG and LEAD (access adjacent rows)
SELECT name, salary,
    LAG(salary, 1) OVER (ORDER BY hire_date) AS prev_salary,
    LEAD(salary, 1) OVER (ORDER BY hire_date) AS next_salary
FROM employees;

-- Running totals with SUM OVER
SELECT name, salary,
    SUM(salary) OVER (ORDER BY hire_date ROWS UNBOUNDED PRECEDING) AS running_total
FROM employees;
```

## Data Modification

### INSERT

```sql
-- Single row
INSERT INTO employees (first_name, last_name, salary) VALUES ('Jane', 'Doe', 75000);

-- Multiple rows
INSERT INTO employees (first_name, last_name, salary) VALUES
    ('Jane', 'Doe', 75000),
    ('John', 'Smith', 68000);

-- Insert from select
INSERT INTO archive_employees SELECT * FROM employees WHERE terminated = true;
```

### UPDATE

```sql
-- Update rows matching condition
UPDATE employees SET salary = salary * 1.10 WHERE department = 'Engineering';

-- Update multiple columns
UPDATE employees SET department = 'Product', title = 'PM' WHERE id = 42;
```

### DELETE

```sql
-- Delete matching rows
DELETE FROM employees WHERE terminated = true;

-- Delete all rows (use TRUNCATE for speed if no rollback needed)
TRUNCATE TABLE temp_imports;
```

## Schema Definition

### CREATE TABLE

```sql
CREATE TABLE employees (
    id          INT PRIMARY KEY AUTO_INCREMENT,     -- auto-incrementing PK
    first_name  VARCHAR(100) NOT NULL,              -- required string
    last_name   VARCHAR(100) NOT NULL,
    email       VARCHAR(255) UNIQUE,                -- unique constraint
    salary      DECIMAL(10,2) DEFAULT 0.00,         -- default value
    department  VARCHAR(50),
    hire_date   DATE NOT NULL DEFAULT CURRENT_DATE,
    active      BOOLEAN DEFAULT true,
    manager_id  INT,
    CHECK (salary >= 0),                            -- check constraint
    FOREIGN KEY (manager_id) REFERENCES employees(id) ON DELETE SET NULL
);
```

### ALTER TABLE

```sql
-- Add column
ALTER TABLE employees ADD COLUMN phone VARCHAR(20);

-- Drop column
ALTER TABLE employees DROP COLUMN phone;

-- Rename column
ALTER TABLE employees RENAME COLUMN dept TO department;

-- Modify column type
ALTER TABLE employees ALTER COLUMN salary TYPE NUMERIC(12,2);

-- Add constraint
ALTER TABLE employees ADD CONSTRAINT uq_email UNIQUE (email);

-- Drop constraint
ALTER TABLE employees DROP CONSTRAINT uq_email;
```

### DROP

```sql
-- Drop table (error if not exists)
DROP TABLE temp_data;

-- Drop table if it exists
DROP TABLE IF EXISTS temp_data;
```

### Indexes

```sql
-- Create index for faster lookups
CREATE INDEX idx_emp_department ON employees (department);

-- Composite index
CREATE INDEX idx_emp_dept_salary ON employees (department, salary);

-- Unique index
CREATE UNIQUE INDEX idx_emp_email ON employees (email);

-- Drop index
DROP INDEX idx_emp_department;
```

### Views

```sql
-- Create a view (virtual table)
CREATE VIEW active_employees AS
SELECT id, first_name, last_name, department FROM employees WHERE active = true;

-- Query a view like a table
SELECT * FROM active_employees WHERE department = 'Sales';

-- Drop a view
DROP VIEW IF EXISTS active_employees;
```

## Transactions

```sql
-- Basic transaction
BEGIN;
UPDATE accounts SET balance = balance - 500 WHERE id = 1;
UPDATE accounts SET balance = balance + 500 WHERE id = 2;
COMMIT;

-- Rollback on error
BEGIN;
UPDATE accounts SET balance = balance - 500 WHERE id = 1;
-- something went wrong
ROLLBACK;

-- Savepoints for partial rollback
BEGIN;
UPDATE inventory SET qty = qty - 1 WHERE product_id = 10;
SAVEPOINT before_shipping;
INSERT INTO shipments (product_id, qty) VALUES (10, 1);
-- undo just the shipment, keep inventory change
ROLLBACK TO SAVEPOINT before_shipping;
COMMIT;
```

## String Functions

```sql
SELECT
    UPPER('hello'),                    -- HELLO
    LOWER('HELLO'),                    -- hello
    LENGTH('hello'),                   -- 5
    TRIM('  hello  '),                 -- hello
    SUBSTRING('hello world', 1, 5),    -- hello
    REPLACE('hello', 'l', 'r'),        -- herro
    CONCAT(first_name, ' ', last_name) -- full name
FROM employees;
```

## Date Functions

```sql
SELECT
    CURRENT_DATE,                                -- today's date
    CURRENT_TIMESTAMP,                           -- current date and time
    EXTRACT(YEAR FROM hire_date),                 -- extract year
    DATE_TRUNC('month', hire_date),               -- truncate to month start
    hire_date + INTERVAL '1 year',                -- add interval
    AGE(CURRENT_DATE, hire_date)                  -- difference between dates
FROM employees;
```

## Tips

- Use parameterized queries to prevent SQL injection; never concatenate user input into SQL strings.
- Prefer explicit column lists over `SELECT *` in production code for clarity and performance.
- Always include a `WHERE` clause with `UPDATE` and `DELETE` to avoid modifying all rows.
- Use `EXPLAIN` or `EXPLAIN ANALYZE` before queries to understand the execution plan.
- Index columns that appear frequently in `WHERE`, `JOIN`, and `ORDER BY` clauses.
- Wrap related changes in transactions to ensure atomicity.
- CTEs improve readability but may not always be optimized the same as subqueries; check your database's behavior.
- Window functions avoid the need for self-joins and correlated subqueries in many reporting scenarios.
- Use `COALESCE(column, default)` to handle NULLs in expressions.
- Syntax varies across databases (MySQL, PostgreSQL, SQLite, SQL Server); always check your target dialect.

## References

- [SQL Standard (ISO 9075)](https://www.iso.org/standard/76583.html)
- [PostgreSQL SQL Reference](https://www.postgresql.org/docs/current/sql.html)
- [MySQL SQL Statement Syntax](https://dev.mysql.com/doc/refman/8.0/en/sql-statements.html)
- [SQLite SQL Syntax](https://www.sqlite.org/lang.html)
- [PostgreSQL Window Functions](https://www.postgresql.org/docs/current/tutorial-window.html)
- [SQL JOIN Types (PostgreSQL)](https://www.postgresql.org/docs/current/queries-table-expressions.html#QUERIES-JOIN)
- [Common Table Expressions (CTEs)](https://www.postgresql.org/docs/current/queries-with.html)
- [SQL Aggregate Functions (PostgreSQL)](https://www.postgresql.org/docs/current/functions-aggregate.html)
- [EXPLAIN Plan Analysis (PostgreSQL)](https://www.postgresql.org/docs/current/using-explain.html)
- [SQL Indexing and Tuning](https://use-the-index-luke.com/)
