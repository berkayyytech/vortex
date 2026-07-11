# VPS Manager

A robust and scalable VPS management application built with Go.

## Standard Go Project Layout

*   **`cmd/vps-manager/`**: The main application entry point. This directory should only contain the `main.go` file which wires up dependencies and starts the application.
*   **`internal/`**: Private application and library code. This code cannot be imported by other projects.
    *   **`internal/api/`**: HTTP handlers, routing, and middleware.
    *   **`internal/ssh/`**: SSH client logic for connecting to and executing commands on remote servers.
    *   **`internal/db/`**: Database connection setup, queries, and repositories.
    *   **`internal/models/`**: Core data structures and domain models (e.g., `Server`, `User`, `Plan`).
    *   **`internal/config/`**: Configuration loading and parsing from environment variables or files.
*   **`pkg/`**: Library code that is okay to be consumed by external applications (e.g., a custom `logger` or utility package).

## Getting Started

To run the application locally from the standard structure:

```bash
go run ./cmd/vps-manager/main.go
```
