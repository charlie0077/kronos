#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
PROJECT_DIR="$(pwd)"
PLAN_FILE="$PROJECT_DIR/PLAN.md"

# Permission mode:
#   "safe" = --allowedTools whitelist (default, uses existing claude CLI login)
#   "skip" = --dangerously-skip-permissions (no safety net, bare metal)
PERMISSION_MODE="${PERMISSION_MODE:-safe}"

ALLOWED_TOOLS="Read,Write,Edit,Glob,Grep,Bash(go *),Bash(mkdir *),Bash(ls *)"

if [[ "$PERMISSION_MODE" == "skip" ]]; then
  echo -e "WARNING: Running with --dangerously-skip-permissions (full machine access)"
  CLAUDE_FLAGS=(--dangerously-skip-permissions)
else
  echo -e "Running in safe mode (allowedTools whitelist)"
  CLAUDE_FLAGS=(--allowedTools "$ALLOWED_TOOLS")
fi

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

PHASES=(
  "Phase 1: Foundation"
  "Phase 2: Core Engine"
  "Phase 3: CLI Commands"
  "Phase 4: TUI"
  "Phase 5: Platform & Daemon"
  "Phase 6: Release"
)

PROMPTS=(
  # Phase 1
  "Implement Phase 1 (Foundation) of the Kronos project. Read PLAN.md for the full spec. Steps:
1. Run 'go mod init github.com/zhenchaochen/kronos'
2. Create cmd/root.go with cobra root command and global flags (--config, --no-color, --json)
3. Create cmd/version.go with version/commit/date via ldflags
4. Create main.go that calls cmd.Execute()
5. Create internal/config/model.go with Job, Settings, Config structs and helpers (IsEnabled, TimeoutDuration)
6. Create internal/config/paths.go with ConfigDir, CacheDir, DefaultConfigPath, LogDir, DBPath, PIDPath
7. Create internal/config/config.go with Load, LoadWithNode, ApplyDefaults, Validate functions
8. Create internal/config/writer.go with comment-preserving Save using yaml.Node
9. Create internal/config/config_test.go with tests for loading, validation, defaults
10. Create testdata/kronos.yaml fixture
Install all needed deps. Verify: go build ./... && go test ./..."

  # Phase 2
  "Implement Phase 2 (Core Engine) of the Kronos project. Read PLAN.md for the full spec. Steps:
1. Create internal/runner/shell.go with DetectShell and ShellCommand (cross-platform)
2. Create internal/runner/runner.go with Runner struct and Run method (shell exec, env merge, cwd, timeout context, combined output capture)
3. Create internal/runner/signal.go with ForwardSignals (SIGTERM/SIGINT to child process)
4. Create internal/runner/failure.go with FailureHandler (retry with exponential/fixed backoff, skip, pause policies)
5. Create internal/scheduler/scheduler.go with Scheduler struct wrapping robfig/cron (LoadJobs, Start, Stop, RunJob, GetEntries, UpdateJobs, PauseAll, ResumeAll)
6. Create internal/scheduler/overlap.go with skip/allow/queue overlap policies
7. Create internal/scheduler/once.go with once-job detection and auto-remove
8. Create internal/store/store.go with bbolt wrapper (Open, Close)
9. Create internal/store/history.go with RunRecord, SaveRun, GetRuns, GetAllRuns, GetLastRun, PruneHistory
10. Create internal/store/lock.go with PIDLock (Acquire, Release, IsLocked with stale detection)
11. Create internal/logger/logger.go with Manager and Logger (timestamped writing, Tail for TUI)
12. Create internal/logger/rotate.go using lumberjack for log rotation
13. Create tests: runner_test.go, scheduler_test.go, store_test.go, logger_test.go
Install all needed deps. Verify: go build ./... && go test ./..."

  # Phase 3
  "Implement Phase 3 (CLI Commands) of the Kronos project. Read PLAN.md for the full spec. Steps:
1. Create cmd/init.go — generate starter kronos.yaml with sample job and settings
2. Create cmd/add.go — add job via flags (--name, --cmd, --schedule, --description, --dir, --shell, --once, --timeout, --overlap, --on-failure, --retry-count, --tag)
3. Create cmd/remove.go — remove job by name with confirmation prompt (--yes to skip)
4. Create cmd/edit.go — open YAML in \$EDITOR, validate on save, re-edit on failure
5. Create cmd/list.go — table output with --json and --tag filter
6. Create cmd/status.go — table with last/next run, last result, --json support
7. Create cmd/run.go — manual trigger with --dry-run support, store result as manual
8. Create cmd/enable.go — enable/disable commands that update YAML
9. Create cmd/pause.go — pause-all/resume-all that toggle all jobs
10. Create cmd/start.go — foreground scheduler with PID lock, signal handling, graceful shutdown (--tui placeholder for Phase 4)
11. Create cmd/doctor.go — validate config, check commands in PATH, check directories writable, check PID lock
Register all commands in root.go. Verify: go build ./... && go test ./..."

  # Phase 4
  "Implement Phase 4 (TUI) of the Kronos project. Read PLAN.md for the full spec. Steps:
1. Create internal/ui/styles.go with lipgloss styles (active/inactive tab, header, selected row, status bar, success/error/warning/muted) and InitStyles(noColor) respecting NO_COLOR
2. Create internal/ui/tabs.go with RenderTabBar function
3. Create internal/ui/jobs.go with JobsModel — table of jobs with name/schedule/status/last/next, cursor navigation, r/e/d/p key bindings
4. Create internal/ui/logs.go with LogsModel — live tail of selected job log, scrollable with j/k and arrows
5. Create internal/ui/history.go with HistoryModel — past runs table with job/time/duration/result/trigger, / to filter
6. Create internal/ui/statusbar.go with context-sensitive key hints per tab
7. Create internal/ui/app.go with root Model (tab switching via tab/shift+tab, 1s auto-refresh tick, delegates to active tab, q/ctrl+c quit)
8. Wire TUI into cmd/start.go --tui flag: create dependencies, run tea.NewProgram with AltScreen, graceful shutdown on exit
Install bubbletea, lipgloss, bubbles deps. Verify: go build ./..."

  # Phase 5
  "Implement Phase 5 (Platform & Daemon) of the Kronos project. Read PLAN.md for the full spec. Steps:
1. Create internal/platform/detect.go with Platform type and Detect() using runtime.GOOS
2. Create internal/platform/daemon.go with Daemonize() — fork process with detached stdin/stdout/stderr, write PID file, parent exits
3. Create internal/platform/launchd.go with InstallLaunchd/UninstallLaunchd — generate plist at ~/Library/LaunchAgents, launchctl load/unload
4. Create internal/platform/systemd.go with InstallSystemd/UninstallSystemd — generate unit at ~/.config/systemd/user/, systemctl --user enable/start
5. Create internal/platform/schtasks.go with InstallSchtasks/UninstallSchtasks — use schtasks /create and /delete
6. Create cmd/daemon.go with 'daemon' (self-daemonize), 'daemon install' (OS-native service), 'daemon uninstall' subcommands
7. Create internal/watcher/watcher.go with fsnotify-based YAML hot reload (debounce 100ms, validate before applying, diff jobs, call scheduler.UpdateJobs)
8. Create internal/watcher/watcher_test.go
9. Wire watcher into cmd/start.go — start after scheduler, stop before scheduler on shutdown
Verify: go build ./... && go test ./..."

  # Phase 6
  "Implement Phase 6 (Release) of the Kronos project. Read PLAN.md for the full spec. Steps:
1. Create .goreleaser.yaml — CGO_ENABLED=0, linux/darwin/windows x amd64/arm64, ldflags for version/commit/date, zip for windows
2. Create .github/workflows/ci.yaml — lint (golangci-lint), test (race + coverage), build matrix (ubuntu/macos/windows)
3. Create .github/workflows/release.yaml — goreleaser on tag push with GITHUB_TOKEN
4. Create internal/updater/updater.go — check GitHub releases API for latest, compare semver, download + replace binary
5. Create cmd/update.go — call updater.Update, print result
Verify: go build ./... && go test ./..."
)

