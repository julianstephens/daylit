# Commands

## `daylit init`

Initialize the configuration and storage files.

```bash
daylit init
```

By default, stores data in `~/.config/daylit/daylit.db`. Use `--config` to specify a different location.

## `daylit tui`

Launch the interactive Text User Interface (TUI).

```bash
daylit tui
# or simply
daylit
```

The TUI provides a dashboard with seven main views:

1.  **Now**: Shows the current task and time.
2.  **Plan**: Displays today's schedule. Press `g` to generate a plan if one doesn't exist.
3.  **Tasks**: Lists all your tasks.
4.  **Habits**: View and manage your daily habits.
5.  **OT**: View and manage Once-Today intentions.
6.  **Alerts**: View and manage scheduled notifications.
7.  **Settings**: View and edit application settings.

**Key Bindings:**

- `Tab` / `Shift+Tab`: Switch between tabs.
- `h` / `l`: Switch between tabs (Vim style).
- `j` / `k`: Navigate up/down in lists.
- `g`: Generate plan (in Plan tab).
- `a`: Add task (in Tasks tab), habit (in Habits tab), or alert (in Alerts tab).
- `e`: Edit task (in Tasks tab), OT (in OT tab), or settings (in Settings tab).
- `d`: Delete task (in Tasks tab), habit (in Habits tab), or alert (in Alerts tab).
- `m`: Mark habit as done (in Habits tab).
- `u`: Unmark habit (in Habits tab).
- `x`: Archive habit (in Habits tab).
- `r`: Restore deleted task/habit.
- `f`: Give feedback on last task.
- `?`: Toggle help.
- `q` / `Ctrl+C`: Quit.

## `daylit task`

Manage tasks and task templates.

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

Delete a task template. This performs a "soft delete", meaning the task is hidden but can be restored later using `daylit restore task`.

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

## `daylit plan`

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

## `daylit plans delete`

Delete a daily plan. This performs a "soft delete", meaning the plan is hidden but can be restored later using `daylit restore plan`.

```bash
daylit plans delete <date>
```

**Arguments:**

- `date`: Date of the plan to delete (YYYY-MM-DD)

**Example:**

```bash
daylit plans delete 2025-01-15
```

## `daylit restore`

Restore soft-deleted items.

### `daylit restore task`

Restore a deleted task template.

```bash
daylit restore task <TASK_ID>
```

### `daylit restore plan`

Restore a deleted daily plan.

```bash
daylit restore plan <date>
```

**Example:**

```bash
daylit restore task 81462541-e5ef-400b-9a8e-de96de1a9574
daylit restore plan 2025-01-15
```

## `daylit now`

Show what you should be doing at the current time.

```bash
daylit now
```

## `daylit feedback`

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

## `daylit optimize`

Analyze feedback history and suggest task optimizations. This command uses accumulated feedback data to identify tasks that may need adjustment.

```bash
daylit optimize [flags]
```

**Flags:**

- `--interactive`: Interactively review and apply optimizations one by one
- `--auto-apply`: Automatically apply all optimizations without confirmation
- `--feedback-limit INT`: Number of recent feedback entries to analyze per task (default: 10)

**How it works:**

The optimizer analyzes feedback patterns:

- **Too much feedback (>50%)**: Suggests reducing duration by 25% for longer tasks, or splitting shorter tasks (30 minutes or less)
- **Unnecessary feedback (≥3 instances or >40%)**: Suggests reducing frequency or removing the task
- **Mixed feedback**: No optimization suggested; task is performing acceptably

**Modes:**

1. **Dry-run mode** (default):
   - Shows suggestions without making any changes
   - Useful for reviewing what could be optimized

2. **Interactive mode** (`--interactive`):
   - Reviews each suggestion one by one
   - Prompts to apply, skip, or skip all remaining
   - Provides full control over which optimizations to apply

3. **Auto-apply mode** (`--auto-apply`):
   - Applies all optimizations automatically
   - No confirmation required
   - Shows summary of applied optimizations

**Examples:**

