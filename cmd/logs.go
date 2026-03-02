package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
	"github.com/zhenchaochen/kronos/internal/logger"
)

// Default values for the logs command flags.
const (
	defaultTailLines = 50
	readBufSize      = 4096
)

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "follow log output (like tail -f)")
	logsCmd.Flags().IntP("lines", "n", defaultTailLines, "number of lines to show")
	rootCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs <job>",
	Short: "View job log output",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		job := cfg.FindJob(name)
		if job == nil {
			return fmt.Errorf("job %q not found", name)
		}

		logPath := filepath.Join(config.LogDir(cfg.Settings), name+".log")
		lg := logger.NewReadOnlyLogger(name, logPath)

		lines, _ := cmd.Flags().GetInt("lines")
		tail, err := lg.Tail(lines)
		if err != nil {
			return fmt.Errorf("reading log: %w", err)
		}
		for _, line := range tail {
			fmt.Println(line)
		}

		follow, _ := cmd.Flags().GetBool("follow")
		if !follow {
			return nil
		}

		return followLog(lg.Path())
	},
}

// followLog opens the log file, seeks to the end, and streams new lines
// as they are written. It exits cleanly on SIGINT or SIGTERM.
func followLog(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist yet; create watcher and wait for it.
			f = nil
		} else {
			return fmt.Errorf("opening log file: %w", err)
		}
	}

	var offset int64
	if f != nil {
		offset, err = f.Seek(0, io.SeekEnd)
		if err != nil {
			f.Close()
			return fmt.Errorf("seeking log file: %w", err)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		if f != nil {
			f.Close()
		}
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the directory so we catch file creation events too.
	dir := filepath.Dir(path)
	if err := watcher.Add(dir); err != nil {
		if f != nil {
			f.Close()
		}
		return fmt.Errorf("watching directory: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	buf := make([]byte, readBufSize)

	for {
		select {
		case <-sigCh:
			if f != nil {
				f.Close()
			}
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only care about writes (or creates) to our log file.
			if event.Name != path {
				continue
			}

			if event.Has(fsnotify.Create) && f == nil {
				f, err = os.Open(path)
				if err != nil {
					continue
				}
				offset = 0
			}

			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			if f == nil {
				continue
			}

			offset, err = readNewBytes(f, offset, buf)
			if err != nil {
				return fmt.Errorf("reading new log data: %w", err)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

// readNewBytes reads any new content from the file starting at offset,
// prints complete lines to stdout, and returns the updated offset.
// The caller-provided buf is reused across calls to avoid per-event allocation.
func readNewBytes(f *os.File, offset int64, buf []byte) (int64, error) {
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset, err
	}

	for {
		n, err := f.Read(buf)
		if n > 0 {
			os.Stdout.Write(buf[:n])
			offset += int64(n)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return offset, err
		}
	}
	return offset, nil
}
