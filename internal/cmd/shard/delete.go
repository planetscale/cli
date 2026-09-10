package shard

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
		Use:     "delete <database> <branch> <shard-id>",
		Short:   "Delete a shard from a Neki database branch",
		Args:    cmdutil.ExactArgs("database", "branch", "shard-id"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, shardID := args[0], args[1], args[2]
			if !force {
				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, shardID)
				if err := ch.Printer.ConfirmCommand(confirmationName, "delete shard", "deletion of shard"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting shard %s from %s/%s", printer.BoldBlue(shardID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.NekiShards.Delete(cmd.Context(), &ps.DeleteNekiShardRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Shard:        shardID,
			})
			if err != nil {
				return handleError(err, database, branch, shardID, "")
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Shard %s was successfully deleted from %s/%s.\n", printer.BoldBlue(shardID), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":   "shard deleted",
				"shard_id": shardID,
				"branch":   branch,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete the shard without confirmation")
	return cmd
}
