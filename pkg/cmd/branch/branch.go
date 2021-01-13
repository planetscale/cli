package branch

import (
	"fmt"

	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// BranchCmd handles the branching of a database.
func BranchCmd(cfg *config.Config) *cobra.Command {
	var notes string
	cmd := &cobra.Command{
		Use:     "branch <source-database> <branch-name> [options]",
		Short:   "Branch a production database",
		Aliases: []string{"b"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If the user does not provide a source database and a branch name,
			// show the usage.
			if len(args) != 2 {
				return cmd.Usage()
			}

			source := args[0]
			branch := args[1]

			// Simplest case, the names are equivalent
			if source == branch {
				return fmt.Errorf("A branch named '%s' already exists", branch)
			}

			_, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			// TODO: Call PlanetScale API client here to create the database branch.

			fmt.Println("database branching has not been implemented yet")
			return nil
		},
	}

	cmd.Flags().StringVar(&notes, "notes", "", "notes for the database branch")
	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(StatusCmd(cfg))

	return cmd
}
