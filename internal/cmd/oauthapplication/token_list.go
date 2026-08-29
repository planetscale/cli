package oauthapplication

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func TokenListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}
	cmd := &cobra.Command{
		Use:     "list <application-id>",
		Short:   "List tokens for an OAuth application",
		Aliases: []string{"ls"},
		Long: `List tokens for an OAuth application.

Results are paginated. Use --page and --per-page to walk applications
with more tokens than one page holds.`,
		Args: cmdutil.RequiredArgs("application-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			applicationID := args[0]
			client, err := ch.Client()
			if err != nil {
				return err
			}
			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching OAuth tokens for %s", printer.BoldBlue(applicationID)))
			defer end()
			tokens, err := client.OAuthApplications.ListTokens(cmd.Context(), &ps.ListOAuthTokensRequest{
				Organization:  ch.Config.Organization,
				ApplicationID: applicationID,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("OAuth application %s does not exist in organization %s",
						printer.BoldBlue(applicationID), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()
			if len(tokens) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No OAuth tokens exist for application %s.\n", printer.BoldBlue(applicationID))
				return nil
			}
			return ch.Printer.PrintResource(toOAuthTokens(tokens))
		},
	}
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 25, "Number of results per page")
	return cmd
}
