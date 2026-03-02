package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
)

func init() {
	rootCmd.AddCommand(pauseAllCmd)
	rootCmd.AddCommand(resumeAllCmd)
}

var pauseAllCmd = &cobra.Command{
	Use:   "pause-all",
	Short: "Disable all jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return setAllEnabled(false)
	},
}

var resumeAllCmd = &cobra.Command{
	Use:   "resume-all",
	Short: "Enable all jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return setAllEnabled(true)
	},
}

func setAllEnabled(enable bool) error {
	path := cfgPath
	if path == "" {
		path = config.DefaultConfigPath()
	}

	loaded, err := config.Load(path)
	if err != nil {
		return err
	}

	count := 0
	for i := range loaded.Jobs {
		if loaded.Jobs[i].IsEnabled() != enable {
			loaded.Jobs[i].Enabled = &enable
			count++
		}
	}

	if err := config.Save(path, loaded, nil); err != nil {
		return err
	}

	action := "Paused"
	if enable {
		action = "Resumed"
	}
	fmt.Printf("%s %d job(s)\n", action, count)
	return nil
}
