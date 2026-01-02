# Daylit

Daylit is a comprehensive daily structure and time-blocking system designed to help you manage your day effectively. It consists of a powerful CLI for scheduling and a lightweight system tray application for desktop integration.

## Documentation

- **[Quick Start Guide](docs/user-guides/QUICK_START.md)**: Get up and running in minutes.
- **[Storage Configuration](docs/user-guides/STORAGE.md)**: Configure SQLite or PostgreSQL backends.
- **[Windows & WSL Setup](docs/user-guides/WINDOWS_WSL_SETUP.md)**: Configure Daylit for Windows and WSL2.
- **[Core Concepts](docs/CONCEPTS.md)**: Understand the scheduling algorithm, data models, and feedback loop.
- **[CLI Reference](docs/CLI_REFERENCE.md)**: Detailed command reference.
- **[Additional User Guides](docs/user-guides/)**: Browse all guides including Alerts & Notifications and advanced configuration.

## Components

The repository is divided into two main components. **See their respective READMEs for technical details and development instructions.**

### 1. [Daylit CLI](./daylit-cli)

The core of the system. A command-line interface tool written in Go that handles task management, scheduling, and tracking.

[🛠️ Developer Documentation](./daylit-cli/README.md)

### 2. [Daylit Tray](./daylit-tray)

A desktop companion application built with Tauri (Rust + React) that handles system tray integration and notifications.

[🛠️ Developer Documentation](./daylit-tray/README.md)

## System Overview

Daylit is designed to work as a cohesive system:

1. **The CLI** acts as the brain. It stores your schedule, knows what time it is, and determines what you should be doing.
2. **The Tray App** acts as the notifier. It runs in the background and exposes a local server.
3. When the CLI needs to alert you (e.g., a task is starting or ending), it discovers the Tray app's port and authentication secret via a lock file and sends an authenticated notification request.
4. The Tray app validates the request's authentication secret and, if valid, displays a native desktop notification window.

### Security

The system uses a secret-in-lockfile authentication mechanism to ensure secure communication. The tray app generates a session-specific secret on startup, which the CLI reads to authenticate notification requests. This ensures that only authorized processes running as the same user can trigger notifications.

## Installation

### Pre-built Binaries

The easiest way to get started is to download pre-built binaries from the [latest release](https://github.com/julianstephens/daylit/releases/latest).

### Building from Source

To build the entire system from source, you will need Go, Node.js, and Rust installed.

```bash
# Build CLI
cd daylit-cli
make build

# Build Tray App
cd ../daylit-tray
npm install
npm run tauri build
```
