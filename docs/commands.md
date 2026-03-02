# Command Reference

Complete reference for every Kronos command. Commands are grouped by
function: setup, job management, execution, monitoring, data, and
maintenance.

## Global Flags

These flags are available on every command.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` / `-f` | string | `~/.config/kronos/kronos.yaml` | Config file path |
| `--no-color` | bool | `false` | Disable color output |
| `--json` | bool | `false` | Machine-readable JSON output |

---

## Setup

### kronos init

Create a starter `kronos.yaml` config file.

Usage: `kronos init [flags]`

No command-specific flags.

Examples:

```bash
# Create config at the default location
kronos init

# Create config at a custom path
kronos init -f /path/to/kronos.yaml
```

Notes:

- If a config file already exists at the target path, you are prompted to
  confirm before overwriting.
- The starter config contains a sample "hello" job that runs every minute.

---

### kronos doctor

Validate your Kronos setup.

Usage: `kronos doctor [flags]`

No command-specific flags.

Examples:

```bash
kronos doctor

# Check a specific config file
kronos doctor -f /path/to/kronos.yaml
```

Checks performed:

1. Config file exists
2. Config parses without errors
3. All jobs pass validation
4. Each job's command is found in `PATH`
5. Log directory is writable
6. Cache directory is writable
7. No other Kronos instance is running (PID lock)

Each check reports `[OK]`, `[WARN]`, or `[FAIL]`.

---

### kronos edit

Open the config file in your editor for manual editing.

Usage: `kronos edit [flags]`

No command-specific flags.

Examples:

```bash
# Open in $EDITOR (falls back to vi on Unix, notepad on Windows)
kronos edit

