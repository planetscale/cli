package shard

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		configurationProfile string
		count                int
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create shards in a Neki configuration profile",
		Args:  cmdutil.ExactArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			if flags.count < 1 {
				return fmt.Errorf("--count must be greater than 0")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating %d shard(s) in configuration profile %s", flags.count, printer.BoldBlue(flags.configurationProfile)))
			defer end()

			response, err := client.NekiShards.Create(cmd.Context(), &ps.CreateNekiShardsRequest{
				Organization:         ch.Config.Organization,
				Database:             database,
				Branch:               branch,
				ConfigurationProfile: flags.configurationProfile,
				Count:                flags.count,
			})
			if err != nil {
				return handleError(err, database, branch, "", flags.configurationProfile)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Created %d shard(s) in configuration profile %s.\n", response.Created, printer.BoldBlue(flags.configurationProfile))
			}
			return ch.Printer.PrintResource(toShardCreation(response))
		},
	}

	cmd.Flags().StringVar(&flags.configurationProfile, "config-profile", "", "Configuration profile that will host the new shards")
	cmd.Flags().IntVar(&flags.count, "count", 1, "Number of shards to create")
	cmd.MarkFlagRequired("config-profile") // nolint:errcheck

	return cmd
}
