package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// SetShardTabletControlCmd updates shard tablet controls via vtctld.
func SetShardTabletControlCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace            string
		shard               string
		tabletType          string
		cells               []string
		deniedTables        []string
		remove              bool
		disableQueryService bool
	}

	cmd := &cobra.Command{
		Use:   "set-shard-tablet-control <database> <branch>",
		Short: "Update shard tablet controls for a branch",
		Long: "Update live shard tablet controls from the cluster via vtctld, " +
			"including denied tables used during MoveTables cleanup.",
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			if flags.keyspace == "" {
				return fmt.Errorf("keyspace is required")
			}
			if flags.shard == "" {
				return fmt.Errorf("shard is required")
			}
			if flags.tabletType == "" {
				return fmt.Errorf("tablet-type is required")
			}

			removeSet := cmd.Flags().Changed("remove")
			disableQueryServiceSet := cmd.Flags().Changed("disable-query-service")
			if !removeSet && len(flags.deniedTables) == 0 && !disableQueryServiceSet {
				return fmt.Errorf("must specify at least one of --remove, --denied-tables, or --disable-query-service")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Updating tablet controls for %s/%s on %s…",
					flags.keyspace, flags.shard,
					progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			req := &ps.VtctldSetShardTabletControlRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
				Shard:        flags.shard,
				TabletType:   flags.tabletType,
				Cells:        flags.cells,
				DeniedTables: flags.deniedTables,
			}
			if removeSet {
				req.Remove = &flags.remove
			}
			if disableQueryServiceSet {
				req.DisableQueryService = &flags.disableQueryService
			}

			data, err := client.Vtctld.SetShardTabletControl(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace name")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Shard name (e.g. \"-\" for unsharded)")
	cmd.Flags().StringVar(&flags.tabletType, "tablet-type", "", "Tablet type (e.g. rdonly, replica, primary)")
	cmd.Flags().StringSliceVar(&flags.cells, "cells", nil, "Cells to update (comma-separated)")
	cmd.Flags().StringSliceVar(&flags.deniedTables, "denied-tables", nil, "Tables to add to or remove from the denylist (comma-separated)")
	cmd.Flags().BoolVar(&flags.remove, "remove", false, "Remove tablet controls (or specific denied tables when combined with --denied-tables)")
	cmd.Flags().BoolVar(&flags.disableQueryService, "disable-query-service", false, "Disable query service on the provided tablets")
	cmd.MarkFlagRequired("keyspace")
	cmd.MarkFlagRequired("shard")
	cmd.MarkFlagRequired("tablet-type")

	return cmd
}
