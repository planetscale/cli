package keyspace

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ReadOnlyRegionsRemoveCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <database> <branch> <keyspace> <region>",
		Short: "Remove a read-only region from a keyspace",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace", "region"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, keyspace, region := args[0], args[1], args[2], args[3]

			client, current, err := getReadOnlyRegions(cmd, ch, database, branch, keyspace)
			if err != nil {
				return err
			}
			index := readOnlyRegionIndex(current, region)
			if index < 0 {
				return fmt.Errorf("read-only region %s is not configured for keyspace %s", printer.BoldBlue(region), printer.BoldBlue(keyspace))
			}

			configs := readOnlyRegionConfigs(current)
			configs = append(configs[:index], configs[index+1:]...)

			end := ch.Printer.PrintProgress(fmt.Sprintf("Removing read-only region %s from keyspace %s", printer.BoldBlue(region), printer.BoldBlue(keyspace)))
			defer end()

			regions, err := updateReadOnlyRegions(cmd, ch, client, database, branch, keyspace, configs)
			if err != nil {
				return err
			}
			end()

			return printReadOnlyRegionMutation(ch, fmt.Sprintf("Removed read-only region %s from keyspace %s.", printer.BoldBlue(region), printer.BoldBlue(keyspace)), regions)
		},
	}

	return cmd
}
