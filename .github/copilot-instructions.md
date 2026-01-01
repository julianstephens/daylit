# Copilot Instructions

You are an expert developer working on the `daylit` project, which consists of a Go CLI (`daylit-cli`) and a Tauri/Rust system tray application (`daylit-tray`).

## General Guidelines

- **Code Quality**: Write clean, maintainable, and idiomatic code.
- **Testing**: Always add or update tests when modifying logic. Ensure all tests pass before considering a task complete.
- **Linting**: Ensure code satisfies the project's linting rules.

## Project Specifics

### Go (daylit-cli)

- **Location**: `./daylit-cli`
- **Testing**: Run `go test ./...` to verify changes.
- **Linting**:
  - Code must pass `go vet ./...`.
  - Code must pass `golangci-lint run`.
- **Formatting**: Use standard `gofmt`.
- **Conventions**:
  - Use `internal/` packages for private implementation details.
  - Follow Go error handling idioms (`if err != nil`).

### Rust (daylit-tray/src-tauri)

- **Location**: `./daylit-tray/src-tauri`
- **Testing**: Run `cargo test --workspace` to verify changes.
- **Linting**:
  - Run `cargo clippy --workspace -- -D warnings` and fix all warnings.
  - Ensure code compiles with `cargo check --workspace`.
- **Formatting**: Use `cargo fmt`.
- **Conventions**:
  - Prefer `if let` chains over nested `if` statements where supported.
  - Handle `Result` and `Option` types explicitly; avoid `unwrap()` in production code unless absolutely safe.

### TypeScript/React (daylit-tray)

- **Location**: `./daylit-tray`
- **Testing/Building**:
  - Ensure the project builds with `npm run build`.
  - Verify types with `npx tsc --noEmit`.
- **Formatting**: Use `npm run format` for code formatting.
- **Linting**: Use `npm run lint` to ensure code quality.
- **Conventions**:
  - Use functional components and Hooks.
  - Ensure strict type safety.

## Workflow

1.  **Analyze**: Understand the file structure and existing patterns before editing.
2.  **Edit**: Make necessary changes.
3.  **Verify**:
    - For Go: `cd daylit-cli && go test ./... && go vet ./...`
    - For Rust: `cd daylit-tray/src-tauri && cargo test && cargo clippy -- -D warnings`
    - For TS: `cd daylit-tray && npx tsc --noEmit`
