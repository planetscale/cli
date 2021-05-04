// +build windows

package xlog

import (
	"os"
)

// NewSysLog creates a new sys log. Because there is no syslog support for
// Windows, we output to os.Stdout.
func NewSysLog(opts ...Option) *Log {
	return NewXLog(os.Stdout, opts...)
}
