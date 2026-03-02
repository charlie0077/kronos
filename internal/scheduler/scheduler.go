package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/logger"
	"github.com/zhenchaochen/kronos/internal/runner"
	"github.com/zhenchaochen/kronos/internal/store"
)

// ScheduledJob holds display information about a scheduled job.
type ScheduledJob struct {
	Name     string
	Schedule string
	Enabled  bool
	NextRun  time.Time
	PrevRun  time.Time
}

// Scheduler manages the cron scheduler and job lifecycle.
type Scheduler struct {
	cron     *cron.Cron
	runner   *runner.Runner
	failure  *runner.FailureHandler
	store    *store.Store
	logMgr   *logger.Manager
	mu       sync.Mutex
	jobs     map[string]cron.EntryID
	configs  map[string]config.Job
	running  map[string]bool
	queues   map[string]chan struct{}
	paused   bool
	onUpdate func(jobName string)
}

// New creates a new Scheduler.
func New(r *runner.Runner, s *store.Store, l *logger.Manager) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithParser(config.CronParser)),
		runner:  r,
		failure: &runner.FailureHandler{},
		store:   s,
		logMgr:  l,
		jobs:    make(map[string]cron.EntryID),
		configs: make(map[string]config.Job),
		running: make(map[string]bool),
		queues:  make(map[string]chan struct{}),
	}
}

// SetOnUpdate sets a callback invoked after a job finishes (for TUI refresh).
func (s *Scheduler) SetOnUpdate(fn func(string)) {
	s.onUpdate = fn
}

// LoadJobs adds all enabled jobs to the cron scheduler.
func (s *Scheduler) LoadJobs(jobs []config.Job) error {
	for _, job := range jobs {
		if err := s.addJob(job); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) addJob(job config.Job) error {
	if !job.IsEnabled() {
		s.mu.Lock()
		s.configs[job.Name] = job
		s.mu.Unlock()
		return nil
	}

	fn := s.makeJobFunc(job)
	wrapped := s.wrapWithOverlapPolicy(job, fn)

	id, err := s.cron.AddFunc(job.Schedule, wrapped)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.jobs[job.Name] = id
	s.configs[job.Name] = job
	s.mu.Unlock()
	return nil
}

func (s *Scheduler) makeJobFunc(job config.Job) func() {
	return func() {
		s.mu.Lock()
		if s.paused {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()

		jobLog := s.logMgr.GetLogger(job.Name)
		r := &runner.Runner{Logger: jobLog}

		result := s.failure.Handle(context.Background(), job, func(ctx context.Context) runner.RunResult {
			return r.Run(ctx, job)
		})

		record := newRunRecord(job.Name, result.RunResult, config.TriggerScheduled)
		_ = s.store.SaveRun(record)

		if result.ShouldPause {
			s.disableJob(job.Name)
		}

		if job.Once && record.Success {
			s.handleOnceJob(job.Name)
		}

		if s.onUpdate != nil {
			s.onUpdate(job.Name)
		}
	}
}

func (s *Scheduler) disableJob(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.jobs[name]; ok {
		s.cron.Remove(id)
		delete(s.jobs, name)
	}
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop gracefully stops the scheduler with a timeout context.
func (s *Scheduler) Stop(ctx context.Context) {
	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
	case <-ctx.Done():
	}
}

// RunJob manually triggers a named job and stores the result.
func (s *Scheduler) RunJob(name string) error {
	s.mu.Lock()
	job, ok := s.configs[name]
	s.mu.Unlock()
	if !ok {
		return &JobNotFoundError{Name: name}
	}

	jobLog := s.logMgr.GetLogger(job.Name)
	r := &runner.Runner{Logger: jobLog}
	result := r.Run(context.Background(), job)

	record := newRunRecord(job.Name, result, config.TriggerManual)
	_ = s.store.SaveRun(record)

	if s.onUpdate != nil {
		s.onUpdate(name)
	}

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetEntries returns info about all known jobs for display.
func (s *Scheduler) GetEntries() []ScheduledJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []ScheduledJob
	for name, job := range s.configs {
		sj := ScheduledJob{
			Name:     name,
			Schedule: job.Schedule,
			Enabled:  job.IsEnabled() && s.jobs[name] != 0,
		}
		if id, ok := s.jobs[name]; ok {
			entry := s.cron.Entry(id)
			sj.NextRun = entry.Next
			sj.PrevRun = entry.Prev
		}
		result = append(result, sj)
	}
	return result
}

// UpdateJobs diffs the current jobs and adds/removes as needed (hot reload).
func (s *Scheduler) UpdateJobs(jobs []config.Job) error {
	newSet := make(map[string]config.Job)
	for _, j := range jobs {
		newSet[j.Name] = j
	}

	s.mu.Lock()
	// Remove jobs no longer in config.
	for name, id := range s.jobs {
		if _, exists := newSet[name]; !exists {
			s.cron.Remove(id)
			delete(s.jobs, name)
			delete(s.configs, name)
		}
	}
	s.mu.Unlock()

	// Add or update jobs.
	for _, job := range jobs {
		s.mu.Lock()
		oldJob, existed := s.configs[job.Name]
		oldID := s.jobs[job.Name]
		s.mu.Unlock()

		needsUpdate := !existed || oldJob.Schedule != job.Schedule || oldJob.IsEnabled() != job.IsEnabled()
		if needsUpdate {
			if existed {
				s.mu.Lock()
				s.cron.Remove(oldID)
				delete(s.jobs, job.Name)
				s.mu.Unlock()
			}
			if err := s.addJob(job); err != nil {
				return err
			}
		} else {
			s.mu.Lock()
			s.configs[job.Name] = job
			s.mu.Unlock()
		}
	}
	return nil
}

// PauseAll temporarily stops all job execution.
func (s *Scheduler) PauseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = true
}

// ResumeAll resumes job execution.
func (s *Scheduler) ResumeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = false
}

// newRunRecord creates a RunRecord from a RunResult with the given trigger source.
func newRunRecord(jobName string, result runner.RunResult, trigger string) store.RunRecord {
	return store.RunRecord{
		JobName:   jobName,
		StartTime: result.StartTime,
		EndTime:   result.EndTime,
		ExitCode:  result.ExitCode,
		Output:    result.Output,
		Trigger:   trigger,
		Success:   result.ExitCode == 0 && result.Error == nil,
	}
}

// JobNotFoundError is returned when a job name doesn't exist.
type JobNotFoundError struct {
	Name string
}

func (e *JobNotFoundError) Error() string {
	return "job not found: " + e.Name
}
