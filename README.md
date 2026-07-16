# Vortex

Vortex is a powerful, keyboard-first Terminal User Interface (TUI) for managing your fleet of VPS servers over SSH. Built entirely in Go using the BubbleTea framework, Vortex eliminates the need for complex web dashboards by providing a fast, visually rich, and highly operational environment directly from your terminal.

## Features

Vortex is divided into multiple specialized modules, all accessible via global keyboard shortcuts or the Command Palette.

### Core Modules
* **Mission Control**: A high-level overview of your infrastructure with live CPU, memory, disk, and network telemetry.
* **Server Manager**: Easily manage, configure, and connect to multiple remote servers via SSH keys or passwords.
* **Network Core**: Monitor server health and infrastructure connectivity visually.

### Operational Modules
* **Process Manager**: View and manage running processes, similar to htop, with the ability to kill misbehaving processes.
* **Docker Manager**: Monitor Docker containers, view container states, and quickly drop into a shell inside running containers.
* **Service Manager**: Manage systemd services (start, stop, restart, enable, disable) and view service logs.
* **Log Viewer**: View and stream real-time logs from various sources across your server.
* **File Manager**: Navigate the server filesystem, read files, and write to restricted files with elevated privileges.

### Action & Security Layer
* **Security & Firewall**: Manage UFW rules, view firewall status, and monitor Fail2Ban jails.
* **Cron Manager**: Create, view, and safely edit cron jobs with real-time, human-readable translations of cron syntax.
* **SSL Certificate Manager**: Monitor SSL certificates, check expiration dates, and renew certificates easily.
* **User & SSH Key Manager**: Manage system users and their authorized SSH keys.

### Advanced Infrastructure Layer
* **Backup Manager**: Create and restore backups, configure backup schedules, and manage retention policies.
* **Database Manager**: View and manage database configurations.
* **Reverse Proxy Manager**: Configure and edit reverse proxy rules dynamically.
* **Secrets Editor**: Safely manage environment variables and secrets with a dedicated editor.
* **Deployments**: Trigger build and restart commands, and monitor deployment logs directly from the terminal.
* **Uptime Monitor**: Track external service uptime and response times, configurable via the central configuration file.

## Configuration

Vortex is fully configurable via `config.yaml`. By default, this file is located at `~/.config/vortex/config.yaml`. The configuration file allows you to customize:
* **Servers**: Define your fleet of servers with hostnames, ports, usernames, and authentication methods.
* **Appearance**: Customize themes, wallpapers, and animations.
* **Keybinds**: Remap the global navigation shortcuts to fit your workflow.
* **Uptime Targets**: Define the external URLs and endpoints that the Uptime module monitors.

## Installation

Ensure you have Go installed (1.20 or later recommended).

Clone the repository and build the binary:

```bash
git clone https://github.com/berkayyytech/vortex.git
cd vortex
go build -o bin/vps-manager.exe ./cmd/vps-manager
```

### Nix

If you have [Nix](https://nixos.org) with flakes enabled, you can run Vortex without cloning:

```bash
nix run github:berkayyytech/vortex
```

Or enter a development shell with Go and supporting tools:

```bash
nix develop
```

## How to Test

1. **Run the Application**: Execute the compiled binary:
   ```bash
   ./bin/vps-manager.exe
   ```

2. **Connecting to a Server**:
   * Navigate to the **Servers** tab using the sidebar (`[` or `]`) or the global hotkey (`Esc`).
   * Select a server and press `Enter` to establish an SSH connection.
   * If you haven't configured a real server yet, edit `~/.config/vortex/config.yaml` to include your actual VPS IP, username, and SSH key path.

3. **Testing Telemetry**:
   * Once connected, navigate to the **Dashboard** (`g` by default). You should see live, fluctuating telemetry data (CPU, RAM, Disk).

4. **Testing Modules**:
   * **Files**: Press `f` to open the File Manager. Navigate through directories using the arrow keys. Try editing a file to see the prompt and save behavior.
   * **Cron**: Press `x` to open the Cron Manager. Try adding a new cron schedule (e.g., `0 2 * * *`) and watch the human-readable translation appear live as you type.
   * **Backup**: Press `k` to open the Backup Manager. Try configuring a new destination path or schedule.

5. **Using the Command Palette**:
   * Press `Ctrl+P` to open the Command Palette.
   * Type fuzzy search terms like "Uptime" or "Secrets" and press `Enter` to jump instantly to those modules.

## Keyboard Shortcuts

Vortex is built to be used without a mouse.
* **[ / ]**: Cycle through the sidebar navigation.
* **Tab**: Toggle Split View to see the sidebar alongside the active module.
* **Ctrl+P**: Open the Command Palette for fuzzy-finding modules.
* **Esc**: Return to the Servers list (when input is not active).
* **Enter**: Confirm actions, connect to servers, or submit forms.
* **Ctrl+C**: Quit the application.

*Note: Global shortcuts are suspended automatically when you are typing in an input field or text editor.*

## License

MIT License. See LICENSE for more information.
