package branch

import (
	"context"
	"errors"
	"fmt"

	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing branches for a database.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <db-name>",
		Short: "List all branches of a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = context.Background()
			_, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return errors.New("<db_name> is missing")
			}

			_ = args[0]

			// TODO: Actually call database branch endpoints here.

			fmt.Println("branch listing not implemented yet")
			return nil
		},
	}

	return cmd
}
