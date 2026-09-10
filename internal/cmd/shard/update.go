package shard

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var displayName string

	cmd := &cobra.Command{
		Use:   "update <database> <branch> <shard-id>",
		Short: "Update a Neki shard's display name",
		Args:  cmdutil.ExactArgs("database", "branch", "shard-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, shardID := args[0], args[1], args[2]
			if !cmd.Flags().Changed("display-name") {
				return fmt.Errorf("--display-name flag is required")
			}

			var displayNameValue *string
			if displayName != "" {
				displayNameValue = &displayName
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating shard %s in %s/%s", printer.BoldBlue(shardID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			shard, err := client.NekiShards.Update(cmd.Context(), &ps.UpdateNekiShardRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Shard:        shardID,
				DisplayName:  displayNameValue,
			})
			if err != nil {
				return handleError(err, database, branch, shardID, "")
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Shard %s was successfully updated.\n", printer.BoldBlue(shardID))
			}
			return ch.Printer.PrintResource(toShard(shard))
		},
	}

	cmd.Flags().StringVar(&displayName, "display-name", "", "New display name; pass an empty string to clear it")
	cmd.MarkFlagRequired("display-name") // nolint:errcheck

	return cmd
}
