package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidYAML(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "testdata", "kronos.yaml"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(cfg.Jobs))
	}

	job := cfg.Jobs[0]
	if job.Name != "backup-db" {
		t.Errorf("expected name backup-db, got %s", job.Name)
	}
	if job.Schedule != "@daily" {
		t.Errorf("expected schedule @daily, got %s", job.Schedule)
	}
	if job.Overlap != "skip" {
		t.Errorf("expected overlap skip, got %s", job.Overlap)
	}
	if job.RetryCount != 3 {
		t.Errorf("expected retry_count 3, got %d", job.RetryCount)
	}
	if !job.IsEnabled() {
		t.Error("expected job to be enabled")
	}
	if job.TimeoutDuration().Minutes() != 30 {
		t.Errorf("expected timeout 30m, got %v", job.TimeoutDuration())
	}
}

func TestLoadWithNode(t *testing.T) {
	cfg, node, err := LoadWithNode(filepath.Join("..", "..", "testdata", "kronos.yaml"))
	if err != nil {
		t.Fatalf("LoadWithNode() error: %v", err)
	}
	if cfg == nil || node == nil {
		t.Fatal("expected non-nil config and node")
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	ApplyDefaults(cfg)

	if cfg.Settings.HistoryLimit != DefaultHistoryLimit {
		t.Errorf("expected history_limit %d, got %d", DefaultHistoryLimit, cfg.Settings.HistoryLimit)
	}
	if cfg.Settings.LogMaxSize != DefaultLogMaxSize {
		t.Errorf("expected log_max_size %d, got %d", DefaultLogMaxSize, cfg.Settings.LogMaxSize)
	}
	if cfg.Settings.LogMaxFiles != DefaultLogMaxFiles {
		t.Errorf("expected log_max_files %d, got %d", DefaultLogMaxFiles, cfg.Settings.LogMaxFiles)
	}
	if cfg.Settings.ShutdownTimeout != DefaultShutdownTimeout {
		t.Errorf("expected shutdown_timeout %s, got %s", DefaultShutdownTimeout, cfg.Settings.ShutdownTimeout)
	}
}

func TestApplyDefaultsPreservesExisting(t *testing.T) {
	cfg := &Config{Settings: Settings{HistoryLimit: 50, LogMaxSize: 5, LogMaxFiles: 3, ShutdownTimeout: "15s"}}
	ApplyDefaults(cfg)

	if cfg.Settings.HistoryLimit != 50 {
		t.Errorf("expected preserved history_limit 50, got %d", cfg.Settings.HistoryLimit)
	}
}

func TestValidateValid(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "testdata", "kronos.yaml"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	errs := Validate(cfg)
	if len(errs) != 0 {
		t.Errorf("expected no validation errors, got %v", errs)
	}
}

func TestValidateMissingName(t *testing.T) {
	cfg := &Config{Jobs: []Job{{Cmd: "echo hi", Schedule: "@daily"}}}
	errs := Validate(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error for missing name")
	}
}

func TestValidateMissingCmd(t *testing.T) {
	cfg := &Config{Jobs: []Job{{Name: "test", Schedule: "@daily"}}}
	errs := Validate(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error for missing cmd")
	}
}

func TestValidateInvalidSchedule(t *testing.T) {
	cfg := &Config{Jobs: []Job{{Name: "test", Cmd: "echo hi", Schedule: "not-a-cron"}}}
	errs := Validate(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error for invalid schedule")
	}
}

func TestValidateDuplicateNames(t *testing.T) {
	cfg := &Config{Jobs: []Job{
		{Name: "dup", Cmd: "echo a", Schedule: "@daily"},
		{Name: "dup", Cmd: "echo b", Schedule: "@hourly"},
	}}
	errs := Validate(cfg)
	found := false
	for _, e := range errs {
		if e.Error() != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for duplicate names")
	}
}

func TestValidateInvalidOverlap(t *testing.T) {
	cfg := &Config{Jobs: []Job{{Name: "test", Cmd: "echo", Schedule: "@daily", Overlap: "bad"}}}
	errs := Validate(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error for invalid overlap")
	}
}

func TestValidateInvalidOnFailure(t *testing.T) {
	cfg := &Config{Jobs: []Job{{Name: "test", Cmd: "echo", Schedule: "@daily", OnFailure: "bad"}}}
	errs := Validate(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error for invalid on_failure")
	}
}

func TestValidateInvalidBackoff(t *testing.T) {
	cfg := &Config{Jobs: []Job{{Name: "test", Cmd: "echo", Schedule: "@daily", Backoff: "bad"}}}
	errs := Validate(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error for invalid backoff")
	}
}

func TestValidateInvalidTimeout(t *testing.T) {
	cfg := &Config{Jobs: []Job{{Name: "test", Cmd: "echo", Schedule: "@daily", Timeout: "notaduration"}}}
	errs := Validate(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error for invalid timeout")
	}
}

func TestValidateInvalidShutdownTimeout(t *testing.T) {
	cfg := &Config{Settings: Settings{ShutdownTimeout: "bad"}}
	errs := Validate(cfg)
	if len(errs) == 0 {
		t.Error("expected validation error for invalid shutdown_timeout")
	}
}

func TestIsEnabled(t *testing.T) {
	// nil = enabled
	j := Job{}
	if !j.IsEnabled() {
		t.Error("expected nil Enabled to mean enabled")
	}

	// explicit true
	tr := true
	j.Enabled = &tr
	if !j.IsEnabled() {
		t.Error("expected true Enabled")
	}

	// explicit false
	fa := false
	j.Enabled = &fa
	if j.IsEnabled() {
		t.Error("expected false Enabled")
	}
}

func TestTimeoutDuration(t *testing.T) {
	j := Job{Timeout: "5m"}
	if j.TimeoutDuration().String() != "5m0s" {
		t.Errorf("expected 5m0s, got %s", j.TimeoutDuration())
	}

	j2 := Job{}
	if j2.TimeoutDuration() != 0 {
		t.Error("expected 0 for empty timeout")
	}

	j3 := Job{Timeout: "invalid"}
	if j3.TimeoutDuration() != 0 {
		t.Error("expected 0 for invalid timeout")
	}
}

func TestPathFunctions(t *testing.T) {
	if ConfigDir() == "" {
		t.Error("ConfigDir() returned empty")
	}
	if CacheDir() == "" {
		t.Error("CacheDir() returned empty")
	}
	if DefaultConfigPath() == "" {
		t.Error("DefaultConfigPath() returned empty")
	}
	if DBPath() == "" {
		t.Error("DBPath() returned empty")
	}
	if PIDPath() == "" {
		t.Error("PIDPath() returned empty")
	}
	if LogDir(Settings{}) == "" {
		t.Error("LogDir() returned empty for default")
	}
	if LogDir(Settings{LogDir: "/custom"}) != "/custom" {
		t.Error("LogDir() should use custom path when set")
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kronos.yaml")

	cfg := &Config{
		Jobs: []Job{{Name: "test", Cmd: "echo hi", Schedule: "@daily"}},
	}
	ApplyDefaults(cfg)

	if err := Save(path, cfg, nil); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}
	if len(reloaded.Jobs) != 1 || reloaded.Jobs[0].Name != "test" {
		t.Error("reloaded config doesn't match saved")
	}
}