commit_phase() {
  local phase_num=$1
  local phase_name=$2
  echo -e "${CYAN}--- Committing Phase $phase_num ---${NC}"

  # Init git repo if needed (Phase 1)
  if [[ ! -d "$PROJECT_DIR/.git" ]]; then
    git init
    echo -e "${GREEN}Initialized git repo${NC}"
  fi

  git add -A
  git commit -m "$(cat <<EOF
feat: implement ${phase_name}

Phase ${phase_num} of Kronos implementation.
See PLAN.md for details.
EOF
  )" || echo -e "${YELLOW}Nothing to commit${NC}"

  echo -e "${GREEN}Committed Phase $phase_num${NC}"
}

verify_phase() {
  local phase_num=$1
  echo -e "${CYAN}--- Verifying Phase $phase_num ---${NC}"

  if [[ ! -f "$PROJECT_DIR/go.mod" ]]; then
    echo -e "${YELLOW}go.mod not found yet, skipping verification${NC}"
    return 0
  fi

  echo "Running: go build ./..."
  if go build ./...; then
    echo -e "${GREEN}Build passed${NC}"
  else
    echo -e "${RED}Build failed${NC}"
    return 1
  fi

  echo "Running: go test ./..."
  if go test ./... 2>&1; then
    echo -e "${GREEN}Tests passed${NC}"
  else
    echo -e "${YELLOW}Some tests failed (may be expected if deps aren't wired yet)${NC}"
  fi
}

