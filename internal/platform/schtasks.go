package platform

import (
	"fmt"
	"os/exec"
)

const schtasksTaskName = "KronosScheduler"

// InstallSchtasks creates a Windows Task Scheduler entry that runs at logon.
func InstallSchtasks(exePath, configPath string) error {
	// Create a task that runs at user logon with restart on failure.
	cmd := exec.Command("schtasks", "/create",
		"/tn", schtasksTaskName,
		"/tr", fmt.Sprintf(`"%s" start -f "%s"`, exePath, configPath),
		"/sc", "onlogon",
		"/rl", "limited",
		"/f", // force overwrite if exists
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks create: %s: %w", string(out), err)
	}

	return nil
}

// UninstallSchtasks removes the Kronos task from Windows Task Scheduler.
func UninstallSchtasks() error {
	cmd := exec.Command("schtasks", "/delete",
		"/tn", schtasksTaskName,
		"/f", // force without confirmation
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks delete: %s: %w", string(out), err)
	}

	return nil
}
