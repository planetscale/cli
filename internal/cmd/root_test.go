package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/keyring"
	"github.com/planetscale/cli/internal/cmd/agentguide"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func TestAddCommandGroupsOmitsNeki(t *testing.T) {
	cmd := &cobra.Command{Use: "pscale"}
	addCommandGroups(cmd)

	for _, group := range cmd.Groups() {
		if group.ID == "neki" {
			t.Fatal("Neki command group should live under branch, not pscale root")
		}
	}
}

const rootCLIHelperEnv = "PSCALE_ROOT_CLI_HELPER"

func TestStoredTokenLoadingIsLazyAndReportsJSONErrors(t *testing.T) {
	for _, backend := range keyring.AvailableBackends() {
		switch backend {
		case keyring.KeychainBackend, keyring.SecretServiceBackend, keyring.KWalletBackend, keyring.WinCredBackend:
			t.Skip("requires file fallback; run with CGO_ENABLED=0 on macOS")
		}
	}
	home := writeHomeConfig(t, "")
	// Reading a directory as a token reliably fails without touching a keychain.
	if err := os.Mkdir(filepath.Join(home, ".config", "planetscale", "access-token"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"--help", "--format", "json"},
		{"--version", "--format", "json"},
		{"--skill", "--format", "json"},
	} {
		result := runRootCLIWithHome(t, home, args...)
		if result.exitCode != 0 {
			t.Fatalf("%v loaded stored credentials: stdout=%q stderr=%q", args, result.stdout, result.stderr)
		}
	}
	for _, args := range [][]string{
		{"org", "list", "--format", "json"},
		{"auth", "check", "--format", "json"},
		{"auth", "logout", "--format", "json"},
	} {
		result := runRootCLIWithHome(t, home, args...)
		if result.exitCode != cmdutil.FatalErrExitCode {
			t.Fatalf("%v exit code = %d, stdout=%q stderr=%q", args, result.exitCode, result.stdout, result.stderr)
		}
		var resp struct {
			Status    string   `json:"status"`
			Error     string   `json:"error"`
			Issues    []any    `json:"issues"`
			NextSteps []string `json:"next_steps"`
		}
		if err := json.Unmarshal([]byte(result.stdout), &resp); err != nil {
			t.Fatalf("%v did not return one JSON envelope: %v, stdout=%q", args, err, result.stdout)
		}
		if resp.Status != "error" || !strings.Contains(resp.Error, "access-token") || len(resp.Issues) == 0 || len(resp.NextSteps) == 0 {
			t.Fatalf("%v invalid JSON error envelope: %s", args, result.stdout)
		}
	}
}

func TestRootSkillFlag(t *testing.T) {
	for _, args := range [][]string{
		{"--skill", "--format", "json"},
		{"--skill=true", "--format", "json"},
		{"--skill=TRUE", "--format", "json"},
		{"--skill=1", "--format", "json"},
	} {
		t.Run(args[0], func(t *testing.T) {
			result := runRootCLI(t, args...)
			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
			}
			if result.stdout != agentguide.SkillDoc() {
				t.Fatalf("skill output differs (prefix=%q)", result.stdout[:min(40, len(result.stdout))])
			}
		})
	}
}

func TestRootSkillFlagDoesNotInterceptSubcommandValues(t *testing.T) {
	result := runRootCLI(t,
		"deploy-request", "review", "db", "1",
		"--comment", "--skill", "--format", "json",
	)

	if result.exitCode != cmdutil.ActionRequestedExitCode {
		t.Fatalf("exit code = %d, stdout = %q, stderr = %q", result.exitCode, result.stdout, result.stderr)
	}
	if strings.HasPrefix(result.stdout, "---\nname: pscale-cli\n") {
		t.Fatal("--skill flag value was intercepted by the root command")
	}
	if !strings.Contains(result.stdout, `"code": "NO_AUTH"`) {
		t.Fatalf("command did not reach authentication, stdout = %q", result.stdout)
	}
}

