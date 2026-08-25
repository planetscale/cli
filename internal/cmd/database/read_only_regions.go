package database

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ReadOnlyRegionsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-only-regions <command>",
		Short: "List read-only regions for a database",
		Long: "List read-only regions for a database's default branch.\n\n" +
			"This command is only supported for Vitess databases.",
	}

	cmd.AddCommand(ReadOnlyRegionsListCmd(ch))
	return cmd
}

func ReadOnlyRegionsListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:     "list <database>",
		Short:   "List read-only regions for a database",
		Aliases: []string{"ls"},
		Args:    cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database := args[0]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching read-only regions for database %s...", printer.BoldBlue(database)))
			defer end()

			regions, err := client.ReadOnlyRegions.List(cmd.Context(), &ps.ListReadOnlyRegionsRequest{
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
					ch.Printer.Println("No read-only regions have been configured for this database.")
				} else {
					ch.Printer.Println("No read-only regions found on this page.")
				}
				return nil
			}

			return ch.Printer.PrintResource(toDatabaseReadOnlyRegions(regions))
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}

type databaseReadOnlyRegion struct {
	ID       string `header:"id" json:"id"`
	Name     string `header:"name" json:"display_name"`
	Region   string `header:"region" json:"region"`
	Provider string `header:"provider" json:"provider"`
	Ready    bool   `header:"ready" json:"ready"`
	orig     *ps.ReadOnlyRegion
}

func toDatabaseReadOnlyRegions(regions []*ps.ReadOnlyRegion) []*databaseReadOnlyRegion {
	out := make([]*databaseReadOnlyRegion, 0, len(regions))
	for _, region := range regions {
		out = append(out, &databaseReadOnlyRegion{
			ID:       region.ID,
			Name:     region.DisplayName,
			Region:   region.Region.Slug,
			Provider: region.Region.Provider,
			Ready:    region.Ready,
			orig:     region,
		})
	}
	return out
}

func (r *databaseReadOnlyRegion) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *databaseReadOnlyRegion) MarshalCSVValue() interface{} {
	return []*databaseReadOnlyRegion{r}
}