```bash
# Review optimization suggestions (dry-run mode)
daylit optimize

# Review and selectively apply optimizations
daylit optimize --interactive

# Automatically apply all optimizations
daylit optimize --auto-apply

# Analyze only the last 5 feedback entries per task
daylit optimize --feedback-limit 5 --interactive
```

**Example output:**

```
Analyzing task feedback history...

📊 Found 2 optimization suggestion(s):

1. ⏱️  Reduce Duration
   Task: Morning Exercise
   Reason: 75% of recent feedback indicates task takes too long (too_much)
   Current: duration_min=60
   Suggested: duration_min=45

2. 📉 Reduce Frequency
   Task: Email Cleanup
   Reason: 60% of recent feedback indicates task is unnecessary
   Current: interval_days=1
   Suggested: interval_days=3

💡 To apply these optimizations:
  - Use --interactive to review and select which to apply
  - Use --auto-apply to apply all automatically
```

**Note:** Task splitting suggestions require manual action, as they cannot be automatically applied.

## `daylit day`

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

## `daylit backup`

Manage database backups. The application automatically creates backups on startup (TUI) and when generating plans.

### `daylit backup` or `daylit backup create`

Create a manual backup of the database.

```bash
daylit backup
# or explicitly
daylit backup create
```

Backups are stored in `~/.config/daylit/backups/` with the format `daylit-YYYYMMDD-HHMM.db`.

### `daylit backup list`

List all available backups.

```bash
daylit backup list
```

Shows:

- Timestamp of each backup
- Filename
- File size
- Total number of backups (retains 14 most recent)

### `daylit backup restore`

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

## `daylit migrate`

Run database schema migrations explicitly. This command applies any pending migrations to bring the database schema up to date.

```bash
daylit migrate
```

**How Migrations Work:**

- Migrations are stored in the `migrations/` directory as numbered SQL files (e.g., `001_init.sql`, `002_add_feature.sql`)
- Each migration is applied in a transaction - if any part fails, the entire migration is rolled back
- The current schema version is tracked in the `schema_version` table
- Migrations are automatically applied on `daylit init`
- The application checks schema version on startup and refuses to run if the database is newer than supported

**Schema Version Safety:**

If you try to use a database with a newer schema version than your application supports, you'll see an error:

```
Error: database schema version (5) is newer than supported version (3) - please upgrade the application
```

This prevents data corruption from using an older version of the application with a newer database.

**Example:**

```bash
# Explicitly run migrations
daylit migrate

# Output when up to date:
Database schema is up to date (version 1)
No migrations to apply. Database is up to date.

# Output when migrations are applied:
Current schema version: 1
Target schema version: 2
Applying 1 migration(s)...
  Applying migration 2: add_feature
  ✓ Migration 2 applied successfully
Applied 1 migration(s) in 2.14ms
```

**Migration Path Configuration:**

By default, migrations are loaded from the `migrations/` directory. You can override this using the `DAYLIT_MIGRATIONS_PATH` environment variable:

```bash
export DAYLIT_MIGRATIONS_PATH=/path/to/migrations
daylit migrate
```

## `daylit doctor`

Run health checks and diagnostics on the daylit installation. This command verifies that all systems are functioning correctly.

```bash
daylit doctor
```

**Checks performed:**

1. **Database reachable**: Verifies the database file exists and can be opened
2. **Schema version valid**: Ensures the database schema version matches the application
3. **Migrations complete**: Confirms all pending migrations have been applied
4. **Backups present**: Checks if backups exist (warning only, not an error)
5. **Data validation**: Validates database integrity and checks for data corruption
6. **Clock/timezone sanity**: Verifies system time is reasonable

**Exit codes:**

- `0`: All checks passed (warnings are acceptable)
- `1`: One or more critical checks failed

**Example output:**

```bash
$ daylit doctor
Running diagnostics...

✓ Database reachable: OK
✓ Schema version: OK
✓ Migrations complete: OK
⚠ Backups present: WARNING
   no backups found - consider creating one with 'daylit backup create'
✓ Data validation: OK
✓ Clock/timezone: OK

All diagnostics passed!
```

