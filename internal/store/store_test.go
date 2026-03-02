package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSaveAndGetRun(t *testing.T) {
	s := tempStore(t)

	rec := RunRecord{
		JobName:   "backup",
		StartTime: time.Now().Add(-time.Minute),
		EndTime:   time.Now(),
		ExitCode:  0,
		Output:    "done",
		Trigger:   "scheduled",
		Success:   true,
	}
	if err := s.SaveRun(rec); err != nil {
		t.Fatalf("SaveRun() error: %v", err)
	}

	runs, err := s.GetRuns("backup", 10)
	if err != nil {
		t.Fatalf("GetRuns() error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].JobName != "backup" {
		t.Errorf("expected job name backup, got %s", runs[0].JobName)
	}
	if !runs[0].Success {
		t.Error("expected success=true")
	}
}

func TestGetLastRun(t *testing.T) {
	s := tempStore(t)

	now := time.Now()
	for i := 0; i < 3; i++ {
		rec := RunRecord{
			JobName:   "test",
			StartTime: now.Add(time.Duration(i) * time.Minute),
			EndTime:   now.Add(time.Duration(i)*time.Minute + time.Second),
			ExitCode:  i,
			Trigger:   "scheduled",
			Success:   i == 0,
		}
		if err := s.SaveRun(rec); err != nil {
			t.Fatal(err)
		}
	}

	last, err := s.GetLastRun("test")
	if err != nil {
		t.Fatal(err)
	}
	if last == nil {
		t.Fatal("expected non-nil last run")
	}
	if last.ExitCode != 2 {
		t.Errorf("expected last run exit code 2, got %d", last.ExitCode)
	}
}

func TestGetLastRunNotFound(t *testing.T) {
	s := tempStore(t)
	last, err := s.GetLastRun("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if last != nil {
		t.Error("expected nil for nonexistent job")
	}
}

func TestGetAllRuns(t *testing.T) {
	s := tempStore(t)

	now := time.Now()
	for _, name := range []string{"a", "b"} {
		rec := RunRecord{
			JobName:   name,
			StartTime: now,
			EndTime:   now.Add(time.Second),
			Trigger:   "manual",
			Success:   true,
		}
		if err := s.SaveRun(rec); err != nil {
			t.Fatal(err)
		}
	}

	runs, err := s.GetAllRuns(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestPruneHistory(t *testing.T) {
	s := tempStore(t)

	now := time.Now()
	for i := 0; i < 5; i++ {
		rec := RunRecord{
			JobName:   "prune",
			StartTime: now.Add(time.Duration(i) * time.Minute),
			EndTime:   now.Add(time.Duration(i)*time.Minute + time.Second),
			Trigger:   "scheduled",
			Success:   true,
		}
		if err := s.SaveRun(rec); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.PruneHistory("prune", 2); err != nil {
		t.Fatalf("PruneHistory() error: %v", err)
	}

	runs, err := s.GetRuns("prune", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs after prune, got %d", len(runs))
	}
}

func TestPIDLockAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	lock := NewPIDLock(filepath.Join(dir, "test.pid"))

	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire() error: %v", err)
	}

	locked, pid, err := lock.IsLocked()
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Error("expected locked after acquire")
	}
	if pid != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), pid)
	}

	// Second acquire should fail.
	if err := lock.Acquire(); err == nil {
		t.Error("expected error on double acquire")
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("Release() error: %v", err)
	}

	locked, _, err = lock.IsLocked()
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("expected unlocked after release")
	}
}

func TestPIDLockStaleCleanup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stale.pid")

	// Write a PID that doesn't exist.
	os.WriteFile(path, []byte("999999"), 0o644)

	lock := NewPIDLock(path)
	locked, _, err := lock.IsLocked()
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("expected stale PID to be cleaned up")
	}
}
