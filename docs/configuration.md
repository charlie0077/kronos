# Configuration Reference

## File Location

Kronos looks for its config file at the OS-native config directory:

| OS | Default Path |
|----|-------------|
| macOS / Linux | `~/.config/kronos/kronos.yaml` |
| Windows | `%APPDATA%/kronos/kronos.yaml` |

Override with the `--config` (or `-f`) flag on any command:

```bash
kronos start --config /path/to/custom.yaml
kronos start -f /path/to/custom.yaml
```

## Complete Example

A fully annotated `kronos.yaml` showing every available field:

```yaml
jobs:
  - name: backup-db                      # required, must be unique
    description: "Backs up the database" # optional
    cmd: pg_dump mydb > /backups/db.sql  # required, shell command to run
    schedule: "@daily"                   # required, cron expression or descriptor
    dir: /opt/myapp                      # optional, working directory
    shell: bash                          # optional, shell interpreter
    enabled: true                        # optional, default: true
    once: false                          # optional, run once then disable
    timeout: 30m                         # optional, Go duration (e.g. 30s, 5m, 1h)
    overlap: skip                        # optional: skip | allow | queue
    on_failure: retry                    # optional: retry | skip | pause
    retry_count: 3                       # optional, used when on_failure is retry
    backoff: exponential                 # optional: exponential | fixed
    backoff_interval: 5s                 # optional, base interval for backoff
    tags: [db, prod]                     # optional, for filtering
    env:                                 # optional, environment variables
      PGPASSWORD: secret123

settings:
  history_limit: 100                     # runs kept in DB per job (default: 100)
  log_dir: ""                            # empty = OS default cache dir
  log_max_size: 10                       # MB per log file (default: 10)
  log_max_files: 5                       # rotated log files kept (default: 5)
  shutdown_timeout: 30s                  # graceful stop timeout (default: 30s)
```

## Job Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | yes | — | Unique identifier for the job |
| `description` | string | no | `""` | Human-readable description |
| `cmd` | string | yes | — | Shell command to execute |
| `schedule` | string | yes | — | Cron expression or descriptor |
| `dir` | string | no | `""` | Working directory for the command |
| `shell` | string | no | `""` | Shell interpreter (e.g. `bash`, `sh`, `zsh`) |
| `enabled` | bool | no | `true` | Whether the job is active |
| `once` | bool | no | `false` | Run once then auto-disable |
| `timeout` | duration | no | `""` (none) | Kill the job after this duration |
| `overlap` | string | no | `"allow"` | Overlap policy (see below) |
| `on_failure` | string | no | `""` | Failure policy (see below) |
| `retry_count` | int | no | `0` | Max retries when `on_failure` is `retry` |
| `backoff` | string | no | `""` | Backoff strategy: `exponential` or `fixed` |
| `backoff_interval` | duration | no | `""` | Base interval between retries |
| `tags` | list | no | `[]` | Arbitrary tags for filtering |
| `env` | map | no | `{}` | Environment variables passed to the command |

## Schedule Format

Kronos uses the [robfig/cron](https://pkg.go.dev/github.com/robfig/cron/v3)
parser with support for the following formats:

### 5-field cron

```
minute  hour  day-of-month  month  day-of-week
  *       *        *          *         *
```

Examples:

| Expression | Meaning |
|-----------|---------|
| `0 * * * *` | Every hour at minute 0 |
| `30 2 * * *` | Daily at 02:30 |
| `0 9 * * 1-5` | Weekdays at 09:00 |
| `*/15 * * * *` | Every 15 minutes |

### Descriptors

| Descriptor | Equivalent Cron |
|-----------|----------------|
| `@yearly` (or `@annually`) | `0 0 1 1 *` |
| `@monthly` | `0 0 1 * *` |
| `@weekly` | `0 0 * * 0` |
| `@daily` (or `@midnight`) | `0 0 * * *` |
| `@hourly` | `0 * * * *` |

### Interval

Run at a fixed interval from the time the scheduler starts:

```
@every <duration>
```

Examples: `@every 5m`, `@every 1h30m`, `@every 30s`

## Overlap Policies

Controls what happens when a job is scheduled to run while a previous
execution is still in progress.

| Policy | Behavior |
|--------|----------|
| `skip` | Skip the new run |
| `allow` | Run in parallel (default) |
| `queue` | Buffer one pending run (queue size 1) |

## Failure Policies

Controls what happens when a job exits with a non-zero status.

| Policy | Behavior |
|--------|----------|
| `retry` | Retry up to `retry_count` times with optional backoff |
| `skip` | Ignore the failure, continue scheduling |
| `pause` | Disable the job until manually re-enabled |

When using `retry`, configure the backoff strategy:

| Field | Values | Description |
|-------|--------|-------------|
| `backoff` | `exponential`, `fixed` | How the delay grows between retries |
| `backoff_interval` | Go duration (e.g. `5s`) | Base interval; exponential doubles each retry |

## Settings Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `history_limit` | int | `100` | Maximum run records kept per job in the database |
| `log_dir` | string | `""` (OS default) | Custom directory for job log files |
| `log_max_size` | int | `10` | Maximum size in MB per log file before rotation |
| `log_max_files` | int | `5` | Number of rotated log files to retain |
| `shutdown_timeout` | duration | `"30s"` | Time to wait for running jobs during graceful shutdown |

## Data Paths

Kronos stores its runtime data in the OS-native cache directory:

| File | macOS / Linux | Windows |
|------|--------------|---------|
| Config | `~/.config/kronos/kronos.yaml` | `%APPDATA%/kronos/kronos.yaml` |
| Database | `~/.cache/kronos/kronos.db` | `%LOCALAPPDATA%/kronos/kronos.db` |
| Logs | `~/.cache/kronos/logs/` | `%LOCALAPPDATA%/kronos/logs/` |
| PID lock | `~/.cache/kronos/kronos.pid` | `%LOCALAPPDATA%/kronos/kronos.pid` |

The log directory can be overridden with the `settings.log_dir` field. All
other paths follow the OS conventions provided by Go's `os.UserConfigDir()`
and `os.UserCacheDir()`.
