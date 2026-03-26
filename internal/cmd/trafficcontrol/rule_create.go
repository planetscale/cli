package trafficcontrol

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func RuleCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		kind        string
		fingerprint string
		keyspace    string
		tags        []string
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch> <budget-id>",
		Short: "Create a traffic rule on a budget",
		Args:  cmdutil.RequiredArgs("database", "branch", "budget-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			budgetID := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating traffic rule on budget %s for %s/%s",
				printer.BoldBlue(budgetID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.CreateTrafficRuleRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				BudgetID:     budgetID,
				Kind:         flags.kind,
			}

			if cmd.Flags().Changed("fingerprint") {
				req.Fingerprint = &flags.fingerprint
			}
			if cmd.Flags().Changed("keyspace") {
				req.Keyspace = &flags.keyspace
			}
			if cmd.Flags().Changed("tag") {
				tags, err := parseTags(flags.tags)
				if err != nil {
					return err
				}
				req.Tags = &tags
			}

			rule, err := client.TrafficRules.Create(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s, branch %s, or budget %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(budgetID), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Traffic rule %s was successfully created on budget %s.\n",
					printer.BoldBlue(rule.ID), printer.BoldBlue(budgetID))
			}

			return ch.Printer.PrintResource(toTrafficRule(rule))
		},
	}

	cmd.Flags().StringVar(&flags.kind, "kind", "match", "Kind of rule")
	cmd.Flags().StringVar(&flags.fingerprint, "fingerprint", "", "SQL fingerprint to match")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace to match")
	cmd.Flags().StringArrayVar(&flags.tags, "tag", nil, "Tag in the format key=<key>,value=<value>,source=<source> (repeatable)")

	return cmd
}

func parseTags(raw []string) ([]ps.TrafficRuleTag, error) {
	tags := make([]ps.TrafficRuleTag, 0, len(raw))
	for _, s := range raw {
		tag, err := parseTag(s)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func parseTag(s string) (ps.TrafficRuleTag, error) {
	tag := ps.TrafficRuleTag{}
	for part := range strings.SplitSeq(s, ",") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return tag, fmt.Errorf("invalid tag format %q: expected key=value pairs separated by commas (e.g. key=query,value=SELECT *,source=sql)", s)
		}
		switch k {
		case "key":
			tag.Key = v
		case "value":
			tag.Value = v
		case "source":
			tag.Source = v
		default:
			return tag, fmt.Errorf("unknown tag field %q in %q: valid fields are key, value, source", k, s)
		}
	}
	if tag.Key == "" {
		return tag, fmt.Errorf("tag %q is missing required field 'key'", s)
	}
	if tag.Value == "" {
		return tag, fmt.Errorf("tag %q is missing required field 'value'", s)
	}
	if tag.Source == "" {
		tag.Source = "sql"
	}
	return tag, nil
}
