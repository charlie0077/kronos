# Kronos — Detailed Implementation Plan

## Overview

Kronos is a cross-platform cron CLI tool written in Go 1.23+ that replaces platform-specific schedulers (crontab, Task Scheduler, launchd) with a single binary. It features a bubbletea TUI, YAML-based config, and self-daemonization.

---

## Project Structure

```
github.com/zhenchaochen/kronos/
├── main.go
├── go.mod
├── go.sum
├── LICENSE                          # MIT
├── .goreleaser.yaml
├── .github/
│   └── workflows/
│       ├── ci.yaml                  # lint + test + build matrix
│       └── release.yaml             # goreleaser on tag push
├── cmd/
│   ├── root.go                      # cobra root, global flags (--no-color, --json)
│   ├── init.go                      # generate starter kronos.yaml
│   ├── add.go                       # add job via flags
│   ├── remove.go                    # remove job (with confirmation)
│   ├── edit.go                      # open YAML in $EDITOR, validate on save
│   ├── list.go                      # list jobs (table or --json)
│   ├── status.go                    # quick status overview (table or --json)
│   ├── run.go                       # manual trigger (--dry-run support)
│   ├── start.go                     # foreground scheduler + optional --tui
│   ├── daemon.go                    # daemon / daemon install / daemon uninstall
│   ├── enable.go                    # enable/disable a job
│   ├── pause.go                     # pause-all / resume-all
│   ├── doctor.go                    # validate setup
│   ├── update.go                    # self-update from GitHub releases
│   └── version.go                   # version info
├── internal/
│   ├── config/
│   │   ├── config.go                # YAML loading, merging, validation
│   │   ├── config_test.go
│   │   ├── model.go                 # Job, Settings structs
│   │   ├── paths.go                 # OS-native dir resolution
│   │   └── writer.go                # comment-preserving YAML write-back
│   ├── scheduler/
│   │   ├── scheduler.go             # robfig/cron wrapper, job lifecycle
│   │   ├── scheduler_test.go
│   │   ├── overlap.go               # skip/allow/queue policy
│   │   └── once.go                  # once-job detection + auto-remove
│   ├── runner/
│   │   ├── runner.go                # shell execution, env, cwd, timeout
│   │   ├── runner_test.go
│   │   ├── shell.go                 # shell detection + per-job shell config
│   │   ├── signal.go                # SIGTERM forwarding
│   │   └── failure.go               # retry/skip/pause + backoff strategies
│   ├── store/
│   │   ├── store.go                 # bbolt wrapper
│   │   ├── store_test.go
│   │   ├── history.go               # run history (manual/scheduled tagged)
│   │   └── lock.go                  # PID file lock
│   ├── logger/
│   │   ├── logger.go                # timestamped file logging
│   │   ├── logger_test.go
│   │   └── rotate.go                # log rotation (5x10MB)
│   ├── platform/
│   │   ├── detect.go                # OS detection
│   │   ├── launchd.go               # macOS plist generation
│   │   ├── systemd.go               # Linux service file generation
│   │   ├── schtasks.go              # Windows Task Scheduler
│   │   └── daemon.go                # self-daemonize + PID management
│   ├── watcher/
│   │   ├── watcher.go               # fsnotify YAML hot reload
│   │   └── watcher_test.go
│   ├── updater/
│   │   └── updater.go               # self-update from GitHub releases
│   └── ui/
│       ├── app.go                   # bubbletea root model
│       ├── tabs.go                  # tab bar component
│       ├── jobs.go                  # Jobs tab view
│       ├── logs.go                  # Logs tab (live tail)
│       ├── history.go               # History tab
│       ├── statusbar.go             # bottom key hints
│       └── styles.go                # lipgloss theme + NO_COLOR
└── testdata/
    └── kronos.yaml                  # fixture for tests
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/robfig/cron/v3` | Cron scheduler |
| `go.etcd.io/bbolt` | Embedded key-value store |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components (table, tabs) |
| `gopkg.in/yaml.v3` | YAML with comment preservation |
| `github.com/fsnotify/fsnotify` | File watching for hot reload |
| `gopkg.in/natefinch/lumberjack.v2` | Log rotation |

---

## Data Model (kronos.yaml)

```yaml
jobs:
  - name: backup-db
    description: "Backs up the production database"
    cmd: pg_dump mydb > /backups/db.sql
    schedule: "@daily"
    dir: /opt/myapp
    shell: bash
    enabled: true
    once: false
    timeout: 30m
    overlap: skip          # skip | allow | queue
    on_failure: retry      # retry | skip | pause
    retry_count: 3
    backoff: exponential   # exponential | fixed
    backoff_interval: 5s   # base interval
    tags: [db, prod]
    env:
      PGPASSWORD: secret123

settings:
  history_limit: 100
  log_dir: ""              # empty = OS default
  log_max_size: 10         # MB per file
  log_max_files: 5
  shutdown_timeout: 30s
```

---

## CLI Commands

```
kronos init                          # generate starter YAML
kronos add --name X --cmd Y --schedule Z [--once] [--tag T]...
kronos remove <name>                 # with confirmation prompt
kronos edit                          # open YAML in $EDITOR
kronos list [--json] [--tag T]       # table or JSON
kronos status [--json]               # quick overview
kronos run <name> [--dry-run]        # manual trigger
kronos start [--tui] [-f file.yaml]  # foreground scheduler
kronos daemon                        # self-daemonize
kronos daemon install                # OS-native service
kronos daemon uninstall
kronos enable <name>
kronos disable <name>
kronos pause-all
kronos resume-all
kronos doctor                        # validate everything
kronos update                        # self-update
kronos version
```

---

## Phase 1: Foundation

**Goal:** Compilable skeleton with config loading and validation.

### Step 1.1 — Go module + main.go + cobra root

- `go mod init github.com/zhenchaochen/kronos`
- `main.go`: just calls `cmd.Execute()`
- `cmd/root.go`: cobra root command with:
  - `--config` / `-f` flag (default: `~/.config/kronos/kronos.yaml`)
  - `--no-color` flag (sets `NO_COLOR=1` env)
  - `--json` flag (machine-readable output)
  - PersistentPreRunE that loads config into a global variable
- `cmd/version.go`: prints version/commit/date (set via ldflags)
- Install deps: `cobra`

### Step 1.2 — Config model (`internal/config/model.go`)

