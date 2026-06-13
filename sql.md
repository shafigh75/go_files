Below is a **PostgreSQL-focused SQL cheatsheet** organized for real-world use: core SQL, schema/DDL, data manipulation, advanced querying, transactions, and PostgreSQL-specific administration (roles, permissions, types, introspection).

---

# 🧠 PostgreSQL SQL Cheatsheet (Comprehensive)

---

# 1. 📦 Database & Connection Commands (psql)

```sql
\l              -- list databases
\c dbname       -- connect to database
\dt             -- list tables
\d tablename    -- describe table
\du             -- list roles/users
\dn             -- list schemas
\df             -- list functions
```

---

# 2. 🏗️ Database & Schema Management

## Create / Drop DB

```sql
CREATE DATABASE mydb;
DROP DATABASE mydb;
```

## Schema

```sql
CREATE SCHEMA analytics;
DROP SCHEMA analytics CASCADE;
```

---

# 3. 🧱 Table Definition (DDL)

## Create Table

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    age INT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Data Types (PostgreSQL-specific highlights)

* `SERIAL`, `BIGSERIAL`
* `UUID`
* `JSON / JSONB`
* `ARRAY`
* `TEXT`, `VARCHAR`
* `TIMESTAMP WITH TIME ZONE (timestamptz)`
* `NUMERIC(p,s)`

---

## Alter Table

```sql
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT true;
ALTER TABLE users DROP COLUMN age;
ALTER TABLE users ALTER COLUMN name TYPE VARCHAR(200);
ALTER TABLE users RENAME COLUMN name TO full_name;
```

---

## Drop / Truncate

```sql
DROP TABLE users;
TRUNCATE TABLE users RESTART IDENTITY;
```

---

# 4. ✍️ Data Manipulation (DML)

## Insert

```sql
INSERT INTO users (name, email)
VALUES ('Ali', 'ali@mail.com');
```

## Update

```sql
UPDATE users
SET name = 'Reza'
WHERE id = 1;
```

## Delete

```sql
DELETE FROM users
WHERE id = 1;
```

---

## Upsert (VERY IMPORTANT PostgreSQL feature)

```sql
INSERT INTO users (email, name)
VALUES ('a@a.com', 'A')
ON CONFLICT (email)
DO UPDATE SET name = EXCLUDED.name;
```

---

# 5. 🔍 SELECT Basics

```sql
SELECT * FROM users;
SELECT name, email FROM users;
SELECT DISTINCT name FROM users;
```

## Filtering

```sql
SELECT * FROM users WHERE age > 18;
SELECT * FROM users WHERE name LIKE 'A%';
SELECT * FROM users WHERE age IN (18, 20, 25);
SELECT * FROM users WHERE email IS NOT NULL;
```

---

# 6. 🔗 Joins

## Inner Join

```sql
SELECT u.name, o.amount
FROM users u
JOIN orders o ON u.id = o.user_id;
```

## Left Join

```sql
SELECT u.name, o.amount
FROM users u
LEFT JOIN orders o ON u.id = o.user_id;
```

## Right Join (rare)

```sql
SELECT * FROM users u
RIGHT JOIN orders o ON u.id = o.user_id;
```

## Full Join

```sql
SELECT * FROM users u
FULL JOIN orders o ON u.id = o.user_id;
```

---

# 7. 📊 Aggregations

```sql
SELECT COUNT(*) FROM users;
SELECT AVG(age) FROM users;
SELECT MAX(age), MIN(age) FROM users;
```

## GROUP BY

```sql
SELECT age, COUNT(*)
FROM users
GROUP BY age;
```

## HAVING

```sql
SELECT age, COUNT(*)
FROM users
GROUP BY age
HAVING COUNT(*) > 1;
```

---

# 8. 📈 Ordering & Limiting

```sql
SELECT * FROM users ORDER BY created_at DESC;
SELECT * FROM users LIMIT 10;
SELECT * FROM users OFFSET 10 LIMIT 10;
```

---

# 9. 🔎 Subqueries

## Scalar subquery

```sql
SELECT name
FROM users
WHERE age > (SELECT AVG(age) FROM users);
```

## IN subquery

```sql
SELECT * FROM users
WHERE id IN (SELECT user_id FROM orders);
```

## EXISTS

```sql
SELECT * FROM users u
WHERE EXISTS (
    SELECT 1 FROM orders o WHERE o.user_id = u.id
);
```

---

# 10. 🧠 CTE (Common Table Expressions)

```sql
WITH active_users AS (
    SELECT * FROM users WHERE is_active = true
)
SELECT * FROM active_users;
```

## Recursive CTE

```sql
WITH RECURSIVE nums AS (
    SELECT 1 AS n
    UNION ALL
    SELECT n + 1 FROM nums WHERE n < 10
)
SELECT * FROM nums;
```

