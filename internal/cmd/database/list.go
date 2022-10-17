package database

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// ListCmd is the command for listing all databases for an authenticated user.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List databases",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
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

			end := ch.Printer.PrintProgress("Fetching databases...")
			defer end()
			databases, err := client.Databases.List(ctx, &planetscale.ListDatabasesRequest{
				Organization: ch.Config.Organization,
			}, planetscale.WithPage(flags.page), planetscale.WithPerPage(flags.perPage))
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if len(databases) == 0 && ch.Printer.Format() == printer.Human {
				if flags.page == 0 {
					ch.Printer.Println("No databases have been created yet.")
				} else {
					ch.Printer.Println("No databases found on this page.")
				}

				return nil
			}

			return ch.Printer.PrintResource(toDatabases(databases))
		},
		TraverseChildren: true,
	}

	cmd.Flags().BoolP("web", "w", false, "Open in your web browser")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")

	return cmd
}