```go
type Config struct {
    Jobs     []Job    `yaml:"jobs"`
    Settings Settings `yaml:"settings"`
}

type Job struct {
    Name            string            `yaml:"name"`
    Description     string            `yaml:"description,omitempty"`
    Cmd             string            `yaml:"cmd"`
    Schedule        string            `yaml:"schedule"`
    Dir             string            `yaml:"dir,omitempty"`
    Shell           string            `yaml:"shell,omitempty"`
    Enabled         *bool             `yaml:"enabled,omitempty"`       // default true
    Once            bool              `yaml:"once,omitempty"`
    Timeout         string            `yaml:"timeout,omitempty"`       // duration string
    Overlap         string            `yaml:"overlap,omitempty"`       // skip|allow|queue
    OnFailure       string            `yaml:"on_failure,omitempty"`    // retry|skip|pause
    RetryCount      int               `yaml:"retry_count,omitempty"`
    Backoff         string            `yaml:"backoff,omitempty"`       // exponential|fixed
    BackoffInterval string            `yaml:"backoff_interval,omitempty"`
    Tags            []string          `yaml:"tags,omitempty,flow"`
    Env             map[string]string `yaml:"env,omitempty"`
}

type Settings struct {
    HistoryLimit    int    `yaml:"history_limit,omitempty"`
    LogDir          string `yaml:"log_dir,omitempty"`
    LogMaxSize      int    `yaml:"log_max_size,omitempty"`      // MB
    LogMaxFiles     int    `yaml:"log_max_files,omitempty"`
    ShutdownTimeout string `yaml:"shutdown_timeout,omitempty"`
}
```

- `Job.IsEnabled()` helper: returns `true` if `Enabled` is nil or `*Enabled == true`
- `Job.TimeoutDuration()` helper: parses timeout string to `time.Duration`
- `Settings` defaults: `history_limit=100`, `log_max_size=10`, `log_max_files=5`, `shutdown_timeout=30s`

### Step 1.3 — OS path resolver (`internal/config/paths.go`)

```go
func ConfigDir() string   // os.UserConfigDir() + "/kronos"
func CacheDir() string    // os.UserCacheDir() + "/kronos"
func DefaultConfigPath() string  // ConfigDir() + "/kronos.yaml"
func LogDir(settings Settings) string  // settings.LogDir or CacheDir()+"/logs"
func DBPath() string      // CacheDir() + "/kronos.db"
func PIDPath() string     // CacheDir() + "/kronos.pid"
```

- Use `os.UserConfigDir()` and `os.UserCacheDir()` for cross-platform paths
- Create directories with `os.MkdirAll(dir, 0o755)` as needed

### Step 1.4 — YAML parser (`internal/config/config.go`)

```go
func Load(path string) (*Config, error)          // read file + yaml.Unmarshal
func LoadWithNode(path string) (*Config, *yaml.Node, error)  // preserves AST for write-back
func ApplyDefaults(cfg *Config)                   // fill in default values
```

- Use `gopkg.in/yaml.v3` for comment preservation
- `Load` reads the file, unmarshals, applies defaults, returns config
- `LoadWithNode` also returns the raw `yaml.Node` tree for comment-preserving writes

### Step 1.5 — Validator (`internal/config/config.go`)

```go
func Validate(cfg *Config) []error
```

Validates:
- Each job has a non-empty `name` (unique across all jobs)
- Each job has a non-empty `cmd`
- Each job has a valid `schedule` (parse with `cron.ParseStandard`)
- `overlap` is one of: `skip`, `allow`, `queue`, or empty
- `on_failure` is one of: `retry`, `skip`, `pause`, or empty
- `backoff` is one of: `exponential`, `fixed`, or empty
- `timeout` parses as a valid `time.Duration` if set
- `backoff_interval` parses as a valid `time.Duration` if set
- `shutdown_timeout` parses as a valid `time.Duration` if set
- `retry_count >= 0` when `on_failure == retry`
- No duplicate job names

### Step 1.6 — Comment-preserving writer (`internal/config/writer.go`)

```go
func Save(path string, cfg *Config, node *yaml.Node) error
```

- Takes the original `yaml.Node` tree and updates values in-place
- Marshals back with `yaml.NewEncoder` to preserve comments
- Falls back to full marshal if node is nil (new file)

### Step 1.7 — Tests (`internal/config/config_test.go`)

- Test loading a valid YAML
- Test validation catches missing name, invalid schedule, invalid overlap, duplicates
- Test default application
- Test OS path functions return non-empty strings

### Verification

```bash
go build ./...
go test ./internal/config/...
```

---

## Phase 2: Core Engine

**Goal:** Job execution, scheduling, persistence, and logging all working.

### Step 2.1 — Shell detection + execution (`internal/runner/shell.go`, `runner.go`)

**shell.go:**
```go
func DetectShell() string  // checks $SHELL, falls back to "sh" (unix) or "cmd" (windows)
func ShellCommand(shell, cmd string) *exec.Cmd
    // unix: exec.Command(shell, "-c", cmd)
    // windows: exec.Command("cmd", "/C", cmd)
```

**runner.go:**
```go
type RunResult struct {
    ExitCode  int
    Output    string           // combined stdout+stderr (capped at 64KB for storage)
    StartTime time.Time
    EndTime   time.Time
    Error     error
}

type Runner struct {
    logger *logger.Logger      // optional, for streaming output to log file
}

func (r *Runner) Run(ctx context.Context, job config.Job) RunResult
```

- Creates command via `ShellCommand`
- Sets `cmd.Dir` if `job.Dir` is set
- Sets `cmd.Env` by merging `os.Environ()` + `job.Env` + `KRONOS_JOB_NAME=<name>`
- Creates timeout context from `job.TimeoutDuration()` if set
- Captures combined stdout+stderr via `cmd.CombinedOutput()` or pipes
- Streams output to logger if available
- Returns `RunResult` with timing and exit code

### Step 2.2 — Signal forwarding (`internal/runner/signal.go`)

```go
func ForwardSignals(cmd *exec.Cmd) func()
```

- Listens for SIGINT, SIGTERM on a signal channel
- Forwards received signal to the child process via `cmd.Process.Signal()`
- Returns a cleanup function to stop the listener
- On Windows: uses `cmd.Process.Kill()` since signals work differently

### Step 2.3 — Failure handler (`internal/runner/failure.go`)

```go
type FailureHandler struct{}

func (fh *FailureHandler) Handle(ctx context.Context, job config.Job, run func(context.Context) RunResult) RunResult
```

- If `on_failure == "retry"`:
  - Retry up to `retry_count` times
  - `backoff == "exponential"`: wait `backoff_interval * 2^attempt`
  - `backoff == "fixed"`: wait `backoff_interval` each time
  - Return last result
- If `on_failure == "skip"`: run once, return result regardless
- If `on_failure == "pause"`: run once, if failed mark job for disabling (return a flag)
- Default (empty): same as "skip"

### Step 2.4 — Scheduler (`internal/scheduler/scheduler.go`)

```go
type Scheduler struct {
    cron     *cron.Cron
    runner   *runner.Runner
    store    *store.Store
    logger   *logger.Manager
    mu       sync.Mutex
    jobs     map[string]cron.EntryID   // job name → cron entry ID
    running  map[string]bool           // overlap tracking
    onUpdate func(jobName string)      // callback for TUI refresh
}

func New(runner *runner.Runner, store *store.Store, logger *logger.Manager) *Scheduler
func (s *Scheduler) LoadJobs(jobs []config.Job) error   // add all jobs to cron
func (s *Scheduler) Start()
func (s *Scheduler) Stop(ctx context.Context)            // graceful with timeout
func (s *Scheduler) RunJob(name string) error            // manual trigger
func (s *Scheduler) GetEntries() []ScheduledJob          // for TUI/list
func (s *Scheduler) UpdateJobs(jobs []config.Job) error  // hot-reload: diff + add/remove
func (s *Scheduler) PauseAll()
func (s *Scheduler) ResumeAll()
```

