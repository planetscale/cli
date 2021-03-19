package token

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func DeleteCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <token>",
		Short: "delete an entire service token in an organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) != 1 {
				return cmd.Usage()
			}

			token := args[0]

			req := &planetscale.DeleteServiceTokenRequest{
				ID:           token,
				Organization: cfg.Organization,
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Deleting Token %s", cmdutil.BoldBlue(token)))
			defer end()

			if err := client.ServiceTokens.Delete(ctx, req); err != nil {
				return err
			}

			end()

			return nil
		},
	}

	return cmd
}
