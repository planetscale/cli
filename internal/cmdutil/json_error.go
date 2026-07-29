package cmdutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/planetscale/cli/internal/planetscale"
)

// JSONErrorIssue mirrors the issue shape used by auth check, sql, and import
// d1 responses: a stable machine-readable code plus the human message.
type JSONErrorIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSONErrorResponse is the fallback JSON envelope for unhandled command
// errors. It follows the same schema as command-specific error responses:
// status, error, issues (code + message), next_steps.
type JSONErrorResponse struct {
	Status    string           `json:"status"`
	Error     string           `json:"error"`
	Issues    []JSONErrorIssue `json:"issues"`
	NextSteps []string         `json:"next_steps"`
}

// Code returns the classification code of the first issue.
func (r JSONErrorResponse) Code() string {
	if len(r.Issues) == 0 {
		return ""
	}
	return r.Issues[0].Code
}

// ansiEscape matches CSI color/style sequences. Error messages built for
// human output may embed them (e.g. via printer.BoldBlue); they must never
// reach a JSON payload.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// GlobalJSONError classifies an error for the global JSON envelope.
//
// Classification is by message text because this is the top-level fallback:
// command-specific handlers with typed errors have already had their chance.
// next_steps must earn their place: each case lists only the commands that
// actually address the failure, not generic bootstrap commands.
func GlobalJSONError(err error) JSONErrorResponse {
	msg := err.Error()
	status := "error"
	code := "COMMAND_FAILED"
	apiCode := ""

	if cmdErr, ok := errors.AsType[*Error](err); ok {
		if cmdErr.Msg != "" {
			msg = cmdErr.Msg
		}
		if cmdErr.ExitCode == ActionRequestedExitCode {
			status = "action_required"
		}
	}
	if perr, ok := err.(*planetscale.Error); ok {
		apiCode = perr.APICode
	}

	msg = ansiEscape.ReplaceAllString(msg, "")

	var nextSteps []string
	lower := strings.ToLower(msg)

	switch {
	// Auth problems: the fix is logging in (or fixing credentials), then verifying.
	case strings.Contains(msg, WarnAuthMessage) ||
		strings.Contains(lower, "not authenticated") ||
		strings.Contains(msg, "access token has expired"):
		status = "action_required"
		code = "NO_AUTH"
		nextSteps = []string{AgentAuthLoginCmd(), AgentAuthCheckCmd()}

	case strings.Contains(msg, "Authentication failed") && strings.Contains(msg, "service token"):
		status = "action_required"
		code = "SERVICE_TOKEN_INVALID"
		nextSteps = []string{
			"Verify --service-token-id and --service-token values (or PLANETSCALE_SERVICE_TOKEN_ID / PLANETSCALE_SERVICE_TOKEN)",
			AgentAuthCheckCmd(),
		}

	// --org on pscale root: show the corrected shape, not a re-auth loop.
	case strings.Contains(msg, "unknown flag: --org"):
		code = "INVALID_FLAG_PLACEMENT"
		nextSteps = []string{
			"Move --org onto the resource subcommand",
			AgentDatabaseListCmd(""),
		}

	// Wrong invocation: cobra's message already names the missing piece and
	// includes usage. Point at the guide for command shapes; auth is not the issue.
	case strings.Contains(msg, "missing argument") ||
		strings.Contains(msg, "missing arguments") ||
		strings.Contains(msg, "missing required flags") ||
		strings.Contains(msg, "required flag"):
		status = "action_required"
		code = "INVALID_USAGE"
		nextSteps = []string{
			"Re-run with the missing arguments or flags named in the error message",
			AgentGuideCmd(),
		}

	case strings.Contains(msg, "unknown command"):
		code = "UNKNOWN_COMMAND"
		nextSteps = []string{
			"Run `pscale --help` to list valid commands",
			AgentGuideCmd(),
		}

	case strings.Contains(msg, "unknown flag") || strings.Contains(msg, "unknown shorthand flag"):
		code = "UNKNOWN_FLAG"
		nextSteps = []string{
			"Run the same command with --help to list valid flags",
			AgentGuideCmd(),
		}

	// TTY-gated commands: messages vary ("interactive shell", "interactive
	// terminal"), so match the shared prefix. Only login has a JSON-mode
	// alternative worth suggesting.
	case strings.Contains(msg, "requires an interactive"):
		status = "action_required"
		code = "TTY_REQUIRED"
		if strings.Contains(lower, "login") {
			nextSteps = []string{AgentAuthLoginCmd()}
		} else {
			nextSteps = []string{"Re-run this command in an interactive terminal"}
		}

	// Confirmation-gated commands: the fix is user approval, then --force.
	case strings.Contains(msg, "run with --force"):
		status = "action_required"
		code = "CONFIRMATION_REQUIRED"
		nextSteps = []string{
			"Ask the user to approve this action",
			"Re-run the same command with --force after approval",
		}

	// Resource lookups that failed: discovery commands are the way forward.
	case strings.Contains(msg, "does not exist"):
		code = "NOT_FOUND"
		nextSteps = []string{
			AgentOrgListCmd(),
			AgentDatabaseListCmd(""),
		}

	// Connectivity: nothing an agent can self-serve beyond retry and status.
	// Match transport-level failures only — a bare "timeout" substring would
	// swallow operation or deadline timeouts that deserve command-specific
	// handling.
	case strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "connection reset by peer") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "tls handshake timeout") ||
		strings.Contains(lower, "temporary failure in name resolution"):
		code = "NETWORK_ERROR"
		nextSteps = []string{
			"Check network connectivity and --api-url, then retry",
			AgentAuthCheckCmd(),
		}

	// Unclassified: auth state is the one thing worth ruling out first.
	default:
		nextSteps = []string{
			AgentAuthCheckCmd(),
			AgentGuideCmd(),
		}
	}

	if apiCode != "" {
		code = apiCode
		if apiCode == "schema_mutation_blocked" {
			nextSteps = []string{
				"Wait for the active vtctld mutation or deploy to finish, then retry",
			}
		}
	}

	return JSONErrorResponse{
		Status:    status,
		Error:     msg,
		Issues:    []JSONErrorIssue{{Code: code, Message: msg}},
		NextSteps: nextSteps,
	}
}

// ReportGlobalJSONError writes the classified envelope for err to w and
// returns the process exit code. On encoding failure it falls back to a
// plain error line on errW.
func ReportGlobalJSONError(w, errW io.Writer, err error) int {
	resp := GlobalJSONError(err)
	if writeErr := WriteJSONError(w, resp); writeErr != nil {
		fmt.Fprintf(errW, "Error: %s\n", err)
	}

	if resp.Status == "action_required" {
		return ActionRequestedExitCode
	}
	if cmdErr, ok := errors.AsType[*Error](err); ok && cmdErr.ExitCode != 0 {
		return cmdErr.ExitCode
	}
	return FatalErrExitCode
}

// WriteJSONError writes a JSON error envelope to w.
func WriteJSONError(w io.Writer, resp JSONErrorResponse) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}
