# Architecture

## Package Structure

The `daylit-cli` project follows a standard Go project layout. The core logic is contained within the `internal` directory to ensure encapsulation.

### Entry Point

- **`cmd/daylit`**: The main entry point for the application. It initializes the CLI and starts the execution.

### Core Packages (`internal/`)

- **`backup`**: Manages database backup creation, rotation, and restoration workflows.
- **`cli`**: Contains the Cobra command definitions. It maps command-line arguments and flags to application logic.
- **`constants`**: Holds application-wide constants, configuration defaults, and shared string literals.
- **`keyring`**: Handles secure storage and retrieval of sensitive information (like database connection strings) using the operating system's keyring.
- **`logger`**: Provides structured logging capabilities for the application.
- **`migration`**: Manages database schema migrations, ensuring the database structure is up-to-date with the application version.
- **`models`**: Defines the core domain entities and data structures used throughout the application (e.g., `Task`, `Habit`, `DayPlan`, `Settings`).
- **`notifier`**: Handles the delivery of system notifications for scheduled tasks, habits, and alerts.
- **`optimizer`**: Contains the logic for the schedule optimization engine, which can suggest adjustments to task durations and frequencies.
- **`scheduler`**: The core domain logic for scheduling. It handles time slot allocation, conflict detection, and plan generation.
- **`storage`**: Defines the `Provider` interface for data persistence and includes implementations for supported backends (SQLite, PostgreSQL).
- **`tui`**: Implements the interactive Terminal User Interface using the Bubble Tea framework. It includes the state management, components, and event handlers for the TUI.
- **`utils`**: General-purpose utility functions used across multiple packages.
- **`validation`**: Contains logic for validating user input and domain constraints.
