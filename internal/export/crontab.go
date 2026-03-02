package export

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zhenchaochen/kronos/internal/config"
)

// cronDescriptors maps cron descriptor strings to their 5-field equivalents.
var cronDescriptors = map[string]string{
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
	"@weekly":   "0 0 * * 0",
	"@monthly":  "0 0 1 * *",
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
}

// ToCrontab converts jobs to crontab format.
// Disabled jobs are skipped. @every schedules are emitted as comments with a warning.
func ToCrontab(jobs []config.Job) (string, error) {
	var b strings.Builder

	for _, j := range jobs {
		if !j.IsEnabled() {
			continue
		}

		schedule := strings.TrimSpace(j.Schedule)
		cmd := buildCrontabCommand(j)

		b.WriteString(fmt.Sprintf("# kronos: %s\n", j.Name))

		if strings.HasPrefix(schedule, "@every ") {
			b.WriteString("# WARNING: @every not supported in cron\n")
			b.WriteString(fmt.Sprintf("# %s %s\n", schedule, cmd))
		} else if mapped, ok := cronDescriptors[schedule]; ok {
			b.WriteString(fmt.Sprintf("%s %s\n", mapped, cmd))
		} else {
			// Assume it's already a valid 5-field cron expression.
			b.WriteString(fmt.Sprintf("%s %s\n", schedule, cmd))
		}

		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

// buildCrontabCommand constructs the full command string for a crontab entry,
// prepending environment variables and a cd command when configured.
func buildCrontabCommand(j config.Job) string {
	var parts []string

	if len(j.Env) > 0 {
		parts = append(parts, formatEnvPairs(j.Env))
	}

	if j.Dir != "" {
		parts = append(parts, fmt.Sprintf("cd %s &&", j.Dir))
	}

	parts = append(parts, j.Cmd)
	return strings.Join(parts, " ")
}

// formatEnvPairs returns sorted KEY=VAL pairs joined by spaces.
func formatEnvPairs(env map[string]string) string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, env[k]))
	}
	return strings.Join(pairs, " ")
}
