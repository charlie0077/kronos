package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/store"
)

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate your Kronos setup",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgPath
		if path == "" {
			path = config.DefaultConfigPath()
		}

		hasError := false

		// Check config file exists.
		if _, err := os.Stat(path); err != nil {
			printCheck(false, "Config file not found: %s", path)
			return fmt.Errorf("config file missing — run 'kronos init' to create one")
		}
		printCheck(true, "Config file found: %s", path)

		// Check config parses.
		loaded, err := config.Load(path)
		if err != nil {
			printCheck(false, "Config parse error: %v", err)
			return fmt.Errorf("config is not valid")
		}
		printCheck(true, "Config is valid (%d jobs)", len(loaded.Jobs))

		// Validate all jobs.
		if errs := config.Validate(loaded); len(errs) > 0 {
			for _, e := range errs {
				printCheck(false, "%v", e)
				hasError = true
			}
		} else {
			printCheck(true, "All jobs pass validation")
		}

		// Check commands in PATH.
		for _, j := range loaded.Jobs {
			cmdName := firstWord(j.Cmd)
			if _, err := exec.LookPath(cmdName); err != nil {
				printWarn("Job %q: %s not found in PATH", j.Name, cmdName)
			} else {
				printCheck(true, "Job %q: %s found in PATH", j.Name, cmdName)
			}
		}

		// Check log directory writable.
		logDir := config.LogDir(loaded.Settings)
		if err := checkDirWritable(logDir); err != nil {
			printCheck(false, "Log directory not writable: %s (%v)", logDir, err)
			hasError = true
		} else {
			printCheck(true, "Log directory writable: %s", logDir)
		}

		// Check cache directory writable.
		cacheDir := config.CacheDir()
		if err := checkDirWritable(cacheDir); err != nil {
			printCheck(false, "Cache directory not writable: %s (%v)", cacheDir, err)
			hasError = true
		} else {
			printCheck(true, "Cache directory writable: %s", cacheDir)
		}

		// Check no other instance running.
		pidLock := store.NewPIDLock(config.PIDPath())
		locked, pid, _ := pidLock.IsLocked()
		if locked {
			printWarn("Another instance is running (PID %d)", pid)
		} else {
			printCheck(true, "No other instance running")
		}

		if hasError {
			return fmt.Errorf("some checks failed")
		}
		return nil
	},
}

func printCheck(ok bool, format string, a ...any) {
	prefix := "[OK]  "
	if !ok {
		prefix = "[FAIL]"
	}
	fmt.Printf(" %s %s\n", prefix, fmt.Sprintf(format, a...))
}

func printWarn(format string, a ...any) {
	fmt.Printf(" [WARN] %s\n", fmt.Sprintf(format, a...))
}

func firstWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return s
	}
	return fields[0]
}

func checkDirWritable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(dir, ".kronos-doctor-check")
	if err := os.WriteFile(tmp, []byte("ok"), 0o644); err != nil {
		return err
	}
	return os.Remove(tmp)
}
