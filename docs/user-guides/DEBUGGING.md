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