---

# 11. 🪟 Window Functions (Advanced + Very Important)

## ROW_NUMBER

```sql
SELECT name,
       ROW_NUMBER() OVER (ORDER BY created_at)
FROM users;
```

## RANK / DENSE_RANK

```sql
SELECT name,
       RANK() OVER (ORDER BY age DESC)
FROM users;
```

## Partitioned window

```sql
SELECT name, age,
       AVG(age) OVER (PARTITION BY age)
FROM users;
```

## Running total

```sql
SELECT id, amount,
       SUM(amount) OVER (ORDER BY id)
FROM orders;
```

---

# 12. 🔁 Transactions

```sql
BEGIN;

UPDATE accounts SET balance = balance - 100 WHERE id = 1;
UPDATE accounts SET balance = balance + 100 WHERE id = 2;

COMMIT;
-- or
ROLLBACK;
```

## Savepoint

```sql
BEGIN;
SAVEPOINT sp1;
ROLLBACK TO sp1;
COMMIT;
```

---

# 13. 🔐 PostgreSQL Roles & Users (VERY IMPORTANT)

## Create Role (User)

```sql
CREATE ROLE dev_user LOGIN PASSWORD 'secret';
```

## Create superuser

```sql
CREATE ROLE admin SUPERUSER LOGIN PASSWORD 'secret';
```

## Alter role

```sql
ALTER ROLE dev_user CREATEDB;
ALTER ROLE dev_user PASSWORD 'newpass';
```

## Drop role

```sql
DROP ROLE dev_user;
```

---

## View Users / Roles

```sql
SELECT * FROM pg_roles;
\du
```

---

# 14. 🔑 Privileges (GRANT / REVOKE)

## Grant database access

```sql
GRANT CONNECT ON DATABASE mydb TO dev_user;
```

## Schema access

```sql
GRANT USAGE ON SCHEMA public TO dev_user;
```

## Table permissions

```sql
GRANT SELECT, INSERT, UPDATE ON users TO dev_user;
REVOKE DELETE ON users FROM dev_user;
```

## Grant all tables in schema

```sql
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO dev_user;
```

---

## Default privileges (important in production)

```sql
ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT SELECT ON TABLES TO dev_user;
```

---

# 15. 🧬 PostgreSQL Custom Types

## ENUM type

```sql
CREATE TYPE mood AS ENUM ('happy', 'sad', 'neutral');
```

```sql
CREATE TABLE person (
    name TEXT,
    current_mood mood
);
```

---

## Composite type

```sql
CREATE TYPE address AS (
    city TEXT,
    zip INT
);
```

---

## Using it

```sql
CREATE TABLE users (
    id SERIAL,
    home address
);
```

---

# 16. 🧾 JSON / JSONB (PostgreSQL feature)

```sql
CREATE TABLE events (
    id SERIAL,
    data JSONB
);
```

## Query JSON

```sql
SELECT data->>'name' FROM events;
SELECT data->'user'->>'id' FROM events;
```

## Filter JSON

```sql
SELECT * FROM events
WHERE data @> '{"type":"click"}';
```

---

# 17. 📐 Indexes

```sql
CREATE INDEX idx_users_email ON users(email);
CREATE UNIQUE INDEX idx_users_email_unique ON users(email);
```

## Partial index

```sql
CREATE INDEX idx_active_users ON users(id)
WHERE is_active = true;
```

## GIN index (for JSONB)

```sql
CREATE INDEX idx_json ON events USING GIN (data);
```

---

# 18. 🔍 Schema Introspection (VERY USEFUL)

```sql
SELECT * FROM information_schema.tables;
SELECT * FROM information_schema.columns WHERE table_name='users';
```

## PostgreSQL system catalogs

```sql
SELECT * FROM pg_tables;
SELECT * FROM pg_class;
SELECT * FROM pg_indexes;
```

---

# 19. ⚡ Performance / Execution

```sql
EXPLAIN SELECT * FROM users;
EXPLAIN ANALYZE SELECT * FROM users;
```

---

# 20. 🔄 Sequences (PostgreSQL internals)

```sql
SELECT nextval('users_id_seq');
SELECT currval('users_id_seq');
```

Reset:

```sql
ALTER SEQUENCE users_id_seq RESTART WITH 1;
```

---

# 21. 🧰 Utility Commands (psql)

```sql
\copy users TO 'file.csv' CSV HEADER;
\copy users FROM 'file.csv' CSV HEADER;
```

---

# 22. 🚀 Quick Advanced Patterns

## Pagination (safe version)

```sql
SELECT * FROM users
ORDER BY id
LIMIT 20 OFFSET 40;
```

## Top-N per group (window)

