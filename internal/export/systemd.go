package export

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zhenchaochen/kronos/internal/config"
)

// systemdCalendars maps cron descriptors to systemd OnCalendar values.
var systemdCalendars = map[string]string{
	"@daily":    "daily",
	"@midnight": "daily",
	"@hourly":   "hourly",
	"@weekly":   "weekly",
	"@monthly":  "monthly",
	"@yearly":   "yearly",
	"@annually": "yearly",
}

// ToSystemd converts jobs to systemd timer and service unit format.
// Disabled jobs are skipped. Each unit is separated by a save-as comment.
func ToSystemd(jobs []config.Job) (string, error) {
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

		timer, err := buildTimerUnit(j)
		if err != nil {
			return "", err
		}

		service := buildServiceUnit(j)

		b.WriteString(fmt.Sprintf("# --- save as kronos-%s.timer ---\n", j.Name))
		b.WriteString(timer)
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("# --- save as kronos-%s.service ---\n", j.Name))
		b.WriteString(service)
		b.WriteString("\n")
	}

	return b.String(), nil
}

// buildTimerUnit generates the .timer unit content for a job.
func buildTimerUnit(j config.Job) (string, error) {
	var b strings.Builder

	b.WriteString("[Unit]\n")
	if j.Description != "" {
		b.WriteString(fmt.Sprintf("Description=%s\n", j.Description))
	} else {
		b.WriteString(fmt.Sprintf("Description=Kronos timer for %s\n", j.Name))
	}

	b.WriteString("\n[Timer]\n")

	schedule := strings.TrimSpace(j.Schedule)
	if strings.HasPrefix(schedule, "@every ") {
		dur, err := time.ParseDuration(strings.TrimPrefix(schedule, "@every "))
		if err != nil {
			return "", fmt.Errorf("job %q: invalid @every duration %q: %w", j.Name, schedule, err)
		}
		b.WriteString(fmt.Sprintf("OnUnitActiveSec=%s\n", formatSystemdDuration(dur)))
	} else if cal, ok := systemdCalendars[schedule]; ok {
		b.WriteString(fmt.Sprintf("OnCalendar=%s\n", cal))
	} else {
		cal, err := cronToSystemdCalendar(schedule)
		if err != nil {
			return "", fmt.Errorf("job %q: %w", j.Name, err)
		}
		b.WriteString(fmt.Sprintf("OnCalendar=%s\n", cal))
	}

	b.WriteString("Persistent=true\n")
	b.WriteString("\n[Install]\n")
	b.WriteString("WantedBy=timers.target\n")

	return b.String(), nil
}

// buildServiceUnit generates the .service unit content for a job.
func buildServiceUnit(j config.Job) string {
	var b strings.Builder

	b.WriteString("[Unit]\n")
	if j.Description != "" {
		b.WriteString(fmt.Sprintf("Description=%s\n", j.Description))
	} else {
		b.WriteString(fmt.Sprintf("Description=Kronos service for %s\n", j.Name))
	}

	b.WriteString("\n[Service]\n")
	b.WriteString("Type=oneshot\n")
	b.WriteString(fmt.Sprintf("ExecStart=/bin/sh -c '%s'\n", j.Cmd))

	if j.Dir != "" {
		b.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", j.Dir))
	}

	if len(j.Env) > 0 {
		keys := make([]string, 0, len(j.Env))
		for k := range j.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			b.WriteString(fmt.Sprintf("Environment=%s=%s\n", k, j.Env[k]))
		}
	}

	return b.String()
}

// formatSystemdDuration converts a time.Duration to a systemd duration string.
func formatSystemdDuration(d time.Duration) string {
	secs := int(d.Seconds())
	if secs <= 0 {
		return "1s"
	}

	const (
		secondsPerMinute = 60
		secondsPerHour   = 3600
	)

	if secs%secondsPerHour == 0 {
		return fmt.Sprintf("%dh", secs/secondsPerHour)
	}
	if secs%secondsPerMinute == 0 {
		return fmt.Sprintf("%dm", secs/secondsPerMinute)
	}
	return fmt.Sprintf("%ds", secs)
}

// cronToSystemdCalendar converts a 5-field cron expression to an OnCalendar value.
func cronToSystemdCalendar(expr string) (string, error) {
	fields := strings.Fields(expr)
	if len(fields) != cronFieldCount {
		return "", fmt.Errorf("expected %d cron fields, got %d", cronFieldCount, len(fields))
	}

	minute := fields[cronFieldMinute]
	hour := fields[cronFieldHour]
	dom := fields[cronFieldDOM]
	month := fields[cronFieldMonth]
	dow := fields[cronFieldDOW]

	// Build DOW prefix if specified.
	var dowPart string
	if dow != "*" {
		dowPart = dowToSystemd(dow) + " "
	}

	return fmt.Sprintf("%s*-%s-%s %s:%s:00", dowPart, month, dom, hour, minute), nil
}

// dowToSystemd converts cron day-of-week values to systemd names.
func dowToSystemd(dow string) string {
	dayNames := map[string]string{
		"0": "Sun", "1": "Mon", "2": "Tue", "3": "Wed",
		"4": "Thu", "5": "Fri", "6": "Sat", "7": "Sun",
	}
	if name, ok := dayNames[dow]; ok {
		return name
	}
	return dow
}
