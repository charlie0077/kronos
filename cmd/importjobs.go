package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/importer"
)

var (
	importFrom string
	importFile string
)

func init() {
	importJobsCmd.Flags().StringVar(&importFrom, "from", "crontab", "import format (currently only crontab)")
	importJobsCmd.Flags().StringVar(&importFile, "file", "", "input file (default: stdin)")
	rootCmd.AddCommand(importJobsCmd)
}

var importJobsCmd = &cobra.Command{
	Use:   "import",
	Short: "Import jobs from crontab or other formats",
	Long:  "Parse a crontab file and merge discovered jobs into the Kronos config.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if importFrom != "crontab" {
			return fmt.Errorf("unsupported format %q (valid: crontab)", importFrom)
		}

		var reader *os.File
		if importFile != "" {
			f, err := os.Open(importFile)
			if err != nil {
				return fmt.Errorf("opening %s: %w", importFile, err)
			}
			defer f.Close()
			reader = f
		} else {
			reader = os.Stdin
		}

		parsed, warnings, err := importer.ParseCrontab(reader)
		if err != nil {
			return err
		}

		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		}

		result := importer.Merge(cfg, parsed)

		if errs := config.Validate(cfg); len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "validation error: %s\n", e)
			}
			return fmt.Errorf("imported config has %d validation error(s)", len(errs))
		}

		if err := config.Save(resolveConfigPath(), cfg, nil); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Imported %d job(s), skipped %d duplicate(s)\n", len(result.Added), len(result.Skipped))
		return nil
	},
}
