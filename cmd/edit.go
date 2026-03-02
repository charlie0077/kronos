package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the config file in your editor",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgPath
		if path == "" {
			path = config.DefaultConfigPath()
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else {
				editor = "vi"
			}
		}

		for {
			c := exec.Command(editor, path)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return fmt.Errorf("editor exited with error: %w", err)
			}

			loaded, err := config.Load(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\nRe-edit? [Y/n] ", err)
				var answer string
				fmt.Scanln(&answer)
				if answer == "n" || answer == "N" {
					return fmt.Errorf("config has errors: %w", err)
				}
				continue
			}

			if errs := config.Validate(loaded); len(errs) > 0 {
				fmt.Fprintf(os.Stderr, "Validation errors:\n")
				for _, e := range errs {
					fmt.Fprintf(os.Stderr, "  - %v\n", e)
				}
				fmt.Fprintf(os.Stderr, "Re-edit? [Y/n] ")
				var answer string
				fmt.Scanln(&answer)
				if answer == "n" || answer == "N" {
					return fmt.Errorf("config has validation errors")
				}
				continue
			}

			fmt.Printf("Config saved: %s (%d jobs)\n", path, len(loaded.Jobs))
			return nil
		}
	},
}