**When to use:**

- After upgrading daylit to verify compatibility
- When experiencing unexpected behavior
- Before submitting bug reports
- To verify installation health

## `daylit validate`

Validate tasks and plans for conflicts and consistency.

```bash
daylit validate
```

This command checks for:

- Overlapping fixed appointments
- Invalid time ranges
- Logical inconsistencies in task definitions
- Conflicts in the current day's plan

**Example:**

```bash
daylit validate
```

## `daylit debug`

Debug commands for troubleshooting and inspecting internals. These commands output machine-readable JSON for scripting and analysis.

### `daylit debug db-path`

Show the database file path.

```bash
daylit debug db-path
```

**Output:**

```json
{
  "path": "/home/user/.config/daylit/daylit.db"
}
```

### `daylit debug dump-plan`

Dump plan data as JSON for a specific date.

```bash
daylit debug dump-plan <date>
```

**Arguments:**

- `date`: Date in `YYYY-MM-DD` format, or `today`

**Example:**

```bash
# Dump today's plan
daylit debug dump-plan today

# Dump a specific date's plan
daylit debug dump-plan 2025-01-15
```

**Output:**

```json
{
  "date": "2025-01-15",
  "revision": 1,
  "accepted_at": "2025-01-15T08:30:00Z",
  "slots": [
    {
      "start": "07:00",
      "end": "07:45",
      "task_id": "abc-123",
      "status": "accepted"
    }
  ]
}
```

**Error handling:**

- Returns non-zero exit code if plan doesn't exist
- Validates date format before querying

### `daylit debug dump-task`

Dump task data as JSON for a specific task ID.

```bash
daylit debug dump-task <id>
```

**Arguments:**

- `id`: Task UUID

**Example:**

```bash
daylit debug dump-task abc-123-def-456
```

**Output:**

```json
{
  "id": "abc-123-def-456",
  "name": "Morning Exercise",
  "kind": "flexible",
  "duration_min": 45,
  "earliest_start": "06:00",
  "latest_end": "08:00",
  "recurrence": {
    "type": "daily",
    "interval_days": 1
  },
  "priority": 1,
  "active": true,
  "success_streak": 5,
  "avg_actual_duration_min": 42
}
```

**Error handling:**

- Returns non-zero exit code if task doesn't exist
- Provides clear error messages for invalid IDs

**Use cases for debug commands:**

- Inspecting plan structure for debugging
- Exporting data for analysis or backup
- Scripting and automation
- Troubleshooting scheduling issues

## `daylit habit`

Manage habits and track daily practices. Habits represent recurring practices like meditation, exercise, or reading - activities you want to show up for each day, without gamification or streak tracking.

### `daylit habit add`

Add a new habit to track.

```bash
daylit habit add <name>
```

**Arguments:**

- `name`: Name of the habit (e.g., "Morning meditation", "Evening reading")

**Example:**

```bash
daylit habit add "Morning meditation"
daylit habit add "Daily exercise"
daylit habit add "Reading before bed"
```

### `daylit habit list`

List all habits.

```bash
daylit habit list [flags]
```

**Flags:**

- `--archived`: Include archived habits in the list
- `--deleted`: Include soft-deleted habits in the list

By default, only shows active (non-archived, non-deleted) habits.

**Example:**

```bash
# List active habits
daylit habit list

# List all habits including archived
daylit habit list --archived

# List all habits including deleted
daylit habit list --deleted
```

### `daylit habit mark`

Mark a habit as completed for a specific day. Uses toggle semantics - marking twice will unmark (soft-delete the entry).

```bash
daylit habit mark <name> [flags]
```

**Arguments:**

- `name`: Name of the habit to mark

**Flags:**

- `--date DATE`: Date in YYYY-MM-DD format (default: today)
- `--note TEXT`: Optional note about this completion

**Example:**

