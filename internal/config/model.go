package config

import "time"

// Config is the top-level Kronos configuration.
type Config struct {
	Jobs     []Job    `yaml:"jobs"`
	Settings Settings `yaml:"settings"`
}

// Job defines a single scheduled job.
type Job struct {
	Name            string            `yaml:"name"`
	Description     string            `yaml:"description,omitempty"`
	Cmd             string            `yaml:"cmd"`
	Schedule        string            `yaml:"schedule"`
	Dir             string            `yaml:"dir,omitempty"`
	Shell           string            `yaml:"shell,omitempty"`
	Enabled         *bool             `yaml:"enabled,omitempty"`
	Once            bool              `yaml:"once,omitempty"`
	Timeout         string            `yaml:"timeout,omitempty"`
	Overlap         string            `yaml:"overlap,omitempty"`
	OnFailure       string            `yaml:"on_failure,omitempty"`
	RetryCount      int               `yaml:"retry_count,omitempty"`
	Backoff         string            `yaml:"backoff,omitempty"`
	BackoffInterval string            `yaml:"backoff_interval,omitempty"`
	Tags            []string          `yaml:"tags,omitempty,flow"`
	Env             map[string]string `yaml:"env,omitempty"`
}

// IsEnabled returns true if the job is enabled (default: true).
func (j Job) IsEnabled() bool {
	return j.Enabled == nil || *j.Enabled
}

// FindJob returns a pointer to the job with the given name, or nil if not found.
func (c *Config) FindJob(name string) *Job {
	for i := range c.Jobs {
		if c.Jobs[i].Name == name {
			return &c.Jobs[i]
		}
	}
	return nil
}

// TimeoutDuration parses the timeout string into a time.Duration.
// Returns 0 if unset or unparsable.
func (j Job) TimeoutDuration() time.Duration {
	if j.Timeout == "" {
		return 0
	}
	d, err := time.ParseDuration(j.Timeout)
	if err != nil {
		return 0
	}
	return d
}

// Settings holds global Kronos settings.
type Settings struct {
	HistoryLimit    int    `yaml:"history_limit,omitempty"`
	LogDir          string `yaml:"log_dir,omitempty"`
	LogMaxSize      int    `yaml:"log_max_size,omitempty"`
	LogMaxFiles     int    `yaml:"log_max_files,omitempty"`
	ShutdownTimeout string `yaml:"shutdown_timeout,omitempty"`
}

// Default values for settings.
const (
	DefaultHistoryLimit    = 100
	DefaultLogMaxSize      = 10 // MB
	DefaultLogMaxFiles     = 5
	DefaultShutdownTimeout = "30s"
)

// Overlap policy constants.
const (
	OverlapSkip  = "skip"
	OverlapAllow = "allow"
	OverlapQueue = "queue"
)

// Failure policy constants.
const (
	OnFailureRetry = "retry"
	OnFailureSkip  = "skip"
	OnFailurePause = "pause"
)

// Backoff strategy constants.
const (
	BackoffExponential = "exponential"
	BackoffFixed       = "fixed"
)

// Trigger source constants.
const (
	TriggerScheduled = "scheduled"
	TriggerManual    = "manual"
)
