package platform

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const launchdLabel = ServiceName

const launchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.ExePath}}</string>
		<string>start</string>
		<string>-f</string>
		<string>{{.ConfigPath}}</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>{{.LogPath}}</string>
	<key>StandardErrorPath</key>
	<string>{{.LogPath}}</string>
</dict>
</plist>
`

type launchdData struct {
	Label      string
	ExePath    string
	ConfigPath string
	LogPath    string
}

// LaunchdPlistPath returns the path where the plist will be installed.
func LaunchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

// xmlEscape returns s with XML special characters escaped.
func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// GenerateLaunchdPlist returns the plist content as a string.
func GenerateLaunchdPlist(exePath, configPath, logPath string) (string, error) {
	tmpl, err := template.New("plist").Parse(launchdPlistTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing plist template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, launchdData{
		Label:      launchdLabel,
		ExePath:    xmlEscape(exePath),
		ConfigPath: xmlEscape(configPath),
		LogPath:    xmlEscape(logPath),
	})
	if err != nil {
		return "", fmt.Errorf("executing plist template: %w", err)
	}
	return buf.String(), nil
}

// InstallLaunchd creates the plist file and loads it via launchctl.
func InstallLaunchd(exePath, configPath, logPath string) error {
	content, err := GenerateLaunchdPlist(exePath, configPath, logPath)
	if err != nil {
		return err
	}

	plistPath := LaunchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("creating LaunchAgents directory: %w", err)
	}

	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing plist file: %w", err)
	}

	cmd := exec.Command("launchctl", "load", plistPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %s: %w", string(out), err)
	}

	return nil
}

// UninstallLaunchd unloads and removes the plist file.
func UninstallLaunchd() error {
	plistPath := LaunchdPlistPath()

	cmd := exec.Command("launchctl", "unload", plistPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl unload: %s: %w", string(out), err)
	}

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist file: %w", err)
	}

	return nil
}

