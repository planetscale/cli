package d1

import "fmt"

// Import stage names for progress reporting.
const (
	ImportStageConnecting    = "connecting"
	ImportStageSQLiteStaging = "sqlite_staging"
	ImportStageSchema        = "schema"
	ImportStagePgloader      = "pgloader"
	ImportStageIndexes       = "indexes"
	ImportStageSequences     = "sequences"
)

// Verify stage names for progress reporting.
const (
	VerifyStageRowCounts    = "row_counts"
	VerifyStageSequences    = "verify_sequences"
	VerifyStageBoolean      = "boolean_columns"
	VerifyStageFingerprints = "fingerprints"
	VerifyStageSampleRows   = "sample_rows"
)

// ImportProgress describes import or verify pipeline progress for CLI and agent feedback.
type ImportProgress struct {
	Stage   string `json:"stage"`
	Current int    `json:"current,omitempty"`
	Total   int    `json:"total,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// ImportProgressFunc receives progress updates during import or verify.
type ImportProgressFunc func(ImportProgress)

// FormatProgressMessage returns a human-readable progress line for CLI and Slack.
func FormatProgressMessage(p ImportProgress) string {
	switch p.Stage {
	case ImportStageConnecting:
		return "Connecting to PlanetScale Postgres..."
	case ImportStageSQLiteStaging:
		return "Staging SQLite database from export..."
	case ImportStageSchema:
		return "Applying PostgreSQL schema..."
	case ImportStagePgloader:
		if p.Total > 0 && p.Detail != "" {
			return fmt.Sprintf("Loading table %d/%d: %s", p.Current, p.Total, p.Detail)
		}
		if p.Detail != "" {
			return fmt.Sprintf("Loading table %s", p.Detail)
		}
		return "Loading tables with pgloader..."
	case ImportStageIndexes:
		return "Building indexes..."
	case ImportStageSequences:
		return "Resetting identity sequences..."
	case VerifyStageRowCounts:
		if p.Total > 0 && p.Detail != "" {
			return fmt.Sprintf("Counting rows %d/%d: %s", p.Current, p.Total, p.Detail)
		}
		return "Comparing row counts..."
	case VerifyStageSequences:
		return "Checking identity sequences..."
	case VerifyStageBoolean:
		return "Checking boolean column coercion..."
	case VerifyStageFingerprints:
		return "Checking table fingerprints..."
	case VerifyStageSampleRows:
		return "Sampling row content..."
	default:
		if p.Detail != "" {
			return p.Detail
		}
		return "Working..."
	}
}

func (opts ImportOptions) reportProgress(p ImportProgress) {
	if opts.OnProgress != nil {
		opts.OnProgress(p)
	}
	if opts.MigrationID != "" {
		notifyImportProgress(opts.NotifyAPI, opts.Org, opts.Database, opts.Branch, opts.MigrationID, opts.notifyBase, p)
	}
}

func (opts VerifyOptions) reportProgress(p ImportProgress) {
	if opts.OnProgress != nil {
		opts.OnProgress(p)
	}
	if opts.MigrationID != "" {
		notifyImportProgress(opts.NotifyAPI, opts.Org, opts.Database, opts.Branch, opts.MigrationID, opts.notifyBase, p)
	}
}
