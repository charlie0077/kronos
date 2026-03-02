package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/export"
)

var (
	exportFormat string
	exportOutput string
)

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", export.FormatCrontab,
		"export format: crontab|launchd|systemd")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "",
		"output file (default: stdout)")
	rootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export jobs to native scheduler formats",
	Long:  "Export jobs to crontab, launchd plist, or systemd timer/service formats.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			result string
			err    error
		)

		switch exportFormat {
		case export.FormatCrontab:
			result, err = export.ToCrontab(cfg.Jobs)
		case export.FormatLaunchd:
			result, err = export.ToLaunchd(cfg.Jobs)
		case export.FormatSystemd:
			result, err = export.ToSystemd(cfg.Jobs)
		default:
			return fmt.Errorf("unsupported format %q (valid: crontab, launchd, systemd)", exportFormat)
		}

		if err != nil {
			return err
		}

		if exportOutput != "" {
			const filePerms = 0o644
			if err := os.WriteFile(exportOutput, []byte(result), filePerms); err != nil {
				return fmt.Errorf("writing to %s: %w", exportOutput, err)
			}
			fmt.Fprintf(os.Stderr, "Exported to %s\n", exportOutput)
			return nil
		}

		fmt.Print(result)
		return nil
	},
}
