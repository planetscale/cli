package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// TagValueRow is a tag value for table output.
type TagValueRow struct {
	Name       string `header:"value" json:"name"`
	Kind       string `header:"kind" json:"kind"`
	QueryCount int64  `header:"queries" json:"query_count"`
}

// TagShowCmd shows one tag key and its values.
func TagShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		period   string
		keyspace string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch> <tag>",
		Short: "Show a query tag key and its values",
		Long: `Show one tag key and the values observed for it.

<tag> is the friendly name from 'pscale insights tags' (e.g. app, username),
matching the Insights UI Key picker. Source-qualified names (sql:app,
system:username) are accepted when a name exists in both sources.`,
		Example: `  pscale insights tags show mydb main app --org myorg
  pscale insights tags show mydb main username --org myorg --period 1h`,
		Args: cmdutil.RequiredArgs("database", "branch", "tag"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, tagInput := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching tag %s for %s in %s...",
				printer.BoldBlue(tagInput), printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			listOpts := []ps.ListOption{ps.WithPeriod(flags.period)}
			if flags.keyspace != "" {
				listOpts = append(listOpts, ps.WithKeyspace(flags.keyspace))
			}

			tags, err := client.QueryInsights.ListTags(ctx, &ps.ListQueryTagsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}, listOpts...)
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}

			ids, err := resolveTagIDs(tags, []string{tagInput})
			if err != nil {
				end()
				return err
			}

			tag, err := client.QueryInsights.GetTag(ctx, &ps.GetQueryTagRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Tag:          ids[0],
			}, listOpts...)
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(tag)
			}

			ch.Printer.Printf("Tag %s (%s) — %d queries\n",
				printer.BoldBlue(tag.Name), tag.Source, tag.QueryCount)

			rows := make([]*TagValueRow, 0, len(tag.Values))
			for _, v := range tag.Values {
				rows = append(rows, &TagValueRow{
					Name:       v.Name,
					Kind:       v.Kind,
					QueryCount: v.QueryCount,
				})
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.Flags().StringVar(&flags.period, "period", "", "Time period to look back (e.g. 1h, 1d)")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Filter tag values to a keyspace")

	return cmd
}
