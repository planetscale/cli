package pgbouncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// RestartCmd restarts a dedicated PgBouncer, or the primary's local PgBouncer, in place.
func RestartCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force       bool
		wait        bool
		waitTimeout time.Duration
	}

	cmd := &cobra.Command{
		Use:   "restart <database> <branch> <name>",
		Short: "Restart a PgBouncer in place",
		Long: `Restart the PgBouncer processes behind a dedicated PgBouncer one instance at a
time. Use "primary" as the name to restart the local PgBouncer of the primary
instance instead.

Each instance starts a new PgBouncer process and keeps accepting connections
while the old process drains for up to two minutes. Connections still open
after the drain are closed and their in-flight transactions are interrupted.

The PgBouncer keeps its name, connection string, size, and parameters. A
restart that fails may have restarted some instances; nothing is rolled back.
Use "pscale pgbouncer restart status" to track it.`,
		Example: `  pscale pgbouncer restart mydb main read-pool
  pscale pgbouncer restart mydb main primary --wait
  pscale pgbouncer restart mydb main read-pool --force --format json`,
		Args: cmdutil.ExactArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			if !flags.force {
				if err := ch.Printer.ConfirmCommand(name, "restart PgBouncer", "restart of PgBouncer"); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Requesting a restart of PgBouncer %s on %s/%s...", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			restart, err := client.PostgresBouncers.CreateRestart(ctx, &ps.CreatePostgresBouncerRestartRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Bouncer:      name,
			})
			if err != nil {
				return restartError(ctx, cmd, ch, err, "write_database", database, branch, name)
			}
			end()

			if flags.wait {
				restart, err = waitForBouncerRestart(ctx, ch, client, database, branch, name, restart, flags.waitTimeout)
				if err != nil {
					return err
				}
			}

			return printRestart(ch, restart, database, branch, name)
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Restart a PgBouncer without confirmation")
	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait for the restart to finish before returning")
	// Each instance can take the full two-minute drain, one at a time.
	cmd.Flags().DurationVar(&flags.waitTimeout, "wait-timeout", 30*time.Minute, "Maximum time to wait for the restart to finish with --wait")

	cmd.AddCommand(RestartStatusCmd(ch))

	return cmd
}

// RestartStatusCmd shows one restart, or the latest restart, of a PgBouncer.
func RestartStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <database> <branch> <name> [<restart-id>]",
		Short: "Show the latest restart of a PgBouncer, or the restart with the given ID",
		Args:  cobra.MatchAll(cmdutil.RequiredArgs("database", "branch", "name"), cobra.MaximumNArgs(4)),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			if len(args) > 3 {
				id := args[3]
				end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching restart %s of PgBouncer %s on %s/%s...", printer.BoldBlue(id), printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
				defer end()

				restart, err := client.PostgresBouncers.GetRestart(ctx, &ps.GetPostgresBouncerRestartRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
					Bouncer:      name,
					ID:           id,
				})
				if err != nil {
					if cmdutil.ErrCode(err) == ps.ErrNotFound {
						return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, "read_branch",
							"restart %s of PgBouncer %s does not exist on %s/%s (organization: %s)",
							printer.BoldBlue(id), printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
					}
					return cmdutil.HandleError(err)
				}
				end()
				return ch.Printer.PrintResource(toPgBouncerRestart(restart))
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching latest restart of PgBouncer %s on %s/%s...", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			restarts, err := client.PostgresBouncers.ListRestarts(ctx, &ps.ListPostgresBouncerRestartsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Bouncer:      name,
			}, ps.WithPerPage(1))
			if err != nil {
				return restartError(ctx, cmd, ch, err, "read_branch", database, branch, name)
			}
			end()

			if len(restarts) == 0 {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("PgBouncer %s has no restarts.\n", printer.BoldBlue(name))
					return nil
				}
				return ch.Printer.PrintResource([]*PgBouncerRestart{})
			}

			return ch.Printer.PrintResource(toPgBouncerRestart(restarts[0]))
		},
	}

	return cmd
}

