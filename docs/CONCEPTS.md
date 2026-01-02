# Concepts

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
- Soft delete status (`deleted_at`)

### Day Plan

For each date:

- Revision number (tracks changes to the plan)
- List of time slots
- Each slot has:
  - Start and end time
  - Task ID
  - Status: `planned`, `accepted`, `done`, or `skipped`
  - Optional feedback with rating and note
- Soft delete status (`deleted_at`)

### Feedback Loop

The system learns from user feedback to improve future plans.

- **Ratings**:
  - `on_track`: The task was completed as planned.
  - `too_much`: The task took longer than expected or was overwhelming.
  - `unnecessary`: The task wasn't valuable or could have been skipped.
- **Impact**:
  - Feedback is stored on the specific slot in the day plan.
  - **Immediate adjustments**: The `feedback` command applies instant adjustments to task properties when feedback is given:
    - `on_track`: Updates the task's average actual duration using exponential moving average
    - `too_much`: Reduces task duration by 10% (minimum 10 minutes)
    - `unnecessary`: Increases recurrence interval for n-days tasks
  - **Historical analysis**: The `optimize` command analyzes accumulated feedback patterns to suggest more substantial optimizations:
    - Tasks with consistent `too_much` feedback (>50%) receive suggestions to reduce duration by 25% or split into smaller tasks
    - Tasks with frequent `unnecessary` feedback (≥3 instances or >40%) receive suggestions to reduce frequency or remove the task
    - Optimizations can be reviewed in dry-run mode, applied selectively in interactive mode, or auto-applied

### Habit

A recurring practice to track (boolean completion):

- Unique ID
- Name
- Creation timestamp
- Archive status (`archived_at`)
- Soft delete status (`deleted_at`)
- Daily entries (`habit_entries`) linking a habit to a date with an optional note

### One Thing (OT)

A single, primary intention for the day:

- Unique ID
- Date
- Title (the intention)
- Optional note
- Soft delete status (`deleted_at`)

## Soft Delete

The soft delete feature allows you to delete tasks and plans without permanently removing them from the database. Deleted items can be restored at any time, providing protection against accidental deletions.

### Commands

#### Restore a Task

```bash
daylit restore task <task-id>
```

#### Restore a Plan

```bash
daylit restore plan <date>
```

### Behavior

- Deleted tasks are hidden from listings and the TUI (not shown, not visually indicated as deleted)
- Deleted tasks won't be scheduled in daily plans
- All data is preserved and can be restored
- Foreign key relationships remain intact

### Technical Details

Soft delete is implemented via `deleted_at` timestamp columns on tasks, plans, and slots tables.

## Timezone Handling

Daylit supports timezone-aware scheduling and time tracking to ensure consistent behavior when traveling or working across different timezones.

### Configuration

Users can configure their timezone preference through:

- The TUI Settings tab (press 'e' to edit, navigate to the Timezone field)
- The CLI: `daylit settings --timezone="America/New_York"`

The timezone can be set to:

- `Local` (default): Uses the system's local timezone
- Any valid IANA timezone name (e.g., `America/New_York`, `Europe/London`, `Asia/Tokyo`, `UTC`)

### How It Works

The timezone configuration provides infrastructure for future timezone-aware operations:

1. **Time Storage**: Times are stored as strings (`HH:MM` for time-of-day, `YYYY-MM-DD` for dates) which are timezone-independent
2. **Timestamps**: Full timestamps (like `created_at`, `last_sent`) are stored in RFC3339 format which includes timezone information
3. **Current Behavior**: The application currently uses the system's local timezone for all date/time operations via `time.Now()`
4. **Future Integration**: Utility functions have been created (`GetTodayInTimezone`, `ParseDateInLocation`, etc.) to enable timezone-aware scheduling in future updates

### Planned Use Cases

Once fully integrated, the timezone setting will enable:

- **Traveling**: Update your timezone setting to match your current location for accurate date determination
- **Remote Work**: Set your timezone to match your team's timezone for consistent scheduling
- **Consistency**: Explicitly set your timezone to ensure consistent behavior regardless of system timezone changes

### Current Status

The timezone setting is **stored and configurable** but not yet integrated into the application logic. The infrastructure is in place with utility functions ready for integration:

- `GetTodayInTimezone(timezone)` - determines "today" in the configured timezone
- `NowInTimezone(timezone)` - gets current time in the configured timezone
- `ParseDateInLocation()`, `CombineDateAndTime()` - timezone-aware parsing functions

Future updates will integrate these functions throughout the codebase to replace direct `time.Now()` calls.
