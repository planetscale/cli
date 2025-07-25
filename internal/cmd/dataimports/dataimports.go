package dataimports

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// DataImportsCmd handles data imports into PlanetScale.
func DataImportsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "data-imports <command>",
		Short:             "Create, list, and delete branch data imports",
		Long:              "Create, list, and delete branch data imports.\n\nThis command is only supported for Vitess databases.",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(LintExternalDataSourceCmd(ch))
	cmd.AddCommand(StartDataImportCmd(ch))
	cmd.AddCommand(DetachExternalDatabaseCmd(ch))
	cmd.AddCommand(MakePlanetScalePrimaryCmd(ch))
	cmd.AddCommand(MakePlanetScaleReplicaCmd(ch))
	cmd.AddCommand(GetDataImportCmd(ch))
	cmd.AddCommand(CancelDataImportCmd(ch))

	return cmd
}
