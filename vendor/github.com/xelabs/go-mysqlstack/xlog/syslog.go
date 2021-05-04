// +build linux darwin dragonfly freebsd netbsd openbsd solaris

package xlog

import "log/syslog"

// NewSysLog creates a new sys log.
func NewSysLog(opts ...Option) *Log {
	w, err := syslog.New(syslog.LOG_DEBUG, "")
	if err != nil {
		panic(err)
	}
	return NewXLog(w, opts...)
}
