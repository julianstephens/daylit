# daylit

A daily structure scheduler and time-blocking companion CLI tool.

## Overview

`daylit` helps you structure your day by:
- Managing task templates (recurring and one-off)
- Generating daily time-blocked schedules
- Tracking what you should be doing now
- Collecting feedback to improve future plans


## Installation

### Prerequisites

- Go 1.25 or later

### Build from source

```bash
git clone https://github.com/julianstephens/daylit.git
cd daylit
go build -o daylit ./cmd/daylit
```

Optionally, move the binary to your PATH:

```bash
sudo mv daylit /usr/local/bin/
```

## Quick Start

```bash
# Initialize daylit
daylit init

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
```

## Commands

### `daylit init`

Initialize the configuration and storage files.

```bash
daylit init
```

By default, stores data in `~/.config/daylit/daylit.db`. Use `--config` to specify a different location.

### `daylit tui`

Launch the interactive Text User Interface (TUI).

```bash
daylit tui
# or simply
daylit
```

The TUI provides a dashboard with three main views:

1.  **Now**: Shows the current task and time.
2.  **Plan**: Displays today's schedule. Press `g` to generate a plan if one doesn't exist.
3.  **Tasks**: Lists all your tasks.

**Key Bindings:**

-   `Tab` / `Shift+Tab`: Switch between tabs.
-   `h` / `l`: Switch between tabs (Vim style).
-   `j` / `k`: Navigate up/down in lists.
-   `g`: Generate plan (in Plan tab).
-   `a`: Add task (in Tasks tab).
-   `e`: Edit task (in Tasks tab).
-   `d`: Delete task (in Tasks tab).
-   `f`: Give feedback on last task.
-   `?`: Toggle help.
-   `q` / `Ctrl+C`: Quit.

### `daylit task add`

Add a new task template.

```bash
daylit task add "Task name" [flags]
```

**Flags:**

- `--duration INT` (required): Duration in minutes
- `--recurrence STRING`: Recurrence type: `daily`, `weekly`, `n_days`, or `ad_hoc` (default: `ad_hoc`)
- `--interval INT`: For `n_days` recurrence, the number of days between occurrences (default: 1)
- `--weekdays STRING`: For `weekly` recurrence, comma-separated weekdays (e.g., `mon,wed,fri`)
- `--earliest TIME`: Earliest start time in HH:MM format
- `--latest TIME`: Latest end time in HH:MM format
- `--fixed-start TIME`: For appointments, fixed start time in HH:MM
- `--fixed-end TIME`: For appointments, fixed end time in HH:MM
- `--priority INT`: Priority level, 1-5 (lower number = higher priority, default: 3)

**Examples:**

```bash
# Daily task with time window
daylit task add "Morning routine" --duration 30 --recurrence daily --earliest 06:00 --latest 08:00

# Task every 3 days
daylit task add "Laundry" --duration 45 --recurrence n_days --interval 3

# Weekly task on specific days
daylit task add "Gym" --duration 60 --recurrence weekly --weekdays mon,wed,fri --earliest 17:00 --latest 19:00

# Fixed appointment
daylit task add "Doctor appointment" --duration 60 --fixed-start 14:00 --fixed-end 15:00
```

### `daylit task edit`

Edit an existing task template.

```bash
daylit task edit <TASK_ID> [flags]
```

To find the Task ID, use `daylit task list --show-ids`.

**Flags:**

- `--name STRING`: New task name
- `--duration INT`: New duration in minutes
- `--recurrence STRING`: New recurrence type (`daily`, `weekly`, `n_days`, `ad_hoc`)
- `--interval INT`: New interval for `n_days` recurrence
- `--weekdays STRING`: New comma-separated weekdays for `weekly` recurrence
- `--earliest TIME`: New earliest start time (HH:MM)
- `--latest TIME`: New latest end time (HH:MM)
- `--fixed-start TIME`: New fixed start time (HH:MM)
- `--fixed-end TIME`: New fixed end time (HH:MM)
- `--priority INT`: New priority (1-5)
- `--active BOOL`: Set active status (true/false)

**Example:**

```bash
# Find the task ID
daylit task list --show-ids

# Edit the task
daylit task edit 81462541-e5ef-400b-9a8e-de96de1a9574 --name "Updated Task" --duration 45
```

### `daylit task delete`

Delete a task template.

```bash
daylit task delete <TASK_ID>
```

**Example:**

```bash
daylit task delete 81462541-e5ef-400b-9a8e-de96de1a9574
```

### `daylit task list`

List all task templates.

```bash
daylit task list [flags]
```

**Flags:**

- `--active-only`: Show only active tasks
- `--show-ids`: Show task IDs (useful for editing)

### `daylit plan`

Generate a time-blocked plan for a specific day.

```bash
daylit plan [date]
```

**Arguments:**

- `date`: Date to plan, either `today` or in `YYYY-MM-DD` format (default: `today`)

The command will:
1. Show the proposed plan
2. Ask if you want to accept it
3. If accepted, save the plan as committed

