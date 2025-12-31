# daylit-tray

`daylit-tray` is a lightweight system tray application designed to accompany the `daylit` CLI tool. It runs in the background and provides desktop notifications triggered via webhooks.

## Features

- **System Tray Integration**: Runs unobtrusively in your system tray.
- **Webhook Server**: Listens on a local port for notification requests.
- **Desktop Notifications**: Displays custom notification windows when triggered.
- **Auto-Discovery**: Writes its listening port to a lock file (`daylit-tray.lock`) for the CLI to find.

## How it Works

1. **Startup**: When launched, the application starts a local HTTP server on an available port.
2. **Registration**: The port number is written to `daylit-tray.lock` in the application's configuration directory.
3. **Listening**: The app waits for incoming HTTP POST requests containing a JSON payload with `text` and `duration_ms`.
4. **Notification**: Upon receiving a valid request, it opens a notification window displaying the message.

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