```sql
SELECT *
FROM (
    SELECT *,
           ROW_NUMBER() OVER (PARTITION BY age ORDER BY id) rn
    FROM users
) t
WHERE rn <= 3;
```

---

# 23. 🔥 Must-know PostgreSQL Features Summary

* `JSONB` indexing (GIN)
* `CTE + recursive queries`
* `Window functions`
* `Upsert (ON CONFLICT)`
* `Roles & inheritance`
* `Partial indexes`
* `Default privileges`
* `Composite + ENUM types`

---

Below is a **deeper, more conceptual + practical explanation** of the three topics in PostgreSQL, with emphasis on *how the engine actually behaves*, not just syntax.

---

# 1. 🔁 UPSERT in PostgreSQL (`INSERT ... ON CONFLICT`)

## What it is (conceptually)

An **upsert** = *insert if not exists, otherwise update existing row*.

It solves the classic race condition:

> “Check if row exists → insert or update”
> This is unsafe under concurrency.

PostgreSQL fixes this with **atomic conflict handling inside the INSERT statement itself**.

---

## Core syntax

```sql
INSERT INTO users (email, name)
VALUES ('a@a.com', 'Ali')
ON CONFLICT (email)
DO UPDATE SET name = EXCLUDED.name;
```

---

## 🔍 Key idea: “Conflict Target”

PostgreSQL needs a **uniquely constrained column**:

* PRIMARY KEY
* UNIQUE constraint
* UNIQUE index

Example:

```sql
CREATE UNIQUE INDEX users_email_unique ON users(email);
```

---

## ⚙️ Execution logic (important)

When you run:

```sql
INSERT ... ON CONFLICT (email)
```

PostgreSQL internally does:

1. Try insert row
2. If no constraint violation → success
3. If violation → jump to conflict handler
4. Execute DO UPDATE or DO NOTHING

👉 This happens **inside a single statement and single transaction context**

---

## DO UPDATE behavior

```sql
ON CONFLICT (email)
DO UPDATE SET
    name = EXCLUDED.name;
```

### Important keyword:

* `EXCLUDED` = the row you *tried to insert*

So:

| Existing row | Incoming row |
| ------------ | ------------ |
| name = Ali   | name = Reza  |

becomes:

```sql
name = EXCLUDED.name  → "Reza"
```

---

## Conditional update (very common)

```sql
ON CONFLICT (email)
DO UPDATE SET name = EXCLUDED.name
WHERE users.name IS DISTINCT FROM EXCLUDED.name;
```

👉 avoids unnecessary writes (important for performance + WAL reduction)

---

## DO NOTHING

```sql
ON CONFLICT (email)
DO NOTHING;
```

Use when:

* you want “insert if new, ignore if exists”

---

## Multi-column conflict target

```sql
ON CONFLICT (email, tenant_id)
```

👉 requires composite unique constraint.

---

## RETURNING (VERY important)

```sql
INSERT INTO users (email, name)
VALUES ('a@a.com', 'Ali')
ON CONFLICT (email)
DO UPDATE SET name = EXCLUDED.name
RETURNING *;
```

Returns:

* inserted row OR updated row

---

## Real-world usage pattern

* user registration (idempotent API)
* event ingestion pipelines
* caching tables
* sync jobs

---

# 2. 🧠 CTE (Common Table Expressions)

## What is a CTE?

A **CTE is a named temporary result set** defined inside a query.

```sql
WITH active_users AS (
    SELECT * FROM users WHERE is_active = true
)
SELECT * FROM active_users;
```

Think of it as:

> “a temporary view that exists only for this query”

---

## Why CTEs exist

They help with:

* readability
* query decomposition
* recursion
* reusing subqueries
* step-by-step transformations

---

## ⚙️ Execution model (VERY IMPORTANT)

### Classic PostgreSQL behavior:

CTEs are:

> **materialized by default (older versions)**

Meaning:

1. CTE is computed fully
2. Stored temporarily
3. Main query runs on result

---

### Modern PostgreSQL (12+ optimization)

CTEs can be:

* **inlined (optimized away)** OR
* **materialized (forced execution boundary)**

Optimizer decides unless you force it:

```sql
WITH cte AS MATERIALIZED (
    SELECT ...
)
```

or

```sql
WITH cte AS NOT MATERIALIZED (
    SELECT ...
)
```

---

## 🔥 Example 1: Step-by-step transformation

```sql
WITH filtered AS (
    SELECT * FROM orders WHERE amount > 100
),
summed AS (
    SELECT user_id, SUM(amount) AS total
    FROM filtered
    GROUP BY user_id
)
SELECT * FROM summed;
```

Execution flow:

1. `filtered` runs
2. `summed` runs on filtered
3. final select runs

---

## 🔥 Example 2: Data cleaning pipeline

