package config

import (
	"fmt"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

// Load reads and parses a YAML config file, applies defaults, and returns it.
func Load(path string) (*Config, error) {
	cfg, _, err := LoadWithNode(path)
	return cfg, err
}

// LoadWithNode reads and parses a YAML config file, returning both the
// typed config and the raw yaml.Node tree for comment-preserving writes.
func LoadWithNode(path string) (*Config, *yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, nil, fmt.Errorf("parsing node tree %s: %w", path, err)
	}

	ApplyDefaults(&cfg)
	return &cfg, &node, nil
}

// ApplyDefaults fills in default values for unset settings.
func ApplyDefaults(cfg *Config) {
	s := &cfg.Settings
	if s.HistoryLimit == 0 {
		s.HistoryLimit = DefaultHistoryLimit
	}
	if s.LogMaxSize == 0 {
		s.LogMaxSize = DefaultLogMaxSize
	}
	if s.LogMaxFiles == 0 {
		s.LogMaxFiles = DefaultLogMaxFiles
	}
	if s.ShutdownTimeout == "" {
		s.ShutdownTimeout = DefaultShutdownTimeout
	}
}

// Validate checks the config for errors and returns all found.
func Validate(cfg *Config) []error {
	var errs []error

	seen := make(map[string]bool)
	for i, j := range cfg.Jobs {
		prefix := fmt.Sprintf("jobs[%d]", i)

		if j.Name == "" {
			errs = append(errs, fmt.Errorf("%s: name is required", prefix))
		} else if seen[j.Name] {
			errs = append(errs, fmt.Errorf("%s: duplicate job name %q", prefix, j.Name))
		} else {
			seen[j.Name] = true
		}

		if j.Cmd == "" {
			errs = append(errs, fmt.Errorf("%s (%s): cmd is required", prefix, j.Name))
		}

		if j.Schedule == "" {
			errs = append(errs, fmt.Errorf("%s (%s): schedule is required", prefix, j.Name))
		} else {
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			if _, err := parser.Parse(j.Schedule); err != nil {
				errs = append(errs, fmt.Errorf("%s (%s): invalid schedule %q: %w", prefix, j.Name, j.Schedule, err))
			}
		}

		if j.Overlap != "" {
			switch j.Overlap {
			case "skip", "allow", "queue":
			default:
				errs = append(errs, fmt.Errorf("%s (%s): invalid overlap %q (must be skip, allow, or queue)", prefix, j.Name, j.Overlap))
			}
		}

		if j.OnFailure != "" {
			switch j.OnFailure {
			case "retry", "skip", "pause":
			default:
				errs = append(errs, fmt.Errorf("%s (%s): invalid on_failure %q (must be retry, skip, or pause)", prefix, j.Name, j.OnFailure))
			}
		}

		if j.Backoff != "" {
			switch j.Backoff {
			case "exponential", "fixed":
			default:
				errs = append(errs, fmt.Errorf("%s (%s): invalid backoff %q (must be exponential or fixed)", prefix, j.Name, j.Backoff))
			}
		}

		if j.Timeout != "" {
			if _, err := time.ParseDuration(j.Timeout); err != nil {
				errs = append(errs, fmt.Errorf("%s (%s): invalid timeout %q: %w", prefix, j.Name, j.Timeout, err))
			}
		}

		if j.BackoffInterval != "" {
			if _, err := time.ParseDuration(j.BackoffInterval); err != nil {
				errs = append(errs, fmt.Errorf("%s (%s): invalid backoff_interval %q: %w", prefix, j.Name, j.BackoffInterval, err))
			}
		}

		if j.OnFailure == "retry" && j.RetryCount < 0 {
			errs = append(errs, fmt.Errorf("%s (%s): retry_count must be >= 0 when on_failure is retry", prefix, j.Name))
		}
	}

	if cfg.Settings.ShutdownTimeout != "" {
		if _, err := time.ParseDuration(cfg.Settings.ShutdownTimeout); err != nil {
			errs = append(errs, fmt.Errorf("settings: invalid shutdown_timeout %q: %w", cfg.Settings.ShutdownTimeout, err))
		}
	}

	return errs
}
