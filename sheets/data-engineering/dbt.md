# dbt (Data Build Tool)

Analytics engineering tool that transforms data in your warehouse using SQL SELECT statements, with built-in testing, documentation, and dependency management.

## Installation

```bash
# pip (dbt-core + adapter)
pip install dbt-core dbt-postgres        # PostgreSQL
pip install dbt-core dbt-bigquery        # BigQuery
pip install dbt-core dbt-snowflake       # Snowflake
pip install dbt-core dbt-redshift        # Redshift
pip install dbt-core dbt-duckdb          # DuckDB (local dev)

# Verify
dbt --version

# Initialize a new project
dbt init my_project
cd my_project
```

## Project Structure

```bash
my_project/
  dbt_project.yml          # project configuration
  profiles.yml             # connection profiles (~/.dbt/profiles.yml)
  models/
    staging/               # raw source cleaning
      stg_customers.sql
      stg_orders.sql
    marts/                 # business logic
      dim_customers.sql
      fct_orders.sql
    schema.yml             # tests and documentation
  seeds/                   # CSV files loaded as tables
    country_codes.csv
  snapshots/               # SCD Type 2 tracking
    snap_customers.sql
  macros/                  # reusable Jinja functions
    generate_schema.sql
  tests/                   # custom data tests
    assert_positive_amount.sql
  analysis/                # ad-hoc queries (compiled, not run)
```

## profiles.yml

```yaml
# ~/.dbt/profiles.yml
my_project:
  target: dev
  outputs:
    dev:
      type: postgres
      host: localhost
      port: 5432
      user: analytics
      password: "{{ env_var('DBT_PASSWORD') }}"
      dbname: warehouse
      schema: dev_analytics
      threads: 4

```

## dbt_project.yml

```yaml
name: 'my_project'
version: '1.0.0'
config-version: 2
profile: 'my_project'

model-paths: ["models"]
seed-paths: ["seeds"]
test-paths: ["tests"]
snapshot-paths: ["snapshots"]
macro-paths: ["macros"]

models:
  my_project:
    staging:
      +materialized: view
      +schema: staging
    marts:
      +materialized: table
      +schema: analytics
```

## Models

```sql
-- models/staging/stg_customers.sql
-- Materialized as view by default (from dbt_project.yml)
WITH source AS (
    SELECT * FROM {{ source('raw', 'customers') }}
)
SELECT
    id AS customer_id,
    LOWER(TRIM(email)) AS email,
    first_name || ' ' || last_name AS full_name,
    created_at,
    updated_at
FROM source
WHERE email IS NOT NULL

-- models/marts/dim_customers.sql
-- {{ config(materialized='table') }}
WITH customers AS (
    SELECT * FROM {{ ref('stg_customers') }}
),
orders AS (
    SELECT * FROM {{ ref('stg_orders') }}
)
SELECT
    c.customer_id,
    c.email,
    c.full_name,
    COUNT(o.order_id) AS total_orders,
    SUM(o.amount) AS lifetime_value,
    MIN(o.ordered_at) AS first_order_at,
    MAX(o.ordered_at) AS last_order_at
FROM customers c
LEFT JOIN orders o ON c.customer_id = o.customer_id
GROUP BY 1, 2, 3
```

## Materializations

```sql
-- View (default)
{{ config(materialized='view') }}

-- Table (full rebuild each run)
{{ config(materialized='table') }}

-- Incremental (append new rows only)
{{ config(
    materialized='incremental',
    unique_key='event_id',
    incremental_strategy='merge'
) }}
SELECT * FROM {{ source('raw', 'events') }}
{% if is_incremental() %}
WHERE event_time > (SELECT MAX(event_time) FROM {{ this }})
{% endif %}

-- Ephemeral (CTE injected into downstream models)
{{ config(materialized='ephemeral') }}
```

## Sources

```yaml
# models/staging/schema.yml
version: 2
sources:
  - name: raw
    database: warehouse
    schema: raw_data
    freshness:
      warn_after: {count: 12, period: hour}
      error_after: {count: 24, period: hour}
    loaded_at_field: _loaded_at
    tables:
      - name: customers
        description: "Raw customer data from app DB"
        columns:
          - name: id
            tests:
              - unique
              - not_null
      - name: orders
        description: "Raw order data"
```

```bash
# Check source freshness
dbt source freshness
```

## Tests

