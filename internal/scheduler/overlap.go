package scheduler

import "github.com/zhenchaochen/kronos/internal/config"

// wrapWithOverlapPolicy wraps a job function with the configured overlap policy.
func (s *Scheduler) wrapWithOverlapPolicy(job config.Job, fn func()) func() {
	name := job.Name
	switch job.Overlap {
	case "skip":
		return func() {
			s.mu.Lock()
			if s.running[name] {
				s.mu.Unlock()
				return
			}
			s.running[name] = true
			s.mu.Unlock()

			defer func() {
				s.mu.Lock()
				s.running[name] = false
				s.mu.Unlock()
			}()
			fn()
		}
	case "queue":
		s.mu.Lock()
		if _, ok := s.queues[name]; !ok {
			s.queues[name] = make(chan struct{}, 1)
		}
		q := s.queues[name]
		s.mu.Unlock()

		return func() {
			q <- struct{}{} // blocks if queue is full (size 1)
			defer func() { <-q }()
			fn()
		}
	default: // "allow" or empty
		return fn
	}
}
