package deployrequest

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// CreateCmd is the command for creating deploy requests.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		into  string
		notes string
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch> [flags]",
		Short: "Create a deploy request from a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Request deploying of %s branch in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			dr, err := client.DeployRequests.Create(ctx, &planetscale.CreateDeployRequestRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				IntoBranch:   flags.into,
				Notes:        flags.notes,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				number := fmt.Sprintf("#%d", dr.Number)
				ch.Printer.Printf("Deploy request %s successfully created.\n\nView this deploy request in the browser: %s\n", printer.BoldBlue(number), printer.BoldBlue(dr.HtmlURL))
				if dr.Deployment.InstantDDLEligible {
					ch.Printer.Printf("This deploy request is instant DDL eligible. Pass the %s flag during deploy to deploy these schema changes using MySQL’s built-in ALGORITHM=INSTANT option. Deployment will be faster, but cannot be reverted.\n", printer.BoldYellow("--instant"))
				}
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.PersistentFlags().StringVar(&flags.into, "into", "", "Branch to deploy into. By default, it's the parent branch (if present) or the database's default branch.")
	cmd.PersistentFlags().StringVar(&flags.notes, "notes", "", "Notes to include with the deploy request.")

	return cmd
}
