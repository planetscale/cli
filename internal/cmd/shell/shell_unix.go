//go:build !windows

package shell

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

// warnSign shows the warning signal for prod branches.
const warnSign = "⚠"

// sysProcAttr returns the attributes for starting the process
// that are platform specific. We set the Foreground flag for unix
// like platforms, which means the new process gets its own process
// group and runs on the foreground. This ensures proper signal handling.
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Foreground: true,
	}
}

// setupSignals does not need to do any work since the foreground
// logic works well for these cases.
func setupSignals(_ context.Context, _ *exec.Cmd, _ chan os.Signal, _ []os.Signal) func() {
	return nil
}
