package branch

import (
	"fmt"

	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func DeleteCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <db_name> <branch_name>",
		Short: "Delete a specific branch of a database and all of it's data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return cmd.Usage()
			}

			_, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			// TODO: Call PlanetScale API here to destroy the database branch.

			fmt.Println("destroying a database branch has not been implemented yet")
			return nil
		},
	}

	return cmd
}
