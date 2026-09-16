package nekichanges

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var targetTypeNames = map[string]string{
	"admin":                    "NekiAdmin",
	"nekiadmin":                "NekiAdmin",
	"cluster":                  "NekiCluster",
	"nekicluster":              "NekiCluster",
	"configprofile":            "NekiConfigurationProfile",
	"configurationprofile":     "NekiConfigurationProfile",
	"nekiconfigurationprofile": "NekiConfigurationProfile",
	"router":                   "NekiRouter",
	"nekirouter":               "NekiRouter",
	"sidecar":                  "NekiSidecar",
	"nekisidecar":              "NekiSidecar",
}

// ChangesCmd lists, shows, and cancels change requests across a Neki branch.
func ChangesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "changes <command>",
		Aliases: []string{"neki-changes"},
		Short:   "List change requests across a Neki branch",
		Long: `List, show, and cancel change requests across a Neki branch.

This includes admin, cluster, configuration profile, router, and sidecar
changes. Per-resource commands still exist under admin, config-profile,
router, and sidecar. This command is only supported for Neki databases.`,
	}
	cmd.AddCommand(changesListCmd(ch), changesShowCmd(ch), changesCancelCmd(ch))
	return cmd
}

func changesListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		states      []string
		targetTypes []string
		targetID    string
		period      string
		completedAt string
		page        int
		perPage     int
	}

	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List change requests for a Neki branch",
		Args:    cmdutil.ExactArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			targetTypes, err := normalizeTargetTypes(flags.targetTypes)
			if err != nil {
				return err
			}
			if flags.targetID != "" && len(targetTypes) == 0 {
				return fmt.Errorf("--target-id requires --target-type")
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching changes for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()
			changes, err := client.NekiChanges.List(cmd.Context(), &ps.ListNekiChangesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				States:       flags.states,
				TargetTypes:  targetTypes,
				TargetID:     flags.targetID,
				Period:       flags.period,
				CompletedAt:  flags.completedAt,
				Page:         flags.page,
				PerPage:      flags.perPage,
			})
			if err != nil {
				return handleError(err, database, branch, "")
			}
			end()

			if len(changes) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No changes found for branch %s.\n", printer.BoldBlue(branch))
				return nil
			}
			return ch.Printer.PrintResource(toChanges(changes))
		},
	}

	cmd.Flags().StringSliceVar(&flags.states, "state", nil, "Filter by state (repeat or comma-separate)")
	cmd.Flags().StringSliceVar(&flags.targetTypes, "target-type", nil, "Filter by target type: admin, cluster, config-profile, router, or sidecar")
	cmd.Flags().StringVar(&flags.targetID, "target-id", "", "Filter by target ID. Requires --target-type")
	cmd.Flags().StringVar(&flags.period, "period", "", "Only show changes from this period")
	cmd.Flags().StringVar(&flags.completedAt, "completed-at", "", "Only show changes completed in this time range")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}

func changesShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "show <database> <branch> <change-id>",
		Short:   "Show a Neki branch change request",
		Args:    cmdutil.ExactArgs("database", "branch", "change-id"),
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, changeID := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching change %s", printer.BoldBlue(changeID)))
			defer end()
			change, err := client.NekiChanges.Get(cmd.Context(), &ps.GetNekiChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Change:       changeID,
			})
			if err != nil {
				return handleError(err, database, branch, changeID)
			}
			end()
			return ch.Printer.PrintResource(toChange(change))
		},
	}
}

func changesCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <database> <branch> <change-id>",
		Short: "Cancel a pending Neki branch change request",
		Args:  cmdutil.ExactArgs("database", "branch", "change-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, changeID := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Cancelling change %s", printer.BoldBlue(changeID)))
			defer end()
			err = client.NekiChanges.Cancel(cmd.Context(), &ps.CancelNekiChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Change:       changeID,
			})
			if err != nil {
				return handleError(err, database, branch, changeID)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Canceled change %s for branch %s.\n", printer.BoldBlue(changeID), printer.BoldBlue(branch))
				return nil
			}
			return ch.Printer.PrintResource(map[string]string{
				"result":    "change canceled",
				"change_id": changeID,
			})
		},
	}
}

func normalizeTargetTypes(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		normalized, err := normalizeTargetType(value)
		if err != nil {
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeTargetType(raw string) (string, error) {
	key := strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(strings.TrimSpace(raw)))
	if mapped, ok := targetTypeNames[key]; ok {
		return mapped, nil
	}
	return "", fmt.Errorf("unknown target type %q; must be one of: admin, cluster, config-profile, router, sidecar", raw)
}

func displayTargetType(apiType string) string {
	switch apiType {
	case "NekiAdmin":
		return "admin"
	case "NekiCluster":
		return "cluster"
	case "NekiConfigurationProfile":
		return "config-profile"
	case "NekiRouter":
		return "router"
	case "NekiSidecar":
		return "sidecar"
	default:
		return apiType
	}
}

type changeDisplay struct {
	ID         string `header:"id" json:"id"`
	State      string `header:"state" json:"state"`
	TargetType string `header:"target" json:"target_type"`
	TargetName string `header:"name" json:"target_name"`
	CreatedAt  string `header:"created at" json:"created_at"`
	orig       *ps.NekiChange
}

func (c *changeDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(c.orig) }
func (c *changeDisplay) MarshalCSVValue() interface{} { return []*changeDisplay{c} }

func toChange(c *ps.NekiChange) *changeDisplay {
	return &changeDisplay{
		ID:         c.ID,
		State:      c.State,
		TargetType: displayTargetType(c.TargetType),
		TargetName: c.TargetName,
		CreatedAt:  c.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:       c,
	}
}

func toChanges(changes []*ps.NekiChange) []*changeDisplay {
	out := make([]*changeDisplay, 0, len(changes))
	for _, change := range changes {
		out = append(out, toChange(change))
	}
	return out
}

func handleError(err error, database, branch, changeID string) error {
	if cmdutil.ErrCode(err) != ps.ErrNotFound {
		return cmdutil.HandleError(err)
	}
	if changeID != "" {
		return fmt.Errorf("change %s does not exist in branch %s of database %s, or the branch is not a Neki branch",
			printer.BoldBlue(changeID), printer.BoldBlue(branch), printer.BoldBlue(database))
	}
	return fmt.Errorf("database %s or branch %s does not exist, or the branch is not a Neki branch",
		printer.BoldBlue(database), printer.BoldBlue(branch))
}
