# Daylit

Daylit is a comprehensive daily structure and time-blocking system designed to help you manage your day effectively. It consists of a powerful CLI for scheduling and a lightweight system tray application for desktop integration.

## Components

The repository is divided into two main components:

### 1. [Daylit CLI](./daylit-cli)
The core of the system. A command-line interface tool written in Go that handles:
- **Task Management**: Manage recurring and one-off task templates.
- **Scheduling**: Generate daily time-blocked schedules based on your templates.
- **Tracking**: Keep track of what you should be doing right now.
- **Feedback**: Collect feedback on your day to improve future schedules.

[Read the CLI Documentation](./daylit-cli/README.md)

### 2. [Daylit Tray](./daylit-tray)
A desktop companion application built with Tauri (Rust + React) that:
- **System Tray Integration**: Runs unobtrusively in your system tray.
- **Notifications**: Displays custom desktop notifications.
- **Webhook Server**: Listens for notification requests from the CLI.

[Read the Tray Documentation](./daylit-tray/README.md)

## How They Work Together

Daylit is designed to work as a cohesive system:

1. **The CLI** acts as the brain. It stores your schedule, knows what time it is, and determines what you should be doing.
2. **The Tray App** acts as the notifier. It runs in the background and exposes a local server.
3. When the CLI needs to alert you (e.g., a task is starting or ending), it discovers the Tray app's port via a lock file and sends a notification request.
4. The Tray app receives the request and displays a native desktop notification window.

## Getting Started

To get the full Daylit experience, you should set up both components.

### Prerequisites
- **Go** (for the CLI)
- **Node.js** & **Rust** (for the Tray app)

### Installation

#### 1. Install the CLI
Navigate to the `daylit-cli` directory and build the binary:
```bash
cd daylit-cli
make build
# Or install directly
go install ./cmd/daylit
```

#### 2. Install and Run the Tray App
Navigate to the `daylit-tray` directory and run the application:
```bash
cd daylit-tray
npm install
npm run tauri dev # For development
# OR build for production
npm run tauri build
```

Once the Tray app is running, it will automatically write its connection details to a lock file that the CLI can read. You can then use the CLI normally, and notifications will be routed to your desktop.

## Development

Please refer to the individual `README.md` files in each subdirectory for specific development instructions, contribution guidelines, and architectural details.