package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/planetscale/cli/internal/connections/history"
)

type CaptureOpener func() (*history.CaptureWriter, string, error)

type CaptureControl struct {
	Open   CaptureOpener
	Writer *history.CaptureWriter
	Path   string
}

func (c *CaptureControl) Close() error {
	if c == nil || c.Writer == nil {
		return nil
	}
	err := c.Writer.Close()
	c.Writer = nil
	c.Path = ""
	return err
}

func (m Model) toggleCapture() (Model, tea.Cmd) {
	if m.capture == nil {
		m.lastError = "capture is not available in this mode"
		return m, nil
	}
	if m.capture.Writer != nil {
		closed := m.capture.Path
		if err := m.capture.Close(); err != nil {
			m.lastError = fmt.Sprintf("stop capture: %v", err)
			m = m.clearNotice()
			return m, nil
		}
		m.lastError = ""
		if closed != "" {
			return m.setNotice("stopped capture: " + closed)
		}
		return m.setNotice("stopped capture")
	}
	if m.capture.Open == nil {
		m.lastError = "capture is not available in this mode"
		return m, nil
	}
	writer, path, err := m.capture.Open()
	if err != nil {
		m.lastError = fmt.Sprintf("start capture: %v", err)
		m = m.clearNotice()
		return m, nil
	}
	m.capture.Writer = writer
	m.capture.Path = path
	if err := m.writeCaptureBackfill(); err != nil {
		m.captureStopped = "capture stopped: " + err.Error()
		_ = m.capture.Close()
		m = m.clearNotice()
		return m, nil
	}
	m.captureStopped = ""
	m.lastError = ""
	return m.setNotice("capturing to " + path)
}

func (m Model) writeCaptureBackfill() error {
	for _, list := range m.samples.All() {
		if err := m.capture.Writer.Write(history.NewCapture(list)); err != nil {
			return err
		}
	}
	return nil
}

func (m Model) captureStatusText() string {
	if m.capture == nil {
		return ""
	}
	if m.capture.Writer == nil {
		return "rec off"
	}
	if m.capture.Path != "" {
		return "rec " + m.capture.Path
	}
	return "rec on"
}
