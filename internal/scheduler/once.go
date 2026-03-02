package scheduler

// handleOnceJob removes a once-job from the scheduler after successful execution.
func (s *Scheduler) handleOnceJob(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.jobs[name]; ok {
		s.cron.Remove(id)
		delete(s.jobs, name)
	}
}
