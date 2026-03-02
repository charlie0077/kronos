package platform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const systemdServiceName = "kronos"

const systemdUnitTemplate = `[Unit]
Description=Kronos Scheduler
After=default.target

[Service]
ExecStart={{.ExePath}} start -f {{.ConfigPath}}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

type systemdData struct {
	ExePath    string
	ConfigPath string
}

// SystemdServicePath returns the path for the user systemd service file.
func SystemdServicePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", systemdServiceName+".service")
}

// GenerateSystemdUnit returns the systemd unit file content as a string.
func GenerateSystemdUnit(exePath, configPath string) (string, error) {
	tmpl, err := template.New("unit").Parse(systemdUnitTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing systemd template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, systemdData{
		ExePath:    exePath,
		ConfigPath: configPath,
	})
	if err != nil {
		return "", fmt.Errorf("executing systemd template: %w", err)
	}
	return buf.String(), nil
}

// InstallSystemd creates the user service file and enables it.
func InstallSystemd(exePath, configPath string) error {
	content, err := GenerateSystemdUnit(exePath, configPath)
	if err != nil {
		return err
	}

	servicePath := SystemdServicePath()
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		return fmt.Errorf("creating systemd user directory: %w", err)
	}

	if err := os.WriteFile(servicePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing service file: %w", err)
	}

	// Reload systemd, enable, and start.
	commands := [][]string{
		{"systemctl", "--user", "daemon-reload"},
		{"systemctl", "--user", "enable", systemdServiceName},
		{"systemctl", "--user", "start", systemdServiceName},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s: %w", args[0], string(out), err)
		}
	}

	return nil
}

// UninstallSystemd stops, disables, and removes the service file.
func UninstallSystemd() error {
	commands := [][]string{
		{"systemctl", "--user", "stop", systemdServiceName},
		{"systemctl", "--user", "disable", systemdServiceName},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		// Ignore errors — service may not be running or enabled.
		_ = cmd.Run()
	}

	servicePath := SystemdServicePath()
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing service file: %w", err)
	}

	// Reload after removal.
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	_ = cmd.Run()

	return nil
}
