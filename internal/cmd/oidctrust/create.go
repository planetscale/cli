package oidctrust

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name                 string
		provider             string
		maxTTL               int
		conditions           []string
		orgAccesses          []string
		databaseAccesses     []string
		allDatabasesAccesses []string
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an OIDC trust",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			conditions, err := parseConditions(flags.conditions)
			if err != nil {
				return err
			}
			accesses, err := parseAccesses(flags.orgAccesses, flags.databaseAccesses)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating OIDC trust %s", printer.BoldBlue(flags.name)))
			defer end()

			trust, err := client.OIDCTrusts.Create(cmd.Context(), &ps.CreateOIDCTrustRequest{
				Organization:         ch.Config.Organization,
				DisplayName:          flags.name,
				Provider:             flags.provider,
				MaxTTLSeconds:        flags.maxTTL,
				Conditions:           conditions,
				Accesses:             accesses,
				BulkDatabaseAccesses: flags.allDatabasesAccesses,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()

			return ch.Printer.PrintResource(toOIDCTrust(trust))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "The display name of the trust (required)")
	cmd.Flags().StringVar(&flags.provider, "provider", "", "The identity provider; run `pscale oidc-trust providers` to list them (required)")
	cmd.Flags().IntVar(&flags.maxTTL, "max-ttl", 0, "The maximum token lifetime in seconds (required)")
	cmd.Flags().StringArrayVar(&flags.conditions, "condition", nil, "A claim the token must match, as claim=value; repeatable")
	cmd.Flags().StringArrayVar(&flags.orgAccesses, "org-access", nil, "An organization access to grant; repeatable")
	cmd.Flags().StringArrayVar(&flags.databaseAccesses, "database-access", nil, "A database access to grant, as database:access; repeatable")
	cmd.Flags().StringArrayVar(&flags.allDatabasesAccesses, "all-databases-access", nil, "An access to grant on all databases; repeatable")

	cmd.MarkFlagRequired("name")     // nolint:errcheck
	cmd.MarkFlagRequired("provider") // nolint:errcheck
	cmd.MarkFlagRequired("max-ttl")  // nolint:errcheck

	return cmd
}
