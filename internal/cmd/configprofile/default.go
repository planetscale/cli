package configprofile

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func DefaultCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "default <database> <branch>",
		Short: "Show the default Neki configuration profile",
		Args:  cmdutil.ExactArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching the default configuration profile for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()
			profile, err := client.NekiShardConfigurationProfiles.GetDefault(cmd.Context(), &ps.GetDefaultNekiShardConfigurationProfileRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleError(err, database, branch, "")
			}
			end()
			return ch.Printer.PrintResource(toProfile(profile))
		},
	}
}
