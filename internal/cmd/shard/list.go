package shard

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		configurationProfile        string
		excludeConfigurationProfile string
		query                       string
		page                        int
		perPage                     int
	}

	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List shards for a Neki database branch",
		Args:    cmdutil.ExactArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching shards for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			shards, err := client.NekiShards.List(cmd.Context(), &ps.ListNekiShardsRequest{
				Organization:                ch.Config.Organization,
				Database:                    database,
				Branch:                      branch,
				ConfigurationProfile:        flags.configurationProfile,
				ExcludeConfigurationProfile: flags.excludeConfigurationProfile,
				Query:                       flags.query,
				Page:                        flags.page,
				PerPage:                     flags.perPage,
			})
			if err != nil {
				return handleError(err, database, branch, "", flags.configurationProfile)
			}
			end()

			if len(shards) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No shards found in %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toShards(shards))
		},
	}

	cmd.Flags().StringVar(&flags.configurationProfile, "config-profile", "", "List shards assigned to this configuration profile")
	cmd.Flags().StringVar(&flags.excludeConfigurationProfile, "exclude-config-profile", "", "Exclude shards assigned to this configuration profile")
	cmd.Flags().StringVarP(&flags.query, "query", "q", "", "Search by shard name, display name, or configuration profile")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	cmd.MarkFlagsMutuallyExclusive("config-profile", "exclude-config-profile")

	return cmd
}
