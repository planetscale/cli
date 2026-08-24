package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/planetscale/cli/internal/cmd/agentguide"
	"github.com/planetscale/cli/internal/cmdutil"
)

const rootCLIHelperEnv = "PSCALE_ROOT_CLI_HELPER"

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

func runRootCLI(t *testing.T, args ...string) rootCLIResult {
	t.Helper()

	testArgs := append([]string{"-test.run=^TestRootCLIHelperProcess$", "--"}, args...)
	command := exec.Command(os.Args[0], testArgs...)
	command.Env = append(os.Environ(),
		rootCLIHelperEnv+"=1",
		"PSCALE_DISABLE_DEV_WARNING=true",
		"PSCALE_NO_UPDATE_NOTIFIER=1",
		"XDG_CONFIG_HOME="+t.TempDir(),
	)

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
