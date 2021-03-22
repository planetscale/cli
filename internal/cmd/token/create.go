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

func CreateCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a service token for the organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			req := &planetscale.CreateServiceTokenRequest{
				Organization: cfg.Organization,
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Creating service token in org %s", cmdutil.BoldBlue(cfg.Organization)))
			defer end()

			token, err := client.ServiceTokens.Create(ctx, req)
			if err != nil {
				return err
			}

			end()

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewServiceTokenPrinter(token))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
