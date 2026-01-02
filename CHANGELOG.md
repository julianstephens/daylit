## v0.5.0

- Adds automatic feedback adjustment
- Adds One Thing (OT) integration to TUI
- Adds OS Keyring support for database credentials
- Adds arbitrary scheduled notifications
- Restricts conflict detection visibility to active plan scope
- Refactors storage layer and cleans up TUI

## v0.4.0

- Adds PostgreSQL storage backend for shared access
- Adds `--source` flag to `daylit init` for database migration
- Adds Habits and Once-Today (OT) intention tracking
- Adds TUI support for Settings, Habits, and OT tasks
- Adds secret-based authentication for tray notifications
- Adds stateful notification tracking
- Adds `DAYLIT_CONFIG` environment variable for unified configuration
- Adds support for DSN format connection strings
- Enforces security check for embedded credentials in CLI flags
- Restructures documentation with dedicated user guides
- Removes deprecated JSON storage support

## v0.3.0

- Adds `daylit backup` for database backups and restoration
- Adds `daylit migrate` for schema version tracking and migrations
- Adds `daylit doctor` and `daylit debug` for diagnostics
- Adds `daylit validate` for conflict detection
- Adds `--force` flag to `daylit init` for storage reset
- Adds soft delete and restore functionality for tasks and plans
- Adds plan revisioning and immutability
- Updates TUI to surface new safety features

## v0.2.0

- Adds SQLite storage option (JSON storage still supported)
- Adds `daylit task edit`
- Adds `daylit task delete`
- Adds all CLI commands to TUI

## v0.1.0

- Adds `daylit` CLI