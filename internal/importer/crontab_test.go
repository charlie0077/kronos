package importer

import (
	"strings"
	"testing"
)

func TestParseCrontab(t *testing.T) {
	input := `# This is a comment
SHELL=/bin/bash
PATH=/usr/local/bin:/usr/bin:/bin

*/5 * * * * /usr/bin/backup --full
0 2 * * * /home/user/scripts/cleanup.sh
@daily /usr/local/bin/report-gen
@reboot /usr/bin/startup-task
`

	jobs, warnings, err := ParseCrontab(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}

	// Check first job.
	if jobs[0].Schedule != "*/5 * * * *" {
		t.Errorf("job[0] schedule = %q, want %q", jobs[0].Schedule, "*/5 * * * *")
	}
	if jobs[0].Cmd != "/usr/bin/backup --full" {
		t.Errorf("job[0] cmd = %q, want %q", jobs[0].Cmd, "/usr/bin/backup --full")
	}
	if jobs[0].Name != "backup" {
		t.Errorf("job[0] name = %q, want %q", jobs[0].Name, "backup")
	}

	// Check env was captured.
	if jobs[0].Env["SHELL"] != "/bin/bash" {
		t.Errorf("job[0] SHELL env = %q, want %q", jobs[0].Env["SHELL"], "/bin/bash")
	}

	// Check @daily descriptor.
	if jobs[2].Schedule != "@daily" {
		t.Errorf("job[2] schedule = %q, want %q", jobs[2].Schedule, "@daily")
	}

	// Check @reboot warning.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "@reboot") {
		t.Errorf("warning should mention @reboot: %s", warnings[0])
	}
}

func TestParseCrontabUserField(t *testing.T) {
	input := `0 3 * * * root /usr/sbin/logrotate /etc/logrotate.conf`

	jobs, _, err := ParseCrontab(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Cmd != "/usr/sbin/logrotate /etc/logrotate.conf" {
		t.Errorf("cmd = %q, want %q", jobs[0].Cmd, "/usr/sbin/logrotate /etc/logrotate.conf")
	}
}

func TestGenerateJobNameDedup(t *testing.T) {
	seen := make(map[string]int)

	n1 := generateJobName("/usr/bin/backup", seen)
	n2 := generateJobName("/opt/bin/backup", seen)
	n3 := generateJobName("/usr/local/backup", seen)

	if n1 != "backup" {
		t.Errorf("first name = %q, want %q", n1, "backup")
	}
	if n2 != "backup-2" {
		t.Errorf("second name = %q, want %q", n2, "backup-2")
	}
	if n3 != "backup-3" {
		t.Errorf("third name = %q, want %q", n3, "backup-3")
	}
}
