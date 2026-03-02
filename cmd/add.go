package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
)

func init() {
	addCmd.Flags().String("name", "", "job name (required)")
	addCmd.Flags().String("cmd", "", "command to run (required)")
	addCmd.Flags().String("schedule", "", "cron schedule (required)")
	addCmd.Flags().String("description", "", "job description")
	addCmd.Flags().String("dir", "", "working directory")
	addCmd.Flags().String("shell", "", "shell to use")
	addCmd.Flags().Bool("once", false, "run only once then disable")
	addCmd.Flags().String("timeout", "", "execution timeout (e.g. 30m)")
	addCmd.Flags().String("overlap", "", "overlap policy: skip|allow|queue")
	addCmd.Flags().String("on-failure", "", "failure policy: retry|skip|pause")
	addCmd.Flags().Int("retry-count", 0, "number of retries on failure")
	addCmd.Flags().StringSlice("tag", nil, "tags (repeatable)")

	_ = addCmd.MarkFlagRequired("name")
	_ = addCmd.MarkFlagRequired("cmd")
	_ = addCmd.MarkFlagRequired("schedule")

	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new job to the config",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgPath
		if path == "" {
			path = config.DefaultConfigPath()
		}

		loaded, err := config.Load(path)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		cmdStr, _ := cmd.Flags().GetString("cmd")
		schedule, _ := cmd.Flags().GetString("schedule")
		desc, _ := cmd.Flags().GetString("description")
		dir, _ := cmd.Flags().GetString("dir")
		shell, _ := cmd.Flags().GetString("shell")
		once, _ := cmd.Flags().GetBool("once")
		timeout, _ := cmd.Flags().GetString("timeout")
		overlap, _ := cmd.Flags().GetString("overlap")
		onFailure, _ := cmd.Flags().GetString("on-failure")
		retryCount, _ := cmd.Flags().GetInt("retry-count")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		if loaded.FindJob(name) != nil {
			return fmt.Errorf("job %q already exists", name)
		}

		job := config.Job{
			Name:        name,
			Description: desc,
			Cmd:         cmdStr,
			Schedule:    schedule,
			Dir:         dir,
			Shell:       shell,
			Once:        once,
			Timeout:     timeout,
			Overlap:     overlap,
			OnFailure:   onFailure,
			RetryCount:  retryCount,
			Tags:        tags,
		}

		loaded.Jobs = append(loaded.Jobs, job)

		// Validate the new config.
		if errs := config.Validate(loaded); len(errs) > 0 {
			return fmt.Errorf("validation failed: %v", errs[0])
		}

		if err := config.Save(path, loaded, nil); err != nil {
			return err
		}

		fmt.Printf("Added job %q\n", name)
		return nil
	},
}