```yaml
# models/schema.yml
version: 2
models:
  - name: dim_customers
    description: "Customer dimension table"
    columns:
      - name: customer_id
        description: "Primary key"
        tests:
          - unique
          - not_null
      - name: email
        tests:
          - unique
          - not_null
      - name: total_orders
        tests:
          - not_null
          - dbt_utils.accepted_range:
              min_value: 0
      - name: lifetime_value
        tests:
          - dbt_utils.accepted_range:
              min_value: 0
              inclusive: true

  - name: fct_orders
    columns:
      - name: customer_id
        tests:
          - relationships:
              to: ref('dim_customers')
              field: customer_id
      - name: status
        tests:
          - accepted_values:
              values: ['pending', 'shipped', 'delivered', 'cancelled']
```

```sql
-- tests/assert_positive_revenue.sql (custom singular test)
-- Returns rows that fail the test (0 rows = pass)
SELECT order_id, amount
FROM {{ ref('fct_orders') }}
WHERE amount < 0
```

## Snapshots (SCD Type 2)

```sql
-- snapshots/snap_customers.sql
{% snapshot snap_customers %}
{{ config(
    target_schema='snapshots',
    unique_key='customer_id',
    strategy='timestamp',
    updated_at='updated_at'
) }}
SELECT * FROM {{ source('raw', 'customers') }}
{% endsnapshot %}
```

## Seeds

```bash
# Load CSV files from seeds/ directory
dbt seed

# Load specific seed
dbt seed --select country_codes

# Full refresh (drop and recreate)
dbt seed --full-refresh
```

## Macros (Jinja)

```sql
-- macros/cents_to_dollars.sql
{% macro cents_to_dollars(column_name, precision=2) %}
    ROUND({{ column_name }}::NUMERIC / 100, {{ precision }})
{% endmacro %}

-- Usage in models:
SELECT
    order_id,
    {{ cents_to_dollars('amount_cents') }} AS amount_dollars
FROM {{ ref('stg_orders') }}
```

## Packages

```yaml
# packages.yml
packages:
  - package: dbt-labs/dbt_utils
    version: [">=1.0.0", "<2.0.0"]
  - package: dbt-labs/codegen
    version: 0.12.1
  - git: https://github.com/org/dbt-package.git
    revision: v1.0.0
```

```bash
dbt deps                                         # install packages
```

## Core Commands

```bash
# Run models
dbt run                                          # run all models
dbt run --select dim_customers                   # run single model
dbt run --select staging.*                       # run all staging models
dbt run --select +dim_customers                  # run model and all upstream
dbt run --select dim_customers+                  # run model and all downstream
dbt run --select +dim_customers+                 # run upstream + model + downstream
dbt run --select tag:daily                       # run by tag
dbt run --full-refresh                           # rebuild incremental models

# Test
dbt test                                         # run all tests
dbt test --select dim_customers                  # test specific model
dbt test --select source:raw                     # test source

# Build (run + test in DAG order)
dbt build                                        # run + test everything
dbt build --select staging.*                     # build subset

# Documentation
dbt docs generate                                # generate docs
dbt docs serve --port 8080                       # serve docs site

# Other
dbt compile                                      # compile SQL without running
dbt debug                                        # test connection and config
dbt ls                                           # list resources
dbt clean                                        # remove target/ and packages/
```

## Tips

- Use ref() for all model references; never hardcode table names
- Organize models in staging (1:1 with sources, light cleaning) and marts (business logic)
- Prefer incremental models for large tables; always define a unique_key for merge strategy
- Run dbt build instead of dbt run + dbt test separately to get test-after-run in DAG order
- Use source freshness checks in CI to catch upstream pipeline failures early
- Keep macros DRY but readable; over-abstraction in Jinja makes debugging painful
- Tag models (e.g., tag:daily, tag:hourly) for selective scheduling in production
- Use dbt_utils package for common patterns: surrogate keys, pivot, unpivot, date spine
- Set threads in profiles.yml to match your warehouse parallelism (4-8 for dev, 16+ for prod)
- Use env_var() in profiles.yml for credentials; never commit passwords to version control
## See Also

- sql
- postgresql
- clickhouse

## References

- [dbt Documentation](https://docs.getdbt.com/)
- [dbt Best Practices](https://docs.getdbt.com/best-practices)
- [dbt-utils Package](https://hub.getdbt.com/dbt-labs/dbt_utils/latest/)
- [dbt Style Guide](https://docs.getdbt.com/best-practices/how-we-style/0-how-we-style-our-dbt-projects)
- [dbt GitHub](https://github.com/dbt-labs/dbt-core)