**Example:**

```bash
# Plan for today
daylit plan today

# Plan for a specific date
daylit plan 2025-01-15
```

### `daylit now`

Show what you should be doing at the current time.

```bash
daylit now
```

### `daylit feedback`

Provide feedback on the most recent completed task.

```bash
daylit feedback --rating RATING [flags]
```

**Flags:**

- `--rating STRING` (required): Rating for the task: `on_track`, `too_much`, or `unnecessary`
- `--note STRING`: Optional note about the task

The feedback helps `daylit` adjust future plans:
- `on_track`: Task duration was appropriate
- `too_much`: Task was too ambitious, will reduce duration in future
- `unnecessary`: Task wasn't needed, will reduce frequency

**Example:**

```bash
daylit feedback --rating on_track
daylit feedback --rating too_much --note "Only needed 20 minutes, not 50"
daylit feedback --rating unnecessary --note "Skip this on Mondays"
```

### `daylit day`

Show the full plan for a specific day, including any feedback.

```bash
daylit day [date]
```

**Arguments:**

- `date`: Date to show, either `today` or in `YYYY-MM-DD` format (default: `today`)

**Example:**

```bash
# Show today's plan
daylit day today

# Show a specific day
daylit day 2025-01-15
```

### `daylit backup`

Manage database backups. The application automatically creates backups on startup (TUI) and when generating plans.

#### `daylit backup` or `daylit backup create`

Create a manual backup of the database.

```bash
daylit backup
# or explicitly
daylit backup create
```

Backups are stored in `~/.config/daylit/backups/` with the format `daylit-YYYYMMDD-HHMM.db`.

#### `daylit backup list`

List all available backups.

```bash
daylit backup list
```

Shows:
- Timestamp of each backup
- Filename
- File size
- Total number of backups (retains 14 most recent)

#### `daylit backup restore`

Restore the database from a backup file.

```bash
daylit backup restore <backup-file>
```

**Arguments:**

- `backup-file`: Path or filename of the backup to restore. Can be:
  - Just the filename (e.g., `daylit-20250130-1230.db`) - will look in the backup directory
  - Full path to a backup file

**Safety:**
- Prompts for confirmation before restoring
- Automatically creates a backup of the current database before restoring
- Verifies backup file integrity before restoring

**Example:**

```bash
# List available backups
daylit backup list

# Restore from a specific backup
daylit backup restore daylit-20250130-1230.db

# Restore from a backup at a specific path
daylit backup restore /path/to/backup/daylit-20250130-1230.db
```

**Automatic Backups:**

The application automatically creates backups:
- When launching the TUI (`daylit tui` or `daylit`)
- When generating a plan (`daylit plan`)
- Before restoring from a backup

Backup retention:
- Keeps the 14 most recent backups
- Automatically deletes older backups

## Configuration

The default configuration file is located at `~/.config/daylit/daylit.db`.

You can specify a different location using the `--config` flag:

```bash
daylit --config /path/to/config.db init
```

### Database Schema

The application uses SQLite for storage. The database contains tables for:

- `tasks`: Task templates
- `plans`: Daily plans
- `slots`: Time slots within plans
- `settings`: Application settings

```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    duration_min INTEGER NOT NULL,
    -- ... other fields
);
```

## Scheduling Algorithm

The scheduler uses a simple, deterministic algorithm:

1. **Place fixed appointments** first (tasks with `--fixed-start` and `--fixed-end`)
2. **Filter flexible tasks** based on recurrence rules:
   - Daily tasks are always candidates
   - Weekly tasks match against the day of the week
   - N-days tasks check if enough days have passed since last completion
3. **Sort by priority** (lower number = higher priority) and lateness
4. **Pack tasks** into free blocks, respecting time constraints

The algorithm prioritizes predictability over optimization.

## Data Model

### Task Template

Each task has:
- Unique ID
- Name
- Kind: `appointment` (fixed time) or `flexible`
- Duration in minutes
- Recurrence: `daily`, `weekly`, `n_days`, or `ad_hoc`
- Optional time constraints (earliest/latest)
- Priority (1-5)
- Feedback statistics (last done, average duration)

### Day Plan

For each date:
- List of time slots
- Each slot has:
  - Start and end time
  - Task ID
  - Status: `planned`, `accepted`, `done`, or `skipped`
  - Optional feedback with rating and note

## Development

### Project Structure

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

### Building

```bash
go build -o daylit ./cmd/daylit
```

### Testing

Run the end-to-end tests:

```bash
# Run basic functionality test
go test ./...

# Or run manual tests
./daylit init
./daylit task add "Test" --duration 30 --recurrence daily
./daylit plan today
```

## Roadmap

Future enhancements (v0.2+):

- [ ] Natural language parsing for task creation
- [ ] Notification daemon
- [ ] Energy level tracking
- [ ] Task dependencies
- [ ] Weekly/monthly planning views
- [ ] Habit tracking integration
