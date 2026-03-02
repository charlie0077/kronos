package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/store"
)

// Named constants for duration parsing.
const (
	hoursPerDay = 24
	daysUnit    = "d"
	weeksUnit   = "w"
	daysPerWeek = 7
)

// pruneResult holds the summary of a prune operation for JSON output.
type pruneResult struct {
	DryRun  bool   `json:"dry_run"`
	Job     string `json:"job,omitempty"`
	Deleted int    `json:"deleted"`
	Message string `json:"message"`
}

func init() {
	pruneCmd.Flags().String("older-than", "", "delete records older than duration (e.g. 30d, 2w, 24h)")
	pruneCmd.Flags().Int("keep", 0, "keep only the last N records per job")
	pruneCmd.Flags().String("job", "", "filter by job name")
	pruneCmd.Flags().Bool("dry-run", false, "show what would be deleted without deleting")
	rootCmd.AddCommand(pruneCmd)
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Delete old history records",
	Long:  "Clean up old run records from the history database by age or count.",
	RunE: func(cmd *cobra.Command, args []string) error {
		olderThan, _ := cmd.Flags().GetString("older-than")
		keep, _ := cmd.Flags().GetInt("keep")
		jobName, _ := cmd.Flags().GetString("job")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if olderThan == "" && keep == 0 {
			return fmt.Errorf("at least one of --older-than or --keep is required")
		}

		db, err := store.Open(config.DBPath())
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer db.Close()

		totalDeleted := 0

		if olderThan != "" {
			dur, err := parseDuration(olderThan)
			if err != nil {
				return fmt.Errorf("invalid --older-than value %q: %w", olderThan, err)
			}
			cutoff := time.Now().Add(-dur)

			if dryRun {
				count, err := db.CountOlderThan(cutoff, jobName)
				if err != nil {
					return fmt.Errorf("counting records: %w", err)
				}
				totalDeleted += count
			} else {
				count, err := db.PruneOlderThan(cutoff, jobName)
				if err != nil {
					return fmt.Errorf("pruning records: %w", err)
				}
				totalDeleted += count
			}
		}

		if keep > 0 {
			count, err := pruneKeepN(db, jobName, keep, dryRun)
			if err != nil {
				return err
			}
			totalDeleted += count
		}

		bothFlags := olderThan != "" && keep > 0
		return outputPruneResult(totalDeleted, jobName, dryRun, bothFlags)
	},
}

// pruneKeepN handles the --keep flag logic. If jobName is specified, it prunes
// only that job; otherwise it discovers all job names and prunes each.
func pruneKeepN(db *store.Store, jobName string, keepN int, dryRun bool) (int, error) {
	if jobName != "" {
		return pruneKeepNForJob(db, jobName, keepN, dryRun)
	}

	names, err := db.GetAllJobNames()
	if err != nil {
		return 0, fmt.Errorf("listing job names: %w", err)
	}

	total := 0
	for _, name := range names {
		count, err := pruneKeepNForJob(db, name, keepN, dryRun)
		if err != nil {
			return total, err
		}
		total += count
	}
	return total, nil
}

// pruneKeepNForJob prunes or counts excess records for a single job.
func pruneKeepNForJob(db *store.Store, jobName string, keepN int, dryRun bool) (int, error) {
	if dryRun {
		count, err := db.CountPruneKeepN(jobName, keepN)
		if err != nil {
			return 0, fmt.Errorf("counting records for %q: %w", jobName, err)
		}
		return count, nil
	}
	count, err := db.PruneKeepN(jobName, keepN)
	if err != nil {
		return 0, fmt.Errorf("pruning records for %q: %w", jobName, err)
	}
	return count, nil
}

// outputPruneResult formats and prints the prune result. When bothFlags is true
// and dryRun is active, a note is appended explaining the count is an upper bound.
func outputPruneResult(deleted int, jobName string, dryRun bool, bothFlags bool) error {
	msg := formatPruneMessage(deleted, dryRun)
	if dryRun && bothFlags {
		msg += " (upper bound — combining --older-than and --keep may delete fewer)"
	}

	if jsonOut {
		result := pruneResult{
			DryRun:  dryRun,
			Job:     jobName,
			Deleted: deleted,
			Message: msg,
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Println(msg)
	return nil
}

// formatPruneMessage builds a human-readable summary of the prune operation.
func formatPruneMessage(count int, dryRun bool) string {
	if dryRun {
		return fmt.Sprintf("Would prune %d record(s)", count)
	}
	return fmt.Sprintf("Pruned %d record(s)", count)
}

// parseDuration extends time.ParseDuration with support for day ("d") and
// week ("w") suffixes. It rejects zero and negative values to prevent
// accidental deletion of all records.
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, daysUnit) && !strings.HasSuffix(s, "ms") {
		numStr := strings.TrimSuffix(s, daysUnit)
		days, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("parsing days: %w", err)
		}
		if days <= 0 {
			return 0, fmt.Errorf("duration must be positive, got %dd", days)
		}
		return time.Duration(days) * hoursPerDay * time.Hour, nil
	}
	if strings.HasSuffix(s, weeksUnit) {
		numStr := strings.TrimSuffix(s, weeksUnit)
		weeks, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("parsing weeks: %w", err)
		}
		if weeks <= 0 {
			return 0, fmt.Errorf("duration must be positive, got %dw", weeks)
		}
		return time.Duration(weeks) * daysPerWeek * hoursPerDay * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive, got %s", s)
	}
	return d, nil
}