- Each job wraps execution in overlap policy check
- After execution, stores result in bbolt via `store.SaveRun()`
- `GetEntries()` returns job info with next/prev run times from cron

### Step 2.5 — Overlap policy (`internal/scheduler/overlap.go`)

```go
func (s *Scheduler) wrapWithOverlapPolicy(job config.Job, fn func()) func()
```

- `skip`: check `s.running[name]`, skip if true
- `allow`: always run (no check)
- `queue`: use a per-job channel (buffered size 1) to queue next run
- Sets `s.running[name] = true` before exec, `false` after

### Step 2.6 — Once detection (`internal/scheduler/once.go`)

```go
func (s *Scheduler) handleOnceJob(name string)
```

- After a once-job executes successfully, remove it from the scheduler
- Optionally mark it as disabled in config (or remove entirely)

### Step 2.7 — bbolt store (`internal/store/store.go`, `history.go`)

**store.go:**
```go
type Store struct {
    db *bbolt.DB
}

func Open(path string) (*Store, error)
func (s *Store) Close() error
```

Buckets: `runs` (key: `<jobname>/<timestamp>`)

**history.go:**
```go
type RunRecord struct {
    JobName   string        `json:"job_name"`
    StartTime time.Time     `json:"start_time"`
    EndTime   time.Time     `json:"end_time"`
    ExitCode  int           `json:"exit_code"`
    Output    string        `json:"output"`       // truncated
    Trigger   string        `json:"trigger"`      // "scheduled" | "manual"
    Success   bool          `json:"success"`
}

func (s *Store) SaveRun(record RunRecord) error
func (s *Store) GetRuns(jobName string, limit int) ([]RunRecord, error)
func (s *Store) GetAllRuns(limit int) ([]RunRecord, error)
func (s *Store) GetLastRun(jobName string) (*RunRecord, error)
func (s *Store) PruneHistory(jobName string, keepN int) error
```

- Store records as JSON-encoded values
- Key format: `jobname/2006-01-02T15:04:05.000Z` for chronological ordering
- `PruneHistory` keeps only the last N records per job

### Step 2.8 — PID lock (`internal/store/lock.go`)

```go
type PIDLock struct {
    path string
}

func NewPIDLock(path string) *PIDLock
func (l *PIDLock) Acquire() error      // write PID, fail if already locked
func (l *PIDLock) Release() error      // remove PID file
func (l *PIDLock) IsLocked() (bool, int, error)  // check if locked, return PID
```

- Write current PID to file
- On acquire: check if existing PID is still running (`os.FindProcess` + signal 0)
- Stale PID files (process dead) are automatically cleaned up

### Step 2.9 — Logger (`internal/logger/logger.go`, `rotate.go`)

**logger.go:**
```go
type Logger struct {
    writer io.WriteCloser
    name   string
}

type Manager struct {
    logDir   string
    maxSize  int  // MB
    maxFiles int
    loggers  map[string]*Logger
}

func NewManager(logDir string, maxSize, maxFiles int) *Manager
func (m *Manager) GetLogger(jobName string) *Logger
func (l *Logger) Write(p []byte) (n int, err error)   // prepends timestamp
func (l *Logger) Tail(n int) ([]string, error)          // last N lines for TUI
```

**rotate.go:**
- Uses `lumberjack.Logger` under the hood:
  ```go
  &lumberjack.Logger{
      Filename:   filepath.Join(logDir, jobName+".log"),
      MaxSize:    maxSize,   // MB
      MaxBackups: maxFiles,
      Compress:   false,
  }
  ```

### Step 2.10 — Tests

- `runner_test.go`: test echo command, timeout, env injection, exit code capture
- `scheduler_test.go`: test job loading, overlap skip policy, start/stop
- `store_test.go`: test save/get/prune runs, PID lock acquire/release
- `logger_test.go`: test timestamped writing, log file creation

### Verification

```bash
go build ./...
go test ./...
```

---

## Phase 3: CLI Commands

**Goal:** All CLI commands functional.

### Step 3.1 — `kronos init` (`cmd/init.go`)

- Check if config file already exists, prompt to overwrite
- Write starter `kronos.yaml`:
  ```yaml
  # Kronos configuration
  # Docs: https://github.com/zhenchaochen/kronos

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
- Create config directory if needed
- Print path of created file

### Step 3.2 — `kronos add` (`cmd/add.go`)

Flags:
- `--name` (required)
- `--cmd` (required)
- `--schedule` (required)
- `--description`
- `--dir`
- `--shell`
- `--once`
- `--timeout`
- `--overlap`
- `--on-failure`
- `--retry-count`
- `--tag` (repeatable)

Logic:
- Load config + node
- Validate new job
- Check name uniqueness
- Append to jobs list
- Save with comment preservation
- Print confirmation

### Step 3.3 — `kronos remove` (`cmd/remove.go`)

- Arg: job name
- Load config
- Find job by name (error if not found)
- Prompt "Remove job 'X'? [y/N]" (skip with `--yes` flag)
- Remove from config
- Save

### Step 3.4 — `kronos edit` (`cmd/edit.go`)

- Open config file in `$EDITOR` (fallback: `vi` on unix, `notepad` on windows)
- After editor closes, reload and validate
- If validation fails, ask to re-edit or abort
- Print summary of changes

### Step 3.5 — `kronos list` (`cmd/list.go`)

- Load config
- Filter by `--tag` if provided
- Table format (default):
  ```
  NAME        SCHEDULE    ENABLED  TAGS        DESCRIPTION
  backup-db   @daily      yes      db,prod     Backs up the production database
  cleanup     0 2 * * *   yes      ops         Clean temp files
  ```
- `--json`: output as JSON array

### Step 3.6 — `kronos status` (`cmd/status.go`)

- Load config + store
- For each job, get last run from store
- Table format:
  ```
  NAME        SCHEDULE    STATUS   LAST RUN             NEXT RUN             LAST RESULT
  backup-db   @daily      active   2026-03-01 02:00:00  2026-03-02 02:00:00  OK (1.2s)
  cleanup     0 2 * * *   paused   2026-03-01 02:00:00  —                    FAIL (exit 1)
  ```
- `--json`: output as JSON

### Step 3.7 — `kronos run` (`cmd/run.go`)

- Arg: job name
- `--dry-run`: print what would run without executing
- Load config, find job
- Create runner, execute job
- Store result with `trigger: "manual"`
- Print result (exit code, duration, output snippet)

### Step 3.8 — `kronos enable` / `kronos disable` (`cmd/enable.go`)

- Arg: job name
- Load config + node
- Set `enabled: true` or `enabled: false`
- Save with comment preservation
- Print confirmation

### Step 3.9 — `kronos pause-all` / `kronos resume-all` (`cmd/pause.go`)

- `pause-all`: set all jobs `enabled: false`, save
- `resume-all`: set all jobs `enabled: true`, save
- Print count of affected jobs

### Step 3.10 — `kronos start` (`cmd/start.go`)

- `--tui`: launch bubbletea TUI (Phase 4)
- `-f` / `--config`: override config path
- Without `--tui`:
  - Print "Kronos started. Press Ctrl+C to stop."
  - Load config, create scheduler, runner, store, logger
  - Acquire PID lock
  - Start scheduler
  - Block on signal (SIGINT, SIGTERM)
  - Graceful shutdown with `settings.shutdown_timeout`
  - Release PID lock

### Step 3.11 — `kronos doctor` (`cmd/doctor.go`)

Checks:
- Config file exists and is valid YAML
- All jobs pass validation
- For each job, check if command is in PATH (first word of `cmd`)
- Check if log directory is writable
- Check if cache directory is writable
- Check if no other kronos instance is running (PID lock)
- Print results with checkmarks/X marks:
  ```
  [OK] Config file found: ~/.config/kronos/kronos.yaml
  [OK] Config is valid (3 jobs)
  [OK] Job "backup-db": pg_dump found in PATH
  [WARN] Job "custom-script": /opt/bin/myscript not found
  [OK] Log directory writable
  [OK] No other instance running
  ```

### Step 3.12 — `kronos version` (`cmd/version.go`)

- Print: `kronos version 0.1.0 (commit abc1234, built 2026-03-02)`
- Variables set via ldflags at build time

### Verification

```bash
go build ./...
go test ./...
# Manual test:
./kronos init
./kronos add --name test --cmd "echo hello" --schedule "@every 10s"
./kronos list
./kronos doctor
./kronos run test
./kronos start  # Ctrl+C to stop
```

---

## Phase 4: TUI

**Goal:** Interactive terminal UI with jobs, logs, and history tabs.

### Step 4.1 — Styles (`internal/ui/styles.go`)

```go
var (
    ActiveTabStyle   lipgloss.Style
    InactiveTabStyle lipgloss.Style
    HeaderStyle      lipgloss.Style
    SelectedRowStyle lipgloss.Style
    StatusBarStyle   lipgloss.Style
    SuccessStyle     lipgloss.Style  // green
    ErrorStyle       lipgloss.Style  // red
    WarningStyle     lipgloss.Style  // yellow
    MutedStyle       lipgloss.Style  // gray
)

