# daylit-cli

A daily structure scheduler and time-blocking companion CLI tool.

## Overview

`daylit` helps you structure your day by:

- Managing task templates (recurring and one-off)
- Generating daily time-blocked schedules
- Tracking what you should be doing now
- Collecting feedback to improve future plans

## Storage Backends

daylit supports multiple storage backends:

- **SQLite** (default): Local file-based database, perfect for single-user desktop use
- **PostgreSQL**: Centralized database server, ideal for concurrent access (e.g., WSL2 + Windows) or multi-device setups

See [PostgreSQL Setup Guide](docs/POSTGRES_SETUP.md) for details on using PostgreSQL.

## Documentation

- [Installation](docs/INSTALLATION.md)
- [Quick Start](docs/QUICK_START.md)
- [CLI Reference](docs/CLI.md)
- [Configuration](docs/CONFIGURATION.md)
- [PostgreSQL Setup](docs/POSTGRES_SETUP.md)
- [Concepts (Scheduling & Data Model)](docs/CONCEPTS.md)
- [Development](docs/DEVELOPMENT.md)
- [Roadmap](docs/ROADMAP.md)
