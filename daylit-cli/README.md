# daylit-cli (Developer Guide)

This document contains technical details for developers working on the `daylit-cli` component. For user documentation, see the [root README](../README.md) and the [`docs/` directory](../docs/).

## Development Setup

### Prerequisites

- **Go 1.25+**
- **Make** (optional, for using the Makefile)
- **PostgreSQL** (optional, for integration testing)

### Setup

1. Clone the repository.
2. Navigate to `daylit-cli`.
3. Install dependencies:
   ```bash
   go mod download
   ```

## Project Structure

See the [architecture doc](docs/ARCHITECTURE.md) for up-to-date package descriptions.

```
daylit/
├── cmd/
│   └── daylit/
│       └── main.go           # CLI interface using kong
├── internal/
│   ├── backup/
│   │   ├── backup.go          # Backup management and operations
│   │   ├── backup_test.go     # Unit tests for backup
│   │   └── integration_test.go # Integration tests for backup
│   ├── cli/
│   │   ├── backup.go          # Backup CLI commands
│   │   ├── plan.go            # Plan CLI commands
│   │   └── ...                # Other CLI commands
│   ├── models/
│   │   ├── task.go            # Task data models
│   │   └── plan.go            # Plan and slot models
│   ├── scheduler/
│   │   └── scheduler.go       # Scheduling algorithm
│   └── storage/
│       ├── interface.go       # Storage interface
│       └── sqlite_store.go    # SQLite storage implementation
├── go.mod
└── go.sum
```

## Building

```bash
go build -o daylit ./cmd/daylit
```

## Testing

### Unit Tests

Run standard Go tests:

```bash
go test ./...
```

### Integration Tests

To run integration tests (including PostgreSQL tests), set the environment variable:

```bash
export POSTGRES_TEST_URL="postgres://user:pass@localhost:5432/daylit_test?sslmode=disable"
go test ./internal/storage/... -v
```
