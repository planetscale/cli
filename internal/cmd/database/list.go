package database

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ListCmd is the command for listing all databases for an authenticated user.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List databases",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				ch.Printer.Println("🌐  Redirecting you to your databases list in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s", cmdutil.ApplicationURL, ch.Config.Organization))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.ProgressPrintf("Fetching databases...")
			defer end()
			databases, err := client.Databases.List(ctx, &planetscale.ListDatabasesRequest{
				Organization: ch.Config.Organization,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("organization %s does not exist\n", cmdutil.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return errors.Wrap(err, "error listing databases")
				}
			}

			end()

			if len(databases) == 0 && !ch.Config.OutputJSON {
				ch.Printer.Println("No databases have been created yet.")
				return nil
			}

			err = ch.Printer.PrintResource(toDatabases(databases))
			if err != nil {
				return err
			}

			return nil
		},
		TraverseChildren: true,
	}

	cmd.Flags().BoolP("web", "w", false, "Open in your web browser")

	return cmd
}
