//go:build windows

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Daemonize forks the current process as a background daemon.
// On Windows, uses CREATE_NEW_PROCESS_GROUP to detach the child.
func Daemonize(exe string, args []string) (int, error) {
	cmd := exec.Command(exe, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to daemonize: %w", err)
	}

	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}
