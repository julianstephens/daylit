# Windows & WSL Setup Guide

This guide explains how to configure `daylit` for a seamless experience where you run the **CLI in WSL2** and the **System Tray application on Windows**, with both accessing the same data.

## Overview

To share data between the Windows host (running `daylit-tray`) and the WSL2 environment (running `daylit-cli`), you must use **PostgreSQL** as the storage backend. SQLite cannot reliably handle concurrent access across the Windows/WSL file system boundary due to file locking limitations.

**Architecture:**
- **Database**: PostgreSQL (running on Windows or Docker Desktop).
- **Windows**: Runs `daylit-tray` (which calls `daylit.exe` internally). Connects to DB via `localhost`.
- **WSL**: Runs `daylit` CLI commands. Connects to DB via the Windows host IP.

## Prerequisites

1.  **WSL2** installed and configured.
2.  **PostgreSQL** accessible from both Windows and WSL.
    -   *Most Reliable*: **Hosted Database** (Supabase, Neon, AWS RDS, etc.).
    -   *Recommended Local*: **Docker Desktop for Windows** (simplest networking).
    -   *Alternative*: Native PostgreSQL installation on Windows.

## Step 1: Database Setup

We recommend using a hosted database (like Supabase, Neon, or AWS RDS) for the best reliability, as it avoids local networking complexities. If you prefer a local setup, Docker Desktop is the next best option.

### Option A: Hosted Database (Most Reliable)

1.  Create a PostgreSQL database on a cloud provider (e.g., Supabase, Neon, AWS RDS, Google Cloud SQL).
2.  Obtain the connection string. It usually looks like:
    ```
    postgres://user:password@host:5432/dbname?sslmode=require
    ```
    *Note: Ensure your database allows connections from your IP address.*

### Option B: Docker Desktop (Recommended Local)

1.  Install Docker Desktop for Windows.
2.  Start a PostgreSQL container:
    ```powershell
    docker run --name daylit-db -e POSTGRES_USER=daylit_user -e POSTGRES_PASSWORD=secure_password -e POSTGRES_DB=daylit -p 5432:5432 -d postgres:15
    ```

### Option C: Native Windows PostgreSQL

1.  Install PostgreSQL for Windows.
2.  Create the database and user as described in the [PostgreSQL Setup Guide](POSTGRES_SETUP.md).
3.  **Important**: You must configure PostgreSQL to listen on all interfaces and allow connections from WSL.
    -   Edit `postgresql.conf`: Set `listen_addresses = '*'`.
    -   Edit `pg_hba.conf`: Add a line to allow **only** the WSL subnet (often `172.x.x.x` or `192.168.x.x`). For example:
        ```
        # Replace 172.22.0.0/16 with your actual WSL subnet
        host    all             all             172.22.0.0/16        scram-sha-256
        ```
        You can find your WSL IP/subnet with `ip addr` inside WSL or via `ipconfig` on Windows and then adjust the CIDR accordingly.

        **Security warning:** Do **not** use `0.0.0.0/0` in `pg_hba.conf` for production or internet-accessible PostgreSQL instances, as it allows connections from any IPv4 address.
    -   Restart the PostgreSQL service.

## Step 2: Configure Windows (Tray App)

The Tray application runs on Windows and uses the Windows version of the `daylit` CLI to check for notifications.

1.  **Install `daylit-cli` on Windows**:
    -   Download the `daylit.exe` binary.
    -   Add it to your system `PATH`.

2.  **Configure the Connection**:
    -   Open PowerShell.
    -   Use the `keyring` command to securely store the connection string. This allows `daylit-tray` to pick it up automatically without needing environment variables.
    ```powershell
    # Replace with your actual connection string
    # For Hosted DB: Use the string provided by your cloud provider
    # For Local/Docker: postgres://daylit_user:secure_password@localhost:5432/daylit?sslmode=disable
    daylit keyring set "postgres://daylit_user:secure_password@localhost:5432/daylit?sslmode=disable"
    ```

3.  **Install and Start `daylit-tray`**:
    -   Install and run the tray application.
    -   It will now use the PostgreSQL database.

## Step 3: Configure WSL (CLI)

Now configure your WSL environment to talk to the same database.

1.  **Install `daylit-cli` in WSL**:
    -   Follow the standard Linux installation instructions.

2.  **Determine the Host Address**:
    -   **If using Hosted DB**: Use the host provided by your cloud provider (e.g., `db.xyz.supabase.co`).
    -   **If using Docker Desktop**: You can use `host.docker.internal`.
    -   **If using Native Postgres**: You need the IP address of the Windows host as seen from WSL. You can find this with:
        ```bash
        grep nameserver /etc/resolv.conf | awk '{print $2}'
        ```

3.  **Configure the Connection**:
    Use the `keyring` command to securely store the connection string. This is preferred over environment variables.

    **For Hosted DB users:**
    ```bash
    daylit keyring set "postgres://user:password@host:5432/dbname?sslmode=require"
    ```

    **For Docker Desktop users:**
    ```bash
    daylit keyring set "postgres://daylit_user:secure_password@host.docker.internal:5432/daylit?sslmode=disable"
    ```

    **For Native Postgres users:**
    *Note: Because the Windows host IP changes on every WSL restart, using the keyring (which stores a static string) is inconvenient. For this specific case, we recommend using an environment variable in your `~/.bashrc`:*
    ```bash
    export WINDOWS_HOST=$(grep nameserver /etc/resolv.conf | awk '{print $2}')
    export DAYLIT_CONFIG="postgres://daylit_user:secure_password@$WINDOWS_HOST:5432/daylit?sslmode=disable"
    ```

4.  **Apply Changes** (if using environment variables):
    ```bash
    source ~/.bashrc
    ```

## Verification

1.  **In WSL**: Add a task.
    ```bash
    daylit task add "Test Sync" --duration 30
    ```

2.  **In Windows (PowerShell)**: List tasks to verify visibility.
    ```powershell
    daylit task list
    ```

3.  **Check Tray**: The tray application should now be monitoring this schedule.

## Troubleshooting

### Connection Refused
-   **Firewall**: Ensure Windows Firewall allows traffic on port 5432. You may need to create an Inbound Rule allowing TCP port 5432.
-   **Postgres Config**: If using native Postgres, double-check `listen_addresses` in `postgresql.conf` is set to `*`.

### "host.docker.internal" not resolving
-   Ensure you are running a recent version of Docker Desktop and WSL2.
-   If it fails, try using the IP address method described for Native Postgres.
