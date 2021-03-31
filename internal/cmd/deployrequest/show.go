package deployrequest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/browser"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// ShowCmd is the command to show a deploy request.
func ShowCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "show <database> <number>",
		Short: "Show a specific deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]
			number := args[1]

			if flags.web {
				fmt.Println("🌐  Redirecting you to your deploy request in your web browser.")
				return browser.OpenURL(fmt.Sprintf("%s/%s/%s/deploy-requests/%s", cmdutil.ApplicationURL, cfg.Organization, database, number))
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			n, err := strconv.ParseUint(number, 10, 64)
			if err != nil {
				return fmt.Errorf("The argument <number> is invalid: %s", err)
			}

			dr, err := client.DeployRequests.Get(ctx, &planetscale.GetDeployRequestRequest{
				Organization: cfg.Organization,
				Database:     database,
				Number:       n,
			})
			if err != nil {
				return err
			}

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewDeployRequestPrinter(dr))
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.web, "web", false, "Open in your web browser")

	return cmd
}
