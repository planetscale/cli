package importcmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
	"github.com/planetscale/cli/internal/printer"
)

const (
	progressPhaseImport = "import"
	progressPhaseVerify = "verify"
)

type progressReporter struct {
	printer  *printer.Printer
	handle   *printer.ProgressHandle
	jsonMode bool
	phase    string
}

func newImportProgressReporter(ch *cmdutil.Helper, tableCount int, sizeBytes int64) *progressReporter {
	r := &progressReporter{
		printer:  ch.Printer,
		jsonMode: ch.Printer.Format() == printer.JSON,
		phase:    progressPhaseImport,
	}
	if r.jsonMode {
		return r
	}
	msg := "Importing D1 export"
	if tableCount > 0 {
		msg = fmt.Sprintf("Importing D1 export (%d tables", tableCount)
		if sizeBytes > 0 {
			msg += fmt.Sprintf(", %.1f GB", float64(sizeBytes)/(1024*1024*1024))
		}
		msg += ")..."
	} else {
		msg += "..."
	}
	r.handle = ch.Printer.StartProgress(msg)
	return r
}

func newVerifyProgressReporter(ch *cmdutil.Helper, tableCount int) *progressReporter {
	r := &progressReporter{
		printer:  ch.Printer,
		jsonMode: ch.Printer.Format() == printer.JSON,
		phase:    progressPhaseVerify,
	}
	if r.jsonMode {
		return r
	}
	msg := "Verifying D1 import..."
	if tableCount > 0 {
		msg = fmt.Sprintf("Verifying D1 import (%d tables)...", tableCount)
	}
	r.handle = ch.Printer.StartProgress(msg)
	return r
}

func (r *progressReporter) Close() {
	if r.handle != nil {
		r.handle.Stop()
	}
}

func (r *progressReporter) Report(p d1.ImportProgress) {
	msg := formatProgressMessage(p)
	if r.jsonMode {
		r.writeJSON(p, msg)
		return
	}
	if r.handle != nil {
		r.handle.Update(msg)
		return
	}
	fmt.Fprintln(os.Stderr, msg)
}

func (r *progressReporter) writeJSON(p d1.ImportProgress, message string) {
	payload := map[string]any{
		"type":    "progress",
		"phase":   r.phase,
		"stage":   p.Stage,
		"message": message,
	}
	if p.Current > 0 {
		payload["current"] = p.Current
	}
	if p.Total > 0 {
		payload["total"] = p.Total
	}
	if p.Detail != "" {
		payload["detail"] = p.Detail
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintln(os.Stderr, string(raw))
}

func formatProgressMessage(p d1.ImportProgress) string {
	return d1.FormatProgressMessage(p)
}
