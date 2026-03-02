package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/updater"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Self-update kronos to the latest release",
	Run: func(cmd *cobra.Command, args []string) {
		if err := updater.Update(version); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
		}
	},
}