func restartError(ctx context.Context, cmd *cobra.Command, ch *cmdutil.Helper, err error, permission, database, branch, name string) error {
	if cmdutil.ErrCode(err) == ps.ErrNotFound {
		return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, permission,
			"PgBouncer %s does not exist on %s/%s (organization: %s)",
			printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
	}
	return cmdutil.HandleError(err)
}

// restartPollInterval is a variable so tests can poll without waiting.
var restartPollInterval = 5 * time.Second

func waitForBouncerRestart(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, database, branch, name string, restart *ps.PostgresBouncerRestart, timeout time.Duration) (*ps.PostgresBouncerRestart, error) {
	end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting for restart %s to finish...", printer.BoldBlue(restart.ID)))
	defer end()

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(restartPollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		if restart.Finished() {
			end()
			return restart, nil
		}

		select {
		case <-waitCtx.Done():
			if ctx.Err() != nil {
				return nil, restartWaitError(ch, ctx.Err(), restart, database, branch, name)
			}
			msg := fmt.Sprintf("restart %s did not finish within %s (current state: %s)", restart.ID, timeout, restart.State)
			if lastErr != nil {
				msg += fmt.Sprintf("; last poll failed: %s", lastErr)
			}
			if ch.Printer.Format() != printer.JSON {
				return nil, fmt.Errorf("%s. Check progress with '%s'", msg, restartStatusCommand(ch, database, branch, name, restart.ID))
			}
			// Timing out stops polling; the restart keeps running.
			return nil, restartProblemError(ch, "action_required", "PGBOUNCER_RESTART_WAIT_TIMEOUT", msg,
				restart, database, branch, name, cmdutil.ActionRequestedExitCode)
		case <-ticker.C:
		}

		latest, err := client.PostgresBouncers.GetRestart(waitCtx, &ps.GetPostgresBouncerRestartRequest{
			Organization: ch.Config.Organization,
			Database:     database,
			Branch:       branch,
			Bouncer:      name,
			ID:           restart.ID,
		})
		if err != nil {
			if waitCtx.Err() != nil || transientRestartError(err) {
				lastErr = err
				continue
			}
			return nil, restartWaitError(ch, cmdutil.HandleError(err), restart, database, branch, name)
		}
		lastErr = nil
		restart = latest
	}
}

// transientRestartError reports whether polling should keep going after err.
// Gateway errors lack API codes; expired tokens can also return ErrInternal.
func transientRestartError(err error) bool {
	var apiErr *ps.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == ps.ErrRetry || apiErr.HTTPStatus >= http.StatusInternalServerError
	}
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, io.ErrUnexpectedEOF)
}

// restartWaitError preserves the restart ID in JSON errors so callers can
// check its status before starting another restart.
func restartWaitError(ch *cmdutil.Helper, err error, restart *ps.PostgresBouncerRestart, database, branch, name string) error {
	if ch.Printer.Format() != printer.JSON {
		return fmt.Errorf("stopped waiting for restart %s: %w. Check progress with '%s'",
			restart.ID, err, restartStatusCommand(ch, database, branch, name, restart.ID))
	}
	resp := cmdutil.GlobalJSONError(err)
	exitCode := cmdutil.FatalErrExitCode
	if resp.Status == "action_required" {
		exitCode = cmdutil.ActionRequestedExitCode
	}
	return printRestartProblem(ch, resp.Status, resp.Error, resp.Issues, restart,
		append(resp.NextSteps, restartStatusCommand(ch, database, branch, name, restart.ID)+" --format json"), exitCode)
}

