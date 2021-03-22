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

func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list service tokens for the organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			req := &planetscale.ListServiceTokensRequest{
				Organization: cfg.Organization,
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Creating service token in org %s", cmdutil.BoldBlue(cfg.Organization)))
			defer end()

			tokens, err := client.ServiceTokens.List(ctx, req)
			if err != nil {
				return err
			}

			end()

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewServiceTokenSlicePrinter(tokens))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
