package branch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// schemaRefreshPollInterval and schemaRefreshTimeout are vars so tests can shorten the wait.
var (
	schemaRefreshPollInterval = 2 * time.Second
	schemaRefreshTimeout      = 2 * time.Minute
)

func RefreshSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh-schema <database> <branch>",
		Short: "Refresh the schema for a MySQL branch",
		Long:  "Refresh the schema for a MySQL branch. Waits until the updated schema snapshot is ready so later commands, such as routing-rules, see it.",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Refreshing schema for %s in %s", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			err = client.DatabaseBranches.RefreshSchema(ctx, &planetscale.RefreshSchemaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return branchNotFoundOr(err, branch, database, ch.Config.Organization)
			}

			if err := waitUntilSchemaReady(ctx, client, ch.Config.Organization, database, branch); err != nil {
				return branchNotFoundOr(err, branch, database, ch.Config.Organization)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully refreshed schema for %s in %s.\n", printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result": "schema refreshed",
				},
			)
		},
	}

	return cmd
}

func branchNotFoundOr(err error, branch, database, org string) error {
	if cmdutil.ErrCode(err) == planetscale.ErrNotFound {
		return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
			printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(org))
	}
	return cmdutil.HandleError(err)
}

// waitUntilSchemaReady polls until the branch schema snapshot is ready.
// refresh-schema points the branch at a new snapshot before that snapshot
// is ready, and commands such as routing-rules fail during that window.
func waitUntilSchemaReady(ctx context.Context, client *planetscale.Client, org, database, branch string) error {
	ctx, cancel := context.WithTimeout(ctx, schemaRefreshTimeout)
	defer cancel()

	ticker := time.NewTicker(schemaRefreshPollInterval)
	defer ticker.Stop()

	for {
		resp, err := client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
			Organization: org,
			Database:     database,
			Branch:       branch,
		})
		if err != nil && cmdutil.ErrCode(err) == planetscale.ErrNotFound {
			return err
		}
		if err == nil && resp.SchemaReady {
			return nil
		}

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return errors.New("schema refreshing; please try again in a few moments")
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
