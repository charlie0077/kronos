package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
)

var (
	cfgPath string
	noColor bool
	jsonOut bool

	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "kronos",
	Short: "A cross-platform cron job manager",
	Long:  "Kronos replaces platform-specific schedulers (crontab, Task Scheduler, launchd) with a single binary.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if noColor {
			os.Setenv("NO_COLOR", "1")
		}

		// Skip config loading for commands that manage their own config.
		switch cmd.Name() {
		case "version", "init", "doctor", "edit":
			return nil
		}

		path := resolveConfigPath()

		loaded, err := config.Load(path)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		cfg = loaded
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "f", "", "config file (default: ~/.config/kronos/kronos.yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")
}

// resolveConfigPath returns the explicit config path if set, otherwise the default.
func resolveConfigPath() string {
	if cfgPath != "" {
		return cfgPath
	}
	return config.DefaultConfigPath()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
