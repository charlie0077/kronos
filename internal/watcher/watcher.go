package watcher

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/scheduler"
)

// debounceInterval is how long to wait after a write event before reloading.
const debounceInterval = 100 * time.Millisecond

// Watcher monitors the config file for changes and hot-reloads the scheduler.
type Watcher struct {
	path      string
	scheduler *scheduler.Scheduler
	onChange  func(*config.Config)
	done      chan struct{}
	watcher   *fsnotify.Watcher
	mu        sync.Mutex
	stopOnce  sync.Once
}

// New creates a new config file watcher.
func New(path string, sched *scheduler.Scheduler) *Watcher {
	return &Watcher{
		path:      path,
		scheduler: sched,
		done:      make(chan struct{}),
	}
}

// SetOnChange sets a callback invoked after successful reload.
func (w *Watcher) SetOnChange(fn func(*config.Config)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onChange = fn
}

// Start begins watching the config file for changes.
func (w *Watcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating file watcher: %w", err)
	}

	if err := fsw.Add(w.path); err != nil {
		fsw.Close()
		return fmt.Errorf("watching config file: %w", err)
	}

	w.watcher = fsw
	go w.loop()
	return nil
}

// Stop stops watching and releases resources. Safe to call multiple times.
func (w *Watcher) Stop() {
	w.stopOnce.Do(func() {
		close(w.done)
		if w.watcher != nil {
			w.watcher.Close()
		}
	})
}

func (w *Watcher) loop() {
	var debounceTimer *time.Timer

	for {
		select {
		case <-w.done:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			// Debounce: reset the timer on each write event.
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceInterval, func() {
				w.reload()
			})

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[watcher] error: %v", err)
		}
	}
}

func (w *Watcher) reload() {
	cfg, err := config.Load(w.path)
	if err != nil {
		log.Printf("[watcher] failed to load config: %v", err)
		return
	}

	errs := config.Validate(cfg)
	if len(errs) > 0 {
		log.Printf("[watcher] config validation failed (%d errors), keeping old config", len(errs))
		for _, e := range errs {
			log.Printf("[watcher]   - %v", e)
		}
		return
	}

	if err := w.scheduler.UpdateJobs(cfg.Jobs); err != nil {
		log.Printf("[watcher] failed to update jobs: %v", err)
		return
	}

	log.Printf("[watcher] config reloaded successfully (%d jobs)", len(cfg.Jobs))

	w.mu.Lock()
	fn := w.onChange
	w.mu.Unlock()
	if fn != nil {
		fn(cfg)
	}
}
