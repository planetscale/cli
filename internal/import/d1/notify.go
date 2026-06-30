package d1

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

const importNotifyTimeout = 3 * time.Second

// D1 import Slack notification event names.
const (
	NotifyEventStarting  = "starting"
	NotifyEventProgress  = "progress"
	NotifyEventImported  = "imported"
	NotifyEventVerifying = "verifying"
	NotifyEventVerified  = "verified"
	NotifyEventComplete  = "complete"
	NotifyEventFailed    = "failed"
)

// NotifyAPIConfig carries the PlanetScale API client for async D1 import notifications.
type NotifyAPIConfig struct {
	Client *ps.Client
	// Disabled skips notifications (--no-notify).
	Disabled bool
}

// NotifyImportEvent posts a D1 import lifecycle event to api-bb asynchronously.
// Progress updates use this path; lifecycle and failure events should use NotifyImportEventSync.
func NotifyImportEvent(api NotifyAPIConfig, org, database, branch, migrationID, event string, extra importNotificationPayload) {
	deliverImportNotification(api, org, database, branch, migrationID, event, extra, false)
}

// NotifyImportEventSync waits briefly for api-bb to accept the notification.
// Used for lifecycle boundaries and failures so Slack is reported before the CLI exits.
func NotifyImportEventSync(api NotifyAPIConfig, org, database, branch, migrationID, event string, extra importNotificationPayload) {
	deliverImportNotification(api, org, database, branch, migrationID, event, extra, true)
}

func deliverImportNotification(api NotifyAPIConfig, org, database, branch, migrationID, event string, extra importNotificationPayload, wait bool) {
	if api.Disabled || api.Client == nil {
		return
	}

	payload := importNotificationPayload{
		MigrationID: migrationID,
		Event:       event,
		Method:      extra.Method,
		ExportBytes: extra.ExportBytes,
		TableCount:  extra.TableCount,
		Matched:     extra.Matched,
		DurationMs:  extra.DurationMs,
		Error:       extra.Error,
		Stage:       extra.Stage,
		Message:     extra.Message,
	}
	if branch != "" {
		payload.BranchName = branch
	}

	send := func() {
		ctx, cancel := context.WithTimeout(context.Background(), importNotifyTimeout)
		defer cancel()
		_ = postImportNotification(ctx, api, org, database, payload)
	}

	if wait {
		send()
		return
	}

	go send()
}

type importNotificationPayload struct {
	BranchName  string
	MigrationID string
	Event       string
	Method      string
	ExportBytes int64
	TableCount  int
	Matched     *bool
	DurationMs  int64
	Error       string
	Stage       string
	Message     string
}

func postImportNotification(ctx context.Context, api NotifyAPIConfig, org, database string, payload importNotificationPayload) error {
	return api.Client.D1ImportNotifications.Create(ctx, &ps.CreateD1ImportNotificationRequest{
		Organization: org,
		Database:     database,
		BranchName:   payload.BranchName,
		MigrationID:  payload.MigrationID,
		Event:        payload.Event,
		Method:       payload.Method,
		ExportBytes:  payload.ExportBytes,
		TableCount:   payload.TableCount,
		Matched:      payload.Matched,
		DurationMs:   payload.DurationMs,
		Error:        payload.Error,
		Stage:        payload.Stage,
		Message:      payload.Message,
	})
}

func notifyPayloadFromImport(opts ImportOptions, result *ImportResult) importNotificationPayload {
	payload := importNotificationPayload{
		Method: opts.Method,
	}
	if result != nil {
		payload.TableCount = result.TablesLoaded
		if result.Timings != nil {
			payload.DurationMs = result.Timings.TotalMs
		}
	}
	if info, err := os.Stat(opts.InputPath); err == nil {
		payload.ExportBytes = info.Size()
	}
	return payload
}