```sql
WITH cleaned AS (
    SELECT
        LOWER(email) AS email,
        name
    FROM users
),
dedup AS (
    SELECT DISTINCT ON (email) *
    FROM cleaned
    ORDER BY email
)
SELECT * FROM dedup;
```

---

## 🔁 Recursive CTE (graph / hierarchy)

### Example: org chart / tree traversal

```sql
WITH RECURSIVE org AS (
    -- base case
    SELECT id, manager_id, name
    FROM employees
    WHERE manager_id IS NULL

    UNION ALL

    -- recursive step
    SELECT e.id, e.manager_id, e.name
    FROM employees e
    JOIN org o ON e.manager_id = o.id
)
SELECT * FROM org;
```

---

### Execution model of recursive CTE:

1. Run base query
2. Store results in working table
3. Repeat:

   * run recursive query using previous results
   * append new rows
4. stop when no new rows

---

## 🧠 Important performance note

Bad recursive CTEs can become:

* O(n²)
* memory-heavy
* slow due to repeated joins

---

# 3. 🪟 Window Functions (DEEP EXPLANATION)

---

## What are window functions?

A **window function performs a calculation across a set of rows related to the current row — without collapsing rows.**

---

### Key difference:

| Group By          | Window Function      |
| ----------------- | -------------------- |
| collapses rows    | keeps rows           |
| aggregates result | adds computed column |

---

## Example intuition

You want:

> “total salary per department, but still show each employee”

### GROUP BY (loses detail)

```sql
SELECT dept, SUM(salary)
FROM employees
GROUP BY dept;
```

### Window function (keeps detail)

```sql
SELECT name, dept, salary,
       SUM(salary) OVER (PARTITION BY dept)
FROM employees;
```

---

# ⚙️ Execution order (CRITICAL)

Window functions are computed in this order:

### SQL logical processing order:

1. FROM
2. WHERE
3. GROUP BY
4. HAVING
5. SELECT
6. WINDOW FUNCTIONS ← 👈 HERE
7. ORDER BY
8. LIMIT

---

## Important implication

You **cannot use window functions in WHERE**

❌ invalid:

```sql
SELECT name,
       ROW_NUMBER() OVER (ORDER BY id) rn
FROM users
WHERE rn = 1;
```

✔ correct:

```sql
SELECT *
FROM (
    SELECT name,
           ROW_NUMBER() OVER (ORDER BY id) rn
    FROM users
) t
WHERE rn = 1;
```

---

# 🧠 How window functions work internally

For:

```sql
ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC)
```

PostgreSQL does:

### Step 1: Partition rows

Split table into groups:

```
HR → [rows]
IT → [rows]
Sales → [rows]
```

### Step 2: Sort inside each partition

```
HR → sorted by salary
```

### Step 3: Compute function per row

Assign row numbers sequentially.

---

## Common window functions

### Ranking functions

```sql
ROW_NUMBER()
RANK()
DENSE_RANK()
```

Difference:

| Function   | Behavior        |
| ---------- | --------------- |
| ROW_NUMBER | unique sequence |
| RANK       | gaps on ties    |
| DENSE_RANK | no gaps         |

---

### Example

| salary | row_number | rank | dense_rank |
| ------ | ---------- | ---- | ---------- |
| 100    | 1          | 1    | 1          |
| 100    | 2          | 1    | 1          |
| 90     | 3          | 3    | 2          |

---

## Aggregates as window functions

```sql
SUM(amount) OVER (PARTITION BY user_id)
AVG(amount) OVER ()
MAX(amount) OVER ()
```

---

## Running total (very common)

```sql
SELECT
    id,
    amount,
    SUM(amount) OVER (ORDER BY id)
FROM payments;
```

👉 This creates a **cumulative sum**

---

## Moving window (rolling analytics)

```sql
SELECT
    id,
    amount,
    AVG(amount) OVER (
        ORDER BY id
        ROWS BETWEEN 2 PRECEDING AND CURRENT ROW
    )
FROM payments;
```

This = rolling average of last 3 rows.

---

## Frame vs Partition (VERY IMPORTANT)

### Partition

Defines *scope*

```sql
PARTITION BY user_id
```

### Frame

Defines *window range inside partition*

```sql
ROWS BETWEEN 5 PRECEDING AND CURRENT ROW
```

---

## Mental model

Think of window functions as:

> “for each row, run an aggregation on a dynamically defined subset of rows”

---

# 🔥 Summary (high-signal)

## UPSERT

* Atomic insert-or-update
* Requires unique constraint
* Uses `EXCLUDED`
* Prevents race conditions

## CTE

* Named query block
* Can be materialized or inlined
* Useful for multi-step queries
* Recursive version = graph traversal engine

## Window Functions

* Aggregation without collapsing rows
* Computed after SELECT phase
* Works on partitions + ordered frames
* Used for ranking, analytics, running totals

---

