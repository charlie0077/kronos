package importer

import (
	"testing"

	"github.com/zhenchaochen/kronos/internal/config"
)

func TestMerge(t *testing.T) {
	cfg := &config.Config{
		Jobs: []config.Job{
			{Name: "existing-job", Cmd: "echo hi", Schedule: "* * * * *"},
		},
	}

	parsed := []ParsedJob{
		{Name: "existing-job", Schedule: "0 * * * *", Cmd: "echo dup"},
		{Name: "new-job", Schedule: "0 2 * * *", Cmd: "/usr/bin/new"},
	}

	result := Merge(cfg, parsed)

	if len(result.Added) != 1 || result.Added[0] != "new-job" {
		t.Errorf("Added = %v, want [new-job]", result.Added)
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != "existing-job" {
		t.Errorf("Skipped = %v, want [existing-job]", result.Skipped)
	}
	if len(cfg.Jobs) != 2 {
		t.Errorf("cfg.Jobs length = %d, want 2", len(cfg.Jobs))
	}
}
