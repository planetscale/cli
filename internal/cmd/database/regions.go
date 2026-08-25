package database

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func RegionsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regions <command>",
		Short: "List regions available to a database",
	}

	cmd.AddCommand(RegionsListCmd(ch))
	return cmd
}

func RegionsListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:     "list <database>",
		Short:   "List regions available to a database",
		Aliases: []string{"ls"},
		Args:    cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database := args[0]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching regions for database %s...", printer.BoldBlue(database)))
			defer end()

			regions, err := client.Databases.ListRegions(cmd.Context(), &ps.ListDatabaseRegionsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(regions) == 0 && ch.Printer.Format() == printer.Human {
				if flags.page == 0 {
					ch.Printer.Println("No regions are available for this database.")
				} else {
					ch.Printer.Println("No regions found on this page.")
				}
				return nil
			}

			return ch.Printer.PrintResource(toDatabaseRegions(regions))
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}

type databaseRegion struct {
	Name                string `header:"name" json:"display_name"`
	Slug                string `header:"slug" json:"slug"`
	Location            string `header:"location" json:"location"`
	Provider            string `header:"provider" json:"provider"`
	Enabled             bool   `header:"enabled" json:"enabled"`
	MySQLSupported      bool   `header:"mysql_supported" json:"mysql_supported"`
	PostgreSQLSupported bool   `header:"postgresql_supported" json:"postgresql_supported"`
	orig                *ps.Region
}

func toDatabaseRegions(regions []*ps.Region) []*databaseRegion {
	out := make([]*databaseRegion, 0, len(regions))
	for _, region := range regions {
		out = append(out, &databaseRegion{
			Name:                region.Name,
			Slug:                region.Slug,
			Location:            region.Location,
			Provider:            region.Provider,
			Enabled:             region.Enabled,
			MySQLSupported:      region.MySQLSupported,
			PostgreSQLSupported: region.PostgreSQLSupported,
			orig:                region,
		})
	}
	return out
}

func (r *databaseRegion) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *databaseRegion) MarshalCSVValue() interface{} {
	return []*databaseRegion{r}
}