```bash
# Mark habit for today
daylit habit mark "Morning meditation"

# Mark habit for a specific date
daylit habit mark "Morning meditation" --date 2025-12-30

# Mark with a note
daylit habit mark "Morning meditation" --note "Felt very centered today"

# Mark again to unmark (toggle off)
daylit habit mark "Morning meditation"
```

### `daylit habit today`

Show today's habit status - which habits have been completed and which haven't.

```bash
daylit habit today
```

Shows:

- `[x]` for habits marked today
- `[ ]` for habits not yet marked today
- Summary of recorded habits (e.g., "Recorded: 2/3")

**Example:**

```bash
$ daylit habit today
Habits for 2025-12-31:

[x] Morning meditation
[ ] Daily exercise
[x] Reading before bed

Recorded: 2/3
```

### `daylit habit log`

Display an ASCII log showing habit completion history over time.

```bash
daylit habit log [flags]
```

**Flags:**

- `--days N`: Number of days to show (default: 14)
- `--habit NAME`: Show log for specific habit only

Shows a visual grid where:

- `x` indicates the habit was completed that day
- `.` indicates the habit was not completed that day

**Example:**

```bash
# Show last 14 days for all habits
daylit habit log

# Show last 7 days
daylit habit log --days 7

# Show log for specific habit
daylit habit log --habit "Morning meditation" --days 30
```

**Output example:**

```
Habit log (last 7 days):

Habit                12/25 12/26 12/27 12/28 12/29 12/30 12/31
--------------------------------------------------------------
Morning meditation    x     x     .     x     x     .     x
Daily exercise        .     x     x     .     .     x     x
Reading before bed    x     .     x     x     x     x     .
```

### `daylit habit archive`

Archive a habit. Archived habits are hidden from default views but their entries are preserved. This is useful for habits you've stopped doing but want to keep the history.

```bash
daylit habit archive <name> [flags]
```

**Arguments:**

- `name`: Name of the habit to archive

**Flags:**

- `--unarchive`: Unarchive the habit instead

**Example:**

```bash
# Archive a habit
daylit habit archive "Old habit"

# Unarchive a habit
daylit habit archive "Old habit" --unarchive
```

### `daylit habit delete`

Soft-delete a habit. The habit and its entries are hidden but not permanently removed, allowing restoration later.

```bash
daylit habit delete <name>
```

**Arguments:**

- `name`: Name of the habit to delete

**Example:**

```bash
daylit habit delete "Obsolete habit"
```

### `daylit habit restore`

Restore a soft-deleted habit.

```bash
daylit habit restore <name>
```

**Arguments:**

- `name`: Name of the deleted habit to restore

**Example:**

```bash
daylit habit restore "Obsolete habit"
```

## `daylit alert`

Manage arbitrary scheduled notifications. Alerts let you set up reminders independent of your task schedule, perfect for recurring reminders like "Drink water", "Take medication", or one-time notifications like appointments.

### `daylit alert add`

Add a new alert notification.

```bash
daylit alert add MESSAGE --time TIME [flags]
```

**Arguments:**

- `MESSAGE`: The alert message to display

**Flags:**

- `--time STRING` (required): Time for the alert in HH:MM format
- `--date STRING`: Date for one-time alert in YYYY-MM-DD format
- `--recurrence STRING`: Recurrence type for recurring alerts: `daily`, `weekly`, or `n_days`
- `--interval N`: Interval for n_days recurrence (default: 1)
- `--weekdays STRING`: Comma-separated weekdays for weekly recurrence (e.g., "mon,wed,fri")

**Alert Types:**

1. **One-time alert**: Specify `--date` for a single notification on a specific date
2. **Recurring alert**: Specify `--recurrence` without `--date` for repeated notifications

**Examples:**

```bash
# One-time alert
daylit alert add "Doctor's Appointment" --time 14:30 --date 2026-01-15

# Daily alert
daylit alert add "Drink Water" --time 10:00 --recurrence daily

# Weekly alert on specific days
daylit alert add "Submit Timesheet" --time 16:45 --recurrence weekly --weekdays fri

# Alert every 3 days
daylit alert add "Water plants" --time 09:00 --recurrence n_days --interval 3
```

