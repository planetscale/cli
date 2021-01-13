package branch

import (
	"fmt"

	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <db_name> <branch_name>",
		Short: "Get a specific branch of a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return cmd.Usage()
			}

			_, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			// TODO: Call PlanetScale API here to get the database branch.

			fmt.Println("getting a database branch has not been implemented yet")
			return nil
		},
	}

	return cmd
}
