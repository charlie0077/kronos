package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/logger"
	"github.com/zhenchaochen/kronos/internal/runner"
	"github.com/zhenchaochen/kronos/internal/scheduler"
	"github.com/zhenchaochen/kronos/internal/store"
)

func init() {
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

		sched.Start()
		fmt.Printf("Kronos started with %d job(s). Press Ctrl+C to stop.\n", len(cfg.Jobs))

		// Wait for signal.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nShutting down...")

		shutdownTimeout, err := time.ParseDuration(cfg.Settings.ShutdownTimeout)
		if err != nil {
			shutdownTimeout = 30 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		sched.Stop(ctx)
		fmt.Println("Stopped.")
		return nil
	},
}
