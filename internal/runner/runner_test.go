package runner

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zhenchaochen/kronos/internal/config"
)

func TestRunEchoCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	r := &Runner{}
	job := config.Job{Name: "test", Cmd: "echo hello", Schedule: "@daily"}
	result := r.Run(context.Background(), job)

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected output to contain 'hello', got %q", result.Output)
	}
	if result.StartTime.IsZero() || result.EndTime.IsZero() {
		t.Error("expected non-zero start/end times")
	}
}

func TestRunTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	r := &Runner{}
	job := config.Job{Name: "sleeper", Cmd: "sleep 10", Schedule: "@daily", Timeout: "100ms"}
	result := r.Run(context.Background(), job)

	if result.Error == nil {
		t.Error("expected timeout error")
	}
	duration := result.EndTime.Sub(result.StartTime)
	if duration > 2*time.Second {
		t.Errorf("expected quick timeout, took %v", duration)
	}
}

func TestRunEnvInjection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	r := &Runner{}
	job := config.Job{
		Name:     "envtest",
		Cmd:      "echo $KRONOS_JOB_NAME $MY_VAR",
		Schedule: "@daily",
		Env:      map[string]string{"MY_VAR": "hello"},
	}
	result := r.Run(context.Background(), job)

	if !strings.Contains(result.Output, "envtest") {
		t.Errorf("expected KRONOS_JOB_NAME in output, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected MY_VAR in output, got %q", result.Output)
	}
}

func TestRunExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	r := &Runner{}
	job := config.Job{Name: "fail", Cmd: "exit 42", Schedule: "@daily"}
	result := r.Run(context.Background(), job)

	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}
}

func TestDetectShell(t *testing.T) {
	shell := DetectShell()
	if shell == "" {
		t.Error("DetectShell() returned empty")
	}
}
