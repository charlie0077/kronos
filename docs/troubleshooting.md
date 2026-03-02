# Troubleshooting

## Doctor Checks

Run `kronos doctor` to validate your setup. Each check reports `[OK]`,
`[WARN]`, or `[FAIL]`. Below is what each check means and how to fix
failures.

| Check | Level | Failure Meaning | Fix |
|-------|-------|----------------|-----|
| Config file found | FAIL | No config file at the expected path | Run `kronos init` to create one, or pass `-f` to point to an existing file |
| Config is valid | FAIL | YAML syntax error or invalid structure | Fix the YAML syntax; run `kronos edit` which validates on save |
| All jobs pass validation | FAIL | A job has invalid or missing fields (e.g. empty `name`, `cmd`, or `schedule`) | Check the specific field error in the output and correct it in the config |
| Job command found in PATH | WARN | The first word of a job's `cmd` is not in your `PATH` | Install the missing tool, or use an absolute path in `cmd` |
| Log directory writable | FAIL | Cannot create or write files in the log directory | Check permissions on `~/.cache/kronos/logs/` (or your custom `settings.log_dir`) |
| Cache directory writable | FAIL | Cannot create or write files in the cache directory | Check permissions on `~/.cache/kronos/` |
| No other instance running | WARN | A PID lock file exists with a live process | Stop the other instance, or remove a stale PID file (see below) |

---

## Common Issues

### "another kronos instance is running"

Kronos uses a PID lock file to prevent multiple instances. If the
previous process crashed without cleaning up, the lock file may be
stale.

```bash
# Check if the PID is actually running
cat ~/.cache/kronos/kronos.pid
ps -p <pid>

# If the process is not running, remove the stale lock
rm ~/.cache/kronos/kronos.pid
```

On Windows, the PID file is at `%LOCALAPPDATA%/kronos/kronos.pid`.

### "Permission denied" on daemon install

- **macOS**: Ensure `~/Library/LaunchAgents/` exists and is writable by
  your user. Kronos creates the directory automatically, but parent
  directory permissions may block it.
- **Linux**: Ensure `~/.config/systemd/user/` is writable. If systemd
  user services are not available, check that `systemctl --user` works.
- **Windows**: Task Scheduler requires standard user privileges. If
  running in a restricted environment, consult your system administrator.

### Jobs not running

1. Verify the job is enabled: `kronos list` shows `yes` in the ENABLED
   column.
2. Check the schedule syntax: run `kronos doctor` to validate.
3. Confirm the scheduler is running: `kronos status` shows job status
   and next run times.
4. Check job logs for errors: `kronos logs <name>`.

### Logs missing

Job logs are stored in the configured log directory:

- Default: `~/.cache/kronos/logs/<name>.log`
- Custom: set `settings.log_dir` in `kronos.yaml`

If the directory does not exist, Kronos creates it on first run. If
logs are still missing, run `kronos doctor` to check directory write
permissions.

### High disk usage from history

Kronos stores run history in a local database. To clean up old records:

```bash
# Delete records older than 30 days
kronos prune --older-than 30d

# Preview before deleting
kronos prune --older-than 30d --dry-run

# Keep only the last 50 records per job
kronos prune --keep 50
```

You can also set `settings.history_limit` in the config to cap the
number of records kept per job automatically.

---

## FAQ

### How do I run Kronos at login?

Install it as an OS-native service:

```bash
kronos daemon install
```

This creates the appropriate service entry for your platform (launchd on
macOS, systemd on Linux, Task Scheduler on Windows). See
[Daemon & Platforms](./daemon-and-platforms.md) for details.

### Where are my logs?

Job logs: `~/.cache/kronos/logs/` (or the path set in
`settings.log_dir`).

Daemon logs (when running as a service): `~/.cache/kronos/daemon.log`
on macOS. On Linux, use `journalctl --user -u kronos`.

### How do I migrate from crontab?

Pipe your existing crontab directly into Kronos:

```bash
crontab -l | kronos import
```

Or import from a file:

```bash
kronos import --file /path/to/crontab-backup.txt
```

See [Import & Export](./import-export.md) for details on how jobs are
named and merged.

### How do I back up my jobs?

Export to a crontab file:

```bash
kronos export -o backup.cron
```

Or export to other formats:

```bash
kronos export --format launchd -o backup.plist
kronos export --format systemd -o backup.service
```

### Can I run multiple Kronos instances?

No. Kronos uses a PID lock file (`~/.cache/kronos/kronos.pid`) to
enforce single-instance operation. If you need to manage separate sets
of jobs, use different config files and separate cache directories.
