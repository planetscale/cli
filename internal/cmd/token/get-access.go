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

			name := args[0]

			req := &planetscale.GetServiceTokenAccessRequest{
				ID:           name,
				Organization: cfg.Organization,
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Creating service token in org %s", cmdutil.BoldBlue(cfg.Organization)))
			defer end()

			accesses, err := client.ServiceTokens.GetAccess(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("access %s does not exist in organization %s\n",
						cmdutil.BoldBlue(name), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewServiceTokenAccessPrinter(accesses))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
