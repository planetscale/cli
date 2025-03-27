package keyspace

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// SettingsCmd is a command that shows the settings for a keyspace
func SettingsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings <database> <branch> <keyspace>",
		Short: "Show the settings for a keyspace",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Retrieving settings for keyspace %s in %s/%s", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			ks, err := client.Keyspaces.Get(ctx, &ps.GetKeyspaceRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("keyspace %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(keyspace), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()
			return ch.Printer.PrintResource(buildKeyspaceSettings(ks))
		},
	}

	return cmd
}

// buildKeyspaceSettings converts a Keyspace API response to a KeyspaceSettings object for display
func buildKeyspaceSettings(ks *ps.Keyspace) *KeyspaceSettings {
	settings := &KeyspaceSettings{
		orig: ks,
	}

	// Set replication durability constraints if available
	if ks.ReplicationDurabilityConstraints != nil {
		// Map API values to more user-friendly display values
		var constraint string
		switch ks.ReplicationDurabilityConstraints.Strategy {
		case "available":
			constraint = "maximum"
		case "always":
			constraint = "minimum"
		case "lag":
			constraint = "dynamic"
		default:
			constraint = ks.ReplicationDurabilityConstraints.Strategy
		}
		settings.ReplicationDurabilityConstraintStrategy = constraint
	} else {
		settings.ReplicationDurabilityConstraintStrategy = "not set"
	}

	// Set VReplication flags if available
	if ks.VReplicationFlags != nil {
		settings.VReplicationFlags = VReplicationFlags{
			OptimizeInserts:           ks.VReplicationFlags.OptimizeInserts,
			AllowNoBlobBinlogRowImage: ks.VReplicationFlags.AllowNoBlobBinlogRowImage,
			VPlayerBatching:           ks.VReplicationFlags.VPlayerBatching,
		}
	}

	return settings
}