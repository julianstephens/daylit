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
  - While currently informational, future versions will use this data to adjust task durations and priorities automatically.

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