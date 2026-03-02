package runner

import (
	"context"
	"math"
	"time"

	"github.com/zhenchaochen/kronos/internal/config"
)

// FailureResult wraps a RunResult with an additional flag indicating
// the job should be paused (disabled) after failure.
type FailureResult struct {
	RunResult
	ShouldPause bool
}

// FailureHandler executes a job with retry/skip/pause logic based on job config.
type FailureHandler struct{}

// Handle runs the job through the configured failure policy.
func (fh *FailureHandler) Handle(ctx context.Context, job config.Job, run func(context.Context) RunResult) FailureResult {
	switch job.OnFailure {
	case "retry":
		return fh.handleRetry(ctx, job, run)
	case "pause":
		return fh.handlePause(ctx, run)
	default: // "skip" or empty
		return FailureResult{RunResult: run(ctx)}
	}
}

func (fh *FailureHandler) handleRetry(ctx context.Context, job config.Job, run func(context.Context) RunResult) FailureResult {
	retries := job.RetryCount
	if retries <= 0 {
		retries = 1
	}

	baseInterval := parseBackoffInterval(job.BackoffInterval)
	var result RunResult

	for attempt := 0; attempt <= retries; attempt++ {
		result = run(ctx)
		if result.ExitCode == 0 && result.Error == nil {
			return FailureResult{RunResult: result}
		}

		if attempt < retries {
			wait := computeBackoff(job.Backoff, baseInterval, attempt)
			select {
			case <-ctx.Done():
				return FailureResult{RunResult: result}
			case <-time.After(wait):
			}
		}
	}

	return FailureResult{RunResult: result}
}

func (fh *FailureHandler) handlePause(ctx context.Context, run func(context.Context) RunResult) FailureResult {
	result := run(ctx)
	shouldPause := result.ExitCode != 0 || result.Error != nil
	return FailureResult{RunResult: result, ShouldPause: shouldPause}
}

const defaultBackoffInterval = 5 * time.Second

func parseBackoffInterval(s string) time.Duration {
	if s == "" {
		return defaultBackoffInterval
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultBackoffInterval
	}
	return d
}

func computeBackoff(strategy string, base time.Duration, attempt int) time.Duration {
	if strategy == "exponential" {
		return base * time.Duration(math.Pow(2, float64(attempt)))
	}
	return base // fixed or default
}