// printRestart prints the restart and returns an error when it ended without
// succeeding, so scripts see a non-zero exit code.
func printRestart(ch *cmdutil.Helper, restart *ps.PostgresBouncerRestart, database, branch, name string) error {
	switch restart.State {
	case ps.PostgresBouncerRestartStateFailed:
		return restartProblemError(ch, "error", "PGBOUNCER_RESTART_FAILED",
			fmt.Sprintf("restart %s of PgBouncer %s failed: %s", restart.ID, name, restartFailureReason(restart)),
			restart, database, branch, name, cmdutil.FatalErrExitCode)
	case ps.PostgresBouncerRestartStateCanceled:
		return restartProblemError(ch, "error", "PGBOUNCER_RESTART_CANCELED",
			fmt.Sprintf("restart %s of PgBouncer %s was canceled", restart.ID, name),
			restart, database, branch, name, cmdutil.FatalErrExitCode)
	}

	if ch.Printer.Format() != printer.Human {
		return ch.Printer.PrintResource(toPgBouncerRestart(restart))
	}

	if restart.State == ps.PostgresBouncerRestartStateSucceeded {
		ch.Printer.Printf("Restart %s of PgBouncer %s succeeded.\n", printer.BoldBlue(restart.ID), printer.BoldBlue(name))
	} else {
		ch.Printer.Printf("Restart %s of PgBouncer %s was accepted (state: %s).\n",
			printer.BoldBlue(restart.ID), printer.BoldBlue(name), printer.BoldBlue(restart.State))
	}
	return nil
}

type restartProblem struct {
	Status    string                   `json:"status"`
	Error     string                   `json:"error"`
	ID        string                   `json:"id"`
	State     string                   `json:"state"`
	Issues    []cmdutil.JSONErrorIssue `json:"issues"`
	NextSteps []string                 `json:"next_steps"`
}

// restartProblemError includes restart status and a follow-up command in JSON errors.
// Other formats return a plain error.
func restartProblemError(ch *cmdutil.Helper, status, code, message string, restart *ps.PostgresBouncerRestart, database, branch, name string, exitCode int) error {
	if ch.Printer.Format() != printer.JSON {
		return errors.New(message)
	}
	return printRestartProblem(ch, status, message, []cmdutil.JSONErrorIssue{{Code: code, Message: message}}, restart,
		[]string{restartStatusCommand(ch, database, branch, name, restart.ID) + " --format json"}, exitCode)
}

func printRestartProblem(ch *cmdutil.Helper, status, message string, issues []cmdutil.JSONErrorIssue, restart *ps.PostgresBouncerRestart, nextSteps []string, exitCode int) error {
	if err := ch.Printer.PrintJSON(restartProblem{
		Status:    status,
		Error:     message,
		ID:        restart.ID,
		State:     restart.State,
		Issues:    issues,
		NextSteps: nextSteps,
	}); err != nil {
		return err
	}
	return cmdutil.JSONReportedError(exitCode)
}

func restartStatusCommand(ch *cmdutil.Helper, database, branch, name, id string) string {
	return fmt.Sprintf("pscale pgbouncer restart status %s %s %s %s --org %s", database, branch, name, id, ch.Config.Organization)
}

func restartFailureReason(restart *ps.PostgresBouncerRestart) string {
	if restart.Error != "" {
		return restart.Error
	}
	return "no reason was reported; some instances may have restarted"
}

// PgBouncerRestart is the human/JSON/CSV view of a PgBouncer restart.
type PgBouncerRestart struct {
	ID          string `header:"id" json:"id"`
	Bouncer     string `header:"bouncer" json:"bouncer"`
	Target      string `header:"target" json:"target"`
	State       string `header:"state" json:"state"`
	Error       string `header:"error" json:"error"`
	Actor       string `header:"actor" json:"actor"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	CompletedAt *int64 `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`

	orig *ps.PostgresBouncerRestart
}

func (r *PgBouncerRestart) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *PgBouncerRestart) MarshalCSVValue() interface{} {
	return []*PgBouncerRestart{r}
}

func toPgBouncerRestart(restart *ps.PostgresBouncerRestart) *PgBouncerRestart {
	out := &PgBouncerRestart{
		ID:        restart.ID,
		Bouncer:   restart.Bouncer,
		Target:    restart.Target,
		State:     restart.State,
		Error:     restart.Error,
		Actor:     restart.Actor.Name,
		CreatedAt: printer.GetMilliseconds(restart.CreatedAt),
		orig:      restart,
	}
	if restart.CompletedAt != nil {
		completedAt := printer.GetMilliseconds(*restart.CompletedAt)
		out.CompletedAt = &completedAt
	}
	if out.Error == "" {
		out.Error = "-"
	}
	if out.Actor == "" {
		out.Actor = "-"
	}
	return out
}
