package deployrequest

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// ForceCutoverCmd is the command for forcing cutover on a stuck deploy request.
func ForceCutoverCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "force-cutover <database> <number>",
		Short: "Force cutover when a migration is delayed by a table lock",
		Long: `Force cutover when a deploy request is delayed waiting for a table lock.

The final step of a migration requires a brief table lock. Long-running
transactions can block that lock and delay completion. PlanetScale keeps
retrying for up to 1 hour, then forces cutover automatically.

Use this command to skip the wait: force cutover kills long-running
transactions that are blocking the table lock so the migration can finish.

Only allowed when the deployment state is in_progress_cutover.`,
		Args: cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			number := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			n, err := strconv.ParseUint(number, 10, 64)
			if err != nil {
				return fmt.Errorf("the argument <number> is invalid: %s", err)
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf(`cannot force cutover with the output format "%s" (run with --force to override)`, ch.Printer.Format())
				}

				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm force cutover (run with --force to override)")
				}

				prompt := &survey.Confirm{
					Message: "Force cutover now? This will kill long-running transactions that are blocking the table lock.",
					Default: false,
				}

				var confirm bool
				err = survey.AskOne(prompt, &confirm)
				if err != nil {
					if err == terminal.InterruptErr {
						os.Exit(0)
					} else {
						return err
					}
				}

				if !confirm {
					return errors.New("force cutover not confirmed, skipping")
				}
			}

			dr, err := client.DeployRequests.ForceCutover(ctx, &planetscale.ForceCutoverDeployRequestRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       n,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy request '%s/%s' does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully requested force cutover for deploy request %s/%s. Vitess will attempt again momentarily.\n",
					printer.BoldBlue(database),
					printer.BoldBlue(dr.Number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip the confirmation prompt and force cutover now.")

	return cmd
}
