package d1

import "time"

// Severity levels for lint/plan issues.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Issue describes a migration concern with agent-friendly remediation.
type Issue struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Table       string `json:"table,omitempty"`
	Column      string `json:"column,omitempty"`
	Message     string `json:"message,omitempty"`
	Remediation string `json:"remediation"`
}

// NextStep guides agents to the next tool or command.
type NextStep struct {
	Tool    string `json:"tool,omitempty"`
	Command string `json:"command,omitempty"`
	Reason  string `json:"reason"`
}

// Response is the common JSON envelope for import d1 commands.
type Response struct {
	Status      string     `json:"status"`
	Command     string     `json:"command,omitempty"`
	Phase       string     `json:"phase,omitempty"`
	MigrationID string     `json:"migration_id,omitempty"`
	Issues      []Issue    `json:"issues,omitempty"`
	NextSteps   []NextStep `json:"next_steps,omitempty"`
	Reminder    string     `json:"reminder,omitempty"`
	Data        any        `json:"data,omitempty"`
	Error       *ErrorInfo `json:"error,omitempty"`
}

// ErrorInfo is a structured CLI/MCP error.
type ErrorInfo struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

// DoctorResult lists prerequisite checks.
type DoctorResult struct {
	Checks []DoctorCheck `json:"checks"`
	Ready  bool          `json:"ready"`
}

// DoctorCheck is a single prerequisite check.
type DoctorCheck struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Version     string `json:"version,omitempty"`
	Message     string `json:"message,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// LintResult summarizes lint output.
type LintResult struct {
	InputPath    string   `json:"input_path"`
	TableCount   int      `json:"table_count"`
	ErrorCount   int      `json:"error_count"`
	WarningCount int      `json:"warning_count"`
	Issues       []Issue  `json:"issues"`
	Tables       []string `json:"tables"`
}

// PlanResult is the migration plan JSON.
type PlanResult struct {
	MigrationID        string      `json:"migration_id"`
	InputPath          string      `json:"input_path"`
	Org                string      `json:"org"`
	Database           string      `json:"database"`
	Branch             string      `json:"branch"`
	RecommendedMethod  string      `json:"recommended_method"`
	EstimatedSizeBytes int64       `json:"estimated_size_bytes,omitempty"`
	Tables             []TablePlan `json:"tables"`
	CastRules          []CastRule  `json:"cast_rules"`
	LoadOrder          []string    `json:"load_order"`
	Issues             []Issue     `json:"issues"`
}

// TablePlan describes a table in the migration plan.
type TablePlan struct {
	Name        string `json:"name"`
	RowEstimate int    `json:"row_estimate,omitempty"`
	HasFK       bool   `json:"has_foreign_keys"`
}

// CastRule maps SQLite types to Postgres casts for pgloader.
type CastRule struct {
	SourceType string `json:"source_type"`
	TargetType string `json:"target_type"`
	Using      string `json:"using,omitempty"`
	Tables     string `json:"tables,omitempty"`
}

// ImportResult describes an import run.
type ImportResult struct {
	MigrationID  string         `json:"migration_id"`
	Method       string         `json:"method"`
	DryRun       bool           `json:"dry_run"`
	TablesLoaded int            `json:"tables_loaded,omitempty"`
	Timings      *ImportTimings `json:"timings,omitempty"`
	Lint         *LintResult    `json:"lint,omitempty"`
	Plan         *PlanResult    `json:"plan,omitempty"`
	CanProceed   bool           `json:"can_proceed"`
}

// ImportTimings breaks down import wall-clock time by phase.
type ImportTimings struct {
	TotalMs         int64             `json:"total_ms"`
	SQLiteStagingMs int64             `json:"sqlite_staging_ms,omitempty"`
	SchemaMs        int64             `json:"schema_ms,omitempty"`
	PgloaderMs      int64             `json:"pgloader_ms,omitempty"`
	IndexBuildMs    int64             `json:"index_build_ms,omitempty"`
	SequenceResetMs int64             `json:"sequence_reset_ms,omitempty"`
	TableLoads      []TableLoadTiming `json:"table_loads,omitempty"`
}

// TableLoadTiming is per-table pgloader duration.
type TableLoadTiming struct {
	Table string `json:"table"`
	Ms    int64  `json:"ms"`
}

// VerifyOptions configures post-import verification.
type VerifyOptions struct {
	Org         string
	Database    string
	Branch      string
	MigrationID string
	InputPath   string
	SQLitePath  string
	DestURI     string
	DBName      string // destination PostgreSQL database name (default postgres)
	NotifyAPI   NotifyAPIConfig
	OnProgress  ImportProgressFunc
	notifyBase  importNotificationPayload
}

// VerifyResult compares source and destination after import.
type VerifyResult struct {
	MigrationID string              `json:"migration_id"`
	Matched     bool                `json:"matched"`
	Tables      []TableVerifyResult `json:"tables"`
	Checks      []VerifyCheckResult `json:"checks,omitempty"`
}

// TableVerifyResult is per-table verification.
type TableVerifyResult struct {
	Table      string `json:"table"`
	SourceRows int64  `json:"source_rows"`
	DestRows   int64  `json:"dest_rows"`
	Match      bool   `json:"match"`
}

// Migration phases persisted in local state.
const (
	PhasePlanned   = "planned"
	PhaseImporting = "importing"
	PhaseImported  = "imported"
	PhaseVerified  = "verified"
	PhaseFailed    = "failed"
	PhaseComplete  = "complete"
)

// MigrationState is persisted local migration metadata.
type MigrationState struct {
	MigrationID   string    `json:"migration_id"`
	Org           string    `json:"org"`
	Database      string    `json:"database"`
	Branch        string    `json:"branch"`
	InputPath     string    `json:"input_path"`
	SQLitePath    string    `json:"sqlite_path,omitempty"`
	DBName        string    `json:"db_name,omitempty"`
	Method        string    `json:"method,omitempty"`
	Phase         string    `json:"phase"`
	SchemaApplied bool      `json:"schema_applied,omitempty"`
	LoadedTables  []string  `json:"loaded_tables,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
