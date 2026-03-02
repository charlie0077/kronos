package importer

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const maxJobNameLen = 30

// ParsedJob represents a single job parsed from a crontab file.
type ParsedJob struct {
	Name     string
	Schedule string
	Cmd      string
	Dir      string
	Env      map[string]string
}

var envLineRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

// ParseCrontab reads a crontab file and returns parsed jobs and warnings.
func ParseCrontab(r io.Reader) ([]ParsedJob, []string, error) {
	var (
		jobs     []ParsedJob
		warnings []string
		env      = make(map[string]string)
		names    = make(map[string]int) // track name usage for dedup
	)

	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Detect KEY=VAL environment lines.
		if m := envLineRe.FindStringSubmatch(line); m != nil {
			env[m[1]] = m[2]
			continue
		}

		// Handle @reboot — skip with warning.
		if strings.HasPrefix(line, "@reboot") {
			warnings = append(warnings, fmt.Sprintf("line %d: @reboot is not supported, skipping", lineNum))
			continue
		}

		schedule, cmd, ok := parseCrontabLine(line)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("line %d: could not parse, skipping", lineNum))
			continue
		}

		name := generateJobName(cmd, names)

		// Snapshot current env for this job.
		var jobEnv map[string]string
		if len(env) > 0 {
			jobEnv = make(map[string]string, len(env))
			for k, v := range env {
				jobEnv[k] = v
			}
		}

		jobs = append(jobs, ParsedJob{
			Name:     name,
			Schedule: schedule,
			Cmd:      cmd,
			Env:      jobEnv,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("reading crontab: %w", err)
	}

	return jobs, warnings, nil
}

// parseCrontabLine extracts the schedule and command from a crontab line.
// It handles 5-field format and detects 6-field (user column) format.
func parseCrontabLine(line string) (schedule, cmd string, ok bool) {
	// Handle shorthand descriptors like @daily, @hourly, etc.
	if strings.HasPrefix(line, "@") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return "", "", false
		}
		return fields[0], strings.TrimSpace(line[len(fields[0]):]), true
	}

	fields := strings.Fields(line)
	const minCrontabFields = 6 // 5 schedule fields + at least 1 command word

	if len(fields) < minCrontabFields {
		return "", "", false
	}

	// Check if the 6th token looks like a username (no '/', no '.') rather than a command.
	// If so, treat as user-field format and skip that column.
	cmdStart := 5
	if len(fields) > minCrontabFields && looksLikeUsername(fields[5]) {
		cmdStart = 6
	}

	if cmdStart >= len(fields) {
		return "", "", false
	}

	schedule = strings.Join(fields[:5], " ")
	cmd = strings.Join(fields[cmdStart:], " ")
	return schedule, cmd, true
}

// looksLikeUsername returns true if the token looks like a username
// (no '/' or '.' characters).
func looksLikeUsername(s string) bool {
	return !strings.ContainsAny(s, "/.")
}

// generateJobName creates a sanitized job name from a command string,
// deduplicating with numeric suffixes.
func generateJobName(cmd string, seen map[string]int) string {
	// Take the first word and get its basename.
	firstWord := strings.Fields(cmd)[0]
	base := filepath.Base(firstWord)

	// Sanitize: lowercase, keep only alphanumeric and hyphens.
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			b.WriteRune(r)
		}
	}

	name := b.String()
	if name == "" {
		name = "job"
	}
	if len(name) > maxJobNameLen {
		name = name[:maxJobNameLen]
	}

	seen[name]++
	if seen[name] > 1 {
		name = fmt.Sprintf("%s-%d", name, seen[name])
	}

	return name
}
