package configprofile

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "show <database> <branch> <name>",
		Aliases: []string{"get"},
		Short:   "Show a Neki configuration profile",
		Args:    cmdutil.ExactArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, profile := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching configuration profile %s", printer.BoldBlue(profile)))
			defer end()
			result, err := client.NekiShardConfigurationProfiles.Get(cmd.Context(), &ps.GetNekiShardConfigurationProfileRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				ConfigurationProfile: profile,
			})
			if err != nil {
				return handleError(err, database, branch, profile)
			}
			end()
			return ch.Printer.PrintResource(toProfile(result))
		},
	}
}
