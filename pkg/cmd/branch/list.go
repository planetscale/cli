package branch

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/lensesio/tableprinter"
	errs "github.com/pkg/errors"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/printer"
	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing branches for a database.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <db-name>",
		Short: "List all branches of a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return errors.New("<db_name> is missing")
			}

			source := args[0]

			// TODO: Actually call database branch endpoints here.
			branches, err := client.DatabaseBranches.List(ctx, cfg.Organization, source)
			if err != nil {
				return errs.Wrap(err, "error listing databases")
			}

			// Technically, this should never actually happen.
			if len(branches) == 0 {
				fmt.Println("No branches exist for this database.")
				return nil
			}

			tableprinter.Print(os.Stdout, printer.NewDatabaseBranchSlicePrinter(branches))

			return nil
		},
	}

	return cmd
}