func InitStyles(noColor bool)  // respect NO_COLOR env + --no-color flag
```

### Step 4.2 — App model (`internal/ui/app.go`)

```go
type Model struct {
    tabs       []string              // ["Jobs", "Logs", "History"]
    activeTab  int
    jobsModel  JobsModel
    logsModel  LogsModel
    histModel  HistoryModel
    scheduler  *scheduler.Scheduler
    store      *store.Store
    logMgr     *logger.Manager
    config     *config.Config
    width      int
    height     int
    quitting   bool
}

func NewModel(sched *scheduler.Scheduler, store *store.Store, logMgr *logger.Manager, cfg *config.Config) Model
func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

- Tab/Shift+Tab switches tabs
- `q` or Ctrl+C quits
- Delegates to active tab's Update/View
- Auto-refresh tick every 1 second

### Step 4.3 — Tab bar (`internal/ui/tabs.go`)

```go
func RenderTabBar(tabs []string, active int, width int) string
```

- Renders horizontal tab bar with active tab highlighted
- Example: `[ Jobs ] | Logs | History`

### Step 4.4 — Jobs tab (`internal/ui/jobs.go`)

```go
type JobsModel struct {
    jobs     []JobRow
    cursor   int
    width    int
    height   int
}

type JobRow struct {
    Name     string
    Schedule string
    Enabled  bool
    LastRun  string
    NextRun  string
    Status   string  // "running", "idle", "disabled"
}
```

- Table with columns: Name, Schedule, Status, Last Run, Next Run
- Up/Down arrow to navigate
- `r` — run selected job manually
- `e` — enable selected job
- `d` — disable selected job
- `p` — pause all
- Color-code status (green=running, gray=idle, red=disabled)

### Step 4.5 — Logs tab (`internal/ui/logs.go`)

```go
type LogsModel struct {
    jobName  string
    lines    []string
    offset   int        // scroll offset
    width    int
    height   int
}
```

- Shows live tail of selected job's log file
- Up/Down to scroll
- Auto-scrolls to bottom on new output
- `j`/`k` for vim-style scrolling
- Reads from `logger.Manager.GetLogger(jobName).Tail(height)`

### Step 4.6 — History tab (`internal/ui/history.go`)

```go
type HistoryModel struct {
    runs    []store.RunRecord
    cursor  int
    filter  string    // job name filter, empty = all
    width   int
    height  int
}
```

- Table: Job, Start Time, Duration, Result, Trigger
- Up/Down to navigate
- `/` to filter by job name
- Color: green for success, red for failure
- Shows "manual" or "scheduled" trigger badge

### Step 4.7 — Status bar (`internal/ui/statusbar.go`)

```go
func RenderStatusBar(activeTab int, width int) string
```

- Context-sensitive key hints based on active tab:
  - Jobs: `[r]un [e]nable [d]isable [p]ause-all  tab:switch  q:quit`
  - Logs: `[j/k]scroll  tab:switch  q:quit`
  - History: `[/]filter  tab:switch  q:quit`

### Step 4.8 — Wire TUI into `kronos start --tui`

- In `cmd/start.go`, when `--tui` is set:
  - Create all dependencies (scheduler, store, logger)
  - Create `ui.NewModel(...)`
  - Run `tea.NewProgram(model, tea.WithAltScreen()).Run()`
  - On exit, graceful shutdown

### Step 4.9 — NO_COLOR support

- Check `NO_COLOR` env var and `--no-color` flag
- Pass to `ui.InitStyles(noColor)`
- lipgloss has built-in `HasDarkBackground()` for adaptive colors

### Verification

```bash
go build ./...
./kronos start --tui
# Verify: tabs switch, jobs show, logs tail, history displays
# Verify: r/e/d keys work on Jobs tab
# Verify: NO_COLOR=1 kronos start --tui shows no colors
```

---

## Phase 5: Platform & Daemon

**Goal:** Daemon mode, OS-native service installation, YAML hot reload.

### Step 5.1 — Self-daemonize (`internal/platform/daemon.go`)

```go
func Daemonize(exe string, args []string) error
```

- Fork the current process with `os/exec`:
  ```go
  cmd := exec.Command(exe, args...)
  cmd.Stdout = nil  // detach
  cmd.Stderr = nil
  cmd.Stdin = nil
  cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}  // unix only
  cmd.Start()
  ```
- Write child PID to PID file
- Parent exits immediately
- Used by `kronos daemon` (without install/uninstall)

### Step 5.2 — Platform detection (`internal/platform/detect.go`)

