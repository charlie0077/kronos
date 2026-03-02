package importer

import "github.com/zhenchaochen/kronos/internal/config"

// MergeResult tracks which jobs were added vs skipped during import.
type MergeResult struct {
	Added   []string
	Skipped []string
}

// Merge adds parsed jobs to the config, skipping duplicates by name.
func Merge(cfg *config.Config, parsed []ParsedJob) MergeResult {
	var result MergeResult

	for _, p := range parsed {
		if cfg.FindJob(p.Name) != nil {
			result.Skipped = append(result.Skipped, p.Name)
			continue
		}

		job := config.Job{
			Name:     p.Name,
			Schedule: p.Schedule,
			Cmd:      p.Cmd,
			Dir:      p.Dir,
			Env:      p.Env,
		}

		cfg.Jobs = append(cfg.Jobs, job)
		result.Added = append(result.Added, p.Name)
	}

	return result
}
