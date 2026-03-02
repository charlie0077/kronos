package platform

import (
	"runtime"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	p := Detect()

	switch runtime.GOOS {
	case "darwin":
		if p != PlatformMacOS {
			t.Errorf("expected macos, got %s", p)
		}
	case "linux":
		if p != PlatformLinux {
			t.Errorf("expected linux, got %s", p)
		}
	case "windows":
		if p != PlatformWindows {
			t.Errorf("expected windows, got %s", p)
		}
	default:
		if p != PlatformUnknown {
			t.Errorf("expected unknown, got %s", p)
		}
	}
}

func TestServiceNameNotEmpty(t *testing.T) {
	if ServiceName == "" {
		t.Error("ServiceName should not be empty")
	}
}

func TestGenerateLaunchdPlist(t *testing.T) {
	plist, err := GenerateLaunchdPlist("/usr/local/bin/kronos", "/home/user/.config/kronos/kronos.yaml", "/home/user/.cache/kronos/daemon.log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"<string>com.kronos.agent</string>",
		"<string>/usr/local/bin/kronos</string>",
		"<string>start</string>",
		"<string>/home/user/.config/kronos/kronos.yaml</string>",
		"<key>RunAtLoad</key>",
		"<key>KeepAlive</key>",
	}

	for _, s := range expected {
		if !strings.Contains(plist, s) {
			t.Errorf("plist missing %q", s)
		}
	}
}

func TestGenerateSystemdUnit(t *testing.T) {
	unit, err := GenerateSystemdUnit("/usr/local/bin/kronos", "/home/user/.config/kronos/kronos.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"Description=Kronos Scheduler",
		"ExecStart=/usr/local/bin/kronos start -f /home/user/.config/kronos/kronos.yaml",
		"Restart=on-failure",
		"RestartSec=5",
		"WantedBy=default.target",
	}

	for _, s := range expected {
		if !strings.Contains(unit, s) {
			t.Errorf("unit file missing %q", s)
		}
	}
}

func TestLaunchdPlistPath(t *testing.T) {
	path := LaunchdPlistPath()
	if !strings.Contains(path, "LaunchAgents") {
		t.Errorf("expected LaunchAgents in path, got %s", path)
	}
	if !strings.HasSuffix(path, ".plist") {
		t.Errorf("expected .plist suffix, got %s", path)
	}
}

func TestSystemdServicePath(t *testing.T) {
	path := SystemdServicePath()
	if !strings.Contains(path, "systemd/user") {
		t.Errorf("expected systemd/user in path, got %s", path)
	}
	if !strings.HasSuffix(path, ".service") {
		t.Errorf("expected .service suffix, got %s", path)
	}
}
