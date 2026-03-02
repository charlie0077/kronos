# Getting Started

## Install

### From source (requires Go 1.23+)

```bash
go install github.com/zhenchaochen/kronos@latest
```

### Binary download

Download the latest release for your platform from
[GitHub Releases](https://github.com/zhenchaochen/kronos/releases)
and place the binary in your `PATH`.

## Quick Start

Three steps to your first scheduled job:

### 1. Create a config file

```bash
kronos init
```

This writes a starter config to `~/.config/kronos/kronos.yaml` containing a
sample "hello" job that runs every minute:

```yaml
jobs:
  - name: hello
    cmd: echo "Hello from Kronos!"
    schedule: "@every 1m"
    description: "A sample job — edit or remove me"

settings:
  history_limit: 100
  log_max_size: 10
  log_max_files: 5
  shutdown_timeout: 30s
```

### 2. Edit the config

Open the config in your `$EDITOR` (falls back to `vi` on Unix, `notepad` on
Windows):

```bash
kronos edit
```

Or open the file directly in any editor. When using `kronos edit`, the config
is validated on save — if there are errors you are prompted to re-edit.

### 3. Start the scheduler

```bash
kronos start
```

Kronos runs in the foreground, executing jobs on their schedules. Press
`Ctrl+C` to stop gracefully.

## Interactive Mode

Launch the terminal UI with:

```bash
kronos start --tui
```

The TUI has three tabs (switch with `Tab` / `Shift+Tab`):

| Tab | Description |
|-----|-------------|
| **Jobs** | Live view of all jobs with status, next run, and last result. Press `r` to run a job manually, `e`/`d` to enable/disable, `p`/`R` to pause/resume all. |
| **Logs** | Streaming output for the selected job. |
| **History** | Run history with timestamps, durations, and exit codes. |

Press `q` or `Ctrl+C` to quit.

## Validate Setup

Run the built-in health checker:

```bash
kronos doctor
```

Example output:

```
 [OK]   Config file found: /home/user/.config/kronos/kronos.yaml
 [OK]   Config is valid (3 jobs)
 [OK]   All jobs pass validation
 [OK]   Job "backup-db": pg_dump found in PATH
 [WARN] Job "deploy": rsync not found in PATH
 [OK]   Log directory writable: /home/user/.cache/kronos/logs
 [OK]   Cache directory writable: /home/user/.cache/kronos
 [OK]   No other instance running
```

Each check reports `[OK]`, `[WARN]`, or `[FAIL]`. Fix any failures before
starting the scheduler.

## Run as a Service

Install Kronos as an OS-native service so it starts automatically at login:

```bash
kronos daemon install
```

This creates the appropriate service entry for your platform (launchd on
macOS, systemd on Linux, Task Scheduler on Windows). See
[Daemon & Platforms](./daemon-and-platforms.md) for details.

To remove the service:

```bash
kronos daemon uninstall
```
