# Configuration

The default configuration file is located at `~/.config/daylit/daylit.db`.

You can specify a different location using the `--config` flag:

```bash
daylit --config /path/to/config.db init
```

## PostgreSQL Backend

daylit also supports PostgreSQL as a storage backend. For security reasons, connection strings with embedded passwords are **NOT ALLOWED**.

**Secure usage:**
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

The application uses SQLite for storage. The database contains tables for:

- `schema_version`: Schema version tracking for migrations
- `tasks`: Task templates
- `plans`: Daily plans
- `slots`: Time slots within plans
- `settings`: Application settings

The schema is managed through migrations stored in the `migrations/` directory. See the `daylit migrate` command for more details.

```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY
);

CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    duration_min INTEGER NOT NULL,
    -- ... other fields
);
```
