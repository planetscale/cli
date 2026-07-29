package cmdutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/planetscale/cli/internal/planetscale"
)

func TestGlobalJSONErrorAuthRequired(t *testing.T) {
	resp := GlobalJSONError(errors.New("not authenticated yet. Please run 'pscale auth login'"))
	if resp.Status != "action_required" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Code() != "NO_AUTH" {
		t.Fatalf("code = %q", resp.Code())
	}
	if len(resp.NextSteps) == 0 || resp.NextSteps[0] != AgentAuthLoginCmd() {
		t.Fatalf("next_steps = %#v", resp.NextSteps)
	}
}

func TestGlobalJSONErrorInvalidUsageSkipsAuth(t *testing.T) {
	resp := GlobalJSONError(errors.New("missing argument <database> \n\nUsage: ..."))
	if resp.Status != "action_required" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Code() != "INVALID_USAGE" {
		t.Fatalf("code = %q", resp.Code())
	}
	for _, step := range resp.NextSteps {
		if step == AgentAuthCheckCmd() || step == AgentAuthLoginCmd() {
			t.Fatalf("invalid usage should not suggest auth commands, got %#v", resp.NextSteps)
		}
	}
}

func TestGlobalJSONErrorOrgFlagPlacement(t *testing.T) {
	resp := GlobalJSONError(errors.New("unknown flag: --org"))
	if resp.Code() != "INVALID_FLAG_PLACEMENT" {
		t.Fatalf("code = %q", resp.Code())
	}
	if len(resp.NextSteps) != 2 || resp.NextSteps[1] != AgentDatabaseListCmd("") {
		t.Fatalf("next_steps = %#v", resp.NextSteps)
	}
}

func TestGlobalJSONErrorUnknownFlag(t *testing.T) {
	resp := GlobalJSONError(errors.New("unknown flag: --bogus"))
	if resp.Code() != "UNKNOWN_FLAG" {
		t.Fatalf("code = %q", resp.Code())
	}
}

func TestGlobalJSONErrorUnknownCommand(t *testing.T) {
	resp := GlobalJSONError(errors.New(`unknown command "wat" for "pscale"`))
	if resp.Code() != "UNKNOWN_COMMAND" {
		t.Fatalf("code = %q", resp.Code())
	}
	for _, step := range resp.NextSteps {
		if step == AgentAuthCheckCmd() {
			t.Fatalf("unknown command should not suggest auth check, got %#v", resp.NextSteps)
		}
	}
}

func TestGlobalJSONErrorConfirmationRequired(t *testing.T) {
	resp := GlobalJSONError(errors.New("cannot confirm deletion of database (run with --force to override)"))
	if resp.Status != "action_required" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Code() != "CONFIRMATION_REQUIRED" {
		t.Fatalf("code = %q", resp.Code())
	}
	if len(resp.NextSteps) != 2 {
		t.Fatalf("next_steps = %#v", resp.NextSteps)
	}
}

func TestGlobalJSONErrorNotFound(t *testing.T) {
	resp := GlobalJSONError(errors.New("database foo does not exist in organization bar"))
	if resp.Code() != "NOT_FOUND" {
		t.Fatalf("code = %q", resp.Code())
	}
	if len(resp.NextSteps) == 0 || resp.NextSteps[0] != AgentOrgListCmd() {
		t.Fatalf("next_steps = %#v", resp.NextSteps)
	}
}

func TestGlobalJSONErrorNetwork(t *testing.T) {
	networkErrors := []string{
		"dial tcp: lookup api.planetscale.com: no such host",
		"dial tcp 10.0.0.1:443: i/o timeout",
		"dial tcp 10.0.0.1:443: connect: connection refused",
		"read tcp 10.0.0.1:443: connection reset by peer",
		"net/http: TLS handshake timeout",
	}
	for _, msg := range networkErrors {
		if resp := GlobalJSONError(errors.New(msg)); resp.Code() != "NETWORK_ERROR" {
			t.Fatalf("code = %q for %q", resp.Code(), msg)
		}
	}
}

func TestGlobalJSONErrorOperationTimeoutIsNotNetwork(t *testing.T) {
	// Operation and deadline timeouts are not transport failures; they must
	// fall through so command-specific handlers or the generic fallback apply.
	operationTimeouts := []string{
		"workflow cutover timed out waiting for traffic switch",
		"context deadline exceeded",
		"query timeout exceeded for SELECT",
	}
	for _, msg := range operationTimeouts {
		if resp := GlobalJSONError(errors.New(msg)); resp.Code() == "NETWORK_ERROR" {
			t.Fatalf("misclassified %q as NETWORK_ERROR", msg)
		}
	}
}

func TestGlobalJSONErrorServiceToken(t *testing.T) {
	resp := GlobalJSONError(errors.New("Authentication failed. Your service token appears to be invalid."))
	if resp.Status != "action_required" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Code() != "SERVICE_TOKEN_INVALID" {
		t.Fatalf("code = %q", resp.Code())
	}
}

