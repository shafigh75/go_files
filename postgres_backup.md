To **back up and restore a PostgreSQL database** (including **roles, privileges, and full schema**) from one server to another **clean PostgreSQL instance**, follow this **step-by-step guide**.

---

## ✅ Requirements

* Source server: Running PostgreSQL with a database `wallet`
* Destination server: Fresh PostgreSQL installation
* You want:

  * Backup of database
  * Backup of roles/users/permissions
  * Restore everything on a new server

---

## 🔁 STEP-BY-STEP PROCESS

---

### 🔒 1. **Ensure access to both PostgreSQL instances**

Make sure you have access to both `psql`, `pg_dump`, and `pg_restore` or can connect remotely (or use SSH to copy files).

---

### 🧾 2. **Backup Global Roles and Permissions**

This step backs up all users, roles, grants, and their privileges.

```bash
pg_dumpall -U postgres -g > globals.sql
```

* `-g`: Dumps only global objects (roles, tablespaces)
* Result: A plain SQL file you’ll use later to recreate users/roles.

> ⚠️ Must be run as a superuser or a role with permission to read all roles.

---

### 💽 3. **Backup the "wallet" Database**

Use `pg_dump` to backup the database.

**Option 1: Custom Format (recommended for flexibility)**

```bash
pg_dump -U postgres -Fc wallet > wallet.backup
```

**Option 2: Plain SQL**

```bash
pg_dump -U postgres wallet > wallet.sql
```

---

### 🚚 4. **Copy Backups to New Server**

Transfer the files using `scp`, `rsync`, or any secure method.

```bash
scp globals.sql wallet.backup user@newserver:/tmp/
```

---

### 🛠️ 5. **Install PostgreSQL on New Server**

On the new server:

```bash
sudo apt update
sudo apt install postgresql
```

Ensure the same (or compatible) PostgreSQL version is used.

---

### 🔐 6. **Restore Global Roles**

On the new PostgreSQL server:

```bash
psql -U postgres -f /tmp/globals.sql
```

> ⚠️ This should be done before restoring the database, as permissions depend on roles.

---

### 🧬 7. **Create the Wallet Database**

Still on the new server:

```bash
createdb -U postgres wallet
```

> Make sure the database is created before using `pg_restore`.

---

### 📥 8. **Restore the Wallet Database**

**If you used custom format (`.backup`)**:

```bash
pg_restore -U postgres -d wallet /tmp/wallet.backup
```

**If you used plain SQL**:

```bash
psql -U postgres -d wallet -f /tmp/wallet.sql
```

---

## ✅ OPTIONAL: Verify the Result

Login and check roles, privileges, and data:

```bash
psql -U postgres -d wallet
\du         -- Check users/roles
\dt         -- List tables
SELECT * FROM some_table;
```

---

## 🧠 TIPS

* Ensure you have the same PostgreSQL version or newer on the new server.
* Make sure `pg_hba.conf` and `postgresql.conf` allow connections on the new server.
* If migrating between different OS or architectures, stick to plain SQL for maximum compatibility.

---

## 🧪 BONUS: One-Liner All-in-One Backup Command

```bash
pg_dumpall -U postgres > full_cluster.sql
```

This backs up **everything**: roles, DBs, schemas, and data (but not in custom format).

To restore:

```bash
psql -U postgres -f full_cluster.sql
```

> Use this only if you're not selectively restoring a database.

---



Here’s a **fully automated backup and restore script** for PostgreSQL, assuming you're doing this via terminal/SSH access between two Linux servers.

---

## 📁 Structure

We’ll split the script into two parts:

1. `pg_backup.sh` – Run on the **source server**
2. `pg_restore.sh` – Run on the **destination server**

---

## 📤 `pg_backup.sh` (run on **source server**)

```bash
#!/bin/bash

# === CONFIGURATION ===
DB_NAME="wallet"
BACKUP_DIR="/tmp/pg_backup"
USER="postgres"
REMOTE_USER="user"
REMOTE_HOST="new.server.ip"     # Replace with your destination IP
REMOTE_DIR="/tmp/pg_restore"

# === CREATE BACKUP DIR ===
mkdir -p "$BACKUP_DIR"

# === BACKUP GLOBALS (ROLES, PERMISSIONS) ===
echo "[*] Backing up global roles and privileges..."
pg_dumpall -U "$USER" -g > "$BACKUP_DIR/globals.sql"

# === BACKUP WALLET DATABASE ===
echo "[*] Backing up $DB_NAME database in custom format..."
pg_dump -U "$USER" -Fc "$DB_NAME" > "$BACKUP_DIR/$DB_NAME.backup"

# === TRANSFER FILES TO DESTINATION ===
echo "[*] Transferring backup files to destination server..."
ssh "$REMOTE_USER@$REMOTE_HOST" "mkdir -p $REMOTE_DIR"
scp "$BACKUP_DIR/globals.sql" "$BACKUP_DIR/$DB_NAME.backup" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/"

echo "[✓] Backup and transfer complete."
```

> ⚠️ Make sure the `postgres` user can dump roles and the destination server is reachable via SSH.

---

## 📥 `pg_restore.sh` (run on **destination server**)

```bash
#!/bin/bash

# === CONFIGURATION ===
DB_NAME="wallet"
BACKUP_DIR="/tmp/pg_restore"
USER="postgres"

# === RESTORE GLOBAL ROLES ===
echo "[*] Restoring global roles and permissions..."
psql -U "$USER" -f "$BACKUP_DIR/globals.sql"

# === CREATE DATABASE IF NOT EXISTS ===
echo "[*] Creating target database if not exists..."
psql -U "$USER" -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 || createdb -U "$USER" "$DB_NAME"

# === RESTORE DATABASE ===
echo "[*] Restoring $DB_NAME database..."
pg_restore -U "$USER" -d "$DB_NAME" "$BACKUP_DIR/$DB_NAME.backup"

echo "[✓] Restore complete. Verify roles and data manually."
```

---

## 🔐 Permissions Note

* Ensure `postgres` has passwordless access (via `.pgpass`) or you will be prompted.
* You can add `PGPASSWORD=yourpassword` before each command for non-interactive runs.

---

## ✅ Usage

### On source server:

```bash
chmod +x pg_backup.sh
./pg_backup.sh
```

### On destination server:

```bash
chmod +x pg_restore.sh
./pg_restore.sh
```

---

## 🧪 Optional Validation Step

After restore, run this on the destination:

```bash
psql -U postgres -d wallet -c "\dt"
psql -U postgres -c "\du"
```

---