```go
type Platform string

const (
    PlatformMacOS   Platform = "macos"
    PlatformLinux   Platform = "linux"
    PlatformWindows Platform = "windows"
)

func Detect() Platform  // runtime.GOOS based
```

### Step 5.3 — macOS launchd (`internal/platform/launchd.go`)

```go
func InstallLaunchd(exePath, configPath string) error
func UninstallLaunchd() error
```

- Generate plist at `~/Library/LaunchAgents/com.kronos.agent.plist`:
  ```xml
  <?xml version="1.0" encoding="UTF-8"?>
  <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" ...>
  <plist version="1.0">
  <dict>
    <key>Label</key><string>com.kronos.agent</string>
    <key>ProgramArguments</key>
    <array>
      <string>/path/to/kronos</string>
      <string>start</string>
      <string>-f</string>
      <string>/path/to/kronos.yaml</string>
    </array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
    <key>StandardOutPath</key><string>~/.cache/kronos/daemon.log</string>
    <key>StandardErrorPath</key><string>~/.cache/kronos/daemon.log</string>
  </dict>
  </plist>
  ```
- `launchctl load/unload` the plist

### Step 5.4 — Linux systemd (`internal/platform/systemd.go`)

```go
func InstallSystemd(exePath, configPath string) error
func UninstallSystemd() error
```

- Generate unit at `~/.config/systemd/user/kronos.service`:
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
- `systemctl --user enable/disable/start/stop kronos`

### Step 5.5 — Windows Task Scheduler (`internal/platform/schtasks.go`)

```go
func InstallSchtasks(exePath, configPath string) error
func UninstallSchtasks() error
```

- Use `schtasks /create` with XML or command-line flags
- Run at logon, restart on failure

### Step 5.6 — `kronos daemon` commands (`cmd/daemon.go`)

```go
// kronos daemon         — self-daemonize (fork + PID file)
// kronos daemon install — create OS-native service
// kronos daemon uninstall — remove OS-native service
```

- `daemon` (no subcommand): call `platform.Daemonize()`
- `daemon install`: detect platform, call appropriate installer
- `daemon uninstall`: detect platform, call appropriate uninstaller
- Print status messages

### Step 5.7 — YAML hot reload (`internal/watcher/watcher.go`)

```go
type Watcher struct {
    path      string
    scheduler *scheduler.Scheduler
    onChange  func(*config.Config)
    done      chan struct{}
}

func New(path string, sched *scheduler.Scheduler) *Watcher
func (w *Watcher) Start() error
func (w *Watcher) Stop()
```

- Uses `fsnotify` to watch the config file
- On `Write`/`Create` event:
  - Debounce (100ms) to avoid double-fires
  - Reload config
  - Validate
  - If valid: diff jobs, call `scheduler.UpdateJobs()`
  - If invalid: log warning, keep running with old config
- Thread-safe

### Step 5.8 — Wire watcher into `kronos start`

- In `cmd/start.go`:
  - After starting scheduler, start watcher
  - On shutdown, stop watcher before scheduler

### Step 5.9 — Tests

- `watcher_test.go`: write YAML, verify reload callback fires
- Platform tests: verify plist/service file content (string matching, no actual install)

### Verification

```bash
go build ./...
go test ./...
# Manual:
./kronos daemon install   # creates OS service
./kronos daemon uninstall
# Edit kronos.yaml while running → verify hot reload
```

---

## Phase 6: Release

**Goal:** CI/CD, self-update, and release automation.

### Step 6.1 — GoReleaser (`.goreleaser.yaml`)

```yaml
version: 2
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/zhenchaochen/kronos/cmd.version={{.Version}}
      - -X github.com/zhenchaochen/kronos/cmd.commit={{.ShortCommit}}
      - -X github.com/zhenchaochen/kronos/cmd.date={{.Date}}
archives:
  - format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: checksums.txt
release:
  github:
    owner: zhenchaochen
    name: kronos
```

### Step 6.2 — GitHub Actions CI (`.github/workflows/ci.yaml`)

```yaml
name: CI
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - uses: golangci/golangci-lint-action@v6
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: go test -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v4
  build:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: go build ./...
```

### Step 6.3 — Release workflow (`.github/workflows/release.yaml`)

```yaml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Step 6.4 — Self-update (`internal/updater/updater.go`)

```go
func Update(currentVersion string) error
```

- Use GitHub API: `GET /repos/zhenchaochen/kronos/releases/latest`
- Compare versions (semver)
- If newer: download appropriate binary for `runtime.GOOS`/`runtime.GOARCH`
- Replace current binary (rename + write + chmod)
- Print old → new version

### Step 6.5 — `kronos update` (`cmd/update.go`)

- Call `updater.Update(version)`
- Print result

### Verification

```bash
go build ./...
go test ./...
# Tag and push to verify CI + release
git tag v0.1.0 && git push --tags
```

---

## Final Verification Checklist (Phases 1–6)

1. `go build ./...` — compiles
2. `go test ./...` — all tests pass
3. `golangci-lint run` — no lint issues
4. Manual test flow:
   - `kronos init` → creates starter YAML
   - `kronos add --name test --cmd "echo hello" --schedule "@every 10s"`
   - `kronos doctor` → all checks pass
   - `kronos start --tui` → TUI shows job running every 10s
   - `kronos run test --dry-run` → prints without executing
   - `kronos run test` → executes, shows in history as manual
   - `kronos list --json` → valid JSON output
   - `kronos disable test` → job stops scheduling
   - Edit YAML while running → hot reload picks up changes
   - Ctrl+C → graceful shutdown within 30s
5. Cross-compile: `GOOS=windows GOARCH=amd64 go build ./...`

---

## Phase 7: `kronos logs <job>` — Log Viewer

**Goal**: View/tail job logs from CLI without TUI.

### Modify `internal/logger/logger.go`

Add two things after the existing `Logger` struct:

1. **`NewReadOnlyLogger(name, path string) *Logger`** — returns a `Logger` with `name` and `path` set, `writer` left `nil`. This is used by the CLI to call `Tail()` without needing a full `Manager`.

2. **`Path() string`** method on `*Logger` — returns `l.path`.

### Create `cmd/logs.go`

- `logsCmd` with `Use: "logs <job>"`, `Args: cobra.ExactArgs(1)`
- Flags: `--follow`/`-f` (bool, default false), `--lines`/`-n` (int, default 50)
- Registered via `rootCmd.AddCommand(logsCmd)` in `init()`

**RunE logic:**

1. `name := args[0]`; find job via `cfg.FindJob(name)` — error if nil
2. Resolve log path: `filepath.Join(config.LogDir(cfg.Settings), name+".log")`
3. Create `logger.NewReadOnlyLogger(name, logPath)`
4. Call `Tail(n)` and print each line to stdout
5. If `--follow`:
   - Open file, seek to end (track offset)
   - Use `fsnotify.NewWatcher()`, add the log file
   - Loop: on `fsnotify.Write` events, read new bytes from offset, print, update offset
   - Trap `SIGINT`/`SIGTERM` via `signal.Notify` on a channel, break loop on signal, clean exit

**Note:** `fsnotify` is already in `go.mod`.

### Verify

```
go build ./... && go test ./...
kronos logs <job> -n 20
kronos logs <job> -f
```

---

## Phase 8: `kronos export` — Export to Native Formats

**Goal**: Export jobs to crontab, launchd plist, or systemd timer formats.

### Create `internal/export/crontab.go`

- `func ToCrontab(jobs []config.Job) (string, error)`
- Skip disabled jobs (`!j.IsEnabled()`)
- Map descriptors to 5-field cron: `@daily` → `0 0 * * *`, `@hourly` → `0 * * * *`, `@weekly` → `0 0 * * 0`, `@monthly` → `0 0 1 * *`, `@yearly`/`@annually` → `0 0 1 1 *`
- `@every` schedules → emit as comment with warning: `# WARNING: @every not supported in cron`
- Prepend `# kronos: <name>` comment before each entry
- Build command: if `j.Env` set, prepend `KEY=VAL` pairs; if `j.Dir` set, prepend `cd <dir> &&`

