### import a bunch of schemas in a directory to a folder:

imagine we have this directory structure:

```bash
root@postgresql:/mnt/postgresql/backup_pdb1# ls -l
total 396
-rw-r--r-- 1 root root 12172 Jun  9 13:31 globals_play.sql
-rw-r--r-- 1 root root  8622 Jun  9 16:31 launcher_next_schema.sql
-rw-r--r-- 1 root root 41440 Jun  9 18:01 mag_schema.sql
-rw-r--r-- 1 root root 21479 Jun 10 11:29 play_ads_schema.sql
-rw-r--r-- 1 root root  6273 Jun  9 15:43 play_files_schema.sql
-rw-r--r-- 1 root root 23047 Jun  9 13:30 play_fwc_schema.sql
-rw-r--r-- 1 root root 26585 Jun  9 16:10 play_home_schema.sql
-rw-r--r-- 1 root root 72093 Jun  9 17:30 play_movie_schema.sql
-rw-r--r-- 1 root root 77718 Jun  9 16:59 play_movie_tmp_schema.sql
-rw-r--r-- 1 root root 48562 Jun 25 16:05 play_schema.sql
-rw-r--r-- 1 root root 23362 Jun 10 12:42 play_user_action_schema.sql
-rw-r--r-- 1 root root 16487 Jun 10 13:52 promotion_schema.sql
-rw-r--r-- 1 root root  1104 Jun 28 13:53 script.sh
```

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
