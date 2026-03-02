package export

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/zhenchaochen/kronos/internal/config"
)

// plistTemplate is the launchd plist XML template for a single job.
var plistTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.kronos.{{ .Name }}</string>
{{- if .ProgramArguments }}
    <key>ProgramArguments</key>
    <array>
{{- range .ProgramArguments }}
        <string>{{ . }}</string>
{{- end }}
    </array>
{{- end }}
{{- if .WorkingDirectory }}
    <key>WorkingDirectory</key>
    <string>{{ .WorkingDirectory }}</string>
{{- end }}
{{- if .EnvironmentVariables }}
    <key>EnvironmentVariables</key>
    <dict>
{{- range $k, $v := .EnvironmentVariables }}
        <key>{{ $k }}</key>
        <string>{{ $v }}</string>
{{- end }}
    </dict>
{{- end }}
{{- if .UseStartInterval }}
    <key>StartInterval</key>
    <integer>{{ .StartInterval }}</integer>
{{- else }}
    <key>StartCalendarInterval</key>
    <dict>
{{- range $k, $v := .CalendarInterval }}
        <key>{{ $k }}</key>
        <integer>{{ $v }}</integer>
{{- end }}
    </dict>
{{- end }}
</dict>
</plist>`))

// launchdJob holds the data for the plist template.
type launchdJob struct {
	Name                 string
	ProgramArguments     []string
	WorkingDirectory     string
	EnvironmentVariables map[string]string
	UseStartInterval     bool
	StartInterval        int
	CalendarInterval     map[string]int
}

// calendarIntervals maps cron descriptors to StartCalendarInterval keys.
var calendarIntervals = map[string]map[string]int{
	"@daily":    {"Hour": 0, "Minute": 0},
	"@midnight": {"Hour": 0, "Minute": 0},
	"@hourly":   {"Minute": 0},
	"@weekly":   {"Weekday": 0, "Hour": 0, "Minute": 0},
	"@monthly":  {"Day": 1, "Hour": 0, "Minute": 0},
	"@yearly":   {"Month": 1, "Day": 1, "Hour": 0, "Minute": 0},
	"@annually": {"Month": 1, "Day": 1, "Hour": 0, "Minute": 0},
}

// ToLaunchd converts jobs to launchd plist XML format.
// Disabled jobs are skipped. Each plist is separated by a save-as comment.
func ToLaunchd(jobs []config.Job) (string, error) {
	var b strings.Builder
	first := true

	for _, j := range jobs {
		if !j.IsEnabled() {
			continue
		}

		if !first {
			b.WriteString("\n")
		}
		first = false

		b.WriteString(fmt.Sprintf("<!-- save as com.kronos.%s.plist -->\n", j.Name))

		data := launchdJob{
			Name:             j.Name,
			ProgramArguments: splitShellCommand(j.Cmd),
			WorkingDirectory: j.Dir,
		}

		if len(j.Env) > 0 {
			data.EnvironmentVariables = j.Env
		}

		schedule := strings.TrimSpace(j.Schedule)
		if strings.HasPrefix(schedule, "@every ") {
			dur, err := time.ParseDuration(strings.TrimPrefix(schedule, "@every "))
			if err != nil {
				return "", fmt.Errorf("job %q: invalid @every duration %q: %w", j.Name, schedule, err)
			}
			data.UseStartInterval = true
			data.StartInterval = int(dur.Seconds())
		} else if ci, ok := calendarIntervals[schedule]; ok {
			data.CalendarInterval = ci
		} else {
			ci, err := parseCronToCalendarInterval(schedule)
			if err != nil {
				return "", fmt.Errorf("job %q: %w", j.Name, err)
			}
			data.CalendarInterval = ci
		}

		if err := plistTemplate.Execute(&b, data); err != nil {
			return "", fmt.Errorf("job %q: template error: %w", j.Name, err)
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

// splitShellCommand splits a command string into shell arguments for ProgramArguments.
// It wraps the command with /bin/sh -c for proper shell interpretation.
func splitShellCommand(cmd string) []string {
	return []string{"/bin/sh", "-c", cmd}
}

// parseCronToCalendarInterval converts a 5-field cron expression to launchd
// StartCalendarInterval dict keys. Only non-wildcard fields are included.
func parseCronToCalendarInterval(expr string) (map[string]int, error) {
	fields := strings.Fields(expr)
	if len(fields) != cronFieldCount {
		return nil, fmt.Errorf("expected %d cron fields, got %d", cronFieldCount, len(fields))
	}

	result := make(map[string]int)

	type fieldMapping struct {
		index int
		key   string
	}
	mappings := []fieldMapping{
		{cronFieldMinute, "Minute"},
		{cronFieldHour, "Hour"},
		{cronFieldDOM, "Day"},
		{cronFieldMonth, "Month"},
		{cronFieldDOW, "Weekday"},
	}

	for _, m := range mappings {
		if fields[m.index] != "*" {
			val := 0
			if _, err := fmt.Sscanf(fields[m.index], "%d", &val); err != nil {
				return nil, fmt.Errorf("cannot convert cron field %q to integer", fields[m.index])
			}
			result[m.key] = val
		}
	}

	return result, nil
}
