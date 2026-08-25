package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

type RecommendationDetailRow struct {
	Number   int    `header:"number" json:"number"`
	State    string `header:"state" json:"state"`
	Type     string `header:"type" json:"recommendation_type"`
	Table    string `header:"table" json:"table_name"`
	Keyspace string `header:"keyspace" json:"keyspace,omitempty"`
	Title    string `header:"title" json:"title"`
	DDL      string `header:"ddl" json:"ddl_statement"`
	URL      string `header:"url" json:"html_url,omitempty"`
}

func RecommendationShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <number>",
		Short: "Show a schema recommendation and its DDL",
		Long: `Show a single schema recommendation, including the full ready-to-apply DDL.

<number> is the recommendation sequence number from
'pscale insights recommendations <database>', the same value used by
'pscale insights recommendations dismiss'.`,
		Example: `  pscale insights recommendations show mydb 1 --org myorg
  pscale insights recommendations show mydb 1 --org myorg --format json`,
		Args: cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			number := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching schema recommendation %s for %s...",
				printer.BoldBlue(number), printer.BoldBlue(database)))
			defer end()

			recommendation, err := client.SchemaRecommendations.Get(ctx, &ps.GetSchemaRecommendationRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				ID:           number,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("schema recommendation %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(number), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(recommendation)
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Number:    %d\n", recommendation.Number)
				ch.Printer.Printf("State:     %s\n", recommendation.State)
				ch.Printer.Printf("Type:      %s\n", recommendation.RecommendationType)
				ch.Printer.Printf("Table:     %s\n", recommendation.Table)
				if recommendation.Keyspace != "" {
					ch.Printer.Printf("Keyspace:  %s\n", recommendation.Keyspace)
				}
				ch.Printer.Printf("Title:     %s\n", recommendation.Title)
				if recommendation.HtmlURL != "" {
					ch.Printer.Printf("URL:       %s\n", recommendation.HtmlURL)
				}
				if recommendation.DDLStatement != "" {
					ch.Printer.Printf("\n%s\n%s\n", printer.Bold("DDL:"), recommendation.DDLStatement)
				}
				return nil
			}

			return ch.Printer.PrintResource([]*RecommendationDetailRow{{
				Number:   recommendation.Number,
				State:    recommendation.State,
				Type:     recommendation.RecommendationType,
				Table:    recommendation.Table,
				Keyspace: recommendation.Keyspace,
				Title:    recommendation.Title,
				DDL:      recommendation.DDLStatement,
				URL:      recommendation.HtmlURL,
			}})
		},
	}

	return cmd
}
