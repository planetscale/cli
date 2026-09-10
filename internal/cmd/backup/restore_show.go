package backup

import (
	"fmt"
	"strings"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func RestoreShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <database> <branch> <backup>",
		Short: "Show the sizes a Neki backup restore will use if you do not override them",
		Long: `Show the configuration profile and router sizes a Neki backup restore will use.

These values come from the live source branch, the same source the dashboard
pickers use. They are not stored on the backup. If the source branch was resized
after the backup, this command shows the current sizes.

<branch> is the source branch the backup belongs to, same as backup show.`,
		Args: cmdutil.RequiredArgs("database", "branch", "backup"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			backup := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			db, err := client.Databases.Get(ctx, &planetscale.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			if db.Kind != planetscale.DatabaseEngineNeki {
				return fmt.Errorf("backup restore show is only supported for Neki databases")
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching restore defaults for backup %s on %s/%s", printer.BoldBlue(backup), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			if _, err := client.Backups.Get(ctx, &planetscale.GetBackupRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Backup:       backup,
			}); err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("backup %s does not exist in branch %s of %s (organization: %s)",
						printer.BoldBlue(backup), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			profiles, err := client.NekiShardConfigurationProfiles.List(ctx, &planetscale.ListNekiShardConfigurationProfilesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			routers, err := client.NekiRouters.List(ctx, &planetscale.ListNekiRoutersRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return printRestoreDefaults(ch, toRestoreDefaults(database, branch, backup, profiles, routers))
		},
	}
}

type restoreDefaultProfile struct {
	Name        string `header:"name" json:"name"`
	Default     bool   `header:"default" json:"default"`
	ClusterSize string `header:"cluster size" json:"cluster_size"`
	Replicas    int    `header:"replicas" json:"replicas"`
}

type restoreDefaultRouter struct {
	Name            string `header:"name" json:"name"`
	Default         bool   `header:"default" json:"default"`
	Size            string `header:"size" json:"size"`
	ReplicasPerCell int    `header:"replicas per cell" json:"replicas_per_cell"`
}

type restoreDefaults struct {
	Database              string                  `json:"database"`
	Branch                string                  `json:"branch"`
	Backup                string                  `json:"backup"`
	ConfigurationProfiles []restoreDefaultProfile `json:"configuration_profiles"`
	Routers               []restoreDefaultRouter  `json:"routers"`
}

func toRestoreDefaults(database, branch, backup string, profiles []*planetscale.NekiShardConfigurationProfile, routers []*planetscale.NekiRouter) *restoreDefaults {
	out := &restoreDefaults{
		Database:              database,
		Branch:                branch,
		Backup:                backup,
		ConfigurationProfiles: make([]restoreDefaultProfile, 0, len(profiles)),
		Routers:               make([]restoreDefaultRouter, 0, len(routers)),
	}
	for _, profile := range profiles {
		out.ConfigurationProfiles = append(out.ConfigurationProfiles, restoreDefaultProfile{
			Name:        profile.Name,
			Default:     profile.Default,
			ClusterSize: profile.ClusterSize,
			Replicas:    profile.Replicas,
		})
	}
	for _, router := range routers {
		size := ""
		if router.SKU != nil {
			size = router.SKU.Name
			if router.SKU.DisplayName != "" {
				size = router.SKU.DisplayName
			}
		}
		out.Routers = append(out.Routers, restoreDefaultRouter{
			Name:            router.Name,
			Default:         router.Default,
			Size:            size,
			ReplicasPerCell: router.ReplicasPerCell,
		})
	}
	return out
}

func printRestoreDefaults(ch *cmdutil.Helper, defaults *restoreDefaults) error {
	if ch.Printer.Format() != printer.Human {
		return ch.Printer.PrintResource(defaults)
	}

	ch.Printer.Printf("If backup %s is restored without --config-profile or --router, these live source sizes are used:\n\n", printer.BoldBlue(defaults.Backup))

	var buf strings.Builder
	if len(defaults.ConfigurationProfiles) == 0 {
		ch.Printer.Println("Configuration profiles: none")
	} else {
		ch.Printer.Println("Configuration profiles")
		tableprinter.Print(&buf, defaults.ConfigurationProfiles)
		ch.Printer.Print(buf.String())
	}

	buf.Reset()
	if len(defaults.Routers) == 0 {
		ch.Printer.Println("Routers: none")
	} else {
		ch.Printer.Println("Routers")
		tableprinter.Print(&buf, defaults.Routers)
		ch.Printer.Print(buf.String())
	}
	return nil
}
