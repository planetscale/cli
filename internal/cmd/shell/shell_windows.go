//go:build windows

package shell

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// warnSign shows the warning signal for prod branches. Windows
// doesn't handle Unicode characters well here, so fall back to ASCII.
const warnSign = "!"

// sysProcAttr returns the attributes for starting the process
// that are platform specific. On Windows no additional flags are set.
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

// setupSignals handles setup for signals. On Windows we need to forward
// signals since there's otherwise no good way to foreground an independent
// MySQL shell. CREATE_NEW_CONSOLE signal wise is what we want, but we don't
// want to open up a new window but use the existing one. CREATE_NEW_PROCESS_GROUP
// doesn't handle Ctrl+c in the way that we want.
func setupSignals(ctx context.Context, c *exec.Cmd, sigc chan os.Signal, signals []os.Signal) func() {
	// Set up a new channel for signals received while MySQL is active.
	// This is registered for all signals, so we forward them all to MySQL,
	// so we behave as much as possible as a regular MySQL.
	// When we exit this function, we stop the custom signal receiver.
	msig := make(chan os.Signal, 1)
	signal.Notify(msig)

	// We stop handling signals for our default setup from the CLI. This
	// is needed, so we stop handling for example the default os.Interrupt
	// that stops the shell and we forward it to MySQL.
	// When we exit from this function, we restore the signals as they were.
	signal.Stop(sigc)

	go func() {
		for {
			select {
			case sig := <-msig:
				_ = c.Process.Signal(sig)
			case <-ctx.Done():
				return
			}
		}

	}()

	return func() {
		signal.Stop(msig)
		signal.Notify(sigc, signals...)
	}
}
