# PostgreSQL Storage Backend

This document describes how to set up and use the PostgreSQL storage backend for daylit.

## Overview

The PostgreSQL storage backend allows daylit to connect to a centralized PostgreSQL database server instead of using a local SQLite file. This is particularly useful when:

- Running `daylit-cli` in WSL2 while `daylit-tray` runs on Windows (avoids file locking issues)
- Multiple clients need concurrent access to the same data
- You want centralized database management and backups

## Prerequisites

### 1. PostgreSQL Server

You need access to a PostgreSQL server (version 12 or later recommended). You can:

- Use an existing PostgreSQL server
- Install PostgreSQL locally:
  - **Ubuntu/Debian**: `sudo apt install postgresql`
  - **macOS**: `brew install postgresql`
  - **Windows**: Download from [postgresql.org](https://www.postgresql.org/download/windows/)
- Use a cloud PostgreSQL service (AWS RDS, Google Cloud SQL, etc.)

### 2. Database Setup

Because daylit isolates its data in a `daylit` schema, you can either create a dedicated database or use an existing one.

#### Option A: Dedicated Database (Recommended)

```bash
# Connect to PostgreSQL as superuser
sudo -u postgres psql

# Or on macOS/Windows:
psql -U postgres
```

```sql
-- Create database
CREATE DATABASE daylit;

-- Create user with password
CREATE USER daylit_user WITH PASSWORD 'your_secure_password_here';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE daylit TO daylit_user;

-- Connect to the database
\c daylit

-- Create the daylit schema and authorize the user
CREATE SCHEMA daylit AUTHORIZATION daylit_user;

-- Set the search path for the user so they default to the daylit schema
ALTER USER daylit_user SET search_path TO daylit;
```

#### Option B: Existing Database

If you prefer to use an existing database (e.g., `my_shared_db`), just create the schema and user within it:

```sql
-- Connect to your existing database
\c my_shared_db

-- Create user (if not using an existing one)
CREATE USER daylit_user WITH PASSWORD 'your_secure_password_here';

-- Grant connect privileges
GRANT CONNECT ON DATABASE my_shared_db TO daylit_user;

-- Create the daylit schema and authorize the user
CREATE SCHEMA daylit AUTHORIZATION daylit_user;

-- Set the search path for the user
ALTER USER daylit_user SET search_path TO daylit;
```

Exit psql with `\q`.

## Configuration

### Connection String Format

PostgreSQL connection strings follow this format:

```
postgres://username@hostname:port/database?options
```

or

```
postgresql://username@hostname:port/database?options
```

**IMPORTANT SECURITY NOTE:** As of the latest version, daylit enforces secure credential handling. Connection strings with embedded passwords (e.g., `postgres://user:password@host/db`) are **NOT ALLOWED** via the `--config` flag. You must use one of the secure alternatives described below.

### Secure Credential Management

daylit supports three secure methods for providing database credentials:

#### 1. Environment Variable (Recommended for Automation)

Set the connection string with credentials in the `DAYLIT_CONFIG` environment variable:

**Bash/Zsh:**
```bash
export DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"

# Then use daylit without the --config flag
daylit init
```

**Windows PowerShell:**
```powershell
$env:DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"

# For persistence:
[System.Environment]::SetEnvironmentVariable('DAYLIT_CONFIG', 'postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable', 'User')
```

#### 2. .pgpass File (Recommended for Interactive Use)

PostgreSQL's standard password file provides secure, automatic credential management.

Create `~/.pgpass` (Linux/macOS) or `%APPDATA%\postgresql\pgpass.conf` (Windows) with permissions `0600`:

```
localhost:5432:daylit:daylit_user:your_password
```

**Set permissions (Linux/macOS):**
```bash
chmod 0600 ~/.pgpass
```

Then use connection string without password:
```bash
daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" init
```

#### 3. OS Keyring (Future Enhancement)

Support for OS keyring storage is planned for a future release. Use `.pgpass` or environment variables in the meantime.
> **Note:** The application automatically appends `search_path=daylit` to the connection string to ensure all tables are created within the `daylit` schema for isolation.

### Examples

**Local database with .pgpass:**
```bash
# Password stored in ~/.pgpass
daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" init
```

**Remote database with environment variable:**
```bash
export DAYLIT_CONFIG="postgres://daylit_user:secure_password@db.example.com:5432/daylit?sslmode=require"
daylit task list
```

**WSL2 accessing Windows PostgreSQL with .pgpass:**
```bash
# Find your Windows host IP from WSL2
ip route | grep default | awk '{print $3}'
# Usually something like 172.x.x.x

# Add to ~/.pgpass:
# 172.20.240.1:5432:daylit:daylit_user:your_password

daylit --config "postgres://daylit_user@172.20.240.1:5432/daylit?sslmode=disable" plan
```

## Usage

### Command Line Restrictions (What is NOT Allowed)

⚠️ **SECURITY WARNING:** Passing connection strings with embedded passwords via command line is **NOT ALLOWED** as it exposes credentials in shell history and process lists.

The following usage will be blocked by the application:

```bash
# ❌ BLOCKED - Password in connection string
daylit --config "postgres://user:password@host:5432/daylit" init
```

Use the secure methods described below instead.

### Secure Method 1: .pgpass File (Recommended for Interactive Use)

Create a `.pgpass` file with your credentials and use a connection string without password:

**Setup:**
```bash
# Create ~/.pgpass with secure permissions
echo "localhost:5432:daylit:daylit_user:your_password" > ~/.pgpass
chmod 0600 ~/.pgpass

# Use connection string without password
daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" init
daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" task list
```

### Secure Method 2: Environment Variable (Recommended for Automation)

Set the connection credentials in the `DAYLIT_CONFIG` environment variable:

**Bash/Zsh:**
```bash
# Set for current session
export DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"

# Or add to ~/.bashrc or ~/.zshrc for persistence
echo 'export DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"' >> ~/.bashrc

# Use without --config flag
daylit init
```

**Windows PowerShell:**
```powershell
$env:DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"

# For persistence:
[System.Environment]::SetEnvironmentVariable('DAYLIT_CONFIG', 'postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable', 'User')
```

### Secure Method 3: Shell Alias with .pgpass

Create an alias for convenience (requires .pgpass setup):

```bash
# In ~/.bashrc or ~/.zshrc
alias daylit='daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable"'

# Then use normally
daylit init
daylit task add "My Task"
daylit plan
```

## Initialization

Before first use, initialize the database. Make sure you're using a secure credential method:

**With .pgpass file (recommended):**
```bash
# Ensure ~/.pgpass contains your credentials
daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" init
```

**With environment variable:**
```bash
export DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"
daylit init
```

This usage is allowed because `DAYLIT_CONFIG` is read from the environment and is **not** exposed via process listings (for example, `ps` output), unlike `--config` command-line arguments. However, embedding passwords in environment variables can still be risky if they end up in shell history, crash dumps, logs, or other debug tooling. Prefer the `.pgpass` file or the `PGPASSWORD` environment variable for credentials whenever possible, and only use an embedded password in `DAYLIT_CONFIG` in controlled environments (for example, systemd unit files or a secrets-managed runtime configuration).

This will:
1. Connect to the PostgreSQL database
2. Create the `daylit` schema if it doesn't exist
3. Run all necessary migrations to create tables
4. Initialize default settings

**Note:** The restriction on embedding passwords applies specifically to the `--config` command-line flag because command-line arguments are visible in process listings. The connection string you provide via `--config` should **not** contain a password. Credentials should currently be provided through `.pgpass`, the `PGPASSWORD` environment variable, or (if necessary) a carefully managed `DAYLIT_CONFIG` environment variable. OS keyring integration is planned for a future release and is not yet available.

## Security Considerations

### Connection String Security

**CRITICAL:** daylit enforces secure credential handling. Connection strings with embedded passwords are blocked at runtime.

**What is NOT allowed:**
```bash
# ❌ BLOCKED - Password in connection string
daylit --config "postgres://user:password@host:5432/daylit" init

# ❌ BLOCKED - Password in DSN format
daylit --config "host=localhost password=secret dbname=daylit" init
```

**What IS allowed:**

1. **Use environment variables:**
   ```bash
   export PGPASSWORD="your_password"
   # Connection string used by libpq will read from environment
   daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" init
   ```

2. **Use a `.pgpass` file (RECOMMENDED):**
   Create `~/.pgpass` with permissions `0600`:
   ```
   localhost:5432:daylit:daylit_user:your_password
   ```
   Then use connection string without password:
   ```bash
   daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" init
   ```

3. **Use PostgreSQL service file:**
   Create `~/.pg_service.conf`:
   ```ini
   [daylit]
   host=localhost
   port=5432
   dbname=daylit
   user=daylit_user
   password=your_password
   ```
   Then use:
   ```bash
   daylit --config "postgres:///?service=daylit" init
   ```

### Why This Matters

Embedding passwords in command-line arguments is insecure because:
- **Shell History**: Commands are saved in `.bash_history`, `.zsh_history`, etc.
- **Process Lists**: Passwords visible in `ps`, `/proc`, Task Manager
- **Logging**: CI/CD systems and debugging tools may log commands
- **Scripts**: Encourages committing credentials to version control

The secure methods above avoid these issues by:
- Storing credentials in files with restricted permissions
- Using PostgreSQL's built-in credential mechanisms
- Keeping credentials out of command-line arguments

### SSL/TLS

For remote databases, always use SSL. When setting `DAYLIT_CONFIG`, include the `sslmode` parameter:

```bash
export DAYLIT_CONFIG="postgres://daylit_user:your_password@host:5432/daylit?sslmode=require"
```

SSL modes:
- `disable`: No SSL (only for local/trusted networks)
- `require`: SSL required (doesn't verify server certificate)
- `verify-ca`: SSL required, verify server certificate
- `verify-full`: SSL required, verify server certificate and hostname

## Migrations

The PostgreSQL backend uses its own set of migrations located in `migrations/postgres/`. These are automatically applied when you run `init` or when the application starts if needed.

To manually run migrations:

```bash
daylit --config "postgres://your-connection-string" migrate
```

## Troubleshooting

### Connection Refused

```
Error: failed to connect to database: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Solutions:**
- Ensure PostgreSQL is running: `sudo systemctl status postgresql` (Linux) or check Services on Windows
- Verify the host and port in your connection string
- Check PostgreSQL is listening on the correct interface (edit `postgresql.conf`: `listen_addresses = '*'`)

### Permission Denied

```
Error: pq: permission denied for schema daylit
```

**Solution:**
Ensure the `daylit` schema exists and the user has permissions. Run the schema creation and authorization commands from the Database Setup section above.

### Authentication Failed

```
Error: pq: password authentication failed for user "daylit_user"
```

**Solutions:**
- Verify password is correct
- Check `pg_hba.conf` allows password authentication
- Try resetting password: `ALTER USER daylit_user WITH PASSWORD 'new_password';`

### Database Not Found

```
Error: pq: database "daylit" does not exist
```

**Solution:**
Create the database as shown in the Database Setup section.

### Migration Errors

If migrations fail:

1. Check the error message carefully
2. Verify database permissions
3. If needed, you can manually inspect/fix the database:
   ```bash
   psql -U daylit_user -d daylit
   # Check schema_version table
   SELECT * FROM schema_version;
   ```

## Performance Tips

1. **Connection Pooling**: The application manages connections efficiently, but for high-concurrency scenarios, consider using pgBouncer.

2. **Indexes**: The migrations create necessary indexes. For large datasets, monitor query performance with `EXPLAIN ANALYZE`.

3. **Backups**: Use PostgreSQL's built-in backup tools:
   ```bash
   pg_dump -U daylit_user daylit > daylit_backup.sql
   ```

4. **Monitoring**: Use PostgreSQL's query logging and monitoring tools to track performance.

## Switching from SQLite to PostgreSQL

To migrate existing SQLite data to PostgreSQL:

1. Set up PostgreSQL database as described above.
2. Run the `init` command with the `--source` flag pointing to your existing SQLite database:

```bash
# Using .pgpass for credentials (recommended)
daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" init --source ~/.config/daylit/daylit.db
```

This will:
1. Initialize the PostgreSQL schema.
2. Automatically migrate all data (tasks, plans, habits, settings, etc.) from the source SQLite database to PostgreSQL.

## Concurrent Access

One of the main benefits of PostgreSQL is proper concurrent access. You can safely:

- Run `daylit-cli` from multiple terminals simultaneously
- Run `daylit-tray` on Windows while using `daylit-cli` in WSL2
- Access from multiple machines (if PostgreSQL is network-accessible)

All operations are properly synchronized by PostgreSQL's transaction management.

## Testing

To run integration tests against a PostgreSQL database:

```bash
export POSTGRES_TEST_URL="postgres://daylit_user:password@localhost:5432/daylit_test?sslmode=disable"
go test ./internal/storage/... -v
```

Make sure to use a separate test database to avoid affecting your production data!
