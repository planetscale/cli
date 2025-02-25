package database

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// CreateCmd is the command for creating a database.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &ps.CreateDatabaseRequest{}

	cmd := &cobra.Command{
		Use:   "create <database>",
		Short: "Create a database instance",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			plan, err := cmd.Flags().GetString("plan")
			if err != nil {
				return err
			}

			createReq.Plan = ps.Plan(plan)

			clusterSize, err := cmd.Flags().GetString("cluster-size")
			if err != nil {
				return err
			}

			createReq.ClusterSize = ps.ClusterSize(clusterSize)

			createReq.Organization = ch.Config.Organization
			createReq.Name = args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress("Creating database...")
			defer end()
			database, err := client.Databases.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Database %s was successfully created.\n\nView this database in the browser: %s\n", printer.BoldBlue(database.Name), printer.BoldBlue(database.HtmlURL))
				return nil
			}

			return ch.Printer.PrintResource(toDatabase(database))
		},
	}

	cmd.Flags().StringVar(&createReq.Notes, "notes", "", "notes for the database")
	cmd.Flags().MarkDeprecated("notes", "is no longer available.")
	cmd.Flags().StringVar(&createReq.Region, "region", "", "region for the database")

	cmd.Flags().String("plan", "", "plan for the database. Options: hobby or scaler_pro")
	cmd.Flags().String("cluster-size", "PS_10", "cluster size for Scaler Pro databases. Options: PS_10, PS_20, PS_40, PS_80, PS_160, PS_320, PS_400")

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		client, err := ch.Client()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		regions, err := client.Regions.List(ctx, &ps.ListRegionsRequest{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		regionStrs := make([]string, 0)

		for _, r := range regions {
			if r.Enabled {
				regionStrs = append(regionStrs, r.Slug)
			}
		}

		return regionStrs, cobra.ShellCompDirectiveDefault
	})

	cmd.RegisterFlagCompletionFunc("cluster_size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		clusterSizes := []string{"PS_10", "PS_20", "PS_40", "PS_80", "PS_160", "PS_320", "PS_400"}

		return clusterSizes, cobra.ShellCompDirectiveDefault
	})

	cmd.RegisterFlagCompletionFunc("plan", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		plans := []string{"hobby", "scaler", "scaler_pro"}

		return plans, cobra.ShellCompDirectiveDefault
	})

	return cmd
}
