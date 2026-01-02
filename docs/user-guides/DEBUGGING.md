# Debugging Guide

This guide explains how to troubleshoot issues with the Daylit CLI using the built-in debugging tools and logging features.

## Enabling Debug Logs

By default, Daylit runs silently to keep your terminal clean. To see detailed execution logs, you can use the global `--debug` flag with any command.

### Usage

Add `--debug` to any command to enable verbose logging:

```bash
# Debug a specific command
daylit plan --debug

# Debug task creation
daylit task add "My Task" --duration 30 --debug

# Debug the TUI (logs will be written to file to avoid UI corruption)
daylit tui --debug
```

When `--debug` is enabled, logs are printed to `stderr` (unless in TUI mode) and also persisted to the log file.

## The `daylit doctor` Command

The `daylit doctor` command runs a suite of health checks and diagnostics to identify common issues with your installation, configuration, and data integrity.

### Usage

```bash
daylit doctor
```

### Checks Performed

The command performs the following checks:

-   **Database Reachability**: Verifies that the application can connect to the configured database (SQLite or PostgreSQL).
-   **Schema Version**: Checks if the database schema version is valid and compatible with the current CLI version.
-   **Migrations Complete**: Ensures all database migrations have been successfully applied.
-   **Backups Present**: Checks for the existence of recent backups (warns if none are found).
-   **Data Validation**: Scans tasks and plans for logical inconsistencies.
-   **Clock/Timezone**: Verifies system clock and timezone settings to prevent scheduling errors.
-   **Habit Integrity**: Checks for consistency in habit definitions and entries.

If any check fails, `daylit doctor` will provide specific error messages to help you resolve the issue.

## The `daylit validate` Command

The `daylit validate` command performs logical checks on your tasks and plans to identify scheduling conflicts, data inconsistencies, and potential issues.

### Usage

```bash
# Run validation checks
daylit validate

# Automatically fix simple issues (like duplicate tasks)
daylit validate --fix
```

### Checks Performed

The command checks for the following issues:

-   **Duplicate Task Names**: Identifies multiple active tasks with the exact same name.
-   **Overlapping Fixed Tasks**: Detects tasks with fixed start/end times that overlap with each other.
-   **Overlapping Slots**: Checks today's plan for time slots that overlap.
-   **Waking Window Violations**: Identifies tasks scheduled outside your configured day start/end times.
-   **Overcommitment**: Warns if the total duration of tasks exceeds the available time in the day.
-   **Orphaned Slots**: Detects slots in the plan that reference non-existent tasks.

### Auto-Fixing

The `--fix` flag can automatically resolve certain types of conflicts, such as:
-   **Duplicate Tasks**: Merges or removes duplicate task definitions to ensure uniqueness.

## The `daylit debug` Command

The `daylit debug` command provides specific tools for inspecting the internal state of the application without modifying it.

### Available Commands

-   **`daylit debug db-path`**: Shows the current database file path.
-   **`daylit debug dump-plan [date]`**: Dumps the raw JSON structure of a day plan.
    ```bash
    daylit debug dump-plan today
    daylit debug dump-plan 2024-01-01
    ```
-   **`daylit debug dump-task <id>`**: Dumps the raw JSON structure of a specific task.
    ```bash
    daylit debug dump-task task_123abc
    ```
-   **`daylit debug dump-habit <id>`**: Dumps the raw JSON structure of a specific habit.
    ```bash
    daylit debug dump-habit habit_456def
    ```
-   **`daylit debug dump-ot [date]`**: Dumps the raw JSON structure of an OT entry for a specific day.
    ```bash
    daylit debug dump-ot today
    daylit debug dump-ot 2024-01-01
    ```
-   **`daylit debug dump-alert <id>`**: Dumps the raw JSON structure of a specific alert.
    ```bash
    daylit debug dump-alert alert_789ghi
    ```
-   **`daylit debug dump-settings`**: Dumps the raw JSON structure of application settings.
    ```bash
    daylit debug dump-settings
    ```

*Note: The `daylit debug` command automatically enables debug logging.*

## Log Files

Daylit persists logs to a rotating log file in your configuration directory. This is useful for reviewing what happened during a TUI session or tracking down intermittent issues.

### Log Location

Logs are stored in the `logs` subdirectory of your Daylit configuration folder:

-   **Linux/macOS**: `~/.config/daylit/logs/daylit.log`
-   **Windows**: `%APPDATA%\daylit\logs\daylit.log`

### Log Rotation

To prevent log files from consuming too much disk space, Daylit automatically rotates logs using the following policy:
- Maximum log file size: 10 MB
- Maximum backup files kept: 3
- Maximum age of backup files: 28 days
- Old backups are automatically compressed

This ensures that logs are preserved for troubleshooting while preventing unbounded disk usage.

## Reporting Issues

When reporting an issue on GitHub, it is helpful to include:

1.  The command you ran.
2.  The output with the `--debug` flag enabled.
3.  Relevant sections from the `daylit.log` file.

```bash
# Example: Capturing debug output to a file for a report
daylit plan --debug 2> debug_output.txt
```
