package oauthapplication

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List OAuth applications for an organization",
		Aliases: []string{"ls"},
		Long: `List OAuth applications for an organization.

Results are paginated. Use --page and --per-page to walk organizations
with more OAuth applications than one page holds.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}
			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching OAuth applications for %s", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			applications, err := client.OAuthApplications.List(cmd.Context(), &ps.ListOAuthApplicationsRequest{
				Organization: ch.Config.Organization,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(applications) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No OAuth applications exist in %s.\n", printer.BoldBlue(ch.Config.Organization))
				return nil
			}
			return ch.Printer.PrintResource(toOAuthApplications(applications))
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 25, "Number of results per page")
	return cmd
}
