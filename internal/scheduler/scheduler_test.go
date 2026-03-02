package scheduler

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/logger"
	"github.com/zhenchaochen/kronos/internal/runner"
	"github.com/zhenchaochen/kronos/internal/store"
)

func setupScheduler(t *testing.T) *Scheduler {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	logMgr := logger.NewManager(filepath.Join(dir, "logs"), 10, 5)
	r := &runner.Runner{}
	return New(r, s, logMgr)
}

func TestLoadJobs(t *testing.T) {
	sched := setupScheduler(t)
	jobs := []config.Job{
		{Name: "a", Cmd: "echo a", Schedule: "@every 1h"},
		{Name: "b", Cmd: "echo b", Schedule: "@every 2h"},
	}
	if err := sched.LoadJobs(jobs); err != nil {
		t.Fatalf("LoadJobs() error: %v", err)
	}

	entries := sched.GetEntries()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestLoadDisabledJob(t *testing.T) {
	sched := setupScheduler(t)
	disabled := false
	jobs := []config.Job{
		{Name: "off", Cmd: "echo off", Schedule: "@every 1h", Enabled: &disabled},
	}
	if err := sched.LoadJobs(jobs); err != nil {
		t.Fatal(err)
	}

	entries := sched.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Enabled {
		t.Error("expected disabled entry")
	}
}

func TestStartStop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	sched := setupScheduler(t)
	jobs := []config.Job{
		{Name: "tick", Cmd: "echo tick", Schedule: "@every 1s"},
	}
	if err := sched.LoadJobs(jobs); err != nil {
		t.Fatal(err)
	}

	sched.Start()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sched.Stop(ctx)
}

func TestOverlapSkip(t *testing.T) {
	sched := setupScheduler(t)
	job := config.Job{Name: "slow", Cmd: "sleep 1", Schedule: "@every 1s", Overlap: "skip"}

	callCount := 0
	fn := func() { callCount++ }
	wrapped := sched.wrapWithOverlapPolicy(job, fn)

	// Simulate running state.
	sched.mu.Lock()
	sched.running["slow"] = true
	sched.mu.Unlock()

	wrapped()
	if callCount != 0 {
		t.Error("expected skip when already running")
	}

	// Clear running state.
	sched.mu.Lock()
	sched.running["slow"] = false
	sched.mu.Unlock()

	wrapped()
	if callCount != 1 {
		t.Error("expected execution when not running")
	}
}

func TestPauseResumeAll(t *testing.T) {
	sched := setupScheduler(t)

	sched.PauseAll()
	sched.mu.Lock()
	paused := sched.paused
	sched.mu.Unlock()
	if !paused {
		t.Error("expected paused after PauseAll")
	}

	sched.ResumeAll()
	sched.mu.Lock()
	paused = sched.paused
	sched.mu.Unlock()
	if paused {
		t.Error("expected not paused after ResumeAll")
	}
}

func TestRunJobNotFound(t *testing.T) {
	sched := setupScheduler(t)
	err := sched.RunJob("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent job")
	}
	if _, ok := err.(*JobNotFoundError); !ok {
		t.Errorf("expected JobNotFoundError, got %T", err)
	}
}