### `daylit alert list`

List all configured alerts.

```bash
daylit alert list
```

Displays all alerts with their ID, message, time, recurrence pattern, and active status.

**Example:**

```bash
daylit alert list
```

### `daylit alert delete`

Delete an alert by its ID.

```bash
daylit alert delete <id>
```

**Arguments:**

- `id`: The alert ID (shown in `daylit alert list`)

**Example:**

```bash
daylit alert delete 53d25b70-cb40-4e64-ba23-0d2ff25b703d
```

**Notes:**

- Alerts are checked by the `daylit notify` command, which should be run every minute (e.g., via cron)
- Alerts respect the notification grace period setting
- One-time alerts are automatically deactivated after they fire
- Alerts are integrated into the TUI in the "Alerts" tab

## `daylit ot`

Manage Once-Today (OT) intentions. OT defines the single guiding intention that gives each day its shape - the one non-negotiable focus or orientation for the day.

### `daylit ot init`

Initialize OT settings with default values.

```bash
daylit ot init
```

Creates the OT settings with defaults:

- `prompt_on_empty`: true (prompts when no OT set)
- `strict_mode`: true (requires title when setting OT)
- `default_log_days`: 14 (days to show in log view)

**Example:**

```bash
daylit ot init
```

### `daylit ot settings`

View or update OT settings.

```bash
daylit ot settings [flags]
```

**Flags:**

- `--prompt-on-empty BOOL`: Enable/disable prompt when OT is empty
- `--strict-mode BOOL`: Enable/disable strict mode (require title)
- `--default-log-days N`: Set default number of days for log view

When called without flags, displays current settings.

**Example:**

```bash
# View current settings
daylit ot settings

# Update settings
daylit ot settings --prompt-on-empty=false
daylit ot settings --default-log-days=30
daylit ot settings --strict-mode=true
```

**Output:**

```
Current OT settings:
  prompt_on_empty: true
  strict_mode: true
  default_log_days: 14
```

### `daylit ot set`

Set or update the OT intention for a day.

```bash
daylit ot set --title TEXT [flags]
```

**Flags:**

- `--title TEXT` (required): The day's intention or focus
- `--day DATE`: Date in YYYY-MM-DD format (default: today)
- `--note TEXT`: Optional additional note

The intention is fully editable - setting it again on the same day will update it.

**Example:**

```bash
# Set today's OT
daylit ot set --title "Complete the daylit feature"

# Set with a note
daylit ot set --title "Complete the daylit feature" --note "Focus on testing and documentation"

# Set for a specific date
daylit ot set --day 2025-12-30 --title "Prepare for presentation"

# Update today's OT (overwrites previous)
daylit ot set --title "Updated intention for today"
```

### `daylit ot show`

Show OT intention for one or more days.

```bash
daylit ot show [flags]
```

**Flags:**

- `--day DATE`: Specific date in YYYY-MM-DD format (default: today)
- `--days N`: Show last N days instead of single day
- `--deleted`: Include deleted entries in date range

**Example:**

```bash
# Show today's OT
daylit ot show

# Show specific day
daylit ot show --day 2025-12-30

# Show last 7 days
daylit ot show --days 7

# Show last 14 days including deleted
daylit ot show --days 14 --deleted
```

**Output:**

```
OT for 2025-12-31:
  Complete the daylit feature
  Note: Focus on testing and documentation
```

### `daylit ot nudge`

Quick check for today's OT. Shows today's intention if set, or prompts to create one if missing (when `prompt_on_empty` is enabled).

```bash
daylit ot nudge
```

This is designed for quick morning checks or reminders throughout the day.

**Example:**

```bash
# When OT is set
$ daylit ot nudge
Today's OT:
  Complete the daylit feature
  Note: Focus on testing and documentation

# When OT is not set
$ daylit ot nudge
No OT set for today.
Set your Once-Today intention with: daylit ot set --title "..."
```

