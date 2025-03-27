package keyspace

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func UpdateSettingsCmd(ch *cmdutil.Helper) *cobra.Command {
	updateReq := &ps.UpdateKeyspaceSettingsRequest{}

	var flags struct {
		replicationDurabilityConstraints *ps.ReplicationDurabilityConstraints
		vreplicationFlags                *ps.VReplicationFlags
	}

	flags.replicationDurabilityConstraints = &ps.ReplicationDurabilityConstraints{}
	flags.vreplicationFlags = &ps.VReplicationFlags{}

	cmd := &cobra.Command{
		Use:   "update-settings <database> <branch> <keyspace>",
		Short: "Update the settings for a keyspace",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			updateReq.Organization = ch.Config.Organization
			updateReq.Database = database
			updateReq.Branch = branch
			updateReq.Keyspace = keyspace

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating settings for keyspace %s in %s/%s", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
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

			// Get initial defaults from the API, then update them using flags.
			if ks.ReplicationDurabilityConstraints != nil {
				updateReq.ReplicationDurabilityConstraints = ks.ReplicationDurabilityConstraints
			}

			if ks.VReplicationFlags != nil {
				updateReq.VReplicationFlags = ks.VReplicationFlags
			}

			// Check if any relevant flags are changing replication durability constraints
			rdcChanged := cmd.Flags().Changed("replication-durability-constraints-strategy")
			if rdcChanged {
				if updateReq.ReplicationDurabilityConstraints == nil {
					updateReq.ReplicationDurabilityConstraints = &ps.ReplicationDurabilityConstraints{}
				}
				updateReq.ReplicationDurabilityConstraints.Strategy = constraintsToStrategy(flags.replicationDurabilityConstraints.Strategy)
			}

			// Check if any relevant flags are changing VReplication flags
			vrfChanged := cmd.Flags().Changed("vreplication-optimize-inserts") ||
				cmd.Flags().Changed("vreplication-enable-noblob-binlog-mode") ||
				cmd.Flags().Changed("vreplication-batch-replication-events")

			if vrfChanged {
				if updateReq.VReplicationFlags == nil {
					updateReq.VReplicationFlags = &ps.VReplicationFlags{}
				}

				if cmd.Flags().Changed("vreplication-optimize-inserts") {
					updateReq.VReplicationFlags.OptimizeInserts = flags.vreplicationFlags.OptimizeInserts
				}

				if cmd.Flags().Changed("vreplication-enable-noblob-binlog-mode") {
					updateReq.VReplicationFlags.AllowNoBlobBinlogRowImage = flags.vreplicationFlags.AllowNoBlobBinlogRowImage
				}

				if cmd.Flags().Changed("vreplication-batch-replication-events") {
					updateReq.VReplicationFlags.VPlayerBatching = flags.vreplicationFlags.VPlayerBatching
				}
			}

			if !rdcChanged && !vrfChanged {
				end()
				ch.Printer.Println("No changes were requested. No update performed.")
				return nil
			}

			k, err := updateKeyspaceSettings(ctx, client, updateReq)
			if err != nil {
				return err
			}

			end()

			return ch.Printer.PrintResource(toKeyspace(k))
		},
	}

	cmd.Flags().StringVar(&flags.replicationDurabilityConstraints.Strategy, "replication-durability-constraints-strategy", "maximum", "By default, replication is configured to maximize safety and data integrity. This setting may be relaxed to favor increased performance and reduced replication lag. Options: maximum, dynamic, minimum")
	cmd.Flags().BoolVar(&flags.vreplicationFlags.OptimizeInserts, "vreplication-optimize-inserts", true, "When enabled, skips sending INSERT events for rows that have yet to be replicated.")
	cmd.Flags().BoolVar(&flags.vreplicationFlags.AllowNoBlobBinlogRowImage, "vreplication-enable-noblob-binlog-mode", true, "When enabled, omits changed BLOB and TEXT columns from replication events, which reduces binlog sizes.")
	cmd.Flags().BoolVar(&flags.vreplicationFlags.VPlayerBatching, "vreplication-batch-replication-events", false, "When enabled, sends fewer queries to MySQL to improve performance.")

	return cmd
}

func updateKeyspaceSettings(ctx context.Context, client *ps.Client, updateReq *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
	k, err := client.Keyspaces.UpdateSettings(ctx, updateReq)
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return nil, fmt.Errorf("keyspace %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(updateReq.Keyspace), printer.BoldBlue(updateReq.Branch), printer.BoldBlue(updateReq.Database), printer.BoldBlue(updateReq.Organization))
		default:
			return nil, cmdutil.HandleError(err)
		}
	}

	return k, nil
}

// Helper function for translating the API value to a more semantic string.
func constraintsToStrategy(constraint string) string {
	switch constraint {
	case "maximum":
		return "available"
	case "minimum":
		return "always"
	case "dynamic":
		return "lag"
	default:
		return constraint
	}
}