func TestRootSkillFlagDoesNotHideInvalidFlags(t *testing.T) {
	result := runRootCLI(t, "--format", "json", "--bogus", "--skill")

	if result.exitCode != cmdutil.FatalErrExitCode {
		t.Fatalf("exit code = %d, stdout = %q, stderr = %q", result.exitCode, result.stdout, result.stderr)
	}
	if strings.HasPrefix(result.stdout, "---\nname: pscale-cli\n") {
		t.Fatal("--skill hid an invalid flag")
	}
	if !strings.Contains(result.stdout, `"code": "UNKNOWN_FLAG"`) {
		t.Fatalf("expected UNKNOWN_FLAG error, stdout = %q", result.stdout)
	}
}

func TestRootSkillFlagIsNotInheritedBySubcommands(t *testing.T) {
	result := runRootCLI(t, "--format", "json", "auth", "check", "--skill")

	if result.exitCode != cmdutil.FatalErrExitCode {
		t.Fatalf("exit code = %d, stdout = %q, stderr = %q", result.exitCode, result.stdout, result.stderr)
	}
	if strings.HasPrefix(result.stdout, "---\nname: pscale-cli\n") {
		t.Fatal("subcommand --skill flag was intercepted by the root command")
	}
	if !strings.Contains(result.stdout, `"code": "UNKNOWN_FLAG"`) {
		t.Fatalf("expected UNKNOWN_FLAG error, stdout = %q", result.stdout)
	}
}

func TestFlagValueWinsOverConfigFile(t *testing.T) {
	home := writeHomeConfig(t, "org: config-org\n")

	result := runRootCLIWithHome(t, home,
		"database", "list",
		"--org", "flag-org",
		"--format", "json",
		"--api-url", "http://127.0.0.1:1/",
		"--service-token-id", "id",
		"--service-token", "secret",
	)

	if !strings.Contains(result.stdout, "organizations/flag-org") {
		t.Fatalf("--org did not reach the API request, stdout = %q, stderr = %q", result.stdout, result.stderr)
	}
}

func TestInterleavedFlagValueWinsOverConfigFile(t *testing.T) {
	home := writeHomeConfig(t, "org: config-org\n")

	result := runRootCLIWithHome(t, home,
		"database",
		"--org", "flag-org",
		"list",
		"--format", "json",
		"--api-url", "http://127.0.0.1:1/",
		"--service-token-id", "id",
		"--service-token", "secret",
	)

	if !strings.Contains(result.stdout, "organizations/flag-org") {
		t.Fatalf("interleaved --org did not reach the API request, stdout = %q, stderr = %q", result.stdout, result.stderr)
	}
}

func TestConfigFileFillsUnsetFlag(t *testing.T) {
	home := writeHomeConfig(t, "org: config-org\n")

	result := runRootCLIWithHome(t, home,
		"database", "list",
		"--format", "json",
		"--api-url", "http://127.0.0.1:1/",
		"--service-token-id", "id",
		"--service-token", "secret",
	)

	if !strings.Contains(result.stdout, "organizations/config-org") {
		t.Fatalf("config org was not used, stdout = %q, stderr = %q", result.stdout, result.stderr)
	}
}

func writeHomeConfig(t *testing.T, contents string) string {
	t.Helper()

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "planetscale")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "pscale.yml"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestTargetCommand(t *testing.T) {
	root := &cobra.Command{Use: "pscale"}
	parent := &cobra.Command{Use: "database"}
	parent.PersistentFlags().String("org", "", "")
	child := &cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}}
	parent.AddCommand(child)
	root.AddCommand(parent)

	if got := targetCommand(root, []string{"database", "list", "--org", "acme"}); got != child {
		t.Fatalf("target = %q, want database list", got.Name())
	}
	if got := targetCommand(root, []string{"database", "--org", "acme", "list"}); got != child {
		t.Fatalf("target with interleaved flag = %q, want database list", got.Name())
	}
	if got := targetCommand(root, nil); got != root {
		t.Fatalf("target = %q, want root", got.Name())
	}
	if got := targetCommand(root, []string{"nope"}); got != root {
		t.Fatalf("target = %q, want root for an unknown command", got.Name())
	}
}

func TestRootCLIHelperProcess(t *testing.T) {
	if os.Getenv(rootCLIHelperEnv) != "1" {
		return
	}

	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator == -1 {
		t.Fatal("missing argument separator")
	}

	os.Args = append([]string{"pscale"}, os.Args[separator+1:]...)
	exitCode := Execute(context.Background(), make(chan os.Signal, 1), []os.Signal{os.Interrupt}, "test", "test", "test")
	os.Exit(exitCode)
}

