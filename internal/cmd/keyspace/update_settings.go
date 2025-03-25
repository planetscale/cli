package keyspace

import (
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

type KeyspaceUpdateSettingsRequest struct {
	ReplicationDurabilityConstraints *ps.ReplicationDurabilityConstraints `json:"replication_durability_constraints"`
	BinlogReplication                *ps.BinlogReplication                `json:"binlog_replication"`
}

func UpdateSettingsCmd(ch *cmdutil.Helper) *cobra.Command {
	updateReq := &ps.UpdateKeyspaceSettingsRequest{}

	var flags struct {
		replicationDurabilityConstraints *ps.ReplicationDurabilityConstraints
		binlogReplication                *ps.BinlogReplication
	}

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

			if cmd.Flags().Changed("replication-durability-constraints-strategy") {
				updateReq.ReplicationDurabilityConstraints.Strategy = flags.replicationDurabilityConstraints.Strategy
			}

			if cmd.Flags().Changed("binlog-replication-optimize-inserts") {
				updateReq.BinlogReplication.OptimizeInserts = flags.binlogReplication.OptimizeInserts
			}

			if cmd.Flags().Changed("binlog-replication-allow-no-blob-binlog-row-image") {
				updateReq.BinlogReplication.AllowNoBlobBinlogRowImage = flags.binlogReplication.AllowNoBlobBinlogRowImage
			}

			if cmd.Flags().Changed("binlog-replication-batch-binlog-statements") {
				updateReq.BinlogReplication.BatchBinlogStatements = flags.binlogReplication.BatchBinlogStatements
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			k, err := client.Keyspaces.UpdateSettings(ctx, updateReq)
			if err != nil {
				return err
			}

			return ch.Printer.PrintResource(toKeyspace(k))
		},
	}

	cmd.Flags().StringVar(&flags.replicationDurabilityConstraints.Strategy, "replication-durability-constraints-strategy", "maximum", "replication durability constraints strategy")
	cmd.Flags().BoolVar(&flags.binlogReplication.OptimizeInserts, "binlog-replication-optimize-inserts", true, "binlog replication optimize inserts")
	cmd.Flags().BoolVar(&flags.binlogReplication.AllowNoBlobBinlogRowImage, "binlog-replication-allow-no-blob-binlog-row-image", true, "binlog replication allow no blob binlog row image")
	cmd.Flags().BoolVar(&flags.binlogReplication.BatchBinlogStatements, "binlog-replication-batch-binlog-statements", false, "binlog replication batch binlog statements")
	return cmd
}
