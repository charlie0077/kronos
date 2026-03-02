package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/runner"
	"github.com/zhenchaochen/kronos/internal/store"
)

var dryRun bool

func init() {
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would run without executing")
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Manually trigger a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		job := cfg.FindJob(name)
		if job == nil {
			return fmt.Errorf("job %q not found", name)
		}

		if dryRun {
			fmt.Printf("Would run: %s\n", job.Cmd)
			if job.Dir != "" {
				fmt.Printf("  dir: %s\n", job.Dir)
			}
			if job.Shell != "" {
				fmt.Printf("  shell: %s\n", job.Shell)
			}
			if job.Timeout != "" {
				fmt.Printf("  timeout: %s\n", job.Timeout)
			}
			return nil
		}

		r := &runner.Runner{}
		result := r.Run(context.Background(), *job)

		// Store result.
		db, err := store.Open(config.DBPath())
		if err == nil {
			_ = db.SaveRun(store.RunRecord{
				JobName:   job.Name,
				StartTime: result.StartTime,
				EndTime:   result.EndTime,
				ExitCode:  result.ExitCode,
				Output:    result.Output,
				Trigger:   config.TriggerManual,
				Success:   result.ExitCode == 0 && result.Error == nil,
			})
			db.Close()
		}

		dur := result.EndTime.Sub(result.StartTime).Truncate(time.Millisecond)
		if result.Error != nil {
			fmt.Printf("FAIL (exit %d, %s)\n", result.ExitCode, dur)
			if result.Output != "" {
				fmt.Println(result.Output)
			}
			return fmt.Errorf("job failed with exit code %d", result.ExitCode)
		}

		fmt.Printf("OK (exit %d, %s)\n", result.ExitCode, dur)
		if result.Output != "" {
			fmt.Print(result.Output)
		}
		return nil
	},
}
