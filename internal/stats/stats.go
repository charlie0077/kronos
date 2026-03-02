package stats

import (
	"sort"
	"time"

	"github.com/zhenchaochen/kronos/internal/store"
)

// p95Percentile is the percentile used for duration calculation.
const p95Percentile = 0.95

// unlimitedRecords signals GetRuns to return all records.
const unlimitedRecords = 0

// JobStats holds computed metrics for a single job.
type JobStats struct {
	Name         string         `json:"name"`
	TotalRuns    int            `json:"total_runs"`
	SuccessCount int            `json:"success_count"`
	FailCount    int            `json:"fail_count"`
	SuccessRate  float64        `json:"success_rate"`
	AvgDuration  time.Duration  `json:"avg_duration"`
	P95Duration  time.Duration  `json:"p95_duration"`
	LastFailure  *time.Time     `json:"last_failure"`
}

// AggregateStats holds summary metrics across all jobs.
type AggregateStats struct {
	TotalJobs   int     `json:"total_jobs"`
	TotalRuns   int     `json:"total_runs"`
	SuccessRate float64 `json:"success_rate"`
}

// StatsReport combines per-job and aggregate metrics.
type StatsReport struct {
	Jobs      []JobStats     `json:"jobs"`
	Aggregate AggregateStats `json:"aggregate"`
}

// Compute calculates metrics for the given job names from the store.
func Compute(db *store.Store, jobNames []string) (*StatsReport, error) {
	report := &StatsReport{
		Jobs: make([]JobStats, 0, len(jobNames)),
	}

	var totalRuns, totalSuccess int

	for _, name := range jobNames {
		runs, err := db.GetRuns(name, unlimitedRecords)
		if err != nil {
			return nil, err
		}

		js := computeJobStats(name, runs)
		report.Jobs = append(report.Jobs, js)

		totalRuns += js.TotalRuns
		totalSuccess += js.SuccessCount
	}

	report.Aggregate = AggregateStats{
		TotalJobs:   len(jobNames),
		TotalRuns:   totalRuns,
		SuccessRate: successRate(totalSuccess, totalRuns),
	}

	return report, nil
}

// computeJobStats derives metrics from a slice of run records for one job.
func computeJobStats(name string, runs []store.RunRecord) JobStats {
	js := JobStats{
		Name:      name,
		TotalRuns: len(runs),
	}

	if len(runs) == 0 {
		return js
	}

	var durations []time.Duration
	var lastFail time.Time

	for _, r := range runs {
		if r.Success {
			js.SuccessCount++
		} else {
			js.FailCount++
			if r.StartTime.After(lastFail) {
				lastFail = r.StartTime
			}
		}
		durations = append(durations, r.EndTime.Sub(r.StartTime))
	}

	js.SuccessRate = successRate(js.SuccessCount, js.TotalRuns)
	js.AvgDuration = avgDuration(durations)
	js.P95Duration = p95Duration(durations)

	if !lastFail.IsZero() {
		t := lastFail
		js.LastFailure = &t
	}

	return js
}

// successRate computes the percentage of successful runs.
func successRate(success, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total) * 100
}

// avgDuration computes the mean duration.
func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

// p95Duration computes the 95th percentile duration.
func p95Duration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(float64(len(sorted)) * p95Percentile)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