type rootCLIResult struct {
	exitCode int
	stdout   string
	stderr   string
}

// runRootCLI runs the CLI against an empty home directory, so the developer's
// own config and credentials stay out of the test.
func runRootCLI(t *testing.T, args ...string) rootCLIResult {
	t.Helper()
	return runRootCLIWithHome(t, t.TempDir(), args...)
}

// runRootCLIWithHome runs the CLI with $HOME pointed at dir, so a test can
// control the config file the CLI reads.
func runRootCLIWithHome(t *testing.T, home string, args ...string) rootCLIResult {
	t.Helper()

	testArgs := append([]string{"-test.run=^TestRootCLIHelperProcess$", "--"}, args...)
	command := exec.Command(os.Args[0], testArgs...)
	command.Env = append(os.Environ(),
		rootCLIHelperEnv+"=1",
		"PSCALE_NO_UPDATE_NOTIFIER=1",
		"XDG_CONFIG_HOME="+t.TempDir(),
	)
	if home != "" {
		command.Env = append(command.Env, "HOME="+home)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	exitCode := 0
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("run helper process: %v", err)
		}
		exitCode = exitError.ExitCode()
	}

	return rootCLIResult{
		exitCode: exitCode,
		stdout:   stdout.String(),
		stderr:   stderr.String(),
	}
}

func TestUnknownCommandsFail(t *testing.T) {
	for _, args := range [][]string{
		{"nonexistentcmd"},
		{"mcp", "install"},
		{"database", "nonexistentsub"},
		{"db", "nonexistentsub"},
		{"org", "sso", "nonexistentsub"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			result := runRootCLI(t, args...)
			if result.exitCode != cmdutil.FatalErrExitCode {
				t.Fatalf("exit code = %d, stdout = %q, stderr = %q", result.exitCode, result.stdout, result.stderr)
			}
			if !strings.Contains(result.stderr, "unknown command") {
				t.Fatalf("stderr = %q, want unknown command error", result.stderr)
			}
			if result.stdout != "" {
				t.Fatalf("stdout = %q, want empty", result.stdout)
			}
		})
	}
}

func TestUnknownCommandPointsAtParentHelp(t *testing.T) {
	result := runRootCLI(t, "role", "describe")
	if !strings.Contains(result.stderr, "Run 'pscale role --help' for usage.") {
		t.Fatalf("stderr = %q, want a pointer to pscale role --help", result.stderr)
	}

	result = runRootCLI(t, "role", "describe", "--format", "json")
	if !strings.Contains(result.stdout, "Run `pscale role --help` to list valid commands") {
		t.Fatalf("stdout = %q, want a next step for pscale role --help", result.stdout)
	}
}

func TestShowAliasesGet(t *testing.T) {
	for _, args := range [][]string{
		{"role", "show"},
		{"branch", "data-topology", "show"},
		{"branch", "routing-rules", "show"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			result := runRootCLI(t, args...)
			if strings.Contains(result.stderr, "unknown command") {
				t.Fatalf("stderr = %q, want show to resolve to get", result.stderr)
			}
			if !strings.Contains(result.stderr, "missing argument") {
				t.Fatalf("stderr = %q, want get's missing argument error", result.stderr)
			}
		})
	}
}

func TestUnknownCommandJSON(t *testing.T) {
	for _, args := range [][]string{
		{"nonexistentcmd", "--format", "json"},
		{"branch", "bogus", "--format", "json"},
		{"--format", "json", "database", "bogus"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			result := runRootCLI(t, args...)
			if result.exitCode != cmdutil.FatalErrExitCode {
				t.Fatalf("exit code = %d, stdout = %q, stderr = %q", result.exitCode, result.stdout, result.stderr)
			}
			if !strings.Contains(result.stdout, `"code": "UNKNOWN_COMMAND"`) {
				t.Fatalf("expected UNKNOWN_COMMAND, stdout = %q", result.stdout)
			}
		})
	}
}

func TestHelpStillWorks(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"database"},
		{"branch", "--format", "json"},
		{"--help"},
		{"database", "--help"},
		{"help", "database"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			result := runRootCLI(t, args...)
			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
			}
			if !strings.Contains(result.stdout, "Usage:") {
				t.Fatalf("stdout = %q, want help", result.stdout)
			}
		})
	}
}