### Create `internal/export/launchd.go`

- `func ToLaunchd(jobs []config.Job) (string, error)`
- Skip disabled jobs
- Use `text/template` to generate plist XML per job
- Map cron schedules to `StartCalendarInterval` dict keys (Hour, Minute, Weekday, Day, Month)
- `@every` → use `StartInterval` with seconds
- Separate each plist with `<!-- save as com.kronos.<name>.plist -->`
- Include `Label`, `ProgramArguments` (split on shell), `WorkingDirectory`, `EnvironmentVariables`

### Create `internal/export/systemd.go`

- `func ToSystemd(jobs []config.Job) (string, error)`
- Skip disabled jobs
- Generate `.timer` unit (with `OnCalendar=` for standard schedules, `OnUnitActiveSec=` for `@every`) + `.service` unit per job
- Separate with `# --- save as kronos-<name>.timer ---` / `# --- save as kronos-<name>.service ---` comments
- Service includes `ExecStart=`, `WorkingDirectory=`, `Environment=`

### Create `cmd/export.go`

- `exportCmd` with `Use: "export"`, no positional args
- Flags: `--format` (string, default `"crontab"`, valid: crontab|launchd|systemd), `--output`/`-o` (string, default "")
- Registered in `init()`

**RunE logic:**

1. Switch on `--format`, call `export.ToCrontab(cfg.Jobs)` / `ToLaunchd` / `ToSystemd`
2. If `--output` set, write to file; else print to stdout

### Verify

```
go build ./... && go test ./...
kronos export --format crontab
kronos export --format systemd -o /tmp/kronos-units.txt
```

---

## Phase 9: `kronos import` — Import from Crontab

**Goal**: Parse crontab files into Kronos jobs and merge into config.

### Create `internal/importer/crontab.go`

- `type ParsedJob struct` with `Name, Schedule, Cmd, Dir string; Env map[string]string`
- `func ParseCrontab(r io.Reader) ([]ParsedJob, []string, error)` — returns jobs + warnings
- Line-by-line parsing:
  - Skip blank lines, `#` comment lines
  - Detect `KEY=VAL` env lines (no spaces in key, `=` present, value before any command chars) → accumulate in env map for subsequent entries
  - Skip `@reboot` with warning
  - Handle 5-field (`min hour dom mon dow cmd`) and detect 6-field (user column) — if 6th token looks like a username (no `/`, no `.`), treat as user-field format, skip that column
  - Extract schedule (first 5 fields or descriptor like `@daily`) + remainder as command
- Name generation: take basename of first word of command, sanitize (lowercase, alphanumeric + hyphens, max 30 chars), deduplicate with `-2`, `-3` suffixes

### Create `internal/importer/merge.go`

- `type MergeResult struct { Added, Skipped []string }`
- `func Merge(cfg *config.Config, parsed []ParsedJob) MergeResult`
- For each parsed job: if `cfg.FindJob(name) != nil`, add to `Skipped`; else build `config.Job`, append to `cfg.Jobs`, add to `Added`

### Create `cmd/importjobs.go`

- `importJobsCmd` with `Use: "import"` (variable named `importJobsCmd` to avoid Go keyword)
- Flags: `--from` (string, default `"crontab"`), `--file` (string, default `""` meaning stdin)
- Registered in `init()`

**RunE logic:**