run_review_loop() {
  local phase_num=$1
  echo -e "${CYAN}--- Running /rl review+fix loop for Phase $phase_num ---${NC}"

  claude "${CLAUDE_FLAGS[@]}" --verbose \
    -p "/rl" \
    2>&1 | tee "$PROJECT_DIR/.phase${phase_num}.rl.log"

  echo -e "${GREEN}Review loop complete for Phase $phase_num${NC}"
}

print_banner() {
  echo ""
  echo -e "${CYAN}╔═══════════════════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║            KRONOS — Phase-by-Phase Build         ║${NC}"
  echo -e "${CYAN}╚═══════════════════════════════════════════════════╝${NC}"
  echo ""
}

# --- Main ---

print_banner

START_PHASE=${1:-1}
END_PHASE=${2:-6}

echo -e "Building phases ${GREEN}$START_PHASE${NC} through ${GREEN}$END_PHASE${NC}"
echo ""

for i in $(seq "$START_PHASE" "$END_PHASE"); do
  idx=$((i - 1))
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${GREEN}▶ ${PHASES[$idx]}${NC}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""

  # Step 1: Implement the phase
  echo -e "${CYAN}[1/5] Implementing...${NC}"
  claude "${CLAUDE_FLAGS[@]}" --verbose \
    -p "${PROMPTS[$idx]}" \
    2>&1 | tee "$PROJECT_DIR/.phase${i}.log"

  echo ""

  # Step 2: Verify build + tests
  echo -e "${CYAN}[2/5] Verifying...${NC}"
  verify_phase "$i"
  echo ""

  # Step 3: Git commit after implementation
  echo -e "${CYAN}[3/5] Committing implementation...${NC}"
  commit_phase "$i" "${PHASES[$idx]}"
  echo ""

  # Step 4: Run /rl review+fix loop
  echo -e "${CYAN}[4/5] Review + fix loop (/rl)...${NC}"
  run_review_loop "$i"
  echo ""

  # Step 5: Final verification + commit review fixes
  echo -e "${CYAN}[5/5] Final verification + commit...${NC}"
  verify_phase "$i"
  echo ""

  # Commit any fixes from the review loop
  if [[ -n "$(git status --porcelain 2>/dev/null)" ]]; then
    git add -A
    git commit -m "$(cat <<EOF
refactor: review fixes for ${PHASES[$idx]}

Automated /rl review+fix pass.
EOF
    )" || true
    echo -e "${GREEN}Committed review fixes${NC}"
  fi
  echo ""

  if [[ "$i" -lt "$END_PHASE" ]]; then
    echo -e "${GREEN}Phase $i complete (implemented + reviewed + committed).${NC}"
    read -rp "Press Enter to continue to Phase $((i + 1)), or Ctrl+C to stop... "
    echo ""
  fi
done

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}All requested phases complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Logs:"
echo "  Implementation: .phase1.log through .phase${END_PHASE}.log"
echo "  Review loops:   .phase1.rl.log through .phase${END_PHASE}.rl.log"
