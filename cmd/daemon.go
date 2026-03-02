package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/platform"
)

func init() {
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	rootCmd.AddCommand(daemonCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run Kronos as a background daemon",
	Long:  "Self-daemonize Kronos. Use 'daemon install' for OS-native service integration.",
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolving executable path: %w", err)
		}

		childArgs := []string{"start", "-f", resolveConfigPath()}
		pid, err := platform.Daemonize(exe, childArgs)
		if err != nil {
			return err
		}

		fmt.Printf("Kronos daemon started (PID %d)\n", pid)
		return nil
	},
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Kronos as an OS-native service",
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolving executable path: %w", err)
		}

		configPath := resolveConfigPath()

		p := platform.Detect()
		switch p {
		case platform.PlatformMacOS:
			logPath := filepath.Join(config.CacheDir(), "daemon.log")
			if err := config.EnsureDir(filepath.Dir(logPath)); err != nil {
				return err
			}
			if err := platform.InstallLaunchd(exe, configPath, logPath); err != nil {
				return err
			}
			fmt.Printf("Installed launchd service: %s\n", platform.LaunchdPlistPath())

		case platform.PlatformLinux:
			if err := platform.InstallSystemd(exe, configPath); err != nil {
				return err
			}
			fmt.Printf("Installed systemd user service: %s\n", platform.SystemdServicePath())

		case platform.PlatformWindows:
			if err := platform.InstallSchtasks(exe, configPath); err != nil {
				return err
			}
			fmt.Println("Installed Windows Task Scheduler entry: KronosScheduler")

		default:
			return fmt.Errorf("unsupported platform: %s", p)
		}

		return nil
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Kronos OS-native service",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := platform.Detect()
		switch p {
		case platform.PlatformMacOS:
			if err := platform.UninstallLaunchd(); err != nil {
				return err
			}
			fmt.Println("Removed launchd service.")

		case platform.PlatformLinux:
			if err := platform.UninstallSystemd(); err != nil {
				return err
			}
			fmt.Println("Removed systemd user service.")

		case platform.PlatformWindows:
			if err := platform.UninstallSchtasks(); err != nil {
				return err
			}
			fmt.Println("Removed Windows Task Scheduler entry.")

		default:
			return fmt.Errorf("unsupported platform: %s", p)
		}

		return nil
	},
}
