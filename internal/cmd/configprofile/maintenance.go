package configprofile

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func MaintenanceCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "maintenance <database> <branch> <name>...",
		Short: "Run maintenance for one or more Neki configuration profiles",
		Long: `Run maintenance for one or more Neki configuration profiles, updating them
to the latest available image.

The upgrade is applied to the replicas first, followed by a switchover from the
old primary to an upgraded replica. That failover leads to a short period of
database unavailability (seconds), and all direct connections to those shards
are terminated.

Use 'pscale branch maintenance run' to maintain every profile on the branch.`,
		Args: cmdutil.RequiredArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, profiles := args[0], args[1], args[2:]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			label := strings.Join(profiles, ", ")
			end := ch.Printer.PrintProgress(fmt.Sprintf("Starting maintenance for configuration profile %s", printer.BoldBlue(label)))
			if len(profiles) > 1 {
				end = ch.Printer.PrintProgress(fmt.Sprintf("Starting maintenance for configuration profiles %s", printer.BoldBlue(label)))
			}
			defer end()

			if len(profiles) == 1 {
				err = client.NekiShardConfigurationProfiles.RunMaintenance(cmd.Context(), &ps.RunNekiShardConfigurationProfileMaintenanceRequest{
					Organization:         ch.Config.Organization,
					Database:             database,
					Branch:               branch,
					ConfigurationProfile: profiles[0],
				})
			} else {
				err = client.NekiShardConfigurationProfiles.RunBulkMaintenance(cmd.Context(), &ps.RunNekiShardConfigurationProfilesMaintenanceRequest{
					Organization:              ch.Config.Organization,
					Database:                  database,
					Branch:                    branch,
					ConfigurationProfileNames: profiles,
				})
			}
			if err != nil {
				profile := ""
				if len(profiles) == 1 {
					profile = profiles[0]
				}
				return handleError(err, database, branch, profile)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				if len(profiles) == 1 {
					ch.Printer.Printf("Maintenance for configuration profile %s was started.\n", printer.BoldBlue(profiles[0]))
					return nil
				}
				ch.Printer.Printf("Maintenance for configuration profiles %s was started.\n", printer.BoldBlue(label))
				return nil
			}

			return ch.Printer.PrintResource(map[string]interface{}{
				"result":                 "maintenance started",
				"configuration_profiles": profiles,
			})
		},
	}
}
