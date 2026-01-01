# Storage Configuration

The default configuration file is located at `~/.config/daylit/daylit.db`.

You can specify a different location using the `--config` flag or the `DAYLIT_CONFIG` environment variable:

```bash
# Using flag
daylit --config /path/to/config.db init

# Using environment variable
export DAYLIT_CONFIG="/path/to/config.db"
daylit init
```

## PostgreSQL Backend

daylit also supports PostgreSQL as a storage backend.

**Security Restriction:** To prevent credential leakage in process lists, connection strings with embedded passwords are **blocked** when passed via the `--config` command-line flag.

However, you **can** securely use embedded passwords by setting the `DAYLIT_CONFIG` environment variable, as environment variables are not visible to other users on the system.

**Secure usage with Environment Variable:**
```bash
export DAYLIT_CONFIG="postgres://daylit_user:password@localhost:5432/daylit?sslmode=disable"
daylit init
```

**Secure usage with .pgpass:**
```bash
# Setup .pgpass file with credentials (recommended)
echo "localhost:5432:daylit:daylit_user:password" > ~/.pgpass
chmod 0600 ~/.pgpass

# Use connection string WITHOUT password
daylit --config "postgres://daylit_user@localhost:5432/daylit?sslmode=disable" init
```

See [POSTGRES_SETUP.md](POSTGRES_SETUP.md) for comprehensive PostgreSQL configuration instructions, including:
- Secure credential management (.pgpass, environment variables)
- Database setup and permissions
- SSL/TLS configuration
- Troubleshooting

## Database Schema

The application uses a SQL database for storage (SQLite or PostgreSQL). The database contains tables for:

- `schema_version`: Schema version tracking for migrations
- `settings`: Key-value store for application configuration (including OT settings)
- `tasks`: Task definitions and templates
- `plans`: Daily plans (keyed by date)
- `slots`: Time slots within plans, linking tasks to specific times
- `habits`: Habit definitions
- `habit_entries`: Daily records of habit completions
- `ot_entries`: Daily "One Thing" (OT) intentions

The schema is managed through migrations stored in the `migrations/` directory. See the `daylit migrate` command for more details.

```sql
-- Example Schema Structure

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    name TEXT,
    kind TEXT,
    duration_min INTEGER,
    -- ... other fields like priority, energy_band
);

CREATE TABLE plans (
    date TEXT PRIMARY KEY -- YYYY-MM-DD
);

CREATE TABLE slots (
    id INTEGER PRIMARY KEY, -- SERIAL in Postgres
    plan_date TEXT REFERENCES plans(date),
    start_time TEXT,
    end_time TEXT,
    task_id TEXT REFERENCES tasks(id),
    status TEXT
);

CREATE TABLE habits (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    archived_at TEXT
);

CREATE TABLE ot_entries (
    id TEXT PRIMARY KEY,
    day TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL
);
```
