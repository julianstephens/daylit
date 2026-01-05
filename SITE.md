# Daylit: Personal Productivity System

Daylit is a comprehensive daily structure and time-blocking system I engineered to manage complex schedules effectively. It combines a robust Go-based CLI for logic and scheduling with a lightweight Rust/Tauri system tray application for desktop integration.

## The "Why" & The Solution

I needed a system that prioritized **predictability over optimization** while remaining flexible enough for real-life interruptions. Daylit isn't just a todo list; it's a scheduling engine with a feedback loop.

- **The Brain (CLI)**: Written in **Go**, it handles complex recurrence logic, time-blocking algorithms, and data persistence (SQLite/PostgreSQL).
- **The Messenger (Tray)**: Built with **Rust (Tauri)** and **React**, it acts as a secure notification endpoint, alerting the user to schedule changes without consuming heavy resources.

## Technical Architecture

The system demonstrates a decoupled, secure architecture:

1.  **Service Discovery & Security**: The Tray app writes a lockfile containing its port and a session-specific **simulated mutual auth secret**.
2.  **Communication**: The CLI discovers the running Tray instance via the lockfile and sends authenticated HTTP requests to trigger notifications.
3.  **Persistence Layer**: Abstracted storage interface supporting both **SQLite** (for portable, zero-config usage) and **PostgreSQL** (for scalability).

### Tech Stack

| Component | Technology | Highlights |
| :--- | :--- | :--- |
| **CLI Core** | **Go** | Clean Architecture, strictly typed domain logic |
| **CLI UI** | **Bubble Tea** | Interactive TUI for plan management |
| **Tray Backend** | **Rust** | Memory safety, OS-native system tray integration |
| **Tray Frontend** | **TypeScript + React** | Modern, reactive UI components |
| **Framework** | **Tauri v2** | Lightweight electron alternative |

## Engineering Standards

### 1. Robust Testing Strategy
- **Unit Tests**: Extensive coverage for domain logic (scheduler, recurrence rules).
- **Integration Tests**: Specialized test harnesses for database interactions (`internal/backup`, `storage`).
- **E2E Tests**: Full workflow validation located in `tests/e2e`.

### 2. Code Quality & Static Analysis
- **Go**: `golangci-lint` with strict presets + `go vet`.
- **Rust**: `cargo clippy` with `-D warnings` (zero warnings policy).
- **TypeScript**: `eslint` + Strict type checking.

### 3. Clean Architecture
 The Go CLI follows a strict package layout to separate concerns:
- `internal/scheduler`: Pure domain logic (no IO).
- `internal/storage`: Implementation details for persistence.
- `internal/cli`: Interface layer using `Kong`.

## Key Features

### Smart Scheduling
**Problem**: Traditional calendars don't handle "do this sometime today" well.

**Solution**: A deterministic bin-packing algorithm that treats the day as a container.

**Implementation**:
- **Fixed-First Strategy**: Anchors the schedule with appointments (hard constraints) first to identify available free blocks.
- **Dynamic Bin-Packing**: Flexible tasks are sorted by priority and "lateness" (days since last completion), then fit into available gaps using a greedy algorithm.
- **Recurrence Engine**: Handles complex patterns (e.g., "every 3 days", "weekdays only") to filter candidate tasks for the day.

### Feedback Loop & Optimization
**Problem**: Schedules usually fail because estimates are wrong.

**Solution**: A closed-loop system that learns from reality.

**Implementation**:
- **Exponential Moving Average (EMA)**: The system tracks "actual" vs "planned" duration for every task execution, updating a weighted average to refine future time slots automatically.
- **Signal Analysis**: The `optimizer` package aggregates qualitative feedback (`too_much`, `unnecessary`) to detect patterns.
- **Actionable Insights**: If a task is consistently marked "too much", the system suggests specific optimizations (e.g., "reduce duration by 10%" or "split task").

### Resilience & Safety
**Problem**: Accidental data loss and destructive edits break trust in the system.

**Solution**: A non-destructive "Soft Delete" architecture.

**Implementation**:
- **Tombstone Pattern**: All core entities (`Task`, `Plan`, `Habit`) use nullable `deleted_at` timestamps instead of hard removal.
- **Restoration API**: First-class CLI support for restoring deleted items allows users to undo mistakes instantly.
- **Referential Integrity**: Keeps historical statistics valid even when tasks are seemingly removed from the active view.

### Secure Credentials
**Problem**: Storing database passwords in plaintext config files is a security risk.

**Solution**: Leveraging the operating system's native secret management.

**Implementation**:
- **OS Integration**: Uses `zalando/go-keyring` to interface directly with macOS Keychain, Windows Credential Manager, or Linux Secret Service (dbus).
- **Zero-Config Secrets**: Connection strings are stored securely at runtime, keeping `config.yaml` clean and safe to check into version control (minus the secrets).

