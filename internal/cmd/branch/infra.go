package branch

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

type InfraPod struct {
	Component     string `header:"component" json:"component"`
	Size          string `header:"size" json:"size"`
	Cell          string `header:"cell" json:"cell"`
	TabletType    string `header:"tablet type" json:"tablet_type"`
	KeyspaceShard string `header:"keyspace/shard" json:"keyspace_shard"`
	Ready         string `header:"ready" json:"ready"`
	Restarts      int    `header:"restarts" json:"restarts"`
	Status        string `header:"status" json:"status"`
	Age           string `header:"age" json:"age"`
	Name          string `header:"pod name" json:"name"`

	orig *ps.BranchInfraPod
}

func (p *InfraPod) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(p.orig, "", "  ")
}

func (p *InfraPod) MarshalCSVValue() interface{} {
	return []*InfraPod{p}
}

func toInfraPod(pod *ps.BranchInfraPod) *InfraPod {
	tabletType := "-"
	if pod.TabletType != nil {
		tabletType = *pod.TabletType
	}

	keyspaceShard := "-"
	if pod.Keyspace != nil && pod.Shard != nil {
		keyspaceShard = fmt.Sprintf("%s/%s", *pod.Keyspace, *pod.Shard)
	}

	age := "-"
	if pod.CreatedAt != nil {
		age = humanAge(time.Since(*pod.CreatedAt))
	}

	return &InfraPod{
		Component:     pod.Component,
		Size:          pod.Size,
		Cell:          pod.Cell,
		TabletType:    tabletType,
		KeyspaceShard: keyspaceShard,
		Ready:         pod.Ready,
		Restarts:      pod.RestartCount,
		Status:        pod.Status,
		Age:           age,
		Name:          pod.Name,
		orig:          pod,
	}
}

func toInfraPods(pods []*ps.BranchInfraPod) []*InfraPod {
	result := make([]*InfraPod, 0, len(pods))
	for _, pod := range pods {
		result = append(result, toInfraPod(pod))
	}
	return result
}

func humanAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	if h == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, h)
}

// InfraCmd shows infrastructure (pods) for a branch.
func InfraCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "infra <database> <branch>",
		Short:  "Show infrastructure (pods) for a branch",
		Hidden: true,
		Args:   cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching infrastructure for %s/%s",
				printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			infra, err := client.BranchInfrastructure.Get(ctx, &ps.GetBranchInfrastructureRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "read_branch",
						"branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if len(infra.Pods) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No pods found for branch %s.\n", printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toInfraPods(infra.Pods))
		},
	}

	return cmd
}
