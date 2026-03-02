package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/logger"
	"github.com/zhenchaochen/kronos/internal/runner"
	"github.com/zhenchaochen/kronos/internal/scheduler"
	"github.com/zhenchaochen/kronos/internal/store"
)

func TestWatcherReloadsOnChange(t *testing.T) {
	// Create temp dir with a config file.
	dir := t.TempDir()
	configPath := filepath.Join(dir, "kronos.yaml")

	initialYAML := `jobs:
  - name: test-job
    cmd: echo hello
    schedule: "@every 1m"
settings:
  history_limit: 100
  log_max_size: 10
  log_max_files: 5
  shutdown_timeout: 30s
`
	if err := os.WriteFile(configPath, []byte(initialYAML), 0o644); err != nil {
		t.Fatalf("writing initial config: %v", err)
	}

	// Set up scheduler dependencies.
	dbPath := filepath.Join(dir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	defer db.Close()

	logDir := filepath.Join(dir, "logs")
	logMgr := logger.NewManager(logDir, 10, 5)
	defer logMgr.Close()

	r := &runner.Runner{}
	sched := scheduler.New(r, db, logMgr)

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if err := sched.LoadJobs(cfg.Jobs); err != nil {
		t.Fatalf("loading jobs: %v", err)
	}
	sched.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		sched.Stop(ctx)
	}()

	// Start watcher.
	var reloadCount atomic.Int32
	w := New(configPath, sched)
	w.SetOnChange(func(c *config.Config) {
		reloadCount.Add(1)
	})

	if err := w.Start(); err != nil {
		t.Fatalf("starting watcher: %v", err)
	}
	defer w.Stop()

	// Give watcher time to register.
	time.Sleep(200 * time.Millisecond)

	// Modify the config file.
	updatedYAML := `jobs:
  - name: test-job
    cmd: echo updated
    schedule: "@every 2m"
  - name: new-job
    cmd: echo new
    schedule: "@every 5m"
settings:
  history_limit: 100
  log_max_size: 10
  log_max_files: 5
  shutdown_timeout: 30s
`
	if err := os.WriteFile(configPath, []byte(updatedYAML), 0o644); err != nil {
		t.Fatalf("writing updated config: %v", err)
	}

	// Wait for debounce + reload.
	time.Sleep(500 * time.Millisecond)

	if count := reloadCount.Load(); count < 1 {
		t.Errorf("expected at least 1 reload, got %d", count)
	}

	// Verify scheduler has the new job.
	entries := sched.GetEntries()
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}
	if !names["new-job"] {
		t.Error("expected new-job to be added after reload")
	}
}

func TestWatcherIgnoresInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "kronos.yaml")

	validYAML := `jobs:
  - name: test-job
    cmd: echo hello
    schedule: "@every 1m"
settings:
  history_limit: 100
  log_max_size: 10
  log_max_files: 5
  shutdown_timeout: 30s
`
	if err := os.WriteFile(configPath, []byte(validYAML), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	defer db.Close()

	logDir := filepath.Join(dir, "logs")
	logMgr := logger.NewManager(logDir, 10, 5)
	defer logMgr.Close()

	r := &runner.Runner{}
	sched := scheduler.New(r, db, logMgr)

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if err := sched.LoadJobs(cfg.Jobs); err != nil {
		t.Fatalf("loading jobs: %v", err)
	}
	sched.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		sched.Stop(ctx)
	}()

	var reloadCount atomic.Int32
	w := New(configPath, sched)
	w.SetOnChange(func(c *config.Config) {
		reloadCount.Add(1)
	})

	if err := w.Start(); err != nil {
		t.Fatalf("starting watcher: %v", err)
	}
	defer w.Stop()

	time.Sleep(200 * time.Millisecond)

	// Write invalid config (missing required cmd field).
	invalidYAML := `jobs:
  - name: bad-job
    schedule: "@every 1m"
settings:
  history_limit: 100
  shutdown_timeout: 30s
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0o644); err != nil {
		t.Fatalf("writing invalid config: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// onChange should NOT have been called (validation failed).
	if count := reloadCount.Load(); count != 0 {
		t.Errorf("expected 0 reloads for invalid config, got %d", count)
	}

	// Original job should still be in the scheduler.
	entries := sched.GetEntries()
	if len(entries) == 0 {
		t.Error("expected original jobs to be preserved after invalid config")
	}
}
