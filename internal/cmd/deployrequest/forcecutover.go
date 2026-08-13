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
		Short: "Force cutover for a deploy request stuck in the cutover phase",
		Long: `Force cutover for a deploy request stuck in the cutover phase.

Use this when a deploy request is in in_progress_cutover and cannot finish because
queries are holding metadata locks. Force cutover may terminate those blocking
queries so the schema swap can complete.

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
					Message: "Are you sure you want to force cutover? This may terminate queries blocking the schema swap.",
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
				ch.Printer.Printf("Force cutover requested for deploy request %s/%s. Cutover should complete once blocking queries are terminated.\n",
					printer.BoldBlue(database),
					printer.BoldBlue(dr.Number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force cutover without prompting for confirmation.")

	return cmd
}
