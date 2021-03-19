package token

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-access <name>",
		Short: "fetch a service token and it's accesses",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) != 1 {
				return cmd.Usage()
			}

			req := &planetscale.GetServiceTokenAccessRequest{
				ID:           args[0],
				Organization: cfg.Organization,
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Creating service token in org %s", cmdutil.BoldBlue(cfg.Organization)))
			defer end()

			accesses, err := client.ServiceTokens.GetAccess(ctx, req)
			if err != nil {
				return err
			}

			end()

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			err = printer.PrintOutput(isJSON, printer.NewServiceTokenAccessPrinter(accesses))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
