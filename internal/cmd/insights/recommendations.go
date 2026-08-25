package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// RecommendationRow is a schema recommendation formatted for table output.
type RecommendationRow struct {
	Number   int    `header:"number" json:"number"`
	State    string `header:"state" json:"state"`
	Type     string `header:"type" json:"recommendation_type"`
	Table    string `header:"table" json:"table_name"`
	Keyspace string `header:"keyspace" json:"keyspace,omitempty"`
	Title    string `header:"title" json:"title"`
}

// RecommendationsCmd lists schema recommendations for a database.
func RecommendationsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recommendations <database>",
		Short: "List schema recommendations (unused/duplicate indexes, bloat, missing indexes)",
		Long: `List PlanetScale's schema recommendations for a database: unused tables and
indexes, duplicate indexes, bloated tables and indexes, missing indexes
derived from production query patterns, and sequence overflow risks. Each
recommendation includes ready-to-apply DDL (shown with --format json).

Use "pscale insights recommendations show" to print a recommendation and its
full DDL. Use "pscale insights recommendations dismiss" to dismiss one.`,
		Aliases: []string{"recommendation"},
		Args:    cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching schema recommendations for %s...",
				printer.BoldBlue(database)))
			defer end()

			recommendations, err := client.SchemaRecommendations.List(ctx, &ps.ListSchemaRecommendationsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				return notFoundError(ch, err, database, "")
			}
			end()

			if len(recommendations) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No schema recommendations for %s. Your schema looks good!\n",
					printer.BoldBlue(database))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(recommendations)
			}

			rows := make([]*RecommendationRow, 0, len(recommendations))
			for _, r := range recommendations {
				rows = append(rows, &RecommendationRow{
					Number:   r.Number,
					State:    r.State,
					Type:     r.RecommendationType,
					Table:    r.Table,
					Keyspace: r.Keyspace,
					Title:    truncate(r.Title, 80),
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.AddCommand(RecommendationShowCmd(ch))
	cmd.AddCommand(RecommendationDismissCmd(ch))

	return cmd
}
