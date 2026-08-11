package insights

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// TagRow is a tag key for table output.
type TagRow struct {
	Name       string `header:"name" json:"name"`
	Source     string `header:"source" json:"source"`
	QueryCount int64  `header:"queries" json:"query_count"`
	TopValues  string `header:"top values" json:"top_values"`
}

// TagsCmd lists query tags and groups related subcommands.
func TagsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		period      string
		fingerprint string
		keyspace    string
		limit       int
	}

	cmd := &cobra.Command{
		Use:   "tags <database> <branch>",
		Short: "List query tag keys (sqlcommenter / system)",
		Long: `List query tag keys observed on a branch.

Tag names match the Key picker in the PlanetScale Insights UI (e.g. app,
username, controller). Use those names with "tags summaries --tags".

Typical agent flow:
  1. pscale insights tags <database> <branch> --format json
  2. pscale insights tags summaries <database> <branch> --tags <name> --format json`,
		Example: `  pscale insights tags mydb main --org myorg
  pscale insights tags mydb main --org myorg --period 1h`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching query tags for %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			opts := []ps.ListOption{ps.WithPeriod(flags.period)}
			if flags.fingerprint != "" {
				opts = append(opts, ps.WithFingerprint(flags.fingerprint))
			}
			if flags.keyspace != "" {
				opts = append(opts, ps.WithKeyspace(flags.keyspace))
			}

			tags, err := client.QueryInsights.ListTags(ctx, &ps.ListQueryTagsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}, opts...)
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(tags) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No query tags recorded for %s in %s.\n",
					printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(tags)
			}

			rows := make([]*TagRow, 0, len(tags))
			for _, t := range tags {
				rows = append(rows, &TagRow{
					Name:       t.Name,
					Source:     t.Source,
					QueryCount: t.QueryCount,
					TopValues:  formatTopValues(t.Values, flags.limit),
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.Flags().StringVar(&flags.period, "period", "", "Time period to look back (e.g. 1h, 1d)")
	cmd.Flags().StringVar(&flags.fingerprint, "fingerprint", "", "Only tags seen on this query fingerprint")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Filter tags to a keyspace")
	cmd.Flags().IntVar(&flags.limit, "limit", 3, "Number of top values to show in human output")

	cmd.AddCommand(TagShowCmd(ch))
	cmd.AddCommand(TagSummariesCmd(ch))

	return cmd
}

func formatTopValues(values []ps.QueryTagValue, limit int) string {
	if limit <= 0 {
		limit = 3
	}
	parts := make([]string, 0, limit)
	for i, v := range values {
		if i >= limit {
			break
		}
		parts = append(parts, fmt.Sprintf("%s (%d)", v.Name, v.QueryCount))
	}
	return strings.Join(parts, ", ")
}
