package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
)

const starterConfig = `# Kronos configuration
# Docs: https://github.com/zhenchaochen/kronos

jobs:
  - name: hello
    cmd: echo "Hello from Kronos!"
    schedule: "@every 1m"
    description: "A sample job — edit or remove me"

settings:
  history_limit: 100
  log_max_size: 10
  log_max_files: 5
  shutdown_timeout: 30s
`

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter kronos.yaml config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgPath
		if path == "" {
			path = config.DefaultConfigPath()
		}

		if _, err := os.Stat(path); err == nil {
			fmt.Printf("Config file already exists: %s\nOverwrite? [y/N] ", path)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y") {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := config.EnsureDir(config.ConfigDir()); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		if err := os.WriteFile(path, []byte(starterConfig), 0o644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}

		fmt.Printf("Created %s\n", path)
		return nil
	},
}
