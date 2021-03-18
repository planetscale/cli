package token

import (
	"context"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func AddAccessCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-access <token> <access> <access> ...",
		Short: "add access to a service token in the organization",
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

			req := &planetscale.AddServiceTokenAccessRequest{
				ID:           token,
				Database:     cfg.Database,
				Organization: cfg.Organization,
				Accesses:     perms,
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Adding access %s to database %s", cmdutil.BoldBlue(strings.Join(perms, ", ")), cmdutil.BoldBlue(cfg.Database)))
			defer end()

			access, err := client.ServiceTokens.AddAccess(ctx, req)
			if err != nil {
				return err
			}

			end()

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			err = printer.PrintOutput(isJSON, printer.NewServiceTokenAccessPrinter(access))
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.Database, "database", cfg.Database, "The database this project is using")
	cmd.MarkPersistentFlagRequired("database") // nolint:errcheck

	return cmd
}
