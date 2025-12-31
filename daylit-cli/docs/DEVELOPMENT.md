# Development

## Project Structure

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

Run the end-to-end tests:

```bash
# Run basic functionality test
go test ./...

# Or run manual tests
./daylit init
./daylit task add "Test" --duration 30 --recurrence daily
./daylit plan today
```