1. Open `--file` or `os.Stdin`
2. Call `importer.ParseCrontab(reader)`, print any warnings
3. Call `importer.Merge(cfg, parsed)`
4. Validate via `config.Validate(cfg)`
5. Save via `config.Save(resolveConfigPath(), cfg, nil)` (nil node since we're appending, comment preservation not critical)
6. Print summary: `"Imported X job(s), skipped Y duplicate(s)"`

### Verify

```
go build ./... && go test ./...
crontab -l | kronos import --from crontab
kronos import --from crontab --file /tmp/test-crontab
```

---

## Phase 10: `kronos prune` — History Cleanup

**Goal**: Delete old run records by age or count.

### Modify `internal/store/history.go`

Add these methods to `*Store`:

1. **`PruneOlderThan(cutoff time.Time, jobName string) (int, error)`** — iterate runs bucket, delete records with `StartTime` before cutoff. If `jobName != ""`, only delete matching prefix. Return count deleted.

2. **`CountOlderThan(cutoff time.Time, jobName string) (int, error)`** — same iteration but count-only (for dry-run). Uses `db.View` instead of `db.Update`.

3. **`PruneKeepN(jobName string, keepN int) (int, error)`** — like existing `PruneHistory` but returns count of deleted records.

4. **`CountPruneKeepN(jobName string, keepN int) (int, error)`** — count excess records without deleting.

5. **`GetAllJobNames() ([]string, error)`** — scan all keys in runs bucket, extract unique job name prefixes (everything before first `/`).

### Create `cmd/prune.go`

- `pruneCmd` with `Use: "prune"`
- Flags: `--older-than` (string, e.g. `"30d"`), `--keep` (int, default 0), `--job` (string, default ""), `--dry-run` (bool)
- Registered in `init()`

**RunE logic:**

1. Require at least one of `--older-than` or `--keep` (error otherwise)
2. Custom duration parser: if string ends with `d`, parse the number and multiply by `24*time.Hour`; else use `time.ParseDuration`
3. Open store via `store.Open(config.DBPath())`
4. If `--older-than`:
   - `cutoff := time.Now().Add(-duration)`
   - If `--dry-run`: call `CountOlderThan(cutoff, jobName)`, print "Would prune X record(s)"
   - Else: call `PruneOlderThan(cutoff, jobName)`, print "Pruned X record(s)"
5. If `--keep`:
   - If `--job` set: prune/count for that single job
   - Else: call `GetAllJobNames()`, iterate each, sum up results
   - Print appropriate message

### Verify

```
go build ./... && go test ./...
kronos prune --older-than 30d --dry-run
kronos prune --keep 50
kronos prune --job backup-db --older-than 7d
```

---

## Phase 11: `kronos status --stats` — Job Metrics

**Goal**: Per-job success rate, avg/P95 duration, total runs, last failure.

### Modify `internal/store/history.go`

Change the loop conditions in `GetRuns` and `GetAllRuns`:

- `GetRuns` line 52: `len(records) < limit` → `limit <= 0 || len(records) < limit`
- `GetAllRuns` line 75: `len(records) < limit` → `limit <= 0 || len(records) < limit`

This makes `limit <= 0` mean "return all records".

### Create `internal/stats/stats.go`

**Types:**

```go
type JobStats struct {
    Name         string
    TotalRuns    int
    SuccessCount int
    FailCount    int
    SuccessRate  float64
    AvgDuration  time.Duration
    P95Duration  time.Duration
    LastFailure  *time.Time
}

type AggregateStats struct {
    TotalJobs   int
    TotalRuns   int
    SuccessRate float64
}

type StatsReport struct {
    Jobs      []JobStats
    Aggregate AggregateStats
}
```

**`func Compute(db *store.Store, jobNames []string) (*StatsReport, error)`:**

- For each job name, call `db.GetRuns(name, 0)` (unlimited)
- Count success/fail, compute `SuccessRate = float64(success) / float64(total) * 100`
- Collect durations (`EndTime.Sub(StartTime)`), compute average
- P95: sort durations ascending, pick index `int(float64(len) * 0.95)` (clamp to last index)
- Track last failure time (most recent record where `!Success`)
- Build `AggregateStats` by summing across all jobs

### Modify `cmd/status.go`

1. Add `--stats` flag (bool) bound to a package-level `var showStats bool`
2. In the `RunE`, after existing logic, add an `if showStats` branch:
   - Collect job names from `cfg.Jobs`
   - Call `stats.Compute(db, jobNames)`
   - If `jsonOut`: JSON-encode the `StatsReport`
   - Else: render with `tabwriter` — columns: `NAME | RUNS | SUCCESS RATE | AVG DURATION | P95 DURATION | LAST FAILURE`
   - Print aggregate summary line at the bottom: total jobs, total runs, overall success rate

### Verify

```
go build ./... && go test ./...
kronos status --stats
kronos status --stats --json
```

---

## File Summary (Phases 7–11)

| Phase | New Files | Modified Files |
|-------|-----------|----------------|
| 7 | `cmd/logs.go` | `internal/logger/logger.go` |
| 8 | `internal/export/crontab.go`, `internal/export/launchd.go`, `internal/export/systemd.go`, `cmd/export.go` | — |
| 9 | `internal/importer/crontab.go`, `internal/importer/merge.go`, `cmd/importjobs.go` | — |
| 10 | `cmd/prune.go` | `internal/store/history.go` |
| 11 | `internal/stats/stats.go` | `internal/store/history.go`, `cmd/status.go` |

## Dependency Order (Phases 7–11)

Phases are independent with one caveat: **Phase 10 and 11 both modify `internal/store/history.go`**. Implement Phase 10's store changes first (adds new methods), then Phase 11's change (modifies existing `GetRuns`/`GetAllRuns` loop conditions). The changes don't conflict — Phase 10 adds new functions, Phase 11 tweaks existing function conditionals.

## Verification per Phase

Each phase: `go build ./...` + `go test ./...` must pass before moving on.

---

## Phase 12: Documentation — Getting Started & Configuration

**Goal:** Create the foundational doc files that cover installation, first-run experience, and the full configuration reference.

### Create `docs/getting-started.md` (~80 lines)

Sections:

1. **Install** — `go install github.com/zhenchaochen/kronos@latest` + binary download from GitHub Releases
2. **Quick Start** — three-step walkthrough:
   - `kronos init` → creates `~/.config/kronos/kronos.yaml` with sample "hello" job
   - Edit the config (`kronos edit` or open directly)
   - `kronos start` → runs scheduler in foreground
3. **Interactive Mode** — `kronos start --tui` with description of the 3 tabs (Jobs, Logs, History)
4. **Validate Setup** — `kronos doctor` with example output showing OK/WARN/FAIL checks
5. **Run as Service** — `kronos daemon install` (brief, links to daemon-and-platforms.md)

### Create `docs/configuration.md` (~150 lines)

Sections:

1. **File Location** — default paths by OS:
   - macOS/Linux: `~/.config/kronos/kronos.yaml`
   - Windows: `%APPDATA%/kronos/kronos.yaml`
   - Override: `--config` / `-f` flag or `KRONOS_CONFIG` env var

2. **Complete Example** — fully annotated `kronos.yaml` showing every field:

```yaml
jobs:
  - name: backup-db                      # required, must be unique
    description: "Backs up the database" # optional
    cmd: pg_dump mydb > /backups/db.sql  # required
    schedule: "@daily"                   # required, cron or descriptor
    dir: /opt/myapp                      # optional, working directory
    shell: bash                          # optional, shell interpreter
    enabled: true                        # optional, default: true
    once: false                          # optional, run once then disable
    timeout: 30m                         # optional, Go duration
    overlap: skip                        # optional: skip|allow|queue
    on_failure: retry                    # optional: retry|skip|pause
    retry_count: 3                       # optional, used with retry
    backoff: exponential                 # optional: exponential|fixed
    backoff_interval: 5s                 # optional, base interval
    tags: [db, prod]                     # optional, for filtering
    env:                                 # optional, env vars
      PGPASSWORD: secret123

settings:
  history_limit: 100                     # runs kept in DB per job
  log_dir: ""                            # empty = OS default
  log_max_size: 10                       # MB per log file
  log_max_files: 5                       # rotated log files
  shutdown_timeout: 30s                  # graceful stop timeout
```

3. **Schedule Format** — table of formats:
   - 5-field cron: `minute hour day-of-month month day-of-week`
   - Descriptors: `@yearly`, `@monthly`, `@weekly`, `@daily`, `@hourly`
   - Interval: `@every 5m`, `@every 1h30m`

4. **Overlap Policies** — `skip` (default, skip if running), `allow` (parallel), `queue` (buffer size 1)

5. **Failure Policies** — `retry` (with backoff), `skip` (ignore), `pause` (disable job)

6. **Data Paths** — table of all file paths:
   - Config, DB (`~/.cache/kronos/kronos.db`), logs (`~/.cache/kronos/logs/`), PID lock

### Verify

- All fields match `internal/config/model.go`
- Default values match `internal/config/config.go` (Validate function defaults)
- Paths match `internal/config/paths.go`

---

## Phase 13: Documentation — Command Reference

**Goal:** Document all 19 commands with flags, examples, and notes in a single self-contained file.

### Create `docs/commands.md` (~400 lines)

Structure — start with global flags, then one section per command in logical groups:

**Global Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` / `-f` | string | `~/.config/kronos/kronos.yaml` | Config file path |
| `--no-color` | bool | `false` | Disable color output |
| `--json` | bool | `false` | Machine-readable JSON output |

**Commands (grouped):**

1. **Setup**: `init`, `doctor`, `edit`
2. **Job Management**: `add`, `remove`, `enable`, `disable`, `pause-all`, `resume-all`
3. **Execution**: `start`, `run`, `daemon` (+ `install` / `uninstall`)
4. **Monitoring**: `status`, `logs`
5. **Data**: `list`, `export`, `import`, `prune`
6. **Maintenance**: `update`, `version`

Each command section follows this template:

```markdown
### kronos <command>

<one-line description>

Usage: `kronos <command> [flags]`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| ... | ... | ... | ... |

Examples:
\`\`\`bash
kronos <command> ...
\`\`\`

Notes:
- ...
```

Source of truth: each file in `cmd/` directory. Every flag registered in `init()` or `Flags()` calls must be documented.

### Verify

- Count: exactly 19 commands documented (init, add, remove, edit, list, status, run, enable, disable, pause-all, resume-all, start, daemon, daemon install, daemon uninstall, doctor, export, import, prune, logs, update, version)
- Every flag from every `cmd/*.go` file is present
- Examples are runnable commands

---

## Phase 14: Documentation — Platform, Import/Export & Troubleshooting

**Goal:** Document OS-specific daemon behavior, data interchange formats, and common issues.

### Create `docs/daemon-and-platforms.md` (~120 lines)

Sections:

1. **Running Kronos** — foreground (`kronos start`) vs background (`kronos daemon`) vs OS service (`kronos daemon install`)
2. **macOS (launchd):**
   - Install: creates `~/Library/LaunchAgents/com.kronos.agent.plist`
   - Starts at login via `RunAtLoad`, auto-restarts via `KeepAlive`
   - Uninstall: removes plist, runs `launchctl unload`
3. **Linux (systemd):**
   - Install: creates `~/.config/systemd/user/kronos.service`
   - Enables via `systemctl --user enable kronos`
   - Restart on failure with 5s delay
   - Uninstall: stops, disables, removes unit file
4. **Windows (Task Scheduler):**
   - Install: creates `KronosScheduler` task via `schtasks`
   - Triggers at user logon, runs at limited privilege
   - Uninstall: deletes scheduled task
5. **Single Instance** — PID lock at `~/.cache/kronos/kronos.pid`, stale PID detection

Source: `internal/platform/launchd.go`, `systemd.go`, `schtasks.go`, `daemon_unix.go`, `daemon_windows.go`, `detect.go`

### Create `docs/import-export.md` (~100 lines)

Sections:

1. **Export** — `kronos export --format <fmt> [-o file]`
   - **crontab**: standard 5-field format, skips disabled jobs, descriptor mapping table (`@daily` → `0 0 * * *`, etc.), warns on `@every`
   - **launchd**: one plist per job, uses `StartCalendarInterval` or `StartInterval`
   - **systemd**: `.timer` + `.service` units per job, uses `OnCalendar` or `OnUnitActiveSec`
   - Example output snippets for each format
2. **Import** — `kronos import --from crontab [--file path]`
   - Reads from file or stdin (`crontab -l | kronos import`)
   - Auto-generates job names from command basename
   - Picks up `KEY=VALUE` env var lines
   - Skips `@reboot` with warning
   - Handles system crontab (6-field with username)
   - Deduplicates by name, skips existing jobs
3. **Limitations** — `@every` not exportable to crontab, `@reboot` not importable, only crontab import currently supported

Source: `internal/export/crontab.go`, `launchd.go`, `systemd.go`, `constants.go`; `internal/importer/crontab.go`, `merge.go`

### Create `docs/troubleshooting.md` (~80 lines)

Sections:

1. **Doctor Checks** — what each check means and how to fix failures:
   - Config file not found → run `kronos init`
   - YAML parse error → check syntax, run `kronos edit` for validation
   - Job validation failure → specific field errors
   - Command not in PATH → install the tool or use absolute path
   - Log/cache directory not writable → check permissions
   - Another instance running → stop it or check stale PID

2. **Common Issues:**
   - "Permission denied" on daemon install → explain per-platform
   - Jobs not running → check `enabled`, schedule syntax, `kronos status`
   - Logs missing → check `settings.log_dir`, default paths
   - High disk usage → `kronos prune --older-than 30d`

3. **FAQ:**
   - How do I run kronos at login? → `kronos daemon install`
   - Where are my logs? → `~/.cache/kronos/logs/` or `settings.log_dir`
   - How do I migrate from crontab? → `crontab -l | kronos import`
   - How do I back up my jobs? → `kronos export --format crontab -o backup.cron`
   - Can I run multiple instances? → No, PID lock prevents it

Source: `cmd/doctor.go`, common patterns from all commands

### Verify

- Platform details match `internal/platform/` implementations
- Export format descriptions match `internal/export/` code
- Import behavior matches `internal/importer/` code
- Doctor checks match `cmd/doctor.go`

---

## Phase 15: Documentation — LLM Index Files

**Goal:** Create the two-tier `llms.txt` entry points that tie all documentation together.

### Create `docs/llms.txt` (~50 lines)

Structure:

```markdown
# kronos

> Cross-platform cron job manager. Single binary replaces crontab, launchd, and Task Scheduler.

## Install

go install github.com/zhenchaochen/kronos@latest

## Quick Start

kronos init                    # create config
kronos edit                    # customize jobs
kronos start                   # run scheduler

## Docs

- [Getting Started](./getting-started.md)
- [Configuration Reference](./configuration.md)
- [Command Reference](./commands.md)
- [Daemon & Platforms](./daemon-and-platforms.md)
- [Import & Export](./import-export.md)
- [Troubleshooting](./troubleshooting.md)

## Optional

- [Full Documentation (single file)](./llms-full.txt)
```

### Create `docs/llms-full.txt` (~900 lines)

- Concatenation of all 6 detail `.md` files in order:
  1. getting-started.md
  2. configuration.md
  3. commands.md
  4. daemon-and-platforms.md
  5. import-export.md
  6. troubleshooting.md
- Each section separated by `---` and a heading comment
- This is the "give me everything in one shot" file for LLMs with large context windows

### Verify

- All links in `llms.txt` resolve to existing files
- `llms-full.txt` contains all content from the 6 detail files
- Total line count of `llms-full.txt` is under 1500 lines (fits in a single LLM context)

---

## File Summary (Phases 12–15)

| Phase | New Files |
|-------|-----------|
| 12 | `docs/getting-started.md`, `docs/configuration.md` |
| 13 | `docs/commands.md` |
| 14 | `docs/daemon-and-platforms.md`, `docs/import-export.md`, `docs/troubleshooting.md` |
| 15 | `docs/llms.txt`, `docs/llms-full.txt` |

## Dependency Order (Phases 12–15)

Phases 12, 13, and 14 are fully independent — they can be written in any order. **Phase 15 depends on all three** because it concatenates their output into `llms-full.txt`.

## Verification per Phase

Each phase: review generated docs against source code files listed in each section. Confirm all fields, flags, and behaviors match the implementation.