# Edit a specific config file
kronos edit -f /path/to/kronos.yaml
```

Notes:

- Uses `$EDITOR` environment variable. Falls back to `vi` on Unix and
  `notepad` on Windows.
- After saving, the config is validated. If errors are found you are prompted
  to re-edit.
- On successful save, prints the file path and job count.

---

## Job Management

### kronos add

Add a new job to the config.

Usage: `kronos add [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | — | Job name (required) |
| `--cmd` | string | — | Command to run (required) |
| `--schedule` | string | — | Cron schedule expression (required) |
| `--description` | string | `""` | Job description |
| `--dir` | string | `""` | Working directory |
| `--shell` | string | `""` | Shell to use |
| `--once` | bool | `false` | Run only once then disable |
| `--timeout` | string | `""` | Execution timeout (e.g. `30m`) |
| `--overlap` | string | `""` | Overlap policy: `skip` \| `allow` \| `queue` |
| `--on-failure` | string | `""` | Failure policy: `retry` \| `skip` \| `pause` |
| `--retry-count` | int | `0` | Number of retries on failure |
| `--tag` | string slice | `[]` | Tags (repeatable) |

Examples:

```bash
# Add a simple job
kronos add --name backup --cmd "pg_dump mydb > /tmp/db.sql" --schedule "@daily"

# Add a job with all options
kronos add \
  --name deploy \
  --cmd "rsync -avz /src/ /dst/" \
  --schedule "0 2 * * *" \
  --description "Nightly deploy" \
  --dir /opt/myapp \
  --shell bash \
  --timeout 30m \
  --overlap skip \
  --on-failure retry \
  --retry-count 3 \
  --tag prod --tag deploy
```

Notes:

- The job name must be unique within the config.
- The config is validated after adding the job. If validation fails the job is
  not saved.

---

### kronos remove

Remove a job from the config.

Usage: `kronos remove <name> [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--yes` / `-y` | bool | `false` | Skip confirmation prompt |

Examples:

```bash
# Remove with confirmation prompt
kronos remove backup

# Remove without confirmation
kronos remove backup --yes
```

Notes:

- Requires exactly one positional argument: the job name.
- Unless `--yes` is passed, you are prompted to confirm the deletion.

---

### kronos enable

Enable a previously disabled job.

Usage: `kronos enable <name>`

No command-specific flags.

Examples:

```bash
kronos enable backup
```

Notes:

- Requires exactly one positional argument: the job name.
- Sets the job's `enabled` field to `true` and saves the config.

---

### kronos disable

Disable a job so it no longer runs on schedule.

Usage: `kronos disable <name>`

No command-specific flags.

Examples:

```bash
kronos disable backup
```

Notes:

- Requires exactly one positional argument: the job name.
- Sets the job's `enabled` field to `false` and saves the config.
- A disabled job can still be triggered manually with `kronos run`.

---

### kronos pause-all

Disable all jobs at once.

Usage: `kronos pause-all`

No command-specific flags.

Examples:

```bash
kronos pause-all
```

Notes:

- Sets `enabled: false` on every job that is currently enabled.
- Prints the number of jobs that were paused.

---

### kronos resume-all

Enable all jobs at once.

Usage: `kronos resume-all`

No command-specific flags.

Examples:

```bash
kronos resume-all
```

Notes:

- Sets `enabled: true` on every job that is currently disabled.
- Prints the number of jobs that were resumed.

---

## Execution

### kronos start

Start the scheduler in the foreground.

Usage: `kronos start [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--tui` | bool | `false` | Launch interactive terminal UI |

Examples:

```bash
# Run in headless mode
kronos start

# Run with the interactive TUI
kronos start --tui

# Use a custom config
kronos start -f /path/to/kronos.yaml
```

Notes:

- Acquires a PID lock to prevent multiple instances from running
  simultaneously.
- In headless mode, press `Ctrl+C` to stop gracefully.
- In TUI mode, press `q` or `Ctrl+C` to quit. The TUI provides live job
  status, log streaming, and run history across three tabs.
- Watches the config file for changes and hot-reloads jobs automatically.
- Respects the `settings.shutdown_timeout` value for graceful shutdown.

---

### kronos run

Manually trigger a single job outside its schedule.

Usage: `kronos run <name> [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | `false` | Print what would run without executing |

Examples:

```bash
# Run a job immediately
kronos run backup

# Preview what would run
kronos run backup --dry-run
```

Notes:

- Requires exactly one positional argument: the job name.
- The run result is saved to the history database with trigger type "manual".
- `--dry-run` prints the command, working directory, shell, and timeout
  without actually executing anything.

---

### kronos daemon

Run Kronos as a background daemon.

Usage: `kronos daemon`

No command-specific flags.

Examples:

```bash
kronos daemon
```

Notes:

- Self-daemonizes by spawning `kronos start` as a detached background process.
- Prints the PID of the daemon process.
- The daemon inherits the current config path (via `-f` flag).
- Use `kronos daemon install` for OS-native service integration that persists
  across reboots.

---

### kronos daemon install

Install Kronos as an OS-native service.

Usage: `kronos daemon install`

No command-specific flags.

Examples:

```bash
kronos daemon install
```

Platform behavior:

| Platform | Service Type | Location |
|----------|-------------|----------|
| macOS | launchd | `~/Library/LaunchAgents/com.kronos.agent.plist` |
| Linux | systemd user service | `~/.config/systemd/user/kronos.service` |
| Windows | Task Scheduler | `KronosScheduler` task |

Notes:

- The service starts automatically at login.
- On macOS, the service restarts automatically via `KeepAlive`.
- On Linux, the service restarts on failure with a delay.
- On Windows, the task is triggered at user logon.

---

### kronos daemon uninstall

Remove the Kronos OS-native service.

Usage: `kronos daemon uninstall`

No command-specific flags.

Examples:

```bash
kronos daemon uninstall
```

Notes:

- Removes the service entry created by `kronos daemon install`.
- On macOS, unloads the launchd plist and deletes the file.
- On Linux, stops, disables, and removes the systemd unit file.
- On Windows, deletes the scheduled task.

---

## Monitoring

### kronos status

Show a status overview of all jobs.

Usage: `kronos status [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--stats` | bool | `false` | Show per-job metrics (runs, success rate, durations) |

Examples:

```bash
# Default status table
kronos status

# Status as JSON
kronos status --json

# Per-job metrics
kronos status --stats

# Metrics as JSON
kronos status --stats --json
```

Default output columns:

| Column | Description |
|--------|-------------|
| NAME | Job name |
| SCHEDULE | Cron expression |
| STATUS | `active` or `disabled` |
| LAST RUN | Timestamp of the most recent run |
| NEXT RUN | Computed next scheduled run time |
| LAST RESULT | `OK` with duration or `FAIL` with exit code |

With `--stats`, output columns:

| Column | Description |
|--------|-------------|
| NAME | Job name |
| RUNS | Total number of recorded runs |
| SUCCESS RATE | Percentage of successful runs |
| AVG DURATION | Average execution time |
| P95 DURATION | 95th percentile execution time |
| LAST FAILURE | Timestamp of the most recent failure |

Notes:

- Reads run history from the local database (`kronos.db`).
- With `--stats`, an aggregate summary line is printed after the table.

---

### kronos logs

View log output for a specific job.

Usage: `kronos logs <job> [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--follow` / `-f` | bool | `false` | Follow log output (like `tail -f`) |
| `--lines` / `-n` | int | `50` | Number of lines to show |

Examples:

```bash
# Show the last 50 lines
kronos logs backup

# Show the last 100 lines
kronos logs backup -n 100

# Stream new log output in real time
kronos logs backup --follow
```

Notes:

- Requires exactly one positional argument: the job name.
- Log files are stored in the configured log directory (default:
  `~/.cache/kronos/logs/<name>.log`).
- `--follow` uses filesystem notifications to stream new lines as they are
  written. Press `Ctrl+C` to stop.
- Note: the `-f` shorthand for `--follow` is a local flag and does not
  conflict with the global `-f` shorthand for `--config`.

---

## Data

### kronos list

List all configured jobs.

Usage: `kronos list [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--tag` | string | `""` | Filter jobs by tag |

Examples:

```bash
# List all jobs
kronos list

# Filter by tag
kronos list --tag prod

# Output as JSON
kronos list --json
```

Output columns:

| Column | Description |
|--------|-------------|
| NAME | Job name |
| SCHEDULE | Cron expression |
| ENABLED | `yes` or `no` |
| TAGS | Comma-separated tags |
| DESCRIPTION | Job description |

---

### kronos export

Export jobs to native scheduler formats.

Usage: `kronos export [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `crontab` | Export format: `crontab` \| `launchd` \| `systemd` |
| `--output` / `-o` | string | `""` (stdout) | Output file path |

Examples:

```bash
# Export to crontab format (stdout)
kronos export

# Export to a file
kronos export -o backup.cron

# Export as launchd plists
kronos export --format launchd -o jobs.plist

# Export as systemd units
kronos export --format systemd -o jobs.service
```

Notes:

- By default, exports to stdout. Use `-o` to write to a file.
- The `crontab` format produces standard 5-field cron lines.
- The `launchd` format produces macOS plist XML.
- The `systemd` format produces systemd timer and service units.

---

### kronos import

Import jobs from crontab or other formats.

Usage: `kronos import [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--from` | string | `crontab` | Import format (currently only `crontab`) |
| `--file` | string | `""` (stdin) | Input file path |

Examples:

```bash
# Import from a file
kronos import --file my-crontab.txt

# Import from current crontab via stdin
crontab -l | kronos import

# Import from a specific file with explicit format
kronos import --from crontab --file /etc/crontab
```

Notes:

- When `--file` is omitted, reads from stdin.
- Auto-generates job names from the command basename.
- Picks up `KEY=VALUE` environment variable lines.
- Skips `@reboot` entries with a warning.
- Handles system crontab format (6-field with username).
- Duplicate job names are skipped. Prints the count of added and skipped jobs.
- The merged config is validated before saving.

---

### kronos prune

Delete old history records from the database.

Usage: `kronos prune [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--older-than` | string | `""` | Delete records older than duration (e.g. `30d`, `2w`, `24h`) |
| `--keep` | int | `0` | Keep only the last N records per job |
| `--job` | string | `""` | Filter by job name |
| `--dry-run` | bool | `false` | Show what would be deleted without deleting |

Examples:

```bash
# Delete records older than 30 days
kronos prune --older-than 30d

# Keep only the last 50 records per job
kronos prune --keep 50

# Preview what would be deleted
kronos prune --older-than 2w --dry-run

# Prune a specific job
kronos prune --older-than 7d --job backup

# Combine both criteria
kronos prune --older-than 30d --keep 100

# JSON output
kronos prune --older-than 30d --json
```

Notes:

- At least one of `--older-than` or `--keep` is required.
- The `--older-than` flag supports day (`d`), week (`w`), and standard Go
  durations (`h`, `m`, `s`). The value must be positive.
- When both `--older-than` and `--keep` are used together with `--dry-run`,
  the reported count is an upper bound since the criteria may overlap.
- Supports `--json` output for scripting.

---

## Maintenance

### kronos update

Self-update Kronos to the latest release.

Usage: `kronos update`

No command-specific flags.

Examples:

```bash
kronos update
```

Notes:

- Downloads the latest release from GitHub.
- Compares the current version against the latest available version.
- Replaces the current binary in place.

---

### kronos version

Print version information.

Usage: `kronos version`

No command-specific flags.

Examples:

```bash
kronos version
```

Output format:

```
kronos version <version> (commit <sha>, built <date>)
```

Notes:

- Version, commit, and build date are set at compile time via `-ldflags`.
- During development, defaults to `version dev (commit none, built unknown)`.
