# Alerts and Notifications

Daylit uses a decoupled architecture for notifications to ensure flexibility and minimal resource usage. The system consists of two parts:

1.  **`daylit-cli`**: Checks your schedule and determines when a notification should be sent.
2.  **`daylit-tray`**: A lightweight system tray application that listens for notification requests and displays them to the user.

This guide explains how to set up this system so you never miss a task.

## Prerequisites

-   **`daylit-cli`** installed and configured.
-   **`daylit-tray`** installed.

## Step 1: Run the Tray Application

The `daylit-tray` application must be running to receive and display notifications. It sits in your system tray and listens for requests from the CLI.

1.  Start the application:
    ```bash
    # Assuming daylit-tray is in your path
    daylit-tray &
    ```
    Or launch it from your desktop environment's application menu if installed.

2.  Verify it's running:
    -   You should see the Daylit icon in your system tray.
    -   It creates a lock file at `~/.config/com.daylit.daylit-tray/daylit-tray.lock` (path may vary by OS) containing connection details.

## Step 2: Configure Notification Settings

Enable and configure notifications using the CLI.

```bash
# Enable notifications
daylit settings --notifications-enabled=true

# Notify 5 minutes before a block starts
daylit settings --notify-block-start=true --block-start-offset-min=5

# Notify when a block ends
daylit settings --notify-block-end=true --block-end-offset-min=0
```

You can verify your settings with:
```bash
daylit settings --list
```

### Setting Up Custom Alerts

In addition to automatic schedule notifications, you can set up custom one-time or recurring alerts.

```bash
# Add a one-time alert for a specific date and time
daylit alert add "Doctor's Appointment" --time 14:30 --date 2024-03-20

# Add a daily recurring alert
daylit alert add "Drink Water" --time 10:00 --recurrence daily

# Add a weekly alert on specific days
daylit alert add "Team Standup" --time 09:00 --recurrence weekly --weekdays mon,tue,wed,thu,fri

# Add an alert every 3 days
daylit alert add "Water Plants" --time 18:00 --recurrence n_days --interval 3
```

## Step 3: Set Up the Scheduler (Optional)

**Note:** If you are running `daylit-tray`, this step is **not required**. The tray application automatically runs the notification check every minute. Follow these instructions only if you are not using `daylit-tray` or prefer to manage the scheduling process yourself (e.g., for a headless server setup).

The `daylit-cli` does not run in the background. You need to schedule the `daylit notify` command to run frequently (e.g., every minute) to check your plan and trigger notifications.

### Option A: Using Cron (Linux/macOS)

1.  Open your crontab for editing:
    ```bash
    crontab -e
    ```

2.  Add the following line to run the check every minute:
    ```cron
    * * * * * /path/to/daylit notify
    ```
    *Replace `/path/to/daylit` with the actual absolute path to your `daylit` binary (e.g., `/usr/local/bin/daylit` or `/home/user/go/bin/daylit`).*

### Option B: Using Systemd Timer (Linux)

For a more robust setup on Linux, you can use a Systemd timer.

1.  Create a service file `~/.config/systemd/user/daylit-notify.service`:
    ```ini
    [Unit]
    Description=Daylit Notification Check

    [Service]
    Type=oneshot
    ExecStart=/path/to/daylit notify
    ```

2.  Create a timer file `~/.config/systemd/user/daylit-notify.timer`:
    ```ini
    [Unit]
    Description=Run Daylit Notification Check every minute

    [Timer]
    OnCalendar=*:0/1
    Persistent=true

    [Install]
    WantedBy=timers.target
    ```

3.  Enable and start the timer:
    ```bash
    systemctl --user daemon-reload
    systemctl --user enable --now daylit-notify.timer
    ```

### Option C: Using Task Scheduler (Windows)

On Windows, you can use the Task Scheduler to run the command every minute.

1.  Open PowerShell or Command Prompt.

2.  Create a scheduled task to run every minute:
    ```powershell
    schtasks /Create /SC MINUTE /MO 1 /TN "DaylitNotify" /TR "C:\path\to\daylit.exe notify"
    ```
    *Replace `C:\path\to\daylit.exe` with the actual absolute path to your `daylit` executable.*

3.  To remove the task later:
    ```powershell
    schtasks /Delete /TN "DaylitNotify"
    ```

## Troubleshooting

If you are not receiving notifications:

1.  **Check Settings**: Ensure notifications are enabled (`daylit settings --list`).
2.  **Dry Run**: Run the notify command manually with the dry-run flag to see what it *would* do:
    ```bash
    daylit notify --dry-run
    ```
    If it says "No plan found for today" or similar, ensure you have generated a plan (`daylit plan today`).
3.  **Check Tray App**: Ensure `daylit-tray` is running.
4.  **Check Paths**: Verify the path to the `daylit` binary in your cron or systemd config is correct. Cron often has a limited `$PATH`.
