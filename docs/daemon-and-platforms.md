# Daemon & Platforms

Kronos supports three execution modes: foreground, background daemon, and
OS-native service. This guide covers each mode and the platform-specific
integration details.

## Running Kronos

### Foreground (`kronos start`)

Runs the scheduler in the current terminal session. Press `Ctrl+C` to stop
gracefully. Add `--tui` for an interactive terminal UI.

```bash
kronos start          # headless mode
kronos start --tui    # interactive terminal UI
```

### Background daemon (`kronos daemon`)

Self-daemonizes by spawning `kronos start` as a detached background process.
The parent exits immediately after printing the child PID.

```bash
kronos daemon
# Kronos daemon started (PID 12345)
```

On Unix, the child process runs in a new session (`setsid`). On Windows, it
uses `CREATE_NEW_PROCESS_GROUP` to detach from the console.

The daemon inherits the current config path. To use a custom config:

```bash
kronos daemon -f /path/to/kronos.yaml
```

### OS-native service (`kronos daemon install`)

Registers Kronos with the platform's service manager so it starts
automatically at login and restarts on failure. Kronos auto-detects the
platform and uses the appropriate integration:

| Platform | Service Manager | Detection |
|----------|----------------|-----------|
| macOS | launchd | `runtime.GOOS == "darwin"` |
| Linux | systemd (user) | `runtime.GOOS == "linux"` |
| Windows | Task Scheduler | `runtime.GOOS == "windows"` |

```bash
kronos daemon install      # install the service
kronos daemon uninstall    # remove the service
```

---

## macOS (launchd)

### Install

`kronos daemon install` creates a launchd property list at:

```
~/Library/LaunchAgents/com.kronos.agent.plist
```

The generated plist contains:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.kronos.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/kronos</string>
        <string>start</string>
        <string>-f</string>
        <string>/path/to/kronos.yaml</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>~/.cache/kronos/daemon.log</string>
    <key>StandardErrorPath</key>
    <string>~/.cache/kronos/daemon.log</string>
</dict>
</plist>
```

Key behaviors:

- **RunAtLoad**: starts automatically when the user logs in.
- **KeepAlive**: launchd restarts Kronos if it exits for any reason.
- **StandardOutPath / StandardErrorPath**: daemon stdout and stderr are
  written to `~/.cache/kronos/daemon.log`.

After writing the plist, Kronos runs `launchctl load <plist-path>` to
activate the service immediately.

### Uninstall

`kronos daemon uninstall` runs `launchctl unload <plist-path>` to stop the
service, then deletes the plist file.

---

## Linux (systemd)

### Install

`kronos daemon install` creates a systemd user service at:

```
~/.config/systemd/user/kronos.service
```

The generated unit file contains:

```ini
[Unit]
Description=Kronos Scheduler
After=default.target

[Service]
ExecStart=/path/to/kronos start -f /path/to/kronos.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

Key behaviors:

- **Restart=on-failure**: systemd restarts Kronos if it exits with a
  non-zero status.
- **RestartSec=5**: waits 5 seconds between restart attempts.
- **WantedBy=default.target**: the service starts when the user session
  is ready.

After writing the unit file, Kronos runs the following commands:

```bash
systemctl --user daemon-reload
systemctl --user enable kronos
systemctl --user start kronos
```

### Uninstall

`kronos daemon uninstall` runs:

```bash
systemctl --user stop kronos
systemctl --user disable kronos
```

Then deletes the service file and runs `systemctl --user daemon-reload` to
clean up. Errors from stop/disable are ignored (the service may not be
running or enabled).

### Useful commands

```bash
# Check service status
systemctl --user status kronos

# View service logs
journalctl --user -u kronos -f

# Restart the service
systemctl --user restart kronos
```

---

## Windows (Task Scheduler)

### Install

`kronos daemon install` creates a scheduled task named `KronosScheduler`
using the `schtasks` command:

```
schtasks /create /tn KronosScheduler /tr "\"kronos\" start -f \"kronos.yaml\"" /sc onlogon /rl limited /f
```

Key behaviors:

- **/sc onlogon**: triggers at user logon.
- **/rl limited**: runs at standard (non-elevated) privilege.
- **/f**: overwrites any existing task with the same name.

### Uninstall

`kronos daemon uninstall` deletes the task:

```
schtasks /delete /tn KronosScheduler /f
```

### Useful commands

```powershell
# List the Kronos task
schtasks /query /tn KronosScheduler /v

# Run the task manually
schtasks /run /tn KronosScheduler

# Open Task Scheduler GUI
taskschd.msc
```

---

## Single Instance (PID Lock)

Kronos uses a PID lock file to ensure only one instance runs at a time.
The lock file is located at:

| OS | Path |
|----|------|
| macOS / Linux | `~/.cache/kronos/kronos.pid` |
| Windows | `%LOCALAPPDATA%/kronos/kronos.pid` |

### How it works

1. On startup (`kronos start`), Kronos writes its PID to the lock file.
2. If a lock file already exists, Kronos reads the stored PID and checks
   whether the process is still alive (using signal 0 on Unix).
3. If the process is alive, Kronos exits with an error:
   `another kronos instance is running (PID <n>)`.
4. If the process is not alive (stale PID), Kronos removes the stale lock
   file and acquires a new lock.
5. On shutdown, Kronos deletes the lock file.

### Stale PID detection

A PID file is considered stale when:

- The stored PID does not correspond to a running process.
- The file contents cannot be parsed as an integer (corrupt file).

In both cases, the stale file is automatically cleaned up.

### Troubleshooting

If Kronos refuses to start due to a PID lock but no other instance is
running, the PID file may be left over from a crash. You can safely remove
it:

```bash
rm ~/.cache/kronos/kronos.pid
```

Or use `kronos doctor` to detect and report the situation.
