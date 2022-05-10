package dataimports

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// PasswordCmd handles branch passwords.
func DataImportsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "data-imports <command>",
		Short:             "Create, list, and delete branch data imports",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(TestConnectionCmd(ch))
	cmd.AddCommand(StartDataImportCmd(ch))
	cmd.AddCommand(MakePlanetScalePrimaryCmd(ch))
	cmd.AddCommand(MakePlanetScaleReplicaCmd(ch))
	cmd.AddCommand(GetDataImportCmd(ch))
	cmd.AddCommand(CancelDataImportCmd(ch))

	return cmd
}
