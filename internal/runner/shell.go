package runner

import (
	"context"
	"os"
	"os/exec"
	"runtime"
)

// DetectShell returns the user's preferred shell, falling back to OS defaults.
func DetectShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

// ShellCommand creates an exec.Cmd that runs cmd string through the given shell.
// The context is used for timeout/cancellation support.
func ShellCommand(ctx context.Context, shell, cmdStr string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", cmdStr)
	}
	return exec.CommandContext(ctx, shell, "-c", cmdStr)
}