### `daylit ot doctor`

Run diagnostics on OT data integrity.

```bash
daylit ot doctor
```

**Checks performed:**

- OT settings exist
- Date formats are valid (YYYY-MM-DD)
- No duplicate days (only one active entry per day)
- Timestamps are not corrupted

**Example:**

```bash
$ daylit ot doctor
Running OT diagnostics...

✓ OT settings: OK
✓ Date validation: OK
✓ Duplicate days: OK
✓ Timestamp validation: OK

All OT diagnostics passed!
```

### `daylit ot delete`

Soft-delete an OT entry for a specific day.

```bash
daylit ot delete [flags]
```

**Flags:**

- `--day DATE`: Date in YYYY-MM-DD format (default: today)

**Example:**

```bash
# Delete today's OT
daylit ot delete

# Delete specific day
daylit ot delete --day 2025-12-30
```

### `daylit ot restore`

Restore a soft-deleted OT entry.

```bash
daylit ot restore [flags]
```

**Flags:**

- `--day DATE`: Date in YYYY-MM-DD format (default: today)

**Example:**

```bash
# Restore today's OT
daylit ot restore

# Restore specific day
daylit ot restore --day 2025-12-30
```

## `daylit settings`

View and manage application settings.

```bash
daylit settings [flags]
```

**Flags:**

- `--list`: List all current settings
- `--timezone STRING`: Set timezone (IANA name, e.g., 'America/New_York', 'Europe/London', or 'Local' for system timezone)
- `--notifications-enabled BOOL`: Enable or disable notifications
- `--notify-block-start BOOL`: Enable block start notifications
- `--notify-block-end BOOL`: Enable block end notifications
- `--block-start-offset-min INT`: Minutes before block start to send notification
- `--block-end-offset-min INT`: Minutes before block end to send notification
- `--ot-prompt-on-empty BOOL`: Prompt when no OT entry exists for today
- `--ot-strict-mode BOOL`: Strict mode - only one OT entry per day
- `--ot-default-log-days INT`: Default number of days to show in OT log view

### View Current Settings

```bash
daylit settings --list
```

**Example output:**

```
Current Settings:
  Day Start:             07:00
  Day End:               22:00
  Default Block Min:     30
  Timezone:              Local

Once Today (OT) Settings:
  Prompt On Empty:       true
  Strict Mode:           true
  Default Log Days:      14

Notification Settings:
  Notifications Enabled: true
  Notify Block Start:    true
  Notify Block End:      true
  Block Start Offset:    5 min
  Block End Offset:      5 min
```

### Update Settings

```bash
# Set timezone to America/New_York
daylit settings --timezone="America/New_York"

# Set timezone to UTC
daylit settings --timezone="UTC"

# Set timezone to system local timezone
daylit settings --timezone="Local"

# Disable notifications
daylit settings --notifications-enabled=false

# Change notification timing
daylit settings --block-start-offset-min=10

# Update OT settings
daylit settings --ot-default-log-days=30
```

### Timezone Configuration

The timezone setting controls how daylit interprets dates and times. This is particularly useful when:

- You travel frequently and want daylit to respect your current timezone
- You want to schedule tasks in a specific timezone different from your system timezone
- You need consistent behavior across devices in different timezones

**Supported values:**

- `Local` (default): Uses your system's local timezone
- IANA timezone names: `America/New_York`, `Europe/London`, `Asia/Tokyo`, `UTC`, etc.

**Important notes:**

- This setting is stored for future use in timezone-aware scheduling and notifications
- Currently, the application uses your system's local timezone for date/time operations
- Future updates will integrate this setting to affect how "today" is determined and when notifications are triggered
- Use the full IANA timezone name (e.g., `America/New_York` not `EST`)

**Example:**

```bash
# View current timezone
daylit settings --list | grep Timezone

# Set timezone to New York
daylit settings --timezone="America/New_York"

# Set timezone to London
daylit settings --timezone="Europe/London"

# Reset to system timezone
daylit settings --timezone="Local"
```
