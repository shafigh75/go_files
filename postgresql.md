### import a bunch of schemas in a directory to a folder:

```bash
#!/bin/bash

# === Configuration ===
PG_USER="postgres"
PG_PASSWORD="your_pg_password"
PG_HOST="localhost"
PG_PORT="5432"
SCHEMA_DIR="."  # Directory containing the .sql files

# === Script ===
export PGPASSWORD="$PG_PASSWORD"
shopt -s nullglob

for schema_file in "$SCHEMA_DIR"/*_schema.sql; do
    base_file=$(basename "$schema_file")
    db_name="${base_file%%_schema.sql}"

    echo "📦 Processing file: $base_file → DB: $db_name"

    # Check if DB exists
    db_exists=$(psql -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" -tAc "SELECT 1 FROM pg_database WHERE datname='$db_name'")

    if [[ "$db_exists" != "1" ]]; then
        echo "🛠️  Creating database $db_name..."
        createdb -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" "$db_name"
    else
        echo "⚠️  Database $db_name already exists, skipping creation."
    fi

    # Import schema
    echo "⬆️  Importing $base_file into $db_name..."
    psql -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" -d "$db_name" -f "$schema_file"

    echo "✅ Done with $db_name"
    echo "---------------------------"
done

echo "🎉 All schemas processed."
```
