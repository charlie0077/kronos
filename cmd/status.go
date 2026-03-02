package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/stats"
	"github.com/zhenchaochen/kronos/internal/store"
)

var showStats bool

func init() {
	statusCmd.Flags().BoolVar(&showStats, "stats", false, "show per-job metrics (runs, success rate, durations)")
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show job status overview",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := store.Open(config.DBPath())
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer db.Close()

		if showStats {
			return runStats(db)
		}

		return runStatus(db)
	},
}

// runStatus renders the default status table.
func runStatus(db *store.Store) error {
	type jobStatus struct {
		Name       string `json:"name"`
		Schedule   string `json:"schedule"`
		Status     string `json:"status"`
		LastRun    string `json:"last_run"`
		NextRun    string `json:"next_run"`
		LastResult string `json:"last_result"`
	}

	var rows []jobStatus
	for _, j := range cfg.Jobs {
		status := "active"
		if !j.IsEnabled() {
			status = "disabled"
		}

		lastRun := "—"
		lastResult := "—"
		if rec, err := db.GetLastRun(j.Name); err == nil && rec != nil {
			lastRun = rec.StartTime.Local().Format("2006-01-02 15:04:05")
			dur := rec.EndTime.Sub(rec.StartTime).Truncate(time.Millisecond)
			if rec.Success {
				lastResult = fmt.Sprintf("OK (%s)", dur)
			} else {
				lastResult = fmt.Sprintf("FAIL (exit %d)", rec.ExitCode)
			}
		}

		nextRun := "—"
		if j.IsEnabled() {
			if sched, err := config.CronParser.Parse(j.Schedule); err == nil {
				nextRun = sched.Next(time.Now()).Local().Format("2006-01-02 15:04:05")
			}
		}

		rows = append(rows, jobStatus{
			Name:       j.Name,
			Schedule:   j.Schedule,
			Status:     status,
			LastRun:    lastRun,
			NextRun:    nextRun,
			LastResult: lastResult,
		})
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(rows)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSCHEDULE\tSTATUS\tLAST RUN\tNEXT RUN\tLAST RESULT")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Name, r.Schedule, r.Status, r.LastRun, r.NextRun, r.LastResult)
	}
	return w.Flush()
}

// runStats computes and renders per-job metrics.
func runStats(db *store.Store) error {
	jobNames := make([]string, 0, len(cfg.Jobs))
	for _, j := range cfg.Jobs {
		jobNames = append(jobNames, j.Name)
	}

	report, err := stats.Compute(db, jobNames)
	if err != nil {
		return fmt.Errorf("computing stats: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(report)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tRUNS\tSUCCESS RATE\tAVG DURATION\tP95 DURATION\tLAST FAILURE")
	for _, js := range report.Jobs {
		lastFail := "—"
		if js.LastFailure != nil {
			lastFail = js.LastFailure.Local().Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%s\t%d\t%.1f%%\t%s\t%s\t%s\n",
			js.Name,
			js.TotalRuns,
			js.SuccessRate,
			js.AvgDuration.Truncate(time.Millisecond),
			js.P95Duration.Truncate(time.Millisecond),
			lastFail,
		)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Total: %d jobs, %d runs, %.1f%% success rate\n",
		report.Aggregate.TotalJobs,
		report.Aggregate.TotalRuns,
		report.Aggregate.SuccessRate,
	)
	return w.Flush()
}
