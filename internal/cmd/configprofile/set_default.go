package configprofile

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func SetDefaultCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "set-default <database> <branch> <name>",
		Short: "Set the default Neki configuration profile",
		Args:  cmdutil.ExactArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Setting configuration profile %s as default", printer.BoldBlue(name)))
			defer end()
			profile, err := client.NekiShardConfigurationProfiles.SetDefault(cmd.Context(), &ps.SetDefaultNekiShardConfigurationProfileRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				ConfigurationProfile: name,
			})
			if err != nil {
				return handleError(err, database, branch, name)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Configuration profile %s is now the default for %s/%s.\n", printer.BoldBlue(profile.Name), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}
			return ch.Printer.PrintResource(toProfile(profile))
		},
	}
}
