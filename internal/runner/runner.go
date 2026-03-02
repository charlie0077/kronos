package runner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/zhenchaochen/kronos/internal/config"
)

const maxOutputSize = 64 * 1024 // 64KB cap for stored output

// RunResult holds the outcome of a job execution.
type RunResult struct {
	ExitCode  int
	Output    string
	StartTime time.Time
	EndTime   time.Time
	Error     error
}

// Runner executes jobs. If logger is non-nil, output streams to it.
type Runner struct {
	Logger io.Writer // optional, for streaming output to log file
}

// Run executes a job and returns the result.
func (r *Runner) Run(ctx context.Context, job config.Job) RunResult {
	shell := job.Shell
	if shell == "" {
		shell = DetectShell()
	}

	if timeout := job.TimeoutDuration(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := ShellCommand(ctx, shell, job.Cmd)
	cmd.Env = buildEnv(job)
	if job.Dir != "" {
		cmd.Dir = job.Dir
	}

	var buf bytes.Buffer
	var writer io.Writer = &buf
	if r.Logger != nil {
		writer = io.MultiWriter(&buf, r.Logger)
	}
	cmd.Stdout = writer
	cmd.Stderr = writer

	cleanup := ForwardSignals(cmd)
	defer cleanup()

	result := RunResult{StartTime: time.Now()}
	result.Error = cmd.Run()
	result.EndTime = time.Now()

	output := buf.Bytes()
	if len(output) > maxOutputSize {
		output = output[:maxOutputSize]
	}
	result.Output = string(output)

	if result.Error != nil {
		var exitErr *exec.ExitError
		if errors.As(result.Error, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return result
}

func buildEnv(job config.Job) []string {
	env := os.Environ()
	for k, v := range job.Env {
		env = append(env, k+"="+v)
	}
	env = append(env, "KRONOS_JOB_NAME="+job.Name)
	return env
}
