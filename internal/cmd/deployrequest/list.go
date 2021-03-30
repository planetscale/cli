package deployrequest

import (
	"errors"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// ListCmd is the command for listing deploy requests.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List deploy requests",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your deploy-requests list in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s", cmdutil.ApplicationURL, cfg.Organization))
				if err != nil {
					return err
				}
				return nil
			}

			_, err = cfg.NewClientFromConfig()
			if err != nil {
				return err
			}
			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching deploy requests for %s", cmdutil.BoldBlue(database)))
			defer end()

			deployRequests, err := client.DeployRequests.List(ctx, &planetscale.ListDeployRequestsRequest{
				Organization: cfg.Organization,
				Database:     database,
			})
			if err != nil {
				return errors.Wrap(err, "error listing deploy requests")
			}
			end()

			if len(deployRequests) == 0 && !cfg.OutputJSON {
				fmt.Printf("No deploy requests exist for %s.\n", cmdutil.BoldBlue(database))
				return nil
			}

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewDeployRequestSlicePrinter(deployRequests))
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
