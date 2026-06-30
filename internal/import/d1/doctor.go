package d1

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/planetscale/cli/internal/postgres"
	execabs "golang.org/x/sys/execabs"
)

const (
	checkOK   = "ok"
	checkWarn = "warn"
	checkFail = "fail"
	checkSkip = "skip"
)

// Doctor runs prerequisite checks for D1 migration.
func Doctor(ctx context.Context) (*DoctorResult, error) {
	checks := []DoctorCheck{
		checkWrangler(ctx),
		checkPgloader(ctx),
		checkPsql(),
		checkSQLite3(ctx),
		checkCloudflareEnv(),
	}

	result := &DoctorResult{Checks: checks, Ready: true}
	for _, c := range checks {
		if c.Status == checkFail {
			result.Ready = false
		}
	}
	return result, nil
}

func checkWrangler(ctx context.Context) DoctorCheck {
	for _, cmd := range []string{"wrangler", "npx"} {
		path, err := execabs.LookPath(cmd)
		if err != nil {
			continue
		}
		if cmd == "npx" {
			c := execabs.CommandContext(ctx, path, "wrangler", "--version")
			out, err := c.CombinedOutput()
			if err == nil {
				return DoctorCheck{
					Name:    "wrangler",
					Status:  checkOK,
					Version: strings.TrimSpace(string(out)),
				}
			}
			continue
		}
		c := execabs.CommandContext(ctx, path, "--version")
		out, err := c.CombinedOutput()
		if err == nil {
			return DoctorCheck{
				Name:    "wrangler",
				Status:  checkOK,
				Version: strings.TrimSpace(string(out)),
			}
		}
	}

	return DoctorCheck{
		Name:        "wrangler",
		Status:      checkWarn,
		Message:     "wrangler not found",
		Remediation: wranglerMissingRemediation,
	}
}

func checkPgloader(ctx context.Context) DoctorCheck {
	path, err := execabs.LookPath("pgloader")
	if err != nil {
		return DoctorCheck{
			Name:        "pgloader",
			Status:      checkFail,
			Message:     "pgloader not found",
			Remediation: pgloaderInstallRemediation,
		}
	}
	c := execabs.CommandContext(ctx, path, "--version")
	out, err := c.CombinedOutput()
	if err != nil {
		return DoctorCheck{
			Name:        "pgloader",
			Status:      checkFail,
			Message:     "pgloader found but --version failed",
			Remediation: "Reinstall pgloader",
		}
	}
	return DoctorCheck{
		Name:    "pgloader",
		Status:  checkOK,
		Version: strings.TrimSpace(string(out)),
	}
}

func checkPsql() DoctorCheck {
	major, minor, err := postgres.CheckPsqlVersion(10)
	if err != nil {
		return DoctorCheck{
			Name:        "psql",
			Status:      checkFail,
			Message:     err.Error(),
			Remediation: "Install PostgreSQL client tools: brew install postgresql@18",
		}
	}
	return DoctorCheck{
		Name:    "psql",
		Status:  checkOK,
		Version: fmt.Sprintf("%d.%d", major, minor),
	}
}

func checkSQLite3(ctx context.Context) DoctorCheck {
	path, err := execabs.LookPath("sqlite3")
	if err != nil {
		return DoctorCheck{
			Name:        "sqlite3",
			Status:      checkSkip,
			Message:     "sqlite3 CLI not found",
			Remediation: "Optional: install sqlite3 for verify and pgloader prep (brew install sqlite)",
		}
	}
	c := execabs.CommandContext(ctx, path, "--version")
	out, err := c.CombinedOutput()
	if err != nil {
		return DoctorCheck{
			Name:   "sqlite3",
			Status: checkSkip,
		}
	}
	return DoctorCheck{
		Name:    "sqlite3",
		Status:  checkOK,
		Version: strings.TrimSpace(string(out)),
	}
}

func checkCloudflareEnv() DoctorCheck {
	token := os.Getenv("CLOUDFLARE_API_TOKEN")
	account := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	if token != "" && account != "" {
		return DoctorCheck{
			Name:   "cloudflare_auth",
			Status: checkOK,
		}
	}
	return DoctorCheck{
		Name:        "cloudflare_auth",
		Status:      checkWarn,
		Message:     "CLOUDFLARE_API_TOKEN and/or CLOUDFLARE_ACCOUNT_ID not set",
		Remediation: "Set Cloudflare env vars for remote export, or pass --input with an existing dump",
	}
}

// DoctorReadinessError summarizes failed prerequisite checks for doctor/start.
func DoctorReadinessError(result *DoctorResult) error {
	if result == nil || result.Ready {
		return nil
	}

	var parts []string
	var remediations []string
	for _, c := range result.Checks {
		if c.Status != checkFail {
			continue
		}
		msg := c.Name
		if c.Message != "" {
			msg += ": " + c.Message
		}
		parts = append(parts, msg)
		if c.Remediation != "" {
			remediations = append(remediations, c.Remediation)
		}
	}

	message := "prerequisites not met"
	if len(parts) > 0 {
		message = strings.Join(parts, "; ")
	}
	remediation := strings.Join(remediations, "; ")
	if remediation == "" {
		remediation = "Run `pscale import d1 doctor` and fix failed checks"
	}
	return newMigrationError(ErrCodePrereqFailed, message, remediation)
}

// DoctorNextSteps suggests next actions after doctor.
func DoctorNextSteps(result *DoctorResult) []NextStep {
	if !result.Ready {
		return []NextStep{
			{Command: "pscale import d1 doctor", Reason: "Fix failed checks and re-run doctor"},
		}
	}
	return []NextStep{
		{Command: "wrangler d1 export <name> --remote --output ./d1-export.sql", Reason: "Export D1 database with wrangler"},
		{Command: "pscale import d1 lint --input ./d1-export.sql", Reason: "Lint the export before import"},
	}
}

// FindPgloader returns pgloader path.
func FindPgloader() (string, error) {
	path, err := execabs.LookPath("pgloader")
	if err != nil {
		return "", errMissingTool("pgloader", pgloaderInstallRemediation)
	}
	return path, nil
}

// FindSQLite3 returns sqlite3 path.
func FindSQLite3() (string, error) {
	path, err := execabs.LookPath("sqlite3")
	if err != nil {
		return "", errMissingTool("sqlite3", "Install with: brew install sqlite")
	}
	return path, nil
}
