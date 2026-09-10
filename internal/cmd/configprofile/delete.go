package configprofile

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <name>",
		Short:   "Delete a Neki configuration profile",
		Args:    cmdutil.ExactArgs("database", "branch", "name"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			if !force {
				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, name)
				if err := ch.Printer.ConfirmCommand(confirmationName, "delete configuration profile", "deletion of configuration profile"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting configuration profile %s", printer.BoldBlue(name)))
			defer end()
			err = client.NekiShardConfigurationProfiles.Delete(cmd.Context(), &ps.DeleteNekiShardConfigurationProfileRequest{
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
				ch.Printer.Printf("Configuration profile %s was successfully deleted from %s/%s.\n", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":                "configuration profile deleted",
				"configuration_profile": name,
				"branch":                branch,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete the configuration profile without confirmation")
	return cmd
}
