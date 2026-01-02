# Quick Start

## Initialization

Initialize the application storage (SQLite by default):

```bash
daylit init
```

For PostgreSQL setup, see [POSTGRES_SETUP.md](../daylit-cli/docs/POSTGRES_SETUP.md).

## Basic Usage

```bash
# Add some tasks
daylit task add "Morning prayer" --duration 30 --recurrence daily --earliest 07:00 --latest 09:00

daylit task add "Deep work" --duration 90 --recurrence n_days --interval 2 --earliest 09:00 --latest 13:00 --priority 1

daylit task add "Gym" --duration 60 --recurrence weekly --weekdays mon,wed,fri --earliest 14:00 --latest 18:00

# Add an appointment
daylit task add "Team meeting" --duration 60 --fixed-start 10:00 --fixed-end 11:00

# Generate today's plan
daylit plan today

# Check what you should be doing now
daylit now

# Give feedback on completed tasks
daylit feedback --rating on_track --note "Went well"

# View today's full plan
daylit day today

# Track Habits
daylit habit add "Read 30 mins"
daylit habit mark "Read 30 mins"
daylit habit today

# Set your "One Thing" (OT) for the day
daylit ot set "Finish the proposal"
daylit ot show

# Launch the interactive TUI (Terminal User Interface)
daylit tui
# Or simply:
daylit
```

## Settings Management

You can view and modify application settings directly from the CLI:

```bash
# List all current settings
daylit settings --list

# Enable notifications
daylit settings --notifications-enabled=true

# Configure notification timing (e.g., 5 minutes before block start)
daylit settings --block-start-offset-min=5

# Configure OT (One Thing) behavior
daylit settings --ot-prompt-on-empty=true --ot-strict-mode=true
```

## Backups

The application automatically creates backups of your SQLite database. You can also manage them manually:

```bash
# Create a manual backup
daylit backup create

# List available backups
daylit backup list

# Restore from a backup (interactive)
daylit backup restore
```

> **Note:** Automatic backups are only supported for SQLite. If you are using PostgreSQL, you must configure your own backup strategy (e.g., using `pg_dump`).

## Troubleshooting

If you encounter issues, use the built-in diagnostic tools:

```bash
# Check system health and configuration
daylit doctor

# View debug information (schema version, settings, etc.)
daylit debug info

# Validate data integrity (check for conflicts or orphans)
daylit validate
```

For more detailed information on debugging and troubleshooting, see the [Debugging Guide](DEBUGGING.md).

## Advanced: Migration

To migrate existing data from SQLite to PostgreSQL, use the `--source` flag during initialization:

```bash
# Initialize Postgres with data from SQLite
daylit --config "postgres://user@localhost:5432/daylit?sslmode=disable" init --source ~/.config/daylit/daylit.db
```

To apply schema updates to an existing database:

```bash
daylit migrate
```
