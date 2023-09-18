package database

import (
	"fmt"
	"net/url"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
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
			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

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

			if web {
				ch.Printer.Println("🌐  Redirecting you to create a database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s?name=%s&notes=%s&showDialog=true", cmdutil.ApplicationURL, ch.Config.Organization, url.QueryEscape(createReq.Name), url.QueryEscape(createReq.Notes)))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			org, err := client.Organizations.Get(cmd.Context(), &ps.GetOrganizationRequest{
				Organization: ch.Config.Organization,
			})
			if err != nil {
				return err
			}

			if org.RemainingFreeDatabases == 0 && !force {
				ch.Printer.Printf("Organization [%s] does not have any free databases remaining\n", org.Name)
				ch.Printer.Printf("If you choose to continue, this database will be created on the Scaler plan. The monthly cost is %v.\n", printer.BoldYellow("$29"))
				confirmationName := "Y"
				confirmError := ch.Printer.ConfirmCommand(confirmationName, "create", "create this database")
				if confirmError != nil {
					return confirmError
				}
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
	cmd.Flags().StringVar(&createReq.Region, "region", "", "region for the database")

	cmd.Flags().String("plan", "", "plan for the database. Options: hobby, scaler, or scaler_pro")
	cmd.Flags().String("cluster-size", "", "cluster size for Scaler Pro databases. Options: PS_10, PS_20, PS_40, PS_80, PS_160, PS_320, PS_400")
	cmd.Flags().Bool("force", false, "Force the creation of a paid database")

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

	cmd.Flags().BoolP("web", "w", false, "Create a database in your web browser")

	return cmd
}
