package configprofile

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "list <database> <branch>",
		Aliases: []string{"ls"},
		Short:   "List configuration profiles for a Neki branch",
		Args:    cmdutil.ExactArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching configuration profiles for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()
			profiles, err := client.NekiShardConfigurationProfiles.List(cmd.Context(), &ps.ListNekiShardConfigurationProfilesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleError(err, database, branch, "")
			}
			end()

			if len(profiles) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No configuration profiles found in %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}
			return ch.Printer.PrintResource(toProfiles(profiles))
		},
	}
}
