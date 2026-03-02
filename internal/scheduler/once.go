package scheduler

// handleOnceJob removes a once-job from the scheduler after successful execution.
func (s *Scheduler) handleOnceJob(name string) {
	s.disableJob(name)
}