func TestGlobalJSONErrorGenericFallback(t *testing.T) {
	resp := GlobalJSONError(errors.New("something went wrong"))
	if resp.Status != "error" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Code() != "COMMAND_FAILED" {
		t.Fatalf("code = %q", resp.Code())
	}
	if len(resp.NextSteps) != 2 || resp.NextSteps[0] != AgentAuthCheckCmd() {
		t.Fatalf("next_steps = %#v", resp.NextSteps)
	}
}

func TestGlobalJSONErrorSchemaMutationBlocked(t *testing.T) {
	resp := GlobalJSONError(&planetscale.Error{
		APICode: "schema_mutation_blocked",
	})
	if resp.Code() != "schema_mutation_blocked" {
		t.Fatalf("code = %q", resp.Code())
	}
	if len(resp.NextSteps) != 1 || resp.NextSteps[0] != "Wait for the active vtctld mutation or deploy to finish, then retry" {
		t.Fatalf("next_steps = %#v", resp.NextSteps)
	}
	for _, step := range resp.NextSteps {
		if step == AgentAuthCheckCmd() || step == AgentAuthLoginCmd() {
			t.Fatalf("schema_mutation_blocked should not suggest auth, got %#v", resp.NextSteps)
		}
	}
}

func TestGlobalJSONErrorPreservesOtherAPICodes(t *testing.T) {
	resp := GlobalJSONError(&planetscale.Error{
		APICode: "unprocessable",
	})
	if resp.Code() != "unprocessable" {
		t.Fatalf("code = %q", resp.Code())
	}
}

func TestGlobalJSONErrorIssuesMirrorError(t *testing.T) {
	resp := GlobalJSONError(errors.New("something went wrong"))
	if len(resp.Issues) != 1 {
		t.Fatalf("issues = %#v", resp.Issues)
	}
	if resp.Issues[0].Message != resp.Error {
		t.Fatalf("issue message %q != error %q", resp.Issues[0].Message, resp.Error)
	}
}

func TestGlobalJSONErrorStripsANSI(t *testing.T) {
	colored := "database \x1b[1;34mfoo\x1b[0m does not exist in organization \x1b[1;34mbar\x1b[0m"
	resp := GlobalJSONError(errors.New(colored))
	want := "database foo does not exist in organization bar"
	if resp.Error != want {
		t.Fatalf("error = %q, want %q", resp.Error, want)
	}
	if resp.Issues[0].Message != want {
		t.Fatalf("issue message = %q, want %q", resp.Issues[0].Message, want)
	}
	if resp.Code() != "NOT_FOUND" {
		t.Fatalf("code = %q", resp.Code())
	}
}

func TestGlobalJSONErrorTTYRequired(t *testing.T) {
	login := GlobalJSONError(errors.New("the 'login' command requires an interactive shell (use --format json; browser opens when possible, then polls until approved)"))
	if login.Status != "action_required" || login.Code() != "TTY_REQUIRED" {
		t.Fatalf("login: status = %q, code = %q", login.Status, login.Code())
	}
	if len(login.NextSteps) == 0 || login.NextSteps[0] != AgentAuthLoginCmd() {
		t.Fatalf("login next_steps = %#v", login.NextSteps)
	}

	replay := GlobalJSONError(errors.New("--replay requires an interactive terminal"))
	if replay.Status != "action_required" || replay.Code() != "TTY_REQUIRED" {
		t.Fatalf("replay: status = %q, code = %q", replay.Status, replay.Code())
	}
	for _, step := range replay.NextSteps {
		if step == AgentAuthLoginCmd() {
			t.Fatalf("replay should not suggest auth login, got %#v", replay.NextSteps)
		}
	}
}

func TestGlobalJSONErrorUsesCmdErrorMessage(t *testing.T) {
	resp := GlobalJSONError(&Error{
		Msg:      "You are not authenticated.",
		ExitCode: ActionRequestedExitCode,
	})
	if resp.Error != "You are not authenticated." {
		t.Fatalf("error = %q", resp.Error)
	}
	if resp.Status != "action_required" {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestGlobalJSONErrorKeepsActionRequiredForUnclassified(t *testing.T) {
	// An action-required exit code must survive classification even when the
	// message matches no known pattern.
	resp := GlobalJSONError(&Error{
		Msg:      "operator approval pending",
		ExitCode: ActionRequestedExitCode,
	})
	if resp.Status != "action_required" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Code() != "COMMAND_FAILED" {
		t.Fatalf("code = %q", resp.Code())
	}
}

func TestReportGlobalJSONError(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := ReportGlobalJSONError(&out, &errOut, errors.New("boom"))
	if exit != FatalErrExitCode {
		t.Fatalf("exit = %d", exit)
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", errOut.String())
	}

	var resp JSONErrorResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Status != "error" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Error == "" || len(resp.Issues) != 1 || len(resp.NextSteps) == 0 {
		t.Fatalf("resp = %#v", resp)
	}
}

func TestReportGlobalJSONErrorActionRequiredExitCode(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := ReportGlobalJSONError(&out, &errOut, &Error{
		Msg:      "needs action",
		ExitCode: ActionRequestedExitCode,
	})
	if exit != ActionRequestedExitCode {
		t.Fatalf("exit = %d", exit)
	}
}
