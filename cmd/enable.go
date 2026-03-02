package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
)

func init() {
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
}

var enableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a job",
	Args:  cobra.ExactArgs(1),
	RunE:  toggleJob(true),
}

var disableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a job",
	Args:  cobra.ExactArgs(1),
	RunE:  toggleJob(false),
}

func toggleJob(enable bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := cfgPath
		if path == "" {
			path = config.DefaultConfigPath()
		}

		loaded, err := config.Load(path)
		if err != nil {
			return err
		}

		job := loaded.FindJob(name)
		if job == nil {
			return fmt.Errorf("job %q not found", name)
		}
		job.Enabled = &enable

		if err := config.Save(path, loaded, nil); err != nil {
			return err
		}

		action := "Enabled"
		if !enable {
			action = "Disabled"
		}
		fmt.Printf("%s job %q\n", action, name)
		return nil
	}
}
