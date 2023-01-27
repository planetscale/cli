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
		deployTo string
		into     string
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

			into := flags.into
			if len(into) == 0 && len(flags.deployTo) > 0 {
				into = flags.deployTo
			}

			dr, err := client.DeployRequests.Create(ctx, &planetscale.CreateDeployRequestRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				IntoBranch:   into,
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
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.PersistentFlags().StringVar(&flags.deployTo, "deploy-to", "", "Database branch to deploy into. Defaults to the parent of the branch (if present) or the database's default branch.")
	cmd.PersistentFlags().MarkDeprecated("deploy-to", "this flag will be removed in a future release. Use --into instead.")
	cmd.PersistentFlags().StringVar(&flags.into, "into", "", "Branch to deploy into. By default, it's the parent branch (if present) or the database's default branch.")

	return cmd
}
