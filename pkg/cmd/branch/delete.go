package branch

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func DeleteCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <db_name> <branch_name>",
		Short: "Delete a specific branch of a database and all of it's data",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) != 2 {
				return cmd.Usage()
			}

			source := args[0]
			branch := args[1]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			err = client.DatabaseBranches.Delete(ctx, cfg.Organization, source, branch)
			if err != nil {
				return err
			}

			fmt.Printf("Successfully deleted branch `%s` from `%s`\n", branch, source)

			return nil
		},
	}

	return cmd
}
