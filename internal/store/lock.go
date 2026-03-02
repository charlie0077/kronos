package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PIDLock manages a PID file to ensure single-instance operation.
type PIDLock struct {
	path string
}

// NewPIDLock creates a new PID lock at the given path.
func NewPIDLock(path string) *PIDLock {
	return &PIDLock{path: path}
}

// Acquire writes the current PID to the lock file. It fails if another
// live process already holds the lock.
func (l *PIDLock) Acquire() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("creating pid directory: %w", err)
	}

	locked, pid, err := l.IsLocked()
	if err != nil {
		return err
	}
	if locked {
		return fmt.Errorf("another kronos instance is running (PID %d)", pid)
	}

	return os.WriteFile(l.path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// Release removes the PID lock file.
func (l *PIDLock) Release() error {
	return os.Remove(l.path)
}

// IsLocked checks if the lock is held by a running process.
// Returns (locked, pid, error).
func (l *PIDLock) IsLocked() (bool, int, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("reading pid file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Corrupt PID file — treat as stale.
		_ = os.Remove(l.path)
		return false, 0, nil
	}

	if !processAlive(pid) {
		// Stale PID file — clean up.
		_ = os.Remove(l.path)
		return false, 0, nil
	}

	return true, pid, nil
}

// processAlive checks if a process with the given PID is running.
func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if the process exists without actually sending a signal.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
