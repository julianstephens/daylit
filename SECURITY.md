# Security

## Tray-CLI Communication Security

### Secret-in-Lockfile Authentication

The communication between the daylit CLI and the daylit-tray application uses a secret-in-lockfile authentication mechanism to prevent unauthorized notification requests.

#### How It Works

1. **Secret Generation**
   - When the tray application starts, it generates a secure 32-character random alphanumeric string using a cryptographically secure random number generator (CSPRNG)
   - The secret is unique to each session and changes every time the tray app is restarted

2. **Lockfile Storage**
   - The tray app writes a lock file to the user's configuration directory (typically `~/.config/com.daylit.daylit-tray/` on Linux/macOS or `%APPDATA%\com.daylit.daylit-tray\` on Windows)
   - The lock file (`daylit-tray.lock`) contains three pipe-separated values:
     ```
     PORT|PID|SECRET
     ```
     - `PORT`: The TCP port number the webhook server is listening on (e.g., `54321`)
     - `PID`: The process ID of the running tray application (e.g., `12345`)
     - `SECRET`: The 32-character authentication secret (e.g., `abc123...xyz`)
   - On Unix-like systems, the lock file is created with `0600` permissions (readable/writable only by the file owner)

3. **CLI Authentication**
   - When the CLI needs to send a notification, it reads the lock file to discover:
     - Which port to connect to
     - Which process ID to validate is running
     - What secret to use for authentication
   - The CLI includes the secret in an `X-Daylit-Secret` HTTP header when making POST requests to the tray server

4. **Server Validation**
   - The tray app's webhook server validates every incoming request
   - If the `X-Daylit-Secret` header is missing or doesn't match the expected secret, the server returns a `401 Unauthorized` response
   - Only requests with the correct secret are processed and result in notifications being displayed

#### Security Properties

- **User Isolation**: Only processes running as the same user can read the lock file (enforced by OS file permissions)
- **Session Binding**: Each tray app session has a unique secret, preventing replay attacks across sessions
- **Local-Only**: The webhook server only listens on `127.0.0.1` (localhost), preventing network-based attacks
- **Unguessable Secrets**: 32 characters of alphanumeric data provide ~190 bits of entropy, making brute-force attacks infeasible
- **Process Validation**: The CLI validates that the PID in the lock file corresponds to a running `daylit-tray` process before attempting to send notifications

#### Threat Model

This authentication mechanism protects against:

- **Other users on the system**: Cannot read the lock file due to file permissions
- **Malicious applications**: Cannot guess the secret to send unauthorized notifications
- **Network attackers**: Cannot reach the server as it only listens on localhost
- **Stale sessions**: Old secrets cannot be reused after the tray app restarts

This mechanism does NOT protect against:

- **Compromised user account**: If an attacker has access to the user's account, they can read the lock file
- **Root/Administrator access**: System administrators can always read user files
- **Process injection**: Attackers with code execution in the user's context can access the lock file

These are considered acceptable trade-offs given the threat model of a local desktop application designed for single-user systems.
