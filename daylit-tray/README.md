# daylit-tray (Developer Guide)

This document contains technical details for developers working on the `daylit-tray` component. For user documentation, see the [root README](../README.md).

## Technical Architecture

### How it Works

1. **Startup**: When launched, the application starts a local HTTP server on an available port.
2. **Registration**: The port number, process ID, and a secure random secret are written to `daylit-tray.lock` in the application's configuration directory.
3. **Listening**: The app waits for incoming HTTP POST requests containing a JSON payload with `text` and `duration_ms`.
4. **Authentication**: Each request must include an `X-Daylit-Secret` header with the secret from the lock file. Requests without a valid secret are rejected with a 401 Unauthorized response.
5. **Notification**: Upon receiving a valid authenticated request, it opens a notification window displaying the message.

### Security Implementation

The tray application implements a secret-in-lockfile authentication mechanism:

- **Secret Generation**: On startup, a secure 32-character random alphanumeric string is generated using a cryptographically secure random number generator.
- **Lockfile Format**: The lock file contains three pipe-separated values: `PORT|PID|SECRET`
  - `PORT`: The TCP port the webhook server is listening on
  - `PID`: The process ID of the running tray application
  - `SECRET`: The authentication secret for this session
- **File Permissions**: On Unix-like systems, the lock file is created with `0600` permissions (readable/writable only by the owner).
- **Header Validation**: All POST requests to the webhook server must include an `X-Daylit-Secret` header with the correct secret value.
- **Session-Specific**: The secret changes every time the tray app restarts, ensuring that old secrets cannot be reused.

This approach ensures that only processes that can read the lock file (i.e., processes running as the same user) can send notifications, preventing unauthorized notification spam from other users or malicious applications.

## Development

### Prerequisites

- [Node.js](https://nodejs.org/)
- [Rust](https://www.rust-lang.org/)
- [Tauri CLI](https://tauri.app/v2/guides/start)

### Setup

1. Install dependencies:

   ```bash
   npm install
   ```

2. Run in development mode:

   ```bash
   npm run tauri dev
   ```

### Build

To build the application for production:

```bash
npm run tauri build
```

## Tech Stack

- **Frontend**: React, TypeScript, Vite
- **Backend**: Rust (Tauri)
