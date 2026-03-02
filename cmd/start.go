package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/logger"
	"github.com/zhenchaochen/kronos/internal/runner"
	"github.com/zhenchaochen/kronos/internal/scheduler"
	"github.com/zhenchaochen/kronos/internal/store"
	"github.com/zhenchaochen/kronos/internal/ui"
	"github.com/zhenchaochen/kronos/internal/watcher"
)

var tuiMode bool

func init() {
	startCmd.Flags().BoolVar(&tuiMode, "tui", false, "launch interactive terminal UI")
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the scheduler in the foreground",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Acquire PID lock.
		pidLock := store.NewPIDLock(config.PIDPath())
		if err := pidLock.Acquire(); err != nil {
			return err
		}
		defer pidLock.Release()

		// Open store.
		db, err := store.Open(config.DBPath())
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer db.Close()

		// Create logger manager.
		logDir := config.LogDir(cfg.Settings)
		logMgr := logger.NewManager(logDir, cfg.Settings.LogMaxSize, cfg.Settings.LogMaxFiles)
		defer logMgr.Close()

		// Create runner and scheduler.
		r := &runner.Runner{}
		sched := scheduler.New(r, db, logMgr)

		if err := sched.LoadJobs(cfg.Jobs); err != nil {
			return fmt.Errorf("loading jobs: %w", err)
		}

		configPath := resolveConfigPath()

		if tuiMode {
			return runTUI(sched, db, logMgr, cfg, configPath)
		}
		return runHeadless(sched, cfg, configPath)
	},
}

// startWatcher creates and starts a config hot-reload watcher. Returns a stop
// function that the caller should defer. If the watcher fails to start, the
// returned stop function is a no-op.
func startWatcher(configPath string, sched *scheduler.Scheduler) (stop func()) {
	w := watcher.New(configPath, sched)
	if err := w.Start(); err != nil {
		log.Printf("[watcher] failed to start: %v (hot reload disabled)", err)
		return func() {}
	}
	return w.Stop
}

func runTUI(sched *scheduler.Scheduler, db *store.Store, logMgr *logger.Manager, c *config.Config, configPath string) error {
	ui.InitStyles(noColor)
	model := ui.NewModel(sched, db, logMgr, c)

	p := tea.NewProgram(model, tea.WithAltScreen())

	// Wire scheduler updates to TUI refresh before starting to avoid race.
	sched.SetOnUpdate(func(_ string) {
		p.Send(ui.RefreshCmd())
	})
	sched.Start()
	defer startWatcher(configPath, sched)()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Graceful shutdown.
	shutdownTimeout := parseShutdownTimeout(c.Settings.ShutdownTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	sched.Stop(ctx)
	return nil
}

func runHeadless(sched *scheduler.Scheduler, c *config.Config, configPath string) error {
	sched.Start()
	defer startWatcher(configPath, sched)()

	fmt.Printf("Kronos started with %d job(s). Press Ctrl+C to stop.\n", len(c.Jobs))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")

	shutdownTimeout := parseShutdownTimeout(c.Settings.ShutdownTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	sched.Stop(ctx)
	fmt.Println("Stopped.")
	return nil
}

func parseShutdownTimeout(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		// Fallback to config default; this shouldn't happen if config validation passed.
		d, _ = time.ParseDuration(config.DefaultShutdownTimeout)
		return d
	}
	return d
}
