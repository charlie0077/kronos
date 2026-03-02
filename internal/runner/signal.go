package runner

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// ForwardSignals listens for SIGINT/SIGTERM and forwards them to the child
// process. Returns a cleanup function to stop the listener.
// Must be called after cmd is created but works whether or not cmd.Process
// is set yet (signals before Start are simply dropped).
func ForwardSignals(cmd *exec.Cmd) func() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case sig := <-sigCh:
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	return func() {
		signal.Stop(sigCh)
		close(done)
	}
}
