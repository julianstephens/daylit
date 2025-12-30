# Soft Delete Feature

## Overview
The soft delete feature allows you to delete tasks and plans without permanently removing them from the database. Deleted items can be restored at any time, providing protection against accidental deletions.

## Commands

### Restore a Task
```bash
daylit restore task <task-id>
```

### Restore a Plan
```bash
daylit restore plan <date>
```

## Behavior
- Deleted tasks don't appear in listings or the TUI
- Deleted tasks won't be scheduled in daily plans
- All data is preserved and can be restored
- Foreign key relationships remain intact

## Technical Details
Soft delete is implemented via `deleted_at` timestamp columns on tasks, plans, and slots tables.
