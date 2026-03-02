# Import & Export

Kronos can export your jobs to native scheduler formats and import jobs
from existing crontab files.

## Export

Usage: `kronos export [--format <fmt>] [--output <file>]`

Supported formats: `crontab` (default), `launchd`, `systemd`.

Disabled jobs are skipped in all export formats. When `-o` is omitted,
output is written to stdout.

### Crontab format

Produces standard 5-field cron lines. Each job is preceded by a
`# kronos: <name>` comment. Descriptor schedules are mapped to their
5-field equivalents:

| Descriptor | Cron Equivalent |
|-----------|----------------|
| `@daily` / `@midnight` | `0 0 * * *` |
| `@hourly` | `0 * * * *` |
| `@weekly` | `0 0 * * 0` |
| `@monthly` | `0 0 1 * *` |
| `@yearly` / `@annually` | `0 0 1 1 *` |

Jobs with `@every` schedules cannot be represented in crontab format.
They are emitted as commented-out lines with a `WARNING` note:

```
# kronos: poller
# WARNING: @every not supported in cron
# @every 30s curl http://localhost/health
```

Environment variables and working directories are inlined into the
command string (e.g. `KEY=VAL cd /dir && cmd`).

Example output:

```cron
# kronos: backup
0 0 * * * pg_dump mydb > /backups/db.sql

# kronos: cleanup
0 * * * * find /tmp -mtime +7 -delete
```

### Launchd format

Produces one macOS plist XML block per job. Each block is preceded by a
`<!-- save as com.kronos.<name>.plist -->` comment.

- Cron schedules and descriptors use `StartCalendarInterval`.
- `@every` schedules use `StartInterval` (seconds).
- Commands are wrapped with `/bin/sh -c` in `ProgramArguments`.
- Working directories use the `WorkingDirectory` key.
- Environment variables use the `EnvironmentVariables` dict.

Example output:

```xml
<!-- save as com.kronos.backup.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.kronos.backup</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/sh</string>
        <string>-c</string>
        <string>pg_dump mydb > /backups/db.sql</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>0</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
</dict>
</plist>
```

### Systemd format

Produces a `.timer` and `.service` unit pair per job. Each unit is
preceded by a `# --- save as kronos-<name>.timer ---` or
`# --- save as kronos-<name>.service ---` comment.

- Cron schedules and descriptors use `OnCalendar`.
- `@every` schedules use `OnUnitActiveSec`.
- Services use `Type=oneshot` with `ExecStart=/bin/sh -c '<cmd>'`.
- Timers include `Persistent=true` so missed runs are caught up.
- Working directories use `WorkingDirectory`.
- Environment variables use `Environment=KEY=VAL` directives.

Example output:

```ini
# --- save as kronos-backup.timer ---
[Unit]
Description=Kronos timer for backup

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target

# --- save as kronos-backup.service ---
[Unit]
Description=Kronos service for backup

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'pg_dump mydb > /backups/db.sql'
```

---

## Import

Usage: `kronos import [--from crontab] [--file <path>]`

Currently only crontab import is supported. When `--file` is omitted,
reads from stdin.

```bash
# Import from a file
kronos import --file my-crontab.txt

# Import from current crontab via stdin
crontab -l | kronos import

# Import a system crontab (6-field format with username column)
kronos import --file /etc/crontab
```

### How it works

1. Blank lines and comments (`#`) are skipped.
2. `KEY=VALUE` lines are captured as environment variables and attached
   to all subsequent jobs.
3. Standard 5-field cron lines are parsed into schedule + command.
4. System crontab lines (6-field with username column) are detected
   automatically — the username field is stripped.
5. Descriptor schedules (`@daily`, `@hourly`, etc.) are preserved as-is.
6. Job names are auto-generated from the command basename, sanitized to
   lowercase alphanumeric characters and hyphens, and truncated to 30
   characters. Duplicates get a numeric suffix (e.g. `curl`, `curl-2`).

### Merge behavior

Imported jobs are merged into the existing config:

- If a job with the same name already exists, the import is **skipped**
  for that job.
- New jobs are appended to the config.
- The merged config is validated before saving.
- A summary is printed showing how many jobs were added and skipped.

### Skipped entries

- `@reboot` lines are skipped with a warning (Kronos does not support
  reboot triggers).
- Lines that cannot be parsed are skipped with a warning.

---

## Limitations

| Limitation | Details |
|-----------|---------|
| `@every` not exportable to crontab | Crontab has no interval syntax; these jobs are commented out with a warning |
| `@reboot` not importable | Kronos does not support run-at-boot triggers |
| Import format | Only `crontab` is currently supported as an import source |
| Cron field restrictions | Launchd export only handles simple integer cron fields (no ranges or step values) |
