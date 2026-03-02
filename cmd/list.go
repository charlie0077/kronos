package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
)

var listTag string

func init() {
	listCmd.Flags().StringVar(&listTag, "tag", "", "filter by tag")
	rootCmd.AddCommand(listCmd)
}

type jobRow struct {
	Name        string   `json:"name"`
	Schedule    string   `json:"schedule"`
	Enabled     bool     `json:"enabled"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

func jobToRow(j config.Job) jobRow {
	return jobRow{j.Name, j.Schedule, j.IsEnabled(), j.Tags, j.Description}
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		var rows []jobRow
		for _, j := range cfg.Jobs {
			if listTag != "" && !hasTag(j.Tags, listTag) {
				continue
			}
			rows = append(rows, jobToRow(j))
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(rows)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSCHEDULE\tENABLED\tTAGS\tDESCRIPTION")
		for _, r := range rows {
			enabled := "yes"
			if !r.Enabled {
				enabled = "no"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				r.Name, r.Schedule, enabled, strings.Join(r.Tags, ","), r.Description)
		}
		return w.Flush()
	},
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