func notifyPayloadFromState(state *MigrationState) importNotificationPayload {
	if state == nil {
		return importNotificationPayload{}
	}

	payload := importNotificationPayload{
		Method: state.Method,
	}
	if state.InputPath != "" {
		if info, err := os.Stat(state.InputPath); err == nil {
			payload.ExportBytes = info.Size()
		}
	}
	if n := len(state.LoadedTables); n > 0 {
		payload.TableCount = n
	}
	if !state.CreatedAt.IsZero() && !state.UpdatedAt.IsZero() {
		payload.DurationMs = state.UpdatedAt.Sub(state.CreatedAt).Milliseconds()
	}
	return payload
}

func notifyPayloadFromVerify(opts VerifyOptions) importNotificationPayload {
	payload := importNotificationPayload{}
	if state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID); err == nil {
		payload.Method = state.Method
		if n := len(state.LoadedTables); n > 0 {
			payload.TableCount = n
		}
	}
	if opts.InputPath != "" {
		if info, err := os.Stat(opts.InputPath); err == nil {
			payload.ExportBytes = info.Size()
		}
	}
	return payload
}

func notifyImportProgress(api NotifyAPIConfig, org, database, branch, migrationID string, base importNotificationPayload, p ImportProgress) {
	if !shouldNotifyProgress(p) {
		return
	}
	payload := base
	payload.Stage = p.Stage
	payload.Message = FormatProgressMessage(p)
	NotifyImportEvent(api, org, database, branch, migrationID, NotifyEventProgress, payload)
}

func shouldNotifyProgress(p ImportProgress) bool {
	switch p.Stage {
	case ImportStageConnecting, ImportStageSQLiteStaging, ImportStageSchema,
		ImportStagePgloader, ImportStageIndexes, ImportStageSequences,
		VerifyStageRowCounts, VerifyStageSequences, VerifyStageBoolean, VerifyStageFingerprints, VerifyStageSampleRows:
		return true
	default:
		return false
	}
}

// notifyImportFailure posts a failed event with structured MigrationError details.
func notifyImportFailure(api NotifyAPIConfig, org, database, branch, migrationID string, base importNotificationPayload, err error, verifyResult *VerifyResult) {
	if err == nil {
		return
	}
	payload := base
	payload.Error = formatNotifyError(err, verifyResult)
	if verifyResult != nil && !verifyResult.Matched {
		matched := false
		payload.Matched = &matched
	}
	NotifyImportEventSync(api, org, database, branch, migrationID, NotifyEventFailed, payload)
}

func formatNotifyError(err error, verifyResult *VerifyResult) string {
	if errors.Is(err, context.Canceled) {
		return "[IMPORT_FAILED] import cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "[IMPORT_FAILED] import timed out"
	}
	for e := err; e != nil; e = errors.Unwrap(e) {
		if me, ok := migrationErr(e); ok {
			var b strings.Builder
			if me.Info.Code != "" {
				b.WriteString("[")
				b.WriteString(me.Info.Code)
				b.WriteString("] ")
			}
			b.WriteString(me.Info.Message)
			if me.Info.Remediation != "" {
				b.WriteString("\n")
				b.WriteString(me.Info.Remediation)
			}
			if verifyResult != nil {
				if summary := verifyFailureSummary(verifyResult); summary != "" {
					b.WriteString("\n")
					b.WriteString(summary)
				}
			}
			return b.String()
		}
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func verifyFailureSummary(result *VerifyResult) string {
	if result == nil {
		return ""
	}

	var parts []string
	for _, table := range result.Tables {
		if table.Match {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: sqlite=%d postgres=%d", table.Table, table.SourceRows, table.DestRows))
	}
	for _, check := range result.Checks {
		if check.Matched {
			continue
		}
		label := check.Name
		if check.Table != "" {
			label = check.Table
			if check.Column != "" {
				label += "." + check.Column
			}
		}
		if check.Message != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", label, check.Message))
		} else {
			parts = append(parts, label)
		}
	}

	if len(parts) == 0 {
		return ""
	}
	const maxParts = 8
	if len(parts) > maxParts {
		return strings.Join(parts[:maxParts], "; ") + fmt.Sprintf("; ... and %d more", len(parts)-maxParts)
	}
	return strings.Join(parts, "; ")
}
