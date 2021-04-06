package snapshot

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CreateCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a new schema snapshot for a database branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database, branch := args[0], args[1]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Creating schema snapshot for %s in %s...", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database)))
			defer end()

			snapshot, err := client.SchemaSnapshots.Create(ctx, &planetscale.CreateSchemaSnapshotRequest{
				Organization: cfg.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}
			end()

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewSchemaSnapshotPrinter(snapshot))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
