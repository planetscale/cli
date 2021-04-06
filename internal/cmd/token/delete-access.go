package token

import (
	"context"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func DeleteAccessCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-access <token> <access> <access> ...",
		Short: "delete access granted to a service token in the organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) < 2 {
				return cmd.Usage()
			}

			token, perms := args[0], args[1:]

			req := &planetscale.DeleteServiceTokenAccessRequest{
				ID:           token,
				Database:     cfg.Database,
				Organization: cfg.Organization,
				Accesses:     perms,
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Removing access %s on database %s", cmdutil.BoldBlue(strings.Join(perms, ", ")), cmdutil.BoldBlue(cfg.Database)))
			defer end()

			if err := client.ServiceTokens.DeleteAccess(ctx, req); err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("%s does not exist in %s\n",
						cmdutil.BoldBlue(cfg.Database), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.Database, "database", cfg.Database, "The database this project is using")
	cmd.MarkPersistentFlagRequired("database") // nolint:errcheck

	return cmd
}
