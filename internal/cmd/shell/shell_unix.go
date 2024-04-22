//go:build !windows

package shell

import "syscall"

const warnSign = "⚠"

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Foreground: true,
	}
}
