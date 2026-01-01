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

Create a database and user for daylit:

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

-- For PostgreSQL 15+, you also need to grant schema privileges
\c daylit
GRANT ALL ON SCHEMA public TO daylit_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO daylit_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO daylit_user;
```

Exit psql with `\q`.

## Configuration

### Connection String Format

PostgreSQL connection strings follow this format:

```
postgres://username:password@hostname:port/database?options
```

or

```
postgresql://username:password@hostname:port/database?options
```

### Examples

**Local database:**
```bash
postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable
```

**Remote database with SSL:**
```bash
postgres://daylit_user:password@db.example.com:5432/daylit?sslmode=require
```

**WSL2 accessing Windows PostgreSQL:**
```bash
# Find your Windows host IP from WSL2
ip route | grep default | awk '{print $3}'
# Usually something like 172.x.x.x

postgres://daylit_user:password@172.20.240.1:5432/daylit?sslmode=disable
```

## Usage

### Method 1: Command Line (Explicit)

Pass the connection string with the `--config` flag:

```bash
daylit --config "postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable" init

daylit --config "postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable" task list

daylit --config "postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable" plan
```

### Method 2: Environment Variable (Recommended)

Set the `DAYLIT_CONFIG` environment variable:

**Bash/Zsh:**
```bash
export DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"

# Add to ~/.bashrc or ~/.zshrc for persistence
echo 'export DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"' >> ~/.bashrc
```

**Windows PowerShell:**
```powershell
$env:DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"

# For persistence:
[System.Environment]::SetEnvironmentVariable('DAYLIT_CONFIG', 'postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable', 'User')
```

Then use daylit normally:

```bash
daylit init
daylit task add "My Task"
daylit plan
```

### Method 3: Shell Alias

Create an alias for convenience:

```bash
# In ~/.bashrc or ~/.zshrc
alias daylit='daylit --config "postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"'
```

## Initialization

Before first use, initialize the database:

```bash
daylit --config "postgres://your-connection-string" init
```

This will:
1. Connect to the PostgreSQL database
2. Run all necessary migrations to create tables
3. Initialize default settings

## Security Considerations

### Connection String Security

**DO NOT** commit connection strings with passwords to version control!

**Safe options:**

1. **Use environment variables:**
   ```bash
   export POSTGRES_PASSWORD="your_password"
   export DAYLIT_CONFIG="postgres://daylit_user:${POSTGRES_PASSWORD}@localhost:5432/daylit?sslmode=disable"
   ```

2. **Use a `.pgpass` file:**
   Create `~/.pgpass` with permissions `0600`:
   ```
   localhost:5432:daylit:daylit_user:your_password
   ```
   Then use connection string without password:
   ```
   postgres://daylit_user@localhost:5432/daylit?sslmode=disable
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
   daylit --config "postgres:///?service=daylit"
   ```

### SSL/TLS

For remote databases, always use SSL:
```
postgres://user:password@host:5432/database?sslmode=require
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
Error: pq: permission denied for schema public
```

**Solution:**
Run the GRANT commands from the Database Setup section above.

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

1. Set up PostgreSQL database as described above
2. Initialize the PostgreSQL database: `daylit --config "postgres://..." init`
3. Export data from SQLite (manual process - copy tasks, plans, etc.)
4. Import into PostgreSQL using the appropriate `daylit` commands

Note: Automatic migration tools are not currently available. Manual data transfer is required.

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
