# VORTEX

The Ultimate Terminal User Interface (TUI) for Remote Server Management.

Vortex is a zero-configuration, blazing-fast VPS Manager that runs entirely inside your terminal. 

Instead of forcing you to install complex monitoring agents on your servers manually, Vortex uses an **Agentless Injection Architecture**. You simply provide your SSH credentials, and Vortex securely pushes its own compiled telemetry payload over the SSH tunnel to stream real-time data back to your dashboard.

No bloat. No background daemons. Just pure terminal speed.

## Features

- **Live Telemetry Dashboard**: Monitor real-time CPU, Memory, Disk, and Network traffic.
- **Application Manager**: Dynamically detects Node.js, Python, PM2, Docker, Go, and Java processes based on active listening ports. Gracefully stop or force-kill processes directly from the UI.
- **Security Center**: Runs a vulnerability audit on connection, checking for insecure SSH root logins, password authentication, and UFW firewall statuses.
- **Interactive Docker Manager**: View, restart, and stop your Docker containers natively from a UI table without ever typing docker ps.
- **Systemd Integration**: Browse your active Linux services (like Nginx or Postgres) and restart them instantly.
- **Remote File Explorer**: An interactive file browser to quickly traverse your server directories.
- **Unified Logging Engine**: Live-stream and pause systemd journalctl logs directly in the terminal.
- **Backup Manager**: Trigger one-click asynchronous tarball backups of directories (stored safely in /tmp).
- **Command Palette**: A VS-Code style global command palette (Ctrl+P) to instantly switch between modules.
- **Dynamic Settings System**: A robust, JSON-backed configuration registry that allows you to hot-swap UI themes and tweak engine behaviors.
- **Native SSH Dropper**: Select a server and press ENTER to drop directly into a native SSH session.

## Installation

### Prerequisites
- Go 1.20+ installed on your machine.
- A remote Linux server with an SSH port open (or a local Docker container for testing).

### Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/your-username/vortex.git
   cd vortex
   ```

2. **Run it directly:**
   ```bash
   go run ./cmd/vps-manager/main.go
   ```

3. **Or, compile it into an executable for maximum speed:**
   ```bash
   go build -o vortex.exe ./cmd/vps-manager
   ```

## How to Use

1. **Add a Server**: Scroll to the bottom of the server list and select [+] Add New Server.
2. **Fill out the Details**: You can use either a Password or an SSH Key path.
3. **Connect**: Select your newly added server and hit ENTER. Vortex will cross-compile the agent, deploy it, and load your dashboard.
4. **Navigate**: Use [ / ] to shift your sidebar context, and ENTER to dive into a specific module.
5. **Command Palette**: Press CTRL+P anywhere to open the global Command Palette.

## Testing Locally (Without a VPS)

If you want to test Vortex but don't have a remote server, you can instantly simulate one using Docker Desktop:

```bash
docker run -d --name vortex-test -e USER_NAME=admin -e USER_PASSWORD=password -e PASSWORD_ACCESS=true -p 2222:2222 lscr.io/linuxserver/openssh-server
```
Then, connect Vortex to:
- **Host**: 127.0.0.1
- **Port**: 2222
- **Username**: admin
- **Password**: password

## Architecture

Vortex is built on a highly modular, decoupled architecture:
1. **The Client (TUI)**: Built with Bubble Tea and Lipgloss. It acts as the SSH controller and UI renderer.
2. **The Service Engines**: Isolated backend modules (Apps, Docker, Files, Logs, Security, Systemd, Backup) that safely execute SSH commands asynchronously via Goroutines.
3. **The Agent**: A lightweight Go binary (cmd/vortex-agent). The Client automatically cross-compiles this binary for Linux, pushes it to the server's /tmp directory via SSH, executes it to fetch JSON stats, and removes it.

---
Built for terminal power users.
